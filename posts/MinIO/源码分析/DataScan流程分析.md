# NSScanner

前面我们分析布隆过滤器时提到 NSScanner，接下来我们看 NSScanner 里到底在扫描什么东西？

```go
func (z *erasureServerPools) NSScanner(ctx context.Context, bf *bloomFilter, updates chan<- DataUsageInfo, wantCycle uint32) error {
   ...

   allBuckets, err := z.ListBuckets(ctx)
   ...

   if len(allBuckets) == 0 {
      updates <- DataUsageInfo{} // no buckets found update data usage to reflect latest state
      return nil
   }

   // Scanner latest allBuckets first.
   sort.Slice(allBuckets, func(i, j int) bool {
      return allBuckets[i].Created.After(allBuckets[j].Created)
   })

   // Collect for each set in serverPools.
   for _, z := range z.serverPools {
      for _, erObj := range z.sets {
         wg.Add(1)
         results = append(results, dataUsageCache{})
         go func(i int, erObj *erasureObjects) {
            updates := make(chan dataUsageCache, 1)
            defer close(updates)
            // Start update collector.
            go func() {
               defer wg.Done()
               // 扫描结果放入 result
               for info := range updates {
                  mu.Lock()
                  results[i] = info
                  mu.Unlock()
               }
            }()
            // Start scanner. Blocks until done.
            err := erObj.nsScanner(ctx, allBuckets, bf, wantCycle, updates)
            ...
         }(len(results)-1, erObj)
      }
   }
    
   updateCloser := make(chan chan struct{})
   go func() {
      updateTicker := time.NewTicker(30 * time.Second)
      defer updateTicker.Stop()
      var lastUpdate time.Time

      // 桶存在于每个 drive 上。因此，为了得到准确的桶的大小，我们必须在转换之前进行合并。
      ...
   }()

   wg.Wait()
   ...
   return firstErr
}
```

对于这一长串代码，其实我们只需要关心 `erObj.nsScanner` 在干什么就好了，在对每个池的每个 set 进行 nsScanner 之后，结果保存在了 results中，因为每个池或者说每个 drive 都包含相同的桶，所以后面需要对结果进行一些合并处理。

set 的 nsScanner 我们分为三部分来看，第一部分是初始化，第二部分是 dataUsageCache 处理，第三部分是磁盘扫描。

```go
func (er erasureObjects) nsScanner(ctx context.Context, buckets []BucketInfo, bf *bloomFilter, wantCycle uint32, updates chan<- dataUsageCache) error {
   if len(buckets) == 0 {
      return nil
   }

   // Collect disks we can use.
   // 故障和正在修复的磁盘不去扫描
   disks, healing := er.getOnlineDisksWithHealing()
   if len(disks) == 0 {
      logger.LogIf(ctx, errors.New("data-scanner: all disks are offline or being healed, skipping scanner cycle"))
      return nil
   }

   // Load bucket totals
   oldCache := dataUsageCache{}
   if err := oldCache.load(ctx, er, dataUsageCacheName); err != nil {
      return err
   }

   // New cache..
   cache := dataUsageCache{
      Info: dataUsageCacheInfo{
         Name:      dataUsageRoot,
         NextCycle: oldCache.Info.NextCycle,
      },
      Cache: make(map[string]dataUsageEntry, len(oldCache.Cache)),
   }
   bloom := bf.bytes()

   // 把所有的 buckets 放入 bucketCh
   bucketCh := make(chan BucketInfo, len(buckets))
   // Add new buckets first
   for _, b := range buckets {
      if oldCache.find(b.Name) == nil {
         bucketCh <- b
      }
   }
   // Add existing buckets.
   for _, b := range buckets {
      e := oldCache.find(b.Name)
      if e != nil {
         cache.replace(b.Name, dataUsageRoot, *e)
         bucketCh <- b
      }
   }
   close(bucketCh)
    
   bucketResults := make(chan dataUsageEntryInfo, len(disks))
   // Start async collector/saver.
   // This goroutine owns the cache.
   var saverWg sync.WaitGroup
   saverWg.Add(1)
   go func() {
      // dataUsageCache 处理
      ...
   }()

   // Shuffle disks to ensure a total randomness of bucket/disk association to ensure
   // that objects that are not present in all disks are accounted and ILM applied.
   r := rand.New(rand.NewSource(time.Now().UnixNano()))
   r.Shuffle(len(disks), func(i, j int) { disks[i], disks[j] = disks[j], disks[i] })

   var wg sync.WaitGroup
   wg.Add(len(disks))
   for i := range disks {
      go func(i int) {
         // Start one scanner per disk
         ...
      }(i)
   }
   wg.Wait()
   close(bucketResults)
   saverWg.Wait()

   return nil
}
```

我们主要关心磁盘数据扫描的部分。对于每个磁盘，我们都会进行如下扫描，这里要注意的是，首先 Shuffle 把磁盘顺序打乱了，然后每个桶扫描的时候是在打乱的磁盘列表中选一个去扫描的，这就表示扫描哪个磁盘的哪个桶完全是随机的，有点像抽查作业，抽查某个桶在某个磁盘上是否完好，好处就是不会一下占用太多资源，坏处就是你也不知道要恢复的数据什么时候能扫描到。

