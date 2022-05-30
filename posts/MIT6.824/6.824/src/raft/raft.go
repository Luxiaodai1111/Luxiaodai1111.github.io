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

// 根据绝对索引获取相对索引处的日志
func (rf *Raft) log(index int) LogEntry {
	return rf.logs[rf.index(index)]
}

// 根据绝对索引获取相对索引
func (rf *Raft) index(index int) int {
	//rf.DPrintf("index : %d, rf.logs[0].CommandIndex: %d", index, rf.logs[0].CommandIndex)
	return index - rf.logs[0].CommandIndex
}

func (rf *Raft) getLastLog() LogEntry {
	return rf.logs[len(rf.logs)-1]
}

// 获取 term 任期内的第一条日志，这里保证 term 任期内一定有日志
func (rf *Raft) getConflictIndex(term int) (ConflictIndex int) {
	low := 0
	high := len(rf.logs)
	middle := (low + high) / 2
	// 二分先找到一条包含该任期的日志
	for ; low < high; middle = (low + high) / 2 {
		if rf.logs[middle].Term == term {
			break
		} else if rf.logs[middle].Term < term {
			low = middle + 1
		} else {
			high = middle - 1
		}
	}

	for i := middle; i >= 0; i-- {
		if rf.logs[i].Term != term {
			rf.DPrintf("====== getFirstLog in term %d: %d ======", term, rf.logs[i].CommandIndex+1)
			return rf.logs[i].CommandIndex + 1
		}
	}

	rf.DPrintf("====== getFirstLog in term %d: %d ======", term, rf.logs[0].CommandIndex)
	return rf.logs[0].CommandIndex
}

// 检查日志是否匹配
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

