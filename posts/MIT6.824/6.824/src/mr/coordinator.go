package mr

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Coordinator struct {
	sync.Mutex
	nReduce          int
	MapTotal         int               // Map 任务总数
	MapTaskCh        chan *Task        // 待执行的 Map 任务全部塞到此通道
	MapProcessing    map[TaskID]*Task  // 所有正在执行的 Map 任务
	MapCompleted     map[TaskID][]File // 记录 Map 任务完成后的中间文件
	ReduceTaskCh     chan *Task        // 待执行的 Reduce 任务全部塞到此通道
	ReduceProcessing map[TaskID]*Task  // 所有正在执行的 Reduce 任务
	ReduceCompleted  map[TaskID][]File // 记录 Reduce 任务完成后的结果文件
	HeartBeatResp    Task
}

var debug = false

func Debug(a ...interface{}) {
	if debug {
		fmt.Println(a...)
	}
}

func (c *Coordinator) AddMapTask(files []string) {
	for idx, filename := range files {
		Debug("Add Map Task", idx)
		c.MapTaskCh <- &Task{
			JobType:      MapJob,
			ID:           TaskID(idx),
			Input:        []File{File(filename)},
			ReduceNumber: c.nReduce,
		}
	}

	return
}

func (c *Coordinator) AddReduceTask() {
	for {
		if c.MapTasksIsCompleted() {
			// 对中间文件进行整理，具有同样尾号的文件发给一个 Reduce worker 处理
			reduceTasks := make(map[int][]File, c.nReduce)
			for _, intermediateFiles := range c.MapCompleted {
				for _, filename := range intermediateFiles {
					info := strings.Split(string(filename), "-")
					idx, _ := strconv.Atoi(info[len(info)-1])
					reduceTasks[idx] = append(reduceTasks[idx], filename)
				}
			}

			// 如果总的任务数小于指定 nReduce，更新 nReduce，
			// 否则会一直等不到任务结束
			if len(reduceTasks) < c.nReduce {
				c.nReduce = len(reduceTasks)
			}

			for idx, files := range reduceTasks {
				Debug("Add Reduce Task", idx)
				c.ReduceTaskCh <- &Task{
					JobType: ReduceJob,
					ID:      TaskID(idx),
					Input:   files,
				}
			}
			return
		} else {
			time.Sleep(time.Second)
		}
	}
}

// 判断 Map 任务是否全部执行完毕
func (c *Coordinator) MapTasksIsCompleted() bool {
	c.Lock()
	defer c.Unlock()
	if len(c.MapCompleted) == c.MapTotal {
		return true
	}
	return false
}

func (c *Coordinator) ReduceTasksIsCompleted() bool {
	c.Lock()
	defer c.Unlock()
	if len(c.ReduceCompleted) == c.nReduce {
		return true
	}
	return false
}

func (c *Coordinator) CheckTaskTimeout(taskProcessing map[TaskID]*Task, taskCh chan *Task) {
	c.Lock()
	// 当 taskCh 没有任务时再来检查是否有超时的 Job
	now := time.Now()
	deleteTask := make([]TaskID, 0)
	for idx, task := range taskProcessing {
		timeout := task.StartTime.Add(time.Second * 10)
		if now.After(timeout) {
			Debug("task", task.ID, "timeout!!!")
			select {
			case taskCh <- task:
				Debug("ReTry Task", task.ID)
				deleteTask = append(deleteTask, idx)
			default:
			}
		}
	}
	// 重新发送到队列里的 task 先从处理字典中清除
	for _, taskID := range deleteTask {
		delete(taskProcessing, taskID)
	}
	c.Unlock()
}

// Your code here -- RPC handlers for the worker to call.
func (c *Coordinator) HeartBeat(args *HeartBeatArgs, reply *HeartBeatReply) error {
	if args.Status == Idle {
		// 所有 reduce 任务完成，通知 worker 退出
		if c.ReduceTasksIsCompleted() {
			reply.Task = Task{JobType: FinishJob}
			Debug("Finish Job")
			return nil
		}
		if c.MapTasksIsCompleted() {
			// 执行 Reduce 任务
			select {
			case task := <-c.ReduceTaskCh:
				Debug("dispatch reduce task", task.ID)
				c.Lock()
				task.StartTime = time.Now()
				reply.Task = *(task)
				c.ReduceProcessing[task.ID] = task
				c.Unlock()
			default:
				c.CheckTaskTimeout(c.ReduceProcessing, c.ReduceTaskCh)
				reply.Task = c.HeartBeatResp
			}
		} else {
			// 执行 Map 任务
			select {
			case task := <-c.MapTaskCh:
				Debug("dispatch map task", task.ID)
				c.Lock()
				task.StartTime = time.Now()
				reply.Task = *(task)
				c.MapProcessing[task.ID] = task
				c.Unlock()
			default:
				c.CheckTaskTimeout(c.MapProcessing, c.MapTaskCh)
				reply.Task = c.HeartBeatResp
			}
		}
	} else if args.Status == InProgress {
		reply.Task = c.HeartBeatResp
	} else if args.Status == Completed {
		task := args.Task
		if args.Task.JobType == MapJob {
			Debug("MapJob task", task.ID, "Completed")
			//Debug("OutPut:", task.OutPut)
			c.Lock()
			delete(c.MapProcessing, task.ID)
			// 重复完成的任务可以不理会
			_, ok := c.MapCompleted[task.ID]
			if !ok {
				c.MapCompleted[task.ID] = task.OutPut
			}
			c.Unlock()
		} else if args.Task.JobType == ReduceJob {
			Debug("ReduceJob task", task.ID, "Completed")
			c.Lock()
			delete(c.ReduceProcessing, task.ID)
			_, ok := c.ReduceCompleted[task.ID]
			if !ok {
				c.ReduceCompleted[task.ID] = task.OutPut
			}
			c.Unlock()
		} else {
			return errors.New("JobType error")
		}
	} else {
		return errors.New("args Status error")
	}

	return nil
}

//
// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
//
func (c *Coordinator) Example(args *ExampleArgs, reply *ExampleReply) error {
	reply.Y = args.X + 1
	return nil
}

//
// start a thread that listens for RPCs from worker.go
//
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := coordinatorSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

//
// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
//
func (c *Coordinator) Done() bool {
	ret := false

	if c.ReduceTasksIsCompleted() {
		ret = true
		// 等待 worker 全部退出
		time.Sleep(time.Second * 3)
		Debug("master exit")
	}

	return ret
}

//
// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
//
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{
		nReduce:          nReduce,
		MapTotal:         len(files),
		MapTaskCh:        make(chan *Task, 10),
		MapProcessing:    make(map[TaskID]*Task),
		MapCompleted:     make(map[TaskID][]File),
		ReduceTaskCh:     make(chan *Task, 10),
		ReduceProcessing: make(map[TaskID]*Task),
		ReduceCompleted:  make(map[TaskID][]File),
		HeartBeatResp:    Task{JobType: HeartBeatResp},
	}

	// Your code here.
	go c.AddMapTask(files)
	go c.AddReduceTask()

	c.server()
	return &c
}
