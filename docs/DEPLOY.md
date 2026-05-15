# 运维数字员工系统部署文档

| 项目 | 内容 |
| --- | --- |
| 文档版本 | v1.0 |
| 生成日期 | 2026-05-15 |
| 部署目标 | OpsMind MVP 单机 Docker Compose 部署 |
| 适用环境 | 本地开发环境、课程演示环境、企业内网测试环境 |

## 1. 部署范围

MVP 部署包含以下组件：

| 组件 | 说明 |
|------|------|
| `web` | Vue 3 前端静态资源和管理端/门户端入口 |
| `server` | Go API 服务 |
| `worker` | 知识同步 Worker |
| `postgres` | PostgreSQL 18 + pgvector |
| `redis` | 缓存、限流、短期状态 |
| `nats` | NATS JetStream 异步事件 |
| `minio` | S3-compatible 对象存储 |
| `anythingllm` | RAG 服务，可独立部署后通过配置接入 |
| `vllm` | OpenAI-compatible 模型服务，可独立 GPU 节点部署后通过配置接入 |
| `nginx` | 前端静态资源和 API 反向代理 |

## 2. 环境要求

| 项 | 最低要求 |
|----|----------|
| CPU | 4 核 |
| 内存 | 8 GB |
| 磁盘 | 100 GB |
| 操作系统 | Linux 或 Windows + Docker Desktop |
| Docker | 支持 Docker Compose v2 |
| 浏览器 | Chrome 或 Edge 最新稳定版 |

说明：4C8GB 环境只保证业务系统和轻量依赖演示。vLLM 模型推理服务建议部署到独立 GPU 节点，业务服务通过 OpenAI-compatible 地址访问。

## 3. 环境变量

根目录 `.env` 基于 `.env.example` 创建。禁止把真实密钥提交到 Git。

| 变量 | 说明 |
|------|------|
| `APP_ENV` | 运行环境，开发环境为 `dev` |
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

## 4. 启动步骤

```bash
cp .env.example .env
docker compose -f deploy/docker-compose.yml up -d postgres redis nats minio
docker compose -f deploy/docker-compose.yml up -d server worker web nginx
```

启动后访问：

| 地址 | 用途 |
|------|------|
| `http://localhost` | 前端入口 |
| `http://localhost/api/v1/auth/login` | API 认证入口 |
| `http://localhost:9001` | MinIO 控制台 |

## 5. 初始化数据

初始化数据由 `scripts/seed.sql` 或后端迁移初始化逻辑写入。

| 数据 | 说明 |
|------|------|
| 默认管理员 | 用于首次登录后台 |
| 默认角色 | 系统管理员、运维人员、知识库管理员 |
| 默认权限 | 与 `docs/API/common/overview.md` 权限码清单一致 |
| 默认知识库 | `ops-faq`，配置 embedding 模型和向量维度 |
| 示例 FAQ | 账号冻结、密码重置、VPN 连接失败等 |

## 6. 健康检查

| 检查项 | 验证方式 |
|--------|----------|
| 后端服务 | 请求 `/api/v1/auth/profile` 未登录应返回 `401 unauthorized` |
| 数据库 | 后端启动日志无迁移错误 |
| Redis | 匿名问答限流可写入计数 |
| NATS | 发布知识后可生成同步事件 |
| MinIO | 门户附件上传返回 `file_id` |
| AnythingLLM | 配置连接测试返回成功 |
| vLLM | 配置连接测试返回成功 |

## 7. 备份与恢复

| 对象 | 备份方式 | 恢复方式 |
|------|----------|----------|
| PostgreSQL | `pg_dump` 或卷快照 | `psql` 导入或恢复卷 |
| MinIO | 备份 bucket 数据目录 | 恢复对象目录和 bucket 配置 |
| 配置 | 备份 `.env` 和 `server/configs/` | 恢复后重启服务 |
| 文档 | Git 管理 | 从仓库恢复 |

## 8. 常见故障处理

| 故障 | 排查步骤 |
|------|----------|
| 前端无法访问 API | 检查 Nginx 代理、后端端口、`/api` 路由 |
| 登录失败 | 检查初始化管理员、JWT 配置、账号状态 |
| 问答失败 | 检查 AnythingLLM、vLLM、知识库编码和后端日志 |
| 附件上传失败 | 检查 MinIO bucket、访问密钥、文件大小和类型 |
| 知识同步失败 | 检查 NATS 事件、Worker 日志、AnythingLLM 响应和 pgvector 维度 |
| 端口冲突 | 修改 `.env` 和 `deploy/docker-compose.yml` 中的端口映射 |

## 9. 验收命令

```bash
docker compose -f deploy/docker-compose.yml ps
docker compose -f deploy/docker-compose.yml logs --tail=100 server
docker compose -f deploy/docker-compose.yml logs --tail=100 worker
```

最终验收以 `docs/TEST/里程碑测试计划.md` 中 M6 测试用例为准。
