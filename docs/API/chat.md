# 智能问答接口

> 基础路径：`/api/v1/portal` | 认证：JWT | 功能：RAG 管道增强问答 + SSE 流式输出

## RAG 管道概述

问答请求经过以下 RAG 管道步骤：

```
用户问题
  → 查询改写 (可选)    — LLM 消除指代歧义
  → 多路检索 (可选)    — LLM 生成 2-4 个子查询
  → 向量检索          — pgvector cosine 相似度
  → BM25 检索 (可选)  — 稀疏检索 + RRF 融合
  → 重排序   (可选)    — LLM 重新评分候选分块
  → LLM 生成          — 带上下文生成答案 (SSE 流式)
```

## 1. 创建问答会话（流式 SSE）

```http
POST /api/v1/portal/chat-sessions/stream
Authorization: Bearer <token>
Content-Type: application/json
```

**请求体：**

```json
{
  "question": "如何重置 VPN 密码？",
  "kb_id": 1,
  "rag_options": {
    "top_k": 5,
    "query_rewrite": true,
    "multi_route": true,
    "hybrid": true,
    "rerank": true
  }
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| question | string | ✓ | 用户问题 |
| kb_id | int64 | ✓ | 目标知识库 ID |
| rag_options | object | | **v2 新增** — RAG 管道选项（不传则使用默认值） |
| rag_options.top_k | int | | 最终返回的分块数，默认 5，范围 1-20 |
| rag_options.query_rewrite | bool | | 是否启用查询改写，默认 true |
| rag_options.multi_route | bool | | 是否启用多路检索，默认 true |
| rag_options.hybrid | bool | | 是否启用 BM25 混合检索，默认 true |
| rag_options.rerank | bool | | 是否启用重排序，默认 true |

**SSE 事件流：**

响应类型为 `text/event-stream`，包含三类事件：

### step 事件 — 管道步骤进度

```
data: {"type":"step","id":"query_rewrite","label":"查询改写"}

data: {"type":"step","id":"multi_route","label":"多路检索"}

data: {"type":"step","id":"vector_retrieval","label":"向量检索"}

data: {"type":"step","id":"bm25_retrieval","label":"BM25检索"}

data: {"type":"step","id":"hybrid_fuse","label":"结果融合"}

data: {"type":"step","id":"rerank","label":"重排序"}

data: {"type":"step","id":"llm_generate","label":"LLM 生成"}
```

| step id | label | 说明 |
|---------|-------|------|
| `query_rewrite` | 查询改写 | LLM 消除对话指代歧义 |
| `multi_route` | 多路检索 | LLM 生成子查询 |
| `vector_retrieval` | 向量检索 | pgvector cosine 相似度检索 |
| `bm25_retrieval` | BM25检索 | 倒排索引稀疏检索 |
| `hybrid_fuse` | 结果融合 | RRF 融合排序 |
| `rerank` | 重排序 | LLM 重排候选分块 |
| `llm_generate` | LLM 生成 | 调用 LLM 生成答案 |

> 管道步骤可能因 `rag_options` 开关而跳过（如 `query_rewrite=false` 时跳过查询改写步骤）。

### token 事件 — 逐 token 流式发送（v2 升级）

```
data: {"type":"token","content":"VPN 密码"}
data: {"type":"token","content":"重置步骤"}
data: {"type":"token","content":"如下：\n1."}
```

> LLM 服务返回的 token 实时转发，输出速度取决于 LLM 推理速度。

### done 事件 — 流式结束，含完整元数据

```
data: {"type":"done","metadata":{"session_id":42,"question":"如何重置 VPN 密码？","answer":"VPN 密码重置步骤：1. 登录自助平台...","sources":[{"doc_name":"VPN 密码重置 FAQ","chunk_content":"...","confidence":0.85}],"confidence":0.85,"can_submit_ticket":false,"duration_ms":3200,"feedback":0,"created_at":"2026-06-11 20:30:00","pipeline":{"steps":[{"id":"query_rewrite","label":"查询改写","duration_ms":120},{"id":"vector_retrieval","label":"向量检索","duration_ms":45},{"id":"bm25_retrieval","label":"BM25检索","duration_ms":38},{"id":"hybrid_fuse","label":"结果融合","duration_ms":2},{"id":"rerank","label":"重排序","duration_ms":180},{"id":"llm_generate","label":"LLM 生成","duration_ms":2800}],"total_duration_ms":3185}}}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| metadata.pipeline | object | **v2 新增** — 管道执行耗时明细 |
| metadata.pipeline.steps | array | 各步骤耗时（含跳过的步骤不会出现） |
| metadata.pipeline.total_duration_ms | int | 管道总耗时（不含 LLM 生成） |

**错误降级（非 SSE）：**

