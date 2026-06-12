// Package middleware 实现 Gin HTTP 中间件。
//
// rbac.go 提供 RBAC 权限校验中间件。
// 设计决策：采用 RequirePermission(permissions ...string) 形式而非 RequireRole，
// 原因是权限比角色更稳定（角色名可能变，但 ticket:read 语义不变），
// 且与 PRD 中"菜单/按钮级权限"的需求对齐。
package middleware

import (
	"opsmind/pkg/errcode"
	"opsmind/pkg/response"

	"github.com/gin-gonic/gin"
)

// RequirePermission 返回检查用户是否拥有指定权限的中间件。
//
// 权限判断逻辑：检查 currentUser.Permissions 中是否包含任意一个所需权限。
// 为什么用"任意一个"而非"全部"：大多数场景下接口只需一个权限即可访问，
// 如需同时满足多个权限，可在业务层做更细粒度的校验。
func RequirePermission(permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO(middleware/rbac): permissions 为空时当前逻辑会恒定拒绝。
		// 建议在启动路由注册时检测空权限配置，避免误把接口配置成不可访问。
		// 从 context 获取 currentUser
		val, exists := c.Get("currentUser")
		if !exists {
			response.Error(c, errcode.ErrAuth, "未登录")
			c.Abort()
			return
		}

		currentUser, ok := val.(CurrentUser)
		if !ok {
			response.Error(c, errcode.ErrUnknown, "用户信息类型错误")
			c.Abort()
			return
		}

		// 检查用户是否拥有任意一个所需权限
		if !hasAnyPermission(currentUser.Permissions, permissions) {
			response.Error(c, errcode.ErrForbidden, "无权限执行此操作")
			c.Abort()
			return
		}

		c.Next()
	}
}

// hasAnyPermission 检查用户权限列表中是否包含任意一个所需权限。
func hasAnyPermission(userPerms []string, required []string) bool {
	// TODO(middleware/rbac): 支持通配权限（如 "knowledge:*"）或权限层级。
	// 当前必须逐一列出按钮权限，角色权限增多后维护成本会快速上升。
	permSet := make(map[string]struct{}, len(userPerms))
	for _, p := range userPerms {
		permSet[p] = struct{}{}
	}
	for _, r := range required {
		if _, ok := permSet[r]; ok {
			return true
		}
	}
	return false
}
