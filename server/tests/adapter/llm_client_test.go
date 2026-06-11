// Package adapter_test 测试 LLMClient 适配器。
//
// 使用 mock HTTP server 验证 OpenAI-compatible 协议的同步/流式调用。
// 所有测试不依赖外部 LLM 服务。
package adapter_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"opsmind/internal/adapter"
)

// mockOpenAIServer 创建模拟 OpenAI-compatible API 的 HTTP 测试服务器。
//
// chatHandler: 处理 /v1/chat/completions 请求（同步）
func mockOpenAIServer(chatHandler http.HandlerFunc) *httptest.Server {
	mux := http.NewServeMux()
	if chatHandler != nil {
		mux.HandleFunc("/v1/chat/completions", chatHandler)
	}
	return httptest.NewServer(mux)
}

// =============================================================================
// 同步 ChatCompletion 测试
// =============================================================================

func TestChatCompletion_Success(t *testing.T) {
	// 模拟返回标准 OpenAI chat completion 响应
	server := mockOpenAIServer(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求方法和 Content-Type
		if r.Method != http.MethodPost {
			t.Errorf("期望 POST, 实际 %s", r.Method)
		}

		var req adapter.ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("请求体解析失败: %v", err)
		}
		if req.Model == "" {
			t.Error("Model 不应为空")
		}

		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]string{
						"role":    "assistant",
						"content": "账号冻结的处理步骤如下：1. 确认账号归属；2. 联系管理员。",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]int{
				"total_tokens": 45,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	client := adapter.NewOpenAIClient(server.URL, "test-key", 10*time.Second)

	resp, err := client.ChatCompletion(context.Background(), adapter.ChatRequest{
		Model: "qwen3-4b",
		Messages: []adapter.ChatMessage{
			{Role: "user", Content: "账号冻结怎么处理？"},
		},
		MaxTokens:   8192,
		Temperature: 0.7,
	})
	if err != nil {
		t.Fatalf("ChatCompletion 失败: %v", err)
	}

	expectedContent := "账号冻结的处理步骤如下"
	if !strings.Contains(resp.Content, expectedContent) {
		t.Errorf("响应内容应包含 %q, 实际 %q", expectedContent, resp.Content)
	}
	if resp.FinishReason != "stop" {
		t.Errorf("FinishReason 期望 stop, 实际 %s", resp.FinishReason)
	}
}

func TestChatCompletion_HTTPError(t *testing.T) {
	server := mockOpenAIServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": {"message": "Invalid API key"}}`))
	})
	defer server.Close()

	client := adapter.NewOpenAIClient(server.URL, "bad-key", 10*time.Second)

	_, err := client.ChatCompletion(context.Background(), adapter.ChatRequest{
		Model:    "qwen3-4b",
		Messages: []adapter.ChatMessage{{Role: "user", Content: "test"}},
	})
	if err == nil {
		t.Error("HTTP 401 应返回错误, 实际 nil")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("错误信息应包含 HTTP 状态码 401, 实际: %v", err)
	}
}

func TestChatCompletion_Timeout(t *testing.T) {
	server := mockOpenAIServer(func(w http.ResponseWriter, r *http.Request) {
		// 模拟超时：永远不返回
		time.Sleep(2 * time.Second)
	})
	defer server.Close()

	client := adapter.NewOpenAIClient(server.URL, "test-key", 500*time.Millisecond)

	_, err := client.ChatCompletion(context.Background(), adapter.ChatRequest{
		Model:    "qwen3-4b",
		Messages: []adapter.ChatMessage{{Role: "user", Content: "test"}},
	})
	if err == nil {
		t.Error("超时应返回错误, 实际 nil")
	}
}

