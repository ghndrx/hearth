package ratelimit

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// ComponentRateLimiter implements ratelimit.ComponentRateLimiter interface
type ComponentRateLimiter struct {
	limiter *Limiter
}

// NewComponentRateLimiter creates a new component rate limiter
func NewComponentRateLimiter(limiter *Limiter) *ComponentRateLimiter {
	return &ComponentRateLimiter{limiter: limiter}
}

// CheckComponentInteraction checks rate limit for component interactions
func (r *ComponentRateLimiter) CheckComponentInteraction(ctx context.Context, userID uuid.UUID) error {
	key := fmt.Sprintf("component:%s", userID)
	return r.limiter.Check(ctx, key, ComponentInteraction)
}

// CheckModalSubmit checks rate limit for modal submissions
func (r *ComponentRateLimiter) CheckModalSubmit(ctx context.Context, userID uuid.UUID) error {
	key := fmt.Sprintf("modal:%s", userID)
	return r.limiter.Check(ctx, key, ModalSubmit)
}
