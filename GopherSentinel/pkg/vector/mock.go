package vector

import (
	"context"
	"sync"
)

// MockVectorStore 模拟向量存储，用于离线测试
type MockVectorStore struct {
	mu      sync.RWMutex
	points  map[string]*mockPoint
}

type mockPoint struct {
	ID       string
	Vector   []float32
	Payload  map[string]interface{}
	Content  string
}

var (
	_ VectorStore = (*MockVectorStore)(nil)
)

// NewMockVectorStore 创建模拟向量存储
func NewMockVectorStore() *MockVectorStore {
	return &MockVectorStore{
		points: make(map[string]*mockPoint),
	}
}

// IndexDocument 索引文档
func (m *MockVectorStore) IndexDocument(ctx context.Context, content string, vector []float32, metadata map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := GeneratePointID()
	m.points[id] = &mockPoint{
		ID:      id,
		Vector:  vector,
		Content: content,
		Payload: metadata,
	}
	return nil
}

// Search 搜索相似向量（Mock 实现）
func (m *MockVectorStore) Search(ctx context.Context, queryVector []float32, limit int, filter *Filter) ([]SearchResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make([]SearchResult, 0, len(m.points))
	for id, p := range m.points {
		// 模拟相似度计算
		score := float64(0.8)
		if len(queryVector) > 0 && len(p.Vector) > 0 {
			// 简单模拟：用向量长度生成伪分数
			score = 0.6 + float64(len(queryVector)%4)*0.1
		}

		results = append(results, SearchResult{
			ID:      id,
			Score:   float32(score),
			Payload: p.Payload,
			Content: p.Content,
		})
	}

	// 限制返回数量
	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// Count 返回存储的向量数量
func (m *MockVectorStore) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.points)
}

// Clear 清空所有数据
func (m *MockVectorStore) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.points = make(map[string]*mockPoint)
}

// VectorStore 接口 - 统一 Qdrant 和 Mock
type VectorStore interface {
	IndexDocument(ctx context.Context, content string, vector []float32, metadata map[string]interface{}) error
	Search(ctx context.Context, queryVector []float32, limit int, filter *Filter) ([]SearchResult, error)
}
