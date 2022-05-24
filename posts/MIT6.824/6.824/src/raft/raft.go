package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	"6.824/labgob"
	"bytes"
	"math/rand"
	//	"bytes"
	"sync"
	"sync/atomic"
	"time"

	//	"6.824/labgob"
	"6.824/labrpc"
)

//
// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in part 2D you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh, but set CommandValid to false for these
// other uses.
//
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int

	// For 2D:
	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  int
	SnapshotIndex int
}

type Role int

const (
	Leader Role = iota
	Candidate
	Follower
)

const PlugNumber = 100

type LogEntry struct {
	Term         int         // 日志所属的任期
	CommandIndex int         // 日志槽位
	Command      interface{} // 状态机命令
}

func (rf *Raft) Lock(owner string) {
	//rf.DPrintf("%s Lock", owner)
	rf.mu.Lock()
}

func (rf *Raft) Unlock(owner string) {
	//rf.DPrintf("%s Unlock", owner)
	rf.mu.Unlock()
}

func (rf *Raft) getLastLog() LogEntry {
	return rf.log[len(rf.log)-1]
}

// 获取 term 任期内的第一条日志，这里保证 term 任期内一定有日志
func (rf *Raft) getFirstLog(term int) (ConflictIndex int) {
	rf.DPrintf("getFirstLog in term %d", term)
	low := 0
	high := len(rf.log)
	middle := (low + high) / 2
	// 二分先找到一条包含该任期的日志
	for ; low < high; middle = (low + high) / 2 {
		if rf.log[middle].Term == term {
			break
		} else if rf.log[middle].Term < term {
			low = middle + 1
		} else {
			high = middle - 1
		}
	}

	for i := middle; i >= 0; i-- {
		if rf.log[i].Term != term {
			rf.DPrintf("getFirstLog in term %d: %d", term, i+1)
			return i + 1
		}
	}

	// 没有 BUG 的话永远不会走到这
	rf.DPrintf("%+v %d", rf.log, middle)
	return -1
}

func (rf *Raft) AppendEntries(request *AppendEntriesArgs, response *AppendEntriesReply) {
	rf.Lock("AppendEntries")
	defer rf.Unlock("AppendEntries")
	if request.Term < rf.currentTerm {
		response.Success = false
		response.Term = rf.currentTerm
		rf.DPrintf("refuse AppendEntries")
		return
	}

	rf.checkTerm(request.LeaderId, request.Term)
	response.Term = rf.currentTerm

	rf.printLog()
	rf.DPrintf("commitIndex: %d, lastApplied: %d", rf.commitIndex, rf.lastApplied)

	// 检查日志
	rf.DPrintf("PrevLogIndex: %d, PrevLogTerm: %d, LeaderCommit: %d", request.PrevLogIndex, request.PrevLogTerm, request.LeaderCommit)
	if len(rf.log) > request.PrevLogIndex && rf.log[request.PrevLogIndex].Term == request.PrevLogTerm {
		// 追加日志
		if len(request.Entries) > 0 {
			// 本地日志要更新一些，拒绝接收
			lastLog := rf.getLastLog()
			lastEntry := request.Entries[len(request.Entries)-1]
			if lastLog.Term > lastEntry.Term || (lastLog.Term == lastEntry.Term && lastLog.CommandIndex > lastEntry.CommandIndex) {
				rf.DPrintf("local log is newer than %d, refuse to recv log", request.LeaderId)
				response.Success = false
				response.ConflictTerm = request.PrevLogTerm
				response.ConflictIndex = request.PrevLogIndex
				return
			}
			// 追加日志
			rf.log = append(rf.log[:request.PrevLogIndex+1], request.Entries...)
			rf.persist()
			rf.DPrintf("====== append log %d-%d ======",
				request.Entries[0].CommandIndex, request.Entries[len(request.Entries)-1].CommandIndex)
		}

		// 如果 leaderCommit 大于 commitIndex，设置本地 commitIndex 为 leaderCommit 和最新日志索引中较小的一个。
		lastLog := rf.getLastLog()
		if request.LeaderCommit > rf.commitIndex {
			if lastLog.CommandIndex < request.LeaderCommit {
				rf.commitIndex = lastLog.CommandIndex
			} else {
				rf.commitIndex = request.LeaderCommit
			}
			rf.DPrintf("update commitIndex %d", rf.commitIndex)
		}

		response.Success = true
		rf.role = Follower
		rf.electionTimer.Reset(rf.ElectionTimeout())
	} else {
		rf.DPrintf("log mismatch")
		// 如果自己不存在索引、任期和 prevLogIndex、 prevLogItem 匹配的日志返回 false。
		response.Success = false
		// 如果存在一条日志索引和 prevLogIndex 相等，但是任期和 prevLogItem 不相同的日志，需要删除这条日志及所有后继日志。
		if len(rf.log) > request.PrevLogIndex && rf.log[request.PrevLogIndex].Term != request.Term {
			rf.DPrintf("====== delete log from %d ======", request.PrevLogIndex)
			rf.log = rf.log[:request.PrevLogIndex]
			rf.persist()
		}
		// 加速日志冲突检查, 获取不大于 request.PrevLogTerm 且包含日志的冲突条目
		lastLog := rf.getLastLog()
		for i := lastLog.CommandIndex; i >= 0; i-- {
			if rf.log[i].Term <= request.PrevLogTerm {
				response.ConflictTerm = rf.log[i].Term
				response.ConflictIndex = rf.getFirstLog(rf.log[i].Term)
				if response.ConflictIndex == request.PrevLogIndex {
					// 获取含有日志的上一个任期
					prevLog := rf.log[response.ConflictIndex-1]
					response.ConflictTerm = prevLog.Term
					response.ConflictIndex = rf.getFirstLog(prevLog.Term)
				}
				break
			}
		}
	}

	return
}

