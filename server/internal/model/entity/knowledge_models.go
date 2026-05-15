// 知识库类 GORM 实体：知识库、分类、条目、切片、审核、同步、候选
package entity

import "time"

type KnowledgeBase struct {
	ID                 int64     `gorm:"primaryKey" json:"id"`
	Code               string    `gorm:"size:64;not null;uniqueIndex" json:"code"`
	Name               string    `gorm:"size:128;not null" json:"name"`
	Description        *string   `gorm:"size:500" json:"description"`
	EmbeddingModel     string    `gorm:"size:128;not null" json:"embedding_model"`
	EmbeddingDimension int       `gorm:"not null" json:"embedding_dimension"`
	RAGProvider        string    `gorm:"size:64;not null;default:anythingllm" json:"rag_provider"`
	Status             int16     `gorm:"not null;default:1" json:"status"` // 1启用 2停用
	CreatedBy          *int64    `json:"created_by"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func (KnowledgeBase) TableName() string { return "knowledge_base" }

type KnowledgeCategory struct {
	ID               int64     `gorm:"primaryKey" json:"id"`
	KnowledgeBaseID  int64     `gorm:"not null;index" json:"knowledge_base_id"`
	ParentID         *int64    `json:"parent_id"`
	Name             string    `gorm:"size:64;not null" json:"name"`
	SortOrder        int       `gorm:"not null;default:0" json:"sort_order"`
	Status           int16     `gorm:"not null;default:1" json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (KnowledgeCategory) TableName() string { return "knowledge_category" }

type KnowledgeArticle struct {
	ID                 int64      `gorm:"primaryKey" json:"id"`
	KnowledgeNo        string     `gorm:"size:64;not null;uniqueIndex" json:"knowledge_no"`
	KnowledgeBaseID    int64      `gorm:"not null" json:"knowledge_base_id"`
	CategoryID         int64      `gorm:"not null" json:"category_id"`
	Title              string     `gorm:"size:255;not null" json:"title"`
	Question           string     `gorm:"type:text;not null" json:"question"`
	Answer             string     `gorm:"type:text;not null" json:"answer"`
	Tags               *string    `gorm:"type:jsonb" json:"tags"`
	ApplicableScope    *string    `gorm:"size:255" json:"applicable_scope"`
	Status             int16      `gorm:"not null;default:1" json:"status"` // 1草稿 2待审核 3已发布 4已停用
	ReviewStatus       int16      `gorm:"not null;default:1" json:"review_status"` // 1待审 2通过 3驳回
	PublishedAt        *time.Time `json:"published_at"`
	MaintainerID       *int64     `json:"maintainer_id"`
	ReviewerID         *int64     `json:"reviewer_id"`
	EmbeddingModel     string     `gorm:"size:128;not null" json:"embedding_model"`
	EmbeddingDimension int        `gorm:"not null" json:"embedding_dimension"`
	RAGSyncStatus      int16      `gorm:"not null;default:1" json:"rag_sync_status"` // 1未同步 2同步中 3成功 4失败
	RAGSyncError       *string    `gorm:"size:500" json:"rag_sync_error"`
	VersionNo          int        `gorm:"not null;default:1" json:"version_no"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

func (KnowledgeArticle) TableName() string { return "knowledge_article" }

type KnowledgeChunk struct {
	ID                 int64     `gorm:"primaryKey" json:"id"`
	KnowledgeBaseID    int64     `gorm:"not null" json:"knowledge_base_id"`
	KnowledgeID        int64     `gorm:"not null;index" json:"knowledge_id"`
	ChunkNo            int       `gorm:"not null" json:"chunk_no"`
	Content            string    `gorm:"type:text;not null" json:"content"`
	Embedding          string    `gorm:"type:vector;not null" json:"embedding"`
	EmbeddingModel     string    `gorm:"size:128;not null" json:"embedding_model"`
	EmbeddingDimension int       `gorm:"not null" json:"embedding_dimension"`
	TokenCount         *int      `json:"token_count"`
	Metadata           *string   `gorm:"type:jsonb" json:"metadata"`
	RAGProvider        string    `gorm:"size:64;not null;default:anythingllm" json:"rag_provider"`
	Status             int16     `gorm:"not null;default:1" json:"status"` // 1有效 2失效
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func (KnowledgeChunk) TableName() string { return "knowledge_chunk" }

type KnowledgeReviewRecord struct {
	ID            int64     `gorm:"primaryKey" json:"id"`
	KnowledgeID   int64     `gorm:"not null;index" json:"knowledge_id"`
	ReviewerID    int64     `gorm:"not null" json:"reviewer_id"`
	ReviewResult  int16     `gorm:"not null" json:"review_result"` // 1通过 2驳回
	ReviewComment *string   `gorm:"size:500" json:"review_comment"`
	CreatedAt     time.Time `json:"created_at"`
}

func (KnowledgeReviewRecord) TableName() string { return "knowledge_review_record" }

type KnowledgeSyncRecord struct {
	ID           int64      `gorm:"primaryKey" json:"id"`
	KnowledgeID  int64      `gorm:"not null;index" json:"knowledge_id"`
	Provider     string     `gorm:"size:64;not null" json:"provider"`
	EventID      *string    `gorm:"size:128" json:"event_id"`
	SyncStatus   int16      `gorm:"not null" json:"sync_status"` // 1处理中 2成功 3失败
	SyncPayload  *string    `gorm:"type:jsonb" json:"sync_payload"`
	ErrorMessage *string    `gorm:"size:500" json:"error_message"`
	SyncedAt     *time.Time `json:"synced_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

func (KnowledgeSyncRecord) TableName() string { return "knowledge_sync_record" }

type KnowledgeCandidate struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	TicketID  int64     `gorm:"not null;uniqueIndex" json:"ticket_id"`
	Title     string    `gorm:"size:255;not null" json:"title"`
	Summary   string    `gorm:"type:text;not null" json:"summary"`
	Status    int16     `gorm:"not null;default:1" json:"status"` // 1待审核 2已转知识 3已忽略
	CreatedBy int64     `gorm:"not null" json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (KnowledgeCandidate) TableName() string { return "knowledge_candidate" }
