package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	ErrRateLimited = errors.New("rate limited")
)

// Cache interface for rate limiting storage
type Cache interface {
	IncrementWithExpiry(ctx context.Context, key string, ttl time.Duration) (int64, error)
	Get(ctx context.Context, key string) ([]byte, error)
}

// Limiter implements rate limiting
type Limiter struct {
	cache Cache
}

// NewLimiter creates a new rate limiter
func NewLimiter(cache Cache) *Limiter {
	return &Limiter{cache: cache}
}

// Config holds rate limit configuration
type Config struct {
	Limit  int           // Maximum requests
	Window time.Duration // Time window
}

// Standard rate limit configurations
var (
	// API rate limits
	APIDefault = Config{Limit: 100, Window: time.Minute}
	APIAuth    = Config{Limit: 5, Window: time.Minute}
	APIUpload  = Config{Limit: 10, Window: time.Minute}

	// Message rate limits
	MessageSend     = Config{Limit: 5, Window: 5 * time.Second}
	MessageEdit     = Config{Limit: 10, Window: time.Minute}
	MessageReaction = Config{Limit: 20, Window: time.Minute}

	// Server rate limits
	ServerCreate = Config{Limit: 10, Window: time.Hour}
	InviteCreate = Config{Limit: 10, Window: time.Minute}
)

// Check checks if the action is allowed
func (l *Limiter) Check(ctx context.Context, key string, cfg Config) error {
	count, err := l.cache.IncrementWithExpiry(ctx, "ratelimit:"+key, cfg.Window)
	if err != nil {
		// If cache fails, allow the request (fail open)
		return nil
	}

	if int(count) > cfg.Limit {
		return ErrRateLimited
	}

	return nil
}

// CheckUser checks rate limit for a user action
func (l *Limiter) CheckUser(ctx context.Context, userID uuid.UUID, action string, cfg Config) error {
	key := fmt.Sprintf("user:%s:%s", userID, action)
	return l.Check(ctx, key, cfg)
}

// CheckChannel checks rate limit for a channel action
func (l *Limiter) CheckChannel(ctx context.Context, userID, channelID uuid.UUID, action string, cfg Config) error {
	key := fmt.Sprintf("channel:%s:%s:%s", channelID, userID, action)
	return l.Check(ctx, key, cfg)
}

// CheckIP checks rate limit for an IP address
func (l *Limiter) CheckIP(ctx context.Context, ip string, action string, cfg Config) error {
	key := fmt.Sprintf("ip:%s:%s", ip, action)
	return l.Check(ctx, key, cfg)
}

// CheckSlowmode checks slowmode for a channel
func (l *Limiter) CheckSlowmode(ctx context.Context, userID, channelID uuid.UUID, slowmodeSeconds int) error {
	if slowmodeSeconds <= 0 {
		return nil
	}

	key := fmt.Sprintf("slowmode:%s:%s", channelID, userID)
	
	// Check if user has sent a message recently
	_, err := l.cache.Get(ctx, key)
	if err == nil {
		// Key exists, user is in slowmode
		return ErrRateLimited
	}

	// Set the key with slowmode duration
	cfg := Config{Limit: 1, Window: time.Duration(slowmodeSeconds) * time.Second}
	return l.Check(ctx, key, cfg)
}

// GetRemainingRequests returns how many requests are remaining
func (l *Limiter) GetRemainingRequests(ctx context.Context, key string, cfg Config) (int, error) {
	count, err := l.cache.IncrementWithExpiry(ctx, "ratelimit:"+key, cfg.Window)
	if err != nil {
		return cfg.Limit, nil
	}

	remaining := cfg.Limit - int(count) + 1 // +1 because we just incremented
	if remaining < 0 {
		remaining = 0
	}

	return remaining, nil
}

// RateLimitInfo contains rate limit information for responses
type RateLimitInfo struct {
	Limit     int   `json:"limit"`
	Remaining int   `json:"remaining"`
	ResetAt   int64 `json:"reset_at"`
}

// GetInfo returns rate limit information for headers
func (l *Limiter) GetInfo(ctx context.Context, key string, cfg Config) (*RateLimitInfo, error) {
	remaining, err := l.GetRemainingRequests(ctx, key, cfg)
	if err != nil {
		return nil, err
	}

	return &RateLimitInfo{
		Limit:     cfg.Limit,
		Remaining: remaining,
		ResetAt:   time.Now().Add(cfg.Window).Unix(),
	}, nil
}
