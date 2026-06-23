// Package service 实现角色权限管理业务逻辑。
//
// RoleService 提供角色 CRUD 功能。
// 角色的 Permissions 使用 JSONB 存储权限列表，序列化/反序列化由 GORM datatypes.JSON 自动处理。
package service

import (
	"errors"
	"encoding/json"
	"log/slog"

	"opsmind/internal/model"
	"opsmind/internal/repository"
	"opsmind/pkg/errcode"

	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// RoleService 角色管理服务。
type RoleService struct {
	repo      *repository.RoleRepo
	userRepo  *repository.UserRepo
	db        *gorm.DB
	auditRepo *repository.AuditRepo
}

// NewRoleService 创建 RoleService 实例。
func NewRoleService(repo *repository.RoleRepo, userRepo *repository.UserRepo, db *gorm.DB, auditRepo *repository.AuditRepo) *RoleService {
	return &RoleService{repo: repo, userRepo: userRepo, db: db, auditRepo: auditRepo}
}

// Create 创建角色。
//
// 校验角色名唯一性，重复返回 10005。
func (s *RoleService) Create(name, description string, permissions []string, operatorID int64) error {
	// TODO(service/role): 对 permissions 做白名单校验。
	// 当前任意字符串都能写入角色权限，拼写错误会导致菜单/接口权限悄悄失效。
	// 校验角色名唯一（通过 Repository 层，保证三层架构一致）
	exists, err := s.repo.ExistsByName(name, 0)
	if err != nil {
		return err
	}
	if exists {
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

	if err := s.repo.Create(role); err != nil {
		return err
	}
	s.writeAudit(operatorID, "role:create", role.ID, name)
	return nil
}

// GetByID 根据 ID 获取角色。
func (s *RoleService) GetByID(id int64) (*model.Role, error) {
	role, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
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
func (s *RoleService) Update(id int64, name, description string, permissions []string, operatorID int64) error {
	role, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return AppError{Code: errcode.ErrNotFound, Message: "角色不存在"}
		}
		return err
	}

	// 校验角色名唯一（排除自身，通过 Repository 层）
	exists, err := s.repo.ExistsByName(name, id)
	if err != nil {
		return err
	}
	if exists {
		return AppError{Code: errcode.ErrConflict, Message: "角色名已存在"}
	}

	permsJSON, err := json.Marshal(permissions)
	if err != nil {
		return err
	}

	role.Name = name
	role.Description = description
	role.Permissions = datatypes.JSON(permsJSON)

	if err := s.repo.Update(role); err != nil {
		return err
	}
	s.writeAudit(operatorID, "role:update", id, name)
	return nil
}

// Delete 删除角色。
func (s *RoleService) Delete(id int64, operatorID int64) error {
	// TODO(service/role): 禁止删除系统内置角色或最后一个系统管理员角色。
	// 这些角色是权限体系的根，删除后可能导致系统无法管理。
	_, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return AppError{Code: errcode.ErrNotFound, Message: "角色不存在"}
		}
		return err
	}

	// 检查关联用户：有关联用户则拒绝删除，避免产生孤儿 user_roles 记录。
	count, err := s.userRepo.CountUsersByRole(id)
	if err != nil {
		return err
	}
	if count > 0 {
		return AppError{Code: errcode.ErrConflict, Message: "角色下存在关联用户，无法删除"}
	}

	if err := s.repo.Delete(id); err != nil {
		return err
	}
	s.writeAudit(operatorID, "role:delete", id, "")
	return nil
}

// ListMenus 获取全部菜单列表（树形结构）。
//
// 菜单权限绑定是本模块的核心功能之一，Menu 存储在独立的 menus 表中，
// 但菜单管理归入角色模块，因为菜单是权限的载体。
func (s *RoleService) ListMenus() ([]model.Menu, error) {
	return s.userRepo.ListMenus()
}

// GetRoleMenus 获取指定角色的菜单 ID 列表。
func (s *RoleService) GetRoleMenus(roleID int64) ([]model.Menu, error) {
	// 先确认角色存在
	if _, err := s.repo.GetByID(roleID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, AppError{Code: errcode.ErrNotFound, Message: "角色不存在"}
		}
		return nil, err
	}
	return s.userRepo.GetRoleMenus(roleID)
}

// UpdateRoleMenus 更新角色的菜单权限绑定。
//
// 采用全量替换策略：先清空角色的所有菜单关联，再插入新关联。
// 为什么全量替换而非增量更新：前端提交的是完整菜单 ID 列表，
// 全量替换避免了前端需要追踪增删的复杂性。
func (s *RoleService) UpdateRoleMenus(roleID int64, menuIDs []int64) error {
	// TODO(service/role): 校验 menuIDs 是否全部存在，且按钮权限不能挂到错误父级。
	// 现在直接写关联表，非法 ID 只能依赖数据库约束或静默产生无效授权。
	// 先确认角色存在
	if _, err := s.repo.GetByID(roleID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return AppError{Code: errcode.ErrNotFound, Message: "角色不存在"}
		}
		return err
	}
	return s.userRepo.UpdateRoleMenus(roleID, menuIDs)
}

// writeAudit 写入一条角色审计日志，失败仅 warn 不阻断主流程。
func (s *RoleService) writeAudit(operatorID int64, action string, targetID int64, detail string) {
	if s.auditRepo == nil {
		return
	}
	detailJSON := datatypes.JSON("{}")
	if detail != "" {
		if d, err := json.Marshal(map[string]string{"content": detail}); err == nil {
			detailJSON = datatypes.JSON(d)
		}
	}
	audit := &model.AuditLog{
		OperatorID: operatorID,
		Action:     action,
		TargetType: "role",
		TargetID:   targetID,
		Detail:     detailJSON,
		CreatedAt:  time.Now(),
	}
	if err := s.auditRepo.Create(audit); err != nil {
		slog.Warn("审计日志写入失败", "action", action, "role_id", targetID, "error", err)
	}
}

