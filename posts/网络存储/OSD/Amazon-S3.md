本文主要介绍 [Amazon S3 基本接口协议](https://docs.aws.amazon.com/zh_cn/AmazonS3/latest/API/API_Operations_Amazon_Simple_Storage_Service.html)，官网是按照字母顺序来介绍的，本文按功能来介绍。

---

# CreateBucket

## 描述

用于创建一个 bucket。永远不允许匿名请求创建存储桶，要创建 bucket，您必须向亚马逊注册，并拥有有效的 AWS Access Key ID 来验证请求。

通过创建存储桶，你将自动成为存储桶的所有者。

有关桶名的规则可以参考 [Bucket naming rules](https://docs.aws.amazon.com/AmazonS3/latest/userguide/bucketnamingrules.html)

默认情况下，桶创建在 US East (N. Virginia) 区域，你也可以指定区域，比如你居住在欧洲，可能在 Europe (Ireland) 区域创建存储桶会更合适。



### Access control lists (ACLs)

ACL可以设置桶的权限

>[!NOTE]
>
>If your CreateBucket request sets bucket owner enforced for S3 Object Ownership and specifies a bucket ACL that provides access to an external AWS account, your request fails with a `400` error and returns the `InvalidBucketAclWithObjectOwnership` error code. For more information, see [Controlling object ownership](https://docs.aws.amazon.com/AmazonS3/latest/userguide/about-object-ownership.html) in the *Amazon S3 User Guide*.

有两种方法可以设置 ACL：

-   使用 `x-amz-acl` 请求头指定预先配置好的 ACL。S3 支持一组预定义的 ACL，称为 `canned ACLs`。具体可参考 [Canned ACL](https://docs.aws.amazon.com/AmazonS3/latest/dev/acl-overview.html#CannedACL)
-   使用 `x-amz-grant-read`，`x-amz-grant-write`，`x-amz-grant-read-acp`，`x-amz-grant-write-acp` 和 `x-amz-grant-full-control` 头显式指定访问权限。每个被授权者格式为 `type=value` 键值对，其中 type 可以为：
    -   id：AWS 账户 ID，比如 `x-amz-grant-read: id="11112222333", id="444455556666"`
    -   uri：向预定的组授权
    -   emailAddress：AWS 账户邮箱（只在部分区域支持）



### 权限

除了 `s3:CreateBucket` 之外，以下权限也需要在 CreateBucket Head 里指定：

-   ACLs - 如果 CreateBucket 请求指定了 ACL 权限，并且 ACL 是 `public-read`，`public-read-write`，`authenticated-read`，或者如果你通过任何其他 ACL 明确指定了访问权限，则 `s3:CreateBucket` 和 `s3:PutBucket` ACL 权限都是必需的。如果 ACL `CreateBucket` 请求是私有的或者没有指定任何 ACL，那么只需要 `s3:CreateBucket` 权限。
-   Object Lock - 如果在 `CreateBucket` 请求中将 `ObjectLockEnabledForBucket` 设置为 `true`，则需要 `s3:PutBucketObjectLockConfiguration` 和 `s3:PutBucketVersioning` 权限。
-   S3 Object Ownership - 如果 `CreateBucket` 请求包含 `x-amz-object-ownership` 标头，则需要 `s3:PutBucketOwnershipControls` 权限。



## Request Syntax

```http
PUT / HTTP/1.1
Host: Bucket.s3.amazonaws.com
x-amz-acl: ACL
x-amz-grant-full-control: GrantFullControl
x-amz-grant-read: GrantRead
x-amz-grant-read-acp: GrantReadACP
x-amz-grant-write: GrantWrite
x-amz-grant-write-acp: GrantWriteACP
x-amz-bucket-object-lock-enabled: ObjectLockEnabledForBucket
x-amz-object-ownership: ObjectOwnership
<?xml version="1.0" encoding="UTF-8"?>
<CreateBucketConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
   <LocationConstraint>string</LocationConstraint>
</CreateBucketConfiguration>
```



## URI Request Parameters

-   Bucket（必须）

要创建的桶的名称

-   x-amz-acl

预设的 ACL 权限，可选值：`private | public-read | public-read-write | authenticated-read`

-   x-amz-bucket-object-lock-enabled

是否开启对象锁定

-   x-amz-grant-full-control

允许被授权者对存储桶拥有读、写、读 ACP 和写 ACP 权限

-   x-amz-grant-read

允许被授权者列出桶中的对象

-   x-amz-grant-read-acp

允许被授权者读取存储桶 ACL

-   x-amz-grant-write

允许被授权者在桶中创建新对象。对于已有对象，如果是桶或对象所有者，还允许删除和覆盖这些对象。

-   x-amz-grant-write-acp

允许被授权者写入适用存储桶的 ACL

-   x-amz-object-ownership

所有者权限控制，可选值：`BucketOwnerPreferred | ObjectWriter | BucketOwnerEnforced`

BucketOwnerPreferred - 如果上传到存储桶的对象是使用 `bucket-owner-full-control` 上传的，则这些对象的所有权转变为存储桶所有者。

ObjectWriter - 如果上传到存储桶的对象是使用 `bucket-owner-full-control` 上传的，则上传用户拥有该对象。

BucketOwnerEnforced - 访问控制列表 (ACL) 被禁用，不再影响权限。存储桶拥有者自动拥有并可以完全控制存储桶中的每个对象。存储桶仅接受未指定 ACL 或存储桶所有者完全控制 ACL 的 PUT 请求，比如 `bucket-owner-full-control` ACL 或以 XML 格式表示的此 ACL 的等效形式。



## Request Body

消息体为 XML 格式

-   CreateBucketConfiguration（必须）

Root level tag for the CreateBucketConfiguration parameters

-   LocationConstraint

指定存储区域，可选值：

```text
af-south-1 | ap-east-1 | ap-northeast-1 | ap-northeast-2 | ap-northeast-3 | ap-south-1 | ap-southeast-1 | ap-southeast-2 | ca-central-1 | cn-north-1 | cn-northwest-1 | EU | eu-central-1 | eu-north-1 | eu-south-1 | eu-west-1 | eu-west-2 | eu-west-3 | me-south-1 | sa-east-1 | us-east-2 | us-gov-east-1 | us-gov-west-1 | us-west-1 | us-west-2
```



## Response Syntax

```http
HTTP/1.1 200
Location: Location
```



## Response Elements

如果操作成功，服务将发回一个 HTTP 200 响应。 响应返回以下 HTTP 头。

-   Location

一个正斜杠，后跟存储桶的名称。



## 示例

这个请求创建了一个名为colorpictures的桶

```http
PUT / HTTP/1.1
Host: colorpictures.s3.<Region>.amazonaws.com
Content-Length: 0
Date: Wed, 01 Mar  2006 12:00:00 GMT
Authorization: authorization string
```

回复

```http
HTTP/1.1 200 OK
x-amz-id-2: YgIPIfBiKa2bj0KMg95r/0zo3emzU4dzsD4rcKCHQUAdQkf3ShJTOOpXUueF6QKo
x-amz-request-id: 236A8905248E5A01
Date: Wed, 01 Mar  2006 12:00:00 GMT

Location: /colorpictures
Content-Length: 0
Connection: close
Server: AmazonS3
```

这个请求设置区域

```http
PUT / HTTP/1.1
Host: bucketName.s3.amazonaws.com
Date: Wed, 12 Oct 2009 17:50:00 GMT
Authorization: authorization string
Content-Type: text/plain
Content-Length: 124

<CreateBucketConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/"> 
	<LocationConstraint>Europe</LocationConstraint> 
</CreateBucketConfiguration >
```

这个请求设置 BucketOwnerEnforced 权限

```http
PUT / HTTP/1.1
Host: DOC-EXAMPLE-BUCKET.s3.<Region>.amazonaws.com
Content-Length: 0
x-amz-object-ownership: BucketOwnerEnforced
Date: Tue, 30 Nov  2021 12:00:00 GMT
Authorization: authorization string
```

这个请求创建了一个名为 colorpictures 的 bucket，并向通过电子邮件地址标识的 AWS 帐户授予写权限。

```http
PUT HTTP/1.1
Host: colorpictures.s3.<Region>.amazonaws.com
x-amz-date: Sat, 07 Apr 2012 00:54:40 GMT
Authorization: authorization string
x-amz-grant-write: emailAddress="xyz@amazon.com", emailAddress="abc@amazon.com"
```

这个请求创建了一个名为 colorpictures 的 bucket，并将 ACL 设置为 private。

```http
PUT / HTTP/1.1
Host: colorpictures.s3.<Region>.amazonaws.com
Content-Length: 0
x-amz-acl: private
Date: Wed, 01 Mar  2006 12:00:00 GMT
Authorization: authorization string
```





---

# PutObject

## 描述







---

# DeleteObject











---

# DeleteBucket







---

# CreateMultipartUpload

## 描述

此操作启动多分段上传并返回 uploadID。此 uploadID 用于关联特定多分段上传中的所有部分。您可以在每个后续的上传分段请求中指定此 uploadID。您还可以用这个 uploadID 去完成或中止多分段上传请求。

如果配置生命周期规则来中止不完整的多分段上传，则上传必须在存储桶生命周期配置中指定的天数内完成。否则，将中止分段上传。

对于请求签名，分段上传只是一系列常规请求。你启动分段上传，发送一个或多个上传分段的请求，然后完成分段上传过程，都是单独签名每个请求。签名分段上传请求并没有什么特别之处。

你可以选择请求服务器端加密。对于服务器端加密，Amazon S3 在将数据写入其数据中心的磁盘时对其进行加密，并在访问时对其进行解密。你也可以提供自己的加密密钥，或使用 AWS KMS 密钥或 Amazon S3 托管加密密钥。 如果你选择提供自己的加密密钥，则在 UploadPart 和 UploadPartCopy 请求中提供的请求标头必须与在 CreateMultipartUpload 启动上传的请求中使用的标头匹配。

如果要使用 AWS KMS 密钥执行加密，请求者必须拥有对密钥执行 `kms:Decrypt` 和 `kms:GenerateDataKey` 操作的权限。

关于权限部分后面再补充



## 语法

### Request Syntax

```http
POST /{Key+}?uploads HTTP/1.1
Host: Bucket.s3.amazonaws.com
x-amz-acl: ACL
Cache-Control: CacheControl
Content-Disposition: ContentDisposition
Content-Encoding: ContentEncoding
Content-Language: ContentLanguage
Content-Type: ContentType
Expires: Expires
x-amz-grant-full-control: GrantFullControl
x-amz-grant-read: GrantRead
x-amz-grant-read-acp: GrantReadACP
x-amz-grant-write-acp: GrantWriteACP
x-amz-server-side-encryption: ServerSideEncryption
x-amz-storage-class: StorageClass
x-amz-website-redirect-location: WebsiteRedirectLocation
x-amz-server-side-encryption-customer-algorithm: SSECustomerAlgorithm
x-amz-server-side-encryption-customer-key: SSECustomerKey
x-amz-server-side-encryption-customer-key-MD5: SSECustomerKeyMD5
x-amz-server-side-encryption-aws-kms-key-id: SSEKMSKeyId
x-amz-server-side-encryption-context: SSEKMSEncryptionContext
x-amz-server-side-encryption-bucket-key-enabled: BucketKeyEnabled
x-amz-request-payer: RequestPayer
x-amz-tagging: Tagging
x-amz-object-lock-mode: ObjectLockMode
x-amz-object-lock-retain-until-date: ObjectLockRetainUntilDate
x-amz-object-lock-legal-hold: ObjectLockLegalHoldStatus
x-amz-expected-bucket-owner: ExpectedBucketOwner
x-amz-checksum-algorithm: ChecksumAlgorithm
```

示例

```http
POST /example-object?uploads HTTP/1.1
Host: example-bucket.s3.<Region>.amazonaws.com
Date: Mon, 1 Nov 2010 20:34:56 GMT
Authorization: authorization string
```



### URI Request Parameters

略



###  request body

无



### Response Syntax

```http
HTTP/1.1 200
x-amz-abort-date: AbortDate
x-amz-abort-rule-id: AbortRuleId
x-amz-server-side-encryption: ServerSideEncryption
x-amz-server-side-encryption-customer-algorithm: SSECustomerAlgorithm
x-amz-server-side-encryption-customer-key-MD5: SSECustomerKeyMD5
x-amz-server-side-encryption-aws-kms-key-id: SSEKMSKeyId
x-amz-server-side-encryption-context: SSEKMSEncryptionContext
x-amz-server-side-encryption-bucket-key-enabled: BucketKeyEnabled
x-amz-request-charged: RequestCharged
x-amz-checksum-algorithm: ChecksumAlgorithm
<?xml version="1.0" encoding="UTF-8"?>
<InitiateMultipartUploadResult>
   <Bucket>string</Bucket>
   <Key>string</Key>
   <UploadId>string</UploadId>
</InitiateMultipartUploadResult>
```

示例

```http
HTTP/1.1 200 OK
x-amz-id-2: Uuag1LuByRx9e6j5Onimru9pO4ZVKnJ2Qz7/C1NPcfTWAtRPfTaOFg==
x-amz-request-id: 656c76696e6727732072657175657374
Date:  Mon, 1 Nov 2010 20:34:56 GMT
Transfer-Encoding: chunked
Connection: keep-alive
Server: AmazonS3
<?xml version="1.0" encoding="UTF-8"?>
<InitiateMultipartUploadResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
	<Bucket>example-bucket</Bucket>
	<Key>example-object</Key>
	<UploadId>VXBsb2FkIElEIGZvciA2aWWpbmcncyBteS1tb3ZpZS5tMnRzIHVwbG9hZA</UploadId>
</InitiateMultipartUploadResult>
```



### Response Elements

略



---

# AbortMultipartUpload

## 描述

此操作用于中止分段上传。中止分段上传后，无法再使用该 uploadID 上传其他分段。任何先前上传的分段所占用的存储空间将被释放。但是，当前正在进行的分段上传可能成功也可能失败。因此，可能需要多次中止给定的分段上传，以完全释放所有分段消耗的存储空间。

为了确认已经移除所有分段，你应该使用 ListParts 去检查分段列表为空。

有关分段上传权限的问题，请参考 [Multipart Upload and Permissions](https://docs.aws.amazon.com/AmazonS3/latest/dev/mpuAndPermissions.html)



## 语法

### Request Syntax

```http
DELETE /Key+?uploadId=UploadId HTTP/1.1
Host: Bucket.s3.amazonaws.com
x-amz-request-payer: RequestPayer
x-amz-expected-bucket-owner: ExpectedBucketOwner
```

示例

```http
DELETE /example-object?uploadId=VXBsb2FkIElEIGZvciBlbHZpbmcncyBteS1tb3ZpZS5tMnRzIHVwbG9hZ HTTP/1.1
Host: example-bucket.s3.<Region>.amazonaws.com
Date:  Mon, 1 Nov 2010 20:34:56 GMT
Authorization: authorization string
```



### URI Request Parameters

-   Bucket（必须）

上传对象所在桶的名称

-   Key（必须）

上传对象的名称

-   uploadID（必须）

用来区分分段上传的标识 ID

-   x-amz-request-payer

确认请求者知道他们将为请求付费。存储桶所有者无需在其请求中指定此参数。

-   x-amz-expected-bucket-owner

预期是 bucket 所有者的账号 ID，如果不对，则返回 `403 Forbidden`



###  request body

无



### Response Syntax

```http
HTTP/1.1 204
x-amz-request-charged: RequestCharged
```

示例

```http
HTTP/1.1 204 OK
x-amz-id-2: Weag1LuByRx9e6j5Onimru9pO4ZVKnJ2Qz7/C1NPcfTWAtRPfTaOFg==
x-amz-request-id: 996c76696e6727732072657175657374
Date:  Mon, 1 Nov 2010 20:34:56 GMT
Content-Length: 0
Connection: keep-alive
Server: AmazonS3
```



### Response Elements

如果操作成功，服务端将发回一个 HTTP 204响应。响应返回以下HTTP头：

-   x-amz-request-charged

如果存在，则表明请求者成功地为该请求付费。值为 `requester`





