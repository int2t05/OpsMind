# 审计日志 + 系统配置 + 站内消息

## 审计日志

> 基础路径：`/api/v1/admin/audit-logs` | 认证：JWT + RBAC

### 查询审计日志

```http
GET /api/v1/admin/audit-logs?page=1&page_size=10&operator_id=0&action=
Authorization: Bearer <token>
```

**查询参数：**

| 参数 | 类型 | 默认 | 说明 |
|------|------|------|------|
| page | int | 1 | 页码 |
| page_size | int | 10 | 每页条数（最大 100） |
| operator_id | int | 0 | 操作人 ID（0=全部） |
| action | string | "" | 操作类型（空=全部） |

**响应：**

```json
{
  "code": 0,
  "data": [
    {
      "id": 1,
      "operator_id": 1,
      "operator_name": "admin",
      "action": "knowledge:publish",
      "target_type": "knowledge_article",
      "target_id": 5,
      "detail": "{\"title\":\"VPN 密码重置 FAQ\"}",
      "ip_address": "192.168.1.1",
      "created_at": "2026-06-11 20:30:00"
    }
  ],
  "total": 150,
  "page": 1,
  "page_size": 10
}
```

**记录的操作类型：**

| action | 说明 |
|--------|------|
| `user:create` | 创建用户 |
| `user:update` | 更新用户 |
| `user:freeze` | 冻结用户 |
| `user:unfreeze` | 恢复用户 |
| `knowledge:create` | 创建知识文章 |
| `knowledge:publish` | 发布知识 |
| `knowledge:disable` | 停用知识 |
| `ticket:status_change` | 申告状态变更 |

---

## 系统配置

> 基础路径：`/api/v1/admin/configs` | 认证：JWT + RBAC

### 获取配置

```http
GET /api/v1/admin/configs/:key
Authorization: Bearer <token>
```

**响应：**

```json
{
  "code": 0,
  "data": {
    "key": "ai_confidence_threshold",
    "value": "0.6"
  }
}
```

**可用配置键：**

| key | 默认值 | 说明 |
|-----|--------|------|
| `app_name` | OpsMind | 应用名称 |

> `ai_confidence_threshold` 和 `ai_default_top_k` 在 [llm-configs API](llm-config.md) 中统一管理。LLM 配置热替换生效，无需重启。

### 更新配置

```http
PUT /api/v1/admin/configs/:key
Authorization: Bearer <token>
```

**请求体：**

```json
{
  "value": "OpsMind"
}
```

---

## 站内消息（门户端）

> 基础路径：`/api/v1/portal/messages` | 认证：JWT

### 消息列表

```http
GET /api/v1/portal/messages?page=1&page_size=10
Authorization: Bearer <token>
```

**响应：**

```json
{
  "code": 0,
  "data": [
    {
      "id": 1,
      "title": "申告「公司邮箱无法登录」需补充信息",
      "content": "请提供错误截图和发生时间",
      "type": "ticket_supplement",
      "is_read": false,
      "created_at": "2026-06-11 15:30:00"
    }
  ],
  "total": 3,
  "page": 1,
  "page_size": 10
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| type | string | 消息类型：`ticket_supplement` / `ticket_resolved` / `system_notice` |
| is_read | bool | 是否已读 |

### 标记已读

```http
PUT /api/v1/portal/messages/:id/read
Authorization: Bearer <token>
```

**成功响应：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "unread_count": 3
  }
}
```

> 标记已读后返回当前未读消息总数，便于前端同步 badge 计数。

### 未读计数

```http
GET /api/v1/portal/messages/unread-count
Authorization: Bearer <token>
```

**响应：**

```json
{
  "code": 0,
  "data": {
    "count": 5
  }
}
```

---

## 健康检查

> 无需认证，供 Docker/K8s 存活探针使用。

```http
GET /health
```

**响应：**

```json
{
  "status": "ok"
}
```
