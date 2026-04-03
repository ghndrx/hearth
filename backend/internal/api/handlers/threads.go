package handlers

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// ThreadServiceInterface defines the interface for thread service operations
type ThreadServiceInterface interface {
	CreateThread(ctx context.Context, channelID, creatorID uuid.UUID, name string, autoArchive *int, parentMessageID *uuid.UUID, tagIDs []uuid.UUID) (*models.Thread, error)
	UpdateThread(ctx context.Context, threadID, requesterID uuid.UUID, req models.UpdateThreadRequest) (*models.Thread, error)
	GetThread(ctx context.Context, threadID uuid.UUID) (*models.Thread, error)
	GetThreadMessages(ctx context.Context, threadID, requesterID uuid.UUID, before *uuid.UUID, limit int) ([]*models.ThreadMessage, error)
	SendThreadMessage(ctx context.Context, threadID, authorID uuid.UUID, content string) (*models.ThreadMessage, error)
	ArchiveThread(ctx context.Context, threadID, requesterID uuid.UUID) error
	UnarchiveThread(ctx context.Context, threadID, requesterID uuid.UUID) error
	GetChannelThreads(ctx context.Context, channelID, requesterID uuid.UUID, includeArchived bool) ([]*models.Thread, error)
	JoinThread(ctx context.Context, threadID, userID uuid.UUID) error
	LeaveThread(ctx context.Context, threadID, userID uuid.UUID) error
	DeleteThread(ctx context.Context, threadID, requesterID uuid.UUID) error
	GetNotificationPreference(ctx context.Context, threadID, userID uuid.UUID) (*models.ThreadNotificationPreference, error)
	SetNotificationPreference(ctx context.Context, threadID, userID uuid.UUID, level models.ThreadNotificationLevel) error
	EnterThread(ctx context.Context, threadID, userID uuid.UUID) (*models.ThreadPresenceResponse, error)
	ExitThread(ctx context.Context, threadID, userID uuid.UUID) error
	GetActiveViewers(ctx context.Context, threadID uuid.UUID) (*models.ThreadPresenceResponse, error)
	HeartbeatPresence(ctx context.Context, threadID, userID uuid.UUID) error
}

// ThreadHandler handles thread-related HTTP requests
type ThreadHandler struct {
	threadService ThreadServiceInterface
}

// NewThreadHandler creates a new thread handler
func NewThreadHandler(threadService ThreadServiceInterface) *ThreadHandler {
	return &ThreadHandler{
		threadService: threadService,
	}
}

