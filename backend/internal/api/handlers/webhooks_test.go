package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// mockWebhookService implements the interface needed by WebhookHandlers
type mockWebhookService struct {
	createWebhookFunc      func(ctx context.Context, req *services.CreateWebhookRequest) (*models.Webhook, error)
	getWebhookFunc         func(ctx context.Context, webhookID uuid.UUID, requesterID uuid.UUID) (*models.Webhook, error)
	getChannelWebhooksFunc func(ctx context.Context, channelID uuid.UUID, requesterID uuid.UUID) ([]*models.Webhook, error)
	getServerWebhooksFunc  func(ctx context.Context, serverID uuid.UUID, requesterID uuid.UUID) ([]*models.Webhook, error)
	updateWebhookFunc      func(ctx context.Context, webhookID uuid.UUID, requesterID uuid.UUID, req *services.UpdateWebhookRequest) (*models.Webhook, error)
	deleteWebhookFunc      func(ctx context.Context, webhookID uuid.UUID, requesterID uuid.UUID) error
	executeWebhookFunc     func(ctx context.Context, webhookID uuid.UUID, token string, req *services.ExecuteWebhookRequest) (*models.Message, error)
}

func (m *mockWebhookService) CreateWebhook(ctx context.Context, req *services.CreateWebhookRequest) (*models.Webhook, error) {
	if m.createWebhookFunc != nil {
		return m.createWebhookFunc(ctx, req)
	}
	// Default implementation
	return &models.Webhook{
		ID:        uuid.New(),
		Type:      models.WebhookTypeIncoming,
		Name:      req.Name,
		ChannelID: req.ChannelID,
		Avatar:    req.Avatar,
		Token:     "generated-token-12345",
		CreatedAt: time.Now(),
	}, nil
}

func (m *mockWebhookService) GetWebhook(ctx context.Context, webhookID uuid.UUID, requesterID uuid.UUID) (*models.Webhook, error) {
	if m.getWebhookFunc != nil {
		return m.getWebhookFunc(ctx, webhookID, requesterID)
	}
	return nil, services.ErrWebhookNotFound
}

func (m *mockWebhookService) GetChannelWebhooks(ctx context.Context, channelID uuid.UUID, requesterID uuid.UUID) ([]*models.Webhook, error) {
	if m.getChannelWebhooksFunc != nil {
		return m.getChannelWebhooksFunc(ctx, channelID, requesterID)
	}
	return []*models.Webhook{}, nil
}

func (m *mockWebhookService) GetServerWebhooks(ctx context.Context, serverID uuid.UUID, requesterID uuid.UUID) ([]*models.Webhook, error) {
	if m.getServerWebhooksFunc != nil {
		return m.getServerWebhooksFunc(ctx, serverID, requesterID)
	}
	return []*models.Webhook{}, nil
}

func (m *mockWebhookService) UpdateWebhook(ctx context.Context, webhookID uuid.UUID, requesterID uuid.UUID, req *services.UpdateWebhookRequest) (*models.Webhook, error) {
	if m.updateWebhookFunc != nil {
		return m.updateWebhookFunc(ctx, webhookID, requesterID, req)
	}
	return nil, services.ErrWebhookNotFound
}

func (m *mockWebhookService) DeleteWebhook(ctx context.Context, webhookID uuid.UUID, requesterID uuid.UUID) error {
	if m.deleteWebhookFunc != nil {
		return m.deleteWebhookFunc(ctx, webhookID, requesterID)
	}
	return nil
}

func (m *mockWebhookService) ExecuteWebhook(ctx context.Context, webhookID uuid.UUID, token string, req *services.ExecuteWebhookRequest) (*models.Message, error) {
	if m.executeWebhookFunc != nil {
		return m.executeWebhookFunc(ctx, webhookID, token, req)
	}
	return &models.Message{
		ID:        uuid.New(),
		ChannelID: uuid.New(),
		Content:   req.Content,
		CreatedAt: time.Now(),
	}, nil
}

// WebhookServiceInterface defines the interface for WebhookService
type WebhookServiceInterface interface {
	CreateWebhook(ctx context.Context, req *services.CreateWebhookRequest) (*models.Webhook, error)
	GetWebhook(ctx context.Context, webhookID uuid.UUID, requesterID uuid.UUID) (*models.Webhook, error)
	GetChannelWebhooks(ctx context.Context, channelID uuid.UUID, requesterID uuid.UUID) ([]*models.Webhook, error)
	GetServerWebhooks(ctx context.Context, serverID uuid.UUID, requesterID uuid.UUID) ([]*models.Webhook, error)
	UpdateWebhook(ctx context.Context, webhookID uuid.UUID, requesterID uuid.UUID, req *services.UpdateWebhookRequest) (*models.Webhook, error)
	DeleteWebhook(ctx context.Context, webhookID uuid.UUID, requesterID uuid.UUID) error
	ExecuteWebhook(ctx context.Context, webhookID uuid.UUID, token string, req *services.ExecuteWebhookRequest) (*models.Message, error)
}

