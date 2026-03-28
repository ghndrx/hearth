package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockRedisClient implements RedisClient for testing
type MockRedisClient struct {
	mu            sync.Mutex
	scripts       map[string]string
	data          map[string]interface{}
	sortedSets    map[string]map[string]float64
	failNext      bool
	failCount     int
	pingErr       error
	scriptLoadErr error
	evalErr       error
}

func NewMockRedisClient() *MockRedisClient {
	return &MockRedisClient{
		scripts:    make(map[string]string),
		data:       make(map[string]interface{}),
		sortedSets: make(map[string]map[string]float64),
	}
}

func (m *MockRedisClient) ScriptLoad(ctx context.Context, script string) *redis.StringCmd {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.scriptLoadErr != nil {
		cmd := redis.NewStringCmd(ctx)
		cmd.SetErr(m.scriptLoadErr)
		return cmd
	}

	sha := fmt.Sprintf("sha_%d", len(m.scripts))
	m.scripts[sha] = script

	cmd := redis.NewStringCmd(ctx)
	cmd.SetVal(sha)
	return cmd
}

func (m *MockRedisClient) EvalSha(ctx context.Context, sha string, keys []string, args ...interface{}) *redis.Cmd {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failNext || m.failCount > 0 {
		m.failNext = false
		if m.failCount > 0 {
			m.failCount--
		}
		cmd := redis.NewCmd(ctx)
		cmd.SetErr(errors.New("redis error"))
		return cmd
	}

	if m.evalErr != nil {
		cmd := redis.NewCmd(ctx)
		cmd.SetErr(m.evalErr)
		return cmd
	}

	// Check if script exists
	if _, ok := m.scripts[sha]; !ok {
		cmd := redis.NewCmd(ctx)
		cmd.SetErr(errors.New("NOSCRIPT No matching script. Please use EVAL."))
		return cmd
	}

	// Simulate sliding window behavior
	key := keys[0]
	nowMicro := args[0].(int64)
	windowMicro := args[1].(int64)
	limit := args[2].(int)

	// Initialize sorted set if needed
	if m.sortedSets[key] == nil {
		m.sortedSets[key] = make(map[string]float64)
	}

	// Remove expired entries
	windowStart := float64(nowMicro - windowMicro)
	for member, score := range m.sortedSets[key] {
		if score < windowStart {
			delete(m.sortedSets[key], member)
		}
	}

	// Count current requests
	currentCount := len(m.sortedSets[key])

	// Check if we would exceed limit
	if currentCount >= limit {
		resetAt := (nowMicro + windowMicro) / 1000
		ttlMs := windowMicro / 1000
		cmd := redis.NewCmd(ctx)
		cmd.SetVal([]interface{}{int64(0), int64(currentCount), ttlMs, resetAt})
		return cmd
	}

	// Add new entry
	member := fmt.Sprintf("%d:%d", nowMicro, len(m.sortedSets[key]))
	m.sortedSets[key][member] = float64(nowMicro)

	newCount := len(m.sortedSets[key])
	resetAt := (nowMicro + windowMicro) / 1000
	ttlMs := windowMicro / 1000

	cmd := redis.NewCmd(ctx)
	cmd.SetVal([]interface{}{int64(1), int64(newCount), ttlMs, resetAt})
	return cmd
}

func (m *MockRedisClient) Ping(ctx context.Context) *redis.StatusCmd {
	m.mu.Lock()
	defer m.mu.Unlock()

	cmd := redis.NewStatusCmd(ctx)
	if m.pingErr != nil {
		cmd.SetErr(m.pingErr)
		return cmd
	}
	cmd.SetVal("PONG")
	return cmd
}

func (m *MockRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	m.mu.Lock()
	defer m.mu.Unlock()

	cmd := redis.NewStringCmd(ctx)
	if val, ok := m.data[key]; ok {
		cmd.SetVal(fmt.Sprintf("%v", val))
		return cmd
	}
	cmd.SetErr(redis.Nil)
	return cmd
}

func (m *MockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data[key] = value
	cmd := redis.NewStatusCmd(ctx)
	cmd.SetVal("OK")
	return cmd
}

