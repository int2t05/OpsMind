# 运维数字员工系统 — 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 按 6 个里程碑交付完整的运维数字员工系统，覆盖后端 Go + Gin 服务、Vue 3 前端、AnythingLLM RAG 集成、Docker Compose 编排。

**Architecture:** 单体分层架构（Modular Monolith），Handler → Service → Repository 三层分离。AnythingLLM 作为 Docker 内部组件通过 `RagClient` 适配层接入，vLLM 通过 AnythingLLM `generic-openai` 提供商调用。PostgreSQL 18 存储业务数据和 AnythingLLM 同步状态，AnythingLLM LanceDB 负责 RAG 检索。

**Tech Stack:** Go 1.22+ / Gin 1.9+ / GORM v1.25+ / PostgreSQL 18 / Vue 3.4+ / Radix Vue 1.9+ / Pinia 2.1+ / AnythingLLM / vLLM / MinIO / JWT (golang-jwt v5)

**关联文档：**
- [PRD.md](PRD.md) — 产品需求文档 v2.2
- [TECH.md](TECH.md) — 技术架构文档 v1.3
- [ANYTHINGLLM_AI_INTEGRATION.md](ANYTHINGLLM_AI_INTEGRATION.md) — AnythingLLM 集成方案 v1.1
- [prompts/DESIGN-linear.app.md](prompts/DESIGN-linear.app.md) — Linear Design 系统约束

---

## 文件结构总览

下表列出本计划涉及的全部文件，按职责分组。每个文件的详细说明见对应任务。

### 后端 (`server/`)

| 文件路径 | 职责 | 所属任务 |
| --- | --- | --- |
| `cmd/main.go` | 程序入口，初始化配置、数据库、路由、调度器 | T01, T05 |
| `internal/config/config.go` | Viper 配置加载，定义全部配置结构体 | T02 |
| `internal/config/config.yaml` | 默认配置文件 | T02 |
| `internal/database/database.go` | GORM 数据库连接初始化 | T03 |
| `internal/database/migrate.go` | 自动迁移（全部 GORM 模型） | T04 |
| `internal/model/user.go` | User / Role / Menu / UserRole / RoleMenu 数据模型 | T04 |
| `internal/model/ticket.go` | Ticket / TicketRecord 数据模型 | T04 |
| `internal/model/knowledge.go` | KnowledgeBase / KnowledgeArticle / KnowledgeChunk 数据模型 | T04 |
| `internal/model/embedding.go` | EmbeddingConfig 数据模型 | T04 |
| `internal/model/chat.go` | ChatSession / ChatMessage 数据模型 | T04 |
| `internal/model/audit.go` | AuditLog 数据模型 | T04 |
| `internal/model/system.go` | SystemConfig 数据模型 | T04 |
| `internal/model/message.go` | Message 数据模型 | T04 |
| `pkg/response/response.go` | 统一 JSON 响应格式封装 | T05 |
| `pkg/errcode/errcode.go` | 全局错误码定义（10001-99999） | T05 |
| `pkg/jwt/jwt.go` | JWT 生成、解析、刷新工具 | T05 |
| `pkg/hash/hash.go` | bcrypt 密码哈希和验证工具 | T05 |
| `internal/middleware/cors.go` | CORS 跨域中间件 | T06 |
| `internal/middleware/logger.go` | 请求日志中间件（slog） | T06 |
| `internal/middleware/request_id.go` | 请求 ID 中间件（UUID 链路追踪） | T06 |
| `internal/middleware/auth.go` | JWT 认证中间件 | T12 |
| `internal/middleware/rbac.go` | RBAC 权限校验中间件 | T13 |
| `internal/router/router.go` | Gin Router 初始化，挂载中间件和路由组 | T07 |
| `internal/router/portal.go` | 门户端路由注册 | T07 |
| `internal/router/admin.go` | 后台管理路由注册 | T07 |
| `internal/dto/request/auth.go` | 认证相关请求 DTO | T11 |
| `internal/dto/request/user.go` | 用户管理请求 DTO | T14 |
| `internal/dto/request/ticket.go` | 申告相关请求 DTO | T24 |
| `internal/dto/request/chat.go` | 问答相关请求 DTO | T26 |
| `internal/dto/request/knowledge.go` | 知识库相关请求 DTO | T18 |
| `internal/dto/request/dashboard.go` | 看板请求 DTO | T32 |
| `internal/dto/response/auth.go` | 认证相关响应 DTO | T11 |
| `internal/dto/response/user.go` | 用户管理响应 DTO | T14 |
| `internal/dto/response/ticket.go` | 申告相关响应 DTO | T24 |
| `internal/dto/response/chat.go` | 问答相关响应 DTO | T26 |
| `internal/dto/response/knowledge.go` | 知识库相关响应 DTO | T18 |
| `internal/dto/response/dashboard.go` | 看板响应 DTO | T32 |
| `internal/repository/user_repo.go` | 用户和角色数据访问 | T10 |
| `internal/repository/ticket_repo.go` | 申告数据访问 | T23 |
| `internal/repository/knowledge_repo.go` | 知识库数据访问 | T17 |
| `internal/repository/chat_repo.go` | 问答会话数据访问 | T25 |
| `internal/repository/audit_repo.go` | 审计日志数据访问 | T33 |
| `internal/repository/config_repo.go` | 系统配置数据访问 | T09 |
| `internal/repository/message_repo.go` | 站内消息数据访问 | T29 |
| `internal/adapter/rag_client.go` | AnythingLLM RagClient 接口 + HTTP 实现 | T20 |
| `internal/adapter/storage_client.go` | MinIO StorageClient 接口 + S3 实现 | T27 |
| `internal/service/auth_service.go` | 登录、刷新令牌、修改密码 | T11 |
| `internal/service/user_service.go` | 用户 CRUD、冻结/恢复 | T14 ✅ |
| `internal/service/role_service.go` | 角色 CRUD、菜单权限绑定 | T15 |
| `internal/service/knowledge_service.go` | 知识库/知识条目 CRUD、审核、发布、停用、重试同步 | T18, T19 |
| `internal/service/chat_service.go` | 问答会话创建、置信度判断、降级处理 | T26 |
| `internal/service/ticket_service.go` | 申告 CRUD、状态机、处理记录、回访 | T24 |
| `internal/service/dashboard_service.go` | 看板统计、趋势数据 | T32 |
| `internal/service/config_service.go` | 系统配置和 embedding 配置管理 | T34 |
| `internal/service/message_service.go` | 站内消息查询、标记已读、未读计数 | T29 |
| `internal/service/scheduler.go` | 后台调度器（TicketAutoCloseJob + MessageNotifyJob） | T30 |
| `internal/handler/auth.go` | 认证接口 Handler | T11 |
| `internal/handler/user.go` | 用户管理接口 Handler | T14 |
| `internal/handler/knowledge.go` | 知识库管理接口 Handler | T18, T19 |
| `internal/handler/chat.go` | 问答接口 Handler | T26 |
| `internal/handler/ticket.go` | 申告管理接口 Handler | T24 |
| `internal/handler/dashboard.go` | 看板接口 Handler | T32 |
| `internal/handler/config.go` | 配置管理接口 Handler | T34 |
| `internal/handler/message.go` | 消息接口 Handler | T29 |
| `internal/handler/audit.go` | 审计日志接口 Handler | T33 |
| `internal/dto/request/audit.go` | 审计日志请求 DTO | T33 |
| `internal/dto/response/audit.go` | 审计日志响应 DTO | T33 |

### 前端 (`web/`)

| 文件路径 | 职责 | 所属任务 |
| --- | --- | --- |
| `package.json` | 项目依赖声明 | T08 |
| `vite.config.ts` | Vite 构建配置、代理配置 | T08 |
| `tsconfig.json` | TypeScript 配置 | T08 |
| `src/main.ts` | Vue 应用入口 | T08 |
| `src/App.vue` | 根组件 | T08 |
| `src/router/index.ts` | Vue Router 路由定义（门户 + 后台 + 公开） | T08 |
| `src/utils/request.ts` | Axios 实例封装（拦截器、token 注入、错误处理） | T08 |
| `src/utils/auth.ts` | Token 存取工具函数 | T08 |
| `src/stores/auth.ts` | 认证状态管理（token、user、roles、permissions、menus） | T16 |
| `src/stores/chat.ts` | 问答状态管理 | T26 |
| `src/stores/app.ts` | 应用全局状态（侧边栏折叠、未读消息数） | T16 |
| `src/styles/global.css` | Linear Design 全局样式（暗色主题、字体、CSS 变量） | T08 |
| `src/api/auth.ts` | 认证 API 封装 | T16 |
| `src/api/chat.ts` | 问答 API 封装 | T26 |
| `src/api/ticket.ts` | 申告 API 封装 | T28 |
| `src/api/knowledge.ts` | 知识库 API 封装 | T22 |
| `src/api/user.ts` | 用户管理 API 封装 | T16 |
| `src/api/dashboard.ts` | 看板 API 封装 | T35 |
| `src/api/message.ts` | 消息 API 封装 | T28 |
| `src/components/layout/AdminLayout.vue` | 后台管理布局（侧边栏 + 顶栏 + 内容区） | T16 |
| `src/components/layout/PortalLayout.vue` | 门户端布局 | T28 |
| `src/components/common/Pagination.vue` | 通用分页组件 | T16 |
| `src/components/common/ConfirmDialog.vue` | 通用确认对话框 | T16 |
| `src/components/common/StatusBadge.vue` | 状态标签组件 | T16 |
| `src/views/auth/Login.vue` | 登录页 | T16 |
| `src/views/auth/ChangePassword.vue` | 修改密码页（首次登录强制） | T16 |
| `src/views/admin/Dashboard.vue` | 数据看板页 | T35 |
| `src/views/admin/TicketList.vue` | 申告列表页 | T28 |
| `src/views/admin/TicketDetail.vue` | 申告处理页（状态操作、处理记录、回访） | T28 |
| `src/views/admin/KnowledgeList.vue` | 知识库列表页 | T22 |
| `src/views/admin/KnowledgeEdit.vue` | 知识编辑页（创建/编辑/审核/发布） | T22 |
| `src/views/admin/UserList.vue` | 用户管理页 | T16 |
| `src/views/admin/RoleManage.vue` | 角色管理页 | T16 |
| `src/views/admin/AuditLog.vue` | 审计日志页 | T35 |
| `src/views/admin/ModelConfig.vue` | 模型配置页（Top K、置信度阈值） | T35 |
| `src/views/admin/EmbeddingConfig.vue` | Embedding 配置页 | T35 |
| `src/views/admin/SystemConfig.vue` | 系统配置页 | T35 |
| `src/views/portal/Chat.vue` | 智能问答页（提问、答案、来源、反馈、转申告） | T28 |
| `src/views/portal/TicketSubmit.vue` | 申告提交页 | T28 |
| `src/views/portal/TicketQuery.vue` | 我的申告列表页 | T28 |
| `src/views/portal/TicketDetail.vue` | 申告详情页（门户端视角 + 补充信息） | T28 |
| `src/views/portal/Messages.vue` | 站内消息页 | T28 |

