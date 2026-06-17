# OpsMind API 集成测试

> 基于 Playwright APIRequestContext 的 REST API 集成测试套件。
> 不启动浏览器，直接通过 HTTP 请求验证 API 契约。

## 快速开始

```bash
cd test

# 安装依赖（仅需执行一次）
npm install

# 安装 Playwright（仅需执行一次，不需要浏览器二进制文件）
npx playwright install --no-shell
# 或者只安装 Chromium（如果需要浏览器测试）
# npx playwright install chromium

# 运行认证设置（登录获取 token）
npm run test:auth

# 运行全部 API 测试
npm run test

# 使用 UI 模式查看测试
npm run test:ui

# 查看测试报告
npm run report
```

## 目录结构

```
test/
├── playwright.config.ts      # Playwright 配置
├── package.json              # 依赖和脚本
├── auth/
│   └── auth.setup.ts         # 认证设置（登录保存 token）
├── api/
│   ├── auth.spec.ts          # 认证接口（登录/刷新/改密/登出）
│   ├── knowledge.spec.ts     # 知识库管理（KB/文章/审核/发布/文档上传）
│   ├── chat.spec.ts          # 智能问答（SSE 流式 + 非流式 + 反馈）
│   ├── tickets.spec.ts       # 申告管理（门户提交 + 后台处理）
│   ├── users.spec.ts         # 用户管理（CRUD + 冻结/恢复）
│   ├── roles.spec.ts         # 角色与菜单管理
│   ├── llm-config.spec.ts    # LLM 配置（llama.cpp / OpenAI-compatible）
│   ├── dashboard.spec.ts     # 数据看板（统计 + 趋势）
│   └── audit-log.spec.ts     # 审计日志 + 系统配置 + 站内消息
└── utils/
    └── test-helpers.ts       # 共享工具函数（校验/认证/工厂）
```

## 测试覆盖范围

### 认证（auth.spec.ts）
- ✅ 登录成功 — 返回 token、用户、角色、权限、菜单
- ✅ 错误密码 — code=10003
- ✅ 不存在用户 — code=10003
- ✅ 缺少字段 — 参数校验失败
- ✅ 刷新 token — 新旧 token 不同
- ✅ 无效 refresh_token — code=10001
- ✅ 修改密码 — 策略校验（短密码/弱密码）
- ✅ 登出 — 认证/未认证

### 知识库（knowledge.spec.ts）
- ✅ 门户端知识库列表（仅 id+name）
- ✅ 后台知识库列表（含管理字段）
- ✅ 创建/更新/删除知识库
- ✅ 创建文章（手动输入）— v2 title/content 字段
- ✅ 文章列表（分页 + 状态筛选）
- ✅ 文章详情（含 chunks）— v2 新增字段
- ✅ 更新文章（仅草稿/驳回状态）
- ✅ 提交审核 → 重复提交失败
- ✅ 文档上传 — 格式校验、知识库存在性
- ✅ 文档处理状态查询
- ✅ 发布/停用/启用 — 状态机校验
- ✅ 权限验证 — 无 token 返回 401

### 智能问答（chat.spec.ts）
- ✅ 非流式问答 — session_id/answer/sources/pipeline
- ✅ SSE 流式 — step/token/done 事件解析
- ✅ rag_options 参数校验（top_k 范围）
- ✅ 不传 rag_options 使用默认值
- ✅ 会话详情查询（不存在 → 404）
- ✅ 用户反馈（有效值/无效值）
- ✅ AI 服务不可用降级（20001/20002）
- ✅ 权限验证

### 申告管理（tickets.spec.ts）
- ✅ 门户创建申告
- ✅ 参数校验（缺少字段、无效 urgency）
- ✅ 我的申告列表（分页 + ticket_no 格式）
- ✅ 申告详情（不存在 → 404）
- ✅ 补充信息 — 状态机限制
- ✅ 后台申告列表（状态筛选、紧急程度筛选）
- ✅ 状态变更（无效 action、不存在申告）
- ✅ 处理记录管理
- ✅ 知识库候选生成
- ✅ 权限验证

### 用户管理（users.spec.ts）
- ✅ 用户列表（分页 + 敏感字段不泄露）
- ✅ 创建用户 — 密码策略、重复用户名(409)
- ✅ 用户详情（不存在 → 404）
- ✅ 更新用户
- ✅ 冻结用户 → 重复冻结失败
- ✅ 恢复用户
- ✅ 权限验证

### 角色管理（roles.spec.ts）
- ✅ 角色列表（分页 + permissions 数组）
- ✅ 创建角色 — 重复名称(409)
- ✅ 角色详情（不存在 → 404）
- ✅ 更新角色（permissions 全量替换）
- ✅ 删除角色 → 重复删除(404)
- ✅ 菜单列表（树形结构）
- ✅ 角色菜单权限更新
- ✅ 权限验证

### LLM 配置（llm-config.spec.ts）
- ✅ 配置列表 — v2 统一字段（embedding_model/vector_dimension）
- ✅ 创建 llama.cpp 配置
- ✅ 创建 OpenAI-compatible 配置 + API Key 掩码验证
- ✅ provider_type 校验
- ✅ 配置详情（不存在 → 404）
- ✅ 更新配置（不传 api_key 保留原值）
- ✅ 不能删除默认配置
- ✅ 测试连接（success/latency_ms/error 字段）
- ✅ 权限验证

### 数据看板（dashboard.spec.ts）
- ✅ 统计数据结构验证（7 个字段类型 + 合理性）
- ✅ 趋势数据（day 粒度 + 日期格式）
- ✅ 参数校验（缺失/格式错误/日期倒序）
- ✅ 权限验证

### 审计日志等（audit-log.spec.ts）
- ✅ 审计日志列表（分页 + 操作人/操作类型筛选）
- ✅ 系统配置获取/更新（不存在 key → 404）
- ✅ 站内消息列表 + 未读计数
- ✅ 健康检查（无需认证）

## 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `API_BASE_URL` | `http://localhost:8080` | API 服务地址 |
| `TEST_ADMIN_USER` | `admin` | 管理员用户名 |
| `TEST_ADMIN_PASS` | `Admin@123` | 管理员密码 |
| `TEST_REPORTER_USER` | `reporter` | 报障人用户名 |
| `TEST_REPORTER_PASS` | `Reporter@123` | 报障人密码 |

## 运行要求

- 后端服务需在 `API_BASE_URL` 启动
- 数据库需有预置的 admin 用户和基础数据（roles/menus/knowledge-bases）
- 部分测试（SSE 流式问答/测试连接）依赖 AI 服务可用性，服务不可用时优雅跳过

## CI/CD 集成

```yaml
# .github/workflows/api-tests.yml
- name: Run API integration tests
  run: |
    cd test
    npm ci
    npx playwright test --project=api-tests
  env:
    API_BASE_URL: http://localhost:8080
```
