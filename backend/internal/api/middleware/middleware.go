package middleware

import (
	"context"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// RateLimitResult represents the result of a rate limit check
type RateLimitResult struct {
	Allowed    bool  // Whether the request is allowed
	Remaining  int   // Number of requests remaining in the window
	ResetAt    int64 // Unix timestamp (milliseconds) when the limit resets
	RetryAfter int64 // Milliseconds until the request can be retried (if not allowed)
	Limit      int   // Maximum requests allowed in the window
}

// RateLimiter defines the interface for rate limiting
type RateLimiter interface {
	Check(ctx context.Context, key string, limit int, window time.Duration) (*RateLimitResult, error)
	IsAvailable() bool
}

// RateLimitConfig configures rate limiting behavior
type RateLimitConfig struct {
	Limit          int
	Window         time.Duration
	SkipPaths      []string
	AuthMultiplier float64
	KeyGenerator   func(*fiber.Ctx) string
	OnLimitReached func(*fiber.Ctx) error
}

// DefaultRateLimitConfig returns sensible rate limit defaults
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Limit:          10000,
		Window:         60 * time.Second,
		AuthMultiplier: 2.0,
		SkipPaths:      []string{"/health", "/healthz", "/readyz", "/metrics"},
	}
}

// Middleware contains all middleware handlers
type Middleware struct {
	jwtSecret   []byte
	rateLimiter RateLimiter
}

// NewMiddleware creates middleware with dependencies
func NewMiddleware(jwtSecret string) *Middleware {
	return &Middleware{
		jwtSecret: []byte(jwtSecret),
	}
}

// NewMiddlewareWithRateLimiter creates middleware with a rate limiter
func NewMiddlewareWithRateLimiter(jwtSecret string, limiter RateLimiter) *Middleware {
	return &Middleware{
		jwtSecret:   []byte(jwtSecret),
		rateLimiter: limiter,
	}
}

// SetRateLimiter sets the rate limiter
func (m *Middleware) SetRateLimiter(limiter RateLimiter) {
	m.rateLimiter = limiter
}

// HasRateLimiter returns true if a rate limiter is configured
func (m *Middleware) HasRateLimiter() bool {
	return m.rateLimiter != nil
}

// IsRateLimiterAvailable returns true if rate limiter is configured and available
func (m *Middleware) IsRateLimiterAvailable() bool {
	return m.rateLimiter != nil && m.rateLimiter.IsAvailable()
}

// Claims represents JWT claims
type Claims struct {
	jwt.RegisteredClaims
	UserID   string `json:"uid"`
	Username string `json:"usr"`
	Type     string `json:"typ"`
	IsAdmin  bool   `json:"adm"`
}

// RequireAuth validates JWT and sets userID in context
func (m *Middleware) RequireAuth(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "missing authorization header",
		})
	}

	// Extract token
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "invalid authorization format",
		})
	}

	tokenString := parts[1]

	// Parse and validate JWT
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fiber.NewError(fiber.StatusUnauthorized, "invalid signing method")
		}
		return m.jwtSecret, nil
	})

	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "invalid token",
		})
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "invalid token claims",
		})
	}

	// Check token type
	if claims.Type != "access" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "invalid token type",
		})
	}

	// Check expiration
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "token expired",
		})
	}

	// Parse and set userID
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "invalid user id in token",
		})
	}

	c.Locals("userID", userID)
	c.Locals("username", claims.Username)
	c.Locals("isAdmin", claims.IsAdmin)
	return c.Next()
}

// WebSocketUpgrade checks if request is a WebSocket upgrade
func (m *Middleware) WebSocketUpgrade(c *fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(c) {
		return c.Next()
	}
	return fiber.ErrUpgradeRequired
}

// RateLimit applies rate limiting
func (m *Middleware) RateLimit(limit int, window int) fiber.Handler {
	return m.RateLimitWithConfig(RateLimitConfig{
		Limit:  limit,
		Window: time.Duration(window) * time.Second,
	})
}

// RateLimitWithConfig applies rate limiting with detailed configuration
func (m *Middleware) RateLimitWithConfig(config RateLimitConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip configured paths
		path := c.Path()
		for _, skip := range config.SkipPaths {
			if path == skip {
				return c.Next()
			}
		}

		// If no rate limiter, pass through
		if m.rateLimiter == nil {
			return c.Next()
		}

		// Generate rate limit key
		var key string
		if config.KeyGenerator != nil {
			key = config.KeyGenerator(c)
		} else if userID, ok := c.Locals("userID").(uuid.UUID); ok {
			key = "user:" + userID.String()
		} else {
			key = "ip:" + c.IP()
		}

		// Calculate effective limit (authenticated users may get higher limit)
		effectiveLimit := config.Limit
		if _, ok := c.Locals("userID").(uuid.UUID); ok && config.AuthMultiplier > 0 {
			effectiveLimit = int(float64(config.Limit) * config.AuthMultiplier)
		}

		// Check rate limit
		result, err := m.rateLimiter.Check(c.Context(), key, effectiveLimit, config.Window)
		if err != nil {
			// Fail open on errors
			return c.Next()
		}

		// Set rate limit headers
		c.Set("X-RateLimit-Limit", formatInt(result.Limit))
		c.Set("X-RateLimit-Remaining", formatInt(result.Remaining))
		c.Set("X-RateLimit-Reset", formatInt64(result.ResetAt))

		if !result.Allowed {
			c.Set("Retry-After", formatInt64(result.RetryAfter/1000))
			if config.OnLimitReached != nil {
				return config.OnLimitReached(c)
			}
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":       "rate_limit_exceeded",
				"retry_after": result.RetryAfter / 1000,
			})
		}

		return c.Next()
	}
}

func formatInt(n int) string {
	return strconv.Itoa(n)
}

func formatInt64(n int64) string {
	return strconv.FormatInt(n, 10)
}

// RequestID adds a unique request ID
func (m *Middleware) RequestID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		requestID := c.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("X-Request-ID", requestID)
		c.Locals("requestID", requestID)
		return c.Next()
	}
}

// Logger logs requests
func (m *Middleware) Logger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Process request
		err := c.Next()

		// Log request details
		duration := time.Since(start)
		requestID := c.Locals("requestID")
		userID := c.Locals("userID")

		logData := map[string]interface{}{
			"method":     c.Method(),
			"path":       c.Path(),
			"status":     c.Response().StatusCode(),
			"duration":   duration.Milliseconds(),
			"ip":         c.IP(),
			"user_agent": c.Get("User-Agent"),
		}

		if requestID != nil {
			logData["request_id"] = requestID
		}

		if userID != nil {
			logData["user_id"] = userID
		}

		// Use structured logging format
		log.Printf("[%s] %s %s - %d (%dms) IP:%s UA:%s",
			logData["request_id"],
			logData["method"],
			logData["path"],
			logData["status"],
			logData["duration"],
			logData["ip"],
			logData["user_agent"])

		return err
	}
}

// RequireAdmin checks if the authenticated user has admin privileges
func (m *Middleware) RequireAdmin(c *fiber.Ctx) error {
	isAdmin, ok := c.Locals("isAdmin").(bool)
	if !ok || !isAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "admin access required",
		})
	}
	return c.Next()
}

// Recover recovers from panics
func (m *Middleware) Recover() fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "internal server error",
				})
			}
		}()
		return c.Next()
	}
}
