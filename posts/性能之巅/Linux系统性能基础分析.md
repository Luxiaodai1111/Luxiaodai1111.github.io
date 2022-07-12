本文不介绍工具安装，如果缺失，在 CentOS 上基本都可以使用 yum install 方式安装。

---

# CPU

## top

top 的原理很简单，就是读取 /proc 下的信息并展示，所以有时候你会看到 top 本身也占用了不少资源，是因为它也要去操作文件。

top 输出一般如下，第一行输出与 uptime 类似，load average 对应相应最近 1、5 和 15 分钟内的平均负载。

```bash
top - 13:58:41 up 8 days,  4:50,  3 users,  load average: 9.66, 2.40, 0.84
Tasks: 461 total,   1 running, 460 sleeping,   0 stopped,   0 zombie
%Cpu(s): 15.5 us, 25.7 sy,  0.0 ni, 14.6 id, 43.6 wa,  0.0 hi,  0.6 si,  0.0 st
KiB Mem：16201188 total,   157880 free,   970568 used, 15072740 buff/cache
KiB Swap:  8191996 total,  8191996 free,        0 used. 14745312 avail Mem 

  PID USER      PR  NI    VIRT    RES    SHR S  %CPU %MEM     TIME+ COMMAND                                                                                                
25083 root      20   0 1896520 650068  20536 S 291.4  4.0  27:20.05 minio                                                                                              
   65 root      20   0       0      0      0 S  18.9  0.0   0:47.44 kswapd0                                                                                                
 3768 root      20   0       0      0      0 S  15.0  0.0   0:05.84 kworker/0:1                                                                                            
 2875 root       0 -20       0      0      0 S   6.3  0.0   0:22.03 kworker/0:1H                                                                                           
 5631 root      20   0  728412  18956   7868 S   1.0  0.1   0:00.36 mc                                                                                                     
 5664 root      20   0       0      0      0 S   0.7  0.0   0:00.11 kworker/1:1                                                                                            
28262 root      20   0       0      0      0 S   0.7  0.0   0:00.25 kworker/3:1                                                                                            
    3 root      20   0       0      0      0 S   0.3  0.0   0:25.75 ksoftirqd/0                                                                                            
   44 root      20   0       0      0      0 S   0.3  0.0   0:00.15 ksoftirqd/7                                                                                            
  535 root      20   0       0      0      0 S   0.3  0.0   0:00.98 kworker/7:1                                                                                            
 4782 root      20   0       0      0      0 S   0.3  0.0   0:00.06 kworker/2:1                                                                                            
 5085 root      20   0       0      0      0 S   0.3  0.0   0:00.11 kworker/4:2                                                                                            
 5332 root      20   0       0      0      0 S   0.3  0.0   0:00.10 kworker/2:0                                                                                            
 5650 root      20   0  162292   2656   1600 R   0.3  0.0   0:00.08 top                                                                                                    
17544 root      20   0       0      0      0 S   0.3  0.0   0:00.73 xfsaild/sda     
```

第二行显示的是任务或者进程的总结。进程可以处于不同的状态。这里显示了全部进程的数量。除此之外，还有正在运行、睡眠、停止、僵尸进程的数量。

第三行是 CPU 信息，这里显示了 CPU 时间分配占比，us 表示（未调整优先级）运行用户态进程的时间，sy 是运行内核进程的时间，ni 表示运行已调整优先级的用户进程的 CPU 时间，wa 表示用于等待 IO 完成的 CPU 时间，hi 和 si 分别是处理硬/软中断的时间，st 表示这个虚拟机被 hypervisor 偷去消耗的时间。我们平时一般只用关注 CPU 的整体使用量。

这些进程概括信息可以用 `t` 切换显示。按 `1` 可以显示每个 CPU 的负载情况。

```bash
# 默认格式
Tasks: 461 total,   1 running, 460 sleeping,   0 stopped,   0 zombie
%Cpu(s): 15.5 us, 25.7 sy,  0.0 ni, 14.6 id, 43.6 wa,  0.0 hi,  0.6 si,  0.0 st

# 按下 t 切换显示模式
Tasks: 466 total,   4 running, 462 sleeping,   0 stopped,   0 zombie
%Cpu(s):  14.4/26.6   41[|||||||||||||||||||||||||||||||||||||||||                                                           ]

# 按下 1 列出每个 CPU 情况
Tasks: 466 total,   1 running, 465 sleeping,   0 stopped,   0 zombie
%Cpu0 ：  9.2/34.2   43[|||||||||||||||||||||||||||||||||||||||||||                                                         ]
%Cpu1 ： 18.4/27.0   45[|||||||||||||||||||||||||||||||||||||||||||||                                                       ]
%Cpu2 ： 17.7/28.3   46[||||||||||||||||||||||||||||||||||||||||||||||                                                      ]
%Cpu3 ： 17.4/28.9   46[||||||||||||||||||||||||||||||||||||||||||||||                                                      ]
%Cpu4 ： 10.8/22.0   33[|||||||||||||||||||||||||||||||||                                                                   ]
%Cpu5 ： 14.0/24.6   39[|||||||||||||||||||||||||||||||||||||||                                                             ]
%Cpu6 ： 14.6/23.7   38[|||||||||||||||||||||||||||||||||||||||                                                             ]
%Cpu7 ： 13.2/27.4   41[||||||||||||||||||||||||||||||||||||||||                                                            ]
```

第四行和第五行是内存使用情况，可以按下 `m` 来切换显示模式。

```bash
# 按下 m 切换显示模式
KiB Mem： 5.2/16201188 [|||||                                                                                               ]
KiB Swap:  0.0/8191996  [                                                                                                    ]
```

再以下就是系统每个进程的状态了，这些字段的含义如下：

| 字段    | 含义                                                         |
| ------- | ------------------------------------------------------------ |
| PID     | 进程ID，进程的唯一标识符                                     |
| USER    | 进程所有者的实际用户名                                       |
| PR      | 进程的调度优先级。这个字段的一些值是'rt'。这意味这这些进程运行在实时态。 |
| NI      | 进程的 nice值(优先级)，越小的值意味着越高的优先级。          |
| VIRT    | 进程使用的虚拟内存                                           |
| RES     | 驻留内存大小。驻留内存是任务使用的非交换物理内存大小。       |
| SHR     | SHR是进程使用的共享内存。                                    |
| S       | 这个是进程的状态，它有以下不同的值：<br>D - 不可中断的睡眠态<br/>R – 运行态<br/>S – 睡眠态<br/>T – 被跟踪或已停止 |
| %CPU    | 自从上一次更新时到现在任务所使用的 CPU 时间百分比            |
| %MEM    | 进程使用的可用物理内存百分比                                 |
| TIME+   | 任务启动后到现在所使用的全部 CPU 时间，精确到百分之一秒      |
| COMMAND | 运行进程所使用的命令                                         |

我们可以快速定位到底那个进程消耗掉了 CPU 的资源。比如这里可以看到 minio 占用了 291.4% 的 CPU 资源，为什么会超过 100%呢，这是因为 top 就是从 /proc 中读取信息，因为有多个核，这里只做了简单的加法而没有按 CPU 数量进行标准化，比如我这里有 8 个核，那么其实跑满应该算 800%。

如果你喜欢花里胡哨的颜色可以按下 `Z` ，top 会向用户显示一个改变 top 命令的输出颜色的屏幕，可以为 8 个任务区域选择 8 种颜色。

另外 htop 是 top 的一个变种，提供了更多的交互功能，但是比 top 多 4 倍多的系统调用，会更大程度地影响系统。由于 top 是对 /proc 进行快照，一些短命进程不会被 top 捕捉到，可以使用 atop 去分析。



## perf

上面 top 只能看到系统里进程的概况，以及哪些进程占用比较多的 CPU 资源，如果想查看到底是哪些代码在占用资源，可以使用 `perf top` 命令来查看，输出如下，可以看到有很多 CPU 都耗在了数据内存拷贝上，至于合不合理，那就需要具体问题具体判断了。

```bash
Samples: 71K of event 'cycles:ppp', 4000 Hz, Event count (approx.): 45608630139 lost: 0/0 drop: 0/0                                                                         
Overhead  Shared Object            Symbol                                                                                                                                   
  11.76%  minio                    [.] runtime.memmove
  10.87%  [kernel]                 [k] copy_user_enhanced_fast_string
   5.26%  minio                    [.] github.com/minio/highwayhash.updateAVX2.abi0
   3.11%  minio                    [.] github.com/klauspost/reedsolomon.mulAvxTwo_5x1_64.abi0
   1.99%  [kernel]                 [k] __list_del_entry
   1.29%  [kernel]                 [k] iov_iter_fault_in_readable
   1.23%  [kernel]                 [k] radix_tree_descend
   1.04%  minio                    [.] github.com/minio/pkg/randreader.xorSlice.abi0
   0.99%  [kernel]                 [k] get_page_from_freelist
   0.84%  [kernel]                 [k] mark_page_accessed
   0.82%  [kernel]                 [k] __mem_cgroup_commit_charge
   0.78%  [kernel]                 [k] __wake_up_bit
   0.76%  [kernel]                 [k] xfs_destroy_ioend
   0.73%  [kernel]                 [k] xfs_do_writepage
   0.71%  [kernel]                 [k] kmem_cache_alloc
   0.71%  [kernel]                 [k] _raw_qspin_lock
   0.67%  [kernel]                 [k] _raw_spin_lock_irqsave
   0.61%  [kernel]                 [k] free_pcppages_bulk
   0.60%  minio                    [.] runtime.mallocgc
   0.59%  [kernel]                 [k] __mem_cgroup_uncharge_common
   0.58%  [kernel]                 [k] __list_add
...
```

 perf 是 linux 官方的剖析器，功能十分强大，感兴趣的同学可以自行去了解 perf 的使用，本文只介绍系统分析基础，下面给出一些示例：

```bash
# 展示全系统的 PMC 统计信息，为期 5s
[root@localhost ~]# perf stat -a -- sleep 5

 Performance counter stats for 'system wide':

         40,030.13 msec cpu-clock                 #    7.999 CPUs utilized          
           902,541      context-switches          #    0.023 M/sec                  
            24,871      cpu-migrations            #    0.621 K/sec                  
               223      page-faults               #    0.006 K/sec                  
    64,615,741,037      cycles                    #    1.614 GHz                    
    36,139,728,452      instructions              #    0.56  insn per cycle         
     5,547,233,177      branches                  #  138.576 M/sec                  
        95,075,033      branch-misses             #    1.71% of all branches        

       5.004554700 seconds time elapsed

# 展示每秒上下文切换频率，perf list 可以查看可统计的事件
[root@localhost ~]# perf stat -e sched:sched_switch -a -I 1000
#           time             counts unit events
     1.000107281            178,165      sched:sched_switch                                          
     2.002199962            170,544      sched:sched_switch                                          
     3.002318446            151,407      sched:sched_switch                                          
     4.002431181            170,984      sched:sched_switch
     ...
     29.006921722            179,526      sched:sched_switch                                          
     30.007039573            169,576      sched:sched_switch                                          
     31.007249693             53,921      sched:sched_switch                                          
     32.007463661              1,880      sched:sched_switch                                          
     33.007859443              2,115      sched:sched_switch 
     ...
     
```

perf 可以用来剖析 CPU 调用路径，查看 CPU 时间到底花在哪了，当然每个语言也有自己的性能分析工具，大家用自己顺手的就行。

```bash
# 对 minio 进程采样 5s，采样结束后生成 perf.data 文件
root@localhost ~]# perf record -p `pidof minio` -g -F 99 -- sleep 5
[ perf record: Woken up 1 times to write data ]
[ perf record: Captured and wrote 0.662 MB perf.data (2690 samples) ]

# 对文件进行分析
[root@localhost ~]# perf report --stdio
# To display the perf.data header info, please use --header/--header-only options.
#
#
# Total Lost Samples: 0
#
# Samples: 2K of event 'cycles:ppp'
# Event count (approx.): 169065810574
#
# Children      Self  Command  Shared Object      Symbol                                                                        
# ........  ........  .......  .................  ..............................................................................
#
    98.65%     0.00%  minio    minio              [.] runtime.goexit.abi0
            |
            ---runtime.goexit.abi0
               |          
               |--66.26%--github.com/minio/minio/cmd.newStreamingBitrotWriter.func1
               |          |          
               |           --66.26%--github.com/minio/minio/cmd.(*xlStorageDiskIDCheck).CreateFile
               |                     |          
               |                      --65.88%--github.com/minio/minio/cmd.(*xlStorage).CreateFile
               |                                |          
               |                                |--48.76%--github.com/minio/minio/internal/ioutil.CopyAligned
               |                                |          |          
               |                                |          |--37.37%--github.com/minio/minio/internal/ioutil.CopyAligned.func1
...
```



## uptime

uptime 用于查看系统平均负载，最后三个数字是最近 1、5 和 15 分钟内的平均负载。这也是 top 第一行的输出。这个命令主要是用来判断系统在最近 15 分钟内负载变化情况。比如下面输出我们可以得知，系统最近的负载突然高了起来。

```bash
[root@localhost ~]# uptime
 08:46:45 up 8 days, 23:38,  3 users,  load average: 7.70, 1.94, 0.69
```



## vmstat

vmstat 是虚拟内存统计命令，在其最后几行打印了系统级的 CPU 平均负载，与 CPU 相关的字段如下：

-   r：运行队列长度（可运行线程的总数），在 Linux 中，r 是等待的任务总数加上正在运行的任务总数
-   us：用户时间比例
-   sy：内核时间比例
-   id：空闲比例
-   wa：等待 IO 比例
-   st：略

CPU 信息输出和 top 基本类似，但是 vmstat 有自己的侧重点，后面我们会继续提到 vmstat 可以观测到的信息，比如内存，交换分区，IO 统计，系统调用等。

```bash
[root@localhost ~]# vmstat 1
procs -----------memory---------- ---swap-- -----io---- -system-- ------cpu-----
 r  b   swpd   free   buff  cache   si   so    bi    bo   in   cs us sy id wa st
12  3      0 192880    164 14974940    0    0   384   784    4    6  0  0 99  0  0
 1 11      0 171220    164 14997332    0    0     0 1445564 44871 174844 13 27 15 45  0
 0  4      0 192236    164 14977300    0    0   116 1441668 45281 171626 13 27 15 45  0
 7 22      0 163080    164 15001920    0    0     0 1488736 46146 179469 14 28 13 44  0
 1  9      0 169288    164 15000516    0    0    16 1469032 46149 174807 15 28 14 43  0
 4  7      0 192404    164 14978504    0    0    32 1493664 45749 173145 14 29 14 43  0
 0 12      0 166936    164 15003516    0    0     0 1488728 46421 178536 13 29 15 43  0
...
```



