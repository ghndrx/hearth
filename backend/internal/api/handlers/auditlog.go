package handlers

import (
	"context"
	"fmt"
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
	GetCategories() []models.AuditLogCategoryInfo
	GetDashboardSummary(ctx context.Context, serverID uuid.UUID, days int) (*models.ModerationDashboardSummary, error)
	GetTrendData(ctx context.Context, serverID uuid.UUID, days int) ([]models.DailyModerationTrend, error)
	GetModeratorActivity(ctx context.Context, serverID uuid.UUID, days int) ([]models.ModeratorStats, error)
	GetRepeatOffenders(ctx context.Context, serverID uuid.UUID, days, minCount int) ([]models.RepeatOffenderStats, error)
	GetAutoModStats(ctx context.Context, serverID uuid.UUID, days int) (*models.AutoModStats, error)
	ExportLogs(ctx context.Context, serverID uuid.UUID, format string, filter services.AuditLogFilter) ([]byte, string, error)
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
// @Summary Get audit logs for a server
// @Description Returns a paginated list of audit log entries for the specified server with optional filtering
// @Tags Audit Logs
// @Accept json
// @Produce json
// @Param id path string true "Server ID (UUID)"
// @Param action_type query string false "Filter by action type (e.g., 'MEMBER_BAN', 'CHANNEL_CREATE')"
// @Param action_category query int false "Filter by action category (10=Member, 20=Channel, 30=Server, 40=Message, 50=Role, 60=Integration, 70=Voice, 80=AutoMod)"
// @Param actor_id query string false "Filter by the user who performed the action (UUID)"
// @Param target_id query string false "Filter by the target of the action (UUID)"
// @Param target_type query string false "Filter by target type (member, message, channel, role)"
// @Param before query string false "Filter entries before this ISO8601 timestamp"
// @Param after query string false "Filter entries after this ISO8601 timestamp"
// @Param reason_keyword query string false "Filter by keyword in reason field"
// @Param limit query integer false "Maximum number of entries (default 50, max 100)"
// @Param offset query integer false "Offset for pagination"
// @Success 200 {object} fiber.Map "{audit_logs: [...], total: number, limit: number, offset: number}"
// @Failure 400 {object} fiber.Map "Invalid server ID or filter parameters"
// @Failure 403 {object} fiber.Map "Missing permission to view audit log or not a server member"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/audit-logs [get]
func (h *AuditLogHandler) GetAuditLogs(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

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

	// Parse action_category
	if categoryStr := c.Query("action_category"); categoryStr != "" {
		category, err := strconv.Atoi(categoryStr)
		if err != nil || category < 0 || category > 89 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid action_category",
			})
		}
		filter.ActionCategory = category
	}

	// Parse actor_id
	if actorIDStr := c.Query("actor_id"); actorIDStr != "" {
		uid, err := uuid.Parse(actorIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid actor_id",
			})
		}
		filter.ActorID = &uid
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

	// Parse target_type
	if targetType := c.Query("target_type"); targetType != "" {
		filter.TargetType = targetType
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

	// Parse reason_keyword
	if reasonKeyword := c.Query("reason_keyword"); reasonKeyword != "" {
		filter.ReasonKeyword = reasonKeyword
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
// @Summary Get a specific audit log entry
// @Description Returns a single audit log entry by its ID for the specified server
// @Tags Audit Logs
// @Accept json
// @Produce json
// @Param id path string true "Server ID (UUID)"
// @Param entryId path string true "Audit log entry ID (UUID)"
// @Success 200 {object} models.AuditLogEntry "Audit log entry"
// @Failure 400 {object} fiber.Map "Invalid server ID or entry ID"
// @Failure 403 {object} fiber.Map "Missing permission to view audit log or not a server member"
// @Failure 404 {object} fiber.Map "Audit log entry not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/audit-logs/{entryId} [get]
func (h *AuditLogHandler) GetAuditLogEntry(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

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
// @Summary Get available audit log action types
// @Description Returns a list of all valid audit log action types that can be used for filtering
// @Tags Audit Logs
// @Accept json
// @Produce json
// @Param id path string true "Server ID (UUID)"
// @Success 200 {object} fiber.Map "{action_types: [string]}"
// @Failure 400 {object} fiber.Map "Invalid server ID"
// @Failure 403 {object} fiber.Map "Missing permission to view audit log or not a server member"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/audit-logs/action-types [get]
func (h *AuditLogHandler) GetActionTypes(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

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

// GetCategories returns all audit log categories
// @Summary Get available audit log categories
// @Description Returns a list of all audit log categories with descriptions
// @Tags Audit Logs
// @Accept json
// @Produce json
// @Param id path string true "Server ID (UUID)"
// @Success 200 {object} fiber.Map "{categories: [...]}"
// @Failure 400 {object} fiber.Map "Invalid server ID"
// @Failure 403 {object} fiber.Map "Missing permission to view audit log or not a server member"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/audit-logs/categories [get]
func (h *AuditLogHandler) GetCategories(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

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

	categories := h.auditLogService.GetCategories()
	return c.JSON(fiber.Map{
		"categories": categories,
	})
}

// GetModerationDashboard returns the moderation dashboard summary
// @Summary Get moderation dashboard summary
// @Description Returns a quick summary of moderation actions for the dashboard
// @Tags Moderation Analytics
// @Accept json
// @Produce json
// @Param id path string true "Server ID (UUID)"
// @Param days query int false "Number of days to analyze (default 7, max 90)"
// @Success 200 {object} models.ModerationDashboardSummary
// @Failure 400 {object} fiber.Map "Invalid server ID"
// @Failure 403 {object} fiber.Map "Missing permission"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/moderation/dashboard [get]
func (h *AuditLogHandler) GetModerationDashboard(c *fiber.Ctx) error {
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

	hasPermission, err := h.checkViewAuditLogPermission(c, serverID, userID)
	if err != nil {
		return err
	}
	if !hasPermission {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "missing permission to view audit log",
		})
	}

	days := 7
	if daysStr := c.Query("days"); daysStr != "" {
		days, err = strconv.Atoi(daysStr)
		if err != nil || days < 1 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid days parameter",
			})
		}
		if days > 90 {
			days = 90
		}
	}

	summary, err := h.auditLogService.GetDashboardSummary(c.Context(), serverID, days)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get moderation dashboard",
		})
	}

	return c.JSON(summary)
}

