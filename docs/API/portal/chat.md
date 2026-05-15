# 门户智能问答 API

## 创建问答会话

- 请求方法：`POST`
- 请求路径：`/api/v1/portal/chat-sessions`
- 功能描述：门户用户提交运维问题，系统通过 AnythingLLM 完整 RAG 流程和 vLLM OpenAI-compatible 适配层同步返回完整答案、来源和置信度。
- 认证要求：公开接口，可匿名访问。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| question | string | 是 | 用户问题 |
| session_no | string | 否 | 已有会话编号；为空时后端生成 |
| knowledge_base_code | string | 否 | 知识库编码；为空时使用默认知识库 |
| reporter_contact | string | 否 | 用户联系方式，用于后续申告关联 |
| context | array | 否 | 可选上下文消息，MVP 可为空 |

- 请求示例：

```json
{
  "question": "账号被冻结了应该怎么处理？",
  "knowledge_base_code": "ops-faq",
  "reporter_contact": "13800000000"
}
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.session_no | string | 会话编号 |
| data.answer | string | 同步返回的完整答案 |
| data.status | string | 状态，`answered`、`need_ticket` 或 `failed` |
| data.confidence_score | number | 置信度，0 到 1 |
| data.sources | array | 来源知识片段 |
| data.sources[].knowledge_id | integer | 知识条目 ID |
| data.sources[].title | string | 来源标题 |
| data.sources[].snippet | string | 命中片段 |
| data.model_name | string | vLLM 模型名称 |
| data.rag_provider | string | RAG 服务，MVP 为 `anythingllm` |
| data.elapsed_ms | integer | 问答耗时毫秒 |

- 响应示例：

```json
{
  "data": {
    "session_no": "CHAT202605150001",
    "answer": "请先确认账号冻结原因，再联系系统管理员进行解冻申请。若连续输错密码导致冻结，可等待策略时间或提交申告。",
    "status": "answered",
    "confidence_score": 0.86,
    "sources": [
      {
        "knowledge_id": 12,
        "title": "账号冻结处理流程",
        "snippet": "账号冻结分为密码错误冻结和管理员冻结两类..."
      }
    ],
    "model_name": "qwen-local",
    "rag_provider": "anythingllm",
    "elapsed_ms": 3850
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| validation_error | 问题为空或长度超限 | 修正问题内容 |
| knowledge_base_not_found | 知识库不存在或停用 | 使用默认知识库或联系管理员 |
| rag_unavailable | AnythingLLM 不可用 | 前端展示转人工入口 |
| model_unavailable | vLLM 不可用 | 前端展示转人工入口 |
| rate_limit_exceeded | 匿名问答过于频繁 | 稍后重试 |

## 提交问答反馈

- 请求方法：`POST`
- 请求路径：`/api/v1/portal/chat-sessions/{session_no}/feedback`
- 功能描述：记录用户对问答结果的反馈，用于统计解决率和触发申告入口。
- 认证要求：公开接口，可匿名访问。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| session_no | string | 是 | 路径参数，问答会话编号 |
| feedback_type | string | 是 | `resolved` 已解决，`unresolved` 未解决 |
| remark | string | 否 | 反馈说明 |

- 请求示例：

```json
{
  "feedback_type": "unresolved",
  "remark": "步骤执行后仍无法登录"
}
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.id | integer | 反馈 ID |
| data.session_no | string | 会话编号 |
| data.feedback_type | string | 反馈类型 |
| data.created_at | string | 创建时间 |

- 响应示例：

```json
{
  "data": {
    "id": 101,
    "session_no": "CHAT202605150001",
    "feedback_type": "unresolved",
    "created_at": "2026-05-15T10:30:00+08:00"
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| session_not_found | 会话不存在 | 提示用户重新发起问答 |
| duplicate_feedback | 已提交过反馈 | 不重复提交 |
| validation_error | 反馈类型非法 | 使用合法枚举值 |
