package handlers

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

type ChannelHandler struct {
	channelService   *services.ChannelService
	messageService   *services.MessageService
	typingService    *services.TypingService
	inviteService    *services.InviteService
	componentService *services.ComponentService
}

func NewChannelHandler(channelService *services.ChannelService, messageService *services.MessageService) *ChannelHandler {
	return &ChannelHandler{
		channelService: channelService,
		messageService: messageService,
	}
}

// NewChannelHandlerWithTyping creates a channel handler with typing service
func NewChannelHandlerWithTyping(channelService *services.ChannelService, messageService *services.MessageService, typingService *services.TypingService) *ChannelHandler {
	return &ChannelHandler{
		channelService: channelService,
		messageService: messageService,
		typingService:  typingService,
	}
}

// NewChannelHandlerFull creates a channel handler with all services
func NewChannelHandlerFull(
	channelService *services.ChannelService,
	messageService *services.MessageService,
	typingService *services.TypingService,
	inviteService *services.InviteService,
) *ChannelHandler {
	return &ChannelHandler{
		channelService: channelService,
		messageService: messageService,
		typingService:  typingService,
		inviteService:  inviteService,
	}
}

// SetComponentService sets the component service for handling message components
func (h *ChannelHandler) SetComponentService(componentService *services.ComponentService) {
	h.componentService = componentService
}

// Get returns a channel by ID
// @Summary Get channel by ID
// @Description Retrieves a channel by its unique identifier
// @Tags Channels
// @Produce json
// @Param id path string true "Channel ID"
// @Success 200 {object} models.Channel "Channel found"
// @Failure 400 {object} fiber.Map "Invalid channel ID"
// @Failure 404 {object} fiber.Map "Channel not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{id} [get]
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
// @Summary Update channel
// @Description Updates a channel's name, topic, position, NSFW flag, or slowmode settings
// @Tags Channels
// @Accept json
// @Produce json
// @Param id path string true "Channel ID"
// @Param body body models.UpdateChannelRequest true "Channel update data"
// @Success 200 {object} models.Channel "Channel updated successfully"
// @Failure 400 {object} fiber.Map "Invalid channel ID or request body"
// @Failure 403 {object} fiber.Map "Not a server member"
// @Failure 404 {object} fiber.Map "Channel not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{id} [patch]
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

	// Parse category_id if provided
	var parentID *uuid.UUID
	if req.CategoryID != nil {
		if *req.CategoryID == "" {
			// Explicitly set to no category (top-level)
			nilUUID := uuid.Nil
			parentID = &nilUUID
		} else if parsed, err := uuid.Parse(*req.CategoryID); err == nil {
			parentID = &parsed
		}
	}

	// Convert request to service update struct
	update := &models.ChannelUpdate{
		Name:     req.Name,
		Topic:    req.Topic,
		Position: req.Position,
		ParentID: parentID,
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
// @Summary Delete channel
// @Description Deletes a channel permanently
// @Tags Channels
// @Param id path string true "Channel ID"
// @Success 204 "Channel deleted successfully"
// @Failure 400 {object} fiber.Map "Invalid channel ID"
// @Failure 403 {object} fiber.Map "Not a server member or cannot delete DM"
// @Failure 404 {object} fiber.Map "Channel not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{id} [delete]
func (h *ChannelHandler) Delete(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	channelID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	if err := h.channelService.DeleteChannel(c.Context(), channelID, userID); err != nil {
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

// ReorderChannels bulk-updates channel positions and category assignments
func (h *ChannelHandler) ReorderChannels(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	var req models.ReorderChannelsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if len(req.Channels) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "channels array is required",
		})
	}

	if err := h.channelService.ReorderChannels(c.Context(), userID, req.Channels); err != nil {
		switch err {
		case services.ErrChannelNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "channel not found",
			})
		case services.ErrNotServerMember:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "not a server member",
			})
		case services.ErrNoPermission:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "insufficient permissions",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to reorder channels",
			})
		}
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// GetMessages returns messages with pagination
// @Summary Get channel messages
// @Description Retrieves messages from a channel with optional pagination using before/after cursors
// @Tags Channels
// @Produce json
// @Param id path string true "Channel ID"
// @Param before query string false "Message ID to get messages before"
// @Param after query string false "Message ID to get messages after"
// @Param limit query int false "Number of messages to return (default: 50)"
// @Success 200 {array} models.Message "List of messages"
// @Failure 400 {object} fiber.Map "Invalid channel ID"
// @Failure 403 {object} fiber.Map "Access denied"
// @Router /channels/{id}/messages [get]
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
		log.Printf("DEBUG GetMessages error for channel %s user %s: %v", channelID, userID, err)
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(messages)
}

