package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

type ChannelHandler struct {
	channelService *services.ChannelService
	messageService *services.MessageService
}

func NewChannelHandler(channelService *services.ChannelService, messageService *services.MessageService) *ChannelHandler {
	return &ChannelHandler{
		channelService: channelService,
		messageService: messageService,
	}
}

// Get returns a channel by ID
func (h *ChannelHandler) Get(c *fiber.Ctx) error {
	channelID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	channel, err := h.channelService.GetChannel(c.Context(), channelID)
	if err != nil {
		if err == services.ErrChannelNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "channel not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get channel",
		})
	}

	return c.JSON(channel)
}

// Update updates a channel
func (h *ChannelHandler) Update(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	channelID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	var req models.UpdateChannelRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Convert request to service update struct
	update := &models.ChannelUpdate{
		Name:     req.Name,
		Topic:    req.Topic,
		Position: req.Position,
		NSFW:     req.NSFW,
		Slowmode: req.SlowmodeSeconds,
	}

	channel, err := h.channelService.UpdateChannel(c.Context(), channelID, userID, update)
	if err != nil {
		switch err {
		case services.ErrChannelNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "channel not found",
			})
		case services.ErrNotServerMember:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "not a server member",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to update channel",
			})
		}
	}

	return c.JSON(channel)
}

// Delete deletes a channel
func (h *ChannelHandler) Delete(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	channelID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	err = h.channelService.DeleteChannel(c.Context(), channelID, userID)
	if err != nil {
		switch err {
		case services.ErrChannelNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "channel not found",
			})
		case services.ErrNotServerMember:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "not a server member",
			})
		case services.ErrCannotDeleteDM:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "cannot delete DM channel",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to delete channel",
			})
		}
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// GetMessages returns messages with pagination
func (h *ChannelHandler) GetMessages(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	channelID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	var before, after *uuid.UUID
	if b := c.Query("before"); b != "" {
		if id, err := uuid.Parse(b); err == nil {
			before = &id
		}
	}
	if a := c.Query("after"); a != "" {
		if id, err := uuid.Parse(a); err == nil {
			after = &id
		}
	}

	limit := c.QueryInt("limit", 50)

	messages, err := h.messageService.GetMessages(c.Context(), channelID, userID, before, after, limit)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(messages)
}

// SendMessage sends a message
func (h *ChannelHandler) SendMessage(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	channelID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	var req struct {
		Content string     `json:"content"`
		ReplyTo *uuid.UUID `json:"reply_to"`
		// Attachments handled separately via multipart
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	message, err := h.messageService.SendMessage(c.Context(), userID, channelID, req.Content, nil, req.ReplyTo)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(message)
}

// GetMessage returns a specific message
func (h *ChannelHandler) GetMessage(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	messageID, err := uuid.Parse(c.Params("messageId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid message id",
		})
	}

	message, err := h.messageService.GetMessage(c.Context(), messageID, userID)
	if err != nil {
		switch err {
		case services.ErrMessageNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "message not found",
			})
		case services.ErrNoPermission:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "no permission to view this message",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to get message",
			})
		}
	}

	return c.JSON(message)
}

// EditMessage edits a message
func (h *ChannelHandler) EditMessage(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	messageID, err := uuid.Parse(c.Params("messageId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid message id",
		})
	}

	var req struct {
		Content string `json:"content"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	message, err := h.messageService.EditMessage(c.Context(), messageID, userID, req.Content)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(message)
}

// DeleteMessage deletes a message
func (h *ChannelHandler) DeleteMessage(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	messageID, err := uuid.Parse(c.Params("messageId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid message id",
		})
	}

	if err := h.messageService.DeleteMessage(c.Context(), messageID, userID); err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// AddReaction adds a reaction
func (h *ChannelHandler) AddReaction(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	messageID, err := uuid.Parse(c.Params("messageId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid message id",
		})
	}
	emoji := c.Params("emoji")

	if err := h.messageService.AddReaction(c.Context(), messageID, userID, emoji); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// RemoveReaction removes a reaction
func (h *ChannelHandler) RemoveReaction(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	messageID, err := uuid.Parse(c.Params("messageId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid message id",
		})
	}
	emoji := c.Params("emoji")

	if err := h.messageService.RemoveReaction(c.Context(), messageID, userID, emoji); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// GetPins returns pinned messages
func (h *ChannelHandler) GetPins(c *fiber.Ctx) error {
	return c.JSON([]interface{}{})
}

// PinMessage pins a message
func (h *ChannelHandler) PinMessage(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	messageID, err := uuid.Parse(c.Params("messageId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid message id",
		})
	}

	if err := h.messageService.PinMessage(c.Context(), messageID, userID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// UnpinMessage unpins a message
func (h *ChannelHandler) UnpinMessage(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	messageID, err := uuid.Parse(c.Params("messageId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid message id",
		})
	}

	if err := h.messageService.UnpinMessage(c.Context(), messageID, userID); err != nil {
		switch err {
		case services.ErrMessageNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "message not found",
			})
		default:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// TriggerTyping triggers typing indicator
func (h *ChannelHandler) TriggerTyping(c *fiber.Ctx) error {
	// TODO: Broadcast typing event via WebSocket
	return c.SendStatus(fiber.StatusNoContent)
}

// CreateInvite creates a channel invite
func (h *ChannelHandler) CreateInvite(c *fiber.Ctx) error {
	// TODO: Implement
	return c.JSON(fiber.Map{})
}
