# checkout

当给定一个提交 OID 时，ugit checkout 将移动到该提交，这意味着它将用提交的内容填充工作目录，并移动 HEAD 指向它。这是一个微小但重要的变化，但它从两个方面极大地扩展了 ugit 的力量：

首先，它让我们可以方便地在历史上旅行。如果我们已经进行了一些提交，并且想要重新访问以前的提交，我们现在可以 checkout 该提交到工作目录，使用它编译、运行测试、读取代码，并且把这当做最后的提交，以便我们在 checkout 的地方继续工作。

您可能想知道，当我们可以有 read-tree 时，为什么还需要 checkout？

答案是除了 read-tree 之外，移动 HEAD 还允许我们记录哪个提交现在被 checkout。如果我们只使用 read-tree，然后忘记我们正在查看哪个提交，我们将在工作目录中看到一堆文件，并且不知道它们来自哪里。另一方面，如果我们使用 checkout，提交将被记录在 HEAD 中，我们总是可以知道我们在看什么，例如通过运行 ugit log 查看到第一个条目。

第二是 checkout 允许历史的多个分支。到目前为止，我们已经将 HEAD 设置为指向创建的最新提交，这意味着我们所有的提交都是线性的，每个新的提交都是在前一个提交的基础上添加的。checkout 命令现在允许我们将 HEAD 移动到任何我们想要的提交位置。然后，将在当前 HEAD 提交的基础上创建新的提交。

例如，假设我们正在处理一些代码。到目前为止，我们已经创建了一些提交：

```text
o-----o-----o-----o
^                 ^
first commit      HEAD
```

然后我们想编写一个新特性。我们在处理该特性时创建了一些提交，新的提交由 @ 表示：

```text
o-----o-----o-----o-----@-----@-----@
^                                   ^
first commit                        HEAD
```

现在我们有了一个实现这个特性的替代方案。我们希望回到过去，尝试一种不同的实现方式，而不抛弃当前的实现方式。我们可以记住当前的 HEAD OID 以便切换回来，然后运行 ugit checkout，通过提供新特性实现之前提交的 OID（可以通过 ugit log 查找）回到过去。

```text
o-----o-----o-----o-----@-----@-----@
^                 ^
first commit      HEAD
```

工作目录将有效地回到过去。我们可以开始另一种实现，并创建新的提交（用 $ 表示）。新的提交将在 HEAD 之上，如下所示：

```text
o-----o-----o-----o-----@-----@-----@
^                  \
first commit        ----$-----$
                              ^
                              HEAD
```

看看历史现在是如何包含两个分支的。我们实际上可以在它们之间来回切换，并并行处理它们。假设我们喜欢第二个分支，我们将继续从它开始工作，未来的提交将如下所示：

```text
o-----o-----o-----o-----@-----@-----@
^                  \
first commit        ----$-----$-----o-----o-----o-----o-----o
                                                            ^
                                                            HEAD
```

上面描述了这么多，但是实际代码修改很简单：

```diff
diff -u ugit.bak/base.py ugit/base.py
--- ugit.bak/base.py    2022-01-30 14:52:57.160167161 +0800
+++ ugit/base.py        2022-01-30 15:10:47.746033900 +0800
@@ -100,6 +100,12 @@
     return oid
 
 
+def checkout(oid):
+    commit = get_commit(oid)
+    read_tree(commit.tree)
+    data.set_HEAD(oid)
+
+
 Commit = namedtuple('Commit', ['tree', 'parent', 'message'])
 def get_commit(oid):
     parent = None
     
     
===========================================================
diff -u ugit.bak/cli.py ugit/cli.py
--- ugit.bak/cli.py     2022-01-30 14:52:57.160167161 +0800
+++ ugit/cli.py 2022-01-30 15:08:58.782188680 +0800
@@ -44,6 +44,10 @@
     log_parser.set_defaults(func=log)
     log_parser.add_argument('oid', nargs='?')
 
+    checkout_parser = commands.add_parser('checkout')
+    checkout_parser.set_defaults(func=checkout)
+    checkout_parser.add_argument('oid')
+
     return parser.parse_args()
 
 
@@ -87,6 +91,10 @@
         oid = commit.parent
 
 
+def checkout(args):
+    base.checkout(args.oid)
+
+
 if __name__ == "__main__":
     main()
```

运行测试：

```sh
# 旧分支
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

# checkout 到历史
[root@localhost repo]# ugit checkout 6fb7146829487660073b8492d2514ec16d4332c1
[root@localhost repo]# cat .ugit/HEAD 
6fb7146829487660073b8492d2514ec16d4332c1 
[root@localhost repo]# ugit log
[commit] 6fb7146829487660073b8492d2514ec16d4332c1

[message]:
    1
==========================================================
[commit] 417b2893adf209c6f2f862ab1c969dac8b069f49

[message]:
    dog1
==========================================================

# 在旧的历史上进行新的提交
[root@localhost repo]# ugit commit -m "new"
blob a45ffde12eabc9c8a9716582ac7f41015d9d5dbb ./cat
blob 0f84d81a999a9768f185ab81862399251d063995 ./dog
blob 5929d7e3b37d56bdd228ca72ce06015c71a03e27 ./others/people
tree e81ec10a8d62334756ea04307dfe81ed25919993 ./others
tree 91c4a8227c4a3356adc12ce1521020dcfe6b0ac7 .
58fdb826bcd2cad82bb075812fa14808439230f1
[root@localhost repo]# ugit log
[commit] 58fdb826bcd2cad82bb075812fa14808439230f1

[message]:
    new
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

我们 checkout 到 6fb7146829487660073b8492d2514ec16d4332c1 提交，checkout 之后我们可以看到 HEAD 指向了这个提交，然后我们进行一次新的提交并查看日志。



---

# tag

现在我们有了分支历史，我们有一些需要跟踪的 oid。假设我们有两个分支：

```text
o-----o-----o-----o-----@-----@-----@
^                  \                ^
first commit        ----$-----$     6c9f80a187ba39b4...
                              ^
                              d8d43b0e3a21df0c...
```

如果我们想通过 checkout 在两个分支之间来回切换，我们需要记住两个 oid，它们都相当长，接下来让我们实现一个命令，将一个名字附加到一个 OID 上。然后我们就可以用这个名字来代替 OID 了。 最终结果如下所示：

```sh
$ # Make some changes
...
$ ugit commit
d8d43b0e3a21df0c845e185d08be8e4028787069
$ ugit tag my-cool-commit d8d43b0e3a21df0c845e185d08be8e4028787069
$ # Make more changes
...
$ ugit commit
e549f09bbd08a8a888110b07982952e17e8c9669

$ ugit checkout my-cool-commit
        or
$ ugit checkout d8d43b0e3a21df0c845e185d08be8e4028787069
```

最后两个命令是等价的，因为 my-cool-commit 是一个指向 d8d43b0e3a21df0c845e185d08be8e4028787069 的标记。

我们将用几个步骤中实现这一点。第一步是创建一个命令行界面命令，调用 base 模块中的相关命令。base 模块在这个阶段不做任何事情。

## 添加命令

修改代码如下：

```diff
diff -u ugit.bak/base.py ugit/base.py
--- ugit.bak/base.py    2022-01-30 17:09:38.359702339 +0800
+++ ugit/base.py        2022-01-30 17:23:01.267893676 +0800
@@ -106,6 +106,10 @@
     data.set_HEAD(oid)
 
 
+def create_tag(name, oid):
+    pass
+
+
 Commit = namedtuple('Commit', ['tree', 'parent', 'message'])
 def get_commit(oid):
     parent = None
 
 
===========================================================
diff -u ugit.bak/cli.py ugit/cli.py
--- ugit.bak/cli.py     2022-01-30 17:09:38.359702339 +0800
+++ ugit/cli.py 2022-01-30 17:22:16.065808906 +0800
@@ -48,6 +48,11 @@
     checkout_parser.set_defaults(func=checkout)
     checkout_parser.add_argument('oid')
 
+    tag_parser = commands.add_parser('tag')
+    tag_parser.set_defaults(func=tag)
+    tag_parser.add_argument('name')
+    tag_parser.add_argument('oid', nargs='?')
+
     return parser.parse_args()
 
 
@@ -95,6 +100,11 @@
     base.checkout(args.oid)
 
 
+def tag(args):
+    oid = args.oid or data.get_HEAD()
+    base.create_tag(args.name, oid)
+
+
 if __name__ == "__main__":
     main()
```



## 将HEAD推广到refs

作为实现 tag 的一部分，我们将概括处理 HEAD 的方式。仔细想想，HEAD 和 tag 差不多，它们都是 ugit 为 OID 命名的方法。如果是 HEAD，名称由 ugit 硬编码；如果是 tag，名称将由用户提供。

请注意，我们在这里没有改变 ugit 的任何行为，这纯粹是重构。

```diff
diff -u ugit.bak/data.py ugit/data.py
--- ugit.bak/data.py    2022-01-30 17:25:12.908064372 +0800
+++ ugit/data.py        2022-01-30 17:44:38.752423174 +0800
@@ -11,13 +11,21 @@
 
 
 def set_HEAD(oid):
-    with open(f'{GIT_DIR}/HEAD', 'w') as f:
-        f.write(oid)
+    update_ref('HEAD', oid)
 
 
 def get_HEAD():
-    if os.path.isfile(f'{GIT_DIR}/HEAD'):
-        with open(f'{GIT_DIR}/HEAD') as f:
+    return get_ref('HEAD')
+
+
+def update_ref(ref, oid):
+    with open(f'{GIT_DIR}/{ref}', 'w') as f:
+        f.write(oid)
+
+
+def get_ref(ref):
+    if os.path.isfile(f'{GIT_DIR}/{ref}'):
+        with open(f'{GIT_DIR}/{ref}') as f:
             return f.read().strip()
```



## tag ref

我们已经在前面的更改中实现了 ref，是时候在用户创建 tag 时创建 ref 了。

create_tag 调用 update_ref 来实际创建标记。

出于命名空间的目的，我们将把所有标签放在 refs/tags/ 下。也就是说，如果用户创建了 my-cool-commit 标记，我们将创建 refs/tags/my-cool-commit 引用来指向所需的 OID。

代码修改如下：

```diff
diff -u ugit.bak/base.py ugit/base.py
--- ugit.bak/base.py    2022-01-30 20:16:36.299467489 +0800
+++ ugit/base.py        2022-01-30 20:27:19.872773281 +0800
@@ -107,7 +107,7 @@
 
 
 def create_tag(name, oid):
