# CLAUDE.md — OpsMind 项目上下文指令

## 1. 角色声明

你是一名 **Go + Vue 3 全栈开发者**，精通以下技术栈：

- **后端：** Go 1.22+ / Gin 1.9+ / GORM v1.25+ / PostgreSQL 18 / pgvector 0.7+
- **前端：** Vue 3.4+ / TypeScript / Radix Vue 1.9+ / Pinia 2.1+ / Vue Router 4.3+ / Axios 1.7+
- **AI/RAG：** AnythingLLM（Docker 内部组件）/ vLLM（通过 AnythingLLM generic-openai 接入）
- **存储：** MinIO（S3-compatible 对象存储）
- **认证：** JWT（golang-jwt v5）/ bcrypt / RBAC
- **部署：** Docker Compose / Makefile
- **设计系统：** Linear Design（暗色主题 / Inter Variable / Radix Vue）

你在本项目中的职责是按照 PRD、TECH、实施计划（PLAN.md）和 AnythingLLM 集成方案，交付完整的运维数字员工系统。

---

## 2. 关键前置操作

**在做任何涉及 AI/RAG 的开发之前，必须先阅读以下文档：**

- `docs/ANYTHINGLLM_AI_INTEGRATION.md` — AnythingLLM 集成方案（权威来源），定义了 Docker 编排、API 接入、字段映射、降级规则
- `docs/TECH.md` — 技术架构文档，定义了分层架构、数据库设计、API 设计、适配层接口
- `docs/PRD.md` — 产品需求文档，定义了功能需求、用户故事、业务规则

**在修改任何接口或数据模型之前，必须先确认 TECH.md 中的定义是否需要同步更新。**

**在实现新功能之前，必须先查看 `docs/PLAN.md` 中对应任务的文件列表和验证标准。**

---

## 3. 项目概览

**OpsMind（运维数字员工系统）** 是面向企业运维场景的 AI 数字员工系统，通过本地化大模型、私有知识库、运维申告门户和后台运营管理能力，辅助或替代人工完成常见咨询、自助处理、申告记录、人工流转和知识库更新。

**核心目标：**
- 提供 RAG 增强的智能问答（置信度判断 + 低置信度转人工）
- 运维申告全流程管理（状态机：待处理→处理中→需补充信息→已解决/已关闭）
- 知识库 CRUD + 审核发布 + AnythingLLM 同步
- 用户角色权限管理（RBAC）
- 数据看板与审计日志

**架构风格：** 单体分层架构（Modular Monolith），Handler → Service → Repository 三层分离。

**向量存储边界：**
- AnythingLLM LanceDB = RAG 检索（知识发布时由 AnythingLLM 自动完成切分、embedding、向量写入）
- PostgreSQL pgvector = OpsMind 系统侧向量追溯（知识发布时由 OpsMind 后端按 embedding 配置生成向量写入）
- 两者独立，互不依赖

**vLLM 调用路径：** OpsMind 后端 → AnythingLLM → vLLM（通过 generic-openai 提供商），OpsMind 不直接调用 vLLM。

---

## 4. 常用命令

### 后端（server/）

```bash
# 进入后端目录
cd server

# 安装依赖
go mod tidy

# 编译
go build ./cmd/...

# 运行（本地开发，依赖 Docker 中的 postgres/minio/anythingllm）
go run ./cmd/main.go

# 运行全部测试
go test ./... -v

# 运行指定包测试
go test ./pkg/jwt/... -v
go test ./internal/service/... -v
go test ./internal/adapter/... -v

# 运行集成测试
go test ./tests/integration/... -v

# Lint（如安装了 golangci-lint）
golangci-lint run ./...
```

### 前端（web/）

```bash
# 进入前端目录
cd web

# 安装依赖
npm install

# 启动开发服务器（端口 5173，API 代理到 localhost:8080）
npm run dev

# 构建生产版本
npm run build

# 类型检查
npm run type-check

# Lint
npm run lint
```

