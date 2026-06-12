// Package handler 实现 HTTP 请求处理。
//
// dashboard.go 提供数据看板相关接口。
// Handler 层职责：参数解析、调用 Service、格式化响应。
// 统计和聚合逻辑在 Service 层完成。
package handler

import (
	"opsmind/internal/dto/request"
	"opsmind/internal/service"
	"opsmind/pkg/errcode"
	"opsmind/pkg/response"

	"github.com/gin-gonic/gin"
)

// DashboardHandler 数据看板接口。
type DashboardHandler struct {
	svc *service.DashboardService
}

// NewDashboardHandler 创建 DashboardHandler 实例。
func NewDashboardHandler(svc *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{svc: svc}
}

// GetStats 获取看板统计数据。
//
// GET /api/v1/admin/dashboard/stats
func (h *DashboardHandler) GetStats(c *gin.Context) {
	resp, err := h.svc.GetStats()
	if err != nil {
		handleServiceError(c, err)
		return
	}

	response.Success(c, resp)
}

// GetTrends 获取趋势数据。
//
// GET /api/v1/admin/dashboard/trends
func (h *DashboardHandler) GetTrends(c *gin.Context) {
	// TODO(handler/dashboard): granularity 参数已在前端/API 类型中出现，但 Service 当前忽略。
	// 应支持 day/week 或从请求 DTO 中移除，避免调用方误以为生效。
	var req request.TrendRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, errcode.ErrParam, "参数校验失败: "+err.Error())
		return
	}

	resp, err := h.svc.GetTrends(req)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	response.Success(c, resp)
}
