# Contributing to OpsMind

感谢你对 OpsMind 的关注！我们欢迎任何形式的贡献。

## 开发环境搭建

```bash
# 克隆项目
git clone https://github.com/int2t05/OpsMind.git
cd OpsMind

# 启动依赖服务（PostgreSQL + MinIO + AnythingLLM）
make dev

# 后端
cd server
go mod tidy
go run ./cmd/main.go

# 前端（新终端）
cd web
npm install
npm run dev
```

## 代码规范

### 架构约束

- **三层架构：** Handler（参数校验/响应）→ Service（业务逻辑）→ Repository（数据访问）
- **适配层隔离：** 外部服务调用必须通过 `adapter/` 接口，禁止直接 HTTP 调用
- **统一响应：** `{"code": 0, "message": "success", "data": {...}}`
- **RBAC 鉴权：** 后台接口必须经过 JWT + RBAC 双重中间件

### 注释规范

- 文件头注释说明模块存在的原因
- 函数注释说明设计决策（为什么这样做）
- 所有注释使用中文

### 提交规范

使用中文 commit message，格式：`类型: 描述`

| 类型 | 说明 |
|------|------|
| `feat` | 新功能 |
| `fix` | 修复 bug |
| `docs` | 文档更新 |
| `refactor` | 代码重构 |
| `test` | 测试 |
| `chore` | 构建/工具 |

## 测试

```bash
# 后端单元测试
cd server && go test ./tests/... -v

# 集成测试（需要 PostgreSQL）
go test ./tests/... -v -tags=integration

# 前端类型检查
cd web && npm run type-check
```

## 提交流程

1. Fork 本仓库
2. 创建功能分支：`git checkout -b feat/my-feature`
3. 提交代码并确保测试通过
4. 提交 PR 到 `main` 分支
5. PR 描述中说明改动目的和验证方式

## 文档

- [技术架构](docs/TECH.md)
- [产品需求](docs/PRD.md)
- [RAG 集成方案](docs/ANYTHINGLLM_AI_INTEGRATION.md)
- [业务数据流图](docs/diagrams/)
