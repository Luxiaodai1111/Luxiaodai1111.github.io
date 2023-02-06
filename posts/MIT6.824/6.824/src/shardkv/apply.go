package shardkv

import (
	"context"
	"fmt"
)

func (kv *ShardKV) applyCommand(command *CommandArgs, applyLogIndex int) {
	reply := &CommonReply{Err: OK}
	var value string
	var ok bool

	shard := key2shard(command.Key)
	kv.Lock("applyCommand")
	// 检查当前是否服务分片
	if kv.checkShard(command.Key, reply) {
		goto replyCommand
	}

	// 检查重复请求
	if command.Op != OpGet && kv.isDupModifyReq(shard, command.ClientId, command.SequenceNum) {
		kv.DPrintf("apply duplicate command: %+v", command)
		kv.Unlock("applyCommand")
		return
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
	if command.Op != OpGet {
		kv.updateDupModifyReq(shard, command.ClientId, command.SequenceNum)
	}

replyCommand:
	kv.Unlock("applyCommand")

	kv.notifyChansLock.Lock()
	defer kv.notifyChansLock.Unlock()
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
	// 防止重复应用同一条修改命令
	if args.UpdateCfg.Num < len(kv.configs) {
		kv.DPrintf("apply duplicate ReConfigLog: %+v", args)
	} else if args.UpdateCfg.Num == len(kv.configs) {
		kv.configs = append(kv.configs, args.UpdateCfg)
		kv.DPrintf("update configs: %+v", kv.configs)
	} else {
		panic(fmt.Sprintf("applyReConfig args:%+v kv.configs:%+v", args, kv.configs))
	}
	kv.Unlock("applyReConfig")

	kv.notifyChansLock.Lock()
	defer kv.notifyChansLock.Unlock()
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
	// 防止重复应用同一条修改命令
	if kv.shardState[args.Shard].CurrentCfg.Num >= args.UpdateCfg.Num {
		kv.DPrintf("apply duplicate PullShardLog: %+v", args)
		goto replyCommand
	} else {
		if kv.shardState[args.Shard].CurrentCfg.Num+1 != args.UpdateCfg.Num {
			panic(fmt.Sprintf("applyPullShard shard %d CurrentCfg.Num %d, args.UpdateCfg.Num %d",
				args.Shard, kv.shardState[args.Shard].CurrentCfg.Num, args.UpdateCfg.Num))
		}
	}

	kv.shardState[args.Shard].PrevCfg = &args.PrevCfg
	kv.shardState[args.Shard].CurrentCfg = &args.UpdateCfg
	kv.DPrintf("update pull shard %d prevCfg:%+v, currentCfg: %+v", args.Shard,
		kv.shardState[args.Shard].PrevCfg, kv.shardState[args.Shard].CurrentCfg)

	prevGID = kv.shardState[args.Shard].PrevCfg.Shards[args.Shard]
	nowGID = kv.shardState[args.Shard].CurrentCfg.Shards[args.Shard]
	if nowGID == kv.gid {
		if prevGID == kv.gid || prevGID == 0 {
			kv.DPrintf("shard %d' gid not change, now cfg num is %d", args.Shard, kv.shardState[args.Shard].CurrentCfg.Num)
			kv.shardState[args.Shard].State = Working
		} else {
			// 由 kv.pullShard 异步去拉取分片
			kv.shardState[args.Shard].State = PreparePull
		}
	} else {
		if prevGID != kv.gid {
			kv.DPrintf("shard %d' gid not change, now cfg num is %d", args.Shard, kv.shardState[args.Shard].CurrentCfg.Num)
			kv.shardState[args.Shard].State = Working
		} else {
			// 等待被拉取分片
			kv.shardState[args.Shard].State = WaitingToBePulled
			data := make(map[string]string)
			for k, v := range kv.db {
				if key2shard(k) == args.Shard {
					data[k] = v
				}
			}
			kv.pullData[args.Shard] = data
		}
	}
	if prevGID == kv.gid && nowGID == 0 {
		panic(fmt.Sprintf("[panic] nowGID is 0"))
	}

replyCommand:
	kv.Unlock("applyPullShard")

	kv.notifyChansLock.Lock()
	defer kv.notifyChansLock.Unlock()
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
	prevGID := kv.shardState[args.Shard].PrevCfg.Shards[args.Shard]
	nowGID := kv.shardState[args.Shard].CurrentCfg.Shards[args.Shard]
	state := kv.shardState[args.Shard].State
	if (state == PreparePull || state == Pulling) &&
		kv.shardState[args.Shard].PrevCfg.Num == args.ShardCfgNum &&
		nowGID == kv.gid && prevGID != kv.gid && prevGID != 0 {
		kv.shardState[args.Shard].State = Working
		for k, v := range args.Data {
			kv.db[k] = v
		}
		kv.pullData[args.Shard] = make(map[string]string)

		for clientId, sequenceNum := range args.DupModifyCommand {
			kv.updateDupModifyReq(args.Shard, clientId, sequenceNum)
		}
		kv.DPrintf("update shard %d data success, now cfg num is %d", args.Shard, kv.shardState[args.Shard].CurrentCfg.Num)
	} else {
		kv.DPrintf("apply duplicate UpdateShardLog: %+v", args)
	}
	kv.Unlock("applyUpdateShard")

	kv.notifyChansLock.Lock()
	defer kv.notifyChansLock.Unlock()
	if _, ok := kv.notifyChans[applyLogIndex]; ok {
		select {
		case kv.notifyChans[applyLogIndex] <- reply:
		default:
			kv.DPrintf("reply to chan index %d failed", applyLogIndex)
		}
	}
}

func (kv *ShardKV) applyDeleteShard(args *DeleteShardArgs, applyLogIndex int) {
	reply := &CommonReply{Err: OK}

	kv.Lock("applyDeleteShard")
	// 防止重复应用同一条修改命令
	if kv.shardState[args.Shard].State == WaitingToBePulled && kv.shardState[args.Shard].PrevCfg.Num == args.ShardCfgNum {
		kv.shardState[args.Shard].State = Working
		for k, _ := range kv.pullData[args.Shard] {
			delete(kv.db, k)
		}
		kv.DPrintf("delete shard %d success, now cfg num is %d", args.Shard, kv.shardState[args.Shard].CurrentCfg.Num)
	} else {
		kv.DPrintf("apply duplicate DeleteShardLog: %+v", args)
	}
	kv.Unlock("applyDeleteShard")

	kv.notifyChansLock.Lock()
	defer kv.notifyChansLock.Unlock()
	if _, ok := kv.notifyChans[applyLogIndex]; ok {
		select {
		case kv.notifyChans[applyLogIndex] <- reply:
		default:
			kv.DPrintf("reply to chan index %d failed", applyLogIndex)
		}
	}
}

func (kv *ShardKV) handleApply(ctx context.Context) {
	for kv.killed() == false {
		select {
		case applyLog := <-kv.applyCh:
			if applyLog.CommandValid {
				op, ok := applyLog.Command.(Op)
				if !ok {
					panic(fmt.Sprintf("[panic] recieved apply log's command error: %+v", applyLog))
				}

				if applyLog.CommandIndex <= kv.lastApplyIndex {
					// 比如 raft 重启了，就要重新 apply
					kv.DPrintf("***** command index %d is older than lastApplyIndex %d *****",
						applyLog.CommandIndex, kv.lastApplyIndex)
					continue
				}
				kv.lastApplyIndex = applyLog.CommandIndex

				switch op.LogType {
				case NoOpLog:
					// nothing to do
					kv.DPrintf("recieve no op apply log: %d", applyLog.CommandIndex)
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
				case DeleteShardLog:
					kv.DPrintf("recieve delete shard apply log: %d, DeleteShardArgs: %+v",
						applyLog.CommandIndex, op.DeleteShardArgs)
					kv.applyDeleteShard(op.DeleteShardArgs, applyLog.CommandIndex)
				default:
					panic(fmt.Sprintf("[panic] unexpected LogType %+v", op))
				}

				// 检测是否需要执行快照
				if kv.rf.RaftStateNeedSnapshot(kv.maxraftstate) {
					kv.makeSnap(applyLog.CommandIndex)
				}
			} else if applyLog.SnapshotValid {
				kv.DPrintf("======== recieve apply snap: %d, lastApplyIndex %d ========",
					applyLog.SnapshotIndex, kv.lastApplyIndex)
				if applyLog.SnapshotIndex <= kv.lastApplyIndex {
					kv.DPrintf("***** snap index %d is older than lastApplyIndex %d *****",
						applyLog.SnapshotIndex, kv.lastApplyIndex)
					continue
				}
				kv.restoreFromSnap(applyLog.Snapshot, applyLog.SnapshotIndex)
			} else {
				panic(fmt.Sprintf("[panic] unexpected applyLog %+v", applyLog))
			}
		case <-ctx.Done():
			return
		default:
			continue
		}
	}
}
