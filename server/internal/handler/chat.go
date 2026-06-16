// Package handler 实现 HTTP 请求处理。
//
// chat.go 提供智能问答相关接口（含 SSE 流式输出）。
// Handler 层职责：参数解析、调用 Service、格式化响应。
// LLM 调用与 RAG 编排已下沉至 service.LLMService，Handler 仅做 SSE 事件代理。
package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"opsmind/internal/dto/request"
	"opsmind/internal/service"
	"opsmind/pkg/errcode"
	"opsmind/pkg/response"

	"github.com/gin-gonic/gin"
)

// ChatHandler 智能问答接口。
type ChatHandler struct {
	svc *service.ChatService
}

// NewChatHandler 创建 ChatHandler 实例。
func NewChatHandler(svc *service.ChatService) *ChatHandler {
	return &ChatHandler{svc: svc}
}

// =============================================================================
// 会话 CRUD
// =============================================================================

// CreateChatSession 创建问答会话（仅创建容器，不含 LLM 调用）。
//
// POST /api/v1/portal/chat-sessions
//
// 会话创建后，通过 StreamChatMessage 在会话中发送消息获取 AI 回复。
// 这样设计的原因是：会话生命周期与 AI 调用解耦，前端可灵活控制创建时机。
func (h *ChatHandler) CreateChatSession(c *gin.Context) {
	var req request.CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.ErrParam, "参数校验失败: "+err.Error())
		return
	}

	userID, _ := getCurrentUserID(c)
	session, err := h.svc.CreateSession(req, userID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	response.Success(c, gin.H{
		"session_id": session.ID,
		"kb_id":      session.KBID,
		"question":   session.Question,
		"created_at": session.CreatedAt.Format("2006-01-02 15:04:05"),
	})
}

// ListSessions 查询当前用户的问答会话列表。
//
// GET /api/v1/portal/chat-sessions
func (h *ChatHandler) ListSessions(c *gin.Context) {
	userID, _ := getCurrentUserID(c)
	page, pageSize := parsePagination(c)

	items, total, err := h.svc.ListSessions(userID, page, pageSize)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	response.SuccessWithPage(c, items, total, page, pageSize)
}

// DeleteSession 删除会话及其全部消息。
//
// DELETE /api/v1/portal/chat-sessions/:id
func (h *ChatHandler) DeleteSession(c *gin.Context) {
	userID, _ := getCurrentUserID(c)
	id, ok := parseID(c, "id")
	if !ok {
		return
	}

	if err := h.svc.DeleteSession(id, userID); err != nil {
		handleServiceError(c, err)
		return
	}

	response.Success(c, nil)
}

// SubmitFeedback 提交问答反馈。
//
// POST /api/v1/portal/chat-sessions/:id/feedback
func (h *ChatHandler) SubmitFeedback(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrParam, "无效的会话 ID")
		return
	}

	var body struct {
		Feedback int16 `json:"feedback"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, errcode.ErrParam, "参数校验失败: "+err.Error())
		return
	}

	if body.Feedback < 0 || body.Feedback > 2 {
		response.Error(c, errcode.ErrParam, "反馈值无效，仅允许 0（未评价）、1（已解决）、2（未解决）")
		return
	}

	if err := h.svc.SubmitFeedback(id, body.Feedback); err != nil {
		handleServiceError(c, err)
		return
	}

	response.Success(c, nil)
}

// GetChatDetail 查询问答会话详情。
//
// GET /api/v1/portal/chat-sessions/:id
func (h *ChatHandler) GetChatDetail(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrParam, "无效的会话 ID")
		return
	}

	resp, err := h.svc.GetChatDetail(id)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	response.Success(c, resp)
}

// =============================================================================
// SSE 流式对话
// =============================================================================

// writeSSEEvent 将事件序列化为 JSON 并以 SSE data 帧格式写入。
//
// 为什么不用字符串拼接：json.Marshal 自动处理控制字符转义，
// 消除手动 escapeSSE 的安全隐患。evt 的类型必须有 json 标签与 SSE 格式对齐。
func writeSSEEvent(w gin.ResponseWriter, evt any) error {
	data, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "data: %s\n\n", string(data))
	return err
}

// StreamChatMessage 在已有会话中发送消息并以 SSE 流式返回 AI 答案。
//
// POST /api/v1/portal/chat-sessions/:id/stream
//
// 与 CreateChatSession 配合使用：先创建会话获取 sessionID，
// 再通过此端点发送消息并流式接收 AI 回复。
//
// 流式输出流程：
//  1. 解析 session ID + 请求 → 设置 SSE 响应头
//  2. ChatService.StreamChat 获取事件通道（含 RAG + LLM 流式）
//  3. 逐事件代理到 SSE（step/token/error/done）
//  4. 检测客户端断开，及时终止
func (h *ChatHandler) StreamChatMessage(c *gin.Context) {
	idStr := c.Param("id")
	sessionID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrParam, "无效的会话 ID")
		return
	}

	var req request.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.ErrParam, "参数校验失败: "+err.Error())
		return
	}

	userID, _ := getCurrentUserID(c)
	ctx := c.Request.Context()

	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		response.Error(c, errcode.ErrUnknown, "当前服务器不支持 SSE 流式输出")
		return
	}

	eventCh, err := h.svc.StreamChat(ctx, sessionID, req.Question, userID)
	if err != nil {
		// SSE 头已发送，改用 SSE error 事件返回错误
		writeSSEEvent(c.Writer, service.StreamEvent{Type: "error", Error: err.Error()})
		flusher.Flush()
		return
	}

	rc := http.NewResponseController(c.Writer)
	for evt := range eventCh {
		select {
		case <-ctx.Done():
			return
		default:
		}
		writeSSEEvent(c.Writer, evt)
		flusher.Flush()
		// 每次写入后延长写超时，保证长 SSE 流不被 WriteTimeout 截断
		rc.SetWriteDeadline(time.Now().Add(30 * time.Second))
	}
}
