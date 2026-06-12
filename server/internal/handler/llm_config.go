// Package handler 实现 HTTP 请求处理。
//
// llm_config.go 提供 LLM 配置管理 API，统一管理 LLM 和 Embedding 提供商配置。
package handler

import (
	"context"
	"strconv"
	"time"

	"opsmind/internal/adapter"
	"opsmind/internal/dto/request"
	"opsmind/internal/model"
	"opsmind/internal/service"
	"opsmind/pkg/errcode"
	"opsmind/pkg/response"

	"github.com/gin-gonic/gin"
)

// =============================================================================
// LLMConfigHandler
// =============================================================================

// LLMConfigHandler LLM 配置管理接口。
type LLMConfigHandler struct {
	svc       llmConfigService
	llmClient adapter.LLMClient // v2: 用于 TestConnection 验证连接
}

// llmConfigService 定义 Handler 需要的 Service 方法（消费者定义接口）。
type llmConfigService interface {
	CreateConfig(name string, providerType int16, baseURL, embeddingBaseURL, apiKey, llmModel, embeddingModel string, maxTokens, vectorDimension int, isDefault bool) error
	ListConfigs() ([]service.LlmConfigResponse, error)
	GetConfig(id int64) (*model.LlmConfig, error)
	UpdateConfig(cfg *model.LlmConfig) error
	DeleteConfig(id int64) error
	GetManager() *service.LLMConfigManager
}

// NewLLMConfigHandler 创建 LLMConfigHandler 实例。
//
// svc 和 llmClient 通过构造函数注入，避免 Setter 注入的时序风险。
// llmClient 可选（传 nil），用于 TestConnection 真实验证。
func NewLLMConfigHandler(svc llmConfigService, llmClient adapter.LLMClient) *LLMConfigHandler {
	return &LLMConfigHandler{
		svc:       svc,
		llmClient: llmClient,
	}
}

// =============================================================================
// CRUD 端点
// =============================================================================

// ListConfigs 列出全部 LLM 配置。
//
// GET /api/v1/admin/llm-configs
func (h *LLMConfigHandler) ListConfigs(c *gin.Context) {
	configs, err := h.svc.ListConfigs()
	if err != nil {
		// TODO: err.Error() 泄露内部错误 — 应使用 handleServiceError(c, err)。
		handleServiceError(c, err)
		return
	}
	response.Success(c, configs)
}

// CreateConfig 创建 LLM 配置。
//
// POST /api/v1/admin/llm-configs
func (h *LLMConfigHandler) CreateConfig(c *gin.Context) {
	var req request.CreateLLMConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.ErrParam, "参数校验失败: "+err.Error())
		return
	}
	if req.MaxTokens == 0 {
		req.MaxTokens = 8192
	}
	if req.VectorDimension == 0 {
		req.VectorDimension = 1024
	}

	err := h.svc.CreateConfig(req.Name, req.ProviderType, req.BaseURL, req.EmbeddingBaseURL, req.APIKey,
		req.LLMModel, req.EmbeddingModel, req.MaxTokens, req.VectorDimension, req.IsDefault)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, nil)
}

// GetConfig 获取单个 LLM 配置详情。
//
// GET /api/v1/admin/llm-configs/:id
func (h *LLMConfigHandler) GetConfig(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrParam, "无效的配置 ID")
		return
	}

	cfg, err := h.svc.GetConfig(id)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, cfg)
}

// UpdateConfig 更新 LLM 配置。
//
// PUT /api/v1/admin/llm-configs/:id
func (h *LLMConfigHandler) UpdateConfig(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrParam, "无效的配置 ID")
		return
	}

	var req request.UpdateLLMConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.ErrParam, "参数校验失败: "+err.Error())
		return
	}
	if req.MaxTokens == 0 {
		req.MaxTokens = 8192
	}
	if req.VectorDimension == 0 {
		req.VectorDimension = 1024
	}

	cfg := &model.LlmConfig{
		ID:               id,
		Name:             req.Name,
		ProviderType:     req.ProviderType,
		BaseURL:          req.BaseURL,
		EmbeddingBaseURL: req.EmbeddingBaseURL,
		APIKey:           req.APIKey,
		LLMModel:         req.LLMModel,
		EmbeddingModel:   req.EmbeddingModel,
		MaxTokens:        req.MaxTokens,
		VectorDimension:  req.VectorDimension,
		IsDefault:       req.IsDefault,
	}

	if err := h.svc.UpdateConfig(cfg); err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, nil)
}

// DeleteConfig 删除 LLM 配置。
//
// DELETE /api/v1/admin/llm-configs/:id
func (h *LLMConfigHandler) DeleteConfig(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrParam, "无效的配置 ID")
		return
	}

	if err := h.svc.DeleteConfig(id); err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, nil)
}

// TestConnection 测试 LLM 连接。
//
// POST /api/v1/admin/llm-configs/:id/test
func (h *LLMConfigHandler) TestConnection(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrParam, "无效的配置 ID")
		return
	}

	cfg, err := h.svc.GetConfig(id)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	// 测试连接：使用注入的 LLMClient 向配置的 BaseURL 发送 /v1/models 请求
	if h.llmClient == nil {
		response.Error(c, errcode.ErrUnknown, "LLM 客户端未初始化，无法测试连接")
		return
	}

	// 构造一个极小的请求来验证连通性
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	testReq := adapter.ChatRequest{
		Model: cfg.LLMModel,
		Messages: []adapter.ChatMessage{
			{Role: "user", Content: "ping"},
		},
		MaxTokens:   1,
		Temperature: 0,
	}

		start := time.Now()
		resp, err := h.llmClient.ChatCompletion(ctx, testReq)
		latency := time.Since(start).Milliseconds()
		if err != nil {
			response.Error(c, errcode.ErrAIUnavailable, "连接测试失败: "+err.Error())
			return
		}

		response.Success(c, gin.H{
			"success":     true,
			"model":       cfg.LLMModel,
			"latency_ms":  latency,
			"tokens_used": resp.TokensUsed,
		})
}
