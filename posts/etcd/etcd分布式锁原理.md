# 分布式锁问题和特性

问题：

- 死锁：比如 A，B 进程，A 进程获取锁后异常崩溃没有释放锁，那么 B 进程就会陷入无限等待的过程中，造成死锁；或者 A 进程获取锁，但是过程中依赖 B 进程，B 进程也需要获取锁，但是此时锁已经被 A 进程获取，造成互相等待而死锁
- 惊群效应：多线程/多进程等待同一个锁释放，当这个事件发生时，这些线程/进程被同时唤醒，就是惊群
- 脑裂：当集群中出现脑裂的时候，往往会出现多个 master 的情况，这样数据的一致性会无法得到保障，从而导致整个服务无法正常运行

 

因此设计的锁要满足以下特性：

- 高可用：也就是可靠性。某一台机器锁不能提供服务了，其他机器仍然可以提供锁服务。
- 互斥性：就像单机系统的锁特性一样具有互斥性。不过分布式系统是由多个机器节点组成的。如果有一个节点获取了锁，其他节点必须等待锁释放或者锁超时后，才可以去获取锁资源。
- 可重入：一个节点获取了锁之后，还可以再次获取整个锁资源而不会死锁。
- 高效：高效是指获取和释放锁高效。 
- 安全性：锁超时，防止死锁的发生，即安全性。
- 公平锁：节点依次获取锁资源，否则可能有些进程一直抢不到锁导致饿死。



---

# etcd如何实现分布式锁

etcd 是怎么解决上面这些问题？它提供了哪些功能来解决上述的特性。

- raft

raft，是工程上使用较为广泛，强一致性、去中心化、高可用的分布式协议。raft 提供了分布式系统的可靠性功能。

- lease功能

> lease 功能，就是租约机制(time to live)。

1、etcd 可以对存储 key-value 的数据设置租约，也就是给 key-value 设置一个过期时间，当租约到期，key-value 将会失效而被 etcd 删除；

2、etcd 同时也支持续约租期，可以通过客户端在租约到期之间续约，以避免 key-value 失效；

3、etcd 还支持解约，一旦解约，与该租约绑定的 key-value 将会失效而删除。

> Lease 功能可以保证分布式锁的安全性，为锁对应的 key 配置租约，即使锁的持有者因故障而不能主动释放锁，锁也会因租约到期而自动释放。

- watch功能

> 监听功能。watch 机制支持监听某个固定的key，它也支持 watch 一个范围（前缀机制），当被 watch 的 key 或范围发生变化时，客户端将收到通知。

在实现分布式锁时，如果抢锁失败，可通过 Prefix 机制返回的 KeyValue 列表获得 Revision 比自己小且相差最小的 key（称为 pre-key），对 pre-key 进行监听，因为只有它释放锁，自己才能获得锁，如果 Watch 到 pre-key 的 DELETE 事件，则说明 pre-key 已经释放，自己已经持有锁。

- prefix功能

> 前缀机制。也称目录机制，如两个 key 命名如下：key1=“/mykey/key1" ， key2="/mykey/key2"，那么，可以通过前缀-“/mykey"查询，返回包含两个 key-value 对的列表。可以和前面的 watch 功能配合使用。

例如，一个名为 /mylock 的锁，两个争抢它的客户端进行写操作，实际写入的 key 分别为：key1="/mylock/UUID1"，key2="/mylock/UUID2"，其中，UUID 表示全局唯一的 ID，确保两个 key 的唯一性。很显然，写操作都会成功，但返回的 Revision 不一样，那么，如何判断谁获得了锁呢？

通过前缀 /mylock 查询，返回包含两个 key-value 对的的 KeyValue 列表，同时也包含它们的 Revision，通过 Revision 大小，客户端可以判断自己是否获得锁，如果抢锁失败，则等待锁释放（对应的 key 被删除或者租约过期），然后再判断自己是否可以获得锁。

> lease 功能和 prefix功能，能解决上面的死锁问题。

- revision功能

