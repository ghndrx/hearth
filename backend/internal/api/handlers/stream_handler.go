package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// StreamHandler handles live streaming API requests
type StreamHandler struct {
	streamService  *services.LiveStreamService
	channelService *services.ChannelService
	permService    *services.PermissionService
}

// NewStreamHandler creates a new stream handler
func NewStreamHandler(
	streamService *services.LiveStreamService,
	channelService *services.ChannelService,
	permService *services.PermissionService,
) *StreamHandler {
	return &StreamHandler{
		streamService:  streamService,
		channelService: channelService,
		permService:    permService,
	}
}

// StartStream starts a live stream in a channel
// POST /api/v1/channels/:channelId/stream/start
func (h *StreamHandler) StartStream(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	channelID, err := uuid.Parse(c.Params("channelId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	var req models.StartLiveStreamRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Validate stream type
	if req.StreamType != models.LiveStreamTypeScreen &&
		req.StreamType != models.LiveStreamTypeApplication &&
		req.StreamType != models.LiveStreamTypeCamera {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "stream_type must be 1 (screen), 2 (application), or 3 (camera)",
		})
	}

	// Validate quality if provided
	if req.Quality != 0 &&
		req.Quality != models.LiveStreamQuality480p &&
		req.Quality != models.LiveStreamQuality720p &&
		req.Quality != models.LiveStreamQuality1080p {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "quality must be 1 (480p), 2 (720p), or 3 (1080p)",
		})
	}

	info, err := h.streamService.StartStream(
		c.Context(),
		channelID,
		userID,
		req.StreamType,
		req.Quality,
	)
	if err != nil {
		switch err {
		case services.ErrChannelNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "channel not found",
			})
		case services.ErrChannelNotVoice:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "channel is not a voice channel",
			})
		case services.ErrNotServerMember:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "you are not a member of this server",
			})
		case services.ErrLiveAlreadyStreaming:
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "already streaming in this channel",
			})
		case services.ErrMissingPermission:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "missing stream permission",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to start stream",
			})
		}
	}

	return c.Status(fiber.StatusCreated).JSON(info)
}

// StopStream stops an active live stream
// POST /api/v1/channels/:channelId/stream/stop
func (h *StreamHandler) StopStream(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	channelID, err := uuid.Parse(c.Params("channelId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	// First get the active stream for this channel
	info, err := h.streamService.GetActiveStreamForChannel(c.Context(), channelID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get stream",
		})
	}
	if info == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "no active stream in this channel",
		})
	}

	err = h.streamService.StopStream(c.Context(), info.ID, userID)
	if err != nil {
		switch err {
		case services.ErrLiveStreamNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "stream not found",
			})
		case services.ErrLiveNotStreamer:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "you are not the streamer",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to stop stream",
			})
		}
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// GetActiveStream returns the active stream for a channel
// GET /api/v1/channels/:channelId/stream
func (h *StreamHandler) GetActiveStream(c *fiber.Ctx) error {
	channelID, err := uuid.Parse(c.Params("channelId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	info, err := h.streamService.GetActiveStreamForChannel(c.Context(), channelID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get stream",
		})
	}

	if info == nil {
		return c.SendStatus(fiber.StatusNoContent)
	}

	return c.JSON(info)
}

// GetStream returns information about a specific stream
// GET /api/v1/streams/:streamId
func (h *StreamHandler) GetStream(c *fiber.Ctx) error {
	streamID, err := uuid.Parse(c.Params("streamId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid stream id",
		})
	}

	info, err := h.streamService.GetStream(c.Context(), streamID)
	if err != nil {
		if err == services.ErrLiveStreamNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "stream not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get stream",
		})
	}

	return c.JSON(info)
}

// JoinStream allows a user to join viewing a stream
// POST /api/v1/streams/:streamId/join
func (h *StreamHandler) JoinStream(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	streamID, err := uuid.Parse(c.Params("streamId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid stream id",
		})
	}

	err = h.streamService.JoinStreamAsViewer(c.Context(), streamID, userID)
	if err != nil {
		switch err {
		case services.ErrLiveStreamNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "stream not found",
			})
		case services.ErrLiveStreamEnded:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "stream has ended",
			})
		case services.ErrLiveCannotJoinOwnStream:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "cannot join your own stream",
			})
		case services.ErrLiveAlreadyViewing:
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "already viewing this stream",
			})
		case services.ErrNotServerMember:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "you are not a member of this server",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to join stream",
			})
		}
	}

	return c.JSON(fiber.Map{"message": "joined stream successfully"})
}

// LeaveStream allows a user to stop viewing a stream
// POST /api/v1/streams/:streamId/leave
func (h *StreamHandler) LeaveStream(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	streamID, err := uuid.Parse(c.Params("streamId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid stream id",
		})
	}

	err = h.streamService.LeaveStream(c.Context(), streamID, userID)
	if err != nil {
		switch err {
		case services.ErrLiveStreamNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "stream not found",
			})
		case services.ErrNotViewing:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "not viewing this stream",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to leave stream",
			})
		}
	}

	return c.JSON(fiber.Map{"message": "left stream successfully"})
}

// UpdateStream updates stream settings
// PATCH /api/v1/streams/:streamId
func (h *StreamHandler) UpdateStream(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	streamID, err := uuid.Parse(c.Params("streamId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid stream id",
		})
	}

	var req models.LiveStreamSettingsUpdate
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Validate quality if provided
	if req.Quality != nil &&
		*req.Quality != models.LiveStreamQuality480p &&
		*req.Quality != models.LiveStreamQuality720p &&
		*req.Quality != models.LiveStreamQuality1080p {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "quality must be 1 (480p), 2 (720p), or 3 (1080p)",
		})
	}

	info, err := h.streamService.UpdateStream(c.Context(), streamID, userID, &req)
	if err != nil {
		switch err {
		case services.ErrLiveStreamNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "stream not found",
			})
		case services.ErrLiveNotStreamer:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "you are not the streamer",
			})
		case services.ErrLiveStreamEnded:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "stream has ended",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to update stream",
			})
		}
	}

	return c.JSON(info)
}

// GetStreamViewers returns all viewers of a stream
// GET /api/v1/streams/:streamId/viewers
func (h *StreamHandler) GetStreamViewers(c *fiber.Ctx) error {
	streamID, err := uuid.Parse(c.Params("streamId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid stream id",
		})
	}

	viewers, err := h.streamService.GetStreamViewers(c.Context(), streamID)
	if err != nil {
		if err == services.ErrLiveStreamNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "stream not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get viewers",
		})
	}

	return c.JSON(fiber.Map{"viewers": viewers})
}
