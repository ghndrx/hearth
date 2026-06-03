package handlers

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
	ws "hearth/internal/websocket"
)

// VoiceHandler handles all voice and RTC operations
type VoiceHandler struct {
	voiceService              *ws.VoiceSignalingService
	liveKitVoiceService       services.VoiceServiceInterface
	userService               services.UserServiceInterface
	channelService            services.ChannelServiceInterface
	liveKitPermService        services.VoicePermissionServiceInterface
	callService               *services.CallService
	callChannelService        CallChannelServiceInterface
	screenShareService        *services.ScreenShareService
	screenShareChannelService *services.ChannelService
	screenSharePermService    *services.PermissionService
	streamService             *services.LiveStreamService
	streamChannelService      *services.ChannelService
	streamPermService         *services.PermissionService
	stageService              *services.StageService
	stageChannelService       *services.ChannelService
	stagePermService          *services.PermissionService
	stageUserService          *services.UserService
	activityService           *services.VoiceActivityService
	activityChannelService    services.ChannelServiceInterface
	activityPermService       services.PermissionServiceInterface
}

// NewVoiceHandler creates a new voice handler
func NewVoiceHandler(voiceService *ws.VoiceSignalingService) *VoiceHandler {
	return &VoiceHandler{voiceService: voiceService}
}

// NewLiveKitVoiceHandler creates a new voice handler with LiveKit services configured
func NewLiveKitVoiceHandler(
	voiceService services.VoiceServiceInterface,
	userService services.UserServiceInterface,
	channelService services.ChannelServiceInterface,
	permService services.VoicePermissionServiceInterface,
) *VoiceHandler {
	h := &VoiceHandler{}
	h.SetLiveKitServices(voiceService, userService, channelService, permService)
	return h
}

// SetLiveKitServices sets the LiveKit voice services
func (h *VoiceHandler) SetLiveKitServices(
	voiceService services.VoiceServiceInterface,
	userService services.UserServiceInterface,
	channelService services.ChannelServiceInterface,
	permService services.VoicePermissionServiceInterface,
) {
	h.liveKitVoiceService = voiceService
	h.userService = userService
	h.channelService = channelService
	h.liveKitPermService = permService
}

// SetCallService sets the call service
func (h *VoiceHandler) SetCallService(callService *services.CallService, channelService CallChannelServiceInterface) {
	h.callService = callService
	h.callChannelService = channelService
}

// SetScreenShareService sets the screen share service
func (h *VoiceHandler) SetScreenShareService(
	screenShareService *services.ScreenShareService,
	channelService *services.ChannelService,
	permService *services.PermissionService,
) {
	h.screenShareService = screenShareService
	h.screenShareChannelService = channelService
	h.screenSharePermService = permService
}

// SetStreamService sets the live stream service
func (h *VoiceHandler) SetStreamService(
	streamService *services.LiveStreamService,
	channelService *services.ChannelService,
	permService *services.PermissionService,
) {
	h.streamService = streamService
	h.streamChannelService = channelService
	h.streamPermService = permService
}

// SetStageService sets the stage service
func (h *VoiceHandler) SetStageService(
	stageService *services.StageService,
	channelService *services.ChannelService,
	permService *services.PermissionService,
	userService *services.UserService,
) {
	h.stageService = stageService
	h.stageChannelService = channelService
	h.stagePermService = permService
	h.stageUserService = userService
}

// SetVoiceActivityService sets the voice activity service
func (h *VoiceHandler) SetVoiceActivityService(
	activityService *services.VoiceActivityService,
	channelService services.ChannelServiceInterface,
	permService services.PermissionServiceInterface,
) {
	h.activityService = activityService
	h.activityChannelService = channelService
	h.activityPermService = permService
}

// GetRegions returns available voice regions
func (h *VoiceHandler) GetRegions(c *fiber.Ctx) error {
	return c.JSON([]fiber.Map{
		{"id": "us-west", "name": "US West", "optimal": true},
		{"id": "us-east", "name": "US East", "optimal": false},
		{"id": "eu-west", "name": "EU West", "optimal": false},
		{"id": "eu-central", "name": "EU Central", "optimal": false},
		{"id": "singapore", "name": "Singapore", "optimal": false},
		{"id": "sydney", "name": "Sydney", "optimal": false},
	})
}

