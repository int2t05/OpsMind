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
	// TODO(handler/audit): 支持 target_type/target_id/date_range 过滤。
	// 审计排障常按资源维度检索，只有 operator/action 不足以定位具体对象历史。
	var req request.AuditLogListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, errcode.ErrParam, "参数校验失败: "+err.Error())
		return
	}

	page, pageSize := parsePagination(c)

	items, total, err := h.svc.List(req.OperatorID, req.Action, page, pageSize)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	response.SuccessWithPage(c, items, total, page, pageSize)
}
