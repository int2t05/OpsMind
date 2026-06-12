# 🤖 OpsMind — 运维数字员工系统

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go" alt="Go">
  <img src="https://img.shields.io/badge/Vue-3.4+-4FC08D?logo=vuedotjs" alt="Vue">
  <img src="https://img.shields.io/badge/PostgreSQL-18-4169E1?logo=postgresql" alt="PostgreSQL">
  <img src="https://img.shields.io/badge/pgvector-hnsw-336791" alt="pgvector">
  <img src="https://img.shields.io/badge/license-MIT-green" alt="License">
</p>

**本地化大模型驱动的企业运维 AI 数字员工** — 基于 Go + Vue 3 分层架构，自建 Go RAG 引擎（BM25 + pgvector 混合检索），支持智能问答、申告全流程、知识库管理和 RBAC 权限控制，完全私有部署。

## 功能特性

| 模块 | 特性 |
|------|------|
| 🤖 **智能问答** | 自建 RAG 管道（查询改写→多路检索→混合检索→重排序→LLM 生成），SSE 流式输出 |
| 📚 **知识库** | 统一文章模型 + 审核发布 + 自动向量写入 pgvector + 多格式文档上传 |
| 🎫 **申告管理** | 完整状态机（待处理→处理中→需补充→已解决/已关闭），站内消息通知 |
| 👥 **用户权限** | JWT 认证 + RBAC 角色权限，菜单动态渲染，密码策略强制 |
| ⚙️ **LLM 配置** | llama.cpp / OpenAI-compatible 双支持，热替换无需重启 |
| 📊 **数据看板** | 实时统计卡片 + 趋势图，问答量/置信度一目了然 |
| 📝 **审计日志** | 敏感操作全量记录，支持按操作类型/操作人筛选 |
| 🐳 **一键部署** | Docker Compose 4 服务编排（+ 可选 llama.cpp） |

## 技术栈

| 层级 | 技术 | 说明 |
|------|------|------|
| 后端框架 | Go 1.22+ / Gin 1.9+ | REST API + SSE 流式 |
| ORM | GORM v1.25+ | PostgreSQL 数据访问 + AutoMigrate |
| 数据库 | PostgreSQL 18 + pgvector | 业务数据 + HNSW 向量索引（halfvec） |
| RAG 引擎 | 自建 Go 模块（`rag/`） | BM25 + 向量混合检索 + RRF 融合 + 重排序 |
| 中文分词 | gse（纯 Go） | BM25 分词，无 CGO 依赖 |
| LLM/Embedding | llama.cpp server 或 OpenAI-compatible API | 后台管理 UI 配置 |
| 对象存储 | MinIO | S3-compatible，文档 + 附件 |
| 认证 | JWT (golang-jwt v5) + bcrypt | access_token 2h + refresh_token 7d |
| 前端框架 | Vue 3.4+ / TypeScript | Composition API + script setup |
| UI 组件 | Naive UI + Radix Vue | Tree-shakable，暗色主题 |
| 状态管理 | Pinia 2.1+ | auth / chat / app 三个 store |
| 部署 | Docker Compose | 4 必须服务 + 1 可选 (llama.cpp) |

## 快速启动

### 前置条件

- Docker Desktop 4.x+（含 Docker Compose v2）
- 磁盘空间 ≥ 10 GB
- 内存 ≥ 8 GB

### 一键启动

```bash
git clone https://github.com/int2t05/OpsMind.git
cd OpsMind
cp .env.example .env
# 编辑 .env，至少设置 JWT_SECRET 和 LLM_BASE_URL
make up
```

等待约 1 分钟，访问：
- http://localhost:5173 — 前端
- http://localhost:8080 — 后端 API
- http://localhost:9001 — MinIO 控制台

### 配置 LLM（可选）

OpsMind 的基础功能（认证、用户管理、申告管理）不依赖 AI 模型。智能问答和知识库检索需要 LLM + Embedding 服务。

