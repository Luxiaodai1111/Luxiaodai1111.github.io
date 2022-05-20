# 模块初始化

MD 模块加载时，它的初始化函数 md_init 将被执行。

```c
static int __init md_init(void)
{
	int ret = -ENOMEM;

	md_wq = alloc_workqueue("md", WQ_MEM_RECLAIM, 0);
	if (!md_wq)
		goto err_wq;

	md_misc_wq = alloc_workqueue("md_misc", 0, 0);
	if (!md_misc_wq)
		goto err_misc_wq;

	/* 注册块设备 */
	if ((ret = register_blkdev(MD_MAJOR, "md")) < 0)
		goto err_md;

	if ((ret = register_blkdev(0, "mdp")) < 0)
		goto err_mdp;
	mdp_major = ret;

	blk_register_region(MKDEV(MD_MAJOR, 0), 512, THIS_MODULE,
			    md_probe, NULL, NULL);
	blk_register_region(MKDEV(mdp_major, 0), 1UL<<MINORBITS, THIS_MODULE,
			    md_probe, NULL, NULL);

	/* 操作系统重启前，必须停止所有的 MD 设备 */
	register_reboot_notifier(&md_notifier);
	/* 注册 MD 模块的系统控制表 */
	raid_table_header = register_sysctl_table(raid_root_table);

	/* 创建 /proc/mdstat */
	md_geninit();
	return 0;

err_mdp:
	unregister_blkdev(MD_MAJOR, "md");
err_md:
	destroy_workqueue(md_misc_wq);
err_misc_wq:
	destroy_workqueue(md_wq);
err_wq:
	return ret;
}
```

register_blkdev 从名字上讲，它的作用是注册块设备。所有块设备模块初始化时，都应该调用 register_blkdev 函数。例如对于 SCSI 磁盘模块，对应的代码在其初始化函数中。

MD 设备支持两种块设备：一种是不可分区的，名字是 md，主设备号是 MD_MAJOR（9）；另一种是可分区的，名字是 mdp，主设备号是动态分配的。一个 mdp 设备最多能支持 64 个分区，包括零号分区。

register_blkdev 注册一个新的块设备。它有两个参数：第一个参数为主设备号 major，在调用时，有两种可能：

-   用户可以在 major 参数指定一个 1 到 255 的值，表示希望使用该主设备编号；
-   用户也可以将 major 设置为 0，这时系统将试图为其分配并返回一个未被使用的主设备编号。

第二个参数为块设备的名字，以 \0 作为字符串结尾，必须在系统内是唯一的。

返回值取决于输入参数 major：如果请求的主设备号在范围 1～255 内，则函数在成功时返回 0 或者返回负的错误码；如果传入 major 为 0，请求一个未使用的主设备号，则返回值为已分配的主设备号，在范围 1～255 之内，或者返回负的错误码。

系统中已注册的块设备名字和块设备编号的对应关系记录在全局变量 major_names 中。这是一个哈希链表数组，数组中共有 256 项，链表元素为 blk_major_name 结构，其结构中的域如下：

```c
static struct blk_major_name {
	struct blk_major_name *next;	// 链入 major_names 数组中对应链表的连接件
	int major;		// 主设备号
	char name[16];	// 设备名
} *major_names[BLKDEV_MAJOR_HASH_SIZE];
```

需要注意的是 register_blkdev 只是被用来为块设备驱动注册（获取）一个可用的块设备编号。

首先处理输入 major 参数为 0 的情况，这时首先在 major_names 循环找空的槽位。循环以索引从大到小的顺序进行，这样保证能尽快找到。

然后分配一个 blk_major_name 结构，初始化各个域包括：设置主设备号和块设备名，将 next 域初始化为 NULL，它是被用来链接到哈希表的。

接下来检查冲突。所谓冲突，指的是主设备号已经被使用，这出现在用户指定主设备号的情况下，对于系统分配主设备号，已经在前面确保了不会发生冲突。冲突检查即在主设备号对应的哈希链表中查找是否已经有该主设备号的项。如果有冲突，将释放已经分配的 blk_major_name 结构，返回 -EBUSY 到调用者。如果没有冲突，就把前面构造好的 blk_major_name 结构链入到链表的尾部，返回 0 或者系统分配的主设备号。

```c
int register_blkdev(unsigned int major, const char *name)
{
	struct blk_major_name **n, *p;
	int index, ret = 0;

	mutex_lock(&block_class_lock);

	/* 找到一个空闲的主设备号 */
	if (major == 0) {
		for (index = ARRAY_SIZE(major_names)-1; index > 0; index--) {
			if (major_names[index] == NULL)
				break;
		}

		if (index == 0) {
			printk("register_blkdev: failed to get major for %s\n",
			       name);
			ret = -EBUSY;
			goto out;
		}
		major = index;
		ret = major;
	}

	/* 分配 blk_major_name 并初始化 */
	p = kmalloc(sizeof(struct blk_major_name), GFP_KERNEL);
	if (p == NULL) {
		ret = -ENOMEM;
		goto out;
	}
	p->major = major;
	strlcpy(p->name, name, sizeof(p->name));
	p->next = NULL;

	/* 检查主设备号冲突 */
	index = major_to_index(major);
	for (n = &major_names[index]; *n; n = &(*n)->next) {
		if ((*n)->major == major)
			break;
	}
	if (!*n)
		*n = p;
	else
		ret = -EBUSY;

	if (ret < 0) {
		printk("register_blkdev: cannot get major %d for %s\n",
		       major, name);
		kfree(p);
	}
out:
	mutex_unlock(&block_class_lock);
	return ret;
}
```