## mpstat

多处理器统计工具，能够报告每个 CPU 的统计情况。这里的信息和 top 也类似，主要是用来识别是否有某个 CPU 过忙的情况，在 top 界面按下 1 也可实时观测，只是少了最后统计的平均时间结果。

```bash
# -P ALL 打印每个 CPU 信息
[root@localhost ~]# mpstat -P ALL 1
Linux 3.10.0-957.el7.x86_64 (localhost.localdomain) 	2022年07月08日 	_x86_64_	(8 CPU)

09时01分16秒  CPU    %usr   %nice    %sys %iowait    %irq   %soft  %steal  %guest  %gnice   %idle
09时01分17秒  all    7.18    0.00    1.89   87.41    0.00    0.13    0.00    0.00    0.00    3.40
09时01分17秒    0    8.08    0.00    2.02   86.87    0.00    2.02    0.00    0.00    0.00    1.01
09时01分17秒    1    9.00    0.00    2.00   86.00    0.00    0.00    0.00    0.00    0.00    3.00
09时01分17秒    2    8.08    0.00    3.03   87.88    0.00    0.00    0.00    0.00    0.00    1.01
09时01分17秒    3    9.09    0.00    2.02   84.85    0.00    0.00    0.00    0.00    0.00    4.04
09时01分17秒    4    6.00    0.00    2.00   83.00    0.00    0.00    0.00    0.00    0.00    9.00
09时01分17秒    5    6.06    0.00    2.02   90.91    0.00    0.00    0.00    0.00    0.00    1.01
09时01分17秒    6    5.05    0.00    2.02   89.90    0.00    0.00    0.00    0.00    0.00    3.03
09时01分17秒    7    6.06    0.00    2.02   89.90    0.00    0.00    0.00    0.00    0.00    2.02

09时01分17秒  CPU    %usr   %nice    %sys %iowait    %irq   %soft  %steal  %guest  %gnice   %idle
09时01分18秒  all    6.17    0.00    1.76   86.78    0.00    0.25    0.00    0.00    0.00    5.04
09时01分18秒    0    6.00    0.00    3.00   83.00    0.00    1.00    0.00    0.00    0.00    7.00
09时01分18秒    1    8.08    0.00    2.02   86.87    0.00    0.00    0.00    0.00    0.00    3.03
09时01分18秒    2    7.14    0.00    1.02   88.78    0.00    0.00    0.00    0.00    0.00    3.06
09时01分18秒    3    7.00    0.00    2.00   90.00    0.00    0.00    0.00    0.00    0.00    1.00
09时01分18秒    4    5.00    0.00    1.00   85.00    0.00    0.00    0.00    0.00    0.00    9.00
09时01分18秒    5    4.95    0.00    1.98   86.14    0.00    0.00    0.00    0.00    0.00    6.93
09时01分18秒    6    5.05    0.00    1.01   86.87    0.00    0.00    0.00    0.00    0.00    7.07
09时01分18秒    7    6.00    0.00    1.00   86.00    0.00    0.00    0.00    0.00    0.00    7.00
^C

平均时间:  CPU    %usr   %nice    %sys %iowait    %irq   %soft  %steal  %guest  %gnice   %idle
平均时间:  all    6.68    0.00    1.83   87.09    0.00    0.19    0.00    0.00    0.00    4.22
平均时间:    0    7.04    0.00    2.51   84.92    0.00    1.51    0.00    0.00    0.00    4.02
平均时间:    1    8.54    0.00    2.01   86.43    0.00    0.00    0.00    0.00    0.00    3.02
平均时间:    2    7.61    0.00    2.03   88.32    0.00    0.00    0.00    0.00    0.00    2.03
平均时间:    3    8.04    0.00    2.01   87.44    0.00    0.00    0.00    0.00    0.00    2.51
平均时间:    4    5.50    0.00    1.50   84.00    0.00    0.00    0.00    0.00    0.00    9.00
平均时间:    5    5.50    0.00    2.00   88.50    0.00    0.00    0.00    0.00    0.00    4.00
平均时间:    6    5.05    0.00    1.52   88.38    0.00    0.00    0.00    0.00    0.00    5.05
平均时间:    7    6.03    0.00    1.51   87.94    0.00    0.00    0.00    0.00    0.00    4.52

# 也可以指定 CPU 观测
[root@localhost ~]# mpstat -P 6 1
Linux 3.10.0-957.el7.x86_64 (localhost.localdomain) 	2022年07月08日 	_x86_64_	(8 CPU)

09时02分18秒  CPU    %usr   %nice    %sys %iowait    %irq   %soft  %steal  %guest  %gnice   %idle
09时02分19秒    6    7.00    0.00    1.00   85.00    0.00    0.00    0.00    0.00    0.00    7.00
09时02分20秒    6    5.05    0.00    1.01   87.88    0.00    0.00    0.00    0.00    0.00    6.06
09时02分21秒    6    4.04    0.00    2.02   86.87    0.00    0.00    0.00    0.00    0.00    7.07
^C
平均时间:    6    5.37    0.00    1.34   86.58    0.00    0.00    0.00    0.00    0.00    6.71
```



## sar

sar 主要是查看系统历史数据，比如半夜有没有负载突然暴涨导致服务奔溃等

```bash
[root@localhost ~]# sar
Linux 3.10.0-957.el7.x86_64 (localhost.localdomain) 	2022年07月08日 	_x86_64_	(8 CPU)

00时00分01秒     CPU     %user     %nice   %system   %iowait    %steal     %idle
00时10分01秒     all      0.06      0.00      0.08      0.01      0.00     99.84
00时20分01秒     all      0.06      0.00      0.09      0.01      0.00     99.84
00时30分01秒     all      0.07      0.00      0.08      0.01      0.00     99.84
00时40分01秒     all      0.06      0.00      0.08      0.00      0.00     99.86
00时50分01秒     all      0.06      0.00      0.08      0.01      0.00     99.85
01时00分01秒     all      0.06      0.00      0.08      0.01      0.00     99.85
01时10分01秒     all      0.06      0.00      0.08      0.01      0.00     99.85
01时20分01秒     all      0.05      0.00      0.08      0.01      0.00     99.85
01时30分01秒     all      0.06      0.00      0.08      0.01      0.00     99.85
01时40分01秒     all      0.06      0.00      0.08      0.01      0.00     99.85
01时50分01秒     all      0.06      0.00      0.08      0.01      0.00     99.84
02时00分01秒     all      0.07      0.00      0.08      0.01      0.00     99.84
02时10分01秒     all      0.07      0.00      0.08      0.01      0.00     99.84
02时20分01秒     all      0.07      0.00      0.09      0.01      0.00     99.83
02时30分01秒     all      0.07      0.00      0.08      0.01      0.00     99.84
02时40分02秒     all      0.06      0.00      0.08      0.01      0.00     99.84
02时50分01秒     all      0.07      0.00      0.08      0.01      0.00     99.84
03时00分01秒     all      0.06      0.00      0.08      0.01      0.00     99.84
03时10分01秒     all      0.06      0.00      0.08      0.01      0.00     99.85
03时20分01秒     all      0.06      0.00      0.08      0.01      0.00     99.84
03时30分01秒     all      0.07      0.00      0.08      0.01      0.00     99.84
03时40分01秒     all      0.06      0.00      0.08      0.01      0.00     99.85
03时50分01秒     all      0.06      0.00      0.07      0.00      0.00     99.86
04时00分01秒     all      0.07      0.00      0.08      0.01      0.00     99.85
04时10分01秒     all      0.07      0.00      0.08      0.01      0.00     99.84
04时20分01秒     all      0.07      0.00      0.08      0.01      0.00     99.84
04时30分01秒     all      0.06      0.00      0.08      0.01      0.00     99.84
04时40分01秒     all      0.07      0.00      0.09      0.01      0.00     99.84
04时50分01秒     all      0.07      0.00      0.08      0.01      0.00     99.84
05时00分01秒     all      0.06      0.00      0.08      0.01      0.00     99.85
05时10分01秒     all      0.07      0.00      0.08      0.01      0.00     99.84
05时20分01秒     all      0.06      0.00      0.08      0.01      0.00     99.85
05时30分01秒     all      0.07      0.00      0.08      0.01      0.00     99.83
05时40分01秒     all      0.07      0.00      0.08      0.01      0.00     99.84
05时50分01秒     all      0.06      0.00      0.08      0.01      0.00     99.85

05时50分01秒     CPU     %user     %nice   %system   %iowait    %steal     %idle
06时00分01秒     all      0.07      0.00      0.08      0.01      0.00     99.84
06时10分01秒     all      0.06      0.00      0.08      0.01      0.00     99.85
06时20分01秒     all      0.07      0.00      0.08      0.01      0.00     99.84
06时30分01秒     all      0.07      0.00      0.09      0.01      0.00     99.84
06时40分01秒     all      0.07      0.00      0.08      0.01      0.00     99.84
06时50分01秒     all      0.06      0.00      0.08      0.01      0.00     99.85
07时00分01秒     all      0.06      0.00      0.08      0.00      0.00     99.85
07时10分01秒     all      0.06      0.00      0.08      0.01      0.00     99.86
07时20分01秒     all      0.06      0.00      0.08      0.01      0.00     99.85
07时30分01秒     all      0.06      0.00      0.08      0.01      0.00     99.85
07时40分01秒     all      0.07      0.00      0.08      0.01      0.00     99.84
07时50分01秒     all      0.07      0.00      0.08      0.01      0.00     99.84
08时00分01秒     all      0.06      0.00      0.08      0.01      0.00     99.84
08时10分01秒     all      0.07      0.00      0.08      0.01      0.00     99.84
08时20分02秒     all      0.07      0.00      0.09      0.01      0.00     99.83
08时30分01秒     all      0.07      0.00      0.09      0.01      0.00     99.83
08时40分01秒     all      0.05      0.00      0.06      0.01      0.00     99.87
08时50分01秒     all      5.42      0.00     10.39     16.69      0.00     67.49
09时00分01秒     all     11.54      0.00     18.09     58.75      0.00     11.62
平均时间:     all      0.37      0.00      0.60      1.39      0.00     97.64
```

我们可以看到 CPU 的历史负载信息，发现在 08 时 50 分 01 秒系统负载不再是一个低水平，而是开始上涨，这是因为我运行了一些测试脚本。Linux 版本为 CPU 分析提供了以下选项：

-   -P ALL：与 mpstat 的 -P ALL 选项相同，可以去分析到某个 CPU
-   -q：包括运行队列长度 runq-sz（类似 vmstat r 列）和平均负载



## ps

ps 用来查看进程状态，`ps aux` 操作风格起源于 BSD，`ps -ef` 风格起源于 SVR4，这两个命令都可以查看所有进程状态。更多的时候我们会搭配 grep 去查看进程是否存在以及它的状态。

```bash
[root@localhost ~]# ps -ef
UID        PID  PPID  C STIME TTY          TIME CMD
root         1     0  0 6月29 ?       00:00:28 /usr/lib/systemd/systemd --switched-root --system --deserialize 22
root         2     0  0 6月29 ?       00:00:00 [kthreadd]
root         3     2  0 6月29 ?       00:00:43 [ksoftirqd/0]
root         5     2  0 6月29 ?       00:00:00 [kworker/0:0H]
root         7     2  0 6月29 ?       00:00:00 [migration/0]
root         8     2  0 6月29 ?       00:00:00 [rcu_bh]
root         9     2  0 6月29 ?       00:17:53 [rcu_sched]
root        10     2  0 6月29 ?       00:00:00 [lru-add-drain]
root        11     2  0 6月29 ?       00:00:03 [watchdog/0]
...
```



## 基准测试

sysbench 系统基准测试套件有一个计算质数的简单 CPU 基准测试工具：

```bash
[root@localhost ~]# sysbench --num-threads=8 --test=cpu --cpu-max-prime=100000 run
WARNING: the --test option is deprecated. You can pass a script name or path on the command line without any options.
WARNING: --num-threads is deprecated, use --threads instead
sysbench 1.0.20 (using bundled LuaJIT 2.1.0-beta2)

Running the test with following options:
Number of threads: 8
Initializing random number generator from current time


Prime numbers limit: 100000

Initializing worker threads...

Threads started!

CPU speed:
    events per second:   314.63

General statistics:
    total time:                          10.0167s
    total number of events:              3152

Latency (ms):
         min:                                   23.95
         avg:                                   25.40
         max:                                   33.50
         95th percentile:                       25.28
         sum:                                80046.46

Threads fairness:
    events (avg/stddev):           394.0000/0.00
    execution time (avg/stddev):   10.0058/0.00

```

这个工具执行 8 个线程，最多计算 100000 个质数，运行时间结果为 10s。这个结果可以用来和其它系统配置进行比较。





---

# 内存

本文关注系统本身的分析，不会去解释虚拟内存和换页的相关知识，请大家自行学习。由于虚拟内存模型和按需换页的结果会导致虚拟页会处于以下状态之一：

-   A. 未分配
-   B. 已分配，未映射
-   C. 已分配，已映射到主存
-   D. 已分配，已映射到物理交换空间（磁盘）

从这几种状态出发，可以定义另外两个内存使用术语：

-   常驻集合大小（RSS）：已分配的主存页（C）大小
-   虚拟内存大小：所有已分配的区域（B + C + D）



## vmstat

之前介绍观测 CPU 的时候我们已经提起过 vmstat 了，实际上很多工具都是综合性的，可以观察到不同的指标，我们需要灵活运用这些工具去关注我们想要关注的点。

