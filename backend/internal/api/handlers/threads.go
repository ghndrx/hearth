package handlers

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
	"hearth/internal/websocket"
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
	GetChannelThreadsPaginated(ctx context.Context, channelID uuid.UUID, requesterID uuid.UUID, sortOrder int, limit, offset int, includeArchived bool) ([]models.Thread, int, error)
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
	threadService      ThreadServiceInterface
	autoArchiveService ThreadAutoArchiveServiceInterface
	forumTagService    *services.ForumTagService
	gateway            *websocket.Gateway
}

// NewThreadHandler creates a new thread handler
func NewThreadHandler(threadService ThreadServiceInterface) *ThreadHandler {
	return &ThreadHandler{
		threadService: threadService,
	}
}

// SetAutoArchiveService sets the auto-archive service
func (h *ThreadHandler) SetAutoArchiveService(service ThreadAutoArchiveServiceInterface) {
	h.autoArchiveService = service
}

// SetForumTagService sets the forum tag service and gateway
func (h *ThreadHandler) SetForumTagService(forumTagService *services.ForumTagService, gateway *websocket.Gateway) {
	h.forumTagService = forumTagService
	h.gateway = gateway
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
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
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
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
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
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
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
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
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
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
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

// GetChannelThreads retrieves all threads in a channel with pagination
// @Summary Get channel threads
// @Description Retrieves threads in the specified channel with pagination, sorted by last_message_at by default
// @Tags Threads
// @Produce json
// @Param id path string true "Channel ID"
// @Param include_archived query bool false "Include archived threads in results"
// @Param sort query int false "Sort order: 0=latest_activity, 1=creation_date, 2=pin_weight (default 0)"
// @Param limit query int false "Limit (default 25, max 50)"
// @Param offset query int false "Offset for pagination"
// @Success 200 {array} models.Thread "List of threads"
// @Failure 400 {object} fiber.Map "Invalid channel ID"
// @Failure 403 {object} fiber.Map "Not a server member"
// @Failure 404 {object} fiber.Map "Channel not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{id}/threads [get]
func (h *ThreadHandler) GetChannelThreads(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
	channelID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	includeArchived := c.QueryBool("include_archived", false)
	sortOrder := c.QueryInt("sort", 0)
	limit := c.QueryInt("limit", 25)
	offset := c.QueryInt("offset", 0)

	if limit <= 0 || limit > 50 {
		limit = 25
	}

	threads, total, err := h.threadService.GetChannelThreadsPaginated(c.Context(), channelID, userID, sortOrder, limit, offset, includeArchived)
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

	return c.JSON(fiber.Map{
		"threads":  threads,
		"total":    total,
		"has_more": offset+len(threads) < total,
	})
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
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
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
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
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
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
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
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
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
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
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
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
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
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
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
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
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
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
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

// ThreadAutoArchiveServiceInterface defines thread auto-archive service operations
type ThreadAutoArchiveServiceInterface interface {
	GetOrCreateServerSettings(ctx context.Context, serverID uuid.UUID) (*models.ThreadAutoArchiveSettings, error)
	UpdateServerSettings(ctx context.Context, serverID, requesterID uuid.UUID, req models.UpdateThreadAutoArchiveSettingsRequest) (*models.ThreadAutoArchiveSettings, error)
	GetServerSettings(ctx context.Context, serverID uuid.UUID) (*models.ThreadAutoArchiveSettings, error)
	SetChannelOverride(ctx context.Context, channelID, requesterID uuid.UUID, req models.SetChannelAutoArchiveOverrideRequest) (*models.ChannelAutoArchiveOverride, error)
	GetChannelOverride(ctx context.Context, channelID uuid.UUID) (*models.ChannelAutoArchiveOverride, error)
	DeleteChannelOverride(ctx context.Context, channelID, requesterID uuid.UUID) error
	GetThreadAutoArchiveStatus(ctx context.Context, threadID uuid.UUID) (*models.ThreadAutoArchiveResponse, error)
	GetServerStats(ctx context.Context, serverID uuid.UUID) (*models.ThreadAutoArchiveStats, error)
	ArchiveThread(ctx context.Context, threadID uuid.UUID) error
}

// GetServerAutoArchiveSettings retrieves server auto-archive settings
// @Summary Get server auto-archive settings
// @Description Gets the auto-archive settings for a server
// @Tags Thread Auto-Archive
// @Produce json
// @Param server_id path string true "Server ID"
// @Success 200 {object} models.ThreadAutoArchiveSettings "Auto-archive settings"
// @Failure 400 {object} fiber.Map "Invalid server ID"
// @Failure 403 {object} fiber.Map "Not a server member"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{server_id}/auto-archive [get]
func (h *ThreadHandler) GetServerAutoArchiveSettings(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
	serverID, err := uuid.Parse(c.Params("server_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	// Verify server membership (handled via middleware, but double-check)
	_ = userID // Used by middleware

	settings, err := h.autoArchiveService.GetServerSettings(c.Context(), serverID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get auto-archive settings",
		})
	}

	return c.JSON(settings)
}

// UpdateServerAutoArchiveSettings updates server auto-archive settings
// @Summary Update server auto-archive settings
// @Description Updates the auto-archive settings for a server (admin only)
// @Tags Thread Auto-Archive
// @Accept json
// @Produce json
// @Param server_id path string true "Server ID"
// @Param body body models.UpdateThreadAutoArchiveSettingsRequest true "Settings update request"
// @Success 200 {object} models.ThreadAutoArchiveSettings "Updated auto-archive settings"
// @Failure 400 {object} fiber.Map "Invalid server ID or request body"
// @Failure 403 {object} fiber.Map "Not authorized to update settings"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{server_id}/auto-archive [patch]
func (h *ThreadHandler) UpdateServerAutoArchiveSettings(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
	serverID, err := uuid.Parse(c.Params("server_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	var req models.UpdateThreadAutoArchiveSettingsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	settings, err := h.autoArchiveService.UpdateServerSettings(c.Context(), serverID, userID, req)
	if err != nil {
		switch err {
		case services.ErrServerNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "server not found",
			})
		case services.ErrNotServerMember:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "not a server member",
			})
		case services.ErrMissingAdministrator:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "not authorized to update settings",
			})
		case services.ErrInvalidAutoArchiveDuration:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid auto-archive duration",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to update auto-archive settings",
			})
		}
	}

	return c.JSON(settings)
}

