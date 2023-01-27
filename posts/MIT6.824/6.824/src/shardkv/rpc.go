package shardkv

import "time"

func (kv *ShardKV) Command(args *CommandArgs, reply *CommonReply) {
	if args.Op != OpGet && kv.isDuplicateLog(CommandLog, args.ClientId, args.SequenceNum) {
		kv.DPrintf("duplicate command request: %+v, reply history response", args)
		reply.Err = OK
		return
	}

	if kv.checkShard(args.Key, reply) {
		kv.DPrintf("checkShard reply request %s %s: %s", args.Op, args.Key, reply.Err)
		return
	}

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

func (kv *ShardKV) ReConfigLog(args *ReConfigLogArgs, reply *CommonReply) {
	if kv.isDuplicateLog(ReConfigLog, int64(args.PrevCfg.Num), int64(args.UpdateCfg.Num)) {
		kv.DPrintf("duplicate reConfig request: %+v, reply history response", args)
		reply.Err = OK
		return
	}

	index, term, isLeader := kv.rf.Start(Op{
		LogType:         ReConfigLog,
		ReConfigLogArgs: args,
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
		currentTerm, isleader := kv.rf.GetState()
		if !isleader || currentTerm != term {
			reply.Err = ErrWrongLeader
			kv.DPrintf("reply now is not leader")
			return
		}
		kv.DPrintf("reply reConfig index: %d", index)

		if reply.Err == ApplySnap {
			reply.Err = ErrRetry
		} else {
			reply.Err, reply.Value = result.Err, result.Value
		}
	case <-time.After(ExecuteTimeout):
		kv.DPrintf("wait reConfig apply log %d time out", index)
		reply.Err = ErrTimeout
	}

	kv.Lock("ReConfig")
	defer kv.Unlock("ReConfig")
	close(kv.notifyChans[index])
	delete(kv.notifyChans, index)
}

func (kv *ShardKV) PullShardLog(args *PullShardLogArgs, reply *CommonReply) {
	if kv.isDuplicateLog(PullShardLog, int64(args.Shard), int64(args.PrevCfg.Num)) {
		kv.DPrintf("duplicate pull shard request: %+v, reply history response", args)
		reply.Err = OK
		return
	}

	index, term, isLeader := kv.rf.Start(Op{
		LogType:          PullShardLog,
		PullShardLogArgs: args,
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
		currentTerm, isleader := kv.rf.GetState()
		if !isleader || currentTerm != term {
			reply.Err = ErrWrongLeader
			kv.DPrintf("reply now is not leader")
			return
		}
		kv.DPrintf("reply pull shard index: %d", index)

		if reply.Err == ApplySnap {
			reply.Err = ErrRetry
		} else {
			reply.Err, reply.Value = result.Err, result.Value
		}
	case <-time.After(ExecuteTimeout):
		kv.DPrintf("wait pull shard apply log %d time out", index)
		reply.Err = ErrTimeout
	}

	kv.Lock("PullShard")
	defer kv.Unlock("PullShard")
	close(kv.notifyChans[index])
	delete(kv.notifyChans, index)
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
	if kv.shardState[args.Shard].state != Working {
		actualShardCfgNum = kv.shardState[args.Shard].prevCfg.Num
	}
	kv.DPrintf("now state: %s, %d, %d", kv.shardState[args.Shard].state, args.PrevCfg.Num, actualShardCfgNum)
	// TODO: 处理重复请求
	if args.PrevCfg.Num == actualShardCfgNum {
		data := make(map[string]string)
		for k, v := range kv.db {
			if key2shard(k) == args.Shard {
				data[k] = v
			}
		}
		reply.Err = OK
		reply.Data = data
		// 每个分片拉取或被拉取成功都表示这个分片可以开始服务了
		kv.DPrintf("=== reply pullShard %d success, cfg num up to %d ===", args.Shard, kv.shardState[args.Shard].currentCfg.Num)
		kv.shardState[args.Shard].state = Working
		// TODO: 写入删除分片日志
	}

	reply.Err = ErrRetry

}
