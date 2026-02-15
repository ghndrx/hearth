package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// Mock ServerService
type mockServerService struct {
	createServerFunc func(ctx context.Context, ownerID uuid.UUID, name, icon string) (*models.Server, error)
	getServerFunc    func(ctx context.Context, id uuid.UUID) (*models.Server, error)
	updateServerFunc func(ctx context.Context, id, requesterID uuid.UUID, updates *models.ServerUpdate) (*models.Server, error)
	deleteServerFunc func(ctx context.Context, id, requesterID uuid.UUID) error
	getMembersFunc   func(ctx context.Context, serverID uuid.UUID, limit, offset int) ([]*models.Member, error)
	getMemberFunc    func(ctx context.Context, serverID, userID uuid.UUID) (*models.Member, error)
	updateMemberFunc func(ctx context.Context, serverID, requesterID, targetID uuid.UUID, nickname *string, roles []uuid.UUID) (*models.Member, error)
	kickMemberFunc   func(ctx context.Context, serverID, requesterID, targetID uuid.UUID, reason string) error
	leaveServerFunc  func(ctx context.Context, serverID, userID uuid.UUID) error
	getBansFunc      func(ctx context.Context, serverID uuid.UUID) ([]*models.Ban, error)
	banMemberFunc    func(ctx context.Context, serverID, requesterID, targetID uuid.UUID, reason string, deleteDays int) error
	unbanMemberFunc  func(ctx context.Context, serverID, requesterID, targetID uuid.UUID) error
	getInvitesFunc   func(ctx context.Context, serverID uuid.UUID) ([]*models.Invite, error)
	getChannelsFunc  func(ctx context.Context, serverID uuid.UUID) ([]*models.Channel, error)
	createInviteFunc func(ctx context.Context, serverID, channelID, creatorID uuid.UUID, maxUses int, expiresIn *time.Duration) (*models.Invite, error)
}

func (m *mockServerService) CreateServer(ctx context.Context, ownerID uuid.UUID, name, icon string) (*models.Server, error) {
	if m.createServerFunc != nil {
		return m.createServerFunc(ctx, ownerID, name, icon)
	}
	return nil, nil
}

func (m *mockServerService) GetServer(ctx context.Context, id uuid.UUID) (*models.Server, error) {
	if m.getServerFunc != nil {
		return m.getServerFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockServerService) UpdateServer(ctx context.Context, id, requesterID uuid.UUID, updates *models.ServerUpdate) (*models.Server, error) {
	if m.updateServerFunc != nil {
		return m.updateServerFunc(ctx, id, requesterID, updates)
	}
	return nil, nil
}

func (m *mockServerService) DeleteServer(ctx context.Context, id, requesterID uuid.UUID) error {
	if m.deleteServerFunc != nil {
		return m.deleteServerFunc(ctx, id, requesterID)
	}
	return nil
}

func (m *mockServerService) GetMembers(ctx context.Context, serverID uuid.UUID, limit, offset int) ([]*models.Member, error) {
	if m.getMembersFunc != nil {
		return m.getMembersFunc(ctx, serverID, limit, offset)
	}
	return nil, nil
}

func (m *mockServerService) GetMember(ctx context.Context, serverID, userID uuid.UUID) (*models.Member, error) {
	if m.getMemberFunc != nil {
		return m.getMemberFunc(ctx, serverID, userID)
	}
	return nil, nil
}

func (m *mockServerService) UpdateMember(ctx context.Context, serverID, requesterID, targetID uuid.UUID, nickname *string, roles []uuid.UUID) (*models.Member, error) {
	if m.updateMemberFunc != nil {
		return m.updateMemberFunc(ctx, serverID, requesterID, targetID, nickname, roles)
	}
	return nil, nil
}

func (m *mockServerService) KickMember(ctx context.Context, serverID, requesterID, targetID uuid.UUID, reason string) error {
	if m.kickMemberFunc != nil {
		return m.kickMemberFunc(ctx, serverID, requesterID, targetID, reason)
	}
	return nil
}

func (m *mockServerService) LeaveServer(ctx context.Context, serverID, userID uuid.UUID) error {
	if m.leaveServerFunc != nil {
		return m.leaveServerFunc(ctx, serverID, userID)
	}
	return nil
}

func (m *mockServerService) GetBans(ctx context.Context, serverID uuid.UUID) ([]*models.Ban, error) {
	if m.getBansFunc != nil {
		return m.getBansFunc(ctx, serverID)
	}
	return nil, nil
}

func (m *mockServerService) BanMember(ctx context.Context, serverID, requesterID, targetID uuid.UUID, reason string, deleteDays int) error {
	if m.banMemberFunc != nil {
		return m.banMemberFunc(ctx, serverID, requesterID, targetID, reason, deleteDays)
	}
	return nil
}

func (m *mockServerService) UnbanMember(ctx context.Context, serverID, requesterID, targetID uuid.UUID) error {
	if m.unbanMemberFunc != nil {
		return m.unbanMemberFunc(ctx, serverID, requesterID, targetID)
	}
	return nil
}

func (m *mockServerService) GetInvites(ctx context.Context, serverID uuid.UUID) ([]*models.Invite, error) {
	if m.getInvitesFunc != nil {
		return m.getInvitesFunc(ctx, serverID)
	}
	return nil, nil
}

func (m *mockServerService) GetChannels(ctx context.Context, serverID uuid.UUID) ([]*models.Channel, error) {
	if m.getChannelsFunc != nil {
		return m.getChannelsFunc(ctx, serverID)
	}
	return nil, nil
}

func (m *mockServerService) CreateInvite(ctx context.Context, serverID, channelID, creatorID uuid.UUID, maxUses int, expiresIn *time.Duration) (*models.Invite, error) {
	if m.createInviteFunc != nil {
		return m.createInviteFunc(ctx, serverID, channelID, creatorID, maxUses, expiresIn)
	}
	return nil, nil
}

// Mock ChannelService
type mockChannelService struct {
	getServerChannelsFunc func(ctx context.Context, serverID, requesterID uuid.UUID) ([]*models.Channel, error)
	createChannelFunc     func(ctx context.Context, serverID, creatorID uuid.UUID, name string, channelType models.ChannelType, parentID *uuid.UUID) (*models.Channel, error)
}

func (m *mockChannelService) GetServerChannels(ctx context.Context, serverID, requesterID uuid.UUID) ([]*models.Channel, error) {
	if m.getServerChannelsFunc != nil {
		return m.getServerChannelsFunc(ctx, serverID, requesterID)
	}
	return nil, nil
}

func (m *mockChannelService) CreateChannel(ctx context.Context, serverID, creatorID uuid.UUID, name string, channelType models.ChannelType, parentID *uuid.UUID) (*models.Channel, error) {
	if m.createChannelFunc != nil {
		return m.createChannelFunc(ctx, serverID, creatorID, name, channelType, parentID)
	}
	return nil, nil
}

// Mock RoleService
type mockRoleService struct {
	getServerRolesFunc func(ctx context.Context, serverID, requesterID uuid.UUID) ([]*models.Role, error)
	createRoleFunc     func(ctx context.Context, serverID, creatorID uuid.UUID, name string, color int, permissions int64) (*models.Role, error)
	updateRoleFunc     func(ctx context.Context, roleID, requesterID uuid.UUID, updates *models.RoleUpdate) (*models.Role, error)
	deleteRoleFunc     func(ctx context.Context, roleID, requesterID uuid.UUID) error
}

func (m *mockRoleService) GetServerRoles(ctx context.Context, serverID, requesterID uuid.UUID) ([]*models.Role, error) {
	if m.getServerRolesFunc != nil {
		return m.getServerRolesFunc(ctx, serverID, requesterID)
	}
	return nil, nil
}

func (m *mockRoleService) CreateRole(ctx context.Context, serverID, creatorID uuid.UUID, name string, color int, permissions int64) (*models.Role, error) {
	if m.createRoleFunc != nil {
		return m.createRoleFunc(ctx, serverID, creatorID, name, color, permissions)
	}
	return nil, nil
}

func (m *mockRoleService) UpdateRole(ctx context.Context, roleID, requesterID uuid.UUID, updates *models.RoleUpdate) (*models.Role, error) {
	if m.updateRoleFunc != nil {
		return m.updateRoleFunc(ctx, roleID, requesterID, updates)
	}
	return nil, nil
}

func (m *mockRoleService) DeleteRole(ctx context.Context, roleID, requesterID uuid.UUID) error {
	if m.deleteRoleFunc != nil {
		return m.deleteRoleFunc(ctx, roleID, requesterID)
	}
	return nil
}

// Test helper to create a Fiber app with server handler
func setupServerTestApp(serverSvc *mockServerService, channelSvc *mockChannelService, roleSvc *mockRoleService, userID uuid.UUID) *fiber.App {
	app := fiber.New()

	// Middleware to set userID
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", userID)
		return c.Next()
	})

	// Create handler with wrapped services
	h := &testServerHandler{
		serverSvc:  serverSvc,
		channelSvc: channelSvc,
		roleSvc:    roleSvc,
	}

	// Setup routes
	servers := app.Group("/servers")
	servers.Post("/", h.Create)
	servers.Get("/:id", h.Get)
	servers.Patch("/:id", h.Update)
	servers.Delete("/:id", h.Delete)
	servers.Get("/:id/members", h.GetMembers)
	// @me route must come before :userId to avoid being matched as a UUID
	servers.Delete("/:id/members/@me", h.Leave)
	servers.Get("/:id/members/:userId", h.GetMember)
	servers.Patch("/:id/members/:userId", h.UpdateMember)
	servers.Delete("/:id/members/:userId", h.RemoveMember)
	servers.Get("/:id/bans", h.GetBans)
	servers.Put("/:id/bans/:userId", h.CreateBan)
	servers.Delete("/:id/bans/:userId", h.RemoveBan)
	servers.Get("/:id/invites", h.GetInvites)
	servers.Get("/:id/roles", h.GetRoles)
	servers.Post("/:id/roles", h.CreateRole)
	servers.Patch("/:id/roles/:roleId", h.UpdateRole)
	servers.Delete("/:id/roles/:roleId", h.DeleteRole)
	servers.Get("/:id/channels", h.GetChannels)
	servers.Post("/:id/channels", h.CreateChannel)

	return app
}

