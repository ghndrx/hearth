package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
	"hearth/internal/websocket"
)

// ForumTagsHandler handles forum tag-related HTTP requests
type ForumTagsHandler struct {
	forumTagService *services.ForumTagService
	threadService   *services.ThreadService
	gateway         *websocket.Gateway
}

// NewForumTagsHandler creates a new forum tags handler
func NewForumTagsHandler(
	forumTagService *services.ForumTagService,
	threadService *services.ThreadService,
	gateway *websocket.Gateway,
) *ForumTagsHandler {
	return &ForumTagsHandler{
		forumTagService: forumTagService,
		threadService:   threadService,
		gateway:         gateway,
	}
}

// ListTags returns all tags for a forum channel
// @Summary List forum channel tags
// @Description Returns all tags for a forum channel
// @Tags ForumTags
// @Produce json
// @Param channelId path string true "Channel ID"
// @Success 200 {array} models.ForumTag
// @Router /channels/{channelId}/tags [get]
func (h *ForumTagsHandler) ListTags(c *fiber.Ctx) error {
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
func (h *ForumTagsHandler) CreateTag(c *fiber.Ctx) error {
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
			Op:       websocket.OpDispatch,
			Type:     websocket.EventForumTagCreate,
			Data:     tag,
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
// @Param tagId path string true "Tag ID"
// @Param tag body models.UpdateForumTagRequest true "Tag data"
// @Success 200 {object} models.ForumTag
// @Router /forum-tags/{tagId} [patch]
func (h *ForumTagsHandler) UpdateTag(c *fiber.Ctx) error {
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
			Op:       websocket.OpDispatch,
			Type:     websocket.EventForumTagUpdate,
			Data:     tag,
			ChannelID: &tag.ChannelID,
		})
	}

	return c.JSON(tag)
}

// DeleteTag deletes a forum tag
// @Summary Delete forum tag
// @Description Deletes a forum tag (requires MANAGE_CHANNELS permission)
// @Tags ForumTags
// @Param tagId path string true "Tag ID"
// @Success 204
// @Router /forum-tags/{tagId} [delete]
func (h *ForumTagsHandler) DeleteTag(c *fiber.Ctx) error {
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
			Op:       websocket.OpDispatch,
			Type:     websocket.EventForumTagDelete,
			Data:     map[string]interface{}{"id": tagID, "channel_id": channelID},
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
func (h *ForumTagsHandler) ApplyTags(c *fiber.Ctx) error {
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
				Op:       websocket.OpDispatch,
				Type:     websocket.EventForumPostUpdate,
				Data:     eventData,
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
func (h *ForumTagsHandler) GetThreadTags(c *fiber.Ctx) error {
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
// @Param sort query int false "Sort order: 0=latest_activity, 1=creation_date, 2=pin_weight"
// @Param limit query int false "Limit (default 25, max 50)"
// @Param offset query int false "Offset for pagination"
// @Success 200
// @Router /channels/{channelId}/posts [get]
func (h *ForumTagsHandler) ListPosts(c *fiber.Ctx) error {
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
func (h *ForumTagsHandler) PinThread(c *fiber.Ctx) error {
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
