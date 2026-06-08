package security

import (
	"context"
	"fmt"
	"time"
)

// RateLimiter 限流器
type RateLimiter struct {
	redis     RedisClient
	limit    int
	window   time.Duration
}

// RedisClient Redis 客户端接口
type RedisClient interface {
	Incr(ctx context.Context, key string) (int64, error)
	Expire(ctx context.Context, key string, expiration time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
}

// NewRateLimiter 创建限流器
func NewRateLimiter(redis RedisClient, limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		redis:   redis,
		limit:   limit,
		window:  window,
	}
}

// Allow 检查是否允许通过
func (r *RateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	rateKey := fmt.Sprintf("ratelimit:%s", key)

	// 增加计数
	count, err := r.redis.Incr(ctx, rateKey)
	if err != nil {
		return false, err
	}

	// 设置过期时间
	if count == 1 {
		r.redis.Expire(ctx, rateKey, r.window)
	}

	return count <= int64(r.limit), nil
}

// RateLimitExceededError 限流错误
type RateLimitExceededError struct {
	Key   string
	Limit int
}

func (e *RateLimitExceededError) Error() string {
	return fmt.Sprintf("rate limit exceeded for %s: max %d requests", e.Key, e.Limit)
}
