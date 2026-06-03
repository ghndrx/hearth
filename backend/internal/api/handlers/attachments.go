package handlers

import (
	"io"
	"log"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/services"
)

// sanitizeFilename removes dangerous characters to prevent HTTP header injection
func sanitizeFilename(name string) string {
	name = path.Base(name)
	// Remove control characters, quotes, backslashes, and newlines
	re := regexp.MustCompile(`[\x00-\x1f\"\\\r\n]`)
	return re.ReplaceAllString(name, "")
}

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
// @Summary Upload file attachment
// @Description Uploads a file attachment to a channel with optional alt text for accessibility
// @Tags Attachments
// @Accept multipart/form-data
// @Produce json
// @Param id path string true "Channel ID"
// @Param file formData file true "File to upload"
// @Param alt_text formData string false "Alt text for accessibility"
// @Success 201 {object} services.Attachment "Attachment uploaded successfully"
// @Failure 400 {object} fiber.Map "Invalid channel ID or file type not allowed"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 413 {object} fiber.Map "File too large"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{id}/attachments [post]
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
// @Summary Get attachment by ID
// @Description Returns attachment metadata by its ID
// @Tags Attachments
// @Produce json
// @Param id path string true "Attachment ID"
// @Success 200 {object} services.Attachment "Attachment metadata"
// @Failure 400 {object} fiber.Map "Invalid attachment ID"
// @Failure 404 {object} fiber.Map "Attachment not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /attachments/{id} [get]
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
// @Summary Download attachment file
// @Description Downloads the actual file content of an attachment
// @Tags Attachments
// @Produce octet-stream
// @Param id path string true "Attachment ID"
// @Success 200 {file} binary "File content"
// @Failure 400 {object} fiber.Map "Invalid attachment ID"
// @Failure 404 {object} fiber.Map "Attachment not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /attachments/{id}/download [get]
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
	defer func() {
		if err := reader.Close(); err != nil {
			log.Printf("Failed to close attachment reader: %v", err)
		}
	}()

	// Set headers
	// Sanitize Content-Type: only use stored value for known safe types
	safeContentType := "application/octet-stream"
	ct := strings.ToLower(attachment.ContentType)
	if strings.HasPrefix(ct, "image/") || strings.HasPrefix(ct, "video/") || strings.HasPrefix(ct, "audio/") || ct == "application/pdf" {
		safeContentType = attachment.ContentType
	}
	c.Set("Content-Type", safeContentType)
	// Sanitize filename to prevent HTTP header injection
	safeFilename := sanitizeFilename(attachment.Filename)
	c.Set("Content-Disposition", "attachment; filename=\""+safeFilename+"\"")

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
// @Summary Get signed URL for attachment
// @Description Returns a temporary signed URL for accessing an attachment (expires in 1 hour)
// @Tags Attachments
// @Produce json
// @Param id path string true "Attachment ID"
// @Success 200 {object} fiber.Map "Signed URL and expiration time"
// @Failure 400 {object} fiber.Map "Invalid attachment ID"
// @Failure 404 {object} fiber.Map "Attachment not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /attachments/{id}/signed-url [get]
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
// @Summary Delete attachment
// @Description Deletes an attachment by its ID (requires ownership or permissions)
// @Tags Attachments
// @Param id path string true "Attachment ID"
// @Success 204 "Attachment deleted successfully"
// @Failure 400 {object} fiber.Map "Invalid attachment ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Access denied"
// @Failure 404 {object} fiber.Map "Attachment not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /attachments/{id} [delete]
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
// @Summary Get channel attachments
// @Description Returns all attachments for a specific channel
// @Tags Attachments
// @Produce json
// @Param id path string true "Channel ID"
// @Success 200 {array} services.Attachment "List of attachments"
// @Failure 400 {object} fiber.Map "Invalid channel ID"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{id}/attachments [get]
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
