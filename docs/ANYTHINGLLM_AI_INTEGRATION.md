# AnythingLLM AI 组件集成文档

| 项目 | 内容 |
| --- | --- |
| 文档版本 | v1.2 |
| 日期 | 2026-06-11 |
| 适用范围 | OpsMind 智能问答、知识库同步、AI 降级处理 |
| 关联文档 | `docs/PRD.md`、`docs/TECH.md` |

---

## 1. 集成结论

AnythingLLM 应作为 OpsMind 的一个内部 Docker 组件集成到项目中，而不是让使用者分别打开、分别运行两个系统。

统一口径：

- 用户只访问 OpsMind 前端。
- OpsMind 后端只暴露自己的业务 API，例如 `POST /api/v1/portal/chat-sessions`。
- AnythingLLM 由 OpsMind 的 `docker-compose.yml` 统一拉起，作为内部 RAG 服务运行。
- OpsMind 后端通过 Docker 内部服务名访问 AnythingLLM：`http://anythingllm:3001/api`。
- AnythingLLM 管理页面不作为日常入口，只在初始化、排障或管理员配置 API Key 时临时访问。
- AnythingLLM API Key 只保存在 OpsMind 后端配置中，不能下发给浏览器。

推荐组件边界：

| 能力 | 组件 | 说明 |
| --- | --- | --- |
| 门户问答入口 | OpsMind Web | 展示提问、答案、来源、反馈、转申告入口 |
| 业务 API | OpsMind Server | 保存会话、判断置信度、处理降级、记录审计 |
| RAG 服务 | AnythingLLM | 负责知识导入、切分、向量检索、RAG 编排、同步问答 |
| 模型推理 | vLLM | 通过 AnythingLLM 的 `generic-openai` 提供商接入 |
| 业务数据库 | PostgreSQL + pgvector | 保存 OpsMind 用户、申告、知识、审计和系统侧向量追溯 |
| 对象存储 | MinIO | 保存申告附件和知识文档原件 |

推荐调用链：

```text
用户
  -> OpsMind Web
  -> OpsMind Server: POST /api/v1/portal/chat-sessions
  -> RagClient
  -> AnythingLLM: POST http://anythingllm:3001/api/v1/workspace/{slug}/chat
  -> vLLM: OpenAI-compatible chat completion
  -> OpsMind Server 保存会话、来源、置信度
  -> OpsMind Web 展示结果或转人工入口
```

---

## 2. AnythingLLM 依据文件

本设计基于本地 AnythingLLM 项目 `D:\Projects\Learning\anything-llm` 的以下文件：

| 文件 | 用途 |
| --- | --- |
| `server/endpoints/api/index.js` | API 路由总入口，确认 `/api/v1/...` 前缀 |
| `server/endpoints/api/workspace/index.js` | workspace 创建、同步文档、同步问答、向量检索接口 |
| `server/endpoints/api/document/index.js` | 文件、链接、raw text 文档导入接口 |
| `server/endpoints/api/openai/index.js` | OpenAI-compatible API，可作为备用对接方式 |
| `server/utils/middleware/validApiKey.js` | API 鉴权方式：`Authorization: Bearer <apiKey>` |
| `server/utils/chats/apiChatHandler.js` | 同步问答返回结构，包含 `textResponse`、`sources`、`metrics` |
| `server/.env.example` | LLM、Embedding、Vector DB 环境变量 |
| `docker/docker-compose.yml` | AnythingLLM Docker 服务结构参考 |
| `docker/HOW_TO_USE_DOCKER.md` | Docker 运行方式参考 |

---

## 3. Docker 集成方案

OpsMind 项目根目录应提供统一的 `docker-compose.yml`。用户执行一次：

```powershell
cd D:\Projects\Personal\OpsMind
docker compose up -d --build
```

即可启动：

- `opsmind-server`
- `opsmind-web`
- `anythingllm`
- `postgres`
- `minio`
- 可选 `vllm`

### 3.1 推荐 Compose 结构

建议在 OpsMind 根目录创建：

```text
OpsMind/
├── docker-compose.yml
├── .env
├── server/
├── web/
└── docs/
```

`docker-compose.yml` 推荐结构：

