package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// InviteHandlers handles invite-related HTTP requests
type InviteHandlers struct {
	inviteService  *services.InviteService
	channelService *services.ChannelService
}

// NewInviteHandlers creates new invite handlers
func NewInviteHandlers(inviteService *services.InviteService, channelService *services.ChannelService) *InviteHandlers {
	return &InviteHandlers{
		inviteService:  inviteService,
		channelService: channelService,
	}
}

// CreateInviteRequest represents an invite creation request
type CreateInviteRequest struct {
	MaxAge    int  `json:"max_age"`  // Seconds, 0 = never expires
	MaxUses   int  `json:"max_uses"` // 0 = unlimited
	Temporary bool `json:"temporary"`
}

// InviteResponse represents an invite in API responses
type InviteResponse struct {
	Code      string     `json:"code"`
	ServerID  string     `json:"guild_id"`
	ChannelID string     `json:"channel_id"`
	CreatorID string     `json:"inviter_id"`
	MaxUses   int        `json:"max_uses"`
	Uses      int        `json:"uses"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	Temporary bool       `json:"temporary"`
	CreatedAt time.Time  `json:"created_at"`
}

// CreateInvite creates a new invite for a channel
// @Summary Create channel invite
// @Description Creates a new invite for a specific channel
// @Tags Invites
// @Accept json
// @Produce json
// @Param channelID path string true "Channel ID"
// @Param body body CreateInviteRequest true "Invite creation options"
// @Success 200 {object} InviteResponse "Invite created successfully"
// @Failure 400 {object} fiber.Map "Invalid channel ID, request body, or DM channel"
// @Failure 404 {object} fiber.Map "Channel not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{channelID}/invites [post]
func (h *InviteHandlers) CreateInvite(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	channelID, err := uuid.Parse(c.Params("channelID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid channel ID",
		})
	}

	var req CreateInviteRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Get channel to find server ID
	channel, err := h.channelService.GetChannel(c.Context(), channelID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Channel not found",
		})
	}
	if channel.ServerID == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot create invite for DM channel",
		})
	}
	serverID := *channel.ServerID

	var maxAge time.Duration
	if req.MaxAge > 0 {
		maxAge = time.Duration(req.MaxAge) * time.Second
	}

	invite, err := h.inviteService.CreateInvite(c.Context(), &services.CreateInviteRequest{
		ServerID:  serverID,
		ChannelID: channelID,
		CreatorID: userID,
		MaxAge:    maxAge,
		MaxUses:   req.MaxUses,
		Temporary: req.Temporary,
	})
	if err != nil {
		if err == services.ErrInviteRateLimited {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":   "rate_limited",
				"message":  "Too many invites created. Please wait before creating more.",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(toInviteResponse(invite))
}

// GetInvite retrieves an invite by code
// @Summary Get invite by code
// @Description Retrieves an invite by its unique code
// @Tags Invites
// @Produce json
// @Param code path string true "Invite code"
// @Success 200 {object} InviteResponse "Invite details"
// @Failure 400 {object} fiber.Map "Invalid invite code"
// @Failure 404 {object} fiber.Map "Invite not found"
// @Router /invites/{code} [get]
func (h *InviteHandlers) GetInvite(c *fiber.Ctx) error {
	code := c.Params("code")
	if code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid invite code",
		})
	}

	invite, err := h.inviteService.GetInvite(c.Context(), code)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Invite not found",
		})
	}

	return c.JSON(toInviteResponse(invite))
}

// UseInvite uses an invite to join a server
// @Summary Use invite to join server
// @Description Uses an invite code to join a server
// @Tags Invites
// @Produce json
// @Param code path string true "Invite code"
// @Success 200 {object} models.Server "Server joined successfully"
// @Failure 400 {object} fiber.Map "Invalid invite code or invite expired/invalid"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Router /invites/{code} [post]
func (h *InviteHandlers) UseInvite(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	code := c.Params("code")
	if code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid invite code",
		})
	}

	server, err := h.inviteService.UseInvite(c.Context(), code, userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(server)
}

// DeleteInvite deletes an invite
// @Summary Delete invite
// @Description Deletes an invite by its code (requires permissions)
// @Tags Invites
// @Param code path string true "Invite code"
// @Success 204 "Invite deleted successfully"
// @Failure 400 {object} fiber.Map "Invalid invite code"
// @Failure 403 {object} fiber.Map "Forbidden - insufficient permissions"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Router /invites/{code} [delete]
func (h *InviteHandlers) DeleteInvite(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	code := c.Params("code")
	if code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid invite code",
		})
	}

	err := h.inviteService.DeleteInvite(c.Context(), code, userID)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// GetChannelInvites returns all invites for a specific channel
// @Summary Get channel invites
// @Description Returns all invites for a specific channel
// @Tags Invites
// @Produce json
// @Param channelID path string true "Channel ID"
// @Success 200 {array} InviteResponse "List of channel invites"
// @Failure 400 {object} fiber.Map "Invalid channel ID or DM channel"
// @Failure 404 {object} fiber.Map "Channel not found"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{channelID}/invites [get]
func (h *InviteHandlers) GetChannelInvites(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	channelID, err := uuid.Parse(c.Params("channelID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid channel ID",
		})
	}

	// Get channel to find server ID
	channel, err := h.channelService.GetChannel(c.Context(), channelID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Channel not found",
		})
	}
	if channel.ServerID == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "DM channels cannot have invites",
		})
	}

	// Get all server invites and filter by channel
	invites, err := h.inviteService.GetServerInvites(c.Context(), *channel.ServerID, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Filter to only this channel's invites
	var channelInvites []InviteResponse
	for _, inv := range invites {
		if inv.ChannelID == channelID {
			channelInvites = append(channelInvites, InviteResponse{
				Code:      inv.Code,
				ServerID:  inv.ServerID.String(),
				ChannelID: inv.ChannelID.String(),
				Uses:      inv.Uses,
				MaxUses:   inv.MaxUses,
				ExpiresAt: inv.ExpiresAt,
				CreatedAt: inv.CreatedAt,
			})
		}
	}

	if channelInvites == nil {
		channelInvites = []InviteResponse{}
	}
	return c.JSON(channelInvites)
}

// GetServerInvites returns all invites for a server
// @Summary Get server invites
// @Description Returns all invites for a specific server
// @Tags Invites
// @Produce json
// @Param serverID path string true "Server ID"
// @Success 200 {array} InviteResponse "List of server invites"
// @Failure 400 {object} fiber.Map "Invalid server ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{serverID}/invites [get]
func (h *InviteHandlers) GetServerInvites(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	serverID, err := uuid.Parse(c.Params("serverID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid server ID",
		})
	}

	invites, err := h.inviteService.GetServerInvites(c.Context(), serverID, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	responses := make([]InviteResponse, len(invites))
	for i, invite := range invites {
		responses[i] = toInviteResponse(invite)
	}

	return c.JSON(responses)
}

// toInviteResponse converts a model invite to an API response
func toInviteResponse(invite *models.Invite) InviteResponse {
	return InviteResponse{
		Code:      invite.Code,
		ServerID:  invite.ServerID.String(),
		ChannelID: invite.ChannelID.String(),
		CreatorID: invite.CreatorID.String(),
		MaxUses:   invite.MaxUses,
		Uses:      invite.Uses,
		ExpiresAt: invite.ExpiresAt,
		Temporary: invite.Temporary,
		CreatedAt: invite.CreatedAt,
	}
}
