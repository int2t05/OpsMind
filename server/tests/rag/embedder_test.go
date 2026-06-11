package rag_test

import (
	"context"
	"fmt"
	"testing"

	"opsmind/internal/adapter"
	"opsmind/internal/rag"
)

// =============================================================================
// mockEmbeddingClient 模拟 EmbeddingClient 用于测试
// =============================================================================

type mockEmbeddingClient struct {
	embeddings map[string][]float32 // text → embedding
	dimension  int
	failCount  int // 失败调用次数（-1 表示永不失败）
}

func (m *mockEmbeddingClient) CreateEmbeddings(ctx context.Context, req adapter.EmbeddingRequest) (*adapter.EmbeddingResponse, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if m.failCount > 0 {
		m.failCount--
		return nil, fmt.Errorf("模拟 Embedding 服务不可用")
	}
	if m.failCount == -2 {
		return nil, fmt.Errorf("模拟 Embedding 服务永久不可用")
	}

	embeddings := make([][]float32, len(req.Input))
	for i, text := range req.Input {
		if emb, ok := m.embeddings[text]; ok {
			embeddings[i] = emb
		} else {
			// 生成假的向量（用文本哈希）
			emb := make([]float32, m.dimension)
			for j := range emb {
				emb[j] = float32(len(text)%(j+1)) * 0.1
			}
			embeddings[i] = emb
		}
	}
	return &adapter.EmbeddingResponse{
		Embeddings: embeddings,
		Dimension:  m.dimension,
		TokensUsed: len(req.Input) * 5,
	}, nil
}

// =============================================================================
// 测试用例
// =============================================================================

// TestEmbedder_SingleBatch 验证单批次（≤ batchSize）的 embedding 生成。
func TestEmbedder_SingleBatch(t *testing.T) {
	mock := &mockEmbeddingClient{dimension: 1024}
	emb := rag.NewEmbedder(mock, 20) // batchSize=20

	texts := []string{"文本A", "文本B", "文本C"}
	vectors, dimension, err := emb.Embed(context.Background(), texts)
	if err != nil {
		t.Fatalf("Embed 失败: %v", err)
	}
	if len(vectors) != 3 {
		t.Errorf("期望 3 个向量, 实际 %d", len(vectors))
	}
	if dimension != 1024 {
		t.Errorf("维度期望 1024, 实际 %d", dimension)
	}
	for i, v := range vectors {
		if len(v) != 1024 {
			t.Errorf("向量 %d 维度不匹配: %d", i, len(v))
		}
	}
}

// TestEmbedder_MultiBatch 验证多批次分页调用（> batchSize 时自动分批）。
func TestEmbedder_MultiBatch(t *testing.T) {
	mock := &mockEmbeddingClient{dimension: 768}
	emb := rag.NewEmbedder(mock, 10) // batchSize=10

	// 25 条文本 → 应拆为 10+10+5 三批
	texts := make([]string, 25)
	for i := range texts {
		texts[i] = fmt.Sprintf("文本%d", i)
	}

	vectors, dimension, err := emb.Embed(context.Background(), texts)
	if err != nil {
		t.Fatalf("Embed 失败: %v", err)
	}
	if len(vectors) != 25 {
		t.Errorf("期望 25 个向量, 实际 %d", len(vectors))
	}
	if dimension != 768 {
		t.Errorf("维度期望 768, 实际 %d", dimension)
	}
}

// TestEmbedder_PartialFailure 验证部分批次失败时返回成功结果。
func TestEmbedder_PartialFailure(t *testing.T) {
	mock := &mockEmbeddingClient{dimension: 1024, failCount: 1}
	emb := rag.NewEmbedder(mock, 5) // batchSize=5

	// 12 条文本 → 5+5+2 三批，第二批失败
	texts := make([]string, 12)
	for i := range texts {
		texts[i] = fmt.Sprintf("文本%d", i)
	}

	vectors, _, err := emb.Embed(context.Background(), texts)
	if err != nil {
		t.Fatalf("部分失败应继续并返回成功结果: %v", err)
	}
	// 第一批(5) + 第三批(2) = 7 成功，第二批(5) 失败
	if len(vectors) < 5 {
		t.Errorf("至少应有 5 个成功向量（第一批）, 实际 %d", len(vectors))
	}
}

// TestEmbedder_DimensionValidation 验证维度校验。
func TestEmbedder_DimensionValidation(t *testing.T) {
	mock := &mockEmbeddingClient{dimension: 1536}
	emb := rag.NewEmbedder(mock, 20)

	vectors, dimension, err := emb.Embed(context.Background(), []string{"测试"})
	if err != nil {
		t.Fatalf("Embed 失败: %v", err)
	}
	if dimension != 1536 {
		t.Errorf("维度期望 1536, 实际 %d", dimension)
	}
	if len(vectors[0]) != 1536 {
		t.Errorf("向量维度期望 1536, 实际 %d", len(vectors[0]))
	}
}

// TestEmbedder_EmptyInput 验证空输入返回空结果。
func TestEmbedder_EmptyInput(t *testing.T) {
	mock := &mockEmbeddingClient{dimension: 1024}
	emb := rag.NewEmbedder(mock, 20)

	vectors, dimension, err := emb.Embed(context.Background(), []string{})
	if err != nil {
		t.Fatalf("空输入不应报错: %v", err)
	}
	if len(vectors) != 0 {
		t.Errorf("空输入期望 0 个向量, 实际 %d", len(vectors))
	}
	if dimension != 0 {
		t.Errorf("空输入维度期望 0, 实际 %d", dimension)
	}
}

// TestEmbedder_LargeBatch 验证大批量正确分页且所有文本都有对应向量。
func TestEmbedder_LargeBatch(t *testing.T) {
	mock := &mockEmbeddingClient{dimension: 1024}
	emb := rag.NewEmbedder(mock, 20) // batchSize=20

	texts := make([]string, 47)
	for i := range texts {
		texts[i] = fmt.Sprintf("运维文档片段 #%d: 账号冻结处理流程", i)
	}

	vectors, _, err := emb.Embed(context.Background(), texts)
	if err != nil {
		t.Fatalf("Embed 失败: %v", err)
	}
	if len(vectors) != 47 {
		t.Errorf("期望 47 个向量, 实际 %d", len(vectors))
	}
}
