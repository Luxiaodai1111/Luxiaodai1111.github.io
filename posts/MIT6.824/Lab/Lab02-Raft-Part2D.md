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

## 结构体设计

根据论文要求，增加了 InstallSnapshotArgs RPC 消息，和论文不一样的是我没有使用 LastIncludeIndex 和 LastIncludeTerm，而是使用了 LastSnapLog 把最后一条日志记录下来了，其实记录的东西是一样的，只是方便我编写代码而已。

```go
type InstallSnapshotArgs struct {
   Term     int // leader 任期
   LeaderId int // 用来 follower 把客户端请求重定向到 leader
   //LastIncludeIndex int      // 快照中包含的最后日志条目的索引值
   //LastIncludeTerm  int      // 快照中包含的最后日志条目的任期号
   Offset      int      //分块在快照中的字节偏移量
   Data        []byte   // 从偏移量开始的快照分块的原始字节
   Done        bool     // 如果这是最后一个分块则为 true
   LastSnapLog LogEntry // 快照最后一条日志内容
}

type InstallSnapshotReply struct {
   Term int // 当前任期
}
```



## 快照

这里我在快照的时候在 logs 里保留的快照的最后一条日志，这样就不用单独记录元数据了，也方便代码编写。

```go
func (rf *Raft) Snapshot(index int, snapshot []byte) {
   // Your code here (2D).
   rf.DPrintf("Snapshot %d", index)

   rf.Lock("Snapshot")
   defer rf.Unlock("Snapshot")

   if index > rf.commitIndex {
      // 不能快照未提交的日志
      rf.DPrintf("[ERROR]: index %d > commitIndex %d", index, rf.commitIndex)
      return
   }
   if index <= rf.logs[0].CommandIndex {
      // 不能回退快照日志
      rf.DPrintf("[ERROR]: index %d <= rf.logs[0].CommandIndex %d", index, rf.logs[0].CommandIndex)
      return
   }

   // 避免切片内存泄露
   // 保留最后一条日志用来记录 Last Snap Log
   logIndex := index - rf.logs[0].CommandIndex
   rf.logs = append([]LogEntry{}, rf.logs[logIndex:]...)

   rf.SaveStateAndSnapshot(snapshot)
}

func (rf *Raft) SaveStateAndSnapshot(snapshot []byte) {
	rf.DPrintf("SaveStateAndSnapshot")
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	if e.Encode(rf.currentTerm) != nil ||
		e.Encode(rf.votedFor) != nil ||
		e.Encode(rf.logs) != nil {
		rf.DPrintf("------ persist encode error ------")
	}
	state := w.Bytes()

	rf.persister.SaveStateAndSnapshot(state, snapshot)
}
```



## 追加日志

追加日志和之前不同的就是如果要发送的日志在快照里，那么就需要发送快照，2B 如果理解了，其实加快照功能并不是很难，细心就行了。

