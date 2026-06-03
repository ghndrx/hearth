package middleware

import (
	"context"
	"fmt"
	"sync"
	"time"

	"hearth/internal/ratelimit"
)

// SimpleInMemoryCache is a minimal in-memory Cache implementation for rate limiting
// when Redis is not available
type SimpleInMemoryCache struct {
	mu     sync.Mutex
	items  map[string]int64
	expiry map[string]time.Time
}

// NewSimpleInMemoryCache creates a new simple in-memory cache for rate limiting
func NewSimpleInMemoryCache() *SimpleInMemoryCache {
	return &SimpleInMemoryCache{
		items:  make(map[string]int64),
		expiry: make(map[string]time.Time),
	}
}

func (c *SimpleInMemoryCache) IncrementWithExpiry(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if expired
	if exp, ok := c.expiry[key]; ok && time.Now().After(exp) {
		c.items[key] = 0
	}

	c.items[key]++
	c.expiry[key] = time.Now().Add(ttl)
	return c.items[key], nil
}

func (c *SimpleInMemoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if exp, ok := c.expiry[key]; ok && time.Now().After(exp) {
		delete(c.items, key)
		delete(c.expiry, key)
		return nil, nil
	}

	if count, ok := c.items[key]; ok {
		return []byte(fmt.Sprintf("%d", count)), nil
	}
	return nil, nil
}

// RedisRateLimiterAdapter adapts ratelimit.RedisLimiter to the api/middleware.RateLimiter interface
type RedisRateLimiterAdapter struct {
	limiter *ratelimit.RedisLimiter
	enabled bool
}

// NewRedisRateLimiterAdapter creates a new adapter for the Redis rate limiter
func NewRedisRateLimiterAdapter(limiter *ratelimit.RedisLimiter) *RedisRateLimiterAdapter {
	return &RedisRateLimiterAdapter{
		limiter: limiter,
		enabled: limiter != nil,
	}
}

// Check implements the RateLimiter interface using Redis-backed sliding window
func (a *RedisRateLimiterAdapter) Check(ctx context.Context, key string, limit int, window time.Duration) (*RateLimitResult, error) {
	if !a.enabled || a.limiter == nil {
		// Fail open if not enabled
		return &RateLimitResult{
			Allowed:   true,
			Remaining: limit,
			ResetAt:   time.Now().Add(window).UnixMilli(),
			Limit:     limit,
		}, nil
	}

	result, err := a.limiter.Check(ctx, key, limit, window)
	if err != nil {
		// Fail open on errors
		return &RateLimitResult{
			Allowed:   true,
			Remaining: limit,
			ResetAt:   time.Now().Add(window).UnixMilli(),
			Limit:     limit,
		}, nil
	}

	return &RateLimitResult{
		Allowed:    result.Allowed,
		Remaining:  result.Remaining,
		ResetAt:    result.ResetAt,
		RetryAfter: result.RetryAfter,
		Limit:      result.Limit,
	}, nil
}

// IsAvailable implements the RateLimiter interface
func (a *RedisRateLimiterAdapter) IsAvailable() bool {
	return a.enabled && a.limiter != nil && a.limiter.IsAvailable()
}

// MemoryRateLimiterAdapter adapts ratelimit.Limiter to the api/middleware.RateLimiter interface
type MemoryRateLimiterAdapter struct {
	limiter *ratelimit.Limiter
	enabled bool
}

// NewMemoryRateLimiterAdapter creates a new adapter for the in-memory rate limiter
func NewMemoryRateLimiterAdapter(limiter *ratelimit.Limiter) *MemoryRateLimiterAdapter {
	return &MemoryRateLimiterAdapter{
		limiter: limiter,
		enabled: limiter != nil,
	}
}

