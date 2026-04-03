package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// ====================================================================================================================================================
// Tests for actual ThreadHandler methods (via NewThreadHandler)
// These tests exercise the real handler methods for proper code coverage
// ====================================================================================================================================================

// --- mockThreadServiceForCoverage implements ThreadServiceInterface for coverage tests ---

type mockThreadServiceForCoverage struct {
	createThreadFunc              func(ctx context.Context, channelID, creatorID uuid.UUID, name string, autoArchive *int, tagIDs []uuid.UUID) (*models.Thread, error)
	updateThreadFunc              func(ctx context.Context, threadID, requesterID uuid.UUID, req models.UpdateThreadRequest) (*models.Thread, error)
	getThreadFunc                 func(ctx context.Context, threadID uuid.UUID) (*models.Thread, error)
	getThreadMessagesFunc         func(ctx context.Context, threadID, requesterID uuid.UUID, before *uuid.UUID, limit int) ([]*models.ThreadMessage, error)
	sendThreadMessageFunc         func(ctx context.Context, threadID, authorID uuid.UUID, content string) (*models.ThreadMessage, error)
	archiveThreadFunc             func(ctx context.Context, threadID, requesterID uuid.UUID) error
	unarchiveThreadFunc           func(ctx context.Context, threadID, requesterID uuid.UUID) error
	getChannelThreadsFunc         func(ctx context.Context, channelID, requesterID uuid.UUID, includeArchived bool) ([]*models.Thread, error)
	joinThreadFunc                func(ctx context.Context, threadID, userID uuid.UUID) error
	leaveThreadFunc               func(ctx context.Context, threadID, userID uuid.UUID) error
	deleteThreadFunc              func(ctx context.Context, threadID, requesterID uuid.UUID) error
	getNotificationPreferenceFunc func(ctx context.Context, threadID, userID uuid.UUID) (*models.ThreadNotificationPreference, error)
	setNotificationPreferenceFunc func(ctx context.Context, threadID, userID uuid.UUID, level models.ThreadNotificationLevel) error
	enterThreadFunc               func(ctx context.Context, threadID, userID uuid.UUID) (*models.ThreadPresenceResponse, error)
	exitThreadFunc                func(ctx context.Context, threadID, userID uuid.UUID) error
	getActiveViewersFunc          func(ctx context.Context, threadID uuid.UUID) (*models.ThreadPresenceResponse, error)
	heartbeatPresenceFunc         func(ctx context.Context, threadID, userID uuid.UUID) error
}

func (m *mockThreadServiceForCoverage) CreateThread(ctx context.Context, channelID, creatorID uuid.UUID, name string, autoArchive *int, parentMessageID *uuid.UUID, tagIDs []uuid.UUID) (*models.Thread, error) {
	if m.createThreadFunc != nil {
		return m.createThreadFunc(ctx, channelID, creatorID, name, autoArchive, tagIDs)
	}
	return &models.Thread{ID: uuid.New(), ParentChannelID: channelID, OwnerID: creatorID, Name: name, AppliedTags: tagIDs}, nil
}

func (m *mockThreadServiceForCoverage) UpdateThread(ctx context.Context, threadID, requesterID uuid.UUID, req models.UpdateThreadRequest) (*models.Thread, error) {
	if m.updateThreadFunc != nil {
		return m.updateThreadFunc(ctx, threadID, requesterID, req)
	}
	name := "Test Thread"
	if req.Name != nil {
		name = *req.Name
	}
	return &models.Thread{ID: threadID, Name: name}, nil
}

func (m *mockThreadServiceForCoverage) GetThread(ctx context.Context, threadID uuid.UUID) (*models.Thread, error) {
	if m.getThreadFunc != nil {
		return m.getThreadFunc(ctx, threadID)
	}
	return &models.Thread{ID: threadID, Name: "Test Thread"}, nil
}

func (m *mockThreadServiceForCoverage) GetThreadMessages(ctx context.Context, threadID, requesterID uuid.UUID, before *uuid.UUID, limit int) ([]*models.ThreadMessage, error) {
	if m.getThreadMessagesFunc != nil {
		return m.getThreadMessagesFunc(ctx, threadID, requesterID, before, limit)
	}
	return []*models.ThreadMessage{}, nil
}

func (m *mockThreadServiceForCoverage) SendThreadMessage(ctx context.Context, threadID, authorID uuid.UUID, content string) (*models.ThreadMessage, error) {
	if m.sendThreadMessageFunc != nil {
		return m.sendThreadMessageFunc(ctx, threadID, authorID, content)
	}
	return &models.ThreadMessage{ID: uuid.New(), ThreadID: threadID, AuthorID: authorID, Content: content, CreatedAt: time.Now()}, nil
}

func (m *mockThreadServiceForCoverage) ArchiveThread(ctx context.Context, threadID, requesterID uuid.UUID) error {
	if m.archiveThreadFunc != nil {
		return m.archiveThreadFunc(ctx, threadID, requesterID)
	}
	return nil
}

func (m *mockThreadServiceForCoverage) UnarchiveThread(ctx context.Context, threadID, requesterID uuid.UUID) error {
	if m.unarchiveThreadFunc != nil {
		return m.unarchiveThreadFunc(ctx, threadID, requesterID)
	}
	return nil
}

func (m *mockThreadServiceForCoverage) GetChannelThreads(ctx context.Context, channelID, requesterID uuid.UUID, includeArchived bool) ([]*models.Thread, error) {
	if m.getChannelThreadsFunc != nil {
		return m.getChannelThreadsFunc(ctx, channelID, requesterID, includeArchived)
	}
	return []*models.Thread{}, nil
}

