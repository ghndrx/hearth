package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// ComponentHandler handles component-related HTTP requests
type ComponentHandler struct {
	componentService  *services.ComponentService
	messageService    *services.MessageService
	channelService    *services.ChannelService
	permissionService *services.PermissionService
}

// NewComponentHandler creates a new component handler
func NewComponentHandler(
	componentService *services.ComponentService,
	messageService *services.MessageService,
	channelService *services.ChannelService,
	permissionService *services.PermissionService,
) *ComponentHandler {
	return &ComponentHandler{
		componentService:  componentService,
		messageService:    messageService,
		channelService:    channelService,
		permissionService: permissionService,
	}
}

// HandleComponentInteractionV2 handles a user's interaction with a component (full context in body)
// @Summary Handle component interaction
// @Description Records a user's interaction with a message component with full context
// @Tags Components
// @Accept json
// @Produce json
//
//	@Param body body struct{
//	   MessageID string `json:"message_id"`
//	   ChannelID string `json:"channel_id"`
//	   CustomID string `json:"custom_id"`
//	   Values []string `json:"values,omitempty"`
//	} true "Interaction data"
//
// @Success 200 {object} models.ComponentInteraction "Interaction recorded"
// @Failure 400 {object} fiber.Map "Invalid request"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Component is disabled"
// @Failure 404 {object} fiber.Map "Component not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /api/v1/interactions/components [post]
func (h *ComponentHandler) HandleComponentInteractionV2(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	var req struct {
		MessageID string   `json:"message_id"`
		ChannelID string   `json:"channel_id"`
		CustomID  string   `json:"custom_id"`
		Values    []string `json:"values,omitempty"`
	}
	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	if req.MessageID == "" || req.ChannelID == "" || req.CustomID == "" {
		return BadRequest(c, "message_id, channel_id, and custom_id are required")
	}

	msgID, err := uuid.Parse(req.MessageID)
	if err != nil {
		return InvalidUUID(c, "message ID")
	}

	channelID, err := uuid.Parse(req.ChannelID)
	if err != nil {
		return InvalidUUID(c, "channel ID")
	}

	// Get component by custom_id and message_id to find the component ID
	components, err := h.componentService.GetMessageComponents(c.Context(), msgID)
	if err != nil {
		return HandleServiceError(c, err)
	}

	var componentID uuid.UUID
	for _, comp := range components {
		if comp.CustomID == req.CustomID {
			componentID = comp.ID
			break
		}
	}

	if componentID == uuid.Nil {
		return NotFound(c, "Component not found")
	}

	interaction, err := h.componentService.HandleInteraction(
		c.Context(), userID, channelID, msgID, componentID, req.CustomID, req.Values,
	)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(interaction)
}

// GetMessageComponents returns all components for a message
// @Summary Get message components
// @Description Returns all interactive components attached to a message
// @Tags Components
// @Produce json
// @Param id path string true "Channel ID"
// @Param messageId path string true "Message ID"
// @Success 200 {array} models.MessageComponent "List of components"
// @Failure 400 {object} fiber.Map "Invalid message ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 404 {object} fiber.Map "Message not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /api/v1/channels/{id}/messages/{messageId}/components [get]
func (h *ComponentHandler) GetMessageComponents(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	messageID, err := uuid.Parse(c.Params("messageId"))
	if err != nil {
		return InvalidUUID(c, "message ID")
	}

	// Verify message exists and user has access
	message, err := h.messageService.GetMessage(c.Context(), messageID, userID)
	if err != nil {
		return HandleServiceError(c, err)
	}
	if message == nil {
		return NotFound(c, "Message not found")
	}

	components, err := h.componentService.GetMessageComponents(c.Context(), messageID)
	if err != nil {
		return HandleServiceError(c, err)
	}

	if components == nil {
		components = []*models.MessageComponent{}
	}

	return c.JSON(components)
}

// UpdateMessageComponents replaces all components on a message
// @Summary Update message components
// @Description Replaces all components on a message with new components
// @Tags Components
// @Accept json
// @Produce json
// @Param id path string true "Channel ID"
// @Param messageId path string true "Message ID"
// @Param body body models.UpdateComponentsRequest true "New components"
// @Success 200 {array} models.MessageComponent "Updated components"
// @Failure 400 {object} fiber.Map "Invalid request"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Not message author"
// @Failure 404 {object} fiber.Map "Message not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /api/v1/channels/{id}/messages/{messageId}/components [patch]
func (h *ComponentHandler) UpdateMessageComponents(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	messageID, err := uuid.Parse(c.Params("messageId"))
	if err != nil {
		return InvalidUUID(c, "message ID")
	}

	var req models.UpdateComponentsRequest
	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	// Verify message exists and user has access
	message, err := h.messageService.GetMessage(c.Context(), messageID, userID)
	if err != nil {
		return HandleServiceError(c, err)
	}
	if message == nil {
		return NotFound(c, "Message not found")
	}

	// Check if user is the message author (only author can modify components)
	if message.AuthorID != userID {
		// For bots/webhooks, allow modification - check if author is a webhook
		// For now, just check if user has manage messages permission in the channel
		channel, err := h.channelService.GetChannel(c.Context(), message.ChannelID)
		if err != nil {
			return HandleServiceError(c, err)
		}
		if channel == nil {
			return NotFound(c, "Channel not found")
		}
		// Check if user has manage messages permission
		permissions, err := h.permissionService.GetChannelPermissions(c.Context(), channel, userID)
		if err != nil {
			return HandleServiceError(c, err)
		}
		if !models.HasPermission(permissions, models.PermManageMessages) {
			return Forbidden(c, "You don't have permission to modify message components")
		}
	}

	// Convert request to components
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

	updated, err := h.componentService.UpdateMessageComponents(c.Context(), messageID, components)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(updated)
}

// RemoveAllComponents removes all components from a message
// @Summary Remove all message components
// @Description Removes all interactive components from a message
// @Tags Components
// @Param id path string true "Channel ID"
// @Param messageId path string true "Message ID"
// @Success 204 "Components removed successfully"
// @Failure 400 {object} fiber.Map "Invalid message ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Not message author"
// @Failure 404 {object} fiber.Map "Message not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /api/v1/channels/{id}/messages/{messageId}/components [delete]
func (h *ComponentHandler) RemoveAllComponents(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	messageID, err := uuid.Parse(c.Params("messageId"))
	if err != nil {
		return InvalidUUID(c, "message ID")
	}

	// Verify message exists and user has access
	message, err := h.messageService.GetMessage(c.Context(), messageID, userID)
	if err != nil {
		return HandleServiceError(c, err)
	}
	if message == nil {
		return NotFound(c, "Message not found")
	}

	// Check if user is the message author
	if message.AuthorID != userID {
		return Forbidden(c, "Only the message author can remove components")
	}

	if err := h.componentService.RemoveAllComponents(c.Context(), messageID); err != nil {
		return HandleServiceError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
