// Package middleware 提供 Gin 中间件。
//
// 本文件实现 JWT 认证中间件，从 Authorization 头提取 Bearer 令牌，
// 解析后将用户信息写入 Gin context，供下游 Handler 使用。
package middleware

import (
	"strings"

	"opsmind/pkg/errcode"
	"opsmind/pkg/jwt"
	"opsmind/pkg/response"

	"github.com/gin-gonic/gin"
)

// CurrentUser JWT 解析后的用户信息，写入 Gin context。
//
// Permissions 由 JWT 中间件根据角色映射填充，RBAC 中间件使用该字段做权限校验。
type CurrentUser struct {
	UserID      int64    `json:"user_id"`
	Username    string   `json:"username"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}

// rolePermissions 定义角色到权限的映射。
//
// 为什么硬编码在中间件中而非数据库：MVP 阶段角色固定（4 个），
// 硬编码避免每次请求查数据库，后续如角色可配置再迁移到数据库查询。
var rolePermissions = map[string][]string{
	"系统管理员":   {"ticket:read", "ticket:write", "ticket:assign", "knowledge:read", "knowledge:write", "knowledge:review", "system:config", "user:manage", "audit:read"},
	"运维人员":    {"ticket:read", "ticket:write", "knowledge:read", "knowledge:write"},
	"知识库管理员": {"knowledge:read", "knowledge:write", "knowledge:review"},
	"报障人":     {},
}

// JWTAuth 返回 JWT 认证中间件。
//
// 为什么返回 gin.HandlerFunc 而非直接使用闭包：
// 函数签名更清晰，调用方通过参数传入 secret，便于测试和配置。
func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			abortWithError(c, errcode.ErrAuth, "缺失 Authorization 头")
			return
		}

		// 提取 Bearer 令牌
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			abortWithError(c, errcode.ErrAuth, "Authorization 格式错误，应为 Bearer <token>")
			return
		}

		claims, err := jwt.ParseToken(parts[1], secret)
		if err != nil {
			abortWithError(c, errcode.ErrAuth, "令牌无效或已过期")
			return
		}

		// 根据角色映射权限
		permissions := resolvePermissions(claims.Roles)

		// 写入 context，供下游 Handler 使用
		c.Set("currentUser", CurrentUser{
			UserID:      claims.UserID,
			Username:    claims.Username,
			Roles:       claims.Roles,
			Permissions: permissions,
		})
		c.Set("userID", claims.UserID)

		c.Next()
	}
}

// resolvePermissions 根据角色列表解析出权限列表。
//
// 使用 Set 去重，原因是同一权限可能被多个角色拥有（如系统管理员和运维人员都有 ticket:read）。
func resolvePermissions(roles []string) []string {
	seen := make(map[string]struct{})
	for _, role := range roles {
		for _, perm := range rolePermissions[role] {
			seen[perm] = struct{}{}
		}
	}
	result := make([]string, 0, len(seen))
	for perm := range seen {
		result = append(result, perm)
	}
	return result
}

// abortWithError 中断请求并返回统一错误响应。
func abortWithError(c *gin.Context, code int, msg string) {
	response.Error(c, code, msg)
	c.Abort()
}
