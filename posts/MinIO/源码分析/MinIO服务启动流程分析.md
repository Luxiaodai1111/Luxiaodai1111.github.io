# 命令初始化

minio 命令是从 minio/main.go 开始的，这里值得注意的是在 import 的过程中，会进行内部组件和网关的一些初始化工作。

```go
import (
	"os"

	// MUST be first import.
	_ "github.com/minio/minio/internal/init"

	minio "github.com/minio/minio/cmd"

	// Import gateway
	_ "github.com/minio/minio/cmd/gateway"
)

func main() {
	minio.Main(os.Args)
}
```

App 是一个 cli 应用程序的主要结构。`cli.NewApp()` 函数来创建一个应用程序。

```go
func Main(args []string) {
	// Set the minio app name.
	appName := filepath.Base(args[0])

	// Run the app - exit on error.
	if err := newApp(appName).Run(args); err != nil {
		os.Exit(1)
	}
}
```

NewApp 里面定义了两个匿名函数，`registerCommand` 用于注册命令，目前只有 serverCmd 和 gatewayCmd，分别对应 minio 命令行的 server 和 gateway 参数。App 里会记录所有的子命令，并且以字典树排列，这用于另一个匿名函数 `findClosestCommands`，在你输错命令时找到最相近的子命令给予提示。

最后就是分配 App 结构体并初始化其字段，我们这里关心的主要是 serverCmd 和 gatewayCmd 的 Action 字段，后面在启动时会调用具体子命令的 Action 回调函数来执行对应子命令的服务。

```go
func newApp(name string) *cli.App {
	// Collection of minio commands currently supported are.
	commands := []cli.Command{}

	// Collection of minio commands currently supported in a trie tree.
	commandsTree := trie.NewTrie()

	// registerCommand registers a cli command.
	registerCommand := func(command cli.Command) {
		commands = append(commands, command)
		commandsTree.Insert(command.Name)
	}

	findClosestCommands := func(command string) []string {
		...

		return closestCommands
	}

	// Register all commands.
	registerCommand(serverCmd)
	registerCommand(gatewayCmd)

	// Set up app.
	cli.HelpFlag = cli.BoolFlag{
		Name:  "help, h",
		Usage: "show help",
	}

	app := cli.NewApp()
	app.Name = name
	app.Author = "MinIO, Inc."
	...

	return app
}
```

App 初始化之后，就调用 Run 方法去启动，Run 是 cli 应用程序的入口点，它负责解析参数片，并路由到适当的 flag/args 组合。这里没什么特别的，如果定义了一些 Before 和 After 操作，那么就会在 Action 之前设置好，最后会使用 HandleAction 去执行 Action 操作，以 server 举例，它对应的启动服务为 `serverMain`。

```go
var serverCmd = cli.Command{
	Name:   "server",
	Usage:  "start object storage server",
	Flags:  append(ServerFlags, GlobalFlags...),
	Action: serverMain,
	CustomHelpTemplate: ...
}
```





---

# server启动

我们首先给出 serverMain 的整体流程：

1.  注册中断信号以优雅地结束
2.  进行一些自检，如 bitrot、EC、压缩等确保硬件计算正确
3.  解析命令行和环境变量参数
4.  初始化子系统，如自动恢复、IAM 等
5.  设置 HTTP 路由并启动 HTTP 服务
6.  初始化对象层，根据不同的参数来决定是否启动 EC，对象层负责实际对象的各种操作
7.  初始化自动修复和后台清理过期对象 routine
8.  启动 console
9.  初始化一些对象复制、数据扫描等功能的 routine

我们按照顺序一个个来分析，Let's Go！



## 退出机制

```go
signal.Notify(globalOSSignalCh, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
go handleSignals()
```

在注册信号之后，启动 `handleSignals` 等待信号中断。

`exit` 匿名函数关闭所有用于服务分析的 Profiler。

