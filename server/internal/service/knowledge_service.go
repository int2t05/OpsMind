// Package service 实现知识库管理业务逻辑。
//
// KnowledgeService 统一管理知识库 CRUD、文章审核发布、pgvector 管道操作和文档上传。
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"opsmind/internal/adapter"
	"opsmind/internal/dto/request"
	"opsmind/internal/dto/response"
	"opsmind/internal/model"
	"opsmind/internal/rag"
	"opsmind/pkg/errcode"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// 消费者接口——KnowledgeService 仅暴露它实际使用的依赖方法，
// 遵循 Go "accept interfaces, return structs" 惯例，便于测试 mock。
type knowledgeChunker interface {
	Split(text string) []string
}

type knowledgeEmbedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, int, error)
}

type knowledgeDocParser interface {
	Parse(reader io.Reader, fileType string) (string, error)
}

// knowledgeRepo KnowledgeService 使用的仓库方法子集。
type knowledgeRepo interface {
	FindKBByID(id int64) (*model.KnowledgeBase, error)
	FindArticleByID(id int64) (*model.KnowledgeArticle, error)
	CreateArticle(article *model.KnowledgeArticle) error
	UpdateArticle(article *model.KnowledgeArticle) error
	UpdateArticleStatus(id int64, status int) error
	CreateKB(kb *model.KnowledgeBase) error
	UpdateKB(kb *model.KnowledgeBase) error
	ListKBs() ([]model.KnowledgeBase, error)
	ListArticles(kbID int64, status int, page, pageSize int) ([]model.KnowledgeArticle, int64, error)
	FindChunksByArticleID(articleID int64) ([]model.KnowledgeChunk, error)
}

// KnowledgeService 知识库管理服务。
//
// 所有依赖使用接口类型，便于测试 mock。
type KnowledgeService struct {
	repo      knowledgeRepo
	chunker   knowledgeChunker
	embedder  knowledgeEmbedder
	store     adapter.VectorStore
	docParser knowledgeDocParser
	processor *rag.Processor
}

// NewKnowledgeService 创建 KnowledgeService 实例。
//
// 接受 interface{} 参数，通过类型断言适配——遵循 Go "accept interfaces, return structs"。
// repo/chunker/embedder/store/docParser/processor 可以为 nil（测试或部分功能不需要时）。
func NewKnowledgeService(repo interface{}, chunker interface{}, embedder interface{}, store adapter.VectorStore, docParser interface{}, processor *rag.Processor) *KnowledgeService {
	svc := &KnowledgeService{
		store:     store,
		processor: processor,
	}
	if r, ok := repo.(knowledgeRepo); ok {
		svc.repo = r
	}
	if c, ok := chunker.(knowledgeChunker); ok {
		svc.chunker = c
	}
	if e, ok := embedder.(knowledgeEmbedder); ok {
		svc.embedder = e
	}
	if d, ok := docParser.(knowledgeDocParser); ok {
		svc.docParser = d
	}
	return svc
}

// =============================================================================
// KnowledgeBase
// =============================================================================

// CreateKB 创建知识库（仅写 PostgreSQL）。
func (s *KnowledgeService) CreateKB(req request.CreateKBRequest, userID int64) error {
	kb := &model.KnowledgeBase{
		Name:        req.Name,
		Description: req.Description,
		CreatedBy:   userID,
	}
	return s.repo.CreateKB(kb)
}

// UpdateKB 更新知识库信息。
func (s *KnowledgeService) UpdateKB(id int64, req request.UpdateKBRequest) error {
	kb, err := s.repo.FindKBByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errcode.AppError{Code: errcode.ErrNotFound, Message: "知识库不存在"}
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
			ID:              kb.ID,
			Name:            kb.Name,
			Description:     kb.Description,
			EmbeddingModel:  kb.EmbeddingModel,
			VectorDimension: kb.VectorDimension,
			CreatedBy:       kb.CreatedBy,
			CreatedAt:       kb.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:       kb.UpdatedAt.Format("2006-01-02 15:04:05"),
		}
	}
	return result, nil
}

