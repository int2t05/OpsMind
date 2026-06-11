# 运维数字员工系统 — 技术架构文档

| 项目 | 内容 |
| --- | --- |
| 文档版本 | v1.3 |
| 日期 | 2026-06-10 |
| 变更说明 | 同步 M5 全部实现：补全 API 端点、修正前端路由、更新文件结构 |
| 关联文档 | [PRD](PRD.md)、[Design System](prompts/DESIGN-linear.app.md)、[AnythingLLM 集成方案](ANYTHINGLLM_AI_INTEGRATION.md) |

---

## 1. 架构总览

### 1.1 系统架构图

```
┌─────────────────────────────────────────────────────────────────────┐
│                          客户端（浏览器）                              │
│  ┌──────────────────────┐    ┌──────────────────────────────────┐  │
│  │   门户端 (Portal)     │    │   后台管理端 (Admin)               │  │
│  │   - 智能问答           │    │   - 申告处理                      │  │
│  │   - 申告提交           │    │   - 知识库管理                    │  │
│  │   - 进度查询           │    │   - 账号管理                      │  │
│  │                       │    │   - 数据看板                      │  │
│  └──────────┬────────────┘    │   - 系统配置                      │  │
│             │                 └──────────────┬───────────────────┘  │
└─────────────┼────────────────────────────────┼──────────────────────┘
              │ HTTP/REST                       │ HTTP/REST
              ▼                                 ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        Go 后端服务 (Gin)                             │
│                                                                     │
│  ┌───────────┐ ┌───────────┐ ┌───────────┐ ┌───────────┐          │
│  │ 认证模块   │ │ 问答模块   │ │ 申告模块   │ │ 知识库模块 │          │
│  │ Auth      │ │ Chat      │ │ Ticket    │ │ Knowledge │          │
│  └─────┬─────┘ └─────┬─────┘ └─────┬─────┘ └─────┬─────┘          │
│        │             │             │             │                  │
│  ┌─────┴─────┐ ┌─────┴─────┐ ┌─────┴─────┐ ┌─────┴─────┐          │
│  │ 用户权限   │ │ 日志模块   │ │ 看板模块   │ │ 配置模块   │          │
│  │ RBAC      │ │ Audit     │ │ Dashboard │ │ Config    │          │
│  └───────────┘ └───────────┘ └───────────┘ └───────────┘          │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                    适配层 (Adapters)                          │   │
│  │  ┌──────────────┐ ┌──────────────┐                          │   │
│  │  │ RagClient    │ │ StorageClient│                          │   │
│  │  │ (AnythingLLM)│ │ (MinIO/S3)   │                          │   │
│  │  └──────┬───────┘ └──────┬───────┘                          │   │
│  └─────────┼────────────────┼──────────────────────────────────┘   │
└────────────┼────────────────┼──────────────────────────────────────┘
             │                │
             ▼                ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    Docker Compose 内部网络 (opsmind)                 │
│                                                                     │
│  ┌──────────────────┐    ┌──────────────────┐                      │
│  │  AnythingLLM     │───→│  vLLM            │                      │
│  │  (RAG 服务)       │    │  (模型推理)       │                      │
│  │  - 知识导入       │    │  - 文本生成       │                      │
│  │  - 切分/检索      │    │  - OpenAI API    │                      │
│  │  - RAG 编排       │    │                  │                      │
│  │  内部向量库:       │    └──────────────────┘                      │
│  │  LanceDB         │                                               │
│  └──────────────────┘                                               │
│                                                                     │
│  ┌──────────────────┐    ┌──────────────────┐                      │
│  │  PostgreSQL 18   │    │  MinIO           │                      │
│  │  + pgvector      │    │  (对象存储)       │                      │
│  │  - 业务数据       │    │  - 附件存储       │                      │
│  │  - 系统侧向量追溯 │    │  - 文档原件       │                      │
│  └──────────────────┘    └──────────────────┘                      │
└─────────────────────────────────────────────────────────────────────┘
```

**向量存储边界：**
- **AnythingLLM LanceDB：** AnythingLLM 内部向量库，负责 RAG 检索。知识发布时由 AnythingLLM 自动完成切分、embedding 和向量写入。
- **PostgreSQL pgvector：** OpsMind 系统侧向量存储，用于追溯和后续原生检索扩展。知识发布时由 OpsMind 后端按知识库配置的 embedding 模型生成向量并写入。
- 两者独立，互不依赖。RAG 检索走 AnythingLLM，系统侧查询走 pgvector。

### 1.2 架构风格

**单体分层架构（Modular Monolith）**

MVP 阶段采用单体分层架构，按业务模块划分包结构。选择理由：

- 课程设计周期短（7 周），微服务引入的运维复杂度不划算。
- 单人开发，无需服务间协作的解耦。
- 后续如需拆分，模块边界已清晰，可按模块独立部署。

### 1.3 请求处理流程

```
客户端请求
  │
  ▼
Gin Router
  │
  ├─→ CORS Middleware
  ├─→ JWT Auth Middleware ──→ 无效令牌 → 401
  ├─→ RBAC Middleware ──→ 无权限 → 403
  ├─→ Request Logger Middleware
  │
  ▼
Handler Layer（参数校验、响应格式化）
  │
  ▼
Service Layer（业务逻辑、事务管理）
  │
  ▼
Repository Layer（数据访问、ORM 查询）
  │
  ▼
Database / External Adapter
```

---

## 2. 技术选型

### 2.1 技术栈总表

| 层级 | 技术 | 版本 | 用途 |
| --- | --- | --- | --- |
| 后端框架 | Go + Gin | Go 1.22+ / Gin 1.9+ | REST API 服务 |
| ORM | GORM | v1.25+ | 数据库访问 |
| 前端框架 | Vue 3 | 3.4+ | 管理端 + 门户端 |
| UI 组件库 | Radix Vue | 1.9+ | 无障碍基础组件 |
| 状态管理 | Pinia | 2.1+ | 前端状态管理 |
| 路由 | Vue Router | 4.3+ | 前端路由 |
| HTTP 客户端 | Axios | 1.7+ | API 调用 |
| 数据库 | PostgreSQL | 18 | 业务数据 + 系统侧向量追溯 |
| 向量扩展 | pgvector | 0.7+ | OpsMind 系统侧知识向量存储 |
| RAG 服务 | AnythingLLM | latest | Docker 内部组件，负责知识导入、切分、向量检索、RAG 编排（内部向量库: LanceDB） |
| AI 推理 | vLLM | 0.4+ | Docker 内部组件，通过 AnythingLLM `generic-openai` 提供商接入 |
| 对象存储 | MinIO | latest | 附件和文档存储 |
| 认证 | JWT (golang-jwt) | v5 | 令牌认证 |
| 密码哈希 | bcrypt | - | 密码安全存储 |
| 日志 | Go slog | stdlib | 结构化日志 |
| 配置管理 | Viper | 1.18+ | 多环境配置 |

### 2.2 选型决策记录

#### ADR-001: 单体分层架构 vs 微服务

**状态：** 已接受

**背景：** 系统包含问答、申告、知识库、账号管理等模块，需要决定部署架构。

**决策：** 采用单体分层架构（Modular Monolith）。

**备选方案：**
- **微服务：** 各模块独立部署，通过 HTTP/gRPC 通信。优势是独立扩展和部署；劣势是运维复杂度高、单人开发效率低。
- **Serverless：** 按函数部署。优势是零运维；劣势是冷启动延迟、本地大模型集成困难。

**后果：**
- 正面：开发效率高、调试简单、部署简单。
- 负面：模块间耦合度较高，后续拆分需要重构。
- 后续扩展：模块边界按 Go package 划分，如需拆分可独立为服务。

