# 认证流程 (Authentication Flow)

> **覆盖模块：** `handler/auth.go` → `service/auth_service.go` → `repository/user_repo.go` → `pkg/jwt` / `pkg/hash`
> **对应任务：** T11（认证 Service + Handler）、T12（JWT 中间件）

---

## 1. 登录流程 (POST /api/v1/auth/login)

```mermaid
sequenceDiagram
    autonumber
    actor Client as 客户端
    participant H as AuthHandler.Login
    participant S as AuthService.Login
    participant R as UserRepo
    participant DB as PostgreSQL
    participant JWT as pkg/jwt
    participant Hash as pkg/hash

    Client->>H: POST /api/v1/auth/login {username, password}
    H->>H: c.ShouldBindJSON(&LoginRequest)
    H->>S: Login(username, password)

    S->>R: GetByUsername(username)
    R->>DB: SELECT * FROM users WHERE username=?
    DB-->>R: User row / nil
    R-->>S: *User / gorm.ErrRecordNotFound

    alt 用户不存在
        S-->>H: AppError{10003, "用户名或密码错误"}
    end

    S->>Hash: CheckPassword(passwordHash, password)
    Hash-->>S: bool

    alt 密码错误
        S-->>H: AppError{10003, "用户名或密码错误"}
    end

    alt 状态=2 (冻结)
        S-->>H: AppError{10002, "账号已被冻结"}
    end

    S->>S: buildLoginResponse(user)

    rect rgb(40, 50, 60)
        Note over S: === buildLoginResponse 内部 ===
        S->>R: GetUserRoles(userID)
        R->>DB: JOIN user_roles + roles
        DB-->>R: []Role
        R-->>S: []Role → roleNames[]

        S->>R: GetUserPermissions(userID)
        R->>DB: JOIN user_roles + roles → Pluck names
        DB-->>R: []string (角色名)
        R-->>S: []string (权限列表，去重)

        S->>S: buildMenuTree(userID, roles)
        alt 系统管理员
            S->>R: ListMenus()
        else 普通用户
            loop 每个角色
                S->>R: GetRoleMenus(roleID)
            end
        end
        S->>S: buildTree(menus, parentID=0)

        S->>JWT: GenerateAccessToken(userID, username, roles, secret, 2h)
        JWT-->>S: access_token
        S->>JWT: GenerateRefreshToken(userID, username, roles, secret, 168h)
        JWT-->>S: refresh_token
    end

    S-->>H: *LoginResponse{access_token, refresh_token, user, roles, permissions, menus}
    H->>H: response.Success(c, resp)
    H-->>Client: 200 {code:0, data: LoginResponse}
```

---

## 2. 刷新令牌 (POST /api/v1/auth/refresh)

```mermaid
sequenceDiagram
    autonumber
    actor Client as 客户端
    participant H as AuthHandler.Refresh
    participant S as AuthService.RefreshToken
    participant JWT as pkg/jwt
    participant R as UserRepo

    Client->>H: POST /api/v1/auth/refresh {refresh_token}
    H->>H: c.ShouldBindJSON(&RefreshRequest)
    H->>S: RefreshToken(refreshToken)

    S->>JWT: ParseToken(refreshToken, secret)
    alt 令牌过期/无效
        JWT-->>S: error
        S-->>H: AppError{10001, "刷新令牌无效或已过期"}
    end
    JWT-->>S: *Claims{UserID, Username, Roles}

    S->>R: GetByID(claims.UserID)
    alt 用户不存在
        S-->>H: AppError{10001, "用户不存在"}
    end
    alt 状态=2 (冻结)
        S-->>H: AppError{10002, "账号已被冻结"}
    end

    S->>S: buildLoginResponse(user)
    Note over S: 同登录流程：查询角色/权限/菜单 → 生成新令牌对

    S-->>H: *LoginResponse
    H-->>Client: 200 {code:0, data: LoginResponse}
```

---

## 3. 修改密码 (POST /api/v1/auth/change-password)

```mermaid
sequenceDiagram
    autonumber
    actor Client as 客户端
    participant MW as middleware.JWTAuth
    participant H as AuthHandler.ChangePassword
    participant S as AuthService.ChangePassword
    participant Hash as pkg/hash
    participant R as UserRepo

    Client->>MW: POST /api/v1/auth/change-password {old_password, new_password}
    MW->>MW: ParseToken(Authorization header)
    MW->>MW: c.Set("userID", claims.UserID)
    MW->>H: c.Next()

    H->>H: c.Get("userID") → userID
    H->>H: c.ShouldBindJSON(&ChangePasswordRequest)
    H->>S: ChangePassword(userID, oldPwd, newPwd)

    S->>R: GetByID(userID)
    R-->>S: *User

    S->>Hash: CheckPassword(user.PasswordHash, oldPwd)
    alt 旧密码错误
        S-->>H: AppError{10003, "旧密码错误"}
    end

    S->>Hash: ValidatePassword(newPwd)
    Note over Hash: 正则: ^(?=.*[a-z])(?=.*[A-Z])(?=.*\d).{8,32}$
    alt 不符合策略
        S-->>H: AppError{10003, "密码不符合策略"}
    end

    S->>Hash: HashPassword(newPwd)
    Hash-->>S: bcrypt_hash (cost=10)

    S->>S: user.PasswordHash = newHash; user.FirstLogin = false
    S->>R: Update(user)
    R->>DB: UPDATE users SET password_hash=?, first_login=false

    S-->>H: nil (成功)
    H-->>Client: 200 {code:0}
```

---

## 4. JWT 认证中间件 (所有受保护路由)

```mermaid
flowchart TD
    A[HTTP 请求到达] --> B{middleware.JWTAuth}
    B --> C["c.GetHeader('Authorization')"]
    C --> D{Header 存在?}
    D -->|否| E["abortWithError(10001)<br/>'缺失 Authorization 头'"]
    D -->|是| F["strings.SplitN(' ', 2)"]
    F --> G{格式为 'Bearer &lt;token&gt;'?}
    G -->|否| H["abortWithError(10001)<br/>'格式错误'"]
    G -->|是| I["jwt.ParseToken(token, secret)"]
    I --> J{令牌有效?}
    J -->|否| K["abortWithError(10001)<br/>'令牌无效或已过期'"]
    J -->|是| L["resolvePermissions(claims.Roles)"]
    L --> M["c.Set('currentUser', CurrentUser{...})<br/>c.Set('userID', claims.UserID)"]
    M --> N["c.Next() → 进入下一个中间件或 Handler"]

    style E fill:#e74c3c,color:#fff
    style H fill:#e74c3c,color:#fff
    style K fill:#e74c3c,color:#fff
    style N fill:#2ecc71,color:#fff
```

---

## 5. RBAC 权限中间件 (后台管理路由)

```mermaid
flowchart TD
    A["middleware.RequirePermission(perms...)"]
    A --> B["c.Get('currentUser')"]
    B --> C{存在?}
    C -->|否| D["response.Error(10001, '未登录')"]
    C -->|是| E["currentUser, ok := val.(CurrentUser)"]
    E --> F{类型正确?}
    F -->|否| G["response.Error(99999)"]
    F -->|是| H["hasAnyPermission(userPerms, required)"]
    H --> I["构建 permSet map"]
    I --> J{任意 required 在 permSet 中?}
    J -->|否| K["response.Error(10002, '无权限执行此操作')"]
    J -->|是| L["c.Next() → Handler"]

    style D fill:#e74c3c,color:#fff
    style G fill:#e74c3c,color:#fff
    style K fill:#e74c3c,color:#fff
    style L fill:#2ecc71,color:#fff
```
