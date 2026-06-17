# 用户与角色权限管理流程 (User & RBAC Management Flow)

> **涉及文件：** `handler/user.go` → `service/user_service.go` → `repository/user_repo.go`
> **角色：** `handler/role.go` → `service/role_service.go` → `repository/role_repo.go`
> **中间件：** `middleware/auth.go` (JWTAuth), `middleware/rbac.go` (RequirePermission)

---

## 1. 用户 CRUD 完整流程

```mermaid
sequenceDiagram
    actor A as 系统管理员
    participant UH as UserHandler<br/>handler/user.go
    participant US as UserService<br/>service/user_service.go
    participant UR as UserRepo<br/>repository/user_repo.go
    participant B as pkg/hash
    participant DB as PostgreSQL

    Note over A,DB: ===== 创建用户 =====
    A->>UH: POST /api/v1/admin/users<br/>{username, password, real_name, phone, email, role_ids}
    UH->>US: s.UserService.Create(req)
    
    US->>UR: ExistsByUsername(username)
    UR->>DB: SELECT COUNT(*) FROM users WHERE username = ?
    DB-->>UR: exists (bool)
    
    alt 用户名已存在
        US-->>UH: AppError{10005, "用户名已存在"}
        UH-->>A: {"code": 10005}
    end
    
    US->>B: ValidatePassword(password)
    Note over B: 正则: ^(?=.*[a-z])(?=.*[A-Z])(?=.*\d).{8,32}$
    alt 密码不符合策略
        US-->>UH: AppError{10003, "密码必须包含..."}
        UH-->>A: {"code": 10003}
    end
    
    US->>B: HashPassword(password)
    B-->>US: passwordHash (bcrypt cost=10)
    
    US->>UR: Create(&User{username, passwordHash, realName, phone, email, Status:1, FirstLogin:true})
    UR->>DB: INSERT INTO users
    
    US->>UR: AssignRoles(userID, roleIDs)
    UR->>DB: DELETE FROM user_roles WHERE user_id=?<br/>INSERT INTO user_roles (user_id, role_id) VALUES ...
    
    US-->>UH: nil
    UH-->>A: {"code": 0}

    Note over A,DB: ===== 冻结用户 =====
    A->>UH: PATCH /api/v1/admin/users/:id/freeze
    UH->>US: s.UserService.Freeze(id)
    
    US->>UR: GetByID(id)
    UR->>DB: SELECT * FROM users WHERE id = ?
    DB-->>UR: *User{Status: 1}
    
    alt user.Status == 2 (已冻结)
        US-->>UH: AppError{10006, "用户已被冻结"}
        UH-->>A: {"code": 10006}
    end
    
    US->>UR: UpdateStatus(id, 2)
    UR->>DB: UPDATE users SET status = 2 WHERE id = ?
    
    US-->>UH: nil
    UH-->>A: {"code": 0}

    Note over A,DB: ===== 恢复用户 =====
    A->>UH: PATCH /api/v1/admin/users/:id/unfreeze
    UH->>US: s.UserService.Restore(id)
    
    US->>UR: GetByID(id)
    
    alt user.Status == 1 (已是正常状态)
        US-->>UH: AppError{10007, "用户已处于正常状态"}
        UH-->>A: {"code": 10007}
    end
    
    US->>UR: UpdateStatus(id, 1)
    UR->>DB: UPDATE users SET status = 1
    
    US-->>UH: nil
    UH-->>A: {"code": 0}

    Note over A,DB: ===== 查询用户列表 =====
    A->>UH: GET /api/v1/admin/users?page=1&page_size=20&keyword=张三
    UH->>US: s.UserService.List(page, pageSize, keyword)
    
    US->>UR: List(page, pageSize, keyword)
    UR->>DB: SELECT * FROM users<br/>WHERE username ILIKE '%keyword%'<br/>OR real_name ILIKE '%keyword%'<br/>LIMIT ? OFFSET ?
    DB-->>UR: []User, total
    
    loop 每个 user
        US->>UR: GetUserRoles(user.ID)
        UR->>DB: SELECT r.name FROM roles r<br/>JOIN user_roles ur ON r.id = ur.role_id<br/>WHERE ur.user_id = ?
        DB-->>UR: []Role
    end
    
    US-->>UH: *UserListResponse{Users: []UserDetailResponse, Total}
    UH-->>A: {"code": 0, "data": {users, total, page, page_size}}
```

---

## 2. 角色权限管理流程