// CreateThread creates a new thread in a channel
// @Summary Create a new thread
// @Description Creates a new thread in the specified channel with the given name and optional auto-archive duration
// @Tags Threads
// @Accept json
// @Produce json
// @Param id path string true "Channel ID"
// @Param body body models.CreateThreadRequest true "Thread creation request"
// @Success 201 {object} models.Thread "Thread created successfully"
// @Failure 400 {object} fiber.Map "Invalid channel ID or request body"
// @Failure 403 {object} fiber.Map "Not a server member"
// @Failure 404 {object} fiber.Map "Channel not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{id}/threads [post]
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

	var parentMessageID *uuid.UUID
	if req.ParentMessageID != nil {
		if id, parseErr := uuid.Parse(*req.ParentMessageID); parseErr == nil {
			parentMessageID = &id
		}
	}

	thread, err := h.threadService.CreateThread(c.Context(), channelID, userID, req.Name, req.AutoArchive, parentMessageID, req.TagIDs)
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
// @Summary Get thread by ID
// @Description Retrieves a thread's details by its ID
// @Tags Threads
// @Produce json
// @Param id path string true "Thread ID"
// @Success 200 {object} models.Thread "Thread details"
// @Failure 400 {object} fiber.Map "Invalid thread ID"
// @Failure 404 {object} fiber.Map "Thread not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /threads/{id} [get]
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
// @Summary Get thread messages
// @Description Retrieves paginated messages from a thread with optional before cursor and limit
// @Tags Threads
// @Produce json
// @Param id path string true "Thread ID"
// @Param before query string false "Message ID to fetch messages before (for pagination)"
// @Param limit query int false "Number of messages to fetch (default: 50, max: 100)"
// @Success 200 {array} models.Message "List of messages"
// @Failure 400 {object} fiber.Map "Invalid thread ID"
// @Failure 403 {object} fiber.Map "Not a server member"
// @Failure 404 {object} fiber.Map "Thread not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /threads/{id}/messages [get]
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
// @Summary Send message to thread
// @Description Sends a new message to the specified thread
// @Tags Threads
// @Accept json
// @Produce json
// @Param id path string true "Thread ID"
// @Param body body models.CreateThreadMessageRequest true "Message content"
// @Success 201 {object} models.Message "Message sent successfully"
// @Failure 400 {object} fiber.Map "Invalid thread ID or request body"
// @Failure 403 {object} fiber.Map "Not a server member, thread is archived, or thread is locked"
// @Failure 404 {object} fiber.Map "Thread not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /threads/{id}/messages [post]
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
// @Summary Archive a thread
// @Description Archives the specified thread, preventing new messages from being sent
// @Tags Threads
// @Param id path string true "Thread ID"
// @Success 204 "Thread archived successfully"
// @Failure 400 {object} fiber.Map "Invalid thread ID"
// @Failure 403 {object} fiber.Map "Not authorized to archive this thread or not a server member"
// @Failure 404 {object} fiber.Map "Thread not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /threads/{id}/archive [post]
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
// @Summary Unarchive a thread
// @Description Unarchives the specified thread, allowing new messages to be sent again
// @Tags Threads
// @Param id path string true "Thread ID"
// @Success 204 "Thread unarchived successfully"
// @Failure 400 {object} fiber.Map "Invalid thread ID"
// @Failure 403 {object} fiber.Map "Not authorized to unarchive this thread or not a server member"
// @Failure 404 {object} fiber.Map "Thread not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /threads/{id}/unarchive [post]
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
// @Summary Get channel threads
// @Description Retrieves all threads in the specified channel, optionally including archived threads
// @Tags Threads
// @Produce json
// @Param id path string true "Channel ID"
// @Param include_archived query bool false "Include archived threads in results"
// @Success 200 {array} models.Thread "List of threads"
// @Failure 400 {object} fiber.Map "Invalid channel ID"
// @Failure 403 {object} fiber.Map "Not a server member"
// @Failure 404 {object} fiber.Map "Channel not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{id}/threads [get]
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
// @Summary Join a thread
// @Description Adds the current user as a member of the specified thread
// @Tags Threads
// @Param id path string true "Thread ID"
// @Success 204 "Successfully joined thread"
// @Failure 400 {object} fiber.Map "Invalid thread ID"
// @Failure 404 {object} fiber.Map "Thread not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /threads/{id}/join [post]
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
// @Summary Leave a thread
// @Description Removes the current user from the specified thread
// @Tags Threads
// @Param id path string true "Thread ID"
// @Success 204 "Successfully left thread"
// @Failure 400 {object} fiber.Map "Invalid thread ID"
// @Failure 404 {object} fiber.Map "Thread not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /threads/{id}/leave [post]
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
// @Summary Delete a thread
// @Description Deletes the specified thread permanently
// @Tags Threads
// @Param id path string true "Thread ID"
// @Success 204 "Thread deleted successfully"
// @Failure 400 {object} fiber.Map "Invalid thread ID"
// @Failure 403 {object} fiber.Map "Not authorized to delete this thread or not a server member"
// @Failure 404 {object} fiber.Map "Thread not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /threads/{id} [delete]
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

// UpdateThread updates a thread's properties
// @Summary Update a thread
// @Description Updates the specified thread's name, archived state, locked state, or auto-archive duration
// @Tags Threads
// @Accept json
// @Produce json
// @Param id path string true "Thread ID"
// @Param body body models.UpdateThreadRequest true "Thread update request"
// @Success 200 {object} models.Thread "Updated thread"
// @Failure 400 {object} fiber.Map "Invalid thread ID or request body"
// @Failure 403 {object} fiber.Map "Not authorized to update this thread"
// @Failure 404 {object} fiber.Map "Thread not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /threads/{id} [patch]
func (h *ThreadHandler) UpdateThread(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	threadID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid thread id",
		})
	}

	var req models.UpdateThreadRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	thread, err := h.threadService.UpdateThread(c.Context(), threadID, userID, req)
	if err != nil {
		switch err {
		case services.ErrThreadNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "thread not found",
			})
		case services.ErrNotThreadOwner:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "not authorized to update this thread",
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
				"error": "failed to update thread",
			})
		}
	}

	return c.JSON(thread)
}

// ============================================================================
// Thread Notification Preferences
// ============================================================================

