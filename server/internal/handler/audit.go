// Package handler 实现 HTTP 请求处理。
//
// audit.go 提供审计日志查询接口。
// 审计日志写入由各 Service 层直接调用 AuditRepo.Create，不经过 Handler。
package handler

import (
	"opsmind/internal/dto/request"
	"opsmind/internal/service"
	"opsmind/pkg/errcode"
	"opsmind/pkg/response"

	"github.com/gin-gonic/gin"
)

// AuditHandler 审计日志查询接口。
type AuditHandler struct {
	svc *service.AuditService
}

// NewAuditHandler 创建 AuditHandler 实例。
func NewAuditHandler(svc *service.AuditService) *AuditHandler {
	return &AuditHandler{svc: svc}
}

// List 查询审计日志列表（分页，支持按操作人和操作类型筛选）。
//
// GET /api/v1/admin/audit-logs?operator_id=1&action=user.create&page=1&page_size=10
func (h *AuditHandler) List(c *gin.Context) {
	var req request.AuditLogListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, errcode.ErrParam, "参数校验失败: "+err.Error())
		return
	}

	page, pageSize := parsePagination(c)

	items, total, err := h.svc.List(req.OperatorID, req.Action, page, pageSize)
	if err != nil {
		// TODO: err.Error() 泄露内部错误 — 应使用 handleServiceError(c, err)。
		response.Error(c, errcode.ErrUnknown, "查询审计日志失败: "+err.Error())
		return
	}

	response.SuccessWithPage(c, items, total, page, pageSize)
}
