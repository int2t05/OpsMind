// Package router 负责注册 Gin 路由。
//
// 本文件注册门户端路由，与 TECH.md §5.2 门户端对齐。
// 所有路由需要 JWT 认证。
package router

import "github.com/gin-gonic/gin"

// registerPortalRoutes 注册门户端路由。
//
// 门户端面向报障人用户，提供智能问答、申告提交、进度查询等功能。
// 路由列表与 TECH.md §5.2 门户端对齐。
func registerPortalRoutes(rg *gin.RouterGroup, h *Handlers) {
	// 智能问答（占位 — T26 实现）
	rg.POST("/chat-sessions", placeholder())
	rg.GET("/chat-sessions/:id", placeholder())
	rg.POST("/chat-sessions/:id/feedback", placeholder())

	// 申告管理（T24 — 已实现）
	if h != nil && h.Ticket != nil {
		rg.POST("/tickets", h.Ticket.CreateTicket)
		rg.GET("/tickets", h.Ticket.ListByUser)
		rg.GET("/tickets/:id", h.Ticket.GetDetail)
		rg.PATCH("/tickets/:id/supplement", h.Ticket.SupplementTicket)
	} else {
		rg.POST("/tickets", placeholder())
		rg.GET("/tickets", placeholder())
		rg.GET("/tickets/:id", placeholder())
		rg.PATCH("/tickets/:id/supplement", placeholder())
	}

	// 站内消息（占位 — T29 实现）
	rg.GET("/messages", placeholder())
	rg.PATCH("/messages/:id/read", placeholder())
	rg.GET("/messages/unread-count", placeholder())
}
