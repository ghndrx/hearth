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

// mockThreadService mocks ThreadService for handler tests
type mockThreadService struct {
	createThreadFunc       func(ctx context.Context, channelID, creatorID uuid.UUID, name string, autoArchive *int) (*models.Thread, error)
	getThreadFunc          func(ctx context.Context, threadID uuid.UUID) (*models.Thread, error)
	getThreadMessagesFunc  func(ctx context.Context, threadID, requesterID uuid.UUID, before *uuid.UUID, limit int) ([]*models.ThreadMessage, error)
	sendThreadMessageFunc  func(ctx context.Context, threadID, authorID uuid.UUID, content string) (*models.ThreadMessage, error)
	archiveThreadFunc      func(ctx context.Context, threadID, requesterID uuid.UUID) error
	unarchiveThreadFunc    func(ctx context.Context, threadID, requesterID uuid.UUID) error
	getChannelThreadsFunc  func(ctx context.Context, channelID, requesterID uuid.UUID, includeArchived bool) ([]*models.Thread, error)
	joinThreadFunc         func(ctx context.Context, threadID, userID uuid.UUID) error
	leaveThreadFunc        func(ctx context.Context, threadID, userID uuid.UUID) error
	deleteThreadFunc       func(ctx context.Context, threadID, requesterID uuid.UUID) error
}

func (m *mockThreadService) CreateThread(ctx context.Context, channelID, creatorID uuid.UUID, name string, autoArchive *int) (*models.Thread, error) {
	if m.createThreadFunc != nil {
		return m.createThreadFunc(ctx, channelID, creatorID, name, autoArchive)
	}
	return &models.Thread{ID: uuid.New(), ParentChannelID: channelID, OwnerID: creatorID, Name: name}, nil
}

func (m *mockThreadService) GetThread(ctx context.Context, threadID uuid.UUID) (*models.Thread, error) {
	if m.getThreadFunc != nil {
		return m.getThreadFunc(ctx, threadID)
	}
	return &models.Thread{ID: threadID, Name: "Test Thread"}, nil
}

func (m *mockThreadService) GetThreadMessages(ctx context.Context, threadID, requesterID uuid.UUID, before *uuid.UUID, limit int) ([]*models.ThreadMessage, error) {
	if m.getThreadMessagesFunc != nil {
		return m.getThreadMessagesFunc(ctx, threadID, requesterID, before, limit)
	}
	return []*models.ThreadMessage{}, nil
}

func (m *mockThreadService) SendThreadMessage(ctx context.Context, threadID, authorID uuid.UUID, content string) (*models.ThreadMessage, error) {
	if m.sendThreadMessageFunc != nil {
		return m.sendThreadMessageFunc(ctx, threadID, authorID, content)
	}
	return &models.ThreadMessage{ID: uuid.New(), ThreadID: threadID, AuthorID: authorID, Content: content, CreatedAt: time.Now()}, nil
}

func (m *mockThreadService) ArchiveThread(ctx context.Context, threadID, requesterID uuid.UUID) error {
	if m.archiveThreadFunc != nil {
		return m.archiveThreadFunc(ctx, threadID, requesterID)
	}
	return nil
}

func (m *mockThreadService) UnarchiveThread(ctx context.Context, threadID, requesterID uuid.UUID) error {
	if m.unarchiveThreadFunc != nil {
		return m.unarchiveThreadFunc(ctx, threadID, requesterID)
	}
	return nil
}

func (m *mockThreadService) GetChannelThreads(ctx context.Context, channelID, requesterID uuid.UUID, includeArchived bool) ([]*models.Thread, error) {
	if m.getChannelThreadsFunc != nil {
		return m.getChannelThreadsFunc(ctx, channelID, requesterID, includeArchived)
	}
	return []*models.Thread{}, nil
}

func (m *mockThreadService) JoinThread(ctx context.Context, threadID, userID uuid.UUID) error {
	if m.joinThreadFunc != nil {
		return m.joinThreadFunc(ctx, threadID, userID)
	}
	return nil
}

func (m *mockThreadService) LeaveThread(ctx context.Context, threadID, userID uuid.UUID) error {
	if m.leaveThreadFunc != nil {
		return m.leaveThreadFunc(ctx, threadID, userID)
	}
	return nil
}

