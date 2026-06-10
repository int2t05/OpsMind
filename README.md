# OpsMind — 运维数字员工系统

面向企业运维场景的 AI 数字员工系统，基于 Go + Vue 3 单体分层架构，集成 AnythingLLM RAG 实现智能问答、申告管理、知识库管理和 RBAC 权限控制。

## 当前进度

| 里程碑 | 状态 | 内容 |
|--------|------|------|
| M1 数据库与后端基础能力 | ✅ 完成 | Go 项目骨架、GORM 模型(16 表)、配置管理、中间件、路由注册 |
| M2 账号权限与后台框架 | ✅ 完成 | JWT 认证、RBAC 权限、用户 CRUD、角色管理、Vue 后台布局 |
| M3 知识库管理与 AI 服务 | ✅ 完成 | 知识库 CRUD、审核发布流程、AnythingLLM 适配器、RagClient |
| M4 智能问答与申告处理 | ✅ 完成 | RAG 集成、问答 API、申告状态机、站内消息、StorageClient |
| M5 数据看板与日志审计 | ✅ 完成 | 看板统计、审计日志、系统配置、模型/Embedding 配置页面 |
| M6 联调测试与文档完善 | ✅ 完成 | 集成测试(17 用例)、演示数据、Docker 6 服务编排、README |

**全部 38 个任务已完成。**

---

## 技术栈

| 层级 | 技术 | 版本 |
|------|------|------|
| 后端框架 | Go + Gin | 1.22+ / 1.9+ |
| ORM | GORM | v1.25+ |
| 数据库 | PostgreSQL + pgvector | 18 / 0.7+ |
| 认证 | JWT (golang-jwt) | v5 |
| 前端框架 | Vue 3 + TypeScript | 3.4+ |
| UI 组件 | Radix Vue | 1.9+ |
| 状态管理 | Pinia | 2.1+ |
| 路由 | Vue Router | 4.3+ |

---

## 快速启动

### 方式一：Docker Compose 一键启动（推荐）

```bash
# 1. 克隆项目
git clone <repo-url> OpsMind
cd OpsMind

# 2. 配置环境变量
cp .env.example .env
# 编辑 .env，设置 OPSMIND_JWT_SECRET 等必要变量

# 3. 一键启动全部服务（不含 vLLM）
docker compose up -d --build

# 4. 加载演示数据
docker compose exec -T postgres psql -U opsmind -d opsmind < server/migrations/seed.sql

# 5. 访问
# 前端: http://localhost:5173
# 后端 API: http://localhost:8080
# MinIO 控制台: http://localhost:9001

# 6. （可选）含 vLLM 的完整 AI 环境
docker compose --profile ai-local up -d --build
```

### 方式二：本地开发（Go + Vue + Docker 依赖服务）

#### 前置条件

- **Go** 1.22+
- **Node.js** 18+
- **Docker Desktop**（提供 PostgreSQL + MinIO）
- **Git Bash** 或 WSL（Windows 推荐）

#### 1. 启动依赖服务

```bash
make dev
# 或手动: docker compose up -d postgres minio
```

#### 2. 启动 Go 后端

```bash
cd server
go mod tidy
go run ./cmd/main.go
```

#### 3. 启动 Vue 前端

```bash
cd web
npm install
npm run dev
```

访问 `http://localhost:5173`

---

## 默认账号

加载 `server/migrations/seed.sql` 后，可使用以下账号登录：

| 账号 | 密码 | 角色 | 说明 |
|------|------|------|------|
| `admin` | `Admin@123` | 系统管理员 | 全部后台权限 |
| `operator1` | `Operator@123` | 运维人员 | 申告处理、知识候选 |
| `operator2` | `Operator@123` | 运维人员 | 同上 |
| `knowledge` | `Knowledge@123` | 知识库管理员 | 知识 CRUD/审核 |
| `reporter1` | `Reporter@123` | 报障人 | 门户端问答/申告 |
| `reporter2` | `Reporter@123` | 报障人 | 同上 |

> 首次登录会强制跳转修改密码页面。

## AnythingLLM API Key 初始化

AnythingLLM 的 API Key 需要在容器启动后手动创建：

```bash
# 1. 临时暴露 AnythingLLM 管理端口（编辑 docker-compose.yml，取消 anythingllm ports 注释）
# 2. 重启
docker compose up -d anythingllm

# 3. 浏览器访问 http://localhost:3001
# 4. 完成初始化向导后，进入 General Settings → API Keys，创建 API Key
# 5. 将 API Key 写入 .env 文件
ANYTHINGLLM_API_KEY=<刚创建的 API Key>

# 6. 重新构建并启动
docker compose up -d --build
```

详细步骤参见 [ANYTHINGLLM_AI_INTEGRATION.md](docs/ANYTHINGLLM_AI_INTEGRATION.md) §3.3。

---

## 项目结构

