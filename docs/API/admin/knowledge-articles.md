# 后台知识条目 API

## 查询知识条目列表

- 请求方法：`GET`
- 请求路径：`/api/v1/admin/knowledge-articles`
- 功能描述：分页查询 FAQ、处理方案等知识条目。
- 认证要求：需要登录，权限码 `knowledge:list`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| page | integer | 否 | 页码 |
| per_page | integer | 否 | 每页数量 |
| q | string | 否 | 标题、问题、答案关键词 |
| knowledge_base_id | integer | 否 | 知识库 ID |
| category_id | integer | 否 | 分类 ID |
| status | string | 否 | `draft`、`pending_review`、`published`、`disabled` |
| rag_sync_status | string | 否 | `not_synced`、`syncing`、`success`、`failed` |
| sort | string | 否 | 排序字段，默认 `-updated_at` |

- 请求示例：

```http
GET /api/v1/admin/knowledge-articles?knowledge_base_id=1&status=published&page=1&per_page=20
Authorization: Bearer <access_token>
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data[].id | integer | 知识 ID |
| data[].knowledge_no | string | 知识编号 |
| data[].title | string | 标题 |
| data[].knowledge_base | object | 所属知识库 |
| data[].category | object | 分类 |
| data[].status | string | 知识状态 |
| data[].rag_sync_status | string | RAG 同步状态 |
| data[].embedding_model | string | embedding 模型快照 |
| data[].embedding_dimension | integer | 向量维度快照 |
| data[].updated_at | string | 更新时间 |
| meta | object | 分页信息 |

- 响应示例：

```json
{
  "data": [
    {
      "id": 12,
      "knowledge_no": "KB202605150001",
      "title": "账号冻结处理流程",
      "knowledge_base": {
        "id": 1,
        "code": "ops-faq",
        "name": "运维 FAQ"
      },
      "category": {
        "id": 3,
        "name": "账号管理"
      },
      "status": "published",
      "rag_sync_status": "success",
      "embedding_model": "bge-m3",
      "embedding_dimension": 1024,
      "updated_at": "2026-05-15T12:00:00+08:00"
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
| forbidden | 无知识列表权限 | 联系管理员授权 |
| validation_error | 查询参数非法 | 修正参数 |

## 创建知识条目

- 请求方法：`POST`
- 请求路径：`/api/v1/admin/knowledge-articles`
- 功能描述：在指定知识库下创建知识草稿。条目的 embedding 模型和向量维度自动继承知识库配置。
- 认证要求：需要登录，权限码 `knowledge:create`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| knowledge_base_id | integer | 是 | 知识库 ID |
| category_id | integer | 是 | 分类 ID |
| title | string | 是 | 标题 |
| question | string | 是 | FAQ 问题 |
| answer | string | 是 | FAQ 答案 |
| tags | array | 否 | 标签列表 |
| applicable_scope | string | 否 | 适用范围 |

- 请求示例：

```json
{
  "knowledge_base_id": 1,
  "category_id": 3,
  "title": "账号冻结处理流程",
  "question": "账号被冻结后如何处理？",
  "answer": "先确认冻结原因。若为密码错误冻结，可由管理员核实后解冻；若为安全冻结，需要走人工审批。",
  "tags": ["账号", "冻结"],
  "applicable_scope": "后台登录账号"
}
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.id | integer | 知识 ID |
| data.knowledge_no | string | 知识编号 |
| data.status | string | 初始状态，`draft` |
| data.embedding_model | string | embedding 模型快照 |
| data.embedding_dimension | integer | 向量维度快照 |

- 响应示例：

```json
{
  "data": {
    "id": 12,
    "knowledge_no": "KB202605150001",
    "status": "draft",
    "embedding_model": "bge-m3",
    "embedding_dimension": 1024
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| knowledge_base_not_found | 知识库不存在或停用 | 选择有效知识库 |
| category_not_found | 分类不存在 | 选择有效分类 |
| validation_error | 参数校验失败 | 修正字段 |

## 获取知识条目详情

- 请求方法：`GET`
- 请求路径：`/api/v1/admin/knowledge-articles/{id}`
- 功能描述：查看知识条目的正文、审核记录和同步状态。
- 认证要求：需要登录，权限码 `knowledge:detail`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| id | integer | 是 | 路径参数，知识 ID |

- 请求示例：

```http
GET /api/v1/admin/knowledge-articles/12
Authorization: Bearer <access_token>
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.id | integer | 知识 ID |
| data.knowledge_no | string | 知识编号 |
| data.title | string | 标题 |
| data.question | string | 问题 |
| data.answer | string | 答案 |
| data.tags | array | 标签 |
| data.status | string | 状态 |
| data.review_records | array | 审核记录 |
| data.rag_sync_status | string | 同步状态 |
| data.rag_sync_error | string | 同步错误 |
| data.embedding_model | string | embedding 模型 |
| data.embedding_dimension | integer | 向量维度 |

- 响应示例：

```json
{
  "data": {
    "id": 12,
    "knowledge_no": "KB202605150001",
    "title": "账号冻结处理流程",
    "question": "账号被冻结后如何处理？",
    "answer": "先确认冻结原因...",
    "tags": ["账号", "冻结"],
    "status": "published",
    "review_records": [],
    "rag_sync_status": "success",
    "rag_sync_error": null,
    "embedding_model": "bge-m3",
    "embedding_dimension": 1024
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| knowledge_not_found | 知识不存在 | 返回列表重新选择 |
| forbidden | 无详情权限 | 联系管理员授权 |

## 更新知识条目

- 请求方法：`PATCH`
- 请求路径：`/api/v1/admin/knowledge-articles/{id}`
- 功能描述：更新草稿或驳回后的知识条目。已发布知识修改后回到草稿或待审核状态，需重新发布后才参与问答。
- 认证要求：需要登录，权限码 `knowledge:update`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| id | integer | 是 | 路径参数，知识 ID |
| category_id | integer | 否 | 分类 ID |
| title | string | 否 | 标题 |
| question | string | 否 | FAQ 问题 |
| answer | string | 否 | FAQ 答案 |
| tags | array | 否 | 标签 |
| applicable_scope | string | 否 | 适用范围 |

- 请求示例：

```json
{
  "answer": "先确认冻结原因。若为密码错误冻结，由管理员核实后解冻并提醒用户修改密码。"
}
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.id | integer | 知识 ID |
| data.status | string | 更新后的状态 |
| data.version_no | integer | 版本号 |
| data.updated_at | string | 更新时间 |

- 响应示例：

```json
{
  "data": {
    "id": 12,
    "status": "draft",
    "version_no": 2,
    "updated_at": "2026-05-15T16:30:00+08:00"
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| knowledge_not_found | 知识不存在 | 返回列表重新选择 |
| invalid_knowledge_status | 当前状态不可编辑 | 停用或复制后再编辑 |
| validation_error | 参数校验失败 | 修正字段 |

## 提交审核

- 请求方法：`POST`
- 请求路径：`/api/v1/admin/knowledge-articles/{id}/submit-review`
- 功能描述：将草稿知识提交给知识库管理员审核。
- 认证要求：需要登录，权限码 `knowledge:submit-review`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| id | integer | 是 | 路径参数，知识 ID |

- 请求示例：

```json
{}
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.id | integer | 知识 ID |
| data.status | string | `pending_review` |

- 响应示例：

```json
{
  "data": {
    "id": 12,
    "status": "pending_review"
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| invalid_knowledge_status | 非草稿状态不可提交 | 刷新详情后重试 |
| knowledge_not_found | 知识不存在 | 返回列表重新选择 |

## 审核知识

- 请求方法：`POST`
- 请求路径：`/api/v1/admin/knowledge-articles/{id}/review`
- 功能描述：审核知识条目。审核通过后知识仍保持 `pending_review`，表示可发布但尚未发布；驳回后回到 `draft` 修改。
- 认证要求：需要登录，权限码 `knowledge:review`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| id | integer | 是 | 路径参数，知识 ID |
| review_result | string | 是 | `approved` 或 `rejected` |
| review_comment | string | 否 | 审核意见 |

- 请求示例：

```json
{
  "review_result": "approved",
  "review_comment": "内容准确，可以发布。"
}
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.id | integer | 知识 ID |
| data.review_status | string | 审核状态 |
| data.status | string | 知识状态；通过后为 `pending_review`，驳回后为 `draft` |

- 响应示例：

```json
{
  "data": {
    "id": 12,
    "review_status": "approved",
    "status": "pending_review"
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| invalid_knowledge_status | 当前状态不可审核 | 刷新详情 |
| validation_error | 审核结果非法 | 使用合法枚举 |

## 发布知识

- 请求方法：`POST`
- 请求路径：`/api/v1/admin/knowledge-articles/{id}/publish`
- 功能描述：发布审核通过的知识，生成知识同步事件，由 Worker 同步到 AnythingLLM，并写入符合知识库模型和维度约束的 pgvector 切片向量。
- 认证要求：需要登录，权限码 `knowledge:publish`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| id | integer | 是 | 路径参数，知识 ID |

- 请求示例：

```json
{}
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.id | integer | 知识 ID |
| data.status | string | `published` |
| data.rag_sync_status | string | 初始为 `syncing` |
| data.event_id | string | 知识同步事件 ID |
| data.published_at | string | 发布时间 |

- 响应示例：

```json
{
  "data": {
    "id": 12,
    "status": "published",
    "rag_sync_status": "syncing",
    "event_id": "knowledge.published.202605150001",
    "published_at": "2026-05-15T17:00:00+08:00"
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| review_required | 知识未审核通过 | 先完成审核 |
| embedding_config_mismatch | 知识快照与知识库模型或维度不一致 | 重建知识条目或联系管理员 |
| queue_unavailable | NATS JetStream 不可用 | 稍后重试或人工处理 |
| knowledge_not_found | 知识不存在 | 返回列表重新选择 |

## 停用知识

- 请求方法：`POST`
- 请求路径：`/api/v1/admin/knowledge-articles/{id}/disable`
- 功能描述：停用已发布知识，使其不再参与后续问答，并记录审计日志。
- 认证要求：需要登录，权限码 `knowledge:disable`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| id | integer | 是 | 路径参数，知识 ID |
| reason | string | 是 | 停用原因 |

- 请求示例：

```json
{
  "reason": "处理流程已更新，旧方案停用。"
}
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.id | integer | 知识 ID |
| data.status | string | `disabled` |
| data.updated_at | string | 更新时间 |

- 响应示例：

```json
{
  "data": {
    "id": 12,
    "status": "disabled",
    "updated_at": "2026-05-15T18:00:00+08:00"
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| invalid_knowledge_status | 当前状态不可停用 | 刷新详情后重试 |
| validation_error | 停用原因为空 | 补充原因 |

## 查询知识同步记录

- 请求方法：`GET`
- 请求路径：`/api/v1/admin/knowledge-articles/{id}/sync-records`
- 功能描述：查看知识发布后同步 AnythingLLM 和写入 pgvector 的记录。
- 认证要求：需要登录，权限码 `knowledge:sync-record:list`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| id | integer | 是 | 路径参数，知识 ID |
| page | integer | 否 | 页码 |
| per_page | integer | 否 | 每页数量 |

- 请求示例：

```http
GET /api/v1/admin/knowledge-articles/12/sync-records?page=1&per_page=10
Authorization: Bearer <access_token>
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data[].id | integer | 同步记录 ID |
| data[].provider | string | RAG 服务 |
| data[].event_id | string | 事件 ID |
| data[].sync_status | string | 同步状态 |
| data[].error_message | string | 错误信息 |
| data[].synced_at | string | 同步完成时间 |
| meta | object | 分页信息 |

- 响应示例：

```json
{
  "data": [
    {
      "id": 9001,
      "provider": "anythingllm",
      "event_id": "knowledge.published.202605150001",
      "sync_status": "success",
      "error_message": null,
      "synced_at": "2026-05-15T17:00:20+08:00"
    }
  ],
  "meta": {
    "total": 1,
    "page": 1,
    "per_page": 10,
    "total_pages": 1
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| knowledge_not_found | 知识不存在 | 返回列表重新选择 |
| forbidden | 无同步记录权限 | 联系管理员授权 |
