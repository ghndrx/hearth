package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockRateLimitStore implements RateLimitStore for testing
type MockRateLimitStore struct {
	mu        sync.Mutex
	counters  map[string]int64
	expiries  map[string]time.Time
	data      map[string][]byte
	failNext  bool
	failCount int // Number of consecutive calls to fail
}

func NewMockRateLimitStore() *MockRateLimitStore {
	return &MockRateLimitStore{
		counters: make(map[string]int64),
		expiries: make(map[string]time.Time),
		data:     make(map[string][]byte),
	}
}

func (m *MockRateLimitStore) IncrementWithExpiry(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failNext || m.failCount > 0 {
		m.failNext = false
		if m.failCount > 0 {
			m.failCount--
		}
		return 0, errors.New("mock store error")
	}

	// Check if key expired
	if exp, ok := m.expiries[key]; ok && time.Now().After(exp) {
		delete(m.counters, key)
		delete(m.expiries, key)
		delete(m.data, key)
	}

	m.counters[key]++
	m.expiries[key] = time.Now().Add(ttl)
	m.data[key] = []byte(strconv.FormatInt(m.counters[key], 10))

	return m.counters[key], nil
}

func (m *MockRateLimitStore) Get(ctx context.Context, key string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failNext || m.failCount > 0 {
		m.failNext = false
		if m.failCount > 0 {
			m.failCount--
		}
		return nil, errors.New("mock store error")
	}

	// Check if key expired
	if exp, ok := m.expiries[key]; ok && time.Now().After(exp) {
		delete(m.counters, key)
		delete(m.expiries, key)
		delete(m.data, key)
		return nil, errors.New("key not found")
	}

	if data, ok := m.data[key]; ok {
		return data, nil
	}
	return nil, errors.New("key not found")
}

func (m *MockRateLimitStore) SetFailNext() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failNext = true
}

func (m *MockRateLimitStore) SetFailCount(count int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failCount = count
}

func (m *MockRateLimitStore) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters = make(map[string]int64)
	m.expiries = make(map[string]time.Time)
	m.data = make(map[string][]byte)
}

func (m *MockRateLimitStore) GetCount(key string) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.counters[key]
}

// Test helpers

func createTestApp(t testing.TB, middleware fiber.Handler) *fiber.App {
	app := fiber.New()
	t.Cleanup(func() {
		if err := app.Shutdown(); err != nil {
			t.Logf("app shutdown error: %v", err)
		}
	})
	app.Use(middleware)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "healthy"})
	})
	app.Get("/protected", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "protected"})
	})
	return app
}

func createAuthenticatedApp(t testing.TB, middleware fiber.Handler) *fiber.App {
	app := fiber.New()
	t.Cleanup(func() {
		if err := app.Shutdown(); err != nil {
			t.Logf("app shutdown error: %v", err)
		}
	})

	// Simulate auth middleware that sets userID
	app.Use(func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "Bearer valid-token" {
			c.Locals("userID", uuid.MustParse("00000000-0000-0000-0000-000000000001"))
		}
		return c.Next()
	})

	app.Use(middleware)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
	return app
}

// Tests

func TestNewRateLimiter(t *testing.T) {
	store := NewMockRateLimitStore()
	config := DefaultRateLimitConfig()

	rl := NewRateLimiter(store, config)

	assert.NotNil(t, rl)
	assert.Equal(t, store, rl.store)
	assert.True(t, rl.config.Enabled)
}

func TestRateLimiter_DisabledBypassesLimiting(t *testing.T) {
	store := NewMockRateLimitStore()
	config := DefaultRateLimitConfig()
	config.Enabled = false

	rl := NewRateLimiter(store, config)
	app := createTestApp(t, rl.Middleware())

	// Make many requests - all should succeed
	for i := 0; i < 200; i++ {
		resp, err := app.Test(httptest.NewRequest("GET", "/test", nil))
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	}
}

func TestRateLimiter_AllowsRequestsUnderLimit(t *testing.T) {
	store := NewMockRateLimitStore()
	config := DefaultRateLimitConfig()
	config.Max = 5
	config.Window = time.Minute

	rl := NewRateLimiter(store, config)
	app := createTestApp(t, rl.Middleware())

	// Make 5 requests - all should succeed
	for i := 0; i < 5; i++ {
		resp, err := app.Test(httptest.NewRequest("GET", "/test", nil))
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		// Check headers
		limit := resp.Header.Get("X-RateLimit-Limit")
		remaining := resp.Header.Get("X-RateLimit-Remaining")
		reset := resp.Header.Get("X-RateLimit-Reset")

		assert.Equal(t, "5", limit)
		assert.Equal(t, strconv.Itoa(4-i), remaining)
		assert.NotEmpty(t, reset)
	}
}

