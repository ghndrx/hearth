package ratelimit

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

//go:embed lua/sliding_window.lua
var slidingWindowScript string

var (
	// ErrRedisUnavailable indicates Redis is not available
	ErrRedisUnavailable = errors.New("redis unavailable")
)

// RedisClient defines the interface for Redis operations needed by the rate limiter
type RedisClient interface {
	EvalSha(ctx context.Context, sha string, keys []string, args ...interface{}) *redis.Cmd
	ScriptLoad(ctx context.Context, script string) *redis.StringCmd
	Ping(ctx context.Context) *redis.StatusCmd
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Incr(ctx context.Context, key string) *redis.IntCmd
	Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd
}

// RedisLimiterConfig holds configuration for the Redis rate limiter
type RedisLimiterConfig struct {
	// Enabled determines if Redis rate limiting is active
	Enabled bool
	// Prefix for all rate limit keys
	KeyPrefix string
	// DefaultWindow is the default time window for rate limiting
	DefaultWindow time.Duration
	// DefaultMax is the default maximum requests per window
	DefaultMax int
	// FallbackOnError determines if we should fail open on Redis errors
	FallbackOnError bool
}

// DefaultRedisLimiterConfig returns sensible defaults
func DefaultRedisLimiterConfig() RedisLimiterConfig {
	return RedisLimiterConfig{
		Enabled:         true,
		KeyPrefix:       "hearth:ratelimit:",
		DefaultWindow:   60 * time.Second,
		DefaultMax:      10000,
		FallbackOnError: true,
	}
}

// RedisLimiter implements distributed rate limiting using Redis
type RedisLimiter struct {
	client     RedisClient
	config     RedisLimiterConfig
	scriptSHA  string
	scriptOnce sync.Once
	scriptErr  error
	scriptMu   sync.RWMutex // Protects scriptSHA and scriptErr reads/writes
	available  bool
	availMu    sync.RWMutex

	// Fallback in-memory limiter for when Redis is unavailable
	fallback *Limiter
}

// NewRedisLimiter creates a new Redis-backed rate limiter
// Script loading is done lazily on first use to avoid race conditions
func NewRedisLimiter(client RedisClient, config RedisLimiterConfig, fallback *Limiter) *RedisLimiter {
	return &RedisLimiter{
		client:    client,
		config:    config,
		available: true,
		fallback:  fallback,
	}
}

// loadScript loads the Lua script into Redis
func (rl *RedisLimiter) loadScript(ctx context.Context) {
	rl.scriptOnce.Do(func() {
		if rl.client == nil {
			rl.scriptMu.Lock()
			rl.scriptErr = ErrRedisUnavailable
			rl.scriptMu.Unlock()
			rl.setAvailable(false)
			return
		}

		result := rl.client.ScriptLoad(ctx, slidingWindowScript)
		sha, err := result.Result()
		if err != nil {
			rl.scriptMu.Lock()
			rl.scriptErr = err
			rl.scriptMu.Unlock()
			rl.setAvailable(false)
			return
		}

		rl.scriptMu.Lock()
		rl.scriptSHA = sha
		rl.scriptMu.Unlock()
		rl.setAvailable(true)
	})
}

// setAvailable sets the availability status
func (rl *RedisLimiter) setAvailable(available bool) {
	rl.availMu.Lock()
	defer rl.availMu.Unlock()
	rl.available = available
}

// IsAvailable returns whether Redis is available for rate limiting
func (rl *RedisLimiter) IsAvailable() bool {
	rl.availMu.RLock()
	defer rl.availMu.RUnlock()
	return rl.available
}

// Result holds the result of a rate limit check
type Result struct {
	// Allowed indicates if the request is allowed
	Allowed bool
	// Remaining is the number of requests remaining in the window
	Remaining int
	// ResetAt is the Unix timestamp (milliseconds) when the window resets
	ResetAt int64
	// RetryAfter is the number of milliseconds to wait before retrying (if rate limited)
	RetryAfter int64
	// Limit is the maximum number of requests allowed
	Limit int
}