```yaml
services:
  opsmind-server:
    build:
      context: ./server
    container_name: opsmind-server
    ports:
      - "8080:8080"
    env_file:
      - .env
    environment:
      OPSMIND_PORT: 8080
      ANYTHINGLLM_BASE_URL: http://anythingllm:3001/api
      ANYTHINGLLM_API_KEY: ${ANYTHINGLLM_API_KEY}
      AI_DEFAULT_TOP_K: 5
      AI_CONFIDENCE_THRESHOLD: 0.6
      DB_HOST: postgres
      MINIO_ENDPOINT: minio:9000
    depends_on:
      - postgres
      - minio
      - anythingllm
    networks:
      - opsmind

  opsmind-web:
    build:
      context: ./web
    container_name: opsmind-web
    ports:
      - "5173:80"
    depends_on:
      - opsmind-server
    networks:
      - opsmind

  anythingllm:
    image: mintplexlabs/anythingllm:latest
    container_name: opsmind-anythingllm
    cap_add:
      - SYS_ADMIN
    env_file:
      - .env
    environment:
      SERVER_PORT: 3001
      STORAGE_DIR: /app/server/storage
      JWT_SECRET: ${ANYTHINGLLM_JWT_SECRET}
      SIG_KEY: ${ANYTHINGLLM_SIG_KEY}
      SIG_SALT: ${ANYTHINGLLM_SIG_SALT}
      # LLM 提供商：generic-openai 支持所有 OpenAI 兼容 API
      LLM_PROVIDER: generic-openai
      GENERIC_OPEN_AI_BASE_PATH: ${LLM_BASE_URL}
      GENERIC_OPEN_AI_MODEL_PREF: ${LLM_MODEL}
      GENERIC_OPEN_AI_MODEL_TOKEN_LIMIT: ${LLM_TOKEN_LIMIT:-8192}
      GENERIC_OPEN_AI_API_KEY: ${LLM_API_KEY}
      EMBEDDING_ENGINE: generic-openai
      EMBEDDING_BASE_PATH: ${EMBEDDING_BASE_URL}
      EMBEDDING_MODEL_PREF: ${EMBEDDING_MODEL}
      EMBEDDING_MODEL_MAX_CHUNK_LENGTH: ${EMBEDDING_MAX_CHUNK_LENGTH:-8192}
      GENERIC_OPEN_AI_EMBEDDING_API_KEY: ${EMBEDDING_API_KEY}
      VECTOR_DB: lancedb
    volumes:
      - anythingllm_storage:/app/server/storage
      - anythingllm_hotdir:/app/collector/hotdir
      - anythingllm_outputs:/app/collector/outputs
    # 默认不暴露给外部；只有需要访问 AnythingLLM 管理页面时再临时打开端口。
    # ports:
    #   - "3001:3001"
    depends_on:
      - vllm
    networks:
      - opsmind

  vllm:
    image: vllm/vllm-openai:latest
    container_name: opsmind-vllm
    # 实际部署时按模型和 GPU 环境补充启动参数。
    # command: ["--model", "/models/qwen3-4b", "--model", "/models/bge-m3", "--served-model-name", "qwen3-4b,bge-m3"]
    volumes:
      - ./models:/models
    networks:
      - opsmind
    profiles:
      - ai-local

  postgres:
    image: pgvector/pgvector:pg18
    container_name: opsmind-postgres
    environment:
      POSTGRES_DB: opsmind
      POSTGRES_USER: opsmind
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    networks:
      - opsmind

  minio:
    image: minio/minio:latest
    container_name: opsmind-minio
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: ${MINIO_ROOT_USER}
      MINIO_ROOT_PASSWORD: ${MINIO_ROOT_PASSWORD}
    ports:
      - "9000:9000"
      - "9001:9001"
    volumes:
      - minio_data:/data
    networks:
      - opsmind

volumes:
  postgres_data:
  minio_data:
  anythingllm_storage:
  anythingllm_hotdir:
  anythingllm_outputs:

networks:
  opsmind:
    driver: bridge
```

### 3.2 LLM 提供商配置

AnythingLLM 通过 `generic-openai` 提供商对接所有 OpenAI 兼容 API。编辑 `.env` 选择一种方案：

<details>
<summary><b>方案 A：本地 vLLM（推荐，完全本地化）</b></summary>