```bash
# 内存默认单位为 KB
[root@localhost ~]# vmstat 1
procs -----------memory---------- ---swap-- -----io---- -system-- ------cpu-----
 r  b   swpd   free   buff  cache   si   so    bi    bo   in   cs us sy id wa st
12  9  46592 13503480      0 1778872    0    0  1232  2377    3    5  0  0 98  2  0
 4  2  46592 12830604      0 2445548    0    0     4 708412 42608 111766 33 21 18 28  0
 1  5  46592 12215392      0 3057948    0    0     0 649112 39669 102396 28 19 25 29  0
 9  9  46592 11589140      0 3683580    0    0     0 665408 42076 107363 29 20 22 29  0
14 11  46592 10975716      0 4295228    0    0     0 651220 40859 107888 29 20 23 29  0
13 15  46592 10339104      0 4929260    0    0     0 672572 41056 104774 29 20 24 27  0
16 12  46592 9781844      0 5483656    0    0     0 590304 36201 93691 26 18 21 35  0

```

和内存相关的字段含义如下：

| 字段  | 含义               |
| ----- | ------------------ |
| swpd  | 交换空间           |
| free  | 可用内存           |
| buff  | 用于缓冲缓存的内存 |
| cache | 用于页缓存的内存   |
| si    | 换入的内存         |
| so    | 换出的内存         |

如果 si 和 so 列一直为非 0，表示系统正存在内存压力并执行交换到交换设备或文件。

swapon 可以显示是否配置了交换设备，以及他们的使用率。

```bash
[root@localhost ~]# swapon 
NAME      TYPE      SIZE  USED PRIO
/dev/dm-1 partition 7.8G 45.5M   -2
```

你可以试着用 -S 选项将单位修改为 MB 来让数字对齐（m 表示 1000000，M 表示 1048576）

```bash
[root@localhost ~]# vmstat -Sm 1
procs -----------memory---------- ---swap-- -----io---- -system-- ------cpu-----
 r  b   swpd   free   buff  cache   si   so    bi    bo   in   cs us sy id wa st
 0  0     47  14848      0    856    0    0  1234  2378    4    5  0  0 98  2  0
 0  0     47  14848      0    856    0    0     0     0  307  226  0  0 100  0  0
 0  0     47  14848      0    856    0    0     0     0  352  250  0  0 100  0  0
 0  0     47  14848      0    856    0    0     0     0  688  406  0  0 100  0  0
 0  0     47  14848      0    856    0    0     0     0  669  439  0  0 100  0  0
^C
[root@localhost ~]# vmstat -SM 1
procs -----------memory---------- ---swap-- -----io---- -system-- ------cpu-----
 r  b   swpd   free   buff  cache   si   so    bi    bo   in   cs us sy id wa st
 0  0     45  14183      0    977    0    0  1234  2378    4    5  0  0 98  2  0
 0  0     45  14202      0    969    0    0     0  4268 1678 6467  0  0 99  0  0
 0  0     45  14220      0    963    0    0     0  3384 1573 6453  0  0 100  0  0
^C

```



## sar

查看历史内存使用信息，有如下选项：

-   -B：换页统计信息
-   -H：巨型页统计信息
-   -r：内存使用率
-   -S：交换空间统计信息
-   -W：交换统计信息



## top

top 命令可以使用 -o 选项来指定排序列 `top -o %MEM`。RES 即 RSS，表示常驻内存大小，VIRT 表示虚拟内存大小。

```bash
top - 15:31:13 up 9 days,  6:23,  3 users,  load average: 0.00, 0.02, 0.22
Tasks: 458 total,   1 running, 457 sleeping,   0 stopped,   0 zombie
%Cpu(s):  0.0 us,  0.0 sy,  0.0 ni,100.0 id,  0.0 wa,  0.0 hi,  0.0 si,  0.0 st
KiB Mem：16201188 total, 15194200 free,   512732 used,   494256 buff/cache
KiB Swap:  8191996 total,  8145404 free,    46592 used. 15276580 avail Mem 

  PID USER      PR  NI    VIRT    RES    SHR S  %CPU %MEM     TIME+ COMMAND                                                                                                
 2245 root      20   0 1835140 323936   7640 S   0.0  2.0 384:19.53 minio                                                                                                  
 2944 root      20   0   47268  11712  11572 S   0.0  0.1   0:02.62 systemd-journal                                                                                        
 5976 root      20   0  224600   6800   6444 S   0.0  0.0   0:28.37 rsyslogd                                                                                               
16592 postfix   20   0   89656   4068   3068 S   0.0  0.0   0:00.01 pickup                                                                                                 
    1 root      20   0  191632   2868   1256 S   0.0  0.0   0:33.58 systemd                                                                                                
18569 root      20   0  162396   2656   1604 R   0.3  0.0   0:00.47 top                                                                                                    
 5973 root      20   0  573820   1560    904 S   0.0  0.0   1:05.76 tuned                                                                                                  
27101 root      20   0  115652   1472    952 S   0.0  0.0   0:00.27 bash                                                                                                   
27099 root      20   0  161364   1400    496 S   0.0  0.0   0:01.29 sshd                                                                                                   
 5652 root      20   0  627780   1336    644 S   0.0  0.0   3:36.94 Network
```



## pmap

pmap 可以列出一个进程的内存映射，这让我们可以更详细地检查进程的内存使用情况。

```bash
[root@localhost ~]# pmap -x `pidof minio`
2245:   /usr/local/bin/minio server --console-address :9001 /mnt/{0...35}
Address           Kbytes     RSS   Dirty Mode  Mapping
0000000000400000   28772    2680       0 r-x-- minio
0000000002019000   62960    5216       0 r---- minio
0000000005d95000    1060     528     384 rw--- minio
0000000005e9e000     344     180     180 rw---   [ anon ]
000000c000000000  987136  253648  253648 rw---   [ anon ]
000000c03c400000   61440       0       0 -----   [ anon ]
00007fd434503000   25348   25328   25328 rw---   [ anon ]
00007fd435dc5000    6980    6956    6956 rw---   [ anon ]
00007fd43649c000    3844    3824    3824 rw---   [ anon ]
00007fd436861000    1408    1364    1364 rw---   [ anon ]
00007fd4369c9000   56284   25532   25532 rw---   [ anon ]
00007fd43a0c0000  263680       0       0 -----   [ anon ]
00007fd44a240000       4       4       4 rw---   [ anon ]
00007fd44a241000  293564       0       0 -----   [ anon ]
00007fd45c0f0000       4       4       4 rw---   [ anon ]
00007fd45c0f1000   36692       0       0 -----   [ anon ]
00007fd45e4c6000       4       4       4 rw---   [ anon ]
00007fd45e4c7000    4580       0       0 -----   [ anon ]
00007fd45e940000       4       4       4 rw---   [ anon ]
00007fd45e941000     508       0       0 -----   [ anon ]
00007fd45e9c0000     384      64      64 rw---   [ anon ]
00007ffd3e47c000     132      12      12 rw---   [ stack ]
00007ffd3e4dd000       8       4       0 r-x--   [ anon ]
ffffffffff600000       4       0       0 r-x--   [ anon ]
---------------- ------- ------- ------- 
total kB         1835144  325352  317308
```

-X 可以显示更多的细节，-XX用于显示内核提供的一切细节。

```bash
[root@localhost ~]# pmap -X `pidof minio`
2245:   /usr/local/bin/minio server --console-address :9001 /mnt/{0...35}
         Address Perm   Offset Device  Inode    Size    Rss    Pss Referenced Anonymous Swap Locked Mapping
        00400000 r-xp 00000000  fd:00 473089   28772   2680   2680       2680         0    0      0 minio
        02019000 r--p 01c19000  fd:00 473089   62960   5216   5216       5216         0    0      0 minio
        05d95000 rw-p 05995000  fd:00 473089    1060    528    528        472       384   36      0 minio
        05e9e000 rw-p 00000000  00:00      0     344    180    180        164       180    4      0 
      c000000000 rw-p 00000000  00:00      0  987136 253648 253648     249948    253648 6448      0 
      c03c400000 ---p 00000000  00:00      0   61440      0      0          0         0    0      0 
    7fd434503000 rw-p 00000000  00:00      0   25348  25328  25328      25328     25328    0      0 
    7fd435dc5000 rw-p 00000000  00:00      0    6980   6956   6956       6956      6956    0      0 
    7fd43649c000 rw-p 00000000  00:00      0    3844   3824   3824       3824      3824    0      0 
    7fd436861000 rw-p 00000000  00:00      0    1408   1364   1364       1364      1364    0      0 
    7fd4369c9000 rw-p 00000000  00:00      0   56284  25532  25532      25188     25532   20      0 
    7fd43a0c0000 ---p 00000000  00:00      0  263680      0      0          0         0    0      0 
    7fd44a240000 rw-p 00000000  00:00      0       4      4      4          4         4    0      0 
    7fd44a241000 ---p 00000000  00:00      0  293564      0      0          0         0    0      0 
    7fd45c0f0000 rw-p 00000000  00:00      0       4      4      4          4         4    0      0 
    7fd45c0f1000 ---p 00000000  00:00      0   36692      0      0          0         0    0      0 
    7fd45e4c6000 rw-p 00000000  00:00      0       4      4      4          4         4    0      0 
    7fd45e4c7000 ---p 00000000  00:00      0    4580      0      0          0         0    0      0 
    7fd45e940000 rw-p 00000000  00:00      0       4      4      4          4         4    0      0 
    7fd45e941000 ---p 00000000  00:00      0     508      0      0          0         0    0      0 
    7fd45e9c0000 rw-p 00000000  00:00      0     384     64     64         60        64    0      0 
    7ffd3e47c000 rw-p 00000000  00:00      0     132     12     12          4        12    8      0 [stack]
    7ffd3e4dd000 r-xp 00000000  00:00      0       8      4      0          4         0    0      0 [vdso]
ffffffffff600000 r-xp 00000000  00:00      0       4      0      0          0         0    0      0 [vsyscall]
                                             ======= ====== ====== ========== ========= ==== ====== 
                                             1835144 325352 325348     321224    317308 6516      0 KB 
```



## perf

前面介绍过 perf 是一个功能很强大的剖析器，是一个你对系统越了解，它能发挥的功能就越强的利器。比如我们来分析缺页，缺页是随着进程常驻集合大小（RSS）的增长而发生的，因此分析它可以解释为什么主存在增长。比如这里就可以看到，minio 在操作数据时，不可避免地要分配内存去处理数据，当然 go 语言不需要你主动去分配内存，它是伴随着比如切片操作，然后去调取 malloc 分配内存，然后使用从而引起了缺页。

```bash
[root@localhost ~]# perf record -e page-faults -p `pidof minio` -g -- sleep 5
[ perf record: Woken up 1 times to write data ]
[ perf record: Captured and wrote 0.067 MB perf.data (152 samples) ]
[root@localhost ~]# perf script
minio  2305 801156.464930:          1 page-faults: 
                  46c57c runtime.memclrNoHeapPointers+0x11c (/usr/local/bin/minio)
                  40d2c9 runtime.mallocgc+0x789 (/usr/local/bin/minio)
                  44ea12 runtime.makeslice+0x52 (/usr/local/bin/minio)
                 1ef6d79 github.com/minio/minio/cmd.(*xlMetaV2).AppendTo+0xb9 (/usr/local/bin/minio)
                 1f18734 github.com/minio/minio/cmd.(*xlStorage).RenameData+0x1bb4 (/usr/local/bin/minio)
                 1ee11b8 github.com/minio/minio/cmd.(*xlStorageDiskIDCheck).RenameData+0x318 (/usr/local/bin/minio)
                 1cf57c9 github.com/minio/minio/cmd.renameData.func1+0x249 (/usr/local/bin/minio)
                 14db08b github.com/minio/minio/internal/sync/errgroup.(*Group).Go.func1+0x1cb (/usr/local/bin/minio)
                  46b8c1 runtime.goexit.abi0+0x1 (/usr/local/bin/minio)

minio  2305 801156.464942:          1 page-faults: 
                  46c57c runtime.memclrNoHeapPointers+0x11c (/usr/local/bin/minio)
                  40d2c9 runtime.mallocgc+0x789 (/usr/local/bin/minio)
                  44ea12 runtime.makeslice+0x52 (/usr/local/bin/minio)
                 1ef6d79 github.com/minio/minio/cmd.(*xlMetaV2).AppendTo+0xb9 (/usr/local/bin/minio)
                 1f18734 github.com/minio/minio/cmd.(*xlStorage).RenameData+0x1bb4 (/usr/local/bin/minio)
                 1ee11b8 github.com/minio/minio/cmd.(*xlStorageDiskIDCheck).RenameData+0x318 (/usr/local/bin/minio)
                 1cf57c9 github.com/minio/minio/cmd.renameData.func1+0x249 (/usr/local/bin/minio)
                 14db08b github.com/minio/minio/internal/sync/errgroup.(*Group).Go.func1+0x1cb (/usr/local/bin/minio)
                  46b8c1 runtime.goexit.abi0+0x1 (/usr/local/bin/minio)
...
minio  2323 801156.465698:          1 page-faults: 
                  46c57c runtime.memclrNoHeapPointers+0x11c (/usr/local/bin/minio)
                  40d2c9 runtime.mallocgc+0x789 (/usr/local/bin/minio)
                  44ea12 runtime.makeslice+0x52 (/usr/local/bin/minio)
                 1f0673f github.com/minio/minio/cmd.(*xlMetaInlineData).serialize+0x9f (/usr/local/bin/minio)
                 1f070dd github.com/minio/minio/cmd.(*xlMetaInlineData).replace+0x79d (/usr/local/bin/minio)
                 1efaed5 github.com/minio/minio/cmd.(*xlMetaV2).AddVersion+0xa15 (/usr/local/bin/minio)
                 1f18685 github.com/minio/minio/cmd.(*xlStorage).RenameData+0x1b05 (/usr/local/bin/minio)
                 1ee11b8 github.com/minio/minio/cmd.(*xlStorageDiskIDCheck).RenameData+0x318 (/usr/local/bin/minio)
                 1cf57c9 github.com/minio/minio/cmd.renameData.func1+0x249 (/usr/local/bin/minio)
                 14db08b github.com/minio/minio/internal/sync/errgroup.(*Group).Go.func1+0x1cb (/usr/local/bin/minio)
                  46b8c1 runtime.goexit.abi0+0x1 (/usr/local/bin/minio)
...

```







---

# 文件系统

## 查看使用的文件描述符

首先使用 ps 或 pidof 查看进程 PID。

```bash
[root@localhost ~]# pidof minio
25083
[root@localhost ~]# ps -ef | grep minio
root     25083     1  0 7月05 ?       00:07:49 /usr/local/bin/minio server --console-address :9001 /mnt/{0...35}
root     27075 26599  0 19:30 pts/0    00:00:00 grep --color=auto minio
```

