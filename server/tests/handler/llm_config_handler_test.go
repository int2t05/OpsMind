package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"opsmind/internal/adapter"
	"opsmind/internal/handler"
	"opsmind/internal/model"
	"opsmind/internal/service"

	"github.com/gin-gonic/gin"
)

// =============================================================================
// Mock LLMConfigService
// =============================================================================

type mockLLMConfigSvc struct {
	configs    []service.LlmConfigResponse
	createFn   func(name string, providerType int16, baseURL, embeddingBaseURL, apiKey, llmModel, embeddingModel string, maxTokens, vectorDimension int, isDefault bool) error
	getConfigFn func(id int64) (*model.LlmConfig, error)
}

func (m *mockLLMConfigSvc) CreateConfig(name string, providerType int16, baseURL, embeddingBaseURL, apiKey, llmModel, embeddingModel string, maxTokens, vectorDimension int, isDefault bool) error {
	if m.createFn != nil {
		return m.createFn(name, providerType, baseURL, embeddingBaseURL, apiKey, llmModel, embeddingModel, maxTokens, vectorDimension, isDefault)
	}
	return nil
}

func (m *mockLLMConfigSvc) ListConfigs() ([]service.LlmConfigResponse, error) {
	return m.configs, nil
}

func (m *mockLLMConfigSvc) GetConfig(id int64) (*model.LlmConfig, error) {
	if m.getConfigFn != nil {
		return m.getConfigFn(id)
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockLLMConfigSvc) UpdateConfig(cfg *model.LlmConfig) error { return nil }
func (m *mockLLMConfigSvc) DeleteConfig(id int64) error              { return nil }
func (m *mockLLMConfigSvc) GetManager() *service.LLMConfigManager    { return nil }

// =============================================================================
// 测试
// =============================================================================

func setupLLMTestRouter(svc *mockLLMConfigSvc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	mockClient := &mockTestLLMClient{response: &adapter.ChatResponse{Content: "ok", FinishReason: "stop"}}
	h := handler.NewLLMConfigHandler(svc, mockClient)
	r.GET("/api/v1/admin/llm-configs", h.ListConfigs)
	r.POST("/api/v1/admin/llm-configs", h.CreateConfig)
	r.GET("/api/v1/admin/llm-configs/:id", h.GetConfig)
	r.PUT("/api/v1/admin/llm-configs/:id", h.UpdateConfig)
	r.DELETE("/api/v1/admin/llm-configs/:id", h.DeleteConfig)
	r.POST("/api/v1/admin/llm-configs/:id/test", h.TestConnection)
	return r
}

// TestLLMConfigHandler_ListConfigs 验证 GET /llm-configs 返回列表。
func TestLLMConfigHandler_ListConfigs(t *testing.T) {
	svc := &mockLLMConfigSvc{
		configs: []service.LlmConfigResponse{
			{ID: 1, Name: "llama.cpp 本地", ProviderType: 1, LLMModel: "qwen3-4b", EmbeddingModel: "bge-m3", IsDefault: true},
			{ID: 2, Name: "OpenAI", ProviderType: 2, APIKey: "sk-****cret", LLMModel: "gpt-4o", EmbeddingModel: "text-embedding-3-small"},
		},
	}

	r := setupLLMTestRouter(svc)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/admin/llm-configs", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("期望 200, 实际 %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["code"].(float64) != 0 {
		t.Errorf("code 期望 0, 实际 %v", resp["code"])
	}
}

// TestLLMConfigHandler_CreateConfig 验证 POST /llm-configs 创建配置。
func TestLLMConfigHandler_CreateConfig(t *testing.T) {
	captured := ""
	svc := &mockLLMConfigSvc{
		createFn: func(name string, providerType int16, baseURL, embeddingBaseURL, apiKey, llmModel, embeddingModel string, maxTokens, vectorDimension int, isDefault bool) error {
			captured = name
			return nil
		},
	}

	r := setupLLMTestRouter(svc)
	body := map[string]interface{}{
		"name":             "DeepSeek",
		"provider_type":    2,
		"base_url":         "https://api.deepseek.com/v1",
		"api_key":          "sk-test123",
		"llm_model":        "deepseek-chat",
		"embedding_model":  "bge-m3",
		"max_tokens":       4096,
		"vector_dimension": 1024,
		"is_default":       false,
	}
	b, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/admin/llm-configs", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("期望 200, 实际 %d: %s", w.Code, w.Body.String())
	}
	if captured != "DeepSeek" {
		t.Errorf("期望 name='DeepSeek', 实际 '%s'", captured)
	}
}

// TestLLMConfigHandler_GetConfig 验证 GET /llm-configs/:id。
func TestLLMConfigHandler_GetConfig(t *testing.T) {
	r := setupLLMTestRouter(&mockLLMConfigSvc{})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/admin/llm-configs/1", nil)
	r.ServeHTTP(w, req)

	// mock 返回 "not found"，应收到 404
	if w.Code != 200 {
		t.Logf("404/错误响应（预期 mock 返回 not found）: %s", w.Body.String())
	}
}

// TestLLMConfigHandler_DeleteConfig 验证 DELETE /llm-configs/:id。
func TestLLMConfigHandler_DeleteConfig(t *testing.T) {
	r := setupLLMTestRouter(&mockLLMConfigSvc{})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/admin/llm-configs/1", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("期望 200, 实际 %d: %s", w.Code, w.Body.String())
	}
}

// TestLLMConfigHandler_TestConnection 验证 POST /llm-configs/:id/test。
func TestLLMConfigHandler_TestConnection(t *testing.T) {
	called := false
	svc := &mockLLMConfigSvc{}
	svc.getConfigFn = func(id int64) (*model.LlmConfig, error) {
		called = true
		return &model.LlmConfig{ID: 1, Name: "test", LLMModel: "qwen3-4b", BaseURL: "http://x:8080/v1"}, nil
	}

	r := setupLLMTestRouter(svc)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/admin/llm-configs/1/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("期望 200, 实际 %d: %s", w.Code, w.Body.String())
	}
	if !called {
		t.Error("应调用 GetConfig 获取配置信息")
	}
}

// mockTestLLMClient 模拟 LLMClient 用于测试连接。
type mockTestLLMClient struct {
	response *adapter.ChatResponse
	err      error
}

func (m *mockTestLLMClient) ChatCompletion(ctx context.Context, req adapter.ChatRequest) (*adapter.ChatResponse, error) {
	return m.response, m.err
}
func (m *mockTestLLMClient) ChatCompletionStream(ctx context.Context, req adapter.ChatRequest) (<-chan adapter.StreamChunk, error) {
	return nil, fmt.Errorf("not impl")
}