// =============================================================================
// KnowledgeArticle CRUD
// =============================================================================

// CreateArticle 创建知识文章（草稿状态）。
func (s *KnowledgeService) CreateArticle(req request.CreateArticleRequest, userID int64) error {
	_, err := s.repo.FindKBByID(req.KBID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errcode.AppError{Code: errcode.ErrNotFound, Message: "知识库不存在"}
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
			return errcode.AppError{Code: errcode.ErrNotFound, Message: "文章不存在"}
		}
		return err
	}
	if article.Status != 1 && article.Status != 5 {
		return errcode.AppError{Code: errcode.ErrParam, Message: "仅草稿和驳回状态可编辑"}
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
			return errcode.AppError{Code: errcode.ErrNotFound, Message: "文章不存在"}
		}
		return err
	}
	if article.Status != 1 {
		return errcode.AppError{Code: errcode.ErrParam, Message: "仅草稿状态可提交审核"}
	}
	return s.repo.UpdateArticleStatus(id, 2)
}

// Review 审核文章（待审核→已通过/已驳回）。
func (s *KnowledgeService) Review(id int64, reviewerID int64, req request.ReviewRequest) error {
	article, err := s.repo.FindArticleByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errcode.AppError{Code: errcode.ErrNotFound, Message: "文章不存在"}
		}
		return err
	}
	if article.Status != 2 {
		return errcode.AppError{Code: errcode.ErrParam, Message: "仅待审核状态可审核"}
	}
	if article.CreatedBy == reviewerID {
		return errcode.AppError{Code: errcode.ErrParam, Message: "审核人不能是创建人"}
	}
	if req.Approved {
		article.Status = 3
		article.ReviewedBy = &reviewerID
		return s.repo.UpdateArticle(article)
	}
	if strings.TrimSpace(req.ReviewComment) == "" {
		return errcode.AppError{Code: errcode.ErrParam, Message: "驳回时必须填写审核意见"}
	}
	article.Status = 5
	article.ReviewComment = req.ReviewComment
	article.ReviewedBy = &reviewerID
	return s.repo.UpdateArticle(article)
}

// =============================================================================
// Publish / Disable / Enable
// =============================================================================

// Publish 发布文章——分块→embedding→pgvector 写入。
//
// 流程：
//  1. 校验状态（仅已通过 status=3 可发布）
//  2. Chunker.Split → 文本分块
//  3. Embedder.Embed → 生成向量
//  4. VectorStore.DeleteByArticle → 清除旧向量
//  5. VectorStore.BatchInsert → 写入新向量
//  6. 更新文章状态为已发布 status=4
func (s *KnowledgeService) Publish(id int64, publisherID int64) error {
	if s.chunker == nil || s.embedder == nil || s.store == nil {
		return errcode.AppError{Code: errcode.ErrUnknown, Message: "管道未初始化（chunker/embedder/store 为空）"}
	}

	article, err := s.repo.FindArticleByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errcode.AppError{Code: errcode.ErrNotFound, Message: "文章不存在"}
		}
		return err
	}
	if article.Status != 3 {
		return errcode.AppError{Code: errcode.ErrParam, Message: "仅已审核通过的文章可发布"}
	}

	// Step 1: 分块
	content := article.Answer
	chunks := s.chunker.Split(content)
	if len(chunks) == 0 {
		return errcode.AppError{Code: errcode.ErrUnknown, Message: "分块结果为空"}
	}

	// Step 2: Embedding
	ctx := context.Background()
	vectors, dimension, err := s.embedder.Embed(ctx, chunks)
	if err != nil {
		return errcode.AppError{Code: errcode.ErrRAGUnavailable, Message: "生成向量失败: " + err.Error()}
	}
	if len(vectors) != len(chunks) {
		return errcode.AppError{Code: errcode.ErrUnknown, Message: fmt.Sprintf("向量数与分块数不匹配: %d vs %d", len(vectors), len(chunks))}
	}

	// Step 3: 清除旧向量
	if err := s.store.DeleteByArticle(ctx, id); err != nil {
		return errcode.AppError{Code: errcode.ErrRAGUnavailable, Message: "清除旧向量失败: " + err.Error()}
	}

	// Step 4: 写入新向量
	vc := make([]adapter.VectorChunk, len(chunks))
	for i, chunk := range chunks {
		vc[i] = adapter.VectorChunk{
			ArticleID:       id,
			KBID:            article.KBID,
			Content:         chunk,
			ChunkIndex:      i,
			Embedding:       vectors[i],
			EmbeddingModel:  "bge-m3",
			VectorDimension: dimension,
		}
	}
	if err := s.store.BatchInsert(ctx, vc); err != nil {
		return errcode.AppError{Code: errcode.ErrRAGUnavailable, Message: "写入向量失败: " + err.Error()}
	}

	// Step 5: 更新状态
	article.Status = 4
	article.PublishedBy = &publisherID
	return s.repo.UpdateArticle(article)
}