func (m *mockThreadService) DeleteThread(ctx context.Context, threadID, requesterID uuid.UUID) error {
	if m.deleteThreadFunc != nil {
		return m.deleteThreadFunc(ctx, threadID, requesterID)
	}
	return nil
}

// setupThreadTestApp creates a test Fiber app with thread routes
func setupThreadTestApp(svc *mockThreadService) *fiber.App {
	app := fiber.New()

	// Inject userID middleware
	app.Use(func(c *fiber.Ctx) error {
		userID := c.Get("X-User-ID")
		if userID != "" {
			id, _ := uuid.Parse(userID)
			c.Locals("userID", id)
		}
		return c.Next()
	})

	// Create Thread
	app.Post("/channels/:id/threads", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		channelID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid channel id"})
		}

		var req models.CreateThreadRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}

		if req.Name == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
		}

		thread, err := svc.CreateThread(c.Context(), channelID, userID, req.Name, req.AutoArchive)
		if err != nil {
			switch err {
			case services.ErrChannelNotFound:
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "channel not found"})
			case services.ErrNotServerMember:
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "not a server member"})
			case services.ErrInvalidAutoArchive:
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid auto archive duration"})
			default:
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create thread"})
			}
		}

		return c.Status(fiber.StatusCreated).JSON(thread)
	})

	// Get Channel Threads
	app.Get("/channels/:id/threads", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		channelID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid channel id"})
		}

		includeArchived := c.QueryBool("include_archived", false)

		threads, err := svc.GetChannelThreads(c.Context(), channelID, userID, includeArchived)
		if err != nil {
			switch err {
			case services.ErrChannelNotFound:
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "channel not found"})
			case services.ErrNotServerMember:
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "not a server member"})
			default:
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to get threads"})
			}
		}

		return c.JSON(threads)
	})

	// Get Thread
	app.Get("/threads/:id", func(c *fiber.Ctx) error {
		threadID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid thread id"})
		}

		thread, err := svc.GetThread(c.Context(), threadID)
		if err != nil {
			if err == services.ErrThreadNotFound {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "thread not found"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to get thread"})
		}

		return c.JSON(thread)
	})

	// Get Thread Messages
	app.Get("/threads/:id/messages", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		threadID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid thread id"})
		}

		var before *uuid.UUID
		if b := c.Query("before"); b != "" {
			if id, err := uuid.Parse(b); err == nil {
				before = &id
			}
		}
		limit := c.QueryInt("limit", 50)

		messages, err := svc.GetThreadMessages(c.Context(), threadID, userID, before, limit)
		if err != nil {
			switch err {
			case services.ErrThreadNotFound:
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "thread not found"})
			case services.ErrNotServerMember:
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "not a server member"})
			default:
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to get thread messages"})
			}
		}

		return c.JSON(messages)
	})

	// Send Thread Message
	app.Post("/threads/:id/messages", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		threadID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid thread id"})
		}

		var req models.CreateThreadMessageRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}

		if req.Content == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "content is required"})
		}

		message, err := svc.SendThreadMessage(c.Context(), threadID, userID, req.Content)
		if err != nil {
			switch err {
			case services.ErrThreadNotFound:
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "thread not found"})
			case services.ErrThreadArchived:
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "thread is archived"})
			case services.ErrThreadLocked:
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "thread is locked"})
			case services.ErrNotServerMember:
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "not a server member"})
			default:
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to send message"})
			}
		}

		return c.Status(fiber.StatusCreated).JSON(message)
	})

	// Archive Thread
	app.Post("/threads/:id/archive", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		threadID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid thread id"})
		}

		if err := svc.ArchiveThread(c.Context(), threadID, userID); err != nil {
			switch err {
			case services.ErrThreadNotFound:
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "thread not found"})
			case services.ErrNotThreadOwner:
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "not authorized to archive this thread"})
			case services.ErrNotServerMember:
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "not a server member"})
			default:
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to archive thread"})
			}
		}

		return c.SendStatus(fiber.StatusNoContent)
	})

	// Unarchive Thread
	app.Post("/threads/:id/unarchive", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		threadID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid thread id"})
		}

		if err := svc.UnarchiveThread(c.Context(), threadID, userID); err != nil {
			switch err {
			case services.ErrThreadNotFound:
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "thread not found"})
			case services.ErrNotThreadOwner:
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "not authorized to unarchive this thread"})
			case services.ErrNotServerMember:
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "not a server member"})
			default:
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to unarchive thread"})
			}
		}

		return c.SendStatus(fiber.StatusNoContent)
	})

	// Join Thread
	app.Post("/threads/:id/join", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		threadID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid thread id"})
		}

		if err := svc.JoinThread(c.Context(), threadID, userID); err != nil {
			if err == services.ErrThreadNotFound {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "thread not found"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to join thread"})
		}

		return c.SendStatus(fiber.StatusNoContent)
	})

	// Leave Thread
	app.Delete("/threads/:id/members/@me", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		threadID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid thread id"})
		}

		if err := svc.LeaveThread(c.Context(), threadID, userID); err != nil {
			if err == services.ErrThreadNotFound {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "thread not found"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to leave thread"})
		}

		return c.SendStatus(fiber.StatusNoContent)
	})

	// Delete Thread
	app.Delete("/threads/:id", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		threadID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid thread id"})
		}

		if err := svc.DeleteThread(c.Context(), threadID, userID); err != nil {
			switch err {
			case services.ErrThreadNotFound:
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "thread not found"})
			case services.ErrNotThreadOwner:
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "not authorized to delete this thread"})
			case services.ErrNotServerMember:
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "not a server member"})
			default:
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete thread"})
			}
		}

		return c.SendStatus(fiber.StatusNoContent)
	})

	return app
}