-    pass
+    data.update_ref(f'{data.TAG_PREFIX}/{name}', oid)
 
 
 Commit = namedtuple('Commit', ['tree', 'parent', 'message'])



===========================================================
diff -u ugit.bak/data.py ugit/data.py
--- ugit.bak/data.py    2022-01-30 20:16:36.299467489 +0800
+++ ugit/data.py        2022-01-30 20:32:23.412851708 +0800
@@ -3,6 +3,7 @@
 
 GIT_DIR = ".ugit"
 OBJECTS_DIR = GIT_DIR + "/objects"
+TAG_PREFIX = "refs/tags"
 
 
 def init():
@@ -19,13 +20,16 @@
 
 
 def update_ref(ref, oid):
-    with open(f'{GIT_DIR}/{ref}', 'w') as f:
+    ref_path = f'{GIT_DIR}/{ref}'
+    os.makedirs(os.path.dirname(ref_path), exist_ok=True)
+    with open(ref_path, 'w') as f:
         f.write(oid)
 
 
 def get_ref(ref):
-    if os.path.isfile(f'{GIT_DIR}/{ref}'):
-        with open(f'{GIT_DIR}/{ref}') as f:
+    ref_path = f'{GIT_DIR}/{ref}'
+    if os.path.isfile(ref_path):
+        with open(ref_path) as f:
             return f.read().strip()
```

测试：

```sh
[root@localhost repo]# ugit tag test
[root@localhost repo]# cat .ugit/HEAD 
58fdb826bcd2cad82bb075812fa14808439230f1
[root@localhost repo]# cat .ugit/
HEAD     objects/ refs/    
[root@localhost repo]# cat .ugit/refs/tags/test 
58fdb826bcd2cad82bb075812fa14808439230f1
```



## 参数解析引用

我们可以创建 tag 了，这很好，但是现在让我们从命令行界面实际使用它们。

在 base.py 中，我们将创建 get_oid 来将 name 解析为 oid。名称可以是引用（在这种情况下 get_oid 将返回引用所指向的 oid）或 OID（在这种情况下 get_oid 将只返回相同的 OID)。

接下来，我们将修改 cli.py 中的参数解析器，对所有预期为 oid 的参数调用 get_oid。这样我们就可以在那里传 ref 而不是 OID。 完成之后我们可以做类似以下的事情：

```sh
$ ugit tag mytag d8d43b0e3a21df0c845e185d08be8e4028787069
$ ugit log refs/tags/mytag
# Will print log of commits starting at d8d43b0e...
$ ugit checkout refs/tags/mytag
# Will checkout commit d8d43b0e...
etc...
```

修改代码如下：

```diff
diff -u ugit.bak/base.py ugit/base.py
--- ugit.bak/base.py    2022-01-30 20:43:51.325865108 +0800
+++ ugit/base.py        2022-01-30 20:45:02.190989421 +0800
@@ -129,6 +129,10 @@
     return Commit(tree=tree, parent=parent, message=message)
 
 
+def get_oid(name):
+    return data.get_ref(name) or name
+
+
 def is_ignored(path):
     return '.ugit' in path.split("/")



===========================================================
diff -u ugit.bak/cli.py ugit/cli.py
--- ugit.bak/cli.py     2022-01-30 20:43:51.325865108 +0800
+++ ugit/cli.py 2022-01-30 20:48:12.223792456 +0800
@@ -18,6 +18,8 @@
     commands = parser.add_subparsers(dest="command")
     commands.required = True
 
+    oid = base.get_oid
+
     init_parser = commands.add_parser("init")
     init_parser.set_defaults(func=init)
 
@@ -27,14 +29,14 @@
 
     cat_file_parser = commands.add_parser('cat-file')
     cat_file_parser.set_defaults(func=cat_file)
-    cat_file_parser.add_argument('object')
+    cat_file_parser.add_argument('object', type=oid)
 
     write_tree_parser = commands.add_parser('write-tree')
     write_tree_parser.set_defaults(func=write_tree)
 
     read_tree_parser = commands.add_parser('read-tree')
     read_tree_parser.set_defaults(func=read_tree)
-    read_tree_parser.add_argument('tree')
+    read_tree_parser.add_argument('tree', type=oid)
 
     commit_parser = commands.add_parser('commit')
     commit_parser.set_defaults(func=commit)
@@ -42,16 +44,16 @@
 
     log_parser = commands.add_parser('log')
     log_parser.set_defaults(func=log)
-    log_parser.add_argument('oid', nargs='?')
+    log_parser.add_argument('oid', nargs='?', type=oid)
 
     checkout_parser = commands.add_parser('checkout')
     checkout_parser.set_defaults(func=checkout)
-    checkout_parser.add_argument('oid')
+    checkout_parser.add_argument('oid', type=oid)
 
     tag_parser = commands.add_parser('tag')
     tag_parser.set_defaults(func=tag)
     tag_parser.add_argument('name')
-    tag_parser.add_argument('oid', nargs='?')
+    tag_parser.add_argument('oid', nargs='?', type=oid)
 
     return parser.parse_args()
```

测试效果：

```sh
[root@localhost repo]# ugit tag test2 refs/tags/test
[root@localhost repo]# cat .ugit/refs/tags/test2 
58fdb826bcd2cad82bb075812fa14808439230f1 
[root@localhost repo]# ugit log refs/tags/test2
[commit] 58fdb826bcd2cad82bb075812fa14808439230f1

[message]:
    new
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



## 优化引用名

在前面的更改中，您可能已经注意到我们需要拼写出标签的全名如 refs/tags/mytag。这不是很方便，我们希望有更短的命令名。例如，如果我们已经创建了 mytag 标签，我们应该能够执行 ugit log mytag，而不是必须指定 ugit log refs/tags/mytag。我们将扩展 get_oid 以在解析名称时搜索不同的 ref 子目录。我们将搜索：

- root（.ugit）：这样我们就可以指定 refs/tags/mytag
- .ugit/refs：这样我们就可以指定 tags/mytag
- .ugit/refs/tags：这样我们就可以指定 mytag
- .ugit/refs/heads：以后用

如果我们在任何目录中找到请求的名称，就返回它。否则假设名字是 OID。

修改代码如下：

```diff
diff -u ugit.bak/base.py ugit/base.py
--- ugit.bak/base.py    2022-01-30 23:05:52.561354546 +0800
+++ ugit/base.py        2022-01-30 23:11:31.624950494 +0800
@@ -1,6 +1,7 @@
 import itertools
 import operator
 import os
+import string
 
 from collections import namedtuple
 
@@ -130,7 +131,23 @@
 
 
 def get_oid(name):
-    return data.get_ref(name) or name
+    # Name is ref
+    refs_to_try = [
+        f'{name}',
+        f'refs/{name}',
+        f'refs/tags/{name}',
+        f'refs/heads/{name}',
+    ]
+    for ref in refs_to_try:
+        if data.get_ref(ref):
+            return data.get_ref(ref)
+
+    # Name is SHA1
+    is_hex = all(c in string.hexdigits for c in name)
+    if len(name) == 40 and is_hex:
+        return name
+
+    assert False, f'Unknown name {name}'
 
 
 def is_ignored(path):
```

测试：

```sh
[root@localhost repo]# ugit log
[commit] 58fdb826bcd2cad82bb075812fa14808439230f1

[message]:
    new
==========================================================
[commit] 6fb7146829487660073b8492d2514ec16d4332c1

[message]:
    1
==========================================================
[commit] 417b2893adf209c6f2f862ab1c969dac8b069f49

[message]:
    dog1
==========================================================
[root@localhost repo]# ugit tag dog_tag 417b2893adf209c6f2f862ab1c969dac8b069f49
[root@localhost repo]# ugit log dog_tag
[commit] 417b2893adf209c6f2f862ab1c969dac8b069f49

[message]:
    dog1
==========================================================
[root@localhost repo]# ugit log tags/dog_tag
[commit] 417b2893adf209c6f2f862ab1c969dac8b069f49

[message]:
    dog1
==========================================================
[root@localhost repo]# ugit log refs/tags/dog_tag
[commit] 417b2893adf209c6f2f862ab1c969dac8b069f49

[message]:
    dog1
==========================================================
```



## 在argparse中默认传递HEAD

首先，让 `@` 成为 HEAD 的别名。(在 get_oid 中实现)

其次，在 cli.py 中做一点重构。有些命令接受一个可选的 OID 参数，如果没有提供该参数，则默认为 HEAD。例如，ugit log 可以从一个 OID 开始输出日志，但默认情况下，它会从 HEAD 开始。

与其让每个命令实现这个逻辑，不如让 @ (HEAD) 成为这些命令的默认值。这个阶段的相关命令是 log 和 tag。接下来会有更多。

修改代码如下：

```diff
diff -u ugit.bak/base.py ugit/base.py
--- ugit.bak/base.py    2022-01-30 23:15:49.957516097 +0800
+++ ugit/base.py        2022-01-30 23:28:57.594453133 +0800
@@ -131,6 +131,9 @@
 
 
 def get_oid(name):
+    if name == '@':
+        name = 'HEAD'
+
     # Name is ref
     refs_to_try = [
         f'{name}',
         
         
===========================================================
diff -u ugit.bak/cli.py ugit/cli.py
--- ugit.bak/cli.py     2022-01-30 23:15:49.957516097 +0800
+++ ugit/cli.py 2022-01-30 23:31:50.225659128 +0800
@@ -44,7 +44,7 @@
 
     log_parser = commands.add_parser('log')
     log_parser.set_defaults(func=log)
-    log_parser.add_argument('oid', nargs='?', type=oid)
+    log_parser.add_argument('oid', nargs='?', type=oid, default='@')
 
     checkout_parser = commands.add_parser('checkout')
     checkout_parser.set_defaults(func=checkout)
@@ -53,7 +53,7 @@
     tag_parser = commands.add_parser('tag')
     tag_parser.set_defaults(func=tag)
     tag_parser.add_argument('name')
-    tag_parser.add_argument('oid', nargs='?', type=oid)
+    tag_parser.add_argument('oid', nargs='?', type=oid, default='@')
 
     return parser.parse_args()
 
@@ -86,7 +86,7 @@
 
 
 def log(args):
-    oid = args.oid or data.get_HEAD()
+    oid = args.oid
     while oid:
         commit = base.get_commit(oid)
 
@@ -103,7 +103,7 @@
 
 
 def tag(args):
-    oid = args.oid or data.get_HEAD()
+    oid = args.oid
     base.create_tag(args.name, oid)
```



