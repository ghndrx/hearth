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
	searchMessagesFunc func(ctx context.Context, opts services.SearchMessageOptions) (*services.SearchResult, error)
	searchUsersFunc    func(ctx context.Context, query string, serverID *uuid.UUID, requesterID uuid.UUID, limit int) ([]*models.PublicUser, error)
	searchChannelsFunc func(ctx context.Context, query string, serverID *uuid.UUID, requesterID uuid.UUID, limit int) ([]*models.Channel, error)
	getSuggestionsFunc func(ctx context.Context, req services.SearchSuggestionsRequest) (*services.SearchSuggestionsResult, error)
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
