# 运维数字员工系统数据库设计文档

| 项目 | 内容 |
| --- | --- |
| 文档版本 | v1.0 |
| 日期 | 2026-05-15 |
| 需求来源 | [docs/PRD.md](./PRD.md) / [docs/TECH.md](./TECH.md) |
| 数据库选型 | PostgreSQL 18 + pgvector |

## 0. MVP 技术决策

- MVP 数据库固定采用 PostgreSQL 18，并启用 pgvector 扩展。
- MVP 模型调用固定通过 vLLM OpenAI-compatible 适配层完成，问答先同步返回完整答案。
- MVP RAG 服务固定通过 AnythingLLM 适配层接入，由 AnythingLLM 负责完整 RAG 流程，知识同步状态需要在业务库留痕。
- MVP 知识向量存储在 pgvector，后续如知识规模增长再评估 Qdrant。
- embedding 模型和向量维度在知识库级别配置；创建知识库时必须选择具体模型和维度，同一知识库下所有知识切片必须使用相同模型和相同维度。
- MVP 附件与知识原文固定通过 MinIO S3-compatible 适配层存储，数据库只保存文件元数据。
- MVP 运维账号管理只做系统内本地模拟，不对接真实企业账号中心。
- MVP 异步能力先满足知识同步和审计留痕，NATS JetStream 事件结果需回写同步记录。

## 1. ER 关系图

本期数据库围绕“认证授权、智能问答、申告闭环、知识库管理、账号管理、审计日志”六类核心业务展开。

### 1.1 文字版实体关系

- `sys_user` 1 对多 `sys_user_role`
- `sys_role` 1 对多 `sys_user_role`
- `sys_role` 1 对多 `sys_role_permission`
- `sys_permission` 1 对多 `sys_role_permission`
- `sys_user` 1 对多 `portal_chat_session`
- `portal_chat_session` 1 对多 `portal_chat_message`
- `portal_chat_session` 1 对多 `portal_chat_feedback`
- `portal_chat_session` 1 对多 `ticket`
- `ticket` 1 对多 `ticket_process_record`
- `ticket` 1 对多 `ticket_visit_record`
- `ticket` 0..1 对 1 `knowledge_candidate`
- `knowledge_base` 1 对多 `knowledge_category`
- `knowledge_base` 1 对多 `knowledge_article`
- `knowledge_category` 1 对多 `knowledge_article`
- `knowledge_article` 1 对多 `knowledge_chunk`
- `knowledge_article` 1 对多 `knowledge_review_record`
- `knowledge_article` 1 对多 `knowledge_sync_record`
- `sys_user` 1 对多 `audit_log`
- `sys_user` 1 对多 `sys_operation_log`
- `sys_user` 1 对多 `sys_login_log`

### 1.2 关系说明

- 认证授权采用 RBAC 模式，用户、角色、权限通过中间表关联。
- 问答会话独立保存，消息、反馈、转工单记录按会话展开，便于追踪完整问答链路。
- 申告单是人工处理的主对象，处理记录、回访记录、知识候选与申告单强关联。
- 知识库通过 `knowledge_base` 配置 embedding 模型和向量维度，同一知识库内知识切片必须保持一致。
- 知识条目按知识库和分类组织，审核、发布、同步都单独留痕，避免“已发布”与“已同步”混淆。
- 知识切片和向量单独存储在 `knowledge_chunk`，便于 pgvector 存储、重建索引和后续替换向量服务。
- 审计日志、登录日志、操作日志分表存储，便于按场景查询和归档。

## 2. 表结构设计

### 2.1 系统用户表 `sys_user`

| 表名 | 字段名 | 类型 | 说明 | 约束 |
|------|--------|------|------|------|
| sys_user | id | bigint | 主键 | PK |
| sys_user | username | varchar(64) | 登录账号 | NOT NULL, UNIQUE |
| sys_user | password_hash | varchar(255) | 密码哈希 | NOT NULL |
| sys_user | real_name | varchar(64) | 姓名 | NOT NULL |
| sys_user | phone | varchar(32) | 联系方式 | NULL |
| sys_user | email | varchar(128) | 邮箱 | NULL |
| sys_user | status | smallint | 账号状态，1正常 2冻结 | NOT NULL, DEFAULT 1 |
| sys_user | last_login_at | timestamptz | 最近登录时间 | NULL |
| sys_user | last_login_ip | varchar(64) | 最近登录 IP | NULL |
| sys_user | remark | varchar(255) | 备注 | NULL |
| sys_user | created_at | timestamptz | 创建时间 | NOT NULL |
| sys_user | updated_at | timestamptz | 更新时间 | NOT NULL |
| sys_user | deleted_at | timestamptz | 软删除时间 | NULL |