// 如果接收到的 RPC 请求或响应中，任期号 T > currentTerm，则令 currentTerm = T，并切换为 follower 状态
func (rf *Raft) checkTerm(peer int, term int) {
	if term > rf.currentTerm {
		rf.DPrintf("====== peer %d's term %d > currentTerm ======", peer, term)
		rf.role = Follower
		rf.currentTerm = term
		rf.votedFor = -1
		rf.persist()
	}
}

func (rf *Raft) sendLog(peer int, logs []LogEntry) {
	rf.Lock("sendLog")
	if rf.role != Leader {
		rf.Unlock("sendLog")
		rf.DPrintf("now is not leader, cancel send log")
		return
	}
	prevLog := rf.log[logs[0].CommandIndex-1]
	request := &AppendEntriesArgs{
		Term:         rf.currentTerm,
		LeaderId:     rf.me,
		PrevLogIndex: prevLog.CommandIndex,
		PrevLogTerm:  prevLog.Term,
		Entries:      logs, // 一次可以发送多条日志
		LeaderCommit: rf.commitIndex,
	}
	rf.Unlock("sendLog")

	response := new(AppendEntriesReply)
	rf.DPrintf("send log %d-%d to %d", logs[0].CommandIndex, logs[len(logs)-1].CommandIndex, peer)
	if rf.sendAppendEntries(peer, request, response) {
		rf.Lock("recvAppendEntries")
		defer rf.Unlock("recvAppendEntries")
		rf.DPrintf("receive AppendEntriesReply from %d, response is %+v", peer, response)
		rf.checkTerm(peer, response.Term)
		if rf.role != Leader {
			rf.DPrintf("now is not leader")
			return
		}
		if response.Success {
			if logs[len(logs)-1].CommandIndex > rf.matchIndex[peer] {
				rf.matchIndex[peer] = logs[len(logs)-1].CommandIndex
			}
			rf.nextIndex[peer] = rf.matchIndex[peer] + 1
			rf.DPrintf("peer %d's matchIndex is %d", peer, rf.matchIndex[peer])
		} else {
			// 检查冲突日志
			if len(rf.log) > response.ConflictIndex && rf.log[response.ConflictIndex].Term == response.ConflictTerm {
				// 如果日志匹配的话，下次就从这条日志发起
				rf.nextIndex[peer] = response.ConflictIndex
			} else {
				// 如果冲突，则从冲突日志的上一条发起
				rf.nextIndex[peer] = response.ConflictIndex - 1
			}
			// 索引至少从 1 开始
			if rf.nextIndex[peer] < 1 {
				rf.nextIndex[peer] = 1
			}

			rf.DPrintf("peer %d's nextIndex is %d", peer, rf.nextIndex[peer])
		}
	} else {
		rf.DPrintf("send log RPC failed: %d", peer)
	}

	return
}

