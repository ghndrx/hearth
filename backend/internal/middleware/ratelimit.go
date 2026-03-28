package middleware

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"hearth/internal/ratelimit"
)

// RateLimitStore defines the interface for rate limit storage backends
type RateLimitStore interface {
	// IncrementWithExpiry atomically increments a counter and sets/updates TTL
	IncrementWithExpiry(ctx context.Context, key string, ttl time.Duration) (int64, error)
	// Get retrieves the current count for a key
	Get(ctx context.Context, key string) ([]byte, error)
}

// RateLimitConfig holds configuration for rate limiting
type RateLimitConfig struct {
	// Enabled determines if rate limiting is active
	Enabled bool
	// Max is the maximum requests allowed in the window
	Max int
	// Window is the time window for rate limiting
	Window time.Duration
	// Burst is the additional allowance above Max (for token bucket)
	Burst int
	// AuthMultiplier multiplies the limit for authenticated users
	AuthMultiplier float64
	// SkipPaths are paths that bypass rate limiting (e.g., health checks)
	SkipPaths []string
	// KeyGenerator generates the rate limit key from request context
	KeyGenerator func(c *fiber.Ctx) string
	// OnLimitReached is called when rate limit is exceeded
	OnLimitReached func(c *fiber.Ctx) error
}

// DefaultRateLimitConfig returns sensible defaults
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Enabled:        true,
		Max:            10000,
		Window:         60 * time.Second,
		Burst:          0,
		AuthMultiplier: 2.0,
		SkipPaths: []string{
			"/health",
			"/healthz",
			"/readyz",
			"/metrics",
		},
		KeyGenerator: defaultKeyGenerator,
		OnLimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":   "rate_limited",
				"message": "Too many requests. Please try again later.",
			})
		},
	}
}

// defaultKeyGenerator extracts client IP for rate limiting
func defaultKeyGenerator(c *fiber.Ctx) string {
	return c.IP()
}

// RateLimiter is the middleware for HTTP rate limiting
type RateLimiter struct {
	store  RateLimitStore
	config RateLimitConfig
}

// NewRateLimiter creates a new rate limiter middleware
func NewRateLimiter(store RateLimitStore, config RateLimitConfig) *RateLimiter {
	if config.KeyGenerator == nil {
		config.KeyGenerator = defaultKeyGenerator
	}
	if config.OnLimitReached == nil {
		config.OnLimitReached = DefaultRateLimitConfig().OnLimitReached
	}
	if config.AuthMultiplier == 0 {
		config.AuthMultiplier = 1.0
	}
	return &RateLimiter{
		store:  store,
		config: config,
	}
}

// Middleware returns a Fiber middleware handler for global rate limiting
func (rl *RateLimiter) Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !rl.config.Enabled {
			return c.Next()
		}

		// Check if path should skip rate limiting
		path := c.Path()
		for _, skipPath := range rl.config.SkipPaths {
			if strings.HasPrefix(path, skipPath) {
				return c.Next()
			}
		}

		// Generate key for this request
		key := rl.config.KeyGenerator(c)
		if key == "" {
			// Can't identify client, allow through
			return c.Next()
		}

		// Calculate effective limit
		limit := rl.config.Max + rl.config.Burst

		// Check if user is authenticated and apply multiplier
		if userID := c.Locals("userID"); userID != nil {
			limit = int(float64(limit) * rl.config.AuthMultiplier)
			// Include user ID in key for per-user limiting
			if uid, ok := userID.(uuid.UUID); ok {
				key = fmt.Sprintf("user:%s:%s", uid.String(), key)
			}
		} else {
			key = fmt.Sprintf("ip:%s", key)
		}

		// Prefix for rate limiting namespace
		fullKey := fmt.Sprintf("ratelimit:global:%s", key)

		// Increment counter
		ctx := c.Context()
		count, err := rl.store.IncrementWithExpiry(ctx, fullKey, rl.config.Window)
		if err != nil {
			// Fail open: if Redis is down, allow the request
			return c.Next()
		}

		// Calculate remaining and reset time
		remaining := limit - int(count)
		if remaining < 0 {
			remaining = 0
		}
		resetAt := time.Now().Add(rl.config.Window).Unix()

		// Set rate limit headers
		c.Set("X-RateLimit-Limit", strconv.Itoa(limit))
		c.Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Set("X-RateLimit-Reset", strconv.FormatInt(resetAt, 10))

		// Check if limit exceeded
		if int(count) > limit {
			retryAfter := int(rl.config.Window.Seconds())
			c.Set("Retry-After", strconv.Itoa(retryAfter))
			return rl.config.OnLimitReached(c)
		}

		return c.Next()
	}
}

