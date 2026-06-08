package memory

import (
	"context"
	"time"
)

// ShortTermMemory 短期记忆（Redis）
type ShortTermMemory struct {
	// 实际实现需要 Redis 客户端
}

// Append 添加到短期记忆
func (m *ShortTermMemory) Append(ctx context.Context, event *MemoryEvent) error {
	// TODO: 实现 Redis 存储
	return nil
}

// GetRecent 获取最近的记忆
func (m *ShortTermMemory) GetRecent(ctx context.Context, n int) ([]*MemoryEvent, error) {
	// TODO: 实现 Redis 查询
	return nil, nil
}

// NewShortTermMemory 创建短期记忆
func NewShortTermMemory() *ShortTermMemory {
	return &ShortTermMemory{}
}

// MemorySystem 完整记忆系统
type MemorySystem struct {
	working   *WorkingMemory
	shortTerm *ShortTermMemory
	longTerm  *LongTermMemory
}

// NewMemorySystem 创建完整记忆系统
func NewMemorySystem(
	working *WorkingMemory,
	shortTerm *ShortTermMemory,
	longTerm *LongTermMemory,
) *MemorySystem {
	return &MemorySystem{
		working:   working,
		shortTerm: shortTerm,
		longTerm:  longTerm,
	}
}

// Remember 记住内容
func (m *MemorySystem) Remember(ctx context.Context, event *MemoryEvent) error {
	// 1. 写入工作记忆
	m.working.Add(event)

	// 2. 异步写入短期记忆
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		m.shortTerm.Append(ctx, event)
	}()

	return nil
}

// Recall 召回记忆
func (m *MemorySystem) Recall(ctx context.Context, query string) (*MemoryContext, error) {
	// 1. 检索长期记忆
	longTermResults, _ := m.longTerm.Recall(ctx, query, 5)

	// 2. 获取短期记忆
	shortTermResults, _ := m.shortTerm.GetRecent(ctx, 5)

	return &MemoryContext{
		LongTerm:  longTermResults,
		ShortTerm: shortTermResults,
	}, nil
}

// MemoryEvent 记忆事件
type MemoryEvent struct {
	Type      string
	Content   string
	Timestamp time.Time
	SessionID string
}

// MemoryContext 记忆上下文
type MemoryContext struct {
	LongTerm  []*Memory
	ShortTerm []*MemoryEvent
}

// WorkingMemory 工作记忆（内存）
type WorkingMemory struct {
	messages []*MemoryEvent
	maxSize  int
}

// NewWorkingMemory 创建工作记忆
func NewWorkingMemory(maxSize int) *WorkingMemory {
	return &WorkingMemory{
		messages: make([]*MemoryEvent, 0, maxSize),
		maxSize:  maxSize,
	}
}

// Add 添加记忆
func (w *WorkingMemory) Add(event *MemoryEvent) {
	w.messages = append(w.messages, event)
	if len(w.messages) > w.maxSize {
		w.messages = w.messages[1:]
	}
}

// GetRecent 获取最近的记忆
func (w *WorkingMemory) GetRecent(n int) []*MemoryEvent {
	if n > len(w.messages) {
		n = len(w.messages)
	}
	result := make([]*MemoryEvent, n)
	copy(result, w.messages[len(w.messages)-n:])
	return result
}

// Clear 清除记忆
func (w *WorkingMemory) Clear() {
	w.messages = w.messages[:0]
}
