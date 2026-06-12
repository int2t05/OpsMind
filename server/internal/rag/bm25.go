// Package rag 实现自建 RAG 检索引擎。
//
// bm25.go 实现 BM25 倒排索引 + 中文分词检索。
//
// 架构设计：
//   - Segmenter 接口：抽象分词器，方便替换不同分词库
//   - BM25Index：单知识库的倒排索引 + Okapi BM25 评分
//   - BM25Retriever：管理多知识库索引的生命周期（懒加载 + TTL）
//
// 为什么使用 BM25 而非 TF-IDF：
// BM25 的文档长度归一化（参数 b）比 TF-IDF 更健壮，
// 长文档不会因为包含更多词条而获得不公平的高分。
// Okapi 变体是学术界和工业界公认的稀疏检索基线。
//
// 为什么懒加载 + TTL（ADR-V2-004）：
// BM25 索引构建需要遍历知识库全量分块并分词，
// 对于大知识库可能耗时数秒。懒加载（首次检索时构建）
// 避免启动时全量加载，TTL 保证索引定期刷新。
package rag

import (
	"context"
	"math"
	"sort"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/go-ego/gse"
)

// =============================================================================
// Segmenter 接口 + gse 实现
// =============================================================================

// Segmenter 定义分词器接口。
type Segmenter interface {
	// Segment 对文本分词，返回 token 列表。
	Segment(text string) []string
	// Close 释放分词器资源。
	Close()
}

// GseSegmenter 使用 gse 库的中文分词器。
//
// 为什么选择 gse：
// gse 是纯 Go 实现，无 CGO 依赖，交叉编译友好。
// 支持中文、英文、日文等多语言分词，
// 内置词典覆盖常见中文词汇，MVP 阶段无需额外训练。
//
// 线程安全：Segment 方法通过 mu 保护 gse 内部状态（HMM 标记器在 Cut 时修改内部结构）。
type GseSegmenter struct {
	seg gse.Segmenter
	mu  sync.Mutex
}

// NewGseSegmenter 创建 gse 分词器实例。
//
// 加载内置中文词典，首次调用约需 100-200ms 加载词典文件。
func NewGseSegmenter() *GseSegmenter {
	s := &GseSegmenter{}
	// 加载内置词典
	_ = s.seg.LoadDict() // gse 内置词典加载失败不影响使用（回退到字符级）
	return s
}

// Segment 对文本分词（线程安全）。
func (s *GseSegmenter) Segment(text string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.seg.Cut(text, true) // true = 启用 HMM 分词
}

// Close 释放分词器资源（gse 无显式释放需求）。
func (s *GseSegmenter) Close() {}

// =============================================================================
// BM25 常量
// =============================================================================

const (
	// bm25K1 BM25 词频饱和参数。
	// k1=1.5 是 Okapi 论文推荐值，平衡了词频的饱和速度和区分度。
	bm25K1 = 1.5

	// bm25B BM25 文档长度归一化参数。
	// b=0.75 是 Okapi 论文推荐值，给中等长度文档更多权重。
	bm25B = 0.75
)

// =============================================================================
// BM25Document — 索引文档
// =============================================================================

// BM25Document 待索引的文档（知识分块）。
type BM25Document struct {
	ChunkID    int64  `json:"chunk_id"`
	ArticleID  int64  `json:"article_id"`
	KBID       int64  `json:"kb_id"`
	Content    string `json:"content"`
	ChunkIndex int    `json:"chunk_index"`
}

// =============================================================================
// BM25Index — 倒排索引 + 打分
// =============================================================================

// bm25Posting 倒排索引的单个命中记录。
type bm25Posting struct {
	ChunkID int64 // 分块 ID
	TermFreq int  // 该词在该文档中的出现次数
}