func (m *mockThreadServiceForCoverage) JoinThread(ctx context.Context, threadID, userID uuid.UUID) error {
	if m.joinThreadFunc != nil {
		return m.joinThreadFunc(ctx, threadID, userID)
	}
	return nil
}

func (m *mockThreadServiceForCoverage) LeaveThread(ctx context.Context, threadID, userID uuid.UUID) error {
	if m.leaveThreadFunc != nil {
		return m.leaveThreadFunc(ctx, threadID, userID)
	}
	return nil
}

func (m *mockThreadServiceForCoverage) DeleteThread(ctx context.Context, threadID, requesterID uuid.UUID) error {
	if m.deleteThreadFunc != nil {
		return m.deleteThreadFunc(ctx, threadID, requesterID)
	}
	return nil
}

func (m *mockThreadServiceForCoverage) GetNotificationPreference(ctx context.Context, threadID, userID uuid.UUID) (*models.ThreadNotificationPreference, error) {
	if m.getNotificationPreferenceFunc != nil {
		return m.getNotificationPreferenceFunc(ctx, threadID, userID)
	}
	return &models.ThreadNotificationPreference{ThreadID: threadID, UserID: userID, Level: models.ThreadNotifyMentions}, nil
}

func (m *mockThreadServiceForCoverage) SetNotificationPreference(ctx context.Context, threadID, userID uuid.UUID, level models.ThreadNotificationLevel) error {
	if m.setNotificationPreferenceFunc != nil {
		return m.setNotificationPreferenceFunc(ctx, threadID, userID, level)
	}
	return nil
}

func (m *mockThreadServiceForCoverage) EnterThread(ctx context.Context, threadID, userID uuid.UUID) (*models.ThreadPresenceResponse, error) {
	if m.enterThreadFunc != nil {
		return m.enterThreadFunc(ctx, threadID, userID)
	}
	return &models.ThreadPresenceResponse{ThreadID: threadID, ActiveUsers: []models.ThreadPresenceUser{{ID: userID, Username: "testuser"}}}, nil
}

func (m *mockThreadServiceForCoverage) ExitThread(ctx context.Context, threadID, userID uuid.UUID) error {
	if m.exitThreadFunc != nil {
		return m.exitThreadFunc(ctx, threadID, userID)
	}
	return nil
}

func (m *mockThreadServiceForCoverage) GetActiveViewers(ctx context.Context, threadID uuid.UUID) (*models.ThreadPresenceResponse, error) {
	if m.getActiveViewersFunc != nil {
		return m.getActiveViewersFunc(ctx, threadID)
	}
	return &models.ThreadPresenceResponse{ThreadID: threadID, ActiveUsers: []models.ThreadPresenceUser{}}, nil
}

func (m *mockThreadServiceForCoverage) HeartbeatPresence(ctx context.Context, threadID, userID uuid.UUID) error {
	if m.heartbeatPresenceFunc != nil {
		return m.heartbeatPresenceFunc(ctx, threadID, userID)
	}
	return nil
}

// ====================================================================================================================================================
// TestThreadHandler_CreateThread
// ====================================================================================================================================================

