package shardkv

import (
	"6.824/labrpc"
	"6.824/shardctrler"
	"bytes"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)
import "6.824/raft"
import "sync"
import "6.824/labgob"

const Debug = true

func (kv *ShardKV) DPrintf(format string, a ...interface{}) {
	if Debug {
		log.Printf(fmt.Sprintf("[ShardKV Server %d-%d]:%s", kv.gid, kv.me, format), a...)
	}
	return
}

type Op struct {
	LogType      string
	CommandArgs  *CommandArgs
	ReConfigArgs *ReConfigArgs
}

func (kv *ShardKV) Lock(owner string) {
	//kv.DPrintf("%s Lock", owner)
	kv.mu.Lock()
}

func (kv *ShardKV) Unlock(owner string) {
	//kv.DPrintf("%s Unlock", owner)
	kv.mu.Unlock()
}

func (kv *ShardKV) RLock(owner string) {
	//kv.DPrintf("%s RLock", owner)
	kv.mu.RLock()
}

func (kv *ShardKV) RUnlock(owner string) {
	//kv.DPrintf("%s RUnlock", owner)
	kv.mu.RUnlock()
}

type ShardState struct {
	state      string
	prevCfg    *shardctrler.Config
	currentCfg *shardctrler.Config
}

type ShardKV struct {
	mu   sync.RWMutex
	dead int32 // set by Kill()

	me      int
	applyCh chan raft.ApplyMsg
	servers []*labrpc.ClientEnd
	rf      *raft.Raft

	make_end       func(string) *labrpc.ClientEnd
	gid            int
	ctrlers        []*labrpc.ClientEnd
	mck            *shardctrler.Clerk
	configs        []shardctrler.Config // 所有的配置信息
	shardState     map[int]*ShardState  // 分片的配置状态
	updateConfigCh chan struct{}        // 用于通知更新配置

	maxraftstate int // snapshot if log grows this big

	lastApplyIndex int
	db             map[string]string            // 内存数据库
	notifyChans    map[int]chan *CommonReply    // 监听请求 apply
	dupReqHistory  map[int64]map[int64]struct{} // 记录已经执行的修改命令，防止重复执行

}

type DupReqHistorySnap map[int64]string

func (kv *ShardKV) makeDupReqHistorySnap() DupReqHistorySnap {
	snap := make(DupReqHistorySnap, 0)
	for clientId, info := range kv.dupReqHistory {
		var seqs []int64
		for sequenceNum := range info {
			seqs = append(seqs, sequenceNum)
		}

		// 排序
		for i := 0; i <= len(seqs)-1; i++ {
			for j := i; j <= len(seqs)-1; j++ {
				if seqs[i] > seqs[j] {
					t := seqs[i]
					seqs[i] = seqs[j]
					seqs[j] = t
				}
			}
		}

		// 将所有序列号压缩(记录和前一条的差值)成一条字符串
		snapString := make([]string, len(seqs))
		var prev int64
		for idx, seq := range seqs {
			if idx == 0 {
				snapString = append(snapString, strconv.FormatInt(seq, 10))
			} else {
				snapString = append(snapString, strconv.FormatInt(seq-prev, 10))
			}
			prev = seq
		}

		snap[clientId] = strings.Join(snapString, "")
	}

	return snap
}

func (kv *ShardKV) restoreDupReqHistorySnap(snap DupReqHistorySnap) {
	kv.dupReqHistory = make(map[int64]map[int64]struct{})
	for clientId, info := range snap {
		if _, ok := kv.dupReqHistory[clientId]; !ok {
			kv.dupReqHistory[clientId] = make(map[int64]struct{})
		}

		snapString := strings.Split(info, "")
		var prev int64
		for idx, value := range snapString {
			if idx == 0 {
				seq, _ := strconv.ParseInt(value, 10, 64)
				prev = seq
			} else {
				seq, _ := strconv.ParseInt(value, 10, 64)
				prev += seq
			}
			kv.dupReqHistory[clientId][prev] = struct{}{}
		}
	}
}

