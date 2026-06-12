// Package service 实现知识库管理业务逻辑。
//
// KnowledgeService 统一管理知识库 CRUD、文章审核发布、pgvector 管道操作和文档上传。
package service

import (
	"errors"
	"context"
	"log/slog"
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
// repo/chunker/embedder/store/docParser/processor 可以为 nil（测试或部分功能不需要时）。
// 直接使用具体接口类型，编译期校验传入类型。
func NewKnowledgeService(repo knowledgeRepo, chunker knowledgeChunker, embedder knowledgeEmbedder, store adapter.VectorStore, docParser knowledgeDocParser, processor *rag.Processor) *KnowledgeService {
	// TODO(service/knowledge): 明确哪些依赖在生产路径必须非 nil，哪些仅测试可 nil。
	// 当前 Publish/Upload/Retry 到运行时才发现依赖缺失，启动期无法暴露装配错误。
	return &KnowledgeService{
		repo:      repo,
		chunker:   chunker,
		embedder:  embedder,
		store:     store,
		docParser: docParser,
		processor: processor,
	}
}

// =============================================================================
// KnowledgeBase
// =============================================================================

// CreateKB 创建知识库（仅写 PostgreSQL）。
func (s *KnowledgeService) CreateKB(req request.CreateKBRequest, userID int64) error {
	// TODO(service/knowledge): CreateKB 未写入 EmbeddingModel、VectorDimension、LLMConfigID。
	// 与 docs/API/knowledge.md 的 v2 知识库模型不一致，后续发布向量时会拿到空模型配置。
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
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
	// TODO(service/knowledge): DTO 仍使用 question/answer，PRD/API v2 已改为 title/content/source_type。
	// 应统一模型语言，否则前端 KnowledgeEdit 的 title/content 会和后端字段错位。
	_, err := s.repo.FindKBByID(req.KBID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
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
	// TODO(service/knowledge): userID 参数当前未使用。
	// 如果需要审计或作者权限校验，应在这里使用；否则从签名移除避免误导调用方。
	article, err := s.repo.FindArticleByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.AppError{Code: errcode.ErrNotFound, Message: "文章不存在"}
		}
		return err
	}
	if article.Status != 1 && article.Status != 5 {
		// TODO(service/knowledge): 这里的 5 被当作驳回，但 enums.go 中 ArticleStatusRejected=5，API 文档中驳回=6。
		// 应统一状态枚举，避免前后端对文章生命周期理解不一致。
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
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
		// TODO(service/knowledge): 管道未初始化应映射为 ErrRAGUnavailable，而不是 ErrUnknown。
		// 调用方可以据此展示“RAG 服务不可用”并触发运维排查。
		return errcode.AppError{Code: errcode.ErrUnknown, Message: "管道未初始化（chunker/embedder/store 为空）"}
	}

	article, err := s.repo.FindArticleByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.AppError{Code: errcode.ErrNotFound, Message: "文章不存在"}
		}
		return err
	}
	if article.Status != 3 {
		// TODO(service/knowledge): status=3 在 enums.go 表示已发布，但这里被当作审核通过。
		// 需要增加 ArticleStatusApproved 或调整现有枚举，避免发布逻辑和状态文案冲突。
		return errcode.AppError{Code: errcode.ErrParam, Message: "仅已审核通过的文章可发布"}
	}

	// Step 1: 分块
	// TODO(service/knowledge): 应将标题和正文一起进入分块，例如 title + "\n\n" + content。
	// 只对 Answer 分块会丢失标题语义，影响短问答类知识的召回。
	content := article.Answer
	chunks := s.chunker.Split(content)
	if len(chunks) == 0 {
		return errcode.AppError{Code: errcode.ErrUnknown, Message: "分块结果为空"}
	}

	// Step 2: Embedding
	// TODO(service/knowledge): 使用 context.Background 会忽略 HTTP 请求取消和超时。
	// Publish 应接收 ctx，由 Handler 传入 c.Request.Context()，避免用户断开后继续消耗 LLM/DB 资源。
	ctx := context.Background()
	vectors, dimension, err := s.embedder.Embed(ctx, chunks)
	if err != nil {
		return errcode.AppError{Code: errcode.ErrRAGUnavailable, Message: "生成向量失败: " + err.Error()}
	}
	if len(vectors) != len(chunks) {
		return errcode.AppError{Code: errcode.ErrUnknown, Message: fmt.Sprintf("向量数与分块数不匹配: %d vs %d", len(vectors), len(chunks))}
	}

	// Step 3: 清除旧向量
	// TODO(service/knowledge): DeleteByArticle 和 BatchInsert 应处于同一事务或采用先写临时版本再切换。
	// 当前先删后写失败会导致已发布文章丢失全部向量。
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
			EmbeddingModel:  article.KnowledgeBase.EmbeddingModel, // 从知识库配置读取
			VectorDimension: dimension,
		}
	}
	if err := s.store.BatchInsert(ctx, vc); err != nil {
		// TODO(service/knowledge): 发布失败时应按 API 文档写 process_status=failed/process_error。
		// 当前只返回错误，文章状态和错误原因不会持久化，前端无法展示可重试原因。
		return errcode.AppError{Code: errcode.ErrRAGUnavailable, Message: "写入向量失败: " + err.Error()}
	}

	// Step 5: 更新状态
	article.Status = 4
	article.PublishedBy = &publisherID
	return s.repo.UpdateArticle(article)
}

