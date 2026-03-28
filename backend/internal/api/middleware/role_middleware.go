package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"hearth/internal/models"
)

// RoleService defines the interface for role-related operations
type RoleService interface {
	GetMemberRoles(ctx interface{}, serverID, userID uuid.UUID) ([]*models.Role, error)
	ComputeMemberPermissions(ctx interface{}, serverID, userID uuid.UUID) (int64, error)
}

// ServerRepository defines the interface for server operations needed by role middleware
type ServerRepository interface {
	GetMember(ctx interface{}, serverID, userID uuid.UUID) (*models.Member, error)
	GetByID(ctx interface{}, id uuid.UUID) (*models.Server, error)
}

// RoleMiddleware handles role and permission checks
type RoleMiddleware struct {
	roleService RoleService
	serverRepo  ServerRepository
}

// NewRoleMiddleware creates a new role middleware instance
func NewRoleMiddleware(roleService RoleService, serverRepo ServerRepository) *RoleMiddleware {
	return &RoleMiddleware{
		roleService: roleService,
		serverRepo:  serverRepo,
	}
}

// RequireRole checks if the user has a specific role in the server
// The role ID should be provided as a parameter or the middleware checks for any of the specified roles
func (rm *RoleMiddleware) RequireRole(roleIDs ...uuid.UUID) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, ok := c.Locals("userID").(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "unauthorized",
			})
		}

		serverIDStr := c.Params("serverID")
		if serverIDStr == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "server ID required",
			})
		}

		serverID, err := uuid.Parse(serverIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid server ID",
			})
		}

		// Check if user is a member of the server
		member, err := rm.serverRepo.GetMember(c.Context(), serverID, userID)
		if err != nil || member == nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "not a member of this server",
			})
		}

		// Get user's roles
		userRoles, err := rm.roleService.GetMemberRoles(c.Context(), serverID, userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to check roles",
			})
		}

		// Check if user has any of the required roles
		userRoleIDs := make(map[uuid.UUID]bool)
		for _, role := range userRoles {
			userRoleIDs[role.ID] = true
		}

		for _, requiredRoleID := range roleIDs {
			if userRoleIDs[requiredRoleID] {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "missing required role",
		})
	}
}

// RequirePermission checks if the user has a specific permission in the server
func (rm *RoleMiddleware) RequirePermission(permission int64) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, ok := c.Locals("userID").(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "unauthorized",
			})
		}

		serverIDStr := c.Params("serverID")
		if serverIDStr == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "server ID required",
			})
		}

		serverID, err := uuid.Parse(serverIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid server ID",
			})
		}

		// Check if user is a member of the server
		member, err := rm.serverRepo.GetMember(c.Context(), serverID, userID)
		if err != nil || member == nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "not a member of this server",
			})
		}

		// Check server ownership
		server, err := rm.serverRepo.GetByID(c.Context(), serverID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to check server",
			})
		}
		if server == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "server not found",
			})
		}

		// Server owner has all permissions
		if server.OwnerID == userID {
			return c.Next()
		}

		// Get user's effective permissions
		permissions, err := rm.roleService.ComputeMemberPermissions(c.Context(), serverID, userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to check permissions",
			})
		}

		// Check if user has the required permission
		if !models.HasPermission(permissions, permission) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "missing required permission",
			})
		}

		return c.Next()
	}
}

// RequirePermissionOrRole checks if user has either the permission OR any of the specified roles
func (rm *RoleMiddleware) RequirePermissionOrRole(permission int64, roleIDs ...uuid.UUID) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, ok := c.Locals("userID").(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "unauthorized",
			})
		}

		serverIDStr := c.Params("serverID")
		if serverIDStr == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "server ID required",
			})
		}

		serverID, err := uuid.Parse(serverIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid server ID",
			})
		}

		// Check if user is a member of the server
		member, err := rm.serverRepo.GetMember(c.Context(), serverID, userID)
		if err != nil || member == nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "not a member of this server",
			})
		}

		// Get server for ownership check
		server, err := rm.serverRepo.GetByID(c.Context(), serverID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to check server",
			})
		}
		if server == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "server not found",
			})
		}

		// Server owner has all permissions
		if server.OwnerID == userID {
			return c.Next()
		}

		// Check roles first (faster check)
		if len(roleIDs) > 0 {
			userRoles, err := rm.roleService.GetMemberRoles(c.Context(), serverID, userID)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "failed to check roles",
				})
			}

			userRoleIDs := make(map[uuid.UUID]bool)
			for _, role := range userRoles {
				userRoleIDs[role.ID] = true
			}

			for _, requiredRoleID := range roleIDs {
				if userRoleIDs[requiredRoleID] {
					return c.Next()
				}
			}
		}

		// Check permission
		permissions, err := rm.roleService.ComputeMemberPermissions(c.Context(), serverID, userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to check permissions",
			})
		}

		if models.HasPermission(permissions, permission) {
			return c.Next()
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "missing required permission or role",
		})
	}
}

// RequireAdmin checks if the user has administrator permission
func (rm *RoleMiddleware) RequireAdmin() fiber.Handler {
	return rm.RequirePermission(models.PermAdministrator)
}

// RequireServerOwner checks if the user is the server owner
func (rm *RoleMiddleware) RequireServerOwner() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, ok := c.Locals("userID").(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "unauthorized",
			})
		}

		serverIDStr := c.Params("serverID")
		if serverIDStr == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "server ID required",
			})
		}

		serverID, err := uuid.Parse(serverIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid server ID",
			})
		}

		server, err := rm.serverRepo.GetByID(c.Context(), serverID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to check server",
			})
		}
		if server == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "server not found",
			})
		}

		if server.OwnerID != userID {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "only server owner can perform this action",
			})
		}

		return c.Next()
	}
}
