redis 集群有三种模式：主从模式，sentinel 模式，cluster 模式

---

# 主从模式

主从模式是三种模式中最简单的，在主从复制中，数据库分为两类：主数据库(master)和从数据库(slave)。

其中主从复制有如下特点：

- 主数据库可以进行读写操作，当读写操作导致数据变化时会自动将数据同步给从数据库
- 从数据库一般都是只读的，并且接收主数据库同步过来的数据
- 一个 master可以拥有多个 slave，但是一个 slave 只能对应一个 master
- slave 挂了不影响其他 slave 的读和 master 的读和写，重新启动后会将数据从 master 同步过来
- master 挂了以后，不影响 slave 的读，但 redis 不再提供写服务，master 重启后 redis 将重新对外提供写服务
- master 挂了以后，不会在 slave 节点中重新选一个 master



## 工作机制

当 slave 启动后，主动向 master 发送 SYNC 命令。

master 接收到 SYNC 命令后在后台保存快照（RDB持久化）和缓存保存快照这段时间的命令，然后将保存的快照文件和缓存的命令发送给 slave。slave 接收到快照文件和命令后加载快照文件和缓存的执行命令。

复制初始化后，master 每次接收到的写命令都会同步发送给 slave，保证主从数据一致性。



## 缺点

从上面可以看出，master 节点在主从模式中唯一，若 master 挂掉，则 redis 无法对外提供写服务。 



## 环境搭建

- 首先安装redis

```bash
tar xvzf redis-6.2.4.tar.gz
mv redis-6.2.4 /usr/local/redis
cd /usr/local/redis
make && make install
```



- 节点配置

|        | IP            |
| ------ | ------------- |
| 主节点 | 192.168.3.159 |
| slave  | 192.168.3.196 |
| slave  | 192.168.3.227 |



- 修改配置文件

master 节点

```bash
# vi /usr/local/redis/redis.conf
#监听ip，多个ip用空格分隔
bind 0.0.0.0
```



slave 节点

```bash
bind 192.168.3.196
replicaof 192.168.3.159 6379
```

```bash
bind 192.168.3.227
replicaof 192.168.3.159 6379
```



- 所有节点全部启动

```bash
redis-server /usr/local/redis/redis.conf
```



查看集群状态

```bash
# redis-cli -h 192.168.3.159

192.168.3.159:6379> ping
PONG
192.168.3.159:6379> info replication
# Replication
role:master
connected_slaves:2
slave0:ip=192.168.3.196,port=6379,state=online,offset=168,lag=1
slave1:ip=192.168.3.227,port=6379,state=online,offset=168,lag=1
master_failover_state:no-failover
master_replid:8006904dee86b5be248be86b6c23235c9f5be2ec
master_replid2:0000000000000000000000000000000000000000
master_repl_offset:168
second_repl_offset:-1
repl_backlog_active:1
repl_backlog_size:1048576
repl_backlog_first_byte_offset:1
repl_backlog_histlen:168
```

```bash
# redis-cli -h 192.168.3.196 info replication

# Replication
role:slave
master_host:192.168.3.159
master_port:6379
master_link_status:up
master_last_io_seconds_ago:6
master_sync_in_progress:0
slave_repl_offset:294
slave_priority:100
slave_read_only:1
replica_announced:1
connected_slaves:0
master_failover_state:no-failover
master_replid:8006904dee86b5be248be86b6c23235c9f5be2ec
master_replid2:0000000000000000000000000000000000000000
master_repl_offset:294
second_repl_offset:-1
repl_backlog_active:1
repl_backlog_size:1048576
repl_backlog_first_byte_offset:1
repl_backlog_histlen:294

```



## 操作演示

主节点

```bash
# redis-cli -h 192.168.3.159

192.168.3.159:6379> ping
PONG
192.168.3.159:6379> keys *
(empty array)

192.168.3.159:6379> set key 100
OK
192.168.3.159:6379> keys *
1) "key"
```

slave 节点

```bash
# redis-cli -h 192.168.3.196

192.168.3.196:6379> keys *
(empty array)

# 主节点写入后，slave节点也可以查看
192.168.3.196:6379> keys *
1) "key"

# slave节点上无法写入数据
192.168.3.196:6379> set key aaa
(error) READONLY You can't write against a read only replica.
```