---

# 可视化

## 打印refs

现在我们有了 refs 和分支提交历史，创建一个可视化工具来查看我们造成的所有混乱是一个好主意。

可视化工具将绘制所有引用和引用所指向的所有提交。

我们运行该工具的命令将被称为 ugit k，类似于gitk（它是 git 的图形可视化工具）。

我们将在 cli.py 中创建一个新的 k 命令。我们将创建 iter_refs，这是一个生成器，它将迭代所有可用的 refs 包括 HEAD。作为第一步，让我们在运行 k 时打印所有 refs。

修改代码如下：

```diff
diff -u ugit.bak/cli.py ugit/cli.py
--- ugit.bak/cli.py     2022-01-31 01:41:00.871552987 +0800
+++ ugit/cli.py 2022-01-31 01:45:01.849096883 +0800
@@ -55,6 +55,9 @@
     tag_parser.add_argument('name')
     tag_parser.add_argument('oid', nargs='?', type=oid, default='@')
 
+    k_parser = commands.add_parser('k')
+    k_parser.set_defaults(func=k)
+
     return parser.parse_args()
 
 
@@ -107,6 +110,12 @@
     base.create_tag(args.name, oid)
 
 
+def k(args):
+    for refname, ref in data.iter_refs():
+        print(refname, ref)
+    # TODO visualize refs
+
+
 if __name__ == "__main__":
     main()



===========================================================
diff -u ugit.bak/data.py ugit/data.py
--- ugit.bak/data.py    2022-01-31 01:41:00.871552987 +0800
+++ ugit/data.py        2022-01-31 01:49:22.551955721 +0800
@@ -33,6 +33,16 @@
             return f.read().strip()
 
 
+def iter_refs():
+    refs = ['HEAD']
+    for root, _, filenames in os.walk(f'{GIT_DIR}/refs/'):
+        root = os.path.relpath(root, GIT_DIR)
+        refs.extend(f'{root}/{name}' for name in filenames)
+
+    for refname in refs:
+        yield refname, get_ref(refname)
+
+
 def hash_object(data, type_="blob"):
     obj = type_.encode() + b"\x00" + data
     oid = hashlib.sha1(obj).hexdigest()
```

测试：

```sh
[root@localhost repo]# ugit k
HEAD 58fdb826bcd2cad82bb075812fa14808439230f1
refs/tags/test 58fdb826bcd2cad82bb075812fa14808439230f1
refs/tags/test2 58fdb826bcd2cad82bb075812fa14808439230f1
refs/tags/dog_tag 417b2893adf209c6f2f862ab1c969dac8b069f49
```



## 打印历史提交

除了打印 refs，我们还将打印所有可从这些 refs 可达的 commit oid。

iter_commits_and_parents 它是一个生成器，它可以从给定的一组 oid 回溯可到达的所有提交。有些提交可以从多个 refs 追溯到，但我们设计每个提交只会出现一次。

修改代码如下：

```diff
diff -u ugit.bak/base.py ugit/base.py
--- ugit.bak/base.py    2022-01-31 02:00:09.379397178 +0800
+++ ugit/base.py        2022-01-31 02:04:50.254986975 +0800
@@ -130,6 +130,22 @@
     return Commit(tree=tree, parent=parent, message=message)
 
 
+
+def iter_commits_and_parents(oids):
+    oids = set(oids)
+    visited = set()
+
+    while oids:
+        oid = oids.pop()
+        if not oid or oid in visited:
+            continue
+        visited.add(oid)
+        yield oid
+
+        commit = get_commit(oid)
+        oids.add(commit.parent)
+
+
 def get_oid(name):
     if name == '@':
         name = 'HEAD'
         
         
===========================================================
diff -u ugit.bak/cli.py ugit/cli.py
--- ugit.bak/cli.py     2022-01-31 02:00:09.379397178 +0800
+++ ugit/cli.py 2022-01-31 02:02:34.098522193 +0800
@@ -111,10 +111,17 @@
 
 
 def k(args):
+    oids = set()
     for refname, ref in data.iter_refs():
         print(refname, ref)
-    # TODO visualize refs
+        oids.add(ref)
 
+    for oid in base.iter_commits_and_parents(oids):
+        commit = base.get_commit(oid)
+        print(oid)
+        if commit.parent:
+            print('Parent', commit.parent)
+    # TODO visualize refs
 
 if __name__ == "__main__":
     main()
```

这里我重新建立了仓库进行测试：

```sh
[root@localhost ~]# mkdir repo
[root@localhost ~]# cd repo/
[root@localhost repo]# ugit init
Initialized empty ugit repository in /root/repo/.ugit
[root@localhost repo]# echo 1 > 1
[root@localhost repo]# ugit commit -m "1"
blob c928f7711483160a5245c9da863f775563cb3584 ./1
tree 2caac71af89baeff576d46248095715b3aa6b9be .
b0822a5cad0797b08614fdf6685fdf8703f3f5a6
[root@localhost repo]# echo 2 > 2 ; ugit commit -m "2"
blob c928f7711483160a5245c9da863f775563cb3584 ./1
blob 4bc9dea33de5dcb1b375f7e90cc3ded4228af369 ./2
tree b32a2c27bc4d6e848b70805e1a128ef534c474b5 .
defea7af40bbd2072b28f737cc0a91b4ca1a6f41
[root@localhost repo]# echo 3 > 3 ; ugit commit -m "3"
blob c928f7711483160a5245c9da863f775563cb3584 ./1
blob 4bc9dea33de5dcb1b375f7e90cc3ded4228af369 ./2
blob 02e4777f2f664a8cbf9319fad4cef9bce967b77a ./3
tree 5c1e39bb87f58df090ea9fe8818c79dda2198033 .
a8b593bb5fac66e68d2f4ecf0677d1a7e56662d4
[root@localhost repo]# ugit tag 3
[root@localhost repo]# ugit checkout defea7af40bbd2072b28f737cc0a91b4ca1a6f41
[root@localhost repo]# echo 4 > 4 ; ugit commit -m "4"
blob c928f7711483160a5245c9da863f775563cb3584 ./1
blob 4bc9dea33de5dcb1b375f7e90cc3ded4228af369 ./2
blob 9b76b5b7e33b52e1c7c3ee63ca39cb415990c556 ./4
tree 1d6295caa5f6b5cac101098609b288a6f9cc0c74 .
c930a86962c2eb5ee708dad01658f3540cf88d9b
```

我们画个示意图：

```text
b0822a5c<----defea7af<----a8b593bb
^            \                ^
first commit  -<--c930a869    refs/tags/3
                     ^
                     HEAD
```

我们使用 ugit k 来打印上面能追溯到的所有提交 oid。

```sh
[root@localhost repo]# ugit k
HEAD c930a86962c2eb5ee708dad01658f3540cf88d9b
refs/tags/3 a8b593bb5fac66e68d2f4ecf0677d1a7e56662d4
c930a86962c2eb5ee708dad01658f3540cf88d9b
Parent defea7af40bbd2072b28f737cc0a91b4ca1a6f41
a8b593bb5fac66e68d2f4ecf0677d1a7e56662d4
Parent defea7af40bbd2072b28f737cc0a91b4ca1a6f41
defea7af40bbd2072b28f737cc0a91b4ca1a6f41
Parent b0822a5cad0797b08614fdf6685fdf8703f3f5a6
b0822a5cad0797b08614fdf6685fdf8703f3f5a6
```



## 画图

k 应该是一个可视化工具，但是到目前为止，我们只是打印了一堆 oid...现在是可视化部分！

这里要使用第三方工具。

```sh
yum install 'graphviz*' -y
```

代码修改如下：

```diff
diff -u ugit.bak/cli.py ugit/cli.py
--- ugit.bak/cli.py     2022-01-31 02:33:31.613814126 +0800
+++ ugit/cli.py 2022-01-31 02:41:09.071489698 +0800
@@ -2,6 +2,7 @@
 import os
 import sys
 import textwrap
+import subprocess
 
 from . import base
 from . import data
@@ -111,17 +112,27 @@
 
 
 def k(args):
+    dot = 'digraph commits {\n'
+
     oids = set()
     for refname, ref in data.iter_refs():
-        print(refname, ref)
+        dot += f'"{refname}" [shape=note]\n'
+        dot += f'"{refname}" -> "{ref}"\n'
         oids.add(ref)
 
     for oid in base.iter_commits_and_parents(oids):
         commit = base.get_commit(oid)
-        print(oid)
+        dot += f'"{oid}" [shape=box style=filled label="{oid[:10]}"]\n'
         if commit.parent:
-            print('Parent', commit.parent)
-    # TODO visualize refs
+            dot += f'"{oid}" -> "{commit.parent}"\n'
+
+    dot += '}'
+    print(dot)
+
+    with subprocess.Popen(
+            ['dot', '-Tgtk', '/dev/stdin'],
+            stdin=subprocess.PIPE) as proc:
+        proc.communicate(dot.encode())
 
 if __name__ == "__main__":
     main()
```





---

# log

提前重构！我们现在有了 k 的 iter_commits_and_parents 函数，让我们也在 log 中使用这个函数。我们需要稍微调整一下，使用 collections.deque 而不是 set，以便提交的顺序是确定的。

这种重构似乎是不必要的，但它在以后会很有用。当我们实现具有多个父级的合并提交时，这种通用的迭代方式将会派上用场。

代码修改如下：

