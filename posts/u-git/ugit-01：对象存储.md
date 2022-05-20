此系列是 [DIY Git in Python](https://www.leshenko.net/p/ugit/#) 的学习笔记。在学习这个系列之前可以去 [Learn Git Branching](https://learngitbranching.js.org/?locale=zh_CN) 熟悉 Git 基本操作。



用 Git 提高效率的更好方法是了解它在幕后是如何工作的，而不是学习更“高级”的 Git 命令。

ugit 是一个类似 git 的版本控制系统的小型实现。它的首要目标是简单和教育价值。此系列以小的增量步骤慢慢实现 ugit，每个步骤都有详细的解释。

ugit 不完全是 git，但它拥有 Git 的重要思想。相比 git，ugit 要简洁得多，它不会实现不相关的功能。例如，为了降低 ugit 的复杂性，ugit 不压缩对象，不保存文件的模式，或者不保存提交的时间。但是重要的思想比如提交、分支、索引、合并和远程都存在，并且与 Git 非常相似。如果你了解 ugit，你将能够在 git 中看到相同的想法。

本项目代码基于 python3，下面我们开始吧！



---

# 添加参数解析

真正的 Git 可执行文件有多个子命令，比如 git init、git commit 等。我们使用 Python 的内置参数解析器 argparse 来实现子命令。

- setup.py

```python
#!/usr/bin/env python3

from setuptools import setup

setup (name = 'ugit',
       version = '1.0',
       packages = ['ugit'],
       entry_points = {
           'console_scripts' : [
               'ugit = ugit.cli:main'
           ]
       })
```

- ugit/cli.py

```python
import argparse


def main():
    args = parse_args()
    args.func(args)


def parse_args():
    parser = argparse.ArgumentParser()

    commands = parser.add_subparsers(dest="command")
    commands.required = True

    init_parser = commands.add_parser("init")
    init_parser.set_defaults(func=init)

    return parser.parse_args()


def init(args):
    print("hello world")


if __name__ == "__main__":
    main()

```



安装后执行，可以看到顺利输出了 hello world。

```bash
[root@localhost ~]# ls ugit
cli.py  __init__.py
[root@localhost ~]# python3 setup.py build
running build
running build_py
copying ugit/__init__.py -> build/lib/ugit
[root@localhost ~]# python3 setup.py install
running install
running bdist_egg
running egg_info
writing ugit.egg-info/PKG-INFO
writing dependency_links to ugit.egg-info/dependency_links.txt
writing entry points to ugit.egg-info/entry_points.txt
writing top-level names to ugit.egg-info/top_level.txt
reading manifest file 'ugit.egg-info/SOURCES.txt'
writing manifest file 'ugit.egg-info/SOURCES.txt'
installing library code to build/bdist.linux-x86_64/egg
running install_lib
running build_py
creating build/bdist.linux-x86_64/egg
creating build/bdist.linux-x86_64/egg/ugit
copying build/lib/ugit/cli.py -> build/bdist.linux-x86_64/egg/ugit
copying build/lib/ugit/__init__.py -> build/bdist.linux-x86_64/egg/ugit
byte-compiling build/bdist.linux-x86_64/egg/ugit/cli.py to cli.cpython-36.pyc
byte-compiling build/bdist.linux-x86_64/egg/ugit/__init__.py to __init__.cpython-36.pyc
creating build/bdist.linux-x86_64/egg/EGG-INFO
copying ugit.egg-info/PKG-INFO -> build/bdist.linux-x86_64/egg/EGG-INFO
copying ugit.egg-info/SOURCES.txt -> build/bdist.linux-x86_64/egg/EGG-INFO
copying ugit.egg-info/dependency_links.txt -> build/bdist.linux-x86_64/egg/EGG-INFO
copying ugit.egg-info/entry_points.txt -> build/bdist.linux-x86_64/egg/EGG-INFO
copying ugit.egg-info/top_level.txt -> build/bdist.linux-x86_64/egg/EGG-INFO
zip_safe flag not set; analyzing archive contents...
creating 'dist/ugit-1.0-py3.6.egg' and adding 'build/bdist.linux-x86_64/egg' to it
removing 'build/bdist.linux-x86_64/egg' (and everything under it)
Processing ugit-1.0-py3.6.egg
Removing /usr/local/lib/python3.6/site-packages/ugit-1.0-py3.6.egg
Copying ugit-1.0-py3.6.egg to /usr/local/lib/python3.6/site-packages
ugit 1.0 is already the active version in easy-install.pth
Installing ugit script to /usr/local/bin

Installed /usr/local/lib/python3.6/site-packages/ugit-1.0-py3.6.egg
Processing dependencies for ugit==1.0
Finished processing dependencies for ugit==1.0

[root@localhost ~]# ugit init
hello world
```



---

# init:创建工作目录

上面我们初始化只是打印了一句话，下面我们修改 ugit init 命令来创建一个新的空存储库。

Git 将所有存储库数据本地存储在名为 `.git` 的子目录中，因此在初始化时我们将创建一个目录命名为 `.ugit`，这样它就不会与 Git 发生冲突，但想法是一样的。

为了实现 init，我们可以从 cli.py 中调用 os.makedirs，但我想在代码的不同逻辑部分之间进行一些分离：

- cli.py - 负责解析和处理用户输入。
- data.py - 管理 .ugit 目录中的数据。 这里是实际接触磁盘上文件的代码。

代码差异如下：

```diff
diff -u ugit.bak/cli.py ugit/cli.py
--- ugit.bak/cli.py     2022-01-29 19:16:11.201111643 +0800
+++ ugit/cli.py 2022-01-29 19:17:58.146699057 +0800
@@ -1,4 +1,7 @@
 import argparse
+import os
+
+from . import data
 
 
 def main():
@@ -19,9 +22,9 @@
 
 
 def init(args):
-    print("hello world")
+    data.init()
+    print("Initialized empty ugit repository in {}/{}".format(os.getcwd(), data.GIT_DIR))


===================================================
只在 ugit 存在：data.py
```

完整代码如下：

- ugit/cli.py

```python
import argparse
import os

from . import data


def main():
    args = parse_args()
    args.func(args)


def parse_args():
    parser = argparse.ArgumentParser()

    commands = parser.add_subparsers(dest="command")
    commands.required = True

    init_parser = commands.add_parser("init")
    init_parser.set_defaults(func=init)

    return parser.parse_args()


def init(args):
    data.init()
    print("Initialized empty ugit repository in {}/{}".format(os.getcwd(), data.GIT_DIR))


if __name__ == "__main__":
    main()

```

- ugit/data.py

```python
import os

GIT_DIR = ".ugit"


def init():
    os.makedirs(GIT_DIR)

```



我们建立一个 repo 文件夹用来测试命令，安装后执行，可以看到建立了一个 .ugit 仓库。

```shell
[root@localhost ~]# mkdir repo
[root@localhost ~]# cd repo/
[root@localhost repo]# ls
[root@localhost repo]# ugit init
Initialized empty ugit repository in /root/repo/.ugit
[root@localhost repo]# ll .
./     ../    .ugit/ 
```



---

# hash-object:保存对象

接下来让我们创建第一个重要的命令。此命令将获取一个文件并将其存储在我们的 .ugit 目录中以供以后检索。在 Git 的行话中，此功能称为 **the object database**。它允许我们存储和检索任意 blob，它们被称为对象。就对象数据库而言，对象的内容没有任何意义（就像文件系统不关心文件的内部结构一样）。

因为此命令需要 .ugit 目录，所以它必须从您执行 ugit init 的同一目录运行。

我们可以存储一个对象，但是我们以后如何引用它呢？

我们可以要求用户提供一个名称和对象，然后使用名称检索对象，但有一个更好的方法：我们可以使用它的散列引用对象。

```shell
[root@localhost repo]# echo -n this is cool | sha1sum
60f51187e76a9de0ff3df31f051bde04da2da891  -
[root@localhost repo]# echo -n this is cooler | sha1sum
f3c953b792f9ab39d1be0bdab7ab5f8350593004 
```

您可以看到对短语 `this is cool` 和 `this is cooler` 进行哈希处理会给出完全不同的哈希值，即使短语之间的差异很小。

我们将使用散列作为对象的名称（我们将此名称称为 OID - Object  ID）。

所以命令 `hash-object` 的流程是：

- 获取要存储的文件的路径。
- 读取文件。
- 使用 SHA-1 散列文件的内容。
- 将文件存储在 `.ugit/objects/{the SHA-1 hash}` 下。

这种类型的存储称为**内容可寻址**存储，因为我们用来查找 blob 的地址基于 blob 本身的内容。与名称可寻址存储相反，例如典型的文件系统，您可以通过名称寻址特定文件，而不管其内容如何。

在不同计算机之间同步数据时，内容可寻址存储具有很好的特性——如果两个存储库有一个具有相同 OID 的对象，我们可以确定它们是同一个对象（与此相对的是，以名称寻址的方法无法保证是同一个文件）。此外，由于实际上可以保证两个不同的对象具有不同的 OID，因此我们在对象之间不会发生命名冲突。

当真正的 Git 存储对象时，它会做一些额外的事情，比如将对象的大小也写入文件，压缩它们并将对象分成 256 个目录等等，这样做是为了避免目录中包含大量文件从而影响性能。为简单起见，我们不会在 ugit 中实现这些东西。



---

# cat-file:打印散列对象

这个命令与 hash-object 相反：它可以通过它的 OID 打印一个对象。 它的实现是读取 `.ugit/objects/{OID}` 处的文件。

名称 hash-object 和 cat-file 并不是最清楚的名称，但它们是 Git 使用的名称，因此我们将坚持使用它们以保持一致性。

修改代码如下：

```diff
diff -u ugit.bak/cli.py ugit/cli.py
--- ugit.bak/cli.py     2022-01-29 19:31:58.540323478 +0800
+++ ugit/cli.py 2022-01-29 23:23:45.660588118 +0800
@@ -1,5 +1,6 @@
 import argparse
 import os
+import sys
 
 from . import data
 
@@ -18,6 +19,13 @@
     init_parser = commands.add_parser("init")
     init_parser.set_defaults(func=init)
 
+    hash_object_parser = commands.add_parser('hash-object')
+    hash_object_parser.set_defaults(func=hash_object)
+    hash_object_parser.add_argument('file')
+
+    cat_file_parser = commands.add_parser('cat-file')
+    cat_file_parser.set_defaults(func=cat_file)
+    cat_file_parser.add_argument('object')
     return parser.parse_args()
 
 
@@ -26,5 +34,16 @@
     print("Initialized empty ugit repository in {}/{}".format(os.getcwd(), data.GIT_DIR))
 
 
+def hash_object(args):
+    with open(args.file, 'rb') as f:
+        print(data.hash_object(f.read()))
+
+
+def cat_file(args):
+    sys.stdout.flush()
+    sys.stdout.buffer.write(data.get_object(args.object))
+
+


===========================================================
diff -u ugit.bak/data.py ugit/data.py
--- ugit.bak/data.py    2022-01-29 19:31:58.541406811 +0800
+++ ugit/data.py        2022-01-29 23:16:01.766460852 +0800
@@ -1,8 +1,23 @@
 import os
+import hashlib
 
 GIT_DIR = ".ugit"
+OBJECTS_DIR = GIT_DIR + "/objects"
 
 
 def init():
     os.makedirs(GIT_DIR)
+    os.makedirs(OBJECTS_DIR)
+
+
+def hash_object(data):
+    oid = hashlib.sha1(data).hexdigest()
+    with open("{}/{}".format(OBJECTS_DIR, oid), "wb") as f:
+        f.write(data)
+    return oid
+
+
+def get_object(oid):
+    with open("{}/{}".format(OBJECTS_DIR, oid), 'rb') as f:
+        return f.read()
```

修改文件完整版如下：

- ugit/cli.py

```python
import argparse
import os
import sys

from . import data


def main():
    args = parse_args()
    args.func(args)


def parse_args():
    parser = argparse.ArgumentParser()

    commands = parser.add_subparsers(dest="command")
    commands.required = True

    init_parser = commands.add_parser("init")
    init_parser.set_defaults(func=init)

    hash_object_parser = commands.add_parser('hash-object')
    hash_object_parser.set_defaults(func=hash_object)
    hash_object_parser.add_argument('file')

    cat_file_parser = commands.add_parser('cat-file')
    cat_file_parser.set_defaults(func=cat_file)
    cat_file_parser.add_argument('object')
    return parser.parse_args()


def init(args):
    data.init()
    print("Initialized empty ugit repository in {}/{}".format(os.getcwd(), data.GIT_DIR))


def hash_object(args):
    with open(args.file, 'rb') as f:
        print(data.hash_object(f.read()))


def cat_file(args):
    sys.stdout.flush()
    sys.stdout.buffer.write(data.get_object(args.object))


if __name__ == "__main__":
    main()

```

- ugit/data.py

```python
import os
import hashlib

GIT_DIR = ".ugit"
OBJECTS_DIR = GIT_DIR + "/objects"


def init():
    os.makedirs(GIT_DIR)
    os.makedirs(OBJECTS_DIR)


def hash_object(data):
    oid = hashlib.sha1(data).hexdigest()
    with open("{}/{}".format(OBJECTS_DIR, oid), "wb") as f:
        f.write(data)
    return oid


def get_object(oid):
    with open("{}/{}".format(OBJECTS_DIR, oid), 'rb') as f:
        return f.read()

```



因为 init 有修改，我们重新建立 repo 目录后运行：

```shell
[root@localhost repo]# ugit init
Initialized empty ugit repository in /root/repo/.ugit
[root@localhost repo]# echo "hello world" > testfile
[root@localhost repo]# ugit hash-object testfile
22596363b3de40b06f981fb85d82312e8c0ed511
[root@localhost repo]# ugit cat-file 22596363b3de40b06f981fb85d82312e8c0ed511
hello world
```

请注意，文件名 testfile 并未作为此过程的一部分保留，对象数据库只是存储字节以供以后检索，它并不关心字节来自哪个文件名。



---

# 添加对象类型

我们很快就会看到，将在不同的上下文中使用不同的逻辑类型的对象（即使从对象数据库的角度来看，它们都只是字节）。

为了降低在错误上下文中使用对象的机会，我们将为每个对象添加一个类型标签。

该类型只是一个字符串，将被添加到文件的开头，后跟一个空字节。 稍后读取文件时，我们将提取类型并验证它是否为预期的类型。

默认类型将是 blob，因为默认情况下，对象是一个没有进一步语义含义的字节集合。

如果我们不想验证类型，我们也可以将 expected=None 传递给 get_object()。 这对于 cat-file 命令很有用，该命令是用于打印所有对象的调试命令。

修改代码如下：

```diff
diff -u ugit.bak/cli.py ugit/cli.py
--- ugit.bak/cli.py     2022-01-29 23:33:21.393433118 +0800
+++ ugit/cli.py 2022-01-29 23:40:09.438631435 +0800
@@ -41,7 +41,7 @@
 
 def cat_file(args):
     sys.stdout.flush()
-    sys.stdout.buffer.write(data.get_object(args.object))
+    sys.stdout.buffer.write(data.get_object(args.object, expected=None))

 
 
===========================================================
diff -u ugit.bak/data.py ugit/data.py
--- ugit.bak/data.py    2022-01-29 23:33:21.393433118 +0800
+++ ugit/data.py        2022-01-29 23:40:25.220599153 +0800
@@ -10,14 +10,23 @@
     os.makedirs(OBJECTS_DIR)
 
 
-def hash_object(data):
-    oid = hashlib.sha1(data).hexdigest()
+def hash_object(data, type_="blob"):
+    obj = type_.encode() + b"\x00" + data
+    oid = hashlib.sha1(obj).hexdigest()
     with open("{}/{}".format(OBJECTS_DIR, oid), "wb") as f:
-        f.write(data)
+        f.write(obj)
     return oid
 
 
-def get_object(oid):
+def get_object(oid, expected="blob"):
     with open("{}/{}".format(OBJECTS_DIR, oid), 'rb') as f:
-        return f.read()
+        obj = f.read()
+
+    type_, _, content = obj.partition(b'\x00')
+    type_ = type_.decode()
+
+    if expected is not None:
+        assert type_ == expected, "Expected {}, got {}".format(expected, type_)
+        
+    return content
```

下面是修改的文件完整代码：

- ugit/cli.py

```python
import argparse
import os
import sys

from . import data


def main():
    args = parse_args()
    args.func(args)


def parse_args():
    parser = argparse.ArgumentParser()

    commands = parser.add_subparsers(dest="command")
    commands.required = True

    init_parser = commands.add_parser("init")
    init_parser.set_defaults(func=init)

    hash_object_parser = commands.add_parser('hash-object')
    hash_object_parser.set_defaults(func=hash_object)
    hash_object_parser.add_argument('file')

    cat_file_parser = commands.add_parser('cat-file')
    cat_file_parser.set_defaults(func=cat_file)
    cat_file_parser.add_argument('object')
    return parser.parse_args()


def init(args):
    data.init()
    print("Initialized empty ugit repository in {}/{}".format(os.getcwd(), data.GIT_DIR))


def hash_object(args):
    with open(args.file, 'rb') as f:
        print(data.hash_object(f.read()))


def cat_file(args):
    sys.stdout.flush()
    sys.stdout.buffer.write(data.get_object(args.object, expected=None))


if __name__ == "__main__":
    main()

```

- ugit/data.py

```python
import os
import hashlib

GIT_DIR = ".ugit"
OBJECTS_DIR = GIT_DIR + "/objects"


def init():
    os.makedirs(GIT_DIR)
    os.makedirs(OBJECTS_DIR)


def hash_object(data, type_="blob"):
    obj = type_.encode() + b"\x00" + data
    oid = hashlib.sha1(obj).hexdigest()
    with open("{}/{}".format(OBJECTS_DIR, oid), "wb") as f:
        f.write(obj)
    return oid


def get_object(oid, expected="blob"):
    with open("{}/{}".format(OBJECTS_DIR, oid), 'rb') as f:
        obj = f.read()

    type_, _, content = obj.partition(b'\x00')
    type_ = type_.decode()

    if expected is not None:
        assert type_ == expected, "Expected {}, got {}".format(expected, type_)
        
    return content

```

我们测试一下：

```shell
[root@localhost repo]# ugit init
Initialized empty ugit repository in /root/repo/.ugit
[root@localhost repo]# ugit hash-object testfile 
68adab4faf9e8a4ca41daa16e22a6be9b1a78f13
[root@localhost repo]# ugit cat-file 68adab4faf9e8a4ca41daa16e22a6be9b1a78f13
hello world
[root@localhost repo]# cat .ugit/objects/68adab4faf9e8a4ca41daa16e22a6be9b1a78f13 
blobhello world
```

可以看到 hash-object 和 cat-file 命令输出没有变化，但是真实存储对象的内容开头加上了类型。



---

# 添加base模块

该模块将具有 ugit 的基本高级逻辑。 它将使用在 data.py 中实现的对象数据库来实现更高级别的逻辑。

- ugit/cli.py

```diff
diff -u ugit.bak/cli.py ugit/cli.py
--- ugit.bak/cli.py     2022-01-29 23:53:30.091827155 +0800
+++ ugit/cli.py 2022-01-29 23:54:13.722987906 +0800
@@ -2,6 +2,7 @@
 import os
 import sys
 
+from . import base
 from . import data

```

- ugit/base.py

```python
from . import data
```





---

# write-tree

## 列出文件

下一个命令是 write-tree。此命令将遍历当前工作目录并将所有内容存储到对象数据库中。如果 hash-object 用于存储单个文件，那么 write-tree 用于存储整个目录。

与 hash-object 一样，write-tree 将在完成后给我们一个 OID，我们将能够使用 OID 以便稍后检索目录。

在 Git 的行话中，`tree` 表示目录，我们将在以后的更改中详细介绍。在此更改中，我们将只准备围绕以下功能的代码：

- 创建 write-tree 命令

- 在基本模块中创建一个 write_tree() 函数。为什么在基本模块中而不是在数据模块中？因为 write_tree() 不会直接写入磁盘，而是使用 data.py 提供的对象数据库来存储目录。因此它属于更高级别的模块。

- write_tree() 递归打印目录（现在没有实际存储对象，我们只是编写了模板来递归扫描目录）。

下面是更改的代码：

```diff
diff -u ugit.bak/base.py ugit/base.py
--- ugit.bak/base.py    2022-01-30 04:21:49.250812712 +0800
+++ ugit/base.py        2022-01-30 04:31:54.651902739 +0800
@@ -1 +1,16 @@
+import os
+
 from . import data
+
+
+def write_tree(directory="."):
+    with os.scandir(directory) as it:
+        for entry in it:
+            full = "{}/{}".format(directory, entry.name)
+            if entry.is_file(follow_symlinks=False):
+                # TODO write the file to object store
+                print(full)
+            elif entry.is_dir(follow_symlinks=False):
+                write_tree(full)
+
+    # TODO actually create the tree object


===========================================================
diff -u ugit.bak/cli.py ugit/cli.py
--- ugit.bak/cli.py     2022-01-30 04:21:49.250812712 +0800
+++ ugit/cli.py 2022-01-30 04:33:20.836313914 +0800
@@ -27,6 +27,10 @@
     cat_file_parser = commands.add_parser('cat-file')
     cat_file_parser.set_defaults(func=cat_file)
     cat_file_parser.add_argument('object')
+
+    write_tree_parser = commands.add_parser('write-tree')
+    write_tree_parser.set_defaults(func=write_tree)
+
     return parser.parse_args()
 
 
@@ -45,6 +49,10 @@
     sys.stdout.buffer.write(data.get_object(args.object, expected=None))
 
 
+def write_tree(args):
+    base.write_tree()
+
+
 if __name__ == "__main__":
     main()
```

以下是修改代码的完整版：

- ugit/base.py

```python
import os

from . import data


def write_tree(directory="."):
    with os.scandir(directory) as it:
        for entry in it:
            full = "{}/{}".format(directory, entry.name)
            if entry.is_file(follow_symlinks=False):
                # TODO write the file to object store
                print(full)
            elif entry.is_dir(follow_symlinks=False):
                write_tree(full)

    # TODO actually create the tree object

```

- ugit/cli.py

```python
import argparse
import os
import sys

from . import base
from . import data


def main():
    args = parse_args()
    args.func(args)


def parse_args():
    parser = argparse.ArgumentParser()

    commands = parser.add_subparsers(dest="command")
    commands.required = True

    init_parser = commands.add_parser("init")
    init_parser.set_defaults(func=init)

    hash_object_parser = commands.add_parser('hash-object')
    hash_object_parser.set_defaults(func=hash_object)
    hash_object_parser.add_argument('file')

    cat_file_parser = commands.add_parser('cat-file')
    cat_file_parser.set_defaults(func=cat_file)
    cat_file_parser.add_argument('object')

    write_tree_parser = commands.add_parser('write-tree')
    write_tree_parser.set_defaults(func=write_tree)

    return parser.parse_args()


def init(args):
    data.init()
    print("Initialized empty ugit repository in {}/{}".format(os.getcwd(), data.GIT_DIR))


def hash_object(args):
    with open(args.file, 'rb') as f:
        print(data.hash_object(f.read()))


def cat_file(args):
    sys.stdout.flush()
    sys.stdout.buffer.write(data.get_object(args.object, expected=None))


def write_tree(args):
    base.write_tree()


if __name__ == "__main__":
    main()

```



示例：

```shell
[root@localhost repo]# ugit write-tree
./testfile
./.ugit/objects/68adab4faf9e8a4ca41daa16e22a6be9b1a78f13
```



## 忽略 .ugit 文件

如果我们运行 ugit write-tree，我们会看到它还会打印 .ugit 目录的内容。 这个目录不是用户文件的一部分，所以让我们忽略它。

实际上，我创建了一个单独的 is_ignored() 函数。 这样，如果我们有任何其他要忽略的文件，我们就可以只更改一个地方。

- ugit/base.py

```diff
diff -u ugit.bak/base.py ugit/base.py
--- ugit.bak/base.py    2022-01-30 04:38:42.813186368 +0800
+++ ugit/base.py        2022-01-30 04:41:58.202805596 +0800
@@ -7,6 +7,9 @@
     with os.scandir(directory) as it:
         for entry in it:
             full = "{}/{}".format(directory, entry.name)
+            if is_ignored(full):
+                continue
+
             if entry.is_file(follow_symlinks=False):
                 # TODO write the file to object store
                 print(full)
@@ -14,3 +17,8 @@
                 write_tree(full)
 
     # TODO actually create the tree object
+
+
+def is_ignored(path):
+    return '.ugit' in path.split("/")
+
```



示例：

```shell
[root@localhost repo]# ugit write-tree
./testfile
```



## 散列文件

接下来让我们将所有文件放入对象数据库中，而不是只打印文件名。 写入时我们打印他们的 OID 和他们的名字。

请注意，我们现在只为文件获得一个单独的 OID，而没有用一个 OID 来表示目录。 另外，请注意文件的名称不存储在对象数据库中，它们只是打印出来然后信息被丢弃。

所以在这个阶段 write-tree 只是将一堆文件保存为 blob，但下一个更改将优化它。

- ugit/base.py

```diff
diff -u ugit.bak/base.py ugit/base.py
--- ugit.bak/base.py    2022-01-30 04:50:07.161108628 +0800
+++ ugit/base.py        2022-01-30 04:51:49.343094992 +0800
@@ -11,8 +11,8 @@
                 continue
 
             if entry.is_file(follow_symlinks=False):
-                # TODO write the file to object store
-                print(full)
+                with open(full, 'rb') as f:
+                    print(data.hash_object(f.read()), full)
             elif entry.is_dir(follow_symlinks=False):
                 write_tree(full)
```

示例：

```shell
[root@localhost repo]# ugit write-tree
68adab4faf9e8a4ca41daa16e22a6be9b1a78f13 ./testfile
```



## 写树对象

现在是有趣的部分，我们怎么表示一个目录呢？例如，如果我们有一个包含两个文件的目录：

```sh
[root@localhost repo]# ls
[root@localhost repo]# echo "dog" > dog
[root@localhost repo]# echo "cat" > cat
[root@localhost repo]# ls
cat  dog
```

而我们要保存目录，我们首先将各个文件放入对象数据库中：

```sh
[root@localhost repo]# ugit hash-object cat
303c4c439ae4aead33fb75975fd90dfdb17a3dca
[root@localhost repo]# ugit hash-object dog 
720b2bdb7e2bef57b6fd707a4ce80c578b6b9490
```

然后我们将创建一个具有以下内容的 tree 对象：

```text
303c4c439ae4aead33fb75975fd90dfdb17a3dca cat
720b2bdb7e2bef57b6fd707a4ce80c578b6b9490 dog
```

我们也将把这个 tree 对象放入对象数据库中。 那么 tree 对象的 OID 将实际代表整个目录！ 为什么？ 

因为我们可以首先通过它的 OID 检索树对象，然后查看它包含的所有文件（它们的名称和 OID），然后读取文件的所有 OID 以获取它们的实际内容。

如果我们的目录包含其他目录怎么办？ 我们也将为它们创建树对象，并允许一个树对象指向另一个：

```sh
[root@localhost repo]# ls
cat  dog  others
[root@localhost repo]# ls others/
people
```

`root tree` 对象将如下所示：

```text
blob 303c4c439ae4aead33fb75975fd90dfdb17a3dca cat
blob 720b2bdb7e2bef57b6fd707a4ce80c578b6b9490 dog
tree e81ec10a8d62334756ea04307dfe81ed25919993 others
```

请注意，我们为每个条目添加了一个类型，以便我们知道它是文件还是目录。 代表 others 目录（OID e81ec10a8d62334756ea04307dfe81ed25919993）的 tree 看起来像：

```text
blob 5929d7e3b37d56bdd228ca72ce06015c71a03e27 people
```

我们可以将此结构视为您从计算机科学中了解到的树，其中每个条目的 OID 作为指向另一棵树或文件（叶节点）的指针。

我们知道 blob 类型的哈希值是根据其文件内容计算出来的，那么 tree 类型的哈希值是怎么计算的呢？如下可见，它是根据自身记录的信息来计算的。

```shell
[root@localhost repo]# echo -n -e 'tree\x00blob 5929d7e3b37d56bdd228ca72ce06015c71a03e27 people\n' | sha1sum 
e81ec10a8d62334756ea04307dfe81ed25919993  -
```

我们实际上在 data.hash_object() 中已经保留了类型为 tree 的对象，因为我们不希望树与常规文件混淆。

修改代码如下：

```diff
diff -u ugit.bak/base.py ugit/base.py
--- ugit.bak/base.py    2022-01-30 04:53:54.032374989 +0800
+++ ugit/base.py        2022-01-30 05:58:16.439474894 +0800
@@ -4,6 +4,7 @@
 
 
 def write_tree(directory="."):
+    entries = []
     with os.scandir(directory) as it:
         for entry in it:
             full = "{}/{}".format(directory, entry.name)
@@ -11,12 +12,21 @@
                 continue
 
             if entry.is_file(follow_symlinks=False):
+                type_ = "blob"
                 with open(full, 'rb') as f:
-                    print(data.hash_object(f.read()), full)
+                    oid = data.hash_object(f.read())
+                    print(type_, oid, full)
             elif entry.is_dir(follow_symlinks=False):
-                write_tree(full)
-
-    # TODO actually create the tree object
+                type_ = "tree"
+                oid = write_tree(full)
+            entries.append((entry.name, oid, type_))
+
+    tree = ''.join(f'{type_} {oid} {name}\n'
+                   for name, oid, type_
+                   in sorted(entries))
+    tree_oid = data.hash_object(tree.encode(), 'tree')
+    print('tree', tree_oid, directory)
+    return tree_oid
 
 
 def is_ignored(path):
```

修改的完整代码如下：

- ugit/base.py

```python
import os

from . import data


def write_tree(directory="."):
    entries = []
    with os.scandir(directory) as it:
        for entry in it:
            full = "{}/{}".format(directory, entry.name)
            if is_ignored(full):
                continue

            if entry.is_file(follow_symlinks=False):
                type_ = "blob"
                with open(full, 'rb') as f:
                    oid = data.hash_object(f.read())
                    print(type_, oid, full)
            elif entry.is_dir(follow_symlinks=False):
                type_ = "tree"
                oid = write_tree(full)
            entries.append((entry.name, oid, type_))

    tree = ''.join(f'{type_} {oid} {name}\n'
                   for name, oid, type_
                   in sorted(entries))
    tree_oid = data.hash_object(tree.encode(), 'tree')
    print('tree', tree_oid, directory)
    return tree_oid


def is_ignored(path):
    return '.ugit' in path.split("/")

```

运行：

```sh
[root@localhost repo]# ugit write-tree
blob 720b2bdb7e2bef57b6fd707a4ce80c578b6b9490 ./dog
blob 303c4c439ae4aead33fb75975fd90dfdb17a3dca ./cat
blob 5929d7e3b37d56bdd228ca72ce06015c71a03e27 ./others/people
tree e81ec10a8d62334756ea04307dfe81ed25919993 ./others
tree 8fbb67903cade11963632d96db8ec2aa473c5c5a .
[root@localhost repo]# cat .ugit/objects/8fbb67903cade11963632d96db8ec2aa473c5c5a
treeblob 303c4c439ae4aead33fb75975fd90dfdb17a3dca cat
blob 720b2bdb7e2bef57b6fd707a4ce80c578b6b9490 dog
tree e81ec10a8d62334756ea04307dfe81ed25919993 others
[root@localhost repo]# cat .ugit/objects/e81ec10a8d62334756ea04307dfe81ed25919993
treeblob 5929d7e3b37d56bdd228ca72ce06015c71a03e27 people
```

8fbb67903cade11963632d96db8ec2aa473c5c5a 是一个 tree 对象，里面有两个 blob 和一个 tree

e81ec10a8d62334756ea04307dfe81ed25919993 代表 others 目录，其下有一个 blob，是 people。





---

# read-tree

## 从对象中提取树

此命令与 write-tree 相反，它将获取树的 OID 并将其提取到工作目录。

我将实现分为几层：

- _iter_tree_entries 是一个生成器，它将获取树的 OID，逐行标记它并生成原始字符串值。

- get_tree 使用 _iter_tree_entries 递归地将树解析为字典。

- read_tree 使用 get_tree 获取文件 OID 并将它们写入工作目录。

现在我们可以实际保存工作目录的版本了！这与最终的版本控制不同，但我们可以看到以下基本流程是可能的：

1. 想象一下，您正在处理一些代码并且想要保存一个版本。
2. 你运行 ugit write-tree。
3. 您还记得打印出来的 OID（写在便利贴或其他东西上 :)）。
4. 继续工作并根据需要重复步骤 2 和 3。
5. 如果要返回到以前的版本，请使用 ugit read-tree 将其恢复到工作目录。

