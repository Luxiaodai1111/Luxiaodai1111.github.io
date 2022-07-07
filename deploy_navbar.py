#!/usr/bin/env python
# _*_ coding:utf-8 _*_
# 将当前目录下的 __navbar.md 拷贝到 posts 里以及其各子目录里

import os
import time


root = "."
posts = "posts"
__navbar = "_navbar.md"


def main():
    abs_root = os.path.abspath(root)
    source = os.path.join(abs_root, __navbar)
    target_root = os.path.join(abs_root, posts)
    cmd = 'copy "{}" "{}"'.format(source, os.path.join(target_root, __navbar))
    print(cmd)
    os.system(cmd)
    for dir in os.listdir(target_root):
        os.chdir(abs_root)
        if os.path.isdir(os.path.join(target_root, dir)):
            cmd = 'copy "{}" "{}"'.format(source, os.path.join(target_root, dir, __navbar))
            print(cmd)
            os.system(cmd)
            # time.sleep(10)
            for filename in os.listdir(os.path.join(target_root, dir)):
                if filename == "deploy_navbar.py":
                    os.chdir(os.path.join(target_root, dir))
                    cmd = "python deploy_navbar.py"
                    os.system(cmd)
                    # time.sleep(10)


if __name__ == "__main__":
    main()
