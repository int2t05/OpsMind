// Package service 实现知识库管理业务逻辑。
//
// knowledge_service_v2.go 提供 v2 版 KnowledgeService（自建 pgvector 发布/停用）。
//
// v1→v2 变更：
//   - 移除 RagClient（AnythingLLM）依赖
//   - Publish：Chunker.Split → Embedder.Embed → VectorStore.BatchInsert（替代 RagClient.SyncDocument）
//   - Disable：VectorStore.DeleteByArticle（替代 RagClient.DisableDocument）
//   - Enable：重置为草稿（替代旧的 RagClient 恢复逻辑）
package service

import (
	"context"
	"fmt"

	"opsmind/internal/adapter"
	"opsmind/internal/model"
	"opsmind/internal/rag"
	"opsmind/pkg/errcode"
)

// =============================================================================
// V2 依赖接口
// =============================================================================

// knowledgeRepoV2 知识库仓库 v2 接口。
type knowledgeRepoV2 interface {
	FindKBByID(id int64) (*model.KnowledgeBase, error)
	FindArticleByID(id int64) (*model.KnowledgeArticle, error)
	UpdateArticle(article *model.KnowledgeArticle) error
	UpdateArticleStatus(id int64, status int16) error
}

// chunkerV2 分块器接口。
type chunkerV2 interface {
	Split(text string) []string
}

// embedderV2 向量生成器接口。
type embedderV2 interface {
	Embed(ctx context.Context, texts []string) ([][]float32, int, error)
}

// vectorStoreV2 向量存储接口。
type vectorStoreV2 interface {
	BatchInsert(ctx context.Context, chunks []adapter.VectorChunk) error
	DeleteByArticle(ctx context.Context, articleID int64) error
}

// =============================================================================
// KnowledgeServiceV2
// =============================================================================

// KnowledgeServiceV2 使用自建 pgvector 管道的知识库服务。
type KnowledgeServiceV2 struct {
	repo      knowledgeRepoV2
	chunker   chunkerV2
	embedder  embedderV2
	store     vectorStoreV2
	processor *rag.Processor
}

// NewKnowledgeServiceV2 创建 KnowledgeServiceV2 实例。
//
// chunker/embedder/store/processor 都可以为 nil（测试不需要全部依赖时）。
func NewKnowledgeServiceV2(repo interface{}, chunker interface{}, embedder interface{}, store interface{}, processor *rag.Processor) *KnowledgeServiceV2 {
	svc := &KnowledgeServiceV2{
		processor: processor,
	}

	if r, ok := repo.(knowledgeRepoV2); ok {
		svc.repo = r
	}
	if c, ok := chunker.(chunkerV2); ok {
		svc.chunker = c
	}
	if e, ok := embedder.(embedderV2); ok {
		svc.embedder = e
	}
	if s, ok := store.(vectorStoreV2); ok {
		svc.store = s
	}

	return svc
}

// =============================================================================
// PublishV2
// =============================================================================

// PublishV2 发布文章（分块→embedding→pgvector 写入）。
//
// 流程：
//  1. 校验状态（仅已通过(status=3)可发布）
//  2. Chunker.Split → 文本分块
//  3. Embedder.Embed → 生成向量
//  4. VectorStore.DeleteByArticle → 清除旧向量
//  5. VectorStore.BatchInsert → 写入新向量
//  6. 更新文章状态为已发布(status=4)
func (s *KnowledgeServiceV2) PublishV2(articleID int64, publisherID int64) error {
	article, err := s.repo.FindArticleByID(articleID)
	if err != nil {
		return AppError{Code: errcode.ErrNotFound, Message: "文章不存在"}
	}
	if article.Status != 3 {
		return AppError{Code: errcode.ErrParam, Message: "仅已审核通过的文章可发布"}
	}

	// Step 1: 分块
	content := article.Answer // v1 model 中 Answer 即正文内容
	chunks := s.chunker.Split(content)
	if len(chunks) == 0 {
		return AppError{Code: errcode.ErrUnknown, Message: "分块结果为空"}
	}

	// Step 2: Embedding
	ctx := context.Background()
	vectors, dimension, err := s.embedder.Embed(ctx, chunks)
	if err != nil {
		return AppError{Code: errcode.ErrRAGUnavailable, Message: "生成向量失败: " + err.Error()}
	}
	if len(vectors) != len(chunks) {
		return AppError{Code: errcode.ErrUnknown, Message: fmt.Sprintf("向量数与分块数不匹配: %d vs %d", len(vectors), len(chunks))}
	}

	// Step 3: 清除旧向量
	if err := s.store.DeleteByArticle(ctx, articleID); err != nil {
		return AppError{Code: errcode.ErrRAGUnavailable, Message: "清除旧向量失败: " + err.Error()}
	}

	// Step 4: 写入新向量
	vc := make([]adapter.VectorChunk, len(chunks))
	for i, chunk := range chunks {
		vc[i] = adapter.VectorChunk{
			ArticleID:       articleID,
			KBID:            article.KBID,
			Content:         chunk,
			ChunkIndex:      i,
			Embedding:       vectors[i],
			EmbeddingModel:  "bge-m3",
			VectorDimension: dimension,
		}
	}

	if err := s.store.BatchInsert(ctx, vc); err != nil {
		return AppError{Code: errcode.ErrRAGUnavailable, Message: "写入向量失败: " + err.Error()}
	}

	// Step 5: 更新文章状态
	article.Status = 4
	article.PublishedBy = &publisherID
	return s.repo.UpdateArticle(article)
}

// =============================================================================
// DisableV2
// =============================================================================

// DisableV2 停用文章，从 pgvector 中删除向量。
func (s *KnowledgeServiceV2) DisableV2(articleID int64) error {
	article, err := s.repo.FindArticleByID(articleID)
	if err != nil {
		return AppError{Code: errcode.ErrNotFound, Message: "文章不存在"}
	}

	// 从 pgvector 删除向量
	ctx := context.Background()
	if err := s.store.DeleteByArticle(ctx, articleID); err != nil {
		return AppError{Code: errcode.ErrRAGUnavailable, Message: "删除向量失败: " + err.Error()}
	}

	// 更新状态
	article.Status = 0
	return s.repo.UpdateArticle(article)
}

// =============================================================================
// EnableV2
// =============================================================================

// EnableV2 恢复已停用文章为草稿状态。
func (s *KnowledgeServiceV2) EnableV2(articleID int64) error {
	article, err := s.repo.FindArticleByID(articleID)
	if err != nil {
		return AppError{Code: errcode.ErrNotFound, Message: "文章不存在"}
	}
	if article.Status != 0 {
		return AppError{Code: errcode.ErrParam, Message: "仅已停用状态的文章可恢复"}
	}

	article.Status = 1 // 已停用 → 草稿
	return s.repo.UpdateArticle(article)
}
