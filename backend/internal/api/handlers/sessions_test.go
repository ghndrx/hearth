package handlers

import (
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

type mockSessionService struct {
	getUserSessionsFunc   func(ctx context.Context, userID uuid.UUID, currentSessionID *uuid.UUID) ([]*models.SessionResponse, error)
	revokeSessionFunc     func(ctx context.Context, userID, sessionID uuid.UUID) error
	revokeAllSessionsFunc func(ctx context.Context, userID uuid.UUID, exceptSessionID *uuid.UUID) error
	createSessionFunc     func(ctx context.Context, userID uuid.UUID, userAgent, ipAddress string) (*models.Session, string, error)
	updateActivityFunc    func(ctx context.Context, sessionID uuid.UUID) error
	createRefreshFunc     func(ctx context.Context, userID, sessionID, familyID uuid.UUID, expiry time.Duration) (string, error)
	rotateRefreshFunc     func(ctx context.Context, oldToken string, expiry time.Duration) (*models.Session, string, error)
	validateRefreshFunc   func(ctx context.Context, token string) (*models.RefreshToken, error)
}

func (m *mockSessionService) GetUserSessions(ctx context.Context, userID uuid.UUID, currentSessionID *uuid.UUID) ([]*models.SessionResponse, error) {
	if m.getUserSessionsFunc != nil {
		return m.getUserSessionsFunc(ctx, userID, currentSessionID)
	}
	return []*models.SessionResponse{}, nil
}

func (m *mockSessionService) RevokeSession(ctx context.Context, userID, sessionID uuid.UUID) error {
	if m.revokeSessionFunc != nil {
		return m.revokeSessionFunc(ctx, userID, sessionID)
	}
	return nil
}

func (m *mockSessionService) RevokeAllSessions(ctx context.Context, userID uuid.UUID, exceptSessionID *uuid.UUID) error {
	if m.revokeAllSessionsFunc != nil {
		return m.revokeAllSessionsFunc(ctx, userID, exceptSessionID)
	}
	return nil
}

func (m *mockSessionService) CreateSession(ctx context.Context, userID uuid.UUID, userAgent, ipAddress string) (*models.Session, string, error) {
	if m.createSessionFunc != nil {
		return m.createSessionFunc(ctx, userID, userAgent, ipAddress)
	}
	return nil, "", nil
}

func (m *mockSessionService) UpdateSessionActivity(ctx context.Context, sessionID uuid.UUID) error {
	if m.updateActivityFunc != nil {
		return m.updateActivityFunc(ctx, sessionID)
	}
	return nil
}

func (m *mockSessionService) CreateRefreshToken(ctx context.Context, userID, sessionID, familyID uuid.UUID, expiry time.Duration) (string, error) {
	if m.createRefreshFunc != nil {
		return m.createRefreshFunc(ctx, userID, sessionID, familyID, expiry)
	}
	return "", nil
}

func (m *mockSessionService) RotateRefreshToken(ctx context.Context, oldToken string, expiry time.Duration) (*models.Session, string, error) {
	if m.rotateRefreshFunc != nil {
		return m.rotateRefreshFunc(ctx, oldToken, expiry)
	}
	return nil, "", nil
}

func (m *mockSessionService) ValidateRefreshToken(ctx context.Context, token string) (*models.RefreshToken, error) {
	if m.validateRefreshFunc != nil {
		return m.validateRefreshFunc(ctx, token)
	}
	return nil, nil
}

func setupSessionTestApp(svc *mockSessionService) *fiber.App {
	app := fiber.New()
	h := NewSessionHandler(svc)

	app.Use(func(c *fiber.Ctx) error {
		userIDStr := c.Get("X-Test-User-ID")
		if userIDStr != "" {
			if uid, err := uuid.Parse(userIDStr); err == nil {
				c.Locals("userID", uid)
			}
		}
		if sessionIDStr := c.Get("X-Test-Session-ID"); sessionIDStr != "" {
			c.Locals("sessionID", sessionIDStr)
		}
		return c.Next()
	})

	app.Get("/auth/sessions", h.GetSessions)
	app.Delete("/auth/sessions/:id", h.RevokeSession)
	app.Delete("/auth/sessions", h.RevokeAllSessions)

	return app
}

func TestSessions_GetSessions_Success(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	sessionID := uuid.New()
	svc := &mockSessionService{
		getUserSessionsFunc: func(ctx context.Context, uid uuid.UUID, currentSID *uuid.UUID) ([]*models.SessionResponse, error) {
			assert.Equal(t, userID, uid)
			return []*models.SessionResponse{
				{ID: sessionID, DeviceName: "Chrome", IsCurrent: true},
			}, nil
		},
	}
	app := setupSessionTestApp(svc)

	req := httptest.NewRequest("GET", "/auth/sessions", nil)
	req.Header.Set("X-Test-User-ID", userID.String())
	req.Header.Set("X-Test-Session-ID", sessionID.String())
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	sessions := result["sessions"].([]interface{})
	assert.Len(t, sessions, 1)
}

func TestSessions_GetSessions_NoUserID(t *testing.T) {
	app := setupSessionTestApp(&mockSessionService{})

	req := httptest.NewRequest("GET", "/auth/sessions", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 401, resp.StatusCode)
}

func TestSessions_GetSessions_ServerError(t *testing.T) {
	svc := &mockSessionService{
		getUserSessionsFunc: func(ctx context.Context, uid uuid.UUID, currentSID *uuid.UUID) ([]*models.SessionResponse, error) {
			return nil, errors.New("db error")
		},
	}
	app := setupSessionTestApp(svc)

	req := httptest.NewRequest("GET", "/auth/sessions", nil)
	req.Header.Set("X-Test-User-ID", uuid.New().String())
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)
}

