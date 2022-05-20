# 🧱 块 I/O 子系统

## blk-sq

块 I/O 子系统单请求队列源码（Linux 3.10）分析

> ### [🏷️ 块 IO 子系统概述](posts/块IO子系统/blk-sq/概述.md)
>
> Linux 内核中负责提交对块设备 IO 请求的子系统被称为块 IO 子系统
>
> ### [🏷️ 相关结构体](posts/块IO子系统/blk-sq/相关结构体.md)
>
> Linux 块 IO 子系统的主要结构体
>
> ### [🏷️ 添加 SCSI 磁盘到系统](posts/块IO子系统/blk-sq/添加磁盘.md)
>
> 在低层驱动发现一个设备，希望将它作为磁盘来使用，或是块设备驱动要生成一个逻辑设备作为磁盘来使用，都需要分配一个通用磁盘描述符，并将它添加到系统。
>
> ### [🏷️ 请求处理流程](posts/块IO子系统/blk-sq/请求处理.md)
>
> 主要包括请求提交，构造 request，蓄流／泄流等流程
>
> ### [🏷️ IO 调度](posts/块IO子系统/blk-sq/IO调度.md)
>
> IO 调度框架以及 deadline 算法分析

​	

---

## multi disk

Linux RAID 模块架构源码（Linux 3.10）分析

>   ### [🏷️ MD 相关结构体](posts/块IO子系统/multi-disk/1-MD相关结构体.md)
>
>   Linux Multi Disk 模块主要结构体

>   ### [🏷️ MD 模块初始化](posts/块IO子系统/multi-disk/2-MD模块初始化.md)
>
>   Multi Disk 模块初始化

>   ### [🏷️ MD 请求执行](posts/块IO子系统/multi-disk/3-MD请求执行.md)
>
>   以线性 RAID 举例说明 Multi Disk 请求执行流程