func (rf *Raft) sendHeartbeat(peer int) {
	rf.Lock("sendHeartbeat")
	if rf.role != Leader {
		rf.Unlock("sendHeartbeat")
		rf.DPrintf("now is not leader, cancel sendHeartbeat")
		return
	}
	request := &AppendEntriesArgs{
		Term:         rf.currentTerm,
		LeaderId:     rf.me,
		PrevLogIndex: rf.matchIndex[peer],
		PrevLogTerm:  rf.log[rf.matchIndex[peer]].Term,
		Entries:      nil,
		LeaderCommit: rf.commitIndex,
	}
	rf.Unlock("sendHeartbeat")

	response := new(AppendEntriesReply)
	rf.DPrintf("send heartbeat %+v to %d", request, peer)
	if rf.sendAppendEntries(peer, request, response) {
		rf.DPrintf("receive AppendEntriesReply from %d, response is %+v", peer, response)
		rf.Lock("recvHeartbeat")
		defer rf.Unlock("recvHeartbeat")
		rf.checkTerm(peer, response.Term)
	} else {
		rf.DPrintf("heartbeat RPC failed: %d", peer)
	}
}

func (rf *Raft) broadcastHeartbeat(syncCommit bool) {
	for peer := range rf.peers {
		if peer == rf.me {
			continue
		}
		if syncCommit {
			// 成为 Leader 后发送最后一条日志来触发提交
			lastLog := rf.getLastLog()
			if lastLog.CommandIndex == 0 {
				go rf.sendHeartbeat(peer)
			} else {
				go rf.sendLog(peer, []LogEntry{lastLog})
			}
			continue
		}
		if rf.nextIndex[peer] < len(rf.log) {
			rf.DPrintf("peer %d's nextIndex is %d", peer, rf.nextIndex[peer])
			go rf.sendLog(peer, rf.log[rf.nextIndex[peer]:])
		} else {
			go rf.sendHeartbeat(peer)
		}
	}

	return
}

//
// A Go object implementing a single Raft peer.
//
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	role        Role       // 服务器当前角色
	currentTerm int        // 服务器已知最新的任期，在服务器首次启动时初始化为 0，单调递增
	votedFor    int        // 当前任期内接受选票的竞选者 Id，如果没有投给任何候选者则为空
	log         []LogEntry // 日志条目

	commitIndex int // 已提交的最高的日志条目的索引
	lastApplied int // 已经被提交到状态机的最后一个日志的索引

	nextIndex  []int // 对于每一台服务器，下条发送到该机器的日志索引
	matchIndex []int // 对于每一台服务器，已经复制到该服务器的最高日志条目的索引

	electionTimer  *time.Timer // 选举计时器
	heartbeatTimer *time.Timer // 心跳计时器

	applyCh chan ApplyMsg
	plug    int // 心跳时间内积攒一定数目的日志再一起发送
}

func (rf *Raft) HeartbeatTimeout() time.Duration {
	return time.Millisecond * 100
}

func (rf *Raft) ElectionTimeout() time.Duration {
	rand.Seed(time.Now().Unix() + int64(rf.me))
	return time.Millisecond * time.Duration(500+rand.Int63n(500))
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {
	var isleader bool

	rf.Lock("GetState")
	defer rf.Unlock("GetState")
	if rf.role == Leader {
		isleader = true
	} else {
		isleader = false
	}

	return rf.currentTerm, isleader
}

//
// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
//
func (rf *Raft) persist() {
	//rf.DPrintf("persist currentTerm: %d, votedFor: %d", rf.currentTerm, rf.votedFor)
	//rf.printLog()
	// Your code here (2C).
	// Example:
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	if e.Encode(rf.currentTerm) != nil ||
		e.Encode(rf.votedFor) != nil ||
		e.Encode(rf.log) != nil {
		rf.DPrintf("------ persist encode error ------")
	}
	data := w.Bytes()
	rf.persister.SaveRaftState(data)
}

//
// restore previously persisted state.
//
func (rf *Raft) readPersist(data []byte) {
	rf.DPrintf("====== readPersist ======")
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
	// Example:
	r := bytes.NewBuffer(data)
	d := labgob.NewDecoder(r)
	var currentTerm int
	var votedFor int
	var log []LogEntry
	if d.Decode(&currentTerm) != nil ||
		d.Decode(&votedFor) != nil ||
		d.Decode(&log) != nil {
		rf.DPrintf("------ decode error ------")
		rf.Kill()
	} else {
		rf.currentTerm = currentTerm
		rf.votedFor = votedFor
		rf.log = log
		rf.DPrintf("====== readPersist currentTerm: %d, votedFor: %d ======", rf.currentTerm, rf.votedFor)
		rf.printLog()
	}
}

//
// A service wants to switch to snapshot.  Only do so if Raft hasn't
// have more recent info since it communicate the snapshot on applyCh.
//
func (rf *Raft) CondInstallSnapshot(lastIncludedTerm int, lastIncludedIndex int, snapshot []byte) bool {

	// Your code here (2D).

	return true
}

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (2D).

}