// GetChannelVoiceStates returns all users in a voice channel
func (h *VoiceHandler) GetChannelVoiceStates(c *fiber.Ctx) error {
	channelID, err := uuid.Parse(c.Params("channelId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	if h.voiceService == nil {
		return c.JSON([]fiber.Map{})
	}

	states, err := h.voiceService.GetChannelVoiceStates(c.Context(), channelID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(states)
}

// GetServerVoiceStates returns all users in voice channels in a server
func (h *VoiceHandler) GetServerVoiceStates(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	if h.voiceService == nil {
		return c.JSON([]fiber.Map{})
	}

	states, err := h.voiceService.GetServerVoiceStates(c.Context(), serverID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(states)
}

// GenerateToken generates a LiveKit token for voice channel access
func (h *VoiceHandler) GenerateToken(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

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

	user, err := h.userService.GetUser(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get user info",
		})
	}

	displayName := user.Username
	if user.DisplayName != nil && *user.DisplayName != "" {
		displayName = *user.DisplayName
	}

	avatarURL := ""
	if user.AvatarURL != nil {
		avatarURL = *user.AvatarURL
	}

	tokenResp, err := h.liveKitVoiceService.GenerateToken(
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

// GetParticipants returns the list of participants in a voice channel
func (h *VoiceHandler) GetParticipants(c *fiber.Ctx) error {
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

	participants, err := h.liveKitVoiceService.GetRoomParticipants(c.Context(), channelID)
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
func (h *VoiceHandler) DisconnectParticipant(c *fiber.Ctx) error {
	requesterID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

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

	channel, err := h.channelService.GetChannel(c.Context(), channelID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "channel not found",
		})
	}
	if channel.ServerID != nil && h.liveKitPermService != nil {
		if err := h.liveKitPermService.RequirePermission(c.Context(), *channel.ServerID, requesterID, models.PermMoveMembers); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "missing MOVE_MEMBERS permission",
			})
		}
	}

	err = h.liveKitVoiceService.DisconnectParticipant(c.Context(), channelID, targetUserID)
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

// MuteParticipant mutes or unmutes a participant in a voice channel (moderation)
func (h *VoiceHandler) MuteParticipant(c *fiber.Ctx) error {
	requesterID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

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

	channel, err := h.channelService.GetChannel(c.Context(), channelID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "channel not found",
		})
	}
	if channel.ServerID != nil && h.liveKitPermService != nil {
		if err := h.liveKitPermService.RequirePermission(c.Context(), *channel.ServerID, requesterID, models.PermMuteMembers); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "missing MUTE_MEMBERS permission",
			})
		}
	}

	err = h.liveKitVoiceService.MuteParticipant(c.Context(), channelID, targetUserID, req.Muted)
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

// GenerateTokenRequest is the request body for generating a voice token
type GenerateTokenRequest struct {
	ChannelID string `json:"channel_id"`
}

// GenerateTokenResponse is the response for generating a voice token
type GenerateTokenResponse struct {
	Token string `json:"token"`
	URL   string `json:"url"`
}

// ParticipantsResponse is the response for listing participants
type ParticipantsResponse struct {
	Participants []services.Participant `json:"participants"`
}

// MuteParticipantRequest is the request body for muting a participant
type MuteParticipantRequest struct {
	Muted bool `json:"muted"`
}

// CallChannelServiceInterface defines methods needed for channel access in call handler
type CallChannelServiceInterface interface {
	GetOrCreateDM(ctx context.Context, user1ID, user2ID uuid.UUID) (*models.Channel, error)
}

