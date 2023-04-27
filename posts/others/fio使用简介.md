快速开始







---

# Job file 常用参数

## 参数类型

| 类型       | 解释                                                         |
| ---------- | ------------------------------------------------------------ |
| str        | 字符串                                                       |
| time       | 可能带有时间后缀的整数。除非另有说明，没有单位的值解释为秒。<br/>接受后缀 d 表示天，h 表示小时，m 表示分钟，s 表示秒，ms 或 msec 表示毫秒，us 或 usec 表示微秒。比如 10 分钟用 10m。 |
| int        | 整数，可以包含一个整数前缀和一个整数后缀<br>可选的整数前缀指定数字的基数。默认值为十进制。0x 指定十六进制。<br/>可选的整数后缀指定了数字的单位，包括可选的单位前缀和可选的单位。<br/>对于数据量，默认单位是字节。对于时间量，除非另有说明，否则默认单位为秒。整数后缀不区分大小写。 |
| bool       | 布尔型。通常被解析为整数，但是只定义为真和假（1 和 0）。     |
| irange     | 带后缀的整数范围。允许给定值范围，如 1024-4096。<br/>冒号也可以用作分隔符，例如 1k:4k。如果该选项允许两组范围，可以用 '，'或 '/' 分隔符指定：1k-4k/8k-32k。 |
| float_list | 由 ：字符分隔的浮点数列表。                                  |

下面列出一些常用的参数，更多信息可以参考官方文档。



## 负载描述

| 参数        | 类型 | 含义                                                         |
| ----------- | ---- | ------------------------------------------------------------ |
| name        | str  | job 名称                                                     |
| description | str  | job 描述                                                     |
| loops       | int  | 重复次数，默认为 1                                           |
| numjobs     | int  | 创建进程或线程执行相同的 job，每个线程单独报告；<br>要查看整体统计信息，请将group_reporting与new_group结合使用。默认值为 1 |



## 时间相关参数

| 参数       | 类型 | 含义                                                         |
| ---------- | ---- | ------------------------------------------------------------ |
| runtime    | time | 告诉 fio 在指定的时间段后终止处理。                          |
| time_based | \    | 如果设置，fio 将在指定的运行时间内运行，即使文件被完全读取或写入。<br/>只要运行时允许，它就会在相同的工作负载上循环多次。 |



## 目标设备

| 参数              | 类型 | 含义                                                         |
| ----------------- | ---- | ------------------------------------------------------------ |
| directory         | str  | 用此目录作为文件名的前缀，即我们要测试的目录。<br/>可以通过用 ：字符分隔名称来指定多个目录，这些目录将被 numjobs 线程平均使用。 |
| filename          | str  | Fio 通常根据作业名、线程号和文件号来创建文件名（参见 filename_format）。<br/>如果要在一个作业或多个具有固定文件路径的作业中的线程之间共享文件，请为每个线程指定一个文件名以覆盖默认值。<br/>如果 ioengine 是基于文件的，您可以通过用冒号分隔名称来指定多个文件。<br/>这也意味着指定该选项 nrfiles 会被忽略。<br/>除非 filesize 指定了明确的大小，否则此选项指定的常规文件的大小将是大小除以文件数。 |
| filename_format   | str  | 缺省为 `$jobname.$jobnum.$filenum`，要让相关 job 共享一组文件，可以设置该选项，让 fio 生成两者共享的文件名。 |
| lockfile          | str  | Fio 默认在对文件进行 I/O 操作之前不锁定任何文件。这通常用于模拟共享文件的真实工作负载。锁定模式包括：<br/>- none：无锁（默认）<br/>- exclusive：一次只能有一个线程或进程执行 I/O<br/>- readwrite：可以同时读，但写得独占 |
| nrfiles           | int  | 用于此作业的文件数。默认为 1。除非 filesize 指定了显式大小，否则文件大小为 size 除以此值。<br/>文件是为每个线程单独创建的，默认情况下，每个文件在其名称中都有一个文件号，如 filename 部分所述。 |
| openfiles         | int  | 同时保持打开的文件数。默认值与 nrfiles 相同，可以设置得更小以限制同时打开的数量。 |
| file_service_type | str  |                                                              |
| create_on_open    | bool | 如果为 true，则不要预先创建文件，而是允许 open 时创建文件。默认值为 false，在开始时预先创建所有必需的文件。 |
| unlink            | bool | 每次测试完删不删测试文件，默认为 false                       |

我们来看下上面参数具体情况都是怎么样使用的，首先我们只在一个目录下测试，可以看到生成了 6 （numjobs）个文件，每个大小为 10M（size）

```bash
[root@localhost test]# fio --name=test --size=10M --directory=/mnt/0/ --runtime=10s --numjobs=6
[root@localhost mnt]# tree 0 1
0
├── test.0.0
├── test.1.0
├── test.2.0
├── test.3.0
├── test.4.0
└── test.5.0
1

0 directories, 6 files
[root@localhost mnt]# ll -h 0/
总用量 60M
-rw-r--r-- 1 root root 10M 7月  14 15:40 test.0.0
-rw-r--r-- 1 root root 10M 7月  14 15:40 test.1.0
-rw-r--r-- 1 root root 10M 7月  14 15:40 test.2.0
-rw-r--r-- 1 root root 10M 7月  14 15:40 test.3.0
-rw-r--r-- 1 root root 10M 7月  14 15:40 test.4.0
-rw-r--r-- 1 root root 10M 7月  14 15:40 test.5.0
```

接下来我们修改 directory，在两个目录下测试，我们可以看到文件被均分到两个目录下了。

```bash
[root@localhost test]# fio --name=test --size=10M --directory=/mnt/0/:/mnt/1/ --runtime=10s --numjobs=6
[root@localhost mnt]# tree 0 1 ; ll -h 0/; ll -h 1/
0
├── test.0.0
├── test.2.0
└── test.4.0
1
├── test.1.0
├── test.3.0
└── test.5.0

0 directories, 6 files
总用量 30M
-rw-r--r-- 1 root root 10M 7月  14 15:43 test.0.0
-rw-r--r-- 1 root root 10M 7月  14 15:43 test.2.0
-rw-r--r-- 1 root root 10M 7月  14 15:43 test.4.0
总用量 30M
-rw-r--r-- 1 root root 10M 7月  14 15:43 test.1.0
-rw-r--r-- 1 root root 10M 7月  14 15:43 test.3.0
-rw-r--r-- 1 root root 10M 7月  14 15:43 test.5.0
```

我们再来看 filename 的效果，filename 用于在多个线程间共享文件，所以这里只创建了一个文件 hello。

```bash
[root@localhost test]# fio --name=test --size=10M --directory=/mnt/0/:/mnt/1/ --filename=hello --runtime=10s --numjobs=6
[root@localhost mnt]# tree 0 1 ; ll -h 0/; ll -h 1/
0
└── hello
1

0 directories, 1 file
总用量 10M
-rw-r--r-- 1 root root 10M 7月  14 15:52 hello
总用量 0
```

我们继续增加 filename 的文件，可以发现共享文件变多了，但是大小变成了原来的一半。所以我们可以使用 filesize 来指定文件大小。

```bash
[root@localhost test]# fio --name=test --size=10M --directory=/mnt/0/:/mnt/1/ --filename=hello:world --runtime=10s --numjobs=6
[root@localhost mnt]# tree 0 1 ; ll -h 0/; ll -h 1/
0
├── hello
└── world
1

0 directories, 2 files
总用量 10M
-rw-r--r-- 1 root root 5.0M 7月  14 15:53 hello
-rw-r--r-- 1 root root 5.0M 7月  14 15:53 world
总用量 0

# 指定 filesize
[root@localhost test]# fio --name=test --size=10M --directory=/mnt/0/:/mnt/1/ --filename=hello:world --runtime=10s --numjobs=6 --filesize=7M
[root@localhost mnt]# tree 0 1 ; ll -h 0/; ll -h 1/
0
├── hello
└── world
1

0 directories, 2 files
总用量 14M
-rw-r--r-- 1 root root 7.0M 7月  14 16:03 hello
-rw-r--r-- 1 root root 7.0M 7月  14 16:03 world
总用量 0
```

我们再来看下 nrfiles 的使用，指定文件个数之后，每个文件大小为 size / nrfiles。

