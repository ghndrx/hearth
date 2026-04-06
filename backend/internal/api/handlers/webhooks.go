package handlers

import (
	"context"
	"log"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// WebhookServiceInterface defines the interface for webhook service operations
type WebhookServiceInterface interface {
	CreateWebhook(ctx context.Context, req *services.CreateWebhookRequest) (*models.Webhook, error)
	GetWebhook(ctx context.Context, webhookID uuid.UUID, requesterID uuid.UUID) (*models.Webhook, error)
	GetChannelWebhooks(ctx context.Context, channelID uuid.UUID, requesterID uuid.UUID) ([]*models.Webhook, error)
	GetServerWebhooks(ctx context.Context, serverID uuid.UUID, requesterID uuid.UUID) ([]*models.Webhook, error)
	UpdateWebhook(ctx context.Context, webhookID uuid.UUID, requesterID uuid.UUID, req *services.UpdateWebhookRequest) (*models.Webhook, error)
	DeleteWebhook(ctx context.Context, webhookID uuid.UUID, requesterID uuid.UUID) error
	ExecuteWebhook(ctx context.Context, webhookID uuid.UUID, token string, req *services.ExecuteWebhookRequest) (*models.Message, error)
	ExecuteWebhookWithRetry(ctx context.Context, webhookID uuid.UUID, token string, req *services.ExecuteWebhookRequest) (*models.Message, error)
	GetWebhookStats(ctx context.Context, webhookID uuid.UUID, requesterID uuid.UUID) (*models.WebhookDeliveryStats, error)
	GetWebhookDeliveries(ctx context.Context, webhookID uuid.UUID, requesterID uuid.UUID, limit, offset int) ([]*models.WebhookDelivery, error)
	TestWebhook(ctx context.Context, webhookID uuid.UUID, requesterID uuid.UUID) (*models.Message, error)
	CheckRateLimit(ctx context.Context, webhookID uuid.UUID) error
}

// WebhookHandlers handles webhook-related HTTP requests
type WebhookHandlers struct {
	webhookService WebhookServiceInterface
}

// NewWebhookHandlers creates new webhook handlers
func NewWebhookHandlers(webhookService WebhookServiceInterface) *WebhookHandlers {
	return &WebhookHandlers{
		webhookService: webhookService,
	}
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

func webhookToResponse(webhook *models.Webhook) WebhookResponse {
	resp := WebhookResponse{
		ID:        webhook.ID.String(),
		Name:      webhook.Name,
		ChannelID: webhook.ChannelID.String(),
		Token:     webhook.Token,
		AvatarURL: webhook.Avatar,
		Type:      int(webhook.Type),
	}
	if webhook.ServerID != nil {
		resp.ServerID = webhook.ServerID.String()
	}
	return resp
}

// CreateWebhook creates a new webhook
// @Summary Create a new webhook
// @Description Creates a new webhook for the specified channel
// @Tags Webhooks
// @Accept json
// @Produce json
// @Param channelID path string true "Channel ID"
// @Param body body struct{Name string `json:"name"`; Avatar *string `json:"avatar,omitempty"`} true "Webhook creation data"
// @Success 201 {object} WebhookResponse "Webhook created successfully"
// @Failure 400 {object} fiber.Map "Invalid channel ID or request body"
// @Failure 403 {object} fiber.Map "Not a server member or missing MANAGE_WEBHOOKS permission"
// @Failure 404 {object} fiber.Map "Channel not found"
// @Failure 429 {object} fiber.Map "Rate limited"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{channelID}/webhooks [post]
func (h *WebhookHandlers) CreateWebhook(c *fiber.Ctx) error {
	channelID, err := uuid.Parse(c.Params("channelID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid channel ID",
		})
	}

	userID := c.Locals("userID").(uuid.UUID)

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
		if err == services.ErrChannelNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Channel not found",
			})
		}
		if err == services.ErrNotServerMember {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "You must be a member of the server to create webhooks",
			})
		}
		if err == services.ErrMissingManageWebhooks {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Missing MANAGE_WEBHOOKS permission",
			})
		}
		if err == services.ErrWebhookNameTooLong || err == services.ErrTooManyWebhooks {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		log.Printf("Error creating webhook: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create webhook",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(webhookToResponse(webhook))
}

// GetChannelWebhooks returns all webhooks for a channel
// @Summary Get channel webhooks
// @Description Returns all webhooks for the specified channel
// @Tags Webhooks
// @Produce json
// @Param channelID path string true "Channel ID"
// @Success 200 {array} WebhookResponse "List of webhooks"
// @Failure 400 {object} fiber.Map "Invalid channel ID"
// @Failure 403 {object} fiber.Map "Not a server member"
// @Failure 404 {object} fiber.Map "Channel not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{channelID}/webhooks [get]
func (h *WebhookHandlers) GetChannelWebhooks(c *fiber.Ctx) error {
	channelID, err := uuid.Parse(c.Params("channelID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid channel ID",
		})
	}

	userID := c.Locals("userID").(uuid.UUID)

	webhooks, err := h.webhookService.GetChannelWebhooks(c.Context(), channelID, userID)
	if err != nil {
		if err == services.ErrChannelNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Channel not found",
			})
		}
		if err == services.ErrNotServerMember {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "You must be a member of the server to view webhooks",
			})
		}
		log.Printf("Error getting channel webhooks: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get webhooks",
		})
	}

	response := make([]WebhookResponse, len(webhooks))
	for i, w := range webhooks {
		response[i] = webhookToResponse(w)
	}
	return c.JSON(response)
}