func TestRateLimiter_BlocksRequestsOverLimit(t *testing.T) {
	store := NewMockRateLimitStore()
	config := DefaultRateLimitConfig()
	config.Max = 3
	config.Window = time.Minute

	rl := NewRateLimiter(store, config)
	app := createTestApp(t, rl.Middleware())

	// Make 3 requests - all should succeed
	for i := 0; i < 3; i++ {
		resp, err := app.Test(httptest.NewRequest("GET", "/test", nil))
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	}

	// 4th request should be blocked
	resp, err := app.Test(httptest.NewRequest("GET", "/test", nil))
	require.NoError(t, err)
	assert.Equal(t, 429, resp.StatusCode)

	// Check Retry-After header
	retryAfter := resp.Header.Get("Retry-After")
	assert.NotEmpty(t, retryAfter)

	// Check response body
	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	assert.Equal(t, "rate_limited", result["error"])
}

func TestRateLimiter_BurstAllowance(t *testing.T) {
	store := NewMockRateLimitStore()
	config := DefaultRateLimitConfig()
	config.Max = 3
	config.Burst = 2
	config.Window = time.Minute

	rl := NewRateLimiter(store, config)
	app := createTestApp(t, rl.Middleware())

	// Should allow 5 requests (3 + 2 burst)
	for i := 0; i < 5; i++ {
		resp, err := app.Test(httptest.NewRequest("GET", "/test", nil))
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode, "request %d should succeed", i+1)
	}

	// 6th request should be blocked
	resp, err := app.Test(httptest.NewRequest("GET", "/test", nil))
	require.NoError(t, err)
	assert.Equal(t, 429, resp.StatusCode)
}

func TestRateLimiter_SkipsHealthEndpoints(t *testing.T) {
	store := NewMockRateLimitStore()
	config := DefaultRateLimitConfig()
	config.Max = 1
	config.SkipPaths = []string{"/health"}

	rl := NewRateLimiter(store, config)
	app := createTestApp(t, rl.Middleware())

	// Health endpoint should not be rate limited
	for i := 0; i < 100; i++ {
		resp, err := app.Test(httptest.NewRequest("GET", "/health", nil))
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	}

	// But /test should be limited
	resp, _ := app.Test(httptest.NewRequest("GET", "/test", nil))
	assert.Equal(t, 200, resp.StatusCode)

	resp, _ = app.Test(httptest.NewRequest("GET", "/test", nil))
	assert.Equal(t, 429, resp.StatusCode)
}

func TestRateLimiter_AuthenticatedUsersGetHigherLimit(t *testing.T) {
	store := NewMockRateLimitStore()
	config := DefaultRateLimitConfig()
	config.Max = 5
	config.AuthMultiplier = 2.0
	config.Window = time.Minute

	rl := NewRateLimiter(store, config)
	app := createAuthenticatedApp(t, rl.Middleware())

	// Authenticated user should get 10 requests (5 * 2.0)
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode, "request %d should succeed for auth user", i+1)
	}

	// 11th should fail
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 429, resp.StatusCode)
}

func TestRateLimiter_SeparateLimitsPerUser(t *testing.T) {
	store := NewMockRateLimitStore()
	config := DefaultRateLimitConfig()
	config.Max = 2
	config.AuthMultiplier = 1.0 // Disable auth multiplier for this test
	config.Window = time.Minute

	rl := NewRateLimiter(store, config)

	// Create app with different user simulation
	app := fiber.New()
	defer func() {
		if err := app.Shutdown(); err != nil {
			t.Logf("app shutdown error: %v", err)
		}
	}()
	app.Use(func(c *fiber.Ctx) error {
		userHeader := c.Get("X-User-ID")
		if userHeader != "" {
			c.Locals("userID", uuid.MustParse(userHeader))
		}
		return c.Next()
	})
	app.Use(rl.Middleware())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	user1 := "00000000-0000-0000-0000-000000000001"
	user2 := "00000000-0000-0000-0000-000000000002"

	// User 1 makes 2 requests (at their limit)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-User-ID", user1)
		resp, _ := app.Test(req)
		assert.Equal(t, 200, resp.StatusCode)
	}

	// User 2 should still be able to make requests
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-User-ID", user2)
	resp, _ := app.Test(req)
	assert.Equal(t, 200, resp.StatusCode)

	// User 1 should be blocked
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-User-ID", user1)
	resp, _ = app.Test(req)
	assert.Equal(t, 429, resp.StatusCode)
}

