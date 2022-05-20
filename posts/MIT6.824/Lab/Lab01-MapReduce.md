# 实验介绍

在本实验中，您将构建一个 MapReduce 系统。本实验用 coordinator 代替论文的 master。



---

# Getting started

首先你需要安装一个 GO 环境，也要有 Git 的一点点基础，把 Lab 代码先克隆下来。

```bash
$ git clone git://g.csail.mit.edu/6.824-golabs-2022 6.824
$ cd 6.824
$ ls
Makefile src
```

我们在 `src/main/mrsequential.go` 中为您提供了一个简单的顺序 mapreduce 实现。我们还为您提供了几个 MapReduce 应用程序：`mrapps/wc.go` 中的 word-count，以及 `mrapps/indexer.go` 中的一个文本索引器。你可以试着先把代码跑起来看看效果：

```bash
$ cd ~/6.824
$ cd src/main
$ go build -race -buildmode=plugin ../mrapps/wc.go
$ rm mr-out*
$ go run -race mrsequential.go wc.so pg*.txt
$ more mr-out-0
A 509
ABOUT 2
ACT 8
...
```

>[!NOTE]
>
>-race 启用 Go race 检测器。我们建议你用 race 检测器开发和测试你的 6.824 实验代码。当我们给你的实验评分时，我们不会使用 race。然而，如果你的代码有竞争，即使没有竞争检测器，当我们测试它时，它也很有可能失败。

mrsequential.go 将其输出保存在文件 mr-out-0 中。输入来自名为 pg-xxx.txt 的文本文件。

可以随意借用 mrsequential.go 的代码，你也应该看看 mrapps/wc.go，看看 MapReduce 应用程序代码是什么样子的。

我们先来看一下 mrapps/wc.go 里 Map 和 Reduce 的实现。Map 函数很简单，就是把文件内容按单词分割，每个单词都由 `mr.KeyValue` 结构表示出现了一次，把所有的 KV 结果组成列表返回。Reduce 负责统计，一个 key 有多少条记录那么就是出现了多少次。

```go
func Map(filename string, contents string) []mr.KeyValue {
   // function to detect word separators.
   ff := func(r rune) bool { return !unicode.IsLetter(r) }

   // split contents into an array of words.
   words := strings.FieldsFunc(contents, ff)

   kva := []mr.KeyValue{}
   for _, w := range words {
      kv := mr.KeyValue{w, "1"}
      kva = append(kva, kv)
   }
   return kva
}

func Reduce(key string, values []string) string {
   // return the number of occurrences of this word.
   return strconv.Itoa(len(values))
}
```

mrsequential.go 的 main 函数则是便利每个文件进行 Map 操作，所有结果记录在 intermediate，里面保存了所有的 KV 对，Map 结束后，对结果 intermediate 进行排序，排序之后相同 Key 的结果就在列表中连续了，这个时候对每个 Key 进行 Reduce 操作，将结果追加放在 mr-out-0 文件里。

```go
func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: mrsequential xxx.so inputfiles...\n")
		os.Exit(1)
	}

	mapf, reducef := loadPlugin(os.Args[1])

	//
	// read each input file,
	// pass it to Map,
	// accumulate the intermediate Map output.
	//
	intermediate := []mr.KeyValue{}
	for _, filename := range os.Args[2:] {
         // 打开文件
		file, err := os.Open(filename)
		if err != nil {
			log.Fatalf("cannot open %v", filename)
		}
         // 读取文件内容
		content, err := ioutil.ReadAll(file)
		if err != nil {
			log.Fatalf("cannot read %v", filename)
		}
		file.Close()
         // Map 操作
		kva := mapf(filename, string(content))
         // KV 结果放入 intermediate
		intermediate = append(intermediate, kva...)
	}

	//
	// a big difference from real MapReduce is that all the
	// intermediate data is in one place, intermediate[],
	// rather than being partitioned into NxM buckets.
	//
	// 排序
	sort.Sort(ByKey(intermediate))

	oname := "mr-out-0"
	ofile, _ := os.Create(oname)

	//
	// call Reduce on each distinct key in intermediate[],
	// and print the result to mr-out-0.
	//
	i := 0
	for i < len(intermediate) {
		j := i + 1
         // 找相同 key 的索引范围
		for j < len(intermediate) && intermediate[j].Key == intermediate[i].Key {
			j++
		}
		values := []string{}
		for k := i; k < j; k++ {
			values = append(values, intermediate[k].Value)
		}
         // Reduce 操作
		output := reducef(intermediate[i].Key, values)

		// this is the correct format for each line of Reduce output.
		fmt.Fprintf(ofile, "%v %v\n", intermediate[i].Key, output)

		i = j
	}

	ofile.Close()
}
```