使用方便吗？不。但这只是开始！

修改代码如下：

```diff
diff -u ugit.bak/base.py ugit/base.py
--- ugit.bak/base.py    2022-01-30 06:32:15.741254391 +0800
+++ ugit/base.py        2022-01-30 07:18:47.289087664 +0800
@@ -29,6 +29,37 @@
     return tree_oid
 
 
+def _iter_tree_entries(oid):
+    if not oid:
+        return
+    tree = data.get_object(oid, 'tree')
+    for entry in tree.decode().splitlines():
+        type_, oid, name = entry.split(' ', 2)
+        yield type_, oid, name
+
+
+def get_tree(oid, base_path=''):
+    result = {}
+    for type_, oid, name in _iter_tree_entries(oid):
+        assert '/' not in name
+        assert name not in ('..', '.')
+        path = base_path + name
+        if type_ == 'blob':
+            result[path] = oid
+        elif type_ == 'tree':
+            result.update(get_tree(oid, f'{path}/'))
+        else:
+            assert False, f'Unknown tree entry {type_}'
+    return result
+
+
+def read_tree(tree_oid):
+    for path, oid in get_tree(tree_oid, base_path='./').items():
+        os.makedirs(os.path.dirname(path), exist_ok=True)
+        with open(path, 'wb') as f:
+            f.write(data.get_object(oid))
+
+
 def is_ignored(path):
     return '.ugit' in path.split("/")
 

===========================================================
diff -u ugit.bak/cli.py ugit/cli.py
--- ugit.bak/cli.py     2022-01-30 06:32:15.741254391 +0800
+++ ugit/cli.py 2022-01-30 06:36:27.807695662 +0800
@@ -31,6 +31,10 @@
     write_tree_parser = commands.add_parser('write-tree')
     write_tree_parser.set_defaults(func=write_tree)
 
+    read_tree_parser = commands.add_parser('read-tree')
+    read_tree_parser.set_defaults(func=read_tree)
+    read_tree_parser.add_argument('tree')
+
     return parser.parse_args()
 
 
@@ -53,6 +57,10 @@
     base.write_tree()
 
 
+def read_tree(args):
+    base.read_tree(args.tree)
+
+
 if __name__ == "__main__":
     main()
```

