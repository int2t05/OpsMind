// Package service 实现智能问答业务逻辑。
//
// ChatService 使用自建 RAG Pipeline（查询改写→多路检索→混合检索→重排序）
// 和 LLMService 进行知识增强问答生成，支持 SSE 流式输出。
//
// 会话与对话分离设计：
// CreateSession 仅创建会话容器，不触发 LLM。StreamChat 在已有会话中
// 发送消息并流式返回 AI 答案。这样的好处是职责单一、前端可灵活控制
// 会话生命周期（如先创建会话占位，再异步发送消息）。
package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"opsmind/internal/adapter"
	"opsmind/internal/dto/request"
	"opsmind/internal/dto/response"
	"opsmind/internal/model"
	"opsmind/internal/rag"
	"opsmind/pkg/errcode"
)

const (
	defaultConfidenceThreshold = 0.6
	fallbackLowConfidence      = "暂未找到足够匹配的知识，建议提交申告由运维人员人工处理"
	fallbackAIUnavailable      = "当前 AI 服务暂不可用，请提交申告由人工处理"
)

// 消费者接口——ChatService 仅暴露它实际使用的依赖方法，
// 遵循 Go "accept interfaces, return structs" 惯例，便于测试 mock。
type chatKnowledgeRepo interface {
	FindKBByID(id int64) (*model.KnowledgeBase, error)
}

type chatSessionRepo interface {
	Create(session *model.ChatSession) error
	CreateBatch(messages []model.ChatMessage) error
	FindByID(id int64) (*model.ChatSession, error)
	FindMessagesBySession(sessionID int64) ([]model.ChatMessage, error)
	UpdateFeedback(id int64, feedback int16) error
	UpdateSession(session *model.ChatSession) error
	ListByUser(userID int64, page, pageSize int) ([]model.ChatSession, int64, error)
	DeleteSession(id, userID int64) error
	CountMessagesBySession(sessionID int64) (int64, error)
}

type chatPipeline interface {
	Execute(ctx context.Context, query string, kbID int64, opts rag.RAGOptions, onStep rag.StepCallback) (*rag.RAGResult, error)
}

// ChatService 智能问答服务。
//
// knowledgeRepo/chatRepo/pipeline 使用接口类型，便于测试 mock。
// llmService 统一管理 RAG+LLM 调用编排（流式）。
type ChatService struct {
	defaultTopK   int
	knowledgeRepo chatKnowledgeRepo
	chatRepo      chatSessionRepo
	pipeline      chatPipeline
	llmService    *LLMService
}

// NewChatService 创建 ChatService 实例。
//
// llmService 可以为 nil（测试或降级场景）。
func NewChatService(knowledgeRepo chatKnowledgeRepo, chatRepo chatSessionRepo, pipeline chatPipeline, llmService *LLMService, defaultTopK int) *ChatService {
	if defaultTopK <= 0 {
		defaultTopK = 5
	}
	return &ChatService{
		knowledgeRepo: knowledgeRepo,
		chatRepo:      chatRepo,
		pipeline:      pipeline,
		llmService:    llmService,
		defaultTopK:   defaultTopK,
	}
}

// =============================================================================
// CreateSession — 创建会话容器
// =============================================================================

// CreateSession 创建问答会话（仅创建容器，不含 LLM 调用）。
//
// 为什么创建和对话分开：前端先创建会话获得 sessionID，
// 再通过 StreamChat 发送消息。职责分离使会话生命周期可控，
// 也避免了非流式端点中 LLM 调用超时阻塞 HTTP 请求的问题。
func (s *ChatService) CreateSession(req request.CreateSessionRequest, userID int64) (*model.ChatSession, error) {
	if s.knowledgeRepo != nil {
		if _, err := s.knowledgeRepo.FindKBByID(req.KBID); err != nil {
			return nil, errcode.AppError{Code: errcode.ErrNotFound, Message: "知识库不存在"}
		}
	}
	if s.chatRepo == nil {
		return nil, errcode.AppError{Code: errcode.ErrUnknown, Message: "服务未初始化"}
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "新会话"
	}

	session := &model.ChatSession{
		UserID:   userID,
		KBID:     req.KBID,
		Question: title,
	}
	if err := s.chatRepo.Create(session); err != nil {
		return nil, errcode.AppError{Code: errcode.ErrUnknown, Message: "创建会话失败"}
	}

	return session, nil
}

