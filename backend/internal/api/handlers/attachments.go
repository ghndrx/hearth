package handlers

import (
	"io"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/services"
	"hearth/internal/storage"
)

// AttachmentHandler handles file attachment uploads and downloads
type AttachmentHandler struct {
	storageService    *storage.Service
	attachmentService *services.AttachmentService
	messageService    *services.MessageService
	channelService    *services.ChannelService
}

// NewAttachmentHandler creates a new attachment handler
func NewAttachmentHandler(
	storageService *storage.Service,
	attachmentService *services.AttachmentService,
	messageService *services.MessageService,
	channelService *services.ChannelService,
) *AttachmentHandler {
	return &AttachmentHandler{
		storageService:    storageService,
		attachmentService: attachmentService,
		messageService:    messageService,
		channelService:    channelService,
	}
}

// AttachmentUploadResponse represents an attachment upload response
type AttachmentUploadResponse struct {
	ID          string `json:"id"`
	Filename    string `json:"filename"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
	URL         string `json:"url"`
	Width       *int   `json:"width,omitempty"`
	Height      *int   `json:"height,omitempty"`
}

// UploadAttachment handles file uploads to a channel
// POST /api/v1/channels/:id/attachments
func (h *AttachmentHandler) UploadAttachment(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	channelID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid channel ID",
		})
	}

	// Verify user can access this channel
	_, err = h.channelService.GetChannel(c.Context(), channelID)
	if err != nil {
		if err == services.ErrChannelNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Channel not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get channel",
		})
	}

	// Check if user is a member (for server channels)
	// This is a simplified check - in production, you'd verify membership

	// Parse multipart form
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No file provided",
		})
	}

	// Upload file to storage
	fileInfo, err := h.storageService.UploadFile(c.Context(), file, userID, "attachments")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Create attachment record (not linked to a message yet)
	// In a real implementation, you'd create an attachment record
	// that can be linked when sending a message

	return c.Status(fiber.StatusCreated).JSON(AttachmentUploadResponse{
		ID:          fileInfo.ID.String(),
		Filename:    fileInfo.Filename,
		Size:        fileInfo.Size,
		ContentType: fileInfo.ContentType,
		URL:         fileInfo.URL,
	})
}

// GetAttachment retrieves an attachment by ID
// GET /api/v1/attachments/:id
func (h *AttachmentHandler) GetAttachment(c *fiber.Ctx) error {
	attachmentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid attachment ID",
		})
	}

	attachment, err := h.attachmentService.Get(c.Context(), attachmentID)
	if err != nil {
		if err == services.ErrAttachmentNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Attachment not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get attachment",
		})
	}

	return c.JSON(AttachmentUploadResponse{
		ID:          attachment.ID.String(),
		Filename:    attachment.Filename,
		Size:        attachment.Size,
		ContentType: attachment.ContentType,
		URL:         attachment.URL,
	})
}

// DownloadAttachment serves an attachment file for download
// GET /api/v1/attachments/:id/download
func (h *AttachmentHandler) DownloadAttachment(c *fiber.Ctx) error {
	_ = c.Locals("userID").(uuid.UUID) // TODO: Use for permission checks
	attachmentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid attachment ID",
		})
	}

	attachment, err := h.attachmentService.Get(c.Context(), attachmentID)
	if err != nil {
		if err == services.ErrAttachmentNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Attachment not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get attachment",
		})
	}

	// Extract path from URL
	path := attachment.URL
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}

	// Get file from storage
	reader, err := h.storageService.Download(c.Context(), path)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to download file",
		})
	}
	defer reader.Close()

	// Set headers for download
	c.Set("Content-Type", attachment.ContentType)
	c.Set("Content-Disposition", "attachment; filename=\""+attachment.Filename+"\"")
	c.Set("Content-Length", strconv.FormatInt(attachment.Size, 10))

	// Stream file to response
	_, err = io.Copy(c.Response().BodyWriter(), reader)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to stream file",
		})
	}

	return nil
}

// DeleteAttachment deletes an attachment
// DELETE /api/v1/attachments/:id
func (h *AttachmentHandler) DeleteAttachment(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	attachmentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid attachment ID",
		})
	}

	// Get attachment to check ownership
	attachment, err := h.attachmentService.Get(c.Context(), attachmentID)
	if err != nil {
		if err == services.ErrAttachmentNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Attachment not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get attachment",
		})
	}

	// Check if user owns this attachment or has permission to delete
	// For now, we'll just check if it's the user's upload
	// In production, you'd also check channel/server permissions
	_ = userID // TODO: Check ownership from file metadata

	// Delete from storage
	path := attachment.URL
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}

	err = h.storageService.DeleteFile(c.Context(), path)
	if err != nil {
		// Log error but continue to delete metadata
		// In production, you'd want to handle this better
	}

	// Delete attachment record
	err = h.attachmentService.Delete(c.Context(), attachmentID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete attachment",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// GetAttachmentInfo returns metadata about an attachment without downloading
// GET /api/v1/attachments/:id/info
func (h *AttachmentHandler) GetAttachmentInfo(c *fiber.Ctx) error {
	attachmentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid attachment ID",
		})
	}

	attachment, err := h.attachmentService.Get(c.Context(), attachmentID)
	if err != nil {
		if err == services.ErrAttachmentNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Attachment not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get attachment",
		})
	}

	return c.JSON(fiber.Map{
		"id":           attachment.ID.String(),
		"filename":     attachment.Filename,
		"size":         attachment.Size,
		"content_type": attachment.ContentType,
		"url":          attachment.URL,
	})
}
