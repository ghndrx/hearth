package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hearth/internal/models"
)

type mockPushService struct {
	registerFunc    func(ctx context.Context, userID uuid.UUID, req *models.CreatePushSubscriptionRequest) error
	unregisterFunc  func(ctx context.Context, userID uuid.UUID, endpoint string) error
	getPrefsFunc    func(ctx context.Context, userID uuid.UUID) (*models.NotificationPreferences, error)
	updatePrefsFunc func(ctx context.Context, userID uuid.UUID, req *models.UpdateNotificationPreferencesRequest) (*models.NotificationPreferences, error)
}

func (m *mockPushService) RegisterSubscription(ctx context.Context, userID uuid.UUID, req *models.CreatePushSubscriptionRequest) error {
	if m.registerFunc != nil {
		return m.registerFunc(ctx, userID, req)
	}
	return nil
}

func (m *mockPushService) UnregisterSubscription(ctx context.Context, userID uuid.UUID, endpoint string) error {
	if m.unregisterFunc != nil {
		return m.unregisterFunc(ctx, userID, endpoint)
	}
	return nil
}

func (m *mockPushService) GetPreferences(ctx context.Context, userID uuid.UUID) (*models.NotificationPreferences, error) {
	if m.getPrefsFunc != nil {
		return m.getPrefsFunc(ctx, userID)
	}
	return &models.NotificationPreferences{UserID: userID, PushEnabled: true}, nil
}

func (m *mockPushService) UpdatePreferences(ctx context.Context, userID uuid.UUID, req *models.UpdateNotificationPreferencesRequest) (*models.NotificationPreferences, error) {
	if m.updatePrefsFunc != nil {
		return m.updatePrefsFunc(ctx, userID, req)
	}
	return &models.NotificationPreferences{UserID: userID}, nil
}

func setupPushTestApp(svc *mockPushService) *fiber.App {
	app := fiber.New()
	h := NewPushHandler(svc)

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", uuid.MustParse("11111111-1111-1111-1111-111111111111"))
		return c.Next()
	})

	app.Post("/push/subscribe", h.RegisterSubscription)
	app.Post("/push/unsubscribe", h.UnregisterSubscription)
	app.Get("/push/preferences", h.GetPreferences)
	app.Patch("/push/preferences", h.UpdatePreferences)

	return app
}

func TestPush_RegisterSubscription_Success(t *testing.T) {
	svc := &mockPushService{}
	app := setupPushTestApp(svc)

	body, _ := json.Marshal(map[string]string{
		"endpoint": "https://push.example.com/sub1",
		"p256dh":   "key123",
		"auth":     "auth123",
	})
	req := httptest.NewRequest("POST", "/push/subscribe", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode)
}

func TestPush_RegisterSubscription_MissingFields(t *testing.T) {
	app := setupPushTestApp(&mockPushService{})

	body, _ := json.Marshal(map[string]string{"endpoint": "https://push.example.com/sub1"})
	req := httptest.NewRequest("POST", "/push/subscribe", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestPush_RegisterSubscription_InvalidBody(t *testing.T) {
	app := setupPushTestApp(&mockPushService{})

	req := httptest.NewRequest("POST", "/push/subscribe", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestPush_RegisterSubscription_ServerError(t *testing.T) {
	svc := &mockPushService{
		registerFunc: func(ctx context.Context, userID uuid.UUID, req *models.CreatePushSubscriptionRequest) error {
			return errors.New("db error")
		},
	}
	app := setupPushTestApp(svc)

	body, _ := json.Marshal(map[string]string{
		"endpoint": "https://push.example.com/sub1",
		"p256dh":   "key123",
		"auth":     "auth123",
	})
	req := httptest.NewRequest("POST", "/push/subscribe", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)
}

func TestPush_UnregisterSubscription_Success(t *testing.T) {
	app := setupPushTestApp(&mockPushService{})

	body, _ := json.Marshal(map[string]string{"endpoint": "https://push.example.com/sub1"})
	req := httptest.NewRequest("POST", "/push/unsubscribe", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 204, resp.StatusCode)
}

func TestPush_UnregisterSubscription_MissingEndpoint(t *testing.T) {
	app := setupPushTestApp(&mockPushService{})

	body, _ := json.Marshal(map[string]string{})
	req := httptest.NewRequest("POST", "/push/unsubscribe", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestPush_UnregisterSubscription_InvalidBody(t *testing.T) {
	app := setupPushTestApp(&mockPushService{})

	req := httptest.NewRequest("POST", "/push/unsubscribe", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestPush_UnregisterSubscription_ServerError(t *testing.T) {
	svc := &mockPushService{
		unregisterFunc: func(ctx context.Context, userID uuid.UUID, endpoint string) error {
			return errors.New("db error")
		},
	}
	app := setupPushTestApp(svc)

	body, _ := json.Marshal(map[string]string{"endpoint": "https://push.example.com/sub1"})
	req := httptest.NewRequest("POST", "/push/unsubscribe", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)
}

func TestPush_GetPreferences_Success(t *testing.T) {
	app := setupPushTestApp(&mockPushService{})

	req := httptest.NewRequest("GET", "/push/preferences", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result models.NotificationPreferences
	json.NewDecoder(resp.Body).Decode(&result)
	assert.True(t, result.PushEnabled)
}

func TestPush_GetPreferences_ServerError(t *testing.T) {
	svc := &mockPushService{
		getPrefsFunc: func(ctx context.Context, userID uuid.UUID) (*models.NotificationPreferences, error) {
			return nil, errors.New("db error")
		},
	}
	app := setupPushTestApp(svc)

	req := httptest.NewRequest("GET", "/push/preferences", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)
}

func TestPush_UpdatePreferences_Success(t *testing.T) {
	app := setupPushTestApp(&mockPushService{})

	body, _ := json.Marshal(map[string]interface{}{"push_enabled": false})
	req := httptest.NewRequest("PATCH", "/push/preferences", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestPush_UpdatePreferences_InvalidBody(t *testing.T) {
	app := setupPushTestApp(&mockPushService{})

	req := httptest.NewRequest("PATCH", "/push/preferences", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestPush_UpdatePreferences_ServerError(t *testing.T) {
	svc := &mockPushService{
		updatePrefsFunc: func(ctx context.Context, userID uuid.UUID, req *models.UpdateNotificationPreferencesRequest) (*models.NotificationPreferences, error) {
			return nil, errors.New("db error")
		},
	}
	app := setupPushTestApp(svc)

	body, _ := json.Marshal(map[string]interface{}{"push_enabled": true})
	req := httptest.NewRequest("PATCH", "/push/preferences", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)
}
