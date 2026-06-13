# OpsMind 代码改进清单

> 基于 2026-06-13 全量代码再审计（前后端 119+ 源文件），按业务流程重组。
> 覆盖 [docs/diagrams/](docs/diagrams/) 中 10 份业务流程图对应的全部模块。
> 优先级：🔴 P0 生产隐患 / 🟡 P1 架构债务 / 🟢 P2 优化改进
> ⭐ 标记为本次审计新发现的问题。

---

## 1. 认证与授权

> 对应图：[auth-flow.md](docs/diagrams/auth-flow.md) — 登录 → JWT 双令牌 → 中间件链 → RBAC 权限校验

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

### 中间件安全

- ✅ [middleware/cors.go](/server/internal/middleware/cors.go) — ~~DNS 重绑定风险~~ — release 模式强制配置 AllowOrigins + localhost 告警
- ✅ [middleware/request_id.go](/server/internal/middleware/request_id.go) — ~~X-Request-ID 注入~~ — 已有长度限制(128) + 字符集校验(regex)
- ✅ [middleware/rbac.go](/server/internal/middleware/rbac.go) — ~~通配权限~~ — 已支持 `*` 全匹配 + `prefix:*` 前缀通配
- ✅ [middleware/rbac.go](/server/internal/middleware/rbac.go) — ~~空 permissions~~ — 注册时 slog.Warn 告警 + 保持不变的安全默认值

### 密码策略

- ✅ [pkg/hash/hash.go](/server/pkg/hash/hash.go) — ~~字节计数~~ — 改用 `utf8.RuneCountInString`（非 ASCII 字符各计 1 位）
- ✅ [pkg/hash/hash.go](/server/pkg/hash/hash.go) — ~~检测不一致~~ — 大小写/数字统一使用 `unicode.IsLower/IsUpper/IsDigit`
- ✅ [pkg/hash/hash.go](/server/pkg/hash/hash.go) — bcrypt cost=10 硬编码，应可配置以应对硬件升级

---

## 2. 智能问答 RAG

> 对应图：[chat-rag-flow.md](docs/diagrams/chat-rag-flow.md) — SSE 流式 → Pipeline 执行 → 多路检索 → 混合融合 → 重排序 → LLM 生成

### SSE 流式输出（核心路径）

- 🔴⭐ [handler/chat.go](/server/internal/handler/chat.go) — **流式答案与存储答案不一致**：`StreamChatSession` 先调 `CreateChatSession`（RAG 增强 LLM 生成 → 存库），再调 `streamWithLLM`（裸问题再次调 LLM → 流式输出）。两次 LLM 调用独立，用户看到的内容与数据库存储的内容不同，且推理成本翻倍。
- 🔴 [handler/chat.go](/server/internal/handler/chat.go) — SSE 流中 LLM 错误被静默吞掉（`continue`），前端不知道生成中断
- 🟡 [handler/chat.go](/server/internal/handler/chat.go) — SSE JSON 未用 `json.Marshal`，控制字符可致 SSE 帧畸形
- 🟡 [handler/chat.go](/server/internal/handler/chat.go) — 模拟流式先完整生成再 SSE 输出，浪费首字节延迟
- 🟡 [service/chat_service.go](/server/internal/service/chat_service.go) — `FinalAnswer`（非流式）和 `streamWithLLM`（流式）各调用一次 LLM，非流式路径多一次调用

### RAG 管道