func TestCreateThread(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	tests := []struct {
		name           string
		channelID      string
		body           interface{}
		setupMock      func(*mockThreadService)
		expectedStatus int
		expectedError  string
	}{
		{
			name:      "success",
			channelID: channelID.String(),
			body: map[string]interface{}{
				"name": "Test Thread",
			},
			setupMock: func(m *mockThreadService) {
				m.createThreadFunc = func(ctx context.Context, cID, uID uuid.UUID, name string, autoArchive *int) (*models.Thread, error) {
					return &models.Thread{
						ID:              uuid.New(),
						ParentChannelID: cID,
						OwnerID:         uID,
						Name:            name,
						CreatedAt:       time.Now(),
					}, nil
				}
			},
			expectedStatus: fiber.StatusCreated,
		},
		{
			name:      "success with auto archive",
			channelID: channelID.String(),
			body: map[string]interface{}{
				"name":         "Test Thread",
				"auto_archive": 60,
			},
			setupMock: func(m *mockThreadService) {
				m.createThreadFunc = func(ctx context.Context, cID, uID uuid.UUID, name string, autoArchive *int) (*models.Thread, error) {
					return &models.Thread{
						ID:              uuid.New(),
						ParentChannelID: cID,
						OwnerID:         uID,
						Name:            name,
						AutoArchive:     *autoArchive,
						CreatedAt:       time.Now(),
					}, nil
				}
			},
			expectedStatus: fiber.StatusCreated,
		},
		{
			name:           "invalid channel id",
			channelID:      "invalid",
			body:           map[string]interface{}{"name": "Test"},
			setupMock:      func(m *mockThreadService) {},
			expectedStatus: fiber.StatusBadRequest,
			expectedError:  "invalid channel id",
		},
		{
			name:           "missing name",
			channelID:      channelID.String(),
			body:           map[string]interface{}{},
			setupMock:      func(m *mockThreadService) {},
			expectedStatus: fiber.StatusBadRequest,
			expectedError:  "name is required",
		},
		{
			name:      "channel not found",
			channelID: channelID.String(),
			body:      map[string]interface{}{"name": "Test Thread"},
			setupMock: func(m *mockThreadService) {
				m.createThreadFunc = func(ctx context.Context, cID, uID uuid.UUID, name string, autoArchive *int) (*models.Thread, error) {
					return nil, services.ErrChannelNotFound
				}
			},
			expectedStatus: fiber.StatusNotFound,
			expectedError:  "channel not found",
		},
		{
			name:      "not server member",
			channelID: channelID.String(),
			body:      map[string]interface{}{"name": "Test Thread"},
			setupMock: func(m *mockThreadService) {
				m.createThreadFunc = func(ctx context.Context, cID, uID uuid.UUID, name string, autoArchive *int) (*models.Thread, error) {
					return nil, services.ErrNotServerMember
				}
			},
			expectedStatus: fiber.StatusForbidden,
			expectedError:  "not a server member",
		},
		{
			name:      "invalid auto archive",
			channelID: channelID.String(),
			body: map[string]interface{}{
				"name":         "Test Thread",
				"auto_archive": 999,
			},
			setupMock: func(m *mockThreadService) {
				m.createThreadFunc = func(ctx context.Context, cID, uID uuid.UUID, name string, autoArchive *int) (*models.Thread, error) {
					return nil, services.ErrInvalidAutoArchive
				}
			},
			expectedStatus: fiber.StatusBadRequest,
			expectedError:  "invalid auto archive duration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockThreadService{}
			tt.setupMock(svc)

			app := setupThreadTestApp(svc)

			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/channels/"+tt.channelID+"/threads", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-User-ID", userID.String())

			resp, _ := app.Test(req)

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			if tt.expectedError != "" {
				var result map[string]interface{}
				json.NewDecoder(resp.Body).Decode(&result)
				if result["error"] != tt.expectedError {
					t.Errorf("expected error %q, got %q", tt.expectedError, result["error"])
				}
			}
		})
	}
}