func (m *MockRedisClient) Incr(ctx context.Context, key string) *redis.IntCmd {
	m.mu.Lock()
	defer m.mu.Unlock()

	val, ok := m.data[key].(int64)
	if !ok {
		val = 0
	}
	val++
	m.data[key] = val

	cmd := redis.NewIntCmd(ctx)
	cmd.SetVal(val)
	return cmd
}

func (m *MockRedisClient) Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	m.mu.Lock()
	defer m.mu.Unlock()

	cmd := redis.NewBoolCmd(ctx)
	cmd.SetVal(true)
	return cmd
}

func (m *MockRedisClient) SetFailNext() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failNext = true
}

func (m *MockRedisClient) SetFailCount(count int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failCount = count
}

func (m *MockRedisClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = make(map[string]interface{})
	m.sortedSets = make(map[string]map[string]float64)
	m.failNext = false
	m.failCount = 0
}

func (m *MockRedisClient) SetPingErr(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pingErr = err
}

func (m *MockRedisClient) SetScriptLoadErr(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.scriptLoadErr = err
}

func (m *MockRedisClient) SetEvalErr(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.evalErr = err
}

// Tests

func TestNewRedisLimiter(t *testing.T) {
	client := NewMockRedisClient()
	config := DefaultRedisLimiterConfig()

	limiter := NewRedisLimiter(client, config, nil)

	assert.NotNil(t, limiter)
	assert.Equal(t, client, limiter.client)
	assert.Equal(t, config.KeyPrefix, limiter.config.KeyPrefix)
}

func TestDefaultRedisLimiterConfig(t *testing.T) {
	config := DefaultRedisLimiterConfig()

	assert.True(t, config.Enabled)
	assert.Equal(t, "hearth:ratelimit:", config.KeyPrefix)
	assert.Equal(t, 60*time.Second, config.DefaultWindow)
	assert.Equal(t, 10000, config.DefaultMax)
	assert.True(t, config.FallbackOnError)
}

func TestRedisLimiter_Check_UnderLimit(t *testing.T) {
	client := NewMockRedisClient()
	config := DefaultRedisLimiterConfig()
	limiter := NewRedisLimiter(client, config, nil)

	// Wait for script to load
	time.Sleep(50 * time.Millisecond)

	ctx := context.Background()

	// Should allow requests under limit
	for i := 0; i < 5; i++ {
		result, err := limiter.Check(ctx, "test-key", 10, time.Minute)
		require.NoError(t, err)
		assert.True(t, result.Allowed, "request %d should be allowed", i+1)
		assert.Equal(t, 10, result.Limit)
		assert.Equal(t, 10-(i+1), result.Remaining)
	}
}

func TestRedisLimiter_Check_OverLimit(t *testing.T) {
	client := NewMockRedisClient()
	config := DefaultRedisLimiterConfig()
	limiter := NewRedisLimiter(client, config, nil)

	time.Sleep(50 * time.Millisecond)

	ctx := context.Background()

	// Use up the limit
	for i := 0; i < 3; i++ {
		result, err := limiter.Check(ctx, "test-key", 3, time.Minute)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
	}

	// Next request should be blocked
	result, err := limiter.Check(ctx, "test-key", 3, time.Minute)
	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Equal(t, 0, result.Remaining)
	assert.Greater(t, result.RetryAfter, int64(0))
}

func TestRedisLimiter_Check_DifferentKeys(t *testing.T) {
	client := NewMockRedisClient()
	config := DefaultRedisLimiterConfig()
	limiter := NewRedisLimiter(client, config, nil)

	time.Sleep(50 * time.Millisecond)

	ctx := context.Background()

	// Use up limit for key1
	for i := 0; i < 2; i++ {
		limiter.Check(ctx, "key1", 2, time.Minute)
	}

	// key2 should still work
	result, err := limiter.Check(ctx, "key2", 2, time.Minute)
	require.NoError(t, err)
	assert.True(t, result.Allowed)

	// key1 should be limited
	result, err = limiter.Check(ctx, "key1", 2, time.Minute)
	require.NoError(t, err)
	assert.False(t, result.Allowed)
}

func TestRedisLimiter_Check_Disabled(t *testing.T) {
	client := NewMockRedisClient()
	config := DefaultRedisLimiterConfig()
	config.Enabled = false
	limiter := NewRedisLimiter(client, config, nil)

	ctx := context.Background()

	// All requests should be allowed when disabled
	for i := 0; i < 100; i++ {
		result, err := limiter.Check(ctx, "test-key", 1, time.Minute)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
	}
}