```go
func (rf *Raft) replicate(peer int, syncCommit bool) {
	rf.Lock("replicate")
	if rf.role != Leader {
		rf.Unlock("replicate")
		rf.DPrintf("now is not leader, cancel send append entries")
		return
	}
	request := &AppendEntriesArgs{
		Term:         rf.currentTerm,
		LeaderId:     rf.me,
		LeaderCommit: rf.commitIndex,
		Entries:      nil,
	}

	if rf.nextIndex[peer] <= rf.logs[0].CommandIndex {
		go rf.sendSnap(peer)
		rf.Unlock("replicate")
		return
	}

	lastLog := rf.getLastLog()
	if rf.nextIndex[peer] < lastLog.CommandIndex+1 {
		// 存在待提交日志
		rf.DPrintf("peer %d's nextIndex is %d", peer, rf.nextIndex[peer])
		request.Entries = rf.logs[rf.index(rf.nextIndex[peer]):]
	}

	// 根据是否携带日志来填充参数
	if len(request.Entries) > 0 {
		prevLog := rf.log(request.Entries[0].CommandIndex - 1)
		request.PrevLogIndex = prevLog.CommandIndex
		request.PrevLogTerm = prevLog.Term
		rf.DPrintf("send log %d-%d to %d",
			request.Entries[0].CommandIndex, request.Entries[len(request.Entries)-1].CommandIndex, peer)
	} else {
		request.PrevLogIndex = rf.nextIndex[peer] - 1
		request.PrevLogTerm = rf.log(rf.nextIndex[peer] - 1).Term
		rf.DPrintf("send heartbeat %+v to %d", request, peer)
	}
	rf.Unlock("replicate")

	response := new(AppendEntriesReply)
	if rf.sendAppendEntries(peer, request, response) {
		rf.DPrintf("receive AppendEntriesReply from %d, response is %+v", peer, response)
		rf.Lock("recvAppendEntries")
		defer rf.Unlock("recvAppendEntries")

		// 过期轮次的回复直接丢弃
		if request.Term < rf.currentTerm {
			return
		}

		rf.checkTerm(peer, response.Term)

		if rf.role != Leader {
			rf.DPrintf("now is not leader")
			return
		}

		if response.Success {
			if request.Entries == nil || len(request.Entries) == 0 {
				return
			}
			lastEntryIndex := request.Entries[len(request.Entries)-1].CommandIndex
			if lastEntryIndex > rf.matchIndex[peer] {
				rf.matchIndex[peer] = lastEntryIndex
				rf.nextIndex[peer] = rf.matchIndex[peer] + 1
				rf.DPrintf("peer %d's matchIndex is %d", peer, rf.matchIndex[peer])
			}
		} else {
			var nextIndex int
			oldNextIndex := rf.nextIndex[peer]
			lastLog = rf.getLastLog()

			// 检查冲突日志
			if rf.logs[0].CommandIndex <= response.ConflictIndex && response.ConflictIndex <= lastLog.CommandIndex &&
				rf.log(response.ConflictIndex).Term == response.ConflictTerm {
				// 如果日志匹配的话，下次就从这条日志发起
				nextIndex = response.ConflictIndex
			} else if response.ConflictIndex < rf.logs[0].CommandIndex {
				// 冲突索引在本地快照中，那么直接发送快照
				nextIndex = response.ConflictIndex
			} else {
				// 如果冲突，则从冲突日志的上一条发起
				if response.ConflictIndex <= oldNextIndex {
					nextIndex = response.ConflictIndex - 1
				} else {
					nextIndex = oldNextIndex - 1
				}
			}
			// 冲突索引只能往回退
			if nextIndex < oldNextIndex {
				rf.nextIndex[peer] = nextIndex
			}
			// 索引要大于 matchIndex
			if rf.matchIndex[peer] >= nextIndex {
				rf.nextIndex[peer] = rf.matchIndex[peer] + 1
			}
			// 有冲突要立马再次发送日志去快速同步
			if rf.nextIndex[peer] < oldNextIndex {
				rf.DPrintf("====== Fast Synchronization %d ======", peer)
				go rf.replicate(peer, false)
			}

			rf.DPrintf("peer %d's nextIndex update to %d", peer, rf.nextIndex[peer])
		}
	} else {
		rf.DPrintf("send append entries RPC to %d failed", peer)
	}
}
```

对于追加日志的处理主要就是匹配日志时，如果在快照中，那么一定就是匹配的，因为快照都是已提交的日志。

```go
func (rf *Raft) checkLogMatch(PrevLogIndex int, PrevLogTerm int) bool {
   lastLog := rf.getLastLog()
   if rf.logs[0].CommandIndex <= PrevLogIndex && PrevLogIndex <= lastLog.CommandIndex &&
      rf.log(PrevLogIndex).Term == PrevLogTerm {
      // 日志在 logs 中存在且匹配
      return true
   } else if PrevLogIndex <= rf.logs[0].CommandIndex {
      // 日志在快照中，一定匹配
      return true
   }

   return false
}
```



## 发送快照

这里根据实验建议没有将快照分片，这样处理比较简单。