// GetServerWebhooks returns all webhooks for a server
// @Summary Get server webhooks
// @Description Returns all webhooks for the specified server (requires MANAGE_WEBHOOKS permission)
// @Tags Webhooks
// @Produce json
// @Param serverID path string true "Server ID"
// @Success 200 {array} WebhookResponse "List of webhooks"
// @Failure 400 {object} fiber.Map "Invalid server ID"
// @Failure 403 {object} fiber.Map "Not a server member or missing MANAGE_WEBHOOKS permission"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{serverID}/webhooks [get]
func (h *WebhookHandlers) GetServerWebhooks(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("serverID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid server ID",
		})
	}

	userID := c.Locals("userID").(uuid.UUID)

	webhooks, err := h.webhookService.GetServerWebhooks(c.Context(), serverID, userID)
	if err != nil {
		if err == services.ErrNotServerMember {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "You must be a member of the server to view webhooks",
			})
		}
		if err == services.ErrMissingManageWebhooks {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Missing MANAGE_WEBHOOKS permission",
			})
		}
		log.Printf("Error getting server webhooks: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get webhooks",
		})
	}

	response := make([]WebhookResponse, len(webhooks))
	for i, w := range webhooks {
		response[i] = webhookToResponse(w)
	}
	return c.JSON(response)
}

// GetWebhook returns a specific webhook
// @Summary Get a webhook
// @Description Returns a specific webhook by ID
// @Tags Webhooks
// @Produce json
// @Param webhookID path string true "Webhook ID"
// @Success 200 {object} WebhookResponse "Webhook details"
// @Failure 400 {object} fiber.Map "Invalid webhook ID"
// @Failure 403 {object} fiber.Map "Not a server member"
// @Failure 404 {object} fiber.Map "Webhook or channel not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /webhooks/{webhookID} [get]
func (h *WebhookHandlers) GetWebhook(c *fiber.Ctx) error {
	webhookID, err := uuid.Parse(c.Params("webhookID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid webhook ID",
		})
	}

	userID := c.Locals("userID").(uuid.UUID)

	webhook, err := h.webhookService.GetWebhook(c.Context(), webhookID, userID)
	if err != nil {
		if err == services.ErrWebhookNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Webhook not found",
			})
		}
		if err == services.ErrChannelNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Channel not found",
			})
		}
		if err == services.ErrNotServerMember {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "You must be a member of the server to view this webhook",
			})
		}
		log.Printf("Error getting webhook: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get webhook",
		})
	}

	return c.JSON(webhookToResponse(webhook))
}

// UpdateWebhook updates a webhook
// @Summary Update a webhook
// @Description Updates a webhook's name, avatar, or channel
// @Tags Webhooks
// @Accept json
// @Produce json
// @Param webhookID path string true "Webhook ID"
// @Param body body struct{Name *string `json:"name,omitempty"`; Avatar *string `json:"avatar,omitempty"`; ChannelID *string `json:"channel_id,omitempty"`} true "Webhook update data"
// @Success 200 {object} WebhookResponse "Updated webhook"
// @Failure 400 {object} fiber.Map "Invalid webhook ID, channel ID, or request body"
// @Failure 403 {object} fiber.Map "Not a server member, missing MANAGE_WEBHOOKS permission, or cannot move to different server"
// @Failure 404 {object} fiber.Map "Webhook or channel not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /webhooks/{webhookID} [patch]
func (h *WebhookHandlers) UpdateWebhook(c *fiber.Ctx) error {
	webhookID, err := uuid.Parse(c.Params("webhookID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid webhook ID",
		})
	}

	userID := c.Locals("userID").(uuid.UUID)

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

	var channelID *uuid.UUID
	if req.ChannelID != nil {
		parsed, err := uuid.Parse(*req.ChannelID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid channel ID",
			})
		}
		channelID = &parsed
	}

	updateReq := &services.UpdateWebhookRequest{
		Name:      req.Name,
		Avatar:    req.Avatar,
		ChannelID: channelID,
	}

	webhook, err := h.webhookService.UpdateWebhook(c.Context(), webhookID, userID, updateReq)
	if err != nil {
		if err == services.ErrWebhookNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Webhook not found",
			})
		}
		if err == services.ErrChannelNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Channel not found",
			})
		}
		if err == services.ErrNotServerMember {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "You must be a member of the server to update webhooks",
			})
		}
		if err == services.ErrMissingManageWebhooks {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Missing MANAGE_WEBHOOKS permission",
			})
		}
		if err == services.ErrWebhookNameTooLong {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		if err == services.ErrNoPermission {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Cannot move webhook to a different server",
			})
		}
		log.Printf("Error updating webhook: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update webhook",
		})
	}

	return c.JSON(webhookToResponse(webhook))
}