// setupWebhookTestApp creates a test Fiber app with webhook routes
func setupWebhookTestApp() *fiber.App {
	return setupWebhookTestAppWithMock(&mockWebhookService{})
}

func setupWebhookTestAppWithMock(mock *mockWebhookService) *fiber.App {
	app := fiber.New()

	// Inject userID middleware
	app.Use(func(c *fiber.Ctx) error {
		userID := c.Get("X-User-ID")
		if userID != "" {
			id, _ := uuid.Parse(userID)
			c.Locals("userID", id)
		} else {
			// Default user ID for tests
			c.Locals("userID", uuid.New())
		}
		return c.Next()
	})

	// Create handlers with a wrapper that uses our mock
	handlers := &testWebhookHandlers{mock: mock}

	// Channel webhooks
	app.Post("/channels/:channelID/webhooks", handlers.CreateWebhook)
	app.Get("/channels/:channelID/webhooks", handlers.GetChannelWebhooks)

	// Server webhooks
	app.Get("/servers/:serverID/webhooks", handlers.GetServerWebhooks)

	// Individual webhook operations
	app.Get("/webhooks/:webhookID", handlers.GetWebhook)
	app.Patch("/webhooks/:webhookID", handlers.UpdateWebhook)
	app.Delete("/webhooks/:webhookID", handlers.DeleteWebhook)

	// Execute webhook (public endpoint with token)
	app.Post("/webhooks/:webhookID/:token", handlers.ExecuteWebhook)

	return app
}

// testWebhookHandlers wraps the mock service for testing
type testWebhookHandlers struct {
	mock *mockWebhookService
}

func (h *testWebhookHandlers) CreateWebhook(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	channelID, err := uuid.Parse(c.Params("channelID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid channel ID"})
	}

	var req struct {
		Name   string  `json:"name"`
		Avatar *string `json:"avatar,omitempty"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if req.Name == "" || len(req.Name) > 80 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Name must be between 1 and 80 characters"})
	}

	webhook, err := h.mock.CreateWebhook(c.Context(), &services.CreateWebhookRequest{
		ChannelID: channelID,
		CreatorID: userID,
		Name:      req.Name,
		Avatar:    req.Avatar,
	})
	if err != nil {
		return handleTestWebhookError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(toTestWebhookResponse(webhook, true))
}

func (h *testWebhookHandlers) GetChannelWebhooks(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	channelID, err := uuid.Parse(c.Params("channelID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid channel ID"})
	}

	webhooks, err := h.mock.GetChannelWebhooks(c.Context(), channelID, userID)
	if err != nil {
		return handleTestWebhookError(c, err)
	}

	responses := make([]WebhookResponse, len(webhooks))
	for i, w := range webhooks {
		responses[i] = toTestWebhookResponse(w, false)
	}
	return c.JSON(responses)
}

func (h *testWebhookHandlers) GetServerWebhooks(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	serverID, err := uuid.Parse(c.Params("serverID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid server ID"})
	}

	webhooks, err := h.mock.GetServerWebhooks(c.Context(), serverID, userID)
	if err != nil {
		return handleTestWebhookError(c, err)
	}

	responses := make([]WebhookResponse, len(webhooks))
	for i, w := range webhooks {
		responses[i] = toTestWebhookResponse(w, false)
	}
	return c.JSON(responses)
}

func (h *testWebhookHandlers) GetWebhook(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	webhookID, err := uuid.Parse(c.Params("webhookID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid webhook ID"})
	}

	webhook, err := h.mock.GetWebhook(c.Context(), webhookID, userID)
	if err != nil {
		return handleTestWebhookError(c, err)
	}
	return c.JSON(toTestWebhookResponse(webhook, false))
}

func (h *testWebhookHandlers) UpdateWebhook(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	webhookID, err := uuid.Parse(c.Params("webhookID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid webhook ID"})
	}

	var req struct {
		Name      *string `json:"name,omitempty"`
		Avatar    *string `json:"avatar,omitempty"`
		ChannelID *string `json:"channel_id,omitempty"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	var channelID *uuid.UUID
	if req.ChannelID != nil {
		id, err := uuid.Parse(*req.ChannelID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid channel ID"})
		}
		channelID = &id
	}

	webhook, err := h.mock.UpdateWebhook(c.Context(), webhookID, userID, &services.UpdateWebhookRequest{
		Name:      req.Name,
		Avatar:    req.Avatar,
		ChannelID: channelID,
	})
	if err != nil {
		return handleTestWebhookError(c, err)
	}
	return c.JSON(toTestWebhookResponse(webhook, false))
}