| 方案 | 成本 | 延迟 | 数据隐私 | 配置难度 |
|------|------|------|----------|----------|
| **本地 llama.cpp** | 免费 | 低 | ✅ 完全本地 | ⭐⭐⭐ |
| **OpenAI API** | 按量付费 | 中 | ❌ 上传云端 | ⭐ |
| **DeepSeek / 其他兼容 API** | 各异 | 各异 | 各异 | ⭐ |

配置方式：在 `.env` 中设置 `LLM_BASE_URL` / `LLM_API_KEY` / `LLM_MODEL` 等变量，或在后台管理「LLM 配置」页面管理。

### 默认账号

加载演示数据后可用：

| 账号 | 密码 | 角色 |
|------|------|------|
| `admin` | `Admin@123` | 系统管理员 |
| `operator1` | `Operator@123` | 运维人员 |
| `knowledge` | `Knowledge@123` | 知识库管理员 |
| `reporter1` | `Reporter@123` | 报障人 |

```bash
docker compose exec -T postgres psql -U opsmind -d opsmind < server/migrations/001_init.sql
```

## 本地开发

不使用 Docker 直接编译运行前后端，适合开发和调试场景。

### 前置条件

| 依赖 | 版本要求 | 说明 |
|------|----------|------|
| Go | 1.22+ | 后端编译和运行 |
| Node.js | 18+（推荐 20 LTS） | 前端开发服务器 |
| PostgreSQL | 16+ | 需安装 pgvector 扩展 |
| MinIO | 任意最新版 | 对象存储（文档/附件） |

> **提示：** PostgreSQL 和 MinIO 可以用 Docker 只启动这两个依赖（`make dev`），也可以直接安装到本机。

### 1. 克隆与配置

```bash
git clone https://github.com/int2t05/OpsMind.git
cd OpsMind
cp .env.example .env
```

编辑 `.env`，确保以下变量正确：

```bash
# === 数据库（如用 make dev 启动，默认值无需修改）===
DB_HOST=localhost
DB_PORT=5432
DB_USER=opsmind
DB_PASSWORD=opsmind_dev
DB_NAME=opsmind

# === MinIO（如用 make dev 启动，默认值无需修改）===
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY_ID=minioadmin
MINIO_SECRET_ACCESS_KEY=minioadmin

# === JWT（务必修改为随机字符串）===
JWT_SECRET=your-random-secret-here

# === LLM（可选，不影响基础功能）===
LLM_BASE_URL=http://localhost:8080/v1
LLM_API_KEY=not-needed
LLM_MODEL=qwen3-4b
```

### 2. 启动依赖服务

**方式 A：Docker 启动（推荐）**

```bash
make dev
```

这会在后台启动 PostgreSQL（含 pgvector）和 MinIO：

- PostgreSQL: `localhost:5432`，数据库 `opsmind`
- MinIO API: `localhost:9000`
- MinIO Web 控制台: `http://localhost:9001`

**方式 B：本机安装**

自行安装 PostgreSQL 16+（确保已启用 pgvector 扩展）和 MinIO，然后参照 `.env.example` 配置连接信息。

### 3. 启动后端

```bash
cd server

# 安装依赖
go mod tidy

# 启动（GORM AutoMigrate 会自动建表）
go run ./cmd/main.go
```

启动成功后会看到：

```
[GIN-debug] Listening and serving HTTP on :8080
```

首次启动时 GORM AutoMigrate 会自动创建所有表结构。

**加载演示数据（可选）：**

```bash
# 如 PostgreSQL 在 Docker 中
make seed

# 如 PostgreSQL 在本机
psql -U opsmind -d opsmind < server/migrations/001_init.sql
```

### 4. 启动前端

```bash
cd web

# 安装依赖（仅首次）
npm install

# 启动开发服务器（热重载，端口 5173）
npm run dev
```

Vite 开发服务器会自动代理 `/api` 请求到后端 `localhost:8080`。

### 5. 访问

