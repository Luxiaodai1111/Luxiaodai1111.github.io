package shardkv

func (kv *ShardKV) updateDupModifyReq(shard int, clientId, sequenceNum int64) {
	if _, ok := kv.dupModifyCommand[shard][clientId]; !ok {
		kv.dupModifyCommand[shard][clientId] = sequenceNum
	}
	if sequenceNum > kv.dupModifyCommand[shard][clientId] {
		kv.dupModifyCommand[shard][clientId] = sequenceNum
	}
}

func (kv *ShardKV) isDupModifyReq(shard int, clientId, sequenceNum int64) bool {
	if _, ok := kv.dupModifyCommand[shard][clientId]; ok {
		if sequenceNum <= kv.dupModifyCommand[shard][clientId] {
			return true
		}
	}

	return false
}
