package middleware

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/contrib/websocket"
	"github.com/golang-jwt/jwt/v5"
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

// Claims represents JWT claims
type Claims struct {
	jwt.RegisteredClaims
	UserID   string `json:"uid"`
	Username string `json:"usr"`
	Type     string `json:"typ"`
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
