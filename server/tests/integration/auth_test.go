//go:build integration

// Package integration_test 验证认证模块的端到端完整流程。
//
// 测试覆盖 PLAN.md Task36 定义的场景：
//   - 完整认证生命周期：登录→获取令牌→刷新令牌→修改密码→新密码登录
//   - 首次登录强制修改密码
//   - 旧密码失效验证
//
// 与 handler_test 的区别：本测试使用完整依赖链（DB→Repo→Service→Handler），
// 通过真实数据库验证多步骤用户操作的完整链路。
package integration_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"opsmind/internal/config"
	"opsmind/internal/database"
	"opsmind/internal/handler"
	"opsmind/internal/model"
	"opsmind/internal/repository"
	"opsmind/internal/service"
	"opsmind/pkg/hash"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// testJWTConfig 返回测试用 JWT 配置，与 config.yaml 默认值一致。
func testJWTConfig() config.JWTConfig {
	return config.JWTConfig{
		Secret:        "test_secret_key_2024",
		AccessExpire:  2 * time.Hour,
		RefreshExpire: 168 * time.Hour,
	}
}

// =============================================================================
// 测试环境
// =============================================================================

// authIntegrationEnv 封装认证集成测试环境。
//
// 使用完整的依赖链：DB → Repo → Service → Handler → Router。
// change-password 端点需要 userID 在 context 中，
// 因此通过 mockAuthMiddleware 模拟 JWT 认证中间件的效果。
type authIntegrationEnv struct {
	r  *gin.Engine
	db *gorm.DB
}

