package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// AnalyticsHandler handles server analytics API requests
type AnalyticsHandler struct {
	analyticsService *services.AnalyticsService
}

// NewAnalyticsHandler creates a new analytics handler
func NewAnalyticsHandler(analyticsService *services.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{
		analyticsService: analyticsService,
	}
}

// GetSummary returns a quick overview of key server metrics
// GET /api/v1/guilds/:id/insights
func (h *AnalyticsHandler) GetSummary(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
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

	summary, err := h.analyticsService.GetSummary(c.Context(), serverID, userID)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(summary)
}

// GetMemberGrowth returns member growth history
// GET /api/v1/guilds/:id/insights/growth?days=30
func (h *AnalyticsHandler) GetMemberGrowth(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
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

	days := h.parseDays(c.Query("days", "7"))

	growth, err := h.analyticsService.GetMemberGrowth(c.Context(), serverID, userID, days)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(growth)
}

// GetMessageActivity returns message activity heatmap data
// GET /api/v1/guilds/:id/insights/activity?days=7
func (h *AnalyticsHandler) GetMessageActivity(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
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

	days := h.parseDays(c.Query("days", "7"))

	activity, err := h.analyticsService.GetMessageActivity(c.Context(), serverID, userID, days)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(activity)
}

// GetTopChannels returns channels ranked by message volume
// GET /api/v1/guilds/:id/insights/channels?days=7&limit=10
func (h *AnalyticsHandler) GetTopChannels(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
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

	days := h.parseDays(c.Query("days", "7"))
	limit := h.parseLimit(c.Query("limit", "10"))

	channels, err := h.analyticsService.GetTopChannels(c.Context(), serverID, userID, days, limit)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(channels)
}

// GetRetention returns retention and engagement metrics
// GET /api/v1/guilds/:id/insights/retention?days=30
func (h *AnalyticsHandler) GetRetention(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
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

	days := h.parseDays(c.Query("days", "30"))

	retention, err := h.analyticsService.GetRetention(c.Context(), serverID, userID, days)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(retention)
}

// GetMostActiveUsers returns the most active users
// GET /api/v1/guilds/:id/insights/users?days=7&limit=10
func (h *AnalyticsHandler) GetMostActiveUsers(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
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

	days := h.parseDays(c.Query("days", "7"))
	limit := h.parseLimit(c.Query("limit", "10"))

	users, err := h.analyticsService.GetMostActiveUsers(c.Context(), serverID, userID, days, limit)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(fiber.Map{
		"server_id": serverID.String(),
		"period":    strconv.Itoa(days) + "d",
		"data":      users,
	})
}

// parseDays parses and normalizes the days query parameter
func (h *AnalyticsHandler) parseDays(s string) int {
	days, err := strconv.Atoi(s)
	if err != nil || days <= 0 {
		return 7
	}
	if days > 90 {
		return 90
	}
	return days
}

// parseLimit parses and normalizes the limit query parameter
func (h *AnalyticsHandler) parseLimit(s string) int {
	limit, err := strconv.Atoi(s)
	if err != nil || limit <= 0 {
		return 10
	}
	if limit > 50 {
		return 50
	}
	return limit
}

// handleError converts service errors to HTTP responses
func (h *AnalyticsHandler) handleError(c *fiber.Ctx, err error) error {
	switch err {
	case services.ErrServerNotFound:
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "server not found",
		})
	case services.ErrNotServerMember:
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "not a member of this server",
		})
	case services.ErrMissingPermission:
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "missing MANAGE_SERVER permission",
		})
	default:
		// Check if it's a permission error
		if err != nil && err.Error() == "missing permission: MANAGE_SERVER" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "missing MANAGE_SERVER permission",
			})
		}
		// Log the actual error for debugging
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}
}

// InvalidateCache clears cached analytics for a server (admin use)
// POST /api/v1/guilds/:id/insights/invalidate
func (h *AnalyticsHandler) InvalidateCache(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
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

	// Just verify permissions - GetSummary does the permission check
	_, err = h.analyticsService.GetSummary(c.Context(), serverID, userID)
	if err != nil {
		return h.handleError(c, err)
	}

	// Invalidate cache
	if err := h.analyticsService.InvalidateCache(c.Context(), serverID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to invalidate cache",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "analytics cache invalidated",
	})
}

// -- Request/Response types for documentation --

// AnalyticsSummaryResponse represents the summary endpoint response
type AnalyticsSummaryResponse = models.ServerInsightsResponse

// MemberGrowthResponse represents the growth endpoint response
type MemberGrowthResponse = models.MemberGrowthResponse

// ActivityHeatmapResponse represents the activity endpoint response
type ActivityHeatmapResponse = models.ActivityHeatmapResponse

// TopChannelsResponse represents the channels endpoint response
type TopChannelsResponse = models.TopChannelsResponse

// RetentionResponse represents the retention endpoint response
type RetentionResponse = models.RetentionResponse
