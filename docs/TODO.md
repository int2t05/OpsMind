# OpsMind 代码改进清单

> 基于 2026-06-15 全量代码再审计（前后端 130+ 源文件 + 文档交叉校验），按业务流程重组。
> 覆盖 [diagrams/](diagrams/) 中 10 份业务流程图对应的全部模块 + [API/](API/) 9 份接口文档。
> 优先级：🔴 P0 生产隐患 / 🟡 P1 架构债务 / 🟢 P2 优化改进
> ⭐ 标记为 2026-06-15 审计新发现的问题。
> 📝 标记为文档一致性缺陷（代码实现与 API 文档/PRD/TECH.md 不一致）。

---
## 1. 认证与授权

> 对应图：[auth-flow.md](diagrams/auth-flow.md) — 登录 → JWT 双令牌 → 中间件链 → RBAC 权限校验
> 对应文档：[API/auth.md](API/auth.md)

### JWT 令牌安全

- ✅ [pkg/jwt/jwt.go](/server/pkg/jwt/jwt.go) — ~~alg 限制不严格~~ — 已改用 `WithValidMethods([]string{"HS256"})` 严格限制
- ✅ [pkg/jwt/jwt.go](/server/pkg/jwt/jwt.go) — ~~secret 为空时静默~~ — `generateToken` 已增加 `secret == ""` 显式校验，与 `ParseToken` 形成纵深防御
- ✅ [service/auth_service.go](/server/internal/service/auth_service.go) — ~~jwtSecret 硬编码默认值~~ — 已移除，改为构造注入 `config.JWT.Secret`
- ✅ [service/auth_service.go](/server/internal/service/auth_service.go) — ~~token 有效期写死 2h/7d~~ — 已改为读取 `s.jwtCfg.AccessExpire` / `s.jwtCfg.RefreshExpire`
- ✅ [service/auth_service.go](/server/internal/service/auth_service.go) — ~~RefreshToken 需校验 TokenType~~ — 已增加 `claims.TokenType != "refresh"` 校验
- ✅ [service/auth_service.go](/server/internal/service/auth_service.go) — ~~增加登录失败限流~~ — 已实现内存限流器（5 次/15 分钟滑动窗口）
- ✅ [middleware/auth.go](/server/internal/middleware/auth.go) — ~~JWT 不检查用户冻结~~ — `JWTAuth` 新增 `db` 参数，解析后查询用户状态（冻结/存在性）
- ✅ [middleware/auth.go](/server/internal/middleware/auth.go) — ~~secret 为空时中间件静默~~ — 增加 secret 空值检查，拒绝所有请求并返回明确错误
- ✅ [pkg/jwt/jwt.go](/server/pkg/jwt/jwt.go) — ~~缺少标准声明~~ — 已增加 Issuer/Subject/ID(jti)/IssuedAt
- ✅ [middleware/auth.go](/server/internal/middleware/auth.go) — ~~TokenType 校验缺失~~ — 已增加 `claims.TokenType != "access"` 校验（refresh token 不能访问 API）
- ✅ [service/auth_service.go](/server/internal/service/auth_service.go) — ~~限流器为内存实现~~ — 已存在，重启后重置为已知行为，暂保持内存实现

### 中间件安全

- ✅ [middleware/cors.go](/server/internal/middleware/cors.go) — ~~DNS 重绑定风险~~ — release 模式强制配置 AllowOrigins + localhost 告警
- ✅ [middleware/request_id.go](/server/internal/middleware/request_id.go) — ~~X-Request-ID 注入~~ — 已有长度限制(128) + 字符集校验(regex)
- ✅ [middleware/rbac.go](/server/internal/middleware/rbac.go) — ~~通配权限~~ — 已支持 `*` 全匹配 + `prefix:*` 前缀通配
- ✅ [middleware/rbac.go](/server/internal/middleware/rbac.go) — ~~空 permissions~~ — 注册时 slog.Warn 告警 + 保持不变的安全默认值

### 密码策略

- ✅ [pkg/hash/hash.go](/server/pkg/hash/hash.go) — ~~字节计数~~ — 改用 `utf8.RuneCountInString`（非 ASCII 字符各计 1 位）
- ✅ [pkg/hash/hash.go](/server/pkg/hash/hash.go) — ~~检测不一致~~ — 大小写/数字统一使用 `unicode.IsLower/IsUpper/IsDigit`
- ✅ [pkg/hash/hash.go](/server/pkg/hash/hash.go) — bcrypt cost=10 硬编码，应可配置以应对硬件升级

### 路由一致性

- ✅ [router/router.go](/server/internal/router/router.go) — ~~Auth 路由路径与文档不一致~~ — 已改为 `/api/v1/auth/me` 子路由组，与 [API/auth.md](API/auth.md) 文档一致。前端 [api/auth.ts](/web/src/api/auth.ts) 同步更新。

---
## 2. 智能问答 RAG

> 对应图：[chat-rag-flow.md](diagrams/chat-rag-flow.md) — SSE 流式 → Pipeline 执行 → 多路检索 → 混合融合 → 重排序 → LLM 生成
> 对应文档：[API/chat.md](API/chat.md)
> 深度审计日期：2026-06-16 — 覆盖 pipeline / query_rewrite / multi_route / hybrid / bm25 / rerank / chunker / embedder / processor / document_parser / retriever / embedding_client / llm_service 共 13 个文件

### SSE 流式输出（核心路径）

- ✅ [handler/chat.go](/server/internal/handler/chat.go) + [service/llm_service.go](/server/internal/service/llm_service.go) — ~~**流式答案与存储答案不一致**~~ — 已通过 LLMService 重构为单次 LLM 调用。
- ✅ [handler/chat.go](/server/internal/handler/chat.go) — ~~SSE 流中 LLM 错误被静默吞掉~~ — StreamEvent 支持 `type: "error"` 事件。
- ✅ [handler/chat.go](/server/internal/handler/chat.go) — ~~模拟流式先完整生成再 SSE 输出~~ — 已改为真实 `ChatCompletionStream`。
- ✅ [service/chat_service.go](/server/internal/service/chat_service.go) — ~~`FinalAnswer` 和流式各一次 LLM 调用~~ — `SyncChat`/`StreamChat` 各自一次。
- ✅ [service/llm_service.go](/server/internal/service/llm_service.go) — ~~RAG 步骤事件未实时流式发送~~ — `executeRAG` 已接入 `onStep rag.StepCallback`，`StreamChat` 实时转换 `rag.StepEvent` 为 SSE step 事件。
- ✅ [service/llm_service.go](/server/internal/service/llm_service.go) — ~~多轮对话历史无长度截断~~ — `buildMessages` 已实现滑动窗口截断，上限由 `maxHistoryMessages`（默认 10，环境变量 `OPSMIND_AI_MAX_HISTORY_MESSAGES`）控制。

### RAG 管道

