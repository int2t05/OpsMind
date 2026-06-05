//go:build integration

package model_test

import (
	"testing"
	"time"

	"opsmind/internal/model"
)

// TestEmbeddingConfig_Fields 验证 EmbeddingConfig 模型字段与 TECH.md §4.2 embedding_configs 表定义一致
func TestEmbeddingConfig_Fields(t *testing.T) {
	now := time.Now()
	ec := model.EmbeddingConfig{
		Name:            "text-embedding-3-small",
		ModelType:       model.EmbeddingTypeAPI,
		APIEndpoint:     "https://api.openai.com/v1/embeddings",
		APIKey:          "sk-xxx",
		LocalPath:       "",
		VectorDimension: 1536,
		IsDefault:       true,
		CreatedAt:       now,
	}
	ec.ID = 1

	if ec.Name != "text-embedding-3-small" {
		t.Errorf("Name = %q, 期望 text-embedding-3-small", ec.Name)
	}
	if ec.ModelType != model.EmbeddingTypeAPI {
		t.Errorf("ModelType = %d, 期望 %d", ec.ModelType, model.EmbeddingTypeAPI)
	}
	if ec.VectorDimension != 1536 {
		t.Errorf("VectorDimension = %d, 期望 1536", ec.VectorDimension)
	}
	if !ec.IsDefault {
		t.Error("IsDefault = false, 期望 true")
	}
	if ec.APIEndpoint != "https://api.openai.com/v1/embeddings" {
		t.Errorf("APIEndpoint = %q, 期望 https://api.openai.com/v1/embeddings", ec.APIEndpoint)
	}
}
