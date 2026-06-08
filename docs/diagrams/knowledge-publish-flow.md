# 知识发布与同步流程 (Knowledge Publish Flow)

> **设计来源：** TECH.md §10.2 知识同步与停用流程 + PLAN.md T18/T19
> **对应任务：** T18（知识库 Service+Handler）、T19（Embedding 配置）、T20（RagClient 适配器）— M3 待实现
> **实现文件：** `handler/knowledge.go` → `service/knowledge_service.go` → `repository/knowledge_repo.go` → `adapter/rag_client.go`（🔲 M3）

---

## 1. 知识条目生命周期 (Status 状态机)

```mermaid
stateDiagram-v2
    [*] --> 草稿 : CreateArticle
    草稿 --> 草稿 : UpdateArticle (可编辑)
    草稿 --> 待审核 : SubmitReview
    草稿 --> 已停用 : Disable

    待审核 --> 已发布 : Review (通过) + Publish
    待审核 --> 驳回 : Review (驳回)<br/>需填写 review_comment

    驳回 --> 草稿 : UpdateArticle (重新编辑)
    驳回 --> 待审核 : SubmitReview

    已发布 --> 已停用 : Disable<br/>调用 RagClient.DisableDocument
    已停用 --> 已发布 : 重试同步? (RetrySync)

    note right of 待审核
        校验: reviewer_id ≠ created_by
        违反 → ErrorCode 10003
    end note

    note left of 已发布
        sync_status = 'synced' | 'failed'
        成功: 保存 rag_document_location
        失败: sync_status='failed', 记录 sync_error
    end note
```

---

## 2. 发布同步流程 (Publish → AnythingLLM)

```mermaid
sequenceDiagram
    autonumber
    actor Admin as 管理员
    participant H as KnowledgeHandler
    participant S as KnowledgeService
    participant R as KnowledgeRepo
    participant RC as RagClient (adapter)
    participant AL as AnythingLLM
    participant PG as PostgreSQL pgvector

    Note over Admin,PG: === POST /api/v1/admin/knowledge-articles/:id/publish ===

    Admin->>H: Publish 请求
    H->>H: c.ShouldBindJSON
    H->>S: Publish(articleID, publisherID)

    rect rgb(40, 50, 60)
        Note over S: === KnowledgeService.Publish ===

        S->>R: FindArticleByID(articleID)
        R-->>S: *KnowledgeArticle (含 KnowledgeBase)

        S->>S: 校验: article.Status == 3 (已审核通过)
        S->>S: 校验: article.CreatedBy != publisherID (不能自审自发)

        par AnythingLLM 同步
            S->>RC: SyncDocument(RAGSyncRequest{<br/>  WorkspaceSlug: kb.RAGWorkspaceSlug,<br/>  Content: "Q: article.Question\nA: article.Answer",<br/>  IsFile: false,<br/>  Metadata: {"docSource": "knowledge_articles:ID"}<br/>})
            RC->>AL: POST /api/v1/document/raw-text
            AL-->>RC: {documents: [{location: "custom-documents/..."}]}
            RC-->>S: &RAGSyncResponse{DocumentLocation}
        and pgvector 写入
            S->>S: 生成 embedding 向量
            Note over S: 按 kb.EmbeddingModel + kb.VectorDimension
            S->>R: CreateChunks([]KnowledgeChunk{{content, embedding, ...}})
            R->>PG: INSERT INTO knowledge_chunks (embedding) VALUES (...)
        end

        S->>R: UpdateArticleStatus(articleID, 3) → status='已发布'
        S->>R: UpdateChunkSyncStatus(articleID, "synced", "")
    end

    S-->>H: nil
    H-->>Admin: 200 {code:0}
```

---

## 3. RagClient 适配器接口

```mermaid
classDiagram
    class RagClient {
        <<interface>>
        +Query(ctx, RAGQueryRequest) *RAGQueryResponse
        +SyncDocument(ctx, RAGSyncRequest) *RAGSyncResponse
        +DisableDocument(ctx, RAGDisableRequest) error
        +CreateWorkspace(ctx, RAGCreateWorkspaceRequest) *RAGCreateWorkspaceResponse
    }

    class anythingLLMClient {
        -baseURL string
        -apiKey string
        -httpClient *http.Client
        +Query(...)
        +SyncDocument(...)
        +DisableDocument(...)
        +CreateWorkspace(...)
    }

    class RAGQueryRequest {
        +WorkspaceSlug string
        +Question string
        +SessionID string
        +TopK int
    }

    class RAGQueryResponse {
        +Answer string
        +Sources []RAGSource
        +Confidence float64
        +ChatID string
        +DurationMS int64
        +Error string
    }

    class RAGSource {
        +DocName string
        +ChunkContent string
        +Score float64
    }

    RagClient <|.. anythingLLMClient
    anythingLLMClient --> RAGQueryRequest : uses
    anythingLLMClient --> RAGQueryResponse : returns
    RAGQueryResponse --> RAGSource : contains
```

---

## 4. 知识同步失败与重试

```mermaid
flowchart TD
    A["Publish 被调用"] --> B["调用 RagClient.SyncDocument"]
    B --> C{AnythingLLM 响应?}

    C -->|成功| D["保存 rag_document_location<br/>sync_status = 'synced'"]
    C -->|失败/超时| E["sync_status = 'failed'<br/>sync_error = 错误详情"]

    D --> F["pgvector 写入"]
    F --> G{写入成功?}
    G -->|是| H["发布完成 ✅"]
    G -->|否| I["记录日志<br/>不影响 AnythingLLM 同步状态"]

    E --> J["管理员在后台看到失败状态"]
    J --> K["点击「重试同步」"]
    K --> L["RetrySync(articleID)"]
    L --> A

    style D fill:#2ecc71,color:#fff
    style E fill:#e74c3c,color:#fff
    style H fill:#2ecc71,color:#fff
```

---

## 5. 知识停用流程

```mermaid
sequenceDiagram
    autonumber
    actor Admin as 管理员
    participant H as KnowledgeHandler
    participant S as KnowledgeService
    participant R as KnowledgeRepo
    participant RC as RagClient
    participant AL as AnythingLLM

    Note over Admin,AL: === POST /api/v1/admin/knowledge-articles/:id/disable ===

    Admin->>H: Disable 请求
    H->>S: Disable(articleID)

    S->>R: FindArticleByID(articleID)
    R-->>S: *KnowledgeArticle

    S->>RC: DisableDocument(RAGDisableRequest{<br/>  WorkspaceSlug,<br/>  DocumentLocations: [article.RAGDocumentLocation]<br/>})
    RC->>AL: POST /api/v1/workspace/{slug}/update-embeddings<br/>{deletes: [document_location]}
    AL-->>RC: 200 OK

    S->>R: UpdateChunkStatusByArticleID(articleID, "disabled")
    S->>R: UpdateArticleStatus(articleID, 4)

    S-->>H: nil
    H-->>Admin: 200 {code:0}
```

---

## 6. 知识同步状态枚举

```mermaid
flowchart LR
    PENDING["pending<br/>待同步"] -->|Publish 成功| SYNCED["synced<br/>已同步"]
    PENDING -->|Publish 失败| FAILED["failed<br/>同步失败"]
    FAILED -->|RetrySync 成功| SYNCED
    SYNCED -->|Disable| DISABLED["disabled<br/>已停用"]
    FAILED -->|Disable| DISABLED
```