func TestRateLimiter_FailsOpenOnStoreError(t *testing.T) {
	store := NewMockRateLimitStore()
	config := DefaultRateLimitConfig()
	config.Max = 1

	rl := NewRateLimiter(store, config)
	app := createTestApp(t, rl.Middleware())

	// First request succeeds
	resp, _ := app.Test(httptest.NewRequest("GET", "/test", nil))
	assert.Equal(t, 200, resp.StatusCode)

	// Simulate store failure
	store.SetFailNext()

	// Should fail open (allow request)
	resp, _ = app.Test(httptest.NewRequest("GET", "/test", nil))
	assert.Equal(t, 200, resp.StatusCode)
}

func TestRateLimiter_CustomKeyGenerator(t *testing.T) {
	store := NewMockRateLimitStore()
	config := DefaultRateLimitConfig()
	config.Max = 1
	config.KeyGenerator = func(c *fiber.Ctx) string {
		return c.Get("X-API-Key")
	}

	rl := NewRateLimiter(store, config)
	app := createTestApp(t, rl.Middleware())

	// API Key 1 makes a request
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "key1")
	resp, _ := app.Test(req)
	assert.Equal(t, 200, resp.StatusCode)

	// API Key 1 is now rate limited
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "key1")
	resp, _ = app.Test(req)
	assert.Equal(t, 429, resp.StatusCode)

	// API Key 2 can still make requests
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "key2")
	resp, _ = app.Test(req)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestRateLimiter_CustomOnLimitReached(t *testing.T) {
	store := NewMockRateLimitStore()
	config := DefaultRateLimitConfig()
	config.Max = 1
	config.OnLimitReached = func(c *fiber.Ctx) error {
		return c.Status(503).JSON(fiber.Map{
			"error":   "service_busy",
			"message": "Please slow down",
		})
	}

	rl := NewRateLimiter(store, config)
	app := createTestApp(t, rl.Middleware())

	// First request succeeds
	resp, _ := app.Test(httptest.NewRequest("GET", "/test", nil))
	assert.Equal(t, 200, resp.StatusCode)

	// Second request gets custom error
	resp, _ = app.Test(httptest.NewRequest("GET", "/test", nil))
	assert.Equal(t, 503, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	assert.Equal(t, "service_busy", result["error"])
}

// Endpoint Rate Limiter Tests

func TestEndpointRateLimiter_Configure(t *testing.T) {
	store := NewMockRateLimitStore()
	defaults := EndpointConfig{Max: 100, Window: time.Minute}

	el := NewEndpointRateLimiter(store, defaults)
	el.Configure("auth", AuthEndpointConfig)

	assert.Equal(t, AuthEndpointConfig, el.configs["auth"])
}

func TestEndpointRateLimiter_UsesDefaults(t *testing.T) {
	store := NewMockRateLimitStore()
	defaults := EndpointConfig{Max: 3, Window: time.Minute}

	el := NewEndpointRateLimiter(store, defaults)

	app := fiber.New()
	defer func() {
		if err := app.Shutdown(); err != nil {
			t.Logf("app shutdown error: %v", err)
		}
	}()
	app.Get("/api/data", el.ForEndpoint("data"), func(c *fiber.Ctx) error {
		return c.SendStatus(200)
	})

	// Should allow 3 requests (default limit)
	for i := 0; i < 3; i++ {
		resp, _ := app.Test(httptest.NewRequest("GET", "/api/data", nil))
		assert.Equal(t, 200, resp.StatusCode)
	}

	// 4th should be blocked
	resp, _ := app.Test(httptest.NewRequest("GET", "/api/data", nil))
	assert.Equal(t, 429, resp.StatusCode)
}

func TestEndpointRateLimiter_UsesCustomConfig(t *testing.T) {
	store := NewMockRateLimitStore()
	defaults := EndpointConfig{Max: 100, Window: time.Minute}

	el := NewEndpointRateLimiter(store, defaults)
	el.Configure("auth", EndpointConfig{Max: 2, Window: time.Minute})

	app := fiber.New()
	defer func() {
		if err := app.Shutdown(); err != nil {
			t.Logf("app shutdown error: %v", err)
		}
	}()
	app.Post("/login", el.ForEndpoint("auth"), func(c *fiber.Ctx) error {
		return c.SendStatus(200)
	})

	// Should allow only 2 requests
	for i := 0; i < 2; i++ {
		resp, _ := app.Test(httptest.NewRequest("POST", "/login", nil))
		assert.Equal(t, 200, resp.StatusCode)
	}

	// 3rd should be blocked
	resp, _ := app.Test(httptest.NewRequest("POST", "/login", nil))
	assert.Equal(t, 429, resp.StatusCode)

	// Check endpoint name in response
	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	assert.Equal(t, "auth", result["endpoint"])
}

func TestEndpointRateLimiter_SeparateLimitsPerEndpoint(t *testing.T) {
	store := NewMockRateLimitStore()
	defaults := EndpointConfig{Max: 100, Window: time.Minute}

	el := NewEndpointRateLimiter(store, defaults)
	el.Configure("endpoint1", EndpointConfig{Max: 1, Window: time.Minute})
	el.Configure("endpoint2", EndpointConfig{Max: 1, Window: time.Minute})

	app := fiber.New()
	defer func() {
		if err := app.Shutdown(); err != nil {
			t.Logf("app shutdown error: %v", err)
		}
	}()
	app.Get("/api/e1", el.ForEndpoint("endpoint1"), func(c *fiber.Ctx) error {
		return c.SendStatus(200)
	})
	app.Get("/api/e2", el.ForEndpoint("endpoint2"), func(c *fiber.Ctx) error {
		return c.SendStatus(200)
	})

	// Endpoint 1 request
	resp, _ := app.Test(httptest.NewRequest("GET", "/api/e1", nil))
	assert.Equal(t, 200, resp.StatusCode)

	// Endpoint 1 is now limited
	resp, _ = app.Test(httptest.NewRequest("GET", "/api/e1", nil))
	assert.Equal(t, 429, resp.StatusCode)

	// Endpoint 2 should still work (separate limit)
	resp, _ = app.Test(httptest.NewRequest("GET", "/api/e2", nil))
	assert.Equal(t, 200, resp.StatusCode)
}

// Sliding Window Limiter Tests

func TestSlidingWindowLimiter_AllowsRequestsUnderLimit(t *testing.T) {
	store := NewMockRateLimitStore()
	config := DefaultRateLimitConfig()
	config.Max = 5
	config.Window = time.Minute

	sw := NewSlidingWindowLimiter(store, config)
	app := createTestApp(t, sw.Middleware())

	// Make 5 requests - all should succeed
	for i := 0; i < 5; i++ {
		resp, err := app.Test(httptest.NewRequest("GET", "/test", nil))
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	}
}

func TestSlidingWindowLimiter_BlocksOverLimit(t *testing.T) {
	store := NewMockRateLimitStore()
	config := DefaultRateLimitConfig()
	config.Max = 3
	config.Window = time.Minute

	sw := NewSlidingWindowLimiter(store, config)
	app := createTestApp(t, sw.Middleware())

	// Make 3 requests
	for i := 0; i < 3; i++ {
		resp, _ := app.Test(httptest.NewRequest("GET", "/test", nil))
		assert.Equal(t, 200, resp.StatusCode)
	}

	// 4th should be blocked
	resp, _ := app.Test(httptest.NewRequest("GET", "/test", nil))
	assert.Equal(t, 429, resp.StatusCode)
}

func TestSlidingWindowLimiter_SkipsHealthEndpoints(t *testing.T) {
	store := NewMockRateLimitStore()
	config := DefaultRateLimitConfig()
	config.Max = 1
	config.SkipPaths = []string{"/health"}

	sw := NewSlidingWindowLimiter(store, config)
	app := createTestApp(t, sw.Middleware())

	// Health endpoint should not be rate limited
	for i := 0; i < 50; i++ {
		resp, _ := app.Test(httptest.NewRequest("GET", "/health", nil))
		assert.Equal(t, 200, resp.StatusCode)
	}
}

func TestSlidingWindowLimiter_DisabledBypassesLimiting(t *testing.T) {
	store := NewMockRateLimitStore()
	config := DefaultRateLimitConfig()
	config.Enabled = false

	sw := NewSlidingWindowLimiter(store, config)
	app := createTestApp(t, sw.Middleware())

	// All requests should pass
	for i := 0; i < 200; i++ {
		resp, _ := app.Test(httptest.NewRequest("GET", "/test", nil))
		assert.Equal(t, 200, resp.StatusCode)
	}
}

// Helper function tests

func TestGetRateLimitInfo(t *testing.T) {
	store := NewMockRateLimitStore()
	config := RateLimitConfig{
		Max:    10,
		Burst:  5,
		Window: time.Minute,
	}

	ctx := context.Background()

	// Fresh key - should have full limit
	info, err := GetRateLimitInfo(ctx, store, "test-key", config)
	require.NoError(t, err)
	assert.Equal(t, 15, info.Limit)
	assert.Equal(t, 15, info.Remaining)
	assert.Equal(t, 0, info.RetryIn)
}

// Concurrent access tests

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	store := NewMockRateLimitStore()
	config := DefaultRateLimitConfig()
	config.Max = 100
	config.Window = time.Minute

	rl := NewRateLimiter(store, config)
	app := createTestApp(t, rl.Middleware())

	var wg sync.WaitGroup
	var allowed, denied int
	var mu sync.Mutex

	// Spawn 150 concurrent requests
	for i := 0; i < 150; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := app.Test(httptest.NewRequest("GET", "/test", nil))
			if err != nil {
				return
			}
			mu.Lock()
			if resp.StatusCode == 200 {
				allowed++
			} else if resp.StatusCode == 429 {
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

// Test predefined endpoint configs

func TestPredefinedEndpointConfigs(t *testing.T) {
	testCases := []struct {
		name   string
		config EndpointConfig
	}{
		{"AuthEndpointConfig", AuthEndpointConfig},
		{"UploadEndpointConfig", UploadEndpointConfig},
		{"MessageEndpointConfig", MessageEndpointConfig},
		{"SearchEndpointConfig", SearchEndpointConfig},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Greater(t, tc.config.Max, 0, "max should be positive")
			assert.Greater(t, tc.config.Window, time.Duration(0), "window should be positive")
		})
	}
}

