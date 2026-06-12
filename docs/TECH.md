# OpsMind — 技术架构文档

| 项目 | 内容 |
| --- | --- |
| 日期 | 2026-06-12 |
| 关联文档 | [PRD](PRD.md) · [API 文档](API/README.md) · [架构图](diagrams/) |

## 1. 架构总览

OpsMind 采用**单体分层架构（Modular Monolith）**，按 Handler → Service → Repository 三层分离。RAG 引擎（`rag/`包）为自包含领域模块，不依赖 HTTP 层。

```
客户端 (Vue 3 SPA)
  │
  ├─ HTTP/REST + SSE ──▶ Go 后端 (Gin, 端口 8080)
  │                        │
  │                        ├─ Handler 层 — 参数校验、响应格式
  │                        ├─ Service 层 — 业务逻辑、事务编排
  │                        ├─ Repository 层 — GORM 数据访问
  │                        ├─ RAG 引擎 (rag/) — 检索管道、文档处理
  │                        └─ Adapter 层 — LLM/Embedding/pgvector/MinIO
  │                             │
  └─────────────────────────────┼──────────────────
                                ▼
              ┌─────────────────┼─────────────────┐
              ▼                 ▼                  ▼
      pgvector/pg18        MinIO           llama.cpp (可选)
      (业务+向量)         (对象存储)       (OpenAI-compat API)
```

## 2. 分层设计

### 2.1 Handler 层

- 职责：请求参数解析与校验、调用 Service、格式化统一响应
- 每个 Handler 对应一个 API 域（Auth / User / Role / Ticket / Knowledge / Chat / LLMConfig / Dashboard / Audit / Config）
- 共享工具函数：`parsePagination`、`parseID`、`getCurrentUserID`、`handleServiceError`

### 2.2 Service 层

- 职责：业务逻辑、状态机校验、事务编排、跨模块调用
- 消费者接口模式：每个 Service 定义它所需的依赖接口，隐式满足而非显式声明
- LLM 配置通过 `atomic.Value` 热替换

### 2.3 Repository 层

- 职责：GORM 数据访问，无业务逻辑
- 通用分页辅助函数 `Paginate[T]` 消除各 Repo 重复

### 2.4 RAG 引擎（`server/internal/rag/`）

自包含的检索增强生成模块，不依赖 HTTP / Handler / Service 层：

| 文件 | 职责 |
|------|------|
| `pipeline.go` | 管道编排器，串联全部步骤 |
| `query_rewrite.go` | LLM 查询改写（消除指代歧义） |
| `multi_route.go` | LLM 多路检索（生成子查询并行执行） |
| `hybrid.go` | RRF 融合（向量 + BM25） |
| `bm25.go` | Okapi BM25 算法（gse 中文分词） |
| `rerank.go` | LLM 重排序 |
| `chunker.go` | RecursiveCharacterTextSplitter 分块 |
| `embedder.go` | 批量 Embedding 生成 |
| `document_parser.go` | PDF/DOCX/MD/TXT 多格式解析 |
| `processor.go` | 异步文档处理 goroutine pool |
| `types.go` | 共享类型和接口定义 |

### 2.5 Adapter 层（`server/internal/adapter/`）

封装外部协议而非特定服务：

| 接口 | 实现 | 说明 |
|------|------|------|
| `LLMClient` | `OpenAIClient` | ChatCompletion + ChatCompletionStream |
| `EmbeddingClient` | `OpenAIEmbeddingClient` | `/v1/embeddings` |
| `VectorStore` | `PgvectorStore` | pgvector batch insert / cosine search / delete |
| `StorageClient` | `MinIOClient` | 对象上传 / presigned URL / 删除 |

## 3. 数据库设计

### 3.1 核心表

| 表 | 说明 |
|------|------|
| `users` | 用户（bcrypt 密码哈希） |
| `roles` / `user_roles` / `role_menus` | RBAC 角色权限 |
| `knowledge_bases` | 知识库，关联 `llm_configs.id` |
| `knowledge_articles` | 知识文章（手动+上传），状态机 + 处理状态 |
| `knowledge_chunks` | 文章分块 + halfvec 向量（HNSW 索引） |
| `chat_sessions` / `chat_messages` | 问答会话和消息 |
| `tickets` / `ticket_records` | 申告和操作记录 |
| `llm_configs` | LLM/Embedding 提供商配置（最多一个默认） |
| `audit_logs` | 操作审计日志 |
| `configs` | 系统配置键值对（如 `app_name`） |
| `messages` | 站内消息通知 |

### 3.2 pgvector 配置

- 类型：`halfvec(N)`（半精度 float16，节省 50% 存储）
- 索引：HNSW（`halfvec_cosine_ops`），查询 < 50ms（10000 分块）
- 算子：`<=>`（cosine distance）

分块参数：chunk_size=1000, overlap=200（RecursiveCharacterTextSplitter）

### 3.3 文章状态机