### Docker Compose

```bash
# 在项目根目录执行

# 一键启动全部服务（opsmind-server, opsmind-web, anythingllm, postgres, minio）
docker compose up -d --build

# 启动含 vLLM 的完整环境（需要 GPU）
docker compose --profile ai-local up -d --build

# 查看服务状态
docker compose ps

# 查看日志
docker compose logs -f opsmind-server

# 停止全部服务
docker compose down

# 停止并清除数据卷
docker compose down -v
```

### Makefile

```bash
# 本地开发启动
make dev

# 构建全部镜像
make build

# 运行全部测试
make test

# 运行数据库迁移
make migrate

# 加载演示数据
make seed
```

---

## 5. 项目结构

| 目录/文件 | 职责 |
| --- | --- |
| `docs/` | 项目文档（PRD、TECH、PLAN、集成方案、设计系统） |
| `docs/PRD.md` | 产品需求文档 v2.2 — 功能需求、用户故事、业务规则 |
| `docs/TECH.md` | 技术架构文档 v1.1 — 分层架构、数据库、API、适配层、部署 |
| `docs/PLAN.md` | 实施计划 — 38 个任务，6 个里程碑 |
| `docs/ANYTHINGLLM_AI_INTEGRATION.md` | AnythingLLM 集成方案 v1.1（权威来源） |
| `docs/prompts/DESIGN-linear.app.md` | Linear Design 系统约束 |
| `server/cmd/main.go` | 后端入口，初始化配置、数据库、路由、调度器 |
| `server/internal/config/` | Viper 配置管理（config.go + config.yaml） |
| `server/internal/middleware/` | Gin 中间件（JWT 认证、RBAC 权限、CORS、请求日志） |
| `server/internal/router/` | 路由注册（router.go + portal.go + admin.go） |
| `server/internal/handler/` | Handler 层 — 参数校验、调用 Service、格式化响应 |
| `server/internal/service/` | Service 层 — 业务逻辑、事务管理、状态机校验 |
| `server/internal/repository/` | Repository 层 — 数据访问、GORM 查询 |
| `server/internal/model/` | GORM 数据模型（对应 TECH.md §4.2 表结构） |
| `server/internal/adapter/` | 外部服务适配层（RagClient / StorageClient） |
| `server/internal/dto/` | 数据传输对象（request/ + response/） |
| `server/pkg/` | 公共工具包（response / errcode / jwt / hash） |
| `server/migrations/` | 数据库迁移和演示数据（seed.sql） |
| `web/src/api/` | 前端 API 请求封装（Axios） |
| `web/src/views/portal/` | 门户端页面（智能问答、申告提交、进度查询） |
| `web/src/views/admin/` | 后台管理页面（看板、申告、知识库、用户、配置） |
| `web/src/views/auth/` | 认证页面（登录、修改密码） |
| `web/src/components/` | 通用组件（布局、分页、确认框、状态标签） |
| `web/src/stores/` | Pinia 状态管理（auth / chat / app） |
| `web/src/router/` | Vue Router 路由定义和守卫 |
| `web/src/styles/` | 全局样式（Linear Design 暗色主题 CSS 变量） |
| `docker-compose.yml` | Docker Compose 编排（6 个服务） |
| `.env` | 环境变量模板 |
| `Makefile` | 构建和开发命令 |

---

## 6. 开发边界

### 始终要做 (Always do)

