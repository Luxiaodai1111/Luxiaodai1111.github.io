# 实验介绍



在本实验中，您将构建一个键/值存储系统，该系统将数据分片（shard）或分区（partition）到一系列复制组（replica group）上。shard 是键/值对的子集；例如，所有以 “a” 开头的键可能是一个 shard，所有以 “b” 开头的键可能是另一个 shard 等等。分片的原因是性能，每个复制组只处理一些 shard 并且这些复制组可以并行操作，因此，总的系统吞吐量与复制组的数量成比例地增加。

您的分片键/值存储系统将有两个主要组件：

首先，一系列复制组。每个复制组负责一些分片。复制组使用 Raft 来复制分片。

第二个组件是分片控制器（shard controller）。控制器决定哪个复制组应该服务于哪些分片；这些信息被称为 configuration。configuration 会随着时间而变化。客户端询问 shard controller 以便找到 key 所属分片的的复制组，复制组询问控制器来知道他们服务哪些分片。shard controller 使用 Raft 的来实现容错服务 。

分片存储系统必须能够在复制组之间转移分片。一个原因是一些组可能比其他组负载更重。另一个原因是复制组可能会加入和离开系统：可能会添加新的复制组来增加容量，或者现有的复制组可能会因修复或淘汰而离线。

本实验的主要挑战将是处理 reconfiguration（将分片分配给组关系的变化）。

在单个复制组中，所有组成员必须就相对于客户端 Put/Append/Get 请求的重新配置发生时间达成一致。例如，Put 请求可能与重新配置同时发生，组中的所有副本必须就 Put 发生在重新配置之前还是之后达成一致。如果在之前，Put 应该生效，分片的新复制组将看到它的效果；如果之后，put 将不会生效，客户必须重新向新的复制组发起请求。推荐的方法是让每个复制组使用 Raft 不仅记录 put、append 和 get 的序列，还记录 reconfiguration 的序列。您将需要确保在任何时候，每个分片至多一个复制组提供服务。

重新配置还需要副本组之间的交互。例如，在配置 10 中，组 G1 可能负责分片 S1。在配置 11 中，组 G2 可能负责分片 S1。在从 10 到 11 的重新配置期间，G1 和 G2 必须使用 RPC 将分片 S1 的内容(键/值对)从 G1 移动到 G2。





---

# Getting Started

我们为您提供 src/shardctrler 和 src/shardkv 中的框架代码和测试。 要启动并运行，请执行以下命令:

```bash
$ cd ~/6.824
$ git pull
...
$ cd src/shardctrler
$ go test
--- FAIL: TestBasic (0.00s)
        test_test.go:11: wanted 1 groups, got 0
FAIL
exit status 1
FAIL    shardctrler     0.008s
$
```

完成后，您的实现应该通过了 src/shardctrler 目录中的所有测试，以及 src/shardkv 中的所有测试。



---

# Part A: The Shard controller

shardctrler 管理一系列编号的配置。每个配置都描述了一组复制组和复制组的 shard分配情况。每当这个分配需要改变的时候，shard 控制器就用新的分配创建一个新的配置。键/值客户端和服务端通过询问 shardctrler 获取当前(或过去)的配置。

您需实现 shardctrler/common.go 中描述的 RPC 接口，该接口由 Join、Leave、Move 和 Query RPC 组成。这些 RPC 旨在允许管理员（和测试）控制 shardctrler 添加、移除复制组以及在复制组之间移动分片。

管理员使用 Join RPC 添加新的复制组。它的参数是从唯一的非零副本组标识符（GID）到服务器名称列表的一组映射。shardctrler 应该通过创建一个包含新复制组的新配置来做出反应。新的配置应该在整个组中尽可能均匀地划分分片，并且应该移动尽可能少的分片。如果 GID 不是当前配置的一部分，shardctrler 应该允许重用它（即应该允许 GID 加入，然后离开，然后再次加入）

Leave RPC 的参数是以前加入的组的 GID 列表。shardctrler 应该创建一个不包括这些组的新配置，并将这些组的分片分配给其余的组。新的配置应该在组中尽可能平均地划分分片，并且应该移动尽可能少的分片。