```
草稿(1) → 已提交审核(2) → 审核通过(3) → 已发布(4) → 已停用(0)
               ↓
          审核驳回(6)
```

文档处理状态：`pending → parsing → chunking → embedding → completed`（失败 → `failed`，可重试）

## 4. RAG 管道配置

```go
type RAGOptions struct {
    TopK         int   // 返回分块数，默认 5，范围 1-20
    QueryRewrite bool  // 查询改写，默认 true
    MultiRoute   bool  // 多路检索，默认 true
    Hybrid       bool  // BM25 混合检索，默认 true
    Rerank       bool  // 重排序，默认 true
    RouteCount   int   // 子查询数，默认 3
    RerankCount  int   // 进入重排序候选数，默认 10
}
```

降级矩阵：

| 步骤 | 失败行为 |
|------|----------|
| 查询改写 | 降级——使用原始 question |
| 多路检索 | 降级——使用单路检索 |
| 向量检索 | **阻塞**——核心路径，返回错误 |
| BM25 检索 | 降级——仅用向量结果 |
| RRF 融合 | 降级——使用单路结果 |
| 重排序 | 降级——使用 RRF 排序结果 |
| LLM 生成 | **阻塞**——核心路径，返回错误 |

## 5. 配置与环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `JWT_SECRET` | JWT 签名密钥 | 生产环境必须设置 |
| `LLM_BASE_URL` | LLM/Embedding API 地址 | `http://llama-cpp:8080/v1` |
| `LLM_API_KEY` | API 密钥（OpenAI 需要；llama.cpp 留空） | — |
| `LLM_MODEL` | LLM 模型名称 | `qwen3-4b` |
| `LLM_MAX_TOKENS` | 最大生成 Token 数 | 8192 |
| `EMBEDDING_MODEL` | Embedding 模型名称 | `bge-m3` |
| `EMBEDDING_DIMENSION` | 向量维度 | 1024 |
| `POSTGRES_PASSWORD` | PostgreSQL 密码 | `opsmind_dev` |
| `MINIO_ROOT_USER` / `MINIO_ROOT_PASSWORD` | MinIO 管理员凭据 | `minioadmin` |

## 6. 模块接口

### 6.1 LLMClient

```go
type LLMClient interface {
    ChatCompletion(ctx, ChatRequest) (*ChatResponse, error)
    ChatCompletionStream(ctx, ChatRequest) (<-chan StreamChunk, error)
}
```

### 6.2 EmbeddingClient

```go
type EmbeddingClient interface {
    CreateEmbeddings(ctx, EmbeddingRequest) (*EmbeddingResponse, error)
}
```

### 6.3 VectorStore

```go
type VectorStore interface {
    BatchInsert(ctx, chunks []VectorChunk) error
    CosineSearch(ctx, kbID int64, embedding []float32, topK int) ([]SearchResult, error)
    DeleteByArticle(ctx, articleID int64) error
    DeleteByKB(ctx, kbID int64) error
    CountByKB(ctx, kbID int64) (int64, error)
    GetChunksByArticle(ctx, articleID int64) ([]ChunkContent, error)
}
```

## 7. 错误码

| code | HTTP | 说明 |
|------|------|------|
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

## 8. 项目结构

```
server/
├── cmd/main.go                     # 入口：DI → 路由 → 调度器 → 优雅关闭
├── internal/
│   ├── config/                     # Viper 配置加载
│   ├── database/                   # GORM 连接 + AutoMigrate
│   ├── middleware/                  # JWT / RBAC / CORS / Logger / RequestID
│   ├── router/                     # 路由注册（public/portal/admin 三组）
│   ├── handler/                    # HTTP Handler 层（10 个模块 + common.go）
│   ├── service/                    # 业务逻辑层（12 个服务 + LLM 配置管理）
│   ├── repository/                 # 数据访问层（9 个 Repo + pagination.go）
│   ├── model/                      # GORM 数据模型（12 个文件）
│   ├── rag/                        # RAG 引擎（11 个文件）
│   ├── adapter/                    # 外部适配层（LLM/Embedding/pgvector/MinIO）
│   └── dto/                        # 请求/响应 DTO
├── pkg/                            # 公共工具（errcode/jwt/hash/response）
├── migrations/                     # 数据库迁移脚本 + 演示数据
└── tests/                          # 测试代码（config/database/model/service/handler/adapter/rag）

web/
├── src/
│   ├── api/                        # Axios API 封装（15 个文件）
│   ├── stores/                     # Pinia 状态管理（auth/chat/app）
│   ├── router/                     # Vue Router + 路由守卫
│   ├── views/                      # 页面（auth/portal/admin）
│   ├── components/                 # 通用组件（布局/分页/状态标签）
│   ├── utils/                      # 工具函数（request/auth）
│   └── styles/                     # Linear Design 暗色主题
└── nginx.conf                      # Nginx 反向代理配置
```
