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