### 基础设施

| 文件路径 | 职责 | 所属任务 |
| --- | --- | --- |
| `docker-compose.yml` | Docker Compose 编排（本地开发：PostgreSQL；T37 补充完整服务） | T09, T37 |
| `.env` | 环境变量配置（已 gitignore） | T09, T37 |
| `.env.example` | 环境变量模板（提交到版本库） | T09, T37 |
| `server/Dockerfile` | Go 后端镜像构建 | T37 |
| `web/Dockerfile` | Vue 前端镜像构建（nginx） | T37 |
| `Makefile` | 构建和开发命令 | T37 |
| `README.md` | 项目说明文档 | T38 |

### 测试文件

> **所有测试代码统一放在 `server/tests/` 目录下**，按子目录组织（外部测试包），仅测试导出 API。
> 运行方式：`go test ./tests/... -v`（非集成），`go test ./tests/... -tags=integration -v`（含集成测试）。

| 文件路径 | 验证范围 | 所属任务 |
| --- | --- | --- |
| `server/tests/pkg/jwt_test.go` | JWT 生成/解析/过期/刷新 | T05 |
| `server/tests/pkg/hash_test.go` | bcrypt 哈希/验证/密码策略正则 | T05 |
| `server/tests/pkg/response_test.go` | 统一响应格式输出 | T05 |
| `server/tests/middleware/cors_test.go` | CORS 允许/禁止来源、预检请求、方法白名单 | T06 |
| `server/tests/middleware/logger_test.go` | 日志字段、JSON 格式、不同状态码 | T06 |
| `server/tests/middleware/request_id_test.go` | 请求 ID 自动生成、客户端透传、Context 写入 | T06 |
| `server/tests/middleware/auth_test.go` | JWT 中间件：有效令牌/过期令牌/缺失/格式错误/无效令牌 | T12 |
| `server/tests/middleware/rbac_test.go` | RBAC 中间件：有权限/无权限/无用户 | T13 ✅ |
| `server/tests/router/router_test.go` | 路由注册骨架、健康检查端点、501 占位响应 | T07 |
| `server/tests/repository/user_repo_test.go` | 用户 Repository：GetByID/GetByUsername/GetByPhone/ExistsByPhone/Create | T10 |
| `server/tests/repository/knowledge_repo_test.go` | 知识库 Repository：KB CRUD/文章分页/Chunk 批量 | T17 |
| `server/tests/repository/ticket_repo_test.go` | 申告 Repository：Ticket CRUD/状态更新/自动关闭/分页 | T23 |
| `server/tests/repository/chat_repo_test.go` | 问答 Repository：Session CRUD/反馈/分页/CreateBatch | T25 |
| `server/tests/adapter/rag_client_test.go` | RagClient：请求构造/响应映射/超时/错误降级 | T20 |
| `server/tests/adapter/storage_client_test.go` | StorageClient：上传/预签名/删除 | T27 |
| `server/tests/service/auth_service_test.go` | 登录成功/密码错误/账号冻结/令牌刷新/修改密码 | T11 |
| `server/tests/service/user_service_test.go` | 用户 CRUD/冻结恢复/密码校验/用户名重复 | T14 ✅ |
| `server/tests/service/role_service_test.go` | 角色 CRUD/菜单绑定 | T15 |
| `server/tests/service/knowledge_service_test.go` | 知识 CRUD/审核流程/发布同步/停用/重试 | T18, T19 |
| `server/tests/service/chat_service_test.go` | 问答创建/置信度判断/低置信度转人工/AI 降级/会话保存 | T26 |
| `server/tests/service/ticket_service_test.go` | 申告 CRUD/状态机转换/补充信息计数/回访/自动关闭 | T24 |
| `server/tests/service/dashboard_service_test.go` | 统计 SQL/趋势数据 | T32 |
| `server/tests/service/config_service_test.go` | 配置 CRUD/embedding 管理 | T34 |
| `server/tests/service/message_service_test.go` | 消息查询/标记已读/未读计数 | T29 |
| `server/tests/service/scheduler_test.go` | 自动关闭 7 天逻辑/消息通知触发 | T30 |
| `server/tests/handler/auth_handler_test.go` | 登录接口集成测试 | T11 |
| `server/tests/handler/chat_test.go` | 问答接口集成测试（含降级场景） | T26 |
| `server/tests/handler/ticket_test.go` | 申告接口集成测试（含状态机） | T24 |
| `server/tests/handler/knowledge_test.go` | 知识管理接口集成测试 | T18, T19 |
| `server/tests/handler/user_handler_test.go` | 用户管理接口集成测试 | T14 ✅ |
| `server/tests/handler/role_handler_test.go` | 角色管理接口集成测试 | T15 |
| `server/tests/database/database_test.go` | 数据库连接和迁移验证 | T03, T04 |
| `server/tests/database/migrate_test.go` | 自动迁移：16 张表创建/列验证 | T04 |
| `server/tests/config/config_test.go` | 配置加载/环境变量覆盖 | T02 |
| `server/tests/model/user_test.go` | User/Role/Menu 模型字段验证 | T04 |
| `server/tests/model/ticket_test.go` | Ticket/TicketRecord 模型字段验证 | T04 |
| `server/tests/model/knowledge_test.go` | KnowledgeBase/Article/Chunk 模型字段验证 | T04 |
| `server/tests/model/embedding_test.go` | EmbeddingConfig 模型字段验证 | T04 |
| `server/tests/model/chat_test.go` | ChatSession/ChatMessage 模型字段验证 | T04 |
| `server/tests/model/audit_test.go` | AuditLog 模型字段验证 | T04 |
| `server/tests/model/system_test.go` | SystemConfig 模型字段验证 | T04 |
| `server/tests/model/message_test.go` | Message 模型字段验证 | T04 |
| `server/tests/model/common_test.go` | PaginateScope 分页工具验证 | T04 |

---

## M1：数据库与后端基础能力（第 1 周）

本里程碑目标：搭建完整的 Go 后端项目骨架，数据库可用，Docker Compose 可一键启动。

---

### Task 01: Go 项目初始化 ✅

**Files:**
- Create: `server/cmd/main.go` ✅
- Create: `server/go.mod` ✅

**说明：**

`go.mod` 声明模块路径为 `opsmind`，Go 版本 1.22+。核心依赖包括：
- `github.com/gin-gonic/gin` — HTTP 框架
- `gorm.io/gorm` + `gorm.io/driver/postgres` — ORM 和 PostgreSQL 驱动
- `github.com/golang-jwt/jwt/v5` — JWT
- `github.com/spf13/viper` — 配置管理
- `github.com/minio/minio-go/v7` — MinIO 客户端
- `golang.org/x/crypto` — bcrypt

`cmd/main.go` 已包含完整的初始化流程（T11 完成后补充了 DB/Repo/Service/Handler/Router 全链路初始化）。

**Verification:** `go build ./cmd/...` 编译通过。

---

### Task 02: 配置管理 ✅

**Files:**
- Create: `server/internal/config/config.go`
- Create: `server/internal/config/config.yaml`

**说明：**

`config.go` 定义以下配置结构体，与 TECH.md §9.3 完全对齐：

- `AppConfig` — 顶层结构体，包含以下字段：
  - `Server`（ServerConfig）— port、mode
  - `Database`（DatabaseConfig）— host、port、user、password、dbname、sslmode
  - `JWT`（JWTConfig）— secret、access_expire、refresh_expire
  - `MinIO`（MinIOConfig）— endpoint、access_key、secret_key、use_ssl
  - `AnythingLLM`（AnythingLLMConfig）— base_url、api_key、default_workspace_slug、timeout_seconds
  - `AI`（AIConfig）— default_top_k、confidence_threshold

- `Load()` 函数使用 Viper 加载 `config.yaml`，并支持环境变量覆盖（`viper.SetEnvPrefix("OPSMIND")`）。环境变量覆盖规则：
  - `DB_HOST` → `Database.Host`
  - `MINIO_ENDPOINT` → `MinIO.Endpoint`
  - `ANYTHINGLLM_BASE_URL` → `AnythingLLM.BaseURL`
  - `ANYTHINGLLM_API_KEY` → `AnythingLLM.APIKey`
  - `JWT_SECRET` → `JWT.Secret`
  - `AI_DEFAULT_TOP_K` → `AI.DefaultTopK`
  - `AI_CONFIDENCE_THRESHOLD` → `AI.ConfidenceThreshold`

