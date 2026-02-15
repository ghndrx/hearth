package handlers

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// VoiceStateServiceInterface defines the methods needed from VoiceStateService
type VoiceStateServiceInterface interface {
	Join(ctx context.Context, userID, channelID, serverID uuid.UUID) error
	Leave(ctx context.Context, userID uuid.UUID) error
	SetMuted(ctx context.Context, userID uuid.UUID, muted bool) error
	SetDeafened(ctx context.Context, userID uuid.UUID, deafened bool) error
	GetChannelUsers(ctx context.Context, channelID uuid.UUID) ([]*services.VoiceState, error)
	GetUserState(ctx context.Context, userID uuid.UUID) (*services.VoiceState, error)
}

// ChannelServiceForVoiceInterface defines channel methods needed for voice
type ChannelServiceForVoiceInterface interface {
	GetChannel(ctx context.Context, id uuid.UUID) (*models.Channel, error)
}

// ServerServiceForVoiceInterface defines server methods needed for voice
type ServerServiceForVoiceInterface interface {
	GetMember(ctx context.Context, serverID, userID uuid.UUID) (*models.Member, error)
}

// VoiceHandler handles voice-related endpoints
type VoiceHandler struct {
	voiceService   VoiceStateServiceInterface
	channelService ChannelServiceForVoiceInterface
	serverService  ServerServiceForVoiceInterface
}

// NewVoiceHandler creates a new voice handler
func NewVoiceHandler() *VoiceHandler {
	return &VoiceHandler{}
}

// NewVoiceHandlerWithService creates a voice handler with actual services
func NewVoiceHandlerWithService(
	voiceService VoiceStateServiceInterface,
	channelService ChannelServiceForVoiceInterface,
	serverService ServerServiceForVoiceInterface,
) *VoiceHandler {
	return &VoiceHandler{
		voiceService:   voiceService,
		channelService: channelService,
		serverService:  serverService,
	}
}

// VoiceRegion represents a voice region
type VoiceRegion struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Optimal bool   `json:"optimal"`
}

// GetRegions returns available voice regions
func (h *VoiceHandler) GetRegions(c *fiber.Ctx) error {
	return c.JSON([]VoiceRegion{
		{ID: "us-west", Name: "US West", Optimal: true},
		{ID: "us-east", Name: "US East", Optimal: false},
		{ID: "eu-west", Name: "EU West", Optimal: false},
		{ID: "eu-central", Name: "EU Central", Optimal: false},
		{ID: "singapore", Name: "Singapore", Optimal: false},
		{ID: "sydney", Name: "Sydney", Optimal: false},
	})
}

// VoiceStateResponse represents a voice state in API responses
type VoiceStateResponse struct {
	UserID    uuid.UUID `json:"user_id"`
	ChannelID uuid.UUID `json:"channel_id"`
	ServerID  uuid.UUID `json:"server_id"`
	Muted     bool      `json:"self_mute"`
	Deafened  bool      `json:"self_deaf"`
	Streaming bool      `json:"self_stream"`
}

// JoinVoiceRequest represents the request to join a voice channel
type JoinVoiceRequest struct {
	ChannelID uuid.UUID `json:"channel_id"`
}

// JoinVoice joins the user to a voice channel
func (h *VoiceHandler) JoinVoice(c *fiber.Ctx) error {
	if h.voiceService == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "voice service not available",
		})
	}

	userID := c.Locals("userID").(uuid.UUID)

	var req JoinVoiceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.ChannelID == uuid.Nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "channel_id is required",
		})
	}

	// Get the channel to verify it exists and is a voice channel
	channel, err := h.channelService.GetChannel(c.Context(), req.ChannelID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "channel not found",
		})
	}

	if channel.Type != models.ChannelTypeVoice && channel.Type != models.ChannelTypeStage {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "channel is not a voice channel",
		})
	}

	// Check if user is a member of the server
	if channel.ServerID != nil {
		member, err := h.serverService.GetMember(c.Context(), *channel.ServerID, userID)
		if err != nil || member == nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "you are not a member of this server",
			})
		}
	}

	serverID := uuid.Nil
	if channel.ServerID != nil {
		serverID = *channel.ServerID
	}

	if err := h.voiceService.Join(c.Context(), userID, req.ChannelID, serverID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to join voice channel",
		})
	}

	state, err := h.voiceService.GetUserState(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get voice state",
		})
	}

	return c.JSON(VoiceStateResponse{
		UserID:    state.UserID,
		ChannelID: state.ChannelID,
		ServerID:  state.ServerID,
		Muted:     state.Muted,
		Deafened:  state.Deafened,
		Streaming: state.Streaming,
	})
}