// SendMessage sends a message
// @Summary Send message
// @Description Sends a new message to a channel
// @Tags Channels
// @Accept json
// @Produce json
// @Param id path string true "Channel ID"
// @Param body body struct{Content string `json:"content"`; ReplyTo *uuid.UUID `json:"reply_to"`} true "Message content"
// @Success 201 {object} models.Message "Message created"
// @Failure 400 {object} fiber.Map "Invalid channel ID or request body"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{id}/messages [post]
func (h *ChannelHandler) SendMessage(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	channelID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	var req struct {
		Content    string                          `json:"content"`
		ReplyTo    *uuid.UUID                      `json:"reply_to"`
		StickerID  *uuid.UUID                      `json:"sticker_id"`
		Components []models.CreateComponentRequest `json:"components,omitempty"`
		// Attachments handled separately via multipart
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	message, err := h.messageService.SendMessage(c.Context(), userID, channelID, req.Content, nil, req.ReplyTo, req.StickerID)
	if err != nil {
		log.Printf("DEBUG SendMessage error for channel %s user %s: %v", channelID, userID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// If components are provided and we have a component service, create them
	if len(req.Components) > 0 && h.componentService != nil {
		var components []*models.MessageComponent
		for _, compReq := range req.Components {
			comp := &models.MessageComponent{
				Type:        compReq.Type,
				Style:       compReq.Style,
				Label:       compReq.Label,
				CustomID:    compReq.CustomID,
				URL:         compReq.URL,
				Disabled:    compReq.Disabled,
				EmojiName:   compReq.Emoji,
				Options:     compReq.Options,
				MinValues:   compReq.MinValues,
				MaxValues:   compReq.MaxValues,
				Placeholder: compReq.Placeholder,
				Required:    compReq.Required,
				Value:       compReq.Value,
				MinLength:   compReq.MinLength,
				MaxLength:   compReq.MaxLength,
			}
			components = append(components, comp)
		}

		createdComponents, err := h.componentService.UpdateMessageComponents(c.Context(), message.ID, components)
		if err != nil {
			// Log error but don't fail the message creation
			log.Printf("Failed to create message components: %v", err)
		} else {
			// Convert pointer slice to value slice
			componentsValue := make([]models.MessageComponent, len(createdComponents))
			for i, comp := range createdComponents {
				componentsValue[i] = *comp
			}
			message.Components = componentsValue
		}
	}

	return c.Status(fiber.StatusCreated).JSON(message)
}

// GetMessage returns a specific message
// @Summary Get message
// @Description Retrieves a specific message by ID
// @Tags Channels
// @Produce json
// @Param id path string true "Channel ID"
// @Param messageId path string true "Message ID"
// @Success 200 {object} models.Message "Message found"
// @Failure 400 {object} fiber.Map "Invalid message ID"
// @Failure 403 {object} fiber.Map "Access denied"
// @Failure 404 {object} fiber.Map "Message or channel not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{id}/messages/{messageId} [get]
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
		case services.ErrChannelNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "channel not found",
			})
		case services.ErrNotServerMember, services.ErrNoPermission:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "access denied",
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
// @Summary Edit message
// @Description Edits the content of an existing message
// @Tags Channels
// @Accept json
// @Produce json
// @Param id path string true "Channel ID"
// @Param messageId path string true "Message ID"
// @Param body body struct{Content string `json:"content"`} true "Updated message content"
// @Success 200 {object} models.Message "Message edited successfully"
// @Failure 400 {object} fiber.Map "Invalid message ID or request body"
// @Router /channels/{id}/messages/{messageId} [patch]
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
// @Summary Delete message
// @Description Deletes a message permanently
// @Tags Channels
// @Param id path string true "Channel ID"
// @Param messageId path string true "Message ID"
// @Success 204 "Message deleted successfully"
// @Failure 400 {object} fiber.Map "Invalid message ID"
// @Failure 403 {object} fiber.Map "Access denied"
// @Router /channels/{id}/messages/{messageId} [delete]
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
// @Summary Add reaction
// @Description Adds a reaction emoji to a message
// @Tags Channels
// @Param id path string true "Channel ID"
// @Param messageId path string true "Message ID"
// @Param emoji path string true "Emoji to add"
// @Success 204 "Reaction added successfully"
// @Failure 400 {object} fiber.Map "Invalid message ID"
// @Router /channels/{id}/messages/{messageId}/reactions/{emoji} [put]
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
// @Summary Remove reaction
// @Description Removes a reaction emoji from a message
// @Tags Channels
// @Param id path string true "Channel ID"
// @Param messageId path string true "Message ID"
// @Param emoji path string true "Emoji to remove"
// @Success 204 "Reaction removed successfully"
// @Failure 400 {object} fiber.Map "Invalid message ID"
// @Router /channels/{id}/messages/{messageId}/reactions/{emoji} [delete]
func (h *ChannelHandler) RemoveReaction(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	messageID, err := uuid.Parse(c.Params("messageId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid message id",
		})
	}
	emoji := c.Params("emoji")

	if err := h.messageService.RemoveReaction(c.Context(), messageID, userID, emoji, nil); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// GetReactions returns all reactions for a message
// @Summary Get all reactions
// @Description Retrieves all reactions for a specific message
// @Tags Channels
// @Produce json
// @Param id path string true "Channel ID"
// @Param messageId path string true "Message ID"
// @Success 200 {array} models.Reaction "List of reactions"
// @Failure 400 {object} fiber.Map "Invalid message ID"
// @Failure 403 {object} fiber.Map "Access denied"
// @Failure 404 {object} fiber.Map "Message or channel not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{id}/messages/{messageId}/reactions [get]
func (h *ChannelHandler) GetReactions(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	messageID, err := uuid.Parse(c.Params("messageId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid message id",
		})
	}

	reactions, err := h.messageService.GetReactions(c.Context(), messageID, userID)
	if err != nil {
		switch err {
		case services.ErrMessageNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "message not found",
			})
		case services.ErrChannelNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "channel not found",
			})
		case services.ErrNotServerMember, services.ErrNoPermission:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "access denied",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to get reactions",
			})
		}
	}

	return c.JSON(reactions)
}