#### ADR-002: PostgreSQL + pgvector vs 专用向量数据库

**状态：** 已接受

**背景：** 知识库需要存储和检索向量，需要选择向量存储方案。

**决策：** 使用 PostgreSQL 18 + pgvector 扩展。

**备选方案：**
- **Milvus/Qdrant：** 专用向量数据库，检索性能更高。优势是专业；劣势是额外部署一个服务、增加运维成本。
- **Elasticsearch + knn：** 全文检索 + 向量检索。优势是功能全面；劣势是资源消耗大（4C8GB 环境吃不消）。

**后果：**
- 正面：单一数据库、无额外服务、事务一致性。
- 负面：向量检索性能不如专用数据库（MVP 数据量可接受）。
- 约束：AnythingLLM 负责主要 RAG 检索，pgvector 仅用于系统侧向量存储和追溯。

#### ADR-003: AnythingLLM 作为内部 Docker 组件集成

**状态：** 已接受

**背景：** 智能问答需要 RAG 能力（知识导入、切分、检索增强、问答编排），需要决定是自建还是使用现有工具。

**决策：** 使用 AnythingLLM 作为 Docker 内部组件集成到 OpsMind，通过 `docker-compose.yml` 统一拉起。后端通过 `RagClient` 适配层接入，AnythingLLM 使用 LanceDB 作为内部向量库，vLLM 通过 AnythingLLM 的 `generic-openai` 提供商接入。

**备选方案：**
- **自建 RAG：** 使用 LangChain/LlamaIndex + pgvector 自建。优势是完全可控；劣势是开发量大、课程设计周期不允许。
- **Dify：** 另一个 RAG 平台。优势是 UI 友好；劣势是 API 灵活度不如 AnythingLLM。
- **AnythingLLM 作为外部服务：** 用户分别部署和运行。优势是解耦；劣势是配置复杂、用户体验差。

**后果：**
- 正面：一键启动、用户只访问 OpsMind、完整 RAG 流程开箱即用。
- 负面：Docker 镜像较大、AnythingLLM 内部行为不完全可控。
- 约束：后端通过 `RagClient` 适配层隔离，后续可替换为自建方案。AnythingLLM 管理页面不作为日常入口，只在初始化和排障时临时访问。

#### ADR-004: vLLM 通过 AnythingLLM generic-openai 接入

**状态：** 已接受

**背景：** 需要调用大模型生成答案，需要决定调用方式。

**决策：** vLLM 通过 AnythingLLM 的 `generic-openai` 提供商接入，OpsMind 后端不直接调用 vLLM。AnythingLLM 负责模型调用编排，OpsMind 只与 AnythingLLM 交互。

**备选方案：**
- **OpsMind 直接调用 vLLM：** 优势是更灵活；劣势是需要 OpsMind 自己处理 RAG 编排和模型调用的协调。
- **直接调用模型 API（如 ChatGLM 原生 API）：** 优势是无中间层；劣势是换模型需要改代码。
- **Ollama：** 本地模型管理工具。优势是简单；劣势是并发性能不如 vLLM。

**后果：**
- 正面：OpsMind 后端只需对接 AnythingLLM 一个服务、模型替换在 AnythingLLM 配置中完成。
- 负面：vLLM 部署需要 GPU 或足够内存，4C8GB 环境可能需要连接远程模型节点。
- 约束：AnythingLLM 环境变量中配置 `GENERIC_OPEN_AI_BASE_PATH` 指向 vLLM 地址。

#### ADR-005: MinIO 对象存储 vs 本地文件系统

**状态：** 已接受

**背景：** 申告附件和知识文档原件需要文件存储。

**决策：** 使用 MinIO（S3-compatible）作为对象存储。

**备选方案：**
- **本地文件系统：** 优势是零依赖；劣势是不支持分布式、备份困难。
- **阿里云 OSS / 腾讯 COS：** 优势是免运维；劣势是企业环境可能无法访问外网。

**后果：**
- 正面：S3 标准协议、本地部署、后续可迁移到云存储。
- 负面：需要额外部署 MinIO 服务。
- 约束：通过 S3-compatible 适配层接入，后续可切换存储后端。

#### ADR-006: JWT 认证 vs Session 认证

**状态：** 已接受

**背景：** 前后端分离架构需要选择认证方式。

**决策：** 使用 JWT（JSON Web Token）进行无状态认证。

**备选方案：**
- **Session + Cookie：** 优势是服务端可控；劣势是前后端分离场景下需要处理跨域。
- **OAuth2：** 优势是标准协议；劣势是 MVP 阶段不需要第三方登录。

**后果：**
- 正面：无状态、前后端分离友好、易于扩展。
- 负面：令牌吊销需要额外机制（MVP 阶段接受短期令牌 + 过期策略）。
- 实现：access_token 有效期 2 小时，refresh_token 有效期 7 天。

---

## 3. 项目结构

