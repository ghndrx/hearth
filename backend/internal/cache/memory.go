package cache

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"hearth/internal/models"
)

// MemoryCache implements a simple in-memory cache for single-instance deployments
// or as a fallback when Redis is unavailable
type MemoryCache struct {
	mu     sync.RWMutex
	items  map[string]*cacheItem
	stop   chan struct{}
	closed bool
}

type cacheItem struct {
	value     []byte
	expiresAt time.Time
}

// NewMemoryCache creates a new in-memory cache
func NewMemoryCache() *MemoryCache {
	mc := &MemoryCache{
		items: make(map[string]*cacheItem),
		stop:  make(chan struct{}),
	}
	// Start background cleanup goroutine
	go mc.cleanup()
	return mc
}

// cleanup periodically removes expired items
func (c *MemoryCache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.mu.Lock()
			now := time.Now()
			for k, v := range c.items {
				if now.After(v.expiresAt) {
					delete(c.items, k)
				}
			}
			c.mu.Unlock()
		case <-c.stop:
			return
		}
	}
}

// Close stops the cleanup goroutine
func (c *MemoryCache) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.closed {
		close(c.stop)
		c.closed = true
	}
	return nil
}

// Generic operations

func (c *MemoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, ok := c.items[key]
	if !ok {
		return nil, ErrCacheMiss
	}

	if time.Now().After(item.expiresAt) {
		return nil, ErrCacheMiss
	}

	return item.value, nil
}

func (c *MemoryCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = &cacheItem{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
	return nil
}

func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
	return nil
}

// User caching (stub implementations for CacheService interface)

func (c *MemoryCache) GetUser(ctx context.Context, id uuid.UUID) (*models.User, error) {
	return nil, ErrCacheMiss
}

func (c *MemoryCache) SetUser(ctx context.Context, user *models.User, ttl time.Duration) error {
	return nil // No-op for memory cache
}

func (c *MemoryCache) DeleteUser(ctx context.Context, id uuid.UUID) error {
	return nil
}

// Server caching

func (c *MemoryCache) GetServer(ctx context.Context, id uuid.UUID) (*models.Server, error) {
	return nil, ErrCacheMiss
}

func (c *MemoryCache) SetServer(ctx context.Context, server *models.Server, ttl time.Duration) error {
	return nil
}

func (c *MemoryCache) DeleteServer(ctx context.Context, id uuid.UUID) error {
	return nil
}

// Channel caching

func (c *MemoryCache) GetChannel(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
	return nil, ErrCacheMiss
}

func (c *MemoryCache) SetChannel(ctx context.Context, channel *models.Channel, ttl time.Duration) error {
	return nil
}

func (c *MemoryCache) DeleteChannel(ctx context.Context, id uuid.UUID) error {
	return nil
}
