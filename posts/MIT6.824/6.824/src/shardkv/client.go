package shardkv

//
// client code to talk to a sharded key/value service.
//
// the client first talks to the shardctrler to find out
// the assignment of shards (keys) to groups, and then
// talks to the group that holds the key's shard.
//

import (
	"6.824/labrpc"
	"fmt"
	"log"
	"sync/atomic"
)
import "crypto/rand"
import "math/big"
import "6.824/shardctrler"
import "time"

func (ck *Clerk) DPrintf(format string, a ...interface{}) {
	if Debug {
		log.Printf(fmt.Sprintf("[ShardKV Clerk %d]:%s", ck.clientId, format), a...)
	}
	return
}

//
// which shard is a key in?
// please use this function,
// and please do not change it.
//
func key2shard(key string) int {
	shard := 0
	if len(key) > 0 {
		shard = int(key[0])
	}
	shard %= shardctrler.NShards
	return shard
}

func nrand() int64 {
	max := big.NewInt(int64(1) << 62)
	bigx, _ := rand.Int(rand.Reader, max)
	x := bigx.Int64()
	return x
}

type Clerk struct {
	sm             *shardctrler.Clerk
	config         shardctrler.Config
	make_end       func(string) *labrpc.ClientEnd
	clientId       int64 // 客户端标识
	maxSequenceNum int64 // 当前使用的最大命令序号
}

//
// the tester calls MakeClerk.
//
// ctrlers[] is needed to call shardctrler.MakeClerk().
//
// make_end(servername) turns a server name from a
// Config.Groups[gid][i] into a labrpc.ClientEnd on which you can
// send RPCs.
//
func MakeClerk(ctrlers []*labrpc.ClientEnd, make_end func(string) *labrpc.ClientEnd) *Clerk {
	ck := new(Clerk)
	ck.sm = shardctrler.MakeClerk(ctrlers)
	ck.make_end = make_end
	ck.clientId = nrand()
	ck.maxSequenceNum = 0
	return ck
}

//
// fetch the current value for a key.
// returns "" if the key does not exist.
// keeps trying forever in the face of all other errors.
// You will have to modify this function.
//
func (ck *Clerk) Get(key string) string {
	args := CommandArgs{
		Key:         key,
		Op:          OpGet,
		ClientId:    ck.clientId,
		SequenceNum: atomic.AddInt64(&ck.maxSequenceNum, 1),
	}

	ck.DPrintf("=== request %d get key: %s ===", args.SequenceNum, key)

	shard := key2shard(key)
	for {
		gid := ck.config.Shards[shard]
		if servers, ok := ck.config.Groups[gid]; ok {
			// try each server for the shard.
			for si := 0; si < len(servers); si++ {
				srv := ck.make_end(servers[si])
				var reply CommonReply
				success := srv.Call("ShardKV.Get", &args, &reply)
				if success && (reply.Err == OK || reply.Err == ErrNoKey) {
					ck.DPrintf("=== request %d get key: %s success ===", args.SequenceNum, key)
					return reply.Value
				}
				if success && (reply.Err == ErrWrongGroup) {
					break
				}
				// ... not ok, or ErrWrongLeader
			}
		}
		time.Sleep(100 * time.Millisecond)
		// ask controler for the latest configuration.
		ck.config = ck.sm.Query(-1)
	}

	return ""
}

//
// shared by Put and Append.
// You will have to modify this function.
//
func (ck *Clerk) PutAppend(key string, value string, op string) {
	args := CommandArgs{
		Key:         key,
		Value:       value,
		Op:          op,
		ClientId:    ck.clientId,
		SequenceNum: atomic.AddInt64(&ck.maxSequenceNum, 1),
	}

	ck.DPrintf("=== request %d %s <%s>:<%s> ===", args.SequenceNum, op, key, value)

	shard := key2shard(key)
	for {
		gid := ck.config.Shards[shard]
		if servers, ok := ck.config.Groups[gid]; ok {
			for si := 0; si < len(servers); si++ {
				srv := ck.make_end(servers[si])
				var reply CommonReply
				success := srv.Call("ShardKV.PutAppend", &args, &reply)
				if success && reply.Err == OK {
					ck.DPrintf("=== request %d %s <%s>:<%s> success ===", args.SequenceNum, op, key, value)
					return
				}
				if success && reply.Err == ErrWrongGroup {
					break
				}
				// ... not ok, or ErrWrongLeader
			}
		}
		time.Sleep(100 * time.Millisecond)
		// ask controler for the latest configuration.
		ck.config = ck.sm.Query(-1)
	}
}

func (ck *Clerk) Put(key string, value string) {
	ck.PutAppend(key, value, "Put")
}

func (ck *Clerk) Append(key string, value string) {
	ck.PutAppend(key, value, "Append")
}
