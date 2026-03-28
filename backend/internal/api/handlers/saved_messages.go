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
// @Summary Save a message
// @Description Saves/bookmarks a message for the current user with an optional note
// @Tags Saved Messages
// @Accept json
// @Produce json
// @Param body body models.SaveMessageRequest true "Message to save"
// @Success 201 {object} models.SavedMessage "Message saved successfully"
// @Failure 400 {object} fiber.Map "Invalid request body, missing message_id, invalid format, or note too long"
// @Failure 404 {object} fiber.Map "Message not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /users/@me/saved-messages [post]
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
// @Summary Get saved messages
// @Description Retrieves all saved messages for the current user with optional pagination
// @Tags Saved Messages
// @Produce json
// @Param before query string false "UUID to get messages before (for pagination)"
// @Param after query string false "UUID to get messages after (for pagination)"
// @Param limit query int false "Number of messages to return (default 50, max 100)"
// @Success 200 {array} models.SavedMessage "List of saved messages"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /users/@me/saved-messages [get]
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
// @Summary Get a saved message
// @Description Retrieves a specific saved message by its ID
// @Tags Saved Messages
// @Produce json
// @Param id path string true "Saved message ID"
// @Success 200 {object} models.SavedMessage "Saved message details"
// @Failure 400 {object} fiber.Map "Invalid saved message ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Access denied"
// @Failure 404 {object} fiber.Map "Saved message not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /users/@me/saved-messages/{id} [get]
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
// @Summary Update saved message note
// @Description Updates the note on an existing saved message
// @Tags Saved Messages
// @Accept json
// @Produce json
// @Param id path string true "Saved message ID"
// @Param body body models.UpdateSavedMessageRequest true "Updated note data"
// @Success 200 {object} models.SavedMessage "Saved message updated successfully"
// @Failure 400 {object} fiber.Map "Invalid saved message ID, invalid request body, or note too long"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Access denied"
// @Failure 404 {object} fiber.Map "Saved message not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /users/@me/saved-messages/{id} [patch]
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
// @Summary Remove saved message
// @Description Removes a saved message by its ID
// @Tags Saved Messages
// @Param id path string true "Saved message ID"
// @Success 204 "Saved message removed successfully"
// @Failure 400 {object} fiber.Map "Invalid saved message ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Access denied"
// @Failure 404 {object} fiber.Map "Saved message not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /users/@me/saved-messages/{id} [delete]
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
// @Summary Remove saved message by message ID
// @Description Removes a saved message by the original message ID
// @Tags Saved Messages
// @Param messageId path string true "Original message ID"
// @Success 204 "Saved message removed successfully"
// @Failure 400 {object} fiber.Map "Invalid message ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 404 {object} fiber.Map "Saved message not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /users/@me/saved-messages/message/{messageId} [delete]
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
// @Summary Check if message is saved
// @Description Checks if a specific message is saved by the current user
// @Tags Saved Messages
// @Produce json
// @Param messageId path string true "Message ID to check"
// @Success 200 {object} fiber.Map "Object with saved boolean"
// @Failure 400 {object} fiber.Map "Invalid message ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /users/@me/saved-messages/check/{messageId} [get]
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
// @Summary Get saved message count
// @Description Returns the total count of saved messages for the current user
// @Tags Saved Messages
// @Produce json
// @Success 200 {object} fiber.Map "Object with count integer"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /users/@me/saved-messages/count [get]
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