```go
func (rf *Raft) sendSnap(peer int) {
   rf.Lock("sendSnap")
   if rf.role != Leader {
      rf.Unlock("sendSnap")
      rf.DPrintf("now is not leader, cancel send snap")
      return
   }

   request := &InstallSnapshotArgs{
      Term:        rf.currentTerm,
      LeaderId:    rf.me,
      Offset:      0,
      Data:        rf.persister.ReadSnapshot(),
      Done:        true, // 不分片，一次传输
      LastSnapLog: rf.logs[0],
   }
   rf.Unlock("sendSnap")

   rf.DPrintf("====== sendSnap %d to %d ======", request.LastSnapLog.CommandIndex, peer)
   response := new(InstallSnapshotReply)
   if rf.sendInstallSnapshot(peer, request, response) {
      rf.DPrintf("receive InstallSnapshotReply from %d, response is %+v", peer, response)
      rf.Lock("recvInstallSnapshotReply")
      defer rf.Unlock("recvInstallSnapshotReply")

      // 过期轮次的回复直接丢弃
      if request.Term < rf.currentTerm {
         return
      }

      rf.checkTerm(peer, response.Term)

      if rf.role != Leader {
         rf.DPrintf("now is not leader")
         return
      }

      rf.matchIndex[peer] = request.LastSnapLog.CommandIndex
      rf.nextIndex[peer] = rf.matchIndex[peer] + 1
      rf.DPrintf("peer %d's matchIndex is %d", peer, rf.matchIndex[peer])

      go rf.replicate(peer, false)
   }

}
```

如果快照比本地快照新，那就无脑追加好了，这里为了处理状态机更新，我把快照的 apply 先放到了 rf.internalApplyList 队列里，然后在 apply 协程统一处理。

```go
func (rf *Raft) InstallSnapshot(request *InstallSnapshotArgs, response *InstallSnapshotReply) {
	rf.Lock("InstallSnapshot")
	defer rf.Unlock("InstallSnapshot")
	response.Term = rf.currentTerm
	if request.Term < rf.currentTerm {
		rf.DPrintf("refuse InstallSnapshot from %d", request.LeaderId)
		return
	}

	rf.checkTerm(request.LeaderId, request.Term)

	// 本地快照更新则忽略此快照
	if request.LastSnapLog.CommandIndex <= rf.logs[0].CommandIndex {
		rf.DPrintf("local snap is more newer")
		return
	}

	findMatchLog := false
	for idx := 0; idx < len(rf.logs); idx++ {
		if rf.logs[idx].CommandIndex == request.LastSnapLog.CommandIndex &&
			rf.logs[idx].Term == request.LastSnapLog.Term {
			rf.logs = append([]LogEntry{}, rf.logs[idx:]...)
			rf.DPrintf("update logs")
			rf.printLog()
			findMatchLog = true
			break
		}
	}
	if !findMatchLog {
		rf.logs = append([]LogEntry{}, request.LastSnapLog)
		rf.DPrintf("update logs")
		rf.printLog()
	}

	rf.SaveStateAndSnapshot(request.Data)

	rf.internalApplyList = append(rf.internalApplyList, ApplyMsg{
		CommandValid:  false,
		Command:       nil,
		CommandIndex:  0,
		SnapshotValid: true,
		Snapshot:      request.Data,
		SnapshotTerm:  request.LastSnapLog.Term,
		SnapshotIndex: request.LastSnapLog.CommandIndex,
	})
}
```



## 更新状态机

这里增加了对快照的 apply。

