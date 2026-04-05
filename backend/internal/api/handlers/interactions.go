package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// InteractionHandler handles interaction HTTP requests
type InteractionHandler struct {
	interactionService *services.InteractionService
}

// NewInteractionHandler creates a new interaction handler
func NewInteractionHandler(interactionService *services.InteractionService) *InteractionHandler {
	return &InteractionHandler{
		interactionService: interactionService,
	}
}

// InteractionRequest represents an incoming interaction request
type InteractionRequest struct {
	ID        uuid.UUID   `json:"id"`
	Type      int         `json:"type"`
	Data      interface{} `json:"data,omitempty"`
	UserID    string      `json:"user_id"`
	ServerID  *string     `json:"guild_id,omitempty"`
	ChannelID string      `json:"channel_id"`
	Member    interface{} `json:"member,omitempty"`
	Token     string      `json:"token"`
	AppID     string      `json:"application_id"`
	Message   interface{} `json:"message,omitempty"`
}

func parseInteraction(req *InteractionRequest) (*models.Interaction, error) {
	interaction := &models.Interaction{
		ID:      req.ID,
		Type:    models.InteractionType(req.Type),
		Token:   req.Token,
		Message: nil,
	}

	if req.UserID != "" {
		if id, err := uuid.Parse(req.UserID); err == nil {
			interaction.UserID = id
		}
	}
	if req.ChannelID != "" {
		if id, err := uuid.Parse(req.ChannelID); err == nil {
			interaction.ChannelID = id
		}
	}
	if req.AppID != "" {
		if id, err := uuid.Parse(req.AppID); err == nil {
			interaction.AppID = id
		}
	}
	if req.ServerID != nil && *req.ServerID != "" {
		if id, err := uuid.Parse(*req.ServerID); err == nil {
			interaction.ServerID = &id
		}
	}

	// Parse data based on type
	switch interaction.Type {
	case models.InteractionTypeApplicationCommand:
		if dataMap, ok := req.Data.(map[string]interface{}); ok {
			interaction.Data = dataMap
		}
	case models.InteractionTypeAutocomplete:
		if dataMap, ok := req.Data.(map[string]interface{}); ok {
			interaction.Data = dataMap
		}
	case models.InteractionTypeMessageComponent:
		if dataMap, ok := req.Data.(map[string]interface{}); ok {
			interaction.Data = dataMap
		}
	case models.InteractionTypeModalSubmit:
		if dataMap, ok := req.Data.(map[string]interface{}); ok {
			interaction.Data = dataMap
		}
	}

	return interaction, nil
}

// HandleInteraction handles an incoming interaction
// POST /api/v1/interactions
func (h *InteractionHandler) HandleInteraction(c *fiber.Ctx) error {
	var req InteractionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.ID == uuid.Nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "interaction ID is required",
		})
	}

	interaction, err := parseInteraction(&req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	response, err := h.interactionService.HandleInteraction(c.Context(), interaction)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(response)
}

// RespondToInteraction creates a follow-up response using interaction_id
// POST /api/v1/interactions/:interaction_id/callback
func (h *InteractionHandler) RespondToInteraction(c *fiber.Ctx) error {
	interactionIDStr := c.Params("interaction_id")
	if interactionIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "interaction_id is required",
		})
	}

	// Parse as UUID first, then try as token string
	token := interactionIDStr

	var resp models.InteractionResponse
	if err := c.BodyParser(&resp); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if err := h.interactionService.CreateResponse(c.Context(), token, &resp); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// RespondToInteractionWithToken creates a follow-up response using interaction_id and token
// POST /api/v1/interactions/:interaction_id/callback/:token
func (h *InteractionHandler) RespondToInteractionWithToken(c *fiber.Ctx) error {
	_ = c.Params("interaction_id") // Could be used for validation
	token := c.Params("token")
	if token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "token is required",
		})
	}

	var resp models.InteractionResponse
	if err := c.BodyParser(&resp); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if err := h.interactionService.CreateResponse(c.Context(), token, &resp); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// EditInteractionResponse edits a follow-up response
// PATCH /api/v1/interactions/:interaction_id/messages/:messageId
func (h *InteractionHandler) EditInteractionResponse(c *fiber.Ctx) error {
	_ = c.Params("interaction_id") // Could be used for validation
	messageID := c.Params("messageId")

	var resp models.InteractionResponse
	if err := c.BodyParser(&resp); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Get token from query param or body if needed, for now use interaction_id as token
	token := c.Query("token", c.Params("interaction_id"))

	if err := h.interactionService.EditResponse(c.Context(), token, &resp); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message_id": messageID,
		"updated":    true,
	})
}

// DeleteInteractionResponse deletes a follow-up response
// DELETE /api/v1/interactions/:interaction_id/messages/:messageId
func (h *InteractionHandler) DeleteInteractionResponse(c *fiber.Ctx) error {
	_ = c.Params("interaction_id") // Could be used for validation
	messageID := c.Params("messageId")

	// Get token from query param if needed
	token := c.Query("token", c.Params("interaction_id"))

	if err := h.interactionService.DeleteResponse(c.Context(), token, messageID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ModalSubmitRequest represents a modal submit request
type ModalSubmitRequest struct {
	CustomID string                      `json:"custom_id"`
	Values   map[string]string           `json:"values"`
}

// HandleModalSubmit processes modal form submissions
// POST /api/v1/interactions/modals/submit
func (h *InteractionHandler) HandleModalSubmit(c *fiber.Ctx) error {
	var req ModalSubmitRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.CustomID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "custom_id is required",
		})
	}

	if req.Values == nil {
		req.Values = make(map[string]string)
	}

	// Get the user ID from the authenticated context
	userID := c.Locals("userID").(uuid.UUID)

	// Create a modal submit interaction
	interaction := &models.Interaction{
		ID:      uuid.New(),
		Type:    models.InteractionTypeModalSubmit,
		Token:   req.CustomID,
		UserID:  userID,
		Message: nil,
	}

	// The Values map contains the form field values keyed by custom_id
	interaction.Data = req.Values

	response, err := h.interactionService.HandleInteraction(c.Context(), interaction)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(response)
}