完整代码如下：

- ugit/cli.py

```python
import argparse
import os
import sys

from . import base
from . import data


def main():
    args = parse_args()
    args.func(args)


def parse_args():
    parser = argparse.ArgumentParser()

    commands = parser.add_subparsers(dest="command")
    commands.required = True

    init_parser = commands.add_parser("init")
    init_parser.set_defaults(func=init)

    hash_object_parser = commands.add_parser('hash-object')
    hash_object_parser.set_defaults(func=hash_object)
    hash_object_parser.add_argument('file')

    cat_file_parser = commands.add_parser('cat-file')
    cat_file_parser.set_defaults(func=cat_file)
    cat_file_parser.add_argument('object')

    write_tree_parser = commands.add_parser('write-tree')
    write_tree_parser.set_defaults(func=write_tree)

    read_tree_parser = commands.add_parser('read-tree')
    read_tree_parser.set_defaults(func=read_tree)
    read_tree_parser.add_argument('tree')

    return parser.parse_args()


def init(args):
    data.init()
    print("Initialized empty ugit repository in {}/{}".format(os.getcwd(), data.GIT_DIR))


def hash_object(args):
    with open(args.file, 'rb') as f:
        print(data.hash_object(f.read()))


def cat_file(args):
    sys.stdout.flush()
    sys.stdout.buffer.write(data.get_object(args.object, expected=None))


def write_tree(args):
    base.write_tree()


def read_tree(args):
    base.read_tree(args.tree)


if __name__ == "__main__":
    main()

```

