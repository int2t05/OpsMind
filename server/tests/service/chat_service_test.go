//go:build integration

// Package service_test 验证 ChatService 业务逻辑。
//
// 测试覆盖 PLAN.md Task26 定义的全部方法：
// CreateChatSession / SubmitFeedback / GetChatDetail
// 覆盖场景：正常问答、低置信度兜底、AI 服务错误降级、不可达降级、参数校验。
package service_test

import (
	"context"
	"errors"
	"testing"

	"opsmind/internal/adapter"
	"opsmind/internal/config"
	"opsmind/internal/database"
	"opsmind/internal/dto/request"
	"opsmind/internal/model"
	"opsmind/internal/repository"
	"opsmind/internal/service"

	"gorm.io/gorm"
)

// =============================================================================
// Mock RagClient（chat 专用）
// =============================================================================

// mockChatRagClient 可定制 Query 行为的 RagClient mock。
type mockChatRagClient struct {
	queryFunc func(ctx context.Context, req adapter.RAGQueryRequest) (*adapter.RAGQueryResponse, error)
}

func (m *mockChatRagClient) Query(ctx context.Context, req adapter.RAGQueryRequest) (*adapter.RAGQueryResponse, error) {
	if m.queryFunc != nil {
		return m.queryFunc(ctx, req)
	}
	return &adapter.RAGQueryResponse{Answer: "默认回答", Confidence: 0.9}, nil
}

func (m *mockChatRagClient) CreateWorkspace(ctx context.Context, req adapter.RAGCreateWorkspaceRequest) (*adapter.RAGCreateWorkspaceResponse, error) {
	return &adapter.RAGCreateWorkspaceResponse{Slug: "mock-ws", ID: 1}, nil
}

func (m *mockChatRagClient) SyncDocument(ctx context.Context, req adapter.RAGSyncRequest) (*adapter.RAGSyncResponse, error) {
	return &adapter.RAGSyncResponse{DocumentLocation: "mock-loc"}, nil
}

func (m *mockChatRagClient) DisableDocument(ctx context.Context, req adapter.RAGDisableRequest) error {
	return nil
}

// =============================================================================
// 测试基础设施
// =============================================================================

var chatSvcDB *gorm.DB

func init() {
	cfg := config.DatabaseConfig{
		Host: "localhost", Port: 5432, User: "opsmind", Password: "opsmind123",
		DBName: "opsmind_test", SSLMode: "disable",
	}
	db, err := database.Init(cfg)
	if err != nil {
		panic(err)
	}
	chatSvcDB = db
}

func setupChatServiceTest(t *testing.T) (*service.ChatService, *mockChatRagClient, *model.KnowledgeBase) {
	t.Helper()

	// 创建表
	chatSvcDB.Exec(`CREATE TABLE IF NOT EXISTS users (
		id BIGSERIAL PRIMARY KEY, username VARCHAR(64) NOT NULL UNIQUE,
		password_hash VARCHAR(255) NOT NULL, real_name VARCHAR(64) NOT NULL,
		phone VARCHAR(11) NOT NULL, email VARCHAR(128),
		status SMALLINT NOT NULL DEFAULT 1, first_login BOOLEAN NOT NULL DEFAULT TRUE,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`)
	chatSvcDB.Exec(`CREATE TABLE IF NOT EXISTS knowledge_bases (
		id BIGSERIAL PRIMARY KEY, name VARCHAR(128) NOT NULL, description TEXT,
		rag_workspace_slug VARCHAR(128), embedding_model VARCHAR(128) NOT NULL DEFAULT '',
		vector_dimension INT NOT NULL DEFAULT 0, created_by BIGINT NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`)
	chatSvcDB.Exec(`CREATE TABLE IF NOT EXISTS chat_sessions (
		id BIGSERIAL PRIMARY KEY, user_id BIGINT NOT NULL, kb_id BIGINT NOT NULL,
		question TEXT NOT NULL, answer TEXT, sources JSONB,
		confidence DOUBLE PRECISION DEFAULT 0, feedback SMALLINT DEFAULT 0,
		duration_ms INT DEFAULT 0, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`)
	chatSvcDB.Exec(`CREATE TABLE IF NOT EXISTS chat_messages (
		id BIGSERIAL PRIMARY KEY, session_id BIGINT NOT NULL,
		role VARCHAR(16) NOT NULL, content TEXT NOT NULL, sources JSONB,
		confidence DOUBLE PRECISION DEFAULT 0, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`)

	// 清理旧数据
	chatSvcDB.Exec("DELETE FROM chat_messages")
	chatSvcDB.Exec("DELETE FROM chat_sessions")
	chatSvcDB.Exec("DELETE FROM knowledge_bases")

	knowledgeRepo := repository.NewKnowledgeRepo(chatSvcDB)
	chatRepo := repository.NewChatRepo(chatSvcDB)
	mockRAG := &mockChatRagClient{}
	svc := service.NewChatService(knowledgeRepo, chatRepo, mockRAG)

	// 创建测试知识库（含 workspace slug）
	kb := &model.KnowledgeBase{
		Name:             "运维知识库",
		Description:      "测试",
		RAGWorkspaceSlug: "ops-workspace",
		EmbeddingModel:   "text-embedding-ada-002",
		VectorDimension:  1536,
		CreatedBy:        1,
	}
	if err := chatSvcDB.Create(kb).Error; err != nil {
		t.Fatalf("创建测试知识库失败: %v", err)
	}

	return svc, mockRAG, kb
}

