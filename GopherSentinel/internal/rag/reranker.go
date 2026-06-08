package rag

import (
	"context"
)

// Reranker 精排器
type Reranker interface {
	Rerank(ctx context.Context, query string, results []SearchResult) ([]SearchResult, error)
}

// BGEReranker BGE-Reranker 实现
type BGEReranker struct {
	baseURL string
	model   string
}

// NewBGEReranker 创建 BGE-Reranker
func NewBGEReranker(baseURL, model string) *BGEReranker {
	return &BGEReranker{
		baseURL: baseURL,
		model:   model,
	}
}

// Rerank 执行精排
func (r *BGEReranker) Rerank(ctx context.Context, query string, results []SearchResult) ([]SearchResult, error) {
	if len(results) == 0 {
		return results, nil
	}

	// 模拟 BGE-Reranker 调用
	// 实际项目中需要调用 BGE-Reranker HTTP API 或本地模型

	// 返回原始结果（模拟重排序）
	// 真实实现中会按相关性重新排序
	return results, nil
}

// ContextCompressor 上下文压缩器
type ContextCompressor struct {
	maxTokens int
}

// NewContextCompressor 创建上下文压缩器
func NewContextCompressor(maxTokens int) *ContextCompressor {
	if maxTokens <= 0 {
		maxTokens = 4000
	}
	return &ContextCompressor{maxTokens: maxTokens}
}

// Compress 压缩上下文
func (c *ContextCompressor) Compress(ctx context.Context, docs []SearchResult) (string, error) {
	// 简单实现：按分数排序后拼接
	var content string

	for i, doc := range docs {
		if i > 0 {
			content += "\n---\n"
		}
		content += doc.Content

		// 简单估算 token（实际应该用 tokenizer）
		estimatedTokens := len(content) / 4 // 粗略估算
		if estimatedTokens > c.maxTokens {
			break
		}
	}

	return content, nil
}