### 2.2 角色表 `sys_role`

| 表名 | 字段名 | 类型 | 说明 | 约束 |
|------|--------|------|------|------|
| sys_role | id | bigint | 主键 | PK |
| sys_role | code | varchar(64) | 角色编码 | NOT NULL, UNIQUE |
| sys_role | name | varchar(64) | 角色名称 | NOT NULL |
| sys_role | status | smallint | 状态，1启用 2停用 | NOT NULL, DEFAULT 1 |
| sys_role | remark | varchar(255) | 备注 | NULL |
| sys_role | created_at | timestamptz | 创建时间 | NOT NULL |
| sys_role | updated_at | timestamptz | 更新时间 | NOT NULL |

### 2.3 权限表 `sys_permission`

| 表名 | 字段名 | 类型 | 说明 | 约束 |
|------|--------|------|------|------|
| sys_permission | id | bigint | 主键 | PK |
| sys_permission | parent_id | bigint | 父权限 ID | NULL |
| sys_permission | type | smallint | 权限类型，1菜单 2按钮 3接口 | NOT NULL |
| sys_permission | name | varchar(64) | 权限名称 | NOT NULL |
| sys_permission | code | varchar(128) | 权限标识 | NOT NULL, UNIQUE |
| sys_permission | path | varchar(255) | 菜单路径或接口路径 | NULL |
| sys_permission | method | varchar(16) | HTTP 方法 | NULL |
| sys_permission | sort_order | int | 排序 | NOT NULL, DEFAULT 0 |
| sys_permission | visible | boolean | 是否可见 | NOT NULL, DEFAULT true |
| sys_permission | status | smallint | 状态，1启用 2停用 | NOT NULL, DEFAULT 1 |
| sys_permission | created_at | timestamptz | 创建时间 | NOT NULL |
| sys_permission | updated_at | timestamptz | 更新时间 | NOT NULL |

### 2.4 用户角色关联表 `sys_user_role`

| 表名 | 字段名 | 类型 | 说明 | 约束 |
|------|--------|------|------|------|
| sys_user_role | id | bigint | 主键 | PK |
| sys_user_role | user_id | bigint | 用户 ID | NOT NULL, FK |
| sys_user_role | role_id | bigint | 角色 ID | NOT NULL, FK |
| sys_user_role | created_at | timestamptz | 创建时间 | NOT NULL |

### 2.5 角色权限关联表 `sys_role_permission`

| 表名 | 字段名 | 类型 | 说明 | 约束 |
|------|--------|------|------|------|
| sys_role_permission | id | bigint | 主键 | PK |
| sys_role_permission | role_id | bigint | 角色 ID | NOT NULL, FK |
| sys_role_permission | permission_id | bigint | 权限 ID | NOT NULL, FK |
| sys_role_permission | created_at | timestamptz | 创建时间 | NOT NULL |

### 2.6 登录日志表 `sys_login_log`

| 表名 | 字段名 | 类型 | 说明 | 约束 |
|------|--------|------|------|------|
| sys_login_log | id | bigint | 主键 | PK |
| sys_login_log | user_id | bigint | 用户 ID | NULL, FK |
| sys_login_log | username | varchar(64) | 登录账号快照 | NOT NULL |
| sys_login_log | login_result | smallint | 登录结果，1成功 2失败 | NOT NULL |
| sys_login_log | fail_reason | varchar(255) | 失败原因 | NULL |
| sys_login_log | ip_address | varchar(64) | 登录 IP | NOT NULL |
| sys_login_log | user_agent | varchar(255) | 客户端信息 | NULL |
| sys_login_log | login_at | timestamptz | 登录时间 | NOT NULL |

### 2.7 操作日志表 `sys_operation_log`