// GetReactionUsers returns users who reacted with a specific emoji
// @Summary Get reaction users
// @Description Retrieves users who reacted with a specific emoji
// @Tags Channels
// @Produce json
// @Param id path string true "Channel ID"
// @Param messageId path string true "Message ID"
// @Param emoji path string true "Emoji"
// @Param limit query int false "Maximum number of users to return (default: 25)"
// @Success 200 {array} models.User "List of users who reacted"
// @Failure 400 {object} fiber.Map "Invalid message ID"
// @Failure 403 {object} fiber.Map "Access denied"
// @Failure 404 {object} fiber.Map "Message or channel not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{id}/messages/{messageId}/reactions/{emoji} [get]
func (h *ChannelHandler) GetReactionUsers(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	messageID, err := uuid.Parse(c.Params("messageId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid message id",
		})
	}
	emoji := c.Params("emoji")

	limit := c.QueryInt("limit", 25)

	reactionUsers, err := h.messageService.GetReactionUsers(c.Context(), messageID, emoji, userID, limit)
	if err != nil {
		switch err {
		case services.ErrMessageNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "message not found",
			})
		case services.ErrChannelNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "channel not found",
			})
		case services.ErrNotServerMember, services.ErrNoPermission:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "access denied",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to get reaction users",
			})
		}
	}

	return c.JSON(reactionUsers)
}

// RemoveAllReactions removes all reactions from a message
// @Summary Remove all reactions
// @Description Removes all reactions from a message (requires manage messages permission)
// @Tags Channels
// @Param id path string true "Channel ID"
// @Param messageId path string true "Message ID"
// @Success 204 "All reactions removed successfully"
// @Failure 400 {object} fiber.Map "Invalid message ID"
// @Failure 403 {object} fiber.Map "Forbidden - insufficient permissions"
// @Failure 404 {object} fiber.Map "Message not found"
// @Router /channels/{id}/messages/{messageId}/reactions [delete]
func (h *ChannelHandler) RemoveAllReactions(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	messageID, err := uuid.Parse(c.Params("messageId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid message ID",
		})
	}

	if err := h.messageService.RemoveAllReactions(c.Context(), messageID, userID); err != nil {
		switch err {
		case services.ErrMessageNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "message not found",
			})
		case services.ErrNoPermission, services.ErrMissingManageMessages:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "missing permissions",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to remove reactions",
			})
		}
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// GetPins returns pinned messages
// @Summary Get pinned messages
// @Description Retrieves all pinned messages in a channel
// @Tags Channels
// @Produce json
// @Param id path string true "Channel ID"
// @Success 200 {array} models.Message "List of pinned messages"
// @Router /channels/{id}/pins [get]
func (h *ChannelHandler) GetPins(c *fiber.Ctx) error {
	return c.JSON([]interface{}{})
}

