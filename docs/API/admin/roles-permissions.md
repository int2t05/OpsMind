# 后台角色权限 API

## 查询角色列表

- 请求方法：`GET`
- 请求路径：`/api/v1/admin/roles`
- 功能描述：查询后台角色列表，用于账号授权。
- 认证要求：需要登录，权限码 `role:list`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| page | integer | 否 | 页码 |
| per_page | integer | 否 | 每页数量 |
| q | string | 否 | 角色编码或名称关键词 |
| status | string | 否 | `active`、`disabled` |

- 请求示例：

```http
GET /api/v1/admin/roles?page=1&per_page=20
Authorization: Bearer <access_token>
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data[].id | integer | 角色 ID |
| data[].code | string | 角色编码 |
| data[].name | string | 角色名称 |
| data[].status | string | 状态 |
| data[].remark | string | 备注 |
| meta | object | 分页信息 |

- 响应示例：

```json
{
  "data": [
    {
      "id": 2,
      "code": "operator",
      "name": "运维人员",
      "status": "active",
      "remark": "处理申告和回访"
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
| forbidden | 无角色列表权限 | 联系管理员授权 |

## 查询权限树

- 请求方法：`GET`
- 请求路径：`/api/v1/admin/permissions`
- 功能描述：查询菜单、按钮、接口权限树，用于角色授权和前端菜单展示。
- 认证要求：需要登录，权限码 `permission:list`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| type | string | 否 | `menu`、`button`、`api`，为空返回全部 |
| status | string | 否 | `active`、`disabled` |

- 请求示例：

```http
GET /api/v1/admin/permissions?type=menu
Authorization: Bearer <access_token>
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data[].id | integer | 权限 ID |
| data[].parent_id | integer | 父权限 ID |
| data[].type | string | 权限类型 |
| data[].name | string | 名称 |
| data[].code | string | 权限编码 |
| data[].path | string | 菜单路径或接口路径 |
| data[].method | string | HTTP 方法 |
| data[].children | array | 子权限 |

- 响应示例：

```json
{
  "data": [
    {
      "id": 100,
      "parent_id": null,
      "type": "menu",
      "name": "申告处理",
      "code": "ticket",
      "path": "/admin/tickets",
      "method": null,
      "children": []
    }
  ]
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| unauthorized | 未登录 | 重新登录 |
| forbidden | 无权限树查看权限 | 联系管理员授权 |
| validation_error | 参数非法 | 修正参数 |

## 更新角色权限

- 请求方法：`PATCH`
- 请求路径：`/api/v1/admin/roles/{id}/permissions`
- 功能描述：更新角色绑定的菜单、按钮和接口权限。
- 认证要求：需要登录，权限码 `role:permission:update`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| id | integer | 是 | 路径参数，角色 ID |
| permission_ids | array | 是 | 权限 ID 列表 |

- 请求示例：

```json
{
  "permission_ids": [100, 101, 102]
}
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.role_id | integer | 角色 ID |
| data.permission_count | integer | 绑定权限数量 |
| data.updated_at | string | 更新时间 |

- 响应示例：

```json
{
  "data": {
    "role_id": 2,
    "permission_count": 3,
    "updated_at": "2026-05-15T18:20:00+08:00"
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| role_not_found | 角色不存在 | 返回列表重新选择 |
| permission_not_found | 权限不存在或停用 | 选择有效权限 |
| validation_error | 参数校验失败 | 修正参数 |
