package handlers

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// SearchHandler handles search-related HTTP requests
type SearchHandler struct {
	searchService *services.SearchService
}

// NewSearchHandler creates a new search handler
func NewSearchHandler(searchService *services.SearchService) *SearchHandler {
	return &SearchHandler{searchService: searchService}
}

// SearchMessagesRequest represents a message search request
type SearchMessagesRequest struct {
	Query          string   `query:"q"`
	ServerID       string   `query:"guild_id"`
	ChannelID      string   `query:"channel_id"`
	AuthorID       string   `query:"author_id"`
	Before         string   `query:"before"`         // ISO8601 timestamp
	After          string   `query:"after"`          // ISO8601 timestamp
	HasAttachments string   `query:"has_attachments"` // "true" or "false"
	HasEmbeds      string   `query:"has_embeds"`      // "true" or "false"
	HasReactions   string   `query:"has_reactions"`   // "true" or "false"
	Pinned         string   `query:"pinned"`          // "true" or "false"
	Mentions       []string `query:"mentions"`        // User IDs
	Limit          int      `query:"limit"`
	Offset         int      `query:"offset"`
}

// SearchMessagesResponse represents the message search response
type SearchMessagesResponse struct {
	Messages   []*MessageSearchResult `json:"messages"`
	TotalCount int                    `json:"total_count"`
	HasMore    bool                   `json:"has_more"`
}

// MessageSearchResult represents a message in search results
type MessageSearchResult struct {
	ID          string               `json:"id"`
	ChannelID   string               `json:"channel_id"`
	ServerID    *string              `json:"guild_id,omitempty"`
	Author      *models.PublicUser   `json:"author"`
	Content     string               `json:"content"`
	Timestamp   time.Time            `json:"timestamp"`
	EditedAt    *time.Time           `json:"edited_timestamp,omitempty"`
	Attachments []models.Attachment  `json:"attachments,omitempty"`
	Embeds      []models.Embed       `json:"embeds,omitempty"`
	Reactions   []models.Reaction    `json:"reactions,omitempty"`
	Pinned      bool                 `json:"pinned"`
	// Context messages around the hit
	Before      []*models.Message    `json:"before,omitempty"`
	After       []*models.Message    `json:"after,omitempty"`
}

// SearchUsersResponse represents the user search response
type SearchUsersResponse struct {
	Users []*models.PublicUser `json:"users"`
}

// SearchChannelsResponse represents the channel search response
type SearchChannelsResponse struct {
	Channels []*ChannelSearchResult `json:"channels"`
}

// ChannelSearchResult represents a channel in search results
type ChannelSearchResult struct {
	ID       string             `json:"id"`
	ServerID *string            `json:"guild_id,omitempty"`
	Name     string             `json:"name"`
	Type     models.ChannelType `json:"type"`
	Topic    string             `json:"topic,omitempty"`
}