- **遵循三层架构：** Handler（参数校验、响应格式）→ Service（业务逻辑、事务）→ Repository（数据访问）。不允许跨层调用。
- **写中文注释：** 每个文件需要文件头注释（说明模块存在的原因），每个关键函数需要函数注释（说明为什么这样实现）。见 §7 注释规范。
- **对齐文档：** 实现任何功能前先查看 `docs/PLAN.md` 对应任务、`docs/TECH.md` 对应章节。实现完成后确认 TECH.md 不需要同步更新。
- **统一响应格式：** 所有 API 响应使用 `pkg/response` 封装，格式为 `{"code": 0, "message": "success", "data": {...}}`，错误码定义见 `pkg/errcode`。
- **密码策略校验：** 所有涉及密码创建/修改的场景，必须调用 `pkg/hash.ValidatePassword` 校验正则 `^(?=.*[a-z])(?=.*[A-Z])(?=.*\d).{8,32}$`。
- **RBAC 权限校验：** 后台管理接口必须经过 JWT 中间件 + RBAC 中间件双重校验。
- **审计日志：** 用户管理、知识管理、申告管理的关键操作必须写入 audit_logs 表。
- **前端遵循 Linear Design：** 使用暗色主题 CSS 变量（`--bg-base: #08090a`、`--accent: #5e6ad2` 等），字体 Inter Variable。
- **AnythingLLM 通过适配层访问：** 后端只能通过 `RagClient` 接口访问 AnythingLLM，禁止直接 HTTP 调用。
- **使用中文 git commit message：** 格式为 `类型: 简短描述`，如 `feat: 实现用户登录接口`、`fix: 修复申告状态机校验`。

### 绝不要做 (Never do)

- **绝不自动执行 `git push`，所有推送操作必须人工确认。**
- **绝不绕过适配层直接调用外部服务：** 禁止在 Service/Handler 中直接调用 AnythingLLM HTTP API 或 MinIO API，必须通过 `RagClient` / `StorageClient` 接口。
- **绝不修改 AnythingLLM 配置文件：** AnythingLLM 的 LLM、Embedding、Vector DB 配置通过 docker-compose.yml 环境变量管理，不要修改 AnythingLLM 内部配置。
- **绝不让前端访问 AnythingLLM：** AnythingLLM API Key 只保存在后端配置中，不下发给浏览器。前端只通过 OpsMind 后端 API 交互。
- **绝不跳过状态机校验：** 申告状态转换必须在 `TicketService.UpdateStatus` 中严格校验前置状态，禁止直接 `UPDATE status`。
- **绝不在 Repository 层写业务逻辑：** Repository 只负责数据访问，业务规则（如审核人≠创建人、补充信息≤3 次）必须在 Service 层。
- **绝不硬编码配置值：** 数据库连接、AnythingLLM 地址、JWT 密钥等必须从配置文件或环境变量读取。
- **绝不使用 localhost 访问 Docker 内部服务：** 容器内访问 AnythingLLM 使用 `http://anythingllm:3001/api`，访问 PostgreSQL 使用 `postgres:5432`。
- **绝不省略错误处理：** 外部服务调用（RagClient、StorageClient）必须处理超时、不可达、返回错误三种情况。
- **绝不跳过降级逻辑：** AI 服务不可用时必须返回明确提示（code=20001），不能静默失败。
- **不要完全覆盖 .gitignore：** 如需添加规则，从文件末尾追加，不要替换已有内容。

---

## 7. 注释规范

### 语言和风格

- **所有注释使用中文。**
- 注释必须解释 **为什么这样做**，而不是重复代码逻辑。
- 文件头注释说明模块存在的原因和解决的问题。
- 函数注释说明设计决策、为什么选择这种实现方案。

### 文件头注释示例

```go
// Package service 实现申告业务逻辑。
//
// 申告状态机是本模块的核心设计，采用显式状态转换表而非隐式条件判断，
// 原因是状态转换规则会随业务变化频繁调整，显式表更容易维护和审计。
package service
```

### 函数注释示例

```go
// UpdateStatus 执行申告状态转换。
//
// 为什么用 switch-case 而不是状态转换矩阵：
// MVP 阶段状态数量有限（5 个），switch-case 更直观且易于调试。
// 后续如状态超过 10 个，可重构为状态转换表。
//
// request_info 操作会同步触发站内消息通知（而非异步），
// 原因是消息写入是轻量操作，同步执行可保证事务一致性。
func (s *TicketService) UpdateStatus(id int64, operatorID int64, req *UpdateTicketStatusRequest) error {
    // ...
}
```

