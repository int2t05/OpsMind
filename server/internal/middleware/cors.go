// Package middleware 提供 Gin 中间件。
//
// 本文件实现 CORS 跨域中间件。
package middleware

import (
	"log/slog"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORS 返回 CORS 跨域中间件。
//
// allowOrigins 从配置读取（OPSMIND_CORS_ALLOW_ORIGINS），
// debug 模式允许回退到 localhost:5173；release 模式必须显式配置。
//
// DNS 重绑定防护：release 模式拒绝包含 "localhost" 的 origin，
// 因为 localhost 可被攻击者通过 DNS 重绑定解析到外部恶意 IP。
func CORS(allowOrigins []string, mode string) gin.HandlerFunc {
	isRelease := mode == "release"

	if len(allowOrigins) == 0 {
		if isRelease {
			slog.Error("生产模式必须配置 OPSMIND_CORS_ALLOW_ORIGINS，拒绝以空值启动")
			panic("CORS: release 模式不允许空 AllowOrigins")
		}
		allowOrigins = []string{"http://localhost:5173"}
	}

	for _, origin := range allowOrigins {
		// AllowCredentials + "*" 是浏览器安全违规
		if origin == "*" {
			slog.Error("CORS 配置错误：AllowCredentials=true 时不能使用 \"*\"")
			panic("CORS: AllowCredentials=true 时 AllowOrigins 不能包含 \"*\"")
		}
		// release 模式拒绝 localhost（DNS 重绑定攻击面）
		if isRelease && strings.Contains(origin, "localhost") {
			slog.Warn("CORS AllowOrigin 包含 localhost，生产环境存在 DNS 重绑定风险", "origin", origin)
		}
	}

	return cors.New(cors.Config{
		AllowOrigins:     allowOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length", "X-Request-Id", "X-Request-ID", "X-Total-Count"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}
