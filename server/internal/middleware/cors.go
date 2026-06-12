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

// CORS 返回 CORS 跨域中间件。
//
// allowOrigins 从配置读取（如 OPSMIND_CORS_ALLOW_ORIGINS），支持环境变量注入。
// 为空时默认使用 localhost:5173（本地开发环境）。
func CORS(allowOrigins []string) gin.HandlerFunc {
	if len(allowOrigins) == 0 {
		allowOrigins = []string{"http://localhost:5173"}
	}

	// TODO(middleware/cors): release 模式下应禁止 "*" 与 localhost 默认值。
	// CORS 属于部署安全配置，建议在 config.Validate 中按环境强校验。
	return cors.New(cors.Config{
		AllowOrigins:     allowOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}
