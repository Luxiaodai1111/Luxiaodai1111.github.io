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

type Op struct {
	ClientId    int64  // 客户端标识
	SequenceNum int64  // 请求序号
	Op          string // "Put" or "Append" or "Get"
	Key         string
	Value       string
}

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

	db          map[string]string // 内存数据库
	notifyChans map[int]chan Op   // 监听请求 apply
}

func (kv *KVServer) Get(args *GetArgs, reply *GetReply) {
	index, _, isLeader := kv.rf.Start(Op{
		ClientId:    args.ClientId,
		SequenceNum: args.SequenceNum,
		Op:          OpGet,
		Key:         args.Key,
	})
	if isLeader {
		// TODO: ch 处理
		kv.notifyChans[index] = make(chan Op)

		select {
		case result := <-kv.notifyChans[index]:
			reply.Err = OK
			kv.DPrintf("get <%s>:<%s>", result.Key, kv.db[result.Key])
			reply.Value = kv.db[result.Key]
		case <-time.After(ExecuteTimeout):
			reply.Err = ErrTimeout
		}
		//close(kv.notifyChans[index])
		//delete(kv.notifyChans, index)
	} else {
		reply.Err = ErrWrongLeader
	}
}

func (kv *KVServer) PutAppend(args *PutAppendArgs, reply *PutAppendReply) {
	index, _, isLeader := kv.rf.Start(Op{
		ClientId:    args.ClientId,
		SequenceNum: args.SequenceNum,
		Op:          args.Op,
		Key:         args.Key,
		Value:       args.Value,
	})
	if isLeader {
		// TODO: ch 处理
		kv.notifyChans[index] = make(chan Op)

		select {
		case result := <-kv.notifyChans[index]:
			// 更新状态机
			_, ok := kv.db[result.Key]
			if result.Op == OpAppend && ok {
				kv.db[result.Key] += result.Value
			} else {
				kv.db[result.Key] = result.Value
			}
			kv.DPrintf("update <%s>:<%s>", result.Key, result.Value)
			reply.Err = OK
		case <-time.After(ExecuteTimeout):
			reply.Err = ErrTimeout
		}
		//close(kv.notifyChans[index])
		//delete(kv.notifyChans, index)
	} else {
		reply.Err = ErrWrongLeader
	}
}

func (kv *KVServer) handleApply() {
	for {
		if kv.killed() {
			return
		}

		select {
		case applyLog := <-kv.applyCh:
			if applyLog.SnapshotValid {
				kv.DPrintf("recieve apply snap: %d", applyLog.SnapshotIndex)
			} else {
				kv.DPrintf("recieve apply log: %d", applyLog.CommandIndex)
			}
			op, ok := applyLog.Command.(Op)
			if ok {
				if applyLog.SnapshotValid {
					kv.DPrintf("xxxxxxxxxxx TODO: handle snap")
				} else {
					kv.notifyChans[applyLog.CommandIndex] <- op
				}
			} else {
				kv.DPrintf("recieved apply log's command error")
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

	kv.applyCh = make(chan raft.ApplyMsg)
	kv.rf = raft.Make(servers, me, persister, kv.applyCh)

	kv.db = make(map[string]string, 1024)
	kv.notifyChans = make(map[int]chan Op, 1024)

	go kv.handleApply() // 处理 raft apply

	return kv
}