| 表名 | 字段名 | 类型 | 说明 | 约束 |
|------|--------|------|------|------|
| sys_operation_log | id | bigint | 主键 | PK |
| sys_operation_log | user_id | bigint | 操作用户 ID | NULL, FK |
| sys_operation_log | module | varchar(64) | 所属模块 | NOT NULL |
| sys_operation_log | action | varchar(64) | 操作动作 | NOT NULL |
| sys_operation_log | target_type | varchar(64) | 目标类型 | NULL |
| sys_operation_log | target_id | bigint | 目标 ID | NULL |
| sys_operation_log | request_path | varchar(255) | 请求路径 | NOT NULL |
| sys_operation_log | request_method | varchar(16) | 请求方法 | NOT NULL |
| sys_operation_log | request_body | jsonb | 请求体快照 | NULL |
| sys_operation_log | response_code | int | 响应码 | NOT NULL |
| sys_operation_log | success | boolean | 是否成功 | NOT NULL |
| sys_operation_log | created_at | timestamptz | 创建时间 | NOT NULL |

### 2.8 审计日志表 `audit_log`

| 表名 | 字段名 | 类型 | 说明 | 约束 |
|------|--------|------|------|------|
| audit_log | id | bigint | 主键 | PK |
| audit_log | user_id | bigint | 操作人 ID | NULL, FK |
| audit_log | module | varchar(64) | 模块名 | NOT NULL |
| audit_log | action | varchar(64) | 审计动作 | NOT NULL |
| audit_log | biz_type | varchar(64) | 业务类型 | NOT NULL |
| audit_log | biz_id | bigint | 业务主键 | NULL |
| audit_log | before_data | jsonb | 变更前数据 | NULL |
| audit_log | after_data | jsonb | 变更后数据 | NULL |
| audit_log | ip_address | varchar(64) | 来源 IP | NULL |
| audit_log | created_at | timestamptz | 创建时间 | NOT NULL |

### 2.9 门户问答会话表 `portal_chat_session`

| 表名 | 字段名 | 类型 | 说明 | 约束 |
|------|--------|------|------|------|
| portal_chat_session | id | bigint | 主键 | PK |
| portal_chat_session | session_no | varchar(64) | 会话编号 | NOT NULL, UNIQUE |
| portal_chat_session | user_id | bigint | 发起用户 ID | NULL, FK |
| portal_chat_session | question | text | 原始问题 | NOT NULL |
| portal_chat_session | answer | text | 最终答案 | NULL |
| portal_chat_session | answer_source | jsonb | 来源列表、命中文档、片段 | NULL |
| portal_chat_session | confidence_score | numeric(5,2) | 置信度 | NULL |
| portal_chat_session | model_name | varchar(128) | 模型名称 | NULL |
| portal_chat_session | model_provider | varchar(64) | 模型服务标识，MVP 为 vllm | NOT NULL, DEFAULT 'vllm' |
| portal_chat_session | rag_provider | varchar(64) | RAG 服务标识 | NULL |
| portal_chat_session | status | smallint | 状态，1处理中 2已完成 3转人工 4失败 | NOT NULL, DEFAULT 1 |
| portal_chat_session | answered_at | timestamptz | 回答完成时间 | NULL |
| portal_chat_session | created_at | timestamptz | 创建时间 | NOT NULL |
| portal_chat_session | updated_at | timestamptz | 更新时间 | NOT NULL |

### 2.10 门户问答消息表 `portal_chat_message`

| 表名 | 字段名 | 类型 | 说明 | 约束 |
|------|--------|------|------|------|
| portal_chat_message | id | bigint | 主键 | PK |
| portal_chat_message | session_id | bigint | 会话 ID | NOT NULL, FK |
| portal_chat_message | role | varchar(16) | 消息角色，user/assistant/system | NOT NULL |
| portal_chat_message | content | text | 消息内容 | NOT NULL |
| portal_chat_message | token_count | int | Token 数量 | NULL |
| portal_chat_message | created_at | timestamptz | 创建时间 | NOT NULL |

### 2.11 问答反馈表 `portal_chat_feedback`

| 表名 | 字段名 | 类型 | 说明 | 约束 |
|------|--------|------|------|------|
| portal_chat_feedback | id | bigint | 主键 | PK |
| portal_chat_feedback | session_id | bigint | 会话 ID | NOT NULL, FK |
| portal_chat_feedback | user_id | bigint | 用户 ID | NULL, FK |
| portal_chat_feedback | feedback_type | smallint | 反馈类型，1已解决 2未解决 | NOT NULL |
| portal_chat_feedback | remark | varchar(255) | 反馈说明 | NULL |
| portal_chat_feedback | created_at | timestamptz | 创建时间 | NOT NULL |