// Disable 停用文章——从 pgvector 删除向量并更新状态。
func (s *KnowledgeService) Disable(id int64) error {
	article, err := s.repo.FindArticleByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errcode.AppError{Code: errcode.ErrNotFound, Message: "文章不存在"}
		}
		return err
	}

	// 从 pgvector 删除向量（如果有 vector store）
	if s.store != nil {
		ctx := context.Background()
		if err := s.store.DeleteByArticle(ctx, id); err != nil {
			return errcode.AppError{Code: errcode.ErrRAGUnavailable, Message: "删除向量失败: " + err.Error()}
		}
	}

	article.Status = 0
	return s.repo.UpdateArticle(article)
}

// Enable 恢复已停用文章为草稿状态。
func (s *KnowledgeService) Enable(id int64) error {
	article, err := s.repo.FindArticleByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errcode.AppError{Code: errcode.ErrNotFound, Message: "文章不存在"}
		}
		return err
	}
	if article.Status != 0 {
		return errcode.AppError{Code: errcode.ErrParam, Message: "仅已停用状态的文章可恢复"}
	}
	article.Status = 1 // 已停用 → 草稿
	return s.repo.UpdateArticle(article)
}

// =============================================================================
// List / Detail
// =============================================================================

// ListArticles 分页查询文章列表。
func (s *KnowledgeService) ListArticles(kbID int64, status int, page, pageSize int) (*response.ArticleListResponse, error) {
	articles, total, err := s.repo.ListArticles(kbID, status, page, pageSize)
	if err != nil {
		return nil, err
	}

	result := make([]response.ArticleResponse, len(articles))
	for i, a := range articles {
		syncStatus := getAggregateSyncStatus(&a)
		result[i] = response.ArticleResponse{
			ID:            a.ID,
			KBID:          a.KBID,
			KBName:        a.KnowledgeBase.Name,
			Question:      a.Question,
			Answer:        a.Answer,
			Category:      a.Category,
			Tags:          unmarshalTags(a.Tags),
			Status:        a.Status,
			StatusText:    statusText(a.Status),
			CreatedBy:     a.CreatedBy,
			ReviewedBy:    a.ReviewedBy,
			ReviewComment: a.ReviewComment,
			SyncStatus:    syncStatus,
			CreatedAt:     a.CreatedAt,
			UpdatedAt:     a.UpdatedAt,
		}
	}

	return &response.ArticleListResponse{
		Articles: result,
		Total:    total,
	}, nil
}

// GetArticleDetail 获取文章详情。
func (s *KnowledgeService) GetArticleDetail(id int64) (*response.ArticleDetailResponse, error) {
	article, err := s.repo.FindArticleByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errcode.AppError{Code: errcode.ErrNotFound, Message: "文章不存在"}
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
			ID:            article.ID,
			KBID:          article.KBID,
			KBName:        article.KnowledgeBase.Name,
			Question:      article.Question,
			Answer:        article.Answer,
			Category:      article.Category,
			Tags:          unmarshalTags(article.Tags),
			Status:        article.Status,
			StatusText:    statusText(article.Status),
			CreatedBy:     article.CreatedBy,
			ReviewedBy:    article.ReviewedBy,
			ReviewComment: article.ReviewComment,
			SyncStatus:    syncStatus,
			CreatedAt:     article.CreatedAt,
			UpdatedAt:     article.UpdatedAt,
		},
		Chunks: chunkResponses,
	}, nil
}

