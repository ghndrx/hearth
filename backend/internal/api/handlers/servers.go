package handlers

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// getUserIDFromContext safely extracts userID from Fiber context
func getUserIDFromContext(c *fiber.Ctx) (uuid.UUID, error) {
	userIDVal := c.Locals("userID")
	if userIDVal == nil {
		return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "invalid user id")
	}
	return userID, nil
}

type ServerHandler struct {
	serverService  *services.ServerService
	channelService *services.ChannelService
	roleService    *services.RoleService
}

// NewServerHandler creates a new server handler
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
// @Summary Create a new server
// @Description Creates a new server with the given name and optional icon
// @Tags Servers
// @Accept json
// @Produce json
// @Param body body struct{Name string `json:"name" validate:"required,min=2,max=100"`; Icon string `json:"icon"`} true "Server creation data"
// @Success 201 {object} models.Server "Server created successfully"
// @Failure 400 {object} fiber.Map "Invalid request body or validation error"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Maximum servers owned limit reached"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers [post]
func (h *ServerHandler) Create(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	var req struct {
		Name string `json:"name" validate:"required,min=2,max=100"`
		Icon string `json:"icon"`
	}

	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	if req.Name == "" || len(req.Name) < 2 {
		return ValidationError(c, "name", "must be at least 2 characters")
	}

	server, err := h.serverService.CreateServer(c.Context(), userID, req.Name, req.Icon)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(server)
}

// Get returns a server by ID
// @Summary Get server by ID
// @Description Returns a server by its unique identifier
// @Tags Servers
// @Produce json
// @Param id path string true "Server ID"
// @Success 200 {object} models.Server "Server found"
// @Failure 400 {object} fiber.Map "Invalid server ID"
// @Failure 404 {object} fiber.Map "Server not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id} [get]
func (h *ServerHandler) Get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return InvalidUUID(c, "server id")
	}

	server, err := h.serverService.GetServer(c.Context(), id)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(server)
}

// Update updates a server
// @Summary Update server
// @Description Updates server information such as name, icon, banner, or description
// @Tags Servers
// @Accept json
// @Produce json
// @Param id path string true "Server ID"
// @Param body body struct{Name *string `json:"name"`; Icon *string `json:"icon"`; Banner *string `json:"banner"`; Description *string `json:"description"`} true "Server update data"
// @Success 200 {object} models.Server "Server updated successfully"
// @Failure 400 {object} fiber.Map "Invalid server ID or request body"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Not a member of this server"
// @Failure 404 {object} fiber.Map "Server not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id} [patch]
func (h *ServerHandler) Update(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return InvalidUUID(c, "server id")
	}

	var req struct {
		Name        *string `json:"name"`
		Icon        *string `json:"icon"`
		Banner      *string `json:"banner"`
		Description *string `json:"description"`
	}

	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	updates := &models.ServerUpdate{
		Name:        req.Name,
		IconURL:     req.Icon,
		BannerURL:   req.Banner,
		Description: req.Description,
	}

	server, err := h.serverService.UpdateServer(c.Context(), id, userID, updates)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(server)
}

// Delete deletes a server
// @Summary Delete server
// @Description Deletes a server permanently. Only the server owner can delete.
// @Tags Servers
// @Param id path string true "Server ID"
// @Success 204 "Server deleted successfully"
// @Failure 400 {object} fiber.Map "Invalid server ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Only server owner can delete"
// @Failure 404 {object} fiber.Map "Server not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id} [delete]
func (h *ServerHandler) Delete(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return InvalidUUID(c, "server id")
	}

	if err := h.serverService.DeleteServer(c.Context(), id, userID); err != nil {
		return HandleServiceError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// TransferOwnership transfers server ownership to another member
// @Summary Transfer server ownership
// @Description Transfers ownership of a server to another member. Only the current owner can transfer.
// @Tags Servers
// @Accept json
// @Produce json
// @Param id path string true "Server ID"
// @Param body body struct{NewOwnerID string `json:"new_owner_id"`} true "New owner ID"
// @Success 200 {object} models.Server "Ownership transferred successfully"
// @Failure 400 {object} fiber.Map "Invalid server ID, user ID, or cannot transfer to self"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Only server owner can transfer ownership"
// @Failure 404 {object} fiber.Map "Server not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/transfer [post]
func (h *ServerHandler) TransferOwnership(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return InvalidUUID(c, "server id")
	}

	var req struct {
		NewOwnerID string `json:"new_owner_id"`
	}

	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	newOwnerID, err := uuid.Parse(req.NewOwnerID)
	if err != nil {
		return InvalidUUID(c, "new_owner_id")
	}

	server, err := h.serverService.TransferOwnership(c.Context(), id, userID, newOwnerID)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(server)
}

