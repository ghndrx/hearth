package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
)

// ====================================================================================================================================================
// Tests for ChannelMuteHandler methods
// These tests exercise the handler methods for proper code coverage
// ====================================================================================================================================================

// --- mockChannelMuteService implements ChannelMuteServiceInterface for coverage tests ---

type mockChannelMuteService struct {
	setChannelMutedFunc    func(ctx context.Context, userID, channelID uuid.UUID, muted bool) (*models.UserChannelSettings, error)
	isChannelMutedFunc     func(ctx context.Context, userID, channelID uuid.UUID) (bool, error)
	getMutedChannelIDsFunc func(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}

func (m *mockChannelMuteService) SetChannelMuted(ctx context.Context, userID, channelID uuid.UUID, muted bool) (*models.UserChannelSettings, error) {
	if m.setChannelMutedFunc != nil {
		return m.setChannelMutedFunc(ctx, userID, channelID, muted)
	}
	return &models.UserChannelSettings{
		UserID:    userID,
		ChannelID: channelID,
		Muted:     muted,
	}, nil
}

func (m *mockChannelMuteService) IsChannelMuted(ctx context.Context, userID, channelID uuid.UUID) (bool, error) {
	if m.isChannelMutedFunc != nil {
		return m.isChannelMutedFunc(ctx, userID, channelID)
	}
	return false, nil
}

func (m *mockChannelMuteService) GetMutedChannelIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	if m.getMutedChannelIDsFunc != nil {
		return m.getMutedChannelIDsFunc(ctx, userID)
	}
	return []uuid.UUID{}, nil
}

// ====================================================================================================================================================
// TestChannelMuteHandler_SetChannelMute
// ====================================================================================================================================================

