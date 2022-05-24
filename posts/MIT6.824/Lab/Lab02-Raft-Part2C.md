# Part 2C: persistence 

真正的实现会在每次 Raft 的持久化状态发生变化时将其写入磁盘，并在重启后重新启动时从磁盘读取状态。

你的实现不会使用磁盘；相反，它将从 Persister 对象（见 persister.go）保存和恢复持久化状态。Raft.Make() 提供了一个 Persister，它最初持有 Raft 最近的持久化状态（如果有的话）。Raft 使用 Persister 的 ReadRaftState 和 SaveRaftState 方法。，从该 Persister 初始化其状态，并在每次状态改变时使用它来保存其持久化状态。

任务：通过添加保存和恢复持久化状态的代码，完成 raft.go 中的 persist() 和 readPersist() 函数。你将需要把状态编码或序列化为一个字节数组，以便将其传递给持久化器。你将使用 labgob 编码器（参见 persist() 和 readPersist() 中的注释）。labgob 就像 Go 的 gob 编码器，但如果你用小写的字段名对结构进行编码，会打印出错误信息。在你的实现改变持久化状态的地方插入对 persist() 的调用。一旦你完成了这些，并且如果你的其他实现是正确的，你就应该通过所有的 2C 测试。

>[!TIP]
>
>- Run `git pull` to get the latest lab software.
>- The 2C tests are more demanding than those for 2A or 2B, and failures may be caused by problems in your code for 2A or 2B.
>- You will probably need the optimization that backs up nextIndex by more than one entry at a time. Look at the [extended Raft paper](https://pdos.csail.mit.edu/6.824/papers/raft-extended.pdf) starting at the bottom of page 7 and top of page 8 (marked by a gray line). The paper is vague about the details; you will need to fill in the gaps, perhaps with the help of the 6.824 Raft lecture notes.

测试通过应该打印如下：

```bash
$ go test -run 2C
Test (2C): basic persistence ...
  ... Passed --   5.0  3   86   22849    6
Test (2C): more persistence ...
  ... Passed --  17.6  5  952  218854   16
Test (2C): partitioned leader and one follower crash, leader restarts ...
  ... Passed --   2.0  3   34    8937    4
Test (2C): Figure 8 ...
  ... Passed --  31.2  5  580  130675   32
Test (2C): unreliable agreement ...
  ... Passed --   1.7  5 1044  366392  246
Test (2C): Figure 8 (unreliable) ...
  ... Passed --  33.6  5 10700 33695245  308
Test (2C): churn ...
  ... Passed --  16.1  5 8864 44771259 1544
Test (2C): unreliable churn ...
  ... Passed --  16.5  5 4220 6414632  906
PASS
ok  	6.824/raft	123.564s
$
```

建议你最好多测试几遍。

```bash
$ for i in {0..10}; do go test; done
```





---

# 设计思路

