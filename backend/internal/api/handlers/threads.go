package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// ThreadHandler handles thread-related HTTP requests
type ThreadHandler struct {
	threadService *services.ThreadService
}

// NewThreadHandler creates a new thread handler
func NewThreadHandler(threadService *services.ThreadService) *ThreadHandler {
	return &ThreadHandler{
		threadService: threadService,
	}
}

// CreateThread creates a new thread in a channel
// POST /channels/:id/threads
func (h *ThreadHandler) CreateThread(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	channelID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	var req models.CreateThreadRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "name is required",
		})
	}

	thread, err := h.threadService.CreateThread(c.Context(), channelID, userID, req.Name, req.AutoArchive)
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
		case services.ErrInvalidAutoArchive:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid auto archive duration",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to create thread",
			})
		}
	}

	return c.Status(fiber.StatusCreated).JSON(thread)
}

// GetThread retrieves a thread by ID
// GET /threads/:id
func (h *ThreadHandler) GetThread(c *fiber.Ctx) error {
	threadID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid thread id",
		})
	}

	thread, err := h.threadService.GetThread(c.Context(), threadID)
	if err != nil {
		if err == services.ErrThreadNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "thread not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get thread",
		})
	}

	return c.JSON(thread)
}

// GetThreadMessages retrieves messages from a thread
// GET /threads/:id/messages
func (h *ThreadHandler) GetThreadMessages(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	threadID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid thread id",
		})
	}

	var before *uuid.UUID
	if b := c.Query("before"); b != "" {
		if id, err := uuid.Parse(b); err == nil {
			before = &id
		}
	}

	limit := c.QueryInt("limit", 50)

	messages, err := h.threadService.GetThreadMessages(c.Context(), threadID, userID, before, limit)
	if err != nil {
		switch err {
		case services.ErrThreadNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "thread not found",
			})
		case services.ErrNotServerMember:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "not a server member",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to get thread messages",
			})
		}
	}

	return c.JSON(messages)
}

// SendThreadMessage sends a message to a thread
// POST /threads/:id/messages
func (h *ThreadHandler) SendThreadMessage(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	threadID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid thread id",
		})
	}

	var req models.CreateThreadMessageRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.Content == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "content is required",
		})
	}

	message, err := h.threadService.SendThreadMessage(c.Context(), threadID, userID, req.Content)
	if err != nil {
		switch err {
		case services.ErrThreadNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "thread not found",
			})
		case services.ErrThreadArchived:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "thread is archived",
			})
		case services.ErrThreadLocked:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "thread is locked",
			})
		case services.ErrNotServerMember:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "not a server member",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to send message",
			})
		}
	}

	return c.Status(fiber.StatusCreated).JSON(message)
}

// ArchiveThread archives a thread
// POST /threads/:id/archive
func (h *ThreadHandler) ArchiveThread(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	threadID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid thread id",
		})
	}

	if err := h.threadService.ArchiveThread(c.Context(), threadID, userID); err != nil {
		switch err {
		case services.ErrThreadNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "thread not found",
			})
		case services.ErrNotThreadOwner:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "not authorized to archive this thread",
			})
		case services.ErrNotServerMember:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "not a server member",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to archive thread",
			})
		}
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// UnarchiveThread unarchives a thread
// POST /threads/:id/unarchive
func (h *ThreadHandler) UnarchiveThread(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	threadID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid thread id",
		})
	}

	if err := h.threadService.UnarchiveThread(c.Context(), threadID, userID); err != nil {
		switch err {
		case services.ErrThreadNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "thread not found",
			})
		case services.ErrNotThreadOwner:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "not authorized to unarchive this thread",
			})
		case services.ErrNotServerMember:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "not a server member",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to unarchive thread",
			})
		}
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// GetChannelThreads retrieves all threads in a channel
// GET /channels/:id/threads
func (h *ThreadHandler) GetChannelThreads(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	channelID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	includeArchived := c.QueryBool("include_archived", false)

	threads, err := h.threadService.GetChannelThreads(c.Context(), channelID, userID, includeArchived)
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
				"error": "failed to get threads",
			})
		}
	}

	return c.JSON(threads)
}

// JoinThread adds the current user to a thread
// POST /threads/:id/join
func (h *ThreadHandler) JoinThread(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	threadID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid thread id",
		})
	}

	if err := h.threadService.JoinThread(c.Context(), threadID, userID); err != nil {
		if err == services.ErrThreadNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "thread not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to join thread",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// LeaveThread removes the current user from a thread
// DELETE /threads/:id/members/@me
func (h *ThreadHandler) LeaveThread(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	threadID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid thread id",
		})
	}

	if err := h.threadService.LeaveThread(c.Context(), threadID, userID); err != nil {
		if err == services.ErrThreadNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "thread not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to leave thread",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// DeleteThread deletes a thread
// DELETE /threads/:id
func (h *ThreadHandler) DeleteThread(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	threadID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid thread id",
		})
	}

	if err := h.threadService.DeleteThread(c.Context(), threadID, userID); err != nil {
		switch err {
		case services.ErrThreadNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "thread not found",
			})
		case services.ErrNotThreadOwner:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "not authorized to delete this thread",
			})
		case services.ErrNotServerMember:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "not a server member",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to delete thread",
			})
		}
	}

	return c.SendStatus(fiber.StatusNoContent)
}