在内核执行这段代码后，通过 `cat /proc/devices` 就可以看到新注册的块设备主编号。

回到 md_init 函数，两次调用 blk_register_region 分别将 MD 设备号和 MDP 设备号注册到块设备映射域，给定的 get 回调函数 md_probe。在打开 MD 设备号或 MDP 设备号时，该函数将被调用。下面即将介绍的打开 MD 设备就是基于这一个原理的。

显然，在操作系统重启前，必须停止所有的 MD 设备。在模块加载时，register_reboot_notifier 函数注册了一个 notifier_block 类型的节点。这个函数是 notifier_chain_register 的封装，将这个节点按照优先级添加到系统重启通知链表 reboot_notifier_list 上。

注册的节点结构 md_notifier 中，给出了节点优先级，以及一个回调函数。由于 MD 设备需要在底层磁盘之前停止，因此优先级数值设置得较大（为 INT_MAX）。回调函数 md_notify_reboot 仅仅是遍历所有的MD设备，停止之。

接下来，调用 register_sysctl_table 注册 MD 模块的系统控制表，主要用于设置用于控制同步速度的内核参数。

最后，调用 md_geninit 创建 /proc/mdstat，关于 MD 设备的状态都是通过这个 proc 项来输出的。



---



# MD 设备创建

## 用户空间打开设备

用户在 mdadm 命令行中给定 RAID 设备名。标准的设备名格式为 /dev/md# 或 /dev/mdp#（其中 # 代表一个数值）。mdadm 根据设备名 md 或 mdp 可以解析出主设备编号，以 # 作为次设备编号。之后调用 mknod 创建具有上述设备编号的块设备文件 /dev/md# 或 /dev/mdp#。

程序继续执行，调用 open 打开上述文件。在 do_open 中调用 get_disk，将调用和该设备编号对应的 bdev_map 项的 get 回调函数，即我们在这里注册的 md_probe。

之后，管理工具打开该设备名，这将调用到块设备文件操作表 def_blk_fops 中的 blkdev_open 函数，在那里调用了 blkdev_get --> get_gendisk，进入在 kobj_loopkup 中调用我们注册的回调函数 md_probe。

```c
static struct kobject *md_probe(dev_t dev, int *part, void *data)
{
	if (create_on_open)
		md_alloc(dev, NULL);
	return NULL;
}
```

此时，MD 设备只有虚拟的设备号范围，在内核中还没有构建对应的 MD 对象。md_probe 函数就是负责创建内核中 MD 对象，它直接调用 md_alloc 函数，它的主要流程如下：

1.  mddev_find 查找或创建一个 mddev 对象。它区分两种情况：如果在调用时指定设备号，则在链表 all_mddevs 中查找具有该设备号的 mddev 对象。如果找到，则递增引用计数后返回。否则，分配一个新的 mddev 对象，设置设备号，并将它初始化后，添加到 all_mddevs 链表，返回该对象。

2.  计算三个句柄变量：partitioned 为 1 表示 MD 设备支持分区；否则为 0。这是根据它的主设备号判断的。shift 反映了分区数目，而 unit 为 MD 设备的次设备号。

3.  如果在调用时指定了名字，则要确保该名字未冲突，在全局 all_mddevs 链表中逐个比较 MD 设备的磁盘名，若重复出现，则返回错误。

4.  分配一个请求队列，保存在 MD 设备描述符的 queue 域。将请求队列的私有数据域 queuedata 指向 MD 设备本身，这一点也很重要，块 I/O 子系统处理请求时需要从它找到 MD 设备描述符。调用 blk_queue_make_request 函数将请求队列的 make_request_fn 回调函数实例化为 md_make_request，这个函数我们在介绍 MD 设备请求执行时就会碰到。

5.  alloc_disk 分配通用磁盘结构，并进行必要的初始化：

    -   设置磁盘设备编号

    -   设置磁盘名。如果指定了设备名，使用之；否则为可分区设备使用 md_d#，其中 # 为 MD 设备次设备号的基本部分，例如假设 MD 设备可有 16（即 2<sup>4</sup>）个分区，则它为次设备号左移 4 位。为不可分区设备使用 md#，其中 # 为 MD 设备的次设备号；

    -   设置块设备操作表为 md_fops；

    -   ```c
        static const struct block_device_operations md_fops =
        {
        	.owner		= THIS_MODULE,
        	.open		= md_open,
        	.release	= md_release,
        	.ioctl		= md_ioctl,
        #ifdef CONFIG_COMPAT
        	.compat_ioctl	= md_compat_ioctl,
        #endif
        	.getgeo		= md_getgeo,
        	.media_changed  = md_media_changed,
        	.revalidate_disk= md_revalidate,
        };
        ```

    -   建立通用磁盘和 mddev 设备之间的联系，将 MD 设备地址保存在通用磁盘的 private_data 域，将通用磁盘的 queue 指向 MD 设备的 queue。

6.  add_disk 后我们这个 MD 设备就对系统可见了。这里需要指出，add_disk 函数调用 blk_register_region，结合在 md_init 函数中的调用，就出现了这样的情况：先是 mddev 在块设备映射域中注册，然后是 gendisk 在块设备映射域注册，后者范围更小，因此如果在映射域中查找块设备编号，将返回与它对应的 get 回调函数实现。这也就是如果 MD 设备被打开后，再次通过块设备编号访问获得的是 gendisk 而不是 mddev 的原因。

