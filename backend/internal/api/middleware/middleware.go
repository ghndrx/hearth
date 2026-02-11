package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/contrib/websocket"
	"github.com/google/uuid"
)

// Middleware contains all middleware handlers
type Middleware struct {
	jwtSecret []byte
}

// NewMiddleware creates middleware with dependencies
func NewMiddleware(jwtSecret string) *Middleware {
	return &Middleware{
		jwtSecret: []byte(jwtSecret),
	}
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
	
	token := parts[1]
	
	// TODO: Validate JWT token
	// For now, try to parse as UUID directly (dev mode)
	userID, err := uuid.Parse(token)
	if err != nil {
		// TODO: Real JWT validation
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "invalid token",
		})
	}
	
	c.Locals("userID", userID)
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
	return func(c *fiber.Ctx) error {
		// TODO: Implement rate limiting with Redis
		return c.Next()
	}
}

// CORS adds CORS headers
func (m *Middleware) CORS() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("Access-Control-Allow-Origin", "*")
		c.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		
		if c.Method() == "OPTIONS" {
			return c.SendStatus(fiber.StatusNoContent)
		}
		
		return c.Next()
	}
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
		// TODO: Structured logging
		return c.Next()
	}
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
