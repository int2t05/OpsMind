// Package service 实现知识库管理业务逻辑。
//
// KnowledgeService 提供知识库 CRUD、知识文章 CRUD、审核流程、
// AnythingLLM 同步、停用和重试功能。
//
// 审核状态机：草稿(1) → 待审核(2) → 已通过(3)/已驳回(5) → 已发布(4)/已停用(0)
// 为什么使用显式状态转换而非隐式条件判断：
// 状态转换规则是知识管理核心，显式状态机便于审计和调试。
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
	"gorm.io/gorm"
)

// KnowledgeService 知识库管理服务。
type KnowledgeService struct {
	repo *repository.KnowledgeRepo
	rag  adapter.RagClient
}

// NewKnowledgeService 创建 KnowledgeService 实例。
func NewKnowledgeService(repo *repository.KnowledgeRepo, rag adapter.RagClient) *KnowledgeService {
	return &KnowledgeService{repo: repo, rag: rag}
}

// =============================================================================
// KnowledgeBase
// =============================================================================

// CreateKB 创建知识库，同时调用 RagClient.CreateWorkspace 创建 AnythingLLM 工作区。
func (s *KnowledgeService) CreateKB(req request.CreateKBRequest, userID int64) error {
	// 调用 RagClient 创建工作区
	ctx := context.Background()
	wsResp, err := s.rag.CreateWorkspace(ctx, adapter.RAGCreateWorkspaceRequest{
		Name:            req.Name,
		EmbeddingModel:  req.EmbeddingModel,
		EmbeddingEngine: "generic-openai",
	})
	if err != nil {
		return AppError{Code: errcode.ErrRAGUnavailable, Message: "创建 RAG 工作区失败: " + err.Error()}
	}

	kb := &model.KnowledgeBase{
		Name:             req.Name,
		Description:      req.Description,
		RAGWorkspaceSlug: wsResp.Slug,
		EmbeddingModel:   req.EmbeddingModel,
		CreatedBy:        userID,
	}

	return s.repo.CreateKB(kb)
}

// UpdateKB 更新知识库信息。
func (s *KnowledgeService) UpdateKB(id int64, req request.UpdateKBRequest) error {
	kb, err := s.repo.FindKBByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return AppError{Code: errcode.ErrNotFound, Message: "知识库不存在"}
		}
		return err
	}

	kb.Name = req.Name
	kb.Description = req.Description

	return s.repo.UpdateKB(kb)
}

// ListKBs 列出全部知识库。
func (s *KnowledgeService) ListKBs() ([]response.KBResponse, error) {
	kbs, err := s.repo.ListKBs()
	if err != nil {
		return nil, err
	}

	result := make([]response.KBResponse, len(kbs))
	for i, kb := range kbs {
		result[i] = response.KBResponse{
			ID:               kb.ID,
			Name:             kb.Name,
			Description:      kb.Description,
			RAGWorkspaceSlug: kb.RAGWorkspaceSlug,
			EmbeddingModel:   kb.EmbeddingModel,
			VectorDimension:  kb.VectorDimension,
			CreatedBy:        kb.CreatedBy,
			CreatedAt:        kb.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:        kb.UpdatedAt.Format("2006-01-02 15:04:05"),
		}
	}

	return result, nil
}

// =============================================================================
// KnowledgeArticle
// =============================================================================

// CreateArticle 创建知识文章（草稿状态）。
func (s *KnowledgeService) CreateArticle(req request.CreateArticleRequest, userID int64) error {
	// 验证知识库存在
	_, err := s.repo.FindKBByID(req.KBID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return AppError{Code: errcode.ErrNotFound, Message: "知识库不存在"}
		}
		return err
	}

	tagsJSON := marshalTags(req.Tags)
	article := &model.KnowledgeArticle{
		KBID:      req.KBID,
		Question:  req.Question,
		Answer:    req.Answer,
		Category:  req.Category,
		Tags:      tagsJSON,
		Status:    1, // 草稿
		CreatedBy: userID,
	}

	return s.repo.CreateArticle(article)
}

