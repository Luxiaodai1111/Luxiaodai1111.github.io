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

想要退出，请先按下 **Ctrl-a**，然后按下 x。

```bash
$ QEMU: Terminated
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

## 实验目标

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

## 实验目标

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

每个子进程都以当前数集中最小的数字作为素数输出，并筛掉输入中该素数的所有倍数，然后将剩下的数传递给下一个子进程，最后会形成一条子进程链。

这里我使用 -1 来标识结束，最后一个子进程接收到的第一个数必然是 -1。根据实验提示，要注意关闭不用的管道。

```c
#include "kernel/types.h"
#include "kernel/stat.h"
#include "user/user.h"

#define NIL -1

int _write(int fdw, int *num) {
    int ret;
    ret = write(fdw, (void *)num, sizeof(*num));
    if (ret != sizeof(*num)){
        fprintf(2, "write failed\n");
        exit(1);
    }
    return ret;
}

void primes(int *fd) {
    int d, p;
    // 关闭写通道，子进程只需要从父进程读取数据
    close(fd[1]);
    if (read(fd[0], (void *)&d, sizeof(d)) != sizeof(d)) {
        fprintf(2, "read failed\n");
        exit(1);
    }

    // 读到的第一个数字就是哨兵，代表是最后的那个子进程
    if (d == NIL) {
        exit(0);
    }

    printf("prime %d\n", d);
    
    int input[2];
    pipe(input);
    if(fork() == 0) {
        primes(input);
    } else {
        // 父进程只需要写
        close(input[0]);
        while(read(fd[0], (void *)&p, sizeof(p)) == sizeof(p)) {
            if (p == NIL) {
                _write(input[1], &p);
                close(fd[0]);
                close(input[1]);
                break;
            }
            if (p % d != 0) {
                _write(input[1], &p);
            }
        }
        wait(0);
    }

    exit(0);
}

int
main(int argc, char *argv[])
{
    int input[2];
    int start = 2;
    int end = 35;
    
    pipe(input);
    if(fork() == 0) {
        primes(input);
    } else {
        // 父进程只需要写
        close(input[0]);
        int i;
        for(i=start; i<=end; i++) {
            _write(input[1], &i);
        }
        // 写入哨兵标记结尾
        i = NIL;
        _write(input[1], &i);
        close(input[1]);
        wait(0);
    }

    exit(0);
}
```



## 测试

```bash
root@lz-VirtualBox:~/lab01# ./grade-lab-util primes
make: “kernel/kernel”已是最新。
== Test primes == primes: OK (5.6s)
```

qemu 测试：

```bash
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
```



---

# find

## 实验目标

编写一个简单版本的UNIX find程序：在一个目录树中查找所有具有特定名称的文件。

>[!NOTE]
>
>-   查看 user/ls.c，了解如何读取目录
>-   使用递归允许 find 进入子目录
>-   不要递归 `.` 和 `..` 目录
>-   qemu 文件系统会持久化，要获得干净的文件系统，请运行 make clean，然后 make qemu
>-   You'll need to use C strings. Have a look at K&R (the C book), for example Section 5.5.
>-   不能用 == 判断字符串相等，请使用 strcmp
>-   Add the program to `UPROGS` in Makefile

实验结果应该如下：

```bash
$ make qemu
...
init: starting sh
$ echo > b
$ mkdir a
$ echo > a/b
$ find . b
./b
./a/b
$ 
```



## 程序分析

首先我们来看下 ls 怎么实现的。main 函数处理非常简单，如果没有携带参数，默认输出 `.` 下的文件，否则遍历参数输出。

```c
int
main(int argc, char *argv[])
{
  int i;

  if(argc < 2){
    ls(".");
    exit(0);
  }
  for(i=1; i<argc; i++)
    ls(argv[i]);
  exit(0);
}