```dotenv
LLM_BASE_URL=http://vllm:8000/v1
LLM_MODEL=qwen3-4b
LLM_TOKEN_LIMIT=8192
LLM_API_KEY=dummy-key
# Embedding 也用 vLLM
EMBEDDING_BASE_URL=http://vllm:8000/v1
EMBEDDING_MODEL=bge-m3
EMBEDDING_API_KEY=dummy-key
```

需要先下载模型（见 §3.3），然后通过 `--profile ai-local` 启动。
</details>

<details>
<summary><b>方案 B：OpenAI 官方 API（最简单）</b></summary>

```dotenv
LLM_BASE_URL=https://api.openai.com/v1
LLM_MODEL=gpt-4o-mini
LLM_TOKEN_LIMIT=16384
LLM_API_KEY=sk-your-openai-api-key
# Embedding
EMBEDDING_BASE_URL=https://api.openai.com/v1
EMBEDDING_MODEL=text-embedding-3-small
EMBEDDING_API_KEY=sk-your-openai-api-key
```

无需下载模型，`docker compose up -d` 即可。
</details>

<details>
<summary><b>方案 C：本地 Ollama（轻量级）</b></summary>

```dotenv
LLM_BASE_URL=http://host.docker.internal:11434/v1
LLM_MODEL=qwen2.5:7b
LLM_TOKEN_LIMIT=8192
LLM_API_KEY=dummy-key
# Embedding
EMBEDDING_BASE_URL=http://host.docker.internal:11434/v1
EMBEDDING_MODEL=nomic-embed-text
EMBEDDING_API_KEY=dummy-key
```

需先安装 Ollama 并 `ollama pull` 模型，无需 Docker 内 vLLM。
</details>

<details>
<summary><b>方案 D：其他兼容 API（DeepSeek / Moonshot / 等）</b></summary>

```dotenv
LLM_BASE_URL=https://api.deepseek.com/v1
LLM_MODEL=deepseek-chat
LLM_TOKEN_LIMIT=16384
LLM_API_KEY=sk-your-api-key
```

替换为对应服务的地址和 Key 即可。
</details>

完整的 `.env` 模板见项目根目录 `.env.example`。

### 3.3 下载本地模型（仅方案 A：vLLM 需要）

> 使用 OpenAI / DeepSeek 等云 API 可跳过此步骤。

| 模型 | 用途 | 大小 |
|------|------|------|
| Qwen3-4B-Instruct | 对话生成 | ~8 GB |
| BGE-M3 | 文本向量化 | ~2.2 GB |

**ModelScope 下载（国内推荐）：**

```powershell
pip install modelscope
cd D:\Projects\Personal\OpsMind
modelscope download --model Qwen/Qwen3-4B-Instruct-2507 --local_dir ./models/qwen3-4b
modelscope download --model BAAI/bge-m3 --local_dir ./models/bge-m3
```

或使用 Makefile：`make model-download`

**HuggingFace 下载：**

```powershell
pip install huggingface_hub
set HF_ENDPOINT=https://hf-mirror.com
huggingface-cli download Qwen/Qwen3-4B-Instruct-2507 --local-dir ./models/qwen3-4b
huggingface-cli download BAAI/bge-m3 --local-dir ./models/bge-m3
```

**配置并启动：**

编辑 `docker-compose.yml` 取消 vllm 服务的 `command` 注释，然后：

```powershell
docker compose --profile ai-local up -d --build
```

### 3.4 初始化 API Key

**推荐：一键自动初始化**

```powershell
# 在项目根目录执行
make setup
```

脚本会自动完成：
1. 启动 AnythingLLM 容器
2. 等待服务就绪
3. 自动尝试生成 API Key
4. 如自动生成失败，自动打开浏览器并引导手动创建
5. 将 API Key 写入 `.env`
6. 创建默认知识库工作区 `opsmind-it-ops`
7. 重启 OpsMind 后端（无编译，仅重启容器）

**备选：纯手动方式（不推荐）**

```powershell
cd D:\Projects\Personal\OpsMind

# 启动 AnythingLLM
docker compose up -d anythingllm

# 浏览器访问 http://localhost:3001，完成初始化向导
# → Settings → API Keys → Create API Key

# 将 Key 写入 .env
# ANYTHINGLLM_API_KEY=刚创建的APIKey

# 重启 OpsMind 后端（不需要 --build！API Key 是运行时环境变量）
docker compose up -d opsmind-server
```

