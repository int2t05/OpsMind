// Package adapter 提供外部服务的适配层。
//
// embedding_client.go 定义 EmbeddingClient 接口和 OpenAI-compatible HTTP 实现。
// 所有 Embedding 调用必须通过此适配层，禁止直接 HTTP 调用。
//
// 为什么 Embedding 和 LLM 使用同一 Base URL 但独立接口：
// LLM 和 Embedding 通常指向同一服务（如 llama.cpp server），但
// 调用模式不同（流式 vs 批量）且返回值类型不同（文本 vs []float32），
// 独立接口保持职责清晰。
package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// =============================================================================
// 接口定义
// =============================================================================

// EmbeddingClient 定义 Embedding 生成接口（OpenAI-compatible 协议）。
//
// 支持任意 OpenAI-compatible /v1/embeddings 端点：
//   - llama.cpp server    → http://llama-cpp:8080/v1
//   - OpenAI              → https://api.openai.com/v1
//   - 其他兼容服务
type EmbeddingClient interface {
	// CreateEmbeddings 生成文本向量。
	// 支持批量输入，一次调用传入多个文本，减少 API 往返。
	CreateEmbeddings(ctx context.Context, req EmbeddingRequest) (*EmbeddingResponse, error)
}

// =============================================================================
// 请求/响应类型
// =============================================================================

// EmbeddingRequest embedding 请求。
type EmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// EmbeddingResponse embedding 响应。
type EmbeddingResponse struct {
	Embeddings [][]float32 `json:"embeddings"` // 每个 input 对应一个向量
	Dimension  int         `json:"dimension"`  // 向量维度
	TokensUsed int         `json:"tokens_used"`
}

// =============================================================================
// OpenAI-compatible 实现
// =============================================================================

// OpenAIEmbeddingClient 实现 EmbeddingClient，对接 OpenAI-compatible API。
//
// 为什么复用 OpenAIClient 的 Base URL 设计模式但使用独立结构体：
// Embedding 和 LLM 虽然使用同一协议族，但超时、重试策略可能不同
//（embedding 批量调用需更长超时）。独立结构体允许独立配置。
type OpenAIEmbeddingClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewOpenAIEmbeddingClient 创建 OpenAIEmbeddingClient 实例。
func NewOpenAIEmbeddingClient(baseURL, apiKey string, timeout time.Duration) *OpenAIEmbeddingClient {
	return &OpenAIEmbeddingClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// openAIEmbeddingsRequest OpenAI /v1/embeddings 请求体。
type openAIEmbeddingsRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// openAIEmbeddingsResponse OpenAI /v1/embeddings 响应体。
type openAIEmbeddingsResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

// CreateEmbeddings 调用 /v1/embeddings 生成向量。
func (c *OpenAIEmbeddingClient) CreateEmbeddings(ctx context.Context, req EmbeddingRequest) (*EmbeddingResponse, error) {
	body := openAIEmbeddingsRequest{
		Model: req.Model,
		Input: req.Input,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("序列化 embedding 请求失败: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/v1/embeddings", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("创建 embedding 请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("请求 embedding 服务 %s 失败: %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取 embedding 响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Embedding API 返回 HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp openAIEmbeddingsResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("解析 embedding 响应失败: %w", err)
	}

	embeddings := make([][]float32, len(apiResp.Data))
	for _, d := range apiResp.Data {
		if d.Index < 0 || d.Index >= len(embeddings) {
			return nil, fmt.Errorf("embedding index 越界: %d (总数 %d)", d.Index, len(embeddings))
		}
		if embeddings[d.Index] != nil {
			return nil, fmt.Errorf("embedding index 重复: %d", d.Index)
		}
		embeddings[d.Index] = d.Embedding
	}

	dimension := 0
	if len(embeddings) > 0 && len(embeddings[0]) > 0 {
		dimension = len(embeddings[0])
	}

	return &EmbeddingResponse{
		Embeddings: embeddings,
		Dimension:  dimension,
		TokensUsed: apiResp.Usage.TotalTokens,
	}, nil
}