// DeleteWebhook deletes a webhook
// @Summary Delete a webhook
// @Description Deletes a webhook permanently
// @Tags Webhooks
// @Param webhookID path string true "Webhook ID"
// @Success 204 "Webhook deleted successfully"
// @Failure 400 {object} fiber.Map "Invalid webhook ID"
// @Failure 403 {object} fiber.Map "Not a server member or missing MANAGE_WEBHOOKS permission"
// @Failure 404 {object} fiber.Map "Webhook or channel not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /webhooks/{webhookID} [delete]
func (h *WebhookHandlers) DeleteWebhook(c *fiber.Ctx) error {
	webhookID, err := uuid.Parse(c.Params("webhookID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid webhook ID",
		})
	}

	userID := c.Locals("userID").(uuid.UUID)

	err = h.webhookService.DeleteWebhook(c.Context(), webhookID, userID)
	if err != nil {
		if err == services.ErrWebhookNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Webhook not found",
			})
		}
		if err == services.ErrChannelNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Channel not found",
			})
		}
		if err == services.ErrNotServerMember {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "You must be a member of the server to delete webhooks",
			})
		}
		if err == services.ErrMissingManageWebhooks {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Missing MANAGE_WEBHOOKS permission",
			})
		}
		log.Printf("Error deleting webhook: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete webhook",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ExecuteWebhookRequest represents the request body for executing a webhook
type ExecuteWebhookRequest struct {
	Content         string          `json:"content,omitempty"`
	Username        *string         `json:"username,omitempty"`
	AvatarURL       *string         `json:"avatar_url,omitempty"`
	TTS             bool            `json:"tts,omitempty"`
	Embeds          []models.Embed  `json:"embeds,omitempty"`
	AllowedMentions *models.WebhookMessage `json:"allowed_mentions,omitempty"`
	ThreadName      string          `json:"thread_name,omitempty"`
}

// ExecuteWebhook executes a webhook (send a message)
// @Summary Execute a webhook
// @Description Sends a message through a webhook using its token
// @Tags Webhooks
// @Accept json
// @Produce json
// @Param webhookID path string true "Webhook ID"
// @Param token path string true "Webhook token"
// @Param wait query boolean false "Wait for message to be created and return it"
// @Param body body ExecuteWebhookRequest true "Message data"
// @Success 200 {object} fiber.Map "Message created (when wait=true)"
// @Success 204 "Message sent (when wait=false)"
// @Failure 400 {object} fiber.Map "Invalid webhook ID, token, or empty message"
// @Failure 401 {object} fiber.Map "Invalid webhook token"
// @Failure 404 {object} fiber.Map "Webhook not found"
// @Failure 429 {object} fiber.Map "Rate limited"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /webhooks/{webhookID}/{token} [post]
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

	var req ExecuteWebhookRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	useRetry := c.Query("retry", "true") == "true"
	
	var message *models.Message
	if useRetry {
		message, err = h.webhookService.ExecuteWebhookWithRetry(c.Context(), webhookID, token, &services.ExecuteWebhookRequest{
			Content:         req.Content,
			Username:        req.Username,
			AvatarURL:       req.AvatarURL,
			TTS:             req.TTS,
			Embeds:          req.Embeds,
			AllowedMentions: req.AllowedMentions,
			ThreadName:      req.ThreadName,
		})
	} else {
		message, err = h.webhookService.ExecuteWebhook(c.Context(), webhookID, token, &services.ExecuteWebhookRequest{
			Content:         req.Content,
			Username:        req.Username,
			AvatarURL:       req.AvatarURL,
			TTS:             req.TTS,
			Embeds:          req.Embeds,
			AllowedMentions: req.AllowedMentions,
			ThreadName:      req.ThreadName,
		})
	}
	
	if err != nil {
		if err == services.ErrWebhookNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Webhook not found",
			})
		}
		if err == services.ErrInvalidWebhookToken {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid webhook token",
			})
		}
		if err == services.ErrEmptyMessage {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Content is required",
			})
		}
		if err == services.ErrWebhookRateLimited {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Webhook is rate limited, please try again later",
				"retry_after": 60,
			})
		}
		if err == services.ErrTooManyEmbeds {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Maximum 10 embeds allowed",
			})
		}
		log.Printf("Error executing webhook: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to execute webhook",
		})
	}

	if c.Query("wait") == "true" {
		return c.JSON(fiber.Map{
			"id":         message.ID.String(),
			"content":    message.Content,
			"webhook_id": webhookID.String(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// GetWebhookStats returns delivery statistics for a webhook
// @Summary Get webhook delivery statistics
// @Description Returns delivery statistics for the specified webhook
// @Tags Webhooks
// @Produce json
// @Param webhookID path string true "Webhook ID"
// @Success 200 {object} models.WebhookDeliveryStats "Webhook delivery statistics"
// @Failure 400 {object} fiber.Map "Invalid webhook ID"
// @Failure 403 {object} fiber.Map "Not a server member or missing MANAGE_WEBHOOKS permission"
// @Failure 404 {object} fiber.Map "Webhook not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /webhooks/{webhookID}/stats [get]
func (h *WebhookHandlers) GetWebhookStats(c *fiber.Ctx) error {
	webhookID, err := uuid.Parse(c.Params("webhookID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid webhook ID",
		})
	}

	userID := c.Locals("userID").(uuid.UUID)

	stats, err := h.webhookService.GetWebhookStats(c.Context(), webhookID, userID)
	if err != nil {
		if err == services.ErrWebhookNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Webhook not found",
			})
		}
		if err == services.ErrNotServerMember {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "You must be a member of the server to view webhook stats",
			})
		}
		if err == services.ErrMissingManageWebhooks {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Missing MANAGE_WEBHOOKS permission",
			})
		}
		log.Printf("Error getting webhook stats: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get webhook stats",
		})
	}

	return c.JSON(stats)
}

