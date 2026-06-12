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
	svc          *service.ChatService
	llmClient    adapter.LLMClient // 真实 token 级流式（nil 时降级到模拟流式）
	defaultModel string            // streamWithLLM 回退模型
}

// NewChatHandler 创建 ChatHandler 实例。
func NewChatHandler(svc *service.ChatService, llmClient adapter.LLMClient, defaultModel string) *ChatHandler {
	return &ChatHandler{svc: svc, llmClient: llmClient, defaultModel: defaultModel}
}

// =============================================================================
// 门户端
// =============================================================================

// CreateChatSession 创建问答会话。
//
// POST /api/v1/portal/chat-sessions
func (h *ChatHandler) CreateChatSession(c *gin.Context) {
	// TODO(handler/chat): 需要把 c.Request.Context() 传入 Service，避免问答请求断开后仍继续调用 RAG/LLM。
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
	// TODO(handler/chat): 复用 parseID，并在 Service 层校验会话归属当前用户。
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrParam, "无效的会话 ID")
		return
	}

	// 解析反馈值（int16: 0=未评价, 1=已解决, 2=未解决）
	var body struct {
		Feedback int16 `json:"feedback"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, errcode.ErrParam, "参数校验失败: "+err.Error())
		return
	}

	// 校验反馈值范围（仅允许 0/1/2）
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
// SSE 流式输出
// =============================================================================

// sseEvent SSE 事件负载结构，用于 json.Marshal 安全构建 JSON。
type sseEvent struct {
	Type    string `json:"type"`
	Content string `json:"content,omitempty"`
}

// writeSSEEvent 使用 json.Marshal 构建并发送 SSE 事件。
//
// 为什么不用字符串拼接：json.Marshal 自动处理 \t、Unicode 控制字符等转义，
// 彻底消除手动 escapeSSE 的安全隐患。
func writeSSEEvent(w gin.ResponseWriter, evt sseEvent) error {
	data, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "data: %s\n\n", string(data))
	return err
}

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
	// TODO(handler/chat): 当前先调用 CreateChatSession 完整生成答案，再把答案模拟/二次 LLM 流式输出。
	// 这不是真正的端到端 token 流式，也无法实时发送 RAG step 事件；应把 RAG 和 LLM 流式编排下沉到 Service。
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
		// TODO(handler/chat): streamWithLLM 只把用户问题发给 LLM，没有带 RAG 检索上下文。
		// 因此流式 token 可能与 resp.Answer 的 RAG 增强答案完全不一致。
		h.streamWithLLM(c, flusher, resp.Answer, req)
	} else {
		h.streamSimulated(c, flusher, resp.Answer)
	}

	// 发送完成事件（含完整元数据）
	metadataJSON, merr := json.Marshal(resp)
	if merr != nil {
		writeSSEEvent(c.Writer, sseEvent{Type: "done"})
	} else {
		fmt.Fprintf(c.Writer, "data: {\"type\":\"done\",\"metadata\":%s}\n\n", string(metadataJSON))
	}
	flusher.Flush()
}

// streamWithLLM 使用 LLMClient.ChatCompletionStream 实现真正的 token 级流式。
func (h *ChatHandler) streamWithLLM(c *gin.Context, flusher http.Flusher, fallbackAnswer string, req request.CreateChatRequest) {
	ctx := c.Request.Context()
	streamReq := adapter.ChatRequest{
		Model: h.defaultModel,
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

	rc := http.NewResponseController(c.Writer)
	for chunk := range tokenCh {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if chunk.Error != nil || chunk.Content == "" {
			// TODO(handler/chat): 流式 chunk.Error 被静默忽略，前端无法得知 LLM 中途失败。
			// 应发送 type=error 事件并停止流式输出。
			continue
		}
		writeSSEEvent(c.Writer, sseEvent{Type: "token", Content: chunk.Content})
		flusher.Flush()
		// 每次写入后延长写超时，保证长 SSE 流不被 WriteTimeout 截断
		rc.SetWriteDeadline(time.Now().Add(30 * time.Second))
		if chunk.FinishReason != "" {
			break
		}
	}
}

// streamSimulated 降级方案：将完整答案按 rune 分块模拟流式输出。
func (h *ChatHandler) streamSimulated(c *gin.Context, flusher http.Flusher, answer string) {
	// TODO(handler/chat): 模拟流式的 time.Sleep 会占用请求 goroutine。
	// 如果只剩完整答案，建议直接返回 done 或让前端自行做打字机效果。
	runes := []rune(answer)
	chunkSize := 5
	rc := http.NewResponseController(c.Writer)
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
		writeSSEEvent(c.Writer, sseEvent{Type: "token", Content: string(runes[i:end])})
		flusher.Flush()
		// 每次写入后延长写超时，保证长 SSE 流不被 WriteTimeout 截断
		rc.SetWriteDeadline(time.Now().Add(30 * time.Second))
		time.Sleep(30 * time.Millisecond)
	}
}
