# OpsMind 代码改进清单

> 基于 2026-06-12 全量代码审查，从 76 个源文件的 208 条 `TODO` 注释汇总而来。
> 优先级定义：🔴 P0 生产隐患 → 🟡 P1 架构债务 → 🟢 P2 优化改进

---

## 1. 认证与安全

- 🔴 [middleware/auth.go](server/internal/middleware/auth.go) — JWT 只校验签名和过期，不检查用户冻结/权限撤销
- 🔴 [middleware/auth.go](server/internal/middleware/auth.go) — secret 为空时应在构造阶段拒绝启动
- 🔴 [pkg/jwt/jwt.go](server/pkg/jwt/jwt.go) — 限制 alg 必须为 HS256
- 🔴 [service/auth_service.go](server/internal/service/auth_service.go) — token 有效期写死 2h/7d，应从配置读取
- 🟡 [service/auth_service.go](server/internal/service/auth_service.go) — 增加登录失败限流/锁定
- 🟡 [service/auth_service.go](server/internal/service/auth_service.go) — RefreshToken 需校验 claims.TokenType
- 🟡 [middleware/cors.go](server/internal/middleware/cors.go) — release 模式禁止 `*` Origin
- 🟡 [middleware/request_id.go](server/internal/middleware/request_id.go) — 校验 X-Request-ID 长度和字符集
- 🟢 [pkg/jwt/jwt.go](server/pkg/jwt/jwt.go) — 增加 issuer/audience/jti/token_version

## 2. 数据完整性与事务

- 🔴 [service/ticket_service.go](server/internal/service/ticket_service.go) — supplement_count 并发竞态，应原子 UPDATE
- 🔴 [service/ticket_service.go](server/internal/service/ticket_service.go) — UpdateStatus + CreateRecord 不在同一事务
- 🔴 [service/user_service.go](server/internal/service/user_service.go) — AssignRoles 内层再开事务，嵌套事务风险
- 🔴 [service/knowledge_service.go](server/internal/service/knowledge_service.go) — DeleteByArticle + BatchInsert 非原子
- 🔴 [repository/ticket_repo.go](server/internal/repository/ticket_repo.go) — AutoClose 批量 UPDATE 不创建 TicketRecord
- 🟡 [service/llm_config_service.go](server/internal/service/llm_config_service.go) — ClearDefault + Create/Update 不在同一事务
- 🟡 [repository/ticket_repo.go](server/internal/repository/ticket_repo.go) — UpdateStatus 应返回 RowsAffected
- 🟡 [repository/chat_repo.go](server/internal/repository/chat_repo.go) — CreateBatch 和 Session 创建不在同一事务

## 3. 错误处理与降级

- 🔴 [service/dashboard_service.go](server/internal/service/dashboard_service.go) — `.Scan()` 错误被丢弃
- 🔴 [service/role_service.go](server/internal/service/role_service.go) — `.Count()` 错误被丢弃，绕过 Repository 直调 DB
- 🔴 [rag/processor.go](server/internal/rag/processor.go) — Stop 后 Submit 会 panic；Stop 非幂等
- 🔴 [rag/embedder.go](server/internal/rag/embedder.go) — 批次失败静默跳过，丢失对应关系
- 🔴 [adapter/storage_client.go](server/internal/adapter/storage_client.go) — ensureBucket 失败只 warn 继续启动
- 🟡 [rag/document_parser.go](server/internal/rag/document_parser.go) — LimitReader 到上限不报错，静默截断
- 🟡 [adapter/vector_store.go](server/internal/adapter/vector_store.go) — NaN/Inf 静默替换为 0.0
- 🟢 [middleware/logger.go](server/internal/middleware/logger.go) — json.Marshal 错误被丢弃

## 4. 申告管理

- 🔴 [service/ticket_service.go](server/internal/service/ticket_service.go) — ticket_no 纳秒+随机数在高并发下碰撞风险
- 🔴 [service/ticket_service.go](server/internal/service/ticket_service.go) — 状态机和 action 使用裸数字而非常量
- 🔴 [service/ticket_service.go](server/internal/service/ticket_service.go) — 门户端 GetDetail 不校验 ticket.UserID
- 🟡 [service/ticket_service.go](server/internal/service/ticket_service.go) — request_info 后应同步创建站内消息
- 🟡 [service/ticket_service.go](server/internal/service/ticket_service.go) — close 是否允许关闭已解决状态需明确
- 🟡 [repository/ticket_repo.go](server/internal/repository/ticket_repo.go) — SELECT ids + UPDATE 不是原子操作
- 🟢 [model/ticket.go](server/internal/model/ticket.go) — contact_phone 长度假设 11 位中国手机号
- 🟢 [dto/request/ticket.go](server/internal/dto/request/ticket.go) — ChatContext 应使用结构化对象

