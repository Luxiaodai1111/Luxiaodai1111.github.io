package shardkv

import (
	"6.824/labgob"
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

type DupHistorySnap map[int64]string

func (kv *ShardKV) makeDupHistorySnap(dupMap map[int64]map[int64]struct{}) DupHistorySnap {
	snap := make(DupHistorySnap, 0)
	for clientId, info := range dupMap {
		var seqs []int64
		for sequenceNum := range info {
			seqs = append(seqs, sequenceNum)
		}

		// 排序
		for i := 0; i <= len(seqs)-1; i++ {
			for j := i; j <= len(seqs)-1; j++ {
				if seqs[i] > seqs[j] {
					t := seqs[i]
					seqs[i] = seqs[j]
					seqs[j] = t
				}
			}
		}

		// 将所有序列号压缩(记录和前一条的差值)成一条字符串
		snapString := make([]string, len(seqs))
		var prev int64
		for idx, seq := range seqs {
			if idx == 0 {
				snapString = append(snapString, strconv.FormatInt(seq, 10))
			} else {
				snapString = append(snapString, strconv.FormatInt(seq-prev, 10))
			}
			prev = seq
		}

		snap[clientId] = strings.Join(snapString, "")
	}

	return snap
}

func (kv *ShardKV) restoreDupHistorySnap(snap DupHistorySnap) map[int64]map[int64]struct{} {
	dupMap := make(map[int64]map[int64]struct{})
	for clientId, info := range snap {
		if _, ok := dupMap[clientId]; !ok {
			dupMap[clientId] = make(map[int64]struct{})
		}

		snapString := strings.Split(info, "")
		var prev int64
		for idx, value := range snapString {
			if idx == 0 {
				seq, _ := strconv.ParseInt(value, 10, 64)
				prev = seq
			} else {
				seq, _ := strconv.ParseInt(value, 10, 64)
				prev += seq
			}
			dupMap[clientId][prev] = struct{}{}
		}
	}

	return dupMap
}

func (kv *ShardKV) makeSnap(applyLogIndex int) {
	kv.DPrintf("======== snapshot %d ========", applyLogIndex)

	kv.Lock("makeSnap")
	defer kv.Unlock("makeSnap")
	dupCommandHistorySnap := kv.makeDupHistorySnap(kv.dupCommand)
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	if e.Encode(kv.db) != nil || e.Encode(kv.configs) != nil || e.Encode(kv.shardState) != nil || e.Encode(dupCommandHistorySnap) != nil {
		panic(fmt.Sprintf("[panic] encode snap error"))
	}
	data := w.Bytes()
	kv.DPrintf("snap size: %d", len(data))
	kv.rf.Snapshot(applyLogIndex, data)
}

func (kv *ShardKV) restoreFromSnap(snapshot []byte, snapshotIndex int) {
	kv.Lock("restoreFromSnap")
	defer kv.Unlock("restoreFromSnap")

	var dupCommandHistorySnap DupHistorySnap
	r := bytes.NewBuffer(snapshot)
	d := labgob.NewDecoder(r)
	if d.Decode(&kv.db) != nil || d.Decode(&kv.configs) != nil || d.Decode(&kv.shardState) != nil || d.Decode(&dupCommandHistorySnap) != nil {
		panic(fmt.Sprintf("[panic] decode snap error"))
	}
	kv.dupCommand = kv.restoreDupHistorySnap(dupCommandHistorySnap)

	for shard, info := range kv.shardState {
		if info.State == ReConfining {
			kv.DPrintf("recieve snap, reConfining shard %d need handle", shard)
			prevGID := info.PrevCfg.Shards[shard]
			nowGID := info.CurrentCfg.Shards[shard]
			if nowGID == kv.gid {
				if prevGID == kv.gid || prevGID == 0 {
					kv.DPrintf("shard %d' gid not change: cfg num up to %d", shard, info.CurrentCfg.Num)
					kv.shardState[shard].State = Working
				} else {
					go kv.pullShard(*info.PrevCfg, shard, prevGID)
				}
			} else {
				if prevGID != kv.gid {
					kv.DPrintf("shard %d' gid not change: cfg num up to %d", shard, info.CurrentCfg.Num)
					kv.shardState[shard].State = Working
				} else {
					// 等待被拉取分片
				}
			}
			if prevGID == kv.gid && nowGID == 0 {
				panic(fmt.Sprintf("[panic] nowGID is 0"))
			}
		}
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
