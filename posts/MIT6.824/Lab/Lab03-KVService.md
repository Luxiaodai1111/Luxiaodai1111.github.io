# 实验介绍

在本实验中，你将使用 Lab 2 中的 Raft 库来构建一个可容错的键值存储服务。客户端可以向服务发送三种不同的 RPC：

-   Put(key, value)：替换数据库中某个特定键的值
-   Append(key，arg)：将 arg 附加到键的值上
-   Get(key)：获取键的当前值

键和值都是字符串。对于一个不存在的键，Get 应该返回一个空字符串。对一个不存在的键的 Append 应该像 Put 一样操作。

每个客户端通过一个带有 Put/Append/Get 方法的 Clerk 与服务器通信，Clerk 负责管理与服务器的 RPC 交互。

服务要求是线性一致的。例如，如果一个客户端从服务中获得了一个更新请求的成功响应，那么随后从其他客户端发起的读取就能保证看到该更新的效果。

本实验有两部分。在 A 部分，您将使用您的 Raft 实现实现一个键/值服务，但不使用快照。在 B 部分中，您将使用 Lab 2D 中的快照实现，这将使 Raft 能够丢弃旧的日志条目。





---

# Getting Started

我们在 src/kvraft 中为你提供了框架代码和测试。你将需要修改 kvraft/client.go，kvraft/server.go，也许还有 kvraft/common.go。

为了启动和运行，执行以下命令。

```bash
$ cd ~/6.824
$ git pull
...
$ cd src/kvraft
$ go test -race
...
$
```





---

# Part A: Key/value service without snapshots

你的每个键/值服务器（"kvservers"）将有一个相关的 Raft peer。Clerks 将 Put、Append 和 Get RPCs 发送到 Raft Leader 的 kvserver。kvserver 代码将 Put/Append/Get 操作提交给 Raft，这样 Raft 日志就持有一连串的 Put/Append/Get 操作。所有的 kvserver 按顺序执行 Raft 日志中的操作，将这些操作应用到他们的键/值数据库中；目的是让服务器保持相同的键/值数据库副本。

Clerk 有时不知道哪个 kvserver 是 Raft 的 leader。如果 Clerk 向错误的 kvserver 发送 RPC，或者无法到达该 kvserver，Clerk 应该重新尝试不同的 kvserver。如果键/值服务将操作提交给它的 Raft 日志（并因此将操作应用于键/值状态机），Leader 通过响应它的 RPC 将结果报告给 Clerk。如果操作未能提交，服务器会报告一个错误，Clerk 会用另一个服务器重试。

你的 kvservers 不应该直接通信，它们应该只通过 Raft 进行相互交流。

实验任务 1：首先实现不丢失消息以及没有失败的服务器场景下的解决方案。

你需要为 client.go 中的 Clerk Put/Append/Get 方法添加 RPC 发送代码，并在 server.go 中实现 PutAppend() 和 Get() RPC 处理程序。这些处理程序应使用 Start() 在 Raft 日志中输入一个 Op；你应在 server.go 中填写 Op 结构体定义使其描述一个 Put/Append/Get 操作。每个服务器应该在 Raft 提交 Op 命令时执行这些命令，也就是说，当它们出现在 applyCh 上时。RPC 处理程序应该注意到 Raft 何时提交其 Op，然后回复 RPC。

>[!TIP]
>
>- After calling `Start()`, your kvservers will need to wait for Raft to complete agreement. Commands that have been agreed upon arrive on the `applyCh`. Your code will need to keep reading `applyCh` while `PutAppend()` and `Get()` handlers submit commands to the Raft log using `Start()`. Beware of deadlock between the kvserver and its Raft library.
>- You are allowed to add fields to the Raft `ApplyMsg`, and to add fields to Raft RPCs such as `AppendEntries`, however this should not be necessary for most implementations.
>- A kvserver should not complete a `Get()` RPC if it is not part of a majority (so that it does not serve stale data). A simple solution is to enter every `Get()` (as well as each `Put()` and `Append()`) in the Raft log. You don't have to implement the optimization for read-only operations that is described in Section 8.
>- It's best to add locking from the start because the need to avoid deadlocks sometimes affects overall code design. Check that your code is race-free using `go test -race`.