// testServerHandler wraps mock services directly for testing
type testServerHandler struct {
	serverSvc  *mockServerService
	channelSvc *mockChannelService
	roleSvc    *mockRoleService
}

func (h *testServerHandler) Create(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	var req struct {
		Name string `json:"name"`
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

	server, err := h.serverSvc.CreateServer(c.Context(), userID, req.Name, req.Icon)
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

func (h *testServerHandler) Get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	server, err := h.serverSvc.GetServer(c.Context(), id)
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

func (h *testServerHandler) Update(c *fiber.Ctx) error {
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

	server, err := h.serverSvc.UpdateServer(c.Context(), id, userID, updates)
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

func (h *testServerHandler) Delete(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	if err := h.serverSvc.DeleteServer(c.Context(), id, userID); err != nil {
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

func (h *testServerHandler) GetMembers(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	members, err := h.serverSvc.GetMembers(c.Context(), id, 100, 0)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(members)
}

func (h *testServerHandler) GetMember(c *fiber.Ctx) error {
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

	member, err := h.serverSvc.GetMember(c.Context(), serverID, userID)
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

func (h *testServerHandler) UpdateMember(c *fiber.Ctx) error {
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
	_ = c.BodyParser(&req)

	member, err := h.serverSvc.UpdateMember(c.Context(), serverID, requesterID, targetID, req.Nickname, req.Roles)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(member)
}

func (h *testServerHandler) RemoveMember(c *fiber.Ctx) error {
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

	if err := h.serverSvc.KickMember(c.Context(), serverID, requesterID, targetID, ""); err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *testServerHandler) Leave(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	if err := h.serverSvc.LeaveServer(c.Context(), id, userID); err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *testServerHandler) GetBans(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	bans, err := h.serverSvc.GetBans(c.Context(), serverID)
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

func (h *testServerHandler) CreateBan(c *fiber.Ctx) error {
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
		Reason            string `json:"reason"`
		DeleteMessageDays int    `json:"delete_message_days"`
	}
	_ = c.BodyParser(&req)

	if err := h.serverSvc.BanMember(c.Context(), serverID, requesterID, targetID, req.Reason, req.DeleteMessageDays); err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *testServerHandler) RemoveBan(c *fiber.Ctx) error {
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

	if err := h.serverSvc.UnbanMember(c.Context(), serverID, requesterID, targetID); err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *testServerHandler) GetInvites(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	invites, err := h.serverSvc.GetInvites(c.Context(), serverID)
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

func (h *testServerHandler) GetRoles(c *fiber.Ctx) error {
	requesterID := c.Locals("userID").(uuid.UUID)
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	roles, err := h.roleSvc.GetServerRoles(c.Context(), serverID, requesterID)
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

func (h *testServerHandler) CreateRole(c *fiber.Ctx) error {
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
	_ = c.BodyParser(&req)

	if req.Name == "" {
		req.Name = "new role"
	}

	role, err := h.roleSvc.CreateRole(c.Context(), serverID, requesterID, req.Name, req.Color, req.Permissions)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(role)
}

func (h *testServerHandler) UpdateRole(c *fiber.Ctx) error {
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
		Permissions *int64  `json:"permissions"`
	}
	_ = c.BodyParser(&req)

	updates := &models.RoleUpdate{
		Name:        req.Name,
		Color:       req.Color,
		Permissions: req.Permissions,
	}

	role, err := h.roleSvc.UpdateRole(c.Context(), roleID, requesterID, updates)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(role)
}

func (h *testServerHandler) DeleteRole(c *fiber.Ctx) error {
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

	if err := h.roleSvc.DeleteRole(c.Context(), roleID, requesterID); err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *testServerHandler) GetChannels(c *fiber.Ctx) error {
	requesterID := c.Locals("userID").(uuid.UUID)
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	channels, err := h.channelSvc.GetServerChannels(c.Context(), serverID, requesterID)
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

func (h *testServerHandler) CreateChannel(c *fiber.Ctx) error {
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

	if req.Type == "" {
		req.Type = models.ChannelTypeText
	}

	channel, err := h.channelSvc.CreateChannel(c.Context(), serverID, requesterID, req.Name, req.Type, req.ParentID)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(channel)
}

// Tests

func TestServerHandler_Create(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()

	tests := []struct {
		name           string
		body           map[string]interface{}
		mockSetup      func(*mockServerService)
		expectedStatus int
		checkBody      func(*testing.T, map[string]interface{})
	}{
		{
			name: "successful server creation",
			body: map[string]interface{}{
				"name": "My Server",
				"icon": "https://example.com/icon.png",
			},
			mockSetup: func(m *mockServerService) {
				m.createServerFunc = func(ctx context.Context, ownerID uuid.UUID, name, icon string) (*models.Server, error) {
					return &models.Server{
						ID:      serverID,
						Name:    name,
						OwnerID: ownerID,
					}, nil
				}
			},
			expectedStatus: fiber.StatusCreated,
			checkBody: func(t *testing.T, body map[string]interface{}) {
				if body["name"] != "My Server" {
					t.Errorf("expected name 'My Server', got %v", body["name"])
				}
			},
		},
		{
			name: "name too short",
			body: map[string]interface{}{
				"name": "A",
			},
			mockSetup:      func(m *mockServerService) {},
			expectedStatus: fiber.StatusBadRequest,
			checkBody: func(t *testing.T, body map[string]interface{}) {
				if body["error"] == nil {
					t.Error("expected error in body")
				}
			},
		},
		{
			name: "empty name",
			body: map[string]interface{}{
				"name": "",
			},
			mockSetup:      func(m *mockServerService) {},
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name: "max servers reached",
			body: map[string]interface{}{
				"name": "Another Server",
			},
			mockSetup: func(m *mockServerService) {
				m.createServerFunc = func(ctx context.Context, ownerID uuid.UUID, name, icon string) (*models.Server, error) {
					return nil, services.ErrMaxServersReached
				}
			},
			expectedStatus: fiber.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serverSvc := &mockServerService{}
			tt.mockSetup(serverSvc)

			app := setupServerTestApp(serverSvc, &mockChannelService{}, &mockRoleService{}, userID)

			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/servers/", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			resp, _ := app.Test(req)

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			if tt.checkBody != nil {
				var body map[string]interface{}
				bodyData, _ := io.ReadAll(resp.Body)
				json.Unmarshal(bodyData, &body)
				tt.checkBody(t, body)
			}
		})
	}
}

func TestServerHandler_Get(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()

	tests := []struct {
		name           string
		serverIDParam  string
		mockSetup      func(*mockServerService)
		expectedStatus int
	}{
		{
			name:          "successful get",
			serverIDParam: serverID.String(),
			mockSetup: func(m *mockServerService) {
				m.getServerFunc = func(ctx context.Context, id uuid.UUID) (*models.Server, error) {
					return &models.Server{
						ID:      serverID,
						Name:    "Test Server",
						OwnerID: userID,
					}, nil
				}
			},
			expectedStatus: fiber.StatusOK,
		},
		{
			name:          "server not found",
			serverIDParam: serverID.String(),
			mockSetup: func(m *mockServerService) {
				m.getServerFunc = func(ctx context.Context, id uuid.UUID) (*models.Server, error) {
					return nil, services.ErrServerNotFound
				}
			},
			expectedStatus: fiber.StatusNotFound,
		},
		{
			name:           "invalid server id",
			serverIDParam:  "not-a-uuid",
			mockSetup:      func(m *mockServerService) {},
			expectedStatus: fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serverSvc := &mockServerService{}
			tt.mockSetup(serverSvc)

			app := setupServerTestApp(serverSvc, &mockChannelService{}, &mockRoleService{}, userID)

			req := httptest.NewRequest("GET", "/servers/"+tt.serverIDParam, nil)
			resp, _ := app.Test(req)

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

func TestServerHandler_Update(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()
	newName := "Updated Server"

	tests := []struct {
		name           string
		serverIDParam  string
		body           map[string]interface{}
		mockSetup      func(*mockServerService)
		expectedStatus int
	}{
		{
			name:          "successful update",
			serverIDParam: serverID.String(),
			body: map[string]interface{}{
				"name": newName,
			},
			mockSetup: func(m *mockServerService) {
				m.updateServerFunc = func(ctx context.Context, id, requesterID uuid.UUID, updates *models.ServerUpdate) (*models.Server, error) {
					return &models.Server{
						ID:      serverID,
						Name:    *updates.Name,
						OwnerID: userID,
					}, nil
				}
			},
			expectedStatus: fiber.StatusOK,
		},
		{
			name:          "not a member",
			serverIDParam: serverID.String(),
			body: map[string]interface{}{
				"name": newName,
			},
			mockSetup: func(m *mockServerService) {
				m.updateServerFunc = func(ctx context.Context, id, requesterID uuid.UUID, updates *models.ServerUpdate) (*models.Server, error) {
					return nil, services.ErrNotServerMember
				}
			},
			expectedStatus: fiber.StatusForbidden,
		},
		{
			name:          "server not found",
			serverIDParam: serverID.String(),
			body: map[string]interface{}{
				"name": newName,
			},
			mockSetup: func(m *mockServerService) {
				m.updateServerFunc = func(ctx context.Context, id, requesterID uuid.UUID, updates *models.ServerUpdate) (*models.Server, error) {
					return nil, services.ErrServerNotFound
				}
			},
			expectedStatus: fiber.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serverSvc := &mockServerService{}
			tt.mockSetup(serverSvc)

			app := setupServerTestApp(serverSvc, &mockChannelService{}, &mockRoleService{}, userID)

			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("PATCH", "/servers/"+tt.serverIDParam, bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			resp, _ := app.Test(req)

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

func TestServerHandler_Delete(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()

	tests := []struct {
		name           string
		serverIDParam  string
		mockSetup      func(*mockServerService)
		expectedStatus int
	}{
		{
			name:          "successful delete",
			serverIDParam: serverID.String(),
			mockSetup: func(m *mockServerService) {
				m.deleteServerFunc = func(ctx context.Context, id, requesterID uuid.UUID) error {
					return nil
				}
			},
			expectedStatus: fiber.StatusNoContent,
		},
		{
			name:          "not owner",
			serverIDParam: serverID.String(),
			mockSetup: func(m *mockServerService) {
				m.deleteServerFunc = func(ctx context.Context, id, requesterID uuid.UUID) error {
					return services.ErrNotServerOwner
				}
			},
			expectedStatus: fiber.StatusForbidden,
		},
		{
			name:          "server not found",
			serverIDParam: serverID.String(),
			mockSetup: func(m *mockServerService) {
				m.deleteServerFunc = func(ctx context.Context, id, requesterID uuid.UUID) error {
					return services.ErrServerNotFound
				}
			},
			expectedStatus: fiber.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serverSvc := &mockServerService{}
			tt.mockSetup(serverSvc)

			app := setupServerTestApp(serverSvc, &mockChannelService{}, &mockRoleService{}, userID)

			req := httptest.NewRequest("DELETE", "/servers/"+tt.serverIDParam, nil)
			resp, _ := app.Test(req)

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

func TestServerHandler_Members(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()
	memberID := uuid.New()

	t.Run("get members", func(t *testing.T) {
		serverSvc := &mockServerService{
			getMembersFunc: func(ctx context.Context, sid uuid.UUID, limit, offset int) ([]*models.Member, error) {
				return []*models.Member{
					{UserID: memberID, ServerID: serverID, JoinedAt: time.Now()},
				}, nil
			},
		}

		app := setupServerTestApp(serverSvc, &mockChannelService{}, &mockRoleService{}, userID)
		req := httptest.NewRequest("GET", "/servers/"+serverID.String()+"/members", nil)
		resp, _ := app.Test(req)

		if resp.StatusCode != fiber.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
	})

	t.Run("get specific member", func(t *testing.T) {
		serverSvc := &mockServerService{
			getMemberFunc: func(ctx context.Context, sid, uid uuid.UUID) (*models.Member, error) {
				return &models.Member{UserID: memberID, ServerID: serverID, JoinedAt: time.Now()}, nil
			},
		}

		app := setupServerTestApp(serverSvc, &mockChannelService{}, &mockRoleService{}, userID)
		req := httptest.NewRequest("GET", "/servers/"+serverID.String()+"/members/"+memberID.String(), nil)
		resp, _ := app.Test(req)

		if resp.StatusCode != fiber.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
	})

	t.Run("member not found", func(t *testing.T) {
		serverSvc := &mockServerService{
			getMemberFunc: func(ctx context.Context, sid, uid uuid.UUID) (*models.Member, error) {
				return nil, services.ErrNotServerMember
			},
		}

		app := setupServerTestApp(serverSvc, &mockChannelService{}, &mockRoleService{}, userID)
		req := httptest.NewRequest("GET", "/servers/"+serverID.String()+"/members/"+memberID.String(), nil)
		resp, _ := app.Test(req)

		if resp.StatusCode != fiber.StatusNotFound {
			t.Errorf("expected 404, got %d", resp.StatusCode)
		}
	})
}

func TestServerHandler_Bans(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()
	targetID := uuid.New()

	t.Run("get bans", func(t *testing.T) {
		serverSvc := &mockServerService{
			getBansFunc: func(ctx context.Context, sid uuid.UUID) ([]*models.Ban, error) {
				return []*models.Ban{}, nil
			},
		}

		app := setupServerTestApp(serverSvc, &mockChannelService{}, &mockRoleService{}, userID)
		req := httptest.NewRequest("GET", "/servers/"+serverID.String()+"/bans", nil)
		resp, _ := app.Test(req)

		if resp.StatusCode != fiber.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
	})

	t.Run("create ban", func(t *testing.T) {
		serverSvc := &mockServerService{
			banMemberFunc: func(ctx context.Context, sid, rid, tid uuid.UUID, reason string, deleteDays int) error {
				return nil
			},
		}

		app := setupServerTestApp(serverSvc, &mockChannelService{}, &mockRoleService{}, userID)
		bodyBytes, _ := json.Marshal(map[string]interface{}{
			"reason": "Spam",
		})
		req := httptest.NewRequest("PUT", "/servers/"+serverID.String()+"/bans/"+targetID.String(), bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := app.Test(req)

		if resp.StatusCode != fiber.StatusNoContent {
			t.Errorf("expected 204, got %d", resp.StatusCode)
		}
	})

	t.Run("remove ban", func(t *testing.T) {
		serverSvc := &mockServerService{
			unbanMemberFunc: func(ctx context.Context, sid, rid, tid uuid.UUID) error {
				return nil
			},
		}

		app := setupServerTestApp(serverSvc, &mockChannelService{}, &mockRoleService{}, userID)
		req := httptest.NewRequest("DELETE", "/servers/"+serverID.String()+"/bans/"+targetID.String(), nil)
		resp, _ := app.Test(req)

		if resp.StatusCode != fiber.StatusNoContent {
			t.Errorf("expected 204, got %d", resp.StatusCode)
		}
	})
}

func TestServerHandler_Roles(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()
	roleID := uuid.New()

	t.Run("get roles", func(t *testing.T) {
		roleSvc := &mockRoleService{
			getServerRolesFunc: func(ctx context.Context, sid, rid uuid.UUID) ([]*models.Role, error) {
				return []*models.Role{
					{ID: roleID, ServerID: serverID, Name: "@everyone"},
				}, nil
			},
		}

		app := setupServerTestApp(&mockServerService{}, &mockChannelService{}, roleSvc, userID)
		req := httptest.NewRequest("GET", "/servers/"+serverID.String()+"/roles", nil)
		resp, _ := app.Test(req)

		if resp.StatusCode != fiber.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
	})

	t.Run("create role", func(t *testing.T) {
		roleSvc := &mockRoleService{
			createRoleFunc: func(ctx context.Context, sid, cid uuid.UUID, name string, color int, perms int64) (*models.Role, error) {
				return &models.Role{
					ID:          uuid.New(),
					ServerID:    sid,
					Name:        name,
					Color:       color,
					Permissions: perms,
				}, nil
			},
		}

		app := setupServerTestApp(&mockServerService{}, &mockChannelService{}, roleSvc, userID)
		bodyBytes, _ := json.Marshal(map[string]interface{}{
			"name":        "Moderator",
			"color":       0xFF0000,
			"permissions": 0x8,
		})
		req := httptest.NewRequest("POST", "/servers/"+serverID.String()+"/roles", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := app.Test(req)

		if resp.StatusCode != fiber.StatusCreated {
			t.Errorf("expected 201, got %d", resp.StatusCode)
		}
	})

	t.Run("delete role", func(t *testing.T) {
		roleSvc := &mockRoleService{
			deleteRoleFunc: func(ctx context.Context, rid, reqID uuid.UUID) error {
				return nil
			},
		}

		app := setupServerTestApp(&mockServerService{}, &mockChannelService{}, roleSvc, userID)
		req := httptest.NewRequest("DELETE", "/servers/"+serverID.String()+"/roles/"+roleID.String(), nil)
		resp, _ := app.Test(req)

		if resp.StatusCode != fiber.StatusNoContent {
			t.Errorf("expected 204, got %d", resp.StatusCode)
		}
	})
}

func TestServerHandler_Channels(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()

	t.Run("get channels", func(t *testing.T) {
		channelSvc := &mockChannelService{
			getServerChannelsFunc: func(ctx context.Context, sid, rid uuid.UUID) ([]*models.Channel, error) {
				return []*models.Channel{
					{ID: uuid.New(), ServerID: &serverID, Name: "general", Type: models.ChannelTypeText},
				}, nil
			},
		}

		app := setupServerTestApp(&mockServerService{}, channelSvc, &mockRoleService{}, userID)
		req := httptest.NewRequest("GET", "/servers/"+serverID.String()+"/channels", nil)
		resp, _ := app.Test(req)

		if resp.StatusCode != fiber.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
	})

	t.Run("create channel", func(t *testing.T) {
		channelSvc := &mockChannelService{
			createChannelFunc: func(ctx context.Context, sid, cid uuid.UUID, name string, chanType models.ChannelType, parentID *uuid.UUID) (*models.Channel, error) {
				return &models.Channel{
					ID:       uuid.New(),
					ServerID: &sid,
					Name:     name,
					Type:     chanType,
				}, nil
			},
		}

		app := setupServerTestApp(&mockServerService{}, channelSvc, &mockRoleService{}, userID)
		bodyBytes, _ := json.Marshal(map[string]interface{}{
			"name": "announcements",
			"type": "text",
		})
		req := httptest.NewRequest("POST", "/servers/"+serverID.String()+"/channels", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := app.Test(req)

		if resp.StatusCode != fiber.StatusCreated {
			t.Errorf("expected 201, got %d", resp.StatusCode)
		}
	})

	t.Run("create channel without name", func(t *testing.T) {
		channelSvc := &mockChannelService{}

		app := setupServerTestApp(&mockServerService{}, channelSvc, &mockRoleService{}, userID)
		bodyBytes, _ := json.Marshal(map[string]interface{}{
			"type": "text",
		})
		req := httptest.NewRequest("POST", "/servers/"+serverID.String()+"/channels", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := app.Test(req)

		if resp.StatusCode != fiber.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
	})
}

// Additional comprehensive tests for CreateServer

func TestServerHandler_Create_InvalidJSON(t *testing.T) {
	userID := uuid.New()
	serverSvc := &mockServerService{}

	app := setupServerTestApp(serverSvc, &mockChannelService{}, &mockRoleService{}, userID)

	req := httptest.NewRequest("POST", "/servers/", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}

	var body map[string]interface{}
	bodyData, _ := io.ReadAll(resp.Body)
	json.Unmarshal(bodyData, &body)

	if body["error"] != "invalid request body" {
		t.Errorf("expected 'invalid request body' error, got %v", body["error"])
	}
}

func TestServerHandler_Create_WithIcon(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()
	iconURL := "https://example.com/icon.png"

	serverSvc := &mockServerService{
		createServerFunc: func(ctx context.Context, ownerID uuid.UUID, name, icon string) (*models.Server, error) {
			if icon != iconURL {
				t.Errorf("expected icon %s, got %s", iconURL, icon)
			}
			return &models.Server{
				ID:      serverID,
				Name:    name,
				OwnerID: ownerID,
				IconURL: &icon,
			}, nil
		},
	}

	app := setupServerTestApp(serverSvc, &mockChannelService{}, &mockRoleService{}, userID)

	bodyBytes, _ := json.Marshal(map[string]interface{}{
		"name": "My Server",
		"icon": iconURL,
	})
	req := httptest.NewRequest("POST", "/servers/", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}
}

func TestServerHandler_Create_InternalError(t *testing.T) {
	userID := uuid.New()

	serverSvc := &mockServerService{
		createServerFunc: func(ctx context.Context, ownerID uuid.UUID, name, icon string) (*models.Server, error) {
			return nil, services.ErrServerNotFound // Simulating an unexpected error
		},
	}

	app := setupServerTestApp(serverSvc, &mockChannelService{}, &mockRoleService{}, userID)

	bodyBytes, _ := json.Marshal(map[string]interface{}{
		"name": "Test Server",
	})
	req := httptest.NewRequest("POST", "/servers/", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

func TestServerHandler_Create_ExactlyTwoCharacterName(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()

	serverSvc := &mockServerService{
		createServerFunc: func(ctx context.Context, ownerID uuid.UUID, name, icon string) (*models.Server, error) {
			return &models.Server{
				ID:      serverID,
				Name:    name,
				OwnerID: ownerID,
			}, nil
		},
	}

	app := setupServerTestApp(serverSvc, &mockChannelService{}, &mockRoleService{}, userID)

	bodyBytes, _ := json.Marshal(map[string]interface{}{
		"name": "AB", // Exactly 2 characters - should be valid
	})
	req := httptest.NewRequest("POST", "/servers/", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}
}

// Additional comprehensive tests for GetServer

func TestServerHandler_Get_InternalError(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()

	serverSvc := &mockServerService{
		getServerFunc: func(ctx context.Context, id uuid.UUID) (*models.Server, error) {
			return nil, services.ErrNotServerMember // Some internal error
		},
	}

	app := setupServerTestApp(serverSvc, &mockChannelService{}, &mockRoleService{}, userID)

	req := httptest.NewRequest("GET", "/servers/"+serverID.String(), nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

func TestServerHandler_Get_ResponseBody(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()
	description := "Test description"

	serverSvc := &mockServerService{
		getServerFunc: func(ctx context.Context, id uuid.UUID) (*models.Server, error) {
			return &models.Server{
				ID:          serverID,
				Name:        "Test Server",
				OwnerID:     userID,
				Description: &description,
				Features:    []string{"COMMUNITY"},
				CreatedAt:   time.Now(),
			}, nil
		},
	}

	app := setupServerTestApp(serverSvc, &mockChannelService{}, &mockRoleService{}, userID)

	req := httptest.NewRequest("GET", "/servers/"+serverID.String(), nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]interface{}
	bodyData, _ := io.ReadAll(resp.Body)
	json.Unmarshal(bodyData, &body)

	if body["name"] != "Test Server" {
		t.Errorf("expected name 'Test Server', got %v", body["name"])
	}
	if body["description"] != description {
		t.Errorf("expected description '%s', got %v", description, body["description"])
	}
}

// Additional comprehensive tests for UpdateServer

func TestServerHandler_Update_InvalidJSON(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()

	serverSvc := &mockServerService{}

	app := setupServerTestApp(serverSvc, &mockChannelService{}, &mockRoleService{}, userID)

	req := httptest.NewRequest("PATCH", "/servers/"+serverID.String(), bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestServerHandler_Update_InvalidServerID(t *testing.T) {
	userID := uuid.New()

	serverSvc := &mockServerService{}

	app := setupServerTestApp(serverSvc, &mockChannelService{}, &mockRoleService{}, userID)

	bodyBytes, _ := json.Marshal(map[string]interface{}{
		"name": "New Name",
	})
	req := httptest.NewRequest("PATCH", "/servers/not-a-uuid", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestServerHandler_Update_AllFields(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()
	newName := "Updated Server"
	newIcon := "https://example.com/new-icon.png"
	newBanner := "https://example.com/banner.png"
	newDescription := "New description"

	serverSvc := &mockServerService{
		updateServerFunc: func(ctx context.Context, id, requesterID uuid.UUID, updates *models.ServerUpdate) (*models.Server, error) {
			if *updates.Name != newName {
				t.Errorf("expected name %s, got %s", newName, *updates.Name)
			}
			if *updates.IconURL != newIcon {
				t.Errorf("expected icon %s, got %s", newIcon, *updates.IconURL)
			}
			if *updates.BannerURL != newBanner {
				t.Errorf("expected banner %s, got %s", newBanner, *updates.BannerURL)
			}
			if *updates.Description != newDescription {
				t.Errorf("expected description %s, got %s", newDescription, *updates.Description)
			}
			return &models.Server{
				ID:          serverID,
				Name:        *updates.Name,
				OwnerID:     userID,
				IconURL:     updates.IconURL,
				BannerURL:   updates.BannerURL,
				Description: updates.Description,
			}, nil
		},
	}

	app := setupServerTestApp(serverSvc, &mockChannelService{}, &mockRoleService{}, userID)

	bodyBytes, _ := json.Marshal(map[string]interface{}{
		"name":        newName,
		"icon":        newIcon,
		"banner":      newBanner,
		"description": newDescription,
	})
	req := httptest.NewRequest("PATCH", "/servers/"+serverID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestServerHandler_Update_InternalError(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()

	serverSvc := &mockServerService{
		updateServerFunc: func(ctx context.Context, id, requesterID uuid.UUID, updates *models.ServerUpdate) (*models.Server, error) {
			return nil, services.ErrNotServerOwner // Some unexpected error falling to default
		},
	}

	app := setupServerTestApp(serverSvc, &mockChannelService{}, &mockRoleService{}, userID)

	bodyBytes, _ := json.Marshal(map[string]interface{}{
		"name": "New Name",
	})
	req := httptest.NewRequest("PATCH", "/servers/"+serverID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)

	// ErrNotServerOwner is not handled specifically in Update, so it should fall through to InternalServerError
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

// Additional comprehensive tests for DeleteServer

func TestServerHandler_Delete_InvalidServerID(t *testing.T) {
	userID := uuid.New()

	serverSvc := &mockServerService{}

	app := setupServerTestApp(serverSvc, &mockChannelService{}, &mockRoleService{}, userID)

	req := httptest.NewRequest("DELETE", "/servers/not-a-uuid", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestServerHandler_Delete_InternalError(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()

	serverSvc := &mockServerService{
		deleteServerFunc: func(ctx context.Context, id, requesterID uuid.UUID) error {
			return services.ErrNotServerMember // Unexpected error falling to default
		},
	}

	app := setupServerTestApp(serverSvc, &mockChannelService{}, &mockRoleService{}, userID)

	req := httptest.NewRequest("DELETE", "/servers/"+serverID.String(), nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

// Additional comprehensive tests for Members

func TestServerHandler_GetMembers_InvalidServerID(t *testing.T) {
	userID := uuid.New()

	serverSvc := &mockServerService{}

	app := setupServerTestApp(serverSvc, &mockChannelService{}, &mockRoleService{}, userID)

	req := httptest.NewRequest("GET", "/servers/invalid-uuid/members", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestServerHandler_GetMember_InvalidUserID(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()

	serverSvc := &mockServerService{}

	app := setupServerTestApp(serverSvc, &mockChannelService{}, &mockRoleService{}, userID)

	req := httptest.NewRequest("GET", "/servers/"+serverID.String()+"/members/invalid-uuid", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestServerHandler_UpdateMember_Success(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()
	targetID := uuid.New()
	nickname := "NewNick"

	serverSvc := &mockServerService{
		updateMemberFunc: func(ctx context.Context, sid, reqID, tarID uuid.UUID, nick *string, roles []uuid.UUID) (*models.Member, error) {
			if *nick != nickname {
				t.Errorf("expected nickname %s, got %s", nickname, *nick)
			}
			return &models.Member{
				UserID:   tarID,
				ServerID: sid,
				Nickname: nick,
				JoinedAt: time.Now(),
			}, nil
		},
	}

	app := setupServerTestApp(serverSvc, &mockChannelService{}, &mockRoleService{}, userID)

	bodyBytes, _ := json.Marshal(map[string]interface{}{
		"nick": nickname,
	})
	req := httptest.NewRequest("PATCH", "/servers/"+serverID.String()+"/members/"+targetID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestServerHandler_UpdateMember_InvalidServerID(t *testing.T) {
	userID := uuid.New()

	serverSvc := &mockServerService{}

	app := setupServerTestApp(serverSvc, &mockChannelService{}, &mockRoleService{}, userID)

	req := httptest.NewRequest("PATCH", "/servers/invalid-uuid/members/"+uuid.New().String(), nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestServerHandler_UpdateMember_InvalidUserID(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()

	serverSvc := &mockServerService{}

	app := setupServerTestApp(serverSvc, &mockChannelService{}, &mockRoleService{}, userID)

	req := httptest.NewRequest("PATCH", "/servers/"+serverID.String()+"/members/invalid-uuid", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestServerHandler_RemoveMember_Success(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()
	targetID := uuid.New()

	serverSvc := &mockServerService{
		kickMemberFunc: func(ctx context.Context, sid, reqID, tarID uuid.UUID, reason string) error {
			return nil
		},
	}

	app := setupServerTestApp(serverSvc, &mockChannelService{}, &mockRoleService{}, userID)

	req := httptest.NewRequest("DELETE", "/servers/"+serverID.String()+"/members/"+targetID.String(), nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusNoContent {
		t.Errorf("expected 204, got %d", resp.StatusCode)
	}
}

func TestServerHandler_RemoveMember_Forbidden(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()
	targetID := uuid.New()

	serverSvc := &mockServerService{
		kickMemberFunc: func(ctx context.Context, sid, reqID, tarID uuid.UUID, reason string) error {
			return services.ErrNotServerOwner
		},
	}

	app := setupServerTestApp(serverSvc, &mockChannelService{}, &mockRoleService{}, userID)

	req := httptest.NewRequest("DELETE", "/servers/"+serverID.String()+"/members/"+targetID.String(), nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("expected 403, got %d", resp.StatusCode)
	}
}

func TestServerHandler_Leave_Success(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()

	serverSvc := &mockServerService{
		leaveServerFunc: func(ctx context.Context, sid, uid uuid.UUID) error {
			return nil
		},
	}

	app := setupServerTestApp(serverSvc, &mockChannelService{}, &mockRoleService{}, userID)

	req := httptest.NewRequest("DELETE", "/servers/"+serverID.String()+"/members/@me", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusNoContent {
		t.Errorf("expected 204, got %d", resp.StatusCode)
	}
}

func TestServerHandler_Leave_InvalidServerID(t *testing.T) {
	userID := uuid.New()

	serverSvc := &mockServerService{}

	app := setupServerTestApp(serverSvc, &mockChannelService{}, &mockRoleService{}, userID)

	req := httptest.NewRequest("DELETE", "/servers/invalid-uuid/members/@me", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestServerHandler_Leave_Forbidden(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()

	serverSvc := &mockServerService{
		leaveServerFunc: func(ctx context.Context, sid, uid uuid.UUID) error {
			return services.ErrNotServerOwner // Owner cannot leave their own server
		},
	}

	app := setupServerTestApp(serverSvc, &mockChannelService{}, &mockRoleService{}, userID)

	req := httptest.NewRequest("DELETE", "/servers/"+serverID.String()+"/members/@me", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("expected 403, got %d", resp.StatusCode)
	}
}

// Additional comprehensive tests for Bans

func TestServerHandler_GetBans_InvalidServerID(t *testing.T) {
	userID := uuid.New()

	serverSvc := &mockServerService{}

	app := setupServerTestApp(serverSvc, &mockChannelService{}, &mockRoleService{}, userID)

	req := httptest.NewRequest("GET", "/servers/invalid-uuid/bans", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestServerHandler_GetBans_WithBans(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()
	bannedID := uuid.New()
	reason := "Spam"

	serverSvc := &mockServerService{
		getBansFunc: func(ctx context.Context, sid uuid.UUID) ([]*models.Ban, error) {
			return []*models.Ban{
				{
					ServerID: sid,
					UserID:   bannedID,
					Reason:   &reason,
				},
			}, nil
		},
	}

	app := setupServerTestApp(serverSvc, &mockChannelService{}, &mockRoleService{}, userID)

	req := httptest.NewRequest("GET", "/servers/"+serverID.String()+"/bans", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var bans []map[string]interface{}
	bodyData, _ := io.ReadAll(resp.Body)
	json.Unmarshal(bodyData, &bans)

	if len(bans) != 1 {
		t.Errorf("expected 1 ban, got %d", len(bans))
	}
}

func TestServerHandler_CreateBan_InvalidServerID(t *testing.T) {
	userID := uuid.New()

	serverSvc := &mockServerService{}

	app := setupServerTestApp(serverSvc, &mockChannelService{}, &mockRoleService{}, userID)

	req := httptest.NewRequest("PUT", "/servers/invalid-uuid/bans/"+uuid.New().String(), nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestServerHandler_CreateBan_InvalidUserID(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()

	serverSvc := &mockServerService{}

	app := setupServerTestApp(serverSvc, &mockChannelService{}, &mockRoleService{}, userID)

	req := httptest.NewRequest("PUT", "/servers/"+serverID.String()+"/bans/invalid-uuid", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestServerHandler_CreateBan_Forbidden(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()
	targetID := uuid.New()

	serverSvc := &mockServerService{
		banMemberFunc: func(ctx context.Context, sid, rid, tid uuid.UUID, reason string, deleteDays int) error {
			return services.ErrNotServerOwner
		},
	}

	app := setupServerTestApp(serverSvc, &mockChannelService{}, &mockRoleService{}, userID)

	req := httptest.NewRequest("PUT", "/servers/"+serverID.String()+"/bans/"+targetID.String(), nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("expected 403, got %d", resp.StatusCode)
	}
}

func TestServerHandler_RemoveBan_InvalidServerID(t *testing.T) {
	userID := uuid.New()

	serverSvc := &mockServerService{}

	app := setupServerTestApp(serverSvc, &mockChannelService{}, &mockRoleService{}, userID)

	req := httptest.NewRequest("DELETE", "/servers/invalid-uuid/bans/"+uuid.New().String(), nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestServerHandler_RemoveBan_InvalidUserID(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()

	serverSvc := &mockServerService{}

	app := setupServerTestApp(serverSvc, &mockChannelService{}, &mockRoleService{}, userID)

	req := httptest.NewRequest("DELETE", "/servers/"+serverID.String()+"/bans/invalid-uuid", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestServerHandler_RemoveBan_Forbidden(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()
	targetID := uuid.New()

	serverSvc := &mockServerService{
		unbanMemberFunc: func(ctx context.Context, sid, rid, tid uuid.UUID) error {
			return services.ErrNotServerOwner
		},
	}

	app := setupServerTestApp(serverSvc, &mockChannelService{}, &mockRoleService{}, userID)

	req := httptest.NewRequest("DELETE", "/servers/"+serverID.String()+"/bans/"+targetID.String(), nil)
	resp, _ := app.Test(req)

	// RemoveBan returns 500 for any error, as per the handler implementation
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("expected 403, got %d", resp.StatusCode)
	}
}

// Additional comprehensive tests for Invites

func TestServerHandler_GetInvites_InvalidServerID(t *testing.T) {
	userID := uuid.New()

	serverSvc := &mockServerService{}

	app := setupServerTestApp(serverSvc, &mockChannelService{}, &mockRoleService{}, userID)

	req := httptest.NewRequest("GET", "/servers/invalid-uuid/invites", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestServerHandler_GetInvites_Success(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()
	inviteCode := "testcode"

	serverSvc := &mockServerService{
		getInvitesFunc: func(ctx context.Context, sid uuid.UUID) ([]*models.Invite, error) {
			return []*models.Invite{
				{
					Code:     inviteCode,
					ServerID: sid,
				},
			}, nil
		},
	}

	app := setupServerTestApp(serverSvc, &mockChannelService{}, &mockRoleService{}, userID)

	req := httptest.NewRequest("GET", "/servers/"+serverID.String()+"/invites", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

// Additional comprehensive tests for Roles

func TestServerHandler_GetRoles_InvalidServerID(t *testing.T) {
	userID := uuid.New()

	app := setupServerTestApp(&mockServerService{}, &mockChannelService{}, &mockRoleService{}, userID)

	req := httptest.NewRequest("GET", "/servers/invalid-uuid/roles", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestServerHandler_CreateRole_InvalidServerID(t *testing.T) {
	userID := uuid.New()

	app := setupServerTestApp(&mockServerService{}, &mockChannelService{}, &mockRoleService{}, userID)

	bodyBytes, _ := json.Marshal(map[string]interface{}{
		"name": "Moderator",
	})
	req := httptest.NewRequest("POST", "/servers/invalid-uuid/roles", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestServerHandler_CreateRole_DefaultName(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()

	roleSvc := &mockRoleService{
		createRoleFunc: func(ctx context.Context, sid, cid uuid.UUID, name string, color int, perms int64) (*models.Role, error) {
			if name != "new role" {
				t.Errorf("expected default name 'new role', got %s", name)
			}
			return &models.Role{
				ID:       uuid.New(),
				ServerID: sid,
				Name:     name,
			}, nil
		},
	}

	app := setupServerTestApp(&mockServerService{}, &mockChannelService{}, roleSvc, userID)

	bodyBytes, _ := json.Marshal(map[string]interface{}{
		// No name provided - should use default
	})
	req := httptest.NewRequest("POST", "/servers/"+serverID.String()+"/roles", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}
}

func TestServerHandler_CreateRole_Forbidden(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()

	roleSvc := &mockRoleService{
		createRoleFunc: func(ctx context.Context, sid, cid uuid.UUID, name string, color int, perms int64) (*models.Role, error) {
			return nil, services.ErrNotServerOwner
		},
	}

	app := setupServerTestApp(&mockServerService{}, &mockChannelService{}, roleSvc, userID)

	bodyBytes, _ := json.Marshal(map[string]interface{}{
		"name": "Admin",
	})
	req := httptest.NewRequest("POST", "/servers/"+serverID.String()+"/roles", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("expected 403, got %d", resp.StatusCode)
	}
}

func TestServerHandler_UpdateRole_InvalidServerID(t *testing.T) {
	userID := uuid.New()

	app := setupServerTestApp(&mockServerService{}, &mockChannelService{}, &mockRoleService{}, userID)

	bodyBytes, _ := json.Marshal(map[string]interface{}{
		"name": "New Name",
	})
	req := httptest.NewRequest("PATCH", "/servers/invalid-uuid/roles/"+uuid.New().String(), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestServerHandler_UpdateRole_InvalidRoleID(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()

	app := setupServerTestApp(&mockServerService{}, &mockChannelService{}, &mockRoleService{}, userID)

	bodyBytes, _ := json.Marshal(map[string]interface{}{
		"name": "New Name",
	})
	req := httptest.NewRequest("PATCH", "/servers/"+serverID.String()+"/roles/invalid-uuid", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestServerHandler_UpdateRole_Success(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()
	roleID := uuid.New()
	newName := "Updated Role"
	newColor := 0xFF0000

	roleSvc := &mockRoleService{
		updateRoleFunc: func(ctx context.Context, rid, reqID uuid.UUID, updates *models.RoleUpdate) (*models.Role, error) {
			return &models.Role{
				ID:       rid,
				ServerID: serverID,
				Name:     *updates.Name,
				Color:    *updates.Color,
			}, nil
		},
	}

	app := setupServerTestApp(&mockServerService{}, &mockChannelService{}, roleSvc, userID)

	bodyBytes, _ := json.Marshal(map[string]interface{}{
		"name":  newName,
		"color": newColor,
	})
	req := httptest.NewRequest("PATCH", "/servers/"+serverID.String()+"/roles/"+roleID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestServerHandler_UpdateRole_Forbidden(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()
	roleID := uuid.New()

	roleSvc := &mockRoleService{
		updateRoleFunc: func(ctx context.Context, rid, reqID uuid.UUID, updates *models.RoleUpdate) (*models.Role, error) {
			return nil, services.ErrNotServerOwner
		},
	}

	app := setupServerTestApp(&mockServerService{}, &mockChannelService{}, roleSvc, userID)

	bodyBytes, _ := json.Marshal(map[string]interface{}{
		"name": "New Name",
	})
	req := httptest.NewRequest("PATCH", "/servers/"+serverID.String()+"/roles/"+roleID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("expected 403, got %d", resp.StatusCode)
	}
}

func TestServerHandler_DeleteRole_InvalidServerID(t *testing.T) {
	userID := uuid.New()

	app := setupServerTestApp(&mockServerService{}, &mockChannelService{}, &mockRoleService{}, userID)

	req := httptest.NewRequest("DELETE", "/servers/invalid-uuid/roles/"+uuid.New().String(), nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestServerHandler_DeleteRole_InvalidRoleID(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()

	app := setupServerTestApp(&mockServerService{}, &mockChannelService{}, &mockRoleService{}, userID)

	req := httptest.NewRequest("DELETE", "/servers/"+serverID.String()+"/roles/invalid-uuid", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestServerHandler_DeleteRole_Forbidden(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()
	roleID := uuid.New()

	roleSvc := &mockRoleService{
		deleteRoleFunc: func(ctx context.Context, rid, reqID uuid.UUID) error {
			return services.ErrNotServerOwner
		},
	}

	app := setupServerTestApp(&mockServerService{}, &mockChannelService{}, roleSvc, userID)

	req := httptest.NewRequest("DELETE", "/servers/"+serverID.String()+"/roles/"+roleID.String(), nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("expected 403, got %d", resp.StatusCode)
	}
}

// Additional comprehensive tests for Channels

func TestServerHandler_GetChannels_InvalidServerID(t *testing.T) {
	userID := uuid.New()

	app := setupServerTestApp(&mockServerService{}, &mockChannelService{}, &mockRoleService{}, userID)

	req := httptest.NewRequest("GET", "/servers/invalid-uuid/channels", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestServerHandler_CreateChannel_InvalidServerID(t *testing.T) {
	userID := uuid.New()

	app := setupServerTestApp(&mockServerService{}, &mockChannelService{}, &mockRoleService{}, userID)

	bodyBytes, _ := json.Marshal(map[string]interface{}{
		"name": "new-channel",
	})
	req := httptest.NewRequest("POST", "/servers/invalid-uuid/channels", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestServerHandler_CreateChannel_InvalidJSON(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()

	app := setupServerTestApp(&mockServerService{}, &mockChannelService{}, &mockRoleService{}, userID)

	req := httptest.NewRequest("POST", "/servers/"+serverID.String()+"/channels", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestServerHandler_CreateChannel_WithParent(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()
	parentID := uuid.New()

	channelSvc := &mockChannelService{
		createChannelFunc: func(ctx context.Context, sid, cid uuid.UUID, name string, chanType models.ChannelType, pid *uuid.UUID) (*models.Channel, error) {
			if pid == nil || *pid != parentID {
				t.Errorf("expected parent_id %s", parentID)
			}
			return &models.Channel{
				ID:       uuid.New(),
				ServerID: &sid,
				Name:     name,
				Type:     chanType,
				ParentID: pid,
			}, nil
		},
	}

	app := setupServerTestApp(&mockServerService{}, channelSvc, &mockRoleService{}, userID)

	bodyBytes, _ := json.Marshal(map[string]interface{}{
		"name":      "sub-channel",
		"type":      "text",
		"parent_id": parentID.String(),
	})
	req := httptest.NewRequest("POST", "/servers/"+serverID.String()+"/channels", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}
}

func TestServerHandler_CreateChannel_Forbidden(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()

	channelSvc := &mockChannelService{
		createChannelFunc: func(ctx context.Context, sid, cid uuid.UUID, name string, chanType models.ChannelType, pid *uuid.UUID) (*models.Channel, error) {
			return nil, services.ErrNotServerOwner
		},
	}

	app := setupServerTestApp(&mockServerService{}, channelSvc, &mockRoleService{}, userID)

	bodyBytes, _ := json.Marshal(map[string]interface{}{
		"name": "new-channel",
	})
	req := httptest.NewRequest("POST", "/servers/"+serverID.String()+"/channels", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("expected 403, got %d", resp.StatusCode)
	}
}