```
OpsMind/
├── docs/                           # 文档
│   ├── PRD.md                      # 产品需求文档
│   ├── TECH.md                     # 技术架构文档（本文档）
│   ├── PLAN.md                     # 实施计划（38 任务/6 里程碑）
│   ├── ANYTHINGLLM_AI_INTEGRATION.md # AnythingLLM 集成方案
│   ├── prompts/                    # 设计约束和提示词
│   │   └── DESIGN-linear.app.md    # Linear Design 系统约束
│   └── diagrams/                   # 架构和流程图表
│
├── server/                         # Go 后端
│   ├── cmd/
│   │   └── main.go                 # 入口
│   ├── internal/
│   │   ├── config/                 # 配置管理（Viper）
│   │   │   ├── config.go
│   │   │   └── config.yaml
│   │   ├── middleware/             # 中间件
│   │   │   ├── auth.go             # JWT 认证
│   │   │   ├── rbac.go             # 权限校验
│   │   │   ├── logger.go           # 请求日志
│   │   │   └── cors.go
│   │   ├── router/                 # 路由注册
│   │   │   ├── router.go
│   │   │   ├── portal.go           # 门户端路由
│   │   │   └── admin.go            # 后台管理路由
│   │   ├── handler/                # Handler 层（参数校验、响应格式化）
│   │   │   ├── auth.go
│   │   │   ├── chat.go
│   │   │   ├── ticket.go
│   │   │   ├── knowledge.go
│   │   │   ├── user.go
│   │   │   ├── role.go
│   │   │   ├── dashboard.go
│   │   │   ├── config.go
│   │   │   ├── message.go
│   │   │   └── audit.go
│   │   ├── service/                # Service 层（业务逻辑）
│   │   │   ├── auth_service.go
│   │   │   ├── user_service.go
│   │   │   ├── role_service.go
│   │   │   ├── chat_service.go
│   │   │   ├── ticket_service.go
│   │   │   ├── knowledge_service.go
│   │   │   ├── dashboard_service.go
│   │   │   ├── config_service.go
│   │   │   ├── message_service.go
│   │   │   └── scheduler.go
│   │   ├── repository/             # Repository 层（数据访问）
│   │   │   ├── user_repo.go
│   │   │   ├── role_repo.go
│   │   │   ├── ticket_repo.go
│   │   │   ├── knowledge_repo.go
│   │   │   ├── chat_repo.go
│   │   │   ├── audit_repo.go
│   │   │   ├── config_repo.go
│   │   │   └── message_repo.go
│   │   ├── model/                  # 数据模型（GORM）
│   │   │   ├── user.go             # User/Role/Menu/UserRole/RoleMenu 数据模型
│   │   │   ├── ticket.go           # Ticket/TicketRecord 数据模型
│   │   │   ├── knowledge.go        # KnowledgeBase/KnowledgeArticle/KnowledgeChunk 数据模型
│   │   │   ├── embedding.go        # EmbeddingConfig 数据模型
│   │   │   ├── chat.go             # ChatSession/ChatMessage 数据模型
│   │   │   ├── audit.go            # AuditLog 数据模型
│   │   │   ├── system.go           # SystemConfig 数据模型
│   │   │   ├── message.go          # Message 数据模型
│   │   │   ├── enums.go            # 枚举常量定义
│   │   │   └── common.go           # 公共模型方法（分页等）
│   │   ├── adapter/                # 外部服务适配层
│   │   │   ├── rag_client.go       # AnythingLLM 适配器
│   │   │   └── storage_client.go   # MinIO 适配器
│   │   └── dto/                    # 数据传输对象
│   │       ├── request/
│   │       └── response/
│   ├── pkg/                        # 公共工具包
│   │   ├── response/               # 统一响应格式
│   │   ├── errcode/                # 错误码定义
│   │   ├── jwt/                    # JWT 工具
│   │   └── hash/                   # 密码哈希工具
│   ├── migrations/                 # 数据库迁移
│   ├── tests/                      # 全部测试代码（外部测试包）
│   │   ├── config/                 # 配置加载测试
│   │   ├── database/               # 数据库连接和迁移测试
│   │   ├── model/                  # 数据模型字段测试
│   │   ├── service/                # Service 层单元测试
│   │   ├── handler/                # Handler 层集成测试
│   │   ├── middleware/             # 中间件测试
│   │   ├── adapter/                # 适配层测试
│   │   └── pkg/                    # 公共工具包测试
│   ├── go.mod
│   └── go.sum
│
├── web/                            # Vue 前端
│   ├── src/
│   │   ├── api/                    # API 请求封装
│   │   │   ├── auth.ts
│   │   │   ├── chat.ts
│   │   │   ├── ticket.ts
│   │   │   ├── knowledge.ts
│   │   │   ├── user.ts
│   │   │   ├── dashboard.ts
│   │   │   └── message.ts
│   │   ├── views/                  # 页面
│   │   │   ├── portal/             # 门户端页面
│   │   │   │   ├── Chat.vue        # 智能问答
│   │   │   │   ├── TicketSubmit.vue # 申告提交
│   │   │   │   └── TicketQuery.vue  # 进度查询
│   │   │   ├── admin/              # 后台管理页面
│   │   │   │   ├── Dashboard.vue   # 数据看板
│   │   │   │   ├── TicketList.vue  # 申告列表
│   │   │   │   ├── TicketDetail.vue # 申告处理
│   │   │   │   ├── KnowledgeList.vue # 知识列表
│   │   │   │   ├── KnowledgeEdit.vue # 知识编辑
│   │   │   │   ├── UserList.vue    # 账号管理
│   │   │   │   ├── RoleManage.vue  # 角色管理
│   │   │   │   ├── AuditLog.vue    # 审计日志
│   │   │   │   ├── ModelConfig.vue # 模型配置
│   │   │   │   ├── EmbeddingConfig.vue # Embedding 配置
│   │   │   │   └── SystemConfig.vue # 系统配置
│   │   │   └── auth/
│   │   │       ├── Login.vue       # 登录
│   │   │       └── ChangePassword.vue # 修改密码
│   │   ├── components/             # 通用组件
│   │   ├── stores/                 # Pinia 状态管理
│   │   │   ├── auth.ts
│   │   │   ├── chat.ts
│   │   │   └── app.ts
│   │   ├── router/                 # Vue Router
│   │   │   └── index.ts
│   │   ├── utils/                  # 工具函数
│   │   ├── styles/                 # 全局样式（Linear Design）
│   │   └── App.vue
│   │
│   ├── package.json
│   └── vite.config.ts
│
├── docker-compose.yml              # 本地开发环境编排
├── Makefile                        # 构建和开发命令
└── README.md                       # 项目说明
```

---

## 4. 数据库设计

### 4.1 ER 图

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   users      │     │   roles      │     │ user_roles   │
│──────────────│     │──────────────│     │──────────────│
│ id (PK)      │◄──┐ │ id (PK)      │◄──┐ │ user_id (FK) │
│ username     │   │ │ name         │   │ │ role_id (FK) │
│ password_hash│   │ │ description  │   │ └──────────────┘
│ real_name    │   │ │ permissions  │   │
│ phone        │   │ │ created_at   │   │ ┌──────────────┐
│ email        │   │ └──────────────┘   │ │ role_menus   │
│ status       │   └────────────────────│─│──────────────│
│ first_login  │                        │ │ role_id (FK) │
│ created_at   │                        │ │ menu_id (FK) │
│ updated_at   │                        │ └──────────────┘
└──────┬───────┘                        │
       │                                │ ┌──────────────┐
       │ 1:N                            │ │   menus      │
       ▼                                │ │──────────────│
┌──────────────┐                        │ │ id (PK)      │
│ tickets      │                        │ │ name         │
│──────────────│                        │ │ path         │
│ id (PK)      │                        │ │ icon         │
│ ticket_no    │                        │ │ parent_id    │
│ user_id (FK) │                        │ │ sort_order   │
│ title        │                        │ │ type         │
│ description  │                        │ └──────────────┘
│ urgency      │                        │
│ impact_scope │                        │
│ affected_sys │                        │
│ contact_phone│                        │
│ contact_email│                        │
│ status       │                        │
│ chat_context │ (JSONB)                │
│ created_at   │                        │
│ updated_at   │                        │
└──────┬───────┘                        │
       │ 1:N                            │
       ▼                                │
┌──────────────┐                        │
│ticket_records│                        │
│──────────────│                        │
│ id (PK)      │                        │
│ ticket_id(FK)│                        │
│ operator_id  │────────────────────────┘
│ action       │ (start/request_info/supplement/resolve/close)
│ content      │ (处理过程描述)
│ detail       │ (JSONB，回访结果等结构化数据，见下方说明)
│ created_at   │
└──────────────┘

ticket_records.detail 字段结构（按 action 类型区分）：

action=resolve 时（回访结果）：
{
  "callback_method": "phone|email|im|offline",
  "callback_content": "回访沟通的具体内容",
  "is_resolved": true,
  "remark": "备注（选填）"
}

action=request_info 时：{"reason": "需要补充的原因"}
action=supplement 时：{"supplement_content": "补充的内容"}
其他 action：detail 可为 null

┌──────────────────┐     ┌──────────────────┐
│knowledge_bases   │     │knowledge_articles│
│──────────────────│     │──────────────────│
│ id (PK)          │◄──┐ │ id (PK)          │
│ name             │   │ │ kb_id (FK)       │
│ description      │   │ │ question         │
│ embedding_model  │   │ │ answer           │
│ vector_dimension │   │ │ category         │
│ created_by (FK)  │   │ │ tags             │
│ created_at       │   │ │ status           │
│ updated_at       │   │ │ created_by (FK)  │
└──────────────────┘   │ │ reviewed_by (FK) │
                       │ │ published_by(FK) │
                       │ │ review_comment   │
                       │ │ created_at       │
                       │ │ updated_at       │
                       │ └────────┬─────────┘
                       │          │ 1:N
                       │          ▼
                       │ ┌──────────────────┐
                       │ │knowledge_chunks  │
                       │ │──────────────────│
                       │ │ id (PK)          │
                       │ │ article_id (FK)  │
                       │ │ content          │
                       │ │ embedding (vec)  │
                       │ │ embedding_model  │
                       │ │ vector_dimension │
                       │ │ synced_at        │
                       │ │ sync_status      │
                       │ │ sync_error       │
                       │ │ created_at       │
                       │ └──────────────────┘
                       │
                       │ ┌──────────────────┐
                       │ │embedding_configs │
                       │ │──────────────────│
                       │ │ id (PK)          │
                       │ │ name             │
                       │ │ model_type       │ (api / local)
                       │ │ api_endpoint     │
                       │ │ api_key          │ (加密存储)
                       │ │ vector_dimension │
                       │ │ is_default       │
                       │ │ created_at       │
                       │ └──────────────────┘
                       │
                       └─────────────────────

