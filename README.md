# OpsMind — 运维数字员工系统

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go" alt="Go">
  <img src="https://img.shields.io/badge/Vue-3.4+-4FC08D?logo=vuedotjs" alt="Vue">
  <img src="https://img.shields.io/badge/PostgreSQL-18-4169E1?logo=postgresql" alt="PostgreSQL">
  <img src="https://img.shields.io/badge/license-MIT-green" alt="License">
  <img src="https://img.shields.io/badge/PRs-welcome-brightgreen" alt="PRs Welcome">
</p>

面向企业运维场景的 AI 数字员工系统。基于 **Go + Vue 3** 单体分层架构，集成 **AnythingLLM RAG** 实现智能问答、申告管理、知识库管理和 RBAC 权限控制，支持本地化大模型私有部署。

## 功能特性

| 模块 | 特性 |
|------|------|
| 🤖 **智能问答** | RAG 知识增强问答，置信度判断，低置信度自动转人工申告 |
| 🎫 **申告管理** | 完整状态机（待处理→处理中→需补充→已解决/已关闭），7 天超时自动关闭 |
| 📚 **知识库** | CRUD + 审核发布工作流，AnythingLLM 自动同步，pgvector 向量追溯 |
| 👥 **用户权限** | JWT 认证 + RBAC 角色权限，菜单动态渲染，密码策略强制 |
| 📊 **数据看板** | 实时统计卡片 + 趋势图，问答量/解决率/命中率一目了然 |
| 📝 **审计日志** | 敏感操作全量记录，支持按操作类型/操作人/时间筛选 |
| 🎨 **双主题** | Naive UI 组件库，暗色/浅色双主题，Linear Design 设计系统 |
| 🐳 **一键部署** | Docker Compose 7 服务编排，AnythingLLM API Key 全自动初始化 |

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

> **一键部署：** `make up` 即可启动全部服务，**AnythingLLM API Key 全自动初始化**，无需手动操作浏览器。

### 前置条件

- **Docker Desktop** 4.x+（含 Docker Compose v2）
- **Git Bash**（Windows）/ 任意终端（Linux/macOS）
- **磁盘空间** ≥ 20 GB（含镜像和模型）
- **内存** ≥ 16 GB（若启用 vLLM）
- **GPU**（可选）：NVIDIA GPU + nvidia-container-toolkit（vLLM 推理需要）

### 服务端口

| 端口 | 服务 | 说明 |
|------|------|------|
| `5173` | OpsMind Web 前端 | 用户入口（Nginx 反向代理到 :80） |
| `8080` | OpsMind Go 后端 | REST API（Gin） |
| `3001` | AnythingLLM | RAG 管理页面（初始化/排障用，日常可关闭） |
| `5432` | PostgreSQL | 业务数据库 + pgvector 向量库 |
| `9000` | MinIO S3 API | 对象存储（申告附件、知识文档原件） |
| `9001` | MinIO Console | MinIO Web 管理后台 |
| `8000` | vLLM（可选） | OpenAI 兼容 API（仅 `--profile ai-local` 时启动） |

> **注意：** vLLM 端口不映射到宿主机，仅在 Docker 内网通过 `http://vllm:8000/v1` 访问。

---

### 第一步：一键启动

```bash
# 克隆项目
git clone https://github.com/int2t05/OpsMind.git
cd OpsMind

# 配置环境变量
cp .env.example .env
# 编辑 .env，至少设置：
#   OPSMIND_JWT_SECRET=<随机字符串 32 位以上>
#   LLM_BASE_URL=<你的 LLM 服务地址>
#   LLM_API_KEY=<你的 LLM API Key>
# 其余配置使用默认值即可

# 一键启动（含 AnythingLLM API Key 自动初始化）
make up
```

`opsmind-setup` 服务会自动完成 AnythingLLM 初始化——**直写 SQLite 生成 API Key**，写入共享卷供后端读取。无需打开浏览器。

等待约 1-2 分钟，验证服务状态：

```bash
docker compose ps
# 期望输出：全部服务 Up，opsmind-postgres (healthy)、opsmind-anythingllm (healthy)
```

访问验证：

| 地址 | 说明 |
|------|------|
| http://localhost:5173 | 前端页面 |
| http://localhost:8080 | 后端 API |
| http://localhost:9001 | MinIO 管理控制台（minioadmin / minioadmin） |

> **手动兜底：** 如果自动初始化意外失败，运行 `make setup` 通过浏览器引导完成。

---

### 第二步（可选）：配置 LLM 启用 AI 问答

> OpsMind 的基础功能（认证、用户管理、申告管理）不依赖 AI 模型。只有**智能问答**和**知识库 RAG 检索**需要 LLM + Embedding 服务。

