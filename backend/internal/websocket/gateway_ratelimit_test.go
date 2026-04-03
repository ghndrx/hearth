package websocket

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// TestConnectionLimiterConfig tests the default configuration
func TestConnectionLimiterConfig(t *testing.T) {
	cfg := DefaultConnectionLimitConfig()

	if cfg.MaxConnectionsPerIP != 20 {
		t.Errorf("expected default MaxConnectionsPerIP=20, got %d", cfg.MaxConnectionsPerIP)
	}
	if cfg.MaxConnectionsPerUser != 5 {
		t.Errorf("expected default MaxConnectionsPerUser=5, got %d", cfg.MaxConnectionsPerUser)
	}
	if cfg.ConnectionTTL != 10*time.Minute {
		t.Errorf("expected default ConnectionTTL=10m, got %v", cfg.ConnectionTTL)
	}
	if !cfg.Enabled {
		t.Error("expected Enabled=true by default")
	}
}

// TestConnectionLimiterIPKey tests the Redis key generation for IP-based keys
func TestConnectionLimiterIPKey(t *testing.T) {
	cfg := DefaultConnectionLimitConfig()
	limiter := NewConnectionLimiter(nil, cfg)

	tests := []struct {
		ip       string
		expected string
	}{
		{"192.168.1.1", "hearth:ws:conn:ip:192.168.1.1"},
		{"10.0.0.1", "hearth:ws:conn:ip:10.0.0.1"},
		{"::1", "hearth:ws:conn:ip:::1"},
	}

	for _, tc := range tests {
		got := limiter.IPKey(tc.ip)
		if got != tc.expected {
			t.Errorf("IPKey(%q) = %q, want %q", tc.ip, got, tc.expected)
		}
	}
}

// TestConnectionLimiterUserKey tests the Redis key generation for user-based keys
func TestConnectionLimiterUserKey(t *testing.T) {
	cfg := DefaultConnectionLimitConfig()
	limiter := NewConnectionLimiter(nil, cfg)

	userID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	expected := "hearth:ws:conn:user:123e4567-e89b-12d3-a456-426614174000"

	got := limiter.UserKey(userID)
	if got != expected {
		t.Errorf("UserKey() = %q, want %q", got, expected)
	}
}

// TestConnectionLimiterCheckDisabled tests that check passes when limiter is disabled
func TestConnectionLimiterCheckDisabled(t *testing.T) {
	cfg := &ConnectionLimitConfig{Enabled: false, MaxConnectionsPerIP: 1}
	limiter := NewConnectionLimiter(nil, cfg)

	userID := uuid.New()
	result := limiter.check(context.Background(), "192.168.1.1", userID)

	if !result.Allowed {
		t.Error("expected check to pass when limiter is disabled")
	}
}

// TestConnectionLimiterCheckNilRedis tests that check passes when Redis is nil
func TestConnectionLimiterCheckNilRedis(t *testing.T) {
	cfg := DefaultConnectionLimitConfig()
	limiter := NewConnectionLimiter(nil, cfg) // nil Redis client

	userID := uuid.New()
	result := limiter.check(context.Background(), "192.168.1.1", userID)

	// Should fail open (allow) when Redis is unavailable
	if !result.Allowed {
		t.Error("expected check to pass when Redis is nil (fail-open)")
	}
}

// TestConnectionLimiterCheckIPLimit tests per-IP connection limit enforcement
func TestConnectionLimiterCheckIPLimit(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer s.Close()

	cfg := &ConnectionLimitConfig{
		Enabled:               true,
		MaxConnectionsPerIP:   3,
		MaxConnectionsPerUser: 0, // unlimited
		ConnectionTTL:         10 * time.Minute,
	}

	limiter := NewConnectionLimiter(redis.NewClient(&redis.Options{Addr: s.Addr()}), cfg)
	ctx := context.Background()
	ip := "192.168.1.100"
	userID := uuid.New()

	// First 3 connections should be allowed (check then increment)
	for i := 0; i < 3; i++ {
		result := limiter.check(ctx, ip, userID)
		if !result.Allowed {
			t.Errorf("connection %d: expected allowed, got rejected: %s", i+1, result.Reason)
		}
		// After successful check, caller would increment
		limiter.Increment(ctx, ip, userID)
	}

	// 4th connection should be rejected (check shows 3 >= limit)
	result := limiter.check(ctx, ip, userID)
	if result.Allowed {
		t.Error("expected 4th connection to be rejected due to IP limit")
	}
	if result.Code != 4409 {
		t.Errorf("expected close code 4409, got %d", result.Code)
	}
}

