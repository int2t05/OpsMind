# OpsMind 运维数字员工系统

OpsMind 是面向企业运维场景的 AI + 自动化运维平台 MVP。当前范围聚焦智能问答、人工申告闭环、知识库管理、运维账号权限、审计日志、系统配置、基础看板和 Docker Compose 单机部署。

## 项目结构

```text
OpsMind/
├─ server/                 # Go + Gin 后端服务
├─ web/                    # Vue 3 + Vite 前端，包含门户端和后台端
├─ deploy/                 # Docker Compose、Nginx、PostgreSQL、监控配置
├─ scripts/                # 开发启动、初始化、备份和演示数据脚本
├─ docs/                   # PRD、TECH、DB、API、FLOW、PLAN、TEST、DEPLOY
├─ .env.example            # 示例环境变量，不包含真实密钥
└─ README.md               # 项目结构、启动方式和文档入口
```

## 技术栈

| 层级 | 技术 |
|------|------|
| 前端 | Vue 3、Vite、TypeScript、Pinia、Vue Router、Element Plus |
| 后端 | Go 1.26.x、Gin、GORM、Casbin、JWT、Zap、Viper |
| 数据库 | PostgreSQL 18 + pgvector |
| 缓存与队列 | Redis、NATS JetStream |
| 对象存储 | MinIO，S3-compatible 适配层 |
| AI/RAG | AnythingLLM 完整 RAG 流程，vLLM OpenAI-compatible 模型服务 |
| 部署 | Docker Compose + Nginx |

## 环境要求

| 项 | 要求 |
|----|------|
| CPU | 4 核及以上 |
| 内存 | 8 GB 及以上 |
| 磁盘 | 100 GB 及以上 |
| Docker | Docker Compose v2 |
| Go | 1.26.x |
| Node.js | 22 LTS 或兼容版本 |
| 包管理器 | pnpm |

4C8GB 环境只保证业务系统和轻量依赖演示。vLLM 模型推理服务建议部署到独立 GPU 节点，业务服务通过 OpenAI-compatible 地址访问。

## 配置

```bash
cp .env.example .env
```

`.env` 至少包含以下配置项：

| 变量 | 说明 |
|------|------|
| `APP_ENV` | 运行环境 |
| `SERVER_PORT` | Go API 服务端口 |
| `DATABASE_DSN` | PostgreSQL 连接串 |
| `REDIS_ADDR` | Redis 地址 |
| `NATS_URL` | NATS 地址 |
| `MINIO_ENDPOINT` | MinIO 地址 |
| `MINIO_ACCESS_KEY` | MinIO Access Key |
| `MINIO_SECRET_KEY` | MinIO Secret Key |
| `MINIO_BUCKET` | 默认 bucket |
| `ANYTHINGLLM_BASE_URL` | AnythingLLM 服务地址 |
| `ANYTHINGLLM_API_KEY` | AnythingLLM API Key |
| `VLLM_BASE_URL` | vLLM OpenAI-compatible 地址 |
| `VLLM_API_KEY` | vLLM API Key |
| `JWT_SECRET` | JWT 签名密钥 |

禁止提交真实密钥、生产连接串和个人访问令牌。

## 本地启动

启动基础依赖：

```bash
docker compose -f deploy/docker-compose.yml up -d postgres redis nats minio
```

启动后端：

```bash
cd server
go run ./cmd/opsmind
```

启动前端：

```bash
cd web
pnpm install
pnpm dev
```

完整 Docker Compose 演示环境：

```bash
docker compose -f deploy/docker-compose.yml up -d
```

访问入口：

| 地址 | 用途 |
|------|------|
| `http://localhost` | 前端入口 |
| `http://localhost/api/v1/auth/login` | API 认证入口 |
| `http://localhost:9001` | MinIO 控制台 |

## 初始化数据

初始化数据由后端迁移逻辑或 `scripts/seed.sql` 写入，需包含：

| 数据 | 说明 |
|------|------|
| 默认管理员 | `admin`，拥有全部权限 |
| 默认角色 | 系统管理员、运维人员、知识库管理员 |
| 默认权限 | 与 `docs/API/common/overview.md` 权限码一致 |
| 默认知识库 | `ops-faq`，绑定 embedding 模型和向量维度 |
| 示例 FAQ | 账号冻结、密码重置、VPN 连接失败等 |

演示账号以实际 seed 脚本为准。首次登录后必须修改默认密码。

## 验证

```bash
docker compose -f deploy/docker-compose.yml ps
docker compose -f deploy/docker-compose.yml logs --tail=100 server
docker compose -f deploy/docker-compose.yml logs --tail=100 worker
```

核心验收以 `docs/TEST/里程碑测试计划.md` 的 M6 用例为准。

## 文档入口

| 文档 | 说明 |
|------|------|
| `docs/PRD.md` | 产品需求、业务范围、API 数据流 |
| `docs/TECH.md` | 技术架构、目录结构、依赖选型 |
| `docs/DB.md` | 数据库表、索引、约束、迁移顺序 |
| `docs/API/common/overview.md` | API 通用约定、状态流、权限码清单 |
| `docs/API/openapi.yaml` | OpenAPI 入口 |
| `docs/FLOW/` | 核心业务流程 |
| `docs/PLAN.md` | 项目实施计划 |
| `docs/TEST/` | 测试方案和里程碑测试计划 |
| `docs/DEPLOY.md` | 部署、备份、排障和验收命令 |
