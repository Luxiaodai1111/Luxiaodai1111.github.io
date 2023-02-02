package shardkv

import (
	"6.824/labgob"
	"6.824/shardctrler"
	"bytes"
	"fmt"
)

type ShardStateSnap map[int]*ShardStateSimple

type ShardStateSimple struct {
	State         string
	PrevCfgNum    int
	CurrentCfgNum int
}

func (kv *ShardKV) encodeShardState() ShardStateSnap {
	snap := make(ShardStateSnap)
	for shard, info := range kv.shardState {
		snap[shard] = &ShardStateSimple{
			State:         info.State,
			PrevCfgNum:    info.PrevCfg.Num,
			CurrentCfgNum: info.CurrentCfg.Num,
		}
	}
	return snap
}

func (kv *ShardKV) decodeShardState(snap ShardStateSnap) {
	kv.shardState = make(map[int]*ShardState, shardctrler.NShards)
	for shard, info := range snap {
		kv.shardState[shard] = &ShardState{
			State:      info.State,
			PrevCfg:    &kv.configs[info.PrevCfgNum],
			CurrentCfg: &kv.configs[info.CurrentCfgNum],
		}
	}
}

func (kv *ShardKV) makeSnap(applyLogIndex int) {
	kv.DPrintf("======== snapshot %d ========", applyLogIndex)

	kv.Lock("makeSnap")
	defer kv.Unlock("makeSnap")
	shardStateSnap := kv.encodeShardState()
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	if e.Encode(kv.db) != nil || e.Encode(kv.configs) != nil ||
		e.Encode(shardStateSnap) != nil || e.Encode(kv.dupModifyCommand) != nil {
		panic(fmt.Sprintf("[panic] encode snap error"))
	}
	data := w.Bytes()
	kv.DPrintf("snap size: %d", len(data))
	kv.rf.Snapshot(applyLogIndex, data)
}

func (kv *ShardKV) restoreFromSnap(snapshot []byte, snapshotIndex int) {
	kv.Lock("restoreFromSnap")
	var shardStateSnap ShardStateSnap
	r := bytes.NewBuffer(snapshot)
	d := labgob.NewDecoder(r)
	if d.Decode(&kv.db) != nil || d.Decode(&kv.configs) != nil ||
		d.Decode(&shardStateSnap) != nil || d.Decode(&kv.dupModifyCommand) != nil {
		panic(fmt.Sprintf("[panic] decode snap error"))
	}
	kv.decodeShardState(shardStateSnap)
	kv.lastApplyIndex = snapshotIndex
	kv.Unlock("restoreFromSnap")

	// lastApplyIndex 到快照之间的修改请求一定会包含在查重哈希表里
	// 对于读只需要让客户端重新尝试即可
	reply := &CommonReply{
		Err: ApplySnap,
	}
	kv.notifyChansLock.Lock()
	for idx := kv.lastApplyIndex + 1; idx <= snapshotIndex; idx++ {
		if _, ok := kv.notifyChans[idx]; ok {
			select {
			case kv.notifyChans[idx] <- reply:
			default:
			}
		}
	}
	kv.notifyChansLock.Unlock()
	return
}
