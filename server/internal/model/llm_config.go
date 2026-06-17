// Package model 定义 GORM 数据模型。
//
// llm_config.go 定义 LLM 配置模型（表 llm_configs），管理 LLM 和 Embedding 的连接参数。
//
// 设计决策：LLM 和 Embedding 各自拥有独立的 Base URL。
// 虽然它们通常指向同一服务（如 llama.cpp server），但以下场景需要拆分：
//   - 使用 OpenAI 做 LLM 生成 + 本地部署 bge-m3 做 Embedding
//   - 使用 DeepSeek API 做 LLM 生成 + Moonshot API 做 Embedding
// EmbeddingBaseURL 为空时回退到 BaseURL（保持向后兼容）。
//
// 提供商类型仅支持两种：
//   1 = llama.cpp（本地部署，无需 API Key）
//   2 = OpenAI-compatible API（OpenAI / DeepSeek / Moonshot 等）
package model

import "time"

// LlmConfig LLM/Embedding 提供商配置。
type LlmConfig struct {
	ID               int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name             string    `gorm:"type:varchar(128);not null" json:"name"`
	ProviderType     int16     `gorm:"not null;default:1;column:provider_type" json:"provider_type"` // 1=llama.cpp, 2=OpenAI-compatible
	BaseURL          string    `gorm:"type:varchar(512);not null;column:base_url" json:"base_url"`
	EmbeddingBaseURL string    `gorm:"type:varchar(512);column:embedding_base_url" json:"embedding_base_url"` // Embedding 独立地址，空则回退到 BaseURL
	// TODO(model/llm_config): api_key 应 AES-256 加密存储，当前明文落库。
	// 当前模型是明文字段，需在 Repository/Service 层加入加解密或使用数据库加密方案。
	APIKey           string    `gorm:"type:varchar(512);column:api_key" json:"api_key"`
	LLMModel         string    `gorm:"type:varchar(128);not null;column:llm_model" json:"llm_model"`
	EmbeddingModel   string    `gorm:"type:varchar(128);not null;column:embedding_model" json:"embedding_model"`
	MaxTokens        int       `gorm:"not null;default:8192;column:max_tokens" json:"max_tokens"`
	VectorDimension  int       `gorm:"not null;default:1024;column:vector_dimension" json:"vector_dimension"`
	IsDefault        bool      `gorm:"not null;default:false;column:is_default" json:"is_default"`
	// TODO(model/llm_config): 为 is_default=true 增加部分唯一索引。
	// PostgreSQL 可用 CREATE UNIQUE INDEX ... WHERE is_default，彻底防并发双默认。
	CreatedAt        time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt        time.Time `gorm:"not null" json:"updated_at"`
}

// TableName 指定表名。
func (LlmConfig) TableName() string { return "llm_configs" }

// GetEmbeddingBaseURL 返回 Embedding 服务地址，空时回退到 LLM BaseURL。
func (c *LlmConfig) GetEmbeddingBaseURL() string {
	if c.EmbeddingBaseURL != "" {
		return c.EmbeddingBaseURL
	}
	return c.BaseURL
}