每个 key 带有一个 Revision 号，每进行一次事务加一，因此它是全局唯一的，如初始值为 0，进行一次 put(key，value)，key 的 Revision 变为 1；同样的操作，再进行一次，Revision 变为 2；换成 key1 进行 put(key1，value) 操作，Revision 将变为 3。

这种机制有一个作用：

> 通过 Revision 的大小就可以知道进行写操作的顺序。在实现分布式锁时，多个客户端同时抢锁，根据 Revision 号大小依次获得锁，可以避免 “羊群效应" （也称 “惊群效应"），实现公平锁。



---

# 源码分析

etcd 的 v3client 里有一个 concurrency 的包，里面实现了分布式锁。 下面先给出一个官方示例：

```go
func ExampleMutex_Lock() {
    // 获取 etcd 客户端
	cli， err := clientv3.New(clientv3.Config{Endpoints: endpoints})
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close()

	// create two separate sessions for lock competition
	s1， err := concurrency.NewSession(cli)
	if err != nil {
		log.Fatal(err)
	}
	defer s1.Close()
	m1 := concurrency.NewMutex(s1， "/my-lock/")

	s2， err := concurrency.NewSession(cli)
	if err != nil {
		log.Fatal(err)
	}
	defer s2.Close()
	m2 := concurrency.NewMutex(s2， "/my-lock/")

	// acquire lock for s1
	if err := m1.Lock(context.TODO()); err != nil {
		log.Fatal(err)
	}
	fmt.Println("acquired lock for s1")

	m2Locked := make(chan struct{})
	go func() {
		defer close(m2Locked)
		// wait until s1 is locks /my-lock/
		if err := m2.Lock(context.TODO()); err != nil {
			log.Fatal(err)
		}
	}()

	if err := m1.Unlock(context.TODO()); err != nil {
		log.Fatal(err)
	}
	fmt.Println("released lock for s1")

	<-m2Locked
	fmt.Println("acquired lock for s2")

	// Output:
	// acquired lock for s1
	// released lock for s1
	// acquired lock for s2
}
```

可以看到一个基础的步骤如下：

- `clientv3.New` 获取客户端
- `concurrency.NewSession` 获取session
- `concurrency.NewMutex` 获取锁
- 加锁和解锁



## NewSession

首先我们看 NewSession 函数，他接收一个操作的客户端和 sessionOptions， 一些相关的定义如下：

```go
type sessionOptions struct {
	ttl     int
	leaseID v3.LeaseID	// 租约 ID，是分布式锁超时机制的核心
	ctx     context.Context
}

// 匿名函数会被调用来修改 sessionOptions 的值
func WithTTL(ttl int) SessionOption {
	return func(so *sessionOptions) {
		if ttl > 0 {
			so.ttl = ttl
		}
	}
}

func WithLease(leaseID v3.LeaseID) SessionOption {
	return func(so *sessionOptions) {
		so.leaseID = leaseID
	}
}

type Session struct {
	client *v3.Client
	opts   *sessionOptions
	id     v3.LeaseID

	cancel context.CancelFunc
	donec  <-chan struct{}
}
```

下面可以看到 session 其实就是租约的一个封装，返回值是 `Session` 结构体指针

```go
// NewSession gets the leased session for a client.
func NewSession(client *v3.Client， opts ...SessionOption) (*Session， error) {
    // defaultSessionTTL 为 60，如果传递了参数，则会去设置参数
    // 比如携带参数 concurrency.WithTTL(ttl)，则会调用函数将 ops 的值改变，这里为设置 ttl
	ops := &sessionOptions{ttl: defaultSessionTTL， ctx: client.Ctx()}
	for _， opt := range opts {
		opt(ops)
	}

    // 如果直接携带了租约 ID（WithLease）则不用去创建租约了，否则使用ttl的时间去创建租约
	id := ops.leaseID
	if id == v3.NoLease {
		resp， err := client.Grant(ops.ctx， int64(ops.ttl))
		if err != nil {
			return nil， err
		}
		id = resp.ID
	}

    // 为租约续租
	ctx， cancel := context.WithCancel(ops.ctx)
	keepAlive， err := client.KeepAlive(ctx， id)
	if err != nil || keepAlive == nil {
		cancel()
		return nil， err
	}

    // donec 用于因为各种原因没有续租上时，session 提供了 Done() 方法给应用程序可以及时感知
	donec := make(chan struct{})
	s := &Session{client: client， opts: ops， id: id， cancel: cancel， donec: donec}

	// keep the lease alive until client error or cancelled context
	go func() {
		defer close(donec)
		for range keepAlive {
			// eat messages until keep alive channel closes
		}
	}()

	return s， nil
}
```



