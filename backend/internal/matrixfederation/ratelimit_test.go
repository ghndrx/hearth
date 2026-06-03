package matrixfederation

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFederationRateLimiter_Allow(t *testing.T) {
	rl := NewFederationRateLimiter(3, time.Second)

	// First 3 requests should be allowed
	require.True(t, rl.Allow("server1.example.com"), "first request should be allowed")
	require.True(t, rl.Allow("server1.example.com"), "second request should be allowed")
	require.True(t, rl.Allow("server1.example.com"), "third request should be allowed")

	// 4th request should be rate limited
	require.False(t, rl.Allow("server1.example.com"), "fourth request should be rate limited")
}

func TestFederationRateLimiter_PerServer(t *testing.T) {
	rl := NewFederationRateLimiter(2, time.Second)

	// server1 hits limit
	require.True(t, rl.Allow("server1.example.com"))
	require.True(t, rl.Allow("server1.example.com"))
	require.False(t, rl.Allow("server1.example.com"))

	// server2 should be independent
	require.True(t, rl.Allow("server2.example.com"))
	require.True(t, rl.Allow("server2.example.com"))
	require.False(t, rl.Allow("server2.example.com"))
}

func TestFederationRateLimiter_WindowReset(t *testing.T) {
	rl := NewFederationRateLimiter(2, 50*time.Millisecond)

	require.True(t, rl.Allow("server1.example.com"))
	require.True(t, rl.Allow("server1.example.com"))
	require.False(t, rl.Allow("server1.example.com"))

	// Wait for window to pass
	time.Sleep(60 * time.Millisecond)

	// Should be allowed again
	require.True(t, rl.Allow("server1.example.com"))
}

func TestFederationRateLimiter_GetRemaining(t *testing.T) {
	rl := NewFederationRateLimiter(5, time.Second)

	// Initially 5 remaining
	require.Equal(t, 5, rl.GetRemaining("server1.example.com"))

	rl.Allow("server1.example.com")
	require.Equal(t, 4, rl.GetRemaining("server1.example.com"))

	rl.Allow("server1.example.com")
	require.Equal(t, 3, rl.GetRemaining("server1.example.com"))

	// Different server has full count
	require.Equal(t, 5, rl.GetRemaining("server2.example.com"))
}

func TestFederationRateLimiter_Reset(t *testing.T) {
	rl := NewFederationRateLimiter(2, time.Second)

	require.True(t, rl.Allow("server1.example.com"))
	require.True(t, rl.Allow("server1.example.com"))
	require.False(t, rl.Allow("server1.example.com"))

	// Reset and should be allowed again
	rl.Reset("server1.example.com")
	require.True(t, rl.Allow("server1.example.com"))
}

func TestFederationRateLimiter_Stop(t *testing.T) {
	rl := NewFederationRateLimiter(10, time.Second)
	rl.Stop() // Should not panic
}
