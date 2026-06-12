// Package service 实现智能问答业务逻辑。
//
// ChatService 使用自建 RAG Pipeline（查询改写→多路检索→混合检索→重排序）
// 和 LLMClient 进行知识增强问答生成，支持 SSE 流式输出。
package service

import (
	"context"
	"encoding/json"
	"fmt"
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
	UpdateFeedback(id int64, feedback int16) error
}

type chatPipeline interface {
	Execute(ctx context.Context, query string, kbID int64, opts rag.RAGOptions, onStep rag.StepCallback) (*rag.RAGResult, error)
}

// ChatService 智能问答服务。
//
// knowledgeRepo/chatRepo/pipeline 使用接口类型，便于测试 mock。
// llmClient 使用 adapter.LLMClient 接口，configMgr 可以为 nil。
type ChatService struct {
	knowledgeRepo chatKnowledgeRepo
	chatRepo      chatSessionRepo
	pipeline      chatPipeline
	llmClient     adapter.LLMClient
	configMgr     *LLMConfigManager
}

// NewChatService 创建 ChatService 实例。
//
// pipeline/llmClient/configMgr 可以为 nil（测试或降级场景）。
// knowledgeRepo/chatRepo 接受接口类型，repository 具体类型隐式满足。
// TODO: 构造函数接受 interface{} 绕过编译期类型检查 — 传入错误类型时静默 nil。
// 应直接使用具体接口类型（chatKnowledgeRepo 等），repository 具体类型隐式满足接口。
func NewChatService(knowledgeRepo interface{}, chatRepo interface{}, pipeline interface{}, llmClient adapter.LLMClient, configMgr *LLMConfigManager) *ChatService {
	svc := &ChatService{
		llmClient: llmClient,
		configMgr: configMgr,
	}
	if r, ok := knowledgeRepo.(chatKnowledgeRepo); ok {
		svc.knowledgeRepo = r
	}
	if r, ok := chatRepo.(chatSessionRepo); ok {
		svc.chatRepo = r
	}
	if p, ok := pipeline.(chatPipeline); ok {
		svc.pipeline = p
	}
	return svc
}

// =============================================================================
// CreateChatSession
// =============================================================================

// CreateChatSession 使用 Pipeline 创建问答会话。
//
// 流程：
//  1. Pipeline.Execute（查询改写→多路检索→混合检索→重排序）
//  2. 构造带上下文的 LLM prompt
//  3. LLMClient.ChatCompletion 生成答案
//  4. 保存会话
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

	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Step 1: Pipeline 检索（如果 pipeline 可用）
	var pipelineChunks []rag.RetrievalResult
	if s.pipeline != nil {
		// TODO: TopK 和各步骤开关均硬编码，未使用 AI_DEFAULT_TOP_K 配置。
		// 应支持：1) 从 config 读取默认值；2) 按知识库粒度覆盖；3) 请求级覆盖。
		result, pipeErr := s.pipeline.Execute(ctx, req.Question, req.KBID, rag.RAGOptions{
			TopK:         5,
			QueryRewrite: true,
			MultiRoute:   true,
			Hybrid:       true,
			Rerank:       true,
		}, nil)
		if pipeErr != nil {
			return nil, errcode.AppError{Code: errcode.ErrRAGUnavailable, Message: "知识检索失败: " + pipeErr.Error()}
		}
		if result != nil {
			pipelineChunks = result.Chunks
		}
	}

	var llmAnswer string
	canSubmit := false

	if len(pipelineChunks) == 0 {
		llmAnswer = "暂未找到足够匹配的知识，建议提交申告由运维人员人工处理。"
		canSubmit = true
	} else if s.llmClient != nil {
		// Step 2: 构造带上下文的 prompt
		// TODO: system prompt 硬编码，不同知识库可能需要不同角色设定（如「网络运维」「DBA」）。
		// 应在知识库或 LLM 配置中支持 prompt_template 字段。
		systemPrompt := "你是一个运维知识助手。根据以下知识库内容回答用户问题。如果知识库中没有相关信息，请如实说明。"
		var contextBuilder strings.Builder
		// TODO: 仅取前 3 个 chunk 注入 LLM，而检索返回 TopK=5，浪费了后 2 个结果。
		// 应改为可配置的 context_chunk_count 或全部使用 TopK 的结果。
		for i, chunk := range pipelineChunks {
			if i >= 3 {
				break
			}
			contextBuilder.WriteString(fmt.Sprintf("【参考资料 %d】%s\n", i+1, chunk.Content))
		}

		messages := []adapter.ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: fmt.Sprintf("知识库内容：\n%s\n\n用户问题：%s", contextBuilder.String(), req.Question)},
		}

		// Step 3: LLM 生成
		model := "default"
		maxTokens := 2048
		if s.configMgr != nil {
			if cfg := s.configMgr.GetConfig(); cfg != nil {
				model = cfg.LLMModel
				maxTokens = cfg.MaxTokens
			}
		}

		llmResp, llmErr := s.llmClient.ChatCompletion(ctx, adapter.ChatRequest{
			Messages:    messages,
			Model:       model,
			MaxTokens:   maxTokens,
			Temperature: 0.3,
		})
		if llmErr != nil {
			llmAnswer = "AI 服务不可用，请稍后重试或提交申告。"
			canSubmit = true
		} else {
			llmAnswer = llmResp.Content
		}
	} else {
		// 无 LLM 客户端：直接返回检索内容摘要
		var summary strings.Builder
		for i, chunk := range pipelineChunks {
			if i >= 3 {
				break
			}
			summary.WriteString(fmt.Sprintf("%d. %s\n", i+1, chunk.Content))
		}
		llmAnswer = "以下是与您问题相关的知识条目：\n\n" + summary.String()
	}

	durationMS := int(time.Since(start).Milliseconds())

	// Step 4: 保存会话
	//
	// TODO: 当前置信度仅按检索命中 chunk 数量 × 0.3 计算，过于粗糙。
	// 问题：5 个不相关 chunk 可得 1.5（不触发申告），1 个高相关 chunk 只得 0.3（触发申告）。
	// 应改为取 RRF 融合后的最高分或平均分作为置信度，更能反映检索质量。
	confidence := float64(len(pipelineChunks)) * 0.3
	session := &model.ChatSession{
		UserID:     userID,
		KBID:       req.KBID,
		Question:   req.Question,
		Answer:     llmAnswer,
		Confidence: confidence,
		DurationMs: durationMS,
	}
	if s.chatRepo != nil {
		if err := s.chatRepo.Create(session); err != nil {
			return nil, errcode.AppError{Code: errcode.ErrUnknown, Message: "保存会话失败"}
		}
	}

	return &ChatSessionResponse{
		SessionID:       session.ID,
		Question:        req.Question,
		Answer:          llmAnswer,
		Confidence:      session.Confidence,
		CanSubmitTicket: canSubmit,
		DurationMS:      durationMS,
	}, nil
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

// GetChatDetail 查询问答会话详情。
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
