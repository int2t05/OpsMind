# 后台知识库 API

## 查询知识库列表

- 请求方法：`GET`
- 请求路径：`/api/v1/admin/knowledge-bases`
- 功能描述：分页查询知识库配置列表。
- 认证要求：需要登录，权限码 `knowledge-base:list`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| page | integer | 否 | 页码 |
| per_page | integer | 否 | 每页数量 |
| q | string | 否 | 知识库编码或名称关键词 |
| status | string | 否 | `active`、`disabled` |

- 请求示例：

```http
GET /api/v1/admin/knowledge-bases?q=ops&page=1&per_page=20
Authorization: Bearer <access_token>
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data[].id | integer | 知识库 ID |
| data[].code | string | 知识库编码 |
| data[].name | string | 知识库名称 |
| data[].embedding_model | string | embedding 模型 |
| data[].embedding_dimension | integer | 向量维度 |
| data[].rag_provider | string | RAG 服务，MVP 为 `anythingllm` |
| data[].status | string | 状态 |
| data[].article_count | integer | 知识条目数量 |
| data[].chunk_count | integer | 切片数量 |
| meta | object | 分页信息 |

- 响应示例：

```json
{
  "data": [
    {
      "id": 1,
      "code": "ops-faq",
      "name": "运维 FAQ",
      "embedding_model": "bge-m3",
      "embedding_dimension": 1024,
      "rag_provider": "anythingllm",
      "status": "active",
      "article_count": 30,
      "chunk_count": 120
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
| forbidden | 无列表权限 | 联系管理员授权 |
| validation_error | 查询参数非法 | 修正参数 |

## 创建知识库

- 请求方法：`POST`
- 请求路径：`/api/v1/admin/knowledge-bases`
- 功能描述：创建知识库，并配置该知识库固定使用的 embedding 模型和向量维度。
- 认证要求：需要登录，权限码 `knowledge-base:create`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| code | string | 是 | 知识库编码，唯一 |
| name | string | 是 | 知识库名称 |
| description | string | 否 | 描述 |
| embedding_model | string | 是 | embedding 模型名称 |
| embedding_dimension | integer | 是 | 向量维度 |
| rag_provider | string | 否 | RAG 服务，默认 `anythingllm` |

- 请求示例：

```json
{
  "code": "ops-faq",
  "name": "运维 FAQ",
  "description": "高频运维问题和处理方案",
  "embedding_model": "bge-m3",
  "embedding_dimension": 1024,
  "rag_provider": "anythingllm"
}
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.id | integer | 知识库 ID |
| data.code | string | 知识库编码 |
| data.name | string | 知识库名称 |
| data.embedding_model | string | embedding 模型 |
| data.embedding_dimension | integer | 向量维度 |
| data.status | string | 状态 |

- 响应示例：

```json
{
  "data": {
    "id": 1,
    "code": "ops-faq",
    "name": "运维 FAQ",
    "embedding_model": "bge-m3",
    "embedding_dimension": 1024,
    "status": "active"
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| duplicate_code | 知识库编码已存在 | 更换编码 |
| invalid_embedding_dimension | 向量维度非法 | 选择正整数维度 |
| unsupported_rag_provider | 不支持的 RAG 服务 | MVP 使用 `anythingllm` |
| validation_error | 参数校验失败 | 按 `details` 修正 |

## 更新知识库

- 请求方法：`PATCH`
- 请求路径：`/api/v1/admin/knowledge-bases/{id}`
- 功能描述：更新知识库基础信息。若知识库已有已发布知识或已同步切片，不允许修改 `embedding_model` 和 `embedding_dimension`。
- 认证要求：需要登录，权限码 `knowledge-base:update`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| id | integer | 是 | 路径参数，知识库 ID |
| name | string | 否 | 知识库名称 |
| description | string | 否 | 描述 |
| embedding_model | string | 否 | embedding 模型；无已发布/同步数据时可改 |
| embedding_dimension | integer | 否 | 向量维度；无已发布/同步数据时可改 |
| status | string | 否 | `active`、`disabled` |

- 请求示例：

```json
{
  "name": "运维 FAQ 知识库",
  "description": "覆盖账号、网络和常见系统问题"
}
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.id | integer | 知识库 ID |
| data.code | string | 知识库编码 |
| data.name | string | 知识库名称 |
| data.embedding_model | string | embedding 模型 |
| data.embedding_dimension | integer | 向量维度 |
| data.status | string | 状态 |
| data.updated_at | string | 更新时间 |

- 响应示例：

```json
{
  "data": {
    "id": 1,
    "code": "ops-faq",
    "name": "运维 FAQ 知识库",
    "embedding_model": "bge-m3",
    "embedding_dimension": 1024,
    "status": "active",
    "updated_at": "2026-05-15T16:00:00+08:00"
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| knowledge_base_not_found | 知识库不存在 | 返回列表重新选择 |
| embedding_config_locked | 已有发布或同步数据，模型和维度不可修改 | 新建知识库或重建知识数据 |
| invalid_embedding_dimension | 向量维度非法 | 选择正整数维度 |
| validation_error | 参数校验失败 | 按 `details` 修正 |