// Disable 停用文章——从 pgvector 删除向量并更新状态。
func (s *KnowledgeService) Disable(id int64) error {
	// TODO(service/knowledge): Disable 未校验当前状态是否为已发布。
	// 草稿/驳回文章停用会让状态机绕过审核流程。
	article, err := s.repo.FindArticleByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.AppError{Code: errcode.ErrNotFound, Message: "文章不存在"}
		}
		return err
	}

	// 从 pgvector 删除向量（如果有 vector store）
	if s.store != nil {
		// TODO(service/knowledge): Disable 使用 context.Background，同样应接收请求 ctx。
		// 向量删除是外部 I/O，必须可取消并受超时控制。
		ctx := context.Background()
		if err := s.store.DeleteByArticle(ctx, id); err != nil {
			return errcode.AppError{Code: errcode.ErrRAGUnavailable, Message: "删除向量失败: " + err.Error()}
		}
	}

	article.Status = model.ArticleStatusDisabled
	return s.repo.UpdateArticle(article)
}

// Enable 恢复已停用文章为草稿状态。
func (s *KnowledgeService) Enable(id int64) error {
	// TODO(service/knowledge): docs/API 要求启用停用文章后重新分块→embedding→写入向量并恢复发布。
	// 当前只把状态改回草稿，行为和接口文档不一致。
	article, err := s.repo.FindArticleByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.AppError{Code: errcode.ErrNotFound, Message: "文章不存在"}
		}
		return err
	}
	if article.Status != model.ArticleStatusDisabled {
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
	// TODO(service/knowledge): source_type/process_status 筛选未实现。
	// API 文档已经定义这些查询参数，后端忽略会导致后台列表筛选失效。
	articles, total, err := s.repo.ListArticles(kbID, status, page, pageSize)
	if err != nil {
		return nil, err
	}

	result := make([]response.ArticleResponse, len(articles))
	for i, a := range articles {
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
	// TODO(service/knowledge): 详情响应仍返回 sync_status/sync_error/synced_at。
	// v2 文档已移除 AnythingLLM 同步概念，应改为 process_status/process_error/chunk_index。
	article, err := s.repo.FindArticleByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
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
			KBID:            c.KBID,
			Content:         c.Content,
			ChunkIndex:      c.ChunkIndex,
			EmbeddingModel:  c.EmbeddingModel,
			VectorDimension: c.VectorDimension,
		}
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
//
// fileSize 用于大小上限校验（最大 50MB），fileType 用于格式白名单校验。
func (s *KnowledgeService) UploadDocuments(kbID int64, userID int64, filename string, fileType string, fileSize int64, content io.Reader) (*model.KnowledgeArticle, error) {
	// TODO(service/knowledge): 上传流程当前没有使用 StorageClient/MinIO，直接解析 multipart 内容。
	// 与 PRD 的“上传到对象存储后异步解析”不一致，服务重启后也无法重试原始文件。
	// 格式白名单校验
	allowedTypes := map[string]bool{"pdf": true, "docx": true, "md": true, "txt": true}
	if !allowedTypes[fileType] {
		return nil, errcode.AppError{Code: errcode.ErrParam, Message: "不支持的文件格式: " + fileType + "（支持: pdf/docx/md/txt）"}
	}

	// 文件大小上限校验
	// TODO(service/knowledge): API 文档写单文件最大 20MB，这里实际限制 50MB。
	// 应统一前后端提示、后端校验和文档约束。
	const maxSize = 50 * 1024 * 1024
	if fileSize > maxSize {
		return nil, errcode.AppError{Code: errcode.ErrParam, Message: "文件大小超过限制（最大 50MB）"}
	}

	if s.docParser == nil {
		return nil, errcode.AppError{Code: errcode.ErrUnknown, Message: "文档解析器未初始化"}
	}

	text, err := s.docParser.Parse(content, fileType)
	if err != nil {
		return nil, errcode.AppError{Code: errcode.ErrParam, Message: "文档解析失败: " + err.Error()}
	}
	// TODO(service/knowledge): 解析在 HTTP 请求内同步完成，较大 PDF/DOCX 会阻塞上传接口。
	// 更符合设计的做法是先落对象存储和 pending 记录，再由 Processor 异步下载解析。
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
		// TODO(service/knowledge): Processor 回调只更新 status 数字，丢弃 errMsg。
		// 需要持久化 process_error，前端才能展示失败原因和重试指导。
		task := rag.ProcessTask{
			ArticleID: article.ID,
			KBID:      kbID,
			Content:   text,
			OnStatusChange: func(aID int64, status, errMsg string) {
				_ = s.repo.UpdateArticleStatus(aID, mapProcessStatus(status))
			},
		}
		if err := s.processor.Submit(task); err != nil {
			return nil, errcode.AppError{Code: errcode.ErrUnknown, Message: "提交处理任务失败: " + err.Error()}
		}
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
	// TODO(service/knowledge): RetryDocument 未校验当前处理状态是否 failed。
	// 非失败状态重复入队可能造成同一文章重复写入向量分块。
	article, err := s.repo.FindArticleByID(articleID)
	if err != nil {
		return errcode.AppError{Code: errcode.ErrNotFound, Message: "文章不存在"}
	}
	if s.processor == nil {
		return errcode.AppError{Code: errcode.ErrUnknown, Message: "文档处理器未初始化"}
	}

	if err := s.repo.UpdateArticleStatus(articleID, 1); err != nil {
		slog.Warn("更新文章状态失败，不阻断主流程", "article_id", articleID, "error", err)
	}
	task := rag.ProcessTask{
		ArticleID: articleID,
		KBID:      article.KBID,
		Content:   article.Answer,
	}
	if err := s.processor.Submit(task); err != nil {
		return errcode.AppError{Code: errcode.ErrUnknown, Message: "提交处理任务失败: " + err.Error()}
	}
	return nil
}

// =============================================================================
// 辅助函数
// =============================================================================

func marshalTags(tags []string) datatypes.JSON {
	// TODO(service/knowledge): tags 应 trim、去重、限制数量和单个长度。
	// 标签直接写入 JSONB 会让前端和检索筛选承受脏数据。
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

// mapProcessStatus 将 Processor 阶段映射为文章状态。
func mapProcessStatus(status string) int {
	// TODO(service/knowledge): 用文章 status 承载文档处理进度会混淆“审核状态”和“处理状态”两个状态机。
	// 模型应增加 process_status/process_error 字段，保持两个生命周期独立。
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