// EndpointConfig holds rate limiting configuration for specific endpoints
type EndpointConfig struct {
	Max    int           // Maximum requests allowed
	Window time.Duration // Time window
	Burst  int           // Burst allowance
}

// EndpointRateLimiter provides per-endpoint rate limiting
type EndpointRateLimiter struct {
	store    RateLimitStore
	defaults EndpointConfig
	configs  map[string]EndpointConfig
}

// NewEndpointRateLimiter creates a new endpoint-specific rate limiter
func NewEndpointRateLimiter(store RateLimitStore, defaults EndpointConfig) *EndpointRateLimiter {
	return &EndpointRateLimiter{
		store:    store,
		defaults: defaults,
		configs:  make(map[string]EndpointConfig),
	}
}

// Configure sets a custom rate limit for a specific endpoint pattern
func (el *EndpointRateLimiter) Configure(pattern string, config EndpointConfig) {
	el.configs[pattern] = config
}

// Common endpoint configurations
var (
	// AuthEndpointConfig for login/register endpoints (per-IP)
	AuthEndpointConfig = EndpointConfig{
		Max:    300,
		Window: 1 * time.Second,
		Burst:  50,
	}

	// UploadEndpointConfig for file upload endpoints
	UploadEndpointConfig = EndpointConfig{
		Max:    50,
		Window: 60 * time.Second,
		Burst:  10,
	}

	// MessageEndpointConfig for message sending (per-user)
	MessageEndpointConfig = EndpointConfig{
		Max:    500,
		Window: 1 * time.Second,
		Burst:  100,
	}

	// SearchEndpointConfig for search endpoints (expensive operations)
	SearchEndpointConfig = EndpointConfig{
		Max:    100,
		Window: 60 * time.Second,
		Burst:  20,
	}
)

// ForEndpoint returns a middleware configured for specific endpoints
func (el *EndpointRateLimiter) ForEndpoint(endpointName string) fiber.Handler {
	config, exists := el.configs[endpointName]
	if !exists {
		config = el.defaults
	}

	return func(c *fiber.Ctx) error {
		// Get user ID or IP for key
		var key string
		if userID := c.Locals("userID"); userID != nil {
			if uid, ok := userID.(uuid.UUID); ok {
				key = fmt.Sprintf("user:%s", uid.String())
			}
		}
		if key == "" {
			key = fmt.Sprintf("ip:%s", c.IP())
		}

		fullKey := fmt.Sprintf("ratelimit:endpoint:%s:%s", endpointName, key)
		limit := config.Max + config.Burst

		ctx := c.Context()
		count, err := el.store.IncrementWithExpiry(ctx, fullKey, config.Window)
		if err != nil {
			// Fail open
			return c.Next()
		}

		remaining := limit - int(count)
		if remaining < 0 {
			remaining = 0
		}
		resetAt := time.Now().Add(config.Window).Unix()

		// Set headers
		c.Set("X-RateLimit-Limit", strconv.Itoa(limit))
		c.Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Set("X-RateLimit-Reset", strconv.FormatInt(resetAt, 10))

		if int(count) > limit {
			retryAfter := int(config.Window.Seconds())
			c.Set("Retry-After", strconv.Itoa(retryAfter))
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":    "rate_limited",
				"message":  fmt.Sprintf("Rate limit exceeded for %s. Please try again later.", endpointName),
				"endpoint": endpointName,
			})
		}

		return c.Next()
	}
}

