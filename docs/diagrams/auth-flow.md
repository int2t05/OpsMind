# 认证与 RBAC 权限 — 函数级调用链

> 代码基准：`handler/auth.go` / `middleware/auth.go` / `middleware/rbac.go` / `service/auth_service.go`
> 更新于 2026-06-12 — TokenType 区分 access/refresh，中间件双令牌校验

## 1. 登录与双令牌生成

```mermaid
sequenceDiagram
    actor U as 用户
    participant AH as AuthHandler.Login<br/>handler/auth.go
    participant AS as AuthService.Login<br/>service/auth_service.go:34
    participant UR as UserRepo.GetByUsername<br/>repository/user_repo.go:41
    participant DB as PostgreSQL
    participant JWT as jwt.GenerateAccessToken<br/>jwt.GenerateRefreshToken<br/>pkg/jwt/jwt.go

    U->>AH: POST /api/v1/auth/login<br/>{username, password}
    AH->>AH: c.ShouldBindJSON(&LoginRequest)

    AH->>AS: Login(req.Username, req.Password)

    AS->>UR: GetByUsername(username)
    UR->>DB: SELECT * FROM users WHERE username=?
    DB-->>UR: *User{PasswordHash, Status}

    AS->>AS: bcrypt.CompareHashAndPassword(hash, password)
    alt 密码错误
        AS-->>AH: AppError{10001, "用户名或密码错误"}
        AH-->>U: 401
    end

    AS->>AS: 状态检查: Status == Active(1)

    AS->>JWT: GenerateAccessToken(userID, username)
    JWT->>JWT: Claims{UserID, Username, TokenType: "access"}<br/>ExpiresAt: now + 2h
    JWT-->>AS: accessToken

    AS->>JWT: GenerateRefreshToken(userID, username)
    JWT->>JWT: Claims{UserID, Username, TokenType: "refresh"}<br/>ExpiresAt: now + 168h
    JWT-->>AS: refreshToken

    AS-->>AH: *LoginResponse{AccessToken, RefreshToken, User}
    AH-->>U: 200 {access_token, refresh_token, user}
```

## 2. 请求认证链（中间件）

```mermaid
sequenceDiagram
    actor C as 客户端
    participant M1 as Recovery<br/>gin.Recovery()
    participant M2 as RequestID<br/>middleware.RequestID()
    participant M3 as CORS<br/>middleware.CORS(origins)
    participant M4 as Logger<br/>middleware.Logger()
    participant M5 as JWTAuth<br/>middleware/auth.go:32
    participant M6 as RBAC<br/>middleware/rbac.go
    participant H as Handler

    C->>M1: HTTP Request
    M1->>M2: next()
    M2->>M2: c.Set("requestID", uuid)
    M2->>M3: next()
    M3->>M3: AllowOrigin 校验
    M3->>M4: next()
    M4->>M4: 记录 start time
    M4->>M5: c.Next()

    Note over M5: === JWT 认证 ===
    M5->>M5: c.GetHeader("Authorization")
    M5->>M5: strings.TrimPrefix("Bearer ")
    M5->>M5: jwt.ParseWithClaims(token, &Claims{}, keyFunc)

    alt 无 token / 解析失败
        M5-->>C: 401 {code:10001, message:"未登录"}
    end

    M5->>M5: claims.TokenType == "access" 校验
    alt TokenType != "access"
        M5-->>C: 401 {code:10001, message:"无效的令牌类型"}
    end

    M5->>M5: c.Set("userID", claims.UserID)
    M5->>M5: c.Set("username", claims.Username)
    M5->>M6: c.Next()

    Note over M6: === RBAC 权限校验 ===
    M6->>M6: c.Get("userPermissions").([]string)
    M6->>M6: 检查 requiredPermission ∈ userPermissions

    alt 无权限
        M6-->>C: 403 {code:10002, message:"无权限"}
    end

    M6->>H: c.Next()

    H-->>C: 200 / 503 / ...
```

## 3. Token 刷新

```mermaid
sequenceDiagram
    actor C as 客户端
    participant AH as AuthHandler.Refresh<br/>handler/auth.go
    participant JWT as pkg/jwt

    C->>AH: POST /api/v1/auth/refresh<br/>{refresh_token}
    AH->>JWT: jwt.ParseWithClaims(refreshToken, &Claims{})
    JWT-->>AH: *Claims{TokenType: "refresh", UserID, Username}

    AH->>AH: claims.TokenType == "refresh" 校验

    AH->>JWT: GenerateAccessToken(claims.UserID, claims.Username)
    JWT-->>AH: newAccessToken

    AH->>JWT: GenerateRefreshToken(claims.UserID, claims.Username)
    JWT-->>AH: newRefreshToken

    AH-->>C: 200 {access_token, refresh_token}
```

## 4. 路由分组

```mermaid
flowchart TD
    R[gin.Engine] --> Public["/api/v1/auth<br/>(无中间件)"]
    R --> AuthMe["/api/v1/auth/me<br/>+ JWTAuth"]
    R --> Portal["/api/v1/portal<br/>+ JWTAuth"]
    R --> Admin["/api/v1/admin<br/>+ JWTAuth + RBAC"]

    Public --> Login["POST /login → AuthHandler.Login"]
    Public --> Refresh["POST /refresh → AuthHandler.Refresh"]

    AuthMe --> ChangePwd["POST /change-password → AuthHandler.ChangePassword"]
    AuthMe --> Logout["POST /logout → AuthHandler.Logout"]

    Portal --> Chat["POST /chat-sessions → ChatHandler"]
    Portal --> ChatStream["POST /chat-sessions/stream → ChatHandler"]
    Portal --> Tickets["CRUD /tickets → TicketHandler"]
    Portal --> Messages["/messages → MessageHandler"]

    Admin --> Dashboard["GET /dashboard → DashboardHandler"]
    Admin --> AdminTickets["/tickets → TicketHandler (admin)"]
    Admin --> Users["/users → UserHandler"]
    Admin --> Roles["/roles → RoleHandler"]
    Admin --> KB["/knowledge-bases → KnowledgeHandler"]
    Admin --> LLMConfig["/llm-configs → LLMConfigHandler"]
    Admin --> Audit["/audit-logs → AuditHandler"]
    Admin --> Config["/config → ConfigHandler"]

    style Public fill:#22c55e20,stroke:#22c55e
    style AuthMe fill:#f59e0b20,stroke:#f59e0b
    style Portal fill:#5e6ad220,stroke:#5e6ad2
    style Admin fill:#ef444420,stroke:#ef4444
```
