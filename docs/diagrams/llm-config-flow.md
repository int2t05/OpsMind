# LLM 配置管理 — 函数级调用链

> 代码基准：`handler/llm_config.go` → `service/llm_config_service.go` → `repository/llm_config_repo.go`
> 更新于 2026-06-12 — 构造函数注入 / APIKey MarshalJSON 自动脱敏 / 事务包裹

## 1. CRUD + 测试连接

```mermaid
sequenceDiagram
    actor A as 系统管理员
    participant LH as LLMConfigHandler<br/>handler/llm_config.go
    participant LS as LLMConfigService<br/>service/llm_config_service.go
    participant LR as LlmConfigRepo<br/>repository/llm_config_repo.go
    participant LLM as OpenAIClient.ChatCompletion<br/>adapter/llm_client.go
    participant DB as PostgreSQL

    Note over A,DB: ====== 创建配置 ======
    A->>LH: POST /api/v1/admin/llm-configs<br/>{name, provider_type, base_url, api_key, llm_model, embedding_model, ...}
    LH->>LH: c.ShouldBindJSON(&CreateLLMConfigRequest)
    LH->>LS: CreateConfig(name, providerType, baseURL, embeddingBaseURL,<br/>apiKey, llmModel, embeddingModel, maxTokens, vectorDimension, isDefault)

    alt isDefault = true
        LS->>LS: ls.db.Transaction(func(tx){...})
        Note over LS,DB: 事务内: ClearDefault() + Create(config)
        LS->>LR: ClearDefault()
        LR->>DB: UPDATE llm_configs SET is_default=false WHERE is_default=true
    end
    LS->>LR: Create(&LlmConfig{...})
    LR->>DB: INSERT INTO llm_configs
    LS-->>LH: nil

    Note over A,DB: ====== 列表查询 ======
    A->>LH: GET /api/v1/admin/llm-configs
    LH->>LS: ListConfigs()
    LS->>LR: List()
    LR->>DB: SELECT * FROM llm_configs ORDER BY id
    DB-->>LR: []LlmConfig
    LS->>LS: 转换为 LlmConfigResponse<br/>（自动 MarshalJSON: APIKey "sk-***cret"）
    LS-->>LH: []LlmConfigResponse
    LH-->>A: 200 {data: [{id, name, api_key: "sk-****cret", ...}]}

    Note over A,DB: ====== 测试连接 ======
    A->>LH: POST /api/v1/admin/llm-configs/:id/test
    LH->>LH: parseID(c) + h.svc.GetConfig(id)

    LH->>LLM: ChatCompletion(ctx, ChatRequest{<br/>Model: cfg.LLMModel, Messages: [{role:"user", content:"ping"}],<br/>MaxTokens: 1, Temperature: 0})

    LH->>LH: latency = time.Since(start).Milliseconds()

    alt LLM 响应成功
        LLM-->>LH: ChatResponse{Content, FinishReason, TokensUsed}
        LH-->>A: 200 {success: true, model, latency_ms, tokens_used}
    else LLM 不可达
        LLM-->>LH: error
        LH-->>A: 503 {code:20001, message:"连接测试失败: ..."}
    end
```

## 2. atomic.Value 热替换

```mermaid
sequenceDiagram
    participant LS as LLMConfigService
    participant MGR as LLMConfigManager<br/>atomic.Value
    participant LR as LlmConfigRepo
    participant CS as ChatService
    participant CL as OpenAIClient

    Note over LS,CL: === 系统启动时 ===
    LS->>LR: FindDefault()
    LR-->>LS: *LlmConfig{BaseURL, APIKey, LLMModel, ...}
    LS->>MGR: manager.cfg.Store(config)
    Note over MGR: atomic.Value 持有当前生效的 LLM 配置

    CS->>MGR: GetConfig()
    MGR-->>CS: *LlmConfig (当前生效)

    Note over LS,CL: === 更新默认配置时 ===
    LS->>LS: CreateConfig / UpdateConfig
    LS->>LS: ls.db.Transaction(func(tx){ ClearDefault() + Save() })

    LS->>MGR: manager.cfg.Store(newConfig)
    Note over MGR: atomic.Value 替换，无需重启<br/>下次 ChatService.GetConfig() 即获取新配置

    CS->>MGR: GetConfig()
    MGR-->>CS: *LlmConfig (新配置 — 热替换生效)
```

## 3. API Key 自动脱敏

```mermaid
flowchart LR
    Create["CreateConfig<br/>apiKey = sk-abc123..."] --> Save["LlmConfigRepo.Create<br/>明文写入 DB"]
    List[ListConfigs<br/>LlmConfigRepo.List] --> Marshal[LlmConfigResponse.MarshalJSON]
    Marshal --> Mask{"len(apiKey) > 8?"}
    Mask -->|Yes| Masked["sk-ab****23<br/>(前4+后4, 中间***)"]
    Mask -->|No| Short["****"]
    Masked --> Response[200 JSON Response]
    Short --> Response
```

## 4. 构造函数演进

```mermaid
flowchart TD
    subgraph Before["Setter 注入"]
        B1[NewLLMConfigHandler svc] --> B2[handler.SetLLMClient client]
        B2 --> B3["风险: 调用 TestConnection<br/>前未 SetLLMClient → nil panic"]
    end
    subgraph After["构造函数注入"]
        A1["NewLLMConfigHandler(svc, llmClient)"] --> A2["llmClient 可选传 nil<br/>TestConnection 检查 nil"]
    end
    Before -->|"§6.5 修复"| After
    style Before fill:#ef444420,stroke:#ef4444
    style After fill:#22c55e20,stroke:#22c55e
```