```c
static int md_alloc(dev_t dev, char *name)
{
	static DEFINE_MUTEX(disks_mutex);
	/*
	 * 查找或创建一个 mddev 对象。
	 * 它区分两种情况：如果在调用时指定设备号，则在链表 all_mddevs 中查找具有该设备号的 mddev 对象。
	 * 如果找到，则递增引用计数后返回。
	 * 否则，分配一个新的 mddev 对象，设置设备号，并将它初始化后，添加到 all_mddevs 链表，返回该对象。
	 */
	struct mddev *mddev = mddev_find(dev);
	struct gendisk *disk;
	int partitioned;
	int shift;
	int unit;
	int error;

	if (!mddev)
		return -ENODEV;

	partitioned = (MAJOR(mddev->unit) != MD_MAJOR);
	shift = partitioned ? MdpMinorShift : 0;
	unit = MINOR(mddev->unit) >> shift;

	/* wait for any previous instance of this device to be completely removed (mddev_delayed_delete). */
	flush_workqueue(md_misc_wq);

	mutex_lock(&disks_mutex);
	error = -EEXIST;
	/* 如果 MD 设备描述符的 gendisk 域不为 NULL，说明 MD 设备已存在 */
	if (mddev->gendisk)
		goto abort;

	if (name && !dev) {
		/* 确保名字没有重复 */
		struct mddev *mddev2;
		spin_lock(&all_mddevs_lock);

		/* 在全局 all_mddevs 链表中逐个比较 MD 设备的磁盘名 */
		list_for_each_entry(mddev2, &all_mddevs, all_mddevs)
			if (mddev2->gendisk &&
			    strcmp(mddev2->gendisk->disk_name, name) == 0) {
				spin_unlock(&all_mddevs_lock);
				goto abort;
			}
		spin_unlock(&all_mddevs_lock);
	}
	if (name && dev)
		/* Creating /dev/mdNNN via "newarray", so adjust hold_active. */
		mddev->hold_active = UNTIL_STOP;

	error = -ENOMEM;
	/* 分配 request_queue，后面设置为 md_make_request */
	mddev->queue = blk_alloc_queue(GFP_KERNEL);
	if (!mddev->queue)
		goto abort;
	mddev->queue->queuedata = mddev;

	blk_queue_make_request(mddev->queue, md_make_request);
	blk_set_stacking_limits(&mddev->queue->limits);

	/* 分配通用磁盘结构并进行必要的初始化 */
	disk = alloc_disk(1 << shift);
	if (!disk) {
		blk_cleanup_queue(mddev->queue);
		mddev->queue = NULL;
		goto abort;
	}
	disk->major = MAJOR(mddev->unit);
	disk->first_minor = unit << shift;
	if (name)
		strcpy(disk->disk_name, name);
	else if (partitioned)
		sprintf(disk->disk_name, "md_d%d", unit);
	else
		sprintf(disk->disk_name, "md%d", unit);
	/* 块设备方法表 */
	disk->fops = &md_fops;
	disk->private_data = mddev;
	disk->queue = mddev->queue;
	blk_queue_flush(mddev->queue, REQ_FLUSH | REQ_FUA);
	/* Allow extended partitions.  This makes the
	 * 'mdp' device redundant, but we can't really
	 * remove it now.
	 */
	disk->flags |= GENHD_FL_EXT_DEVT;
	mddev->gendisk = disk;
	/* As soon as we call add_disk(), another thread could get
	 * through to md_open, so make sure it doesn't get too far
	 */
	mutex_lock(&mddev->open_mutex);
	add_disk(disk);

	error = kobject_init_and_add(&mddev->kobj, &md_ktype,
				     &disk_to_dev(disk)->kobj, "%s", "md");
	if (error) {
		/* This isn't possible, but as kobject_init_and_add is marked
		 * __must_check, we must do something with the result
		 */
		pr_debug("md: cannot register %s/md - name in use\n",
			 disk->disk_name);
		error = 0;
	}
	if (mddev->kobj.sd &&
	    sysfs_create_group(&mddev->kobj, &md_bitmap_group))
		pr_debug("pointless warning\n");
	mutex_unlock(&mddev->open_mutex);
 abort:
	mutex_unlock(&disks_mutex);
	if (!error && mddev->kobj.sd) {
		kobject_uevent(&mddev->kobj, KOBJ_ADD);
		mddev->sysfs_state = sysfs_get_dirent_safe(mddev->kobj.sd, "array_state");
	}
	mddev_put(mddev);
	return error;
}
```

md_probe 之后系统调用 open 的实现代码继续执行，将调用块设备操作表的 open 回调函数。上面刚刚将块设备操作表设置为 md_fops，其 open 回调函数被实例化为 md_open，流程如下：

1.  首先调用 mddev_find 函数查找 mddev 对象，这个 MD 设备必定已经存在。
2.  确保 MD 设备和要打开的块设备相关联，即它们都关联到同一个通用磁盘描述符。
3.  确保这个通用磁盘描述符的 private_data 域又反过来指向 MD 设备。
4.  递增 MD 设备的打开计数。
5.  check_disk_change 函数检查是否可移除介质发生了改变，对于 MD 设备，则检查是否有 reshape 或 resize 等，如果有，则使得系统中所有 buffer-cache-entry 失效。

