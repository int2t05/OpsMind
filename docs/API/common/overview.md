# API 通用约定

本文档定义运维数字员工系统 MVP 的 REST API 通用约定。业务接口按模块拆分在 `docs/API/auth/`、`docs/API/portal/`、`docs/API/admin/` 等目录下。

## 1. 基础路径

- API 版本：`/api/v1`
- 数据格式：`application/json; charset=utf-8`
- 时间格式：ISO 8601，例如 `2026-05-15T10:30:00+08:00`
- 字段命名：请求和响应 JSON 使用 `snake_case`
- 管理端接口：`/api/v1/admin/*`
- 门户端接口：`/api/v1/portal/*`

## 2. 认证方式

后台管理端接口统一使用 Bearer Token：

```http
Authorization: Bearer <access_token>
```

门户端问答、申告创建和申告查询允许匿名访问。匿名接口仍需要记录来源 IP、User-Agent 和必要审计字段。

后台接口必须同时完成认证和接口权限校验。各业务文档中的权限码为后端鉴权中间件和前端菜单按钮控制的共同依据。

匿名接口必须启用 IP 级限流，避免问答、申告和附件上传被滥用。匿名上传仅允许申告附件场景使用，详见文件与对象存储 API。

## 3. 成功响应结构

单对象响应：

```json
{
  "data": {
    "id": 1
  }
}
```

列表响应：

```json
{
  "data": [],
  "meta": {
    "total": 0,
    "page": 1,
    "per_page": 20,
    "total_pages": 0
  }
}
```

## 4. 错误响应结构

错误响应必须配合 HTTP 状态码使用，不允许所有异常都返回 `200`。业务错误码使用 `snake_case`，字段级校验错误放入 `details`。

```json
{
  "error": {
    "code": "validation_error",
    "message": "请求参数校验失败",
    "details": [
      {
        "field": "title",
        "message": "标题不能为空",
        "code": "required"
      }
    ]
  }
}
```

常见错误码：

| 错误码 | 建议 HTTP 状态码 | 说明 |
|--------|------------------|------|
| validation_error | 422 | 请求字段合法 JSON 但不满足业务校验 |
| invalid_json | 400 | JSON 解析失败或请求体格式错误 |
| unauthorized | 401 | 未登录、令牌缺失或令牌无效 |
| forbidden | 403 | 已登录但缺少接口权限 |
| not_found | 404 | 资源不存在 |
| conflict | 409 | 唯一键冲突、状态冲突或配置锁定 |
| rate_limit_exceeded | 429 | 请求超过限流阈值 |
| upstream_unavailable | 502 | vLLM、AnythingLLM、MinIO 等上游不可用 |
| service_unavailable | 503 | 服务临时不可用 |

## 5. 分页和筛选

列表接口默认支持以下公共参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| page | integer | 否 | 页码，默认 `1` |
| per_page | integer | 否 | 每页数量，默认 `20`，最大 `100` |
| q | string | 否 | 关键词搜索 |
| sort | string | 否 | 排序字段，`-created_at` 表示倒序 |

## 6. 通用 HTTP 状态码

| 状态码 | 含义 | 使用场景 |
|--------|------|----------|
| 200 | OK | 查询、更新成功并返回响应体 |
| 201 | Created | 创建成功 |
| 204 | No Content | 删除、退出登录等成功且无响应体 |
| 400 | Bad Request | JSON 格式错误或基础参数错误 |
| 401 | Unauthorized | 未登录或 token 无效 |
| 403 | Forbidden | 已登录但无权限 |
| 404 | Not Found | 资源不存在 |
| 409 | Conflict | 唯一键冲突、状态冲突、配置被锁定 |
| 422 | Unprocessable Entity | 业务参数校验失败 |
| 429 | Too Many Requests | 限流 |
| 500 | Internal Server Error | 服务内部异常 |
| 502 | Bad Gateway | vLLM、AnythingLLM、MinIO 等上游异常 |
| 503 | Service Unavailable | 服务临时不可用 |

## 7. 业务枚举

API 对外统一返回字符串枚举。数据库可以使用 `smallint` 存储，但后端必须在边界层完成映射，避免前端依赖数据库数字值。

### 7.1 问答会话状态

| API 值 | 数据库值 | 说明 |
|--------|----------|------|
| processing | 1 | 处理中 |
| answered | 2 | 已完成并返回答案 |
| need_ticket | 3 | 需要转人工申告 |
| failed | 4 | 问答失败 |

### 7.2 申告状态

| API 值 | 数据库值 | 说明 |
|--------|----------|------|
| pending | 1 | 待处理 |
| processing | 2 | 处理中 |
| need_more_info | 3 | 待补充 |
| completed | 4 | 已完成 |
| closed | 5 | 已关闭 |

允许状态流转：

| 当前状态 | 可流转到 |
|----------|----------|
| pending | processing、closed |
| processing | need_more_info、completed、closed |
| need_more_info | processing、closed |
| completed | closed |
| closed | 无 |

### 7.3 紧急程度

| API 值 | 数据库值 | 说明 |
|--------|----------|------|
| low | 1 | 低 |
| medium | 2 | 中 |
| high | 3 | 高 |

### 7.4 问答反馈