---

# sentinel模式

主从模式的弊端就是不具备高可用性，当 master 挂掉以后，Redis 将不能再对外提供写入操作，因此 sentinel 应运而生。

sentinel 中文含义为哨兵，顾名思义，它的作用就是监控 redis 集群的运行状况，特点如下：

- sentinel 模式是建立在主从模式的基础上，如果只有一个 Redis 节点，sentinel 就没有任何意义
- 当 master 挂了以后，sentinel 会在 slave 中选择一个做为 master，并修改它们的配置文件，其他 slave 的配置文件也会被修改，比如 slaveof 属性会指向新的master
- 当 master 重新启动后，它将不再是 master 而是做为 slave 接收新的 master 的同步数据
- sentinel 因为也是一个进程有挂掉的可能，所以 sentinel 也会启动多个形成一个 sentinel 集群
- 多 sentinel 配置的时候，sentinel 之间也会自动监控
- 当主从模式配置密码时，sentinel 也会同步将配置信息修改到配置文件中，不需要担心
- 一个 sentinel 或 sentinel 集群可以管理多个主从 Redis，多个 sentinel 也可以监控同一个 redis
- sentinel 最好不要和 Redis 部署在同一台机器，不然 Redis 的服务器挂了以后，sentinel 也挂了



## 工作机制

- 每个 sentinel 以每秒钟一次的频率向它所知的 master，slave以及其他 sentinel 实例发送一个 PING 命令 
- 如果一个实例距离最后一次有效回复 PING 命令的时间超过 down-after-milliseconds 选项所指定的值， 则这个实例会被 sentinel 标记为主观下线。 
- 如果一个 master 被标记为主观下线，则正在监视这个 master 的所有 sentinel 要以每秒一次的频率确认 master 的确进入了主观下线状态
- 当有足够数量的 sentinel（大于等于配置文件指定的值）在指定的时间范围内确认 master 的确进入了主观下线状态， 则 master 会被标记为客观下线 
- 在一般情况下， 每个 sentinel 会以每 10 秒一次的频率向它已知的所有 master，slave发送 INFO 命令 
- 当 master 被 sentinel 标记为客观下线时，sentinel 向下线的 master 的所有 slave 发送 INFO 命令的频率会从 10 秒一次改为 1 秒一次 
- 若没有足够数量的 sentinel 同意 master 已经下线，master 的客观下线状态就会被移除；若 master 重新向 sentinel 的 PING 命令返回有效回复，master 的主观下线状态就会被移除

当使用 sentinel 模式的时候，客户端就不要直接连接 Redis，而是连接 sentinel 的 ip 和 por t，由 sentinel 来提供具体的可提供服务的Redis实现，这样当 master 节点挂掉以后，sentinel 就会感知并将新的 master 节点提供给使用者。



## 环境搭建

每个sentinel节点都修改配置文件 

```bash
# vi /usr/local/redis/sentinel.conf
#判断master失效至少需要2个sentinel同意，建议设置为n/2+1，n为sentinel个数
sentinel monitor mymaster 192.168.3.159 6379 2
#判断master主观下线时间，默认30s
sentinel down-after-milliseconds mymaster 30000
```

所有节点启动

```bash
redis-sentinel /usr/local/redis/sentinel.conf
```



Sentinel 模式下的几个事件： 

