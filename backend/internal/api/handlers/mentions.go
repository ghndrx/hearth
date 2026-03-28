package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// MentionsHandler handles mention-related HTTP requests
type MentionsHandler struct {
	mentionService *services.MentionAPIService
}

// NewMentionsHandler creates a new mentions handler
func NewMentionsHandler(mentionService *services.MentionAPIService) *MentionsHandler {
	return &MentionsHandler{
		mentionService: mentionService,
	}
}

// GetMentions returns a list of mentions for the current user
// @Summary Get user mentions
// @Description Returns a paginated list of mentions for the current user with optional filtering
// @Tags Mentions
// @Accept json
// @Produce json
// @Param unread query boolean false "Filter by unread status"
// @Param type query string false "Filter by mention type (reply, mention, everyone, here)"
// @Param channel_id query string false "Filter by channel ID"
// @Param guild_id query string false "Filter by guild/server ID"
// @Param limit query integer false "Number of results to return (default: 50, max: 100)"
// @Param offset query integer false "Number of results to skip"
// @Param before query string false "RFC3339 timestamp to get mentions before"
// @Param after query string false "RFC3339 timestamp to get mentions after"
// @Success 200 {object} models.MentionsListResponse "List of mentions"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /mentions [get]
func (h *MentionsHandler) GetMentions(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	// Parse query parameters
	filter := &models.MentionFilter{
		UserID: userID,
	}

	// Unread filter
	if unread := c.Query("unread"); unread != "" {
		isUnread := unread == "true" || unread == "1"
		filter.Unread = &isUnread
	}

	// Type filter
	if mentionType := c.Query("type"); mentionType != "" {
		mt := models.MentionKind(mentionType)
		filter.MentionType = &mt
	}

	// Channel filter
	if channelID := c.Query("channel_id"); channelID != "" {
		id, err := uuid.Parse(channelID)
		if err == nil {
			filter.ChannelID = &id
		}
	}

	// Guild/Server filter
	if guildID := c.Query("guild_id"); guildID != "" {
		id, err := uuid.Parse(guildID)
		if err == nil {
			filter.GuildID = &id
		}
	}

	// Pagination
	filter.Limit = c.QueryInt("limit", 50)
	filter.Offset = c.QueryInt("offset", 0)

	// Before timestamp
	if before := c.Query("before"); before != "" {
		if t, err := time.Parse(time.RFC3339, before); err == nil {
			filter.Before = &t
		}
	}

	// After timestamp
	if after := c.Query("after"); after != "" {
		if t, err := time.Parse(time.RFC3339, after); err == nil {
			filter.After = &t
		}
	}

	mentions, total, err := h.mentionService.GetMentions(c.Context(), filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get mentions",
		})
	}

	hasMore := len(mentions) > filter.Limit
	if hasMore {
		mentions = mentions[:filter.Limit]
	}

	return c.JSON(models.MentionsListResponse{
		Mentions:   mentions,
		TotalCount: total,
		HasMore:    hasMore,
	})
}

// GetUnreadCount returns the count of unread mentions
// @Summary Get unread mentions count
// @Description Returns the total count of unread mentions for the current user
// @Tags Mentions
// @Produce json
// @Success 200 {object} fiber.Map "Unread count"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /mentions/unread/count [get]
func (h *MentionsHandler) GetUnreadCount(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	count, err := h.mentionService.GetUnreadCount(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get unread count",
		})
	}

	return c.JSON(fiber.Map{
		"count": count,
	})
}

// GetStats returns mention statistics
// @Summary Get mention statistics
// @Description Returns statistics about mentions for the current user including unread counts by type
// @Tags Mentions
// @Produce json
// @Success 200 {object} models.MentionStats "Mention statistics"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /mentions/stats [get]
func (h *MentionsHandler) GetStats(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	stats, err := h.mentionService.GetStats(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get mention stats",
		})
	}

	return c.JSON(stats)
}

// MarkAsRead marks a single mention as read
// @Summary Mark mention as read
// @Description Marks a specific mention as read for the current user
// @Tags Mentions
// @Accept json
// @Produce json
// @Param id path string true "Mention ID"
// @Success 200 {object} fiber.Map "Success response"
// @Failure 400 {object} fiber.Map "Invalid mention ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 404 {object} fiber.Map "Mention not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /mentions/{id}/read [patch]
func (h *MentionsHandler) MarkAsRead(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	mentionID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid mention ID",
		})
	}

	if err := h.mentionService.MarkAsRead(c.Context(), mentionID, userID); err != nil {
		if err == services.ErrMentionNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "mention not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to mark mention as read",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
	})
}

// MarkAllAsRead marks all mentions as read
// @Summary Mark all mentions as read
// @Description Marks all mentions for the current user as read
// @Tags Mentions
// @Accept json
// @Produce json
// @Success 200 {object} fiber.Map "Success response with count of mentions marked as read"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /mentions/read-all [post]
func (h *MentionsHandler) MarkAllAsRead(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	count, err := h.mentionService.MarkAllAsRead(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to mark mentions as read",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"count":   count,
	})
}

// MarkChannelMentionsAsRead marks all mentions in a channel as read
// @Summary Mark channel mentions as read
// @Description Marks all mentions in a specific channel as read for the current user
// @Tags Mentions
// @Accept json
// @Produce json
// @Param channelId path string true "Channel ID"
// @Success 200 {object} fiber.Map "Success response with count of mentions marked as read"
// @Failure 400 {object} fiber.Map "Invalid channel ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /mentions/channel/{channelId}/read-all [post]
func (h *MentionsHandler) MarkChannelMentionsAsRead(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	channelID, err := uuid.Parse(c.Params("channelId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel ID",
		})
	}

	count, err := h.mentionService.MarkChannelMentionsAsRead(c.Context(), userID, channelID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to mark mentions as read",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"count":   count,
	})
}

// Search searches mentions
// @Summary Search mentions
// @Description Searches through user's mentions for matching content
// @Tags Mentions
// @Produce json
// @Param q query string true "Search query string"
// @Param limit query integer false "Maximum number of results (default: 20, max: 100)"
// @Success 200 {object} fiber.Map "Search results with mentions and count"
// @Failure 400 {object} fiber.Map "Invalid request or missing query"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /mentions/search [get]
func (h *MentionsHandler) Search(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	query := c.Query("q")
	if query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "search query is required",
		})
	}

	limit := c.QueryInt("limit", 20)

	mentions, err := h.mentionService.Search(c.Context(), userID, query, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to search mentions",
		})
	}

	return c.JSON(fiber.Map{
		"mentions": mentions,
		"count":    len(mentions),
	})
}