func TestGetThread(t *testing.T) {
	threadID := uuid.New()

	tests := []struct {
		name           string
		threadID       string
		setupMock      func(*mockThreadService)
		expectedStatus int
		expectedError  string
	}{
		{
			name:     "success",
			threadID: threadID.String(),
			setupMock: func(m *mockThreadService) {
				m.getThreadFunc = func(ctx context.Context, tID uuid.UUID) (*models.Thread, error) {
					return &models.Thread{
						ID:              tID,
						Name:            "Test Thread",
						ParentChannelID: uuid.New(),
						OwnerID:         uuid.New(),
						CreatedAt:       time.Now(),
					}, nil
				}
			},
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "invalid thread id",
			threadID:       "invalid",
			setupMock:      func(m *mockThreadService) {},
			expectedStatus: fiber.StatusBadRequest,
			expectedError:  "invalid thread id",
		},
		{
			name:     "thread not found",
			threadID: threadID.String(),
			setupMock: func(m *mockThreadService) {
				m.getThreadFunc = func(ctx context.Context, tID uuid.UUID) (*models.Thread, error) {
					return nil, services.ErrThreadNotFound
				}
			},
			expectedStatus: fiber.StatusNotFound,
			expectedError:  "thread not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockThreadService{}
			tt.setupMock(svc)

			app := setupThreadTestApp(svc)

			req := httptest.NewRequest("GET", "/threads/"+tt.threadID, nil)

			resp, _ := app.Test(req)

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			if tt.expectedError != "" {
				var result map[string]interface{}
				json.NewDecoder(resp.Body).Decode(&result)
				if result["error"] != tt.expectedError {
					t.Errorf("expected error %q, got %q", tt.expectedError, result["error"])
				}
			}
		})
	}
}