Move RPC 的参数是一个分片号和一个 GID。shardctrler 应该创建一个新的配置，在这个配置中 shard 被分配给 GID 代表的组。Move 用来测试您的代码。Move 后的 Join 或 Leave 操作可能会导致 Move 撤销，因为 Join 或 Leave 会导致重平衡。

Query RPC 的参数是一个配置号。shardctrler 用具有该编号的配置进行回复。如果该数字为 -1 或大于已知的最大配置数，shardctrler 应该用最新的配置进行回复。查询 -1 的结果应该反映 shardctrler 在收到查询 -1 RPC 之前完成处理的每个 Join、Leave 或 Move RPC。

第一个配置应该编号为零。它不应该包含任何组，所有分片都应该分配给 GID 0（一个无效的 GID）。下一个配置（为响应 Join RPC 而创建的）应该编号为 1。分片通常比组多得多（即每个组将服务多个分片），以便负载可以以相当精细的粒度转移。



>[!TIP]
>
>- Start with a stripped-down copy of your kvraft server.
>- You should implement duplicate client request detection for RPCs to the shard controller. The shardctrler tests don't test this, but the shardkv tests will later use your shardctrler on an unreliable network; you may have trouble passing the shardkv tests if your shardctrler doesn't filter out duplicate RPCs.
>- The code in your state machine that performs the shard rebalancing needs to be deterministic. In Go, map iteration order is [not deterministic](https://blog.golang.org/maps#TOC_7.).
>- Go maps are references. If you assign one variable of type map to another, both variables refer to the same map. Thus if you want to create a new `Config` based on a previous one, you need to create a new map object (with `make()`) and copy the keys and values individually.
>- The Go race detector (go test -race) may help you find bugs.

实验任务：您的任务是使用 lab 2/3 中的 Raft 库在 shardctrler/ 目录中的 client.go 和 server.go 中实现上面指定的接口。您的 shardctrler 必须是容错的。请注意，在对 lab 4 评分时，我们将重新运行 lab 2和 lab 3 中的测试，因此请确保您没有将错误引入到 Raft 实现中，当您通过 shardctrler/ 中的所有测试时，您就完成了这项任务。

```bash
$ cd ~/6.824/src/shardctrler
$ go test -race
Test: Basic leave/join ...
  ... Passed
Test: Historical queries ...
  ... Passed
Test: Move ...
  ... Passed
Test: Concurrent leave/join ...
  ... Passed
Test: Minimal transfers after joins ...
  ... Passed
Test: Minimal transfers after leaves ...
  ... Passed
Test: Multi-group join/leave ...
  ... Passed
Test: Concurrent multi leave/join ...
  ... Passed
Test: Minimal transfers after multijoins ...
  ... Passed
Test: Minimal transfers after multileaves ...
  ... Passed
Test: Check Same config on servers ...
  ... Passed
PASS
ok  	6.824/shardctrler	5.863s
$
```





# Part B: Sharded Key/Value Server

现在您将构建 shardkv，一个分片的容错键/值存储系统。您将修改 shardkv/client.go、shardkv/common.go 和 shardkv/server.go。

每个 shardkv 服务器都作为复制组的一部分运行。每个复制组为一些分片提供服务。在 client.go 中使用 key2shard() 来查找一个 key 属于哪个 shard，多个复制组合作提供完整的服务。shardctrler 服务将 shard 分配给复制组；当这种分配改变时，复制组必须相互传递分片，同时确保客户端不会看到不一致的响应。

您的存储系统必须为使用其客户端接口的应用程序提供可线性化的接口，shardkv/client.go 中的 Append() 方法必须以相同的顺序影响所有副本，Get() 应该看到最近一次 Put/Append 写入的值，即使请求与配置更改同时发生。

只有当 shard 的 Raft 副本组中的大多数服务器都处于活动状态并且可以相互通信，并且可以与大多数 shardctrler 服务器通信时，才能正常服务请求。即使某些复制组中的少数服务器停止运行、暂时不可用或运行缓慢，您的实现也必须能够保证系统继续运行（满足请求并能够根据需要重新配置）

shardkv 服务器只是一个副本组的成员，给定复制组中的服务器集永远不会改变（raft 成员不会变更）。

我们为您提供了 client.go 代码框架，该代码将每个 RPC 发送到对应的复制组。如果复制组说它不负责这个 key，它就重试；在这种情况下，客户端代码向 shard 控制器请求最新的配置，然后重试。您须处理重复的客户端请求，就像在 kvraft lab 中一样。

>[!TIP]
>
>- Add code to `server.go` to periodically fetch the latest configuration from the shardctrler, and add code to reject client requests if the receiving group isn't responsible for the client's key's shard. You should still pass the first test.
>- Your server should respond with an `ErrWrongGroup` error to a client RPC with a key that the server isn't responsible for (i.e. for a key whose shard is not assigned to the server's group). Make sure your `Get`, `Put`, and `Append` handlers make this decision correctly in the face of a concurrent re-configuration.
>- Process re-configurations one at a time, in order.
>- If a test fails, check for gob errors (e.g. "gob: type not registered for interface ..."). Go doesn't consider gob errors to be fatal, although they are fatal for the lab.
>- You'll need to provide at-most-once semantics (duplicate detection) for client requests across shard movement.
>- Think about how the shardkv client and server should deal with `ErrWrongGroup`. Should the client change the sequence number if it receives `ErrWrongGroup`? Should the server update the client state if it returns `ErrWrongGroup` when executing a `Get`/`Put` request?
>- After a server has moved to a new configuration, it is acceptable for it to continue to store shards that it no longer owns (though this would be regrettable in a real system). This may help simplify your server implementation.
>- When group G1 needs a shard from G2 during a configuration change, does it matter at what point during its processing of log entries G2 sends the shard to G1?
>- You can send an entire map in an RPC request or reply, which may help keep the code for shard transfer simple.
>- If one of your RPC handlers includes in its reply a map (e.g. a key/value map) that's part of your server's state, you may get bugs due to races. The RPC system has to read the map in order to send it to the caller, but it isn't holding a lock that covers the map. Your server, however, may proceed to modify the same map while the RPC system is reading it. The solution is for the RPC handler to include a copy of the map in the reply.
>- If you put a map or a slice in a Raft log entry, and your key/value server subsequently sees the entry on the `applyCh` and saves a reference to the map/slice in your key/value server's state, you may have a race. Make a copy of the map/slice, and store the copy in your key/value server's state. The race is between your key/value server modifying the map/slice and Raft reading it while persisting its log.
>- During a configuration change, a pair of groups may need to move shards in both directions between them. If you see deadlock, this is a possible source.

