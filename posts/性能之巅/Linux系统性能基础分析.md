本文不介绍工具安装，如果缺失，在 CentOS 上基本都可以使用 yum install 方式安装。

---

# CPU

## top

top 的原理很简单，就是读取 /proc 下的信息并展示，所以有时候你会看到 top 本身也占用了不少资源，是因为它也要去操作文件。





---

# 内存







---

# 磁盘









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



---

# 网络







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

