func TestGetThreadMessages(t *testing.T) {
	userID := uuid.New()
	threadID := uuid.New()

	tests := []struct {
		name           string
		threadID       string
		query          string
		setupMock      func(*mockThreadService)
		expectedStatus int
		expectedError  string
	}{
		{
			name:     "success",
			threadID: threadID.String(),
			setupMock: func(m *mockThreadService) {
				m.getThreadMessagesFunc = func(ctx context.Context, tID, uID uuid.UUID, before *uuid.UUID, limit int) ([]*models.ThreadMessage, error) {
					return []*models.ThreadMessage{
						{ID: uuid.New(), ThreadID: tID, AuthorID: uID, Content: "Hello", CreatedAt: time.Now()},
						{ID: uuid.New(), ThreadID: tID, AuthorID: uID, Content: "World", CreatedAt: time.Now()},
					}, nil
				}
			},
			expectedStatus: fiber.StatusOK,
		},
		{
			name:     "success with pagination",
			threadID: threadID.String(),
			query:    "?limit=10&before=" + uuid.New().String(),
			setupMock: func(m *mockThreadService) {
				m.getThreadMessagesFunc = func(ctx context.Context, tID, uID uuid.UUID, before *uuid.UUID, limit int) ([]*models.ThreadMessage, error) {
					if limit != 10 {
						t.Errorf("expected limit 10, got %d", limit)
					}
					if before == nil {
						t.Error("expected before to be set")
					}
					return []*models.ThreadMessage{}, nil
				}
			},
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "invalid thread id",
			threadID:       "invalid",
			setupMock:      func(m *mockThreadService) {},
			expectedStatus: fiber.StatusBadRequest,
			expectedError:  "invalid thread id",
		},
		{
			name:     "thread not found",
			threadID: threadID.String(),
			setupMock: func(m *mockThreadService) {
				m.getThreadMessagesFunc = func(ctx context.Context, tID, uID uuid.UUID, before *uuid.UUID, limit int) ([]*models.ThreadMessage, error) {
					return nil, services.ErrThreadNotFound
				}
			},
			expectedStatus: fiber.StatusNotFound,
			expectedError:  "thread not found",
		},
		{
			name:     "not server member",
			threadID: threadID.String(),
			setupMock: func(m *mockThreadService) {
				m.getThreadMessagesFunc = func(ctx context.Context, tID, uID uuid.UUID, before *uuid.UUID, limit int) ([]*models.ThreadMessage, error) {
					return nil, services.ErrNotServerMember
				}
			},
			expectedStatus: fiber.StatusForbidden,
			expectedError:  "not a server member",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockThreadService{}
			tt.setupMock(svc)

			app := setupThreadTestApp(svc)

			req := httptest.NewRequest("GET", "/threads/"+tt.threadID+"/messages"+tt.query, nil)
			req.Header.Set("X-User-ID", userID.String())

			resp, _ := app.Test(req)

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			if tt.expectedError != "" {
				var result map[string]interface{}
				json.NewDecoder(resp.Body).Decode(&result)
				if result["error"] != tt.expectedError {
					t.Errorf("expected error %q, got %q", tt.expectedError, result["error"])
				}
			}
		})
	}
}

func TestSendThreadMessage(t *testing.T) {
	userID := uuid.New()
	threadID := uuid.New()

	tests := []struct {
		name           string
		threadID       string
		body           interface{}
		setupMock      func(*mockThreadService)
		expectedStatus int
		expectedError  string
	}{
		{
			name:     "success",
			threadID: threadID.String(),
			body:     map[string]interface{}{"content": "Hello thread!"},
			setupMock: func(m *mockThreadService) {
				m.sendThreadMessageFunc = func(ctx context.Context, tID, aID uuid.UUID, content string) (*models.ThreadMessage, error) {
					return &models.ThreadMessage{
						ID:        uuid.New(),
						ThreadID:  tID,
						AuthorID:  aID,
						Content:   content,
						CreatedAt: time.Now(),
					}, nil
				}
			},
			expectedStatus: fiber.StatusCreated,
		},
		{
			name:           "invalid thread id",
			threadID:       "invalid",
			body:           map[string]interface{}{"content": "Hello"},
			setupMock:      func(m *mockThreadService) {},
			expectedStatus: fiber.StatusBadRequest,
			expectedError:  "invalid thread id",
		},
		{
			name:           "empty content",
			threadID:       threadID.String(),
			body:           map[string]interface{}{"content": ""},
			setupMock:      func(m *mockThreadService) {},
			expectedStatus: fiber.StatusBadRequest,
			expectedError:  "content is required",
		},
		{
			name:     "thread not found",
			threadID: threadID.String(),
			body:     map[string]interface{}{"content": "Hello"},
			setupMock: func(m *mockThreadService) {
				m.sendThreadMessageFunc = func(ctx context.Context, tID, aID uuid.UUID, content string) (*models.ThreadMessage, error) {
					return nil, services.ErrThreadNotFound
				}
			},
			expectedStatus: fiber.StatusNotFound,
			expectedError:  "thread not found",
		},
		{
			name:     "thread archived",
			threadID: threadID.String(),
			body:     map[string]interface{}{"content": "Hello"},
			setupMock: func(m *mockThreadService) {
				m.sendThreadMessageFunc = func(ctx context.Context, tID, aID uuid.UUID, content string) (*models.ThreadMessage, error) {
					return nil, services.ErrThreadArchived
				}
			},
			expectedStatus: fiber.StatusForbidden,
			expectedError:  "thread is archived",
		},
		{
			name:     "thread locked",
			threadID: threadID.String(),
			body:     map[string]interface{}{"content": "Hello"},
			setupMock: func(m *mockThreadService) {
				m.sendThreadMessageFunc = func(ctx context.Context, tID, aID uuid.UUID, content string) (*models.ThreadMessage, error) {
					return nil, services.ErrThreadLocked
				}
			},
			expectedStatus: fiber.StatusForbidden,
			expectedError:  "thread is locked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockThreadService{}
			tt.setupMock(svc)

			app := setupThreadTestApp(svc)

			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/threads/"+tt.threadID+"/messages", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-User-ID", userID.String())

			resp, _ := app.Test(req)

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			if tt.expectedError != "" {
				var result map[string]interface{}
				json.NewDecoder(resp.Body).Decode(&result)
				if result["error"] != tt.expectedError {
					t.Errorf("expected error %q, got %q", tt.expectedError, result["error"])
				}
			}
		})
	}
}