// Test DefaultRateLimitConfig

func TestDefaultRateLimitConfig(t *testing.T) {
	config := DefaultRateLimitConfig()

	assert.True(t, config.Enabled)
	assert.Equal(t, 10000, config.Max)
	assert.Equal(t, 60*time.Second, config.Window)
	assert.Equal(t, 0, config.Burst)
	assert.Equal(t, 2.0, config.AuthMultiplier)
	assert.Contains(t, config.SkipPaths, "/health")
	assert.Contains(t, config.SkipPaths, "/healthz")
	assert.Contains(t, config.SkipPaths, "/readyz")
	assert.NotNil(t, config.KeyGenerator)
	assert.NotNil(t, config.OnLimitReached)
}

// Test rate limit headers

func TestRateLimitHeaders(t *testing.T) {
	store := NewMockRateLimitStore()
	config := DefaultRateLimitConfig()
	config.Max = 10
	config.Window = time.Minute

	rl := NewRateLimiter(store, config)
	app := createTestApp(t, rl.Middleware())

	resp, _ := app.Test(httptest.NewRequest("GET", "/test", nil))

	// Check all rate limit headers are present
	assert.Equal(t, "10", resp.Header.Get("X-RateLimit-Limit"))
	assert.Equal(t, "9", resp.Header.Get("X-RateLimit-Remaining"))

	resetStr := resp.Header.Get("X-RateLimit-Reset")
	assert.NotEmpty(t, resetStr)

	resetTime, err := strconv.ParseInt(resetStr, 10, 64)
	require.NoError(t, err)

	// Reset time should be ~60 seconds from now
	now := time.Now().Unix()
	assert.InDelta(t, now+60, resetTime, 5) // Within 5 seconds
}