| API 值 | 数据库值 | 说明 |
|--------|----------|------|
| resolved | 1 | 已解决 |
| unresolved | 2 | 未解决 |

### 7.5 知识库状态

| API 值 | 数据库值 | 说明 |
|--------|----------|------|
| active | 1 | 启用 |
| disabled | 2 | 停用 |

### 7.6 知识条目状态

| API 值 | 数据库值 | 说明 |
|--------|----------|------|
| draft | 1 | 草稿 |
| pending_review | 2 | 待审核 |
| published | 3 | 已发布 |
| disabled | 4 | 已停用 |

允许状态流转：

| 当前状态 | 操作 | 流转结果 |
|----------|------|----------|
| draft | submit-review | pending_review |
| pending_review | review approved | pending_review，`review_status` 变为 `approved` |
| pending_review | review rejected | draft，`review_status` 变为 `rejected` |
| pending_review | publish | published |
| published | update | draft，需重新审核发布 |
| published | disable | disabled |
| disabled | 无 | 无 |

### 7.7 知识审核状态

| API 值 | 数据库值 | 说明 |
|--------|----------|------|
| pending | 1 | 待审核 |
| approved | 2 | 审核通过 |
| rejected | 3 | 审核驳回 |

### 7.8 知识同步状态

| API 值 | 数据库值 | 说明 |
|--------|----------|------|
| not_synced | 1 | 未同步 |
| syncing | 2 | 同步中 |
| success | 3 | 同步成功 |
| failed | 4 | 同步失败 |

### 7.9 知识候选状态

| API 值 | 数据库值 | 说明 |
|--------|----------|------|
| pending_review | 1 | 待审核 |
| converted | 2 | 已转为知识条目 |
| ignored | 3 | 已忽略 |

### 7.10 回访结果

| API 值 | 数据库值 | 说明 |
|--------|----------|------|
| satisfied | 1 | 满意 |
| normal | 2 | 一般 |
| unsatisfied | 3 | 不满意 |

## 8. 权限码清单

后台权限码按资源和动作命名，格式为 `<resource>:<action>`。前端用于控制菜单、按钮和路由展示；后端用于接口鉴权，后端鉴权结果为准。

| 模块 | 权限码 |
|------|--------|
| 账号 | `account:list`、`account:create`、`account:detail`、`account:update`、`account:freeze`、`account:restore` |
| 角色权限 | `role:list`、`permission:list`、`role:permission:update` |
| 申告 | `ticket:list`、`ticket:detail`、`ticket:process`、`ticket:visit` |
| 知识库 | `knowledge-base:list`、`knowledge-base:create`、`knowledge-base:update` |
| 知识分类 | `knowledge-category:list`、`knowledge-category:create`、`knowledge-category:update` |
| 知识条目 | `knowledge:list`、`knowledge:create`、`knowledge:detail`、`knowledge:update`、`knowledge:submit-review`、`knowledge:review`、`knowledge:publish`、`knowledge:disable`、`knowledge:sync-record:list` |
| 知识候选 | `knowledge-candidate:list`、`knowledge-candidate:create`、`knowledge-candidate:convert`、`knowledge-candidate:ignore` |
| 文件 | `file:upload`、`file:download`、`file:detail`、`file:update` |
| 看板 | `dashboard:summary`、`dashboard:chat-trends`、`dashboard:todos` |
| 配置 | `config:list`、`config:detail`、`config:update`、`config:test-connection`、`config:model-options` |
| 日志审计 | `audit:list`、`audit:detail`、`audit:export`、`operation-log:list`、`login-log:list` |

## 9. 文件上传边界

文件上传使用 `multipart/form-data`，其他接口默认使用 `application/json`。MVP 上传限制如下：

| 项 | 约束 |
|----|------|
| 单文件大小 | 最大 20 MB |
| 允许类型 | `image/png`、`image/jpeg`、`application/pdf`、`text/plain`、`application/vnd.openxmlformats-officedocument.wordprocessingml.document` |
| 申告匿名上传 | 仅允许 `biz_type=ticket` 且 `biz_id` 为空，创建申告成功后绑定 |
| 后台上传 | 需要 Bearer Token 和 `file:upload` 权限 |
| 下载地址 | 使用短期预签名 URL，默认有效期 600 秒 |

## 10. OpenAPI 入口

机器可读接口定义维护在 `docs/API/openapi.yaml`。Markdown 文档用于业务说明和评审，OpenAPI 用于 Swagger UI、前端 API client、Mock 和联调校验。两者冲突时，以 `docs/API/openapi.yaml` 和本通用约定为准，并同步修正文档。

## 11. MVP 技术边界

- 问答接口 MVP 同步返回完整答案，SSE 流式接口后续补充。
- RAG 流程由 AnythingLLM 完整负责，后端通过 `RagClient` 适配层调用。
- 模型调用统一走 vLLM OpenAI-compatible 适配层。
- pgvector 存储系统侧知识切片向量，用于追溯和后续原生检索扩展。
- 同一个知识库必须使用同一个 `embedding_model` 和 `embedding_dimension`。
- 已存在发布或已同步切片的知识库不允许修改 embedding 模型和向量维度。
- 对象存储通过 S3-compatible 适配层接入 MinIO。
