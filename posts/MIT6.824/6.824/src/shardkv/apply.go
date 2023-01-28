package shardkv

import (
	"6.824/labgob"
	"6.824/shardctrler"
	"bytes"
)

func (kv *ShardKV) applyCommand(command *CommandArgs, applyLogIndex int) {
	reply := &CommonReply{Err: OK}
	var value string
	var ok bool

	kv.Lock("applyCommand")
	defer kv.Unlock("applyCommand")
	// 防止重复应用同一条修改命令
	if command.Op != OpGet && kv.isDuplicateLog(CommandLog, command.ClientId, command.SequenceNum) {
		kv.DPrintf("apply duplicate CommandLog: %+v", command)
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
	if command.Op != OpGet {
		kv.updateDupLog(CommandLog, command.ClientId, command.SequenceNum)
	}
	if _, ok = kv.notifyChans[applyLogIndex]; ok {
		select {
		case kv.notifyChans[applyLogIndex] <- reply:
		default:
			kv.DPrintf("reply to chan index %d failed", applyLogIndex)
		}
	}
}

func (kv *ShardKV) applyReConfig(args *ReConfigLogArgs, applyLogIndex int) {
	reply := &CommonReply{Err: OK}

	kv.Lock("applyReConfig")
	defer kv.Unlock("applyReConfig")
	// 防止重复应用同一条修改命令
	if kv.isDuplicateLog(ReConfigLog, int64(args.PrevCfg.Num), int64(args.UpdateCfg.Num)) {
		kv.DPrintf("apply duplicate ReConfigLog: %+v", args)
		goto replyCommand
	}

	if args.UpdateCfg.Num == len(kv.configs) {
		kv.configs = append(kv.configs, args.UpdateCfg)
		kv.DPrintf("update configs: %+v", kv.configs)
	}
	kv.updateDupLog(ReConfigLog, int64(args.PrevCfg.Num), int64(args.UpdateCfg.Num))

replyCommand:
	if _, ok := kv.notifyChans[applyLogIndex]; ok {
		select {
		case kv.notifyChans[applyLogIndex] <- reply:
		default:
			kv.DPrintf("reply to chan index %d failed", applyLogIndex)
		}
	}
}

func (kv *ShardKV) applyPullShard(args *PullShardLogArgs, applyLogIndex int) {
	reply := &CommonReply{Err: OK}
	var prevGID, nowGID int

	kv.Lock("applyPullShard")
	defer kv.Unlock("applyPullShard")
	// 防止重复应用同一条修改命令
	if kv.isDuplicateLog(PullShardLog, int64(args.Shard), int64(args.PrevCfg.Num)) {
		kv.DPrintf("apply duplicate PullShardLog: %+v", args)
		goto replyCommand
	}

	kv.updateDupLog(PullShardLog, int64(args.Shard), int64(args.PrevCfg.Num))
	kv.shardState[args.Shard].state = ReConfining
	kv.shardState[args.Shard].prevCfg = &args.PrevCfg
	kv.shardState[args.Shard].currentCfg = &args.UpdateCfg
	kv.DPrintf("update pull shard %d prevCfg:%+v, currentCfg: %+v", args.Shard,
		kv.shardState[args.Shard].prevCfg, kv.shardState[args.Shard].currentCfg)

	prevGID = kv.shardState[args.Shard].prevCfg.Shards[args.Shard]
	nowGID = kv.shardState[args.Shard].currentCfg.Shards[args.Shard]
	if nowGID == kv.gid {
		if prevGID == kv.gid || prevGID == 0 {
			kv.DPrintf("shard %d' gid not change: cfg num up to %d", args.Shard, kv.shardState[args.Shard].currentCfg.Num)
			kv.shardState[args.Shard].state = Working
		} else {
			go kv.pullShard(*kv.shardState[args.Shard].prevCfg, args.Shard, prevGID)
		}
	} else {
		if prevGID != kv.gid {
			kv.DPrintf("shard %d' gid not change: cfg num up to %d", args.Shard, kv.shardState[args.Shard].currentCfg.Num)
			kv.shardState[args.Shard].state = Working
		} else {
			// 等待被拉取分片
		}
	}
	if prevGID == kv.gid && nowGID == 0 {
		kv.DPrintf("[panic] nowGID is 0")
		kv.Kill()
		return
	}

replyCommand:
	if _, ok := kv.notifyChans[applyLogIndex]; ok {
		select {
		case kv.notifyChans[applyLogIndex] <- reply:
		default:
			kv.DPrintf("reply to chan index %d failed", applyLogIndex)
		}
	}
}

func (kv *ShardKV) applyUpdateShard(args *UpdateShardLogArgs, applyLogIndex int) {
	reply := &CommonReply{Err: OK}

	kv.Lock("applyUpdateShard")
	defer kv.Unlock("applyUpdateShard")
	// 防止重复应用同一条修改命令
	if kv.isDuplicateLog(UpdateShardLog, int64(args.Shard), int64(args.ShardCfgNum)) {
		kv.DPrintf("apply duplicate UpdateShardLog: %+v", args)
		goto replyCommand
	}

	kv.updateDupLog(UpdateShardLog, int64(args.Shard), int64(args.ShardCfgNum))
	kv.shardState[args.Shard].state = Working
	for k, v := range args.Data {
		kv.db[k] = v
	}

replyCommand:
	if _, ok := kv.notifyChans[applyLogIndex]; ok {
		select {
		case kv.notifyChans[applyLogIndex] <- reply:
		default:
			kv.DPrintf("reply to chan index %d failed", applyLogIndex)
		}
	}
}

func (kv *ShardKV) makeSnap(applyLogIndex int) {
	kv.DPrintf("======== snapshot %d ========", applyLogIndex)

	kv.Lock("makeSnap")
	defer kv.Unlock("makeSnap")
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	if e.Encode(kv.db) != nil || e.Encode(kv.configs) != nil || e.Encode(kv.shardState) != nil ||
		e.Encode(kv.dupCommand) != nil || e.Encode(kv.dupReconfig) != nil ||
		e.Encode(kv.dupPullShard) != nil || e.Encode(kv.dupUpdateShard) != nil {
		kv.DPrintf("[panic] encode snap error")
		kv.Kill()
		return
	}
	data := w.Bytes()
	kv.DPrintf("snap size: %d", len(data))
	kv.rf.Snapshot(applyLogIndex, data)
}

func (kv *ShardKV) restoreFromSnap(snapshot []byte, snapshotIndex int) {
	kv.Lock("restoreFromSnap")
	defer kv.Unlock("restoreFromSnap")

	kv.db = make(map[string]string)
	kv.configs = make([]shardctrler.Config, 1)
	kv.shardState = make(map[int]*ShardState, shardctrler.NShards)
	kv.dupCommand = make(map[int64]map[int64]struct{})
	kv.dupReconfig = make(map[int64]map[int64]struct{})
	kv.dupPullShard = make(map[int64]map[int64]struct{})
	kv.dupUpdateShard = make(map[int64]map[int64]struct{})
	r := bytes.NewBuffer(snapshot)
	d := labgob.NewDecoder(r)
	if d.Decode(&kv.db) != nil || d.Decode(&kv.configs) != nil || d.Decode(&kv.shardState) != nil ||
		d.Decode(&kv.dupCommand) != nil || d.Decode(&kv.dupReconfig) != nil ||
		d.Decode(&kv.dupPullShard) != nil || d.Decode(&kv.dupUpdateShard) != nil {
		kv.DPrintf("[panic] decode snap error")
		kv.Kill()
		return
	}

	// lastApplyIndex 到快照之间的修改请求一定会包含在查重哈希表里
	// 对于读只需要让客户端重新尝试即可
	reply := &CommonReply{
		Err: ApplySnap,
	}
	for idx := kv.lastApplyIndex + 1; idx <= snapshotIndex; idx++ {
		if _, ok := kv.notifyChans[idx]; ok {
			select {
			case kv.notifyChans[idx] <- reply:
			default:
			}
		}
	}
	kv.lastApplyIndex = snapshotIndex
	return
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

				switch op.LogType {
				case CommandLog:
					kv.DPrintf("recieve command apply log: %d, CommandArgs: %+v",
						applyLog.CommandIndex, op.CommandArgs)
					kv.applyCommand(op.CommandArgs, applyLog.CommandIndex)
				case ReConfigLog:
					kv.DPrintf("recieve reConfig apply log: %d, ReConfigLogArgs: %+v",
						applyLog.CommandIndex, op.ReConfigLogArgs)
					kv.applyReConfig(op.ReConfigLogArgs, applyLog.CommandIndex)
				case PullShardLog:
					kv.DPrintf("recieve pull shard apply log: %d, PullShardLogArgs: %+v",
						applyLog.CommandIndex, op.PullShardLogArgs)
					kv.applyPullShard(op.PullShardLogArgs, applyLog.CommandIndex)
				case UpdateShardLog:
					kv.DPrintf("recieve update shard apply log: %d, UpdateShardLogArgs: %+v",
						applyLog.CommandIndex, op.UpdateShardLogArgs)
					kv.applyUpdateShard(op.UpdateShardLogArgs, applyLog.CommandIndex)
				default:
					kv.DPrintf("[panic] unexpected LogType %+v", op)
					kv.Kill()
					return
				}

				// 检测是否需要执行快照
				if kv.rf.RaftStateNeedSnapshot(kv.maxraftstate) {
					kv.makeSnap(applyLog.CommandIndex)
				}
			} else if applyLog.SnapshotValid {
				kv.DPrintf("======== recieve apply snap: %d ========", applyLog.SnapshotIndex)
				if applyLog.SnapshotIndex <= kv.lastApplyIndex {
					kv.DPrintf("***** snap index %d is older than lastApplyIndex %d *****",
						applyLog.SnapshotIndex, kv.lastApplyIndex)
					continue
				}
				kv.restoreFromSnap(applyLog.Snapshot, applyLog.SnapshotIndex)
			} else {
				kv.DPrintf("[panic] unexpected applyLog %+v", applyLog)
				kv.Kill()
				return
			}
		default:
			continue
		}
	}
}