// Check performs a rate limit check using the sliding window algorithm
func (rl *RedisLimiter) Check(ctx context.Context, key string, limit int, window time.Duration) (*Result, error) {
	if !rl.config.Enabled {
		return &Result{
			Allowed:   true,
			Remaining: limit,
			Limit:     limit,
			ResetAt:   time.Now().Add(window).UnixMilli(),
		}, nil
	}

	// Ensure script is loaded (check with read lock first)
	rl.scriptMu.RLock()
	sha := rl.scriptSHA
	scriptErr := rl.scriptErr
	rl.scriptMu.RUnlock()

	if sha == "" {
		rl.loadScript(ctx)
		// Re-read after loading
		rl.scriptMu.RLock()
		sha = rl.scriptSHA
		scriptErr = rl.scriptErr
		rl.scriptMu.RUnlock()
	}

	// Check if Redis is available
	if !rl.IsAvailable() || scriptErr != nil {
		return rl.handleFallback(ctx, key, limit, window)
	}

	fullKey := rl.config.KeyPrefix + key

	// Execute the Lua script
	now := time.Now().UnixMicro()
	windowMicro := window.Microseconds()

	result := rl.client.EvalSha(
		ctx,
		sha,
		[]string{fullKey},
		now,
		windowMicro,
		limit,
		1, // weight
	)

	vals, err := result.Result()
	if err != nil {
		// Redis error - check if it's a NOSCRIPT error (script not loaded)
		if err.Error() == "NOSCRIPT No matching script. Please use EVAL." {
			// Reload script and retry once
			rl.scriptOnce = sync.Once{}
			rl.loadScript(ctx)
			return rl.Check(ctx, key, limit, window)
		}

		rl.setAvailable(false)
		return rl.handleFallback(ctx, key, limit, window)
	}

	// Parse result
	resultSlice, ok := vals.([]interface{})
	if !ok || len(resultSlice) < 4 {
		return nil, errors.New("unexpected response from rate limit script")
	}

	allowed := resultSlice[0].(int64) == 1
	currentCount := int(resultSlice[1].(int64))
	ttlMs := resultSlice[2].(int64)
	resetAt := resultSlice[3].(int64)

	remaining := limit - currentCount
	if remaining < 0 {
		remaining = 0
	}

	res := &Result{
		Allowed:   allowed,
		Remaining: remaining,
		ResetAt:   resetAt,
		Limit:     limit,
	}

	if !allowed {
		res.RetryAfter = ttlMs
	}

	return res, nil
}

// handleFallback handles rate limiting when Redis is unavailable
func (rl *RedisLimiter) handleFallback(ctx context.Context, key string, limit int, window time.Duration) (*Result, error) {
	if !rl.config.FallbackOnError {
		return nil, ErrRedisUnavailable
	}

	// Use in-memory fallback if available
	if rl.fallback != nil {
		cfg := Config{Limit: limit, Window: window}
		err := rl.fallback.Check(ctx, key, cfg)
		if err == ErrRateLimited {
			return &Result{
				Allowed:    false,
				Remaining:  0,
				RetryAfter: window.Milliseconds(),
				ResetAt:    time.Now().Add(window).UnixMilli(),
				Limit:      limit,
			}, nil
		}
		return &Result{
			Allowed:   true,
			Remaining: limit - 1,
			ResetAt:   time.Now().Add(window).UnixMilli(),
			Limit:     limit,
		}, nil
	}

	// No fallback - fail open
	return &Result{
		Allowed:   true,
		Remaining: limit,
		ResetAt:   time.Now().Add(window).UnixMilli(),
		Limit:     limit,
	}, nil
}

// CheckUser checks rate limit for a user action
func (rl *RedisLimiter) CheckUser(ctx context.Context, userID uuid.UUID, action string, limit int, window time.Duration) (*Result, error) {
	key := fmt.Sprintf("user:%s:%s", userID, action)
	return rl.Check(ctx, key, limit, window)
}