## 5. 知识库与 RAG 引擎

- 🔴 [service/knowledge_service.go](server/internal/service/knowledge_service.go) — 管道未初始化应映射为 ErrRAGUnavailable
- 🔴 [service/knowledge_service.go](server/internal/service/knowledge_service.go) — Publish 使用 context.Background 忽略请求取消
- 🔴 [rag/processor.go](server/internal/rag/processor.go) — embedding 模型硬编码为空字符串
- 🟡 [rag/bm25.go](server/internal/rag/bm25.go) — 超 10 万篇后 map 内存压力大
- 🟡 [rag/bm25.go](server/internal/rag/bm25.go) — BuildIndex 同步分词，请求路径调用造成长尾
- 🟡 [rag/pipeline.go](server/internal/rag/pipeline.go) — QueryRewrite 的 history 始终为 nil，上下文消歧未生效
- 🟡 [rag/pipeline.go](server/internal/rag/pipeline.go) — 重排序候选过多，应提前截断
- 🟡 [rag/rerank.go](server/internal/rag/rerank.go) — 用 LLM 做重排序，每次消耗 token 且延迟高
- 🟡 [rag/hybrid.go](server/internal/rag/hybrid.go) — 单路结果直接返回时未按 topK 截断
- 🟡 [rag/multi_route.go](server/internal/rag/multi_route.go) — LLM 输出子查询的清洗逻辑脆弱
- 🟡 [rag/query_rewrite.go](server/internal/rag/query_rewrite.go) — llm 为 nil 时应降级返回原 query
- 🟡 [service/knowledge_service.go](server/internal/service/knowledge_service.go) — Publish/Disable 应接收请求 ctx
- 🟡 [service/knowledge_service.go](server/internal/service/knowledge_service.go) — Disable 未校验当前状态是否为已发布
- 🟡 [service/knowledge_service.go](server/internal/service/knowledge_service.go) — RetryDocument 未校验是否处于 failed 状态
- 🟡 [service/knowledge_service.go](server/internal/service/knowledge_service.go) — 文章 status 和 process_status 状态机混淆
- 🟢 [service/knowledge_service.go](server/internal/service/knowledge_service.go) — tags 应 trim/去重/限制数量
- 🟢 [rag/document_parser.go](server/internal/rag/document_parser.go) — DOCX 只读 w:t 丢失表格和超链接
- 🟢 [rag/chunker.go](server/internal/rag/chunker.go) — mergeSplits 未实现 chunkOverlap
- 🟢 [rag/chunker.go](server/internal/rag/chunker.go) — 分块前应做文本归一化
- 🟢 [rag/types.go](server/internal/rag/types.go) — 增加 Normalize/Validate 方法

## 6. LLM 配置与适配层

- 🔴 [adapter/llm_client.go](server/internal/adapter/llm_client.go) — 校验 baseURL 非空且合法
- 🔴 [adapter/llm_client.go](server/internal/adapter/llm_client.go) — req.Model 为空时应返回参数错误
- 🔴 [adapter/llm_client.go](server/internal/adapter/llm_client.go) — 流式请求无 429/503 重试
- 🔴 [adapter/llm_client.go](server/internal/adapter/llm_client.go) — bufio.Scanner 默认上限 64K 可能溢出
- 🟡 [service/llm_config_service.go](server/internal/service/llm_config_service.go) — store 前应复制 cfg 防指针修改
- 🟡 [service/llm_config_service.go](server/internal/service/llm_config_service.go) — 构造函数不应 panic
- 🟡 [service/llm_config_service.go](server/internal/service/llm_config_service.go) — 默认配置切换后未重建 LLM/Embedding 客户端
- 🟡 [handler/llm_config.go](server/internal/handler/llm_config.go) — TestConnection 应基于被测配置创建临时客户端
- 🟡 [adapter/storage_client.go](server/internal/adapter/storage_client.go) — 上传 key 应由上层 helper 统一生成
- 🟡 [adapter/vector_store.go](server/internal/adapter/vector_store.go) — 应配置连接池并暴露 Close()
- 🟢 [model/llm_config.go](server/internal/model/llm_config.go) — api_key 应 AES-256 加密存储

## 7. 代码架构与规范

