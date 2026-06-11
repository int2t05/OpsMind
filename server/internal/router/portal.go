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
	// 知识库列表（门户端 Chat 需要选择知识库，无需 admin 权限）
	if h != nil && h.Knowledge != nil {
		rg.GET("/knowledge-bases", h.Knowledge.ListKBsForPortal)
	} else {
		rg.GET("/knowledge-bases", placeholder())
	}

	// 智能问答（T26 — 已实现）
	// chat-sessions/stream 必须在 :id 路由之前注册，避免 "stream" 被当作 :id 参数捕获
	if h != nil && h.Chat != nil {
		rg.POST("/chat-sessions/stream", h.Chat.StreamChatSession)
		rg.POST("/chat-sessions", h.Chat.CreateChatSession)
		rg.GET("/chat-sessions/:id", h.Chat.GetChatDetail)
		rg.POST("/chat-sessions/:id/feedback", h.Chat.SubmitFeedback)
	} else {
		rg.POST("/chat-sessions/stream", placeholder())
		rg.POST("/chat-sessions", placeholder())
		rg.GET("/chat-sessions/:id", placeholder())
		rg.POST("/chat-sessions/:id/feedback", placeholder())
	}

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

	// 站内消息（T29 — 已实现）
	if h != nil && h.Message != nil {
		rg.GET("/messages", h.Message.ListMessages)
		rg.PUT("/messages/:id/read", h.Message.MarkAsRead)
		rg.GET("/messages/unread-count", h.Message.CountUnread)
	} else {
		rg.GET("/messages", placeholder())
		rg.PUT("/messages/:id/read", placeholder())
		rg.GET("/messages/unread-count", placeholder())
	}
}
