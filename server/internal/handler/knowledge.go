// Package handler 实现 HTTP 请求处理。
//
// knowledge.go 提供知识库管理相关接口。
// Handler 层职责：参数解析、调用 Service、格式化响应。
// 审核流程和业务规则在 Service 层完成。
package handler

import (
	"strconv"

	"opsmind/internal/dto/request"
	"opsmind/internal/service"
	"opsmind/pkg/errcode"
	"opsmind/pkg/response"

	"github.com/gin-gonic/gin"
)

// KnowledgeHandler 知识库管理接口。
type KnowledgeHandler struct {
	svc *service.KnowledgeService
}

// NewKnowledgeHandler 创建 KnowledgeHandler 实例。
func NewKnowledgeHandler(svc *service.KnowledgeService) *KnowledgeHandler {
	return &KnowledgeHandler{svc: svc}
}

// =============================================================================
// KnowledgeBase
// =============================================================================

// ListKBsForPortal 门户端知识库列表（无需 admin 权限，供 Chat 页选择知识库下拉框使用）。
//
// GET /api/v1/portal/knowledge-bases
func (h *KnowledgeHandler) ListKBsForPortal(c *gin.Context) {
	h.ListKBs(c)
}

// CreateKB 创建知识库。
//
// POST /api/v1/admin/knowledge-bases
func (h *KnowledgeHandler) CreateKB(c *gin.Context) {
	var req request.CreateKBRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.ErrParam, "参数校验失败: "+err.Error())
		return
	}

	userID := getCurrentUserID(c)
	if err := h.svc.CreateKB(req, userID); err != nil {
		handleServiceError(c, err)
		return
	}

	response.Success(c, nil)
}

// UpdateKB 更新知识库。
//
// PUT /api/v1/admin/knowledge-bases/:id
func (h *KnowledgeHandler) UpdateKB(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrParam, "无效的知识库 ID")
		return
	}

	var req request.UpdateKBRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.ErrParam, "参数校验失败: "+err.Error())
		return
	}

	if svcErr := h.svc.UpdateKB(id, req); svcErr != nil {
		handleServiceError(c, svcErr)
		return
	}

	response.Success(c, nil)
}

// ListKBs 列出全部知识库。
//
// GET /api/v1/admin/knowledge-bases
func (h *KnowledgeHandler) ListKBs(c *gin.Context) {
	kbs, err := h.svc.ListKBs()
	if err != nil {
		response.Error(c, errcode.ErrUnknown, err.Error())
		return
	}

	response.Success(c, gin.H{"items": kbs})
}

// =============================================================================
// KnowledgeArticle
// =============================================================================

// CreateArticle 创建知识文章。
//
// POST /api/v1/admin/knowledge-bases/:kb_id/articles
func (h *KnowledgeHandler) CreateArticle(c *gin.Context) {
	kbID, err := strconv.ParseInt(c.Param("kb_id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrParam, "无效的知识库 ID")
		return
	}

	var req request.CreateArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.ErrParam, "参数校验失败: "+err.Error())
		return
	}
	req.KBID = kbID

	userID := getCurrentUserID(c)
	if svcErr := h.svc.CreateArticle(req, userID); svcErr != nil {
		handleServiceError(c, svcErr)
		return
	}

	response.Success(c, nil)
}

// UpdateArticle 更新知识文章。
//
// PUT /api/v1/admin/articles/:id
func (h *KnowledgeHandler) UpdateArticle(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrParam, "无效的文章 ID")
		return
	}

	var req request.UpdateArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.ErrParam, "参数校验失败: "+err.Error())
		return
	}

	userID := getCurrentUserID(c)
	if svcErr := h.svc.UpdateArticle(id, req, userID); svcErr != nil {
		handleServiceError(c, svcErr)
		return
	}

	response.Success(c, nil)
}

// SubmitReview 提交审核。
//
// POST /api/v1/admin/articles/:id/submit-review
func (h *KnowledgeHandler) SubmitReview(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrParam, "无效的文章 ID")
		return
	}

	userID := getCurrentUserID(c)
	if svcErr := h.svc.SubmitReview(id, userID); svcErr != nil {
		handleServiceError(c, svcErr)
		return
	}

	response.Success(c, nil)
}

// Review 审核文章。
//
// POST /api/v1/admin/articles/:id/review
func (h *KnowledgeHandler) Review(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrParam, "无效的文章 ID")
		return
	}

	var req request.ReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.ErrParam, "参数校验失败: "+err.Error())
		return
	}

	userID := getCurrentUserID(c)
	if svcErr := h.svc.Review(id, userID, req); svcErr != nil {
		handleServiceError(c, svcErr)
		return
	}

	response.Success(c, nil)
}

