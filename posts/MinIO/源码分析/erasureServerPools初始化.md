# 概述

前面我们分析 minio server 初始化过程中，对于对象层的初始化只分析了 FS 类型的初始化，这里我们再对 EC 模式的对象层初始化进行分析。整个初始化流程如下：

1.  为每个池初始化 EC 比例，初始化 set
    -   `ecDrivesNoConfig` 计算 EC 比例
    -   `waitForFormatErasure` 在初始化对象层之前对磁盘进行格式化
    -   `newErasureSets` 初始化 set
2.  `z.Init` 初始化整个 EC Server Pool
3.  和 FS 模式一样，开启布隆过滤器用来优化统计存储使用量

```go
func newErasureServerPools(ctx context.Context, endpointServerPools EndpointServerPools) (ObjectLayer, error) {
   var (
      ...
      formats      = make([]*formatErasureV3, len(endpointServerPools))
      storageDisks = make([][]StorageAPI, len(endpointServerPools))
      z            = &erasureServerPools{
         serverPools: make([]*erasureSets, len(endpointServerPools)),
      }
   )

   ...
   for i, ep := range endpointServerPools {
      if commonParityDrives == 0 {
         commonParityDrives = ecDrivesNoConfig(ep.DrivesPerSet)
      }
      ...

      storageDisks[i], formats[i], err = waitForFormatErasure(local, ep.Endpoints, i+1,
         ep.SetCount, ep.DrivesPerSet, deploymentID, distributionAlgo)
      ...

      z.serverPools[i], err = newErasureSets(ctx, ep, storageDisks[i], formats[i], commonParityDrives, i)
   }

   ...
   for {
      err := z.Init(ctx) // Initializes all pools.
      if err != nil {
         ...
         time.Sleep(retry)
         continue
      }
      break
   }

   ...
   go intDataUpdateTracker.start(ctx, drives...)
   return z, nil
}
```

我们接下来一个一个分析，Let’s Go！



---

# 计算 EC 比例

ecDrivesNoConfig 用于计算每个 set 里用于存储数据和纠删的驱动器数目。

首先调用 LookupConfig 从配置中去找，如果有配置的话，就使用配置里指定的值。GetParityForSC 返回基于 storageclass 的数据和奇偶校验驱动器计数。如果用户没有指定 EC 比例，那么就返回默认值。

```go
func ecDrivesNoConfig(setDriveCount int) int {
   sc, _ := storageclass.LookupConfig(config.KVS{}, setDriveCount)
   ecDrives := sc.GetParityForSC(storageclass.STANDARD)
   if ecDrives <= 0 {
      ecDrives = getDefaultParityBlocks(setDriveCount)
   }
   return ecDrives
}
```

MinIO 当前支持两种存储级别：**Reduced Redundancy** 和 **Standard**，通过对两种级别的设置来修改对象的 `Parity Drives (P)`（奇偶校验块）和 `Data Drives (D)`（数据块）的比例，让用户能够更好的控制磁盘使用率和容错性。

STANDARD 存储级别包含比 REDUCED_REDUNDANCY 存储级别更多的奇偶校验块，因此 STANDARD 存储级别的奇偶校验块需要满足如下条件：

-   STANDARD 存储级别的奇偶校验块需要大于等于 2（也就是说比如 4:1 这种比例是设置不了的）
-   在设置了 REDUCED_REDUNDANCY 存储级别的情况下，STANDARD 存储级别的奇偶校验块需要大于 REDUCED_REDUNDANCY 存储级别的奇偶校验块数量
-   奇偶校验块的数量必须小于数据块数量，所以 STANDARD 存储级别的奇偶校验块不能大于 N/2（N 为 Erasure Set 中的磁盘数量）

STANDARD 存储级别的奇偶校验块的默认值取决于 Erasure Set 中的磁盘数量：