```bash
[root@localhost test]# fio --name=test --size=10M --directory=/mnt/0/:/mnt/1/ --runtime=10s --numjobs=6 --nrfiles=5
[root@localhost mnt]# tree 0 1 ; ll -h 0/; ll -h 1/
0
├── test.0.0
├── test.0.1
├── test.0.2
├── test.0.3
├── test.0.4
├── test.2.0
├── test.2.1
├── test.2.2
├── test.2.3
├── test.2.4
├── test.4.0
├── test.4.1
├── test.4.2
├── test.4.3
└── test.4.4
1
├── test.1.0
├── test.1.1
├── test.1.2
├── test.1.3
├── test.1.4
├── test.3.0
├── test.3.1
├── test.3.2
├── test.3.3
├── test.3.4
├── test.5.0
├── test.5.1
├── test.5.2
├── test.5.3
└── test.5.4

0 directories, 30 files
总用量 30M
-rw-r--r-- 1 root root 2.0M 7月  14 18:18 test.0.0
-rw-r--r-- 1 root root 2.0M 7月  14 18:18 test.0.1
...
-rw-r--r-- 1 root root 2.0M 7月  14 18:18 test.4.3
-rw-r--r-- 1 root root 2.0M 7月  14 18:18 test.4.4
总用量 30M
-rw-r--r-- 1 root root 2.0M 7月  14 18:18 test.1.0
-rw-r--r-- 1 root root 2.0M 7月  14 18:18 test.1.1
...
-rw-r--r-- 1 root root 2.0M 7月  14 18:18 test.5.3
-rw-r--r-- 1 root root 2.0M 7月  14 18:18 test.5.4
```

如果我们指定 filesize，可以看到每个文件大小就为 filesize，但是个数还是 nrfiles。

```bash
[root@localhost test]# fio --name=test --size=10M --directory=/mnt/0/:/mnt/1/ --runtime=10s --numjobs=6 --nrfiles=5 --filesize=1M
[root@localhost mnt]# tree 0 1 ; ll -h 0/; ll -h 1/
0
├── test.0.0
├── test.0.1
├── test.0.2
├── test.0.3
├── test.0.4
├── test.2.0
├── test.2.1
├── test.2.2
├── test.2.3
├── test.2.4
├── test.4.0
├── test.4.1
├── test.4.2
├── test.4.3
└── test.4.4
1
├── test.1.0
├── test.1.1
├── test.1.2
├── test.1.3
├── test.1.4
├── test.3.0
├── test.3.1
├── test.3.2
├── test.3.3
├── test.3.4
├── test.5.0
├── test.5.1
├── test.5.2
├── test.5.3
└── test.5.4

0 directories, 30 files
总用量 15M
-rw-r--r-- 1 root root 1.0M 7月  14 18:21 test.0.0
...
-rw-r--r-- 1 root root 1.0M 7月  14 18:21 test.4.4
总用量 15M
-rw-r--r-- 1 root root 1.0M 7月  14 18:21 test.1.0
...
-rw-r--r-- 1 root root 1.0M 7月  14 18:21 test.5.4
```



## IO类型

| 参数            | 类型 | 含义                                                         |
| --------------- | ---- | ------------------------------------------------------------ |
| direct          | bool | 默认为 false，如果值为 true，则使用非缓冲 I/O。这通常是 O_DIRECT。<br>请注意，Solaris 上的 OpenBSD 和 ZFS 不支持 direct I/O。在 Windows 上，同步 ioengines 不支持 direct I/O。默认值：false。 |
| readwrite（rw） | str  | I/O 模式的类型。可接受的值为：<br/>- read / randread：顺序读 / 随机读<br/>- write / randwrite：顺序写 / 随机写<br/>- trim / randtrim：顺序 trim / 随机 trim (Linux block devices and SCSI character devices only).<br/>- readwrite（rw）/ randrw：顺序混合读写 / 随机混合读写，默认混合比例为 50/50<br/>- trimwrite：顺序的 trim + write<br/>PS：冒号可以让 IO 偏移，比如 rw=write:4k 表示每次写之后将会跳过 4K，它将顺序的IO转化为带有洞的顺序IO。rw=randread:8 表示每 8 次 IO 之后执行 seek，而不是每次 IO 之后。 |
| rw_sequencer    | str  | 如果 rw 用冒号指定了偏移量，那么这里用来控制偏移行为：<br/>- sequential：产生顺序的偏移，仅适用于随机 IO<br/>- identical：产生相同的偏移 |
| fsync           | int  | 表示执行多少次 IO 后 fsync                                   |
| fdatasync       | int  | 表示执行多少次 IO 后 fdatasync，fdatasync 只同步元数据       |
| write_barrier   | int  | 表示每多少次 IO 作为一次写入屏障                             |
| end_fsync       | bool | 默认为 false。如果为 true，则在写入阶段完成时，fsync         |
| fsync_on_close  | bool | 默认为 false。如果为 true，则在每次关闭文件时，fsync         |
| rwmixread       | int  | 混合读的比例，默认 50                                        |
| rwmixwrite      | int  | 混合写的比例，默认 50，如果和 rwmixread 加起来不是 100，则用后面的值覆盖前面的 |



## Block 大小

**blocksize（bs）**用于 I/O 单元的块大小（以字节为单位）。默认值:4096。语法为 `bs=int[,int][,int]`，单个值适用于读取、写入和 trim，也可以为读取、写入和 trim 指定值，不以逗号结尾的值适用于后续类型。示例如下：

>   bs=256k
>
>   means 256k for reads, writes and trims.
>
>   bs=8k,32k
>
>   means 8k for reads, 32k for writes and trims.
>
>   bs=8k,32k,
>
>   means 8k for reads, 32k for writes, and default for trims.
>
>   bs=,8k
>
>   means default for reads, 8k for writes and trims.
>
>   bs=,8k,
>
>   means default for reads, 8k for writes, and default for trims.

**blocksize_range（bsrange）** 表示 I/O 单元的块大小范围（以字节为单位），语法同 blocksize，比如 `bsrange=1k-4k，2k-8k`

有时，您想要对发出的块大小进行更细粒度的控制，而不仅仅是在它们之间进行平均分割。**bssplit** 选项允许您对各种块大小进行加权，以便您能够定义发出的块大小的特定数量。该选项的格式为：`bssplit=blocksize/percentage:blocksize/percentage`

比如您想要定义一个包含 50% 64k 数据块、10% 4k 数据块和 40% 32k 数据块的工作负载：`bssplit=4k/10:64k/50:32k/40`

顺序并不重要。如果百分比为空，fio 将平均填充剩余的值。所以像这样的一个 bssplit 选项：`bssplit=4k/50:1k/:32k/` 表示将有 50% 的 4k ios，以及 25% 的 1k 和 32k ios。百分比的总和总是100，如果给 bssplit 总和大于100，它将出错。



## buffer和内存选项

**sync** 表示写 IO 时候同步：

-   默认为 none，不用同步
-   sync：对于大多数 I/O 引擎，这意味着使用 O_SYNC
-   dsync：对于大多数 I/O 引擎，这意味着使用 O_DSYNC。

**lockmem** 可以限制使用内存，比如 lockmem=1g 只使用 1g 内存进行测试。

**refill_buffers** 重新填充缓冲区，以免上次测试的数据影响。



## IO 大小

**size** 表示每个线程的总 IO 大小，也可以使用百分比来测试，如 size=20%

通常情况下，fio在由 size 设置的区域内运行，这意味着 size 选项设置了要执行的 I/O 的区域和大小。有时候这不是你想要的。通过这个 **io_size** 选项，可以定义 fio 应该执行的 I/O 量。例如，如果 size 设置为 20GiB，io_size 设置为 5GiB，则 fio 将在第一个 20GiB 内执行I/O，但在 5GiB 完成后退出。相反的情况也是可能的，如果 size 设置为 20GiB，io_size 设置为 40GiB，那么 fio 将在 0..20GiB 区域内执行。

**filesize** 表示单个文件大小，也可以是一个范围，在这种情况下，fio 将在给定的范围内随机选择文件的大小。如果未给定，则每个创建的文件大小相同。此选项在文件大小方面覆盖了 size，即如果指定了 filesize，则 size 仅成为 io_size 的默认值，如果明确设置了 io_size，则 size 没有任何影响。



## IO 引擎

指定 IO 发起的方式，下面列出部分：