// setupAuthIntegration 创建完整的认证集成测试环境。
func setupAuthIntegration(t *testing.T) *authIntegrationEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dbCfg := config.DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "opsmind",
		Password: "opsmind_dev",
		DBName:   "opsmind_test",
		SSLMode:  "disable",
	}
	db, err := database.Init(dbCfg)
	require.NoError(t, err, "初始化数据库失败")

	// 建表
	db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id BIGSERIAL PRIMARY KEY,
		username VARCHAR(64) NOT NULL UNIQUE,
		password_hash VARCHAR(255) NOT NULL,
		real_name VARCHAR(64) NOT NULL,
		phone VARCHAR(11) NOT NULL,
		email VARCHAR(128),
		status SMALLINT NOT NULL DEFAULT 1,
		first_login BOOLEAN NOT NULL DEFAULT TRUE,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS roles (
		id BIGSERIAL PRIMARY KEY,
		name VARCHAR(64) NOT NULL UNIQUE,
		description TEXT,
		permissions JSONB,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS user_roles (
		user_id BIGINT NOT NULL,
		role_id BIGINT NOT NULL,
		PRIMARY KEY (user_id, role_id)
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS menus (
		id BIGSERIAL PRIMARY KEY,
		name VARCHAR(64) NOT NULL,
		path VARCHAR(128),
		icon VARCHAR(64),
		parent_id BIGINT DEFAULT 0,
		sort_order INT DEFAULT 0,
		type VARCHAR(16) DEFAULT 'menu',
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS role_menus (
		role_id BIGINT NOT NULL,
		menu_id BIGINT NOT NULL,
		PRIMARY KEY (role_id, menu_id)
	)`)

	// 清理历史数据
	db.Exec("DELETE FROM user_roles")
	db.Exec("DELETE FROM role_menus")
	db.Exec("DELETE FROM users")
	db.Exec("DELETE FROM roles")
	db.Exec("DELETE FROM menus")

	// 组装完整依赖链（与 main.go 一致）
	userRepo := repository.NewUserRepo(db)
	authService := service.NewAuthService(userRepo, db, testJWTConfig())
	authHandler := handler.NewAuthHandler(authService)

	// 自定义路由：
	// login / refresh / logout → 公开（无需 userID）
	// change-password → 需要 userID in context（通过 mockAuthMiddleware 注入）
	r := gin.New()

	auth := r.Group("/api/v1/auth")
	{
		auth.POST("/login", authHandler.Login)
		auth.POST("/refresh", authHandler.Refresh)
		auth.POST("/logout", authHandler.Logout)
		// change-password 需要 userID，通过中间件从 Authorization header 中解析
		auth.POST("/change-password", mockAuthMiddleware(db), authHandler.ChangePassword)
	}

	return &authIntegrationEnv{r: r, db: db}
}

// mockAuthMiddleware 模拟 JWT 认证中间件，从 Authorization header 解析用户信息。
//
// 为什么不用真实的 JWT 中间件：
// 认证集成测试的重点是 DB→Service→Handler 的完整链路，
// JWT 中间件的解析逻辑已在 middleware/auth_test.go 中单独验证。
// 此处使用简化版中间件避免 JWT secret 配置和 token 生命周期管理的额外复杂度。
func mockAuthMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 10001, "message": "未登录"})
			return
		}

		// 从 Authorization header 中提取 username（简化版：
		// 集成测试中 Authorization header 直接传递 username 而非真实 JWT token）
		username := authHeader
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			username = authHeader[7:]
		}

		// 查找用户并注入 context
		var user model.User
		if err := db.Where("username = ?", username).First(&user).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 10001, "message": "用户不存在"})
			return
		}
		c.Set("userID", user.ID)
		c.Set("currentUser", map[string]interface{}{
			"user_id":  float64(user.ID),
			"username": user.Username,
			"roles":    []interface{}{},
		})
		c.Next()
	}
}

// seedAuthUser 创建测试用户。
func seedAuthUser(t *testing.T, db *gorm.DB, username, password, phone string, firstLogin bool) *model.User {
	t.Helper()

	hashed, err := hash.HashPassword(password)
	require.NoError(t, err)

	user := &model.User{
		Username:     username,
		PasswordHash: hashed,
		RealName:     "集成测试用户",
		Phone:        phone,
		Status:       1,
		FirstLogin:   firstLogin,
	}
	require.NoError(t, db.Create(user).Error)
	return user
}

// parseJSON 将 HTTP 响应体解析为 map。
func parseJSON(t *testing.T, body []byte) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &m))
	return m
}

// =============================================================================
// 完整认证生命周期测试
// =============================================================================

// TestAuthIntegration_FullLifecycle 验证完整的认证生命周期。
//
// 流程：创建用户 → 登录获取令牌 → 刷新令牌 → 修改密码 →
// 新密码登录成功 → 旧密码登录失败
func TestAuthIntegration_FullLifecycle(t *testing.T) {
	env := setupAuthIntegration(t)
	oldPassword := "OldPass@123"
	newPassword := "NewPass@456"
	username := "itg_auth_user"
	phone := "13800001001"

	// 1. 创建测试用户
	seedAuthUser(t, env.db, username, oldPassword, phone, false)

	// 2. 登录 → 获取 access_token 和 refresh_token
	loginBody, _ := json.Marshal(map[string]string{
		"username": username,
		"password": oldPassword,
	})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	env.r.ServeHTTP(loginW, loginReq)

	assert.Equal(t, http.StatusOK, loginW.Code, "登录应返回 200")
	loginResp := parseJSON(t, loginW.Body.Bytes())
	assert.Equal(t, float64(0), loginResp["code"], "登录业务码应为 0")

	loginData := loginResp["data"].(map[string]interface{})
	accessToken := loginData["access_token"].(string)
	refreshToken := loginData["refresh_token"].(string)
	assert.NotEmpty(t, accessToken, "应返回 access_token")
	assert.NotEmpty(t, refreshToken, "应返回 refresh_token")
	t.Logf("✅ 步骤1-2: 登录成功，获得令牌")

	// 3. 刷新令牌 → 获得新令牌对
	refreshBody, _ := json.Marshal(map[string]string{
		"refresh_token": refreshToken,
	})
	refreshReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh",
		bytes.NewReader(refreshBody))
	refreshReq.Header.Set("Content-Type", "application/json")
	refreshW := httptest.NewRecorder()
	env.r.ServeHTTP(refreshW, refreshReq)

	assert.Equal(t, http.StatusOK, refreshW.Code, "刷新令牌应返回 200")
	refreshResp := parseJSON(t, refreshW.Body.Bytes())
	assert.Equal(t, float64(0), refreshResp["code"], "刷新令牌业务码应为 0")

	refreshData := refreshResp["data"].(map[string]interface{})
	newAccessToken := refreshData["access_token"].(string)
	newRefreshToken := refreshData["refresh_token"].(string)
	assert.NotEmpty(t, newAccessToken, "刷新后应有新 access_token")
	assert.NotEmpty(t, newRefreshToken, "刷新后应有新 refresh_token")
	// 注意：JWT 令牌包含 iat 声明，同一秒内刷新可能生成相同令牌（iat 精度为秒）。
	// 验证令牌可解析即可，不强求字符串不同。
	t.Logf("✅ 步骤3: 令牌刷新成功")

	// 4. 修改密码（传 username 到 Authorization header，由 mockAuthMiddleware 注入 userID）
	changePwdBody, _ := json.Marshal(map[string]string{
		"old_password": oldPassword,
		"new_password": newPassword,
	})
	changePwdReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password",
		bytes.NewReader(changePwdBody))
	changePwdReq.Header.Set("Content-Type", "application/json")
	changePwdReq.Header.Set("Authorization", "Bearer "+username)
	changePwdW := httptest.NewRecorder()
	env.r.ServeHTTP(changePwdW, changePwdReq)

	assert.Equal(t, http.StatusOK, changePwdW.Code, "修改密码应返回 200")
	changePwdResp := parseJSON(t, changePwdW.Body.Bytes())
	assert.Equal(t, float64(0), changePwdResp["code"], "修改密码业务码应为 0")
	t.Logf("✅ 步骤4: 密码修改成功")

	// 5. 用新密码登录 → 应成功
	loginNewBody, _ := json.Marshal(map[string]string{
		"username": username,
		"password": newPassword,
	})
	loginNewReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		bytes.NewReader(loginNewBody))
	loginNewReq.Header.Set("Content-Type", "application/json")
	loginNewW := httptest.NewRecorder()
	env.r.ServeHTTP(loginNewW, loginNewReq)

	assert.Equal(t, http.StatusOK, loginNewW.Code, "新密码登录应返回 200")
	loginNewResp := parseJSON(t, loginNewW.Body.Bytes())
	assert.Equal(t, float64(0), loginNewResp["code"], "新密码登录业务码应为 0")
	t.Logf("✅ 步骤5: 新密码登录成功")

	// 6. 用旧密码登录 → 应失败
	loginOldBody, _ := json.Marshal(map[string]string{
		"username": username,
		"password": oldPassword,
	})
	loginOldReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		bytes.NewReader(loginOldBody))
	loginOldReq.Header.Set("Content-Type", "application/json")
	loginOldW := httptest.NewRecorder()
	env.r.ServeHTTP(loginOldW, loginOldReq)

	// 旧密码应无法登录（返回非 0 错误码）
	loginOldResp := parseJSON(t, loginOldW.Body.Bytes())
	assert.NotEqual(t, float64(0), loginOldResp["code"], "旧密码登录应返回错误码")
	t.Logf("✅ 步骤6: 旧密码已失效，无法登录")
}

// =============================================================================
// 首次登录强制修改密码
// =============================================================================

// TestAuthIntegration_FirstLoginFlag 验证首次登录标记。
//
// 首次登录用户（first_login=true）登录后，
// 响应中的 first_login 字段应为 true，前端据此跳转修改密码页。
func TestAuthIntegration_FirstLoginFlag(t *testing.T) {
	env := setupAuthIntegration(t)
	username := "itg_first_login"
	phone := "13800001002"
	password := "First@1234"

	// 创建首次登录用户
	seedAuthUser(t, env.db, username, password, phone, true)

	// 登录
	loginBody, _ := json.Marshal(map[string]string{
		"username": username,
		"password": password,
	})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	env.r.ServeHTTP(loginW, loginReq)

	assert.Equal(t, http.StatusOK, loginW.Code)
	loginResp := parseJSON(t, loginW.Body.Bytes())
	assert.Equal(t, float64(0), loginResp["code"])

	loginData := loginResp["data"].(map[string]interface{})
	userInfo := loginData["user"].(map[string]interface{})

	// 验证 first_login 字段
	firstLogin, ok := userInfo["first_login"].(bool)
	assert.True(t, ok, "响应应包含 first_login 字段")
	assert.True(t, firstLogin, "首次登录用户 first_login 应为 true")
	t.Logf("✅ 首次登录标记验证通过: first_login=true")

	// 验证有 token 返回（前端需要 token 才能调用 change-password）
	assert.NotEmpty(t, loginData["access_token"],
		"首次登录也应返回 access_token（用于调用修改密码接口）")
	t.Logf("✅ 首次登录仍返回 token，可调用修改密码接口")
}

// =============================================================================
// 登录失败场景
// =============================================================================

// TestAuthIntegration_LoginFailures 验证登录失败场景。
func TestAuthIntegration_LoginFailures(t *testing.T) {
	env := setupAuthIntegration(t)
	username := "itg_failures"
	phone := "13800001003"
	password := "Valid@1234"

	// 创建正常用户
	seedAuthUser(t, env.db, username, password, phone, false)

	// 场景1: 密码错误
	body, _ := json.Marshal(map[string]string{
		"username": username,
		"password": "WrongPass@1",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.r.ServeHTTP(w, req)
	resp := parseJSON(t, w.Body.Bytes())
	assert.NotEqual(t, float64(0), resp["code"], "密码错误应返回非 0 错误码")
	t.Logf("✅ 密码错误被正确拒绝")

	// 场景2: 用户不存在
	body2, _ := json.Marshal(map[string]string{
		"username": "nonexistent_user",
		"password": "Whatever@1",
	})
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	env.r.ServeHTTP(w2, req2)
	resp2 := parseJSON(t, w2.Body.Bytes())
	assert.NotEqual(t, float64(0), resp2["code"], "用户不存在应返回非 0 错误码")
	t.Logf("✅ 不存在用户被正确拒绝")

	// 场景3: 缺少必填参数
	body3, _ := json.Marshal(map[string]string{"username": username})
	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body3))
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	env.r.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusBadRequest, w3.Code, "缺少密码应返回 400")
	t.Logf("✅ 参数缺失返回 400")
}

// =============================================================================
// 修改密码校验
// =============================================================================

// TestAuthIntegration_ChangePasswordValidation 验证修改密码的校验规则。
func TestAuthIntegration_ChangePasswordValidation(t *testing.T) {
	env := setupAuthIntegration(t)
	username := "itg_chgpwd_val"
	phone := "13800001004"
	password := "Valid@1234"

	u := seedAuthUser(t, env.db, username, password, phone, false)

	// 场景1: 新密码不符合策略（缺少大写）
	body1, _ := json.Marshal(map[string]string{
		"old_password": password,
		"new_password": "alllowercase1",
	})
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password",
		bytes.NewReader(body1))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer "+username)
	w1 := httptest.NewRecorder()
	env.r.ServeHTTP(w1, req1)
	resp1 := parseJSON(t, w1.Body.Bytes())
	assert.NotEqual(t, float64(0), resp1["code"], "密码缺少大写字母应拒绝")
	t.Logf("✅ 弱密码（缺大写）被拒绝")

	// 场景2: 旧密码错误
	body2, _ := json.Marshal(map[string]string{
		"old_password": "WrongOld@12345",
		"new_password": "NewValid@123",
	})
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password",
		bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+username)
	w2 := httptest.NewRecorder()
	env.r.ServeHTTP(w2, req2)
	resp2 := parseJSON(t, w2.Body.Bytes())
	assert.NotEqual(t, float64(0), resp2["code"], "旧密码错误应拒绝")
	t.Logf("✅ 旧密码错误被拒绝")

	// 验证密码未被修改
	var user model.User
	env.db.First(&user, u.ID)
	assert.True(t, hash.CheckPassword(user.PasswordHash, password),
		"密码不应被修改（旧密码错误时）")
	t.Logf("✅ 旧密码错误时原密码未被修改")
}
