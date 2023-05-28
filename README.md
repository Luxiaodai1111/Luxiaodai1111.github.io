# 🎨 前言

>[!NOTE]
>
>- ✨ 本站主要分享计算机数据存储方面的一些知识，既是自己学习的过程，也希望完成一幅存储知识体系的指引地图
>
>
>
>- 🍉 What I cannot create, I do not understand.
>
>
>
>- 🔗 本站基于 [docsify](https://docsify.js.org/) 搭建，字体为 [霞鹜文楷](https://github.com/lxgw/LxgwWenKai)
>
>
>
>- 📧 文章若有错误或描述不准确之处，请邮件联系 497976404@qq.com



​	

---

<!-- panels:start -->

<!-- div:title-panel -->

# 💿 Storage is not magic

<!-- div:left-panel -->

>   ## [📁 文件系统](posts/文件系统/Content.md)
>
>   Linux 文件系统相关知识如 VFS、XFS、etc.

> ## [🧱 块 I/O 子系统](posts/块IO子系统/Content.md)
>
> Linux 块 IO 子系统相关知识，像 Multi Disk、Device Mapper、Bcache 等模块都是工作在这一层

>   ## [💾 存储硬件](posts/存储硬件/Content.md)
>
>   SSD / HDD 等存储介质相关知识

<!-- div:right-panel -->

>   ## [⚡ 高速缓存](posts/高速缓存/Content.md)
>
>   Cache 和内存模型相关知识

>## [☔ 存储安全](posts/存储安全/Content.md)
>
>RAID、EC、快照、etc.

>## [🚀 存储加速](posts/存储加速/Content.md)
>
>SPDK、DPDK 加速套件（挖坑未填）

<!-- panels:end -->



---

# 🌊 分布式

>## [🎯 MIT 6.824](posts/MIT6.824/Content.md)
>
>麻省理工分布式课程，这门课每节课都会精读一篇分布式系统领域的经典论文，并由此传授分布式系统设计与实现的重要原则和关键技术。同时其课程 Project 也是以其难度之大而闻名遐迩，4 个编程作业循序渐进带你实现一个基于 Raft 共识算法的 KV-store 框架，让你在痛苦的 debug 中体会并行与分布式带来的随机性和复杂性。

>## 🛶 Raft专题
>
>[raft.github.io](https://raft.github.io/ ) 收录了关于 Raft 的论文、课程、书籍等资料，以及相关开源项目，帮你彻底搞懂 Raft
>
>[Raft 动画演示](http://kailing.pub/raft/index.html)
>
>[Raft 的运行情况可视化](https://raft.github.io/raftscope/index.html)
>
>[📄 In Search of an Understandable Consensus Algorithm (Extended Version)](posts/经典论文导读/Raft-extended.md)
>
>[📄 Raft 博士毕业论文翻译](posts/经典论文导读/raft博士论文翻译.md)

​	

---

# ☁️ 对象存储

>   ## [🌩️ MinIO](posts/MinIO/Content.md)
>
>   MinIO offers high-performance, S3 compatible object storage. Native to Kubernetes, MinIO is the only object storage suite available on every public cloud, every Kubernetes distribution, the private cloud and the edge. MinIO is software-defined and is 100% open source under GNU AGPL v3.

> ## [🐍 DIY Git in Python](posts/u-git/Content.md)
>
> ugit 是一个类似 git 的版本控制系统的简单实现。
>
> 它使用 Python 实现 Git 版本控制系统的过程。可以让我们更好地理解 Git 的原理和工作方式，同时可以更灵活地自定义 Git 的功能和行为。这个过程可以包括创建 Git 仓库、添加文件、提交更改、查看提交历史等基本步骤，也可以实现更高级的功能，例如分支和合并。

​	

---

# 📼 数据库

[数据库排名 DB-Engines Ranking](https://db-engines.com/en/ranking)

>## [🚀 存储引擎专题](posts/存储引擎/Content.md)
>
>B+ Tree / LSM / Hash 等存储引擎以及相关的实现

>## [🔍 How Does a Database Work](posts/How-Does-a-Database-Work/Content.md)
>
>"How Does a Database Work" 是一个关于如何构建一个简单的数据库管理系统的教程网站。该教程使用 C 语言和 Unix 操作系统，涵盖了数据库的基本原理和实现细节，包括储存和读取数据、查询优化、索引、事务等方面内容。

>   ## [⚓ etcd](posts/etcd/Content.md)
>
>   A distributed, reliable key-value store for the most critical data of a distributed system

>   ## [🔥 redis](posts/redis/Content.md)
>
>   The open source, in-memory data store used by millions of developers as a database, cache, streaming engine, and message broker.

​	



---

# ®️ Rust 项目

>## [📚 Rust 基础在线学习](https://course.rs/about-book.html)
>
>《Rust 语言圣经》涵盖了 Rust 语言从入门到精通的全部知识。该书目前还未完成，正处于积极更新的状态。

>## [☢️ rCore-Tutorial-Book](posts/rCore-Tutorial-Book/Content.md)
>
>这本教程旨在一步一步展示如何从零开始用 Rust 语言写一个基于 RISC-V 架构的类 Unix 内核。
>
>另外麻省理工得 6.S081 也是很好的操作系统课程，不过我更愿意尝试用 Rust 去构造操作系统，这看起来更有意思一些
>
>[MIT 6.S081（unimplemented）](posts/MIT6.S081/Content.md)



​	

---

# 📄 经典论文导读

>每一个领域内，都有非常多优秀的认可度高的会议或者期刊。对于计算机领域而言，一般的分类方式是 [CCF](https://www.ccf.org.cn/) 评级，从 A 到 C 含金量依次降低。
>
>此外还可以参考这个网站的论文分级：[Computer Science Conference Rankings](https://link.zhihu.com/?target=http%3A//webdocs.cs.ualberta.ca/~zaiane/htmldocs/ConfRanking.html)
>
>[🔑 开始探索](posts/经典论文导读/Content.md)

​	

---

# 🔮 Others

> ## [🌌 Software Engineering at Google](posts/Software-Engineering-at-Google/Content.md)
>
> "Software Engineering at Google" 是一本由谷歌员工编写的书籍，介绍了谷歌在软件工程方面的实践和经验。该书由多位谷歌员工合作编写，包括著名的软件工程师和项目经理。
>
> 该书的内容涵盖了谷歌的软件开发流程、测试、代码审查、软件架构、部署、运维等多个方面，介绍了谷歌在这些方面的最佳实践和经验。此外，该书还讨论了一些谷歌独特的软件工程实践，例如代码共享和内部工具的使用。
>
> 该书的目标读者是软件工程师和项目经理，他们希望了解谷歌在软件工程方面的最佳实践和经验，并将其应用到自己的工作中。该书的内容深入浅出，既适合有经验的软件工程师，也适合新手。

> ## 🖥️ [The Art of Command Line](https://github.com/jlevy/the-art-of-command-line)
>
> "The Art of Command Line" 是一本由一个名为 "jlevy" 的开发者编写的在线书籍，介绍了如何使用命令行工具进行各种任务。该书涵盖了许多常见的命令行工具，包括 bash、grep、sed、awk、curl 等。
>
> 该书的目的是帮助读者理解命令行的工作原理和基本概念，以及如何使用命令行工具来加快各种任务的执行速度。该书的作者提供了大量的示例代码和命令行演示，可以帮助读者更好地理解命令行工具的使用方法。
>
> 此外，该书还介绍了一些高级命令行工具和技术，例如管道、重定向、正则表达式等。这些工具和技术可以帮助读者更高效地使用命令行工具，并在实际工作中提高生产力。

>## [🔧 杂七杂八](posts/others/Content.md)
>
>杂七杂八的记录
