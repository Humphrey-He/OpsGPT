package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// TokenBucket 令牌桶限流器
type TokenBucket struct {
	mu       sync.Mutex
	rate     float64       // 每秒生成的令牌数
	capacity int           // 桶容量
	tokens   float64       // 当前令牌数
	lastTime time.Time      // 上次更新时间
}

// NewTokenBucket 创建令牌桶限流器
func NewTokenBucket(rate float64, capacity int) *TokenBucket {
	return &TokenBucket{
		rate:     rate,
		capacity: capacity,
		tokens:   float64(capacity), // 初始满桶
		lastTime: time.Now(),
	}
}

// Allow 检查是否允许请求
func (b *TokenBucket) Allow() bool {
	return b.AllowN(1)
}

// AllowN 检查是否允许 N 个请求
func (b *TokenBucket) AllowN(n int) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	// 更新令牌数
	now := time.Now()
	elapsed := now.Sub(b.lastTime).Seconds()
	b.tokens += elapsed * b.rate
	if b.tokens > float64(b.capacity) {
		b.tokens = float64(b.capacity)
	}
	b.lastTime = now

	// 检查令牌是否足够
	if b.tokens >= float64(n) {
		b.tokens -= float64(n)
		return true
	}

	return false
}

// Wait 等待获取令牌（阻塞）
func (b *TokenBucket) Wait(ctx context.Context) error {
	return b.WaitN(ctx, 1)
}

// WaitN 等待获取 N 个令牌
func (b *TokenBucket) WaitN(ctx context.Context, n int) error {
	for {
		if b.AllowN(n) {
			return nil
		}

		// 计算需要等待的时间
		b.mu.Lock()
		needed := float64(n) - b.tokens
		waitTime := time.Duration(needed/b.rate*float64(time.Second))
		b.mu.Unlock()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
		}
	}
}

// Rate 返回每秒令牌数
func (b *TokenBucket) Rate() float64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.rate
}

// Capacity 返回桶容量
func (b *TokenBucket) Capacity() int {
	return b.capacity
}

// Tokens 返回当前令牌数
func (b *TokenBucket) Tokens() float64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	// 更新令牌数
	now := time.Now()
	elapsed := now.Sub(b.lastTime).Seconds()
	tokens := b.tokens + elapsed*b.rate
	if tokens > float64(b.capacity) {
		tokens = float64(b.capacity)
	}
	return tokens
}

// Reset 重置限流器
func (b *TokenBucket) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.tokens = float64(b.capacity)
	b.lastTime = time.Now()
}

// LeakyBucket 漏桶限流器
type LeakyBucket struct {
	mu       sync.Mutex
	rate     float64       // 每秒漏出的请求数
	capacity int           // 桶容量
	level    int           // 当前请求数
	lastTime time.Time      // 上次漏出时间
}

// NewLeakyBucket 创建漏桶限流器
func NewLeakyBucket(rate float64, capacity int) *LeakyBucket {
	return &LeakyBucket{
		rate:     rate,
		capacity: capacity,
		level:    0,
		lastTime: time.Now(),
	}
}

// Allow 检查是否允许请求
func (b *LeakyBucket) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	// 漏出
	now := time.Now()
	elapsed := now.Sub(b.lastTime).Seconds()
	b.level -= int(elapsed * b.rate)
	if b.level < 0 {
		b.level = 0
	}
	b.lastTime = now

	// 检查桶是否有空间
	if b.level < b.capacity {
		b.level++
		return true
	}

	return false
}

// RateLimiter 接口
type RateLimiter interface {
	Allow() bool
	Wait(ctx context.Context) error
	Reset()
}

// MultiLimiter 多维度限流器
type MultiLimiter struct {
	mu       sync.RWMutex
	limiters map[string]RateLimiter
}

// NewMultiLimiter 创建多维度限流器
func NewMultiLimiter() *MultiLimiter {
	return &MultiLimiter{
		limiters: make(map[string]RateLimiter),
	}
}

// Add 添加限流器
func (m *MultiLimiter) Add(key string, limiter RateLimiter) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.limiters[key] = limiter
}

