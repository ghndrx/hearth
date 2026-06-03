package handlers

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"hearth/internal/models"
)

type mockPushService struct {
	registerSubscriptionFunc   func(ctx context.Context, userID uuid.UUID, req *models.CreatePushSubscriptionRequest) error
	unregisterSubscriptionFunc func(ctx context.Context, userID uuid.UUID, endpoint string) error
	getPreferencesFunc         func(ctx context.Context, userID uuid.UUID) (*models.NotificationPreferences, error)
	updatePreferencesFunc      func(ctx context.Context, userID uuid.UUID, req *models.UpdateNotificationPreferencesRequest) (*models.NotificationPreferences, error)
}

func (m *mockPushService) RegisterSubscription(ctx context.Context, userID uuid.UUID, req *models.CreatePushSubscriptionRequest) error {
	if m.registerSubscriptionFunc != nil {
		return m.registerSubscriptionFunc(ctx, userID, req)
	}
	return nil
}

func (m *mockPushService) UnregisterSubscription(ctx context.Context, userID uuid.UUID, endpoint string) error {
	if m.unregisterSubscriptionFunc != nil {
		return m.unregisterSubscriptionFunc(ctx, userID, endpoint)
	}
	return nil
}

func (m *mockPushService) GetPreferences(ctx context.Context, userID uuid.UUID) (*models.NotificationPreferences, error) {
	if m.getPreferencesFunc != nil {
		return m.getPreferencesFunc(ctx, userID)
	}
	return nil, nil
}

func (m *mockPushService) UpdatePreferences(ctx context.Context, userID uuid.UUID, req *models.UpdateNotificationPreferencesRequest) (*models.NotificationPreferences, error) {
	if m.updatePreferencesFunc != nil {
		return m.updatePreferencesFunc(ctx, userID, req)
	}
	return nil, nil
}

func setupPushTestApp(mock *mockPushService) *fiber.App {
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

	handler := &NotificationHandler{pushService: mock}
	app.Post("/push/subscription", handler.RegisterSubscription)
	app.Delete("/push/subscription", handler.UnregisterSubscription)
	app.Get("/push/preferences", handler.GetPushPreferences)
	app.Patch("/push/preferences", handler.UpdatePushPreferences)

	return app
}

func TestRegisterSubscription_Success(t *testing.T) {
	userID := uuid.New()
	calledWith := (*models.CreatePushSubscriptionRequest)(nil)

	mock := &mockPushService{
		registerSubscriptionFunc: func(ctx context.Context, uID uuid.UUID, req *models.CreatePushSubscriptionRequest) error {
			calledWith = req
			return nil
		},
	}

	app := setupPushTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"endpoint":"https://push.example.com/123","p256dh":"BCk","auth":"abc123"}`
	req := httptest.NewRequest("POST", "/push/subscription", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode)

	assert.NotNil(t, calledWith)
	assert.Equal(t, "https://push.example.com/123", calledWith.Endpoint)
	assert.Equal(t, "BCk", calledWith.P256dh)
	assert.Equal(t, "abc123", calledWith.Auth)
}

func TestRegisterSubscription_MissingFields(t *testing.T) {
	userID := uuid.New()
	mock := &mockPushService{}

	app := setupPushTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	// Missing endpoint
	body := `{"p256dh":"BCk","auth":"abc123"}`
	req := httptest.NewRequest("POST", "/push/subscription", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	// Missing p256dh
	body = `{"endpoint":"https://push.example.com/123","auth":"abc123"}`
	req = httptest.NewRequest("POST", "/push/subscription", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err = app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	// Missing auth
	body = `{"endpoint":"https://push.example.com/123","p256dh":"BCk"}`
	req = httptest.NewRequest("POST", "/push/subscription", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err = app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestRegisterSubscription_InvalidBody(t *testing.T) {
	userID := uuid.New()
	mock := &mockPushService{}

	app := setupPushTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `invalid json`
	req := httptest.NewRequest("POST", "/push/subscription", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestUnregisterSubscription_Success(t *testing.T) {
	userID := uuid.New()
	calledWith := ""

	mock := &mockPushService{
		unregisterSubscriptionFunc: func(ctx context.Context, uID uuid.UUID, endpoint string) error {
			calledWith = endpoint
			return nil
		},
	}

	app := setupPushTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"endpoint":"https://push.example.com/123"}`
	req := httptest.NewRequest("DELETE", "/push/subscription", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 204, resp.StatusCode)
	assert.Equal(t, "https://push.example.com/123", calledWith)
}

func TestUnregisterSubscription_MissingEndpoint(t *testing.T) {
	userID := uuid.New()
	mock := &mockPushService{}

	app := setupPushTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{}`
	req := httptest.NewRequest("DELETE", "/push/subscription", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestGetPreferences_Success(t *testing.T) {
	userID := uuid.New()

	mock := &mockPushService{
		getPreferencesFunc: func(ctx context.Context, uID uuid.UUID) (*models.NotificationPreferences, error) {
			return &models.NotificationPreferences{
				UserID:             uID,
				PushEnabled:        true,
				PushMentions:       true,
				PushDirectMessages: true,
				SoundEnabled:       true,
				DesktopEnabled:     true,
				UpdatedAt:          time.Now(),
			}, nil
		},
	}

	app := setupPushTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("GET", "/push/preferences", nil)
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result models.NotificationPreferences
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.True(t, result.PushEnabled)
	assert.True(t, result.PushMentions)
}

func TestGetPreferences_ServiceError(t *testing.T) {
	userID := uuid.New()

	mock := &mockPushService{
		getPreferencesFunc: func(ctx context.Context, uID uuid.UUID) (*models.NotificationPreferences, error) {
			return nil, assert.AnError
		},
	}

	app := setupPushTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("GET", "/push/preferences", nil)
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)
}

func TestUpdatePreferences_Success(t *testing.T) {
	userID := uuid.New()
	calledWithReq := (*models.UpdateNotificationPreferencesRequest)(nil)

	mock := &mockPushService{
		updatePreferencesFunc: func(ctx context.Context, uID uuid.UUID, req *models.UpdateNotificationPreferencesRequest) (*models.NotificationPreferences, error) {
			calledWithReq = req
			return &models.NotificationPreferences{
				UserID:             uID,
				PushEnabled:        true,
				PushMentions:       false,
				PushDirectMessages: true,
				SoundEnabled:       true,
				DesktopEnabled:     true,
				UpdatedAt:          time.Now(),
			}, nil
		},
	}

	app := setupPushTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	pushMentions := false
	body := `{"push_mentions":false}`
	req := httptest.NewRequest("PATCH", "/push/preferences", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	assert.NotNil(t, calledWithReq)
	assert.NotNil(t, calledWithReq.PushMentions)
	assert.False(t, *calledWithReq.PushMentions)

	_ = pushMentions // avoid unused
}

func TestUpdatePreferences_InvalidBody(t *testing.T) {
	userID := uuid.New()
	mock := &mockPushService{}

	app := setupPushTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `invalid json`
	req := httptest.NewRequest("PATCH", "/push/preferences", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}
