package shardkv

import (
	"time"
)

func (kv *ShardKV) Command(args *CommandArgs, reply *CommonReply) {
	if args.Op != OpGet && kv.isDuplicateLogWithLock(CommandLog, args.ClientId, args.SequenceNum) {
		kv.DPrintf("duplicate command request: %+v, reply history response", args)
		reply.Err = OK
		return
	}

	kv.RLock("checkShard")
	if kv.checkShard(args.Key, reply) {
		kv.RUnlock("checkShard")
		kv.DPrintf("checkShard reply request %s %s: %s", args.Op, args.Key, reply.Err)
		return
	}
	kv.RUnlock("checkShard")

	index, term, isLeader := kv.rf.Start(Op{
		LogType:     CommandLog,
		CommandArgs: args,
	})
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
		kv.DPrintf("wait apply log %d time out", index)
		reply.Err = ErrTimeout
	}

	kv.Lock("Command")
	defer kv.Unlock("Command")
	close(kv.notifyChans[index])
	delete(kv.notifyChans, index)
}

func (kv *ShardKV) Get(args *CommandArgs, reply *CommonReply) {
	kv.Command(args, reply)
}

func (kv *ShardKV) PutAppend(args *CommandArgs, reply *CommonReply) {
	kv.Command(args, reply)
}

func (kv *ShardKV) replyCommon(index, term int, isLeader bool, reply *CommonReply) {
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
			reply.Err = OK
		} else {
			reply.Err, reply.Value = result.Err, result.Value
		}
	case <-time.After(ExecuteTimeout):
		kv.DPrintf("wait apply log %d time out", index)
		reply.Err = ErrTimeout
	}

	kv.Lock("cleanNotifyChan")
	defer kv.Unlock("cleanNotifyChan")
	close(kv.notifyChans[index])
	delete(kv.notifyChans, index)
}

func (kv *ShardKV) ReConfigLog(args *ReConfigLogArgs, reply *CommonReply) {
	if kv.isDuplicateLogWithLock(ReConfigLog, int64(args.PrevCfg.Num), int64(args.UpdateCfg.Num)) {
		kv.DPrintf("duplicate reConfig request: %+v, reply history response", args)
		reply.Err = OK
		return
	}

	index, term, isLeader := kv.rf.Start(Op{
		LogType:         ReConfigLog,
		ReConfigLogArgs: args,
	})

	kv.replyCommon(index, term, isLeader, reply)
}

func (kv *ShardKV) PullShardLog(args *PullShardLogArgs, reply *CommonReply) {
	if kv.isDuplicateLogWithLock(PullShardLog, int64(args.Shard), int64(args.PrevCfg.Num)) {
		kv.DPrintf("duplicate pull shard request: %+v, reply history response", args)
		reply.Err = OK
		return
	}

	index, term, isLeader := kv.rf.Start(Op{
		LogType:          PullShardLog,
		PullShardLogArgs: args,
	})

	kv.replyCommon(index, term, isLeader, reply)
}

func (kv *ShardKV) UpdateShardLog(args *UpdateShardLogArgs, reply *CommonReply) {
	if kv.isDuplicateLogWithLock(UpdateShardLog, int64(args.Shard), int64(args.ShardCfgNum)) {
		kv.DPrintf("duplicate update shard request: %+v, reply history response", args)
		reply.Err = OK
		return
	}

	index, term, isLeader := kv.rf.Start(Op{
		LogType:            UpdateShardLog,
		UpdateShardLogArgs: args,
	})

	kv.replyCommon(index, term, isLeader, reply)
}

func (kv *ShardKV) DeleteShardLog(args *DeleteShardArgs, reply *CommonReply) {
	if kv.isDuplicateLogWithLock(DeleteShardLog, int64(args.Shard), int64(args.PrevCfg.Num)) {
		kv.DPrintf("duplicate delete shard request: %+v, reply history response", args)
		reply.Err = OK
		return
	}

	index, term, isLeader := kv.rf.Start(Op{
		LogType:         DeleteShardLog,
		DeleteShardArgs: args,
	})

	kv.replyCommon(index, term, isLeader, reply)
}

func (kv *ShardKV) PullShard(args *PullShardArgs, reply *PullShardReply) {
	_, isleader := kv.rf.GetState()
	if !isleader {
		reply.Err = ErrWrongLeader
		return
	}

	kv.Lock("PullShard")
	defer kv.Unlock("PullShard")

	kv.DPrintf("recv pullShard %d request: ", args.Shard)
	actualShardCfgNum := kv.shardState[args.Shard].currentCfg.Num
	if kv.shardState[args.Shard].state == ReConfining {
		actualShardCfgNum = kv.shardState[args.Shard].prevCfg.Num
	}
	kv.DPrintf("now state: %s, %d, %d", kv.shardState[args.Shard].state, args.PrevCfg.Num, actualShardCfgNum)
	// TODO: 考虑快照会导致状态跳变
	if kv.shardState[args.Shard].state == ReConfining && args.PrevCfg.Num == kv.shardState[args.Shard].prevCfg.Num {
		// 必须等到本服务器也开始迁移分片才能回复，否则数据库数据是不完全的
		data := make(map[string]string)
		for k, v := range kv.db {
			if key2shard(k) == args.Shard {
				data[k] = v
			}
		}
		reply.Err = OK
		reply.Data = data

		kv.DPrintf("=== reply pullShard %d success, cfg num up to %d ===", args.Shard, kv.shardState[args.Shard].currentCfg.Num)
		// 每个分片拉取或被拉取成功都表示这个分片可以开始服务了
		// 写入删除分片日志，同步 Working 状态
		go kv.WriteLog(DeleteShardLog, DeleteShardArgs{
			Shard:   args.Shard,
			PrevCfg: args.PrevCfg,
		})
	} else if args.PrevCfg.Num < actualShardCfgNum {
		reply.Err = ErrDupPull
	} else {
		reply.Err = ErrRetry
	}
}