// GetChannelAutoArchiveOverride retrieves channel-level auto-archive override
// @Summary Get channel auto-archive override
// @Description Gets the auto-archive override for a specific channel
// @Tags Thread Auto-Archive
// @Produce json
// @Param channel_id path string true "Channel ID"
// @Success 200 {object} models.ChannelAutoArchiveOverride "Channel override"
// @Failure 400 {object} fiber.Map "Invalid channel ID"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{channel_id}/auto-archive [get]
func (h *ThreadHandler) GetChannelAutoArchiveOverride(c *fiber.Ctx) error {
	channelID, err := uuid.Parse(c.Params("channel_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	override, err := h.autoArchiveService.GetChannelOverride(c.Context(), channelID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get channel override",
		})
	}

	if override == nil {
		return c.JSON(fiber.Map{
			"channel_id": channelID,
			"override":   nil,
		})
	}

	return c.JSON(override)
}

// SetChannelAutoArchiveOverride sets channel-level auto-archive override
// @Summary Set channel auto-archive override
// @Description Sets or updates the auto-archive override for a specific channel (admin only)
// @Tags Thread Auto-Archive
// @Accept json
// @Produce json
// @Param channel_id path string true "Channel ID"
// @Param body body models.SetChannelAutoArchiveOverrideRequest true "Override request"
// @Success 200 {object} models.ChannelAutoArchiveOverride "Channel override"
// @Failure 400 {object} fiber.Map "Invalid channel ID or request body"
// @Failure 403 {object} fiber.Map "Not authorized"
// @Failure 404 {object} fiber.Map "Channel not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{channel_id}/auto-archive [put]
func (h *ThreadHandler) SetChannelAutoArchiveOverride(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
	channelID, err := uuid.Parse(c.Params("channel_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	var req models.SetChannelAutoArchiveOverrideRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	override, err := h.autoArchiveService.SetChannelOverride(c.Context(), channelID, userID, req)
	if err != nil {
		switch err {
		case services.ErrChannelNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "channel not found",
			})
		case services.ErrChannelTypeNotSupported:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "channel is not a server channel",
			})
		case services.ErrAutoArchiveNotAllowed:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "auto-archive override not allowed for this server",
			})
		case services.ErrMissingAdministrator:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "not authorized to set override",
			})
		case services.ErrInvalidAutoArchiveDuration:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid auto-archive duration",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to set channel override",
			})
		}
	}

	return c.JSON(override)
}