## NewMutex

NewMutex 函数本身非常简单，接收上面创建的 session 以及锁的前缀

```go
func NewMutex(s *Session， pfx string) *Mutex {
   return &Mutex{s， pfx + "/"， ""， -1， nil}
}
```

返回 `Mutex` 结构体指针

```go
type Mutex struct {
   s *Session	// 上面创建的 session

   pfx   string	// 记录这个锁的名称
   myKey string	// 锁名称和租约ID组成的键
   myRev int64	// revision
   hdr   *pb.ResponseHeader
}
```



## lock

接下来是重头戏，加锁，主要思路如下：

- `tryAcquire` 获取主键和锁拥有者的 revision
- 如果没有人竞争或者是自己之前加的锁，那么就获取锁
- 否则 `waitDeletes` 拿到 revision 比自己小的一系列 waiter 中 revision 最大的主键，并监听它的 delete 事件，当它释放时，获取锁

```go
func (m *Mutex) Lock(ctx context.Context) error {
    // 获取主键和锁拥有者的 revision
	resp， err := m.tryAcquire(ctx)
	if err != nil {
		return err
	}
   	// 如果上面 tryAcquire 操作成功，则 myRev 是当前客户端创建的 key 的 revision 值。
    // ownerKey 是 tryAcquire 里 getOwner 的结果，即锁前缀的最早版本持有者
    // 如果没有人持有锁，那么理所当然直接获取锁了
    // 如果有人持有锁，但发现是自己创建的，那么也获取到锁了
	ownerKey := resp.Responses[1].GetResponseRange().Kvs
	if len(ownerKey) == 0 || ownerKey[0].CreateRevision == m.myRev {
		m.hdr = resp.Header
		return nil
	}
    
 	client := m.s.Client()
	// TODO: early termination if the session key is deleted before other session keys with smaller revisions.
    // waitDeletes 匹配 m.pfx 这个前缀并且 revision 小于 m.myRev-1 所有 key
    // 等待比它小的所有 key 中 revision 最大的 key（即与自己 revision 相差最小）的 delete 事件
    // 监测到事件后检查自己 revision 是否最小，如果是则获取锁，不是则继续监听
	_， werr := waitDeletes(ctx， client， m.pfx， m.myRev-1)
	// release lock key if wait failed
	if werr != nil {
		m.Unlock(client.Ctx())
		return werr
	}

	// 确保 session 没过期，且主键存在（存在检查的时候刚好是过期前一刻的风险）
	gresp， werr := client.Get(ctx， m.myKey)
	if werr != nil {
		m.Unlock(client.Ctx())
		return werr
	}
	if len(gresp.Kvs) == 0 {
		return ErrSessionExpired
	}
	m.hdr = gresp.Header

	return nil
}
```



### tryAcquire

tryAcquire 是去尝试获取锁，流程如下：

- 锁的主键是 锁名+租约ID
- 检查主键是否被人创建过，`createRevision` 是表示这个key创建时被分配的这个序号。 当key不存在时，createRevision 是 0。因此判断 createRevision 就可以知道这个键存不存在，如果不存在则 `put` 写入，如果已经存在则通过 `get` 去获取。
- 最后可以通过事务进入的分支（成功表示执行了 put ，失败则是 get）去得到一个这个键创建时的 `revision`（revision 是 etcd 一个全局的序列号，全局唯一且递增，每一个对 etcd 存储进行改动都会分配一个这个序号）
- `getOwner` 去获取具有锁名前缀的最早创建的 revision。注意这里是用 m.pfx 来查询的，并且带了查询参数 WithFirstCreate()。使用 pfx 来查询是因为其他的 session 也会用同样的 pfx 来尝试加锁，并且因为每个 LeaseID 都不同，所以第一次肯定会 put 成功。但是只有最早使用这个 pfx 的 session 才是持有锁的。

