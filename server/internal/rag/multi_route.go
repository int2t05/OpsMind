// Package rag 实现自建 RAG 检索引擎。
//
// multi_route.go 实现多路检索（Multi-Route Retrieval）。
//
// 为什么需要多路检索：
// 单一查询只能覆盖知识库的一个视角。通过 LLM 从不同角度
// 生成 2-4 个互补子查询（如"VPN 连接"→"VPN客户端配置"、
// "VPN服务器地址"、"VPN证书安装"），多路检索后融合，
// 显著提升查全率。
//
// 降级策略：
// LLM 调用失败时降级为单路检索（仅原始查询）。
package rag

import (
	"context"
	"fmt"
	"strings"

	"opsmind/internal/adapter"
)

// MultiRoute 使用 LLM 从不同角度生成多个子查询。
//
// count 控制生成的子查询数量（2-4 个）。
// LLM 调用失败时降级返回 [query] 单路检索。
func MultiRoute(ctx context.Context, llm adapter.LLMClient, query string, count int) ([]string, error) {
	// TODO(rag/multi_route): count 应限制到 2-4，与 API 文档一致。
	// 过大的 count 会放大检索次数和 RRF 候选池，影响延迟。
	if count <= 0 {
		count = 3
	}

	systemMsg := fmt.Sprintf("你是一个查询扩展助手。从不同角度将用户问题扩展为 %d 个互补的子查询。每行一个子查询，不要编号，不要添加解释。", count)
	userMsg := fmt.Sprintf("原始查询：%s", query)

	resp, err := llm.ChatCompletion(ctx, adapter.ChatRequest{
		Messages: []adapter.ChatMessage{
			{Role: "system", Content: systemMsg},
			{Role: "user", Content: userMsg},
		},
		MaxTokens:   512,
		Temperature: 0.3,
	})
	if err != nil {
		// 降级：仅原始查询
		return []string{query}, nil
	}

	// 解析子查询（按行分割，过滤空行、去重）
	lines := strings.Split(strings.TrimSpace(resp.Content), "\n")
	var routes []string
	seen := make(map[string]bool)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// 去除编号前缀：匹配 "1." "1、" "1) " "- " 等格式
		// TODO: 脆弱的字符串清理 — 依赖 LLM 输出固定编号格式，容易因 LLM 输出变化（如 "Route 1:"、"："）而失败。
		// 应使用正则提取或让 LLM 输出 JSON 格式的子查询数组。
		line = strings.TrimLeft(line, "0123456789")
		line = strings.TrimLeft(line, ".、) -")
		line = strings.TrimSpace(line)
		if line != "" && line != query && !seen[line] {
			seen[line] = true
			routes = append(routes, line)
		}
	}

	if len(routes) == 0 {
		return []string{query}, nil
	}
	return routes, nil
}
