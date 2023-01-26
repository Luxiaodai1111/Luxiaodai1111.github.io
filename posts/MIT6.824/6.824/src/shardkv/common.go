package shardkv

import "time"

//
// Sharded key/value server.
// Lots of replica groups, each running Raft.
// Shardctrler decides which group serves each shard.
// Shardctrler may change shard assignment from time to time.
//
// You will have to modify these definitions.
//

const (
	OK             = "OK"
	ErrNoKey       = "ErrNoKey"
	ErrWrongGroup  = "ErrWrongGroup"
	ErrWrongLeader = "ErrWrongLeader"
	ErrTimeout     = "ErrTimeout"
	ErrRetry       = "ErrRetry"

	ApplySnap = "ApplySnap"

	OpPut    = "Put"
	OpAppend = "Append"
	OpGet    = "Get"

	ExecuteTimeout = time.Millisecond * 500

	// 分片状态
	Working         = "Working"
	PrepareReConfig = "PrepareReConfig"
	ReConfining     = "ReConfining"

	// 日志类型
	CommandLog  = "CommandLog"
	ReConfigLog = "ReConfigLog"

	// ReConfigArgs op
	Push = "Push"
	Pull = "Pull"
)

type Err string

type PushShardArgs struct {
	Data        map[string]string
	Shard       int // 更改的分片
	ShardCfgNum int // 变更前的配置序号
}

type ReConfigArgs struct {
	Server string // GID + kv.me, just debug
	Shard  int    // 更改的分片
	Num    int    // 变更前的配置序号
}

type CommandArgs struct {
	Key         string
	Value       string
	Op          string // "Put" or "Append" or "Get"
	ClientId    int64  // 客户端标识
	SequenceNum int64  // 请求序号
}

type CommonReply struct {
	Err   Err
	Value string
}
