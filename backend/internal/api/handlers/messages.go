package handlers

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
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
	ID          string  `json:"id"`
	Filename    string  `json:"filename"`
	Size        int64   `json:"size"`
	URL         string  `json:"url"`
	ContentType string  `json:"content_type"`
	Width       *int    `json:"width,omitempty"`
	Height      *int    `json:"height,omitempty"`
	AltText     *string `json:"alt_text,omitempty"` // A11Y-004: Accessibility description
}

// ReactionResponse represents a reaction in API responses
type ReactionResponse struct {
	Emoji string `json:"emoji"`
	Count int    `json:"count"`
	Me    bool   `json:"me"`
}

// SendMessage creates a new message in a channel
// @Summary Send message
// @Description Creates a new message in the specified channel
// @Tags Messages
// @Accept json
// @Produce json
// @Param channelID path string true "Channel ID"
// @Param body body struct{Content string `json:"content"`; Nonce *string `json:"nonce,omitempty"`; TTS bool `json:"tts"`; ReplyToID *string `json:"message_reference,omitempty"`} true "Message data"
// @Success 201 {object} models.Message "Message created successfully"
// @Failure 400 {object} fiber.Map "Invalid request or channel ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Forbidden - not a member of channel"
// @Failure 404 {object} fiber.Map "Channel not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{channelID}/messages [post]
func (h *MessageHandlers) SendMessage(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	channelID, err := uuid.Parse(c.Params("channelID"))
	if err != nil {
		return InvalidUUID(c, "channel ID")
	}

	var req struct {
		Content   string  `json:"content"`
		Nonce     *string `json:"nonce,omitempty"`
		TTS       bool    `json:"tts"`
		ReplyToID *string `json:"message_reference,omitempty"`
		StickerID *string `json:"sticker_id,omitempty"`
	}
	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	var replyToID *uuid.UUID
	if req.ReplyToID != nil {
		id, err := uuid.Parse(*req.ReplyToID)
		if err == nil {
			replyToID = &id
		}
	}

	var stickerID *uuid.UUID
	if req.StickerID != nil {
		id, err := uuid.Parse(*req.StickerID)
		if err == nil {
			stickerID = &id
		}
	}

	message, err := h.messageService.SendMessage(c.Context(), userID, channelID, req.Content, nil, replyToID, stickerID)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(message)
}

// GetMessages returns messages in a channel with pagination
// @Summary Get messages
// @Description Returns a list of messages in a channel with optional pagination
// @Tags Messages
// @Produce json
// @Param channelID path string true "Channel ID"
// @Param limit query int false "Number of messages to return (max 100, default 50)"
// @Param before query string false "Get messages before this message ID"
// @Param after query string false "Get messages after this message ID"
// @Success 200 {array} models.Message "List of messages"
// @Failure 400 {object} fiber.Map "Invalid channel ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Forbidden - not a member of channel"
// @Failure 404 {object} fiber.Map "Channel not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{channelID}/messages [get]
func (h *MessageHandlers) GetMessages(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	channelID, err := uuid.Parse(c.Params("channelID"))
	if err != nil {
		return InvalidUUID(c, "channel ID")
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
		return HandleServiceError(c, err)
	}

	if messages == nil {
		messages = []*models.Message{}
	}

	return c.JSON(messages)
}

// GetMessage returns a specific message
// @Summary Get message
// @Description Returns a specific message by ID
// @Tags Messages
// @Produce json
// @Param channelID path string true "Channel ID"
// @Param messageID path string true "Message ID"
// @Success 200 {object} models.Message "Message object"
// @Failure 400 {object} fiber.Map "Invalid message ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Forbidden - not a member of channel"
// @Failure 404 {object} fiber.Map "Message not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{channelID}/messages/{messageID} [get]
func (h *MessageHandlers) GetMessage(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	messageID, err := uuid.Parse(c.Params("messageID"))
	if err != nil {
		return InvalidUUID(c, "message ID")
	}

	message, err := h.messageService.GetMessage(c.Context(), messageID, userID)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(message)
}

