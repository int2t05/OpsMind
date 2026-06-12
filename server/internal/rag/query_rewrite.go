// Package rag 实现自建 RAG 检索引擎。
//
// query_rewrite.go 实现查询改写（Query Rewrite）。
//
// 为什么需要查询改写：
// 用户的原始查询通常口语化、不完整（如"VPN怎么连"），
// 直接检索可能命中率低。通过 LLM 将口语化查询改写为正式、
// 信息完整的检索查询，显著提升召回率。
//
// 降级策略：
// LLM 调用失败时，不阻塞管道，直接返回原始查询继续后续步骤。
package rag

import (
	"context"
	"fmt"
	"strings"

	"opsmind/internal/adapter"
)

// QueryRewrite 使用 LLM 改写查询为更适合检索的形式。
//
// history 为最近 N 轮对话（每轮含 role/content），用于上下文消歧。
// LLM 调用失败时降级返回原始 query。
func QueryRewrite(ctx context.Context, llm adapter.LLMClient, query string, history []map[string]string) (string, error) {
	// TODO(rag/query_rewrite): llm 为 nil 时应直接降级返回 query。
	// 让辅助步骤天然可选，避免测试或纯检索模式下出现 nil pointer。
	// 构造 prompt
	systemMsg := "你是一个查询改写助手。将用户的口语化问题改写为更适合知识库检索的正式查询。只输出改写后的查询文本，不要添加解释。"
	userMsg := fmt.Sprintf("原始查询：%s", query)

	messages := []adapter.ChatMessage{
		{Role: "system", Content: systemMsg},
	}

	// 添加历史对话（最近 3 轮）
	for _, h := range history {
		role := h["role"]
		content := h["content"]
		if role == "user" || role == "assistant" {
			messages = append(messages, adapter.ChatMessage{Role: role, Content: content})
		}
	}

	messages = append(messages, adapter.ChatMessage{Role: "user", Content: userMsg})

	resp, err := llm.ChatCompletion(ctx, adapter.ChatRequest{
		Messages:    messages,
		MaxTokens:   256,
		Temperature: 0.1, // 低温度保证输出稳定
	})
	if err != nil {
		// 降级：返回原始查询，但上报错误让管道步骤显示失败状态
		return query, fmt.Errorf("查询改写 LLM 调用失败: %w", err)
	}
	result := strings.TrimSpace(resp.Content)
	if result == "" {
		return query, nil
	}
	return result, nil
}
