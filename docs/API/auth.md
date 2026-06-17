# 认证接口

> 基础路径：`/api/v1/auth` | 认证：公开（login/refresh）或 JWT（change-password/logout）

## 1. 登录

```http
POST /api/v1/auth/login
```

**请求体：**

```json
{
  "username": "admin",
  "password": "Admin@123"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| username | string | ✓ | 用户名 |
| password | string | ✓ | 密码 |

**成功响应 (200)：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
    "user": {
      "id": 1,
      "username": "admin",
      "real_name": "系统管理员",
      "phone": "13800000001",
      "email": "admin@opsmind.local",
      "first_login": false
    },
    "roles": ["系统管理员"],
    "permissions": ["user:manage", "ticket:manage", "knowledge:manage"],
    "menus": [
      {
        "id": 1,
        "name": "仪表盘",
        "path": "/admin/dashboard",
        "icon": "dashboard",
        "children": []
      }
    ]
  }
}
```

**错误响应：**

| code | 说明 |
|------|------|
| 10003 | 用户名或密码错误（同一用户名 15 分钟内最多 5 次失败尝试，超额后返回此错误） |
| 10002 | 账号已被冻结 |

> 登录失败审计日志由服务端 `slog` 记录（含限流拒绝/用户不存在/密码错误/已冻结），用户名不存在与密码错误返回相同错误码以防用户名枚举。

---

## 2. 刷新令牌

```http
POST /api/v1/auth/refresh
```

**请求体：**

```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

**成功响应 (200)：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
    "user": { },
    "roles": [],
    "permissions": [],
    "menus": []
  }
}
```

> 刷新令牌有效期 7 天。过期需重新登录。

---

## 3. 修改密码

```http
POST /api/v1/auth/me/change-password
Authorization: Bearer <token>
```

**请求体：**

```json
{
  "old_password": "Admin@123",
  "new_password": "NewPass456"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| old_password | string | ✓ | 旧密码 |
| new_password | string | ✓ | 新密码（8-32 位，须含大小写字母+数字） |

**密码策略：**
- 长度 8-32 位
- 必须包含大写字母、小写字母、数字
- 正则：`^(?=.*[a-z])(?=.*[A-Z])(?=.*\d).{8,32}$`

**成功响应 (200)：**

```json
{
  "code": 0,
  "message": "success",
  "data": null
}
```

---

## 4. 登出

```http
POST /api/v1/auth/me/logout
Authorization: Bearer <token>
```

**请求体：**

```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| refresh_token | string | ✓ | 需失效的刷新令牌 |

**成功响应 (200)：**

```json
{
  "code": 0,
  "message": "success",
  "data": null
}
```

> 登出时将 refresh token 加入内存黑名单，阻止其被用于刷新。黑名单条目在 token 到期后自动清理。token 已过期或无效时仍视为登出成功。
