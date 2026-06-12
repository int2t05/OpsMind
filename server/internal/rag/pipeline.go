// Package rag 实现自建 RAG 检索引擎。
//
// pipeline.go 实现 RAG 管道编排器（Pipeline）。
//
// 管道执行流程（按序）：
//  1. QueryRewrite    — LLM 改写口语化查询
//  2. MultiRoute      — LLM 生成多路子查询
//  3. HybridRetrieve  — 向量检索 + BM25 → RRF 融合（可选）
//  4. Rerank          — LLM 重排序候选
//
// 每步失败按降级矩阵处理：
//   - 查询改写/多路路由/重排序失败 → 降级继续（不阻塞）
//   - 向量检索失败（无 BM25 降级）→ 返回错误（核心路径不可降级）
//   - BM25 失败 → 仅用向量结果
//
// 设计决策（ADR-V2-001）：
// Pipeline 通过 Retriever 接口调用向量检索和 BM25，
// 不直接依赖 VectorStore 或 BM25Retriever 的具体实现。
// 这样 BM25 懒加载和 TTL 对 Pipeline 完全透明。
package rag

import (
	"context"
	"fmt"
	"time"

	"opsmind/internal/adapter"
)

// =============================================================================
// Pipeline — RAG 管道编排器
// =============================================================================

// Pipeline 组装 RAG 管道各步骤，按序执行检索流程。
type Pipeline struct {
	vectorRetriever Retriever         // 向量检索器（通过 Embedder+VectorStore 实现，不可为 nil）
	bm25Retriever   Retriever         // BM25 检索器（可为 nil，表示不启用 BM25）
	llmClient       adapter.LLMClient // LLM 客户端（查询改写/多路/重排序）
	embedder        *Embedder         // 向量嵌入器
}

// NewPipeline 创建 Pipeline 实例。
//
// vectorRet 不可为 nil。bm25Ret 可以为 nil（不启用 BM25 混合检索）。
// onStep 回调通过 Execute 的 onStep 参数传入，不在此处存储（避免闭包陷阱）。
func NewPipeline(vectorRet, bm25Ret Retriever, llm adapter.LLMClient, emb *Embedder) *Pipeline {
	return &Pipeline{
		vectorRetriever: vectorRet,
		bm25Retriever:   bm25Ret,
		llmClient:       llm,
		embedder:        emb,
	}
}

// =============================================================================
// Execute — 管道主入口
// =============================================================================

// Execute 执行 RAG 管道，返回检索结果和管道指标。
//
// kbID 为知识库 ID，opts 控制各步骤开关和参数。
func (p *Pipeline) Execute(ctx context.Context, query string, kbID int64, opts RAGOptions, onStep StepCallback) (*RAGResult, error) {
	start := time.Now()
	metrics := PipelineMetrics{}
	var steps []StepMetric

	// 追踪步骤用时
	track := func(stepID, label string, fn func() error) error {
		stepStart := time.Now()
		if onStep != nil {
			onStep(StepEvent{Type: "step", ID: stepID, Label: label})
		}
		err := fn()
		dur := time.Since(stepStart).Milliseconds()
		sm := StepMetric{
			StepID:     stepID,
			Label:      label,
			DurationMS: dur,
			Success:    err == nil,
		}
		if err != nil {
			sm.Error = err.Error()
		}
		steps = append(steps, sm)
		return err
	}

	// ─── Step 1: 查询改写 ───
	rewrittenQuery := query
	if opts.QueryRewrite {
		_ = track("query_rewrite", "查询改写", func() error {
			// TODO: history 始终传 nil，QueryRewrite 的上下文消歧功能从未生效。
			// 应由调用方通过 RAGOptions 传入最近 N 轮对话历史。
			rw, err := QueryRewrite(ctx, p.llmClient, query, nil)
			if err != nil {
				return err
			}
			rewrittenQuery = rw
			return nil
		})
	}

	// ─── Step 2: 多路检索 ───
	routes := []string{rewrittenQuery}
	if opts.MultiRoute && opts.RouteCount > 1 {
		_ = track("multi_route", "多路检索", func() error {
			rts, err := MultiRoute(ctx, p.llmClient, rewrittenQuery, opts.RouteCount)
			if err != nil {
				return err
			}
			routes = rts
			return nil
		})
	}

	// ─── Step 3: 混合检索 ───
	// Retriever 接口内部处理 embedding 生成，Pipeline 不直接调用 Embedder。
	var allChunks []RetrievalResult
	if opts.Hybrid && p.bm25Retriever != nil {
		var vectorResults, bm25Results []RetrievalResult

		// 3a: 向量检索（核心路径—失败不可降级）
		err := track("vector_retrieve", "向量检索", func() error {
			for _, route := range routes {
				results, err := p.vectorRetriever.Retrieve(ctx, route, kbID, opts.TopK)
				if err != nil {
					return err
				}
				vectorResults = append(vectorResults, results...)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("向量检索失败: %w", err)
		}

		// 3b: BM25 检索（降级—失败不阻塞）
		_ = track("bm25_retrieve", "BM25 检索", func() error {
			for _, route := range routes {
				results, err := p.bm25Retriever.Retrieve(ctx, route, kbID, opts.TopK)
				if err != nil {
					return err
				}
				bm25Results = append(bm25Results, results...)
			}
			return nil
		})

		// 3c: RRF 融合
		fuseErr := track("hybrid_fuse", "混合融合", func() error {
			allChunks = HybridFuse(vectorResults, bm25Results, 60, opts.RerankCount)
			if len(allChunks) == 0 {
				return fmt.Errorf("混合融合后无结果")
			}
			return nil
		})
		if fuseErr != nil && len(vectorResults) == 0 && len(bm25Results) == 0 {
			// 两路都为空时才传播错误（否则融合失败不影响已有结果）
			return nil, fmt.Errorf("混合检索无结果: %w", fuseErr)
		}
		if len(allChunks) == 0 && len(vectorResults) > 0 {
			allChunks = vectorResults
		} else if len(allChunks) == 0 && len(bm25Results) > 0 {
			allChunks = bm25Results
		}
	} else {
		// 纯向量模式
		err := track("vector_retrieve", "向量检索", func() error {
			for _, route := range routes {
				results, err := p.vectorRetriever.Retrieve(ctx, route, kbID, opts.TopK)
				if err != nil {
					return err
				}
				allChunks = append(allChunks, results...)
			}
			if len(allChunks) == 0 {
				return fmt.Errorf("向量检索无结果")
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("向量检索失败: %w", err)
		}
	}

	// ─── Step 4: 重排序 ───
	if opts.Rerank && len(allChunks) > 1 {
		_ = track("rerank", "重排序", func() error {
			reranked, err := Rerank(ctx, p.llmClient, rewrittenQuery, allChunks)
			if err != nil {
				return err
			}
			allChunks = reranked
			return nil
		})
	}

	// 截取 TopK
	//
	// TODO: 应校验 RerankCount >= TopK，否则重排序候选池小于目标数，
	// 重排序后截断 TopK 形同虚设（候选池本身不足）。
	if len(allChunks) > opts.TopK {
		allChunks = allChunks[:opts.TopK]
	}

	metrics.Steps = steps
	metrics.TotalDurationMS = time.Since(start).Milliseconds()

	return &RAGResult{
		Chunks:  allChunks,
		Metrics: metrics,
	}, nil
}
