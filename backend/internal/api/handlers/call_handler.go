package handlers

import (
	"context"
	"encoding/json"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
	"hearth/internal/websocket"
)

// CallHandler handles video/audio call HTTP endpoints
type CallHandler struct {
	callService    *services.CallService
	channelService CallChannelServiceInterface
	videoService   VideoSignalingServiceInterface
}

// VideoSignalingServiceInterface defines the video signaling methods needed by CallHandler
type VideoSignalingServiceInterface interface {
	SignalVideo(ctx context.Context, senderID uuid.UUID, signalType string, data json.RawMessage) error
}

// CallChannelServiceInterface defines methods needed for channel access in call handler
type CallChannelServiceInterface interface {
	GetOrCreateDM(ctx context.Context, user1ID, user2ID uuid.UUID) (*models.Channel, error)
}

// NewCallHandler creates a new call handler
func NewCallHandler(callService *services.CallService, channelService CallChannelServiceInterface, videoService *websocket.VideoSignalingService) *CallHandler {
	return &CallHandler{
		callService:    callService,
		channelService: channelService,
		videoService:   videoService,
	}
}

// NewCallHandlerWithInterface creates a new call handler with a custom signaling service interface
func NewCallHandlerWithInterface(callService *services.CallService, channelService CallChannelServiceInterface, videoService VideoSignalingServiceInterface) *CallHandler {
	return &CallHandler{
		callService:    callService,
		channelService: channelService,
		videoService:   videoService,
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

// SignalRequest represents the body of a signaling request
type SignalRequest struct {
	Type string          `json:"type"`
	Data  json.RawMessage `json:"data"`
}

// Signal relays signaling data for WebRTC negotiation via HTTP
// POST /api/v1/calls/:id/signal
// This endpoint exists as a fallback for environments where WebSocket is unavailable.
// Primary signaling is handled via WebSocket in websocket/video.go.
func (h *CallHandler) Signal(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	callID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid call ID",
		})
	}

	// Verify the call exists
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

	// Verify user is a participant in the call
	isParticipant := false
	for _, participant := range call.Participants {
		if participant.UserID == userID {
			isParticipant = true
			break
		}
	}
	if !isParticipant {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "you are not a participant in this call",
		})
	}

	// Parse signaling request
	var req SignalRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.Type == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "signal type is required",
		})
	}

	if req.Data == nil || len(req.Data) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "signal data is required",
		})
	}

	// Validate signal type
	validTypes := map[string]bool{
		"VIDEO_OFFER":         true,
		"VIDEO_ANSWER":        true,
		"VIDEO_ICE_CANDIDATE": true,
	}
	if !validTypes[req.Type] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid signal type, must be VIDEO_OFFER, VIDEO_ANSWER, or VIDEO_ICE_CANDIDATE",
		})
	}

	// If videoService is not available, return an error with guidance
	if h.videoService == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "video signaling service not available",
			"hint":  "use WebSocket signaling via the gateway connection",
		})
	}

	// Relay signaling data via the video signaling service
	if err := h.videoService.SignalVideo(c.Context(), userID, req.Type, req.Data); err != nil {
		log.Printf("[CallHandler] Failed to relay signaling: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to relay signaling data",
		})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}