func TestRateLimitHeaders_WhenLimited(t *testing.T) {
	store := NewMockRateLimitStore()
	config := DefaultRateLimitConfig()
	config.Max = 1
	config.Window = time.Minute

	rl := NewRateLimiter(store, config)
	app := createTestApp(t, rl.Middleware())

	// First request succeeds
	app.Test(httptest.NewRequest("GET", "/test", nil))

	// Second request should include Retry-After
	resp, _ := app.Test(httptest.NewRequest("GET", "/test", nil))
	assert.Equal(t, 429, resp.StatusCode)
	assert.NotEmpty(t, resp.Header.Get("Retry-After"))
	assert.Equal(t, "0", resp.Header.Get("X-RateLimit-Remaining"))
}

// Integration test: Full request flow

func TestRateLimiter_FullFlow(t *testing.T) {
	store := NewMockRateLimitStore()
	config := DefaultRateLimitConfig()
	config.Max = 5
	config.Burst = 2
	config.AuthMultiplier = 2.0
	config.Window = time.Minute

	rl := NewRateLimiter(store, config)

	app := fiber.New()
	defer func() {
		if err := app.Shutdown(); err != nil {
			t.Logf("app shutdown error: %v", err)
		}
	}()
	app.Use(func(c *fiber.Ctx) error {
		if c.Get("Authorization") == "Bearer token" {
			c.Locals("userID", uuid.MustParse("00000000-0000-0000-0000-000000000001"))
		}
		return c.Next()
	})
	app.Use(rl.Middleware())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendStatus(200)
	})
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendStatus(200)
	})

	// Health check should always work
	for i := 0; i < 100; i++ {
		resp, _ := app.Test(httptest.NewRequest("GET", "/health", nil))
		assert.Equal(t, 200, resp.StatusCode)
	}

	// Unauthenticated user gets 7 requests (5 + 2 burst)
	for i := 0; i < 7; i++ {
		resp, _ := app.Test(httptest.NewRequest("GET", "/test", nil))
		assert.Equal(t, 200, resp.StatusCode, "unauth request %d", i+1)
	}

	// 8th unauthenticated request should fail
	resp, _ := app.Test(httptest.NewRequest("GET", "/test", nil))
	assert.Equal(t, 429, resp.StatusCode)

	// Authenticated user gets 14 requests (7 * 2.0)
	store.Reset() // Reset to test auth separately
	for i := 0; i < 14; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer token")
		resp, _ := app.Test(req)
		assert.Equal(t, 200, resp.StatusCode, "auth request %d", i+1)
	}

	// 15th authenticated request should fail
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer token")
	resp, _ = app.Test(req)
	assert.Equal(t, 429, resp.StatusCode)
}

