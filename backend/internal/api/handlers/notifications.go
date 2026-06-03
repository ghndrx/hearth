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

type NotificationServiceInterface interface {
	GetNotification(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*models.NotificationWithActor, error)
	ListNotifications(ctx context.Context, userID uuid.UUID, opts models.NotificationListOptions) ([]models.NotificationWithActor, error)
	GetNotificationStats(ctx context.Context, userID uuid.UUID) (*models.NotificationStats, error)
	MarkAsRead(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	MarkAllAsRead(ctx context.Context, userID uuid.UUID) (int64, error)
	DeleteNotification(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	DeleteAllReadNotifications(ctx context.Context, userID uuid.UUID) (int64, error)
}

type PushServiceInterface interface {
	RegisterSubscription(ctx context.Context, userID uuid.UUID, req *models.CreatePushSubscriptionRequest) error
	UnregisterSubscription(ctx context.Context, userID uuid.UUID, endpoint string) error
	GetPreferences(ctx context.Context, userID uuid.UUID) (*models.NotificationPreferences, error)
	UpdatePreferences(ctx context.Context, userID uuid.UUID, req *models.UpdateNotificationPreferencesRequest) (*models.NotificationPreferences, error)
}

type ReadStateServiceInterface interface {
	MarkChannelAsRead(ctx context.Context, userID, channelID uuid.UUID, messageID *uuid.UUID) (*models.AckResponse, error)
	GetChannelReadState(ctx context.Context, userID, channelID uuid.UUID) (*models.ReadState, error)
	GetChannelUnreadInfo(ctx context.Context, userID, channelID uuid.UUID) (*models.ChannelUnreadInfo, error)
	GetUnreadSummary(ctx context.Context, userID uuid.UUID) (*models.UnreadSummary, error)
	GetServerUnreadSummary(ctx context.Context, userID, serverID uuid.UUID) (*models.UnreadSummary, error)
	MarkServerAsRead(ctx context.Context, userID, serverID uuid.UUID) error
}

type SmartNotificationServiceInterface interface {
	ScoreNotification(ctx context.Context, input *models.PriorityScoringInput) (*models.SmartNotification, error)
	SnoozeNotifications(ctx context.Context, userID uuid.UUID, req *models.SnoozeRequest) (*models.SnoozeConfig, error)
	UnsnoozeNotifications(ctx context.Context, userID uuid.UUID, serverID, channelID *uuid.UUID) error
	IsNotificationSnoozed(ctx context.Context, userID uuid.UUID, serverID, channelID *uuid.UUID) (bool, *time.Time, error)
	MuteNotifications(ctx context.Context, userID uuid.UUID, req *models.MuteRequest) (*models.MuteConfig, error)
	IsNotificationMuted(ctx context.Context, userID uuid.UUID, serverID, channelID *uuid.UUID) (bool, error)
	TrackNotificationClick(ctx context.Context, userID uuid.UUID, notificationID uuid.UUID) error
	TrackNotificationDismissed(ctx context.Context, userID uuid.UUID) error
	GetUserEngagement(ctx context.Context, userID uuid.UUID) (*models.UserEngagement, error)
	GetUserPreferences(ctx context.Context, userID uuid.UUID) *models.SmartNotificationPreferences
	UpdateUserPreferences(ctx context.Context, userID uuid.UUID, prefs *models.SmartNotificationPreferences) error
	GetDigest(ctx context.Context, userID uuid.UUID, digestID uuid.UUID) (*models.NotificationDigest, error)
	ListDigests(ctx context.Context, userID uuid.UUID, opts models.DigestListOptions) ([]models.NotificationDigest, error)
	MarkDigestRead(ctx context.Context, userID uuid.UUID, digestID uuid.UUID) error
	RouteNotification(ctx context.Context, userID uuid.UUID, smartNotif *models.SmartNotification) (*models.SmartNotification, error)
}

// NotificationHandler handles all notification-related HTTP requests
type NotificationHandler struct {
	notificationService NotificationServiceInterface
	pushService         PushServiceInterface
	mentionService      *services.MentionAPIService
	readStateService    ReadStateServiceInterface
	smartNotifService   SmartNotificationServiceInterface
}

// NewNotificationHandler creates a new notification handler
func NewNotificationHandler(notificationService NotificationServiceInterface) *NotificationHandler {
	return &NotificationHandler{
		notificationService: notificationService,
	}
}

// GetNotifications returns the current user's notifications
// @Summary Get user's notifications
// @Description Returns a list of the current user's notifications with optional filtering
// @Tags Notifications
// @Produce json
// @Param limit query int false "Number of notifications to return (max 100, default 50)"
// @Param offset query int false "Offset for pagination (default 0)"
// @Param unread query bool false "Filter by unread status"
// @Success 200 {object} fiber.Map "notifications, total count, and unread count"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /users/@me/notifications [get]
func (h *NotificationHandler) GetNotifications(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	// Parse query parameters
	opts := models.NotificationListOptions{
		Limit:  50,
		Offset: 0,
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 100 {
			opts.Limit = limit
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			opts.Offset = offset
		}
	}

	if unreadStr := c.Query("unread"); unreadStr != "" {
		unread := unreadStr == "true"
		opts.Unread = &unread
	}

	notifications, err := h.notificationService.ListNotifications(c.Context(), userID, opts)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get notifications",
		})
	}

	// Get stats as well
	stats, err := h.notificationService.GetNotificationStats(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get notification stats",
		})
	}

	return c.JSON(fiber.Map{
		"notifications": notifications,
		"total":         stats.Total,
		"unread":        stats.Unread,
	})
}

