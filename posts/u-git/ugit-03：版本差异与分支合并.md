# show

到目前为止，从用户的角度来看，提交对象是不透明的。用户可以使用 log 或 cat-file 看到提交消息，也许用户可以读取与提交相关联的 tree 对象，但基本也就能做到这样。

最重要的是，用户看不到提交之间的变化，也称为差异。如您所见，Git 为每次提交存储整个目录的快照，而不是差异。

在版本控制系统中，查看差异是一个非常重要的特性，因为通常在审查提交时，差异是您用来判断提交内容的部分。

我们将添加一个 ugit show CLI 命令，它将以一种有用的方式为我们打印一个提交对象——提交消息和与上次提交不同的文本。

在这次更改中，让我们为 show 创建一个新的 CLI 命令，并从打印提交消息开始。我们将以与日志相同的格式打印它，所以让我们将其重构为 _print_commit。

在 Git 中，git show 可以显示各种类型的对象。在 ugit 中，ugit show 只适用于提交对象，以简化代码。

## 添加命令

修改代码如下：

```diff
diff -u ugit.bak/cli.py ugit/cli.py
--- ugit.bak/cli.py     2022-01-31 19:32:27.958690393 +0800
+++ ugit/cli.py 2022-01-31 19:37:54.819093385 +0800
@@ -47,6 +47,10 @@
     log_parser.set_defaults(func=log)
     log_parser.add_argument('oid', nargs='?', type=oid, default='@')
 
+    show_parser = commands.add_parser('show')
+    show_parser.set_defaults(func=show)
+    show_parser.add_argument('oid', default='@', type=oid, nargs='?')
+
     checkout_parser = commands.add_parser('checkout')
     checkout_parser.set_defaults(func=checkout)
     checkout_parser.add_argument('commit')
@@ -101,6 +105,14 @@
     print(base.commit(args.message))
 
 
+def _print_commit(oid, commit, refs=None):
+    refs_str = f'({", ".join(refs)})' if refs else ''
+    print(f'[commit] {oid} {refs_str}\n')
+    print('[message]:')
+    print(textwrap.indent(commit.message, '    '))
+    print('=' * 58)
+
+
 def log(args):
     refs = {}
     for refname, ref in data.iter_refs():
@@ -108,12 +120,14 @@
 
     for oid in base.iter_commits_and_parents({args.oid}):
         commit = base.get_commit(oid)
+        _print_commit(oid, commit, refs.get(oid))
+
 
-        refs_str = f'({", ".join(refs[oid])})' if oid in refs else ''
-        print(f'[commit] {oid} {refs_str}\n')
-        print('[message]:')
-        print(textwrap.indent(commit.message, '    '))
-        print('=' * 58)
+def show(args):
+    if not args.oid:
+        return
+    commit = base.get_commit(args.oid)
+    _print_commit(args.oid, commit)
 
 
 def checkout(args):
```

测试：

```sh
[root@localhost repo]# ugit show
[commit] defea7af40bbd2072b28f737cc0a91b4ca1a6f41 

[message]:
    2
==========================================================
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



## 列出提交时更改的文件

让我们首先找出哪些文件被提交更改了。我们将把提交树与其父提交树进行比较。如果一个文件在前一个树中有一个 OID，在后一个树中有一个不同的 OID，这意味着该文件已被更改。

让我们创建一个新的模块 diff.py。这个模块将包含处理计算对象之间差异的代码。我们将实现一个名为 compare_trees() 的重要函数，该函数将获取一个树列表，并返回按文件名分组的树。这样，对于每个文件，我们可以在不同的树中获得它的所有 oid。

使用 diff.compare_trees()，显示所有已更改文件的任务非常简单。我们将创建 diff.diff_trees()，它获取两个树，对它们进行比较，并返回具有不同 oid 的所有条目。然后剩下的就是用正确的树（我们想要显示的提交及其父树）从 show 调用diff_trees()。

代码修改如下：

```diff
diff -u ugit.bak/cli.py ugit/cli.py
--- ugit.bak/cli.py     2022-01-31 19:54:29.239945262 +0800
+++ ugit/cli.py 2022-01-31 19:56:58.797026312 +0800
@@ -6,6 +6,7 @@
 
 from . import base
 from . import data
+from . import diff
 
 
 def main():
@@ -127,7 +128,13 @@
     if not args.oid:
         return
     commit = base.get_commit(args.oid)
+    parent_tree = None
+    if commit.parent:
+        parent_tree = base.get_commit(commit.parent).tree
+
     _print_commit(args.oid, commit)
+    result = diff.diff_trees(base.get_tree(parent_tree), base.get_tree(commit.tree))
+    print(result)
 
 
 def checkout(args):
 
 
只在 ugit 存在：diff.py
```

ugit/diff.py

```python
from collections import defaultdict


def compare_trees(*trees):
    entries = defaultdict(lambda: [None] * len(trees))
    for i, tree in enumerate(trees):
        for path, oid in tree.items():
            entries[path][i] = oid

    for path, oids in entries.items():
        yield(path, *oids)


def diff_trees(t_from, t_to):
    output = ''
    for path, o_from, o_to in compare_trees(t_from, t_to):
        if o_from != o_to:
            output += f'changed: {path}\n'
    return output
```



## 打印提交的差异

最后，让我们打印实际的差异（行首带有加号和减号的格式），而不是只打印提交中更改的文件。

我们将创建一个函数 diff_blobs()，它将获取两个 blob OIDs 并返回它们之间的差异。对于每个更改的文件，我们将调用 diff_blobs()。

对于 diff_blobs()，我们将使用一个名为 diff 的外部 Unix 实用程序。它接收两个文件作为参数，并以 diff 格式打印它们之间的差异，就像我们需要的那样。我们需要从对象数据库中读取 blobs，并将它们写入一个临时文件，该文件将提供给 diff 程序。

由于 diff 的输出是一个字节字符串，我们将使用 sys.stdout.buffer.write() 将它输出到 stdout。

在调用 diff 实用程序时，我传递了一些额外的选项（- unified，- show-c-function，- label）来使 diff 看起来更漂亮。您可以在 diff 的手册页中了解这些选项。

就这样！我们实际上可以在任何提交上运行 ugit show，并查看提交的全部内容。试试看！

修改代码如下：

```diff
diff -u ugit.bak/cli.py ugit/cli.py
--- ugit.bak/cli.py     2022-01-31 22:12:13.238660229 +0800
+++ ugit/cli.py 2022-01-31 22:24:46.955494425 +0800
@@ -134,7 +134,8 @@
 
     _print_commit(args.oid, commit)
     result = diff.diff_trees(base.get_tree(parent_tree), base.get_tree(commit.tree))
