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
	defaultTopK   int // 默认检索 TopK（从配置读取）
	knowledgeRepo chatKnowledgeRepo
	chatRepo      chatSessionRepo
	pipeline      chatPipeline
	llmClient     adapter.LLMClient
	configMgr     *LLMConfigManager
}

// NewChatService 创建 ChatService 实例。
//
// pipeline/llmClient/configMgr 可以为 nil（测试或降级场景）。
// knowledgeRepo/chatRepo/pipeline 直接使用具体接口类型，编译期校验传入类型。
func NewChatService(knowledgeRepo chatKnowledgeRepo, chatRepo chatSessionRepo, pipeline chatPipeline, llmClient adapter.LLMClient, configMgr *LLMConfigManager, defaultTopK int) *ChatService {
	if defaultTopK <= 0 {
		defaultTopK = 5
	}
	return &ChatService{
		knowledgeRepo: knowledgeRepo,
		chatRepo:      chatRepo,
		pipeline:      pipeline,
		llmClient:     llmClient,
		configMgr:     configMgr,
		defaultTopK:   defaultTopK,
	}
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
	// TODO(service/chat): 接收 context.Context 参数，避免 context.Background 让请求取消无法传播到 RAG/LLM/DB。
	// SSE 和非流式问答都应使用 c.Request.Context() 作为根上下文。
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
	// TODO(service/chat): 60s 超时应配置化，并按 RAG 检索和 LLM 生成拆分预算。
	// 当前一个 timeout 覆盖全部步骤，长生成可能挤占检索时间，短查询也无法快速失败。
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Step 1: Pipeline 检索（如果 pipeline 可用）
	var pipelineChunks []rag.RetrievalResult
	if s.pipeline != nil {
		// RAG 选项从 ChatService.defaultTopK 读取（源自配置 AI_DEFAULT_TOP_K）。
		// 各步骤开关当前默认开启，后续可按知识库粒度或请求级覆盖。
		// TODO(service/chat): req.RAGOptions 被忽略，前端高级设置不会真正影响后端管道。
		// 应将 request.RAGOptions 映射到 rag.RAGOptions，并对 TopK/RouteCount/RerankCount 做范围校验。
		result, pipeErr := s.pipeline.Execute(ctx, req.Question, req.KBID, rag.RAGOptions{
			TopK:         s.defaultTopK,
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
		// TODO(service/chat): 最终答案生成和 Handler.streamWithLLM 会各调用一次 LLM。
		// 应让 Service 暴露一个流式生成入口，避免先完整生成再重复流式生成导致答案不一致和成本翻倍。
		// Step 2: 构造带上下文的 prompt
		// 注意：system prompt 当前为全局默认值。后续可在知识库或 LLM 配置中
		// 增加 prompt_template 字段，支持按知识库定制角色设定。
		systemPrompt := "你是一个运维知识助手。根据以下知识库内容回答用户问题。如果知识库中没有相关信息，请如实说明。"
		var contextBuilder strings.Builder
		for i, chunk := range pipelineChunks {
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
			// TODO(service/chat): LLM 生成失败按 API 文档应返回 ErrAIUnavailable，而不是保存兜底答案为成功会话。
			// 需要区分“低置信度兜底”和“核心路径失败”两类场景。
			llmAnswer = "AI 服务不可用，请稍后重试或提交申告。"
			canSubmit = true
		} else {
			llmAnswer = llmResp.Content
		}
	} else {
		// 无 LLM 客户端：直接返回检索内容摘要
		var summary strings.Builder
		for i, chunk := range pipelineChunks {
			summary.WriteString(fmt.Sprintf("%d. %s\n", i+1, chunk.Content))
		}
		llmAnswer = "以下是与您问题相关的知识条目：\n\n" + summary.String()
	}

	durationMS := int(time.Since(start).Milliseconds())

	// Step 4: 保存会话
	//
	// 置信度取 RAG 检索结果中的最高分（RRF 融合后分数），无结果时默认 0。
	// 为什么用最高分而非平均分：多 chunk 场景中最高分反映最佳匹配质量，
	// 避免大量低相关 chunk 拉高或拉低均值。
	var confidence float64
	if len(pipelineChunks) > 0 {
		for _, c := range pipelineChunks {
			if c.Score > confidence {
				confidence = c.Score
			}
		}
	}
	// TODO(service/chat): 使用 RRF/BM25 原始分数直接和 0.6 阈值比较不可靠。
	// 应对不同检索来源做分数归一化或引入独立 confidence 估计，否则 can_submit_ticket 判断会漂移。
	session := &model.ChatSession{
		UserID:     userID,
		KBID:       req.KBID,
		Question:   req.Question,
		Answer:     llmAnswer,
		Confidence: confidence,
		DurationMs: durationMS,
	}
	if s.chatRepo != nil {
		// TODO(service/chat): sources 和 chat_messages 没有持久化。
		// GetChatDetail 读取 session.Sources，但创建会话时未写入来源，历史详情会缺少引用证据。
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
