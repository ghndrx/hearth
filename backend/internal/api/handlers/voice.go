package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// LiveKitVoiceHandler handles LiveKit voice channel operations
type LiveKitVoiceHandler struct {
	voiceService   *services.VoiceService
	userService    *services.UserService
	channelService *services.ChannelService
	permService    *services.PermissionService
}

// NewLiveKitVoiceHandler creates a new LiveKit voice handler
func NewLiveKitVoiceHandler(
	voiceService *services.VoiceService,
	userService *services.UserService,
	channelService *services.ChannelService,
	permService *services.PermissionService,
) *LiveKitVoiceHandler {
	return &LiveKitVoiceHandler{
		voiceService:   voiceService,
		userService:    userService,
		channelService: channelService,
		permService:    permService,
	}
}

// GenerateTokenRequest is the request body for generating a voice token
type GenerateTokenRequest struct {
	ChannelID string `json:"channel_id"`
}

// GenerateTokenResponse is the response for generating a voice token
type GenerateTokenResponse struct {
	Token string `json:"token"`
	URL   string `json:"url"`
}

// GenerateToken generates a LiveKit access token for the requesting user
// @Summary Generate voice token
// @Description Generates a LiveKit access token for joining a voice channel
// @Tags Voice
// @Accept json
// @Produce json
// @Param body body GenerateTokenRequest true "Voice channel ID"
// @Success 200 {object} GenerateTokenResponse "Token generated successfully"
// @Failure 400 {object} fiber.Map "Invalid request body or channel ID format"
// @Failure 403 {object} fiber.Map "Not a member of the server"
// @Failure 404 {object} fiber.Map "Channel not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Failure 503 {object} fiber.Map "Voice service not configured"
// @Router /voice/token [post]
func (h *LiveKitVoiceHandler) GenerateToken(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	var req GenerateTokenRequest
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

	// Get user info for metadata
	user, err := h.userService.GetUser(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get user info",
		})
	}

	// Determine display name
	displayName := user.Username
	if user.DisplayName != nil && *user.DisplayName != "" {
		displayName = *user.DisplayName
	}

	// Get avatar URL
	avatarURL := ""
	if user.AvatarURL != nil {
		avatarURL = *user.AvatarURL
	}

	// Generate the token
	tokenResp, err := h.voiceService.GenerateToken(
		c.Context(),
		userID,
		channelID,
		user.Username,
		displayName,
		avatarURL,
	)
	if err != nil {
		switch err {
		case services.ErrLiveKitNotConfigured:
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error": "voice service is not configured",
			})
		case services.ErrVoiceChannelNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "channel not found",
			})
		case services.ErrNotVoiceChannel:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "channel is not a voice channel",
			})
		case services.ErrNotServerMember:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "you are not a member of this server",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
	}

	return c.JSON(GenerateTokenResponse{
		Token: tokenResp.Token,
		URL:   tokenResp.URL,
	})
}

// ParticipantsResponse is the response for listing participants
type ParticipantsResponse struct {
	Participants []services.Participant `json:"participants"`
}

