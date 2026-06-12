// Package model 定义 GORM 数据模型。
//
// llm_config.go 定义 LLM 配置模型（表 llm_configs），统一管理 llama.cpp 和 OpenAI-compatible API 的连接参数。
package model

import "time"

// LlmConfig LLM/Embedding 提供商配置。
//
// 一行配置同时描述 LLM 和 Embedding 的连接信息，
// 因为两者通常指向同一服务（如 llama.cpp server 或 OpenAI API），
// 只是调用的端点（/v1/chat/completions vs /v1/embeddings）和模型名不同。
type LlmConfig struct {
	ID              int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name            string    `gorm:"type:varchar(128);not null" json:"name"`
	ProviderType    int16     `gorm:"not null;default:1;column:provider_type" json:"provider_type"` // 1=llama.cpp, 2=OpenAI-compatible
	BaseURL         string    `gorm:"type:varchar(512);not null;column:base_url" json:"base_url"`
	APIKey          string    `gorm:"type:varchar(512);column:api_key" json:"api_key"`
	LLMModel        string    `gorm:"type:varchar(128);not null;column:llm_model" json:"llm_model"`
	EmbeddingModel  string    `gorm:"type:varchar(128);not null;column:embedding_model" json:"embedding_model"`
	MaxTokens       int       `gorm:"not null;default:8192;column:max_tokens" json:"max_tokens"`
	VectorDimension int       `gorm:"not null;default:1024;column:vector_dimension" json:"vector_dimension"`
	IsDefault       bool      `gorm:"not null;default:false;column:is_default" json:"is_default"`
	CreatedAt       time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt       time.Time `gorm:"not null" json:"updated_at"`
}

// TableName 指定表名。
func (LlmConfig) TableName() string { return "llm_configs" }
