package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// VoiceActivityHandler handles voice activity HTTP endpoints
type VoiceActivityHandler struct {
	activityService *services.VoiceActivityService
	channelService  services.ChannelServiceInterface
	permService     services.PermissionServiceInterface
}

// NewVoiceActivityHandler creates a new voice activity handler
func NewVoiceActivityHandler(
	activityService *services.VoiceActivityService,
	channelService services.ChannelServiceInterface,
	permService services.PermissionServiceInterface,
) *VoiceActivityHandler {
	return &VoiceActivityHandler{
		activityService: activityService,
		channelService:  channelService,
		permService:     permService,
	}
}

// StartActivity starts a new activity in a voice channel
// POST /channels/:channelId/activities
func (h *VoiceActivityHandler) StartActivity(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
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

// JoinActivity joins an existing activity
// POST /activities/:activityId/join
func (h *VoiceActivityHandler) JoinActivity(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
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

// LeaveActivity leaves an activity
// DELETE /activities/:activityId/participants/@me
func (h *VoiceActivityHandler) LeaveActivity(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	activityID, err := uuid.Parse(c.Params("activityId"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid activity ID"})
	}

	if err := h.activityService.LeaveActivity(c.Context(), activityID, userID); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(204)
}

// EndActivity ends an activity (creator only)
// DELETE /activities/:activityId
func (h *VoiceActivityHandler) EndActivity(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	activityID, err := uuid.Parse(c.Params("activityId"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid activity ID"})
	}

	if err := h.activityService.EndActivity(c.Context(), activityID, userID); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(204)
}

// GetActivity gets an activity with participants
// GET /activities/:activityId
func (h *VoiceActivityHandler) GetActivity(c *fiber.Ctx) error {
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

// GetChannelActivity gets the active activity for a channel
// GET /channels/:channelId/activities
func (h *VoiceActivityHandler) GetChannelActivity(c *fiber.Ctx) error {
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

// GetGameState gets the current game state
// GET /activities/:activityId/state
func (h *VoiceActivityHandler) GetGameState(c *fiber.Ctx) error {
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

// GameMove processes a game move
// POST /activities/:activityId/moves
func (h *VoiceActivityHandler) GameMove(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
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