// Check implements the RateLimiter interface using in-memory sliding window
func (a *MemoryRateLimiterAdapter) Check(ctx context.Context, key string, limit int, window time.Duration) (*RateLimitResult, error) {
	if !a.enabled || a.limiter == nil {
		return &RateLimitResult{
			Allowed:   true,
			Remaining: limit,
			ResetAt:   time.Now().Add(window).UnixMilli(),
			Limit:     limit,
		}, nil
	}

	cfg := ratelimit.Config{
		Limit:  limit,
		Window: window,
	}

	err := a.limiter.Check(ctx, "ratelimit:"+key, cfg)
	remaining := limit - 1
	if remaining < 0 {
		remaining = 0
	}

	result := &RateLimitResult{
		Allowed:   err == nil,
		Remaining: remaining,
		ResetAt:   time.Now().Add(window).UnixMilli(),
		Limit:     limit,
	}

	if err == ratelimit.ErrRateLimited {
		result.RetryAfter = window.Milliseconds()
	}

	return result, nil
}

// IsAvailable implements the RateLimiter interface
func (a *MemoryRateLimiterAdapter) IsAvailable() bool {
	return a.enabled && a.limiter != nil
}

// HybridRateLimiterAdapter provides rate limiting with Redis primary and memory fallback
type HybridRateLimiterAdapter struct {
	redisLimiter  *ratelimit.RedisLimiter
	memoryLimiter *ratelimit.Limiter
	useRedis      bool
	useMemory     bool
}

// NewHybridRateLimiterAdapter creates a hybrid rate limiter adapter
func NewHybridRateLimiterAdapter(redisLimiter *ratelimit.RedisLimiter, memoryLimiter *ratelimit.Limiter) *HybridRateLimiterAdapter {
	return &HybridRateLimiterAdapter{
		redisLimiter:  redisLimiter,
		memoryLimiter: memoryLimiter,
		useRedis:      redisLimiter != nil,
		useMemory:     memoryLimiter != nil,
	}
}

// Check implements the RateLimiter interface
func (a *HybridRateLimiterAdapter) Check(ctx context.Context, key string, limit int, window time.Duration) (*RateLimitResult, error) {
	// Try Redis first if enabled and available
	if a.useRedis && a.redisLimiter != nil && a.redisLimiter.IsAvailable() {
		result, err := a.redisLimiter.Check(ctx, key, limit, window)
		if err == nil {
			return &RateLimitResult{
				Allowed:    result.Allowed,
				Remaining:  result.Remaining,
				ResetAt:    result.ResetAt,
				RetryAfter: result.RetryAfter,
				Limit:      result.Limit,
			}, nil
		}
		// Fall through to memory on Redis error
	}

	// Fall back to memory limiter
	if a.useMemory && a.memoryLimiter != nil {
		cfg := ratelimit.Config{
			Limit:  limit,
			Window: window,
		}

		err := a.memoryLimiter.Check(ctx, "ratelimit:"+key, cfg)
		remaining := limit
		if err == nil {
			// We consumed one request
			remaining = limit - 1
		}
		if remaining < 0 {
			remaining = 0
		}

		result := &RateLimitResult{
			Allowed:   err == nil,
			Remaining: remaining,
			ResetAt:   time.Now().Add(window).UnixMilli(),
			Limit:     limit,
		}

		if err == ratelimit.ErrRateLimited {
			result.RetryAfter = window.Milliseconds()
		}

		return result, nil
	}

	// No limiter available - fail open
	return &RateLimitResult{
		Allowed:   true,
		Remaining: limit,
		ResetAt:   time.Now().Add(window).UnixMilli(),
		Limit:     limit,
	}, nil
}

// IsAvailable implements the RateLimiter interface
func (a *HybridRateLimiterAdapter) IsAvailable() bool {
	if a.useRedis && a.redisLimiter != nil && a.redisLimiter.IsAvailable() {
		return true
	}
	if a.useMemory && a.memoryLimiter != nil {
		return true
	}
	return false
}

// IsRedisAvailable returns whether Redis is currently available for rate limiting
func (a *HybridRateLimiterAdapter) IsRedisAvailable() bool {
	return a.useRedis && a.redisLimiter != nil && a.redisLimiter.IsAvailable()
}
