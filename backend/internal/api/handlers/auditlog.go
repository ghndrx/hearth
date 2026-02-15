package handlers

import (
	"context"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// AuditLogServiceInterface defines the methods needed from AuditLogService
type AuditLogServiceInterface interface {
	GetLogs(ctx context.Context, serverID uuid.UUID, filter services.AuditLogFilter) ([]models.AuditLogEntry, int, error)
	GetLogByID(ctx context.Context, serverID, entryID uuid.UUID) (*models.AuditLogEntry, error)
	GetActionTypes() []string
}

// ServerServiceForAuditLog defines the methods needed from ServerService for permission checks
type ServerServiceForAuditLog interface {
	GetMember(ctx context.Context, serverID, userID uuid.UUID) (*models.Member, error)
	GetMemberPermissions(ctx context.Context, serverID, userID uuid.UUID) (int64, error)
}

// AuditLogHandler handles audit log-related HTTP requests
type AuditLogHandler struct {
	auditLogService AuditLogServiceInterface
	serverService   ServerServiceForAuditLog
}

// NewAuditLogHandler creates a new audit log handler
func NewAuditLogHandler(auditLogService AuditLogServiceInterface, serverService ServerServiceForAuditLog) *AuditLogHandler {
	return &AuditLogHandler{
		auditLogService: auditLogService,
		serverService:   serverService,
	}
}

// GetAuditLogs returns the audit logs for a server with filtering
// GET /api/v1/servers/:id/audit-logs
// Query parameters:
//   - action_type: Filter by action type (e.g., "MEMBER_BAN", "CHANNEL_CREATE")
//   - user_id: Filter by the user who performed the action
//   - target_id: Filter by the target of the action
//   - before: Filter entries before this ISO8601 timestamp
//   - after: Filter entries after this ISO8601 timestamp
//   - limit: Maximum number of entries (default 50, max 100)
//   - offset: Offset for pagination
func (h *AuditLogHandler) GetAuditLogs(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	// Parse server ID
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	// Check permission to view audit log
	hasPermission, err := h.checkViewAuditLogPermission(c, serverID, userID)
	if err != nil {
		return err
	}
	if !hasPermission {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "missing permission to view audit log",
		})
	}

	// Build filter from query parameters
	filter := services.AuditLogFilter{
		Limit:  50,
		Offset: 0,
	}

	// Parse action_type
	if actionType := c.Query("action_type"); actionType != "" {
		filter.ActionType = actionType
	}

	// Parse user_id
	if userIDStr := c.Query("user_id"); userIDStr != "" {
		uid, err := uuid.Parse(userIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid user_id",
			})
		}
		filter.UserID = &uid
	}

	// Parse target_id
	if targetIDStr := c.Query("target_id"); targetIDStr != "" {
		tid, err := uuid.Parse(targetIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid target_id",
			})
		}
		filter.TargetID = &tid
	}

	// Parse before
	if beforeStr := c.Query("before"); beforeStr != "" {
		before, err := time.Parse(time.RFC3339, beforeStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid before timestamp, use ISO8601 format",
			})
		}
		filter.Before = &before
	}

	// Parse after
	if afterStr := c.Query("after"); afterStr != "" {
		after, err := time.Parse(time.RFC3339, afterStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid after timestamp, use ISO8601 format",
			})
		}
		filter.After = &after
	}

	// Parse limit
	if limitStr := c.Query("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid limit",
			})
		}
		if limit > 100 {
			limit = 100
		}
		filter.Limit = limit
	}

	// Parse offset
	if offsetStr := c.Query("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid offset",
			})
		}
		filter.Offset = offset
	}

	// Get logs
	logs, total, err := h.auditLogService.GetLogs(c.Context(), serverID, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get audit logs",
		})
	}

	return c.JSON(fiber.Map{
		"audit_logs": logs,
		"total":      total,
		"limit":      filter.Limit,
		"offset":     filter.Offset,
	})
}

// GetAuditLogEntry returns a specific audit log entry
// GET /api/v1/servers/:id/audit-logs/:entryId
func (h *AuditLogHandler) GetAuditLogEntry(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	// Parse server ID
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	// Parse entry ID
	entryID, err := uuid.Parse(c.Params("entryId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid audit log entry id",
		})
	}

	// Check permission to view audit log
	hasPermission, err := h.checkViewAuditLogPermission(c, serverID, userID)
	if err != nil {
		return err
	}
	if !hasPermission {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "missing permission to view audit log",
		})
	}

	entry, err := h.auditLogService.GetLogByID(c.Context(), serverID, entryID)
	if err != nil {
		if err == services.ErrAuditLogNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "audit log entry not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get audit log entry",
		})
	}

	return c.JSON(entry)
}

// GetActionTypes returns all valid audit log action types
// GET /api/v1/servers/:id/audit-logs/action-types
func (h *AuditLogHandler) GetActionTypes(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	// Parse server ID
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	// Check permission to view audit log
	hasPermission, err := h.checkViewAuditLogPermission(c, serverID, userID)
	if err != nil {
		return err
	}
	if !hasPermission {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "missing permission to view audit log",
		})
	}

	types := h.auditLogService.GetActionTypes()
	return c.JSON(fiber.Map{
		"action_types": types,
	})
}

// checkViewAuditLogPermission checks if a user has permission to view the audit log
func (h *AuditLogHandler) checkViewAuditLogPermission(c *fiber.Ctx, serverID, userID uuid.UUID) (bool, error) {
	// First check if user is a member of the server
	_, err := h.serverService.GetMember(c.Context(), serverID, userID)
	if err != nil {
		if err == services.ErrNotServerMember {
			return false, c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "not a member of this server",
			})
		}
		return false, c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to verify server membership",
		})
	}

	// Check if user has VIEW_AUDIT_LOG permission
	perms, err := h.serverService.GetMemberPermissions(c.Context(), serverID, userID)
	if err != nil {
		return false, c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to check permissions",
		})
	}

	// Check for VIEW_AUDIT_LOG permission or ADMINISTRATOR
	hasPermission := (perms & models.PermViewAuditLog) != 0 || (perms & models.PermAdministrator) != 0
	return hasPermission, nil
}
