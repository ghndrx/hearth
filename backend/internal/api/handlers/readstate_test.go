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
	"hearth/internal/services"
)

type mockReadStateService struct {
	markChannelAsReadFunc    func(ctx context.Context, userID, channelID uuid.UUID, messageID *uuid.UUID) (*models.AckResponse, error)
	getChannelReadStateFunc  func(ctx context.Context, userID, channelID uuid.UUID) (*models.ReadState, error)
	getChannelUnreadInfoFunc func(ctx context.Context, userID, channelID uuid.UUID) (*models.ChannelUnreadInfo, error)
	getUnreadSummaryFunc     func(ctx context.Context, userID uuid.UUID) (*models.UnreadSummary, error)
	getServerUnreadFunc      func(ctx context.Context, userID, serverID uuid.UUID) (*models.UnreadSummary, error)
	markServerAsReadFunc     func(ctx context.Context, userID, serverID uuid.UUID) error
}

func (m *mockReadStateService) MarkChannelAsRead(ctx context.Context, userID, channelID uuid.UUID, messageID *uuid.UUID) (*models.AckResponse, error) {
	if m.markChannelAsReadFunc != nil {
		return m.markChannelAsReadFunc(ctx, userID, channelID, messageID)
	}
	return &models.AckResponse{ChannelID: channelID}, nil
}

func (m *mockReadStateService) GetChannelReadState(ctx context.Context, userID, channelID uuid.UUID) (*models.ReadState, error) {
	if m.getChannelReadStateFunc != nil {
		return m.getChannelReadStateFunc(ctx, userID, channelID)
	}
	return &models.ReadState{}, nil
}

func (m *mockReadStateService) GetChannelUnreadInfo(ctx context.Context, userID, channelID uuid.UUID) (*models.ChannelUnreadInfo, error) {
	if m.getChannelUnreadInfoFunc != nil {
		return m.getChannelUnreadInfoFunc(ctx, userID, channelID)
	}
	return &models.ChannelUnreadInfo{ChannelID: channelID}, nil
}

func (m *mockReadStateService) GetUnreadSummary(ctx context.Context, userID uuid.UUID) (*models.UnreadSummary, error) {
	if m.getUnreadSummaryFunc != nil {
		return m.getUnreadSummaryFunc(ctx, userID)
	}
	return &models.UnreadSummary{}, nil
}

func (m *mockReadStateService) GetServerUnreadSummary(ctx context.Context, userID, serverID uuid.UUID) (*models.UnreadSummary, error) {
	if m.getServerUnreadFunc != nil {
		return m.getServerUnreadFunc(ctx, userID, serverID)
	}
	return &models.UnreadSummary{}, nil
}

func (m *mockReadStateService) MarkServerAsRead(ctx context.Context, userID, serverID uuid.UUID) error {
	if m.markServerAsReadFunc != nil {
		return m.markServerAsReadFunc(ctx, userID, serverID)
	}
	return nil
}

func setupReadStateTestApp(svc *mockReadStateService) *fiber.App {
	app := fiber.New()
	h := &NotificationHandler{readStateService: svc}

	app.Use(func(c *fiber.Ctx) error {
		userIDStr := c.Get("X-Test-User-ID")
		if userIDStr != "" {
			if uid, err := uuid.Parse(userIDStr); err == nil {
				c.Locals("userID", uid)
			}
		} else {
			c.Locals("userID", uuid.MustParse("11111111-1111-1111-1111-111111111111"))
		}
		return c.Next()
	})

	app.Post("/channels/:id/ack", h.MarkChannelAsRead)
	app.Get("/channels/:id/unread", h.GetChannelUnread)
	app.Get("/users/@me/unread", h.GetUnreadSummary)
	app.Get("/servers/:id/unread", h.GetServerUnread)
	app.Post("/servers/:id/ack", h.MarkServerAsRead)

	return app
}

func TestReadState_MarkChannelAsRead_Success(t *testing.T) {
	channelID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	svc := &mockReadStateService{
		markChannelAsReadFunc: func(ctx context.Context, userID, chID uuid.UUID, msgID *uuid.UUID) (*models.AckResponse, error) {
			return &models.AckResponse{ChannelID: chID, MentionCount: 0}, nil
		},
	}
	app := setupReadStateTestApp(svc)

	req := httptest.NewRequest("POST", "/channels/"+channelID.String()+"/ack", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result models.AckResponse
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, channelID, result.ChannelID)
}