// Publish 发布文章。
//
// POST /api/v1/admin/articles/:id/publish
func (h *KnowledgeHandler) Publish(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrParam, "无效的文章 ID")
		return
	}

	userID := getCurrentUserID(c)
	if svcErr := h.svc.Publish(id, userID); svcErr != nil {
		handleServiceError(c, svcErr)
		return
	}

	response.Success(c, nil)
}

// Disable 停用文章。
//
// POST /api/v1/admin/articles/:id/disable
func (h *KnowledgeHandler) Disable(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrParam, "无效的文章 ID")
		return
	}

	if svcErr := h.svc.Disable(id); svcErr != nil {
		handleServiceError(c, svcErr)
		return
	}

	response.Success(c, nil)
}

// Enable 恢复已停用文章为草稿。
//
// POST /api/v1/admin/articles/:id/enable
func (h *KnowledgeHandler) Enable(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrParam, "无效的文章 ID")
		return
	}

	if svcErr := h.svc.Enable(id); svcErr != nil {
		handleServiceError(c, svcErr)
		return
	}

	response.Success(c, nil)
}

// RetrySync 重试同步。
//
// POST /api/v1/admin/articles/:id/retry-sync
func (h *KnowledgeHandler) RetrySync(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrParam, "无效的文章 ID")
		return
	}

	if svcErr := h.svc.RetrySync(id); svcErr != nil {
		handleServiceError(c, svcErr)
		return
	}

	response.Success(c, nil)
}

// ListArticles 分页查询文章列表。
//
// GET /api/v1/admin/knowledge-bases/:kb_id/articles
func (h *KnowledgeHandler) ListArticles(c *gin.Context) {
	kbID, err := strconv.ParseInt(c.Param("kb_id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrParam, "无效的知识库 ID")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	status, _ := strconv.Atoi(c.DefaultQuery("status", "-1"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	result, svcErr := h.svc.ListArticles(kbID, status, page, pageSize)
	if svcErr != nil {
		handleServiceError(c, svcErr)
		return
	}

	response.SuccessWithPage(c, result.Articles, result.Total, page, pageSize)
}

// GetArticleDetail 获取文章详情。
//
// GET /api/v1/admin/articles/:id
func (h *KnowledgeHandler) GetArticleDetail(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrParam, "无效的文章 ID")
		return
	}

	result, svcErr := h.svc.GetArticleDetail(id)
	if svcErr != nil {
		handleServiceError(c, svcErr)
		return
	}

	response.Success(c, result)
}

// =============================================================================
// 辅助函数
// =============================================================================

// =============================================================================
// EmbeddingConfig
// =============================================================================

// CreateEmbeddingConfig 创建 Embedding 配置。
//
// POST /api/v1/admin/embedding-configs
func (h *KnowledgeHandler) CreateEmbeddingConfig(c *gin.Context) {
	var req request.CreateEmbeddingConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.ErrParam, "参数校验失败: "+err.Error())
		return
	}

	if err := h.svc.CreateEmbeddingConfig(req); err != nil {
		handleServiceError(c, err)
		return
	}

	response.Success(c, nil)
}

// UpdateEmbeddingConfig 更新 Embedding 配置。
//
// PUT /api/v1/admin/embedding-configs/:id
func (h *KnowledgeHandler) UpdateEmbeddingConfig(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrParam, "无效的配置 ID")
		return
	}

	var req request.UpdateEmbeddingConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.ErrParam, "参数校验失败: "+err.Error())
		return
	}

	if svcErr := h.svc.UpdateEmbeddingConfig(id, req); svcErr != nil {
		handleServiceError(c, svcErr)
		return
	}

	response.Success(c, nil)
}

// ListEmbeddingConfigs 列出全部 Embedding 配置。
//
// GET /api/v1/admin/embedding-configs
func (h *KnowledgeHandler) ListEmbeddingConfigs(c *gin.Context) {
	configs, err := h.svc.ListEmbeddingConfigs()
	if err != nil {
		response.Error(c, errcode.ErrUnknown, err.Error())
		return
	}

	response.Success(c, gin.H{"items": configs})
}

// DeleteEmbeddingConfig 删除 Embedding 配置。
//
// DELETE /api/v1/admin/embedding-configs/:id
func (h *KnowledgeHandler) DeleteEmbeddingConfig(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrParam, "无效的配置 ID")
		return
	}

	if err := h.svc.DeleteEmbeddingConfig(id); err != nil {
		handleServiceError(c, err)
		return
	}

	response.Success(c, nil)
}

// getCurrentUserID 从 Gin context 中获取当前用户 ID。
//
// JWTAuth 中间件将当前用户 ID 以 int64 类型写入 context，key 为 "userID"。
// 测试环境中可能不存在，返回 0 作为默认值。
func getCurrentUserID(c *gin.Context) int64 {
	if val, exists := c.Get("userID"); exists {
		if id, ok := val.(int64); ok {
			return id
		}
	}
	return 0
}
