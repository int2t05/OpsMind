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
	"strings"
	"sync"

	"opsmind/internal/adapter"
)

// =============================================================================
// ProcessTask — 处理任务
// =============================================================================

// ProcessTask 单个文档处理任务。
//
// 支持两种来源：
//   - MinIO 路径：设置 Bucket/Key/FileType，worker 自动下载并解析
//   - 纯文本：设置 Content，worker 直接分块（手动创建的文章）
type ProcessTask struct {
	ArticleID      int64                                          `json:"article_id"`
	KBID           int64                                          `json:"kb_id"`
	Content        string                                         `json:"content"`  // 纯文本内容（与 MinIO 路径二选一）
	Bucket         string                                         `json:"bucket"`   // MinIO bucket（如 opsmind-documents）
	Key            string                                         `json:"key"`      // MinIO object key
	FileType       string                                         `json:"file_type"` // 文件类型扩展名（用于选择解析器）
	OnStatusChange func(articleID int64, status, errMsg string) `json:"-"`        // 状态变更回调
	OnMetrics      func(articleID int64, wordCount, chunkCount int) `json:"-"`    // 分块后回调（写 WordCount/ChunkCount）
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
	storage  adapter.StorageClient // MinIO 对象存储（nil 时 MinIO 路径任务降级为纯文本）

	taskCh   chan ProcessTask
	poolSize int
	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewProcessor 创建文档处理器实例。
//
// storage 可以为 nil（MinIO 不可用时自动降级到 Content 模式）。
// poolSize 为 worker goroutine 数量，建议 2-4。
func NewProcessor(parser *DocParser, chunker *Chunker, embedder *Embedder, store adapter.VectorStore, storage adapter.StorageClient, poolSize int) *Processor {
	if poolSize <= 0 {
		poolSize = 2
	}
	ctx, cancel := context.WithCancel(context.Background())
	p := &Processor{
		parser:   parser,
		chunker:  chunker,
		embedder: embedder,
		store:    store,
		storage:  storage,
		taskCh:   make(chan ProcessTask, 100),
		poolSize: poolSize,
		ctx:      ctx,
		cancel:   cancel,
	}

	for i := 0; i < poolSize; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
	return p
}

// Submit 提交处理任务（非阻塞）。
//
// 任务进入缓冲队列后立即返回，不等待处理完成。
// 队列满时返回 error，调用方可据此重试或向客户端返回错误。
func (p *Processor) Submit(task ProcessTask) error {
	// TODO(rag/processor): Stop 后 Submit 会向已关闭 channel 发送并 panic。
	// 需要引入 stopped 标志或用 recover 转换为 error。
	select {
	case p.taskCh <- task:
		return nil
	default:
		// 队列满时通知回调并返回错误，调用方可据此响应
		if task.OnStatusChange != nil {
			task.OnStatusChange(task.ArticleID, "failed", "处理队列已满")
		}
		return fmt.Errorf("处理队列已满")
	}
}

// Stop 优雅关闭处理器。
//
// 先 cancel context（中断正在进行的 I/O），关闭任务通道（不再接收新任务），
// 然后等待所有 worker 完成当前任务后退出。
func (p *Processor) Stop() {
	// TODO(rag/processor): Stop 不是幂等的，重复调用 close(taskCh) 会 panic。
	// Scheduler/HTTP 优雅关闭多处调用时需要保证安全。
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

	// TODO(rag/processor): worker 无 panic recovery——若 processTask 发生 panic，
	// goroutine 直接退出，poolSize 减小。应加 recover() 并记录错误后继续。
	// TODO(rag/processor): 所有任务共用 p.ctx，单个任务无独立超时——卡住的下载/embedding
	// 会永久占用 worker goroutine。应给每个 task 派生带超时的子 context。
	for task := range p.taskCh {
		p.processTask(task)
	}
}

// processTask 处理单个文档的完整流程。
//
// 支持两种模式：
//   - MinIO 路径模式：从 MinIO 下载原始文件 → 解析 → 分块 → embedding → pgvector
//   - 纯文本模式：直接对 Content 分块（手动创建的文章）
func (p *Processor) processTask(task ProcessTask) {
	ctx := p.ctx
	articleID := task.ArticleID

	var content string
	p.updateStatus(task, "parsing", "")
	if task.Bucket != "" && task.Key != "" {
		// MinIO 路径模式：下载原始文件并解析
		if p.storage == nil {
			p.updateStatus(task, "failed", "StorageClient 未初始化，无法下载 MinIO 文件")
			return
		}
		reader, err := p.storage.Download(ctx, task.Bucket, task.Key)
		if err != nil {
			p.updateStatus(task, "failed", fmt.Sprintf("从 MinIO 下载文件失败: %v", err))
			return
		}
		defer reader.Close()

		fileType := task.FileType
		if fileType == "" {
			// 从 key 后缀推断文件类型
			if idx := strings.LastIndex(task.Key, "."); idx >= 0 {
				fileType = task.Key[idx+1:]
			}
		}

		parsed, err := p.parser.Parse(reader, fileType)
		if err != nil {
			p.updateStatus(task, "failed", fmt.Sprintf("文档解析失败: %v", err))
			return
		}
		if strings.TrimSpace(parsed) == "" {
			p.updateStatus(task, "failed", "文档内容为空")
			return
		}
		content = parsed
	} else {
		// 纯文本模式：直接使用 Content
		content = task.Content
	}

	// 阶段 2: 分块
	p.updateStatus(task, "chunking", "")
	chunks := p.chunker.Split(content)
	if len(chunks) == 0 {
		p.updateStatus(task, "failed", "分块结果为空")
		return
	}
	// 写回字数和分块数
	if task.OnMetrics != nil {
		task.OnMetrics(articleID, len([]rune(content)), len(chunks))
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
			// TODO(rag/processor): EmbeddingModel 应优先从 KB 配置读取，而非硬编码空字符串。
			// 空字符串回退到全局默认模型，当 KB 绑定了独立 LLM 配置时可能出现模型不匹配。
			EmbeddingModel:  "",
			VectorDimension: len(vectors[i]),
		}
	}

	// 阶段 3: 写入 pgvector
	// TODO(rag/processor): 这里重复 updateStatus("indexing")，前端进度可能收到两次相同阶段。
	// 保留一个即可，另一个可改为更细的 batch 写入进度。
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
