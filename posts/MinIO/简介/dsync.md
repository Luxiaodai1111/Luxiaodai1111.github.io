# 概述

[minio/dsync](https://github.com/minio/dsync) 是一个 go 语言实现的，分布式锁工具库，其设计宗旨是追求简单，因此横向扩展能力比较局限，通常小于 32 个分布式节点。任一节点会向全部节点广播锁请求消息，获得 `n/2 + 1` 赞同的节点会成功获取到锁。释放锁时，会向全部节点广播请求。

该软件包是为 MinIO 对象存储的分布式服务器版本开发的。为此，我们需要一个分布式锁定机制，最多用于 32 台服务器，每台服务器都将运行 `minio server`。锁定机制本身应该是一个读/写互斥锁，意味着它可以由一个写或任意数量的读持有。



---

# 设计目标

-   简单设计：通过保持简单的设计，可以避免许多棘手的边缘情况。
-   没有主节点：没有主节点的概念，如果使用主节点，而主节点发生故障，会导致锁定完全停止。除非你有一个带有从属节点的设计，但这增加了更多的复杂性。
-   弹性容错：如果一个或多个节点发生故障，其他节点不应受到影响，可以继续获得锁（只要不超过 `n/2-1` 个节点发生故障）。
-   自动化重组：宕机节点自动重新连接到（重新启动的）节点。



---

# 限制

-   有限的可扩展性：最多 32 个节点。
-   固定配置：dsync 的实现并不能想以往的 raft、gossip 那样动态添加、删除节点、更新节点信息。如果需要变更集群的配置，需要修改、重启集群内全部节点才能生效。
-   不是为高性能应用设计的，如键/值存储。



---

# 性能

作为一个分布式锁，其性能也至关重要，因为其很可能是一个高频操作。官方给出的数据是：

-   在适度强大的服务器硬件上，对于 16 个节点的规模，支持总共 7500 个锁/秒（每台服务器消耗 10% 的 CPU 使用率）。
-   锁定请求（成功）的时间不应超过 1ms（如果节点之间有 1 Gbit 或更高的网络连接）。



## 不同节点数量下的性能

| EC2 Instance Type | Nodes | Locks/server/sec     | Total Locks/sec | CPU Usage |
| ----------------- | ----- | -------------------- | --------------- | --------- |
| c3.2xlarge        | 4     | (min=3110, max=3376) | 12972           | 25%       |
| c3.2xlarge        | 8     | (min=1884, max=2096) | 15920           | 25%       |
| c3.2xlarge        | 12    | (min=1239, max=1558) | 16782           | 25%       |
| c3.2xlarge        | 16    | (min=996, max=1391)  | 19096           | 25%       |

最小和最大 Locks/server/sec 逐渐下降，但由于节点数量增加，在相同的 CPU 使用水平下锁的总数稳步上升。



## 不同实例类型的性能

| EC2 Instance Type    | Nodes | Locks/server/sec     | Total Locks/sec | CPU Usage |
| -------------------- | ----- | -------------------- | --------------- | --------- |
| c3.large (2 vCPU)    | 8     | (min=823, max=896)   | 6876            | 75%       |
| c3.2xlarge (8 vCPU)  | 8     | (min=1884, max=2096) | 15920           | 25%       |
| c3.8xlarge (32 vCPU) | 8     | (min=2601, max=2898) | 21996           | 10%       |

随着内核数量的增加，CPU 的负载减少，整体性能提高。



## 压力测试

| EC2 Instance Type   | Nodes | Locks/server/sec     | Total Locks/sec | CPU Usage |
| ------------------- | ----- | -------------------- | --------------- | --------- |
| c3.8xlarge(32 vCPU) | 8     | (min=2601, max=2898) | 21996           | 10%       |
| c3.8xlarge(32 vCPU) | 8     | (min=4756, max=5227) | 39932           | 20%       |
| c3.8xlarge(32 vCPU) | 8     | (min=7979, max=8517) | 65984           | 40%       |
| c3.8xlarge(32 vCPU) | 8     | (min=9267, max=9469) | 74944           | 50%       |

在 50% 的 CPU 负载下，该系统可以达到 75K Locks/sec。



---

# 使用

## 互斥锁

```go
import (
	"github.com/minio/dsync/v3"
)

func lockSameResource() {

	// Create distributed mutex to protect resource 'test'
	dm := dsync.NewDRWMutex(context.Background(), "test", ds)

	dm.Lock("lock-1", "example.go:505:lockSameResource()")
	log.Println("first lock granted")

	// Release 1st lock after 5 seconds
	go func() {
		time.Sleep(5 * time.Second)
		log.Println("first lock unlocked")
		dm.Unlock()
	}()

	// Try to acquire lock again, will block until initial lock is released
	log.Println("about to lock same resource again...")
	dm.Lock("lock-1", "example.go:515:lockSameResource()")
	log.Println("second lock granted")

	time.Sleep(2 * time.Second)
	dm.Unlock()
}
```

输出如下：

```bash
2016/09/02 14:50:00 first lock granted
2016/09/02 14:50:00 about to lock same resource again...
2016/09/02 14:50:05 first lock unlocked
2016/09/02 14:50:05 second lock granted
```



## 读锁

```go
func twoReadLocksAndSingleWriteLock() {

	drwm := dsync.NewDRWMutex(context.Background(), "resource", ds)

	drwm.RLock("RLock-1", "example.go:416:twoReadLocksAndSingleWriteLock()")
	log.Println("1st read lock acquired, waiting...")

	drwm.RLock("RLock-2", "example.go:420:twoReadLocksAndSingleWriteLock()")
	log.Println("2nd read lock acquired, waiting...")

	go func() {
		time.Sleep(1 * time.Second)
		drwm.RUnlock()
		log.Println("1st read lock released, waiting...")
	}()

	go func() {
		time.Sleep(2 * time.Second)
		drwm.RUnlock()
		log.Println("2nd read lock released, waiting...")
	}()

	log.Println("Trying to acquire write lock, waiting...")
	drwm.Lock("Lock-1", "example.go:445:twoReadLocksAndSingleWriteLock()")
	log.Println("Write lock acquired, waiting...")

	time.Sleep(3 * time.Second)

	drwm.Unlock()
}
```

输出如下：

```bash
2016/09/02 15:05:20 1st read lock acquired, waiting...
2016/09/02 15:05:20 2nd read lock acquired, waiting...
2016/09/02 15:05:20 Trying to acquire write lock, waiting...
2016/09/02 15:05:22 1st read lock released, waiting...
2016/09/02 15:05:24 2nd read lock released, waiting...
2016/09/02 15:05:24 Write lock acquired, waiting...
```



---

# 实现

由于没有动态成员变更的设计，其实现就非常的简单，基本上就是一个 `FOR-EACH` 框架：

## 获取锁

-   向全部节点广播获取锁的请求消息
-   在超时时间内收集每个节点的回复信息
-   如果获得 `n/2 + 1` 个节点的赞同，则获得锁
-   否则，广播释放消息，然后在等待一个随机的延时后，再次尝试

## 释放锁

-   向全部节点广播释放锁的消息
-   如果某一节点通信失败，再次尝试
-   忽略"结果"（目标节点已经故障并恢复的情况）。



---

# stale lock

持有锁的实例已经宕机，或者由于网络故障造成锁释放的消息无法被送达。在分布式系统中，stale lock 不是那么容易被检测到的，其会大大影响整个系统的效率。因此 dsync 中加入了 stale lock [检测机制](https://github.com/minio/dsync/pull/22#issue-176751755)：其首先假设每个节点本地不存在网络故障，因此每个节点本地记录着正确的自身节点的锁持有情况，然后其他节点，周期性调用锁拥有者节点的检验接口，查询对应的锁是否过期。



---

# 已知缺陷

虽然一个节点需要得到 `n/2 + 1` 个节点的同意才能获得锁，但是在这期间，有些节点可能会宕机重启，它们可能会在宕机前、宕机后分别同意不同节点的请求，造成它们都获得了 `n/2 + 1` 个同意回复，造成它们同时获得排他锁。

​	
