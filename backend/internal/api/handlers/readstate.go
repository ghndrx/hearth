package handlers

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// ReadStateServiceInterface defines the methods needed from ReadStateService
type ReadStateServiceInterface interface {
	MarkChannelAsRead(ctx context.Context, userID, channelID uuid.UUID, messageID *uuid.UUID) (*models.AckResponse, error)
	GetChannelReadState(ctx context.Context, userID, channelID uuid.UUID) (*models.ReadState, error)
	GetChannelUnreadInfo(ctx context.Context, userID, channelID uuid.UUID) (*models.ChannelUnreadInfo, error)
	GetUnreadSummary(ctx context.Context, userID uuid.UUID) (*models.UnreadSummary, error)
	GetServerUnreadSummary(ctx context.Context, userID, serverID uuid.UUID) (*models.UnreadSummary, error)
	MarkServerAsRead(ctx context.Context, userID, serverID uuid.UUID) error
}

// ReadStateHandler handles read state HTTP requests
type ReadStateHandler struct {
	readStateService ReadStateServiceInterface
}

// NewReadStateHandler creates a new read state handler
func NewReadStateHandler(readStateService ReadStateServiceInterface) *ReadStateHandler {
	return &ReadStateHandler{
		readStateService: readStateService,
	}
}

// MarkChannelAsRead marks a channel as read
// POST /channels/:id/ack
func (h *ReadStateHandler) MarkChannelAsRead(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

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
func (h *ReadStateHandler) GetChannelUnread(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

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
func (h *ReadStateHandler) GetUnreadSummary(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

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
func (h *ReadStateHandler) GetServerUnread(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

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
func (h *ReadStateHandler) MarkServerAsRead(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

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