| Erasure Set Size | Default Parity (EC:N) |
| ---------------- | --------------------- |
| 5 or fewer       | EC:2                  |
| 6-7              | EC:3                  |
| 8 -16            | EC:4                  |

STANDARD 存储级别的奇偶校验块默认值为：`EC:4`

REDUCED_REDUNDANCY 存储级别包含比 STANDARD 存储级别更少的奇偶校验块，因此 REDUCED_REDUNDANCY 存储级别的奇偶校验块需要满足如下条件：

-   在设置了 STANDARD 存储级别的情况下，REDUCED_REDUNDANCY 存储级别的奇偶校验块需要小于 STANDARD 存储级别的奇偶校验块数量
-   奇偶校验块的数量必须小于数据块数量
-   结合上述两条，REDUCED_REDUNDANCY 存储级别的奇偶校验块需要大于等于 2，所以 Erasure Set 中的磁盘数量大于 4 的时候才支持 REDUCED_REDUNDANCY 存储级别。

REDUCED_REDUNDANCY 存储级别的奇偶校验块默认值为：`EC:2`

可以通过设置环境变量 MINIO_STORAGE_CLASS_STANDARD 和 MINIO_STORAGE_CLASS_RRS 来修改或者在对象上传时，通过设置 `x-amz-storage-class` 元数据为 `REDUCED_REDUNDANCY` 或 `STANDARD` 来为对象选择不同的存储级别。

LookupConfig 首先从环境变量获取对应值，如果获取到了，使用 parseStorageClass 解析，格式必须是 `EC:Number`，RRS 默认设置为 EC:2。最后检查设置的值是否合理，规则就如上面介绍的那样。

```go
func LookupConfig(kvs config.KVS, setDriveCount int) (cfg Config, err error) {
   ...

   ssc := env.Get(StandardEnv, kvs.Get(ClassStandard))
   rrsc := env.Get(RRSEnv, kvs.Get(ClassRRS))
   // Check for environment variables and parse into storageClass struct
   if ssc != "" {
      cfg.Standard, err = parseStorageClass(ssc)
   }

   if rrsc != "" {
      cfg.RRS, err = parseStorageClass(rrsc)
   }
   if cfg.RRS.Parity == 0 {
      cfg.RRS.Parity = defaultRRSParity
   }

   if err = validateParity(cfg.Standard.Parity, cfg.RRS.Parity, setDriveCount); err != nil {
      return Config{}, err
   }

   return cfg, nil
}
```

GetParityForSC 返回基于 storageclass 的数据和奇偶校验驱动器计数，如果使用环境变量 MINIO_STORAGE_CLASS_RRS 和 MINIO_STORAGE_CLASS_STANDARD 或服务器配置字段设置存储类别，则返回相应的值。

-   如果输入的存储类别是空的，则假定为标准的。
-   如果输入是 RRS，但 RRS 没有被配置，则假定 RRS 的奇偶性为 2。
-   如果输入是 STANDARD，但 STANDARD 没有配置，则返回 0，期望调用者在这一点上选择正确的奇偶校验。

初始化这里显然调用的是 STANDARD，就表示如果用户指定了，就用用户指定值，否则返回 0。

```go
const (
	ClassStandard = "standard"
	ClassRRS      = "rrs"

	// Reduced redundancy storage class environment variable
	RRSEnv = "MINIO_STORAGE_CLASS_RRS"
	// Standard storage class environment variable
	StandardEnv = "MINIO_STORAGE_CLASS_STANDARD"

	// Supported storage class scheme is EC
	schemePrefix = "EC"

	// Min parity disks
	minParityDisks = 2

	// Default RRS parity is always minimum parity.
	defaultRRSParity = minParityDisks
)

func (sCfg Config) GetParityForSC(sc string) (parity int) {
   ConfigLock.RLock()
   defer ConfigLock.RUnlock()
   switch strings.TrimSpace(sc) {
   case RRS:
      // set the rrs parity if available
      if sCfg.RRS.Parity == 0 {
         return defaultRRSParity
      }
      return sCfg.RRS.Parity
   default:
      return sCfg.Standard.Parity
   }
}
```

