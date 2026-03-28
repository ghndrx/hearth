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
// @Summary Get user settings
// @Description Returns the current authenticated user's settings
// @Tags Settings
// @Produce json
// @Success 200 {object} models.UserSettings "User settings"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /users/@me/settings [get]
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
// @Summary Update user settings
// @Description Updates the current user's application settings (theme, display, notifications, privacy, locale)
// @Tags Settings
// @Accept json
// @Produce json
// @Param body body models.UpdateUserSettingsRequest true "Settings update request"
// @Success 200 {object} models.UserSettings "Updated user settings"
// @Failure 400 {object} fiber.Map "Invalid request body or validation error"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /users/@me/settings [patch]
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
// @Summary Reset user settings
// @Description Resets the current user's settings to their default values
// @Tags Settings
// @Produce json
// @Success 200 {object} models.UserSettings "Reset user settings with default values"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /users/@me/settings [delete]
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