-    print(result)
+    sys.stdout.flush()
+    sys.stdout.buffer.write(result)
 
 
 def checkout(args):
 
 
===========================================================
diff -u ugit.bak/diff.py ugit/diff.py
--- ugit.bak/diff.py    2022-01-31 22:12:13.239743560 +0800
+++ ugit/diff.py        2022-01-31 22:34:34.037578723 +0800
@@ -1,4 +1,9 @@
+import subprocess
+
 from collections import defaultdict
+from tempfile import NamedTemporaryFile as Temp
+
+from . import data
 
 
 def compare_trees(*trees):
@@ -12,8 +17,26 @@
 
 
 def diff_trees(t_from, t_to):
-    output = ''
+    output = b''
     for path, o_from, o_to in compare_trees(t_from, t_to):
         if o_from != o_to:
-            output += f'changed: {path}\n'
+            output += diff_blobs(o_from, o_to, path)
     return output
+
+
+def diff_blobs(o_from, o_to, path='blob'):
+    with Temp() as f_from, Temp() as f_to:
+        for oid, f in((o_from, f_from), (o_to, f_to)):
+            if oid:
+                f.write(data.get_object(oid))
+                f.flush()
+
+        with subprocess.Popen(
+            ['diff', '--unified', '--show-c-function',
+             '--label', f'old/{path}', f_from.name,
+             '--label', f'new/{path}', f_to.name],
+            stdout=subprocess.PIPE) as proc:
+            output, _ = proc.communicate()
+
+        return output
+
```

我们随便修改个文件进行测试：

```sh
[root@localhost repo]# vim 2.txt 
[root@localhost repo]# ugit commit -m "test show"
blob c928f7711483160a5245c9da863f775563cb3584 ./1
blob 6abd9b71d3527d0c35e73115ba5640aa1c9a34aa ./2.txt
tree 27a5cfac78977c535d7780c72c11b670ca688a23 .
51d05279508e71dba02eab1dfb3cbb42bf167e97
[root@localhost repo]# ugit show
[commit] 51d05279508e71dba02eab1dfb3cbb42bf167e97 

[message]:
    test show
==========================================================
--- old/2.txt
+++ new/2.txt
@@ -1 +1,2 @@
-666
+777
+888
```



## 将工作树与提交进行比较

接下来让我们实现另一个有用的命令 diff，一个显示自上次提交以来工作目录中发生了什么变化的命令。如果你不记得你到底改变了什么，或者如果你想确保你没有错误地引入任何意想不到的改变，这是很有用的。该命令将被称为 diff（镜像 Git 的名称）。

我们的 diff 实现将使用我们之前为了展示而创建的 diff.diff_trees()。这一次，我们将要求 diff_trees() 将 working tree 与某个提交进行比较。working tree 是描述工作目录中文件的字典。

我们将实现 get_working_tree()，它将遍历工作目录中的所有文件，将它们放在对象数据库中，并创建一个包含所有 oid 的字典。这本字典将代表一个 tree，但实际上没有写 tree 对象。

每次将所有文件放在对象数据库中只是为了计算一个 diff，这可能看起来很奇怪，但是无论如何，当我们实现一个叫做 index 的东西时，它将是以后需要的。

然后，让我们创建一个 CLI 命令 diff，它将只比较当前的工作树和提交树（默认为 HEAD，但它可以是我们想要的任何提交）

代码修改如下：

```diff
diff -u ugit.bak/base.py ugit/base.py
--- ugit.bak/base.py    2022-01-31 23:17:29.172239208 +0800
+++ ugit/base.py        2022-01-31 23:21:01.776000182 +0800
@@ -63,6 +63,18 @@
     return result
 
 
+def get_working_tree():
+    result = {}
+    for root, _, filenames in os.walk('.'):
+        for filename in filenames:
+            path = os.path.relpath(f'{root}/{filename}')
+            if is_ignored(path) or not os.path.isfile(path):
+                continue
+            with open(path, 'rb') as f:
+                result[path] = data.hash_object(f.read())
+    return result
+
+
 def _empty_current_directory():
     for root, dirnames, filenames in os.walk('.', topdown=False):
         for filename in filenames:
diff -u ugit.bak/cli.py ugit/cli.py
--- ugit.bak/cli.py     2022-01-31 23:17:29.172239208 +0800
+++ ugit/cli.py 2022-01-31 23:19:48.771389519 +0800
@@ -52,6 +52,10 @@
     show_parser.set_defaults(func=show)
     show_parser.add_argument('oid', default='@', type=oid, nargs='?')
 
+    diff_parser = commands.add_parser('diff')
+    diff_parser.set_defaults(func=_diff)
+    diff_parser.add_argument('commit', default='@', type=oid, nargs='?')
+
     checkout_parser = commands.add_parser('checkout')
     checkout_parser.set_defaults(func=checkout)
     checkout_parser.add_argument('commit')
@@ -137,6 +141,14 @@
     sys.stdout.flush()
     sys.stdout.buffer.write(result)
 
+
+def _diff(args):
+    tree = args.commit and base.get_commit(args.commit).tree
+
+    result = diff.diff_trees(base.get_tree(tree), base.get_working_tree())
+    sys.stdout.flush()
+    sys.stdout.buffer.write(result)
+
 
 def checkout(args):
     base.checkout(args.commit)
```

测试：

```sh
[root@localhost repo]# ugit diff
[root@localhost repo]# echo 'test diff' > 1234.txt
[root@localhost repo]# ugit diff
--- old/1234.txt
+++ new/1234.txt
@@ -0,0 +1 @@
+test diff
```



## 只显示更改的文件

我们实现的命令 diff 很有用，但有时可能太冗长（如果您已经更改了很多行）。让我们实现一个更简单的版本作为 status 的一部分，它将只列出已更改的文件，而不列出完整的 diff。

在 diff.py 中，我们将添加 iter_changed_files()，它接收两个树，并输出所有已更改的路径以及更改类型（已删除、已创建、已修改）。然后在 status 中，我们将调用 iter_changed_files()，比较 HEAD 和工作树（就像 diff 一样）。

代码修改如下：

```diff
diff -u ugit.bak/cli.py ugit/cli.py
--- ugit.bak/cli.py     2022-01-31 23:31:37.885389099 +0800
+++ ugit/cli.py 2022-01-31 23:33:59.799622902 +0800
@@ -203,6 +203,11 @@
     else:
         print(f'HEAD detached at {HEAD[:10]}')
 
