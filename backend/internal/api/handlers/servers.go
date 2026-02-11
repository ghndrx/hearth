package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	
	"hearth/internal/services"
)

type ServerHandler struct {
	serverService *services.ServerService
}

func NewServerHandler(serverService *services.ServerService) *ServerHandler {
	return &ServerHandler{serverService: serverService}
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
	
	server, err := h.serverService.CreateServer(c.Context(), userID, req.Name, req.Icon)
	if err != nil {
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
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "server not found",
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
	
	// TODO: Pass updates to service
	_, _ = userID, id
	
	return c.JSON(fiber.Map{"status": "updated"})
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
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	
	return c.SendStatus(fiber.StatusNoContent)
}

// GetMembers returns server members
func (h *ServerHandler) GetMembers(c *fiber.Ctx) error {
	// TODO: Implement with pagination
	return c.JSON([]interface{}{})
}

// GetMember returns a specific member
func (h *ServerHandler) GetMember(c *fiber.Ctx) error {
	// TODO: Implement
	return c.JSON(fiber.Map{})
}

// UpdateMember updates a member (nickname, roles)
func (h *ServerHandler) UpdateMember(c *fiber.Ctx) error {
	// TODO: Implement
	return c.JSON(fiber.Map{})
}

// RemoveMember kicks a member
func (h *ServerHandler) RemoveMember(c *fiber.Ctx) error {
	// TODO: Implement
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
	return c.JSON([]interface{}{})
}

// CreateBan bans a user
func (h *ServerHandler) CreateBan(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}

// RemoveBan unbans a user
func (h *ServerHandler) RemoveBan(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}

// GetInvites returns server invites
func (h *ServerHandler) GetInvites(c *fiber.Ctx) error {
	return c.JSON([]interface{}{})
}

// GetRoles returns server roles
func (h *ServerHandler) GetRoles(c *fiber.Ctx) error {
	return c.JSON([]interface{}{})
}

// CreateRole creates a new role
func (h *ServerHandler) CreateRole(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{})
}

// UpdateRole updates a role
func (h *ServerHandler) UpdateRole(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{})
}

// DeleteRole deletes a role
func (h *ServerHandler) DeleteRole(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}

// GetChannels returns server channels
func (h *ServerHandler) GetChannels(c *fiber.Ctx) error {
	return c.JSON([]interface{}{})
}

// CreateChannel creates a new channel
func (h *ServerHandler) CreateChannel(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{})
}
