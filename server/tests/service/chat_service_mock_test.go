package service_test

import (
	"context"
	"fmt"
	"testing"

	"opsmind/internal/adapter"
	"opsmind/internal/dto/request"
	"opsmind/internal/model"
	"opsmind/internal/rag"
	"opsmind/internal/service"
)

// =============================================================================
// Mock 组件
// =============================================================================

type mockChatKBRepo struct {
	kb *model.KnowledgeBase
}

func (m *mockChatKBRepo) FindKBByID(id int64) (*model.KnowledgeBase, error) {
	if m.kb == nil {
		return nil, fmt.Errorf("not found")
	}
	return m.kb, nil
}

type mockChatSessRepo struct {
	lastSession *model.ChatSession
}

func (m *mockChatSessRepo) Create(session *model.ChatSession) error {
	session.ID = 1
	m.lastSession = session
	return nil
}
func (m *mockChatSessRepo) CreateBatch(msgs []model.ChatMessage) error { return nil }
func (m *mockChatSessRepo) FindByID(id int64) (*model.ChatSession, error) {
	if m.lastSession != nil {
		return m.lastSession, nil
	}
	return nil, fmt.Errorf("not found")
}
func (m *mockChatSessRepo) UpdateFeedback(id int64, fb int16) error { return nil }

type mockChatLLM struct {
	response *adapter.ChatResponse
	err      error
}

func (m *mockChatLLM) ChatCompletion(ctx context.Context, req adapter.ChatRequest) (*adapter.ChatResponse, error) {
	return m.response, m.err
}
func (m *mockChatLLM) ChatCompletionStream(ctx context.Context, req adapter.ChatRequest) (<-chan adapter.StreamChunk, error) {
	return nil, fmt.Errorf("not implemented")
}

type mockChatPipe struct {
	result *rag.RAGResult
	err    error
}

func (m *mockChatPipe) Execute(ctx context.Context, query string, kbID int64, opts rag.RAGOptions, onStep rag.StepCallback) (*rag.RAGResult, error) {
	return m.result, m.err
}

// =============================================================================
// 测试用例
// =============================================================================

// TestChatMock_Success 验证 ChatService 使用 Pipeline 检索 + LLM 生成。
func TestChatMock_Success(t *testing.T) {
	kbRepo := &mockChatKBRepo{
		kb: &model.KnowledgeBase{ID: 1, Name: "测试知识库"},
	}
	sessionRepo := &mockChatSessRepo{}
	pipeline := &mockChatPipe{
		result: &rag.RAGResult{
			Chunks: []rag.RetrievalResult{
				{ChunkID: 1, Content: "VPN配置步骤：1.下载客户端 2.输入地址", Score: 0.9, Source: "hybrid"},
			},
			Metrics: rag.PipelineMetrics{TotalDurationMS: 150},
		},
	}
	llm := &mockChatLLM{
		response: &adapter.ChatResponse{Content: "VPN配置方法如下...", FinishReason: "stop"},
	}

	svc := service.NewChatService(kbRepo, sessionRepo, pipeline, llm, nil, 5, "")

	resp, err := svc.CreateChatSession(request.CreateChatRequest{
		Question: "VPN怎么配置",
		KBID:     1,
	}, 1)
	if err != nil {
		t.Fatalf("CreateChatSession 失败: %v", err)
	}
	if resp.Answer == "" {
		t.Error("回答不应为空")
	}
	if resp.Confidence <= 0 {
		t.Error("应返回正置信度")
	}
}

// TestChatMock_RAGFail 验证 RAG 检索失败时的降级。
func TestChatMock_RAGFail(t *testing.T) {
	kbRepo := &mockChatKBRepo{
		kb: &model.KnowledgeBase{ID: 1, Name: "测试"},
	}
	sessionRepo := &mockChatSessRepo{}
	pipeline := &mockChatPipe{
		err: fmt.Errorf("pgvector 不可用"),
	}
	llm := &mockChatLLM{
		response: &adapter.ChatResponse{Content: "回答", FinishReason: "stop"},
	}

	svc := service.NewChatService(kbRepo, sessionRepo, pipeline, llm, nil, 5, "")

	_, err := svc.CreateChatSession(request.CreateChatRequest{
		Question: "test",
		KBID:     1,
	}, 1)
	if err == nil {
		t.Error("RAG 检索失败应返回错误")
	}
}

// TestChatMock_LLMFail 验证 LLM 生成失败时的降级。
func TestChatMock_LLMFail(t *testing.T) {
	kbRepo := &mockChatKBRepo{
		kb: &model.KnowledgeBase{ID: 1, Name: "测试"},
	}
	sessionRepo := &mockChatSessRepo{}
	pipeline := &mockChatPipe{
		result: &rag.RAGResult{
			Chunks: []rag.RetrievalResult{
				{ChunkID: 1, Content: "内容", Score: 0.9, Source: "vector"},
			},
		},
	}
	llm := &mockChatLLM{
		err: fmt.Errorf("LLM 服务不可用"),
	}

	svc := service.NewChatService(kbRepo, sessionRepo, pipeline, llm, nil, 5, "")

	// LLM 失败时降级返回兜底文本（不返回 error）
	resp, err := svc.CreateChatSession(request.CreateChatRequest{
		Question: "test",
		KBID:     1,
	}, 1)
	if err != nil {
		t.Fatalf("LLM 降级不应返回 error: %v", err)
	}
	if resp.Answer == "" {
		t.Error("降级时应返回兜底文本")
	}
}

// TestChatMock_LowConfidence 验证低置信度时引导提交申告。
func TestChatMock_LowConfidence(t *testing.T) {
	kbRepo := &mockChatKBRepo{
		kb: &model.KnowledgeBase{ID: 1, Name: "测试"},
	}
	sessionRepo := &mockChatSessRepo{}
	pipeline := &mockChatPipe{
		result: &rag.RAGResult{
			Chunks:  []rag.RetrievalResult{},
			Metrics: rag.PipelineMetrics{TotalDurationMS: 50},
		},
	}
	llm := &mockChatLLM{
		response: &adapter.ChatResponse{Content: "答案", FinishReason: "stop"},
	}

	svc := service.NewChatService(kbRepo, sessionRepo, pipeline, llm, nil, 5, "")

	resp, err := svc.CreateChatSession(request.CreateChatRequest{
		Question: "test",
		KBID:     1,
	}, 1)
	if err != nil {
		t.Fatalf("不应报错: %v", err)
	}
	if !resp.CanSubmitTicket {
		t.Error("无检索结果时 CanSubmitTicket 应为 true")
	}
}
