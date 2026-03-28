package handlers

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
)

// DMServiceInterface defines the methods needed from DMService
type DMServiceInterface interface {
	AddUserToGroupDM(ctx context.Context, channelID, requesterID, userID uuid.UUID) (*models.Channel, error)
	RemoveUserFromGroupDM(ctx context.Context, channelID, requesterID, userID uuid.UUID) error
	LeaveDM(ctx context.Context, channelID, userID uuid.UUID) error
}

// DMHandler handles DM-specific API endpoints
type DMHandler struct {
	dmService      DMServiceInterface
	channelService ChannelServiceForUsersInterface
	userService    UserServiceInterface
	messageService MessageServiceInterface
}

// MessageServiceInterface defines message service methods needed by DM handler
type MessageServiceInterface interface {
	GetMessages(ctx context.Context, channelID uuid.UUID, requesterID uuid.UUID, before, after *uuid.UUID, limit int) ([]*models.Message, error)
	SendMessage(ctx context.Context, authorID, channelID uuid.UUID, content string, attachments []*models.Attachment, replyTo *uuid.UUID, stickerID *uuid.UUID) (*models.Message, error)
}

// NewDMHandler creates a new DM handler
func NewDMHandler(
	dmService DMServiceInterface,
	channelService ChannelServiceForUsersInterface,
	userService UserServiceInterface,
	messageService MessageServiceInterface,
) *DMHandler {
	return &DMHandler{
		dmService:      dmService,
		channelService: channelService,
		userService:    userService,
		messageService: messageService,
	}
}

// GetUserDMs returns the current user's DM channels
func (h *DMHandler) GetUserDMs(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	channels, err := h.channelService.GetUserDMs(c.Context(), userID)
	if err != nil {
		return InternalError(c, "failed to get DMs")
	}

	response := make([]DMChannelResponse, len(channels))
	for i, ch := range channels {
		recipients := h.resolveRecipients(c.Context(), ch.Recipients, userID)
		response[i] = DMChannelResponse{
			ID:            ch.ID,
			Type:          ch.Type,
			Name:          ch.Name,
			OwnerID:       ch.OwnerID,
			Recipients:    recipients,
			LastMessageID: ch.LastMessageID,
			CreatedAt:     ch.CreatedAt,
		}
	}

	return c.JSON(response)
}

// CreateDM creates or retrieves a 1:1 DM channel
func (h *DMHandler) CreateDM(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	var req models.CreateDMRequest
	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	if req.RecipientID == "" {
		return BadRequest(c, "recipient_id is required")
	}

	recipientID, err := uuid.Parse(req.RecipientID)
	if err != nil {
		return InvalidUUID(c, "recipient_id")
	}

	if recipientID == userID {
		return BadRequest(c, "cannot create DM with yourself")
	}

	channel, err := h.channelService.GetOrCreateDM(c.Context(), userID, recipientID)
	if err != nil {
		return InternalError(c, "failed to create DM channel")
	}

	recipientUser, err := h.userService.GetUser(c.Context(), recipientID)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(DMChannelResponse{
		ID:            channel.ID,
		Type:          channel.Type,
		Recipients:    []UserResponse{*toUserResponse(recipientUser)},
		LastMessageID: channel.LastMessageID,
		CreatedAt:     channel.CreatedAt,
	})
}

// CreateGroupDM creates a new group DM
func (h *DMHandler) CreateGroupDM(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	var req models.CreateGroupDMRequest
	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	if len(req.RecipientIDs) == 0 {
		return BadRequest(c, "at least one recipient is required")
	}
	if len(req.RecipientIDs) > 9 {
		return BadRequest(c, "group DM can have at most 10 members")
	}

	recipientIDs := make([]uuid.UUID, 0, len(req.RecipientIDs))
	for _, idStr := range req.RecipientIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return BadRequest(c, fmt.Sprintf("invalid recipient_id: %s", idStr))
		}
		if id == userID {
			continue
		}
		recipientIDs = append(recipientIDs, id)
	}

	if len(recipientIDs) == 0 {
		return BadRequest(c, "at least one other recipient is required")
	}

	name := ""
	if req.Name != nil {
		name = *req.Name
	}

	channel, err := h.channelService.CreateGroupDM(c.Context(), userID, name, recipientIDs)
	if err != nil {
		return InternalError(c, "failed to create group DM")
	}

	recipients := h.resolveRecipients(c.Context(), channel.Recipients, uuid.Nil)
	return c.Status(fiber.StatusCreated).JSON(DMChannelResponse{
		ID:            channel.ID,
		Type:          channel.Type,
		Name:          channel.Name,
		OwnerID:       channel.OwnerID,
		Recipients:    recipients,
		LastMessageID: channel.LastMessageID,
		CreatedAt:     channel.CreatedAt,
	})
}

// CreateDMWithUser creates a DM with a user by their user ID (convenience route)
func (h *DMHandler) CreateDMWithUser(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	targetIDStr := c.Params("userId")
	targetID, err := uuid.Parse(targetIDStr)
	if err != nil {
		return InvalidUUID(c, "userId")
	}

	if targetID == userID {
		return BadRequest(c, "cannot create DM with yourself")
	}

	channel, err := h.channelService.GetOrCreateDM(c.Context(), userID, targetID)
	if err != nil {
		return InternalError(c, "failed to create DM channel")
	}

	recipientUser, err := h.userService.GetUser(c.Context(), targetID)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(DMChannelResponse{
		ID:            channel.ID,
		Type:          channel.Type,
		Recipients:    []UserResponse{*toUserResponse(recipientUser)},
		LastMessageID: channel.LastMessageID,
		CreatedAt:     channel.CreatedAt,
	})
}

