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