┌──────────────────┐     ┌──────────────────┐
│ chat_sessions    │     │ chat_messages    │
│──────────────────│     │──────────────────│
│ id (PK)          │◄──┐ │ id (PK)          │
│ user_id (FK)     │   │ │ session_id (FK)  │
│ kb_id (FK)       │   │ │ role             │ (user / assistant)
│ question         │   │ │ content          │
│ answer           │   │ │ sources (JSONB)  │
│ sources (JSONB)  │   │ │ confidence       │
│ confidence       │   │ │ created_at       │
│ feedback         │   │ └──────────────────┘
│ duration_ms      │
│ created_at       │
└──────────────────┘

┌──────────────────┐
│ audit_logs       │
│──────────────────│
│ id (PK)          │
│ operator_id (FK) │
│ action           │
│ target_type      │
│ target_id        │
│ detail (JSONB)   │
│ ip_address       │
│ created_at       │
└──────────────────┘

┌──────────────────┐
│ system_configs   │
│──────────────────│
│ id (PK)          │
│ key              │ (唯一)
│ value            │ (JSONB)
│ description      │
│ updated_by (FK)  │
│ updated_at       │
└──────────────────┘

┌──────────────────┐
│ messages         │ (站内消息)
│──────────────────│
│ id (PK)          │
│ user_id (FK)     │
│ title            │
│ content          │
│ type             │ (ticket_supplement / system)
│ related_type     │ (ticket / knowledge / ...)
│ related_id       │
│ is_read          │
│ created_at       │
└──────────────────┘
```

### 4.2 关键表说明

#### users 表

| 字段 | 类型 | 约束 | 说明 |
| --- | --- | --- | --- |
| id | bigint | PK, auto_increment | 主键 |
| username | varchar(64) | UNIQUE, NOT NULL | 登录账号 |
| password_hash | varchar(255) | NOT NULL | bcrypt 哈希 |
| real_name | varchar(64) | NOT NULL | 真实姓名 |
| phone | varchar(11) | NOT NULL | 手机号 |
| email | varchar(128) | | 邮箱（选填） |
| status | smallint | NOT NULL, DEFAULT 1 | 1=正常, 2=冻结 |
| first_login | boolean | NOT NULL, DEFAULT true | 是否首次登录 |
| created_at | timestamptz | NOT NULL | 创建时间 |
| updated_at | timestamptz | NOT NULL | 更新时间 |

#### tickets 表

| 字段 | 类型 | 约束 | 说明 |
| --- | --- | --- | --- |
| id | bigint | PK, auto_increment | 主键 |
| ticket_no | varchar(32) | UNIQUE, NOT NULL | 申告编号 TK-YYYYMMDD-XXXX |
| user_id | bigint | FK → users.id | 提交人 |
| title | varchar(255) | NOT NULL | 问题标题 |
| description | text | NOT NULL | 问题描述 |
| urgency | smallint | NOT NULL | 1=低, 2=中, 3=高 |
| impact_scope | smallint | | 1=个人, 2=部门, 3=全公司 |
| affected_systems | jsonb | | 受影响系统列表 |
| contact_phone | varchar(11) | NOT NULL | 联系手机号 |
| contact_email | varchar(128) | | 联系邮箱 |
| status | smallint | NOT NULL, DEFAULT 1 | 1=待处理, 2=处理中, 3=需补充信息, 4=已解决, 5=已关闭 |
| supplement_count | smallint | NOT NULL, DEFAULT 0 | 补充信息次数（≤3） |
| chat_context | jsonb | | 原始问答上下文 |
| source | smallint | NOT NULL, DEFAULT 1 | 1=门户提交, 2=问答转申告 |
| created_at | timestamptz | NOT NULL | 创建时间 |
| updated_at | timestamptz | NOT NULL | 更新时间 |

#### ticket_records 表

| 字段 | 类型 | 约束 | 说明 |
| --- | --- | --- | --- |
| id | bigint | PK, auto_increment | 主键 |
| ticket_id | bigint | FK → tickets.id | 所属申告 |
| operator_id | bigint | FK → users.id | 操作人 |
| action | varchar(32) | NOT NULL | 操作类型：start / request_info / supplement / resolve / close |
| content | text | | 处理过程描述 |
| detail | jsonb | | 结构化数据（回访结果等，见 §4.2 detail 字段说明） |
| created_at | timestamptz | NOT NULL | 操作时间 |

#### knowledge_bases 表

| 字段 | 类型 | 约束 | 说明 |
| --- | --- | --- | --- |
| id | bigint | PK, auto_increment | 主键 |
| name | varchar(128) | NOT NULL | 知识库名称 |
| description | text | | 知识库描述 |
| rag_workspace_slug | varchar(128) | UNIQUE | AnythingLLM workspace slug（创建知识库时由后端调用 AnythingLLM `/workspace/new` 生成） |
| embedding_model | varchar(128) | NOT NULL | embedding 模型名称（用于 pgvector 侧向量生成） |
| vector_dimension | integer | NOT NULL | 向量维度（用于 pgvector 侧向量生成） |
| created_by | bigint | FK → users.id | 创建人 |
| created_at | timestamptz | NOT NULL | 创建时间 |
| updated_at | timestamptz | NOT NULL | 更新时间 |

#### knowledge_articles 表

| 字段 | 类型 | 约束 | 说明 |
| --- | --- | --- | --- |
| id | bigint | PK, auto_increment | 主键 |
| kb_id | bigint | FK → knowledge_bases.id | 所属知识库 |
| question | text | NOT NULL | 问题 |
| answer | text | NOT NULL | 答案 |
| category | varchar(64) | | 分类 |
| tags | jsonb | | 标签列表 |
| status | smallint | NOT NULL, DEFAULT 1 | 1=草稿, 2=待审核, 3=已发布, 4=已停用, 5=驳回 |
| created_by | bigint | FK → users.id | 创建人 |
| reviewed_by | bigint | FK → users.id | 审核人 |
| published_by | bigint | FK → users.id | 发布人 |
| review_comment | text | | 审核意见（驳回原因） |
| rag_document_location | varchar(512) | | AnythingLLM 返回的 document location（同步成功后保存，停用时用于移除 embedding） |
| created_at | timestamptz | NOT NULL | 创建时间 |
| updated_at | timestamptz | NOT NULL | 更新时间 |

#### knowledge_chunks 表

| 字段 | 类型 | 约束 | 说明 |
| --- | --- | --- | --- |
| id | bigint | PK, auto_increment | 主键 |
| article_id | bigint | FK → knowledge_articles.id | 所属知识条目 |
| content | text | NOT NULL | 切片内容 |
| embedding | vector(N) | | 向量（N 由知识库配置决定） |
| embedding_model | varchar(128) | NOT NULL | 使用的 embedding 模型 |
| vector_dimension | integer | NOT NULL | 向量维度 |
| sync_status | varchar(16) | NOT NULL, DEFAULT 'pending' | pending=待同步, synced=已同步, failed=同步失败, disabled=已停用 |
| sync_error | text | | 同步失败原因 |
| synced_at | timestamptz | | 同步时间 |
| created_at | timestamptz | NOT NULL | 创建时间 |

#### embedding_configs 表

| 字段 | 类型 | 约束 | 说明 |
| --- | --- | --- | --- |
| id | bigint | PK, auto_increment | 主键 |
| name | varchar(128) | NOT NULL | 模型名称（如 text-embedding-3-small） |
| model_type | smallint | NOT NULL | 1=API 接入, 2=本地部署 |
| api_endpoint | varchar(512) | | API 地址（model_type=1 时必填） |
| api_key | varchar(512) | | API 密钥（加密存储） |
| local_path | varchar(512) | | 本地模型路径（model_type=2 时必填） |
| vector_dimension | integer | NOT NULL | 向量维度（如 384, 768, 1536） |
| is_default | boolean | NOT NULL, DEFAULT false | 是否默认模型 |
| created_at | timestamptz | NOT NULL | 创建时间 |

### 4.3 索引策略

```sql
-- users
CREATE UNIQUE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_status ON users(status);

