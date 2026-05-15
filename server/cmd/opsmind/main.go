// OpsMind 后端服务入口：加载配置、初始化依赖、启动 HTTP 服务
package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	// 健康检查端点 — 用于验证服务是否可启动
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "opsmind-server",
		})
	})

	// API v1 路由组占位
	v1 := r.Group("/api/v1")
	{
		v1.GET("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "pong"})
		})
	}

	// 默认端口 8080，后续由 Viper 读取 config.yaml 覆盖
	addr := ":8080"
	log.Printf("OpsMind server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server failed to start: %v", err)
	}
}
