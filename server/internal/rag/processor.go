// Package rag 实现自建 RAG 检索引擎。
//
// processor.go 实现文档异步处理管线。
//
// 为什么使用 goroutine pool 而非直接同步处理：
// 文档处理（解析→分块→embedding→pgvector 写入）可能耗时数十秒甚至数分钟，
// 同步处理会阻塞 HTTP 请求。goroutine pool 模式将耗时任务异步化，
// 客户端通过 process_status 轮询进度。
//
// 优雅关闭设计：
// Stop() 关闭任务队列（不再接收新任务），worker 完成当前正在处理的任务后退出，
// 保证不丢失已提交但未完成的任务。
package rag

import (
	"context"
	"fmt"
	"sync"

	"opsmind/internal/adapter"
)

// =============================================================================
// ProcessTask — 处理任务
// =============================================================================

// ProcessTask 单个文档处理任务。
type ProcessTask struct {
	ArticleID      int64                                     `json:"article_id"`
	KBID           int64                                     `json:"kb_id"`
	Content        string                                    `json:"content"`         // 文档原始内容（已从 MinIO 下载）
	OnStatusChange func(articleID int64, status, errMsg string) `json:"-"`           // 状态变更回调
}

// =============================================================================
// Processor — 异步文档处理器
// =============================================================================

// Processor 管理文档处理的 goroutine pool。
type Processor struct {
	parser   *DocParser
	chunker  *Chunker
	embedder *Embedder
	store    adapter.VectorStore

	taskCh   chan ProcessTask
	poolSize int
	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewProcessor 创建文档处理器实例。
//
// poolSize 为 worker goroutine 数量，建议 2-4。
func NewProcessor(parser *DocParser, chunker *Chunker, embedder *Embedder, store adapter.VectorStore, poolSize int) *Processor {
	if poolSize <= 0 {
		poolSize = 2
	}
	ctx, cancel := context.WithCancel(context.Background())
	p := &Processor{
		parser:   parser,
		chunker:  chunker,
		embedder: embedder,
		store:    store,
		taskCh:   make(chan ProcessTask, 100), // 缓冲队列
		poolSize: poolSize,
		ctx:      ctx,
		cancel:   cancel,
	}

	// 启动 worker pool
	for i := 0; i < poolSize; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
	return p
}

// Submit 提交处理任务（非阻塞）。
//
// 任务进入缓冲队列后立即返回，不等待处理完成。
func (p *Processor) Submit(task ProcessTask) {
	select {
	case p.taskCh <- task:
	default:
		// 队列满时丢弃（生产环境应由调用方重试）
		if task.OnStatusChange != nil {
			task.OnStatusChange(task.ArticleID, "failed", "处理队列已满")
		}
	}
}

// Stop 优雅关闭处理器。
//
// 先 cancel context（中断正在进行的 I/O），关闭任务通道（不再接收新任务），
// 然后等待所有 worker 完成当前任务后退出。
func (p *Processor) Stop() {
	p.cancel() // 先中断所有进行中的 I/O 操作
	close(p.taskCh)
	p.wg.Wait()
}

// worker 处理任务循环。
//
// 流程：接收任务→解析→分块→embedding→pgvector 写入。
// 每阶段失败时更新 process_status 为 failed 并跳过该文档。
func (p *Processor) worker(id int) {
	defer p.wg.Done()

	for task := range p.taskCh {
		p.processTask(task)
	}
}

// processTask 处理单个文档的完整流程。
func (p *Processor) processTask(task ProcessTask) {
	ctx := p.ctx
	articleID := task.ArticleID

	// 阶段 1: 分块
	p.updateStatus(task, "chunking", "")
	chunks := p.chunker.Split(task.Content)
	if len(chunks) == 0 {
		p.updateStatus(task, "failed", "分块结果为空")
		return
	}

	// 阶段 2: Embedding
	p.updateStatus(task, "embedding", "")
	vectors, _, err := p.embedder.Embed(ctx, chunks)
	if err != nil {
		p.updateStatus(task, "failed", fmt.Sprintf("embedding 失败: %v", err))
		return
	}
	if len(vectors) != len(chunks) {
		p.updateStatus(task, "failed", fmt.Sprintf("embedding 数量不匹配: 期望 %d, 实际 %d", len(chunks), len(vectors)))
		return
	}

	// 阶段 3: 写入 pgvector
	p.updateStatus(task, "indexing", "")
	vc := make([]adapter.VectorChunk, len(chunks))
	for i, chunk := range chunks {
		vc[i] = adapter.VectorChunk{
			ArticleID:       articleID,
			KBID:            task.KBID,
			Content:         chunk,
			ChunkIndex:      i,
			Embedding:       vectors[i],
			EmbeddingModel:  "", // 从配置读取
			VectorDimension: len(vectors[i]),
		}
	}

	// 阶段 3: 写入 pgvector
	// 注意：这里不执行 DeleteByArticle——避免「先删后写失败导致数据丢失」。
	// 旧向量由调用方（Service 层）在重新发布时负责清理。
	p.updateStatus(task, "indexing", "")
	if err := p.store.BatchInsert(ctx, vc); err != nil {
		p.updateStatus(task, "failed", fmt.Sprintf("写入向量失败: %v", err))
		return
	}

	p.updateStatus(task, "completed", "")
}

// updateStatus 更新处理状态（通过回调）。
func (p *Processor) updateStatus(task ProcessTask, status, errMsg string) {
	if task.OnStatusChange != nil {
		task.OnStatusChange(task.ArticleID, status, errMsg)
	}
}