func (rf *Raft) AppendEntries(request *AppendEntriesArgs, response *AppendEntriesReply) {
	rf.Lock("AppendEntries")
	defer rf.Unlock("AppendEntries")
	if request.Term < rf.currentTerm {
		response.Success = false
		response.Term = rf.currentTerm
		rf.DPrintf("refuse AppendEntries from %d", request.LeaderId)
		return
	}

	rf.checkTerm(request.LeaderId, request.Term)
	response.Term = rf.currentTerm

	rf.printLog()
	rf.DPrintf("commitIndex: %d, lastApplied: %d", rf.commitIndex, rf.lastApplied)

	// 检查日志
	rf.DPrintf("PrevLogIndex: %d, PrevLogTerm: %d, LeaderCommit: %d", request.PrevLogIndex, request.PrevLogTerm, request.LeaderCommit)
	if rf.checkLogMatch(request.PrevLogIndex, request.PrevLogTerm) {
		// 日志匹配，追加日志
		if len(request.Entries) > 0 {
			// 本地日志要更新一些，拒绝接收
			lastLog := rf.getLastLog()
			lastEntry := request.Entries[len(request.Entries)-1]
			if lastLog.Term > lastEntry.Term || (lastLog.Term == lastEntry.Term && lastLog.CommandIndex > lastEntry.CommandIndex) {
				// TODO：让对端变成 follower
				rf.DPrintf("local log is newer than %d, refuse to recv log", request.LeaderId)
				response.Success = false
				response.ConflictTerm = request.PrevLogTerm
				response.ConflictIndex = request.PrevLogIndex
				return
			}
			// 追加日志
			if lastEntry.CommandIndex > rf.commitIndex {
				// 不要重复追加日志
				for idx, entry := range request.Entries {
					if lastLog.CommandIndex < entry.CommandIndex ||
						(rf.logs[0].CommandIndex <= entry.CommandIndex && entry.Term != rf.log(entry.CommandIndex).Term) {
						rf.logs = append(rf.logs[:rf.index(entry.CommandIndex)], request.Entries[idx:]...)
						rf.persist()
						rf.DPrintf("====== append log %d-%d ======", entry.CommandIndex, lastEntry.CommandIndex)
						break
					}
				}
			}
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
		rf.ResetElectionTimeout()
	} else {
		rf.DPrintf("====== log mismatch ======")
		// 如果自己不存在索引、任期和 prevLogIndex、 prevLogItem 匹配的日志返回 false。
		response.Success = false
		rf.role = Follower
		// 如果存在一条日志索引和 prevLogIndex 相等，但是任期和 prevLogItem 不相同的日志，需要删除这条日志及所有后继日志。
		lastLog := rf.getLastLog()
		if rf.logs[0].CommandIndex <= request.PrevLogIndex && request.PrevLogIndex <= lastLog.CommandIndex &&
			rf.log(request.PrevLogIndex).Term != request.Term {
			rf.DPrintf("====== delete log from %d ======", request.PrevLogIndex)
			rf.logs = rf.logs[:rf.index(request.PrevLogIndex)]
		}
		// 加速日志冲突检查, 获取不大于 request.PrevLogTerm 且包含日志的冲突条目
		lastLog = rf.getLastLog()
		for idx := lastLog.CommandIndex; idx >= 0; idx-- {
			if rf.log(idx).Term <= request.PrevLogTerm {
				response.ConflictTerm = rf.log(idx).Term
				response.ConflictIndex = rf.getConflictIndex(response.ConflictTerm)
				if response.ConflictIndex == request.PrevLogIndex && response.ConflictIndex > 0 {
					// 获取含有日志的上一个任期
					prevLog := rf.log(response.ConflictIndex - 1)
					response.ConflictTerm = prevLog.Term
					response.ConflictIndex = rf.getConflictIndex(response.ConflictTerm)
				}
				break
			}
		}
		// 即使日志冲突但是这里仍然是认可 Leader 的，所以也要重置竞选超时
		rf.ResetElectionTimeout()
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
	if !syncCommit {
		if rf.nextIndex[peer] < lastLog.CommandIndex+1 {
			// 存在待提交日志
			rf.DPrintf("peer %d's nextIndex is %d", peer, rf.nextIndex[peer])
			request.Entries = rf.logs[rf.index(rf.nextIndex[peer]):]
		}
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

			rf.DPrintf("peer %d's nextIndex is %d", peer, rf.nextIndex[peer])
		}
	} else {
		rf.DPrintf("send append entries RPC to %d failed", peer)
	}
}

func (rf *Raft) broadcast(syncCommit bool) {
	rf.plug = 0
	for peer := range rf.peers {
		if peer == rf.me {
			continue
		}
		go rf.replicate(peer, syncCommit)
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
	logs        []LogEntry // 日志条目

	commitIndex int // 已提交的最高的日志条目的索引
	lastApplied int // 已经被提交到状态机的最后一个日志的索引

	nextIndex  []int // 对于每一台服务器，下条发送到该机器的日志索引
	matchIndex []int // 对于每一台服务器，已经复制到该服务器的最高日志条目的索引

	electionTimeout time.Time   // go timer reset bugfix
	electionTimer   *time.Timer // 选举计时器

	applyCh           chan ApplyMsg
	internalApplyList []ApplyMsg // 内部 apply 队列
	plug              int        // 心跳时间内积攒一定数目的日志再一起发送
}

const HeartbeatTimeoutBase = 100

func (rf *Raft) HeartbeatTimeout() time.Duration {
	return time.Millisecond * HeartbeatTimeoutBase
}

const ElectionTimeoutBase = 500

func (rf *Raft) ElectionTimeout() time.Duration {
	//rand.Seed(time.Now().Unix() + int64(rf.me))
	return time.Millisecond * time.Duration(ElectionTimeoutBase+rand.Int63n(ElectionTimeoutBase))
}

// 如果明确 timer 已经expired，并且 t.C 已经被取空，那么可以直接使用 Reset；
// 如果程序之前没有从 t.C 中读取过值，这时需要首先调用 Stop()，
// 如果返回 true，说明 timer 还没有 expire，stop 成功删除 timer，可直接 reset；
// 如果返回 false，说明 stop 前已经 expire，需要显式 drain channel。
func (rf *Raft) ResetElectionTimeout() {
	rf.DPrintf("Reset ElectionTimeout")
	if !rf.electionTimer.Stop() {
		// 利用一个 select 来包裹 channel drain，这样无论 channel 中是否有数据，drain 都不会阻塞住
		select {
		case <-rf.electionTimer.C:
		default:
		}
	}
	rf.electionTimer.Reset(rf.ElectionTimeout())
	rf.electionTimeout = time.Now().Add(time.Millisecond * ElectionTimeoutBase)
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
		e.Encode(rf.logs) != nil {
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
	var logs []LogEntry
	if d.Decode(&currentTerm) != nil ||
		d.Decode(&votedFor) != nil ||
		d.Decode(&logs) != nil {
		rf.DPrintf("------ decode error ------")
		rf.Kill()
	} else {
		rf.currentTerm = currentTerm
		rf.votedFor = votedFor
		rf.logs = logs
		// logs 第一条日志一定是已经提交和应用了的
		rf.lastApplied = rf.logs[0].CommandIndex
		rf.commitIndex = rf.lastApplied
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
	rf.DPrintf("CondInstallSnapshot")

	return true
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

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
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
	rf.role = Follower
	rf.votedFor = request.CandidateId
	rf.persist()
	response.Term, response.VoteGranted = rf.currentTerm, true
	rf.ResetElectionTimeout()
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
			rf.Lock("sendRequestVote")
			if rf.role != Candidate {
				rf.Unlock("sendRequestVote")
				rf.DPrintf("now is not candidate")
				return
			}
			rf.Unlock("sendRequestVote")

			response := new(RequestVoteReply)
			rf.DPrintf("send RequestVote %+v to %d", request, peer)
			if rf.sendRequestVote(peer, request, response) {
				rf.DPrintf("receive RequestVote from %d, response is %+v", peer, response)
				rf.Lock("recvRequestVote")
				defer rf.Unlock("recvRequestVote")

				// 过期轮次的回复直接丢弃
				if request.Term < rf.currentTerm {
					return
				}

				rf.checkTerm(peer, response.Term)

				// 已经不是竞选者角色了也不用理会回复
				if rf.role != Candidate {
					rf.DPrintf("now is not candidate")
					return
				}

				if response.VoteGranted {
					// 获得选票
					grantedVotes += 1
					if grantedVotes >= quotaNum {
						// 竞选成功
						rf.DPrintf("====== candidate success ======")
						rf.role = Leader
						rf.ResetElectionTimeout()
						// 每次选举后重新初始化
						for i := 0; i < len(rf.peers); i++ {
							rf.nextIndex[i] = rf.getLastLog().CommandIndex + 1
							rf.matchIndex[i] = 0
						}
						// 这里应该要提交一条空日志，但是 2B 测试通不过，所以改为发送最后一条日志来同步提交
						rf.broadcast(true)
					}
				}
			} else {
				rf.DPrintf("RequestVote RPC to %d failed", peer)
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
		isLeader = true
		rf.DPrintf("====== start cmd %+v ======", command)
		// 添加本地日志，后续请求结果会异步发送到 applyCh，index 就是 key
		index = rf.getLastLog().CommandIndex + 1
		term = rf.currentTerm
		log := LogEntry{
			Term:         term,
			CommandIndex: index, // 初始有效索引为 1
			Command:      command,
		}
		rf.logs = append(rf.logs, log)
		rf.persist()
		// 请求达到一定数目再一起发送
		rf.plug += 1
		if rf.plug >= PlugNumber {
			rf.plug = 0
			rf.broadcast(false)
		}
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
	high := rf.getLastLog().CommandIndex
	if low > high {
		return
	}
	// 只能提交当前任期的日志，但由于测试没法提交空日志，所以这里有 liveness 的问题
	for i := high; i >= low && rf.log(i).Term == rf.currentTerm; i-- {
		if rf.commitCheck(i) {
			rf.commitIndex = i
			rf.DPrintf("====== commit log %d ======", i)
			return
		}
	}

	return
}

func (rf *Raft) heartbeat() {
	for rf.killed() == false {
		time.Sleep(rf.HeartbeatTimeout())
		rf.Lock("heartbeatTimer")
		if rf.role == Leader {
			// 更新提交索引
			rf.commitLog()
			// Leader 定期发送心跳
			rf.broadcast(false)
			rf.ResetElectionTimeout()
		}
		rf.Unlock("heartbeatTimer")
	}
}

// The ticker go routine starts a new election if this peer hasn't received
// heartsbeats recently.
func (rf *Raft) election() {
	rf.ResetElectionTimeout()
	for rf.killed() == false {
		select {
		case <-rf.electionTimer.C:
			rf.Lock("electionTimer")
			if time.Now().Before(rf.electionTimeout) {
				// go timer reset 的 bug：
				// 如果 sendTime 的执行发生在 drain channel 执行后，那么问题就来了，
				// 虽然 Stop 返回 false（因为 timer 已经 expire），但 drain channel 并没有读出任何数据。
				// 之后，sendTime 将数据发到 channel 中。timer Reset 后的 Timer 中的 Channel 实际上已经有了数据
				// 所以我增加了 electionTime 字段，如果与重置时间距离过短，那么就重置这个超时
				rf.DPrintf("go timer reset bugfix")
				rf.ResetElectionTimeout()
				rf.Unlock("electionTimer")
				continue
			}
			// 开始竞选，任期加一
			rf.DPrintf("====== election timeout ======")
			rf.role = Candidate
			rf.currentTerm += 1
			rf.ResetElectionTimeout()
			rf.startElection()
			rf.Unlock("electionTimer")
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
		peers:             peers,
		persister:         persister,
		me:                me,
		dead:              0,
		role:              Follower,
		currentTerm:       0,
		votedFor:          -1,
		logs:              make([]LogEntry, 0),
		commitIndex:       0,
		lastApplied:       0,
		nextIndex:         make([]int, len(peers)),
		matchIndex:        make([]int, len(peers)),
		applyCh:           applyCh,
		internalApplyList: make([]ApplyMsg, 0),
		plug:              0,
	}
	rf.logs = append(rf.logs, LogEntry{})
	rf.electionTimer = time.NewTimer(rf.ElectionTimeout())

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	// start ticker goroutine to start elections
	go rf.election()
	go rf.heartbeat()

	go rf.apply()

	return rf
}
