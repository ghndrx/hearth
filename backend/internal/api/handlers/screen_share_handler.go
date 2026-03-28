package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// ScreenShareHandler handles screen share API requests
type ScreenShareHandler struct {
	screenShareService *services.ScreenShareService
	channelService     *services.ChannelService
	permService        *services.PermissionService
}

// NewScreenShareHandler creates a new screen share handler
func NewScreenShareHandler(
	screenShareService *services.ScreenShareService,
	channelService *services.ChannelService,
	permService *services.PermissionService,
) *ScreenShareHandler {
	return &ScreenShareHandler{
		screenShareService: screenShareService,
		channelService:     channelService,
		permService:        permService,
	}
}

// StartStream starts a screen share in a channel
// @Summary Start stream
// @Description Starts a screen share or application stream in a voice channel
// @Tags Streams
// @Accept json
// @Produce json
// @Param channelId path string true "Channel ID"
// @Param body body StartStreamRequest true "Stream settings"
// @Success 201 {object} models.StreamInfo "Stream started successfully"
// @Failure 400 {object} fiber.Map "Invalid channel ID or request body"
// @Failure 403 {object} fiber.Map "Not a member or missing permission"
// @Failure 404 {object} fiber.Map "Channel not found"
// @Failure 409 {object} fiber.Map "Already streaming"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{channelId}/streams [post]
func (h *ScreenShareHandler) StartStream(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	channelID, err := uuid.Parse(c.Params("channelId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	var req models.StartStreamRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Validate stream type
	if req.StreamType != models.StreamTypeScreen && req.StreamType != models.StreamTypeApplication {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "stream_type must be 1 (screen) or 2 (application)",
		})
	}

	// Validate resolution if provided
	if req.Resolution != "" && req.Resolution != "720p" && req.Resolution != "1080p" && req.Resolution != "1440p" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "resolution must be 720p, 1080p, or 1440p",
		})
	}

	// Validate frame rate if provided
	if req.FrameRate != 0 && req.FrameRate != 30 && req.FrameRate != 60 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "frame_rate must be 30 or 60",
		})
	}

	info, err := h.screenShareService.StartStream(
		c.Context(),
		channelID,
		userID,
		req.StreamType,
		req.Resolution,
		req.FrameRate,
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
		case services.ErrAlreadyStreaming:
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "already streaming in this channel",
			})
		default:
			// Check for permission errors
			if err == services.ErrMissingPermission {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "missing stream permission",
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to start stream",
			})
		}
	}

	return c.Status(fiber.StatusCreated).JSON(info)
}

// EndStream stops an active stream
// @Summary End stream
// @Description Stops an active screen share or application stream
// @Tags Streams
// @Param streamId path string true "Stream ID"
// @Success 204 "Stream ended successfully"
// @Failure 403 {object} fiber.Map "Not the streamer"
// @Failure 404 {object} fiber.Map "Stream not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /streams/{streamId} [delete]
func (h *ScreenShareHandler) EndStream(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	streamID, err := uuid.Parse(c.Params("streamId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid stream id",
		})
	}

	err = h.screenShareService.EndStream(c.Context(), streamID, userID)
	if err != nil {
		switch err {
		case services.ErrStreamNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "stream not found",
			})
		case services.ErrNotStreamer:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "you are not the streamer",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to end stream",
			})
		}
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// GetStreamInfo returns information about a stream
// @Summary Get stream info
// @Description Returns information about a stream session
// @Tags Streams
// @Produce json
// @Param streamId path string true "Stream ID"
// @Success 200 {object} models.StreamInfo "Stream information"
// @Failure 404 {object} fiber.Map "Stream not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /streams/{streamId} [get]
func (h *ScreenShareHandler) GetStreamInfo(c *fiber.Ctx) error {
	streamID, err := uuid.Parse(c.Params("streamId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid stream id",
		})
	}

	info, err := h.screenShareService.GetStreamInfo(c.Context(), streamID)
	if err != nil {
		if err == services.ErrStreamNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "stream not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get stream info",
		})
	}

	return c.JSON(info)
}

// JoinStream allows a user to join viewing a stream
// @Summary Join stream
// @Description Allows a user to join viewing a stream
// @Tags Streams
// @Param streamId path string true "Stream ID"
// @Success 200 {object} fiber.Map "Joined successfully"
// @Failure 400 {object} fiber.Map "Cannot join own stream"
// @Failure 403 {object} fiber.Map "Not a server member"
// @Failure 404 {object} fiber.Map "Stream not found"
// @Failure 409 {object} fiber.Map "Already viewing"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /streams/{streamId}/join [post]
func (h *ScreenShareHandler) JoinStream(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	streamID, err := uuid.Parse(c.Params("streamId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid stream id",
		})
	}

	err = h.screenShareService.JoinStream(c.Context(), streamID, userID)
	if err != nil {
		switch err {
		case services.ErrStreamNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "stream not found",
			})
		case services.ErrNoActiveStream:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "stream has ended",
			})
		case services.ErrCannotJoinOwnStream:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "cannot join your own stream",
			})
		case services.ErrAlreadyViewing:
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
// @Summary Leave stream
// @Description Allows a user to stop viewing a stream
// @Tags Streams
// @Param streamId path string true "Stream ID"
// @Success 200 {object} fiber.Map "Left successfully"
// @Failure 400 {object} fiber.Map "Not viewing stream"
// @Failure 404 {object} fiber.Map "Stream not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /streams/{streamId}/leave [delete]
func (h *ScreenShareHandler) LeaveStream(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	streamID, err := uuid.Parse(c.Params("streamId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid stream id",
		})
	}

	err = h.screenShareService.LeaveStream(c.Context(), streamID, userID)
	if err != nil {
		switch err {
		case services.ErrStreamNotFound:
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
// @Summary Update stream
// @Description Updates stream settings (resolution, frame rate)
// @Tags Streams
// @Accept json
// @Produce json
// @Param streamId path string true "Stream ID"
// @Param body body models.StreamUpdate true "Stream update data"
// @Success 200 {object} models.StreamInfo "Stream updated successfully"
// @Failure 403 {object} fiber.Map "Not the streamer"
// @Failure 404 {object} fiber.Map "Stream not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /streams/{streamId} [patch]
func (h *ScreenShareHandler) UpdateStream(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	streamID, err := uuid.Parse(c.Params("streamId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid stream id",
		})
	}

	var req models.StreamUpdate
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	info, err := h.screenShareService.UpdateStream(c.Context(), streamID, userID, &req)
	if err != nil {
		switch err {
		case services.ErrStreamNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "stream not found",
			})
		case services.ErrNotStreamer:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "you are not the streamer",
			})
		case services.ErrNoActiveStream:
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

// GetActiveStreamForChannel returns the active stream for a channel
// @Summary Get channel stream
// @Description Returns the active stream for a channel if any
// @Tags Streams
// @Produce json
// @Param channelId path string true "Channel ID"
// @Success 200 {object} models.StreamInfo "Active stream info"
// @Success 204 {object} nil "No active stream"
// @Failure 400 {object} fiber.Map "Invalid channel ID"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{channelId}/streams [get]
func (h *ScreenShareHandler) GetActiveStreamForChannel(c *fiber.Ctx) error {
	channelID, err := uuid.Parse(c.Params("channelId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	info, err := h.screenShareService.GetActiveStreamForChannel(c.Context(), channelID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get channel stream",
		})
	}

	if info == nil {
		return c.SendStatus(fiber.StatusNoContent)
	}

	return c.JSON(info)
}