- ugit/base.py

```python
import os

from . import data


def write_tree(directory="."):
    entries = []
    with os.scandir(directory) as it:
        for entry in it:
            full = "{}/{}".format(directory, entry.name)
            if is_ignored(full):
                continue

            if entry.is_file(follow_symlinks=False):
                type_ = "blob"
                with open(full, 'rb') as f:
                    oid = data.hash_object(f.read())
                    print(type_, oid, full)
            elif entry.is_dir(follow_symlinks=False):
                type_ = "tree"
                oid = write_tree(full)
            entries.append((entry.name, oid, type_))

    tree = ''.join(f'{type_} {oid} {name}\n'
                   for name, oid, type_
                   in sorted(entries))
    tree_oid = data.hash_object(tree.encode(), 'tree')
    print('tree', tree_oid, directory)
    return tree_oid


def _iter_tree_entries(oid):
    if not oid:
        return
    tree = data.get_object(oid, 'tree')
    for entry in tree.decode().splitlines():
        type_, oid, name = entry.split(' ', 2)
        yield type_, oid, name


def get_tree(oid, base_path=''):
    result = {}
    for type_, oid, name in _iter_tree_entries(oid):
        assert '/' not in name
        assert name not in ('..', '.')
        path = base_path + name
        if type_ == 'blob':
            result[path] = oid
        elif type_ == 'tree':
            result.update(get_tree(oid, f'{path}/'))
        else:
            assert False, f'Unknown tree entry {type_}'
    return result


def read_tree(tree_oid):
    for path, oid in get_tree(tree_oid, base_path='./').items():
        os.makedirs(os.path.dirname(path), exist_ok=True)
        with open(path, 'wb') as f:
            f.write(data.get_object(oid))


def is_ignored(path):
    return '.ugit' in path.split("/")

```



