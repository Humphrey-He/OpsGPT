package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// RedisConfig Redis 配置
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
	PoolSize int
}

// DefaultRedisConfig 默认配置
func DefaultRedisConfig() RedisConfig {
	return RedisConfig{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
		PoolSize: 10,
	}
}

// RedisMemory Redis 短期记忆
type RedisMemory struct {
	client *InMemoryClient // 使用内存客户端，实际项目用 Redis 客户端
	config RedisConfig
	ttl    time.Duration
}

// NewRedisMemory 创建 Redis 记忆
func NewRedisMemory(config RedisConfig) (*RedisMemory, error) {
	return &RedisMemory{
		client: NewInMemoryClient(),
		config: config,
		ttl:    24 * time.Hour,
	}, nil
}

// SetTTL 设置 TTL
func (m *RedisMemory) SetTTL(ttl time.Duration) {
	m.ttl = ttl
}

// Append 添加记忆
func (m *RedisMemory) Append(ctx context.Context, event *MemoryEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	key := m.eventKey(event.SessionID)
	m.client.RPush(ctx, key, string(data))

	return nil
}

// GetRecent 获取最近的记忆
func (m *RedisMemory) GetRecent(ctx context.Context, sessionID string, n int) ([]*MemoryEvent, error) {
	key := m.eventKey(sessionID)

	events, err := m.client.LRange(ctx, key, int64(-n), -1)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent events: %w", err)
	}

	result := make([]*MemoryEvent, 0, len(events))
	for _, e := range events {
		var event MemoryEvent
		if err := json.Unmarshal([]byte(e), &event); err != nil {
			continue
		}
		result = append(result, &event)
	}

	return result, nil
}

// GetByType 获取指定类型的记忆
func (m *RedisMemory) GetByType(ctx context.Context, sessionID, eventType string, n int) ([]*MemoryEvent, error) {
	events, err := m.GetRecent(ctx, sessionID, n*2)
	if err != nil {
		return nil, err
	}

	result := make([]*MemoryEvent, 0)
	for _, e := range events {
		if e.Type == eventType {
			result = append(result, e)
			if len(result) >= n {
				break
			}
		}
	}

	return result, nil
}

// Clear 清除记忆
func (m *RedisMemory) Clear(ctx context.Context, sessionID string) error {
	key := m.eventKey(sessionID)
	return m.client.Del(ctx, key)
}

// Search 搜索记忆
func (m *RedisMemory) Search(ctx context.Context, sessionID string, keyword string) ([]*MemoryEvent, error) {
	events, err := m.GetRecent(ctx, sessionID, 100)
	if err != nil {
		return nil, err
	}

	result := make([]*MemoryEvent, 0)
	for _, e := range events {
		if strings.Contains(e.Content, keyword) {
			result = append(result, e)
		}
	}

	return result, nil
}

// eventKey 生成事件存储的 key
func (m *RedisMemory) eventKey(sessionID string) string {
	return fmt.Sprintf("memory:events:%s", sessionID)
}