如果没有配置 EC，那么 getDefaultParityBlocks 返回默认的 EC 比例。

```go
func getDefaultParityBlocks(drive int) int {
	switch drive {
	case 3, 2:
		return 1
	case 4, 5:
		return 2
	case 6, 7:
		return 3
	default:
		return 4
	}
}
```





---

# 磁盘初始化

抛开重试的代码来看，waitForFormatErasure 就是调用 connectLoadInitFormats 函数来进行初始化。

```go
func waitForFormatErasure(firstDisk bool, endpoints Endpoints, poolCount, setCount, setDriveCount int, 
                          deploymentID, distributionAlgo string) ([]StorageAPI, *formatErasureV3, error) {
   ...

   var tries int
   var verboseLogging bool
   storageDisks, format, err := connectLoadInitFormats(verboseLogging, firstDisk, endpoints, poolCount, setCount, setDriveCount,
                                                       deploymentID, distributionAlgo)
   if err == nil {
      return storageDisks, format, nil
   }

   tries++ // tried already once

   // Wait on each try for an update.
   ticker := time.NewTicker(150 * time.Millisecond)
   defer ticker.Stop()

   for {
       ...
   }
}
```

connectLoadInitFormats 连接 endpoints 列表并加载所有的 Erasure 磁盘格式，验证格式是否正确并在法定人数内，如果没有找到 format，则尝试首次初始化所有的格式。另外，确保关闭这次尝试中使用的所有磁盘。整体流程如下：

1.  初始化所有磁盘，即初始化操作所有磁盘的接口
2.  尝试从所有磁盘中加载 `format.json`
3.  预先检查一个格式化的磁盘是否无效。这个函数在大多数情况下返回成功，除非其中一个格式与预期的 Erasure format 不一致。例如，如果一个用户试图将 FS 后端汇集到一个 Erasure Set。
4.  如果所有磁盘都报告未格式化（errUnformattedDisk），我们应该初始化 format，初始化完毕后返回
5.  后面是一些判断，比如池中磁盘要超过法定数目才算启动，根节点磁盘标记为不可用等等