// LeaveVoice removes the user from their current voice channel
func (h *VoiceHandler) LeaveVoice(c *fiber.Ctx) error {
	if h.voiceService == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "voice service not available",
		})
	}

	userID := c.Locals("userID").(uuid.UUID)

	if err := h.voiceService.Leave(c.Context(), userID); err != nil {
		// Even if user wasn't in voice, treat as success
		if err == services.ErrUserNotInVoice {
			return c.SendStatus(fiber.StatusNoContent)
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to leave voice channel",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// UpdateVoiceStateRequest represents the request to update voice state
type UpdateVoiceStateRequest struct {
	Muted    *bool `json:"self_mute"`
	Deafened *bool `json:"self_deaf"`
}

// UpdateVoiceState updates the user's voice state (mute/deafen)
func (h *VoiceHandler) UpdateVoiceState(c *fiber.Ctx) error {
	if h.voiceService == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "voice service not available",
		})
	}

	userID := c.Locals("userID").(uuid.UUID)

	var req UpdateVoiceStateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.Muted != nil {
		if err := h.voiceService.SetMuted(c.Context(), userID, *req.Muted); err != nil {
			if err == services.ErrUserNotInVoice {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "you are not in a voice channel",
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to update mute state",
			})
		}
	}

	if req.Deafened != nil {
		if err := h.voiceService.SetDeafened(c.Context(), userID, *req.Deafened); err != nil {
			if err == services.ErrUserNotInVoice {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "you are not in a voice channel",
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to update deafen state",
			})
		}
	}

	state, err := h.voiceService.GetUserState(c.Context(), userID)
	if err != nil {
		if err == services.ErrUserNotInVoice {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "you are not in a voice channel",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get voice state",
		})
	}

	return c.JSON(VoiceStateResponse{
		UserID:    state.UserID,
		ChannelID: state.ChannelID,
		ServerID:  state.ServerID,
		Muted:     state.Muted,
		Deafened:  state.Deafened,
		Streaming: state.Streaming,
	})
}

// GetMyVoiceState returns the current user's voice state
func (h *VoiceHandler) GetMyVoiceState(c *fiber.Ctx) error {
	if h.voiceService == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "voice service not available",
		})
	}

	userID := c.Locals("userID").(uuid.UUID)

	state, err := h.voiceService.GetUserState(c.Context(), userID)
	if err != nil {
		if err == services.ErrUserNotInVoice {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "not in a voice channel",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get voice state",
		})
	}

	return c.JSON(VoiceStateResponse{
		UserID:    state.UserID,
		ChannelID: state.ChannelID,
		ServerID:  state.ServerID,
		Muted:     state.Muted,
		Deafened:  state.Deafened,
		Streaming: state.Streaming,
	})
}

// GetChannelVoiceStates returns voice states for users in a channel
func (h *VoiceHandler) GetChannelVoiceStates(c *fiber.Ctx) error {
	if h.voiceService == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "voice service not available",
		})
	}

	channelIDParam := c.Params("id")
	channelID, err := uuid.Parse(channelIDParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	userID := c.Locals("userID").(uuid.UUID)

	// Verify channel exists and user has access
	channel, err := h.channelService.GetChannel(c.Context(), channelID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "channel not found",
		})
	}

	// Check if user is a member of the server
	if channel.ServerID != nil {
		member, err := h.serverService.GetMember(c.Context(), *channel.ServerID, userID)
		if err != nil || member == nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "you are not a member of this server",
			})
		}
	}

	states, err := h.voiceService.GetChannelUsers(c.Context(), channelID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get voice states",
		})
	}

	response := make([]VoiceStateResponse, len(states))
	for i, state := range states {
		response[i] = VoiceStateResponse{
			UserID:    state.UserID,
			ChannelID: state.ChannelID,
			ServerID:  state.ServerID,
			Muted:     state.Muted,
			Deafened:  state.Deafened,
			Streaming: state.Streaming,
		}
	}

	return c.JSON(response)
}