现在你应该修改你的解决方案，以便在面对网络和服务器故障时能够继续工作。你将面临的一个问题是，Clerk 可能要多次发送 RPC，直到它找到一个积极回复的 kvserver。如果一个 Leader 在向 Raft 日志提交条目后发生故障，Clerk 可能不会收到回复，因此可能会向另一个 Leader 重新发送请求。对 Clerk.Put() 或 Clerk.Append() 的每次调用应该只导致一次执行，所以你必须确保重新发送不会导致服务器执行两次请求。

实验任务 2：添加代码来处理失败，以及处理重复的请求。

>[!TIP]
>
>- Your solution needs to handle a leader that has called Start() for a Clerk's RPC, but loses its leadership before the request is committed to the log. In this case you should arrange for the Clerk to re-send the request to other servers until it finds the new leader. One way to do this is for the server to detect that it has lost leadership, by noticing that a different request has appeared at the index returned by Start(), or that Raft's term has changed. If the ex-leader is partitioned by itself, it won't know about new leaders; but any client in the same partition won't be able to talk to a new leader either, so it's OK in this case for the server and client to wait indefinitely until the partition heals.
>- You will probably have to modify your Clerk to remember which server turned out to be the leader for the last RPC, and send the next RPC to that server first. This will avoid wasting time searching for the leader on every RPC, which may help you pass some of the tests quickly enough.
>- You will need to uniquely identify client operations to ensure that the key/value service executes each one just once.
>- Your scheme for duplicate detection should free server memory quickly, for example by having each RPC imply that the client has seen the reply for its previous RPC. It's OK to assume that a client will make only one call into a Clerk at a time.

你的代码应该通过 go test -run 3A -race 测试。

```bash
$ go test -run 3A -race
Test: one client (3A) ...
  ... Passed --  15.5  5  4576  903
Test: ops complete fast enough (3A) ...
  ... Passed --  15.7  3  3022    0
Test: many clients (3A) ...
  ... Passed --  15.9  5  5884 1160
Test: unreliable net, many clients (3A) ...
  ... Passed --  19.2  5  3083  441
Test: concurrent append to same key, unreliable (3A) ...
  ... Passed --   2.5  3   218   52
Test: progress in majority (3A) ...
  ... Passed --   1.7  5   103    2
Test: no progress in minority (3A) ...
  ... Passed --   1.0  5   102    3
Test: completion after heal (3A) ...
  ... Passed --   1.2  5    70    3
Test: partitions, one client (3A) ...
  ... Passed --  23.8  5  4501  765
Test: partitions, many clients (3A) ...
  ... Passed --  23.5  5  5692  974
Test: restarts, one client (3A) ...
  ... Passed --  22.2  5  4721  908
Test: restarts, many clients (3A) ...
  ... Passed --  22.5  5  5490 1033
Test: unreliable net, restarts, many clients (3A) ...
  ... Passed --  26.5  5  3532  474
Test: restarts, partitions, many clients (3A) ...
  ... Passed --  29.7  5  6122 1060
Test: unreliable net, restarts, partitions, many clients (3A) ...
  ... Passed --  32.9  5  2967  317
Test: unreliable net, restarts, partitions, random keys, many clients (3A) ...
  ... Passed --  35.0  7  8249  746
PASS
ok  	6.824/kvraft	290.184s
```

每个 Passed 后面的数字是实时时间（秒）、peer 数量、发送的 RPC 数量（包括客户端 RPC）和执行的键/值操作数量（Clark Get/Put/Append 调用）。



# Part B: Key/value service with snapshots

目前的情况是，你的键/值服务器不调用你的 Raft 库的 Snapshot() 方法，所以重新启动的服务器必须重放完整的持久 Raft 日志以恢复它的状态。现在，您将使用 lab 2D 的 Snapshot() 来修改 kvserver，使其与 Raft 协作以节省日志空间，并减少重启时间。

测试人员将 maxraftstate 传递给 StartKVServer()。maxraftstate 以字节表示持久 Raft 状态的最大允许大小（包括日志，但不包括快照）。您应该将 maxraftstate 与 persister.RaftStateSize() 进行比较。每当您的键/值服务器检测到 Raft 状态大小接近这个阈值时，它应该通过调用 Raft 的快照来保存快照。如果 maxraftstate 为 -1，则不必拍摄快照。maxraftstate 应用于 Raft 传递给 persister.SaveRaftState()