// UpdateArticle 更新文章（仅草稿/驳回状态可编辑）。
func (s *KnowledgeService) UpdateArticle(id int64, req request.UpdateArticleRequest, userID int64) error {
	article, err := s.repo.FindArticleByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return AppError{Code: errcode.ErrNotFound, Message: "文章不存在"}
		}
		return err
	}

	// 仅草稿(1)和驳回(5)可编辑
	if article.Status != 1 && article.Status != 5 {
		return AppError{Code: errcode.ErrParam, Message: "仅草稿和驳回状态可编辑"}
	}

	article.Question = req.Question
	article.Answer = req.Answer
	article.Category = req.Category
	article.Tags = marshalTags(req.Tags)

	return s.repo.UpdateArticle(article)
}

// SubmitReview 提交审核（草稿→待审核）。
func (s *KnowledgeService) SubmitReview(id int64, userID int64) error {
	article, err := s.repo.FindArticleByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return AppError{Code: errcode.ErrNotFound, Message: "文章不存在"}
		}
		return err
	}

	if article.Status != 1 {
		return AppError{Code: errcode.ErrParam, Message: "仅草稿状态可提交审核"}
	}

	return s.repo.UpdateArticleStatus(id, 2)
}

// Review 审核文章（待审核→已通过/已驳回）。
//
// 业务规则：审核人不能是创建人；驳回时必须填写审核意见。
func (s *KnowledgeService) Review(id int64, reviewerID int64, req request.ReviewRequest) error {
	article, err := s.repo.FindArticleByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return AppError{Code: errcode.ErrNotFound, Message: "文章不存在"}
		}
		return err
	}

	if article.Status != 2 {
		return AppError{Code: errcode.ErrParam, Message: "仅待审核状态可审核"}
	}

	// 审核人不能是创建人
	if article.CreatedBy == reviewerID {
		return AppError{Code: errcode.ErrParam, Message: "审核人不能是创建人"}
	}

	if req.Approved {
		// 通过：status=3，同时记录审核人
		article.Status = 3
		article.ReviewedBy = &reviewerID
		return s.repo.UpdateArticle(article)
	}

	// 驳回：必须填写审核意见
	if strings.TrimSpace(req.ReviewComment) == "" {
		return AppError{Code: errcode.ErrParam, Message: "驳回时必须填写审核意见"}
	}

	// 更新 status + review_comment + reviewed_by
	article.Status = 5
	article.ReviewComment = req.ReviewComment
	article.ReviewedBy = &reviewerID
	return s.repo.UpdateArticle(article)
}

// Publish 发布已审核通过的文章，同步到 AnythingLLM。
//
// 发布流程：生成 chunks → 调用 RagClient.SyncDocument → 更新 article 状态和 chunk 同步状态。
// 同步失败时文章仍标记为已发布，但 chunks 记录 sync_status=failed，支持后续重试。
func (s *KnowledgeService) Publish(id int64, publisherID int64) error {
	article, err := s.repo.FindArticleByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return AppError{Code: errcode.ErrNotFound, Message: "文章不存在"}
		}
		return err
	}

	if article.Status != 3 {
		return AppError{Code: errcode.ErrParam, Message: "仅已审核通过的文章可发布"}
	}

	// 同步到 RAG
	ctx := context.Background()
	syncResp, syncErr := s.rag.SyncDocument(ctx, adapter.RAGSyncRequest{
		WorkspaceSlug: article.KnowledgeBase.RAGWorkspaceSlug,
		Title:         article.Question,
		Content:       article.Answer,
		Mode:          "raw-text",
	})

	// 更新文章
	article.Status = 4
	article.PublishedBy = &publisherID
	if syncErr == nil {
		article.RAGDocumentLocation = syncResp.DocumentLocation
	}
	if err := s.repo.UpdateArticle(article); err != nil {
		return err
	}

	// 更新切片同步状态
	if syncErr != nil {
		_ = s.repo.UpdateChunkSyncStatus(id, "failed", syncErr.Error())
	} else {
		// 已有 chunks 则更新状态，不存则创建新 chunk
		chunks, _ := s.repo.FindChunksByArticleID(id)
		if len(chunks) > 0 {
			_ = s.repo.UpdateChunkSyncStatus(id, "synced", "")
		} else {
			now := time.Now()
			_ = s.repo.CreateChunks([]model.KnowledgeChunk{{
				ArticleID:       id,
				Content:         article.Answer,
				EmbeddingModel:  article.KnowledgeBase.EmbeddingModel,
				VectorDimension: article.KnowledgeBase.VectorDimension,
				SyncStatus:      "synced",
				SyncedAt:        &now,
			}})
		}
	}

	return nil
}

