package handlers

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
)

// ChannelMuteServiceInterface defines the methods needed from UserChannelSettingsService
type ChannelMuteServiceInterface interface {
	SetChannelMuted(ctx context.Context, userID, channelID uuid.UUID, muted bool) (*models.UserChannelSettings, error)
	IsChannelMuted(ctx context.Context, userID, channelID uuid.UUID) (bool, error)
	GetMutedChannelIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}

// ChannelMuteHandler handles channel mute-related HTTP requests
type ChannelMuteHandler struct {
	muteService ChannelMuteServiceInterface
}

// NewChannelMuteHandler creates a new channel mute handler
func NewChannelMuteHandler(muteService ChannelMuteServiceInterface) *ChannelMuteHandler {
	return &ChannelMuteHandler{
		muteService: muteService,
	}
}

// SetChannelMute sets the mute state for a channel
// @Summary Mute or unmute a channel
// @Description Sets the mute state for a channel for the current user
// @Tags Channel Settings
// @Accept json
// @Produce json
// @Param id path string true "Channel ID (UUID)"
// @Param body body models.UpdateChannelMuteRequest true "Mute state"
// @Success 200 {object} models.UserChannelSettings "Updated settings"
// @Failure 400 {object} fiber.Map "Invalid request"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /users/@me/channels/{id}/mute [put]
func (h *ChannelMuteHandler) SetChannelMute(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	channelID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	var req models.UpdateChannelMuteRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	settings, err := h.muteService.SetChannelMuted(c.Context(), userID, channelID, req.Muted)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to update channel mute state",
		})
	}

	return c.JSON(settings)
}

// GetChannelMuteState returns whether a channel is muted
// @Summary Get channel mute state
// @Description Returns whether the channel is muted for the current user
// @Tags Channel Settings
// @Produce json
// @Param id path string true "Channel ID (UUID)"
// @Success 200 {object} fiber.Map "Mute state"
// @Failure 400 {object} fiber.Map "Invalid channel ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /users/@me/channels/{id}/mute [get]
func (h *ChannelMuteHandler) GetChannelMuteState(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	channelID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	muted, err := h.muteService.IsChannelMuted(c.Context(), userID, channelID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get channel mute state",
		})
	}

	return c.JSON(fiber.Map{
		"channel_id": channelID,
		"muted":      muted,
	})
}

// GetMutedChannels returns all muted channel IDs for the current user
// @Summary Get muted channels
// @Description Returns a list of all muted channel IDs for the current user
// @Tags Channel Settings
// @Produce json
// @Success 200 {object} fiber.Map "List of muted channel IDs"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /users/@me/channels/muted [get]
func (h *ChannelMuteHandler) GetMutedChannels(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	ids, err := h.muteService.GetMutedChannelIDs(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get muted channels",
		})
	}

	if ids == nil {
		ids = []uuid.UUID{}
	}

	return c.JSON(fiber.Map{
		"muted_channel_ids": ids,
	})
}