```diff
diff -u ugit.bak/base.py ugit/base.py
--- ugit.bak/base.py    2022-01-31 02:57:43.369082077 +0800
+++ ugit/base.py        2022-01-31 03:15:18.006119370 +0800
@@ -3,7 +3,7 @@
 import os
 import string
 
-from collections import namedtuple
+from collections import namedtuple, deque
 
 from . import data
 
@@ -132,18 +132,19 @@
 
 
 def iter_commits_and_parents(oids):
-    oids = set(oids)
+    oids = deque(oids)
     visited = set()
 
     while oids:
-        oid = oids.pop()
+        oid = oids.popleft()
         if not oid or oid in visited:
             continue
         visited.add(oid)
         yield oid
 
         commit = get_commit(oid)
-        oids.add(commit.parent)
+        # Return parent next
+        oids.appendleft(commit.parent)
 
 
 def get_oid(name):
 
 
===========================================================
diff -u ugit.bak/cli.py ugit/cli.py
--- ugit.bak/cli.py     2022-01-31 02:57:43.369082077 +0800
+++ ugit/cli.py 2022-01-31 03:10:55.325109151 +0800
@@ -90,8 +90,7 @@
 
 
 def log(args):
-    oid = args.oid
-    while oid:
+    for oid in base.iter_commits_and_parents({args.oid}):
         commit = base.get_commit(oid)
 
         print(f'[commit] {oid}\n')
@@ -99,8 +98,6 @@
         print(textwrap.indent(commit.message, '    '))
         print('=' * 58)
 
-        oid = commit.parent
-
 
 def checkout(args):
     base.checkout(args.oid)
```





---

# branch

## 命令实现

tag 是一种进步，因为它将我们从直接记住 oid 的负担中解放出来。但是它仍然有些不方便，因为它们是静态的。让我举例说明：

```text
o-----o-----o-----o-----o-----o-----o
                   \                ^
                    ----o-----o  tag2,HEAD
                              ^
                           tag1
```

如果我们有上述情况，我们可以很容易地通过 checkout 在 tag1 和 tag2 之间切换。但是如果我们这样做了会发生什么呢?

- ugit checkout tag2
- Make some changes
- ugit commit?

现在看起来是这样的：

```text
o-----o-----o-----o-----o-----o-----o-----o
                   \                ^     ^
                    ----o-----o  tag2     HEAD
                              ^
                           tag1
```

上面的分支已经前进，但是 tag2 仍然指向前一个提交。这是设计好的，因为标签应该只命名一个特定的 OID。因此，如果我们想记住新的 HEAD 位置，我们需要创建另一个标签。但是现在让我们创建一个引用，它将随着分支的增长而前进。就像我们有 ugit tag 一样，我们将创建 ugit branch，branch 也将指向一个特定的 OID，这一次将在 refs/heads 下创建 ref。

在此阶段，branch 看起来与 tag 没有任何不同，唯一的区别是 branch 是在 refs/heads 下创建的，而不是在 refs/tags 下创建的。但是一旦我们尝试 checkout 到一个 branch，神奇的事情就会发生。

到目前为止，当我们 checkout 的时候，我们更新 HEAD 指向我们刚刚 checkout 的 OID。但是如果我们按名称 checkout 到一个分支，我们会做一些不同的事情，我们会更新 HEAD 来指向该分支的名称！假设我们这里有一个分支：

```text
o-----o-----o-----o-----o-----o-----o
                   \                ^
                    ----o-----o tag2,branch2
                              ^
                           tag1
```

运行 ugit checkout branch2 将产生以下情况：

```text
o-----o-----o-----o-----o-----o-----o
                   \                ^
                    ----o-----o tag2,branch2 <--- HEAD
                              ^
                           tag1
```

看到了吗？HEAD 指向 branch2，而不是直接指向提交的 OID。现在，如果我们创建另一个提交，ugit 将更新 HEAD 以指向最新的提交(就像每次一样)，但作为副作用，它也将更新 branch2 以指向最新的提交。

```text
o-----o-----o-----o-----o-----o-----o-----o
                   \                ^     ^
                    ----o-----o  tag2     branch2 <--- HEAD
                              ^
                           tag1
```

这样，如果我们 checkout 一个分支并在其上创建一些提交，branch 引用将总是指向最新的提交。

但是现在 HEAD 只指向 OID。它不能像上面描述的那样指向另一个引用。所以我们的下一步是实现这个概念。为了反映 Git 的术语，我们将指向另一个引用的引用称为 `symbolic ref`。symbolic refs 实现参考下一个更改。

首先增加添加分支的命令，修改代码如下：

```diff
diff -u ugit.bak/base.py ugit/base.py
--- ugit.bak/base.py    2022-01-31 05:23:52.608508223 +0800
+++ ugit/base.py        2022-01-31 05:28:46.452660627 +0800
@@ -111,6 +111,10 @@
     data.update_ref(f'{data.TAG_PREFIX}/{name}', oid)
 
 
+def create_branch(name, oid):
+    data.update_ref(f'{data.BRANCH_PREFIX}/{name}', oid)
+
+
 Commit = namedtuple('Commit', ['tree', 'parent', 'message'])
 def get_commit(oid):
     parent = None


===========================================================
diff -u ugit.bak/cli.py ugit/cli.py
--- ugit.bak/cli.py     2022-01-31 05:23:52.608508223 +0800
+++ ugit/cli.py 2022-01-31 05:26:55.606093459 +0800
@@ -56,6 +56,11 @@
     tag_parser.add_argument('name')
     tag_parser.add_argument('oid', nargs='?', type=oid, default='@')
 
+    branch_parser = commands.add_parser('branch')
+    branch_parser.set_defaults(func=branch)
+    branch_parser.add_argument('name')
+    branch_parser.add_argument('start_point', default='@', type=oid, nargs='?')
+
     k_parser = commands.add_parser('k')
     k_parser.set_defaults(func=k)
 
@@ -108,6 +113,11 @@
     base.create_tag(args.name, oid)
 
 
+def branch(args):
+    base.create_branch(args.name, args.start_point)
+    print(f'Branch {args.name} created at {args.start_point[:10]}')
+
+
 def k(args):
     dot = 'digraph commits {\n'


===========================================================
diff -u ugit.bak/data.py ugit/data.py
--- ugit.bak/data.py    2022-01-31 05:23:52.606341558 +0800
+++ ugit/data.py        2022-01-31 05:29:19.189881422 +0800
@@ -4,6 +4,7 @@
 GIT_DIR = ".ugit"
 OBJECTS_DIR = GIT_DIR + "/objects"
 TAG_PREFIX = "refs/tags"
+BRANCH_PREFIX = "refs/heads"
```



## symbolic refs

如果表示引用的文件包含 OID，我们将假设引用指向 OID。如果文件包含内容 `ref: <refname>`，我们将假设指向引用，并且我们将递归地取消引用并找到 OID。

代码修改如下：

```diff
diff -u ugit.bak/data.py ugit/data.py
--- ugit.bak/data.py    2022-01-31 05:41:33.593797100 +0800
+++ ugit/data.py        2022-01-31 05:43:34.577099837 +0800
@@ -29,9 +29,15 @@
 
 def get_ref(ref):
     ref_path = f'{GIT_DIR}/{ref}'
+    value = None
     if os.path.isfile(ref_path):
         with open(ref_path) as f:
-            return f.read().strip()
+            value = f.read().strip()
+
+    if value and value.startswith('ref:'):
+        return get_ref(value.split(':', 1)[1].strip())
+
+    return value
 
 
 def iter_refs():
```



## RefValue容器

为了使使用 symbolic refs 更容易，我们将创建一个 RefValue 容器来表示引用的值。

RefValue 将有一个符号属性，它将说明它是符号还是直接引用。

这个变化只是重构，我们将把从引用中写入或读取的每个 OID 都包装在一个 RefValue 中。

代码修改如下：

```diff
diff -u ugit.bak/base.py ugit/base.py
--- ugit.bak/base.py    2022-01-31 05:50:36.682368975 +0800
+++ ugit/base.py        2022-01-31 06:01:30.506126281 +0800
@@ -88,7 +88,7 @@
 def commit(message):
     commit = f'tree {write_tree()}\n'
 
-    HEAD = data.get_HEAD()
+    HEAD = data.get_HEAD().value
     if HEAD:
         commit += f'parent {HEAD}\n'
 
@@ -96,7 +96,7 @@
     commit += f'{message}\n'
 
     oid = data.hash_object(commit.encode(), 'commit')
-    data.set_HEAD(oid)
+    data.set_HEAD(data.RefValue(symbolic=False, value=oid))
 
     return oid
 
@@ -104,15 +104,15 @@
 def checkout(oid):
     commit = get_commit(oid)
     read_tree(commit.tree)
-    data.set_HEAD(oid)
+    data.set_HEAD(data.RefValue(symbolic=False, value=oid))
 
 
 def create_tag(name, oid):
-    data.update_ref(f'{data.TAG_PREFIX}/{name}', oid)
+    data.update_ref(f'{data.TAG_PREFIX}/{name}', data.RefValue(symbolic=False, value=oid))
 
 
 def create_branch(name, oid):
-    data.update_ref(f'{data.BRANCH_PREFIX}/{name}', oid)
+    data.update_ref(f'{data.BRANCH_PREFIX}/{name}', data.RefValue(symbolic=False, value=oid))
 
 
 Commit = namedtuple('Commit', ['tree', 'parent', 'message'])
@@ -163,8 +163,8 @@
         f'refs/heads/{name}',
     ]
     for ref in refs_to_try:
-        if data.get_ref(ref):
-            return data.get_ref(ref)
+        if data.get_ref(ref).value:
+            return data.get_ref(ref).value
 
     # Name is SHA1
     is_hex = all(c in string.hexdigits for c in name)



===========================================================
diff -u ugit.bak/cli.py ugit/cli.py
--- ugit.bak/cli.py     2022-01-31 05:50:36.682368975 +0800
+++ ugit/cli.py 2022-01-31 06:02:15.428677384 +0800
@@ -124,8 +124,8 @@
     oids = set()
     for refname, ref in data.iter_refs():
         dot += f'"{refname}" [shape=note]\n'
-        dot += f'"{refname}" -> "{ref}"\n'
-        oids.add(ref)
+        dot += f'"{refname}" -> "{ref.value}"\n'
+        oids.add(ref.value)
 
     for oid in base.iter_commits_and_parents(oids):
         commit = base.get_commit(oid)



===========================================================
diff -u ugit.bak/data.py ugit/data.py
--- ugit.bak/data.py    2022-01-31 05:50:36.683452307 +0800
+++ ugit/data.py        2022-01-31 05:56:32.016555225 +0800
@@ -1,6 +1,8 @@
 import os
 import hashlib
 
+from collections import namedtuple
+
 GIT_DIR = ".ugit"
 OBJECTS_DIR = GIT_DIR + "/objects"
 TAG_PREFIX = "refs/tags"
@@ -11,7 +13,6 @@
     os.makedirs(GIT_DIR)
     os.makedirs(OBJECTS_DIR)
 
-
 def set_HEAD(oid):
     update_ref('HEAD', oid)
 
@@ -20,11 +21,13 @@
     return get_ref('HEAD')
 
 
-def update_ref(ref, oid):
+RefValue = namedtuple('RefValue', ['symbolic', 'value'])
+def update_ref(ref, value):
+    assert not value.symbolic
     ref_path = f'{GIT_DIR}/{ref}'
     os.makedirs(os.path.dirname(ref_path), exist_ok=True)
     with open(ref_path, 'w') as f:
-        f.write(oid)
+        f.write(value.value)
 
 
 def get_ref(ref):
@@ -37,7 +40,7 @@
     if value and value.startswith('ref:'):
         return get_ref(value.split(':', 1)[1].strip())
 
-    return value
+    return RefValue(symbolic=False, value=value)
 
 
 def iter_refs():
```