-   sync：同步模型，即使用基本的 read write lseek 等
-   libaio：Linux 原生异步 I/O。请注意，Linux 可能只支持非缓冲 I/O 的排队行为（设置 direct=1 或 buffered=0）
-   posixaio：POSIX 异步 IO
-   io_uring：Linux 的异步 IO 模型，另外还有一些其余的异步 IO 模型我没有列出来，是因为它们既不常用也不实用
-   mmap：文件通过内存映射到用户空间，使用 memcpy 写入和读出数据
-   rmda：RDMA I/O引擎支持 InfiniBand、RoCE 和 iWARP 协议的 RDMA 内存语义（RDMA 写/RDMA 读）和通道语义(发送/接收)。
-   rados：I/O引擎支持通过librados直接访问Ceph可靠的自主分布式对象存储(RADOS)。
-   rdb：I/O 引擎支持通过 librbd 直接访问 Ceph Rados 块设备 RBD，而无需使用内核 rbd 驱动程序。
-   http：I/O 引擎支持使用 libcurl 通过 HTTP(S) 向 WebDAV 或 S3 端点发送 GET/PUT 请求。
-   nfs：I/O 引擎支持通过 libnfs 从用户空间对 NFS 文件系统进行异步读写操作。这有助于实现比内核 NFS 更高的并发性和吞吐量。
-   。。。

某些引擎需要设定特定的参数才能工作，具体请参考官方文档，比如测试 S3：

-   http_host：要连接的主机名，默认为 localhost
-   http_user：HTTP 认证用户名
-   http_pass：HTTP 认证密码
-   https：是否启用 https，默认为 off
-   http_mode：使用哪种 HTTP 访问模式：webdav、swift 或 s3。默认为 webdav
-   http_s3_region：The S3 region/zone string. Default is us-east-1
-   http_s3_key：The S3 secret key.
-   http_s3_keyid：The S3 key/access id.
-   http_verbose：从 libcurl 启用详细请求。对调试有用。1 从 libcurl 打开详细日志记录，2 另外启用 HTTP IO 跟踪。默认值为 0



## 队列深度

如果 IO 引擎是异步的，那么就可以指定 **iodepth** 需要保持的队列深度。



## IO 重放

可以配合 blktrace 重放 IO



## 进程选项

-   thread：fio 默认使用 fork 创建 job，但是如果给定了这个选项，fio 将使用 POSIX Threads 的函数 pthread_create 创建线程来创建 job。



## 报告

-   group_reporting：线程报告汇总





---

# 输出解释

运行时，fio 将显示所创建 job 的状态。一个例子是：

```bash
Jobs: 1 (f=1): [_(1),M(1)][24.8%][r=20.5MiB/s,w=23.5MiB/s][r=82,w=94 IOPS][eta 01m:31s]
```

第一组方括号内的字符表示每个线程的当前状态。第一个字符是作业文件中定义的第一个 job，依此类推。可能的值(按照典型的生命周期顺序)为：

| Idle | Run  |                                                           |
| :--- | :--- | :-------------------------------------------------------- |
| P    |      | Thread setup, but not started.                            |
| C    |      | Thread created.                                           |
| I    |      | Thread initialized, waiting or generating necessary data. |
|      | p    | Thread running pre-reading file(s).                       |
|      | /    | Thread is in ramp period.                                 |
|      | R    | Running, doing sequential reads.                          |
|      | r    | Running, doing random reads.                              |
|      | W    | Running, doing sequential writes.                         |
|      | w    | Running, doing random writes.                             |
|      | M    | Running, doing mixed sequential reads/writes.             |
|      | m    | Running, doing mixed random reads/writes.                 |
|      | D    | Running, doing sequential trims.                          |
|      | d    | Running, doing random trims.                              |
|      | F    | Running, currently waiting for *fsync(2)*.                |
|      | V    | Running, doing verification of written data.              |
| f    |      | Thread finishing.                                         |
| E    |      | Thread exited, not reaped by main thread yet.             |
| _    |      | Thread reaped.                                            |
| X    |      | Thread reaped, exited with an error.                      |
| K    |      | Thread reaped, exited due to signal.                      |

fio 将压缩线程字符串，以免在命令行上占用不必要的空间。例如，如果有 10 个读和 10 个写在运行，输出将如下所示：

```bash
Jobs: 20 (f=20): [R(10),W(10)][4.0%][r=20.5MiB/s,w=23.5MiB/s][r=82,w=94 IOPS][eta 57m:36s]
```

状态字符串是按顺序显示的，因此可以看出哪些作业当前正在做什么。在上面的例子中，1-10 号 job 是读，11-20 是写。

其他值是不言而喻的，当前运行和执行 I/O 的线程数量、当前打开的文件数量（f=）、估计完成百分比、自上次检查以来的 I/O 速率，以带宽和 IOPS 表示，以及当前运行组的完成时间。

当 fio 完成时或者被 Ctrl-C 中断，它将按顺序显示每个线程、线程组和磁盘的数据。对于每个线程或组，输出如下：

```bash
Client1: (groupid=0, jobs=1): err= 0: pid=16109: Sat Jun 24 12:07:54 2017
  write: IOPS=88, BW=623KiB/s (638kB/s)(30.4MiB/50032msec)
    slat (nsec): min=500, max=145500, avg=8318.00, stdev=4781.50
    clat (usec): min=170, max=78367, avg=4019.02, stdev=8293.31
     lat (usec): min=174, max=78375, avg=4027.34, stdev=8291.79
    clat percentiles (usec):
     |  1.00th=[  302],  5.00th=[  326], 10.00th=[  343], 20.00th=[  363],
     | 30.00th=[  392], 40.00th=[  404], 50.00th=[  416], 60.00th=[  445],
     | 70.00th=[  816], 80.00th=[ 6718], 90.00th=[12911], 95.00th=[21627],
     | 99.00th=[43779], 99.50th=[51643], 99.90th=[68682], 99.95th=[72877],
     | 99.99th=[78119]
   bw (  KiB/s): min=  532, max=  686, per=0.10%, avg=622.87, stdev=24.82, samples=  100
   iops        : min=   76, max=   98, avg=88.98, stdev= 3.54, samples=  100
  lat (usec)   : 250=0.04%, 500=64.11%, 750=4.81%, 1000=2.79%
  lat (msec)   : 2=4.16%, 4=1.84%, 10=4.90%, 20=11.33%, 50=5.37%
  lat (msec)   : 100=0.65%
  cpu          : usr=0.27%, sys=0.18%, ctx=12072, majf=0, minf=21
  IO depths    : 1=85.0%, 2=13.1%, 4=1.8%, 8=0.1%, 16=0.0%, 32=0.0%, >=64=0.0%
     submit    : 0=0.0%, 4=100.0%, 8=0.0%, 16=0.0%, 32=0.0%, 64=0.0%, >=64=0.0%
     complete  : 0=0.0%, 4=100.0%, 8=0.0%, 16=0.0%, 32=0.0%, 64=0.0%, >=64=0.0%
     issued rwt: total=0,4450,0, short=0,0,0, dropped=0,0,0
     latency   : target=0, window=0, percentile=100.00%, depth=8
```

第一行是作业信息，下面每行分别是：

-   read/write/trim

冒号前的字符串显示统计数据的 I/O 方向。IOPS 是每秒执行的平均 IO 数。BW 是平均带宽速率，表示为 2 的幂的值（10 的幂的值）。最后两个值显示以 2 的幂格式的总 IO 量和线程运行时间。

-   slat

提交延迟（min 为最小值，max 为最大值，avg 为平均值，stdev 为标准偏差）。这是提交 I/O 所用的时间。对于同步 I/O，此行不显示，因为这种情况下 slat 实际上是完成延迟。该值可以是纳秒、微秒或毫秒，fio 将选择最合适的基数并打印出来（在上面的例子中，纳秒是最好的刻度）。

注意：`--minimal` 模式下的延迟总是以微秒表示。

-   clat

完成延迟，表示从提交到完成 I/O 的时间。

-   lat

总延迟。表示从 fio 创建 I/O 单元到 I/O 操作完成的时间。

-   clat percentiles

clat 的百分位数分布情况。例如中位数指标非常适合描述多少用户需要等待多少时间，比如上面表示有一半的 IO clat 小于 416 usec。

-   bw

带宽。sample 是采样数目，per 表示该线程在其组中接收的总聚合带宽的近似百分比。只有当这个组中的线程在同一个磁盘上时，最后一个值才真正有用，因为它们会竞争磁盘访问，只有当这个组中的线程在同一个磁盘上时，per 才真正有用，因为它们会竞争磁盘访问。

