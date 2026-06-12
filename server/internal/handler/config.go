// Package handler 实现 HTTP 请求处理。
//
// config.go 提供系统配置管理接口。
// 支持获取和更新系统配置项（如 AI 参数、系统行为开关）。
package handler

import (
	"opsmind/internal/service"
	"opsmind/pkg/errcode"
	"opsmind/pkg/response"

	"github.com/gin-gonic/gin"
)

// ConfigHandler 系统配置管理接口。
type ConfigHandler struct {
	svc *service.ConfigService
}

// NewConfigHandler 创建 ConfigHandler 实例。
func NewConfigHandler(svc *service.ConfigService) *ConfigHandler {
	return &ConfigHandler{svc: svc}
}

// Get 获取指定 key 的配置值。
//
// GET /api/v1/admin/configs/:key
func (h *ConfigHandler) Get(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		response.Error(c, errcode.ErrParam, "配置 key 不能为空")
		return
	}

	val, err := h.svc.GetConfig(key)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	response.Success(c, val)
}

// Update 更新或创建系统配置。
//
// PUT /api/v1/admin/configs/:key
func (h *ConfigHandler) Update(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		response.Error(c, errcode.ErrParam, "配置 key 不能为空")
		return
	}

	var body struct {
		Value interface{} `json:"value" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, errcode.ErrParam, "参数校验失败: "+err.Error())
		return
	}

	// 从 JWT context 获取操作人 ID
	updatedBy, _ := getCurrentUserID(c)

	if err := h.svc.UpdateConfig(key, body.Value, updatedBy); err != nil {
		handleServiceError(c, err)
		return
	}

	response.Success(c, nil)
}