func (h *VoiceHandler) Create(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

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
		channel, err := h.callChannelService.GetOrCreateDM(c.Context(), userID, targetUserID)
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

func (h *VoiceHandler) Get(c *fiber.Ctx) error {
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

func (h *VoiceHandler) Join(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

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

func (h *VoiceHandler) Leave(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

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

func (h *VoiceHandler) Signal(c *fiber.Ctx) error {
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

func (h *VoiceHandler) StartStream(c *fiber.Ctx) error {
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

func (h *VoiceHandler) EndStream(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

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

func (h *VoiceHandler) GetStreamInfo(c *fiber.Ctx) error {
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

func (h *VoiceHandler) JoinStream(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

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

func (h *VoiceHandler) LeaveStream(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

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

func (h *VoiceHandler) UpdateStream(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

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

func (h *VoiceHandler) GetActiveStreamForChannel(c *fiber.Ctx) error {
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

func (h *VoiceHandler) StartLiveStream(c *fiber.Ctx) error {
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

func (h *VoiceHandler) StopLiveStream(c *fiber.Ctx) error {
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

func (h *VoiceHandler) GetActiveLiveStream(c *fiber.Ctx) error {
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

func (h *VoiceHandler) GetLiveStream(c *fiber.Ctx) error {
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

func (h *VoiceHandler) JoinLiveStream(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

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

func (h *VoiceHandler) LeaveLiveStream(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

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

func (h *VoiceHandler) UpdateLiveStream(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

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

func (h *VoiceHandler) GetLiveStreamViewers(c *fiber.Ctx) error {
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

func (h *VoiceHandler) CreateStage(c *fiber.Ctx) error {
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

	var req models.CreateStageRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.Topic == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "topic is required",
		})
	}

	stageInfo, err := h.stageService.CreateStage(c.Context(), channelID, userID, &req)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(stageInfo)
}

func (h *VoiceHandler) GetStage(c *fiber.Ctx) error {
	channelID, err := uuid.Parse(c.Params("channelId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	stageInfo, err := h.stageService.GetStage(c.Context(), channelID)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(stageInfo)
}

func (h *VoiceHandler) UpdateStage(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	stageID, err := uuid.Parse(c.Params("stageId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid stage id",
		})
	}

	var req models.UpdateStageRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	stageInfo, err := h.stageService.UpdateStage(c.Context(), stageID, userID, &req)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(stageInfo)
}

func (h *VoiceHandler) UpdateStageConfig(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	stageID, err := uuid.Parse(c.Params("stageId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid stage id",
		})
	}

	var req models.StageConfig
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	stageInfo, err := h.stageService.UpdateStageConfig(c.Context(), stageID, userID, &req)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(stageInfo)
}

func (h *VoiceHandler) PauseStage(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	stageID, err := uuid.Parse(c.Params("stageId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid stage id",
		})
	}

	stageInfo, err := h.stageService.PauseStage(c.Context(), stageID, userID)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(stageInfo)
}

func (h *VoiceHandler) ResumeStage(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	stageID, err := uuid.Parse(c.Params("stageId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid stage id",
		})
	}

	stageInfo, err := h.stageService.ResumeStage(c.Context(), stageID, userID)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(stageInfo)
}

func (h *VoiceHandler) EndStage(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	stageID, err := uuid.Parse(c.Params("stageId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid stage id",
		})
	}

	if err := h.stageService.EndStage(c.Context(), stageID, userID); err != nil {
		return h.handleError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *VoiceHandler) JoinStage(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	stageID, err := uuid.Parse(c.Params("stageId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid stage id",
		})
	}

	if err := h.stageService.JoinStage(c.Context(), stageID, userID); err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(fiber.Map{"message": "joined stage"})
}

func (h *VoiceHandler) LeaveStage(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	stageID, err := uuid.Parse(c.Params("stageId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid stage id",
		})
	}

	if err := h.stageService.LeaveStage(c.Context(), stageID, userID); err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(fiber.Map{"message": "left stage"})
}

func (h *VoiceHandler) RequestToSpeak(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	stageID, err := uuid.Parse(c.Params("stageId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid stage id",
		})
	}

	if err := h.stageService.RequestToSpeak(c.Context(), stageID, userID); err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(fiber.Map{"message": "request submitted"})
}

func (h *VoiceHandler) CancelRequestToSpeak(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	stageID, err := uuid.Parse(c.Params("stageId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid stage id",
		})
	}

	if err := h.stageService.CancelRequestToSpeak(c.Context(), stageID, userID); err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(fiber.Map{"message": "request cancelled"})
}

func (h *VoiceHandler) ApproveSpeaker(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	stageID, err := uuid.Parse(c.Params("stageId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid stage id",
		})
	}

	targetUserID, err := uuid.Parse(c.Params("userId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid user id",
		})
	}

	if err := h.stageService.ApproveSpeaker(c.Context(), stageID, userID, targetUserID); err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(fiber.Map{"message": "speaker approved"})
}

func (h *VoiceHandler) DenySpeaker(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	stageID, err := uuid.Parse(c.Params("stageId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid stage id",
		})
	}

	targetUserID, err := uuid.Parse(c.Params("userId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid user id",
		})
	}

	if err := h.stageService.DenySpeaker(c.Context(), stageID, userID, targetUserID); err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(fiber.Map{"message": "speaker denied"})
}

func (h *VoiceHandler) PromoteToSpeaker(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	stageID, err := uuid.Parse(c.Params("stageId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid stage id",
		})
	}

	targetUserID, err := uuid.Parse(c.Params("userId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid user id",
		})
	}

	if err := h.stageService.PromoteToSpeaker(c.Context(), stageID, userID, targetUserID); err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(fiber.Map{"message": "promoted to speaker"})
}

func (h *VoiceHandler) DemoteToAudience(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	stageID, err := uuid.Parse(c.Params("stageId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid stage id",
		})
	}

	targetUserID, err := uuid.Parse(c.Params("userId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid user id",
		})
	}

	if err := h.stageService.DemoteToAudience(c.Context(), stageID, userID, targetUserID); err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(fiber.Map{"message": "demoted to audience"})
}

func (h *VoiceHandler) AddModerator(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	stageID, err := uuid.Parse(c.Params("stageId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid stage id",
		})
	}

	targetUserID, err := uuid.Parse(c.Params("userId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid user id",
		})
	}

	if err := h.stageService.AddModerator(c.Context(), stageID, userID, targetUserID); err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(fiber.Map{"message": "moderator added"})
}

func (h *VoiceHandler) RemoveModerator(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	stageID, err := uuid.Parse(c.Params("stageId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid stage id",
		})
	}

	targetUserID, err := uuid.Parse(c.Params("userId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid user id",
		})
	}

	if err := h.stageService.RemoveModerator(c.Context(), stageID, userID, targetUserID); err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(fiber.Map{"message": "moderator removed"})
}