// GetMembers returns server members
// @Summary Get server members
// @Description Returns a paginated list of members in a server
// @Tags Servers
// @Produce json
// @Param id path string true "Server ID"
// @Param limit query int false "Number of members to return (default 100, max 1000)"
// @Param offset query int false "Offset for pagination (default 0)"
// @Success 200 {array} models.ServerMember "List of server members"
// @Failure 400 {object} fiber.Map "Invalid server ID"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/members [get]
func (h *ServerHandler) GetMembers(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	// Parse pagination with defaults
	limit, err := strconv.Atoi(c.Query("limit", "100"))
	if err != nil || limit < 1 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	offset, err := strconv.Atoi(c.Query("offset", "0"))
	if err != nil || offset < 0 {
		offset = 0
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
// @Summary Get specific member
// @Description Returns a specific member's information in a server
// @Tags Servers
// @Produce json
// @Param id path string true "Server ID"
// @Param userId path string true "User ID"
// @Success 200 {object} models.ServerMember "Member found"
// @Failure 400 {object} fiber.Map "Invalid server ID or user ID"
// @Failure 404 {object} fiber.Map "Member not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/members/{userId} [get]
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
// @Summary Update member
// @Description Updates a member's nickname or roles in a server
// @Tags Servers
// @Accept json
// @Produce json
// @Param id path string true "Server ID"
// @Param userId path string true "User ID"
// @Param body body struct{Nickname *string `json:"nick"`; Roles []uuid.UUID `json:"roles"`} true "Member update data"
// @Success 200 {object} models.ServerMember "Member updated successfully"
// @Failure 400 {object} fiber.Map "Invalid server ID, user ID, or request body"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 404 {object} fiber.Map "Member not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/members/{userId} [patch]
func (h *ServerHandler) UpdateMember(c *fiber.Ctx) error {
	requesterID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}
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
// @Summary Remove/kick member
// @Description Kicks a member from the server
// @Tags Servers
// @Param id path string true "Server ID"
// @Param userId path string true "User ID"
// @Param reason query string false "Kick reason"
// @Success 204 "Member kicked successfully"
// @Failure 400 {object} fiber.Map "Invalid server ID or user ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Insufficient permissions"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/members/{userId} [delete]
func (h *ServerHandler) RemoveMember(c *fiber.Ctx) error {
	requesterID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}
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
// @Summary Leave server
// @Description Allows the current user to leave a server. Owners must transfer ownership first.
// @Tags Servers
// @Param id path string true "Server ID"
// @Success 204 "Left server successfully"
// @Failure 400 {object} fiber.Map "Invalid server ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Owner cannot leave server without transferring ownership"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/leave [delete]
func (h *ServerHandler) Leave(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}
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
// @Summary Get server bans
// @Description Returns a list of banned users for a server
// @Tags Servers
// @Produce json
// @Param id path string true "Server ID"
// @Success 200 {array} models.Ban "List of bans"
// @Failure 400 {object} fiber.Map "Invalid server ID"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/bans [get]
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
// @Summary Ban member
// @Description Bans a user from the server
// @Tags Servers
// @Accept json
// @Param id path string true "Server ID"
// @Param userId path string true "User ID"
// @Param body body struct{Reason string `json:"reason"`; DeleteMessageDays int `json:"delete_message_days"`; DeleteMessageSeconds int `json:"delete_message_seconds"`} false "Ban options"
// @Success 204 "User banned successfully"
// @Failure 400 {object} fiber.Map "Invalid server ID or user ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Insufficient permissions"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/bans/{userId} [put]
func (h *ServerHandler) CreateBan(c *fiber.Ctx) error {
	requesterID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}
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
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

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
// @Summary Unban member
// @Description Removes a ban from a user, allowing them to rejoin the server
// @Tags Servers
// @Param id path string true "Server ID"
// @Param userId path string true "User ID"
// @Success 204 "User unbanned successfully"
// @Failure 400 {object} fiber.Map "Invalid server ID or user ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/bans/{userId} [delete]
func (h *ServerHandler) RemoveBan(c *fiber.Ctx) error {
	requesterID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}
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
// @Summary Get server invites
// @Description Returns a list of active invites for a server
// @Tags Servers
// @Produce json
// @Param id path string true "Server ID"
// @Success 200 {array} models.Invite "List of invites"
// @Failure 400 {object} fiber.Map "Invalid server ID"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/invites [get]
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

// SetVanityURL sets a vanity invite URL for a server
func (h *ServerHandler) SetVanityURL(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	var req struct {
		VanityCode string `json:"vanity_code"`
		ChannelID  string `json:"channel_id"`
	}
	if err := c.BodyParser(&req); err != nil || req.VanityCode == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "vanity_code is required",
		})
	}

	channelID := uuid.Nil
	if req.ChannelID != "" {
		var parseErr error
		channelID, parseErr = uuid.Parse(req.ChannelID)
		if parseErr != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid channel_id",
			})
		}
	}

	invite, err := h.serverService.CreateVanityInvite(c.Context(), serverID, channelID, userID, req.VanityCode)
	if err != nil {
		status := fiber.StatusInternalServerError
		if err == services.ErrVanityCodeTaken || err == services.ErrVanityCodeInvalid {
			status = fiber.StatusBadRequest
		} else if err == services.ErrNotServerMember || err == services.ErrMissingManageServer {
			status = fiber.StatusForbidden
		}
		return c.Status(status).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(invite)
}