+    print('\nChanges to be committed:\n')
+    HEAD_tree = HEAD and base.get_commit(HEAD).tree
+    for path, action in diff.iter_changed_files(base.get_tree(HEAD_tree), base.get_working_tree()):
+        print(f'{action:>12}: {path}')
+
 
 def reset(args):
     base.reset(args.commit)


===========================================================
diff -u ugit.bak/diff.py ugit/diff.py
--- ugit.bak/diff.py    2022-01-31 23:31:37.885389099 +0800
+++ ugit/diff.py        2022-01-31 23:32:25.440383237 +0800
@@ -16,6 +16,15 @@
         yield(path, *oids)
 
 
+def iter_changed_files(t_from, t_to):
+    for path, o_from, o_to in compare_trees(t_from, t_to):
+        if o_from != o_to:
+            action = ('new file' if not o_from else
+                      'deleted' if not o_to else
+                      'modified')
+            yield path, action
+
+
 def diff_trees(t_from, t_to):
     output = b''
     for path, o_from, o_to in compare_trees(t_from, t_to):
```

测试：

```sh
[root@localhost repo]# ugit status
On branch master

Changes to be committed:

     deleted: 1
    modified: 2.txt
    new file: 1234.txt
[root@localhost repo]# ugit diff
--- old/1
+++ new/1
@@ -1 +0,0 @@
-1
--- old/2.txt
+++ new/2.txt
@@ -1,2 +1 @@
-777
-888
+666
--- old/1234.txt
+++ new/1234.txt
@@ -0,0 +1 @@
+test diff
```

完整代码如下：

- ugit/cli.py

```python
import argparse
import os
import sys
import textwrap
import subprocess

from . import base
from . import data
from . import diff


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

    show_parser = commands.add_parser('show')
    show_parser.set_defaults(func=show)
    show_parser.add_argument('oid', default='@', type=oid, nargs='?')

    diff_parser = commands.add_parser('diff')
    diff_parser.set_defaults(func=_diff)
    diff_parser.add_argument('commit', default='@', type=oid, nargs='?')

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
    if commit.parent:
        parent_tree = base.get_commit(commit.parent).tree

    _print_commit(args.oid, commit)
    result = diff.diff_trees(base.get_tree(parent_tree), base.get_tree(commit.tree))
    sys.stdout.flush()
    sys.stdout.buffer.write(result)


def _diff(args):
    tree = args.commit and base.get_commit(args.commit).tree

    result = diff.diff_trees(base.get_tree(tree), base.get_working_tree())
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
        if commit.parent:
            dot += f'"{oid}" -> "{commit.parent}"\n'

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

    print('\nChanges to be committed:\n')
    HEAD_tree = HEAD and base.get_commit(HEAD).tree
    for path, action in diff.iter_changed_files(base.get_tree(HEAD_tree), base.get_working_tree()):
        print(f'{action:>12}: {path}')


def reset(args):
    base.reset(args.commit)


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


def reset(oid):
    data.update_ref('HEAD', data.RefValue(symbolic=False, value=oid))


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


def iter_refs(prefix='', deref=True):
    refs = ['HEAD']
    for root, _, filenames in os.walk(f'{GIT_DIR}/refs/'):
        root = os.path.relpath(root, GIT_DIR)
        refs.extend(f'{root}/{name}' for name in filenames)

    for refname in refs:
        if not refname.startswith(prefix):
            continue
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



- ugit/diff.py

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

```



---

# merge

到目前为止，我们已经实现了对并行开发不同代码分支的良好支持。我们可以很容易地创建新的分支并在它们之间切换。现在让我们把平行的分支放在一起。这就是所谓的合并。

如果多人并行开发代码，并且希望将他们的更改合并到一个提交中，这将非常有用。例如，如果我们有：

```text
                HEAD
                v
o---o---o---o---o
     \
      --o---o
            ^
            some-branch
```

我们把HEAD合并成一个分支，我们应该得到：

```text
o---o---o---o---o
     \           \
      --o---o-----o < HEAD
            ^
            some-branch
```

这是最终目标，我们需要一些时间才能一小步一小步地实现。在这个变化中，我们创建了一个空的 merge 命令。

## 创建命令

代码修改如下：

```diff
diff -u ugit.bak/base.py ugit/base.py
--- ugit.bak/base.py    2022-01-31 23:54:49.815548442 +0800
+++ ugit/base.py        2022-02-01 00:42:26.063000843 +0800
@@ -135,6 +135,11 @@
     data.update_ref('HEAD', data.RefValue(symbolic=False, value=oid))
 
 
+def merge(other):
+    # TODO merge HEAD into other
+    pass
+
+
 def create_tag(name, oid):
     data.update_ref(f'{data.TAG_PREFIX}/{name}', data.RefValue(symbolic=False, value=oid))



===========================================================
diff -u ugit.bak/cli.py ugit/cli.py
--- ugit.bak/cli.py     2022-02-01 00:41:42.507664963 +0800
+++ ugit/cli.py 2022-02-01 00:40:34.435457884 +0800
@@ -80,6 +80,10 @@
     reset_parser.set_defaults(func=reset)
     reset_parser.add_argument('commit', type=oid)
 
+    merge_parser = commands.add_parser('merge')
+    merge_parser.set_defaults(func=merge)
+    merge_parser.add_argument('commit', type=oid)
+
     return parser.parse_args()
 
 
@@ -213,6 +217,10 @@
     base.reset(args.commit)
 
 
+def merge(args):
+    base.merge(args.commit)
+
+
 if __name__ == "__main__":
     main()
 
```



## 在工作目录中合并

我们继续前面的例子：

```text
                HEAD
                v
o---o---o---o---o
     \
      --o---o
            ^
            some-branch
```

正如你所记得的，我们想合并 HEAD 和 some-branch。假设 HEAD 和 some-branch 在它们的树中都有一个名为 main.py 的文件。合并时，我们希望创建一个新的 main.py，它将包含 HEAD 的 main.py 和 some-branch 的 main.py 的内容。

下面给出 HEAD 的 main.py 内容：

```python
def main ():
    print ("This functions is cool")
    print ("It prints stuff")
    print ("It can even return a number:")
    return 7
```

some-branch 的 main.py 内容：

```python
def main ():
    print ("1+1 = 2")
    print ("This functions is cool")
    print ("It prints stuff")
```

你可以看到他们有这些共同点：

```python
def main ():
    print ("This functions is cool")
    print ("It prints stuff")
```

也有一些独特的点。当将它们合并在一起时，理想情况下，我们会得到一个如下所示的组合 main.py：

```python
def main ():
    print ("1+1 = 2")
    print ("This functions is cool")
    print ("It prints stuff")
    print ("It can even return a number:")
    return 7