- 🔴 [rag/pipeline.go](/server/internal/rag/pipeline.go) — `llmClient` 为 nil 时，QueryRewrite/MultiRoute/Rerank 会 panic（nil 指针解引用）
- 🔴⭐ [rag/rerank.go](/server/internal/rag/rerank.go) — 重排序 prompt 无长度截断：候选 chunk 全量拼接，可超出 LLM 上下文窗口致 400 错误
- 🟡 [rag/pipeline.go](/server/internal/rag/pipeline.go) — QueryRewrite 的 history 始终为 nil，上下文消歧功能未生效
- 🟡 [rag/pipeline.go](/server/internal/rag/pipeline.go) — 重排序候选过多时应提前截断，减少 token 消耗和延迟
- 🟡 [rag/query_rewrite.go](/server/internal/rag/query_rewrite.go) — llm 为 nil 时应降级返回原 query，而非 panic
- 🟡 [rag/multi_route.go](/server/internal/rag/multi_route.go) — LLM 输出子查询的清洗逻辑脆弱（正则匹配依赖特定格式）
- 🟡⭐ [rag/multi_route.go](/server/internal/rag/multi_route.go) — k（子查询数量）无上限，k=100 可致百倍检索放大
- 🟡 [rag/hybrid.go](/server/internal/rag/hybrid.go) — 单路结果直接返回时未按 topK 截断
- 🟡 [service/chat_service.go](/server/internal/service/chat_service.go) — 前端 RAGOptions 完全被忽略：`CreateChatSession` 硬编码全部 `true`，高级设置表单无效
- 🟡 [rag/types.go](/server/internal/rag/types.go) — RAGOptions 使用裸 `bool`，零值问题致空 JSON `{}` 全部禁用，与文档默认「全部启用」矛盾

### BM25 检索

- 🟡 [rag/bm25.go](/server/internal/rag/bm25.go) — 超 10 万篇后内存 map 压力大，应考虑分片或磁盘索引
- 🟡 [rag/bm25.go](/server/internal/rag/bm25.go) — BuildIndex 同步分词，请求路径调用造成长尾延迟
- 🟡 [rag/bm25.go](/server/internal/rag/bm25.go) — LoadDict 错误被丢弃，分词器静默回退到字符级切分

### 置信度与数据完整性

- 🟡 [service/chat_service.go](/server/internal/service/chat_service.go) — 置信度算法粗糙：直接取 max(chunk.Score)，但 BM25 和 RRF 分数量纲不同
- 🟡 [service/chat_service.go](/server/internal/service/chat_service.go) — Sources（检索引用）在 `CreateChatSession` 中未写入 `session.Sources` 字段
- 🔴 [service/chat_service.go](/server/internal/service/chat_service.go) — `CreateChatSession` 用 `context.Background` 不传播请求取消
- 🟡 [service/chat_service.go](/server/internal/service/chat_service.go) — `GetChatDetail` 未校验 `session.UserID` 归属，任意用户可通过猜测 ID 查看他人对话
- 🟡 [repository/chat_repo.go](/server/internal/repository/chat_repo.go) — `CreateBatch` 和 Session 创建不在同一事务
- 🟢⭐ [model/chat.go](/server/internal/model/chat.go) — `ChatMessage.SessionID` 缺少索引和 FK 约束，历史查询将全表扫描

### 输入校验

- 🟢 [dto/request/chat.go](/server/internal/dto/request/chat.go) — Question 字段缺少 `max` 绑定，可发送 MB 级文本直入管道和 LLM

---

## 3. 知识库与文档管理

> 对应图：[knowledge-publish-flow.md](docs/diagrams/knowledge-publish-flow.md) + [document-upload-flow.md](docs/diagrams/document-upload-flow.md) — 文章生命周期 → 分块 → Embedding → pgvector；文档上传 → 解析 → 异步处理

### 知识发布管道（核心路径）

- 🔴 [service/knowledge_service.go](/server/internal/service/knowledge_service.go) — `DeleteByArticle` + `BatchInsert` 非原子：删除旧向量成功但新向量写入失败 → 文章向量永久丢失
- 🔴 [service/knowledge_service.go](/server/internal/service/knowledge_service.go) — Publish 使用 `context.Background` 忽略请求取消
- 🔴 [service/knowledge_service.go](/server/internal/service/knowledge_service.go) — 管道未初始化（`pipeline == nil`）时应映射为 `ErrRAGUnavailable`，而非静默跳过
- 🔴 [rag/processor.go](/server/internal/rag/processor.go) — embedding 模型硬编码为空字符串 `""`，应从 KB 配置读取