| 地址 | 说明 |
|------|------|
| http://localhost:5173 | 前端（Vite 开发服务器） |
| http://localhost:8080 | 后端 API |
| http://localhost:9001 | MinIO 控制台（如用 Docker 启动） |

### 可选：本地 llama.cpp

如需使用本地 AI 模型（对话 + Embedding），在 `.env` 中设置 `LLM_BASE_URL` 指向你的 llama.cpp server 地址，或使用 Docker 启动完整 AI 环境：

```bash
# 先下载模型文件到 ./models/
make model-download

# 启动含 llama.cpp 的完整环境
make up-ai
```

## 项目结构

```
OpsMind/
├── server/                            # Go 后端
│   ├── cmd/main.go                    # 入口（DB→Repo→Service→Handler→Router→Scheduler）
│   ├── internal/
│   │   ├── config/                    # Viper 配置管理
│   │   ├── database/                  # GORM 连接 + AutoMigrate
│   │   ├── middleware/                # JWT / RBAC / CORS / Logger / RequestID
│   │   ├── router/                    # Gin 路由（public / portal / admin）
│   │   ├── handler/                   # Handler 层（10 模块 + common.go）
│   │   ├── service/                   # Service 层（12 服务 + LLM 配置管理）
│   │   ├── repository/                # Repository 层（9 Repo + pagination.go）
│   │   ├── model/                     # GORM 数据模型
│   │   ├── rag/                       # RAG 引擎（Pipeline / BM25 / Hybrid / Rerank / Processor）
│   │   ├── adapter/                   # 外部适配层（LLMClient / EmbeddingClient / VectorStore / StorageClient）
│   │   └── dto/                       # 请求/响应 DTO
│   ├── pkg/                           # 公共工具（errcode / jwt / hash / response）
│   ├── migrations/                    # DDL + 演示数据
│   └── tests/                         # 测试代码
│
├── web/                               # Vue 3 前端
│   ├── src/
│   │   ├── api/                       # Axios API 封装
│   │   ├── stores/                    # Pinia 状态管理
│   │   ├── router/                    # Vue Router + 路由守卫
│   │   ├── views/                     # 页面（auth / admin / portal）
│   │   ├── components/                # 通用组件
│   │   └── utils/                     # 工具函数
│   └── nginx.conf                     # Nginx 配置
│
├── test/                              # Playwright API 集成测试（116 个）
├── docs/                              # 项目文档
│   ├── PRD.md                         # 产品需求文档
│   ├── TECH.md                        # 技术架构文档
│   ├── API/                           # REST API 接口文档（9 组）
│   ├── diagrams/                      # 架构图（Mermaid）
│   └── prompts/                       # 设计系统约束
├── docker-compose.yml                 # Docker 4 服务编排
├── Makefile                           # 构建和开发命令
└── CLAUDE.md                          # 项目上下文指令
```

## 常用命令

```bash
# 后端
cd server
go build ./cmd/...                    # 编译
go run ./cmd/main.go                   # 运行
go test ./tests/... -v                 # 测试

# 前端
cd web
npm install && npm run dev             # 开发 (localhost:5173)
npm run build                          # 生产构建

# Docker
docker compose up -d --build           # 启动
docker compose --profile ai-local up -d --build  # 含 llama.cpp
docker compose down                    # 停止

# API 集成测试
cd test && npm install && npm run test:auth && npm run test
```

## 文档

| 文档 | 说明 |
|------|------|
| [PRD.md](docs/PRD.md) | 产品需求文档 |
| [TECH.md](docs/TECH.md) | 技术架构文档 |
| [API/](docs/API/README.md) | REST API 接口文档（9 组） |
| [diagrams/](docs/diagrams/) | 架构图和数据流图 |
| [CLAUDE.md](CLAUDE.md) | 项目上下文指令（AI 开发辅助） |

## 参与贡献

本项目采用 MIT 许可证，欢迎提交 Issue 和 Pull Request。