func TestThreadHandler_CreateThread(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	tests := []struct {
		name           string
		channelIDParam string
		body           interface{}
		setupMock      func(*mockThreadServiceForCoverage)
		expectedStatus int
	}{
		{
			name:           "success",
			channelIDParam: channelID.String(),
			body:           map[string]interface{}{"name": "My Thread"},
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.createThreadFunc = func(ctx context.Context, cID, crID uuid.UUID, name string, autoArchive *int, tagIDs []uuid.UUID) (*models.Thread, error) {
					return &models.Thread{ID: uuid.New(), ParentChannelID: cID, OwnerID: crID, Name: name}, nil
				}
			},
			expectedStatus: fiber.StatusCreated,
		},
		{
			name:           "invalid channel id",
			channelIDParam: "not-a-uuid",
			body:           map[string]interface{}{"name": "My Thread"},
			setupMock:      func(m *mockThreadServiceForCoverage) {},
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:           "invalid request body",
			channelIDParam: channelID.String(),
			body:           "invalid-json",
			setupMock:      func(m *mockThreadServiceForCoverage) {},
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:           "missing name",
			channelIDParam: channelID.String(),
			body:           map[string]interface{}{},
			setupMock:      func(m *mockThreadServiceForCoverage) {},
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:           "channel not found",
			channelIDParam: channelID.String(),
			body:           map[string]interface{}{"name": "My Thread"},
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.createThreadFunc = func(ctx context.Context, cID, crID uuid.UUID, name string, autoArchive *int, tagIDs []uuid.UUID) (*models.Thread, error) {
					return nil, services.ErrChannelNotFound
				}
			},
			expectedStatus: fiber.StatusNotFound,
		},
		{
			name:           "not server member",
			channelIDParam: channelID.String(),
			body:           map[string]interface{}{"name": "My Thread"},
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.createThreadFunc = func(ctx context.Context, cID, crID uuid.UUID, name string, autoArchive *int, tagIDs []uuid.UUID) (*models.Thread, error) {
					return nil, services.ErrNotServerMember
				}
			},
			expectedStatus: fiber.StatusForbidden,
		},
		{
			name:           "invalid auto archive",
			channelIDParam: channelID.String(),
			body:           map[string]interface{}{"name": "My Thread", "auto_archive": 9999},
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.createThreadFunc = func(ctx context.Context, cID, crID uuid.UUID, name string, autoArchive *int, tagIDs []uuid.UUID) (*models.Thread, error) {
					return nil, services.ErrInvalidAutoArchive
				}
			},
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:           "success with auto archive",
			channelIDParam: channelID.String(),
			body:           map[string]interface{}{"name": "My Thread", "auto_archive": 1440},
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.createThreadFunc = func(ctx context.Context, cID, crID uuid.UUID, name string, autoArchive *int, tagIDs []uuid.UUID) (*models.Thread, error) {
					return &models.Thread{ID: uuid.New(), ParentChannelID: cID, OwnerID: crID, Name: name, AutoArchive: *autoArchive}, nil
				}
			},
			expectedStatus: fiber.StatusCreated,
		},
		{
			name:           "success with parent message id",
			channelIDParam: channelID.String(),
			body:           map[string]interface{}{"name": "My Thread", "parent_message_id": uuid.New().String()},
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.createThreadFunc = func(ctx context.Context, cID, crID uuid.UUID, name string, autoArchive *int, tagIDs []uuid.UUID) (*models.Thread, error) {
					return &models.Thread{ID: uuid.New(), ParentChannelID: cID, OwnerID: crID, Name: name, AppliedTags: tagIDs}, nil
				}
			},
			expectedStatus: fiber.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockThreadServiceForCoverage{}
			tt.setupMock(svc)
			h := NewThreadHandler(svc)

			app := fiber.New()
			app.Use(func(c *fiber.Ctx) error {
				c.Locals("userID", userID)
				return c.Next()
			})
			app.Post("/channels/:id/threads", h.CreateThread)

			var body []byte
			if s, ok := tt.body.(string); ok {
				body = []byte(s)
			} else {
				body, _ = json.Marshal(tt.body)
			}

			req := httptest.NewRequest("POST", "/channels/"+tt.channelIDParam+"/threads", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			resp, _ := app.Test(req)
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

// ====================================================================================================================================================
// TestThreadHandler_GetThread
// ====================================================================================================================================================

func TestThreadHandler_GetThread(t *testing.T) {
	threadID := uuid.New()

	tests := []struct {
		name           string
		threadIDParam  string
		setupMock      func(*mockThreadServiceForCoverage)
		expectedStatus int
	}{
		{
			name:          "success",
			threadIDParam: threadID.String(),
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.getThreadFunc = func(ctx context.Context, tID uuid.UUID) (*models.Thread, error) {
					return &models.Thread{ID: tID, Name: "Test Thread"}, nil
				}
			},
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "invalid thread id",
			threadIDParam:  "not-a-uuid",
			setupMock:      func(m *mockThreadServiceForCoverage) {},
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:          "thread not found",
			threadIDParam: threadID.String(),
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.getThreadFunc = func(ctx context.Context, tID uuid.UUID) (*models.Thread, error) {
					return nil, services.ErrThreadNotFound
				}
			},
			expectedStatus: fiber.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockThreadServiceForCoverage{}
			tt.setupMock(svc)
			h := NewThreadHandler(svc)

			app := fiber.New()
			app.Get("/threads/:id", h.GetThread)

			req := httptest.NewRequest("GET", "/threads/"+tt.threadIDParam, nil)
			resp, _ := app.Test(req)
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

// ====================================================================================================================================================
// TestThreadHandler_GetThreadMessages
// ====================================================================================================================================================

func TestThreadHandler_GetThreadMessages(t *testing.T) {
	userID := uuid.New()
	threadID := uuid.New()

	tests := []struct {
		name           string
		threadIDParam  string
		setupMock      func(*mockThreadServiceForCoverage)
		expectedStatus int
	}{
		{
			name:          "success",
			threadIDParam: threadID.String(),
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.getThreadMessagesFunc = func(ctx context.Context, tID, rID uuid.UUID, before *uuid.UUID, limit int) ([]*models.ThreadMessage, error) {
					return []*models.ThreadMessage{}, nil
				}
			},
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "invalid thread id",
			threadIDParam:  "not-a-uuid",
			setupMock:      func(m *mockThreadServiceForCoverage) {},
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:          "thread not found",
			threadIDParam: threadID.String(),
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.getThreadMessagesFunc = func(ctx context.Context, tID, rID uuid.UUID, before *uuid.UUID, limit int) ([]*models.ThreadMessage, error) {
					return nil, services.ErrThreadNotFound
				}
			},
			expectedStatus: fiber.StatusNotFound,
		},
		{
			name:          "not server member",
			threadIDParam: threadID.String(),
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.getThreadMessagesFunc = func(ctx context.Context, tID, rID uuid.UUID, before *uuid.UUID, limit int) ([]*models.ThreadMessage, error) {
					return nil, services.ErrNotServerMember
				}
			},
			expectedStatus: fiber.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockThreadServiceForCoverage{}
			tt.setupMock(svc)
			h := NewThreadHandler(svc)

			app := fiber.New()
			app.Get("/threads/:id/messages", func(c *fiber.Ctx) error {
				c.Locals("userID", userID)
				return h.GetThreadMessages(c)
			})

			req := httptest.NewRequest("GET", "/threads/"+tt.threadIDParam+"/messages", nil)
			resp, _ := app.Test(req)
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

// ====================================================================================================================================================
// TestThreadHandler_SendThreadMessage
// ====================================================================================================================================================

func TestThreadHandler_SendThreadMessage(t *testing.T) {
	userID := uuid.New()
	threadID := uuid.New()

	tests := []struct {
		name           string
		threadIDParam  string
		body           interface{}
		setupMock      func(*mockThreadServiceForCoverage)
		expectedStatus int
	}{
		{
			name:          "success",
			threadIDParam: threadID.String(),
			body:          map[string]interface{}{"content": "Hello thread"},
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.sendThreadMessageFunc = func(ctx context.Context, tID, aID uuid.UUID, content string) (*models.ThreadMessage, error) {
					return &models.ThreadMessage{ID: uuid.New(), ThreadID: tID, AuthorID: aID, Content: content}, nil
				}
			},
			expectedStatus: fiber.StatusCreated,
		},
		{
			name:           "invalid thread id",
			threadIDParam:  "not-a-uuid",
			body:           map[string]interface{}{"content": "Hello"},
			setupMock:      func(m *mockThreadServiceForCoverage) {},
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:           "invalid request body",
			threadIDParam:  threadID.String(),
			body:           "not-json",
			setupMock:      func(m *mockThreadServiceForCoverage) {},
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:           "missing content",
			threadIDParam:  threadID.String(),
			body:           map[string]interface{}{},
			setupMock:      func(m *mockThreadServiceForCoverage) {},
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:          "thread not found",
			threadIDParam: threadID.String(),
			body:          map[string]interface{}{"content": "Hello"},
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.sendThreadMessageFunc = func(ctx context.Context, tID, aID uuid.UUID, content string) (*models.ThreadMessage, error) {
					return nil, services.ErrThreadNotFound
				}
			},
			expectedStatus: fiber.StatusNotFound,
		},
		{
			name:          "thread archived",
			threadIDParam: threadID.String(),
			body:          map[string]interface{}{"content": "Hello"},
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.sendThreadMessageFunc = func(ctx context.Context, tID, aID uuid.UUID, content string) (*models.ThreadMessage, error) {
					return nil, services.ErrThreadArchived
				}
			},
			expectedStatus: fiber.StatusForbidden,
		},
		{
			name:          "thread locked",
			threadIDParam: threadID.String(),
			body:          map[string]interface{}{"content": "Hello"},
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.sendThreadMessageFunc = func(ctx context.Context, tID, aID uuid.UUID, content string) (*models.ThreadMessage, error) {
					return nil, services.ErrThreadLocked
				}
			},
			expectedStatus: fiber.StatusForbidden,
		},
		{
			name:          "not server member",
			threadIDParam: threadID.String(),
			body:          map[string]interface{}{"content": "Hello"},
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.sendThreadMessageFunc = func(ctx context.Context, tID, aID uuid.UUID, content string) (*models.ThreadMessage, error) {
					return nil, services.ErrNotServerMember
				}
			},
			expectedStatus: fiber.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockThreadServiceForCoverage{}
			tt.setupMock(svc)
			h := NewThreadHandler(svc)

			app := fiber.New()
			app.Post("/threads/:id/messages", func(c *fiber.Ctx) error {
				c.Locals("userID", userID)
				return h.SendThreadMessage(c)
			})

			var body []byte
			if s, ok := tt.body.(string); ok {
				body = []byte(s)
			} else {
				body, _ = json.Marshal(tt.body)
			}

			req := httptest.NewRequest("POST", "/threads/"+tt.threadIDParam+"/messages", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			resp, _ := app.Test(req)
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

// ====================================================================================================================================================
// TestThreadHandler_ArchiveThread
// ====================================================================================================================================================

func TestThreadHandler_ArchiveThread(t *testing.T) {
	userID := uuid.New()
	threadID := uuid.New()

	tests := []struct {
		name           string
		threadIDParam  string
		setupMock      func(*mockThreadServiceForCoverage)
		expectedStatus int
	}{
		{
			name:          "success",
			threadIDParam: threadID.String(),
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.archiveThreadFunc = func(ctx context.Context, tID, rID uuid.UUID) error {
					return nil
				}
			},
			expectedStatus: fiber.StatusNoContent,
		},
		{
			name:           "invalid thread id",
			threadIDParam:  "not-a-uuid",
			setupMock:      func(m *mockThreadServiceForCoverage) {},
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:          "thread not found",
			threadIDParam: threadID.String(),
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.archiveThreadFunc = func(ctx context.Context, tID, rID uuid.UUID) error {
					return services.ErrThreadNotFound
				}
			},
			expectedStatus: fiber.StatusNotFound,
		},
		{
			name:          "not thread owner",
			threadIDParam: threadID.String(),
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.archiveThreadFunc = func(ctx context.Context, tID, rID uuid.UUID) error {
					return services.ErrNotThreadOwner
				}
			},
			expectedStatus: fiber.StatusForbidden,
		},
		{
			name:          "not server member",
			threadIDParam: threadID.String(),
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.archiveThreadFunc = func(ctx context.Context, tID, rID uuid.UUID) error {
					return services.ErrNotServerMember
				}
			},
			expectedStatus: fiber.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockThreadServiceForCoverage{}
			tt.setupMock(svc)
			h := NewThreadHandler(svc)

			app := fiber.New()
			app.Post("/threads/:id/archive", func(c *fiber.Ctx) error {
				c.Locals("userID", userID)
				return h.ArchiveThread(c)
			})

			req := httptest.NewRequest("POST", "/threads/"+tt.threadIDParam+"/archive", nil)
			resp, _ := app.Test(req)
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