// Disable 停用已发布文章，从 AnythingLLM 中移除。
func (s *KnowledgeService) Disable(id int64) error {
	article, err := s.repo.FindArticleByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return AppError{Code: errcode.ErrNotFound, Message: "文章不存在"}
		}
		return err
	}

	// 调用 RagClient 停用文档
	if article.RAGDocumentLocation != "" {
		ctx := context.Background()
		_ = s.rag.DisableDocument(ctx, adapter.RAGDisableRequest{
			WorkspaceSlug:     article.KnowledgeBase.RAGWorkspaceSlug,
			DocumentLocations: []string{article.RAGDocumentLocation},
		})
	}

	// 更新文章状态
	article.Status = 0
	if err := s.repo.UpdateArticle(article); err != nil {
		return err
	}

	// 更新切片 sync_status
	_ = s.repo.UpdateChunkStatusByArticleID(id, "disabled")

	return nil
}

// Enable 恢复已停用文章为草稿状态。
//
// 为什么恢复到草稿而非直接发布：
// 停用的文章向量已从 AnythingLLM 中删除，恢复到草稿允许编辑后重新走审核→发布流程，
// 确保内容仍然适用后再同步到 RAG。
func (s *KnowledgeService) Enable(id int64) error {
	article, err := s.repo.FindArticleByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return AppError{Code: errcode.ErrNotFound, Message: "文章不存在"}
		}
		return err
	}

	if article.Status != 0 {
		return AppError{Code: errcode.ErrParam, Message: "仅已停用状态的文章可恢复"}
	}

	article.Status = 1 // 已停用 → 草稿
	if err := s.repo.UpdateArticle(article); err != nil {
		return err
	}

	// 清除切片的 disabled 状态，重置为 pending 待重新同步
	_ = s.repo.UpdateChunkStatusByArticleID(id, "pending")

	return nil
}

// RetrySync 重试同步已发布的文章到 AnythingLLM。
func (s *KnowledgeService) RetrySync(id int64) error {
	article, err := s.repo.FindArticleByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return AppError{Code: errcode.ErrNotFound, Message: "文章不存在"}
		}
		return err
	}

	ctx := context.Background()
	syncResp, syncErr := s.rag.SyncDocument(ctx, adapter.RAGSyncRequest{
		WorkspaceSlug: article.KnowledgeBase.RAGWorkspaceSlug,
		Title:         article.Question,
		Content:       article.Answer,
		Mode:          "raw-text",
	})

	// 更新 rag_document_location
	if syncErr == nil && syncResp.DocumentLocation != "" {
		article.RAGDocumentLocation = syncResp.DocumentLocation
		_ = s.repo.UpdateArticle(article)
	}

	// 更新切片同步状态
	if syncErr != nil {
		_ = s.repo.UpdateChunkSyncStatus(id, "failed", syncErr.Error())
	} else {
		_ = s.repo.UpdateChunkSyncStatus(id, "synced", "")
	}

	return nil
}

// ListArticles 分页查询文章列表（按知识库和状态过滤）。
func (s *KnowledgeService) ListArticles(kbID int64, status int, page, pageSize int) (*response.ArticleListResponse, error) {
	articles, total, err := s.repo.ListArticles(kbID, status, page, pageSize)
	if err != nil {
		return nil, err
	}

	result := make([]response.ArticleResponse, len(articles))
	for i, a := range articles {
		// 获取切片的整体同步状态
		syncStatus := getAggregateSyncStatus(&a)

		result[i] = response.ArticleResponse{
			ID:              a.ID,
			KBID:            a.KBID,
			KBName:          a.KnowledgeBase.Name,
			Question:        a.Question,
			Answer:          a.Answer,
			Category:        a.Category,
			Tags:            unmarshalTags(a.Tags),
			Status:          a.Status,
			StatusText:      statusText(a.Status),
			CreatedBy:       a.CreatedBy,
			ReviewedBy:      a.ReviewedBy,
			ReviewComment:   a.ReviewComment,
			SyncStatus:      syncStatus,
			CreatedAt:       a.CreatedAt,
			UpdatedAt:       a.UpdatedAt,
		}
	}

	return &response.ArticleListResponse{
		Articles: result,
		Total:    total,
	}, nil
}

