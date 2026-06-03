package matrixfederation

import (
	"sync"
	"time"
)

// FederationRateLimiter rate limits outgoing federation requests per remote server.
// This prevents flooding remote servers and helps manage resource usage.
type FederationRateLimiter struct {
	mu sync.RWMutex
	// requests per server in the last window
	requests map[string]*serverRate
	// Max requests per server per window
	maxRequests int
	// Time window
	window time.Duration
	// Cleanup interval
	cleanupInterval time.Duration
	stopCh chan struct{}
}

type serverRate struct {
	count      int
	windowStart time.Time
}

// Default limits
const (
	// DefaultMaxRequestsPerWindow is the default max requests per remote server per window
	DefaultMaxRequestsPerWindow = 100
	// DefaultWindow is the default time window
	DefaultWindow = 10 * time.Second
	// DefaultCleanupInterval is how often we clean up stale entries
	DefaultCleanupInterval = time.Minute
)

// NewFederationRateLimiter creates a new federation rate limiter.
func NewFederationRateLimiter(maxRequests int, window time.Duration) *FederationRateLimiter {
	if maxRequests <= 0 {
		maxRequests = DefaultMaxRequestsPerWindow
	}
	if window <= 0 {
		window = DefaultWindow
	}

	rl := &FederationRateLimiter{
		requests:        make(map[string]*serverRate),
		maxRequests:     maxRequests,
		window:         window,
		cleanupInterval: DefaultCleanupInterval,
		stopCh:         make(chan struct{}),
	}

	// Start cleanup goroutine
	go rl.cleanupLoop()

	return rl
}

// Allow checks if a request to the given server is allowed.
// Returns true if the request should proceed, false if rate limited.
func (rl *FederationRateLimiter) Allow(serverName string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Get or create server entry
	entry, exists := rl.requests[serverName]
	if !exists {
		rl.requests[serverName] = &serverRate{
			count:      1,
			windowStart: now,
		}
		return true
	}

	// Check if we're in a new window
	if now.Sub(entry.windowStart) >= rl.window {
		entry.count = 1
		entry.windowStart = now
		return true
	}

	// Check if we've exceeded the limit
	if entry.count >= rl.maxRequests {
		return false
	}

	entry.count++
	return true
}

// Reset clears the rate limit for a server (useful after a successful request)
func (rl *FederationRateLimiter) Reset(serverName string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.requests, serverName)
}

// GetRemaining returns the number of remaining requests allowed for this window
func (rl *FederationRateLimiter) GetRemaining(serverName string) int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	entry, exists := rl.requests[serverName]
	if !exists {
		return rl.maxRequests
	}

	now := time.Now()
	if now.Sub(entry.windowStart) >= rl.window {
		return rl.maxRequests
	}

	return rl.maxRequests - entry.count
}

// cleanupLoop periodically removes stale entries
func (rl *FederationRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-rl.stopCh:
			return
		case <-ticker.C:
			rl.cleanup()
		}
	}
}

func (rl *FederationRateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for server, entry := range rl.requests {
		if now.Sub(entry.windowStart) >= rl.window*2 {
			delete(rl.requests, server)
		}
	}
}

// Stop stops the rate limiter
func (rl *FederationRateLimiter) Stop() {
	close(rl.stopCh)
}
