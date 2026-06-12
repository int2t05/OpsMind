//go:build tools

// Package tools 声明项目核心依赖，确保 go mod tidy 不会移除。
//
// 为什么使用 tools.go 模式而非 cmd/deps.go：
// //go:build tools 构建约束将此文件排除在生产二进制之外，仅用于依赖跟踪。
// 这是 Go 社区约定俗成的模式（见 go.dev/wiki/Modules#how-can-i-track-tool-dependencies）。
package main

import (
	// HTTP 框架 — 处理 REST API 请求
	_ "github.com/gin-gonic/gin"
	// CORS 中间件 — 前后端分离跨域支持
	_ "github.com/gin-contrib/cors"
	// JWT 令牌生成和解析
	_ "github.com/golang-jwt/jwt/v5"
	// MinIO 对象存储客户端
	_ "github.com/minio/minio-go/v7"
	// Viper 多环境配置管理
	_ "github.com/spf13/viper"
	// GORM ORM 框架
	_ "gorm.io/gorm"
	// GORM PostgreSQL 驱动
	_ "gorm.io/datatypes"
	_ "gorm.io/driver/postgres"

	// 内部包
	_ "opsmind/internal/config"
)