然后列出 /proc/pid/fd 下使用的描述符，通常我们会关心进程是否有未关闭的 TCP 连接等，这种情况 fd 目录下会有很多未关闭的连接，我们可以通过 wc 来统计，如果数目比较大，很大概率就是有未释放的文件描述符。

```bash
[root@localhost ~]# ll /proc/25083/fd
总用量 0
lr-x------ 1 root root 64 7月   5 19:05 0 -> /dev/null
lrwx------ 1 root root 64 7月   5 19:05 1 -> socket:[179370]
lrwx------ 1 root root 64 7月   5 19:05 10 -> socket:[175528]
lrwx------ 1 root root 64 7月   5 19:05 19 -> socket:[177611]
lrwx------ 1 root root 64 7月   5 19:05 2 -> socket:[179370]
lrwx------ 1 root root 64 7月   5 19:05 3 -> anon_inode:[eventpoll]
lr-x------ 1 root root 64 7月   5 19:05 4 -> pipe:[173809]
l-wx------ 1 root root 64 7月   5 19:05 5 -> pipe:[173809]
lrwx------ 1 root root 64 7月   5 19:05 6 -> socket:[173812]
lrwx------ 1 root root 64 7月   5 19:05 7 -> socket:[173813]
l-wx------ 1 root root 64 7月   5 19:05 8 -> /mnt/0/.minio.sys/ilm/deletion-journal.bin
[root@localhost ~]# ll /proc/25083/fd | wc -l
12
```



## 缓存

Unix 原本只用缓冲区高速缓存（buffer）来提高块设备访问性能，如今，linux 有多种缓存，比如 dentry 缓存，inode 缓存，页缓存，缓冲区等。

Linux 原本和 Unix 一样使用缓冲区高速缓存，从 linux 2.4 开始，缓冲区高速缓存就被储存在了页缓存里，防止双重缓存和同步的开销。页缓存可以简单理解为文件系统层面数据的缓存，目录项缓存（Dcache）记录了 dentry 到 inode 的映射关系，提升的是路径名查找的性能（比如 open）。inode 缓存记录的是 inode 的信息，inode 记录的是文件的信息，比如目录下有哪些文件，文件具体保存在哪个地方等。

free 命令展示内存和交换区的统计信息，单位是 MB（-m）

```bash
[root@localhost ~]# free -wm
              total        used        free      shared     buffers       cache   available
Mem:          15821         203       15474           8           2         142       15361
Swap:          7999           0        7999
```

slabtop 打印有关内存 slab（linux 内核内存管理器）缓存的信息，slab 可能包含以下内容：

-   dentry：目录项缓存
-   inode_cache：inode 缓存
-   xfs_inode：XFS 的 inode 缓存
-   buffer_head：缓冲区高速缓存
-   etc...

```bash
 Active / Total Objects (% used)   ：1903270 / 2219372 (85.8%)
 Active / Total Slabs (% used)     ：59381 / 59381 (100.0%)
 Active / Total Caches (% used)    ：66 / 96 (68.8%)
 Active / Total Size (% used)      ：382311.58K / 462753.92K (82.6%)
 Minimum / Average / Maximum Object：0.01K / 0.21K / 12.75K

  OBJS ACTIVE  USE OBJ SIZE  SLABS OBJ/SLAB CACHE SIZE NAME                   
1370616 1158626  84%    0.10K  35144       39    140576K buffer_head
147420 147420 100%    0.19K   3510	 42     28080K dentry
146720  48254  32%    0.57K   5240	 28     83840K radix_tree_node
114614 114472  99%    0.94K   3371	 34    107872K xfs_inode
112896 112896 100%    0.16K   4704	 24     18816K xfs_ili
 90176  90176 100%    0.06K   1409	 64      5636K kmalloc-64
 34204  34204 100%    0.12K   1006	 34	 4024K kernfs_node_cache
 30496  29878  97%    0.50K    953	 32     15248K kmalloc-512
 30492  30492 100%    0.09K    726	 42	 2904K kmalloc-96
 24800  24750  99%    1.00K    775	 32     24800K kmalloc-1024
 19890  19890 100%    0.04K    195	102       780K selinux_inode_security
 13689  12376  90%    0.58K    507	 27      8112K inode_cache
  8704   8704 100%    0.01K     17	512	   68K kmalloc-8
  8448   8448 100%    0.02K     33	256	  132K kmalloc-16
  6846   6739  98%    0.38K    163	 42	 2608K mnt_cache
  6608   6608 100%    0.07K    118	 56	  472K avc_node
  6528   6528 100%    0.03K     51	128	  204K kmalloc-32
  5670   5670 100%    0.19K    135	 42	 1080K kmalloc-192
  4884   4884 100%    0.21K    132	 37	 1056K vm_area_struct
  4539   4539 100%    0.08K     89	 51	  356K anon_vma
  3968   2981  75%    0.25K    124	 32	  992K kmalloc-256
  3424   2912  85%    0.12K    107	 32	  428K kmalloc-128
  3315   3315 100%    0.05K     39	 85	  156K shared_policy_node
  2256   2023  89%    0.64K     94	 24	 1504K proc_inode_cache
  2211    982  44%    0.24K     67	 33	  536K posix_timers_cache
  2136   2136 100%    0.66K	89	 24      1424K shmem_inode_cache
  1190   1190 100%    0.02K      7	170	   28K fsnotify_mark_connector
   924    924 100%    1.12K     33	 28      1056K signal_cache
   900    900 100%    0.11K     25	 36	  100K task_delay_info

```





## 历史信息

sar 命令查看历史信息：-v 参数提供以下信息：

-   dentunusd：目录项缓存未用计数即可用项（这个含义有点迷惑，大家可以先忽略这个字段）
-   file-nr：使用中的文件句柄个数
-   inode-nr：使用中的 inode 个数
-   pty-nr：使用的伪终端数目

```bash
[root@localhost ~]# sar -v
Linux 3.10.0-957.el7.x86_64 (localhost.localdomain) 	2022年07月11日 	_x86_64_	(8 CPU)

00时00分01秒 dentunusd   file-nr  inode-nr    pty-nr
00时10分01秒     11850       928     18917         0
00时20分01秒     11856       928     18917         0
...
03时30分01秒     11996       928     18943         0
03时40分01秒     14952       928     21841         0
03时50分01秒     14958       960     21841         0
04时00分01秒     14964       896     21841         0
04时10分02秒     14976       928     21841         0
04时20分01秒     14982       960     21841         0
04时30分01秒     14988       960     21841         0
04时40分01秒     14994       960     21841         0
04时50分01秒     15000       992     21841         0
05时00分01秒     15006       992     21841         0
05时10分01秒     15018       992     21841         0
05时20分01秒     15024       960     21841         0
05时30分01秒     15030       960     21841         0
05时40分01秒     15036       960     21841         0
05时50分01秒     15042       992     21841         0

05时50分01秒 dentunusd   file-nr  inode-nr    pty-nr
06时00分01秒     15048       960     21841         0
平均时间:     13203       931     20138         0

06时08分38秒       LINUX RESTART

06时10分01秒 dentunusd   file-nr  inode-nr    pty-nr
06时20分01秒     11236       896     18875         0
06时30分01秒     11245       928     18884         0
...
11时10分01秒     12139       992     19304         1
11时20分01秒    104575      1088     92982         2
11时30分01秒     28788      1120     16431         2
11时40分01秒     29154      1088     16431         2
11时50分01秒     29516      1088     16425         2
12时00分01秒     29882      1088     16425         2
12时10分01秒     30261      1120     16425         2
12时20分01秒     30627      1088     16425         2
12时30分01秒     30993      1120     16425         2
12时40分01秒     31359      1088     16425         2
12时50分01秒     31744      1088     16423         2
13时00分01秒     32110      1120     16423         2
13时10分02秒    111232      1088     96977         2
平均时间:     20515       992     21943         1

```





---

# 磁盘

## iostat

这是最常用的磁盘状态监控工具

```bash
# -m：以 MB 为单位，默认是 512B
# -t：输出时间戳
# -x：扩展信息
[root@localhost ~]# iostat -mtx 1 /dev/sde
2022年07月12日 09时30分49秒
avg-cpu:  %user   %nice %system %iowait  %steal   %idle
          26.40    0.00   18.62   30.61    0.00   24.36

Device:         rrqm/s   wrqm/s     r/s     w/s    rMB/s    wMB/s avgrq-sz avgqu-sz   await r_await w_await  svctm  %util
sde               0.00     0.00    0.00  279.00     0.00    16.28   119.48     0.17    0.62    0.00    0.62   0.56  15.60

2022年07月12日 09时30分50秒
avg-cpu:  %user   %nice %system %iowait  %steal   %idle
          28.93    0.00   18.91   28.43    0.00   23.73

Device:         rrqm/s   wrqm/s     r/s     w/s    rMB/s    wMB/s avgrq-sz avgqu-sz   await r_await w_await  svctm  %util
sde               0.00     0.00    0.00  279.00     0.00    16.12   118.34     0.18    0.65    0.00    0.65   0.61  16.90
```

字段含义如下：

-   rrqm/s：每秒合并读操作的次数
-   wrqm/s：每秒合并写操作的次数
-   r/s：每秒读操作的次数，即 iops
-   w/s：每秒写操作的次数
-   rMB/s：每秒读取的 MB 字节数，即带宽
-   wMB/s：每秒写入的 MB 字节数
-   avgrq-sz：每个 IO 的平均扇区数，即所有请求的平均大小，以扇区（512字节）为单位
-   avgqu-sz：平均为完成的 IO 请求数量，即平均意义上的请求队列长度
-   await：平均每个 IO 所需要的时间，包括在队列等待的时间，也包括磁盘控制器处理本次请求的有效时间。
    -   r_wait：每个读操作平均所需要的时间，不仅包括硬盘设备读操作的时间，也包括在内核队列中的时间。
    -   w_wait：每个写操平均所需要的时间，不仅包括硬盘设备写操作的时间，也包括在队列中等待的时间。
-   svctm：表面看是每个 IO 请求的服务时间，不包括等待时间，但是实际上，这个指标已经废弃。实际上，iostat 工具没有任何一输出项表示的是硬盘设备平均每次 IO 的时间。
-   %util：工作时间或者繁忙时间占总时间的百分比



### avgqu-sz 和繁忙程度

首先我们用超市购物来比对 iostat 的输出。我们在超市结账的时候，一般会有很多队可以排，队列的长度，在一定程度上反应了该收银柜台的繁忙程度。那么这个变量是 avgqu-sz 这个输出反应的，该值越大，表示排队等待处理的 io 越多。

我们用 4K 的随机 IO，使用 fio 来测试，一个 iodepth=1，一个 iodepth=16

```bash
# sde iodepth=1
$ fio --name=randwrite --rw=randwrite --bs=4k --size=10G --runtime=600 --ioengine=libaio --iodepth=1 --numjobs=1 --filename=/dev/sde --direct=1 --group_reporting
# sdf iodepth=16
$ fio --name=randwrite --rw=randwrite --bs=4k --size=10G --runtime=600 --ioengine=libaio --iodepth=16 --numjobs=1 --filename=/dev/sdf --direct=1 --group_reporting
```

iostat 观察结果：

```bash
[root@localhost ~]# iostat -mtx 1 /dev/sd[e-f]
2022年07月12日 09时37分57秒
avg-cpu:  %user   %nice %system %iowait  %steal   %idle
           0.13    0.00    0.25    0.00    0.00   99.62

Device:         rrqm/s   wrqm/s     r/s     w/s    rMB/s    wMB/s avgrq-sz avgqu-sz   await r_await w_await  svctm  %util
sdf               0.00     0.00    0.00  549.00     0.00     2.14     8.00    15.99   29.19    0.00   29.19   1.82 100.00
sde               0.00     0.00    0.00  503.00     0.00     1.96     8.00     0.97    1.94    0.00    1.94   1.94  97.50

2022年07月12日 09时37分58秒
avg-cpu:  %user   %nice %system %iowait  %steal   %idle
           0.13    0.00    0.38    0.00    0.00   99.50

Device:         rrqm/s   wrqm/s     r/s     w/s    rMB/s    wMB/s avgrq-sz avgqu-sz   await r_await w_await  svctm  %util
sdf               0.00     0.00    0.00  598.00     0.00     2.34     8.00    15.98   26.71    0.00   26.71   1.67 100.00
sde               0.00     0.00    0.00  451.00     0.00     1.76     8.00     0.98    2.17    0.00    2.17   2.17  97.70

2022年07月12日 09时37分59秒
avg-cpu:  %user   %nice %system %iowait  %steal   %idle
           0.13    0.00    0.25    0.00    0.00   99.62

Device:         rrqm/s   wrqm/s     r/s     w/s    rMB/s    wMB/s avgrq-sz avgqu-sz   await r_await w_await  svctm  %util
sdf               0.00     0.00    0.00  567.00     0.00     2.21     8.00    15.99   28.22    0.00   28.22   1.76 100.00
sde               0.00     0.00    0.00  493.00     0.00     1.93     8.00     0.98    1.99    0.00    1.99   1.99  98.30


```

我们可以观察到因为 avgqu-sz 大小不一样，所以一个 IO 等待时间（await）就不一样。就好像你在超时排队，有一队没有人，而另一队队伍长度达到16 ，那么很明显，队伍长队为16的更繁忙一些。



### avgrq-sz

avgrq-sz 这个值反应了用户的 IO-Pattern。我们经常关心，用户过来的 IO 是大 IO 还是小 IO，那么 avgrq-sz 反应了这个要素。它的含义是说，平均下来，这段时间内，所有请求的平均大小，单位是扇区（512字节）。

上面测试 avgrq-sz 总是 8，即 8 个扇区 = 8 * 512（Byte） = 4KB，这是因为我们用 fio 打 IO 的时候，用的 bs=4k。我们可以修改为 128k 测试一下：

