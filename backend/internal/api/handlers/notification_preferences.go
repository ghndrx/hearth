package handlers

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
)

// NotificationCoordinatorInterface defines the methods needed from NotificationCoordinator
type NotificationCoordinatorInterface interface {
	GetChannelPreference(ctx context.Context, userID, channelID, serverID uuid.UUID) (*models.ChannelNotificationPreference, error)
	UpdateChannelPreference(ctx context.Context, userID, channelID, serverID uuid.UUID, req *models.UpdateChannelNotificationPreferenceRequest) (*models.ChannelNotificationPreference, error)
	GetServerPreference(ctx context.Context, userID, serverID uuid.UUID) (*models.ServerNotificationPreference, error)
	UpdateServerPreference(ctx context.Context, userID, serverID uuid.UUID, req *models.UpdateServerNotificationPreferenceRequest) (*models.ServerNotificationPreference, error)
}

// ChannelNotificationPreferenceHandler handles per-channel notification preferences
type ChannelNotificationPreferenceHandler struct {
	coordinator NotificationCoordinatorInterface
}

// NewChannelNotificationPreferenceHandler creates a new channel notification preference handler
func NewChannelNotificationPreferenceHandler(coordinator NotificationCoordinatorInterface) *ChannelNotificationPreferenceHandler {
	return &ChannelNotificationPreferenceHandler{coordinator: coordinator}
}

// GetChannelNotificationPreference returns notification preferences for a channel
// @Summary Get channel notification preferences
// @Tags Notification Preferences
// @Produce json
// @Success 200 {object} models.ChannelNotificationPreference
// @Router /users/@me/channels/{channelId}/notifications [get]
func (h *ChannelNotificationPreferenceHandler) GetChannelNotificationPreference(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	channelID, err := uuid.Parse(c.Params("channelId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	serverID, err := uuid.Parse(c.Query("server_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	pref, err := h.coordinator.GetChannelPreference(c.Context(), userID, channelID, serverID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get channel notification preference",
		})
	}

	return c.JSON(pref)
}

// UpdateChannelNotificationPreference updates notification preferences for a channel
// @Summary Update channel notification preferences
// @Tags Notification Preferences
// @Accept json
// @Produce json
// @Success 200 {object} models.ChannelNotificationPreference
// @Router /users/@me/channels/{channelId}/notifications [patch]
func (h *ChannelNotificationPreferenceHandler) UpdateChannelNotificationPreference(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	channelID, err := uuid.Parse(c.Params("channelId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	serverID, err := uuid.Parse(c.Query("server_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	var req models.UpdateChannelNotificationPreferenceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	pref, err := h.coordinator.UpdateChannelPreference(c.Context(), userID, channelID, serverID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to update channel notification preference",
		})
	}

	return c.JSON(pref)
}

// ServerNotificationPreferenceHandler handles per-server notification preferences
type ServerNotificationPreferenceHandler struct {
	coordinator NotificationCoordinatorInterface
}

// NewServerNotificationPreferenceHandler creates a new server notification preference handler
func NewServerNotificationPreferenceHandler(coordinator NotificationCoordinatorInterface) *ServerNotificationPreferenceHandler {
	return &ServerNotificationPreferenceHandler{coordinator: coordinator}
}

// GetServerNotificationPreference returns notification preferences for a server
// @Summary Get server notification preferences
// @Tags Notification Preferences
// @Produce json
// @Success 200 {object} models.ServerNotificationPreference
// @Router /users/@me/servers/{serverId}/notifications [get]
func (h *ServerNotificationPreferenceHandler) GetServerNotificationPreference(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	serverID, err := uuid.Parse(c.Params("serverId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	pref, err := h.coordinator.GetServerPreference(c.Context(), userID, serverID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get server notification preference",
		})
	}

	return c.JSON(pref)
}

// UpdateServerNotificationPreference updates notification preferences for a server
// @Summary Update server notification preferences
// @Tags Notification Preferences
// @Accept json
// @Produce json
// @Success 200 {object} models.ServerNotificationPreference
// @Router /users/@me/servers/{serverId}/notifications [patch]
func (h *ServerNotificationPreferenceHandler) UpdateServerNotificationPreference(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	serverID, err := uuid.Parse(c.Params("serverId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	var req models.UpdateServerNotificationPreferenceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	pref, err := h.coordinator.UpdateServerPreference(c.Context(), userID, serverID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to update server notification preference",
		})
	}

	return c.JSON(pref)
}
