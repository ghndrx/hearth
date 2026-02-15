package handlers

import (
	"context"
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// SavedMessagesServiceInterface defines the methods needed from SavedMessagesService
type SavedMessagesServiceInterface interface {
	SaveMessage(ctx context.Context, userID, messageID uuid.UUID, note *string) (*models.SavedMessage, error)
	GetSavedMessages(ctx context.Context, userID uuid.UUID, opts *models.SavedMessagesQueryOptions) ([]*models.SavedMessage, error)
	GetSavedMessage(ctx context.Context, userID, savedID uuid.UUID) (*models.SavedMessage, error)
	UpdateSavedMessageNote(ctx context.Context, userID, savedID uuid.UUID, note *string) (*models.SavedMessage, error)
	RemoveSavedMessage(ctx context.Context, userID, savedID uuid.UUID) error
	RemoveSavedMessageByMessageID(ctx context.Context, userID, messageID uuid.UUID) error
	IsSaved(ctx context.Context, userID, messageID uuid.UUID) (bool, error)
	GetSavedCount(ctx context.Context, userID uuid.UUID) (int, error)
}

// SavedMessagesHandler handles saved messages HTTP requests
type SavedMessagesHandler struct {
	service SavedMessagesServiceInterface
}

// NewSavedMessagesHandler creates a new saved messages handler
func NewSavedMessagesHandler(service SavedMessagesServiceInterface) *SavedMessagesHandler {
	return &SavedMessagesHandler{
		service: service,
	}
}

// SaveMessage saves/bookmarks a message for the current user
// POST /users/@me/saved-messages
func (h *SavedMessagesHandler) SaveMessage(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	var req models.SaveMessageRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.MessageID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "message_id is required",
		})
	}

	messageID, err := uuid.Parse(req.MessageID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid message_id format",
		})
	}

	// Validate note length if provided
	if req.Note != nil && len(*req.Note) > 500 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "note must be 500 characters or less",
		})
	}

	saved, err := h.service.SaveMessage(c.Context(), userID, messageID, req.Note)
	if err != nil {
		if errors.Is(err, services.ErrMessageNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "message not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to save message",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(saved)
}

// GetSavedMessages retrieves all saved messages for the current user
// GET /users/@me/saved-messages
func (h *SavedMessagesHandler) GetSavedMessages(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	opts := &models.SavedMessagesQueryOptions{
		Limit: 50,
	}

	// Parse query parameters
	if before := c.Query("before"); before != "" {
		if id, err := uuid.Parse(before); err == nil {
			opts.Before = &id
		}
	}
	if after := c.Query("after"); after != "" {
		if id, err := uuid.Parse(after); err == nil {
			opts.After = &id
		}
	}
	if limit := c.QueryInt("limit", 50); limit > 0 && limit <= 100 {
		opts.Limit = limit
	}

	saved, err := h.service.GetSavedMessages(c.Context(), userID, opts)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get saved messages",
		})
	}

	return c.JSON(saved)
}

// GetSavedMessage retrieves a specific saved message
// GET /users/@me/saved-messages/:id
func (h *SavedMessagesHandler) GetSavedMessage(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	savedID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid saved message id",
		})
	}

	saved, err := h.service.GetSavedMessage(c.Context(), userID, savedID)
	if err != nil {
		if errors.Is(err, services.ErrSavedMessageNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "saved message not found",
			})
		}
		if errors.Is(err, services.ErrUnauthorized) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "access denied",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get saved message",
		})
	}

	return c.JSON(saved)
}

// UpdateSavedMessage updates the note on a saved message
// PATCH /users/@me/saved-messages/:id
func (h *SavedMessagesHandler) UpdateSavedMessage(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	savedID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid saved message id",
		})
	}

	var req models.UpdateSavedMessageRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Validate note length if provided
	if req.Note != nil && len(*req.Note) > 500 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "note must be 500 characters or less",
		})
	}

	saved, err := h.service.UpdateSavedMessageNote(c.Context(), userID, savedID, req.Note)
	if err != nil {
		if errors.Is(err, services.ErrSavedMessageNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "saved message not found",
			})
		}
		if errors.Is(err, services.ErrUnauthorized) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "access denied",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to update saved message",
		})
	}

	return c.JSON(saved)
}

// RemoveSavedMessage removes a saved message by its ID
// DELETE /users/@me/saved-messages/:id
func (h *SavedMessagesHandler) RemoveSavedMessage(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	savedID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid saved message id",
		})
	}

	err = h.service.RemoveSavedMessage(c.Context(), userID, savedID)
	if err != nil {
		if errors.Is(err, services.ErrSavedMessageNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "saved message not found",
			})
		}
		if errors.Is(err, services.ErrUnauthorized) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "access denied",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to remove saved message",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// RemoveSavedMessageByMessage removes a saved message by the original message ID
// DELETE /users/@me/saved-messages/message/:messageId
func (h *SavedMessagesHandler) RemoveSavedMessageByMessage(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	messageID, err := uuid.Parse(c.Params("messageId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid message id",
		})
	}

	err = h.service.RemoveSavedMessageByMessageID(c.Context(), userID, messageID)
	if err != nil {
		if errors.Is(err, services.ErrSavedMessageNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "saved message not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to remove saved message",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// IsSaved checks if a message is saved by the current user
// GET /users/@me/saved-messages/check/:messageId
func (h *SavedMessagesHandler) IsSaved(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	messageID, err := uuid.Parse(c.Params("messageId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid message id",
		})
	}

	isSaved, err := h.service.IsSaved(c.Context(), userID, messageID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to check saved status",
		})
	}

	return c.JSON(fiber.Map{
		"saved": isSaved,
	})
}

// GetSavedCount returns the count of saved messages for the current user
// GET /users/@me/saved-messages/count
func (h *SavedMessagesHandler) GetSavedCount(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	count, err := h.service.GetSavedCount(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get saved count",
		})
	}

	return c.JSON(fiber.Map{
		"count": count,
	})
}
