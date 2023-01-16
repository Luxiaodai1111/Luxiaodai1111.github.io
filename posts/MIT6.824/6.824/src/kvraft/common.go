package kvraft

import "time"

const (
	OK             = "OK"
	ErrNoKey       = "ErrNoKey"
	ErrWrongLeader = "ErrWrongLeader"
	ErrTimeout     = "ErrTimeout"

	NotLeader = "NotLeader"

	OpPut    = "Put"
	OpAppend = "Append"
	OpGet    = "Get"

	ExecuteTimeout = time.Second
)

type Err string

// Put or Append
type PutAppendArgs struct {
	Key         string
	Value       string
	Op          string // "Put" or "Append"
	ClientId    int64  // 客户端标识
	SequenceNum int64  // 请求序号
}

type PutAppendReply struct {
	Err Err
}

type GetArgs struct {
	Key         string
	ClientId    int64 // 客户端标识
	SequenceNum int64 // 请求序号
}

type GetReply struct {
	Err   Err
	Value string
}