AnythingLLM 通过 `generic-openai` 提供商支持**所有 OpenAI 兼容 API**。选择下面一种方案，在 `.env` 中填写对应的 `LLM_*` 配置即可。

#### 方案对比

| 方案 | 成本 | 延迟 | 数据隐私 | 需要 GPU | 配置难度 |
|------|------|------|----------|----------|----------|
| **A: 本地 vLLM** | 免费 | 低 | ✅ 完全本地 | ✅ 需要 | ⭐⭐⭐ |
| **B: OpenAI API** | 按量付费 | 中 | ❌ 上传云端 | 不需要 | ⭐ |
| **C: 本地 Ollama** | 免费 | 低 | ✅ 完全本地 | 不需要 | ⭐⭐ |
| **D: 其他兼容 API** | 各异 | 各异 | 各异 | 不需要 | ⭐ |

---

#### 方案 A：本地 vLLM（推荐，数据不出域）

```bash
# 1. 下载模型（合计约 10 GB）
pip install modelscope
# Qwen3-4B：约 8 GB，如需更小模型可换 Qwen/Qwen2-1.5B-Instruct（约 3 GB）
modelscope download --model Qwen/Qwen3-4B-Instruct-2507 --local_dir ./models/qwen3-4b
modelscope download --model BAAI/bge-m3 --local_dir ./models/bge-m3
# 或使用 Makefile: make model-download

# 2. 编辑 .env，设置模型名称：
# LLM_BASE_URL=http://vllm:8000/v1
# LLM_MODEL=qwen3-4b

# 3. 取消 docker-compose.yml 中 vllm 服务的 command 和 deploy 注释
# 4. 启动
docker compose --profile ai-local up -d --build
```

> 如有 NVIDIA GPU，取消 `deploy.resources.reservations.devices` 注释启用 GPU 加速。

#### 方案 B：OpenAI 官方 API（最简单）

```bash
# 编辑 .env，改用 OpenAI 配置：
# LLM_BASE_URL=https://api.openai.com/v1
# LLM_MODEL=gpt-4o-mini
# LLM_API_KEY=sk-your-openai-api-key
#
# Embedding 也用 OpenAI:
# EMBEDDING_BASE_URL=https://api.openai.com/v1
# EMBEDDING_MODEL=text-embedding-3-small
# EMBEDDING_API_KEY=sk-your-openai-api-key

# 启动（无需 vLLM profile）
docker compose up -d --build
```

#### 方案 C：本地 Ollama（轻量级，无需 GPU，无需下载原始权重）

```bash
# 1. 安装 Ollama 并拉取模型
ollama pull qwen2.5:7b
ollama pull nomic-embed-text   # Ollama 自带的 embedding 模型

# 2. 编辑 .env：
# LLM_BASE_URL=http://host.docker.internal:11434/v1
# LLM_MODEL=qwen2.5:7b
# LLM_API_KEY=dummy-key
# EMBEDDING_BASE_URL=http://host.docker.internal:11434/v1
# EMBEDDING_MODEL=nomic-embed-text
# EMBEDDING_API_KEY=dummy-key

# 3. 启动（无需 vLLM profile）
docker compose up -d --build
```

#### 方案 D：DeepSeek / Moonshot 等国内 API

```bash
# 编辑 .env 即可，无需下载模型：
# LLM_BASE_URL=https://api.deepseek.com/v1
# LLM_MODEL=deepseek-chat
# LLM_API_KEY=sk-your-deepseek-api-key
```

> 更多 OpenAI 兼容服务参见 [ANYTHINGLLM_AI_INTEGRATION.md](docs/ANYTHINGLLM_AI_INTEGRATION.md)。

---

### 第三步：加载演示数据

```bash
# 加载预设角色、用户、知识库、申告等
docker compose exec -T postgres psql -U opsmind -d opsmind < server/migrations/seed.sql
```

---

### ✅ 验证完成

```bash
# 登录测试
curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"Admin@123"}' | grep -o '"code":[0-9]*'
# 期望输出: "code":0

# 打开浏览器访问 http://localhost:5173
# 使用 admin / Admin@123 登录
```

> **验证 AI 问答**（需先完成第二步 LLM 配置和第三步 API Key，否则跳过此步骤）：
> ```bash
> # 获取 token 后测试智能问答
> TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
>   -H "Content-Type: application/json" \
>   -d '{"username":"admin","password":"Admin@123"}' | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
> 
> curl -s -X POST http://localhost:8080/api/v1/portal/chat-sessions \
>   -H "Authorization: Bearer $TOKEN" \
>   -H "Content-Type: application/json" \
>   -d '{"question":"如何重置VPN密码？","kb_id":1}'
> ```

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
├── docker-compose.yml                 # Docker 7 服务编排
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
