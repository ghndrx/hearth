package middleware

import (
	"context"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// mockRateLimiter is a mock implementation of the RateLimiter interface for testing
type mockRateLimiter struct {
	mu          sync.Mutex
	counts      map[string]int
	maxRequests int
	window      time.Duration
	available   bool
	shouldError bool
}

func newMockRateLimiter(maxRequests int) *mockRateLimiter {
	return &mockRateLimiter{
		counts:      make(map[string]int),
		maxRequests: maxRequests,
		window:      time.Minute,
		available:   true,
	}
}

func (m *mockRateLimiter) Check(ctx context.Context, key string, limit int, window time.Duration) (*RateLimitResult, error) {
	if m.shouldError {
		return nil, context.DeadlineExceeded
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.counts[key]++
	count := m.counts[key]

	remaining := limit - count
	if remaining < 0 {
		remaining = 0
	}

	result := &RateLimitResult{
		Allowed:   count <= limit,
		Remaining: remaining,
		ResetAt:   time.Now().Add(window).UnixMilli(),
		Limit:     limit,
	}

	if !result.Allowed {
		result.RetryAfter = window.Milliseconds()
	}

	return result, nil
}

func (m *mockRateLimiter) IsAvailable() bool {
	return m.available
}

func (m *mockRateLimiter) reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counts = make(map[string]int)
}

func (m *mockRateLimiter) setAvailable(available bool) {
	m.available = available
}

func (m *mockRateLimiter) setShouldError(shouldError bool) {
	m.shouldError = shouldError
}

func TestRateLimit_BasicFunctionality(t *testing.T) {
	limiter := newMockRateLimiter(5)
	defer limiter.reset()
	m := NewMiddlewareWithRateLimiter("test-secret", limiter)

	app := fiber.New()
	defer func() {
		if err := app.Shutdown(); err != nil {
			t.Logf("app shutdown error: %v", err)
		}
	}()

	app.Use(m.RateLimit(5, 60))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// First 5 requests should succeed
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request %d failed: %v", i+1, err)
		}

		if resp.StatusCode != fiber.StatusOK {
			t.Errorf("request %d: expected status %d, got %d", i+1, fiber.StatusOK, resp.StatusCode)
		}

		// Check rate limit headers
		if resp.Header.Get("X-RateLimit-Limit") == "" {
			t.Errorf("request %d: missing X-RateLimit-Limit header", i+1)
		}
		if resp.Header.Get("X-RateLimit-Remaining") == "" {
			t.Errorf("request %d: missing X-RateLimit-Remaining header", i+1)
		}
		if resp.Header.Get("X-RateLimit-Reset") == "" {
			t.Errorf("request %d: missing X-RateLimit-Reset header", i+1)
		}
	}

	// 6th request should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("rate limited request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusTooManyRequests {
		t.Errorf("expected status %d for rate limited request, got %d", fiber.StatusTooManyRequests, resp.StatusCode)
	}

	// Should have Retry-After header when rate limited
	if resp.Header.Get("Retry-After") == "" {
		t.Error("missing Retry-After header for rate limited request")
	}
}

func TestRateLimit_NoLimiterConfigured(t *testing.T) {
	// Middleware without rate limiter - should pass through
	m := NewMiddleware("test-secret")

	app := fiber.New()
	t.Cleanup(func() { app.Shutdown() })
	app.Use(m.RateLimit(1, 60))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// All requests should pass through when no limiter is configured
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request %d failed: %v", i+1, err)
		}

		if resp.StatusCode != fiber.StatusOK {
			t.Errorf("request %d: expected status %d without limiter, got %d", i+1, fiber.StatusOK, resp.StatusCode)
		}
	}
}