- 🔴 [rag/pipeline.go](/server/internal/rag/pipeline.go) — `llmClient` 为 nil 时，QueryRewrite/MultiRoute/Rerank 会 panic（nil 指针解引用）
- 🔴⭐ [rag/rerank.go](/server/internal/rag/rerank.go) — 重排序 prompt 无长度截断：候选 chunk 全量拼接，可超出 LLM 上下文窗口致 400 错误
- 🟡 [rag/pipeline.go](/server/internal/rag/pipeline.go) — QueryRewrite 的 history 始终为 nil，上下文消歧功能未生效
- 🟡 [rag/pipeline.go](/server/internal/rag/pipeline.go) — 重排序候选过多时应提前截断：Rerank 在重排**前**未按 `RerankCount` 截断候选列表；事后检查（第 207 行）为时已晚
- 🟡 [rag/pipeline.go](/server/internal/rag/pipeline.go) — Execute 缺少入口 opts 规范化：TopK=0 → LIMIT 0，RouteCount=0 → 多路检索跳过，与「零值使用默认」注释不一致
- 🟡 [rag/query_rewrite.go](/server/internal/rag/query_rewrite.go) — llm 为 nil 时应降级返回原 query，而非 panic
- 🟡 [rag/multi_route.go](/server/internal/rag/multi_route.go) — LLM 输出子查询的清洗逻辑脆弱（`TrimLeft` 依赖特定前缀格式）
- 🟡⭐ [rag/multi_route.go](/server/internal/rag/multi_route.go) — k（子查询数量）无上限，k=100 可致百倍检索放大
- 🟡 [rag/hybrid.go](/server/internal/rag/hybrid.go) — 单路结果直接返回时未按 topK 截断
- 🟡 [service/chat_service.go](/server/internal/service/chat_service.go) — 前端 RAGOptions 完全被忽略：`CreateChatSession` 硬编码全部 `true`，高级设置表单无效。📝 [PRD.md §3.1](PRD.md) 记载「每个步骤可独立开关（`rag_options`）」但代码未实现。
- 🟡 [rag/types.go](/server/internal/rag/types.go) — RAGOptions 使用裸 `bool`，零值问题致空 JSON `{}` 全部禁用，与文档默认「全部启用」矛盾
- 🟡 [rag/types.go](/server/internal/rag/types.go) — `RetrievalResult.Score` 文档注释称「归一化到 [0,1]」，但 BM25 分数无边界，RRF 融合后可 >1。注释与实现不一致。
- 🟢 [rag/rerank.go](/server/internal/rag/rerank.go) — `_ = i` 调试残留（第 79 行），无实际用途
- 🟢 [rag/pipeline.go](/server/internal/rag/pipeline.go) — 步骤 ID 拼写不一致：`vector_retrieve`（pipeline）vs `vector_retrieval`（retriever 注释/docs）
- 🟢 [service/chat_service.go](/server/internal/service/chat_service.go) — `pipeline` 字段为死存储：ChatService 通过 LLMService 间接使用 pipeline，自身不再直接调用 `pipeline.Execute`

### BM25 检索

- 🟡 [rag/bm25.go](/server/internal/rag/bm25.go) — 超 10 万篇后内存 map 压力大，应考虑分片或磁盘索引
- 🟡 [rag/bm25.go](/server/internal/rag/bm25.go) — BuildIndex 同步分词，请求路径调用造成长尾延迟
- 🟡 [rag/bm25.go](/server/internal/rag/bm25.go) — LoadDict 错误被丢弃，分词器静默回退到字符级切分
- 🟡 [rag/bm25.go](/server/internal/rag/bm25.go) — 缺少 token 过滤（停用词/标点/过短/空白），低质量 token 污染评分
- 🟡 [rag/bm25.go](/server/internal/rag/bm25.go) — topK <= 0 时返回空结果，管线误判为「无检索结果」

### Embedding 与向量检索

- 🟡 [rag/embedder.go](/server/internal/rag/embedder.go) — 部分批次失败静默跳过，破坏 `vectors[i]` 与 `texts[i]` 的索引对应关系。调用方检测长度不匹配后全量失败，浪费之前成功的批次。
- 🟡 [rag/embedder.go](/server/internal/rag/embedder.go) — 批次失败仅计数不记错误信息，调试困难
- 🟡 [rag/embedder.go](/server/internal/rag/embedder.go) — 不校验各批次的 embedding 维度一致性，若中途模型变更可致混维向量写入 pgvector 报错
- 🟡 [rag/embedder.go](/server/internal/rag/embedder.go) — `client` 为 nil 时不校验——构造函数无防护，首次 `Embed` 调用才 panic
- 🟡 [adapter/vector_store.go](/server/internal/adapter/vector_store.go) — `BatchInsert` 无跨 chunk 维度一致性校验，维度不匹配时 pgvector 报错不友好
- 🟡 [adapter/vector_store.go](/server/internal/adapter/vector_store.go) — `CosineSearch` 无 topK 范围和空 embedding 防护
- 🟡 [adapter/vector_store.go](/server/internal/adapter/vector_store.go) — NaN/Inf 用 `> 1e30` 检测而非 `math.IsInf`；静默替换为 0.0 污染向量空间

### 文档处理（chunker / parser / processor）

- 🟡 [rag/chunker.go](/server/internal/rag/chunker.go) — `mergeSplits` 未实现 chunkOverlap：仅 `splitByRunes` 支持 overlap，段落/句子级切分后相邻 chunk 无重叠
- 🟡 [rag/chunker.go](/server/internal/rag/chunker.go) — 分块前未做文本归一化（全角→半角、多余空白合并），影响 BM25 分词品质和 embedding 稳定性
- 🟡 [rag/document_parser.go](/server/internal/rag/document_parser.go) — PDF 二进制字节被 `string(b)` 转换破坏：非 UTF-8 字节在 Go string 往返中损坏。应使用 `bytes.NewReader(b)`。
- 🟡 [rag/document_parser.go](/server/internal/rag/document_parser.go) — DOCX 解析只读 `<w:t>` 元素，丢弃表格/超链接/页眉页脚
- 🟡 [rag/document_parser.go](/server/internal/rag/document_parser.go) — DOCX XML 命名空间硬编码，非标准 Office 软件生成的 docx 可能返回空内容（无报错）
- 🟡 [rag/document_parser.go](/server/internal/rag/document_parser.go) — PDF 单页解析失败静默跳过，用户收到部分内容无告警
- 🔴 [rag/processor.go](/server/internal/rag/processor.go) — Stop 后 Submit 会 panic（向已关闭 channel 发送）；Stop 非幂等（两次 close 同一 channel panic）
- 🔴 [rag/processor.go](/server/internal/rag/processor.go) — embedding 模型硬编码为空字符串 `""`，应从 KB 配置读取
- 🟡 [rag/processor.go](/server/internal/rag/processor.go) — worker 无 panic recovery：processTask 崩溃导致 goroutine 退出，poolSize 永久减小
- 🟡 [rag/processor.go](/server/internal/rag/processor.go) — 所有任务共用 `p.ctx`，单个任务无独立超时——卡住的下载/embedding 永久占用 worker
- 🟢 [rag/processor.go](/server/internal/rag/processor.go) — `updateStatus("indexing")` 调用两次，前端可能收到重复进度事件

### 置信度与数据完整性

- 🟡 [service/chat_service.go](/server/internal/service/chat_service.go) — 置信度算法粗糙：直接取 max(chunk.Score)，但 BM25 和 RRF 分数量纲不同
- ✅ [service/chat_service.go](/server/internal/service/chat_service.go) — ~~Sources（检索引用）未写入~~ — 已修复。
- 🔴 [service/chat_service.go](/server/internal/service/chat_service.go) — `CreateChatSession` 用 `context.Background` 不传播请求取消
- 🟡 [service/chat_service.go](/server/internal/service/chat_service.go) — `GetChatDetail` 未校验 `session.UserID` 归属
- ✅ [repository/chat_repo.go](/server/internal/repository/chat_repo.go) — ~~`CreateBatch` 零调用方~~ — 已修复。
- 🟢 [model/chat.go](/server/internal/model/chat.go) — `ChatMessage.SessionID` 缺少索引。需 `CREATE INDEX idx_chat_messages_session ON chat_messages(session_id)`。

### 文档一致性