// =============================================================================
// CreateChatSession
// =============================================================================

func TestChatService_CreateChatSession_Success(t *testing.T) {
	svc, mockRAG, kb := setupChatServiceTest(t)

	// 设置 mock 返回高置信度答案
	mockRAG.queryFunc = func(ctx context.Context, req adapter.RAGQueryRequest) (*adapter.RAGQueryResponse, error) {
		if req.WorkspaceSlug != "ops-workspace" {
			t.Errorf("期望 workspace='ops-workspace', got '%s'", req.WorkspaceSlug)
		}
		if req.Question != "如何重置密码？" {
			t.Errorf("期望 question='如何重置密码？', got '%s'", req.Question)
		}
		return &adapter.RAGQueryResponse{
			Answer:     "请前往设置页面修改密码",
			Confidence: 0.85,
			Sources: []adapter.RAGSource{
				{DocName: "账号管理", ChunkContent: "修改密码...", Confidence: 0.85},
			},
		}, nil
	}

	req := request.CreateChatRequest{
		Question: "如何重置密码？",
		KBID:     kb.ID,
	}

	resp, err := svc.CreateChatSession(req, 1)
	if err != nil {
		t.Fatalf("期望无错误, got %v", err)
	}
	if resp.Answer != "请前往设置页面修改密码" {
		t.Errorf("期望 Answer, got '%s'", resp.Answer)
	}
	if resp.Confidence != 0.85 {
		t.Errorf("期望 Confidence=0.85, got %f", resp.Confidence)
	}
	if len(resp.Sources) != 1 {
		t.Errorf("期望 1 个来源, got %d", len(resp.Sources))
	}
	if resp.CanSubmitTicket {
		t.Error("高置信度时 CanSubmitTicket 应为 false")
	}
	if resp.SessionID == 0 {
		t.Error("应填充 SessionID")
	}
}

func TestChatService_CreateChatSession_LowConfidence(t *testing.T) {
	svc, mockRAG, kb := setupChatServiceTest(t)

	mockRAG.queryFunc = func(ctx context.Context, req adapter.RAGQueryRequest) (*adapter.RAGQueryResponse, error) {
		return &adapter.RAGQueryResponse{
			Answer:     "不太确定...",
			Confidence: 0.3,
			Sources:    []adapter.RAGSource{},
		}, nil
	}

	req := request.CreateChatRequest{Question: "复杂问题", KBID: kb.ID}
	resp, err := svc.CreateChatSession(req, 1)
	if err != nil {
		t.Fatalf("期望无错误, got %v", err)
	}
	if !resp.CanSubmitTicket {
		t.Error("低置信度时 CanSubmitTicket 应为 true")
	}
	if len(resp.Answer) == 0 {
		t.Error("低置信度时也应返回兜底答案")
	}
}

