package raft

type InstallSnapshotArgs struct {
	Term             int    // leader 任期
	LeaderId         int    // 用来 follower 把客户端请求重定向到 leader
	LastIncludeIndex int    // 快照中包含的最后日志条目的索引值
	LastIncludeTerm  int    // 快照中包含的最后日志条目的任期号
	offset           int    //分块在快照中的字节偏移量
	data             []byte // 从偏移量开始的快照分块的原始字节
	done             bool   // 如果这是最后一个分块则为 true
}

type InstallSnapshotReply struct {
	Term int // 当前任期
}

func (rf *Raft) sendInstallSnapshot(server int, args *InstallSnapshotArgs, reply *InstallSnapshotReply) bool {
	ok := rf.peers[server].Call("Raft.InstallSnapshot", args, reply)
	return ok
}

type AppendEntriesArgs struct {
	Term         int        // leader 任期
	LeaderId     int        // 用来 follower 把客户端请求重定向到 leader
	PrevLogIndex int        // 紧邻新日志条目之前的那个日志条目的索引
	PrevLogTerm  int        // 紧邻新日志条目之前的那个日志条目的任期
	Entries      []LogEntry // 日志
	LeaderCommit int        // leader 的 commitIndex
}

type AppendEntriesReply struct {
	Term          int  // 当前任期
	Success       bool // 如果包含索引为 prevLogIndex 和任期为 prevLogItem 的日志，则为 true
	ConflictTerm  int  // 告诉 Leader 下次检查的 ConflictTerm
	ConflictIndex int  // ConflictTerm 任期内最大的 Index
}

func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}

//
// example RequestVote RPC arguments structure.
// field names must start with capital letters!
//
type RequestVoteArgs struct {
	Term         int // 候选者的任期号
	CandidateId  int // 请求选票的候选者的 ID
	LastLogIndex int // 候选者的最后日志条目的索引值
	LastLogTerm  int // 候选者最后日志条目的任期号
}

//
// example RequestVote RPC reply structure.
// field names must start with capital letters!
//
type RequestVoteReply struct {
	Term        int  // 当前任期号，以便于候选者去更新自己的任期号
	VoteGranted bool // 候选者赢得了此张选票时为 true
}

//
// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
//
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}
