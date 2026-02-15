package ratelimit

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockCache implements Cache interface for testing
type MockCache struct {
	mu       sync.Mutex
	counters map[string]int64
	data     map[string][]byte
	expiries map[string]time.Time
	failNext bool
}

func NewMockCache() *MockCache {
	return &MockCache{
		counters: make(map[string]int64),
		data:     make(map[string][]byte),
		expiries: make(map[string]time.Time),
	}
}

func (m *MockCache) IncrementWithExpiry(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failNext {
		m.failNext = false
		return 0, errors.New("cache error")
	}

	// Check if key expired
	if exp, ok := m.expiries[key]; ok && time.Now().After(exp) {
		delete(m.counters, key)
		delete(m.expiries, key)
	}

	m.counters[key]++
	m.expiries[key] = time.Now().Add(ttl)

	return m.counters[key], nil
}

func (m *MockCache) Get(ctx context.Context, key string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failNext {
		m.failNext = false
		return nil, errors.New("cache error")
	}

	// Check if key expired
	if exp, ok := m.expiries[key]; ok && time.Now().After(exp) {
		delete(m.data, key)
		delete(m.expiries, key)
		return nil, errors.New("key not found")
	}

	if data, ok := m.data[key]; ok {
		return data, nil
	}
	return nil, errors.New("key not found")
}

func (m *MockCache) SetFailNext() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failNext = true
}

func (m *MockCache) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters = make(map[string]int64)
	m.data = make(map[string][]byte)
	m.expiries = make(map[string]time.Time)
}

func TestNewLimiter(t *testing.T) {
	cache := NewMockCache()
	limiter := NewLimiter(cache)

	assert.NotNil(t, limiter)
	assert.Equal(t, cache, limiter.cache)
}

func TestCheck_UnderLimit(t *testing.T) {
	cache := NewMockCache()
	limiter := NewLimiter(cache)
	ctx := context.Background()

	cfg := Config{Limit: 5, Window: time.Minute}

	// Should allow first 5 requests
	for i := 0; i < 5; i++ {
		err := limiter.Check(ctx, "test-key", cfg)
		assert.NoError(t, err, "request %d should be allowed", i+1)
	}
}

func TestCheck_OverLimit(t *testing.T) {
	cache := NewMockCache()
	limiter := NewLimiter(cache)
	ctx := context.Background()

	cfg := Config{Limit: 3, Window: time.Minute}

	// Use up the limit
	for i := 0; i < 3; i++ {
		err := limiter.Check(ctx, "test-key", cfg)
		require.NoError(t, err)
	}

	// Next request should be rate limited
	err := limiter.Check(ctx, "test-key", cfg)
	assert.Error(t, err)
	assert.Equal(t, ErrRateLimited, err)
}

func TestCheck_DifferentKeys(t *testing.T) {
	cache := NewMockCache()
	limiter := NewLimiter(cache)
	ctx := context.Background()

	cfg := Config{Limit: 2, Window: time.Minute}

	// Use up limit for key1
	for i := 0; i < 2; i++ {
		limiter.Check(ctx, "key1", cfg)
	}

	// key2 should still work
	err := limiter.Check(ctx, "key2", cfg)
	assert.NoError(t, err)

	// key1 should be limited
	err = limiter.Check(ctx, "key1", cfg)
	assert.Equal(t, ErrRateLimited, err)
}

func TestCheck_CacheFailure(t *testing.T) {
	cache := NewMockCache()
	limiter := NewLimiter(cache)
	ctx := context.Background()

	cfg := Config{Limit: 1, Window: time.Minute}

	// Set cache to fail
	cache.SetFailNext()

	// Should fail open (allow request)
	err := limiter.Check(ctx, "test-key", cfg)
	assert.NoError(t, err)
}

