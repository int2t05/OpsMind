# 后台数据看板 API

## 查询看板摘要

- 请求方法：`GET`
- 请求路径：`/api/v1/admin/dashboard/summary`
- 功能描述：查询 MVP 数据看板摘要，包括问答量、解决率、转人工量、申告处理量、知识同步状态和外部服务健康状态。
- 认证要求：需要登录，权限码 `dashboard:summary`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| date_start | string | 否 | 统计开始日期，格式 `YYYY-MM-DD` |
| date_end | string | 否 | 统计结束日期，格式 `YYYY-MM-DD` |

- 请求示例：

```http
GET /api/v1/admin/dashboard/summary?date_start=2026-05-01&date_end=2026-05-15
Authorization: Bearer <access_token>
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.range.date_start | string | 统计开始日期 |
| data.range.date_end | string | 统计结束日期 |
| data.chat.total_sessions | integer | 问答会话数 |
| data.chat.resolved_count | integer | 已解决反馈数 |
| data.chat.unresolved_count | integer | 未解决反馈数 |
| data.chat.resolve_rate | number | 解决率，0 到 1 |
| data.chat.avg_elapsed_ms | integer | 平均问答耗时 |
| data.tickets.total | integer | 申告总数 |
| data.tickets.pending | integer | 待处理数量 |
| data.tickets.processing | integer | 处理中数量 |
| data.tickets.completed | integer | 已完成数量 |
| data.knowledge.total_articles | integer | 知识条目总数 |
| data.knowledge.published_articles | integer | 已发布知识数 |
| data.knowledge.sync_failed | integer | 同步失败知识数 |
| data.services[].target | string | 外部服务 |
| data.services[].healthy | boolean | 是否健康 |
| data.services[].latency_ms | integer | 最近检测耗时 |

- 响应示例：

```json
{
  "data": {
    "range": {
      "date_start": "2026-05-01",
      "date_end": "2026-05-15"
    },
    "chat": {
      "total_sessions": 128,
      "resolved_count": 86,
      "unresolved_count": 22,
      "resolve_rate": 0.8,
      "avg_elapsed_ms": 4200
    },
    "tickets": {
      "total": 34,
      "pending": 6,
      "processing": 8,
      "completed": 18,
      "closed": 2
    },
    "knowledge": {
      "total_articles": 45,
      "published_articles": 32,
      "sync_failed": 1
    },
    "services": [
      {
        "target": "anythingllm",
        "healthy": true,
        "latency_ms": 120
      },
      {
        "target": "vllm",
        "healthy": true,
        "latency_ms": 260
      },
      {
        "target": "minio",
        "healthy": true,
        "latency_ms": 40
      },
      {
        "target": "nats",
        "healthy": true,
        "latency_ms": 20
      }
    ]
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| unauthorized | 未登录 | 重新登录 |
| forbidden | 无看板权限 | 联系管理员授权 |
| validation_error | 日期参数非法 | 使用 `YYYY-MM-DD` 格式 |

## 查询问答趋势

- 请求方法：`GET`
- 请求路径：`/api/v1/admin/dashboard/chat-trends`
- 功能描述：查询指定日期范围内的问答量、解决量和转人工趋势。
- 认证要求：需要登录，权限码 `dashboard:chat-trends`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| date_start | string | 否 | 统计开始日期，格式 `YYYY-MM-DD` |
| date_end | string | 否 | 统计结束日期，格式 `YYYY-MM-DD` |
| granularity | string | 否 | `day`、`week`，默认 `day` |

- 请求示例：

```http
GET /api/v1/admin/dashboard/chat-trends?date_start=2026-05-01&date_end=2026-05-15&granularity=day
Authorization: Bearer <access_token>
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data[].date | string | 统计日期 |
| data[].total_sessions | integer | 问答会话数 |
| data[].resolved_count | integer | 已解决数 |
| data[].unresolved_count | integer | 未解决数 |
| data[].ticket_created_count | integer | 转申告数量 |

- 响应示例：

```json
{
  "data": [
    {
      "date": "2026-05-15",
      "total_sessions": 16,
      "resolved_count": 10,
      "unresolved_count": 3,
      "ticket_created_count": 2
    }
  ]
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| forbidden | 无趋势查看权限 | 联系管理员授权 |
| validation_error | 日期或粒度非法 | 修正查询参数 |

## 查询待办摘要

- 请求方法：`GET`
- 请求路径：`/api/v1/admin/dashboard/todos`
- 功能描述：查询当前用户的待处理申告、待审核知识、同步失败知识等后台待办数量。
- 认证要求：需要登录，权限码 `dashboard:todos`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| 无 | - | - | 无 |

- 请求示例：

```http
GET /api/v1/admin/dashboard/todos
Authorization: Bearer <access_token>
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.pending_tickets | integer | 待处理申告数 |
| data.assigned_tickets | integer | 分配给当前用户的未完成申告数 |
| data.pending_knowledge_reviews | integer | 待审核知识数 |
| data.failed_knowledge_syncs | integer | 同步失败知识数 |
| data.unresolved_feedbacks | integer | 未解决反馈数 |

- 响应示例：

```json
{
  "data": {
    "pending_tickets": 6,
    "assigned_tickets": 3,
    "pending_knowledge_reviews": 4,
    "failed_knowledge_syncs": 1,
    "unresolved_feedbacks": 8
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| unauthorized | 未登录 | 重新登录 |
| forbidden | 无待办摘要权限 | 联系管理员授权 |
