# 角色与菜单管理接口

> 基础路径：`/api/v1/admin` | 认证：JWT + `user:manage` 权限

## 预设角色

| 角色 | 说明 | 典型权限 |
|------|------|----------|
| 系统管理员 | 系统全局管理 | user:manage, ticket:manage, knowledge:manage, config:manage |
| 运维人员 | 处理申告和回访 | ticket:manage, knowledge:create |
| 知识库管理员 | 维护和审核知识 | knowledge:manage |
| 报障人 | 门户端用户 | 无后台权限 |

---

## 角色 CRUD

### 1. 角色列表

```http
GET /api/v1/admin/roles?page=1&page_size=10
Authorization: Bearer <token>
```

**响应：**

```json
{
  "code": 0,
  "data": [
    {
      "id": 1,
      "name": "系统管理员",
      "description": "系统全局管理",
      "permissions": ["user:manage", "ticket:manage", "knowledge:manage"],
      "created_at": "2026-06-11T19:27:43Z",
      "updated_at": "2026-06-11T19:27:43Z"
    }
  ],
  "total": 4,
  "page": 1,
  "page_size": 10
}
```

### 2. 创建角色

```http
POST /api/v1/admin/roles
Authorization: Bearer <token>
```

**请求体：**

```json
{
  "name": "审计员",
  "description": "仅查看审计日志",
  "permissions": ["audit:view"]
}
```

**错误：**

| code | 说明 |
|------|------|
| 10005 | 角色名已存在 |

### 3. 角色详情

```http
GET /api/v1/admin/roles/:id
Authorization: Bearer <token>
```

### 4. 更新角色

```http
PUT /api/v1/admin/roles/:id
Authorization: Bearer <token>
```

**请求体：** 同创建（`permissions` 为全量替换）

### 5. 删除角色

```http
DELETE /api/v1/admin/roles/:id
Authorization: Bearer <token>
```

**错误：**

| code | 说明 |
|------|------|
| 10004 | 角色不存在 |

---

## 菜单管理

### 6. 菜单列表

```http
GET /api/v1/admin/menus
Authorization: Bearer <token>
```

**响应：**

```json
{
  "code": 0,
  "data": [
    {
      "id": 1,
      "name": "仪表盘",
      "path": "/admin/dashboard",
      "icon": "dashboard",
      "parent_id": 0,
      "sort_order": 1,
      "type": "menu"
    },
    {
      "id": 2,
      "name": "申告管理",
      "path": "/admin/tickets",
      "icon": "ticket",
      "parent_id": 0,
      "sort_order": 2,
      "type": "menu"
    }
  ]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| id | int64 | 菜单 ID |
| name | string | 菜单名称 |
| path | string | 前端路由路径 |
| icon | string | 图标标识 |
| parent_id | int64 | 父菜单 ID（0=顶级） |
| sort_order | int | 排序 |
| type | string | 菜单类型（menu/button） |

### 7. 更新角色菜单权限

```http
PUT /api/v1/admin/roles/:id/menus
Authorization: Bearer <token>
```

**请求体：**

```json
{
  "menu_ids": [1, 2, 3, 4]
}
```

> 采用全量替换策略：先清空角色的所有菜单关联，再插入新关联。

**成功响应：**

```json
{
  "code": 0,
  "message": "success",
  "data": null
}
```
