# Server TODO List — 后端代码改进清单

> 基于 2026-06-12 深度代码审查，按业务领域整理。  
> 共 **87 条** TODO，覆盖 **30+ 文件**。  
> 优先级定义：🔴 P0 生产隐患 → 🟡 P1 架构债务 → 🟢 P2 代码优化

---

## 1. 认证与安全 (Auth & Security)

| # | 优先级 | 文件 | 行 | 问题 | 建议修复 |
|---|--------|------|-----|------|----------|
| 1.1 | 🔴 P0 | [jwt.go](server/pkg/jwt/jwt.go) | 25 | Access Token 与 Refresh Token 结构完全相同 — 丧失双令牌安全模型，Refresh Token 可当 Access Token 使用 | Claims 增加 `TokenType` 字段；中间件拒绝 `token_type != "access"` |
| 1.2 | 🔴 P0 | [auth.go](server/internal/middleware/auth.go) | 36 | 中间件缺少 token_type 校验 | 配合 1.1 在 JWTAuth 中校验 token_type |
| 1.3 | 🔴 P0 | [auth.go](server/internal/handler/auth.go) | 85 | `userID.(int64)` 不安全类型断言 — 缺少 comma-ok，中间件重构会导致 panic | 改为 `uid, ok := userID.(int64)` |
| 1.4 | 🔴 P0 | [message.go](server/internal/handler/message.go) | 58 | `MarkAsRead` 不校验消息归属 — 水平越权，用户 A 可标记用户 B 消息已读 | 传入 currentUserID，Service 层校验 `message.user_id` |
| 1.5 | 🟡 P1 | [llm_config.go](server/internal/dto/response/llm_config.go) | 12 | APIKey 脱敏无编译期保障 — Service 层遗漏则密钥泄露 | 定义 `MarshalJSON` 自动脱敏，或拆分 List/Detail 两种响应 |
| 1.6 | 🟡 P1 | [cors.go](server/internal/middleware/cors.go) | 19 | AllowOrigins 硬编码 `localhost:5173` | 从配置读取，支持环境变量 |
| 1.7 | 🟢 P2 | [auth.go](server/internal/handler/auth.go) | 115 | 部分 Handler 未使用 `handleServiceError()`，将内部错误信息泄露到 HTTP 响应 | 全局统一错误处理 |

---

## 2. 数据一致性与事务 (Data Consistency & Transactions)

| # | 优先级 | 文件 | 行 | 问题 | 建议修复 |
|---|--------|------|-----|------|----------|
| 2.1 | 🔴 P0 | [user_service.go](server/internal/service/user_service.go) | 101 | `Create` + `AssignRoles` 不在同一事务 | 包裹在 `db.Transaction()` 中 |
| 2.2 | 🔴 P0 | [user_service.go](server/internal/service/user_service.go) | 134 | `Update` + `AssignRoles` 不在同一事务 | 同 2.1 |
| 2.3 | 🔴 P0 | [ticket_service.go](server/internal/service/ticket_service.go) | 190 | `SupplementCount >= 3` 存在竞态条件 — 并发请求可绕过上限 | SQL 原子检查: `UPDATE ... WHERE supplement_count < 3` |
| 2.4 | 🔴 P0 | [ticket_service.go](server/internal/service/ticket_service.go) | 222 | `UpdateStatus` + `CreateRecord` 不在同一事务 | 包裹在事务中 |
| 2.5 | 🔴 P0 | [scheduler.go](server/internal/service/scheduler.go) | 79-81 | `AutoCloseTickets` 不创建 TicketRecord 且无审计日志 | 遍历关闭的 ticket，逐个创建 record + audit |
| 2.6 | 🔴 P0 | [ticket_repo.go](server/internal/repository/ticket_repo.go) | 158 | 同上 — 裸 UPDATE 无 timeline 记录 | 配合 2.5 |
| 2.7 | 🟡 P1 | [llm_config_service.go](server/internal/service/llm_config_service.go) | 124 | `ClearDefault` + `Create` 不在同一事务 | 包裹在事务中 |
| 2.8 | 🟡 P1 | [llm_config_service.go](server/internal/service/llm_config_service.go) | 160 | `ClearDefault` + `Update` 不在同一事务 | 同 2.7 |
| 2.9 | 🟢 P2 | [knowledge_repo.go](server/internal/repository/knowledge_repo.go) | 118 | `ListArticles` 缺少 `.Preload("KnowledgeBase")` — 触发 N+1 查询 | 链式添加 `.Preload("KnowledgeBase")` |

