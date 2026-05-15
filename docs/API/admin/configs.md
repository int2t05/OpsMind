# 后台系统配置 API

## 查询系统配置

- 请求方法：`GET`
- 请求路径：`/api/v1/admin/configs`
- 功能描述：查询系统配置列表，包括 vLLM OpenAI-compatible、AnythingLLM、MinIO、知识同步等配置。敏感值默认脱敏。
- 认证要求：需要登录，权限码 `config:list`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| page | integer | 否 | 页码 |
| per_page | integer | 否 | 每页数量 |
| q | string | 否 | 配置键或备注关键词 |
| group | string | 否 | 配置分组，`model`、`rag`、`storage`、`queue`、`security` |

- 请求示例：

```http
GET /api/v1/admin/configs?group=rag&page=1&per_page=20
Authorization: Bearer <access_token>
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data[].id | integer | 配置 ID |
| data[].config_key | string | 配置键 |
| data[].config_value | string | 配置值，敏感值脱敏 |
| data[].value_type | string | 值类型 |
| data[].group | string | 配置分组 |
| data[].remark | string | 说明 |
| data[].updated_at | string | 更新时间 |
| meta | object | 分页信息 |

- 响应示例：

```json
{
  "data": [
    {
      "id": 20,
      "config_key": "rag.anythingllm.base_url",
      "config_value": "http://anythingllm:3001",
      "value_type": "string",
      "group": "rag",
      "remark": "AnythingLLM 服务地址",
      "updated_at": "2026-05-15T10:00:00+08:00"
    }
  ],
  "meta": {
    "total": 1,
    "page": 1,
    "per_page": 20,
    "total_pages": 1
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| unauthorized | 未登录 | 重新登录 |
| forbidden | 无配置查看权限 | 联系管理员授权 |
| validation_error | 查询参数非法 | 修正参数 |

## 更新系统配置

- 请求方法：`PATCH`
- 请求路径：`/api/v1/admin/configs/{config_key}`
- 功能描述：更新指定系统配置。用于调整 vLLM 地址、AnythingLLM 地址、MinIO bucket、同步参数等。
- 认证要求：需要登录，权限码 `config:update`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| config_key | string | 是 | 路径参数，配置键 |
| config_value | string | 是 | 配置值 |
| remark | string | 否 | 配置说明 |

- 请求示例：

```json
{
  "config_value": "http://anythingllm:3001",
  "remark": "AnythingLLM 服务地址"
}
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.config_key | string | 配置键 |
| data.config_value | string | 配置值，敏感值脱敏 |
| data.updated_at | string | 更新时间 |

- 响应示例：

```json
{
  "data": {
    "config_key": "rag.anythingllm.base_url",
    "config_value": "http://anythingllm:3001",
    "updated_at": "2026-05-15T18:30:00+08:00"
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| config_not_found | 配置不存在 | 检查配置键 |
| readonly_config | 内置只读配置不可修改 | 修改环境变量或部署配置 |
| invalid_config_value | 配置值不合法 | 按配置类型修正 |
| validation_error | 参数校验失败 | 修正字段 |

## 获取系统配置详情

- 请求方法：`GET`
- 请求路径：`/api/v1/admin/configs/{config_key}`
- 功能描述：查询单个配置详情。敏感配置默认脱敏，仅返回是否已配置。
- 认证要求：需要登录，权限码 `config:detail`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| config_key | string | 是 | 路径参数，配置键 |

- 请求示例：

```http
GET /api/v1/admin/configs/rag.anythingllm.base_url
Authorization: Bearer <access_token>
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.config_key | string | 配置键 |
| data.config_value | string | 配置值，敏感值脱敏 |
| data.value_type | string | 值类型 |
| data.group | string | 配置分组 |
| data.sensitive | boolean | 是否敏感配置 |
| data.readonly | boolean | 是否只读 |
| data.remark | string | 配置说明 |
| data.updated_by | object | 最近更新人 |
| data.updated_at | string | 更新时间 |

- 响应示例：

```json
{
  "data": {
    "config_key": "rag.anythingllm.base_url",
    "config_value": "http://anythingllm:3001",
    "value_type": "string",
    "group": "rag",
    "sensitive": false,
    "readonly": false,
    "remark": "AnythingLLM 服务地址",
    "updated_by": {
      "id": 1,
      "real_name": "系统管理员"
    },
    "updated_at": "2026-05-15T18:30:00+08:00"
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| config_not_found | 配置不存在 | 检查配置键 |
| forbidden | 无配置详情权限 | 联系管理员授权 |

## 测试外部服务连接

- 请求方法：`POST`
- 请求路径：`/api/v1/admin/configs/test-connections`
- 功能描述：测试 vLLM、AnythingLLM、MinIO、NATS 等 MVP 依赖服务是否可连接。
- 认证要求：需要登录，权限码 `config:test-connection`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| targets | array | 是 | 测试目标，支持 `vllm`、`anythingllm`、`minio`、`nats` |

- 请求示例：

```json
{
  "targets": ["vllm", "anythingllm", "minio", "nats"]
}
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data[].target | string | 测试目标 |
| data[].success | boolean | 是否成功 |
| data[].latency_ms | integer | 耗时毫秒 |
| data[].message | string | 测试结果说明 |

- 响应示例：

```json
{
  "data": [
    {
      "target": "anythingllm",
      "success": true,
      "latency_ms": 120,
      "message": "连接正常"
    },
    {
      "target": "vllm",
      "success": false,
      "latency_ms": 3000,
      "message": "连接超时"
    }
  ]
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| validation_error | targets 为空或非法 | 选择合法测试目标 |
| forbidden | 无连接测试权限 | 联系管理员授权 |

## 查询可选模型配置

- 请求方法：`GET`
- 请求路径：`/api/v1/admin/configs/model-options`
- 功能描述：查询可用于知识库配置的 embedding 模型和向量维度选项，以及当前 vLLM 默认生成模型。该接口只返回系统配置，不直接调用模型推理。
- 认证要求：需要登录，权限码 `config:model-options`。
- 请求参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| 无 | - | - | 无 |

- 请求示例：

```http
GET /api/v1/admin/configs/model-options
Authorization: Bearer <access_token>
```

- 响应参数：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.chat_models | array | vLLM OpenAI-compatible 生成模型选项 |
| data.embedding_models | array | embedding 模型选项 |
| data.embedding_models[].model | string | 模型名称 |
| data.embedding_models[].dimensions | array | 支持的向量维度 |
| data.default_knowledge_base_code | string | 默认知识库编码 |

- 响应示例：

```json
{
  "data": {
    "chat_models": [
      {
        "model": "qwen-local",
        "provider": "vllm",
        "default": true
      }
    ],
    "embedding_models": [
      {
        "model": "bge-m3",
        "dimensions": [1024],
        "default": true
      },
      {
        "model": "text-embedding-local",
        "dimensions": [768]
      }
    ],
    "default_knowledge_base_code": "ops-faq"
  }
}
```

- 错误码：

| 错误码 | 含义 | 处理方式 |
|--------|------|---------|
| forbidden | 无模型配置查看权限 | 联系管理员授权 |
| config_not_found | 模型选项配置缺失 | 初始化系统配置 |
