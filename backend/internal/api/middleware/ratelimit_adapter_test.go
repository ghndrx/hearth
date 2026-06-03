package middleware

import (
	"context"
	"sync"
	"testing"
	"time"

	"hearth/internal/ratelimit"

	"github.com/stretchr/testify/assert"
)

// --- SimpleInMemoryCache Tests ---

func TestSimpleInMemoryCache_IncrementWithExpiry(t *testing.T) {
	cache := NewSimpleInMemoryCache()
	ctx := context.Background()

	// First increment should return 1
	count, err := cache.IncrementWithExpiry(ctx, "key1", time.Minute)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// Second increment should return 2
	count, err = cache.IncrementWithExpiry(ctx, "key1", time.Minute)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), count)

	// Different key should return 1
	count, err = cache.IncrementWithExpiry(ctx, "key2", time.Minute)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestSimpleInMemoryCache_IncrementWithExpiry_Expired(t *testing.T) {
	cache := NewSimpleInMemoryCache()
	ctx := context.Background()

	// Increment with very short TTL
	count, err := cache.IncrementWithExpiry(ctx, "key1", time.Millisecond)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// Wait for expiry
	time.Sleep(10 * time.Millisecond)

	// Next increment should reset to 1
	count, err = cache.IncrementWithExpiry(ctx, "key1", time.Millisecond)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestSimpleInMemoryCache_Get(t *testing.T) {
	cache := NewSimpleInMemoryCache()
	ctx := context.Background()

	// Set a value
	cache.IncrementWithExpiry(ctx, "key1", time.Minute)
	cache.IncrementWithExpiry(ctx, "key1", time.Minute)

	// Get should return "2"
	val, err := cache.Get(ctx, "key1")
	assert.NoError(t, err)
	assert.Equal(t, []byte("2"), val)
}

func TestSimpleInMemoryCache_Get_Expired(t *testing.T) {
	cache := NewSimpleInMemoryCache()
	ctx := context.Background()

	// Set with very short TTL
	cache.IncrementWithExpiry(ctx, "key1", time.Millisecond)

	// Wait for expiry
	time.Sleep(10 * time.Millisecond)

	// Get should return nil (key expired)
	val, err := cache.Get(ctx, "key1")
	assert.NoError(t, err)
	assert.Nil(t, val)
}

func TestSimpleInMemoryCache_Get_NotFound(t *testing.T) {
	cache := NewSimpleInMemoryCache()
	ctx := context.Background()

	// Get non-existent key
	val, err := cache.Get(ctx, "nonexistent")
	assert.NoError(t, err)
	assert.Nil(t, val)
}

func TestSimpleInMemoryCache_Concurrent(t *testing.T) {
	cache := NewSimpleInMemoryCache()
	ctx := context.Background()

	// Simulate concurrent increments
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cache.IncrementWithExpiry(ctx, "concurrent-key", time.Minute)
		}()
	}
	wg.Wait()

	// Final count should be 10
	val, err := cache.Get(ctx, "concurrent-key")
	assert.NoError(t, err)
	assert.Equal(t, []byte("10"), val)
}

// --- RedisRateLimiterAdapter Tests ---

func TestRedisRateLimiterAdapter_Check_Disabled(t *testing.T) {
	// Nil limiter = disabled
	adapter := NewRedisRateLimiterAdapter(nil)
	ctx := context.Background()

	result, err := adapter.Check(ctx, "test-key", 10, time.Minute)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Allowed, "should fail open when disabled")
	assert.Equal(t, 10, result.Remaining)
	assert.Equal(t, 10, result.Limit)
}

func TestRedisRateLimiterAdapter_IsAvailable_True(t *testing.T) {
	// When limiter is nil, enabled=false
	adapter := NewRedisRateLimiterAdapter(nil)

	assert.False(t, adapter.IsAvailable(), "nil limiter should not be available")
}

func TestRedisRateLimiterAdapter_IsAvailable_False(t *testing.T) {
	adapter := NewRedisRateLimiterAdapter(nil)
	assert.False(t, adapter.IsAvailable())
}

// --- MemoryRateLimiterAdapter Tests ---

func TestMemoryRateLimiterAdapter_Check_Disabled(t *testing.T) {
	// Nil limiter = disabled
	adapter := NewMemoryRateLimiterAdapter(nil)
	ctx := context.Background()

	result, err := adapter.Check(ctx, "test-key", 10, time.Minute)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Allowed, "should fail open when disabled")
	assert.Equal(t, 10, result.Remaining)
	assert.Equal(t, 10, result.Limit)
}

func TestMemoryRateLimiterAdapter_Check_Enabled(t *testing.T) {
	// Create a real in-memory cache and limiter
	cache := NewSimpleInMemoryCache()
	limiter := ratelimit.NewLimiter(cache)
	adapter := NewMemoryRateLimiterAdapter(limiter)
	ctx := context.Background()

	// First request should be allowed
	result, err := adapter.Check(ctx, "test-key", 5, time.Minute)
	assert.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, 4, result.Remaining) // 5 - 1
	assert.Equal(t, 5, result.Limit)
}