运行测试：

```sh
[root@localhost repo]# cat cat dog others/people 
cat
dog
people
[root@localhost repo]# ugit write-tree
blob 720b2bdb7e2bef57b6fd707a4ce80c578b6b9490 ./dog
blob 303c4c439ae4aead33fb75975fd90dfdb17a3dca ./cat
blob 5929d7e3b37d56bdd228ca72ce06015c71a03e27 ./others/people
tree e81ec10a8d62334756ea04307dfe81ed25919993 ./others
tree 8fbb67903cade11963632d96db8ec2aa473c5c5a .
[root@localhost repo]# echo "cat1" > cat
[root@localhost repo]# ugit write-tree
blob 720b2bdb7e2bef57b6fd707a4ce80c578b6b9490 ./dog
blob a45ffde12eabc9c8a9716582ac7f41015d9d5dbb ./cat
blob 5929d7e3b37d56bdd228ca72ce06015c71a03e27 ./others/people
tree e81ec10a8d62334756ea04307dfe81ed25919993 ./others
tree e73bfc8a71f02abb8496995306eec9ed5e8cc2f5 .
[root@localhost repo]# ugit read-tree 8fbb67903cade11963632d96db8ec2aa473c5c5a
[root@localhost repo]# cat cat dog others/people 
cat
dog
people
[root@localhost repo]# ugit read-tree e73bfc8a71f02abb8496995306eec9ed5e8cc2f5
[root@localhost repo]# cat cat dog others/people 
cat1
dog
people
```

可以看到我们通过使用 write-tree 生成的 OID 可以恢复到任意版本。



## 清理旧文件

这样做是为了在 read-tree 后我们不会有任何旧文件。在此更改之前，如果我们保存只包含 a.txt 的树 A，然后保存包含 a.txt 和 b.txt 的树 B，然后 read-tree A，我们将在工作目录中留下 b.txt。

代码更改如下：

```diff
diff -u ugit.bak/base.py ugit/base.py
--- ugit.bak/base.py    2022-01-30 07:37:24.325186842 +0800
+++ ugit/base.py        2022-01-30 07:45:51.543364487 +0800
@@ -53,7 +53,27 @@
     return result
 
 
+def _empty_current_directory():
+    for root, dirnames, filenames in os.walk('.', topdown=False):
+        for filename in filenames:
+            path = os.path.relpath(f'{root}/{filename}')
+            if is_ignored(path) or not os.path.isfile(path):
+                continue
+            os.remove(path)
+        for dirname in dirnames:
+            path = os.path.relpath(f'{root}/{dirname}')
+            if is_ignored(path):
+                continue
+            try:
+                os.rmdir(path)
+            except (FileNotFoundError, OSError):
+                # Deletion might fail if the directory contains ignored files,
+                # so it's OK
+                pass
+
+
 def read_tree(tree_oid):
+    _empty_current_directory()
     for path, oid in get_tree(tree_oid, base_path='./').items():
         os.makedirs(os.path.dirname(path), exist_ok=True)
         with open(path, 'wb') as f:
```

测试如下，我们

```sh
[root@localhost ~]# cd repo/
[root@localhost repo]# ls
cat  dog  others
[root@localhost repo]# ugit write-tree
blob 720b2bdb7e2bef57b6fd707a4ce80c578b6b9490 ./dog
blob a45ffde12eabc9c8a9716582ac7f41015d9d5dbb ./cat
blob 5929d7e3b37d56bdd228ca72ce06015c71a03e27 ./others/people
tree e81ec10a8d62334756ea04307dfe81ed25919993 ./others
tree e73bfc8a71f02abb8496995306eec9ed5e8cc2f5 .
[root@localhost repo]# touch add
[root@localhost repo]# ls
add  cat  dog  others
[root@localhost repo]# ugit read-tree e73bfc8a71f02abb8496995306eec9ed5e8cc2f5
[root@localhost repo]# ls
cat  dog  others
```





---

# commit

## 创建提交

到目前为止，我们使用 write-tree 能够保存目录的版本，但是没有任何额外的上下文。实际上，当我们保存快照时，我们希望附加一些数据，比如：

- 消息描述
- 何时创建
- 谁创建的
- 。。。

我们将创建一个名为 commit 的新类型的对象来存储所有这些信息。提交只是存储在对象数据库中的文本文件，类型为 commit。