// Allow 检查所有限流器
func (m *MultiLimiter) Allow(key string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	limiter, ok := m.limiters[key]
	if !ok {
		return true
	}
	return limiter.Allow()
}

// AllowAll 检查所有限流器
func (m *MultiLimiter) AllowAll() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, limiter := range m.limiters {
		if !limiter.Allow() {
			return false
		}
	}
	return true
}

// Wait 等待获取令牌
func (m *MultiLimiter) Wait(ctx context.Context, key string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	limiter, ok := m.limiters[key]
	if !ok {
		return nil
	}
	return limiter.Wait(ctx)
}

// UserRateLimiter 基于用户的限流器
type UserRateLimiter struct {
	mu      sync.RWMutex
	users   map[string]*TokenBucket
	rate    float64
	capacity int
	ttl     time.Duration // 用户记录过期时间
}

// NewUserRateLimiter 创建用户限流器
func NewUserRateLimiter(rate float64, capacity int, ttl time.Duration) *UserRateLimiter {
	return &UserRateLimiter{
		users:    make(map[string]*TokenBucket),
		rate:     rate,
		capacity: capacity,
		ttl:     ttl,
	}
}

// Allow 检查用户是否允许请求
func (l *UserRateLimiter) Allow(userID string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	bucket, ok := l.users[userID]
	if !ok {
		bucket = NewTokenBucket(l.rate, l.capacity)
		l.users[userID] = bucket
		return true
	}

	return bucket.Allow()
}

// Cleanup 清理过期用户
func (l *UserRateLimiter) Cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	for userID, bucket := range l.users {
		if now.Sub(bucket.lastTime) > l.ttl {
			delete(l.users, userID)
		}
	}
}

// Count 返回当前用户数
func (l *UserRateLimiter) Count() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.users)
}

// IPRateLimiter 基于 IP 的限流器
type IPRateLimiter struct {
	mu      sync.RWMutex
	ips     map[string]*TokenBucket
	rate    float64
	capacity int
	ttl     time.Duration
}

// NewIPRateLimiter 创建 IP 限流器
func NewIPRateLimiter(rate float64, capacity int, ttl time.Duration) *IPRateLimiter {
	return &IPRateLimiter{
		ips:      make(map[string]*TokenBucket),
		rate:     rate,
		capacity: capacity,
		ttl:     ttl,
	}
}

// Allow 检查 IP 是否允许请求
func (l *IPRateLimiter) Allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	bucket, ok := l.ips[ip]
	if !ok {
		bucket = NewTokenBucket(l.rate, l.capacity)
		l.ips[ip] = bucket
		return true
	}

	return bucket.Allow()
}

// Cleanup 清理过期 IP
func (l *IPRateLimiter) Cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	for ip, bucket := range l.ips {
		if now.Sub(bucket.lastTime) > l.ttl {
			delete(l.ips, ip)
		}
	}
}

// Count 返回当前 IP 数
func (l *IPRateLimiter) Count() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.ips)
}

// RateLimitMiddleware 限流中间件
type RateLimitMiddleware struct {
	limiter RateLimiter
	onLimitExceeded func(ip string)
}

// NewRateLimitMiddleware 创建限流中间件
func NewRateLimitMiddleware(limiter RateLimiter) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		limiter: limiter,
	}
}

// SetOnLimitExceeded 设置超限回调
func (m *RateLimitMiddleware) SetOnLimitExceeded(fn func(ip string)) {
	m.onLimitExceeded = fn
}

// Handle 处理请求
func (m *RateLimitMiddleware) Handle(ip string) error {
	if !m.limiter.Allow() {
		if m.onLimitExceeded != nil {
			m.onLimitExceeded(ip)
		}
		return fmt.Errorf("rate limit exceeded for IP: %s", ip)
	}
	return nil
}

// Config 限流配置
type Config struct {
	RequestsPerSecond float64 // 每秒请求数
	BurstSize        int     // 突发大小
	Enabled          bool    // 是否启用
}

// DefaultConfig 返回默认配置
func DefaultConfig() Config {
	return Config{
		RequestsPerSecond: 10,
		BurstSize:        20,
		Enabled:          true,
	}
}