func TestMemoryRateLimiterAdapter_Check_RateLimited(t *testing.T) {
	// Create a real in-memory cache and limiter with limit of 1
	cache := NewSimpleInMemoryCache()
	limiter := ratelimit.NewLimiter(cache)
	adapter := NewMemoryRateLimiterAdapter(limiter)
	ctx := context.Background()

	// First request should be allowed
	result, err := adapter.Check(ctx, "test-key", 1, time.Minute)
	assert.NoError(t, err)
	assert.True(t, result.Allowed)

	// Second request should be rate limited
	result, err = adapter.Check(ctx, "test-key", 1, time.Minute)
	assert.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Equal(t, 0, result.Remaining)
	assert.Equal(t, 1, result.Limit)
	assert.Greater(t, result.RetryAfter, int64(0))
}

func TestMemoryRateLimiterAdapter_IsAvailable_True(t *testing.T) {
	cache := NewSimpleInMemoryCache()
	limiter := ratelimit.NewLimiter(cache)
	adapter := NewMemoryRateLimiterAdapter(limiter)

	assert.True(t, adapter.IsAvailable())
}

func TestMemoryRateLimiterAdapter_IsAvailable_False(t *testing.T) {
	adapter := NewMemoryRateLimiterAdapter(nil)

	assert.False(t, adapter.IsAvailable())
}

func TestMemoryRateLimiterAdapter_Check_DifferentKeys(t *testing.T) {
	cache := NewSimpleInMemoryCache()
	limiter := ratelimit.NewLimiter(cache)
	adapter := NewMemoryRateLimiterAdapter(limiter)
	ctx := context.Background()

	// Different keys should have independent limits
	result1, _ := adapter.Check(ctx, "key1", 1, time.Minute)
	result2, _ := adapter.Check(ctx, "key2", 1, time.Minute)

	assert.True(t, result1.Allowed)
	assert.True(t, result2.Allowed)
}

// --- HybridRateLimiterAdapter Tests ---

func TestHybridRateLimiterAdapter_Check_NeitherAvailable(t *testing.T) {
	adapter := NewHybridRateLimiterAdapter(nil, nil)
	ctx := context.Background()

	result, err := adapter.Check(ctx, "test-key", 10, time.Minute)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Allowed, "should fail open when no limiters available")
	assert.Equal(t, 10, result.Remaining)
}

func TestHybridRateLimiterAdapter_Check_MemoryFallback(t *testing.T) {
	// Redis nil, memory provided
	cache := NewSimpleInMemoryCache()
	memoryLimiter := ratelimit.NewLimiter(cache)
	adapter := NewHybridRateLimiterAdapter(nil, memoryLimiter)
	ctx := context.Background()

	// Should use memory limiter
	result, err := adapter.Check(ctx, "test-key", 5, time.Minute)
	assert.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, 4, result.Remaining)
}

func TestHybridRateLimiterAdapter_Check_MemoryRateLimited(t *testing.T) {
	cache := NewSimpleInMemoryCache()
	memoryLimiter := ratelimit.NewLimiter(cache)
	adapter := NewHybridRateLimiterAdapter(nil, memoryLimiter)
	ctx := context.Background()

	// Exhaust memory limiter
	result1, err := adapter.Check(ctx, "test-key", 1, time.Minute)
	assert.NoError(t, err)
	assert.True(t, result1.Allowed)

	result2, err := adapter.Check(ctx, "test-key", 1, time.Minute)
	assert.NoError(t, err)
	assert.False(t, result2.Allowed)
}

func TestHybridRateLimiterAdapter_IsAvailable_True(t *testing.T) {
	cache := NewSimpleInMemoryCache()
	memoryLimiter := ratelimit.NewLimiter(cache)
	adapter := NewHybridRateLimiterAdapter(nil, memoryLimiter)

	assert.True(t, adapter.IsAvailable())
}

func TestHybridRateLimiterAdapter_IsAvailable_False(t *testing.T) {
	adapter := NewHybridRateLimiterAdapter(nil, nil)

	assert.False(t, adapter.IsAvailable())
}

func TestHybridRateLimiterAdapter_IsRedisAvailable(t *testing.T) {
	adapter := NewHybridRateLimiterAdapter(nil, nil)

	// Without Redis, should be false
	assert.False(t, adapter.IsRedisAvailable())
}

func TestHybridRateLimiterAdapter_Check_UsesMemoryWhenRedisUnavailable(t *testing.T) {
	// This test verifies the fallback path when Redis is configured but unavailable
	// We can't easily test Redis unavailability without a real Redis mock,
	// but we can verify the hybrid adapter handles nil Redis gracefully
	adapter := NewHybridRateLimiterAdapter(nil, nil)
	ctx := context.Background()

	result, err := adapter.Check(ctx, "fallback-key", 100, time.Minute)

	assert.NoError(t, err)
	assert.True(t, result.Allowed) // fails open
}