func TestRedisLimiter_Check_WithFallback(t *testing.T) {
	client := NewMockRedisClient()
	client.SetScriptLoadErr(errors.New("connection refused"))

	config := DefaultRedisLimiterConfig()
	config.FallbackOnError = true

	// Create memory fallback
	memCache := NewMockCache()
	memLimiter := NewLimiter(memCache)

	limiter := NewRedisLimiter(client, config, memLimiter)

	time.Sleep(50 * time.Millisecond)

	ctx := context.Background()

	// Should use fallback and allow requests
	for i := 0; i < 3; i++ {
		result, err := limiter.Check(ctx, "test-key", 3, time.Minute)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
	}

	// Fallback should also rate limit
	result, err := limiter.Check(ctx, "test-key", 3, time.Minute)
	require.NoError(t, err)
	assert.False(t, result.Allowed)
}

func TestRedisLimiter_Check_FallbackDisabled(t *testing.T) {
	client := NewMockRedisClient()
	client.SetScriptLoadErr(errors.New("connection refused"))

	config := DefaultRedisLimiterConfig()
	config.FallbackOnError = false

	limiter := NewRedisLimiter(client, config, nil)

	time.Sleep(50 * time.Millisecond)

	ctx := context.Background()

	// Should return error when fallback is disabled
	_, err := limiter.Check(ctx, "test-key", 10, time.Minute)
	assert.Error(t, err)
	assert.Equal(t, ErrRedisUnavailable, err)
}

func TestRedisLimiter_CheckUser(t *testing.T) {
	client := NewMockRedisClient()
	config := DefaultRedisLimiterConfig()
	limiter := NewRedisLimiter(client, config, nil)

	time.Sleep(50 * time.Millisecond)

	ctx := context.Background()
	userID := uuid.New()

	// First request should succeed
	result, err := limiter.CheckUser(ctx, userID, "send_message", 2, time.Minute)
	require.NoError(t, err)
	assert.True(t, result.Allowed)

	// Second request should succeed
	result, err = limiter.CheckUser(ctx, userID, "send_message", 2, time.Minute)
	require.NoError(t, err)
	assert.True(t, result.Allowed)

	// Third should be blocked
	result, err = limiter.CheckUser(ctx, userID, "send_message", 2, time.Minute)
	require.NoError(t, err)
	assert.False(t, result.Allowed)

	// Different action should still work
	result, err = limiter.CheckUser(ctx, userID, "edit_message", 2, time.Minute)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
}

func TestRedisLimiter_CheckIP(t *testing.T) {
	client := NewMockRedisClient()
	config := DefaultRedisLimiterConfig()
	limiter := NewRedisLimiter(client, config, nil)

	time.Sleep(50 * time.Millisecond)

	ctx := context.Background()

	// First request should succeed
	result, err := limiter.CheckIP(ctx, "192.168.1.1", "login", 1, time.Minute)
	require.NoError(t, err)
	assert.True(t, result.Allowed)

	// Second should be blocked
	result, err = limiter.CheckIP(ctx, "192.168.1.1", "login", 1, time.Minute)
	require.NoError(t, err)
	assert.False(t, result.Allowed)

	// Different IP should work
	result, err = limiter.CheckIP(ctx, "192.168.1.2", "login", 1, time.Minute)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
}

func TestRedisLimiter_CheckEndpoint(t *testing.T) {
	client := NewMockRedisClient()
	config := DefaultRedisLimiterConfig()
	limiter := NewRedisLimiter(client, config, nil)

	time.Sleep(50 * time.Millisecond)

	ctx := context.Background()

	// Test endpoint rate limiting
	result, err := limiter.CheckEndpoint(ctx, "/api/login", "user123", 1, time.Minute)
	require.NoError(t, err)
	assert.True(t, result.Allowed)

	result, err = limiter.CheckEndpoint(ctx, "/api/login", "user123", 1, time.Minute)
	require.NoError(t, err)
	assert.False(t, result.Allowed)

	// Different endpoint should work
	result, err = limiter.CheckEndpoint(ctx, "/api/register", "user123", 1, time.Minute)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
}

