package kvraft

import (
	"6.824/labgob"
	"6.824/labrpc"
	"6.824/raft"
	"bytes"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

const Debug = false

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

func (kv *KVServer) RLock(owner string) {
	//kv.DPrintf("%s RLock", owner)
	kv.mu.RLock()
}

func (kv *KVServer) RUnlock(owner string) {
	//kv.DPrintf("%s RUnlock", owner)
	kv.mu.RUnlock()
}

type KVServer struct {
	mu      sync.RWMutex
	me      int
	rf      *raft.Raft
	applyCh chan raft.ApplyMsg
	dead    int32 // set by Kill()

	maxraftstate   int // snapshot if log grows this big
	lastApplyIndex int

	db               map[string]string         // 内存数据库
	notifyChans      map[int]chan *CommonReply // 监听请求 apply
	dupModifyCommand map[int64]int64           // 记录每个客户端最后一条修改成功的序列号，防止重复执行
}

func (kv *KVServer) Command(args *CommonArgs, reply *CommonReply) {
	kv.RLock("Command")
	if args.Op != OpGet && kv.isDupModifyReq(args.ClientId, args.SequenceNum) {
		kv.DPrintf("duplicate command request: %+v, reply history response", args)
		reply.Err = OK
		kv.RUnlock("Command")
		return
	}
	kv.RUnlock("Command")

	/*
	 * 要使用 term 和 index 来代表一条日志
	 * 对于 apply 超时，我们也要关闭通道，因为重新选主之后，这个通道再也用不到了
	 */
	index, term, isLeader := kv.rf.Start(*args)
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

func (kv *KVServer) Get(args *CommonArgs, reply *CommonReply) {
	kv.Command(args, reply)
}

func (kv *KVServer) PutAppend(args *CommonArgs, reply *CommonReply) {
	kv.Command(args, reply)
}

func (kv *KVServer) updateDupModifyReq(clientId, sequenceNum int64) {
	if _, ok := kv.dupModifyCommand[clientId]; !ok {
		kv.dupModifyCommand[clientId] = sequenceNum
	}
	if sequenceNum > kv.dupModifyCommand[clientId] {
		kv.dupModifyCommand[clientId] = sequenceNum
	}
}

func (kv *KVServer) isDupModifyReq(clientId, sequenceNum int64) bool {
	if _, ok := kv.dupModifyCommand[clientId]; ok {
		if sequenceNum <= kv.dupModifyCommand[clientId] {
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
					panic(fmt.Sprintf("[panic] recieved apply log's command error"))
				}

				reply := &CommonReply{
					Err: OK,
				}
				var value string

				if applyLog.CommandIndex <= kv.lastApplyIndex {
					// 比如 raft 重启了，就要重新 apply
					kv.DPrintf("***** command index %d is older than lastApplyIndex %d *****",
						applyLog.CommandIndex, kv.lastApplyIndex)
					continue
				}
				kv.lastApplyIndex = applyLog.CommandIndex

				kv.DPrintf("recieve apply log: %d, op info: %+v", applyLog.CommandIndex, op)
				// 检查重复请求
				if op.Op != OpGet && kv.isDupModifyReq(op.ClientId, op.SequenceNum) {
					kv.DPrintf("apply duplicate command: %+v", op)
					// 同一客户端某个序号请求没有返回成功就不会有大于它的序号请求产生，所以重复的请求没有必要回复
					continue
				}

				// 更新状态机
				value, ok = kv.db[op.Key]
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

				kv.Lock("replyCommand")
				if op.Op != OpGet {
					kv.updateDupModifyReq(op.ClientId, op.SequenceNum)
				}

				/*
				 * 只要有通道存在，说明可能是当前 leader，也可能曾经作为 leader 接收过请求
				 * 通道可能处于等待消息状态，或者正在返回错误等待销毁，所以不管怎么样，都往通道里返回消息
				 * 如果已经销毁，说明已经返回了等待超时错误
				 */
				if _, ok := kv.notifyChans[applyLog.CommandIndex]; ok {
					select {
					case kv.notifyChans[applyLog.CommandIndex] <- reply:
					default:
						kv.DPrintf("reply to chan index %d failed", applyLog.CommandIndex)
					}
				}
				kv.Unlock("replyCommand")

				// 检测是否需要执行快照
				if kv.rf.RaftStateNeedSnapshot(kv.maxraftstate) {
					kv.DPrintf("======== snapshot %d ========", applyLog.CommandIndex)
					w := new(bytes.Buffer)
					e := labgob.NewEncoder(w)
					kv.Lock("snap")
					if e.Encode(kv.db) != nil || e.Encode(kv.dupModifyCommand) != nil {
						kv.Unlock("snap")
						panic(fmt.Sprintf("[panic] encode snap error"))
					}
					kv.Unlock("snap")
					data := w.Bytes()
					kv.DPrintf("snap size: %d", len(data))
					kv.rf.Snapshot(applyLog.CommandIndex, data)
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
				if d.Decode(&kv.db) != nil || d.Decode(&kv.dupModifyCommand) != nil {
					kv.Unlock("applySnap")
					panic(fmt.Sprintf("[panic] decode snap error"))
				}
				kv.Unlock("applySnap")

				// lastApplyIndex 到快照之间的修改请求一定会包含在查重哈希表里
				// 对于读只需要让客户端重新尝试即可满足线性一致
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
				kv.lastApplyIndex = applyLog.SnapshotIndex
				kv.Unlock("replyCommand")
			} else {
				panic(fmt.Sprintf("[panic] unexpected applyLog %+v", applyLog))
			}
		default:
			continue
		}
	}
}

// the tester calls Kill() when a KVServer instance won't
// be needed again. for your convenience, we supply
// code to set rf.dead (without needing a lock),
// and a killed() method to test rf.dead in
// long-running loops. you can also add your own
// code to Kill(). you're not required to do anything
// about this, but it may be convenient (for example)
// to suppress debug output from a Kill()ed instance.
func (kv *KVServer) Kill() {
	atomic.StoreInt32(&kv.dead, 1)
	kv.rf.Kill()
	// Your code here, if desired.
}

func (kv *KVServer) killed() bool {
	z := atomic.LoadInt32(&kv.dead)
	return z == 1
}

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
func StartKVServer(servers []*labrpc.ClientEnd, me int, persister *raft.Persister, maxraftstate int) *KVServer {
	// call labgob.Register on structures you want
	// Go's RPC library to marshall/unmarshall.
	labgob.Register(Op{})

	kv := new(KVServer)
	kv.me = me
	kv.maxraftstate = maxraftstate
	kv.lastApplyIndex = 0

	kv.applyCh = make(chan raft.ApplyMsg, 6)
	kv.rf = raft.Make(servers, me, persister, kv.applyCh)

	kv.db = make(map[string]string)
	kv.notifyChans = make(map[int]chan *CommonReply)
	kv.dupModifyCommand = make(map[int64]int64)

	go kv.handleApply() // 处理 raft apply

	return kv
}