// GetNotificationPreference gets the user's notification preference for a thread
// @Summary Get notification preference
// @Description Gets the current user's notification preference for the specified thread
// @Tags Threads
// @Produce json
// @Param id path string true "Thread ID"
// @Success 200 {object} models.ThreadNotificationPreference "Notification preference"
// @Failure 400 {object} fiber.Map "Invalid thread ID"
// @Failure 404 {object} fiber.Map "Thread not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /threads/{id}/notifications [get]
func (h *ThreadHandler) GetNotificationPreference(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	threadID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid thread id",
		})
	}

	pref, err := h.threadService.GetNotificationPreference(c.Context(), threadID, userID)
	if err != nil {
		if err == services.ErrThreadNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "thread not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get notification preference",
		})
	}

	return c.JSON(pref)
}

// SetNotificationPreference sets the user's notification preference for a thread
// @Summary Set notification preference
// @Description Sets the current user's notification preference for the specified thread
// @Tags Threads
// @Accept json
// @Produce json
// @Param id path string true "Thread ID"
// @Param body body models.UpdateThreadNotificationRequest true "Notification preference request"
// @Success 200 {object} fiber.Map "Notification preference updated"
// @Failure 400 {object} fiber.Map "Invalid thread ID or request body"
// @Failure 404 {object} fiber.Map "Thread not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /threads/{id}/notifications [put]
func (h *ThreadHandler) SetNotificationPreference(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	threadID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid thread id",
		})
	}

	var req models.UpdateThreadNotificationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Validate level
	switch req.Level {
	case models.ThreadNotifyAll, models.ThreadNotifyMentions, models.ThreadNotifyNone:
		// Valid
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid notification level, must be 'all', 'mentions', or 'none'",
		})
	}

	if err := h.threadService.SetNotificationPreference(c.Context(), threadID, userID, req.Level); err != nil {
		if err == services.ErrThreadNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "thread not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to set notification preference",
		})
	}

	return c.JSON(fiber.Map{
		"thread_id": threadID,
		"level":     req.Level,
	})
}

// ============================================================================
// Thread Presence (Active Viewers)
// ============================================================================

// EnterThread marks the user as viewing a thread and returns active viewers
// @Summary Enter thread presence
// @Description Marks the user as currently viewing the thread and returns the list of active viewers
// @Tags Threads
// @Produce json
// @Param id path string true "Thread ID"
// @Success 200 {object} models.ThreadPresenceResponse "Thread presence response with active viewers"
// @Failure 400 {object} fiber.Map "Invalid thread ID"
// @Failure 404 {object} fiber.Map "Thread not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /threads/{id}/presence [post]
func (h *ThreadHandler) EnterThread(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	threadID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid thread id",
		})
	}

	response, err := h.threadService.EnterThread(c.Context(), threadID, userID)
	if err != nil {
		if err == services.ErrThreadNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "thread not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to enter thread",
		})
	}

	return c.JSON(response)
}

// ExitThreadPresence removes the user's presence from a thread (stops viewing)
// @Summary Exit thread presence
// @Description Removes the user from the thread's active viewers list
// @Tags Threads
// @Param id path string true "Thread ID"
// @Success 204 "Successfully exited thread presence"
// @Failure 400 {object} fiber.Map "Invalid thread ID"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /threads/{id}/presence [delete]
func (h *ThreadHandler) ExitThreadPresence(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	threadID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid thread id",
		})
	}

	if err := h.threadService.ExitThread(c.Context(), threadID, userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to exit thread",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// GetActiveViewers gets users currently viewing a thread
// @Summary Get active viewers
// @Description Returns a list of users currently viewing the thread
// @Tags Threads
// @Produce json
// @Param id path string true "Thread ID"
// @Success 200 {object} models.ThreadPresenceResponse "Active viewers response"
// @Failure 400 {object} fiber.Map "Invalid thread ID"
// @Failure 404 {object} fiber.Map "Thread not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /threads/{id}/presence [get]
func (h *ThreadHandler) GetActiveViewers(c *fiber.Ctx) error {
	threadID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid thread id",
		})
	}

	response, err := h.threadService.GetActiveViewers(c.Context(), threadID)
	if err != nil {
		if err == services.ErrThreadNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "thread not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get active viewers",
		})
	}

	return c.JSON(response)
}

// HeartbeatPresence updates the user's presence timestamp
// @Summary Update presence heartbeat
// @Description Updates the user's presence timestamp to indicate they are still viewing the thread
// @Tags Threads
// @Param id path string true "Thread ID"
// @Success 204 "Presence heartbeat updated"
// @Failure 400 {object} fiber.Map "Invalid thread ID"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /threads/{id}/presence [patch]
func (h *ThreadHandler) HeartbeatPresence(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	threadID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid thread id",
		})
	}

	if err := h.threadService.HeartbeatPresence(c.Context(), threadID, userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to update presence",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
