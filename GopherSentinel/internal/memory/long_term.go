package memory

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"GopherSentinel/pkg/llm"
	"GopherSentinel/pkg/vector"
)

// LongTermMemory 长期记忆（使用向量库存储）
type LongTermMemory struct {
	vectorStore *vector.QdrantClient
	llmClient  *llm.OllamaClient
	config     LongTermConfig
}

// LongTermConfig 长期记忆配置
type LongTermConfig struct {
	TopK            int
	ScoreThreshold  float32
	MaxContentLen   int
	SummaryEnabled  bool
}

// DefaultLongTermConfig 默认配置
func DefaultLongTermConfig() LongTermConfig {
	return LongTermConfig{
		TopK:           5,
		ScoreThreshold: 0.7,
		MaxContentLen: 2000,
		SummaryEnabled: true,
	}
}

// NewLongTermMemory 创建长期记忆
func NewLongTermMemory(vectorStore *vector.QdrantClient, llmClient *llm.OllamaClient) *LongTermMemory {
	return &LongTermMemory{
		vectorStore: vectorStore,
		llmClient:  llmClient,
		config:     DefaultLongTermConfig(),
	}
}

// SetConfig 设置配置
func (m *LongTermMemory) SetConfig(config LongTermConfig) {
	m.config = config
}

// Store 存储记忆
func (m *LongTermMemory) Store(ctx context.Context, content string, metadata map[string]interface{}) error {
	// 1. 总结记忆（压缩存储）
	processedContent := content
	if m.config.SummaryEnabled && m.llmClient != nil {
		summary, err := m.summarize(ctx, content)
		if err == nil && summary != "" {
			processedContent = summary
		}
	}

	// 2. 截断过长的内容
	if len(processedContent) > m.config.MaxContentLen {
		processedContent = processedContent[:m.config.MaxContentLen]
	}

	// 3. 生成向量
	embedding, err := m.llmClient.GetEmbedding(ctx, processedContent)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	// 4. 准备元数据
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["timestamp"] = time.Now().Unix()
	metadata["original_content"] = content
	metadata["content_length"] = len(content)

	// 5. 存入向量库
	return m.vectorStore.IndexDocument(ctx, processedContent, embedding, metadata)
}

// StoreMemory 存储 Memory 对象
func (m *LongTermMemory) StoreMemory(ctx context.Context, memory *Memory) error {
	return m.Store(ctx, memory.Content, memory.Metadata)
}

// Recall 检索记忆
func (m *LongTermMemory) Recall(ctx context.Context, query string, limit int) ([]*Memory, error) {
	if limit <= 0 {
		limit = m.config.TopK
	}

	// 1. 生成查询向量
	embedding, err := m.llmClient.GetEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// 2. 检索相似记忆
	results, err := m.vectorStore.SearchWithScoreThreshold(
		ctx,
		embedding,
		limit,
		m.config.ScoreThreshold,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	// 3. 转换为 Memory 对象
	memories := make([]*Memory, 0, len(results))
	for _, r := range results {
		memory := &Memory{
			ID:       r.ID,
			Content:  r.Content,
			Metadata: r.Payload,
		}

		// 提取时间戳
		if ts, ok := r.Payload["timestamp"].(float64); ok {
			memory.Timestamp = int64(ts)
		}

		// 提取原始内容
		if orig, ok := r.Payload["original_content"].(string); ok {
			memory.Metadata["content"] = orig
		}

		memories = append(memories, memory)
	}

	return memories, nil
}

// Delete 删除记忆
func (m *LongTermMemory) Delete(ctx context.Context, id string) error {
	return m.vectorStore.Delete(ctx, []string{id})
}

// GetStats 获取记忆统计
func (m *LongTermMemory) GetStats(ctx context.Context) (*MemoryStats, error) {
	info, err := m.vectorStore.GetCollectionInfo(ctx)
	if err != nil {
		return nil, err
	}

	return &MemoryStats{
		TotalMemories: int(info.Points),
		VectorSize:    int(info.VectorSize),
		Status:        info.Status,
	}, nil
}

// MemoryStats 记忆统计
type MemoryStats struct {
	TotalMemories int
	VectorSize    int
	Status        string
}

// summarize 总结内容
func (m *LongTermMemory) summarize(ctx context.Context, content string) (string, error) {
	if m.llmClient == nil {
		return content, nil
	}

	prompt := fmt.Sprintf(`请用一句话总结以下内容的要点（不超过200字）：

%s

总结：`, content)

	messages := []llm.Message{
		{Role: "user", Content: prompt},
	}

	resp, err := m.llmClient.Chat(ctx, messages)
	if err != nil {
		return "", err
	}

	return resp.Message.Content, nil
}

// Memory 记忆结构
type Memory struct {
	ID        string                 `json:"id"`
	Content   string                 `json:"content"`
	Timestamp int64                 `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NewMemory 创建新记忆
func NewMemory(content string, metadata map[string]interface{}) *Memory {
	return &Memory{
		ID:        uuid.New().String(),
		Content:   content,
		Timestamp: time.Now().Unix(),
		Metadata:  metadata,
	}
}