提交中的开始几行将是键值对，然后以一个空白行标记键值的结束，提交消息将跟随其后，就像这样：

```text
tree 5e550586c91fce59e0006799e0d46b3948f05693
author Nikita Leshenko
time 2019-09-14T09:31:09+00:00

This is the commit message!
```

现在，我们将只编写包含 tree 和提交消息的 commit 对象。 我们将创建一个新的 ugit commit 命令，该命令将接受一个提交消息，配合使用 ugit write-tree 可以对当前目录进行快照。

代码修改如下：

```diff
diff -u ugit.bak/base.py ugit/base.py
--- ugit.bak/base.py    2022-01-30 11:14:27.263186422 +0800
+++ ugit/base.py        2022-01-30 11:23:08.138864164 +0800
@@ -80,6 +80,14 @@
             f.write(data.get_object(oid))
 
 
+def commit(message):
+    commit = f'tree {write_tree()}\n'
+    commit += '\n'
+    commit += f'{message}\n'
+
+    return data.hash_object(commit.encode(), 'commit')
+
+
 def is_ignored(path):
     return '.ugit' in path.split("/")


===========================================================
diff -u ugit.bak/cli.py ugit/cli.py
--- ugit.bak/cli.py     2022-01-30 11:14:27.262103091 +0800
+++ ugit/cli.py 2022-01-30 11:17:40.638995433 +0800
@@ -35,6 +35,10 @@
     read_tree_parser.set_defaults(func=read_tree)
     read_tree_parser.add_argument('tree')
 
+    commit_parser = commands.add_parser('commit')
+    commit_parser.set_defaults(func=commit)
+    commit_parser.add_argument('-m', '--message', required=True)
+
     return parser.parse_args()
 
 
@@ -57,10 +61,14 @@
     base.write_tree()
 

def read_tree(args):
     base.read_tree(args.tree)
 
 
+def commit(args):
+    print(base.commit(args.message))
+
+
 if __name__ == "__main__":
     main()
```

运行测试：

```sh
[root@localhost repo]# ugit commit -m "first commit"
blob a45ffde12eabc9c8a9716582ac7f41015d9d5dbb ./cat
blob 720b2bdb7e2bef57b6fd707a4ce80c578b6b9490 ./dog
blob 5929d7e3b37d56bdd228ca72ce06015c71a03e27 ./others/people
tree e81ec10a8d62334756ea04307dfe81ed25919993 ./others
tree e73bfc8a71f02abb8496995306eec9ed5e8cc2f5 .
9141f88bf2de5747ab33f1729d63bfcc0c07fff2
[root@localhost repo]# ugit cat-file 9141f88bf2de5747ab33f1729d63bfcc0c07fff2
tree e73bfc8a71f02abb8496995306eec9ed5e8cc2f5

first commit
```

提交之后产生了一个提交记录 9141f88bf2de5747ab33f1729d63bfcc0c07fff2，里面记录的是此次提交的 tree oid 和提交信息。



## 记录HEAD

如果我们在工作目录中进行更改并进行定期提交，每个提交都将是一个独立的对象。现在我想将新提交链接到旧提交，让它们联系在一起的动机是为了让我们可以将提交看作是以某种顺序排列的一系列沙漏。

在此之前，让我们记录下我们创建的最后一个提交的 OID。我们将最后一次提交称为 HEAD，只需将 OID 放入 `.ugit/HEAD` 文件即可。

代码修改如下：

```diff
diff -u ugit.bak/base.py ugit/base.py
--- ugit.bak/base.py    2022-01-30 11:34:50.513741117 +0800
+++ ugit/base.py        2022-01-30 11:37:33.868247229 +0800
@@ -85,8 +85,10 @@
     commit += '\n'
     commit += f'{message}\n'
 
-    return data.hash_object(commit.encode(), 'commit')
+    oid = data.hash_object(commit.encode(), 'commit')
+    data.set_HEAD(oid)
 
+    return oid
 
 def is_ignored(path):
     return '.ugit' in path.split("/")


===========================================================
diff -u ugit.bak/data.py ugit/data.py
--- ugit.bak/data.py    2022-01-30 11:34:50.513741117 +0800
+++ ugit/data.py        2022-01-30 11:36:36.522082851 +0800
@@ -10,6 +10,11 @@
     os.makedirs(OBJECTS_DIR)
 
 
+def set_HEAD(oid):
+    with open(f'{GIT_DIR}/HEAD', 'w') as f:
+        f.write(oid)
+
+
 def hash_object(data, type_="blob"):
     obj = type_.encode() + b"\x00" + data
     oid = hashlib.sha1(obj).hexdigest()
```

运行：

```sh
[root@localhost repo]# ugit commit -m "dog1"
blob a45ffde12eabc9c8a9716582ac7f41015d9d5dbb ./cat
blob 5929d7e3b37d56bdd228ca72ce06015c71a03e27 ./others/people
tree e81ec10a8d62334756ea04307dfe81ed25919993 ./others
blob 0f84d81a999a9768f185ab81862399251d063995 ./dog
tree 91c4a8227c4a3356adc12ce1521020dcfe6b0ac7 .
417b2893adf209c6f2f862ab1c969dac8b069f49
[root@localhost repo]# cat .ugit/HEAD 
417b2893adf209c6f2f862ab1c969dac8b069f49
```



## 链接提交

我们将以前的提交称为父提交，并将它的 OID 保存在下一次提交对象的父键中。

例如，HEAD 当前为 bd0de093f1a0f90f54913d694a11cccf450bd990，我们创建一个新的提交，新的提交在对象存储中看起来如下所示:

```text
tree 50bed982245cd21e2798f179e0b032904398485b
parent bd0de093f1a0f90f54913d694a11cccf450bd990

This is the commit message!
```

存储库中的第一个提交显然没有父提交。

现在我们可以通过引用最后一次提交来检索整个提交列表！我们可以从 HEAD 开始，读取 HEAD 提交上的父键，就可以发现 HEAD 上一次的提交。然后阅读该提交的父键，并继续下去...这基本上是一个在对象数据库上实现的提交链表。

修改代码如下：

```diff
diff -u ugit.bak/base.py ugit/base.py
--- ugit.bak/base.py    2022-01-30 12:12:06.724269833 +0800
+++ ugit/base.py        2022-01-30 12:18:34.229762543 +0800
@@ -82,6 +82,11 @@
 
 def commit(message):
     commit = f'tree {write_tree()}\n'
+
+    HEAD = data.get_HEAD()
+    if HEAD:
+        commit += f'parent {HEAD}\n'
+
     commit += '\n'
     commit += f'{message}\n'


===========================================================
diff -u ugit.bak/data.py ugit/data.py
--- ugit.bak/data.py    2022-01-30 12:12:06.724269833 +0800
+++ ugit/data.py        2022-01-30 12:16:20.275828850 +0800
@@ -15,6 +15,12 @@
         f.write(oid)
 
 
+def get_HEAD():
+    if os.path.isfile(f'{GIT_DIR}/HEAD'):
+        with open(f'{GIT_DIR}/HEAD') as f:
+            return f.read().strip()
+
+
 def hash_object(data, type_="blob"):
     obj = type_.encode() + b"\x00" + data
     oid = hashlib.sha1(obj).hexdigest()
```

运行：

```sh
[root@localhost repo]# ugit commit -m "1"
blob a45ffde12eabc9c8a9716582ac7f41015d9d5dbb ./cat
blob 5929d7e3b37d56bdd228ca72ce06015c71a03e27 ./others/people
tree e81ec10a8d62334756ea04307dfe81ed25919993 ./others
blob 0f84d81a999a9768f185ab81862399251d063995 ./dog
tree 91c4a8227c4a3356adc12ce1521020dcfe6b0ac7 .
6fb7146829487660073b8492d2514ec16d4332c1
[root@localhost repo]# ugit commit -m "2"
blob a45ffde12eabc9c8a9716582ac7f41015d9d5dbb ./cat
blob 5929d7e3b37d56bdd228ca72ce06015c71a03e27 ./others/people
tree e81ec10a8d62334756ea04307dfe81ed25919993 ./others
blob 0f84d81a999a9768f185ab81862399251d063995 ./dog
tree 91c4a8227c4a3356adc12ce1521020dcfe6b0ac7 .
e481292c50e99b80085bef44b2a5d0bd9d55da51
[root@localhost repo]# cat .ugit/HEAD 
e481292c50e99b80085bef44b2a5d0bd9d55da51
[root@localhost repo]# ugit cat-file e481292c50e99b80085bef44b2a5d0bd9d55da51
tree 91c4a8227c4a3356adc12ce1521020dcfe6b0ac7
parent 6fb7146829487660073b8492d2514ec16d4332c1

2
[root@localhost repo]# ugit cat-file 6fb7146829487660073b8492d2514ec16d4332c1
tree 91c4a8227c4a3356adc12ce1521020dcfe6b0ac7
parent 417b2893adf209c6f2f862ab1c969dac8b069f49

1
[root@localhost repo]# ugit cat-file 417b2893adf209c6f2f862ab1c969dac8b069f49
tree 91c4a8227c4a3356adc12ce1521020dcfe6b0ac7

dog1
```