- 🔴 [cmd/main.go](server/cmd/main.go) — 初始化流程需拆成 wireApp()/runServer()
- 🔴 [cmd/main.go](server/cmd/main.go) — 数据库连接池参数应配置化
- 🔴 [cmd/main.go](server/cmd/main.go) — AutoMigrate 不适合生产环境
- 🔴 [cmd/main.go](server/cmd/main.go) — LLM/Embedding 超时应区分场景配置化
- 🔴 [cmd/main.go](server/cmd/main.go) — VectorStore 初始化失败应返回健康状态
- 🔴 [cmd/main.go](server/cmd/main.go) — ReadTimeout/WriteTimeout 应配置化，SSE 需单独超时
- 🟡 [repository/pagination.go](server/internal/repository/pagination.go) — 所有 Repo 方法缺 context.Context
- 🟡 [service/user_service.go](server/internal/service/user_service.go) — 列表 N+1 查询角色
- 🟡 [service/role_service.go](server/internal/service/role_service.go) — 禁止删除系统内置角色
- 🟡 [service/chat_service.go](server/internal/service/chat_service.go) — CreateChatSession 用 context.Background 不传播取消
- 🟡 [service/chat_service.go](server/internal/service/chat_service.go) — 置信度算法过于粗糙
- 🟡 [handler/chat.go](server/internal/handler/chat.go) — 模拟流式先完整生成再 SSE 输出，浪费首字节延迟
- 🟡 [middleware/rbac.go](server/internal/middleware/rbac.go) — 支持通配权限（knowledge:*）
- 🟡 [router/admin.go](server/internal/router/admin.go) — 权限字符串散落，建议集中为常量
- 🟢 [config/config.go](server/internal/config/config.go) — 增加 Validate() 统一校验配置
- 🟢 [config/config.go](server/internal/config/config.go) — 日志脱敏 password/api_key/secret
- 🟢 [middleware/logger.go](server/internal/middleware/logger.go) — 将 request-ID/userID/错误码写入日志

## 8. API 规范与前端交互

- 🟡 [pkg/response/response.go](server/pkg/response/response.go) — 错误响应应带 request_id
- 🟡 [pkg/response/response.go](server/pkg/response/response.go) — 分页响应格式与前端部分类型不一致
- 🟡 [handler/chat.go](server/internal/handler/chat.go) — SSE JSON 未用 json.Marshal，控制字符可致畸形
- 🟡 [handler/knowledge.go](server/internal/handler/knowledge.go) — UploadDocuments 应从 multipart 读取 files 数组
- 🟡 [handler/knowledge.go](server/internal/handler/knowledge.go) — 文档上传应结合 MIME sniffing 而不是只信任扩展名
- 🟡 [handler/knowledge.go](server/internal/handler/knowledge.go) — GetDocumentStatus 未校验 kb_id 与 article.KBID 一致
- 🟡 [handler/llm_config.go](server/internal/handler/llm_config.go) — CreateConfig 默认值 8192/1024 应在 Service 层
- 🟡 [service/chat_service.go](server/internal/service/chat_service.go) — GetChatDetail 未校验 session.UserID 归属
- 🟡 [service/chat_service.go](server/internal/service/chat_service.go) — FinalAnswer 和 streamWithLLM 各调用一次 LLM
- 🟡 [handler/common.go](server/internal/handler/common.go) — 各 Handler 分页规则不统一，应复用 parsePagination
- 🟡 [dto/response/knowledge.go](server/internal/dto/response/knowledge.go) — 门户端不应返回内部字段
- 🟢 [handler/chat.go](server/internal/handler/chat.go) — 部分 Handler 未使用 handleServiceError 封装
- 🟢 [router/router.go](server/internal/router/router.go) — 增加 /readyz 健康探针

## 9. 部署与运维

- 🔴 [database/database.go](server/internal/database/database.go) — DSN 密码直接拼接，特殊字符可致连接失败
- 🔴 [database/database.go](server/internal/database/database.go) — 生产环境打印 SQL 可能泄露数据
- 🔴 [database/database.go](server/internal/database/database.go) — 启动时应 PingContext 超时校验
- 🟡 [database/migrate.go](server/internal/database/migrate.go) — AutoMigrate 不启用 pgvector 扩展，索引需手动创建
- 🟡 [router/router.go](server/internal/router/router.go) — placeholder 路由生产环境应 fail fast
- 🟢 [service/scheduler.go](server/internal/service/scheduler.go) — Start 应防重复调用

## 10. 整表空数据（架构性变更）

以下表仅定义了 model 和 repository，但无任何 Service 层代码写入数据：

- 🔴 `audit_logs` — `AuditRepo.Create` 存在但零调用方，所有敏感操作无审计记录
- 🔴 `chat_messages` — `ChatRepo.CreateBatch` 存在但零调用方，对话历史从未持久化
- 🔴 `chat_sessions.sources` — `CreateChatSession` 未填充 `Sources` 字段，检索引用证据永远为空
- 🟡 `system_configs.description` — `Upsert` 未设置 `Description`，配置说明永远为空

---

**统计**: 共 212 条，覆盖 76 个源文件 | 🔴 P0: 61 · 🟡 P1: 108 · 🟢 P2: 43

**统计**: 共 208 条 TODO，覆盖 76 个源文件 | 🔴 P0: 58 · 🟡 P1: 107 · 🟢 P2: 43