// ====================================================================================================================================================
// TestThreadHandler_UnarchiveThread
// ====================================================================================================================================================

func TestThreadHandler_UnarchiveThread(t *testing.T) {
	userID := uuid.New()
	threadID := uuid.New()

	tests := []struct {
		name           string
		threadIDParam  string
		setupMock      func(*mockThreadServiceForCoverage)
		expectedStatus int
	}{
		{
			name:          "success",
			threadIDParam: threadID.String(),
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.unarchiveThreadFunc = func(ctx context.Context, tID, rID uuid.UUID) error {
					return nil
				}
			},
			expectedStatus: fiber.StatusNoContent,
		},
		{
			name:           "invalid thread id",
			threadIDParam:  "not-a-uuid",
			setupMock:      func(m *mockThreadServiceForCoverage) {},
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:          "thread not found",
			threadIDParam: threadID.String(),
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.unarchiveThreadFunc = func(ctx context.Context, tID, rID uuid.UUID) error {
					return services.ErrThreadNotFound
				}
			},
			expectedStatus: fiber.StatusNotFound,
		},
		{
			name:          "not thread owner",
			threadIDParam: threadID.String(),
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.unarchiveThreadFunc = func(ctx context.Context, tID, rID uuid.UUID) error {
					return services.ErrNotThreadOwner
				}
			},
			expectedStatus: fiber.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockThreadServiceForCoverage{}
			tt.setupMock(svc)
			h := NewThreadHandler(svc)

			app := fiber.New()
			app.Post("/threads/:id/unarchive", func(c *fiber.Ctx) error {
				c.Locals("userID", userID)
				return h.UnarchiveThread(c)
			})

			req := httptest.NewRequest("POST", "/threads/"+tt.threadIDParam+"/unarchive", nil)
			resp, _ := app.Test(req)
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