// =============================================================================
// StreamChat — 流式对话（在已有会话中）
// =============================================================================

// StreamChat 在已有会话中发送消息并以流式事件通道返回 AI 答案。
//
// 会话必须已通过 CreateSession 创建。历史消息自动加载并注入 LLM 上下文。
// 流的 done 事件触发时，自动持久化 user+assistant 消息到 chat_messages 表，
// 并更新 chat_sessions 的 answer/confidence/duration_ms 字段。
//
// 单次 LLM 调用：用户看到的 token 与存入 DB 的答案完全一致。
func (s *ChatService) StreamChat(ctx context.Context, sessionID int64, question string, userID int64) (<-chan StreamEvent, error) {
	if strings.TrimSpace(question) == "" {
		return nil, errcode.AppError{Code: errcode.ErrParam, Message: "问题不能为空"}
	}
	if s.llmService == nil {
		return nil, errcode.AppError{Code: errcode.ErrAIUnavailable, Message: fallbackAIUnavailable}
	}
	if s.chatRepo == nil {
		return nil, errcode.AppError{Code: errcode.ErrUnknown, Message: "服务未初始化"}
	}

	// 加载会话并校验归属
	session, err := s.chatRepo.FindByID(sessionID)
	if err != nil {
		return nil, errcode.AppError{Code: errcode.ErrNotFound, Message: "会话不存在"}
	}
	if session.UserID != userID {
		return nil, errcode.AppError{Code: errcode.ErrForbidden, Message: "无权访问该会话"}
	}

	// 加载历史消息
	var history []adapter.ChatMessage
	msgs, _ := s.chatRepo.FindMessagesBySession(sessionID)
	for _, m := range msgs {
		history = append(history, adapter.ChatMessage{Role: m.Role, Content: m.Content})
	}

	opts := rag.RAGOptions{
		TopK:         s.defaultTopK,
		QueryRewrite: true,
		MultiRoute:   true,
		Hybrid:       true,
		Rerank:       true,
	}

	llmEvents, err := s.llmService.StreamChat(ctx, question, session.KBID, opts, history)
	if err != nil {
		return nil, errcode.AppError{Code: errcode.ErrRAGUnavailable, Message: err.Error()}
	}

	// 代理事件通道，done 时持久化消息
	outCh := make(chan StreamEvent, 100)
	go func() {
		defer close(outCh)
		for evt := range llmEvents {
			select {
			case <-ctx.Done():
				return
			default:
			}
			if evt.Type == "done" && evt.Metadata != nil && s.chatRepo != nil {
				srcJSON, _ := json.Marshal(evt.Metadata.Sources)

				// 更新会话摘要
				_ = s.chatRepo.UpdateSession(&model.ChatSession{
					ID:         sessionID,
					Answer:     evt.Metadata.Answer,
					Sources:    srcJSON,
					Confidence: evt.Metadata.Confidence,
					DurationMs: evt.Metadata.DurationMS,
				})

				// 持久化消息（user + assistant）
				_ = s.chatRepo.CreateBatch([]model.ChatMessage{
					{Role: "user", Content: question, SessionID: sessionID},
					{Role: "assistant", Content: evt.Metadata.Answer, SessionID: sessionID,
						Sources: srcJSON, Confidence: evt.Metadata.Confidence},
				})

				evt.Metadata.SessionID = sessionID
				evt.Metadata.Question = question
				evt.Metadata.Feedback = 0
				evt.Metadata.CreatedAt = time.Now().Format("2006-01-02 15:04:05")
			}
			if ok := sendOrCancel(ctx, outCh, evt); !ok {
				return
			}
		}
	}()

	return outCh, nil
}

// =============================================================================
// SubmitFeedback
// =============================================================================