```go
go func(i int) {
   defer wg.Done()
   disk := disks[i]

   // 每次只随机扫描某个磁盘的某个桶
   for bucket := range bucketCh {
      select {
      case <-ctx.Done():
         return
      default:
      }

      // Load cache for bucket
      cacheName := pathJoin(bucket.Name, dataUsageCacheName)
      cache := dataUsageCache{}
      logger.LogIf(ctx, cache.load(ctx, er, cacheName))
      if cache.Info.Name == "" {
         cache.Info.Name = bucket.Name
      }
      cache.Info.BloomFilter = bloom
      cache.Info.SkipHealing = healing
      cache.Info.NextCycle = wantCycle
      if cache.Info.Name != bucket.Name {
         // 如果桶名变更了就不使用布隆过滤器而是全部扫描了
         logger.LogIf(ctx, fmt.Errorf("cache name mismatch: %s != %s", cache.Info.Name, bucket.Name))
         cache.Info = dataUsageCacheInfo{
            Name:       bucket.Name,
            LastUpdate: time.Time{},
            NextCycle:  wantCycle,
         }
      }
      // Collect updates.
      updates := make(chan dataUsageEntry, 1)
      var wg sync.WaitGroup
      wg.Add(1)
      go func(name string) {
         defer wg.Done()
         // 对变更的信息发送到 bucketResults
         for update := range updates {
            bucketResults <- dataUsageEntryInfo{
               Name:   name,
               Parent: dataUsageRoot,
               Entry:  update,
            }
         }
      }(cache.Info.Name)
      // Calc usage
      before := cache.Info.LastUpdate
      var err error
      // 扫描磁盘数据
      cache, err = disk.NSScanner(ctx, cache, updates)
      cache.Info.BloomFilter = nil
      ...

      wg.Wait()
      ...
   }
}(i)
```

这里同样一长串代码看着眼花，其实我们重点关注的地方还是一个，那就是 `disk.NSScanner`，其余部分的架构都很类似，都是开启一个协程来从通道里获取结果，通道里的数据都是由扫描协程塞入的，所以我们继续往里探索这一层又一层的 scan。

```go
func (p *xlStorageDiskIDCheck) NSScanner(ctx context.Context, cache dataUsageCache, updates chan<- dataUsageEntry) (dataUsageCache, error) {
   if contextCanceled(ctx) {
      return dataUsageCache{}, ctx.Err()
   }

   if err := p.checkDiskStale(); err != nil {
      return dataUsageCache{}, err
   }
   return p.storage.NSScanner(ctx, cache, updates)
}
```

接下来就是 scan 的实际操作了。首先检查是否有生命周期策略，是否有复制配置等等。然后就是 scanDataFolder。

```go
func (s *xlStorage) NSScanner(ctx context.Context, cache dataUsageCache, updates chan<- dataUsageEntry) (dataUsageCache, error) {
   // Updates must be closed before we return.
   defer close(updates)
   var lc *lifecycle.Lifecycle
   var err error

   // Check if the current bucket has a configured lifecycle policy
   if globalLifecycleSys != nil {
      ...
   }

   // Check if the current bucket has replication configuration
   if rcfg, err := globalBucketMetadataSys.GetReplicationConfig(ctx, cache.Info.Name); err == nil {
      ...
   }
   // return initialized object layer
   objAPI := newObjectLayerFn()
   ...

   cache.Info.updates = updates

   poolIdx, setIdx, _ := s.GetDiskLoc()

   dataUsageInfo, err := scanDataFolder(ctx, poolIdx, setIdx, s.diskPath, cache, func(item scannerItem) (sizeSummary, error) {
      ...
   })
   if err != nil {
      return dataUsageInfo, err
   }

   dataUsageInfo.Info.LastUpdate = time.Now()
   return dataUsageInfo, nil
}
```

我们看看 scanDataFolder 是怎么扫描目录的。这里传入了一个匿名函数，这个函数比较长，后面分析的时候再贴出来。这里初始化了 folderScanner 结构体，并把传入的匿名函数赋给了 getSize 字段，然后又跳入 s.scanFolder（还没进入主题，我都快被跳烦了……）

```go
func scanDataFolder(ctx context.Context, poolIdx, setIdx int, basePath string, cache dataUsageCache, getSize getSizeFn) 
(dataUsageCache, error) {
   ...

   s := folderScanner{
      root:                  basePath,
      getSize:               getSize,
      oldCache:              cache,
      newCache:              dataUsageCache{Info: cache.Info},
      updateCache:           dataUsageCache{Info: cache.Info},
      dataUsageScannerDebug: intDataUpdateTracker.debug,
      healFolderInclude:     0,
      healObjectSelect:      0,
      updates:               cache.Info.updates,
   }

   // Add disks for set healing.
   s.disks = objAPI.serverPools[poolIdx].sets[setIdx].getDisks()
   s.disksQuorum = len(s.disks) / 2

   ...
   root := dataUsageEntry{}
   folder := cachedFolder{name: cache.Info.Name, objectHealProbDiv: 1}
   err := s.scanFolder(ctx, folder, &root)

   ...
   s.newCache.Info.LastUpdate = UTCNow()
   s.newCache.Info.NextCycle = cache.Info.NextCycle
   return s.newCache, nil
}
```

先不看了，头大了……







---

# 参考与感谢

-   [add data update tracking using Bloom filter](https://github.com/minio/minio/pull/9208)



