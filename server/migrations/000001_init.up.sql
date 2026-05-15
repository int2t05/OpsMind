-- OpsMind M1 初始化迁移：23 张核心表 + pgvector + 索引 + 种子数据
-- 依赖顺序：系统表 → 扩展 → 门户/申告 → 知识库 → 附件 → 种子数据

BEGIN;

-- ============ pgvector 扩展 ============
CREATE EXTENSION IF NOT EXISTS vector;

-- ============ 1. 系统用户表 ============
CREATE TABLE sys_user (
    id              BIGSERIAL PRIMARY KEY,
    username        VARCHAR(64)  NOT NULL,
    password_hash   VARCHAR(255) NOT NULL,
    real_name       VARCHAR(64)  NOT NULL,
    phone           VARCHAR(32),
    email           VARCHAR(128),
    status          SMALLINT     NOT NULL DEFAULT 1, -- 1正常 2冻结
    last_login_at   TIMESTAMPTZ,
    last_login_ip   VARCHAR(64),
    remark          VARCHAR(255),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);
CREATE UNIQUE INDEX uk_sys_user_username ON sys_user(username);
CREATE INDEX idx_sys_user_status ON sys_user(status);

-- ============ 2. 角色表 ============
CREATE TABLE sys_role (
    id         BIGSERIAL PRIMARY KEY,
    code       VARCHAR(64)  NOT NULL,
    name       VARCHAR(64)  NOT NULL,
    status     SMALLINT     NOT NULL DEFAULT 1, -- 1启用 2停用
    remark     VARCHAR(255),
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX uk_sys_role_code ON sys_role(code);

-- ============ 3. 权限表 ============
CREATE TABLE sys_permission (
    id         BIGSERIAL PRIMARY KEY,
    parent_id  BIGINT,
    type       SMALLINT     NOT NULL, -- 1菜单 2按钮 3接口
    name       VARCHAR(64)  NOT NULL,
    code       VARCHAR(128) NOT NULL,
    path       VARCHAR(255),
    method     VARCHAR(16),
    sort_order INT          NOT NULL DEFAULT 0,
    visible    BOOLEAN      NOT NULL DEFAULT TRUE,
    status     SMALLINT     NOT NULL DEFAULT 1, -- 1启用 2停用
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX uk_sys_permission_code ON sys_permission(code);
CREATE INDEX idx_sys_permission_tree ON sys_permission(parent_id, sort_order);

-- ============ 4. 用户角色关联表 ============
CREATE TABLE sys_user_role (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT       NOT NULL REFERENCES sys_user(id),
    role_id    BIGINT       NOT NULL REFERENCES sys_role(id),
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX uk_sys_user_role ON sys_user_role(user_id, role_id);

-- ============ 5. 角色权限关联表 ============
CREATE TABLE sys_role_permission (
    id            BIGSERIAL PRIMARY KEY,
    role_id       BIGINT       NOT NULL REFERENCES sys_role(id),
    permission_id BIGINT       NOT NULL REFERENCES sys_permission(id),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX uk_sys_role_permission ON sys_role_permission(role_id, permission_id);

-- ============ 6. 登录日志表 ============
CREATE TABLE sys_login_log (
    id           BIGSERIAL PRIMARY KEY,
    user_id      BIGINT       REFERENCES sys_user(id),
    username     VARCHAR(64)  NOT NULL,
    login_result SMALLINT     NOT NULL, -- 1成功 2失败
    fail_reason  VARCHAR(255),
    ip_address   VARCHAR(64)  NOT NULL,
    user_agent   VARCHAR(255),
    login_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_sys_login_log_query ON sys_login_log(username, login_at);

-- ============ 7. 操作日志表 ============
CREATE TABLE sys_operation_log (
    id             BIGSERIAL PRIMARY KEY,
    user_id        BIGINT       REFERENCES sys_user(id),
    module         VARCHAR(64)  NOT NULL,
    action         VARCHAR(64)  NOT NULL,
    target_type    VARCHAR(64),
    target_id      BIGINT,
    request_path   VARCHAR(255) NOT NULL,
    request_method VARCHAR(16)  NOT NULL,
    request_body   JSONB,
    response_code  INT          NOT NULL,
    success        BOOLEAN      NOT NULL,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_sys_operation_log_module ON sys_operation_log(module, created_at);

-- ============ 8. 审计日志表 ============
CREATE TABLE audit_log (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT       REFERENCES sys_user(id),
    module      VARCHAR(64)  NOT NULL,
    action      VARCHAR(64)  NOT NULL,
    biz_type    VARCHAR(64)  NOT NULL,
    biz_id      BIGINT,
    before_data JSONB,
    after_data  JSONB,
    ip_address  VARCHAR(64),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_audit_log_biz ON audit_log(biz_type, biz_id);

-- ============ 9. 门户问答会话表 ============
CREATE TABLE portal_chat_session (
    id               BIGSERIAL PRIMARY KEY,
    session_no       VARCHAR(64)  NOT NULL,
    user_id          BIGINT       REFERENCES sys_user(id),
    question         TEXT         NOT NULL,
    answer           TEXT,
    answer_source    JSONB,
    confidence_score NUMERIC(5,2),
    model_name       VARCHAR(128),
    model_provider   VARCHAR(64)  NOT NULL DEFAULT 'vllm',
    rag_provider     VARCHAR(64),
    status           SMALLINT     NOT NULL DEFAULT 1, -- 1处理中 2已完成 3转人工 4失败
    answered_at      TIMESTAMPTZ,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX uk_portal_chat_session_no ON portal_chat_session(session_no);
CREATE INDEX idx_portal_chat_session_user ON portal_chat_session(user_id, created_at);
CREATE INDEX idx_portal_chat_session_provider ON portal_chat_session(model_provider, rag_provider);

-- ============ 10. 门户问答消息表 ============
CREATE TABLE portal_chat_message (
    id          BIGSERIAL PRIMARY KEY,
    session_id  BIGINT       NOT NULL REFERENCES portal_chat_session(id),
    role        VARCHAR(16)  NOT NULL, -- user/assistant/system
    content     TEXT         NOT NULL,
    token_count INT,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_portal_chat_message_session ON portal_chat_message(session_id, created_at);

-- ============ 11. 问答反馈表 ============
CREATE TABLE portal_chat_feedback (
    id            BIGSERIAL PRIMARY KEY,
    session_id    BIGINT       NOT NULL REFERENCES portal_chat_session(id),
    user_id       BIGINT       REFERENCES sys_user(id),
    feedback_type SMALLINT     NOT NULL, -- 1已解决 2未解决
    remark        VARCHAR(255),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_portal_chat_feedback_session ON portal_chat_feedback(session_id);

-- ============ 12. 申告单表 ============
CREATE TABLE ticket (
    id               BIGSERIAL PRIMARY KEY,
    ticket_no        VARCHAR(64)  NOT NULL,
    source_session_id BIGINT      REFERENCES portal_chat_session(id),
    source_feedback_id BIGINT     REFERENCES portal_chat_feedback(id),
    reporter_name    VARCHAR(64)  NOT NULL,
    reporter_phone   VARCHAR(32)  NOT NULL,
    title            VARCHAR(255) NOT NULL,
    description      TEXT         NOT NULL,
    impact_scope     VARCHAR(255),
    urgency_level    SMALLINT     NOT NULL, -- 1低 2中 3高
    status           SMALLINT     NOT NULL DEFAULT 1, -- 1待处理 2处理中 3待补充 4已完成 5已关闭
    assignee_id      BIGINT       REFERENCES sys_user(id),
    ai_context       JSONB,
    attachment_count INT          NOT NULL DEFAULT 0,
    closed_reason    VARCHAR(255),
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX uk_ticket_no ON ticket(ticket_no);
CREATE INDEX idx_ticket_status ON ticket(status, urgency_level, created_at);
CREATE INDEX idx_ticket_assignee ON ticket(assignee_id, status);

-- ============ 13. 申告处理记录表 ============
CREATE TABLE ticket_process_record (
    id                  BIGSERIAL PRIMARY KEY,
    ticket_id           BIGINT       NOT NULL REFERENCES ticket(id),
    handler_id          BIGINT       NOT NULL REFERENCES sys_user(id),
    process_status      SMALLINT     NOT NULL,
    process_content     TEXT         NOT NULL,
    process_result      TEXT,
    requires_more_info  BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_ticket_process_record ON ticket_process_record(ticket_id, created_at);

-- ============ 14. 申告回访记录表 ============
CREATE TABLE ticket_visit_record (
    id           BIGSERIAL PRIMARY KEY,
    ticket_id    BIGINT       NOT NULL REFERENCES ticket(id),
    visitor_id   BIGINT       NOT NULL REFERENCES sys_user(id),
    visit_result SMALLINT     NOT NULL, -- 1满意 2一般 3不满意
    visit_content VARCHAR(500),
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_ticket_visit_record ON ticket_visit_record(ticket_id, created_at);

-- ============ 15. 知识库表 ============
CREATE TABLE knowledge_base (
    id                  BIGSERIAL PRIMARY KEY,
    code                VARCHAR(64)  NOT NULL,
    name                VARCHAR(128) NOT NULL,
    description         VARCHAR(500),
    embedding_model     VARCHAR(128) NOT NULL,
    embedding_dimension INT          NOT NULL CHECK (embedding_dimension > 0),
    rag_provider        VARCHAR(64)  NOT NULL DEFAULT 'anythingllm',
    status              SMALLINT     NOT NULL DEFAULT 1, -- 1启用 2停用
    created_by          BIGINT       REFERENCES sys_user(id),
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX uk_knowledge_base_code ON knowledge_base(code);
-- 支撑 embedding 配置一致性外键
CREATE UNIQUE INDEX uk_kb_embedding_config ON knowledge_base(id, embedding_model, embedding_dimension);

-- ============ 16. 知识分类表 ============
CREATE TABLE knowledge_category (
    id                BIGSERIAL PRIMARY KEY,
    knowledge_base_id BIGINT       NOT NULL REFERENCES knowledge_base(id),
    parent_id         BIGINT,
    name              VARCHAR(64)  NOT NULL,
    sort_order        INT          NOT NULL DEFAULT 0,
    status            SMALLINT     NOT NULL DEFAULT 1,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX uk_knowledge_category ON knowledge_category(knowledge_base_id, name);

-- ============ 17. 知识条目表 ============
CREATE TABLE knowledge_article (
    id                  BIGSERIAL PRIMARY KEY,
    knowledge_no        VARCHAR(64)  NOT NULL,
    knowledge_base_id   BIGINT       NOT NULL REFERENCES knowledge_base(id),
    category_id         BIGINT       NOT NULL REFERENCES knowledge_category(id),
    title               VARCHAR(255) NOT NULL,
    question            TEXT         NOT NULL,
    answer              TEXT         NOT NULL,
    tags                JSONB,
    applicable_scope    VARCHAR(255),
    status              SMALLINT     NOT NULL DEFAULT 1, -- 1草稿 2待审核 3已发布 4已停用
    review_status       SMALLINT     NOT NULL DEFAULT 1, -- 1待审 2通过 3驳回
    published_at        TIMESTAMPTZ,
    maintainer_id       BIGINT       REFERENCES sys_user(id),
    reviewer_id         BIGINT       REFERENCES sys_user(id),
    embedding_model     VARCHAR(128) NOT NULL,
    embedding_dimension INT          NOT NULL CHECK (embedding_dimension > 0),
    rag_sync_status     SMALLINT     NOT NULL DEFAULT 1, -- 1未同步 2同步中 3成功 4失败
    rag_sync_error      VARCHAR(500),
    version_no          INT          NOT NULL DEFAULT 1,
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX uk_knowledge_article_no ON knowledge_article(knowledge_no);
CREATE INDEX idx_knowledge_article_category ON knowledge_article(category_id, status, updated_at);
CREATE INDEX idx_knowledge_article_kb ON knowledge_article(knowledge_base_id, status, updated_at);
CREATE INDEX idx_knowledge_article_sync ON knowledge_article(review_status, rag_sync_status);
-- 支撑切片 embedding 配置一致性外键
CREATE UNIQUE INDEX uk_article_embedding_config ON knowledge_article(id, knowledge_base_id, embedding_model, embedding_dimension);

-- embedding 配置一致性外键
ALTER TABLE knowledge_article
  ADD CONSTRAINT fk_article_kb_embedding
  FOREIGN KEY (knowledge_base_id, embedding_model, embedding_dimension)
  REFERENCES knowledge_base (id, embedding_model, embedding_dimension);

-- ============ 18. 知识切片向量表 ============
CREATE TABLE knowledge_chunk (
    id                  BIGSERIAL PRIMARY KEY,
    knowledge_base_id   BIGINT      NOT NULL REFERENCES knowledge_base(id),
    knowledge_id        BIGINT      NOT NULL REFERENCES knowledge_article(id),
    chunk_no            INT         NOT NULL,
    content             TEXT        NOT NULL,
    embedding           vector      NOT NULL,
    embedding_model     VARCHAR(128) NOT NULL,
    embedding_dimension INT         NOT NULL CHECK (embedding_dimension > 0),
    token_count         INT,
    metadata            JSONB,
    rag_provider        VARCHAR(64) NOT NULL DEFAULT 'anythingllm',
    status              SMALLINT    NOT NULL DEFAULT 1, -- 1有效 2失效
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX uk_knowledge_chunk ON knowledge_chunk(knowledge_id, chunk_no);
CREATE INDEX idx_knowledge_chunk_list ON knowledge_chunk(knowledge_base_id, knowledge_id, chunk_no);

-- pgvector embedding 配置一致性外键
ALTER TABLE knowledge_chunk
  ADD CONSTRAINT fk_chunk_article_embedding
  FOREIGN KEY (knowledge_id, knowledge_base_id, embedding_model, embedding_dimension)
  REFERENCES knowledge_article (id, knowledge_base_id, embedding_model, embedding_dimension);

-- 向量维度校验
ALTER TABLE knowledge_chunk
  ADD CONSTRAINT ck_chunk_embedding_dimension
  CHECK (vector_dims(embedding) = embedding_dimension);

-- ============ 19. 知识审核记录表 ============
CREATE TABLE knowledge_review_record (
    id             BIGSERIAL PRIMARY KEY,
    knowledge_id   BIGINT       NOT NULL REFERENCES knowledge_article(id),
    reviewer_id    BIGINT       NOT NULL REFERENCES sys_user(id),
    review_result  SMALLINT     NOT NULL, -- 1通过 2驳回
    review_comment VARCHAR(500),
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_knowledge_review_record ON knowledge_review_record(knowledge_id, created_at);

-- ============ 20. 知识同步记录表 ============
CREATE TABLE knowledge_sync_record (
    id            BIGSERIAL PRIMARY KEY,
    knowledge_id  BIGINT       NOT NULL REFERENCES knowledge_article(id),
    provider      VARCHAR(64)  NOT NULL,
    event_id      VARCHAR(128),
    sync_status   SMALLINT     NOT NULL, -- 1处理中 2成功 3失败
    sync_payload  JSONB,
    error_message VARCHAR(500),
    synced_at     TIMESTAMPTZ,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_knowledge_sync_record_kid ON knowledge_sync_record(knowledge_id, created_at);
CREATE INDEX idx_knowledge_sync_record_provider ON knowledge_sync_record(provider, sync_status, created_at);

-- ============ 21. 知识候选表 ============
CREATE TABLE knowledge_candidate (
    id         BIGSERIAL PRIMARY KEY,
    ticket_id  BIGINT       NOT NULL REFERENCES ticket(id),
    title      VARCHAR(255) NOT NULL,
    summary    TEXT         NOT NULL,
    status     SMALLINT     NOT NULL DEFAULT 1, -- 1待审核 2已转知识 3已忽略
    created_by BIGINT       NOT NULL REFERENCES sys_user(id),
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX uk_knowledge_candidate_ticket ON knowledge_candidate(ticket_id);

-- ============ 22. 系统配置表 ============
CREATE TABLE sys_config (
    id           BIGSERIAL PRIMARY KEY,
    config_key   VARCHAR(128) NOT NULL,
    config_value TEXT         NOT NULL,
    value_type   VARCHAR(32)  NOT NULL,
    remark       VARCHAR(255),
    updated_by   BIGINT       REFERENCES sys_user(id),
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX uk_sys_config_key ON sys_config(config_key);

-- ============ 23. 附件表 ============
CREATE TABLE sys_file (
    id               BIGSERIAL PRIMARY KEY,
    biz_type         VARCHAR(64)  NOT NULL,
    biz_id           BIGINT       NOT NULL DEFAULT 0,
    file_name        VARCHAR(255) NOT NULL,
    file_path        VARCHAR(500) NOT NULL,
    file_size        BIGINT       NOT NULL,
    mime_type        VARCHAR(128),
    storage_provider VARCHAR(64)  NOT NULL DEFAULT 'minio',
    uploaded_by      BIGINT       REFERENCES sys_user(id),
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_sys_file_biz ON sys_file(biz_type, biz_id);

-- ============ 种子数据 ============

-- 系统管理员角色
INSERT INTO sys_role (code, name, remark) VALUES
  ('admin', '系统管理员', '拥有全部权限'),
  ('operator', '运维人员', '负责申告处理、知识维护'),
  ('viewer', '浏览者', '仅查看看板和日志');

-- 基础权限菜单树
-- 一级菜单
INSERT INTO sys_permission (id, parent_id, type, name, code, path, method, sort_order) VALUES
  (1,  NULL, 1, '工作台',     'dashboard',           '/admin/dashboard',          NULL, 1),
  (2,  NULL, 1, '申告管理',   'ticket',              '/admin/tickets',            NULL, 2),
  (3,  NULL, 1, '知识库',     'knowledge',           '/admin/knowledge',          NULL, 3),
  (4,  NULL, 1, '账号管理',   'account',             '/admin/accounts',           NULL, 4),
  (5,  NULL, 1, '系统配置',   'config',              '/admin/configs',            NULL, 5),
  (6,  NULL, 1, '日志审计',   'audit',               '/admin/audit',              NULL, 6),
  -- 账号管理子权限
  (7,  4, 2, '账号列表',      'account:list',        NULL,                        NULL, 1),
  (8,  4, 2, '创建账号',      'account:create',      NULL,                        NULL, 2),
  (9,  4, 2, '账号详情',      'account:detail',      NULL,                        NULL, 3),
  (10, 4, 2, '编辑账号',      'account:update',      NULL,                        NULL, 4),
  (11, 4, 2, '冻结账号',      'account:freeze',      NULL,                        NULL, 5),
  (12, 4, 2, '恢复账号',      'account:restore',     NULL,                        NULL, 6),
  -- 角色权限子权限
  (13, 4, 2, '角色列表',      'role:list',           NULL,                        NULL, 7),
  (14, 4, 2, '权限列表',      'permission:list',     NULL,                        NULL, 8),
  (15, 4, 2, '角色权限更新',  'role:permission:update', NULL,                     NULL, 9);

-- 管理员角色授予全部权限
INSERT INTO sys_role_permission (role_id, permission_id)
  SELECT (SELECT id FROM sys_role WHERE code='admin'), id FROM sys_permission;

-- 操作员角色授予申告和知识权限
INSERT INTO sys_role_permission (role_id, permission_id)
  SELECT (SELECT id FROM sys_role WHERE code='operator'), id
  FROM sys_permission WHERE code IN ('dashboard', 'ticket', 'knowledge');

-- 默认管理员: admin / admin123 (bcrypt hash)
-- 密码哈希由后端启动时动态生成，此处占位
INSERT INTO sys_user (username, password_hash, real_name, status) VALUES
  ('admin', '$2a$10$PLACEHOLDER_WILL_BE_REPLACED_BY_BOOTSTRAP', '系统管理员', 1);

-- 管理员角色绑定
INSERT INTO sys_user_role (user_id, role_id)
  SELECT u.id, r.id FROM sys_user u, sys_role r WHERE u.username='admin' AND r.code='admin';

-- 默认系统配置
INSERT INTO sys_config (config_key, config_value, value_type, remark) VALUES
  ('vllm.base_url',    'http://localhost:8000', 'string', 'vLLM OpenAI-compatible 地址'),
  ('vllm.api_key',     '',                      'string', 'vLLM API Key'),
  ('vllm.model',       'default',               'string', 'vLLM 模型名称'),
  ('anythingllm.base_url', 'http://localhost:3001', 'string', 'AnythingLLM 地址'),
  ('anythingllm.api_key',  '',                      'string', 'AnythingLLM API Key'),
  ('minio.bucket',     'opsmind',               'string', 'MinIO 默认 bucket'),
  ('nats.sync_subject','knowledge.published',   'string', '知识同步事件 subject'),
  ('embedding.default_model',     'text-embedding-ada-002', 'string', '默认 embedding 模型'),
  ('embedding.default_dimension', '1536',                 'int',    '默认向量维度');

COMMIT;