```c
static int md_open(struct block_device *bdev, fmode_t mode)
{
	struct mddev *mddev = mddev_find(bdev->bd_dev);
	int err;

	if (!mddev)
		return -ENODEV;

	/* 确保 MD 设备和要打开的块设备相关联 */
	if (mddev->gendisk != bdev->bd_disk) {
		mddev_put(mddev);
		flush_workqueue(md_misc_wq);
		return -ERESTARTSYS;
	}
	/* 确保这个通用磁盘描述符的 private_data 域又反过来指向 MD 设备 */
	BUG_ON(mddev != bdev->bd_disk->private_data);

	if ((err = mutex_lock_interruptible(&mddev->open_mutex)))
		goto out;

	if (test_bit(MD_CLOSING, &mddev->flags)) {
		mutex_unlock(&mddev->open_mutex);
		err = -ENODEV;
		goto out;
	}

	err = 0;
	/* 递增 MD 设备的打开计数 */
	atomic_inc(&mddev->openers);
	mutex_unlock(&mddev->open_mutex);

	check_disk_change(bdev);
 out:
	if (err)
		mddev_put(mddev);
	return err;
}
```



## ioctl 创建 MD

使用管理工具创建 MD 是通过 ioctl 来实现的。在打开 MD 设备文件（如 /dev/md0）情况下，文件操作表被设置为 def_blk_fops，其 unlocked_ioctl 回调函数的实现为 block_ioctl，而后者经调用 blkdev_ioctl，再调用 __blkdev_driver_ioctl，最后调用的是通用磁盘的块设备操作表中的 ioctl 回调函数。

前面看到，管理工具打开该设备文件时，将 MD 设备对应通用磁盘的块设备操作表设置为 md_fops，其中 ioctl 回调函数被实例化为 md_ioctl。这就说明，md_ioctl 是对 RAID 设备控制的入口，向内核和用户接口同时提供控制功能。实际上，mdadm 是通过该函数根据 MD 配置文件创建 RAID 的。

mdadm 有多种方式来构造一个 MD 设备，包括装配、创建等。我们这里跟踪一种最简单的方式，即 SET_ARRAY_INFO/ADD_NEW_DISK/RUN_ARRAY 的 ioctl 系列。其任务是找到构成一个阵列（按照它们的超级块）的设备集，将这一设备集提交给 MD 驱动。这包括提交一个没有参数的 SET_ARRAY_INFO ioctl——以准备阵列——然后提交多个 ADD_NEW_DISK ioctl 来将成员磁盘添加到阵列中，最后可以提交 RUN_ARRAY ioctl 以启动阵列。

```c
static int md_ioctl(struct block_device *bdev, fmode_t mode,
			unsigned int cmd, unsigned long arg)
{
	...

	switch (cmd) {
	case RAID_VERSION:
	case GET_ARRAY_INFO:
	case GET_DISK_INFO:
		break;
	default:
		if (!capable(CAP_SYS_ADMIN))
			return -EACCES;
	}

	/* Commands dealing with the RAID driver but not any particular array */
	...

	/* Commands creating/starting a new array */
	mddev = bdev->bd_disk->private_data;

	/* Some actions do not requires the mutex */
	...

	if (cmd == ADD_NEW_DISK)
		/* need to ensure md_delayed_delete() has completed */
		flush_workqueue(md_misc_wq);

	...
	err = mddev_lock(mddev);
	...

	if (cmd == SET_ARRAY_INFO) {
		mdu_array_info_t info;
		if (!arg)
			memset(&info, 0, sizeof(info));
		else if (copy_from_user(&info, argp, sizeof(info))) {
			err = -EFAULT;
			goto unlock;
		}
		if (mddev->pers) {
			err = update_array_info(mddev, &info);
			...
		}
		...
		err = set_array_info(mddev, &info);
		...
	}

	/* Commands querying/configuring an existing array */
	...

	/* Commands even a read-only array can execute */
	switch (cmd) {
	case ...:
		...
	case ADD_NEW_DISK:
		/* We can support ADD_NEW_DISK on read-only arrays
		 * only if we are re-adding a preexisting device.
		 * So require mddev->pers and MD_DISK_SYNC.
		 */
		if (mddev->pers) {
			mdu_disk_info_t info;
			if (copy_from_user(&info, argp, sizeof(info)))
				err = -EFAULT;
			else if (!(info.state & (1<<MD_DISK_SYNC)))
				/* Need to clear read-only for this */
				break;
			else
				err = add_new_disk(mddev, &info);
			goto unlock;
		}
		break;
	}

	/*
	 * The remaining ioctls are changing the state of the
	 * superblock, so we do not allow them on read-only arrays.
	 */
	...

	switch (cmd) {
	case ADD_NEW_DISK:
	{
		mdu_disk_info_t info;
		if (copy_from_user(&info, argp, sizeof(info)))
			err = -EFAULT;
		else
			err = add_new_disk(mddev, &info);
		goto unlock;
	}
	case RUN_ARRAY:
		err = do_md_run(mddev);
		goto unlock;
	case ...:
		...
	default:
		err = -EINVAL;
		goto unlock;
	}

unlock:
	if (mddev->hold_active == UNTIL_IOCTL &&
	    err != -EINVAL)
		mddev->hold_active = 0;
	mddev_unlock(mddev);
out:
	if(did_set_md_closing)
		clear_bit(MD_CLOSING, &mddev->flags);
	return err;
}
```

