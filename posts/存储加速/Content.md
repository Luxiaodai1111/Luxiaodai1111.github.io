

>[!NOTE]
>
>随着 NVME SSD 的出现，存储设备的性能大幅度提升，软件协议栈开始成为性能瓶颈，Intel 推出 SPDK / DPDK 套件绕过内核来加速存储 / 网络性能。内核方面发现大家都想绕过它，也推出了像 io_uring 这样的技术来保住自己的地位。另外像 RDMA 看起来也很有趣。
>
>*挖坑待填，有机会再研究* 🙂

---

# 🚀 io_uring

Linux 最新的异步 IO 模型，其实这些技术说是加速，其实就是针对旧架构无法跟上新设备的特性，从而适配出的更现代化的架构，不存在孰优孰劣，只是大家都在自己的时代里发光发热

> ## 待研究



---

# 🚀 SPDK

官网：https://spdk.io/

The Storage Performance Development Kit (SPDK) provides a set of tools and libraries for writing high performance, scalable, user-mode storage applications.

> ## 待研究



---

# 🚀 DPDK

官网：https://www.dpdk.org/

DPDK is the Data Plane Development Kit that consists of libraries to accelerate packet processing workloads running on a wide variety of CPU architectures.

>   ## 待研究
