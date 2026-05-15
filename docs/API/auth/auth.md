# 认证与权限 API

## 登录

- 请求方法：`POST`
- 请求路径：`/api/v1/auth/login`
- 功能描述：后台用户登录，返回访问令牌、用户信息、角色和权限。
- 认证要求：公开接口。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| username | string | 是 | 登录账号 |
| password | string | 是 | 登录密码 |

- 请求示例：

```json
{
  "username": "admin",
  "password": "Admin@123456"
}
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.access_token | string | 访问令牌 |
| data.expires_in | integer | 过期秒数 |
| data.user.id | integer | 用户 ID |
| data.user.username | string | 登录账号 |
| data.user.real_name | string | 姓名 |
| data.user.status | string | 账号状态，`active` 或 `frozen` |
| data.roles | array | 角色编码列表 |
| data.permissions | array | 权限编码列表 |

- 响应示例：

```json
{
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_in": 7200,
    "user": {
      "id": 1,
      "username": "admin",
      "real_name": "系统管理员",
      "status": "active"
    },
    "roles": ["system_admin"],
    "permissions": ["ticket:list", "knowledge:publish"]
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| invalid_credentials | 账号或密码错误 | 提示用户重新输入 |
| account_frozen | 账号已冻结 | 联系管理员恢复账号 |
| validation_error | 参数校验失败 | 按 `details` 修正字段 |
| rate_limit_exceeded | 登录尝试过多 | 稍后重试 |

## 刷新令牌

- 请求方法：`POST`
- 请求路径：`/api/v1/auth/refresh`
- 功能描述：刷新访问令牌，延长后台登录态。
- 认证要求：需要 `Authorization: Bearer <access_token>`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| refresh_token | string | 否 | MVP 可不启用 refresh token；启用时传入刷新令牌 |

- 请求示例：

```json
{
  "refresh_token": "optional-refresh-token"
}
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.access_token | string | 新访问令牌 |
| data.expires_in | integer | 过期秒数 |

- 响应示例：

```json
{
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_in": 7200
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| unauthorized | token 无效或已过期 | 重新登录 |
| account_frozen | 账号已冻结 | 联系管理员 |

## 退出登录

- 请求方法：`POST`
- 请求路径：`/api/v1/auth/logout`
- 功能描述：退出后台登录态，将当前 token 加入短期黑名单。
- 认证要求：需要 `Authorization: Bearer <access_token>`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| 无 | - | - | 无请求体 |

- 请求示例：

```json
{}
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| 无 | - | 成功时返回 HTTP 204 |

- 响应示例：

```json
{}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| unauthorized | token 无效 | 前端清理登录态 |

## 获取当前用户

- 请求方法：`GET`
- 请求路径：`/api/v1/auth/profile`
- 功能描述：获取当前登录用户资料、角色和账号状态。
- 认证要求：需要 `Authorization: Bearer <access_token>`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| 无 | - | - | 无 |

- 请求示例：

```http
GET /api/v1/auth/profile
Authorization: Bearer <access_token>
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.id | integer | 用户 ID |
| data.username | string | 登录账号 |
| data.real_name | string | 姓名 |
| data.phone | string | 手机号 |
| data.email | string | 邮箱 |
| data.status | string | 账号状态 |
| data.roles | array | 角色列表 |

- 响应示例：

```json
{
  "data": {
    "id": 1,
    "username": "admin",
    "real_name": "系统管理员",
    "phone": "13800000000",
    "email": "admin@example.com",
    "status": "active",
    "roles": ["system_admin"]
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| unauthorized | 未登录或 token 失效 | 重新登录 |
| account_frozen | 账号已冻结 | 退出登录并提示 |

## 获取当前权限

- 请求方法：`GET`
- 请求路径：`/api/v1/auth/permissions`
- 功能描述：获取当前用户的菜单树、按钮权限和接口权限。
- 认证要求：需要 `Authorization: Bearer <access_token>`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| 无 | - | - | 无 |

- 请求示例：

```http
GET /api/v1/auth/permissions
Authorization: Bearer <access_token>
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.menus | array | 菜单树 |
| data.buttons | array | 按钮权限编码 |
| data.apis | array | 接口权限编码 |

- 响应示例：

```json
{
  "data": {
    "menus": [
      {
        "name": "申告处理",
        "code": "ticket",
        "path": "/admin/tickets",
        "children": []
      }
    ],
    "buttons": ["ticket:process", "knowledge:publish"],
    "apis": ["GET:/api/v1/admin/tickets", "POST:/api/v1/admin/knowledge-articles"]
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| unauthorized | 未登录或 token 失效 | 重新登录 |
| forbidden | 无权限 | 联系管理员分配角色 |
