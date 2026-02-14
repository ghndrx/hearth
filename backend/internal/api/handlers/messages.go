package handlers

import (
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/services"
)

// MessageHandlers handles message-related HTTP requests
type MessageHandlers struct {
	messageService *services.MessageService
	channelService *services.ChannelService
}

// NewMessageHandlers creates new message handlers
func NewMessageHandlers(messageService *services.MessageService, channelService *services.ChannelService) *MessageHandlers {
	return &MessageHandlers{
		messageService: messageService,
		channelService: channelService,
	}
}

// MessageResponse represents a message in API responses
type MessageResponse struct {
	ID              string               `json:"id"`
	ChannelID       string               `json:"channel_id"`
	ServerID        *string              `json:"guild_id,omitempty"`
	AuthorID        string               `json:"author_id"`
	Content         string               `json:"content"`
	Type            int                  `json:"type"`
	Timestamp       time.Time            `json:"timestamp"`
	EditedTimestamp *time.Time           `json:"edited_timestamp,omitempty"`
	Pinned          bool                 `json:"pinned"`
	TTS             bool                 `json:"tts"`
	ReplyToID       *string              `json:"referenced_message_id,omitempty"`
	Attachments     []AttachmentResponse `json:"attachments,omitempty"`
	Reactions       []ReactionResponse   `json:"reactions,omitempty"`
}

// AttachmentResponse represents an attachment in API responses
type AttachmentResponse struct {
	ID          string `json:"id"`
	Filename    string `json:"filename"`
	Size        int64  `json:"size"`
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
	Width       *int   `json:"width,omitempty"`
	Height      *int   `json:"height,omitempty"`
}

// ReactionResponse represents a reaction in API responses
type ReactionResponse struct {
	Emoji string `json:"emoji"`
	Count int    `json:"count"`
	Me    bool   `json:"me"`
}

// SendMessage creates a new message in a channel
func (h *MessageHandlers) SendMessage(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	channelID, err := uuid.Parse(c.Params("channelID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid channel ID",
		})
	}

	var req struct {
		Content   string  `json:"content"`
		Nonce     *string `json:"nonce,omitempty"`
		TTS       bool    `json:"tts"`
		ReplyToID *string `json:"message_reference,omitempty"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	var replyToID *uuid.UUID
	if req.ReplyToID != nil {
		id, err := uuid.Parse(*req.ReplyToID)
		if err == nil {
			replyToID = &id
		}
	}

	message, err := h.messageService.SendMessage(c.Context(), userID, channelID, req.Content, nil, replyToID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(message)
}

// GetMessages returns messages in a channel with pagination
func (h *MessageHandlers) GetMessages(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	channelID, err := uuid.Parse(c.Params("channelID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid channel ID",
		})
	}

	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
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

	messages, err := h.messageService.GetMessages(c.Context(), channelID, userID, before, after, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(messages)
}

// GetMessage returns a specific message
func (h *MessageHandlers) GetMessage(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	messageID, err := uuid.Parse(c.Params("messageID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid message ID",
		})
	}

	message, err := h.messageService.GetMessage(c.Context(), messageID, userID)
	if err != nil {
		if errors.Is(err, services.ErrMessageNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Message not found",
			})
		}
		if errors.Is(err, services.ErrNotServerMember) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Not a member of this server",
			})
		}
		if errors.Is(err, services.ErrNoPermission) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "No permission to view this message",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(message)
}

// EditMessage edits a message
func (h *MessageHandlers) EditMessage(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	messageID, err := uuid.Parse(c.Params("messageID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid message ID",
		})
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	message, err := h.messageService.EditMessage(c.Context(), messageID, userID, req.Content)
	if err != nil {
		if errors.Is(err, services.ErrMessageNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Message not found",
			})
		}
		if errors.Is(err, services.ErrNotMessageAuthor) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "You can only edit your own messages",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(message)
}

// DeleteMessage deletes a message
func (h *MessageHandlers) DeleteMessage(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	messageID, err := uuid.Parse(c.Params("messageID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid message ID",
		})
	}

	err = h.messageService.DeleteMessage(c.Context(), messageID, userID)
	if err != nil {
		if errors.Is(err, services.ErrMessageNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Message not found",
			})
		}
		if errors.Is(err, services.ErrNotMessageAuthor) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "You can only delete your own messages",
			})
		}
		if errors.Is(err, services.ErrNoPermission) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "No permission to delete this message",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// BulkDeleteMessages deletes multiple messages
// Note: Bulk delete not implemented in service yet
func (h *MessageHandlers) BulkDeleteMessages(c *fiber.Ctx) error {
	_, err := uuid.Parse(c.Params("channelID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid channel ID",
		})
	}

	// TODO: Implement BulkDeleteMessages in service
	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
		"error": "Not implemented",
	})
}

// AddReaction adds a reaction to a message
func (h *MessageHandlers) AddReaction(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	messageID, err := uuid.Parse(c.Params("messageID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid message ID",
		})
	}
	emoji := c.Params("emoji")

	err = h.messageService.AddReaction(c.Context(), messageID, userID, emoji)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// RemoveReaction removes a reaction from a message
func (h *MessageHandlers) RemoveReaction(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	messageID, err := uuid.Parse(c.Params("messageID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid message ID",
		})
	}
	emoji := c.Params("emoji")

	err = h.messageService.RemoveReaction(c.Context(), messageID, userID, emoji)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// GetPinnedMessages returns all pinned messages in a channel
func (h *MessageHandlers) GetPinnedMessages(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	channelID, err := uuid.Parse(c.Params("channelID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid channel ID",
		})
	}

	messages, err := h.messageService.GetPinnedMessages(c.Context(), channelID, userID)
	if err != nil {
		if errors.Is(err, services.ErrChannelNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Channel not found",
			})
		}
		if errors.Is(err, services.ErrNotServerMember) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Not a member of this server",
			})
		}
		if errors.Is(err, services.ErrNoPermission) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "No permission to access this channel",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(messages)
}

// PinMessage pins a message
func (h *MessageHandlers) PinMessage(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	messageID, err := uuid.Parse(c.Params("messageID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid message ID",
		})
	}

	err = h.messageService.PinMessage(c.Context(), messageID, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// UnpinMessage unpins a message
func (h *MessageHandlers) UnpinMessage(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	messageID, err := uuid.Parse(c.Params("messageID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid message ID",
		})
	}

	err = h.messageService.UnpinMessage(c.Context(), messageID, userID)
	if err != nil {
		if errors.Is(err, services.ErrMessageNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Message not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