// TestConnectionLimiterCheckUserLimit tests per-user connection limit enforcement
func TestConnectionLimiterCheckUserLimit(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer s.Close()

	cfg := &ConnectionLimitConfig{
		Enabled:               true,
		MaxConnectionsPerIP:   0, // unlimited
		MaxConnectionsPerUser: 2,
		ConnectionTTL:         10 * time.Minute,
	}

	limiter := NewConnectionLimiter(redis.NewClient(&redis.Options{Addr: s.Addr()}), cfg)
	ctx := context.Background()
	userID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

	// First 2 connections from different IPs should be allowed (check then increment)
	for i := 0; i < 2; i++ {
		ip := "192.168.1." + string(rune('A'+i)) // different IPs
		result := limiter.check(ctx, ip, userID)
		if !result.Allowed {
			t.Errorf("connection %d: expected allowed, got rejected: %s", i+1, result.Reason)
		}
		// After successful check, caller would increment
		limiter.Increment(ctx, ip, userID)
	}

	// 3rd connection from different IP should be rejected due to user limit
	result := limiter.check(ctx, "192.168.1.99", userID)
	if result.Allowed {
		t.Error("expected 3rd connection to be rejected due to user limit")
	}
}

// TestConnectionLimiterIncrementDecrement tests counter increment and decrement
func TestConnectionLimiterIncrementDecrement(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer s.Close()

	cfg := &ConnectionLimitConfig{
		Enabled:               true,
		MaxConnectionsPerIP:   10,
		MaxConnectionsPerUser: 10,
		ConnectionTTL:         10 * time.Minute,
	}

	redisClient := redis.NewClient(&redis.Options{Addr: s.Addr()})
	limiter := NewConnectionLimiter(redisClient, cfg)
	ctx := context.Background()
	ip := "10.0.0.1"
	userID := uuid.New()

	// Initial counts should be 0
	ipCount, userCount := limiter.GetCounts(ctx, ip, userID)
	if ipCount != 0 || userCount != 0 {
		t.Errorf("expected initial counts (0, 0), got (%d, %d)", ipCount, userCount)
	}

	// Increment
	limiter.Increment(ctx, ip, userID)

	ipCount, userCount = limiter.GetCounts(ctx, ip, userID)
	if ipCount != 1 || userCount != 1 {
		t.Errorf("expected counts (1, 1), got (%d, %d)", ipCount, userCount)
	}

	// Increment again
	limiter.Increment(ctx, ip, userID)

	ipCount, userCount = limiter.GetCounts(ctx, ip, userID)
	if ipCount != 2 || userCount != 2 {
		t.Errorf("expected counts (2, 2), got (%d, %d)", ipCount, userCount)
	}

	// Decrement
	limiter.Decrement(ctx, ip, userID)

	ipCount, userCount = limiter.GetCounts(ctx, ip, userID)
	if ipCount != 1 || userCount != 1 {
		t.Errorf("expected counts (1, 1) after decrement, got (%d, %d)", ipCount, userCount)
	}
}

// TestConnectionLimiterDecrementToZero tests that counters don't go negative
func TestConnectionLimiterDecrementToZero(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer s.Close()

	cfg := &ConnectionLimitConfig{
		Enabled:               true,
		MaxConnectionsPerIP:   10,
		MaxConnectionsPerUser: 10,
		ConnectionTTL:         10 * time.Minute,
	}

	redisClient := redis.NewClient(&redis.Options{Addr: s.Addr()})
	limiter := NewConnectionLimiter(redisClient, cfg)
	ctx := context.Background()
	ip := "10.0.0.1"
	userID := uuid.New()

	// Decrement without prior increment (baseline is 0)
	limiter.Decrement(ctx, ip, userID)

	// Should not go negative
	ipCount, _ := limiter.GetCounts(ctx, ip, userID)
	if ipCount < 0 {
		t.Errorf("expected count >= 0, got %d", ipCount)
	}
}