- 📝⭐ [API/chat.md](API/chat.md) — SSE `done` 事件文档记载含 `metadata.pipeline.steps[]` 及 `total_duration_ms`，但 `CreateChatSession` 未填充 `Pipeline` 字段。
- 🟢 [rag/pipeline.go](/server/internal/rag/pipeline.go) — StepMetric 文档注释中步骤 ID `hybrid_retrieve` 不存在于代码中（实际拆分为 `vector_retrieve` + `bm25_retrieve` + `hybrid_fuse`）

### 输入校验

- 🟢 [dto/request/chat.go](/server/internal/dto/request/chat.go) — Question 字段缺少 `max` 绑定，可发送 MB 级文本直入管道和 LLM
- 🟢 [dto/request/chat.go](/server/internal/dto/request/chat.go) — RAGOptions 缺少 `route_count`/`rerank_count` 字段，与 `rag.RAGOptions` 类型不同步

### 服务层集成

- 🟡 [service/chat_service.go](/server/internal/service/chat_service.go) — `SubmitFeedback` 校验 feedback 值为 0/1/2 的规则应在 Service 层而非仅 Handler 层
- 🟡 [cmd/main.go](/server/cmd/main.go) — config.yaml 的 LLM 模型未同步到 `LLMConfigManager` 作为 fallback。DB 无配置时 `getModelConfig()` 回退到 `model="default"`，忽略 `cfg.LLM.Model`
- 🟢 [cmd/main.go](/server/cmd/main.go) — BM25 索引 TTL 30 分钟，新发布的知识文章在 BM25 检索中最多延迟 30 分钟可见
- 🟢 [cmd/main.go](/server/cmd/main.go) — Processor goroutine pool 仅 2 个 worker，大量文档上传时排队

---
## 3. 知识库与文档管理

> 对应图：[knowledge-publish-flow.md](diagrams/knowledge-publish-flow.md) + [document-upload-flow.md](diagrams/document-upload-flow.md) — 文章生命周期 → 分块 → Embedding → pgvector；文档上传 → 解析 → 异步处理
> 对应文档：[API/knowledge.md](API/knowledge.md)

### 知识发布管道（核心路径）

- 🔴 [service/knowledge_service.go](/server/internal/service/knowledge_service.go) — `DeleteByArticle` + `BatchInsert` 非原子：删除旧向量成功但新向量写入失败 → 文章向量永久丢失
- 🔴 [service/knowledge_service.go](/server/internal/service/knowledge_service.go) — Publish 使用 `context.Background` 忽略请求取消
- 🔴 [service/knowledge_service.go](/server/internal/service/knowledge_service.go) — 管道未初始化（`pipeline == nil`）时应映射为 `ErrRAGUnavailable`，而非静默跳过
- 🔴 [rag/processor.go](/server/internal/rag/processor.go) — embedding 模型硬编码为空字符串 `""`，应从 KB 配置读取

### 文章状态机

- 📝⭐ [API/knowledge.md](API/knowledge.md) vs [model/enums.go](/server/internal/model/enums.go) — **文章状态编号不一致**：API 文档定义生命周期为「草稿(1)→已提交审核(2)→审核通过(3)→已发布(4)→已停用(5)/驳回(6)」，代码定义 Disabled=0、Draft=1、Reviewing=2、Approved=3、Published=4、Rejected=5。Disabled 在文档中编号 5（与 Closed 混淆），代码中编号 0。
- 🟡 [service/knowledge_service.go](/server/internal/service/knowledge_service.go) — Enable 仅重置 status 为 Draft，不重新执行分块/Embedding/pgvector 写入。📝 [API/knowledge.md](API/knowledge.md) 记载「启用后需重新执行分块→embedding→pgvector 写入（因为停用时向量已删除）」。
- 🟡 [service/knowledge_service.go](/server/internal/service/knowledge_service.go) — Disable 未校验当前状态是否为已发布（Draft 可直接 Disable）。📝 [API/knowledge.md](API/knowledge.md) 记载「状态：已发布(4) → 已停用(5)」。
- 🟡 [service/knowledge_service.go](/server/internal/service/knowledge_service.go) — 文章 `status`（审核流程）和 `process_status`（文档处理）两个状态机概念混淆
- 🟡 [service/knowledge_service.go](/server/internal/service/knowledge_service.go) — Publish/Disable 应接收请求 ctx 而非 `context.Background`
- 🟡 [repository/knowledge_repo.go](/server/internal/repository/knowledge_repo.go) — `UpdateArticleStatus` 不检查 `RowsAffected`，更新不存在的 ID 静默成功

### 文档上传与异步处理

- 🔴 [rag/processor.go](/server/internal/rag/processor.go) — Stop 后 Submit 会 panic（向已关闭 channel 发送）；Stop 非幂等（两次 close 同一 channel panic）
- 🔴 [rag/embedder.go](/server/internal/rag/embedder.go) — 批次失败静默跳过，丢失 chunk 与向量的对应关系；部分批次失败时全部重试浪费之前成功的批次
- 🔴 [adapter/storage_client.go](/server/internal/adapter/storage_client.go) — `ensureBucket` 失败只 warn 继续启动，后续上传操作失败时错误信息令人困惑
- 🟡 [rag/document_parser.go](/server/internal/rag/document_parser.go) — `io.LimitReader` 到 100MB 上限不报错，静默截断文档内容
- 🟡 [service/knowledge_service.go](/server/internal/service/knowledge_service.go) — `RetryDocument` 未校验当前是否处于 failed 状态，可对已成功文档重复入队
- 🟡 [handler/knowledge.go](/server/internal/handler/knowledge.go) — 文档上传应结合 MIME sniffing 判断文件类型，而非仅信任扩展名
- 🟡 [handler/knowledge.go](/server/internal/handler/knowledge.go) — `GetDocumentStatus` 未校验 `kb_id` 与 article.KBID 一致，URL 资源层级校验缺失
- 📝⭐ [handler/knowledge.go](/server/internal/handler/knowledge.go) — **上传 API 字段名与文档不一致**：[API/knowledge.md](API/knowledge.md) 指定 multipart 字段名 `files`（复数，多文件），代码读取 `c.FormFile("file")`（单数，仅单文件）。响应形状也不一致：文档返回 `documents` 数组含 `file_size`，代码返回扁平对象含 `article_id`/`filename`/`kb_id`。

### 文档响应形状对齐

- 🟡⭐ [dto/response/knowledge.go](/server/internal/dto/response/knowledge.go) — KB 响应缺少 `llm_config_id`/`article_count` 字段（[API/knowledge.md](API/knowledge.md) 文档中有，DTO 未实现）。需补充字段并修改 Service 层 `ListKBs` 填充。
- 🟡⭐ [dto/request/knowledge.go](/server/internal/dto/request/knowledge.go) — 创建 KB 请求缺少 `llm_config_id` 字段（[API/knowledge.md](API/knowledge.md) 记为可选参数）。需补充 DTO + Service 关联逻辑。
- 🟡⭐ [dto/response/knowledge.go](/server/internal/dto/response/knowledge.go) — Article 响应缺少 `source_type_text`/`created_by_name`/`published_by_name` 字段（[API/knowledge.md](API/knowledge.md) 文档中有，需 Service 层 join 查询填充）。
- 🟡⭐ [handler/knowledge.go](/server/internal/handler/knowledge.go) — `GetDocumentStatus` 响应缺少 `file_name`/`process_error`/`progress` 对象（[API/knowledge.md](API/knowledge.md) 文档详细但代码仅返回 `article_id` + `process_status` 字符串）。
- 🟡⭐ [service/knowledge_service.go](/server/internal/service/knowledge_service.go) — `RetryDocument` 返回 message 与 [API/knowledge.md](API/knowledge.md) 不一致（"重试已提交" vs "已重新加入处理队列"）。
- 🟢⭐ [dto/request/chat.go](/server/internal/dto/request/chat.go) — RAGOptions 缺少 `route_count`/`rerank_count` 字段，与 `rag.RAGOptions` 类型不同步，前端无法传递这两个参数。