// ====================================================================================================================================================
// TestThreadHandler_GetChannelThreads
// ====================================================================================================================================================

func TestThreadHandler_GetChannelThreads(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	tests := []struct {
		name           string
		channelIDParam string
		query          string
		setupMock      func(*mockThreadServiceForCoverage)
		expectedStatus int
	}{
		{
			name:           "success",
			channelIDParam: channelID.String(),
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.getChannelThreadsFunc = func(ctx context.Context, cID, rID uuid.UUID, includeArchived bool) ([]*models.Thread, error) {
					return []*models.Thread{}, nil
				}
			},
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "invalid channel id",
			channelIDParam: "not-a-uuid",
			setupMock:      func(m *mockThreadServiceForCoverage) {},
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:           "channel not found",
			channelIDParam: channelID.String(),
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.getChannelThreadsFunc = func(ctx context.Context, cID, rID uuid.UUID, includeArchived bool) ([]*models.Thread, error) {
					return nil, services.ErrChannelNotFound
				}
			},
			expectedStatus: fiber.StatusNotFound,
		},
		{
			name:           "not server member",
			channelIDParam: channelID.String(),
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.getChannelThreadsFunc = func(ctx context.Context, cID, rID uuid.UUID, includeArchived bool) ([]*models.Thread, error) {
					return nil, services.ErrNotServerMember
				}
			},
			expectedStatus: fiber.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockThreadServiceForCoverage{}
			tt.setupMock(svc)
			h := NewThreadHandler(svc)

			app := fiber.New()
			app.Get("/channels/:id/threads", func(c *fiber.Ctx) error {
				c.Locals("userID", userID)
				return h.GetChannelThreads(c)
			})

			req := httptest.NewRequest("GET", "/channels/"+tt.channelIDParam+"/threads"+tt.query, nil)
			resp, _ := app.Test(req)
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

// ====================================================================================================================================================
// TestThreadHandler_JoinThread
// ====================================================================================================================================================