// BM25Index 单知识库的倒排索引。
//
// 为什么用 map 而非 B-Tree：
// MVP 阶段知识库规模有限（< 10万篇），
// Go 原生 map 的 O(1) 查询性能足够，
// 后续如数据量增长可考虑 B-Tree 或 Roaring Bitmap 优化。
//
// TODO: 知识库超 10 万篇后 map[string][]bm25Posting 内存压力大。
// 方案：1) 定期清除低频词（TF < 2）；2) Roaring Bitmap 压缩 posting list；
// 3) 索引持久化到磁盘避免每次启动全量重建。
type BM25Index struct {
	// 倒排索引：token → posting list
	inverted map[string][]bm25Posting
	// 文档长度：ChunkID → rune count
	docLens map[int64]int
	// 文档元数据：ChunkID → {ArticleID, ChunkIndex, Content}
	docMeta map[int64]BM25Document
	// 平均文档长度
	avgdl float64
	// 文档总数
	docCount int
}

// newBM25Index 创建空的 BM25 索引。
func newBM25Index() *BM25Index {
	return &BM25Index{
		inverted: make(map[string][]bm25Posting),
		docLens:  make(map[int64]int),
		docMeta:  make(map[int64]BM25Document),
	}
}

// =============================================================================
// BM25Retriever — 多知识库管理
// =============================================================================

// BM25Retriever 实现 Retriever 接口，管理多知识库的 BM25 索引。
//
// 每个知识库有独立的索引，通过懒加载按需构建。
// 索引在 TTL 后自动失效，下次检索时重建。
type BM25Retriever struct {
	segmenter Segmenter
	ttl       time.Duration

	mu      sync.RWMutex
	indexes map[int64]*bm25Entry // kbID → 索引条目
}

// bm25Entry 单个知识库的 BM25 索引及其元数据。
type bm25Entry struct {
	index     *BM25Index
	documents []BM25Document // 保存原始文档用于重建
	builtAt   time.Time
}

// NewBM25Retriever 创建 BM25Retriever 实例。
//
// ttl 为索引自动过期时间。设 0 禁用自动过期，索引永久有效。
func NewBM25Retriever(seg Segmenter, ttl time.Duration) *BM25Retriever {
	r := &BM25Retriever{
		segmenter: seg,
		ttl:       ttl,
		indexes:   make(map[int64]*bm25Entry),
	}
	return r
}

// BuildIndex 为知识库构建（或重建）BM25 索引。
//
// 此方法会替换该知识库的全部旧索引数据。
func (r *BM25Retriever) BuildIndex(kbID int64, docs []BM25Document) {
	idx := r.buildIndex(docs)

	r.mu.Lock()
	r.indexes[kbID] = &bm25Entry{
		index:     idx,
		documents: docs,
		builtAt:   time.Now(),
	}
	r.mu.Unlock()
}

// buildIndex 构建 BM25 倒排索引的内部实现。
func (r *BM25Retriever) buildIndex(docs []BM25Document) *BM25Index {
	idx := newBM25Index()
	if len(docs) == 0 {
		return idx
	}

	idx.docCount = len(docs)
	var totalLen int

	for _, doc := range docs {
		idx.docMeta[doc.ChunkID] = doc
		tokens := r.segmenter.Segment(doc.Content)
		docLen := utf8.RuneCountInString(doc.Content)
		idx.docLens[doc.ChunkID] = docLen
		totalLen += docLen

		// 统计词频，按 ChunkID 分组
		tf := make(map[string]int)
		for _, tok := range tokens {
			tf[tok]++
		}

		// 写入倒排索引
		for tok, freq := range tf {
			idx.inverted[tok] = append(idx.inverted[tok], bm25Posting{
				ChunkID:  doc.ChunkID,
				TermFreq: freq,
			})
		}
	}

	if idx.docCount > 0 {
		idx.avgdl = float64(totalLen) / float64(idx.docCount)
	}
	return idx
}