---

# 目标

你的目标是实现一个分布式 MapReduce，由两个程序组成，coordinator 和 worker。只有一个 coordinator 进程和一个或多个并行执行的 worker 进程。在真实的系统中，工作人员会在一系列不同的机器上运行，但在本实验中，您将在一台机器上运行他们。worker 之间将通过 RPC 与 coordinator 对话。每个 worker 将向 coordinator 请求一个任务，从一个或多个文件中读取任务的输入，执行任务，并将任务的输出写入一个或多个文件。coordinator 应该注意到 worker 是否在合理的时间内(对于本实验，使用 10 秒)没有完成其任务，并将相同的任务分配给另一个 worker。

coordinator 和 worker 的主例程在 `main/mrcoordinator.go` 和 `main/mrworker.go` 中；不要更改这些文件。您应该将您的实现放在 mr/coordinator.go、mr/worker.go 和 mr/rpc.go 中。

以下演示如何在字数统计 MapReduce 应用程序上运行你的代码。首先，确保字数统计插件是新构建的:

```bash
$ go build -race -buildmode=plugin ../mrapps/wc.go
```

在 main 目录中，运行 coordinator。

```bash
$ rm mr-out*
$ go run -race mrcoordinator.go pg-*.txt
```

pg-*.txt 是输入文件，每个文件对应于一个"分割"，并且是一个 Map 任务的输入。然后运行一个或多个 worker 进程：

```bash
$ go run -race mrworker.go wc.so
```

当 worker 和 coordinator 完成后，查看 mr-out-* 中的输出。完成实验后，输出文件应该按顺序输出，如下所示：

```bash
$ cat mr-out-* | sort | more
A 509
ABOUT 2
ACT 8
...
```

我们在 `main/test-mr.sh` 中为您提供了一个测试脚本。该测试检查当给定 pg-xxx.txt 文件作为输入时，wc 和 indexer MapReduce 应用程序是否产生正确的输出。测试还检查您的实现是否并行运行 Map 和 Reduce 任务，以及您的实现是否从运行任务时崩溃的 workers 中恢复。

如果您现在运行测试脚本，它将会挂起，因为 coordinator 永远不会结束：

```bash
$ cd ~/6.824/src/main
$ bash test-mr.sh
*** Starting wc test.
```

您可以在 mr/coordinator.go 中的 Done 函数中将 `ret := false` 更改为 true，以便 coordinator 立即退出。

```bash
$ bash test-mr.sh
*** Starting wc test.
sort: No such file or directory
cmp: EOF on mr-wc-all
--- wc output is not the same as mr-correct-wc.txt
--- wc test: FAIL
$
```

测试脚本期望在名为 mr-out-X 的文件中看到输出，每个 reduce 任务一个。mr/coordinator.go 和 mr/worker.go 的空实现不会产生这些文件(或者做很多其他事情)，所以测试失败。

当你完成实验后，测试脚本的输出应该如下所示：

```bash
$ bash test-mr.sh
*** Starting wc test.
--- wc test: PASS
*** Starting indexer test.
--- indexer test: PASS
*** Starting map parallelism test.
--- map parallelism test: PASS
*** Starting reduce parallelism test.
--- reduce parallelism test: PASS
*** Starting crash test.
--- crash test: PASS
*** PASSED ALL TESTS
$
```

您还会看到 Go RPC 包中的一些错误，如下所示：

```bash
2019/12/16 13:27:09 rpc.Register: method "Done" has 1 input parameters; needs exactly three
```

忽略这些消息；将 coordinator 注册为 RPC 服务器会检查它的所有方法是否都适合 RPC（有 3 个输入）；我们知道 Done 不是通过 RPC 调用的。



---

# 一些规则