实验任务：修改您的 kvserver，使其能够检测持续的 Raft 状态何时变得过大，然后将快照传递给 Raft。当 kvserver 服务器重新启动时，它应该从 persister 中读取快照，并从快照中恢复其状态。

>[!TIP]
>
>- Think about when a kvserver should snapshot its state and what should be included in the snapshot. Raft stores each snapshot in the persister object using `SaveStateAndSnapshot()`, along with corresponding Raft state. You can read the latest stored snapshot using `ReadSnapshot()`.
>- Your kvserver must be able to detect duplicated operations in the log across checkpoints, so any state you are using to detect them must be included in the snapshots.
>- Capitalize all fields of structures stored in the snapshot.
>- You may have bugs in your Raft library that this lab exposes. If you make changes to your Raft implementation make sure it continues to pass all of the Lab 2 tests.
>- A reasonable amount of time to take for the Lab 3 tests is 400 seconds of real time and 700 seconds of CPU time. Further, `go test -run TestSnapshotSize` should take less than 20 seconds of real time.

您的代码应该通过 3B 测试以及 3A 测试（并且您的 Raft 必须继续通过 lab2 测试）

```bash
$ go test -run 3B -race
Test: InstallSnapshot RPC (3B) ...
  ... Passed --   4.0  3   289   63
Test: snapshot size is reasonable (3B) ...
  ... Passed --   2.6  3  2418  800
Test: ops complete fast enough (3B) ...
  ... Passed --   3.2  3  3025    0
Test: restarts, snapshots, one client (3B) ...
  ... Passed --  21.9  5 29266 5820
Test: restarts, snapshots, many clients (3B) ...
  ... Passed --  21.5  5 33115 6420
Test: unreliable net, snapshots, many clients (3B) ...
  ... Passed --  17.4  5  3233  482
Test: unreliable net, restarts, snapshots, many clients (3B) ...
  ... Passed --  22.7  5  3337  471
Test: unreliable net, restarts, partitions, snapshots, many clients (3B) ...
  ... Passed --  30.4  5  2725  274
Test: unreliable net, restarts, partitions, snapshots, random keys, many clients (3B) ...
  ... Passed --  37.7  7  8378  681
PASS
ok  	6.824/kvraft	161.538s
```







---

# 设计思路

## 论文消息体设计

raft 的博士毕业论文里对 client 的设计讲的会比较详细一些，首先它像之前一样列出了实现的 RPC：

![](Lab03-KVService/6_1.png)

客户端调用 ClientRequest RPC 来修改状态；他们调用 ClientQuery RPC 来查询状态。新的客户端使用 RegisterClient RPC 接收其客户端标识符。在该图中，非领导者的服务器将客户端重定向到领导者。

**ClientRequest RPC**

| 参数        | 解释                         |
| ----------- | ---------------------------- |
| clientId    | 客户端标识                   |
| sequenceNum | 消除重复的请求               |
| command     | 状态机的请求，可能会影响状态 |

| 返回值     | 解释                               |
| ---------- | ---------------------------------- |
| status     | 如果状态机应用了命令返回 OK        |
| response   | 状态机的输出，如果成功的话         |
| leaderHint | 如果知道的话返回最近的 leader 地址 |



**ClientQuery RPC**

| 参数  | 解释     |
| ----- | -------- |
| query | 查询请求 |

| 返回值     | 解释                               |
| ---------- | ---------------------------------- |
| status     | 如果状态机处理了请求返回 OK        |
| response   | 状态机的输出，如果成功的话         |
| leaderHint | 如果知道的话返回最近的 leader 地址 |



**RegisterClient RPC**

| 参数 | 解释 |
| ---- | ---- |
| 无   |      |

| 返回值     | 解释                               |
| ---------- | ---------------------------------- |
| status     | 如果状态机处理了请求返回 OK        |
| clientId   | 客户端标识                         |
| leaderHint | 如果知道的话返回最近的 leader 地址 |





## 结构体设计

本次实验和论文里的框架还有点不太一样，可以使用通用的结构体来表示读写请求，这样处理比较方便

