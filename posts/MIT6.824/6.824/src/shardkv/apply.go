package shardkv

import (
	"6.824/labgob"
	"bytes"
	"fmt"
)

func (kv *ShardKV) applyCommand(command *CommandArgs, applyLogIndex int) {
	reply := &CommonReply{Err: OK}
	var value string
	var ok bool

	// 防止重复应用同一条修改命令
	if command.Op != OpGet && kv.isDuplicateLog(CommandLog, command.ClientId, command.SequenceNum) {
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
		kv.updateDupLog(CommandLog, command.ClientId, command.SequenceNum)
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
		dupReqHistorySnap := kv.makeDupLogHistorySnap(kv.dupCommand)
		dupReconfigHistorySnap := kv.makeDupLogHistorySnap(kv.dupReconfig)
		dupPullShardHistorySnap := kv.makeDupLogHistorySnap(kv.dupPullShard)
		if e.Encode(kv.db) != nil || e.Encode(dupReqHistorySnap) != nil ||
			e.Encode(dupReconfigHistorySnap) != nil || e.Encode(dupPullShardHistorySnap) != nil {
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

func (kv *ShardKV) applyReConfig(args *ReConfigLogArgs, applyLogIndex int) {
	reply := &CommonReply{Err: OK}

	// 防止重复应用同一条修改命令
	if kv.isDuplicateLog(ReConfigLog, int64(args.PrevCfg.Num), int64(args.UpdateCfg.Num)) {
		kv.DPrintf("found duplicate reConfig log: %+v", args)
		goto replyCommand
	}

	kv.Lock("applyReConfig")
	if args.UpdateCfg.Num == len(kv.configs) {
		kv.configs = append(kv.configs, args.UpdateCfg)
	}
	kv.updateDupLog(ReConfigLog, int64(args.PrevCfg.Num), int64(args.UpdateCfg.Num))
	kv.DPrintf("update configs: %+v", kv.configs)
	kv.Unlock("applyReConfig")

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

func (kv *ShardKV) applyPullShard(args *PullShardLogArgs, applyLogIndex int) {
	reply := &CommonReply{Err: OK}
	var prevGID, nowGID int

	// 防止重复应用同一条修改命令
	if kv.isDuplicateLog(PullShardLog, int64(args.Shard), int64(args.PrevCfg.Num)) {
		kv.DPrintf("found duplicate pullShard request: %+v", args)
		goto replyCommand
	}

	kv.Lock("checkShard")
	kv.updateDupLog(PullShardLog, int64(args.Shard), int64(args.PrevCfg.Num))
	kv.shardState[args.Shard].state = ReConfining
	kv.shardState[args.Shard].prevCfg = &args.PrevCfg
	kv.shardState[args.Shard].currentCfg = &args.UpdateCfg

	prevGID = kv.shardState[args.Shard].prevCfg.Shards[args.Shard]
	nowGID = kv.shardState[args.Shard].currentCfg.Shards[args.Shard]
	if nowGID == kv.gid {
		if prevGID == kv.gid || prevGID == 0 {
			kv.shardState[args.Shard].state = Working
		} else {
			go kv.pullShard(*kv.shardState[args.Shard].prevCfg, args.Shard, prevGID)
		}
	}
	if prevGID == kv.gid && nowGID == 0 {
		kv.DPrintf("[panic] nowGID is 0")
		kv.Kill()
		return
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
					kv.DPrintf("[panic] recieved apply log's command error: %+v", applyLog)
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
					kv.DPrintf("recieve command apply log: %d, CommandArgs: %+v",
						applyLog.CommandIndex, op.CommandArgs)
					kv.applyCommand(op.CommandArgs, applyLog.CommandIndex)
				} else if op.LogType == ReConfigLog {
					kv.DPrintf("recieve reConfig apply log: %d, ReConfigLogArgs: %+v",
						applyLog.CommandIndex, op.ReConfigLogArgs)
					kv.applyReConfig(op.ReConfigLogArgs, applyLog.CommandIndex)
				} else if op.LogType == PullShardLog {
					kv.DPrintf("recieve pull shard apply log: %d, PullShardLogArgs: %+v",
						applyLog.CommandIndex, op.PullShardLogArgs)
					kv.applyPullShard(op.PullShardLogArgs, applyLog.CommandIndex)
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
				var dupReconfigHistorySnap DupReqHistorySnap
				var dupPullShardHistorySnap DupReqHistorySnap
				if d.Decode(&kv.db) != nil || d.Decode(&dupReqHistorySnap) != nil ||
					d.Decode(&dupReconfigHistorySnap) != nil || d.Decode(&dupPullShardHistorySnap) != nil {
					kv.DPrintf("[panic] decode snap error")
					kv.Unlock("applySnap")
					kv.Kill()
					return
				}
				kv.dupCommand = kv.restoreDupHistorySnap(dupReqHistorySnap)
				kv.dupReconfig = kv.restoreDupHistorySnap(dupReconfigHistorySnap)
				kv.dupPullShard = kv.restoreDupHistorySnap(dupPullShardHistorySnap)
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
