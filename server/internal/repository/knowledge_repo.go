// Package repository 提供知识库的数据访问层。
//
// KnowledgeRepo 封装 knowledge_bases、knowledge_articles、knowledge_chunks、
// embedding_configs 四张表的 CRUD 操作，供 KnowledgeService 调用。
// 为什么独立于 UserRepo：知识库相关表的查询模式不同（按 kb_id 过滤分页、
// 同步状态批量更新、预加载关联表），独立 Repo 更利于职责聚焦。
package repository

import (
	"opsmind/internal/model"
	"time"

	"gorm.io/gorm"
)

// KnowledgeRepo 知识库数据访问
type KnowledgeRepo struct {
	db *gorm.DB
}

// NewKnowledgeRepo 创建 KnowledgeRepo 实例
func NewKnowledgeRepo(db *gorm.DB) *KnowledgeRepo {
	return &KnowledgeRepo{db: db}
}

// =============================================================================
// KnowledgeBase
// =============================================================================

// CreateKB 创建知识库。
//
// 创建后 kb.ID 会被 GORM 自动填充。
// rag_workspace_slug 唯一约束由数据库保证，重复时返回 PostgreSQL 错误。
func (r *KnowledgeRepo) CreateKB(kb *model.KnowledgeBase) error {
	return r.db.Create(kb).Error
}

// FindKBByID 按 ID 查询知识库。
//
// ID 是主键，查询不到时返回 gorm.ErrRecordNotFound。
func (r *KnowledgeRepo) FindKBByID(id int64) (*model.KnowledgeBase, error) {
	var kb model.KnowledgeBase
	err := r.db.Where("id = ?", id).First(&kb).Error
	if err != nil {
		return nil, err
	}
	return &kb, nil
}

// UpdateKB 更新知识库全部字段。
//
// 为什么用 Save 而非 Updates：Save 会更新所有字段（包括零值），
// 适用于名称、描述、embedding 配置等全量覆盖场景。
func (r *KnowledgeRepo) UpdateKB(kb *model.KnowledgeBase) error {
	return r.db.Save(kb).Error
}

// ListKBs 列出全部知识库。
//
// 为什么返回空切片而非 nil：调用方可以直接 range 遍历，无需额外 nil 判断。
func (r *KnowledgeRepo) ListKBs() ([]model.KnowledgeBase, error) {
	var kbs []model.KnowledgeBase
	err := r.db.Order("id ASC").Find(&kbs).Error
	if err != nil {
		return nil, err
	}
	if kbs == nil {
		kbs = []model.KnowledgeBase{}
	}
	return kbs, nil
}

// =============================================================================
// KnowledgeArticle
// =============================================================================

// CreateArticle 创建知识文章。
//
// 创建后 article.ID 会被 GORM 自动填充。
func (r *KnowledgeRepo) CreateArticle(article *model.KnowledgeArticle) error {
	return r.db.Create(article).Error
}

// FindArticleByID 按 ID 查询文章，预加载关联的 KnowledgeBase。
//
// 为什么预加载 KnowledgeBase：文章详情需要显示所属知识库名称，
// Preload 一条 SQL 完成 JOIN 避免 N+1 查询。
func (r *KnowledgeRepo) FindArticleByID(id int64) (*model.KnowledgeArticle, error) {
	var article model.KnowledgeArticle
	err := r.db.Preload("KnowledgeBase").Where("id = ?", id).First(&article).Error
	if err != nil {
		return nil, err
	}
	return &article, nil
}

// UpdateArticle 更新文章全部字段。
//
// 为什么用 Save 而非 Updates：编辑文章时会修改 question、answer、category、tags
// 等多个字段，Save 确保全部更新（包括零值字段）。
func (r *KnowledgeRepo) UpdateArticle(article *model.KnowledgeArticle) error {
	return r.db.Save(article).Error
}

