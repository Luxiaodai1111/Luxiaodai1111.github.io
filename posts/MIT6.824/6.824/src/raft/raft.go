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

type LogEntry struct {
	Term         int         // 日志所属的任期
	CommandIndex int         // 日志槽位
	Command      interface{} // 状态机命令
}

func (rf *Raft) getLastLog() LogEntry {
	return rf.log[len(rf.log)-1]
}

func (rf *Raft) AppendEntries(request *AppendEntriesArgs, response *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if request.Term >= rf.currentTerm {
		rf.role = Follower
		rf.currentTerm = request.Term
		rf.votedFor = -1
		rf.electionTimer.Reset(rf.ElectionTimeout())
		response.Success = true
	} else {
		response.Success = false
	}

	response.Term = rf.currentTerm

	rf.DPrintf("log %+v, commitIndex: %d, lastApplied: %d", rf.log, rf.commitIndex, rf.lastApplied)
	// 检查日志
	if request.Entries == nil ||
		(len(rf.log) > request.PrevLogIndex && rf.log[request.PrevLogIndex].Term == request.PrevLogTerm) {
		response.Success = true
		// 如果 leader 复制的日志本地没有，则直接追加存储。
		for i, entry := range request.Entries {
			if entry.CommandIndex == len(rf.log) {
				rf.log = append(rf.log, request.Entries[i:]...)
				rf.DPrintf("append log %+v", request.Entries[i:])
				break
			}
		}
		// 如果 leaderCommit>commitIndex，设置本地 commitIndex 为 leaderCommit 和最新日志索引中较小的一个。
		lastLog := rf.getLastLog()
		if request.LeaderCommit > rf.commitIndex {
			if lastLog.CommandIndex < request.LeaderCommit {
				rf.commitIndex = lastLog.CommandIndex
			} else {
				rf.commitIndex = request.LeaderCommit
			}
			rf.DPrintf("update commitIndex %d", rf.commitIndex)
		}
		rf.DPrintf("rf.log: %+v", rf.log)
	} else {
		rf.DPrintf("log unmatch")
		// 如果自己不存在索引、任期和 prevLogIndex、 prevLogItem 匹配的日志返回 false。
		response.Success = false
		// 如果存在一条日志索引和 prevLogIndex 相等，但是任期和 prevLogItem 不相同的日志，需要删除这条日志及所有后继日志。
		if len(rf.log) > request.PrevLogIndex && rf.log[request.PrevLogIndex].Term != request.Term {
			rf.DPrintf("delete log from %d", request.PrevLogIndex)
			rf.log = rf.log[:request.PrevLogIndex]
		}
	}

	return
}

// 如果接收到的 RPC 请求或响应中，任期号 T > currentTerm，则令 currentTerm = T，并切换为 follower 状态
func (rf *Raft) checkTerm(term int) {
	if term > rf.currentTerm {
		rf.DPrintf("peer term %d > currentTerm", term)
		rf.role = Follower
		rf.currentTerm = term
		rf.votedFor = -1
		rf.electionTimer.Reset(rf.ElectionTimeout())
	}
}

func (rf *Raft) sendLog(peer int, logs []LogEntry) {
	rf.mu.Lock()
	prevLog := rf.log[logs[0].CommandIndex-1]
	request := &AppendEntriesArgs{
		Term:         rf.currentTerm,
		LeaderId:     rf.me,
		PrevLogIndex: prevLog.CommandIndex,
		PrevLogTerm:  prevLog.Term,
		Entries:      logs, // 一次可以发送多条日志
		LeaderCommit: rf.commitIndex,
	}
	rf.mu.Unlock()

	response := new(AppendEntriesReply)
	rf.DPrintf("send log %+v to %d", request, peer)
	if rf.sendAppendEntries(peer, request, response) {
		rf.mu.Lock()
		defer rf.mu.Unlock()
		rf.checkTerm(response.Term)
		if rf.role != Leader {
			return
		}
		if response.Success {
			rf.matchIndex[peer] = logs[len(logs)-1].CommandIndex
			rf.nextIndex[peer] = rf.matchIndex[peer] + 1
			rf.DPrintf("peer %d's matchIndex is %d", peer, rf.matchIndex[peer])
		} else {
			rf.nextIndex[peer] -= 1
		}
	}

	return
}

func (rf *Raft) sendHeartbeat(peer int) {
	rf.mu.Lock()
	request := &AppendEntriesArgs{
		Term:         rf.currentTerm,
		LeaderId:     rf.me,
		PrevLogIndex: 0,
		PrevLogTerm:  0,
		Entries:      nil,
		LeaderCommit: rf.commitIndex,
	}
	rf.mu.Unlock()

	response := new(AppendEntriesReply)
	rf.DPrintf("send heartbeat %+v to %d", request, peer)
	if rf.sendAppendEntries(peer, request, response) {
		rf.DPrintf("receive AppendEntriesReply from %d, response is %+v", peer, response)
		rf.mu.Lock()
		defer rf.mu.Unlock()
		rf.checkTerm(response.Term)
	}
}