### 2.12 申告单表 `ticket`

| 表名 | 字段名 | 类型 | 说明 | 约束 |
|------|--------|------|------|------|
| ticket | id | bigint | 主键 | PK |
| ticket | ticket_no | varchar(64) | 申告编号 | NOT NULL, UNIQUE |
| ticket | source_session_id | bigint | 来源问答会话 ID | NULL, FK |
| ticket | source_feedback_id | bigint | 来源反馈 ID | NULL, FK |
| ticket | reporter_name | varchar(64) | 报障人姓名 | NOT NULL |
| ticket | reporter_phone | varchar(32) | 联系方式 | NOT NULL |
| ticket | title | varchar(255) | 问题标题 | NOT NULL |
| ticket | description | text | 问题描述 | NOT NULL |
| ticket | impact_scope | varchar(255) | 影响范围 | NULL |
| ticket | urgency_level | smallint | 紧急程度，1低 2中 3高 | NOT NULL |
| ticket | status | smallint | 状态，1待处理 2处理中 3待补充 4已完成 5已关闭 | NOT NULL, DEFAULT 1 |
| ticket | assignee_id | bigint | 当前处理人 | NULL, FK |
| ticket | ai_context | jsonb | 原问答上下文 | NULL |
| ticket | attachment_count | int | 附件数量 | NOT NULL, DEFAULT 0 |
| ticket | closed_reason | varchar(255) | 关闭原因 | NULL |
| ticket | created_at | timestamptz | 创建时间 | NOT NULL |
| ticket | updated_at | timestamptz | 更新时间 | NOT NULL |

### 2.13 申告处理记录表 `ticket_process_record`

| 表名 | 字段名 | 类型 | 说明 | 约束 |
|------|--------|------|------|------|
| ticket_process_record | id | bigint | 主键 | PK |
| ticket_process_record | ticket_id | bigint | 申告单 ID | NOT NULL, FK |
| ticket_process_record | handler_id | bigint | 处理人 | NOT NULL, FK |
| ticket_process_record | process_status | smallint | 处理状态 | NOT NULL |
| ticket_process_record | process_content | text | 处理过程 | NOT NULL |
| ticket_process_record | process_result | text | 处理结果 | NULL |
| ticket_process_record | requires_more_info | boolean | 是否需要补充信息 | NOT NULL, DEFAULT false |
| ticket_process_record | created_at | timestamptz | 创建时间 | NOT NULL |

### 2.14 申告回访记录表 `ticket_visit_record`

| 表名 | 字段名 | 类型 | 说明 | 约束 |
|------|--------|------|------|------|
| ticket_visit_record | id | bigint | 主键 | PK |
| ticket_visit_record | ticket_id | bigint | 申告单 ID | NOT NULL, FK |
| ticket_visit_record | visitor_id | bigint | 回访人 | NOT NULL, FK |
| ticket_visit_record | visit_result | smallint | 回访结果，1满意 2一般 3不满意 | NOT NULL |
| ticket_visit_record | visit_content | varchar(500) | 回访内容 | NULL |
| ticket_visit_record | created_at | timestamptz | 创建时间 | NOT NULL |

### 2.15 知识库表 `knowledge_base`

| 表名 | 字段名 | 类型 | 说明 | 约束 |
|------|--------|------|------|------|
| knowledge_base | id | bigint | 主键 | PK |
| knowledge_base | code | varchar(64) | 知识库编码 | NOT NULL, UNIQUE |
| knowledge_base | name | varchar(128) | 知识库名称 | NOT NULL |
| knowledge_base | description | varchar(500) | 知识库说明 | NULL |
| knowledge_base | embedding_model | varchar(128) | embedding 模型名称 | NOT NULL |
| knowledge_base | embedding_dimension | int | 向量维度 | NOT NULL, CHECK > 0 |
| knowledge_base | rag_provider | varchar(64) | RAG 服务标识，MVP 为 anythingllm | NOT NULL, DEFAULT 'anythingllm' |
| knowledge_base | status | smallint | 状态，1启用 2停用 | NOT NULL, DEFAULT 1 |
| knowledge_base | created_by | bigint | 创建人 | NULL, FK |
| knowledge_base | created_at | timestamptz | 创建时间 | NOT NULL |
| knowledge_base | updated_at | timestamptz | 更新时间 | NOT NULL |

