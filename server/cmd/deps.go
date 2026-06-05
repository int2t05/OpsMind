// Package main 的依赖声明文件。
//
// 该文件仅用于声明项目核心依赖，确保 go mod tidy 不会移除它们。
// 实际使用这些依赖的代码在 internal/ 包中，此处通过匿名导入引入。
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
	// pgvector 向量类型支持
	_ "github.com/pgvector/pgvector-go"
	// Viper 多环境配置管理
	_ "github.com/spf13/viper"
	// GORM ORM 框架
	_ "gorm.io/gorm"
	// GORM PostgreSQL 驱动
	_ "gorm.io/driver/postgres"

	// 内部包 — 确保编译通过
	_ "opsmind/internal/config"
)
