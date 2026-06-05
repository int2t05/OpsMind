package config_test

import (
	"path/filepath"
	"testing"

	"opsmind/internal/config"
)

// TestLoad_DefaultValues 验证从 config.yaml 加载默认值
func TestLoad_DefaultValues(t *testing.T) {
	// 使用项目中的 config.yaml
	cfgPath := filepath.Join("..", "..", "internal", "config", "config.yaml")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() 失败: %v", err)
	}

	// 验证 config.yaml 中定义的默认值
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, 期望 8080", cfg.Server.Port)
	}

	if cfg.Server.Mode != "debug" {
		t.Errorf("Server.Mode = %q, 期望 debug", cfg.Server.Mode)
	}

	if cfg.Database.Host != "localhost" {
		t.Errorf("Database.Host = %q, 期望 localhost", cfg.Database.Host)
	}

	if cfg.Database.Port != 5432 {
		t.Errorf("Database.Port = %d, 期望 5432", cfg.Database.Port)
	}

	if cfg.Database.User != "opsmind" {
		t.Errorf("Database.User = %q, 期望 opsmind", cfg.Database.User)
	}

	// 密码通过环境变量设置，config.yaml 中默认为空

	if cfg.Database.DBName != "opsmind" {
		t.Errorf("Database.DBName = %q, 期望 opsmind", cfg.Database.DBName)
	}

	if cfg.Database.SSLMode != "disable" {
		t.Errorf("Database.SSLMode = %q, 期望 disable", cfg.Database.SSLMode)
	}

	if cfg.MinIO.Endpoint != "localhost:9000" {
		t.Errorf("MinIO.Endpoint = %q, 期望 localhost:9000", cfg.MinIO.Endpoint)
	}

	if cfg.MinIO.AccessKey != "minioadmin" {
		t.Errorf("MinIO.AccessKey = %q, 期望 minioadmin", cfg.MinIO.AccessKey)
	}

	if cfg.MinIO.SecretKey != "minioadmin" {
		t.Errorf("MinIO.SecretKey = %q, 期望 minioadmin", cfg.MinIO.SecretKey)
	}

	if cfg.MinIO.UseSSL != false {
		t.Error("MinIO.UseSSL = true, 期望 false")
	}

	if cfg.AnythingLLM.BaseURL != "http://anythingllm:3001/api" {
		t.Errorf("AnythingLLM.BaseURL = %q, 期望 http://anythingllm:3001/api", cfg.AnythingLLM.BaseURL)
	}

	if cfg.AI.ConfidenceThreshold != 0.6 {
		t.Errorf("AI.ConfidenceThreshold = %f, 期望 0.6", cfg.AI.ConfidenceThreshold)
	}

	if cfg.AI.DefaultTopK != 5 {
		t.Errorf("AI.DefaultTopK = %d, 期望 5", cfg.AI.DefaultTopK)
	}
}

// TestLoad_EnvOverride 验证环境变量覆盖配置文件值
func TestLoad_EnvOverride(t *testing.T) {
	// t.Setenv 在测试结束时自动恢复原值，并标记测试不可并行
	t.Setenv("OPSMIND_SERVER_PORT", "9090")
	t.Setenv("OPSMIND_DATABASE_HOST", "remote-host")
	t.Setenv("OPSMIND_DATABASE_PORT", "5433")
	t.Setenv("OPSMIND_ANYTHINGLLM_API_KEY", "test-api-key")
	t.Setenv("OPSMIND_JWT_SECRET", "test-jwt-secret-32chars-long!!!!!")

	cfgPath := filepath.Join("..", "..", "internal", "config", "config.yaml")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() 失败: %v", err)
	}

	if cfg.Server.Port != 9090 {
		t.Errorf("Server.Port = %d, 期望 9090（环境变量覆盖）", cfg.Server.Port)
	}

	if cfg.Database.Host != "remote-host" {
		t.Errorf("Database.Host = %q, 期望 remote-host（环境变量覆盖）", cfg.Database.Host)
	}

	if cfg.Database.Port != 5433 {
		t.Errorf("Database.Port = %d, 期望 5433（环境变量覆盖）", cfg.Database.Port)
	}

	if cfg.AnythingLLM.APIKey != "test-api-key" {
		t.Errorf("AnythingLLM.APIKey = %q, 期望 test-api-key（环境变量覆盖）", cfg.AnythingLLM.APIKey)
	}

	if cfg.JWT.Secret != "test-jwt-secret-32chars-long!!!!!" {
		t.Errorf("JWT.Secret = %q, 期望 test-jwt-secret-32chars-long!!!!!", cfg.JWT.Secret)
	}
}

// TestLoad_StructFields 验证所有配置结构体字段被正确填充
func TestLoad_StructFields(t *testing.T) {
	cfgPath := filepath.Join("..", "..", "internal", "config", "config.yaml")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() 失败: %v", err)
	}

	// 验证 Server 结构体
	if cfg.Server.Port == 0 {
		t.Error("Server.Port 未填充")
	}
	if cfg.Server.Mode == "" {
		t.Error("Server.Mode 未填充")
	}

	// 验证 Database 结构体
	if cfg.Database.Host == "" {
		t.Error("Database.Host 未填充")
	}
	if cfg.Database.Port == 0 {
		t.Error("Database.Port 未填充")
	}
	if cfg.Database.User == "" {
		t.Error("Database.User 未填充")
	}
	if cfg.Database.DBName == "" {
		t.Error("Database.DBName 未填充")
	}
	if cfg.Database.SSLMode == "" {
		t.Error("Database.SSLMode 未填充")
	}

	// 验证 MinIO 结构体
	if cfg.MinIO.Endpoint == "" {
		t.Error("MinIO.Endpoint 未填充")
	}
	if cfg.MinIO.AccessKey == "" {
		t.Error("MinIO.AccessKey 未填充")
	}
	if cfg.MinIO.SecretKey == "" {
		t.Error("MinIO.SecretKey 未填充")
	}

	// 验证 AnythingLLM 结构体
	if cfg.AnythingLLM.BaseURL == "" {
		t.Error("AnythingLLM.BaseURL 未填充")
	}

	// 验证 AI 结构体
	if cfg.AI.ConfidenceThreshold == 0 {
		t.Error("AI.ConfidenceThreshold 未填充")
	}
	if cfg.AI.DefaultTopK == 0 {
		t.Error("AI.DefaultTopK 未填充")
	}
}
