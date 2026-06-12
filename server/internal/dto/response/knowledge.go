// Package response 定义知识库管理相关的响应结构体。
//
// 与 TECH.md §5.2 知识库管理 API 对齐。
package response

import "time"

// KBResponse 知识库响应。
type KBResponse struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	RAGWorkspaceSlug string `json:"rag_workspace_slug"`
	EmbeddingModel string `json:"embedding_model"`
	VectorDimension int  `json:"vector_dimension"`
	CreatedBy      int64  `json:"created_by"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

// KBListResponse 知识库列表响应。
type KBListResponse struct {
	Items []KBResponse `json:"items"`
}

// ArticleListResponse 文章列表响应（含分页）。
type ArticleListResponse struct {
	Articles []ArticleResponse `json:"articles"`
	Total    int64             `json:"total"`
}

// ArticleResponse 文章列表项响应。
type ArticleResponse struct {
	ID            int64     `json:"id"`
	KBID          int64     `json:"kb_id"`
	KBName        string    `json:"kb_name"`
	Question      string    `json:"question"`
	Answer        string    `json:"answer"`
	Category      string    `json:"category"`
	Tags          []string  `json:"tags"`
	Status        int16     `json:"status"`
	StatusText    string    `json:"status_text"`
	CreatedBy     int64     `json:"created_by"`
	ReviewedBy    *int64    `json:"reviewed_by"`
	ReviewComment string    `json:"review_comment"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ArticleDetailResponse 文章详情响应（含切片）。
type ArticleDetailResponse struct {
	ArticleResponse
	Chunks []ChunkResponse `json:"chunks"`
}

// ChunkResponse 知识切片响应。
type ChunkResponse struct {
	ID              int64  `json:"id"`
	KBID            int64  `json:"kb_id"`
	Content         string `json:"content"`
	ChunkIndex      int    `json:"chunk_index"`
	EmbeddingModel  string `json:"embedding_model"`
	VectorDimension int    `json:"vector_dimension"`
}
