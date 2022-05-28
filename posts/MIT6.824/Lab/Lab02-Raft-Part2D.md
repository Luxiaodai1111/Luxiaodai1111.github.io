# Part 2D: log compaction

按照目前的情况，重新启动的服务器会复制完整的 Raft 日志以恢复其状态。然而，对于一个长期运行的服务来说，永远记住完整的 Raft 日志是不现实的。相反，你将修改 Raft，使其与那些不时持久性地存储其状态的 "快照" 的服务合作，此时 Raft 会丢弃快照之前的日志条目。其结果是持久性数据量更小，重启速度更快。然而，现在追随者有可能落后太多，以至于领导者丢弃了它需要追赶的日志条目；然后领导者必须发送一个快照，加上快照时间开始的日志。论文的第 7 节概述了该方案；你必须自己设计实现细节。

你的 Raft 必须提供以下函数，服务可以使用其状态的序列化快照来调用该函数：

```go
Snapshot(index int, snapshot []byte)
```

在 Lab 2D 中，测试者定期调用 Snapshot()。在  Lab 3 中，你将编写一个调用 Snapshot() 的键值服务器；快照将包含键值对的完整表格。服务层在每个实例上调用 Snapshot()（而不仅仅是在领导者上）。

index 参数表示在快照中反映的最高日志条目。Raft 应该丢弃在该点之前的日志条目。你需要修改你的 Raft 代码，以便在操作时只存储日志的尾部。

你需要实现论文中讨论的 InstallSnapshot RPC，它允许 Raft 领导告诉落后的 Raft 对端用快照替换其状态。你可能需要考虑 InstallSnapshot 应该如何与图 2 中的状态和规则互动。

当跟随者的 Raft 代码收到 InstallSnapshot RPC 时，它可以使用 applyCh 在 ApplyMsg 中向服务发送快照。ApplyMsg 结构定义已经包含了您需要的字段（也是测试人员所期望的）。请注意，这些快照只能推进服务的状态，而不会导致它向后移动。

如果一个服务器崩溃了，它必须从持久化的数据中重新启动。你的 Raft 应该同时保存 Raft 状态和相应的快照。使用 persister.SaveStateAndSnapshot()，它为 Raft 状态和相应的快照接受单独的参数。如果没有快照，则传递 nil 作为快照参数。

当服务器重新启动时，应用层会读取持久化的快照并恢复其保存的状态。

以前本实验会建议你实现一个叫做 CondInstallSnapshot 的函数，以避免发送给 applyCh 的快照和日志条目被 coordinated。这个残存的 API 接口仍然存在，但我们不鼓励你去实现它：相反，我们建议你只需让它返回 true。

任务：实现 Snapshot() 和 InstallSnapshot RPC，以及对 Raft 修改以支持这些功能（例如，用修剪后的日志进行操作）。当你的解决方案通过 2D 测试（以及之前所有的 Lab 2 测试）时，就完成了。

>[!TIP]
>
>- `git pull` to make sure you have the latest software.
>- A good place to start is to modify your code to so that it is able to store just the part of the log starting at some index X. Initially you can set X to zero and run the 2B/2C tests. Then make `Snapshot(index)` discard the log before `index`, and set X equal to `index`. If all goes well you should now pass the first 2D test.
>- You won't be able to store the log in a Go slice and use Go slice indices interchangeably with Raft log indices; you'll need to index the slice in a way that accounts for the discarded portion of the log.
>- Next: have the leader send an InstallSnapshot RPC if it doesn't have the log entries required to bring a follower up to date.
>- Send the entire snapshot in a single InstallSnapshot RPC. Don't implement Figure 13's `offset` mechanism for splitting up the snapshot.
>- Raft must discard old log entries in a way that allows the Go garbage collector to free and re-use the memory; this requires that there be no reachable references (pointers) to the discarded log entries.
>- Even when the log is trimmed, your implemention still needs to properly send the term and index of the entry prior to new entries in `AppendEntries` RPCs; this may require saving and referencing the latest snapshot's `lastIncludedTerm/lastIncludedIndex` (consider whether this should be persisted).
>- A reasonable amount of time to consume for the full set of Lab 2 tests (2A+2B+2C+2D) without `-race` is 6 minutes of real time and one minute of CPU time. When running with `-race`, it is about 10 minutes of real time and two minutes of CPU time.

你的代码应该通过所有的 2D 测试（如下图所示），以及 2A、2B 和 2C 测试。

```bash
$ go test -run 2D
Test (2D): snapshots basic ...
  ... Passed --  11.6  3  176   61716  192
Test (2D): install snapshots (disconnect) ...
  ... Passed --  64.2  3  878  320610  336
Test (2D): install snapshots (disconnect+unreliable) ...
  ... Passed --  81.1  3 1059  375850  341
Test (2D): install snapshots (crash) ...
  ... Passed --  53.5  3  601  256638  339
Test (2D): install snapshots (unreliable+crash) ...
  ... Passed --  63.5  3  687  288294  336
Test (2D): crash and restart all servers ...
  ... Passed --  19.5  3  268   81352   58
PASS
ok      6.824/raft      293.456s
```





---

# 设计思路

