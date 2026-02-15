package handlers

import (
	"context"
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// WebhookService defines the interface for webhook operations
type WebhookService interface {
	CreateWebhook(ctx context.Context, req *services.CreateWebhookRequest) (*models.Webhook, error)
	GetChannelWebhooks(ctx context.Context, channelID, requesterID uuid.UUID) ([]*models.Webhook, error)
	GetServerWebhooks(ctx context.Context, serverID, requesterID uuid.UUID) ([]*models.Webhook, error)
	GetWebhook(ctx context.Context, webhookID, requesterID uuid.UUID) (*models.Webhook, error)
	UpdateWebhook(ctx context.Context, webhookID, requesterID uuid.UUID, req *services.UpdateWebhookRequest) (*models.Webhook, error)
	DeleteWebhook(ctx context.Context, webhookID, requesterID uuid.UUID) error
	ExecuteWebhook(ctx context.Context, webhookID uuid.UUID, token string, req *services.ExecuteWebhookRequest) (*models.Message, error)
}

// WebhookHandlers handles webhook-related HTTP requests
type WebhookHandlers struct {
	webhookService WebhookService
}

// NewWebhookHandlers creates new webhook handlers
func NewWebhookHandlers(webhookService WebhookService) *WebhookHandlers {
	return &WebhookHandlers{
		webhookService: webhookService,
	}
}

// WebhookResponse represents a webhook in API responses
type WebhookResponse struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	ChannelID string  `json:"channel_id"`
	ServerID  *string `json:"guild_id,omitempty"`
	Token     string  `json:"token,omitempty"`
	AvatarURL *string `json:"avatar,omitempty"`
	Type      int     `json:"type"`
}

// toWebhookResponse converts a models.Webhook to WebhookResponse
func toWebhookResponse(webhook *models.Webhook, includeToken bool) *WebhookResponse {
	if webhook == nil {
		return nil
	}

	resp := &WebhookResponse{
		ID:        webhook.ID.String(),
		Name:      webhook.Name,
		ChannelID: webhook.ChannelID.String(),
		AvatarURL: webhook.Avatar,
		Type:      int(webhook.Type),
	}

	if webhook.ServerID != nil {
		serverID := webhook.ServerID.String()
		resp.ServerID = &serverID
	}

	if includeToken {
		resp.Token = webhook.Token
	}

	return resp
}

// CreateWebhook creates a new webhook
func (h *WebhookHandlers) CreateWebhook(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	channelID, err := uuid.Parse(c.Params("id"))
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

	webhook, err := h.webhookService.CreateWebhook(c.Context(), &services.CreateWebhookRequest{
		ChannelID: channelID,
		CreatorID: userID,
		Name:      req.Name,
		Avatar:    req.Avatar,
	})

	if err != nil {
		switch {
		case errors.Is(err, services.ErrChannelNotFound):
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Channel not found",
			})
		case errors.Is(err, services.ErrNotServerMember):
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "You don't have permission to create webhooks in this channel",
			})
		case errors.Is(err, services.ErrWebhookNameTooLong):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Webhook name must be between 1 and 80 characters",
			})
		case errors.Is(err, services.ErrTooManyWebhooks):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Maximum number of webhooks (10) reached for this channel",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to create webhook",
			})
		}
	}

	return c.Status(fiber.StatusCreated).JSON(toWebhookResponse(webhook, true))
}

// GetChannelWebhooks returns all webhooks for a channel
func (h *WebhookHandlers) GetChannelWebhooks(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	channelID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid channel ID",
		})
	}

	webhooks, err := h.webhookService.GetChannelWebhooks(c.Context(), channelID, userID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrChannelNotFound):
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Channel not found",
			})
		case errors.Is(err, services.ErrNotServerMember):
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "You don't have access to this channel",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to get webhooks",
			})
		}
	}

	responses := make([]*WebhookResponse, len(webhooks))
	for i, webhook := range webhooks {
		responses[i] = toWebhookResponse(webhook, false)
	}

	return c.JSON(responses)
}

// GetServerWebhooks returns all webhooks for a server
func (h *WebhookHandlers) GetServerWebhooks(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid server ID",
		})
	}

	webhooks, err := h.webhookService.GetServerWebhooks(c.Context(), serverID, userID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrNotServerMember):
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "You don't have access to this server",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to get webhooks",
			})
		}
	}

	responses := make([]*WebhookResponse, len(webhooks))
	for i, webhook := range webhooks {
		responses[i] = toWebhookResponse(webhook, false)
	}

	return c.JSON(responses)
}

// GetWebhook returns a specific webhook
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
		switch {
		case errors.Is(err, services.ErrWebhookNotFound):
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Webhook not found",
			})
		case errors.Is(err, services.ErrNotServerMember):
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "You don't have access to this webhook",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to get webhook",
			})
		}
	}

	return c.JSON(toWebhookResponse(webhook, false))
}

// UpdateWebhook updates a webhook
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

	updateReq := &services.UpdateWebhookRequest{
		Name:   req.Name,
		Avatar: req.Avatar,
	}

	if req.ChannelID != nil {
		channelID, err := uuid.Parse(*req.ChannelID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid channel ID",
			})
		}
		updateReq.ChannelID = &channelID
	}

	webhook, err := h.webhookService.UpdateWebhook(c.Context(), webhookID, userID, updateReq)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrWebhookNotFound):
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Webhook not found",
			})
		case errors.Is(err, services.ErrNotServerMember):
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "You don't have permission to update this webhook",
			})
		case errors.Is(err, services.ErrChannelNotFound):
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Channel not found",
			})
		case errors.Is(err, services.ErrNoPermission):
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Cannot move webhook to a different server",
			})
		case errors.Is(err, services.ErrWebhookNameTooLong):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Webhook name must be between 1 and 80 characters",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to update webhook",
			})
		}
	}

	return c.JSON(toWebhookResponse(webhook, false))
}

// DeleteWebhook deletes a webhook
func (h *WebhookHandlers) DeleteWebhook(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	webhookID, err := uuid.Parse(c.Params("webhookID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid webhook ID",
		})
	}

	err = h.webhookService.DeleteWebhook(c.Context(), webhookID, userID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrWebhookNotFound):
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Webhook not found",
			})
		case errors.Is(err, services.ErrNotServerMember):
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "You don't have permission to delete this webhook",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to delete webhook",
			})
		}
	}

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

	message, err := h.webhookService.ExecuteWebhook(c.Context(), webhookID, token, &services.ExecuteWebhookRequest{
		Content:   req.Content,
		Username:  req.Username,
		AvatarURL: req.AvatarURL,
		TTS:       req.TTS,
	})

	if err != nil {
		switch {
		case errors.Is(err, services.ErrWebhookNotFound):
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Webhook not found",
			})
		case errors.Is(err, services.ErrInvalidWebhookToken):
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid webhook token",
			})
		case errors.Is(err, services.ErrEmptyMessage):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Message content cannot be empty",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to execute webhook",
			})
		}
	}

	// Return 204 if wait=false (default), or message if wait=true
	if c.Query("wait") == "true" {
		return c.JSON(message)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