```mermaid
sequenceDiagram
    actor A as 系统管理员
    participant RH as RoleHandler<br/>handler/role.go
    participant RS as RoleService<br/>service/role_service.go
    participant RR as RoleRepo<br/>repository/role_repo.go
    participant UR as UserRepo
    participant DB as PostgreSQL

    Note over A,DB: ===== 创建角色 + 绑定菜单 =====
    A->>RH: POST /api/v1/admin/roles<br/>{name, description, permissions[], menu_ids[]}
    RH->>RS: s.RoleService.Create(name, description, permissions)
    RS->>RR: CreateRole(&Role{Name, Description, Permissions(JSONB)})
    RR->>DB: INSERT INTO roles
    
    RS->>UR: UpdateRoleMenus(roleID, menuIDs)
    UR->>DB: DELETE FROM role_menus WHERE role_id = ?<br/>INSERT INTO role_menus (role_id, menu_id) VALUES ...
    
    RS-->>RH: nil
    RH-->>A: {"code": 0}

    Note over A,DB: ===== 查询角色详情（含菜单树）=====
    A->>RH: GET /api/v1/admin/roles/:id
    RH->>RS: s.RoleService.GetByID(id)
    RS->>RR: FindRoleByID(id)
    RR->>DB: SELECT * FROM roles WHERE id = ?
    DB-->>RR: *Role
    
    RS->>UR: GetRoleMenus(roleID)
    UR->>DB: SELECT m.* FROM menus m<br/>JOIN role_menus rm ON m.id = rm.menu_id<br/>WHERE rm.role_id = ?
    DB-->>UR: []Menu
    
    RS->>RS: buildTree(menus, 0) → 菜单树
    
    RS-->>RH: *Role (含 Menus 树)
    RH-->>A: {"code": 0, "data": {role, menus}}
```

---

## 3. 权限数据模型关系

```mermaid
erDiagram
    USERS ||--o{ USER_ROLES : "AssignRoles()"
    ROLES ||--o{ USER_ROLES : has
    ROLES ||--o{ ROLE_MENUS : "UpdateRoleMenus()"
    MENUS ||--o{ ROLE_MENUS : belongs_to
    ROLES ||--o{ USER_ROLES : "GetUserRoles()"
    USERS ||--o{ USER_ROLES : "GetUserPermissions()"

    USERS {
        bigint id PK
        varchar username UK
        varchar password_hash
        varchar real_name
        varchar phone
        varchar email
        smallint status "1正常 2冻结"
        boolean first_login
    }

    ROLES {
        bigint id PK
        varchar name UK
        varchar description
        jsonb permissions "ticket.read, user.manage 等权限"
    }

    USER_ROLES {
        bigint user_id PK_FK
        bigint role_id PK_FK
    }

    MENUS {
        bigint id PK
        varchar name
        varchar path
        varchar icon
        bigint parent_id "0=一级菜单"
        int sort_order
        varchar type "catalog/menu/button"
    }

    ROLE_MENUS {
        bigint role_id PK_FK
        bigint menu_id PK_FK
    }
```

---

## 4. 登录响应构建流程 (buildLoginResponse)

```mermaid
flowchart TD
    Start([AuthService.buildLoginResponse]) --> GetRoles[UserRepo.GetUserRoles<br/>查询 user_roles + roles]
    GetRoles --> ExtractNames["提取 roleNames[]"]
    
    ExtractNames --> GetPerms[UserRepo.GetUserPermissions<br/>聚合所有角色的 permissions JSONB]
    GetPerms --> Perms["permissions[] 字符串数组"]
    
    Perms --> BuildMenu[buildMenuTree]
    BuildMenu --> IsAdmin{系统管理员?}
    
    IsAdmin -->|是| AllMenus[UserRepo.ListMenus<br/>获取全部菜单]
    IsAdmin -->|否| AggMenus[遍历每个角色:<br/>UserRepo.GetRoleMenus<br/>按 menu.ID 去重]
    
    AllMenus --> BuildTree["buildTree menus, parentID=0<br/>递归构建树结构"]
    AggMenus --> BuildTree
    
    BuildTree --> GenAT[GenerateAccessToken<br/>2h 有效期]
    GenAT --> GenRT[GenerateRefreshToken<br/>7d 有效期]
    
    GenRT --> Response["LoginResponse{<br/>  AccessToken, RefreshToken,<br/>  UserInfo, Roles[],<br/>  Permissions[], Menus[]<br/>}"]
```