// Test empty key handling

func TestRateLimiter_EmptyKeyAllowsThrough(t *testing.T) {
	store := NewMockRateLimitStore()
	config := DefaultRateLimitConfig()
	config.Max = 1
	config.KeyGenerator = func(c *fiber.Ctx) string {
		return "" // Return empty key
	}

	rl := NewRateLimiter(store, config)
	app := createTestApp(t, rl.Middleware())

	// All requests should pass (can't identify client)
	for i := 0; i < 10; i++ {
		resp, _ := app.Test(httptest.NewRequest("GET", "/test", nil))
		assert.Equal(t, 200, resp.StatusCode)
	}
}

// Test multiple skip paths

func TestRateLimiter_MultipleSkipPaths(t *testing.T) {
	store := NewMockRateLimitStore()
	config := DefaultRateLimitConfig()
	config.Max = 1
	config.SkipPaths = []string{"/health", "/api/internal", "/metrics"}

	rl := NewRateLimiter(store, config)

	app := fiber.New()
	defer func() {
		if err := app.Shutdown(); err != nil {
			t.Logf("app shutdown error: %v", err)
		}
	}()
	app.Use(rl.Middleware())
	app.Get("/health", func(c *fiber.Ctx) error { return c.SendStatus(200) })
	app.Get("/healthz", func(c *fiber.Ctx) error { return c.SendStatus(200) })
	app.Get("/api/internal/status", func(c *fiber.Ctx) error { return c.SendStatus(200) })
	app.Get("/metrics", func(c *fiber.Ctx) error { return c.SendStatus(200) })
	app.Get("/api/data", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	// All skip paths should work
	for i := 0; i < 5; i++ {
		resp, _ := app.Test(httptest.NewRequest("GET", "/health", nil))
		assert.Equal(t, 200, resp.StatusCode)

		resp, _ = app.Test(httptest.NewRequest("GET", "/api/internal/status", nil))
		assert.Equal(t, 200, resp.StatusCode)

		resp, _ = app.Test(httptest.NewRequest("GET", "/metrics", nil))
		assert.Equal(t, 200, resp.StatusCode)
	}

	// Regular API should be limited
	resp, _ := app.Test(httptest.NewRequest("GET", "/api/data", nil))
	assert.Equal(t, 200, resp.StatusCode)

	resp, _ = app.Test(httptest.NewRequest("GET", "/api/data", nil))
	assert.Equal(t, 429, resp.StatusCode)
}

// Benchmark tests

func BenchmarkRateLimiter_AllowedRequest(b *testing.B) {
	store := NewMockRateLimitStore()
	config := DefaultRateLimitConfig()
	config.Max = 1000000 // High limit

	rl := NewRateLimiter(store, config)
	app := createTestApp(b, rl.Middleware())

	req := httptest.NewRequest("GET", "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app.Test(req)
	}
}

func BenchmarkRateLimiter_RateLimited(b *testing.B) {
	store := NewMockRateLimitStore()
	config := DefaultRateLimitConfig()
	config.Max = 0 // Immediately limited

	rl := NewRateLimiter(store, config)
	app := createTestApp(b, rl.Middleware())

	req := httptest.NewRequest("GET", "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app.Test(req)
	}
}

// Test custom key generator for API key-based rate limiting

func TestRateLimiter_APIKeyBasedLimiting(t *testing.T) {
	store := NewMockRateLimitStore()
	config := DefaultRateLimitConfig()
	config.Max = 1
	// Use API key from header as the rate limit key
	config.KeyGenerator = func(c *fiber.Ctx) string {
		return c.Get("X-API-Key")
	}

	rl := NewRateLimiter(store, config)

	app := fiber.New()
	defer func() {
		if err := app.Shutdown(); err != nil {
			t.Logf("app shutdown error: %v", err)
		}
	}()
	app.Use(rl.Middleware())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendStatus(200)
	})

	// Request with API key 1
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "api-key-1")
	resp, _ := app.Test(req)
	assert.Equal(t, 200, resp.StatusCode)

	// Same API key should be limited
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "api-key-1")
	resp, _ = app.Test(req)
	assert.Equal(t, 429, resp.StatusCode)

	// Different API key should work
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "api-key-2")
	resp, _ = app.Test(req)
	assert.Equal(t, 200, resp.StatusCode)
}

// Test large window duration