func (h *VoiceHandler) ListStageParticipants(c *fiber.Ctx) error {
	stageID, err := uuid.Parse(c.Params("stageId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid stage id",
		})
	}

	participants, err := h.stageService.ListParticipants(c.Context(), stageID)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(participants)
}

func (h *VoiceHandler) ListPendingRequests(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	stageID, err := uuid.Parse(c.Params("stageId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid stage id",
		})
	}

	requests, err := h.stageService.ListPendingRequests(c.Context(), stageID, userID)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(requests)
}

func (h *VoiceHandler) MuteStageParticipant(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	stageID, err := uuid.Parse(c.Params("stageId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid stage id",
		})
	}

	targetUserID, err := uuid.Parse(c.Params("userId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid user id",
		})
	}

	if err := h.stageService.MuteParticipant(c.Context(), stageID, userID, targetUserID, true); err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(fiber.Map{"message": "participant muted"})
}

func (h *VoiceHandler) UnmuteStageParticipant(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	stageID, err := uuid.Parse(c.Params("stageId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid stage id",
		})
	}

	targetUserID, err := uuid.Parse(c.Params("userId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid user id",
		})
	}

	if err := h.stageService.MuteParticipant(c.Context(), stageID, userID, targetUserID, false); err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(fiber.Map{"message": "participant unmuted"})
}

func (h *VoiceHandler) GetVoiceToken(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	stageID, err := uuid.Parse(c.Params("stageId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid stage id",
		})
	}

	// Get user info for token
	user, err := h.userService.GetUser(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get user",
		})
	}

	displayName := ""
	if user.DisplayName != nil {
		displayName = *user.DisplayName
	}
	avatarURL := ""
	if user.AvatarURL != nil {
		avatarURL = *user.AvatarURL
	}

	token, err := h.stageService.GenerateVoiceToken(
		c.Context(), stageID, userID,
		user.Username, displayName, avatarURL,
	)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(token)
}

