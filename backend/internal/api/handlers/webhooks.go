package handlers

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// WebhookHandlers handles webhook-related HTTP requests
type WebhookHandlers struct {
	webhookService *services.WebhookService
}

// NewWebhookHandlers creates new webhook handlers
func NewWebhookHandlers(webhookService *services.WebhookService) *WebhookHandlers {
	return &WebhookHandlers{
		webhookService: webhookService,
	}
}

// WebhookResponse represents a webhook in API responses
type WebhookResponse struct {
	ID            string    `json:"id"`
	Type          int       `json:"type"`
	Name          string    `json:"name"`
	ChannelID     string    `json:"channel_id"`
	ServerID      *string   `json:"guild_id,omitempty"`
	Token         string    `json:"token,omitempty"`
	AvatarURL     *string   `json:"avatar,omitempty"`
	ApplicationID *string   `json:"application_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// CreateWebhook creates a new webhook for a channel
// POST /channels/:channelID/webhooks
func (h *WebhookHandlers) CreateWebhook(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
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

	webhook, err := h.webhookService.CreateWebhook(c.Context(), &services.CreateWebhookRequest{
		ChannelID: channelID,
		CreatorID: userID,
		Name:      req.Name,
		Avatar:    req.Avatar,
	})
	if err != nil {
		return handleWebhookError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(toWebhookResponse(webhook, true))
}

// GetChannelWebhooks returns all webhooks for a channel
// GET /channels/:channelID/webhooks
func (h *WebhookHandlers) GetChannelWebhooks(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	channelID, err := uuid.Parse(c.Params("channelID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid channel ID",
		})
	}

	webhooks, err := h.webhookService.GetChannelWebhooks(c.Context(), channelID, userID)
	if err != nil {
		return handleWebhookError(c, err)
	}

	responses := make([]WebhookResponse, len(webhooks))
	for i, w := range webhooks {
		responses[i] = toWebhookResponse(w, false)
	}

	return c.JSON(responses)
}

// GetServerWebhooks returns all webhooks for a server
// GET /guilds/:serverID/webhooks
func (h *WebhookHandlers) GetServerWebhooks(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	serverID, err := uuid.Parse(c.Params("serverID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid server ID",
		})
	}

	webhooks, err := h.webhookService.GetServerWebhooks(c.Context(), serverID, userID)
	if err != nil {
		return handleWebhookError(c, err)
	}

	responses := make([]WebhookResponse, len(webhooks))
	for i, w := range webhooks {
		responses[i] = toWebhookResponse(w, false)
	}

	return c.JSON(responses)
}

// GetWebhook returns a specific webhook by ID
// GET /webhooks/:webhookID
func (h *WebhookHandlers) GetWebhook(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	webhookID, err := uuid.Parse(c.Params("webhookID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid webhook ID",
		})
	}

	webhook, err := h.webhookService.GetWebhook(c.Context(), webhookID, userID)
	if err != nil {
		return handleWebhookError(c, err)
	}

	return c.JSON(toWebhookResponse(webhook, false))
}

// UpdateWebhook updates a webhook's properties
// PATCH /webhooks/:webhookID
func (h *WebhookHandlers) UpdateWebhook(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	webhookID, err := uuid.Parse(c.Params("webhookID"))
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

	// Validate name if provided
	if req.Name != nil && (len(*req.Name) == 0 || len(*req.Name) > 80) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Name must be between 1 and 80 characters",
		})
	}

	// Parse channel ID if provided
	var channelID *uuid.UUID
	if req.ChannelID != nil {
		id, err := uuid.Parse(*req.ChannelID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid channel ID",
			})
		}
		channelID = &id
	}

	webhook, err := h.webhookService.UpdateWebhook(c.Context(), webhookID, userID, &services.UpdateWebhookRequest{
		Name:      req.Name,
		Avatar:    req.Avatar,
		ChannelID: channelID,
	})
	if err != nil {
		return handleWebhookError(c, err)
	}

	return c.JSON(toWebhookResponse(webhook, false))
}

// DeleteWebhook deletes a webhook
// DELETE /webhooks/:webhookID
func (h *WebhookHandlers) DeleteWebhook(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	webhookID, err := uuid.Parse(c.Params("webhookID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid webhook ID",
		})
	}

	if err := h.webhookService.DeleteWebhook(c.Context(), webhookID, userID); err != nil {
		return handleWebhookError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ExecuteWebhook executes a webhook (sends a message)
// POST /webhooks/:webhookID/:token
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

	message, err := h.webhookService.ExecuteWebhook(c.Context(), webhookID, token, &services.ExecuteWebhookRequest{
		Content:   req.Content,
		Username:  req.Username,
		AvatarURL: req.AvatarURL,
		TTS:       req.TTS,
	})
	if err != nil {
		return handleWebhookError(c, err)
	}

	// Return 204 if wait=false (default), or message if wait=true
	if c.Query("wait") == "true" {
		return c.JSON(fiber.Map{
			"id":         message.ID.String(),
			"content":    message.Content,
			"channel_id": message.ChannelID.String(),
			"webhook_id": webhookID.String(),
			"timestamp":  message.CreatedAt,
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// toWebhookResponse converts a webhook model to an API response
func toWebhookResponse(w *models.Webhook, includeToken bool) WebhookResponse {
	resp := WebhookResponse{
		ID:        w.ID.String(),
		Type:      int(w.Type),
		Name:      w.Name,
		ChannelID: w.ChannelID.String(),
		AvatarURL: w.Avatar,
		CreatedAt: w.CreatedAt,
	}

	if w.ServerID != nil {
		serverID := w.ServerID.String()
		resp.ServerID = &serverID
	}

	if w.ApplicationID != nil {
		appID := w.ApplicationID.String()
		resp.ApplicationID = &appID
	}

	if includeToken {
		resp.Token = w.Token
	}

	return resp
}

// handleWebhookError maps service errors to HTTP responses
func handleWebhookError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, services.ErrWebhookNotFound):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Webhook not found",
		})
	case errors.Is(err, services.ErrChannelNotFound):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Channel not found",
		})
	case errors.Is(err, services.ErrNotServerMember):
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You are not a member of this server",
		})
	case errors.Is(err, services.ErrNoPermission):
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Missing permissions",
		})
	case errors.Is(err, services.ErrWebhookNameTooLong):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Webhook name must be between 1 and 80 characters",
		})
	case errors.Is(err, services.ErrTooManyWebhooks):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Maximum 10 webhooks per channel",
		})
	case errors.Is(err, services.ErrInvalidWebhookToken):
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid webhook token",
		})
	case errors.Is(err, services.ErrEmptyMessage):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Message content is required",
		})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}
}
