这个实验算是一个热身实验，帮助你熟悉 xv6 和它的一些系统调用

---

# Boot xv6

首先要搭建环境，参考 [lab tools page](https://pdos.csail.mit.edu/6.828/2021/tools.html)，安装过程很简单，就不讲述了。

接着拉取实验代码：

```bash
$ git clone git://g.csail.mit.edu/xv6-labs-2021
Cloning into 'xv6-labs-2021'...
...
$ cd xv6-labs-2021
$ git checkout util
Branch 'util' set up to track remote branch 'util' from 'origin'.
Switched to a new branch 'util'
```

运行 make qemu 测试环境是否正常：

```bash
root@lz-VirtualBox:~/lab01# make qemu
riscv64-linux-gnu-gcc    -c -o kernel/entry.o kernel/entry.S
riscv64-linux-gnu-gcc -Wall -Werror -O -fno-omit-frame-pointer -ggdb -DSOL_UTIL -DLAB_UTIL -MD -mcmodel=medany -ffreestanding -fno-common -nostdlib -mno-relax -I. -fno-stack-protector -fno-pie -no-pie  -c -o kernel/kalloc.o kernel/kalloc.c
...
riscv64-linux-gnu-objdump -S user/_zombie > user/zombie.asm
riscv64-linux-gnu-objdump -t user/_zombie | sed '1,/SYMBOL TABLE/d; s/ .* / /; /^$/d' > user/zombie.sym
mkfs/mkfs fs.img README  user/xargstest.sh user/_cat user/_echo user/_forktest user/_grep user/_init user/_kill user/_ln user/_ls user/_mkdir user/_rm user/_sh user/_stressfs user/_usertests user/_grind user/_wc user/_zombie 
nmeta 46 (boot, super, log blocks 30 inode blocks 13, bitmap blocks 1) blocks 954 total 1000
balloc: first 599 blocks have been allocated
balloc: write bitmap block at sector 45
qemu-system-riscv64 -machine virt -bios none -kernel kernel/kernel -m 128M -smp 3 -nographic -drive file=fs.img,if=none,format=raw,id=x0 -device virtio-blk-device,drive=x0,bus=virtio-mmio-bus.0

xv6 kernel is booting

hart 2 starting
hart 1 starting
init: starting sh
$ 

```

这里和 Linux 的 shell 界面就类似了，你可以敲 ls 命令来查看文件：

```bash
$ ls
.              1 1 1024
..             1 1 1024
README         2 2 2226
xargstest.sh   2 3 93
cat            2 4 23880
echo           2 5 22712
forktest       2 6 13072
grep           2 7 27240
init           2 8 23816
kill           2 9 22688
ln             2 10 22640
ls             2 11 26112
mkdir          2 12 22784
rm             2 13 22776
sh             2 14 41648
stressfs       2 15 23784
usertests      2 16 156000
grind          2 17 37960
wc             2 18 25024
zombie         2 19 22176
console        3 20 0
```

xv6 没有提供 ps 命令，你可以使用 **Ctrl-p** 来查看：

```bash
$ 
1 sleep  init
2 sleep  sh
```





---

# sleep

## 实验目标

目标：实现 sleep，能够根据用户传参睡眠指定时间。

>[!NOTE]
>
>-   开始编码前，阅读 [xv6 book](https://pdos.csail.mit.edu/6.828/2021/xv6/book-riscv-rev2.pdf) 第一章
>-   查看 user/ 中的一些其他程序例如 user/echo.c、user/grep.c 和 user/rm.c，了解如何获得传递给程序的命令行参数。
>-   如果用户忘记传递参数，sleep 要显示错误消息。
>-   命令行参数作为字符串传递；可以使用 atoi 将其转换为整数（参见user/ulib.c）。
>-   使用 sleep 系统调用
>-   查看 kernel/sysproc.c 有关实现 sleep 系统调用的 xv6 内核代码（搜索 sys_sleep），user/user.h 是有关可从用户程序调用的 sleep 的 C 定义，user/usys.S 是从用户代码跳转到内核休眠的汇编代码。
>-   确保 main 调用 exit() 退出程序
>-   在 Makefile 的 UPROGS 部分添加 sleep，这样 make qemu 之后会帮你自动编译进系统
>-   学习 C 语言

实现完成后你可以运行：

```bash
$ make qemu
...
init: starting sh
$ sleep 10
(nothing happens for a little while)
$
```

如果你的程序如上所示运行时暂停就是正确的。运行 make grade，看看你是否真的通过了 sleep 测试。

请注意，make grade 运行所有测试，包括下面作业的测试。如果您想要运行一个作业的等级测试，请键入：

```bash
$ ./grade-lab-util sleep
or
$ make GRADEFLAGS=sleep grade
```



## 程序分析

我们首先来看一下 user/echo.c，argc 是传参数目，第一个参数一般是命令本身，因此从 1 开始索引。echo 的功能就是把后面跟的参数全部打印一遍，因此循环遍历每个参数并打印。在 Unix 系统里面，默认情况下 `0` 代表 `stdin`，`1` 代表 `stdout`，`2` 代表 `stderr`。这 3 个文件描述符在进程创建时就已经打开了的（从父进程复制过来的），可以直接使用。所以程序就是往标准输出打印传参，如果遍历完毕了，则打印 `\n`。

```c
#include "kernel/types.h"
#include "kernel/stat.h"
#include "user/user.h"

int
main(int argc, char *argv[])
{
  int i;

  for(i = 1; i < argc; i++){
    write(1, argv[i], strlen(argv[i]));
    if(i + 1 < argc){
      write(1, " ", 1);
    } else {
      write(1, "\n", 1);
    }
  }
  exit(0);
}
```

我们可以执行看下 echo 命令的效果：

```bash
$ echo hello world
hello world
```

下面我们再来看下 rm 的实现，其实这个程序就和我们要实现的 sleep 比较像了，先判断了参数，然后调用 unlink 系统调用去删除。

```c
#include "kernel/types.h"
#include "kernel/stat.h"
#include "user/user.h"

int
main(int argc, char *argv[])
{
  int i;

  if(argc < 2){
    fprintf(2, "Usage: rm files...\n");
    exit(1);
  }

  for(i = 1; i < argc; i++){
    if(unlink(argv[i]) < 0){
      fprintf(2, "rm: %s failed to delete\n", argv[i]);
      break;
    }
  }

  exit(0);
}
```



## 实现

user/sleep.c

```c
#include "kernel/types.h"
#include "kernel/stat.h"
#include "user/user.h"

int
main(int argc, char *argv[])
{
  if(argc != 2){
    fprintf(2, "Usage: sleep <ticks>\n");
    exit(1);
  }

  sleep(atoi(argv[1]));

  exit(0);
}
```

记得在 Makefile 中添加测试项：

```makefile
UPROGS=\
        $U/_cat\
        $U/_echo\
        $U/_forktest\
        $U/_grep\
        $U/_init\
        $U/_kill\
        $U/_ln\
        $U/_ls\
        $U/_mkdir\
        $U/_rm\
        $U/_sh\
        $U/_stressfs\
        $U/_usertests\
        $U/_grind\
        $U/_wc\
        $U/_zombie\
        $U/_sleep\

```



## 测试

```bash
root@lz-VirtualBox:~/lab01# ./grade-lab-util sleep
make: “kernel/kernel”已是最新。
== Test sleep, no arguments == sleep, no arguments: OK (3.7s) 
== Test sleep, returns == sleep, returns: OK (3.5s) 
== Test sleep, makes syscall == sleep, makes syscall: OK (3.3s) 
```





---

# pingpong

# 实验目标

目标：编写一个程序，使用 UNIX 系统调用通过一对管道在两个进程之间传输一个字节。父进程应该发送一个字节给子进程；子进程应该打印 `"<pid>: received ping"`，其中 < pid > 是其进程 id，将管道上的字节写入父进程，然后退出；父进程应该从子进程读取该字节，打印 `"<pid>: received pong"`，然后退出。

>[!NOTE]
>
>-   使用 pipe 创建管道
>-   使用 fork 创建子进程
>-   使用 read 从管道中读取数据，使用 write 向管道写入数据
>-   使用 getpid 获取进程 PID
>-   Makefile 中 UPROGS 记得添加条目
>-   xv6 上的用户程序只能使用有限的库函数。可以在 user/user.h 中看到列表；源代码（系统调用除外）位于 user/ulib.c、user/printf.c 和 user/umalloc.c中。

从 xv6 shell 运行该程序，它应该产生以下输出：

```bash
$ make qemu
...
init: starting sh
$ pingpong
4: received ping
3: received pong
$
```



## 实现

pipe 的输入为长度为 2 的 int 数组 p，其中 p[0] 对应输入文件描述符，p[1] 对应输出文件描述符。

```c
#include "kernel/types.h"
#include "kernel/stat.h"
#include "user/user.h"

int
main(int argc, char *argv[])
{
  int p2c[2], c2p[2];
  if (pipe(p2c) < 0) {
    printf("pipe error");
    exit(-1);
  }
  if (pipe(c2p) < 0) {
    printf("pipe error");
    exit(-1);
  }

  int pid = fork();
  if (pid == 0) {
    // 子进程，关闭p2c写通道，并在从父进程读取数据后关闭读通道
    char buf[2];
    close(p2c[1]);
    if (read(p2c[0], &buf, 1) != 1) {
      printf("can't read from parent\n");
      exit(-1);
    }
    printf("child receive: %c\n", buf[0]);
    close(p2c[0]);
    printf("%d: received ping\n", getpid());

    // 关闭c2p读通道，并在向c2p写通道写完数据后关闭写通道
    close(c2p[0]);
    if (write(c2p[1], "C", 1) != 1) {
      printf("can't write to parent\n");
      exit(-1);
    }
    close(c2p[1]);
    exit(0);
  } else {
    // 父进程，关闭p2c读通道，并在向p2c写通道写完数据后关闭写通道
    close(p2c[0]);
    if (write(p2c[1], "P", 1) != 1) {
      printf("can't write to child\n");
      exit(-1);
    }
    close(p2c[1]);

    char buf[2];
    // 关闭c2p写通道，等待从子进程读出数据后，关闭读通道
    close(c2p[1]);
    if (read(c2p[0], &buf, 1) != 1) {
      printf("can't read from parent\n");
      exit(-1);
    }
    printf("parent receive: %c\n", buf[0]);
    printf("%d: received pong\n", getpid());
    close(c2p[0]);

    wait(0);
    exit(0);
  }
}
```





## 测试

```bash
root@lz-VirtualBox:~/lab01# ./grade-lab-util pingpong
make: “kernel/kernel”已是最新。
== Test pingpong == pingpong: OK (2.1s)
```

qemu 测试：

```bash
$ pingpong
child receive: P
4: received ping
parent receive: C
3: received pong
```



---

# primes

# 实验目标

目标：利用 pipe 和 fork 来统计素数，由于 xv6 文件描述符数目有限，统计 35 以内即可。

>[!NOTE]
>
>-   注意关闭一个进程不需要的文件描述符，因为否则你的程序会在第一个进程达到 35 之前耗尽 xv6 的资源
>-   一旦第一个进程达到 35，它应该等待，直到整个管道终止，包括所有子进程、孙进程等等。因此，主 prime 进程应该只在所有输出都已打印并且所有其他 prime 进程都已退出之后才退出
>-   当管道的写端关闭时，read返回零
>-   最简单的方法是直接将 32 位（4 字节）int 写入管道，而不是使用格式化的 ASCII I/O
>-   You should create the processes in the pipeline only as they are needed.
>-   Add the program to `UPROGS` in Makefile.

实现结果应该如下：

```bash
$ make qemu
...
init: starting sh
$ primes
prime 2
prime 3
prime 5
prime 7
prime 11
prime 13
prime 17
prime 19
prime 23
prime 29
prime 31
$
```



## 实现





## 测试







---

# find







---

# xargs





---

# 挑战：修改 sh