> **注意：** API Key 是运行时环境变量，写入 `.env` 后只需要 `docker compose up -d` 重启容器，**不需要重新编译**。

---

## 4. OpsMind 后端配置

Go 后端配置只保留内部访问地址：

```yaml
anythingllm:
  base_url: http://anythingllm:3001/api
  api_key: ${ANYTHINGLLM_API_KEY}
  default_workspace_slug: opsmind-it-ops
  timeout_seconds: 20

ai:
  default_top_k: 5
  confidence_threshold: 0.6
```

注意：

- 容器内访问 AnythingLLM 用 `http://anythingllm:3001/api`。
- 只有本机调试或初始化管理页面时才使用 `http://localhost:3001`。
- 前端不得配置 AnythingLLM 地址和密钥。

---

## 5. AnythingLLM API 接入

统一请求头：

```text
Authorization: Bearer <ANYTHINGLLM_API_KEY>
Content-Type: application/json
```

在 OpsMind 后端容器内，`RagClient` 使用：

```text
base_url = http://anythingllm:3001/api
```

### 5.1 创建知识库 Workspace

OpsMind 中一个知识库对应 AnythingLLM 一个 workspace。

后端调用：

```http
POST http://anythingllm:3001/api/v1/workspace/new
```

请求体：

```json
{
  "name": "opsmind-it-ops",
  "chatMode": "query",
  "topN": 5,
  "similarityThreshold": 0.6,
  "openAiPrompt": "你是企业运维数字员工。只能基于知识库内容回答，并输出可执行处理步骤；无法确认时提示用户提交申告。"
}
```

返回中的 `workspace.slug` 保存到 OpsMind `knowledge_bases.rag_workspace_slug`。

### 5.2 同步 FAQ 或处理方案

结构化 FAQ 推荐使用 raw text 接口：

```http
POST http://anythingllm:3001/api/v1/document/raw-text
```

请求体：

```json
{
  "textContent": "问题：账号被冻结怎么办？\n答案：1. 确认账号归属；2. 联系管理员核验身份；3. 由管理员执行恢复；4. 首次登录后修改密码。",
  "addToWorkspaces": "opsmind-it-ops",
  "metadata": {
    "title": "账号冻结处理 FAQ",
    "docAuthor": "OpsMind",
    "description": "账号冻结和恢复处理流程",
    "docSource": "knowledge_articles:123",
    "chunkSource": "账号管理"
  }
}
```

文件类知识使用上传接口：

```http
POST http://anythingllm:3001/api/v1/document/upload
Content-Type: multipart/form-data
```

表单字段：

| 字段 | 说明 |
| --- | --- |
| `file` | 知识文档文件 |
| `addToWorkspaces` | workspace slug，例如 `opsmind-it-ops` |
| `metadata` | JSON 字符串，至少包含 `title`、`docSource` |

同步成功后，保存 AnythingLLM 返回的 `documents[0].location` 到 OpsMind `knowledge_articles.rag_document_location`。

### 5.3 门户智能问答

OpsMind 前端调用：

```http
POST /api/v1/portal/chat-sessions
```

OpsMind 后端调用 AnythingLLM：

```http
POST http://anythingllm:3001/api/v1/workspace/{slug}/chat
```

请求体：

```json
{
  "message": "账号冻结怎么处理？",
  "mode": "query",
  "sessionId": "opsmind-user-1001-session-20260605",
  "reset": false
}
```

AnythingLLM 典型返回：

```json
{
  "id": "chat-uuid",
  "type": "textResponse",
  "chatId": 1,
  "textResponse": "账号冻结处理步骤：...",
  "sources": [
    {
      "title": "账号冻结处理 FAQ",
      "text": "问题：账号被冻结怎么办？...",
      "score": 0.85
    }
  ],
  "close": true,
  "error": null,
  "metrics": {}
}
```