// GetInviteAnalytics returns analytics for all invites in a server
func (h *ServerHandler) GetInviteAnalytics(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	analytics, err := h.serverService.GetServerInviteAnalytics(c.Context(), serverID, userID)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if analytics == nil {
		analytics = []models.InviteAnalytics{}
	}
	return c.JSON(analytics)
}

// GetRoles returns server roles
// @Summary Get server roles
// @Description Returns a list of roles in a server
// @Tags Servers
// @Produce json
// @Param id path string true "Server ID"
// @Success 200 {array} models.Role "List of roles"
// @Failure 400 {object} fiber.Map "Invalid server ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/roles [get]
func (h *ServerHandler) GetRoles(c *fiber.Ctx) error {
	requesterID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}
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
// @Summary Create server role
// @Description Creates a new role in a server
// @Tags Servers
// @Accept json
// @Produce json
// @Param id path string true "Server ID"
// @Param body body struct{Name string `json:"name"`; Color int `json:"color"`; Permissions int64 `json:"permissions"`} true "Role creation data"
// @Success 201 {object} models.Role "Role created successfully"
// @Failure 400 {object} fiber.Map "Invalid server ID or request body"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Insufficient permissions"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/roles [post]
func (h *ServerHandler) CreateRole(c *fiber.Ctx) error {
	requesterID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}
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
// @Summary Update server role
// @Description Updates an existing role's properties
// @Tags Servers
// @Accept json
// @Produce json
// @Param id path string true "Server ID"
// @Param roleId path string true "Role ID"
// @Param body body struct{Name *string `json:"name"`; Color *int `json:"color"`; Hoist *bool `json:"hoist"`; Permissions *int64 `json:"permissions"`; Mentionable *bool `json:"mentionable"`; Position *int `json:"position"`} true "Role update data"
// @Success 200 {object} models.Role "Role updated successfully"
// @Failure 400 {object} fiber.Map "Invalid server ID, role ID, or request body"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Insufficient permissions"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/roles/{roleId} [patch]
func (h *ServerHandler) UpdateRole(c *fiber.Ctx) error {
	requesterID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}
	_, err = uuid.Parse(c.Params("id"))
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
// @Summary Delete server role
// @Description Deletes a role from a server
// @Tags Servers
// @Param id path string true "Server ID"
// @Param roleId path string true "Role ID"
// @Success 204 "Role deleted successfully"
// @Failure 400 {object} fiber.Map "Invalid server ID or role ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Insufficient permissions"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/roles/{roleId} [delete]
func (h *ServerHandler) DeleteRole(c *fiber.Ctx) error {
	requesterID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}
	_, err = uuid.Parse(c.Params("id"))
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