// ListArticles 分页查询文章列表，支持按知识库和状态过滤。
//
// 参数说明：
//   - kbID: 知识库 ID，必传（>0）
//   - status: 文章状态，传 -1 表示不过滤，0 表示已停用
//   - page/pageSize: 分页参数
//
// 为什么按 id DESC 排序：最新创建的文章排在前面，符合管理后台使用习惯。
func (r *KnowledgeRepo) ListArticles(kbID int64, status int, page, pageSize int) ([]model.KnowledgeArticle, int64, error) {
	var articles []model.KnowledgeArticle
	var total int64

	query := r.db.Model(&model.KnowledgeArticle{}).Where("kb_id = ?", kbID)
	// TODO: 缺少 .Preload("KnowledgeBase") — 导致 ListArticles 的 KBName 字段触发 N+1 查询。
	if status >= 0 {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("id DESC").Find(&articles).Error; err != nil {
		return nil, 0, err
	}

	return articles, total, nil
}

// UpdateArticleStatus 更新文章状态。
//
// 为什么单独封装而非复用 UpdateArticle：仅更新 status 字段，
// 避免 Save 时意外覆盖其他字段（如 reviewed_by、review_comment 等）。
func (r *KnowledgeRepo) UpdateArticleStatus(id int64, status int) error {
	return r.db.Model(&model.KnowledgeArticle{}).Where("id = ?", id).Update("status", status).Error
}

// =============================================================================
// KnowledgeChunk
// =============================================================================

// CreateChunks 批量创建知识切片。
//
// 为什么用 Create 批量而非逐条插入：单条 INSERT 即可完成，
// GORM 自动将切片参数展开为多行 VALUES。
func (r *KnowledgeRepo) CreateChunks(chunks []model.KnowledgeChunk) error {
	if len(chunks) == 0 {
		return nil
	}
	return r.db.Create(&chunks).Error
}

// FindChunksByArticleID 按文章 ID 查询全部切片。
//
// 返回的切片按 ID 升序排列，保证顺序一致。
func (r *KnowledgeRepo) FindChunksByArticleID(articleID int64) ([]model.KnowledgeChunk, error) {
	var chunks []model.KnowledgeChunk
	err := r.db.Where("article_id = ?", articleID).Order("id ASC").Find(&chunks).Error
	if err != nil {
		return nil, err
	}
	if chunks == nil {
		chunks = []model.KnowledgeChunk{}
	}
	return chunks, nil
}

// UpdateChunkSyncStatus 更新指定文章全部切片的同步状态。
//
// 为什么按 article_id 批量更新而非逐条：一次 SQL 更新所有切片，
// 避免 N 次数据库往返。syncError 参数同时更新错误信息。
// synced_at 字段在成功时记录当前时间，失败时设为 nil。
func (r *KnowledgeRepo) UpdateChunkSyncStatus(articleID int64, status string, syncError string) error {
	updates := map[string]interface{}{
		"sync_status": status,
		"sync_error":  syncError,
	}
	if status == "synced" {
		now := time.Now()
		updates["synced_at"] = &now
	} else {
		updates["synced_at"] = nil
	}
	return r.db.Model(&model.KnowledgeChunk{}).
		Where("article_id = ?", articleID).
		Updates(updates).Error
}

// UpdateChunkStatusByArticleID 批量更新指定文章全部切片的同步状态。
//
// 为什么独立于 UpdateChunkSyncStatus：停用操作不需要记录错误信息，
// 也不需要更新 synced_at，接口更简洁。
func (r *KnowledgeRepo) UpdateChunkStatusByArticleID(articleID int64, status string) error {
	return r.db.Model(&model.KnowledgeChunk{}).
		Where("article_id = ?", articleID).
		Update("sync_status", status).Error
}

// =============================================================================
// EmbeddingConfig
// TODO: 以下 5 个方法 (CreateEmbeddingConfig / UpdateEmbeddingConfig / ListEmbeddingConfigs /
// DeleteEmbeddingConfig / GetDefaultEmbeddingConfig) 在 Service 层无调用方，仅测试引用，
// 属于死代码。若短期内无使用计划应删除。
// =============================================================================

// CreateEmbeddingConfig 创建 Embedding 配置。
//
// 创建后 cfg.ID 会被 GORM 自动填充。
func (r *KnowledgeRepo) CreateEmbeddingConfig(cfg *model.EmbeddingConfig) error {
	return r.db.Create(cfg).Error
}

// UpdateEmbeddingConfig 更新 Embedding 配置全部字段。
//
// 为什么用 Save：更新配置时需要覆盖所有字段（包括 model_type、api_endpoint 等改回零值的场景）。
func (r *KnowledgeRepo) UpdateEmbeddingConfig(cfg *model.EmbeddingConfig) error {
	return r.db.Save(cfg).Error
}

// ListEmbeddingConfigs 列出全部 Embedding 配置。
func (r *KnowledgeRepo) ListEmbeddingConfigs() ([]model.EmbeddingConfig, error) {
	var configs []model.EmbeddingConfig
	err := r.db.Order("id ASC").Find(&configs).Error
	if err != nil {
		return nil, err
	}
	if configs == nil {
		configs = []model.EmbeddingConfig{}
	}
	return configs, nil
}

// DeleteEmbeddingConfig 删除 Embedding 配置。
//
// 按 ID 删除，记录不存在时返回 gorm.ErrRecordNotFound。
func (r *KnowledgeRepo) DeleteEmbeddingConfig(id int64) error {
	result := r.db.Delete(&model.EmbeddingConfig{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// GetDefaultEmbeddingConfig 获取默认的 Embedding 配置。
//
// 查询 is_default=true 的配置，不存在时返回 gorm.ErrRecordNotFound。
// 为什么查询不到报错而非返回 nil：语义更清晰（调用方立刻知道没有默认配置），
// 避免在 Service 层重复 nil 检查。
func (r *KnowledgeRepo) GetDefaultEmbeddingConfig() (*model.EmbeddingConfig, error) {
	var cfg model.EmbeddingConfig
	err := r.db.Where("is_default = ?", true).First(&cfg).Error
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