OpsMind 响应给前端：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "session_id": 12345,
    "answer": "账号冻结处理步骤：...",
    "sources": [
      {
        "doc_name": "账号冻结处理 FAQ",
        "chunk_content": "问题：账号被冻结怎么办？...",
        "confidence": 0.85
      }
    ],
    "confidence": 0.85,
    "can_submit_ticket": false,
    "duration_ms": 3200
  }
}
```

字段映射：

| AnythingLLM 字段 | OpsMind 字段 |
| --- | --- |
| `textResponse` | `answer` |
| `sources[].title` 或 `sources[].chunkSource` | `sources[].doc_name` |
| `sources[].text` | `sources[].chunk_content` |
| `max(sources[].score)` | `confidence` |
| `error != null` | AI 服务异常，触发降级 |
| `sources` 为空或 `confidence < threshold` | `can_submit_ticket=true` |

### 5.4 置信度预检，可选

如果需要先判断是否命中知识，可调用：

```http
POST http://anythingllm:3001/api/v1/workspace/{slug}/vector-search
```

请求体：

```json
{
  "query": "账号冻结怎么处理？",
  "topN": 5,
  "scoreThreshold": 0.6
}
```

MVP 不强制预检。默认从 `/chat` 返回的 `sources[].score` 计算置信度即可。

### 5.5 备用 OpenAI-compatible 方式

AnythingLLM 也提供：

```text
GET  /api/v1/openai/models
POST /api/v1/openai/chat/completions
POST /api/v1/openai/embeddings
```

该方式把 workspace slug 当作 `model`。MVP 不作为主接入方式，因为 OpsMind 需要稳定拿到 AnythingLLM 原生 `sources` 和 `chatId`。

---

## 6. OpsMind 已实现的代码文件

以下文件已在 M1-M6 各里程碑中全部实现，与 `docs/TECH.md` 的 Go + Vue 结构对齐：

### 6.1 后端（全部已实现 ✅）

| 文件 | 作用 |
| --- | --- |
| `server/internal/config/config.go` | `AnythingLLMConfig`、`AIConfig`，读取 Docker 内部 URL |
| `server/internal/adapter/rag_client.go` | `RagClient` 接口和 AnythingLLM HTTP 实现 |
| `server/tests/adapter/rag_client_test.go` | Mock AnythingLLM，验证请求、响应映射和异常降级 |
| `server/internal/service/chat_service.go` | `CreateChatSession`，调用 `RagClient.Query`，保存问答和来源 |
| `server/internal/service/knowledge_service.go` | 发布知识时调用 `RagClient.SyncDocument`，保存同步状态 |
| `server/internal/handler/chat.go` | `POST /api/v1/portal/chat-sessions` 和反馈接口 |
| `server/internal/handler/knowledge.go` | 知识发布、停用、重试同步接口 |
| `server/internal/model/chat.go` | 问题、答案、sources、confidence、feedback、duration |
| `server/internal/model/knowledge.go` | `rag_workspace_slug`、`rag_document_location`、`sync_status`、`sync_error` |
| `server/internal/repository/chat_repo.go` | 问答会话持久化 |
| `server/internal/repository/knowledge_repo.go` | 知识发布和同步状态持久化 |

`RagClient` 接口（与 TECH.md §7.1 完全对齐，包含 CreateWorkspace）：

```go
type RagClient interface {
    Query(ctx context.Context, req RAGQueryRequest) (*RAGQueryResponse, error)
    SyncDocument(ctx context.Context, req RAGSyncRequest) (*RAGSyncResponse, error)
    DisableDocument(ctx context.Context, req RAGDisableRequest) error
    CreateWorkspace(ctx context.Context, req RAGCreateWorkspaceRequest) (*RAGCreateWorkspaceResponse, error)
}
```

### 6.2 前端（全部已实现 ✅）

| 文件 | 作用 |
| --- | --- |
| `web/src/api/chat.ts` | 调用 OpsMind `POST /api/v1/portal/chat-sessions` |
| `web/src/views/portal/Chat.vue` | 智能问答页面，展示答案、来源、反馈、转申告入口 |
| `web/src/api/knowledge.ts` | 调用知识发布、重试同步接口 |
| `web/src/views/admin/KnowledgeList.vue` | 展示知识同步状态和失败原因 |
| `web/src/views/admin/ModelConfig.vue` | 配置 Top K、置信度阈值；不暴露 AnythingLLM API Key 明文 |

前端只展示 OpsMind 后端返回的数据，不直接请求 AnythingLLM。

---

## 7. 业务流程落地

### 7.1 智能问答

```text
用户提问
  -> web/src/views/portal/Chat.vue
  -> POST /api/v1/portal/chat-sessions
  -> ChatService.CreateChatSession
  -> RagClient.Query
  -> POST http://anythingllm:3001/api/v1/workspace/{slug}/chat
  -> 保存 chat_sessions
  -> 返回 answer、sources、confidence、can_submit_ticket
