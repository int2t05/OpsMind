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
	UpdateFeedback(id int64, feedback int16) error
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
// 流程：
//  1. 校验参数
//  2. LLMService.SyncChat（RAG 检索 + prompt 构建 + LLM 同步生成）
//  3. 保存会话到 DB
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

	// 构建 RAGOptions（当前使用默认值，后续可从 req.RAGOptions 映射）
	// TODO(service/chat): req.RAGOptions 被忽略，前端高级设置不会真正影响后端管道。
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
		// TODO(service/chat): 接收 context.Context 参数，避免 context.Background。
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		start := time.Now()
		result, syncErr := s.llmService.SyncChat(ctx, req.Question, req.KBID, opts)
		durationMS = int(time.Since(start).Milliseconds())
		if syncErr != nil {
			return nil, errcode.AppError{Code: errcode.ErrRAGUnavailable, Message: syncErr.Error()}
		}
		answer = result.Answer
		sources = result.Sources
		confidence = result.Confidence
		pipeMeta = result.Pipeline
	} else {
		// 无 LLMService：降级提示
		answer = "当前 AI 服务暂不可用，请提交申告由人工处理"
	}

	canSubmit := len(sources) == 0 || confidence < defaultConfidenceThreshold

	// 保存会话
	sess := &model.ChatSession{
		UserID:     userID,
		KBID:       req.KBID,
		Question:   req.Question,
		Answer:     answer,
		Confidence: confidence,
		DurationMs: durationMS,
	}
	if len(sources) > 0 {
		if srcJSON, err := json.Marshal(sources); err == nil {
			sess.Sources = srcJSON
		}
	}
	if s.chatRepo != nil {
		if err := s.chatRepo.Create(sess); err != nil {
			return nil, errcode.AppError{Code: errcode.ErrUnknown, Message: "保存会话失败"}
		}
	}

	return &ChatSessionResponse{
		SessionID:       sess.ID,
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

// GetChatDetail 查询问答会话详情。
func (s *ChatService) GetChatDetail(sessionID int64) (*response.ChatSessionResponse, error) {
	// TODO(service/chat): GetChatDetail 未校验 session.UserID，门户端用户可查询他人的会话 ID。
	// 应接收 currentUserID 或拆分 Admin/Portal 查询接口。
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
	}, nil
}

// =============================================================================
// StreamChat — SSE 流式问答
// =============================================================================

// StreamChat 创建问答会话并以流式事件通道返回。
//
// 流程：
//  1. 校验参数
//  2. LLMService.StreamChat 获取事件通道
//  3. goroutine 代理事件，done 时创建 session 填入 session_id
//
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

	llmEvents, err := s.llmService.StreamChat(ctx, req.Question, req.KBID, opts)
	if err != nil {
		return nil, errcode.AppError{Code: errcode.ErrRAGUnavailable, Message: err.Error()}
	}

	// 代理事件通道，done 时持久化 session
	outCh := make(chan StreamEvent, 100)
	go func() {
		defer close(outCh)
		for evt := range llmEvents {
			select {
			case <-ctx.Done():
				return
			default:
			}
			// done 事件到达：创建 session 并回填 session_id
			if evt.Type == "done" && evt.Metadata != nil && s.chatRepo != nil {
				srcJSON, _ := json.Marshal(evt.Metadata.Sources)
				sess := &model.ChatSession{
					UserID:     userID,
					KBID:       req.KBID,
					Question:   req.Question,
					Answer:     evt.Metadata.Answer,
					Sources:    srcJSON,
					Confidence: evt.Metadata.Confidence,
					DurationMs: evt.Metadata.DurationMS,
				}
				if err := s.chatRepo.Create(sess); err == nil {
					evt.Metadata.SessionID = sess.ID
					evt.Metadata.Question = req.Question
					evt.Metadata.Feedback = 0
					evt.Metadata.CreatedAt = time.Now().Format("2006-01-02 15:04:05")
				}
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