func TestRateLimiter_LargeWindow(t *testing.T) {
	store := NewMockRateLimitStore()
	config := DefaultRateLimitConfig()
	config.Max = 2
	config.Window = 24 * time.Hour

	rl := NewRateLimiter(store, config)
	app := createTestApp(t, rl.Middleware())

	for i := 0; i < 2; i++ {
		resp, _ := app.Test(httptest.NewRequest("GET", "/test", nil))
		assert.Equal(t, 200, resp.StatusCode)
	}

	resp, _ := app.Test(httptest.NewRequest("GET", "/test", nil))
	assert.Equal(t, 429, resp.StatusCode)

	// Check reset header reflects the large window
	resetStr := resp.Header.Get("X-RateLimit-Reset")
	resetTime, _ := strconv.ParseInt(resetStr, 10, 64)
	now := time.Now().Unix()
	// Should be approximately 24 hours from now
	assert.InDelta(t, now+86400, resetTime, 60) // Within 1 minute
}

// Test that paths are matched with prefix

func TestRateLimiter_PathPrefixMatching(t *testing.T) {
	store := NewMockRateLimitStore()
	config := DefaultRateLimitConfig()
	config.Max = 1
	config.SkipPaths = []string{"/api/internal"}

	rl := NewRateLimiter(store, config)

	app := fiber.New()
	defer func() {
		if err := app.Shutdown(); err != nil {
			t.Logf("app shutdown error: %v", err)
		}
	}()
	app.Use(rl.Middleware())
	app.Get("/api/internal", func(c *fiber.Ctx) error { return c.SendStatus(200) })
	app.Get("/api/internal/deep/path", func(c *fiber.Ctx) error { return c.SendStatus(200) })
	app.Get("/api/public", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	// /api/internal and subpaths should be skipped
	for i := 0; i < 5; i++ {
		resp, _ := app.Test(httptest.NewRequest("GET", "/api/internal", nil))
		assert.Equal(t, 200, resp.StatusCode)

		resp, _ = app.Test(httptest.NewRequest("GET", "/api/internal/deep/path", nil))
		assert.Equal(t, 200, resp.StatusCode)
	}

	// /api/public should be rate limited
	resp, _ := app.Test(httptest.NewRequest("GET", "/api/public", nil))
	assert.Equal(t, 200, resp.StatusCode)

	resp, _ = app.Test(httptest.NewRequest("GET", "/api/public", nil))
	assert.Equal(t, 429, resp.StatusCode)
}

// Test endpoint limiter with authenticated user

func TestEndpointRateLimiter_AuthenticatedUser(t *testing.T) {
	store := NewMockRateLimitStore()
	defaults := EndpointConfig{Max: 1, Window: time.Minute}

	el := NewEndpointRateLimiter(store, defaults)

	app := fiber.New()
	defer func() {
		if err := app.Shutdown(); err != nil {
			t.Logf("app shutdown error: %v", err)
		}
	}()
	app.Use(func(c *fiber.Ctx) error {
		if c.Get("Authorization") == "Bearer token" {
			c.Locals("userID", uuid.MustParse("00000000-0000-0000-0000-000000000001"))
		}
		return c.Next()
	})
	app.Get("/test", el.ForEndpoint("test"), func(c *fiber.Ctx) error {
		return c.SendStatus(200)
	})

	// Auth user makes request
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer token")
	resp, _ := app.Test(req)
	assert.Equal(t, 200, resp.StatusCode)

	// Auth user is limited
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer token")
	resp, _ = app.Test(req)
	assert.Equal(t, 429, resp.StatusCode)

	// Unauth user can still make request (different key)
	resp, _ = app.Test(httptest.NewRequest("GET", "/test", nil))
	assert.Equal(t, 200, resp.StatusCode)
}

// Test NewRateLimiter with nil functions

func TestNewRateLimiter_NilFunctions(t *testing.T) {
	store := NewMockRateLimitStore()
	config := RateLimitConfig{
		Enabled:        true,
		Max:            1,
		Window:         time.Minute,
		KeyGenerator:   nil, // Should use default
		OnLimitReached: nil, // Should use default
	}

	rl := NewRateLimiter(store, config)
	app := createTestApp(t, rl.Middleware())

	// Should work with defaults
	resp, _ := app.Test(httptest.NewRequest("GET", "/test", nil))
	assert.Equal(t, 200, resp.StatusCode)

	resp, _ = app.Test(httptest.NewRequest("GET", "/test", nil))
	assert.Equal(t, 429, resp.StatusCode)
}

// Test middleware returns immediately if store IncrementWithExpiry fails for sliding window

func TestSlidingWindowLimiter_FailsOpenOnStoreError(t *testing.T) {
	store := NewMockRateLimitStore()
	config := DefaultRateLimitConfig()
	config.Max = 100 // High limit so we don't hit it

	sw := NewSlidingWindowLimiter(store, config)
	app := createTestApp(t, sw.Middleware())

	// First request to establish baseline
	resp, _ := app.Test(httptest.NewRequest("GET", "/test", nil))
	assert.Equal(t, 200, resp.StatusCode)

	// Fail 2 consecutive calls (Get for previous window + IncrementWithExpiry)
	store.SetFailCount(2)

	// Should fail open - the store errors should not block the request
	resp, _ = app.Test(httptest.NewRequest("GET", "/test", nil))
	assert.Equal(t, 200, resp.StatusCode)
}

// Test auth multiplier of 0 defaults to 1.0

func TestRateLimiter_ZeroAuthMultiplier(t *testing.T) {
	store := NewMockRateLimitStore()
	config := RateLimitConfig{
		Enabled:        true,
		Max:            5,
		Window:         time.Minute,
		AuthMultiplier: 0, // Should default to 1.0
	}

	rl := NewRateLimiter(store, config)

	// Verify the multiplier was set to 1.0
	assert.Equal(t, 1.0, rl.config.AuthMultiplier)
}

// Test context is properly passed

func TestRateLimiter_UsesContext(t *testing.T) {
	store := NewMockRateLimitStore()
	config := DefaultRateLimitConfig()
	config.Max = 10

	rl := NewRateLimiter(store, config)
	app := createTestApp(t, rl.Middleware())

	// Make a request - context should be used internally
	resp, err := app.Test(httptest.NewRequest("GET", "/test", nil))
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

// Verify key prefixes are correct

func TestRateLimiter_KeyPrefixes(t *testing.T) {
	store := NewMockRateLimitStore()
	config := DefaultRateLimitConfig()
	config.Max = 1

	rl := NewRateLimiter(store, config)

	app := fiber.New()
	defer func() {
		if err := app.Shutdown(); err != nil {
			t.Logf("app shutdown error: %v", err)
		}
	}()
	app.Use(func(c *fiber.Ctx) error {
		if c.Get("X-User-ID") != "" {
			c.Locals("userID", uuid.MustParse(c.Get("X-User-ID")))
		}
		return c.Next()
	})
	app.Use(rl.Middleware())
	app.Get("/test", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	// Unauthenticated request
	app.Test(httptest.NewRequest("GET", "/test", nil))

	// Check that an IP-prefixed key was created
	found := false
	for key := range store.counters {
		if strings.HasPrefix(key, "ratelimit:global:ip:") {
			found = true
			break
		}
	}
	assert.True(t, found, "IP-prefixed key should exist")

	// Authenticated request
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-User-ID", "00000000-0000-0000-0000-000000000001")
	app.Test(req)

	// Check that a user-prefixed key was created
	found = false
	for key := range store.counters {
		if strings.HasPrefix(key, "ratelimit:global:user:") {
			found = true
			break
		}
	}
	assert.True(t, found, "User-prefixed key should exist")
}

// Test JSON error response format

func TestRateLimitErrorResponseFormat(t *testing.T) {
	store := NewMockRateLimitStore()
	config := DefaultRateLimitConfig()
	config.Max = 0 // Immediately limited

	rl := NewRateLimiter(store, config)
	app := createTestApp(t, rl.Middleware())

	resp, _ := app.Test(httptest.NewRequest("GET", "/test", nil))
	assert.Equal(t, 429, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(t, err)

	assert.Contains(t, result, "error")
	assert.Contains(t, result, "message")
	assert.Equal(t, "rate_limited", result["error"])
	assert.NotEmpty(t, result["message"])
}

// Ensure proper handling when AuthMultiplier makes limit fractional

func TestRateLimiter_FractionalLimit(t *testing.T) {
	store := NewMockRateLimitStore()
	config := DefaultRateLimitConfig()
	config.Max = 3
	config.AuthMultiplier = 1.5 // Should give 4.5 -> 4 (int conversion)

	rl := NewRateLimiter(store, config)

	app := fiber.New()
	defer func() {
		if err := app.Shutdown(); err != nil {
			t.Logf("app shutdown error: %v", err)
		}
	}()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", uuid.MustParse("00000000-0000-0000-0000-000000000001"))
		return c.Next()
	})
	app.Use(rl.Middleware())
	app.Get("/test", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	// Should allow 4 requests (3 * 1.5 = 4.5, truncated to 4)
	for i := 0; i < 4; i++ {
		resp, _ := app.Test(httptest.NewRequest("GET", "/test", nil))
		assert.Equal(t, 200, resp.StatusCode, fmt.Sprintf("request %d should succeed", i+1))
	}

	// 5th should be blocked
	resp, _ := app.Test(httptest.NewRequest("GET", "/test", nil))
	assert.Equal(t, 429, resp.StatusCode)
}
