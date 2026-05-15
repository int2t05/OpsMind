# 后台知识候选 API

## 查询知识候选列表

- 请求方法：`GET`
- 请求路径：`/api/v1/admin/knowledge-candidates`
- 功能描述：查询由人工处理申告沉淀出的知识候选。
- 认证要求：需要登录，权限码 `knowledge-candidate:list`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| page | integer | 否 | 页码 |
| per_page | integer | 否 | 每页数量 |
| q | string | 否 | 标题或摘要关键词 |
| status | string | 否 | `pending_review`、`converted`、`ignored` |

- 请求示例：

```http
GET /api/v1/admin/knowledge-candidates?status=pending_review&page=1&per_page=20
Authorization: Bearer <access_token>
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data[].id | integer | 候选 ID |
| data[].ticket_id | integer | 来源申告 ID |
| data[].ticket_no | string | 来源申告编号 |
| data[].title | string | 标题 |
| data[].summary | string | 摘要 |
| data[].status | string | 状态 |
| data[].created_by | object | 创建人 |
| data[].created_at | string | 创建时间 |
| meta | object | 分页信息 |

- 响应示例：

```json
{
  "data": [
    {
      "id": 7001,
      "ticket_id": 501,
      "ticket_no": "TK202605150001",
      "title": "账号密码错误冻结处理流程",
      "summary": "当账号因连续密码错误被冻结时...",
      "status": "pending_review",
      "created_by": {
        "id": 2,
        "real_name": "运维人员"
      },
      "created_at": "2026-05-15T15:20:00+08:00"
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
| forbidden | 无知识候选列表权限 | 联系管理员授权 |

## 转为知识条目

- 请求方法：`POST`
- 请求路径：`/api/v1/admin/knowledge-candidates/{id}/convert`
- 功能描述：将知识候选转为知识条目草稿，后续仍需走提交审核和发布流程。
- 认证要求：需要登录，权限码 `knowledge-candidate:convert`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| id | integer | 是 | 路径参数，候选 ID |
| knowledge_base_id | integer | 是 | 目标知识库 ID |
| category_id | integer | 是 | 目标分类 ID |
| title | string | 是 | 知识标题 |
| question | string | 是 | FAQ 问题 |
| answer | string | 是 | FAQ 答案 |
| tags | array | 否 | 标签 |

- 请求示例：

```json
{
  "knowledge_base_id": 1,
  "category_id": 3,
  "title": "账号密码错误冻结处理流程",
  "question": "账号因密码错误被冻结后如何处理？",
  "answer": "由管理员核实冻结原因后解冻，并提醒用户修改密码。",
  "tags": ["账号", "冻结"]
}
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.candidate_id | integer | 候选 ID |
| data.knowledge_id | integer | 新知识 ID |
| data.status | string | 候选状态，`converted` |

- 响应示例：

```json
{
  "data": {
    "candidate_id": 7001,
    "knowledge_id": 12,
    "status": "converted"
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| candidate_not_found | 候选不存在 | 返回列表重新选择 |
| invalid_candidate_status | 候选已处理 | 刷新列表 |
| knowledge_base_not_found | 知识库不存在或停用 | 选择有效知识库 |
| validation_error | 参数校验失败 | 修正字段 |

## 忽略知识候选

- 请求方法：`POST`
- 请求路径：`/api/v1/admin/knowledge-candidates/{id}/ignore`
- 功能描述：忽略不适合沉淀为知识库的候选。
- 认证要求：需要登录，权限码 `knowledge-candidate:ignore`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| id | integer | 是 | 路径参数，候选 ID |
| reason | string | 是 | 忽略原因 |

- 请求示例：

```json
{
  "reason": "该问题属于一次性环境故障，不沉淀为通用知识。"
}
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.id | integer | 候选 ID |
| data.status | string | `ignored` |

- 响应示例：

```json
{
  "data": {
    "id": 7001,
    "status": "ignored"
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| candidate_not_found | 候选不存在 | 返回列表重新选择 |
| invalid_candidate_status | 候选已处理 | 刷新列表 |
| validation_error | 忽略原因为空 | 补充原因 |