```

ls 在打开和访问文件后，如果是文件，则直接输出信息，如果是目录，则要去遍历目录底下每个文件。xv6 中目录其实是一个包含一连串 dirent 结构的文件。

```bash
// Directory is a file containing a sequence of dirent structures.
#define DIRSIZ 14

struct dirent {
  ushort inum;
  char name[DIRSIZ];
};
```

所以每次从 fd 中读取 sizeof(de)，如果文件 inum == 0，跳过。否则 stat 读取文件信息并打印。

```c
void
ls(char *path)
{
  char buf[512], *p;
  int fd;
  struct dirent de;
  struct stat st;

  // 打开文件，获取文件描述符
  if((fd = open(path, 0)) < 0){
    fprintf(2, "ls: cannot open %s\n", path);
    return;
  }

  // 访问文件，并把文件信息存入 st
  if(fstat(fd, &st) < 0){
    fprintf(2, "ls: cannot stat %s\n", path);
    close(fd);
    return;
  }

  // 根据 st 信息判断是目录还是文件
  switch(st.type){
  case T_FILE:
    // 如果是文件，输出文件信息
    printf("%s %d %d %l\n", fmtname(path), st.type, st.ino, st.size);
    break;

  case T_DIR:
    // 如果是目录，则需要打印目录底下所有文件的信息
    if(strlen(path) + 1 + DIRSIZ + 1 > sizeof buf){
      printf("ls: path too long\n");
      break;
    }
    // 记录目录前缀
    strcpy(buf, path);
    p = buf+strlen(buf);
    *p++ = '/';
    // 读取目录信息
    while(read(fd, &de, sizeof(de)) == sizeof(de)){
      if(de.inum == 0)
        continue;
      // path/de.name
      memmove(p, de.name, DIRSIZ);
      p[DIRSIZ] = 0;
      if(stat(buf, &st) < 0){
        printf("ls: cannot stat %s\n", buf);
        continue;
      }
      printf("%s %d %d %d\n", fmtname(buf), st.type, st.ino, st.size);
    }
    break;
  }
  close(fd);
}
```





## 实现

这里基于 ls 来修改即可，遍历目录下每个文件，如果是文件则比较匹配，如果是目录，则递归调用 find。

```c
#include "kernel/types.h"
#include "kernel/stat.h"
#include "user/user.h"
#include "kernel/fs.h"

void
find(char *dirname, char *filename)
{
  char buf[512], *p;
  int fd;
  struct dirent de;
  struct stat st;

  if((fd = open(dirname, 0)) < 0){
    fprintf(2, "cannot open %s\n", dirname);
    return;
  }

  if(fstat(fd, &st) < 0){
    fprintf(2, "cannot stat %s\n", dirname);
    close(fd);
    return;
  }

  if (st.type != T_DIR) {
    fprintf(2, "%s's type must be dir\n", dirname);
    close(fd);
    return;
  }

  if (strlen(dirname) + 1 + DIRSIZ + 1 > sizeof buf) {
    fprintf(2, "dirname too long\n");
    close(fd);
    return;
  }

  strcpy(buf, dirname);
  p = buf+strlen(buf);
  *p++ = '/';

  // 遍历目录下每个文件并比较
  while(read(fd, &de, sizeof(de)) == sizeof(de)){
    if(de.inum == 0 || strcmp(de.name, ".") == 0 || strcmp(de.name, "..") == 0) {
      continue;
    }
    memmove(p, de.name, DIRSIZ);
    p[DIRSIZ] = 0;
    if(stat(buf, &st) < 0){
      printf("cannot stat %s\n", buf);
      continue;
    }
    switch(st.type){
      case T_FILE:
      if (strcmp(de.name, filename) == 0) {
        printf("%s\n", buf);
      }
      break;
      case T_DIR:
      // 递归查找
      find(buf, filename);
      break;
    }
  }
  close(fd);
}