// SearchMessages handles message search requests
// GET /api/search/messages
func (h *SearchHandler) SearchMessages(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	var req SearchMessagesRequest
	if err := c.QueryParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid query parameters",
		})
	}

	// Require a query string
	if req.Query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Query parameter 'q' is required",
		})
	}

	// Build search options
	opts := services.SearchMessageOptions{
		Query:       req.Query,
		RequesterID: userID,
		Limit:       req.Limit,
		Offset:      req.Offset,
	}

	// Parse optional server ID
	if req.ServerID != "" {
		serverID, err := uuid.Parse(req.ServerID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid guild_id",
			})
		}
		opts.ServerID = &serverID
	}

	// Parse optional channel ID
	if req.ChannelID != "" {
		channelID, err := uuid.Parse(req.ChannelID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid channel_id",
			})
		}
		opts.ChannelID = &channelID
	}

	// Parse optional author ID
	if req.AuthorID != "" {
		authorID, err := uuid.Parse(req.AuthorID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid author_id",
			})
		}
		opts.AuthorID = &authorID
	}

	// Parse time filters
	if req.Before != "" {
		before, err := time.Parse(time.RFC3339, req.Before)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid 'before' timestamp (expected ISO8601)",
			})
		}
		opts.Before = &before
	}

	if req.After != "" {
		after, err := time.Parse(time.RFC3339, req.After)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid 'after' timestamp (expected ISO8601)",
			})
		}
		opts.After = &after
	}

	// Parse boolean filters
	if req.HasAttachments != "" {
		val := req.HasAttachments == "true"
		opts.HasAttachments = &val
	}

	if req.HasEmbeds != "" {
		val := req.HasEmbeds == "true"
		opts.HasEmbeds = &val
	}

	if req.HasReactions != "" {
		val := req.HasReactions == "true"
		opts.HasReactions = &val
	}

	if req.Pinned != "" {
		val := req.Pinned == "true"
		opts.Pinned = &val
	}

	// Parse mentions filter
	for _, mentionStr := range req.Mentions {
		mentionID, err := uuid.Parse(mentionStr)
		if err != nil {
			continue // Skip invalid UUIDs
		}
		opts.Mentions = append(opts.Mentions, mentionID)
	}

	// Perform search
	result, err := h.searchService.SearchMessages(c.Context(), opts)
	if err != nil {
		switch err {
		case services.ErrNotServerMember:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "You are not a member of this server",
			})
		case services.ErrChannelNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Channel not found",
			})
		case services.ErrNoPermission:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "You do not have permission to search this channel",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Search failed",
			})
		}
	}

	// Convert to response format
	messages := make([]*MessageSearchResult, 0, len(result.Messages))
	for _, msg := range result.Messages {
		searchResult := &MessageSearchResult{
			ID:          msg.ID.String(),
			ChannelID:   msg.ChannelID.String(),
			Author:      msg.Author,
			Content:     msg.Content,
			Timestamp:   msg.CreatedAt,
			EditedAt:    msg.EditedAt,
			Attachments: msg.Attachments,
			Embeds:      msg.Embeds,
			Reactions:   msg.Reactions,
			Pinned:      msg.Pinned,
		}
		if msg.ServerID != nil {
			serverIDStr := msg.ServerID.String()
			searchResult.ServerID = &serverIDStr
		}
		messages = append(messages, searchResult)
	}

	return c.JSON(SearchMessagesResponse{
		Messages:   messages,
		TotalCount: result.Total,
		HasMore:    result.HasMore,
	})
}

// SearchUsers handles user search requests
// GET /api/search/users
func (h *SearchHandler) SearchUsers(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	query := c.Query("q")
	if query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Query parameter 'q' is required",
		})
	}

	// Parse optional server ID filter
	var serverID *uuid.UUID
	if serverIDStr := c.Query("guild_id"); serverIDStr != "" {
		parsed, err := uuid.Parse(serverIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid guild_id",
			})
		}
		serverID = &parsed
	}

	// Parse limit
	limit := 25
	if limitStr := c.Query("limit"); limitStr != "" {
		parsed, err := strconv.Atoi(limitStr)
		if err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	// Perform search
	users, err := h.searchService.SearchUsers(c.Context(), query, serverID, userID, limit)
	if err != nil {
		switch err {
		case services.ErrNotServerMember:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "You are not a member of this server",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Search failed",
			})
		}
	}

	return c.JSON(SearchUsersResponse{
		Users: users,
	})
}

// SearchChannels handles channel search requests
// GET /api/search/channels
func (h *SearchHandler) SearchChannels(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	query := c.Query("q")
	if query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Query parameter 'q' is required",
		})
	}

	// Parse optional server ID filter
	var serverID *uuid.UUID
	if serverIDStr := c.Query("guild_id"); serverIDStr != "" {
		parsed, err := uuid.Parse(serverIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid guild_id",
			})
		}
		serverID = &parsed
	}

	// Parse limit
	limit := 25
	if limitStr := c.Query("limit"); limitStr != "" {
		parsed, err := strconv.Atoi(limitStr)
		if err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	// Perform search
	channels, err := h.searchService.SearchChannels(c.Context(), query, serverID, userID, limit)
	if err != nil {
		switch err {
		case services.ErrNotServerMember:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "You are not a member of this server",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Search failed",
			})
		}
	}

	// Convert to response format
	results := make([]*ChannelSearchResult, 0, len(channels))
	for _, ch := range channels {
		result := &ChannelSearchResult{
			ID:    ch.ID.String(),
			Name:  ch.Name,
			Type:  ch.Type,
			Topic: ch.Topic,
		}
		if ch.ServerID != nil {
			serverIDStr := ch.ServerID.String()
			result.ServerID = &serverIDStr
		}
		results = append(results, result)
	}

	return c.JSON(SearchChannelsResponse{
		Channels: results,
	})
}

