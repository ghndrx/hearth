package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/services"
)

// ForwardHandlers handles message forwarding HTTP requests
type ForwardHandlers struct {
	forwardService *services.ForwardService
}

// NewForwardHandlers creates new forward handlers
func NewForwardHandlers(forwardService *services.ForwardService) *ForwardHandlers {
	return &ForwardHandlers{
		forwardService: forwardService,
	}
}

// ForwardMessage forwards a message to another channel
// @Summary Forward message
// @Description Forwards a message to a destination channel
// @Tags Messages
// @Accept json
// @Produce json
// @Param messageID path string true "Message ID to forward"
// @Param request body ForwardMessageRequest true "Forward request"
// @Success 201 {object} models.ForwardedMessage "Message forwarded successfully"
// @Failure 400 {object} fiber.Map "Invalid request"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Forbidden"
// @Failure 404 {object} fiber.Map "Message or channel not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /messages/{messageID}/forward [post]
func (h *ForwardHandlers) ForwardMessage(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	messageID, err := uuid.Parse(c.Params("messageID"))
	if err != nil {
		return InvalidUUID(c, "message ID")
	}

	var req struct {
		DestinationChannelID string `json:"destination_channel_id"`
		Comment              string `json:"comment,omitempty"`
	}
	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	destChannelID, err := uuid.Parse(req.DestinationChannelID)
	if err != nil {
		return InvalidUUID(c, "destination channel ID")
	}

	forwardedMsg, err := h.forwardService.ForwardMessage(
		c.Context(),
		messageID,
		userID,
		destChannelID,
		req.Comment,
	)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(forwardedMsg)
}

// GetMessageForwards returns all forwards of a message
// @Summary Get message forwards
// @Description Returns all forwards of a message
// @Tags Messages
// @Produce json
// @Param messageID path string true "Message ID"
// @Success 200 {array} models.ForwardedMessage "List of forwards"
// @Failure 400 {object} fiber.Map "Invalid message ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /messages/{messageID}/forwards [get]
func (h *ForwardHandlers) GetMessageForwards(c *fiber.Ctx) error {
	messageID, err := uuid.Parse(c.Params("messageID"))
	if err != nil {
		return InvalidUUID(c, "message ID")
	}

	forwards, err := h.forwardService.GetForwardsByOriginalMessage(c.Context(), messageID)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(forwards)
}

// ForwardMessageRequest represents a request to forward a message
type ForwardMessageRequest struct {
	DestinationChannelID string `json:"destination_channel_id"`
	Comment              string `json:"comment,omitempty"`
}