### 2.16 知识分类表 `knowledge_category`

| 表名 | 字段名 | 类型 | 说明 | 约束 |
|------|--------|------|------|------|
| knowledge_category | id | bigint | 主键 | PK |
| knowledge_category | knowledge_base_id | bigint | 知识库 ID | NOT NULL, FK |
| knowledge_category | parent_id | bigint | 父分类 ID | NULL |
| knowledge_category | name | varchar(64) | 分类名称 | NOT NULL |
| knowledge_category | sort_order | int | 排序 | NOT NULL, DEFAULT 0 |
| knowledge_category | status | smallint | 状态，1启用 2停用 | NOT NULL, DEFAULT 1 |
| knowledge_category | created_at | timestamptz | 创建时间 | NOT NULL |
| knowledge_category | updated_at | timestamptz | 更新时间 | NOT NULL |

### 2.17 知识条目表 `knowledge_article`

| 表名 | 字段名 | 类型 | 说明 | 约束 |
|------|--------|------|------|------|
| knowledge_article | id | bigint | 主键 | PK |
| knowledge_article | knowledge_no | varchar(64) | 知识编号 | NOT NULL, UNIQUE |
| knowledge_article | knowledge_base_id | bigint | 知识库 ID | NOT NULL, FK |
| knowledge_article | category_id | bigint | 分类 ID | NOT NULL, FK |
| knowledge_article | title | varchar(255) | 问题标题 | NOT NULL |
| knowledge_article | question | text | FAQ 问题 | NOT NULL |
| knowledge_article | answer | text | FAQ 答案 | NOT NULL |
| knowledge_article | tags | jsonb | 标签列表 | NULL |
| knowledge_article | applicable_scope | varchar(255) | 适用范围 | NULL |
| knowledge_article | status | smallint | 状态，1草稿 2待审核 3已发布 4已停用 | NOT NULL, DEFAULT 1 |
| knowledge_article | review_status | smallint | 审核状态，1待审 2通过 3驳回 | NOT NULL, DEFAULT 1 |
| knowledge_article | published_at | timestamptz | 发布时间 | NULL |
| knowledge_article | maintainer_id | bigint | 维护人 | NULL, FK |
| knowledge_article | reviewer_id | bigint | 审核人 | NULL, FK |
| knowledge_article | embedding_model | varchar(128) | embedding 模型快照 | NOT NULL |
| knowledge_article | embedding_dimension | int | 向量维度快照 | NOT NULL, CHECK > 0 |
| knowledge_article | rag_sync_status | smallint | 同步状态，1未同步 2同步中 3成功 4失败 | NOT NULL, DEFAULT 1 |
| knowledge_article | rag_sync_error | varchar(500) | 同步错误 | NULL |
| knowledge_article | version_no | int | 版本号 | NOT NULL, DEFAULT 1 |
| knowledge_article | created_at | timestamptz | 创建时间 | NOT NULL |
| knowledge_article | updated_at | timestamptz | 更新时间 | NOT NULL |

### 2.18 知识切片向量表 `knowledge_chunk`

| 表名 | 字段名 | 类型 | 说明 | 约束 |
|------|--------|------|------|------|
| knowledge_chunk | id | bigint | 主键 | PK |
| knowledge_chunk | knowledge_base_id | bigint | 知识库 ID | NOT NULL, FK |
| knowledge_chunk | knowledge_id | bigint | 知识条目 ID | NOT NULL, FK |
| knowledge_chunk | chunk_no | int | 切片序号 | NOT NULL |
| knowledge_chunk | content | text | 切片内容 | NOT NULL |
| knowledge_chunk | embedding | vector | pgvector 向量字段 | NOT NULL |
| knowledge_chunk | embedding_model | varchar(128) | 向量模型名称 | NOT NULL |
| knowledge_chunk | embedding_dimension | int | 向量维度 | NOT NULL, CHECK > 0 |
| knowledge_chunk | token_count | int | Token 数量 | NULL |
| knowledge_chunk | metadata | jsonb | 来源、标签、段落等元数据 | NULL |
| knowledge_chunk | rag_provider | varchar(64) | RAG 服务标识，MVP 为 anythingllm | NOT NULL, DEFAULT 'anythingllm' |
| knowledge_chunk | status | smallint | 状态，1有效 2失效 | NOT NULL, DEFAULT 1 |
| knowledge_chunk | created_at | timestamptz | 创建时间 | NOT NULL |
| knowledge_chunk | updated_at | timestamptz | 更新时间 | NOT NULL |