### 内容质量

- 🟢 [rag/chunker.go](/server/internal/rag/chunker.go) — `mergeSplits` 未实现 chunkOverlap（仅 `splitByRunes` 硬切分支持 overlap）
- 🟢 [rag/chunker.go](/server/internal/rag/chunker.go) — 分块前应做文本归一化（全角→半角、多余空白合并）
- 🟢 [rag/document_parser.go](/server/internal/rag/document_parser.go) — DOCX 只读 `w:t` 元素，丢失表格和超链接内容
- 🟢 [service/knowledge_service.go](/server/internal/service/knowledge_service.go) — tags 应 trim/去重/限制数量
- 🟢 [dto/response/knowledge.go](/server/internal/dto/response/knowledge.go) — 门户端不应返回 `RAGWorkspaceSlug`/`EmbeddingModel` 等内部基础设施字段

### 重排序

- 🟡 [rag/rerank.go](/server/internal/rag/rerank.go) — 用 LLM 做重排序，每次消耗 token 且延迟高；应考虑 Cross-Encoder 模型
- 🟢⭐ [rag/rerank.go](/server/internal/rag/rerank.go) — `_ = i` 无操作语句，调试残留

---
## 4. 申告管理

> 对应图：[ticket-lifecycle.md](diagrams/ticket-lifecycle.md) + [ticket-state-machine.md](diagrams/ticket-state-machine.md) — 创建→处理→补充→解决→关闭 + AutoClose
> 对应文档：[API/tickets.md](API/tickets.md)

### 数据完整性与事务

- 🔴 [service/ticket_service.go](/server/internal/service/ticket_service.go) — `supplement_count` 并发竞态：应在 Repository 层用 `UPDATE ... WHERE supplement_count < 3` 原子操作
- 🔴 [service/ticket_service.go](/server/internal/service/ticket_service.go) — `UpdateStatus` + `CreateRecord` 不在同一事务（应先获取 Ticket 再在同一 TxManager 事务中 Update + CreateRecord）
- 🔴 [repository/ticket_repo.go](/server/internal/repository/ticket_repo.go) — AutoClose 批量 UPDATE 不创建 TicketRecord（应在 Service 层 TxManager 事务内为每个 ID 创建 record）
- 🔴 [repository/ticket_repo.go](/server/internal/repository/ticket_repo.go) — `SELECT ids + UPDATE` 不是原子操作，AutoClose 有 TOCTOU 竞态
- 🟡 [repository/ticket_repo.go](/server/internal/repository/ticket_repo.go) — `UpdateStatus` 应返回 `RowsAffected`，否则更新不存在的 ID 静默成功

### 状态机

- 🔴 [service/ticket_service.go](/server/internal/service/ticket_service.go) — 状态机和 action 使用裸数字（1,2,3,4,5）而非常量 `TicketStatusXxx`，可读性和可维护性差
- 🟡 [service/ticket_service.go](/server/internal/service/ticket_service.go) — close 操作是否允许关闭已解决（Resolved）状态需在代码中明确语义
- 🟡 [service/ticket_service.go](/server/internal/service/ticket_service.go) — `request_info` 后应同步创建站内消息通知（当前图中有但代码中需验证）

### 安全与校验

- 🔴 [service/ticket_service.go](/server/internal/service/ticket_service.go) — `ticket_no` 使用 `time.Now().UnixNano()%1000000` + 随机数，高并发下碰撞风险（纳秒取模后仅 6 位）+ 随机数种子上未显示初始化
- 🔴 [service/ticket_service.go](/server/internal/service/ticket_service.go) — 门户端 `GetDetail` 不校验 `ticket.UserID == 当前用户`，任意用户可查看他人申告
- 🟢 [model/ticket.go](/server/internal/model/ticket.go) — `contact_phone` 长度假设 11 位中国手机号，国际化和扩展号码（分机）不友好
- 🟢 [dto/request/ticket.go](/server/internal/dto/request/ticket.go) — `ChatContext` 应使用结构化对象而非 `string`

### 文档一致性

- 📝⭐ [PRD.md §3.3](PRD.md) vs [model/enums.go](/server/internal/model/enums.go) — **影响范围取值不一致**：PRD 文档记载 impact_scope 取值 0/1/2/3，enums.go 实际定义 ImpactPersonal=1、ImpactDept=2、ImpactCompany=3（无 0 值）。需统一定义并更新文档。

---
## 5. 用户与角色管理

> 对应图：[user-rbac-flow.md](diagrams/user-rbac-flow.md) — 用户 CRUD + 角色权限 + 菜单树
> 对应文档：[API/users.md](API/users.md) + [API/roles.md](API/roles.md)

### 数据安全

- 🔴⭐ [repository/role_repo.go](/server/internal/repository/role_repo.go) — **GORM 零值陷阱**：`Delete(&Role{}, 0)` 删除全部角色。需加 `if id <= 0` 守卫或改用 `Where("id = ?", id).Delete()`
- 🔴 [service/user_service.go](/server/internal/service/user_service.go) — `AssignRoles` 内层再开事务（嵌套事务风险：外层事务回滚不包含 AssignRoles 的变更）
- 🟡⭐ [service/user_service.go](/server/internal/service/user_service.go) — **丢失更新竞态**：`Update` 用 `GetByID` → 修改内存 → `Save`，并发 `ChangePassword` 的密码哈希会被 `Save` 覆盖
- 🟡⭐ [service/role_service.go](/server/internal/service/role_service.go) — **TOCTOU 竞态**：`Delete` 先 `CountUsersByRole` 检查再 `Delete`，并发 `AssignRoles` 可在检查后分配用户到此角色
- 🟡⭐ [repository/user_repo.go](/server/internal/repository/user_repo.go) — **TOCTOU 竞态**：`UpdateRoleMenus` 删除-插入非原子，并发更新同角色菜单可致部分关联永久丢失
- 🟡 [service/role_service.go](/server/internal/service/role_service.go) — 禁止删除系统内置角色（当前无 built-in 标记，无删除保护）
- 🟡 [service/user_service.go](/server/internal/service/user_service.go) — 无「最后一个管理员」保护：冻结/修改最后的管理员角色会使系统无法管理
- 🟡⭐ [service/user_service.go](/server/internal/service/user_service.go) — 无自我保护：用户可以冻结自己、降级自己的角色权限
- 🟡 [repository/user_repo.go](/server/internal/repository/user_repo.go) — `AssignRoles` 不校验 roleID 是否存在、不排重、不过滤 ≤0 的非法值

### 查询性能

- 🟡 [service/user_service.go](/server/internal/service/user_service.go) — 列表 N+1 查询角色：每个用户额外一次 DB 查询获取角色名
- 🟢 [repository/user_repo.go](/server/internal/repository/user_repo.go) — `ExistsBy*` 用 `COUNT(*)` 而非 `SELECT 1 LIMIT 1`，大表效率低

### 权限与角色

- 🔴 [service/role_service.go](/server/internal/service/role_service.go) — `.Count()` 错误被丢弃，绕过 Repository 直调 DB（分层违规）
- 🟡 [service/role_service.go](/server/internal/service/role_service.go) — 角色权限无白名单校验：任意字符串可作为权限写入 JSONB
- 🟡 [repository/role_repo.go](/server/internal/repository/role_repo.go) — `Delete` 不检查 `RowsAffected`
- 🟡 [repository/user_repo.go](/server/internal/repository/user_repo.go) — 角色权限 JSON 解析失败静默跳过 `continue`，数据损坏被掩盖