```bash
# fio --name=randwrite --rw=randwrite --bs=128k --size=10G --runtime=600 --ioengine=libaio --iodepth=1 --numjobs=1 --filename=/dev/sde --direct=1 --group_reporting
[root@localhost ~]# iostat -mtx 1 /dev/sde
2022年07月12日 10时00分47秒
avg-cpu:  %user   %nice %system %iowait  %steal   %idle
           0.00    0.00    0.13    0.00    0.00   99.87

Device:         rrqm/s   wrqm/s     r/s     w/s    rMB/s    wMB/s avgrq-sz avgqu-sz   await r_await w_await  svctm  %util
sde               0.00     0.00    0.00  398.00     0.00    49.75   256.00     0.99    2.47    0.00    2.47   2.48  98.60

2022年07月12日 10时00分48秒
avg-cpu:  %user   %nice %system %iowait  %steal   %idle
           0.00    0.00    0.25    0.00    0.00   99.75

Device:         rrqm/s   wrqm/s     r/s     w/s    rMB/s    wMB/s avgrq-sz avgqu-sz   await r_await w_await  svctm  %util
sde               0.00     0.00    0.00  392.00     0.00    49.00   256.00     0.98    2.50    0.00    2.50   2.51  98.30
```

avgrq-sz 这列的值变成了 256，即 256 个扇区 = 256 * 512 Byte = 128KB。

当然，这个值也不是为所欲为的，它受内核参数的控制：

```bash
[root@localhost ~]# cat /sys/block/sde/queue/max_sectors_kb 
320
```

我们分别使用 320 KB 和 321 KB 测试：

```bash
# fio --name=randwrite --rw=randwrite --bs=320k --size=10G --runtime=600 --ioengine=libaio --iodepth=1 --numjobs=1 --filename=/dev/sde --direct=1 --group_reporting
[root@localhost ~]# iostat -mtx 1 /dev/sde
Linux 3.10.0-957.el7.x86_64 (localhost.localdomain) 	2022年07月12日 	_x86_64_	(8 CPU)

2022年07月12日 10时04分47秒
avg-cpu:  %user   %nice %system %iowait  %steal   %idle
           0.12    0.00    0.37    0.00    0.00   99.50

Device:         rrqm/s   wrqm/s     r/s     w/s    rMB/s    wMB/s avgrq-sz avgqu-sz   await r_await w_await  svctm  %util
sde               0.00     0.00    0.00  252.00     0.00    78.75   640.00     0.99    3.94    0.00    3.94   3.92  98.70

2022年07月12日 10时04分48秒
avg-cpu:  %user   %nice %system %iowait  %steal   %idle
           0.00    0.00    0.00    0.00    0.00  100.00

Device:         rrqm/s   wrqm/s     r/s     w/s    rMB/s    wMB/s avgrq-sz avgqu-sz   await r_await w_await  svctm  %util
sde               0.00     0.00    0.00  263.00     0.00    82.19   640.00     0.99    3.78    0.00    3.78   3.78  99.30

# fio --name=randwrite --rw=randwrite --bs=321k --size=10G --runtime=600 --ioengine=libaio --iodepth=1 --numjobs=1 --filename=/dev/sde --direct=1 --group_reporting
[root@localhost ~]# iostat -mtx 1 /dev/sde
Linux 3.10.0-957.el7.x86_64 (localhost.localdomain) 	2022年07月12日 	_x86_64_	(8 CPU)

2022年07月12日 10时05分39秒
avg-cpu:  %user   %nice %system %iowait  %steal   %idle
           0.00    0.00    0.13    0.00    0.00   99.87

Device:         rrqm/s   wrqm/s     r/s     w/s    rMB/s    wMB/s avgrq-sz avgqu-sz   await r_await w_await  svctm  %util
sde               0.00     0.00    0.00  288.00     0.00    45.14   321.00     1.98    6.80    0.00    6.80   3.43  98.80

2022年07月12日 10时05分40秒
avg-cpu:  %user   %nice %system %iowait  %steal   %idle
           0.13    0.00    0.13    0.00    0.00   99.75

Device:         rrqm/s   wrqm/s     r/s     w/s    rMB/s    wMB/s avgrq-sz avgqu-sz   await r_await w_await  svctm  %util
sde               0.00     0.00    0.00  324.00     0.00    50.78   321.00     1.98    6.20    0.00    6.20   3.06  99.20

```

我们可以看到当负载为 320KB 时，avgrq-sz = 640，这是匹配的，当负载为 321KB 时，由于超出了限制，所以负载被切割了，所以 avgrq-sz 反而变小了，因为 IO 变多了，所以可以观察到 avgqu-sz 和 await 都变成了两倍，这也很容易理解，就像超市里你买的东西一个购物篮放不下，于是你放到了两个购物篮里，然后让你朋友拿一个购物篮也开始排队，队伍自然就长了。



### rrqm/s 和 wrqm/s

块设备有相应的调度算法。如果两个 IO 发生在相邻的数据块时，他们可以合并成 1 个 IO。

这个简单的可以理解为快递员要给一个 18 层的公司所有员工送快递，每一层都有一些包裹，对于快递员来说，最好的办法是同一楼层相近的位置的包裹一起投递，否则如果不采用这种算法，采用最原始的来一个送一个（即 noop 算法），那么这个快递员，可能先送了一个包括到 18 层，又不得不跑到 2 层送另一个包裹，然后有不得不跑到 16 层送第三个包裹，然后跑到 1 层送第三个包裹，那么快递员的轨迹是杂乱无章的，也是非常低效的。

```bash
[root@localhost ~]# cat /sys/block/sde/queue/scheduler 
noop [deadline] cfq 
```



### svctm 和 %util

svctm 本意是单个IO被块设备处理的有效时间，但是目前这个字段并非如此，我们从 man 手册里也能看到这个字段以后将会被废弃。如果你感兴趣可以去了解 blktrace，这个工具能够帮你去分析 IO 服务的时间。

```bash
svctm
The average service time (in milliseconds) for I/O requests that were issued to the device. Warning! Do not trust this field any more.
This field will be removed in a future sysstat version.
```

同时不能理解成 %util 到了 100% ，磁盘工作就饱和了，不能继续提升了，因为可能 IO 队列没塞满，可能是虚拟块设备等等。%util 并不关心等待在队里里面 IO 的个数，它只关心队列中有没有 IO。

和超时排队结账这个类比最本质的区别在于，现代硬盘都有并行处理多个 IO 的能力，但是收银员没有。收银员无法做到同时处理 10 个顾客的结账任务而消耗的总时间与处理一个顾客结账任务相差无几。但是磁盘可以。所以，即使 %util 到了 100%，也并不意味着设备饱和了。

最简单的例子是，某硬盘处理单个 IO 请求需要 0.1 秒，有能力同时处理 10 个。但是当 10 个请求依次提交的时候，需要 1 秒钟才能完成请求，在 1 秒的采样周期里，%util 达到了 100%。但是如果 10 个请一次性提交的话， 硬盘可以在 0.1 秒内全部完成，这时候，%util 只有 10%。

因此，在上面的例子中，一秒中 10 个 IO，即 IOPS=10 的时候，%util 就达到了 100%，这并不能表明，该盘的 IOPS 就只能到 10，事实上，纵使 %util 到了 100%，硬盘可能仍然有很大的余力处理更多的请求，即并未达到饱和的状态。

那么有没有一个指标用来衡量硬盘设备的饱和程度呢。很遗憾，iostat 没有一个指标可以衡量磁盘设备的饱和度。



## sar

`sar -d` 查看磁盘历史信息，字段含义和 iostat 类似。