接下来我们不仅在读取的时候解引用，在写入的时候也增加此修改，修改代码如下：

```diff
diff -u ugit.bak/data.py ugit/data.py
--- ugit.bak/data.py    2022-01-31 05:50:36.683452307 +0800
+++ ugit/data.py        2022-01-31 06:24:39.838034512 +0800  
  def update_ref(ref, value):
      assert not value.symbolic
+     ref = _get_ref_internal(ref)[0]
      ref_path = f'{GIT_DIR}/{ref}'
      os.makedirs (os.path.dirname (ref_path), exist_ok=True)
    with open(ref_path, 'w') as f:
        f.write(value.value)


 def get_ref(ref):
+    return _get_ref_internal(ref)[1]
+
+
+def _get_ref_internal(ref):
     ref_path = f'{GIT_DIR}/{ref}'
     value = None
     if os.path.isfile(ref_path):
         with open(ref_path) as f:
             value = f.read().strip()
 
-    if value and value.startswith('ref:'):
-        return get_ref(value.split(':', 1)[1].strip())
+    symbolic = bool(value) and value.startswith('ref:')
+    if symbolic:
+        value = value.split(':', 1)[1].strip()
+        return _get_ref_internal(value)
 
-    return value
+    return ref, RefValue(symbolic=False, value=value)
```



## 增加解除引用的标志位

实际上，一路取消引用并不总是可取的。有时候我们想知道哪个是符号引用，而不是最终的 OID。或者我们希望直接更新引用，而不是更新链中的最后一个引用。

一个这样的用例是 ugit k，当可视化引用时，最好能看到哪个引用指向哪个引用。我们很快会看到另一个用例。

为了适应这一点，我们将为 get_ref`, `iter_refs 和 update_ref 添加一个 deref 选项。如果使用 deref=False 调用它们，它们将处理引用的原始值，而不会取消引用任何符号引用。

修改代码如下：

```diff
diff -u ugit.bak/base.py ugit/base.py
--- ugit.bak/base.py    2022-01-31 06:31:41.580566535 +0800
+++ ugit/base.py        2022-01-31 06:47:57.320987581 +0800
@@ -163,7 +163,7 @@
         f'refs/heads/{name}',
     ]
     for ref in refs_to_try:
-        if data.get_ref(ref).value:
+        if data.get_ref(ref, deref=False).value:
             return data.get_ref(ref).value
 
     # Name is SHA1


===========================================================
diff -u ugit.bak/cli.py ugit/cli.py
--- ugit.bak/cli.py     2022-01-31 06:31:41.580566535 +0800
+++ ugit/cli.py 2022-01-31 06:49:25.121850364 +0800
@@ -122,10 +122,11 @@
     dot = 'digraph commits {\n'
 
     oids = set()
-    for refname, ref in data.iter_refs():
+    for refname, ref in data.iter_refs(deref=False):
         dot += f'"{refname}" [shape=note]\n'
         dot += f'"{refname}" -> "{ref.value}"\n'
-        oids.add(ref.value)
+        if not ref.symbolic:
+            oids.add(ref.value)
 
     for oid in base.iter_commits_and_parents(oids):
         commit = base.get_commit(oid)


===========================================================
diff -u ugit.bak/data.py ugit/data.py
--- ugit.bak/data.py    2022-01-31 06:31:41.580566535 +0800
+++ ugit/data.py        2022-01-31 06:46:40.927617809 +0800
@@ -22,20 +22,20 @@
 
 
 RefValue = namedtuple('RefValue', ['symbolic', 'value'])
-def update_ref(ref, value):
+def update_ref(ref, value, deref=True):
     assert not value.symbolic
-    ref = _get_ref_internal(ref)[0]
+    ref = _get_ref_internal(ref, deref)[0]
     ref_path = f'{GIT_DIR}/{ref}'
     os.makedirs(os.path.dirname(ref_path), exist_ok=True)
     with open(ref_path, 'w') as f:
         f.write(value.value)
 
 
-def get_ref(ref):
-    return _get_ref_internal(ref)[1]
+def get_ref(ref, deref=True):
+    return _get_ref_internal(ref, deref)[1]
 
 
-def _get_ref_internal(ref):
+def _get_ref_internal(ref, deref):
     ref_path = f'{GIT_DIR}/{ref}'
     value = None
     if os.path.isfile(ref_path):
@@ -45,19 +45,20 @@
     symbolic = bool(value) and value.startswith('ref:')
     if symbolic:
         value = value.split(':', 1)[1].strip()
-        return _get_ref_internal(value)
+        if deref:
+            return _get_ref_internal(value, deref)
 
-     return ref, RefValue(symbolic=False, value=value)
+     return ref, RefValue(symbolic=symbolic, value=value)
 
 
-def iter_refs():
+def iter_refs(deref=True):
     refs = ['HEAD']
     for root, _, filenames in os.walk(f'{GIT_DIR}/refs/'):
         root = os.path.relpath(root, GIT_DIR)
         refs.extend(f'{root}/{name}' for name in filenames)
 
     for refname in refs:
-        yield refname, get_ref(refname)
+        yield refname, get_ref(refname, deref)
 
 
 def hash_object(data, type_="blob"):
```



## 写symbolic refs

在最终实现 branch 之前，我们还有一步。我们将向 update_ref 添加可以写入符号链接的代码。

注意，在之前的修改中，我们实现了向 symbolic refs 写入的能力，但是我们只能向它写入 oid。现在，我们实现将 refs 值写入 symbolic refs 的能力，我们所写的引用本身是符号的还是非符号的并不重要。这两个听起来很相似，也很混乱，但是一定要停下来，理解为什么它们是不同的东西。

修改代码如下：

```diff
diff -u ugit.bak/data.py ugit/data.py
--- ugit.bak/data.py    2022-01-31 06:55:37.171852796 +0800
+++ ugit/data.py        2022-01-31 06:56:50.803800378 +0800
@@ -23,12 +23,18 @@
 
 RefValue = namedtuple('RefValue', ['symbolic', 'value'])
 def update_ref(ref, value, deref=True):
-    assert not value.symbolic
     ref = _get_ref_internal(ref, deref)[0]
+
+    assert value.value
+    if value.symbolic:
+        value = f'ref: {value.value}'
+    else:
+        value = value.value
+
     ref_path = f'{GIT_DIR}/{ref}'
     os.makedirs(os.path.dirname(ref_path), exist_ok=True)
     with open(ref_path, 'w') as f:
-        f.write(value.value)
+        f.write(value)
```



## 切换分支

终于！在这个改变之后，如果我们 checkout branch，我们将把 HEAD 指向那个分支的引用，此时 HEAD 是符号引用。如果我们 checkout 任何其他东西例如 tag 或 OID，我们将 HEAD 直接指向提交 OID 头部，此时 HEAD 非符号引用。

我们将 refs/heads 底下每个引用都当做一个分支。

分支介绍中描述的行为现在起作用了。我们可以对一个分支进行提交，它将随着 HEAD 一起前进。如果我们有这个：

```text
o-----o-----o-----o-----o-----o-----O (3d8773...)
                   \                ^
                    ----o---o    branch1
                            ^
                         branch2
```

假设最后一次提交的 OID 是 3d8773。如果我们通过分支名称签出提交（ugit checkout branch1），HEAD 将成为指向 branch1 的符号引用。HEAD 的值将是 `ref:refs/HEAD/branch1`。

```text
o-----o-----o-----o-----o-----o-----O (3d8773...)
                   \                ^
                    ----o---o    branch1 <--- HEAD
                            ^
                         branch2
```

如果您创建了一个提交，它将与 HEAD 一起推进 branch1：

```text
o-----o-----o-----o-----o-----o-----O-----o
                   \                      ^
                    ----o---o          branch1 <--- HEAD
                            ^
                      branch2
```

请注意，我们可以 ugit checkout 3d8773，而不是 ugit checkout branch1，但这其实是另一回事！

在这两种情况下，HEAD 最终仍然指向同一个提交，但是如果我们检查 OID 的提交，HEAD 将直接指向 OID 3d8773

```text
o-----o-----o-----o-----o-----o-----O (3d8773...)
                   \                ^
                    ----o---o    branch1,HEAD
                            ^
                         branch2
```

HEAD 的物理值将为 3d8773。现在提交不会推进 branch1，只有 HEAD 会推进：

```text
o-----o-----o-----o-----o-----o-----O-----o
                   \                ^     ^
                    ----o---o    branch1  HEAD
                            ^
                         branch2
```

这种情况称为 **detached HEAD**。当你和真正的 Git 一起工作时，你可能听说过。起初这是一个令人困惑的概念，但希望现在更容易理解，因为我们已经自己构建了一切。

分离 HEAD 是一个有点危险的情况，因为如果你在分离头中工作，然后切换到另一个分支，你可能会丢失一些提交。假设你现在执行 ugit checkout branch2：

