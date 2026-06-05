// Package config 的配置加载测试。
//
// 验证 Viper 能正确读取默认配置和环境变量覆盖。
package config

import (
	"os"
	"testing"
	"time"
)

// TestLoad_DefaultValues 验证从 config.yaml 读取默认值。
func TestLoad_DefaultValues(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load() 返回错误: %v", err)
	}

	// Server 配置
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, 期望 8080", cfg.Server.Port)
	}
	if cfg.Server.Mode != "debug" {
		t.Errorf("Server.Mode = %q, 期望 %q", cfg.Server.Mode, "debug")
	}

	// Database 配置
	if cfg.Database.Host != "localhost" {
		t.Errorf("Database.Host = %q, 期望 %q", cfg.Database.Host, "localhost")
	}
	if cfg.Database.Port != 5432 {
		t.Errorf("Database.Port = %d, 期望 5432", cfg.Database.Port)
	}
	if cfg.Database.User != "opsmind" {
		t.Errorf("Database.User = %q, 期望 %q", cfg.Database.User, "opsmind")
	}
	if cfg.Database.DBName != "opsmind" {
		t.Errorf("Database.DBName = %q, 期望 %q", cfg.Database.DBName, "opsmind")
	}

	// JWT 配置
	if cfg.JWT.AccessExpire != 2*time.Hour {
		t.Errorf("JWT.AccessExpire = %v, 期望 %v", cfg.JWT.AccessExpire, 2*time.Hour)
	}
	if cfg.JWT.RefreshExpire != 168*time.Hour {
		t.Errorf("JWT.RefreshExpire = %v, 期望 %v", cfg.JWT.RefreshExpire, 168*time.Hour)
	}

	// AnythingLLM 配置
	if cfg.AnythingLLM.BaseURL != "http://anythingllm:3001/api" {
		t.Errorf("AnythingLLM.BaseURL = %q, 期望 %q", cfg.AnythingLLM.BaseURL, "http://anythingllm:3001/api")
	}
	if cfg.AnythingLLM.TimeoutSeconds != 20 {
		t.Errorf("AnythingLLM.TimeoutSeconds = %d, 期望 20", cfg.AnythingLLM.TimeoutSeconds)
	}

	// AI 配置
	if cfg.AI.DefaultTopK != 5 {
		t.Errorf("AI.DefaultTopK = %d, 期望 5", cfg.AI.DefaultTopK)
	}
	if cfg.AI.ConfidenceThreshold != 0.6 {
		t.Errorf("AI.ConfidenceThreshold = %f, 期望 0.6", cfg.AI.ConfidenceThreshold)
	}
}

// TestLoad_EnvOverride 验证环境变量覆盖默认配置。
func TestLoad_EnvOverride(t *testing.T) {
	// 设置环境变量（Viper AutomaticEnv 格式：PREFIX_SECTION_KEY）
	os.Setenv("OPSMIND_DATABASE_HOST", "remote-postgres")
	os.Setenv("OPSMIND_DATABASE_PASSWORD", "secret123")
	os.Setenv("OPSMIND_ANYTHINGLLM_BASE_URL", "http://custom-llm:3001/api")
	os.Setenv("OPSMIND_JWT_SECRET", "my-secret-key")
	os.Setenv("OPSMIND_AI_CONFIDENCE_THRESHOLD", "0.8")
	defer func() {
		os.Unsetenv("OPSMIND_DATABASE_HOST")
		os.Unsetenv("OPSMIND_DATABASE_PASSWORD")
		os.Unsetenv("OPSMIND_ANYTHINGLLM_BASE_URL")
		os.Unsetenv("OPSMIND_JWT_SECRET")
		os.Unsetenv("OPSMIND_AI_CONFIDENCE_THRESHOLD")
	}()

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load() 返回错误: %v", err)
	}

	if cfg.Database.Host != "remote-postgres" {
		t.Errorf("Database.Host = %q, 期望 %q (环境变量覆盖)", cfg.Database.Host, "remote-postgres")
	}
	if cfg.Database.Password != "secret123" {
		t.Errorf("Database.Password = %q, 期望 %q (环境变量覆盖)", cfg.Database.Password, "secret123")
	}
	if cfg.AnythingLLM.BaseURL != "http://custom-llm:3001/api" {
		t.Errorf("AnythingLLM.BaseURL = %q, 期望 %q (环境变量覆盖)", cfg.AnythingLLM.BaseURL, "http://custom-llm:3001/api")
	}
	if cfg.JWT.Secret != "my-secret-key" {
		t.Errorf("JWT.Secret = %q, 期望 %q (环境变量覆盖)", cfg.JWT.Secret, "my-secret-key")
	}
	if cfg.AI.ConfidenceThreshold != 0.8 {
		t.Errorf("AI.ConfidenceThreshold = %f, 期望 0.8 (环境变量覆盖)", cfg.AI.ConfidenceThreshold)
	}
}

// TestLoad_StructFields 验证所有配置结构体字段完整。
func TestLoad_StructFields(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load() 返回错误: %v", err)
	}

	// 验证 AppConfig 包含所有子结构
	if cfg.Server.Port == 0 {
		t.Error("Server 结构体未初始化")
	}
	if cfg.Database.Host == "" {
		t.Error("Database 结构体未初始化")
	}
	if cfg.JWT.AccessExpire == 0 {
		t.Error("JWT 结构体未初始化")
	}
	if cfg.MinIO.Endpoint == "" {
		t.Error("MinIO 结构体未初始化")
	}
	if cfg.AnythingLLM.BaseURL == "" {
		t.Error("AnythingLLM 结构体未初始化")
	}
	if cfg.AI.DefaultTopK == 0 {
		t.Error("AI 结构体未初始化")
	}
}
