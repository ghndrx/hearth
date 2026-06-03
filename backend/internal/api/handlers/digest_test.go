package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hearth/internal/models"
	"hearth/internal/services"
)

type mockDigestService struct {
	getPrefsFunc          func(ctx context.Context, userID uuid.UUID) (*models.DigestPreferences, error)
	updatePrefsFunc       func(ctx context.Context, userID uuid.UUID, req *models.UpdateDigestPreferencesRequest) (*models.DigestPreferences, error)
	getChannelPrefFunc    func(ctx context.Context, userID, channelID uuid.UUID) (*models.DigestChannelPreference, error)
	getChannelPrefsFunc   func(ctx context.Context, userID uuid.UUID) ([]models.DigestChannelPreference, error)
	updateChannelPrefFunc func(ctx context.Context, userID, channelID uuid.UUID, mode models.DigestMode) error
	getServerPrefFunc     func(ctx context.Context, userID, serverID uuid.UUID) (*models.DigestServerPreference, error)
	getServerPrefsFunc    func(ctx context.Context, userID uuid.UUID) ([]models.DigestServerPreference, error)
	updateServerPrefFunc  func(ctx context.Context, userID, serverID uuid.UUID, mode models.DigestMode) error
	getDigestPreviewFunc  func(ctx context.Context, userID uuid.UUID) (*models.DigestPreview, error)
	clearDigestQueueFunc  func(ctx context.Context, userID uuid.UUID) (int64, error)
	getDigestHistoryFunc  func(ctx context.Context, userID uuid.UUID, opts models.DigestHistoryListOptions) ([]models.DigestHistory, error)
	getDigestByIDFunc     func(ctx context.Context, userID uuid.UUID, digestID uuid.UUID) (*models.DigestHistory, error)
	generateDigestFunc    func(ctx context.Context, userID uuid.UUID) (*models.DigestHistory, error)
}

func (m *mockDigestService) GetPreferences(ctx context.Context, userID uuid.UUID) (*models.DigestPreferences, error) {
	if m.getPrefsFunc != nil {
		return m.getPrefsFunc(ctx, userID)
	}
	return &models.DigestPreferences{UserID: userID, Enabled: true}, nil
}

func (m *mockDigestService) UpdatePreferences(ctx context.Context, userID uuid.UUID, req *models.UpdateDigestPreferencesRequest) (*models.DigestPreferences, error) {
	if m.updatePrefsFunc != nil {
		return m.updatePrefsFunc(ctx, userID, req)
	}
	return &models.DigestPreferences{UserID: userID}, nil
}

func (m *mockDigestService) GetChannelPreference(ctx context.Context, userID, channelID uuid.UUID) (*models.DigestChannelPreference, error) {
	if m.getChannelPrefFunc != nil {
		return m.getChannelPrefFunc(ctx, userID, channelID)
	}
	return &models.DigestChannelPreference{ChannelID: channelID}, nil
}

func (m *mockDigestService) GetChannelPreferences(ctx context.Context, userID uuid.UUID) ([]models.DigestChannelPreference, error) {
	if m.getChannelPrefsFunc != nil {
		return m.getChannelPrefsFunc(ctx, userID)
	}
	return []models.DigestChannelPreference{}, nil
}

func (m *mockDigestService) UpdateChannelPreference(ctx context.Context, userID, channelID uuid.UUID, mode models.DigestMode) error {
	if m.updateChannelPrefFunc != nil {
		return m.updateChannelPrefFunc(ctx, userID, channelID, mode)
	}
	return nil
}

func (m *mockDigestService) GetServerPreference(ctx context.Context, userID, serverID uuid.UUID) (*models.DigestServerPreference, error) {
	if m.getServerPrefFunc != nil {
		return m.getServerPrefFunc(ctx, userID, serverID)
	}
	return &models.DigestServerPreference{ServerID: serverID}, nil
}

