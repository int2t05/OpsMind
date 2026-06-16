# 智能问答 RAG 管道 — 函数级调用链

> 代码基准：`handler/chat.go` → `service/chat_service.go` → `service/llm_service.go` → `rag/pipeline.go` → `adapter/llm_client.go`
> 更新于 2026-06-16 — 会话/对话分离：CreateSession 仅创建容器，StreamChatMessage 在已有会话中流式对话

## 1. SSE 流式问答 — 完整函数调用链

```mermaid
sequenceDiagram
    actor U as 用户
    participant CH as ChatHandler.StreamChatMessage<br/>handler/chat.go:160
    participant CS as ChatService.StreamChat<br/>service/chat_service.go:119
    participant LS as LLMService.StreamChat<br/>service/llm_service.go:198
    participant SR as ChatRepo.FindByID<br/>repository/chat_repo.go
    participant Pipe as Pipeline.Execute<br/>rag/pipeline.go:52
    participant QR as QueryRewrite<br/>rag/query_rewrite.go
    participant MR as MultiRoute<br/>rag/multi_route.go
    participant VR as VectorStore.CosineSearch<br/>adapter/vector_store.go
    participant B5 as BM25Retriever.Retrieve<br/>rag/bm25.go
    participant HF as HybridFuse<br/>rag/hybrid.go
    participant RR as Rerank<br/>rag/rerank.go
    participant LLM as OpenAIClient.ChatCompletionStream<br/>adapter/llm_client.go:191
    participant CR as ChatRepo<br/>repository/chat_repo.go
    participant DB as PostgreSQL

    U->>CH: POST /api/v1/portal/chat-sessions/:id/stream<br/>{question}
    CH->>CH: parseID("id") → sessionID
    CH->>CH: c.ShouldBindJSON(&SendMessageRequest)
    CH->>CH: getCurrentUserID(c) → userID
    CH->>CH: Set SSE headers + c.Status(200)
    CH->>CH: c.Request.Context() → ctx

    CH->>CS: StreamChat(ctx, sessionID, question, userID)

    Note over CS: === 1. 参数校验 + 会话加载 ===
    CS->>CS: strings.TrimSpace(question) — 非空校验
    CS->>SR: FindByID(sessionID)
    SR->>DB: SELECT FROM chat_sessions WHERE id=?
    DB-->>SR: *ChatSession
    CS->>CS: 校验 session.UserID == userID（归属检查）
    CS->>SR: FindMessagesBySession(sessionID) → 历史消息

    Note over CS,LS: === 2. RAG 管道 + LLM 流式 (单次调用) ===
    CS->>LS: StreamChat(ctx, question, kbID, opts, history)

    Note over LS,Pipe: === 2a. RAG 检索 ===
    LS->>LS: onStep = func(evt) { eventCh ← StreamEvent{Type:"step"} }
    LS->>Pipe: Execute(ctx, question, kbID, opts, onStep)

    alt QueryRewrite = true
        Pipe->>QR: rewrite(ctx, question, history)
        QR->>LLM: ChatCompletion(ctx, systemPrompt + question)
        LLM-->>QR: rewrittenQuery
    end

    alt MultiRoute = true
        Pipe->>MR: route(ctx, rewrittenQuery)
        MR->>LLM: ChatCompletion(ctx, routingPrompt)
        LLM-->>MR: []subQueries (2-4个)
    end

    par 向量检索
        Pipe->>VR: CosineSearch(ctx, kbID, embedding, topK)
        VR->>DB: SELECT * FROM knowledge_chunks<br/>ORDER BY embedding <=> $1 LIMIT $2
        DB-->>VR: []SearchResult{ChunkID, Score}
    and BM25 检索 (Hybrid=true)
        Pipe->>B5: Retrieve(ctx, kbID, query)
        B5->>B5: gse 分词 → 倒排索引 → Okapi BM25(k1=1.5,b=0.75)
        B5-->>Pipe: []bm25Result
    end

    alt Hybrid = true
        Pipe->>HF: fuse(vectorResults, bm25Results)
        Note over HF: RRF(k=60): score = Σ 1/(60+rank_i)
        HF-->>Pipe: []fusedResult
    end

    alt Rerank = true
        Pipe->>RR: Rerank(ctx, question, topCandidates)
        RR->>LLM: ChatCompletion(ctx, rerankPrompt)
        LLM-->>RR: rerankOrder
    end

    Pipe-->>LS: *RAGResult{Chunks []RetrievalResult, Metrics}

    Note over LS,LLM: === 2b. 构建 prompt + LLM 流式生成 ===
    LS->>LS: eventCh ← StreamEvent{Type:"step", ID:"llm_generate"}
    LS->>LS: buildMessages(chunks, question, history) → [system, ...history, user]
    LS->>LS: getModelConfig() → model + maxTokens

    LS->>LLM: ChatCompletionStream(ctx, ChatRequest{Model, Messages, MaxTokens, Temperature:0.3})

    loop 逐 token（实时 SSE）
        LLM-->>LS: StreamChunk{Content, FinishReason}
        LS->>LS: answerBuf.WriteString(chunk.Content)
        LS->>LS: eventCh ← StreamEvent{Type:"token", Content}
    end

    LS->>LS: extractSources + maxConfidence
    LS->>LS: eventCh ← StreamEvent{Type:"done", Metadata:{Answer, Sources, Confidence, DurationMS, Pipeline}}

    LS-->>CS: eventCh (通过 channel 代理事件)

    Note over CS: === 3. 会话持久化（done 事件时） ===
    CS->>CS: done 事件 → 填充 SessionID/Question/Feedback/CreatedAt
    CS->>CR: UpdateSession(&ChatSession{ID, Answer, Sources, Confidence, DurationMs})
    CR->>DB: UPDATE chat_sessions SET answer=?, sources=?, confidence=?, duration_ms=?
    CS->>CR: CreateBatch([{user}, {assistant}])
    CR->>DB: INSERT INTO chat_messages

    Note over CH: === 4. SSE 事件代理 ===
    loop 逐事件 (step/token/error/done)
        CS-->>CH: StreamEvent (通过 outCh channel)
        CH->>CH: writeSSEEvent(w, evt)
        CH->>CH: flusher.Flush()
        CH->>CH: rc.SetWriteDeadline(now + 30s)
    end

    CH-->>U: SSE stream complete
```

