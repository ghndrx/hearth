package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"hearth/internal/ai"
	"hearth/internal/models"
)

// MockChatService is a mock implementation of the chat service
type MockChatService struct {
	mock.Mock
}

func (m *MockChatService) CreateConversation(ctx context.Context, userID uuid.UUID, req *models.CreateConversationRequest) (*models.AIConversation, error) {
	args := m.Called(ctx, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AIConversation), args.Error(1)
}

func (m *MockChatService) GetConversation(ctx context.Context, userID, conversationID uuid.UUID) (*models.AIConversation, error) {
	args := m.Called(ctx, userID, conversationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AIConversation), args.Error(1)
}

func (m *MockChatService) GetConversationWithMessages(ctx context.Context, userID, conversationID uuid.UUID, limit int) (*models.AIConversationWithMessages, error) {
	args := m.Called(ctx, userID, conversationID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AIConversationWithMessages), args.Error(1)
}

func (m *MockChatService) ListConversations(ctx context.Context, params *models.ConversationListParams) ([]*models.AIConversation, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.AIConversation), args.Error(1)
}

func (m *MockChatService) UpdateConversation(ctx context.Context, userID, conversationID uuid.UUID, req *models.UpdateConversationRequest) (*models.AIConversation, error) {
	args := m.Called(ctx, userID, conversationID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AIConversation), args.Error(1)
}

func (m *MockChatService) DeleteConversation(ctx context.Context, userID, conversationID uuid.UUID) error {
	args := m.Called(ctx, userID, conversationID)
	return args.Error(0)
}

func (m *MockChatService) SendMessage(ctx context.Context, userID, conversationID uuid.UUID, req *models.SendChatMessageRequest) (*models.ConversationMessage, error) {
	args := m.Called(ctx, userID, conversationID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ConversationMessage), args.Error(1)
}

func (m *MockChatService) SendMessageStream(ctx context.Context, userID, conversationID uuid.UUID, req *models.SendChatMessageRequest, callback func(*models.StreamChunk) error) (*models.ConversationMessage, error) {
	args := m.Called(ctx, userID, conversationID, req, callback)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ConversationMessage), args.Error(1)
}

func (m *MockChatService) GetConversationMessages(ctx context.Context, userID, conversationID uuid.UUID, limit, offset int) ([]*models.ConversationMessage, error) {
	args := m.Called(ctx, userID, conversationID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ConversationMessage), args.Error(1)
}

func (m *MockChatService) DeleteMessage(ctx context.Context, userID, conversationID, messageID uuid.UUID) error {
	args := m.Called(ctx, userID, conversationID, messageID)
	return args.Error(0)
}

func (m *MockChatService) RegenerateMessage(ctx context.Context, userID, conversationID, messageID uuid.UUID, stream bool, callback func(*models.StreamChunk) error) (*models.ConversationMessage, error) {
	args := m.Called(ctx, userID, conversationID, messageID, stream, callback)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ConversationMessage), args.Error(1)
}

func (m *MockChatService) GetTemplates(ctx context.Context, userID *uuid.UUID, category string) ([]*models.AIChatTemplate, error) {
	args := m.Called(ctx, userID, category)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.AIChatTemplate), args.Error(1)
}

func (m *MockChatService) GetTemplate(ctx context.Context, id uuid.UUID) (*models.AIChatTemplate, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AIChatTemplate), args.Error(1)
}

func (m *MockChatService) CreateTemplate(ctx context.Context, userID uuid.UUID, tmpl *models.AIChatTemplate) error {
	args := m.Called(ctx, userID, tmpl)
	return args.Error(0)
}

func (m *MockChatService) UpdateTemplate(ctx context.Context, userID uuid.UUID, tmpl *models.AIChatTemplate) error {
	args := m.Called(ctx, userID, tmpl)
	return args.Error(0)
}

func (m *MockChatService) DeleteTemplate(ctx context.Context, userID, templateID uuid.UUID) error {
	args := m.Called(ctx, userID, templateID)
	return args.Error(0)
}

func (m *MockChatService) ShareConversation(ctx context.Context, userID, conversationID uuid.UUID, isPublic, canContinue bool, expiresIn *time.Duration) (*models.AIConversationShare, error) {
	args := m.Called(ctx, userID, conversationID, isPublic, canContinue, expiresIn)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AIConversationShare), args.Error(1)
}

func (m *MockChatService) GetSharedConversation(ctx context.Context, shareCode string) (*models.AIConversationWithMessages, error) {
	args := m.Called(ctx, shareCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AIConversationWithMessages), args.Error(1)
}

