// Package service 实现智能问答业务逻辑。
//
// ChatService 提供问答会话创建、置信度判断、降级处理和反馈管理功能。
//
// 问答核心流程：
//  1. 查询知识库获取 RAG workspace slug
//  2. 调用 RagClient.Query 获取 AI 答案
//  3. 根据置信度阈值判断是否需要转人工
//  4. 保存会话和消息到数据库
//
// 为什么置信度判断在 Service 层而非 Handler 层：
// 阈值来自系统配置，且判断逻辑涉及多个条件（sources 为空/confidence < threshold/RAG error），
// Service 层集中处理便于测试和后续调整。
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
	"opsmind/internal/repository"
	"opsmind/pkg/errcode"

	"gorm.io/datatypes"
)

const (
	// 置信度阈值：低于此值判定为低置信度，引导用户提交申告。
	// MVP 阶段硬编码，后续从 system_configs 表读取。
	defaultConfidenceThreshold = 0.6

	// 降级兜底文本（与 ANYTHINGLLM_AI_INTEGRATION.md §7.1 对齐）
	fallbackLowConfidence = "暂未找到足够匹配的知识，建议提交申告由运维人员人工处理"
	fallbackAIUnavailable  = "当前 AI 服务暂不可用，请提交申告由人工处理"
)

// ChatService 智能问答服务。
type ChatService struct {
	knowledgeRepo *repository.KnowledgeRepo
	chatRepo      *repository.ChatRepo
	ragClient     adapter.RagClient
}

// NewChatService 创建 ChatService 实例。
func NewChatService(knowledgeRepo *repository.KnowledgeRepo, chatRepo *repository.ChatRepo, ragClient adapter.RagClient) *ChatService {
	return &ChatService{
		knowledgeRepo: knowledgeRepo,
		chatRepo:      chatRepo,
		ragClient:     ragClient,
	}
}

// =============================================================================
// CreateChatSession
// =============================================================================

// CreateChatSession 创建问答会话并返回 AI 答案。
//
// 核心流程：
//  1. 参数校验（问题非空、知识库存在）
//  2. 调用 RagClient.Query 获取 AI 答案
//  3. 置信度判断和降级处理
//  4. 保存 ChatSession 和 2 条 ChatMessage（用户问题 + AI 回答）
//  5. 返回 ChatSessionResponse
//
// 降级规则（与集成方案 §10.1 对齐）：
//  - RagClient 不可达（网络错误）→ 返回 AppError code=20001
//  - AnythingLLM 返回 error != null → 返回兜底答案 + can_submit_ticket=true
//  - confidence < 0.6 或 sources 为空 → 返回兜底答案 + can_submit_ticket=true
//  - 正常且置信度达标 → 返回 AI 答案 + can_submit_ticket=false
func (s *ChatService) CreateChatSession(req request.CreateChatRequest, userID int64) (*response.ChatSessionResponse, error) {
	// 参数校验
	if strings.TrimSpace(req.Question) == "" {
		return nil, AppError{Code: errcode.ErrParam, Message: "问题不能为空"}
	}

	// 查询知识库获取 workspace slug
	kb, err := s.knowledgeRepo.FindKBByID(req.KBID)
	if err != nil {
		return nil, AppError{Code: errcode.ErrNotFound, Message: "知识库不存在"}
	}

	start := time.Now()

	// 调用 RagClient.Query
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ragResp, err := s.ragClient.Query(ctx, adapter.RAGQueryRequest{
		WorkspaceSlug: kb.RAGWorkspaceSlug,
		Question:      req.Question,
		TopK:          5,
	})

	// RagClient 不可达 → 返回 20001
	if err != nil {
		return nil, AppError{Code: errcode.ErrAIUnavailable, Message: fallbackAIUnavailable}
	}

	durationMS := int(time.Since(start).Milliseconds())

	// 构建响应
	var answer string
	var sources []response.SourceItem
	var confidence float64
	var canSubmit bool

	// AnythingLLM 返回 error 或 sources 为空 → 降级
	if ragResp.Error != "" || len(ragResp.Sources) == 0 || ragResp.Confidence < defaultConfidenceThreshold {
		answer = fallbackLowConfidence
		confidence = ragResp.Confidence
		canSubmit = true
	} else {
		answer = ragResp.Answer
		confidence = ragResp.Confidence
		canSubmit = false
		sources = make([]response.SourceItem, len(ragResp.Sources))
		for i, src := range ragResp.Sources {
			sources[i] = response.SourceItem{
				DocName:      src.DocName,
				ChunkContent: src.ChunkContent,
				Confidence:   src.Confidence,
			}
		}
	}

	// 序列化 sources 为 JSONB
	var sourcesJSON datatypes.JSON
	if len(sources) > 0 {
		if b, err := json.Marshal(sources); err == nil {
			sourcesJSON = datatypes.JSON(b)
		}
	}

	// 保存 ChatSession
	session := &model.ChatSession{
		UserID:     userID,
		KBID:       req.KBID,
		Question:   req.Question,
		Answer:     answer,
		Sources:    sourcesJSON,
		Confidence: confidence,
		DurationMs: durationMS,
	}
	if err := s.chatRepo.Create(session); err != nil {
		return nil, AppError{Code: errcode.ErrUnknown, Message: "保存会话失败"}
	}

	// 保存 2 条 ChatMessage（用户问题 + AI 回答）
	now := time.Now()
	messages := []model.ChatMessage{
		{SessionID: session.ID, Role: "user", Content: req.Question, CreatedAt: now},
		{SessionID: session.ID, Role: "assistant", Content: answer, Sources: sourcesJSON, Confidence: confidence, CreatedAt: now},
	}
	if err := s.chatRepo.CreateBatch(messages); err != nil {
		// 消息写入失败不影响会话查询，仅记录（MVP 阶段不阻塞主流程）
		// 后续可引入异步重试
		_ = err
	}

	return &response.ChatSessionResponse{
		SessionID:       session.ID,
		Question:        req.Question,
		Answer:          answer,
		Sources:         sources,
		Confidence:      confidence,
		CanSubmitTicket: canSubmit,
		DurationMS:      durationMS,
		Feedback:        session.Feedback,
		CreatedAt:       session.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

// =============================================================================
// SubmitFeedback
// =============================================================================

// SubmitFeedback 提交问答反馈。
//
// feedback: 0=未评价, 1=已解决, 2=未解决。
// 为什么直接接受 int16：Repository 层和 Model 层均使用 int16 存储，
// 在 Service 层保持类型一致避免不必要的转换。
func (s *ChatService) SubmitFeedback(sessionID int64, feedback int16) error {
	// 先检查会话是否存在
	if _, err := s.chatRepo.FindByID(sessionID); err != nil {
		return AppError{Code: errcode.ErrNotFound, Message: "会话不存在"}
	}
	return s.chatRepo.UpdateFeedback(sessionID, feedback)
}

// =============================================================================
// GetChatDetail
// =============================================================================

// GetChatDetail 查询问答会话详情。
//
// 返回 ChatSession 的基础信息，不含 messages 列表（消息通过独立接口查询）。
func (s *ChatService) GetChatDetail(sessionID int64) (*response.ChatSessionResponse, error) {
	session, err := s.chatRepo.FindByID(sessionID)
	if err != nil {
		return nil, AppError{Code: errcode.ErrNotFound, Message: "会话不存在"}
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
