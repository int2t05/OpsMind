//go:build integration

// Package middleware_test 验证 RBAC 权限中间件。
package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"opsmind/internal/middleware"

	"github.com/gin-gonic/gin"
)

// setupRBACRouter 创建带 RBAC 中间件的测试路由。
func setupRBACRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// 模拟 JWT 中间件注入 currentUser
	r.Use(func(c *gin.Context) {
		userJSON := c.GetHeader("X-Test-User")
		if userJSON != "" {
			var user middleware.CurrentUser
			json.Unmarshal([]byte(userJSON), &user)
			c.Set("currentUser", user)
			c.Set("userID", user.UserID)
		}
		c.Next()
	})

	admin := r.Group("/api/v1/admin")
	admin.Use(middleware.RequirePermission("ticket:read", "ticket:write"))
	admin.GET("/tickets", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
	})

	configGroup := r.Group("/api/v1/admin/config")
	configGroup.Use(middleware.RequirePermission("system:config"))
	configGroup.GET("", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
	})

	return r
}

func TestRBAC_UserWithPermission(t *testing.T) {
	r := setupRBACRouter()

	user := middleware.CurrentUser{
		UserID:      1,
		Username:    "admin",
		Roles:       []string{"系统管理员"},
		Permissions: []string{"ticket:read", "ticket:write", "system:config"},
	}
	userJSON, _ := json.Marshal(user)

	req := httptest.NewRequest("GET", "/api/v1/admin/tickets", nil)
	req.Header.Set("X-Test-User", string(userJSON))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望 200, got %d", w.Code)
	}
}

func TestRBAC_UserWithoutPermission(t *testing.T) {
	r := setupRBACRouter()

	user := middleware.CurrentUser{
		UserID:   2,
		Username: "reporter",
		Roles:    []string{"报障人"},
	}
	userJSON, _ := json.Marshal(user)

	req := httptest.NewRequest("GET", "/api/v1/admin/tickets", nil)
	req.Header.Set("X-Test-User", string(userJSON))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("期望 403, got %d", w.Code)
	}
}

func TestRBAC_NoCurrentUser(t *testing.T) {
	r := setupRBACRouter()

	req := httptest.NewRequest("GET", "/api/v1/admin/tickets", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望 401, got %d", w.Code)
	}
}