func TestSessions_RevokeSession_Success(t *testing.T) {
	userID := uuid.New()
	targetSessionID := uuid.New()

	svc := &mockSessionService{
		revokeSessionFunc: func(ctx context.Context, uid, sid uuid.UUID) error {
			assert.Equal(t, userID, uid)
			assert.Equal(t, targetSessionID, sid)
			return nil
		},
	}
	app := setupSessionTestApp(svc)

	req := httptest.NewRequest("DELETE", "/auth/sessions/"+targetSessionID.String(), nil)
	req.Header.Set("X-Test-User-ID", userID.String())
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 204, resp.StatusCode)
}

func TestSessions_RevokeSession_NoUserID(t *testing.T) {
	app := setupSessionTestApp(&mockSessionService{})

	req := httptest.NewRequest("DELETE", "/auth/sessions/"+uuid.New().String(), nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 401, resp.StatusCode)
}

func TestSessions_RevokeSession_InvalidID(t *testing.T) {
	app := setupSessionTestApp(&mockSessionService{})

	req := httptest.NewRequest("DELETE", "/auth/sessions/not-a-uuid", nil)
	req.Header.Set("X-Test-User-ID", uuid.New().String())
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestSessions_RevokeSession_CannotRevokeCurrent(t *testing.T) {
	sessionID := uuid.New()
	app := setupSessionTestApp(&mockSessionService{})

	req := httptest.NewRequest("DELETE", "/auth/sessions/"+sessionID.String(), nil)
	req.Header.Set("X-Test-User-ID", uuid.New().String())
	req.Header.Set("X-Test-Session-ID", sessionID.String())
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Contains(t, result["message"], "Cannot revoke current session")
}

func TestSessions_RevokeSession_NotFound(t *testing.T) {
	svc := &mockSessionService{
		revokeSessionFunc: func(ctx context.Context, uid, sid uuid.UUID) error {
			return services.ErrSessionNotFound
		},
	}
	app := setupSessionTestApp(svc)

	req := httptest.NewRequest("DELETE", "/auth/sessions/"+uuid.New().String(), nil)
	req.Header.Set("X-Test-User-ID", uuid.New().String())
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode)
}

func TestSessions_RevokeSession_Unauthorized(t *testing.T) {
	svc := &mockSessionService{
		revokeSessionFunc: func(ctx context.Context, uid, sid uuid.UUID) error {
			return services.ErrUnauthorized
		},
	}
	app := setupSessionTestApp(svc)

	req := httptest.NewRequest("DELETE", "/auth/sessions/"+uuid.New().String(), nil)
	req.Header.Set("X-Test-User-ID", uuid.New().String())
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 403, resp.StatusCode)
}

func TestSessions_RevokeSession_ServerError(t *testing.T) {
	svc := &mockSessionService{
		revokeSessionFunc: func(ctx context.Context, uid, sid uuid.UUID) error {
			return errors.New("db error")
		},
	}
	app := setupSessionTestApp(svc)

	req := httptest.NewRequest("DELETE", "/auth/sessions/"+uuid.New().String(), nil)
	req.Header.Set("X-Test-User-ID", uuid.New().String())
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)
}

func TestSessions_RevokeAllSessions_Success(t *testing.T) {
	userID := uuid.New()
	app := setupSessionTestApp(&mockSessionService{})

	req := httptest.NewRequest("DELETE", "/auth/sessions", nil)
	req.Header.Set("X-Test-User-ID", userID.String())
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Contains(t, result["message"], "All other sessions have been revoked")
}

func TestSessions_RevokeAllSessions_NoUserID(t *testing.T) {
	app := setupSessionTestApp(&mockSessionService{})

	req := httptest.NewRequest("DELETE", "/auth/sessions", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 401, resp.StatusCode)
}

func TestSessions_RevokeAllSessions_ServerError(t *testing.T) {
	svc := &mockSessionService{
		revokeAllSessionsFunc: func(ctx context.Context, uid uuid.UUID, exceptSID *uuid.UUID) error {
			return errors.New("db error")
		},
	}
	app := setupSessionTestApp(svc)

	req := httptest.NewRequest("DELETE", "/auth/sessions", nil)
	req.Header.Set("X-Test-User-ID", uuid.New().String())
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)
}