```
OpsMind/
├── docs/                              # 项目文档
│   ├── PRD.md                          # 产品需求文档 v2.2
│   ├── TECH.md                         # 技术架构文档 v1.2
│   ├── PLAN.md                         # 实施计划（38 任务，6 里程碑）
│   └── ANYTHINGLLM_AI_INTEGRATION.md    # AnythingLLM 集成方案 v1.1
│
├── server/                            # Go 后端
│   ├── cmd/main.go                    # 入口（DB→Repo→Service→Handler→Router→Scheduler）
│   ├── Dockerfile                     # 多阶段构建（Alpine 运行时）
│   ├── internal/
│   │   ├── config/                    # Viper 配置管理
│   │   ├── database/                  # GORM 连接 + AutoMigrate
│   │   ├── model/                     # 16 张表 GORM 模型
│   │   ├── middleware/                # JWT 认证 / RBAC / CORS / Logger / RequestID
│   │   ├── router/                    # Gin 路由（public / portal / admin 三组）
│   │   ├── handler/                   # Handler 层（全部 10 个模块）
│   │   ├── service/                   # Service 层（含后台调度器）
│   │   ├── repository/                # Repository 层
│   │   ├── adapter/                   # 外部适配层（RagClient / StorageClient）
│   │   └── dto/                       # 请求/响应 DTO
│   ├── pkg/                           # 公共工具（response / errcode / jwt / hash）
│   ├── migrations/
│   │   └── seed.sql                   # 演示数据（预设角色/用户/知识/申告）
│   └── tests/                         # 测试代码（含 17 个集成测试）
│
├── web/                               # Vue 3 前端
│   ├── Dockerfile                     # 多阶段构建（nginx 运行时）
│   ├── nginx.conf                     # nginx 反向代理配置
│   └── src/
│       ├── api/                       # Axios API 封装（auth/user/ticket/chat/knowledge...）
│       ├── stores/                    # Pinia 状态（auth/chat/app）
│       ├── router/                    # Vue Router + 路由守卫
│       ├── views/                     # 页面（auth/admin/portal）
│       ├── components/                # 通用组件（AdminLayout/PortalLayout/Pagination...）
│       ├── utils/                     # 工具函数（request/auth）
│       └── styles/                    # Linear Design 暗色主题
│
├── docker-compose.yml                 # Docker 6 服务编排
├── .env.example                       # 环境变量模板
├── Makefile                           # 构建和开发命令
└── README.md                          # 本文件
```

---

## 常用命令

### Makefile（推荐）

```bash
make dev                  # 本地开发（启动依赖服务）
make build                # 构建全部 Docker 镜像
make up                   # 一键启动全部服务
make up-ai                # 启动含 vLLM 的完整环境
make down                 # 停止全部服务
make test                 # 运行非集成测试
make test-integration     # 运行全部集成测试
make seed                 # 加载演示数据
make clean                # 清理构建产物和数据
```

### 后端

```bash
cd server
go build ./cmd/...            # 编译
go run ./cmd/main.go           # 运行
go test ./tests/... -v         # 非集成测试
go test ./tests/... -v -tags=integration  # 集成测试（需 PostgreSQL）
```

### 前端

```bash
cd web
npm install                    # 安装依赖
npm run dev                    # 开发服务器 (localhost:5173)
npm run build                  # 生产构建
npm run type-check             # TypeScript 类型检查
```

### Docker

```bash
docker compose up -d --build   # 构建并启动
docker compose ps              # 查看服务状态
docker compose logs -f         # 查看日志
docker compose down            # 停止
docker compose down -v         # 停止并清除数据
```

---

## 文档索引

| 文档 | 说明 |
|------|------|
| [PRD.md](docs/PRD.md) | 产品需求文档 v2.2 |
| [TECH.md](docs/TECH.md) | 技术架构文档 v1.2 |
| [PLAN.md](docs/PLAN.md) | 实施计划（38 任务，6 里程碑） |
| [ANYTHINGLLM_AI_INTEGRATION.md](docs/ANYTHINGLLM_AI_INTEGRATION.md) | AnythingLLM 集成方案 v1.1 |
| [CLAUDE.md](CLAUDE.md) | 项目上下文指令（AI 开发辅助） |

---

## 预设角色与权限

| 角色 | 典型权限 |
|------|---------|
| 系统管理员 | ticket:read/write/assign, knowledge:read/write/review, system:config, user:manage, audit:read |
| 运维人员 | ticket:read/write, knowledge:read/write |
| 知识库管理员 | knowledge:read/write/review |
| 报障人 | 无后台权限，仅门户端 |

## 错误码速查

| 错误码 | HTTP 状态 | 说明 |
|--------|----------|------|
| 0 | 200 | 成功 |
| 10001 | 401 | 未登录或令牌过期 |
| 10002 | 403 | 无权限 |
| 10003 | 400 | 参数校验失败 |
| 10004 | 404 | 资源不存在 |
| 10005 | 409 | 资源冲突（如用户名重复） |
| 10006 | 409 | 用户已被冻结 |
| 10007 | 409 | 用户已处于正常状态 |
| 20001 | 500 | AI 服务不可用 |
| 20002 | 500 | RAG 服务不可用 |
| 20003 | 500 | 存储服务不可用 |
| 99999 | 500 | 未知错误 |