```

如你所见，它既包含了公共部分，也包含了各分支的差异部分。

我们之前使用的 diff 命令实际上可以完成这种合并。命令`diff -DHEAD file1 file2` 输出文件 1 和文件 2 的合并版本。如果我们在两个版本的 main.py 上运行 diff -DHEAD，我们会得到：

```sh
[root@localhost test]# diff -DHEAD 1.py 2.py 
def main ():
#ifndef HEAD
    print (*This functions is cool*)
#else /* HEAD */
    print ("1+1 = 2")
    print ("This functions is cool")
#endif /* HEAD */
    print ("It prints stuff")
#ifndef HEAD
    print ("It can even return a number:")
    return 7
#endif /* ! HEAD */
```

这非常接近我们想要的！(有 `#ifndef` 行标记每一行来自哪里，它们会增加一些干扰，但我们现在将忽略它们，稍后会解决这个问题。)

上面的例子是针对单个文件的。让我们写一些逻辑来将两个完整的树合并在一起。如您所知，我们有 base.read_tree() 获取一棵树并将其提取到工作目录中。我们现在将创建 base.read_tree_merged()，它将获取两个树，并将它们的合并版本提取到工作目录中。

我们需要调用 diff 命令进行文件合并。为此，我们将创建 diff.merge_blobs()，传入两个 oid 并返回它们的合并内容。

然后我们将创建 diff.merge_trees()，它获取两个树，并依次调用 diff.merge_blobs() 来合并树中的每两个文件，输出一个合并的树。

base.read_tree_merged() 只是调用 diff.merge_trees() 并将得到的合并树写入工作目录。

最后，base.merge() 获取 HEAD 的树和我们想要合并的分支的树，并调用 base.read_tree_merged()。

代码修改如下：

```diff
diff -u ugit.bak/base.py ugit/base.py
--- ugit.bak/base.py    2022-02-01 01:26:18.164852673 +0800
+++ ugit/base.py        2022-02-01 01:46:03.947894953 +0800
@@ -6,6 +6,7 @@
 from collections import namedtuple, deque
 
 from . import data
+from . import diff
 
 
 def init():
@@ -102,6 +103,14 @@
             f.write(data.get_object(oid))
 
 
+def read_tree_merged(t_HEAD, t_other):
+    _empty_current_directory()
+    for path, blob in diff.merge_trees(get_tree(t_HEAD), get_tree(t_other)).items():
+        os.makedirs(f'./{os.path.dirname(path)}', exist_ok=True)
+        with open(path, 'wb') as f:
+            f.write(blob)
+
+
 def commit(message):
     commit = f'tree {write_tree()}\n'
 
@@ -136,8 +145,13 @@
 
 
 def merge(other):
-    # TODO merge HEAD into other
-    pass
+    HEAD = data.get_ref('HEAD').value
+    assert HEAD
+    c_HEAD = get_commit(HEAD)
+    c_other = get_commit(other)
+
+    read_tree_merged(c_HEAD.tree, c_other.tree)
+    print('Merged in working tree')
 
 
 def create_tag(name, oid):
 
 
===========================================================
diff -u ugit.bak/diff.py ugit/diff.py
--- ugit.bak/diff.py    2022-02-01 01:26:18.164852673 +0800
+++ ugit/diff.py        2022-02-01 01:30:55.940752166 +0800
@@ -49,3 +49,28 @@
 
         return output
 
+
+def merge_trees(t_HEAD, t_other):
+    tree = {}
+    for path, o_HEAD, o_other in compare_trees(t_HEAD, t_other):
+        tree[path] = merge_blobs(o_HEAD, o_other)
+    return tree
+
+
+def merge_blobs(o_HEAD, o_other):
+    with Temp() as f_HEAD, Temp() as f_other:
+        for oid, f in ((o_HEAD, f_HEAD), (o_other, f_other)):
+            if oid:
+                f.write(data.get_object(oid))
+                f.flush()
+
+        with subprocess.Popen(
+            ['diff',
+             '-DHEAD', 
+             f_HEAD.name,
+             f_other.name
+            ], stdout=subprocess.PIPE) as proc:
+            output, _ = proc.communicate()
+
+        return output
+
```

测试：

```sh
[root@localhost repo]# ls
1234.txt  2.txt
[root@localhost repo]# cp ../test/1.py main.py
[root@localhost repo]# rm 1234.txt 2.txt -f
[root@localhost repo]# ugit commit -m "main master"
blob 396497ecae64a7108508e808bbca6cdfb9f6b21d ./main.py
tree a71e0603ae4b61ad10980b735dd6ab66e76f3bc5 .
63e3ffa9abc3b4804404666563950b55a30a311e
[root@localhost repo]# ugit branch some-branch
Branch some-branch created at 63e3ffa9ab
[root@localhost repo]# ugit checkout some-branch
[root@localhost repo]# ls
main.py
[root@localhost repo]# cp ../test/2.py main.py 
cp：是否覆盖"main.py"？ y
[root@localhost repo]# ugit commit -m "main some-branch"
blob b9cd8535d59e0fb1be82bd2d261bd9037dab7ef3 ./main.py
tree 245913d4d6a1b1ec6d82f7e1e71c14c4afcb6230 .
4554de6d666c2a53af4a77228c156c0fb1e5f701
[root@localhost repo]# ugit checkout master
[root@localhost repo]# cat main.py 
def main ():
    print (*This functions is cool*)
    print ("It prints stuff")
    print ("It can even return a number:")
    return 7
[root@localhost repo]# ugit merge some-branch
Merged in working tree
[root@localhost repo]# cat main.py 
def main ():
#ifndef HEAD
    print (*This functions is cool*)
#else /* HEAD */
    print ("1+1 = 2")
    print ("This functions is cool")
#endif /* HEAD */
    print ("It prints stuff")
#ifndef HEAD
    print ("It can even return a number:")
    return 7
#endif /* ! HEAD */
```



## 支持多个父提交

你还记得我们想要的合并是这样的吗？

```text
o---o---o---o---o
     \           \
      --o---o-----o < HEAD
            ^
            some-branch
```

HEAD 提交是将两个提交合并在一起的提交，因此 HEAD 提交有两个父提交。目前 ugit 不支持这一点，所以我们将扩展 commit 对象来可选地支持多个父对象。

base.py 中的 Commit 对象现在将包含父对象列表，而不是单个父对象。

大的修改在 base.get_commit() 中，它解析提交对象：对于我们遇到的每一个“父”行，我们都会将其附加到父列表中。因此，如果提交有一个父元素，列表将包含一个元素。如果提交有多个父项，列表将包含所有父项。如果提交没有父项(第一次提交)，列表将为空。

