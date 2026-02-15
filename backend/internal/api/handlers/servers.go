package handlers

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

type ServerHandler struct {
	serverService  *services.ServerService
	channelService *services.ChannelService
	roleService    *services.RoleService
}

func NewServerHandler(
	serverService *services.ServerService,
	channelService *services.ChannelService,
	roleService *services.RoleService,
) *ServerHandler {
	return &ServerHandler{
		serverService:  serverService,
		channelService: channelService,
		roleService:    roleService,
	}
}

// Create creates a new server
func (h *ServerHandler) Create(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	var req struct {
		Name string `json:"name" validate:"required,min=2,max=100"`
		Icon string `json:"icon"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.Name == "" || len(req.Name) < 2 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "name must be at least 2 characters",
		})
	}

	server, err := h.serverService.CreateServer(c.Context(), userID, req.Name, req.Icon)
	if err != nil {
		if err == services.ErrMaxServersReached {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "maximum servers owned limit reached",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(server)
}

// Get returns a server by ID
func (h *ServerHandler) Get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	server, err := h.serverService.GetServer(c.Context(), id)
	if err != nil {
		if err == services.ErrServerNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "server not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(server)
}

// Update updates a server
func (h *ServerHandler) Update(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	var req struct {
		Name        *string `json:"name"`
		Icon        *string `json:"icon"`
		Banner      *string `json:"banner"`
		Description *string `json:"description"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	updates := &models.ServerUpdate{
		Name:        req.Name,
		IconURL:     req.Icon,
		BannerURL:   req.Banner,
		Description: req.Description,
	}

	server, err := h.serverService.UpdateServer(c.Context(), id, userID, updates)
	if err != nil {
		switch err {
		case services.ErrServerNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "server not found",
			})
		case services.ErrNotServerMember:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "not a member of this server",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
	}

	return c.JSON(server)
}