func (kv *ShardKV) checkShard(key string, reply *CommonReply) bool {
	kv.RLock("checkShard")
	defer kv.Unlock("checkShard")

	shard := key2shard(key)
	shardInfo := kv.shardState[shard]
	if shardInfo.state == Working {
		// 当前分片不由 gid 负责
		if shardInfo.currentCfg.Shards[shard] != kv.gid {
			reply.Err = ErrWrongGroup
			kv.DPrintf("shard %d response %s", shard, reply.Err)
			return true
		}
	} else {
		if shardInfo.state == ReConfining {
			// 之前不负责，现在需要负责的分片，等待分片传输完成再服务
			if shardInfo.prevCfg.Shards[shard] != kv.gid && shardInfo.currentCfg.Shards[shard] == kv.gid {
				reply.Err = ErrRetry
				kv.DPrintf("Waiting for shard %d migration", shard)
				return true
			}
			// 之前负责，现在不负责的分片，在开始 reconfig 后需要丢弃
			if shardInfo.prevCfg.Shards[shard] == kv.gid && shardInfo.currentCfg.Shards[shard] != kv.gid {
				reply.Err = ErrWrongGroup
				kv.DPrintf("The shard %d after ReConfining needs to be discard", shard)
				return true
			}
		} else {
			// 在 ReConfining 之前负责的分片需要处理
			if shardInfo.prevCfg.Shards[shard] != kv.gid {
				reply.Err = ErrWrongGroup
				kv.DPrintf("The shard %d before ReConfining needs to be process", shard)
				return true
			}
		}
	}

	return false
}

func (kv *ShardKV) Command(args *CommandArgs, reply *CommonReply) {
	if args.Op != OpGet && kv.isDuplicateRequest(args.ClientId, args.SequenceNum) {
		kv.DPrintf("duplicate command request: %+v, reply history response", args)
		reply.Err = OK
		return
	}

	if kv.checkShard(args.Key, reply) {
		return
	}

	index, term, isLeader := kv.rf.Start(&Op{
		LogType:     CommandLog,
		CommandArgs: args,
	})
	if !isLeader {
		reply.Err = ErrWrongLeader
		return
	}

	kv.Lock("getNotifyChan")
	if _, ok := kv.notifyChans[index]; !ok {
		kv.notifyChans[index] = make(chan *CommonReply)
	}
	kv.Unlock("getNotifyChan")

	select {
	case result := <-kv.notifyChans[index]:
		if reply.Err == ErrWrongGroup {
			// 尝试更新配置，防止自己配置更新滞后
			select {
			case kv.updateConfigCh <- struct{}{}:
			default:
			}
			return
		}
		currentTerm, isleader := kv.rf.GetState()
		if !isleader || currentTerm != term {
			reply.Err = ErrWrongLeader
			kv.DPrintf("reply now is not leader")
			return
		}
		kv.DPrintf("reply index: %d", index)

		if reply.Err == ApplySnap {
			if args.Op != OpGet {
				reply.Err = OK
			} else {
				reply.Err = ErrRetry
			}
		} else {
			reply.Err, reply.Value = result.Err, result.Value
		}
	case <-time.After(ExecuteTimeout):
		kv.DPrintf("wait apply log %d time out", index)
		reply.Err = ErrTimeout
	}

	kv.Lock("Command")
	defer kv.Unlock("Command")
	close(kv.notifyChans[index])
	delete(kv.notifyChans, index)
}

func (kv *ShardKV) Get(args *CommandArgs, reply *CommonReply) {
	kv.Command(args, reply)
}

func (kv *ShardKV) PutAppend(args *CommandArgs, reply *CommonReply) {
	kv.Command(args, reply)
}

