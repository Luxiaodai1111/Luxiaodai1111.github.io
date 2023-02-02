package shardkv

import (
	"6.824/shardctrler"
	"time"
)

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
	Working           = "Working"
	PrepareReConfig   = "PrepareReConfig"
	PreparePull       = "PreparePull"
	Pulling           = "Pulling"
	WaitingToBePulled = "WaitingToBePulled"

	// 日志类型
	NoOpLog        = "NoOpLog"
	CommandLog     = "CommandLog"
	ReConfigLog    = "ReConfigLog"
	PullShardLog   = "PullShardLog"
	UpdateShardLog = "UpdateShardLog"
	DeleteShardLog = "DeleteShardLog"
)

type Err string

type PullShardArgs PullShardLogArgs

type DeleteShardArgs struct {
	Shard       int // 更改的分片
	ShardCfgNum int
}

type PullShardReply struct {
	Err  Err
	Data map[string]string
}

type UpdateShardLogArgs struct {
	Shard       int // 更改的分片
	ShardCfgNum int
	Data        map[string]string
}

type PullShardLogArgs struct {
	Shard     int // 更改的分片
	PrevCfg   shardctrler.Config
	UpdateCfg shardctrler.Config
}

type ReConfigLogArgs struct {
	PrevCfg   shardctrler.Config
	UpdateCfg shardctrler.Config
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