// Delete deletes a server
func (h *ServerHandler) Delete(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	if err := h.serverService.DeleteServer(c.Context(), id, userID); err != nil {
		switch err {
		case services.ErrServerNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "server not found",
			})
		case services.ErrNotServerOwner:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "only server owner can delete",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// GetMembers returns server members
func (h *ServerHandler) GetMembers(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	// Parse pagination
	limit, _ := strconv.Atoi(c.Query("limit", "100"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	if limit > 1000 {
		limit = 1000
	}
	if limit < 1 {
		limit = 100
	}

	members, err := h.serverService.GetMembers(c.Context(), id, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(members)
}

// GetMember returns a specific member
func (h *ServerHandler) GetMember(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	userID, err := uuid.Parse(c.Params("userId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid user id",
		})
	}

	member, err := h.serverService.GetMember(c.Context(), serverID, userID)
	if err != nil {
		if err == services.ErrNotServerMember {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "member not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(member)
}

// UpdateMember updates a member (nickname, roles)
func (h *ServerHandler) UpdateMember(c *fiber.Ctx) error {
	requesterID := c.Locals("userID").(uuid.UUID)
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	targetID, err := uuid.Parse(c.Params("userId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid user id",
		})
	}

	var req struct {
		Nickname *string     `json:"nick"`
		Roles    []uuid.UUID `json:"roles"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	member, err := h.serverService.UpdateMember(c.Context(), serverID, requesterID, targetID, req.Nickname, req.Roles)
	if err != nil {
		if err == services.ErrNotServerMember {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "member not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(member)
}

// RemoveMember kicks a member
func (h *ServerHandler) RemoveMember(c *fiber.Ctx) error {
	requesterID := c.Locals("userID").(uuid.UUID)
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	targetID, err := uuid.Parse(c.Params("userId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid user id",
		})
	}

	// Optional reason in body or query
	reason := c.Query("reason", "")

	if err := h.serverService.KickMember(c.Context(), serverID, requesterID, targetID, reason); err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// Leave leaves the server
func (h *ServerHandler) Leave(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	if err := h.serverService.LeaveServer(c.Context(), id, userID); err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// GetBans returns server bans
func (h *ServerHandler) GetBans(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	bans, err := h.serverService.GetBans(c.Context(), serverID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if bans == nil {
		bans = []*models.Ban{}
	}
	return c.JSON(bans)
}

// CreateBan bans a user
func (h *ServerHandler) CreateBan(c *fiber.Ctx) error {
	requesterID := c.Locals("userID").(uuid.UUID)
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	targetID, err := uuid.Parse(c.Params("userId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid user id",
		})
	}

	var req struct {
		Reason               string `json:"reason"`
		DeleteMessageDays    int    `json:"delete_message_days"`
		DeleteMessageSeconds int    `json:"delete_message_seconds"`
	}
	_ = c.BodyParser(&req)

	// Convert delete days to actual days if seconds provided
	deleteDays := req.DeleteMessageDays
	if req.DeleteMessageSeconds > 0 {
		deleteDays = req.DeleteMessageSeconds / 86400
	}

	if err := h.serverService.BanMember(c.Context(), serverID, requesterID, targetID, req.Reason, deleteDays); err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// RemoveBan unbans a user
func (h *ServerHandler) RemoveBan(c *fiber.Ctx) error {
	requesterID := c.Locals("userID").(uuid.UUID)
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	targetID, err := uuid.Parse(c.Params("userId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid user id",
		})
	}

	if err := h.serverService.UnbanMember(c.Context(), serverID, requesterID, targetID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// GetInvites returns server invites
func (h *ServerHandler) GetInvites(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	invites, err := h.serverService.GetInvites(c.Context(), serverID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if invites == nil {
		invites = []*models.Invite{}
	}
	return c.JSON(invites)
}

// GetRoles returns server roles
func (h *ServerHandler) GetRoles(c *fiber.Ctx) error {
	requesterID := c.Locals("userID").(uuid.UUID)
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	roles, err := h.roleService.GetServerRoles(c.Context(), serverID, requesterID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if roles == nil {
		roles = []*models.Role{}
	}
	return c.JSON(roles)
}

// CreateRole creates a new role
func (h *ServerHandler) CreateRole(c *fiber.Ctx) error {
	requesterID := c.Locals("userID").(uuid.UUID)
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	var req struct {
		Name        string `json:"name"`
		Color       int    `json:"color"`
		Permissions int64  `json:"permissions"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.Name == "" {
		req.Name = "new role"
	}

	role, err := h.roleService.CreateRole(c.Context(), serverID, requesterID, req.Name, req.Color, req.Permissions)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(role)
}

// UpdateRole updates a role
func (h *ServerHandler) UpdateRole(c *fiber.Ctx) error {
	requesterID := c.Locals("userID").(uuid.UUID)
	_, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	roleID, err := uuid.Parse(c.Params("roleId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid role id",
		})
	}

	var req struct {
		Name        *string `json:"name"`
		Color       *int    `json:"color"`
		Hoist       *bool   `json:"hoist"`
		Permissions *int64  `json:"permissions"`
		Mentionable *bool   `json:"mentionable"`
		Position    *int    `json:"position"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	updates := &models.RoleUpdate{
		Name:        req.Name,
		Color:       req.Color,
		Hoist:       req.Hoist,
		Permissions: req.Permissions,
		Mentionable: req.Mentionable,
		Position:    req.Position,
	}

	role, err := h.roleService.UpdateRole(c.Context(), roleID, requesterID, updates)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(role)
}

// DeleteRole deletes a role
func (h *ServerHandler) DeleteRole(c *fiber.Ctx) error {
	requesterID := c.Locals("userID").(uuid.UUID)
	_, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	roleID, err := uuid.Parse(c.Params("roleId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid role id",
		})
	}

	if err := h.roleService.DeleteRole(c.Context(), roleID, requesterID); err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// GetChannels returns server channels
func (h *ServerHandler) GetChannels(c *fiber.Ctx) error {
	requesterID := c.Locals("userID").(uuid.UUID)
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	channels, err := h.channelService.GetServerChannels(c.Context(), serverID, requesterID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if channels == nil {
		channels = []*models.Channel{}
	}
	return c.JSON(channels)
}

// CreateChannel creates a new channel
func (h *ServerHandler) CreateChannel(c *fiber.Ctx) error {
	requesterID := c.Locals("userID").(uuid.UUID)
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	var req struct {
		Name     string             `json:"name"`
		Type     models.ChannelType `json:"type"`
		ParentID *uuid.UUID         `json:"parent_id"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "name is required",
		})
	}

	// Default to text channel
	if req.Type == "" {
		req.Type = models.ChannelTypeText
	}

	channel, err := h.channelService.CreateChannel(
		c.Context(),
		serverID,
		requesterID,
		req.Name,
		req.Type,
		req.ParentID,
	)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(channel)
}

// CreateInvite creates a new invite for this server (convenience endpoint)
func (h *ServerHandler) CreateInvite(c *fiber.Ctx) error {
	requesterID := c.Locals("userID").(uuid.UUID)
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	var req struct {
		MaxAge  int `json:"max_age"`  // seconds, 0 = never
		MaxUses int `json:"max_uses"` // 0 = unlimited
	}

	_ = c.BodyParser(&req)

	// Get default channel for invite
	channels, err := h.serverService.GetChannels(c.Context(), serverID)
	if err != nil || len(channels) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "no channels in server",
		})
	}

	channelID := channels[0].ID // Use first channel

	var expiresIn *time.Duration
	if req.MaxAge > 0 {
		d := time.Duration(req.MaxAge) * time.Second
		expiresIn = &d
	}

	invite, err := h.serverService.CreateInvite(c.Context(), serverID, channelID, requesterID, req.MaxUses, expiresIn)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(invite)
}