`config.yaml` 提供本地开发默认值。AnythingLLM 地址默认为 `http://anythingllm:3001/api`（Docker 内部地址）。

**Verification:** 单元测试验证 `Load()` 能正确读取默认值和环境变量覆盖。

---

### Task 03: 数据库连接 ✅

**Files:**
- Create: `server/internal/database/database.go`
- Test: `server/tests/database/database_test.go`

**说明：**

`database.go` 提供 `Init(cfg config.DatabaseConfig) (*gorm.DB, error)` 函数：
- 使用 `gorm.io/driver/postgres` 构建 DSN
- 配置连接池：`MaxOpenConns=25`、`MaxIdleConns=10`、`ConnMaxLifetime=5min`
- 返回 `*gorm.DB` 实例

`database_test.go` 验证：
- 使用测试数据库配置建立连接
- 验证连接池配置（MaxOpenConns=25）

**Verification:** 连接本地 PostgreSQL 成功。

---

### Task 04: GORM 数据模型 ✅

**Files:**
- Create: `server/internal/model/user.go`
- Create: `server/internal/model/ticket.go`
- Create: `server/internal/model/knowledge.go`
- Create: `server/internal/model/embedding.go`
- Create: `server/internal/model/chat.go`
- Create: `server/internal/model/audit.go`
- Create: `server/internal/model/system.go`
- Create: `server/internal/model/message.go`
- Create: `server/internal/database/migrate.go`

**说明：**

每个模型文件定义对应 TECH.md §4.2 中的表结构。关键要求：

- **user.go** — `User` 结构体：ID、Username、PasswordHash、RealName、Phone、Email、Status（1=正常,2=冻结）、FirstLogin（bool）、CreatedAt、UpdatedAt。同时包含 `Role`（ID、Name、Description、Permissions JSONB、CreatedAt）、`Menu`（ID、Name、Path、Icon、ParentID、SortOrder、Type）、`UserRole` 中间表（UserID、RoleID）、`RoleMenu` 中间表（RoleID、MenuID）。
- **ticket.go** — `Ticket` 结构体：全部字段与 TECH.md tickets 表对齐，含 SupplementCount、Source、ChatContext（JSONB）。`TicketRecord` 结构体：含 Detail（JSONB）字段。
- **knowledge.go** — `KnowledgeBase` 含 RagWorkspaceSlug。`KnowledgeArticle` 含 RagDocumentLocation。`KnowledgeChunk` 含 SyncStatus（varchar(16)，值为 pending/synced/failed/disabled）、SyncError。
- **embedding.go** — `EmbeddingConfig`：含 ModelType（1=API,2=本地）、LocalPath、IsDefault。
- **chat.go** — `ChatSession` 含 Sources（JSONB）、Feedback、DurationMS。`ChatMessage` 含 Sources（JSONB）、Confidence。
- **audit.go** — `AuditLog`：含 Detail（JSONB）、IPAddress。
- **system.go** — `SystemConfig`：Key（唯一）、Value（JSONB）。
- **message.go** — `Message`：含 Type、RelatedType、RelatedID、IsRead。

所有模型实现 `TableName()` 方法返回确切表名。时间字段使用 `time.Time`，GORM 自动管理 `created_at`/`updated_at`。

`migrate.go` 提供 `AutoMigrate(db *gorm.DB) error` 函数，按依赖顺序迁移所有表。注意：`knowledge_chunks.embedding` 列需要先执行 `CREATE EXTENSION IF NOT EXISTS vector`，然后 GORM 自动迁移 `vector(N)` 类型。

**Verification:** 执行 `AutoMigrate` 后，所有表和字段在数据库中存在。查询 `information_schema.columns` 验证字段类型。

---

### Task 05: 公共工具包 ✅

**Files:**
- Create: `server/pkg/response/response.go`
- Create: `server/pkg/errcode/errcode.go`
- Create: `server/pkg/jwt/jwt.go`
- Create: `server/pkg/hash/hash.go`
- Test: `server/tests/pkg/response_test.go`
- Test: `server/tests/pkg/jwt_test.go`
- Test: `server/tests/pkg/hash_test.go`

**说明：**

- **response.go** — 提供 `Success(c *gin.Context, data interface{})` 和 `Error(c *gin.Context, code int, message string)` 函数。输出格式与 TECH.md §5.1 对齐：`{"code": 0, "message": "success", "data": {...}}` 和 `{"code": 10001, "message": "...", "data": null}`。分页响应增加 `total`、`page`、`page_size` 字段。
- **errcode.go** — 定义 TECH.md §5.1 中全部错误码常量：`ErrAuth = 10001`、`ErrForbidden = 10002`、`ErrParam = 10003`、`ErrNotFound = 10004`、`ErrConflict = 10005`、`ErrAIUnavailable = 20001`、`ErrRAGUnavailable = 20002`、`ErrStorageUnavailable = 20003`、`ErrUnknown = 99999`。
- **jwt.go** — 提供 `GenerateAccessToken(userID int64, username string, roles []string, secret string, expire time.Duration) (string, error)`、`GenerateRefreshToken(...)`、`ParseToken(tokenString string, secret string) (*Claims, error)`。Claims 包含 UserID、Username、Roles。
- **hash.go** — 提供 `HashPassword(password string) (string, error)` 使用 bcrypt cost=10、`CheckPassword(hashed, password string) bool`、`ValidatePassword(password string) error` 使用正则 `^(?=.*[a-z])(?=.*[A-Z])(?=.*\d).{8,32}$` 校验密码策略。

**测试覆盖：**
- `response_test.go` — 验证 JSON 输出格式
- `jwt_test.go` — 生成令牌、解析有效令牌、解析过期令牌、刷新令牌
- `hash_test.go` — 哈希和验证匹配、密码策略校验（长度不足、缺少大写/小写/数字）

**Verification:** `go test ./pkg/... -v` 全部通过。

---

### Task 06: 中间件（CORS + 日志） ✅

**Files:**
- Create: `server/internal/middleware/cors.go`
- Create: `server/internal/middleware/logger.go`

**说明：**

- **cors.go** — 使用 `github.com/gin-contrib/cors` 配置 CORS。允许来源 `http://localhost:5173`（开发环境），允许方法 GET/POST/PUT/PATCH/DELETE/OPTIONS，允许 Header Authorization/Content-Type，暴露 Header Content-Length，MaxAge 12 小时。
- **logger.go** — 使用 `log/slog` 记录每个请求：方法、路径、状态码、耗时（ms）、客户端 IP。格式为结构化 JSON。

**Verification:** 启动 Gin 服务，发送请求后在 stdout 看到结构化日志。

---

### Task 07: 路由注册骨架 ✅

**Files:**
- Create: `server/internal/router/router.go`
- Create: `server/internal/router/portal.go`
- Create: `server/internal/router/admin.go`
- Modify: `server/cmd/main.go`

**说明：**

- **router.go** — `Setup(cfg *config.AppConfig, h *Handlers) *gin.Engine` 函数：
  - 创建 Gin 实例（release/debug 模式由配置决定）
  - 注册 CORS 中间件、Logger 中间件、RequestID 中间件、Recovery 中间件
  - 挂载 `/api/v1/auth` 公开路由组（已绑定 AuthHandler，或使用占位 Handler）
  - 挂载 `/api/v1/portal` 路由组（已应用 `middleware.JWTAuth`）
  - 挂载 `/api/v1/admin` 路由组（已应用 `middleware.JWTAuth`）
  - 返回 engine

- **portal.go** — 注册门户端路由，占位 Handler 函数（返回 501 Not Implemented）。JWT 认证在 router.go 中统一应用。

- **admin.go** — 注册后台管理路由。**已实现的路由绑定真实 Handler（T14 用户管理、T15 角色管理），未实现的仍用占位。** 
  - 用户管理：GET/POST/PUT/PATCH `/users/...`（已绑定 UserHandler，含 RBAC）
  - 角色权限：GET/POST/GET/PUT/DELETE `/roles/...`（已绑定 RoleHandler，含 RBAC）
  - 其他路由（申告、知识库、看板、配置等）使用占位 Handler
  - 操作日志：GET `/audit-logs`
  - 系统配置：GET/PUT `/configs/:key`、GET/POST/PUT `/embedding-configs/...`

- **main.go** — 调用 `config.Load()`、`database.Init()`、`router.Setup()`、`http.ListenAndServe()`。

**Verification:** 启动服务后，`curl http://localhost:8080/api/v1/auth/login` 返回 501（占位）。

---

### Task 08: Vue 前端项目初始化 ✅

**Files:**
- Create: `web/package.json`
- Create: `web/vite.config.ts`
- Create: `web/tsconfig.json`
- Create: `web/src/main.ts`
- Create: `web/src/App.vue`
- Create: `web/src/router/index.ts`
- Create: `web/src/utils/request.ts`
- Create: `web/src/utils/auth.ts`
- Create: `web/src/styles/global.css`

**说明：**

