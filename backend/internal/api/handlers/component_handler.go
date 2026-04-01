package handlers

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
)

// ComponentServiceInterface defines the methods needed from ComponentService
type ComponentServiceInterface interface {
	GetMessageComponents(ctx context.Context, messageID uuid.UUID) ([]*models.MessageComponent, error)
	HandleInteraction(ctx context.Context, userID, channelID, messageID, componentID uuid.UUID, customID string, values []string) (*models.ComponentInteraction, error)
	UpdateMessageComponents(ctx context.Context, messageID uuid.UUID, components []*models.MessageComponent) ([]*models.MessageComponent, error)
	RemoveAllComponents(ctx context.Context, messageID uuid.UUID) error
	// Modal methods
	CreateModal(ctx context.Context, modal *models.ModalComponent) (*models.ModalComponent, error)
	GetModalByCustomID(ctx context.Context, customID string) (*models.ModalComponent, error)
	DeleteModal(ctx context.Context, id uuid.UUID) error
	HandleModalSubmit(ctx context.Context, userID, channelID, msgID, modalID, componentID uuid.UUID, customID string, values map[string]string) (*models.ModalInteraction, error)
}

// MessageServiceGetMessageInterface defines the GetMessage method needed from MessageService
type MessageServiceGetMessageInterface interface {
	GetMessage(ctx context.Context, messageID uuid.UUID, requesterID uuid.UUID) (*models.Message, error)
}

// ChannelServiceGetChannelInterface defines the GetChannel method needed from ChannelService
type ChannelServiceGetChannelInterface interface {
	GetChannel(ctx context.Context, channelID uuid.UUID) (*models.Channel, error)
}

// PermissionServiceGetChannelPermissionsInterface defines the GetChannelPermissions method needed from PermissionService
type PermissionServiceGetChannelPermissionsInterface interface {
	GetChannelPermissions(ctx context.Context, channel *models.Channel, userID uuid.UUID) (int64, error)
}

// ComponentHandler handles component-related HTTP requests
type ComponentHandler struct {
	componentService  ComponentServiceInterface
	messageService    MessageServiceGetMessageInterface
	channelService    ChannelServiceGetChannelInterface
	permissionService PermissionServiceGetChannelPermissionsInterface
}

// NewComponentHandler creates a new component handler
func NewComponentHandler(
	componentService ComponentServiceInterface,
	messageService MessageServiceGetMessageInterface,
	channelService ChannelServiceGetChannelInterface,
	permissionService PermissionServiceGetChannelPermissionsInterface,
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

// HandleModalSubmit handles a user's modal submission
// @Summary Handle modal submission
// @Description Records a user's modal submission
// @Tags Components
// @Accept json
// @Produce json
//
//	@Param body body models.ModalSubmitRequest true "Modal submission data"
//
// @Success 200 {object} models.ModalInteraction "Modal interaction recorded"
// @Failure 400 {object} fiber.Map "Invalid request"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 404 {object} fiber.Map "Modal not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /api/v1/interactions/modals/submit [post]
func (h *ComponentHandler) HandleModalSubmit(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	var req models.ModalSubmitRequest
	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	if req.MessageID == "" || req.ChannelID == "" || req.ModalID == "" {
		return BadRequest(c, "message_id, channel_id, and modal_id are required")
	}

	msgID, err := uuid.Parse(req.MessageID)
	if err != nil {
		return InvalidUUID(c, "message ID")
	}

	channelID, err := uuid.Parse(req.ChannelID)
	if err != nil {
		return InvalidUUID(c, "channel ID")
	}

	modalID, err := uuid.Parse(req.ModalID)
	if err != nil {
		return InvalidUUID(c, "modal ID")
	}

	componentID, err := uuid.Parse(req.ComponentID)
	if err != nil {
		return InvalidUUID(c, "component ID")
	}

	interaction, err := h.componentService.HandleModalSubmit(
		c.Context(), userID, channelID, msgID, modalID, componentID, req.CustomID, req.Values,
	)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(interaction)
}

// CreateModal creates a new modal
// @Summary Create modal
// @Description Creates a new modal component
// @Tags Components
// @Accept json
// @Produce json
//
//	@Param body body models.CreateModalRequest true "Modal data"
//
// @Success 201 {object} models.ModalComponent "Modal created"
// @Failure 400 {object} fiber.Map "Invalid request"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /api/v1/modals [post]
func (h *ComponentHandler) CreateModal(c *fiber.Ctx) error {
	var req models.CreateModalRequest
	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	// Convert request to modal
	var rows []models.ModalRow
	for _, rowReq := range req.Rows {
		var components []models.MessageComponent
		for _, compReq := range rowReq.Components {
			comp := models.MessageComponent{
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
		rows = append(rows, models.ModalRow{Components: components})
	}

	modal := &models.ModalComponent{
		CustomID: req.CustomID,
		Type:     req.Type,
		Title:    req.Title,
		Rows:     rows,
	}

	created, err := h.componentService.CreateModal(c.Context(), modal)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(created)
}

// GetModal retrieves a modal by custom_id
// @Summary Get modal
// @Description Retrieves a modal by its custom_id
// @Tags Components
// @Produce json
// @Param customId path string true "Modal Custom ID"
// @Success 200 {object} models.ModalComponent "Modal found"
// @Failure 404 {object} fiber.Map "Modal not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /api/v1/modals/{customId} [get]
func (h *ComponentHandler) GetModal(c *fiber.Ctx) error {
	customID := c.Params("customId")
	if customID == "" {
		return BadRequest(c, "custom_id is required")
	}

	modal, err := h.componentService.GetModalByCustomID(c.Context(), customID)
	if err != nil {
		return HandleServiceError(c, err)
	}
	if modal == nil {
		return NotFound(c, "Modal not found")
	}

	return c.JSON(modal)
}

// DeleteModal deletes a modal by ID
// @Summary Delete modal
// @Description Deletes a modal by its ID
// @Tags Components
// @Param id path string true "Modal ID"
// @Success 204 "Modal deleted successfully"
// @Failure 400 {object} fiber.Map "Invalid modal ID"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /api/v1/modals/{id} [delete]
func (h *ComponentHandler) DeleteModal(c *fiber.Ctx) error {
	modalID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return InvalidUUID(c, "modal ID")
	}

	if err := h.componentService.DeleteModal(c.Context(), modalID); err != nil {
		return HandleServiceError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
