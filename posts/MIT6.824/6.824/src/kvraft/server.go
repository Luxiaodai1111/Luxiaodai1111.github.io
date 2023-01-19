package kvraft

import (
	"6.824/labgob"
	"6.824/labrpc"
	"6.824/raft"
	"bytes"
	"fmt"
	"log"
	"strconv"
	"strings"
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

	db            map[string]string            // 内存数据库
	notifyChans   map[int]chan *CommonReply    // 监听请求 apply
	dupReqHistory map[int64]map[int64]struct{} // 记录已经执行的修改命令，防止重复执行
}

type DupReqHistorySnap map[int64]string

func (kv *KVServer) makeDupReqHistorySnap() DupReqHistorySnap {
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

func (kv *KVServer) restoreDupReqHistorySnap(snap DupReqHistorySnap) {
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

func (kv *KVServer) Command(args *CommonArgs, reply *CommonReply) {
	// 修改请求重复
	if args.Op != OpGet && kv.isDuplicateRequest(args.ClientId, args.SequenceNum) {
		kv.DPrintf("found duplicate request: %+v, reply history response", args)
		reply.Err = OK
		return
	}

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

func (kv *KVServer) updateDupReqHistory(clientId, sequenceNum int64) {
	if _, ok := kv.dupReqHistory[clientId]; !ok {
		kv.dupReqHistory[clientId] = make(map[int64]struct{})
	}
	kv.dupReqHistory[clientId][sequenceNum] = struct{}{}
}

func (kv *KVServer) isDuplicateRequest(clientId, sequenceNum int64) bool {
	kv.RLock("isDuplicateRequest")
	defer kv.RUnlock("isDuplicateRequest")
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
					kv.DPrintf("[panic] recieved apply log's command error")
					kv.Kill()
					return
				}

				reply := &CommonReply{
					Err: OK,
				}

				if applyLog.CommandIndex <= kv.lastApplyIndex {
					// 比如 raft 重启了，就要重新 apply
					kv.DPrintf("***** command index %d is older than lastApplyIndex %d *****",
						applyLog.SnapshotIndex, kv.lastApplyIndex)
					continue
				}
				kv.lastApplyIndex = applyLog.CommandIndex

				kv.DPrintf("recieve apply log: %d, op info: %+v", applyLog.CommandIndex, op)
				// 防止重复应用同一条修改命令
				if op.Op != OpGet && kv.isDuplicateRequest(op.ClientId, op.SequenceNum) {
					kv.DPrintf("found duplicate request: %+v", op)
					continue
				}

				// 更新状态机
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

				kv.Lock("replyCommand")
				if op.Op != OpGet {
					kv.updateDupReqHistory(op.ClientId, op.SequenceNum)
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
					}
				}
				kv.Unlock("replyCommand")

				// 检测是否需要执行快照
				if kv.rf.RaftStateNeedSnapshot(kv.maxraftstate) {
					kv.DPrintf("======== snapshot %d ========", applyLog.CommandIndex)
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
	kv.dupReqHistory = make(map[int64]map[int64]struct{})

	go kv.handleApply() // 处理 raft apply

	return kv
}
