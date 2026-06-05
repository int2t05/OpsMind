package model

import "time"

// EmbeddingConfig Embedding 模型配置表
type EmbeddingConfig struct {
	ID             int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name           string    `gorm:"type:varchar(128);not null" json:"name"`
	ModelType      int16     `gorm:"not null;column:model_type" json:"model_type"`
	APIEndpoint    string    `gorm:"type:varchar(512);column:api_endpoint" json:"api_endpoint"`
	APIKey         string    `gorm:"type:varchar(512);column:api_key" json:"api_key"`
	LocalPath      string    `gorm:"type:varchar(512);column:local_path" json:"local_path"`
	VectorDimension int      `gorm:"not null;column:vector_dimension" json:"vector_dimension"`
	IsDefault      bool      `gorm:"not null;default:false;column:is_default" json:"is_default"`
	CreatedAt      time.Time `gorm:"not null" json:"created_at"`
}

func (EmbeddingConfig) TableName() string { return "embedding_configs" }
