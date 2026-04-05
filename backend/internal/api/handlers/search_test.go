package handlers

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"hearth/internal/models"
	"hearth/internal/services"
)

type mockSearchService struct {
	searchMessagesFunc      func(ctx context.Context, opts services.SearchMessageOptions) (*services.SearchResult, error)
	searchUsersFunc         func(ctx context.Context, query string, serverID *uuid.UUID, requesterID uuid.UUID, limit int) ([]*models.PublicUser, error)
	searchChannelsFunc      func(ctx context.Context, query string, serverID *uuid.UUID, requesterID uuid.UUID, limit int) ([]*models.Channel, error)
	getSuggestionsFunc      func(ctx context.Context, req services.SearchSuggestionsRequest) (*services.SearchSuggestionsResult, error)
	globalSearchMessagesFunc func(ctx context.Context, opts services.GlobalSearchMessageOptions) (*services.GlobalSearchResult, error)
}

func setupSearchTestApp(mock *mockSearchService) *fiber.App {
	app := fiber.New()

	app.Use(func(c *fiber.Ctx) error {
		userIDStr := c.Get("X-Test-User-ID")
		if userIDStr != "" {
			userID, err := uuid.Parse(userIDStr)
			if err == nil {
				c.Locals("userID", userID)
			}
		}
		return c.Next()
	})

	app.Get("/search/messages", func(c *fiber.Ctx) error {
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

		if req.Query == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Query parameter 'q' is required",
			})
		}

		opts := services.SearchMessageOptions{
			Query:       req.Query,
			RequesterID: userID,
			Limit:       req.Limit,
			Offset:      req.Offset,
		}

		if req.ServerID != "" {
			serverID, err := uuid.Parse(req.ServerID)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "Invalid guild_id",
				})
			}
			opts.ServerID = &serverID
		}

		if req.ChannelID != "" {
			channelID, err := uuid.Parse(req.ChannelID)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "Invalid channel_id",
				})
			}
			opts.ChannelID = &channelID
		}

		if req.AuthorID != "" {
			authorID, err := uuid.Parse(req.AuthorID)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "Invalid author_id",
				})
			}
			opts.AuthorID = &authorID
		}

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

		for _, mentionStr := range req.Mentions {
			mentionID, err := uuid.Parse(mentionStr)
			if err != nil {
				continue
			}
			opts.Mentions = append(opts.Mentions, mentionID)
		}

		result, err := mock.searchMessagesFunc(c.Context(), opts)
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

		messages := make([]*MessageSearchResult, 0, len(result.Messages))
		for _, msg := range result.Messages {
			searchResult := &MessageSearchResult{
				ID:        msg.ID.String(),
				ChannelID: msg.ChannelID.String(),
				Author:    msg.Author,
				Content:   msg.Content,
				Timestamp: msg.CreatedAt,
				Pinned:    msg.Pinned,
			}
			messages = append(messages, searchResult)
		}

		return c.JSON(SearchMessagesResponse{
			Messages:   messages,
			TotalCount: result.Total,
			HasMore:    result.HasMore,
		})
	})

	app.Get("/search/users", func(c *fiber.Ctx) error {
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

		limit := 25
		if limitStr := c.Query("limit"); limitStr != "" {
			if parsed := c.QueryInt("limit"); parsed > 0 && parsed <= 100 {
				limit = parsed
			}
		}

		users, err := mock.searchUsersFunc(c.Context(), query, serverID, userID, limit)
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
	})

	app.Get("/search/channels", func(c *fiber.Ctx) error {
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

		limit := 25
		if limitStr := c.Query("limit"); limitStr != "" {
			if parsed := c.QueryInt("limit"); parsed > 0 && parsed <= 100 {
				limit = parsed
			}
		}

		channels, err := mock.searchChannelsFunc(c.Context(), query, serverID, userID, limit)
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
	})

	app.Get("/search/suggestions", func(c *fiber.Ctx) error {
		userID, ok := c.Locals("userID").(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized",
			})
		}

		query := c.Query("q")

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

		limit := 5
		if limitStr := c.Query("limit"); limitStr != "" {
			if parsed := c.QueryInt("limit"); parsed > 0 && parsed <= 20 {
				limit = parsed
			}
		}

		result, err := mock.getSuggestionsFunc(c.Context(), services.SearchSuggestionsRequest{
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
	})

	app.Get("/search", func(c *fiber.Ctx) error {
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

		response := fiber.Map{}

		msgOpts := services.SearchMessageOptions{
			Query:       query,
			RequesterID: userID,
			ServerID:    serverID,
			Limit:       10,
		}
		msgResult, err := mock.searchMessagesFunc(c.Context(), msgOpts)
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
				messages = append(messages, searchResult)
			}
			response["messages"] = fiber.Map{
				"results":     messages,
				"total_count": msgResult.Total,
				"has_more":    msgResult.HasMore,
			}
		}

		users, err := mock.searchUsersFunc(c.Context(), query, serverID, userID, 5)
		if err == nil && len(users) > 0 {
			response["users"] = users
		}

		channels, err := mock.searchChannelsFunc(c.Context(), query, serverID, userID, 5)
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
					sIDStr := ch.ServerID.String()
					result.ServerID = &sIDStr
				}
				results = append(results, result)
			}
			response["channels"] = results
		}

		return c.JSON(response)
	})

	app.Get("/search/global/messages", func(c *fiber.Ctx) error {
		userID, ok := c.Locals("userID").(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized",
			})
		}

		var req GlobalSearchMessagesRequest
		if err := c.QueryParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid query parameters",
			})
		}

		if req.Query == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Query parameter 'q' is required",
			})
		}

		opts := services.GlobalSearchMessageOptions{
			Query:       req.Query,
			RequesterID: userID,
			Limit:       req.Limit,
			Offset:      req.Offset,
		}

		if req.AuthorID != "" {
			authorID, err := uuid.Parse(req.AuthorID)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "Invalid author_id",
				})
			}
			opts.AuthorID = &authorID
		}

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

		if req.HasAttachments != "" {
			val := req.HasAttachments == "true"
			opts.HasAttachments = &val
		}

		if req.HasEmbeds != "" {
			val := req.HasEmbeds == "true"
			opts.HasEmbeds = &val
		}

		if req.HasLinks != "" {
			val := req.HasLinks == "true"
			opts.HasLinks = &val
		}

		if req.HasReactions != "" {
			val := req.HasReactions == "true"
			opts.HasReactions = &val
		}

		if req.Pinned != "" {
			val := req.Pinned == "true"
			opts.Pinned = &val
		}

		for _, sid := range req.ServerIDs {
			serverID, err := uuid.Parse(sid)
			if err != nil {
				continue
			}
			opts.ServerIDs = append(opts.ServerIDs, serverID)
		}

		if req.IncludeDMs != "" {
			opts.IncludeDMs = req.IncludeDMs == "true"
		}

		result, err := mock.globalSearchMessagesFunc(c.Context(), opts)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Search failed",
			})
		}

		messages := make([]*GlobalMessageSearchResult, 0, len(result.Messages))
		for _, msg := range result.Messages {
			var serverIDStr *string
			if msg.ServerID != nil {
				s := msg.ServerID.String()
				serverIDStr = &s
			}
			searchResult := &GlobalMessageSearchResult{
				ID:          msg.ID.String(),
				ChannelID:   msg.ChannelID.String(),
				ServerID:    serverIDStr,
				ServerName:  msg.ServerName,
				ChannelName: msg.ChannelName,
				IsDM:        msg.IsDM,
				Author:      msg.Author,
				Content:     msg.Content,
				Timestamp:   msg.CreatedAt,
				EditedAt:    msg.EditedAt,
				Attachments: msg.Attachments,
				Embeds:      msg.Embeds,
				Reactions:   msg.Reactions,
				Pinned:      msg.Pinned,
			}
			messages = append(messages, searchResult)
		}

		return c.JSON(GlobalSearchMessagesResponse{
			Messages:   messages,
			TotalCount: result.Total,
			HasMore:    result.HasMore,
		})
	})

	return app
}

