package shardctrler

//
// Shardctrler clerk.
//

import (
	"6.824/labrpc"
	"fmt"
	"log"
	"sync/atomic"
)
import "time"
import "crypto/rand"
import "math/big"

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

func (ck *Clerk) Command(args *CommonArgs) Config {
	args.ClientId = ck.clientId
	args.SequenceNum = atomic.AddInt64(&ck.maxSequenceNum, 1)

	for {
		// try each known server.
		for _, srv := range ck.servers {
			var reply CommonReply
			ok := false
			if args.Op == OpJoin {
				ok = srv.Call("ShardCtrler.Join", args, &reply)
			} else if args.Op == OpLeave {
				ok = srv.Call("ShardCtrler.Leave", args, &reply)
			} else if args.Op == OpMove {
				ok = srv.Call("ShardCtrler.Move", args, &reply)
			} else {
				ok = srv.Call("ShardCtrler.Query", args, &reply)
			}

			if ok && reply.Err == OK {
				if args.Op == OpQuery {
					return reply.Config
				}
				return Config{}
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (ck *Clerk) Query(num int) Config {
	args := &CommonArgs{
		Op:  OpQuery,
		Num: num,
	}

	return ck.Command(args)
}

func (ck *Clerk) Join(servers map[int][]string) {
	args := &CommonArgs{
		Op:      OpJoin,
		Servers: servers,
	}

	ck.Command(args)
}

func (ck *Clerk) Leave(gids []int) {
	args := &CommonArgs{
		Op:   OpLeave,
		GIDs: gids,
	}

	ck.Command(args)
}

func (ck *Clerk) Move(shard int, gid int) {
	args := &CommonArgs{
		Op:    OpMove,
		Shard: shard,
		GID:   gid,
	}

	ck.Command(args)
}
