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
KiB Mem : 16201188 total,   157880 free,   970568 used, 15072740 buff/cache
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
%Cpu0  :   9.2/34.2   43[|||||||||||||||||||||||||||||||||||||||||||                                                         ]
%Cpu1  :  18.4/27.0   45[|||||||||||||||||||||||||||||||||||||||||||||                                                       ]
%Cpu2  :  17.7/28.3   46[||||||||||||||||||||||||||||||||||||||||||||||                                                      ]
%Cpu3  :  17.4/28.9   46[||||||||||||||||||||||||||||||||||||||||||||||                                                      ]
%Cpu4  :  10.8/22.0   33[|||||||||||||||||||||||||||||||||                                                                   ]
%Cpu5  :  14.0/24.6   39[|||||||||||||||||||||||||||||||||||||||                                                             ]
%Cpu6  :  14.6/23.7   38[|||||||||||||||||||||||||||||||||||||||                                                             ]
%Cpu7  :  13.2/27.4   41[||||||||||||||||||||||||||||||||||||||||                                                            ]
```

第四行和第五行是内存使用情况，可以按下 `m` 来切换显示模式。

```bash
# 按下 m 切换显示模式
KiB Mem :  5.2/16201188 [|||||                                                                                               ]
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
KiB Mem : 16201188 total, 15194200 free,   512732 used,   494256 buff/cache
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
-   ...

```bash
 Active / Total Objects (% used)    : 1903270 / 2219372 (85.8%)
 Active / Total Slabs (% used)      : 59381 / 59381 (100.0%)
 Active / Total Caches (% used)     : 66 / 96 (68.8%)
 Active / Total Size (% used)       : 382311.58K / 462753.92K (82.6%)
 Minimum / Average / Maximum Object : 0.01K / 0.21K / 12.75K

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











---

# 网络

## sar



## IP冲突检查



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

