```go
func connectLoadInitFormats(verboseLogging bool, firstDisk bool, endpoints Endpoints, poolCount, setCount, setDriveCount int,
                            deploymentID, distributionAlgo string) (storageDisks []StorageAPI, format *formatErasureV3, err error) {
   // 初始化所有存储磁盘
   storageDisks, errs := initStorageDisksWithErrors(endpoints)

   defer func(storageDisks []StorageAPI) {
      if err != nil {
         closeStorageDisks(storageDisks)
      }
   }(storageDisks)

   ...

   // 尝试从所有磁盘中加载 `format.json`
   formatConfigs, sErrs := loadFormatErasureAll(storageDisks, false)

   /*
    * 预先检查一个格式化的磁盘是否无效。
    * 这个函数在大多数情况下返回成功，除非其中一个格式与预期的 Erasure format 不一致。
    * 例如，如果一个用户试图将 FS 后端汇集到一个 Erasure Set。
    */
   if err = checkFormatErasureValues(formatConfigs, storageDisks, setDriveCount); err != nil {
      return nil, nil, err
   }

   // All disks report unformatted we should initialized everyone.
   if shouldInitErasureDisks(sErrs) && firstDisk {
      format, err = initFormatErasure(GlobalContext, storageDisks, setCount, setDriveCount, deploymentID, distributionAlgo, sErrs)
      globalDeploymentID = format.ID
      return storageDisks, format, nil
   }

   /* 
    * 检查未格式化的磁盘是否超过写入法定人数（(len(errs)/2)+1）
    * 如果有问题返回具体的错误
    */
   unformattedDisks := quorumUnformattedDisks(sErrs)
   if unformattedDisks && !firstDisk {
      return nil, nil, errNotFirstDisk
   }
   if unformattedDisks && firstDisk {
      return nil, nil, errFirstDiskWait
   }

   // 标记所有根磁盘不让使用
   markRootDisksAsDown(storageDisks, sErrs)

   // Following function is added to fix a regressions which was introduced
   // in release RELEASE.2018-03-16T22-52-12Z after migrating v1 to v2 to v3.
   // This migration failed to capture '.This' field properly which indicates
   // the disk UUID association. Below function is called to handle and fix
   // this regression, for more info refer https://github.com/minio/minio/issues/5667
   // bugFix:我们可以先忽略
   if err = fixFormatErasureV3(storageDisks, endpoints, formatConfigs); err != nil {
      logger.LogIf(GlobalContext, err)
      return nil, nil, err
   }

   // If any of the .This field is still empty, we return error.
   if formatErasureV3ThisEmpty(formatConfigs) {
      return nil, nil, errErasureV3ThisEmpty
   }

   format, err = getFormatErasureInQuorum(formatConfigs)
   if err != nil {
      logger.LogIf(GlobalContext, err)
      return nil, nil, err
   }

   if format.ID == "" {
      // Not a first disk, wait until first disk fixes deploymentID
      if !firstDisk {
         return nil, nil, errNotFirstDisk
      }
      if err = formatErasureFixDeploymentID(endpoints, storageDisks, format); err != nil {
         logger.LogIf(GlobalContext, err)
         return nil, nil, err
      }
   }

   globalDeploymentID = format.ID

   if err = formatErasureFixLocalDeploymentID(endpoints, storageDisks, format); err != nil {
      logger.LogIf(GlobalContext, err)
      return nil, nil, err
   }

   return storageDisks, format, nil
}
```



## 接口初始化

initStorageDisksWithErrors 会并行初始化所有磁盘接口。

```go
func initStorageDisksWithErrors(endpoints Endpoints) ([]StorageAPI, []error) {
   // Bootstrap disks.
   storageDisks := make([]StorageAPI, len(endpoints))
   g := errgroup.WithNErrs(len(endpoints))
   for index := range endpoints {
      index := index
      g.Go(func() (err error) {
         storageDisks[index], err = newStorageAPI(endpoints[index])
         return err
      }, index)
   }
   return storageDisks, g.Wait()
}
```

newStorageAPI 根据磁盘类型网络或本地，初始化 StorageAPI，StorageAPI 是一个接口，其中包括的对磁盘的各种操作，如状态获取、修复、读写操作等等。

```go
func newStorageAPI(endpoint Endpoint) (storage StorageAPI, err error) {
   if endpoint.IsLocal {
      storage, err := newXLStorage(endpoint)
      if err != nil {
         return nil, err
      }
      return newXLStorageDiskIDCheck(storage), nil
   }

   return newStorageRESTClient(endpoint, true), nil
}
```

本地磁盘接口初始化。newXLStorage 返回 xlStorage 结构，如果已经存在的话，则会清理一些旧数据，更新 format 到最新的格式。newXLStorageDiskIDCheck 返回 xlStorageDiskIDCheck 结构，增加磁盘状态检测。

```go
func newXLStorageDiskIDCheck(storage *xlStorage) *xlStorageDiskIDCheck {
   xl := xlStorageDiskIDCheck{
      storage: storage,
      health:  newDiskHealthTracker(),
   }
   for i := range xl.apiLatencies[:] {
      xl.apiLatencies[i] = &lockedLastMinuteLatency{}
   }
   return &xl
}
```



## 元数据初始化

初始化完对磁盘的接口后，接下来进行元数据的初始化。这里其实和 FS 模式也没什么不同，就是记录的信息更多了一些。它需要记录 Set 的信息以便在故障时进行恢复。