```go
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
```

对于客户端，需要知道 leader 是谁，一个唯一的标识，以及命令的序号生成

```go
type Clerk struct {
	servers        []*labrpc.ClientEnd
	leader         int   // leader 的地址
	clientId       int64 // 客户端标识
	maxSequenceNum int64 // 当前使用的最大命令序号
}
```

对于请求的框架，其实读写差不多，就是给命令编号，如果失败了就重新选一个服务端发送，这里简单地切换服务器来尝试，实际上由于 raft 每个服务器都知道 leader 是谁，可以优化成更优雅的方式。

```go
func (ck *Clerk) Get(key string) string {
	ck.DPrintf("=== request get key: %s ===", key)

	args := &CommonArgs{
		Key:         key,
		Op:          OpGet,
		ClientId:    ck.clientId,
		SequenceNum: ck.maxSequenceNum,
	}
	atomic.AddInt64(&ck.maxSequenceNum, 1)

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
```

服务器端稍微复杂一点，首先我们看结构体定义，我们使用 map 来当内存数据库，也就是状态机，另外由于请求需要 raft 复制到半数节点，所以请求需要通道来通知可以返回，另外请求被提交不代表能正确回复客户端，假如请求迟迟没有回复，客户端重试将请求发给了新的 leader，那么日志就会有两个同样的命令，但是客户端期待的是只执行一遍，所以我们需要在 apply 的时候根据客户端标识和请求序号对请求去重

```go
type KVServer struct {
	mu      sync.RWMutex
	me      int
	rf      *raft.Raft
	applyCh chan raft.ApplyMsg
	dead    int32 // set by Kill()

	maxraftstate   int // snapshot if log grows this big
	lastApplyIndex int

	db            map[string]string            // 内存数据库
	notifyChans   map[int]chan *CommonReply    // 监听请求 apply
	dupReqHistory map[int64]map[int64]struct{} // 记录已经执行的修改命令，防止重复执行
}
```

首先收到请求后，建立通道，然后等待 raft apply，对于返回的请求，如果不是 leader 了就不用返回了，如果任期和 Start 时不一致了，此时也不返回，因为 index 可能会错乱，从而错乱回复。另外设置了超时，比如分区情况下，可能迟迟不会 apply，此时不能阻塞客户端请求。

```go
func (kv *KVServer) Command(args *CommonArgs, reply *CommonReply) {
	// 修改请求重复
	if args.Op != OpGet && kv.isDuplicateRequest(args.ClientId, args.SequenceNum) {
		kv.DPrintf("found duplicate request: %+v, reply history response", args)
		reply.Err = OK
		return
	}

	/*
	 * 要使用 term 和 index 来代表一条日志
	 * 对于 apply 超时，我们也要关闭通道，因为重新选主之后，这个通道再也用不到了
	 */
	index, term, isLeader := kv.rf.Start(*args)
	if !isLeader {
		reply.Err = ErrWrongLeader
		return
	}

	kv.Lock("getNotifyChan")
	if _, ok := kv.notifyChans[index]; !ok {
		kv.notifyChans[index] = make(chan *CommonReply)
	}
	kv.Unlock("getNotifyChan")

	select {
	case result := <-kv.notifyChans[index]:
		currentTerm, isleader := kv.rf.GetState()
		if !isleader || currentTerm != term {
			reply.Err = ErrWrongLeader
			kv.DPrintf("reply now is not leader")
			return
		}
		kv.DPrintf("reply index: %d", index)

		if reply.Err == ApplySnap {
			if args.Op != OpGet {
				reply.Err = OK
			} else {
				reply.Err = ErrRetry
			}
		} else {
			reply.Err, reply.Value = result.Err, result.Value
		}
	case <-time.After(ExecuteTimeout):
		kv.DPrintf("wait apply log %d time out", index)
		reply.Err = ErrTimeout
	}

	kv.Lock("Command")
	defer kv.Unlock("Command")
	close(kv.notifyChans[index])
	delete(kv.notifyChans, index)
}
```

重头戏其实是 apply 的处理，apply 可能是日志，也可能是快照，不过我之前的 raft 实现保证了他们是按日志索引有序 apply 的。

