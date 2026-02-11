package handlers

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// WebhookHandlers handles webhook-related HTTP requests
type WebhookHandlers struct {
	// webhookRepo would go here
}

// NewWebhookHandlers creates new webhook handlers
func NewWebhookHandlers() *WebhookHandlers {
	return &WebhookHandlers{}
}

// WebhookResponse represents a webhook in API responses
type WebhookResponse struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	ChannelID string  `json:"channel_id"`
	ServerID  string  `json:"guild_id"`
	Token     string  `json:"token,omitempty"`
	AvatarURL *string `json:"avatar,omitempty"`
	Type      int     `json:"type"`
}

// CreateWebhook creates a new webhook
func (h *WebhookHandlers) CreateWebhook(c *fiber.Ctx) error {
	channelID, err := uuid.Parse(c.Params("channelID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid channel ID",
		})
	}

	var req struct {
		Name   string  `json:"name"`
		Avatar *string `json:"avatar,omitempty"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Name == "" || len(req.Name) > 80 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Name must be between 1 and 80 characters",
		})
	}

	// Generate webhook token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate token",
		})
	}
	token := hex.EncodeToString(tokenBytes)

	// TODO: Save webhook to database

	return c.Status(fiber.StatusCreated).JSON(WebhookResponse{
		ID:        uuid.New().String(),
		Name:      req.Name,
		ChannelID: channelID.String(),
		ServerID:  "", // Would come from channel lookup
		Token:     token,
		AvatarURL: req.Avatar,
		Type:      1, // Incoming webhook
	})
}

// GetChannelWebhooks returns all webhooks for a channel
func (h *WebhookHandlers) GetChannelWebhooks(c *fiber.Ctx) error {
	_, err := uuid.Parse(c.Params("channelID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid channel ID",
		})
	}

	// TODO: Get webhooks from database
	return c.JSON([]WebhookResponse{})
}

// GetServerWebhooks returns all webhooks for a server
func (h *WebhookHandlers) GetServerWebhooks(c *fiber.Ctx) error {
	_, err := uuid.Parse(c.Params("serverID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid server ID",
		})
	}

	// TODO: Get webhooks from database
	return c.JSON([]WebhookResponse{})
}

// GetWebhook returns a specific webhook
func (h *WebhookHandlers) GetWebhook(c *fiber.Ctx) error {
	_, err := uuid.Parse(c.Params("webhookID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid webhook ID",
		})
	}

	// TODO: Get webhook from database
	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
		"error": "Webhook not found",
	})
}

// UpdateWebhook updates a webhook
func (h *WebhookHandlers) UpdateWebhook(c *fiber.Ctx) error {
	_, err := uuid.Parse(c.Params("webhookID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid webhook ID",
		})
	}

	var req struct {
		Name      *string `json:"name,omitempty"`
		Avatar    *string `json:"avatar,omitempty"`
		ChannelID *string `json:"channel_id,omitempty"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// TODO: Update webhook in database
	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
		"error": "Webhook not found",
	})
}

// DeleteWebhook deletes a webhook
func (h *WebhookHandlers) DeleteWebhook(c *fiber.Ctx) error {
	_, err := uuid.Parse(c.Params("webhookID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid webhook ID",
		})
	}

	// TODO: Delete webhook from database
	return c.SendStatus(fiber.StatusNoContent)
}

// ExecuteWebhook executes a webhook (send a message)
func (h *WebhookHandlers) ExecuteWebhook(c *fiber.Ctx) error {
	webhookID, err := uuid.Parse(c.Params("webhookID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid webhook ID",
		})
	}
	token := c.Params("token")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid webhook token",
		})
	}

	var req struct {
		Content   string  `json:"content,omitempty"`
		Username  *string `json:"username,omitempty"`
		AvatarURL *string `json:"avatar_url,omitempty"`
		TTS       bool    `json:"tts"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Content == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Content is required",
		})
	}

	// TODO: Verify webhook token and send message
	_ = webhookID

	// Return 204 if wait=false (default), or message if wait=true
	if c.Query("wait") == "true" {
		return c.JSON(fiber.Map{
			"id":         uuid.New().String(),
			"content":    req.Content,
			"webhook_id": webhookID.String(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