### 反面示例（不要这样写）

```go
// 错误：机械重复代码逻辑
// GetUserByID 根据 ID 获取用户
func (r *UserRepo) GetUserByID(id int64) (*User, error) {

// 错误：没有解释为什么
// 密码使用 bcrypt 哈希
func HashPassword(password string) (string, error) {

// 正确：解释为什么选择 bcrypt
// HashPassword 使用 bcrypt 对密码进行单向哈希。
// 选择 bcrypt 而非 argon2 的原因：Go 标准库直接支持，MVP 阶段无需额外依赖。
// cost=10 在 4C8GB 环境下单次哈希约 100ms，满足登录场景性能要求。
func HashPassword(password string) (string, error) {
```

---

## 8. 资源

### 项目文档

| 文档 | 用途 |
| --- | --- |
| `docs/PRD.md` | 产品需求文档 — 功能需求、用户故事、业务规则、验收标准 |
| `docs/TECH.md` | 技术架构文档 — 分层架构、数据库 ER 图和表结构、API 端点清单、适配层接口、部署配置 |
| `docs/PLAN.md` | 实施计划 — 38 个任务按 6 个里程碑组织，每个任务列出文件、验证标准 |
| `docs/ANYTHINGLLM_AI_INTEGRATION.md` | AnythingLLM 集成方案（权威来源）— Docker 编排、API 接入、字段映射、降级规则 |
| `docs/prompts/DESIGN-linear.app.md` | Linear Design 系统约束 — 暗色主题色值、字体配置、组件样式 |

### 外部依赖文档

| 依赖 | 文档地址 |
| --- | --- |
| Gin | https://gin-gonic.com/docs/ |
| GORM | https://gorm.io/docs/ |
| pgvector-go | https://github.com/pgvector/pgvector-go |
| golang-jwt | https://github.com/golang-jwt/jwt |
| MinIO Go SDK | https://min.io/docs/minio/linux/developers/go/API.html |
| Vue 3 | https://vuejs.org/guide/ |
| Radix Vue | https://www.radix-vue.com/ |
| Pinia | https://pinia.vuejs.org/ |
| AnythingLLM API | 本地参考：`D:\Projects\Learning\anything-llm\server\endpoints\api\` |

### 环境变量模板

完整 `.env` 配置见 `docs/ANYTHINGLLM_AI_INTEGRATION.md §3.2`，核心变量：

| 变量 | 说明 | 默认值 |
| --- | --- | --- |
| `POSTGRES_PASSWORD` | PostgreSQL 密码 | opsmind_dev |
| `MINIO_ROOT_USER` | MinIO 管理员用户名 | minioadmin |
| `MINIO_ROOT_PASSWORD` | MinIO 管理员密码 | minioadmin |
| `ANYTHINGLLM_BASE_URL` | AnythingLLM 内部地址 | http://anythingllm:3001/api |
| `ANYTHINGLLM_API_KEY` | AnythingLLM API Key | 需初始化后填写 |
| `JWT_SECRET` | JWT 签名密钥 | 需手动设置 |
| `VLLM_BASE_URL` | vLLM 地址（配置在 AnythingLLM） | http://vllm:8000/v1 |
| `AI_CONFIDENCE_THRESHOLD` | 置信度阈值 | 0.6 |
| `AI_DEFAULT_TOP_K` | RAG 检索 Top K | 5 |

### 预设角色

| 角色 | 说明 | 典型权限 |
| --- | --- | --- |
| 系统管理员 | 系统全局管理 | 全部后台权限、角色权限管理、系统配置 |
| 运维人员 | 处理申告和回访 | 申告查看/处理、回访记录、知识候选提交 |
| 知识库管理员 | 维护和审核知识 | 知识 CRUD、知识审核、知识库配置 |
| 报障人 | 门户端用户 | 智能问答、申告提交、进度查询（仅门户端，无后台权限） |

### 错误码速查

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