### 2.19 知识审核记录表 `knowledge_review_record`

| 表名 | 字段名 | 类型 | 说明 | 约束 |
|------|--------|------|------|------|
| knowledge_review_record | id | bigint | 主键 | PK |
| knowledge_review_record | knowledge_id | bigint | 知识条目 ID | NOT NULL, FK |
| knowledge_review_record | reviewer_id | bigint | 审核人 | NOT NULL, FK |
| knowledge_review_record | review_result | smallint | 审核结果，1通过 2驳回 | NOT NULL |
| knowledge_review_record | review_comment | varchar(500) | 审核意见 | NULL |
| knowledge_review_record | created_at | timestamptz | 创建时间 | NOT NULL |

### 2.20 知识同步记录表 `knowledge_sync_record`

| 表名 | 字段名 | 类型 | 说明 | 约束 |
|------|--------|------|------|------|
| knowledge_sync_record | id | bigint | 主键 | PK |
| knowledge_sync_record | knowledge_id | bigint | 知识条目 ID | NOT NULL, FK |
| knowledge_sync_record | provider | varchar(64) | RAG 服务标识 | NOT NULL |
| knowledge_sync_record | event_id | varchar(128) | NATS 事件 ID | NULL |
| knowledge_sync_record | sync_status | smallint | 同步状态，1处理中 2成功 3失败 | NOT NULL |
| knowledge_sync_record | sync_payload | jsonb | 同步内容快照 | NULL |
| knowledge_sync_record | error_message | varchar(500) | 错误信息 | NULL |
| knowledge_sync_record | synced_at | timestamptz | 同步时间 | NULL |
| knowledge_sync_record | created_at | timestamptz | 创建时间 | NOT NULL |

### 2.21 知识候选表 `knowledge_candidate`

| 表名 | 字段名 | 类型 | 说明 | 约束 |
|------|--------|------|------|------|
| knowledge_candidate | id | bigint | 主键 | PK |
| knowledge_candidate | ticket_id | bigint | 来源申告单 ID | NOT NULL, UNIQUE, FK |
| knowledge_candidate | title | varchar(255) | 候选标题 | NOT NULL |
| knowledge_candidate | summary | text | 候选摘要 | NOT NULL |
| knowledge_candidate | status | smallint | 状态，1待审核 2已转知识 3已忽略 | NOT NULL, DEFAULT 1 |
| knowledge_candidate | created_by | bigint | 创建人 | NOT NULL, FK |
| knowledge_candidate | created_at | timestamptz | 创建时间 | NOT NULL |
| knowledge_candidate | updated_at | timestamptz | 更新时间 | NOT NULL |

### 2.22 系统配置表 `sys_config`

| 表名 | 字段名 | 类型 | 说明 | 约束 |
|------|--------|------|------|------|
| sys_config | id | bigint | 主键 | PK |
| sys_config | config_key | varchar(128) | 配置键 | NOT NULL, UNIQUE |
| sys_config | config_value | text | 配置值 | NOT NULL |
| sys_config | value_type | varchar(32) | 值类型 | NOT NULL |
| sys_config | remark | varchar(255) | 说明 | NULL |
| sys_config | updated_by | bigint | 更新人 | NULL, FK |
| sys_config | created_at | timestamptz | 创建时间 | NOT NULL |
| sys_config | updated_at | timestamptz | 更新时间 | NOT NULL |

### 2.23 附件表 `sys_file`