func (kv *ShardKV) ReConfig(args *ReConfigArgs, reply *CommonReply) {
	if kv.isDuplicateRequest(int64(args.Shard), int64(args.Num)) {
		kv.DPrintf("duplicate reconfig request: %+v, reply history response", args)
		reply.Err = OK
		return
	}

	index, term, isLeader := kv.rf.Start(&Op{
		LogType:      ReConfigLog,
		ReConfigArgs: args,
	})
	if !isLeader {
		reply.Err = ErrWrongLeader
		return
	}

	kv.Lock("getNotifyChan")
	if _, ok := kv.notifyChans[index]; !ok {
		kv.notifyChans[index] = make(chan *CommonReply)
	}
	kv.Unlock("getNotifyChan")

	select {
	case result := <-kv.notifyChans[index]:
		currentTerm, isleader := kv.rf.GetState()
		if !isleader || currentTerm != term {
			reply.Err = ErrWrongLeader
			kv.DPrintf("reply now is not leader")
			return
		}
		kv.DPrintf("reply index: %d", index)

		if reply.Err == ApplySnap {
			reply.Err = ErrRetry
		} else {
			reply.Err, reply.Value = result.Err, result.Value
		}
	case <-time.After(ExecuteTimeout):
		kv.DPrintf("wait apply log %d time out", index)
		reply.Err = ErrTimeout
	}

	kv.Lock("ReConfig")
	defer kv.Unlock("ReConfig")
	close(kv.notifyChans[index])
	delete(kv.notifyChans, index)
}

func (kv *ShardKV) writeReConfigLog() {
	for kv.killed() == false {
		kv.Lock("writeReConfigLog")
		for shard, info := range kv.shardState {
			if info.state == Working && info.currentCfg.Num+1 < len(kv.configs) {
				kv.shardState[shard].state = PrepareReConfig
				kv.shardState[shard].prevCfg = kv.shardState[shard].currentCfg
				cfgNum := kv.shardState[shard].prevCfg.Num
				kv.shardState[shard].currentCfg = &kv.configs[kv.shardState[shard].currentCfg.Num+1]
				go func() {
					// 写入日志直到成功，重复写入也没关系，在 apply 处理即可
					args := ReConfigArgs{
						Server: fmt.Sprintf("%d-%d", kv.gid, kv.me),
						Shard:  shard,
						Num:    cfgNum,
					}
					writeLogSuccess := false
					for !writeLogSuccess {
						_, isleader := kv.rf.GetState()
						if isleader {
							var reply CommonReply
							kv.ReConfig(&args, &reply)
							if reply.Err == OK {
								writeLogSuccess = true
							}
						} else {
							// try each server for the shard.
							for si := 0; si < len(kv.servers); si++ {
								srv := kv.servers[si]
							retry:
								var reply CommonReply
								ok := srv.Call("ShardKV.ReConfig", &args, &reply)
								if ok && (reply.Err == OK) {
									writeLogSuccess = true
									break
								}
								if ok && (reply.Err == ErrRetry) {
									kv.DPrintf("retry write reConfig log")
									goto retry
								}
							}
						}
						if !writeLogSuccess {
							time.Sleep(10 * time.Millisecond)
						}
					}
					kv.DPrintf("===== writeReConfigLog Success: %+v =====", args)
				}()
			}
		}
		kv.Unlock("writeReConfigLog")
		time.Sleep(time.Millisecond)
	}
}

func (kv *ShardKV) PushShard(args *PushShardArgs, reply *CommonReply) {

}

// 把分片 shard 推送给 gid 直到成功
func (kv *ShardKV) pushShard(data map[string]string, gidServers []string, shardCfgNum, shard, gid int) {
	for kv.killed() {
		args := PushShardArgs{
			Data:        data,
			Shard:       shard,
			ShardCfgNum: shardCfgNum,
		}

		// try each server for the shard.
		for si := 0; si < len(gidServers); si++ {
			srv := kv.make_end(gidServers[si])
			var reply CommonReply
			ok := srv.Call("ShardKV.PushShard", &args, &reply)
			if ok && (reply.Err == OK) {
				kv.DPrintf("=== pushShard %d to %v success ===", shard, gid)
				return
			}
		}

		time.Sleep(10 * time.Millisecond)
	}
}