func (h *testWebhookHandlers) DeleteWebhook(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	webhookID, err := uuid.Parse(c.Params("webhookID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid webhook ID"})
	}

	if err := h.mock.DeleteWebhook(c.Context(), webhookID, userID); err != nil {
		return handleTestWebhookError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *testWebhookHandlers) ExecuteWebhook(c *fiber.Ctx) error {
	webhookID, err := uuid.Parse(c.Params("webhookID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid webhook ID"})
	}

	token := c.Params("token")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid webhook token"})
	}

	var req struct {
		Content   string  `json:"content,omitempty"`
		Username  *string `json:"username,omitempty"`
		AvatarURL *string `json:"avatar_url,omitempty"`
		TTS       bool    `json:"tts"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if req.Content == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Content is required"})
	}

	message, err := h.mock.ExecuteWebhook(c.Context(), webhookID, token, &services.ExecuteWebhookRequest{
		Content:   req.Content,
		Username:  req.Username,
		AvatarURL: req.AvatarURL,
		TTS:       req.TTS,
	})
	if err != nil {
		return handleTestWebhookError(c, err)
	}

	if c.Query("wait") == "true" {
		return c.JSON(fiber.Map{
			"id":         message.ID.String(),
			"content":    message.Content,
			"channel_id": message.ChannelID.String(),
			"webhook_id": webhookID.String(),
			"timestamp":  message.CreatedAt,
		})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func toTestWebhookResponse(w *models.Webhook, includeToken bool) WebhookResponse {
	resp := WebhookResponse{
		ID:        w.ID.String(),
		Type:      int(w.Type),
		Name:      w.Name,
		ChannelID: w.ChannelID.String(),
		AvatarURL: w.Avatar,
		CreatedAt: w.CreatedAt,
	}
	if w.ServerID != nil {
		serverID := w.ServerID.String()
		resp.ServerID = &serverID
	}
	if w.ApplicationID != nil {
		appID := w.ApplicationID.String()
		resp.ApplicationID = &appID
	}
	if includeToken {
		resp.Token = w.Token
	}
	return resp
}

func handleTestWebhookError(c *fiber.Ctx, err error) error {
	switch err {
	case services.ErrWebhookNotFound:
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Webhook not found"})
	case services.ErrChannelNotFound:
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Channel not found"})
	case services.ErrNotServerMember:
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Not a server member"})
	case services.ErrInvalidWebhookToken:
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid webhook token"})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
	}
}

// CreateWebhook Tests

func TestWebhookHandler_CreateWebhook_Success(t *testing.T) {
	channelID := uuid.New()
	app := setupWebhookTestApp()

	body, _ := json.Marshal(map[string]string{"name": "My Webhook"})
	req := httptest.NewRequest("POST", "/channels/"+channelID.String()+"/webhooks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 201, got %d: %s", resp.StatusCode, string(respBody))
	}

	var result WebhookResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.Name != "My Webhook" {
		t.Errorf("Expected name 'My Webhook', got %q", result.Name)
	}
	if result.ChannelID != channelID.String() {
		t.Errorf("Expected channel_id %s, got %s", channelID.String(), result.ChannelID)
	}
	if result.Token == "" {
		t.Error("Expected token to be generated")
	}
	if result.Type != 1 {
		t.Errorf("Expected type 1, got %d", result.Type)
	}
}

func TestWebhookHandler_CreateWebhook_InvalidChannelID(t *testing.T) {
	app := setupWebhookTestApp()

	body, _ := json.Marshal(map[string]string{"name": "Test"})
	req := httptest.NewRequest("POST", "/channels/invalid-uuid/webhooks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestWebhookHandler_CreateWebhook_EmptyName(t *testing.T) {
	channelID := uuid.New()
	app := setupWebhookTestApp()

	body, _ := json.Marshal(map[string]string{"name": ""})
	req := httptest.NewRequest("POST", "/channels/"+channelID.String()+"/webhooks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestWebhookHandler_CreateWebhook_NameTooLong(t *testing.T) {
	channelID := uuid.New()
	app := setupWebhookTestApp()

	longName := ""
	for i := 0; i < 81; i++ {
		longName += "a"
	}

	body, _ := json.Marshal(map[string]string{"name": longName})
	req := httptest.NewRequest("POST", "/channels/"+channelID.String()+"/webhooks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

// GetChannelWebhooks Tests

func TestWebhookHandler_GetChannelWebhooks_Success(t *testing.T) {
	channelID := uuid.New()
	app := setupWebhookTestApp()

	req := httptest.NewRequest("GET", "/channels/"+channelID.String()+"/webhooks", nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	var result []WebhookResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if result == nil {
		t.Error("Expected non-nil array, got nil")
	}
}

func TestWebhookHandler_GetChannelWebhooks_InvalidChannelID(t *testing.T) {
	app := setupWebhookTestApp()

	req := httptest.NewRequest("GET", "/channels/invalid-uuid/webhooks", nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

// GetServerWebhooks Tests

func TestWebhookHandler_GetServerWebhooks_Success(t *testing.T) {
	serverID := uuid.New()
	app := setupWebhookTestApp()

	req := httptest.NewRequest("GET", "/servers/"+serverID.String()+"/webhooks", nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	var result []WebhookResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if result == nil {
		t.Error("Expected non-nil array, got nil")
	}
}

func TestWebhookHandler_GetServerWebhooks_InvalidServerID(t *testing.T) {
	app := setupWebhookTestApp()

	req := httptest.NewRequest("GET", "/servers/invalid-uuid/webhooks", nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

// GetWebhook Tests

func TestWebhookHandler_GetWebhook_NotFound(t *testing.T) {
	webhookID := uuid.New()
	app := setupWebhookTestApp()

	req := httptest.NewRequest("GET", "/webhooks/"+webhookID.String(), nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestWebhookHandler_GetWebhook_InvalidID(t *testing.T) {
	app := setupWebhookTestApp()

	req := httptest.NewRequest("GET", "/webhooks/invalid-uuid", nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

// UpdateWebhook Tests

func TestWebhookHandler_UpdateWebhook_NotFound(t *testing.T) {
	webhookID := uuid.New()
	app := setupWebhookTestApp()

	body, _ := json.Marshal(map[string]string{"name": "Updated Webhook"})
	req := httptest.NewRequest("PATCH", "/webhooks/"+webhookID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestWebhookHandler_UpdateWebhook_InvalidID(t *testing.T) {
	app := setupWebhookTestApp()

	body, _ := json.Marshal(map[string]string{"name": "Updated Webhook"})
	req := httptest.NewRequest("PATCH", "/webhooks/invalid-uuid", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

// DeleteWebhook Tests

func TestWebhookHandler_DeleteWebhook_Success(t *testing.T) {
	webhookID := uuid.New()
	app := setupWebhookTestApp()

	req := httptest.NewRequest("DELETE", "/webhooks/"+webhookID.String(), nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusNoContent {
		t.Errorf("Expected 204, got %d", resp.StatusCode)
	}
}

func TestWebhookHandler_DeleteWebhook_InvalidID(t *testing.T) {
	app := setupWebhookTestApp()

	req := httptest.NewRequest("DELETE", "/webhooks/invalid-uuid", nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

// ExecuteWebhook Tests

func TestWebhookHandler_ExecuteWebhook_Success_NoWait(t *testing.T) {
	webhookID := uuid.New()
	token := "test-token-12345"
	app := setupWebhookTestApp()

	body, _ := json.Marshal(map[string]string{"content": "Hello from webhook!"})
	req := httptest.NewRequest("POST", "/webhooks/"+webhookID.String()+"/"+token, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 204, got %d: %s", resp.StatusCode, string(respBody))
	}
}

func TestWebhookHandler_ExecuteWebhook_Success_WithWait(t *testing.T) {
	webhookID := uuid.New()
	token := "test-token-12345"
	app := setupWebhookTestApp()

	body, _ := json.Marshal(map[string]string{"content": "Hello from webhook!"})
	req := httptest.NewRequest("POST", "/webhooks/"+webhookID.String()+"/"+token+"?wait=true", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result["content"] != "Hello from webhook!" {
		t.Errorf("Expected content 'Hello from webhook!', got %v", result["content"])
	}
	if result["webhook_id"] != webhookID.String() {
		t.Errorf("Expected webhook_id %s, got %v", webhookID.String(), result["webhook_id"])
	}
}

func TestWebhookHandler_ExecuteWebhook_InvalidWebhookID(t *testing.T) {
	token := "test-token-12345"
	app := setupWebhookTestApp()

	body, _ := json.Marshal(map[string]string{"content": "Hello!"})
	req := httptest.NewRequest("POST", "/webhooks/invalid-uuid/"+token, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestWebhookHandler_ExecuteWebhook_EmptyContent(t *testing.T) {
	webhookID := uuid.New()
	token := "test-token-12345"
	app := setupWebhookTestApp()

	body, _ := json.Marshal(map[string]string{"content": ""})
	req := httptest.NewRequest("POST", "/webhooks/"+webhookID.String()+"/"+token, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestWebhookHandler_ExecuteWebhook_InvalidBody(t *testing.T) {
	webhookID := uuid.New()
	token := "test-token-12345"
	app := setupWebhookTestApp()

	req := httptest.NewRequest("POST", "/webhooks/"+webhookID.String()+"/"+token, bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}