// GetNotification returns a specific notification
// @Summary Get a notification
// @Description Returns a specific notification by ID
// @Tags Notifications
// @Produce json
// @Param id path string true "Notification ID (UUID)"
// @Success 200 {object} models.NotificationWithActor "Notification details"
// @Failure 400 {object} fiber.Map "Invalid notification ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 404 {object} fiber.Map "Notification not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /users/@me/notifications/{id} [get]
func (h *NotificationHandler) GetNotification(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	notificationID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid notification id",
		})
	}

	notification, err := h.notificationService.GetNotification(c.Context(), notificationID, userID)
	if err != nil {
		if err == services.ErrNotificationNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "notification not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get notification",
		})
	}

	return c.JSON(notification)
}

// MarkAsRead marks a notification as read
// @Summary Mark notification as read
// @Description Marks a specific notification as read
// @Tags Notifications
// @Param id path string true "Notification ID (UUID)"
// @Success 204 "Notification marked as read"
// @Failure 400 {object} fiber.Map "Invalid notification ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 404 {object} fiber.Map "Notification not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /users/@me/notifications/{id}/read [post]
func (h *NotificationHandler) MarkAsRead(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	notificationID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid notification id",
		})
	}

	err = h.notificationService.MarkAsRead(c.Context(), notificationID, userID)
	if err != nil {
		if err == services.ErrNotificationNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "notification not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to mark notification as read",
		})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// MarkAllAsRead marks all notifications as read
// @Summary Mark all notifications as read
// @Description Marks all of the current user's notifications as read
// @Tags Notifications
// @Produce json
// @Success 200 {object} fiber.Map "Number of notifications marked as read"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /users/@me/notifications/read-all [post]
func (h *NotificationHandler) MarkAllAsRead(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	count, err := h.notificationService.MarkAllAsRead(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to mark all notifications as read",
		})
	}

	return c.JSON(fiber.Map{
		"marked": count,
	})
}

// DeleteNotification deletes a notification
// @Summary Delete a notification
// @Description Deletes a specific notification by ID
// @Tags Notifications
// @Param id path string true "Notification ID (UUID)"
// @Success 204 "Notification deleted"
// @Failure 400 {object} fiber.Map "Invalid notification ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 404 {object} fiber.Map "Notification not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /users/@me/notifications/{id} [delete]
func (h *NotificationHandler) DeleteNotification(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	notificationID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid notification id",
		})
	}

	err = h.notificationService.DeleteNotification(c.Context(), notificationID, userID)
	if err != nil {
		if err == services.ErrNotificationNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "notification not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to delete notification",
		})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// DeleteAllRead deletes all read notifications
// @Summary Delete all read notifications
// @Description Deletes all of the current user's read notifications
// @Tags Notifications
// @Produce json
// @Success 200 {object} fiber.Map "Number of notifications deleted"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /users/@me/notifications/read [delete]
func (h *NotificationHandler) DeleteAllRead(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	count, err := h.notificationService.DeleteAllReadNotifications(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to delete read notifications",
		})
	}

	return c.JSON(fiber.Map{
		"deleted": count,
	})
}

// GetNotificationStats returns notification statistics
// @Summary Get notification statistics
// @Description Returns statistics for the current user's notifications
// @Tags Notifications
// @Produce json
// @Success 200 {object} models.NotificationStats "Total and unread notification counts"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /users/@me/notifications/stats [get]
func (h *NotificationHandler) GetNotificationStats(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	stats, err := h.notificationService.GetNotificationStats(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get notification stats",
		})
	}

	return c.JSON(stats)
}


