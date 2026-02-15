package handlers

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
)

// SettingsServiceInterface defines the methods needed from SettingsService
type SettingsServiceInterface interface {
	GetSettings(ctx context.Context, userID uuid.UUID) (*models.UserSettings, error)
	UpdateSettings(ctx context.Context, userID uuid.UUID, updates *models.UpdateUserSettingsRequest) (*models.UserSettings, error)
	ResetSettings(ctx context.Context, userID uuid.UUID) (*models.UserSettings, error)
}

// SettingsHandler handles settings-related HTTP requests
type SettingsHandler struct {
	settingsService SettingsServiceInterface
}

// NewSettingsHandler creates a new settings handler
func NewSettingsHandler(settingsService SettingsServiceInterface) *SettingsHandler {
	return &SettingsHandler{
		settingsService: settingsService,
	}
}

// GetSettings returns the current user's settings
func (h *SettingsHandler) GetSettings(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	settings, err := h.settingsService.GetSettings(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get settings",
		})
	}

	return c.JSON(settings)
}

// UpdateSettings updates the current user's settings
func (h *SettingsHandler) UpdateSettings(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	var req models.UpdateUserSettingsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Validate theme if provided
	if req.Theme != nil {
		validThemes := map[string]bool{"dark": true, "light": true, "system": true}
		if !validThemes[*req.Theme] {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid theme: must be 'dark', 'light', or 'system'",
			})
		}
	}

	// Validate message display if provided
	if req.MessageDisplay != nil {
		validDisplays := map[string]bool{"cozy": true, "compact": true}
		if !validDisplays[*req.MessageDisplay] {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid message_display: must be 'cozy' or 'compact'",
			})
		}
	}

	// Validate locale if provided
	if req.Locale != nil {
		if len(*req.Locale) < 2 || len(*req.Locale) > 10 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid locale: must be 2-10 characters",
			})
		}
	}

	settings, err := h.settingsService.UpdateSettings(c.Context(), userID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to update settings",
		})
	}

	return c.JSON(settings)
}

// ResetSettings resets the current user's settings to defaults
func (h *SettingsHandler) ResetSettings(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	settings, err := h.settingsService.ResetSettings(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to reset settings",
		})
	}

	return c.JSON(settings)
}
