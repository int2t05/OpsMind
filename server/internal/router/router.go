// Gin 路由注册：全局中间件 + API 路由分组 + 统一 404/405
package router

import (
	"opsmind/server/internal/middleware"
	"opsmind/server/internal/pkg/errors"
	"opsmind/server/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

func Setup() *gin.Engine {
	r := gin.New()

	// 全局中间件
	r.Use(middleware.Recovery())
	r.Use(middleware.RequestID())

	// 统一 404
	r.NoRoute(func(c *gin.Context) {
		response.ErrorRaw(c, 404, errors.CodeNotFound, "接口不存在")
	})

	// 统一 405
	r.NoMethod(func(c *gin.Context) {
		response.ErrorRaw(c, 405, errors.CodeValidationError, "不支持的请求方法")
	})

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "opsmind-server"})
	})

	// API v1
	v1 := r.Group("/api/v1")
	{
		v1.GET("/ping", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "pong"})
		})
	}

	return r
}