| 表名 | 字段名 | 类型 | 说明 | 约束 |
|------|--------|------|------|------|
| sys_file | id | bigint | 主键 | PK |
| sys_file | biz_type | varchar(64) | 业务类型 | NOT NULL |
| sys_file | biz_id | bigint | 业务 ID | NOT NULL |
| sys_file | file_name | varchar(255) | 文件名 | NOT NULL |
| sys_file | file_path | varchar(500) | 存储路径 | NOT NULL |
| sys_file | file_size | bigint | 文件大小 | NOT NULL |
| sys_file | mime_type | varchar(128) | 文件类型 | NULL |
| sys_file | storage_provider | varchar(64) | 存储服务 | NOT NULL |
| sys_file | uploaded_by | bigint | 上传人 | NULL, FK |
| sys_file | created_at | timestamptz | 创建时间 | NOT NULL |

## 3. 索引设计

| 表名 | 索引类型 | 字段 | 用途 |
|------|----------|------|------|
| sys_user | UNIQUE | username | 登录账号唯一性校验 |
| sys_user | INDEX | status | 快速筛选正常/冻结账号 |
| sys_user_role | UNIQUE | user_id, role_id | 防止重复授权 |
| sys_role | UNIQUE | code | 角色编码唯一 |
| sys_permission | UNIQUE | code | 权限标识唯一 |
| sys_permission | INDEX | parent_id, sort_order | 菜单树加载 |
| sys_role_permission | UNIQUE | role_id, permission_id | 防止重复授权 |
| sys_login_log | INDEX | username, login_at | 登录审计查询 |
| sys_operation_log | INDEX | module, created_at | 按模块查询操作日志 |
| audit_log | INDEX | biz_type, biz_id | 业务审计追踪 |
| portal_chat_session | UNIQUE | session_no | 会话编号查询 |
| portal_chat_session | INDEX | user_id, created_at | 用户历史问答查询 |
| portal_chat_session | INDEX | model_provider, rag_provider | 模型/RAG 调用统计 |
| portal_chat_message | INDEX | session_id, created_at | 会话消息回放 |
| portal_chat_feedback | INDEX | session_id | 反馈关联查询 |
| ticket | UNIQUE | ticket_no | 申告编号查询 |
| ticket | INDEX | status, urgency_level, created_at | 待办列表筛选 |
| ticket | INDEX | assignee_id, status | 处理人任务列表 |
| ticket_process_record | INDEX | ticket_id, created_at | 处理流转追踪 |
| ticket_visit_record | INDEX | ticket_id, created_at | 回访记录查询 |
| knowledge_base | UNIQUE | code | 知识库编码唯一 |
| knowledge_base | UNIQUE | id, embedding_model, embedding_dimension | 支撑同知识库模型维度一致性外键 |
| knowledge_category | UNIQUE | knowledge_base_id, name | 同一知识库内分类唯一 |
| knowledge_article | UNIQUE | knowledge_no | 知识条目唯一性 |
| knowledge_article | UNIQUE | id, knowledge_base_id, embedding_model, embedding_dimension | 支撑切片模型维度一致性外键 |
| knowledge_article | INDEX | category_id, status, updated_at | 知识库列表筛选 |
| knowledge_article | INDEX | knowledge_base_id, status, updated_at | 按知识库筛选知识 |
| knowledge_article | INDEX | review_status, rag_sync_status | 审核与同步状态查询 |
| knowledge_chunk | UNIQUE | knowledge_id, chunk_no | 同一知识条目切片序号唯一 |
| knowledge_chunk | INDEX | knowledge_base_id, knowledge_id, chunk_no | 知识切片回放和重建 |
| knowledge_chunk | VECTOR | embedding | pgvector 向量相似度检索 |
| knowledge_review_record | INDEX | knowledge_id, created_at | 审核历史查询 |
| knowledge_sync_record | INDEX | knowledge_id, created_at | 同步记录查询 |
| knowledge_sync_record | INDEX | provider, sync_status, created_at | AnythingLLM 同步状态查询 |
| knowledge_candidate | UNIQUE | ticket_id | 一个申告只生成一个候选 |
| sys_config | UNIQUE | config_key | 配置键查询 |
| sys_file | INDEX | biz_type, biz_id | 附件按业务归档 |

### 3.1 全文/向量索引

MVP 启用 PostgreSQL pgvector，并在数据库层约束同知识库同模型同维度：