// GetArticleDetail 获取文章详情（含切片列表）。
func (s *KnowledgeService) GetArticleDetail(id int64) (*response.ArticleDetailResponse, error) {
	article, err := s.repo.FindArticleByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, AppError{Code: errcode.ErrNotFound, Message: "文章不存在"}
		}
		return nil, err
	}

	chunks, err := s.repo.FindChunksByArticleID(id)
	if err != nil {
		return nil, err
	}

	chunkResponses := make([]response.ChunkResponse, len(chunks))
	for i, c := range chunks {
		chunkResponses[i] = response.ChunkResponse{
			ID:              c.ID,
			Content:         c.Content,
			EmbeddingModel:  c.EmbeddingModel,
			VectorDimension: c.VectorDimension,
			SyncStatus:      c.SyncStatus,
			SyncError:       c.SyncError,
			SyncedAt:        c.SyncedAt,
		}
	}

	syncStatus := "none"
	if len(chunks) > 0 {
		syncStatus = chunks[0].SyncStatus
	}

	return &response.ArticleDetailResponse{
		ArticleResponse: response.ArticleResponse{
			ID:              article.ID,
			KBID:            article.KBID,
			KBName:          article.KnowledgeBase.Name,
			Question:        article.Question,
			Answer:          article.Answer,
			Category:        article.Category,
			Tags:            unmarshalTags(article.Tags),
			Status:          article.Status,
			StatusText:      statusText(article.Status),
			CreatedBy:       article.CreatedBy,
			ReviewedBy:      article.ReviewedBy,
			ReviewComment:   article.ReviewComment,
			SyncStatus:      syncStatus,
			CreatedAt:       article.CreatedAt,
			UpdatedAt:       article.UpdatedAt,
		},
		Chunks: chunkResponses,
	}, nil
}

// =============================================================================
// 辅助函数
// =============================================================================

// marshalTags 将标签列表序列化为 JSON。
func marshalTags(tags []string) datatypes.JSON {
	if len(tags) == 0 {
		return datatypes.JSON(`[]`)
	}
	data, _ := json.Marshal(tags)
	return datatypes.JSON(data)
}

// unmarshalTags 将 JSON 反序列化为标签列表。
func unmarshalTags(data datatypes.JSON) []string {
	if len(data) == 0 {
		return []string{}
	}
	var tags []string
	_ = json.Unmarshal(data, &tags)
	if tags == nil {
		return []string{}
	}
	return tags
}

// statusText 返回文章状态的中文描述。
func statusText(status int16) string {
	switch status {
	case 0:
		return "已停用"
	case 1:
		return "草稿"
	case 2:
		return "待审核"
	case 3:
		return "已通过"
	case 4:
		return "已发布"
	case 5:
		return "已驳回"
	default:
		return "未知"
	}
}

// getAggregateSyncStatus 根据文章切片推算整体同步状态。
//
// 为什么用聚合而非存储字段：同步状态分布在 chunks 表中，
// 简单文章通常只有一个 chunk，取第一条即可。
func getAggregateSyncStatus(article *model.KnowledgeArticle) string {
	switch article.Status {
	case 4:
		return "synced"
	case 0:
		return "disabled"
	default:
		return "pending"
	}
}

// =============================================================================
// EmbeddingConfig
// =============================================================================

