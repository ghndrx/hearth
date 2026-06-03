package handlers

import (
	"context"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

type NotificationCoordinatorInterface interface {
	GetChannelPreference(ctx context.Context, userID, channelID, serverID uuid.UUID) (*models.ChannelNotificationPreference, error)
	UpdateChannelPreference(ctx context.Context, userID, channelID, serverID uuid.UUID, req *models.UpdateChannelNotificationPreferenceRequest) (*models.ChannelNotificationPreference, error)
	GetServerPreference(ctx context.Context, userID, serverID uuid.UUID) (*models.ServerNotificationPreference, error)
	UpdateServerPreference(ctx context.Context, userID, serverID uuid.UUID, req *models.UpdateServerNotificationPreferenceRequest) (*models.ServerNotificationPreference, error)
}

type DigestServiceInterface interface {
	// Preferences
	GetPreferences(ctx context.Context, userID uuid.UUID) (*models.DigestPreferences, error)
	UpdatePreferences(ctx context.Context, userID uuid.UUID, req *models.UpdateDigestPreferencesRequest) (*models.DigestPreferences, error)

	// Channel preferences
	GetChannelPreference(ctx context.Context, userID, channelID uuid.UUID) (*models.DigestChannelPreference, error)
	GetChannelPreferences(ctx context.Context, userID uuid.UUID) ([]models.DigestChannelPreference, error)
	UpdateChannelPreference(ctx context.Context, userID, channelID uuid.UUID, mode models.DigestMode) error

	// Server preferences
	GetServerPreference(ctx context.Context, userID, serverID uuid.UUID) (*models.DigestServerPreference, error)
	GetServerPreferences(ctx context.Context, userID uuid.UUID) ([]models.DigestServerPreference, error)
	UpdateServerPreference(ctx context.Context, userID, serverID uuid.UUID, mode models.DigestMode) error

	// Queue & Preview
	GetDigestPreview(ctx context.Context, userID uuid.UUID) (*models.DigestPreview, error)
	ClearDigestQueue(ctx context.Context, userID uuid.UUID) (int64, error)

	// History
	GetDigestHistory(ctx context.Context, userID uuid.UUID, opts models.DigestHistoryListOptions) ([]models.DigestHistory, error)
	GetDigestByID(ctx context.Context, userID uuid.UUID, digestID uuid.UUID) (*models.DigestHistory, error)

	// Generation
	GenerateDigest(ctx context.Context, userID uuid.UUID) (*models.DigestHistory, error)
}

// NotificationPreferenceHandler handles all notification preference HTTP requests
type NotificationPreferenceHandler struct {
	coordinator   NotificationCoordinatorInterface
	digestService DigestServiceInterface
}

// NewNotificationPreferenceHandler creates a new notification preference handler
func NewNotificationPreferenceHandler(coordinator NotificationCoordinatorInterface, digestService DigestServiceInterface) *NotificationPreferenceHandler {
	return &NotificationPreferenceHandler{
		coordinator:   coordinator,
		digestService: digestService,
	}
}

// GetChannelNotificationPreference returns notification preferences for a channel
// @Summary Get channel notification preferences
// @Tags Notification Preferences
// @Produce json
// @Success 200 {object} models.ChannelNotificationPreference
// @Router /users/@me/channels/{channelId}/notifications [get]
func (h *NotificationPreferenceHandler) GetChannelNotificationPreference(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

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
func (h *NotificationPreferenceHandler) UpdateChannelNotificationPreference(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

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


// GetServerNotificationPreference returns notification preferences for a server
// @Summary Get server notification preferences
// @Tags Notification Preferences
// @Produce json
// @Success 200 {object} models.ServerNotificationPreference
// @Router /users/@me/servers/{serverId}/notifications [get]
func (h *NotificationPreferenceHandler) GetServerNotificationPreference(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

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
func (h *NotificationPreferenceHandler) UpdateServerNotificationPreference(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

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

// GetPreferences returns the current user's digest preferences
func (h *NotificationPreferenceHandler) GetPreferences(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	prefs, err := h.digestService.GetPreferences(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get digest preferences",
		})
	}

	return c.JSON(prefs)
}

// UpdatePreferences updates the current user's digest preferences
func (h *NotificationPreferenceHandler) UpdatePreferences(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	var req models.UpdateDigestPreferencesRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	prefs, err := h.digestService.UpdatePreferences(c.Context(), userID, &req)
	if err != nil {
		switch err {
		case services.ErrInvalidFrequency:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid frequency value",
			})
		case services.ErrInvalidTimezone:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid timezone",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to update digest preferences",
			})
		}
	}

	return c.JSON(prefs)
}