// CheckIP checks rate limit for an IP address
func (rl *RedisLimiter) CheckIP(ctx context.Context, ip string, action string, limit int, window time.Duration) (*Result, error) {
	key := fmt.Sprintf("ip:%s:%s", ip, action)
	return rl.Check(ctx, key, limit, window)
}

// CheckEndpoint checks rate limit for a specific endpoint
func (rl *RedisLimiter) CheckEndpoint(ctx context.Context, endpoint string, identifier string, limit int, window time.Duration) (*Result, error) {
	key := fmt.Sprintf("endpoint:%s:%s", endpoint, identifier)
	return rl.Check(ctx, key, limit, window)
}

// CheckChannel checks rate limit for a channel action
func (rl *RedisLimiter) CheckChannel(ctx context.Context, userID, channelID uuid.UUID, action string, limit int, window time.Duration) (*Result, error) {
	key := fmt.Sprintf("channel:%s:%s:%s", channelID, userID, action)
	return rl.Check(ctx, key, limit, window)
}

// GetInfo retrieves current rate limit info without consuming a request
func (rl *RedisLimiter) GetInfo(ctx context.Context, key string, limit int, window time.Duration) (*Result, error) {
	if !rl.config.Enabled || !rl.IsAvailable() {
		return &Result{
			Allowed:   true,
			Remaining: limit,
			Limit:     limit,
			ResetAt:   time.Now().Add(window).UnixMilli(),
		}, nil
	}

	fullKey := rl.config.KeyPrefix + key

	// Get current count from sorted set
	client, ok := rl.client.(*redis.Client)
	if !ok {
		return &Result{
			Allowed:   true,
			Remaining: limit,
			Limit:     limit,
			ResetAt:   time.Now().Add(window).UnixMilli(),
		}, nil
	}

	now := time.Now().UnixMicro()
	windowStart := now - window.Microseconds()

	// Count entries in the current window
	count, err := client.ZCount(ctx, fullKey, fmt.Sprintf("%d", windowStart), fmt.Sprintf("%d", now)).Result()
	if err != nil {
		// Redis error or key doesn't exist - return full limit
		return &Result{
			Allowed:   true,
			Remaining: limit,
			Limit:     limit,
			ResetAt:   time.Now().Add(window).UnixMilli(),
		}, nil
	}

	remaining := limit - int(count)
	if remaining < 0 {
		remaining = 0
	}

	return &Result{
		Allowed:   remaining > 0,
		Remaining: remaining,
		Limit:     limit,
		ResetAt:   time.Now().Add(window).UnixMilli(),
	}, nil
}

// HealthCheck performs a health check on the Redis connection
func (rl *RedisLimiter) HealthCheck(ctx context.Context) error {
	if rl.client == nil {
		return ErrRedisUnavailable
	}

	result := rl.client.Ping(ctx)
	if err := result.Err(); err != nil {
		rl.setAvailable(false)
		return err
	}

	// Try to reload script if it failed before
	rl.scriptMu.RLock()
	hasScriptErr := rl.scriptErr != nil
	rl.scriptMu.RUnlock()

	if hasScriptErr {
		rl.scriptOnce = sync.Once{}
		rl.loadScript(ctx)
	}

	rl.scriptMu.RLock()
	scriptErr := rl.scriptErr
	rl.scriptMu.RUnlock()

	if scriptErr == nil {
		rl.setAvailable(true)
	}

	return scriptErr
}

// RateLimitStoreAdapter adapts RedisLimiter to the middleware.RateLimitStore interface
// ResetScriptState resets the script loading state (for testing only)
func (rl *RedisLimiter) ResetScriptState() {
	rl.scriptMu.Lock()
	rl.scriptOnce = sync.Once{}
	rl.scriptSHA = ""
	rl.scriptErr = nil
	rl.scriptMu.Unlock()
}

