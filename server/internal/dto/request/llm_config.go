// Package request 定义 API 请求体结构。
//
// llm_config.go 定义 LLM 配置相关请求体。
package request

// CreateLLMConfigRequest 创建 LLM 配置请求。
// TODO: 与 UpdateLLMConfigRequest 字段完全相同 — 重复定义。
// 若 Create/Update 语义无差别可合并为 LLMConfigRequest；若后续可能分化则保留但加注释说明。
type CreateLLMConfigRequest struct {
	Name             string `json:"name" binding:"required"`
	ProviderType     int16  `json:"provider_type" binding:"required"`
	BaseURL          string `json:"base_url" binding:"required"`
	EmbeddingBaseURL string `json:"embedding_base_url"`
	APIKey           string `json:"api_key"`
	LLMModel         string `json:"llm_model" binding:"required"`
	EmbeddingModel   string `json:"embedding_model" binding:"required"`
	MaxTokens        int    `json:"max_tokens"`
	VectorDimension  int    `json:"vector_dimension"`
	IsDefault        bool   `json:"is_default"`
}

// UpdateLLMConfigRequest 更新 LLM 配置请求。
type UpdateLLMConfigRequest struct {
	Name             string `json:"name" binding:"required"`
	ProviderType     int16  `json:"provider_type" binding:"required"`
	BaseURL          string `json:"base_url" binding:"required"`
	EmbeddingBaseURL string `json:"embedding_base_url"`
	APIKey           string `json:"api_key"`
	LLMModel         string `json:"llm_model" binding:"required"`
	EmbeddingModel   string `json:"embedding_model" binding:"required"`
	MaxTokens        int    `json:"max_tokens"`
	VectorDimension  int    `json:"vector_dimension"`
	IsDefault        bool   `json:"is_default"`
}
