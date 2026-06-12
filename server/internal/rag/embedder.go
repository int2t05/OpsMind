// Package rag 实现自建 RAG 检索引擎。
//
// embedder.go 实现批量文本向量化。
//
// 为什么需要批量分页：
// Embedding API 通常有单次请求的文本数量限制（如 OpenAI 限制 2048 tokens），
// 大批量文本需要拆分为多个小批次调用。
// batchSize=20 是经验值——在减少 API 往返次数和单次请求大小之间取得平衡。
package rag

import (
	"context"
	"fmt"

	"opsmind/internal/adapter"
)

// Embedder 批量文本向量化器。
//
// 封装 EmbeddingClient，自动分批调用 + 部分失败处理。
type Embedder struct {
	client    adapter.EmbeddingClient
	batchSize int
}

// NewEmbedder 创建 Embedder 实例。
//
// client 为 OpenAI-compatible Embedding 客户端。
// batchSize 控制每批最大文本数，建议 20。
func NewEmbedder(client adapter.EmbeddingClient, batchSize int) *Embedder {
	// TODO(rag/embedder): client 为 nil 时应返回错误或禁用 Embedder。
	// 当前 Embed 调用 nil client 会 panic，启动期无法发现装配问题。
	if batchSize <= 0 {
		batchSize = 20
	}
	return &Embedder{
		client:    client,
		batchSize: batchSize,
	}
}

// Embed 将文本列表批量转换为向量。
//
// 返回的向量列表顺序与输入 texts 一致。
// 如果某批次调用失败，会跳过该批次继续处理后续批次，
// 最终返回所有成功批次的向量合并结果（不报错）。
// 当全部批次失败时返回 error，调用方据此降级处理。
//
// 空输入返回空向量列表且 dimension=0。
func (e *Embedder) Embed(ctx context.Context, texts []string) ([][]float32, int, error) {
	// TODO(rag/embedder): 部分批次失败时跳过会破坏向量与文本的索引对应关系。
	// 调用方通常要求 len(vectors)==len(texts)，应返回带批次位置的错误，而不是静默少返回。
	if len(texts) == 0 {
		return nil, 0, nil
	}

	var (
		allVectors [][]float32
		dimension  int
		failed     int
	)

	for i := 0; i < len(texts); i += e.batchSize {
		end := i + e.batchSize
		if end > len(texts) {
			end = len(texts)
		}
		batch := texts[i:end]

		resp, err := e.client.CreateEmbeddings(ctx, adapter.EmbeddingRequest{
			Model: "", // 空字符串表示使用 EmbeddingClient 配置的默认模型
			Input: batch,
		})
		if err != nil {
			failed += len(batch)
			// 部分批次失败时跳过，全部失败时返回 error
			continue
		}

		if dimension == 0 && resp.Dimension > 0 {
			dimension = resp.Dimension
		}
		allVectors = append(allVectors, resp.Embeddings...)
	}

	if len(allVectors) == 0 && len(texts) > 0 {
		return nil, 0, fmt.Errorf("全部 %d 个批次 embedding 均失败 (失败文本数=%d)", (len(texts)+e.batchSize-1)/e.batchSize, failed)
	}

	return allVectors, dimension, nil
}