// Retrieve 执行 BM25 检索，返回 topK 个结果。
//
// 如果知识库索引不存在或已过期，返回空结果（不报错）。
// TODO: ctx 参数未使用——分词和评分操作不支持 context 取消。
// 大知识库检索可能耗时数秒，应通过 context 支持超时和取消。
func (r *BM25Retriever) Retrieve(ctx context.Context, query string, kbID int64, topK int) ([]RetrievalResult, error) {
	r.mu.RLock()
	entry, exists := r.indexes[kbID]
	r.mu.RUnlock()

	if !exists || entry == nil || entry.index.docCount == 0 {
		return nil, nil
	}

	// TTL 检查：过期时自动重建索引（使用存储的文档副本）
	if r.ttl > 0 && time.Since(entry.builtAt) > r.ttl {
		// 升级为写锁并重建（RLock 已在入口处释放，直接获取写锁即可；原 r.mu.RUnlock() 为重复释放 bug）
		r.mu.Lock()
		// 双重检查（其他 goroutine 可能已重建）
		if e, stillExists := r.indexes[kbID]; stillExists && time.Since(e.builtAt) > r.ttl {
			if len(e.documents) > 0 {
				idx := r.buildIndex(e.documents)
				r.indexes[kbID] = &bm25Entry{
					index:     idx,
					documents: e.documents,
					builtAt:   time.Now(),
				}
			}
		}
		r.mu.Unlock()
		r.mu.RLock()
		defer r.mu.RUnlock() // 确保函数退出时释放读锁（此前遗漏，导致 TTL 过期后死锁）
		// 重新获取（可能已更新）
		entry = r.indexes[kbID]
		if entry == nil || entry.index.docCount == 0 {
			return nil, nil
		}
	}

	// 对查询分词
	queryTokens := r.segmenter.Segment(query)

	// 计算每个文档的 BM25 分数
	scores := r.scoreQuery(entry.index, queryTokens)

	// 按分数降序排列
	type scoredDoc struct {
		chunkID int64
		score   float64
	}
	var sorted []scoredDoc
	for chunkID, score := range scores {
		if score > 0 {
			sorted = append(sorted, scoredDoc{chunkID: chunkID, score: score})
		}
	}

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].score > sorted[j].score
	})

	// 截取 topK
	if len(sorted) > topK {
		sorted = sorted[:topK]
	}

	// 转换为 RetrievalResult
	results := make([]RetrievalResult, 0, len(sorted))
	for _, s := range sorted {
		meta := entry.index.docMeta[s.chunkID]
		results = append(results, RetrievalResult{
			ChunkID:    s.chunkID,
			ArticleID:  meta.ArticleID,
			Content:    meta.Content,
			Score:      s.score,
			Source:     "bm25",
			ChunkIndex: meta.ChunkIndex,
		})
	}
	return results, nil
}

// scoreQuery 计算查询与所有文档的 BM25 分数。
func (r *BM25Retriever) scoreQuery(idx *BM25Index, queryTokens []string) map[int64]float64 {
	scores := make(map[int64]float64)
	N := float64(idx.docCount)

	// 对每个查询 token 累加分数
	seenTokens := make(map[string]bool)
	for _, token := range queryTokens {
		if seenTokens[token] {
			continue // 查询中去重
		}
		seenTokens[token] = true

		postings, exists := idx.inverted[token]
		if !exists {
			continue
		}

		// IDF = ln((N - n + 0.5) / (n + 0.5) + 1)
		n := float64(len(postings))
		idf := math.Log((N-n+0.5)/(n+0.5) + 1.0)

		for _, p := range postings {
			docLen := float64(idx.docLens[p.ChunkID])

			// 防止 avgdl=0 导致除零（所有文档内容为空时）
			avgdl := idx.avgdl
			if avgdl == 0 {
				avgdl = 1 // 默认长度为 1，避免除零
			}

			// tf_norm = f * (k1 + 1) / (f + k1 * (1 - b + b * dl/avgdl))
			f := float64(p.TermFreq)
			normFactor := bm25K1 * (1 - bm25B + bm25B*docLen/avgdl)
			tfNorm := (f * (bm25K1 + 1)) / (f + normFactor)

			scores[p.ChunkID] += idf * tfNorm
		}
	}
	return scores
}