func (h *VoiceHandler) handleError(c *fiber.Ctx, err error) error {
	switch err {
	case services.ErrStageNotFound:
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "stage not found"})
	case services.ErrStageAlreadyExists:
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "stage already exists for this channel"})
	case services.ErrStageNotActive:
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "stage is not active"})
	case services.ErrStageNotLive:
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "stage is not live"})
	case services.ErrStageNotPaused:
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "stage is not paused"})
	case services.ErrNotStageHost:
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "not the stage host"})
	case services.ErrNotStageModerator:
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "not a stage moderator"})
	case services.ErrNotStageParticipant:
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "not a stage participant"})
	case services.ErrCannotModifyHost:
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "cannot modify host role"})
	case services.ErrMaxSpeakersReached:
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "maximum speakers reached"})
	case services.ErrSpeakerRequestPending:
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "speaker request already pending"})
	case services.ErrSpeakerRequestNotPending:
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "no pending speaker request"})
	case services.ErrNotAudienceMember:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "user is not an audience member"})
	case services.ErrNotSpeaker:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "user is not a speaker"})
	case services.ErrModeratorOnly:
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "stage is moderator-only"})
	case services.ErrChannelNotStage:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "channel is not a stage channel"})
	case services.ErrNotStageHostOrMod:
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "not the stage host or a moderator"})
	case services.ErrChannelNotFound:
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "channel not found"})
	case services.ErrNotServerMember:
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "not a server member"})
	case services.ErrLiveKitNotConfigured:
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "voice not available"})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
	}
}

func (h *VoiceHandler) StartActivity(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
	channelID, err := uuid.Parse(c.Params("channelId"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid channel ID"})
	}

	var req models.StartActivityRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	// Look up the channel to get server ID
	channel, err := h.channelService.GetChannel(c.Context(), channelID)
	if err != nil || channel == nil {
		return c.Status(404).JSON(fiber.Map{"error": "channel not found"})
	}

	if channel.ServerID == nil {
		return c.Status(400).JSON(fiber.Map{"error": "activities are only available in server voice channels"})
	}

	result, err := h.activityService.StartActivity(c.Context(), channelID, *channel.ServerID, userID, &req)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(201).JSON(result)
}

func (h *VoiceHandler) JoinActivity(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
	activityID, err := uuid.Parse(c.Params("activityId"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid activity ID"})
	}

	result, err := h.activityService.JoinActivity(c.Context(), activityID, userID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(result)
}

func (h *VoiceHandler) LeaveActivity(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
	activityID, err := uuid.Parse(c.Params("activityId"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid activity ID"})
	}

	if err := h.activityService.LeaveActivity(c.Context(), activityID, userID); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(204)
}

func (h *VoiceHandler) EndActivity(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
	activityID, err := uuid.Parse(c.Params("activityId"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid activity ID"})
	}

	if err := h.activityService.EndActivity(c.Context(), activityID, userID); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(204)
}

func (h *VoiceHandler) GetActivity(c *fiber.Ctx) error {
	activityID, err := uuid.Parse(c.Params("activityId"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid activity ID"})
	}

	result, err := h.activityService.GetActivity(c.Context(), activityID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(result)
}

func (h *VoiceHandler) GetChannelActivity(c *fiber.Ctx) error {
	channelID, err := uuid.Parse(c.Params("channelId"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid channel ID"})
	}

	result, err := h.activityService.GetChannelActivity(c.Context(), channelID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	if result == nil {
		return c.Status(404).JSON(fiber.Map{"error": "no active activity in this channel"})
	}

	return c.JSON(result)
}

func (h *VoiceHandler) GetGameState(c *fiber.Ctx) error {
	activityID, err := uuid.Parse(c.Params("activityId"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid activity ID"})
	}

	state, err := h.activityService.GetGameState(c.Context(), activityID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	if state == nil {
		return c.Status(404).JSON(fiber.Map{"error": "game state not found"})
	}

	return c.JSON(state)
}

func (h *VoiceHandler) GameMove(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
	activityID, err := uuid.Parse(c.Params("activityId"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid activity ID"})
	}

	var req models.GameMoveRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	state, err := h.activityService.ProcessGameMove(c.Context(), activityID, userID, &req)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(state)
}