func TestChannelMuteHandler_SetChannelMute(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	tests := []struct {
		name           string
		channelIDParam string
		requestBody    interface{}
		setupMock      func(*mockChannelMuteService)
		setupUser      func(*fiber.Ctx)
		expectedStatus int
	}{
		{
			name:           "success - mute channel",
			channelIDParam: channelID.String(),
			requestBody:    models.UpdateChannelMuteRequest{Muted: true},
			setupMock: func(m *mockChannelMuteService) {
				m.setChannelMutedFunc = func(ctx context.Context, uID, cID uuid.UUID, muted bool) (*models.UserChannelSettings, error) {
					return &models.UserChannelSettings{
						UserID:    uID,
						ChannelID: cID,
						Muted:     muted,
					}, nil
				}
			},
			setupUser: func(c *fiber.Ctx) {
				c.Locals("userID", userID)
			},
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "success - unmute channel",
			channelIDParam: channelID.String(),
			requestBody:    models.UpdateChannelMuteRequest{Muted: false},
			setupMock: func(m *mockChannelMuteService) {
				m.setChannelMutedFunc = func(ctx context.Context, uID, cID uuid.UUID, muted bool) (*models.UserChannelSettings, error) {
					return &models.UserChannelSettings{
						UserID:    uID,
						ChannelID: cID,
						Muted:     muted,
					}, nil
				}
			},
			setupUser: func(c *fiber.Ctx) {
				c.Locals("userID", userID)
			},
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "invalid channel id",
			channelIDParam: "not-a-uuid",
			requestBody:    models.UpdateChannelMuteRequest{Muted: true},
			setupMock:      func(m *mockChannelMuteService) {},
			setupUser: func(c *fiber.Ctx) {
				c.Locals("userID", userID)
			},
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:           "invalid request body",
			channelIDParam: channelID.String(),
			requestBody:    "invalid json",
			setupMock:      func(m *mockChannelMuteService) {},
			setupUser: func(c *fiber.Ctx) {
				c.Locals("userID", userID)
			},
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:           "service error",
			channelIDParam: channelID.String(),
			requestBody:    models.UpdateChannelMuteRequest{Muted: true},
			setupMock: func(m *mockChannelMuteService) {
				m.setChannelMutedFunc = func(ctx context.Context, uID, cID uuid.UUID, muted bool) (*models.UserChannelSettings, error) {
					return nil, context.DeadlineExceeded
				}
			},
			setupUser: func(c *fiber.Ctx) {
				c.Locals("userID", userID)
			},
			expectedStatus: fiber.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockChannelMuteService{}
			tt.setupMock(svc)
			h := NewChannelMuteHandler(svc)

			app := fiber.New()
			app.Put("/users/@me/channels/:id/mute", func(c *fiber.Ctx) error {
				tt.setupUser(c)
				return h.SetChannelMute(c)
			})

			var req *http.Request
			switch v := tt.requestBody.(type) {
			case string:
				req = httptest.NewRequest("PUT", "/users/@me/channels/"+tt.channelIDParam+"/mute", bytes.NewBufferString(v))
				req.Header.Set("Content-Type", "application/json")
			default:
				bodyBytes, _ := json.Marshal(tt.requestBody)
				req = httptest.NewRequest("PUT", "/users/@me/channels/"+tt.channelIDParam+"/mute", bytes.NewBuffer(bodyBytes))
				req.Header.Set("Content-Type", "application/json")
			}

			resp, _ := app.Test(req)
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

// ====================================================================================================================================================
// TestChannelMuteHandler_GetChannelMuteState
// ====================================================================================================================================================

func TestChannelMuteHandler_GetChannelMuteState(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	tests := []struct {
		name           string
		channelIDParam string
		setupMock      func(*mockChannelMuteService)
		setupUser      func(*fiber.Ctx)
		expectedStatus int
	}{
		{
			name:           "success - channel is muted",
			channelIDParam: channelID.String(),
			setupMock: func(m *mockChannelMuteService) {
				m.isChannelMutedFunc = func(ctx context.Context, uID, cID uuid.UUID) (bool, error) {
					return true, nil
				}
			},
			setupUser: func(c *fiber.Ctx) {
				c.Locals("userID", userID)
			},
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "success - channel is not muted",
			channelIDParam: channelID.String(),
			setupMock: func(m *mockChannelMuteService) {
				m.isChannelMutedFunc = func(ctx context.Context, uID, cID uuid.UUID) (bool, error) {
					return false, nil
				}
			},
			setupUser: func(c *fiber.Ctx) {
				c.Locals("userID", userID)
			},
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "invalid channel id",
			channelIDParam: "not-a-uuid",
			setupMock:      func(m *mockChannelMuteService) {},
			setupUser: func(c *fiber.Ctx) {
				c.Locals("userID", userID)
			},
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:           "service error",
			channelIDParam: channelID.String(),
			setupMock: func(m *mockChannelMuteService) {
				m.isChannelMutedFunc = func(ctx context.Context, uID, cID uuid.UUID) (bool, error) {
					return false, context.DeadlineExceeded
				}
			},
			setupUser: func(c *fiber.Ctx) {
				c.Locals("userID", userID)
			},
			expectedStatus: fiber.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockChannelMuteService{}
			tt.setupMock(svc)
			h := NewChannelMuteHandler(svc)

			app := fiber.New()
			app.Get("/users/@me/channels/:id/mute", func(c *fiber.Ctx) error {
				tt.setupUser(c)
				return h.GetChannelMuteState(c)
			})

			req := httptest.NewRequest("GET", "/users/@me/channels/"+tt.channelIDParam+"/mute", nil)
			resp, _ := app.Test(req)
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

// ====================================================================================================================================================
// TestChannelMuteHandler_GetMutedChannels
// ====================================================================================================================================================

func TestChannelMuteHandler_GetMutedChannels(t *testing.T) {
	userID := uuid.New()
	channelID1 := uuid.New()
	channelID2 := uuid.New()

	tests := []struct {
		name           string
		setupMock      func(*mockChannelMuteService)
		setupUser      func(*fiber.Ctx)
		expectedStatus int
	}{
		{
			name: "success - returns muted channels",
			setupMock: func(m *mockChannelMuteService) {
				m.getMutedChannelIDsFunc = func(ctx context.Context, uID uuid.UUID) ([]uuid.UUID, error) {
					return []uuid.UUID{channelID1, channelID2}, nil
				}
			},
			setupUser: func(c *fiber.Ctx) {
				c.Locals("userID", userID)
			},
			expectedStatus: fiber.StatusOK,
		},
		{
			name: "success - no muted channels",
			setupMock: func(m *mockChannelMuteService) {
				m.getMutedChannelIDsFunc = func(ctx context.Context, uID uuid.UUID) ([]uuid.UUID, error) {
					return []uuid.UUID{}, nil
				}
			},
			setupUser: func(c *fiber.Ctx) {
				c.Locals("userID", userID)
			},
			expectedStatus: fiber.StatusOK,
		},
		{
			name: "success - nil return treated as empty",
			setupMock: func(m *mockChannelMuteService) {
				m.getMutedChannelIDsFunc = func(ctx context.Context, uID uuid.UUID) ([]uuid.UUID, error) {
					return nil, nil
				}
			},
			setupUser: func(c *fiber.Ctx) {
				c.Locals("userID", userID)
			},
			expectedStatus: fiber.StatusOK,
		},
		{
			name: "service error",
			setupMock: func(m *mockChannelMuteService) {
				m.getMutedChannelIDsFunc = func(ctx context.Context, uID uuid.UUID) ([]uuid.UUID, error) {
					return nil, context.DeadlineExceeded
				}
			},
			setupUser: func(c *fiber.Ctx) {
				c.Locals("userID", userID)
			},
			expectedStatus: fiber.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockChannelMuteService{}
			tt.setupMock(svc)
			h := NewChannelMuteHandler(svc)

			app := fiber.New()
			app.Get("/users/@me/channels/muted", func(c *fiber.Ctx) error {
				tt.setupUser(c)
				return h.GetMutedChannels(c)
			})

			req := httptest.NewRequest("GET", "/users/@me/channels/muted", nil)
			resp, _ := app.Test(req)
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}
