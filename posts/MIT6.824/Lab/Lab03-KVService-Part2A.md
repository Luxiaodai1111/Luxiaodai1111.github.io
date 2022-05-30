# 实验介绍

在本实验中，你将使用 Lab 2 中的 Raft 库来构建一个可容错的键值存储服务。客户端可以向服务发送三种不同的 RPC：

-   Put(key, value)：替换数据库中某个特定键的值
-   Append(key，arg)：将 arg 附加到键的值上
-   Get(key)：获取键的当前值

键和值都是字符串。对于一个不存在的键，Get 应该返回一个空字符串。对一个不存在的键的 Append 应该像 Put 一样操作。

每个客户端通过一个带有 Put/Append/Get 方法的 Clerk 与服务对话。Clerk 负责管理与服务器的 RPC 交互。

服务要求是线性一致的。例如，如果一个客户端从服务中获得了一个更新请求的成功响应，那么随后从其他客户端发起的读取就能保证看到该更新的效果。

本实验有两部分。在 A 部分，您将使用您的 Raft 实现实现一个键/值服务，但不使用快照。在 B 部分中，您将使用 Lab 2D 中的快照实现，这将使 Raft 能够丢弃旧的日志条目。





---

# Getting Started

我们在 src/kvraft 中为你提供了骨架代码和测试。你将需要修改 kvraft/client.go，kvraft/server.go，也许还有 kvraft/common.go。

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

Clerk 有时不知道哪个 kvserver 是 Raft 的领导者。如果 Clerk 向错误的 kvserver 发送 RPC，或者无法到达该 kvserver，Clerk 应该通过向不同的 kvserver 发送来重新尝试。如果键/值服务将操作提交给它的 Raft 日志（并因此将操作应用于键/值状态机），领导者通过响应它的 RPC 将结果报告给 Clerk。如果操作未能提交（例如，如果领导者被替换了），服务器会报告一个错误，Clerk 会用另一个服务器重试。

你的 kvservers 不应该直接交流，它们应该只通过 Raft 进行相互交流。

实验任务 1：首先实现不丢失消息以及没有失败的服务器场景下的解决方案。你需要为 client.go 中的 Clerk Put/Append/Get 方法添加 RPC 发送代码，并在 server.go 中实现 PutAppend() 和 Get() RPC 处理程序。这些处理程序应使用 Start() 在 Raft 日志中输入一个Op；你应在 server.go 中填写 Op 结构定义，使其描述一个 Put/Append/Get 操作。每个服务器应该在 Raft 提交 Op 命令时执行这些命令，也就是说，当它们出现在 applyCh 上时。RPC 处理程序应该注意到 Raft 何时提交其 Op，然后回复 RPC。

>[!TIP]
>
>- After calling `Start()`, your kvservers will need to wait for Raft to complete agreement. Commands that have been agreed upon arrive on the `applyCh`. Your code will need to keep reading `applyCh` while `PutAppend()` and `Get()` handlers submit commands to the Raft log using `Start()`. Beware of deadlock between the kvserver and its Raft library.
>- You are allowed to add fields to the Raft `ApplyMsg`, and to add fields to Raft RPCs such as `AppendEntries`, however this should not be necessary for most implementations.
>- A kvserver should not complete a `Get()` RPC if it is not part of a majority (so that it does not serve stale data). A simple solution is to enter every `Get()` (as well as each `Put()` and `Append()`) in the Raft log. You don't have to implement the optimization for read-only operations that is described in Section 8.
>- It's best to add locking from the start because the need to avoid deadlocks sometimes affects overall code design. Check that your code is race-free using `go test -race`.

现在你应该修改你的解决方案，以便在面对网络和服务器故障时能够继续下去。你将面临的一个问题是，Clerk 可能要多次发送 RPC，直到它找到一个积极回复的 kvserver。如果一个领导者在向 Raft 日志提交条目后发生故障，Clerk 可能不会收到回复，因此可能会向另一个领导者重新发送请求。对 Clerk.Put() 或 Clerk.Append() 的每次调用应该只导致一次执行，所以你必须确保重新发送不会导致服务器执行两次请求。

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





---

# 设计思路



​	