func (kv *ShardKV) updateDupReqHistory(clientId, sequenceNum int64) {
	if _, ok := kv.dupReqHistory[clientId]; !ok {
		kv.dupReqHistory[clientId] = make(map[int64]struct{})
	}
	kv.dupReqHistory[clientId][sequenceNum] = struct{}{}
}

func (kv *ShardKV) isDuplicateRequest(clientId, sequenceNum int64) bool {
	kv.RLock("isDuplicateRequest")
	defer kv.RUnlock("isDuplicateRequest")
	if _, ok := kv.dupReqHistory[clientId]; ok {
		if _, ok := kv.dupReqHistory[clientId][sequenceNum]; ok {
			return true
		}
	}

	return false
}

func (kv *ShardKV) applyCommand(command *CommandArgs, applyLogIndex int) {
	reply := &CommonReply{Err: OK}
	var value string
	var ok bool

	// 防止重复应用同一条修改命令
	if command.Op != OpGet && kv.isDuplicateRequest(command.ClientId, command.SequenceNum) {
		kv.DPrintf("found duplicate request: %+v", command)
		goto replyCommand
	}

	// 检查当前是否服务分片
	if kv.checkShard(command.Key, reply) {
		goto replyCommand
	}

	// 更新状态机
	value, ok = kv.db[command.Key]
	if command.Op == OpGet {
		if ok {
			reply.Value = value
			kv.DPrintf("get <%s>:<%s>", command.Key, value)
		} else {
			reply.Err = ErrNoKey
		}
	} else {
		if command.Op == OpAppend && ok {
			kv.db[command.Key] += command.Value
		} else {
			kv.db[command.Key] = command.Value
		}
		kv.DPrintf("update <%s>:<%s>", command.Key, kv.db[command.Key])
	}

replyCommand:
	kv.Lock("replyCommand")
	if command.Op != OpGet {
		kv.updateDupReqHistory(command.ClientId, command.SequenceNum)
	}
	if _, ok := kv.notifyChans[applyLogIndex]; ok {
		select {
		case kv.notifyChans[applyLogIndex] <- reply:
		default:
			kv.DPrintf("reply to chan index %d failed", applyLogIndex)
		}
	}
	kv.Unlock("replyCommand")

	// 检测是否需要执行快照
	if kv.rf.RaftStateNeedSnapshot(kv.maxraftstate) {
		kv.DPrintf("======== snapshot %d ========", applyLogIndex)
		w := new(bytes.Buffer)
		e := labgob.NewEncoder(w)
		kv.Lock("snap")
		dupReqHistorySnap := kv.makeDupReqHistorySnap()
		if e.Encode(kv.db) != nil || e.Encode(dupReqHistorySnap) != nil {
			kv.DPrintf("[panic] encode snap error")
			kv.Unlock("snap")
			kv.Kill()
			return
		}
		kv.Unlock("snap")
		data := w.Bytes()
		kv.DPrintf("snap size: %d", len(data))
		kv.rf.Snapshot(applyLogIndex, data)
	}
}

func (kv *ShardKV) applyReConfig(args *ReConfigArgs, applyLogIndex int) {
	reply := &CommonReply{Err: OK}

	// 防止重复应用同一条修改命令
	if kv.isDuplicateRequest(int64(args.Shard), int64(args.Num)) {
		kv.DPrintf("found duplicate request: %+v", args)
		goto replyCommand
	}
	kv.Lock("checkShard")
	kv.updateDupReqHistory(int64(args.Shard), int64(args.Num))
	kv.shardState[args.Shard].state = ReConfining
	for idx := range kv.shardState[args.Shard].currentCfg.Shards {
		prevGID := kv.shardState[args.Shard].prevCfg.Shards[idx]
		nowGID := kv.shardState[args.Shard].currentCfg.Shards[idx]
		if prevGID == kv.gid {
			if nowGID != kv.gid && nowGID != 0 {
				data := make(map[string]string)
				for k, v := range kv.db {
					shard := key2shard(k)
					// 找出需要迁移的分片数据，并清除本地副本
					if shard == idx {
						data[k] = v
						delete(kv.db, k)
					}
				}
				gidServers := kv.shardState[args.Shard].currentCfg.Groups[nowGID]
				shardCfgNum := kv.shardState[args.Shard].prevCfg.Num
				go kv.pushShard(data, gidServers, shardCfgNum, idx, nowGID)
			} else if nowGID == kv.gid {
				kv.shardState[args.Shard].state = Working
			}
		}
	}
	kv.Unlock("checkShard")

replyCommand:
	kv.Lock("replyCommand")
	if _, ok := kv.notifyChans[applyLogIndex]; ok {
		select {
		case kv.notifyChans[applyLogIndex] <- reply:
		default:
			kv.DPrintf("reply to chan index %d failed", applyLogIndex)
		}
	}
	kv.Unlock("replyCommand")
}

