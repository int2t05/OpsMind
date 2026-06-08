//go:build integration

// Package repository_test 验证 KnowledgeRepo 数据访问层。
//
// 测试覆盖 TECH.md §9.2 定义的 KnowledgeBase、KnowledgeArticle、KnowledgeChunk、
// EmbeddingConfig 的全部数据访问方法。
// 使用独立的 opsmind_test 数据库，每个测试用例通过清理保证隔离性。
package repository_test

import (
	"testing"
	"time"

	"opsmind/internal/config"
	"opsmind/internal/database"
	"opsmind/internal/model"
	"opsmind/internal/repository"

	"github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// setupKnowledgeTestDB 初始化测试数据库连接并确保知识库相关表存在。
func setupKnowledgeTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbCfg := config.DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "opsmind",
		Password: "opsmind123",
		DBName:   "opsmind_test",
		SSLMode:  "disable",
	}

	db, err := database.Init(dbCfg)
	if err != nil {
		t.Fatalf("初始化数据库失败: %v", err)
	}

	// 按依赖顺序创建表
	db.Exec(`CREATE TABLE IF NOT EXISTS knowledge_bases (
		id BIGSERIAL PRIMARY KEY,
		name VARCHAR(128) NOT NULL,
		description TEXT,
		rag_workspace_slug VARCHAR(128) UNIQUE,
		embedding_model VARCHAR(128) NOT NULL DEFAULT '',
		vector_dimension INTEGER NOT NULL DEFAULT 0,
		created_by BIGINT NOT NULL DEFAULT 0,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS knowledge_articles (
		id BIGSERIAL PRIMARY KEY,
		kb_id BIGINT NOT NULL,
		question TEXT NOT NULL,
		answer TEXT NOT NULL,
		category VARCHAR(64),
		tags JSONB,
		status SMALLINT NOT NULL DEFAULT 1,
		created_by BIGINT NOT NULL DEFAULT 0,
		reviewed_by BIGINT,
		published_by BIGINT,
		review_comment TEXT,
		rag_document_location VARCHAR(512),
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS knowledge_chunks (
		id BIGSERIAL PRIMARY KEY,
		article_id BIGINT NOT NULL,
		content TEXT NOT NULL,
		embedding vector,
		embedding_model VARCHAR(128) NOT NULL DEFAULT '',
		vector_dimension INTEGER NOT NULL DEFAULT 0,
		sync_status VARCHAR(16) NOT NULL DEFAULT 'pending',
		sync_error TEXT,
		synced_at TIMESTAMPTZ,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS embedding_configs (
		id BIGSERIAL PRIMARY KEY,
		name VARCHAR(128) NOT NULL,
		model_type SMALLINT NOT NULL DEFAULT 1,
		api_endpoint VARCHAR(512),
		api_key VARCHAR(512),
		local_path VARCHAR(512),
		vector_dimension INTEGER NOT NULL DEFAULT 0,
		is_default BOOLEAN NOT NULL DEFAULT FALSE,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`)

	return db
}

// cleanKnowledgeTables 清理测试数据
func cleanKnowledgeTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	db.Exec("DELETE FROM knowledge_chunks")
	db.Exec("DELETE FROM knowledge_articles")
	db.Exec("DELETE FROM knowledge_bases")
	db.Exec("DELETE FROM embedding_configs")
}

// testVector 返回一个用于测试的有效 pgvector 向量。
//
// pgvector 要求向量至少 1 维，空切片会被 PostgreSQL 拒绝。
func testVector(dim int) pgvector.Vector {
	vec := make([]float32, dim)
	for i := range vec {
		vec[i] = float32(i+1) * 0.01
	}
	return pgvector.NewVector(vec)
}

// =============================================================================
// KnowledgeBase 测试
// =============================================================================