// SlidingWindowLimiter implements a more accurate sliding window algorithm
type SlidingWindowLimiter struct {
	store  RateLimitStore
	config RateLimitConfig
}

// NewSlidingWindowLimiter creates a new sliding window rate limiter
func NewSlidingWindowLimiter(store RateLimitStore, config RateLimitConfig) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		store:  store,
		config: config,
	}
}

// Middleware returns the sliding window rate limiting middleware
func (sw *SlidingWindowLimiter) Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !sw.config.Enabled {
			return c.Next()
		}

		// Check if path should skip rate limiting
		path := c.Path()
		for _, skipPath := range sw.config.SkipPaths {
			if strings.HasPrefix(path, skipPath) {
				return c.Next()
			}
		}

		// Generate key
		key := sw.config.KeyGenerator(c)
		if key == "" {
			return c.Next()
		}

		// Calculate effective limit based on authentication
		limit := sw.config.Max + sw.config.Burst
		if userID := c.Locals("userID"); userID != nil {
			limit = int(float64(limit) * sw.config.AuthMultiplier)
			if uid, ok := userID.(uuid.UUID); ok {
				key = fmt.Sprintf("user:%s:%s", uid.String(), key)
			}
		} else {
			key = fmt.Sprintf("ip:%s", key)
		}

		// For sliding window, we use two buckets: current and previous window
		now := time.Now()
		windowStart := now.Truncate(sw.config.Window)
		previousWindowStart := windowStart.Add(-sw.config.Window)

		// Calculate weight of previous window (how much of the previous window is still relevant)
		elapsedInCurrentWindow := now.Sub(windowStart)
		previousWeight := 1.0 - (float64(elapsedInCurrentWindow) / float64(sw.config.Window))

		// Build keys for both windows
		currentKey := fmt.Sprintf("ratelimit:sliding:%s:%d", key, windowStart.Unix())
		previousKey := fmt.Sprintf("ratelimit:sliding:%s:%d", key, previousWindowStart.Unix())

		ctx := c.Context()

		// Get previous window count
		var previousCount int64
		if data, err := sw.store.Get(ctx, previousKey); err == nil {
			if count, err := strconv.ParseInt(string(data), 10, 64); err == nil {
				previousCount = count
			}
		}

		// Increment current window
		currentCount, err := sw.store.IncrementWithExpiry(ctx, currentKey, sw.config.Window*2)
		if err != nil {
			// Fail open
			return c.Next()
		}

		// Calculate weighted count using sliding window algorithm
		weightedCount := int(float64(previousCount)*previousWeight) + int(currentCount)

		remaining := limit - weightedCount
		if remaining < 0 {
			remaining = 0
		}
		resetAt := windowStart.Add(sw.config.Window).Unix()

		// Set headers
		c.Set("X-RateLimit-Limit", strconv.Itoa(limit))
		c.Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Set("X-RateLimit-Reset", strconv.FormatInt(resetAt, 10))

		if weightedCount > limit {
			retryAfter := int(sw.config.Window.Seconds() - elapsedInCurrentWindow.Seconds())
			if retryAfter < 1 {
				retryAfter = 1
			}
			c.Set("Retry-After", strconv.Itoa(retryAfter))
			return sw.config.OnLimitReached(c)
		}

		return c.Next()
	}
}

// RateLimitInfo represents rate limit state for a client
type RateLimitInfo struct {
	Limit     int   `json:"limit"`
	Remaining int   `json:"remaining"`
	ResetAt   int64 `json:"reset_at"`
	RetryIn   int   `json:"retry_in,omitempty"`
}

// GetRateLimitInfo retrieves current rate limit state for a key
func GetRateLimitInfo(ctx context.Context, store RateLimitStore, key string, config RateLimitConfig) (*RateLimitInfo, error) {
	fullKey := fmt.Sprintf("ratelimit:global:%s", key)

	data, err := store.Get(ctx, fullKey)
	if err != nil {
		// Key doesn't exist, return full limit
		return &RateLimitInfo{
			Limit:     config.Max + config.Burst,
			Remaining: config.Max + config.Burst,
			ResetAt:   time.Now().Add(config.Window).Unix(),
		}, nil
	}

	count, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil {
		return nil, err
	}

	limit := config.Max + config.Burst
	remaining := limit - int(count)
	if remaining < 0 {
		remaining = 0
	}

	info := &RateLimitInfo{
		Limit:     limit,
		Remaining: remaining,
		ResetAt:   time.Now().Add(config.Window).Unix(),
	}

	if remaining == 0 {
		info.RetryIn = int(config.Window.Seconds())
	}

	return info, nil
}

