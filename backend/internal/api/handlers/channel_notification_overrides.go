package handlers

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
)

// ChannelNotificationOverrideService defines the interface for channel override operations
type ChannelNotificationOverrideService interface {
	GetChannelOverride(ctx context.Context, userID, channelID uuid.UUID) (*models.ChannelNotificationOverride, error)
	SetChannelOverride(ctx context.Context, userID, channelID uuid.UUID, level models.ChannelNotificationLevel) (*models.ChannelNotificationOverride, error)
	ClearChannelOverride(ctx context.Context, userID, channelID uuid.UUID) error
	ListChannelOverrides(ctx context.Context, userID uuid.UUID) ([]models.ChannelNotificationOverride, error)
}

// ChannelNotificationOverrideHandler handles per-channel notification override HTTP requests
type ChannelNotificationOverrideHandler struct {
	service ChannelNotificationOverrideService
}

// NewChannelNotificationOverrideHandler creates a new handler
func NewChannelNotificationOverrideHandler(service ChannelNotificationOverrideService) *ChannelNotificationOverrideHandler {
	return &ChannelNotificationOverrideHandler{service: service}
}

// GetChannelOverride returns the notification override for a specific channel
// @Summary Get channel notification override
// @Description Returns the notification override level for a specific channel
// @Tags Notification Overrides
// @Produce json
// @Param channel_id path string true "Channel ID (UUID)"
// @Success 200 {object} models.ChannelNotificationOverride
// @Failure 400 {object} fiber.Map "Invalid channel ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /users/@me/notification-overrides/{channel_id} [get]
func (h *ChannelNotificationOverrideHandler) GetChannelOverride(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	channelID, err := uuid.Parse(c.Params("channel_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	override, err := h.service.GetChannelOverride(c.Context(), userID, channelID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get channel override",
		})
	}

	return c.JSON(override)
}

// SetChannelOverride sets or updates the notification override for a channel
// @Summary Set channel notification override
// @Description Sets the notification level override for a specific channel
// @Tags Notification Overrides
// @Accept json
// @Produce json
// @Param channel_id path string true "Channel ID (UUID)"
// @Param body body models.SetChannelOverrideRequest true "Override settings"
// @Success 200 {object} models.ChannelNotificationOverride
// @Failure 400 {object} fiber.Map "Invalid request"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /users/@me/notification-overrides/{channel_id} [put]
func (h *ChannelNotificationOverrideHandler) SetChannelOverride(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	channelID, err := uuid.Parse(c.Params("channel_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	var req models.SetChannelOverrideRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Validate notification level
	if req.NotificationLevel != models.ChannelNotificationLevelAllMessages &&
		req.NotificationLevel != models.ChannelNotificationLevelMentionsOnly &&
		req.NotificationLevel != models.ChannelNotificationLevelNothing {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid notification_level. must be one of: all_messages, mentions_only, nothing",
		})
	}

	override, err := h.service.SetChannelOverride(c.Context(), userID, channelID, req.NotificationLevel)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to set channel override",
		})
	}

	return c.JSON(override)
}

// ClearChannelOverride removes the notification override for a channel
// @Summary Clear channel notification override
// @Description Removes the notification override for a specific channel (reverts to default)
// @Tags Notification Overrides
// @Param channel_id path string true "Channel ID (UUID)"
// @Success 204 "Override cleared"
// @Failure 400 {object} fiber.Map "Invalid channel ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /users/@me/notification-overrides/{channel_id} [delete]
func (h *ChannelNotificationOverrideHandler) ClearChannelOverride(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	channelID, err := uuid.Parse(c.Params("channel_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	if err := h.service.ClearChannelOverride(c.Context(), userID, channelID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to clear channel override",
		})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// ListChannelOverrides returns all channel notification overrides for the current user
// @Summary List channel notification overrides
// @Description Returns all channel notification overrides for the current user
// @Tags Notification Overrides
// @Produce json
// @Success 200 {object} models.ListChannelOverridesResponse
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /users/@me/notification-overrides [get]
func (h *ChannelNotificationOverrideHandler) ListChannelOverrides(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	overrides, err := h.service.ListChannelOverrides(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to list channel overrides",
		})
	}

	return c.JSON(models.ListChannelOverridesResponse{
		Overrides: overrides,
	})
}
