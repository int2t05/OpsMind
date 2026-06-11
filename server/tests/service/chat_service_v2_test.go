package service_test

import (
	"context"
	"fmt"
	"testing"

	"opsmind/internal/adapter"
	"opsmind/internal/model"
	"opsmind/internal/rag"
	"opsmind/internal/service"
)

// =============================================================================
// v2 mocks
// =============================================================================

type mockChatKnowledgeRepo struct {
	kb *model.KnowledgeBase
}

func (m *mockChatKnowledgeRepo) FindKBByID(id int64) (*model.KnowledgeBase, error) {
	if m.kb == nil {
		return nil, fmt.Errorf("not found")
	}
	return m.kb, nil
}

type mockChatSessionRepo struct {
	lastSession *model.ChatSession
}

func (m *mockChatSessionRepo) Create(session *model.ChatSession) error {
	session.ID = 1
	m.lastSession = session
	return nil
}
func (m *mockChatSessionRepo) CreateBatch(msgs []model.ChatMessage) error { return nil }
func (m *mockChatSessionRepo) FindByID(id int64) (*model.ChatSession, error) {
	if m.lastSession != nil {
		return m.lastSession, nil
	}
	return nil, fmt.Errorf("not found")
}
func (m *mockChatSessionRepo) UpdateFeedback(id int64, fb int16) error { return nil }

type mockChatLLMClient struct {
	response *adapter.ChatResponse
	err      error
}

func (m *mockChatLLMClient) ChatCompletion(ctx context.Context, req adapter.ChatRequest) (*adapter.ChatResponse, error) {
	return m.response, m.err
}
func (m *mockChatLLMClient) ChatCompletionStream(ctx context.Context, req adapter.ChatRequest) (<-chan adapter.StreamChunk, error) {
	return nil, fmt.Errorf("not implemented")
}

type mockChatPipeline struct {
	result *rag.RAGResult
	err    error
}

func (m *mockChatPipeline) Execute(ctx context.Context, query string, kbID int64, opts rag.RAGOptions, onStep rag.StepCallback) (*rag.RAGResult, error) {
	return m.result, m.err
}

// =============================================================================
// 测试用例
// =============================================================================

// TestChatV2_Success 验证 v2 ChatService 使用 Pipeline 检索 + LLM 生成。
func TestChatV2_Success(t *testing.T) {
	kbRepo := &mockChatKnowledgeRepo{
		kb: &model.KnowledgeBase{ID: 1, Name: "测试知识库"},
	}
	sessionRepo := &mockChatSessionRepo{}
	pipeline := &mockChatPipeline{
		result: &rag.RAGResult{
			Chunks: []rag.RetrievalResult{
				{ChunkID: 1, Content: "VPN配置步骤：1.下载客户端 2.输入地址", Score: 0.9, Source: "hybrid"},
			},
			Metrics: rag.PipelineMetrics{TotalDurationMS: 150},
		},
	}
	llm := &mockChatLLMClient{
		response: &adapter.ChatResponse{Content: "VPN配置方法如下...", FinishReason: "stop"},
	}

	svc := service.NewChatServiceV2(kbRepo, sessionRepo, pipeline, llm, nil)

	resp, err := svc.CreateChatSessionV2("VPN怎么配置", 1, 1, rag.DefaultRAGOptions())
	if err != nil {
		t.Fatalf("CreateChatSessionV2 失败: %v", err)
	}
	if resp.Answer == "" {
		t.Error("回答不应为空")
	}
	if resp.Confidence <= 0 {
		t.Error("应返回正置信度")
	}
}

// TestChatV2_RAGFail 验证 RAG 检索失败时的降级。
func TestChatV2_RAGFail(t *testing.T) {
	kbRepo := &mockChatKnowledgeRepo{
		kb: &model.KnowledgeBase{ID: 1, Name: "测试"},
	}
	sessionRepo := &mockChatSessionRepo{}
	pipeline := &mockChatPipeline{
		err: fmt.Errorf("pgvector 不可用"),
	}
	llm := &mockChatLLMClient{
		response: &adapter.ChatResponse{Content: "回答", FinishReason: "stop"},
	}

	svc := service.NewChatServiceV2(kbRepo, sessionRepo, pipeline, llm, nil)

	_, err := svc.CreateChatSessionV2("test", 1, 1, rag.DefaultRAGOptions())
	if err == nil {
		t.Error("RAG 检索失败应返回错误")
	}
}

// TestChatV2_LLMFail 验证 LLM 生成失败时的降级。
func TestChatV2_LLMFail(t *testing.T) {
	kbRepo := &mockChatKnowledgeRepo{
		kb: &model.KnowledgeBase{ID: 1, Name: "测试"},
	}
	sessionRepo := &mockChatSessionRepo{}
	pipeline := &mockChatPipeline{
		result: &rag.RAGResult{
			Chunks: []rag.RetrievalResult{
				{ChunkID: 1, Content: "内容", Score: 0.9, Source: "vector"},
			},
		},
	}
	llm := &mockChatLLMClient{
		err: fmt.Errorf("LLM 服务不可用"),
	}

	svc := service.NewChatServiceV2(kbRepo, sessionRepo, pipeline, llm, nil)

	_, err := svc.CreateChatSessionV2("test", 1, 1, rag.DefaultRAGOptions())
	if err == nil {
		t.Error("LLM 失败应返回错误")
	}
}

// TestChatV2_LowConfidence 验证低置信度时引导提交申告。
func TestChatV2_LowConfidence(t *testing.T) {
	kbRepo := &mockChatKnowledgeRepo{
		kb: &model.KnowledgeBase{ID: 1, Name: "测试"},
	}
	sessionRepo := &mockChatSessionRepo{}
	pipeline := &mockChatPipeline{
		result: &rag.RAGResult{
			Chunks:  []rag.RetrievalResult{},
			Metrics: rag.PipelineMetrics{TotalDurationMS: 50},
		},
	}
	llm := &mockChatLLMClient{
		response: &adapter.ChatResponse{Content: "答案", FinishReason: "stop"},
	}

	svc := service.NewChatServiceV2(kbRepo, sessionRepo, pipeline, llm, nil)

	resp, err := svc.CreateChatSessionV2("test", 1, 1, rag.DefaultRAGOptions())
	if err != nil {
		t.Fatalf("不应报错: %v", err)
	}
	if !resp.CanSubmitTicket {
		t.Error("无检索结果时 CanSubmitTicket 应为 true")
	}
}