管理工具首先以 SET_ARRAY_INFO 为参数执行 ioctl，这将在内核中分配一个 MD 设备结构（mddev），并创建该结构到 /dev/md0 的映射，同时还会为 MD 设备结构分配 MD 超级块结构，并设置其中的阵列相关信息，包括 RAID 级别、磁盘数目、持续化标志和 chunk_size 等。

接着，管理工具将依次以 ADD_NEW_DISK 为参数执行 ioctl，这将填充 MD 设备的超级块结构中的成员磁盘描述符信息，并根据它同步各成员磁盘的 MD 超级块信息。

在所有磁盘添加成功之后，我们就可以运行该 MD 设备了。管理工具以 RUN_ARRAY 为参数调用 ioctl，这将调用对应的 MD 个性的 run 方法，设置该 MD 设备对应的块设备信息，并将 MD 设备的超级块信息写到所有的成员磁盘上。



### SET_ARRAY_INFO 

在 md_ioctl，处理 SET_ARRAY_INFO 控制码的代码首先从用户空间复制参数，然后根据 MD 设备是否已经存在分别调用 update_array_info 和 set_array_info 函数。我们这里只讨论后者。

```c
static int set_array_info(struct mddev *mddev, mdu_array_info_t *info)
{

	if (info->raid_disks == 0) {
		/* just setting version number for superblock loading */
		if (info->major_version < 0 ||
		    info->major_version >= ARRAY_SIZE(super_types) ||
		    super_types[info->major_version].name == NULL) {
			/* maybe try to auto-load a module? */
			pr_warn("md: superblock version %d not known\n",
				info->major_version);
			return -EINVAL;
		}
		mddev->major_version = info->major_version;
		mddev->minor_version = info->minor_version;
		mddev->patch_version = info->patch_version;
		mddev->persistent = !info->not_persistent;
		/* ensure mddev_put doesn't delete this now that there
		 * is some minimal configuration.
		 */
		mddev->ctime         = get_seconds();
		return 0;
	}
	mddev->major_version = MD_MAJOR_VERSION;
	mddev->minor_version = MD_MINOR_VERSION;
	mddev->patch_version = MD_PATCHLEVEL_VERSION;
	mddev->ctime         = get_seconds();

	mddev->level         = info->level;
	mddev->clevel[0]     = 0;
	mddev->dev_sectors   = 2 * (sector_t)info->size;
	mddev->raid_disks    = info->raid_disks;
	/* don't set md_minor, it is determined by which /dev/md* was openned */
	if (info->state & (1<<MD_SB_CLEAN))
		mddev->recovery_cp = MaxSector;
	else
		mddev->recovery_cp = 0;
	mddev->persistent    = ! info->not_persistent;
	mddev->external	     = 0;

	mddev->layout        = info->layout;
	mddev->chunk_sectors = info->chunk_size >> 9;

	if (mddev->persistent) {
		mddev->max_disks = MD_SB_DISKS;
		mddev->flags = 0;
		mddev->sb_flags = 0;
	}
	set_bit(MD_SB_CHANGE_DEVS, &mddev->sb_flags);

	mddev->bitmap_info.default_offset = MD_SB_BYTES >> 9;
	mddev->bitmap_info.default_space = 64*2 - (MD_SB_BYTES >> 9);
	mddev->bitmap_info.offset = 0;

	mddev->reshape_position = MaxSector;

	/* Generate a 128 bit UUID */
	get_random_bytes(mddev->uuid, 16);

	mddev->new_level = mddev->level;
	mddev->new_chunk_sectors = mddev->chunk_sectors;
	mddev->new_layout = mddev->layout;
	mddev->delta_disks = 0;
	mddev->reshape_backwards = 0;

	return 0;
}
```

set_array_info 在两种方式下使用：最初用在创建新的阵列时，这种情况下，raid_disks 总是大于 0，并且它和 level、size、not_persistent、layout、chunksize 确定了阵列的 shape。它总是创建超级块类型为 0.90.0 的阵列。最新的使用方式是组装阵列，这种情况下，raid_disks 总是 0，并且主版本号被用来确定要在设备上查找那种风格的超级块，次版本号和修正号也被保存下来，以便超级块处理句柄需要解释它们时使用。



### ADD_NEW_DISK 

在 md_ioctl，处理 ADD_NEW_DISK 控制码的代码首先从用户空间复制参数，然后调用 add_new_disk 函数，流程如下：

1.  首先处理的是组装阵列的情况。这种情况下，我们期望设备有一个有效的超级块，调用 md_import_device 函数导入成员磁盘，这个函数将读取超级块信息，并校验超级块的完好。如果一切顺利，还需要将这个成员磁盘的超级块和 MD 设备链表中的已有成员磁盘的超级块进行比较，这是由该超级块类型的 load_super 回调函数做的。如果没有发生冲突，最终调用 bind_rdev_to_array 函数将这个成员磁盘添加到 MD 设备中。
2.  处理热插入成员磁盘的情况，这种情况下最终需要调用到 MD 个性化结构中的 hot_add_disk 回调函数。
3.  处理添加创建阵列的情况

MD 设备的配置信息被称为 RAID 超级块，它分为两种：非持久性（non-persistent）RAID 超级块和持久性（persistent）超级块。非持久性 RAID 超级块只保存在内存中，而持久性超级块可以根据需要保存在其低层设备上，以便在系统重启之后能够读取这些信息，重建 MD 设备，这个过程称为持续化。