- map 阶段应该将中间结果分成用于 nReduce 个 Reduce 任务的桶，其中 nReduce 是 reduce 任务的数量—— main/mrcoordinator.go 传递给 MakeCoordinator() 的参数。因此，每个 Map 任务需要创建 nReduce  个中间文件，供 reduce 任务使用。
- worker 实现应该将第 X 个 reduce 任务的输出放在 mr-out-X 文件中。
- mr-out-X 文件应该包含每个 Reduce 函数输出的一行。这一行应该使用 Go "%v %v " 格式生成，用键和值调用。如果您的实现偏离这种格式太多，测试脚本将会失败。
- 您可以修改 mr/worker.go、mr/coordinator.go 和 mr/rpc.go。您可以临时修改其他文件以进行测试，但请确保您的代码与原始版本兼容。我们将用原始版本进行测试。
- worker 应将 intermediate Map 输出放在当前目录的文件中以便 worker 稍后可以将它们作为 Reduce 输入读取。
- main/mrcoordinator.go 期望 mr/coordinator.go 实现一个 Done() 方法，该方法在 MapReduce 完全完成时返回 true，此时，mrcoordinator.go 将退出。
- 当任务完全完成时，worker 进程应该退出。实现这一点的一个简单方法是使用 call() 的返回值：如果 worker 无法联系 coordinator，它可以假设 coordinator 已经退出，因为任务已经完成，因此 worker 也可以终止。根据您的设计，您可能还会发现有一个 "please exit" 的伪任务是很有帮助的，coordinator 可以将这个伪任务交给 worker。





---

# 代码框架分析

论文里的 master 在这里是启动一个 main/mrcoordinator.go，简单来说他会启动 mr.MakeCoordinator，并且通过 Done 方法来决定是否退出，按实验要求，应该在任务完成后，Done 方法返回值变为 true。

```go
func main() {
   if len(os.Args) < 2 {
      fmt.Fprintf(os.Stderr, "Usage: mrcoordinator inputfiles...\n")
      os.Exit(1)
   }

   m := mr.MakeCoordinator(os.Args[1:], 10)
   for m.Done() == false {
      time.Sleep(time.Second)
   }

   time.Sleep(time.Second)
}
```

mr/coordinator.go 中 MakeCoordinator 函数需要实现 coordinator 即 master 的功能，其中 files 参数是要被处理的任务文件，nReduce 表示 Reduce 任务的数目。因为 worker 和 coordinator 使用 RPC 通信，所以你要自定义 RPC 的消息结构，`c.server()` 用于启动 RPC 服务器。

```go
func MakeCoordinator(files []string, nReduce int) *Coordinator {
   c := Coordinator{}

   // Your code here.


   c.server()
   return &c
}
```

Coordinator 结构体用于定义 master 管理的一些成员，它的 Done 方法需要你自己去实现。

```go
type Coordinator struct {
   // Your definitions here.

}

func (c *Coordinator) Done() bool {
	ret := false

	// Your code here.


	return ret
}
```

同时实验还给了一个 RPC 通信的例子，这个例子表示实现了 Example 方法，参数是 ExampleArgs，回应是 ExampleReply。

```go
func (c *Coordinator) Example(args *ExampleArgs, reply *ExampleReply) error {
   reply.Y = args.X + 1
   return nil
}
```

至于 RPC 结构体的定义你要到 mr/rpc.go 中自定义，比如 Example 中定义如下：

```go
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
```

对于 worker 角色，每个 worker 都从 main/mrworker.go 启动，传入的参数为应用层 Map 和 Reduce 函数的实现。

```go
func main() {
   if len(os.Args) != 2 {
      fmt.Fprintf(os.Stderr, "Usage: mrworker xxx.so\n")
      os.Exit(1)
   }

   mapf, reducef := loadPlugin(os.Args[1])

   mr.Worker(mapf, reducef)
}
```

mr/worker.go 里的 Worker 没有实现任何东西，都需要你自己去实现，被注释的 CallExample 则是 RPC 通信的例子。

```go
func Worker(mapf func(string, string) []KeyValue,
   reducef func(string, []string) string) {

   // Your worker implementation here.

   // uncomment to send the Example RPC to the coordinator.
   // CallExample()

}
```

另外由于中间结果你要保存到文件里，对于这种 KV 结构可以使用 json 库去处理。

```go
type KeyValue struct {
   Key   string
   Value string
}
```

在文件里还提供了 ihash 函数方便你选择 key 分配给哪个 Reduce 节点去处理。

```go
func ihash(key string) int {
   h := fnv.New32a()
   h.Write([]byte(key))
   return int(h.Sum32() & 0x7fffffff)
}
```

整体来说，代码框架还是简单易懂的。





---

# 设计思路

## worker

首先给 worker 分配一个结构体，包含了其运行的 Map 和 Reduce 函数，以及发送心跳时表明自己的工作状态。

```go
type worker struct {
   sync.Mutex
   status  WorkerStatus
   mapf    func(string, string) []KeyValue
   reducef func(string, []string) string
}
```

