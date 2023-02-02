package shardkv

func (kv *ShardKV) updateDupModifyReq(clientId, sequenceNum int64) {
	if sequenceNum > kv.dupModifyCommand[clientId] {
		kv.dupModifyCommand[clientId] = sequenceNum
	}
}

func (kv *ShardKV) isDupModifyReq(clientId, sequenceNum int64) bool {
	if sequenceNum <= kv.dupModifyCommand[clientId] {
		return true
	}

	return false
}