如果 RAID 超级块需要持续化，则它被保存在 RAID 集合中的每个成员磁盘的尾部。虽然当前的超级块的长度为 4 KB，但实际为它预留了一个完整的 MD_RESERVED_SECTORS 个扇区（相当于 64 KB）的空间。

因此，假设 x 是成员磁盘的的真实设备长度，那么 RAID 超级块在它尾部的保存位置可通过 MD_NEW_SIZE_SECTORS 宏如下计算，事实上，calc_dev_sboffset 函数调用的就是这个宏。

```c
#define MD_RESERVED_BYTES		(64 * 1024)
#define MD_RESERVED_SECTORS		(MD_RESERVED_BYTES / 512)
#define MD_NEW_SIZE_SECTORS(x)		((x & ~(MD_RESERVED_SECTORS - 1)) - MD_RESERVED_SECTORS)

static inline sector_t calc_dev_sboffset(struct md_rdev *rdev)
{
	sector_t num_sectors = i_size_read(rdev->bdev->bd_inode) / 512;
	return MD_NEW_SIZE_SECTORS(num_sectors);
}
```



### RUN_ARRAY 

在 md_ioctl，处理 RUN_ARRAY 控制码的代码直接调用 do_md_run 函数。

```c
static int do_md_run(struct mddev *mddev)
{
	int err;

	err = md_run(mddev);
	if (err)
		goto out;
	err = bitmap_load(mddev);
	if (err) {
		bitmap_destroy(mddev);
		goto out;
	}

	/* run start up tasks that require md_thread */
	md_start(mddev);

	/* 唤醒 MD 设备守护线程和 MD 设备同步线程看是否有工作可做 */
	md_wakeup_thread(mddev->thread);
	md_wakeup_thread(mddev->sync_thread); /* possibly kick off a reshape */

	set_capacity(mddev->gendisk, mddev->array_sectors);
	revalidate_disk(mddev->gendisk);
	mddev->changed = 1;
	/* 向用户空间发送KOBJ_CHANGE消息 */
	kobject_uevent(&disk_to_dev(mddev->gendisk)->kobj, KOBJ_CHANGE);
out:
	return err;
}
```

主要工作在于 md_run：