### 输入校验

- 🟢 [service/user_service.go](/server/internal/service/user_service.go) — 缺少对 phone/email/realName 的格式/trim 校验
- 🟢 [model/user.go](/server/internal/model/user.go) — phone 字段缺少唯一索引
- 🟢 [model/role.go](/server/internal/model/role.go) — Role 缺少 `is_system` 不可变标记字段

---
## 6. LLM 配置与适配层

> 对应图：[llm-config-flow.md](diagrams/llm-config-flow.md) — CRUD + TestConnection + atomic.Value 热替换 + API Key 脱敏
> 对应文档：[API/llm-config.md](API/llm-config.md)

### LLM 客户端

- 🔴⭐ [adapter/llm_client.go](/server/internal/adapter/llm_client.go) — **重试机制完全失效**：`doHTTPRequest` 将 429/503 包装为普通 `fmt.Errorf` 而非 `*retryableError`，`isRetryable()` 永远返回 false。LLM 同步重试和 Embedding 重试均不工作。
- 🔴 [adapter/llm_client.go](/server/internal/adapter/llm_client.go) — 校验 baseURL 非空且合法（空字符串产生误导性 "unsupported protocol scheme" 错误）
- 🔴 [adapter/llm_client.go](/server/internal/adapter/llm_client.go) — `req.Model` 为空时应返回参数错误（而非将空字符串发给 API）
- 🔴 [adapter/llm_client.go](/server/internal/adapter/llm_client.go) — `bufio.Scanner` 默认 64KB 上限，大 SSE `data:` 行会触发 `ErrTooLong` 静默截断流
- 🔴 [adapter/llm_client.go](/server/internal/adapter/llm_client.go) — 流式请求无 429/503 重试（直接 return status code error）

### LLM 配置管理

- 🔴⭐ [service/llm_config_service.go](/server/internal/service/llm_config_service.go) — **事务内使用 repo 原始 DB 而非 tx**：`ClearDefault()` 和 `Create()` 在 `db.Transaction()` 内调但用 `s.repo` 的 `*gorm.DB`，操作在事务外执行，原子性为空壳
- 🔴⭐ [handler/llm_config.go](/server/internal/handler/llm_config.go) — **TestConnection 测试的是全局默认客户端而非被测配置**：始终用 `h.llmClient`，选择 `:id` 配置无意义——功能完全失效。📝 [API/llm-config.md](API/llm-config.md) 记载测试「指定配置的连接」，但代码实现与之完全不符。
- 🔴⭐ [handler/llm_config.go](/server/internal/handler/llm_config.go) — **UpdateConfig 会清空 api_key**：若请求不传 `api_key`，零值 `""` 覆盖数据库中已存储的密钥。📝 [API/llm-config.md](API/llm-config.md) 记载「api_key 不传时保留原有密钥（不传 ≠ 清空）」，代码行为与文档相反。
- 🔴 [service/llm_config_service.go](/server/internal/service/llm_config_service.go) — `atomic.Store` 直接存指针，未复制 cfg；调用方修改原对象 → 并发读看到中间态（数据竞态）
- 🔴 [service/llm_config_service.go](/server/internal/service/llm_config_service.go) — 构造函数不应 panic：mock 类型不匹配会直接崩溃进程，应返回 error
- 🔴⭐ [model/llm_config.go](/server/internal/model/llm_config.go) — **API Key 明文存储**：数据库泄露 = 所有 LLM 提供商密钥暴露。需 AES-256 加密存储。📝 [API/llm-config.md](API/llm-config.md) 记载「api_key 在数据库中以 AES-256 加密存储」，但代码完全未实现加密。
- 🔴⭐ [repository/llm_config_repo.go](/server/internal/repository/llm_config_repo.go) — **缺少 `is_default` 部分唯一索引**：并发创建默认配置可产生多条 `is_default=true` 记录
- 🟡 [service/llm_config_service.go](/server/internal/service/llm_config_service.go) — 默认配置切换后未重建 LLM/Embedding 客户端（仅替换了配置值，已初始化的 HTTP 客户端仍指向旧 Base URL）
- 🟡 [service/llm_config_service.go](/server/internal/service/llm_config_service.go) — 缺少输入校验：providerType 白名单、baseURL 格式、maxTokens/vectorDimension 范围
- 🟡 [service/llm_config_service.go](/server/internal/service/llm_config_service.go) — 删除配置前未检查 FK 引用（knowledge_bases.llm_config_id）
- 🟡 [handler/llm_config.go](/server/internal/handler/llm_config.go) — CreateConfig 默认值 8192/1024 应在 Service 层而非 Handler 层
- 📝⭐ [handler/llm_config.go:216](/server/internal/handler/llm_config.go) — **TestConnection 错误码与 API 文档不一致**：[API/llm-config.md](API/llm-config.md) 记载失败时 `code=0, data.success=false`，代码返回 `code=20001`（ErrAIUnavailable）。行为与文档约定相反。

### 适配层通用

- 🔴⭐ [adapter/vector_store.go](/server/internal/adapter/vector_store.go) — **双数据库连接池**：PgvectorStore 创建独立的 `sql.DB`，与 GORM 连接池并存且指向同一 PostgreSQL，浪费连接资源
- 🔴 [adapter/vector_store.go](/server/internal/adapter/vector_store.go) — NaN/Inf 静默替换为 0.0（应告警或拒绝写入，静默替换污染向量空间）
- 🟡 [adapter/vector_store.go](/server/internal/adapter/vector_store.go) — 应暴露 `Close()` 方法以在优雅关闭时清理独立连接池
- 🟡 [adapter/vector_store.go](/server/internal/adapter/vector_store.go) — `fmt.Sprintf("%.6f")` 截断 float32 精度，叠加 halfvec 量化进一步损失召回率
- 🟡 [adapter/storage_client.go](/server/internal/adapter/storage_client.go) — 上传 key 应由上层 helper 统一生成（当前分散在调用方拼接）

---
## 7. 数据看板与审计

> 对应图：[dashboard-audit-flow.md](diagrams/dashboard-audit-flow.md) — 7 项聚合统计 + 趋势图 + 审计日志查询
> 对应文档：[API/dashboard.md](API/dashboard.md) + [API/audit-log.md](API/audit-log.md)

### 看板统计

- 🔴 [service/dashboard_service.go](/server/internal/service/dashboard_service.go) — `.Scan()` 错误被丢弃：聚合查询失败时静默返回零值，掩盖数据库故障
- 🟡 [router/admin.go](/server/internal/router/admin.go) — Dashboard 路由使用 `audit:read` 权限控制，应有独立的 `dashboard:read` 权限

### 审计日志

#### P0 — 审计写入缺失（零调用方）

`AuditRepo.Create` 存在但零调用方，以下敏感操作全部无审计记录：

- 🔴 [service/user_service.go](/server/internal/service/user_service.go) — `Create`/`Update`/`Freeze`/`Restore` 无审计写入
- 🔴 [service/role_service.go](/server/internal/service/role_service.go) — `Create`/`Update`/`Delete` 无审计写入
- 🔴 [service/knowledge_service.go](/server/internal/service/knowledge_service.go) — `Publish`/`Disable` 无审计写入
- 🔴 [service/ticket_service.go](/server/internal/service/ticket_service.go) — `UpdateStatus` 无审计写入
- 🔴 [service/config_service.go](/server/internal/service/config_service.go) — `UpdateConfig` 无审计写入
- 🔴 [service/llm_config_service.go](/server/internal/service/llm_config_service.go) — `UpdateConfig` 无审计写入
- 🔴 [service/scheduler.go](/server/internal/service/scheduler.go) — `AutoClose` 无审计写入（需 operatorID=0 表示系统操作）