func (m *mockDigestService) GetServerPreferences(ctx context.Context, userID uuid.UUID) ([]models.DigestServerPreference, error) {
	if m.getServerPrefsFunc != nil {
		return m.getServerPrefsFunc(ctx, userID)
	}
	return []models.DigestServerPreference{}, nil
}

func (m *mockDigestService) UpdateServerPreference(ctx context.Context, userID, serverID uuid.UUID, mode models.DigestMode) error {
	if m.updateServerPrefFunc != nil {
		return m.updateServerPrefFunc(ctx, userID, serverID, mode)
	}
	return nil
}

func (m *mockDigestService) GetDigestPreview(ctx context.Context, userID uuid.UUID) (*models.DigestPreview, error) {
	if m.getDigestPreviewFunc != nil {
		return m.getDigestPreviewFunc(ctx, userID)
	}
	return &models.DigestPreview{PendingCount: 5}, nil
}

func (m *mockDigestService) ClearDigestQueue(ctx context.Context, userID uuid.UUID) (int64, error) {
	if m.clearDigestQueueFunc != nil {
		return m.clearDigestQueueFunc(ctx, userID)
	}
	return 3, nil
}

func (m *mockDigestService) GetDigestHistory(ctx context.Context, userID uuid.UUID, opts models.DigestHistoryListOptions) ([]models.DigestHistory, error) {
	if m.getDigestHistoryFunc != nil {
		return m.getDigestHistoryFunc(ctx, userID, opts)
	}
	return []models.DigestHistory{}, nil
}

func (m *mockDigestService) GetDigestByID(ctx context.Context, userID uuid.UUID, digestID uuid.UUID) (*models.DigestHistory, error) {
	if m.getDigestByIDFunc != nil {
		return m.getDigestByIDFunc(ctx, userID, digestID)
	}
	return &models.DigestHistory{ID: digestID}, nil
}

func (m *mockDigestService) GenerateDigest(ctx context.Context, userID uuid.UUID) (*models.DigestHistory, error) {
	if m.generateDigestFunc != nil {
		return m.generateDigestFunc(ctx, userID)
	}
	return &models.DigestHistory{ID: uuid.New(), SentAt: time.Now()}, nil
}

func setupDigestTestApp(svc *mockDigestService) *fiber.App {
	app := fiber.New()
	h := &NotificationPreferenceHandler{digestService: svc}

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", uuid.MustParse("11111111-1111-1111-1111-111111111111"))
		return c.Next()
	})

	app.Get("/digest/preferences", h.GetPreferences)
	app.Patch("/digest/preferences", h.UpdatePreferences)
	app.Get("/digest/channels", h.GetChannelPreferences)
	app.Get("/digest/channels/:channelId", h.GetChannelPreference)
	app.Put("/digest/channels/:channelId", h.UpdateChannelPreference)
	app.Get("/digest/servers", h.GetServerPreferences)
	app.Get("/digest/servers/:serverId", h.GetServerPreference)
	app.Put("/digest/servers/:serverId", h.UpdateServerPreference)
	app.Get("/digest/preview", h.GetDigestPreview)
	app.Delete("/digest/queue", h.ClearDigestQueue)
	app.Get("/digest/history", h.GetDigestHistory)
	app.Get("/digest/:digestId", h.GetDigest)
	app.Post("/digest/generate", h.GenerateDigestNow)

	return app
}