// DeleteChannelAutoArchiveOverride removes channel-level auto-archive override
// @Summary Delete channel auto-archive override
// @Description Removes the auto-archive override for a specific channel (admin only)
// @Tags Thread Auto-Archive
// @Param channel_id path string true "Channel ID"
// @Success 204 "Override deleted successfully"
// @Failure 400 {object} fiber.Map "Invalid channel ID"
// @Failure 403 {object} fiber.Map "Not authorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{channel_id}/auto-archive [delete]
func (h *ThreadHandler) DeleteChannelAutoArchiveOverride(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
	channelID, err := uuid.Parse(c.Params("channel_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	if err := h.autoArchiveService.DeleteChannelOverride(c.Context(), channelID, userID); err != nil {
		switch err {
		case services.ErrChannelNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "channel not found",
			})
		case services.ErrChannelTypeNotSupported:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "channel is not a server channel",
			})
		case services.ErrMissingAdministrator:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "not a server member",
			})
		case services.ErrMissingAdministrator:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "not authorized to delete override",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to delete channel override",
			})
		}
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// GetThreadAutoArchiveStatus retrieves auto-archive status for a thread
// @Summary Get thread auto-archive status
// @Description Gets the current auto-archive status for a thread
// @Tags Thread Auto-Archive
// @Produce json
// @Param thread_id path string true "Thread ID"
// @Success 200 {object} models.ThreadAutoArchiveResponse "Thread auto-archive status"
// @Failure 400 {object} fiber.Map "Invalid thread ID"
// @Failure 404 {object} fiber.Map "Thread not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /threads/{thread_id}/auto-archive [get]
func (h *ThreadHandler) GetThreadAutoArchiveStatus(c *fiber.Ctx) error {
	threadID, err := uuid.Parse(c.Params("thread_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid thread id",
		})
	}

	status, err := h.autoArchiveService.GetThreadAutoArchiveStatus(c.Context(), threadID)
	if err != nil {
		if err == services.ErrThreadNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "thread not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get thread auto-archive status",
		})
	}

	return c.JSON(status)
}

// GetServerAutoArchiveStats retrieves auto-archive statistics for a server
// @Summary Get server auto-archive statistics
// @Description Gets statistics about thread auto-archive for a server
// @Tags Thread Auto-Archive
// @Produce json
// @Param server_id path string true "Server ID"
// @Success 200 {object} models.ThreadAutoArchiveStats "Server auto-archive statistics"
// @Failure 400 {object} fiber.Map "Invalid server ID"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{server_id}/auto-archive/stats [get]
func (h *ThreadHandler) GetServerAutoArchiveStats(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("server_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	stats, err := h.autoArchiveService.GetServerStats(c.Context(), serverID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get auto-archive stats",
		})
	}

	return c.JSON(stats)
}

// ListTags returns all tags for a forum channel
// @Summary List forum channel tags
// @Description Returns all tags for a forum channel
// @Tags ForumTags
// @Produce json
// @Param channelId path string true "Channel ID"
// @Success 200 {array} models.ForumTag
// @Router /channels/{channelId}/tags [get]
func (h *ThreadHandler) ListTags(c *fiber.Ctx) error {
	channelID, err := uuid.Parse(c.Params("channelId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	tags, err := h.forumTagService.GetChannelTags(c.Context(), channelID)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(fiber.Map{
		"tags": tags,
	})
}