// GetChannelPreferences returns all channel-specific digest preferences
func (h *NotificationPreferenceHandler) GetChannelPreferences(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	prefs, err := h.digestService.GetChannelPreferences(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get channel digest preferences",
		})
	}

	return c.JSON(fiber.Map{
		"preferences": prefs,
	})
}

// GetChannelPreference returns a channel-specific digest preference
func (h *NotificationPreferenceHandler) GetChannelPreference(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	channelID, err := uuid.Parse(c.Params("channelId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	pref, err := h.digestService.GetChannelPreference(c.Context(), userID, channelID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get channel digest preference",
		})
	}

	return c.JSON(pref)
}

// UpdateChannelPreference updates a channel-specific digest preference
func (h *NotificationPreferenceHandler) UpdateChannelPreference(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	channelID, err := uuid.Parse(c.Params("channelId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	var req models.UpdateDigestChannelPreferenceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if err := h.digestService.UpdateChannelPreference(c.Context(), userID, channelID, req.DigestMode); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// GetServerPreferences returns all server-specific digest preferences
func (h *NotificationPreferenceHandler) GetServerPreferences(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	prefs, err := h.digestService.GetServerPreferences(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get server digest preferences",
		})
	}

	return c.JSON(fiber.Map{
		"preferences": prefs,
	})
}

// GetServerPreference returns a server-specific digest preference
func (h *NotificationPreferenceHandler) GetServerPreference(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	serverID, err := uuid.Parse(c.Params("serverId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	pref, err := h.digestService.GetServerPreference(c.Context(), userID, serverID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get server digest preference",
		})
	}

	return c.JSON(pref)
}

// UpdateServerPreference updates a server-specific digest preference
func (h *NotificationPreferenceHandler) UpdateServerPreference(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	serverID, err := uuid.Parse(c.Params("serverId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	var req models.UpdateDigestServerPreferenceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if err := h.digestService.UpdateServerPreference(c.Context(), userID, serverID, req.DigestMode); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// GetDigestPreview returns a preview of the pending digest
func (h *NotificationPreferenceHandler) GetDigestPreview(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	preview, err := h.digestService.GetDigestPreview(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get digest preview",
		})
	}

	return c.JSON(preview)
}

// ClearDigestQueue clears all pending digest items
func (h *NotificationPreferenceHandler) ClearDigestQueue(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	count, err := h.digestService.ClearDigestQueue(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to clear digest queue",
		})
	}

	return c.JSON(fiber.Map{
		"cleared": count,
	})
}

// GetDigestHistory returns the user's digest history
func (h *NotificationPreferenceHandler) GetDigestHistory(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	opts := models.DigestHistoryListOptions{
		Limit:  20,
		Offset: 0,
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 100 {
			opts.Limit = limit
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			opts.Offset = offset
		}
	}

	history, err := h.digestService.GetDigestHistory(c.Context(), userID, opts)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get digest history",
		})
	}

	return c.JSON(fiber.Map{
		"digests": history,
	})
}

// GetDigest returns a specific digest by ID
func (h *NotificationPreferenceHandler) GetDigest(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	digestID, err := uuid.Parse(c.Params("digestId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid digest id",
		})
	}

	digest, err := h.digestService.GetDigestByID(c.Context(), userID, digestID)
	if err != nil {
		if err == services.ErrDigestNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "digest not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get digest",
		})
	}

	return c.JSON(digest)
}

// GenerateDigestNow manually triggers digest generation
func (h *NotificationPreferenceHandler) GenerateDigestNow(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	digest, err := h.digestService.GenerateDigest(c.Context(), userID)
	if err != nil {
		if err == services.ErrDigestDisabled {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "digest notifications are disabled",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to generate digest",
		})
	}

	return c.JSON(digest)
}

