// Package service 实现认证业务逻辑。
//
// AuthService 处理登录、刷新令牌、修改密码等认证相关操作。
// 依赖 UserRepo 进行用户数据访问，依赖 pkg/jwt 生成令牌。
package service

import (
	"errors"
	"fmt"
	"os"
	"time"

	"opsmind/internal/dto/response"
	"opsmind/internal/model"
	"opsmind/internal/repository"
	"opsmind/pkg/errcode"
	"opsmind/pkg/hash"
	"opsmind/pkg/jwt"

	"gorm.io/gorm"
)

// AppError 是 errcode.AppError 的类型别名，供 service 包内其他文件使用。
type AppError = errcode.AppError

// AuthService 认证业务逻辑
type AuthService struct {
	userRepo *repository.UserRepo
	db       *gorm.DB
}

// NewAuthService 创建 AuthService 实例
func NewAuthService(userRepo *repository.UserRepo, db *gorm.DB) *AuthService {
	return &AuthService{userRepo: userRepo, db: db}
}

// Login 用户登录。
//
// 流程：查用户 → bcrypt 校验 → 检查状态 → 生成令牌 → 组装返回。
// 为什么密码错误和用户不存在返回相同错误码（10003）：
// 避免用户名枚举攻击，不暴露"用户是否存在"信息。
func (s *AuthService) Login(username, password string) (*response.LoginResponse, error) {
	user, err := s.userRepo.GetByUsername(username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, AppError{Code: 10003, Message: "用户名或密码错误"}
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}

	if !hash.CheckPassword(user.PasswordHash, password) {
		return nil, AppError{Code: 10003, Message: "用户名或密码错误"}
	}

	if user.Status == 2 {
		return nil, AppError{Code: 10002, Message: "账号已被冻结"}
	}

	return s.buildLoginResponse(user)
}

// RefreshToken 刷新令牌。
//
// 解析 refresh_token 后重新生成令牌对。
// 为什么不直接生成新 access_token：统一走令牌对刷新，客户端逻辑更简单。
func (s *AuthService) RefreshToken(refreshToken string) (*response.LoginResponse, error) {
	claims, err := jwt.ParseToken(refreshToken, jwtSecret())
	if err != nil {
		return nil, AppError{Code: 10001, Message: "刷新令牌无效或已过期"}
	}

	user, err := s.userRepo.GetByID(claims.UserID)
	if err != nil {
		return nil, AppError{Code: 10001, Message: "用户不存在"}
	}

	if user.Status == 2 {
		return nil, AppError{Code: 10002, Message: "账号已被冻结"}
	}

	return s.buildLoginResponse(user)
}

// ChangePassword 修改密码。
//
// 流程：查用户 → 校验旧密码 → 校验新密码策略 → 更新哈希 → 设置 first_login=false。
// 为什么先校验旧密码再校验新密码策略：旧密码错误是更常见的场景，先返回更有用的错误信息。
func (s *AuthService) ChangePassword(userID int64, oldPwd, newPwd string) error {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return fmt.Errorf("查询用户失败: %w", err)
	}

	if !hash.CheckPassword(user.PasswordHash, oldPwd) {
		return AppError{Code: 10003, Message: "旧密码错误"}
	}

	if err := hash.ValidatePassword(newPwd); err != nil {
		return AppError{Code: 10003, Message: err.Error()}
	}

	newHash, err := hash.HashPassword(newPwd)
	if err != nil {
		return fmt.Errorf("密码哈希失败: %w", err)
	}

	user.PasswordHash = newHash
	user.FirstLogin = false
	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(user); err != nil {
		return fmt.Errorf("更新密码失败: %w", err)
	}

	return nil
}