### 文章状态机

- 🟡 [service/knowledge_service.go](/server/internal/service/knowledge_service.go) — Enable 仅重置 status 为 Draft，不重新执行分块/Embedding/pgvector 写入
- 🟡 [service/knowledge_service.go](/server/internal/service/knowledge_service.go) — Disable 未校验当前状态是否为已发布（Draft 可直接 Disable）
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
- 🟡 [handler/knowledge.go](/server/internal/handler/knowledge.go) — UploadDocuments 应从 multipart 读取 `files` 数组，当前只读单个 `file` 字段

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

> 对应图：[ticket-lifecycle.md](docs/diagrams/ticket-lifecycle.md) + [ticket-state-machine.md](docs/diagrams/ticket-state-machine.md) — 创建→处理→补充→解决→关闭 + AutoClose

### 数据完整性与事务

- 🔴 [service/ticket_service.go](/server/internal/service/ticket_service.go) — `supplement_count` 并发竞态：应在 Repository 层用 `UPDATE ... WHERE supplement_count < 3` 原子操作
- 🔴 [service/ticket_service.go](/server/internal/service/ticket_service.go) — `UpdateStatus` + `CreateRecord` 不在同一事务（应先获取 Ticket 再在同一 TxManager 事务中 Update + CreateRecord）
- 🔴 [repository/ticket_repo.go](/server/internal/repository/ticket_repo.go) — AutoClose 批量 UPDATE 不创建 TicketRecord（应在 Service 层 TxManager 事务内为每个 ID 创建 record）
- 🔴 [repository/ticket_repo.go](/server/internal/repository/ticket_repo.go) — `SELECT ids + UPDATE` 不是原子操作，AutoClose 有 TOCTOU 竞态
- 🟡 [repository/ticket_repo.go](/server/internal/repository/ticket_repo.go) — `UpdateStatus` 应返回 `RowsAffected`，否则更新不存在的 ID 静默成功

### 状态机

- 🔴 [service/ticket_service.go](/server/internal/service/ticket_service.go) — 状态机和 action 使用裸数字（1,2,3,4,5）而非常量，可读性和可维护性差
- 🟡 [service/ticket_service.go](/server/internal/service/ticket_service.go) — close 操作是否允许关闭已解决（Resolved）状态需在代码中明确语义
- 🟡 [service/ticket_service.go](/server/internal/service/ticket_service.go) — `request_info` 后应同步创建站内消息通知（当前图中有但代码中需验证）

### 安全与校验

- 🔴 [service/ticket_service.go](/server/internal/service/ticket_service.go) — `ticket_no` 使用 `time.Now().UnixNano()%1000000` + 随机数，高并发下碰撞风险（纳秒取模后仅 6 位）+ 随机数种子上未显示初始化
- 🔴 [service/ticket_service.go](/server/internal/service/ticket_service.go) — 门户端 `GetDetail` 不校验 `ticket.UserID == 当前用户`，任意用户可查看他人申告
- 🟢 [model/ticket.go](/server/internal/model/ticket.go) — `contact_phone` 长度假设 11 位中国手机号，国际化和扩展号码（分机）不友好
- 🟢 [dto/request/ticket.go](/server/internal/dto/request/ticket.go) — `ChatContext` 应使用结构化对象而非 `string`

---

## 5. 用户与角色管理

> 对应图：[user-rbac-flow.md](docs/diagrams/user-rbac-flow.md) — 用户 CRUD + 角色权限 + 菜单树

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

> 对应图：[llm-config-flow.md](docs/diagrams/llm-config-flow.md) — CRUD + TestConnection + atomic.Value 热替换 + API Key 脱敏

### LLM 客户端