## 2. 会话创建

```mermaid
sequenceDiagram
    actor U as 用户
    participant CH as ChatHandler.CreateChatSession<br/>handler/chat.go:40
    participant CS as ChatService.CreateSession<br/>service/chat_service.go:87
    participant KR as KnowledgeRepo.FindKBByID
    participant CR as ChatRepo.Create
    participant DB as PostgreSQL

    U->>CH: POST /api/v1/portal/chat-sessions<br/>{kb_id, title?}
    CH->>CH: c.ShouldBindJSON(&CreateSessionRequest)
    CH->>CH: getCurrentUserID(c) → userID
    CH->>CS: CreateSession(req, userID)

    CS->>KR: FindKBByID(req.KBID) — 校验知识库存在
    CS->>CS: title = "" → "新会话"

    CS->>CR: Create(&ChatSession{UserID, KBID, Question: title})
    CR->>DB: INSERT INTO chat_sessions

    CS-->>CH: *ChatSession{ID, KBID, Question, CreatedAt}
    CH-->>U: 200 {code:0, data:{session_id, kb_id, question, created_at}}
```

## 3. 降级矩阵

```mermaid
flowchart TD
    Start([Pipeline.Execute]) --> QR{QueryRewrite?}
    QR -->|true| QR_LLM[QueryRewrite → LLMClient.ChatCompletion]
    QR -->|false| MR
    QR_LLM -->|OK| MR{MultiRoute?}
    QR_LLM -->|fail| QR_DG[降级：使用原始 question]
    QR_DG --> MR

    MR -->|true| MR_LLM[MultiRoute → LLMClient.ChatCompletion]
    MR -->|false| VR
    MR_LLM -->|OK| VR[VectorStore.CosineSearch]
    MR_LLM -->|fail| VR_DG[降级：单路检索]
    VR_DG --> VR

    VR -->|OK| BM{Hybrid?}
    VR -->|fail ❌| VRFail[返回 code=20002 ErrRAGUnavailable]

    BM -->|true| BM25[BM25Retriever.Retrieve]
    BM -->|false| Rerank
    BM25 -->|OK| Fuse[HybridFuse: RRF k=60]
    BM25 -->|fail| BM_DG[降级：仅向量结果]
    BM_DG --> Rerank
    Fuse --> Rerank

    Rerank{Rerank?} -->|true| Rerank_LLM[Rerank → LLMClient.ChatCompletion]
    Rerank -->|false| LLMGen
    Rerank_LLM -->|OK| LLMGen[LLMClient.ChatCompletion → 生成答案]
    Rerank_LLM -->|fail| Rerank_DG[降级：RRF 排序结果]
    Rerank_DG --> LLMGen

    LLMGen -->|OK| Done([返回答案])
    LLMGen -->|fail ❌| LLMFail[返回 code=20001 ErrAIUnavailable]

    style VRFail fill:#ef444420,stroke:#ef4444
    style LLMFail fill:#ef444420,stroke:#ef4444
    style QR_DG fill:#f59e0b20,stroke:#f59e0b
    style VR_DG fill:#f59e0b20,stroke:#f59e0b
    style BM_DG fill:#f59e0b20,stroke:#f59e0b
    style Rerank_DG fill:#f59e0b20,stroke:#f59e0b
```