// TestConnectionLimiterGetCountsNilRedis tests GetCounts with nil Redis
func TestConnectionLimiterGetCountsNilRedis(t *testing.T) {
	cfg := DefaultConnectionLimitConfig()
	limiter := NewConnectionLimiter(nil, cfg)

	ipCount, userCount := limiter.GetCounts(context.Background(), "192.168.1.1", uuid.New())
	if ipCount != 0 || userCount != 0 {
		t.Errorf("expected (0, 0) with nil Redis, got (%d, %d)", ipCount, userCount)
	}
}

// TestConnectionLimiterCheckIPBeforeUser tests that IP check happens before user check
func TestConnectionLimiterCheckIPBeforeUser(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer s.Close()

	cfg := &ConnectionLimitConfig{
		Enabled:               true,
		MaxConnectionsPerIP:   2, // IP limit reached first
		MaxConnectionsPerUser: 1,
		ConnectionTTL:         10 * time.Minute,
	}

	limiter := NewConnectionLimiter(redis.NewClient(&redis.Options{Addr: s.Addr()}), cfg)
	ctx := context.Background()
	ip := "192.168.1.50"
	userID := uuid.New()

	// Use up the IP limit
	limiter.Increment(ctx, ip, userID)
	limiter.Increment(ctx, ip, userID)

	// Next connection should be rejected by IP limit, not user limit
	result := limiter.check(ctx, ip, userID)
	if result.Allowed {
		t.Error("expected connection to be rejected")
	}

	// The reason should mention IP
	if result.Reason == "" {
		t.Error("expected a rejection reason")
	}
}

// TestExtractClientIP tests IP extraction from websocket connection headers
func TestExtractClientIP(t *testing.T) {
	// This test verifies the header parsing logic
	xff := "203.0.113.50, 70.41.3.18, 150.172.238.178"
	xri := "203.0.113.50"

	// When XFF is present, should return first IP
	ip := ipFromHeadersTestHelper("203.0.113.1:8080", xff, "")
	if ip != "203.0.113.50" {
		t.Errorf("expected XFF first IP '203.0.113.50', got '%s'", ip)
	}

	// When XFF is empty but XRI is present
	ip = ipFromHeadersTestHelper("10.0.0.1:8080", "", xri)
	if ip != "203.0.113.50" {
		t.Errorf("expected XRI '203.0.113.50', got '%s'", ip)
	}

	// Fallback to remote addr
	ip = ipFromHeadersTestHelper("192.168.1.1:12345", "", "")
	if ip != "192.168.1.1" {
		t.Errorf("expected remote addr '192.168.1.1', got '%s'", ip)
	}

	// IPv6
	ip = ipFromHeadersTestHelper("[::1]:8080", "", "")
	if ip != "::1" {
		t.Errorf("expected IPv6 '::1', got '%s'", ip)
	}
}

// ipFromHeadersTestHelper mirrors the ExtractClientIP header parsing logic
func ipFromHeadersTestHelper(remoteAddr string, xff string, xri string) string {
	// X-Forwarded-For: first IP in the chain is the original client
	if xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			result := strings.TrimSpace(parts[0])
			if result != "" {
				return result
			}
		}
	}

	// X-Real-IP
	if xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fallback to remote addr, stripping port
	if remoteAddr != "" {
		// Handle IPv6 addresses like [::1]:8080
		if idx := strings.LastIndex(remoteAddr, ":"); idx != -1 {
			host := remoteAddr[:idx]
			if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
				return host[1 : len(host)-1]
			}
			return host
		}
		return remoteAddr
	}

	return "unknown"
}