对于 worker 来说，首先要定期向 master 发送心跳，这就意味着这个 worker 向 master 注册了，master 可以选择分配任务给这个 worker。

master 无需主动派发任务，只需要在 worker 心跳来临时，如果维护的状态中该 worker 的状态为空闲，则可以告诉它去执行任务。

所以对于 worker 而言，它不需要做太多的事情，只是听从 master 的命令，它叫我们干什么我们就干什么，做一个无情的工作机器。因此 worker 的主函数就是一个无限循环，定期向 master 发送心跳，并根据回复来执行相应的任务。

MapJob 和 ReduceJob 顾名思义就是执行 Map 和 Reduce 任务，HeartBeatResp 表示 worker 你就保持现在的状态就好了，FinishJob 是 MapReduce 任务完成后，master 通知 worker 退出，这也是实验要求的一部分。

```go
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
}
```



### RPC 设计

心跳就是定时告知 master worker自身的状态，master 会根据状态来决定 worker 来执行什么任务。

首先我们来定义一下 RPC 传输的消息体。WorkerStatus 有三个状态，分别表示 worker 空闲，worker 正在执行任务，worker 完成了任务，当 worker 空闲时，master 就会分配一个任务给它，当 worker 忙时，master 不会做多余的动作，任务完成时，心跳包状态为 Completed，此时 master 从中获取任务结果。

```go
type WorkerStatus int
const (
	Idle WorkerStatus = iota
	InProgress
	Completed
)
```

任务类型我们上面已经介绍了

```go
type JobType int
const (
   MapJob JobType = iota
   ReduceJob
   HeartBeatResp
   FinishJob
)
```

接下来是任务本身的定义，虽然有 Map 和 Reduce 两种任务，但是他们可以用一个结构体表示，对于 Map 来说 Input 是输入的分割的文件，OutPut 是中间 KV 结果；对于 Reduce 来说，Input 是 Map 生成的中间文件，OutPut 是最终结果。

StartTime 用于检测任务是否超时，ID 标记一个任务，ReduceNumber 用于 Map 生成中间文件，后面会介绍。

```go
type TaskID int
type File string
type Task struct {
   JobType      JobType
   ID           TaskID
   Input        []File
   OutPut       []File
   StartTime    time.Time
   ReduceNumber int
}
```

最后就是通信的消息体了，也非常简单。

```go
type HeartBeatArgs struct {
   Status WorkerStatus
   Task   Task
}

type HeartBeatReply struct {
   Task Task
}
```



### 心跳设计

对于 worker 来说，它维护自身的状态，并通过心跳发送给 master，这里要注意的就是，普通心跳 1s 一次，但是任务完成时会立马发送一次心跳，所以对于 Completed 状态由具体的任务上报，而不是定时上报。

```go
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
```



### Map 处理

当收到一个 Map 任务时，worker 需要把状态置为 InProgress，这里不能同步返回，因为 Map 是 IO 密集型操作，在海量数据实际场景下耗时较长。

```go
func (w *worker) Map(task *Task) {
   w.SetStatus(InProgress)

   // map 是 IO 密集型操作，要异步操作
   go w.DoMapf(task)
}
```

对于 Map 的流程，其实和代码框架里给的那个示例差不多，首先是读取文件内容，然后执行具体的 map 操作，这里生成的 KV 中间结果我们需要本地持久化，持久化完毕后就可以告知 master 任务完成了。

```go
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
```

这里要注意的是持久化操作，首先我们要根据 key 值将中间结果分到多个文件，课程建议以 Map-X-Y 的格式持久化，X 代表 Map 任务号，Y 代表 Reduce 任务号，相同 Y 值的文件要交给一个 Reduce worker 去处理，这也是理所当然的，就比如字符统计，相同的 key 肯定被 hash 到了同样 Y 值的文件里，Reduce 只有统计所有这些文件才有意义。另外课程给了提示可以用 json 来进行持久化操作。

另一个要注意的点就是持久化操作时，要考虑 crash，这也是实验要求之一，因为 worker 可能在任何时刻故障，所以我们不能直接去写目标文件，最好是先写一个临时文件，然后 rename 操作，因为 rename 是原子的，所以没有中间状态。

```go
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
	_ = os.MkdirAll(intermediateDir, os.ModePerm)
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
```



### Reduce 处理

Reduce 其实和 Map 类似，也是设置状态，然后处理。