func TestReadState_MarkChannelAsRead_InvalidID(t *testing.T) {
	app := setupReadStateTestApp(&mockReadStateService{})

	req := httptest.NewRequest("POST", "/channels/not-a-uuid/ack", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestReadState_MarkChannelAsRead_ChannelNotFound(t *testing.T) {
	svc := &mockReadStateService{
		markChannelAsReadFunc: func(ctx context.Context, userID, chID uuid.UUID, msgID *uuid.UUID) (*models.AckResponse, error) {
			return nil, services.ErrChannelNotFound
		},
	}
	app := setupReadStateTestApp(svc)

	req := httptest.NewRequest("POST", "/channels/"+uuid.New().String()+"/ack", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode)
}

func TestReadState_MarkChannelAsRead_ServerError(t *testing.T) {
	svc := &mockReadStateService{
		markChannelAsReadFunc: func(ctx context.Context, userID, chID uuid.UUID, msgID *uuid.UUID) (*models.AckResponse, error) {
			return nil, errors.New("db error")
		},
	}
	app := setupReadStateTestApp(svc)

	req := httptest.NewRequest("POST", "/channels/"+uuid.New().String()+"/ack", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)
}

func TestReadState_MarkChannelAsRead_WithMessageID(t *testing.T) {
	msgID := uuid.New()
	svc := &mockReadStateService{
		markChannelAsReadFunc: func(ctx context.Context, userID, chID uuid.UUID, mid *uuid.UUID) (*models.AckResponse, error) {
			assert.NotNil(t, mid)
			assert.Equal(t, msgID, *mid)
			return &models.AckResponse{ChannelID: chID, LastMessageID: mid}, nil
		},
	}
	app := setupReadStateTestApp(svc)

	body, _ := json.Marshal(map[string]string{"message_id": msgID.String()})
	req := httptest.NewRequest("POST", "/channels/"+uuid.New().String()+"/ack", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestReadState_GetChannelUnread_Success(t *testing.T) {
	channelID := uuid.New()
	svc := &mockReadStateService{
		getChannelUnreadInfoFunc: func(ctx context.Context, userID, chID uuid.UUID) (*models.ChannelUnreadInfo, error) {
			return &models.ChannelUnreadInfo{ChannelID: chID, UnreadCount: 5, MentionCount: 2, HasUnread: true}, nil
		},
	}
	app := setupReadStateTestApp(svc)

	req := httptest.NewRequest("GET", "/channels/"+channelID.String()+"/unread", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result models.ChannelUnreadInfo
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, 5, result.UnreadCount)
	assert.Equal(t, 2, result.MentionCount)
	assert.True(t, result.HasUnread)
}

func TestReadState_GetChannelUnread_InvalidID(t *testing.T) {
	app := setupReadStateTestApp(&mockReadStateService{})

	req := httptest.NewRequest("GET", "/channels/bad-id/unread", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestReadState_GetChannelUnread_NotFound(t *testing.T) {
	svc := &mockReadStateService{
		getChannelUnreadInfoFunc: func(ctx context.Context, userID, chID uuid.UUID) (*models.ChannelUnreadInfo, error) {
			return nil, services.ErrChannelNotFound
		},
	}
	app := setupReadStateTestApp(svc)

	req := httptest.NewRequest("GET", "/channels/"+uuid.New().String()+"/unread", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode)
}

func TestReadState_GetChannelUnread_ServerError(t *testing.T) {
	svc := &mockReadStateService{
		getChannelUnreadInfoFunc: func(ctx context.Context, userID, chID uuid.UUID) (*models.ChannelUnreadInfo, error) {
			return nil, errors.New("db error")
		},
	}
	app := setupReadStateTestApp(svc)

	req := httptest.NewRequest("GET", "/channels/"+uuid.New().String()+"/unread", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)
}

func TestReadState_GetUnreadSummary_Success(t *testing.T) {
	svc := &mockReadStateService{
		getUnreadSummaryFunc: func(ctx context.Context, userID uuid.UUID) (*models.UnreadSummary, error) {
			return &models.UnreadSummary{TotalUnread: 10, TotalMentions: 3}, nil
		},
	}
	app := setupReadStateTestApp(svc)

	req := httptest.NewRequest("GET", "/users/@me/unread", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result models.UnreadSummary
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, 10, result.TotalUnread)
	assert.Equal(t, 3, result.TotalMentions)
}

func TestReadState_GetUnreadSummary_ServerError(t *testing.T) {
	svc := &mockReadStateService{
		getUnreadSummaryFunc: func(ctx context.Context, userID uuid.UUID) (*models.UnreadSummary, error) {
			return nil, errors.New("db error")
		},
	}
	app := setupReadStateTestApp(svc)

	req := httptest.NewRequest("GET", "/users/@me/unread", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)
}

func TestReadState_GetServerUnread_Success(t *testing.T) {
	serverID := uuid.New()
	svc := &mockReadStateService{
		getServerUnreadFunc: func(ctx context.Context, userID, sID uuid.UUID) (*models.UnreadSummary, error) {
			return &models.UnreadSummary{TotalUnread: 7, TotalMentions: 1}, nil
		},
	}
	app := setupReadStateTestApp(svc)

	req := httptest.NewRequest("GET", "/servers/"+serverID.String()+"/unread", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result models.UnreadSummary
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, 7, result.TotalUnread)
}

func TestReadState_GetServerUnread_InvalidID(t *testing.T) {
	app := setupReadStateTestApp(&mockReadStateService{})

	req := httptest.NewRequest("GET", "/servers/bad-id/unread", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestReadState_GetServerUnread_ServerError(t *testing.T) {
	svc := &mockReadStateService{
		getServerUnreadFunc: func(ctx context.Context, userID, sID uuid.UUID) (*models.UnreadSummary, error) {
			return nil, errors.New("db error")
		},
	}
	app := setupReadStateTestApp(svc)

	req := httptest.NewRequest("GET", "/servers/"+uuid.New().String()+"/unread", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)
}

func TestReadState_MarkServerAsRead_Success(t *testing.T) {
	app := setupReadStateTestApp(&mockReadStateService{})

	req := httptest.NewRequest("POST", "/servers/"+uuid.New().String()+"/ack", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 204, resp.StatusCode)
}

func TestReadState_MarkServerAsRead_InvalidID(t *testing.T) {
	app := setupReadStateTestApp(&mockReadStateService{})

	req := httptest.NewRequest("POST", "/servers/bad-id/ack", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestReadState_MarkServerAsRead_ServerError(t *testing.T) {
	svc := &mockReadStateService{
		markServerAsReadFunc: func(ctx context.Context, userID, sID uuid.UUID) error {
			return errors.New("db error")
		},
	}
	app := setupReadStateTestApp(svc)

	req := httptest.NewRequest("POST", "/servers/"+uuid.New().String()+"/ack", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)
}
