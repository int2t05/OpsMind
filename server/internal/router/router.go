// Package router 负责注册 Gin 路由。
//
// 路由分为三组：
// - /api/v1/auth — 公开路由（登录、刷新令牌等）
// - /api/v1/portal — 门户端路由（需要 JWT 认证）
// - /api/v1/admin — 后台管理路由（需要 JWT 认证 + RBAC 权限）
//
// MVP 阶段部分路由 Handler 返回 501 Not Implemented，后续任务逐步替换。
package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"opsmind/internal/config"
	"opsmind/internal/handler"
	"opsmind/internal/middleware"
)

// Handlers 聚合所有 Handler 实例，供路由注册使用。
//
// 为什么用结构体而非多参数：Handler 数量随里程碑增加，
// 结构体便于扩展，添加新 Handler 时只需加字段，不影响 Setup 函数签名。
type Handlers struct {
	Auth      *handler.AuthHandler
	User      *handler.UserHandler
	Role      *handler.RoleHandler
	Knowledge *handler.KnowledgeHandler
	Ticket    *handler.TicketHandler
	Chat      *handler.ChatHandler
	Message   *handler.MessageHandler
	Dashboard *handler.DashboardHandler
	Audit     *handler.AuditHandler
	Config    *handler.ConfigHandler
}

// Setup 初始化 Gin 引擎并注册所有路由。
//
// cfg 用于设置 Gin 模式（debug/release）和中间件配置。
// h 包含所有已初始化的 Handler，nil 字段使用占位 Handler。
func Setup(cfg *config.AppConfig, h *Handlers) *gin.Engine {
	// 设置 Gin 模式
	gin.SetMode(cfg.Server.Mode)

	r := gin.New()

	// 注册全局中间件
	r.Use(middleware.RequestID())
	r.Use(middleware.CORS())
	r.Use(middleware.Logger())
	r.Use(gin.Recovery())

	// 健康检查端点（无需认证，供 Docker/K8s 存活探针使用）
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 公开路由组（无需认证）
	public := r.Group("/api/v1/auth")
	registerPublicRoutes(public, h)

	// 门户端路由组（需要 JWT 认证）
	portal := r.Group("/api/v1/portal")
	portal.Use(middleware.JWTAuth(cfg.JWT.Secret))
	registerPortalRoutes(portal, h)

	// 后台管理路由组（需要 JWT 认证 + RBAC 权限）
	admin := r.Group("/api/v1/admin")
	admin.Use(middleware.JWTAuth(cfg.JWT.Secret))
	registerAdminRoutes(admin, h)

	return r
}

// placeholder 返回一个占位 Handler，返回 501 Not Implemented。
// 用于路由骨架注册，后续任务替换为真实 Handler。
func placeholder() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{
			"code":    501,
			"message": "Not Implemented",
			"data":    nil,
		})
	}
}

// registerPublicRoutes 注册公开路由（无需认证）。
func registerPublicRoutes(rg *gin.RouterGroup, h *Handlers) {
	if h != nil && h.Auth != nil {
		rg.POST("/login", h.Auth.Login)
		rg.POST("/refresh", h.Auth.Refresh)
		rg.POST("/change-password", h.Auth.ChangePassword)
		rg.POST("/logout", h.Auth.Logout)
	} else {
		rg.POST("/login", placeholder())
		rg.POST("/refresh", placeholder())
		rg.POST("/change-password", placeholder())
		rg.POST("/logout", placeholder())
	}
}
