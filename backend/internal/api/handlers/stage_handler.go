package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// StageHandler handles stage channel API requests
type StageHandler struct {
	stageService   *services.StageService
	channelService *services.ChannelService
	permService    *services.PermissionService
	userService    *services.UserService
}

// NewStageHandler creates a new stage handler
func NewStageHandler(
	stageService *services.StageService,
	channelService *services.ChannelService,
	permService *services.PermissionService,
	userService *services.UserService,
) *StageHandler {
	return &StageHandler{
		stageService:   stageService,
		channelService: channelService,
		permService:    permService,
		userService:    userService,
	}
}

// CreateStage creates and starts a new stage
// @Summary Create stage
// @Description Creates and starts a new stage in a stage channel
// @Tags Stages
// @Accept json
// @Produce json
// @Param channelId path string true "Channel ID"
// @Param body body models.CreateStageRequest true "Stage settings"
// @Success 201 {object} models.StageInfo "Stage created successfully"
// @Failure 400 {object} fiber.Map "Invalid channel ID or request body"
// @Failure 403 {object} fiber.Map "Not a member or missing permission"
// @Failure 404 {object} fiber.Map "Channel not found"
// @Failure 409 {object} fiber.Map "Stage already exists for this channel"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{channelId}/stage [post]
func (h *StageHandler) CreateStage(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

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

// GetStage gets the current stage for a channel
// @Summary Get stage
// @Description Gets the current stage info for a channel
// @Tags Stages
// @Produce json
// @Param channelId path string true "Channel ID"
// @Success 200 {object} models.StageInfo "Stage info"
// @Failure 400 {object} fiber.Map "Invalid channel ID"
// @Failure 404 {object} fiber.Map "Stage not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{channelId}/stage [get]
func (h *StageHandler) GetStage(c *fiber.Ctx) error {
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

// UpdateStage updates stage metadata
// @Summary Update stage
// @Description Updates stage topic and description
// @Tags Stages
// @Accept json
// @Produce json
// @Param stageId path string true "Stage ID"
// @Param body body models.UpdateStageRequest true "Stage updates"
// @Success 200 {object} models.StageInfo "Stage updated"
// @Failure 400 {object} fiber.Map "Invalid stage ID or request body"
// @Failure 403 {object} fiber.Map "Not authorized"
// @Failure 404 {object} fiber.Map "Stage not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /stages/{stageId} [patch]
func (h *StageHandler) UpdateStage(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

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

// UpdateStageConfig updates stage configuration
// @Summary Update stage config
// @Description Updates stage configuration (moderator only, request to speak, etc.)
// @Tags Stages
// @Accept json
// @Produce json
// @Param stageId path string true "Stage ID"
// @Param body body models.StageConfig true "Stage config"
// @Success 200 {object} models.StageInfo "Stage updated"
// @Failure 400 {object} fiber.Map "Invalid stage ID or request body"
// @Failure 403 {object} fiber.Map "Not authorized"
// @Failure 404 {object} fiber.Map "Stage not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /stages/{stageId}/config [patch]
func (h *StageHandler) UpdateStageConfig(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

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

// PauseStage pauses a live stage
// @Summary Pause stage
// @Description Pauses a live stage
// @Tags Stages
// @Param stageId path string true "Stage ID"
// @Success 200 {object} models.StageInfo "Stage paused"
// @Failure 400 {object} fiber.Map "Invalid stage ID"
// @Failure 403 {object} fiber.Map "Not authorized"
// @Failure 404 {object} fiber.Map "Stage not found"
// @Failure 409 {object} fiber.Map "Stage not live"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /stages/{stageId}/pause [post]
func (h *StageHandler) PauseStage(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

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

// ResumeStage resumes a paused stage
// @Summary Resume stage
// @Description Resumes a paused stage
// @Tags Stages
// @Param stageId path string true "Stage ID"
// @Success 200 {object} models.StageInfo "Stage resumed"
// @Failure 400 {object} fiber.Map "Invalid stage ID"
// @Failure 403 {object} fiber.Map "Not authorized"
// @Failure 404 {object} fiber.Map "Stage not found"
// @Failure 409 {object} fiber.Map "Stage not paused"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /stages/{stageId}/resume [post]
func (h *StageHandler) ResumeStage(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

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

// EndStage ends a stage
// @Summary End stage
// @Description Ends a stage and removes all participants
// @Tags Stages
// @Param stageId path string true "Stage ID"
// @Success 204
// @Failure 400 {object} fiber.Map "Invalid stage ID"
// @Failure 403 {object} fiber.Map "Not authorized"
// @Failure 404 {object} fiber.Map "Stage not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /stages/{stageId} [delete]
func (h *StageHandler) EndStage(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

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

// JoinStage joins a stage as an audience member
// @Summary Join stage
// @Description Joins a stage as an audience member
// @Tags Stages
// @Param stageId path string true "Stage ID"
// @Success 200 {object} fiber.Map "Joined"
// @Failure 400 {object} fiber.Map "Invalid stage ID"
// @Failure 404 {object} fiber.Map "Stage not found"
// @Failure 409 {object} fiber.Map "Stage not active"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /stages/{stageId}/join [post]
func (h *StageHandler) JoinStage(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

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

// LeaveStage leaves a stage
// @Summary Leave stage
// @Description Leaves a stage
// @Tags Stages
// @Param stageId path string true "Stage ID"
// @Success 200 {object} fiber.Map "Left"
// @Failure 400 {object} fiber.Map "Invalid stage ID"
// @Failure 403 {object} fiber.Map "Cannot leave as host"
// @Failure 404 {object} fiber.Map "Stage not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /stages/{stageId}/leave [post]
func (h *StageHandler) LeaveStage(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

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

// RequestToSpeak requests speaker privileges
// @Summary Request to speak
// @Description Requests to become a speaker
// @Tags Stages
// @Param stageId path string true "Stage ID"
// @Success 200 {object} fiber.Map "Request submitted"
// @Failure 400 {object} fiber.Map "Invalid stage ID"
// @Failure 403 {object} fiber.Map "Moderator only stage"
// @Failure 404 {object} fiber.Map "Stage not found"
// @Failure 409 {object} fiber.Map "Request pending or max speakers reached"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /stages/{stageId}/request-to-speak [post]
func (h *StageHandler) RequestToSpeak(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

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

// CancelRequestToSpeak cancels a pending speaker request
// @Summary Cancel request to speak
// @Description Cancels a pending speaker request
// @Tags Stages
// @Param stageId path string true "Stage ID"
// @Success 200 {object} fiber.Map "Request cancelled"
// @Failure 400 {object} fiber.Map "Invalid stage ID"
// @Failure 404 {object} fiber.Map "Stage not found"
// @Failure 409 {object} fiber.Map "No pending request"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /stages/{stageId}/request-to-speak [delete]
func (h *StageHandler) CancelRequestToSpeak(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

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

// ApproveSpeaker approves a speaker request
// @Summary Approve speaker
// @Description Approves a user's request to speak
// @Tags Stages
// @Param stageId path string true "Stage ID"
// @Param userId path string true "User ID"
// @Success 200 {object} fiber.Map "Approved"
// @Failure 400 {object} fiber.Map "Invalid stage or user ID"
// @Failure 403 {object} fiber.Map "Not authorized"
// @Failure 404 {object} fiber.Map "Stage not found"
// @Failure 409 {object} fiber.Map "No pending request"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /stages/{stageId}/approve/{userId} [post]
func (h *StageHandler) ApproveSpeaker(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

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

// DenySpeaker denies a speaker request
// @Summary Deny speaker
// @Description Denies a user's request to speak
// @Tags Stages
// @Param stageId path string true "Stage ID"
// @Param userId path string true "User ID"
// @Success 200 {object} fiber.Map "Denied"
// @Failure 400 {object} fiber.Map "Invalid stage or user ID"
// @Failure 403 {object} fiber.Map "Not authorized"
// @Failure 404 {object} fiber.Map "Stage not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /stages/{stageId}/deny/{userId} [post]
func (h *StageHandler) DenySpeaker(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

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

// PromoteToSpeaker promotes an audience member to speaker
// @Summary Promote to speaker
// @Description Promotes an audience member to speaker
// @Tags Stages
// @Param stageId path string true "Stage ID"
// @Param userId path string true "User ID"
// @Success 200 {object} fiber.Map "Promoted"
// @Failure 400 {object} fiber.Map "Invalid stage or user ID"
// @Failure 403 {object} fiber.Map "Not authorized"
// @Failure 404 {object} fiber.Map "Stage not found"
// @Failure 409 {object} fiber.Map "Max speakers reached"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /stages/{stageId}/promote/{userId} [post]
func (h *StageHandler) PromoteToSpeaker(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

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

// DemoteToAudience demotes a speaker to audience
// @Summary Demote to audience
// @Description Demotes a speaker to audience
// @Tags Stages
// @Param stageId path string true "Stage ID"
// @Param userId path string true "User ID"
// @Success 200 {object} fiber.Map "Demoted"
// @Failure 400 {object} fiber.Map "Invalid stage or user ID"
// @Failure 403 {object} fiber.Map "Not authorized"
// @Failure 404 {object} fiber.Map "Stage not found"
// @Failure 409 {object} fiber.Map "Cannot demote host"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /stages/{stageId}/demote/{userId} [post]
func (h *StageHandler) DemoteToAudience(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

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

// AddModerator adds a moderator to the stage
// @Summary Add moderator
// @Description Adds a moderator to the stage (host only)
// @Tags Stages
// @Param stageId path string true "Stage ID"
// @Param userId path string true "User ID"
// @Success 200 {object} fiber.Map "Moderator added"
// @Failure 400 {object} fiber.Map "Invalid stage or user ID"
// @Failure 403 {object} fiber.Map "Not authorized"
// @Failure 404 {object} fiber.Map "Stage not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /stages/{stageId}/moderators/{userId} [post]
func (h *StageHandler) AddModerator(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

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

// RemoveModerator removes a moderator from the stage
// @Summary Remove moderator
// @Description Removes a moderator from the stage (host only)
// @Tags Stages
// @Param stageId path string true "Stage ID"
// @Param userId path string true "User ID"
// @Success 200 {object} fiber.Map "Moderator removed"
// @Failure 400 {object} fiber.Map "Invalid stage or user ID"
// @Failure 403 {object} fiber.Map "Not authorized"
// @Failure 404 {object} fiber.Map "Stage not found"
// @Failure 409 {object} fiber.Map "User is not a moderator"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /stages/{stageId}/moderators/{userId} [delete]
func (h *StageHandler) RemoveModerator(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

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

// ListParticipants lists all participants in a stage
// @Summary List participants
// @Description Lists all participants in a stage
// @Tags Stages
// @Produce json
// @Param stageId path string true "Stage ID"
// @Success 200 {array} models.ParticipantInfo "Participants"
// @Failure 400 {object} fiber.Map "Invalid stage ID"
// @Failure 404 {object} fiber.Map "Stage not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /stages/{stageId}/participants [get]
func (h *StageHandler) ListParticipants(c *fiber.Ctx) error {
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

// ListPendingRequests lists pending speaker requests
// @Summary List speaker requests
// @Description Lists pending speaker requests (host/mod only)
// @Tags Stages
// @Produce json
// @Param stageId path string true "Stage ID"
// @Success 200 {array} models.ParticipantInfo "Requests"
// @Failure 400 {object} fiber.Map "Invalid stage ID"
// @Failure 403 {object} fiber.Map "Not authorized"
// @Failure 404 {object} fiber.Map "Stage not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /stages/{stageId}/requests [get]
func (h *StageHandler) ListPendingRequests(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

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

// MuteParticipant mutes a participant
// @Summary Mute participant
// @Description Mutes a participant (host/mod only)
// @Tags Stages
// @Param stageId path string true "Stage ID"
// @Param userId path string true "User ID"
// @Success 200 {object} fiber.Map "Muted"
// @Failure 400 {object} fiber.Map "Invalid stage or user ID"
// @Failure 403 {object} fiber.Map "Not authorized"
// @Failure 404 {object} fiber.Map "Stage not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /stages/{stageId}/mute/{userId} [post]
func (h *StageHandler) MuteParticipant(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

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

// UnmuteParticipant unmutes a participant
// @Summary Unmute participant
// @Description Unmutes a participant (host/mod only)
// @Tags Stages
// @Param stageId path string true "Stage ID"
// @Param userId path string true "User ID"
// @Success 200 {object} fiber.Map "Unmuted"
// @Failure 400 {object} fiber.Map "Invalid stage or user ID"
// @Failure 403 {object} fiber.Map "Not authorized"
// @Failure 404 {object} fiber.Map "Stage not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /stages/{stageId}/unmute/{userId} [post]
func (h *StageHandler) UnmuteParticipant(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

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

// GetVoiceToken generates a LiveKit token for stage participation
// @Summary Get voice token
// @Description Gets a LiveKit token for joining the stage voice channel
// @Tags Stages
// @Produce json
// @Param stageId path string true "Stage ID"
// @Success 200 {object} services.VoiceTokenResponse "Token"
// @Failure 400 {object} fiber.Map "Invalid stage ID"
// @Failure 403 {object} fiber.Map "Not a stage participant"
// @Failure 404 {object} fiber.Map "Stage not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /stages/{stageId}/token [get]
func (h *StageHandler) GetVoiceToken(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

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

// handleError converts service errors to HTTP responses
func (h *StageHandler) handleError(c *fiber.Ctx, err error) error {
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
