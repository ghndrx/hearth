package handlers

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// RoleServiceInterface defines the interface for role service operations
type RoleServiceInterface interface {
	CreateRole(ctx context.Context, serverID, creatorID uuid.UUID, name string, color int, permissions int64) (*models.Role, error)
	GetServerRoles(ctx context.Context, serverID, requesterID uuid.UUID) ([]*models.Role, error)
	GetRole(ctx context.Context, roleID, requesterID uuid.UUID) (*models.Role, error)
	UpdateRole(ctx context.Context, roleID, requesterID uuid.UUID, updates *models.RoleUpdate) (*models.Role, error)
	DeleteRole(ctx context.Context, roleID, requesterID uuid.UUID) error
	UpdateRolePositions(ctx context.Context, serverID, requesterID uuid.UUID, positions map[uuid.UUID]int) error
	AddRoleToMember(ctx context.Context, serverID, userID, roleID, requesterID uuid.UUID) error
	RemoveRoleFromMember(ctx context.Context, serverID, userID, roleID, requesterID uuid.UUID) error
}

// RoleHandlers handles role-related HTTP requests
type RoleHandlers struct {
	roleService RoleServiceInterface
}

// NewRoleHandlers creates new role handlers
func NewRoleHandlers(roleService RoleServiceInterface) *RoleHandlers {
	return &RoleHandlers{roleService: roleService}
}

// CreateRole creates a new role in a server
// @Summary Create a new role
// @Description Creates a new role in the specified server with the given name, color, and permissions
// @Tags Roles
// @Accept json
// @Produce json
// @Param serverID path string true "Server ID"
// @Param body body struct{Name string `json:"name"`; Color int `json:"color"`; Permissions int64 `json:"permissions"`; Hoist bool `json:"hoist"`; Mentionable bool `json:"mentionable"`} true "Role creation data"
// @Success 201 {object} models.Role "Role created successfully"
// @Failure 400 {object} fiber.Map "Invalid server ID or request body"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{serverID}/roles [post]
func (h *RoleHandlers) CreateRole(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
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
// @Summary Get server roles
// @Description Returns all roles in the specified server
// @Tags Roles
// @Produce json
// @Param serverID path string true "Server ID"
// @Success 200 {array} models.Role "List of roles"
// @Failure 400 {object} fiber.Map "Invalid server ID"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{serverID}/roles [get]
func (h *RoleHandlers) GetRoles(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
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
// @Summary Get role by ID
// @Description Returns a specific role by its ID
// @Tags Roles
// @Produce json
// @Param roleID path string true "Role ID"
// @Success 200 {object} models.Role "Role details"
// @Failure 400 {object} fiber.Map "Invalid role ID"
// @Failure 403 {object} fiber.Map "Not a member of this server"
// @Failure 404 {object} fiber.Map "Role not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /roles/{roleID} [get]
func (h *RoleHandlers) GetRole(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
	roleID, err := uuid.Parse(c.Params("roleID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid role ID",
		})
	}

	role, err := h.roleService.GetRole(c.Context(), roleID, userID)
	if err != nil {
		if err == services.ErrRoleNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Role not found",
			})
		}
		if err == services.ErrNotServerMember {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Not a member of this server",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(role)
}

// UpdateRole updates a role
// @Summary Update a role
// @Description Updates a role's properties such as name, color, permissions, hoist, mentionable, or position
// @Tags Roles
// @Accept json
// @Produce json
// @Param roleID path string true "Role ID"
// @Param body body struct{Name *string `json:"name,omitempty"`; Color *int `json:"color,omitempty"`; Permissions *int64 `json:"permissions,omitempty"`; Hoist *bool `json:"hoist,omitempty"`; Mentionable *bool `json:"mentionable,omitempty"`; Position *int `json:"position,omitempty"`} true "Role update data"
// @Success 200 {object} models.Role "Updated role"
// @Failure 400 {object} fiber.Map "Invalid role ID or request body"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /roles/{roleID} [patch]
func (h *RoleHandlers) UpdateRole(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
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
// @Summary Delete a role
// @Description Deletes the specified role from the server
// @Tags Roles
// @Param roleID path string true "Role ID"
// @Success 204 "Role deleted successfully"
// @Failure 400 {object} fiber.Map "Invalid role ID"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /roles/{roleID} [delete]
func (h *RoleHandlers) DeleteRole(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
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

// BatchUpdateRolesPositions updates the positions of multiple roles (reorder)
// @Summary Reorder roles
// @Description Updates the position/order of multiple roles in a server
// @Tags Roles
// @Accept json
// @Produce json
// @Param serverID path string true "Server ID"
// @Param body body struct{Positions []struct{ID string `json:"id"`; Position int `json:"position"`} `json:"positions"`} true "Role positions"
// @Success 200 {array} models.Role "Updated roles in new order"
// @Failure 400 {object} fiber.Map "Invalid request body"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{serverID}/roles [patch]
func (h *RoleHandlers) BatchUpdateRolesPositions(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
	serverID, err := uuid.Parse(c.Params("serverID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid server ID",
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
			"error": "Invalid request body",
		})
	}

	if len(req.Positions) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No positions provided",
		})
	}

	positions := make(map[uuid.UUID]int)
	for _, p := range req.Positions {
		roleID, err := uuid.Parse(p.ID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid role ID: " + p.ID,
			})
		}
		positions[roleID] = p.Position
	}

	if err := h.roleService.UpdateRolePositions(c.Context(), serverID, userID, positions); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Return updated roles in new order
	roles, err := h.roleService.GetServerRoles(c.Context(), serverID, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(roles)
}

// AddMemberRole adds a role to a member
// @Summary Add role to member
// @Description Adds a role to a member in the specified server
// @Tags Roles
// @Param serverID path string true "Server ID"
// @Param memberID path string true "Member ID"
// @Param roleID path string true "Role ID"
// @Success 204 "Role added to member successfully"
// @Failure 400 {object} fiber.Map "Invalid server ID, member ID, or role ID"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{serverID}/members/{memberID}/roles/{roleID} [put]
func (h *RoleHandlers) AddMemberRole(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
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
// @Summary Remove role from member
// @Description Removes a role from a member in the specified server
// @Tags Roles
// @Param serverID path string true "Server ID"
// @Param memberID path string true "Member ID"
// @Param roleID path string true "Role ID"
// @Success 204 "Role removed from member successfully"
// @Failure 400 {object} fiber.Map "Invalid server ID, member ID, or role ID"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{serverID}/members/{memberID}/roles/{roleID} [delete]
func (h *RoleHandlers) RemoveMemberRole(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
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