`stopProcess` 匿名函数首先向各种运行程序发送 cancel 信号，告知它们需要退出，然后关闭所有需要通知的目标，接下来关闭启动的服务，包括 HTTP 和 Console 的服务以及对象层的服务，其中 HTTP 和 对象层是 MinIO 工作的核心服务，前者提供 S3 以及各集群的通信，后者提供对象操作的各种接口。

当收到中断信号后，退出进程，并告知退出结果。

如果是重启消息，则重新启动命令行，`restartProcess` 启动一个新的进程并把活动的fd传给它。它并不进行 fork 操作，而是使用与最初启动时相同的环境和参数启动一个新的进程，这使得新部署的二进制文件可以被启动。成功时，它返回新启动的进程的 pid。

```go
func handleSignals() {
   // Custom exit function
   exit := func(success bool) {
      // If global profiler is set stop before we exit.
      globalProfilerMu.Lock()
      defer globalProfilerMu.Unlock()
      for _, p := range globalProfiler {
         p.Stop()
      }

      if success {
         os.Exit(0)
      }

      os.Exit(1)
   }

   stopProcess := func() bool {
      // send signal to various go-routines that they need to quit.
      cancelGlobalContext()

      globalNotificationSys.RemoveAllRemoteTargets()

      httpServer := newHTTPServerFn()
      httpServer.Shutdown()

      objAPI := newObjectLayerFn()
      objAPI.Shutdown(context.Background())

      srv := newConsoleServerFn()
      srv.Shutdown()

      return (err == nil && oerr == nil)
   }

   for {
      select {
      case <-globalHTTPServerErrorCh:
         exit(stopProcess())
      case osSignal := <-globalOSSignalCh:
         logger.Info("Exiting on signal: %s", strings.ToUpper(osSignal.String()))
         exit(stopProcess())
      case signal := <-globalServiceSignalCh:
         switch signal {
         case serviceRestart:
            logger.Info("Restarting on service signal")
            stop := stopProcess()
            rerr := restartProcess()
            logger.LogIf(context.Background(), rerr)
            exit(stop && rerr == nil)
         case serviceStop:
            logger.Info("Stopping on service signal")
            exit(stopProcess())
         }
      }
   }
}
```



## 硬件自检

`bitrotSelfTest` 执行自检，以确保 bitrot 算法能计算出正确的校验和。如果任何算法产生一个不正确的校验和，它就会以一个硬件错误而失败退出。bitrotSelfTest 试图尽早发现 bitrot 实现中的任何问题，而不是默默地破坏数据。从 bitrotSelfTest 里我们看到 MinIO 支持 SHA256、BLAKE2b512、HighwayHash256、HighwayHash256S 哈希算法。

```go
// Perform any self-tests
bitrotSelfTest()
erasureSelfTest()
compressSelfTest()
```

`erasureSelfTest` 和 `compressSelfTest` 也类似，用于检测 EC 算法和压缩算法计算是否正确。



## 解析参数

```go
// Handle all server command args.
serverHandleCmdArgs(ctx)

// Handle all server environment vars.
serverHandleEnvVars()
```

serverHandleCmdArgs 主要负责解析命令行的参数，serverHandleEnvVars 负责处理环境变量参数。这里主要讲一下 serverHandleCmdArgs 中 `createServerEndpoints` 函数的实现，它负责解析集群成员，支持省略号这种语法糖，这也是官方推荐的书写方式。不论使不使用可省略号语法，流程基本是一致的，基本上就是调用 `GetAllSets` 和 `CreateEndpoints` 来解析。对于 EC 模式，一般使用省略号的语法来指定。

