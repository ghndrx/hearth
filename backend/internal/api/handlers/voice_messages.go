package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/services"
)

// VoiceMessageHandler handles voice message endpoints
type VoiceMessageHandler struct {
	voiceMessageService *services.VoiceMessageService
	channelService     *services.ChannelService
}

// NewVoiceMessageHandler creates a new voice message handler
func NewVoiceMessageHandler(
	voiceMessageService *services.VoiceMessageService,
	channelService *services.ChannelService,
) *VoiceMessageHandler {
	return &VoiceMessageHandler{
		voiceMessageService: voiceMessageService,
		channelService:     channelService,
	}
}

// Upload handles voice message upload
// @Summary Upload voice message
// @Description Uploads a voice message recording to a channel
// @Tags Voice Messages
// @Accept multipart/form-data
// @Produce json
// @Param id path string true "Channel ID"
// @Param file formData file true "Voice message file (WEBM, OGG, OPUS, MP3)"
// @Success 201 {object} models.VoiceMessageResponse "Voice message uploaded successfully"
// @Failure 400 {object} fiber.Map "Invalid channel ID or file type not allowed"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 413 {object} fiber.Map "File too large"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{id}/voice-messages [post]
func (h *VoiceMessageHandler) Upload(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	channelIDStr := c.Params("id")
	channelID, err := uuid.Parse(channelIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel ID",
		})
	}

	// Get the file from the request
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "no file provided",
		})
	}

	// Upload the voice message
	voiceMessage, err := h.voiceMessageService.UploadVoiceMessage(c.Context(), file, userID, channelID)
	if err != nil {
		switch err {
		case services.ErrVoiceMessageTooLarge:
			return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{
				"error": err.Error(),
			})
		case services.ErrVoiceMessageFormat:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		case services.ErrVoiceMessageDuration:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to upload voice message",
			})
		}
	}

	return c.Status(fiber.StatusCreated).JSON(voiceMessage.ToResponse())
}

// GetChannelVoiceMessages retrieves all voice messages for a channel
// @Summary Get channel voice messages
// @Description Retrieves all voice messages for a channel
// @Tags Voice Messages
// @Produce json
// @Param id path string true "Channel ID"
// @Param limit query int false "Maximum number of messages to return (default 50)"
// @Success 200 {array} models.VoiceMessageResponse "Voice messages retrieved successfully"
// @Failure 400 {object} fiber.Map "Invalid channel ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{id}/voice-messages [get]
func (h *VoiceMessageHandler) GetChannelVoiceMessages(c *fiber.Ctx) error {
	channelIDStr := c.Params("id")
	channelID, err := uuid.Parse(channelIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel ID",
		})
	}

	limit := c.QueryInt("limit", 50)
	if limit <= 0 {
		limit = 50
	}

	voiceMessages, err := h.voiceMessageService.GetChannelVoiceMessages(c.Context(), channelID, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get voice messages",
		})
	}

	// Convert to response format
	responses := make([]map[string]interface{}, len(voiceMessages))
	for i, vm := range voiceMessages {
		responses[i] = map[string]interface{}{
			"id":            vm.ID.String(),
			"channel_id":    vm.ChannelID.String(),
			"user_id":       vm.UserID.String(),
			"file_url":      vm.FileURL,
			"duration_ms":   vm.DurationMs,
			"waveform_data": vm.WaveformData,
			"transcription": vm.Transcription,
			"created_at":    vm.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return c.JSON(responses)
}

// GetVoiceMessage retrieves a specific voice message
// @Summary Get voice message
// @Description Retrieves a specific voice message by ID
// @Tags Voice Messages
// @Produce json
// @Param id path string true "Voice Message ID"
// @Success 200 {object} models.VoiceMessageResponse "Voice message retrieved successfully"
// @Failure 400 {object} fiber.Map "Invalid voice message ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 404 {object} fiber.Map "Voice message not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /voice-messages/{id} [get]
func (h *VoiceMessageHandler) GetVoiceMessage(c *fiber.Ctx) error {
	voiceMessageIDStr := c.Params("id")
	voiceMessageID, err := uuid.Parse(voiceMessageIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid voice message ID",
		})
	}

	voiceMessage, err := h.voiceMessageService.GetVoiceMessage(c.Context(), voiceMessageID)
	if err != nil {
		if err == services.ErrVoiceMessageNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "voice message not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get voice message",
		})
	}

	return c.JSON(voiceMessage.ToResponse())
}

// DeleteVoiceMessage deletes a voice message
// @Summary Delete voice message
// @Description Deletes a voice message (only the owner can delete)
// @Tags Voice Messages
// @Produce json
// @Param id path string true "Voice Message ID"
// @Success 204 "Voice message deleted successfully"
// @Failure 400 {object} fiber.Map "Invalid voice message ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Cannot delete another user's voice message"
// @Failure 404 {object} fiber.Map "Voice message not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /voice-messages/{id} [delete]
func (h *VoiceMessageHandler) DeleteVoiceMessage(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	voiceMessageIDStr := c.Params("id")
	voiceMessageID, err := uuid.Parse(voiceMessageIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid voice message ID",
		})
	}

	err = h.voiceMessageService.DeleteVoiceMessage(c.Context(), voiceMessageID, userID)
	if err != nil {
		if err == services.ErrVoiceMessageNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "voice message not found",
			})
		}
		if err.Error() == "cannot delete another user's voice message" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to delete voice message",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
