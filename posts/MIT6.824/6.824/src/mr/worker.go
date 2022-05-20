package mr

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"sort"
	"sync"
	"time"
)
import "log"
import "net/rpc"
import "hash/fnv"

//
// Map functions return a slice of KeyValue.
//
type KeyValue struct {
	Key   string
	Value string
}

// for sorting by key.
type ByKey []KeyValue

// for sorting by key.
func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return a[i].Key < a[j].Key }

type worker struct {
	sync.Mutex
	status  WorkerStatus
	mapf    func(string, string) []KeyValue
	reducef func(string, []string) string
}

func (w *worker) Debug(a ...interface{}) {
	if debug {
		fmt.Println(a...)
	}
}

//
// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
//
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

func (w *worker) SetStatus(status WorkerStatus) {
	w.Lock()
	w.status = status
	w.Unlock()
}

func (w *worker) HeartBeat(args *HeartBeatArgs) (*HeartBeatReply, error) {
	if args == nil {
		w.Lock()
		status := w.status
		w.Unlock()
		if status == Completed {
			return nil, errors.New("other routine will handle heartbeat")
		}
		args = &HeartBeatArgs{
			Status: status,
		}
	}
	reply := &HeartBeatReply{}
	ok := call("Coordinator.HeartBeat", args, reply)
	if ok {
		return reply, nil
	} else {
		return nil, errors.New("call failed")
	}
}

func (w *worker) readFromLocalFile(task *Task) ([]KeyValue, error) {
	intermediate := []KeyValue{}
	for _, input := range task.Input {
		var kva []KeyValue
		filename := string(input)
		file, err := os.Open(filename)
		if err != nil {
			return nil, err
		}
		dec := json.NewDecoder(file)
		err = dec.Decode(&kva)
		if err != nil {
			file.Close()
			return nil, err
		}
		file.Close()
		if len(kva) > 0 {
			intermediate = append(intermediate, kva...)
		}
	}

	return intermediate, nil
}

func (w *worker) writeToLocalFile(kva []KeyValue, task *Task) error {
	// 对数据结果进行处理，分成 nReduce 个文件
	buffer := make([][]KeyValue, task.ReduceNumber)
	for _, intermediate := range kva {
		slot := ihash(intermediate.Key) % task.ReduceNumber
		buffer[slot] = append(buffer[slot], intermediate)
	}

	// 结果持久化
	task.OutPut = make([]File, 0)
	intermediateDir := "/tmp/intermediate"
	err := os.MkdirAll(intermediateDir, os.ModePerm)
	if err != nil {
		return err
	}
	for i := 0; i < task.ReduceNumber; i++ {
		intermediateFile := fmt.Sprintf("%s/Map-%d-%d", intermediateDir, task.ID, i)
		rand.Seed(time.Now().UnixNano())
		ramdom := rand.Int63()
		tmp := fmt.Sprintf("%s.%d", intermediateFile, ramdom)
		file, err := os.OpenFile(tmp, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.ModePerm)
		if err != nil {
			return err
		}
		enc := json.NewEncoder(file)
		err = enc.Encode(buffer[i])
		if err != nil {
			file.Close()
			os.Remove(tmp)
			return err
		}
		file.Close()
		// rename 原子操作防止 crash
		err = os.Rename(tmp, intermediateFile)
		if err != nil {
			os.Remove(tmp)
			return err
		}

		// 中间文件列表返回给 master
		task.OutPut = append(task.OutPut, File(intermediateFile))
	}

	return nil
}

func (w *worker) DoMapf(task *Task) {
	Debug("Do Map:", task.ID, task.Input)
	filename := string(task.Input[0])
	// 偷懒，不考虑文件读写异常情况，直接置为 Idle，
	// 至于任务超时后怎么处理，由 master 决定
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		Debug(err)
		w.SetStatus(Idle)
		return
	}

	// 执行 map 操作
	kva := w.mapf(filename, string(content))

	// 生成本地中间文件
	err = w.writeToLocalFile(kva, task)
	if err != nil {
		Debug(err)
		w.SetStatus(Idle)
		return
	}

	// 方便模拟故障
	if debug {
		time.Sleep(time.Second)
	}

	// 回复 master 任务完成
	w.SetStatus(Completed)
	args := &HeartBeatArgs{
		Status: w.status,
		Task:   *task,
	}
	_, _ = w.HeartBeat(args)
	// 任务完成，可以重新接收任务
	w.SetStatus(Idle)
}

