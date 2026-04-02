package handlers

import (
	"context"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
)

// SmartNotificationServiceInterface defines the methods needed from SmartNotificationService
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

// SmartNotificationHandler handles smart notification HTTP requests
type SmartNotificationHandler struct {
	service SmartNotificationServiceInterface
}

// NewSmartNotificationHandler creates a new smart notification handler
func NewSmartNotificationHandler(service SmartNotificationServiceInterface) *SmartNotificationHandler {
	return &SmartNotificationHandler{service: service}
}

// ScoreNotification scores a notification for priority
// @Summary Score a notification
// @Tags Smart Notifications
// @Accept json
// @Produce json
// @Success 200 {object} models.SmartNotification
// @Router /users/@me/notifications/score [post]
func (h *SmartNotificationHandler) ScoreNotification(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	var input models.PriorityScoringInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}
	input.RecipientID = userID

	result, err := h.service.ScoreNotification(c.Context(), &input)
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
func (h *SmartNotificationHandler) SnoozeNotifications(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

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

	config, err := h.service.SnoozeNotifications(c.Context(), userID, &req)
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
func (h *SmartNotificationHandler) UnsnoozeNotifications(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

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

	if err := h.service.UnsnoozeNotifications(c.Context(), userID, serverID, channelID); err != nil {
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
func (h *SmartNotificationHandler) GetSnoozeStatus(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

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

	snoozed, until, err := h.service.IsNotificationSnoozed(c.Context(), userID, serverID, channelID)
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
func (h *SmartNotificationHandler) MuteNotifications(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	var req models.MuteRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	config, err := h.service.MuteNotifications(c.Context(), userID, &req)
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
func (h *SmartNotificationHandler) GetMuteStatus(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

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

	muted, err := h.service.IsNotificationMuted(c.Context(), userID, serverID, channelID)
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
func (h *SmartNotificationHandler) TrackClick(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	notificationID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid notification id",
		})
	}

	if err := h.service.TrackNotificationClick(c.Context(), userID, notificationID); err != nil {
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
func (h *SmartNotificationHandler) DismissNotification(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	if err := h.service.TrackNotificationDismissed(c.Context(), userID); err != nil {
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
func (h *SmartNotificationHandler) GetEngagement(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	engagement, err := h.service.GetUserEngagement(c.Context(), userID)
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
func (h *SmartNotificationHandler) GetPreferences(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	prefs := h.service.GetUserPreferences(c.Context(), userID)
	return c.JSON(prefs)
}

// UpdatePreferences updates smart notification preferences
// @Summary Update smart notification preferences
// @Tags Smart Notifications
// @Accept json
// @Produce json
// @Success 200 {object} models.SmartNotificationPreferences
// @Router /users/@me/notifications/preferences [patch]
func (h *SmartNotificationHandler) UpdatePreferences(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

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

	if err := h.service.UpdateUserPreferences(c.Context(), userID, &prefs); err != nil {
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
func (h *SmartNotificationHandler) ListDigests(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

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

	digests, err := h.service.ListDigests(c.Context(), userID, opts)
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
func (h *SmartNotificationHandler) GetDigest(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	digestID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid digest id",
		})
	}

	digest, err := h.service.GetDigest(c.Context(), userID, digestID)
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
func (h *SmartNotificationHandler) MarkDigestRead(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	digestID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid digest id",
		})
	}

	if err := h.service.MarkDigestRead(c.Context(), userID, digestID); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "digest not found",
		})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}