func TestArchiveThread(t *testing.T) {
	userID := uuid.New()
	threadID := uuid.New()

	tests := []struct {
		name           string
		threadID       string
		setupMock      func(*mockThreadService)
		expectedStatus int
		expectedError  string
	}{
		{
			name:     "success",
			threadID: threadID.String(),
			setupMock: func(m *mockThreadService) {
				m.archiveThreadFunc = func(ctx context.Context, tID, uID uuid.UUID) error {
					return nil
				}
			},
			expectedStatus: fiber.StatusNoContent,
		},
		{
			name:           "invalid thread id",
			threadID:       "invalid",
			setupMock:      func(m *mockThreadService) {},
			expectedStatus: fiber.StatusBadRequest,
			expectedError:  "invalid thread id",
		},
		{
			name:     "thread not found",
			threadID: threadID.String(),
			setupMock: func(m *mockThreadService) {
				m.archiveThreadFunc = func(ctx context.Context, tID, uID uuid.UUID) error {
					return services.ErrThreadNotFound
				}
			},
			expectedStatus: fiber.StatusNotFound,
			expectedError:  "thread not found",
		},
		{
			name:     "not thread owner",
			threadID: threadID.String(),
			setupMock: func(m *mockThreadService) {
				m.archiveThreadFunc = func(ctx context.Context, tID, uID uuid.UUID) error {
					return services.ErrNotThreadOwner
				}
			},
			expectedStatus: fiber.StatusForbidden,
			expectedError:  "not authorized to archive this thread",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockThreadService{}
			tt.setupMock(svc)

			app := setupThreadTestApp(svc)

			req := httptest.NewRequest("POST", "/threads/"+tt.threadID+"/archive", nil)
			req.Header.Set("X-User-ID", userID.String())

			resp, _ := app.Test(req)

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			if tt.expectedError != "" {
				var result map[string]interface{}
				json.NewDecoder(resp.Body).Decode(&result)
				if result["error"] != tt.expectedError {
					t.Errorf("expected error %q, got %q", tt.expectedError, result["error"])
				}
			}
		})
	}
}

func TestUnarchiveThread(t *testing.T) {
	userID := uuid.New()
	threadID := uuid.New()

	tests := []struct {
		name           string
		threadID       string
		setupMock      func(*mockThreadService)
		expectedStatus int
		expectedError  string
	}{
		{
			name:     "success",
			threadID: threadID.String(),
			setupMock: func(m *mockThreadService) {
				m.unarchiveThreadFunc = func(ctx context.Context, tID, uID uuid.UUID) error {
					return nil
				}
			},
			expectedStatus: fiber.StatusNoContent,
		},
		{
			name:           "invalid thread id",
			threadID:       "invalid",
			setupMock:      func(m *mockThreadService) {},
			expectedStatus: fiber.StatusBadRequest,
			expectedError:  "invalid thread id",
		},
		{
			name:     "thread not found",
			threadID: threadID.String(),
			setupMock: func(m *mockThreadService) {
				m.unarchiveThreadFunc = func(ctx context.Context, tID, uID uuid.UUID) error {
					return services.ErrThreadNotFound
				}
			},
			expectedStatus: fiber.StatusNotFound,
			expectedError:  "thread not found",
		},
		{
			name:     "not thread owner",
			threadID: threadID.String(),
			setupMock: func(m *mockThreadService) {
				m.unarchiveThreadFunc = func(ctx context.Context, tID, uID uuid.UUID) error {
					return services.ErrNotThreadOwner
				}
			},
			expectedStatus: fiber.StatusForbidden,
			expectedError:  "not authorized to unarchive this thread",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockThreadService{}
			tt.setupMock(svc)

			app := setupThreadTestApp(svc)

			req := httptest.NewRequest("POST", "/threads/"+tt.threadID+"/unarchive", nil)
			req.Header.Set("X-User-ID", userID.String())

			resp, _ := app.Test(req)

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			if tt.expectedError != "" {
				var result map[string]interface{}
				json.NewDecoder(resp.Body).Decode(&result)
				if result["error"] != tt.expectedError {
					t.Errorf("expected error %q, got %q", tt.expectedError, result["error"])
				}
			}
		})
	}
}