func TestCheckUser(t *testing.T) {
	cache := NewMockCache()
	limiter := NewLimiter(cache)
	ctx := context.Background()

	userID := uuid.New()
	cfg := Config{Limit: 2, Window: time.Minute}

	// First two requests should succeed
	err := limiter.CheckUser(ctx, userID, "send_message", cfg)
	assert.NoError(t, err)

	err = limiter.CheckUser(ctx, userID, "send_message", cfg)
	assert.NoError(t, err)

	// Third should fail
	err = limiter.CheckUser(ctx, userID, "send_message", cfg)
	assert.Equal(t, ErrRateLimited, err)

	// Different action should still work
	err = limiter.CheckUser(ctx, userID, "edit_message", cfg)
	assert.NoError(t, err)

	// Different user should still work
	otherUser := uuid.New()
	err = limiter.CheckUser(ctx, otherUser, "send_message", cfg)
	assert.NoError(t, err)
}

func TestCheckChannel(t *testing.T) {
	cache := NewMockCache()
	limiter := NewLimiter(cache)
	ctx := context.Background()

	userID := uuid.New()
	channelID := uuid.New()
	cfg := Config{Limit: 1, Window: time.Minute}

	// First request should succeed
	err := limiter.CheckChannel(ctx, userID, channelID, "message", cfg)
	assert.NoError(t, err)

	// Second should fail
	err = limiter.CheckChannel(ctx, userID, channelID, "message", cfg)
	assert.Equal(t, ErrRateLimited, err)

	// Same user, different channel should work
	otherChannel := uuid.New()
	err = limiter.CheckChannel(ctx, userID, otherChannel, "message", cfg)
	assert.NoError(t, err)
}

func TestCheckIP(t *testing.T) {
	cache := NewMockCache()
	limiter := NewLimiter(cache)
	ctx := context.Background()

	cfg := Config{Limit: 1, Window: time.Minute}

	// First request should succeed
	err := limiter.CheckIP(ctx, "192.168.1.1", "login", cfg)
	assert.NoError(t, err)

	// Second should fail
	err = limiter.CheckIP(ctx, "192.168.1.1", "login", cfg)
	assert.Equal(t, ErrRateLimited, err)

	// Different IP should work
	err = limiter.CheckIP(ctx, "192.168.1.2", "login", cfg)
	assert.NoError(t, err)
}

func TestCheckSlowmode_Disabled(t *testing.T) {
	cache := NewMockCache()
	limiter := NewLimiter(cache)
	ctx := context.Background()

	userID := uuid.New()
	channelID := uuid.New()

	// Slowmode of 0 should always allow
	err := limiter.CheckSlowmode(ctx, userID, channelID, 0)
	assert.NoError(t, err)

	// Negative should also allow
	err = limiter.CheckSlowmode(ctx, userID, channelID, -5)
	assert.NoError(t, err)
}

func TestCheckSlowmode_Enabled(t *testing.T) {
	cache := NewMockCache()
	limiter := NewLimiter(cache)
	ctx := context.Background()

	userID := uuid.New()
	channelID := uuid.New()
	slowmodeSeconds := 10

	// First message should succeed
	err := limiter.CheckSlowmode(ctx, userID, channelID, slowmodeSeconds)
	assert.NoError(t, err)

	// Second immediately after should fail
	err = limiter.CheckSlowmode(ctx, userID, channelID, slowmodeSeconds)
	assert.Equal(t, ErrRateLimited, err)
}

func TestGetRemainingRequests(t *testing.T) {
	cache := NewMockCache()
	limiter := NewLimiter(cache)
	ctx := context.Background()

	cfg := Config{Limit: 5, Window: time.Minute}

	// Initially should have 5 remaining (minus the check itself)
	remaining, err := limiter.GetRemainingRequests(ctx, "test-key", cfg)
	assert.NoError(t, err)
	assert.Equal(t, 5, remaining)

	// After using one more
	remaining, err = limiter.GetRemainingRequests(ctx, "test-key", cfg)
	assert.NoError(t, err)
	assert.Equal(t, 4, remaining)

	// Use up remaining
	for i := 0; i < 3; i++ {
		limiter.GetRemainingRequests(ctx, "test-key", cfg)
	}

	// Should be 0 remaining
	remaining, err = limiter.GetRemainingRequests(ctx, "test-key", cfg)
	assert.NoError(t, err)
	assert.Equal(t, 0, remaining)
}