```bash
[root@localhost ~]# sar -d
Linux 3.10.0-957.el7.x86_64 (localhost.localdomain) 	2022年07月12日 	_x86_64_	(8 CPU)

00时00分01秒       DEV       tps  rd_sec/s  wr_sec/s  avgrq-sz  avgqu-sz     await     svctm     %util
00时10分01秒   dev8-16      0.21      0.00     80.74    390.71      0.00      9.79      2.48      0.05
00时10分01秒    dev8-0      0.21      0.00     80.74    390.71      0.00     13.83      3.29      0.07
00时10分01秒   dev8-48      0.21      0.00     80.74    390.71      0.00      9.34      2.52      0.05
00时10分01秒   dev8-32      0.21      0.00     80.74    390.71      0.00      9.40      2.46      0.05
00时10分01秒   dev8-80      0.21      0.00     80.74    390.71      0.00     10.54      2.58      0.05
00时10分01秒   dev8-96      0.21      0.03     80.74    384.63      0.00      8.22      2.47      0.05
00时10分01秒  dev8-128      0.21      0.03     80.75    381.67      0.00     10.88      2.61      0.06
00时10分01秒  dev8-112      0.21      0.03     80.74    384.63      0.00     10.84      2.79      0.06
00时10分01秒  dev8-144      0.21      0.03     80.75    381.67      0.00     11.29      2.72      0.06
00时10分01秒  dev8-160      0.21      0.03     80.75    381.67      0.00     10.16      2.61      0.06
00时10分01秒  dev8-176      0.21      0.03     80.75    381.67      0.00     10.17      2.70      0.06
00时10分01秒   dev8-64      0.21      0.00     80.75    387.65      0.00     10.51      2.74      0.06
00时10分01秒  dev8-192      0.21      0.03     80.75    381.67      0.00     10.74      2.86      0.06
00时10分01秒  dev8-208      0.21      0.03     80.74    384.63      0.00      9.17      2.57      0.05
00时10分01秒  dev8-224      0.21      0.03     80.74    384.63      0.00      8.18      2.25      0.05
00时10分01秒   dev65-0      0.21      0.03     80.74    384.63      0.00      9.52      2.82      0.06
00时10分01秒  dev8-240      0.21      0.03     80.74    384.63      0.00      9.87      2.62      0.05
00时10分01秒  dev65-16      0.21      0.03     80.74    384.63      0.00      9.70      2.58      0.05
00时10分01秒  dev65-32      0.53      0.27     85.78    162.36      0.00      7.42      1.32      0.07
00时10分01秒  dev65-48      0.52      0.27     85.52    165.53      0.00      7.55      1.33      0.07
00时10分01秒  dev65-64      0.50      0.27     85.72    170.30      0.00      6.36      1.24      0.06
00时10分01秒  dev65-80      0.51      0.27     85.82    167.17      0.00      5.16      1.18      0.06
00时10分01秒 dev65-128      0.21      0.00     80.74    390.71      0.00      9.36      2.38      0.05
00时10分01秒  dev65-96      0.51      0.27     85.47    168.13      0.00      6.83      1.23      0.06
00时10分01秒 dev65-112      0.51      0.27     85.75    167.04      0.00      6.58      1.15      0.06
00时10分01秒 dev65-144      0.21      0.00     80.74    390.71      0.00     10.31      2.86      0.06
00时10分01秒 dev65-192      0.21      0.00     80.74    390.71      0.00      9.18      2.36      0.05
00时10分01秒 dev65-160      0.21      0.00     80.74    390.71      0.00     10.47      2.65      0.05
00时10分01秒 dev65-176      0.21      0.00     80.74    390.71      0.00      8.73      2.02      0.04
00时10分01秒 dev65-208      0.21      0.00     80.74    390.71      0.00      8.48      2.40      0.05
00时10分01秒 dev65-240      0.21      0.00     80.75    387.65      0.00     10.34      2.71      0.06
00时10分01秒 dev65-224      0.21      0.00     80.74    390.71      0.00      6.80      2.04      0.04
00时10分01秒  dev66-16      0.21      0.00     80.75    387.65      0.00      9.62      2.42      0.05
00时10分01秒   dev66-0      0.21      0.00     80.74    390.71      0.00      9.27      2.54      0.05
00时10分01秒  dev66-64      0.12      0.27      1.37     13.31      0.00      0.85      0.31      0.00
00时10分01秒  dev66-32      0.21      0.00     80.74    390.71      0.00      8.56      2.04      0.04
00时10分01秒  dev66-48      0.21      0.00     80.75    387.65      0.00      8.78      2.50      0.05
00时10分01秒  dev253-0      0.12      0.27      1.37     13.31      0.00      0.96      0.31      0.00
00时10分01秒  dev253-1      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
00时10分01秒  dev253-2      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00

...

09时30分01秒       DEV       tps  rd_sec/s  wr_sec/s  avgrq-sz  avgqu-sz     await     svctm     %util
09时40分01秒   dev8-16     23.22   1319.34   1370.43    115.86      0.03      1.26      0.74      1.72
09时40分01秒    dev8-0     23.14   1319.56   1370.30    116.24      0.03      1.22      0.73      1.69
09时40分01秒   dev8-48     23.21   1319.34   1370.77    115.92      0.03      1.24      0.75      1.74
09时40分01秒   dev8-32     23.17   1319.45   1370.55    116.09      0.03      1.20      0.72      1.68
09时40分01秒   dev8-80    152.80   1323.14   2405.75     24.40      3.66     23.94      1.61     24.52
09时40分01秒   dev8-96     23.12   1311.44   1366.49    115.83      0.02      1.08      0.67      1.55
09时40分01秒  dev8-128     23.09   1311.33   1366.81    116.01      0.03      1.19      0.72      1.67
09时40分01秒  dev8-112     23.10   1311.44   1365.51    115.87      0.03      1.09      0.67      1.55
09时40分01秒  dev8-144     23.12   1311.44   1365.41    115.79      0.03      1.24      0.73      1.70
09时40分01秒  dev8-160     23.13   1311.33   1365.94    115.77      0.03      1.20      0.72      1.67
09时40分01秒  dev8-176     23.07   1311.33   1364.38    115.99      0.03      1.18      0.70      1.62
09时40分01秒   dev8-64    142.08   1315.37   2271.00     25.24      0.26      1.80      1.67     23.79
09时40分01秒  dev8-192     23.10   1330.38   1374.17    117.07      0.03      1.17      0.70      1.62
09时40分01秒  dev8-208     23.08   1330.49   1373.03    117.12      0.03      1.24      0.74      1.70
09时40分01秒  dev8-224     23.08   1330.38   1374.82    117.21      0.03      1.32      0.74      1.72
09时40分01秒   dev65-0     23.10   1330.38   1373.47    117.07      0.03      1.34      0.69      1.60
09时40分01秒  dev8-240     23.11   1330.28   1374.73    117.04      0.03      1.21      0.71      1.64
09时40分01秒  dev65-16     23.09   1330.60   1372.91    117.11      0.03      1.25      0.73      1.69
09时40分01秒  dev65-32     23.54   1332.97   1389.66    115.68      0.03      1.19      0.68      1.61
09时40分01秒  dev65-48     23.54   1333.18   1389.17    115.65      0.03      1.22      0.71      1.68
09时40分01秒  dev65-64     23.56   1333.08   1391.55    115.66      0.03      1.30      0.68      1.59
09时40分01秒  dev65-80     23.59   1333.29   1391.66    115.53      0.03      1.16      0.69      1.63
09时40分01秒 dev65-128     22.92   1307.14   1359.23    116.33      0.03      1.19      0.72      1.64
09时40分01秒  dev65-96     23.56   1333.29   1389.83    115.59      0.03      1.24      0.72      1.70
09时40分01秒 dev65-112     23.57   1333.08   1391.10    115.56      0.03      1.14      0.68      1.60
09时40分01秒 dev65-144     22.89   1306.98   1357.81    116.44      0.03      1.21      0.73      1.67
09时40分01秒 dev65-192     22.90   1306.93   1359.58    116.45      0.03      1.13      0.69      1.59
09时40分01秒 dev65-160     22.84   1306.98   1357.09    116.63      0.03      1.24      0.73      1.67
09时40分01秒 dev65-176     22.92   1306.98   1358.31    116.31      0.03      1.12      0.68      1.57
09时40分01秒 dev65-208     22.86   1307.09   1357.45    116.58      0.03      1.17      0.71      1.63
09时40分01秒 dev65-240     23.42   1341.72   1387.09    116.51      0.03      1.19      0.72      1.68
09时40分01秒 dev65-224     23.41   1341.61   1387.06    116.54      0.03      1.12      0.68      1.59
09时40分01秒  dev66-16     23.40   1341.60   1385.83    116.56      0.03      1.26      0.76      1.77
09时40分01秒   dev66-0     23.42   1341.40   1387.37    116.51      0.03      1.20      0.73      1.71
09时40分01秒  dev66-64      0.78     36.73      2.48     50.06      0.00      0.51      0.21      0.02
09时40分01秒  dev66-32     23.43   1341.72   1389.69    116.58      0.03      1.09      0.66      1.56
09时40分01秒  dev66-48     23.40   1341.82   1387.50    116.64      0.03      1.13      0.68      1.60
09时40分01秒  dev253-0      0.77     36.73      2.48     50.82      0.00      0.53      0.21      0.02
09时40分01秒  dev253-1      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
09时40分01秒  dev253-2      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00

...

11时00分02秒       DEV       tps  rd_sec/s  wr_sec/s  avgrq-sz  avgqu-sz     await     svctm     %util
11时10分01秒   dev8-16      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
11时10分01秒    dev8-0      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
11时10分01秒   dev8-48      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
11时10分01秒   dev8-32      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
11时10分01秒   dev8-80      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
11时10分01秒   dev8-96      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
11时10分01秒  dev8-128      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
11时10分01秒  dev8-112      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
11时10分01秒  dev8-144      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
11时10分01秒  dev8-160      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
11时10分01秒  dev8-176      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
11时10分01秒   dev8-64      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
11时10分01秒  dev8-192      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
11时10分01秒  dev8-208      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
11时10分01秒  dev8-224      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
11时10分01秒   dev65-0      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
11时10分01秒  dev8-240      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
11时10分01秒  dev65-16      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
11时10分01秒  dev65-32      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
11时10分01秒  dev65-48      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
11时10分01秒  dev65-64      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
11时10分01秒  dev65-80      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
11时10分01秒 dev65-128      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
11时10分01秒  dev65-96      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
11时10分01秒 dev65-112      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
11时10分01秒 dev65-144      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
11时10分01秒 dev65-192      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
11时10分01秒 dev65-160      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
11时10分01秒 dev65-176      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
11时10分01秒 dev65-208      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
11时10分01秒 dev65-240      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
11时10分01秒 dev65-224      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
11时10分01秒  dev66-16      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
11时10分01秒   dev66-0      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
11时10分01秒  dev66-64      0.11      0.00      1.31     11.73      0.00      0.97      0.36      0.00
11时10分01秒  dev66-32      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
11时10分01秒  dev66-48      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
11时10分01秒  dev253-0      0.10      0.00      1.31     12.89      0.00      1.11      0.39      0.00
11时10分01秒  dev253-1      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
11时10分01秒  dev253-2      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00

平均时间:       DEV       tps  rd_sec/s  wr_sec/s  avgrq-sz  avgqu-sz     await     svctm     %util
平均时间:   dev8-16      0.99     19.92    148.47    170.40      0.00      2.79      1.00      0.10
平均时间:    dev8-0      0.99     19.92    148.53    170.61      0.00      2.80      0.99      0.10
平均时间:   dev8-48      0.99     19.92    148.49    170.65      0.00      2.91      1.03      0.10
平均时间:   dev8-32      0.99     19.92    148.53    170.82      0.00      2.82      1.01      0.10
平均时间:   dev8-80      9.50     20.03    216.67     24.92      0.24     25.43      1.68      1.60
平均时间:   dev8-96      0.99     19.82    148.36    169.19      0.00      2.76      0.99      0.10
平均时间:  dev8-128      0.99     19.82    148.36    169.14      0.00      2.83      1.02      0.10
平均时间:  dev8-112      1.00     19.82    148.20    168.83      0.00      2.77      0.98      0.10
平均时间:  dev8-144      1.00     19.82    148.35    169.01      0.00      2.91      1.03      0.10
平均时间:  dev8-160      1.00     19.82    148.37    168.89      0.00      2.89      1.04      0.10
平均时间:  dev8-176      0.99     19.82    148.34    169.30      0.00      2.90      1.02      0.10
平均时间:   dev8-64    305.67    229.30   2314.31      8.32      0.23      0.76      0.10      3.04
平均时间:  dev8-192      0.99     20.11    145.47    167.61      0.00      2.87      1.01      0.10
平均时间:  dev8-208      0.98     20.11    145.59    168.41      0.00      2.79      1.00      0.10
平均时间:  dev8-224      0.99     20.11    145.52    168.01      0.00      2.79      1.00      0.10
平均时间:   dev65-0      0.99     20.11    145.47    167.82      0.00      2.79      0.99      0.10
平均时间:  dev8-240      0.98     20.10    145.50    168.15      0.00      2.81      0.99      0.10
平均时间:  dev65-16      0.99     20.11    145.52    168.03      0.00      2.81      1.01      0.10
平均时间:  dev65-32      1.25     20.34    151.58    137.28      0.00      3.19      0.84      0.11
平均时间:  dev65-48      1.25     20.35    151.41    137.90      0.00      3.21      0.86      0.11
平均时间:  dev65-64      1.24     20.34    151.58    138.71      0.00      3.09      0.83      0.10
平均时间:  dev65-80      1.24     20.35    151.74    138.60      0.00      3.02      0.83      0.10
平均时间: dev65-128      0.99     19.74    147.06    168.64      0.00      2.83      0.98      0.10
平均时间:  dev65-96      1.24     20.35    151.38    138.56      0.00      3.12      0.84      0.10
平均时间: dev65-112      1.24     20.34    151.59    138.72      0.00      3.02      0.82      0.10
平均时间: dev65-144      0.99     19.73    147.20    169.22      0.00      2.78      0.98      0.10
平均时间: dev65-192      0.99     19.73    147.13    168.75      0.00      2.74      0.96      0.09
平均时间: dev65-160      0.99     19.73    147.04    168.77      0.00      2.69      0.96      0.10
平均时间: dev65-176      0.99     19.73    147.15    168.84      0.00      2.68      0.94      0.09
平均时间: dev65-208      0.99     19.74    146.97    168.64      0.00      2.75      0.97      0.10
平均时间: dev65-240      0.98     20.25    145.45    168.27      0.00      2.77      0.98      0.10
平均时间: dev65-224      0.98     20.25    145.40    168.27      0.00      2.63      0.94      0.09
平均时间:  dev66-16      0.98     20.25    145.36    168.29      0.00      2.84      1.01      0.10
平均时间:   dev66-0      0.98     20.25    145.34    168.28      0.00      2.68      0.97      0.10
平均时间:  dev66-64      0.23     22.38     53.15    327.95      0.01     52.65      0.64      0.01
平均时间:  dev66-32      0.99     20.25    145.44    168.20      0.00      2.62      0.94      0.09
平均时间:  dev66-48      0.98     20.25    145.42    168.42      0.00      2.79      0.97      0.10
平均时间:  dev253-0      0.22     22.22     53.15    340.82      0.01     55.20      0.67      0.01
平均时间:  dev253-1      0.00      0.05      0.00     48.19      0.00      0.19      0.12      0.00
平均时间:  dev253-2      0.00      0.05      0.00     48.19      0.00      0.14      0.12      0.00
```



## pidstat

pidstat 默认输出 CPU 使用情况，-d 参数可以查看磁盘的 IO 统计信息

```bash
[root@localhost ~]# pidstat -d 1
Linux 3.10.0-957.el7.x86_64 (localhost.localdomain) 	2022年07月12日 	_x86_64_	(8 CPU)

11时16分41秒   UID       PID   kB_rd/s   kB_wr/s kB_ccwr/s  Command
11时16分42秒     0     10316      0.00 180992.00      0.00  mkfs.xfs

11时16分42秒   UID       PID   kB_rd/s   kB_wr/s kB_ccwr/s  Command
11时16分43秒     0     10316      0.00 177664.00      0.00  mkfs.xfs

11时16分43秒   UID       PID   kB_rd/s   kB_wr/s kB_ccwr/s  Command
11时16分44秒     0      2976      8.00      0.00      0.00  systemd-udevd
11时16分44秒     0     10240     12.00 1907884.00      0.00  python
11时16分44秒     0     10320      4.00  68368.00      0.00  mkfs.xfs
11时16分44秒     0     10321   1036.00      0.00      0.00  systemd-udevd
^C

平均时间:   UID       PID   kB_rd/s   kB_wr/s kB_ccwr/s  Command
平均时间:     0      2976      2.67      0.00      0.00  systemd-udevd
平均时间:     0     10240      4.00 635961.33      0.00  python
平均时间:     0     10320      1.33  22789.33      0.00  mkfs.xfs
平均时间:     0     10321    345.33      0.00      0.00  systemd-udevd

```

也可以查看指定进程的 IO 情况：

```bash
[root@localhost ~]# pidstat -d 1 -p `pidof minio`
Linux 3.10.0-957.el7.x86_64 (localhost.localdomain) 	2022年07月12日 	_x86_64_	(8 CPU)

11时20分27秒   UID       PID   kB_rd/s   kB_wr/s kB_ccwr/s  Command
11时20分28秒     0     10771      0.00 585204.00      0.00  minio
11时20分29秒     0     10771      0.00 589268.00      0.00  minio
11时20分30秒     0     10771      0.00 575432.00      0.00  minio
11时20分31秒     0     10771      0.00 596484.00      0.00  minio
11时20分32秒     0     10771      0.00 577260.00      0.00  minio
^C
平均时间:     0     10771      0.00 584729.60      0.00  minio

```



## other

另外像 perf 和 blktrace 都可以进一步去跟踪 IO 的情况，但是本文不打算继续介绍他们，因为会涉及很多具体的存储相关的知识，本文只抛砖引玉，介绍一些基本的性能分析方法，感兴趣的朋友可以自行深入了解。



---

# 网络

## ss

ss 是一个套接字统计工具，显示的内容和 netstat 类似，但能够显示更多更详细的有关 TCP 和连接状态的信息，而且比 netstat 更快。

```bash
[root@localhost ~]# ss
Netid  State      Recv-Q Send-Q                         Local Address:Port       Peer Address:Port
u_str  ESTAB      0      0                                  * 32137                         * 34063
u_str  ESTAB      0      0        /run/dbus/system_bus_socket 34063                         * 32137
...               
u_str  ESTAB      0      0        /run/dbus/system_bus_socket 34062                         * 34061
u_str  ESTAB      0      0                                  * 30525                         * 34904
tcp    ESTAB      0      664                    192.168.0.175:ssh                192.168.2.23:63372
tcp    ESTAB      0      0                          127.0.0.1:55488                 127.0.0.1:cslistener
tcp    ESTAB      0      0                      192.168.0.175:ssh                192.168.2.23:53697
tcp    ESTAB      0      0                      192.168.0.175:ssh                192.168.2.23:59593
tcp    ESTAB      0      0                   ::ffff:127.0.0.1:cslistener     ::ffff:127.0.0.1:55488
```

-t 参数可以只显示 TCP 套接字， -u 表示只显示 UDP

```bash
[root@localhost ~]# ss -t
State      Recv-Q Send-Q       Local Address:Port                        Peer Address:Port
ESTAB      0      248          192.168.0.175:ssh                         192.168.2.23:63372
ESTAB      0      0                127.0.0.1:55488                          127.0.0.1:cslistener
ESTAB      0      0            192.168.0.175:ssh                         192.168.2.23:53697
ESTAB      0      116          192.168.0.175:ssh                         192.168.2.23:59593
ESTAB      0      0         ::ffff:127.0.0.1:cslistener              ::ffff:127.0.0.1:55488
```

-i 显示 TCP 内部信息，-e 显示扩展的套接字信息，-p 显示进程信息，-m 显示内存使用情况。

