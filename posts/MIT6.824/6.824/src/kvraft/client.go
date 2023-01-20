package kvraft

import (
	"6.824/labrpc"
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"sync/atomic"
)

func (ck *Clerk) DPrintf(format string, a ...interface{}) {
	if Debug {
		log.Printf(fmt.Sprintf("[Clerk]:%s", format), a...)
	}
	return
}

type Clerk struct {
	servers        []*labrpc.ClientEnd
	leader         int   // leader 的地址
	clientId       int64 // 客户端标识
	maxSequenceNum int64 // 当前使用的最大命令序号
}

func nrand() int64 {
	max := big.NewInt(int64(1) << 62)
	bigx, _ := rand.Int(rand.Reader, max)
	x := bigx.Int64()
	return x
}

func MakeClerk(servers []*labrpc.ClientEnd) *Clerk {
	ck := new(Clerk)
	ck.servers = servers
	ck.leader = 0
	ck.clientId = nrand()
	ck.maxSequenceNum = 0
	return ck
}

//
// fetch the current value for a key.
// returns "" if the key does not exist.
// keeps trying forever in the face of all other errors.
//
// you can send an RPC with code like this:
// ok := ck.servers[i].Call("KVServer.Get", &args, &reply)
//
// the types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. and reply must be passed as a pointer.
//
func (ck *Clerk) Get(key string) string {
	ck.DPrintf("=== request get key: %s ===", key)

	args := &CommonArgs{
		Key:         key,
		Op:          OpGet,
		ClientId:    ck.clientId,
		SequenceNum: atomic.AddInt64(&ck.maxSequenceNum, 1),
	}

	leader := ck.leader
	for {
		reply := new(CommonReply)
		ok := ck.servers[leader].Call("KVServer.Get", args, reply)
		if ok {
			if reply.Err == OK {
				ck.DPrintf("=== get <%s>:<%s> from leader %d success ===", key, reply.Value, leader)
				ck.leader = leader
				return reply.Value
			} else if reply.Err == ErrNoKey {
				ck.leader = leader
				ck.DPrintf("get <%s> from leader %d failed: %s", key, leader, reply.Err)
				return ""
			} else if reply.Err == ErrRetry {
				ck.DPrintf("get <%s> from leader %d failed: %s", key, leader, reply.Err)
				ck.DPrintf("retry get <%s> from leader %d", key, leader)
				continue
			}
		}
		leader = (leader + 1) % len(ck.servers)
		ck.DPrintf("retry get <%s> from leader %d", key, leader)
	}
}

//
// shared by Put and Append.
//
// you can send an RPC with code like this:
// ok := ck.servers[i].Call("KVServer.PutAppend", &args, &reply)
//
// the types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. and reply must be passed as a pointer.
//
func (ck *Clerk) PutAppend(key string, value string, op string) {
	ck.DPrintf("=== request %s <%s>:<%s> ===", op, key, value)

	args := &CommonArgs{
		Key:         key,
		Value:       value,
		Op:          op,
		ClientId:    ck.clientId,
		SequenceNum: atomic.AddInt64(&ck.maxSequenceNum, 1),
	}

	leader := ck.leader
	for {
		reply := new(CommonReply)
		ok := ck.servers[leader].Call("KVServer.PutAppend", args, reply)
		if ok {
			if reply.Err == OK {
				ck.DPrintf("=== %s <%s>:<%s> to leader %d success ===", op, key, value, leader)
				ck.leader = leader
				return
			}
			ck.DPrintf("%s <%s>:<%s> to leader %d failed: %s", op, key, value, leader, reply.Err)
		}
		leader = (leader + 1) % len(ck.servers)
		ck.DPrintf("retry %s <%s>:<%s> to leader %d", op, key, value, leader)
	}
}

func (ck *Clerk) Put(key string, value string) {
	ck.PutAppend(key, value, "Put")
}

func (ck *Clerk) Append(key string, value string) {
	ck.PutAppend(key, value, "Append")
}