// buildLoginResponse 根据用户信息组装登录响应。
//
// 查询用户角色、权限、菜单树，组装完整的 LoginResponse。
// 菜单树构建思路：先从全部菜单中分离一级菜单，再递归挂载子菜单。
func (s *AuthService) buildLoginResponse(user *model.User) (*response.LoginResponse, error) {
	// 查询用户角色
	roles, err := s.userRepo.GetUserRoles(user.ID)
	if err != nil {
		return nil, fmt.Errorf("查询用户角色失败: %w", err)
	}

	roleNames := make([]string, 0, len(roles))
	for _, role := range roles {
		roleNames = append(roleNames, role.Name)
	}

	// 查询用户权限
	permissions, err := s.userRepo.GetUserPermissions(user.ID)
	if err != nil {
		return nil, fmt.Errorf("查询用户权限失败: %w", err)
	}

	// 查询用户菜单树
	menuTree, err := s.buildMenuTree(user.ID, roles)
	if err != nil {
		return nil, fmt.Errorf("查询用户菜单失败: %w", err)
	}

	accessToken, err := jwt.GenerateAccessToken(
		user.ID, user.Username, roleNames, permissions,
		jwtSecret(), 2*time.Hour,
	)
	if err != nil {
		return nil, fmt.Errorf("生成 access_token 失败: %w", err)
	}

	refreshToken, err := jwt.GenerateRefreshToken(
		user.ID, user.Username, roleNames, permissions,
		jwtSecret(), 7*24*time.Hour,
	)
	if err != nil {
		return nil, fmt.Errorf("生成 refresh_token 失败: %w", err)
	}

	return &response.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: response.UserInfo{
			ID:         user.ID,
			Username:   user.Username,
			RealName:   user.RealName,
			Phone:      user.Phone,
			Email:      user.Email,
			FirstLogin: user.FirstLogin,
		},
		Roles:       roleNames,
		Permissions: permissions,
		Menus:       menuTree,
	}, nil
}

// buildMenuTree 构建用户的菜单树。
//
// 为什么在 Service 层而非 Repository 层构建树结构：
// 树构建是展示逻辑，属于业务层的职责。Repository 只负责数据查询。
//
// 系统管理员自动获得全部菜单。
func (s *AuthService) buildMenuTree(userID int64, roles []model.Role) ([]response.MenuItem, error) {
	// 判断是否为系统管理员
	isAdmin := false
	for _, role := range roles {
		if role.Name == "系统管理员" {
			isAdmin = true
			break
		}
	}

	var menus []model.Menu
	var err error

	if isAdmin {
		// 系统管理员获取全部菜单
		menus, err = s.userRepo.ListMenus()
	} else {
		// 其他用户：聚合所有角色的菜单（去重）
		menuMap := make(map[int64]model.Menu)
		for _, role := range roles {
			// TODO: 在循环中调用 GetRoleMenus — 用户有 N 个角色则产生 N 次 DB 查询。
			// 应增加 BatchGetRoleMenus(roleIDs) 方法，一次查询获取所有角色的菜单。
			roleMenus, roleErr := s.userRepo.GetRoleMenus(role.ID)
			if roleErr != nil {
				return nil, roleErr
			}
			for _, m := range roleMenus {
				menuMap[m.ID] = m
			}
		}
		for _, m := range menuMap {
			menus = append(menus, m)
		}
	}

	if err != nil {
		return nil, err
	}

	// 构建菜单树
	return buildTree(menus, 0), nil
}

// buildTree 递归构建菜单树。
//
// parentID=0 表示一级菜单，子菜单通过 parentID 关联。
func buildTree(menus []model.Menu, parentID int64) []response.MenuItem {
	var result []response.MenuItem
	for _, m := range menus {
		if m.ParentID == parentID {
			item := response.MenuItem{
				ID:        m.ID,
				Name:      m.Name,
				Path:      m.Path,
				Icon:      m.Icon,
				ParentID:  m.ParentID,
				SortOrder: m.SortOrder,
				Type:      m.Type,
				Children:  buildTree(menus, m.ID),
			}
			result = append(result, item)
		}
	}
	return result
}

// jwtSecret 从环境变量读取 JWT 密钥。
//
// 为什么提供默认值：本地开发环境便利性。
// 生产环境必须通过环境变量 OPSMIND_JWT_SECRET 覆盖。
func jwtSecret() string {
	if s := os.Getenv("OPSMIND_JWT_SECRET"); s != "" {
		return s
	}
	return "opsmind_dev_secret_key_2024"
}
