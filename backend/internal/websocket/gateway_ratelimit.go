package websocket

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// ConnectionLimitConfig holds per-IP and per-user WebSocket connection limits
type ConnectionLimitConfig struct {
	// MaxConnectionsPerIP limits concurrent WebSocket connections from a single IP address.
	// A value of 0 means unlimited.
	MaxConnectionsPerIP int

	// MaxConnectionsPerUser limits concurrent WebSocket connections for a single user.
	// A value of 0 means unlimited.
	MaxConnectionsPerUser int

	// ConnectionTTL is how long a connection entry remains in Redis before expiring.
	// Should be longer than the WebSocket session timeout.
	ConnectionTTL time.Duration

	// Enabled controls whether connection limiting is active.
	Enabled bool
}

// DefaultConnectionLimitConfig returns sensible defaults
func DefaultConnectionLimitConfig() *ConnectionLimitConfig {
	return &ConnectionLimitConfig{
		MaxConnectionsPerIP:   20,  // 20 connections per IP
		MaxConnectionsPerUser: 5,   // 5 connections per user (web + mobile + desktop etc.)
		ConnectionTTL:          10 * time.Minute,
		Enabled:               true,
	}
}

// ConnectionLimiter enforces per-IP and per-user WebSocket connection limits
// using Redis for distributed coordination across multiple gateway instances.
type ConnectionLimiter struct {
	redis    *redis.Client
	config   *ConnectionLimitConfig
	keyPrefix string
}

// NewConnectionLimiter creates a ConnectionLimiter with the given Redis client and config.
// If redis is nil, the limiter will operate in no-op (allow-all) mode.
func NewConnectionLimiter(redisClient *redis.Client, config *ConnectionLimitConfig) *ConnectionLimiter {
	if config == nil {
		config = DefaultConnectionLimitConfig()
	}
	return &ConnectionLimiter{
		redis:     redisClient,
		config:    config,
		keyPrefix: "hearth:ws:conn:",
	}
}

// IPKey returns the Redis key for tracking connections from an IP address
func (l *ConnectionLimiter) IPKey(ip string) string {
	return l.keyPrefix + "ip:" + ip
}

// UserKey returns the Redis key for tracking connections for a user
func (l *ConnectionLimiter) UserKey(userID uuid.UUID) string {
	return l.keyPrefix + "user:" + userID.String()
}

// CheckResult describes the result of a connection limit check
type CheckResult struct {
	Allowed         bool
	Reason          string
	Code            int    // WebSocket close code to send
	CurrentIPCount  int64  // Current connection count for this IP
	CurrentUserCount int64 // Current connection count for this user
}

// check performs the limit check for IP and/or user.
// Returns a CheckResult indicating whether the connection should be allowed.
func (l *ConnectionLimiter) check(ctx context.Context, ip string, userID uuid.UUID) CheckResult {
	// Fast path: if limiting is disabled, allow everything
	if !l.config.Enabled {
		return CheckResult{Allowed: true}
	}

	// Fast path: if Redis is unavailable, fail open (allow) or fail closed based on config
	if l.redis == nil {
		log.Printf("[ConnectionLimiter] Redis unavailable, allowing connection (fail-open)")
		return CheckResult{Allowed: true}
	}

	result := CheckResult{}

	// Check per-IP limit
	if l.config.MaxConnectionsPerIP > 0 {
		ipCount, err := l.redis.Get(ctx, l.IPKey(ip)).Int64()
		if err != nil && err != redis.Nil {
			log.Printf("[ConnectionLimiter] Redis error checking IP limit: %v", err)
			// Fail open on Redis errors to avoid blocking legitimate connections
			return CheckResult{Allowed: true}
		}
		result.CurrentIPCount = ipCount
		if ipCount >= int64(l.config.MaxConnectionsPerIP) {
			result.Allowed = false
			result.Reason = fmt.Sprintf("too many connections from IP (limit: %d)", l.config.MaxConnectionsPerIP)
			result.Code = 4409 // "Too Many Requests" - close code 4409 is "Rate Limited"
			return result
		}
	}

	// Check per-user limit
	if l.config.MaxConnectionsPerUser > 0 {
		userCount, err := l.redis.Get(ctx, l.UserKey(userID)).Int64()
		if err != nil && err != redis.Nil {
			log.Printf("[ConnectionLimiter] Redis error checking user limit: %v", err)
			return CheckResult{Allowed: true}
		}
		result.CurrentUserCount = userCount
		if userCount >= int64(l.config.MaxConnectionsPerUser) {
			result.Allowed = false
			result.Reason = fmt.Sprintf("too many connections for user (limit: %d)", l.config.MaxConnectionsPerUser)
			result.Code = 4409
			return result
		}
	}

	return CheckResult{Allowed: true}
}

