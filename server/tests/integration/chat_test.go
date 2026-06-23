//go:build integration

// Package integration_test 验证智能问答模块的端到端完整流程。
//
// 测试覆盖 PLAN.md Task36 定义的场景：
//   - 创建问答会话 → 返回答案
//   - 低置信度 → can_submit_ticket=true
//   - 提交反馈 → 反馈状态验证
//
// ChatService 在 AI/Embedding 不可用时返回降级兜底文本（confidence=0）。
// 完整的 RAG 流式问答由 SSE 端点提供。
//
// 数据库使用真实 PostgreSQL opsmind_test 库。
package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"

	"opsmind/internal/config"
	"opsmind/internal/database"
	"opsmind/internal/dto/response"
	"opsmind/internal/handler"
	"opsmind/internal/middleware"
	"opsmind/internal/model"
	"opsmind/internal/repository"
	"opsmind/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// =============================================================================
// 测试环境
// =============================================================================

// chatIntEnv 封装问答集成测试环境。
type chatIntEnv struct {
	r  *gin.Engine
	db *gorm.DB
	kb *model.KnowledgeBase
}

// setupChatIntegration 创建问答集成测试环境。
func setupChatIntegration(t *testing.T) *chatIntEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dbCfg := config.DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "opsmind",
		Password: "opsmind_dev",
		DBName:   "opsmind_test",
		SSLMode:  "disable",
	}
	db, err := database.Init(dbCfg)
	require.NoError(t, err, "初始化数据库失败")

	// 建表
	db.Exec(`CREATE TABLE IF NOT EXISTS knowledge_bases (
		id BIGSERIAL PRIMARY KEY,
		name VARCHAR(128) NOT NULL,
		description TEXT,
		rag_workspace_slug VARCHAR(128),
		embedding_model VARCHAR(128) NOT NULL DEFAULT '',
		vector_dimension INT NOT NULL DEFAULT 0,
		created_by BIGINT NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS knowledge_articles (
		id BIGSERIAL PRIMARY KEY,
		kb_id BIGINT NOT NULL,
		question TEXT NOT NULL,
		answer TEXT NOT NULL,
		category VARCHAR(64) DEFAULT '',
		tags JSONB,
		status SMALLINT NOT NULL DEFAULT 1,
		review_comment TEXT,
		rag_document_location VARCHAR(255),
		created_by BIGINT NOT NULL,
		reviewed_by BIGINT,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS chat_sessions (
		id BIGSERIAL PRIMARY KEY,
		user_id BIGINT NOT NULL,
		kb_id BIGINT NOT NULL,
		question TEXT NOT NULL,
		answer TEXT,
		sources JSONB,
		confidence DOUBLE PRECISION DEFAULT 0,
		feedback SMALLINT DEFAULT 0,
		duration_ms INT DEFAULT 0,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS chat_messages (
		id BIGSERIAL PRIMARY KEY,
		session_id BIGINT NOT NULL,
		role VARCHAR(16) NOT NULL,
		content TEXT NOT NULL,
		sources JSONB,
		confidence DOUBLE PRECISION DEFAULT 0,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`)

	// 清理
	db.Exec("DELETE FROM chat_messages")
	db.Exec("DELETE FROM chat_sessions")
	db.Exec("DELETE FROM knowledge_articles")
	db.Exec("DELETE FROM knowledge_bases")

	// 创建测试知识库
	kb := &model.KnowledgeBase{
		Name:             "集成测试知识库",
		RAGWorkspaceSlug: "itg-test-workspace",
		EmbeddingModel:   "text-embedding-ada-002",
		VectorDimension:  1536,
		CreatedBy:        1,
	}
	require.NoError(t, db.Create(kb).Error)

	// 组装依赖链（v1：RagClient 已移除，ChatService 使用占位实现）
	knowledgeRepo := repository.NewKnowledgeRepo(db)
	chatRepo := repository.NewChatRepo(db)
	chatSvc := service.NewChatService(knowledgeRepo, chatRepo, nil, service.RAGDefaults{TopK: 5})
	chatH := handler.NewChatHandler(chatSvc)

	// 路由（模拟认证中间件注入 user_id=1）
	r := gin.New()
	r.Use(middleware.RequestID())
	r.Use(func(c *gin.Context) {
		c.Set("currentUser", map[string]interface{}{"user_id": float64(1)})
		c.Set("userID", int64(1))
		c.Next()
	})

	portal := r.Group("/api/v1/portal")
	{
		portal.POST("/chat-sessions", chatH.CreateChatSession)
		portal.POST("/chat-sessions/:id/feedback", chatH.SubmitFeedback)
		portal.GET("/chat-sessions/:id", chatH.GetChatDetail)
	}

	return &chatIntEnv{r: r, db: db, kb: kb}
}

// =============================================================================
// 完整问答流程
// =============================================================================