//
// example RequestVote RPC handler.
//
func (rf *Raft) RequestVote(request *RequestVoteArgs, response *RequestVoteReply) {
	rf.Lock("RequestVote")
	defer rf.Unlock("RequestVote")

	rf.checkTerm(request.CandidateId, request.Term)

	// 对端任期小或者本端已经投票过了，那么拒绝投票
	if request.Term < rf.currentTerm || (request.Term == rf.currentTerm && rf.votedFor != -1 && rf.votedFor != request.CandidateId) {
		rf.DPrintf("already vote for %d, refuse to vote for %d", rf.votedFor, request.CandidateId)
		response.Term, response.VoteGranted = rf.currentTerm, false
		return
	}

	// 本地日志要更新一些，拒绝投票
	lastLog := rf.getLastLog()
	if lastLog.Term > request.LastLogTerm || (lastLog.Term == request.LastLogTerm && lastLog.CommandIndex > request.LastLogIndex) {
		rf.DPrintf("local log is newer than %d, refuse to vote", request.CandidateId)
		response.Term, response.VoteGranted = rf.currentTerm, false
		return
	}

	// 投票，重复回复也没事，TCP 会帮你处理掉的
	rf.DPrintf("vote for %d", request.CandidateId)
	// 既然要投票给别人，那自己肯定就不竞选了
	rf.electionTimer.Reset(rf.ElectionTimeout())
	rf.role = Follower
	rf.votedFor = request.CandidateId
	rf.persist()
	response.Term, response.VoteGranted = rf.currentTerm, true
}

func (rf *Raft) startElection() {
	rf.DPrintf("====== start candidate ======")
	lastLog := rf.getLastLog()
	request := &RequestVoteArgs{
		Term:         rf.currentTerm,
		CandidateId:  rf.me,
		LastLogIndex: lastLog.CommandIndex,
		LastLogTerm:  lastLog.Term,
	}
	grantedVotes := 1
	quotaNum := len(rf.peers)/2 + 1
	rf.votedFor = rf.me
	rf.persist()

	for peer := range rf.peers {
		if peer == rf.me {
			continue
		}
		go func(peer int) {
			response := new(RequestVoteReply)
			rf.DPrintf("send RequestVote %+v to %d", request, peer)
			if rf.sendRequestVote(peer, request, response) {
				rf.DPrintf("receive RequestVote from %d, response is %+v", peer, response)
				rf.Lock("sendRequestVote")
				defer rf.Unlock("sendRequestVote")
				// 过期轮次的回复直接丢弃
				if request.Term < rf.currentTerm {
					return
				}

				rf.checkTerm(peer, response.Term)

				// 已经不是竞选者角色了也不用理会回复
				if rf.role != Candidate {
					return
				}

				if response.VoteGranted {
					// 获得选票
					grantedVotes += 1
					if grantedVotes >= quotaNum {
						// 竞选成功
						rf.DPrintf("====== candidate success ======")
						rf.role = Leader
						rf.electionTimer.Reset(rf.ElectionTimeout())
						// 每次选举后重新初始化
						for i := 0; i < len(rf.peers); i++ {
							rf.nextIndex[i] = len(rf.log)
							rf.matchIndex[i] = 0
						}
						rf.broadcastHeartbeat(true)
					}
				}
			} else {
				rf.DPrintf("RequestVote RPC failed")
			}
		}(peer)
	}
}

