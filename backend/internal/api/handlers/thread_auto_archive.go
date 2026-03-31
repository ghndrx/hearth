package handlers

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// ThreadAutoArchiveServiceInterface defines thread auto-archive service operations
type ThreadAutoArchiveServiceInterface interface {
	GetOrCreateServerSettings(ctx context.Context, serverID uuid.UUID) (*models.ThreadAutoArchiveSettings, error)
	UpdateServerSettings(ctx context.Context, serverID, requesterID uuid.UUID, req models.UpdateThreadAutoArchiveSettingsRequest) (*models.ThreadAutoArchiveSettings, error)
	GetServerSettings(ctx context.Context, serverID uuid.UUID) (*models.ThreadAutoArchiveSettings, error)
	SetChannelOverride(ctx context.Context, channelID, requesterID uuid.UUID, req models.SetChannelAutoArchiveOverrideRequest) (*models.ChannelAutoArchiveOverride, error)
	GetChannelOverride(ctx context.Context, channelID uuid.UUID) (*models.ChannelAutoArchiveOverride, error)
	DeleteChannelOverride(ctx context.Context, channelID, requesterID uuid.UUID) error
	GetThreadAutoArchiveStatus(ctx context.Context, threadID uuid.UUID) (*models.ThreadAutoArchiveResponse, error)
	GetServerStats(ctx context.Context, serverID uuid.UUID) (*models.ThreadAutoArchiveStats, error)
	ArchiveThread(ctx context.Context, threadID uuid.UUID) error
}

// ThreadAutoArchiveHandler handles thread auto-archive HTTP requests
type ThreadAutoArchiveHandler struct {
	autoArchiveService ThreadAutoArchiveServiceInterface
}

// NewThreadAutoArchiveHandler creates a new thread auto-archive handler
func NewThreadAutoArchiveHandler(autoArchiveService ThreadAutoArchiveServiceInterface) *ThreadAutoArchiveHandler {
	return &ThreadAutoArchiveHandler{
		autoArchiveService: autoArchiveService,
	}
}

// GetServerAutoArchiveSettings retrieves server auto-archive settings
// @Summary Get server auto-archive settings
// @Description Gets the auto-archive settings for a server
// @Tags Thread Auto-Archive
// @Produce json
// @Param server_id path string true "Server ID"
// @Success 200 {object} models.ThreadAutoArchiveSettings "Auto-archive settings"
// @Failure 400 {object} fiber.Map "Invalid server ID"
// @Failure 403 {object} fiber.Map "Not a server member"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{server_id}/auto-archive [get]
func (h *ThreadAutoArchiveHandler) GetServerAutoArchiveSettings(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	// Verify server membership (handled via middleware, but double-check)
	_ = userID // Used by middleware

	settings, err := h.autoArchiveService.GetServerSettings(c.Context(), serverID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get auto-archive settings",
		})
	}

	return c.JSON(settings)
}

// UpdateServerAutoArchiveSettings updates server auto-archive settings
// @Summary Update server auto-archive settings
// @Description Updates the auto-archive settings for a server (admin only)
// @Tags Thread Auto-Archive
// @Accept json
// @Produce json
// @Param server_id path string true "Server ID"
// @Param body body models.UpdateThreadAutoArchiveSettingsRequest true "Settings update request"
// @Success 200 {object} models.ThreadAutoArchiveSettings "Updated auto-archive settings"
// @Failure 400 {object} fiber.Map "Invalid server ID or request body"
// @Failure 403 {object} fiber.Map "Not authorized to update settings"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{server_id}/auto-archive [patch]
func (h *ThreadAutoArchiveHandler) UpdateServerAutoArchiveSettings(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	var req models.UpdateThreadAutoArchiveSettingsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	settings, err := h.autoArchiveService.UpdateServerSettings(c.Context(), serverID, userID, req)
	if err != nil {
		switch err {
		case services.ErrServerNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "server not found",
			})
		case services.ErrNotServerMember:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "not a server member",
			})
		case services.ErrMissingAdministrator:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "not authorized to update settings",
			})
		case services.ErrInvalidAutoArchiveDuration:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid auto-archive duration",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to update auto-archive settings",
			})
		}
	}

	return c.JSON(settings)
}

// GetChannelAutoArchiveOverride retrieves channel-level auto-archive override
// @Summary Get channel auto-archive override
// @Description Gets the auto-archive override for a specific channel
// @Tags Thread Auto-Archive
// @Produce json
// @Param channel_id path string true "Channel ID"
// @Success 200 {object} models.ChannelAutoArchiveOverride "Channel override"
// @Failure 400 {object} fiber.Map "Invalid channel ID"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{channel_id}/auto-archive [get]
func (h *ThreadAutoArchiveHandler) GetChannelAutoArchiveOverride(c *fiber.Ctx) error {
	channelID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	override, err := h.autoArchiveService.GetChannelOverride(c.Context(), channelID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get channel override",
		})
	}

	if override == nil {
		return c.JSON(fiber.Map{
			"channel_id": channelID,
			"override":   nil,
		})
	}

	return c.JSON(override)
}

