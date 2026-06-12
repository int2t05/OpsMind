// Package handler 实现 HTTP 请求处理。
//
// chat.go 提供智能问答相关接口（含 SSE 流式输出）。
// Handler 层职责：参数解析、调用 Service、格式化响应。
// 置信度判断和降级逻辑在 Service 层完成。
//
// 流式输出设计决策：
// 为什么在 Handler 层做 SSE 流式而非 Service 层：
// SSE 是 HTTP 协议层面的传输方式，属于表示层关注点。Service 层返回完整业务结果，
// Handler 层决定以 JSON 还是 SSE 方式交付给客户端，符合单一职责原则。
// 使用 LLMClient.ChatCompletionStream 实现真正的 token 级流式。
package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"opsmind/internal/adapter"
	"opsmind/internal/dto/request"
	"opsmind/internal/service"
	"opsmind/pkg/errcode"
	"opsmind/pkg/response"
	"time"

	"github.com/gin-gonic/gin"
)

// ChatHandler 智能问答接口。
type ChatHandler struct {
	svc       *service.ChatService
	llmClient adapter.LLMClient // 真实 token 级流式（nil 时降级到模拟流式）
}

// NewChatHandler 创建 ChatHandler 实例。
func NewChatHandler(svc *service.ChatService, llmClient adapter.LLMClient) *ChatHandler {
	return &ChatHandler{svc: svc, llmClient: llmClient}
}

// =============================================================================
// 门户端
// =============================================================================

// CreateChatSession 创建问答会话。
//
// POST /api/v1/portal/chat-sessions
func (h *ChatHandler) CreateChatSession(c *gin.Context) {
	var req request.CreateChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.ErrParam, "参数校验失败: "+err.Error())
		return
	}

	userID, _ := getCurrentUserID(c)
	resp, err := h.svc.CreateChatSession(req, userID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	response.Success(c, resp)
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

	// 解析反馈值（int16: 0=未评价, 1=已解决, 2=未解决）
	// TODO: 缺少反馈值范围校验 — 任意 int16 都能通过，应限制为 0/1/2。
	var body struct {
		Feedback int16 `json:"feedback"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, errcode.ErrParam, "参数校验失败: "+err.Error())
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
// SSE 流式输出
// =============================================================================

// StreamChatSession 创建问答会话并以 SSE 流式返回答案。
//
// POST /api/v1/portal/chat-sessions/stream
//
// 流式输出流程：
//  1. 解析请求参数并调用 ChatService.CreateChatSession 获取完整答案
//  2. 设置 SSE 响应头（text/event-stream）
//  3. 以字符块（每次 5 个 rune）流式发送答案文本
//  4. 流式发送完成后，发送 done 事件（含 session_id、sources、confidence 等元数据）
//  5. 发送期间检测客户端断开，及时终止
//
// 为什么在 Handler 层而非 Service 层做流式：
// SSE 是 HTTP 传输层关注点。Service 层返回完整业务结果，
// Handler 层决定以 JSON 还是 SSE 交付，符合单一职责原则。
// 通过 LLMClient.ChatCompletionStream 实现真正的 token 级流式。
func (h *ChatHandler) StreamChatSession(c *gin.Context) {
	var req request.CreateChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.ErrParam, "参数校验失败: "+err.Error())
		return
	}

	userID, _ := getCurrentUserID(c)

	// 调用 Service 层获取完整答案（业务逻辑不变）
	resp, err := h.svc.CreateChatSession(req, userID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	// 设置 SSE 响应头
	// X-Accel-Buffering: no 用于防止 nginx 缓冲 SSE 事件
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	// 检测是否支持 Flusher（所有主流 HTTP 实现都支持）
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		// 不支持的场景降级为普通 JSON 响应
		response.Success(c, resp)
		return
	}

	// LLMClient 可用时使用真正的 token 级流式，不可用时降级到模拟流式
	if h.llmClient != nil {
		h.streamWithLLM(c, flusher, resp.Answer, req)
	} else {
		h.streamSimulated(c, flusher, resp.Answer)
	}

	// 发送完成事件（含完整元数据）
	metadataJSON, merr := json.Marshal(resp)
	if merr != nil {
		fmt.Fprintf(c.Writer, "data: {\"type\":\"done\",\"session_id\":%d}\n\n", resp.SessionID)
	} else {
		fmt.Fprintf(c.Writer, "data: {\"type\":\"done\",\"metadata\":%s}\n\n", string(metadataJSON))
	}
	flusher.Flush()
}

// streamWithLLM 使用 LLMClient.ChatCompletionStream 实现真正的 token 级流式。
func (h *ChatHandler) streamWithLLM(c *gin.Context, flusher http.Flusher, fallbackAnswer string, req request.CreateChatRequest) {
	ctx := c.Request.Context()
	streamReq := adapter.ChatRequest{
		Messages: []adapter.ChatMessage{
			{Role: "user", Content: req.Question},
		},
		MaxTokens:   2048,
		Temperature: 0.3,
	}

	tokenCh, err := h.llmClient.ChatCompletionStream(ctx, streamReq)
	if err != nil {
		h.streamSimulated(c, flusher, fallbackAnswer)
		return
	}

	for chunk := range tokenCh {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if chunk.Error != nil || chunk.Content == "" {
			continue
		}
		fmt.Fprintf(c.Writer, "data: {\"type\":\"token\",\"content\":\"%s\"}\n\n", escapeSSE(chunk.Content))
		flusher.Flush()
		if chunk.FinishReason != "" {
			break
		}
	}
}

// streamSimulated 降级方案：将完整答案按 rune 分块模拟流式输出。
// TODO: 同上 — 使用字符串拼接而非 json.Marshal 构建 SSE 事件，控制字符可能导致 JSON 畸形。
func (h *ChatHandler) streamSimulated(c *gin.Context, flusher http.Flusher, answer string) {
	runes := []rune(answer)
	chunkSize := 5
	for i := 0; i < len(runes); i += chunkSize {
		select {
		case <-c.Request.Context().Done():
			return
		default:
		}
		end := i + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		fmt.Fprintf(c.Writer, "data: {\"type\":\"token\",\"content\":\"%s\"}\n\n", escapeSSE(string(runes[i:end])))
		flusher.Flush()
		time.Sleep(30 * time.Millisecond)
	}
}

// escapeSSE 对 SSE 数据中的特殊字符进行转义。
// TODO: 字符串拼接构建 JSON 不安全 — escapeSSE 不处理 \t、Unicode 控制字符等。
// 应使用 json.Marshal 为每个 token 事件生成 payload，彻底消除手动转义需求。
func escapeSSE(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	return s
}