// RateLimiterFactoryConfig holds configuration for creating rate limiters
type RateLimiterFactoryConfig struct {
	// RedisEnabled enables Redis-based distributed rate limiting
	RedisEnabled bool
	// RedisClient is the Redis client for distributed rate limiting
	RedisClient *redis.Client
	// RedisKeyPrefix is the prefix for rate limit keys in Redis
	RedisKeyPrefix string
	// RedisWindow is the sliding window duration for Redis rate limiter
	RedisWindow time.Duration
	// RedisMax is the maximum requests per window for Redis rate limiter
	RedisMax int
	// RedisFallback enables fallback to in-memory when Redis is unavailable
	RedisFallback bool
	// MemoryStore is the in-memory store for fallback rate limiting
	MemoryStore RateLimitStore
	// Config is the base rate limit configuration
	Config RateLimitConfig
}

// HybridRateLimiter provides rate limiting with Redis as primary and memory as fallback
type HybridRateLimiter struct {
	redisLimiter  *ratelimit.RedisLimiter
	memoryStore   RateLimitStore
	config        RateLimitConfig
	useRedis      bool
	redisFallback bool
}

// NewHybridRateLimiter creates a new hybrid rate limiter
func NewHybridRateLimiter(factoryConfig RateLimiterFactoryConfig) *HybridRateLimiter {
	var redisLimiter *ratelimit.RedisLimiter

	if factoryConfig.RedisEnabled && factoryConfig.RedisClient != nil {
		// Create memory limiter for fallback
		var memLimiter *ratelimit.Limiter
		if factoryConfig.RedisFallback && factoryConfig.MemoryStore != nil {
			// Create a wrapper cache for the memory store
			memLimiter = ratelimit.NewLimiter(&memoryStoreCache{store: factoryConfig.MemoryStore})
		}

		redisConfig := ratelimit.RedisLimiterConfig{
			Enabled:         true,
			KeyPrefix:       factoryConfig.RedisKeyPrefix,
			DefaultWindow:   factoryConfig.RedisWindow,
			DefaultMax:      factoryConfig.RedisMax,
			FallbackOnError: factoryConfig.RedisFallback,
		}

		redisLimiter = ratelimit.NewRedisLimiter(factoryConfig.RedisClient, redisConfig, memLimiter)
	}

	return &HybridRateLimiter{
		redisLimiter:  redisLimiter,
		memoryStore:   factoryConfig.MemoryStore,
		config:        factoryConfig.Config,
		useRedis:      factoryConfig.RedisEnabled && factoryConfig.RedisClient != nil,
		redisFallback: factoryConfig.RedisFallback,
	}
}

// memoryStoreCache wraps RateLimitStore to implement ratelimit.Cache
type memoryStoreCache struct {
	store RateLimitStore
}

func (c *memoryStoreCache) IncrementWithExpiry(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	return c.store.IncrementWithExpiry(ctx, key, ttl)
}

func (c *memoryStoreCache) Get(ctx context.Context, key string) ([]byte, error) {
	return c.store.Get(ctx, key)
}

