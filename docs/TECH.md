# 运维数字员工系统技术架构文档

| 项目 | 内容 |
| --- | --- |
| 文档版本 | v1.0 |
| 生成日期 | 2026-05-15 |
| 需求来源 | [docs/PRD.md](./PRD.md) |
| 输出目标 | 运维数字员工系统 MVP 的技术选型、架构、目录、依赖、部署和风险方案 |

## 0. 需求理解与架构假设

本系统面向企业内部运维场景，MVP 聚焦智能问答、申告记录、人工处理、运维账号管理、知识库管理、登录权限和审计日志。PRD 明确要求后端优先 Go、前端优先 Vue、架构参考 Gin-Vue-Admin 的前后端分离与权限模型，并优先支持私有化部署。

核心约束如下：

- 最低运行环境参考 4 核 CPU、8GB 内存、100GB 硬盘。
- 后端提供 REST API，前端提供门户端和后台管理端。
- MVP 模型服务固定使用 vLLM，并通过 OpenAI-compatible 适配层隔离业务代码与模型服务。
- MVP RAG 服务固定通过 AnythingLLM 适配层接入，由 AnythingLLM 负责完整 RAG 流程；PostgreSQL pgvector 作为系统侧向量存储和后续原生检索扩展基础。
- MVP 每个知识库必须配置一个 embedding 模型和一个向量维度，创建知识库时必须从系统配置的模型和维度选项中选择，且同一知识库内所有切片向量必须一致。
- MVP 对象存储固定使用 MinIO，并通过 S3-compatible 适配层接入。
- MVP 问答链路先同步返回完整答案，SSE 流式输出只做目录和接口扩展预留。
- MVP 运维账号管理只做系统内本地模拟，不对接真实企业账号中心。
- MVP 异步任务先满足知识同步和审计留痕，不提前建设完整任务编排平台。
- 问答答案需要保留知识来源、命中文档、置信度和用户反馈。
- 敏感操作必须审计，后台接口必须认证和授权。
- MVP 不做复杂多租户、生产级自愈闭环、大规模日志分析和商业化计费。

架构策略：MVP 采用“模块化单体 + 外部 AI/RAG 服务”的方式交付。业务系统保持一个 Go API 服务，内部按认证、用户权限、知识、问答、申告、账号、日志等模块拆分；AI、RAG、对象存储、缓存、消息队列都以接口适配器接入，后续可按压力和组织边界拆成独立服务。

## 1. 技术选型

### 1.1 技术选型结论