// SubmitFeedback 提交问答反馈。
func (s *ChatService) SubmitFeedback(sessionID int64, feedback int16) error {
	if s.chatRepo == nil {
		return errcode.AppError{Code: errcode.ErrUnknown, Message: "服务未初始化"}
	}
	if _, err := s.chatRepo.FindByID(sessionID); err != nil {
		return errcode.AppError{Code: errcode.ErrNotFound, Message: "会话不存在"}
	}
	return s.chatRepo.UpdateFeedback(sessionID, feedback)
}

// =============================================================================
// GetChatDetail
// =============================================================================

// GetChatDetail 查询问答会话详情（含多轮对话消息历史）。
func (s *ChatService) GetChatDetail(sessionID int64) (*response.ChatSessionResponse, error) {
	if s.chatRepo == nil {
		return nil, errcode.AppError{Code: errcode.ErrUnknown, Message: "服务未初始化"}
	}
	session, err := s.chatRepo.FindByID(sessionID)
	if err != nil {
		return nil, errcode.AppError{Code: errcode.ErrNotFound, Message: "会话不存在"}
	}

	var sources []response.SourceItem
	if len(session.Sources) > 0 {
		json.Unmarshal(session.Sources, &sources)
	}

	// 加载消息历史
	var messages []response.MessageItem
	if msgs, msgErr := s.chatRepo.FindMessagesBySession(sessionID); msgErr == nil {
		for _, m := range msgs {
			var msgSources []response.SourceItem
			if len(m.Sources) > 0 {
				json.Unmarshal(m.Sources, &msgSources)
			}
			messages = append(messages, response.MessageItem{
				ID:         m.ID,
				Role:       m.Role,
				Content:    m.Content,
				Sources:    msgSources,
				Confidence: m.Confidence,
				CreatedAt:  m.CreatedAt.Format("2006-01-02 15:04:05"),
			})
		}
	}

	return &response.ChatSessionResponse{
		SessionID:       session.ID,
		Question:        session.Question,
		Answer:          session.Answer,
		Sources:         sources,
		Confidence:      session.Confidence,
		CanSubmitTicket: session.Confidence < defaultConfidenceThreshold,
		DurationMS:      session.DurationMs,
		Feedback:        session.Feedback,
		CreatedAt:       session.CreatedAt.Format("2006-01-02 15:04:05"),
		Messages:        messages,
	}, nil
}

// =============================================================================
// ListSessions — 会话列表
// =============================================================================

// ListSessions 分页查询用户的问答会话列表。
//
// 每条会话返回首轮问题标题 + 最后一条回复摘要 + 消息总数。
func (s *ChatService) ListSessions(userID int64, page, pageSize int) ([]response.SessionListItem, int64, error) {
	if s.chatRepo == nil {
		return nil, 0, errcode.AppError{Code: errcode.ErrUnknown, Message: "服务未初始化"}
	}
	sessions, total, err := s.chatRepo.ListByUser(userID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	items := make([]response.SessionListItem, 0, len(sessions))
	for _, sess := range sessions {
		count, _ := s.chatRepo.CountMessagesBySession(sess.ID)
		lastAnswer := truncateText(sess.Answer, 100)
		items = append(items, response.SessionListItem{
			ID:           sess.ID,
			Question:     sess.Question,
			LastAnswer:   lastAnswer,
			MessageCount: count,
			CreatedAt:    sess.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:    sess.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return items, total, nil
}

// DeleteSession 删除会话及其全部消息（含归属校验）。
func (s *ChatService) DeleteSession(sessionID, userID int64) error {
	if s.chatRepo == nil {
		return errcode.AppError{Code: errcode.ErrUnknown, Message: "服务未初始化"}
	}
	session, err := s.chatRepo.FindByID(sessionID)
	if err != nil {
		return errcode.AppError{Code: errcode.ErrNotFound, Message: "会话不存在"}
	}
	if session.UserID != userID {
		return errcode.AppError{Code: errcode.ErrForbidden, Message: "无权删除该会话"}
	}
	return s.chatRepo.DeleteSession(sessionID, userID)
}

// truncateText 截断文本到 maxRunes 个字符，超出加 "..."
func truncateText(text string, maxRunes int) string {
	runes := []rune(text)
	if len(runes) <= maxRunes {
		return text
	}
	return string(runes[:maxRunes]) + "..."
}