func TestSearchMessages_Success(t *testing.T) {
	userID := uuid.New()
	msgID := uuid.New()
	channelID := uuid.New()

	mock := &mockSearchService{
		searchMessagesFunc: func(ctx context.Context, opts services.SearchMessageOptions) (*services.SearchResult, error) {
			return &services.SearchResult{
				Messages: []*models.Message{
					{
						ID:        msgID,
						ChannelID: channelID,
						AuthorID:  uuid.New(),
						Content:   "hello world",
						CreatedAt: time.Now(),
					},
				},
				Total:   1,
				HasMore: false,
			}, nil
		},
	}

	app := setupSearchTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("GET", "/search/messages?q=hello", nil)
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result SearchMessagesResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, 1, result.TotalCount)
	assert.Len(t, result.Messages, 1)
	assert.Equal(t, msgID.String(), result.Messages[0].ID)
}

func TestSearchMessages_MissingQuery(t *testing.T) {
	userID := uuid.New()

	mock := &mockSearchService{}

	app := setupSearchTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("GET", "/search/messages", nil)
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestSearchMessages_InvalidGuildID(t *testing.T) {
	userID := uuid.New()

	mock := &mockSearchService{}

	app := setupSearchTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("GET", "/search/messages?q=hello&guild_id=not-a-uuid", nil)
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestSearchMessages_NotServerMember(t *testing.T) {
	userID := uuid.New()

	mock := &mockSearchService{
		searchMessagesFunc: func(ctx context.Context, opts services.SearchMessageOptions) (*services.SearchResult, error) {
			return nil, services.ErrNotServerMember
		},
	}

	app := setupSearchTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("GET", "/search/messages?q=hello", nil)
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 403, resp.StatusCode)
}

func TestSearchUsers_Success(t *testing.T) {
	userID := uuid.New()

	mock := &mockSearchService{
		searchUsersFunc: func(ctx context.Context, query string, serverID *uuid.UUID, requesterID uuid.UUID, limit int) ([]*models.PublicUser, error) {
			return []*models.PublicUser{
				{
					ID:            uuid.New(),
					Username:      "testuser",
					Discriminator: "0001",
				},
			}, nil
		},
	}

	app := setupSearchTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("GET", "/search/users?q=test", nil)
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result SearchUsersResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Len(t, result.Users, 1)
	assert.Equal(t, "testuser", result.Users[0].Username)
}

func TestSearchUsers_MissingQuery(t *testing.T) {
	userID := uuid.New()

	mock := &mockSearchService{}

	app := setupSearchTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("GET", "/search/users", nil)
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestSearchChannels_Success(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()

	mock := &mockSearchService{
		searchChannelsFunc: func(ctx context.Context, query string, sID *uuid.UUID, requesterID uuid.UUID, limit int) ([]*models.Channel, error) {
			return []*models.Channel{
				{
					ID:       uuid.New(),
					Name:     "general",
					Type:     models.ChannelTypeText,
					Topic:    "General discussion",
					ServerID: &serverID,
				},
			}, nil
		},
	}

	app := setupSearchTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("GET", "/search/channels?q=general", nil)
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result SearchChannelsResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Len(t, result.Channels, 1)
	assert.Equal(t, "general", result.Channels[0].Name)
}

func TestSearchChannels_MissingQuery(t *testing.T) {
	userID := uuid.New()

	mock := &mockSearchService{}

	app := setupSearchTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("GET", "/search/channels", nil)
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestSearchAll_Success(t *testing.T) {
	userID := uuid.New()
	msgID := uuid.New()
	channelID := uuid.New()

	mock := &mockSearchService{
		searchMessagesFunc: func(ctx context.Context, opts services.SearchMessageOptions) (*services.SearchResult, error) {
			return &services.SearchResult{
				Messages: []*models.Message{
					{
						ID:        msgID,
						ChannelID: channelID,
						Content:   "test message",
						CreatedAt: time.Now(),
					},
				},
				Total:   1,
				HasMore: false,
			}, nil
		},
		searchUsersFunc: func(ctx context.Context, query string, serverID *uuid.UUID, requesterID uuid.UUID, limit int) ([]*models.PublicUser, error) {
			return []*models.PublicUser{
				{
					ID:       uuid.New(),
					Username: "testuser",
				},
			}, nil
		},
		searchChannelsFunc: func(ctx context.Context, query string, serverID *uuid.UUID, requesterID uuid.UUID, limit int) ([]*models.Channel, error) {
			return []*models.Channel{
				{
					ID:   uuid.New(),
					Name: "test-channel",
					Type: models.ChannelTypeText,
				},
			}, nil
		},
	}

	app := setupSearchTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("GET", "/search?q=test", nil)
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.NotNil(t, result["messages"])
	assert.NotNil(t, result["users"])
	assert.NotNil(t, result["channels"])
}

func TestSearchAll_MissingQuery(t *testing.T) {
	userID := uuid.New()

	mock := &mockSearchService{}

	app := setupSearchTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("GET", "/search", nil)
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestGetSuggestions_Success(t *testing.T) {
	userID := uuid.New()

	mock := &mockSearchService{
		getSuggestionsFunc: func(ctx context.Context, req services.SearchSuggestionsRequest) (*services.SearchSuggestionsResult, error) {
			return &services.SearchSuggestionsResult{
				Users: []services.SearchSuggestion{
					{Type: "user", Name: "testuser", Value: "from:testuser"},
				},
				Channels: []services.SearchSuggestion{},
				Filters:  []services.SearchSuggestion{},
			}, nil
		},
	}

	app := setupSearchTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("GET", "/search/suggestions?q=test", nil)
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result services.SearchSuggestionsResult
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Len(t, result.Users, 1)
	assert.Equal(t, "testuser", result.Users[0].Name)
}

func TestGlobalSearchMessages_Success(t *testing.T) {
	userID := uuid.New()
	msgID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	mock := &mockSearchService{
		globalSearchMessagesFunc: func(ctx context.Context, opts services.GlobalSearchMessageOptions) (*services.GlobalSearchResult, error) {
			return &services.GlobalSearchResult{
				Messages: []*services.GlobalSearchMessage{
					{
						Message: &models.Message{
							ID:        msgID,
							ChannelID: channelID,
							AuthorID:  uuid.New(),
							Content:   "hello world from another server",
							CreatedAt: time.Now(),
						},
						ServerID:    &serverID,
						ServerName:  "Test Server",
						ChannelName: "general",
						IsDM:        false,
					},
				},
				Total:   1,
				HasMore: false,
			}, nil
		},
	}

	app := setupSearchTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("GET", "/search/global/messages?q=hello", nil)
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result GlobalSearchMessagesResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, 1, result.TotalCount)
	assert.Len(t, result.Messages, 1)
	assert.Equal(t, msgID.String(), result.Messages[0].ID)
	assert.Equal(t, "Test Server", result.Messages[0].ServerName)
	assert.Equal(t, "general", result.Messages[0].ChannelName)
	assert.False(t, result.Messages[0].IsDM)
}

func TestGlobalSearchMessages_WithFilters(t *testing.T) {
	userID := uuid.New()
	msgID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	mock := &mockSearchService{
		globalSearchMessagesFunc: func(ctx context.Context, opts services.GlobalSearchMessageOptions) (*services.GlobalSearchResult, error) {
			// Verify filters are passed correctly
			assert.NotNil(t, opts.AuthorID)
			assert.NotNil(t, opts.HasAttachments)
			assert.True(t, *opts.HasAttachments)
			assert.True(t, opts.IncludeDMs)
			assert.Len(t, opts.ServerIDs, 1)
			assert.Equal(t, serverID, opts.ServerIDs[0])

			return &services.GlobalSearchResult{
				Messages: []*services.GlobalSearchMessage{
					{
						Message: &models.Message{
							ID:          msgID,
							ChannelID:   channelID,
							AuthorID:    uuid.New(),
							Content:     "test message with attachment",
							CreatedAt:   time.Now(),
							Attachments: []models.Attachment{{ID: uuid.New(), Filename: "test.png"}},
						},
						ServerID:    &serverID,
						ServerName:  "Filtered Server",
						ChannelName: "attachments",
						IsDM:        false,
					},
				},
				Total:   1,
				HasMore: true,
			}, nil
		},
	}

	app := setupSearchTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	authorID := uuid.New()
	req := httptest.NewRequest("GET", "/search/global/messages?q=test&author_id="+authorID.String()+"&has_attachments=true&include_dms=true&server_ids="+serverID.String(), nil)
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result GlobalSearchMessagesResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, 1, result.TotalCount)
	assert.True(t, result.HasMore)
	assert.Len(t, result.Messages, 1)
}

func TestGlobalSearchMessages_MissingQuery(t *testing.T) {
	userID := uuid.New()

	mock := &mockSearchService{}

	app := setupSearchTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("GET", "/search/global/messages", nil)
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestGlobalSearchMessages_Unauthorized(t *testing.T) {
	mock := &mockSearchService{}

	app := setupSearchTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("GET", "/search/global/messages?q=test", nil)
	// No X-Test-User-ID header - simulates unauthenticated request

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 401, resp.StatusCode)
}

func TestGlobalSearchMessages_WithDateRange(t *testing.T) {
	userID := uuid.New()
	msgID := uuid.New()
	channelID := uuid.New()

	beforeTime := time.Now().Add(-24 * time.Hour)
	afterTime := time.Now().Add(-48 * time.Hour)

	mock := &mockSearchService{
		globalSearchMessagesFunc: func(ctx context.Context, opts services.GlobalSearchMessageOptions) (*services.GlobalSearchResult, error) {
			// Verify date filters are passed correctly
			assert.NotNil(t, opts.Before)
			assert.NotNil(t, opts.After)
			assert.Equal(t, beforeTime.Unix(), opts.Before.Unix())
			assert.Equal(t, afterTime.Unix(), opts.After.Unix())

			return &services.GlobalSearchResult{
				Messages: []*services.GlobalSearchMessage{
					{
						Message: &models.Message{
							ID:        msgID,
							ChannelID: channelID,
							Content:   "dated message",
							CreatedAt: time.Now().Add(-36 * time.Hour),
						},
						ServerID:    nil,
						ServerName:  "",
						ChannelName: "general",
						IsDM:        true,
					},
				},
				Total:   1,
				HasMore: false,
			}, nil
		},
	}

	app := setupSearchTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	beforeStr := beforeTime.Format(time.RFC3339)
	afterStr := afterTime.Format(time.RFC3339)
	req := httptest.NewRequest("GET", "/search/global/messages?q=dated&before="+beforeStr+"&after="+afterStr, nil)
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result GlobalSearchMessagesResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Len(t, result.Messages, 1)
	assert.True(t, result.Messages[0].IsDM) // DM since ServerID is nil
}

func TestGlobalSearchMessages_WithBooleanFilters(t *testing.T) {
	userID := uuid.New()
	msgID := uuid.New()
	channelID := uuid.New()

	testCases := []struct {
		name            string
		queryParams     string
		expectedHasReactions *bool
		expectedHasEmbeds   *bool
		expectedHasLinks    *bool
		expectedPinned      *bool
	}{
		{
			name:        "has_reactions filter",
			queryParams: "q=test&has_reactions=true",
			expectedHasReactions: func() *bool { v := true; return &v }(),
		},
		{
			name:        "has_embeds filter",
			queryParams: "q=test&has_embeds=true",
			expectedHasEmbeds: func() *bool { v := true; return &v }(),
		},
		{
			name:        "has_links filter",
			queryParams: "q=test&has_links=true",
			expectedHasLinks: func() *bool { v := true; return &v }(),
		},
		{
			name:        "pinned filter",
			queryParams: "q=test&pinned=true",
			expectedPinned: func() *bool { v := true; return &v }(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockSearchService{
				globalSearchMessagesFunc: func(ctx context.Context, opts services.GlobalSearchMessageOptions) (*services.GlobalSearchResult, error) {
					if tc.expectedHasReactions != nil {
						assert.NotNil(t, opts.HasReactions)
						assert.Equal(t, *tc.expectedHasReactions, *opts.HasReactions)
					}
					if tc.expectedHasEmbeds != nil {
						assert.NotNil(t, opts.HasEmbeds)
						assert.Equal(t, *tc.expectedHasEmbeds, *opts.HasEmbeds)
					}
					if tc.expectedHasLinks != nil {
						assert.NotNil(t, opts.HasLinks)
						assert.Equal(t, *tc.expectedHasLinks, *opts.HasLinks)
					}
					if tc.expectedPinned != nil {
						assert.NotNil(t, opts.Pinned)
						assert.Equal(t, *tc.expectedPinned, *opts.Pinned)
					}

					return &services.GlobalSearchResult{
						Messages: []*services.GlobalSearchMessage{
							{
								Message: &models.Message{
									ID:        msgID,
									ChannelID: channelID,
									Content:   "test",
									CreatedAt: time.Now(),
								},
								ServerID:    nil,
								ChannelName: "general",
								IsDM:        false,
							},
						},
						Total:   1,
						HasMore: false,
					}, nil
				},
			}

			app := setupSearchTestApp(mock)
			t.Cleanup(func() { _ = app.Shutdown() })

			req := httptest.NewRequest("GET", "/search/global/messages?"+tc.queryParams, nil)
			req.Header.Set("X-Test-User-ID", userID.String())

			resp, err := app.Test(req, -1)
			assert.NoError(t, err)
			assert.Equal(t, 200, resp.StatusCode)
		})
	}
}

func TestGlobalSearchMessages_Pagination(t *testing.T) {
	userID := uuid.New()
	msgID1 := uuid.New()
	msgID2 := uuid.New()
	channelID := uuid.New()

	mock := &mockSearchService{
		globalSearchMessagesFunc: func(ctx context.Context, opts services.GlobalSearchMessageOptions) (*services.GlobalSearchResult, error) {
			// Verify pagination params
			assert.Equal(t, 10, opts.Limit)
			assert.Equal(t, 25, opts.Offset)

			return &services.GlobalSearchResult{
				Messages: []*services.GlobalSearchMessage{
					{
						Message: &models.Message{
							ID:        msgID1,
							ChannelID: channelID,
							Content:   "page 2 message 1",
							CreatedAt: time.Now(),
						},
						ServerID:    nil,
						ChannelName: "general",
						IsDM:        true,
					},
					{
						Message: &models.Message{
							ID:        msgID2,
							ChannelID: channelID,
							Content:   "page 2 message 2",
							CreatedAt: time.Now(),
						},
						ServerID:    nil,
						ChannelName: "general",
						IsDM:        true,
					},
				},
				Total:   50,
				HasMore: true,
			}, nil
		},
	}

	app := setupSearchTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("GET", "/search/global/messages?q=test&limit=10&offset=25", nil)
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result GlobalSearchMessagesResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, 50, result.TotalCount)
	assert.True(t, result.HasMore)
	assert.Len(t, result.Messages, 2)
}
