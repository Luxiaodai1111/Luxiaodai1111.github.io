# Part 2B: log

任务：实现 leader 和 follower 代码，以追加新的日志条目，从而使 go test -run 2B 测试通过。

>[!TIP]
>
>- Run `git pull` to get the latest lab software.
>- Your first goal should be to pass `TestBasicAgree2B()`. Start by implementing `Start()`, then write the code to send and receive new log entries via `AppendEntries` RPCs, following Figure 2. Send each newly committed entry on `applyCh` on each peer.
>- You will need to implement the election restriction (section 5.4.1 in the paper).
>- One way to fail to reach agreement in the early Lab 2B tests is to hold repeated elections even though the leader is alive. Look for bugs in election timer management, or not sending out heartbeats immediately after winning an election.
>- Your code may have loops that repeatedly check for certain events. Don't have these loops execute continuously without pausing, since that will slow your implementation enough that it fails tests. Use Go's [condition variables](https://golang.org/pkg/sync/#Cond), or insert a `time.Sleep(10 * time.Millisecond)` in each loop iteration.
>- Do yourself a favor for future labs and write (or re-write) code that's clean and clear. For ideas, re-visit our the [Guidance page](https://pdos.csail.mit.edu/6.824/labs/guidance.html) with tips on how to develop and debug your code.
>- If you fail a test, look over the code for the test in `config.go` and `test_test.go` to get a better understanding what the test is testing. `config.go` also illustrates how the tester uses the Raft API.

如果你的代码运行太慢，测试可能会失败。你可以用时间命令检查你的解决方案使用了多少时间。下面是典型的输出：

```bash
$ time go test -run 2B
Test (2B): basic agreement ...
  ... Passed --   0.9  3   16    4572    3
Test (2B): RPC byte count ...
  ... Passed --   1.7  3   48  114536   11
Test (2B): agreement after follower reconnects ...
  ... Passed --   3.6  3   78   22131    7
Test (2B): no agreement if too many followers disconnect ...
  ... Passed --   3.8  5  172   40935    3
Test (2B): concurrent Start()s ...
  ... Passed --   1.1  3   24    7379    6
Test (2B): rejoin of partitioned leader ...
  ... Passed --   5.1  3  152   37021    4
Test (2B): leader backs up quickly over incorrect follower logs ...
  ... Passed --  17.2  5 2080 1587388  102
Test (2B): RPC counts aren't too high ...
  ... Passed --   2.2  3   60   20119   12
PASS
ok  	6.824/raft	35.557s

real	0m35.899s
user	0m2.556s
sys	0m1.458s
$
```

"ok 6.824/raft 35.557s " 意味着 Go 测试 2B 所花费的时间是 35.557 秒的实际时间。"user 0m2.556s " 意味着代码消耗了 2.556 秒的 CPU 时间，或实际执行指令的时间（而不是等待或睡眠）。如果你的解决方案在 2B 测试中使用的实际时间远远超过 1 分钟，或者远远超过 5 秒的 CPU 时间，你以后可能会遇到麻烦。你需要寻找时间耗费在哪了，比如等待 RPC 超时或等待通道消息或发送大量的 RPC。





---

# 设计思路



