- `knowledge_article.question`、`knowledge_article.answer` 建议建立 `GIN` 全文索引，支持关键词检索。
- `knowledge_chunk.embedding` 建议建立 `hnsw` 向量索引；数据量较小时也可先使用精确检索，待切片规模增长后再创建近似索引。
- `knowledge_base.embedding_model` 和 `knowledge_base.embedding_dimension` 为知识库级必填配置，创建知识库时必须从系统配置的可用模型和维度中选择。
- `knowledge_article` 保存知识库 embedding 配置快照，并通过外键约束与 `knowledge_base(id, embedding_model, embedding_dimension)` 一致。
- `knowledge_chunk` 保存切片 embedding 配置，并通过外键约束与 `knowledge_article(id, knowledge_base_id, embedding_model, embedding_dimension)` 一致。
- `knowledge_chunk.embedding` 使用 pgvector `vector` 类型，并添加 `CHECK (vector_dims(embedding) = embedding_dimension)`，确保实际向量维度等于知识库配置维度。

约束示例：

```sql
ALTER TABLE knowledge_base
  ADD CONSTRAINT uk_kb_embedding_config
  UNIQUE (id, embedding_model, embedding_dimension);

ALTER TABLE knowledge_article
  ADD CONSTRAINT uk_article_embedding_config
  UNIQUE (id, knowledge_base_id, embedding_model, embedding_dimension);

ALTER TABLE knowledge_article
  ADD CONSTRAINT fk_article_kb_embedding_config
  FOREIGN KEY (knowledge_base_id, embedding_model, embedding_dimension)
  REFERENCES knowledge_base (id, embedding_model, embedding_dimension);

ALTER TABLE knowledge_chunk
  ADD CONSTRAINT fk_chunk_article_embedding_config
  FOREIGN KEY (knowledge_id, knowledge_base_id, embedding_model, embedding_dimension)
  REFERENCES knowledge_article (id, knowledge_base_id, embedding_model, embedding_dimension);

ALTER TABLE knowledge_chunk
  ADD CONSTRAINT ck_chunk_embedding_dimension
  CHECK (vector_dims(embedding) = embedding_dimension);
```

## 4. 分库分表策略（如需要）

MVP 阶段不建议分库分表。

原因：

- 业务范围以单企业内部使用为主，数据量预计处于中小规模。
- 核心链路包含事务一致性要求，申告、知识审核、审计日志都需要跨表关联。
- PostgreSQL 单库足以覆盖 MVP 的查询、事务、审计、知识元数据和 pgvector 向量存储。

后续扩展建议：

- `audit_log`、`sys_operation_log`、`sys_login_log` 可按月分区，降低历史日志查询和归档成本。
- `portal_chat_session`、`portal_chat_message`、`ticket` 可按创建时间做时间分区，便于冷热数据拆分。
- 当知识库规模明显增长时，再把向量检索迁出到 Qdrant 等独立向量服务，不建议在 MVP 阶段拆库。

## 5. 数据迁移方案（如需要）

建议使用版本化迁移，不直接依赖手工建表。

### 5.1 迁移原则

- 所有表结构通过迁移脚本创建和修改。
- 主数据初始化与结构迁移分离。
- 每次迁移保持可回滚。
- 迁移脚本需要与后端版本同步管理。

### 5.2 迁移顺序建议

1. 先建基础系统表：`sys_user`、`sys_role`、`sys_permission`、关联表、日志表、配置表。
2. 启用 PostgreSQL 扩展：`vector`。
3. 再建门户与申告表：`portal_chat_session`、`portal_chat_message`、`portal_chat_feedback`、`ticket`、处理记录、回访记录。
4. 再建知识库表：`knowledge_base`、`knowledge_category`、`knowledge_article`、`knowledge_chunk`、`knowledge_review_record`、`knowledge_sync_record`、`knowledge_candidate`。
5. 最后补充附件表和初始化字典数据。

### 5.3 回滚策略

- 每个 `up` 脚本都应配套 `down` 脚本。
- 删除字段或表时，先确认后端代码已完成兼容切换。
- 生产环境回滚优先使用反向迁移，不直接手工改库。

### 5.4 初始化数据

建议初始化以下基础数据：

- 默认管理员角色。
- 默认系统管理员账号。
- 基础权限菜单树。
- 申告状态字典、紧急程度字典、知识状态字典、反馈字典。
- 默认系统配置：vLLM OpenAI-compatible 地址、AnythingLLM 地址、MinIO bucket、知识同步 NATS subject。
- 默认知识库配置：embedding 模型名称、向量维度、AnythingLLM 工作区或知识库标识。
