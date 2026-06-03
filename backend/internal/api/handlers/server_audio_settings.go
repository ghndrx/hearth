package handlers

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
)

// ServerAudioSettingsServiceInterface defines the methods needed from ServerAudioSettingsService
type ServerAudioSettingsServiceInterface interface {
	GetSettings(ctx context.Context, userID, serverID uuid.UUID) (*models.ServerAudioSettings, error)
	GetAllForUser(ctx context.Context, userID uuid.UUID) ([]*models.ServerAudioSettings, error)
	UpdateSettings(ctx context.Context, userID, serverID uuid.UUID, updates *models.UpdateServerAudioSettingsRequest) (*models.ServerAudioSettings, error)
	DeleteSettings(ctx context.Context, userID, serverID uuid.UUID) error
}

// ServerAudioSettingsHandler handles server audio settings HTTP requests
type ServerAudioSettingsHandler struct {
	service ServerAudioSettingsServiceInterface
}

// NewServerAudioSettingsHandler creates a new server audio settings handler
func NewServerAudioSettingsHandler(service ServerAudioSettingsServiceInterface) *ServerAudioSettingsHandler {
	return &ServerAudioSettingsHandler{service: service}
}

// GetSettings returns audio settings for the current user in a specific server
func (h *ServerAudioSettingsHandler) GetSettings(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server ID",
		})
	}

	settings, err := h.service.GetSettings(c.Context(), userID, serverID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get audio settings",
		})
	}

	return c.JSON(settings)
}

// GetAllSettings returns audio settings for the current user across all servers
func (h *ServerAudioSettingsHandler) GetAllSettings(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	settings, err := h.service.GetAllForUser(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get audio settings",
		})
	}

	return c.JSON(settings)
}

// UpdateSettings updates audio settings for the current user in a specific server
func (h *ServerAudioSettingsHandler) UpdateSettings(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server ID",
		})
	}

	var req models.UpdateServerAudioSettingsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Validate volumes
	if req.InputVolume != nil && (*req.InputVolume < 0 || *req.InputVolume > 100) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "input_volume must be between 0 and 100",
		})
	}
	if req.OutputVolume != nil && (*req.OutputVolume < 0 || *req.OutputVolume > 100) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "output_volume must be between 0 and 100",
		})
	}

	settings, err := h.service.UpdateSettings(c.Context(), userID, serverID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to update audio settings",
		})
	}

	return c.JSON(settings)
}

// DeleteSettings resets audio settings for the current user in a specific server
func (h *ServerAudioSettingsHandler) DeleteSettings(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server ID",
		})
	}

	if err := h.service.DeleteSettings(c.Context(), userID, serverID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to delete audio settings",
		})
	}

	return c.JSON(fiber.Map{"success": true})
}