其余的修改是简单的重构，以支持新的提交对象。

修改代码如下：

```diff
diff -u ugit.bak/base.py ugit/base.py
--- ugit.bak/base.py    2022-02-01 01:55:59.398786327 +0800
+++ ugit/base.py        2022-02-01 02:17:35.865819585 +0800
@@ -180,9 +180,9 @@
     return os.path.relpath(HEAD, data.BRANCH_PREFIX)
 
 
-Commit = namedtuple('Commit', ['tree', 'parent', 'message'])
+Commit = namedtuple('Commit', ['tree', 'parents', 'message'])
 def get_commit(oid):
-    parent = None
+    parents = []
 
     commit = data.get_object(oid, 'commit').decode()
     lines = iter(commit.splitlines())
@@ -191,12 +191,12 @@
         if key == 'tree':
             tree = value
         elif key == 'parent':
-            parent = value
+            parents.append(value)
         else:
             assert False, f'Unknown field {key}'
 
     message = '\n'.join(lines)
-    return Commit(tree=tree, parent=parent, message=message)
+    return Commit(tree=tree, parents=parents, message=message)
 
 
 
@@ -212,8 +212,10 @@
         yield oid
 
         commit = get_commit(oid)
-        # Return parent next
-        oids.appendleft(commit.parent)
+        # Return first parent next
+        oids.extendleft(commit.parents[:1])
+        # Return other parents later
+        oids.extend(commit.parents[1:])
 
 
 def get_oid(name):



===========================================================
diff -u ugit.bak/cli.py ugit/cli.py
--- ugit.bak/cli.py     2022-02-01 01:55:59.398786327 +0800
+++ ugit/cli.py 2022-02-01 02:19:04.117242075 +0800
@@ -137,8 +137,8 @@
         return
     commit = base.get_commit(args.oid)
     parent_tree = None
-    if commit.parent:
-        parent_tree = base.get_commit(commit.parent).tree
+    if commit.parents:
+        parent_tree = base.get_commit(commit.parents[0]).tree
 
     _print_commit(args.oid, commit)
     result = diff.diff_trees(base.get_tree(parent_tree), base.get_tree(commit.tree))
@@ -187,8 +187,8 @@
     for oid in base.iter_commits_and_parents(oids):
         commit = base.get_commit(oid)
         dot += f'"{oid}" [shape=box style=filled label="{oid[:10]}"]\n'
-        if commit.parent:
-            dot += f'"{oid}" -> "{commit.parent}"\n'
+        for parent in commit.parents:
+            dot += f'"{oid}" -> "{parent}"\n'
 
     dot += '}'
     print(dot)
```



## 删除引用

为了准备下一次更改，让我们创建 data.delete_ref() 来移除现有的引用。目前没有使用，但很快就会有用。

代码修改如下：

```diff
diff -u ugit.bak/data.py ugit/data.py
--- ugit.bak/data.py    2022-02-01 02:21:48.662275510 +0800
+++ ugit/data.py        2022-02-01 02:22:51.162911599 +0800
@@ -34,6 +34,11 @@
     return _get_ref_internal(ref, deref)[1]
 
 
+def delete_ref(ref, deref=True):
+    ref = _get_ref_internal(ref, deref)[0]
+    os.remove(f'{GIT_DIR}/{ref}')
+
+
 def _get_ref_internal(ref, deref):
     ref_path = f'{GIT_DIR}/{ref}'
     value = None
```



## 在提交中记录所有parent

之前我们在工作目录中合并了两个提交，现在我们将确保一旦我们提交了合并的工作目录，就创建了一个合并提交。

ugit merge 命令将把我们要合并的分支存储在一个名为 MERGE_HEAD 的新引用中。所以在我们前面的例子中：

```text
                HEAD
                v
o---o---o---o---o
     \
      --o---o
            ^
            some-branch
```

在运行 ugit merge some-branch 之后，将创建 MERGE_HEAD 并指向 some-branch。MERGE_HEAD 的出现告诉 ugit，下一次提交是一个合并提交，有两个父级—— HEAD 和 MERGE_HEAD。

```text
                HEAD
                v
o---o---o---o---o
     \
      --o---o < MERGE_HEAD
            ^
            some-branch
```

因此在 ugit 提交时，ugit 将创建一个看起来像我们想要的提交：

```text
o---o---o---o---o
     \           \
      --o---o-----o < HEAD
            ^
            some-branch
```

此时可以删除 MERGE_HEAD。

您可以看到我们更改了 base.merge 来设置 MERGE_HEAD，base.commit 来考虑 MERGE_HEAD，cli.status 来帮助通知用户我们正在进行合并。

代码修改如下：

```diff
diff -u ugit.bak/base.py ugit/base.py
--- ugit.bak/base.py    2022-02-01 04:39:10.215067258 +0800
+++ ugit/base.py        2022-02-01 04:42:37.671936318 +0800
@@ -118,6 +118,11 @@
     if HEAD:
         commit += f'parent {HEAD}\n'
 
+    MERGE_HEAD = data.get_ref('MERGE_HEAD').value
+    if MERGE_HEAD:
+        commit += f'parent {MERGE_HEAD}\n'
+        data.delete_ref('MERGE_HEAD', deref=False)
+
     commit += '\n'
     commit += f'{message}\n'
 
@@ -150,8 +155,10 @@
     c_HEAD = get_commit(HEAD)
     c_other = get_commit(other)
 
+    data.update_ref('MERGE_HEAD', data.RefValue(symbolic=False, value=other))
+
     read_tree_merged(c_HEAD.tree, c_other.tree)
-    print('Merged in working tree')
+    print('Merged in working tree\nPlease commit')
 
 
 def create_tag(name, oid):


===========================================================
diff -u ugit.bak/cli.py ugit/cli.py
--- ugit.bak/cli.py     2022-02-01 04:39:10.215067258 +0800
+++ ugit/cli.py 2022-02-01 04:40:10.479706552 +0800
@@ -207,6 +207,10 @@
     else:
         print(f'HEAD detached at {HEAD[:10]}')
 
+    MERGE_HEAD = data.get_ref('MERGE_HEAD').value
+    if MERGE_HEAD:
+        print(f'Merging with {MERGE_HEAD[:10]}')
+
     print('\nChanges to be committed:\n')
     HEAD_tree = HEAD and base.get_commit(HEAD).tree
     for path, action in diff.iter_changed_files(base.get_tree(HEAD_tree), base.get_working_tree()):
```

测试：