```go
type formatErasureV3 struct {
   formatMetaV1
   Erasure struct {
      Version string `json:"version"` // Version of 'xl' format.
      This    string `json:"this"`    // This field carries assigned disk uuid.
      // Sets field carries the input disk order generated the first
      // time when fresh disks were supplied, it is a two dimensional
      // array second dimension represents list of disks used per set.
      Sets [][]string `json:"sets"`
      // Distribution algorithm represents the hashing algorithm
      // to pick the right set index for an object.
      DistributionAlgo string `json:"distributionAlgo"`
   } `json:"xl"`
}
```

首先通过 newFormatErasureV3 分配元数据，里面包括了所有 Set 的 UUID，然后再分配下去。分配完毕之后调用 saveFormatErasureAll 将元数据持久化。

```go
func initFormatErasure(ctx context.Context, storageDisks []StorageAPI, setCount, setDriveCount int, 
                       deploymentID, distributionAlgo string, sErrs []error) (*formatErasureV3, error) {
   format := newFormatErasureV3(setCount, setDriveCount)
   formats := make([]*formatErasureV3, len(storageDisks))
   wantAtMost := ecDrivesNoConfig(setDriveCount)

   for i := 0; i < setCount; i++ {
      hostCount := make(map[string]int, setDriveCount)
      for j := 0; j < setDriveCount; j++ {
         disk := storageDisks[i*setDriveCount+j]
         newFormat := format.Clone()
         newFormat.Erasure.This = format.Erasure.Sets[i][j]
         if distributionAlgo != "" {
            newFormat.Erasure.DistributionAlgo = distributionAlgo
         }
         if deploymentID != "" {
            newFormat.ID = deploymentID
         }
         hostCount[disk.Hostname()]++
         formats[i*setDriveCount+j] = newFormat
      }
      ...
   }

   // Mark all root disks down
   markRootDisksAsDown(storageDisks, sErrs)

   // Save formats `format.json` across all disks.
   if err := saveFormatErasureAll(ctx, storageDisks, formats); err != nil {
      return nil, err
   }

   return getFormatErasureInQuorum(formats)
}
```





---

# Set 初始化

当单个存储池的磁盘初始化完毕之后，接下来初始化池中的 Set 组，用 erasureSets 结构表示。

1.  首先分配 erasureSets 结构体
2.  分配缓存池，因为对象会切割成固定大小处理，所以我们可以准备好内存以免经常分配释放影响性能
3.  填充 erasureSets，包括存储池号，Set 数目，每个 Set 包含的 Drive 数目，元数据信息，EC 比例，以及保存表示每个 Set 的 erasureObjects 对象，其中包含了其编号及磁盘操作接口。
4.  最后开启几个 routine
    -   `cleanupStaleUploads` 清理 old multipart uploads
    -   `cleanupDeletedObjects` 每隔5分钟清理 `.trash/` 文件夹
    -   `monitorAndConnectEndpoints` 进行磁盘状态监控，在后面分析数据恢复时我们再细讲





---

# 所有存储池初始化

Init() 初始化 pool，并在 `pool.bin` 中保存关于它们的额外信息用于退役存储池。整个流程非常简单，如果没有找到元数据，则生成。

```go
func (z *erasureServerPools) Init(ctx context.Context) error {
	meta := poolMeta{}

	meta.load(ctx, z.serverPools[0], z.serverPools);

	update, err := meta.validate(z.serverPools)

	// if no update is needed return right away.
	if !update {
		z.poolMeta = meta

		// 目前只支持单池退役
		for _, pool := range meta.returnResumablePools(1) {
			...
		}

		return nil
	}

	meta = poolMeta{}
	meta.Version = poolMetaVersion
	for idx, pool := range z.serverPools {
		meta.Pools = append(meta.Pools, PoolStatus{
			CmdLine:    pool.endpoints.CmdLine,
			ID:         idx,
			LastUpdate: UTCNow(),
		})
	}
	if err = meta.save(ctx, z.serverPools); err != nil {
		return err
	}
	z.poolMeta = meta
	return nil
}
```











