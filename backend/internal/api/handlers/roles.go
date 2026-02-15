package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// RoleHandlers handles role-related HTTP requests
type RoleHandlers struct {
	roleService *services.RoleService
}

// NewRoleHandlers creates new role handlers
func NewRoleHandlers(roleService *services.RoleService) *RoleHandlers {
	return &RoleHandlers{roleService: roleService}
}

// CreateRole creates a new role in a server
func (h *RoleHandlers) CreateRole(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	serverID, err := uuid.Parse(c.Params("serverID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid server ID",
		})
	}

	var req struct {
		Name        string `json:"name"`
		Color       int    `json:"color"`
		Permissions int64  `json:"permissions"`
		Hoist       bool   `json:"hoist"`
		Mentionable bool   `json:"mentionable"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	role, err := h.roleService.CreateRole(c.Context(), serverID, userID, req.Name, req.Color, req.Permissions)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(role)
}

// GetRoles returns all roles in a server
func (h *RoleHandlers) GetRoles(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	serverID, err := uuid.Parse(c.Params("serverID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid server ID",
		})
	}

	roles, err := h.roleService.GetServerRoles(c.Context(), serverID, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(roles)
}

// GetRole returns a specific role
func (h *RoleHandlers) GetRole(c *fiber.Ctx) error {
	roleID, err := uuid.Parse(c.Params("roleID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid role ID",
		})
	}

	role, err := h.roleService.GetRole(c.Context(), roleID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if role == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Role not found",
		})
	}

	return c.JSON(role)
}

// UpdateRole updates a role
func (h *RoleHandlers) UpdateRole(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	roleID, err := uuid.Parse(c.Params("roleID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid role ID",
		})
	}

	var req struct {
		Name        *string `json:"name,omitempty"`
		Color       *int    `json:"color,omitempty"`
		Permissions *int64  `json:"permissions,omitempty"`
		Hoist       *bool   `json:"hoist,omitempty"`
		Mentionable *bool   `json:"mentionable,omitempty"`
		Position    *int    `json:"position,omitempty"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	updates := &models.RoleUpdate{
		Name:        req.Name,
		Color:       req.Color,
		Permissions: req.Permissions,
		Hoist:       req.Hoist,
		Mentionable: req.Mentionable,
		Position:    req.Position,
	}

	role, err := h.roleService.UpdateRole(c.Context(), roleID, userID, updates)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(role)
}

// DeleteRole deletes a role
func (h *RoleHandlers) DeleteRole(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	roleID, err := uuid.Parse(c.Params("roleID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid role ID",
		})
	}

	err = h.roleService.DeleteRole(c.Context(), roleID, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// AddMemberRole adds a role to a member
func (h *RoleHandlers) AddMemberRole(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	serverID, err := uuid.Parse(c.Params("serverID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid server ID",
		})
	}
	memberID, err := uuid.Parse(c.Params("memberID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid member ID",
		})
	}
	roleID, err := uuid.Parse(c.Params("roleID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid role ID",
		})
	}

	err = h.roleService.AddRoleToMember(c.Context(), serverID, memberID, roleID, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// RemoveMemberRole removes a role from a member
func (h *RoleHandlers) RemoveMemberRole(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	serverID, err := uuid.Parse(c.Params("serverID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid server ID",
		})
	}
	memberID, err := uuid.Parse(c.Params("memberID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid member ID",
		})
	}
	roleID, err := uuid.Parse(c.Params("roleID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid role ID",
		})
	}

	err = h.roleService.RemoveRoleFromMember(c.Context(), serverID, memberID, roleID, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
