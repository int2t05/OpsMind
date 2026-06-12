# Web 前端代码审查 TODO 清单

> 审查日期：2026-06-12 &emsp; 审查范围：`web/src/` 全部源文件 &emsp; 审查维度：简洁性 / 规范性 / 严谨性

---

## 🔴 P0 — 影响功能正确性（立即修复）

### 1. Pagination 分页事件不匹配导致分页失效

- **文件**: [`web/src/views/portal/Messages.vue`](../web/src/views/portal/Messages.vue#L32) / [`web/src/views/portal/TicketQuery.vue`](../web/src/views/portal/TicketQuery.vue#L45)
- **根因**: `Pagination` 组件仅 emit `update:current-page` 和 `update:page-size`，但两处模板使用 `@change="handlePageChange"` 监听不存在的事件
- **修复**: 改为 `@update:current-page="handlePageChange"` 或 `v-model:current-page`

### 2. 路由守卫未执行角色权限校验

- **文件**: [`web/src/router/index.ts`](../web/src/router/index.ts#L154-L168)
- **根因**: `beforeEach` 守卫只检查 token 是否存在，未读取 `to.meta.roles`。Portal 路由已定义 `meta.roles: ['reporter']` 但从未被执行
- **影响**: 任意已登录用户可访问所有 portal/admin 路由
- **修复**: 在守卫中获取 auth store 的 user roles，与 `to.meta.roles` 做交集判断

### 3. 空 catch 块静默吞错误（6 处）

| 文件 | 代码 |
|------|------|
| [`views/admin/AuditLog.vue`](../web/src/views/admin/AuditLog.vue) | `catch { /* ignore */ }` |
| [`views/admin/RoleManage.vue`](../web/src/views/admin/RoleManage.vue) | `catch { /* ignore */ }` |
| [`views/admin/TicketList.vue`](../web/src/views/admin/TicketList.vue) | `catch { /* API error */ }` |
| [`views/portal/Chat.vue`](../web/src/views/portal/Chat.vue) | `catch { /* 静默失败 */ }` |
| [`components/layout/PortalLayout.vue`](../web/src/views/../components/layout/PortalLayout.vue) | `catch { /* 静默失败 */ }` |
| [`views/admin/UserList.vue`](../web/src/views/admin/UserList.vue) `fetchRoles()` | `catch { /* ignore */ }` |

- **修复**: 每处至少添加 `console.error` + 用户可见的 toast 提示

### 4. API 模块全部函数返回 `any`

- **文件**: [`api/user.ts`](../web/src/api/user.ts)（6 个函数）、[`api/knowledge.ts`](../web/src/api/knowledge.ts)（17 个函数）
- **影响**: 项目中最大的两个 API 模块完全没有 TypeScript 类型保护
- **修复**: 为每个函数添加泛型参数，如 `request.get<ApiResponse<UserListData>>(...)`

---

## 🟡 P1 — 影响代码质量和可维护性（本周修复）

### 5. 17 个文件使用 `as any` 强制类型转换

- **涉及文件**: AuditLog, RoleManage, Dashboard, TicketDetail(admin), UserList, TicketList, SystemConfig, KnowledgeList, KnowledgeEdit, LLMConfig, ModelConfig, Messages, TicketQuery, TicketDetail(portal), Chat, PortalLayout
- **根因**: API 响应类型缺失 `ApiResponse<T>` 包装层，视图层只能强制断言
- **修复**: 建立共享 `ApiResponse<T>` 后，移除全部 `as any`

### 6. API 响应类型 `ApiResponse<T>` 定义分散

- **文件**: [`api/auth.ts`](../web/src/api/auth.ts) (L6)、[`api/dashboard.ts`](../web/src/api/dashboard.ts) (L8)
- **问题**: 同一类型在两处重复定义，其他 API 模块（message、ticket、admin、user、knowledge）缺少此包装
- **修复**: 创建 [`web/src/types/api.ts`](../web/src/types/index.ts) 统一定义并全项目引用

### 7. 401 重定向无防循环保护

- **文件**: [`utils/request.ts`](../web/src/utils/request.ts)
- **根因**: 拦截器收到 401 无条件 `router.push('/login')`，若用户已在 `/login` 页面则无限循环
- **修复**: 添加 `router.currentRoute.value.path !== '/login'` 判断

### 8. 7 个文件 toast 定时器未在组件卸载时清理

- **涉及文件**: TicketDetail(admin), UserList, SystemConfig, LLMConfig, ModelConfig, Messages(portal), TicketQuery(portal)
- **问题**: 使用 `setTimeout` 实现 toast 自动消失，但 `onUnmounted` 未清理定时器
- **修复**: 添加 `onUnmounted(() => clearTimeout(timer))`

### 9. 响应数据解包方式不统一（6 种模式）

| 模式 | 使用位置 |
|------|---------|
| `res?.data \|\| res` | AuditLog, TicketDetail(admin), TicketQuery, TicketDetail(portal) |
| `res?.items \|\| res?.data?.items \|\| res \|\| []` | UserList |
| `res?.data \|\| res?.items \|\| []` | TicketList |
| `res.data`（直接） | Login |
| `(res as any).data \|\| res` | Dashboard, LLMConfig, ModelConfig, Messages |
| `(res.data as any).items \|\| (res as any).items \|\| []` | KnowledgeList, KnowledgeEdit |

- **修复**: 统一 API 响应类型 `ApiResponse<T>` 后，视图层直接 `res.data` 即可

### 10. `Chat.vue` 使用数组索引作为 `v-for` key

- **文件**: [`views/portal/Chat.vue`](../web/src/views/portal/Chat.vue)
- **问题**: `v-for="(msg, i) in chatStore.messages" :key="i"` — 流式更新时 Vue patch 算法无法正确追踪元素
- **修复**: 在消息模型中添加唯一 ID 字段，使用 `msg.id` 作为 key

---

## 🟢 P2 — 影响代码一致性（本迭代修复）

### 11. 重复的辅助函数应提取为共享工具

| 函数 | 重复次数 | 涉及文件 |
|------|---------|---------|
| `urgencyText()` | ×4 | TicketDetail(admin), TicketList, TicketQuery, TicketDetail(portal) |
| `statusClass()` (ticket) | ×2 | TicketDetail(admin), TicketList |
| `statusClass()`/`statusText()` (knowledge) | ×2 | KnowledgeList, KnowledgeEdit |
| `processText()`/`processClass()` | ×2 | KnowledgeList, KnowledgeEdit |
| `formatDate()` | ×N | 几乎所有视图，实现细节略不同 |

- **修复**: 创建 `web/src/utils/ticket.ts`、`web/src/utils/knowledge.ts` 集中管理

### 12. 重复的 Toast 逻辑应提取为 composable

- **涉及文件**: TicketDetail(admin), UserList, SystemConfig, LLMConfig, ModelConfig（共 7 处）
- **问题**: 各自独立实现 `ref` + `showToast()` + `setTimeout` 自动消失 + 内联 HTML/CSS
- **修复**: 创建 `web/src/composables/useToast.ts`

### 13. SystemConfig 与 ModelConfig 管理相同配置项

- **文件**: [`views/admin/SystemConfig.vue`](../web/src/views/admin/SystemConfig.vue) / [`views/admin/ModelConfig.vue`](../web/src/views/admin/ModelConfig.vue)
- **问题**: 两个页面都操作 `ai.default_top_k` 和 `ai.confidence_threshold`，互相不同步
- **修复**: 合并为统一的 AI 配置入口，或共享配置服务

### 14. 死代码清理

| 文件 | 问题 |
|------|------|
| [`components/common/ConfirmDialog.vue`](../web/src/components/common/ConfirmDialog.vue) | 无任何视图/组件 import，所有页面使用内联模态框 |
| [`views/admin/KnowledgeEdit.vue`](../web/src/views/admin/KnowledgeEdit.vue) `fileIconClass()` | 生成的 CSS class 无对应样式规则 |
| [`components/common/StatusBadge.vue`](../web/src/components/common/StatusBadge.vue) `knowledge` 分支 | 状态映射与实际后端数据不一致（0-5 vs 0-3），且未被使用 |

### 15. Import 路径不统一

- **文件**: [`KnowledgeList.vue`](../web/src/views/admin/KnowledgeList.vue) / [`KnowledgeEdit.vue`](../web/src/views/admin/KnowledgeEdit.vue)
- **问题**: 使用相对路径 `../../api/knowledge` 而全项目统一使用 `@/api/...`
- **修复**: 统一为 `@/api/knowledge` 和 `@/components/common/Pagination.vue`

### 16. `radix-vue` 依赖未使用

- **文件**: [`package.json`](../web/package.json)
- **问题**: `radix-vue` 在 dependencies 中但未找到任何 import。CLAUDE.md 标注为核心依赖
- **处理**: 确认是否实际使用；若否，从 `package.json` 移除

---

## 🔵 P3 — 架构优化（后续重构）

### 17. 组件拆分（>300 行）

| 文件 | 行数 | 建议 |
|------|------|------|
| [`views/admin/LLMConfig.vue`](../web/src/views/admin/LLMConfig.vue) | **614** | 拆分为配置列表、编辑弹窗、连接测试子组件 |
| [`views/portal/Chat.vue`](../web/src/views/portal/Chat.vue) | **563** | 拆分为消息区、输入区、高级设置面板子组件 |
| [`views/admin/KnowledgeEdit.vue`](../web/src/views/admin/KnowledgeEdit.vue) | **402** | 拆分为文档上传区、元信息表单、处理状态子组件 |
| [`views/portal/TicketSubmit.vue`](../web/src/views/portal/TicketSubmit.vue) | **342** | 拆分表单字段组件、验证逻辑 |
| [`views/admin/ModelConfig.vue`](../web/src/views/admin/ModelConfig.vue) | **313** | 提取滑块配置子组件 |

### 18. 缺少的类型定义

- **文件**: 无 `web/src/types/` 目录（已创建骨架）
- **需新增**:
  - `types/api.ts` — `ApiResponse<T>`、`PageResponse<T>`
  - `types/menu.ts` — `MenuItem`（替代 `auth.ts` 的 `menus: any[]`）
  - `types/ticket.ts` — 申告状态枚举映射
  - `types/knowledge.ts` — 知识状态/处理状态枚举映射

### 19. `utils/request.ts` — Axios 拦截器增强

- 403 处理仅 `console.error`，应弹 toast 通知用户
- 缺少全局 loading 计数器（请求进行中自动展示加载指示器）
- login 接口返回 `refresh_token` 但拦截器未使用（token 过期应自动刷新）
- 泛型默认值 `T = any` 应改为 `T = unknown` 强制调用方显式声明

### 20. Router 增强

- 缺少 `scrollBehavior` 配置（页面导航不恢复/重置滚动位置）
- 缺少 JWT 过期主动检查（仅在守卫层判断 token 是否存在，过期 token 导致页面闪白）
- `/admin` 路由组缺少 `meta.roles` 定义

### 21. 根入口文件增强

- **文件**: [`main.ts`](../web/src/main.ts) — 缺少 `app.config.errorHandler`，Vue 渲染错误静默消失
- **文件**: [`App.vue`](../web/src/App.vue) — light 主题时 `naiveTheme = null`，Naive UI 回退到默认亮色主题，可能与自定义 CSS 变量不一致

### 22. 样式覆盖脆弱性

- **文件**: [`styles/global.css`](../web/src/styles/global.css)
- **问题**: `.n-menu`、`.n-config-provider`、`.n-button__content` 使用 `!important` 覆盖 Naive UI 样式
- **修复**: 改用更高特异性选择器替代 `!important`

### 23. API 模块细节修正

| 文件 | 问题 |
|------|------|
| `api/chat.ts` | `streamChatSession` 使用原生 `fetch`（无超时、无 401/403 拦截） |
| `api/chat.ts` | `catch (err: any)` 应为 `unknown` + 类型守卫 |
| `api/auth.ts` | `menus: any[]` 应使用 `MenuItem[]` |
| `api/auth.ts` | `refreshToken()` 无泛型参数 |
| `api/admin.ts` | `listAllTickets` 响应类型缺少 `ApiResponse` 包装，与后端结构不一致 |
| `api/message.ts` | `listMessages`/`getUnreadCount` 缺少 `ApiResponse` 包装 |
| `api/ticket.ts` | `createTicket`/`supplementTicket` 无泛型参数 |
| `api/ticket.ts` | `Record<string, any>` 应替换为具体类型 |
| `stores/chat.ts` | `sources: any[]` 应使用 `SourceItem[]` |
| `stores/chat.ts` | `(session as any).pipeline_metrics` 应在 `ChatSessionResponse` 中增加可选字段 |

---

## 📋 修复检查清单（按顺序执行）

- [ ] P0-1: 修复 Pagination `@change` → `@update:current-page`（Messages.vue + TicketQuery.vue）
- [ ] P0-2: 路由守卫增加 `meta.roles` 权限校验
- [ ] P0-3: 6 处空 catch 块添加 `console.error` + toast 提示
- [ ] P0-4: `api/user.ts` 6 个函数补全泛型
- [ ] P1-5: 创建 `types/api.ts` — `ApiResponse<T>` + `PageResponse<T>`
- [ ] P1-6: `api/knowledge.ts` 17 个函数补全泛型
- [ ] P1-7: `utils/request.ts` 401 处理增加路径判断防循环
- [ ] P1-8: 7 处 toast 添加 `onUnmounted` 定时器清理
- [ ] P1-9: 统一 API 响应解包为 `res.data` → 逐个视图清理 `as any`
- [ ] P1-10: `Chat.vue` `:key="i"` 改为消息唯一 ID
- [ ] P2-11: 提取 `utils/ticket.ts` / `utils/knowledge.ts` 共享工具函数
- [ ] P2-12: 创建 `composables/useToast.ts`
- [ ] P2-13: 合并 SystemConfig / ModelConfig AI 配置
- [ ] P2-14: 清理死代码（ConfirmDialog / fileIconClass / StatusBadge knowledge 分支）
- [ ] P2-15: KnowledgeList / KnowledgeEdit import 路径改为 `@/` 别名
- [ ] P2-16: 确认 radix-vue 实际使用情况
- [ ] P3-17: 拆分 5 个 >300 行的大组件
- [ ] P3-18: 补全 `types/` 目录下的共享类型
- [ ] P3-19: `request.ts` 增强（403 toast、全局 loading、token 刷新）
- [ ] P3-20: Router 增加 scrollBehavior + JWT 过期检查 + meta.roles
- [ ] P3-21: `main.ts` 增加 errorHandler + `App.vue` 完善 light theme
- [ ] P3-22: `global.css` 替换 `!important` 为高特异性选择器
- [ ] P3-23: API 模块细节修正（fetch→axios、`err: any`→`unknown`、`any[]`→`ConcreteType[]`）

---

> 代码中已通过 `TODO(scope):` 注释标记所有问题点，可直接搜索 `TODO(` 定位。
