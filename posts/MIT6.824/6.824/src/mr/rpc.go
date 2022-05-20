package mr

//
// RPC definitions.
//
// remember to capitalize all names.
//

import (
	"os"
	"time"
)
import "strconv"

//
// example to show how to declare the arguments
// and reply for an RPC.
//

type ExampleArgs struct {
	X int
}

type ExampleReply struct {
	Y int
}

// Add your RPC definitions here.
type WorkerStatus int
type JobType int
type TaskID int
type File string

const (
	Idle WorkerStatus = iota
	InProgress
	Completed
)

const (
	MapJob JobType = iota
	ReduceJob
	HeartBeatResp
	FinishJob
)

type Task struct {
	JobType      JobType
	ID           TaskID
	Input        []File
	OutPut       []File
	StartTime    time.Time
	ReduceNumber int
}

type HeartBeatArgs struct {
	Status WorkerStatus
	Task   Task
}

type HeartBeatReply struct {
	Task Task
}

// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the coordinator.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func coordinatorSock() string {
	s := "/var/tmp/824-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}
