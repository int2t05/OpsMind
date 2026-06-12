// Package rag 实现自建 RAG 检索引擎。
//
// rag 包是 OpsMind v2 的核心领域模块，包含：
//   - 文本分块（RecursiveCharacterTextSplitter）
//   - BM25 倒排索引 + 中文分词（gse）
//   - 向量 Embedding 批量生成
//   - RRF 混合融合（BM25 + 向量）
//   - 文档解析（PDF/DOCX/MD/TXT）
//   - RAG 管道编排（查询改写→多路检索→混合检索→重排序）
//   - 文档异步处理管线（goroutine pool）
//
// rag 包不依赖 HTTP 层（Handler/Service/Repository），
// 仅依赖 adapter 包中的接口（LLMClient/EmbeddingClient/VectorStore）。
// 这样设计的目的是保持 RAG 引擎的可测试性和可替换性。
package rag

import "context"

// =============================================================================
// Retriever 接口
// =============================================================================

// Retriever 定义检索引擎接口。
// BM25Retriever 和 VectorRetriever 都实现此接口，
// Pipeline 通过此接口统一调度多路检索。
type Retriever interface {
	// Retrieve 执行检索，返回 topK 个最相关的结果。
	Retrieve(ctx context.Context, query string, kbID int64, topK int) ([]RetrievalResult, error)
}

// =============================================================================
// RAGOptions — 检索参数
// =============================================================================

// RAGOptions 控制 RAG 管道各步骤的开关和参数。
//
// 所有字段均为可选，零值表示使用默认配置（全部开启，TopK=5）。
type RAGOptions struct {
	// TODO(rag/types): bool 零值无法表达“未传则默认 true”和“用户显式 false”的区别。
	// 请求层需要用 *bool 或先 DefaultRAGOptions 再覆盖，否则默认选项容易被零值关闭。
	TopK          int  `json:"top_k"`          // 最终返回的检索结果数，默认 5
	QueryRewrite  bool `json:"query_rewrite"`  // 是否启用查询改写
	MultiRoute    bool `json:"multi_route"`    // 是否启用多路检索（生成子查询）
	Hybrid        bool `json:"hybrid"`         // 是否启用 BM25+向量混合检索
	Rerank        bool `json:"rerank"`         // 是否启用重排序
	RouteCount    int  `json:"route_count"`    // 多路检索生成的子查询数，默认 3
	RerankCount   int  `json:"rerank_count"`   // 送入重排序的候选数，默认 topK*3
}

// DefaultRAGOptions 返回默认的 RAG 检索选项。
func DefaultRAGOptions() RAGOptions {
	// TODO(rag/types): 增加 Normalize/Validate 方法统一处理 TopK、RouteCount、RerankCount 的范围。
	// 现在各调用方可能各自补默认值，行为容易不一致。
	return RAGOptions{
		TopK:         5,
		QueryRewrite: true,
		MultiRoute:   true,
		Hybrid:       true,
		Rerank:       true,
		RouteCount:   3,
		RerankCount:  15,
	}
}

// =============================================================================
// 检索结果类型
// =============================================================================

// RetrievalResult 单条检索命中结果。
type RetrievalResult struct {
	ChunkID    int64   `json:"chunk_id"`    // knowledge_chunks.id
	ArticleID  int64   `json:"article_id"`  // knowledge_articles.id
	Content    string  `json:"content"`     // 分块文本内容
	Score      float64 `json:"score"`       // 相关度分数（归一化到 [0,1]）
	Source     string  `json:"source"`      // 检索来源："vector" | "bm25" | "hybrid"
	ChunkIndex int     `json:"chunk_index"` // 分块序号
}

// =============================================================================
// 管道指标类型
// =============================================================================

// RAGResult RAG 管道执行最终结果。
type RAGResult struct {
	Chunks  []RetrievalResult `json:"chunks"`  // 检索到的分块列表（按分数降序）
	Metrics PipelineMetrics   `json:"metrics"` // 管道各步骤耗时与状态
}

// PipelineMetrics 管道各步骤的执行指标。
type PipelineMetrics struct {
	Steps           []StepMetric `json:"steps"`             // 各步骤指标（按执行顺序）
	TotalDurationMS int64        `json:"total_duration_ms"` // 管道总耗时（毫秒）
}

// StepMetric 单个管道步骤的执行指标。
type StepMetric struct {
	StepID     string `json:"step_id"`     // 步骤标识：query_rewrite / multi_route / hybrid_retrieve / rerank / llm_generate
	Label      string `json:"label"`       // 步骤显示名称（中文）
	DurationMS int64  `json:"duration_ms"` // 步骤耗时（毫秒）
	Success    bool   `json:"success"`     // 是否成功
	Error      string `json:"error"`       // 失败时的错误信息
}

// =============================================================================
// SSE 步骤事件
// =============================================================================

// StepEvent SSE 流式响应中的管道步骤进度事件。
type StepEvent struct {
	Type  string `json:"type"`  // "step"
	ID    string `json:"id"`    // 步骤标识
	Label string `json:"label"` // 步骤显示名称
}

// =============================================================================
// 回调类型
// =============================================================================

// StepCallback 管道步骤回调函数。
// Pipeline 在进入每个步骤时调用此回调，用于向前端发送 SSE 步骤事件。
type StepCallback func(event StepEvent)
