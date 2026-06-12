// Package adapter_test 测试 EmbeddingClient 适配器。
//
// 使用 mock HTTP server 验证 OpenAI-compatible /embeddings 接口。
// 所有测试不依赖外部 Embedding 服务。
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

// mockEmbeddingServer 创建模拟 OpenAI-compatible embeddings API 的 HTTP 测试服务器。
func mockEmbeddingServer(handler http.HandlerFunc) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/embeddings", handler)
	return httptest.NewServer(mux)
}

// =============================================================================
// 测试用例
// =============================================================================

func TestCreateEmbeddings_Single(t *testing.T) {
	server := mockEmbeddingServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("期望 POST, 实际 %s", r.Method)
		}

		var req adapter.EmbeddingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("请求体解析失败: %v", err)
		}
		if req.Model == "" {
			t.Error("Model 不应为空")
		}
		if len(req.Input) == 0 {
			t.Error("Input 不应为空")
		}

		// 返回 1024 维向量（bge-m3）
		embedding := make([]float32, 1024)
		embedding[0] = 0.1
		embedding[1] = 0.2
		embedding[1023] = 0.9

		resp := map[string]interface{}{
			"object": "list",
			"data": []map[string]interface{}{
				{
					"object":    "embedding",
					"index":     0,
					"embedding": embedding,
				},
			},
			"model": req.Model,
			"usage": map[string]int{
				"total_tokens": 5,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	client := adapter.NewOpenAIEmbeddingClient(server.URL, "test-key", "", 10*time.Second)

	resp, err := client.CreateEmbeddings(context.Background(), adapter.EmbeddingRequest{
		Model: "bge-m3",
		Input: []string{"如何重置 VPN 密码？"},
	})
	if err != nil {
		t.Fatalf("CreateEmbeddings 失败: %v", err)
	}

	if len(resp.Embeddings) != 1 {
		t.Fatalf("期望 1 个 embedding, 实际 %d", len(resp.Embeddings))
	}
	if len(resp.Embeddings[0]) != 1024 {
		t.Errorf("期望向量维度 1024, 实际 %d", len(resp.Embeddings[0]))
	}
	if resp.Dimension != 1024 {
		t.Errorf("Dimension 期望 1024, 实际 %d", resp.Dimension)
	}
	if resp.Embeddings[0][0] != 0.1 || resp.Embeddings[0][1023] != 0.9 {
		t.Error("向量值与期望不符")
	}
}

func TestCreateEmbeddings_Batch(t *testing.T) {
	server := mockEmbeddingServer(func(w http.ResponseWriter, r *http.Request) {
		var req adapter.EmbeddingRequest
		json.NewDecoder(r.Body).Decode(&req)

		// 验证批量输入
		if len(req.Input) != 3 {
			t.Errorf("期望 3 条输入, 实际 %d", len(req.Input))
		}

		embeddings := make([][]float32, len(req.Input))
		for i := range embeddings {
			emb := make([]float32, 1536)
			emb[0] = float32(i) * 0.1
			embeddings[i] = emb
		}

		resp := map[string]interface{}{
			"object": "list",
			"data":   buildEmbeddingData(embeddings),
			"model":  req.Model,
			"usage":  map[string]int{"total_tokens": 15},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	client := adapter.NewOpenAIEmbeddingClient(server.URL, "test-key", "", 10*time.Second)

	resp, err := client.CreateEmbeddings(context.Background(), adapter.EmbeddingRequest{
		Model: "text-embedding-3-small",
		Input: []string{"文本A", "文本B", "文本C"},
	})
	if err != nil {
		t.Fatalf("批量 CreateEmbeddings 失败: %v", err)
	}

	if len(resp.Embeddings) != 3 {
		t.Fatalf("期望 3 个 embedding, 实际 %d", len(resp.Embeddings))
	}
	if resp.Dimension != 1536 {
		t.Errorf("Dimension 期望 1536, 实际 %d", resp.Dimension)
	}
}

func TestCreateEmbeddings_DimensionValidation(t *testing.T) {
	// 验证返回维度与输入模型的一致性
	server := mockEmbeddingServer(func(w http.ResponseWriter, r *http.Request) {
		var req adapter.EmbeddingRequest
		json.NewDecoder(r.Body).Decode(&req)

		var dim int
		if req.Model == "bge-m3" {
			dim = 1024
		} else if req.Model == "text-embedding-3-small" {
			dim = 1536
		} else {
			dim = 768
		}

		emb := make([]float32, dim)
		resp := map[string]interface{}{
			"object": "list",
			"data":   buildEmbeddingData([][]float32{emb}),
			"model":  req.Model,
			"usage":  map[string]int{"total_tokens": 5},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	client := adapter.NewOpenAIEmbeddingClient(server.URL, "test-key", "", 10*time.Second)

	// bge-m3 → 1024 维
	resp1, err1 := client.CreateEmbeddings(context.Background(), adapter.EmbeddingRequest{
		Model: "bge-m3",
		Input: []string{"test"},
	})
	if err1 != nil {
		t.Fatalf("bge-m3 CreateEmbeddings 失败: %v", err1)
	}
	if resp1.Dimension != 1024 {
		t.Errorf("bge-m3 维期望 1024, 实际 %d", resp1.Dimension)
	}

	// text-embedding-3-small → 1536 维
	resp2, err2 := client.CreateEmbeddings(context.Background(), adapter.EmbeddingRequest{
		Model: "text-embedding-3-small",
		Input: []string{"test"},
	})
	if err2 != nil {
		t.Fatalf("text-embedding-3-small CreateEmbeddings 失败: %v", err2)
	}
	if resp2.Dimension != 1536 {
		t.Errorf("text-embedding-3-small 维期望 1536, 实际 %d", resp2.Dimension)
	}
}

func TestCreateEmbeddings_HTTPError(t *testing.T) {
	server := mockEmbeddingServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error": {"message": "Rate limit exceeded"}}`))
	})
	defer server.Close()

	client := adapter.NewOpenAIEmbeddingClient(server.URL, "test-key", "", 10*time.Second)

	_, err := client.CreateEmbeddings(context.Background(), adapter.EmbeddingRequest{
		Model: "bge-m3",
		Input: []string{"test"},
	})
	if err == nil {
		t.Error("HTTP 429 应返回错误, 实际 nil")
	}
	if !strings.Contains(err.Error(), "429") {
		t.Errorf("错误信息应包含 429, 实际: %v", err)
	}
}

func TestCreateEmbeddings_EmptyInput(t *testing.T) {
	server := mockEmbeddingServer(func(w http.ResponseWriter, r *http.Request) {
		// 即使 API 返回空（不应该），客户端应该能处理
		resp := map[string]interface{}{
			"object": "list",
			"data":   []interface{}{},
			"model":  "bge-m3",
			"usage":  map[string]int{"total_tokens": 0},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	client := adapter.NewOpenAIEmbeddingClient(server.URL, "test-key", "", 10*time.Second)

	resp, err := client.CreateEmbeddings(context.Background(), adapter.EmbeddingRequest{
		Model: "bge-m3",
		Input: []string{},
	})
	if err != nil {
		t.Fatalf("空输入应成功, 实际错误: %v", err)
	}
	if resp.Dimension != 0 {
		t.Errorf("空输入 Dimension 期望 0, 实际 %d", resp.Dimension)
	}
}

// =============================================================================
// 辅助函数
// =============================================================================

func buildEmbeddingData(embeddings [][]float32) []map[string]interface{} {
	data := make([]map[string]interface{}, len(embeddings))
	for i, emb := range embeddings {
		data[i] = map[string]interface{}{
			"object":    "embedding",
			"index":     i,
			"embedding": emb,
		}
	}
	return data
}