-   iops

每秒完成的 IO 数目。

-   lat (nsec/usec/msec) 

I/O 延迟的分布。250=0.04% 意味着 0.04% 的 IO 在不到 250us 的时间内完成。500=64.11% 意味着 64.11% 的 IO 需要 250 到 499us 才能完成，以此类推。

-   cpu

CPU 负载情况。包括用户和系统时间，该线程经历的上下文切换次数，以及主要和次要 page faults 的数量。CPU 利用率是该报告组中作业的平均值，而上下文和故障计数器是累加的。

-   IO depths

I/O 队列深度的分布。这些数字被分成 2 的幂，每个条目覆盖从该值到下一个条目的深度，例如，16=覆盖从16到31的深度。注意，深度分布不等效提交分布。

-   IO submit

在一次提交调用中提交了多少个 I/O。每个条目表示该数量及以下，直到前一个条目，例如，16=100% 表示我们每个提交调用提交了 9 到 16 个 IO。

-   IO complete

类似 IO submit，只是表示完成的 IO

-   IO issued rwt

发出的 read/write/trim IO 数量

-   IO latency

这些值用于 latency_target 和相关选项。使用这些选项时，本节描述了满足指定延迟目标所需的 I/O 深度。

列出每个客户端后，将打印组统计信息。它们看起来会像这样，括号外是 2 的幂格式，括号内是 10 的幂格式。范围表示所有线程中最小和最大值。bw 是带宽，io 是总 IO 统计，run 是运行时间。

```bash
Run status group 0 (all jobs):
   READ: bw=20.9MiB/s (21.9MB/s), 10.4MiB/s-10.8MiB/s (10.9MB/s-11.3MB/s), io=64.0MiB (67.1MB), run=2973-3069msec
  WRITE: bw=1231KiB/s (1261kB/s), 616KiB/s-621KiB/s (630kB/s-636kB/s), io=64.0MiB (67.1MB), run=52747-53223msec
```

最后，打印磁盘统计信息。这是 Linux 特有的。它们看起来会像这样：

```bash
Disk stats (read/write):
  sda: ios=16398/16511, merge=30/162, ticks=6853/819634, in_queue=826487, util=100.00%
```

ios 表示所有组执行 IO 的数量，merge 表示由 I/O 调度程序执行的合并次数。in_queue 表示花费在磁盘队列的时间，util 表示磁盘繁忙度。







---

# 常用测试项

## 测试磁盘设备

单线程顺序写，粒度为 1MB。数据量一般要选择大一点，这样可以测试磁盘的真实性能，防止写缓存测试出不准确的结果。另外对于同步引擎队列深度是没有意义的。

```bash
# fio -name=write -filename=/dev/sdp -direct=1 -rw=write -ioengine=sync -bs=1M -size=10G -thread -numjobs=1 -runtime=300 -group_reporting
write: (g=0): rw=write, bs=(R) 1024KiB-1024KiB, (W) 1024KiB-1024KiB, (T) 1024KiB-1024KiB, ioengine=sync, iodepth=1
fio-3.7
Starting 1 thread
Jobs: 1 (f=1): [W(1)][100.0%][r=0KiB/s,w=189MiB/s][r=0,w=189 IOPS][eta 00m:00s]
write: (groupid=0, jobs=1): err= 0: pid=28034: Tue Jul 19 15:35:13 2022
  write: IOPS=200, BW=200MiB/s (210MB/s)(10.0GiB/51186msec)
    clat (msec): min=2, max=114, avg= 4.97, stdev= 1.58
     lat (msec): min=2, max=114, avg= 5.00, stdev= 1.58
    clat percentiles (usec):
     |  1.00th=[ 4293],  5.00th=[ 4359], 10.00th=[ 4490], 20.00th=[ 4555],
     | 30.00th=[ 4752], 40.00th=[ 4883], 50.00th=[ 4948], 60.00th=[ 5014],
     | 70.00th=[ 5014], 80.00th=[ 5145], 90.00th=[ 5342], 95.00th=[ 5407],
     | 99.00th=[ 5604], 99.50th=[11600], 99.90th=[21890], 99.95th=[30016],
     | 99.99th=[46400]
   bw (  KiB/s): min=165888, max=225280, per=100.00%, avg=204855.43, stdev=10950.72, samples=102
   iops        : min=  162, max=  220, avg=200.02, stdev=10.68, samples=102
  lat (msec)   : 4=0.19%, 10=99.28%, 20=0.36%, 50=0.17%, 250=0.01%
  cpu          : usr=0.61%, sys=0.75%, ctx=10245, majf=0, minf=8
  IO depths    : 1=100.0%, 2=0.0%, 4=0.0%, 8=0.0%, 16=0.0%, 32=0.0%, >=64=0.0%
     submit    : 0=0.0%, 4=100.0%, 8=0.0%, 16=0.0%, 32=0.0%, 64=0.0%, >=64=0.0%
     complete  : 0=0.0%, 4=100.0%, 8=0.0%, 16=0.0%, 32=0.0%, 64=0.0%, >=64=0.0%
     issued rwts: total=0,10240,0,0 short=0,0,0,0 dropped=0,0,0,0
     latency   : target=0, window=0, percentile=100.00%, depth=1

Run status group 0 (all jobs):
  WRITE: bw=200MiB/s (210MB/s), 200MiB/s-200MiB/s (210MB/s-210MB/s), io=10.0GiB (10.7GB), run=51186-51186msec

Disk stats (read/write):
  sdp: ios=39/40924, merge=0/0, ticks=16/166807, in_queue=166742, util=98.74%
```

对于顺序读来说，如果我们开启了多线程，可能会互相影响从而导致磁盘寻道，所以这里我们使用单线程来测试顺序读的性能。

```bash
# fio -name=read -filename=/dev/sdq -direct=1 -rw=read -ioengine=sync -bs=1M -size=10G -runtime=300
read: (g=0): rw=read, bs=(R) 1024KiB-1024KiB, (W) 1024KiB-1024KiB, (T) 1024KiB-1024KiB, ioengine=sync, iodepth=1
fio-3.7
Starting 1 process
Jobs: 1 (f=1): [R(1)][100.0%][r=198MiB/s,w=0KiB/s][r=198,w=0 IOPS][eta 00m:00s]
read: (groupid=0, jobs=1): err= 0: pid=28085: Tue Jul 19 15:39:28 2022
   read: IOPS=197, BW=198MiB/s (208MB/s)(10.0GiB/51722msec)
    clat (usec): min=4443, max=96415, avg=5049.51, stdev=1520.19
     lat (usec): min=4444, max=96416, avg=5049.68, stdev=1520.19
    clat percentiles (usec):
     |  1.00th=[ 4490],  5.00th=[ 4490], 10.00th=[ 4555], 20.00th=[ 4686],
     | 30.00th=[ 4752], 40.00th=[ 4883], 50.00th=[ 5014], 60.00th=[ 5080],
     | 70.00th=[ 5080], 80.00th=[ 5145], 90.00th=[ 5276], 95.00th=[ 5473],
     | 99.00th=[ 6390], 99.50th=[13042], 99.90th=[29754], 99.95th=[30540],
     | 99.99th=[38536]
   bw (  KiB/s): min=163840, max=217088, per=99.98%, avg=202685.13, stdev=9945.75, samples=103
   iops        : min=  160, max=  212, avg=197.89, stdev= 9.73, samples=103
  lat (msec)   : 10=99.26%, 20=0.54%, 50=0.20%, 100=0.01%
  cpu          : usr=0.04%, sys=0.93%, ctx=10244, majf=0, minf=289
  IO depths    : 1=100.0%, 2=0.0%, 4=0.0%, 8=0.0%, 16=0.0%, 32=0.0%, >=64=0.0%
     submit    : 0=0.0%, 4=100.0%, 8=0.0%, 16=0.0%, 32=0.0%, 64=0.0%, >=64=0.0%
     complete  : 0=0.0%, 4=100.0%, 8=0.0%, 16=0.0%, 32=0.0%, 64=0.0%, >=64=0.0%
     issued rwts: total=10240,0,0,0 short=0,0,0,0 dropped=0,0,0,0
     latency   : target=0, window=0, percentile=100.00%, depth=1

Run status group 0 (all jobs):
   READ: bw=198MiB/s (208MB/s), 198MiB/s-198MiB/s (208MB/s-208MB/s), io=10.0GiB (10.7GB), run=51722-51722msec

Disk stats (read/write):
  sdq: ios=40893/0, merge=0/0, ticks=169138/0, in_queue=169052, util=99.00%
```

