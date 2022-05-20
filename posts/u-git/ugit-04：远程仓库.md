# 允许仓库变更

我们将很快实现存储库同步（FETCH，PUSH）。为此，我们需要允许 ugit 更改当前的 GIT_DIR，以便在同步的同时临时查看不同的存储库。

让我们使用 contextManager 来允许在 with 语句中执行更改，这样目录更改就可以很容易地堆叠起来，并且是可逆性的。

如果您不熟悉 ConextManager 装饰器，[see this tutorial](https://book.pythontips.com/en/latest/context_managers.html)

代码修改如下：

```diff
diff -u ugit.bak/cli.py ugit/cli.py
--- ugit.bak/cli.py     2022-02-09 09:08:20.165000000 +0800
+++ ugit/cli.py 2022-02-09 09:09:32.528000000 +0800
@@ -10,8 +10,9 @@
 
 
 def main():
-    args = parse_args()
-    args.func(args)
+    with data.change_git_dir('.'):
+        args = parse_args()
+        args.func(args)
 
 
 def parse_args():
 
 
===========================================================
diff -u ugit.bak/data.py ugit/data.py
--- ugit.bak/data.py    2022-02-09 09:08:20.165000000 +0800
+++ ugit/data.py        2022-02-09 09:24:34.085000000 +0800
@@ -2,16 +2,28 @@
 import hashlib
 
 from collections import namedtuple
+from contextlib import contextmanager
+
+# Will be initialized in cli.main()
+GIT_DIR = None
+
+
+@contextmanager
+def change_git_dir(new_dir):
+    global GIT_DIR
+    old_dir = GIT_DIR
+    GIT_DIR = f'{new_dir}/.ugit'
+    yield
+    GIT_DIR = old_dir
+
 
-GIT_DIR = ".ugit"
-OBJECTS_DIR = GIT_DIR + "/objects"
 TAG_PREFIX = "refs/tags"
 BRANCH_PREFIX = "refs/heads"
 
 
 def init():
     os.makedirs(GIT_DIR)
-    os.makedirs(OBJECTS_DIR)
+    os.makedirs(f'{GIT_DIR}/objects')
 
 
 RefValue = namedtuple('RefValue', ['symbolic', 'value'])
@@ -72,13 +84,13 @@
 def hash_object(data, type_="blob"):
     obj = type_.encode() + b"\x00" + data
     oid = hashlib.sha1(obj).hexdigest()
-    with open("{}/{}".format(OBJECTS_DIR, oid), "wb") as f:
+    with open(f'{GIT_DIR}/objects/{oid}', 'wb') as f:
         f.write(obj)
     return oid
 
 
 def get_object(oid, expected="blob"):
-    with open("{}/{}".format(OBJECTS_DIR, oid), 'rb') as f:
+    with open(f'{GIT_DIR}/objects/{oid}', 'rb') as f:
         obj = f.read()
 
     type_, _, content = obj.partition(b'\x00')
```





---

# fetch

## 打印远端refs

在下面的更改中，我们将处理 FETCH 命令。此命令的目的是从远程存储库下载 refs 和相关对象。

从更大的角度来看，它将允许我们同步不同的存储库。例如，让我们假设两个人（我和同事）在同一个存储库上协作，每个人都有一个存储库的副本。最初，我们有相同的提交，但随后该同事在 master 上添加了另一个提交（标记为 * ）。

-   我的仓库

```text
                master
                v
o---o---o---o---o
```

-   同事的仓库

```text
                    master
                    v
o---o---o---o---o---*
```

我希望将我的（本地）存储库与其（远程）存储库同步。因此，我将运行 ugit fetch /path/to/remote/repo，这将带来远程的 refs 和提交：

```text
                master
                v
o---o---o---o---o---*
                    ^
                    remote/master
```

请注意，远程 master 在 remote/ 命名空间下以区别于我的 master。接下来，如果我想同步我的 master，我可以做 ugit merge remote/master（假设 HEAD 指向 master）：

```text
                    master
                    v
o---o---o---o---o---*
                    ^
                    remote/master
```

实现整个 FETCH 命令将需要一些时间，但我仍然希望给出最终目标。

现在我们将从第一部分开始工作：

让我们首先列出远程存储库上的所有 refs。FETCH 将接受远程存储库作为参数，将 GIT_DIR 更改为指向远程存储库，并使用 iter_refs 函数列出所有 refs。因为我只想打印分支而不是 HEAD 之类的，所以我们得到以 refs/head 开头的 refs。

我创建了一个新的 remote.py 模块，它将包含我们所有的远程同步代码。

还要注意，我们只支持位于同一个文件系统上的远程存储库，这意味着它们并不像位于另一台计算机上的那样远程。在我们的例子中，remote 意味着一个外部存储库，即使它位于同一个文件系统上。Git 也支持相同文件系统上的远程处理，但也支持其他远程类型，如 SSH 或 HTTP。我不想通过这些网络协议实现远程支持，因为它们要复杂得多，基于文件系统也能说明重要的概念。

代码修改如下：

```diff
diff -u ugit.bak/cli.py ugit/cli.py
--- ugit.bak/cli.py     2022-02-09 09:31:57.779000000 +0800
+++ ugit/cli.py 2022-02-09 09:45:33.015000000 +0800
@@ -7,6 +7,7 @@
 from . import base
 from . import data
 from . import diff
+from . import remote
 
 
 def main():
@@ -90,6 +91,10 @@
     merge_base_parser.add_argument('commit1', type=oid)
     merge_base_parser.add_argument('commit2', type=oid)
 
+    fetch_parser = commands.add_parser('fetch')
+    fetch_parser.set_defaults(func=fetch)
+    fetch_parser.add_argument('remote')
+
     return parser.parse_args()
 
 
@@ -235,6 +240,11 @@
     print(base.get_merge_base(args.commit1, args.commit2))
 
 
+
+def fetch(args):
+    remote.fetch(args.remote)
+
+
 if __name__ == "__main__":
     main()


============================================================
Only in ugit: remote.py
+from . import data
+
+
+def fetch(remote_path):
+    print('Will fetch the following refs:')
+    with data.change_git_dir(remote_path):
+        for refname, _ in data.iter_refs(data.BRANCH_PREFIX):
+            print(f'- {refname}')
+
```

我们用 fetch 打印本地分支试试：

```sh
[root@centos repo]# ugit fetch .
Will fetch the following refs:
- refs/heads/master
- refs/heads/dev
```

下面一个小的重构更改，对以后的工作很有用。我创建了一个单独的函数，从远程存储库获取所有的 ref 名称和值，然后从 FETCH 调用它。

代码修改如下：

```diff
diff -u ugit.bak/remote.py ugit/remote.py
--- ugit.bak/remote.py  2022-02-09 09:48:58.690000000 +0800
+++ ugit/remote.py      2022-02-09 09:51:04.399000000 +0800
@@ -3,7 +3,11 @@
 
 def fetch(remote_path):
     print('Will fetch the following refs:')
+    for refname, _ in _get_remote_refs(remote_path, data.BRANCH_PREFIX).items():
+        print(f'- {refname}')
+
+
+def _get_remote_refs(remote_path, prefix=''):
     with data.change_git_dir(remote_path):
-        for refname, _ in data.iter_refs(data.BRANCH_PREFIX):
-            print(f'- {refname}')
+        return {refname: ref.value for refname, ref in data.iter_refs(prefix)}
 
```



## 下载远端refs

与其仅仅打印远程 refs，不如将它们保存在本地。我们将为远程 refs 使用一个单独的 ref 命名空间，因此，如果远程存储库有 refs/head/master，我们将在本地保存它作为 refs/emote/master，以区分远程 master 和本地 master。

-   我的仓库

```text
                refs/heads/master
                v
o---o---o---o---o
```

-   远端仓库

```text
                    refs/heads/master
                    v
o---o---o---o---o---*
```

运行 fetch 之后：

```text
                refs/heads/master
                v
o---o---o---o---o   *
                    ^
                    refs/remote/master
```

请注意，我的对象存储库中仍然缺少远程提交，因此 refs/emote/master 指向一个不存在的对象，任何使用它的尝试都会导致错误。我们将在下一次更改中修复错误。

请注意，真正的 Git 支持多个远端，并且每个远端都有一个名称。例如，如果有一个名为 origin 的远程分支，那么它的分支将位于 refs/remote/origin/ 下面。但是在 ugit 中，为了简单起见，我们假设只有一个远端，并且只将其分支置于 refs/remote/ 下面。

代码修改如下：

```diff
diff -u ugit.bak/remote.py ugit/remote.py
--- ugit.bak/remote.py  2022-02-09 09:53:56.045000000 +0800
+++ ugit/remote.py      2022-02-09 10:56:26.472000000 +0800
@@ -1,10 +1,19 @@
+import os
 from . import data
 
 
+REMOTE_REFS_BASE = 'refs/heads/'
+LOCAL_REFS_BASE = 'refs/remote/'
+
+
 def fetch(remote_path):
-    print('Will fetch the following refs:')
-    for refname, _ in _get_remote_refs(remote_path, data.BRANCH_PREFIX).items():
-        print(f'- {refname}')
+    # Get refs from server
+    refs = _get_remote_refs(remote_path, REMOTE_REFS_BASE)
+
+    # Update local refs to match server
+    for remote_name, value in refs.items():
+        refname = os.path.relpath(remote_name, REMOTE_REFS_BASE)
+        data.update_ref(f'{LOCAL_REFS_BASE}/{refname}', data.RefValue(symbolic=False, value=value))
 
 
 def _get_remote_refs(remote_path, prefix=''):
```



## 下载远端对象

除了获取 refs 之外，我们还将获取 refs 所指向的对象。为此，我们需要一些额外的功能：

-   In data.py

添加 fetch_object_if_missing 函数根据 oid 来拷贝远端 blob

-   In base.py

添加 iter_objects_in_commits 函数，它将获取提交列表，并返回所有可从这些提交中访问的对象。

它主要依赖于 iter_commits_and_parents 来获得所有提交的列表，然后递归地迭代每个提交中的树。

-   In remote.py

我们已经在 data.py 和 base.py 中完成了繁重的工作，因此在 fetch 过程中，我们只需要迭代所有对象并获取缺少的对象。

就这样！我们结束了 fetch 工作！它不是最有效的实现，但功能非常强大。

现在，您可以使用 FETCH 和 merge 来将当前工作分支和远程存储库同步。

代码修改如下：

```diff
diff -u ugit.bak/base.py ugit/base.py
--- ugit.bak/base.py    2022-02-09 10:58:50.875000000 +0800
+++ ugit/base.py        2022-02-09 13:34:14.506000000 +0800
@@ -242,6 +242,28 @@
         oids.extend(commit.parents[1:])
 
 
+def iter_objects_in_commits(oids):
+    # N.B. Must yield the oid before acccessing it (to allow caller to fetch it if needed)
+
+    visited = set()
+    def iter_objects_in_tree(oid):
+        visited.add(oid)
+        yield oid
+        for type_, oid, _ in _iter_tree_entries(oid):
+            if oid not in visited:
+                if type_ == 'tree':
+                    yield from iter_objects_in_tree(oid)
+                else:
+                    visited.add(oid)
+                    yield oid
+
+    for oid in iter_commits_and_parents(oids):
+        yield oid
+        commit = get_commit(oid)
+        if commit.tree not in visited:
+            yield from iter_objects_in_tree(commit.tree)
+
+
 def get_oid(name):
     if name == '@':
         name = 'HEAD'


===========================================================
diff -u ugit.bak/data.py ugit/data.py
--- ugit.bak/data.py    2022-02-09 10:58:50.875000000 +0800
+++ ugit/data.py        2022-02-09 11:16:53.127000000 +0800
@@ -1,5 +1,6 @@
 import os
 import hashlib
+import shutil
 
 from collections import namedtuple
 from contextlib import contextmanager
@@ -101,3 +102,14 @@
         
     return content
 
+
+def object_exists(oid):
+    return os.path.isfile(f'{GIT_DIR}/objects/{oid}')
+
+
+def fetch_object_if_missing(oid, remote_git_dir):
+    if object_exists(oid):
+        return
+    remote_git_dir += '/.ugit'
+    shutil.copy(f'{remote_git_dir}/objects/{oid}', f'{GIT_DIR}/objects/{oid}')
+


===========================================================
diff -u ugit.bak/remote.py ugit/remote.py
--- ugit.bak/remote.py  2022-02-09 10:58:50.875000000 +0800
+++ ugit/remote.py      2022-02-09 11:21:05.625000000 +0800
@@ -1,5 +1,7 @@
 import os
+
 from . import data
+from . import base
 
 
 REMOTE_REFS_BASE = 'refs/heads/'
@@ -10,6 +12,10 @@
     # Get refs from server
     refs = _get_remote_refs(remote_path, REMOTE_REFS_BASE)
 
+    # Fetch missing objects by iterating and fetching on demand
+    for oid in base.iter_objects_in_commits(refs.values()):
+        data.fetch_object_if_missing(oid, remote_path)
+
     # Update local refs to match server
     for remote_name, value in refs.items():
         refname = os.path.relpath(remote_name, REMOTE_REFS_BASE)
```

测试，我们首先模拟一个远程仓库，在上面做点提交：

```sh
[root@centos diyGit]# cp repo remote -rf
[root@centos diyGit]# cd remote/
[root@centos remote]# ugit status
On branch master

Changes to be committed:

[root@centos remote]# ls
1.txt  2.txt  3.txt
[root@centos remote]# echo 4 > 4.txt
[root@centos remote]# ugit commit -m "4"
blob c928f7711483160a5245c9da863f775563cb3584 ./1.txt
blob 4bc9dea33de5dcb1b375f7e90cc3ded4228af369 ./2.txt
blob 02e4777f2f664a8cbf9319fad4cef9bce967b77a ./3.txt
blob 9b76b5b7e33b52e1c7c3ee63ca39cb415990c556 ./4.txt
tree 7483b2bb9a99373d03a8d24a2ebfab5498c36199 .
3df6e23aaf5ad3da7ab391a654b49546f0084215
```

然后进行 fetch 操作将远端同步过来，之后执行 merge 合并分支，我们可以看到远端更新的 4.txt 现在已经同步到本地仓库了：

```sh
[root@centos remote]# cd ../repo
[root@centos repo]# ls .ugit/refs/
heads  tags
[root@centos repo]# ugit fetch ../remote/
[root@centos repo]# ls .ugit/refs/
heads  remote  tags
[root@centos repo]# ugit merge refs/remote/master
Fast-forward merge, no need to commit
[root@centos repo]# ls
1.txt  2.txt  3.txt  4.txt
```





---

# push

## 同步全部对象

与 fetch 相反的是 push。它不是下载远程 refs 和对象，而是上传对象并将本地 refs 同步到远程。当您添加了一些提交时，您希望更新一个远程存储库，以便它与您的本地版本同步。

-   我的仓库

```text
                refs/heads/master
                v
o---o---o---o---o
```

-   远程仓库

```text
        refs/heads/master
        v
o---o---o
```

如果我们运行 ugit push /path/to/remote master，结果会是：

-   我的仓库

```text
                refs/heads/master
                v
o---o---o---o---o
```

-   远程仓库

```text
                refs/heads/master
                v
o---o---o---o---o
```

我们将创建一个新的 push 命令，我们在 data.py 添加一个 push_object 函数，它将一个本地对象复制到一个远程存储库中，最后我们将它绑定到远程存储库中：将所有可从分支到达的对象推到远程（用我们喜爱的 iter_objects_in_commits 找到它们），并将远程引用和本地的 ref 值同步。

这种实现效率低下，因为每次它都将所有对象推送到远程存储库，而不管那里存在哪些对象。我们将在后面修改中改进它。

代码修改如下：

```diff
diff -u ugit.bak/cli.py ugit/cli.py
--- ugit.bak/cli.py     2022-02-09 13:39:20.961000000 +0800
+++ ugit/cli.py 2022-02-09 13:44:11.783000000 +0800
@@ -95,6 +95,11 @@
     fetch_parser.set_defaults(func=fetch)
     fetch_parser.add_argument('remote')
 
+    push_parser = commands.add_parser('push')
+    push_parser.set_defaults(func=push)
+    push_parser.add_argument('remote')
+    push_parser.add_argument('branch')
+
     return parser.parse_args()
 
 
@@ -245,6 +250,10 @@
     remote.fetch(args.remote)
 
 
+def push(args):
+    remote.push(args.remote, f'refs/heads/{args.branch}')
+
+
 if __name__ == "__main__":
     main()


===========================================================
diff -u ugit.bak/data.py ugit/data.py
--- ugit.bak/data.py    2022-02-09 13:39:20.960000000 +0800
+++ ugit/data.py        2022-02-09 13:44:53.560000000 +0800
@@ -113,3 +113,8 @@
     remote_git_dir += '/.ugit'
     shutil.copy(f'{remote_git_dir}/objects/{oid}', f'{GIT_DIR}/objects/{oid}')
 
+
+def push_object(oid, remote_git_dir):
+    remote_git_dir += '/.ugit'
+    shutil.copy(f'{GIT_DIR}/objects/{oid}', f'{remote_git_dir}/objects/{oid}')
+


===========================================================
diff -u ugit.bak/remote.py ugit/remote.py
--- ugit.bak/remote.py  2022-02-09 13:39:20.960000000 +0800
+++ ugit/remote.py      2022-02-09 13:46:20.857000000 +0800
@@ -22,6 +22,22 @@
         data.update_ref(f'{LOCAL_REFS_BASE}/{refname}', data.RefValue(symbolic=False, value=value))
 
 
+def push(remote_path, refname):
+    # Get refs data
+    local_ref = data.get_ref(refname).value
+    assert local_ref
+
+    objects_to_push = base.iter_objects_in_commits({local_ref})
+
+    # Push all objects
+    for oid in objects_to_push:
+        data.push_object(oid, remote_path)
+
+    # Update server ref to our value
+    with data.change_git_dir(remote_path):
+        data.update_ref(refname, data.RefValue(symbolic=False, value=local_ref))
+
+
 def _get_remote_refs(remote_path, prefix=''):
     with data.change_git_dir(remote_path):
         return {refname: ref.value for refname, ref in data.iter_refs(prefix)}
```

测试，可以看到远端仓库已经有了我们最新提交的 tree 以及 master 更新为最新的提交记录：

```sh
[root@centos repo]# echo 5 > 5.txt
[root@centos repo]# ugit commit -m "5"
blob c928f7711483160a5245c9da863f775563cb3584 ./1.txt
blob 4bc9dea33de5dcb1b375f7e90cc3ded4228af369 ./2.txt
blob 02e4777f2f664a8cbf9319fad4cef9bce967b77a ./3.txt
blob 9b76b5b7e33b52e1c7c3ee63ca39cb415990c556 ./4.txt
blob 9e3ebc18c9035a6bea6e8ad4785a286e21f7f7dc ./5.txt
tree b45596006fa5d0751fc1ffe6e3bc7a9a284a495c .
5dccef7d1e7e525387b13ad5f1b31a06e98416e1
[root@centos repo]# ugit push ../remote/ master
[root@centos repo]# cd ../remote
[root@centos remote]# cat .ugit/refs/heads/master 
5dccef7d1e7e525387b13ad5f1b31a06e98416e1
[root@centos remote]# ll .ugit/objects/b45596006fa5d0751fc1ffe6e3bc7a9a284a495c 
-rw-r--r--. 1 root root 265 Feb  9 13:47 .ugit/objects/b45596006fa5d0751fc1ffe6e3bc7a9a284a495c
```



## 只发送缺失的对象

如前所述，每次推送复制所有对象都是效率低下的，因为远端仓库可能已经拥有其中的一些对象。让我们添加一个简单的检查，以确定远程有哪些对象，而不是推送所有对象：

-   获取所有远端分支，并过滤掉本地不存在的分支，得到 known_remote_refs
-   获取 known_remote_refs 所有对象
-   获取本地所有对象并和上面的结果求差集

GIT 有更高级的启发式方法来确定应该推送哪些对象，但我们将就此打住。

代码修改如下：

```diff
diff -u ugit.bak/remote.py ugit/remote.py
--- ugit.bak/remote.py  2022-02-09 13:54:41.638000000 +0800
+++ ugit/remote.py      2022-02-09 14:06:47.512000000 +0800
@@ -24,12 +24,17 @@
 
 def push(remote_path, refname):
     # Get refs data
+    remote_refs = _get_remote_refs(remote_path)
     local_ref = data.get_ref(refname).value
     assert local_ref
 
-    objects_to_push = base.iter_objects_in_commits({local_ref})
+    # Compute which objects the server doesn't have
+    known_remote_refs = filter(data.object_exists, remote_refs.values())
+    remote_objects = set(base.iter_objects_in_commits(known_remote_refs))
+    local_objects = set(base.iter_objects_in_commits({local_ref}))
+    objects_to_push = local_objects - remote_objects
 
-    # Push all objects
+    # Push missing objects
     for oid in objects_to_push:
         data.push_object(oid, remote_path)
 
```



## 不允许强制推送

我们目前的推送实现是不安全的。如果两个人在同一个分支上工作，而这个分支被推送到一个公共的远程存储库中，那么他们可能会覆盖对方的工作。考虑这一情况：

-   我的仓库

```text
                refs/heads/master
                v
o---o---o---o---@
```

-   我同事的仓库

```text
                refs/heads/master
                v
o---o---o---o---$
```

-   远端仓库

```text
            refs/heads/master
            v
o---o---o---o
```

请注意，我和我的同事都在使用同一个远程分支，我们每个人都在远程存储库的同一个提交基础上进行了不同的提交。现在，如果我的同事 push，这个时候看起来就像：

-   我的仓库

```text
                refs/heads/master
                v
o---o---o---o---@
```

-   我同事的仓库

```text
                refs/heads/master
                v
o---o---o---o---$
```

-   远端仓库

```text
                refs/heads/master
                v
o---o---o---o---$
```

如果我不注意并运行 push，ugit 将很高兴地覆盖我同事的最新提交，如下所示：

-   我的仓库

```text
                refs/heads/master
                v
o---o---o---o---@
```

-   我同事的仓库

```text
                refs/heads/master
                v
o---o---o---o---$
```

-   远端仓库

```text
                refs/heads/master
                v
o---o---o---o---@
```

为了防止这种情况发生，我们只允许在两种情况下推送：

-   我们推送的分支在远端还不存在。这意味着它是一个新分支，不存在覆盖他人工作的风险。
-   如果远程引用确实存在，则必须指向推送引用的某个祖先提交。这意味着本地提交基于远程提交，也意味着远程提交不会被覆盖，因为它是新推送提交历史的一部分。

让我们返回前面的示例。假设情况如下：

-   我的仓库

```text
                refs/heads/master
                v
o---o---o---o---@
```

-   我同事的仓库

```text
                refs/heads/master
                v
o---o---o---o---$
```

-   远端仓库

```text
                refs/heads/master
                v
o---o---o---o---$
```

如果我试图运行 ugit push master，那么断言将触发，因为 $ 提交不是 @ 提交的祖先，要进行协调，我需要运行 ugit fetch，它将检索最新的主服务器：

-   我的仓库

```text
                refs/heads/master
                v
o---o---o---o---@
             \
              --$
                ^
                refs/remote/master
```

-   远端仓库

```text
                refs/heads/master
                v
o---o---o---o---$
```

然后我需要运行 ugit merge remote/master，这将使分支合并：

-   我的仓库

```text
                    refs/heads/master
                    v
o---o---o---o---@---o
             \     /
              --$--
                ^
                refs/remote/master
```

-   远端仓库

```text
                refs/heads/master
                v
o---o---o---o---$
```

只有现在，我才能成功地运行 push，因为远程指向的 $ 提交是 master 的一个祖先。推送后：

-   我的仓库

```text
                    refs/heads/master
                    v
o---o---o---o---@---o < refs/remote/master
             \     /
              --$--
```

-   远端仓库

```text
                    refs/heads/master
                    v
o---o---o---o---@---o
             \     /
              --$--
```

BTW，真正的 Git 有一个选项 --force 忽略这个安全检查，但我们这里不予实现。

代码修改如下：

```diff
diff -u ugit.bak/base.py ugit/base.py
--- ugit.bak/base.py    2022-02-09 14:29:26.156000000 +0800
+++ ugit/base.py        2022-02-09 14:32:16.193000000 +0800
@@ -178,6 +178,10 @@
             return oid
 
 
+def is_ancestor_of(commit, maybe_ancestor):
+    return maybe_ancestor in iter_commits_and_parents({commit})
+
+
 def create_tag(name, oid):
     data.update_ref(f'{data.TAG_PREFIX}/{name}', data.RefValue(symbolic=False, value=oid))


===========================================================
diff -u ugit.bak/remote.py ugit/remote.py
--- ugit.bak/remote.py  2022-02-09 14:29:26.157000000 +0800
+++ ugit/remote.py      2022-02-09 14:40:08.197000000 +0800
@@ -25,9 +25,13 @@
 def push(remote_path, refname):
     # Get refs data
     remote_refs = _get_remote_refs(remote_path)
+    remote_ref = remote_refs.get(refname)
     local_ref = data.get_ref(refname).value
     assert local_ref
 
+    # Don't allow force push
+    assert not remote_ref or base.is_ancestor_of(local_ref, remote_ref)
+
     # Compute which objects the server doesn't have
     known_remote_refs = filter(data.object_exists, remote_refs.values())
     remote_objects = set(base.iter_objects_in_commits(known_remote_refs))
```



---

# add

## 在索引区中记录添加的文件

现在，当我们运行 ugit commit 时，工作目录中的所有更改都会写入下一个提交。在某些情况下，这样做不太方便。例如，我们可以在工作过程中进行多个无关的更改，我们不一定希望将它们分组在同一个提交中。

我们现在将实现一个名为 index 的特性，它将允许对已提交的文件进行更细粒度的控制。它将允许我们指定工作目录中哪些文件应该是提交的一部分，哪些不应该。

流程如下所示：

-   用户修改工作目录中的一些文件。
-   对于用户希望提交的每个文件，他将运行 ugit add path/to/file。ugit 将把文件放入对象数据库，并在索引中记住它的 OID。索引是一个字典，它将文件名映射到他们最后记住的 OID。索引将保存为 .ugit 目录中的 JSON 文件，以便我们可以在调用 ugit 持久化它。
-   用户运行 ugit commit，此时 write-tree 将获取索引的内容并将其写入树对象。因为索引将文件名映射到 OID，就像树对象一样，所以转换很容易。

以前，write-tree 会从工作目录中获取数据。现在它将从索引中获取数据，并将有一个单独的 add 命令将数据放入索引中。这实际上允许用户通过在索引中添加相关文件（Git 中的暂存区）来控制下一次提交的内容。

在这一变化中，我们将为 index 奠定基础。我们将添加一个全新的 add 命令，然后添加一个名为 get_index 的函数，该函数可以以 JSON 格式读写索引，然后一旦对文件调用 add，它将在索引中记录文件的当前状态。

请注意，这次更改在任何方面都不会影响 ugit，因为 write-tree 仍然从工作目录接收文件。我们将在下面的更改中将 write-tree 连接到索引。现在，您可以在一堆文件上运行 ugit add，并确保它们确实出现在 `.ugit/index` 中。

BTW，对于索引来说，真正的 Git 有一个更复杂和优化的数据结构，但是 JSON 将足以让我们实现这个想法。

代码修改如下：

```diff
diff -u ugit.bak/base.py ugit/base.py
--- ugit.bak/base.py    2022-02-09 15:05:56.322000000 +0800
+++ ugit/base.py        2022-02-09 15:18:14.908000000 +0800
@@ -291,6 +291,16 @@
     assert False, f'Unknown name {name}'
 
 
+def add(filenames):
+    with data.get_index() as index:
+        for filename in filenames:
+            # Normalize path
+            filename = os.path.relpath(filename)
+            with open(filename, 'rb') as f:
+                oid = data.hash_object(f.read())
+            index[filename] = oid
+
+
 def is_ignored(path):
     return '.ugit' in path.split("/")



===========================================================
diff -u ugit.bak/cli.py ugit/cli.py
--- ugit.bak/cli.py     2022-02-09 15:05:56.322000000 +0800
+++ ugit/cli.py 2022-02-09 15:11:17.682000000 +0800
@@ -100,6 +100,10 @@
     push_parser.add_argument('remote')
     push_parser.add_argument('branch')
 
+    add_parser = commands.add_parser('add')
+    add_parser.set_defaults(func=add)
+    add_parser.add_argument('files', nargs='+')
+
     return parser.parse_args()
 
 
@@ -254,6 +258,10 @@
     remote.push(args.remote, f'refs/heads/{args.branch}')
 
 
+def add(args):
+    base.add(args.files)
+
+
 if __name__ == "__main__":
     main()
 
 
===========================================================
diff -u ugit.bak/data.py ugit/data.py
--- ugit.bak/data.py    2022-02-09 15:05:56.322000000 +0800
+++ ugit/data.py        2022-02-09 15:19:09.996000000 +0800
@@ -1,6 +1,7 @@
 import os
 import hashlib
 import shutil
+import json
 
 from collections import namedtuple
 from contextlib import contextmanager
@@ -82,6 +83,19 @@
             yield refname, ref
 
 
+@contextmanager
+def get_index():
+    index = {}
+    if os.path.isfile(f'{GIT_DIR}/index'):
+        with open(f'{GIT_DIR}/index') as f:
+            index = json.load(f)
+
+    yield index
+
+    with open(f'{GIT_DIR}/index', 'w') as f:
+        json.dump(index, f)
+
+
 def hash_object(data, type_="blob"):
     obj = type_.encode() + b"\x00" + data
     oid = hashlib.sha1(obj).hexdigest()
```

测试：

```sh
[root@centos repo]# echo 6 > 6.txt
[root@centos repo]# ugit add 6.txt 
[root@centos repo]# ll .ugit/
HEAD     index    objects/ refs/    
[root@centos repo]# cat .ugit/index 
{"6.txt": "c692c2974b398a30957fc14365165152e623e928"}
```



## 添加目录

为了更方便地使用索引，我们还将接受目录作为 ugit add 的参数。这在概念上并没有改变任何东西，但是会更方便。特别是，我们甚至可以做 `ugit add .` 添加工作目录中的所有内容。

代码修改如下：

```diff
diff -u ugit.bak/base.py ugit/base.py
--- ugit.bak/base.py    2022-02-09 15:29:30.348000000 +0800
+++ ugit/base.py        2022-02-09 15:31:25.609000000 +0800
@@ -292,13 +292,29 @@
 
 
 def add(filenames):
+
+    def add_file(filename):
+        # Normalize path
+        filename = os.path.relpath(filename)
+        with open(filename, 'rb') as f:
+            oid = data.hash_object(f.read())
+        index[filename] = oid
+
+    def add_directory(dirname):
+        for root, _, filenames in os.walk(dirname):
+            for filename in filenames:
+                # Normalize path
+                path = os.path.relpath(f'{root}/{filename}')
+                if is_ignored(path) or not os.path.isfile(path):
+                    continue
+                add_file(path)
+
     with data.get_index() as index:
-        for filename in filenames:
-            # Normalize path
-            filename = os.path.relpath(filename)
-            with open(filename, 'rb') as f:
-                oid = data.hash_object(f.read())
-            index[filename] = oid
+        for name in filenames:
+            if os.path.isfile(name):
+                add_file(name)
+            elif os.path.isdir(name):
+                add_directory(name)
 
 
 def is_ignored(path):
```

测试：

```sh
[root@centos repo]# ugit add .
[root@centos repo]# cat .ugit/index 
{"6.txt": "c692c2974b398a30957fc14365165152e623e928", "1.txt": "c928f7711483160a5245c9da863f775563cb3584", "2.txt": "4bc9dea33de5dcb1b375f7e90cc3ded4228af369", "3.txt": "02e4777f2f664a8cbf9319fad4cef9bce967b77a", "4.txt": "9b76b5b7e33b52e1c7c3ee63ca39cb415990c556", "5.txt": "9e3ebc18c9035a6bea6e8ad4785a286e21f7f7dc"}
```



## 修改write-tree和read-tree

以前，write_tree() 会递归地扫描目录，为它遇到的每个目录编写树对象。现在，它将从 index 中获取数据。为此，我们需要完全重写 write_tree()。

首先，我们需要将索引（这是一个扁平的文件列表）转换为字典树。索引存储为一个平面列表，以便更容易地添加和删除项，并使其与 diff.py 中的 diff 逻辑兼容（稍后我们将看到这一点）。但是 ugit 将完整的树存储在树对象的层次结构中，所以我们需要将数据转换成字典树。

然后，我们将调用 write_tree_recursive()，它基本上与旧的 write_tree() 相同，但是它不会在文件系统上操作，而是使用字典树并将它们写到对象存储中。

cool！现在索引起作用了，从现在开始，您必须记住在提交文件之前添加它们，否则它们将不会是提交的一部分。

代码修改如下：

```diff
diff -u ugit.bak/base.py ugit/base.py
--- ugit.bak/base.py    2022-02-09 15:45:09.513000000 +0800
+++ ugit/base.py        2022-02-09 15:49:29.505000000 +0800
@@ -14,29 +14,38 @@
     data.update_ref('HEAD', data.RefValue(symbolic=True, value=f'{data.BRANCH_PREFIX}/master'))
 
 
-def write_tree(directory="."):
-    entries = []
-    with os.scandir(directory) as it:
-        for entry in it:
-            full = "{}/{}".format(directory, entry.name)
-            if is_ignored(full):
-                continue
-
-            if entry.is_file(follow_symlinks=False):
-                type_ = "blob"
-                with open(full, 'rb') as f:
-                    oid = data.hash_object(f.read())
-                    print(type_, oid, full)
-            elif entry.is_dir(follow_symlinks=False):
-                type_ = "tree"
-                oid = write_tree(full)
-            entries.append((entry.name, oid, type_))
-
-    tree = ''.join(f'{type_} {oid} {name}\n'
-                   for name, oid, type_
-                   in sorted(entries))
-    tree_oid = data.hash_object(tree.encode(), 'tree')
-    print('tree', tree_oid, directory)
+def write_tree():
+    # Index is flat, we need it as a tree of dicts
+    index_as_tree = {}
+    with data.get_index() as index:
+        for path, oid in index.items():
+            path = path.split('/')
+            dirpath, filename = path[:-1], path[-1]
+
+            current = index_as_tree
+            # Find the dict for the directory of this file
+            for dirname in dirpath:
+                current = current.setdefault(dirname, {})
+            current[filename] = oid
+
+    def write_tree_recursive(tree_dict):
+        entries = []
+        for name, value in tree_dict.items():
+            if type(value) is dict:
+                type_ = 'tree'
+                oid = write_tree_recursive(value)
+            else:
+                type_ = 'blob'
+                oid = value
+            entries.append((name, oid, type_))
+
+        tree = ''.join(f'{type_} {oid} {name}\n'
+                        for name, oid, type_
+                        in sorted(entries))
+        return data.hash_object(tree.encode(), 'tree')
+
+    tree_oid = write_tree_recursive(index_as_tree)
+    print('tree', tree_oid)
     return tree_oid
```

为了对称起见，我们也需要在 read-tree 时填充索引。

在提交期间，数据从工作目录到索引到提交，因此当读取树（在签出或合并期间）时，数据将从树到索引到工作目录。

如果不进行此更改，则当用户 checkout 不同的提交时，索引的状态仍将反映旧提交期间的文件。但现在它将得到正确的更新。

修改代码如下：

```diff
diff -u ugit.bak/base.py ugit/base.py
--- ugit.bak/base.py    2022-02-09 16:02:58.089000000 +0800
+++ ugit/base.py        2022-02-09 16:10:01.761000000 +0800
@@ -104,20 +104,34 @@
                 pass
 
 
-def read_tree(tree_oid):
-    _empty_current_directory()
-    for path, oid in get_tree(tree_oid, base_path='./').items():
-        os.makedirs(os.path.dirname(path), exist_ok=True)
-        with open(path, 'wb') as f:
-            f.write(data.get_object(oid))
+def read_tree(tree_oid, update_working=False):
+    with data.get_index() as index:
+        index.clear()
+        index.update(get_tree(tree_oid))
+
+        if update_working:
+            _checkout_index(index)
+
+
+def read_tree_merged(t_base, t_HEAD, t_other, update_working=False):
+    with data.get_index() as index:
+        index.clear()
+        index.update(diff.merge_trees(
+            get_tree(t_base),
+            get_tree(t_HEAD),
+            get_tree(t_other)
+        ))
+
+        if update_working:
+            _checkout_index(index)
 
 
-def read_tree_merged(t_base, t_HEAD, t_other):
+def _checkout_index(index):
     _empty_current_directory()
-    for path, blob in diff.merge_trees(get_tree(t_base), get_tree(t_HEAD), get_tree(t_other)).items():
-        os.makedirs(f'./{os.path.dirname(path)}', exist_ok=True)
+    for path, oid in index.items():
+        os.makedirs(os.path.dirname(f'./{path}'), exist_ok=True)
         with open(path, 'wb') as f:
-            f.write(blob)
+            f.write(data.get_object(oid, 'blob'))
 
 
 def commit(message):
@@ -144,7 +158,7 @@
 def checkout(name):
     oid = get_oid(name)
     commit = get_commit(oid)
-    read_tree(commit.tree)
+    read_tree(commit.tree, update_working=True)
 
     if is_branch(name):
         HEAD = data.RefValue(symbolic=True, value=f'refs/heads/{name}')
@@ -166,7 +180,7 @@
 
     # Handle fast-forward merge
     if merge_base == HEAD:
-        read_tree(c_other.tree)
+        read_tree(c_other.tree, update_working=True)
         data.update_ref('HEAD', data.RefValue(symbolic=False, value=other))
         print('Fast-forward merge, no need to commit')
         return
@@ -175,7 +189,7 @@
 
     c_base = get_commit(merge_base)
     c_HEAD = get_commit(HEAD)
-    read_tree_merged(c_base.tree, c_HEAD.tree, c_other.tree)
+    read_tree_merged(c_base.tree, c_HEAD.tree, c_other.tree, update_working=True)
     print('Merged in working tree\nPlease commit')
 

===========================================================
diff -u ugit.bak/diff.py ugit/diff.py
--- ugit.bak/diff.py    2022-02-09 16:02:58.088000000 +0800
+++ ugit/diff.py        2022-02-09 16:11:07.033000000 +0800
@@ -53,7 +53,7 @@
 def merge_trees(t_base, t_HEAD, t_other):
     tree = {}
     for path, o_base, o_HEAD, o_other in compare_trees(t_base, t_HEAD, t_other):
-        tree[path] = merge_blobs(o_base, o_HEAD, o_other)
+        tree[path] = data.hash_object(merge_blobs(o_base, o_HEAD, o_other))
     return tree
 
 
```



## 展示暂存区

为了使 ugit 更方便，让我们扩展 status 命令，以显示哪些已更改的文件将被提交，哪些文件被更改，但不会提交。

更改非常简单，因为它依赖于现有的基础设施。我们将比较索引和 HEAD 来显示将要提交的更改文件。将索引与工作树进行比较来显示未提交的更改文件。

请注意，两个组可以存在同一个文件，比如也许我们更改了它，将它添加到索引中，对它做了更多的更改，但没有将最新的更改添加到索引中。

修改代码如下：

```diff
diff -u ugit.bak/base.py ugit/base.py
--- ugit.bak/base.py    2022-02-09 16:14:13.989000000 +0800
+++ ugit/base.py        2022-02-09 16:17:05.835000000 +0800
@@ -85,6 +85,11 @@
     return result
 
 
+def get_index_tree():
+    with data.get_index() as index:
+        return index
+
+
 def _empty_current_directory():
     for root, dirnames, filenames in os.walk('.', topdown=False):
         for filename in filenames:


===========================================================
diff -u ugit.bak/cli.py ugit/cli.py
--- ugit.bak/cli.py     2022-02-09 16:14:13.989000000 +0800
+++ ugit/cli.py 2022-02-09 16:16:32.427000000 +0800
@@ -233,7 +233,11 @@
 
     print('\nChanges to be committed:\n')
     HEAD_tree = HEAD and base.get_commit(HEAD).tree
-    for path, action in diff.iter_changed_files(base.get_tree(HEAD_tree), base.get_working_tree()):
+    for path, action in diff.iter_changed_files(base.get_tree(HEAD_tree), base.get_index_tree()):
+        print(f'{action:>12}: {path}')
+
+    print('\nChanges not staged for commit:\n')
+    for path, action in diff.iter_changed_files(base.get_index_tree(), base.get_working_tree()):
         print(f'{action:>12}: {path}')
 
```



首先我们测试下暂存区状态显示：

```sh
[root@centos repo]# ugit init
Initialized empty ugit repository in /diyGit/repo/./.ugit
[root@centos repo]# echo 1 > 1.txt
[root@centos repo]# ugit add .
[root@centos repo]# ugit status
On branch master

Changes to be committed:

    new file: 1.txt

Changes not staged for commit:

[root@centos repo]# echo 2 > 2.txt
[root@centos repo]# ugit status
On branch master

Changes to be committed:

    new file: 1.txt

Changes not staged for commit:

    new file: 2.txt
[root@centos repo]# cat .ugit/index 
{"1.txt": "c928f7711483160a5245c9da863f775563cb3584"}
```

测试 checkout 时暂存区变化：

```sh
[root@centos repo]# ugit add 2.txt 
[root@centos repo]# cat .ugit/index 
{"1.txt": "c928f7711483160a5245c9da863f775563cb3584", "2.txt": "4bc9dea33de5dcb1b375f7e90cc3ded4228af369"}[root@centos repo]# ugit commit -m "commit 2"
tree c0b6dd6b6a33eeb6c24be9718023c5195683f866
68f6f9b75117f5ddff33ada6d6584643a05b24e4
[root@centos repo]# ugit log
[commit] 68f6f9b75117f5ddff33ada6d6584643a05b24e4 (HEAD, refs/heads/master)

[message]:
    commit 2
==========================================================
[commit] e0b804153663baa1ab5c280b5485ea34c68d4710 

[message]:
    commit 1
==========================================================
[root@centos repo]# echo 3 > 3.txt
[root@centos repo]# ugit add .
[root@centos repo]# ugit tag tag1 e0b804153663baa1ab5c280b5485ea34c68d4710
[root@centos repo]# ugit checkout tag1
[root@centos repo]# ls
1.txt
[root@centos repo]# cat .ugit/index 
{"1.txt": "c928f7711483160a5245c9da863f775563cb3584"}
```





## 比较差异

让我们添加另一个模式到 ugit diff。在此之前，该命令将区分工作树和另一个提交。现在我们已经有了索引，让我们介绍一些更有用的模式来区分：

-   如果没有提供参数，则从索引到工作目录的差异。这样，您就可以快速地看到暂存区的更改。
-   如果提供 `--cached`，则从头到索引的差异。这样你就可以快速地看到哪些变化将被执行。
-   如果提供了特定的提交，则与提交到索引或工作目录的差异（取决于是否提供了--cached）。

代码修改如下：

```diff
diff -u ugit.bak/cli.py ugit/cli.py
--- ugit.bak/cli.py     2022-02-09 16:22:24.944000000 +0800
+++ ugit/cli.py 2022-02-09 16:24:17.790000000 +0800
@@ -56,7 +56,8 @@
 
     diff_parser = commands.add_parser('diff')
     diff_parser.set_defaults(func=_diff)
-    diff_parser.add_argument('commit', default='@', type=oid, nargs='?')
+    diff_parser.add_argument('--cached', action='store_true')
+    diff_parser.add_argument('commit', nargs='?')
 
     checkout_parser = commands.add_parser('checkout')
     checkout_parser.set_defaults(func=checkout)
@@ -167,9 +168,25 @@
 
 
 def _diff(args):
-    tree = args.commit and base.get_commit(args.commit).tree
+    oid = args.commit and base.get_oid(args.commit)
 
-    result = diff.diff_trees(base.get_tree(tree), base.get_working_tree())
+    if args.commit:
+        # If a commit was provided explicitly, diff from it
+        tree_from = base.get_tree(oid and base.get_commit(oid).tree)
+
+    if args.cached:
+        tree_to = base.get_index_tree()
+        if not args.commit:
+            # If no commit was provided, diff from HEAD
+            oid = base.get_oid('@')
+            tree_from = base.get_tree(oid and base.get_commit(oid).tree)
+    else:
+        tree_to = base.get_working_tree()
+        if not args.commit:
+            # If no commit was provided, diff from index
+            tree_from = base.get_index_tree()
+
+    result = diff.diff_trees(tree_from, tree_to)
     sys.stdout.flush()
     sys.stdout.buffer.write(result)
 
```





---

# 完整代码

-   setup.py

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



-   ugit/cli.py

```python
import argparse
import os
import sys
import textwrap
import subprocess

from . import base
from . import data
from . import diff
from . import remote


def main():
    with data.change_git_dir('.'):
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

    show_parser = commands.add_parser('show')
    show_parser.set_defaults(func=show)
    show_parser.add_argument('oid', default='@', type=oid, nargs='?')

    diff_parser = commands.add_parser('diff')
    diff_parser.set_defaults(func=_diff)
    diff_parser.add_argument('--cached', action='store_true')
    diff_parser.add_argument('commit', nargs='?')

    checkout_parser = commands.add_parser('checkout')
    checkout_parser.set_defaults(func=checkout)
    checkout_parser.add_argument('commit')

    tag_parser = commands.add_parser('tag')
    tag_parser.set_defaults(func=tag)
    tag_parser.add_argument('name')
    tag_parser.add_argument('oid', nargs='?', type=oid, default='@')

    branch_parser = commands.add_parser('branch')
    branch_parser.set_defaults(func=branch)
    branch_parser.add_argument('name', nargs='?')
    branch_parser.add_argument('start_point', default='@', type=oid, nargs='?')

    k_parser = commands.add_parser('k')
    k_parser.set_defaults(func=k)

    status_parser = commands.add_parser('status')
    status_parser.set_defaults(func=status)

    reset_parser = commands.add_parser('reset')
    reset_parser.set_defaults(func=reset)
    reset_parser.add_argument('commit', type=oid)

    merge_parser = commands.add_parser('merge')
    merge_parser.set_defaults(func=merge)
    merge_parser.add_argument('commit', type=oid)

    merge_base_parser = commands.add_parser('merge-base')
    merge_base_parser.set_defaults(func=merge_base)
    merge_base_parser.add_argument('commit1', type=oid)
    merge_base_parser.add_argument('commit2', type=oid)

    fetch_parser = commands.add_parser('fetch')
    fetch_parser.set_defaults(func=fetch)
    fetch_parser.add_argument('remote')

    push_parser = commands.add_parser('push')
    push_parser.set_defaults(func=push)
    push_parser.add_argument('remote')
    push_parser.add_argument('branch')

    add_parser = commands.add_parser('add')
    add_parser.set_defaults(func=add)
    add_parser.add_argument('files', nargs='+')

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


def _print_commit(oid, commit, refs=None):
    refs_str = f'({", ".join(refs)})' if refs else ''
    print(f'[commit] {oid} {refs_str}\n')
    print('[message]:')
    print(textwrap.indent(commit.message, '    '))
    print('=' * 58)


def log(args):
    refs = {}
    for refname, ref in data.iter_refs():
        refs.setdefault(ref.value, []).append(refname)

    for oid in base.iter_commits_and_parents({args.oid}):
        commit = base.get_commit(oid)
        _print_commit(oid, commit, refs.get(oid))


def show(args):
    if not args.oid:
        return
    commit = base.get_commit(args.oid)
    parent_tree = None
    if commit.parents:
        parent_tree = base.get_commit(commit.parents[0]).tree

    _print_commit(args.oid, commit)
    result = diff.diff_trees(base.get_tree(parent_tree), base.get_tree(commit.tree))
    sys.stdout.flush()
    sys.stdout.buffer.write(result)


def _diff(args):
    oid = args.commit and base.get_oid(args.commit)

    if args.commit:
        # If a commit was provided explicitly, diff from it
        tree_from = base.get_tree(oid and base.get_commit(oid).tree)

    if args.cached:
        tree_to = base.get_index_tree()
        if not args.commit:
            # If no commit was provided, diff from HEAD
            oid = base.get_oid('@')
            tree_from = base.get_tree(oid and base.get_commit(oid).tree)
    else:
        tree_to = base.get_working_tree()
        if not args.commit:
            # If no commit was provided, diff from index
            tree_from = base.get_index_tree()

    result = diff.diff_trees(tree_from, tree_to)
    sys.stdout.flush()
    sys.stdout.buffer.write(result)


def checkout(args):
    base.checkout(args.commit)


def tag(args):
    oid = args.oid
    base.create_tag(args.name, oid)


def branch(args):
    if not args.name:
        current = base.get_branch_name()
        for branch in base.iter_branch_names():
            prefix = '*' if branch == current else ' '
            print(f'{prefix} {branch}')
    else:
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
        for parent in commit.parents:
            dot += f'"{oid}" -> "{parent}"\n'

    dot += '}'
    print(dot)

    with subprocess.Popen(
            ['dot', '-Tgtk', '/dev/stdin'],
            stdin=subprocess.PIPE) as proc:
        proc.communicate(dot.encode())


def status(args):
    HEAD = base.get_oid('@')
    branch = base.get_branch_name()
    if branch:
        print(f'On branch {branch}')
    else:
        print(f'HEAD detached at {HEAD[:10]}')

    MERGE_HEAD = data.get_ref('MERGE_HEAD').value
    if MERGE_HEAD:
        print(f'Merging with {MERGE_HEAD[:10]}')

    print('\nChanges to be committed:\n')
    HEAD_tree = HEAD and base.get_commit(HEAD).tree
    for path, action in diff.iter_changed_files(base.get_tree(HEAD_tree), base.get_index_tree()):
        print(f'{action:>12}: {path}')

    print('\nChanges not staged for commit:\n')
    for path, action in diff.iter_changed_files(base.get_index_tree(), base.get_working_tree()):
        print(f'{action:>12}: {path}')


def reset(args):
    base.reset(args.commit)


def merge(args):
    base.merge(args.commit)


def merge_base(args):
    print(base.get_merge_base(args.commit1, args.commit2))



def fetch(args):
    remote.fetch(args.remote)


def push(args):
    remote.push(args.remote, f'refs/heads/{args.branch}')


def add(args):
    base.add(args.files)


if __name__ == "__main__":
    main()

```



-   ugit/base.py

```python
import itertools
import operator
import os
import string

from collections import namedtuple, deque

from . import data
from . import diff


def init():
    data.init()
    data.update_ref('HEAD', data.RefValue(symbolic=True, value=f'{data.BRANCH_PREFIX}/master'))


def write_tree():
    # Index is flat, we need it as a tree of dicts
    index_as_tree = {}
    with data.get_index() as index:
        for path, oid in index.items():
            path = path.split('/')
            dirpath, filename = path[:-1], path[-1]

            current = index_as_tree
            # Find the dict for the directory of this file
            for dirname in dirpath:
                current = current.setdefault(dirname, {})
            current[filename] = oid

    def write_tree_recursive(tree_dict):
        entries = []
        for name, value in tree_dict.items():
            if type(value) is dict:
                type_ = 'tree'
                oid = write_tree_recursive(value)
            else:
                type_ = 'blob'
                oid = value
            entries.append((name, oid, type_))

        tree = ''.join(f'{type_} {oid} {name}\n'
                        for name, oid, type_
                        in sorted(entries))
        return data.hash_object(tree.encode(), 'tree')

    tree_oid = write_tree_recursive(index_as_tree)
    print('tree', tree_oid)
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


def get_working_tree():
    result = {}
    for root, _, filenames in os.walk('.'):
        for filename in filenames:
            path = os.path.relpath(f'{root}/{filename}')
            if is_ignored(path) or not os.path.isfile(path):
                continue
            with open(path, 'rb') as f:
                result[path] = data.hash_object(f.read())
    return result


def get_index_tree():
    with data.get_index() as index:
        return index


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


def read_tree(tree_oid, update_working=False):
    with data.get_index() as index:
        index.clear()
        index.update(get_tree(tree_oid))

        if update_working:
            _checkout_index(index)


def read_tree_merged(t_base, t_HEAD, t_other, update_working=False):
    with data.get_index() as index:
        index.clear()
        index.update(diff.merge_trees(
            get_tree(t_base),
            get_tree(t_HEAD),
            get_tree(t_other)
        ))

        if update_working:
            _checkout_index(index)


def _checkout_index(index):
    _empty_current_directory()
    for path, oid in index.items():
        os.makedirs(os.path.dirname(f'./{path}'), exist_ok=True)
        with open(path, 'wb') as f:
            f.write(data.get_object(oid, 'blob'))


def commit(message):
    commit = f'tree {write_tree()}\n'

    HEAD = data.get_ref('HEAD').value
    if HEAD:
        commit += f'parent {HEAD}\n'

    MERGE_HEAD = data.get_ref('MERGE_HEAD').value
    if MERGE_HEAD:
        commit += f'parent {MERGE_HEAD}\n'
        data.delete_ref('MERGE_HEAD', deref=False)

    commit += '\n'
    commit += f'{message}\n'

    oid = data.hash_object(commit.encode(), 'commit')
    data.update_ref('HEAD', data.RefValue(symbolic=False, value=oid))

    return oid


def checkout(name):
    oid = get_oid(name)
    commit = get_commit(oid)
    read_tree(commit.tree, update_working=True)

    if is_branch(name):
        HEAD = data.RefValue(symbolic=True, value=f'refs/heads/{name}')
    else:
        HEAD = data.RefValue(symbolic=False, value=oid)

    data.update_ref('HEAD', HEAD, deref=False)


def reset(oid):
    data.update_ref('HEAD', data.RefValue(symbolic=False, value=oid))


def merge(other):
    HEAD = data.get_ref('HEAD').value
    assert HEAD
    merge_base = get_merge_base(other, HEAD)
    c_other = get_commit(other)

    # Handle fast-forward merge
    if merge_base == HEAD:
        read_tree(c_other.tree, update_working=True)
        data.update_ref('HEAD', data.RefValue(symbolic=False, value=other))
        print('Fast-forward merge, no need to commit')
        return

    data.update_ref('MERGE_HEAD', data.RefValue(symbolic=False, value=other))

    c_base = get_commit(merge_base)
    c_HEAD = get_commit(HEAD)
    read_tree_merged(c_base.tree, c_HEAD.tree, c_other.tree, update_working=True)
    print('Merged in working tree\nPlease commit')


def get_merge_base(oid1, oid2):
    parents1 = set(iter_commits_and_parents({oid1}))

    for oid in iter_commits_and_parents({oid2}):
        if oid in parents1:
            return oid


def is_ancestor_of(commit, maybe_ancestor):
    return maybe_ancestor in iter_commits_and_parents({commit})


def create_tag(name, oid):
    data.update_ref(f'{data.TAG_PREFIX}/{name}', data.RefValue(symbolic=False, value=oid))


def iter_branch_names():
    for refname, _ in data.iter_refs(data.BRANCH_PREFIX):
        yield os.path.relpath(refname, data.BRANCH_PREFIX)


def is_branch(branch):
    return data.get_ref(f'{data.BRANCH_PREFIX}/{branch}').value is not None


def create_branch(name, oid):
    data.update_ref(f'{data.BRANCH_PREFIX}/{name}', data.RefValue(symbolic=False, value=oid))


def get_branch_name():
    HEAD = data.get_ref('HEAD', deref=False)
    if not HEAD.symbolic:
        return None
    HEAD = HEAD.value
    assert HEAD.startswith(data.BRANCH_PREFIX)
    return os.path.relpath(HEAD, data.BRANCH_PREFIX)


Commit = namedtuple('Commit', ['tree', 'parents', 'message'])
def get_commit(oid):
    parents = []

    commit = data.get_object(oid, 'commit').decode()
    lines = iter(commit.splitlines())
    for line in itertools.takewhile(operator.truth, lines):
        key, value = line.split(' ', 1)
        if key == 'tree':
            tree = value
        elif key == 'parent':
            parents.append(value)
        else:
            assert False, f'Unknown field {key}'

    message = '\n'.join(lines)
    return Commit(tree=tree, parents=parents, message=message)



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
        # Return first parent next
        oids.extendleft(commit.parents[:1])
        # Return other parents later
        oids.extend(commit.parents[1:])


def iter_objects_in_commits(oids):
    # N.B. Must yield the oid before acccessing it (to allow caller to fetch it if needed)

    visited = set()
    def iter_objects_in_tree(oid):
        visited.add(oid)
        yield oid
        for type_, oid, _ in _iter_tree_entries(oid):
            if oid not in visited:
                if type_ == 'tree':
                    yield from iter_objects_in_tree(oid)
                else:
                    visited.add(oid)
                    yield oid

    for oid in iter_commits_and_parents(oids):
        yield oid
        commit = get_commit(oid)
        if commit.tree not in visited:
            yield from iter_objects_in_tree(commit.tree)


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


def add(filenames):

    def add_file(filename):
        # Normalize path
        filename = os.path.relpath(filename)
        with open(filename, 'rb') as f:
            oid = data.hash_object(f.read())
        index[filename] = oid

    def add_directory(dirname):
        for root, _, filenames in os.walk(dirname):
            for filename in filenames:
                # Normalize path
                path = os.path.relpath(f'{root}/{filename}')
                if is_ignored(path) or not os.path.isfile(path):
                    continue
                add_file(path)

    with data.get_index() as index:
        for name in filenames:
            if os.path.isfile(name):
                add_file(name)
            elif os.path.isdir(name):
                add_directory(name)


def is_ignored(path):
    return '.ugit' in path.split("/")

```



-   ugit/data.py

```python
import os
import hashlib
import shutil
import json

from collections import namedtuple
from contextlib import contextmanager

# Will be initialized in cli.main()
GIT_DIR = None


@contextmanager
def change_git_dir(new_dir):
    global GIT_DIR
    old_dir = GIT_DIR
    GIT_DIR = f'{new_dir}/.ugit'
    yield
    GIT_DIR = old_dir


TAG_PREFIX = "refs/tags"
BRANCH_PREFIX = "refs/heads"


def init():
    os.makedirs(GIT_DIR)
    os.makedirs(f'{GIT_DIR}/objects')


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


def delete_ref(ref, deref=True):
    ref = _get_ref_internal(ref, deref)[0]
    os.remove(f'{GIT_DIR}/{ref}')


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


def iter_refs(prefix='', deref=True):
    refs = ['HEAD', 'MERGE_HEAD']
    for root, _, filenames in os.walk(f'{GIT_DIR}/refs/'):
        root = os.path.relpath(root, GIT_DIR)
        refs.extend(f'{root}/{name}' for name in filenames)

    for refname in refs:
        if not refname.startswith(prefix):
            continue
        ref = get_ref(refname, deref)
        if ref.value:
            yield refname, ref


@contextmanager
def get_index():
    index = {}
    if os.path.isfile(f'{GIT_DIR}/index'):
        with open(f'{GIT_DIR}/index') as f:
            index = json.load(f)

    yield index

    with open(f'{GIT_DIR}/index', 'w') as f:
        json.dump(index, f)


def hash_object(data, type_="blob"):
    obj = type_.encode() + b"\x00" + data
    oid = hashlib.sha1(obj).hexdigest()
    with open(f'{GIT_DIR}/objects/{oid}', 'wb') as f:
        f.write(obj)
    return oid


def get_object(oid, expected="blob"):
    with open(f'{GIT_DIR}/objects/{oid}', 'rb') as f:
        obj = f.read()

    type_, _, content = obj.partition(b'\x00')
    type_ = type_.decode()

    if expected is not None:
        assert type_ == expected, "Expected {}, got {}".format(expected, type_)
        
    return content


def object_exists(oid):
    return os.path.isfile(f'{GIT_DIR}/objects/{oid}')


def fetch_object_if_missing(oid, remote_git_dir):
    if object_exists(oid):
        return
    remote_git_dir += '/.ugit'
    shutil.copy(f'{remote_git_dir}/objects/{oid}', f'{GIT_DIR}/objects/{oid}')


def push_object(oid, remote_git_dir):
    remote_git_dir += '/.ugit'
    shutil.copy(f'{GIT_DIR}/objects/{oid}', f'{remote_git_dir}/objects/{oid}')

```



-   ugit/diff.py

```python
import subprocess

from collections import defaultdict
from tempfile import NamedTemporaryFile as Temp

from . import data


def compare_trees(*trees):
    entries = defaultdict(lambda: [None] * len(trees))
    for i, tree in enumerate(trees):
        for path, oid in tree.items():
            entries[path][i] = oid

    for path, oids in entries.items():
        yield(path, *oids)


def iter_changed_files(t_from, t_to):
    for path, o_from, o_to in compare_trees(t_from, t_to):
        if o_from != o_to:
            action = ('new file' if not o_from else
                      'deleted' if not o_to else
                      'modified')
            yield path, action


def diff_trees(t_from, t_to):
    output = b''
    for path, o_from, o_to in compare_trees(t_from, t_to):
        if o_from != o_to:
            output += diff_blobs(o_from, o_to, path)
    return output


def diff_blobs(o_from, o_to, path='blob'):
    with Temp() as f_from, Temp() as f_to:
        for oid, f in((o_from, f_from), (o_to, f_to)):
            if oid:
                f.write(data.get_object(oid))
                f.flush()

        with subprocess.Popen(
            ['diff', '--unified', '--show-c-function',
             '--label', f'old/{path}', f_from.name,
             '--label', f'new/{path}', f_to.name],
            stdout=subprocess.PIPE) as proc:
            output, _ = proc.communicate()

        return output


def merge_trees(t_base, t_HEAD, t_other):
    tree = {}
    for path, o_base, o_HEAD, o_other in compare_trees(t_base, t_HEAD, t_other):
        tree[path] = data.hash_object(merge_blobs(o_base, o_HEAD, o_other))
    return tree


def merge_blobs(o_base, o_HEAD, o_other):
    with Temp() as f_base, Temp() as f_HEAD, Temp() as f_other:

        # Write blobs to files
        for oid, f in ((o_base, f_base), (o_HEAD, f_HEAD), (o_other, f_other)):
            if oid:
                f.write(data.get_object(oid))
                f.flush()

        with subprocess.Popen(
            ['diff3', '-m',
             '-L', 'HEAD', f_HEAD.name,
             '-L', 'BASE', f_base.name,
             '-L', 'MERGE_HEAD', f_other.name,
            ], stdout=subprocess.PIPE) as proc:
            output, _ = proc.communicate()
            assert proc.returncode in (0, 1)

        return output

```



-   ugit/remote.py

```python
import os

from . import data
from . import base


REMOTE_REFS_BASE = 'refs/heads/'
LOCAL_REFS_BASE = 'refs/remote/'


def fetch(remote_path):
    # Get refs from server
    refs = _get_remote_refs(remote_path, REMOTE_REFS_BASE)

    # Fetch missing objects by iterating and fetching on demand
    for oid in base.iter_objects_in_commits(refs.values()):
        data.fetch_object_if_missing(oid, remote_path)

    # Update local refs to match server
    for remote_name, value in refs.items():
        refname = os.path.relpath(remote_name, REMOTE_REFS_BASE)
        data.update_ref(f'{LOCAL_REFS_BASE}/{refname}', data.RefValue(symbolic=False, value=value))


def push(remote_path, refname):
    # Get refs data
    remote_refs = _get_remote_refs(remote_path)
    remote_ref = remote_refs.get(refname)
    local_ref = data.get_ref(refname).value
    assert local_ref

    # Don't allow force push
    assert not remote_ref or base.is_ancestor_of(local_ref, remote_ref)

    # Compute which objects the server doesn't have
    known_remote_refs = filter(data.object_exists, remote_refs.values())
    remote_objects = set(base.iter_objects_in_commits(known_remote_refs))
    local_objects = set(base.iter_objects_in_commits({local_ref}))
    objects_to_push = local_objects - remote_objects

    # Push missing objects
    for oid in objects_to_push:
        data.push_object(oid, remote_path)

    # Update server ref to our value
    with data.change_git_dir(remote_path):
        data.update_ref(refname, data.RefValue(symbolic=False, value=local_ref))


def _get_remote_refs(remote_path, prefix=''):
    with data.change_git_dir(remote_path):
        return {refname: ref.value for refname, ref in data.iter_refs(prefix)}

```