lastApplyIndex 只是我防御性编程，已经 apply 过就不要重复 apply 了，但实际上这些会记录在重复请求哈希表里，如果检查到修改请求重复了，一定不能再 apply 一次的。

更改状态机没什么好说的，更改完成后就可以通知请求回复了。注意的是，此时可能已经切换 leader 了，这个时候通道可能还存在，甚至本节点的日志会被别人覆盖，从而导致 index 错乱，这个时候如果回复，那么会有什么结果就不得而知了，所以通道那头需要判断当前是否为主，任期是否改变等等。

最后就是检查是否需要大快照了，需要注意的是，重复请求哈希表也需要快照，因为 apply 快照实际上相当于跳过了一些日志 apply，如果没有同步更新重复哈希表，那么就可能造成请求重复执行（比如客户端重新发送了同样的请求到这个节点，然后添加到了日志并执行）。

对于跳过的那些请求，如果是修改请求可以直接返回 OK，如果是读请求，直接让客户端重试就好了。

```go
func (kv *KVServer) handleApply() {
	for kv.killed() == false {
		select {
		case applyLog := <-kv.applyCh:
			if applyLog.CommandValid {
				op, ok := applyLog.Command.(Op)
				if !ok {
					kv.DPrintf("[panic] recieved apply log's command error")
					kv.Kill()
					return
				}

				reply := &CommonReply{
					Err: OK,
				}

				if applyLog.CommandIndex <= kv.lastApplyIndex {
					// 比如 raft 重启了，就要重新 apply
					kv.DPrintf("***** command index %d is older than lastApplyIndex %d *****",
						applyLog.SnapshotIndex, kv.lastApplyIndex)
					continue
				}
				kv.lastApplyIndex = applyLog.CommandIndex

				kv.DPrintf("recieve apply log: %d, op info: %+v", applyLog.CommandIndex, op)
				// 防止重复应用同一条修改命令
				if op.Op != OpGet && kv.isDuplicateRequest(op.ClientId, op.SequenceNum) {
					kv.DPrintf("found duplicate request: %+v", op)
					continue
				}

				// 更新状态机
				value, ok := kv.db[op.Key]
				if op.Op == OpGet {
					if ok {
						reply.Value = value
						kv.DPrintf("get <%s>:<%s>", op.Key, value)
					} else {
						reply.Err = ErrNoKey
					}
				} else {
					if op.Op == OpAppend && ok {
						kv.db[op.Key] += op.Value
					} else {
						kv.db[op.Key] = op.Value
					}
					kv.DPrintf("update <%s>:<%s>", op.Key, kv.db[op.Key])
				}

				kv.Lock("replyCommand")
				if op.Op != OpGet {
					kv.updateDupReqHistory(op.ClientId, op.SequenceNum)
				}
				/*
				 * 只要有通道存在，说明可能是当前 leader，也可能曾经作为 leader 接收过请求
				 * 通道可能处于等待消息状态，或者正在返回错误等待销毁，所以不管怎么样，都往通道里返回消息
				 * 如果已经销毁，说明已经返回了等待超时错误
				 */
				if _, ok := kv.notifyChans[applyLog.CommandIndex]; ok {
					select {
					case kv.notifyChans[applyLog.CommandIndex] <- reply:
					default:
					}
				}
				kv.Unlock("replyCommand")

				// 检测是否需要执行快照
				if kv.rf.NeedSnapshot(kv.maxraftstate) {
					kv.DPrintf("======== snapshot %d ========", applyLog.CommandIndex)
					w := new(bytes.Buffer)
					e := labgob.NewEncoder(w)
					kv.Lock("snap")
					dupReqHistorySnap := kv.makeDupReqHistorySnap()
					if e.Encode(kv.db) != nil || e.Encode(dupReqHistorySnap) != nil {
						kv.DPrintf("[panic] encode snap error")
						kv.Unlock("snap")
						kv.Kill()
						return
					}
					kv.Unlock("snap")
					data := w.Bytes()
					kv.DPrintf("snap size: %d", len(data))
					kv.rf.Snapshot(applyLog.CommandIndex, data)
				}
			} else if applyLog.SnapshotValid {
				kv.DPrintf("======== recieve apply snap: %d ========", applyLog.SnapshotIndex)
				if applyLog.SnapshotIndex <= kv.lastApplyIndex {
					kv.DPrintf("***** snap index %d is older than lastApplyIndex %d *****",
						applyLog.SnapshotIndex, kv.lastApplyIndex)
					continue
				}

				r := bytes.NewBuffer(applyLog.Snapshot)
				d := labgob.NewDecoder(r)
				kv.Lock("applySnap")
				kv.db = make(map[string]string)
				var dupReqHistorySnap DupReqHistorySnap
				if d.Decode(&kv.db) != nil || d.Decode(&dupReqHistorySnap) != nil {
					kv.DPrintf("[panic] decode snap error")
					kv.Unlock("applySnap")
					kv.Kill()
					return
				}
				kv.restoreDupReqHistorySnap(dupReqHistorySnap)
				kv.Unlock("applySnap")

				// lastApplyIndex 到快照之间的修改请求一定会包含在查重哈希表里
				// 对于读只需要让客户端重新尝试即可
				kv.Lock("replyCommand")
				reply := &CommonReply{
					Err: ApplySnap,
				}
				for idx := kv.lastApplyIndex + 1; idx <= applyLog.SnapshotIndex; idx++ {
					if _, ok := kv.notifyChans[idx]; ok {
						select {
						case kv.notifyChans[idx] <- reply:
						default:
						}
					}
				}
				kv.Unlock("replyCommand")
				kv.lastApplyIndex = applyLog.SnapshotIndex
			} else {
				kv.DPrintf(fmt.Sprintf("[panic] unexpected applyLog %v", applyLog))
				kv.Kill()
				return
			}
		default:
			continue
		}
	}
}
```



