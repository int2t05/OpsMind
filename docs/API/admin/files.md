# 文件与对象存储 API

## 上传文件

- 请求方法：`POST`
- 请求路径：`/api/v1/admin/files`
- 功能描述：上传申告附件、知识文档原件或导入文件。MVP 通过 S3-compatible 适配层写入 MinIO，数据库仅保存文件元数据。
- 认证要求：需要登录，权限码 `file:upload`。门户匿名附件上传使用 `POST /api/v1/portal/files`，不复用后台路径。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| file | file | 是 | 表单文件字段 |
| biz_type | string | 是 | 业务类型，`ticket`、`knowledge`、`import` |
| biz_id | integer | 否 | 业务 ID；创建前上传可为空 |

- 请求示例：

```http
POST /api/v1/admin/files
Content-Type: multipart/form-data
Authorization: Bearer <access_token>

file=@error-screenshot.png
biz_type=ticket
biz_id=501
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.id | integer | 文件 ID |
| data.file_name | string | 原始文件名 |
| data.file_size | integer | 文件大小，单位字节 |
| data.mime_type | string | MIME 类型 |
| data.storage_provider | string | 存储服务，MVP 为 `minio` |
| data.file_path | string | 对象存储路径 |
| data.created_at | string | 上传时间 |

- 响应示例：

```json
{
  "data": {
    "id": 9001,
    "file_name": "error-screenshot.png",
    "file_size": 245760,
    "mime_type": "image/png",
    "storage_provider": "minio",
    "file_path": "tickets/2026/05/15/error-screenshot.png",
    "created_at": "2026-05-15T10:32:00+08:00"
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| file_required | 未上传文件 | 选择文件后重试 |
| file_too_large | 文件超过大小限制 | 压缩或更换文件 |
| unsupported_file_type | 文件类型不支持 | 上传允许的文件类型 |
| object_storage_unavailable | MinIO 或 S3 适配层不可用 | 稍后重试或联系管理员 |
| validation_error | 业务参数非法 | 修正 `biz_type` 或 `biz_id` |

## 门户上传申告附件

- 请求方法：`POST`
- 请求路径：`/api/v1/portal/files`
- 功能描述：门户用户在创建申告前匿名上传附件，成功后将返回的文件 ID 放入 `POST /api/v1/portal/tickets` 的 `attachment_ids`。文件创建时 `biz_id` 为空，申告创建成功后由后端自动绑定。
- 认证要求：公开接口，可匿名访问；仅允许 `biz_type=ticket`，必须启用 IP 级限流。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| file | file | 是 | 表单文件字段 |
| biz_type | string | 是 | 固定为 `ticket` |

- 请求示例：

```http
POST /api/v1/portal/files
Content-Type: multipart/form-data

file=@error-screenshot.png
biz_type=ticket
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.id | integer | 文件 ID |
| data.file_name | string | 原始文件名 |
| data.file_size | integer | 文件大小，单位字节 |
| data.mime_type | string | MIME 类型 |
| data.storage_provider | string | 存储服务，MVP 为 `minio` |
| data.file_path | string | 对象存储路径 |
| data.created_at | string | 上传时间 |

- 响应示例：

```json
{
  "data": {
    "id": 9001,
    "file_name": "error-screenshot.png",
    "file_size": 245760,
    "mime_type": "image/png",
    "storage_provider": "minio",
    "file_path": "tickets/tmp/2026/05/15/error-screenshot.png",
    "created_at": "2026-05-15T10:32:00+08:00"
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| file_required | 未上传文件 | 选择文件后重试 |
| file_too_large | 文件超过大小限制 | 压缩或更换文件 |
| unsupported_file_type | 文件类型不支持 | 上传允许的文件类型 |
| invalid_biz_type | 匿名上传只允许申告附件 | 使用 `ticket` |
| object_storage_unavailable | MinIO 或 S3 适配层不可用 | 稍后重试或联系管理员 |
| rate_limit_exceeded | 匿名上传过于频繁 | 稍后重试 |

## 获取文件下载地址

- 请求方法：`GET`
- 请求路径：`/api/v1/admin/files/{id}/download-url`
- 功能描述：获取文件临时下载地址。后端应生成短期有效的 MinIO/S3 预签名地址。
- 认证要求：需要登录，权限码 `file:download`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| id | integer | 是 | 路径参数，文件 ID |

- 请求示例：

```http
GET /api/v1/admin/files/9001/download-url
Authorization: Bearer <access_token>
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.file_id | integer | 文件 ID |
| data.download_url | string | 临时下载地址 |
| data.expires_in | integer | 有效期秒数 |

- 响应示例：

```json
{
  "data": {
    "file_id": 9001,
    "download_url": "https://minio.example.com/opsmind/tickets/2026/05/15/error-screenshot.png?X-Amz-Signature=...",
    "expires_in": 600
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| file_not_found | 文件不存在 | 检查文件 ID |
| object_storage_unavailable | 对象存储不可用 | 稍后重试 |
| forbidden | 无下载权限 | 联系管理员授权 |

## 查询文件详情

- 请求方法：`GET`
- 请求路径：`/api/v1/admin/files/{id}`
- 功能描述：查询文件元数据，不直接返回文件内容。
- 认证要求：需要登录，权限码 `file:detail`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| id | integer | 是 | 路径参数，文件 ID |

- 请求示例：

```http
GET /api/v1/admin/files/9001
Authorization: Bearer <access_token>
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.id | integer | 文件 ID |
| data.biz_type | string | 业务类型 |
| data.biz_id | integer | 业务 ID |
| data.file_name | string | 原始文件名 |
| data.file_path | string | 对象存储路径 |
| data.file_size | integer | 文件大小，单位字节 |
| data.mime_type | string | MIME 类型 |
| data.storage_provider | string | 存储服务，MVP 为 `minio` |
| data.uploaded_by | object | 上传人 |
| data.created_at | string | 上传时间 |

- 响应示例：

```json
{
  "data": {
    "id": 9001,
    "biz_type": "ticket",
    "biz_id": 501,
    "file_name": "error-screenshot.png",
    "file_path": "tickets/2026/05/15/error-screenshot.png",
    "file_size": 245760,
    "mime_type": "image/png",
    "storage_provider": "minio",
    "uploaded_by": {
      "id": 2,
      "real_name": "运维人员"
    },
    "created_at": "2026-05-15T10:32:00+08:00"
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| file_not_found | 文件不存在 | 检查文件 ID |
| forbidden | 无文件详情权限 | 联系管理员授权 |

## 绑定文件业务对象

- 请求方法：`PATCH`
- 请求路径：`/api/v1/admin/files/{id}/binding`
- 功能描述：将创建前上传的临时文件绑定到申告、知识条目或导入任务。数据库只更新文件元数据，文件仍保存在 MinIO。
- 认证要求：需要登录，权限码 `file:update`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| id | integer | 是 | 路径参数，文件 ID |
| biz_type | string | 是 | `ticket`、`knowledge`、`import` |
| biz_id | integer | 是 | 业务 ID |

- 请求示例：

```json
{
  "biz_type": "ticket",
  "biz_id": 501
}
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.id | integer | 文件 ID |
| data.biz_type | string | 业务类型 |
| data.biz_id | integer | 业务 ID |
| data.updated_at | string | 更新时间 |

- 响应示例：

```json
{
  "data": {
    "id": 9001,
    "biz_type": "ticket",
    "biz_id": 501,
    "updated_at": "2026-05-15T10:35:00+08:00"
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| file_not_found | 文件不存在 | 检查文件 ID |
| invalid_biz_type | 业务类型非法 | 使用合法业务类型 |
| biz_target_not_found | 业务对象不存在 | 检查业务 ID |
| validation_error | 参数校验失败 | 修正参数 |