- 🔴⭐ [adapter/llm_client.go](/server/internal/adapter/llm_client.go) — **重试机制完全失效**：`doHTTPRequest` 将 429/503 包装为普通 `fmt.Errorf` 而非 `*retryableError`，`isRetryable()` 永远返回 false。LLM 同步重试和 Embedding 重试均不工作。
- 🔴 [adapter/llm_client.go](/server/internal/adapter/llm_client.go) — 校验 baseURL 非空且合法（空字符串产生误导性 "unsupported protocol scheme" 错误）
- 🔴 [adapter/llm_client.go](/server/internal/adapter/llm_client.go) — `req.Model` 为空时应返回参数错误（而非将空字符串发给 API）
- 🔴 [adapter/llm_client.go](/server/internal/adapter/llm_client.go) — `bufio.Scanner` 默认 64KB 上限，大 SSE `data:` 行会触发 `ErrTooLong` 静默截断流
- 🔴 [adapter/llm_client.go](/server/internal/adapter/llm_client.go) — 流式请求无 429/503 重试（直接 return status code error）

### LLM 配置管理

- 🔴⭐ [service/llm_config_service.go](/server/internal/service/llm_config_service.go) — **事务内使用 repo 原始 DB 而非 tx**：`ClearDefault()` 和 `Create()` 在 `db.Transaction()` 内调但用 `s.repo` 的 `*gorm.DB`，操作在事务外执行，原子性为空壳
- 🔴⭐ [handler/llm_config.go](/server/internal/handler/llm_config.go) — **TestConnection 测试的是全局默认客户端而非被测配置**：始终用 `h.llmClient`，选择 `:id` 配置无意义——功能完全失效
- 🔴⭐ [handler/llm_config.go](/server/internal/handler/llm_config.go) — **UpdateConfig 会清空 api_key**：若请求不传 `api_key`，零值 `""` 覆盖数据库中已存储的密钥
- 🔴 [service/llm_config_service.go](/server/internal/service/llm_config_service.go) — `atomic.Store` 直接存指针，未复制 cfg；调用方修改原对象 → 并发读看到中间态（数据竞态）
- 🔴 [service/llm_config_service.go](/server/internal/service/llm_config_service.go) — 构造函数不应 panic：mock 类型不匹配会直接崩溃进程，应返回 error
- 🔴⭐ [model/llm_config.go](/server/internal/model/llm_config.go) — **API Key 明文存储**：数据库泄露 = 所有 LLM 提供商密钥暴露。需 AES-256 加密存储
- 🔴⭐ [repository/llm_config_repo.go](/server/internal/repository/llm_config_repo.go) — **缺少 `is_default` 部分唯一索引**：并发创建默认配置可产生多条 `is_default=true` 记录
- 🟡 [service/llm_config_service.go](/server/internal/service/llm_config_service.go) — 默认配置切换后未重建 LLM/Embedding 客户端（仅替换了配置值，已初始化的 HTTP 客户端仍指向旧 Base URL）
- 🟡 [service/llm_config_service.go](/server/internal/service/llm_config_service.go) — 缺少输入校验：providerType 白名单、baseURL 格式、maxTokens/vectorDimension 范围
- 🟡 [service/llm_config_service.go](/server/internal/service/llm_config_service.go) — 删除配置前未检查 FK 引用（knowledge_bases.llm_config_id）
- 🟡 [handler/llm_config.go](/server/internal/handler/llm_config.go) — CreateConfig 默认值 8192/1024 应在 Service 层而非 Handler 层
- 🟢 [handler/llm_config.go](/server/internal/handler/llm_config.go) — TestConnection 失败应返回 `code=0, success=false` 而非 `code=20001`

### 适配层通用