func TestGetRemainingRequests_CacheFailure(t *testing.T) {
	cache := NewMockCache()
	limiter := NewLimiter(cache)
	ctx := context.Background()

	cfg := Config{Limit: 5, Window: time.Minute}

	cache.SetFailNext()

	// Should return full limit on cache failure
	remaining, err := limiter.GetRemainingRequests(ctx, "test-key", cfg)
	assert.NoError(t, err)
	assert.Equal(t, cfg.Limit, remaining)
}

func TestGetInfo(t *testing.T) {
	cache := NewMockCache()
	limiter := NewLimiter(cache)
	ctx := context.Background()

	cfg := Config{Limit: 10, Window: time.Minute}

	info, err := limiter.GetInfo(ctx, "test-key", cfg)
	require.NoError(t, err)

	assert.Equal(t, 10, info.Limit)
	assert.Equal(t, 10, info.Remaining)
	assert.Greater(t, info.ResetAt, time.Now().Unix())
	assert.LessOrEqual(t, info.ResetAt, time.Now().Add(cfg.Window).Unix()+1)
}

func TestPredefinedConfigs(t *testing.T) {
	// Verify predefined configs have sensible values
	testCases := []struct {
		name   string
		config Config
	}{
		{"APIDefault", APIDefault},
		{"APIAuth", APIAuth},
		{"APIUpload", APIUpload},
		{"MessageSend", MessageSend},
		{"MessageEdit", MessageEdit},
		{"MessageReaction", MessageReaction},
		{"ServerCreate", ServerCreate},
		{"InviteCreate", InviteCreate},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Greater(t, tc.config.Limit, 0, "limit should be positive")
			assert.Greater(t, tc.config.Window, time.Duration(0), "window should be positive")
		})
	}
}

func TestConfigValues(t *testing.T) {
	// Verify specific config values
	assert.Equal(t, 100, APIDefault.Limit)
	assert.Equal(t, time.Minute, APIDefault.Window)

	assert.Equal(t, 5, APIAuth.Limit)
	assert.Equal(t, time.Minute, APIAuth.Window)

	assert.Equal(t, 5, MessageSend.Limit)
	assert.Equal(t, 5*time.Second, MessageSend.Window)

	assert.Equal(t, 10, ServerCreate.Limit)
	assert.Equal(t, time.Hour, ServerCreate.Window)
}

func TestConcurrentAccess(t *testing.T) {
	cache := NewMockCache()
	limiter := NewLimiter(cache)
	ctx := context.Background()

	cfg := Config{Limit: 100, Window: time.Minute}

	var wg sync.WaitGroup
	var allowed, denied int
	var mu sync.Mutex

	// Spawn 150 concurrent requests
	for i := 0; i < 150; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := limiter.Check(ctx, "concurrent-key", cfg)
			mu.Lock()
			if err == nil {
				allowed++
			} else {
				denied++
			}
			mu.Unlock()
		}()
	}

	wg.Wait()

	// Should have allowed exactly 100 and denied 50
	assert.Equal(t, 100, allowed)
	assert.Equal(t, 50, denied)
}

func TestRateLimitInfo_JSONTags(t *testing.T) {
	info := RateLimitInfo{
		Limit:     100,
		Remaining: 50,
		ResetAt:   1234567890,
	}

	// Verify struct can be created and accessed
	assert.Equal(t, 100, info.Limit)
	assert.Equal(t, 50, info.Remaining)
	assert.Equal(t, int64(1234567890), info.ResetAt)
}

func TestErrRateLimited(t *testing.T) {
	// Verify error message
	assert.Equal(t, "rate limited", ErrRateLimited.Error())
}

func TestCheckKeyPrefix(t *testing.T) {
	cache := NewMockCache()
	limiter := NewLimiter(cache)
	ctx := context.Background()

	cfg := Config{Limit: 1, Window: time.Minute}

	// Check adds "ratelimit:" prefix
	_ = limiter.Check(ctx, "my-key", cfg)

	// Verify the key in cache has prefix
	cache.mu.Lock()
	_, exists := cache.counters["ratelimit:my-key"]
	cache.mu.Unlock()

	assert.True(t, exists, "key should have ratelimit: prefix")
}
