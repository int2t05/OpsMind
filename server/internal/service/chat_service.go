// Package service 实现智能问答业务逻辑。
//
// ChatService 使用自建 RAG Pipeline（查询改写→多路检索→混合检索→重排序）
// 和 LLMClient 进行知识增强问答生成，支持 SSE 流式输出。
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
// llmService 统一管理 RAG+LLM 调用编排（流式/非流式）。
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
// CreateChatSession
// =============================================================================

// CreateChatSession 使用 RAG 管道 + LLM 创建问答会话。
//
// 支持多轮对话：req.SessionID > 0 时加载历史消息作为上下文，
// 并追加新消息到已有会话。
//
// 流程：
//  1. 校验参数 + 加载历史（多轮时）
//  2. LLMService.SyncChat（含历史上下文）
//  3. 保存/复用会话 + 持久化 user+assistant 消息
func (s *ChatService) CreateChatSession(req request.CreateChatRequest, userID int64) (*ChatSessionResponse, error) {
	if strings.TrimSpace(req.Question) == "" {
		return nil, errcode.AppError{Code: errcode.ErrParam, Message: "问题不能为空"}
	}
	if s.knowledgeRepo == nil {
		return nil, errcode.AppError{Code: errcode.ErrAIUnavailable, Message: fallbackAIUnavailable}
	}
	_, err := s.knowledgeRepo.FindKBByID(req.KBID)
	if err != nil {
		return nil, errcode.AppError{Code: errcode.ErrNotFound, Message: "知识库不存在"}
	}

	// 多轮对话：加载历史消息
	var history []adapter.ChatMessage
	var session *model.ChatSession
	isMultiTurn := req.SessionID > 0 && s.chatRepo != nil
	if isMultiTurn {
		session, err = s.chatRepo.FindByID(req.SessionID)
		if err != nil {
			return nil, errcode.AppError{Code: errcode.ErrNotFound, Message: "会话不存在"}
		}
		if session.UserID != userID {
			return nil, errcode.AppError{Code: errcode.ErrForbidden, Message: "无权访问该会话"}
		}
		msgs, _ := s.chatRepo.FindMessagesBySession(req.SessionID)
		for _, m := range msgs {
			history = append(history, adapter.ChatMessage{Role: m.Role, Content: m.Content})
		}
	}

	opts := rag.RAGOptions{
		TopK:         s.defaultTopK,
		QueryRewrite: true,
		MultiRoute:   true,
		Hybrid:       true,
		Rerank:       true,
	}

	var answer string
	var sources []response.SourceItem
	var confidence float64
	var pipeMeta *ChatPipelineMeta
	durationMS := 0

	if s.llmService != nil {
		// TODO(service/chat): 接收 context.Context 参数。
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		start := time.Now()
		result, syncErr := s.llmService.SyncChat(ctx, req.Question, req.KBID, opts, history)
		durationMS = int(time.Since(start).Milliseconds())
		if syncErr != nil {
			return nil, errcode.AppError{Code: errcode.ErrRAGUnavailable, Message: syncErr.Error()}
		}
		answer = result.Answer
		sources = result.Sources
		confidence = result.Confidence
		pipeMeta = result.Pipeline
	} else {
		answer = "当前 AI 服务暂不可用，请提交申告由人工处理"
	}

	canSubmit := len(sources) == 0 || confidence < defaultConfidenceThreshold

	// 持久化会话（新会话创建，已有会话复用）
	if s.chatRepo != nil {
		if !isMultiTurn {
			session = &model.ChatSession{
				UserID: userID, KBID: req.KBID, Question: req.Question,
				Answer: answer, Confidence: confidence, DurationMs: durationMS,
			}
			if len(sources) > 0 {
				if srcJSON, _ := json.Marshal(sources); len(srcJSON) > 0 {
					session.Sources = srcJSON
				}
			}
			if err := s.chatRepo.Create(session); err != nil {
				return nil, errcode.AppError{Code: errcode.ErrUnknown, Message: "保存会话失败"}
			}
		}

		// 持久化消息（user + assistant）
		srcJSON, _ := json.Marshal(sources)
		_ = s.chatRepo.CreateBatch([]model.ChatMessage{
			{Role: "user", Content: req.Question, SessionID: session.ID},
			{Role: "assistant", Content: answer, SessionID: session.ID,
				Sources: srcJSON, Confidence: confidence},
		})
	}

	return &ChatSessionResponse{
		SessionID:       session.ID,
		Question:        req.Question,
		Answer:          answer,
		Sources:         sources,
		Confidence:      confidence,
		CanSubmitTicket: canSubmit,
		DurationMS:      durationMS,
		Pipeline:        pipeMeta,
	}, nil
}

// =============================================================================
// SubmitFeedback
// =============================================================================

