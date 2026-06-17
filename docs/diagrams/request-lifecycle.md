# 请求生命周期 (Request Lifecycle)

> **覆盖模块：** `middleware/` → `router/` → `handler/` → `service/` → `repository/`

---

## 1. 完整请求处理链

```mermaid
sequenceDiagram
    autonumber
    actor Client as 客户端
    participant Gin as Gin Engine
    participant RID as middleware.RequestID
    participant CORS as middleware.CORS
    participant LOG as middleware.Logger
    participant JWT as middleware.JWTAuth
    participant RBAC as middleware.RequirePermission
    participant H as Handler
    participant S as Service
    participant R as Repository
    participant DB as PostgreSQL

    Client->>Gin: HTTP Request

    Note over Gin: 全局中间件链 (router.go:Setup)
    Gin->>REC: Recovery() (最外层，捕获后续中间件 panic)
    Gin->>RID: RequestID()
    RID->>RID: 生成/透传 X-Request-ID (UUID)
    Gin->>CORS: CORS()
    CORS->>CORS: 校验 Origin / Method / Header
    Gin->>LOG: Logger()
    LOG->>LOG: 记录 method/path/status/latency/IP

    alt 公开路由 /api/v1/auth/*
        Gin->>H: 直接进入 Handler (无 JWT/RBAC)
    else 门户路由 /api/v1/portal/*
        Gin->>JWT: JWTAuth(secret)
        JWT->>JWT: 提取 Bearer token → ParseToken → c.Set("currentUser")
        JWT->>H: c.Next()
    else 后台路由 /api/v1/admin/*
        Gin->>JWT: JWTAuth(secret)
        JWT->>JWT: 提取 Bearer token → ParseToken → c.Set("currentUser")
        JWT->>RBAC: c.Next()
        RBAC->>RBAC: c.Get("currentUser") → hasAnyPermission()
        RBAC->>H: c.Next()
    end

    H->>H: c.ShouldBindJSON(&RequestDTO)
    H->>S: 调用 Service 方法

    Note over S: 业务逻辑层 (service/)
    S->>S: 业务校验 (唯一性/状态机/密码策略等)
    S->>R: 调用 Repository 方法

    Note over R: 数据访问层 (repository/)
    R->>DB: GORM 查询 (Find/Create/Update/Delete)
    DB-->>R: 结果 / 错误
    R-->>S: 数据 / error

    S-->>H: 响应 DTO / AppError

    H->>H: response.Success(c, data) / response.Error(c, code, msg)
    H-->>Gin: JSON Response
    LOG->>LOG: 写入结构化日志 (status/latency)
    Gin-->>Client: HTTP Response
```

---

## 2. 中间件注册顺序

```mermaid
flowchart LR
    subgraph Router["router.Setup(cfg, handlers)"]
        direction TB
        S1["gin.SetMode(cfg.Server.Mode)"]
        S2["r = gin.New()"]
        S3["r.Use(RequestID())"]
        S4["r.Use(CORS())"]
        S5["r.Use(Logger())"]
        S6["r.Use(Recovery())"]
        S7["GET /health"]
    end

    S1 --> S2 --> S3 --> S4 --> S5 --> S6 --> S7

    subgraph Public["公开路由组 /api/v1/auth"]
        A1["registerPublicRoutes(rg, h)"]
        A2["POST /login → AuthHandler.Login"]
        A3["POST /refresh → AuthHandler.Refresh"]
        A4["POST /change-password → AuthHandler.ChangePassword"]
        A5["POST /logout → AuthHandler.Logout"]
    end

    subgraph Portal["门户路由组 /api/v1/portal"]
        P1["portal.Use(JWTAuth(secret))"]
        P2["registerPortalRoutes(rg)"]
        P3["POST /chat-sessions → ChatHandler"]
        P4["POST /tickets, GET /tickets → TicketHandler"]
        P5["GET /messages → MessageHandler"]
    end

    subgraph Admin["后台路由组 /api/v1/admin"]
        D1["admin.Use(JWTAuth(secret))"]
        D2["registerAdminRoutes(rg, h)"]
        D3["/users/* → UserHandler + RequirePermission('user:manage')"]
        D4["/roles/* → RoleHandler + RequirePermission('user:manage')"]
        D5["/tickets/*, /knowledge-* → TicketHandler / KnowledgeHandler"]
    end

    S7 --> Public
    S7 --> Portal
    S7 --> Admin
```

---

## 3. 错误处理路径

```mermaid
flowchart TD
    REQ[HTTP 请求] --> MW{中间件链}
    MW -->|401| E1["middleware.abortWithError → response.Error(10001)"]
    MW -->|403| E2["middleware.abortWithError → response.Error(10002)"]
    MW -->|放行| H[Handler]

    H -->|参数错误| E3["response.Error(10003)"]
    H -->|调用 Service| S[Service]

    S -->|业务错误| E4["AppError{Code, Message}"]
    S -->|Repository 错误| E5["fmt.Errorf / gorm error"]
    S -->|成功| OK[返回 DTO]

    E4 --> H2["handler.handleServiceError"]
    H2 -->|AppError| R1["response.Error(appErr.Code, appErr.Message)"]
    H2 -->|其他| R2["response.Error(99999, '服务器内部错误')"]

    E5 --> H2

    OK --> H3["response.Success(c, data)"]
    H3 --> RES["JSON {code:0, data:...}"]
    R1 --> RES2["JSON {code:xxxx, message:...}"]
    R2 --> RES2
    E1 --> RES2
    E2 --> RES2
    E3 --> RES2

    subgraph mapHTTPStatus["HTTP 状态码映射 (pkg/response)"]
        M1["10001 → 401"]
        M2["10002 → 403"]
        M3["10003 → 400"]
        M4["10004 → 404"]
        M5["10005 → 409"]
        M6["default → 500"]
    end
```

---

## 4. 初始化启动顺序 (cmd/main.go)

```mermaid
flowchart TD
    START["main()"] --> S1["slog.Info('OpsMind 服务启动中...')"]
    S1 --> S2["config.Load('')"]
    S2 --> S3{"cfg.JWT.Secret 为空?"}
    S3 -->|release 模式| EXIT["slog.Error → os.Exit(1)"]
    S3 -->|debug 模式| WARN["slog.Warn (仅警告)"]
    S3 -->|非空| OK1
    WARN --> OK1

    OK1["database.Init(cfg.Database)"]
    OK1 --> OK2["database.AutoMigrate(db) — 16 张表"]
    OK2 --> OK3["repository.NewUserRepo(db)"]
    OK3 --> OK4["repository.NewRoleRepo(db)"]
    OK4 --> OK5["service.NewAuthService(userRepo, db)"]
    OK5 --> OK6["service.NewUserService(userRepo, db)"]
    OK6 --> OK7["service.NewRoleService(roleRepo, db)"]
    OK7 --> OK8["handler.NewAuthHandler(authService)"]
    OK8 --> OK9["handler.NewUserHandler(userService)"]
    OK9 --> OK10["handler.NewRoleHandler(roleService)"]
    OK10 --> OK11["router.Setup(cfg, &Handlers{...})"]
    OK11 --> OK12["r.Run(':8080')"]

    style EXIT fill:#e74c3c,color:#fff
    style OK12 fill:#2ecc71,color:#fff
```
