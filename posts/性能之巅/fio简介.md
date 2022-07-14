# 快速开始







---

# Job file 参数

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
| lockfile          | str  | Fio 默认在对文件进行 I/O 操作之前不锁定任何文件。这通常用于模拟共享文件的真实工作负载。锁定模式包括：<br/>- none：无锁（默认）<br/>- exclusive：次只能有一个线程或进程执行 I/O<br/>- readwrite：可以同时读，但写得独占 |
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

>   -   **bs=256k**
>
>       means 256k for reads, writes and trims.
>
>       **bs=8k,32k**
>
>       means 8k for reads, 32k for writes and trims.
>
>       **bs=8k,32k,**
>
>       means 8k for reads, 32k for writes, and default for trims.
>
>       **bs=,8k**
>
>       means default for reads, 8k for writes and trims.
>
>       **bs=,8k,**
>
>       means default for reads, 8k for writes, and default for trims.

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



## IO 大小









-   ioengine=libaio 指定io引擎使用libaio方式。libaio：Linux本地异步I/O。请注意，Linux可能只支持具有非缓冲I/O的排队行为（设置为“direct=1”或“buffered=0”）；rbd:通过librbd直接访问CEPH Rados 
-   iodepth=16 队列的深度为16.在异步模式下，CPU不能一直无限的发命令到SSD。比如SSD执行读写如果发生了卡顿，那有可能系统会一直不停的发命令，几千个，甚至几万个，这样一方面SSD扛不住，另一方面这么多命令会很占内存，系统也要挂掉了。这样，就带来一个参数叫做队列深度。
    Block Devices（RBD），无需使用内核RBD驱动程序（rbd.ko）。该参数包含很多ioengine，如：libhdfs/rdma等
-   group_reporting 关于显示结果的，汇总每个进程的信息。



磁盘读写常用测试点：

1. Read=100% Ramdon=100% rw=randread (100%随机读)
2. Read=100% Sequence=100% rw=read （100%顺序读）
3. Write=100% Sequence=100% rw=write （100%顺序写）
4. Write=100% Ramdon=100% rw=randwrite （100%随机写）
5. Read=70% Sequence=100% rw=rw, rwmixread=70, rwmixwrite=30
（70%顺序读，30%顺序写）
6. Read=70% Ramdon=100% rw=randrw, rwmixread=70, rwmixwrite=30
(70%随机读，30%随机写)





---

# 常用测试项













---

# 参考与感谢

-   [Welcome to FIO’s documentation!](https://fio.readthedocs.io/en/latest/)
-   [fio Github](https://github.com/axboe/fio)