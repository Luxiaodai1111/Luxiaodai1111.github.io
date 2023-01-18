package kvraft

import "time"

const (
	OK             = "OK"
	ErrNoKey       = "ErrNoKey"
	ErrWrongLeader = "ErrWrongLeader"
	ErrTimeout     = "ErrTimeout"
	ErrRetry       = "ErrRetry"

	ApplySnap = "ApplySnap"

	OpPut    = "Put"
	OpAppend = "Append"
	OpGet    = "Get"

	ExecuteTimeout = time.Millisecond * 500
)

type Err string

type CommonArgs struct {
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