```go
func (m *Mutex) tryAcquire(ctx context.Context) (*v3.TxnResponse， error) {
    s := m.s
    client := m.s.Client()

    // 这里将 pfx 和 lease ID 拼接起来成为主键，一把锁不同的 session 会有不同的租约 ID，但他们都会同样的前缀
    m.myKey = fmt.Sprintf("%s%x"， m.pfx， s.Lease())

    // 接下来这部分实现了如果不存在这个 key，则将这个 key 写入到 etcd，如果存在则读取这个 key 的值这样的功能
    cmp := v3.Compare(v3.CreateRevision(m.myKey)， "="， 0)
    put := v3.OpPut(m.myKey， ""， v3.WithLease(s.Lease()))
    get := v3.OpGet(m.myKey)
    getOwner := v3.OpGet(m.pfx， v3.WithFirstCreate()...)
    resp， err := client.Txn(ctx).If(cmp).Then(put， getOwner).Else(get， getOwner).Commit()
    if err != nil {
        return nil， err
    }

    // 本次操作的 revision
    m.myRev = resp.Header.Revision
    // 如果比较失败，则表示主键已存在，则拿 get 的结果，即已有的 revision
    if !resp.Succeeded {
        m.myRev = resp.Responses[0].GetResponseRange().Kvs[0].CreateRevision
    }
    return resp， nil
}
```



### waitDeletes

waitDeletes 方法就是获取比当前 revision 更小的 key，如果不存在则获取锁；存在则监听最新的 key 的删除，等这个 key 删除了，自己也就拿到锁了。

注意这里是个 for 循环，当检查到 delete 事件时，需要再次去获取自己是否当前 revision 最小，因为可能自己监听的 key 还未拿到锁就异常关闭，但是实际锁还未被释放。比如存在 1 和 2，我们应该监听 2 的事件，但是 2 因为异常退出了，此时检查还存在 1 比自己小，那就继续监听 1。

```go
func waitDeletes(ctx context.Context， client *v3.Client， pfx string， maxCreateRev int64) (*pb.ResponseHeader， error) {
    getOpts := append(v3.WithLastCreate()， v3.WithMaxCreateRev(maxCreateRev))
    for {
        // 获取比客户端 revision 小的 key
        resp， err := client.Get(ctx， pfx， getOpts...)
        if err != nil {
            return nil， err
        }
        // 如果没有就直接获取锁
        if len(resp.Kvs) == 0 {
            return resp.Header， nil
        }
        // 否则去监听 Revision 比自己小且相差最小的 key
        lastKey := string(resp.Kvs[0].Key)
        if err = waitDelete(ctx， client， lastKey， resp.Header.Revision); err != nil {
            return nil， err
        }
    }
}

func waitDelete(ctx context.Context， client *v3.Client， key string， rev int64) error {
    cctx， cancel := context.WithCancel(ctx)
    defer cancel()

    var wr v3.WatchResponse
    wch := client.Watch(cctx， key， v3.WithRev(rev))
    for wr = range wch {
        for _， ev := range wr.Events {
            if ev.Type == mvccpb.DELETE {
                return nil
            }
        }
    }
    if err := wr.Err(); err != nil {
        return err
    }
    if err := ctx.Err(); err != nil {
        return err
    }
    return fmt.Errorf("lost watcher waiting for delete")
}
```



## unlock

解锁就非常简单了，就是删除对应的 myKey，这里可以看到即使一个 session 可以多次获取同一把锁，但是在解锁的时候就一次释放了

```go
func (m *Mutex) Unlock(ctx context.Context) error {
   client := m.s.Client()
   if _， err := client.Delete(ctx， m.myKey); err != nil {
      return err
   }
   m.myKey = "\x00"
   m.myRev = -1
   return nil
}
```



## 总结

这种分布式锁的实现不存在锁的竞争，不存在重复的尝试加锁的操作。而是通过使用统一的前缀 pfx 来 put，然后根据各自的版本号来排队获取锁，效率非常的高。



