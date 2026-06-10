// Package service 实现角色权限管理业务逻辑。
//
// RoleService 提供角色 CRUD 功能。
// 角色的 Permissions 使用 JSONB 存储权限列表，序列化/反序列化由 GORM datatypes.JSON 自动处理。
package service

import (
	"encoding/json"

	"opsmind/internal/model"
	"opsmind/internal/repository"
	"opsmind/pkg/errcode"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// RoleService 角色管理服务。
type RoleService struct {
	repo *repository.RoleRepo
	db   *gorm.DB
}

// NewRoleService 创建 RoleService 实例。
func NewRoleService(repo *repository.RoleRepo, db *gorm.DB) *RoleService {
	return &RoleService{repo: repo, db: db}
}

// Create 创建角色。
//
// 校验角色名唯一性，重复返回 10005。
func (s *RoleService) Create(name, description string, permissions []string) error {
	// 校验角色名唯一
	var count int64
	s.db.Model(&model.Role{}).Where("name = ?", name).Count(&count)
	if count > 0 {
		return AppError{Code: errcode.ErrConflict, Message: "角色名已存在"}
	}

	permsJSON, err := json.Marshal(permissions)
	if err != nil {
		return err
	}

	role := &model.Role{
		Name:        name,
		Description: description,
		Permissions: datatypes.JSON(permsJSON),
	}

	return s.repo.Create(role)
}

// GetByID 根据 ID 获取角色。
func (s *RoleService) GetByID(id int64) (*model.Role, error) {
	role, err := s.repo.GetByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, AppError{Code: errcode.ErrNotFound, Message: "角色不存在"}
		}
		return nil, err
	}
	return role, nil
}

// List 查询角色列表（分页）。
func (s *RoleService) List(page, pageSize int) ([]model.Role, int64, error) {
	return s.repo.List(page, pageSize)
}

// Update 更新角色。
//
// 校验新名称是否与其他角色冲突（排除自身），
// 与 Create 保持一致的唯一性约束。
func (s *RoleService) Update(id int64, name, description string, permissions []string) error {
	role, err := s.repo.GetByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return AppError{Code: errcode.ErrNotFound, Message: "角色不存在"}
		}
		return err
	}

	// 校验角色名唯一（排除自身）
	var count int64
	s.db.Model(&model.Role{}).Where("name = ? AND id != ?", name, id).Count(&count)
	if count > 0 {
		return AppError{Code: errcode.ErrConflict, Message: "角色名已存在"}
	}

	permsJSON, err := json.Marshal(permissions)
	if err != nil {
		return err
	}

	role.Name = name
	role.Description = description
	role.Permissions = datatypes.JSON(permsJSON)

	return s.repo.Update(role)
}

// Delete 删除角色。
func (s *RoleService) Delete(id int64) error {
	_, err := s.repo.GetByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return AppError{Code: errcode.ErrNotFound, Message: "角色不存在"}
		}
		return err
	}

	return s.repo.Delete(id)
}
