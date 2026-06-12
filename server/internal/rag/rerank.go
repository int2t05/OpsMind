// Package rag 实现自建 RAG 检索引擎。
//
// rerank.go 实现 LLM 驱动的重排序。
//
// 为什么需要重排序：
// 向量检索和 BM25 都是基于词法/语义相似度的"粗排"，
// 可能将不相关但词汇相似的文档排在高位。
// LLM 重排序利用深层语义理解对候选池重新评分，
// 将最相关的文档排在前面，提升最终答案质量。
//
// 降级策略：
// LLM 调用失败时不阻塞管道，保留原始排序返回。
package rag

import (
	"context"
	"fmt"
	"strings"

	"opsmind/internal/adapter"
)

// Rerank 使用 LLM 对候选文档重新排序。
//
// TODO: 当前用 LLM 做重排序，每次消耗 token 且延迟 500ms-2s。
// 应改用交叉编码器 Rerank 模型（如 bge-reranker、Cohere Rerank API），
// 通过调用 Python 推理服务或 SaaS API 实现，成本更低、速度更快。
// candidates 为待重排序的候选文档列表，
// 返回重新排序后的列表（保持相同数量）。
// LLM 调用失败时降级返回原始排序。
func Rerank(ctx context.Context, llm adapter.LLMClient, query string, candidates []RetrievalResult) ([]RetrievalResult, error) {
	if len(candidates) <= 1 {
		return candidates, nil
	}

	// 构造 prompt：告知 LLM 按相关性重新编号
	systemMsg := "你是一个文档排序助手。根据用户查询，对以下候选文档按相关性从高到低重新排序。只输出排序后的文档编号（用逗号分隔），不要添加解释。"

	var docList strings.Builder
	for i, c := range candidates {
		fmt.Fprintf(&docList, "[%d] %s\n", i+1, c.Content)
	}

	resp, err := llm.ChatCompletion(ctx, adapter.ChatRequest{
		Messages: []adapter.ChatMessage{
			{Role: "system", Content: systemMsg},
			{Role: "user", Content: fmt.Sprintf("查询：%s\n\n候选文档：\n%s\n\n请输出排序结果（编号逗号分隔）：", query, docList.String())},
		},
		MaxTokens:   128,
		Temperature: 0.1,
	})
	if err != nil {
		return candidates, nil // 降级：保留原排序
	}

	// 解析 LLM 返回的编号
	order := parseRankOrder(resp.Content, len(candidates))
	if len(order) == 0 {
		return candidates, nil // 解析失败，降级
	}

	// 重新排列
	reranked := make([]RetrievalResult, 0, len(candidates))
	seen := make(map[int64]bool)
	for _, idx := range order {
		if idx >= 0 && idx < len(candidates) && !seen[candidates[idx].ChunkID] {
			reranked = append(reranked, candidates[idx])
			seen[candidates[idx].ChunkID] = true
		}
	}

	// 补充未被 LLM 提到的候选
	for i, c := range candidates {
		if !seen[c.ChunkID] {
			reranked = append(reranked, c)
			// 标记已添加以避免重复（虽然不应该出现）
			_ = i
		}
	}

	if len(reranked) == 0 {
		return candidates, nil
	}
	return reranked, nil
}

// parseRankOrder 从 LLM 响应中解析排序编号。
//
// 支持格式："3,1,2"、"[3,1,2]"、"3 1 2" 等。
func parseRankOrder(response string, maxIdx int) []int {
	// 清理响应
	s := strings.TrimSpace(response)
	s = strings.Trim(s, "[]")

	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\n'
	})

	var order []int
	for _, p := range parts {
		var idx int
		if _, err := fmt.Sscanf(p, "%d", &idx); err == nil && idx >= 1 && idx <= maxIdx {
			order = append(order, idx-1) // 转为 0-based
		}
	}
	return order
}