## raft 速度问题

之前实现的 raft 虽然正确性没问题，但是 apply 速度很慢，发现其中一个原因是日志提交慢了，后面调整复制日志成功后更新 matchIndex 时也更新 commitIndex ，这样更新会比较及时。为什么心跳检查提交还保留呢？假设某个 leader 把日志复制给了大多数就故障了，然后期间没有提交日志，它又竞选成功，那么此时就没有方法去触发 commitIndex 的更新了。

另外 apply 索引排序也优化了下，不过测试下来好像没啥大影响；另外这里其实可以不用排序，交给上层去处理也是没问题的，我只是觉得排序了对上层逻辑更清晰简单一些。

3A 的速度测试是一个请求一个请求的发，收到回复再发下一个，对于之前的设计，满足阈值的条目数会立刻发送，否则等待心跳发送日志，之前心跳设置的 100 ms，那么相当于一秒就复制 10 条日志，满足不了测试要求，因此调整阈值为 1 表示收到请求立即复制来满足测试要求。



## lab02 的 bug

在做 lab03 时，发现 lab02 的快照实现有点问题，主要是在 recover 时，lastApplied 设置为了当前第一条日志的索引（一定已提交），实际上日志已提交不代表已应用，所以 lastApplied 应该从 0 开始，如果有快照，还要重新 apply 快照。

这也导致我做 3B 实验时，快照恢复的测试一开始不通过。



## 快照大小问题

由于记录快照不仅要记录数据库，还需要记录重复请求的哈希索引，而哈希索引是随着请求越来越多的，怎么办呢？

一是读请求不用过滤重复，因为它不影响状态；

二是在封装快照时，可以把序列号压缩一下，我采用的方法是先排序，然后只记录最小值和他们之间的差值，并转成一条字符串保存

不知道我方向有没有走偏……也许实验是让我思考一种正确保存快照的办法，而我在这搞些邪门歪道哈哈哈

```go
func (kv *KVServer) makeDupReqHistorySnap() DupReqHistorySnap {
	snap := make(DupReqHistorySnap, 0)
	for clientId, info := range kv.dupReqHistory {
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

func (kv *KVServer) restoreDupReqHistorySnap(snap DupReqHistorySnap) {
	kv.dupReqHistory = make(map[int64]map[int64]struct{})
	for clientId, info := range snap {
		if _, ok := kv.dupReqHistory[clientId]; !ok {
			kv.dupReqHistory[clientId] = make(map[int64]struct{})
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
			kv.dupReqHistory[clientId][prev] = struct{}{}
		}
	}
}
```





