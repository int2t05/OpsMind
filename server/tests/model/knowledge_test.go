//go:build integration

package model_test

import (
	"testing"
	"time"

	"opsmind/internal/model"

	"github.com/pgvector/pgvector-go"
	"gorm.io/datatypes"
)

// TestKnowledgeBase_Fields 验证 KnowledgeBase 模型字段与 TECH.md §4.2 knowledge_bases 表定义一致
func TestKnowledgeBase_Fields(t *testing.T) {
	now := time.Now()
	kb := model.KnowledgeBase{
		Name:             "运维知识库",
		Description:      "常见运维问题解答",
		RAGWorkspaceSlug: "ops-kb",
		EmbeddingModel:   "text-embedding-3-small",
		VectorDimension:  1536,
		CreatedBy:        1,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	kb.ID = 1

	if kb.Name != "运维知识库" {
		t.Errorf("Name = %q, 期望 运维知识库", kb.Name)
	}
	if kb.EmbeddingModel != "text-embedding-3-small" {
		t.Errorf("EmbeddingModel = %q, 期望 text-embedding-3-small", kb.EmbeddingModel)
	}
	if kb.VectorDimension != 1536 {
		t.Errorf("VectorDimension = %d, 期望 1536", kb.VectorDimension)
	}
	if kb.RAGWorkspaceSlug != "ops-kb" {
		t.Errorf("RAGWorkspaceSlug = %q, 期望 ops-kb", kb.RAGWorkspaceSlug)
	}
}

// TestKnowledgeArticle_Fields 验证 KnowledgeArticle 模型字段与 TECH.md §4.2 knowledge_articles 表定义一致
func TestKnowledgeArticle_Fields(t *testing.T) {
	now := time.Now()
	tags := datatypes.JSON(`["OA","登录"]`)
	art := model.KnowledgeArticle{
		KBID:               1,
		Question:           "OA系统无法登录怎么办？",
		Answer:             "请清除浏览器缓存后重试",
		Category:           "OA系统",
		Tags:               tags,
		Status:             model.ArticleStatusDraft,
		CreatedBy:          1,
		RAGDocumentLocation: "doc-uuid-123",
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	art.ID = 1

	if art.KBID != 1 {
		t.Errorf("KBID = %d, 期望 1", art.KBID)
	}
	if art.Status != model.ArticleStatusDraft {
		t.Errorf("Status = %d, 期望 %d", art.Status, model.ArticleStatusDraft)
	}
	if art.Tags == nil {
		t.Error("Tags 为 nil, 期望有值")
	}
}

// TestKnowledgeChunk_Fields 验证 KnowledgeChunk 模型字段与 TECH.md §4.2 knowledge_chunks 表定义一致
func TestKnowledgeChunk_Fields(t *testing.T) {
	now := time.Now()
	vec := pgvector.NewVector([]float32{0.1, 0.2, 0.3})
	chunk := model.KnowledgeChunk{
		ArticleID:       1,
		Content:         "OA系统登录问题",
		Embedding:       vec,
		EmbeddingModel:  "text-embedding-3-small",
		VectorDimension: 3,
		SyncStatus:      model.ChunkSyncPending,
		CreatedAt:       now,
	}
	chunk.ID = 1

	if chunk.ArticleID != 1 {
		t.Errorf("ArticleID = %d, 期望 1", chunk.ArticleID)
	}
	if chunk.SyncStatus != model.ChunkSyncPending {
		t.Errorf("SyncStatus = %q, 期望 %q", chunk.SyncStatus, model.ChunkSyncPending)
	}
	if chunk.VectorDimension != 3 {
		t.Errorf("VectorDimension = %d, 期望 3", chunk.VectorDimension)
	}
	if chunk.Embedding.Slice()[0] != 0.1 {
		t.Errorf("Embedding[0] = %f, 期望 0.1", chunk.Embedding.Slice()[0])
	}
}
