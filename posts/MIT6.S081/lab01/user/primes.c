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