// GetModerationTrend returns moderation trend data over time
// @Summary Get moderation trend data
// @Description Returns daily moderation action counts for trend analysis
// @Tags Moderation Analytics
// @Accept json
// @Produce json
// @Param id path string true "Server ID (UUID)"
// @Param days query int false "Number of days to analyze (default 7, max 90)"
// @Success 200 {object} fiber.Map "{trend_data: [...]}"
// @Failure 400 {object} fiber.Map "Invalid server ID"
// @Failure 403 {object} fiber.Map "Missing permission"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/moderation/trends [get]
func (h *AuditLogHandler) GetModerationTrend(c *fiber.Ctx) error {
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

	hasPermission, err := h.checkViewAuditLogPermission(c, serverID, userID)
	if err != nil {
		return err
	}
	if !hasPermission {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "missing permission to view audit log",
		})
	}

	days := 7
	if daysStr := c.Query("days"); daysStr != "" {
		days, err = strconv.Atoi(daysStr)
		if err != nil || days < 1 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid days parameter",
			})
		}
		if days > 90 {
			days = 90
		}
	}

	trend, err := h.auditLogService.GetTrendData(c.Context(), serverID, days)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get moderation trends",
		})
	}

	return c.JSON(fiber.Map{
		"period":     fmt.Sprintf("%dd", days),
		"trend_data": trend,
	})
}

// GetModeratorActivity returns moderation activity by moderator
// @Summary Get moderator activity
// @Description Returns moderation action counts per moderator
// @Tags Moderation Analytics
// @Accept json
// @Produce json
// @Param id path string true "Server ID (UUID)"
// @Param days query int false "Number of days to analyze (default 7, max 90)"
// @Success 200 {object} fiber.Map "{moderators: [...]}"
// @Failure 400 {object} fiber.Map "Invalid server ID"
// @Failure 403 {object} fiber.Map "Missing permission"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/moderation/moderators [get]
func (h *AuditLogHandler) GetModeratorActivity(c *fiber.Ctx) error {
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

	hasPermission, err := h.checkViewAuditLogPermission(c, serverID, userID)
	if err != nil {
		return err
	}
	if !hasPermission {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "missing permission to view audit log",
		})
	}

	days := 7
	if daysStr := c.Query("days"); daysStr != "" {
		days, err = strconv.Atoi(daysStr)
		if err != nil || days < 1 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid days parameter",
			})
		}
		if days > 90 {
			days = 90
		}
	}

	activity, err := h.auditLogService.GetModeratorActivity(c.Context(), serverID, days)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get moderator activity",
		})
	}

	return c.JSON(fiber.Map{
		"period":    fmt.Sprintf("%dd", days),
		"moderators": activity,
	})
}

// GetRepeatOffenders returns users who have been moderated multiple times
// @Summary Get repeat offenders
// @Description Returns users who have been moderated multiple times
// @Tags Moderation Analytics
// @Accept json
// @Produce json
// @Param id path string true "Server ID (UUID)"
// @Param days query int false "Number of days to analyze (default 30, max 90)"
// @Param min_count query int false "Minimum moderation count to include (default 2)"
// @Success 200 {object} fiber.Map "{offenders: [...]}"
// @Failure 400 {object} fiber.Map "Invalid server ID"
// @Failure 403 {object} fiber.Map "Missing permission"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/moderation/offenders [get]
func (h *AuditLogHandler) GetRepeatOffenders(c *fiber.Ctx) error {
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

	hasPermission, err := h.checkViewAuditLogPermission(c, serverID, userID)
	if err != nil {
		return err
	}
	if !hasPermission {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "missing permission to view audit log",
		})
	}

	days := 30
	if daysStr := c.Query("days"); daysStr != "" {
		days, err = strconv.Atoi(daysStr)
		if err != nil || days < 1 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid days parameter",
			})
		}
		if days > 90 {
			days = 90
		}
	}

	minCount := 2
	if minCountStr := c.Query("min_count"); minCountStr != "" {
		minCount, err = strconv.Atoi(minCountStr)
		if err != nil || minCount < 1 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid min_count parameter",
			})
		}
	}

	offenders, err := h.auditLogService.GetRepeatOffenders(c.Context(), serverID, days, minCount)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get repeat offenders",
		})
	}

	return c.JSON(fiber.Map{
		"period":     fmt.Sprintf("%dd", days),
		"min_count":  minCount,
		"offenders":  offenders,
	})
}

