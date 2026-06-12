package model

import (
	"time"

	"gorm.io/datatypes"
)

// KnowledgeBase 知识库表
type KnowledgeBase struct {
	ID               int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name             string    `gorm:"type:varchar(128);not null" json:"name"`
	Description      string    `gorm:"type:text" json:"description"`
	RAGWorkspaceSlug string    `gorm:"type:varchar(128);uniqueIndex;column:rag_workspace_slug" json:"rag_workspace_slug"`
	EmbeddingModel   string    `gorm:"type:varchar(128);not null;column:embedding_model" json:"embedding_model"`
	VectorDimension  int       `gorm:"not null;column:vector_dimension" json:"vector_dimension"`
	CreatedBy        int64     `gorm:"column:created_by" json:"created_by"`
	CreatedAt        time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt        time.Time `gorm:"not null" json:"updated_at"`
}

func (KnowledgeBase) TableName() string { return "knowledge_bases" }

// KnowledgeArticle 知识文章表
type KnowledgeArticle struct {
	ID                  int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	KBID                int64          `gorm:"not null;column:kb_id" json:"kb_id"`
	KnowledgeBase       KnowledgeBase  `gorm:"foreignKey:KBID;references:ID" json:"knowledge_base,omitempty"`
	Question            string         `gorm:"type:text;not null" json:"question"`
	Answer              string         `gorm:"type:text;not null" json:"answer"`
	Category            string         `gorm:"type:varchar(64)" json:"category"`
	Tags                datatypes.JSON `gorm:"type:jsonb" json:"tags"`
	Status              int16          `gorm:"not null;default:1;index:idx_articles_status" json:"status"`
	CreatedBy           int64          `gorm:"column:created_by" json:"created_by"`
	ReviewedBy          *int64         `gorm:"column:reviewed_by" json:"reviewed_by"`
	PublishedBy         *int64         `gorm:"column:published_by" json:"published_by"`
	ReviewComment       string         `gorm:"type:text;column:review_comment" json:"review_comment"`
	RAGDocumentLocation string         `gorm:"type:varchar(512);column:rag_document_location" json:"rag_document_location"`
	CreatedAt           time.Time      `gorm:"not null" json:"created_at"`
	UpdatedAt           time.Time      `gorm:"not null" json:"updated_at"`
}

func (KnowledgeArticle) TableName() string { return "knowledge_articles" }

// KnowledgeChunk 知识切片表。
// 记录知识条目发布时的切片内容和 pgvector 向量。
// embedding 向量以 halfvec 类型存储在 pgvector 中（通过 VectorStore 接口管理）。
type KnowledgeChunk struct {
	ID              int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	ArticleID       int64      `gorm:"not null;column:article_id;index:idx_chunks_article_id" json:"article_id"`
	Content         string     `gorm:"type:text;not null" json:"content"`
	EmbeddingModel  string     `gorm:"type:varchar(128);not null;column:embedding_model" json:"embedding_model"`
	VectorDimension int        `gorm:"not null;column:vector_dimension" json:"vector_dimension"`
	SyncStatus      string     `gorm:"type:varchar(16);not null;default:'pending';column:sync_status" json:"sync_status"`
	SyncError       string     `gorm:"type:text;column:sync_error" json:"sync_error"`
	SyncedAt        *time.Time `gorm:"column:synced_at" json:"synced_at"`
	CreatedAt       time.Time  `gorm:"not null" json:"created_at"`
}

func (KnowledgeChunk) TableName() string { return "knowledge_chunks" }