> 修复方式：各 Service 注入 `*AuditRepo`，在关键操作的事务内同步写入审计记录。
> 为什么同步而非异步：MVP 阶段审计写入是轻量 INSERT，同步执行保证事务一致性（CLAUDE.md §4 明确要求）。

#### P0 — 查询能力不足

- 🔴 [repository/audit_repo.go](/server/internal/repository/audit_repo.go) — `List()` 无日期范围过滤：全表扫描 `created_at` 无时间边界，大时间跨度查询可能超时
- 🔴 [dto/request/audit.go](/server/internal/dto/request/audit.go) — 缺少 `date_from`/`date_to` 参数
- 🟡 [repository/audit_repo.go](/server/internal/repository/audit_repo.go) — 查询维度不足：不支持 `target_type`/`target_id` 筛选，无法按资源维度检索（"谁改过这个申告？"）

#### P1 — 查询性能与数据质量

- 🟡 [service/audit_service.go](/server/internal/service/audit_service.go) — N+1 查询模式：先查 `audit_logs`，再批量查 `users`（两条 SQL），应改为单条 LEFT JOIN
- 🟡 [service/audit_service.go](/server/internal/service/audit_service.go) — `operatorID=0`（系统操作）未映射为"系统"显示名，前端展示空字符串
- 🟡 [dto/request/audit.go](/server/internal/dto/request/audit.go) — `action` 仅支持精确匹配，不支持前缀/模糊搜索（如 `user.*` 查看所有用户操作）
- 🟡 [service/audit_service.go](/server/internal/service/audit_service.go) — `batchGetOperatorNames` 中 `userRepo.FindByIDs` 失败时静默返回空 map，丢失错误信息

#### P2 — 测试与文档

- 🟡 [tests/repository/audit_repo_test.go](/server/tests/repository/audit_repo_test.go) — 测试使用 `init()` 直接 panic + 硬编码数据库凭据，不符合其他测试模块的标准模式
- 🟡 缺少 Service 层集成测试（验证各 Service 的审计写入正确性）
- 📝 [API/audit-log.md](/docs/API/audit-log.md) — action 分隔符不一致：文档用 `:`（`user:create`），代码用 `.`（`user.create`）
- 📝 [API/audit-log.md](/docs/API/audit-log.md) — 缺少新增查询参数（`target_type`/`target_id`/`date_from`/`date_to`）
- 📝 [diagrams/dashboard-audit-flow.md](/docs/diagrams/dashboard-audit-flow.md) — 审计查询流程图显示 JOIN 查询，但代码实际用两条 SQL + Go 层拼接

---
## 8. 基础设施与部署

> 对应图：[architecture.md](diagrams/architecture.md) + [request-lifecycle.md](diagrams/request-lifecycle.md) — 启动流程 → 中间件链 → 路由 → 数据库

### 启动流程（main.go）

- 🔴 [cmd/main.go](/server/cmd/main.go) — 初始化流程需拆成 `wireApp()`/`runServer()` 独立函数
- 🔴 [cmd/main.go](/server/cmd/main.go) — 数据库连接池参数应配置化（`MaxOpenConns=25` 等硬编码）
- 🔴 [cmd/main.go](/server/cmd/main.go) — AutoMigrate 不适合生产环境：自动建表/加列可能锁表或破坏数据
- 🔴 [cmd/main.go](/server/cmd/main.go) — LLM/Embedding 超时应区分场景配置化（查询改写 vs 生成 vs embedding 超时需求不同）
- 🔴 [cmd/main.go](/server/cmd/main.go) — VectorStore 初始化失败应返回健康状态（而非仅 warn）
- 🔴 [cmd/main.go](/server/cmd/main.go) — `ReadTimeout`/`WriteTimeout` 应配置化：SSE 路由需更长的 WriteTimeout
- 🔴⭐ [cmd/main.go](/server/cmd/main.go) — **nil 传播**：pgvector/MinIO 初始化失败后 `vectorStore`/`storageClient` 为 nil 但仍传给下游服务，后续调用 panic

### 配置管理

- 🔴⭐ [config/config.go](/server/internal/config/config.go) — **配置错误静默吞掉**：非 `ConfigFileNotFoundError` 的 `ReadInConfig` 错误（如 YAML 格式错误）被丢弃，应用以默认值启动
- 🟡 [config/config.go](/server/internal/config/config.go) — 增加 `Validate()` 统一校验：mode/port 有效性、JWT 非空、AI 阈值范围
- 🟡 [config/config.go](/server/internal/config/config.go) — 日志脱敏 password/api_key/secret：`config.Dump()` 或日志输出前应掩码
- 🟡 [config/config.go](/server/internal/config/config.go) — `BindEnv` 24 处返回值全部忽略
- 🟢 [config/config.go](/server/internal/config/config.go) — `time.Duration` 解析：`OPSMIND_JWT_ACCESS_EXPIRE=3600`（裸数字）会导致解析失败

### 数据库