// Helper function to create a test app with the handler
func setupAIChatTestApp(t testing.TB, handler *AIChatHandler) *fiber.App {
	app := fiber.New()
	t.Cleanup(func() { _ = app.Shutdown() })

	// Add mock authentication middleware
	app.Use(func(c *fiber.Ctx) error {
		userID := c.Get("X-Test-User-ID")
		if userID != "" {
			uid, _ := uuid.Parse(userID)
			c.Locals("userID", uid)
		}
		return c.Next()
	})

	// Setup routes
	ai := app.Group("/api/v1/ai")

	// Conversations
	ai.Get("/conversations", handler.ListConversations)
	ai.Post("/conversations", handler.CreateConversation)
	ai.Get("/conversations/:id", handler.GetConversation)
	ai.Patch("/conversations/:id", handler.UpdateConversation)
	ai.Delete("/conversations/:id", handler.DeleteConversation)

	// Messages
	ai.Get("/conversations/:id/messages", handler.GetMessages)
	ai.Post("/conversations/:id/messages", handler.SendMessage)
	ai.Post("/conversations/:id/messages/:messageId/regenerate", handler.RegenerateMessage)
	ai.Delete("/conversations/:id/messages/:messageId", handler.DeleteMessage)

	// Templates
	ai.Get("/templates", handler.ListTemplates)
	ai.Post("/templates", handler.CreateTemplate)
	ai.Get("/templates/:id", handler.GetTemplate)
	ai.Patch("/templates/:id", handler.UpdateTemplate)
	ai.Delete("/templates/:id", handler.DeleteTemplate)

	// Shares
	ai.Post("/conversations/:id/share", handler.ShareConversation)
	ai.Get("/shared/:code", handler.GetSharedConversation)

	return app
}

// Tests for Conversations