```sh
[root@localhost repo]# cat .ugit/refs/heads/master 
63e3ffa9abc3b4804404666563950b55a30a311e
[root@localhost repo]# cat .ugit/refs/heads/some-branch 
4554de6d666c2a53af4a77228c156c0fb1e5f701
[root@localhost repo]# ugit status
On branch master

Changes to be committed:
[root@localhost repo]# ugit merge some-branch
Merged in working tree
Please commit
[root@localhost repo]# cat .ugit/MERGE_HEAD 
4554de6d666c2a53af4a77228c156c0fb1e5f701
[root@localhost repo]# ugit commit -m "merge"
blob 3d99cb61600491f8f22872d324a6b8b3ad21733f ./main.py
tree 1181a062e8ebf2f824466cfac2ead77ab5d8fd4c .
5a18262c1e8af2d004cb01315a2c14a39010998c
[root@localhost repo]# ugit cat-file 5a18262c1e8af2d004cb01315a2c14a39010998c
tree 1181a062e8ebf2f824466cfac2ead77ab5d8fd4c
parent 63e3ffa9abc3b4804404666563950b55a30a311e
parent 4554de6d666c2a53af4a77228c156c0fb1e5f701

merge
```



由于我们添加了 MERGE_HEAD，所以在运行 ugit k 时将其可视化也将非常有用。另外由于 MERGE_HEAD 可能不存在，所以在 iter_refs 中添加一个复选标记，以避免返回不存在的 refs。

代码修改如下：

```diff
diff -u ugit.bak/data.py ugit/data.py
--- ugit.bak/data.py    2022-02-01 05:29:05.111636286 +0800
+++ ugit/data.py        2022-02-01 05:31:31.914949005 +0800
@@ -56,7 +56,7 @@
 
 
 def iter_refs(prefix='', deref=True):
-    refs = ['HEAD']
+    refs = ['HEAD', 'MERGE_HEAD']
     for root, _, filenames in os.walk(f'{GIT_DIR}/refs/'):
         root = os.path.relpath(root, GIT_DIR)
         refs.extend(f'{root}/{name}' for name in filenames)
@@ -64,7 +64,9 @@
     for refname in refs:
         if not refname.startswith(prefix):
             continue
-        yield refname, get_ref(refname, deref)
+        ref = get_ref(refname, deref)
+        if ref.value:
+            yield refname, ref
 
 
 def hash_object(data, type_="blob"):
```



## 计算提交的公共祖先

这个命令叫做 merge-base，它将接收两个提交 oid 并找到它们的共同祖先。换句话说，它会找到它们共享的第一个父提交。从图形上看，它是这样的：

```text
    commit C    commit A
    v           v
o---o---o---o---o
     \
      --o---o
            ^
            commit B
```

提交 C 是提交 A 和 B 的第一个共同祖先。

这个命令本身不是很有用，但是我们将在下面的更改中解释它的动机。名称 merge-base 暗示我们将改进我们的合并命令。

它的操作相当暴力，它将第一次提交的所有父项保存到一个列表中，并按照祖先顺序迭代第二次提交的父项，直到找到它们第一个公共提交。

代码更改如下：

```diff
diff -u ugit.bak/base.py ugit/base.py
--- ugit.bak/base.py    2022-02-01 05:33:14.301676929 +0800
+++ ugit/base.py        2022-02-01 05:46:39.186526522 +0800
@@ -161,6 +161,14 @@
     print('Merged in working tree\nPlease commit')
 
 
+def get_merge_base(oid1, oid2):
+    parents1 = set(iter_commits_and_parents({oid1}))
+
+    for oid in iter_commits_and_parents({oid2}):
+        if oid in parents1:
+            return oid
+
+
 def create_tag(name, oid):
     data.update_ref(f'{data.TAG_PREFIX}/{name}', data.RefValue(symbolic=False, value=oid))



===========================================================
diff -u ugit.bak/cli.py ugit/cli.py
--- ugit.bak/cli.py     2022-02-01 05:33:14.301676929 +0800
+++ ugit/cli.py 2022-02-01 05:45:53.159028042 +0800
@@ -84,6 +84,11 @@
     merge_parser.set_defaults(func=merge)
     merge_parser.add_argument('commit', type=oid)
 
+    merge_base_parser = commands.add_parser('merge-base')
+    merge_base_parser.set_defaults(func=merge_base)
+    merge_base_parser.add_argument('commit1', type=oid)
+    merge_base_parser.add_argument('commit2', type=oid)
+
     return parser.parse_args()
 
 
@@ -225,6 +230,10 @@
     base.merge(args.commit)
 
 
+def merge_base(args):
+    print(base.get_merge_base(args.commit1, args.commit2))
+
+
 if __name__ == "__main__":
     main()
 
```



## Three-way merge

接下来，让我们通过使用一种称为 **three-way merge** 的技术来提高合并的质量。在讨论 three-way merge 之前，我们先来讨论 two-way merge，这是我们到目前为止使用的技术。

two-way merge 意味着我们查看同一个文件的两个版本，并尝试将它们合并成一个版本。有时候效果很好。例如，如果我们有：

- Version A

```python
def be_a_cat ():
    print ("Meow")
    while True:
          print ("Purr")
    return True

def be_a_dog ():
    print ("Bark!")
    return False
```

- Version B

```python
def be_a_cat ():
    print ("Meow")
    return True

def be_a_dog ():
    print ("Bark!")
    print ("Wiggle tail")
    return False
```

将它们合并在一起非常容易，因为从上下文中可以清楚地看出哪些是公共部分，哪些部分是在每个版本中添加的。

- A and B merged

```python
def be_a_cat ():
    print ("Meow")
    while True:
          print ("Purr")
    return True

def be_a_dog ():
    print ("Bark!")
    print ("Wiggle tail")
    return False
```

这正是我们的 diff 命令要做的。但是让我们看一个 two-way merge 效果不好的例子：

- Version A

```python
def be_a_cat ():
    print ("Sleep")
    return True

def be_a_dog ():
    print ("Bark!")
    return False
```

- Version B

```python
def be_a_cat ():
    print ("Meow")
    return True

def be_a_dog ():
    print ("Eat homework")
    return False
```

我们应该如何合并它们？我们在每份书面陈述中都有冲突。这意味着我们能做的最好的事情就是把两个改变都留下，让用户来决定：

- A and B merged

```text
def be_a_cat ():
<<<<<<< Version A
    print ("Sleep")
=======
    print ("Meow")
>>>>>>> Version B
    return True

def be_a_dog ():
<<<<<<< Version A
    print ("Bark!")
=======
    print ("Eat homework")
>>>>>>> Version B
    return False
```