// CreateTag creates a new tag in a forum channel
// @Summary Create forum tag
// @Description Creates a new tag in a forum channel (requires MANAGE_CHANNELS permission)
// @Tags ForumTags
// @Accept json
// @Produce json
// @Param channelId path string true "Channel ID"
// @Param tag body models.CreateForumTagRequest true "Tag data"
// @Success 201 {object} models.ForumTag
// @Router /channels/{channelId}/tags [post]
func (h *ThreadHandler) CreateTag(c *fiber.Ctx) error {
	channelID, err := uuid.Parse(c.Params("channelId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	var req models.CreateForumTagRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	tag, err := h.forumTagService.CreateTag(c.Context(), channelID, userID, &req)
	if err != nil {
		return HandleServiceError(c, err)
	}

	// Dispatch WebSocket event
	if h.gateway != nil {
		h.gateway.Hub().SendToChannel(channelID, &websocket.Event{
			Op:        websocket.OpDispatch,
			Type:      websocket.EventForumTagCreate,
			Data:      tag,
			ChannelID: &channelID,
		})
	}

	return c.Status(fiber.StatusCreated).JSON(tag)
}

// UpdateTag updates a forum tag
// @Summary Update forum tag
// @Description Updates a forum tag (requires MANAGE_CHANNELS permission)
// @Tags ForumTags
// @Accept json
// @Produce json
// @Param channelId path string false "Channel ID (alternative to tagId)"
// @Param tagId path string true "Tag ID"
// @Param tag body models.UpdateForumTagRequest true "Tag data"
// @Success 200 {object} models.ForumTag
// @Router /forum-tags/{tagId} [patch]
// @Router /channels/{channelId}/tags/{tagId} [patch]
func (h *ThreadHandler) UpdateTag(c *fiber.Ctx) error {
	tagID, err := uuid.Parse(c.Params("tagId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid tag id",
		})
	}

	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	var req models.UpdateForumTagRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	tag, err := h.forumTagService.UpdateTag(c.Context(), tagID, userID, &req)
	if err != nil {
		return HandleServiceError(c, err)
	}

	// Dispatch WebSocket event
	if h.gateway != nil {
		h.gateway.Hub().SendToChannel(tag.ChannelID, &websocket.Event{
			Op:        websocket.OpDispatch,
			Type:      websocket.EventForumTagUpdate,
			Data:      tag,
			ChannelID: &tag.ChannelID,
		})
	}

	return c.JSON(tag)
}

// DeleteTag deletes a forum tag
// @Summary Delete forum tag
// @Description Deletes a forum tag (requires MANAGE_CHANNELS permission)
// @Tags ForumTags
// @Param channelId path string false "Channel ID (alternative to tagId)"
// @Param tagId path string true "Tag ID"
// @Success 204
// @Router /forum-tags/{tagId} [delete]
// @Router /channels/{channelId}/tags/{tagId} [delete]
func (h *ThreadHandler) DeleteTag(c *fiber.Ctx) error {
	tagID, err := uuid.Parse(c.Params("tagId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid tag id",
		})
	}

	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	// Get tag info before deleting to know the channel ID for the event
	tag, _ := h.forumTagService.GetChannelTags(c.Context(), tagID)
	var channelID uuid.UUID
	for _, t := range tag {
		if t.ID == tagID {
			channelID = t.ChannelID
			break
		}
	}

	if err := h.forumTagService.DeleteTag(c.Context(), tagID, userID); err != nil {
		return HandleServiceError(c, err)
	}

	// Dispatch WebSocket event
	if h.gateway != nil && channelID != uuid.Nil {
		h.gateway.Hub().SendToChannel(channelID, &websocket.Event{
			Op:        websocket.OpDispatch,
			Type:      websocket.EventForumTagDelete,
			Data:      map[string]interface{}{"id": tagID, "channel_id": channelID},
			ChannelID: &channelID,
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ApplyTags applies tags to a forum post
// @Summary Apply tags to forum post
// @Description Applies tags to a forum post (thread)
// @Tags ForumTags
// @Accept json
// @Produce json
// @Param threadId path string true "Thread ID"
// @Param tags body fiber.Map true "Tag IDs"
// @Success 200
// @Router /threads/{threadId}/tags [put]
func (h *ThreadHandler) ApplyTags(c *fiber.Ctx) error {
	threadID, err := uuid.Parse(c.Params("threadId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid thread id",
		})
	}

	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	var body struct {
		TagIDs []uuid.UUID `json:"tag_ids"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if err := h.forumTagService.ApplyTagsToThread(c.Context(), threadID, userID, body.TagIDs); err != nil {
		return HandleServiceError(c, err)
	}

	// Dispatch WebSocket event for tag update on the thread
	if h.gateway != nil {
		// Get thread's channel ID to broadcast to the right channel
		thread, _ := h.threadService.GetThread(c.Context(), threadID)
		if thread != nil {
			eventData := map[string]interface{}{
				"thread_id": threadID,
				"tag_ids":   body.TagIDs,
			}
			h.gateway.Hub().SendToChannel(thread.ParentChannelID, &websocket.Event{
				Op:        websocket.OpDispatch,
				Type:      websocket.EventForumPostUpdate,
				Data:      eventData,
				ChannelID: &thread.ParentChannelID,
			})
		}
	}

	return c.JSON(fiber.Map{"success": true})
}

// GetThreadTags returns tags applied to a forum post
// @Summary Get forum post tags
// @Description Returns tags applied to a forum post
// @Tags ForumTags
// @Produce json
// @Param threadId path string true "Thread ID"
// @Success 200 {array} models.ForumTag
// @Router /threads/{threadId}/tags [get]
func (h *ThreadHandler) GetThreadTags(c *fiber.Ctx) error {
	threadID, err := uuid.Parse(c.Params("threadId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid thread id",
		})
	}

	tags, err := h.forumTagService.GetThreadTags(c.Context(), threadID)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(fiber.Map{
		"tags": tags,
	})
}

// ListPosts returns forum posts with optional tag filtering
// @Summary List forum posts
// @Description Returns forum posts (threads) for a channel with optional tag filtering
// @Tags ForumTags
// @Produce json
// @Param channelId path string true "Channel ID"
// @Param tag_ids query string false "Comma-separated tag IDs to filter by"
// @Param sort query int false "Sort order: 0=latest_activity, 1=creation_date, 2=pin_weight, 3=most_reactions, 4=solved_first"
// @Param author_id query string false "Filter by author ID"
// @Param pinned_only query bool false "Show only pinned posts"
// @Param search query string false "Search in post titles"
// @Param limit query int false "Limit (default 25, max 50)"
// @Param offset query int false "Offset for pagination"
// @Success 200
// @Router /channels/{channelId}/posts [get]
func (h *ThreadHandler) ListPosts(c *fiber.Ctx) error {
	channelID, err := uuid.Parse(c.Params("channelId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	filter := &models.ForumPostFilter{
		SortOrder: c.QueryInt("sort", 0),
	}

	// Parse tag_ids query param
	if tagIDsStr := c.Query("tag_ids"); tagIDsStr != "" {
		tagIDStrs := splitAndTrim(tagIDsStr, ",")
		for _, idStr := range tagIDStrs {
			if id, err := uuid.Parse(idStr); err == nil {
				filter.TagIDs = append(filter.TagIDs, id)
			}
		}
	}

	// Parse author_id
	if authorIDStr := c.Query("author_id"); authorIDStr != "" {
		if authorID, err := uuid.Parse(authorIDStr); err == nil {
			filter.AuthorID = &authorID
		}
	}

	// Parse pinned_only
	filter.PinnedOnly = c.QueryBool("pinned_only", false)

	// Parse search query
	filter.SearchQuery = c.Query("search", "")

	limit := c.QueryInt("limit", 25)
	if limit <= 0 || limit > 50 {
		limit = 25
	}
	offset := c.QueryInt("offset", 0)

	threads, tags, total, err := h.forumTagService.FilterForumPosts(c.Context(), channelID, filter, limit, offset)
	if err != nil {
		return HandleServiceError(c, err)
	}

	// Build tag lookup map
	tagMap := make(map[uuid.UUID][]models.ForumTag)
	for _, tag := range tags {
		tagMap[tag.ID] = append(tagMap[tag.ID], tag)
	}

	// Attach tags to threads
	type ThreadWithTags struct {
		models.Thread
		Tags []models.ForumTag `json:"tags"`
	}
	threadsWithTags := make([]ThreadWithTags, len(threads))
	for i, t := range threads {
		var threadTags []models.ForumTag
		for _, tagID := range t.AppliedTags {
			if ts, ok := tagMap[tagID]; ok {
				threadTags = append(threadTags, ts...)
			}
		}
		threadsWithTags[i] = ThreadWithTags{Thread: t, Tags: threadTags}
	}

	return c.JSON(fiber.Map{
		"threads":  threadsWithTags,
		"total":    total,
		"has_more": offset+len(threads) < total,
	})
}

// PinThread pins or unpins a forum post
// @Summary Pin/unpin forum post
// @Description Pins or unpins a forum post (requires MANAGE_CHANNELS permission)
// @Tags ForumTags
// @Accept json
// @Produce json
// @Param threadId path string true "Thread ID"
// @Param pin body fiber.Map true "Pin state"
// @Success 200
// @Router /threads/{threadId}/pin [put]
func (h *ThreadHandler) PinThread(c *fiber.Ctx) error {
	threadID, err := uuid.Parse(c.Params("threadId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid thread id",
		})
	}

	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	var body struct {
		Pin bool `json:"pin"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if err := h.forumTagService.PinThread(c.Context(), threadID, userID, body.Pin); err != nil {
		return HandleServiceError(c, err)
	}

	// Dispatch WebSocket event
	if h.gateway != nil {
		eventType := websocket.EventForumPostPin
		if !body.Pin {
			eventType = websocket.EventForumPostUnpin
		}
		h.gateway.Hub().SendToChannel(threadID, &websocket.Event{
			Op:   websocket.OpDispatch,
			Type: eventType,
			Data: map[string]interface{}{"thread_id": threadID, "pin": body.Pin},
		})
	}

	return c.JSON(fiber.Map{"success": true})
}

// MarkSolved marks or unmarks a forum post as solved
// @Summary Mark/unmark forum post as solved
// @Description Marks or unmarks a forum post as solved/answered
// @Tags ForumTags
// @Accept json
// @Produce json
// @Param threadId path string true "Thread ID"
// @Param body body fiber.Map true "Solved state and optional message ID"
// @Success 200
// @Router /threads/{threadId}/solved [put]
func (h *ThreadHandler) MarkSolved(c *fiber.Ctx) error {
	threadID, err := uuid.Parse(c.Params("threadId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid thread id",
		})
	}

	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	var body struct {
		Solved          bool    `json:"solved"`
		SolvedMessageID *string `json:"solved_message_id,omitempty"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	var solvedMsgID *uuid.UUID
	if body.SolvedMessageID != nil && *body.SolvedMessageID != "" {
		id, err := uuid.Parse(*body.SolvedMessageID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid solved_message_id",
			})
		}
		solvedMsgID = &id
	}

	if err := h.forumTagService.MarkThreadSolved(c.Context(), threadID, userID, body.Solved, solvedMsgID); err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(fiber.Map{"success": true})
}

// GetForumConfig returns the forum configuration for a channel
// @Summary Get forum channel config
// @Description Returns the forum configuration for a channel
// @Tags ForumTags
// @Produce json
// @Param channelId path string true "Channel ID"
// @Success 200 {object} models.ForumConfig
// @Router /channels/{channelId}/forum-config [get]
func (h *ThreadHandler) GetForumConfig(c *fiber.Ctx) error {
	channelID, err := uuid.Parse(c.Params("channelId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	config, err := h.forumTagService.GetForumChannelConfig(c.Context(), channelID)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(config)
}

// UpdateForumConfig updates the forum configuration for a channel
// @Summary Update forum channel config
// @Description Updates the forum configuration for a channel (requires MANAGE_CHANNELS permission)
// @Tags ForumTags
// @Accept json
// @Produce json
// @Param channelId path string true "Channel ID"
// @Param body body models.ForumConfig true "Forum config"
// @Success 200 {object} models.ForumConfig
// @Router /channels/{channelId}/forum-config [patch]
func (h *ThreadHandler) UpdateForumConfig(c *fiber.Ctx) error {
	channelID, err := uuid.Parse(c.Params("channelId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	var config models.ForumConfig
	if err := c.BodyParser(&config); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if err := h.forumTagService.UpdateForumChannelConfig(c.Context(), channelID, userID, &config); err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(config)
}

// splitAndTrim is a helper to parse comma-separated UUIDs
func splitAndTrim(s, sep string) []string {
	if s == "" {
		return nil
	}
	result := make([]string, 0)
	for _, part := range splitString(s, sep) {
		trimmed := trimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func splitString(s, sep string) []string {
	result := make([]string, 0)
	start := 0
	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
			i = start - 1
		}
	}
	result = append(result, s[start:])
	return result
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