// TestKnowledgeRepo_CreateKB 创建知识库
func TestKnowledgeRepo_CreateKB(t *testing.T) {
	db := setupKnowledgeTestDB(t)
	cleanKnowledgeTables(t, db)
	repo := repository.NewKnowledgeRepo(db)

	now := time.Now()
	kb := &model.KnowledgeBase{
		Name:             "测试知识库",
		Description:      "用于测试的知识库",
		RAGWorkspaceSlug: "test-workspace-slug",
		EmbeddingModel:   "text-embedding-3-small",
		VectorDimension:  1536,
		CreatedBy:        1,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	err := repo.CreateKB(kb)
	require.NoError(t, err)
	assert.NotZero(t, kb.ID, "创建后应自动填充 ID")

	// 验证可查回
	got, err := repo.FindKBByID(kb.ID)
	require.NoError(t, err)
	assert.Equal(t, "测试知识库", got.Name)
	assert.Equal(t, "用于测试的知识库", got.Description)
	assert.Equal(t, "test-workspace-slug", got.RAGWorkspaceSlug)
	assert.Equal(t, "text-embedding-3-small", got.EmbeddingModel)
	assert.Equal(t, 1536, got.VectorDimension)
}

// TestKnowledgeRepo_FindKBByID_NotFound 查询不存在的知识库
func TestKnowledgeRepo_FindKBByID_NotFound(t *testing.T) {
	db := setupKnowledgeTestDB(t)
	repo := repository.NewKnowledgeRepo(db)

	got, err := repo.FindKBByID(999999)
	assert.Error(t, err)
	assert.Nil(t, got)
	assert.Equal(t, gorm.ErrRecordNotFound, err)
}

// TestKnowledgeRepo_UpdateKB 更新知识库
func TestKnowledgeRepo_UpdateKB(t *testing.T) {
	db := setupKnowledgeTestDB(t)
	cleanKnowledgeTables(t, db)
	repo := repository.NewKnowledgeRepo(db)

	now := time.Now()
	kb := &model.KnowledgeBase{
		Name:             "旧名称",
		Description:      "旧描述",
		RAGWorkspaceSlug: "old-slug",
		EmbeddingModel:   "old-model",
		VectorDimension:  768,
		CreatedBy:        1,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	require.NoError(t, db.Create(kb).Error)

	kb.Name = "新名称"
	kb.Description = "新描述"
	kb.UpdatedAt = time.Now()

	err := repo.UpdateKB(kb)
	require.NoError(t, err)

	got, err := repo.FindKBByID(kb.ID)
	require.NoError(t, err)
	assert.Equal(t, "新名称", got.Name)
	assert.Equal(t, "新描述", got.Description)
}

// TestKnowledgeRepo_ListKBs 列出全部知识库
func TestKnowledgeRepo_ListKBs(t *testing.T) {
	db := setupKnowledgeTestDB(t)
	cleanKnowledgeTables(t, db)
	repo := repository.NewKnowledgeRepo(db)

	now := time.Now()
	kb1 := &model.KnowledgeBase{
		Name:             "知识库1",
		RAGWorkspaceSlug: "slug-1",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	kb2 := &model.KnowledgeBase{
		Name:             "知识库2",
		RAGWorkspaceSlug: "slug-2",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	require.NoError(t, db.Create(kb1).Error)
	require.NoError(t, db.Create(kb2).Error)

	kbs, err := repo.ListKBs()
	require.NoError(t, err)
	assert.Len(t, kbs, 2)
	assert.True(t, kbs[0].Name == "知识库1" || kbs[0].Name == "知识库2")
}

// TestKnowledgeRepo_ListKBs_Empty 无知识库时返回空切片
func TestKnowledgeRepo_ListKBs_Empty(t *testing.T) {
	db := setupKnowledgeTestDB(t)
	cleanKnowledgeTables(t, db)
	repo := repository.NewKnowledgeRepo(db)

	kbs, err := repo.ListKBs()
	require.NoError(t, err)
	assert.Empty(t, kbs)
}

// =============================================================================
// KnowledgeArticle 测试
// =============================================================================

// TestKnowledgeRepo_CreateArticle 创建知识文章
func TestKnowledgeRepo_CreateArticle(t *testing.T) {
	db := setupKnowledgeTestDB(t)
	cleanKnowledgeTables(t, db)
	repo := repository.NewKnowledgeRepo(db)

	now := time.Now()
	kb := &model.KnowledgeBase{
		Name:             "文章测试库",
		RAGWorkspaceSlug: "article-test-slug",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	require.NoError(t, db.Create(kb).Error)

	article := &model.KnowledgeArticle{
		KBID:      kb.ID,
		Question:  "如何重置密码？",
		Answer:    "请访问设置页面，点击修改密码。",
		Category:  "账号管理",
		Tags:      datatypes.JSON(`["运维", "网络"]`),
		Status:    1,
		CreatedBy: 1,
		CreatedAt: now,
		UpdatedAt: now,
	}

	err := repo.CreateArticle(article)
	require.NoError(t, err)
	assert.NotZero(t, article.ID)

	// 验证可查回（含预加载 KnowledgeBase）
	got, err := repo.FindArticleByID(article.ID)
	require.NoError(t, err)
	assert.Equal(t, "如何重置密码？", got.Question)
	assert.Equal(t, "请访问设置页面，点击修改密码。", got.Answer)
	assert.Equal(t, "账号管理", got.Category)
	assert.Equal(t, int16(1), got.Status)
	assert.Equal(t, kb.ID, got.KBID)
	// 验证预加载了 KnowledgeBase
	assert.NotNil(t, got.KnowledgeBase)
	assert.Equal(t, "文章测试库", got.KnowledgeBase.Name)
}

// TestKnowledgeRepo_FindArticleByID_NotFound 查询不存在的文章
func TestKnowledgeRepo_FindArticleByID_NotFound(t *testing.T) {
	db := setupKnowledgeTestDB(t)
	repo := repository.NewKnowledgeRepo(db)

	got, err := repo.FindArticleByID(999999)
	assert.Error(t, err)
	assert.Nil(t, got)
	assert.Equal(t, gorm.ErrRecordNotFound, err)
}

// TestKnowledgeRepo_UpdateArticle 更新文章
func TestKnowledgeRepo_UpdateArticle(t *testing.T) {
	db := setupKnowledgeTestDB(t)
	cleanKnowledgeTables(t, db)
	repo := repository.NewKnowledgeRepo(db)

	now := time.Now()
	kb := &model.KnowledgeBase{
		Name:             "更新测试库",
		RAGWorkspaceSlug: "update-test-slug",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	require.NoError(t, db.Create(kb).Error)

	article := &model.KnowledgeArticle{
		KBID:      kb.ID,
		Question:  "旧问题",
		Answer:    "旧答案",
		Status:    1,
		CreatedBy: 1,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, db.Create(article).Error)

	article.Question = "新问题"
	article.Answer = "新答案"
	article.UpdatedAt = time.Now()

	err := repo.UpdateArticle(article)
	require.NoError(t, err)

	got, err := repo.FindArticleByID(article.ID)
	require.NoError(t, err)
	assert.Equal(t, "新问题", got.Question)
	assert.Equal(t, "新答案", got.Answer)
}

// TestKnowledgeRepo_ListArticles 分页查询文章列表（按知识库和状态过滤）
func TestKnowledgeRepo_ListArticles(t *testing.T) {
	db := setupKnowledgeTestDB(t)
	cleanKnowledgeTables(t, db)
	repo := repository.NewKnowledgeRepo(db)

	now := time.Now()
	kb := &model.KnowledgeBase{
		Name:             "列表测试库",
		RAGWorkspaceSlug: "list-test-slug",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	require.NoError(t, db.Create(kb).Error)

	// 创建 3 篇文章：2 篇草稿，1 篇已发布
	for i := 0; i < 2; i++ {
		a := &model.KnowledgeArticle{
			KBID:      kb.ID,
			Question:  "问题" + string(rune('A'+i)),
			Answer:    "答案" + string(rune('A'+i)),
			Status:    1, // 草稿
			CreatedBy: 1,
			CreatedAt: now,
			UpdatedAt: now,
		}
		require.NoError(t, db.Create(a).Error)
	}

	published := &model.KnowledgeArticle{
		KBID:      kb.ID,
		Question:  "已发布问题",
		Answer:    "已发布答案",
		Status:    3, // 已发布
		CreatedBy: 1,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, db.Create(published).Error)

	// 查询草稿（status=1）
	articles, total, err := repo.ListArticles(kb.ID, 1, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, articles, 2)

	// 查询已发布（status=3）
	articles, total, err = repo.ListArticles(kb.ID, 3, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, articles, 1)
	assert.Equal(t, "已发布问题", articles[0].Question)

	// 查询全部（status=0）
	articles, total, err = repo.ListArticles(kb.ID, 0, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, articles, 3)
}

// TestKnowledgeRepo_ListArticles_Pagination 分页功能
func TestKnowledgeRepo_ListArticles_Pagination(t *testing.T) {
	db := setupKnowledgeTestDB(t)
	cleanKnowledgeTables(t, db)
	repo := repository.NewKnowledgeRepo(db)

	now := time.Now()
	kb := &model.KnowledgeBase{
		Name:             "分页测试库",
		RAGWorkspaceSlug: "page-test-slug",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	require.NoError(t, db.Create(kb).Error)

	for i := 0; i < 5; i++ {
		a := &model.KnowledgeArticle{
			KBID:      kb.ID,
			Question:  "问题",
			Answer:    "答案",
			Status:    1,
			CreatedBy: 1,
			CreatedAt: now,
			UpdatedAt: now,
		}
		require.NoError(t, db.Create(a).Error)
	}

	// 第 1 页，每页 2 条
	articles, total, err := repo.ListArticles(kb.ID, 0, 1, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, articles, 2)

	// 第 2 页
	articles, total, err = repo.ListArticles(kb.ID, 0, 2, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, articles, 2)

	// 第 3 页（仅 1 条）
	articles, total, err = repo.ListArticles(kb.ID, 0, 3, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, articles, 1)
}

// TestKnowledgeRepo_UpdateArticleStatus 更新文章状态
func TestKnowledgeRepo_UpdateArticleStatus(t *testing.T) {
	db := setupKnowledgeTestDB(t)
	cleanKnowledgeTables(t, db)
	repo := repository.NewKnowledgeRepo(db)

	now := time.Now()
	kb := &model.KnowledgeBase{
		Name:             "状态测试库",
		RAGWorkspaceSlug: "status-test-slug",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	require.NoError(t, db.Create(kb).Error)

	article := &model.KnowledgeArticle{
		KBID:      kb.ID,
		Question:  "状态问题",
		Answer:    "状态答案",
		Status:    1, // 草稿
		CreatedBy: 1,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, db.Create(article).Error)

	// 草稿 → 待审核
	err := repo.UpdateArticleStatus(article.ID, 2)
	require.NoError(t, err)

	got, err := repo.FindArticleByID(article.ID)
	require.NoError(t, err)
	assert.Equal(t, int16(2), got.Status)
}

// =============================================================================
// KnowledgeChunk 测试
// =============================================================================

// TestKnowledgeRepo_CreateChunks 批量创建知识切片
func TestKnowledgeRepo_CreateChunks(t *testing.T) {
	db := setupKnowledgeTestDB(t)
	cleanKnowledgeTables(t, db)
	repo := repository.NewKnowledgeRepo(db)

	chunks := []model.KnowledgeChunk{
		{
			ArticleID:       1,
			Content:         "切片内容1",
			Embedding:       testVector(1536),
			EmbeddingModel:  "text-embedding-3-small",
			VectorDimension: 1536,
			SyncStatus:      "pending",
			CreatedAt:       time.Now(),
		},
		{
			ArticleID:       1,
			Content:         "切片内容2",
			Embedding:       testVector(1536),
			EmbeddingModel:  "text-embedding-3-small",
			VectorDimension: 1536,
			SyncStatus:      "pending",
			CreatedAt:       time.Now(),
		},
	}

	err := repo.CreateChunks(chunks)
	require.NoError(t, err)

	// 验证 ID 被填充
	assert.NotZero(t, chunks[0].ID)
	assert.NotZero(t, chunks[1].ID)

	// 验证可查回
	got, err := repo.FindChunksByArticleID(1)
	require.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Equal(t, "切片内容1", got[0].Content)
	assert.Equal(t, "切片内容2", got[1].Content)
}

// TestKnowledgeRepo_FindChunksByArticleID_Empty 无切片时返回空
func TestKnowledgeRepo_FindChunksByArticleID_Empty(t *testing.T) {
	db := setupKnowledgeTestDB(t)
	cleanKnowledgeTables(t, db)
	repo := repository.NewKnowledgeRepo(db)

	got, err := repo.FindChunksByArticleID(999999)
	require.NoError(t, err)
	assert.Empty(t, got)
}

// TestKnowledgeRepo_UpdateChunkSyncStatus 更新切片的同步状态
func TestKnowledgeRepo_UpdateChunkSyncStatus(t *testing.T) {
	db := setupKnowledgeTestDB(t)
	cleanKnowledgeTables(t, db)
	repo := repository.NewKnowledgeRepo(db)

	chunks := []model.KnowledgeChunk{
		{
			ArticleID:       100,
			Content:         "待同步内容",
			Embedding:       testVector(768),
			EmbeddingModel:  "test-model",
			VectorDimension: 768,
			SyncStatus:      "pending",
			CreatedAt:       time.Now(),
		},
		{
			ArticleID:       100,
			Content:         "待同步内容2",
			Embedding:       testVector(768),
			EmbeddingModel:  "test-model",
			VectorDimension: 768,
			SyncStatus:      "pending",
			CreatedAt:       time.Now(),
		},
	}
	require.NoError(t, db.Create(&chunks).Error)

	// 更新同步状态为 synced
	err := repo.UpdateChunkSyncStatus(100, "synced", "")
	require.NoError(t, err)

	got, err := repo.FindChunksByArticleID(100)
	require.NoError(t, err)
	assert.Len(t, got, 2)
	for _, c := range got {
		assert.Equal(t, "synced", c.SyncStatus)
	}
}

// TestKnowledgeRepo_UpdateChunkSyncStatus_WithError 更新同步状态含错误信息
func TestKnowledgeRepo_UpdateChunkSyncStatus_WithError(t *testing.T) {
	db := setupKnowledgeTestDB(t)
	cleanKnowledgeTables(t, db)
	repo := repository.NewKnowledgeRepo(db)

	chunk := model.KnowledgeChunk{
		ArticleID:       200,
		Content:         "同步失败内容",
		Embedding:       testVector(768),
		EmbeddingModel:  "test-model",
		VectorDimension: 768,
		SyncStatus:      "pending",
		CreatedAt:       time.Now(),
	}
	require.NoError(t, db.Create(&chunk).Error)

	err := repo.UpdateChunkSyncStatus(200, "failed", "Embedding API 超时")
	require.NoError(t, err)

	got, err := repo.FindChunksByArticleID(200)
	require.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, "failed", got[0].SyncStatus)
	assert.Equal(t, "Embedding API 超时", got[0].SyncError)
}

// TestKnowledgeRepo_UpdateChunkStatusByArticleID 批量更新切片的同步状态
func TestKnowledgeRepo_UpdateChunkStatusByArticleID(t *testing.T) {
	db := setupKnowledgeTestDB(t)
	cleanKnowledgeTables(t, db)
	repo := repository.NewKnowledgeRepo(db)

	chunks := []model.KnowledgeChunk{
		{
			ArticleID:       300,
			Content:         "批量更新1",
			Embedding:       testVector(768),
			EmbeddingModel:  "test-model",
			VectorDimension: 768,
			SyncStatus:      "synced",
			CreatedAt:       time.Now(),
		},
		{
			ArticleID:       300,
			Content:         "批量更新2",
			Embedding:       testVector(768),
			EmbeddingModel:  "test-model",
			VectorDimension: 768,
			SyncStatus:      "synced",
			CreatedAt:       time.Now(),
		},
	}
	require.NoError(t, db.Create(&chunks).Error)

	// 批量标记为 disabled
	err := repo.UpdateChunkStatusByArticleID(300, "disabled")
	require.NoError(t, err)

	got, err := repo.FindChunksByArticleID(300)
	require.NoError(t, err)
	for _, c := range got {
		assert.Equal(t, "disabled", c.SyncStatus)
	}
}

// =============================================================================
// EmbeddingConfig 测试
// =============================================================================

// TestKnowledgeRepo_CreateEmbeddingConfig 创建 Embedding 配置
func TestKnowledgeRepo_CreateEmbeddingConfig(t *testing.T) {
	db := setupKnowledgeTestDB(t)
	cleanKnowledgeTables(t, db)
	repo := repository.NewKnowledgeRepo(db)

	cfg := &model.EmbeddingConfig{
		Name:           "OpenAI Embedding",
		ModelType:      1, // API 类型
		APIEndpoint:    "https://api.openai.com/v1/embeddings",
		APIKey:         "sk-test-key",
		VectorDimension: 1536,
		IsDefault:      true,
		CreatedAt:      time.Now(),
	}

	err := repo.CreateEmbeddingConfig(cfg)
	require.NoError(t, err)
	assert.NotZero(t, cfg.ID)

	// 验证可查回
	configs, err := repo.ListEmbeddingConfigs()
	require.NoError(t, err)
	assert.Len(t, configs, 1)
	assert.Equal(t, "OpenAI Embedding", configs[0].Name)
	assert.Equal(t, int16(1), configs[0].ModelType)
	assert.True(t, configs[0].IsDefault)
}

// TestKnowledgeRepo_CreateEmbeddingConfig_Local 创建本地类型 Embedding 配置
func TestKnowledgeRepo_CreateEmbeddingConfig_Local(t *testing.T) {
	db := setupKnowledgeTestDB(t)
	cleanKnowledgeTables(t, db)
	repo := repository.NewKnowledgeRepo(db)

	cfg := &model.EmbeddingConfig{
		Name:           "本地 BGE 模型",
		ModelType:      2, // 本地类型
		LocalPath:      "/models/bge-large-zh",
		VectorDimension: 1024,
		IsDefault:      false,
		CreatedAt:      time.Now(),
	}

	err := repo.CreateEmbeddingConfig(cfg)
	require.NoError(t, err)

	got, err := repo.ListEmbeddingConfigs()
	require.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, int16(2), got[0].ModelType)
	assert.Equal(t, "/models/bge-large-zh", got[0].LocalPath)
	assert.Empty(t, got[0].APIEndpoint)
}

// TestKnowledgeRepo_UpdateEmbeddingConfig 更新 Embedding 配置
func TestKnowledgeRepo_UpdateEmbeddingConfig(t *testing.T) {
	db := setupKnowledgeTestDB(t)
	cleanKnowledgeTables(t, db)
	repo := repository.NewKnowledgeRepo(db)

	cfg := &model.EmbeddingConfig{
		Name:           "旧名称",
		ModelType:      1,
		APIEndpoint:    "https://old.example.com",
		VectorDimension: 1536,
		IsDefault:      false,
		CreatedAt:      time.Now(),
	}
	require.NoError(t, db.Create(cfg).Error)

	cfg.Name = "新名称"
	cfg.APIEndpoint = "https://new.example.com"
	cfg.IsDefault = true

	err := repo.UpdateEmbeddingConfig(cfg)
	require.NoError(t, err)

	got, err := repo.GetDefaultEmbeddingConfig()
	require.NoError(t, err)
	assert.Equal(t, "新名称", got.Name)
	assert.Equal(t, "https://new.example.com", got.APIEndpoint)
	assert.True(t, got.IsDefault)
}

// TestKnowledgeRepo_ListEmbeddingConfigs 列出全部 Embedding 配置
func TestKnowledgeRepo_ListEmbeddingConfigs(t *testing.T) {
	db := setupKnowledgeTestDB(t)
	cleanKnowledgeTables(t, db)
	repo := repository.NewKnowledgeRepo(db)

	cfg1 := &model.EmbeddingConfig{
		Name:           "配置1",
		ModelType:      1,
		VectorDimension: 768,
		CreatedAt:      time.Now(),
	}
	cfg2 := &model.EmbeddingConfig{
		Name:           "配置2",
		ModelType:      2,
		VectorDimension: 1024,
		CreatedAt:      time.Now(),
	}
	require.NoError(t, db.Create(cfg1).Error)
	require.NoError(t, db.Create(cfg2).Error)

	configs, err := repo.ListEmbeddingConfigs()
	require.NoError(t, err)
	assert.Len(t, configs, 2)
}

// TestKnowledgeRepo_GetDefaultEmbeddingConfig 获取默认 Embedding 配置
func TestKnowledgeRepo_GetDefaultEmbeddingConfig(t *testing.T) {
	db := setupKnowledgeTestDB(t)
	cleanKnowledgeTables(t, db)
	repo := repository.NewKnowledgeRepo(db)

	// 创建一个非默认和一个默认
	cfg1 := &model.EmbeddingConfig{
		Name:           "非默认",
		ModelType:      1,
		VectorDimension: 768,
		IsDefault:      false,
		CreatedAt:      time.Now(),
	}
	cfg2 := &model.EmbeddingConfig{
		Name:           "默认配置",
		ModelType:      1,
		VectorDimension: 1536,
		IsDefault:      true,
		CreatedAt:      time.Now(),
	}
	require.NoError(t, db.Create(cfg1).Error)
	require.NoError(t, db.Create(cfg2).Error)

	got, err := repo.GetDefaultEmbeddingConfig()
	require.NoError(t, err)
	assert.Equal(t, "默认配置", got.Name)
	assert.True(t, got.IsDefault)
}

// TestKnowledgeRepo_GetDefaultEmbeddingConfig_NotFound 无默认配置时返回错误
func TestKnowledgeRepo_GetDefaultEmbeddingConfig_NotFound(t *testing.T) {
	db := setupKnowledgeTestDB(t)
	cleanKnowledgeTables(t, db)
	repo := repository.NewKnowledgeRepo(db)

	got, err := repo.GetDefaultEmbeddingConfig()
	assert.Error(t, err)
	assert.Nil(t, got)
	assert.Equal(t, gorm.ErrRecordNotFound, err)
}