func TestDigest_GetPreferences_Success(t *testing.T) {
	app := setupDigestTestApp(&mockDigestService{})

	req := httptest.NewRequest("GET", "/digest/preferences", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestDigest_GetPreferences_ServerError(t *testing.T) {
	svc := &mockDigestService{
		getPrefsFunc: func(ctx context.Context, userID uuid.UUID) (*models.DigestPreferences, error) {
			return nil, errors.New("db error")
		},
	}
	app := setupDigestTestApp(svc)

	req := httptest.NewRequest("GET", "/digest/preferences", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)
}

func TestDigest_UpdatePreferences_Success(t *testing.T) {
	app := setupDigestTestApp(&mockDigestService{})

	body, _ := json.Marshal(map[string]interface{}{"enabled": true})
	req := httptest.NewRequest("PATCH", "/digest/preferences", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestDigest_UpdatePreferences_InvalidBody(t *testing.T) {
	app := setupDigestTestApp(&mockDigestService{})

	req := httptest.NewRequest("PATCH", "/digest/preferences", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestDigest_UpdatePreferences_InvalidFrequency(t *testing.T) {
	svc := &mockDigestService{
		updatePrefsFunc: func(ctx context.Context, userID uuid.UUID, req *models.UpdateDigestPreferencesRequest) (*models.DigestPreferences, error) {
			return nil, services.ErrInvalidFrequency
		},
	}
	app := setupDigestTestApp(svc)

	body, _ := json.Marshal(map[string]interface{}{"frequency": "invalid"})
	req := httptest.NewRequest("PATCH", "/digest/preferences", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestDigest_UpdatePreferences_InvalidTimezone(t *testing.T) {
	svc := &mockDigestService{
		updatePrefsFunc: func(ctx context.Context, userID uuid.UUID, req *models.UpdateDigestPreferencesRequest) (*models.DigestPreferences, error) {
			return nil, services.ErrInvalidTimezone
		},
	}
	app := setupDigestTestApp(svc)

	body, _ := json.Marshal(map[string]interface{}{"timezone": "Invalid/Zone"})
	req := httptest.NewRequest("PATCH", "/digest/preferences", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestDigest_UpdatePreferences_ServerError(t *testing.T) {
	svc := &mockDigestService{
		updatePrefsFunc: func(ctx context.Context, userID uuid.UUID, req *models.UpdateDigestPreferencesRequest) (*models.DigestPreferences, error) {
			return nil, errors.New("db error")
		},
	}
	app := setupDigestTestApp(svc)

	body, _ := json.Marshal(map[string]interface{}{"enabled": true})
	req := httptest.NewRequest("PATCH", "/digest/preferences", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)
}

func TestDigest_GetChannelPreferences_Success(t *testing.T) {
	app := setupDigestTestApp(&mockDigestService{})

	req := httptest.NewRequest("GET", "/digest/channels", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestDigest_GetChannelPreferences_ServerError(t *testing.T) {
	svc := &mockDigestService{
		getChannelPrefsFunc: func(ctx context.Context, userID uuid.UUID) ([]models.DigestChannelPreference, error) {
			return nil, errors.New("db error")
		},
	}
	app := setupDigestTestApp(svc)

	req := httptest.NewRequest("GET", "/digest/channels", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)
}

func TestDigest_GetChannelPreference_Success(t *testing.T) {
	app := setupDigestTestApp(&mockDigestService{})

	req := httptest.NewRequest("GET", "/digest/channels/"+uuid.New().String(), nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestDigest_GetChannelPreference_InvalidID(t *testing.T) {
	app := setupDigestTestApp(&mockDigestService{})

	req := httptest.NewRequest("GET", "/digest/channels/bad-id", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestDigest_GetChannelPreference_ServerError(t *testing.T) {
	svc := &mockDigestService{
		getChannelPrefFunc: func(ctx context.Context, userID, channelID uuid.UUID) (*models.DigestChannelPreference, error) {
			return nil, errors.New("db error")
		},
	}
	app := setupDigestTestApp(svc)

	req := httptest.NewRequest("GET", "/digest/channels/"+uuid.New().String(), nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)
}

func TestDigest_UpdateChannelPreference_Success(t *testing.T) {
	app := setupDigestTestApp(&mockDigestService{})

	body, _ := json.Marshal(map[string]string{"digest_mode": "include"})
	req := httptest.NewRequest("PUT", "/digest/channels/"+uuid.New().String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 204, resp.StatusCode)
}

func TestDigest_UpdateChannelPreference_InvalidID(t *testing.T) {
	app := setupDigestTestApp(&mockDigestService{})

	body, _ := json.Marshal(map[string]string{"digest_mode": "include"})
	req := httptest.NewRequest("PUT", "/digest/channels/bad-id", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestDigest_UpdateChannelPreference_InvalidBody(t *testing.T) {
	app := setupDigestTestApp(&mockDigestService{})

	req := httptest.NewRequest("PUT", "/digest/channels/"+uuid.New().String(), bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestDigest_UpdateChannelPreference_ServiceError(t *testing.T) {
	svc := &mockDigestService{
		updateChannelPrefFunc: func(ctx context.Context, userID, channelID uuid.UUID, mode models.DigestMode) error {
			return errors.New("invalid mode")
		},
	}
	app := setupDigestTestApp(svc)

	body, _ := json.Marshal(map[string]string{"digest_mode": "bad"})
	req := httptest.NewRequest("PUT", "/digest/channels/"+uuid.New().String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestDigest_GetServerPreferences_Success(t *testing.T) {
	app := setupDigestTestApp(&mockDigestService{})

	req := httptest.NewRequest("GET", "/digest/servers", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestDigest_GetServerPreferences_ServerError(t *testing.T) {
	svc := &mockDigestService{
		getServerPrefsFunc: func(ctx context.Context, userID uuid.UUID) ([]models.DigestServerPreference, error) {
			return nil, errors.New("db error")
		},
	}
	app := setupDigestTestApp(svc)

	req := httptest.NewRequest("GET", "/digest/servers", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)
}

func TestDigest_GetServerPreference_Success(t *testing.T) {
	app := setupDigestTestApp(&mockDigestService{})

	req := httptest.NewRequest("GET", "/digest/servers/"+uuid.New().String(), nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestDigest_GetServerPreference_InvalidID(t *testing.T) {
	app := setupDigestTestApp(&mockDigestService{})

	req := httptest.NewRequest("GET", "/digest/servers/bad-id", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestDigest_GetServerPreference_ServerError(t *testing.T) {
	svc := &mockDigestService{
		getServerPrefFunc: func(ctx context.Context, userID, serverID uuid.UUID) (*models.DigestServerPreference, error) {
			return nil, errors.New("db error")
		},
	}
	app := setupDigestTestApp(svc)

	req := httptest.NewRequest("GET", "/digest/servers/"+uuid.New().String(), nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)
}

func TestDigest_UpdateServerPreference_Success(t *testing.T) {
	app := setupDigestTestApp(&mockDigestService{})

	body, _ := json.Marshal(map[string]string{"digest_mode": "exclude"})
	req := httptest.NewRequest("PUT", "/digest/servers/"+uuid.New().String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 204, resp.StatusCode)
}

func TestDigest_UpdateServerPreference_InvalidID(t *testing.T) {
	app := setupDigestTestApp(&mockDigestService{})

	body, _ := json.Marshal(map[string]string{"digest_mode": "exclude"})
	req := httptest.NewRequest("PUT", "/digest/servers/bad-id", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestDigest_UpdateServerPreference_InvalidBody(t *testing.T) {
	app := setupDigestTestApp(&mockDigestService{})

	req := httptest.NewRequest("PUT", "/digest/servers/"+uuid.New().String(), bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestDigest_UpdateServerPreference_ServiceError(t *testing.T) {
	svc := &mockDigestService{
		updateServerPrefFunc: func(ctx context.Context, userID, serverID uuid.UUID, mode models.DigestMode) error {
			return errors.New("invalid mode")
		},
	}
	app := setupDigestTestApp(svc)

	body, _ := json.Marshal(map[string]string{"digest_mode": "bad"})
	req := httptest.NewRequest("PUT", "/digest/servers/"+uuid.New().String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestDigest_GetPreview_Success(t *testing.T) {
	app := setupDigestTestApp(&mockDigestService{})

	req := httptest.NewRequest("GET", "/digest/preview", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result models.DigestPreview
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, 5, result.PendingCount)
}

func TestDigest_GetPreview_ServerError(t *testing.T) {
	svc := &mockDigestService{
		getDigestPreviewFunc: func(ctx context.Context, userID uuid.UUID) (*models.DigestPreview, error) {
			return nil, errors.New("db error")
		},
	}
	app := setupDigestTestApp(svc)

	req := httptest.NewRequest("GET", "/digest/preview", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)
}

func TestDigest_ClearQueue_Success(t *testing.T) {
	app := setupDigestTestApp(&mockDigestService{})

	req := httptest.NewRequest("DELETE", "/digest/queue", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, float64(3), result["cleared"])
}

func TestDigest_ClearQueue_ServerError(t *testing.T) {
	svc := &mockDigestService{
		clearDigestQueueFunc: func(ctx context.Context, userID uuid.UUID) (int64, error) {
			return 0, errors.New("db error")
		},
	}
	app := setupDigestTestApp(svc)

	req := httptest.NewRequest("DELETE", "/digest/queue", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)
}

func TestDigest_GetHistory_Success(t *testing.T) {
	app := setupDigestTestApp(&mockDigestService{})

	req := httptest.NewRequest("GET", "/digest/history?limit=10&offset=5", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestDigest_GetHistory_ServerError(t *testing.T) {
	svc := &mockDigestService{
		getDigestHistoryFunc: func(ctx context.Context, userID uuid.UUID, opts models.DigestHistoryListOptions) ([]models.DigestHistory, error) {
			return nil, errors.New("db error")
		},
	}
	app := setupDigestTestApp(svc)

	req := httptest.NewRequest("GET", "/digest/history", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)
}

func TestDigest_GetDigest_Success(t *testing.T) {
	digestID := uuid.New()
	app := setupDigestTestApp(&mockDigestService{})

	req := httptest.NewRequest("GET", "/digest/"+digestID.String(), nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestDigest_GetDigest_InvalidID(t *testing.T) {
	app := setupDigestTestApp(&mockDigestService{})

	req := httptest.NewRequest("GET", "/digest/bad-id", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestDigest_GetDigest_NotFound(t *testing.T) {
	svc := &mockDigestService{
		getDigestByIDFunc: func(ctx context.Context, userID uuid.UUID, digestID uuid.UUID) (*models.DigestHistory, error) {
			return nil, services.ErrDigestNotFound
		},
	}
	app := setupDigestTestApp(svc)

	req := httptest.NewRequest("GET", "/digest/"+uuid.New().String(), nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode)
}

func TestDigest_GetDigest_ServerError(t *testing.T) {
	svc := &mockDigestService{
		getDigestByIDFunc: func(ctx context.Context, userID uuid.UUID, digestID uuid.UUID) (*models.DigestHistory, error) {
			return nil, errors.New("db error")
		},
	}
	app := setupDigestTestApp(svc)

	req := httptest.NewRequest("GET", "/digest/"+uuid.New().String(), nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)
}

func TestDigest_GenerateNow_Success(t *testing.T) {
	app := setupDigestTestApp(&mockDigestService{})

	req := httptest.NewRequest("POST", "/digest/generate", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestDigest_GenerateNow_Disabled(t *testing.T) {
	svc := &mockDigestService{
		generateDigestFunc: func(ctx context.Context, userID uuid.UUID) (*models.DigestHistory, error) {
			return nil, services.ErrDigestDisabled
		},
	}
	app := setupDigestTestApp(svc)

	req := httptest.NewRequest("POST", "/digest/generate", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestDigest_GenerateNow_ServerError(t *testing.T) {
	svc := &mockDigestService{
		generateDigestFunc: func(ctx context.Context, userID uuid.UUID) (*models.DigestHistory, error) {
			return nil, errors.New("db error")
		},
	}
	app := setupDigestTestApp(svc)

	req := httptest.NewRequest("POST", "/digest/generate", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)
}