---

## 3. 错误处理与日志 (Error Handling & Logging)

| # | 优先级 | 文件 | 行 | 问题 | 建议修复 |
|---|--------|------|-----|------|----------|
| 3.1 | 🔴 P0 | [dashboard_service.go](server/internal/service/dashboard_service.go) | 148,172 | `GetTrends` 2 处 `.Scan()` 错误被丢弃 | 改为 `if err := ...Scan().Error; err != nil { ... }` |
| 3.2 | 🔴 P0 | [role_service.go](server/internal/service/role_service.go) | 35,90 | 2 处 `.Count(&count)` 错误被丢弃 + 绕过 Repository 直接操作 DB | 增加 `repo.ExistsByName()`，正确处理错误 |
| 3.3 | 🔴 P0 | [processor.go](server/internal/rag/processor.go) | 88 | `Submit` 队列满时静默丢弃任务 — 调用方收到 200 但任务未提交 | `Submit` 返回 error，或改为阻塞发送 |
| 3.4 | 🔴 P0 | [embedder.go](server/internal/rag/embedder.go) | 71 | Embedding 批次失败静默跳过 — 丢失哪些 chunk 的向量不可知 | 返回失败索引列表或支持「失败即中止」 |
| 3.5 | 🔴 P0 | [storage_client.go](server/internal/adapter/storage_client.go) | 70 | `ensureBucket` 错误静默丢弃 | 至少 `slog.Warn` 记录 |
| 3.6 | 🟡 P1 | [user.go](server/internal/handler/user.go) | 83 | `err.Error()` 泄露内部错误到 HTTP 响应 | 使用 `handleServiceError(c, err)` |
| 3.7 | 🟡 P1 | [role.go](server/internal/handler/role.go) | 75 | 同上 | 同上 |
| 3.8 | 🟡 P1 | [knowledge.go](server/internal/handler/knowledge.go) | 88,338 | 同上 (2 处) | 同上 |
| 3.9 | 🟡 P1 | [audit.go](server/internal/handler/audit.go) | 40 | 同上 | 同上 |
| 3.10 | 🟡 P1 | [llm_config.go](server/internal/handler/llm_config.go) | 71 | 同上 | 同上 |
| 3.11 | 🟡 P1 | [vector_store.go](server/internal/adapter/vector_store.go) | 252 | NaN/Inf 被静默替换为 0.0 — 隐藏上游 bug | `slog.Warn` 记录或提供「拒绝写入」严格模式 |
| 3.12 | 🟡 P1 | [knowledge_service.go](server/internal/service/knowledge_service.go) | 505 | `UpdateArticleStatus` 错误静默丢弃 | 记录日志或向上返回 |
| 3.13 | 🟢 P2 | [logger.go](server/internal/middleware/logger.go) | 50 | `json.Marshal` 错误被丢弃 | 检查并记录 |
| 3.14 | 🟢 P2 | [storage_client.go](server/internal/adapter/storage_client.go) | 71 | `ensureBucket` 使用 `context.Background()` 无法取消 | 传入可取消 context |

---

## 4. 申告管理 (Ticket Management)

