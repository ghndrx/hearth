package handlers

import (
	"context"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// DigestServiceInterface defines the methods needed from DigestService
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

// DigestHandler handles digest-related HTTP requests
type DigestHandler struct {
	digestService DigestServiceInterface
}

// NewDigestHandler creates a new digest handler
func NewDigestHandler(digestService DigestServiceInterface) *DigestHandler {
	return &DigestHandler{
		digestService: digestService,
	}
}

// GetPreferences returns the current user's digest preferences
func (h *DigestHandler) GetPreferences(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	prefs, err := h.digestService.GetPreferences(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get digest preferences",
		})
	}

	return c.JSON(prefs)
}

// UpdatePreferences updates the current user's digest preferences
func (h *DigestHandler) UpdatePreferences(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

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
func (h *DigestHandler) GetChannelPreferences(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

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
func (h *DigestHandler) GetChannelPreference(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

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
func (h *DigestHandler) UpdateChannelPreference(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

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
func (h *DigestHandler) GetServerPreferences(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

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
func (h *DigestHandler) GetServerPreference(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

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
func (h *DigestHandler) UpdateServerPreference(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

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
func (h *DigestHandler) GetDigestPreview(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	preview, err := h.digestService.GetDigestPreview(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get digest preview",
		})
	}

	return c.JSON(preview)
}

// ClearDigestQueue clears all pending digest items
func (h *DigestHandler) ClearDigestQueue(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

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
func (h *DigestHandler) GetDigestHistory(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

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
func (h *DigestHandler) GetDigest(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

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
func (h *DigestHandler) GenerateDigestNow(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

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