```bash
[root@localhost ~]# ss -tiepm
State      Recv-Q Send-Q                                       Local Address:Port                                                        Peer Address:Port                
ESTAB      0      36                                           192.168.0.175:ssh                                                         192.168.2.23:63372                 users:(("sshd",pid=6779,fd=3)) timer:(on,222ms,0) ino:40613 sk:ffff8e39f4778000 <->
	 skmem:(r0,rb369280,t0,tb87040,f1792,w2304,o0,bl0,d0) sack cubic wscale:8,7 rto:235 rtt:34.435/19.834 ato:40 mss:1460 rcvmss:1460 advmss:1460 cwnd:10 ssthresh:18 bytes_acked:2259225 bytes_received:117018 segs_out:10922 segs_in:8537 send 3.4Mbps lastsnd:13 lastrcv:14 lastack:14 pacing_rate 6.8Mbps unacked:1 rcv_rtt:355890 rcv_space:34960
ESTAB      0      0                                            192.168.0.175:ssh                                                         192.168.2.23:53697                 users:(("sshd",pid=10011,fd=3)) timer:(keepalive,53min,0) ino:57768 sk:ffff8e39f2a187c0 <->
	 skmem:(r0,rb369280,t0,tb87040,f4096,w0,o0,bl0,d0) sack cubic wscale:8,7 rto:213 rtt:12.667/20.394 ato:40 mss:1460 rcvmss:1460 advmss:1460 cwnd:10 bytes_acked:111021 bytes_received:31790 segs_out:1590 segs_in:1514 send 9.2Mbps lastsnd:26666266 lastrcv:23823 lastack:3923856 pacing_rate 18.4Mbps rcv_rtt:286094 rcv_space:29378
ESTAB      0      0                                                127.0.0.1:55490                                                          127.0.0.1:cslistener            users:(("mc",pid=11517,fd=8)) timer:(keepalive,12sec,0) ino:67188 sk:ffff8e39ef3a1740 <->
	 skmem:(r0,rb1061296,t0,tb2626560,f4096,w0,o0,bl0,d0) ts sack cubic wscale:7,7 rto:201 rtt:0.016/0.008 ato:40 mss:22400 rcvmss:556 advmss:65483 cwnd:10 bytes_acked:545 bytes_received:97639 segs_out:487 segs_in:486 send 112000.0Mbps lastsnd:242300 lastrcv:299 lastack:1799 rcv_rtt:501 rcv_space:43690
ESTAB      0      0                                            192.168.0.175:ssh                                                         192.168.2.23:59593                 users:(("sshd",pid=7259,fd=3)) timer:(keepalive,31min,0) ino:43143 sk:ffff8e39eef68000 <->
	 skmem:(r0,rb471680,t0,tb87040,f4096,w0,o0,bl0,d0) sack cubic wscale:8,7 rto:249 rtt:48.441/4.309 ato:40 mss:1460 rcvmss:1460 advmss:1460 cwnd:10 ssthresh:16 bytes_acked:1715137 bytes_received:150986 segs_out:17483 segs_in:18221 send 2.4Mbps lastsnd:115 lastrcv:22940 lastack:60 pacing_rate 4.8Mbps rcv_rtt:419530 rcv_space:46936
ESTAB      0      0                                         ::ffff:127.0.0.1:cslistener                                              ::ffff:127.0.0.1:55490                 users:(("minio",pid=10771,fd=11)) timer:(keepalive,12sec,0) ino:75877 sk:ffff8e39f6a198c0 <->
	 skmem:(r0,rb1061488,t0,tb2626560,f0,w0,o0,bl0,d0) ts sack cubic wscale:7,7 rto:201 rtt:0.026/0.007 ato:40 mss:65483 rcvmss:544 advmss:65483 cwnd:10 bytes_acked:97639 bytes_received:544 segs_out:485 segs_in:485 send 201486.2Mbps lastsnd:310 lastrcv:242311 lastack:310 rcv_rtt:1 rcv_space:43690


```

比如可以看最后一项，`users:(("minio",pid=10771,fd=11))` 表示这个 TCP 由 minio 进程创建，pid 为 10771，打开的文件描述符为 11。`timer:(keepalive,12sec,0)` 表示使用的是长连接，已经连接了 12s。`rto:201` 表示 TCP 重传超时为 201 ms。`rtt:0.026/0.007` 表示平均往返时间为 0.026 ms，有 0.007 ms 平均偏差，`mss:65483` 表示最大的分段大小为 65483 字节。`cwnd:10` 表示拥塞窗口大小为 10 MSS。`bytes_acked:97639 bytes_received:544` 表示成功传输 97639 字节，接收 544 字节。

另外还有一些常用参数，比如 -n 表示不解析服务的名称而是直接显示端口号，-l 表示只显示处于监听状态的端口

```bash
# 查看主机监听的端口
[root@localhost ~]# ss -tnlp
State      Recv-Q Send-Q                                         Local Address:Port                                                        Peer Address:Port              
LISTEN     0      128                                                        *:22                                                                     *:*                   users:(("sshd",pid=5982,fd=3))
LISTEN     0      100                                                127.0.0.1:25                                                                     *:*                   users:(("master",pid=6371,fd=13))
LISTEN     0      128                                                       :::9000                                                                  :::*                   users:(("minio",pid=10771,fd=10))
LISTEN     0      128                                                       :::9001                                                                  :::*                   users:(("minio",pid=10771,fd=9))
LISTEN     0      128                                                       :::22                                                                    :::*                   users:(("sshd",pid=5982,fd=4))
LISTEN     0      100                                                      ::1:25                                                                    :::*                   users:(("master",pid=6371,fd=14))

# -r 可以解析 IP 和端口号
[root@localhost ~]# ss -trlp
State      Recv-Q Send-Q                                       Local Address:Port                                                        Peer Address:Port                
LISTEN     0      128                                                      *:ssh                                                                    *:*                     users:(("sshd",pid=5982,fd=3))
LISTEN     0      100                                              localhost:smtp                                                                   *:*                     users:(("master",pid=6371,fd=13))
LISTEN     0      128                                                     :::cslistener                                                            :::*                     users:(("minio",pid=10771,fd=10))
LISTEN     0      128                                                     :::etlservicemgr                                                         :::*                     users:(("minio",pid=10771,fd=9))
LISTEN     0      128                                                     :::ssh                                                                   :::*                     users:(("sshd",pid=5982,fd=4))
LISTEN     0      100                                              localhost:smtp                                                                  :::*                     users:(("master",pid=6371,fd=14))

# -a 既包含监听的端口，也包含建立的连接
[root@localhost ~]# ss -tna
State      Recv-Q Send-Q                                         Local Address:Port                                                        Peer Address:Port              
LISTEN     0      128                                                        *:22                                                                     *:*                  
LISTEN     0      100                                                127.0.0.1:25                                                                     *:*                  
ESTAB      0      248                                            192.168.0.175:22                                                          192.168.2.23:63372              
ESTAB      0      0                                              192.168.0.175:22                                                          192.168.2.23:53697              
ESTAB      0      0                                                  127.0.0.1:55490                                                          127.0.0.1:9000               
ESTAB      0      116                                            192.168.0.175:22                                                          192.168.2.23:59593              
LISTEN     0      128                                                       :::9000                                                                  :::*                  
LISTEN     0      128                                                       :::9001                                                                  :::*                  
LISTEN     0      128                                                       :::22                                                                    :::*                  
LISTEN     0      100                                                      ::1:25                                                                    :::*                  
ESTAB      0      0                                           ::ffff:127.0.0.1:9000                                                    ::ffff:127.0.0.1:55490              

# -s 显示概要信息
[root@localhost ~]# ss -s
Total: 229 (kernel 425)
TCP:   10 (estab 3, closed 1, orphaned 0, synrecv 0, timewait 1/0), ports 0

Transport Total     IP        IPv6
*	  425       -         -        
RAW	  1         0         1        
UDP	  2         1         1        
TCP	  9         5         4        
INET	  12        6         6        
FRAG	  0         0         0 
```



## nstat

这个命令可输出由内核维护的各种网络指标以及它们的 SNMP 名称，默认行为是重置内核计数器，-s 参数来比秒重新设置计数器。如果网络有问题，可以通过重置计数器，然后观察参数变化来定位问题。

```bash
[root@localhost ~]# nstat -s
#kernel
IpInReceives                    52                 0.0
IpInDelivers                    34                 0.0
IpOutRequests                   24                 0.0
TcpInSegs                       32                 0.0
TcpOutSegs                      24                 0.0
Ip6InReceives                   30                 0.0
Ip6InDelivers                   30                 0.0
Ip6InMcastPkts                  30                 0.0
Ip6InOctets                     2160               0.0
Ip6InMcastOctets                2160               0.0
Ip6InNoECTPkts                  30                 0.0
Icmp6InMsgs                     30                 0.0
Icmp6InNeighborAdvertisements   30                 0.0
Icmp6InType136                  30                 0.0
TcpExtDelayedACKs               2                  0.0
TcpExtTCPHPHits                 12                 0.0
TcpExtTCPPureAcks               4                  0.0
TcpExtTCPHPAcks                 10                 0.0
TcpExtTCPAutoCorking            1                  0.0
TcpExtTCPOrigDataSent           16                 0.0
IpExtInBcastPkts                2                  0.0
IpExtInOctets                   3995               0.0
IpExtOutOctets                  2300               0.0
IpExtInBcastOctets              390                0.0
IpExtInNoECTPkts                52                 0.0
```



## sar

在 linux 中有以下选项提供网络统计信息

-   -n DEV：网络接口统计信息
-   -n EDEV：网络接口错误
-   -n IP：IP 数据报统计信息
-   -n EIP：IP 错误统计信息
-   -n TCP：TCP 统计信息
-   -n ETCP：TCP 错误统计信息
-   -n SOCK：套接字使用信息

```bash
[root@localhost ~]# sar -n DEV 1
Linux 3.10.0-957.el7.x86_64 (localhost.localdomain) 	2022年07月12日 	_x86_64_	(8 CPU)

19时29分33秒     IFACE   rxpck/s   txpck/s    rxkB/s    txkB/s   rxcmp/s   txcmp/s  rxmcst/s
19时29分34秒      eno1    113.00      0.00      6.62      0.00      0.00      0.00      5.00
19时29分34秒      eno2      0.00      0.00      0.00      0.00      0.00      0.00      0.00
19时29分34秒        lo      0.00      0.00      0.00      0.00      0.00      0.00      0.00

```

输出信息说明：

-   IFACE：LAN接口
-   rxpck/s：每秒钟接收的数据包
-   txpck/s：每秒钟发送的数据包
-   rxbyt/s：每秒钟接收的字节数
-   txbyt/s：每秒钟发送的字节数
-   rxcmp/s：每秒钟接收的压缩数据包
-   txcmp/s：每秒钟发送的压缩数据包
-   rxmcst/s：每秒钟接收的多播数据包



## IP冲突检查

日常工作中我们可能会遇到 IP 冲突的情况，可以通过 arping 来检查。arping 主要干的活就是查看 ip 的 MAC 地址及 IP 占用的问题。

```bash
# 首先查看自己有哪些网络端口
[root@localhost ~]# ip addr
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN group default qlen 1000
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
    inet 127.0.0.1/8 scope host lo
       valid_lft forever preferred_lft forever
    inet6 ::1/128 scope host 
       valid_lft forever preferred_lft forever
2: eno1: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc mq state UP group default qlen 1000
    link/ether a4:bf:01:4b:b2:12 brd ff:ff:ff:ff:ff:ff
    inet 192.168.0.175/24 brd 192.168.0.255 scope global noprefixroute eno1
       valid_lft forever preferred_lft forever
    inet6 fe80::8bcb:71db:7690:c743/64 scope link noprefixroute 
       valid_lft forever preferred_lft forever
3: eno2: <NO-CARRIER,BROADCAST,MULTICAST,UP> mtu 1500 qdisc mq state DOWN group default qlen 1000
    link/ether a4:bf:01:4b:b2:13 brd ff:ff:ff:ff:ff:ff
    
# -I 用来发送 ARP REQUEST 包的网络设备的名称，如果返回多个 MAC 地址，那么就是 IP 冲突了
[root@localhost ~]# arping -I eno1 192.168.2.23
ARPING 192.168.2.23 from 192.168.0.175 eno1
Unicast reply from 192.168.2.23 [8C:EC:4B:8B:0C:48]  0.656ms
Unicast reply from 192.168.2.23 [8C:EC:4B:8B:0C:48]  0.679ms
Unicast reply from 192.168.2.23 [8C:EC:4B:8B:0C:48]  0.849ms
Unicast reply from 192.168.2.23 [8C:EC:4B:8B:0C:48]  0.824ms
```





---

# 系统调用分析

我们执行的程序几乎最后都会调用操作系统的接口也就是系统调用去实现，所以有时候我们可以跟踪系统调用来辅助我们分析问题。

## strace

strace 可以监控程序运行时调用的系统调用，比如我现在想要知道 ls 的流程，可以看到 ls 实际是调用了 /usr/bin/ls，实际上这里是会从系统路径里去尝试可运行的 ls，然后去访问一下动态库来装载完整的程序，最后可以看到通过 openat 打开了 `.` 文件，返回了文件描述符 3，getdents 从文件描述符 3 中读取 `.` 目录下的文件信息。

```bash
[root@localhost 35]# strace ls
execve("/usr/bin/ls", ["ls"], 0x7ffcf0deb4c0 /* 23 vars */) = 0
brk(NULL)                               = 0x1726000
mmap(NULL, 4096, PROT_READ|PROT_WRITE, MAP_PRIVATE|MAP_ANONYMOUS, -1, 0) = 0x7f4b71d4f000
access("/etc/ld.so.preload", R_OK)      = -1 ENOENT (没有那个文件或目录)
open("/etc/ld.so.cache", O_RDONLY|O_CLOEXEC) = 3
fstat(3, {st_mode=S_IFREG|0644, st_size=20333, ...}) = 0
mmap(NULL, 20333, PROT_READ, MAP_PRIVATE, 3, 0) = 0x7f4b71d4a000
close(3)                                = 0
open("/lib64/libselinux.so.1", O_RDONLY|O_CLOEXEC) = 3
read(3, "\177ELF\2\1\1\0\0\0\0\0\0\0\0\0\3\0>\0\1\0\0\0\320i\0\0\0\0\0\0"..., 832) = 832
fstat(3, {st_mode=S_IFREG|0755, st_size=155784, ...}) = 0
mmap(NULL, 2255184, PROT_READ|PROT_EXEC, MAP_PRIVATE|MAP_DENYWRITE, 3, 0) = 0x7f4b71908000
mprotect(0x7f4b7192c000, 2093056, PROT_NONE) = 0
mmap(0x7f4b71b2b000, 8192, PROT_READ|PROT_WRITE, MAP_PRIVATE|MAP_FIXED|MAP_DENYWRITE, 3, 0x23000) = 0x7f4b71b2b000
mmap(0x7f4b71b2d000, 6480, PROT_READ|PROT_WRITE, MAP_PRIVATE|MAP_FIXED|MAP_ANONYMOUS, -1, 0) = 0x7f4b71b2d000
close(3)                                = 0
...
ioctl(1, TCGETS, {B38400 opost isig icanon echo ...}) = 0
ioctl(1, TIOCGWINSZ, {ws_row=37, ws_col=172, ws_xpixel=0, ws_ypixel=0}) = 0
openat(AT_FDCWD, ".", O_RDONLY|O_NONBLOCK|O_CLOEXEC|O_DIRECTORY) = 3
getdents(3, /* 3 entries */, 32768)     = 80
getdents(3, /* 0 entries */, 32768)     = 0
close(3)                                = 0
close(1)                                = 0
close(2)                                = 0
exit_group(0)                           = ?
+++ exited with 0 +++
```

