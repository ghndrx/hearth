package handlers

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
)

// PushServiceInterface defines the methods needed from push services
type PushServiceInterface interface {
	RegisterSubscription(ctx context.Context, userID uuid.UUID, req *models.CreatePushSubscriptionRequest) error
	UnregisterSubscription(ctx context.Context, userID uuid.UUID, endpoint string) error
	GetPreferences(ctx context.Context, userID uuid.UUID) (*models.NotificationPreferences, error)
	UpdatePreferences(ctx context.Context, userID uuid.UUID, req *models.UpdateNotificationPreferencesRequest) (*models.NotificationPreferences, error)
}

// PushHandler handles push notification-related HTTP requests
type PushHandler struct {
	pushService PushServiceInterface
}

// NewPushHandler creates a new push handler
func NewPushHandler(pushService PushServiceInterface) *PushHandler {
	return &PushHandler{
		pushService: pushService,
	}
}

// RegisterSubscription registers a push subscription for the current user
func (h *PushHandler) RegisterSubscription(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

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

	err := h.pushService.RegisterSubscription(c.Context(), userID, &req)
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
func (h *PushHandler) UnregisterSubscription(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

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

	err := h.pushService.UnregisterSubscription(c.Context(), userID, req.Endpoint)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to unregister subscription",
		})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// GetPreferences returns the user's notification preferences
func (h *PushHandler) GetPreferences(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	prefs, err := h.pushService.GetPreferences(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get preferences",
		})
	}

	return c.JSON(prefs)
}

// UpdatePreferences updates the user's notification preferences
func (h *PushHandler) UpdatePreferences(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

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