-- tickets
CREATE UNIQUE INDEX idx_tickets_ticket_no ON tickets(ticket_no);
CREATE INDEX idx_tickets_user_id ON tickets(user_id);
CREATE INDEX idx_tickets_status ON tickets(status);
CREATE INDEX idx_tickets_created_at ON tickets(created_at DESC);

-- knowledge_articles
CREATE INDEX idx_articles_kb_id ON knowledge_articles(kb_id);
CREATE INDEX idx_articles_status ON knowledge_articles(status);
CREATE INDEX idx_articles_created_by ON knowledge_articles(created_by);

-- knowledge_chunks
CREATE INDEX idx_chunks_article_id ON knowledge_chunks(article_id);
CREATE INDEX idx_chunks_sync_status ON knowledge_chunks(sync_status);
-- pgvector HNSW 索引（按知识库配置的维度创建）
-- CREATE INDEX idx_chunks_embedding ON knowledge_chunks
--   USING hnsw (embedding vector_cosine_ops) WITH (m = 16, ef_construction = 64);

-- chat_sessions
CREATE INDEX idx_chat_user_id ON chat_sessions(user_id);
CREATE INDEX idx_chat_created_at ON chat_sessions(created_at DESC);

-- audit_logs
CREATE INDEX idx_audit_operator ON audit_logs(operator_id);
CREATE INDEX idx_audit_action ON audit_logs(action);
CREATE INDEX idx_audit_created_at ON audit_logs(created_at DESC);

-- messages
CREATE INDEX idx_messages_user_id ON messages(user_id);
CREATE INDEX idx_messages_is_read ON messages(user_id, is_read);
```

---

## 5. API 设计

### 5.1 通用约定

#### 请求格式

- Content-Type: `application/json`
- 认证: `Authorization: Bearer <access_token>`
- 分页: `?page=1&page_size=20`
- 排序: `?sort_by=created_at&sort_order=desc`

#### 响应格式

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

#### 错误码

| 错误码 | HTTP 状态 | 说明 |
| --- | --- | --- |
| 0 | 200 | 成功 |
| 10001 | 401 | 未登录或令牌过期 |
| 10002 | 403 | 无权限 |
| 10003 | 400 | 参数校验失败 |
| 10004 | 404 | 资源不存在 |
| 10005 | 409 | 资源冲突（如账号名重复） |
| 20001 | 500 | AI 服务不可用 |
| 20002 | 500 | RAG 服务不可用 |
| 20003 | 500 | 存储服务不可用 |
| 99999 | 500 | 未知错误 |

### 5.2 API 端点清单

#### 认证模块 `/api/v1/auth`

| 方法 | 路径 | 说明 | 权限 |
| --- | --- | --- | --- |
| POST | `/login` | 用户登录 | 公开 |
| POST | `/refresh` | 刷新令牌 | 登录用户 |
| POST | `/change-password` | 修改密码 | 登录用户 |
| POST | `/logout` | 退出登录 | 登录用户 |

#### 门户端 `/api/v1/portal`

| 方法 | 路径 | 说明 | 权限 |
| --- | --- | --- | --- |
| POST | `/chat-sessions` | 创建问答会话 | 登录用户 |
| GET | `/chat-sessions/:id` | 获取问答详情 | 登录用户 |
| POST | `/chat-sessions/:id/feedback` | 提交问答反馈 | 登录用户 |
| POST | `/tickets` | 创建申告 | 登录用户 |
| GET | `/tickets` | 查询我的申告列表 | 登录用户 |
| GET | `/tickets/:id` | 获取申告详情 | 登录用户 |
| PATCH | `/tickets/:id/supplement` | 补充申告信息 | 登录用户 |
| GET | `/messages` | 获取站内消息 | 登录用户 |
| PATCH | `/messages/:id/read` | 标记消息已读 | 登录用户 |
| GET | `/messages/unread-count` | 获取未读消息数 | 登录用户 |

#### 后台管理 `/api/v1/admin`

**申告管理：**

| 方法 | 路径 | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | `/tickets` | 申告列表（支持筛选） | 运维人员+ |
| GET | `/tickets/:id` | 申告详情 | 运维人员+ |
| PATCH | `/tickets/:id/status` | 更新申告状态 | 运维人员+ |
| POST | `/tickets/:id/records` | 添加处理记录 | 运维人员+ |
| POST | `/tickets/:id/knowledge-candidate` | 生成知识候选 | 运维人员+ |

**知识库管理：**

| 方法 | 路径 | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | `/knowledge-bases` | 知识库列表 | 知识库管理员 |
| POST | `/knowledge-bases` | 创建知识库 | 知识库管理员 |
| PUT | `/knowledge-bases/:id` | 更新知识库 | 知识库管理员 |
| GET | `/knowledge-bases/:kb_id/articles` | 知识条目列表 | 知识库管理员 |
| POST | `/knowledge-bases/:kb_id/articles` | 创建知识条目 | 知识库管理员 |
| GET | `/articles/:id` | 获取知识条目详情 | 知识库管理员 |
| PUT | `/articles/:id` | 更新知识条目 | 知识库管理员 |
| POST | `/articles/:id/submit-review` | 提交审核 | 知识库管理员 |
| POST | `/articles/:id/review` | 审核知识 | 知识库管理员（非创建人） |
| POST | `/articles/:id/publish` | 发布知识 | 知识库管理员 |
| POST | `/articles/:id/disable` | 停用知识 | 知识库管理员 |
| POST | `/articles/:id/retry-sync` | 重试同步 | 知识库管理员 |
**审核业务规则（Service 层强制校验）：**
- 审核人必须具有"知识库管理员"角色（RBAC 中间件校验）。
- 审核人不能是知识条目的创建人（`KnowledgeService.Review` 方法内校验 `article.CreatedBy != currentUser.ID`，违反时返回错误码 10003）。
- 驳回时必须填写 `review_comment`（参数校验）。

**用户管理：**

| 方法 | 路径 | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | `/users` | 用户列表 | 系统管理员 |
| POST | `/users` | 创建用户 | 系统管理员 |
| PUT | `/users/:id` | 更新用户 | 系统管理员 |
| PATCH | `/users/:id/freeze` | 冻结用户 | 系统管理员 |
| PATCH | `/users/:id/unfreeze` | 恢复用户 | 系统管理员 |

**角色权限：**

| 方法 | 路径 | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | `/roles` | 角色列表 | 系统管理员 |
| POST | `/roles` | 创建角色 | 系统管理员 |
| GET | `/roles/:id` | 获取角色详情 | 系统管理员 |
| PUT | `/roles/:id` | 更新角色 | 系统管理员 |
| DELETE | `/roles/:id` | 删除角色 | 系统管理员 |
| GET | `/menus` | 菜单列表 | 系统管理员 |
| PUT | `/roles/:id/menus` | 更新角色菜单 | 系统管理员 |

**数据看板：**

| 方法 | 路径 | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | `/dashboard/stats` | 看板统计数据 | 运维人员+ |
| GET | `/dashboard/trends` | 趋势数据 | 运维人员+ |

**操作日志：**

| 方法 | 路径 | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | `/audit-logs` | 审计日志列表 | 系统管理员 |

**系统配置：**

| 方法 | 路径 | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | `/configs/:key` | 获取配置 | 系统管理员 |
| PUT | `/configs/:key` | 更新配置 | 系统管理员 |
| GET | `/embedding-configs` | embedding 模型列表 | 系统管理员 |
| POST | `/embedding-configs` | 添加 embedding 模型 | 系统管理员 |
| PUT | `/embedding-configs/:id` | 更新 embedding 模型 | 系统管理员 |
| DELETE | `/embedding-configs/:id` | 删除 embedding 模型 | 系统管理员 |

### 5.3 关键接口详细设计

#### POST `/api/v1/portal/chat-sessions` — 创建问答会话

**请求：**

```json
{
  "question": "账号冻结怎么处理",
  "kb_id": 1
}
```

**成功响应：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "session_id": 12345,
    "answer": "账号冻结处理步骤：\n1. 联系系统管理员...\n2. 提供身份验证...",
    "sources": [
      {
        "doc_name": "账号管理FAQ",
        "chunk_content": "账号冻结通常是由于...",
        "confidence": 0.85
      }
    ],
    "confidence": 0.85,
    "can_submit_ticket": false,
    "duration_ms": 3200
  }
}
```

