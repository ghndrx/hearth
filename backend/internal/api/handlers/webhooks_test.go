package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"hearth/internal/models"
	"hearth/internal/services"
)

// --- mockWebhookService ---

type mockWebhookService struct {
	mock.Mock
}

func (m *mockWebhookService) CreateWebhook(ctx context.Context, req *services.CreateWebhookRequest) (*models.Webhook, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Webhook), args.Error(1)
}

func (m *mockWebhookService) GetWebhook(ctx context.Context, webhookID uuid.UUID, requesterID uuid.UUID) (*models.Webhook, error) {
	args := m.Called(ctx, webhookID, requesterID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Webhook), args.Error(1)
}

func (m *mockWebhookService) GetChannelWebhooks(ctx context.Context, channelID uuid.UUID, requesterID uuid.UUID) ([]*models.Webhook, error) {
	args := m.Called(ctx, channelID, requesterID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Webhook), args.Error(1)
}

func (m *mockWebhookService) GetServerWebhooks(ctx context.Context, serverID uuid.UUID, requesterID uuid.UUID) ([]*models.Webhook, error) {
	args := m.Called(ctx, serverID, requesterID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Webhook), args.Error(1)
}

func (m *mockWebhookService) UpdateWebhook(ctx context.Context, webhookID uuid.UUID, requesterID uuid.UUID, req *services.UpdateWebhookRequest) (*models.Webhook, error) {
	args := m.Called(ctx, webhookID, requesterID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Webhook), args.Error(1)
}

func (m *mockWebhookService) DeleteWebhook(ctx context.Context, webhookID uuid.UUID, requesterID uuid.UUID) error {
	args := m.Called(ctx, webhookID, requesterID)
	return args.Error(0)
}

func (m *mockWebhookService) CheckRateLimit(ctx context.Context, webhookID uuid.UUID) error {
	args := m.Called(ctx, webhookID)
	return args.Error(0)
}

func (m *mockWebhookService) ExecuteWebhook(ctx context.Context, webhookID uuid.UUID, token string, req *services.ExecuteWebhookRequest) (*models.Message, error) {
	args := m.Called(ctx, webhookID, token, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Message), args.Error(1)
}

func (m *mockWebhookService) ExecuteWebhookWithRetry(ctx context.Context, webhookID uuid.UUID, token string, req *services.ExecuteWebhookRequest) (*models.Message, error) {
	args := m.Called(ctx, webhookID, token, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Message), args.Error(1)
}

func (m *mockWebhookService) GetWebhookStats(ctx context.Context, webhookID uuid.UUID, requesterID uuid.UUID) (*models.WebhookDeliveryStats, error) {
	args := m.Called(ctx, webhookID, requesterID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.WebhookDeliveryStats), args.Error(1)
}

func (m *mockWebhookService) GetWebhookDeliveries(ctx context.Context, webhookID uuid.UUID, requesterID uuid.UUID, limit, offset int) ([]*models.WebhookDelivery, error) {
	args := m.Called(ctx, webhookID, requesterID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.WebhookDelivery), args.Error(1)
}

func (m *mockWebhookService) TestWebhook(ctx context.Context, webhookID uuid.UUID, requesterID uuid.UUID) (*models.Message, error) {
	args := m.Called(ctx, webhookID, requesterID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Message), args.Error(1)
}

// --- test app setup ---

func setupWebhookTestApp(t *testing.T, mockService *mockWebhookService) (*fiber.App, uuid.UUID) {
	app := fiber.New()
	t.Cleanup(func() { app.Shutdown() })
	userID := uuid.New()

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", userID)
		return c.Next()
	})

	handlers := NewWebhookHandlers(mockService)

	app.Post("/channels/:channelID/webhooks", handlers.CreateWebhook)
	app.Get("/channels/:channelID/webhooks", handlers.GetChannelWebhooks)
	app.Get("/servers/:serverID/webhooks", handlers.GetServerWebhooks)
	app.Get("/webhooks/:webhookID", handlers.GetWebhook)
	app.Patch("/webhooks/:webhookID", handlers.UpdateWebhook)
	app.Delete("/webhooks/:webhookID", handlers.DeleteWebhook)
	app.Post("/webhooks/:webhookID/:token", handlers.ExecuteWebhook)

	return app, userID
}