// SubmitFeedback 提交问答反馈。
func (s *ChatService) SubmitFeedback(sessionID int64, feedback int16) error {
	// TODO(service/chat): 校验 feedback 只能是 0/1/2 的规则应放在 Service 层。
	// TODO(service/chat): ChatService.pipeline 字段为死存储——ChatService 通过 LLMService 间接使用 pipeline，
	// 自身不再直接调用 pipeline.Execute。可移除该字段简化构造。
	// Handler 已校验但其他调用方或测试替身仍可能绕过。
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
	// TODO(service/chat): 应接收 currentUserID 校验会话归属。
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
// 每条会话返回首轮问题 + 最后一条回复摘要 + 消息总数。
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

// =============================================================================
// StreamChat — SSE 流式问答
// =============================================================================

// StreamChat 创建/追加问答会话并以流式事件通道返回。
//
// 支持多轮：req.SessionID > 0 时加载历史注入上下文，追加消息到已有会话。
// 单次 LLM 调用：用户看到的 token 与最终存入 DB 的答案完全一致。
func (s *ChatService) StreamChat(ctx context.Context, req request.CreateChatRequest, userID int64) (<-chan StreamEvent, error) {
	if strings.TrimSpace(req.Question) == "" {
		return nil, errcode.AppError{Code: errcode.ErrParam, Message: "问题不能为空"}
	}
	if s.knowledgeRepo == nil {
		return nil, errcode.AppError{Code: errcode.ErrAIUnavailable, Message: fallbackAIUnavailable}
	}
	_, err := s.knowledgeRepo.FindKBByID(req.KBID)
	if err != nil {
		return nil, errcode.AppError{Code: errcode.ErrNotFound, Message: "知识库不存在"}
	}

	// 多轮对话：加载历史消息
	var history []adapter.ChatMessage
	var session *model.ChatSession
	isMultiTurn := req.SessionID > 0 && s.chatRepo != nil
	if isMultiTurn {
		session, err = s.chatRepo.FindByID(req.SessionID)
		if err != nil {
			return nil, errcode.AppError{Code: errcode.ErrNotFound, Message: "会话不存在"}
		}
		if session.UserID != userID {
			return nil, errcode.AppError{Code: errcode.ErrForbidden, Message: "无权访问该会话"}
		}
		msgs, _ := s.chatRepo.FindMessagesBySession(req.SessionID)
		for _, m := range msgs {
			history = append(history, adapter.ChatMessage{Role: m.Role, Content: m.Content})
		}
	}

	opts := rag.RAGOptions{
		TopK:         s.defaultTopK,
		QueryRewrite: true,
		MultiRoute:   true,
		Hybrid:       true,
		Rerank:       true,
	}

	if s.llmService == nil {
		return nil, errcode.AppError{Code: errcode.ErrAIUnavailable, Message: fallbackAIUnavailable}
	}

	llmEvents, err := s.llmService.StreamChat(ctx, req.Question, req.KBID, opts, history)
	if err != nil {
		return nil, errcode.AppError{Code: errcode.ErrRAGUnavailable, Message: err.Error()}
	}

	// 代理事件通道，done 时持久化 session + messages
	outCh := make(chan StreamEvent, 100)

	// 闭包捕获 isMultiTurn / sessionID
	sessionID := req.SessionID
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
				if !isMultiTurn {
					// 新会话：创建
					sess := &model.ChatSession{
						UserID: userID, KBID: req.KBID, Question: req.Question,
						Answer: evt.Metadata.Answer, Sources: srcJSON,
						Confidence: evt.Metadata.Confidence, DurationMs: evt.Metadata.DurationMS,
					}
					if err := s.chatRepo.Create(sess); err == nil {
						sessionID = sess.ID
					}
				} else {
					// 已有会话：更新 answer
					s.chatRepo.UpdateSession(&model.ChatSession{
						ID: sessionID, Answer: evt.Metadata.Answer,
						Sources: srcJSON, Confidence: evt.Metadata.Confidence,
						DurationMs: evt.Metadata.DurationMS,
					})
				}

				// 持久化消息
				_ = s.chatRepo.CreateBatch([]model.ChatMessage{
					{Role: "user", Content: req.Question, SessionID: sessionID},
					{Role: "assistant", Content: evt.Metadata.Answer, SessionID: sessionID,
						Sources: srcJSON, Confidence: evt.Metadata.Confidence},
				})

				evt.Metadata.SessionID = sessionID
				evt.Metadata.Question = req.Question
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
// 辅助类型
// =============================================================================

// ChatSessionResponse 问答响应（供 Handler 层 SSE 流式输出使用）。
type ChatSessionResponse struct {
	SessionID       int64                   `json:"session_id"`
	Question        string                  `json:"question"`
	Answer          string                  `json:"answer"`
	Sources         []response.SourceItem   `json:"sources,omitempty"`
	Confidence      float64                 `json:"confidence"`
	CanSubmitTicket bool                    `json:"can_submit_ticket"`
	DurationMS      int                     `json:"duration_ms"`
	Pipeline        *ChatPipelineMeta       `json:"pipeline,omitempty"`
}

// ChatPipelineMeta 管道执行元数据。
type ChatPipelineMeta struct {
	Steps           []ChatPipelineStep `json:"steps"`
	TotalDurationMS int                `json:"total_duration_ms"`
}

// ChatPipelineStep 管道单步骤耗时。
type ChatPipelineStep struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	DurationMS int    `json:"duration_ms"`
}
