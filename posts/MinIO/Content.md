# ☁️ MinIO

官网：https://min.io/

MinIO offers high-performance, S3 compatible object storage. Native to Kubernetes, MinIO is the only object storage suite available on every public cloud, every Kubernetes distribution, the private cloud and the edge. MinIO is software-defined and is 100% open source under GNU AGPL v3.

> ## [🏷️ MinIO 简介](posts/MinIO/简介/MinIO简介.md)
>
> MinIO 简单介绍

>   ## [🏷️ MinIO 部署和扩容](posts/MinIO/简介/MinIO部署和扩容.md)
>
>   分布式部署和扩容

>   ## [🏷️ AWS S3 简介](posts/MinIO/简介/AWS-S3-简介.md)
>
>   Amazon Simple Storage Service (Amazon S3) 是一种对象存储服务，提供行业领先的可扩展性、数据可用性、安全性和性能

>   ## [🏷️ MinIO 分布式锁 dsync 简介](posts/MinIO/简介/dsync.md)
>
>   dsync 是 MinIO 为自己设计的一套简单的分布式锁系统

---

# 源码分析

基于版本 RELEASE.2022-04-01T03-41-39Z 分析主要工作流程

>   ## [🏷️ MinIO 服务启动流程分析](posts/MinIO/源码分析/MinIO服务启动流程分析.md)
>
>   minio server 启动流程分析

>   ## [🏷️ 纠删池初始化分析](posts/MinIO/源码分析/erasureServerPools初始化.md)
>
>   ec pool 初始化流程分析

>   ## [🏷️ 对象读写基础流程分析](posts/MinIO/源码分析/对象读写基础流程分析.md)
>
>   EC 模式对象读写基础流程分析

>## [🏷️ 对象多版本、lock、retention、legal hold功能实现分析](posts/MinIO/源码分析/对象多版本实现分析.md)
>
>对象多版本、lock、retention、legal hold功能实现分析

>   ## [🏷️ 故障恢复](posts/MinIO/源码分析/故障恢复.md)
>
>   EC 模式故障恢复

>   ## [🏷️ 布隆过滤器实现](posts/MinIO/源码分析/MinIO布隆过滤器实现.md)
>
>   MinIO 布隆过滤器实现