- 🔴⭐ [adapter/vector_store.go](/server/internal/adapter/vector_store.go) — **双数据库连接池**：PgvectorStore 创建独立的 `sql.DB`，与 GORM 连接池并存且指向同一 PostgreSQL，浪费连接资源
- 🔴 [adapter/vector_store.go](/server/internal/adapter/vector_store.go) — NaN/Inf 静默替换为 0.0（应告警或拒绝写入，静默替换污染向量空间）
- 🟡 [adapter/vector_store.go](/server/internal/adapter/vector_store.go) — 应暴露 `Close()` 方法以在优雅关闭时清理独立连接池
- 🟡 [adapter/vector_store.go](/server/internal/adapter/vector_store.go) — `fmt.Sprintf("%.6f")` 截断 float32 精度，叠加 halfvec 量化进一步损失召回率
- 🟡 [adapter/storage_client.go](/server/internal/adapter/storage_client.go) — 上传 key 应由上层 helper 统一生成（当前分散在调用方拼接）

---

## 7. 数据看板与审计

> 对应图：[dashboard-audit-flow.md](docs/diagrams/dashboard-audit-flow.md) — 7 项聚合统计 + 趋势图 + 审计日志查询

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
- 📝 [docs/API/audit-log.md](/docs/API/audit-log.md) — action 分隔符不一致：文档用 `:`（`user:create`），代码用 `.`（`user.create`）
- 📝 [docs/API/audit-log.md](/docs/API/audit-log.md) — 缺少新增查询参数（`target_type`/`target_id`/`date_from`/`date_to`）
- 📝 [docs/diagrams/dashboard-audit-flow.md](/docs/diagrams/dashboard-audit-flow.md) — 审计查询流程图显示 JOIN 查询，但代码实际用两条 SQL + Go 层拼接

---

## 8. 基础设施与部署

> 对应图：[architecture.md](docs/diagrams/architecture.md) + [request-lifecycle.md](docs/diagrams/request-lifecycle.md) — 启动流程 → 中间件链 → 路由 → 数据库

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
- 🟢 [middleware/logger.go](/server/internal/middleware/logger.go) — 将 request-ID/userID/错误码写入日志行（当前仅记录 method/path/status/latency）

### Repository 层

- 🔴 [repository/pagination.go](/server/internal/repository/pagination.go) — 所有 Repo 方法缺 `context.Context`：HTTP 取消不传播到 DB 查询，追踪无法关联
- 🟡 [repository/pagination.go](/server/internal/repository/pagination.go) — 分页辅助函数零调用方（死代码），各 Repo 自行实现分页
- 🟡 [repository/knowledge_repo.go](/server/internal/repository/knowledge_repo.go) — GORM query 对象复用于 Count 和 Find，session 状态可能泄漏

### 调度器

- 🟢 [service/scheduler.go](/server/internal/service/scheduler.go) — `Start` 应防重复调用（当前无幂等保护）

### 日志与错误

- 🟢 [middleware/logger.go](/server/internal/middleware/logger.go) — 部分 Handler 未使用 `handleServiceError` 封装（分散在 `auth.go` 而非 `common.go`）

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
- 🟡 [views/admin/TicketSubmit.vue](/web/src/views/admin/TicketSubmit.vue) — `chat_context` 来自 URL query 参数直接传入 API，无校验（JSON 注入风险）

### 配置管理

- 🔴⭐ [views/admin/LLMConfig.vue](/web/src/views/admin/LLMConfig.vue) — **创建配置时测试连接崩溃**：`handleTestConnection` 调 `updateLLMConfig(editingId.value!)`，新建时 `editingId` 为 null，`!` 断言导致运行时崩溃
- 🟡 [views/admin/LLMConfig.vue](/web/src/views/admin/LLMConfig.vue) — 每次编辑必须重新输入 API Key（后端返回脱敏值，前端清空表单）
- 🟡⭐ [views/admin/ModelConfig.vue](/web/src/views/admin/ModelConfig.vue) + [views/admin/SystemConfig.vue](/web/src/views/admin/SystemConfig.vue) — **重复配置管理**：两页面独立管理 `ai.default_top_k` 和 `ai.confidence_threshold`，修改互不可见，最后写入胜出

### 组件拆分与重复代码