func TestThreadHandler_JoinThread(t *testing.T) {
	userID := uuid.New()
	threadID := uuid.New()

	tests := []struct {
		name           string
		threadIDParam  string
		setupMock      func(*mockThreadServiceForCoverage)
		expectedStatus int
	}{
		{
			name:          "success",
			threadIDParam: threadID.String(),
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.joinThreadFunc = func(ctx context.Context, tID, uID uuid.UUID) error {
					return nil
				}
			},
			expectedStatus: fiber.StatusNoContent,
		},
		{
			name:           "invalid thread id",
			threadIDParam:  "not-a-uuid",
			setupMock:      func(m *mockThreadServiceForCoverage) {},
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:          "thread not found",
			threadIDParam: threadID.String(),
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.joinThreadFunc = func(ctx context.Context, tID, uID uuid.UUID) error {
					return services.ErrThreadNotFound
				}
			},
			expectedStatus: fiber.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockThreadServiceForCoverage{}
			tt.setupMock(svc)
			h := NewThreadHandler(svc)

			app := fiber.New()
			app.Post("/threads/:id/join", func(c *fiber.Ctx) error {
				c.Locals("userID", userID)
				return h.JoinThread(c)
			})

			req := httptest.NewRequest("POST", "/threads/"+tt.threadIDParam+"/join", nil)
			resp, _ := app.Test(req)
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

// ====================================================================================================================================================
// TestThreadHandler_LeaveThread
// ====================================================================================================================================================

func TestThreadHandler_LeaveThread(t *testing.T) {
	userID := uuid.New()
	threadID := uuid.New()

	tests := []struct {
		name           string
		threadIDParam  string
		setupMock      func(*mockThreadServiceForCoverage)
		expectedStatus int
	}{
		{
			name:          "success",
			threadIDParam: threadID.String(),
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.leaveThreadFunc = func(ctx context.Context, tID, uID uuid.UUID) error {
					return nil
				}
			},
			expectedStatus: fiber.StatusNoContent,
		},
		{
			name:           "invalid thread id",
			threadIDParam:  "not-a-uuid",
			setupMock:      func(m *mockThreadServiceForCoverage) {},
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:          "thread not found",
			threadIDParam: threadID.String(),
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.leaveThreadFunc = func(ctx context.Context, tID, uID uuid.UUID) error {
					return services.ErrThreadNotFound
				}
			},
			expectedStatus: fiber.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockThreadServiceForCoverage{}
			tt.setupMock(svc)
			h := NewThreadHandler(svc)

			app := fiber.New()
			app.Delete("/threads/:id/members/@me", func(c *fiber.Ctx) error {
				c.Locals("userID", userID)
				return h.LeaveThread(c)
			})

			req := httptest.NewRequest("DELETE", "/threads/"+tt.threadIDParam+"/members/@me", nil)
			resp, _ := app.Test(req)
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

// ====================================================================================================================================================
// TestThreadHandler_DeleteThread
// ====================================================================================================================================================

// ====================================================================================================================================================
// TestThreadHandler_DeleteThread
// ====================================================================================================================================================

func TestThreadHandler_DeleteThread(t *testing.T) {
	userID := uuid.New()
	threadID := uuid.New()

	tests := []struct {
		name           string
		threadIDParam  string
		setupMock      func(*mockThreadServiceForCoverage)
		expectedStatus int
	}{
		{
			name:          "success",
			threadIDParam: threadID.String(),
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.deleteThreadFunc = func(ctx context.Context, tID, rID uuid.UUID) error {
					return nil
				}
			},
			expectedStatus: fiber.StatusNoContent,
		},
		{
			name:           "invalid thread id",
			threadIDParam:  "not-a-uuid",
			setupMock:      func(m *mockThreadServiceForCoverage) {},
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:          "thread not found",
			threadIDParam: threadID.String(),
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.deleteThreadFunc = func(ctx context.Context, tID, rID uuid.UUID) error {
					return services.ErrThreadNotFound
				}
			},
			expectedStatus: fiber.StatusNotFound,
		},
		{
			name:          "not thread owner",
			threadIDParam: threadID.String(),
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.deleteThreadFunc = func(ctx context.Context, tID, rID uuid.UUID) error {
					return services.ErrNotThreadOwner
				}
			},
			expectedStatus: fiber.StatusForbidden,
		},
		{
			name:          "not server member",
			threadIDParam: threadID.String(),
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.deleteThreadFunc = func(ctx context.Context, tID, rID uuid.UUID) error {
					return services.ErrNotServerMember
				}
			},
			expectedStatus: fiber.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockThreadServiceForCoverage{}
			tt.setupMock(svc)
			h := NewThreadHandler(svc)

			app := fiber.New()
			app.Delete("/threads/:id", func(c *fiber.Ctx) error {
				c.Locals("userID", userID)
				return h.DeleteThread(c)
			})

			req := httptest.NewRequest("DELETE", "/threads/"+tt.threadIDParam, nil)
			resp, _ := app.Test(req)
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

// ====================================================================================================================================================
// TestThreadHandler_UpdateThread
// ====================================================================================================================================================

func TestThreadHandler_UpdateThread(t *testing.T) {
	userID := uuid.New()
	threadID := uuid.New()
	name := "Updated Thread"

	tests := []struct {
		name           string
		threadIDParam  string
		body           interface{}
		setupMock      func(*mockThreadServiceForCoverage)
		expectedStatus int
	}{
		{
			name:          "success",
			threadIDParam: threadID.String(),
			body:          map[string]interface{}{"name": name},
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.updateThreadFunc = func(ctx context.Context, tID, rID uuid.UUID, req models.UpdateThreadRequest) (*models.Thread, error) {
					n := name
					if req.Name != nil {
						n = *req.Name
					}
					return &models.Thread{ID: tID, Name: n}, nil
				}
			},
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "invalid thread id",
			threadIDParam:  "not-a-uuid",
			body:           map[string]interface{}{"name": name},
			setupMock:      func(m *mockThreadServiceForCoverage) {},
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:           "invalid request body",
			threadIDParam:  threadID.String(),
			body:           "not-json",
			setupMock:      func(m *mockThreadServiceForCoverage) {},
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:          "thread not found",
			threadIDParam: threadID.String(),
			body:          map[string]interface{}{"name": name},
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.updateThreadFunc = func(ctx context.Context, tID, rID uuid.UUID, req models.UpdateThreadRequest) (*models.Thread, error) {
					return nil, services.ErrThreadNotFound
				}
			},
			expectedStatus: fiber.StatusNotFound,
		},
		{
			name:          "not thread owner",
			threadIDParam: threadID.String(),
			body:          map[string]interface{}{"name": name},
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.updateThreadFunc = func(ctx context.Context, tID, rID uuid.UUID, req models.UpdateThreadRequest) (*models.Thread, error) {
					return nil, services.ErrNotThreadOwner
				}
			},
			expectedStatus: fiber.StatusForbidden,
		},
		{
			name:          "not server member",
			threadIDParam: threadID.String(),
			body:          map[string]interface{}{"name": name},
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.updateThreadFunc = func(ctx context.Context, tID, rID uuid.UUID, req models.UpdateThreadRequest) (*models.Thread, error) {
					return nil, services.ErrNotServerMember
				}
			},
			expectedStatus: fiber.StatusForbidden,
		},
		{
			name:          "invalid auto archive",
			threadIDParam: threadID.String(),
			body:          map[string]interface{}{"auto_archive": 9999},
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.updateThreadFunc = func(ctx context.Context, tID, rID uuid.UUID, req models.UpdateThreadRequest) (*models.Thread, error) {
					return nil, services.ErrInvalidAutoArchive
				}
			},
			expectedStatus: fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockThreadServiceForCoverage{}
			tt.setupMock(svc)
			h := NewThreadHandler(svc)

			app := fiber.New()
			app.Patch("/threads/:id", func(c *fiber.Ctx) error {
				c.Locals("userID", userID)
				return h.UpdateThread(c)
			})

			var body []byte
			if s, ok := tt.body.(string); ok {
				body = []byte(s)
			} else {
				body, _ = json.Marshal(tt.body)
			}

			req := httptest.NewRequest("PATCH", "/threads/"+tt.threadIDParam, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			resp, _ := app.Test(req)
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

// ====================================================================================================================================================
// TestThreadHandler_GetNotificationPreference
// ====================================================================================================================================================

func TestThreadHandler_GetNotificationPreference(t *testing.T) {
	userID := uuid.New()
	threadID := uuid.New()

	tests := []struct {
		name           string
		threadIDParam  string
		setupMock      func(*mockThreadServiceForCoverage)
		expectedStatus int
	}{
		{
			name:          "success",
			threadIDParam: threadID.String(),
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.getNotificationPreferenceFunc = func(ctx context.Context, tID, uID uuid.UUID) (*models.ThreadNotificationPreference, error) {
					return &models.ThreadNotificationPreference{ThreadID: tID, UserID: uID, Level: models.ThreadNotifyMentions}, nil
				}
			},
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "invalid thread id",
			threadIDParam:  "not-a-uuid",
			setupMock:      func(m *mockThreadServiceForCoverage) {},
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:          "thread not found",
			threadIDParam: threadID.String(),
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.getNotificationPreferenceFunc = func(ctx context.Context, tID, uID uuid.UUID) (*models.ThreadNotificationPreference, error) {
					return nil, services.ErrThreadNotFound
				}
			},
			expectedStatus: fiber.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockThreadServiceForCoverage{}
			tt.setupMock(svc)
			h := NewThreadHandler(svc)

			app := fiber.New()
			app.Get("/threads/:id/notifications", func(c *fiber.Ctx) error {
				c.Locals("userID", userID)
				return h.GetNotificationPreference(c)
			})

			req := httptest.NewRequest("GET", "/threads/"+tt.threadIDParam+"/notifications", nil)
			resp, _ := app.Test(req)
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

// ====================================================================================================================================================
// TestThreadHandler_SetNotificationPreference
// ====================================================================================================================================================

func TestThreadHandler_SetNotificationPreference(t *testing.T) {
	userID := uuid.New()
	threadID := uuid.New()

	tests := []struct {
		name           string
		threadIDParam  string
		body           interface{}
		setupMock      func(*mockThreadServiceForCoverage)
		expectedStatus int
	}{
		{
			name:          "success all",
			threadIDParam: threadID.String(),
			body:          map[string]interface{}{"level": "all"},
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.setNotificationPreferenceFunc = func(ctx context.Context, tID, uID uuid.UUID, level models.ThreadNotificationLevel) error {
					return nil
				}
			},
			expectedStatus: fiber.StatusOK,
		},
		{
			name:          "success mentions",
			threadIDParam: threadID.String(),
			body:          map[string]interface{}{"level": "mentions"},
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.setNotificationPreferenceFunc = func(ctx context.Context, tID, uID uuid.UUID, level models.ThreadNotificationLevel) error {
					return nil
				}
			},
			expectedStatus: fiber.StatusOK,
		},
		{
			name:          "success none",
			threadIDParam: threadID.String(),
			body:          map[string]interface{}{"level": "none"},
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.setNotificationPreferenceFunc = func(ctx context.Context, tID, uID uuid.UUID, level models.ThreadNotificationLevel) error {
					return nil
				}
			},
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "invalid thread id",
			threadIDParam:  "not-a-uuid",
			body:           map[string]interface{}{"level": "all"},
			setupMock:      func(m *mockThreadServiceForCoverage) {},
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:           "invalid level",
			threadIDParam:  threadID.String(),
			body:           map[string]interface{}{"level": "invalid"},
			setupMock:      func(m *mockThreadServiceForCoverage) {},
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:           "invalid request body",
			threadIDParam:  threadID.String(),
			body:           "not-json",
			setupMock:      func(m *mockThreadServiceForCoverage) {},
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:          "thread not found",
			threadIDParam: threadID.String(),
			body:          map[string]interface{}{"level": "all"},
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.setNotificationPreferenceFunc = func(ctx context.Context, tID, uID uuid.UUID, level models.ThreadNotificationLevel) error {
					return services.ErrThreadNotFound
				}
			},
			expectedStatus: fiber.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockThreadServiceForCoverage{}
			tt.setupMock(svc)
			h := NewThreadHandler(svc)

			app := fiber.New()
			app.Put("/threads/:id/notifications", func(c *fiber.Ctx) error {
				c.Locals("userID", userID)
				return h.SetNotificationPreference(c)
			})

			var body []byte
			if s, ok := tt.body.(string); ok {
				body = []byte(s)
			} else {
				body, _ = json.Marshal(tt.body)
			}

			req := httptest.NewRequest("PUT", "/threads/"+tt.threadIDParam+"/notifications", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			resp, _ := app.Test(req)
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

// ====================================================================================================================================================
// TestThreadHandler_EnterThread
// ====================================================================================================================================================

func TestThreadHandler_EnterThread(t *testing.T) {
	userID := uuid.New()
	threadID := uuid.New()

	tests := []struct {
		name           string
		threadIDParam  string
		setupMock      func(*mockThreadServiceForCoverage)
		expectedStatus int
	}{
		{
			name:          "success",
			threadIDParam: threadID.String(),
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.enterThreadFunc = func(ctx context.Context, tID, uID uuid.UUID) (*models.ThreadPresenceResponse, error) {
					return &models.ThreadPresenceResponse{ThreadID: tID, ActiveUsers: []models.ThreadPresenceUser{{ID: uID, Username: "testuser"}}}, nil
				}
			},
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "invalid thread id",
			threadIDParam:  "not-a-uuid",
			setupMock:      func(m *mockThreadServiceForCoverage) {},
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:          "thread not found",
			threadIDParam: threadID.String(),
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.enterThreadFunc = func(ctx context.Context, tID, uID uuid.UUID) (*models.ThreadPresenceResponse, error) {
					return nil, services.ErrThreadNotFound
				}
			},
			expectedStatus: fiber.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockThreadServiceForCoverage{}
			tt.setupMock(svc)
			h := NewThreadHandler(svc)

			app := fiber.New()
			app.Post("/threads/:id/presence", func(c *fiber.Ctx) error {
				c.Locals("userID", userID)
				return h.EnterThread(c)
			})

			req := httptest.NewRequest("POST", "/threads/"+tt.threadIDParam+"/presence", nil)
			resp, _ := app.Test(req)
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

// ====================================================================================================================================================
// TestThreadHandler_ExitThreadPresence
// ====================================================================================================================================================

func TestThreadHandler_ExitThreadPresence(t *testing.T) {
	userID := uuid.New()
	threadID := uuid.New()

	tests := []struct {
		name           string
		threadIDParam  string
		setupMock      func(*mockThreadServiceForCoverage)
		expectedStatus int
	}{
		{
			name:          "success",
			threadIDParam: threadID.String(),
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.exitThreadFunc = func(ctx context.Context, tID, uID uuid.UUID) error {
					return nil
				}
			},
			expectedStatus: fiber.StatusNoContent,
		},
		{
			name:           "invalid thread id",
			threadIDParam:  "not-a-uuid",
			setupMock:      func(m *mockThreadServiceForCoverage) {},
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:          "internal error",
			threadIDParam: threadID.String(),
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.exitThreadFunc = func(ctx context.Context, tID, uID uuid.UUID) error {
					return context.DeadlineExceeded
				}
			},
			expectedStatus: fiber.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockThreadServiceForCoverage{}
			tt.setupMock(svc)
			h := NewThreadHandler(svc)

			app := fiber.New()
			app.Delete("/threads/:id/presence", func(c *fiber.Ctx) error {
				c.Locals("userID", userID)
				return h.ExitThreadPresence(c)
			})

			req := httptest.NewRequest("DELETE", "/threads/"+tt.threadIDParam+"/presence", nil)
			resp, _ := app.Test(req)
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

// ====================================================================================================================================================
// TestThreadHandler_GetActiveViewers
// ====================================================================================================================================================

func TestThreadHandler_GetActiveViewers(t *testing.T) {
	threadID := uuid.New()

	tests := []struct {
		name           string
		threadIDParam  string
		setupMock      func(*mockThreadServiceForCoverage)
		expectedStatus int
	}{
		{
			name:          "success",
			threadIDParam: threadID.String(),
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.getActiveViewersFunc = func(ctx context.Context, tID uuid.UUID) (*models.ThreadPresenceResponse, error) {
					return &models.ThreadPresenceResponse{ThreadID: tID, ActiveUsers: []models.ThreadPresenceUser{{ID: uuid.New(), Username: "user1"}}}, nil
				}
			},
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "invalid thread id",
			threadIDParam:  "not-a-uuid",
			setupMock:      func(m *mockThreadServiceForCoverage) {},
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:          "thread not found",
			threadIDParam: threadID.String(),
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.getActiveViewersFunc = func(ctx context.Context, tID uuid.UUID) (*models.ThreadPresenceResponse, error) {
					return nil, services.ErrThreadNotFound
				}
			},
			expectedStatus: fiber.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockThreadServiceForCoverage{}
			tt.setupMock(svc)
			h := NewThreadHandler(svc)

			app := fiber.New()
			app.Get("/threads/:id/presence", h.GetActiveViewers)

			req := httptest.NewRequest("GET", "/threads/"+tt.threadIDParam+"/presence", nil)
			resp, _ := app.Test(req)
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

// ====================================================================================================================================================
// TestThreadHandler_HeartbeatPresence
// ====================================================================================================================================================

func TestThreadHandler_HeartbeatPresence(t *testing.T) {
	userID := uuid.New()
	threadID := uuid.New()

	tests := []struct {
		name           string
		threadIDParam  string
		setupMock      func(*mockThreadServiceForCoverage)
		expectedStatus int
	}{
		{
			name:          "success",
			threadIDParam: threadID.String(),
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.heartbeatPresenceFunc = func(ctx context.Context, tID, uID uuid.UUID) error {
					return nil
				}
			},
			expectedStatus: fiber.StatusNoContent,
		},
		{
			name:           "invalid thread id",
			threadIDParam:  "not-a-uuid",
			setupMock:      func(m *mockThreadServiceForCoverage) {},
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:          "internal error",
			threadIDParam: threadID.String(),
			setupMock: func(m *mockThreadServiceForCoverage) {
				m.heartbeatPresenceFunc = func(ctx context.Context, tID, uID uuid.UUID) error {
					return context.DeadlineExceeded
				}
			},
			expectedStatus: fiber.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockThreadServiceForCoverage{}
			tt.setupMock(svc)
			h := NewThreadHandler(svc)

			app := fiber.New()
			app.Patch("/threads/:id/presence", func(c *fiber.Ctx) error {
				c.Locals("userID", userID)
				return h.HeartbeatPresence(c)
			})

			req := httptest.NewRequest("PATCH", "/threads/"+tt.threadIDParam+"/presence", nil)
			resp, _ := app.Test(req)
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}
