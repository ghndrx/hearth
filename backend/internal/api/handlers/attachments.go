package handlers

import (
	"io"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/services"
)

// AttachmentHandler handles attachment endpoints
type AttachmentHandler struct {
	attachmentService *services.AttachmentService
	channelService    *services.ChannelService
}

// NewAttachmentHandler creates a new attachment handler
func NewAttachmentHandler(
	attachmentService *services.AttachmentService,
	channelService *services.ChannelService,
) *AttachmentHandler {
	return &AttachmentHandler{
		attachmentService: attachmentService,
		channelService:    channelService,
	}
}

// Upload handles file upload
// POST /channels/:id/attachments
// Form fields: file (required), alt_text (optional - for accessibility)
func (h *AttachmentHandler) Upload(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	channelID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	// Get file from form
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "no file provided",
		})
	}

	// Get optional alt text for accessibility (A11Y-004)
	altText := c.FormValue("alt_text")

	// Validate file
	if !services.ValidateFileExtension(file.Filename) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "file type not allowed",
		})
	}

	contentType := file.Header.Get("Content-Type")
	if !services.ValidateContentType(contentType) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "content type not allowed",
		})
	}

	// Upload file with alt text
	attachment, err := h.attachmentService.UploadWithAltText(c.Context(), file, userID, channelID, altText)
	if err != nil {
		if err == services.ErrFileTooLarge {
			return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{
				"error": "file too large",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to upload file",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(attachment)
}

// Get retrieves an attachment by ID
// GET /attachments/:id
func (h *AttachmentHandler) Get(c *fiber.Ctx) error {
	attachmentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid attachment id",
		})
	}

	attachment, err := h.attachmentService.Get(c.Context(), attachmentID)
	if err != nil {
		if err == services.ErrAttachmentNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "attachment not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get attachment",
		})
	}

	return c.JSON(attachment)
}

// Download downloads an attachment file
// GET /attachments/:id/download
func (h *AttachmentHandler) Download(c *fiber.Ctx) error {
	attachmentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid attachment id",
		})
	}

	reader, attachment, err := h.attachmentService.Download(c.Context(), attachmentID)
	if err != nil {
		if err == services.ErrAttachmentNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "attachment not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to download attachment",
		})
	}
	defer reader.Close()

	// Set headers
	c.Set("Content-Type", attachment.ContentType)
	c.Set("Content-Disposition", "attachment; filename=\""+attachment.Filename+"\"")

	// Stream the file
	data, err := io.ReadAll(reader)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to read file",
		})
	}

	return c.Send(data)
}

// GetSignedURL returns a signed URL for the attachment
// GET /attachments/:id/signed-url
func (h *AttachmentHandler) GetSignedURL(c *fiber.Ctx) error {
	attachmentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid attachment id",
		})
	}

	// Default expiry of 1 hour
	expiry := time.Hour

	url, err := h.attachmentService.GetSignedURL(c.Context(), attachmentID, expiry)
	if err != nil {
		if err == services.ErrAttachmentNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "attachment not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to generate signed URL",
		})
	}

	return c.JSON(fiber.Map{
		"url":        url,
		"expires_at": time.Now().Add(expiry).Unix(),
	})
}

// Delete deletes an attachment
// DELETE /attachments/:id
func (h *AttachmentHandler) Delete(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	attachmentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid attachment id",
		})
	}

	err = h.attachmentService.Delete(c.Context(), attachmentID, userID)
	if err != nil {
		switch err {
		case services.ErrAttachmentNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "attachment not found",
			})
		case services.ErrAttachmentAccessDenied:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "access denied",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to delete attachment",
			})
		}
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// GetChannelAttachments retrieves all attachments for a channel
// GET /channels/:id/attachments
func (h *AttachmentHandler) GetChannelAttachments(c *fiber.Ctx) error {
	channelID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	attachments, err := h.attachmentService.GetByChannel(c.Context(), channelID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get attachments",
		})
	}

	if attachments == nil {
		attachments = []*services.Attachment{}
	}

	return c.JSON(attachments)
}