// GetDMMessages returns messages in a DM channel
func (h *DMHandler) GetDMMessages(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	channelIDStr := c.Params("channelId")
	channelID, err := uuid.Parse(channelIDStr)
	if err != nil {
		return InvalidUUID(c, "channelId")
	}

	// Verify user is a participant in this DM
	channels, err := h.channelService.GetUserDMs(c.Context(), userID)
	if err != nil {
		return InternalError(c, "failed to verify DM membership")
	}

	found := false
	for _, ch := range channels {
		if ch.ID == channelID {
			found = true
			break
		}
	}
	if !found {
		return Forbidden(c, "you are not a participant in this DM")
	}

	var before, after *uuid.UUID
	if beforeStr := c.Query("before"); beforeStr != "" {
		if id, err := uuid.Parse(beforeStr); err == nil {
			before = &id
		}
	}
	if afterStr := c.Query("after"); afterStr != "" {
		if id, err := uuid.Parse(afterStr); err == nil {
			after = &id
		}
	}
	limit := c.QueryInt("limit", 50)
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	messages, err := h.messageService.GetMessages(c.Context(), channelID, userID, before, after, limit)
	if err != nil {
		return InternalError(c, "failed to get messages")
	}

	return c.JSON(messages)
}

// SendDMMessage sends a message to a DM channel
func (h *DMHandler) SendDMMessage(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	channelIDStr := c.Params("channelId")
	channelID, err := uuid.Parse(channelIDStr)
	if err != nil {
		return InvalidUUID(c, "channelId")
	}

	// Verify user is a participant in this DM
	channels, err := h.channelService.GetUserDMs(c.Context(), userID)
	if err != nil {
		return InternalError(c, "failed to verify DM membership")
	}

	found := false
	for _, ch := range channels {
		if ch.ID == channelID {
			found = true
			break
		}
	}
	if !found {
		return Forbidden(c, "you are not a participant in this DM")
	}

	var req models.CreateMessageRequest
	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	if req.Content == "" {
		return BadRequest(c, "message content is required")
	}

	var replyTo *uuid.UUID
	if req.ReplyToID != nil {
		id, err := uuid.Parse(*req.ReplyToID)
		if err != nil {
			return InvalidUUID(c, "reply_to_id")
		}
		replyTo = &id
	}

	var stickerID *uuid.UUID
	if req.StickerID != nil {
		id, err := uuid.Parse(*req.StickerID)
		if err != nil {
			return InvalidUUID(c, "sticker_id")
		}
		stickerID = &id
	}

	msg, err := h.messageService.SendMessage(c.Context(), userID, channelID, req.Content, nil, replyTo, stickerID)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(msg)
}

// AddParticipant adds a user to a group DM
func (h *DMHandler) AddParticipant(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	channelIDStr := c.Params("channelId")
	channelID, err := uuid.Parse(channelIDStr)
	if err != nil {
		return InvalidUUID(c, "channelId")
	}

	var req struct {
		UserID string `json:"user_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	if req.UserID == "" {
		return BadRequest(c, "user_id is required")
	}

	targetID, err := uuid.Parse(req.UserID)
	if err != nil {
		return InvalidUUID(c, "user_id")
	}

	channel, err := h.dmService.AddUserToGroupDM(c.Context(), channelID, userID, targetID)
	if err != nil {
		return HandleServiceError(c, err)
	}

	recipients := h.resolveRecipients(c.Context(), channel.Recipients, uuid.Nil)
	return c.JSON(DMChannelResponse{
		ID:            channel.ID,
		Type:          channel.Type,
		Name:          channel.Name,
		OwnerID:       channel.OwnerID,
		Recipients:    recipients,
		LastMessageID: channel.LastMessageID,
		CreatedAt:     channel.CreatedAt,
	})
}

// RemoveParticipant removes a user from a group DM
func (h *DMHandler) RemoveParticipant(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	channelIDStr := c.Params("channelId")
	channelID, err := uuid.Parse(channelIDStr)
	if err != nil {
		return InvalidUUID(c, "channelId")
	}

	// Accept user_id from query param or request body
	targetIDStr := c.Query("user_id")
	if targetIDStr == "" {
		var req struct {
			UserID string `json:"user_id"`
		}
		if err := c.BodyParser(&req); err != nil {
			return ParseError(c, err)
		}
		targetIDStr = req.UserID
	}

	if targetIDStr == "" {
		return BadRequest(c, "user_id is required")
	}

	targetID, err := uuid.Parse(targetIDStr)
	if err != nil {
		return InvalidUUID(c, "user_id")
	}

	if err := h.dmService.RemoveUserFromGroupDM(c.Context(), channelID, userID, targetID); err != nil {
		return HandleServiceError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// LeaveDM removes the current user from a DM
func (h *DMHandler) LeaveDM(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	channelIDStr := c.Params("channelId")
	channelID, err := uuid.Parse(channelIDStr)
	if err != nil {
		return InvalidUUID(c, "channelId")
	}

	if err := h.dmService.LeaveDM(c.Context(), channelID, userID); err != nil {
		return HandleServiceError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// resolveRecipients converts recipient UUIDs to UserResponse objects
func (h *DMHandler) resolveRecipients(ctx context.Context, recipientIDs []uuid.UUID, excludeUserID uuid.UUID) []UserResponse {
	recipients := make([]UserResponse, 0, len(recipientIDs))
	for _, recipientID := range recipientIDs {
		if recipientID == excludeUserID {
			continue
		}
		user, err := h.userService.GetUser(ctx, recipientID)
		if err != nil {
			continue
		}
		recipients = append(recipients, *toUserResponse(user))
	}
	return recipients
}