**兜底响应（置信度低于阈值）：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "session_id": 12346,
    "answer": "抱歉，暂未找到与您问题匹配的解决方案。建议您提交申告，由运维人员人工处理。",
    "sources": [],
    "confidence": 0.32,
    "can_submit_ticket": true,
    "duration_ms": 1500
  }
}
```

**AI 服务不可用响应：**

```json
{
  "code": 20001,
  "message": "当前 AI 服务暂不可用，请提交申告由人工处理",
  "data": {
    "session_id": null,
    "answer": null,
    "sources": [],
    "confidence": 0,
    "can_submit_ticket": true,
    "duration_ms": 0
  }
}
```

#### PATCH `/api/v1/admin/tickets/:id/status` — 更新申告状态

**请求：**

```json
{
  "action": "resolve",
  "result": "已帮用户重置密码，账号恢复正常",
  "to_knowledge_candidate": true
}
```

**action 取值：**

| action | 说明 | 前置状态 | 目标状态 | 触发方 |
| --- | --- | --- | --- | --- |
| `start` | 开始处理 | 待处理 | 处理中 | 运维人员（后台） |
| `request_info` | 需补充信息 | 处理中 | 需补充信息 | 运维人员（后台） |
| `supplement` | 补充信息 | 需补充信息 | 处理中 | 报障人（门户端） |
| `resolve` | 处理完成 | 处理中 | 已解决 | 运维人员（后台） |
| `close` | 关闭 | 任意 | 已关闭 | 运维人员（后台）/ 系统自动（7 天超时） |

**业务规则：**
- `request_info` 操作时 `supplement_count` +1，超过 3 次时禁止再次发起，强制走 `resolve`。
- `supplement` 操作通过门户端 API `PATCH /api/v1/portal/tickets/:id/supplement` 触发。
- 7 天自动关闭由后台调度器执行（见 §10.3）。

---

## 6. 前端架构

### 6.1 路由结构

```typescript
// 公开路由
/login                          // 登录

// 门户端路由（需要登录，报障人角色）
/portal/chat                    // 智能问答
/portal/tickets/submit          // 申告提交
/portal/tickets                 // 我的申告
/portal/tickets/:id             // 申告详情
/portal/messages                // 站内消息

// 后台管理路由（需要登录 + 对应角色权限）
/admin/dashboard                // 数据看板
/admin/tickets                  // 申告列表
/admin/tickets/:id              // 申告处理
/admin/knowledge                // 知识库列表
/admin/knowledge/:id            // 知识编辑
/admin/users                    // 用户管理
/admin/roles                    // 角色管理
/admin/audit-logs               // 审计日志
/admin/model-config             // 模型配置
/admin/embedding-config         // embedding 配置
/admin/system-config            // 系统配置
```

### 6.2 状态管理（Pinia）

```typescript
// stores/auth.ts
interface AuthState {
  token: string | null
  user: UserInfo | null
  roles: string[]
  permissions: string[]
  menus: MenuItem[]
}

// stores/chat.ts
interface ChatState {
  currentSession: ChatSession | null
  messages: ChatMessage[]
  loading: boolean
}

// stores/app.ts
interface AppState {
  sidebarCollapsed: boolean
  unreadMessageCount: number
}
```

### 6.3 Design System 约束

前端 UI 严格遵循 `docs/prompts/DESIGN-linear.app.md` 中的 Linear Design 约束：

- 暗色主题：`#08090a` 营销背景、`#0f1011` 面板背景、`#191a1b` 浮层背景。
- 字体：Inter Variable，OpenType 特性 `"cv01", "ss03"` 全局启用。
- 强调字重：510（介于 regular 和 medium 之间）。
- 品牌色：`#5e6ad2`（背景）/ `#7170ff`（交互）/ `#828fff`（悬停）。
- 边框：半透明白色 `rgba(255,255,255,0.05)` 到 `rgba(255,255,255,0.08)`。
- 组件基础：Radix Vue。

---

## 7. 适配层设计

### 7.1 AnythingLLM 适配器 (`RagClient`)

AnythingLLM 作为 Docker 内部组件集成，OpsMind 后端通过 Docker 内部网络访问：`http://anythingllm:3001/api`。vLLM 由 AnythingLLM 内部调用（通过 `generic-openai` 提供商），OpsMind 后端不直接调用 vLLM。