实验任务：你的第一个任务是通过第一个 shardkv 测试用例。在这个测试中，只有一个分片，所以你的代码应该和你的 Lab 3 服务器非常相似。最大的修改将是让您的服务器检测配置并开始接受与它现在拥有的分片相匹配的请求。

注意：您的服务器不应该调用 shard 控制器的 Join() 处理程序。测试人员将在适当的时候调用 Join()。

既然您的解决方案已经适用于静态分片情况，那么是时候解决配置更改的问题了。您将需要让您的服务器监视配置更改，当检测到一个更改时，启动分片迁移。如果复制组不再负责一个分片，它必须立即停止为该分片中的键提供请求，并开始将该分片的数据迁移到接管所有权的复制组。如果一个复制组开始负责一个分片，它需要等待前一个所有者发送旧的分片数据，然后才能接受新的对该分片的请求。

实验任务：在配置更改期间实现分片迁移。确保复制组中的所有服务器在它们执行的操作序列中的同一点进行迁移，以便它们都接受或拒绝并发的客户端请求。在进行后面的测试之前，您应该专注于通过第二个测试（join then leave）。当您通过所有测试（不包括 TestDelete）时，此任务就完成了。

注意：

- 您的服务器需要定期轮询 shardctrler 以了解新的配置。测试预计您的代码大约每 100 毫秒轮询一次；快一点是可以的，但是慢了可能会出问题。
- 服务器需要互相发送 RPC，以便在配置更改期间传输分片。shardctrler 的配置结构包含服务器名，但是您需要一个 labrpc 以便发送 RPC。您应该使用 make_end() 函数将传递给 StartServer() 的服务器名称转换为 ClientEnd。可以参考 shardkv/client.go 有实现这些代码。



完成后，您的代码应该通过除挑战测试之外的所有 shardkv 测试:

```bash
$ cd ~/6.824/src/shardkv
$ go test -race
Test: static shards ...
  ... Passed
Test: join then leave ...
  ... Passed
Test: snapshots, join, and leave ...
  ... Passed
Test: servers miss configuration changes...
  ... Passed
Test: concurrent puts and configuration changes...
  ... Passed
Test: more concurrent puts and configuration changes...
  ... Passed
Test: concurrent configuration change and restart...
  ... Passed
Test: unreliable 1...
  ... Passed
Test: unreliable 2...
  ... Passed
Test: unreliable 3...
  ... Passed
Test: shard deletion (challenge 1) ...
  ... Passed
Test: unaffected shard access (challenge 2) ...
  ... Passed
Test: partial migration shard access (challenge 2) ...
  ... Passed
PASS
ok  	6.824/shardkv	101.503s
$
```



# 两个挑战

## Garbage collection of state

当一个复制组失去一个分片的所有权时，该复制组应该消除数据库中不再负责的数据。对于 It 部门来说，保留不再负责的数据是一种浪费。然而，这给迁移带来了一些问题。假设我们有两个组，G1 和 G2，并且有一个新的配置 C 将分片从 G1 移动到 G2。如果 G1 在转移到 C 时从数据库中删除了 S 中的所有键，那么 G2 在试图转移到 C 时如何获得 S 的数据呢？

challenge：使每个复制组保留旧分片的时间不超过绝对必要的时间。您的解决方案必须能够工作，即使副本组中的所有服务器（如上面的 G1 服务器）崩溃然后重新启动。如果您通过了 TestChallenge1Delete，您就完成了此挑战。



## Client requests during configuration changes

处理配置更改的最简单方法是在转换完成之前禁止所有客户端操作。虽然概念上很简单，但这种方法在生产级系统中是不可行的；每当 Join 或 Leave 机器时，都会导致所有客户端长时间暂停。最好是继续提供不受正在进行的配置更改影响的分片服务。

challenge：修改您的解决方案，以便在配置更改期间，客户端对未受影响的分片中的键的操作可以继续执行。当您顺利通过 TestChallenge2Unaffected 时，您就完成了这项挑战。

虽然上面的优化不错，但我们还可以做得更好。假设某个复制组 G3 在过渡到 C 时，需要来自 G1 的分片 S1 和来自 G2 的分片 S2。我们真的希望 G3 在收到必要的状态后立即开始服务一个分片，即使它还在等待其他分片。例如，如果 G1 停机，G3 一旦从 G2 收到适当的数据，仍然应该开始为 S2 的请求提供服务，尽管到 C 的转换尚未完成。

challenge：修改您的解决方案，以便复制组在能够提供分片服务时就开始提供服务，即使配置仍在进行中。当您通过 TestChallenge2Partial 时，您就完成了这项挑战。





---

# Part A 设计思路

对于 Part A，其实和 lab3 差不多，代码 copy 过来就行了，主要是更新状态机的部分不同罢了，另外配置本身数据量很少，所以快照也不太需要。

