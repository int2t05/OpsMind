# 后台运维账号 API

## 查询账号列表

- 请求方法：`GET`
- 请求路径：`/api/v1/admin/accounts`
- 功能描述：查询系统内本地模拟的运维账号列表。MVP 不对接真实企业账号中心。
- 认证要求：需要登录，权限码 `account:list`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| page | integer | 否 | 页码 |
| per_page | integer | 否 | 每页数量 |
| q | string | 否 | 账号、姓名、手机号关键词 |
| status | string | 否 | `active`、`frozen` |
| role_code | string | 否 | 角色编码 |
| sort | string | 否 | 排序字段，默认 `-created_at` |

- 请求示例：

```http
GET /api/v1/admin/accounts?q=ops&status=active&page=1&per_page=20
Authorization: Bearer <access_token>
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data[].id | integer | 账号 ID |
| data[].username | string | 登录账号 |
| data[].real_name | string | 姓名 |
| data[].phone | string | 手机号 |
| data[].email | string | 邮箱 |
| data[].status | string | 状态 |
| data[].roles | array | 角色列表 |
| data[].last_login_at | string | 最近登录时间 |
| data[].created_at | string | 创建时间 |
| meta | object | 分页信息 |

- 响应示例：

```json
{
  "data": [
    {
      "id": 2,
      "username": "ops001",
      "real_name": "运维人员",
      "phone": "13800000001",
      "email": "ops001@example.com",
      "status": "active",
      "roles": [
        {
          "code": "operator",
          "name": "运维人员"
        }
      ],
      "last_login_at": "2026-05-15T09:00:00+08:00",
      "created_at": "2026-05-10T10:00:00+08:00"
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
| forbidden | 无账号列表权限 | 联系管理员授权 |
| validation_error | 查询参数非法 | 修正参数 |

## 创建账号

- 请求方法：`POST`
- 请求路径：`/api/v1/admin/accounts`
- 功能描述：创建本地模拟运维账号，并分配角色。
- 认证要求：需要登录，权限码 `account:create`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| username | string | 是 | 登录账号，唯一 |
| password | string | 是 | 初始密码 |
| real_name | string | 是 | 姓名 |
| phone | string | 否 | 手机号 |
| email | string | 否 | 邮箱 |
| role_ids | array | 是 | 角色 ID 列表 |
| remark | string | 否 | 备注 |

- 请求示例：

```json
{
  "username": "ops001",
  "password": "Ops@123456",
  "real_name": "运维人员",
  "phone": "13800000001",
  "email": "ops001@example.com",
  "role_ids": [2],
  "remark": "MVP 本地模拟账号"
}
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.id | integer | 账号 ID |
| data.username | string | 登录账号 |
| data.real_name | string | 姓名 |
| data.status | string | 状态 |
| data.roles | array | 角色列表 |

- 响应示例：

```json
{
  "data": {
    "id": 2,
    "username": "ops001",
    "real_name": "运维人员",
    "status": "active",
    "roles": [
      {
        "code": "operator",
        "name": "运维人员"
      }
    ]
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| duplicate_username | 登录账号已存在 | 更换账号 |
| role_not_found | 角色不存在或停用 | 选择有效角色 |
| weak_password | 密码强度不足 | 使用符合规则的密码 |
| validation_error | 参数校验失败 | 按 `details` 修正 |

## 获取账号详情

- 请求方法：`GET`
- 请求路径：`/api/v1/admin/accounts/{id}`
- 功能描述：查看本地模拟运维账号详情。
- 认证要求：需要登录，权限码 `account:detail`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| id | integer | 是 | 路径参数，账号 ID |

- 请求示例：

```http
GET /api/v1/admin/accounts/2
Authorization: Bearer <access_token>
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.id | integer | 账号 ID |
| data.username | string | 登录账号 |
| data.real_name | string | 姓名 |
| data.phone | string | 手机号 |
| data.email | string | 邮箱 |
| data.status | string | 状态 |
| data.roles | array | 角色列表 |
| data.last_login_at | string | 最近登录时间 |
| data.remark | string | 备注 |

- 响应示例：

```json
{
  "data": {
    "id": 2,
    "username": "ops001",
    "real_name": "运维人员",
    "phone": "13800000001",
    "email": "ops001@example.com",
    "status": "active",
    "roles": [
      {
        "id": 2,
        "code": "operator",
        "name": "运维人员"
      }
    ],
    "last_login_at": "2026-05-15T09:00:00+08:00",
    "remark": "MVP 本地模拟账号"
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| account_not_found | 账号不存在 | 返回列表重新选择 |
| forbidden | 无详情权限 | 联系管理员授权 |

## 更新账号

- 请求方法：`PATCH`
- 请求路径：`/api/v1/admin/accounts/{id}`
- 功能描述：修改本地模拟运维账号基础资料和角色。
- 认证要求：需要登录，权限码 `account:update`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| id | integer | 是 | 路径参数，账号 ID |
| real_name | string | 否 | 姓名 |
| phone | string | 否 | 手机号 |
| email | string | 否 | 邮箱 |
| role_ids | array | 否 | 角色 ID 列表 |
| remark | string | 否 | 备注 |

- 请求示例：

```json
{
  "real_name": "运维值班人员",
  "role_ids": [2, 3]
}
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.id | integer | 账号 ID |
| data.real_name | string | 姓名 |
| data.roles | array | 角色列表 |
| data.updated_at | string | 更新时间 |

- 响应示例：

```json
{
  "data": {
    "id": 2,
    "real_name": "运维值班人员",
    "roles": [
      {
        "code": "operator",
        "name": "运维人员"
      },
      {
        "code": "knowledge_admin",
        "name": "知识库管理员"
      }
    ],
    "updated_at": "2026-05-15T16:30:00+08:00"
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| account_not_found | 账号不存在 | 返回列表重新选择 |
| role_not_found | 角色不存在或停用 | 选择有效角色 |
| validation_error | 参数校验失败 | 修正字段 |

## 冻结账号

- 请求方法：`POST`
- 请求路径：`/api/v1/admin/accounts/{id}/freeze`
- 功能描述：冻结本地模拟运维账号，冻结后不可登录后台。
- 认证要求：需要登录，权限码 `account:freeze`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| id | integer | 是 | 路径参数，账号 ID |
| reason | string | 是 | 冻结原因 |

- 请求示例：

```json
{
  "reason": "人员离岗，暂时冻结账号。"
}
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.id | integer | 账号 ID |
| data.status | string | `frozen` |
| data.updated_at | string | 更新时间 |

- 响应示例：

```json
{
  "data": {
    "id": 2,
    "status": "frozen",
    "updated_at": "2026-05-15T17:00:00+08:00"
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| account_not_found | 账号不存在 | 返回列表重新选择 |
| self_freeze_forbidden | 不允许冻结当前登录账号 | 使用其他管理员账号操作 |
| validation_error | 冻结原因为空 | 补充原因 |

## 恢复账号

- 请求方法：`POST`
- 请求路径：`/api/v1/admin/accounts/{id}/restore`
- 功能描述：恢复已冻结的本地模拟运维账号。
- 认证要求：需要登录，权限码 `account:restore`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| id | integer | 是 | 路径参数，账号 ID |
| reason | string | 是 | 恢复原因 |

- 请求示例：

```json
{
  "reason": "人员恢复值班。"
}
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.id | integer | 账号 ID |
| data.status | string | `active` |
| data.updated_at | string | 更新时间 |

- 响应示例：

```json
{
  "data": {
    "id": 2,
    "status": "active",
    "updated_at": "2026-05-15T17:10:00+08:00"
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| account_not_found | 账号不存在 | 返回列表重新选择 |
| invalid_account_status | 当前账号不是冻结状态 | 刷新详情 |
| validation_error | 恢复原因为空 | 补充原因 |