// MemoryEvent 记忆事件
type MemoryEvent struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"` // user, assistant, system, tool
	Content   string                 `json:"content"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp int64                 `json:"timestamp"`
	SessionID string                `json:"session_id"`
}

// NewMemoryEvent 创建记忆事件
func NewMemoryEvent(sessionID, eventType, content string) *MemoryEvent {
	return &MemoryEvent{
		ID:        uuid.New().String(),
		Type:      eventType,
		Content:   content,
		Metadata:  make(map[string]interface{}),
		Timestamp: time.Now().Unix(),
		SessionID: sessionID,
	}
}

// InMemoryClient 内存实现的客户端（用于测试或无 Redis 环境）
type InMemoryClient struct {
	data map[string]string
	list map[string][]string
}

// NewInMemoryClient 创建内存客户端
func NewInMemoryClient() *InMemoryClient {
	return &InMemoryClient{
		data: make(map[string]string),
		list: make(map[string][]string),
	}
}

// Get 获取值
func (c *InMemoryClient) Get(ctx context.Context, key string) (string, error) {
	return c.data[key], nil
}

// Set 设置值
func (c *InMemoryClient) Set(ctx context.Context, key string, value string, expiration time.Duration) error {
	c.data[key] = value
	return nil
}

// Del 删除键
func (c *InMemoryClient) Del(ctx context.Context, keys ...string) error {
	for _, key := range keys {
		delete(c.data, key)
		delete(c.list, key)
	}
	return nil
}

// Expire 设置过期时间（内存实现忽略）
func (c *InMemoryClient) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return nil
}

// LPush 左插入
func (c *InMemoryClient) LPush(ctx context.Context, key string, values ...string) error {
	list := c.list[key]
	for i := len(values) - 1; i >= 0; i-- {
		list = append([]string{values[i]}, list...)
	}
	c.list[key] = list
	return nil
}

// RPush 右插入
func (c *InMemoryClient) RPush(ctx context.Context, key string, values ...string) error {
	list := c.list[key]
	list = append(list, values...)
	c.list[key] = list
	return nil
}

// LRange 范围获取
func (c *InMemoryClient) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	list := c.list[key]
	if list == nil {
		return []string{}, nil
	}

	listLen := int64(len(list))

	// 处理负数索引
	if start < 0 {
		start = listLen + start
	}
	if stop < 0 {
		stop = listLen + stop
	}

	// 边界检查
	if start < 0 {
		start = 0
	}
	if stop >= listLen {
		stop = listLen - 1
	}

	if start > stop || start >= listLen {
		return []string{}, nil
	}

	return list[start : stop+1], nil
}

// HGet 获取哈希字段
func (c *InMemoryClient) HGet(ctx context.Context, key, field string) (string, error) {
	hashKey := key + ":" + field
	return c.data[hashKey], nil
}

// HSet 设置哈希字段
func (c *InMemoryClient) HSet(ctx context.Context, key, field string, value interface{}) error {
	hashKey := key + ":" + field
	c.data[hashKey] = fmt.Sprintf("%v", value)
	return nil
}

// HGetAll 获取所有哈希字段
func (c *InMemoryClient) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	result := make(map[string]string)
	prefix := key + ":"
	for k, v := range c.data {
		if strings.HasPrefix(k, prefix) {
			result[strings.TrimPrefix(k, prefix)] = v
		}
	}
	return result, nil
}

// ConversationSessionManager 会话管理器
type ConversationSessionManager struct {
	memory *RedisMemory
}

// NewConversationSessionManager 创建会话管理器
func NewConversationSessionManager(memory *RedisMemory) *ConversationSessionManager {
	return &ConversationSessionManager{
		memory: memory,
	}
}

// Session 会话
type Session struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"user_id"`
	CreatedAt time.Time             `json:"created_at"`
	UpdatedAt time.Time             `json:"updated_at"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NewSession 创建新会话
func NewSession(userID string) *Session {
	now := time.Now()
	return &Session{
		ID:        uuid.New().String(),
		UserID:    userID,
		CreatedAt: now,
		UpdatedAt: now,
		Metadata:  make(map[string]interface{}),
	}
}

// CreateSession 创建会话
func (m *ConversationSessionManager) CreateSession(ctx context.Context, userID string) (*Session, error) {
	session := NewSession(userID)

	data, err := json.Marshal(session)
	if err != nil {
		return nil, err
	}

	key := fmt.Sprintf("session:%s", session.ID)
	m.memory.client.Set(ctx, key, string(data), 7*24*time.Hour)

	return session, nil
}

// GetSession 获取会话
func (m *ConversationSessionManager) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	key := fmt.Sprintf("session:%s", sessionID)
	data, err := m.memory.client.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	var session Session
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, err
	}

	return &session, nil
}

// DeleteSession 删除会话
func (m *ConversationSessionManager) DeleteSession(ctx context.Context, sessionID string) error {
	key := fmt.Sprintf("session:%s", sessionID)
	memoryKey := m.memory.eventKey(sessionID)
	return m.memory.client.Del(ctx, key, memoryKey)
}

// AddMessage 添加消息到会话
func (m *ConversationSessionManager) AddMessage(ctx context.Context, sessionID, role, content string) error {
	event := NewMemoryEvent(sessionID, role, content)
	return m.memory.Append(ctx, event)
}

// GetMessages 获取会话消息
func (m *ConversationSessionManager) GetMessages(ctx context.Context, sessionID string, limit int) ([]*MemoryEvent, error) {
	return m.memory.GetRecent(ctx, sessionID, limit)
}