```c
int md_run(struct mddev *mddev)
{
	int err;
	struct md_rdev *rdev;
	struct md_personality *pers;

	/* MD设备中至少有一个成员磁盘 */
	if (list_empty(&mddev->disks))
		/* cannot run an array with no devices.. */
		return -EINVAL;

	/* 个性化指针为空 */
	if (mddev->pers)
		return -EBUSY;
	/* Cannot run until previous stop completes properly */
	if (mddev->sysfs_active)
		return -EBUSY;

	/*
	 * 和 MD 设备级别相对应的 MD 个性化模块可能还没有加载，尝试加载之
	 */
	if (!mddev->raid_disks) {
		if (!mddev->persistent)
			return -EINVAL;
		analyze_sbs(mddev);
	}
	if (mddev->level != LEVEL_NONE)
		request_module("md-level-%d", mddev->level);
	else if (mddev->clevel[0])
		request_module("md-%s", mddev->clevel);

	/*
	 * 冲刷所有成员磁盘在缓冲区中的数据，
	 * 从现在开始这个成员磁盘不能被单独使用，只能通过 MD 设备来使用。
	 * 在此过程中，还需要验证成员磁盘的数据区和元数据区没有发生重叠
	 */
	mddev->has_superblocks = false;
	rdev_for_each(rdev, mddev) {
		if (test_bit(Faulty, &rdev->flags))
			continue;
		sync_blockdev(rdev->bdev);
		invalidate_bdev(rdev->bdev);
		if (mddev->ro != 1 &&
		    (bdev_read_only(rdev->bdev) ||
		     bdev_read_only(rdev->meta_bdev))) {
			mddev->ro = 1;
			if (mddev->gendisk)
				set_disk_ro(mddev->gendisk, 1);
		}

		if (rdev->sb_page)
			mddev->has_superblocks = true;

		/* perform some consistency tests on the device.
		 * We don't want the data to overlap the metadata,
		 * Internal Bitmap issues have been handled elsewhere.
		 */
		if (rdev->meta_bdev) {
			/* Nothing to check */;
		} else if (rdev->data_offset < rdev->sb_start) {
			if (mddev->dev_sectors &&
			    rdev->data_offset + mddev->dev_sectors
			    > rdev->sb_start) {
				pr_warn("md: %s: data overlaps metadata\n",
					mdname(mddev));
				return -EINVAL;
			}
		} else {
			if (rdev->sb_start + rdev->sb_size/512
			    > rdev->data_offset) {
				pr_warn("md: %s: metadata overlaps data\n",
					mdname(mddev));
				return -EINVAL;
			}
		}
		/* 属性发生变化时主动通知用户空间 */
		sysfs_notify_dirent_safe(rdev->sysfs_state);
	}

	if (mddev->bio_set == NULL) {
		mddev->bio_set = bioset_create(BIO_POOL_SIZE, 0);
		if (!mddev->bio_set)
			return -ENOMEM;
	}
	if (mddev->sync_set == NULL) {
		mddev->sync_set = bioset_create(BIO_POOL_SIZE, 0);
		if (!mddev->sync_set) {
			err = -ENOMEM;
			goto abort;
		}
	}
	if (mddev->flush_pool == NULL) {
		mddev->flush_pool = mempool_create(NR_FLUSH_INFOS, flush_info_alloc,
						flush_info_free, mddev);
		if (!mddev->flush_pool) {
			err = -ENOMEM;
			goto abort;
		}
	}
	if (mddev->flush_bio_pool == NULL) {
		mddev->flush_bio_pool = mempool_create(NR_FLUSH_BIOS, flush_bio_alloc,
						flush_bio_free, mddev);
		if (!mddev->flush_bio_pool) {
			err = -ENOMEM;
			goto abort;
		}
	}

	/* 
	 * 在系统个性化链表（pers_list）中查找与该 MD 设备的级别相对应的 MD 个性化指针，
	 * 保存在 MD 设备描述符的 pers 域。
	 */
	spin_lock(&pers_lock);
	pers = find_pers(mddev->level, mddev->clevel);
	if (!pers || !try_module_get(pers->owner)) {
		spin_unlock(&pers_lock);
		if (mddev->level != LEVEL_NONE)
			pr_warn("md: personality for level %d is not loaded!\n",
				mddev->level);
		else
			pr_warn("md: personality for level %s is not loaded!\n",
				mddev->clevel);
		err = -EINVAL;
		goto abort;
	}
	spin_unlock(&pers_lock);
	if (mddev->level != pers->level) {
		mddev->level = pers->level;
		mddev->new_level = pers->level;
	}
	strlcpy(mddev->clevel, pers->name, sizeof(mddev->clevel));

	/* 如果 MD 设备有 reshape 的需求，那么个性化结构中必须定义有 start_reshape 回调函数 */
	if (mddev->reshape_position != MaxSector &&
	    pers->start_reshape == NULL) {
		/* This personality cannot handle reshaping... */
		module_put(pers->owner);
		err = -EINVAL;
		goto abort;
	}

	/* 
	 * RAID 设备的成员设备物理上要独立不相关，尤其对于支持冗余特性的 MD 设备。
	 * 也就是说，如果 MD 个性化结构定义了 sync_request 回调函数，
	 * 我们希望它的任何一个成员磁盘都不会和其他 MD 设备的成员磁盘是属于同一个物理磁盘，
	 * 即对应块设备描述符具有相同的 bd_contains 域。这是一种"愚蠢"的配置，
	 * 系统有义务打印警告信息。
	 */
	if (pers->sync_request) {
		/* Warn if this is a potentially silly
		 * configuration.
		 */
		char b[BDEVNAME_SIZE], b2[BDEVNAME_SIZE];
		struct md_rdev *rdev2;
		int warned = 0;

		rdev_for_each(rdev, mddev)
			rdev_for_each(rdev2, mddev) {
				if (rdev < rdev2 &&
				    rdev->bdev->bd_contains ==
				    rdev2->bdev->bd_contains) {
					pr_warn("%s: WARNING: %s appears to be on the same physical disk as %s.\n",
						mdname(mddev),
						bdevname(rdev->bdev,b),
						bdevname(rdev2->bdev,b2));
					warned = 1;
				}
			}

		if (warned)
			pr_warn("True protection against single-disk failure might be compromised.\n");
	}

	mddev->recovery = 0;
	/* may be over-ridden by personality */
	mddev->resync_max_sectors = mddev->dev_sectors;

	mddev->ok_start_degraded = start_dirty_degraded;

	if (start_readonly && mddev->ro == 0)
		mddev->ro = 2; /* read-only, but switch on first write */

	/* 调用 run 回调函数启动 MD 设备 */
	err = pers->run(mddev);
	if (err)
		pr_warn("md: pers->run() failed ...\n");
	/* 调用 size 回调函数验证阵列长度有效 */
	else if (pers->size(mddev, 0, 0) < mddev->array_sectors) {
		WARN_ONCE(!mddev->external_size,
			  "%s: default size too small, but 'external_size' not in effect?\n",
			  __func__);
		pr_warn("md: invalid array_size %llu > default size %llu\n",
			(unsigned long long)mddev->array_sectors / 2,
			(unsigned long long)pers->size(mddev, 0, 0) / 2);
		err = -EINVAL;
	}
	/* 如果 MD 设备支持冗余特性，调用 bitmap_create 创建 MD 设备位图 */
	if (err == 0 && pers->sync_request &&
	    (mddev->bitmap_info.file || mddev->bitmap_info.offset)) {
		err = bitmap_create(mddev);
		if (err)
			pr_warn("%s: failed to create bitmap (%d)\n",
			       mdname(mddev), err);
	}
	if (err) {
		mddev_detach(mddev);
		if (mddev->private)
			pers->free(mddev, mddev->private);
		mddev->private = NULL;
		module_put(pers->owner);
		bitmap_destroy(mddev);
		goto abort;
	}
	if (mddev->queue) {
		bool nonrot = true;

		rdev_for_each(rdev, mddev) {
			if (rdev->raid_disk >= 0 &&
			    !blk_queue_nonrot(bdev_get_queue(rdev->bdev))) {
				nonrot = false;
				break;
			}
		}
		if (mddev->degraded)
			nonrot = false;
		if (nonrot)
			queue_flag_set_unlocked(QUEUE_FLAG_NONROT, mddev->queue);
		else
			queue_flag_clear_unlocked(QUEUE_FLAG_NONROT, mddev->queue);
		mddev->queue->backing_dev_info.congested_data = mddev;
		mddev->queue->backing_dev_info.congested_fn = md_congested;
		blk_queue_merge_bvec(mddev->queue, md_mergeable_bvec);

		mddev->queue->can_split_bio = 1;
	}
	/* 如果 MD 设备支持冗余特性，在 sysfs 文件系统对应目录下创建冗余相关的属性文件 */
	if (pers->sync_request) {
		if (mddev->kobj.sd &&
		    sysfs_create_group(&mddev->kobj, &md_redundancy_group))
			pr_warn("md: cannot register extra attributes for %s\n",
				mdname(mddev));
		mddev->sysfs_action = sysfs_get_dirent_safe(mddev->kobj.sd, "sync_action");
	} else if (mddev->ro == 2) /* auto-readonly not meaningful */
		mddev->ro = 0;

	atomic_set(&mddev->max_corr_read_errors,
		   MD_DEFAULT_MAX_CORRECTED_READ_ERRORS);
	mddev->safemode = 0;
	mddev->safemode_delay = (200 * HZ)/1000 +1; /* 200 msec delay */
	mddev->in_sync = 1;
	smp_wmb();
	spin_lock(&mddev->lock);
	mddev->pers = pers;
	spin_unlock(&mddev->lock);
	rdev_for_each(rdev, mddev)
		if (rdev->raid_disk >= 0)
			if (sysfs_link_rdev(mddev, rdev))
				/* failure here is OK */;

	if (mddev->degraded && !mddev->ro)
		/* This ensures that recovering status is reported immediately
		 * via sysfs - until a lack of spares is confirmed.
		 */
		set_bit(MD_RECOVERY_RECOVER, &mddev->recovery);
	set_bit(MD_RECOVERY_NEEDED, &mddev->recovery);

	/* 将超级块信息更新到所有成员磁盘上 */
	if (mddev->sb_flags)
		md_update_sb(mddev, 0);

	md_new_event(mddev);
	sysfs_notify_dirent_safe(mddev->sysfs_state);
	sysfs_notify_dirent_safe(mddev->sysfs_action);
	sysfs_notify(&mddev->kobj, NULL, "degraded");
	return 0;

abort:
	if (mddev->flush_bio_pool) {
		mempool_destroy(mddev->flush_bio_pool);
		mddev->flush_bio_pool = NULL;
	}
	if (mddev->flush_pool) {
		mempool_destroy(mddev->flush_pool);
		mddev->flush_pool = NULL;
	}
	if (mddev->bio_set) {
		bioset_free(mddev->bio_set);
		mddev->bio_set = NULL;
	}
	if (mddev->sync_set) {
		bioset_free(mddev->sync_set);
		mddev->sync_set = NULL;
	}

	return err;
}
```