func TestGetChannelThreads(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	tests := []struct {
		name           string
		channelID      string
		query          string
		setupMock      func(*mockThreadService)
		expectedStatus int
		expectedError  string
	}{
		{
			name:      "success",
			channelID: channelID.String(),
			setupMock: func(m *mockThreadService) {
				m.getChannelThreadsFunc = func(ctx context.Context, cID, uID uuid.UUID, includeArchived bool) ([]*models.Thread, error) {
					return []*models.Thread{
						{ID: uuid.New(), ParentChannelID: cID, Name: "Thread 1"},
						{ID: uuid.New(), ParentChannelID: cID, Name: "Thread 2"},
					}, nil
				}
			},
			expectedStatus: fiber.StatusOK,
		},
		{
			name:      "success with include archived",
			channelID: channelID.String(),
			query:     "?include_archived=true",
			setupMock: func(m *mockThreadService) {
				m.getChannelThreadsFunc = func(ctx context.Context, cID, uID uuid.UUID, includeArchived bool) ([]*models.Thread, error) {
					if !includeArchived {
						t.Error("expected includeArchived to be true")
					}
					return []*models.Thread{}, nil
				}
			},
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "invalid channel id",
			channelID:      "invalid",
			setupMock:      func(m *mockThreadService) {},
			expectedStatus: fiber.StatusBadRequest,
			expectedError:  "invalid channel id",
		},
		{
			name:      "channel not found",
			channelID: channelID.String(),
			setupMock: func(m *mockThreadService) {
				m.getChannelThreadsFunc = func(ctx context.Context, cID, uID uuid.UUID, includeArchived bool) ([]*models.Thread, error) {
					return nil, services.ErrChannelNotFound
				}
			},
			expectedStatus: fiber.StatusNotFound,
			expectedError:  "channel not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockThreadService{}
			tt.setupMock(svc)

			app := setupThreadTestApp(svc)

			req := httptest.NewRequest("GET", "/channels/"+tt.channelID+"/threads"+tt.query, nil)
			req.Header.Set("X-User-ID", userID.String())

			resp, _ := app.Test(req)

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			if tt.expectedError != "" {
				var result map[string]interface{}
				json.NewDecoder(resp.Body).Decode(&result)
				if result["error"] != tt.expectedError {
					t.Errorf("expected error %q, got %q", tt.expectedError, result["error"])
				}
			}
		})
	}
}

func TestJoinThread(t *testing.T) {
	userID := uuid.New()
	threadID := uuid.New()

	tests := []struct {
		name           string
		threadID       string
		setupMock      func(*mockThreadService)
		expectedStatus int
		expectedError  string
	}{
		{
			name:     "success",
			threadID: threadID.String(),
			setupMock: func(m *mockThreadService) {
				m.joinThreadFunc = func(ctx context.Context, tID, uID uuid.UUID) error {
					return nil
				}
			},
			expectedStatus: fiber.StatusNoContent,
		},
		{
			name:           "invalid thread id",
			threadID:       "invalid",
			setupMock:      func(m *mockThreadService) {},
			expectedStatus: fiber.StatusBadRequest,
			expectedError:  "invalid thread id",
		},
		{
			name:     "thread not found",
			threadID: threadID.String(),
			setupMock: func(m *mockThreadService) {
				m.joinThreadFunc = func(ctx context.Context, tID, uID uuid.UUID) error {
					return services.ErrThreadNotFound
				}
			},
			expectedStatus: fiber.StatusNotFound,
			expectedError:  "thread not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockThreadService{}
			tt.setupMock(svc)

			app := setupThreadTestApp(svc)

			req := httptest.NewRequest("POST", "/threads/"+tt.threadID+"/join", nil)
			req.Header.Set("X-User-ID", userID.String())

			resp, _ := app.Test(req)

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			if tt.expectedError != "" {
				var result map[string]interface{}
				json.NewDecoder(resp.Body).Decode(&result)
				if result["error"] != tt.expectedError {
					t.Errorf("expected error %q, got %q", tt.expectedError, result["error"])
				}
			}
		})
	}
}