| # | 优先级 | 文件 | 行 | 问题 | 建议修复 |
|---|--------|------|-----|------|----------|
| 4.1 | 🔴 P0 | [ticket_repo.go](server/internal/repository/ticket_repo.go) | 156 | `AutoCloseTickets` 使用魔数 1,2,3,5 — 枚举变更则静默失败 | 使用 `model.TicketStatusPending` 等常量 |
| 4.2 | 🔴 P0 | [ticket_service.go](server/internal/service/ticket_service.go) | 381,396 | `marshalTicketTags`/`unmarshalTicketTags` 手动拼接/解析 JSON — 含双引号或逗号则畸形 | 改用 `json.Marshal`/`json.Unmarshal` |
| 4.3 | 🟡 P1 | [ticket_service.go](server/internal/service/ticket_service.go) | 66 | `rand.Intn(10000)` 生成 ticket_no — 高并发碰撞风险 | 改用雪花算法或 DB 序列 |
| 4.4 | 🟢 P2 | [ticket.go](server/internal/model/ticket.go) | 36 | `TicketRecord.TicketID` 缺少索引 — 时间线查询退化为全表扫描 | 添加 `index:idx_ticket_records_ticket_id` |
| 4.5 | 🟢 P2 | [ticket.go](server/internal/model/ticket.go) | 37 | `OperatorID` 缺少 FK 约束标签 | 添加 GORM 约束标签 |

---

## 5. 知识库与 RAG 引擎 (Knowledge & RAG)

| # | 优先级 | 文件 | 行 | 问题 | 建议修复 |
|---|--------|------|-----|------|----------|
| 5.1 | 🔴 P0 | [knowledge_service.go](server/internal/service/knowledge_service.go) | 322 | `ArticleStatusDisabled` 定义为 4，但 `Disable()` 写入 `article.Status = 0` — 枚举常量从未被使用 | 统一改为 `model.ArticleStatusDisabled` |
| 5.2 | 🔴 P0 | [knowledge_service.go](server/internal/service/knowledge_service.go) | 290 | Publish 中 embedding 模型名硬编码 `"bge-m3"` | 从 `KnowledgeBase.EmbeddingModel` 读取 |
| 5.3 | 🔴 P0 | [document_parser.go](server/internal/rag/document_parser.go) | 57,67,101 | 3 处 `io.ReadAll` 无大小限制 — 恶意文件可导致 OOM | 使用 `io.LimitReader` 设置上限 (100MB) |
| 5.4 | 🟡 P1 | [chunker.go](server/internal/rag/chunker.go) | 37 | `chunkSize <= 0` 无合法性校验 | 添加 `if chunkSize <= 0 { return nil, error }` |
| 5.5 | 🟡 P1 | [pipeline.go](server/internal/rag/pipeline.go) | 92 | `query_rewrite` 的 history 参数始终传 nil — 上下文消歧从未生效 | 调用方传入最近 N 轮对话历史 |
| 5.6 | 🟡 P1 | [pipeline.go](server/internal/rag/pipeline.go) | 115 | 多路检索时 `rewrittenQuery` 可能与 `routes[0]` 不一致 | 使用实际检索路由进行重排序 |
| 5.7 | 🟡 P1 | [pipeline.go](server/internal/rag/pipeline.go) | 200 | `RerankCount` 应校验 `>= TopK` | 添加参数校验 |
| 5.8 | 🟡 P1 | [rerank.go](server/internal/rag/rerank.go) | 25 | LLM 做重排序 — 每次消耗大量 token，延迟 500ms-2s | 改用交叉编码器 Rerank 模型 |
| 5.9 | 🟡 P1 | [bm25.go](server/internal/rag/bm25.go) | 250 | TTL 过期的锁降级模式脆弱 — RLock→Unlock→Lock→Unlock→RLock 已出过 bug | 提取 `tryRefreshIndex` 辅助方法 |
| 5.10 | 🟡 P1 | [bm25.go](server/internal/rag/bm25.go) | 237 | `ctx` 参数未使用 — 分词/评分操作不支持取消 | 在 `scoreQuery` 循环中检查 `ctx.Done()` |
| 5.11 | 🟡 P1 | [bm25.go](server/internal/rag/bm25.go) | 121 | 知识库超 10 万篇后 map 内存压力大 | 考虑磁盘索引或分片 |
| 5.12 | 🟡 P1 | [multi_route.go](server/internal/rag/multi_route.go) | 55 | LLM 输出子查询的编号清理逻辑脆弱 — 依赖 LLM 固定格式 | 改用正则匹配或让 LLM 输出 JSON |
| 5.13 | 🟡 P1 | [processor.go](server/internal/rag/processor.go) | 154 | `EmbeddingModel` 硬编码为空字符串 | 从知识库或系统配置读取 |
| 5.14 | 🟡 P1 | [embedder.go](server/internal/rag/embedder.go) | 66 | 模型名硬编码为空字符串 | 从配置显式传入 |
| 5.15 | 🟢 P2 | [knowledge_service.go](server/internal/service/knowledge_service.go) | 339 | `Enable()` 中状态检查用 `!= 0` 而非枚举常量 | 同 5.1，统一使用 `model.ArticleStatusDisabled` |