// =============================================================================
// 文档上传与处理
// =============================================================================

// UploadDocuments 上传文档到知识库（解析→创建文章→入队异步处理）。
func (s *KnowledgeService) UploadDocuments(kbID int64, userID int64, filename string, fileType string, content io.Reader) (*model.KnowledgeArticle, error) {
	if s.docParser == nil {
		return nil, errcode.AppError{Code: errcode.ErrUnknown, Message: "文档解析器未初始化"}
	}

	text, err := s.docParser.Parse(content, fileType)
	if err != nil {
		return nil, errcode.AppError{Code: errcode.ErrParam, Message: "文档解析失败: " + err.Error()}
	}
	if strings.TrimSpace(text) == "" {
		return nil, errcode.AppError{Code: errcode.ErrParam, Message: "文档内容为空"}
	}

	article := &model.KnowledgeArticle{
		KBID:      kbID,
		Question:  filename,
		Answer:    text,
		Category:  "文档上传",
		Status:    1, // 草稿
		CreatedBy: userID,
	}
	if err := s.repo.CreateArticle(article); err != nil {
		return nil, errcode.AppError{Code: errcode.ErrUnknown, Message: "创建文章失败: " + err.Error()}
	}

	if s.processor != nil {
		task := rag.ProcessTask{
			ArticleID: article.ID,
			KBID:      kbID,
			Content:   text,
			OnStatusChange: func(aID int64, status, errMsg string) {
				_ = s.repo.UpdateArticleStatus(aID, mapProcessStatus(status))
			},
		}
		s.processor.Submit(task)
	}

	return article, nil
}

// GetDocumentStatus 查询文档处理状态。
func (s *KnowledgeService) GetDocumentStatus(articleID int64) (string, error) {
	article, err := s.repo.FindArticleByID(articleID)
	if err != nil {
		return "", errcode.AppError{Code: errcode.ErrNotFound, Message: "文章不存在"}
	}
	return mapArticleToProcessStatus(article), nil
}

// RetryDocument 重试文档处理（重新入队）。
func (s *KnowledgeService) RetryDocument(articleID int64) error {
	article, err := s.repo.FindArticleByID(articleID)
	if err != nil {
		return errcode.AppError{Code: errcode.ErrNotFound, Message: "文章不存在"}
	}
	if s.processor == nil {
		return errcode.AppError{Code: errcode.ErrUnknown, Message: "文档处理器未初始化"}
	}

	_ = s.repo.UpdateArticleStatus(articleID, 1)
	task := rag.ProcessTask{
		ArticleID: articleID,
		KBID:      article.KBID,
		Content:   article.Answer,
	}
	s.processor.Submit(task)
	return nil
}

// =============================================================================
// 辅助函数
// =============================================================================

func marshalTags(tags []string) datatypes.JSON {
	if len(tags) == 0 {
		return datatypes.JSON(`[]`)
	}
	data, _ := json.Marshal(tags)
	return datatypes.JSON(data)
}

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

// mapProcessStatus 将 Processor 阶段映射为文章状态。
func mapProcessStatus(status string) int {
	switch status {
	case "chunking", "embedding", "indexing":
		return 1
	case "completed":
		return 3
	case "failed":
		return 1
	default:
		return 1
	}
}

// mapArticleToProcessStatus 将文章状态映射为处理阶段描述。
func mapArticleToProcessStatus(article *model.KnowledgeArticle) string {
	switch article.Status {
	case 1:
		return "pending"
	case 3:
		return "completed"
	case 4:
		return "published"
	case 0:
		return "disabled"
	default:
		return "unknown"
	}
}
