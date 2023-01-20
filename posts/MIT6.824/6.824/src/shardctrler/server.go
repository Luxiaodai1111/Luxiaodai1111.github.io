package shardctrler

import (
	"6.824/raft"
	"fmt"
	"log"
	"sync/atomic"
	"time"
)
import "6.824/labrpc"
import "sync"
import "6.824/labgob"

const Debug = false

func (sc *ShardCtrler) DPrintf(format string, a ...interface{}) {
	if Debug {
		log.Printf(fmt.Sprintf("[KVServer %d]:%s", sc.me, format), a...)
	}
	return
}

func (sc *ShardCtrler) Lock(owner string) {
	//kv.DPrintf("%s Lock", owner)
	sc.mu.Lock()
}

func (sc *ShardCtrler) Unlock(owner string) {
	//kv.DPrintf("%s Unlock", owner)
	sc.mu.Unlock()
}

type ShardCtrler struct {
	mu      sync.Mutex
	me      int
	rf      *raft.Raft
	dead    int32 // set by Kill()
	applyCh chan raft.ApplyMsg

	lastApplyIndex int

	configs       []Config                     // indexed by config num
	notifyChans   map[int]chan *CommonReply    // 监听请求 apply
	dupReqHistory map[int64]map[int64]struct{} // 记录已经执行的修改命令，防止重复执行
}

type Op = CommonArgs

func (sc *ShardCtrler) Command(args *CommonArgs, reply *CommonReply) {
	if args.Op != OpQuery && sc.isDuplicateRequest(args.ClientId, args.SequenceNum) {
		sc.DPrintf("found duplicate request: %+v, reply history response", args)
		reply.Err = OK
		return
	}

	index, term, isLeader := sc.rf.Start(*args)
	if !isLeader {
		reply.Err = ErrWrongLeader
		return
	}

	sc.Lock("getNotifyChan")
	if _, ok := sc.notifyChans[index]; !ok {
		sc.notifyChans[index] = make(chan *CommonReply)
	}
	sc.Unlock("getNotifyChan")

	select {
	case result := <-sc.notifyChans[index]:
		currentTerm, isleader := sc.rf.GetState()
		if !isleader || currentTerm != term {
			reply.Err = ErrWrongLeader
			sc.DPrintf("reply now is not leader")
			return
		}

		reply.Err, reply.Config = result.Err, result.Config
	case <-time.After(ExecuteTimeout):
		sc.DPrintf("wait apply log %d time out", index)
		reply.Err = ErrTimeout
	}

	sc.Lock("Command")
	defer sc.Unlock("Command")
	close(sc.notifyChans[index])
	delete(sc.notifyChans, index)
}

func (sc *ShardCtrler) Join(args *CommonArgs, reply *CommonReply) {
	sc.Command(args, reply)
}

func (sc *ShardCtrler) Leave(args *CommonArgs, reply *CommonReply) {
	sc.Command(args, reply)
}

func (sc *ShardCtrler) Move(args *CommonArgs, reply *CommonReply) {
	sc.Command(args, reply)
}

func (sc *ShardCtrler) Query(args *CommonArgs, reply *CommonReply) {
	sc.Command(args, reply)
}

//
// the tester calls Kill() when a ShardCtrler instance won't
// be needed again. you are not required to do anything
// in Kill(), but it might be convenient to (for example)
// turn off debug output from this instance.
//
func (sc *ShardCtrler) Kill() {
	atomic.StoreInt32(&sc.dead, 1)
	sc.rf.Kill()
	// Your code here, if desired.
}

func (sc *ShardCtrler) killed() bool {
	z := atomic.LoadInt32(&sc.dead)
	return z == 1
}

// needed by shardkv tester
func (sc *ShardCtrler) Raft() *raft.Raft {
	return sc.rf
}

func (sc *ShardCtrler) updateDupReqHistory(clientId, sequenceNum int64) {
	if _, ok := sc.dupReqHistory[clientId]; !ok {
		sc.dupReqHistory[clientId] = make(map[int64]struct{})
	}
	sc.dupReqHistory[clientId][sequenceNum] = struct{}{}
}

func (sc *ShardCtrler) isDuplicateRequest(clientId, sequenceNum int64) bool {
	sc.Lock("isDuplicateRequest")
	defer sc.Unlock("isDuplicateRequest")
	if _, ok := sc.dupReqHistory[clientId]; ok {
		if _, ok := sc.dupReqHistory[clientId][sequenceNum]; ok {
			return true
		}
	}

	return false
}