但是实际用 fio 多线程测试时，读性能会更高，这是因为每个线程都从一个地方开始顺序读，会有很多缓存命中，从而导致性能看起来很高。

```bash
# fio -name=read -filename=/dev/sdq -direct=1 -rw=read -ioengine=sync -bs=1M -size=10G -thread -numjobs=16 -runtime=300 -group_reporting
read: (g=0): rw=read, bs=(R) 1024KiB-1024KiB, (W) 1024KiB-1024KiB, (T) 1024KiB-1024KiB, ioengine=sync, iodepth=1
...
fio-3.7
Starting 16 threads
Jobs: 12 (f=12): [R(2),_(1),R(1),_(1),R(3),_(1),R(2),_(1),R(4)][20.4%][r=419MiB/s,w=0KiB/s][r=419,w=0 IOPS][eta 19m:28s]
read: (groupid=0, jobs=16): err= 0: pid=27716: Tue Jul 19 14:58:59 2022
   read: IOPS=384, BW=384MiB/s (403MB/s)(113GiB/300111msec)
    clat (msec): min=2, max=6088, avg=41.39, stdev=387.58
     lat (msec): min=2, max=6088, avg=41.39, stdev=387.58
    clat percentiles (msec):
     |  1.00th=[    4],  5.00th=[    5], 10.00th=[    5], 20.00th=[    5],
     | 30.00th=[    5], 40.00th=[    5], 50.00th=[    5], 60.00th=[    6],
     | 70.00th=[    6], 80.00th=[    6], 90.00th=[    6], 95.00th=[    7],
     | 99.00th=[ 1036], 99.50th=[ 3104], 99.90th=[ 6007], 99.95th=[ 6007],
     | 99.99th=[ 6074]
   bw (  KiB/s): min= 2043, max=217088, per=23.05%, avg=90648.96, stdev=80460.17, samples=2601
   iops        : min=    1, max=  212, avg=88.49, stdev=78.58, samples=2601
  lat (msec)   : 4=3.70%, 10=93.40%, 20=0.89%, 50=0.54%, 100=0.17%
  lat (msec)   : 250=0.11%, 500=0.03%, 750=0.02%, 1000=0.03%
  cpu          : usr=0.00%, sys=0.19%, ctx=115363, majf=0, minf=1608
  IO depths    : 1=100.0%, 2=0.0%, 4=0.0%, 8=0.0%, 16=0.0%, 32=0.0%, >=64=0.0%
     submit    : 0=0.0%, 4=100.0%, 8=0.0%, 16=0.0%, 32=0.0%, 64=0.0%, >=64=0.0%
     complete  : 0=0.0%, 4=100.0%, 8=0.0%, 16=0.0%, 32=0.0%, 64=0.0%, >=64=0.0%
     issued rwts: total=115280,0,0,0 short=0,0,0,0 dropped=0,0,0,0
     latency   : target=0, window=0, percentile=100.00%, depth=1

Run status group 0 (all jobs):
   READ: bw=384MiB/s (403MB/s), 384MiB/s-384MiB/s (403MB/s-403MB/s), io=113GiB (121GB), run=300111-300111msec

Disk stats (read/write):
  sdq: ios=461112/0, merge=0/0, ticks=18202714/0, in_queue=18209671, util=100.00%
```

最后我们使用异步 IO 来测试一下顺序读的性能，此时我们就要设置队列深度来加大负载压力。

```bash
# fio -name=read -filename=/dev/sdq -direct=1 -rw=read -iodepth=32 -ioengine=posixaio -bs=1M -size=10G -runtime=300
read: (g=0): rw=read, bs=(R) 1024KiB-1024KiB, (W) 1024KiB-1024KiB, (T) 1024KiB-1024KiB, ioengine=posixaio, iodepth=32
fio-3.7
Starting 1 process
Jobs: 1 (f=1): [R(1)][100.0%][r=222MiB/s,w=0KiB/s][r=222,w=0 IOPS][eta 00m:00s]
read: (groupid=0, jobs=1): err= 0: pid=27751: Tue Jul 19 15:00:58 2022
   read: IOPS=197, BW=197MiB/s (207MB/s)(10.0GiB/51849msec)
    slat (nsec): min=88, max=69729, avg=306.47, stdev=763.16
    clat (msec): min=148, max=262, avg=161.94, stdev=11.22
     lat (msec): min=148, max=262, avg=161.94, stdev=11.22
    clat percentiles (msec):
     |  1.00th=[  153],  5.00th=[  153], 10.00th=[  153], 20.00th=[  155],
     | 30.00th=[  157], 40.00th=[  159], 50.00th=[  159], 60.00th=[  163],
     | 70.00th=[  163], 80.00th=[  171], 90.00th=[  171], 95.00th=[  180],
     | 99.00th=[  211], 99.50th=[  218], 99.90th=[  262], 99.95th=[  262],
     | 99.99th=[  264]
   bw (  KiB/s): min=131072, max=262144, per=99.97%, avg=202175.72, stdev=22777.71, samples=103
   iops        : min=  128, max=  256, avg=197.41, stdev=22.25, samples=103
  lat (msec)   : 250=99.69%, 500=0.31%
  cpu          : usr=0.08%, sys=0.04%, ctx=2560, majf=0, minf=51
  IO depths    : 1=0.1%, 2=0.1%, 4=0.1%, 8=25.0%, 16=50.0%, 32=24.9%, >=64=0.0%
     submit    : 0=0.0%, 4=100.0%, 8=0.0%, 16=0.0%, 32=0.0%, 64=0.0%, >=64=0.0%
     complete  : 0=0.0%, 4=97.5%, 8=0.1%, 16=0.0%, 32=2.5%, 64=0.0%, >=64=0.0%
     issued rwts: total=10240,0,0,0 short=0,0,0,0 dropped=0,0,0,0
     latency   : target=0, window=0, percentile=100.00%, depth=32

Run status group 0 (all jobs):
   READ: bw=197MiB/s (207MB/s), 197MiB/s-197MiB/s (207MB/s-207MB/s), io=10.0GiB (10.7GB), run=51849-51849msec

Disk stats (read/write):
  sdq: ios=40789/0, merge=0/0, ticks=167320/0, in_queue=167246, util=98.69%
```

对于随机写的测试，我们一般使用比较小的粒度如 4k。

```bash
# fio -name=randwrite -filename=/dev/sdp -direct=1 -rw=randwrite -ioengine=sync -bs=4k -size=10G -thread -numjobs=16 -runtime=600 -group_reporting
randwrite: (g=0): rw=randwrite, bs=(R) 4096B-4096B, (W) 4096B-4096B, (T) 4096B-4096B, ioengine=sync, iodepth=1
...
fio-3.7
Starting 16 threads
Jobs: 16 (f=16): [w(16)][100.0%][r=0KiB/s,w=2078KiB/s][r=0,w=519 IOPS][eta 00m:00s]
randwrite: (groupid=0, jobs=16): err= 0: pid=28143: Tue Jul 19 15:50:54 2022
  write: IOPS=573, BW=2296KiB/s (2351kB/s)(1345MiB/600044msec)
    clat (usec): min=656, max=138430, avg=27873.91, stdev=8277.16
     lat (usec): min=657, max=138430, avg=27874.15, stdev=8277.16
    clat percentiles (msec):
     |  1.00th=[   13],  5.00th=[   18], 10.00th=[   20], 20.00th=[   22],
     | 30.00th=[   24], 40.00th=[   26], 50.00th=[   28], 60.00th=[   29],
     | 70.00th=[   31], 80.00th=[   33], 90.00th=[   37], 95.00th=[   42],
     | 99.00th=[   58], 99.50th=[   65], 99.90th=[   80], 99.95th=[   84],
     | 99.99th=[  111]
   bw (  KiB/s): min=  103, max=  680, per=6.25%, avg=143.44, stdev=19.09, samples=19200
   iops        : min=   25, max=  170, avg=35.82, stdev= 4.78, samples=19200
  lat (usec)   : 750=0.01%, 1000=0.01%
  lat (msec)   : 2=0.01%, 4=0.01%, 10=0.59%, 20=11.58%, 50=85.64%
  lat (msec)   : 100=2.15%, 250=0.01%
  cpu          : usr=0.01%, sys=0.05%, ctx=344499, majf=0, minf=13
  IO depths    : 1=100.0%, 2=0.0%, 4=0.0%, 8=0.0%, 16=0.0%, 32=0.0%, >=64=0.0%
     submit    : 0=0.0%, 4=100.0%, 8=0.0%, 16=0.0%, 32=0.0%, 64=0.0%, >=64=0.0%
     complete  : 0=0.0%, 4=100.0%, 8=0.0%, 16=0.0%, 32=0.0%, 64=0.0%, >=64=0.0%
     issued rwts: total=0,344390,0,0 short=0,0,0,0 dropped=0,0,0,0
     latency   : target=0, window=0, percentile=100.00%, depth=1

Run status group 0 (all jobs):
  WRITE: bw=2296KiB/s (2351kB/s), 2296KiB/s-2296KiB/s (2351kB/s-2351kB/s), io=1345MiB (1411MB), run=600044-600044msec

Disk stats (read/write):
  sdp: ios=76/344262, merge=0/0, ticks=28/9591929, in_queue=9592096, util=100.00%
```