```go
func createServerEndpoints(serverAddr string, args ...string) (
	endpointServerPools EndpointServerPools, setupType SetupType, err error,
) {
	...

	if !ellipses.HasEllipses(args...) {
		setArgs, err := GetAllSets(args...)
		
		endpointList, newSetupType, err := CreateEndpoints(serverAddr, false, setArgs...)
	
		endpointServerPools = append(endpointServerPools, PoolEndpoints{
			Legacy:       true,
			SetCount:     len(setArgs),
			DrivesPerSet: len(setArgs[0]),
			Endpoints:    endpointList,
			CmdLine:      strings.Join(args, " "),
		})
		setupType = newSetupType
		return endpointServerPools, setupType, nil
	}

	var foundPrevLocal bool
	for _, arg := range args {
		setArgs, err := GetAllSets(arg)

		endpointList, gotSetupType, err := CreateEndpoints(serverAddr, foundPrevLocal, setArgs...)
		
		if err = endpointServerPools.Add(PoolEndpoints{
			SetCount:     len(setArgs),
			DrivesPerSet: len(setArgs[0]),
			Endpoints:    endpointList,
			CmdLine:      arg,
		}); err != nil {
			return nil, -1, err
		}
		foundPrevLocal = endpointList.atleastOneEndpointLocal()
		if setupType == UnknownSetupType {
			setupType = gotSetupType
		}
		if setupType == ErasureSetupType && gotSetupType == DistErasureSetupType {
			setupType = DistErasureSetupType
		}
	}

	return endpointServerPools, setupType, nil
}
```

在 MinIO 里有 FS 模式（单存储点）和 EC 模式，其中 EC 模式又分为单机 EC 和分布式 EC，虽然在大部分场景下都是分布式 EC 的模式，不过剩下两种模式也是很有用处的，比如 gateway 工作原理就类似于 FS 模式，单机 EC 其实可以认为是一个更灵活的 RAID。

FS 模式命令行启动格式如下：

```bash
minio server E:/data
```

单机 EC 启动格式如下：

```bash
minio server E:/data/{1...4}
# 或者
minio server E:/data/1 E:/data/2 E:/data/3 E:/data/4
```

分布式 EC 启动格式如下：

```bash
minio server hostname{1...4}:9000/E:/data/{1...4}
```

命令表示使用 hostname1，hostname2，hostname3，hostname4 四个节点，每个节点上有 4 个存储点，总共 16 个存储点。

多个 server pool 启动格式如下：

```bash
minio server hostname{1...4}:9000/E:/data/{1...4} hostname{5...6}:9000/E:/data/{1...4}
```

命令表示有两个 server pool，每个 pool 是独立的。



### GetAllSets

GetAllSets 解析所有省略号的输入参数，将它们扩展为相应的端点列表，并按照特定的集合大小均匀地分块。比如说。{1...64} 被分为 4 组，每组大小为 16。

我们可以看到函数主要分为三部分：

-   读取 MINIO_ERASURE_SET_DRIVE_COUNT 环境变量，看是否指定了每个 set 里的 drive 数目
-   获取 setArgs，这个稍后分析
-   最后判断从参数中解析的 setArgs 是否包含了重复的存储点

```go
func GetAllSets(args ...string) ([][]string, error) {
   var customSetDriveCount uint64
   if v := env.Get(EnvErasureSetDriveCount, ""); v != "" {
      driveCount, err := strconv.Atoi(v)
      ...
      customSetDriveCount = uint64(driveCount)
   }

   var setArgs [][]string
   if !ellipses.HasEllipses(args...) {
      var setIndexes [][]uint64
      // EC 模式的磁盘是一个一个指定的，不推荐这种写法
      if len(args) > 1 {
         var err error
         setIndexes, err = getSetIndexes(args, []uint64{uint64(len(args))}, customSetDriveCount, nil)
         ...
      } else {
         // FS 模式下就只有一个存储点
         setIndexes = [][]uint64{{uint64(len(args))}}
      }
      s := endpointSet{
         endpoints:  args,
         setIndexes: setIndexes,
      }
      setArgs = s.Get()
   } else {
      // 省略号语法糖
      s, err := parseEndpointSet(customSetDriveCount, args...)
      ...
      setArgs = s.Get()
   }

   // 检查是否有重复的存储点
   uniqueArgs := set.NewStringSet()
   for _, sargs := range setArgs {
      for _, arg := range sargs {
         if uniqueArgs.Contains(arg) {
            return nil, config.ErrInvalidErasureEndpoints(nil).Msg(fmt.Sprintf("Input args (%s) has duplicate ellipses", args))
         }
         uniqueArgs.Add(arg)
      }
   }

   return setArgs, nil
}
```