```text
o-----o-----o-----o-----o-----o-----O-----o
                   \                ^
                    ----o---o    branch1
                            ^
                         branch2 <--- HEAD
```

如您所见，没有引用指向我们之前在 branch1 上所做的最新提交。你需要知道它的 OID 才能再次找回它，这是目前的问题。所以，确保你永远不会处于 detached HEAD 状态，除非你打算这样做。

代码修改如下：

```diff
diff -u ugit.bak/base.py ugit/base.py
--- ugit.bak/base.py    2022-01-31 09:49:48.606109182 +0800
+++ ugit/base.py        2022-01-31 10:17:01.206539516 +0800
@@ -101,16 +101,27 @@
     return oid
 
 
-def checkout(oid):
+def checkout(name):
+    oid = get_oid(name)
     commit = get_commit(oid)
     read_tree(commit.tree)
-    data.set_HEAD(data.RefValue(symbolic=False, value=oid))
+
+    if is_branch(name):
+        HEAD = data.RefValue(symbolic=True, value=f'refs/heads/{name}')
+    else:
+        HEAD = data.RefValue(symbolic=False, value=oid)
+
+    data.update_ref('HEAD', HEAD, deref=False)
 
 
 def create_tag(name, oid):
     data.update_ref(f'{data.TAG_PREFIX}/{name}', data.RefValue(symbolic=False, value=oid))
 
 
+def is_branch(branch):
+    return data.get_ref(f'{data.BRANCH_PREFIX}/{branch}').value is not None
+
+
 def create_branch(name, oid):
     data.update_ref(f'{data.BRANCH_PREFIX}/{name}', data.RefValue(symbolic=False, value=oid))
 

===========================================================
diff -u ugit.bak/cli.py ugit/cli.py
--- ugit.bak/cli.py     2022-01-31 09:49:48.606109182 +0800
+++ ugit/cli.py 2022-01-31 09:55:31.627091590 +0800
@@ -49,7 +49,7 @@
 
     checkout_parser = commands.add_parser('checkout')
     checkout_parser.set_defaults(func=checkout)
-    checkout_parser.add_argument('oid', type=oid)
+    checkout_parser.add_argument('commit')
 
     tag_parser = commands.add_parser('tag')
     tag_parser.set_defaults(func=tag)
@@ -105,7 +105,7 @@
 
 
 def checkout(args):
-    base.checkout(args.oid)
+    base.checkout(args.commit)
 
 
 def tag(args):
```



## 初始化master分支

初始化仓库时我们将使 HEAD 指向 refs/heads/master，这样存储库就有了一个初始分支，称为 master，意味着主分支。这个分支可以被命名为我们想要的任何名称，但是 master 是 Git 中第一个分支的标准名称。

如果没有这个改变，一个新的 ugit 存储库将会在分离的 HEAD 中启动，但是现在我们将从一开始就有分支，以避免用户混乱。

请注意，现在我们将有两个初始化函数：一个在 base.py 中，一个在 data.py 中。

base.init() 调用 data.init()，每个函数初始化存储库的不同方面。如您所知，data.py 处理直接接触磁盘（对象数据库和引用）的所有内容，base.py 处理建立在 data.py 之上的更高级概念。因为设置初始分支是一个更高级的概念，所以它的初始化属于 base.py。

修改代码如下：

```diff
diff -u ugit.bak/base.py ugit/base.py
--- ugit.bak/base.py    2022-01-31 11:06:35.493157563 +0800
+++ ugit/base.py        2022-01-31 11:16:22.180398651 +0800
@@ -8,6 +8,11 @@
 from . import data
 
 
+def init():
+    data.init()
+    data.update_ref('HEAD', data.RefValue(symbolic=True, value=f'{data.BRANCH_PREFIX}/master'))
+
+
 def write_tree(directory="."):
     entries = []
     with os.scandir(directory) as it:


===========================================================
diff -u ugit.bak/cli.py ugit/cli.py
--- ugit.bak/cli.py     2022-01-31 11:06:35.493157563 +0800
+++ ugit/cli.py 2022-01-31 11:06:59.185634267 +0800
@@ -68,7 +68,7 @@
 
 
 def init(args):
-    data.init()
+    base.init()
     print("Initialized empty ugit repository in {}/{}".format(os.getcwd(), data.GIT_DIR))
 
```

我们来进行测试，首先建立仓库，可以看到 HEAD 指向 refs/heads/master：

```sh
[root@localhost repo]# ugit init
Initialized empty ugit repository in /root/repo/.ugit
[root@localhost repo]# cat .
./     ../    .ugit/ 
[root@localhost repo]# cat .ugit/
HEAD     objects/ 
[root@localhost repo]# cat .ugit/HEAD 
ref: refs/heads/master
```

然后我们提交几个记录，可以看到 master 分支随着提交一起移动：

```sh
[root@localhost repo]# echo 1 > 1
[root@localhost repo]# ugit commit -m "1"
blob c928f7711483160a5245c9da863f775563cb3584 ./1
tree 2caac71af89baeff576d46248095715b3aa6b9be .
b0822a5cad0797b08614fdf6685fdf8703f3f5a6   
[root@localhost repo]# cat .ugit/refs/heads/master 
b0822a5cad0797b08614fdf6685fdf8703f3f5a6
[root@localhost repo]# cat .ugit/HEAD 
ref: refs/heads/master

[root@localhost repo]# echo 2 > 2 ; ugit commit -m "2"
blob c928f7711483160a5245c9da863f775563cb3584 ./1
blob 4bc9dea33de5dcb1b375f7e90cc3ded4228af369 ./2
tree b32a2c27bc4d6e848b70805e1a128ef534c474b5 .
defea7af40bbd2072b28f737cc0a91b4ca1a6f41

[root@localhost repo]# echo 3 > 3
[root@localhost repo]# ugit commit -m "3"
blob c928f7711483160a5245c9da863f775563cb3584 ./1
blob 4bc9dea33de5dcb1b375f7e90cc3ded4228af369 ./2
blob 02e4777f2f664a8cbf9319fad4cef9bce967b77a ./3
tree 5c1e39bb87f58df090ea9fe8818c79dda2198033 .
a8b593bb5fac66e68d2f4ecf0677d1a7e56662d4

[root@localhost repo]# cat .ugit/HEAD 
ref: refs/heads/master
[root@localhost repo]# cat .ugit/refs/heads/master 
a8b593bb5fac66e68d2f4ecf0677d1a7e56662d4
```

我们建立一个 dev 分支，然后在这个分支提交记录：

```sh
[root@localhost repo]# ugit branch dev
Branch dev created at a8b593bb5f 
[root@localhost repo]# cat .ugit/refs/heads/dev 
a8b593bb5fac66e68d2f4ecf0677d1a7e56662d4
[root@localhost repo]# echo 4 > 4 ; ugit commit -m "4"
blob c928f7711483160a5245c9da863f775563cb3584 ./1
blob 4bc9dea33de5dcb1b375f7e90cc3ded4228af369 ./2
blob 02e4777f2f664a8cbf9319fad4cef9bce967b77a ./3
blob 9b76b5b7e33b52e1c7c3ee63ca39cb415990c556 ./4
tree 1e94ad054aaef781aa1a64a46ed6f1557a1b6bd6 .
df9f8446dbf6c4a3d52334d330817e169e2f66fa
[root@localhost repo]# cat .ugit/refs/heads/master 
df9f8446dbf6c4a3d52334d330817e169e2f66fa
[root@localhost repo]# cat .ugit/refs/heads/dev 
a8b593bb5fac66e68d2f4ecf0677d1a7e56662d4
```

我们看到此时 master  跟着提交在继续前进，这是因为我们虽然创建了 dev 分支，但是没有 checkout 过去，我们操作移动到 dev 分支上再执行提交操作，可以看到这个时候提交沿着 dev 分支走下去了：

```sh
[root@localhost repo]# ugit checkout dev
[root@localhost repo]# echo 5 > 5; ugit commit -m "5"
blob c928f7711483160a5245c9da863f775563cb3584 ./1
blob 4bc9dea33de5dcb1b375f7e90cc3ded4228af369 ./2
blob 02e4777f2f664a8cbf9319fad4cef9bce967b77a ./3
blob 9e3ebc18c9035a6bea6e8ad4785a286e21f7f7dc ./5
tree e2fd3b94ef65897a40e41132b7facef51bf38e03 .
f93d352f74d576d5663aba7224a5611722b965ea
[root@localhost repo]# ugit log
[commit] f93d352f74d576d5663aba7224a5611722b965ea

[message]:
    5
==========================================================
[commit] a8b593bb5fac66e68d2f4ecf0677d1a7e56662d4

[message]:
    3
==========================================================
[commit] defea7af40bbd2072b28f737cc0a91b4ca1a6f41

[message]:
    2
==========================================================
[commit] b0822a5cad0797b08614fdf6685fdf8703f3f5a6

[message]:
    1
==========================================================
[root@localhost repo]# ugit checkout master
[root@localhost repo]# ugit log
[commit] df9f8446dbf6c4a3d52334d330817e169e2f66fa

[message]:
    4
==========================================================
[commit] a8b593bb5fac66e68d2f4ecf0677d1a7e56662d4

[message]:
    3
==========================================================
[commit] defea7af40bbd2072b28f737cc0a91b4ca1a6f41

[message]:
    2
==========================================================
[commit] b0822a5cad0797b08614fdf6685fdf8703f3f5a6

[message]:
    1
==========================================================
```

以下是目前为止完整的代码：

- ugit/cli.py