---

# 基于etcd的选举

master 选举根本上也是抢锁 ，etcd 的实现总的来说也是分为以下几步：

- 连接客户端

- 获取 session
- `concurrency.NewElection` 获取选举者
- 竞选

前面两步和分布式锁一致，我们从第三步开始讲。



## NewElection

创建一个选举者和创建一个锁几乎一样，结构体也和锁的结构体类似

```go
func NewElection(s *Session， pfx string) *Election {
   return &Election{session: s， keyPrefix: pfx + "/"}
}

type Election struct {
	session *Session

	keyPrefix string

	leaderKey     string
	leaderRev     int64
	leaderSession *Session
	hdr           *pb.ResponseHeader
}
```



## 竞选

竞选的接口会传递一个值，这个值就是竞选者的名称

和锁很类似，这里也是以竞选者前缀 + 租约ID 作为主键，不同有以下几点：

- 分布式锁对应的键值为空，这里是传递进来的字符串
- 不用去获取最早创建竞选者前缀的 revision

这里的 leader 并不是说竞选的 leader，就是指自己，并且当值变化时，提供了 `Proclaim` 接口让领导者无需再次选举即可宣布新值。

waitDeletes 和分布式锁一样。

```go
func (e *Election) Campaign(ctx context.Context， val string) error {
    s := e.session
    client := e.session.Client()

    // 和分布式锁类似，获取客户端主键的 revision
    k := fmt.Sprintf("%s%x"， e.keyPrefix， s.Lease())
    txn := client.Txn(ctx).If(v3.Compare(v3.CreateRevision(k)， "="， 0))
    txn = txn.Then(v3.OpPut(k， val， v3.WithLease(s.Lease())))
    txn = txn.Else(v3.OpGet(k))
    resp， err := txn.Commit()
    if err != nil {
        return err
    }
    e.leaderKey， e.leaderRev， e.leaderSession = k， resp.Header.Revision， s
    if !resp.Succeeded {
        kv := resp.Responses[0].GetResponseRange().Kvs[0]
        e.leaderRev = kv.CreateRevision
        if string(kv.Value) != val {
            // 让领导者无需再次选举即可宣布新值，如果失败则主动放弃领导者角色
            if err = e.Proclaim(ctx， val); err != nil {
                e.Resign(ctx)
                return err
            }
        }
    }

    _， err = waitDeletes(ctx， client， e.keyPrefix， e.leaderRev-1)
    if err != nil {
        // clean up in case of context cancel
        select {
            case <-ctx.Done():
            e.Resign(client.Ctx())
            default:
            e.leaderSession = nil
        }
        return err
    }
    e.hdr = resp.Header

    return nil
}
```

`Proclaim` 接口让领导者无需再次选举即可宣布新值，简单地检查是否自己创建的 key 之后用 put 修改键值对即可。

```go
func (e *Election) Proclaim(ctx context.Context， val string) error {
   if e.leaderSession == nil {
      return ErrElectionNotLeader
   }
   client := e.session.Client()
   cmp := v3.Compare(v3.CreateRevision(e.leaderKey)， "="， e.leaderRev)
   txn := client.Txn(ctx).If(cmp)
   txn = txn.Then(v3.OpPut(e.leaderKey， val， v3.WithLease(e.leaderSession.Lease())))
   tresp， terr := txn.Commit()
   if terr != nil {
      return terr
   }
   if !tresp.Succeeded {
      e.leaderKey = ""
      return ErrElectionNotLeader
   }

   e.hdr = tresp.Header
   return nil
}
```



## 放弃领导者

```go
func (e *Election) Resign(ctx context.Context) (err error) {
   if e.leaderSession == nil {
      return nil
   }
   client := e.session.Client()
   cmp := v3.Compare(v3.CreateRevision(e.leaderKey)， "="， e.leaderRev)
   resp， err := client.Txn(ctx).If(cmp).Then(v3.OpDelete(e.leaderKey)).Commit()
   if err == nil {
      e.hdr = resp.Header
   }
   e.leaderKey = ""
   e.leaderSession = nil
   return err
}
```