func TestChatCompletion_ContextCancellation(t *testing.T) {
	server := mockOpenAIServer(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	})
	defer server.Close()

	client := adapter.NewOpenAIClient(server.URL, "test-key", 30*time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	_, err := client.ChatCompletion(ctx, adapter.ChatRequest{
		Model:    "qwen3-4b",
		Messages: []adapter.ChatMessage{{Role: "user", Content: "test"}},
	})
	if err == nil {
		t.Error("context 取消应返回错误, 实际 nil")
	}
}

// =============================================================================
// 流式 ChatCompletionStream 测试
// =============================================================================

func TestChatCompletionStream_Success(t *testing.T) {
	// 模拟 SSE 流式响应
	server := mockOpenAIServer(func(w http.ResponseWriter, r *http.Request) {
		// 验证 stream: true 参数
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)
		if stream, ok := req["stream"].(bool); !ok || !stream {
			t.Error("流式请求应设置 stream: true")
		}

		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("ResponseWriter 不支持 Flusher")
		}

		// 逐 token 发送 SSE 事件
		tokens := []string{"账", "号", "冻", "结", "的", "处", "理", "步", "骤"}
		for _, tok := range tokens {
			chunk := map[string]interface{}{
				"choices": []map[string]interface{}{
					{
						"delta": map[string]string{
							"content": tok,
						},
						"index": 0,
					},
				},
			}
			data, _ := json.Marshal(chunk)
			w.Write([]byte("data: " + string(data) + "\n\n"))
			flusher.Flush()
		}

		// 发送结束标记
		w.Write([]byte("data: [DONE]\n\n"))
		flusher.Flush()
	})
	defer server.Close()

	client := adapter.NewOpenAIClient(server.URL, "test-key", 10*time.Second)

	ch, err := client.ChatCompletionStream(context.Background(), adapter.ChatRequest{
		Model:    "qwen3-4b",
		Messages: []adapter.ChatMessage{{Role: "user", Content: "账号冻结怎么处理？"}},
		Stream:   true,
	})
	if err != nil {
		t.Fatalf("ChatCompletionStream 失败: %v", err)
	}

	var fullContent strings.Builder
	for chunk := range ch {
		if chunk.Error != nil {
			t.Fatalf("流式 chunk 错误: %v", chunk.Error)
		}
		fullContent.WriteString(chunk.Content)
	}

	result := fullContent.String()
	if result != "账号冻结的处理步骤" {
		t.Errorf("流式拼接结果期望 %q, 实际 %q", "账号冻结的处理步骤", result)
	}
}

func TestChatCompletionStream_ClientDisconnect(t *testing.T) {
	// 模拟客户端中途断开连接
	server := mockOpenAIServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)

		// 发送第一个 token
		chunk := map[string]interface{}{
			"choices": []map[string]interface{}{
				{"delta": map[string]string{"content": "A"}, "index": 0},
			},
		}
		data, _ := json.Marshal(chunk)
		w.Write([]byte("data: " + string(data) + "\n\n"))
		flusher.Flush()

		// 客户端断开后，ctx 取消，服务器应该停止
		time.Sleep(5 * time.Second) // 模拟长时间等待（实际 ctx 取消会中断）
	})
	defer server.Close()

	client := adapter.NewOpenAIClient(server.URL, "test-key", 30*time.Second)
	ctx, cancel := context.WithCancel(context.Background())

	ch, err := client.ChatCompletionStream(ctx, adapter.ChatRequest{
		Model:  "qwen3-4b",
		Messages: []adapter.ChatMessage{{Role: "user", Content: "test"}},
		Stream: true,
	})
	if err != nil {
		t.Fatalf("ChatCompletionStream 初始化失败: %v", err)
	}

	// 读取第一个 token
	firstChunk := <-ch
	if firstChunk.Error != nil {
		t.Fatalf("第一个 chunk 错误: %v", firstChunk.Error)
	}

	// 立即取消 context（模拟客户端断开）
	cancel()

	// channel 应该在 context 取消后关闭
	// 尝试读取直到 channel 关闭
	for range ch {
		// drain channel
	}

	t.Log("context 取消后 channel 正确关闭")
}
