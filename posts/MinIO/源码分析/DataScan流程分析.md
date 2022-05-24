# NSScanner

前面我们分析布隆过滤器时提到 NSScanner，接下来我们看 NSScanner 里到底在扫描什么东西？

```go
func (z *erasureServerPools) NSScanner(ctx context.Context, bf *bloomFilter, updates chan<- DataUsageInfo, wantCycle uint32) error {
   ...

   // 列出所有的桶
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
            // 开始扫描，会阻塞到完成
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

对于这一长串代码，其实我们只需要关心 `erObj.nsScanner` 在干什么就好了，在对每个池的每个 set 进行 nsScanner 之后，结果保存在了 results 中，因为每个池或者说每个 drive 都包含相同的桶，所以后面需要对结果进行一些合并处理。

set 的 nsScanner 我们分为三部分来看，第一部分是初始化，第二部分是 dataUsageCache 处理，第三部分是磁盘扫描。

```go
func (er erasureObjects) nsScanner(ctx context.Context, buckets []BucketInfo, bf *bloomFilter, wantCycle uint32, updates chan<- dataUsageCache) error {
   ...
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

我们看看 scanDataFolder 是怎么扫描目录的。这里传入了一个匿名函数，这个函数比较长，后面分析的时候再贴出来。scanDataFolder 初始化了 folderScanner 结构体，并把传入的匿名函数赋给了 getSize 字段，然后又跳入 s.scanFolder（还没进入主题，我都快被跳烦了……）

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

下面来介绍真正干活的部分。





---

# scanFolder

scanFolder 代码精简如下：

```go
func (f *folderScanner) scanFolder(ctx context.Context, folder cachedFolder, into *dataUsageEntry) error {
   done := ctx.Done()
   scannerLogPrefix := color.Green("folder-scanner:")
   thisHash := hashPath(folder.name)
   // Store initial compaction state.
   wasCompacted := into.Compacted
   atomic.AddUint64(&globalScannerStats.accFolders, 1)

   for {
      select {
      case <-done:
         return ctx.Err()
      default:
      }
      existing, ok := f.oldCache.Cache[thisHash.Key()]
      var abandonedChildren dataUsageHashMap
      if !into.Compacted {
         abandonedChildren = f.oldCache.findChildrenCopy(thisHash)
      }

      // 如果该前缀有生命周期规则，则删除过滤器
      filter := f.withFilter
      _, prefix := path2BucketObjectWithBasePath(f.root, folder.name)
      var activeLifeCycle *lifecycle.Lifecycle
      if f.oldCache.Info.lifeCycle != nil && f.oldCache.Info.lifeCycle.HasActiveRules(prefix, true) {
         activeLifeCycle = f.oldCache.Info.lifeCycle
         filter = nil
      }
       
      // 如果该前缀有复制规则，删除过滤器
      var replicationCfg replicationConfig
      if !f.oldCache.Info.replication.Empty() && f.oldCache.Info.replication.Config.HasActiveRules(prefix, true) {
         replicationCfg = f.oldCache.Info.replication
         filter = nil
      }
       
      // 检查是否可以根据 bloom filter 跳过扫描
      if filter != nil && ok && existing.Compacted {
         // If folder isn't in filter and we have data, skip it completely.
         ...
      }
      scannerSleeper.Sleep(ctx, dataScannerSleepPerFolder)

      var existingFolders, newFolders []cachedFolder
      var foundObjects bool
      // 对目录里的每个条目应用 fn 函数，不对目录本身进行递归，如果 dirPath 不存在，这个函数不会返回错误。
      err := readDirFn(path.Join(f.root, folder.name), func(entName string, typ os.FileMode) error {
         ...
      })
      if err != nil {
         return err
      }

      if foundObjects && globalIsErasure {
         // If we found an object in erasure mode, we skip subdirs (only datadirs)...
         break
      }

      // 如果我们有很多子目录，就需要压缩
      if !into.Compacted &&
         f.newCache.Info.Name != folder.name &&
         len(existingFolders)+len(newFolders) >= dataScannerCompactAtFolders {
         into.Compacted = true
         newFolders = append(newFolders, existingFolders...)
         existingFolders = nil
      }

      // scanFolder 函数，后续分析
      scanFolder := func(folder cachedFolder) {
         ...
      }

      // Transfer existing
      if !into.Compacted {
         for _, folder := range existingFolders {
            h := hashPath(folder.name)
            f.updateCache.copyWithChildren(&f.oldCache, h, folder.parent)
         }
      }
      // Scan new...
      for _, folder := range newFolders {
         ...
         scanFolder(folder)
         ...
      }

      // Scan existing...
      for _, folder := range existingFolders {
         ...
         scanFolder(folder)
         ...
      }

      // Scan for healing
      if f.healObjectSelect == 0 || len(abandonedChildren) == 0 {
         // 没有要修复的对象, return now.
         break
      }

      objAPI, ok := newObjectLayerFn().(*erasureServerPools)
      if !ok || len(f.disks) == 0 || f.disksQuorum == 0 {
         break
      }

      bgSeq, found := globalBackgroundHealState.getHealSequenceByToken(bgHealingUUID)
      if !found {
         break
      }

      /*
       * 在'abandonedChildren'中剩下的都是这一层的文件夹，这些文件夹在之前的运行中存在，但现在没有被发现。
       * 这可能是由于两个原因。
       * 1）文件夹/对象被删除了。
       * 2）我们来自另一个磁盘，这个磁盘错过了写入。
       * 因此，我们进行了一次修复检查。如果没有恢复，我们就删除这个文件夹，并假定它被删除了。这意味着下次运行时将不会再寻找它。
       */
      resolver := metadataResolutionParams{
         dirQuorum: f.disksQuorum,
         objQuorum: f.disksQuorum,
         bucket:    "",
         strict:    false,
      }

      healObjectsPrefix := color.Green("healObjects:")
      for k := range abandonedChildren {
         // 相关操作，后续分析
         ...
      }
      break
   }
   // compact 相关操作
   ...

   return nil
}
```

readDirFn 传入的匿名函数如下：

```go
func(entName string, typ os.FileMode) error {
   // Parse
   entName = pathClean(path.Join(folder.name, entName))
   if entName == "" || entName == folder.name {
      if f.dataUsageScannerDebug {
         console.Debugf(scannerLogPrefix+" no entity (%s,%s)\n", f.root, entName)
      }
      return nil
   }
   bucket, prefix := path2BucketObjectWithBasePath(f.root, entName)
   if bucket == "" {
      if f.dataUsageScannerDebug {
         console.Debugf(scannerLogPrefix+" no bucket (%s,%s)\n", f.root, entName)
      }
      return errDoneForNow
   }

   if isReservedOrInvalidBucket(bucket, false) {
      if f.dataUsageScannerDebug {
         console.Debugf(scannerLogPrefix+" invalid bucket: %v, entry: %v\n", bucket, entName)
      }
      return errDoneForNow
   }

   select {
   case <-done:
      return errDoneForNow
   default:
   }

   if typ&os.ModeDir != 0 {
      h := hashPath(entName)
      _, exists := f.oldCache.Cache[h.Key()]
      if h == thisHash {
         return nil
      }
      this := cachedFolder{name: entName, parent: &thisHash, objectHealProbDiv: folder.objectHealProbDiv}
      delete(abandonedChildren, h.Key()) // h.Key() already accounted for.
      if exists {
         existingFolders = append(existingFolders, this)
         f.updateCache.copyWithChildren(&f.oldCache, h, &thisHash)
      } else {
         newFolders = append(newFolders, this)
      }
      return nil
   }

   // Dynamic time delay.
   wait := scannerSleeper.Timer(ctx)

   // Get file size, ignore errors.
   item := scannerItem{
      Path:        path.Join(f.root, entName),
      Typ:         typ,
      bucket:      bucket,
      prefix:      path.Dir(prefix),
      objectName:  path.Base(entName),
      debug:       f.dataUsageScannerDebug,
      lifeCycle:   activeLifeCycle,
      replication: replicationCfg,
      heal:        thisHash.modAlt(f.oldCache.Info.NextCycle/folder.objectHealProbDiv, f.healObjectSelect/folder.objectHealProbDiv) && globalIsErasure,
   }
   // if the drive belongs to an erasure set
   // that is already being healed, skip the
   // healing attempt on this drive.
   item.heal = item.heal && f.healObjectSelect > 0

   sz, err := f.getSize(item)
   if err != nil {
      wait() // wait to proceed to next entry.
      if err != errSkipFile && f.dataUsageScannerDebug {
         console.Debugf(scannerLogPrefix+" getSize \"%v/%v\" returned err: %v\n", bucket, item.objectPath(), err)
      }
      return nil
   }

   // successfully read means we have a valid object.
   foundObjects = true
   // Remove filename i.e is the meta file to construct object name
   item.transformMetaDir()

   // Object already accounted for, remove from heal map,
   // simply because getSize() function already heals the
   // object.
   delete(abandonedChildren, path.Join(item.bucket, item.objectPath()))

   into.addSizes(sz)
   into.Objects++

   wait() // wait to proceed to next entry.

   return nil
})
```

scanFolder 递归调用 f.scanFolder，头大……

```go
scanFolder := func(folder cachedFolder) {
   if contextCanceled(ctx) {
      return
   }
   dst := into
   if !into.Compacted {
      dst = &dataUsageEntry{Compacted: false}
   }
   if err := f.scanFolder(ctx, folder, dst); err != nil {
      logger.LogIf(ctx, err)
      return
   }
   if !into.Compacted {
      h := dataUsageHash(folder.name)
      into.addChild(h)
      // We scanned a folder, optionally send update.
      f.updateCache.deleteRecursive(h)
      f.updateCache.copyWithChildren(&f.newCache, h, folder.parent)
      f.sendUpdate()
   }
}
```





---

# 参考与感谢

-   [add data update tracking using Bloom filter](https://github.com/minio/minio/pull/9208)