// SearchAll performs a combined search across messages, users, and channels
// GET /api/search
func (h *SearchHandler) SearchAll(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	query := c.Query("q")
	if query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Query parameter 'q' is required",
		})
	}

	// Parse optional server ID filter
	var serverID *uuid.UUID
	if serverIDStr := c.Query("guild_id"); serverIDStr != "" {
		parsed, err := uuid.Parse(serverIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid guild_id",
			})
		}
		serverID = &parsed
	}

	// Combined response
	response := fiber.Map{}

	// Search messages (limit 10 for combined search)
	msgOpts := services.SearchMessageOptions{
		Query:       query,
		RequesterID: userID,
		ServerID:    serverID,
		Limit:       10,
	}
	msgResult, err := h.searchService.SearchMessages(c.Context(), msgOpts)
	if err == nil && len(msgResult.Messages) > 0 {
		messages := make([]*MessageSearchResult, 0, len(msgResult.Messages))
		for _, msg := range msgResult.Messages {
			searchResult := &MessageSearchResult{
				ID:        msg.ID.String(),
				ChannelID: msg.ChannelID.String(),
				Author:    msg.Author,
				Content:   msg.Content,
				Timestamp: msg.CreatedAt,
			}
			if msg.ServerID != nil {
				serverIDStr := msg.ServerID.String()
				searchResult.ServerID = &serverIDStr
			}
			messages = append(messages, searchResult)
		}
		response["messages"] = fiber.Map{
			"results":     messages,
			"total_count": msgResult.Total,
			"has_more":    msgResult.HasMore,
		}
	}

	// Search users (limit 5 for combined search)
	users, err := h.searchService.SearchUsers(c.Context(), query, serverID, userID, 5)
	if err == nil && len(users) > 0 {
		response["users"] = users
	}

	// Search channels (limit 5 for combined search)
	channels, err := h.searchService.SearchChannels(c.Context(), query, serverID, userID, 5)
	if err == nil && len(channels) > 0 {
		results := make([]*ChannelSearchResult, 0, len(channels))
		for _, ch := range channels {
			result := &ChannelSearchResult{
				ID:    ch.ID.String(),
				Name:  ch.Name,
				Type:  ch.Type,
				Topic: ch.Topic,
			}
			if ch.ServerID != nil {
				serverIDStr := ch.ServerID.String()
				result.ServerID = &serverIDStr
			}
			results = append(results, result)
		}
		response["channels"] = results
	}

	return c.JSON(response)
}

// GetSuggestions returns search suggestions for type-ahead
// GET /api/search/suggestions
func (h *SearchHandler) GetSuggestions(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	query := c.Query("q")

	// Parse optional server ID filter
	var serverID *uuid.UUID
	if serverIDStr := c.Query("guild_id"); serverIDStr != "" {
		parsed, err := uuid.Parse(serverIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid guild_id",
			})
		}
		serverID = &parsed
	}

	// Parse limit
	limit := 5
	if limitStr := c.Query("limit"); limitStr != "" {
		parsed, err := strconv.Atoi(limitStr)
		if err == nil && parsed > 0 && parsed <= 20 {
			limit = parsed
		}
	}

	// Get suggestions
	result, err := h.searchService.GetSearchSuggestions(c.Context(), services.SearchSuggestionsRequest{
		Query:    query,
		ServerID: serverID,
		Limit:    limit,
		UserID:   userID,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get suggestions",
		})
	}

	return c.JSON(result)
}