func (rf *Raft) broadcastHeartbeat() {
	for peer := range rf.peers {
		if peer == rf.me {
			continue
		}
		if rf.nextIndex[peer] < len(rf.log) {
			rf.DPrintf("peer %d's nextIndex is %d, rf.log: %+v", peer, rf.nextIndex[peer], rf.log)
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

	rf.mu.Lock()
	defer rf.mu.Unlock()
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
	// Your code here (2C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// data := w.Bytes()
	// rf.persister.SaveRaftState(data)
}

//
// restore previously persisted state.
//
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
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
	rf.mu.Lock()
	defer rf.mu.Unlock()

	// 对端任期小或者本端已经投票过了，那么拒绝投票
	if request.Term < rf.currentTerm || (request.Term == rf.currentTerm && rf.votedFor != -1 && rf.votedFor != request.CandidateId) {
		response.Term, response.VoteGranted = rf.currentTerm, false
		return
	}

	// 本地日志要更新一些，拒绝投票
	lastLog := rf.getLastLog()
	if lastLog.Term > request.LastLogTerm || (lastLog.Term == request.LastLogTerm && lastLog.CommandIndex > request.LastLogIndex) {
		response.Term, response.VoteGranted = rf.currentTerm, false
		return
	}

	rf.checkTerm(request.Term)
	// 投票，重复回复也没事，TCP 会帮你处理掉的
	rf.DPrintf("vote for %d", request.CandidateId)
	rf.votedFor = request.CandidateId
	response.Term, response.VoteGranted = rf.currentTerm, true
}

func (rf *Raft) startElection() {
	rf.DPrintf("start candidate")
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

	for peer := range rf.peers {
		if peer == rf.me {
			continue
		}
		go func(peer int) {
			response := new(RequestVoteReply)
			rf.DPrintf("send RequestVote %+v to %d", request, peer)
			if rf.sendRequestVote(peer, request, response) {
				rf.DPrintf("receive RequestVote from %d, response is %+v", peer, response)
				rf.mu.Lock()
				defer rf.mu.Unlock()
				// 过期轮次的回复直接丢弃
				if request.Term < rf.currentTerm {
					return
				}
				// 已经不是竞选者角色了也不用理会回复
				if rf.role != Candidate {
					return
				}

				rf.checkTerm(response.Term)
				if response.VoteGranted {
					// 获得选票
					grantedVotes += 1
					if grantedVotes >= quotaNum {
						// 竞选成功
						rf.DPrintf("====== candidate success ======")
						rf.role = Leader
						rf.broadcastHeartbeat()
					}
				}
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
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if rf.role == Leader {
		rf.DPrintf("start cmd %+v", command)
		// 添加本地日志
		log := LogEntry{
			Term:         rf.currentTerm,
			CommandIndex: len(rf.log), // 初始有效索引为 1
			Command:      command,
		}
		rf.log = append(rf.log, log)
		rf.broadcastHeartbeat()
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
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

// 判断索引是否可以提交
func (rf *Raft) commitCheck(commitIndex int) bool {
	quotaNum := len(rf.peers)/2 + 1
	n := 0
	for _, matchIndex := range rf.matchIndex {
		if matchIndex >= commitIndex {
			n += 1
			if n >= quotaNum {
				return true
			}
		}
	}

	return false
}

func (rf *Raft) applyCommitLog() {
	for idx := rf.lastApplied + 1; idx <= rf.commitIndex; idx++ {
		select {
		case rf.applyCh <- ApplyMsg{
			CommandValid: true,
			Command:      rf.log[idx].Command,
			CommandIndex: rf.log[idx].CommandIndex,
		}:
			rf.DPrintf("apply commited log %d", idx)
			rf.lastApplied = idx
		default:
			break
		}
	}
}

func (rf *Raft) commitLog() {
	// 二分查找可以提交的索引
	low := rf.commitIndex + 1
	high := len(rf.log) - 1
	nextCommitIndex := (low + high) / 2
	for ; low <= high; nextCommitIndex = (low + high) / 2 {
		if rf.commitCheck(nextCommitIndex) {
			rf.commitIndex = nextCommitIndex
			rf.DPrintf("commit log %d", nextCommitIndex)
			break
		} else {
			high = nextCommitIndex
		}
	}
}

// The ticker go routine starts a new election if this peer hasn't received
// heartsbeats recently.
func (rf *Raft) ticker() {
	for rf.killed() == false {
		select {
		case <-rf.electionTimer.C:
			rf.mu.Lock()
			// 开始竞选，任期加一
			rf.role = Candidate
			rf.currentTerm += 1
			rf.startElection()
			rf.electionTimer.Reset(rf.ElectionTimeout())
			rf.mu.Unlock()
		case <-rf.heartbeatTimer.C:
			rf.mu.Lock()
			if rf.role == Leader {
				// 更新提交索引
				rf.commitLog()
				// Leader 定期发送心跳
				rf.broadcastHeartbeat()
				rf.electionTimer.Reset(rf.ElectionTimeout())
			}
			// 更新应用索引
			rf.applyCommitLog()
			rf.heartbeatTimer.Reset(rf.HeartbeatTimeout())
			rf.mu.Unlock()
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
		nextIndex:   make([]int, 0),
		matchIndex:  make([]int, len(peers)),
		applyCh:     applyCh,
	}
	for i := 0; i < len(peers); i++ {
		rf.nextIndex = append(rf.nextIndex, 1)
	}
	rf.log = append(rf.log, LogEntry{})
	rf.heartbeatTimer = time.NewTimer(rf.HeartbeatTimeout())
	rf.electionTimer = time.NewTimer(rf.ElectionTimeout())

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	// start ticker goroutine to start elections
	go rf.ticker()

	return rf
}