func TestChatService_CreateChatSession_RAGError(t *testing.T) {
	svc, mockRAG, kb := setupChatServiceTest(t)

	mockRAG.queryFunc = func(ctx context.Context, req adapter.RAGQueryRequest) (*adapter.RAGQueryResponse, error) {
		return &adapter.RAGQueryResponse{
			Error: "workspace not found",
		}, nil
	}

	req := request.CreateChatRequest{Question: "问题", KBID: kb.ID}
	resp, err := svc.CreateChatSession(req, 1)
	if err != nil {
		t.Fatalf("期望无错误（降级处理）, got %v", err)
	}
	if !resp.CanSubmitTicket {
		t.Error("AI 服务返回错误时 CanSubmitTicket 应为 true")
	}
}

func TestChatService_CreateChatSession_Unreachable(t *testing.T) {
	svc, mockRAG, kb := setupChatServiceTest(t)

	mockRAG.queryFunc = func(ctx context.Context, req adapter.RAGQueryRequest) (*adapter.RAGQueryResponse, error) {
		return nil, errors.New("connection refused")
	}

	req := request.CreateChatRequest{Question: "问题", KBID: kb.ID}
	_, err := svc.CreateChatSession(req, 1)
	if err == nil {
		t.Fatal("期望错误, got nil")
	}
	// 应返回 AppError 且 code=20001
	appErr, ok := err.(service.AppError)
	if !ok {
		t.Fatalf("期望 AppError, got %T", err)
	}
	if appErr.Code != 20001 {
		t.Errorf("期望 code=20001 (AI 服务不可用), got %d", appErr.Code)
	}
}

func TestChatService_CreateChatSession_InvalidKB(t *testing.T) {
	svc, _, _ := setupChatServiceTest(t)

	req := request.CreateChatRequest{Question: "问题", KBID: 999999}
	_, err := svc.CreateChatSession(req, 1)
	if err == nil {
		t.Fatal("期望错误, got nil")
	}
}

func TestChatService_CreateChatSession_EmptyQuestion(t *testing.T) {
	svc, _, kb := setupChatServiceTest(t)

	req := request.CreateChatRequest{Question: "", KBID: kb.ID}
	_, err := svc.CreateChatSession(req, 1)
	if err == nil {
		t.Fatal("期望错误, got nil")
	}
}

// =============================================================================
// SubmitFeedback
// =============================================================================

func TestChatService_SubmitFeedback(t *testing.T) {
	svc, mockRAG, kb := setupChatServiceTest(t)

	// 先创建会话
	mockRAG.queryFunc = func(ctx context.Context, req adapter.RAGQueryRequest) (*adapter.RAGQueryResponse, error) {
		return &adapter.RAGQueryResponse{Answer: "答案", Confidence: 0.9}, nil
	}
	resp, err := svc.CreateChatSession(request.CreateChatRequest{Question: "问题", KBID: kb.ID}, 1)
	if err != nil {
		t.Fatalf("创建会话失败: %v", err)
	}

	// 提交反馈
	err = svc.SubmitFeedback(resp.SessionID, 1) // 1 = 已解决
	if err != nil {
		t.Fatalf("期望无错误, got %v", err)
	}

	// 验证反馈已更新
	detail, err := svc.GetChatDetail(resp.SessionID)
	if err != nil {
		t.Fatalf("查询详情失败: %v", err)
	}
	if detail.Feedback != 1 {
		t.Errorf("期望 Feedback=1, got %d", detail.Feedback)
	}
}

// =============================================================================
// GetChatDetail
// =============================================================================

func TestChatService_GetChatDetail_NotFound(t *testing.T) {
	svc, _, _ := setupChatServiceTest(t)

	_, err := svc.GetChatDetail(999999)
	if err == nil {
		t.Fatal("期望错误, got nil")
	}
}
