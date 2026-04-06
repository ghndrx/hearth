package handlers

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// CallHandler handles video/audio call HTTP endpoints
type CallHandler struct {
	callService    *services.CallService
	channelService CallChannelServiceInterface
}

// CallChannelServiceInterface defines methods needed for channel access in call handler
type CallChannelServiceInterface interface {
	GetOrCreateDM(ctx context.Context, user1ID, user2ID uuid.UUID) (*models.Channel, error)
}

// NewCallHandler creates a new call handler
func NewCallHandler(callService *services.CallService, channelService CallChannelServiceInterface) *CallHandler {
	return &CallHandler{
		callService:    callService,
		channelService: channelService,
	}
}

// Create creates a new call
// POST /api/v1/calls
func (h *CallHandler) Create(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	var req models.CreateCallRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	var channelID uuid.UUID
	var serverID *uuid.UUID

	// Handle direct calls via target_user_id - get or create DM channel
	if req.TargetUserID != "" {
		targetUserID, err := uuid.Parse(req.TargetUserID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid target_user_id format",
			})
		}

		if targetUserID == userID {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "cannot start a call with yourself",
			})
		}

		// Get or create DM channel for direct call
		channel, err := h.channelService.GetOrCreateDM(c.Context(), userID, targetUserID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to get or create DM channel for call",
			})
		}
		channelID = channel.ID
		// Direct calls don't have a server
		serverID = nil
	} else if req.ChannelID != "" {
		// Standard call via channel_id (server channel)
		var err error
		channelID, err = uuid.Parse(req.ChannelID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid channel_id format",
			})
		}

		if req.ServerID != "" {
			parsed, err := uuid.Parse(req.ServerID)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "invalid server_id format",
				})
			}
			serverID = &parsed
		}
	} else {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "either channel_id or target_user_id is required",
		})
	}

	callType := models.CallType(req.Type)
	if callType == "" {
		callType = models.CallTypeDirect
	}

	call, err := h.callService.CreateCall(c.Context(), userID, channelID, serverID, callType)
	if err != nil {
		if err == services.ErrInvalidCallType {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create call",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(call)
}

// Get retrieves call details
// GET /api/v1/calls/:id
func (h *CallHandler) Get(c *fiber.Ctx) error {
	callID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid call ID",
		})
	}

	call, err := h.callService.GetCall(c.Context(), callID)
	if err != nil {
		if err == services.ErrCallNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "call not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get call",
		})
	}

	return c.JSON(call)
}

// Join adds the user to an existing call
// POST /api/v1/calls/:id/join
func (h *CallHandler) Join(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	callID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid call ID",
		})
	}

	resp, err := h.callService.JoinCall(c.Context(), callID, userID)
	if err != nil {
		switch err {
		case services.ErrCallNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "call not found",
			})
		case services.ErrCallAlreadyEnded:
			return c.Status(fiber.StatusGone).JSON(fiber.Map{
				"error": "call has already ended",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to join call",
			})
		}
	}

	return c.JSON(resp)
}

// Leave removes the user from a call
// POST /api/v1/calls/:id/leave
func (h *CallHandler) Leave(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	callID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid call ID",
		})
	}

	if err := h.callService.LeaveCall(c.Context(), callID, userID); err != nil {
		if err == services.ErrCallNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "call not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to leave call",
		})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// Signal relays signaling data for WebRTC negotiation
// POST /api/v1/calls/:id/signal
// TODO: This is a stub - actual signaling happens via WebSocket in websocket/video.go.
// This endpoint exists as a fallback for environments where WebSocket is unavailable.
func (h *CallHandler) Signal(c *fiber.Ctx) error {
	callID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid call ID",
		})
	}

	// Verify the call exists
	_, err = h.callService.GetCall(c.Context(), callID)
	if err != nil {
		if err == services.ErrCallNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "call not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get call",
		})
	}

	// TODO: Implement HTTP-based signaling fallback
	// Primary signaling is handled via WebSocket in websocket/video.go
	// This endpoint would parse the signal type (offer/answer/ice) and
	// relay it through the VideoSignalingService
	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
		"error": "use WebSocket signaling via the gateway connection",
		"hint":  "connect to /gateway and use VIDEO_OFFER, VIDEO_ANSWER, VIDEO_ICE_CANDIDATE message types",
	})
}