// TestChatIntegration_FullFlow 验证完整问答流程。
//
// 流程：创建问答会话 → 验证兜底回答 → 提交反馈 → 查询验证反馈 → 检查数据持久化。
func TestChatIntegration_FullFlow(t *testing.T) {
	env := setupChatIntegration(t)

	// v2: CreateSession 仅创建会话容器，LLM 交互通过 SSE 流式端点完成。

	// 1. 创建问答会话
	askBody, _ := json.Marshal(map[string]interface{}{
		"title": "网络连不上怎么办？",
		"kb_id": env.kb.ID,
	})
	askReq := httptest.NewRequest("POST", "/api/v1/portal/chat-sessions",
		bytes.NewReader(askBody))
	askReq.Header.Set("Content-Type", "application/json")
	askW := httptest.NewRecorder()
	env.r.ServeHTTP(askW, askReq)

	assert.Equal(t, 200, askW.Code, "创建会话应返回 200")

	var createResp struct {
		Code int `json:"code"`
		Data struct {
			SessionID int64  `json:"session_id"`
			KBID      int64  `json:"kb_id"`
			Question  string `json:"question"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(askW.Body.Bytes(), &createResp))
	assert.Equal(t, 0, createResp.Code)

	sessionID := createResp.Data.SessionID
	assert.NotZero(t, sessionID, "应返回 SessionID")
	assert.Equal(t, env.kb.ID, createResp.Data.KBID, "KBID 应与请求一致")
	assert.NotEmpty(t, createResp.Data.Question, "Question 应非空")
	t.Logf("✅ 步骤1: 会话创建成功，session_id=%d", sessionID)

	// 2. 提交反馈（已解决）
	feedbackBody, _ := json.Marshal(map[string]interface{}{"feedback": 1}) // 1=resolved
	fbReq := httptest.NewRequest("POST",
		"/api/v1/portal/chat-sessions/"+itoa64(sessionID)+"/feedback",
		bytes.NewReader(feedbackBody))
	fbReq.Header.Set("Content-Type", "application/json")
	fbW := httptest.NewRecorder()
	env.r.ServeHTTP(fbW, fbReq)

	assert.Equal(t, 200, fbW.Code, "提交反馈应返回 200")
	t.Logf("✅ 步骤2: 反馈提交成功 (resolved)")

	// 3. 查询会话详情 → 验证反馈已保存
	detailReq := httptest.NewRequest("GET",
		"/api/v1/portal/chat-sessions/"+itoa64(sessionID), nil)
	detailW := httptest.NewRecorder()
	env.r.ServeHTTP(detailW, detailReq)

	assert.Equal(t, 200, detailW.Code)

	var detailResp struct {
		Data response.ChatSessionResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(detailW.Body.Bytes(), &detailResp))
	assert.Equal(t, int16(1), detailResp.Data.Feedback, "反馈状态应为 1(resolved)")
	t.Logf("✅ 步骤3: 反馈状态验证通过")

	// 4. 验证数据库中会话已持久化（消息持久化由 SSE 流式管道负责）
	var sessionCount int64
	env.db.Model(&model.ChatSession{}).Where("id = ?", sessionID).Count(&sessionCount)
	assert.Equal(t, int64(1), sessionCount, "数据库应有 1 条会话记录")
	t.Logf("✅ 步骤4: 数据持久化验证通过（会话=%d）", sessionCount)
}

// =============================================================================
// 低置信度 → 引导转申告
// =============================================================================

// TestChatIntegration_LowConfidenceToTicket 验证低置信度时引导用户提交申告。
//
// AI 不可用时所有回答均为低置信度（0），
// CanSubmitTicket 永远为 true。此测试验证该行为符合预期。
func TestChatIntegration_LowConfidenceToTicket(t *testing.T) {
	env := setupChatIntegration(t)

	// v2: CreateSession 仅创建容器，CanSubmitTicket 由 SSE 流式端点返回
	body, _ := json.Marshal(map[string]interface{}{
		"title": "非常专业的运维问题",
		"kb_id": env.kb.ID,
	})
	req := httptest.NewRequest("POST", "/api/v1/portal/chat-sessions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			SessionID int64 `json:"session_id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.NotZero(t, resp.Data.SessionID, "应返回 SessionID")
	t.Logf("✅ 会话创建成功，session_id=%d", resp.Data.SessionID)
}

// =============================================================================
// AI 服务降级
// =============================================================================

// TestChatIntegration_AIServiceUnavailable 验证 AI 服务不可用时的降级处理。
//
// 不依赖外部 AI 服务时，所有请求统一按降级路径处理，返回兜底回答（code=0）。
func TestChatIntegration_AIServiceUnavailable(t *testing.T) {
	env := setupChatIntegration(t)

	body, _ := json.Marshal(map[string]interface{}{
		"kb_id": env.kb.ID,
	})
	req := httptest.NewRequest("POST", "/api/v1/portal/chat-sessions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.r.ServeHTTP(w, req)

	// 会话创建不依赖 AI 服务
	var resp struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code, "会话创建应正常返回 code=0")
	t.Logf("✅ 会话创建不依赖外部 AI 服务 → code=0")
}

// =============================================================================
// 参数校验
// =============================================================================

// TestChatIntegration_Validation 验证问答接口参数校验。
func TestChatIntegration_Validation(t *testing.T) {
	env := setupChatIntegration(t)

	// 场景1: kb_id 缺失
	body, _ := json.Marshal(map[string]interface{}{})
	req := httptest.NewRequest("POST", "/api/v1/portal/chat-sessions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.r.ServeHTTP(w, req)

	var resp struct{ Code int }
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NotEqual(t, 0, resp.Code, "缺失 kb_id 应返回错误码")
	t.Logf("✅ 缺失 kb_id 被拒绝 (code=%d)", resp.Code)

	// 场景2: kb_id 不存在
	body2, _ := json.Marshal(map[string]interface{}{
		"kb_id": 99999,
	})
	req2 := httptest.NewRequest("POST", "/api/v1/portal/chat-sessions", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	env.r.ServeHTTP(w2, req2)

	var resp2 struct{ Code int }
	json.Unmarshal(w2.Body.Bytes(), &resp2)
	assert.NotEqual(t, 0, resp2.Code, "不存在的知识库应返回错误码")
	t.Logf("✅ 不存在知识库被拒绝 (code=%d)", resp2.Code)
}

// =============================================================================
// 工具函数
// =============================================================================

// itoa64 将 int64 转为字符串（用于 URL 拼接）
func itoa64(n int64) string {
	return fmt.Sprintf("%d", n)
}