| 层级 | 技术选型 | 选型理由 | 预计效果 | 其他方案 |
| ---- | -------- | -------- | -------- | -------- |
| 前端 | Vue 3 + Vite + TypeScript + Pinia + Vue Router + Element Plus + ECharts | PRD 明确偏好 Vue，Gin-Vue-Admin 也以 Vue/Gin 为基础，并集成动态路由、动态菜单、JWT、Casbin 等后台常用能力；Vue 3、Vite、Pinia、Element Plus 的 GitHub 仓库在 2026-05-14 仍保持活跃，Element Plus 定位为 Vue 3 UI 库，适合后台管理界面。参考：[Gin-Vue-Admin](https://github.com/flipped-aurora/gin-vue-admin)、[Vue](https://github.com/vuejs/core)、[Vite](https://github.com/vitejs/vite)、[Pinia](https://github.com/vuejs/pinia)、[Element Plus](https://github.com/element-plus/element-plus)。 | 快速落地门户端与后台端；TypeScript 降低接口联调成本；Element Plus 提供表单、表格、弹窗、菜单等管理端基础组件；自定义主题可按 `docs/prompts/DESIGN-linear.app.md` 实现 Linear Design。 | React + Ant Design Pro：生态强但与项目 Vue/GVA 约束不一致；Angular：规范强但学习和交付成本偏高；Naive UI：设计轻量但 GVA 对齐度不如 Element Plus。 |
| 后端 | Go 1.26.x + Gin + GORM + Casbin + JWT + Zap + Viper | Go 官方版本文件在 2026-05-14 返回 `go1.26.3`；Gin 是 Go 高性能 HTTP 框架，GORM 提供 Go ORM 能力，Casbin 支持 ACL/RBAC/ABAC 等访问控制模型，Zap 提供结构化分级日志。Gin-Vue-Admin 当前仓库也以 Gin、GORM、Casbin、JWT、Vue 作为核心。参考：[Go version](https://go.dev/VERSION?m=text)、[Gin](https://github.com/gin-gonic/gin)、[GORM](https://github.com/go-gorm/gorm)、[Casbin](https://github.com/apache/casbin)、[Zap](https://github.com/uber-go/zap)。 | 单机资源占用低，适合 4C8GB MVP；REST API、RBAC、审计日志、事务处理和配置管理都能按企业后台模式实现；后续拆分服务时接口边界清晰。 | GoFrame：一体化程度高但与 GVA 参考不完全一致；Fiber：性能好但生态与 Gin/GVA 对齐度弱；Spring Boot：企业生态强但资源占用和课程交付成本更高。 |
| 数据库 | PostgreSQL 18 + pgvector | PostgreSQL 官网首页显示 PostgreSQL 18；pgvector 提供 Postgres 内的向量相似度搜索，支持 exact/approximate nearest neighbor、HNSW、IVFFlat、cosine/L2 等距离，并能保留 Postgres 的事务、JOIN、备份等能力。参考：[PostgreSQL](https://www.postgresql.org/)、[pgvector](https://github.com/pgvector/pgvector)。 | 业务数据、审计数据、知识元数据和系统侧向量统一存储；AnythingLLM 负责完整 RAG 流程，pgvector 保存同知识库同模型同维度的切片向量，用于追溯和后续原生检索扩展。 | MySQL：GVA 默认适配成熟，团队熟悉度可能更高，但向量检索和 JSON/全文能力不如 PostgreSQL 一体化；SQLite：适合 demo，不适合多用户后台；Qdrant：专业向量库，适合后期向量规模增长，但 MVP 不引入。 |
| 缓存 | Redis | Redis 是高性能内存数据结构服务器，支持字符串、哈希、列表、集合、有序集合和 JSON 等结构，适合验证码、登录态黑名单、热点配置、问答限流和短期会话缓存。参考：[Redis](https://github.com/redis/redis)、[Redis Docs](https://redis.io/docs/latest/get-started/)。 | 生态成熟，客户端、监控、运维和云托管选择都最完整；单机部署简单，后续可平滑扩展到 Redis Sentinel/Cluster 或托管服务。 | Memcached：简单但数据结构能力不足；本地内存缓存：简单但多实例一致性差；兼容实现：若后续需要迁移，可继续保持 Redis 协议客户端不变。 |
| 消息队列 | NATS JetStream | NATS Server 使用 Go 编写，JetStream 提供持久化分布式流能力，官方文档定位为 NATS 的 persistence and streaming 层。参考：[NATS Server](https://github.com/nats-io/nats-server)、[NATS JetStream](https://docs.nats.io/nats-concepts/jetstream)。 | 用于知识发布后异步同步 RAG、问答日志异步分析、审计日志落库重试、附件扫描等低耦合任务；比 Kafka 更轻，适合单机和小集群。 | RabbitMQ：路由能力强，运维复杂度中等；Kafka：吞吐强但对 MVP 过重；数据库任务表：组件少但延迟、重试、消费并发能力弱。 |
| 存储 | MinIO + S3-compatible 适配层 | MinIO 是 S3 兼容对象存储，GitHub 描述为 high-performance, S3 compatible object store，官方文档提供容器化部署方式。参考：[MinIO GitHub](https://github.com/minio/minio)、[MinIO Container Docs](https://min.io/docs/minio/container/index.html)。 | MVP 固定使用 MinIO 存储申告附件、知识文档原件、导入导出文件和模型/RAG 同步材料；S3 API 便于后续迁移到企业对象存储。 | 本地文件系统：MVP 最简单但备份和扩容弱；阿里云 OSS/腾讯 COS/华为 OBS：生产托管能力强但私有化课程环境依赖外部云。 |
| AI/RAG | AnythingLLM 完整 RAG 流程 + vLLM OpenAI-compatible 模型适配器 + pgvector | AnythingLLM 仓库定位为 privacy-first AI productivity/RAG 工具，2026-05-14 GitHub API 显示仓库仍活跃；vLLM 提供 OpenAI-compatible Server，便于统一模型调用协议并复用 OpenAI SDK；pgvector 提供 PostgreSQL 内向量检索能力。参考：[AnythingLLM](https://github.com/Mintplex-Labs/anything-llm)、[vLLM](https://github.com/vllm-project/vllm)、[vLLM OpenAI-compatible server](https://docs.vllm.ai/en/stable/serving/openai_compatible_server.html)、[pgvector](https://github.com/pgvector/pgvector)。 | MVP 通过 AnythingLLM 负责知识导入、切分、检索增强和问答编排，通过 vLLM OpenAI-compatible 适配层同步生成答案，通过 pgvector 保存系统侧切片向量；后端只依赖 `RagClient`、`ModelClient` 和 `EmbeddingClient` 接口。 | 自研 RAG：可控性强但交付成本高；LangChain/LlamaIndex：生态成熟但 Python 服务栈增加；Qdrant：专业向量库，适合后期规模增长，但 MVP 不引入。 |
| 部署 | Docker Compose + Nginx/Caddy + Linux 单机优先；预留 Kubernetes | Docker Compose 官方定位为定义和运行多容器应用，适合 MVP 同机编排；后续生产可把无状态 Web/API 横向扩展，并把 PostgreSQL、Redis、NATS、MinIO、RAG/模型拆到独立节点。参考：[Docker Compose](https://docs.docker.com/compose/)、[Kubernetes HPA](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/)。 | 一台测试服务器即可运行完整演示；部署脚本清晰；生产演进路径明确，不提前引入 K8s 复杂度。 | 裸机 systemd：组件少但环境漂移明显；Kubernetes：扩展强但课程/MVP 交付成本高；PaaS：部署快但私有化和数据安全受限。 |

### 1.2 技术选型思考过程

1. 前端优先满足 PRD 和 GVA 约束，因此 Vue 3 是明确方向。React/Angular 不是能力不足，而是会偏离项目“Go + Vue + GVA 参考架构”的约束，导致示例、权限、菜单、表格表单实现都要重新设计。
2. 后端采用 Go + Gin，而不是 Java/Spring Boot，是因为本项目 MVP 需要在 4C8GB 环境运行，且主要是 REST API、RBAC、审计、工单状态流转、RAG 调用编排，Go 的部署体积和资源占用更适合课程原型和企业内网轻量部署。
3. 数据库固定采用 PostgreSQL + pgvector，而不是 MySQL，是因为知识库元数据、JSON 配置、全文检索、向量检索和事务一致性可以在一个数据库生态内完成。
4. AI/RAG 的 MVP 决策是：模型侧使用 vLLM OpenAI-compatible 适配层，RAG 侧由 AnythingLLM 负责完整 RAG 流程，项目内知识向量落在 pgvector。每个知识库必须配置 embedding 模型和向量维度，且同一知识库内必须一致。后端只定义 `RagService`、`EmbeddingService`、`ModelService` 端口，具体实现放在 adapter 层，避免业务层直接绑定第三方 API。
5. 消息队列选 NATS JetStream，MVP 只用于知识同步和审计相关异步任务，避免提前建设完整任务编排平台。
6. 部署先 Docker Compose，是因为它能覆盖本期“可运行原型、完整文档、内网部署”的目标。Kubernetes 只作为扩容路径，不作为 MVP 前置依赖。

### 1.3 候选方案健康度摘要

以下数据来自 2026-05-14 的 GitHub API 检索，用于判断生态活跃度，不作为唯一决策依据。

| 项目 | 仓库 | 语言 | License | Stars | Forks | Open issues | 最近 push | 选型含义 |
| --- | --- | --- | --- | ---: | ---: | ---: | --- | --- |
| Gin-Vue-Admin | [flipped-aurora/gin-vue-admin](https://github.com/flipped-aurora/gin-vue-admin) | Go | Apache-2.0 | 24,671 | 7,077 | 43 | 2026-05-11 | 与项目约束高度一致，可借鉴目录、RBAC、动态菜单和代码分层。 |
| Vue Core | [vuejs/core](https://github.com/vuejs/core) | TypeScript | MIT | 53,673 | 9,112 | 980 | 2026-05-14 | 前端主框架活跃，适合长期维护。 |
| Vite | [vitejs/vite](https://github.com/vitejs/vite) | TypeScript | MIT | 80,602 | 8,175 | 727 | 2026-05-14 | 现代前端构建工具，开发体验好。 |
| Element Plus | [element-plus/element-plus](https://github.com/element-plus/element-plus) | TypeScript | MIT | 27,411 | 19,881 | 1,236 | 2026-05-14 | Vue 3 管理端组件丰富，但需自定义主题以满足 Linear Design。 |
| Gin | [gin-gonic/gin](https://github.com/gin-gonic/gin) | Go | MIT | 88,502 | 8,609 | 685 | 2026-05-09 | Go REST API 生态成熟。 |
| GORM | [go-gorm/gorm](https://github.com/go-gorm/gorm) | Go | MIT | 39,747 | 4,145 | 518 | 2026-05-11 | ORM 成熟，适合 CRUD、事务和迁移辅助。 |
| Casbin | [apache/casbin](https://github.com/apache/casbin) | Go | Apache-2.0 | 20,110 | 1,736 | 71 | 2026-05-06 | RBAC/ABAC 权限模型成熟，适合菜单和接口鉴权。 |
| pgvector | [pgvector/pgvector](https://github.com/pgvector/pgvector) | C | NOASSERTION | 21,284 | 1,171 | 12 | 2026-04-27 | PostgreSQL 内向量检索能力强，issue 数低。 |
| NATS Server | [nats-io/nats-server](https://github.com/nats-io/nats-server) | Go | Apache-2.0 | 19,787 | 1,798 | 497 | 2026-05-14 | Go 生态消息系统，适合轻量异步。 |
| AnythingLLM | [Mintplex-Labs/anything-llm](https://github.com/Mintplex-Labs/anything-llm) | JavaScript | MIT | 60,033 | 6,489 | 359 | 2026-05-13 | MVP 私有 RAG 工具成熟度较高，但要通过适配层接入以降低锁定。 |
| Qdrant | [qdrant/qdrant](https://github.com/qdrant/qdrant) | Rust | Apache-2.0 | 31,312 | 2,261 | 549 | 2026-05-14 | 后续专业向量检索扩展选择。 |
| MinIO | [minio/minio](https://github.com/minio/minio) | Go | AGPL-3.0 | 60,930 | 7,492 | 81 | 2026-04-24 | S3 兼容对象存储强，但 AGPL 许可证需在商用分发前复核。 |

## 2. 系统架构图（ASCII）

```text
                          +---------------------------+
                          |  浏览器 / 企业内网客户端  |
                          |  门户端 + 后台管理端      |
                          +-------------+-------------+
                                        |
                                        | HTTPS / REST(同步问答) / SSE(后续预留)
                                        v
+---------------------+      +---------+----------+       +----------------------+
| Nginx / Caddy       |-----> | Vue 3 静态资源    |       | 运维人员 / 管理员    |
| TLS / 反向代理      |      | portal + admin     |       | RBAC 菜单权限        |
+----------+----------+      +--------------------+       +----------+-----------+
           |                                                       |
           | /api                                                  |
           v                                                       v
+------------------------------ Go API Server --------------------------------+
| Gin Router                                                                  |
|  ├─ Auth/RBAC Middleware: JWT + Casbin + 审计上下文                         |
|  ├─ Portal API: 智能问答、申告提交、进度查询、反馈记录                       |
|  ├─ Admin API: 申告处理、账号管理、知识审核、系统配置、日志查询               |
|  ├─ Domain Services: Chat / Ticket / Knowledge / Account / Audit / Dashboard |
|  ├─ Repository: PostgreSQL 事务、分页、状态流转、审计写入                    |
|  └─ Adapters: RAG / LLM / Cache / Queue / Object Storage / Observability     |
+---------+--------------+--------------+--------------+-------------+---------+
          |              |              |              |             |
          | SQL          | Cache        | Event        | S3 API      | HTTP API
          v              v              v              v             v
+---------+------+ +-----+------+ +-----+--------+ +---+-------+ +---+----------------+
| PostgreSQL 18  | | Redis      | | NATS        | | MinIO     | | RAG Service       |
| 业务库/审计库  | | 会话/限流  | | JetStream   | | 附件/文档 | | AnythingLLM       |
| pgvector 向量  | | 热点配置   | | 同步/审计  | | 导入导出  | | 完整 RAG 流程    |
+---------+------+ +-----+------+ +-----+--------+ +---+-------+ +---+--------+-------+
          |                           |                                |
          | backups                   | async sync                     | prompt + context
          v                           v                                v
+---------+------+            +-------+---------+              +-------+---------+
| 备份存储/归档  |            | Worker 任务    |              | Model Gateway |
| SQL dump/WAL   |            | 知识同步/审计  |              | vLLM              |
+----------------+            +-----------------+              | OpenAI-compatible |
                                                               +-------+---------+
                                                                       |
                                                                       v
                                                        +--------------+-------------+
                                                        | 本地模型 / 独立模型节点    |
                                                        +----------------------------+

             +----------------------------------------------------------+
             | Observability: Prometheus + Grafana + Loki + OpenTelemetry |
             | 指标、日志、链路追踪、告警规则、运行仪表盘                 |
             +----------------------------------------------------------+
```

### 2.1 主要业务数据流

智能问答数据流：

1. 前端调用 `POST /api/v1/portal/chat-sessions`，提交问题、知识库编码和用户联系方式。
2. Go API 写入问答会话初始记录，并调用 `RagService.Chat` 提交问题、会话编号和知识库标识。
3. `RagService` 通过 AnythingLLM 适配层完成检索增强和 RAG 编排。
4. AnythingLLM 通过后端配置的 vLLM OpenAI-compatible 适配层完成模型推理。
5. vLLM 同步返回完整答案后，Go API 保存答案、来源、置信度、耗时、模型参数和命中信息。
6. 前端展示答案、来源、反馈按钮和“转人工申告”入口。

申告处理数据流：

1. 用户从问答失败或门户入口提交申告。
2. Go API 使用事务写入申告主表、问答上下文、附件元数据和初始处理记录。
3. 运维人员后台筛选待处理申告，更新状态、处理过程和回访结果。
4. 处理完成后可生成知识候选，进入知识审核流程。
5. 知识审核通过后发布，触发 NATS 事件异步同步到 AnythingLLM，并按知识库配置将同模型同维度切片向量写入 pgvector，同时写入审计日志。

知识发布数据流：

1. 管理员维护 FAQ、处理方案或上传文档。
2. PostgreSQL 记录草稿、审核、发布和停用状态。
3. 发布时生成 `knowledge.published` 事件。
4. Worker 消费事件，将知识内容、标签、分类和来源同步到 AnythingLLM，由 AnythingLLM 负责完整 RAG 流程。
5. Worker 按知识库配置生成或接收切片向量，校验 embedding 模型和向量维度后写入 pgvector。
6. 同步结果回写 `rag_sync_status` 和同步记录表，失败时支持重试、人工查看错误和审计追踪。

账号权限数据流：

1. 用户登录 `POST /api/v1/auth/login`。
2. 后端校验密码哈希、账号状态和角色。
3. 返回 JWT、用户信息、菜单权限、接口权限。
4. 前端动态加载菜单；后端中间件对每个后台接口执行认证和 Casbin 授权。
5. 冻结账号、发布知识、处理申告、修改角色等敏感操作写入审计日志。

## 3. 目录结构

### 3.1 总体目录

```text
OpsMind/
├─ server/                         # Go 后端服务
├─ web/                            # Vue 3 前端，包含门户端和后台端
├─ deploy/                         # Docker Compose、Nginx、监控、初始化脚本
├─ docs/                           # 项目文档：PRD、TECH、DB、API、FLOW、PLAN、TEST、DEPLOY
├─ scripts/                        # 开发、构建、备份、初始化脚本
├─ .env.example                    # 根级示例环境变量，禁止提交真实密钥
├─ README.md                       # 项目结构、启动方式、部署说明
└─ AGENTS.md                       # Codex/Agent 项目约束
```

### 3.2 后端目录

```text
server/
├─ cmd/
│  └─ opsmind/
│     └─ main.go                   # 后端入口：加载配置、初始化依赖、启动 HTTP 服务
├─ configs/
│  ├─ config.yaml                  # 默认配置：端口、数据库、缓存、对象存储、模型网关
│  ├─ config.dev.yaml              # 开发环境覆盖配置
│  └─ config.prod.yaml             # 生产环境覆盖配置，不包含真实密钥
├─ internal/
│  ├─ bootstrap/
│  │  ├─ app.go                    # 应用装配：DB、Cache、Queue、Logger、Adapters
│  │  └─ migrate.go                # 启动时迁移检查和基础数据初始化
│  ├─ router/
│  │  ├─ router.go                 # Gin 路由注册入口
│  │  ├─ portal.go                 # 前台门户路由
│  │  └─ admin.go                  # 后台管理路由
│  ├─ middleware/
│  │  ├─ auth.go                   # JWT 认证
│  │  ├─ rbac.go                   # Casbin 接口授权
│  │  ├─ audit.go                  # 敏感操作审计上下文
│  │  ├─ recovery.go               # panic 保护和统一错误返回
│  │  └─ rate_limit.go             # 登录、问答、申告限流
│  ├─ api/
│  │  └─ v1/
│  │     ├─ auth_handler.go        # 登录、退出、刷新令牌
│  │     ├─ chat_handler.go        # 智能问答与反馈
│  │     ├─ ticket_handler.go      # 申告提交、查询、后台处理
│  │     ├─ file_handler.go        # 门户附件上传、后台文件查询和下载
│  │     ├─ knowledge_handler.go   # 知识库增删改查、审核、发布
│  │     ├─ account_handler.go     # 运维账号管理
│  │     ├─ role_handler.go        # 角色列表、权限树、角色权限维护
│  │     ├─ config_handler.go      # 系统配置、连接测试、模型选项
│  │     ├─ audit_handler.go       # 审计日志查询
│  │     └─ dashboard_handler.go   # 数据看板
│  ├─ domain/
│  │  ├─ auth/                     # 认证、令牌、密码策略领域逻辑
│  │  ├─ user/                     # 用户、角色、菜单、Casbin policy
│  │  ├─ chat/                     # 问答会话、答案来源、反馈
│  │  ├─ ticket/                   # 申告状态机、处理记录、回访
│  │  ├─ knowledge/                # 知识条目、审核、发布、RAG 同步状态
│  │  ├─ ops_account/              # 运维账号新增、修改、冻结、恢复
│  │  ├─ audit/                    # 审计事件、敏感操作记录
│  │  └─ dashboard/                # 指标聚合、解决率、命中率
│  ├─ service/
│  │  ├─ chat_service.go           # 编排 RAG 检索、模型生成、问答记录
│  │  ├─ ticket_service.go         # 申告创建、状态流转、知识候选生成
│  │  ├─ knowledge_service.go      # 知识审核发布和同步事件
│  │  ├─ account_service.go        # 运维账号业务规则
│  │  └─ permission_service.go     # 菜单和接口权限管理
│  ├─ repository/
│  │  ├─ tx.go                     # 事务封装
│  │  ├─ chat_repo.go              # 问答会话、答案、反馈持久化
│  │  ├─ ticket_repo.go            # 申告和处理记录持久化
│  │  ├─ knowledge_repo.go         # 知识条目和同步状态持久化
│  │  ├─ account_repo.go           # 运维账号持久化
│  │  └─ audit_repo.go             # 审计日志持久化
│  ├─ model/
│  │  ├─ entity/                   # GORM 实体，对应数据库表
│  │  ├─ request/                  # API 请求 DTO
│  │  └─ response/                 # API 响应 DTO
│  ├─ adapter/
│  │  ├─ rag/
│  │  │  ├─ client.go              # RAG 统一端口定义
│  │  │  ├─ anythingllm.go         # AnythingLLM 完整 RAG 流程适配器
│  │  │  ├─ pgvector.go            # pgvector 向量存储和检索适配器
│  │  │  └─ qdrant.go              # Qdrant 预留适配器
│  │  ├─ llm/
│  │  │  ├─ client.go              # 模型统一端口定义
│  │  │  ├─ openai_compatible.go   # OpenAI 兼容模型网关适配器
│  │  │  └─ vllm.go                # vLLM OpenAI-compatible 服务适配器
│  │  ├─ cache/                    # Redis 客户端、限流、短期会话缓存
│  │  ├─ queue/                    # NATS JetStream 发布和消费
│  │  ├─ storage/                  # MinIO/S3、本地存储适配器
│  │  └─ observability/            # Prometheus metrics、trace、日志字段
│  ├─ worker/
│  │  ├─ knowledge_sync_worker.go  # 消费知识发布事件，同步 RAG
│  │  ├─ audit_worker.go           # 审计异步落库或归档
│  │  └─ cleanup_worker.go         # 过期会话、临时文件清理
│  └─ pkg/
│     ├─ errors/                   # 统一错误码和错误包装
│     ├─ response/                 # 统一响应结构
│     ├─ pagination/               # 分页参数和响应
│     ├─ validator/                # 请求校验和业务校验
│     └─ security/                 # 密码哈希、脱敏、密钥读取
├─ migrations/
│  ├─ 000001_init_schema.up.sql    # 初始表结构
│  └─ 000001_init_schema.down.sql  # 回滚脚本
├─ tests/
│  ├─ integration/                 # API、数据库、RAG mock 集成测试
│  └─ fixtures/                    # 演示数据和测试数据
├─ go.mod
└─ go.sum
```

### 3.3 前端目录

```text
web/
├─ index.html
├─ package.json
├─ vite.config.ts                  # Vite 构建配置、代理配置、别名
├─ tsconfig.json
├─ src/
│  ├─ main.ts                      # Vue 应用入口
│  ├─ App.vue                      # 根组件
│  ├─ api/
│  │  ├─ request.ts                # Axios 实例、token 注入、错误处理
│  │  ├─ auth.ts                   # 登录、退出、权限接口
│  │  ├─ portalChat.ts             # 门户问答接口
│  │  ├─ portalTickets.ts          # 门户申告接口
│  │  ├─ adminTickets.ts           # 后台申告接口
│  │  ├─ knowledgeBases.ts         # 知识库接口
│  │  ├─ knowledgeArticles.ts      # 知识条目接口
│  │  ├─ knowledgeCandidates.ts    # 知识候选接口
│  │  ├─ accounts.ts               # 运维账号接口
│  │  ├─ roles.ts                  # 角色权限接口
│  │  ├─ files.ts                  # 附件上传和文件查询接口
│  │  ├─ configs.ts                # 系统配置接口
│  │  ├─ audit.ts                  # 审计、操作、登录日志接口
│  │  └─ dashboard.ts              # 数据看板接口
│  ├─ assets/
│  │  ├─ fonts/                    # Inter Variable、Berkeley Mono 可选字体
│  │  └─ images/                   # 产品图、空状态图、登录背景等
│  ├─ design/
│  │  ├─ tokens.css                # Linear Design 色彩、边框、阴影、字体变量
│  │  ├─ element-theme.scss        # Element Plus 主题覆盖
│  │  └─ layout.css                # 页面级布局、响应式规则
│  ├─ components/
│  │  ├─ AppShell.vue              # 后台应用壳：侧栏、顶栏、内容区
│  │  ├─ PortalShell.vue           # 门户应用壳
│  │  ├─ ChatPanel.vue             # 问答输入和答案展示
│  │  ├─ SourceList.vue            # 知识来源列表
│  │  ├─ TicketStatusTag.vue       # 申告状态标签
│  │  ├─ AuditTimeline.vue         # 处理/审计时间线
│  │  └─ PermissionButton.vue      # 基于权限控制按钮显示
│  ├─ composables/
│  │  ├─ useAuth.ts                # 当前用户、权限、登录态
│  │  ├─ useTableQuery.ts          # 表格分页筛选
│  │  ├─ useConfirmAction.ts       # 二次确认和敏感操作
│  │  └─ useSseChat.ts             # 流式问答预留
│  ├─ router/
│  │  ├─ index.ts                  # 路由入口
│  │  ├─ portal.ts                 # 门户端路由
│  │  ├─ admin.ts                  # 后台静态路由
│  │  └─ permission.ts             # 动态路由和权限守卫
│  ├─ stores/
│  │  ├─ auth.ts                   # 用户、token、权限
│  │  ├─ app.ts                    # 主题、布局、菜单折叠
│  │  └─ dictionary.ts             # 字典项、状态枚举
│  ├─ views/
│  │  ├─ portal/
│  │  │  ├─ ChatHome.vue           # 前台问答首页
│  │  │  ├─ TicketCreate.vue       # 申告提交
│  │  │  └─ TicketQuery.vue        # 申告进度查询
│  │  └─ admin/
│  │     ├─ Login.vue              # 后台登录
│  │     ├─ Dashboard.vue          # 数据看板
│  │     ├─ tickets/               # 申告列表、详情、处理
│  │     ├─ knowledge/             # 知识列表、编辑、审核、发布
│  │     ├─ accounts/              # 运维账号管理
│  │     ├─ system/                # 用户、角色、菜单、配置
│  │     └─ audit/                 # 审计日志
│  ├─ utils/
│  │  ├─ format.ts                 # 时间、状态、文件大小格式化
│  │  ├─ permission.ts             # 权限判断
│  │  └─ constants.ts              # 枚举常量
│  └─ types/
│     ├─ api.ts                    # 通用 API 类型
│     ├─ ticket.ts                 # 申告类型
│     ├─ knowledge.ts              # 知识类型
│     └─ user.ts                   # 用户权限类型
└─ public/
   └─ favicon.svg
```

### 3.4 部署与文档目录

```text
deploy/
├─ docker-compose.yml              # 本地开发、课程演示和单机 MVP 编排
├─ nginx/
│  └─ default.conf                 # 静态资源和 API 反代配置
├─ postgres/
│  ├─ init.sql                     # 数据库初始化
│  └─ backup.ps1                   # Windows 环境备份脚本示例
├─ observability/
│  ├─ prometheus.yml               # Prometheus 抓取配置
│  ├─ loki.yml                     # Loki 日志配置
│  └─ grafana/                     # 仪表盘 JSON
└─ README.md                       # 部署说明

docs/
├─ PRD.md                          # 产品需求文档
├─ TECH.md                         # 技术架构文档
├─ DB.md                           # 数据库设计文档
├─ API/                            # API 设计文档和 OpenAPI 入口
├─ FLOW/                           # 业务流程文档
├─ TEST/                           # 测试方案和里程碑测试计划
├─ PLAN.md                         # 项目实施计划
├─ DEPLOY.md                       # 部署文档
└─ prompts/                        # 文档生成提示词和设计规范
```

## 4. 第三方依赖

| 类型 | 依赖 | MVP 是否必须 | 接入方式 | 说明 |
| --- | --- | --- | --- | --- |
| 模型服务 | vLLM | 必须 | 后端 `ModelClient` HTTP 适配器 | MVP 统一使用 OpenAI-compatible 请求/响应结构，由 vLLM 提供模型推理服务。 |
| RAG 服务 | AnythingLLM | 必须 | 后端 `RagClient` HTTP 适配器 | MVP 通过适配层接入 AnythingLLM，由 AnythingLLM 负责完整 RAG 流程。 |
| 向量数据库 | pgvector | 必须 | PostgreSQL SQL 适配器 | MVP 使用 pgvector 存储知识切片向量；同一知识库必须使用同一个 embedding 模型和同一个向量维度。 |
| 对象存储 | MinIO | 必须 | S3-compatible API | MVP 通过 S3 适配层接入 MinIO，存储附件、知识文档原件、导入导出文件。 |
| 缓存 | Redis | 推荐 | Redis 协议客户端 | 登录态黑名单、验证码、限流、热点字典。 |
| 消息队列 | NATS JetStream | 必须 | NATS Go client | MVP 先满足知识同步和审计异步任务，后续再扩展任务编排。 |
| 监控指标 | Prometheus | 生产推荐 | `/metrics` 抓取 | Prometheus 官方定位为 metrics collection and alerting，适合 API 延迟、错误率、队列积压监控。 |
| 可视化 | Grafana | 生产推荐 | Prometheus/Loki 数据源 | 展示运行仪表盘、问答耗时、工单量、命中率。 |
| 日志聚合 | Loki | 生产推荐 | Promtail/Vector 或 Docker 日志采集 | Loki 官方面向日志聚合，适合与 Grafana 同套观测平台集成。 |
| 链路追踪 | OpenTelemetry | 后续推荐 | Go SDK / Collector | OpenTelemetry 覆盖 traces、metrics、logs，便于定位 RAG/模型调用慢点。 |
| 通知服务 | 企业微信、钉钉、邮件 SMTP | P1 可选 | Webhook/SMTP 适配器 | 用于高优先级申告提醒和知识审核通知。 |
| 支付/短信 | 无 | 不需要 | 不接入 | PRD 不包含商业计费和短信验证码，MVP 不引入。 |

## 5. 部署方案

### 5.1 部署环境

开发环境：

- 操作系统：Windows 11 + Docker Desktop 或 Linux。
- 后端：Go 1.26.x。
- 前端：Node.js LTS、pnpm。
- 数据库与中间件：Docker Compose 启动 PostgreSQL、Redis、NATS、MinIO。
- AI/RAG：AnythingLLM、vLLM 通过 Docker Compose 或独立节点运行；4C8GB 环境建议将 vLLM 放在独立 GPU 节点，业务服务仍通过 OpenAI-compatible 接口调用。

测试/演示环境：

- Linux 单机，最低 4C8GB/100GB。
- Docker Compose 编排 `web`、`server`、`postgres`、`redis`、`nats`、`minio`、`anythingllm`、`vllm`。
- Nginx/Caddy 统一 TLS 和反向代理。
- PostgreSQL、MinIO、Redis（如启用持久化）、AnythingLLM 数据目录挂载到宿主机持久卷；vLLM 模型权重按独立卷或 GPU 节点管理。

生产小规模环境：

- 推荐至少 8C16GB，vLLM 模型服务最好独立到 GPU 节点，业务 API 通过 OpenAI-compatible 接口访问。
- PostgreSQL 独立磁盘和定期备份。
- MinIO 使用独立数据盘。
- Redis/NATS 可先单节点，后续按业务重要性切高可用。
- 监控、日志、告警必须启用。

### 5.2 单机 Compose 拓扑

```text
opsmind-network
├─ nginx:80/443
├─ web: static files
├─ server: Go API
├─ postgres:5432
├─ redis:6379
├─ nats:4222/8222
├─ minio:9000/9001
├─ anythingllm:3001
├─ vllm:8000
├─ prometheus:9090
├─ grafana:3000
└─ loki:3100
```

### 5.3 扩缩容策略

前端：

- 构建为静态资源，通过 Nginx/Caddy 服务。
- 多实例部署时无需状态同步。

后端：

- Go API 保持无状态，登录态以 JWT 为主，短期黑名单和限流状态放 Redis。
- 横向扩展时多个 API 实例共用 PostgreSQL、Redis、NATS、MinIO。
- RAG 和模型调用设置超时、重试、熔断和降级到转人工。

数据库：

- MVP 单主 PostgreSQL。
- 通过定时备份、WAL 归档和只读副本预留恢复与分析能力。
- 知识与工单数据量增长后，再评估按时间分区或冷热分离。

中间件：

- Redis 只承担缓存和短期状态，不放核心业务事实数据。
- NATS JetStream 用于异步同步和重试，不承载高价值唯一事实源。
- MinIO 按桶和前缀管理附件、导入包和知识原文。

模型与 RAG：

- 模型服务优先外置，避免和业务 API 争抢内存；业务服务只调用 vLLM OpenAI-compatible 接口。
- AnythingLLM MVP 阶段默认单实例即可，负责完整 RAG 流程；pgvector 与 PostgreSQL 同库部署，用于系统侧向量存储。
- 知识库配置保存 embedding 模型和向量维度，发布和同步时必须校验同知识库下切片向量一致性。
- 问答链路先同步返回完整答案，并设置超时、降级和人工兜底；SSE 流式输出后续再补。

### 5.4 监控告警

监控目标：

- API 延迟、错误率、请求量、登录失败率。
- 问答同步返回耗时、转人工比例、知识命中率；首字响应时间作为后续 SSE 流式输出指标预留。
- 申告队列积压、处理超时、状态停留时长。
- PostgreSQL 连接数、慢查询、磁盘空间、备份结果。
- Redis 内存占用、命中率、淘汰数。
- NATS JetStream 消费滞后、重试次数、失败消息。
- MinIO 容量、对象写入失败、桶访问异常。

告警策略：

- P0 接口错误率持续升高或问答链路不可用时，立即通知管理员和值班运维。
- 申告积压超过阈值时通知运维人员。
- 数据库磁盘、水位、备份失败、日志采集失败必须告警。
- 模型或 RAG 不可用时，前端直接显示转人工入口，不阻塞用户提交。

告警渠道：

- 企业微信、钉钉、邮件至少接入一种。
- 演示环境可先仅保留站内告警和日志提示。

## 6. 风险与应对

| 风险 | 影响 | 应对措施 |
| --- | --- | --- |
| 知识库质量不足 | 问答命中率低，用户体验差 | 先覆盖高频 FAQ，建立人工回流和审核机制，保留来源展示和转人工入口。 |
| 模型幻觉或回答不稳定 | 错误指引、用户误操作 | 强制展示来源片段、限定回答模板、设置低置信度兜底。 |
| 4C8GB 资源不足 | 模型推理慢，系统卡顿 | 模型外置或使用小模型，限制上下文长度，将模型服务与业务 API 分离。 |
| RAG 或向量检索锁定 | 后续替换成本高 | MVP 固定 AnythingLLM 完整 RAG 流程 + pgvector 向量存储，但后端统一 `RagClient` 和 `EmbeddingClient` 接口，后续可替换为 Qdrant 或自研检索。 |
| 权限模型设计不严谨 | 敏感数据泄露或误操作 | 采用 JWT + Casbin，菜单和接口双重鉴权，敏感操作强制审计。 |
| 单机部署单点故障 | 业务中断 | PostgreSQL、MinIO、配置和备份要可恢复，后续再演进到副本和多实例。 |
| 目录与模块膨胀 | 代码难维护 | 按认证、问答、申告、知识、账号、日志、配置分层，避免把业务逻辑塞进 handler。 |
| 观测缺失 | 问题难定位 | 默认接入结构化日志、指标和基础追踪字段。 |
| 对象存储许可证与合规 | 商用或分发存在不确定性 | 交付前复核 MinIO 和替代对象存储的许可证与部署边界。 |

## 7. 验证建议

建议先做三个最小验证：

1. 问答验证：接入 vLLM OpenAI-compatible 服务、AnythingLLM 和一组 20 条高频运维 FAQ，验证同步回答、来源展示和转人工流程。
2. 申告验证：完成申告提交、后台处理、状态流转和回访记录，确认事务一致性和审计日志可追踪。
3. 知识发布验证：从草稿到审核再到发布，验证知识同步到 AnythingLLM、pgvector 同模型同维度约束、审计留痕和失败重试。

验收指标建议：

- 后台常规接口在本地环境下稳定返回。
- 问答链路能返回答案或明确兜底提示。
- 申告状态变化和处理记录一致。
- 账号冻结后不可登录后台。
- 审计日志能还原关键敏感操作。

## 8. 结论

本项目 MVP 技术路线固定为：前端采用 Vue 3 + Vite + TypeScript + Pinia + Element Plus，后端采用 Go + Gin + GORM + Casbin + JWT + Zap，业务数据和知识向量以 PostgreSQL 18 + pgvector 为中心，缓存选 Redis，异步任务选 NATS JetStream 且先满足知识同步和审计，对象存储选 MinIO + S3-compatible 适配层，AI/RAG 采用 vLLM OpenAI-compatible 适配层 + AnythingLLM 完整 RAG 流程适配层，知识库 embedding 模型和维度可配置但同库必须一致，问答先同步返回完整答案，部署以 Docker Compose 为 MVP 起点，监控以 Prometheus + Grafana + Loki + OpenTelemetry 为主。

这条路线的核心价值不是“组件最多”，而是把 MVP 的复杂度压在单一 Go 服务和少量成熟基础设施上，同时保留后续拆分和扩展空间。

## 9. Sources

- Source: [Gin-Vue-Admin README](https://github.com/flipped-aurora/gin-vue-admin)
  - Accessed: 2026-05-14
  - Date: not visible
  - Reliability: GitHub
  - Supports: GVA 以 Gin + Vue 3 + JWT + Casbin + Gorm 为核心参考架构
- Source: [Vue Core repository](https://github.com/vuejs/core)
  - Accessed: 2026-05-14
  - Date: not visible
  - Reliability: GitHub
  - Supports: Vue 3 仍在活跃维护，适合前端主框架
- Source: [Vite repository](https://github.com/vitejs/vite)
  - Accessed: 2026-05-14
  - Date: not visible
  - Reliability: GitHub
  - Supports: 现代前端构建工具适合管理端开发
- Source: [Element Plus repository](https://github.com/element-plus/element-plus)
  - Accessed: 2026-05-14
  - Date: not visible
  - Reliability: GitHub
  - Supports: Vue 3 组件库适合表单、表格、弹窗和管理端布局
- Source: [Pinia repository](https://github.com/vuejs/pinia)
  - Accessed: 2026-05-14
  - Date: not visible
  - Reliability: GitHub
  - Supports: Vue 状态管理方案适合前端权限和布局状态
- Source: [Gin repository](https://github.com/gin-gonic/gin)
  - Accessed: 2026-05-14
  - Date: not visible
  - Reliability: GitHub
  - Supports: Go REST API 框架成熟且活跃
- Source: [GORM repository](https://github.com/go-gorm/gorm)
  - Accessed: 2026-05-14
  - Date: not visible
  - Reliability: GitHub
  - Supports: Go ORM 适合 CRUD、事务和迁移辅助
- Source: [Casbin repository](https://github.com/apache/casbin)
  - Accessed: 2026-05-14
  - Date: not visible
  - Reliability: GitHub
  - Supports: RBAC/ABAC 权限模型适合菜单和接口鉴权
- Source: [Zap repository](https://github.com/uber-go/zap)
  - Accessed: 2026-05-14
  - Date: not visible
  - Reliability: GitHub
  - Supports: Go 结构化日志适合审计和排障
- Source: [Go version](https://go.dev/VERSION?m=text)
  - Accessed: 2026-05-14
  - Date: 2026-05-04T20:36:18Z
  - Reliability: official docs
  - Supports: 当前 Go 最新稳定版本为 go1.26.3
- Source: [PostgreSQL home](https://www.postgresql.org/)
  - Accessed: 2026-05-14
  - Date: not visible
  - Reliability: official docs
  - Supports: PostgreSQL 18 当前可用
- Source: [pgvector README](https://github.com/pgvector/pgvector)
  - Accessed: 2026-05-14
  - Date: not visible
  - Reliability: GitHub
  - Supports: Postgres 向量相似度检索、HNSW、IVFFlat 和 ACID 能力
- Source: [Redis repository](https://github.com/redis/redis)
  - Accessed: 2026-05-14
  - Date: not visible
  - Reliability: GitHub
  - Supports: 高性能 key/value 缓存服务和文档/向量查询引擎
- Source: [Redis docs](https://redis.io/docs/latest/get-started/)
  - Accessed: 2026-05-14
  - Date: not visible
  - Reliability: official docs
  - Supports: Redis 作为缓存、文档数据库、流引擎和消息 broker 的定位
- Source: [NATS JetStream docs](https://docs.nats.io/nats-concepts/jetstream)
  - Accessed: 2026-05-14
  - Date: not visible
  - Reliability: official docs
  - Supports: 持久化流和异步任务编排
- Source: [MinIO container docs](https://min.io/docs/minio/container/index.html)
  - Accessed: 2026-05-14
  - Date: not visible
  - Reliability: official docs
  - Supports: S3 兼容对象存储的容器化部署
- Source: [AnythingLLM repository](https://github.com/Mintplex-Labs/anything-llm)
  - Accessed: 2026-05-14
  - Date: not visible
  - Reliability: GitHub
  - Supports: 私有化 RAG 工具适合作为 MVP 适配对象
- Source: [vLLM repository](https://github.com/vllm-project/vllm)
  - Accessed: 2026-05-14
  - Date: not visible
  - Reliability: GitHub
  - Supports: 高吞吐、内存高效的模型推理与服务引擎
- Source: [vLLM OpenAI-compatible server docs](https://docs.vllm.ai/en/stable/serving/openai_compatible_server.html)
  - Accessed: 2026-05-14
  - Date: not visible
  - Reliability: official docs
  - Supports: 统一模型调用协议并复用 OpenAI 客户端
- Source: [Qdrant repository](https://github.com/qdrant/qdrant)
  - Accessed: 2026-05-14
  - Date: not visible
  - Reliability: GitHub
  - Supports: 专业向量数据库适合作为后续扩展
- Source: [Docker Compose docs](https://docs.docker.com/compose/)
  - Accessed: 2026-05-14
  - Date: not visible
  - Reliability: official docs
  - Supports: 多容器 MVP 编排
- Source: [Prometheus overview](https://prometheus.io/docs/introduction/overview/)
  - Accessed: 2026-05-14
  - Date: not visible
  - Reliability: official docs
  - Supports: 指标采集与告警
- Source: [Grafana Loki overview](https://grafana.com/docs/loki/latest/get-started/overview/)
  - Accessed: 2026-05-14
  - Date: not visible
  - Reliability: official docs
  - Supports: 日志聚合
- Source: [OpenTelemetry overview](https://opentelemetry.io/docs/what-is-opentelemetry/)
  - Accessed: 2026-05-14
  - Date: not visible
  - Reliability: official docs
  - Supports: traces、metrics、logs 统一观测