// Middleware returns the hybrid rate limiting middleware
func (h *HybridRateLimiter) Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !h.config.Enabled {
			return c.Next()
		}

		// Check if path should skip rate limiting
		path := c.Path()
		for _, skipPath := range h.config.SkipPaths {
			if strings.HasPrefix(path, skipPath) {
				return c.Next()
			}
		}

		// Generate key
		key := h.config.KeyGenerator(c)
		if key == "" {
			return c.Next()
		}

		// Calculate effective limit
		limit := h.config.Max + h.config.Burst
		if userID := c.Locals("userID"); userID != nil {
			limit = int(float64(limit) * h.config.AuthMultiplier)
			if uid, ok := userID.(uuid.UUID); ok {
				key = fmt.Sprintf("user:%s:%s", uid.String(), key)
			}
		} else {
			key = fmt.Sprintf("ip:%s", key)
		}

		ctx := c.Context()

		// Try Redis first if enabled and available
		if h.useRedis && h.redisLimiter != nil && h.redisLimiter.IsAvailable() {
			result, err := h.redisLimiter.Check(ctx, "global:"+key, limit, h.config.Window)
			if err == nil {
				// Set headers
				c.Set("X-RateLimit-Limit", strconv.Itoa(result.Limit))
				c.Set("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
				c.Set("X-RateLimit-Reset", strconv.FormatInt(result.ResetAt/1000, 10)) // Convert ms to seconds
				c.Set("X-RateLimit-Backend", "redis")

				if !result.Allowed {
					retryAfter := int(result.RetryAfter / 1000) // Convert ms to seconds
					if retryAfter < 1 {
						retryAfter = 1
					}
					c.Set("Retry-After", strconv.Itoa(retryAfter))
					return h.config.OnLimitReached(c)
				}

				return c.Next()
			}
			// Fall through to memory store if Redis fails and fallback is enabled
			if !h.redisFallback {
				// Fail open if no fallback
				return c.Next()
			}
		}

		// Use memory store
		if h.memoryStore == nil {
			return c.Next()
		}

		fullKey := fmt.Sprintf("ratelimit:global:%s", key)

		count, err := h.memoryStore.IncrementWithExpiry(ctx, fullKey, h.config.Window)
		if err != nil {
			// Fail open
			return c.Next()
		}

		remaining := limit - int(count)
		if remaining < 0 {
			remaining = 0
		}
		resetAt := time.Now().Add(h.config.Window).Unix()

		// Set headers
		c.Set("X-RateLimit-Limit", strconv.Itoa(limit))
		c.Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Set("X-RateLimit-Reset", strconv.FormatInt(resetAt, 10))
		c.Set("X-RateLimit-Backend", "memory")

		if int(count) > limit {
			retryAfter := int(h.config.Window.Seconds())
			c.Set("Retry-After", strconv.Itoa(retryAfter))
			return h.config.OnLimitReached(c)
		}

		return c.Next()
	}
}

// IsRedisAvailable returns whether Redis is currently being used for rate limiting
func (h *HybridRateLimiter) IsRedisAvailable() bool {
	return h.useRedis && h.redisLimiter != nil && h.redisLimiter.IsAvailable()
}

// HealthCheck performs a health check on the rate limiter
func (h *HybridRateLimiter) HealthCheck(ctx context.Context) error {
	if h.redisLimiter != nil {
		return h.redisLimiter.HealthCheck(ctx)
	}
	return nil
}

// RedisRateLimiter provides rate limiting using Redis with sliding window algorithm
type RedisRateLimiter struct {
	limiter *ratelimit.RedisLimiter
	config  RateLimitConfig
}

// NewRedisRateLimiter creates a new Redis-backed rate limiter
func NewRedisRateLimiter(client *redis.Client, config RateLimitConfig, redisConfig ratelimit.RedisLimiterConfig) *RedisRateLimiter {
	limiter := ratelimit.NewRedisLimiter(client, redisConfig, nil)
	return &RedisRateLimiter{
		limiter: limiter,
		config:  config,
	}
}

