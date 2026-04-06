package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// CallHandler handles video/audio call HTTP endpoints
type CallHandler struct {
	callService *services.CallService
}

// NewCallHandler creates a new call handler
func NewCallHandler(callService *services.CallService) *CallHandler {
	return &CallHandler{callService: callService}
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

	if req.ChannelID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "channel_id is required",
		})
	}

	channelID, err := uuid.Parse(req.ChannelID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel_id format",
		})
	}

	callType := models.CallType(req.Type)
	if callType == "" {
		callType = models.CallTypeDirect
	}

	var serverID *uuid.UUID
	if req.ServerID != "" {
		parsed, err := uuid.Parse(req.ServerID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid server_id format",
			})
		}
		serverID = &parsed
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