- 🔴 [database/database.go](/server/internal/database/database.go) — DSN 密码直接 `fmt.Sprintf` 拼接：特殊字符（空格、`'`、`\`）导致连接失败
- 🔴 [database/database.go](/server/internal/database/database.go) — 生产环境打印 SQL 可能泄露业务数据和 PII
- 🔴 [database/database.go](/server/internal/database/database.go) — 启动时应 `PingContext` 超时校验：当前完全不 Ping，DB 不可达只在首次查询时暴露
- 🟡 [database/migrate.go](/server/internal/database/migrate.go) — AutoMigrate 不启用 pgvector 扩展，HNSW 索引需手动创建
- 🔴⭐ [database/migrate.go](/server/internal/database/migrate.go) — **ASC+DESC 双重索引**：GORM AutoMigrate 创建 `created_at` ASC 索引，migrate.go 又创建 DESC 索引，同一列两个索引加倍存储和写入开销

### 路由与中间件

- 🟡 [router/router.go](/server/internal/router/router.go) — placeholder 路由生产环境应 fail fast（而非返回 501 掩盖配置错误）
- 🟡 [router/router.go](/server/internal/router/router.go) — 增加 `/readyz` 健康探针（含 DB/VectorStore/MinIO/LLM 可达性检查）
- 🟡 [router/admin.go](/server/internal/router/admin.go) — 权限字符串散落各处，建议集中为常量文件
- 🟡 [router/portal.go](/server/internal/router/portal.go) — 门户路由无角色校验：仅需 JWT，不验证用户是否为报障人角色
- 🟡 [middleware/logger.go](/server/internal/middleware/logger.go) — `json.Marshal` 错误被丢弃
- 🟢 [middleware/logger.go](/server/internal/middleware/logger.go) — request-ID/userID 已记录，缺少**业务错误码**写入日志行（当前仅 HTTP status，无法关联 errcode 10001/20001 等）

### Repository 层

- 🔴 [repository/pagination.go](/server/internal/repository/pagination.go) — 所有 Repo 方法缺 `context.Context`：HTTP 取消不传播到 DB 查询，追踪无法关联
- 🟡 [repository/pagination.go](/server/internal/repository/pagination.go) — 分页辅助函数零调用方（死代码），各 Repo 自行实现分页
- 🟡 [repository/knowledge_repo.go](/server/internal/repository/knowledge_repo.go) — GORM query 对象复用于 Count 和 Find，session 状态可能泄漏

### 调度器

- 🟢 [service/scheduler.go](/server/internal/service/scheduler.go) — `Start` 应防重复调用（当前无幂等保护）
- 🟢⭐ [service/scheduler.go](/server/internal/service/scheduler.go) — **调度器首次启动不立即执行 AutoClose**：必须等待首个完整 cron 周期，频繁重启时超期工单可能长时间未关闭

### 日志与错误

- 🟡 [pkg/response/response.go](/server/pkg/response/response.go) — 错误响应缺少 `request_id`，前端报错难以和服务端日志关联
- 🟡 [pkg/response/response.go](/server/pkg/response/response.go) — 分页响应格式不统一（顶层 `total/page/page_size` vs 部分前端类型期望 `data.items/data.total`）
- 🟢 [middleware/logger.go](/server/internal/middleware/logger.go) — 部分 Handler 未使用 `handleServiceError` 封装（分散在 `auth.go` 而非 `common.go`）

### 文档一致性

- 📝⭐ [TECH.md §2.1](TECH.md) — **模块文件计数偏差**：TECH.md 记载「11 handlers」（实际 12 个 .go 文件 + common.go）、「12 services」（实际 13 个 .go 文件 + tx_manager.go）、「12 model 文件」（实际 10 个 .go 文件）、「15 web API 文件」（实际 12 个 .ts 文件）、「11 RAG 文件」（实际 12 个含 retriever.go）。文件计数需更新或改为"约 N 个"。
- 📝⭐ docs 目录引用 `docs/v2/` 和 `server/migrations/v2/`，但两个目录均不存在，迁移脚本缺失。

---
## 9. 前端架构与交互

> 对应全部业务流程图的前端部分（Vue 3 + TypeScript）

### 认证与安全

- 🔴⭐ [router/index.ts](/web/src/router/index.ts) — **JWT 过期检查 Bug**：`atob` 不兼容 base64url 编码（`-`/`_` 替代 `+`/`/`）。某些 JWT payload 会解码失败返回 null，导致过期检查失效——过期 token 被当作有效。
- 🔴⭐ [api/chat.ts](/web/src/api/chat.ts) — **SSE 流绕过 Axios 拦截器**：Token 刷新（401 响应）对流式请求完全失效。若 token 在流中途过期，连接断开且无恢复。
- 🔴⭐ [utils/request.ts](/web/src/utils/request.ts) — **Token 刷新竞态**：刷新失败时 `refreshSubscribers` 重置为 `[]` 但未通知已订阅者，其 Promise 永久挂起（内存泄漏）。
- 🟡 [views/auth/Login.vue](/web/src/views/auth/Login.vue) — 错误信息提取不完整：`catch` 用 `err?.message`（Axios 通用字符串），后端真实错误在 `err.response?.data?.message`

### 数据流与类型安全

- 🟡⭐ **系统性 `as any` 类型侵蚀**（~15 个文件）：`(res as any).data || res` 模式遍布组件和 Store，TypeScript 类型检查形同虚设。根因是组件不确定响应拦截器是否已解包 `ApiResponse<T>` 包装。
- 🟡⭐ [stores/auth.ts](/web/src/stores/auth.ts) — **循环依赖风险**：`api/auth.ts` 从 `@/stores/auth` 导入类型，若 store 未来从 API 导入则形成循环
- 🟡 [stores/chat.ts](/web/src/stores/chat.ts) — 反馈提交错误仅 `console.error`，用户点击「已解决/未解决」后静默失败
- 🟡 [views/portal/TicketSubmit.vue](/web/src/views/portal/TicketSubmit.vue) — `chat_context` 来自 URL query 参数直接传入 API，无校验（JSON 注入风险）

### 配置管理

- 🔴⭐ [views/admin/LLMConfig.vue](/web/src/views/admin/LLMConfig.vue) — **创建配置时测试连接崩溃**：`handleTestConnection` 调 `updateLLMConfig(editingId.value!)`，新建时 `editingId` 为 null，`!` 断言导致运行时崩溃
- 🟡 [views/admin/LLMConfig.vue](/web/src/views/admin/LLMConfig.vue) — 每次编辑必须重新输入 API Key（后端返回脱敏值，前端清空表单）
- 🟡⭐ [views/admin/ModelConfig.vue](/web/src/views/admin/ModelConfig.vue) + [views/admin/SystemConfig.vue](/web/src/views/admin/SystemConfig.vue) — **重复配置管理**：两页面独立管理 `ai.default_top_k` 和 `ai.confidence_threshold`，修改互不可见，最后写入胜出。📝 [PRD.md §3.1](PRD.md) 记载这两个参数应为统一 AI 配置，而非分散在两个独立页面。

### 组件拆分与重复代码

- 🟡 [views/portal/Chat.vue](/web/src/views/portal/Chat.vue) — 组件 >560 行，应拆分为 ChatInput/ChatMessage/ChatPipeline 子组件
- 🟡 [views/admin/LLMConfig.vue](/web/src/views/admin/LLMConfig.vue) — 组件 >610 行，应拆分
- 🟡 [views/admin/KnowledgeEdit.vue](/web/src/views/admin/KnowledgeEdit.vue) — 组件 >400 行，多文件上传只显示单个汇总结果
- 🟡⭐ **重复 `formatDate`**：[utils/date.ts](/web/src/utils/date.ts)、[utils/format.ts](/web/src/utils/format.ts)、Messages.vue、Dashboard.vue 各有一份独立实现
- 🟡 [views/admin/KnowledgeEdit.vue](/web/src/views/admin/KnowledgeEdit.vue) — `router.back()` 在直接访问页面时可能离开应用
- 🟡 [components/common/StatusBadge.vue](/web/src/components/common/StatusBadge.vue) — `knowledge` 类型未实现：TEXT_MAP 和 TYPE_MAP 中缺少 knowledge 键，知识文章状态渲染为「未知」

### 类型与边界情况

- 🟡 [views/admin/TicketDetail.vue](/web/src/views/admin/TicketDetail.vue) — 申告创建时间显示为原始 ISO 字符串，应使用 `formatDate`
- 🟡 [views/admin/TicketDetail.vue](/web/src/views/admin/TicketDetail.vue) — 知识候选创建：KB ID 自由输入无下拉选择/存在性校验
- 🟡 [views/admin/SystemConfig.vue](/web/src/views/admin/SystemConfig.vue) — Number 强制转换可能丢失前导零（如 `"00123"` → `123`）
- 🟡 [views/admin/UserList.vue](/web/src/views/admin/UserList.vue) — 本地 `UserItem` 接口与 API 类型不一致（自创 `role_ids` 字段）
- 🟡 [views/admin/KnowledgeList.vue](/web/src/views/admin/KnowledgeList.vue) — `(res.data as any).articles || (res.data as any).items || []` 三种回退提取，暴露 API 响应形状不确定
- 🟡 [components/layout/AdminLayout.vue](/web/src/components/layout/AdminLayout.vue) — 菜单路径硬编码字符串，应引用路由名称

### 基础设施

- 🟢 [utils/request.ts](/web/src/utils/request.ts) — `loadingState` 模块级全局计数器，SSR/并发测试不安全
- 🟢 [utils/request.ts](/web/src/utils/request.ts) — `baseURL` 为空依赖 Vite 代理，应读取 `VITE_API_BASE_URL`
- 🟢 [composables/useToast.ts](/web/src/composables/useToast.ts) — 每个组件独立 toast 状态，多 toast 冲突
- 🟢 [composables/useTheme.ts](/web/src/composables/useTheme.ts) — 模块级 `localStorage` 访问，SSR 不兼容
- 🟢 [api/dashboard.ts](/web/src/api/dashboard.ts) — `granularity` 参数定义但后端未实现（死代码）

---
## 10. 整表空数据（架构性变更）

> 以下表定义了 model 和 repository，但无 Service 层代码实际写入数据：

- 🔴 `audit_logs` — `AuditRepo.Create` 存在但零调用方，详细调用点清单见 §7 审计日志（共 7 个 Service 缺失审计写入）
- ✅ `chat_messages` — ~~`ChatRepo.CreateBatch` 零调用方~~ — 多轮对话重构后已修复：每次 CreateChatSession/StreamChat 均写入 chat_messages。
- 🔴 `chat_sessions.sources` — `CreateChatSession` 未填充 `Sources` 字段，检索引用证据永远为空
- 🟡 `system_configs.description` — `Upsert` 未设置 `Description`，配置说明永远为空

---
## 11. 代码 TODO 注释 ↔ 文档双向一致性（新增专项）

> ⭐ 本段为 2026-06-15 审计新增。记录代码中已有的 TODO 注释与文档的对应关系。

### 11.1 代码中有 TODO 但 TODO.md 未收录

以下代码注释了 TODO，但未在本文档对应章节列出。保证代码读者和文档读者看到一致的问题清单。

| 代码位置 | TODO 内容 | 应归类 |
|----------|-----------|--------|
| [pkg/response/response.go:43](/server/pkg/response/response.go) | 错误响应应带 `request_id` | §8 日志与错误 |
| [pkg/response/response.go:54](/server/pkg/response/response.go) | 分页响应格式不统一 | §8 日志与错误 |
| [service/llm_service.go:139](/server/internal/service/llm_service.go) | LLM 生成失败应返回 ErrAIUnavailable，而非保存兜底答案为成功（SSE 重构后移至 LLMService） | §2 智能问答 |
| [service/config_service.go:37](/server/internal/service/config_service.go) | Config key whitelist and type definitions needed | §8 配置管理 |
| [service/config_service.go:62](/server/internal/service/config_service.go) | 更新 ai 配置项未同步到运行时 | §8 配置管理 |
| [handler/ticket.go:60](/server/internal/handler/ticket.go) | ListByUser should reuse parsePagination | §4 申告管理 |
| [handler/role.go:63](/server/internal/handler/role.go) | Role list should support keyword search | §5 角色管理 |
| [handler/common.go:24](/server/internal/handler/common.go) | page_size max 应可配置 | §8 基础设施 |
| [handler/common.go:58](/server/internal/handler/common.go) | 应提供 `mustCurrentUserID()` helper | §8 基础设施 |
| [model/enums.go:94](/server/internal/model/enums.go) | 为知识文章/处理状态/紧急程度/影响范围提供统一 Text 方法 | §3 知识库 |
| [model/ticket.go:40](/server/internal/model/ticket.go) | OperatorID=0 for system operations conflicts with FK | §4 申告管理 |
| [dto/response/chat.go:15](/server/internal/dto/response/chat.go) | Pipeline metrics field needs naming unification | §2 智能问答 |
| [composables/useAIConfig.ts:48](/web/src/composables/useAIConfig.ts) | loadConfig swallows errors, uses defaults silently | §9 前端 |
| [views/admin/KnowledgeEdit.vue:151](/web/src/views/admin/KnowledgeEdit.vue) | fetchArticle/fetchKBs only console.error on failure | §9 前端 |
| [views/portal/TicketDetail.vue:96](/web/src/views/portal/TicketDetail.vue) | API call failures silently set null | §9 前端 |

### 11.2 P0/TODO 注释覆盖度验证

经逐项核验，所有 18 个 P0 项在对应代码文件中**均已存在** `// TODO(...)` 注释标记 — 开发者阅读代码时可以触达已知缺陷。

