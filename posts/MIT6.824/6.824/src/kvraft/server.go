package kvraft

import (
	"6.824/labgob"
	"6.824/labrpc"
	"6.824/raft"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

const Debug = true

func (kv *KVServer) DPrintf(format string, a ...interface{}) {
	if Debug {
		log.Printf(fmt.Sprintf("[KVServer %d]:%s", kv.me, format), a...)
	}
	return
}

type Op = CommonArgs

func (kv *KVServer) Lock(owner string) {
	//kv.DPrintf("%s Lock", owner)
	kv.mu.Lock()
}

func (kv *KVServer) Unlock(owner string) {
	//kv.DPrintf("%s Unlock", owner)
	kv.mu.Unlock()
}

type KVServer struct {
	mu      sync.Mutex
	me      int
	rf      *raft.Raft
	applyCh chan raft.ApplyMsg
	dead    int32 // set by Kill()

	maxraftstate int // snapshot if log grows this big

	db            map[string]string               // 内存数据库
	notifyChans   map[int]chan *CommonReply       // 监听请求 apply
	dupReqHistory map[int64]map[int64]CommonReply // 记录已经执行的命令，防止重复执行
}

func (kv *KVServer) Command(args *CommonArgs, reply *CommonReply) {
	// 请求重复则直接返回之前执行的结果
	if kv.isDuplicateRequest(args.ClientId, args.SequenceNum) {
		replyHistory := kv.dupReqHistory[args.ClientId][args.SequenceNum]
		reply.Err, reply.Value = replyHistory.Err, replyHistory.Value
		return
	}
	/*
	 * 如果 raft 崩溃了，那么 index 是可能回退的，因为它并不代表已提交
	 * 但是我们只要确保 index 有对应的通道即可，因为对于同一个 index，一定只会 apply 一次
	 * 对于 apply 超时，我们也要关闭通道，因为重新选主之后，这个 index 的通道可能再也用不到了
	 */
	index, _, isLeader := kv.rf.Start(*args)
	if !isLeader {
		reply.Err = ErrWrongLeader
		return
	}

	kv.Lock("Command")
	if _, ok := kv.notifyChans[index]; !ok {
		kv.notifyChans[index] = make(chan *CommonReply)
	}
	kv.Unlock("Command")

	select {
	case result := <-kv.notifyChans[index]:
		reply.Err, reply.Value = result.Err, result.Value
	case <-time.After(ExecuteTimeout):
		kv.DPrintf("wait apply log %d time out", index)
		reply.Err = ErrTimeout
	}

	go func() {
		kv.Lock("Command")
		defer kv.Unlock("Command")
		close(kv.notifyChans[index])
		delete(kv.notifyChans, index)
	}()
}

func (kv *KVServer) Get(args *CommonArgs, reply *CommonReply) {
	kv.Command(args, reply)
}

func (kv *KVServer) PutAppend(args *CommonArgs, reply *CommonReply) {
	kv.Command(args, reply)
}

func (kv *KVServer) updateDupReqHistory(clientId, sequenceNum int64, result CommonReply) {
	if _, ok := kv.dupReqHistory[clientId]; !ok {
		kv.dupReqHistory[clientId] = make(map[int64]CommonReply)
	}
	kv.dupReqHistory[clientId][sequenceNum] = result
}

func (kv *KVServer) isDuplicateRequest(clientId, sequenceNum int64) bool {
	if _, ok := kv.dupReqHistory[clientId]; ok {
		if _, ok := kv.dupReqHistory[clientId][sequenceNum]; ok {
			return true
		}
	}

	return false
}

func (kv *KVServer) handleApply() {
	for kv.killed() == false {
		select {
		case applyLog := <-kv.applyCh:
			if applyLog.CommandValid {
				op, ok := applyLog.Command.(Op)
				if !ok {
					panic("recieved apply log's command error")
				}

				reply := &CommonReply{
					Err: OK,
				}

				kv.DPrintf("recieve apply log: %d, op info: %+v", applyLog.CommandIndex, op)
				// 防止重复应用同一条命令
				if kv.isDuplicateRequest(op.ClientId, op.SequenceNum) {
					kv.DPrintf("found duplicate request: %+v", op)
					continue
				}

				value, ok := kv.db[op.Key]
				if op.Op == OpGet {
					if ok {
						reply.Value = value
						kv.DPrintf("get <%s>:<%s>", op.Key, value)
					} else {
						reply.Err = ErrNoKey
					}
				} else {
					if op.Op == OpAppend && ok {
						kv.db[op.Key] += op.Value
					} else {
						kv.db[op.Key] = op.Value
					}
					kv.DPrintf("update <%s>:<%s>", op.Key, kv.db[op.Key])
				}
				kv.updateDupReqHistory(op.ClientId, op.SequenceNum, *reply)

				go func() {
					// 只有 leader 需要回复
					_, isleader := kv.rf.GetState()
					if !isleader {
						return
					}

					kv.Lock("Command")
					if _, ok := kv.notifyChans[applyLog.CommandIndex]; ok {
						kv.notifyChans[applyLog.CommandIndex] <- reply
					}
					kv.Unlock("Command")
				}()
			} else if applyLog.SnapshotValid {
				kv.DPrintf("recieve apply snap: %d", applyLog.SnapshotIndex)
				kv.DPrintf("xxxxxxxxxxx TODO: handle snap")
			} else {
				panic(fmt.Sprintf("unexpected applyLog %v", applyLog))
			}
		default:
			continue
		}
	}
}

//
// the tester calls Kill() when a KVServer instance won't
// be needed again. for your convenience, we supply
// code to set rf.dead (without needing a lock),
// and a killed() method to test rf.dead in
// long-running loops. you can also add your own
// code to Kill(). you're not required to do anything
// about this, but it may be convenient (for example)
// to suppress debug output from a Kill()ed instance.
//
func (kv *KVServer) Kill() {
	atomic.StoreInt32(&kv.dead, 1)
	kv.rf.Kill()
	// Your code here, if desired.
}

func (kv *KVServer) killed() bool {
	z := atomic.LoadInt32(&kv.dead)
	return z == 1
}

//
// servers[] contains the ports of the set of
// servers that will cooperate via Raft to
// form the fault-tolerant key/value service.
// me is the index of the current server in servers[].
// the k/v server should store snapshots through the underlying Raft
// implementation, which should call persister.SaveStateAndSnapshot() to
// atomically save the Raft state along with the snapshot.
// the k/v server should snapshot when Raft's saved state exceeds maxraftstate bytes,
// in order to allow Raft to garbage-collect its log. if maxraftstate is -1,
// you don't need to snapshot.
// StartKVServer() must return quickly, so it should start goroutines
// for any long-running work.
//
func StartKVServer(servers []*labrpc.ClientEnd, me int, persister *raft.Persister, maxraftstate int) *KVServer {
	// call labgob.Register on structures you want
	// Go's RPC library to marshall/unmarshall.
	labgob.Register(Op{})

	kv := new(KVServer)
	kv.me = me
	kv.maxraftstate = maxraftstate

	kv.applyCh = make(chan raft.ApplyMsg, 1024)
	kv.rf = raft.Make(servers, me, persister, kv.applyCh)

	kv.db = make(map[string]string, 1024)
	kv.notifyChans = make(map[int]chan *CommonReply, 1024)
	kv.dupReqHistory = make(map[int64]map[int64]CommonReply)

	go kv.handleApply() // 处理 raft apply

	return kv
}
