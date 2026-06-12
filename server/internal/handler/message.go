// Package handler 实现 HTTP 请求处理。
//
// message.go 提供站内消息相关接口。
// Handler 层职责：参数解析、调用 Service、格式化响应。
package handler

import (
	"strconv"

	"opsmind/internal/service"
	"opsmind/pkg/errcode"
	"opsmind/pkg/response"

	"github.com/gin-gonic/gin"
)

// MessageHandler 站内消息接口。
type MessageHandler struct {
	svc *service.MessageService
}

// NewMessageHandler 创建 MessageHandler 实例。
func NewMessageHandler(svc *service.MessageService) *MessageHandler {
	return &MessageHandler{svc: svc}
}

// =============================================================================
// 门户端
// =============================================================================

// ListMessages 查询当前用户的消息列表。
//
// GET /api/v1/portal/messages
func (h *MessageHandler) ListMessages(c *gin.Context) {
	userID := getCurrentUserID(c)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}

	msgs, total, err := h.svc.ListMessages(userID, page, pageSize)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	response.SuccessWithPage(c, msgs, total, page, pageSize)
}

// MarkAsRead 标记消息为已读。
//
// PUT /api/v1/portal/messages/:id/read
// TODO: 不校验消息归属 — 用户 A 可标记用户 B 的消息已读（水平越权）。
// 应传入 currentUserID 并在 Service/Repo 层校验消息的 user_id 归属。
func (h *MessageHandler) MarkAsRead(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrParam, "无效的消息 ID")
		return
	}

	if err := h.svc.MarkAsRead(id); err != nil {
		handleServiceError(c, err)
		return
	}

	response.Success(c, nil)
}

// CountUnread 查询未读消息数。
//
// GET /api/v1/portal/messages/unread-count
func (h *MessageHandler) CountUnread(c *gin.Context) {
	userID := getCurrentUserID(c)

	count, err := h.svc.CountUnread(userID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	response.Success(c, map[string]int64{"count": count})
}