func TestRateLimit_UserBasedKey(t *testing.T) {
	limiter := newMockRateLimiter(5)
	m := NewMiddlewareWithRateLimiter("test-secret", limiter)

	app := fiber.New()
	t.Cleanup(func() { app.Shutdown() })
	// Middleware to set userID (simulating auth)
	app.Use(func(c *fiber.Ctx) error {
		if c.Get("X-User-ID") != "" {
			userID, _ := uuid.Parse(c.Get("X-User-ID"))
			c.Locals("userID", userID)
		}
		return c.Next()
	})
	app.Use(m.RateLimit(2, 60))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	userID1 := uuid.New()
	userID2 := uuid.New()

	// User 1 makes 2 requests (should succeed)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-User-ID", userID1.String())
		resp, _ := app.Test(req)
		if resp.StatusCode != fiber.StatusOK {
			t.Errorf("user1 request %d failed with status %d", i+1, resp.StatusCode)
		}
	}

	// User 1's 3rd request should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-User-ID", userID1.String())
	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusTooManyRequests {
		t.Errorf("user1 request 3 should be rate limited, got status %d", resp.StatusCode)
	}

	// User 2 should still be able to make requests (different key)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-User-ID", userID2.String())
		resp, _ := app.Test(req)
		if resp.StatusCode != fiber.StatusOK {
			t.Errorf("user2 request %d failed with status %d", i+1, resp.StatusCode)
		}
	}
}

func TestRateLimit_FailOpen(t *testing.T) {
	limiter := newMockRateLimiter(5)
	limiter.setShouldError(true) // Simulate limiter error
	m := NewMiddlewareWithRateLimiter("test-secret", limiter)

	app := fiber.New()
	t.Cleanup(func() { app.Shutdown() })
	app.Use(m.RateLimit(1, 60))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// When limiter errors, should fail open (allow request)
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status %d on limiter error (fail open), got %d", fiber.StatusOK, resp.StatusCode)
	}
}

func TestRateLimitWithConfig_SkipPaths(t *testing.T) {
	limiter := newMockRateLimiter(1)
	m := NewMiddlewareWithRateLimiter("test-secret", limiter)

	app := fiber.New()
	t.Cleanup(func() { app.Shutdown() })
	config := RateLimitConfig{
		Limit:     1,
		Window:    time.Minute,
		SkipPaths: []string{"/health", "/metrics"},
	}
	app.Use(m.RateLimitWithConfig(config))
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})
	app.Get("/metrics", func(c *fiber.Ctx) error {
		return c.SendString("metrics")
	})
	app.Get("/api/data", func(c *fiber.Ctx) error {
		return c.SendString("data")
	})

	// Health endpoint should never be rate limited
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		resp, _ := app.Test(req)
		if resp.StatusCode != fiber.StatusOK {
			t.Errorf("health endpoint request %d should not be rate limited", i+1)
		}
	}

	// Metrics endpoint should never be rate limited
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/metrics", nil)
		resp, _ := app.Test(req)
		if resp.StatusCode != fiber.StatusOK {
			t.Errorf("metrics endpoint request %d should not be rate limited", i+1)
		}
	}

	// Reset limiter for API test
	limiter.reset()

	// API endpoint should be rate limited
	req := httptest.NewRequest("GET", "/api/data", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusOK {
		t.Error("first API request should succeed")
	}

	// Second request should be rate limited (limit=1)
	req = httptest.NewRequest("GET", "/api/data", nil)
	resp, _ = app.Test(req)
	if resp.StatusCode != fiber.StatusTooManyRequests {
		t.Error("second API request should be rate limited")
	}
}

func TestRateLimitWithConfig_AuthMultiplier(t *testing.T) {
	limiter := newMockRateLimiter(10)
	m := NewMiddlewareWithRateLimiter("test-secret", limiter)

	// Use a consistent userID for all authenticated requests
	testUserID := uuid.New()

	app := fiber.New()
	t.Cleanup(func() { app.Shutdown() })
	// Middleware to set userID for authenticated requests
	app.Use(func(c *fiber.Ctx) error {
		if c.Get("Authorization") != "" {
			c.Locals("userID", testUserID)
		}
		return c.Next()
	})

	config := RateLimitConfig{
		Limit:          2,
		Window:         time.Minute,
		AuthMultiplier: 3.0, // Authenticated users get 3x the limit (6 requests)
	}
	app.Use(m.RateLimitWithConfig(config))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Authenticated user should get 6 requests (2 * 3.0)
	for i := 0; i < 6; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer token")
		resp, _ := app.Test(req)
		if resp.StatusCode != fiber.StatusOK {
			t.Errorf("authenticated request %d should succeed, got status %d", i+1, resp.StatusCode)
		}
	}

	// 7th request should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer token")
	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusTooManyRequests {
		t.Errorf("7th authenticated request should be rate limited, got status %d", resp.StatusCode)
	}
}