现在假设版本 A 和版本 B 是同一个文件，但是来自不同的分支。比如这样：

```text
                version A
                v
o---o---o---o---o
     \
      --o---o
            ^
            version B
```

还有一个有趣的信息，我们在 two-way merge 过程中没有用到：文件可能存在于它们的共同祖先中，它可能会给我们关于合并的有用上下文。例如，假设我们知道这一点：

- Version A

```python
def be_a_cat ():
    print ("Sleep")
    return True

def be_a_dog ():
    print ("Bark!")
    return False
```

- Version B

```python
def be_a_cat ():
    print ("Meow")
    return True

def be_a_dog ():
    print ("Eat homework")
    return False
```

- 公共祖先

```python
def be_a_cat ():
    print ("Meow")
    return True

def be_a_dog ():
    print ("Bark!")
    return False
```

我们其实可以理解为，版本 A 改的只是 be_a_cat 的打印语句，而版本 B 改的只是 be_a_dog 的打印语句。这意味着每个版本都改变了不相关的代码片段，我们可以安全地将它们合并到：

```python
def be_a_cat ():
    print ("Sleep")
    return True

def be_a_dog ():
    print ("Eat homework")
    return False
```

这被称为 **three-way merge**：一种使用两个文件的共同祖先作为向导来合并这两个文件的算法。它极大地提高了合并质量（意味着每次合并的冲突更少）。如果你想了解更多，请在网上搜索:)。

我们希望在 ugit 中使用 three-way merge。为此，我们将使用前面实现的 get_merge_base，并修改 diff.merge_trees 以接受一个共同的祖先称为 base。

我们不需要很多逻辑上的改变来实现它。我们的 compare_trees 函数可以比较任意数量的树，所以我们也可以在那里添加 base tree。然后 merge_blobs 将调用 diff3 命令，该命令执行 three-way merge。

您可以尝试使用生成的代码来查看合并结果是否更好。

代码修改如下：

```diff
diff -u ugit.bak/base.py ugit/base.py
--- ugit.bak/base.py    2022-02-01 06:05:42.241911224 +0800
+++ ugit/base.py        2022-02-01 06:14:55.958547455 +0800
@@ -103,9 +103,9 @@
             f.write(data.get_object(oid))
 
 
-def read_tree_merged(t_HEAD, t_other):
+def read_tree_merged(t_base, t_HEAD, t_other):
     _empty_current_directory()
-    for path, blob in diff.merge_trees(get_tree(t_HEAD), get_tree(t_other)).items():
+    for path, blob in diff.merge_trees(get_tree(t_base), get_tree(t_HEAD), get_tree(t_other)).items():
         os.makedirs(f'./{os.path.dirname(path)}', exist_ok=True)
         with open(path, 'wb') as f:
             f.write(blob)
@@ -152,12 +152,14 @@
 def merge(other):
     HEAD = data.get_ref('HEAD').value
     assert HEAD
+    merge_base = get_merge_base(other, HEAD)
+    c_base = get_commit(merge_base)
     c_HEAD = get_commit(HEAD)
     c_other = get_commit(other)
 
     data.update_ref('MERGE_HEAD', data.RefValue(symbolic=False, value=other))
 
-    read_tree_merged(c_HEAD.tree, c_other.tree)
+    read_tree_merged(c_base.tree, c_HEAD.tree, c_other.tree)
     print('Merged in working tree\nPlease commit')
 


===========================================================
diff -u ugit.bak/diff.py ugit/diff.py
--- ugit.bak/diff.py    2022-02-01 06:05:42.241911224 +0800
+++ ugit/diff.py        2022-02-01 06:09:34.823395069 +0800
@@ -50,27 +50,30 @@
         return output
 
 
-def merge_trees(t_HEAD, t_other):
+def merge_trees(t_base, t_HEAD, t_other):
     tree = {}
-    for path, o_HEAD, o_other in compare_trees(t_HEAD, t_other):
-        tree[path] = merge_blobs(o_HEAD, o_other)
+    for path, o_base, o_HEAD, o_other in compare_trees(t_base, t_HEAD, t_other):
+        tree[path] = merge_blobs(o_base, o_HEAD, o_other)
     return tree
 
 
-def merge_blobs(o_HEAD, o_other):
-    with Temp() as f_HEAD, Temp() as f_other:
-        for oid, f in((o_HEAD, f_HEAD), (o_other, f_other)):
+def merge_blobs(o_base, o_HEAD, o_other):
+    with Temp() as f_base, Temp() as f_HEAD, Temp() as f_other:
+
+        # Write blobs to files
+        for oid, f in ((o_base, f_base), (o_HEAD, f_HEAD), (o_other, f_other)):
             if oid:
                 f.write(data.get_object(oid))
                 f.flush()
 
         with subprocess.Popen(
-            ['diff',
-             '-DHEAD', 
-             f_HEAD.name,
-             f_other.name
+            ['diff3', '-m',
+             '-L', 'HEAD', f_HEAD.name,
+             '-L', 'BASE', f_base.name,
+             '-L', 'MERGE_HEAD', f_other.name,
             ], stdout=subprocess.PIPE) as proc:
             output, _ = proc.communicate()
+            assert proc.returncode in (0, 1)
 
         return output
```

测试：

```sh
# 制造公共历史
[root@localhost repo]# cat main.py 
def be_a_cat ():
    print ("Meow")
    return True

def be_a_dog ():
    print ("Bark!")
    return False
[root@localhost repo]# ugit commit -m "common"
blob 7483051875d1cb0682b068aab8078221dce05b7b ./main.py
tree e0ef49ac693f4707f677dbbbc8d39dce5857970e .
9eb21e5df19f4cf414cc74272726625fbedf270c

# 创建分支 A 和 B
[root@localhost repo]# ugit branch A
Branch A created at 9eb21e5df1
[root@localhost repo]# ugit branch B
Branch B created at 9eb21e5df1

# 修改 A 分支文件
[root@localhost repo]# ugit checkout A
[root@localhost repo]# vim main.py 
[root@localhost repo]# cat main.py 
def be_a_cat ():
    print ("Sleep")
    return True

def be_a_dog ():
    print ("Bark!")
    return False
[root@localhost repo]# ugit commit -m "A"
blob 72487ede7172918c04f73ba9fc0ad32303a76189 ./main.py
tree 03392838015f39176332fb64ee99afbdd9acd22b .
b3927dd05233dd210acbf595f9157c680c3cc918

# 修改 B 分支文件
[root@localhost repo]# ugit checkout B
[root@localhost repo]# vim main.py 
[root@localhost repo]# cat main.py 
def be_a_cat ():
    print ("Meow")
    return True

def be_a_dog ():
    print ("Eat homework")
    return False
[root@localhost repo]# ugit commit -m "B"
blob 7ec6a2f45c98a231d7286803b29136464c725ad0 ./main.py
tree d12fa97fa6d2f189ee8f98dd478fb58d4fc6e9d0 .
56fc2e3b3014fe89c754c9da0412c384e6f330a9

# 测试合并
[root@localhost repo]# ugit merge A
Merged in working tree
Please commit
[root@localhost repo]# ugit status
On branch B
Merging with b3927dd052

Changes to be committed:

    modified: main.py
[root@localhost repo]# ugit commit -m "merge"
blob 3ec566c1e60d577c5177d7cee4a6efcc091c7a37 ./main.py
tree 4a6a712783bcda1801ecf73ae7a8255a8510a7d5 .
4c2da930364978f144fb27c564fa14022cf7de90
[root@localhost repo]# cat main.py 
def be_a_cat ():
    print ("Sleep")
    return True

def be_a_dog ():
    print ("Eat homework")
    return False
```



