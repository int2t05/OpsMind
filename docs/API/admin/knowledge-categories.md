# 后台知识分类 API

## 查询知识分类树

- 请求方法：`GET`
- 请求路径：`/api/v1/admin/knowledge-categories`
- 功能描述：按知识库查询分类树，用于知识条目归类。
- 认证要求：需要登录，权限码 `knowledge-category:list`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| knowledge_base_id | integer | 是 | 知识库 ID |
| status | string | 否 | `active`、`disabled` |

- 请求示例：

```http
GET /api/v1/admin/knowledge-categories?knowledge_base_id=1
Authorization: Bearer <access_token>
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data[].id | integer | 分类 ID |
| data[].parent_id | integer | 父分类 ID |
| data[].name | string | 分类名称 |
| data[].sort_order | integer | 排序 |
| data[].status | string | 状态 |
| data[].children | array | 子分类 |

- 响应示例：

```json
{
  "data": [
    {
      "id": 3,
      "parent_id": null,
      "name": "账号管理",
      "sort_order": 1,
      "status": "active",
      "children": []
    }
  ]
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| knowledge_base_not_found | 知识库不存在 | 选择有效知识库 |
| forbidden | 无分类列表权限 | 联系管理员授权 |

## 创建知识分类

- 请求方法：`POST`
- 请求路径：`/api/v1/admin/knowledge-categories`
- 功能描述：在指定知识库下创建分类。
- 认证要求：需要登录，权限码 `knowledge-category:create`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| knowledge_base_id | integer | 是 | 知识库 ID |
| parent_id | integer | 否 | 父分类 ID |
| name | string | 是 | 分类名称 |
| sort_order | integer | 否 | 排序，默认 `0` |

- 请求示例：

```json
{
  "knowledge_base_id": 1,
  "name": "账号管理",
  "sort_order": 1
}
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.id | integer | 分类 ID |
| data.knowledge_base_id | integer | 知识库 ID |
| data.name | string | 分类名称 |
| data.status | string | 状态 |

- 响应示例：

```json
{
  "data": {
    "id": 3,
    "knowledge_base_id": 1,
    "name": "账号管理",
    "status": "active"
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| duplicate_category_name | 同知识库分类名称重复 | 更换分类名称 |
| knowledge_base_not_found | 知识库不存在或停用 | 选择有效知识库 |
| parent_category_not_found | 父分类不存在 | 选择有效父分类 |
| validation_error | 参数校验失败 | 修正字段 |

## 更新知识分类

- 请求方法：`PATCH`
- 请求路径：`/api/v1/admin/knowledge-categories/{id}`
- 功能描述：更新知识分类名称、排序或状态。
- 认证要求：需要登录，权限码 `knowledge-category:update`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| id | integer | 是 | 路径参数，分类 ID |
| name | string | 否 | 分类名称 |
| sort_order | integer | 否 | 排序 |
| status | string | 否 | `active`、`disabled` |

- 请求示例：

```json
{
  "name": "账号与权限",
  "sort_order": 2
}
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.id | integer | 分类 ID |
| data.name | string | 分类名称 |
| data.sort_order | integer | 排序 |
| data.status | string | 状态 |
| data.updated_at | string | 更新时间 |

- 响应示例：

```json
{
  "data": {
    "id": 3,
    "name": "账号与权限",
    "sort_order": 2,
    "status": "active",
    "updated_at": "2026-05-15T18:00:00+08:00"
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| category_not_found | 分类不存在 | 返回分类树重新选择 |
| duplicate_category_name | 同知识库分类名称重复 | 更换分类名称 |
| validation_error | 参数校验失败 | 修正字段 |
