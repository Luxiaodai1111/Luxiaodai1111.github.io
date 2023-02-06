package shardkv

import (
	"time"
)

func (kv *ShardKV) Command(args *CommandArgs, reply *CommonReply) {
	kv.RLock("checkShard")
	if kv.checkShard(args.Key, reply) {
		kv.RUnlock("checkShard")
		kv.DPrintf("checkShard reply request %s %s: %s", args.Op, args.Key, reply.Err)
		return
	}
	kv.RUnlock("checkShard")

	kv.RLock("Command")
	if args.Op != OpGet && kv.isDupModifyReq(key2shard(args.Key), args.ClientId, args.SequenceNum) {
		kv.DPrintf("duplicate command request: %+v, reply history response", args)
		reply.Err = OK
		kv.RUnlock("Command")
		return
	}
	kv.RUnlock("Command")

	index, term, isLeader := kv.rf.Start(Op{
		LogType:     CommandLog,
		CommandArgs: args,
	})
	if !isLeader {
		reply.Err = ErrWrongLeader
		return
	}

	kv.notifyChansLock.Lock()
	if _, ok := kv.notifyChans[index]; !ok {
		kv.notifyChans[index] = make(chan *CommonReply)
	}
	kv.notifyChansLock.Unlock()

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
		kv.DPrintf("wait command apply log %d time out", index)
		reply.Err = ErrTimeout
	}

	kv.notifyChansLock.Lock()
	defer kv.notifyChansLock.Unlock()
	close(kv.notifyChans[index])
	delete(kv.notifyChans, index)
}

func (kv *ShardKV) Get(args *CommandArgs, reply *CommonReply) {
	kv.Command(args, reply)
}

func (kv *ShardKV) PutAppend(args *CommandArgs, reply *CommonReply) {
	kv.Command(args, reply)
}

func (kv *ShardKV) replyCommon(LogType string, index, term int, isLeader bool, reply *CommonReply) {
	if !isLeader {
		reply.Err = ErrWrongLeader
		return
	}

	kv.notifyChansLock.Lock()
	if _, ok := kv.notifyChans[index]; !ok {
		kv.notifyChans[index] = make(chan *CommonReply)
	}
	kv.notifyChansLock.Unlock()

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
			reply.Err = OK
		} else {
			reply.Err, reply.Value = result.Err, result.Value
		}
	case <-time.After(WriteLogTimeout):
		kv.DPrintf("wait %s apply log %d time out", LogType, index)
		reply.Err = ErrTimeout
	}

	kv.notifyChansLock.Lock()
	defer kv.notifyChansLock.Unlock()
	close(kv.notifyChans[index])
	delete(kv.notifyChans, index)
}

func (kv *ShardKV) ReConfigLog(args *ReConfigLogArgs, reply *CommonReply) {
	kv.RLock("ReConfigLog")
	if args.UpdateCfg.Num < len(kv.configs) {
		kv.RUnlock("ReConfigLog")
		kv.DPrintf("duplicate reConfig request: %+v, reply history response", args)
		reply.Err = OK
		return
	}
	kv.RUnlock("ReConfigLog")

	index, term, isLeader := kv.rf.Start(Op{
		LogType:         ReConfigLog,
		ReConfigLogArgs: args,
	})

	kv.replyCommon(ReConfigLog, index, term, isLeader, reply)
}

func (kv *ShardKV) PullShardLog(args *PullShardLogArgs, reply *CommonReply) {
	kv.RLock("PullShardLog")
	if kv.shardState[args.Shard].CurrentCfg.Num >= args.UpdateCfg.Num {
		kv.RUnlock("PullShardLog")
		kv.DPrintf("duplicate pull shard request: %+v, reply history response", args)
		reply.Err = OK
		return
	}
	kv.RUnlock("PullShardLog")

	index, term, isLeader := kv.rf.Start(Op{
		LogType:          PullShardLog,
		PullShardLogArgs: args,
	})

	kv.replyCommon(PullShardLog, index, term, isLeader, reply)
}

func (kv *ShardKV) UpdateShardLog(args *UpdateShardLogArgs, reply *CommonReply) {
	// 在调用更外层（pullShard）去重

	index, term, isLeader := kv.rf.Start(Op{
		LogType:            UpdateShardLog,
		UpdateShardLogArgs: args,
	})

	kv.replyCommon(UpdateShardLog, index, term, isLeader, reply)
}

func (kv *ShardKV) DeleteShardLog(args *DeleteShardArgs, reply *CommonReply) {
	// 去重同 UpdateShardLog

	index, term, isLeader := kv.rf.Start(Op{
		LogType:         DeleteShardLog,
		DeleteShardArgs: args,
	})

	kv.replyCommon(DeleteShardLog, index, term, isLeader, reply)
}

func (kv *ShardKV) PullShard(args *PullShardArgs, reply *PullShardReply) {
	_, isleader := kv.rf.GetState()
	if !isleader {
		reply.Err = ErrWrongLeader
		return
	}

	kv.DPrintf("recv pullShard %d request: ", args.Shard)

	kv.Lock("PullShard")
	defer kv.Unlock("PullShard")

	actualShardCfgNum := kv.shardState[args.Shard].CurrentCfg.Num
	if kv.shardState[args.Shard].State != Working && kv.shardState[args.Shard].State != PrepareReConfig {
		actualShardCfgNum = kv.shardState[args.Shard].PrevCfg.Num
	}
	kv.DPrintf("now state: %s, %d, %d", kv.shardState[args.Shard].State, args.PrevCfg.Num, actualShardCfgNum)
	if args.PrevCfg.Num == actualShardCfgNum {
		// 必须等到本服务器也开始迁移分片才能回复，否则数据库数据是不完全的
		if kv.shardState[args.Shard].State != WaitingToBePulled {
			reply.Err = ErrRetry
			return
		}
	} else if args.PrevCfg.Num < actualShardCfgNum {
		// 考虑快照会导致状态跳变，当分片配置大于参数配置时，表示一定曾经拉取成功了
		// 但是回复不一定可以成功到达，所以这里也需要回复完整的数据，这里分片的数据不会变得比参数更新，因为如果要变化，需要依赖拉取服务器先更新成功
	} else {
		reply.Err = ErrRetry
		return
	}

	reply.Err = OK
	reply.Data = kv.pullData[args.Shard]
	reply.DupModifyCommand = kv.dupModifyCommand[args.Shard]

	kv.DPrintf("=== reply %+v pullShard %d success, cfg num up to %d ===", reply, args.Shard, kv.shardState[args.Shard].CurrentCfg.Num)
	// 每个分片拉取或被拉取成功都表示这个分片可以开始服务了
	// 写入删除分片日志，同步 Working 状态
	go kv.WriteLog(DeleteShardLog, DeleteShardArgs{
		Shard:       args.Shard,
		ShardCfgNum: args.PrevCfg.Num,
	})
}
