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
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

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
	LLMConfig *handler.LLMConfigHandler
}

// Setup 初始化 Gin 引擎并注册所有路由。
//
// cfg 用于设置 Gin 模式（debug/release）和中间件配置。
// h 包含所有已初始化的 Handler，nil 字段使用占位 Handler。
func Setup(cfg *config.AppConfig, db *gorm.DB, h *Handlers) *gin.Engine {
	// 设置 Gin 模式
	gin.SetMode(cfg.Server.Mode)

	r := gin.New()

	// TODO(router): 增加 /readyz 就绪探针，检查 DB、VectorStore、MinIO、默认 LLM 配置是否可用。
	// /health 只能证明进程存活，不能证明核心依赖可服务。
	// 注册全局中间件
	// Recovery 注册在最外层（第一个）以捕获后续所有中间件的 panic。
	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.CORS(parseCORSOrigins(cfg.CORS.AllowOrigins), cfg.Server.Mode))
	r.Use(middleware.Logger())

	// 健康检查端点（无需认证，供 Docker/K8s 存活探针使用）
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 公开路由组（无需认证）— 仅登录和刷新令牌
	public := r.Group("/api/v1/auth")
	registerPublicRoutes(public, h)

	// JWT 认证路由（需要登录但不需要 RBAC）— 修改密码和登出
	// 使用 /api/v1/auth/me 子路由组，与 docs/API/auth.md 文档一致。
	authRequired := r.Group("/api/v1/auth/me")
	authRequired.Use(middleware.JWTAuth(db, cfg.JWT.Secret))
	registerAuthRequiredRoutes(authRequired, h)

	// 门户端路由组（需要 JWT 认证）
	portal := r.Group("/api/v1/portal")
	portal.Use(middleware.JWTAuth(db, cfg.JWT.Secret))
	registerPortalRoutes(portal, h)

	// 后台管理路由组（需要 JWT 认证 + RBAC 权限）
	admin := r.Group("/api/v1/admin")
	admin.Use(middleware.JWTAuth(db, cfg.JWT.Secret))
	registerAdminRoutes(admin, h)

	return r
}

// registerPublicRoutes 注册公开路由（无需认证）。
func registerPublicRoutes(rg *gin.RouterGroup, h *Handlers) {
	// TODO(router): placeholder 路由适合开发早期，生产环境应在启动时发现 nil Handler 并 fail fast。
	// 否则未装配模块会以运行时 501 暴露给用户。
	if h != nil && h.Auth != nil {
		rg.POST("/login", h.Auth.Login)
		rg.POST("/refresh", h.Auth.Refresh)
	} else {
		rg.POST("/login", placeholder())
		rg.POST("/refresh", placeholder())
	}
}

// registerAuthRequiredRoutes 注册需要 JWT 认证的 auth/me 路由。
//
// 使用 /api/v1/auth/me 子路由组（含 JWTAuth 中间件），
// 与 docs/API/auth.md 文档路径 `/api/v1/auth/me/change-password` 和 `/api/v1/auth/me/logout` 一致。
// ChangePassword handler 需要 JWT 中间件注入的 userID 来识别当前用户。
func registerAuthRequiredRoutes(rg *gin.RouterGroup, h *Handlers) {
	if h != nil && h.Auth != nil {
		rg.POST("/change-password", h.Auth.ChangePassword)
		rg.POST("/logout", h.Auth.Logout)
	} else {
		rg.POST("/change-password", placeholder())
		rg.POST("/logout", placeholder())
	}
}

// parseCORSOrigins 将逗号分隔的字符串解析为 []string。
//
// 配置为空字符串时返回 nil，由 CORS() 中间件使用默认值 localhost:5173。
func parseCORSOrigins(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
