# 后台日志审计 API

## 查询审计日志

- 请求方法：`GET`
- 请求路径：`/api/v1/admin/audit-logs`
- 功能描述：查询敏感业务操作审计日志，如冻结账号、发布知识、处理申告、修改配置。
- 认证要求：需要登录，权限码 `audit:list`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| page | integer | 否 | 页码 |
| per_page | integer | 否 | 每页数量 |
| module | string | 否 | 模块名 |
| action | string | 否 | 操作动作 |
| biz_type | string | 否 | 业务类型 |
| biz_id | integer | 否 | 业务 ID |
| user_id | integer | 否 | 操作人 ID |
| created_at_start | string | 否 | 开始时间 |
| created_at_end | string | 否 | 结束时间 |

- 请求示例：

```http
GET /api/v1/admin/audit-logs?module=knowledge&action=publish&page=1&per_page=20
Authorization: Bearer <access_token>
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data[].id | integer | 审计日志 ID |
| data[].user | object | 操作人 |
| data[].module | string | 模块 |
| data[].action | string | 动作 |
| data[].biz_type | string | 业务类型 |
| data[].biz_id | integer | 业务 ID |
| data[].before_data | object | 变更前数据 |
| data[].after_data | object | 变更后数据 |
| data[].ip_address | string | 来源 IP |
| data[].created_at | string | 创建时间 |
| meta | object | 分页信息 |

- 响应示例：

```json
{
  "data": [
    {
      "id": 10001,
      "user": {
        "id": 1,
        "real_name": "系统管理员"
      },
      "module": "knowledge",
      "action": "publish",
      "biz_type": "knowledge_article",
      "biz_id": 12,
      "before_data": {
        "status": "pending_review"
      },
      "after_data": {
        "status": "published",
        "rag_sync_status": "syncing"
      },
      "ip_address": "192.168.1.10",
      "created_at": "2026-05-15T17:00:00+08:00"
    }
  ],
  "meta": {
    "total": 1,
    "page": 1,
    "per_page": 20,
    "total_pages": 1
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| unauthorized | 未登录 | 重新登录 |
| forbidden | 无审计日志权限 | 联系管理员授权 |
| validation_error | 查询参数非法 | 修正参数 |

## 查询操作日志

- 请求方法：`GET`
- 请求路径：`/api/v1/admin/operation-logs`
- 功能描述：查询后台接口操作日志，用于排查请求、响应和接口权限问题。
- 认证要求：需要登录，权限码 `operation-log:list`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| page | integer | 否 | 页码 |
| per_page | integer | 否 | 每页数量 |
| module | string | 否 | 模块名 |
| action | string | 否 | 操作动作 |
| success | boolean | 否 | 是否成功 |
| user_id | integer | 否 | 操作人 ID |
| created_at_start | string | 否 | 开始时间 |
| created_at_end | string | 否 | 结束时间 |

- 请求示例：

```http
GET /api/v1/admin/operation-logs?success=false&page=1&per_page=20
Authorization: Bearer <access_token>
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data[].id | integer | 操作日志 ID |
| data[].user | object | 操作人 |
| data[].module | string | 模块 |
| data[].action | string | 动作 |
| data[].request_method | string | 请求方法 |
| data[].request_path | string | 请求路径 |
| data[].response_code | integer | HTTP 状态码 |
| data[].success | boolean | 是否成功 |
| data[].created_at | string | 创建时间 |
| meta | object | 分页信息 |

- 响应示例：

```json
{
  "data": [
    {
      "id": 20001,
      "user": {
        "id": 2,
        "real_name": "运维人员"
      },
      "module": "ticket",
      "action": "status_update",
      "request_method": "PATCH",
      "request_path": "/api/v1/admin/tickets/501/status",
      "response_code": 200,
      "success": true,
      "created_at": "2026-05-15T14:20:00+08:00"
    }
  ],
  "meta": {
    "total": 1,
    "page": 1,
    "per_page": 20,
    "total_pages": 1
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| unauthorized | 未登录 | 重新登录 |
| forbidden | 无操作日志权限 | 联系管理员授权 |
| validation_error | 查询参数非法 | 修正参数 |

## 查询登录日志

- 请求方法：`GET`
- 请求路径：`/api/v1/admin/login-logs`
- 功能描述：查询后台账号登录成功和失败记录。
- 认证要求：需要登录，权限码 `login-log:list`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| page | integer | 否 | 页码 |
| per_page | integer | 否 | 每页数量 |
| username | string | 否 | 登录账号 |
| login_result | string | 否 | `success`、`failed` |
| login_at_start | string | 否 | 登录开始时间 |
| login_at_end | string | 否 | 登录结束时间 |

- 请求示例：

```http
GET /api/v1/admin/login-logs?login_result=failed&page=1&per_page=20
Authorization: Bearer <access_token>
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data[].id | integer | 登录日志 ID |
| data[].user_id | integer | 用户 ID |
| data[].username | string | 登录账号快照 |
| data[].login_result | string | 登录结果 |
| data[].fail_reason | string | 失败原因 |
| data[].ip_address | string | 来源 IP |
| data[].user_agent | string | 浏览器信息 |
| data[].login_at | string | 登录时间 |
| meta | object | 分页信息 |

- 响应示例：

```json
{
  "data": [
    {
      "id": 30001,
      "user_id": null,
      "username": "ops001",
      "login_result": "failed",
      "fail_reason": "invalid_credentials",
      "ip_address": "192.168.1.11",
      "user_agent": "Mozilla/5.0",
      "login_at": "2026-05-15T08:50:00+08:00"
    }
  ],
  "meta": {
    "total": 1,
    "page": 1,
    "per_page": 20,
    "total_pages": 1
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| unauthorized | 未登录 | 重新登录 |
| forbidden | 无登录日志权限 | 联系管理员授权 |
| validation_error | 查询参数非法 | 修正参数 |

## 获取审计日志详情

- 请求方法：`GET`
- 请求路径：`/api/v1/admin/audit-logs/{id}`
- 功能描述：查询单条审计日志详情，用于还原敏感操作的变更前后数据。
- 认证要求：需要登录，权限码 `audit:detail`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| id | integer | 是 | 路径参数，审计日志 ID |

- 请求示例：

```http
GET /api/v1/admin/audit-logs/10001
Authorization: Bearer <access_token>
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.id | integer | 审计日志 ID |
| data.user | object | 操作人 |
| data.module | string | 模块 |
| data.action | string | 动作 |
| data.biz_type | string | 业务类型 |
| data.biz_id | integer | 业务 ID |
| data.before_data | object | 变更前数据 |
| data.after_data | object | 变更后数据 |
| data.ip_address | string | 来源 IP |
| data.created_at | string | 创建时间 |

- 响应示例：

```json
{
  "data": {
    "id": 10001,
    "user": {
      "id": 1,
      "username": "admin",
      "real_name": "系统管理员"
    },
    "module": "knowledge",
    "action": "publish",
    "biz_type": "knowledge_article",
    "biz_id": 12,
    "before_data": {
      "status": "pending_review",
      "rag_sync_status": "not_synced"
    },
    "after_data": {
      "status": "published",
      "rag_sync_status": "syncing"
    },
    "ip_address": "192.168.1.10",
    "created_at": "2026-05-15T17:00:00+08:00"
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| audit_log_not_found | 审计日志不存在 | 返回列表重新选择 |
| forbidden | 无审计详情权限 | 联系管理员授权 |

## 导出审计日志

- 请求方法：`POST`
- 请求路径：`/api/v1/admin/audit-logs/export`
- 功能描述：按筛选条件导出审计日志。MVP 可同步生成 CSV 文件并返回文件 ID，文件通过 MinIO/S3 适配层保存。
- 认证要求：需要登录，权限码 `audit:export`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| module | string | 否 | 模块名 |
| action | string | 否 | 操作动作 |
| biz_type | string | 否 | 业务类型 |
| user_id | integer | 否 | 操作人 ID |
| created_at_start | string | 否 | 开始时间 |
| created_at_end | string | 否 | 结束时间 |
| file_format | string | 否 | `csv`，默认 `csv` |

- 请求示例：

```json
{
  "module": "knowledge",
  "action": "publish",
  "created_at_start": "2026-05-01T00:00:00+08:00",
  "created_at_end": "2026-05-15T23:59:59+08:00",
  "file_format": "csv"
}
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.file_id | integer | 导出文件 ID |
| data.file_name | string | 文件名 |
| data.total | integer | 导出记录数 |
| data.created_at | string | 创建时间 |

- 响应示例：

```json
{
  "data": {
    "file_id": 9100,
    "file_name": "audit-logs-20260515.csv",
    "total": 120,
    "created_at": "2026-05-15T18:40:00+08:00"
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| forbidden | 无审计导出权限 | 联系管理员授权 |
| validation_error | 导出参数非法 | 修正筛选条件 |
| object_storage_unavailable | MinIO 或 S3 适配层不可用 | 稍后重试 |