## Fast-forward merge

到目前为止，我们一直将合并视为两个独立的发展方向的结合，比如：

```text
                HEAD
                v
o---o---o---o---o
     \
      --o---o
            ^
            some-branch
```

结果是：

```text
o---o---o---o---o
     \           \
      --o---o-----o < HEAD
            ^
            some-branch
```

但是有一个边缘案例值得讨论——如果 HEAD 没有和分支并行推进，我们有下面这个情况怎么办？

```text
    HEAD
    v
o---o
     \
      --o---o
            ^
            some-branch
```

目前我们的代码并不关心这种情况，当合并 HEAD 和 some-branch 时，它会创建一个合并提交，如下所示：

```text
o---o---------\
     \         \
      --o---o---o < HEAD
            ^
            some-branch
```

但是这个提交没有携带任何信息！我们可以去掉它，而是直接将 HEAD 前进到某个分支：

这种类型的合并称为 fast-forward 合并。

为了知道我们是否可以执行快进合并，我们需要检查 HEAD 是否是我们想要合并到的分支的祖先。如果 HEAD 和某个分支的共同祖先是 HEAD 本身，这意味着 HEAD 是某个分支的祖先。这就是为什么在合并中我们要检查 merge_base == HEAD。如果是这样的话，我们将把 HEAD 更新到新的位置，并读取该树。

在某些情况下，即使可以进行快进合并，也可能仍然希望创建合并提交。例如，您可能希望记录发生合并的事实。我们暂时忽略这个用例。

代码修改如下：

```diff
diff -u ugit.bak/base.py ugit/base.py
--- ugit.bak/base.py    2022-02-01 06:40:06.617481893 +0800
+++ ugit/base.py        2022-02-01 06:42:40.013944960 +0800
@@ -153,12 +153,19 @@
     HEAD = data.get_ref('HEAD').value
     assert HEAD
     merge_base = get_merge_base(other, HEAD)
-    c_base = get_commit(merge_base)
-    c_HEAD = get_commit(HEAD)
     c_other = get_commit(other)
 
+    # Handle fast-forward merge
+    if merge_base == HEAD:
+        read_tree(c_other.tree)
+        data.update_ref('HEAD', data.RefValue(symbolic=False, value=other))
+        print('Fast-forward merge, no need to commit')
+        return
+
     data.update_ref('MERGE_HEAD', data.RefValue(symbolic=False, value=other))
 
+    c_base = get_commit(merge_base)
+    c_HEAD = get_commit(HEAD)
     read_tree_merged(c_base.tree, c_HEAD.tree, c_other.tree)
     print('Merged in working tree\nPlease commit')
 
```

测试：

```sh
[root@localhost repo]# echo 1234 > 1234.txt
[root@localhost repo]# ugit commit -m "1234"
blob f8610f64c2aff2a57bfd6614f9841424cbcd1612 ./main.py
blob c928f7711483160a5245c9da863f775563cb3584 ./1
blob e0a3793226f28531f470c7f03aaf256e0a9800c1 ./1234.txt
tree 793b3552f4ce9bf86431ba677b1fbecb6487ffe4 .
4af67da167465c46560cae61aa6109727c56394c
[root@localhost repo]# ugit checkout A
[root@localhost repo]# ugit merge B
Fast-forward merge, no need to commit
```

完整代码如下：

- ugit/cli.py

```python
import argparse
import os
import sys
import textwrap
import subprocess

from . import base
from . import data
from . import diff


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

    show_parser = commands.add_parser('show')
    show_parser.set_defaults(func=show)
    show_parser.add_argument('oid', default='@', type=oid, nargs='?')

    diff_parser = commands.add_parser('diff')
    diff_parser.set_defaults(func=_diff)
    diff_parser.add_argument('commit', default='@', type=oid, nargs='?')

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
    tree = args.commit and base.get_commit(args.commit).tree

    result = diff.diff_trees(base.get_tree(tree), base.get_working_tree())
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
    for path, action in diff.iter_changed_files(base.get_tree(HEAD_tree), base.get_working_tree()):
        print(f'{action:>12}: {path}')


def reset(args):
    base.reset(args.commit)


def merge(args):
    base.merge(args.commit)


def merge_base(args):
    print(base.get_merge_base(args.commit1, args.commit2))


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
from . import diff


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


def read_tree_merged(t_base, t_HEAD, t_other):
    _empty_current_directory()
    for path, blob in diff.merge_trees(get_tree(t_base), get_tree(t_HEAD), get_tree(t_other)).items():
        os.makedirs(f'./{os.path.dirname(path)}', exist_ok=True)
        with open(path, 'wb') as f:
            f.write(blob)


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
    read_tree(commit.tree)

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
        read_tree(c_other.tree)
        data.update_ref('HEAD', data.RefValue(symbolic=False, value=other))
        print('Fast-forward merge, no need to commit')
        return

    data.update_ref('MERGE_HEAD', data.RefValue(symbolic=False, value=other))

    c_base = get_commit(merge_base)
    c_HEAD = get_commit(HEAD)
    read_tree_merged(c_base.tree, c_HEAD.tree, c_other.tree)
    print('Merged in working tree\nPlease commit')


def get_merge_base(oid1, oid2):
    parents1 = set(iter_commits_and_parents({oid1}))

    for oid in iter_commits_and_parents({oid2}):
        if oid in parents1:
            return oid


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



- ugit/diff.py

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
        tree[path] = merge_blobs(o_base, o_HEAD, o_other)
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