int
main(int argc, char *argv[])
{
  if(argc != 3){
    fprintf(2, "Please enter a dir and a filename!\n");
    exit(1);
  }

  find(argv[1], argv[2]);
  exit(0);
}
```





## 测试

```bash
root@lz-VirtualBox:~/lab01# ./grade-lab-util find
make: “kernel/kernel”已是最新。
== Test find, in current directory == find, in current directory: OK (4.0s) 
== Test find, recursive == find, recursive: OK (4.8s)
```

qemu 测试：

```bash
$ echo > b
$ mkdir a
$ echo > a/b
$ find . b
./b
./a/b
```



---

# xargs

## 实验目标

目标：编写 xargs 工具，从标准输入读入数据，将每一行当作参数，加入到传给 xargs 的程序名和参数后面作为额外参数，然后执行。

>[!NOTE]
>
>-   使用 fork 和 exec 调用每行输入的命令。在父进程中使用 wait 等待子进程完成命令
>-   要读取单独的输入行，请一次读取一个字符，直到出现换行符(' \n ')
>-   kernel/param.h 声明了 MAXARG，如果需要声明 argv 数组，这可能会很有用
>-   Add the program to `UPROGS` in Makefile.
>-   qemu 文件系统会持久化，要获得干净的文件系统，请运行 make clean，然后 make qemu

以下示例说明 xargs 行为：

```bash
$ echo hello too | xargs echo bye
bye hello too
$
```

请注意，UNIX 上的 xargs 进行了优化，它一次将不止一个参数提供给命令。我们不期望您进行这种优化。要使 UNIX 上的 xargs 按照我们希望的方式运行，请将 -n 选项设置为 1。例如：

```bash
$ echo "1\n2" | xargs -n 1 echo line
line 1
line 2
$
```

qemu 表现应该如下：

```bash
$ make qemu
...
init: starting sh
$ sh < xargstest.sh
$ $ $ $ $ $ hello
hello
hello
$ $ 
```

输出中有许多 $ 是因为 xv6 shell 没有意识到它是从文件而不是从控制台处理命令，并为文件中的每个命令打印一个 $ 号。



## 实现

首先将 xargs 命令传入的参数保存，然后从标准输入解析输入参数，根据 `\n` 将参数划分至多行，每行参数把空格替换成 `\0`，并放在参数最后一项，利用 exec 调用函数。

```c
#include "kernel/types.h"
#include "kernel/stat.h"
#include "kernel/param.h"
#include "user/user.h"

int main(int argc, char *argv[]) {
    if (argc < 2) {
        printf("please enter more parameters!\n");
        exit(1);
    }

    int i;
    char *params[MAXARG];
    for (i=1 ; i<argc ; i++) {
        params[i-1] = argv[i];
    }

    char buf[2048];
    int j = 0;
    // 一个字节一个字节读，直到遇到换行符
    while(read(0, buf+j, 1) != 0) {
        if (buf[j] == '\n') {
            buf[j] = '\0';
            // 无内容可读
            if (j == 0) {
                break;
            }
            params[argc -1] = buf;
            if (fork() == 0) {
                exec(params[0], params);
                exit(0);
            } else {
                wait(0);
            }
            memset(buf, 0, sizeof(buf));
            j = 0;
            continue;
        }
        j++;
        // 直接将空格替换为 \0 分割开各个参数
        if (buf[j] == ' ') {
            buf[j] = '\0';
        }
        if (j == 2048) {
            printf("parameters are too long\n");
            exit(1);
        }
    }

    exit(0);
}
```



## 测试

```bash
root@lz-VirtualBox:~/lab01# ./grade-lab-util xargs
make: “kernel/kernel”已是最新。
== Test xargs == xargs: OK (6.4s)
```

qemu 测试，记得先 make clean，再 make qemu 启动：

```bash
$ sh < xargstest.sh
$ $ $ $ $ $ hello
hello
hello
$ $ 
```





---

# 挑战：修改 sh







