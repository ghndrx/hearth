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
// @Summary Get unread summary
// @Description Returns a summary of unread messages and mentions across all channels for the current user
// @Tags ReadState
// @Produce json
// @Success 200 {object} models.UnreadSummary "Unread summary for all channels"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /users/@me/unread [get]
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
// @Summary Mark server as read
// @Description Marks all channels in a server as read for the current user
// @Tags ReadState
// @Param id path string true "Server ID"
// @Success 204 "Server marked as read successfully"
// @Failure 400 {object} fiber.Map "Invalid server ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/ack [post]
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
