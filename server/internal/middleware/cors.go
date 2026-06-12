// Package middleware 提供 Gin 中间件。
//
// 本文件实现 CORS 跨域中间件，配置遵循 TECH.md 规范：
// - 允许来源：http://localhost:5173（开发环境）
// - 允许方法：GET/POST/PUT/PATCH/DELETE/OPTIONS
// - 允许 Header：Authorization/Content-Type
// - 暴露 Header：Content-Length
// - MaxAge：12 小时
package middleware

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORS 返回 CORS 跨域中间件
// TODO: AllowOrigins 硬编码为 localhost:5173 — 非本地环境需手动修改代码。
// 应从配置读取（如 config.AppConfig.CORS.AllowOrigins），支持环境变量注入。
func CORS() gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}