// GetParticipants returns the list of participants in a voice channel
// @Summary Get voice participants
// @Description Returns the list of participants currently in a voice channel
// @Tags Voice
// @Produce json
// @Param channelId path string true "Channel ID"
// @Success 200 {object} ParticipantsResponse "List of participants"
// @Failure 400 {object} fiber.Map "Invalid channel ID"
// @Failure 403 {object} fiber.Map "Not a member of the server"
// @Failure 404 {object} fiber.Map "Channel not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Failure 503 {object} fiber.Map "Voice service not configured"
// @Router /voice/participants/{channelId} [get]
func (h *LiveKitVoiceHandler) GetParticipants(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	channelID, err := uuid.Parse(c.Params("channelId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	// Verify user has access to the channel
	channel, err := h.channelService.GetChannel(c.Context(), channelID)
	if err != nil {
		if err == services.ErrChannelNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "channel not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// For server channels, verify membership
	if channel.ServerID != nil {
		_, err := h.channelService.GetServerChannels(c.Context(), *channel.ServerID, userID)
		if err != nil {
			if err == services.ErrNotServerMember {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "you are not a member of this server",
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
	}

	participants, err := h.voiceService.GetRoomParticipants(c.Context(), channelID)
	if err != nil {
		if err == services.ErrLiveKitNotConfigured {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error": "voice service is not configured",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(ParticipantsResponse{
		Participants: participants,
	})
}

// DisconnectParticipant removes a participant from a voice channel (moderation)
// @Summary Disconnect voice participant
// @Description Removes a participant from a voice channel (requires MOVE_MEMBERS permission)
// @Tags Voice
// @Param channelId path string true "Channel ID"
// @Param userId path string true "User ID to disconnect"
// @Success 204 "Participant disconnected successfully"
// @Failure 400 {object} fiber.Map "Invalid channel or user ID"
// @Failure 403 {object} fiber.Map "Missing MOVE_MEMBERS permission"
// @Failure 404 {object} fiber.Map "Channel not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Failure 503 {object} fiber.Map "Voice service not configured"
// @Router /voice/participants/{channelId}/{userId} [delete]
func (h *LiveKitVoiceHandler) DisconnectParticipant(c *fiber.Ctx) error {
	requesterID := c.Locals("userID").(uuid.UUID)

	channelID, err := uuid.Parse(c.Params("channelId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	targetUserID, err := uuid.Parse(c.Params("userId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid user id",
		})
	}

	// Check MOVE_MEMBERS permission
	channel, err := h.channelService.GetChannel(c.Context(), channelID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "channel not found",
		})
	}
	if channel.ServerID != nil && h.permService != nil {
		if err := h.permService.RequirePermission(c.Context(), *channel.ServerID, requesterID, models.PermMoveMembers); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "missing MOVE_MEMBERS permission",
			})
		}
	}

	err = h.voiceService.DisconnectParticipant(c.Context(), channelID, targetUserID)
	if err != nil {
		if err == services.ErrLiveKitNotConfigured {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error": "voice service is not configured",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// MuteParticipantRequest is the request body for muting a participant
type MuteParticipantRequest struct {
	Muted bool `json:"muted"`
}

// MuteParticipant mutes or unmutes a participant in a voice channel (moderation)
// @Summary Mute/unmute voice participant
// @Description Mutes or unmutes a participant in a voice channel (requires MUTE_MEMBERS permission)
// @Tags Voice
// @Accept json
// @Param channelId path string true "Channel ID"
// @Param userId path string true "User ID to mute/unmute"
// @Param body body MuteParticipantRequest true "Mute state"
// @Success 204 "Mute state changed successfully"
// @Failure 400 {object} fiber.Map "Invalid channel/user ID or request body"
// @Failure 403 {object} fiber.Map "Missing MUTE_MEMBERS permission"
// @Failure 404 {object} fiber.Map "Channel not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Failure 503 {object} fiber.Map "Voice service not configured"
// @Router /voice/participants/{channelId}/{userId}/mute [post]
func (h *LiveKitVoiceHandler) MuteParticipant(c *fiber.Ctx) error {
	requesterID := c.Locals("userID").(uuid.UUID)

	channelID, err := uuid.Parse(c.Params("channelId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	targetUserID, err := uuid.Parse(c.Params("userId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid user id",
		})
	}

	var req MuteParticipantRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Check MUTE_MEMBERS permission
	channel, err := h.channelService.GetChannel(c.Context(), channelID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "channel not found",
		})
	}
	if channel.ServerID != nil && h.permService != nil {
		if err := h.permService.RequirePermission(c.Context(), *channel.ServerID, requesterID, models.PermMuteMembers); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "missing MUTE_MEMBERS permission",
			})
		}
	}

	err = h.voiceService.MuteParticipant(c.Context(), channelID, targetUserID, req.Muted)
	if err != nil {
		if err == services.ErrLiveKitNotConfigured {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error": "voice service is not configured",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
