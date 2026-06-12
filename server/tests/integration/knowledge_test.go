//go:build integration

// Package integration_test 验证知识库模块的端到端完整生命周期。
//
// 测试覆盖 PLAN.md Task36 定义的场景：
//   - 创建知识库 → 创建知识条目 → 提交审核 → 审核通过 → 发布 → 停用 → 重试同步
//   - 审核驳回流程
//   - 知识库列表和文章列表查询
//
// v2 迁移说明：RagClient（AnythingLLM）已移除，KnowledgeService(v1) 中的
// Publish/Disable/RetrySync 仅管理数据库状态，不再调用外部 RAG 服务。
// 真正的向量同步由 KnowledgeServiceV2（自建 pgvector 管道）负责。
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
	"opsmind/internal/dto/request"
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

// knowledgeIntEnv 封装知识库集成测试环境。
type knowledgeIntEnv struct {
	r  *gin.Engine
	db *gorm.DB
}

// setupKnowledgeIntegration 创建知识库集成测试环境。
func setupKnowledgeIntegration(t *testing.T) *knowledgeIntEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dbCfg := config.DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "opsmind",
		Password: "opsmind123",
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
		rag_document_location VARCHAR(512),
		created_by BIGINT NOT NULL,
		reviewed_by BIGINT,
		published_by BIGINT,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS knowledge_chunks (
		id BIGSERIAL PRIMARY KEY,
		article_id BIGINT NOT NULL,
		content TEXT NOT NULL,
		embedding_model VARCHAR(128) NOT NULL,
		vector_dimension INT NOT NULL,
		sync_status VARCHAR(16) NOT NULL DEFAULT 'pending',
		sync_error TEXT,
		synced_at TIMESTAMPTZ,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`)

	// 清理（按外键依赖顺序）
	db.Exec("DELETE FROM knowledge_chunks")
	db.Exec("DELETE FROM knowledge_articles")
	db.Exec("DELETE FROM knowledge_bases")

	// 组装依赖链（v1：RagClient 已移除，KnowledgeService 管理数据库状态）
	knowledgeRepo := repository.NewKnowledgeRepo(db)
	knowledgeSvc := service.NewKnowledgeService(knowledgeRepo, nil, nil, nil, nil, nil)
	knowledgeH := handler.NewKnowledgeHandler(knowledgeSvc)

	// 路由（模拟管理员用户 user_id=1）
	r := gin.New()
	r.Use(middleware.RequestID())
	r.Use(func(c *gin.Context) {
		c.Set("currentUser", map[string]interface{}{
			"user_id":  float64(1),
			"username": "admin",
			"roles":    []interface{}{"admin"},
		})
		c.Set("userID", int64(1))
		c.Next()
	})

	admin := r.Group("/api/v1/admin")
	{
		// 知识库管理
		admin.GET("/knowledge-bases", knowledgeH.ListKBs)
		admin.POST("/knowledge-bases", knowledgeH.CreateKB)
		admin.PUT("/knowledge-bases/:id", knowledgeH.UpdateKB)
		// 文章管理
		admin.GET("/knowledge-bases/:kb_id/articles", knowledgeH.ListArticles)
		admin.POST("/knowledge-bases/:kb_id/articles", knowledgeH.CreateArticle)
		admin.PUT("/articles/:id", knowledgeH.UpdateArticle)
		admin.GET("/articles/:id", knowledgeH.GetArticleDetail)
		admin.POST("/articles/:id/submit-review", knowledgeH.SubmitReview)
		admin.POST("/articles/:id/review", knowledgeH.Review)
		admin.POST("/articles/:id/publish", knowledgeH.Publish)
		admin.POST("/articles/:id/disable", knowledgeH.Disable)
		admin.POST("/articles/:id/retry-sync", knowledgeH.RetrySync)
	}

	return &knowledgeIntEnv{r: r, db: db}
}

// postJSON 发送 JSON POST 请求并返回响应 body。
func postJSON(t *testing.T, env *knowledgeIntEnv, url string, body interface{}) []byte {
	t.Helper()
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", url, bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.r.ServeHTTP(w, req)
	return w.Body.Bytes()
}

// =============================================================================
// 完整知识生命周期测试
// =============================================================================

// TestKnowledgeIntegration_FullLifecycle 验证完整知识生命周期。
//
// 流程：创建知识库 → 创建草稿 → 提交审核 → 审核通过 → 发布 → 停用 → 重试同步。
// v2 迁移：RagClient 已移除，Publish/Disable/RetrySync 仅管理数据库状态，
// 不再调用外部 RAG 服务。
func TestKnowledgeIntegration_FullLifecycle(t *testing.T) {
	env := setupKnowledgeIntegration(t)

	// 1. 创建知识库（handler 返回 data=nil，从 DB 获取 ID）
	kbBody := postJSON(t, env, "/api/v1/admin/knowledge-bases", request.CreateKBRequest{
		Name:        "集成测试知识库",
		Description: "用于集成测试的知识库",
	})
	var kbResp struct{ Code int }
	require.NoError(t, json.Unmarshal(kbBody, &kbResp))
	assert.Equal(t, 0, kbResp.Code, "创建知识库业务码应为 0")

	var kb model.KnowledgeBase
	env.db.Order("id desc").First(&kb)
	kbID := kb.ID
	assert.NotZero(t, kbID)
	t.Logf("✅ 步骤1: 知识库创建成功, ID=%d", kbID)

	// 2. 创建知识文章（草稿, status=1）— handler 返回 data=nil，从 DB 获取 ID
	articleBody := postJSON(t, env, fmt.Sprintf("/api/v1/admin/knowledge-bases/%d/articles", kbID),
		request.CreateArticleRequest{
			KBID:     kbID,
			Question: "如何重置公司 VPN 密码？",
			Answer:   "请登录 VPN 自助服务平台 https://vpn.company.com，点击「忘记密码」按提示操作。如无法自助重置，请联系 IT 服务台。",
			Category: "网络与VPN",
			Tags:     []string{"VPN", "密码", "自助"},
		})
	var articleResp struct{ Code int }
	require.NoError(t, json.Unmarshal(articleBody, &articleResp))
	assert.Equal(t, 0, articleResp.Code, "创建文章业务码应为 0")

	var article model.KnowledgeArticle
	env.db.Order("id desc").First(&article)
	articleID := article.ID
	assert.NotZero(t, articleID)
	t.Logf("✅ 步骤2: 文章草稿创建成功, ID=%d", articleID)

	// 3. 提交审核 → status: 1→2
	reviewBody := postJSON(t, env, fmt.Sprintf("/api/v1/admin/articles/%d/submit-review", articleID), nil)
	var submitResp struct{ Code int }
	require.NoError(t, json.Unmarshal(reviewBody, &submitResp))
	assert.Equal(t, 0, submitResp.Code, "提交审核业务码应为 0")
	t.Logf("✅ 步骤3: 提交审核成功")

	// 验证状态变为待审核(2)
	env.db.First(&article, articleID)
	assert.Equal(t, int16(2), article.Status, "提交审核后状态应为 2(待审核)")

	// 将文章的 created_by 改为 2，避免"审核人=创建人"被拒绝
	// （mock 中间件注入的 user_id=1 同时用于创建和审核）
	env.db.Model(&article).Update("created_by", int64(2))

	// 4. 审核通过 → status: 2→3
	approveBody := postJSON(t, env, fmt.Sprintf("/api/v1/admin/articles/%d/review", articleID),
		request.ReviewRequest{Approved: true})
	var approveResp struct{ Code int }
	require.NoError(t, json.Unmarshal(approveBody, &approveResp))
	assert.Equal(t, 0, approveResp.Code, "审核通过业务码应为 0")
	t.Logf("✅ 步骤4: 审核通过")

	env.db.First(&article, articleID)
	assert.Equal(t, int16(3), article.Status, "审核通过后状态应为 3(已审核)")

	// 5. 发布 → status: 3→4
	// v2 迁移：RagClient 已移除，发布仅更新数据库状态，不再调用 SyncDocument。
	publishBody := postJSON(t, env, fmt.Sprintf("/api/v1/admin/articles/%d/publish", articleID), nil)
	var publishResp struct{ Code int }
	require.NoError(t, json.Unmarshal(publishBody, &publishResp))
	assert.Equal(t, 0, publishResp.Code, "发布业务码应为 0")
	t.Logf("✅ 步骤5: 发布成功")

	// 验证文章状态为已发布
	env.db.First(&article, articleID)
	assert.Equal(t, int16(4), article.Status, "发布后状态应为 4(已发布)")
	t.Logf("   文章状态=已发布(4)")

	// 6. 停用知识 → status: 4→0
	disableBody := postJSON(t, env, fmt.Sprintf("/api/v1/admin/articles/%d/disable", articleID), nil)
	var disableResp struct{ Code int }
	require.NoError(t, json.Unmarshal(disableBody, &disableResp))
	assert.Equal(t, 0, disableResp.Code, "停用业务码应为 0")
	t.Logf("✅ 步骤6: 停用成功")

	// 验证文章状态为已停用
	env.db.First(&article, articleID)
	assert.Equal(t, int16(0), article.Status, "停用后状态应为 0(已停用)")
	t.Logf("   文章状态=已停用(0)")

	// 7. 重试同步 → 仅重置数据库同步状态（v1 占位，不调用 RAG）
	retryBody := postJSON(t, env, fmt.Sprintf("/api/v1/admin/articles/%d/retry-sync", articleID), nil)
	var retryResp struct{ Code int }
	require.NoError(t, json.Unmarshal(retryBody, &retryResp))
	assert.Equal(t, 0, retryResp.Code, "重试同步业务码应为 0")
	t.Logf("✅ 步骤7: 重试同步成功")

	// 验证文章状态不变
	env.db.First(&article, articleID)
	assert.Equal(t, int16(0), article.Status, "重试同步不应改变文章状态")
}

// =============================================================================
// 审核驳回流程
// =============================================================================

// TestKnowledgeIntegration_ReviewReject 验证审核驳回流程。
func TestKnowledgeIntegration_ReviewReject(t *testing.T) {
	env := setupKnowledgeIntegration(t)

	// 1. 创建知识库（handler 返回 data=nil，从 DB 获取 ID）
	kbBody := postJSON(t, env, "/api/v1/admin/knowledge-bases", request.CreateKBRequest{
		Name: "驳回测试知识库",
	})
	var kbResp struct{ Code int }
	json.Unmarshal(kbBody, &kbResp)
	require.Equal(t, 0, kbResp.Code)

	var kb model.KnowledgeBase
	env.db.Order("id desc").First(&kb)
	kbID := kb.ID

	// 2. 创建草稿（从 DB 获取 ID）
	articleBody := postJSON(t, env, fmt.Sprintf("/api/v1/admin/knowledge-bases/%d/articles", kbID),
		request.CreateArticleRequest{
			KBID:     kbID,
			Question: "待驳回的问题",
			Answer:   "待驳回的答案",
		})
	var articleResp struct{ Code int }
	json.Unmarshal(articleBody, &articleResp)
	require.Equal(t, 0, articleResp.Code)

	var article model.KnowledgeArticle
	env.db.Order("id desc").First(&article)
	articleID := article.ID

	// 3. 提交审核
	postJSON(t, env, fmt.Sprintf("/api/v1/admin/articles/%d/submit-review", articleID), nil)

	// 将文章的 created_by 改为 2，避免审核人=创建人被拒绝
	env.db.Model(&article).Update("created_by", int64(2))

	// 4. 审核驳回 → status: 2→5
	rejectBody := postJSON(t, env, fmt.Sprintf("/api/v1/admin/articles/%d/review", articleID),
		request.ReviewRequest{Approved: false, ReviewComment: "答案不够详细，需要补充操作步骤"})
	var rejectResp struct{ Code int }
	require.NoError(t, json.Unmarshal(rejectBody, &rejectResp))
	assert.Equal(t, 0, rejectResp.Code, "驳回业务码应为 0")

	// 验证状态变为已驳回(5) 且 review_comment 已保存
	env.db.First(&article, articleID)
	assert.Equal(t, int16(5), article.Status, "驳回后状态应为 5(已驳回)")
	assert.Equal(t, "答案不够详细，需要补充操作步骤", article.ReviewComment,
		"应保存驳回意见")
	t.Logf("✅ 审核驳回: status=5, review_comment='%s'", article.ReviewComment)
}

// =============================================================================
// 知识库列表和详情
// =============================================================================

// TestKnowledgeIntegration_ListAndDetail 验证知识库列表和文章列表查询。
func TestKnowledgeIntegration_ListAndDetail(t *testing.T) {
	env := setupKnowledgeIntegration(t)

	// 创建知识库
	postJSON(t, env, "/api/v1/admin/knowledge-bases", request.CreateKBRequest{
		Name: "列表测试知识库",
	})

	// 查询知识库列表
	req := httptest.NewRequest("GET", "/api/v1/admin/knowledge-bases", nil)
	w := httptest.NewRecorder()
	env.r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code, "列表查询应返回 200")
	var listResp struct {
		Code int                       `json:"code"`
		Data []model.KnowledgeBase     `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &listResp))
	assert.Equal(t, 0, listResp.Code)
	assert.GreaterOrEqual(t, len(listResp.Data), 1, "至少应有 1 个知识库")
	t.Logf("✅ 知识库列表: %d 个", len(listResp.Data))

	// 创建文章并查询文章列表
	kbID := listResp.Data[0].ID
	postJSON(t, env, fmt.Sprintf("/api/v1/admin/knowledge-bases/%d/articles", kbID),
		request.CreateArticleRequest{
			KBID:     kbID,
			Question: "列表测试问题",
			Answer:   "列表测试答案",
		})

	req2 := httptest.NewRequest("GET",
		fmt.Sprintf("/api/v1/admin/knowledge-bases/%d/articles?page=1&page_size=10", kbID), nil)
	w2 := httptest.NewRecorder()
	env.r.ServeHTTP(w2, req2)

	assert.Equal(t, 200, w2.Code, "文章列表查询应返回 200")
	// 文章列表使用分页响应格式：total/page/page_size 在顶层，data 为数组
	var articleListResp struct {
		Code     int           `json:"code"`
		Data     []interface{} `json:"data"`
		Total    int64         `json:"total"`
		Page     int           `json:"page"`
		PageSize int           `json:"page_size"`
	}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &articleListResp))
	assert.Equal(t, 0, articleListResp.Code)
	assert.GreaterOrEqual(t, articleListResp.Total, int64(1),
		"至少应有 1 篇文章")
	t.Logf("✅ 文章列表: total=%d", articleListResp.Total)
}