```bash
·    +reset-master ：主服务器已被重置。

·    +slave ：一个新的从服务器已经被 Sentinel 识别并关联。

·    +failover-state-reconf-slaves ：故障转移状态切换到了 reconf-slaves 状态。

·    +failover-detected ：另一个 Sentinel 开始了一次故障转移操作，或者一个从服务器转换成了主服务器。

·    +slave-reconf-sent ：领头（leader）的 Sentinel 向实例发送了 [SLAVEOF](/commands/slaveof.html) 命令，为实例设置新的主服务器。

·    +slave-reconf-inprog ：实例正在将自己设置为指定主服务器的从服务器，但相应的同步过程仍未完成。

·    +slave-reconf-done ：从服务器已经成功完成对新主服务器的同步。

·    -dup-sentinel ：对给定主服务器进行监视的一个或多个 Sentinel 已经因为重复出现而被移除 —— 当 Sentinel 实例重启的时候，就会出现这种情况。

·    +sentinel ：一个监视给定主服务器的新 Sentinel 已经被识别并添加。

·    +sdown ：给定的实例现在处于主观下线状态。

·    -sdown ：给定的实例已经不再处于主观下线状态。

·    +odown ：给定的实例现在处于客观下线状态。

·    -odown ：给定的实例已经不再处于客观下线状态。

·    +new-epoch ：当前的纪元（epoch）已经被更新。

·    +try-failover ：一个新的故障迁移操作正在执行中，等待被大多数 Sentinel 选中（waiting to be elected by the majority）。

·    +elected-leader ：赢得指定纪元的选举，可以进行故障迁移操作了。

·    +failover-state-select-slave ：故障转移操作现在处于 select-slave 状态 —— Sentinel 正在寻找可以升级为主服务器的从服务器。

·    no-good-slave ：Sentinel 操作未能找到适合进行升级的从服务器。Sentinel 会在一段时间之后再次尝试寻找合适的从服务器来进行升级，
又或者直接放弃执行故障转移操作。

·    selected-slave ：Sentinel 顺利找到适合进行升级的从服务器。

·    failover-state-send-slaveof-noone ：Sentinel 正在将指定的从服务器升级为主服务器，等待升级功能完成。

·    failover-end-for-timeout ：故障转移因为超时而中止，不过最终所有从服务器都会开始复制新的主服务器
（slaves will eventually be configured to replicate with the new master anyway）。

·    failover-end ：故障转移操作顺利完成。所有从服务器都开始复制新的主服务器了。

·    +switch-master ：配置变更，主服务器的 IP 和地址已经改变。 这是绝大多数外部用户都关心的信息。

·    +tilt ：进入 tilt 模式。

·    -tilt ：退出 tilt 模式。
```



查看输出可以看到监控 redis 集群开始，也可以看到其余的 sentinel 加入

```bash
16915:X 01 Jul 2021 14:06:26.205 # Sentinel ID is 8a72188ed7816dcee0bce0fe44221856a1f89116
16915:X 01 Jul 2021 14:06:26.206 # +monitor master mymaster 192.168.3.159 6379 quorum 2
16915:X 01 Jul 2021 14:06:26.211 * +slave slave 192.168.3.227:6379 192.168.3.227 6379 @ mymaster 192.168.3.159 6379
16915:X 01 Jul 2021 14:06:26.291 * +slave slave 192.168.3.196:6379 192.168.3.196 6379 @ mymaster 192.168.3.159 6379
16915:X 01 Jul 2021 14:07:50.679 * +sentinel sentinel 6567afcabc81934572aea5801c84845661b2d21a 192.168.3.201 26379 @ mymaster 192.168.3.159 6379
16915:X 01 Jul 2021 14:07:52.512 * +sentinel sentinel 42ce6f8dc93333c4e7c95608281672e916c5174b 192.168.3.218 26379 @ mymaster 192.168.3.159 6379
```



## 故障模拟

将 192.168.3.159 服务关闭之后，查看 Sentinel 输出，可以看到 master 转移到了 192.168.3.196 上

