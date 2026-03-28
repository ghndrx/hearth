package handlers

import (
	"context"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// NotificationServiceInterface defines the methods needed from NotificationService
type NotificationServiceInterface interface {
	GetNotification(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*models.NotificationWithActor, error)
	ListNotifications(ctx context.Context, userID uuid.UUID, opts models.NotificationListOptions) ([]models.NotificationWithActor, error)
	GetNotificationStats(ctx context.Context, userID uuid.UUID) (*models.NotificationStats, error)
	MarkAsRead(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	MarkAllAsRead(ctx context.Context, userID uuid.UUID) (int64, error)
	DeleteNotification(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	DeleteAllReadNotifications(ctx context.Context, userID uuid.UUID) (int64, error)
}

// NotificationHandler handles notification-related HTTP requests
type NotificationHandler struct {
	notificationService NotificationServiceInterface
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
	userID := c.Locals("userID").(uuid.UUID)

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
	userID := c.Locals("userID").(uuid.UUID)

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
	userID := c.Locals("userID").(uuid.UUID)

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
	userID := c.Locals("userID").(uuid.UUID)

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
	userID := c.Locals("userID").(uuid.UUID)

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
	userID := c.Locals("userID").(uuid.UUID)

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
	userID := c.Locals("userID").(uuid.UUID)

	stats, err := h.notificationService.GetNotificationStats(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get notification stats",
		})
	}

	return c.JSON(stats)
}