```go
func (rf *Raft) apply() {
	for rf.killed() == false {
		if rf.lastApplied < rf.commitIndex {
			// 先把要提交的日志整合出来，避免占用锁
			rf.Lock("apply")
			internalApplyList := make([]ApplyMsg, 0)
			if len(rf.internalApplyList) > 0 {
				// 取出要 apply 的快照
				internalApplyList = append(internalApplyList, rf.internalApplyList...)
				// 清空队列
				rf.internalApplyList = make([]ApplyMsg, 0)
			}
			if rf.lastApplied >= rf.logs[0].CommandIndex {
				for idx := rf.lastApplied + 1; idx <= rf.commitIndex; idx++ {
					internalApplyList = append(internalApplyList, ApplyMsg{
						CommandValid: true,
						Command:      rf.log(idx).Command,
						CommandIndex: rf.log(idx).CommandIndex,
					})
				}
			}
			rf.Unlock("apply")

			// 对 internalApplyList 按日志索引排序
			for i := 0; i <= len(internalApplyList)-1; i++ {
				for j := i; j <= len(internalApplyList)-1; j++ {
					x := internalApplyList[i].CommandIndex
					if internalApplyList[i].SnapshotValid {
						x = internalApplyList[i].SnapshotIndex
					}
					y := internalApplyList[j].CommandIndex
					if internalApplyList[j].SnapshotValid {
						y = internalApplyList[j].SnapshotIndex
					}
					if x > y {
						t := internalApplyList[i]
						internalApplyList[i] = internalApplyList[j]
						internalApplyList[j] = t
					}
				}
			}

			for _, applyMsg := range internalApplyList {
				if (applyMsg.CommandValid && applyMsg.CommandIndex > rf.lastApplied) ||
					(applyMsg.SnapshotValid && applyMsg.SnapshotIndex > rf.lastApplied) {
					rf.applyCh <- applyMsg
					if applyMsg.SnapshotValid {
						rf.DPrintf("====== apply snap, committed index: %d ======", applyMsg.SnapshotIndex)
						rf.lastApplied = applyMsg.SnapshotIndex
					} else {
						rf.DPrintf("====== apply committed log %d ======", applyMsg.CommandIndex)
						rf.lastApplied = applyMsg.CommandIndex
					}
				}
			}
		}
	}
}
```



## 测试

Lab 2 整个实验还是蛮有难度的，通过撒花🎉🎉

```bash
root@root:~/lz/6.824/src/raft# go test
Test (2A): initial election ...
  ... Passed --   3.5  3   61   17858    0
Test (2A): election after network failure ...
  ... Passed --   5.6  3  127   27240    0
Test (2A): multiple elections ...
  ... Passed --   7.4  7  590  125349    0
Test (2B): basic agreement ...
  ... Passed --   1.2  3   15    4456    3
Test (2B): RPC byte count ...
  ... Passed --   2.9  3   46  114189   11
Test (2B): agreement after follower reconnects ...
  ... Passed --   4.9  3   85   23472    7
Test (2B): no agreement if too many followers disconnect ...
  ... Passed --   3.9  5  167   40326    3
Test (2B): concurrent Start()s ...
  ... Passed --   1.4  3   25    7693    6
Test (2B): rejoin of partitioned leader ...
  ... Passed --   6.8  3  186   48291    4
Test (2B): leader backs up quickly over incorrect follower logs ...
  ... Passed --  25.5  5 2430 2291421  102
Test (2B): RPC counts aren't too high ...
  ... Passed --   2.5  3   59   19324   12
Test (2C): basic persistence ...
  ... Passed --   4.9  3   78   21593    6
Test (2C): more persistence ...
  ... Passed --  19.3  5  933  222051   16
Test (2C): partitioned leader and one follower crash, leader restarts ...
  ... Passed --   2.3  3   34    9167    4
Test (2C): Figure 8 ...
  ... Passed --  29.5  5  572  138293   35
Test (2C): unreliable agreement ...
  ... Passed --   3.5  5  996  362805  246
Test (2C): Figure 8 (unreliable) ...
  ... Passed --  44.5  5 8280 16249316   63
Test (2C): churn ...
  ... Passed --  16.3  5 7373 54233707 1763
Test (2C): unreliable churn ...
  ... Passed --  16.3  5 2454 4397045  578
Test (2D): snapshots basic ...
  ... Passed --   3.5  3  489  242625  207
Test (2D): install snapshots (disconnect) ...
  ... Passed --  43.7  3 1472  961293  313
Test (2D): install snapshots (disconnect+unreliable) ...
  ... Passed --  56.3  3 1780  896737  343
Test (2D): install snapshots (crash) ...
  ... Passed --  38.0  3 1173  556516  328
Test (2D): install snapshots (unreliable+crash) ...
  ... Passed --  44.5  3 1299  796843  340
Test (2D): crash and restart all servers ...
  ... Passed --  15.2  3  255   79096   59
PASS
ok  	6.824/raft	403.618s
```

​	