// GetWebhookDeliveries returns delivery history for a webhook
// @Summary Get webhook delivery history
// @Description Returns the delivery history for the specified webhook
// @Tags Webhooks
// @Produce json
// @Param webhookID path string true "Webhook ID"
// @Param limit query int false "Number of results to return (default 50, max 100)"
// @Param offset query int false "Offset for pagination"
// @Success 200 {array} models.WebhookDelivery "List of webhook deliveries"
// @Failure 400 {object} fiber.Map "Invalid webhook ID"
// @Failure 403 {object} fiber.Map "Not a server member or missing MANAGE_WEBHOOKS permission"
// @Failure 404 {object} fiber.Map "Webhook not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /webhooks/{webhookID}/deliveries [get]
func (h *WebhookHandlers) GetWebhookDeliveries(c *fiber.Ctx) error {
	webhookID, err := uuid.Parse(c.Params("webhookID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid webhook ID",
		})
	}

	userID := c.Locals("userID").(uuid.UUID)

	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	offset := 0
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	deliveries, err := h.webhookService.GetWebhookDeliveries(c.Context(), webhookID, userID, limit, offset)
	if err != nil {
		if err == services.ErrWebhookNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Webhook not found",
			})
		}
		if err == services.ErrNotServerMember {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "You must be a member of the server to view webhook deliveries",
			})
		}
		if err == services.ErrMissingManageWebhooks {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Missing MANAGE_WEBHOOKS permission",
			})
		}
		log.Printf("Error getting webhook deliveries: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get webhook deliveries",
		})
	}

	return c.JSON(deliveries)
}

// TestWebhook tests a webhook by sending a test message
// @Summary Test a webhook
// @Description Sends a test message through the webhook
// @Tags Webhooks
// @Produce json
// @Param webhookID path string true "Webhook ID"
// @Success 200 {object} fiber.Map "Test message sent"
// @Failure 400 {object} fiber.Map "Invalid webhook ID"
// @Failure 403 {object} fiber.Map "Not a server member or missing MANAGE_WEBHOOKS permission"
// @Failure 404 {object} fiber.Map "Webhook not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /webhooks/{webhookID}/test [post]
func (h *WebhookHandlers) TestWebhook(c *fiber.Ctx) error {
	webhookID, err := uuid.Parse(c.Params("webhookID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid webhook ID",
		})
	}

	userID := c.Locals("userID").(uuid.UUID)

	message, err := h.webhookService.TestWebhook(c.Context(), webhookID, userID)
	if err != nil {
		if err == services.ErrWebhookNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Webhook not found",
			})
		}
		if err == services.ErrNotServerMember {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "You must be a member of the server to test webhooks",
			})
		}
		if err == services.ErrMissingManageWebhooks {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Missing MANAGE_WEBHOOKS permission",
			})
		}
		log.Printf("Error testing webhook: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to test webhook",
		})
	}

	return c.JSON(fiber.Map{
		"success":    true,
		"message":    "Test message sent successfully",
		"message_id": message.ID.String(),
	})
}
