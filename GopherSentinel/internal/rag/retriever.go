package rag

import (
	"context"
	"sync"
)

// HybridRetriever 混合检索器
type HybridRetriever struct {
	vectorRetriever VectorRetriever
	bm25Retriever  BM25Retriever
	fusionStrategy FusionStrategy
}

// VectorRetriever 向量检索器
type VectorRetriever interface {
	Search(ctx context.Context, query string, topK int) ([]SearchResult, error)
}

// BM25Retriever BM25 检索器
type BM25Retriever interface {
	Search(ctx context.Context, query string, topK int) ([]SearchResult, error)
}

// SearchResult 搜索结果
type SearchResult struct {
	ID        string
	Content   string
	Score     float64
	Metadata  map[string]interface{}
}

// FusionStrategy 融合策略
type FusionStrategy interface {
	Fuse(results [][]SearchResult, weights []float64) []SearchResult
}

// RRFusion RRF (Reciprocal Rank Fusion) 融合
type RRFusion struct {
	k int // RRF 参数，通常设为 60
}

// NewRRFusion 创建 RRF 融合
func NewRRFusion(k int) *RRFusion {
	if k <= 0 {
		k = 60
	}
	return &RRFusion{k: k}
}

// Fuse 实现 RRF 融合
func (r *RRFusion) Fuse(results [][]SearchResult, weights []float64) []SearchResult {
	if len(results) == 0 {
		return []SearchResult{}
	}

	// 计算总结果数
	totalResults := make(map[string]*SearchResult)
	for _, resultList := range results {
		for i, result := range resultList {
			if _, exists := totalResults[result.ID]; !exists {
				totalResults[result.ID] = &SearchResult{
					ID:       result.ID,
					Content:  result.Content,
					Metadata: result.Metadata,
					Score:    0,
				}
			}
			// 计算 RRF 分数
			weight := 1.0
			if len(weights) > i {
				weight = weights[i]
			}
			totalResults[result.ID].Score += weight / float64(r.k+len(resultList))
		}
	}

	// 排序
	fused := make([]SearchResult, 0, len(totalResults))
	for _, result := range totalResults {
		fused = append(fused, *result)
	}

	// 简单排序（按分数降序）
	for i := 0; i < len(fused)-1; i++ {
		for j := i + 1; j < len(fused); j++ {
			if fused[j].Score > fused[i].Score {
				fused[i], fused[j] = fused[j], fused[i]
			}
		}
	}

	return fused
}

// NewHybridRetriever 创建混合检索器
func NewHybridRetriever(vector VectorRetriever, bm25 BM25Retriever) *HybridRetriever {
	return &HybridRetriever{
		vectorRetriever: vector,
		bm25Retriever:  bm25,
		fusionStrategy: NewRRFusion(60),
	}
}

// Search 执行混合检索
func (r *HybridRetriever) Search(ctx context.Context, query string, topK int) ([]SearchResult, error) {
	var wg sync.WaitGroup
	var vectorResults []SearchResult
	var bm25Results []SearchResult
	var vectorErr, bm25Err error

	// 并行执行向量检索和 BM25 检索
	wg.Add(2)

	go func() {
		defer wg.Done()
		vectorResults, vectorErr = r.vectorRetriever.Search(ctx, query, topK)
	}()

	go func() {
		defer wg.Done()
		bm25Results, bm25Err = r.bm25Retriever.Search(ctx, query, topK)
	}()

	wg.Wait()

	// 处理错误
	if vectorErr != nil && bm25Err != nil {
		return nil, vectorErr
	}

	// 融合结果
	// 向量检索权重 0.7，BM25 权重 0.3
	weights := []float64{0.7, 0.3}
	fused := r.fusionStrategy.Fuse([][]SearchResult{vectorResults, bm25Results}, weights)

	// 返回 Top K
	if len(fused) > topK {
		return fused[:topK], nil
	}
	return fused, nil
}
