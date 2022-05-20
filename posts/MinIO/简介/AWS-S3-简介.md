本文转自：[一文读懂 AWS S3](http://www.thinkingincrowd.me/2020/03/10/aws-s3/) 和 [一文读懂 AWS IAM](http://www.thinkingincrowd.me/2020/02/16/aws-iam/)



AWS S3 全名是 Simple Storage Service，简便的存储服务。为什么这么起名啊？它：

1. 提供了统一的接口 REST/SOAP 来统一访问任何数据
2. 对 S3 来说，存在里面的数据就是对象名（键），和数据（值）
3. 不限量，单个文件最高可达 5TB
4. 高速。每个 bucket 下每秒可达 3500 PUT/COPY/POST/DELETE 或 5500 GET/HEAD 请求
5. 具备版本，权限控制能力
6. 具备数据生命周期管理能力



---

# 基本概念

## Bucket

要存储数据在 S3 里，首先我们要建立一个 Bucket。Bucket 默认是不公开的。

Bucket 有几个特点：

- 命名需全球唯一。每个帐号默认可建 100 个，可申请至最多 1000 个
- 创建者的拥有权不可转让，也不可以从一个 Region 转去别的 Region
- 没有对象存储数量限制

Bucket 就像是电脑里面的某一个顶层分区。所有的对象都必须保存在某一个 bucket 下面。



## Object

Bucket 里面每一个存储的数据就是对象，由对象名（键），和数据（值）组成。

对象的键（Key）可以很长，甚至按照一定前缀格式来指定，从而模拟文件夹的层级结构，比如 `Photo/Family/2020-01-25-new-year/altogether.jpg`。

每一个对象其实还包含一些元信息（Meta-data），包括系统指定的文件类型，创建时间，加密算法等，和用户上传时指定的元信息。元信息在对象创建后都无法更改。

我们也可以为对象指定最多 10 个标签（Tag），标签的键和值的最大长度是 128 和 256 个字符。这个标签和元信息有什么不同呢？标签是可以修改和新增的。它最大的好处，是可以结合权限控制，生命周期管理，和数据分析等使用。

单个文件上传最大是 5GB。超过的话，需要使用 multipart upload API。最大支持 5TB。



## 一致性特性

对程序员来说，这么一个类似数据库的东西，肯定需要关心它的读写特性和一致性模型。

- 没有锁的功能。如果同时（几乎）发起两个更新对象的 PUT 请求，键相同，那么，以到达 S3 时间先后处理更新。
- 不同对象的更新，没法做到原子操作。
- 对全新的对象来说，它是 Read-after-Write Consistency 的。也就是写了之后马上读，肯定就是你刚才上传的数据。
- 如果你要更新数据，那就变成 Eventual Consistency 了。也就是说，更新后马上读，可能是旧的数据，也可能是新的。

这里有一个比较坑的地方是，如果你先调用 GET 请求访问一个不存在的资源，S3 告诉你它不存在。然后你马上上传数据，再调用一个 GET，这时候是有可能拿不回来的。



---

# 存储级别

作为一个“云盘”，S3 的好处是可以把你存储的数据，按不同的存储级别来计费。这个存储级别是每个对象不同，上传时指定的。

我们看看不同的场景，应该选择哪种级别的存储：

- 经常访问的数据对象
  - STANDARD - 这是最普通，最常用的类型
  - REDUCED_REDUNDANCY (RRS) - 不建议使用。仅为不重要，可再建数据设计，还有每年平均 0.01% 数据丢失的可能性。

- 按访问频率自动优化的数据
  - INTELLIGENT_TIERING - 可以智能地把不热门的数据自动转级别。但是，每个文件最低收费标准是 128KB，存 30天。

- 不经常访问的数据
  - STANDARD_IA
  - ONEZONE_IA

AWS 一个 Region，有两到三个 Zone。这两种级别的区别就是，One Zone 的数据就单点保存，丢了就丢了。

- 归档的数据
  - S3 Glacier - 最低保存 90天。取出时间 1分钟至 12小时。
  - S3 Glacier Deep Archive - 最低保存 180天。默认 12小时内取出。

S3 计费的大头主要是存储容量。但是，S3 还会按照数据获取的次数，和传输容量来计费。越不常访问的级别，虽然存储便宜，但是访问贵。INTELLIGENT_TIERING 还会收监测和管理费用。



## 生命周期的管理

除了手动指定，或者使用 INTELLIGENT_TIERING 外，S3 其实还可以让我们在 bucket 上定义生命周期管理的策略（Policy），来自动转换对象的存储级别。

生命周期的管理可以做到： 

1.   转换存储级别 
2.   过期删除



---

# 数据安全

数据安全，是数据存储服务非常重要的一部分。S3 提供了很多方面的功能来保障这一点。

## 多版本

不小心把数据删除了的痛，程序员应该都懂。但是，后悔药是没有的。所以，我们很多时候并不会做永久删除，而是实现软删除的功能。S3 就提供了多版本的功能。只要 bucket 打开了多版本的选项，每次对象的更新都会新加一个版本，而不是覆盖。删除对象，也只是新增一个删除标识。

当然，你要强行删除特定版本的数据也是可以的，它只是让这件事变得难一些而已。它甚至可以把 bucket 设置成只有通过 MFA 认证的请求才能实现永久删除。

要注意的是： 

1.   打开版本控制的 bucket，是没法关闭的，顶多可以暂停。也就是说，暂停后的 bucket，新加对象的时候，版本 id 会设为 null。
2.   无论打开，或者暂停版本控制，对 bucket 内已经存在的对象是没有影响的。



## 锁定

除了使用多版本控制让覆盖或者删除变得更难，S3 还可以锁定特定版本的对象。这种模型被称为 write-once-read-many (WORM)。

有两种锁定的方式： 

- 设定保留期限 - 在某固定期限内，对象 WORM。
- 法定留存 - 仅当这个留存标识被删除后，对象才能被覆盖或删除。

一个特定版本的对象，可以同时设置这两种保护，或任意一种。

因为锁定是针对特定版本的对象，如果你的更改或删除操作请求只根据对象的键，那它还是允许你新增一个版本，或加上删除标识。只是这个锁定，还能防止对象因为生命周期的设置而被删除掉。



## 服务端加密

数据传输过程（in-transit）中的保护，现在基本都由 SSL/TLS 来实现的。AWS 也提供 VPN 或者网络直连服务。

S3 提供了服务端数据加密的功能，可实现数据的存储（at rest）方面的安全。不过它只支持对称加密，不支持非对称加密。虽然你可以本地把数据加密了再上传到 S3，但是，这需要自己保护好密钥，其实更不容易。

服务端加密开启后，bucket 内已经存在的对象不会被自动加密。而且，只有数据被加密，元信息（meta data），标签（Tag）不会被加密。

S3 的服务端加密有三种方式：

1. SSE-S3 - S3 自管理的密钥，使用 AES-256 加密算法。每个对象的密钥不同，并且它还被定期更换的主密钥同时加密。
2. SSE-KMS - 密钥存放在 KMS（软硬件结合的密钥管理系统）。
3. SSE-C - 在请求时自己提供密钥，S3 只管加解密逻辑和存储。S3 不保存密钥，只保存随机加盐的 HMAC 值来验证今后请求的合法性。

这里主要说一下 S3 使用 SSE-KMS 特点：

- 启用前，如果没有指定客户管理的 CMK（customer master key），S3 会自动创建一个由 AWS 管理的 CMK
- 加密数据的密钥，同时也被加密，并和数据保存在一起
- 有请求频率限制
- 只支持对称密钥
- CMK 必须和 bucket 在同一个区（Region）



## IAM 集成

### Concept

IAM 是 AWS 云平台中负责身份认证，和权限控制的服务。AWS 云虽然分了很多个区（Region），但 IAM 是 Global，全局的。所以，它的数据和配置的更改，也是 Eventually Consistent 的。



### Best Practices

在讲 IAM 的权限控制是怎么工作之前，先强调两个最重要的安全理念。

**Grant Least Privilege**：在 AWS 里面，每一个用户默认都是没有任何权限的。他甚至不能查看自己的密码或 access key，丢失了也只能重新生成。

**Lock Away Your AWS Account Root User**：AWS 账户开通的时候，你的登录邮箱和密码，就成为了这个账户下的超级管理员，它默认是什么都可以干的。所以，和在 Linux 下不要滥用 root 一样，不要用这个超级帐号做日常操作，而是创建一个有 Full Administrator 权限的用户。



### How It Works?

权限控制有两个基本概念：

1. **Authentication** - 确认是否为有效用户，是否允许登录/接入
2. **Authorization** - 确认用户当前请求的操作（读写资源），是否合法

所以，IAM 最重要就是管理 Identity，和控制 Resource 的操作。

#### Identity/Principal

从资源访问的角度来看，使用 AWS 资源的其实不单单是具体的人，还可能是 Application。所以，AWS 里面的身份，分几种：

- User
- Application
- Federated User
- Role

能在 AWS IAM 控制台里创建的，只有 User 和 Role。而 User 在创建的时候，可以指定它的访问类型。是凭借用户名密码在 Console 登录，还是使用 Access Key ID 及 Secret 通过 API 来访问，还是两者皆可。

要特别注意的是，User 是直接操作 AWS 资源的用户，而不是你自己开发并部署在 AWS 的系统里面的用户。IAM 的 User 是有数量限制的，最多 5000 个。

如果你开发的系统需要操作 AWS 资源，比如说上传文件到 S3，那你需要用的是 Federated User。通过 OpenID Connect（如 Google/Facebook）或者 SAML 2.0（如 Microsoft AD），你的系统用户可以在登录后换取代表某个 AWS Role 的临时 token 来访问 AWS 资源。

#### Authentication

访问和使用 AWS 资源有两种方式，一种是通过页面登录，也就是 Console。一种是通过 AWS API，也就是接口，包括 CLI, SDK 或 HTTPS 请求。

IAM User 在 Console 页面登录需要提供 AWS 帐号名，IAM User 名和密码。AWS 帐号名是 AWS 云服务开通时，系统生成的一串数字，或者是你赋予的别名。它其实就是一个多租户系统里面的租户帐号。AWS 还会为每个帐号提供一个独特的登录链接。

而如果是使用 API 访问 AWS，我们是需要用 IAM User 的 Access Key ID 及 Secret 来为这个 HTTP 请求生成签名的。为请求签名，是大多数的 API 集成的一种安全性考量。微信，支付宝等平台都这么做。为什么呢？

1. 确认请求发起方是合法的，就是确保你就是你。
2. 保护数据传输过程的安全，就是确保数据没被篡改。
3. 防止重放攻击，就是确保一个请求不被多次使用，滥用或者冒用。

签名需要根据什么信息生成呢？可以说是包含了请求唯一性的所有信息：

请求的接口版本号 请求的操作是什么（Action） 请求的日期 所有请求的参数等

AWS 的请求样例：

```bash
https://iam.amazonaws.com/?Action=AddUserToGroup
&GroupName=Managers
&UserName=Bob
&Version=2010-05-08
&AUTHPARAMS
```

其实，如果你是使用 AWS SDK 或者 CLI，它会根据你配置的 Access Key 自动签名。只有当你自己发起一个 HTTP 请求的时候，才需要自己实现签名的逻辑。

#### Authorization

所谓是否有足够的权限，就是验证以下三者在一个请求的场景下，是否被允许：

1. 主体（Identity）
2. 操作（Action）
3. 资源（Resource）

AWS 是通过策略（Policy）来定义权限（Permissions）的。最基本的策略有两大类。一种是 Identity-based policy，另一种是 Resource-based policy。前一种挂在 User/Role/Group 上面，用以定义这些被挂载的主体，能对什么资源进行怎样的操作。而后一种直接挂载在 AWS 资源上面，用以定义哪些主体可以对这个资源做什么样的操作。

AWS Policy 的 Permissions 定义，在内部是通过一个 JSON 格式来表示的。我们来看一个样例：

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "ListAndDescribe",
      "Effect": "Allow",
      "Action": [
        "dynamodb:List*",
        "dynamodb:Describe*"
      ],
      "Resource": "*"
    },
    {
      "Sid": "SpecificTable",
      "Effect": "Allow",
      "Action": [
        "dynamodb:BatchGet*",
        "dynamodb:Get*",
        "dynamodb:Query",
        "dynamodb:Scan",
        "dynamodb:BatchWrite*",
        "dynamodb:Delete*",
        "dynamodb:Update*",
        "dynamodb:PutItem"
      ],
      "Resource": "arn:aws:dynamodb:*:*:table/MyTable"
    },
    {
      "Sid": "AllowAllActionsForEC2",
      "Effect": "Allow",
      "Action": "ec2:*",
      "Resource": "*"
    },
    {
      "Sid": "DenyStopAndTerminateWhenMFAIsNotPresent",
      "Effect": "Deny",
      "Action": [
        "ec2:StopInstances",
        "ec2:TerminateInstances"
      ],
      "Resource": "*",
      "Condition": {
        "BoolIfExists": {
          "aws:MultiFactorAuthPresent": false
        }
      }
    }
  ]
}
```

这个策略控制了 DynamoDB 和 EC2 的访问权限。它看起来很复杂，但其实结构很清晰。这里面最主要的元素就是 `Effect`, `Action`, 和 `Resource`。它们确定了什么资源上的哪些操作，是被允许，还是禁止的。它们是 AND 的逻辑组合。

Statement 里前两个 Permission，允许用户获取 DynamoDB 里面的资源信息，但是只有 MyTable 这个表能做写操作。而后两部分允许用户对 EC2 做任何操作，但是停止和结束 Instance 则必须通过了 MFA 登录认证后才可以。

#### Policy Evaluation Logic

一个用户或者角色主体上，可以拥有多个不同的 Policy。所以，Policy 的权限验证逻辑，可谓相当复杂。在讲验证流程前，我再重复一次 AWS 权限的设计原则，这对流程的理解很重要。

- 如果**有显式的 Deny，就禁止**。
- **Grant Least Privilege** 原则。如果没有显式赋予权限，也就是没有任何 Policy 为请求的资源和操作定义了 `Allow` 权限，那这个主体就没有权限（Implicit Deny）。

AWS 对收到的操作请求，会根据以下的流程来判断这个请求的主体是否有操作权限：

1. Deny evaluation
2. AWS Organizations service control policies (SCP)
3. Resource-based policies
4. IAM permissions boundaries
5. Session policies
6. Identity-based policies

第一步，首先把 2 至 6 里面的所有 policy 的显式 Deny 拿出来。如果当前的请求属于 Deny 的范围，直接禁止操作。这个就是第一个原则。

第二步到第六步，是具体的 policy。如果该主体有这个类型的 policy 存在，就按照第二个原则处理。如果没有，跳到下一个 policy 类别的检查。

那么多种的 Policy 类别，为什么是这个排列顺序呢？我是这么理解的：

1. Organization SCP 作为组织级别策略，优先级最高。
2. Resource-based policy 可以跨帐号赋予权限，级别比后面的高一些。
3. Permission Boundary 的作用是提前为用户定义一个最大的权限范围，避免意外打开了权限的情况，所以比后面的级别要高。
4. Session policies 是会话级别，允许临时赋予权限，所以比 Identity-based policies 高。
5. Identity-based policies 是最稳定的，所以检查放在最后。

不过，这里有一个特例，就是 Resource-based policy。如果它是 Implicit Deny 的情况，还是会继续后面的检查，不会阻止。还有一个复杂的情况是关于 Session policy 的，这个就不在本文解释了，具体可看[文档](https://link.zhihu.com/?target=https%3A//docs.aws.amazon.com/IAM/latest/UserGuide/access_policies.html%23policies_session)。

其实，即便逻辑复杂，判断是否有权限还是可以简单地总结为两条：

> **只有具备显式的 Allow，并且没有显式的 Deny，才有权限。**
>
> 或者
>
> **如果没有显式的 Allow，或者有显式的 Deny，就没有权限。**



上面详细介绍了 AWS IAM 的功能，S3 当然能根据 IAM 的设置来控制权限。S3 的资源，除了 bucket 和 object 外，还包含了一些子资源。

Bucket 子资源: lifecycle website versioning policy cors logging

Object 子资源: acl restore

在了解 S3 如何控制权限以前，我们要理解资源的拥有者这个概念。在 S3 里面，资源是谁创建的，它所属的 AWS 帐号，就是这个资源的拥有者。有一种情况是，Bucket 是帐号 A 创建的。但是 A 允许 B 在里面创建对象 X。这个 X 的拥有者是 B 而不是 A。如果资源拥有者授权 A，A 可以把自己的权限委托给自己帐号内的其它人，但不可以再一次跨帐号授权。



## S3 如何验证请求

当 S3 收到请求时，会经过下面几个步骤验证请求：

1. 把所有相关的策略（user policy, bucket policy, ACL）集合起来。
2. 根据下面 3 小步，拿出全集中的合适子集来分别验证：
   1. 用户范畴 如果请求发起者是 IAM User 或 Role，它所属的 aws 帐号就会先检查它是否有权限做这种类型的操作（user policy）。假如刚好要操作的资源（bucket 或 object）属于当前帐号，那么就同时检查相应的 bucket policy, bucket ACL 和 object ACL。如果请求发起者不属于 IAM，则跳至下一步。
   2. Bucket 范畴 S3 会检查拥有 bucket 的 aws 帐号的策略。如果操作的是 bucket，那请求的用户需要有 bucket owner 赋予的权限。如果操作的是对象，需要检查 bucket owner 是否有显式 deny 对象的设置。
   3. Object 范畴 当请求是关于对象的时，最后检查对象 owner 的策略子集。

天啊，这看上去好复杂。其实，**和一个小孩想玩玩具一样**：

首先，小孩必须获得父母的请求，可以玩玩具。然后，看这个玩具拥有者是谁，如果是自己父母，就看这个玩具是否能给孩子玩（比如可能年龄还不合适，超时等）。如果这个玩具是其它人的，那就要还获得其它人的允许。



## 不同策略的场景

对于 S3 验证请求的时候，需要验证的那几种不同的策略，究竟各自的使用场景是什么呢？

- Object ACL
  - 唯一一种管理保存在他人 bucket 里的对象权限的方式
  - 定义在单个对象级别
  - 最多包含 100 个授权信息

- Bucket ACL
  - 唯一推荐使用的场景是为 S3 Log Delivery 赋予写访问日志的权限
  - 虽然可以配置跨帐号权限，但仅仅支持有限的设置

- Bucket Policy
  - 能给自己帐号内的用户赋权
  - 支持所有 S3 操作的跨帐号权限设置
  - Policy 自身大小不超过 20KB

- User Policy
  - 能给自己帐号内的用户赋权



---

# 副本备份

S3 不仅通过多点存储提高健壮性，还提供了自动的异步数据备份的功能。不仅支持同 Region，不同 bucket 的备份，还支持跨 Region，不同帐号的备份。要开启副本备份，首先必须在源和目标 bucket 同时打开多版本的设置。

## 为什么要使用？

- 备份同时保留元数据
- 备份至不同存储级别
- 更改备份数据的拥有权
- 15 分钟内自动备份

## 什么时候跨区备份（CRR）

- 满足监管需求
- 减少数据传输延时（地域原因）
- 提高数据操作的效率

## 什么时候同区备份（SRR）

- 合并日志
- 生产和测试用户间数据同步
- 满足数据主权法规

## 什么会同步？

- 备份配置生效后新建的对象
- 没加密的对象
- 通过 SSE-S3 或者 SSE-KMS CMK（必须显式启用）加密的对象
- 对象元数据
- bucket 拥有者有权读取的对象
- 对象 ACL 除非备份同属一个 aws 帐号
- 对象标签
- 对象的锁信息

## 什么不同步？

- 备份配置生效前新建的对象
- 使用 SSE-C 加密的对象
- 保存在 Glacier 或 Glacier Deep Archive 的对象
- bucket 级别子资源的更新
- 由于生命周期配置导致的操作
- 源 bucket 中本来就是副本的对象
- 删除标识
- 源 bucket 中被删除的特定版本的对象



---

# 知识小点与周边

## 路由请求

- S3 使用的是 DNS 来接收转发请求。如果请求对象的 S3 地址不对，会返回一个临时的重定向。但是对那些 2019 年 3 月 20 日后启用的 Region，地址错误返回的则是 HTTP 400 状态。
- S3 DNS 会按需更新 IP 地址。所以，对那些长期运行的客户端，可能需要采取特殊手段来更新 IP 信息。



## 静态资源网站

S3 的 bucket 可以直接配置为静态资源网站。但是需要结合 CloudFront 才能支持 HTTPS 访问。请求者付费的 bucket，不允许设置为静态网站。

CloudFront 数据的分发支持两种类型：

- Web Distribution
- RTMP



## Storage Gateway

当你本地服务器想要访问 AWS S3 的时候，除了 API，AWS 还提供了几种网关可供使用：

- File Gateway - 像访问文件或者共享文件那样访问 S3 资源
- Volume Gateway - 通过 iSCSI 设备的方式连接。细分为两种：
- Stored Volumes - 所有数据都保存在本地，但是能异步备份到 S3
- Cached Volumes - 所有的数据都保存到 S3，本地只存放经常访问的数据
- Tape Gateway - 模拟磁带访问的网关，数据异步备份到 S3 Glacier 或 Glacier Deep Archive



## Athena 和 Macie

Athena 是交互式的查询服务，无须部署。可使用 SQL 来查询 S3 数据。支持的数据格式包括：CSV，JSON，Apache Parquet。

Macie 是一种可通过 NLP 和 ML 来协助你发现，分类和保护敏感数据的服务。它可以扫描 S3 中的数据，看是否包含 PII（Personally Identifiable Information）或者涉及版权的数据。

​	