---

# 自动检测和运行 RAID

在发现磁盘设备时，Linux 内核将调用 rescan_partitions 函数扫描磁盘上的所有分区。如果是 Linux RAID 分区并且配置了 CONFIG_BLK_DEV_MD 就会调用 md_autodetect_dev 函数，它负责把 Linux RAID 分区的设备号记录在 all_detected_devices 链表中。

```c
#ifdef CONFIG_BLK_DEV_MD
		if (state->parts[p].flags & ADDPART_FLAG_RAID)
			md_autodetect_dev(part_to_dev(part)->devt);
#endif

void md_autodetect_dev(dev_t dev)
{
	struct detected_devices_node *node_detected_dev;

	node_detected_dev = kzalloc(sizeof(*node_detected_dev), GFP_KERNEL);
	if (node_detected_dev) {
		node_detected_dev->dev = dev;
		mutex_lock(&detected_devices_mutex);
		list_add_tail(&node_detected_dev->list, &all_detected_devices);
		mutex_unlock(&detected_devices_mutex);
	}
}
```

此后，系统继续初始化，prepare_namespace 函数（文件init/do_mounts.c）会得到执行。它会等待所有的设备探测都已完成，然后调用 md_run_setup 函数。后者进而调用 autodetect_raid 函数。

```c
void __init prepare_namespace(void)
{
	...
	wait_for_device_probe();

	md_run_setup();

	...
}

void __init md_run_setup(void)
{
	create_dev("/dev/md0", MKDEV(MD_MAJOR, 0));

	if (raid_noautodetect)
		printk(KERN_INFO "md: Skipping autodetection of RAID arrays. (raid=autodetect will force)\n");
	else
		autodetect_raid();
	md_setup_drive();
}

```

在 autodetect_raid 函数中，会通过 open 系统调用打开 /dev/md0，并以 RAID_AUTORUN 为控制码发送 ioctl。

```c
static void __init autodetect_raid(void)
{
	int fd;

	/*
	 * Since we don't want to detect and use half a raid array, we need to
	 * wait for the known devices to complete their probing
	 */
	printk(KERN_INFO "md: Waiting for all devices to be available before autodetect\n");
	printk(KERN_INFO "md: If you don't use raid, use raid=noautodetect\n");

	wait_for_device_probe();

	fd = sys_open("/dev/md0", 0, 0);
	if (fd >= 0) {
		sys_ioctl(fd, RAID_AUTORUN, raid_autopart);
		sys_close(fd);
	}
}
```

Linux MD 实现代码处理 RAID_AUTORUN 的函数是 autostart_arrays。它根据前面记录在 all_detected_devices 中的成员磁盘设备号链表分析超级块、组装并启动 MD 设备。

```c
#ifndef MODULE
	case RAID_AUTORUN:
		err = 0;
		autostart_arrays(arg);
		goto out;
#endif
```

​	