func (kv *ShardKV) handleApply() {
	for kv.killed() == false {
		select {
		case applyLog := <-kv.applyCh:
			if applyLog.CommandValid {
				op, ok := applyLog.Command.(Op)
				if !ok {
					kv.DPrintf("[panic] recieved apply log's command error")
					kv.Kill()
					return
				}

				if applyLog.CommandIndex <= kv.lastApplyIndex {
					// 比如 raft 重启了，就要重新 apply
					kv.DPrintf("***** command index %d is older than lastApplyIndex %d *****",
						applyLog.CommandIndex, kv.lastApplyIndex)
					continue
				}
				kv.lastApplyIndex = applyLog.CommandIndex

				if op.LogType == CommandLog {
					kv.DPrintf("recieve command apply log: %d, CommandArgs: %+v", applyLog.CommandIndex, op.CommandArgs)
					kv.applyCommand(op.CommandArgs, applyLog.CommandIndex)
				} else if op.LogType == ReConfigLog {
					kv.DPrintf("recieve reConfig apply log: %d, ReConfigArgs: %+v", applyLog.CommandIndex, op.ReConfigArgs)
					kv.applyReConfig(op.ReConfigArgs, applyLog.CommandIndex)
				} else {
					kv.DPrintf(fmt.Sprintf("[panic] unexpected LogType %v", op))
					kv.Kill()
					return
				}
			} else if applyLog.SnapshotValid {
				kv.DPrintf("======== recieve apply snap: %d ========", applyLog.SnapshotIndex)
				if applyLog.SnapshotIndex <= kv.lastApplyIndex {
					kv.DPrintf("***** snap index %d is older than lastApplyIndex %d *****",
						applyLog.SnapshotIndex, kv.lastApplyIndex)
					continue
				}

				r := bytes.NewBuffer(applyLog.Snapshot)
				d := labgob.NewDecoder(r)
				kv.Lock("applySnap")
				kv.db = make(map[string]string)
				var dupReqHistorySnap DupReqHistorySnap
				if d.Decode(&kv.db) != nil || d.Decode(&dupReqHistorySnap) != nil {
					kv.DPrintf("[panic] decode snap error")
					kv.Unlock("applySnap")
					kv.Kill()
					return
				}
				kv.restoreDupReqHistorySnap(dupReqHistorySnap)
				kv.Unlock("applySnap")

				// lastApplyIndex 到快照之间的修改请求一定会包含在查重哈希表里
				// 对于读只需要让客户端重新尝试即可
				kv.Lock("replyCommand")
				reply := &CommonReply{
					Err: ApplySnap,
				}
				for idx := kv.lastApplyIndex + 1; idx <= applyLog.SnapshotIndex; idx++ {
					if _, ok := kv.notifyChans[idx]; ok {
						select {
						case kv.notifyChans[idx] <- reply:
						default:
						}
					}
				}
				kv.Unlock("replyCommand")
				kv.lastApplyIndex = applyLog.SnapshotIndex
			} else {
				kv.DPrintf(fmt.Sprintf("[panic] unexpected applyLog %v", applyLog))
				kv.Kill()
				return
			}
		default:
			continue
		}
	}
}

