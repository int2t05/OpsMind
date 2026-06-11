// Package service 实现智能问答业务逻辑。
//
// chat_service_v2.go 提供 v2 版 ChatService（依赖自建 RAG Pipeline + LLMClient）。
//
// v1→v2 变更：
//   - 移除 RagClient（AnythingLLM）依赖
//   - 新增 rag.Pipeline（检索） + adapter.LLMClient（生成）
//   - CreateChatSession 改为调用 Pipeline.Execute → LLMClient.ChatCompletion
//   - 置信度基于检索结果数判断（而非 AnythingLLM 返回的 confidence 字段）
package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"opsmind/internal/adapter"
	"opsmind/internal/model"
	"opsmind/internal/rag"
	"opsmind/pkg/errcode"
)

// =============================================================================
// V2 依赖接口（消费者定义）
// =============================================================================

// chatKnowledgeRepo v2 知识库仓库接口（仅暴露 ChatService 需要的方法）。
type chatKnowledgeRepo interface {
	FindKBByID(id int64) (*model.KnowledgeBase, error)
}

// chatSessionRepo v2 会话仓库接口。
type chatSessionRepo interface {
	Create(session *model.ChatSession) error
	CreateBatch(messages []model.ChatMessage) error
	FindByID(id int64) (*model.ChatSession, error)
	UpdateFeedback(id int64, feedback int16) error
}

// chatPipeline v2 RAG 管道接口。
type chatPipeline interface {
	Execute(ctx context.Context, query string, kbID int64, opts rag.RAGOptions, onStep rag.StepCallback) (*rag.RAGResult, error)
}

// =============================================================================
// ChatServiceV2
// =============================================================================

// ChatServiceV2 使用自建 RAG 管道的问答服务。
type ChatServiceV2 struct {
	knowledgeRepo chatKnowledgeRepo
	chatRepo      chatSessionRepo
	pipeline      chatPipeline
	llmClient     adapter.LLMClient
	configMgr     *LLMConfigManager
}

// NewChatServiceV2 创建 ChatServiceV2 实例。
//
// configMgr 可以为 nil（测试或用默认配置时）。
func NewChatServiceV2(knowledgeRepo interface{}, chatRepo interface{}, pipeline interface{}, llmClient adapter.LLMClient, configMgr *LLMConfigManager) *ChatServiceV2 {
	svc := &ChatServiceV2{
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
// CreateChatSessionV2
// =============================================================================

// CreateChatSessionV2 使用 v2 Pipeline 创建问答会话。
//
// 流程：
//  1. Pipeline.Execute（查询改写→多路检索→混合检索→重排序）
//  2. 构造带上下文的 LLM prompt
//  3. LLMClient.ChatCompletion 生成答案
//  4. 保存会话和消息
func (s *ChatServiceV2) CreateChatSessionV2(question string, kbID int64, userID int64, opts rag.RAGOptions) (*ChatSessionResponseV2, error) {
	if strings.TrimSpace(question) == "" {
		return nil, AppError{Code: errcode.ErrParam, Message: "问题不能为空"}
	}

	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Step 1: Pipeline 检索
	pipelineResult, err := s.pipeline.Execute(ctx, question, kbID, opts, nil)
	if err != nil {
		return nil, AppError{Code: errcode.ErrRAGUnavailable, Message: "知识检索失败: " + err.Error()}
	}

	// Step 2: 构造 LLM prompt（含检索上下文）
	var llmAnswer string
	canSubmit := false

	if len(pipelineResult.Chunks) == 0 {
		// 无检索结果 → 降级：引导提交申告
		llmAnswer = "暂未找到足够匹配的知识，建议提交申告由运维人员人工处理。"
		canSubmit = true
	} else {
		// 构造带上下文的 prompt
		systemPrompt := "你是一个运维知识助手。根据以下知识库内容回答用户问题。如果知识库中没有相关信息，请如实说明。"
		var contextBuilder strings.Builder
		for i, chunk := range pipelineResult.Chunks {
			if i >= 3 { // 最多 3 条上下文
				break
			}
			contextBuilder.WriteString(fmt.Sprintf("【参考资料 %d】%s\n", i+1, chunk.Content))
		}

		messages := []adapter.ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: fmt.Sprintf("知识库内容：\n%s\n\n用户问题：%s", contextBuilder.String(), question)},
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
			return nil, AppError{Code: errcode.ErrAIUnavailable, Message: "AI 服务不可用，请稍后重试"}
		}
		llmAnswer = llmResp.Content
	}

	durationMS := int(time.Since(start).Milliseconds())

	// Step 4: 保存会话
	session := &model.ChatSession{
		UserID:     userID,
		KBID:       kbID,
		Question:   question,
		Answer:     llmAnswer,
		Confidence: float64(len(pipelineResult.Chunks)) * 0.3, // 简单置信度：chunk 数量映射
		DurationMs: durationMS,
	}
	if err := s.chatRepo.Create(session); err != nil {
		return nil, AppError{Code: errcode.ErrUnknown, Message: "保存会话失败"}
	}

	return &ChatSessionResponseV2{
		SessionID:       session.ID,
		Question:        question,
		Answer:          llmAnswer,
		Confidence:      session.Confidence,
		CanSubmitTicket: canSubmit,
		DurationMS:      durationMS,
	}, nil
}

// =============================================================================
// 辅助类型
// =============================================================================

// ChatSessionResponseV2 v2 问答响应。
type ChatSessionResponseV2 struct {
	SessionID       int64   `json:"session_id"`
	Question        string  `json:"question"`
	Answer          string  `json:"answer"`
	Confidence      float64 `json:"confidence"`
	CanSubmitTicket bool    `json:"can_submit_ticket"`
	DurationMS      int     `json:"duration_ms"`
}