> 验证通过。但以下 5 个 TODOs 注释位于函数体内部深处（不在文件头/函数签名附近），可读性可提升：`migrate.go:48`（双重索引）、`llm_client.go:409`（doHTTPRequest 429/503）、`llm_config_service.go:152`（事务 DB 句柄）、`knowledge_service.go:303`（DeleteByArticle 非原子）、`knowledge_service.go:276`（status=3 语义混淆）。建议将注释提升到函数签名级或文件头。

---
## 统计

| 业务流程 | 🔴 P0 | 🟡 P1 | 🟢 P2 | 合计 |
|----------|-------|-------|-------|------|
| 1. 认证与授权 | 0 | 0 | 1 | 1 |
| 2. 智能问答 RAG | 5 | 36 | 14+1📝 | 55(+32) |
| 3. 知识库与文档管理 | 5 | 16+2📝 | 6 | 27(+2) |
| 4. 申告管理 | 5 | 6+1📝 | 2 | 12(+1) |
| 5. 用户与角色管理 | 2 | 9 | 4 | 15 |
| 6. LLM 配置与适配层 | 12+2📝 | 9 | 2 | 23(+2) |
| 7. 数据看板与审计 | 11 | 5 | 2+2📝 | 18(+2) |
| 8. 基础设施与部署 | 11 | 14+1📝 | 6 | 31(+3) |
| 9. 前端架构与交互 | 4 | 18 | 6 | 28(+2) |
| 10. 整表空数据 | 3 | 1 | 0 | 4 |
| 11. TODO ↔ 文档双向一致性 | — | — | — | (新增) |
| **合计** | **56** | **113** | **42+7📝** | **211** |

> ⭐ 标记项为 2026-06-15 再审计新发现问题（共 22 项，含 5 项 P0，5 项 📝 文档一致性缺陷）。
> 📝 标记项为代码实现与 API 文档/PRD/TECH.md 不一致的文档缺陷（共 8 项）。

### P0 速览（生产环境最优先修复）

1. LLM/Embedding 重试机制完全失效（`doHTTPRequest` 不包装 `retryableError`）
2. API Key 明文存储（数据库泄露 = 全部密钥暴露）📝 文档声明加密但未实现
3. TestConnection 测试错误端点（功能完全失效）📝 行为与 API 文档完全不符
4. UpdateConfig 清空 api_key（零值覆盖数据库中的密钥）📝 文档声明不传=保留但行为相反
5. 事务用错 DB 句柄（`ClearDefault` + `Create` 在事务外执行）
6. Processor Stop/Submit panic（优雅关闭时崩溃）
7. `role_repo.Delete(0)` 删除全部角色（GORM 零值陷阱）
8. pgvector/MinIO 初始化失败后 nil 传播到下游 panic
9. 配置 YAML 格式错误静默吞掉（应用以默认值启动）
10. ASC+DESC 双重索引加倍存储写入开销
11. JWT 过期检查 `atob` base64url 不兼容（前端）
12. SSE 流绕过 Axios 拦截器（token 过期无法刷新）
13. 前端创建 LLM 配置时测试连接崩溃
14. Repository 全层缺少 `context.Context`（取消/追踪断裂）
15. `is_default` 缺少部分唯一索引（并发创建多个默认配置）
16. embedding 模型硬编码为空字符串
17. `DeleteByArticle` + `BatchInsert` 非原子（文章向量丢失）
18. 文章状态编号文档与代码不一致（Disabled 文档=5 / 代码=0）
19. 上传 API 字段名与文档不一致（文档 `files` vs 代码 `file`）

---
**最后更新**：2026-06-16（RAG 模块深度审计：覆盖 pipeline/query_rewrite/multi_route/hybrid/bm25/rerank/chunker/embedder/processor/document_parser/retriever/embedding_client/llm_service 共 13 个文件，§2 扩容至 55 项）