```go
func (w *worker) Reduce(task *Task) {
   w.SetStatus(InProgress)

   // reduce 是 IO 密集型操作，要异步操作
   go w.DoReducef(task)
}
```

这里可以借鉴 mrsequential.go 里的代码，只是对于中间处理结果，你需要从文件里读出来而已。

```go

```



## master

我们知道任务要经过**未处理 --> Map --> 输出中间文件 --> Reduce --> 输出结果**这么一个过程，Coordinator 定义如下：

```go
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
```

分割的文件总数则是 Map 任务数，Task 通过通道来传递，对于 Map 任务，就是从文件中生成 Map Task。

```go
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
```

对于 Reduce 任务需要等待 Map 任务全部完成后，然后整理中间文件生成 Reduce Task。这两个生成任务都是用 goroutine 处理。

```go
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
```

判断任务阶段也很简单，如果 MapCompleted 的长度等于 Map 任务数，则表示 Map 任务全部执行完毕了，Reduce 同理。另外实验要求任务结束后，master 和 worker 都要退出，所以这里也可以根据阶段来设置 Done 方法的返回值。

```go
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
```

因此我们初始化流程如下，在初始化 Coordinator 结构体后，使用 AddMapTask 和 AddReduceTask 来创建任务，之后 RPC 服务启动，此时可以开始接收 worker 的消息了。

```go
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
```

下面是重头戏，让我们来梳理一下流程：

1. 当 worker 是空闲时，我们根据当前阶段来给他派发任务；
2. 如果 worker 在工作，则只需要回复普通心跳即可；
3. 当 worker 告知任务完成时，将任务从 Processing 字典转移到 Completed 字典

```go
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
```

这里还有一个问题就是 worker 故障了怎么办，论文中讨论了 master 和 worker 故障，但是实验只要求我们处理 worker 故障。

因为实验是在单机上运行的，所以我这里偷了个懒，因为一旦 Map 任务完成，在本地持久化的文件就不会丢失了，所以我没有 worker id 的处理，而是只需要对任务超时进行判断。这里我是在 worker 报告空闲时，且通道没有任务时去检查，如果任务有超时了，就重新塞到通道里去运行。

这里我考虑的是 worker 执行哪个 Map 任务不重要，重要的是 worker 在执行任务，所以对于任务超时了，只要通道里还有任务，我其实不着急往里面去塞。

```go
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
				Debug("ReTry Reduce Task", task.ID)
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
```

在实际场景中，Map 如果故障了，要重新生成 Map 中间文件，因为它是持久化在本地的，如果节点故障了，那么 Reduce 就获取不到对应的结果了。但是 Reduce 一旦成功就不需要再重试了，因为它是保存在像 GFS 这样的分布式文件系统中。

好了，最后我们来测试一下：

```bash
root@lz-VirtualBox:~/6.824/src/main# bash test-mr.sh 
*** Starting wc test.
--- wc test: PASS
*** Starting indexer test.
--- indexer test: PASS
*** Starting map parallelism test.
--- map parallelism test: PASS
*** Starting reduce parallelism test.
--- reduce parallelism test: PASS
*** Starting job count test.
--- job count test: PASS
*** Starting early exit test.
--- early exit test: PASS
*** Starting crash test.
2022/05/02 16:48:11 dialing:dial unix /var/tmp/824-mr-0: connect: connection refused
2022/05/02 16:48:11 dialing:dial unix /var/tmp/824-mr-0: connect: connection refused
2022/05/02 16:48:11 dialing:dial unix /var/tmp/824-mr-0: connect: connection refused
--- crash test: PASS
*** PASSED ALL TESTS
```

对于实验代码 MIT 是不允许大家上传到网上的，所以这里也不贴完整的代码了（其实也贴得七七八八了）。事实上，自己一步步做出来，踩过坑，才会理解更深刻一些。

另外对于测试最好多执行几遍，你可以写个脚本让它自己测试个 100 遍。因为你写的代码可能并不是很健壮，多测试几遍有利于暴露问题，对于我们程序员来说，我们要追求写出没有 BUG 的代码，而不仅仅是一份能运行的代码，毕竟对于自己创造的世界，怎么能容许缺陷的存在呢。

另外测试并发时，有时候会显示 FAIL，我看了下测试用例，对应的 rtiming 默认定时 1 s，而我的心跳也是 1 s，这样可能有时候检测不到并发存在，所以我把 rtiming 增加到了 3 s，表示应用需要 3s 时间来处理 Reduce。