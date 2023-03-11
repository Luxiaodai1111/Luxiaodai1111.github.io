# 🎨 前言

>[!NOTE]
>
>- ✨ 本站主要分享计算机数据存储方面的一些知识，既是自己学习的过程，也希望完成一幅存储的指引地图。
>
>
>
>- 🍉 写（or 搬运整理）文章是件很花精力的事情，但是分享本身也是巩固学习的一个过程，在此也很感谢那些提供参考的书籍和文章。
>
>
>
>- 🔗 本站基于 [docsify](https://docsify.js.org/) 搭建，字体为 [霞鹜文楷](https://github.com/lxgw/LxgwWenKai)。
>
>
>
>- 📧 文章若有错误或描述不准确之处，请邮件联系 497976404@qq.com



​	

---

<!-- panels:start -->

<!-- div:title-panel -->

# 💿 Storage

<!-- div:left-panel -->

>   ## [📁 文件系统](posts/文件系统/Content.md)
>
>   Linux 文件系统相关知识如 VFS、XFS、etc.

> ## [🧱 块 I/O 子系统](posts/块IO子系统/Content.md)
>
> Linux 块 IO 子系统相关知识，像 Multi Disk、Device Mapper、Bcache 等模块都是工作在这一层

> ## [🗃️ 网络存储](posts/网络存储/Content.md)
>
> SAN / NAS / OSD 相关知识（整理中）

>   ## [💾 存储硬件](posts/存储硬件/Content.md)
>
>   SSD / HDD 等存储介质相关知识

>   ## [☔ 存储安全](posts/存储安全/Content.md)
>
>   RAID、EC、快照、etc.

<!-- div:right-panel -->

> ## [🚀 存储加速](posts/存储加速/Content.md)
>
> SPDK、DPDK 加速套件（挖坑未填）

>   ## [⚡ 高速缓存](posts/高速缓存/Content.md)
>
>   Cache 和内存模型相关知识

>   ## [🧮 数据结构与算法](posts/数据结构与算法/Content.md)
>
>   存储系统常用引擎、Linux 内核数据结构等

>   ## [🔧 常用技术](posts/常用技术/Content.md)
>
>   日常开发常用技术

>## [🌋 性能之巅](posts/性能之巅/Content.md)
>
>Linux 性能分析

<!-- panels:end -->



---

# 🛶 Raft专题

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

# 🎯 Open Source Project Learning

> ## [🌊 MIT 6.824](posts/MIT6.824/Content.md)
>
> 麻省理工分布式课程，这门课每节课都会精读一篇分布式系统领域的经典论文，并由此传授分布式系统设计与实现的重要原则和关键技术。同时其课程 Project 也是以其难度之大而闻名遐迩，4 个编程作业循序渐进带你实现一个基于 Raft 共识算法的 KV-store 框架，让你在痛苦的 debug 中体会并行与分布式带来的随机性和复杂性。

>   ## [☢️ MIT 6.S081](posts/MIT6.S081/Content.md)
>
>   麻省理工学院大名鼎鼎的 PDOS 实验室开设的面向 MIT 本科生的操作系统课程。前身是 MIT 著名的课程 6.828，MIT 的几位教授为了这门课曾专门开发了一个基于 x86 的教学用操作系统 JOS，被众多名校作为自己的操统课程实验。但随着 RISC-V 的横空出世，这几位教授又基于 RISC-V 开发了一个新的教学用操作系统 xv6，并开设了 MIT6.S081 这门课。由于 RISC-V 轻便易学的特点，学生不需要像此前 JOS 一样纠结于众多 x86 “特有的” 为了兼容而遗留下来的复杂机制，而可以专注于操作系统层面的开发。

>   ## [☁️ MinIO](posts/MinIO/Content.md)
>
>   MinIO offers high-performance, S3 compatible object storage. Native to Kubernetes, MinIO is the only object storage suite available on every public cloud, every Kubernetes distribution, the private cloud and the edge. MinIO is software-defined and is 100% open source under GNU AGPL v3.

> ## [🐍 DIY Git in Python](posts/u-git/Content.md)
>
> ugit 是一个类似 git 的版本控制系统的简单实现。

>   ## [⚓ etcd](posts/etcd/Content.md)
>
>   A distributed, reliable key-value store for the most critical data of a distributed system

>   ## [🔥 redis](posts/redis/Content.md)
>
>   The open source, in-memory data store used by millions of developers as a database, cache, streaming engine, and message broker.

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

[🌌 Software Engineering at Google](posts/Software-Engineering-at-Google/Content.md)

🖥️ [The Art of Command Line](https://github.com/jlevy/the-art-of-command-line)

