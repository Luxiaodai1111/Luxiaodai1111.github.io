package shardkv

import (
	"6.824/labrpc"
	"6.824/shardctrler"
	"fmt"
	"log"
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

type Op struct {
	LogType          string
	CommandArgs      *CommandArgs
	ReConfigLogArgs  *ReConfigLogArgs
	PullShardLogArgs *PullShardLogArgs
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
	dupCommand     map[int64]map[int64]struct{} // 记录已经执行的修改命令，防止重复执行
	dupReconfig    map[int64]map[int64]struct{}
	dupPullShard   map[int64]map[int64]struct{}
}

func (kv *ShardKV) checkShard(key string, reply *CommonReply) bool {
	kv.RLock("checkShard")
	defer kv.RUnlock("checkShard")

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

func (kv *ShardKV) pullShard(prevCfg shardctrler.Config, shard, gid int) {
	kv.DPrintf("=== pullShard %d from %v ===", shard, gid)
	for kv.killed() {
		args := PullShardArgs{
			Shard:   shard,
			PrevCfg: prevCfg,
		}
		gidServers := args.PrevCfg.Groups[shard]

		// try each server for the shard.
		for si := 0; si < len(gidServers); si++ {
			srv := kv.make_end(gidServers[si])
			var reply PullShardReply
			ok := srv.Call("ShardKV.PullShard", &args, &reply)
			if ok && (reply.Err == OK) {
				kv.DPrintf("=== pullShard %d from %v success ===", shard, gid)
				kv.Lock("pullShard")
				for k, v := range reply.Data {
					kv.db[k] = v
				}
				kv.shardState[shard].state = Working
				kv.Unlock("pullShard")
				return
			}
		}

		time.Sleep(10 * time.Millisecond)
	}
}

// WriteLog 写入日志直到成功，重复写入也没关系，在 apply 处理即可
func (kv *ShardKV) WriteLog(logType string, args interface{}) {
	kv.DPrintf("===== write %s: %+v =====", logType, args)
	writeLogSuccess := false
	for !writeLogSuccess {
		_, isleader := kv.rf.GetState()
		if isleader {
			var reply CommonReply
			if logType == ReConfigLog {
				reConfigLogArgs, ok := args.(ReConfigLogArgs)
				if !ok {
					goto logTypePanic
				}
				kv.ReConfigLog(&reConfigLogArgs, &reply)
			} else if logType == PullShardLog {
				pullShardLogArgs, ok := args.(PullShardLogArgs)
				if !ok {
					goto logTypePanic
				}
				kv.PullShardLog(&pullShardLogArgs, &reply)
			} else {
				goto logTypePanic
			}
			if reply.Err == OK {
				writeLogSuccess = true
			}
		} else {
			// try each server for the shard.
			for si := 0; si < len(kv.servers); si++ {
				srv := kv.servers[si]
			retry:
				var reply CommonReply
				var ok bool
				if logType == ReConfigLog {
					ok = srv.Call("ShardKV.ReConfigLog", &args, &reply)
				} else if logType == PullShardLog {
					ok = srv.Call("ShardKV.PullShardLog", &args, &reply)
				} else {
					goto logTypePanic
				}
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
	kv.DPrintf("===== write %s success: %+v =====", logType, args)
	return

logTypePanic:
	kv.DPrintf("[panic] error %s args %+v", logType, args)
	kv.Kill()
	return
}

func (kv *ShardKV) updateConfig() {
	ticker := time.NewTicker(time.Millisecond * 100)
	for kv.killed() == false {
		select {
		case <-kv.updateConfigCh:
		case <-ticker.C:
		}

		_, isleader := kv.rf.GetState()
		if !isleader {
			continue
		}

		kv.RLock("queryNum")
		queryNum := len(kv.configs)
		kv.RUnlock("queryNum")
		cfg := kv.mck.Query(queryNum)

		kv.RLock("checkConfigUpdate")
		if cfg.Num == len(kv.configs) {
			args := ReConfigLogArgs{
				PrevCfg:   kv.configs[len(kv.configs)-1],
				UpdateCfg: cfg,
			}
			kv.RUnlock("checkConfigUpdate")

			kv.WriteLog(ReConfigLog, args)

			kv.Lock("updateConfig")
			kv.configs = append(kv.configs, cfg)
			kv.Unlock("updateConfig")
		} else {
			kv.RUnlock("checkConfigUpdate")
		}
	}
}

func (kv *ShardKV) updatePullShardLog() {
	for kv.killed() == false {
		time.Sleep(time.Millisecond)

		_, isleader := kv.rf.GetState()
		if !isleader {
			continue
		}

		kv.Lock("updatePullShardLog")
		for shard, info := range kv.shardState {
			if info.state == Working && kv.shardState[shard].currentCfg.Num+1 < len(kv.configs) {
				kv.shardState[shard].state = PrepareReConfig
				args := PullShardLogArgs{
					Shard:     shard,
					PrevCfg:   *kv.shardState[shard].currentCfg,
					UpdateCfg: kv.configs[kv.shardState[shard].currentCfg.Num+1],
				}
				kv.shardState[shard].prevCfg = &args.PrevCfg
				kv.shardState[shard].currentCfg = &args.UpdateCfg
				go kv.WriteLog(PullShardLog, args)
			}
		}
		kv.Unlock("updatePullShardLog")
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
	labgob.Register(CommandArgs{})
	labgob.Register(ReConfigLogArgs{})
	labgob.Register(PullShardLogArgs{})

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
	kv.dupCommand = make(map[int64]map[int64]struct{})
	kv.dupReconfig = make(map[int64]map[int64]struct{})
	kv.dupPullShard = make(map[int64]map[int64]struct{})

	go kv.updateConfig()       // 负责从 shardctrler 拉取配置，写入配置更新日志
	go kv.updatePullShardLog() // 负责写入配置更改日志
	go kv.handleApply()        // 处理 raft apply

	return kv
}
