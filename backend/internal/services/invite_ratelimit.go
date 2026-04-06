package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"hearth/internal/ratelimit"
)

// DefaultInviteRateLimit is the default maximum invites per user per hour
const DefaultInviteRateLimit = 5

// DefaultInviteRateWindow is the default time window for invite rate limiting
const DefaultInviteRateWindow = 1 * time.Hour

// InviteRateLimiterConfig holds configuration for invite rate limiting
type InviteRateLimiterConfig struct {
	// MaxInvites is the maximum number of invites allowed per window
	MaxInvites int
	// Window is the duration of the rate limit window
	Window time.Duration
}

// DefaultInviteRateLimiterConfig returns the default configuration
func DefaultInviteRateLimiterConfig() InviteRateLimiterConfig {
	return InviteRateLimiterConfig{
		MaxInvites: DefaultInviteRateLimit,
		Window:     DefaultInviteRateWindow,
	}
}

// RedisInviteRateLimiter implements InviteRateLimiter using Redis
type RedisInviteRateLimiter struct {
	redisLimiter *ratelimit.RedisLimiter
	config       InviteRateLimiterConfig
}

// NewRedisInviteRateLimiter creates a new Redis-backed invite rate limiter
func NewRedisInviteRateLimiter(redisLimiter *ratelimit.RedisLimiter, config InviteRateLimiterConfig) *RedisInviteRateLimiter {
	if config.MaxInvites == 0 {
		config.MaxInvites = DefaultInviteRateLimit
	}
	if config.Window == 0 {
		config.Window = DefaultInviteRateWindow
	}
	return &RedisInviteRateLimiter{
		redisLimiter: redisLimiter,
		config:       config,
	}
}

// CheckInviteCreation checks if the user can create an invite
func (r *RedisInviteRateLimiter) CheckInviteCreation(ctx context.Context, userID uuid.UUID) error {
	result, err := r.redisLimiter.CheckUser(ctx, userID, "invite", r.config.MaxInvites, r.config.Window)
	if err != nil {
		// On error, allow the request (fail open)
		return nil
	}

	if !result.Allowed {
		return ErrInviteRateLimited
	}

	return nil
}

// MemoryInviteRateLimiter implements InviteRateLimiter using in-memory storage
type MemoryInviteRateLimiter struct {
	config InviteRateLimiterConfig
	// userWindows stores the start time of each user's current window
	userWindows map[uuid.UUID]windowState
}

type windowState struct {
	start     time.Time
	count     int
	windowEnd time.Time
}

// NewMemoryInviteRateLimiter creates a new in-memory invite rate limiter
func NewMemoryInviteRateLimiter(config InviteRateLimiterConfig) *MemoryInviteRateLimiter {
	if config.MaxInvites == 0 {
		config.MaxInvites = DefaultInviteRateLimit
	}
	if config.Window == 0 {
		config.Window = DefaultInviteRateWindow
	}
	return &MemoryInviteRateLimiter{
		config:     config,
		userWindows: make(map[uuid.UUID]windowState),
	}
}

// CheckInviteCreation checks if the user can create an invite
func (r *MemoryInviteRateLimiter) CheckInviteCreation(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()

	state, exists := r.userWindows[userID]
	if !exists || now.After(state.windowEnd) {
		// Start a new window
		r.userWindows[userID] = windowState{
			start:     now,
			count:     1,
			windowEnd: now.Add(r.config.Window),
		}
		return nil
	}

	// Within window, check count
	if state.count >= r.config.MaxInvites {
		return ErrInviteRateLimited
	}

	// Increment count
	state.count++
	r.userWindows[userID] = state
	return nil
}

// CacheInviteRateLimiter implements InviteRateLimiter using the cache service
// This provides persistence across service restarts
type CacheInviteRateLimiter struct {
	cache   CacheService
	config  InviteRateLimiterConfig
}

// NewCacheInviteRateLimiter creates a new cache-backed invite rate limiter
func NewCacheInviteRateLimiter(cache CacheService, config InviteRateLimiterConfig) *CacheInviteRateLimiter {
	if config.MaxInvites == 0 {
		config.MaxInvites = DefaultInviteRateLimit
	}
	if config.Window == 0 {
		config.Window = DefaultInviteRateWindow
	}
	return &CacheInviteRateLimiter{
		cache:  cache,
		config: config,
	}
}

// cacheKey returns the cache key for a user's invite rate limit
func (r *CacheInviteRateLimiter) cacheKey(userID uuid.UUID) string {
	return fmt.Sprintf("invite_ratelimit:%s", userID.String())
}

// CheckInviteCreation checks if the user can create an invite
func (r *CacheInviteRateLimiter) CheckInviteCreation(ctx context.Context, userID uuid.UUID) error {
	key := r.cacheKey(userID)

	// Try to get existing count
	data, err := r.cache.Get(ctx, key)
	if err != nil || len(data) == 0 {
		// No existing entry, allow and set initial count
		err := r.cache.Set(ctx, key, []byte("1"), r.config.Window)
		if err != nil {
			// Cache error, allow the request
			return nil
		}
		return nil
	}

	// Parse existing count - data is a byte slice
	var count int
	_, parseErr := fmt.Sscanf(string(data), "%d", &count)
	if parseErr != nil {
		// Invalid data, reset
		r.cache.Set(ctx, key, []byte("1"), r.config.Window)
		return nil
	}

	if count >= r.config.MaxInvites {
		return ErrInviteRateLimited
	}

	// Increment count - note: this is not atomic, but for rate limiting it's acceptable
	newCount := count + 1
	r.cache.Set(ctx, key, []byte(fmt.Sprintf("%d", newCount)), r.config.Window)
	return nil
}