// BatchUpdateRolesPositions updates the positions of multiple roles (reorder)
// @Summary Reorder roles
// @Description Updates the position/order of multiple roles in a server
// @Tags Servers
// @Accept json
// @Produce json
// @Param id path string true "Server ID"
// @Param body body struct{Positions []struct{ID string `json:"id"`; Position int `json:"position"`} `json:"positions"`} true "Role positions"
// @Success 200 {array} models.Role "Updated roles in new order"
// @Failure 400 {object} fiber.Map "Invalid server ID or request body"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Insufficient permissions"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/roles [patch]
func (h *ServerHandler) BatchUpdateRolesPositions(c *fiber.Ctx) error {
	requesterID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	var req struct {
		Positions []struct {
			ID       string `json:"id"`
			Position int    `json:"position"`
		} `json:"positions"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if len(req.Positions) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "no positions provided",
		})
	}

	positions := make(map[uuid.UUID]int)
	for _, p := range req.Positions {
		roleID, err := uuid.Parse(p.ID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid role id: " + p.ID,
			})
		}
		positions[roleID] = p.Position
	}

	if err := h.roleService.UpdateRolePositions(c.Context(), serverID, requesterID, positions); err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": err.Error(),
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

// GetChannels returns server channels
// @Summary Get server channels
// @Description Returns a list of channels in a server
// @Tags Servers
// @Produce json
// @Param id path string true "Server ID"
// @Success 200 {array} models.Channel "List of channels"
// @Failure 400 {object} fiber.Map "Invalid server ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/channels [get]
func (h *ServerHandler) GetChannels(c *fiber.Ctx) error {
	requesterID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}
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
// @Summary Create server channel
// @Description Creates a new channel in a server
// @Tags Servers
// @Accept json
// @Produce json
// @Param id path string true "Server ID"
// @Param body body struct{Name string `json:"name"`; Type models.ChannelType `json:"type"`; ParentID *uuid.UUID `json:"parent_id"`} true "Channel creation data"
// @Success 201 {object} models.Channel "Channel created successfully"
// @Failure 400 {object} fiber.Map "Invalid server ID or request body"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Insufficient permissions"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/channels [post]
func (h *ServerHandler) CreateChannel(c *fiber.Ctx) error {
	requesterID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}
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
// @Summary Create server invite
// @Description Creates a new invite for the server (uses first available channel)
// @Tags Servers
// @Accept json
// @Produce json
// @Param id path string true "Server ID"
// @Param body body struct{MaxAge int `json:"max_age"`; MaxUses int `json:"max_uses"`} false "Invite options"
// @Success 201 {object} models.Invite "Invite created successfully"
// @Failure 400 {object} fiber.Map "Invalid server ID or no channels in server"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Insufficient permissions"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/invites [post]
func (h *ServerHandler) CreateInvite(c *fiber.Ctx) error {
	requesterID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}
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

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

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
