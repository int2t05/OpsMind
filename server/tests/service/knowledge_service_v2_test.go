package service_test

import (
	"context"
	"fmt"
	"testing"

	"opsmind/internal/adapter"
	"opsmind/internal/model"
	"opsmind/internal/service"
)

// =============================================================================
// v2 mocks
// =============================================================================

type mockKnowledgeRepoV2 struct {
	kbs      map[int64]*model.KnowledgeBase
	articles map[int64]*model.KnowledgeArticle
}

func (m *mockKnowledgeRepoV2) FindKBByID(id int64) (*model.KnowledgeBase, error) {
	if kb, ok := m.kbs[id]; ok {
		return kb, nil
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockKnowledgeRepoV2) FindArticleByID(id int64) (*model.KnowledgeArticle, error) {
	if a, ok := m.articles[id]; ok {
		return a, nil
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockKnowledgeRepoV2) CreateArticle(article *model.KnowledgeArticle) error {
	article.ID = int64(len(m.articles) + 100)
	m.articles[article.ID] = article
	return nil
}

func (m *mockKnowledgeRepoV2) UpdateArticle(article *model.KnowledgeArticle) error {
	m.articles[article.ID] = article
	return nil
}

func (m *mockKnowledgeRepoV2) UpdateArticleStatus(id int64, status int) error {
	if a, ok := m.articles[id]; ok {
		a.Status = int16(status)
		return nil
	}
	return fmt.Errorf("not found")
}

func (m *mockKnowledgeRepoV2) CreateKB(kb *model.KnowledgeBase) error {
	kb.ID = int64(len(m.kbs) + 1)
	m.kbs[kb.ID] = kb
	return nil
}

func (m *mockKnowledgeRepoV2) UpdateKB(kb *model.KnowledgeBase) error {
	m.kbs[kb.ID] = kb
	return nil
}

func (m *mockKnowledgeRepoV2) ListKBs() ([]model.KnowledgeBase, error) {
	var result []model.KnowledgeBase
	for _, kb := range m.kbs {
		result = append(result, *kb)
	}
	return result, nil
}

func (m *mockKnowledgeRepoV2) ListArticles(kbID int64, status int, page, pageSize int) ([]model.KnowledgeArticle, int64, error) {
	var result []model.KnowledgeArticle
	for _, a := range m.articles {
		if a.KBID == kbID && (status == -1 || int(a.Status) == status) {
			result = append(result, *a)
		}
	}
	return result, int64(len(result)), nil
}

func (m *mockKnowledgeRepoV2) FindChunksByArticleID(articleID int64) ([]model.KnowledgeChunk, error) {
	return nil, nil
}

type mockChunkerV2 struct {
	chunks []string
}

func (m *mockChunkerV2) Split(text string) []string {
	return m.chunks
}

type mockEmbedderV2 struct {
	vectors   [][]float32
	dimension int
	err       error
}

func (m *mockEmbedderV2) Embed(ctx context.Context, texts []string) ([][]float32, int, error) {
	return m.vectors, m.dimension, m.err
}

type mockVectorStoreV2 struct {
	inserted   []adapter.VectorChunk
	deletedIDs []int64
}

func (m *mockVectorStoreV2) BatchInsert(ctx context.Context, chunks []adapter.VectorChunk) error {
	m.inserted = chunks
	return nil
}
func (m *mockVectorStoreV2) DeleteByArticle(ctx context.Context, articleID int64) error {
	m.deletedIDs = append(m.deletedIDs, articleID)
	return nil
}
func (m *mockVectorStoreV2) CosineSearch(ctx context.Context, kbID int64, embedding []float32, topK int) ([]adapter.SearchResult, error) {
	return nil, nil
}
func (m *mockVectorStoreV2) DeleteByKB(ctx context.Context, kbID int64) error { return nil }
func (m *mockVectorStoreV2) CountByKB(ctx context.Context, kbID int64) (int64, error) {
	return 0, nil
}
func (m *mockVectorStoreV2) GetChunksByArticle(ctx context.Context, articleID int64) ([]adapter.ChunkContent, error) {
	return nil, nil
}

// =============================================================================
// 测试用例
// =============================================================================

// TestKnowledgeV2_Publish 验证 Publish 走 Chunker→Embedder→VectorStore 流程。
func TestKnowledgeV2_Publish(t *testing.T) {
	repo := &mockKnowledgeRepoV2{
		kbs: map[int64]*model.KnowledgeBase{
			1: {ID: 1, Name: "测试KB"},
		},
		articles: map[int64]*model.KnowledgeArticle{
			10: {
				ID: 10, KBID: 1, Question: "VPN配置", Answer: "VPN配置步骤...",
				Status: 3, // 已通过
			},
		},
	}
	chunker := &mockChunkerV2{chunks: []string{"VPN配置步骤1", "VPN配置步骤2"}}
	embedder := &mockEmbedderV2{
		vectors:   [][]float32{{0.1, 0.2}, {0.3, 0.4}},
		dimension: 2,
	}
	store := &mockVectorStoreV2{}

	svc := service.NewKnowledgeService(nil, chunker, embedder, store, nil, nil)
	// 注入 repo（绕过 interface{} 构造器的限制）
	svc2 := service.NewKnowledgeService(repo, chunker, embedder, store, nil, nil)

	_ = svc
	err := svc2.Publish(10, 1)
	if err != nil {
		t.Fatalf("Publish 失败: %v", err)
	}

	// 向量应已写入
	if len(store.inserted) != 2 {
		t.Errorf("期望写入 2 个向量, 实际 %d", len(store.inserted))
	}

	// 旧向量应先被清理
	if len(store.deletedIDs) == 0 || store.deletedIDs[0] != 10 {
		t.Errorf("期望先清理 article 10 的旧向量, 实际 %v", store.deletedIDs)
	}
}

// TestKnowledgeV2_Disable 验证停用时删除向量。
func TestKnowledgeV2_Disable(t *testing.T) {
	repo := &mockKnowledgeRepoV2{
		articles: map[int64]*model.KnowledgeArticle{
			20: {ID: 20, KBID: 1, Question: "旧文档", Answer: "...", Status: 4}, // 已发布
		},
	}
	store := &mockVectorStoreV2{}

	svc := service.NewKnowledgeService(repo, nil, nil, store, nil, nil)

	err := svc.Disable(20)
	if err != nil {
		t.Fatalf("Disable 失败: %v", err)
	}

	if len(store.deletedIDs) != 1 || store.deletedIDs[0] != 20 {
		t.Errorf("期望删除 article 20 的向量")
	}
}

// TestKnowledgeV2_Enable 验证启用后重置为草稿状态。
func TestKnowledgeV2_Enable(t *testing.T) {
	repo := &mockKnowledgeRepoV2{
		articles: map[int64]*model.KnowledgeArticle{
			30: {ID: 30, KBID: 1, Question: "旧文档", Answer: "...", Status: 0}, // 已停用
		},
	}

	svc := service.NewKnowledgeService(repo, nil, nil, nil, nil, nil)

	err := svc.Enable(30)
	if err != nil {
		t.Fatalf("Enable 失败: %v", err)
	}

	if repo.articles[30].Status != 1 { // 草稿
		t.Errorf("启用后期望 status=1(草稿), 实际 %d", repo.articles[30].Status)
	}
}

// TestKnowledgeV2_PublishNotApproved 验证非已通过状态不可发布。
func TestKnowledgeV2_PublishNotApproved(t *testing.T) {
	repo := &mockKnowledgeRepoV2{
		articles: map[int64]*model.KnowledgeArticle{
			40: {ID: 40, KBID: 1, Question: "草稿", Answer: "...", Status: 1}, // 草稿
		},
	}

	svc := service.NewKnowledgeService(repo, nil, nil, nil, nil, nil)

	err := svc.Publish(40, 1)
	if err == nil {
		t.Error("草稿状态不可发布，应返回错误")
	}
}