获取 setArgs 对于 FS 模式，set 数目很明显为 1。

对于不带省略号和带省略号的 EC 来说，处理流程本质上是一样的，省略号只是 minio 提供的语法糖，还是会被解析出单独的一个个参数，最后都会调用 getSetIndexes 来获取 setArgs。

getSetIndexes 有四个参数，第一个为命令行参数，第二个为每个池的后端存储点数目，第三个为指定的每个 set 中的 drive 数目，为 0 表示未指定，第四个为参数模式。返回值为列表，代表每个池中 set 的个数。流程如下：

-   `getDivisibleSize` 求最大公约数（没想到什么时候会有多个 totalSizes）

-   ```go
    // getDivisibleSize - returns a greatest common divisor of all the ellipses sizes.
    func getDivisibleSize(totalSizes []uint64) (result uint64) {
       gcd := func(x, y uint64) uint64 {
          for y != 0 {
             x, y = y, x%y
          }
          return x
       }
       result = totalSizes[0]
       for i := 1; i < len(totalSizes); i++ {
          result = gcd(result, totalSizes[i])
       }
       return result
    }
    ```

-   `possibleSetCounts` 的方法就是只要上面的最大公约数能被 setSizes 中的数字整除，就先记录下来。比如传入了 20 个存储点，那么可能的 setCounts = [4，5，10]。当计算出来的 setCounts 列表长度为 0 时（比如总数为 19 块盘），返回错误。

-   ```go
    var setSizes = []uint64{4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
    
    possibleSetCounts := func(setSize uint64) (ss []uint64) {
       for _, s := range setSizes {
          if setSize%s == 0 {
             ss = append(ss, s)
          }
       }
       return ss
    }
    ```
    
-   如果用户指定了每个 set 里的 drive 数目，那么就从 setCounts 结果里看是否有满足的，得出 setSize。

-   否则先使用 possibleSetCountsWithSymmetry 返回对称的 setCounts，什么是对称呢？比如 `hostname{1...2}:9000/E:/data/{1...9}` 这种格式表示有两个节点，每个节点 9 块盘，那么总共 18 块盘可以得出 setCounts = [6，9]，如果采用 9 块，那么磁盘分散在两个节点上就会不对称。这里如果找不出对称的值也会返回失败，比如像 `hostname{1...2}:9000/E:/data/{1...11}`。

-   commonSetDriveCount 从对称的 setCounts 结果中找到使 set 数目最少的（这里直接选 setCounts 里最大的不就行了？不太明白什么情况下一定需要计算）

-   ```go
    // Returns possible set counts with symmetry.
    setCounts = possibleSetCountsWithSymmetry(setCounts, argPatterns)
    
    if len(setCounts) == 0 {
    	msg := fmt.Sprintf("No symmetric distribution detected with input endpoints provided %s, 
                           disks %d cannot be spread symmetrically by any supported erasure set sizes %d", args, commonSize, setSizes)
    	return nil, config.ErrInvalidNumberOfErasureEndpoints(nil).Msg(msg)
    }
    
    // Final set size with all the symmetry accounted for.
    setSize = commonSetDriveCount(commonSize, setCounts)
    ```
    
-   最后检查 setSize 是否在支持的范围内。





### CreateEndpoints

