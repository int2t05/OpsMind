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
//
// 空输入返回空向量列表且 dimension=0。
func (e *Embedder) Embed(ctx context.Context, texts []string) ([][]float32, int, error) {
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
			Model: "", // TODO: 模型名应从配置显式传入，而非依赖 EmbeddingClient 底层默认值
			Input: batch,
		})
		if err != nil {
			failed += len(batch)
			// TODO: 部分批次失败时静默跳过，调用方无法知道哪些文本的 embedding 丢失。
			// 应返回失败索引列表或允许调用方配置「失败即中止」策略。
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