要跟踪某个具体的系统调用，-e trace=xxx 即可。但有时候我们要跟踪一类系统调用，比如所有和文件名有关的调用、所有和内存分配有关的调用。

如果人工输入每一个具体的系统调用名称，可能容易遗漏。于是strace提供了几类常用的系统调用组合名字。

```bash
-e trace=file     跟踪和文件访问相关的调用(参数中有文件名)
-e trace=process  和进程管理相关的调用，比如fork/exec/exit_group
-e trace=network  和网络通信相关的调用，比如socket/sendto/connect
-e trace=signal   信号发送和处理相关，比如kill/sigaction
-e trace=desc     和文件描述符相关，比如write/read/select/epoll等
-e trace=ipc      进程间通信相关，比如shmget等
```

绝大多数情况，我们使用上面的组合名字就够了。

当程序出错时，也可以通过 strace 来快速分析到底是哪里出错了，比如有如下的 C 程序。

```c
int main(void) {
	int a = 0;
	int b = 1 / a;
	return 0;
}
```

我们使用 strace 来分析，发现我们收到了 SIGFPE 信号量，这表示一个除零异常信号，这样就可以快速定位问题。

```bash
[root@localhost ~]# strace ./a.out 
execve("./a.out", ["./a.out"], 0x7ffca85a4dd0 /* 23 vars */) = 0
brk(NULL)                               = 0x143b000
mmap(NULL, 4096, PROT_READ|PROT_WRITE, MAP_PRIVATE|MAP_ANONYMOUS, -1, 0) = 0x7f54962ff000
access("/etc/ld.so.preload", R_OK)      = -1 ENOENT (没有那个文件或目录)
open("/etc/ld.so.cache", O_RDONLY|O_CLOEXEC) = 3
fstat(3, {st_mode=S_IFREG|0644, st_size=21782, ...}) = 0
mmap(NULL, 21782, PROT_READ, MAP_PRIVATE, 3, 0) = 0x7f54962f9000
close(3)                                = 0
open("/lib64/libc.so.6", O_RDONLY|O_CLOEXEC) = 3
read(3, "\177ELF\2\1\1\3\0\0\0\0\0\0\0\0\3\0>\0\1\0\0\0`&\2\0\0\0\0\0"..., 832) = 832
fstat(3, {st_mode=S_IFREG|0755, st_size=2156592, ...}) = 0
mmap(NULL, 3985920, PROT_READ|PROT_EXEC, MAP_PRIVATE|MAP_DENYWRITE, 3, 0) = 0x7f5495d11000
mprotect(0x7f5495ed5000, 2093056, PROT_NONE) = 0
mmap(0x7f54960d4000, 24576, PROT_READ|PROT_WRITE, MAP_PRIVATE|MAP_FIXED|MAP_DENYWRITE, 3, 0x1c3000) = 0x7f54960d4000
mmap(0x7f54960da000, 16896, PROT_READ|PROT_WRITE, MAP_PRIVATE|MAP_FIXED|MAP_ANONYMOUS, -1, 0) = 0x7f54960da000
close(3)                                = 0
mmap(NULL, 4096, PROT_READ|PROT_WRITE, MAP_PRIVATE|MAP_ANONYMOUS, -1, 0) = 0x7f54962f8000
mmap(NULL, 8192, PROT_READ|PROT_WRITE, MAP_PRIVATE|MAP_ANONYMOUS, -1, 0) = 0x7f54962f6000
arch_prctl(ARCH_SET_FS, 0x7f54962f6740) = 0
access("/etc/sysconfig/strcasecmp-nonascii", F_OK) = -1 ENOENT (没有那个文件或目录)
access("/etc/sysconfig/strcasecmp-nonascii", F_OK) = -1 ENOENT (没有那个文件或目录)
mprotect(0x7f54960d4000, 16384, PROT_READ) = 0
mprotect(0x600000, 4096, PROT_READ)     = 0
mprotect(0x7f5496300000, 4096, PROT_READ) = 0
munmap(0x7f54962f9000, 21782)           = 0
--- SIGFPE {si_signo=SIGFPE, si_code=FPE_INTDIV, si_addr=0x4004de} ---
+++ killed by SIGFPE +++
浮点数例外
```

strace 不光能追踪系统调用，通过使用参数 -c，它还能将进程所有的系统调用做一个统计分析给你。

```bash
[root@localhost 35]# strace -c ls
% time     seconds  usecs/call     calls    errors syscall
------ ----------- ----------- --------- --------- ----------------
 12.71    0.000023           2        10           read
 12.15    0.000022           1        14           close
 11.05    0.000020           1        11           open
  9.94    0.000018           9         2         2 statfs
  7.18    0.000013           0        27           mmap
  6.08    0.000011           5         2           ioctl
  5.52    0.000010           5         2           rt_sigaction
  5.52    0.000010           5         2           getdents
  4.97    0.000009           4         2           munmap
  4.97    0.000009           3         3           brk
  4.42    0.000008           0        11           fstat
  3.87    0.000007           7         1         1 stat
  3.87    0.000007           7         1           openat
  3.31    0.000006           3         2         1 access
  2.21    0.000004           4         1           rt_sigprocmask
  2.21    0.000004           4         1           getrlimit
  0.00    0.000000           0        18           mprotect
  0.00    0.000000           0         1           execve
  0.00    0.000000           0         1           arch_prctl
  0.00    0.000000           0         1           set_tid_address
  0.00    0.000000           0         1           set_robust_list
------ ----------- ----------- --------- --------- ----------------
100.00    0.000181                   114         4 total
```





## perf

perf 的 trace 子命令默认跟踪系统调用，我们可以观察到进程当前正在做什么。当然 perf 还可以跟踪整个系统，而 strace 只限于一组进程（通常是一个进程），不过由于系统调用是一件很频繁的事情，当你去观察的时候，你最好已经知道你想干什么，而不是对着哗啦啦的输出发呆。

```bash
[root@localhost ~]# perf trace -p $(pgrep minio)
         ? (         ): minio/25084  ... [continued]: futex()) = -1 ETIMEDOUT Connection timed out
     0.009 ( 0.003 ms): minio/25084 futex(uaddr: 0xc00346cd50, op: WAKE|PRIV, val: 1                      ) = 1
         ? (         ): minio/25088  ... [continued]: futex()) = 0
     0.013 (         ): minio/25084 nanosleep(rqtp: 0xc000085f10                                          ) ...
     0.119 ( 0.003 ms): minio/25088 futex(uaddr: 0xc000701950, op: WAKE|PRIV, val: 1                      ) = 1
     0.657 ( 0.003 ms): minio/25088 epoll_pwait(epfd: 3<anon_inode:[eventpoll]>, events: 0xc00010d8e8, maxevents: 128, sigsetsize: 139804515624960) = 0
     0.691 ( 0.001 ms): minio/25088 epoll_pwait(epfd: 3<anon_inode:[eventpoll]>, events: 0xc00010d8e8, maxevents: 128, sigsetsize: 139804515643363) = 0
     0.739 ( 0.001 ms): minio/25088 epoll_pwait(epfd: 3<anon_inode:[eventpoll]>, events: 0xc00010d8e8, maxevents: 128, sigsetsize: 139804515641344) = 0
     0.801 ( 0.002 ms): minio/25088 epoll_pwait(epfd: 3<anon_inode:[eventpoll]>, events: 0xc00010d8e8, maxevents: 128, sigsetsize: 139804515637248) = 0
     0.860 ( 0.001 ms): minio/25088 epoll_pwait(epfd: 3<anon_inode:[eventpoll]>, events: 0xc00010d8e8, maxevents: 128, sigsetsize: 139804515633152) = 0
     0.968 ( 0.003 ms): minio/25088 epoll_pwait(epfd: 3<anon_inode:[eventpoll]>, events: 0xc00010d8e8, maxevents: 128, sigsetsize: 139804515629056) = 0
     0.984 ( 0.002 ms): minio/25088 epoll_pwait(epfd: 3<anon_inode:[eventpoll]>, events: 0xc00010d8e8, maxevents: 128, sigsetsize: 139804514582006) = 0
     1.111 ( 0.002 ms): minio/25088 epoll_pwait(epfd: 3<anon_inode:[eventpoll]>, events: 0xc00010d8e8, maxevents: 128, sigsetsize: 139804515126286) = 0
     1.298 ( 0.003 ms): minio/25088 epoll_pwait(epfd: 3<anon_inode:[eventpoll]>, events: 0xc00010d8e8, maxevents: 128, sigsetsize: 139804515082696) = 0
...
     8.988 ( 0.002 ms): minio/25143 getpid(                                                               ) = 25083
     8.992 ( 0.004 ms): minio/25143 tgkill(tgid: 25083 (minio), pid: 25088 (minio), sig: URG              ) = 0
    20.178 (         ): minio/25143 rt_sigreturn(arg0: 76917, arg1: 99477884, arg2: 0, arg3: 4325312, arg4: 89197280, arg5: 315) ...
         ? (         ): minio/25146  ... [continued]: futex()) = 0
    21.328 (         ): minio/25143 futex(uaddr: 0xc003483950, op: WAIT|PRIV                              ) ...
    10.896 ( 9.289 ms): minio/25146 futex(uaddr: 0xc00346d150, op: WAIT|PRIV                              ) = 0
    21.707 ( 0.002 ms): minio/25146 getpid(                                                               ) = 25083
    21.711 ( 0.003 ms): minio/25146 tgkill(tgid: 25083 (minio), pid: 25098 (minio), sig: URG              ) = 0
    21.917 ( 0.125 ms): minio/25146 futex(uaddr: 0xc00346d150, op: WAIT|PRIV                              ) = 0
    22.048 ( 0.002 ms): minio/25146 sched_yield(                                                          ) = 0
    22.050 ( 0.208 ms): minio/25146 futex(uaddr: 0x5edd6a8, op: WAIT|PRIV, val: 2                         ) = 0
    22.260 ( 0.001 ms): minio/25146 futex(uaddr: 0x5edd6a8, op: WAKE|PRIV, val: 1                         ) = 0
    22.290 ( 0.001 ms): minio/25146 sched_yield(                                                          ) = 0
    22.292 ( 0.136 ms): minio/25146 futex(uaddr: 0x5eaef78, op: WAIT|PRIV, val: 2                         ) = 0
    22.430 ( 0.001 ms): minio/25146 futex(uaddr: 0x5eaf098, op: WAKE|PRIV, val: 1                         ) = 0
    22.432 ( 0.001 ms): minio/25146 futex(uaddr: 0x5eaef78, op: WAKE|PRIV, val: 1                         ) = 0
...
```

perf 也可以对结果汇总，一般先查看汇总信息可以帮助我们快速了解情况。

```bash
[root@localhost ~]# perf trace -s -p $(pgrep minio)
^C
 Summary of events:

 minio (25088), 72 events, 1.9%

   syscall            calls    total       min       avg       max      stddev
                               (msec)    (msec)    (msec)    (msec)        (%)
   --------------- -------- --------- --------- --------- ---------     ------
   epoll_pwait           19  7103.086     0.003   373.847  2000.936     42.37%
   futex                 10     0.172     0.000     0.017     0.107     58.53%
   sched_yield            2     0.022     0.010     0.011     0.012     12.08%
   newfstatat             1     0.018     0.018     0.018     0.018      0.00%
   read                   1     0.012     0.012     0.012     0.012      0.00%
   tgkill                 1     0.005     0.005     0.005     0.005      0.00%
   getpid                 1     0.002     0.002     0.002     0.002      0.00%


 minio (25093), 492 events, 13.1%

   syscall            calls    total       min       avg       max      stddev
                               (msec)    (msec)    (msec)    (msec)        (%)
   --------------- -------- --------- --------- --------- ---------     ------
   epoll_pwait          243     0.999     0.003     0.004     0.062      5.97%
   futex                  2     0.012     0.000     0.006     0.012    100.00%


 minio (25146), 738 events, 19.6%

   syscall            calls    total       min       avg       max      stddev
                               (msec)    (msec)    (msec)    (msec)        (%)
   --------------- -------- --------- --------- --------- ---------     ------
   futex                 18  7179.190     0.000   398.844  2001.533     46.11%
   epoll_pwait          330     1.376     0.002     0.004     0.016      1.83%
   nanosleep              1     0.060     0.060     0.060     0.060      0.00%
   getdents64             6     0.049     0.005     0.008     0.014     16.14%
   tgkill                 1     0.026     0.026     0.026     0.026      0.00%
   write                  2     0.024     0.006     0.012     0.018     50.12%
   close                  3     0.018     0.006     0.006     0.006      4.74%
   newfstatat             1     0.015     0.015     0.015     0.015      0.00%
   openat                 1     0.014     0.014     0.014     0.014      0.00%
   epoll_ctl              2     0.013     0.005     0.006     0.008     20.08%
   getpid                 1     0.008     0.008     0.008     0.008      0.00%
   sched_yield            2     0.005     0.002     0.003     0.003     11.86%

...

```



---

# Linux性能分析60秒

| 工具                 | 检查                                                         |
| -------------------- | ------------------------------------------------------------ |
| uptime               | 平均负载可识别负载的趋势                                     |
| dmesg -T &#124; tail | 快速查看是否有内核错误                                       |
| vmstat -SM 1         | 系统级统计：运行队列长度、swap、CPU 总体使用情况             |
| mpstat -P ALL 1      | CPU 平衡情况，单个 CPU 很繁忙意味着线程扩展性糟糕            |
| pidstat 1            | 每个进程的 CPU 使用情况：识别意外的 CPU 消费者以及每个进程的 user/sys CPU 时间 |
| iostat -mx 1         | 磁盘 IO 统计                                                 |
| free -m              | 内存使用情况                                                 |
| sar -n DEV 1         | 网络 IO 统计：数据包和吞吐量等                               |
| sar -n TCP,ETCP 1    | TCP 统计：连接率，重传等                                     |
| top                  | 检查概览                                                     |





---

# 参考与感谢

-   [深入理解 iostat](http://bean-li.github.io/dive-into-iostat/)
-   性能之巅 -- 【美】Brendan Gregg