// PinMessage pins a message
// @Summary Pin message
// @Description Pins a message to the channel
// @Tags Channels
// @Param id path string true "Channel ID"
// @Param messageId path string true "Message ID"
// @Success 204 "Message pinned successfully"
// @Failure 400 {object} fiber.Map "Invalid message ID"
// @Router /channels/{id}/pins/{messageId} [put]
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
// @Summary Unpin message
// @Description Unpins a message from the channel
// @Tags Channels
// @Param id path string true "Channel ID"
// @Param messageId path string true "Message ID"
// @Success 204 "Message unpinned successfully"
// @Failure 400 {object} fiber.Map "Invalid message ID"
// @Failure 403 {object} fiber.Map "Access denied"
// @Failure 404 {object} fiber.Map "Message or channel not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{id}/pins/{messageId} [delete]
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
		case services.ErrChannelNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "channel not found",
			})
		case services.ErrNotServerMember, services.ErrNoPermission:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "access denied",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to unpin message",
			})
		}
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// TriggerTyping triggers typing indicator
// @Summary Trigger typing indicator
// @Description Triggers a typing indicator for the current user in a channel
// @Tags Channels
// @Param id path string true "Channel ID"
// @Success 204 "Typing indicator triggered"
// @Failure 400 {object} fiber.Map "Invalid channel ID"
// @Failure 404 {object} fiber.Map "Channel not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{id}/typing [post]
func (h *ChannelHandler) TriggerTyping(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	channelID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	// Verify user has access to the channel
	_, err = h.channelService.GetChannel(c.Context(), channelID)
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

	// Start typing (will broadcast via event bus)
	if h.typingService != nil {
		if err := h.typingService.StartTyping(c.Context(), channelID, userID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to trigger typing",
			})
		}
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// GetTypingUsers returns users currently typing in a channel
// @Summary Get typing users
// @Description Retrieves a list of users currently typing in the channel
// @Tags Channels
// @Produce json
// @Param id path string true "Channel ID"
// @Success 200 {array} models.TypingIndicator "List of typing indicators"
// @Failure 400 {object} fiber.Map "Invalid channel ID"
// @Failure 404 {object} fiber.Map "Channel not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{id}/typing [get]
func (h *ChannelHandler) GetTypingUsers(c *fiber.Ctx) error {
	channelID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	// Verify user has access to the channel
	_, err = h.channelService.GetChannel(c.Context(), channelID)
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

	// Get typing users
	if h.typingService == nil {
		return c.JSON([]interface{}{})
	}

	indicators, err := h.typingService.GetTypingUsers(c.Context(), channelID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get typing users",
		})
	}

	// Return typing indicators with timestamps
	return c.JSON(indicators)
}

// CreateInvite creates a channel invite
// @Summary Create invite
// @Description Creates an invite link for a channel
// @Tags Channels
// @Accept json
// @Produce json
// @Param id path string true "Channel ID"
// @Param body body struct{MaxAge int `json:"max_age"`; MaxUses int `json:"max_uses"`; Temporary bool `json:"temporary"`} false "Invite settings"
// @Success 201 {object} fiber.Map "Invite created"
// @Failure 400 {object} fiber.Map "Invalid channel ID or cannot create invite for DM"
// @Failure 403 {object} fiber.Map "Not a server member"
// @Failure 404 {object} fiber.Map "Channel not found"
// @Failure 501 {object} fiber.Map "Invite service not configured"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{id}/invites [post]
func (h *ChannelHandler) CreateInvite(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	channelID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	var req struct {
		MaxAge    int  `json:"max_age"`  // Seconds, 0 = never expires
		MaxUses   int  `json:"max_uses"` // 0 = unlimited
		Temporary bool `json:"temporary"`
	}
	_ = c.BodyParser(&req) // Optional body

	// Get channel to find server ID
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

	// Can only create invites for server channels
	if channel.ServerID == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "cannot create invite for DM channel",
		})
	}

	// Check if invite service is available
	if h.inviteService == nil {
		return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
			"error": "invite service not configured",
		})
	}

	var maxAge time.Duration
	if req.MaxAge > 0 {
		maxAge = time.Duration(req.MaxAge) * time.Second
	}

	invite, err := h.inviteService.CreateInvite(c.Context(), &services.CreateInviteRequest{
		ServerID:  *channel.ServerID,
		ChannelID: channelID,
		CreatorID: userID,
		MaxAge:    maxAge,
		MaxUses:   req.MaxUses,
		Temporary: req.Temporary,
	})
	if err != nil {
		switch err {
		case services.ErrNotServerMember:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "not a server member",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to create invite",
			})
		}
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"code":        invite.Code,
		"guild_id":    invite.ServerID.String(),
		"channel_id":  invite.ChannelID.String(),
		"inviter_id":  invite.CreatorID.String(),
		"max_uses":    invite.MaxUses,
		"uses":        invite.Uses,
		"expires_at":  invite.ExpiresAt,
		"temporary":   invite.Temporary,
		"is_vanity":   invite.IsVanity,
		"vanity_code": invite.VanityCode,
		"created_at":  invite.CreatedAt,
	})
}