```go
func (sc *ShardCtrler) handleApply() {
	for sc.killed() == false {
		select {
		case applyLog := <-sc.applyCh:
			if applyLog.CommandValid {
				op, ok := applyLog.Command.(Op)
				if !ok {
					sc.DPrintf("[panic] recieved apply log's command error")
					sc.Kill()
					return
				}

				reply := &CommonReply{
					Err: OK,
				}

				if applyLog.CommandIndex <= sc.lastApplyIndex {
					// 比如 raft 重启了，就要重新 apply
					sc.DPrintf("***** command index %d is older than lastApplyIndex %d *****",
						applyLog.CommandIndex, sc.lastApplyIndex)
					continue
				}
				sc.lastApplyIndex = applyLog.CommandIndex

				sc.DPrintf("recieve apply log: %d, op info: %+v", applyLog.CommandIndex, op)
				// 防止重复应用同一条修改命令
				if op.Op != OpQuery && sc.isDuplicateRequest(op.ClientId, op.SequenceNum) {
					sc.DPrintf("found duplicate request: %+v", op)
					continue
				}

				// 更新状态机
				lastNum := len(sc.configs) - 1
				groups := make(map[int][]string)
				for k, v := range sc.configs[lastNum].Groups {
					groups[k] = v
				}
				if op.Op == OpJoin {
					for gid, servers := range op.Servers {
						groups[gid] = servers
					}
					newShards := sc.balanceShard(groups)
					sc.configs = append(sc.configs, Config{
						Num:    lastNum + 1,
						Shards: newShards,
						Groups: groups,
					})
					sc.DPrintf("config %d is %+v", lastNum+1, sc.configs[lastNum+1])
				} else if op.Op == OpLeave {
					for idx := range op.GIDs {
						gid := op.GIDs[idx]
						if _, ok := groups[gid]; ok {
							delete(groups, gid)
						}
					}
					newShards := sc.balanceShard(groups)
					sc.configs = append(sc.configs, Config{
						Num:    lastNum + 1,
						Shards: newShards,
						Groups: groups,
					})
					sc.DPrintf("config %d is %+v", lastNum+1, sc.configs[lastNum+1])
				} else if op.Op == OpMove {
					shardsMap := sc.configs[lastNum].Shards
					if op.Shard < 0 || op.Shard > NShards-1 {
						sc.DPrintf("move args error")
						sc.Kill()
						return
					}
					if _, ok := groups[op.GID]; ok {
						shardsMap[op.Shard] = op.GID
						sc.configs = append(sc.configs, Config{
							Num:    lastNum + 1,
							Shards: shardsMap,
							Groups: sc.configs[lastNum].Groups,
						})
						sc.DPrintf("config %d is %+v", lastNum+1, sc.configs[lastNum+1])
					} else {
						sc.DPrintf("undo move %d %d", op.Shard, op.GID)
					}
				} else {
					var idx int
					if op.Num == -1 || idx > lastNum {
						idx = lastNum
					} else {
						idx = op.Num
					}
					reply.Config = sc.configs[idx]
					sc.DPrintf("query config %d is %+v", idx, reply.Config)
				}

				sc.Lock("replyCommand")
				if op.Op != OpQuery {
					sc.updateDupReqHistory(op.ClientId, op.SequenceNum)
				}

				if _, ok := sc.notifyChans[applyLog.CommandIndex]; ok {
					select {
					case sc.notifyChans[applyLog.CommandIndex] <- reply:
					default:
						sc.DPrintf("reply to chan index %d failed", applyLog.CommandIndex)
					}
				}
				sc.Unlock("replyCommand")
			} else {
				sc.DPrintf(fmt.Sprintf("[panic] unexpected applyLog %v", applyLog))
				sc.Kill()
				return
			}
		default:
			continue
		}
	}
}
```

这里的点在于配置更新后，要平均负载，并尽可能地少移动数据，思路就是，先统计配置更新后无人负责的分片，并将他们分配给负载最低的 gid，然后再循环找出最大和最小负载，如果他们差值为 1 或 0，那么就表示已经平衡了，否则把负载最大的分片分一部分给最小负载的 gid。