```go
// adapter/rag_client.go
type RagClient interface {
    // Query 提交问题进行 RAG 检索和问答
    // 内部调用 POST /api/v1/workspace/{slug}/chat
    Query(ctx context.Context, req RAGQueryRequest) (*RAGQueryResponse, error)
    // SyncDocument 同步知识文档到 AnythingLLM
    // 结构化 FAQ 调用 POST /api/v1/document/raw-text
    // 文件类知识调用 POST /api/v1/document/upload
    SyncDocument(ctx context.Context, req RAGSyncRequest) (*RAGSyncResponse, error)
    // DisableDocument 停用知识（从 AnythingLLM 移除 embedding）
    // 调用 POST /api/v1/workspace/{slug}/update-embeddings，deletes 传入 rag_document_location
    DisableDocument(ctx context.Context, req RAGDisableRequest) error
    // CreateWorkspace 创建知识库对应的 workspace
    // 调用 POST /api/v1/workspace/new
    CreateWorkspace(ctx context.Context, req RAGCreateWorkspaceRequest) (*RAGCreateWorkspaceResponse, error)
}

type RAGQueryRequest struct {
    WorkspaceSlug string `json:"workspace_slug"`
    Question      string `json:"question"`
    SessionID     string `json:"session_id"` // AnythingLLM 会话标识，用于上下文关联
    TopK          int    `json:"top_k"`
}

type RAGQueryResponse struct {
    Answer     string      `json:"answer"`
    Sources    []RAGSource `json:"sources"`
    Confidence float64     `json:"confidence"` // max(sources[].score)
    ChatID     string      `json:"chat_id"`    // AnythingLLM 返回的 chatId
    DurationMS int64       `json:"duration_ms"`
    Error      string      `json:"error"` // AnythingLLM 返回的 error，非空表示异常
}

type RAGSource struct {
    DocName      string  `json:"doc_name"`      // 映射自 AnythingLLM sources[].title 或 sources[].chunkSource
    ChunkContent string  `json:"chunk_content"` // 映射自 AnythingLLM sources[].text
    Score        float64 `json:"score"`         // 映射自 AnythingLLM sources[].score
}

type RAGSyncRequest struct {
    WorkspaceSlug string            `json:"workspace_slug"`
    Content       string            `json:"content"`        // raw text 内容
    Filename      string            `json:"filename"`       // 文件名（上传时）
    IsFile        bool              `json:"is_file"`        // true=文件上传, false=raw text
    Metadata      map[string]string `json:"metadata"`       // title, docSource, chunkSource 等
}

type RAGSyncResponse struct {
    DocumentLocation string `json:"document_location"` // AnythingLLM 返回的 documents[0].location
}

type RAGDisableRequest struct {
    WorkspaceSlug    string   `json:"workspace_slug"`
    DocumentLocations []string `json:"document_locations"` // 要移除的文档 location 列表
}

type RAGCreateWorkspaceRequest struct {
    Name                string  `json:"name"`
    ChatMode            string  `json:"chat_mode"`            // "query"
    TopN                int     `json:"top_n"`
    SimilarityThreshold float64 `json:"similarity_threshold"`
    OpenAiPrompt        string  `json:"open_ai_prompt"`       // 系统提示词
}

type RAGCreateWorkspaceResponse struct {
    Slug string `json:"slug"` // workspace slug，保存到 knowledge_bases.rag_workspace_slug
}
```

**AnythingLLM 字段映射：**

| AnythingLLM 返回字段 | OpsMind 字段 | 说明 |
| --- | --- | --- |
| `textResponse` | `answer` | 模型生成的答案文本 |
| `sources[].title` 或 `sources[].chunkSource` | `sources[].doc_name` | 来源文档名称 |
| `sources[].text` | `sources[].chunk_content` | 命中的知识片段 |
| `max(sources[].score)` | `confidence` | 最高置信度 |
| `error != null` | 触发降级 | AnythingLLM 返回错误 |
| `sources` 为空或 `confidence < threshold` | `can_submit_ticket=true` | 引导转人工 |

**说明：** vLLM 由 AnythingLLM 通过 `generic-openai` 提供商内部调用，OpsMind 后端不直接调用 vLLM。系统提示词通过 AnythingLLM workspace 的 `openAiPrompt` 参数注入。

### 7.3 MinIO 适配器 (`StorageClient`)

```go
// adapter/storage_client.go
type StorageClient interface {
    // Upload 上传文件，返回对象 key
    Upload(ctx context.Context, bucket string, key string, reader io.Reader, size int64, contentType string) (string, error)
    // GetPresignedURL 获取预签名下载 URL
    GetPresignedURL(ctx context.Context, bucket string, key string, expiry time.Duration) (string, error)
    // Delete 删除对象
    Delete(ctx context.Context, bucket string, key string) error
}
```

**Bucket 规划：**

| Bucket | 用途 |
| --- | --- |
| `opsmind-attachments` | 申告附件 |
| `opsmind-documents` | 知识文档原件 |

---

## 8. 安全设计

### 8.1 认证流程

```
1. 用户 POST /api/v1/auth/login {username, password}
2. 后端校验：
   a. 用户名是否存在
   b. 密码 bcrypt 验证
   c. 账号状态是否正常（非冻结）
3. 生成 JWT：
   - access_token: 有效期 2 小时，payload 含 user_id, username, roles
   - refresh_token: 有效期 7 天
4. 返回 {access_token, refresh_token, user, roles, permissions}
5. 后续请求携带 Authorization: Bearer <access_token>
6. 中间件校验令牌 → 注入用户信息到 context
```

### 8.2 权限模型

**RBAC（基于角色的访问控制）**

```
用户 ──N:N──→ 角色 ──N:N──→ 菜单/接口权限
```

- 每个用户可拥有多个角色。
- 每个角色可关联多个菜单权限和接口权限。
- 后端中间件校验当前用户的角色是否包含目标接口的权限。
- 前端根据权限动态渲染菜单。

### 8.3 密码策略

| 规则 | 说明 | 校验位置 |
| --- | --- | --- |
| 长度 | 8-32 位 | Service 层 `AuthService.CreateUser` / `AuthService.ChangePassword` |
| 复杂度 | 必须包含大写字母、小写字母和数字 | 同上 |
| 存储 | bcrypt 哈希（cost=10），禁止明文 | Repository 层写入时哈希 |
| 首次登录 | `first_login=true` 时强制跳转修改密码页 | 中间件拦截 + 前端路由守卫 |
| 初始密码 | 由管理员创建账号时设置 | `UserService.CreateUser` |

**校验正则：** `^(?=.*[a-z])(?=.*[A-Z])(?=.*\d).{8,32}$`

### 8.4 敏感数据处理

| 数据 | 处理方式 |
| --- | --- |
| 密码 | bcrypt 哈希，不可逆 |
| API 密钥 | AES-256 加密存储，配置页面掩码显示 |
| JWT 令牌 | HTTPS 传输（生产环境） |
| 问答内容 | 按企业内部数据处理，不外传 |

---

## 9. 部署架构

### 9.1 本地开发环境