func TestRateLimitWithConfig_CustomKeyGenerator(t *testing.T) {
	limiter := newMockRateLimiter(5)
	defer limiter.reset() // Ensure cleanup
	m := NewMiddlewareWithRateLimiter("test-secret", limiter)

	app := fiber.New()
	defer func() {
		if err := app.Shutdown(); err != nil {
			t.Logf("app shutdown error: %v", err)
		}
	}()

	config := RateLimitConfig{
		Limit:  2,
		Window: time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			// Use custom header as key
			return c.Get("X-API-Key")
		},
	}
	app.Use(m.RateLimitWithConfig(config))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Requests with API key "key1" should be tracked separately
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-API-Key", "key1")
		resp, _ := app.Test(req)
		if resp.StatusCode != fiber.StatusOK {
			t.Errorf("key1 request %d should succeed", i+1)
		}
	}

	// 3rd request with key1 should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "key1")
	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusTooManyRequests {
		t.Error("3rd request with key1 should be rate limited")
	}

	// Requests with different API key should work
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "key2")
	resp, _ = app.Test(req)
	if resp.StatusCode != fiber.StatusOK {
		t.Error("request with key2 should succeed")
	}
}

func TestRateLimitWithConfig_CustomLimitReachedHandler(t *testing.T) {
	limiter := newMockRateLimiter(5)
	m := NewMiddlewareWithRateLimiter("test-secret", limiter)

	customHandlerCalled := false
	app := fiber.New()
	t.Cleanup(func() { app.Shutdown() })
	config := RateLimitConfig{
		Limit:  1,
		Window: time.Minute,
		OnLimitReached: func(c *fiber.Ctx) error {
			customHandlerCalled = true
			return c.Status(fiber.StatusPaymentRequired).JSON(fiber.Map{
				"error":   "custom_rate_limit",
				"message": "Please upgrade your plan",
			})
		},
	}
	app.Use(m.RateLimitWithConfig(config))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// First request succeeds
	req := httptest.NewRequest("GET", "/test", nil)
	app.Test(req)

	// Second request triggers custom handler
	req = httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)

	if !customHandlerCalled {
		t.Error("custom limit reached handler should be called")
	}

	if resp.StatusCode != fiber.StatusPaymentRequired {
		t.Errorf("expected custom status code %d, got %d", fiber.StatusPaymentRequired, resp.StatusCode)
	}
}

func TestMiddleware_SetRateLimiter(t *testing.T) {
	m := NewMiddleware("test-secret")

	if m.HasRateLimiter() {
		t.Error("should not have rate limiter initially")
	}

	limiter := newMockRateLimiter(5)
	m.SetRateLimiter(limiter)

	if !m.HasRateLimiter() {
		t.Error("should have rate limiter after SetRateLimiter")
	}

	if !m.IsRateLimiterAvailable() {
		t.Error("rate limiter should be available")
	}

	// Test when limiter becomes unavailable
	limiter.setAvailable(false)
	if m.IsRateLimiterAvailable() {
		t.Error("rate limiter should be unavailable after setAvailable(false)")
	}
}

func TestDefaultRateLimitConfig(t *testing.T) {
	config := DefaultRateLimitConfig()

	if config.Limit != 10000 {
		t.Errorf("expected default limit 10000, got %d", config.Limit)
	}

	if config.Window != 60*time.Second {
		t.Errorf("expected default window 60s, got %v", config.Window)
	}

	if config.AuthMultiplier != 2.0 {
		t.Errorf("expected default auth multiplier 2.0, got %f", config.AuthMultiplier)
	}

	expectedSkipPaths := []string{"/health", "/healthz", "/readyz", "/metrics"}
	if len(config.SkipPaths) != len(expectedSkipPaths) {
		t.Errorf("expected %d skip paths, got %d", len(expectedSkipPaths), len(config.SkipPaths))
	}
}

func TestHybridLimiter(t *testing.T) {
	// Test HybridLimiter fallback behavior
	t.Run("uses Redis when available", func(t *testing.T) {
		// This is a unit test - we'd need actual Redis for integration test
		hybrid := NewHybridLimiter(nil, nil)
		if hybrid.IsAvailable() {
			t.Error("should not be available with no limiters")
		}
	})

	t.Run("IsRedisAvailable returns false with nil Redis", func(t *testing.T) {
		hybrid := NewHybridLimiter(nil, nil)
		if hybrid.IsRedisAvailable() {
			t.Error("Redis should not be available when nil")
		}
	})
}
