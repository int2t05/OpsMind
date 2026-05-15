# 门户申告 API

## 创建申告

- 请求方法：`POST`
- 请求路径：`/api/v1/portal/tickets`
- 功能描述：门户用户在机器人无法解决时提交申告，系统生成申告编号并进入待处理状态。
- 认证要求：公开接口，可匿名访问。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| source_session_no | string | 否 | 来源问答会话编号 |
| reporter_name | string | 是 | 报障人姓名 |
| reporter_phone | string | 是 | 联系方式 |
| title | string | 是 | 问题标题 |
| description | string | 是 | 问题描述 |
| impact_scope | string | 否 | 影响范围 |
| urgency_level | string | 是 | `low`、`medium`、`high` |
| attachment_ids | array | 否 | 已通过 `POST /api/v1/portal/files` 上传的附件 ID 列表 |

- 请求示例：

```json
{
  "source_session_no": "CHAT202605150001",
  "reporter_name": "张三",
  "reporter_phone": "13800000000",
  "title": "账号冻结无法登录",
  "description": "根据机器人指引处理后仍然无法登录后台。",
  "impact_scope": "影响运维值班登录",
  "urgency_level": "high",
  "attachment_ids": [9001]
}
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.id | integer | 申告 ID |
| data.ticket_no | string | 申告编号 |
| data.status | string | 申告状态，初始为 `pending` |
| data.created_at | string | 创建时间 |

- 响应示例：

```json
{
  "data": {
    "id": 501,
    "ticket_no": "TK202605150001",
    "status": "pending",
    "created_at": "2026-05-15T10:35:00+08:00"
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| validation_error | 必填字段缺失或格式错误 | 按 `details` 修正 |
| source_session_not_found | 来源会话不存在 | 去掉来源会话后重新提交 |
| file_not_found | 附件不存在 | 重新上传附件 |
| rate_limit_exceeded | 提交过于频繁 | 稍后重试 |

## 查询申告详情

- 请求方法：`GET`
- 请求路径：`/api/v1/portal/tickets/{ticket_no}`
- 功能描述：门户用户按申告编号查询申告状态、处理记录和回访结果。
- 认证要求：公开接口；需传入联系方式做轻量校验。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| ticket_no | string | 是 | 路径参数，申告编号 |
| reporter_phone | string | 是 | 查询参数，报障人联系方式 |

- 请求示例：

```http
GET /api/v1/portal/tickets/TK202605150001?reporter_phone=13800000000
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.ticket_no | string | 申告编号 |
| data.title | string | 标题 |
| data.status | string | 申告状态 |
| data.urgency_level | string | 紧急程度 |
| data.process_records | array | 处理记录 |
| data.visit_records | array | 回访记录 |
| data.created_at | string | 创建时间 |
| data.updated_at | string | 更新时间 |

- 响应示例：

```json
{
  "data": {
    "ticket_no": "TK202605150001",
    "title": "账号冻结无法登录",
    "status": "processing",
    "urgency_level": "high",
    "process_records": [
      {
        "process_status": "processing",
        "process_content": "已联系管理员核查冻结原因",
        "created_at": "2026-05-15T11:00:00+08:00"
      }
    ],
    "visit_records": [],
    "created_at": "2026-05-15T10:35:00+08:00",
    "updated_at": "2026-05-15T11:00:00+08:00"
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| ticket_not_found | 申告不存在 | 检查申告编号 |
| contact_mismatch | 联系方式不匹配 | 使用提交申告时的联系方式 |
| validation_error | 参数格式错误 | 修正查询参数 |
