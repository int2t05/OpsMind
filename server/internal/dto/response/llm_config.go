// Package response 定义 API 响应体结构。
//
// llm_config.go 定义 LLM 配置相关响应体。
package response

// LLMConfigResponse LLM 配置列表响应项。
type LLMConfigResponse struct {
	ID              int64  `json:"id"`
	Name            string `json:"name"`
	ProviderType    int16  `json:"provider_type"`
	BaseURL         string `json:"base_url"`
	// TODO: APIKey 脱敏无编译期保障 — 脱敏逻辑在 Service 层，若遗漏则密钥泄露到客户端。
	// 建议：1) 定义 MarshalJSON 方法自动脱敏；2) 或拆分为 ListItem（不含密钥）和 Detail（含密钥）两种响应。
	APIKey          string `json:"api_key"`           // 已脱敏
	LLMModel        string `json:"llm_model"`
	EmbeddingModel  string `json:"embedding_model"`
	MaxTokens       int    `json:"max_tokens"`
	VectorDimension int    `json:"vector_dimension"`
	IsDefault       bool   `json:"is_default"`
}