type RateLimitStoreAdapter struct {
	limiter *RedisLimiter
	config  RedisLimiterConfig
}

// NewRateLimitStoreAdapter creates a new adapter for the middleware
func NewRateLimitStoreAdapter(limiter *RedisLimiter) *RateLimitStoreAdapter {
	return &RateLimitStoreAdapter{
		limiter: limiter,
		config:  limiter.config,
	}
}

// IncrementWithExpiry implements the middleware.RateLimitStore interface
func (a *RateLimitStoreAdapter) IncrementWithExpiry(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	// For backward compatibility, use simple increment
	if a.limiter.client == nil || !a.limiter.IsAvailable() {
		return 0, ErrRedisUnavailable
	}

	client, ok := a.limiter.client.(*redis.Client)
	if !ok {
		return 0, errors.New("invalid redis client type")
	}

	fullKey := a.config.KeyPrefix + "simple:" + key

	pipe := client.Pipeline()
	incr := pipe.Incr(ctx, fullKey)
	pipe.Expire(ctx, fullKey, ttl)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}

	return incr.Val(), nil
}

// Get implements the middleware.RateLimitStore interface
func (a *RateLimitStoreAdapter) Get(ctx context.Context, key string) ([]byte, error) {
	if a.limiter.client == nil || !a.limiter.IsAvailable() {
		return nil, ErrRedisUnavailable
	}

	client, ok := a.limiter.client.(*redis.Client)
	if !ok {
		return nil, errors.New("invalid redis client type")
	}

	fullKey := a.config.KeyPrefix + "simple:" + key
	return client.Get(ctx, fullKey).Bytes()
}

// MultiLimiter combines multiple rate limiters with fallback support
type MultiLimiter struct {
	redis    *RedisLimiter
	memory   *Limiter
	useRedis bool
}

// NewMultiLimiter creates a new multi-backend rate limiter
func NewMultiLimiter(redis *RedisLimiter, memory *Limiter) *MultiLimiter {
	useRedis := redis != nil && redis.config.Enabled
	return &MultiLimiter{
		redis:    redis,
		memory:   memory,
		useRedis: useRedis,
	}
}

// Check performs a rate limit check, preferring Redis but falling back to memory
func (ml *MultiLimiter) Check(ctx context.Context, key string, cfg Config) error {
	if ml.useRedis && ml.redis != nil && ml.redis.IsAvailable() {
		result, err := ml.redis.Check(ctx, key, cfg.Limit, cfg.Window)
		if err == nil {
			if !result.Allowed {
				return ErrRateLimited
			}
			return nil
		}
		// Fall through to memory limiter on error
	}

	if ml.memory != nil {
		return ml.memory.Check(ctx, key, cfg)
	}

	// No limiter available - fail open
	return nil
}

// CheckUser checks rate limit for a user action
func (ml *MultiLimiter) CheckUser(ctx context.Context, userID uuid.UUID, action string, cfg Config) error {
	key := fmt.Sprintf("user:%s:%s", userID, action)
	return ml.Check(ctx, key, cfg)
}

// CheckIP checks rate limit for an IP address
func (ml *MultiLimiter) CheckIP(ctx context.Context, ip string, action string, cfg Config) error {
	key := fmt.Sprintf("ip:%s:%s", ip, action)
	return ml.Check(ctx, key, cfg)
}

// CheckChannel checks rate limit for a channel action
func (ml *MultiLimiter) CheckChannel(ctx context.Context, userID, channelID uuid.UUID, action string, cfg Config) error {
	key := fmt.Sprintf("channel:%s:%s:%s", channelID, userID, action)
	return ml.Check(ctx, key, cfg)
}

// IsRedisAvailable returns whether Redis is currently being used
func (ml *MultiLimiter) IsRedisAvailable() bool {
	return ml.useRedis && ml.redis != nil && ml.redis.IsAvailable()
}