- **package.json** — Vue 3.4+ / Vite / TypeScript / Radix Vue / Pinia / Vue Router / Axios。`devDependencies` 包含 `@vitejs/plugin-vue`、`typescript`。
- **vite.config.ts** — 配置 Vue 插件、API 代理 `/api → http://localhost:8080`、resolve alias `@ → src/`。
- **tsconfig.json** — strict 模式，paths 配置 `@/*`。
- **main.ts** — 创建 Vue app，注册 Pinia、Router、全局样式。
- **App.vue** — `<router-view />`。
- **router/index.ts** — 路由定义与 TECH.md §6.1 对齐：公开路由（/login）、门户路由（/portal/*，需要登录 + 报障人角色）、后台路由（/admin/*，需要登录 + 对应角色权限）。路由守卫检查 token 有效性、首次登录强制跳转修改密码页、角色权限校验。
- **utils/request.ts** — 创建 Axios 实例，baseURL 为空（通过 Vite proxy），请求拦截器注入 `Authorization: Bearer <token>`，响应拦截器处理 401（跳转登录）、403（提示无权限）、统一提取 `data`。
- **utils/auth.ts** — `getToken()`/`setToken()`/`removeToken()` 操作 localStorage。
- **styles/global.css** — Linear Design 暗色主题 CSS 变量（参考 DESIGN-linear.app.md）：
  - `--bg-base: #08090a`、`--bg-panel: #0f1011`、`--bg-overlay: #191a1b`
  - `--border-default: rgba(255,255,255,0.05)`、`--border-hover: rgba(255,255,255,0.08)`
  - `--accent: #5e6ad2`、`--accent-interactive: #7170ff`、`--accent-hover: #828fff`
  - `--text-primary: #e8e8e8`、`--text-secondary: #8a8a8a`
  - 字体：Inter Variable，OpenType 特性 `"cv01", "ss03"`
  - 强调字重：510

**Verification:** `npm run dev` 启动开发服务器，浏览器访问 `http://localhost:5173` 显示空白页（无报错）。

---

## M2：账号权限与后台框架（第 2 周）

本里程碑目标：完整的认证授权体系、用户角色管理、Vue 后台管理框架。

---

### Task 09: 系统配置 Repository ✅

**Files:**
- Create: `server/internal/repository/config_repo.go` ✅
- Test: `server/tests/service/config_service_test.go` ✅

**说明：**

`config_repo.go` 提供 SystemConfig 的 CRUD 操作：
- `GetByKey(key string) (*SystemConfig, error)` — 按 key 查询配置
- `Upsert(key string, value interface{}, updatedBy int64) error` — 更新或插入配置
- `List() ([]SystemConfig, error)` — 列出全部配置

测试验证：按 key 查询、更新已有配置、插入新配置。

---

### Task 10: 用户 Repository ✅

**Files:**
- Create: `server/internal/repository/user_repo.go` ✅
- Test: `server/tests/repository/user_repo_test.go` ✅

**说明：**

提供 User、Role、Menu、UserRole、RoleMenu 的数据访问方法：

**User 相关：**
- `Create(user *User) error`
- `FindByID(id int64) (*User, error)`
- `FindByUsername(username string) (*User, error)`
- `Update(user *User) error`
- `List(page, pageSize int) ([]User, int64, error)` — 分页查询，返回总数
- `UpdateStatus(id int64, status int) error` — 冻结/恢复

**Role 相关：**
- `CreateRole(role *Role) error`
- `FindRoleByID(id int64) (*Role, error)`
- `UpdateRole(role *Role) error`
- `ListRoles() ([]Role, error)`

**Menu 相关：**
- `ListMenus() ([]Menu, error)`
- `UpdateRoleMenus(roleID int64, menuIDs []int64) error` — 先删后插

**UserRole/RoleMenu 相关：**
- `AssignRoles(userID int64, roleIDs []int64) error` — 先删后插
- `GetUserRoles(userID int64) ([]Role, error)`
- `GetRoleMenus(roleID int64) ([]Menu, error)`
- `GetUserPermissions(userID int64) ([]string, error)` — 聚合角色的 permissions JSONB

---

### Task 11: 认证 Service + Handler ✅

**Files:**
- Create: `server/internal/dto/request/auth.go` ✅
- Create: `server/internal/dto/response/auth.go` ✅
- Create: `server/internal/service/auth_service.go` ✅
- Create: `server/internal/handler/auth.go` ✅
- Modify: `server/internal/repository/user_repo.go`（添加 Update 方法）✅
- Test: `server/tests/service/auth_service_test.go` ✅
- Test: `server/tests/handler/auth_handler_test.go` ✅

**说明：**

**dto/request/auth.go** — 定义请求结构体：
- `LoginRequest` — Username、Password（binding required）
- `RefreshRequest` — RefreshToken（binding required）
- `ChangePasswordRequest` — OldPassword、NewPassword（binding required）

**dto/response/auth.go** — 定义响应结构体：
- `LoginResponse` — AccessToken、RefreshToken、User（UserInfo）、Roles（[]string）、Permissions（[]string）、Menus（[]MenuItem）
- `UserInfo` — ID、Username、RealName、Phone、Email、FirstLogin
- `MenuItem` — ID、Name、Path、Icon、ParentID、SortOrder、Type、Children

**auth_service.go** — `AuthService` 结构体依赖 UserRepository。方法：
- `Login(username, password string) (*LoginResponse, error)` — 查用户、bcrypt 校验、检查状态（冻结返回 10002）、生成 access_token + refresh_token、组装返回（含角色、权限、菜单）
- `RefreshToken(refreshToken string) (*LoginResponse, error)` — 解析 refresh_token、重新生成令牌对
- `ChangePassword(userID int64, oldPwd, newPwd string) error` — 校验旧密码、校验新密码策略（正则）、更新密码哈希、设置 `first_login=false`

**auth.go Handler** — 绑定 Gin HandlerFunc：
- `POST /login` — 绑定 LoginRequest，调用 AuthService.Login，返回 LoginResponse
- `POST /refresh` — 绑定 RefreshRequest，调用 AuthService.RefreshToken
- `POST /change-password` — 从 context 取 userID，调用 AuthService.ChangePassword
- `POST /logout` — MVP 阶段返回成功（无状态 JWT，客户端清除 token 即可）

**T15 完成后补充：** `buildLoginResponse` 已更新为查询真实角色、权限和菜单树（系统管理员自动获得全部菜单，非管理员按角色菜单聚合）。

**测试覆盖：**
- `auth_service_test.go` — 使用 mock UserRepository：登录成功、密码错误、账号冻结、刷新令牌、修改密码（旧密码错误、新密码不符合策略）
- `auth_handler_test.go` — 使用 httptest：POST /login 正常返回、参数缺失返回 400

---

### Task 12: JWT 认证中间件 ✅

**Files:**
- Create: `server/internal/middleware/auth.go` ✅
- Test: `server/tests/middleware/auth_test.go` ✅

**说明：**

`auth.go` — `JWTAuth(secret string) gin.HandlerFunc`：
- 从 `Authorization: Bearer <token>` 提取令牌
- 调用 `pkg/jwt.ParseToken` 解析
- 失败返回 10001（401）
- 成功将 UserID、Username、Roles 写入 `c.Set("currentUser", ...)`

**测试覆盖：**
- 有效令牌 → context 中有 currentUser
- 过期令牌 → 返回 401
- 缺失 Authorization → 返回 401
- 格式错误（无 Bearer 前缀）→ 返回 401

---

### Task 13: RBAC 权限中间件 ✅

**Files:**
- Create: `server/internal/middleware/rbac.go` ✅
- Test: `server/tests/middleware/rbac_test.go` ✅

**说明：**

`rbac.go` — `RequirePermission(permissions ...string) gin.HandlerFunc`：
- 从 `c.Get("currentUser")` 获取用户信息
- 查询用户权限列表（可缓存到 context）
- 检查是否包含目标权限
- 系统管理员角色（`admin`）自动放行
- 无权限返回 10002（403）

**测试覆盖：**
- 有目标权限 → 放行 ✅
- 无目标权限 → 返回 403 ✅
- 系统管理员 → 自动放行 ✅

---

### Task 14: 用户管理 Service + Handler ✅

**Files:**
- Create: `server/internal/dto/request/user.go`
- Create: `server/internal/dto/response/user.go`
- Create: `server/internal/service/user_service.go` ✅
- Create: `server/internal/handler/user.go` ✅
- Test: `server/tests/service/user_service_test.go` ✅
- Test: `server/tests/handler/user_handler_test.go` ✅

**说明：**

**dto/request/user.go** ✅ — `CreateUserRequest`（Username、Password、RealName、Phone、Email、RoleIDs）、`UpdateUserRequest`（RealName、Phone、Email、RoleIDs）。

**dto/response/user.go** ✅ — `UserListResponse`（含分页）、`UserDetailResponse`（含角色列表）。

**user_repo.go** ✅ — 补充了 `List`、`UpdateStatus`、`ExistsByUsername`、`GetUserRoles`、`AssignRoles`、`ListMenus`、`GetRoleMenus`、`UpdateRoleMenus`、`GetUserPermissions` 等方法。

**user_service.go** — 方法：
- `Create(req CreateUserRequest) error` ✅ — 校验用户名唯一（冲突返回 10005）、校验密码策略、bcrypt 哈希、创建用户、分配角色
- `Update(id int64, req UpdateUserRequest) error` ✅ — 更新基本信息、重新分配角色
- `Freeze(id int64) error` ✅ — 设置 status=2
- `Restore(id int64) error` ✅ — 设置 status=1
- `List(page, pageSize int, keyword string) (*UserListResponse, error)` ✅ — 分页+搜索，返回含角色名的详情列表
- `GetByID(id int64) (*UserDetailResponse, error)` ✅ — 查询用户详情+角色列表

**user.go Handler** ✅ — 绑定全部 CRUD 端点（Create、GetByID、List、Update、Freeze、Restore）。

**测试覆盖：**
- 创建用户成功、用户名重复返回 10005、密码不符合策略
- 冻结/恢复用户
- 分页列表

---

### Task 15: 角色管理 Service + Handler ✅

**Files:**
- Create: `server/internal/repository/role_repo.go` ✅
- Create: `server/internal/service/role_service.go` ✅
- Create: `server/internal/handler/role.go` ✅
- Create: `server/internal/dto/request/role.go` ✅
- Test: `server/tests/service/role_service_test.go` ✅
- Test: `server/tests/handler/role_handler_test.go` ✅

**说明：**

`role_service.go` — 方法：
- `Create(name, description string, permissions []string) error`
- `GetByID(id int64) (*Role, error)`
- `List(page, pageSize int) ([]Role, int64, error)`
- `Update(id int64, name, description string, permissions []string) error`
- `Delete(id int64) error`

**测试覆盖：** 角色 CRUD（创建成功/重复/查询/列表/更新/删除） ✅

---

### Task 16: Vue 后台框架 + 登录页面 ✅

**Files:**
- Create: `src/stores/auth.ts` ✅
- Create: `src/stores/app.ts` ✅
- Create: `src/api/auth.ts` ✅
- Create: `src/api/user.ts` ✅
- Create: `src/components/layout/AdminLayout.vue` ✅
- Create: `src/components/common/Pagination.vue` ✅
- Create: `src/components/common/ConfirmDialog.vue` ✅
- Create: `src/components/common/StatusBadge.vue` ✅
- Create: `src/views/auth/Login.vue` ✅
- Create: `src/views/auth/ChangePassword.vue` ✅
- Create: `src/views/admin/UserList.vue`
- Create: `src/views/admin/RoleManage.vue`

**说明：**

- **stores/auth.ts** — Pinia store：state 含 token、user、roles、permissions、menus。actions 含 login、refresh、logout、fetchUserInfo。getters 含 isLoggedIn、hasPermission。
- **stores/app.ts** — state 含 sidebarCollapsed、unreadMessageCount。
- **api/auth.ts** — 封装 POST /login、/refresh、/change-password、/logout。
- **api/user.ts** — 封装用户管理全部 API。
- **AdminLayout.vue** — 后台管理布局：左侧动态菜单（根据 menus 渲染）、顶部栏（用户名、未读消息、退出）、内容区 `<router-view />`。遵循 Linear Design 暗色主题。
- **Pagination.vue** — 通用分页组件：上一页/下一页、页码、每页条数选择。
- **ConfirmDialog.vue** — 基于 Radix Vue Dialog 的确认弹窗。
- **StatusBadge.vue** — 根据 status 值显示不同颜色的标签。
- **Login.vue** — 登录表单（用户名 + 密码），登录成功后检查 `first_login`，首次登录跳转修改密码页。
- **ChangePassword.vue** — 修改密码表单（旧密码 + 新密码 + 确认密码），前端校验密码策略。
- **UserList.vue** — 用户列表表格（分页、搜索、新增、编辑、冻结/恢复按钮）。新增/编辑弹窗包含角色多选。
- **RoleManage.vue** — 角色列表 + 新增/编辑弹窗 + 菜单权限树形选择。

**Verification:** 登录 → 跳转后台 → 侧边栏显示菜单 → 用户管理页面可操作。

---

## M3：知识库管理与 AI 服务（第 3 周）

本里程碑目标：知识库 CRUD、审核发布流程、AnythingLLM 适配器、embedding 配置管理。

---

### Task 17: 知识库 Repository ✅

**Files:**
- Create: `server/internal/repository/knowledge_repo.go`

**说明：**

提供 KnowledgeBase、KnowledgeArticle、KnowledgeChunk、EmbeddingConfig 的数据访问方法：

**KnowledgeBase：**
- `CreateKB(kb *KnowledgeBase) error`
- `FindKBByID(id int64) (*KnowledgeBase, error)`
- `UpdateKB(kb *KnowledgeBase) error`
- `ListKBs() ([]KnowledgeBase, error)`

**KnowledgeArticle：**
- `CreateArticle(article *KnowledgeArticle) error`
- `FindArticleByID(id int64) (*KnowledgeArticle, error)` — 预加载 KnowledgeBase
- `UpdateArticle(article *KnowledgeArticle) error`
- `ListArticles(kbID int64, status int, page, pageSize int) ([]KnowledgeArticle, int64, error)`
- `UpdateArticleStatus(id int64, status int) error`

**KnowledgeChunk：**
- `CreateChunks(chunks []KnowledgeChunk) error` — 批量创建
- `UpdateChunkSyncStatus(articleID int64, status string, syncError string) error`
- `FindChunksByArticleID(articleID int64) ([]KnowledgeChunk, error)`
- `UpdateChunkStatusByArticleID(articleID int64, status string) error`

**EmbeddingConfig：**
- `CreateEmbeddingConfig(cfg *EmbeddingConfig) error`
- `UpdateEmbeddingConfig(cfg *EmbeddingConfig) error`
- `ListEmbeddingConfigs() ([]EmbeddingConfig, error)`
- `GetDefaultEmbeddingConfig() (*EmbeddingConfig, error)`

---

### Task 18: 知识库 Service + Handler（CRUD + 审核） ✅

**Files:**
- Create: `server/internal/dto/request/knowledge.go`
- Create: `server/internal/dto/response/knowledge.go`
- Create: `server/internal/service/knowledge_service.go`
- Create: `server/internal/handler/knowledge.go`
- Test: `server/tests/service/knowledge_service_test.go`
- Test: `server/tests/handler/knowledge_test.go`

**说明：**

**dto/request/knowledge.go** — 请求结构体：
- `CreateKBRequest` — Name、Description
- `UpdateKBRequest` — Name、Description
- `CreateArticleRequest` — KBID、Question、Answer、Category、Tags
- `UpdateArticleRequest` — Question、Answer、Category、Tags
- `ReviewRequest` — Approved（bool）、ReviewComment

**dto/response/knowledge.go** — 响应结构体：
- `KBListResponse`、`ArticleListResponse`（分页）、`ArticleDetailResponse`（含 chunks 和同步状态）

**knowledge_service.go** — 方法：
- `CreateKB(req CreateKBRequest, userID int64) error` — 创建知识库，同时调用 `RagClient.CreateWorkspace` 生成 AnythingLLM workspace，保存 `rag_workspace_slug`
- `UpdateKB(id int64, req UpdateKBRequest) error`
- `ListKBs() ([]KnowledgeBase, error)`
- `CreateArticle(req CreateArticleRequest, userID int64) error` — 创建草稿（status=1）
- `UpdateArticle(id int64, req UpdateArticleRequest) error` — 仅草稿/驳回状态可编辑
- `SubmitReview(id int64) error` — 草稿→待审核（status=2）
- `Review(id int64, reviewerID int64, req ReviewRequest) error` — 校验审核人≠创建人；通过→status=3；驳回→status=5+review_comment
- `Publish(id int64, publisherID int64) error` — 调用 `RagClient.SyncDocument` 同步到 AnythingLLM + 更新 sync_status + 写审计日志
- `Disable(id int64) error` — 调用 `RagClient.DisableDocument` + 更新 sync_status='disabled' + 写审计日志
- `RetrySync(id int64) error` — 重新执行 Publish 逻辑
- `ListArticles(kbID int64, status int, page, pageSize int) (*ArticleListResponse, error)`
- `GetArticleDetail(id int64) (*ArticleDetailResponse, error)`

**knowledge.go Handler** — 绑定全部知识库管理 API 端点。

**测试覆盖：**
- 知识库 CRUD
- 文章创建→提交审核→审核通过→发布（mock RagClient）
- 审核人=创建人被拒绝
- 驳回时必须填写 review_comment
- 发布同步失败→sync_status='failed'
- 重试同步
- 停用知识

---

### Task 19: Embedding 配置管理 ✅

**Files:**
- Modify: `server/internal/service/knowledge_service.go`（增加 EmbeddingConfig 相关方法）
- Modify: `server/internal/handler/knowledge.go`（增加 EmbeddingConfig 端点）

**说明：**

在 knowledge_service.go 中增加：
- `CreateEmbeddingConfig(req CreateEmbeddingConfigRequest) error` — 校验 model_type=1 时 api_endpoint 必填，model_type=2 时 local_path 必填；如果 is_default=true，先将其他配置的 is_default 设为 false
- `UpdateEmbeddingConfig(id int64, req UpdateEmbeddingConfigRequest) error`
- `ListEmbeddingConfigs() ([]EmbeddingConfig, error)`

在 knowledge.go Handler 中增加 embedding-configs 端点处理。

**测试覆盖：**
- 创建 API 类型 embedding 配置
- 创建本地类型 embedding 配置
- 设为默认时其他配置取消默认

---

### Task 20: AnythingLLM RagClient 适配器 ✅

**Files:**
- Create: `server/internal/adapter/rag_client.go`
- Test: `server/tests/adapter/rag_client_test.go`

**说明：**

`rag_client.go` 定义 `RagClient` 接口和 `anythingLLMClient` 实现：

**接口（与 TECH.md §7.1 完全对齐）：**
- `Query(ctx, req RAGQueryRequest) (*RAGQueryResponse, error)` — POST `/api/v1/workspace/{slug}/chat`
- `SyncDocument(ctx, req RAGSyncRequest) (*RAGSyncResponse, error)` — 结构化 FAQ 调用 POST `/api/v1/document/raw-text`；文件类调用 POST `/api/v1/document/upload`
- `DisableDocument(ctx, req RAGDisableRequest) error` — POST `/api/v1/workspace/{slug}/update-embeddings`，deletes 传入 document_locations
- `CreateWorkspace(ctx, req RAGCreateWorkspaceRequest) (*RAGCreateWorkspaceResponse, error)` — POST `/api/v1/workspace/new`

**实现要点：**
- 使用 `net/http` 客户端，超时由配置 `AnythingLLM.TimeoutSeconds` 控制
- 请求头：`Authorization: Bearer <api_key>`、`Content-Type: application/json`
- 基础 URL 从配置读取，默认 `http://anythingllm:3001/api`
- 字段映射与集成方案 §5.3 对齐：`textResponse→answer`、`sources[].title→doc_name`、`sources[].text→chunk_content`、`max(sources[].score)→confidence`
- 错误处理：AnythingLLM 返回 `error != null` 时，`RAGQueryResponse.Error` 非空

**测试覆盖（使用 httptest mock AnythingLLM）：**
- Query 正常响应 → 字段映射正确
- Query 返回 error → Error 字段非空
- Query 网络超时 → 返回错误
- SyncDocument raw text → 请求体构造正确
- SyncDocument 文件上传 → multipart 格式正确
- DisableDocument → deletes 字段正确
- CreateWorkspace → slug 返回正确

---

### Task 21: 后端 main.go 完善 M1-M3 初始化 ✅

**Files:**
- Modify: `server/cmd/main.go`

**说明：**

完善 main.go 初始化流程：
1. 加载配置 `config.Load()`
2. 初始化数据库 `database.Init(cfg.Database)`
3. 自动迁移 `database.AutoMigrate(db)`
4. 初始化 Adapter：创建 RagClient 实例（传入 AnythingLLM 配置）
5. 初始化 Repository：创建各 Repository 实例（传入 db）
6. 初始化 Service：创建各 Service 实例（传入 Repository 和 Adapter）
7. 初始化 Handler：创建各 Handler 实例（传入 Service）
8. 设置路由 `router.Setup(cfg, handlerSet)`
9. 启动 HTTP 服务

**Verification:** `go build ./cmd/...` 编译通过。

---

### Task 22: Vue 知识库管理页面 ✅

**Files:**
- Create: `src/api/knowledge.ts`
- Create: `src/views/admin/KnowledgeList.vue`
- Create: `src/views/admin/KnowledgeEdit.vue`

**说明：**

- **api/knowledge.ts** — 封装知识库和知识条目全部 API。包含 embedding 配置 API。
- **KnowledgeList.vue** — 左侧知识库列表 + 右侧知识条目列表。条目列表显示同步状态（pending/synced/failed/disabled）和失败原因。操作按钮：新增、编辑、提交审核、停用、重试同步。
- **KnowledgeEdit.vue** — 知识编辑表单：所属知识库（下拉）、问题、答案（富文本或多行文本）、分类、标签（标签输入）。审核区域：审核意见输入、通过/驳回按钮。

**Verification:** 创建知识条目→提交审核→审核通过→发布→同步状态显示 synced。

---

## M4：智能问答与申告处理（第 4-5 周）

本里程碑目标：问答会话 API、RAG 集成、置信度判断、申告 CRUD、状态机、处理记录、回访、站内消息。

---

### Task 23: 申告 Repository ✅

**Files:**
- Create: `server/internal/repository/ticket_repo.go`

**说明：**

提供 Ticket、TicketRecord 的数据访问方法：

**Ticket：**
- `Create(ticket *Ticket) error`
- `FindByID(id int64) (*Ticket, error)` — 预加载 TicketRecords 和 User
- `Update(ticket *Ticket) error`
- `UpdateStatus(id int64, status int) error`
- `IncrementSupplementCount(id int64) error`
- `ListByUser(userID int64, page, pageSize int) ([]Ticket, int64, error)`
- `ListAll(status int, urgency int, page, pageSize int) ([]Ticket, int64, error)` — 支持筛选
- `AutoCloseTickets(olderThan time.Time) (int64, error)` — 批量关闭超时申告，返回关闭数量

**TicketRecord：**
- `CreateRecord(record *TicketRecord) error`
- `FindByTicketID(ticketID int64) ([]TicketRecord, error)` — 按时间正序

---

### Task 24: 申告 Service + Handler ✅

**Files:**
- Create: `server/internal/dto/request/ticket.go`
- Create: `server/internal/dto/response/ticket.go`
- Create: `server/internal/service/ticket_service.go`
- Create: `server/internal/handler/ticket.go`
- Test: `server/tests/service/ticket_service_test.go`
- Test: `server/tests/handler/ticket_test.go`

**说明：**

**dto/request/ticket.go** — 请求结构体：
- `CreateTicketRequest` — Title、Description、Urgency（1-3）、ImpactScope（1-3）、AffectedSystems、ContactPhone、ContactEmail、ChatContext（可选，从问答转申告时带入）
- `SupplementTicketRequest` — Content（补充内容）
- `UpdateTicketStatusRequest` — Action（start/request_info/resolve/close）、Result、ToKnowledgeCandidate（bool）
- `CreateTicketRecordRequest` — Action、Content、Detail（JSONB，回访结果等）

**dto/response/ticket.go** — 响应结构体：
- `TicketListResponse`（分页）、`TicketDetailResponse`（含 records、submitter info）

**ticket_service.go** — 方法：
- `CreateTicket(req CreateTicketRequest, userID int64) error` — 生成编号 `TK-YYYYMMDD-XXXX`（XXXX 为当日自增序号）、创建申告（status=1, source=门户提交或问答转申告）、写审计日志
- `SupplementTicket(id int64, userID int64, req SupplementTicketRequest) error` — 校验申告属于当前用户、校验状态为"需补充信息"(3)、创建记录（action=supplement）、更新状态为"处理中"(2)
- `UpdateStatus(id int64, operatorID int64, req UpdateTicketStatusRequest) error` — **状态机校验**（与 TECH.md §5.3 action 表完全对齐）：
  - `start`: 待处理(1)→处理中(2)，仅运维人员
  - `request_info`: 处理中(2)→需补充信息(3)，supplement_count+1，超过 3 次禁止
  - `resolve`: 处理中(2)→已解决(4)，写入回访 detail
  - `close`: 任意→已关闭(5)
  - 创建 TicketRecord、写审计日志
  - `request_info` 时触发 MessageNotifyJob（写站内消息）
- `AddRecord(id int64, operatorID int64, req CreateTicketRecordRequest) error` — 添加处理记录（不影响状态）
- `ListByUser(userID int64, page, pageSize int) (*TicketListResponse, error)`
- `ListAll(status, urgency, page, pageSize int) (*TicketListResponse, error)`
- `GetDetail(id int64) (*TicketDetailResponse, error)`

**ticket.go Handler** — 绑定全部申告 API 端点。申告附件上传使用 `StorageClient.Upload` 存储到 `opsmind-attachments` bucket，附件 URL 保存到申告记录。

**测试覆盖：**
- 创建申告→编号生成正确
- 状态机全部转换路径：待处理→处理中→需补充信息→处理中→已解决
- request_info 超过 3 次被拒绝
- supplement 操作仅申告人可执行
- 7 天自动关闭（mock time）
- 回访 detail 结构正确

---

### Task 25: 问答 Repository ✅

**Files:**
- Create: `server/internal/repository/chat_repo.go` ✅
- Test: `server/tests/repository/chat_repo_test.go` ✅

**说明：**

提供 ChatSession、ChatMessage 的数据访问方法：

**ChatSession：**
- `Create(session *ChatSession) error`
- `FindByID(id int64) (*ChatSession, error)`
- `UpdateFeedback(id int64, feedback string) error`
- `ListByUser(userID int64, page, pageSize int) ([]ChatSession, int64, error)`

**ChatMessage：**
- `CreateBatch(messages []ChatMessage) error` — 批量创建（用户问题 + AI 回答）

---

### Task 26: 智能问答 Service + Handler ✅

**Files:**
- Create: `server/internal/dto/request/chat.go` ✅
- Create: `server/internal/dto/response/chat.go` ✅
- Create: `server/internal/service/chat_service.go` ✅
- Create: `server/internal/handler/chat.go` ✅
- Create: `web/src/stores/chat.ts` ✅
- Create: `web/src/api/chat.ts` ✅
- Modify: `web/src/views/portal/Chat.vue` ✅
- Test: `server/tests/service/chat_service_test.go` ✅
- Test: `server/tests/handler/chat_test.go` ✅

**说明：**

**dto/request/chat.go** — `CreateChatRequest`：Question（required）、KBID（required）。

**dto/response/chat.go** — `ChatSessionResponse`：SessionID、Answer、Sources（[]SourceItem）、Confidence、CanSubmitTicket、DurationMS。`SourceItem`：DocName、ChunkContent、Confidence。`FeedbackRequest`：Feedback（resolved/unresolved）。

**chat_service.go** — 方法：
- `CreateChatSession(req CreateChatRequest, userID int64) (*ChatSessionResponse, error)` — 核心流程：
  1. 查询 KnowledgeBase 获取 rag_workspace_slug
  2. 调用 `RagClient.Query`（传入 workspace_slug、question、top_k）
  3. 处理返回结果：
     - AnythingLLM 返回 error != null → 记录错误，返回转人工提示（code=20001 场景）
     - AnythingLLM 不可达 → 返回 code=20001
     - sources 为空或 confidence < threshold（默认 0.6）→ 返回兜底答案 + can_submit_ticket=true
     - 正常且置信度达标 → 返回答案和来源 + can_submit_ticket=false
  4. 保存 ChatSession 和 ChatMessage（用户问题 + AI 回答）
  5. 记录 duration_ms
- `SubmitFeedback(sessionID int64, feedback string) error` — 更新 feedback 字段
- `GetChatDetail(sessionID int64) (*ChatSessionResponse, error)`

**降级兜底文本（与集成方案 §7.1 对齐）：**
- AI 服务不可用：`当前 AI 服务暂不可用，请提交申告由人工处理`
- 低置信度：`暂未找到足够匹配的知识，建议提交申告由运维人员人工处理`

**chat.go Handler** — 绑定门户端问答 API。

**前端文件：**
- **stores/chat.ts** — state 含 currentSession、messages、loading。actions 含 createSession、submitFeedback。
- **api/chat.ts** — 封装 POST /portal/chat-sessions、POST feedback。
- **Chat.vue** — 智能问答页面：
  - 顶部知识库选择（下拉）
  - 问答区域：用户问题气泡 + AI 回答气泡（含来源文档列表）
  - 底部输入框（多行 + 提交按钮）
  - 回答下方：反馈按钮（已解决/未解决）
  - 未解决时展示"提交申告"按钮，点击跳转申告提交页并自动带入问题和问答上下文
  - can_submit_ticket=true 时展示引导文案
  - 遵循 Linear Design 暗色主题

**测试覆盖：**
- 正常问答→答案和来源返回正确
- 低置信度→can_submit_ticket=true
- AnythingLLM 返回 error→降级处理
- AnythingLLM 不可达→code=20001
- 反馈提交成功

---

### Task 27: MinIO StorageClient 适配器 ✅

**Files:**
- Create: `server/internal/adapter/storage_client.go` ✅
- Test: `server/tests/adapter/storage_client_test.go` ✅

**说明：**

`storage_client.go` 定义 `StorageClient` 接口和 `minioClient` 实现（与 TECH.md §7.3 对齐）：
- `Upload(ctx, bucket, key string, reader io.Reader, size int64, contentType string) (string, error)` — 上传文件，返回对象 key
- `GetPresignedURL(ctx, bucket, key string, expiry time.Duration) (string, error)` — 获取预签名下载 URL
- `Delete(ctx, bucket, key string) error` — 删除对象

Bucket 规划：`opsmind-attachments`（申告附件）、`opsmind-documents`（知识文档原件）。初始化时自动创建 bucket。

**测试覆盖（使用 minio mock 或测试 bucket）：**
- 上传文件成功
- 获取预签名 URL
- 删除文件

---

### Task 28: Vue 门户端页面 ✅

**Files:**
- Modify: `web/src/components/layout/PortalLayout.vue` ✅
- Create: `web/src/api/ticket.ts` ✅
- Create: `web/src/api/message.ts` ✅
- Modify: `web/src/views/portal/TicketSubmit.vue` ✅
- Modify: `web/src/views/portal/TicketQuery.vue` ✅
- Modify: `web/src/views/portal/TicketDetail.vue` ✅
- Modify: `web/src/views/portal/Messages.vue` ✅
- Test: `web/src/api/__tests__/ticket.test.ts` ✅
- Test: `web/src/api/__tests__/message.test.ts` ✅

**说明：**

- **PortalLayout.vue** — 门户端布局：顶部导航栏（Logo、导航链接：智能问答/提交申告/我的申告/消息）、内容区。简洁风格，与后台管理布局区分。
- **api/ticket.ts** — 封装门户端申告 API（创建、列表、详情、补充信息）。
- **api/message.ts** — 封装消息 API（列表、标记已读、未读数）。
- **TicketSubmit.vue** — 申告提交表单：标题、描述（多行）、紧急程度（低/中/高）、影响范围（个人/部门/全公司）、受影响系统（标签输入）、联系手机号（必填）、联系邮箱（选填）。支持从问答页带入上下文。
- **TicketQuery.vue** — 我的申告列表：表格展示（编号、标题、状态、紧急程度、创建时间），状态标签颜色区分。
- **TicketDetail.vue** — 申告详情：基本信息 + 处理记录时间线 + 补充信息入口（仅"需补充信息"状态可用）。
- **Messages.vue** — 站内消息列表：未读高亮、点击标记已读、关联申告跳转。

**Verification:** 门户端完整流程：提问→转申告→查看申告→补充信息→查看消息。

---

### Task 29: 站内消息 Service + Handler ✅

**Files:**
- Create: `server/internal/repository/message_repo.go` ✅
- Create: `server/internal/service/message_service.go` ✅
- Create: `server/internal/handler/message.go` ✅
- Test: `server/tests/service/message_service_test.go` ✅

**说明：**

**message_repo.go** — 数据访问：
- `Create(msg *Message) error`
- `ListByUser(userID int64, page, pageSize int) ([]Message, int64, error)`
- `MarkAsRead(id int64) error`
- `CountUnread(userID int64) (int64, error)`

**message_service.go** — 方法：
- `ListMessages(userID int64, page, pageSize int) ([]Message, int64, error)`
- `MarkAsRead(id int64) error`
- `CountUnread(userID int64) (int64, error)`
- `NotifySupplement(ticketID int64, userID int64) error` — 被 TicketService 调用，写入 type=ticket_supplement 的消息

**message.go Handler** — 绑定门户端消息 API。

**测试覆盖：**
- 消息创建和查询
- 标记已读
- 未读计数

---

### Task 30: 后台调度器 ✅

**Files:**
- Create: `server/internal/service/scheduler.go` ✅
- Test: `server/tests/service/scheduler_test.go` ✅

**说明：**

`scheduler.go` — `Scheduler` 结构体，依赖 TicketRepository、MessageService。

- `Start(ctx context.Context)` — 启动调度器，通过 `context.WithCancel` 管理生命周期
- `stop()` — 取消 context，停止所有 ticker

**TicketAutoCloseJob：**
- 使用 `time.NewTicker(1 * time.Hour)`
- 查询 `status IN (1,2,3) AND created_at < NOW() - 7天`
- 批量更新为 `status=5`（已关闭）
- 写入 audit_log（action=auto_close）

**MessageNotifyJob：**
- 在 `TicketService.UpdateStatus` 中同步调用（非独立 goroutine）
- 状态变为"需补充信息"(3) 时，调用 `MessageService.NotifySupplement`

**测试覆盖：**
- 自动关闭：模拟 7 天前创建的申告，验证被关闭
- 消息通知：状态变为需补充信息时，验证消息被创建

---

### Task 31: main.go 集成调度器 ✅

**Files:**
- Modify: `server/cmd/main.go` ✅
- Modify: `server/internal/router/router.go` ✅
- Modify: `server/internal/router/portal.go` ✅

**说明：**

在 main.go 中：
1. 创建 Scheduler 实例
2. 调用 `scheduler.Start(ctx)` 启动
3. 监听 `os.Signal`（SIGINT/SIGTERM），收到信号后 `cancel()` 停止调度器
4. 优雅关闭 HTTP 服务（`srv.Shutdown(ctx)`）

**Verification:** 启动服务后日志显示调度器已启动，Ctrl+C 后优雅退出。

---

## M5：数据看板与日志审计（第 6 周）

本里程碑目标：看板统计、审计日志、模型/embedding 配置页面。

---

### Task 32: 数据看板 Service + Handler ✅

**Files:**
- Create: `server/internal/dto/request/dashboard.go` ✅
- Create: `server/internal/dto/response/dashboard.go` ✅
- Create: `server/internal/service/dashboard_service.go` ✅
- Create: `server/internal/handler/dashboard.go` ✅
- Create: `src/api/dashboard.ts` ✅
- Modify: `src/views/admin/Dashboard.vue` ✅（全功能实现：统计卡片/柱状趋势图/日期选择/空-加载-错误三态）
- Test: `server/tests/service/dashboard_service_test.go` ✅（6 个测试：统计/空数据/趋势/日期范围/无效粒度/空范围）

**说明：**

**dto/request/dashboard.go** — `TrendRequest`：StartDate、EndDate、Granularity（day/week）。

**dto/response/dashboard.go** — `StatsResponse`：TodayTickets、PendingTickets、ProcessingTickets、ResolvedTickets、TodayChats、AvgConfidence、KnowledgeCount。`TrendResponse`：DataPoints（[]DataPoint），DataPoint 含 Date、TicketCount、ChatCount。

**dashboard_service.go** — 方法：
- `GetStats() (*StatsResponse, error)` — SQL 聚合查询：今日申告数、各状态申告数、今日问答数、平均置信度、知识条目数
- `GetTrends(req TrendRequest) (*TrendResponse, error)` — 按日期/周聚合申告和问答趋势

**dashboard.go Handler** — 绑定看板 API。

**前端：**
- **api/dashboard.ts** — 封装看板 API。
- **Dashboard.vue** — 数据看板页面：
  - 顶部统计卡片（今日申告、待处理、处理中、已解决、今日问答、平均置信度、知识条目数）
  - 中部趋势图（折线图：申告趋势 + 问答趋势，支持日期范围选择）
  - 遵循 Linear Design 暗色主题

**测试覆盖：**
- 统计数据 SQL 正确性
- 趋势数据按天/周聚合

---

### Task 33: 审计日志 Repository + 查询 ✅

**Files:**
- Create: `server/internal/repository/audit_repo.go` ✅
- Create: `server/internal/handler/audit.go` ✅
- Create: `server/internal/dto/request/audit.go` ✅
- Create: `server/internal/dto/response/audit.go` ✅
- Test: `server/tests/repository/audit_repo_test.go` ✅（7 个测试：Create/全量/按操作人/按操作/组合/分页/空数据）

**说明：**

**audit_repo.go** — 数据访问：
- `Create(log *AuditLog) error`
- `List(operatorID int64, action string, page, pageSize int) ([]AuditLog, int64, error)` — 支持按操作人和操作类型筛选

**dto/request/audit.go** — `AuditLogListRequest`：OperatorID、Action、Page、PageSize。

**dto/response/audit.go** — `AuditLogListResponse`（分页，含操作人姓名）。

**audit.go Handler** — 绑定 `GET /api/v1/admin/audit-logs`，从 context 取当前用户角色（仅系统管理员可访问）。

**前端：**
- **src/views/admin/AuditLog.vue** — 审计日志列表页：表格展示（操作人、操作类型、目标、详情、IP 地址、时间），支持筛选和分页。

**说明：** 审计日志的写入分散在各 Service 中（用户管理、知识管理、申告管理等），本任务只负责查询和展示。

---

### Task 34: 系统配置 Service + Handler ✅

**Files:**
- Create: `server/internal/service/config_service.go` ✅
- Create: `server/internal/handler/config.go` ✅
- Test: `server/tests/service/config_service_test.go` ✅（13 个测试：6 Repo + 7 Service — 获取/不存在/JSON对象/创建/更新/字符串/nil拒绝）

**说明：**

**config_service.go** — 方法：
- `GetConfig(key string) (interface{}, error)`
- `UpdateConfig(key string, value interface{}, updatedBy int64) error`
- 系统配置项：`ai.default_top_k`、`ai.confidence_threshold`

**前端：**
- **src/views/admin/ModelConfig.vue** — 模型配置页：Top K 滑块（1-10）、置信度阈值滑块（0.1-1.0）。不暴露 AnythingLLM API Key 明文。
- **src/views/admin/EmbeddingConfig.vue** — Embedding 配置管理页：列表展示已有配置、新增/编辑弹窗（模型名称、类型选择 API/本地、API 地址/本地路径、向量维度、是否默认）。
- **src/views/admin/SystemConfig.vue** — 系统配置页：展示和编辑可配置项。

---

## M6：联调测试与文档完善（第 7 周）

本里程碑目标：集成测试、演示数据、API 文档、README。

---

### Task 35: Vue 看板和配置页面完善 ✅

**Files:**
- Modify: `src/views/admin/Dashboard.vue` ✅（全功能：7 统计卡片/CSS 柱状趋势图/日期选择/空-加载-错误三态）
- Modify: `src/views/admin/ModelConfig.vue` ✅（全功能：Top K + 置信度双滑块/实时数值/保存 Toast）
- Modify: `src/views/admin/EmbeddingConfig.vue` ✅（全功能：列表+新增/编辑弹窗+类型切换+删除确认对话框+Toast）
- Modify: `src/views/admin/SystemConfig.vue` ✅（全功能：表格展示+行内编辑+保存/取消+值类型自动识别+Toast）
- Create: `src/api/dashboard.ts` ✅（看板 API 封装：getStats/getTrends）

**说明：**

完成 M5 中创建的前端页面，处理边界情况：
- ✅ 看板数据为空时的占位展示「暂无趋势数据」
- ✅ 配置保存成功/失败的 Toast 提示（3 秒自动消失，绿/红）
- ✅ Embedding 配置删除确认对话框（不可撤销提示）
- ✅ 加载中状态（居中 loading 提示）
- ✅ API 错误处理（错误消息 + 重试按钮 / Toast 错误提示）
- 全部遵循 Linear Design 暗色主题

---

### Task 36: 集成测试

**Files:**
- Create: `server/tests/integration/auth_test.go`
- Create: `server/tests/integration/chat_test.go`
- Create: `server/tests/integration/ticket_test.go`
- Create: `server/tests/integration/knowledge_test.go`

**说明：**

集成测试使用测试数据库（独立的 `opsmind_test` 库），不 mock 外部服务。AnythingLLM 和 MinIO 使用 testcontainers 或预配置的测试实例。

**auth_test.go：**
- 登录→获取令牌→刷新令牌→修改密码→退出
- 首次登录强制修改密码

**chat_test.go：**
- 创建问答会话→返回答案和来源
- 低置信度→can_submit_ticket=true
- 提交反馈

**ticket_test.go：**
- 创建申告→开始处理→需补充信息→补充信息→解决→回访
- request_info 超过 3 次被拒绝
- 7 天自动关闭

**knowledge_test.go：**
- 创建知识库→创建知识条目→提交审核→审核通过→发布→同步状态
- 停用知识→同步状态变 disabled
- 重试同步

---

### Task 37: Docker Compose 和部署配置

**Files:**
- Create: `docker-compose.yml`
- Create: `.env`
- Create: `server/Dockerfile`
- Create: `web/Dockerfile`
- Create: `Makefile`

**说明：**

**docker-compose.yml** — 与 ANYTHINGLLM_AI_INTEGRATION.md §3.1 完全对齐：
- 7 个服务：opsmind-server、opsmind-web、opsmind-setup（自动初始化）、anythingllm、vllm（ai-local profile）、postgres、minio
- AnythingLLM 默认不暴露端口
- vLLM 通过 `--profile ai-local` 启用
- 所有服务通过 `opsmind` bridge 网络互通
- 环境变量引用 `.env`

**.env** — 与集成方案 §3.2 对齐，包含全部环境变量模板。

**server/Dockerfile** — 多阶段构建：Go 编译 → Alpine 运行时。暴露 8080 端口。

**web/Dockerfile** — 多阶段构建：Node 编译 → nginx 运行时。配置 nginx 反向代理 `/api → opsmind-server:8080`。

**Makefile** — 常用命令：
- `make dev` — 本地开发启动
- `make build` — 构建全部镜像
- `make up` — docker compose up
- `make down` — docker compose down
- `make test` — 运行全部测试
- `make migrate` — 运行数据库迁移
- `make seed` — 加载演示数据

**Verification:** `docker compose up -d --build` 一键启动全部服务，访问 `http://localhost:5173` 可用。

---

### Task 38: 演示数据和 README

**Files:**
- Create: `server/migrations/seed.sql`
- Create: `README.md`

**说明：**

**seed.sql** — 演示数据，覆盖：
- 4 个预设角色（系统管理员、运维人员、知识库管理员、报障人）
- 菜单数据（后台管理全部菜单项）
- 1 个系统管理员账号（admin / Admin@123）
- 2 个运维人员账号
- 1 个知识库管理员账号
- 2 个报障人账号
- 1 个知识库 + 5 条知识条目（运维 FAQ）
- 3 条申告（不同状态）
- 1 个 embedding 配置（默认）

**README.md** — 项目说明文档：
- 项目简介和功能概述
- 技术栈说明
- 快速启动指南（Docker Compose 一键启动）
- 初始化 AnythingLLM API Key 步骤（引用集成方案 §3.3）
- 开发环境搭建（本地 Go + Vue + Docker 依赖服务）
- 项目结构说明
- API 概览
- 默认账号信息
- 常见问题

---

## PRD 功能需求覆盖检查

| PRD 功能需求 | 对应任务 |
| --- | --- |
| FR-1: 问答 API 同步返回答案 | T26 |
| FR-2: 置信度判断与兜底 | T26 |
| FR-3: 申告 CRUD 和状态机 | T24 |
| FR-4: 知识库 CRUD 和审核流程 | T18, T19 |
| FR-5: AnythingLLM RAG 集成 | T20 |
| FR-6: 用户账号管理 | T14 |
| FR-7: 角色权限管理 | T15 |
| FR-8: 数据看板统计 | T32 |
| FR-9: 操作日志审计 | T33 |
| FR-10: 模型和 embedding 配置 | T19, T34 |
| FR-11: 站内消息 | T29 |
| FR-12: 知识同步状态管理 | T18 |
| FR-13: 降级和兜底策略 | T26 |
| FR-14: 7 天自动关闭 | T30 |

## PRD 用户故事覆盖检查

| PRD 用户故事 | 对应任务 |
| --- | --- |
| US-001: 智能问答 | T26, T28 (Chat.vue) |
| US-002: 申告记录 | T24, T28 (TicketSubmit.vue) |
| US-003: 人工处理与回访 | T24 (状态机 + 回访 detail) |
| US-004: 进度查询 | T28 (TicketQuery.vue, TicketDetail.vue) |
| US-005: 知识库维护 | T18, T22 |
| US-006: 知识审核 | T18 (Review + Publish) |
| US-007: 账号管理 | T14, T16 (UserList.vue) |
| US-008: 角色权限 | T15, T16 (RoleManage.vue) |
| US-009: 系统配置 | T34, T35 |
| US-010: 数据看板 | T32, T35 |
| US-011: 操作日志 | T33 |
| US-012: 站内消息 | T29, T28 (Messages.vue) |

## TECH 一致性检查

| TECH 要求 | 对应任务 | 验证点 |
| --- | --- | --- |
| §4.2 表结构全部字段 | T04 | GORM 模型与表定义完全对齐 |
| §5.2 全部 API 端点 | T07, T11, T14, T18, T24, T26, T29, T32, T33, T34 | 路由注册覆盖全部端点 |
| §7.1 RagClient 接口 | T20 | 4 个方法 + 字段映射 |
| §7.3 StorageClient 接口 | T27 | 3 个方法 |
| §8.3 密码策略 | T05, T11 | 正则校验 + bcrypt |
| §9.1 Docker Compose | T37 | 7 个服务编排 |
| §10.1 降级规则 | T26 | 6 种场景 |
| §10.2 知识同步流程 | T18 | 发布/停用/重试 |
| §10.3 后台调度器 | T30 | 自动关闭 + 消息通知 |