func TestRedisLimiter_CheckChannel(t *testing.T) {
	client := NewMockRedisClient()
	config := DefaultRedisLimiterConfig()
	limiter := NewRedisLimiter(client, config, nil)

	time.Sleep(50 * time.Millisecond)

	ctx := context.Background()
	userID := uuid.New()
	channelID := uuid.New()

	result, err := limiter.CheckChannel(ctx, userID, channelID, "message", 1, time.Minute)
	require.NoError(t, err)
	assert.True(t, result.Allowed)

	result, err = limiter.CheckChannel(ctx, userID, channelID, "message", 1, time.Minute)
	require.NoError(t, err)
	assert.False(t, result.Allowed)

	// Different channel should work
	otherChannel := uuid.New()
	result, err = limiter.CheckChannel(ctx, userID, otherChannel, "message", 1, time.Minute)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
}

func TestRedisLimiter_HealthCheck(t *testing.T) {
	client := NewMockRedisClient()
	config := DefaultRedisLimiterConfig()
	limiter := NewRedisLimiter(client, config, nil)

	time.Sleep(50 * time.Millisecond)

	ctx := context.Background()

	// Should succeed
	err := limiter.HealthCheck(ctx)
	assert.NoError(t, err)
	assert.True(t, limiter.IsAvailable())
}

func TestRedisLimiter_HealthCheck_Failure(t *testing.T) {
	client := NewMockRedisClient()
	client.SetPingErr(errors.New("connection refused"))

	config := DefaultRedisLimiterConfig()
	limiter := NewRedisLimiter(client, config, nil)

	ctx := context.Background()

	err := limiter.HealthCheck(ctx)
	assert.Error(t, err)
	assert.False(t, limiter.IsAvailable())
}

func TestRedisLimiter_IsAvailable(t *testing.T) {
	client := NewMockRedisClient()
	config := DefaultRedisLimiterConfig()
	limiter := NewRedisLimiter(client, config, nil)

	// Initially should be available
	assert.True(t, limiter.IsAvailable())

	// After script load failure, should be unavailable
	limiter.setAvailable(false)
	assert.False(t, limiter.IsAvailable())

	limiter.setAvailable(true)
	assert.True(t, limiter.IsAvailable())
}

func TestRedisLimiter_NilClient(t *testing.T) {
	config := DefaultRedisLimiterConfig()
	memCache := NewMockCache()
	memLimiter := NewLimiter(memCache)

	limiter := NewRedisLimiter(nil, config, memLimiter)

	time.Sleep(50 * time.Millisecond)

	ctx := context.Background()

	// Should fall back to memory limiter
	result, err := limiter.Check(ctx, "test-key", 2, time.Minute)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
}