// EditMessage edits a message
// @Summary Edit message
// @Description Edits the content of an existing message (author only)
// @Tags Messages
// @Accept json
// @Produce json
// @Param channelID path string true "Channel ID"
// @Param messageID path string true "Message ID"
// @Param body body struct{Content string `json:"content"`} true "Updated message content"
// @Success 200 {object} models.Message "Updated message"
// @Failure 400 {object} fiber.Map "Invalid request or message ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Forbidden - not the message author"
// @Failure 404 {object} fiber.Map "Message not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{channelID}/messages/{messageID} [patch]
func (h *MessageHandlers) EditMessage(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	messageID, err := uuid.Parse(c.Params("messageID"))
	if err != nil {
		return InvalidUUID(c, "message ID")
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	message, err := h.messageService.EditMessage(c.Context(), messageID, userID, req.Content)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(message)
}

// DeleteMessage deletes a message
// @Summary Delete message
// @Description Deletes a message (author or channel moderator only)
// @Tags Messages
// @Param channelID path string true "Channel ID"
// @Param messageID path string true "Message ID"
// @Success 204 "Message deleted successfully"
// @Failure 400 {object} fiber.Map "Invalid message ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Forbidden - cannot delete this message"
// @Failure 404 {object} fiber.Map "Message not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{channelID}/messages/{messageID} [delete]
func (h *MessageHandlers) DeleteMessage(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	messageID, err := uuid.Parse(c.Params("messageID"))
	if err != nil {
		return InvalidUUID(c, "message ID")
	}

	err = h.messageService.DeleteMessage(c.Context(), messageID, userID)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// BulkDeleteMessages deletes multiple messages
// @Summary Bulk delete messages
// @Description Deletes multiple messages at once (requires manage messages permission)
// @Tags Messages
// @Accept json
// @Param channelID path string true "Channel ID"
// @Param body body struct{Messages []string `json:"messages"`} true "Array of message IDs to delete"
// @Success 204 "Messages deleted successfully"
// @Failure 400 {object} fiber.Map "Invalid request or channel ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Forbidden - insufficient permissions"
// @Failure 404 {object} fiber.Map "Channel not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{channelID}/messages/bulk-delete [post]
func (h *MessageHandlers) BulkDeleteMessages(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	channelID, err := uuid.Parse(c.Params("channelID"))
	if err != nil {
		return InvalidUUID(c, "channel ID")
	}

	var req struct {
		Messages []string `json:"messages"`
	}

	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	if len(req.Messages) == 0 {
		return BadRequest(c, "messages array is empty")
	}

	messageIDs := make([]uuid.UUID, 0, len(req.Messages))
	for _, msgStr := range req.Messages {
		msgID, err := uuid.Parse(msgStr)
		if err != nil {
			return BadRequest(c, "invalid message ID: "+msgStr)
		}
		messageIDs = append(messageIDs, msgID)
	}

	err = h.messageService.BulkDeleteMessages(c.Context(), channelID, messageIDs, userID)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// AddReaction adds a reaction to a message
// @Summary Add reaction
// @Description Adds a reaction emoji to a message
// @Tags Messages
// @Param channelID path string true "Channel ID"
// @Param messageID path string true "Message ID"
// @Param emoji path string true "Emoji to add (Unicode or custom emoji)"
// @Success 204 "Reaction added successfully"
// @Failure 400 {object} fiber.Map "Invalid message ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Forbidden - not a member of channel"
// @Failure 404 {object} fiber.Map "Message not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{channelID}/messages/{messageID}/reactions/{emoji} [put]
func (h *MessageHandlers) AddReaction(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	messageID, err := uuid.Parse(c.Params("messageID"))
	if err != nil {
		return InvalidUUID(c, "message ID")
	}
	emoji := c.Params("emoji")

	err = h.messageService.AddReaction(c.Context(), messageID, userID, emoji)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// RemoveReaction removes a reaction from a message
// @Summary Remove reaction
// @Description Removes a reaction emoji from a message
// @Tags Messages
// @Param channelID path string true "Channel ID"
// @Param messageID path string true "Message ID"
// @Param emoji path string true "Emoji to remove (Unicode or custom emoji)"
// @Success 204 "Reaction removed successfully"
// @Failure 400 {object} fiber.Map "Invalid message ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Forbidden - not a member of channel"
// @Failure 404 {object} fiber.Map "Message or reaction not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{channelID}/messages/{messageID}/reactions/{emoji} [delete]
func (h *MessageHandlers) RemoveReaction(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	messageID, err := uuid.Parse(c.Params("messageID"))
	if err != nil {
		return InvalidUUID(c, "message ID")
	}
	emoji := c.Params("emoji")

	err = h.messageService.RemoveReaction(c.Context(), messageID, userID, emoji, nil)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// RemoveAllReactions removes all reactions from a message
// @Summary Remove all reactions
// @Description Removes all reactions from a message (requires manage messages permission)
// @Tags Messages
// @Param channelID path string true "Channel ID"
// @Param messageID path string true "Message ID"
// @Success 204 "All reactions removed successfully"
// @Failure 400 {object} fiber.Map "Invalid message ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Forbidden - insufficient permissions"
// @Failure 404 {object} fiber.Map "Message not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{channelID}/messages/{messageID}/reactions [delete]
func (h *MessageHandlers) RemoveAllReactions(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	messageID, err := uuid.Parse(c.Params("messageID"))
	if err != nil {
		return InvalidUUID(c, "message ID")
	}

	err = h.messageService.RemoveAllReactions(c.Context(), messageID, userID)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// GetPinnedMessages returns all pinned messages in a channel
// @Summary Get pinned messages
// @Description Returns a list of all pinned messages in a channel
// @Tags Messages
// @Produce json
// @Param channelID path string true "Channel ID"
// @Success 200 {array} models.Message "List of pinned messages"
// @Failure 400 {object} fiber.Map "Invalid channel ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Forbidden - not a member of channel"
// @Failure 404 {object} fiber.Map "Channel not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{channelID}/pins [get]
func (h *MessageHandlers) GetPinnedMessages(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	channelID, err := uuid.Parse(c.Params("channelID"))
	if err != nil {
		return InvalidUUID(c, "channel ID")
	}

	messages, err := h.messageService.GetPinnedMessages(c.Context(), channelID, userID)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(messages)
}

// PinMessage pins a message
// @Summary Pin message
// @Description Pins a message in a channel (requires manage messages permission)
// @Tags Messages
// @Param channelID path string true "Channel ID"
// @Param messageID path string true "Message ID"
// @Success 204 "Message pinned successfully"
// @Failure 400 {object} fiber.Map "Invalid message ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Forbidden - insufficient permissions"
// @Failure 404 {object} fiber.Map "Message not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{channelID}/pins/{messageID} [put]
func (h *MessageHandlers) PinMessage(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	messageID, err := uuid.Parse(c.Params("messageID"))
	if err != nil {
		return InvalidUUID(c, "message ID")
	}

	err = h.messageService.PinMessage(c.Context(), messageID, userID)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// UnpinMessage unpins a message
// @Summary Unpin message
// @Description Unpins a message in a channel (requires manage messages permission)
// @Tags Messages
// @Param channelID path string true "Channel ID"
// @Param messageID path string true "Message ID"
// @Success 204 "Message unpinned successfully"
// @Failure 400 {object} fiber.Map "Invalid message ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Forbidden - insufficient permissions"
// @Failure 404 {object} fiber.Map "Message not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{channelID}/pins/{messageID} [delete]
func (h *MessageHandlers) UnpinMessage(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	messageID, err := uuid.Parse(c.Params("messageID"))
	if err != nil {
		return InvalidUUID(c, "message ID")
	}

	err = h.messageService.UnpinMessage(c.Context(), messageID, userID)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
