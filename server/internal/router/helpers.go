// Package router 负责注册 Gin 路由。
//
// helpers.go 提供路由注册辅助函数，消除路由文件中的 nil-check 样板代码。
package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// placeholder 返回一个占位 Handler，返回 501 Not Implemented。
func placeholder() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{
			"code":    501,
			"message": "功能未实现",
			"data":    nil,
		})
	}
}

// portal.go 和 admin.go 使用手写 nil-check 注册路由。
// 保留 helpers.go 仅提供 placeholder() 辅助函数。