// RegisterSubscription registers a push subscription for the current user
func (h *NotificationHandler) RegisterSubscription(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	var req models.CreatePushSubscriptionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Validate required fields
	if req.Endpoint == "" || req.P256dh == "" || req.Auth == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "endpoint, p256dh, and auth are required",
		})
	}

	// Get user agent from request
	req.UserAgent = c.Get("User-Agent")

	err = h.pushService.RegisterSubscription(c.Context(), userID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to register subscription",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "subscription registered",
	})
}

// UnregisterSubscription removes a push subscription
func (h *NotificationHandler) UnregisterSubscription(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	var req struct {
		Endpoint string `json:"endpoint"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.Endpoint == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "endpoint is required",
		})
	}

	err = h.pushService.UnregisterSubscription(c.Context(), userID, req.Endpoint)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to unregister subscription",
		})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// GetPreferences returns the user's notification preferences
func (h *NotificationHandler) GetPushPreferences(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	prefs, err := h.pushService.GetPreferences(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get preferences",
		})
	}

	return c.JSON(prefs)
}

// UpdatePreferences updates the user's notification preferences
func (h *NotificationHandler) UpdatePushPreferences(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	var req models.UpdateNotificationPreferencesRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	prefs, err := h.pushService.UpdatePreferences(c.Context(), userID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to update preferences",
		})
	}

	return c.JSON(prefs)
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
func (h *NotificationHandler) GetMentions(c *fiber.Ctx) error {
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
func (h *NotificationHandler) GetUnreadCount(c *fiber.Ctx) error {
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
func (h *NotificationHandler) GetStats(c *fiber.Ctx) error {
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
func (h *NotificationHandler) MarkMentionAsRead(c *fiber.Ctx) error {
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
func (h *NotificationHandler) MarkAllMentionsAsRead(c *fiber.Ctx) error {
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
func (h *NotificationHandler) MarkChannelMentionsAsRead(c *fiber.Ctx) error {
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
func (h *NotificationHandler) Search(c *fiber.Ctx) error {
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


// MarkChannelAsRead marks a channel as read
// POST /channels/:id/ack
// @Summary Mark channel as read
// @Description Marks a channel as read up to a specific message ID, or the latest message if no ID provided
// @Tags ReadState
// @Accept json
// @Produce json
// @Param id path string true "Channel ID"
// @Param body body models.MarkReadRequest false "Optional message ID to mark as read up to"
// @Success 200 {object} models.AckResponse "Channel marked as read successfully"
// @Failure 400 {object} fiber.Map "Invalid channel ID or request body"
// @Failure 404 {object} fiber.Map "Channel not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{id}/ack [post]
func (h *NotificationHandler) MarkChannelAsRead(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	channelID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	// Parse optional message ID from body
	var req models.MarkReadRequest
	if err := c.BodyParser(&req); err != nil && len(c.Body()) > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	ack, err := h.readStateService.MarkChannelAsRead(c.Context(), userID, channelID, req.MessageID)
	if err != nil {
		if err == services.ErrChannelNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "channel not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to mark channel as read",
		})
	}

	return c.JSON(ack)
}

// GetChannelUnread gets the unread information for a channel
// GET /channels/:id/unread
// @Summary Get channel unread info
// @Description Returns unread message count and mention count for a specific channel
// @Tags ReadState
// @Produce json
// @Param id path string true "Channel ID"
// @Success 200 {object} models.ChannelUnreadInfo "Unread information for the channel"
// @Failure 400 {object} fiber.Map "Invalid channel ID"
// @Failure 404 {object} fiber.Map "Channel not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{id}/unread [get]
func (h *NotificationHandler) GetChannelUnread(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	channelID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	info, err := h.readStateService.GetChannelUnreadInfo(c.Context(), userID, channelID)
	if err != nil {
		if err == services.ErrChannelNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "channel not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get unread info",
		})
	}

	return c.JSON(info)
}

// GetUnreadSummary gets the unread summary for all channels
// GET /users/@me/unread
// @Summary Get unread summary
// @Description Returns a summary of unread messages and mentions across all channels for the current user
// @Tags ReadState
// @Produce json
// @Success 200 {object} models.UnreadSummary "Unread summary for all channels"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /users/@me/unread [get]
func (h *NotificationHandler) GetUnreadSummary(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	summary, err := h.readStateService.GetUnreadSummary(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get unread summary",
		})
	}

	return c.JSON(summary)
}

// GetServerUnread gets the unread summary for a server
// GET /servers/:id/unread
// @Summary Get server unread summary
// @Description Returns a summary of unread messages and mentions across all channels in a specific server
// @Tags ReadState
// @Produce json
// @Param id path string true "Server ID"
// @Success 200 {object} models.UnreadSummary "Unread summary for the server"
// @Failure 400 {object} fiber.Map "Invalid server ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/unread [get]
func (h *NotificationHandler) GetServerUnread(c *fiber.Ctx) error {
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

	summary, err := h.readStateService.GetServerUnreadSummary(c.Context(), userID, serverID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get server unread summary",
		})
	}

	return c.JSON(summary)
}

// MarkServerAsRead marks all channels in a server as read
// POST /servers/:id/ack
// @Summary Mark server as read
// @Description Marks all channels in a server as read for the current user
// @Tags ReadState
// @Param id path string true "Server ID"
// @Success 204 "Server marked as read successfully"
// @Failure 400 {object} fiber.Map "Invalid server ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/ack [post]
func (h *NotificationHandler) MarkServerAsRead(c *fiber.Ctx) error {
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

	err = h.readStateService.MarkServerAsRead(c.Context(), userID, serverID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to mark server as read",
		})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}


// ScoreNotification scores a notification for priority
// @Summary Score a notification
// @Tags Smart Notifications
// @Accept json
// @Produce json
// @Success 200 {object} models.SmartNotification
// @Router /users/@me/notifications/score [post]
func (h *NotificationHandler) ScoreNotification(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	var input models.PriorityScoringInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}
	input.RecipientID = userID

	result, err := h.smartNotifService.ScoreNotification(c.Context(), &input)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to score notification",
		})
	}

	return c.JSON(result)
}

// SnoozeNotifications snoozes notifications
// @Summary Snooze notifications
// @Tags Smart Notifications
// @Accept json
// @Produce json
// @Success 200 {object} models.SnoozeConfig
// @Router /users/@me/notifications/snooze [post]
func (h *NotificationHandler) SnoozeNotifications(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	var req models.SnoozeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.DurationMins < 1 || req.DurationMins > 10080 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "duration must be between 1 and 10080 minutes",
		})
	}

	config, err := h.smartNotifService.SnoozeNotifications(c.Context(), userID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to snooze notifications",
		})
	}

	return c.JSON(config)
}

// UnsnoozeNotifications removes a snooze
// @Summary Unsnooze notifications
// @Tags Smart Notifications
// @Success 204
// @Router /users/@me/notifications/snooze [delete]
func (h *NotificationHandler) UnsnoozeNotifications(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	var serverID, channelID *uuid.UUID
	if s := c.Query("server_id"); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			serverID = &id
		}
	}
	if ch := c.Query("channel_id"); ch != "" {
		if id, err := uuid.Parse(ch); err == nil {
			channelID = &id
		}
	}

	if err := h.smartNotifService.UnsnoozeNotifications(c.Context(), userID, serverID, channelID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to unsnooze notifications",
		})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// GetSnoozeStatus returns the current snooze status
// @Summary Get snooze status
// @Tags Smart Notifications
// @Produce json
// @Success 200 {object} fiber.Map
// @Router /users/@me/notifications/snooze [get]
func (h *NotificationHandler) GetSnoozeStatus(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	var serverID, channelID *uuid.UUID
	if s := c.Query("server_id"); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			serverID = &id
		}
	}
	if ch := c.Query("channel_id"); ch != "" {
		if id, err := uuid.Parse(ch); err == nil {
			channelID = &id
		}
	}

	snoozed, until, err := h.smartNotifService.IsNotificationSnoozed(c.Context(), userID, serverID, channelID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get snooze status",
		})
	}

	return c.JSON(fiber.Map{
		"snoozed": snoozed,
		"until":   until,
	})
}

// MuteNotifications mutes/unmutes notifications
// @Summary Mute or unmute notifications
// @Tags Smart Notifications
// @Accept json
// @Produce json
// @Success 200 {object} models.MuteConfig
// @Router /users/@me/notifications/mute [post]
func (h *NotificationHandler) MuteNotifications(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	var req models.MuteRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	config, err := h.smartNotifService.MuteNotifications(c.Context(), userID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to update mute settings",
		})
	}

	return c.JSON(config)
}

// GetMuteStatus returns the current mute status
// @Summary Get mute status
// @Tags Smart Notifications
// @Produce json
// @Success 200 {object} fiber.Map
// @Router /users/@me/notifications/mute [get]
func (h *NotificationHandler) GetMuteStatus(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	var serverID, channelID *uuid.UUID
	if s := c.Query("server_id"); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			serverID = &id
		}
	}
	if ch := c.Query("channel_id"); ch != "" {
		if id, err := uuid.Parse(ch); err == nil {
			channelID = &id
		}
	}

	muted, err := h.smartNotifService.IsNotificationMuted(c.Context(), userID, serverID, channelID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get mute status",
		})
	}

	return c.JSON(fiber.Map{
		"muted": muted,
	})
}

// TrackClick records a notification click
// @Summary Track notification click
// @Tags Smart Notifications
// @Success 204
// @Router /users/@me/notifications/{id}/click [post]
func (h *NotificationHandler) TrackClick(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	notificationID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid notification id",
		})
	}

	if err := h.smartNotifService.TrackNotificationClick(c.Context(), userID, notificationID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to track click",
		})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// DismissNotification tracks a notification dismissal
// @Summary Dismiss notification
// @Tags Smart Notifications
// @Success 204
// @Router /users/@me/notifications/{id}/dismiss [post]
func (h *NotificationHandler) DismissNotification(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	if err := h.smartNotifService.TrackNotificationDismissed(c.Context(), userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to track dismissal",
		})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// GetEngagement returns engagement stats
// @Summary Get notification engagement stats
// @Tags Smart Notifications
// @Produce json
// @Success 200 {object} models.UserEngagement
// @Router /users/@me/notifications/engagement [get]
func (h *NotificationHandler) GetEngagement(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	engagement, err := h.smartNotifService.GetUserEngagement(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get engagement stats",
		})
	}

	return c.JSON(engagement)
}

// GetPreferences returns smart notification preferences
// @Summary Get smart notification preferences
// @Tags Smart Notifications
// @Produce json
// @Success 200 {object} models.SmartNotificationPreferences
// @Router /users/@me/notifications/preferences [get]
func (h *NotificationHandler) GetSmartNotificationPreferences(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
	prefs := h.smartNotifService.GetUserPreferences(c.Context(), userID)
	return c.JSON(prefs)
}

// UpdatePreferences updates smart notification preferences
// @Summary Update smart notification preferences
// @Tags Smart Notifications
// @Accept json
// @Produce json
// @Success 200 {object} models.SmartNotificationPreferences
// @Router /users/@me/notifications/preferences [patch]
func (h *NotificationHandler) UpdateSmartNotificationPreferences(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	var prefs models.SmartNotificationPreferences
	if err := c.BodyParser(&prefs); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if prefs.DigestIntervalMins < 5 {
		prefs.DigestIntervalMins = 5
	}
	if prefs.DigestIntervalMins > 1440 {
		prefs.DigestIntervalMins = 1440
	}

	if err := h.smartNotifService.UpdateUserPreferences(c.Context(), userID, &prefs); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to update preferences",
		})
	}

	return c.JSON(prefs)
}

// ListDigests returns notification digests
// @Summary List notification digests
// @Tags Smart Notifications
// @Produce json
// @Success 200 {array} models.NotificationDigest
// @Router /users/@me/notifications/digests [get]
func (h *NotificationHandler) ListDigests(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	opts := models.DigestListOptions{
		Limit:  20,
		Offset: 0,
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 100 {
			opts.Limit = limit
		}
	}
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			opts.Offset = offset
		}
	}
	if unreadStr := c.Query("unread"); unreadStr != "" {
		unread := unreadStr == "true"
		opts.Unread = &unread
	}

	digests, err := h.smartNotifService.ListDigests(c.Context(), userID, opts)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to list digests",
		})
	}

	if digests == nil {
		digests = []models.NotificationDigest{}
	}

	return c.JSON(fiber.Map{
		"digests": digests,
	})
}

// GetDigest returns a specific digest
// @Summary Get a notification digest
// @Tags Smart Notifications
// @Produce json
// @Success 200 {object} models.NotificationDigest
// @Router /users/@me/notifications/digests/{id} [get]
func (h *NotificationHandler) GetDigest(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	digestID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid digest id",
		})
	}

	digest, err := h.smartNotifService.GetDigest(c.Context(), userID, digestID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "digest not found",
		})
	}

	return c.JSON(digest)
}

// MarkDigestRead marks a digest as read
// @Summary Mark digest as read
// @Tags Smart Notifications
// @Success 204
// @Router /users/@me/notifications/digests/{id}/read [post]
func (h *NotificationHandler) MarkDigestRead(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	digestID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid digest id",
		})
	}

	if err := h.smartNotifService.MarkDigestRead(c.Context(), userID, digestID); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "digest not found",
		})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