func (w *worker) Map(task *Task) {
	w.SetStatus(InProgress)

	// map 是 IO 密集型操作，要异步操作
	go w.DoMapf(task)
}

func (w *worker) DoReducef(task *Task) {
	Debug("Do Reduce:", task.ID)

	intermediate, err := w.readFromLocalFile(task)
	if err != nil {
		Debug(err)
		w.SetStatus(Idle)
		return
	}

	// 排序
	sort.Sort(ByKey(intermediate))

	// 执行 reduce 操作
	rand.Seed(time.Now().UnixNano())
	ramdom := rand.Int63()
	oname := fmt.Sprintf("mr-out-%d", task.ID)
	otmp := fmt.Sprintf("%s.%d", oname, ramdom)
	task.OutPut = []File{File(oname)}
	ofile, err := os.Create(otmp)
	if err != nil {
		Debug(err)
		w.SetStatus(Idle)
		return
	}

	i := 0
	for i < len(intermediate) {
		j := i + 1
		for j < len(intermediate) && intermediate[j].Key == intermediate[i].Key {
			j++
		}
		values := []string{}
		for k := i; k < j; k++ {
			values = append(values, intermediate[k].Value)
		}
		output := w.reducef(intermediate[i].Key, values)

		// this is the correct format for each line of Reduce output.
		_, err = fmt.Fprintf(ofile, "%v %v\n", intermediate[i].Key, output)
		if err != nil {
			ofile.Close()
			os.Remove(otmp)
			Debug(err)
			w.SetStatus(Idle)
			return
		}

		i = j
	}

	err = ofile.Close()
	if err != nil {
		Debug(err)
		w.SetStatus(Idle)
		os.Remove(otmp)
		return
	}
	// 防止 crash 破坏文件
	err = os.Rename(otmp, oname)
	if err != nil {
		Debug(err)
		w.SetStatus(Idle)
		os.Remove(otmp)
		return
	}

	// 回复 master 任务完成
	w.SetStatus(Completed)
	args := &HeartBeatArgs{
		Status: w.status,
		Task:   *task,
	}
	_, _ = w.HeartBeat(args)
	// 任务完成，可以重新接收任务
	w.SetStatus(Idle)
}

func (w *worker) Reduce(task *Task) {
	w.SetStatus(InProgress)

	// reduce 是 IO 密集型操作，要异步操作
	go w.DoReducef(task)
}

//
// main/mrworker.go calls this function.
//
func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {

	// Your worker implementation here.
	w := worker{
		status:  Idle,
		mapf:    mapf,
		reducef: reducef,
	}

	for {
		time.Sleep(time.Second)
		reply, err := w.HeartBeat(nil)
		if err != nil {
			Debug("heart beat err:", err)
			continue
		}
		switch reply.Task.JobType {
		case MapJob:
			w.Map(&reply.Task)
		case ReduceJob:
			w.Reduce(&reply.Task)
		case HeartBeatResp:
			Debug("waiting...")
		case FinishJob:
			Debug("worker exit")
			return
		default:
		}
	}

	// uncomment to send the Example RPC to the coordinator.
	// CallExample()

}

//
// example function to show how to make an RPC call to the coordinator.
//
// the RPC argument and reply types are defined in rpc.go.
//
func CallExample() {

	// declare an argument structure.
	args := ExampleArgs{}

	// fill in the argument(s).
	args.X = 99

	// declare a reply structure.
	reply := ExampleReply{}

	// send the RPC request, wait for the reply.
	// the "Coordinator.Example" tells the
	// receiving server that we'd like to call
	// the Example() method of struct Coordinator.
	ok := call("Coordinator.Example", &args, &reply)
	if ok {
		// reply.Y should be 100.
		fmt.Printf("reply.Y %v\n", reply.Y)
	} else {
		fmt.Printf("call failed!\n")
	}
}

//
// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
//
func call(rpcname string, args interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := coordinatorSock()
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}