- 🟡 [views/portal/Chat.vue](/web/src/views/portal/Chat.vue) — 组件 >560 行，应拆分为 ChatInput/ChatMessage/ChatPipeline 子组件
- 🟡 [views/admin/LLMConfig.vue](/web/src/views/admin/LLMConfig.vue) — 组件 >610 行，应拆分
- 🟡 [views/admin/KnowledgeEdit.vue](/web/src/views/admin/KnowledgeEdit.vue) — 组件 >400 行，多文件上传只显示单个汇总结果
- 🟡⭐ **重复 `formatDate`**：`utils/date.ts`、`utils/format.ts`、`Messages.vue`、`Dashboard.vue` 各有一份实现
- 🟡 [views/admin/KnowledgeEdit.vue](/web/src/views/admin/KnowledgeEdit.vue) — `router.back()` 在直接访问页面时可能离开应用
- 🟡 [components/common/StatusBadge.vue](/web/src/components/common/StatusBadge.vue) — `knowledge` 类型未实现（渲染「未知」）

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
- 🔴 `chat_messages` — `ChatRepo.CreateBatch` 存在但零调用方，对话历史从未持久化（用户刷新后历史消失）
- 🔴 `chat_sessions.sources` — `CreateChatSession` 未填充 `Sources` 字段，检索引用证据永远为空
- 🟡 `system_configs.description` — `Upsert` 未设置 `Description`，配置说明永远为空

---

## 统计

| 业务流程 | 🔴 P0 | 🟡 P1 | 🟢 P2 | 合计 |
|----------|-------|-------|-------|------|
| 1. 认证与授权 | 0 | 0 | 1 | 1 |
| 2. 智能问答 RAG | 5 | 14 | 5 | 24 |
| 3. 知识库与文档管理 | 5 | 14 | 6 | 25 |
| 4. 申告管理 | 5 | 5 | 2 | 12 |
| 5. 用户与角色管理 | 2 | 9 | 4 | 15 |
| 6. LLM 配置与适配层 | 11 | 8 | 2 | 21 |
| 7. 数据看板与审计 | 11 | 5 | 2 | 18 |
| 8. 基础设施与部署 | 11 | 12 | 5 | 28 |
| 9. 前端架构与交互 | 4 | 16 | 6 | 26 |
| 10. 整表空数据 | 3 | 1 | 0 | 4 |
| **合计** | **56** | **84** | **33** | **173** |

> ⭐ 标记项为本次再审计新发现问题（共 31 项，含 18 项 P0）。

### P0 速览（生产环境最优先修复）

1. LLM/Embedding 重试机制完全失效（`doHTTPRequest` 不包装 `retryableError`）
2. 流式答案与存储答案不一致（两次独立 LLM 调用）
3. API Key 明文存储（数据库泄露 = 全部密钥暴露）
4. TestConnection 测试错误端点（功能完全失效）
5. UpdateConfig 清空 api_key（零值覆盖数据库中的密钥）
6. 事务用错 DB 句柄（`ClearDefault` + `Create` 在事务外执行）
7. Processor Stop/Submit panic（优雅关闭时崩溃）
8. `role_repo.Delete(0)` 删除全部角色（GORM 零值陷阱）
9. pgvector/MinIO 初始化失败后 nil 传播到下游 panic
10. 配置 YAML 格式错误静默吞掉（应用以默认值启动）
11. ASC+DESC 双重索引加倍存储写入开销
12. JWT 过期检查 `atob` base64url 不兼容（前端）
13. SSE 流绕过 Axios 拦截器（token 过期无法刷新）
14. 前端创建 LLM 配置时测试连接崩溃
15. Repository 全层缺少 `context.Context`（取消/追踪断裂）
16. `is_default` 缺少部分唯一索引（并发创建多个默认配置）
17. embedding 模型硬编码为空字符串
18. `DeleteByArticle` + `BatchInsert` 非原子（文章向量丢失）

---

**最后更新**：2026-06-13（全量再审计 + 认证模块 9 项全部清零 + 审计日志模块深度审查）