对于 FS 模式，这里可以简化为如下代码，流程也非常简单，无非就是根据唯一的 set 来创建 endpoint，模式设置为 FSSetupType。

```go
if len(args) == 1 && len(args[0]) == 1 {
   var endpoint Endpoint
   endpoint, err = NewEndpoint(args[0][0])
   
   endpoint.UpdateIsLocal()
    
   endpoints = append(endpoints, endpoint)
   setupType = FSSetupType

   // Check for cross device mounts if any.
   if err = checkCrossDeviceMounts(endpoints); err != nil {
      return endpoints, setupType, config.ErrInvalidFSEndpoint(nil).Msg(err.Error())
   }

   return endpoints, setupType, nil
}
```

对于 EC 模式，每个 drive 都对应一个 endpoint，其实就类似进行了多次 FS 模式的初始化，模式设置为 ErasureSetupType，最后返回一个列表。

```go
for _, iargs := range args {
	// Convert args to endpoints
	eps, err := NewEndpoints(iargs...)

	// Check for cross device mounts if any.
	if err = checkCrossDeviceMounts(eps); err != nil {
		return endpoints, setupType, config.ErrInvalidErasureEndpoints(nil).Msg(err.Error())
	}
    
	endpoints = append(endpoints, eps...)
}

// Return Erasure setup when all endpoints are path style.
if endpoints[0].Type() == PathEndpointType {
	setupType = ErasureSetupType
	return endpoints, setupType, nil
}
```



## initAllSubsystems

基本就是初始化各种子系统的数据结构和相关的 routine，后面分析到的时候再来回头看，比如用于故障恢复的 allHealState 等





## 初始化HTTP服务

`configureServer` 处理程序返回 HTTP 服务器的最终处理程序，然后开启 HTTP 服务并设置全局变量 globalHTTPServer

```go
// Configure server.
handler, err := configureServerHandler(globalEndpoints)
if err != nil {
    logger.Fatal(config.ErrUnexpectedError(err), "Unable to configure one of server's RPC services")
}

httpServer := xhttp.NewServer(addrs).
		UseHandler(setCriticalErrorHandler(corsHandler(handler))).
		UseTLSConfig(newTLSConfig(getCert)).
		UseShutdownTimeout(ctx.Duration("shutdown-timeout")).
		UseBaseContext(GlobalContext).
		UseCustomLogger(log.New(ioutil.Discard, "", 0)) // Turn-off random logging by Go stdlib

// 启动 HTTP
go func() {
	globalHTTPServerErrorCh <- httpServer.Start(GlobalContext)
}()

// 设置 globalHTTPServer
setHTTPServer(httpServer)
```

configureServerHandler 使用 gorilla/mux 库来设置路由，包括一些分布式的、管理、监控、S3 等接口。

```go
func configureServerHandler(endpointServerPools EndpointServerPools) (http.Handler, error) {
   // Initialize router. `SkipClean(true)` stops gorilla/mux from
   // normalizing URL path minio/minio#3256
   router := mux.NewRouter().SkipClean(true).UseEncodedPath()

   // Initialize distributed NS lock.
   if globalIsDistErasure {
      registerDistErasureRouters(router, endpointServerPools)
   }

   // Add Admin router, all APIs are enabled in server mode.
   registerAdminRouter(router, true)

   // Add healthcheck router
   registerHealthCheckRouter(router)

   // Add server metrics router
   registerMetricsRouter(router)

   // Add STS router always.
   registerSTSRouter(router)

   // Add API router
   registerAPIRouter(router)

   router.Use(globalHandlers...)

   return router, nil
}
```

比如在 registerAPIRouter 中就设置了 PutObject 路由，具体的对象操作分析我们留到以后再讲。

```go
// PutObject
router.Methods(http.MethodPut).Path("/{object:.+}").HandlerFunc(
   collectAPIStats("putobject", maxClients(gz(httpTraceHdrs(api.PutObjectHandler)))))
```