我们制造了两个提交，最后一次提交记录为 e481292c50e99b80085bef44b2a5d0bd9d55da51，我们顺着这个提交记录可以一直往回追溯。

完整代码如下：

- ugit/cli.py

```python
import argparse
import os
import sys

from . import base
from . import data


def main():
    args = parse_args()
    args.func(args)


def parse_args():
    parser = argparse.ArgumentParser()

    commands = parser.add_subparsers(dest="command")
    commands.required = True

    init_parser = commands.add_parser("init")
    init_parser.set_defaults(func=init)

    hash_object_parser = commands.add_parser('hash-object')
    hash_object_parser.set_defaults(func=hash_object)
    hash_object_parser.add_argument('file')

    cat_file_parser = commands.add_parser('cat-file')
    cat_file_parser.set_defaults(func=cat_file)
    cat_file_parser.add_argument('object')

    write_tree_parser = commands.add_parser('write-tree')
    write_tree_parser.set_defaults(func=write_tree)

    read_tree_parser = commands.add_parser('read-tree')
    read_tree_parser.set_defaults(func=read_tree)
    read_tree_parser.add_argument('tree')

    commit_parser = commands.add_parser('commit')
    commit_parser.set_defaults(func=commit)
    commit_parser.add_argument('-m', '--message', required=True)

    return parser.parse_args()


def init(args):
    data.init()
    print("Initialized empty ugit repository in {}/{}".format(os.getcwd(), data.GIT_DIR))


def hash_object(args):
    with open(args.file, 'rb') as f:
        print(data.hash_object(f.read()))


def cat_file(args):
    sys.stdout.flush()
    sys.stdout.buffer.write(data.get_object(args.object, expected=None))


def write_tree(args):
    base.write_tree()


def read_tree(args):
    base.read_tree(args.tree)


def commit(args):
    print(base.commit(args.message))


if __name__ == "__main__":
    main()

```

- ugit/data.py

```python
import os
import hashlib

GIT_DIR = ".ugit"
OBJECTS_DIR = GIT_DIR + "/objects"


def init():
    os.makedirs(GIT_DIR)
    os.makedirs(OBJECTS_DIR)


def set_HEAD(oid):
    with open(f'{GIT_DIR}/HEAD', 'w') as f:
        f.write(oid)


def get_HEAD():
    if os.path.isfile(f'{GIT_DIR}/HEAD'):
        with open(f'{GIT_DIR}/HEAD') as f:
            return f.read().strip()


def hash_object(data, type_="blob"):
    obj = type_.encode() + b"\x00" + data
    oid = hashlib.sha1(obj).hexdigest()
    with open("{}/{}".format(OBJECTS_DIR, oid), "wb") as f:
        f.write(obj)
    return oid


def get_object(oid, expected="blob"):
    with open("{}/{}".format(OBJECTS_DIR, oid), 'rb') as f:
        obj = f.read()

    type_, _, content = obj.partition(b'\x00')
    type_ = type_.decode()

    if expected is not None:
        assert type_ == expected, "Expected {}, got {}".format(expected, type_)
        
    return content

```

- ugit/base.py

```python
import os

from . import data


def write_tree(directory="."):
    entries = []
    with os.scandir(directory) as it:
        for entry in it:
            full = "{}/{}".format(directory, entry.name)
            if is_ignored(full):
                continue

            if entry.is_file(follow_symlinks=False):
                type_ = "blob"
                with open(full, 'rb') as f:
                    oid = data.hash_object(f.read())
                    print(type_, oid, full)
            elif entry.is_dir(follow_symlinks=False):
                type_ = "tree"
                oid = write_tree(full)
            entries.append((entry.name, oid, type_))

    tree = ''.join(f'{type_} {oid} {name}\n'
                   for name, oid, type_
                   in sorted(entries))
    tree_oid = data.hash_object(tree.encode(), 'tree')
    print('tree', tree_oid, directory)
    return tree_oid


def _iter_tree_entries(oid):
    if not oid:
        return
    tree = data.get_object(oid, 'tree')
    for entry in tree.decode().splitlines():
        type_, oid, name = entry.split(' ', 2)
        yield type_, oid, name


def get_tree(oid, base_path=''):
    result = {}
    for type_, oid, name in _iter_tree_entries(oid):
        assert '/' not in name
        assert name not in ('..', '.')
        path = base_path + name
        if type_ == 'blob':
            result[path] = oid
        elif type_ == 'tree':
            result.update(get_tree(oid, f'{path}/'))
        else:
            assert False, f'Unknown tree entry {type_}'
    return result


def _empty_current_directory():
    for root, dirnames, filenames in os.walk('.', topdown=False):
        for filename in filenames:
            path = os.path.relpath(f'{root}/{filename}')
            if is_ignored(path) or not os.path.isfile(path):
                continue
            os.remove(path)
        for dirname in dirnames:
            path = os.path.relpath(f'{root}/{dirname}')
            if is_ignored(path):
                continue
            try:
                os.rmdir(path)
            except (FileNotFoundError, OSError):
                # Deletion might fail if the directory contains ignored files,
                # so it's OK
                pass


def read_tree(tree_oid):
    _empty_current_directory()
    for path, oid in get_tree(tree_oid, base_path='./').items():
        os.makedirs(os.path.dirname(path), exist_ok=True)
        with open(path, 'wb') as f:
            f.write(data.get_object(oid))


def commit(message):
    commit = f'tree {write_tree()}\n'

    HEAD = data.get_HEAD()
    if HEAD:
        commit += f'parent {HEAD}\n'

    commit += '\n'
    commit += f'{message}\n'

    oid = data.hash_object(commit.encode(), 'commit')
    data.set_HEAD(oid)

    return oid

def is_ignored(path):
    return '.ugit' in path.split("/")

```



---

# log

log 命令将遍历提交列表并打印它们。 我们将从实现 get_commit() 开始，它将解析 OID 提交的对象。 

然后我们将从 HEAD 提交开始，遍历它的父级，直到到达没有父级的提交。 结果是，一旦我们运行 ugit log，整个提交历史就会打印到屏幕上。

代码修改如下：

```diff
diff -u ugit.bak/base.py ugit/base.py
--- ugit.bak/base.py    2022-01-30 12:42:33.846255307 +0800
+++ ugit/base.py        2022-01-30 12:49:26.651936678 +0800
@@ -1,5 +1,9 @@
+import itertools
+import operator
 import os
 
+from collections import namedtuple
+
 from . import data
 
 
@@ -95,6 +99,26 @@
 
     return oid
 
+
+Commit = namedtuple('Commit', ['tree', 'parent', 'message'])
+def get_commit(oid):
+    parent = None
+
+    commit = data.get_object(oid, 'commit').decode()
+    lines = iter(commit.splitlines())
+    for line in itertools.takewhile(operator.truth, lines):
+        key, value = line.split(' ', 1)
+        if key == 'tree':
+            tree = value
+        elif key == 'parent':
+            parent = value
+        else:
+            assert False, f'Unknown field {key}'
+
+    message = '\n'.join(lines)
+    return Commit(tree=tree, parent=parent, message=message)
+
+
 def is_ignored(path):
     return '.ugit' in path.split("/")


===========================================================
diff -u ugit.bak/cli.py ugit/cli.py
--- ugit.bak/cli.py     2022-01-30 12:42:33.846255307 +0800
+++ ugit/cli.py 2022-01-30 13:02:27.338620351 +0800
@@ -1,6 +1,7 @@
 import argparse
 import os
 import sys
+import textwrap
 
 from . import base
 from . import data
@@ -39,6 +40,9 @@
     commit_parser.set_defaults(func=commit)
     commit_parser.add_argument('-m', '--message', required=True)
 
+    log_parser = commands.add_parser('log')
+    log_parser.set_defaults(func=log)
+
     return parser.parse_args()
 
 
@@ -69,6 +73,19 @@
     print(base.commit(args.message))
 
 
+def log(args):
+    oid = data.get_HEAD()
+    while oid:
+        commit = base.get_commit(oid)
+
+        print(f'[commit] {oid}\n')
+        print('[message]:')
+        print(textwrap.indent(commit.message, '    '))
+        print('=' * 58)
+
+        oid = commit.parent
+
+
 if __name__ == "__main__":
     main()
```