随机读和随机写类似。

```bash
# fio -name=randread -filename=/dev/sdq -direct=1 -rw=randread -ioengine=sync -bs=4k -size=10G -thread -numjobs=16 -runtime=300 -group_reporting
randread: (g=0): rw=randread, bs=(R) 4096B-4096B, (W) 4096B-4096B, (T) 4096B-4096B, ioengine=sync, iodepth=1
...
fio-3.7
Starting 16 threads
Jobs: 16 (f=16): [r(16)][100.0%][r=2150KiB/s,w=0KiB/s][r=537,w=0 IOPS][eta 00m:00s]
randread: (groupid=0, jobs=16): err= 0: pid=27830: Tue Jul 19 15:25:10 2022
   read: IOPS=512, BW=2052KiB/s (2101kB/s)(601MiB/300106msec)
    clat (usec): min=299, max=532397, avg=31185.83, stdev=35233.68
     lat (usec): min=299, max=532398, avg=31185.97, stdev=35233.68
    clat percentiles (msec):
     |  1.00th=[    3],  5.00th=[    4], 10.00th=[    6], 20.00th=[    8],
     | 30.00th=[   11], 40.00th=[   14], 50.00th=[   19], 60.00th=[   26],
     | 70.00th=[   34], 80.00th=[   48], 90.00th=[   74], 95.00th=[  102],
     | 99.00th=[  171], 99.50th=[  201], 99.90th=[  275], 99.95th=[  309],
     | 99.99th=[  401]
   bw (  KiB/s): min=   16, max=  272, per=6.25%, avg=128.22, stdev=36.10, samples=9600
   iops        : min=    4, max=   68, avg=32.03, stdev= 9.03, samples=9600
  lat (usec)   : 500=0.05%, 750=0.01%
  lat (msec)   : 2=0.03%, 4=5.49%, 10=23.62%, 20=23.74%, 50=28.18%
  lat (msec)   : 100=13.62%, 250=5.12%, 500=0.15%, 750=0.01%
  cpu          : usr=0.01%, sys=0.03%, ctx=153926, majf=0, minf=30
  IO depths    : 1=100.0%, 2=0.0%, 4=0.0%, 8=0.0%, 16=0.0%, 32=0.0%, >=64=0.0%
     submit    : 0=0.0%, 4=100.0%, 8=0.0%, 16=0.0%, 32=0.0%, 64=0.0%, >=64=0.0%
     complete  : 0=0.0%, 4=100.0%, 8=0.0%, 16=0.0%, 32=0.0%, 64=0.0%, >=64=0.0%
     issued rwts: total=153918,0,0,0 short=0,0,0,0 dropped=0,0,0,0
     latency   : target=0, window=0, percentile=100.00%, depth=1

Run status group 0 (all jobs):
   READ: bw=2052KiB/s (2101kB/s), 2052KiB/s-2052KiB/s (2101kB/s-2101kB/s), io=601MiB (630MB), run=300106-300106msec

Disk stats (read/write):
  sdq: ios=153917/0, merge=0/0, ticks=4799037/0, in_queue=4799125, util=100.00%
```







## 测试文件系统

首先还是测试顺序写，还是先按单线程测试，这样就是只有一个线程在写一个 1G 的文件，IO 大小为 1MB。

```bash
# fio -name=test1 -directory=/mnt/0 -direct=1 -rw=write -ioengine=sync -bs=1M -size=10G -thread -numjobs=1 -runtime=300 -group_reporting
test1: (g=0): rw=write, bs=(R) 1024KiB-1024KiB, (W) 1024KiB-1024KiB, (T) 1024KiB-1024KiB, ioengine=sync, iodepth=1
fio-3.7
Starting 1 thread
test1: Laying out IO file (1 file / 10240MiB)
Jobs: 1 (f=1): [W(1)][100.0%][r=0KiB/s,w=209MiB/s][r=0,w=209 IOPS][eta 00m:00s]
test1: (groupid=0, jobs=1): err= 0: pid=28370: Tue Jul 19 16:09:58 2022
  write: IOPS=201, BW=201MiB/s (211MB/s)(10.0GiB/50819msec)
    clat (usec): min=2250, max=56912, avg=4933.79, stdev=1276.93
     lat (usec): min=2284, max=56940, avg=4961.31, stdev=1276.78
    clat percentiles (usec):
     |  1.00th=[ 4490],  5.00th=[ 4490], 10.00th=[ 4555], 20.00th=[ 4686],
     | 30.00th=[ 4686], 40.00th=[ 4752], 50.00th=[ 4883], 60.00th=[ 5080],
     | 70.00th=[ 5080], 80.00th=[ 5145], 90.00th=[ 5211], 95.00th=[ 5211],
     | 99.00th=[ 5211], 99.50th=[ 5276], 99.90th=[27657], 99.95th=[38536],
     | 99.99th=[46400]
   bw (  KiB/s): min=184320, max=217088, per=99.97%, avg=206268.09, stdev=6303.09, samples=101
   iops        : min=  180, max=  212, avg=201.42, stdev= 6.16, samples=101
  lat (msec)   : 4=0.61%, 10=99.13%, 20=0.13%, 50=0.13%, 100=0.01%
  cpu          : usr=0.64%, sys=0.91%, ctx=10251, majf=0, minf=7
  IO depths    : 1=100.0%, 2=0.0%, 4=0.0%, 8=0.0%, 16=0.0%, 32=0.0%, >=64=0.0%
     submit    : 0=0.0%, 4=100.0%, 8=0.0%, 16=0.0%, 32=0.0%, 64=0.0%, >=64=0.0%
     complete  : 0=0.0%, 4=100.0%, 8=0.0%, 16=0.0%, 32=0.0%, 64=0.0%, >=64=0.0%
     issued rwts: total=0,10240,0,0 short=0,0,0,0 dropped=0,0,0,0
     latency   : target=0, window=0, percentile=100.00%, depth=1

Run status group 0 (all jobs):
  WRITE: bw=201MiB/s (211MB/s), 201MiB/s-201MiB/s (211MB/s-211MB/s), io=10.0GiB (10.7GB), run=50819-50819msec

Disk stats (read/write):
  sdx: ios=0/40817, merge=0/3, ticks=0/164640, in_queue=164577, util=98.43%
```

然后我们生成 16 个线程，这样每个线程都写自己大小为 1G 的文件，按照粒度 1M 发送 IO。我们会发现性能其实反而下降了。