## newObjectLayer

存储层有两种引擎：FSObjects 和 erasureServerPools，他们都是 ObjectLayer 接口，这样上层服务不需要知道底层是怎么实现的，只需要调用对应的接口即可。



### FSObjects初始化

这里我们只分析 FSObjects 的初始化流程，erasureServerPools 的初始化我们单独开一篇介绍，NewFSObjectLayer 精简代码如下：

```go
func NewFSObjectLayer(fsPath string) (ObjectLayer, error) {
   ctx := GlobalContext
   if fsPath == "" {
      return nil, errInvalidArgument
   }

   var err error
   if fsPath, err = getValidPath(fsPath); err != nil {
      ...
   }

   // 每次服务启动实例都会有随机的 UUID，在 .minio.sys/tmp/uuid 下会存放临时的对象文件
   fsUUID := mustGetUUID()

   // 初始化 meta volume
   initMetaVolumeFS(fsPath, fsUUID)

   // 初始化 `format.json`
   initFormatFS(ctx, fsPath)

   // Initialize fs objects.
   fs := &FSObjects{
      fsPath:       fsPath,
      metaJSONFile: fsMetaJSONFile,
      fsUUID:       fsUUID,
      rwPool: &fsIOPool{
         readersMap: make(map[string]*lock.RLockedFile),
      },
      nsMutex:       newNSLock(false),
      listPool:      NewTreeWalkPool(globalLookupTimeout),
      appendFileMap: make(map[string]*fsAppendFile),
      diskMount:     mountinfo.IsLikelyMountPoint(fsPath),
   }

   fs.fsFormatRlk = rlk

   go fs.cleanupStaleUploads(ctx)
   go intDataUpdateTracker.start(ctx, fsPath)

   // Return successfully initialized object layer.
   return fs, nil
}
```

`initMetaVolumeFS` 初始化时会在我们指定的目录下创建元数据目录 `.minio.sys`

```bash
└─.minio.sys
    ├─buckets                                       // dataUsageBucket
    ├─multipart                                     // minioMetaMultipartBucket
    └─tmp                                           // minioMetaTmpBucket
        └─107ac8b1-dad5-40f0-9e49-811741c526c4      // fsUUID
            └─bg-appends                            // bgAppendsDirName
```

`initFormatFS` 初始化 `.minio.sys/format.json`。该函数向调用者返回一个被读锁定的 format.json 引用，该文件描述符应在程序运行生命周期中保持 open。一旦文件系统被初始化，就会在服务器的生命期内保持读锁。这样做是为了确保在 FS 的共享后端模式下，别的 minio 进程不会迁移或导致后端格式的改变。

format 格式结构体如下，包含一个基础的 formatMetaV1 信息，以及特有的 FS Version 信息。基础信息包括版本号、类型和 ID 标识符。

```go
type formatMetaV1 struct {
	// Version of the format config.
	Version string `json:"version"`
	// Format indicates the backend format type, supports two values 'xl' and 'fs'.
	Format string `json:"format"`
	// ID is the identifier for the minio deployment
	ID string `json:"id"`
}

type formatFSV1 struct {
   formatMetaV1
   FS struct {
      Version string `json:"version"`
   } `json:"fs"`
}
```

如果文件不存在的话则会创建它，内容如下：

```json
{"version":"1","format":"fs","id":"1715267d-e670-4aab-9498-8c273af50bfe","fs":{"version":"1"}}
```

版本迁移在 formatFSV1.FS.Version 改变时发生。当结构 formatFSV1.FS 发生变化或后端文件系统树结构发生任何变化时，这个版本就会发生变化。当前最新版本为 2，因此需要迁移。

从 V1 迁移到 V2。V2 实现了新的 multipart 上传的后端格式。因此删除以前的 multipart 目录再重建。迁移后 format.json 内容如下：

