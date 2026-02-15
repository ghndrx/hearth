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
	inviteService *services.InviteService
}

// NewInviteHandlers creates new invite handlers
func NewInviteHandlers(inviteService *services.InviteService) *InviteHandlers {
	return &InviteHandlers{inviteService: inviteService}
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
	// TODO: Get channel from service to get server ID
	serverID := uuid.Nil // Placeholder

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
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(toInviteResponse(invite))
}

// GetInvite retrieves an invite by code
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

// GetChannelInvites returns all invites for a channel
// Note: Channel-specific invites not implemented - use server invites instead
func (h *InviteHandlers) GetChannelInvites(c *fiber.Ctx) error {
	_, err := uuid.Parse(c.Params("channelID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid channel ID",
		})
	}

	// TODO: Filter server invites by channel
	return c.JSON([]InviteResponse{})
}

// GetServerInvites returns all invites for a server
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