func TestRedisLimiter_ConcurrentAccess(t *testing.T) {
	client := NewMockRedisClient()
	config := DefaultRedisLimiterConfig()
	limiter := NewRedisLimiter(client, config, nil)

	time.Sleep(50 * time.Millisecond)

	ctx := context.Background()

	var wg sync.WaitGroup
	var allowed, denied int
	var mu sync.Mutex

	// Spawn 150 concurrent requests with limit of 100
	for i := 0; i < 150; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := limiter.Check(ctx, "concurrent-key", 100, time.Minute)
			if err != nil {
				return
			}
			mu.Lock()
			if result.Allowed {
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

func TestResult_Fields(t *testing.T) {
	result := &Result{
		Allowed:    true,
		Remaining:  5,
		ResetAt:    time.Now().UnixMilli(),
		RetryAfter: 0,
		Limit:      10,
	}

	assert.True(t, result.Allowed)
	assert.Equal(t, 5, result.Remaining)
	assert.Equal(t, 10, result.Limit)
	assert.Greater(t, result.ResetAt, int64(0))
}

// MultiLimiter Tests

func TestNewMultiLimiter(t *testing.T) {
	client := NewMockRedisClient()
	config := DefaultRedisLimiterConfig()
	redis := NewRedisLimiter(client, config, nil)

	memCache := NewMockCache()
	memory := NewLimiter(memCache)

	ml := NewMultiLimiter(redis, memory)

	assert.NotNil(t, ml)
	assert.True(t, ml.useRedis)
}

func TestMultiLimiter_Check_WithRedis(t *testing.T) {
	client := NewMockRedisClient()
	config := DefaultRedisLimiterConfig()
	redis := NewRedisLimiter(client, config, nil)

	time.Sleep(50 * time.Millisecond)

	memCache := NewMockCache()
	memory := NewLimiter(memCache)

	ml := NewMultiLimiter(redis, memory)
	ctx := context.Background()

	cfg := Config{Limit: 2, Window: time.Minute}

	// Should use Redis
	err := ml.Check(ctx, "test-key", cfg)
	assert.NoError(t, err)

	err = ml.Check(ctx, "test-key", cfg)
	assert.NoError(t, err)

	// Third should be rate limited
	err = ml.Check(ctx, "test-key", cfg)
	assert.Equal(t, ErrRateLimited, err)
}

func TestMultiLimiter_Check_FallbackToMemory(t *testing.T) {
	client := NewMockRedisClient()
	client.SetScriptLoadErr(errors.New("connection refused"))

	config := DefaultRedisLimiterConfig()
	redis := NewRedisLimiter(client, config, nil)

	memCache := NewMockCache()
	memory := NewLimiter(memCache)

	ml := NewMultiLimiter(redis, memory)
	ctx := context.Background()

	cfg := Config{Limit: 2, Window: time.Minute}

	// First request: Redis fails and returns "fail open" (Allowed: true)
	// This doesn't count against the memory limiter
	err := ml.Check(ctx, "test-key", cfg)
	assert.NoError(t, err)

	// Now IsAvailable() returns false, so all subsequent requests go to memory
	// Memory limiter counts: 1
	err = ml.Check(ctx, "test-key", cfg)
	assert.NoError(t, err)

	// Memory limiter counts: 2 (at limit)
	err = ml.Check(ctx, "test-key", cfg)
	assert.NoError(t, err)

	// Memory limiter counts: 3 (over limit)
	err = ml.Check(ctx, "test-key", cfg)
	assert.Equal(t, ErrRateLimited, err)
}

func TestMultiLimiter_CheckUser(t *testing.T) {
	client := NewMockRedisClient()
	config := DefaultRedisLimiterConfig()
	redis := NewRedisLimiter(client, config, nil)

	time.Sleep(50 * time.Millisecond)

	ml := NewMultiLimiter(redis, nil)
	ctx := context.Background()

	userID := uuid.New()
	cfg := Config{Limit: 1, Window: time.Minute}

	err := ml.CheckUser(ctx, userID, "action", cfg)
	assert.NoError(t, err)

	err = ml.CheckUser(ctx, userID, "action", cfg)
	assert.Equal(t, ErrRateLimited, err)
}

func TestMultiLimiter_CheckIP(t *testing.T) {
	client := NewMockRedisClient()
	config := DefaultRedisLimiterConfig()
	redis := NewRedisLimiter(client, config, nil)

	time.Sleep(50 * time.Millisecond)

	ml := NewMultiLimiter(redis, nil)
	ctx := context.Background()

	cfg := Config{Limit: 1, Window: time.Minute}

	err := ml.CheckIP(ctx, "10.0.0.1", "login", cfg)
	assert.NoError(t, err)

	err = ml.CheckIP(ctx, "10.0.0.1", "login", cfg)
	assert.Equal(t, ErrRateLimited, err)
}

func TestMultiLimiter_CheckChannel(t *testing.T) {
	client := NewMockRedisClient()
	config := DefaultRedisLimiterConfig()
	redis := NewRedisLimiter(client, config, nil)

	time.Sleep(50 * time.Millisecond)

	ml := NewMultiLimiter(redis, nil)
	ctx := context.Background()

	userID := uuid.New()
	channelID := uuid.New()
	cfg := Config{Limit: 1, Window: time.Minute}

	err := ml.CheckChannel(ctx, userID, channelID, "message", cfg)
	assert.NoError(t, err)

	err = ml.CheckChannel(ctx, userID, channelID, "message", cfg)
	assert.Equal(t, ErrRateLimited, err)
}

func TestMultiLimiter_IsRedisAvailable(t *testing.T) {
	client := NewMockRedisClient()
	config := DefaultRedisLimiterConfig()
	redis := NewRedisLimiter(client, config, nil)

	time.Sleep(50 * time.Millisecond)

	ml := NewMultiLimiter(redis, nil)
	assert.True(t, ml.IsRedisAvailable())

	// Disable Redis
	redis.setAvailable(false)
	assert.False(t, ml.IsRedisAvailable())
}

func TestMultiLimiter_NilRedis(t *testing.T) {
	memCache := NewMockCache()
	memory := NewLimiter(memCache)

	ml := NewMultiLimiter(nil, memory)
	ctx := context.Background()

	assert.False(t, ml.useRedis)

	cfg := Config{Limit: 2, Window: time.Minute}

	// Should use memory limiter
	err := ml.Check(ctx, "test-key", cfg)
	assert.NoError(t, err)

	err = ml.Check(ctx, "test-key", cfg)
	assert.NoError(t, err)

	err = ml.Check(ctx, "test-key", cfg)
	assert.Equal(t, ErrRateLimited, err)
}

func TestMultiLimiter_NoLimiter(t *testing.T) {
	ml := NewMultiLimiter(nil, nil)
	ctx := context.Background()

	cfg := Config{Limit: 1, Window: time.Minute}

	// Should fail open
	for i := 0; i < 10; i++ {
		err := ml.Check(ctx, "test-key", cfg)
		assert.NoError(t, err)
	}
}

// RateLimitStoreAdapter Tests

func TestRateLimitStoreAdapter_IncrementWithExpiry(t *testing.T) {
	// Skip this test as it requires a real Redis client
	// The adapter uses *redis.Client methods directly
	t.Skip("Requires real Redis client - covered by integration tests")
}

// Error handling tests

func TestRedisLimiter_ScriptReload_OnNOSCRIPT(t *testing.T) {
	client := NewMockRedisClient()
	config := DefaultRedisLimiterConfig()
	limiter := NewRedisLimiter(client, config, nil)

	time.Sleep(50 * time.Millisecond)

	ctx := context.Background()

	// First request works
	result, err := limiter.Check(ctx, "test-key", 10, time.Minute)
	require.NoError(t, err)
	assert.True(t, result.Allowed)

	// Clear scripts to simulate NOSCRIPT error
	client.mu.Lock()
	client.scripts = make(map[string]string)
	client.mu.Unlock()

	// Reset script state using safe method
	limiter.ResetScriptState()

	// Next request should trigger reload and succeed
	result, err = limiter.Check(ctx, "test-key", 10, time.Minute)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
}

func TestRedisLimiter_EvalError_FallsBack(t *testing.T) {
	client := NewMockRedisClient()
	config := DefaultRedisLimiterConfig()
	config.FallbackOnError = true

	memCache := NewMockCache()
	memLimiter := NewLimiter(memCache)

	limiter := NewRedisLimiter(client, config, memLimiter)

	time.Sleep(50 * time.Millisecond)

	ctx := context.Background()

	// First request works
	result, err := limiter.Check(ctx, "test-key", 10, time.Minute)
	require.NoError(t, err)
	assert.True(t, result.Allowed)

	// Simulate eval error
	client.SetFailNext()

	// Should fall back to memory limiter
	result, err = limiter.Check(ctx, "test-key", 10, time.Minute)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
}

// Benchmark tests

func BenchmarkRedisLimiter_Check_Allowed(b *testing.B) {
	client := NewMockRedisClient()
	config := DefaultRedisLimiterConfig()
	limiter := NewRedisLimiter(client, config, nil)

	time.Sleep(50 * time.Millisecond)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Check(ctx, fmt.Sprintf("bench-key-%d", i), 1000000, time.Minute)
	}
}

func BenchmarkRedisLimiter_Check_RateLimited(b *testing.B) {
	client := NewMockRedisClient()
	config := DefaultRedisLimiterConfig()
	limiter := NewRedisLimiter(client, config, nil)

	time.Sleep(50 * time.Millisecond)

	ctx := context.Background()

	// Use up the limit
	limiter.Check(ctx, "bench-key", 1, time.Minute)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Check(ctx, "bench-key", 1, time.Minute)
	}
}

func BenchmarkMultiLimiter_Check(b *testing.B) {
	client := NewMockRedisClient()
	config := DefaultRedisLimiterConfig()
	redis := NewRedisLimiter(client, config, nil)

	time.Sleep(50 * time.Millisecond)

	memCache := NewMockCache()
	memory := NewLimiter(memCache)

	ml := NewMultiLimiter(redis, memory)
	ctx := context.Background()

	cfg := Config{Limit: 1000000, Window: time.Minute}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ml.Check(ctx, fmt.Sprintf("bench-key-%d", i), cfg)
	}
}