// Middleware returns the Redis rate limiting middleware
func (r *RedisRateLimiter) Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !r.config.Enabled {
			return c.Next()
		}

		// Check if path should skip rate limiting
		path := c.Path()
		for _, skipPath := range r.config.SkipPaths {
			if strings.HasPrefix(path, skipPath) {
				return c.Next()
			}
		}

		// Generate key
		key := r.config.KeyGenerator(c)
		if key == "" {
			return c.Next()
		}

		// Calculate effective limit
		limit := r.config.Max + r.config.Burst
		if userID := c.Locals("userID"); userID != nil {
			limit = int(float64(limit) * r.config.AuthMultiplier)
			if uid, ok := userID.(uuid.UUID); ok {
				key = fmt.Sprintf("user:%s:%s", uid.String(), key)
			}
		} else {
			key = fmt.Sprintf("ip:%s", key)
		}

		ctx := c.Context()
		result, err := r.limiter.Check(ctx, "global:"+key, limit, r.config.Window)
		if err != nil {
			// Fail open on error
			return c.Next()
		}

		// Set headers
		c.Set("X-RateLimit-Limit", strconv.Itoa(result.Limit))
		c.Set("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
		c.Set("X-RateLimit-Reset", strconv.FormatInt(result.ResetAt/1000, 10))

		if !result.Allowed {
			retryAfter := int(result.RetryAfter / 1000)
			if retryAfter < 1 {
				retryAfter = 1
			}
			c.Set("Retry-After", strconv.Itoa(retryAfter))
			return r.config.OnLimitReached(c)
		}

		return c.Next()
	}
}

// IsAvailable returns whether Redis is available for rate limiting
func (r *RedisRateLimiter) IsAvailable() bool {
	return r.limiter.IsAvailable()
}

// GetRedisLimiter returns the underlying Redis limiter
func (r *RedisRateLimiter) GetRedisLimiter() *ratelimit.RedisLimiter {
	return r.limiter
}

// EndpointRedisRateLimiter provides per-endpoint rate limiting using Redis
type EndpointRedisRateLimiter struct {
	limiter  *ratelimit.RedisLimiter
	defaults EndpointConfig
	configs  map[string]EndpointConfig
}

// NewEndpointRedisRateLimiter creates a new Redis-backed endpoint rate limiter
func NewEndpointRedisRateLimiter(client *redis.Client, redisConfig ratelimit.RedisLimiterConfig, defaults EndpointConfig) *EndpointRedisRateLimiter {
	limiter := ratelimit.NewRedisLimiter(client, redisConfig, nil)
	return &EndpointRedisRateLimiter{
		limiter:  limiter,
		defaults: defaults,
		configs:  make(map[string]EndpointConfig),
	}
}

// Configure sets a custom rate limit for a specific endpoint pattern
func (er *EndpointRedisRateLimiter) Configure(pattern string, config EndpointConfig) {
	er.configs[pattern] = config
}

// ForEndpoint returns a middleware configured for specific endpoints
func (er *EndpointRedisRateLimiter) ForEndpoint(endpointName string) fiber.Handler {
	config, exists := er.configs[endpointName]
	if !exists {
		config = er.defaults
	}

	return func(c *fiber.Ctx) error {
		// Get user ID or IP for key
		var key string
		if userID := c.Locals("userID"); userID != nil {
			if uid, ok := userID.(uuid.UUID); ok {
				key = fmt.Sprintf("user:%s", uid.String())
			}
		}
		if key == "" {
			key = fmt.Sprintf("ip:%s", c.IP())
		}

		limit := config.Max + config.Burst
		ctx := c.Context()

		result, err := er.limiter.CheckEndpoint(ctx, endpointName, key, limit, config.Window)
		if err != nil {
			// Fail open
			return c.Next()
		}

		// Set headers
		c.Set("X-RateLimit-Limit", strconv.Itoa(result.Limit))
		c.Set("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
		c.Set("X-RateLimit-Reset", strconv.FormatInt(result.ResetAt/1000, 10))

		if !result.Allowed {
			retryAfter := int(result.RetryAfter / 1000)
			if retryAfter < 1 {
				retryAfter = 1
			}
			c.Set("Retry-After", strconv.Itoa(retryAfter))
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":    "rate_limited",
				"message":  fmt.Sprintf("Rate limit exceeded for %s. Please try again later.", endpointName),
				"endpoint": endpointName,
			})
		}

		return c.Next()
	}
}

// IsAvailable returns whether Redis is available
func (er *EndpointRedisRateLimiter) IsAvailable() bool {
	return er.limiter.IsAvailable()
}