// GetPermissionOverrides returns all permission overrides for a channel
// @Summary Get permission overrides
// @Description Retrieves all permission overrides for a channel
// @Tags Channels
// @Produce json
// @Param id path string true "Channel ID"
// @Success 200 {array} models.PermissionOverride "List of permission overrides"
// @Failure 400 {object} fiber.Map "Invalid channel ID"
// @Failure 403 {object} fiber.Map "Access denied"
// @Failure 404 {object} fiber.Map "Channel not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{id}/permission-overwrites [get]
func (h *ChannelHandler) GetPermissionOverrides(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	channelID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	overrides, err := h.channelService.GetPermissionOverrides(c.Context(), channelID, userID)
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
		case services.ErrMissingManageChannels:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "missing permissions",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to get permission overrides",
			})
		}
	}

	return c.JSON(overrides)
}

// SetPermissionOverride creates or updates a permission override for a channel
// @Summary Set permission override
// @Description Creates or updates a permission override for a user or role in a channel
// @Tags Channels
// @Accept json
// @Produce json
// @Param id path string true "Channel ID"
// @Param body body struct{TargetType string `json:"target_type"`; TargetID string `json:"target_id"`; Allow int64 `json:"allow"`; Deny int64 `json:"deny"`} true "Permission override data"
// @Success 200 {object} models.PermissionOverride "Permission override updated"
// @Failure 400 {object} fiber.Map "Invalid channel ID or request body"
// @Failure 403 {object} fiber.Map "Access denied"
// @Failure 404 {object} fiber.Map "Channel not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{id}/permission-overwrites [put]
func (h *ChannelHandler) SetPermissionOverride(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	channelID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	var req struct {
		TargetType string `json:"target_type"` // "role" or "user"
		TargetID   string `json:"target_id"`
		Allow      int64  `json:"allow"`
		Deny       int64  `json:"deny"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.TargetType != "role" && req.TargetType != "user" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "target_type must be 'role' or 'user'",
		})
	}

	targetID, err := uuid.Parse(req.TargetID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid target_id",
		})
	}

	override, err := h.channelService.SetPermissionOverride(c.Context(), channelID, targetID, req.TargetType, req.Allow, req.Deny, userID)
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
		case services.ErrMissingManageChannels:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "missing permissions",
			})
		case services.ErrCannotDeleteDM:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "cannot set permissions on DM channel",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to set permission override",
			})
		}
	}

	return c.JSON(override)
}

// DeletePermissionOverride removes a permission override from a channel
// @Summary Delete permission override
// @Description Removes a permission override from a channel
// @Tags Channels
// @Param id path string true "Channel ID"
// @Param targetType path string true "Target type (role or user)"
// @Param targetId path string true "Target ID"
// @Success 204 "Permission override deleted successfully"
// @Failure 400 {object} fiber.Map "Invalid channel ID or target"
// @Failure 403 {object} fiber.Map "Access denied"
// @Failure 404 {object} fiber.Map "Channel not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{id}/permission-overwrites/{targetType}/{targetId} [delete]
func (h *ChannelHandler) DeletePermissionOverride(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	channelID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	targetType := c.Params("targetType")
	if targetType != "role" && targetType != "user" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "target_type must be 'role' or 'user'",
		})
	}

	targetID, err := uuid.Parse(c.Params("targetId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid target_id",
		})
	}

	if err := h.channelService.DeletePermissionOverride(c.Context(), channelID, targetID, targetType, userID); err != nil {
		switch err {
		case services.ErrChannelNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "channel not found",
			})
		case services.ErrNotServerMember:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "not a server member",
			})
		case services.ErrMissingManageChannels:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "missing permissions",
			})
		case services.ErrCannotDeleteDM:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "cannot delete permissions on DM channel",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to delete permission override",
			})
		}
	}

	return c.SendStatus(fiber.StatusNoContent)
}
