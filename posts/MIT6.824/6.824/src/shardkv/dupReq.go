package shardkv

func (kv *ShardKV) updateDupLog(logType string, k1, k2 int64) {
	var dupMap map[int64]map[int64]struct{}
	if logType == CommandLog {
		dupMap = kv.dupCommand
	} else if logType == ReConfigLog {
		dupMap = kv.dupReconfig
	} else if logType == PullShardLog {
		dupMap = kv.dupPullShard
	} else if logType == UpdateShardLog {
		dupMap = kv.dupUpdateShard
	} else if logType == DeleteShardLog {
		dupMap = kv.dupDeleteShard
	}
	if _, ok := dupMap[k1]; !ok {
		dupMap[k1] = make(map[int64]struct{})
	}
	dupMap[k1][k2] = struct{}{}
}

func (kv *ShardKV) isDuplicateLogWithLock(logType string, k1, k2 int64) bool {
	kv.RLock("isDuplicateLogWithLock")
	defer kv.RUnlock("isDuplicateLogWithLock")
	return kv.isDuplicateLog(logType, k1, k2)
}

func (kv *ShardKV) isDuplicateLog(logType string, k1, k2 int64) bool {
	var dupMap map[int64]map[int64]struct{}
	if logType == CommandLog {
		dupMap = kv.dupCommand
	} else if logType == ReConfigLog {
		dupMap = kv.dupReconfig
	} else if logType == PullShardLog {
		dupMap = kv.dupPullShard
	} else if logType == UpdateShardLog {
		dupMap = kv.dupUpdateShard
	} else if logType == DeleteShardLog {
		dupMap = kv.dupDeleteShard
	}
	if _, ok := dupMap[k1]; ok {
		if _, ok := dupMap[k1][k2]; ok {
			return true
		}
	}

	return false
}