完整 `docker-compose.yml` 结构见 [AnythingLLM 集成方案 §3.1](ANYTHINGLLM_AI_INTEGRATION.md#31-推荐-compose-结构)。

核心服务：

| 服务 | 镜像 | 端口 | 说明 |
| --- | --- | --- | --- |
| opsmind-server | 自构建 | 8080 | Go 后端 |
| opsmind-web | 自构建 | 5173 | Vue 前端 |
| anythingllm | mintplexlabs/anythingllm:latest | 内部 3001（默认不暴露） | RAG 服务，Docker 内部组件 |
| vllm | vllm/vllm-openai:latest | 内部 8000 | 模型推理，通过 `ai-local` profile 启用 |
| postgres | pgvector/pgvector:pg18 | 5432 | 业务数据库 |
| minio | minio/minio:latest | 9000/9001 | 对象存储 |

**关键约束：**
- AnythingLLM 默认不暴露端口，仅 OpsMind 后端通过 Docker 内部网络 `http://anythingllm:3001/api` 访问。
- 初始化 API Key 时临时暴露 3001 端口，完成后移除。详见 [集成方案 §3.3](ANYTHINGLLM_AI_INTEGRATION.md#33-初始化-api-key)。
- vLLM 通过 `--profile ai-local` 启用，开发阶段可使用远程模型服务替代。
- 所有服务通过 `opsmind` bridge 网络互通。

**一键启动：**

```powershell
cd D:\Projects\Personal\OpsMind
docker compose up -d --build
```

### 9.2 环境配置

| 环境 | 数据库 | AI 服务 | 存储 | 说明 |
| --- | --- | --- | --- | --- |
| 开发 | 本地 Docker | Mock / 远程 | 本地 Docker MinIO | 快速启动 |
| 测试 | 测试服务器 | 测试 vLLM + AnythingLLM | 测试 MinIO | 集成测试 |
| 生产 | 生产 PostgreSQL | 生产 vLLM + AnythingLLM | 生产 MinIO | 企业部署 |

### 9.3 配置文件结构

Docker 环境变量（`.env`）完整配置见 [集成方案 §3.2](ANYTHINGLLM_AI_INTEGRATION.md#32-env-推荐配置)。

Go 后端 `config.yaml`（容器内使用环境变量覆盖）：

```yaml
# config.yaml
server:
  port: 8080
  mode: debug  # debug / release

database:
  host: ${DB_HOST:localhost}        # Docker 内: postgres
  port: 5432
  user: opsmind
  password: ${DB_PASSWORD}
  dbname: opsmind
  sslmode: disable

jwt:
  secret: ${JWT_SECRET}
  access_expire: 2h
  refresh_expire: 168h

minio:
  endpoint: ${MINIO_ENDPOINT:localhost:9000}  # Docker 内: minio:9000
  access_key: ${MINIO_ROOT_USER}
  secret_key: ${MINIO_ROOT_PASSWORD}
  use_ssl: false

anythingllm:
  base_url: ${ANYTHINGLLM_BASE_URL:http://anythingllm:3001/api}  # Docker 内部地址
  api_key: ${ANYTHINGLLM_API_KEY}
  default_workspace_slug: opsmind-it-ops
  timeout_seconds: 20

ai:
  default_top_k: ${AI_DEFAULT_TOP_K:5}
  confidence_threshold: ${AI_CONFIDENCE_THRESHOLD:0.6}
```

**配置约束：**
- AnythingLLM 地址固定使用 Docker 内部 URL `http://anythingllm:3001/api`，不使用 localhost。
- AnythingLLM API Key 只保存在后端配置中，不下发给前端。
- 系统提示词通过 AnythingLLM workspace 的 `openAiPrompt` 参数注入，不在 OpsMind 后端单独配置。
- vLLM 地址配置在 AnythingLLM 环境变量中（`GENERIC_OPEN_AI_BASE_PATH`），OpsMind 后端不直接访问 vLLM。

---

## 10. 错误处理与降级

### 10.1 AI 服务降级策略

AnythingLLM 负责完整 RAG 流程（内部通过 `generic-openai` 调用 vLLM），OpsMind 后端不直接调用 vLLM。

```
用户提问
  │
  ▼
OpsMind 后端调用 AnythingLLM: POST /api/v1/workspace/{slug}/chat
  │
  ├─→ 成功且 AnythingLLM 返回 error == null
  │     │
  │     ├─→ sources 非空 且 confidence ≥ 阈值 → 返回答案 + 来源文档名称
  │     └─→ sources 为空 或 confidence < 阈值 → 返回兜底提示，can_submit_ticket=true
  │
  ├─→ AnythingLLM 返回 error != null
  │     │
  │     ▼
  │   记录错误日志，返回转人工提示（can_submit_ticket=true）
  │
  └─→ AnythingLLM 容器不可达（网络/超时）
        │
        ▼
      返回 code=20001，提示"当前 AI 服务暂不可用，请提交申告由人工处理"
      can_submit_ticket=true
```

**降级规则明细：**

| 场景 | OpsMind 处理 |
| --- | --- |
| AnythingLLM 容器不可达 | 返回 `code=20001`，提示 AI 服务不可用 |
| vLLM 不可达导致 AnythingLLM 返回错误 | 记录错误，返回转人工提示 |
| AnythingLLM 返回 `error` 字段非空 | 记录错误，返回转人工提示 |
| `sources` 为空 | 返回兜底答案，`can_submit_ticket=true` |
| `confidence < threshold` | 返回兜底答案，`can_submit_ticket=true` |
| 成功且置信度达标 | 返回答案和来源，`can_submit_ticket=false` |

**兜底提示文本：**
- AI 服务不可用：`当前 AI 服务暂不可用，请提交申告由人工处理`
- 低置信度：`暂未找到足够匹配的知识，建议提交申告由运维人员人工处理`

### 10.2 知识同步与停用

**发布同步流程：**

```
知识审核通过 → KnowledgeService.Publish
  │
  ├─→ 调用 RagClient.SyncDocument
  │     结构化 FAQ: POST /api/v1/document/raw-text
  │     文件类知识: POST /api/v1/document/upload
  │     addToWorkspaces = knowledge_bases.rag_workspace_slug
  │     metadata.docSource = "knowledge_articles:{id}"
  │
  ├─→ 成功 → 保存 rag_document_location, sync_status = 'synced'
  │
  ├─→ 失败 → sync_status = 'failed', 记录 sync_error
  │
  └─→ 同时 → 生成向量写入 pgvector（系统侧追溯）
        ├─→ 成功 → pgvector 写入完成
        └─→ 失败 → 记录日志（不影响 AnythingLLM 同步状态）
```

**停用流程：**

```
知识停用 → KnowledgeService.Disable
  │
  ├─→ 调用 RagClient.DisableDocument
  │     POST /api/v1/workspace/{slug}/update-embeddings
  │     deletes = [rag_document_location]
  │
  ├─→ sync_status = 'disabled'
  │
  └─→ 写审计日志
```

**重试同步：** 管理员可在后台查看失败原因，点击"重试同步"重新执行发布流程。

**同步状态值：** `pending`（待同步）→ `synced`（已同步）/ `failed`（同步失败）/ `disabled`（已停用）

### 10.3 后台调度器

MVP 阶段使用 Go goroutine + `time.Ticker` 实现轻量级定时任务，无需外部依赖。

```
┌─────────────────────────────────────────┐
│           Scheduler (goroutine)         │
│                                         │
│  ┌─────────────────────────────────┐   │
│  │  TicketAutoCloseJob             │   │
│  │  每小时执行一次                    │   │
│  │  查询 status IN (1,2,3)          │   │
│  │  AND created_at < NOW() - 7天    │   │
│  │  → 批量更新为 status=5 (已关闭)   │   │
│  │  → 写入 audit_log               │   │
│  └─────────────────────────────────┘   │
│                                         │
│  ┌─────────────────────────────────┐   │
│  │  MessageNotifyJob               │   │
│  │  申告状态变更时即时触发            │   │
│  │  status 变为 需补充信息(3) 时      │   │
│  │  → 写入 messages 表              │   │
│  │  → type = ticket_supplement      │   │
│  └─────────────────────────────────┘   │
└─────────────────────────────────────────┘
```

**实现说明：**
- 调度器在 `main.go` 中随服务启动，通过 `context.WithCancel` 管理生命周期。
- `TicketAutoCloseJob` 使用 `time.NewTicker(1 * time.Hour)`，每次执行 SQL 批量更新。
- `MessageNotifyJob` 在 `TicketService.UpdateStatus` 中同步调用，状态变为"需补充信息"时写入消息。
- 后续如需更复杂的调度能力，可替换为 `robfig/cron` 库或独立 Worker 进程。

---

## 11. 开发里程碑与技术交付物

| 阶段 | 周期 | 技术交付物 |
| --- | --- | --- |
| M1：数据库与后端基础能力 | 1 周 | 数据库 DDL、Go 项目骨架、GORM 模型、分层架构、统一响应、Docker Compose |
| M2：账号权限与后台框架 | 1 周 | JWT 认证、RBAC 中间件、用户 CRUD、角色菜单、Vue 后台布局、路由守卫 |
| M3：知识库管理与 AI 服务 | 1 周 | 知识库 CRUD、审核发布流程、AnythingLLM 适配器（含 vLLM 调用）、embedding 配置、向量写入 |
| M4：智能问答与申告处理 | 2 周 | 问答会话 API、RAG 集成、置信度判断、申告 CRUD、状态机、处理记录、回访、站内消息 |
| M5：数据看板与日志审计 | 1 周 | 看板统计 SQL、趋势图表、审计日志记录、模型配置页面、embedding 配置页面 |
| M6：联调测试与文档完善 | 1 周 | 集成测试、演示数据、API 文档、README、项目结构说明 |