---

## 6. LLM 配置与适配层 (LLM Config & Adapter Layer)

| # | 优先级 | 文件 | 行 | 问题 | 建议修复 |
|---|--------|------|-----|------|----------|
| 6.1 | 🔴 P0 | [llm_client.go](server/internal/adapter/llm_client.go) | 294 | HTTP 调用无重试逻辑 — 429/503 瞬时故障直接失败 | 指数退避 + 可配置最大重试次数 |
| 6.2 | 🔴 P0 | [embedding_client.go](server/internal/adapter/embedding_client.go) | 103 | 与 `llm_client.go` 的 HTTP 样板代码高度重复 | 提取公共 `doRequest` 辅助函数 |
| 6.3 | 🔴 P0 | [embedding_client.go](server/internal/adapter/embedding_client.go) | 105 | 同样无重试逻辑 | 同 6.1 |
| 6.4 | 🔴 P0 | [storage_client.go](server/internal/adapter/storage_client.go) | 72 | 构造函数无超时参数 — MinIO 连接无法限时 | 添加 `timeout` 参数 |
| 6.5 | 🟡 P1 | [llm_config.go](server/internal/handler/llm_config.go) | 55 | `SetLLMClient` Setter 注入脆弱 — 可能在调用后才设置 | 改为构造函数注入 |
| 6.6 | 🟡 P1 | [llm_config.go](server/internal/handler/llm_config.go) | 225 | `TestConnection` 中 `TokensUsed` 当作 `latency` 返回 — 语义错误 | 测量 `time.Since(start)` |
| 6.7 | 🟢 P2 | [llm_client.go](server/internal/adapter/llm_client.go) | 53 | `ChatRequest.Stream` 字段死代码 | 删除或在内部类型中使用 |

---

## 7. 代码架构与规范 (Architecture & Standards)