当 LLM/Embedding 服务不可用时，直接返回 JSON 错误：

```json
{
  "code": 20001,
  "message": "当前 AI 服务暂不可用，请提交申告由人工处理",
  "data": null
}
```

**前端消费示例（v2 更新）：**

```typescript
import { streamChatSession } from '@/api/chat'

await streamChatSession(
  {
    question: '如何重置 VPN 密码？',
    kb_id: 1,
    rag_options: { top_k: 5, hybrid: true, rerank: true }
  },
  {
    onStep(id: string, label: string) {
      // 展示管道步骤进度
      currentStep.value = label
    },
    onToken(content: string) {
      // 逐 token 追加到 UI（真实流式）
      assistantMessage.content += content
    },
    onDone(session: ChatSessionResponse) {
      // 流式完成，含 pipeline 耗时明细
      currentSession.value = session
    },
    onError(error: string) {
      showError(error)
    }
  }
)
```

---

## 2. 创建问答会话（非流式）

```http
POST /api/v1/portal/chat-sessions
Authorization: Bearer <token>
```

**请求体：** 同流式接口

**成功响应 (200)：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "session_id": 42,
    "question": "如何重置 VPN 密码？",
    "answer": "VPN 密码重置步骤：1. 登录自助平台...",
    "sources": [
      {
        "doc_name": "VPN 密码重置 FAQ",
        "chunk_content": "问题：如何重置 VPN 密码？答案：...",
        "confidence": 0.85
      }
    ],
    "confidence": 0.85,
    "can_submit_ticket": false,
    "duration_ms": 3200,
    "feedback": 0,
    "pipeline": {
      "steps": [
        {"id": "query_rewrite", "label": "查询改写", "duration_ms": 120},
        {"id": "vector_retrieval", "label": "向量检索", "duration_ms": 45},
        {"id": "rerank", "label": "重排序", "duration_ms": 180},
        {"id": "llm_generate", "label": "LLM 生成", "duration_ms": 2800}
      ],
      "total_duration_ms": 3185
    },
    "created_at": "2026-06-11 20:30:00"
  }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| session_id | int64 | 会话 ID |
| question | string | 用户问题 |
| answer | string | AI 答案（或降级兜底文本） |
| sources | array | 知识来源列表 |
| sources[].doc_name | string | 来源文档名称 |
| sources[].chunk_content | string | 匹配的切片内容 |
| sources[].confidence | float | 该来源置信度 (0-1) |
| confidence | float | 整体置信度（取 sources 最高分） |
| can_submit_ticket | bool | 是否建议转人工申告 |
| duration_ms | int | RAG 管道总耗时（毫秒） |
| pipeline | object | **v2 新增** — 管道执行指标 |

---

## 3. 查询会话详情

```http
GET /api/v1/portal/chat-sessions/:id
Authorization: Bearer <token>
```

**响应：** 同创建会话响应

---

## 4. 提交反馈

```http
POST /api/v1/portal/chat-sessions/:id/feedback
Authorization: Bearer <token>
```

**请求体：**

```json
{
  "feedback": 1
}
```

| 值 | 说明 |
|----|------|
| 0 | 未评价（默认） |
| 1 | 已解决 |
| 2 | 未解决 |

---

## 降级规则

v2 RAG 管道的降级策略比 v1 更细粒度——单步骤失败不阻塞后续步骤：

| 步骤 | 失败行为 | 降级结果 |
|------|----------|----------|
| 查询改写 | 降级 | 使用原始 question |
| 多路检索 | 降级 | 使用单路检索 |
| 向量检索 | **阻塞** | 返回 `code=20002`（核心路径） |
| BM25 检索 | 降级 | BM25 结果为空，仅用向量结果 |
| RRF 融合 | 降级 | 使用单路结果（哪个有结果用哪个） |
| 重排序 | 降级 | 使用 RRF 排序结果 |
| LLM 生成 | **阻塞** | 返回 `code=20001`（核心路径） |

| 最终场景 | 行为 |
|----------|------|
| LLM 服务不可达 | 返回 `code=20001`，提示 AI 不可用 |
| pgvector 检索失败 | 返回 `code=20002`，提示 RAG 服务不可用 |
| 检索结果为空 | 返回兜底答案 + `can_submit_ticket=true` |
| 置信度 < 阈值（默认 0.6） | 返回兜底答案 + `can_submit_ticket=true` |

**兜底文本：**
- AI 服务不可用：「当前 AI 服务暂不可用，请提交申告由人工处理」
- 低置信度/无结果：「暂未找到足够匹配的知识，建议提交申告由运维人员人工处理」

## 置信度阈值

默认阈值 `0.6`，可通过 LLM 配置 API 修改：

```http
PUT /api/v1/admin/llm-configs/:id
```