// Increment increments the connection counters for the given IP and user.
// Must be called after a successful Check() and after the connection is established.
func (l *ConnectionLimiter) Increment(ctx context.Context, ip string, userID uuid.UUID) {
	if !l.config.Enabled || l.redis == nil {
		return
	}

	ttl := l.config.ConnectionTTL
	pipe := l.redis.Pipeline()

	if l.config.MaxConnectionsPerIP > 0 {
		pipe.Incr(ctx, l.IPKey(ip))
		pipe.Expire(ctx, l.IPKey(ip), ttl)
	}
	if l.config.MaxConnectionsPerUser > 0 {
		pipe.Incr(ctx, l.UserKey(userID))
		pipe.Expire(ctx, l.UserKey(userID), ttl)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		log.Printf("[ConnectionLimiter] Redis error incrementing counters: %v", err)
	}
}

// Decrement decrements the connection counters for the given IP and user.
// Must be called when a connection closes.
func (l *ConnectionLimiter) Decrement(ctx context.Context, ip string, userID uuid.UUID) {
	if !l.config.Enabled || l.redis == nil {
		return
	}

	// Use DECR and ensure we don't go below 0
	// Note: We use Lua script to atomically decrement and delete if <= 0
	script := redis.NewScript(`
		local current = redis.call('DECR', KEYS[1])
		if current <= 0 then
			redis.call('DEL', KEYS[1])
		end
		return current
	`)

	if l.config.MaxConnectionsPerIP > 0 {
		if _, err := script.Run(ctx, l.redis, []string{l.IPKey(ip)}).Result(); err != nil && err != redis.Nil {
			log.Printf("[ConnectionLimiter] Redis error decrementing IP counter: %v", err)
		}
	}
	if l.config.MaxConnectionsPerUser > 0 {
		if _, err := script.Run(ctx, l.redis, []string{l.UserKey(userID)}).Result(); err != nil && err != redis.Nil {
			log.Printf("[ConnectionLimiter] Redis error decrementing user counter: %v", err)
		}
	}
}

// GetCounts returns the current connection counts for a given IP and user
func (l *ConnectionLimiter) GetCounts(ctx context.Context, ip string, userID uuid.UUID) (ipCount, userCount int64) {
	if l.redis == nil {
		return 0, 0
	}

	if l.config.MaxConnectionsPerIP > 0 {
		if count, err := l.redis.Get(ctx, l.IPKey(ip)).Int64(); err == nil {
			ipCount = count
		}
	}
	if l.config.MaxConnectionsPerUser > 0 {
		if count, err := l.redis.Get(ctx, l.UserKey(userID)).Int64(); err == nil {
			userCount = count
		}
	}
	return
}

// ExtractClientIP extracts the real client IP from a Fiber websocket.Conn.
// It checks X-Forwarded-For and X-Real-IP headers first (for proxied requests),
// then falls back to the remote address.
func ExtractClientIP(conn *websocket.Conn) string {
	// X-Forwarded-For: first IP in the chain is the original client
	xff := conn.Headers("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if ip != "" {
				return ip
			}
		}
	}

	// X-Real-IP
	xri := conn.Headers("X-Real-IP")
	if xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fallback to remote address, stripping port
	remoteAddr := conn.RemoteAddr().String()
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