//
// the tester calls Kill() when a ShardKV instance won't
// be needed again. you are not required to do anything
// in Kill(), but it might be convenient to (for example)
// turn off debug output from this instance.
//
func (kv *ShardKV) Kill() {
	atomic.StoreInt32(&kv.dead, 1)
	kv.rf.Kill()
	// Your code here, if desired.
}

func (kv *ShardKV) killed() bool {
	z := atomic.LoadInt32(&kv.dead)
	return z == 1
}

func (kv *ShardKV) updateConfig() {
	ticker := time.NewTicker(time.Millisecond * 100)
	for kv.killed() == false {
		select {
		case <-kv.updateConfigCh:
		case <-ticker.C:
		}

		kv.RLock("queryNum")
		queryNum := len(kv.configs)
		kv.RUnlock("queryNum")
		cfg := kv.mck.Query(queryNum)

		kv.Lock("checkConfigUpdate")
		if cfg.Num == len(kv.configs) {
			kv.configs = append(kv.configs, cfg)
			kv.DPrintf("===== kv.configs update: =====", kv.configs)
			for i := range kv.configs {
				kv.DPrintf("%+v", kv.configs[i])
			}
		}
		kv.Unlock("checkConfigUpdate")
	}
}

//
// servers[] contains the ports of the servers in this group.
//
// me is the index of the current server in servers[].
//
// the k/v server should store snapshots through the underlying Raft
// implementation, which should call persister.SaveStateAndSnapshot() to
// atomically save the Raft state along with the snapshot.
//
// the k/v server should snapshot when Raft's saved state exceeds
// maxraftstate bytes, in order to allow Raft to garbage-collect its
// log. if maxraftstate is -1, you don't need to snapshot.
//
// gid is this group's GID, for interacting with the shardctrler.
//
// pass ctrlers[] to shardctrler.MakeClerk() so you can send
// RPCs to the shardctrler.
//
// make_end(servername) turns a server name from a
// Config.Groups[gid][i] into a labrpc.ClientEnd on which you can
// send RPCs. You'll need this to send RPCs to other groups.
//
// look at client.go for examples of how to use ctrlers[]
// and make_end() to send RPCs to the group owning a specific shard.
//
// StartServer() must return quickly, so it should start goroutines
// for any long-running work.
//
func StartServer(servers []*labrpc.ClientEnd, me int, persister *raft.Persister, maxraftstate int, gid int, ctrlers []*labrpc.ClientEnd, make_end func(string) *labrpc.ClientEnd) *ShardKV {
	// call labgob.Register on structures you want
	// Go's RPC library to marshall/unmarshall.
	labgob.Register(Op{})

	kv := new(ShardKV)
	kv.me = me
	kv.applyCh = make(chan raft.ApplyMsg, 6)
	kv.servers = servers
	kv.rf = raft.Make(servers, me, persister, kv.applyCh)

	kv.maxraftstate = maxraftstate

	kv.make_end = make_end
	kv.gid = gid
	kv.ctrlers = ctrlers
	// Use something like this to talk to the shardctrler:
	kv.mck = shardctrler.MakeClerk(kv.ctrlers)

	kv.configs = make([]shardctrler.Config, 1)
	kv.configs[0].Groups = map[int][]string{}

	kv.shardState = make(map[int]*ShardState, shardctrler.NShards)
	for i := 0; i < shardctrler.NShards; i++ {
		kv.shardState[i] = &ShardState{
			state:      Working,
			prevCfg:    &kv.configs[0],
			currentCfg: &kv.configs[0],
		}
	}
	kv.updateConfigCh = make(chan struct{})

	kv.lastApplyIndex = 0
	kv.db = make(map[string]string)
	kv.notifyChans = make(map[int]chan *CommonReply)
	kv.dupReqHistory = make(map[int64]map[int64]struct{})

	go kv.handleApply()      // 处理 raft apply
	go kv.updateConfig()     // 负责从 shardctrler 拉取配置，写入配置更新日志
	go kv.writeReConfigLog() // 负责写入配置更改日志

	return kv
}
