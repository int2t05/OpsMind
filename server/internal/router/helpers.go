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
			"message": "Not Implemented",
			"data":    nil,
		})
	}
}

// register 条件注册路由：当 handler 非 nil 时注册真实 handler，否则注册占位。
//
// 为什么集中 nil-guard 逻辑：
// 3 个路由文件中 if h != nil && h.Xxx != nil { ... } else { placeholder() }
// 模式重复 14 次，register 辅助函数压缩为 1 行，让路由结构一目了然。
//
// TODO: 此函数和 registerGroup 目前无任何调用方 — portal.go 和 admin.go 仍使用手写 nil-check。
// 应统一采用此辅助函数（消除 14 处重复），或删除 helpers.go 彻底放弃统一。
func register(rg gin.IRouter, handler interface{}, method, path string, realHandler gin.HandlerFunc) {
	if handler != nil {
		rg.Handle(method, path, realHandler)
	} else {
		rg.Handle(method, path, placeholder())
	}
}

// registerGroup 批量注册同一 handler 下的多条路由。
//
// route 为 {method, path, handlerFunc} 三元组。
// handlerGuard 为 nil-check 的值（如 h.Knowledge）。
func registerGroup(rg gin.IRouter, handlerGuard interface{}, routes []struct {
	Method  string
	Path    string
	Handler gin.HandlerFunc
}) {
	for _, r := range routes {
		register(rg, handlerGuard, r.Method, r.Path, r.Handler)
	}
}