func TestListConversations(t *testing.T) {
	// Use real ChatService for tests that don't need mock verification;
	// use MockChatService for tests that verify specific service calls.
	handler := NewAIChatHandler(&ai.ChatService{})

	t.Run("unauthorized without user ID", func(t *testing.T) {
		app := setupAIChatTestApp(t, handler)
		req := httptest.NewRequest("GET", "/api/v1/ai/conversations", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	})
}

func TestCreateConversation(t *testing.T) {
	userID := uuid.New()

	t.Run("create with defaults", func(t *testing.T) {
		// Test body parsing
		body := models.CreateConversationRequest{
			Title: "Test Conversation",
		}
		jsonBody, _ := json.Marshal(body)

		app := fiber.New()
		t.Cleanup(func() { _ = app.Shutdown() })
		app.Post("/test", func(c *fiber.Ctx) error {
			var req models.CreateConversationRequest
			if err := c.BodyParser(&req); err != nil {
				return c.Status(400).JSON(fiber.Map{"error": err.Error()})
			}
			assert.Equal(t, "Test Conversation", req.Title)
			return c.JSON(fiber.Map{"ok": true})
		})

		req := httptest.NewRequest("POST", "/test", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-User-ID", userID.String())

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("create with all options", func(t *testing.T) {
		temp := float32(0.8)
		maxTokens := 4096
		model := "gpt-4"
		systemPrompt := "You are a helpful assistant"

		body := models.CreateConversationRequest{
			Title:        "Full Options Test",
			ModelID:      &model,
			SystemPrompt: &systemPrompt,
			Temperature:  temp,
			MaxTokens:    maxTokens,
		}
		jsonBody, _ := json.Marshal(body)

		app := fiber.New()
		t.Cleanup(func() { _ = app.Shutdown() })
		app.Post("/test", func(c *fiber.Ctx) error {
			var req models.CreateConversationRequest
			if err := c.BodyParser(&req); err != nil {
				return c.Status(400).JSON(fiber.Map{"error": err.Error()})
			}
			assert.Equal(t, "Full Options Test", req.Title)
			assert.Equal(t, float32(0.8), req.Temperature)
			assert.Equal(t, 4096, req.MaxTokens)
			assert.NotNil(t, req.ModelID)
			assert.Equal(t, "gpt-4", *req.ModelID)
			return c.JSON(fiber.Map{"ok": true})
		})

		req := httptest.NewRequest("POST", "/test", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})
}

func TestGetConversation(t *testing.T) {
	t.Run("invalid conversation ID", func(t *testing.T) {
		handler := NewAIChatHandler(&ai.ChatService{})
		app := setupAIChatTestApp(t, handler)

		userID := uuid.New()
		req := httptest.NewRequest("GET", "/api/v1/ai/conversations/invalid-uuid", nil)
		req.Header.Set("X-Test-User-ID", userID.String())

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

func TestUpdateConversation(t *testing.T) {
	t.Run("update title", func(t *testing.T) {
		title := "Updated Title"
		body := models.UpdateConversationRequest{
			Title: &title,
		}
		jsonBody, _ := json.Marshal(body)

		app := fiber.New()
		t.Cleanup(func() { _ = app.Shutdown() })
		app.Patch("/test", func(c *fiber.Ctx) error {
			var req models.UpdateConversationRequest
			if err := c.BodyParser(&req); err != nil {
				return c.Status(400).JSON(fiber.Map{"error": err.Error()})
			}
			assert.NotNil(t, req.Title)
			assert.Equal(t, "Updated Title", *req.Title)
			return c.JSON(fiber.Map{"ok": true})
		})

		req := httptest.NewRequest("PATCH", "/test", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("update archived and pinned", func(t *testing.T) {
		archived := true
		pinned := false
		body := models.UpdateConversationRequest{
			IsArchived: &archived,
			IsPinned:   &pinned,
		}
		jsonBody, _ := json.Marshal(body)

		app := fiber.New()
		t.Cleanup(func() { _ = app.Shutdown() })
		app.Patch("/test", func(c *fiber.Ctx) error {
			var req models.UpdateConversationRequest
			if err := c.BodyParser(&req); err != nil {
				return c.Status(400).JSON(fiber.Map{"error": err.Error()})
			}
			assert.NotNil(t, req.IsArchived)
			assert.True(t, *req.IsArchived)
			assert.NotNil(t, req.IsPinned)
			assert.False(t, *req.IsPinned)
			return c.JSON(fiber.Map{"ok": true})
		})

		req := httptest.NewRequest("PATCH", "/test", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})
}

func TestDeleteConversation(t *testing.T) {
	t.Run("invalid conversation ID", func(t *testing.T) {
		handler := NewAIChatHandler(&ai.ChatService{})
		app := setupAIChatTestApp(t, handler)

		userID := uuid.New()
		req := httptest.NewRequest("DELETE", "/api/v1/ai/conversations/not-a-uuid", nil)
		req.Header.Set("X-Test-User-ID", userID.String())

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

// Tests for Messages

func TestSendMessage(t *testing.T) {
	t.Run("valid message", func(t *testing.T) {
		body := models.SendChatMessageRequest{
			Content: "Hello, AI!",
			Stream:  false,
		}
		jsonBody, _ := json.Marshal(body)

		app := fiber.New()
		t.Cleanup(func() { _ = app.Shutdown() })
		app.Post("/test", func(c *fiber.Ctx) error {
			var req models.SendChatMessageRequest
			if err := c.BodyParser(&req); err != nil {
				return c.Status(400).JSON(fiber.Map{"error": err.Error()})
			}
			assert.Equal(t, "Hello, AI!", req.Content)
			assert.False(t, req.Stream)
			return c.JSON(fiber.Map{"ok": true})
		})

		req := httptest.NewRequest("POST", "/test", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("empty message rejected", func(t *testing.T) {
		handler := NewAIChatHandler(&ai.ChatService{})
		app := setupAIChatTestApp(t, handler)

		userID := uuid.New()
		convID := uuid.New()

		body := models.SendChatMessageRequest{
			Content: "   ", // whitespace only
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/v1/ai/conversations/"+convID.String()+"/messages", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-User-ID", userID.String())

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("message with overrides", func(t *testing.T) {
		temp := float32(0.9)
		maxTokens := 1000
		body := models.SendChatMessageRequest{
			Content:     "Test with overrides",
			Stream:      true,
			ModelID:     "claude-3",
			Temperature: &temp,
			MaxTokens:   &maxTokens,
		}
		jsonBody, _ := json.Marshal(body)

		app := fiber.New()
		t.Cleanup(func() { _ = app.Shutdown() })
		app.Post("/test", func(c *fiber.Ctx) error {
			var req models.SendChatMessageRequest
			if err := c.BodyParser(&req); err != nil {
				return c.Status(400).JSON(fiber.Map{"error": err.Error()})
			}
			assert.Equal(t, "Test with overrides", req.Content)
			assert.True(t, req.Stream)
			assert.Equal(t, "claude-3", req.ModelID)
			assert.NotNil(t, req.Temperature)
			assert.Equal(t, float32(0.9), *req.Temperature)
			assert.NotNil(t, req.MaxTokens)
			assert.Equal(t, 1000, *req.MaxTokens)
			return c.JSON(fiber.Map{"ok": true})
		})

		req := httptest.NewRequest("POST", "/test", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})
}

func TestGetMessages(t *testing.T) {
	t.Run("invalid conversation ID", func(t *testing.T) {
		handler := NewAIChatHandler(&ai.ChatService{})
		app := setupAIChatTestApp(t, handler)

		userID := uuid.New()
		req := httptest.NewRequest("GET", "/api/v1/ai/conversations/bad-id/messages", nil)
		req.Header.Set("X-Test-User-ID", userID.String())

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

func TestDeleteMessage(t *testing.T) {
	t.Run("invalid message ID", func(t *testing.T) {
		handler := NewAIChatHandler(&ai.ChatService{})
		app := setupAIChatTestApp(t, handler)

		userID := uuid.New()
		convID := uuid.New()
		req := httptest.NewRequest("DELETE", "/api/v1/ai/conversations/"+convID.String()+"/messages/invalid", nil)
		req.Header.Set("X-Test-User-ID", userID.String())

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

// Tests for Templates

func TestListTemplates(t *testing.T) {
	t.Run("templates accessible without auth for public", func(t *testing.T) {
		mockSvc := new(MockChatService)
		handler := NewAIChatHandler(mockSvc)

		publicTemplates := []*models.AIChatTemplate{
			{
				ID:           uuid.New(),
				Name:         "Public Assistant",
				SystemPrompt: "You are a helpful assistant",
				Category:     stringPtr("general"),
				IsPublic:     true,
			},
		}
		mockSvc.On("GetTemplates", mock.Anything, (*uuid.UUID)(nil), "").Return(publicTemplates, nil)

		app := setupAIChatTestApp(t, handler)
		req := httptest.NewRequest("GET", "/api/v1/ai/templates", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var result map[string][]*models.AIChatTemplate
		err = json.Unmarshal(body, &result)
		require.NoError(t, err)
		assert.Len(t, result["templates"], 1)
		assert.Equal(t, "Public Assistant", result["templates"][0].Name)
		mockSvc.AssertExpectations(t)
	})

	t.Run("templates with category filter", func(t *testing.T) {
		mockSvc := new(MockChatService)
		handler := NewAIChatHandler(mockSvc)

		userID := uuid.New()
		filteredTemplates := []*models.AIChatTemplate{
			{
				ID:           uuid.New(),
				Name:         "Coding Assistant",
				SystemPrompt: "You are a coding expert",
				Category:     stringPtr("programming"),
				IsPublic:     true,
			},
		}
		mockSvc.On("GetTemplates", mock.Anything, &userID, "programming").Return(filteredTemplates, nil)

		app := setupAIChatTestApp(t, handler)
		req := httptest.NewRequest("GET", "/api/v1/ai/templates?category=programming", nil)
		req.Header.Set("X-Test-User-ID", userID.String())

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var result map[string][]*models.AIChatTemplate
		err = json.Unmarshal(body, &result)
		require.NoError(t, err)
		assert.Len(t, result["templates"], 1)
		assert.Equal(t, "Coding Assistant", result["templates"][0].Name)
		mockSvc.AssertExpectations(t)
	})

	t.Run("templates returns internal error on service failure", func(t *testing.T) {
		mockSvc := new(MockChatService)
		handler := NewAIChatHandler(mockSvc)

		mockSvc.On("GetTemplates", mock.Anything, (*uuid.UUID)(nil), "").Return(nil, errors.New("database error"))

		app := setupAIChatTestApp(t, handler)
		req := httptest.NewRequest("GET", "/api/v1/ai/templates", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
		mockSvc.AssertExpectations(t)
	})
}

func TestCreateTemplate(t *testing.T) {
	t.Run("valid template", func(t *testing.T) {
		body := models.AIChatTemplate{
			Name:         "Test Template",
			SystemPrompt: "You are a test assistant",
			Category:     stringPtr("testing"),
			IsPublic:     false,
		}
		jsonBody, _ := json.Marshal(body)

		app := fiber.New()
		t.Cleanup(func() { _ = app.Shutdown() })
		app.Post("/test", func(c *fiber.Ctx) error {
			var tmpl models.AIChatTemplate
			if err := c.BodyParser(&tmpl); err != nil {
				return c.Status(400).JSON(fiber.Map{"error": err.Error()})
			}
			assert.Equal(t, "Test Template", tmpl.Name)
			assert.Equal(t, "You are a test assistant", tmpl.SystemPrompt)
			return c.JSON(fiber.Map{"ok": true})
		})

		req := httptest.NewRequest("POST", "/test", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})
}

func TestGetTemplate(t *testing.T) {
	t.Run("invalid template ID", func(t *testing.T) {
		handler := NewAIChatHandler(&ai.ChatService{})
		app := setupAIChatTestApp(t, handler)

		req := httptest.NewRequest("GET", "/api/v1/ai/templates/not-a-uuid", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

// Tests for Shares

func TestShareConversation(t *testing.T) {
	t.Run("share request parsing", func(t *testing.T) {
		body := map[string]interface{}{
			"is_public":    true,
			"can_continue": false,
			"expires_in":   "24h",
		}
		jsonBody, _ := json.Marshal(body)

		app := fiber.New()
		t.Cleanup(func() { _ = app.Shutdown() })
		app.Post("/test", func(c *fiber.Ctx) error {
			var req struct {
				IsPublic    bool   `json:"is_public"`
				CanContinue bool   `json:"can_continue"`
				ExpiresIn   string `json:"expires_in"`
			}
			if err := c.BodyParser(&req); err != nil {
				return c.Status(400).JSON(fiber.Map{"error": err.Error()})
			}
			assert.True(t, req.IsPublic)
			assert.False(t, req.CanContinue)
			assert.Equal(t, "24h", req.ExpiresIn)
			return c.JSON(fiber.Map{"ok": true})
		})

		req := httptest.NewRequest("POST", "/test", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})
}

func TestGetSharedConversation(t *testing.T) {
	t.Run("share code parameter extraction", func(t *testing.T) {
		app := fiber.New()
		t.Cleanup(func() { _ = app.Shutdown() })
		app.Get("/shared/:code", func(c *fiber.Ctx) error {
			code := c.Params("code")
			assert.Equal(t, "abc123xyz", code)
			return c.JSON(fiber.Map{"code": code})
		})

		req := httptest.NewRequest("GET", "/shared/abc123xyz", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})
}

// Model validation tests

func TestConversationMessageRole(t *testing.T) {
	t.Run("valid roles", func(t *testing.T) {
		assert.True(t, models.ConvRoleSystem.Valid())
		assert.True(t, models.ConvRoleUser.Valid())
		assert.True(t, models.ConvRoleAssistant.Valid())
		assert.True(t, models.ConvRoleTool.Valid())
	})

	t.Run("invalid role", func(t *testing.T) {
		assert.False(t, models.ConversationMessageRole("invalid").Valid())
	})
}

func TestToolCallsJSON(t *testing.T) {
	t.Run("marshal and unmarshal", func(t *testing.T) {
		toolCalls := models.ToolCallsJSON{
			{
				ID:   "call_123",
				Type: "function",
				Function: models.ConversationToolCallFunction{
					Name:      "get_weather",
					Arguments: `{"location": "NYC"}`,
				},
			},
		}

		// Test Value
		value, err := toolCalls.Value()
		require.NoError(t, err)
		assert.NotNil(t, value)

		// Test Scan
		var decoded models.ToolCallsJSON
		err = decoded.Scan(value)
		require.NoError(t, err)
		assert.Len(t, decoded, 1)
		assert.Equal(t, "call_123", decoded[0].ID)
		assert.Equal(t, "get_weather", decoded[0].Function.Name)
	})

	t.Run("nil handling", func(t *testing.T) {
		var toolCalls models.ToolCallsJSON

		value, err := toolCalls.Value()
		require.NoError(t, err)
		assert.Nil(t, value)

		err = toolCalls.Scan(nil)
		require.NoError(t, err)
		assert.Nil(t, toolCalls)
	})
}

func TestStreamChunk(t *testing.T) {
	t.Run("stream chunk serialization", func(t *testing.T) {
		chunk := models.StreamChunk{
			ID:      "chatcmpl-123",
			Object:  "chat.completion.chunk",
			Created: time.Now().Unix(),
			Model:   "gpt-4",
			Choices: []models.StreamChoice{
				{
					Index: 0,
					Delta: models.StreamDelta{
						Content: "Hello",
					},
				},
			},
		}

		data, err := json.Marshal(chunk)
		require.NoError(t, err)

		var decoded models.StreamChunk
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, "chatcmpl-123", decoded.ID)
		assert.Equal(t, "Hello", decoded.Choices[0].Delta.Content)
	})
}

// Helper
func stringPtr(s string) *string {
	return &s
}
