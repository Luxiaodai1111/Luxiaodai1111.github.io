package raft

import (
	"fmt"
	"log"
)

// Debugging
const Debug = false

func (rf *Raft) DPrintf(format string, a ...interface{}) {
	if Debug {
		var role string
		if rf.role == Leader {
			role = "leader"
		} else if rf.role == Candidate {
			role = "candidate"
		} else {
			role = "follower"
		}
		log.Printf(fmt.Sprintf("[term %d][node %d][role %s]:%s", rf.currentTerm, rf.me, role, format), a...)
	}
	return
}

func (rf *Raft) printLog() {
	if Debug {
		logs := "log:"
		for _, l := range rf.log {
			logs += fmt.Sprintf("(%d, %d)", l.Term, l.CommandIndex)
		}
		logs += "\n"
		rf.DPrintf("%s", logs)
	}
}