```bash
16915:X 01 Jul 2021 14:15:27.325 # +sdown master mymaster 192.168.3.159 6379
16915:X 01 Jul 2021 14:15:27.378 # +odown master mymaster 192.168.3.159 6379 #quorum 3/2
16915:X 01 Jul 2021 14:15:27.378 # +new-epoch 1
16915:X 01 Jul 2021 14:15:27.378 # +try-failover master mymaster 192.168.3.159 6379
16915:X 01 Jul 2021 14:15:27.610 # +vote-for-leader 8a72188ed7816dcee0bce0fe44221856a1f89116 1
16915:X 01 Jul 2021 14:15:27.756 # 6567afcabc81934572aea5801c84845661b2d21a voted for 8a72188ed7816dcee0bce0fe44221856a1f89116 1
16915:X 01 Jul 2021 14:15:27.786 # +elected-leader master mymaster 192.168.3.159 6379
16915:X 01 Jul 2021 14:15:27.786 # +failover-state-select-slave master mymaster 192.168.3.159 6379
16915:X 01 Jul 2021 14:15:27.853 # +selected-slave slave 192.168.3.196:6379 192.168.3.196 6379 @ mymaster 192.168.3.159 6379
16915:X 01 Jul 2021 14:15:27.853 * +failover-state-send-slaveof-noone slave 192.168.3.196:6379 192.168.3.196 6379 @ mymaster 192.168.3.159 6379
16915:X 01 Jul 2021 14:15:27.925 # 42ce6f8dc93333c4e7c95608281672e916c5174b voted for 42ce6f8dc93333c4e7c95608281672e916c5174b 1
16915:X 01 Jul 2021 14:15:27.955 * +failover-state-wait-promotion slave 192.168.3.196:6379 192.168.3.196 6379 @ mymaster 192.168.3.159 6379
16915:X 01 Jul 2021 14:15:28.945 # +promoted-slave slave 192.168.3.196:6379 192.168.3.196 6379 @ mymaster 192.168.3.159 6379
16915:X 01 Jul 2021 14:15:28.945 # +failover-state-reconf-slaves master mymaster 192.168.3.159 6379
16915:X 01 Jul 2021 14:15:28.946 * +slave-reconf-sent slave 192.168.3.227:6379 192.168.3.227 6379 @ mymaster 192.168.3.159 6379
16915:X 01 Jul 2021 14:15:29.209 * +slave-reconf-inprog slave 192.168.3.227:6379 192.168.3.227 6379 @ mymaster 192.168.3.159 6379
16915:X 01 Jul 2021 14:15:29.397 # -odown master mymaster 192.168.3.159 6379
16915:X 01 Jul 2021 14:15:30.278 * +slave-reconf-done slave 192.168.3.227:6379 192.168.3.227 6379 @ mymaster 192.168.3.159 6379
16915:X 01 Jul 2021 14:15:30.354 # +failover-end master mymaster 192.168.3.159 6379
16915:X 01 Jul 2021 14:15:30.354 # +switch-master mymaster 192.168.3.159 6379 192.168.3.196 6379
16915:X 01 Jul 2021 14:15:30.354 * +slave slave 192.168.3.227:6379 192.168.3.227 6379 @ mymaster 192.168.3.196 6379
```



查看 redis 集群的状态

```bash
redis-cli -h 192.168.3.196 info replication
# Replication
role:master
connected_slaves:1
slave0:ip=192.168.3.227,port=6379,state=online,offset=97302,lag=1
master_failover_state:no-failover
master_replid:3bb4662b227225141286489f6c1c0570242a3c5c
master_replid2:d7645336e06bdc66e0c768cbc343e4049276fe42
master_repl_offset:97302
second_repl_offset:92458
repl_backlog_active:1
repl_backlog_size:1048576
repl_backlog_first_byte_offset:1
repl_backlog_histlen:97302

```



然后再将 192.168.3.159 启动，可以看到 159 是作为 slave 节点重新加入集群

```bash
[root@etcd-2 opt]# redis-cli -h 192.168.3.196 info replication
# Replication
role:master
connected_slaves:2
slave0:ip=192.168.3.227,port=6379,state=online,offset=161635,lag=0
slave1:ip=192.168.3.159,port=6379,state=online,offset=161635,lag=1
master_failover_state:no-failover
master_replid:3bb4662b227225141286489f6c1c0570242a3c5c
master_replid2:d7645336e06bdc66e0c768cbc343e4049276fe42
master_repl_offset:161635
second_repl_offset:92458
repl_backlog_active:1
repl_backlog_size:1048576
repl_backlog_first_byte_offset:1
repl_backlog_histlen:161635
```







---

# 参考与感谢

- [redis sentinel原理介绍](http://www.redis.cn/topics/sentinel.html) 
- [redis复制与高可用配置](https://www.cnblogs.com/itzhouq/p/redis5.html)  
- [redis cluster介绍](http://redisdoc.com/topic/cluster-spec.html) 
- [redis cluster原理](https://www.cnblogs.com/williamjie/p/11132211.html) 
- [redis cluster详细配置](https://www.cnblogs.com/renpingsheng/p/9813959.html) 
- [redis+redission实现分布式锁](https://www.jianshu.com/p/47fd7f86c848)  



延伸阅读： 

- [Hazelcast介绍](https://gitbook.cn/gitchat/activity/5ce3edc02cdc9c79e575397a) 
- [Vert.x入门](https://gitbook.cn/gitchat/activity/5c6f733f5cef2d4672764a74) 











