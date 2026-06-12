// Package handler 实现 HTTP 请求处理。
//
// role.go 提供角色管理相关接口。
// Handler 层职责：参数解析、调用 Service、格式化响应。
package handler

import (
	"strconv"

	"opsmind/internal/dto/request"
	"opsmind/internal/service"
	"opsmind/pkg/errcode"
	"opsmind/pkg/response"

	"github.com/gin-gonic/gin"
)

// RoleHandler 角色管理接口。
type RoleHandler struct {
	svc *service.RoleService
}

// NewRoleHandler 创建 RoleHandler 实例。
func NewRoleHandler(svc *service.RoleService) *RoleHandler {
	return &RoleHandler{svc: svc}
}

// Create 创建角色。
func (h *RoleHandler) Create(c *gin.Context) {
	var req request.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.ErrParam, "参数校验失败: "+err.Error())
		return
	}

	if err := h.svc.Create(req.Name, req.Description, req.Permissions); err != nil {
		handleServiceError(c, err)
		return
	}

	response.Success(c, nil)
}

// GetByID 获取角色详情。
func (h *RoleHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrParam, "无效的角色 ID")
		return
	}

	role, svcErr := h.svc.GetByID(id)
	if svcErr != nil {
		handleServiceError(c, svcErr)
		return
	}

	response.Success(c, role)
}

// List 角色列表（分页）。
func (h *RoleHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	roles, total, err := h.svc.List(page, pageSize)
	if err != nil {
		// TODO: err.Error() 泄露内部错误 — 应使用 handleServiceError(c, err)。
		response.Error(c, errcode.ErrUnknown, err.Error())
		return
	}

	response.SuccessWithPage(c, roles, total, page, pageSize)
}

// Update 更新角色。
func (h *RoleHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrParam, "无效的角色 ID")
		return
	}

	var req request.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.ErrParam, "参数校验失败: "+err.Error())
		return
	}

	if err := h.svc.Update(id, req.Name, req.Description, req.Permissions); err != nil {
		handleServiceError(c, err)
		return
	}

	response.Success(c, nil)
}

// Delete 删除角色。
func (h *RoleHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrParam, "无效的角色 ID")
		return
	}

	if err := h.svc.Delete(id); err != nil {
		handleServiceError(c, err)
		return
	}

	response.Success(c, nil)
}

// =============================================================================
// 菜单管理
// =============================================================================

// ListMenus 获取全部菜单列表。
//
// GET /api/v1/admin/menus
func (h *RoleHandler) ListMenus(c *gin.Context) {
	menus, err := h.svc.ListMenus()
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, menus)
}

// UpdateRoleMenus 更新角色菜单权限绑定。
//
// PUT /api/v1/admin/roles/:id/menus
func (h *RoleHandler) UpdateRoleMenus(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrParam, "无效的角色 ID")
		return
	}

	var body struct {
		MenuIDs []int64 `json:"menu_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, errcode.ErrParam, "参数校验失败: "+err.Error())
		return
	}

	if err := h.svc.UpdateRoleMenus(id, body.MenuIDs); err != nil {
		handleServiceError(c, err)
		return
	}

	response.Success(c, nil)
}