```bash
# fio -name=test2 -directory=/mnt/0 -direct=1 -rw=write -ioengine=sync -bs=1M -size=10G -thread -numjobs=16 -runtime=300 -group_reporting
test2: (g=0): rw=write, bs=(R) 1024KiB-1024KiB, (W) 1024KiB-1024KiB, (T) 1024KiB-1024KiB, ioengine=sync, iodepth=1
...
fio-3.7
Starting 16 threads
Jobs: 16 (f=16): [W(16)][100.0%][r=0KiB/s,w=118MiB/s][r=0,w=118 IOPS][eta 00m:00s]
test2: (groupid=0, jobs=16): err= 0: pid=28430: Tue Jul 19 16:25:28 2022
  write: IOPS=115, BW=115MiB/s (121MB/s)(33.8GiB/300136msec)
    clat (msec): min=14, max=243, avg=138.86, stdev= 7.64
     lat (msec): min=15, max=243, avg=138.90, stdev= 7.64
    clat percentiles (msec):
     |  1.00th=[  127],  5.00th=[  130], 10.00th=[  132], 20.00th=[  136],
     | 30.00th=[  136], 40.00th=[  138], 50.00th=[  138], 60.00th=[  138],
     | 70.00th=[  140], 80.00th=[  144], 90.00th=[  146], 95.00th=[  150],
     | 99.00th=[  163], 99.50th=[  169], 99.90th=[  186], 99.95th=[  194],
     | 99.99th=[  226]
   bw (  KiB/s): min= 6131, max=10240, per=6.25%, avg=7369.35, stdev=1008.51, samples=9600
   iops        : min=    5, max=   10, avg= 7.13, stdev= 1.02, samples=9600
  lat (msec)   : 20=0.01%, 50=0.08%, 100=0.04%, 250=99.87%
  cpu          : usr=0.03%, sys=0.12%, ctx=34603, majf=0, minf=8
  IO depths    : 1=100.0%, 2=0.0%, 4=0.0%, 8=0.0%, 16=0.0%, 32=0.0%, >=64=0.0%
     submit    : 0=0.0%, 4=100.0%, 8=0.0%, 16=0.0%, 32=0.0%, 64=0.0%, >=64=0.0%
     complete  : 0=0.0%, 4=100.0%, 8=0.0%, 16=0.0%, 32=0.0%, 64=0.0%, >=64=0.0%
     issued rwts: total=0,34565,0,0 short=0,0,0,0 dropped=0,0,0,0
     latency   : target=0, window=0, percentile=100.00%, depth=1

Run status group 0 (all jobs):
  WRITE: bw=115MiB/s (121MB/s), 115MiB/s-115MiB/s (121MB/s-121MB/s), io=33.8GiB (36.2GB), run=300136-300136msec

Disk stats (read/write):
  sdx: ios=0/138264, merge=0/0, ticks=0/19001243, in_queue=19002795, util=100.00%
```

我们再尝试写多个文件，每个线程写入 1024 个文件，每个文件大小为 5 MB，这里限制每个线程最多打开 10 个文件，以免文件描述符打开过多。

```bash
# fio -name=test3 -directory=/mnt/0 -direct=1 -rw=write -ioengine=sync -bs=1M -nrfiles=1024 -filesize=5M -openfiles=10 -thread -numjobs=16 -runtime=300 -group_reporting
test3: (g=0): rw=write, bs=(R) 1024KiB-1024KiB, (W) 1024KiB-1024KiB, (T) 1024KiB-1024KiB, ioengine=sync, iodepth=1
...
fio-3.7
Starting 16 threads
test3: Laying out IO files (1024 files / total 5120MiB)
test3: Laying out IO files (1024 files / total 5120MiB)
test3: Laying out IO files (1024 files / total 5120MiB)
test3: Laying out IO files (1024 files / total 5120MiB)
test3: Laying out IO files (1024 files / total 5120MiB)
test3: Laying out IO files (1024 files / total 5120MiB)
test3: Laying out IO files (1024 files / total 5120MiB)
test3: Laying out IO files (1024 files / total 5120MiB)
test3: Laying out IO files (1024 files / total 5120MiB)
test3: Laying out IO files (1024 files / total 5120MiB)
test3: Laying out IO files (1024 files / total 5120MiB)
test3: Laying out IO files (1024 files / total 5120MiB)
test3: Laying out IO files (1024 files / total 5120MiB)
test3: Laying out IO files (1024 files / total 5120MiB)
test3: Laying out IO files (1024 files / total 5120MiB)
test3: Laying out IO files (1024 files / total 5120MiB)
Jobs: 16 (f=160): [W(16)][100.0%][r=0KiB/s,w=119MiB/s][r=0,w=119 IOPS][eta 00m:00s]
test3: (groupid=0, jobs=16): err= 0: pid=28778: Tue Jul 19 16:51:36 2022
  write: IOPS=116, BW=117MiB/s (123MB/s)(34.3GiB/300127msec)
    clat (msec): min=3, max=406, avg=136.70, stdev=10.11
     lat (msec): min=3, max=406, avg=136.75, stdev=10.10
    clat percentiles (msec):
     |  1.00th=[  121],  5.00th=[  126], 10.00th=[  128], 20.00th=[  131],
     | 30.00th=[  134], 40.00th=[  136], 50.00th=[  136], 60.00th=[  138],
     | 70.00th=[  140], 80.00th=[  142], 90.00th=[  144], 95.00th=[  150],
     | 99.00th=[  165], 99.50th=[  171], 99.90th=[  194], 99.95th=[  262],
     | 99.99th=[  397]
   bw (  KiB/s): min= 2048, max=10240, per=6.25%, avg=7485.49, stdev=990.63, samples=9600
   iops        : min=    2, max=   10, avg= 7.26, stdev= 1.00, samples=9600
  lat (msec)   : 4=0.01%, 10=0.01%, 20=0.01%, 50=0.07%, 100=0.04%
  lat (msec)   : 250=99.80%, 500=0.07%
  cpu          : usr=0.03%, sys=0.11%, ctx=35173, majf=0, minf=9
  IO depths    : 1=100.0%, 2=0.0%, 4=0.0%, 8=0.0%, 16=0.0%, 32=0.0%, >=64=0.0%
     submit    : 0=0.0%, 4=100.0%, 8=0.0%, 16=0.0%, 32=0.0%, 64=0.0%, >=64=0.0%
     complete  : 0=0.0%, 4=100.0%, 8=0.0%, 16=0.0%, 32=0.0%, 64=0.0%, >=64=0.0%
     issued rwts: total=0,35108,0,0 short=0,0,0,0 dropped=0,0,0,0
     latency   : target=0, window=0, percentile=100.00%, depth=1

Run status group 0 (all jobs):
  WRITE: bw=117MiB/s (123MB/s), 117MiB/s-117MiB/s (123MB/s-123MB/s), io=34.3GiB (36.8GB), run=300127-300127msec

Disk stats (read/write):
  sdx: ios=0/140938, merge=0/769, ticks=0/19001069, in_queue=19004748, util=100.00%
```

其余的测试类似，就不一一演示了。







## 测试对象存储系统

由于对象存储需要 http 引擎，所以我们需要安装 libcurl 后然后重新编译安装 fio。

```bash
# 安装 libcurl
$ yum install openssl-devel libcurl-devel -y

# 临时升级 GCC 编译环境
$ yum install centos-release-scl -y
$ yum install devtoolset-8-gcc* -y
$ scl enable devtoolset-8 bash
$ gcc -v
...
gcc version 8.3.1 20190311 (Red Hat 8.3.1-3) (GCC) 

# 编译 fio
$ git clone git://git.kernel.dk/fio.git
$ cd fio
$ git checkout fio-3.30
$ ./configure
$ make
$ make install
$ fio -v
fio-3.30
$ fio --enghelp
Available IO engines:
	cpuio
	mmap
	sync
	psync
	vsync
	pvsync
	pvsync2
	null
	net
	netsplice
	ftruncate
	filecreate
	filestat
	filedelete
	exec
	posixaio
	falloc
	e4defrag
	splice
	mtd
	sg
	io_uring
	libaio
	rados
	http
```

我们写一个 jobfile

```bash
; -- start job file --
[s3test]
ioengine=http
size=10G
bs=5M
rw=write
filename=/bucket1/testobj
http_host=127.0.0.1:9000
http_mode=s3
http_s3_key=minioadmin
http_s3_keyid=minioadmin
http_s3_region=us-east-1
http_verbose=0
direct=1
thread
numjobs=32
runtime=300
group_reporting
; -- end job file --
```

测试