```python
import argparse
import os
import sys
import textwrap
import subprocess

from . import base
from . import data


def main():
    args = parse_args()
    args.func(args)


def parse_args():
    parser = argparse.ArgumentParser()

    commands = parser.add_subparsers(dest="command")
    commands.required = True

    oid = base.get_oid

    init_parser = commands.add_parser("init")
    init_parser.set_defaults(func=init)

    hash_object_parser = commands.add_parser('hash-object')
    hash_object_parser.set_defaults(func=hash_object)
    hash_object_parser.add_argument('file')

    cat_file_parser = commands.add_parser('cat-file')
    cat_file_parser.set_defaults(func=cat_file)
    cat_file_parser.add_argument('object', type=oid)

    write_tree_parser = commands.add_parser('write-tree')
    write_tree_parser.set_defaults(func=write_tree)

    read_tree_parser = commands.add_parser('read-tree')
    read_tree_parser.set_defaults(func=read_tree)
    read_tree_parser.add_argument('tree', type=oid)

    commit_parser = commands.add_parser('commit')
    commit_parser.set_defaults(func=commit)
    commit_parser.add_argument('-m', '--message', required=True)

    log_parser = commands.add_parser('log')
    log_parser.set_defaults(func=log)
    log_parser.add_argument('oid', nargs='?', type=oid, default='@')

    checkout_parser = commands.add_parser('checkout')
    checkout_parser.set_defaults(func=checkout)
    checkout_parser.add_argument('commit')

    tag_parser = commands.add_parser('tag')
    tag_parser.set_defaults(func=tag)
    tag_parser.add_argument('name')
    tag_parser.add_argument('oid', nargs='?', type=oid, default='@')

    branch_parser = commands.add_parser('branch')
    branch_parser.set_defaults(func=branch)
    branch_parser.add_argument('name')
    branch_parser.add_argument('start_point', default='@', type=oid, nargs='?')

    k_parser = commands.add_parser('k')
    k_parser.set_defaults(func=k)

    return parser.parse_args()


def init(args):
    base.init()
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
    for oid in base.iter_commits_and_parents({args.oid}):
        commit = base.get_commit(oid)

        print(f'[commit] {oid}\n')
        print('[message]:')
        print(textwrap.indent(commit.message, '    '))
        print('=' * 58)


def checkout(args):
    base.checkout(args.commit)


def tag(args):
    oid = args.oid
    base.create_tag(args.name, oid)


def branch(args):
    base.create_branch(args.name, args.start_point)
    print(f'Branch {args.name} created at {args.start_point[:10]}')


def k(args):
    dot = 'digraph commits {\n'

    oids = set()
    for refname, ref in data.iter_refs(deref=False):
        dot += f'"{refname}" [shape=note]\n'
        dot += f'"{refname}" -> "{ref.value}"\n'
        if not ref.symbolic:
            oids.add(ref.value)

    for oid in base.iter_commits_and_parents(oids):
        commit = base.get_commit(oid)
        dot += f'"{oid}" [shape=box style=filled label="{oid[:10]}"]\n'
        if commit.parent:
            dot += f'"{oid}" -> "{commit.parent}"\n'

    dot += '}'
    print(dot)

    with subprocess.Popen(
            ['dot', '-Tgtk', '/dev/stdin'],
            stdin=subprocess.PIPE) as proc:
        proc.communicate(dot.encode())

if __name__ == "__main__":
    main()

```



- ugit/base.py

```python
import itertools
import operator
import os
import string

from collections import namedtuple, deque

from . import data


def init():
    data.init()
    data.update_ref('HEAD', data.RefValue(symbolic=True, value=f'{data.BRANCH_PREFIX}/master'))


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

    HEAD = data.get_ref('HEAD').value
    if HEAD:
        commit += f'parent {HEAD}\n'

    commit += '\n'
    commit += f'{message}\n'

    oid = data.hash_object(commit.encode(), 'commit')
    data.update_ref('HEAD', data.RefValue(symbolic=False, value=oid))

    return oid


def checkout(name):
    oid = get_oid(name)
    commit = get_commit(oid)
    read_tree(commit.tree)

    if is_branch(name):
        HEAD = data.RefValue(symbolic=True, value=f'refs/heads/{name}')
    else:
        HEAD = data.RefValue(symbolic=False, value=oid)

    data.update_ref('HEAD', HEAD, deref=False)


def create_tag(name, oid):
    data.update_ref(f'{data.TAG_PREFIX}/{name}', data.RefValue(symbolic=False, value=oid))


def is_branch(branch):
    return data.get_ref(f'{data.BRANCH_PREFIX}/{branch}').value is not None


def create_branch(name, oid):
    data.update_ref(f'{data.BRANCH_PREFIX}/{name}', data.RefValue(symbolic=False, value=oid))


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



def iter_commits_and_parents(oids):
    oids = deque(oids)
    visited = set()

    while oids:
        oid = oids.popleft()
        if not oid or oid in visited:
            continue
        visited.add(oid)
        yield oid

        commit = get_commit(oid)
        # Return parent next
        oids.appendleft(commit.parent)


def get_oid(name):
    if name == '@':
        name = 'HEAD'

    # Name is ref
    refs_to_try = [
        f'{name}',
        f'refs/{name}',
        f'refs/tags/{name}',
        f'refs/heads/{name}',
    ]
    for ref in refs_to_try:
        if data.get_ref(ref, deref=False).value:
            return data.get_ref(ref).value

    # Name is SHA1
    is_hex = all(c in string.hexdigits for c in name)
    if len(name) == 40 and is_hex:
        return name

    assert False, f'Unknown name {name}'


def is_ignored(path):
    return '.ugit' in path.split("/")

```



- ugit/data.py

```python
import os
import hashlib

from collections import namedtuple

GIT_DIR = ".ugit"
OBJECTS_DIR = GIT_DIR + "/objects"
TAG_PREFIX = "refs/tags"
BRANCH_PREFIX = "refs/heads"


def init():
    os.makedirs(GIT_DIR)
    os.makedirs(OBJECTS_DIR)


RefValue = namedtuple('RefValue', ['symbolic', 'value'])
def update_ref(ref, value, deref=True):
    ref = _get_ref_internal(ref, deref)[0]

    assert value.value
    if value.symbolic:
        value = f'ref: {value.value}'
    else:
        value = value.value

    ref_path = f'{GIT_DIR}/{ref}'
    os.makedirs(os.path.dirname(ref_path), exist_ok=True)
    with open(ref_path, 'w') as f:
        f.write(value)


def get_ref(ref, deref=True):
    return _get_ref_internal(ref, deref)[1]


def _get_ref_internal(ref, deref):
    ref_path = f'{GIT_DIR}/{ref}'
    value = None
    if os.path.isfile(ref_path):
        with open(ref_path) as f:
            value = f.read().strip()

    symbolic = bool(value) and value.startswith('ref:')
    if symbolic:
        value = value.split(':', 1)[1].strip()
        if deref:
            return _get_ref_internal(value, deref)

    return ref, RefValue(symbolic=symbolic, value=value)


def iter_refs(deref=True):
    refs = ['HEAD']
    for root, _, filenames in os.walk(f'{GIT_DIR}/refs/'):
        root = os.path.relpath(root, GIT_DIR)
        refs.extend(f'{root}/{name}' for name in filenames)

    for refname in refs:
        yield refname, get_ref(refname, deref)


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



## 打印当前分支名

让我们创建一个新的命令行界面命令 status。通常，这个命令将打印关于我们工作目录的有用信息。

现在它将打印当前的分支名称。在后面的更改中，我们将打印更多有趣的信息，例如哪些文件被更改。

修改代码如下：

```diff
diff -u ugit.bak/base.py ugit/base.py
--- ugit.bak/base.py    2022-01-31 15:59:23.434693590 +0800
+++ ugit/base.py        2022-01-31 16:08:14.484046769 +0800
@@ -131,6 +131,15 @@
     data.update_ref(f'{data.BRANCH_PREFIX}/{name}', data.RefValue(symbolic=False, value=oid))
 
 
+def get_branch_name():
+    HEAD = data.get_ref('HEAD', deref=False)
+    if not HEAD.symbolic:
+        return None
+    HEAD = HEAD.value
+    assert HEAD.startswith(data.BRANCH_PREFIX)
+    return os.path.relpath(HEAD, data.BRANCH_PREFIX)
+
+
 Commit = namedtuple('Commit', ['tree', 'parent', 'message'])
 def get_commit(oid):
     parent = None


===========================================================
diff -u ugit.bak/cli.py ugit/cli.py
--- ugit.bak/cli.py     2022-01-31 15:59:23.434693590 +0800
+++ ugit/cli.py 2022-01-31 16:01:01.586747107 +0800
@@ -64,6 +64,9 @@
     k_parser = commands.add_parser('k')
     k_parser.set_defaults(func=k)
 
+    status_parser = commands.add_parser('status')
+    status_parser.set_defaults(func=status)
+
     return parser.parse_args()
 
 
@@ -142,6 +145,16 @@
             stdin=subprocess.PIPE) as proc:
         proc.communicate(dot.encode())
 
+
+def status(args):
+    HEAD = base.get_oid('@')
+    branch = base.get_branch_name()
+    if branch:
+        print(f'On branch {branch}')
+    else:
+        print(f'HEAD detached at {HEAD[:10]}')
+
+
 if __name__ == "__main__":
     main()
```

测试：

```sh
[root@localhost repo]# ugit status
On branch master
[root@localhost repo]# ugit checkout defea7af40bbd2072b28f737cc0a91b4ca1a6f41
[root@localhost repo]# ugit status
HEAD detached at defea7af40
[root@localhost repo]# ugit checkout dev
[root@localhost repo]# ugit status
On branch dev
```



## 打印所有分支

之前，branch 命令只创建新分支。为了方便用户，我们将扩展命令，以便当用户在没有参数的情况下运行它时，列出所有现有的分支。此外，我们将在当前分支旁边打印一个星号。

为了实现这个功能，我们需要一种迭代所有分支的方法。我们已经有了 data.iter_refs，它迭代所有的 refs，而分支只是存在于 refs/heads/namespace 下的 refs。让我们给 data.iter_refs 添加一个 prefix 参数，这样调用者就可以限制它将获得哪些 refs。

然后我们将实现 base.iter_branch_names，它将遍历所有分支引用（如refs/heads/master），并只输出分支的名称（如 master）。

最后，我们将使 branch CLI 命令的参数可选，并在用户运行 ugit branch 时打印所有分支。

修改代码如下：

```diff
diff -u ugit.bak/base.py ugit/base.py
--- ugit.bak/base.py    2022-01-31 16:36:52.786102546 +0800
+++ ugit/base.py        2022-01-31 16:40:20.881170153 +0800
@@ -123,6 +123,11 @@
     data.update_ref(f'{data.TAG_PREFIX}/{name}', data.RefValue(symbolic=False, value=oid))
 
 
