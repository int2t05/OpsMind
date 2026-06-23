// Package request 定义认证相关的请求结构体。
//
// 使用 Gin 的 binding 校验器进行参数校验。
package request

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"` // 用户名
	Password string `json:"password" binding:"required"` // 密码
}

// RefreshRequest 刷新令牌请求
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"` // 刷新令牌
}

// LogoutRequest 退出登录请求
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"` // 需失效的刷新令牌
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"` // 旧密码
	NewPassword string `json:"new_password" binding:"required"` // 新密码
}

// ResetPasswordRequest 忘记密码重置请求。
//
// 无需旧密码，仅需用户名和新密码即可重置。
// 适用于登录页"忘记密码"流程。
type ResetPasswordRequest struct {
	Username    string `json:"username" binding:"required"`    // 用户名
	NewPassword string `json:"new_password" binding:"required"` // 新密码
}