```

降级规则：

| 场景 | OpsMind 处理 |
| --- | --- |
| AnythingLLM 容器不可达 | 返回 `code=20001`，提示 AI 服务不可用 |
| vLLM 不可达导致 AnythingLLM 返回错误 | 记录错误，返回转人工提示 |
| AnythingLLM 返回 `error` | 记录错误，返回转人工提示 |
| `sources` 为空 | 返回兜底答案，`can_submit_ticket=true` |
| `confidence < threshold` | 返回兜底答案，`can_submit_ticket=true` |
| 成功且置信度达标 | 返回答案和来源，`can_submit_ticket=false` |

服务不可用兜底提示：

```text
当前 AI 服务暂不可用，请提交申告由人工处理
```

低置信度兜底提示：

```text
暂未找到足够匹配的知识，建议提交申告由运维人员人工处理
```

### 7.2 知识发布

```text
知识草稿
  -> 提交审核
  -> 审核通过
  -> KnowledgeService.Publish
  -> RagClient.SyncDocument
  -> AnythingLLM /document/raw-text 或 /document/upload
  -> 保存 rag_document_location 和 sync_status
  -> 写审计日志
```

同步状态建议：

| 值 | 状态 |
| --- | --- |
| `pending` | 待同步 |
| `synced` | 已同步 |
| `failed` | 同步失败 |
| `disabled` | 已停用 |

停用知识时：

1. OpsMind 先把知识状态改为停用。
2. 调用 AnythingLLM `POST /api/v1/workspace/{slug}/update-embeddings`，`deletes` 传入该知识的 `rag_document_location`。
3. 记录审计日志。

---

## 8. 测试和验证

### 8.1 一键启动验证

```powershell
cd D:\Projects\Personal\OpsMind
docker compose up -d --build
docker compose ps
```

期望：

- `opsmind-server` 正常运行。
- `opsmind-web` 正常运行。
- `opsmind-anythingllm` 正常运行。
- `opsmind-postgres` 正常运行。
- `opsmind-minio` 正常运行。

### 8.2 容器内连通性

从 OpsMind 后端容器访问 AnythingLLM：

```powershell
docker compose exec opsmind-server sh -c "wget -qO- http://anythingllm:3001/api/v1/workspaces"
```

期望：如果 API Key 校验生效，会返回未授权或 API Key 错误；这说明网络可达。带 API Key 的真实请求由后端 `RagClient` 发起。

### 8.3 API 级集成测试

建议覆盖：

| 测试文件 | 验证点 |
| --- | --- |
| `server/tests/adapter/rag_client_test.go` | Authorization header、内部 URL、超时、响应字段映射 |
| `server/tests/service/chat_service_test.go` | 低置信度转人工、AI 异常降级、会话保存 |
| `server/tests/service/knowledge_service_test.go` | 发布后同步、失败状态、重试同步 |

---

## 9. 一致性要求

实现和文档必须统一遵守：

| 项 | 统一口径 |
| --- | --- |
| 用户入口 | 只打开 OpsMind Web |
| AnythingLLM 部署 | OpsMind `docker-compose.yml` 内部组件 |
| 后端访问地址 | `http://anythingllm:3001/api` |
| 本机管理地址 | `http://localhost:3001` 只用于初始化或排障 |
| API Key | 后端密文保存，前端不下发 |
| 问答接口 | OpsMind 对外暴露 `POST /api/v1/portal/chat-sessions` |
| AnythingLLM 问答 | 后端内部调用 `/api/v1/workspace/{slug}/chat` |
| MVP 输出 | 同步返回完整答案，不接 SSE |
| 向量库 | AnythingLLM 默认 LanceDB；OpsMind pgvector 用于系统侧追溯和扩展 |

