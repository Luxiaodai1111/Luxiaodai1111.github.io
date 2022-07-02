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
