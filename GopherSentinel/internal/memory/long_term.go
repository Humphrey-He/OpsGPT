package memory

import (
	"context"
)

// LongTermMemory 长期记忆
type LongTermMemory struct {
	vectorStore VectorStore
	summarizer Summarizer
}

// NewLongTermMemory 创建长期记忆
func NewLongTermMemory(vectorStore VectorStore, summarizer Summarizer) *LongTermMemory {
	return &LongTermMemory{
		vectorStore: vectorStore,
		summarizer: summarizer,
	}
}

// Store 存储记忆
func (m *LongTermMemory) Store(ctx context.Context, memory *Memory) error {
	// 1. 总结记忆（压缩存储）
	if m.summarizer != nil {
		summary, err := m.summarizer.Summarize(ctx, memory.Content)
		if err != nil {
			return err
		}
		memory.Content = summary
	}

	// 2. 生成向量
	embedding, err := m.vectorStore.Embed(ctx, memory.Content)
	if err != nil {
		return err
	}

	// 3. 存入向量库
	return m.vectorStore.Store(ctx, memory, embedding)
}

// Recall 检索记忆
func (m *LongTermMemory) Recall(ctx context.Context, query string, limit int) ([]*Memory, error) {
	// 1. 生成查询向量
	embedding, err := m.vectorStore.Embed(ctx, query)
	if err != nil {
		return nil, err
	}

	// 2. 检索相似记忆
	return m.vectorStore.Search(ctx, embedding, limit)
}

// VectorStore 向量存储接口
type VectorStore interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	Store(ctx context.Context, memory *Memory, embedding []float32) error
	Search(ctx context.Context, query []float32, limit int) ([]*Memory, error)
}

// Summarizer 总结器接口
type Summarizer interface {
	Summarize(ctx context.Context, content string) (string, error)
}

// Memory 记忆结构
type Memory struct {
	ID        string
	Content   string
	Timestamp int64
	Metadata  map[string]interface{}
}