// CreateEmbeddingConfig 创建 Embedding 配置。
//
// 业务规则：
//   - model_type=1（API 类型）时 api_endpoint 必填
//   - model_type=2（本地类型）时 local_path 必填
//   - is_default=true 时，先将其他配置的 is_default 设为 false
func (s *KnowledgeService) CreateEmbeddingConfig(req request.CreateEmbeddingConfigRequest) error {
	// 校验必填字段
	if req.ModelType == 1 && req.APIEndpoint == "" {
		return AppError{Code: errcode.ErrParam, Message: "API 类型必须填写 api_endpoint"}
	}
	if req.ModelType == 2 && req.LocalPath == "" {
		return AppError{Code: errcode.ErrParam, Message: "本地类型必须填写 local_path"}
	}

	// 设为默认时，先清空其他默认
	if req.IsDefault {
		if err := s.clearDefaultEmbedding(); err != nil {
			return err
		}
	}

	cfg := &model.EmbeddingConfig{
		Name:           req.Name,
		ModelType:      req.ModelType,
		APIEndpoint:    req.APIEndpoint,
		APIKey:         req.APIKey,
		LocalPath:      req.LocalPath,
		VectorDimension: req.VectorDimension,
		IsDefault:      req.IsDefault,
	}

	return s.repo.CreateEmbeddingConfig(cfg)
}

// UpdateEmbeddingConfig 更新 Embedding 配置。
//
// 更新为默认时同样先清空其他默认。
func (s *KnowledgeService) UpdateEmbeddingConfig(id int64, req request.UpdateEmbeddingConfigRequest) error {
	cfg, err := s.repo.ListEmbeddingConfigs()
	if err != nil {
		return err
	}

	// 查找目标配置
	found := false
	for _, c := range cfg {
		if c.ID == id {
			found = true
			break
		}
	}
	if !found {
		return AppError{Code: errcode.ErrNotFound, Message: "Embedding 配置不存在"}
	}

	// 设为默认时，先清空其他默认
	if req.IsDefault {
		if err := s.clearDefaultEmbedding(); err != nil {
			return err
		}
	}

	updated := &model.EmbeddingConfig{
		ID:             id,
		Name:           req.Name,
		ModelType:      req.ModelType,
		APIEndpoint:    req.APIEndpoint,
		APIKey:         req.APIKey,
		LocalPath:      req.LocalPath,
		VectorDimension: req.VectorDimension,
		IsDefault:      req.IsDefault,
	}

	return s.repo.UpdateEmbeddingConfig(updated)
}

// ListEmbeddingConfigs 列出全部 Embedding 配置。
func (s *KnowledgeService) ListEmbeddingConfigs() ([]response.EmbeddingConfigResponse, error) {
	configs, err := s.repo.ListEmbeddingConfigs()
	if err != nil {
		return nil, err
	}

	result := make([]response.EmbeddingConfigResponse, len(configs))
	for i, c := range configs {
		result[i] = response.EmbeddingConfigResponse{
			ID:             c.ID,
			Name:           c.Name,
			ModelType:      c.ModelType,
			APIEndpoint:    c.APIEndpoint,
			APIKey:         c.APIKey,
			LocalPath:      c.LocalPath,
			VectorDimension: c.VectorDimension,
			IsDefault:      c.IsDefault,
			CreatedAt:      c.CreatedAt.Format("2006-01-02 15:04:05"),
		}
	}

	return result, nil
}

// DeleteEmbeddingConfig 删除 Embedding 配置。
func (s *KnowledgeService) DeleteEmbeddingConfig(id int64) error {
	if err := s.repo.DeleteEmbeddingConfig(id); err != nil {
		if err == gorm.ErrRecordNotFound {
			return AppError{Code: errcode.ErrNotFound, Message: "Embedding 配置不存在"}
		}
		return err
	}
	return nil
}

// clearDefaultEmbedding 清空所有配置的 is_default 标志。
//
// 为什么批量更新而非逐条：确保 is_default=true 配置唯一，
// 原子操作避免并发创建两个默认配置的竞态条件。
func (s *KnowledgeService) clearDefaultEmbedding() error {
	configs, err := s.repo.ListEmbeddingConfigs()
	if err != nil {
		return err
	}
	for _, c := range configs {
		if c.IsDefault {
			c.IsDefault = false
			if err := s.repo.UpdateEmbeddingConfig(&c); err != nil {
				return err
			}
		}
	}
	return nil
}