| # | 优先级 | 文件 | 行 | 问题 | 建议修复 |
|---|--------|------|-----|------|----------|
| 7.1 | 🔴 P0 | [role_service.go](server/internal/service/role_service.go) | 35,90 | 绕过 Repository 直接使用 `s.db` — 破坏三层架构 | 在 RoleRepo 增加 `ExistsByName()` 方法 |
| 7.2 | 🔴 P0 | [knowledge_repo.go](server/internal/repository/knowledge_repo.go) | 206 | 5 个 EmbeddingConfig 方法 + `CreateChunks`/`UpdateChunkSyncStatus` 无 Service 层调用方（死代码） | 接入业务逻辑或删除 |
| 7.3 | 🔴 P0 | [router/helpers.go](server/internal/router/helpers.go) | 29 | `register()`/`registerGroup()` 零调用方（死代码） | 统一使用或删除 |
| 7.4 | 🔴 P0 | [pagination.go](server/internal/repository/pagination.go) | 17 | `Paginate[T]` 泛型函数零调用方 — 7 个 Repo 手写分页（~60 行重复） | 统一迁移或删除文件 |
| 7.5 | 🔴 P0 | [pagination.go](server/internal/repository/pagination.go) | 5 | 所有 Repository 方法缺少 `context.Context` 参数 | 逐步改为 `Method(ctx context.Context, ...)` |
| 7.6 | 🟡 P1 | [chat_service.go](server/internal/service/chat_service.go) | 61 | 构造函数接受 `interface{}` 绕过类型检查 | 直接使用具体接口类型 |
| 7.7 | 🟡 P1 | [knowledge_service.go](server/internal/service/knowledge_service.go) | 68 | 同上 | 同上 |
| 7.8 | 🟡 P1 | [llm_config.go](server/internal/handler/llm_config.go) | 44 | 同上 | 同上 |
| 7.9 | 🟡 P1 | [role_repo.go](server/internal/repository/role_repo.go) | 32 | `GetByID` 返回 `&role, err` 而非 `nil, err` — 与其他 Repo 不一致 | 改为 `nil, err` 模式 |
| 7.10 | 🟡 P1 | [repository/](server/internal/repository/) | - | `gorm.ErrRecordNotFound` 比较方式不一致（`==` vs `errors.Is`） | 统一使用 `errors.Is` |
| 7.11 | 🟡 P1 | [common.go](server/internal/handler/common.go) | 55 | `getCurrentUserID` 返回 0 表示未认证 — 可能记录错误的 `updatedBy=0` | 添加 `exists` 返回值 |
| 7.12 | 🟡 P1 | [llm_config_repo.go](server/internal/repository/llm_config_repo.go) | 13 | `ErrNotFound` 导出哨兵无调用方（死代码） | 删除或接入 |
| 7.13 | 🟡 P1 | [deps.go](server/cmd/deps.go) | 5 | 非标准 `import _` 模式确保依赖 — 应使用 `tools.go` | 改为 `//go:build tools` 约束 |
| 7.14 | 🟢 P2 | [ticket.go](server/internal/dto/response/ticket.go) | 51 | `TicketStatusText` 位于 DTO 包 — DTO 包惯例只放数据结构 | 移至 `model/enums.go` 或 service 包 |
| 7.15 | 🟢 P2 | [llm_config.go](server/internal/dto/request/llm_config.go) | 7 | Create/Update 请求体字段完全相同 | 合并或显式注释说明分化预期 |
| 7.16 | 🟢 P2 | [logger.go](server/internal/middleware/logger.go) | 18 | `Logger()` 是 `LoggerWithWriter(nil)` 的薄封装 | 内联或移除 |
| 7.17 | 🟢 P2 | [config.go](server/internal/config/config.go) | 166,207 | 2 处缩进不一致 | 修复对齐 |

---

## 8. 前端交互与 API 规范 (API Standards)

| # | 优先级 | 文件 | 行 | 问题 | 建议修复 |
|---|--------|------|-----|------|----------|
| 8.1 | 🔴 P0 | [chat.go](server/internal/handler/chat.go) | 239 | SSE JSON 使用字符串拼接而非 `json.Marshal` — 控制字符导致畸形 JSON | 改用 `json.Marshal` 生成每个 token 事件 |
| 8.2 | 🟡 P1 | [chat.go](server/internal/handler/chat.go) | 78 | `SubmitFeedback` 缺少值范围校验 (0/1/2) | 添加 `if body.Feedback < 0 \|\| body.Feedback > 2` |
| 8.3 | 🟡 P1 | [response.go](server/pkg/response/response.go) | 64 | AI/RAG/Storage 错误码未映射为 HTTP 503 | 添加 `case errcode.ErrAIUnavailable... → StatusServiceUnavailable` |
| 8.4 | 🟡 P1 | [knowledge.go](server/internal/handler/knowledge.go) | 93 | `ListKBs` 响应用 `gin.H{"items": kbs}` 包裹 — 与其他列表不一致 | 改为 `response.Success(c, kbs)` |
| 8.5 | 🟢 P2 | [message.go](server/internal/handler/message.go) | 42 | 分页 `pageSize` 上限 50 vs 其他端点 100 — 不一致且未注释 | 文档化原因或对齐 |
| 8.6 | 🟢 P2 | [router.go](server/internal/router/router.go) | 67 | `/api/v1/auth` 前辍同时用于公开和受保护路由 — 容易混淆 | 将受保护路由移至 `/api/v1/auth/me/` |

---

## 9. 部署与配置 (Deployment & Configuration)