func (sc *ShardCtrler) balanceShard(groups map[int][]string, deletedGIDs []int) [NShards]int {
	lastNum := len(sc.configs) - 1
	shardMap := sc.configs[lastNum].Shards
	gidShardLoadInfo := make(map[int][]int)
	noGidShardList := make([]int, 0)
	// 统计当前服务器负载分布
	for gid, _ := range groups {
		gidShardLoadInfo[gid] = make([]int, 0)
	}
	for idx := range shardMap {
		gid := shardMap[idx]
		if gid == 0 {
			noGidShardList = append(noGidShardList, idx)
			continue
		}
		if _, ok := gidShardLoadInfo[gid]; ok {
			// 记录 GID 负责的 shard
			gidShardLoadInfo[gid] = append(gidShardLoadInfo[gid], idx)
		} else {
			noGidShardList = append(noGidShardList, idx)
		}
	}
	// TODO:不再工作的 GID 把它的 shard 分配给当前 shard 负载最低的 GID
	if len(noGidShardList) > 0 {

	}

	// TODO:平均每个 GID 的负载，每次均衡最大和最小负载的 GID，直到他们差值为 1 或 0

	return shardMap
}

func (sc *ShardCtrler) handleApply() {
	for sc.killed() == false {
		select {
		case applyLog := <-sc.applyCh:
			if applyLog.CommandValid {
				op, ok := applyLog.Command.(Op)
				if !ok {
					sc.DPrintf("[panic] recieved apply log's command error")
					sc.Kill()
					return
				}

				reply := &CommonReply{
					Err: OK,
				}

				if applyLog.CommandIndex <= sc.lastApplyIndex {
					// 比如 raft 重启了，就要重新 apply
					sc.DPrintf("***** command index %d is older than lastApplyIndex %d *****",
						applyLog.CommandIndex, sc.lastApplyIndex)
					continue
				}
				sc.lastApplyIndex = applyLog.CommandIndex

				sc.DPrintf("recieve apply log: %d, op info: %+v", applyLog.CommandIndex, op)
				// 防止重复应用同一条修改命令
				if op.Op != OpQuery && sc.isDuplicateRequest(op.ClientId, op.SequenceNum) {
					sc.DPrintf("found duplicate request: %+v", op)
					continue
				}

				// 更新状态机
				if op.Op == OpJoin {
					lastNum := len(sc.configs) - 1
					groups := sc.configs[lastNum].Groups
					for gid, servers := range op.Servers {
						groups[gid] = servers
					}
					newShards := sc.balanceShard(groups, []int{})
					sc.configs = append(sc.configs, Config{
						Num:    lastNum + 1,
						Shards: newShards,
						Groups: groups,
					})
				} else if op.Op == OpLeave {
					lastNum := len(sc.configs) - 1
					groups := sc.configs[lastNum].Groups
					for idx := range op.GIDs {
						gid := op.GIDs[idx]
						delete(groups, gid)
					}
					newShards := sc.balanceShard(groups, op.GIDs)
					sc.configs = append(sc.configs, Config{
						Num:    lastNum + 1,
						Shards: newShards,
						Groups: groups,
					})
				} else if op.Op == OpMove {
					lastNum := len(sc.configs) - 1
					shardsMap := sc.configs[lastNum].Shards
					if op.Shard < 0 || op.Shard > NShards-1 {
						sc.DPrintf("move args error")
						sc.Kill()
						return
					}
					shardsMap[op.Shard] = op.GID
					sc.configs = append(sc.configs, Config{
						Num:    lastNum + 1,
						Shards: shardsMap,
						Groups: sc.configs[lastNum].Groups,
					})
				} else {
					reply.Config = sc.configs[op.Num]
				}

				sc.Lock("replyCommand")
				if op.Op != OpQuery {
					sc.updateDupReqHistory(op.ClientId, op.SequenceNum)
				}

				if _, ok := sc.notifyChans[applyLog.CommandIndex]; ok {
					select {
					case sc.notifyChans[applyLog.CommandIndex] <- reply:
					default:
						sc.DPrintf("reply to chan index %d failed", applyLog.CommandIndex)
					}
				}
				sc.Unlock("replyCommand")
			} else {
				sc.DPrintf(fmt.Sprintf("[panic] unexpected applyLog %v", applyLog))
				sc.Kill()
				return
			}
		default:
			continue
		}
	}
}

//
// servers[] contains the ports of the set of
// servers that will cooperate via Raft to
// form the fault-tolerant shardctrler service.
// me is the index of the current server in servers[].
//
func StartServer(servers []*labrpc.ClientEnd, me int, persister *raft.Persister) *ShardCtrler {
	sc := new(ShardCtrler)
	sc.me = me
	sc.lastApplyIndex = 0

	sc.configs = make([]Config, 1)
	sc.configs[0].Groups = map[int][]string{}

	labgob.Register(Op{})
	sc.applyCh = make(chan raft.ApplyMsg, 6)
	sc.rf = raft.Make(servers, me, persister, sc.applyCh)

	// Your code here.
	sc.notifyChans = make(map[int]chan *CommonReply)
	sc.dupReqHistory = make(map[int64]map[int64]struct{})

	go sc.handleApply() // 处理 raft apply

	return sc
}