+def iter_branch_names():
+    for refname, _ in data.iter_refs(data.BRANCH_PREFIX):
+        yield os.path.relpath(refname, data.BRANCH_PREFIX)
+
+
 def is_branch(branch):
     return data.get_ref(f'{data.BRANCH_PREFIX}/{branch}').value is not None



===========================================================
diff -u ugit.bak/cli.py ugit/cli.py
--- ugit.bak/cli.py     2022-01-31 16:36:52.786102546 +0800
+++ ugit/cli.py 2022-01-31 16:42:32.833168313 +0800
@@ -58,7 +58,7 @@
 
     branch_parser = commands.add_parser('branch')
     branch_parser.set_defaults(func=branch)
-    branch_parser.add_argument('name')
+    branch_parser.add_argument('name', nargs='?')
     branch_parser.add_argument('start_point', default='@', type=oid, nargs='?') 
     k_parser = commands.add_parser('k')
@@ -117,8 +117,14 @@
 
 
 def branch(args):
-    base.create_branch(args.name, args.start_point)
-    print(f'Branch {args.name} created at {args.start_point[:10]}')
+    if not args.name:
+        current = base.get_branch_name()
+        for branch in base.iter_branch_names():
+            prefix = '*' if branch == current else ' '
+            print(f'{prefix} {branch}')
+    else:
+        base.create_branch(args.name, args.start_point)
+        print(f'Branch {args.name} created at {args.start_point[:10]}')
 
 
 def k(args):
 
 
===========================================================
diff -u ugit.bak/data.py ugit/data.py
--- ugit.bak/data.py    2022-01-31 16:36:52.786102546 +0800
+++ ugit/data.py        2022-01-31 16:38:21.585739151 +0800
@@ -50,13 +50,15 @@
     return ref, RefValue(symbolic=symbolic, value=value)
 
 
-def iter_refs(deref=True):
+def iter_refs(prefix='', deref=True):
     refs = ['HEAD']
     for root, _, filenames in os.walk(f'{GIT_DIR}/refs/'):
         root = os.path.relpath(root, GIT_DIR)
         refs.extend(f'{root}/{name}' for name in filenames)
 
     for refname in refs:
+        if not refname.startswith(prefix):
+            continue
         yield refname, get_ref(refname, deref)
 
 
```

测试效果：

```sh
[root@localhost repo]# ugit branch
  master
* dev
[root@localhost repo]# ugit checkout master
[root@localhost repo]# ugit branch
* master
  dev
```





---

# 显示指向每个提交的引用

这是另一个方便用户的功能。让我们修改 log 命令，在提交旁边打印引用，这样，对于打印的每个提交，我们将另外打印指向该提交的所有引用。

首先，让我们构建一个反向查找字典，从提交 OID 到引用它的 refs。

然后，对于每个提交，我们将在OID旁边打印参考文献。

修改代码如下：

```diff
diff -u ugit.bak/cli.py ugit/cli.py
--- ugit.bak/cli.py     2022-01-31 16:44:32.928098290 +0800
+++ ugit/cli.py 2022-01-31 16:52:48.774548444 +0800
@@ -98,10 +98,15 @@
 
 
 def log(args):
+    refs = {}
+    for refname, ref in data.iter_refs():
+        refs.setdefault(ref.value, []).append(refname)
+
     for oid in base.iter_commits_and_parents({args.oid}):
         commit = base.get_commit(oid)
 
-        print(f'[commit] {oid}\n')
+        refs_str = f'({", ".join(refs[oid])})' if oid in refs else ''
+        print(f'[commit] {oid} {refs_str}\n')
         print('[message]:')
         print(textwrap.indent(commit.message, '    '))
         print('=' * 58)
```

测试效果：

```sh
[root@localhost repo]# ugit log
[commit] df9f8446dbf6c4a3d52334d330817e169e2f66fa (HEAD, refs/heads/master)

[message]:
    4
==========================================================
[commit] a8b593bb5fac66e68d2f4ecf0677d1a7e56662d4

[message]:
    3
==========================================================
[commit] defea7af40bbd2072b28f737cc0a91b4ca1a6f41

[message]:
    2
==========================================================
[commit] b0822a5cad0797b08614fdf6685fdf8703f3f5a6

[message]:
    1
==========================================================
[root@localhost repo]# ugit tag 2 defea7af40bbd2072b28f737cc0a91b4ca1a6f41
[root@localhost repo]# ugit log
[commit] df9f8446dbf6c4a3d52334d330817e169e2f66fa (HEAD, refs/heads/master)

[message]:
    4
==========================================================
[commit] a8b593bb5fac66e68d2f4ecf0677d1a7e56662d4

[message]:
    3
==========================================================
[commit] defea7af40bbd2072b28f737cc0a91b4ca1a6f41 (refs/tags/2)

[message]:
    2
==========================================================
[commit] b0822a5cad0797b08614fdf6685fdf8703f3f5a6

[message]:
    1
==========================================================
```



---

# reset

ugit reset 将允许我们将 HEAD 移动到我们选择的 OID。如果我们想要撤销提交，这是很有用的。例如，我们遇到这种情况：

```text
        7fa39 ac120
            v     v
o-----o-----o-----o
                  ^
                  master <--- HEAD
```

为了清楚起见，从现在开始OID被缩写。我们可以运行这个命令来撤销提交 ac120：

```sh
$ ugit reset 7fa39
```

这将使 master 指向 7fa49，有效地擦除最新的提交 ac120，请注意，不仅 HEAD 移动了，而且 master 也移动了，因为 HEAD 是指向 master 的符号引用。

```text
        7fa39 ac120
            v     v
o-----o-----o-----o
            ^
            master <--- HEAD
```

如果我们再做一次提交，我们会得到这个并且提交 ac120 被有效地从 master 分支中移除：

```text
        7fa39 ac120
            v     v
o-----o-----o-----o
             \
              ----o
                  ^
                  master <--- HEAD
```

您可能想知道 checkout 和 reset 之间有什么区别，因为它们似乎都将 HEAD 移动到另一个提交...我们不能用 ugit checkout 7fa39 来做同样的事情吗？嗯，不行。

与 reset 不同，checkout 会移动 HEAD，而不会取消对它的引用（通过传递 deref=False）。因此，如果我们使用 checkout，我们会得到：

```text
        7fa39 ac120
            v     v
o-----o-----o-----o
            ^     ^
          HEAD    master
```

这就是为什么我们需要 reset 来移动实际的分支而不仅仅是 HEAD。

请注意，如果我们向后移动 HEAD 几个提交，我们也可以撤销多个提交，如下所示：

```text
  0dda2 7fa39 ac120
      v     v     v
o-----o-----o-----o
                  ^
                  master <--- HEAD

$ ugit reset 0dda2
```

最终结果是撤销 7fa39 和 ac120：

```text
  0dda2 7fa39 ac120
      v     v     v
o-----o-----o-----o
      ^
      master <--- HEAD
```

关于 reset 的另一点：即使我像某种撤销命令一样谈论了这个命令，重要的是要记住 reset 只是将 HEAD 移动到一个指定的状态。实际上，这可以用于撤销提交，但 reset 不是通用的撤销命令。它可以通过将 HEAD 移动到更早的提交来撤销从分支顶端开始的任意数量的提交，但是它不能撤销任意的提交。

例如，如果我们只想撤销 7fa39，我们不能通过 reset 来完成：

```text
  0dda2 7fa39 ac120
      v     v     v
o-----o-----o-----o
                  ^
                  master <--- HEAD
```

如果我们将 master 移动到 0dda2，我们将撤销 ac120 和 7fa39，这与仅撤销 7fa39 不同。因此，reset 不能用于从历史记录中间删除提交。

代码修改如下：

```diff
diff -u ugit.bak/base.py ugit/base.py
--- ugit.bak/base.py    2022-01-31 18:21:26.253136709 +0800
+++ ugit/base.py        2022-01-31 19:14:33.215481460 +0800
@@ -119,6 +119,10 @@
     data.update_ref('HEAD', HEAD, deref=False)
 
 
+def reset(oid):
+    data.update_ref('HEAD', data.RefValue(symbolic=False, value=oid))
+
+
 def create_tag(name, oid):
     data.update_ref(f'{data.TAG_PREFIX}/{name}', data.RefValue(symbolic=False, value=oid))


===========================================================
diff -u ugit.bak/cli.py ugit/cli.py
--- ugit.bak/cli.py     2022-01-31 19:16:47.671990148 +0800
+++ ugit/cli.py 2022-01-31 19:13:51.524582123 +0800
@@ -67,6 +67,10 @@
     status_parser = commands.add_parser('status')
     status_parser.set_defaults(func=status)
 
+    reset_parser = commands.add_parser('reset')
+    reset_parser.set_defaults(func=reset)
+    reset_parser.add_argument('commit', type=oid)
+
     return parser.parse_args()
 
 
@@ -166,6 +170,10 @@
         print(f'HEAD detached at {HEAD[:10]}')
 
 
+def reset(args):
+    base.reset(args.commit)
+
+
 if __name__ == "__main__":
     main()
 
```

测试：

```sh
[root@localhost repo]# ugit log
[commit] df9f8446dbf6c4a3d52334d330817e169e2f66fa (HEAD, refs/heads/master)

[message]:
    4
==========================================================
[commit] a8b593bb5fac66e68d2f4ecf0677d1a7e56662d4 

[message]:
    3
==========================================================
[commit] defea7af40bbd2072b28f737cc0a91b4ca1a6f41 (refs/tags/2)

[message]:
    2
==========================================================
[commit] b0822a5cad0797b08614fdf6685fdf8703f3f5a6 

[message]:
    1
==========================================================
[root@localhost repo]# ugit reset 2
[root@localhost repo]# ugit log
[commit] defea7af40bbd2072b28f737cc0a91b4ca1a6f41 (HEAD, refs/heads/master, refs/tags/2)

[message]:
    2
==========================================================
[commit] b0822a5cad0797b08614fdf6685fdf8703f3f5a6 

[message]:
    1
==========================================================
```

注意：通常默认情况下，运行 git reset 时，真正的 Git 不会重置工作目录，所以这里也这样做了。Git 会在传递 --hard 标志时重置工作目录。