| # | 优先级 | 文件 | 行 | 问题 | 建议修复 |
|---|--------|------|-----|------|----------|
| 9.1 | 🔴 P0 | [migrate/main.go](server/cmd/migrate/main.go) | 13 | 迁移工具硬编码数据库密码 `opsmind123` | 从环境变量/命令行参数读取 |
| 9.2 | 🟡 P1 | [main.go](server/cmd/main.go) | 166 | `WriteTimeout=0` 全局禁用写超时 — SSE 需要但影响所有接口 | 仅对 SSE 路由单独设置 |
| 9.3 | 🟡 P1 | [router.go](server/internal/router/router.go) | 49 | `Recovery` 中间件注册顺序不符惯例（应在最外层） | 移至第一个 `r.Use()` |
| 9.4 | 🟢 P2 | [migrate/main.go](server/cmd/migrate/main.go) | 23 | `panic()` 用于错误处理 — 栈追踪不适合 CI/CD 日志 | 改用 `log.Fatalf()` |

---

## 10. 业务逻辑补强 (Business Logic Enhancement)

| # | 优先级 | 文件 | 行 | 问题 | 建议修复 |
|---|--------|------|-----|------|----------|
| 10.1 | 🟡 P1 | [chat_service.go](server/internal/service/chat_service.go) | 111 | RAG 管道 TopK 和各步骤开关硬编码 | 从配置/AI_DEFAULT_TOP_K 读取 |
| 10.2 | 🟡 P1 | [chat_service.go](server/internal/service/chat_service.go) | 136 | System prompt 硬编码 — 不同知识库可能需要不同角色设定 | 支持按知识库配置 prompt 模板 |
| 10.3 | 🟡 P1 | [chat_service.go](server/internal/service/chat_service.go) | 140 | 仅取前 3 个 chunk 注入 LLM，浪费了 TopK=5 的后 2 个结果 | 调整 `maxContextChunks` 或 TopK 对齐 |
| 10.4 | 🟡 P1 | [chat_service.go](server/internal/service/chat_service.go) | 192 | 置信度计算过于粗糙（仅按命中 chunk 数 × 0.3） | 结合检索分数、重排序分数综合计算 |
| 10.5 | 🟡 P1 | [role_service.go](server/internal/service/role_service.go) | 119 | `Delete` 未检查关联用户 — 留下孤儿 `user_roles` | 先 count 关联用户，>0 则拒绝 |
| 10.6 | 🟢 P2 | [auth_service.go](server/internal/service/auth_service.go) | 205 | `GetRoleMenus` 在循环内调用 — 用户有 N 个角色则 N 次查询 | 增加 `BatchGetRoleMenus(roleIDs)` |
| 10.7 | 🟢 P2 | [router/helpers.go](server/internal/router/helpers.go) | 15 | `placeholder()` 返回英文 "Not Implemented" — 与项目中文规范不一致 | 改为中文或使用 `errcode` 常量 |

---

## 统计摘要

| 优先级 | 数量 | 占⽐ |
|--------|------|------|
| 🔴 P0 — 生产隐患 | 28 | 32% |
| 🟡 P1 — 架构债务 | 38 | 44% |
| 🟢 P2 — 代码优化 | 21 | 24% |
| **合计** | **87** | **100%** |

### 按领域分布

```
认证与安全          8  ████████
数据一致性与事务     9  █████████
错误处理与日志      14  ██████████████
申告管理            5  █████
知识库与 RAG        15  ███████████████
LLM 配置与适配层     7  ███████
代码架构与规范      17  █████████████████
API 规范            6  ██████
部署与配置          4  ████
业务逻辑补强        7  ███████
```

### 建议修复路线

1. **第一轮（P0，1-2 周）**：修复安全漏洞 + 数据完整性 + 静默错误
2. **第二轮（P1，2-4 周）**：消除架构债务 + API 规范化
3. **第三轮（P2，持续）**：代码风格统一 + 死代码清理

---

> 📅 生成日期: 2026-06-12  
> 🔍 基于: server/ 全量代码审查（163 文件 / ~25,200 行）  
> 📝 所有 TODO 已同步写入源代码，可用 `grep -rn "TODO:" server/` 定位