```json
{"version":"1","format":"fs","id":"1715267d-e670-4aab-9498-8c273af50bfe","fs":{"version":"2"}}
```

设置完 FSObjects 之后，开启了两个 goroutine，一个用于清理 multipart 上传遗留的超时对象，一个用于统计使用量的布隆过滤器优化。

至此我们核心的 HTTP 服务和对象处理层已经初始化完毕。



## 其余初始化

如果是 EC 模式，则需要有自动修复的线程，这个我们后面专门写一篇来讲数据恢复。

```go
// Enable background operations for erasure coding
if globalIsErasure {
   initAutoHeal(GlobalContext, newObject)
   initHealMRF(GlobalContext, newObject)
}
```

initBackgroundExpiry 负责清理过期对象，另外默认会开启 Console。

```go
initBackgroundExpiry(GlobalContext, newObject)
...
initServer(GlobalContext, newObject)
...
initConsoleServer()
...
```

最后的一些初始化会异步去做，比如复制之类的，这个分析到具体的功能我们再来分析。

```go
// Background all other operations such as initializing bucket metadata etc.
go func() {
   // Initialize transition tier configuration manager
   if globalIsErasure {
      initBackgroundReplication(GlobalContext, newObject)
      initBackgroundTransition(GlobalContext, newObject)

      go func() {
         if err := globalTierConfigMgr.Init(GlobalContext, newObject); err != nil {
            logger.LogIf(GlobalContext, err)
         }

         globalTierJournal, err = initTierDeletionJournal(GlobalContext)
         if err != nil {
            logger.FatalIf(err, "Unable to initialize remote tier pending deletes journal")
         }
      }()
   }

   // Initialize site replication manager.
   globalSiteReplicationSys.Init(GlobalContext, newObject)

   // Initialize quota manager.
   globalBucketQuotaSys.Init(newObject)

   initDataScanner(GlobalContext, newObject)

   // List buckets to heal, and be re-used for loading configs.
   buckets, err := newObject.ListBuckets(GlobalContext)
   if err != nil {
      logger.LogIf(GlobalContext, fmt.Errorf("Unable to list buckets to heal: %w", err))
   }
   // initialize replication resync state.
   go globalReplicationPool.initResync(GlobalContext, buckets, newObject)

   // Populate existing buckets to the etcd backend
   if globalDNSConfig != nil {
      // Background this operation.
      go initFederatorBackend(buckets, newObject)
   }

   // Initialize bucket metadata sub-system.
   globalBucketMetadataSys.Init(GlobalContext, buckets, newObject)

   // Initialize bucket notification targets.
   globalNotificationSys.InitBucketTargets(GlobalContext, newObject)

   // initialize the new disk cache objects.
   if globalCacheConfig.Enabled {
      logger.Info(color.Yellow("WARNING: Disk caching is deprecated for single/multi drive MinIO setups. Please migrate to using MinIO S3 gateway instead of disk caching"))
      var cacheAPI CacheObjectLayer
      cacheAPI, err = newServerCacheObjects(GlobalContext, globalCacheConfig)
      logger.FatalIf(err, "Unable to initialize disk caching")

      setCacheObjectLayer(cacheAPI)
   }

   // Prints the formatted startup message, if err is not nil then it prints additional information as well.
   printStartupMessage(getAPIEndpoints(), err)
}()
```

最后打印初始化信息，服务启动完成。

```go
if serverDebugLog {
   logger.Info("== DEBUG Mode enabled ==")
   logger.Info("Currently set environment settings:")
   ks := []string{
      config.EnvAccessKey,
      config.EnvSecretKey,
      config.EnvRootUser,
      config.EnvRootPassword,
   }
   for _, v := range os.Environ() {
      // Do not print sensitive creds in debug.
      if contains(ks, strings.Split(v, "=")[0]) {
         continue
      }
      logger.Info(v)
   }
   logger.Info("======")
}

<-globalOSSignalCh
```

​	