func TestLeaveThread(t *testing.T) {
	userID := uuid.New()
	threadID := uuid.New()

	tests := []struct {
		name           string
		threadID       string
		setupMock      func(*mockThreadService)
		expectedStatus int
		expectedError  string
	}{
		{
			name:     "success",
			threadID: threadID.String(),
			setupMock: func(m *mockThreadService) {
				m.leaveThreadFunc = func(ctx context.Context, tID, uID uuid.UUID) error {
					return nil
				}
			},
			expectedStatus: fiber.StatusNoContent,
		},
		{
			name:           "invalid thread id",
			threadID:       "invalid",
			setupMock:      func(m *mockThreadService) {},
			expectedStatus: fiber.StatusBadRequest,
			expectedError:  "invalid thread id",
		},
		{
			name:     "thread not found",
			threadID: threadID.String(),
			setupMock: func(m *mockThreadService) {
				m.leaveThreadFunc = func(ctx context.Context, tID, uID uuid.UUID) error {
					return services.ErrThreadNotFound
				}
			},
			expectedStatus: fiber.StatusNotFound,
			expectedError:  "thread not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockThreadService{}
			tt.setupMock(svc)

			app := setupThreadTestApp(svc)

			req := httptest.NewRequest("DELETE", "/threads/"+tt.threadID+"/members/@me", nil)
			req.Header.Set("X-User-ID", userID.String())

			resp, _ := app.Test(req)

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			if tt.expectedError != "" {
				var result map[string]interface{}
				json.NewDecoder(resp.Body).Decode(&result)
				if result["error"] != tt.expectedError {
					t.Errorf("expected error %q, got %q", tt.expectedError, result["error"])
				}
			}
		})
	}
}

func TestDeleteThread(t *testing.T) {
	userID := uuid.New()
	threadID := uuid.New()

	tests := []struct {
		name           string
		threadID       string
		setupMock      func(*mockThreadService)
		expectedStatus int
		expectedError  string
	}{
		{
			name:     "success",
			threadID: threadID.String(),
			setupMock: func(m *mockThreadService) {
				m.deleteThreadFunc = func(ctx context.Context, tID, uID uuid.UUID) error {
					return nil
				}
			},
			expectedStatus: fiber.StatusNoContent,
		},
		{
			name:           "invalid thread id",
			threadID:       "invalid",
			setupMock:      func(m *mockThreadService) {},
			expectedStatus: fiber.StatusBadRequest,
			expectedError:  "invalid thread id",
		},
		{
			name:     "thread not found",
			threadID: threadID.String(),
			setupMock: func(m *mockThreadService) {
				m.deleteThreadFunc = func(ctx context.Context, tID, uID uuid.UUID) error {
					return services.ErrThreadNotFound
				}
			},
			expectedStatus: fiber.StatusNotFound,
			expectedError:  "thread not found",
		},
		{
			name:     "not thread owner",
			threadID: threadID.String(),
			setupMock: func(m *mockThreadService) {
				m.deleteThreadFunc = func(ctx context.Context, tID, uID uuid.UUID) error {
					return services.ErrNotThreadOwner
				}
			},
			expectedStatus: fiber.StatusForbidden,
			expectedError:  "not authorized to delete this thread",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockThreadService{}
			tt.setupMock(svc)

			app := setupThreadTestApp(svc)

			req := httptest.NewRequest("DELETE", "/threads/"+tt.threadID, nil)
			req.Header.Set("X-User-ID", userID.String())

			resp, _ := app.Test(req)

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			if tt.expectedError != "" {
				var result map[string]interface{}
				json.NewDecoder(resp.Body).Decode(&result)
				if result["error"] != tt.expectedError {
					t.Errorf("expected error %q, got %q", tt.expectedError, result["error"])
				}
			}
		})
	}
}