运行：

```sh
[root@localhost repo]# ugit log
[commit] e481292c50e99b80085bef44b2a5d0bd9d55da51

[message]:
    2
==========================================================
[commit] 6fb7146829487660073b8492d2514ec16d4332c1

[message]:
    1
==========================================================
[commit] 417b2893adf209c6f2f862ab1c969dac8b069f49

[message]:
    dog1
==========================================================
```

下面做个小的修饰性改变：不是总是从 HEAD 打印提交列表，而是添加一个可选参数可以指定从某个提交开始。默认情况下，它仍然是 HEAD。

修改代码：

```diff
diff -u ugit.bak/cli.py ugit/cli.py
--- ugit.bak/cli.py     2022-01-30 13:18:11.761376168 +0800
+++ ugit/cli.py 2022-01-30 13:10:25.540743168 +0800
@@ -42,6 +42,7 @@
 
     log_parser = commands.add_parser('log')
     log_parser.set_defaults(func=log)
+    log_parser.add_argument('oid', nargs='?')
 
     return parser.parse_args()
 
@@ -74,7 +75,7 @@
 
 
 def log(args):
-    oid = data.get_HEAD()
+    oid = args.oid or data.get_HEAD()
     while oid:
         commit = base.get_commit(oid)
```

测试：

```sh
[root@localhost repo]# ugit log
[commit] e481292c50e99b80085bef44b2a5d0bd9d55da51

[message]:
    2
==========================================================
[commit] 6fb7146829487660073b8492d2514ec16d4332c1

[message]:
    1
==========================================================
[commit] 417b2893adf209c6f2f862ab1c969dac8b069f49

[message]:
    dog1
==========================================================
[root@localhost repo]# ugit log 6fb7146829487660073b8492d2514ec16d4332c1
[commit] 6fb7146829487660073b8492d2514ec16d4332c1

[message]:
    1
==========================================================
[commit] 417b2893adf209c6f2f862ab1c969dac8b069f49

[message]:
    dog1
==========================================================
```

完整代码如下：

- ugit/cli.py

```python
import argparse
import os
import sys
import textwrap

from . import base
from . import data


def main():
    args = parse_args()
    args.func(args)


def parse_args():
    parser = argparse.ArgumentParser()

    commands = parser.add_subparsers(dest="command")
    commands.required = True

    init_parser = commands.add_parser("init")
    init_parser.set_defaults(func=init)

    hash_object_parser = commands.add_parser('hash-object')
    hash_object_parser.set_defaults(func=hash_object)
    hash_object_parser.add_argument('file')

    cat_file_parser = commands.add_parser('cat-file')
    cat_file_parser.set_defaults(func=cat_file)
    cat_file_parser.add_argument('object')

    write_tree_parser = commands.add_parser('write-tree')
    write_tree_parser.set_defaults(func=write_tree)

    read_tree_parser = commands.add_parser('read-tree')
    read_tree_parser.set_defaults(func=read_tree)
    read_tree_parser.add_argument('tree')

    commit_parser = commands.add_parser('commit')
    commit_parser.set_defaults(func=commit)
    commit_parser.add_argument('-m', '--message', required=True)

    log_parser = commands.add_parser('log')
    log_parser.set_defaults(func=log)
    log_parser.add_argument('oid', nargs='?')

    return parser.parse_args()


def init(args):
    data.init()
    print("Initialized empty ugit repository in {}/{}".format(os.getcwd(), data.GIT_DIR))


def hash_object(args):
    with open(args.file, 'rb') as f:
        print(data.hash_object(f.read()))


def cat_file(args):
    sys.stdout.flush()
    sys.stdout.buffer.write(data.get_object(args.object, expected=None))


def write_tree(args):
    base.write_tree()


def read_tree(args):
    base.read_tree(args.tree)


def commit(args):
    print(base.commit(args.message))


def log(args):
    oid = args.oid or data.get_HEAD()
    while oid:
        commit = base.get_commit(oid)

        print(f'[commit] {oid}\n')
        print('[message]:')
        print(textwrap.indent(commit.message, '    '))
        print('=' * 58)

        oid = commit.parent


if __name__ == "__main__":
    main()

```



- ugit/data.py

```python
import os
import hashlib

GIT_DIR = ".ugit"
OBJECTS_DIR = GIT_DIR + "/objects"


def init():
    os.makedirs(GIT_DIR)
    os.makedirs(OBJECTS_DIR)


def set_HEAD(oid):
    with open(f'{GIT_DIR}/HEAD', 'w') as f:
        f.write(oid)


def get_HEAD():
    if os.path.isfile(f'{GIT_DIR}/HEAD'):
        with open(f'{GIT_DIR}/HEAD') as f:
            return f.read().strip()


def hash_object(data, type_="blob"):
    obj = type_.encode() + b"\x00" + data
    oid = hashlib.sha1(obj).hexdigest()
    with open("{}/{}".format(OBJECTS_DIR, oid), "wb") as f:
        f.write(obj)
    return oid


def get_object(oid, expected="blob"):
    with open("{}/{}".format(OBJECTS_DIR, oid), 'rb') as f:
        obj = f.read()

    type_, _, content = obj.partition(b'\x00')
    type_ = type_.decode()

    if expected is not None:
        assert type_ == expected, "Expected {}, got {}".format(expected, type_)
        
    return content

```



- ugit/base.py

```python
import itertools
import operator
import os

from collections import namedtuple

from . import data


def write_tree(directory="."):
    entries = []
    with os.scandir(directory) as it:
        for entry in it:
            full = "{}/{}".format(directory, entry.name)
            if is_ignored(full):
                continue

            if entry.is_file(follow_symlinks=False):
                type_ = "blob"
                with open(full, 'rb') as f:
                    oid = data.hash_object(f.read())
                    print(type_, oid, full)
            elif entry.is_dir(follow_symlinks=False):
                type_ = "tree"
                oid = write_tree(full)
            entries.append((entry.name, oid, type_))

    tree = ''.join(f'{type_} {oid} {name}\n'
                   for name, oid, type_
                   in sorted(entries))
    tree_oid = data.hash_object(tree.encode(), 'tree')
    print('tree', tree_oid, directory)
    return tree_oid


def _iter_tree_entries(oid):
    if not oid:
        return
    tree = data.get_object(oid, 'tree')
    for entry in tree.decode().splitlines():
        type_, oid, name = entry.split(' ', 2)
        yield type_, oid, name


def get_tree(oid, base_path=''):
    result = {}
    for type_, oid, name in _iter_tree_entries(oid):
        assert '/' not in name
        assert name not in ('..', '.')
        path = base_path + name
        if type_ == 'blob':
            result[path] = oid
        elif type_ == 'tree':
            result.update(get_tree(oid, f'{path}/'))
        else:
            assert False, f'Unknown tree entry {type_}'
    return result


def _empty_current_directory():
    for root, dirnames, filenames in os.walk('.', topdown=False):
        for filename in filenames:
            path = os.path.relpath(f'{root}/{filename}')
            if is_ignored(path) or not os.path.isfile(path):
                continue
            os.remove(path)
        for dirname in dirnames:
            path = os.path.relpath(f'{root}/{dirname}')
            if is_ignored(path):
                continue
            try:
                os.rmdir(path)
            except (FileNotFoundError, OSError):
                # Deletion might fail if the directory contains ignored files,
                # so it's OK
                pass


def read_tree(tree_oid):
    _empty_current_directory()
    for path, oid in get_tree(tree_oid, base_path='./').items():
        os.makedirs(os.path.dirname(path), exist_ok=True)
        with open(path, 'wb') as f:
            f.write(data.get_object(oid))


def commit(message):
    commit = f'tree {write_tree()}\n'

    HEAD = data.get_HEAD()
    if HEAD:
        commit += f'parent {HEAD}\n'

    commit += '\n'
    commit += f'{message}\n'

    oid = data.hash_object(commit.encode(), 'commit')
    data.set_HEAD(oid)

    return oid


Commit = namedtuple('Commit', ['tree', 'parent', 'message'])
def get_commit(oid):
    parent = None

    commit = data.get_object(oid, 'commit').decode()
    lines = iter(commit.splitlines())
    for line in itertools.takewhile(operator.truth, lines):
        key, value = line.split(' ', 1)
        if key == 'tree':
            tree = value
        elif key == 'parent':
            parent = value
        else:
            assert False, f'Unknown field {key}'

    message = '\n'.join(lines)
    return Commit(tree=tree, parent=parent, message=message)


def is_ignored(path):
    return '.ugit' in path.split("/")

```

