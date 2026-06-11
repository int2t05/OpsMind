// Package adapter 提供外部服务的适配层。
//
// llm_client.go 定义 LLMClient 接口和 OpenAI-compatible HTTP 实现。
// 所有 LLM 调用（文本生成、流式输出）必须通过此适配层，禁止直接 HTTP 调用。
//
// 接口设计决策（ADR-V2-002）：
// ChatCompletion 和 ChatCompletionStream 是两个独立方法，不通过参数切换。
// 调用方在编译时就知道自己需要流式还是非流式，分离方法比运行时判断更清晰。
package adapter

import (
	"bufio"
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

// LLMClient 定义 LLM 调用接口（OpenAI-compatible 协议）。
//
// 支持任意 OpenAI-compatible API：
//   - llama.cpp server    → http://llama-cpp:8080/v1
//   - OpenAI              → https://api.openai.com/v1
//   - DeepSeek / Moonshot → 各服务商地址
type LLMClient interface {
	// ChatCompletion 同步对话 — 用于查询改写、多路路由、重排序等非流式场景。
	ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error)

	// ChatCompletionStream 流式对话 — 用于对用户的 SSE 实时回答。
	// 返回 channel 逐 token 输出，调用方通过 range channel 消费。
	// channel 在流式结束后由实现方关闭。
	ChatCompletionStream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error)
}

// =============================================================================
// 请求/响应类型
// =============================================================================

// ChatRequest 对话请求。
type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
	Stream      bool          `json:"stream,omitempty"` // 仅 ChatCompletionStream 使用
}

// ChatMessage 对话消息。
type ChatMessage struct {
	Role    string `json:"role"`    // "system" | "user" | "assistant"
	Content string `json:"content"`
}

// ChatResponse 同步对话响应。
type ChatResponse struct {
	Content      string `json:"content"`       // 完整回复文本
	FinishReason string `json:"finish_reason"` // "stop" | "length"
	TokensUsed   int    `json:"tokens_used"`
}

// StreamChunk SSE 流式的单个 token 块。
type StreamChunk struct {
	Content      string `json:"content"`       // token 文本
	FinishReason string `json:"finish_reason"` // "stop" | "length" | ""（空表示未结束）
	Error        error  `json:"-"`             // 流式传输错误（channel 关闭前发送）
}

// =============================================================================
// OpenAI-compatible 实现
// =============================================================================

// OpenAIClient 实现 LLMClient，对接 OpenAI-compatible API。
//
// 为什么使用标准 net/http 而非第三方 SDK：
// OpenAI-compatible API 足够简单（两个端点），标准库即可满足需求，避免引入额外依赖。
type OpenAIClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewOpenAIClient 创建 OpenAIClient 实例。
func NewOpenAIClient(baseURL, apiKey string, timeout time.Duration) *OpenAIClient {
	return &OpenAIClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// =============================================================================
// ChatCompletion — 同步调用
// =============================================================================

// openAICompletionRequest OpenAI /v1/chat/completions 请求体。
type openAICompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
	Stream      bool          `json:"stream"`
}

// openAICompletionResponse OpenAI /v1/chat/completions 响应体。
type openAICompletionResponse struct {
	Choices []struct {
		Index int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

// ChatCompletion 发送同步对话请求。
func (c *OpenAIClient) ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	body := openAICompletionRequest{
		Model:       req.Model,
		Messages:    req.Messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      false,
	}

	respBody, err := c.doRequest(ctx, "/v1/chat/completions", body)
	if err != nil {
		return nil, err
	}

	var apiResp openAICompletionResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("解析 LLM 响应失败: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("LLM 返回空 choices")
	}

	return &ChatResponse{
		Content:      apiResp.Choices[0].Message.Content,
		FinishReason: apiResp.Choices[0].FinishReason,
		TokensUsed:   apiResp.Usage.TotalTokens,
	}, nil
}

// =============================================================================
// ChatCompletionStream — 流式调用
// =============================================================================

// openAIStreamChunk OpenAI 流式响应的单个 SSE data 块。
type openAIStreamChunk struct {
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

// ChatCompletionStream 发送流式对话请求，返回 token channel。
//
// 为什么使用 buffered channel（容量 100）：
// HTTP 读取 goroutine 将解析后的 token 写入 channel，调用方从 channel 读取。
// buffered channel 避免网络抖动时 reader goroutine 阻塞，减少延迟。
func (c *OpenAIClient) ChatCompletionStream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error) {
	body := openAICompletionRequest{
		Model:       req.Model,
		Messages:    req.Messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      true,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("序列化流式请求失败: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/v1/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("创建流式请求失败: %w", err)
	}
	c.setHeaders(httpReq)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("流式请求 %s 失败: %w", c.baseURL, err)
	}
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("LLM API 返回 HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	ch := make(chan StreamChunk, 100)
	go c.readSSEStream(ctx, resp, ch)

	return ch, nil
}

// readSSEStream 读取 SSE 流式响应，解析 data: 行并通过 channel 发送。
//
// 为什么在 goroutine 中读取而非调用方直接读取 Body：
// 流式读取需要持续占用 goroutine，channel 模式将「网络 IO」和「业务处理」解耦，
// 调用方可以用 range channel 消费 token，同时检测 ctx.Done() 实现断连处理。
//
// 所有 ch <- send 都通过 sendChunk 辅助函数执行，
// 当 ctx 取消或 channel 满且消费者已断开时，goroutine 优雅退出而非永久阻塞。
func (c *OpenAIClient) readSSEStream(ctx context.Context, resp *http.Response, ch chan<- StreamChunk) {
	defer close(ch)
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		// 移除 "data: " 前缀
		data := strings.TrimPrefix(line, "data: ")
		// 流式结束标记
		if data == "[DONE]" {
			return
		}

		var chunk openAIStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			// 解析失败：发送错误 token 并继续（非致命）
			if !sendChunk(ctx, ch, StreamChunk{Error: fmt.Errorf("解析 SSE chunk 失败: %w", err)}) {
				return
			}
			continue
		}

		if len(chunk.Choices) > 0 {
			content := chunk.Choices[0].Delta.Content
			var finishReason string
			if chunk.Choices[0].FinishReason != nil {
				finishReason = *chunk.Choices[0].FinishReason
			}
			if content != "" || finishReason != "" {
				if !sendChunk(ctx, ch, StreamChunk{
					Content:      content,
					FinishReason: finishReason,
				}) {
					return
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		sendChunk(ctx, ch, StreamChunk{Error: fmt.Errorf("读取 SSE 流失败: %w", err)})
	}
}

// sendChunk 安全地向 channel 发送 chunk，ctx 取消时返回 false。
//
// 使用 select 同时监听 ctx.Done() 和 channel send，
// 消费者断开连接时 goroutine 不会永久阻塞在 channel send 上。
func sendChunk(ctx context.Context, ch chan<- StreamChunk, chunk StreamChunk) bool {
	select {
	case <-ctx.Done():
		return false
	case ch <- chunk:
		return true
	}
}

// =============================================================================
// 辅助方法
// =============================================================================

// doRequest 发送 HTTP 请求并返回响应体。
func (c *OpenAIClient) doRequest(ctx context.Context, path string, body interface{}) ([]byte, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("请求 %s 失败: %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("LLM API 返回 HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// setHeaders 设置通用请求头。
func (c *OpenAIClient) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
}