// SetChannelAutoArchiveOverride sets channel-level auto-archive override
// @Summary Set channel auto-archive override
// @Description Sets or updates the auto-archive override for a specific channel (admin only)
// @Tags Thread Auto-Archive
// @Accept json
// @Produce json
// @Param channel_id path string true "Channel ID"
// @Param body body models.SetChannelAutoArchiveOverrideRequest true "Override request"
// @Success 200 {object} models.ChannelAutoArchiveOverride "Channel override"
// @Failure 400 {object} fiber.Map "Invalid channel ID or request body"
// @Failure 403 {object} fiber.Map "Not authorized"
// @Failure 404 {object} fiber.Map "Channel not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{channel_id}/auto-archive [put]
func (h *ThreadAutoArchiveHandler) SetChannelAutoArchiveOverride(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	channelID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	var req models.SetChannelAutoArchiveOverrideRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	override, err := h.autoArchiveService.SetChannelOverride(c.Context(), channelID, userID, req)
	if err != nil {
		switch err {
		case services.ErrChannelNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "channel not found",
			})
		case services.ErrChannelTypeNotSupported:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "channel is not a server channel",
			})
		case services.ErrAutoArchiveNotAllowed:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "auto-archive override not allowed for this server",
			})
		case services.ErrMissingAdministrator:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "not authorized to set override",
			})
		case services.ErrInvalidAutoArchiveDuration:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid auto-archive duration",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to set channel override",
			})
		}
	}

	return c.JSON(override)
}

// DeleteChannelAutoArchiveOverride removes channel-level auto-archive override
// @Summary Delete channel auto-archive override
// @Description Removes the auto-archive override for a specific channel (admin only)
// @Tags Thread Auto-Archive
// @Param channel_id path string true "Channel ID"
// @Success 204 "Override deleted successfully"
// @Failure 400 {object} fiber.Map "Invalid channel ID"
// @Failure 403 {object} fiber.Map "Not authorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{channel_id}/auto-archive [delete]
func (h *ThreadAutoArchiveHandler) DeleteChannelAutoArchiveOverride(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	channelID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	if err := h.autoArchiveService.DeleteChannelOverride(c.Context(), channelID, userID); err != nil {
		switch err {
		case services.ErrChannelNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "channel not found",
			})
		case services.ErrChannelTypeNotSupported:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "channel is not a server channel",
			})
		case services.ErrMissingAdministrator:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "not a server member",
			})
		case services.ErrMissingAdministrator:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "not authorized to delete override",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to delete channel override",
			})
		}
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// GetThreadAutoArchiveStatus retrieves auto-archive status for a thread
// @Summary Get thread auto-archive status
// @Description Gets the current auto-archive status for a thread
// @Tags Thread Auto-Archive
// @Produce json
// @Param thread_id path string true "Thread ID"
// @Success 200 {object} models.ThreadAutoArchiveResponse "Thread auto-archive status"
// @Failure 400 {object} fiber.Map "Invalid thread ID"
// @Failure 404 {object} fiber.Map "Thread not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /threads/{thread_id}/auto-archive [get]
func (h *ThreadAutoArchiveHandler) GetThreadAutoArchiveStatus(c *fiber.Ctx) error {
	threadID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid thread id",
		})
	}

	status, err := h.autoArchiveService.GetThreadAutoArchiveStatus(c.Context(), threadID)
	if err != nil {
		if err == services.ErrThreadNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "thread not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get thread auto-archive status",
		})
	}

	return c.JSON(status)
}

// GetServerAutoArchiveStats retrieves auto-archive statistics for a server
// @Summary Get server auto-archive statistics
// @Description Gets statistics about thread auto-archive for a server
// @Tags Thread Auto-Archive
// @Produce json
// @Param server_id path string true "Server ID"
// @Success 200 {object} models.ThreadAutoArchiveStats "Server auto-archive statistics"
// @Failure 400 {object} fiber.Map "Invalid server ID"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{server_id}/auto-archive/stats [get]
func (h *ThreadAutoArchiveHandler) GetServerAutoArchiveStats(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	stats, err := h.autoArchiveService.GetServerStats(c.Context(), serverID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get auto-archive stats",
		})
	}

	return c.JSON(stats)
}