//
// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
//
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := false

	// Your code here (2B).
	rf.Lock("Start")
	defer rf.Unlock("Start")
	if rf.role == Leader {
		rf.DPrintf("====== start cmd %+v ======", command)
		// 添加本地日志
		log := LogEntry{
			Term:         rf.currentTerm,
			CommandIndex: len(rf.log), // 初始有效索引为 1
			Command:      command,
		}
		rf.log = append(rf.log, log)
		rf.persist()
		rf.plug += 1
		if rf.plug >= PlugNumber {
			rf.plug = 0
			rf.broadcastHeartbeat(false)
		}
		// 后续请求结果会异步发送到 applyCh，index 就是 key
		index = log.CommandIndex
		term = log.Term
		isLeader = true
	}

	return index, term, isLeader
}

//
// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
//
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
	rf.Lock("Kill")
	defer rf.Unlock("Kill")
	rf.persist()
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

// 更新状态机
func (rf *Raft) applyCommitLog() {
	for rf.killed() == false {
		time.Sleep(time.Millisecond)
		if rf.lastApplied < rf.commitIndex {
			rf.Lock("applyCommitLog")
			for idx := rf.lastApplied + 1; idx <= rf.commitIndex; idx++ {
				select {
				case rf.applyCh <- ApplyMsg{
					CommandValid: true,
					Command:      rf.log[idx].Command,
					CommandIndex: rf.log[idx].CommandIndex,
				}:
					rf.DPrintf("====== apply committed log %d ======", idx)
					rf.lastApplied = idx
				default:
					goto Unlock
				}
			}
		Unlock:
			rf.Unlock("applyCommitLog")
		}
	}
}

// 判断索引是否可以提交
func (rf *Raft) commitCheck(commitIndex int) bool {
	quotaNum := len(rf.peers)/2 + 1
	n := 0
	for idx, matchIndex := range rf.matchIndex {
		if idx == rf.me {
			n += 1
		} else if matchIndex >= commitIndex {
			n += 1
		}
		if n >= quotaNum {
			return true
		}
	}

	return false
}

func (rf *Raft) commitLog() {
	low := rf.commitIndex + 1
	high := len(rf.log) - 1
	if low > high {
		return
	}
	// 只能提交当前任期的日志
	for i := high; i >= low && rf.log[i].Term == rf.currentTerm; i-- {
		if rf.commitCheck(i) {
			rf.commitIndex = i
			rf.DPrintf("====== commit log %d ======", i)
			return
		}
	}

	return
}

// The ticker go routine starts a new election if this peer hasn't received
// heartsbeats recently.
func (rf *Raft) ticker() {
	for rf.killed() == false {
		select {
		case <-rf.electionTimer.C:
			rf.Lock("electionTimer")
			// 开始竞选，任期加一
			rf.role = Candidate
			rf.currentTerm += 1
			rf.votedFor = -1
			rf.persist()
			rf.startElection()
			rf.electionTimer.Reset(rf.ElectionTimeout())
			rf.Unlock("electionTimer")
		case <-rf.heartbeatTimer.C:
			rf.Lock("heartbeatTimer")
			if rf.role == Leader {
				// 更新提交索引
				rf.commitLog()
				// Leader 定期发送心跳
				rf.broadcastHeartbeat(false)
				rf.electionTimer.Reset(rf.ElectionTimeout())
			}
			rf.heartbeatTimer.Reset(rf.HeartbeatTimeout())
			rf.Unlock("heartbeatTimer")
		}
	}
}

//
// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
//
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{
		peers:       peers,
		persister:   persister,
		me:          me,
		dead:        0,
		role:        Follower,
		currentTerm: 0,
		votedFor:    -1,
		log:         make([]LogEntry, 0),
		commitIndex: 0,
		lastApplied: 0,
		nextIndex:   make([]int, len(peers)),
		matchIndex:  make([]int, len(peers)),
		applyCh:     applyCh,
		plug:        0,
	}
	rf.log = append(rf.log, LogEntry{})
	rf.heartbeatTimer = time.NewTimer(rf.HeartbeatTimeout())
	rf.electionTimer = time.NewTimer(rf.ElectionTimeout())

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	// start ticker goroutine to start elections
	go rf.ticker()

	go rf.applyCommitLog()

	return rf
}