```go
func (sc *ShardCtrler) balanceShard(groups map[int][]string) [NShards]int {
	lastNum := len(sc.configs) - 1
	shardMap := sc.configs[lastNum].Shards
	// 现在没有可用服务器
	if len(groups) == 0 {
		sc.DPrintf("No servers available now")
		for idx := range shardMap {
			shardMap[idx] = 0
		}
		return shardMap
	}

	// 统计当前服务器负载分布
	gidShardLoadInfo := make(map[int][]int)
	keys := make([]int, 0)
	for gid, _ := range groups {
		gidShardLoadInfo[gid] = make([]int, 0)
		keys = append(keys, gid)
	}
	sort.Ints(keys)

	noGidShardList := make([]int, 0)
	for idx := range shardMap {
		gid := shardMap[idx]
		if gid == 0 {
			noGidShardList = append(noGidShardList, idx)
			continue
		}
		if _, ok := gidShardLoadInfo[gid]; ok {
			// 记录 GID 负责的 shard
			gidShardLoadInfo[gid] = append(gidShardLoadInfo[gid], idx)
		} else {
			noGidShardList = append(noGidShardList, idx)
		}
	}
	sc.DPrintf("gidShardLoadInfo: %+v", gidShardLoadInfo)
	sc.DPrintf("noGidShardList: %v", noGidShardList)

	// 不再工作的 GID 把它的 shard 分配给当前 shard 负载最低的 GID
	if len(noGidShardList) > 0 {
		for i := range noGidShardList {
			shard := noGidShardList[i]
			minLoad := NShards + 1
			minLoadGid := 0
			for j := range keys {
				gid := keys[j]
				info := gidShardLoadInfo[gid]
				if len(info) < minLoad {
					minLoad = len(info)
					minLoadGid = gid
				}
			}
			gidShardLoadInfo[minLoadGid] = append(gidShardLoadInfo[minLoadGid], shard)
		}
		sc.DPrintf("gidShardLoadInfo after allocate noGidShardList: %+v", gidShardLoadInfo)
	}

	// 平均每个 GID 的负载，每次均衡最大和最小负载的 GID，直到他们差值为 1 或 0
	for {
		minLoadGid, maxLoadGid := 0, 0
		minLoad, maxLoad := NShards+1, -1
		for i := range keys {
			gid := keys[i]
			info := gidShardLoadInfo[gid]
			if len(info) < minLoad {
				minLoad = len(info)
				minLoadGid = gid
			}
			if len(info) > maxLoad {
				maxLoad = len(info)
				maxLoadGid = gid
			}
		}

		if maxLoad-minLoad < 2 {
			break
		} else {
			for maxLoad-minLoad > 1 {
				idx := len(gidShardLoadInfo[maxLoadGid]) - 1
				balanceShard := gidShardLoadInfo[maxLoadGid][idx]
				gidShardLoadInfo[maxLoadGid] = gidShardLoadInfo[maxLoadGid][:idx]
				maxLoad -= 1

				gidShardLoadInfo[minLoadGid] = append(gidShardLoadInfo[minLoadGid], balanceShard)
				minLoad += 1
			}
		}
	}

	sc.DPrintf("gidShardLoadInfo after balance: %+v", gidShardLoadInfo)

	// 生成 shardMap
	for gid, info := range gidShardLoadInfo {
		for i := range info {
			shardMap[info[i]] = gid
		}
	}
	sc.DPrintf("shardMap: %v", shardMap)
	return shardMap
}
```





# Part B 设计思路

Part B 算是整个实验最复杂的部分了，我们先来理清一下思路。每个服务器都会轮询配置，当配置变化时，服务器负责的分片情况可能会变化，那么有以下问题需要解决：

- 配置变化是否可以直接跳到最终配置？我觉得是不可以的，假设某个分片在配置 1 2 3 中分别由 x y z 负责，如果 z 直接从 x 中拉取分片，那么在 y 中接收的新请求就丢失了（y 可能还没检测到配置 3，并且分片已经从 1 拉取完毕）
- 确定了配置要按顺序变化之后，还是上面那个场景，我们怎么保证 z 从 y 拉取分片时，y 已经从 x 拉取分片成功了呢？这就需要大家保证自己处于同一轮配置变化，等待操作完毕后才可以进入下一轮
  - 每个服务器首先确认自己要推送哪些分片出去以及要新负责哪些分片，分片迁移需要携带配置变更的信息
  - 当对方确认接收完毕以及等待自己接收到需要负责的分片后，可以进入下一轮
  - 接收到推送数据时，只有其配置序号和自己一致时才可接收（因为一定要接收到指定分片才可以进入下一轮，所以不会出现某个服务器配置太高，却又需要之前某一轮配置的分片，只会存在某个服务器一直收不到某个分片而进入不了下一轮的情况）
    - 考虑一种情况，服务器 x 从 y 获取了分片，但是 y 一直没有获取到自己想要的分片，在下一轮配置，分片又分给了 y，此时 x 无法将分片推给 y，而 y 在等待另一个分片到达（这里一定要等待，因为一定要从最初的服务器获取分片才可以继续服务，问题就在于正常的分片因此阻塞而无法继续服务了）
    - 所以我们考虑仅仅服务器有配置序号信息是不够的，粒度应该更细一点，对分片记录配置序号信息，当接收的分片配置序号不小于本地时，可以接收并更新
  - 在推送和接收分片的服务器，都要暂停对此分片的服务
- 客户端发送请求时需要携带配置的序号，只有客户端和服务端分片处于同一配置时才可正常工作
  - 如果客户端配置高，那么服务端要立刻去查询新配置，并进行迁移
  - 如果服务端配置比较高，那么返回错误让客户端去更新配置













