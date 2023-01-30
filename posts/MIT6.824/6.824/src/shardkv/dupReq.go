package shardkv

import (
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

func (kv *ShardKV) updateDupLog(logType string, k1, k2 int64) {
	var dupMap map[int64]map[int64]struct{}
	if logType == CommandLog {
		dupMap = kv.dupCommand
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
	}
	if _, ok := dupMap[k1]; ok {
		if _, ok := dupMap[k1][k2]; ok {
			return true
		}
	}

	return false
}