// GetAutoModStats returns auto-moderation statistics
// @Summary Get auto-moderation statistics
// @Description Returns auto-mod trigger and action statistics
// @Tags Moderation Analytics
// @Accept json
// @Produce json
// @Param id path string true "Server ID (UUID)"
// @Param days query int false "Number of days to analyze (default 7, max 90)"
// @Success 200 {object} models.AutoModStats
// @Failure 400 {object} fiber.Map "Invalid server ID"
// @Failure 403 {object} fiber.Map "Missing permission"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/moderation/automod [get]
func (h *AuditLogHandler) GetAutoModStats(c *fiber.Ctx) error {
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

	hasPermission, err := h.checkViewAuditLogPermission(c, serverID, userID)
	if err != nil {
		return err
	}
	if !hasPermission {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "missing permission to view audit log",
		})
	}

	days := 7
	if daysStr := c.Query("days"); daysStr != "" {
		days, err = strconv.Atoi(daysStr)
		if err != nil || days < 1 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid days parameter",
			})
		}
		if days > 90 {
			days = 90
		}
	}

	stats, err := h.auditLogService.GetAutoModStats(c.Context(), serverID, days)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get auto-mod stats",
		})
	}

	return c.JSON(fiber.Map{
		"period": fmt.Sprintf("%dd", days),
		"stats":  stats,
	})
}

// ExportAuditLogs exports audit logs in the specified format
// @Summary Export audit logs
// @Description Exports audit logs as JSON or CSV
// @Tags Audit Logs
// @Accept json
// @Produce json,text/csv
// @Param id path string true "Server ID (UUID)"
// @Param format query string false "Export format: json or csv (default json)"
// @Param action_type query string false "Filter by action type"
// @Param before query string false "Filter entries before this ISO8601 timestamp"
// @Param after query string false "Filter entries after this ISO8601 timestamp"
// @Success 200 {file} "Audit log export"
// @Failure 400 {object} fiber.Map "Invalid server ID or format"
// @Failure 403 {object} fiber.Map "Missing permission"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/audit-logs/export [get]
func (h *AuditLogHandler) ExportAuditLogs(c *fiber.Ctx) error {
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

	hasPermission, err := h.checkViewAuditLogPermission(c, serverID, userID)
	if err != nil {
		return err
	}
	if !hasPermission {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "missing permission to view audit log",
		})
	}

	format := c.Query("format", "json")
	if format != "json" && format != "csv" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid format, must be 'json' or 'csv'",
		})
	}

	filter := services.AuditLogFilter{}

	// Parse action_type
	if actionType := c.Query("action_type"); actionType != "" {
		filter.ActionType = actionType
	}

	// Parse before
	if beforeStr := c.Query("before"); beforeStr != "" {
		before, err := time.Parse(time.RFC3339, beforeStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid before timestamp",
			})
		}
		filter.Before = &before
	}

	// Parse after
	if afterStr := c.Query("after"); afterStr != "" {
		after, err := time.Parse(time.RFC3339, afterStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid after timestamp",
			})
		}
		filter.After = &after
	}

	data, contentType, err := h.auditLogService.ExportLogs(c.Context(), serverID, format, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to export audit logs",
		})
	}

	filename := fmt.Sprintf("audit-logs-%s-%s.%s", serverID.String()[:8], time.Now().Format("2006-01-02"), format)

	c.Set("Content-Type", contentType)
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	return c.Send(data)
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
	hasPermission := (perms&models.PermViewAuditLog) != 0 || (perms&models.PermAdministrator) != 0
	return hasPermission, nil
}

// RegisterAuditLogRoutes registers the audit log routes
func (h *AuditLogHandler) RegisterAuditLogRoutes(router fiber.Router) {
	auditLogs := router.Group("/:id/audit-logs")
	auditLogs.Get("", h.GetAuditLogs)
	auditLogs.Get("/action-types", h.GetActionTypes)
	auditLogs.Get("/categories", h.GetCategories)
	auditLogs.Get("/export", h.ExportAuditLogs)
	auditLogs.Get("/:entryId", h.GetAuditLogEntry)

	moderation := router.Group("/:id/moderation")
	moderation.Get("/dashboard", h.GetModerationDashboard)
	moderation.Get("/trends", h.GetModerationTrend)
	moderation.Get("/moderators", h.GetModeratorActivity)
	moderation.Get("/offenders", h.GetRepeatOffenders)
	moderation.Get("/automod", h.GetAutoModStats)
}