```bash
[root@localhost ~]# fio s3 
s3test: (g=0): rw=write, bs=(R) 5120KiB-5120KiB, (W) 5120KiB-5120KiB, (T) 5120KiB-5120KiB, ioengine=http, iodepth=1
...
fio-3.30
Starting 32 threads
Jobs: 4 (f=4): [_(12),W(1),_(6),W(2),_(8),W(1),_(2)][53.0%][w=646MiB/s][w=129 IOPS][eta 04m:27s]
s3test: (groupid=0, jobs=32): err= 0: pid=27405: Wed Jul 20 11:21:46 2022
  write: IOPS=116, BW=582MiB/s (611MB/s)(171GiB/300187msec); 0 zone resets
    clat (msec): min=52, max=1037, avg=274.38, stdev=55.37
     lat (msec): min=52, max=1037, avg=274.63, stdev=55.37
    clat percentiles (msec):
     |  1.00th=[  169],  5.00th=[  197], 10.00th=[  211], 20.00th=[  230],
     | 30.00th=[  245], 40.00th=[  257], 50.00th=[  271], 60.00th=[  284],
     | 70.00th=[  296], 80.00th=[  313], 90.00th=[  342], 95.00th=[  372],
     | 99.00th=[  435], 99.50th=[  468], 99.90th=[  584], 99.95th=[  667],
     | 99.99th=[  860]
   bw (  KiB/s): min=323285, max=1025037, per=100.00%, avg=597557.97, stdev=4804.72, samples=19138
   iops        : min=   32, max=  200, avg=111.22, stdev= 1.01, samples=19138
  lat (msec)   : 100=0.03%, 250=34.41%, 500=65.31%, 750=0.23%, 1000=0.03%
  lat (msec)   : 2000=0.01%
  cpu          : usr=8.05%, sys=0.49%, ctx=1779478, majf=0, minf=5471
  IO depths    : 1=100.0%, 2=0.0%, 4=0.0%, 8=0.0%, 16=0.0%, 32=0.0%, >=64=0.0%
     submit    : 0=0.0%, 4=100.0%, 8=0.0%, 16=0.0%, 32=0.0%, 64=0.0%, >=64=0.0%
     complete  : 0=0.0%, 4=100.0%, 8=0.0%, 16=0.0%, 32=0.0%, 64=0.0%, >=64=0.0%
     issued rwts: total=0,34968,0,0 short=0,0,0,0 dropped=0,0,0,0
     latency   : target=0, window=0, percentile=100.00%, depth=1

Run status group 0 (all jobs):
  WRITE: bw=582MiB/s (611MB/s), 582MiB/s-582MiB/s (611MB/s-611MB/s), io=171GiB (183GB), run=300187-300187msec
```

这里我发现 fio 测试的性能没有 minio 自带的测试工具高。

```bash
[root@localhost ~]# mc support perf myminio --duration 60s --size=5MiB --concurrent=32

   	THROUGHPUT   	IOPS           
PUT	1.1 GiB/s	228 objs/s	
GET	1.0 GiB/s	205 objs/s	

Speedtest: MinIO DEVELOPMENT.GOGET, 1 servers, 36 drives, 5.0 MiB objects, 8 threads
```

于是打开 top 观察，发现 fio 占用 CPU 资源非常多，我这是 8 核的机器，CPU 已经被 minio 和 fio 全部瓜分了。

```bash
top - 11:15:32 up 5 days,  1:19,  3 users,  load average: 11.90, 11.99, 7.25
Tasks: 506 total,   2 running, 504 sleeping,   0 stopped,   0 zombie
%Cpu0  : 64.9 us, 25.8 sy,  0.0 ni,  1.3 id,  3.7 wa,  0.0 hi,  4.3 si,  0.0 st
%Cpu1  : 79.5 us, 13.5 sy,  0.0 ni,  2.0 id,  4.4 wa,  0.0 hi,  0.7 si,  0.0 st
%Cpu2  : 80.6 us, 14.7 sy,  0.0 ni,  1.0 id,  3.3 wa,  0.0 hi,  0.3 si,  0.0 st
%Cpu3  : 76.5 us, 13.8 sy,  0.0 ni,  2.7 id,  6.7 wa,  0.0 hi,  0.3 si,  0.0 st
%Cpu4  : 80.3 us, 14.4 sy,  0.0 ni,  1.3 id,  3.0 wa,  0.0 hi,  1.0 si,  0.0 st
%Cpu5  : 78.9 us, 14.1 sy,  0.0 ni,  2.7 id,  4.0 wa,  0.0 hi,  0.3 si,  0.0 st
%Cpu6  : 78.1 us, 14.1 sy,  0.0 ni,  2.0 id,  5.1 wa,  0.0 hi,  0.7 si,  0.0 st
%Cpu7  : 75.8 us, 15.4 sy,  0.0 ni,  2.7 id,  5.7 wa,  0.0 hi,  0.3 si,  0.0 st
KiB Mem : 16201188 total,  1867080 free,  1816688 used, 12517420 buff/cache
KiB Swap:  8191996 total,  8191996 free,        0 used. 13763520 avail Mem 

  PID USER      PR  NI    VIRT    RES    SHR S  %CPU %MEM     TIME+ COMMAND                                                                                                
25055 root      20   0 4712304   1.3g  23968 S 458.1  8.1  27:59.98 minio                                                                                                  
27326 root      20   0 3099540 183340  11028 S 273.4  1.1   0:47.24 fio                                                                                                    
27368 root      20   0       0      0      0 R   8.6  0.0   0:01.44 kworker/0:1                                                                                            
 2877 root       0 -20       0      0      0 S   4.0  0.0   0:33.16 kworker/0:1H                                                                                           
   14 root      20   0       0      0      0 S   0.7  0.0   0:00.68 ksoftirqd/1          
```

使用 perf 观察，发现 CPU 资源基本都是 libcrypto.so 动态库占用了。

```bash
$ perf top -p `pidof fio`
Samples: 436K of event 'cycles:ppp', 4000 Hz, Event count (approx.): 146491065746 lost: 0/0 drop: 0/0                                                                       
Overhead  Shared Object        Symbol                                                                                                                                       
   3.26%  libc-2.17.so         [.] __memcpy_ssse3_back
   1.35%  [kernel]             [k] copy_user_enhanced_fast_string
   1.07%  fio                  [.] get_io_u
   0.97%  libcrypto.so.1.0.2k  [.] 0x000000000008072e
   0.97%  libcrypto.so.1.0.2k  [.] 0x0000000000080483
   0.91%  libcrypto.so.1.0.2k  [.] 0x00000000000804c2
   0.90%  libcrypto.so.1.0.2k  [.] 0x00000000000804a6
   0.87%  libcrypto.so.1.0.2k  [.] 0x00000000000804e5
   0.87%  libcrypto.so.1.0.2k  [.] 0x000000000008074f
   0.82%  libcrypto.so.1.0.2k  [.] 0x0000000000080547
   0.81%  libcrypto.so.1.0.2k  [.] 0x000000000008070c
   0.81%  libcrypto.so.1.0.2k  [.] 0x0000000000080524
   0.78%  libcrypto.so.1.0.2k  [.] 0x00000000000805aa
   0.75%  libcrypto.so.1.0.2k  [.] 0x0000000000080507
   0.75%  libcrypto.so.1.0.2k  [.] 0x00000000000806ce
   0.75%  libcrypto.so.1.0.2k  [.] 0x00000000000805ea
   0.74%  libcrypto.so.1.0.2k  [.] 0x0000000000080587
   0.73%  libcrypto.so.1.0.2k  [.] 0x000000000008076c
   0.73%  libcrypto.so.1.0.2k  [.] 0x000000000008060c
   0.72%  libcrypto.so.1.0.2k  [.] 0x000000000008056a
   0.72%  libcrypto.so.1.0.2k  [.] 0x000000000008064b
   0.72%  libcrypto.so.1.0.2k  [.] 0x00000000000805cd
   0.68%  libcrypto.so.1.0.2k  [.] 0x000000000008066d
   0.67%  libcrypto.so.1.0.2k  [.] 0x00000000000806ac
   0.66%  libcrypto.so.1.0.2k  [.] 0x000000000008062f
   0.62%  libcrypto.so.1.0.2k  [.] 0x0000000000080460
   0.61%  libcrypto.so.1.0.2k  [.] 0x00000000000806f1
   0.60%  libcrypto.so.1.0.2k  [.] 0x000000000008068f
   0.52%  libcrypto.so.1.0.2k  [.] 0x000000000007f49c
   0.46%  libcrypto.so.1.0.2k  [.] 0x000000000007f488
   0.43%  [kernel]             [k] system_call_after_swapgs
   0.43%  libcrypto.so.1.0.2k  [.] 0x000000000007fce5

```





---

# 参考与感谢

-   [Welcome to FIO’s documentation!](https://fio.readthedocs.io/en/latest/)
-   [fio Github](https://github.com/axboe/fio)
-   [fio使用指南（最全的参数说明）](https://blog.csdn.net/sch0120/article/details/76154205)
-   [CentOS 7升级gcc版本](https://www.likecs.com/show-205131329.html)