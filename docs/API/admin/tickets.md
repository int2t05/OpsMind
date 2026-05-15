# 后台申告处理 API

## 查询申告列表

- 请求方法：`GET`
- 请求路径：`/api/v1/admin/tickets`
- 功能描述：后台运维人员分页查询申告列表，支持按状态、紧急程度、处理人和关键词筛选。
- 认证要求：需要登录，权限码 `ticket:list`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| page | integer | 否 | 页码，默认 `1` |
| per_page | integer | 否 | 每页数量，默认 `20` |
| q | string | 否 | 标题、描述、申告编号关键词 |
| status | string | 否 | `pending`、`processing`、`need_more_info`、`completed`、`closed` |
| urgency_level | string | 否 | `low`、`medium`、`high` |
| assignee_id | integer | 否 | 当前处理人 ID |
| created_at_start | string | 否 | 创建开始时间 |
| created_at_end | string | 否 | 创建结束时间 |
| sort | string | 否 | 排序字段，默认 `-created_at` |

- 请求示例：

```http
GET /api/v1/admin/tickets?status=pending&urgency_level=high&page=1&per_page=20
Authorization: Bearer <access_token>
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data[].id | integer | 申告 ID |
| data[].ticket_no | string | 申告编号 |
| data[].title | string | 标题 |
| data[].reporter_name | string | 报障人 |
| data[].reporter_phone | string | 联系方式 |
| data[].urgency_level | string | 紧急程度 |
| data[].status | string | 状态 |
| data[].assignee | object | 当前处理人 |
| data[].created_at | string | 创建时间 |
| meta | object | 分页信息 |

- 响应示例：

```json
{
  "data": [
    {
      "id": 501,
      "ticket_no": "TK202605150001",
      "title": "账号冻结无法登录",
      "reporter_name": "张三",
      "reporter_phone": "13800000000",
      "urgency_level": "high",
      "status": "pending",
      "assignee": null,
      "created_at": "2026-05-15T10:35:00+08:00"
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
| unauthorized | 未登录或 token 无效 | 重新登录 |
| forbidden | 无申告列表权限 | 联系管理员授权 |
| validation_error | 查询参数非法 | 修正参数 |

## 获取申告详情

- 请求方法：`GET`
- 请求路径：`/api/v1/admin/tickets/{id}`
- 功能描述：查看申告基础信息、问答上下文、附件、处理记录和回访记录。
- 认证要求：需要登录，权限码 `ticket:detail`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| id | integer | 是 | 路径参数，申告 ID |

- 请求示例：

```http
GET /api/v1/admin/tickets/501
Authorization: Bearer <access_token>
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.id | integer | 申告 ID |
| data.ticket_no | string | 申告编号 |
| data.source_session | object | 来源问答会话 |
| data.reporter_name | string | 报障人 |
| data.reporter_phone | string | 联系方式 |
| data.title | string | 标题 |
| data.description | string | 描述 |
| data.impact_scope | string | 影响范围 |
| data.urgency_level | string | 紧急程度 |
| data.status | string | 状态 |
| data.attachments | array | 附件列表 |
| data.process_records | array | 处理记录 |
| data.visit_records | array | 回访记录 |

- 响应示例：

```json
{
  "data": {
    "id": 501,
    "ticket_no": "TK202605150001",
    "source_session": {
      "session_no": "CHAT202605150001",
      "question": "账号被冻结了应该怎么处理？",
      "answer": "请先确认账号冻结原因..."
    },
    "reporter_name": "张三",
    "reporter_phone": "13800000000",
    "title": "账号冻结无法登录",
    "description": "根据机器人指引处理后仍然无法登录后台。",
    "impact_scope": "影响运维值班登录",
    "urgency_level": "high",
    "status": "pending",
    "attachments": [],
    "process_records": [],
    "visit_records": []
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| ticket_not_found | 申告不存在 | 返回列表重新选择 |
| unauthorized | 未登录 | 重新登录 |
| forbidden | 无详情权限 | 联系管理员授权 |

## 更新申告状态

- 请求方法：`PATCH`
- 请求路径：`/api/v1/admin/tickets/{id}/status`
- 功能描述：运维人员执行申告状态流转，记录处理过程和处理结果。
- 认证要求：需要登录，权限码 `ticket:process`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| id | integer | 是 | 路径参数，申告 ID |
| action | string | 是 | `start_processing`、`request_more_info`、`complete`、`close` |
| process_content | string | 是 | 处理过程 |
| process_result | string | 否 | 处理结果；`complete` 时必填 |
| assignee_id | integer | 否 | 指定处理人；为空默认当前用户 |
| closed_reason | string | 否 | 关闭原因；`close` 时必填 |

- 请求示例：

```json
{
  "action": "complete",
  "process_content": "已核实为密码连续错误导致冻结，并完成解冻。",
  "process_result": "用户已可正常登录后台。"
}
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.id | integer | 申告 ID |
| data.ticket_no | string | 申告编号 |
| data.status | string | 更新后的状态 |
| data.process_record_id | integer | 处理记录 ID |
| data.updated_at | string | 更新时间 |

- 响应示例：

```json
{
  "data": {
    "id": 501,
    "ticket_no": "TK202605150001",
    "status": "completed",
    "process_record_id": 8001,
    "updated_at": "2026-05-15T14:20:00+08:00"
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| invalid_status_transition | 状态流转不合法 | 刷新详情后按当前状态操作 |
| validation_error | 必填字段缺失 | 补充处理过程或处理结果 |
| ticket_not_found | 申告不存在 | 返回列表重新选择 |
| forbidden | 无处理权限 | 联系管理员授权 |

## 新增回访记录

- 请求方法：`POST`
- 请求路径：`/api/v1/admin/tickets/{id}/visit-records`
- 功能描述：为已处理申告登记回访结果。
- 认证要求：需要登录，权限码 `ticket:visit`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| id | integer | 是 | 路径参数，申告 ID |
| visit_result | string | 是 | `satisfied`、`normal`、`unsatisfied` |
| visit_content | string | 否 | 回访内容 |

- 请求示例：

```json
{
  "visit_result": "satisfied",
  "visit_content": "用户确认已恢复登录。"
}
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.id | integer | 回访记录 ID |
| data.ticket_id | integer | 申告 ID |
| data.visit_result | string | 回访结果 |
| data.created_at | string | 创建时间 |

- 响应示例：

```json
{
  "data": {
    "id": 3001,
    "ticket_id": 501,
    "visit_result": "satisfied",
    "created_at": "2026-05-15T15:00:00+08:00"
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| ticket_not_found | 申告不存在 | 返回列表重新选择 |
| invalid_ticket_status | 当前状态不适合回访 | 先完成处理 |
| validation_error | 回访结果非法 | 使用合法枚举 |

## 生成知识候选

- 请求方法：`POST`
- 请求路径：`/api/v1/admin/tickets/{id}/knowledge-candidate`
- 功能描述：从已处理申告生成知识库候选，进入知识审核沉淀流程。
- 认证要求：需要登录，权限码 `knowledge-candidate:create`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| id | integer | 是 | 路径参数，申告 ID |
| title | string | 是 | 候选标题 |
| summary | string | 是 | 候选摘要 |

- 请求示例：

```json
{
  "title": "账号密码错误冻结处理流程",
  "summary": "当账号因连续密码错误被冻结时，先核实冻结原因，再由管理员执行解冻并提醒用户修改密码。"
}
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.id | integer | 知识候选 ID |
| data.ticket_id | integer | 来源申告 ID |
| data.status | string | 候选状态 |

- 响应示例：

```json
{
  "data": {
    "id": 7001,
    "ticket_id": 501,
    "status": "pending_review"
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| ticket_not_found | 申告不存在 | 返回列表重新选择 |
| invalid_ticket_status | 申告未完成 | 完成处理后再生成 |
| candidate_exists | 已生成候选 | 打开已有候选 |
| validation_error | 参数校验失败 | 修正标题或摘要 |
