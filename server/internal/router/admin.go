// Package router 负责注册 Gin 路由。
//
// 本文件注册后台管理路由，与 TECH.md §5.2 后台管理对齐。
// 所有路由需要 JWT 认证 + RBAC 权限（在 router.go 中统一挂载）。
package router

import (
	"opsmind/internal/middleware"

	"github.com/gin-gonic/gin"
)

// registerAdminRoutes 注册后台管理路由。
//
// 后台管理面向运维人员和管理员，提供申告处理、知识库管理、用户管理等功能。
// 路由列表与 TECH.md §5.2 后台管理对齐。
// 已实现 Handler 的路由绑定真实 Handler，未实现的仍使用占位。
func registerAdminRoutes(rg *gin.RouterGroup, h *Handlers) {
	// 申告管理（T24 — 已实现）
	if h != nil && h.Ticket != nil {
		rg.GET("/tickets", h.Ticket.ListAll)
		rg.GET("/tickets/:id", h.Ticket.GetDetail)
		rg.PATCH("/tickets/:id/status", h.Ticket.UpdateStatus)
		rg.POST("/tickets/:id/records", h.Ticket.AddRecord)
	} else {
		rg.GET("/tickets", placeholder())
		rg.GET("/tickets/:id", placeholder())
		rg.PATCH("/tickets/:id/status", placeholder())
		rg.POST("/tickets/:id/records", placeholder())
	}
	rg.POST("/tickets/:id/knowledge-candidate", placeholder())

	// 知识库管理（T18 — 已实现）
	if h != nil && h.Knowledge != nil {
		rg.GET("/knowledge-bases", h.Knowledge.ListKBs)
		rg.POST("/knowledge-bases", h.Knowledge.CreateKB)
		rg.PUT("/knowledge-bases/:id", h.Knowledge.UpdateKB)
		rg.GET("/knowledge-bases/:kb_id/articles", h.Knowledge.ListArticles)
		rg.POST("/knowledge-bases/:kb_id/articles", h.Knowledge.CreateArticle)
		rg.PUT("/articles/:id", h.Knowledge.UpdateArticle)
		rg.GET("/articles/:id", h.Knowledge.GetArticleDetail)
		rg.POST("/articles/:id/submit-review", h.Knowledge.SubmitReview)
		rg.POST("/articles/:id/review", h.Knowledge.Review)
		rg.POST("/articles/:id/publish", h.Knowledge.Publish)
		rg.POST("/articles/:id/disable", h.Knowledge.Disable)
		rg.POST("/articles/:id/retry-sync", h.Knowledge.RetrySync)
	} else {
		rg.GET("/knowledge-bases", placeholder())
		rg.POST("/knowledge-bases", placeholder())
		rg.PUT("/knowledge-bases/:id", placeholder())
		rg.GET("/knowledge-bases/:kb_id/articles", placeholder())
		rg.POST("/knowledge-bases/:kb_id/articles", placeholder())
		rg.PUT("/articles/:id", placeholder())
		rg.GET("/articles/:id", placeholder())
		rg.POST("/articles/:id/submit-review", placeholder())
		rg.POST("/articles/:id/review", placeholder())
		rg.POST("/articles/:id/publish", placeholder())
		rg.POST("/articles/:id/disable", placeholder())
		rg.POST("/articles/:id/retry-sync", placeholder())
	}

	// 用户管理（T14 — 已实现）
	userRoutes := rg.Group("/users")
	{
		if h != nil && h.User != nil {
			userRoutes.GET("", middleware.RequirePermission("user:manage"), h.User.List)
			userRoutes.POST("", middleware.RequirePermission("user:manage"), h.User.Create)
			userRoutes.GET("/:id", middleware.RequirePermission("user:manage"), h.User.GetByID)
			userRoutes.PUT("/:id", middleware.RequirePermission("user:manage"), h.User.Update)
			userRoutes.PATCH("/:id/freeze", middleware.RequirePermission("user:manage"), h.User.Freeze)
			userRoutes.PATCH("/:id/unfreeze", middleware.RequirePermission("user:manage"), h.User.Restore)
		} else {
			userRoutes.GET("", placeholder())
			userRoutes.POST("", placeholder())
			userRoutes.GET("/:id", placeholder())
			userRoutes.PUT("/:id", placeholder())
			userRoutes.PATCH("/:id/freeze", placeholder())
			userRoutes.PATCH("/:id/unfreeze", placeholder())
		}
	}

	// 角色权限（T15 — 已实现）
	roleRoutes := rg.Group("/roles")
	{
		if h != nil && h.Role != nil {
			roleRoutes.GET("", middleware.RequirePermission("user:manage"), h.Role.List)
			roleRoutes.POST("", middleware.RequirePermission("user:manage"), h.Role.Create)
			roleRoutes.GET("/:id", middleware.RequirePermission("user:manage"), h.Role.GetByID)
			roleRoutes.PUT("/:id", middleware.RequirePermission("user:manage"), h.Role.Update)
			roleRoutes.DELETE("/:id", middleware.RequirePermission("user:manage"), h.Role.Delete)
		} else {
			roleRoutes.GET("", placeholder())
			roleRoutes.POST("", placeholder())
			roleRoutes.GET("/:id", placeholder())
			roleRoutes.PUT("/:id", placeholder())
			roleRoutes.DELETE("/:id", placeholder())
		}
	}

	// 菜单（占位 — T15 菜单权限绑定）
	rg.GET("/menus", placeholder())
	rg.PUT("/roles/:id/menus", placeholder())

	// 数据看板（占位 — T32 实现）
	rg.GET("/dashboard/stats", placeholder())
	rg.GET("/dashboard/trends", placeholder())

	// 操作日志（占位 — T33 实现）
	rg.GET("/audit-logs", placeholder())

	// Embedding 配置（T19 — 已实现）
	if h != nil && h.Knowledge != nil {
		rg.GET("/embedding-configs", h.Knowledge.ListEmbeddingConfigs)
		rg.POST("/embedding-configs", h.Knowledge.CreateEmbeddingConfig)
		rg.PUT("/embedding-configs/:id", h.Knowledge.UpdateEmbeddingConfig)
	} else {
		rg.GET("/embedding-configs", placeholder())
		rg.POST("/embedding-configs", placeholder())
		rg.PUT("/embedding-configs/:id", placeholder())
	}

	// 系统配置（占位 — T34 实现）
	rg.GET("/configs/:key", placeholder())
	rg.PUT("/configs/:key", placeholder())
}