// --- CreateWebhook tests ---

func TestCreateWebhook_Success(t *testing.T) {
	mockService := new(mockWebhookService)
	app, userID := setupWebhookTestApp(t, mockService)
	channelID := uuid.New()
	webhookID := uuid.New()
	serverID := uuid.New()

	expected := &models.Webhook{
		ID:        webhookID,
		Name:      "test-webhook",
		ChannelID: channelID,
		ServerID:  &serverID,
		Token:     "test-token",
		Type:      1,
	}

	mockService.On("CreateWebhook", mock.Anything, mock.MatchedBy(func(req *services.CreateWebhookRequest) bool {
		return req.ChannelID == channelID && req.Name == "test-webhook"
	})).Return(expected, nil)

	body := `{"name":"test-webhook"}`
	req := httptest.NewRequest(http.MethodPost, "/channels/"+channelID.String()+"/webhooks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// set userID via middleware (uses captured userID from setup)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result WebhookResponse
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, webhookID.String(), result.ID)
	assert.Equal(t, "test-webhook", result.Name)

	mockService.AssertExpectations(t)
}

func TestCreateWebhook_InvalidChannelID(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)

	body := `{"name":"test-webhook"}`
	req := httptest.NewRequest(http.MethodPost, "/channels/invalid-uuid/webhooks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestCreateWebhook_InvalidBody(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)
	channelID := uuid.New()

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/channels/"+channelID.String()+"/webhooks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestCreateWebhook_ChannelNotFound(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)
	channelID := uuid.New()

	mockService.On("CreateWebhook", mock.Anything, mock.Anything).Return(nil, services.ErrChannelNotFound)

	body := `{"name":"test-webhook"}`
	req := httptest.NewRequest(http.MethodPost, "/channels/"+channelID.String()+"/webhooks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestCreateWebhook_NotServerMember(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)
	channelID := uuid.New()

	mockService.On("CreateWebhook", mock.Anything, mock.Anything).Return(nil, services.ErrNotServerMember)

	body := `{"name":"test-webhook"}`
	req := httptest.NewRequest(http.MethodPost, "/channels/"+channelID.String()+"/webhooks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestCreateWebhook_MissingManageWebhooks(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)
	channelID := uuid.New()

	mockService.On("CreateWebhook", mock.Anything, mock.Anything).Return(nil, services.ErrMissingManageWebhooks)

	body := `{"name":"test-webhook"}`
	req := httptest.NewRequest(http.MethodPost, "/channels/"+channelID.String()+"/webhooks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestCreateWebhook_NameTooLong(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)
	channelID := uuid.New()

	mockService.On("CreateWebhook", mock.Anything, mock.Anything).Return(nil, services.ErrWebhookNameTooLong)

	body := `{"name":"test-webhook"}`
	req := httptest.NewRequest(http.MethodPost, "/channels/"+channelID.String()+"/webhooks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestCreateWebhook_TooManyWebhooks(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)
	channelID := uuid.New()

	mockService.On("CreateWebhook", mock.Anything, mock.Anything).Return(nil, services.ErrTooManyWebhooks)

	body := `{"name":"test-webhook"}`
	req := httptest.NewRequest(http.MethodPost, "/channels/"+channelID.String()+"/webhooks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestCreateWebhook_InternalError(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)
	channelID := uuid.New()

	mockService.On("CreateWebhook", mock.Anything, mock.Anything).Return(nil, assert.AnError)

	body := `{"name":"test-webhook"}`
	req := httptest.NewRequest(http.MethodPost, "/channels/"+channelID.String()+"/webhooks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	mockService.AssertExpectations(t)
}

// --- GetChannelWebhooks tests ---

func TestGetChannelWebhooks_Success(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)
	channelID := uuid.New()
	webhookID := uuid.New()
	serverID := uuid.New()

	expected := []*models.Webhook{
		{ID: webhookID, Name: "hook1", ChannelID: channelID, ServerID: &serverID, Token: "tok1", Type: 1},
	}

	mockService.On("GetChannelWebhooks", mock.Anything, channelID, mock.Anything).Return(expected, nil)

	req := httptest.NewRequest(http.MethodGet, "/channels/"+channelID.String()+"/webhooks", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result []WebhookResponse
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Len(t, result, 1)
	assert.Equal(t, webhookID.String(), result[0].ID)

	mockService.AssertExpectations(t)
}

func TestGetChannelWebhooks_InvalidChannelID(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)

	req := httptest.NewRequest(http.MethodGet, "/channels/not-a-uuid/webhooks", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestGetChannelWebhooks_ChannelNotFound(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)
	channelID := uuid.New()

	mockService.On("GetChannelWebhooks", mock.Anything, channelID, mock.Anything).Return(nil, services.ErrChannelNotFound)

	req := httptest.NewRequest(http.MethodGet, "/channels/"+channelID.String()+"/webhooks", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestGetChannelWebhooks_NotServerMember(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)
	channelID := uuid.New()

	mockService.On("GetChannelWebhooks", mock.Anything, channelID, mock.Anything).Return(nil, services.ErrNotServerMember)

	req := httptest.NewRequest(http.MethodGet, "/channels/"+channelID.String()+"/webhooks", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestGetChannelWebhooks_InternalError(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)
	channelID := uuid.New()

	mockService.On("GetChannelWebhooks", mock.Anything, channelID, mock.Anything).Return(nil, assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/channels/"+channelID.String()+"/webhooks", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	mockService.AssertExpectations(t)
}

// --- GetServerWebhooks tests ---

func TestGetServerWebhooks_Success(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)
	serverID := uuid.New()
	webhookID := uuid.New()
	channelID := uuid.New()

	expected := []*models.Webhook{
		{ID: webhookID, Name: "server-hook", ChannelID: channelID, ServerID: &serverID, Token: "tok1", Type: 1},
	}

	mockService.On("GetServerWebhooks", mock.Anything, serverID, mock.Anything).Return(expected, nil)

	req := httptest.NewRequest(http.MethodGet, "/servers/"+serverID.String()+"/webhooks", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result []WebhookResponse
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Len(t, result, 1)
	assert.Equal(t, webhookID.String(), result[0].ID)

	mockService.AssertExpectations(t)
}

func TestGetServerWebhooks_InvalidServerID(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)

	req := httptest.NewRequest(http.MethodGet, "/servers/not-a-uuid/webhooks", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestGetServerWebhooks_NotServerMember(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)
	serverID := uuid.New()

	mockService.On("GetServerWebhooks", mock.Anything, serverID, mock.Anything).Return(nil, services.ErrNotServerMember)

	req := httptest.NewRequest(http.MethodGet, "/servers/"+serverID.String()+"/webhooks", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestGetServerWebhooks_MissingManageWebhooks(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)
	serverID := uuid.New()

	mockService.On("GetServerWebhooks", mock.Anything, serverID, mock.Anything).Return(nil, services.ErrMissingManageWebhooks)

	req := httptest.NewRequest(http.MethodGet, "/servers/"+serverID.String()+"/webhooks", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestGetServerWebhooks_InternalError(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)
	serverID := uuid.New()

	mockService.On("GetServerWebhooks", mock.Anything, serverID, mock.Anything).Return(nil, assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/servers/"+serverID.String()+"/webhooks", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	mockService.AssertExpectations(t)
}

// --- GetWebhook tests ---

func TestGetWebhook_Success(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)
	webhookID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	expected := &models.Webhook{
		ID:        webhookID,
		Name:      "my-hook",
		ChannelID: channelID,
		ServerID:  &serverID,
		Token:     "secret-token",
		Type:      1,
	}

	mockService.On("GetWebhook", mock.Anything, webhookID, mock.Anything).Return(expected, nil)

	req := httptest.NewRequest(http.MethodGet, "/webhooks/"+webhookID.String(), nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result WebhookResponse
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, webhookID.String(), result.ID)
	assert.Equal(t, "my-hook", result.Name)

	mockService.AssertExpectations(t)
}

func TestGetWebhook_InvalidWebhookID(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)

	req := httptest.NewRequest(http.MethodGet, "/webhooks/not-a-uuid", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestGetWebhook_NotFound(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)
	webhookID := uuid.New()

	mockService.On("GetWebhook", mock.Anything, webhookID, mock.Anything).Return(nil, services.ErrWebhookNotFound)

	req := httptest.NewRequest(http.MethodGet, "/webhooks/"+webhookID.String(), nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestGetWebhook_ChannelNotFound(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)
	webhookID := uuid.New()

	mockService.On("GetWebhook", mock.Anything, webhookID, mock.Anything).Return(nil, services.ErrChannelNotFound)

	req := httptest.NewRequest(http.MethodGet, "/webhooks/"+webhookID.String(), nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestGetWebhook_NotServerMember(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)
	webhookID := uuid.New()

	mockService.On("GetWebhook", mock.Anything, webhookID, mock.Anything).Return(nil, services.ErrNotServerMember)

	req := httptest.NewRequest(http.MethodGet, "/webhooks/"+webhookID.String(), nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestGetWebhook_InternalError(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)
	webhookID := uuid.New()

	mockService.On("GetWebhook", mock.Anything, webhookID, mock.Anything).Return(nil, assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/webhooks/"+webhookID.String(), nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	mockService.AssertExpectations(t)
}

// --- UpdateWebhook tests ---

func TestUpdateWebhook_Success(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)
	webhookID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	expected := &models.Webhook{
		ID:        webhookID,
		Name:      "updated-hook",
		ChannelID: channelID,
		ServerID:  &serverID,
		Token:     "tok",
		Type:      1,
	}

	mockService.On("UpdateWebhook", mock.Anything, webhookID, mock.Anything, mock.Anything).Return(expected, nil)

	body := `{"name":"updated-hook"}`
	req := httptest.NewRequest(http.MethodPatch, "/webhooks/"+webhookID.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result WebhookResponse
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "updated-hook", result.Name)

	mockService.AssertExpectations(t)
}

func TestUpdateWebhook_InvalidWebhookID(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)

	body := `{"name":"updated"}`
	req := httptest.NewRequest(http.MethodPatch, "/webhooks/not-a-uuid", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestUpdateWebhook_InvalidBody(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)
	webhookID := uuid.New()

	body := `{invalid}`
	req := httptest.NewRequest(http.MethodPatch, "/webhooks/"+webhookID.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestUpdateWebhook_InvalidChannelID(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)
	webhookID := uuid.New()

	body := `{"channel_id":"not-a-uuid"}`
	req := httptest.NewRequest(http.MethodPatch, "/webhooks/"+webhookID.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestUpdateWebhook_WebhookNotFound(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)
	webhookID := uuid.New()

	mockService.On("UpdateWebhook", mock.Anything, webhookID, mock.Anything, mock.Anything).Return(nil, services.ErrWebhookNotFound)

	body := `{"name":"updated"}`
	req := httptest.NewRequest(http.MethodPatch, "/webhooks/"+webhookID.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestUpdateWebhook_ChannelNotFound(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)
	webhookID := uuid.New()

	mockService.On("UpdateWebhook", mock.Anything, webhookID, mock.Anything, mock.Anything).Return(nil, services.ErrChannelNotFound)

	body := `{"name":"updated"}`
	req := httptest.NewRequest(http.MethodPatch, "/webhooks/"+webhookID.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestUpdateWebhook_NotServerMember(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)
	webhookID := uuid.New()

	mockService.On("UpdateWebhook", mock.Anything, webhookID, mock.Anything, mock.Anything).Return(nil, services.ErrNotServerMember)

	body := `{"name":"updated"}`
	req := httptest.NewRequest(http.MethodPatch, "/webhooks/"+webhookID.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestUpdateWebhook_MissingManageWebhooks(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)
	webhookID := uuid.New()

	mockService.On("UpdateWebhook", mock.Anything, webhookID, mock.Anything, mock.Anything).Return(nil, services.ErrMissingManageWebhooks)

	body := `{"name":"updated"}`
	req := httptest.NewRequest(http.MethodPatch, "/webhooks/"+webhookID.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestUpdateWebhook_NameTooLong(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)
	webhookID := uuid.New()

	mockService.On("UpdateWebhook", mock.Anything, webhookID, mock.Anything, mock.Anything).Return(nil, services.ErrWebhookNameTooLong)

	body := `{"name":"updated"}`
	req := httptest.NewRequest(http.MethodPatch, "/webhooks/"+webhookID.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestUpdateWebhook_NoPermission_CrossServer(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)
	webhookID := uuid.New()

	mockService.On("UpdateWebhook", mock.Anything, webhookID, mock.Anything, mock.Anything).Return(nil, services.ErrNoPermission)

	body := `{"name":"updated"}`
	req := httptest.NewRequest(http.MethodPatch, "/webhooks/"+webhookID.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestUpdateWebhook_InternalError(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)
	webhookID := uuid.New()

	mockService.On("UpdateWebhook", mock.Anything, webhookID, mock.Anything, mock.Anything).Return(nil, assert.AnError)

	body := `{"name":"updated"}`
	req := httptest.NewRequest(http.MethodPatch, "/webhooks/"+webhookID.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	mockService.AssertExpectations(t)
}

// --- DeleteWebhook tests ---

func TestDeleteWebhook_Success(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)
	webhookID := uuid.New()

	mockService.On("DeleteWebhook", mock.Anything, webhookID, mock.Anything).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/webhooks/"+webhookID.String(), nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestDeleteWebhook_InvalidWebhookID(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)

	req := httptest.NewRequest(http.MethodDelete, "/webhooks/not-a-uuid", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestDeleteWebhook_NotFound(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)
	webhookID := uuid.New()

	mockService.On("DeleteWebhook", mock.Anything, webhookID, mock.Anything).Return(services.ErrWebhookNotFound)

	req := httptest.NewRequest(http.MethodDelete, "/webhooks/"+webhookID.String(), nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestDeleteWebhook_ChannelNotFound(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)
	webhookID := uuid.New()

	mockService.On("DeleteWebhook", mock.Anything, webhookID, mock.Anything).Return(services.ErrChannelNotFound)

	req := httptest.NewRequest(http.MethodDelete, "/webhooks/"+webhookID.String(), nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestDeleteWebhook_NotServerMember(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)
	webhookID := uuid.New()

	mockService.On("DeleteWebhook", mock.Anything, webhookID, mock.Anything).Return(services.ErrNotServerMember)

	req := httptest.NewRequest(http.MethodDelete, "/webhooks/"+webhookID.String(), nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestDeleteWebhook_MissingManageWebhooks(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)
	webhookID := uuid.New()

	mockService.On("DeleteWebhook", mock.Anything, webhookID, mock.Anything).Return(services.ErrMissingManageWebhooks)

	req := httptest.NewRequest(http.MethodDelete, "/webhooks/"+webhookID.String(), nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestDeleteWebhook_InternalError(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)
	webhookID := uuid.New()

	mockService.On("DeleteWebhook", mock.Anything, webhookID, mock.Anything).Return(assert.AnError)

	req := httptest.NewRequest(http.MethodDelete, "/webhooks/"+webhookID.String(), nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	mockService.AssertExpectations(t)
}

// --- ExecuteWebhook tests ---

func TestExecuteWebhook_Success_NoWait(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)
	webhookID := uuid.New()
	token := "test-token"

	// Default retry=true uses ExecuteWebhookWithRetry
	mockService.On("ExecuteWebhookWithRetry", mock.Anything, webhookID, token, mock.Anything).Return(&models.Message{
		ID:        uuid.New(),
		Content:   "hello",
		ChannelID: uuid.New(),
		AuthorID:  uuid.New(),
	}, nil)

	body := `{"content":"hello from webhook"}`
	req := httptest.NewRequest(http.MethodPost, "/webhooks/"+webhookID.String()+"/"+token+"?retry=true", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestExecuteWebhook_Success_WithWait(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)
	webhookID := uuid.New()
	token := "test-token"
	msgID := uuid.New()

	// wait=true still uses retry=true by default (uses ExecuteWebhookWithRetry)
	mockService.On("ExecuteWebhookWithRetry", mock.Anything, webhookID, token, mock.Anything).Return(&models.Message{
		ID:        msgID,
		Content:   "hello with wait",
		ChannelID: uuid.New(),
		AuthorID:  uuid.New(),
	}, nil)

	body := `{"content":"hello with wait"}`
	req := httptest.NewRequest(http.MethodPost, "/webhooks/"+webhookID.String()+"/"+token+"?wait=true&retry=true", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, msgID.String(), result["id"])

	mockService.AssertExpectations(t)
}

func TestExecuteWebhook_InvalidWebhookID(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)

	body := `{"content":"hello"}`
	req := httptest.NewRequest(http.MethodPost, "/webhooks/not-a-uuid/some-token", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestExecuteWebhook_InvalidBody(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)
	webhookID := uuid.New()

	body := `{invalid}`
	req := httptest.NewRequest(http.MethodPost, "/webhooks/"+webhookID.String()+"/some-token", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestExecuteWebhook_WebhookNotFound(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)
	webhookID := uuid.New()
	token := "test-token"

	mockService.On("ExecuteWebhookWithRetry", mock.Anything, webhookID, token, mock.Anything).Return(nil, services.ErrWebhookNotFound)

	body := `{"content":"hello"}`
	req := httptest.NewRequest(http.MethodPost, "/webhooks/"+webhookID.String()+"/"+token+"?retry=true", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestExecuteWebhook_InvalidToken(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)
	webhookID := uuid.New()
	token := "bad-token"

	mockService.On("ExecuteWebhookWithRetry", mock.Anything, webhookID, token, mock.Anything).Return(nil, services.ErrInvalidWebhookToken)

	body := `{"content":"hello"}`
	req := httptest.NewRequest(http.MethodPost, "/webhooks/"+webhookID.String()+"/"+token+"?retry=true", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestExecuteWebhook_EmptyMessage(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)
	webhookID := uuid.New()
	token := "test-token"

	mockService.On("ExecuteWebhookWithRetry", mock.Anything, webhookID, token, mock.Anything).Return(nil, services.ErrEmptyMessage)

	body := `{"content":""}`
	req := httptest.NewRequest(http.MethodPost, "/webhooks/"+webhookID.String()+"/"+token+"?retry=true", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestExecuteWebhook_InternalError(t *testing.T) {
	mockService := new(mockWebhookService)
	app, _ := setupWebhookTestApp(t, mockService)
	webhookID := uuid.New()
	token := "test-token"

	mockService.On("ExecuteWebhookWithRetry", mock.Anything, webhookID, token, mock.Anything).Return(nil, assert.AnError)

	body := `{"content":"hello"}`
	req := httptest.NewRequest(http.MethodPost, "/webhooks/"+webhookID.String()+"/"+token+"?retry=true", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	mockService.AssertExpectations(t)
}

// --- webhookToResponse tests ---

func TestWebhookToResponse(t *testing.T) {
	serverID := uuid.New()
	avatar := "https://example.com/avatar.png"

	webhook := &models.Webhook{
		ID:        uuid.New(),
		Name:      "test-hook",
		ChannelID: uuid.New(),
		ServerID:  &serverID,
		Token:     "secret-token",
		Avatar:    &avatar,
		Type:      1,
	}

	resp := webhookToResponse(webhook)
	assert.Equal(t, webhook.ID.String(), resp.ID)
	assert.Equal(t, webhook.Name, resp.Name)
	assert.Equal(t, webhook.ChannelID.String(), resp.ChannelID)
	assert.Equal(t, webhook.ServerID.String(), resp.ServerID)
	assert.Equal(t, webhook.Token, resp.Token)
	assert.NotNil(t, resp.AvatarURL)
	assert.Equal(t, avatar, *resp.AvatarURL)
	assert.Equal(t, 1, resp.Type)
}

func TestWebhookToResponse_NoServerID(t *testing.T) {
	webhook := &models.Webhook{
		ID:        uuid.New(),
		Name:      "test-hook",
		ChannelID: uuid.New(),
		ServerID:  nil,
		Token:     "token",
		Avatar:    nil,
		Type:      0,
	}

	resp := webhookToResponse(webhook)
	assert.Equal(t, "", resp.ServerID)
	assert.Nil(t, resp.AvatarURL)
}

// --- NewWebhookHandlers tests ---

func TestNewWebhookHandlers(t *testing.T) {
	mockService := new(mockWebhookService)
	handlers := NewWebhookHandlers(mockService)
	assert.NotNil(t, handlers)
	assert.Equal(t, mockService, handlers.webhookService)
}
