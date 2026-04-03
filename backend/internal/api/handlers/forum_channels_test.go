package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"hearth/internal/models"
	"hearth/internal/services"
)

// mockThreadServiceForForumChannels implements ThreadServiceInterface for forum channel tests
type mockThreadServiceForForumChannels struct {
	getChannelThreadsPaginatedFunc func(ctx context.Context, channelID, requesterID uuid.UUID, sortOrder int, limit, offset int, includeArchived bool) ([]models.Thread, int, error)
	createThreadFunc               func(ctx context.Context, channelID, creatorID uuid.UUID, name string, autoArchive *int, parentMessageID *uuid.UUID) (*models.Thread, error)
}

func (m *mockThreadServiceForForumChannels) GetChannelThreadsPaginated(ctx context.Context, channelID, requesterID uuid.UUID, sortOrder int, limit, offset int, includeArchived bool) ([]models.Thread, int, error) {
	if m.getChannelThreadsPaginatedFunc != nil {
		return m.getChannelThreadsPaginatedFunc(ctx, channelID, requesterID, sortOrder, limit, offset, includeArchived)
	}
	return nil, 0, nil
}

func (m *mockThreadServiceForForumChannels) CreateThread(ctx context.Context, channelID, creatorID uuid.UUID, name string, autoArchive *int, parentMessageID *uuid.UUID) (*models.Thread, error) {
	if m.createThreadFunc != nil {
		return m.createThreadFunc(ctx, channelID, creatorID, name, autoArchive, parentMessageID)
	}
	return nil, nil
}

func (m *mockThreadServiceForForumChannels) GetChannelThreads(ctx context.Context, channelID, requesterID uuid.UUID, includeArchived bool) ([]*models.Thread, error) {
	return nil, nil
}

func (m *mockThreadServiceForForumChannels) UpdateThread(ctx context.Context, threadID, requesterID uuid.UUID, req models.UpdateThreadRequest) (*models.Thread, error) {
	return nil, nil
}

func (m *mockThreadServiceForForumChannels) GetThread(ctx context.Context, threadID uuid.UUID) (*models.Thread, error) {
	return nil, nil
}

func (m *mockThreadServiceForForumChannels) GetThreadMessages(ctx context.Context, threadID, requesterID uuid.UUID, before *uuid.UUID, limit int) ([]*models.ThreadMessage, error) {
	return nil, nil
}

func (m *mockThreadServiceForForumChannels) SendThreadMessage(ctx context.Context, threadID, authorID uuid.UUID, content string) (*models.ThreadMessage, error) {
	return nil, nil
}

func (m *mockThreadServiceForForumChannels) ArchiveThread(ctx context.Context, threadID, requesterID uuid.UUID) error {
	return nil
}

func (m *mockThreadServiceForForumChannels) UnarchiveThread(ctx context.Context, threadID, requesterID uuid.UUID) error {
	return nil
}

func (m *mockThreadServiceForForumChannels) JoinThread(ctx context.Context, threadID, userID uuid.UUID) error {
	return nil
}

func (m *mockThreadServiceForForumChannels) LeaveThread(ctx context.Context, threadID, userID uuid.UUID) error {
	return nil
}

func (m *mockThreadServiceForForumChannels) DeleteThread(ctx context.Context, threadID, requesterID uuid.UUID) error {
	return nil
}

func (m *mockThreadServiceForForumChannels) GetNotificationPreference(ctx context.Context, threadID, userID uuid.UUID) (*models.ThreadNotificationPreference, error) {
	return nil, nil
}

func (m *mockThreadServiceForForumChannels) SetNotificationPreference(ctx context.Context, threadID, userID uuid.UUID, level models.ThreadNotificationLevel) error {
	return nil
}

func (m *mockThreadServiceForForumChannels) EnterThread(ctx context.Context, threadID, userID uuid.UUID) (*models.ThreadPresenceResponse, error) {
	return nil, nil
}

func (m *mockThreadServiceForForumChannels) ExitThread(ctx context.Context, threadID, userID uuid.UUID) error {
	return nil
}

func (m *mockThreadServiceForForumChannels) GetActiveViewers(ctx context.Context, threadID uuid.UUID) (*models.ThreadPresenceResponse, error) {
	return nil, nil
}

func (m *mockThreadServiceForForumChannels) HeartbeatPresence(ctx context.Context, threadID, userID uuid.UUID) error {
	return nil
}

// mockForumTagServiceForForumChannels implements ForumTagServiceInterface for forum channel tests
type mockForumTagServiceForForumChannels struct {
	createTagFunc      func(ctx context.Context, channelID, userID uuid.UUID, req *models.CreateForumTagRequest) (*models.ForumTag, error)
	updateTagFunc      func(ctx context.Context, tagID, userID uuid.UUID, req *models.UpdateForumTagRequest) (*models.ForumTag, error)
	deleteTagFunc      func(ctx context.Context, tagID, userID uuid.UUID) error
	getChannelTagsFunc func(ctx context.Context, channelID uuid.UUID) ([]models.ForumTag, error)
}

func (m *mockForumTagServiceForForumChannels) CreateTag(ctx context.Context, channelID, userID uuid.UUID, req *models.CreateForumTagRequest) (*models.ForumTag, error) {
	if m.createTagFunc != nil {
		return m.createTagFunc(ctx, channelID, userID, req)
	}
	return nil, nil
}

func (m *mockForumTagServiceForForumChannels) UpdateTag(ctx context.Context, tagID, userID uuid.UUID, req *models.UpdateForumTagRequest) (*models.ForumTag, error) {
	if m.updateTagFunc != nil {
		return m.updateTagFunc(ctx, tagID, userID, req)
	}
	return nil, nil
}

func (m *mockForumTagServiceForForumChannels) DeleteTag(ctx context.Context, tagID, userID uuid.UUID) error {
	if m.deleteTagFunc != nil {
		return m.deleteTagFunc(ctx, tagID, userID)
	}
	return nil
}

func (m *mockForumTagServiceForForumChannels) GetChannelTags(ctx context.Context, channelID uuid.UUID) ([]models.ForumTag, error) {
	if m.getChannelTagsFunc != nil {
		return m.getChannelTagsFunc(ctx, channelID)
	}
	return nil, nil
}

func (m *mockForumTagServiceForForumChannels) ApplyTagsToThread(ctx context.Context, threadID, userID uuid.UUID, tagIDs []uuid.UUID) error {
	return nil
}

func (m *mockForumTagServiceForForumChannels) GetThreadTags(ctx context.Context, threadID uuid.UUID) ([]models.ForumTag, error) {
	return nil, nil
}

func (m *mockForumTagServiceForForumChannels) PinThread(ctx context.Context, threadID, userID uuid.UUID, pin bool) error {
	return nil
}

func (m *mockForumTagServiceForForumChannels) FilterForumPosts(ctx context.Context, channelID uuid.UUID, filter *models.ForumPostFilter, limit, offset int) ([]models.Thread, []models.ForumTag, int, error) {
	return nil, nil, 0, nil
}

func setupForumChannelsApp(mockThread *mockThreadServiceForForumChannels, mockTag *mockForumTagServiceForForumChannels) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			var httpErr *HTTPError
			if errors.As(err, &httpErr) {
				return c.Status(httpErr.Status).JSON(fiber.Map{
					"error":   httpErr.ErrorType,
					"message": httpErr.Message,
					"code":    httpErr.Code,
				})
			}
			// Fallback for other errors
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error":   "internal_error",
				"message": err.Error(),
			})
		},
	})

	// Mock auth middleware
	app.Use(func(c *fiber.Ctx) error {
		userIDStr := c.Get("X-Test-User-ID", uuid.New().String())
		uid, _ := uuid.Parse(userIDStr)
		c.Locals("userID", uid)
		return c.Next()
	})

	// Thread routes
	app.Get("/channels/:id/threads", func(c *fiber.Ctx) error {
		channelID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid channel ID"})
		}
		userID := c.Locals("userID").(uuid.UUID)
		includeArchived := c.QueryBool("include_archived", false)
		sortOrder := c.QueryInt("sort", 0)
		limit := c.QueryInt("limit", 25)
		offset := c.QueryInt("offset", 0)

		threads, total, err := mockThread.getChannelThreadsPaginatedFunc(c.Context(), channelID, userID, sortOrder, limit, offset, includeArchived)
		if err != nil {
			return HandleServiceError(c, err)
		}

		return c.JSON(fiber.Map{
			"threads":  threads,
			"total":    total,
			"has_more": offset+len(threads) < total,
		})
	})

	app.Post("/channels/:id/threads", func(c *fiber.Ctx) error {
		channelID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid channel ID"})
		}
		userID := c.Locals("userID").(uuid.UUID)

		var req models.CreateThreadRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}

		// Validate request
		if req.Name == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
		}

		if mockThread.createThreadFunc == nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "mock not configured"})
		}

		var parentMsgID *uuid.UUID
		if req.ParentMessageID != nil {
			if parsed, err := uuid.Parse(*req.ParentMessageID); err == nil {
				parentMsgID = &parsed
			}
		}

		thread, err := mockThread.createThreadFunc(c.Context(), channelID, userID, req.Name, req.AutoArchive, parentMsgID)
		if err != nil {
			return HandleServiceError(c, err)
		}

		return c.Status(fiber.StatusCreated).JSON(thread)
	})

	// Tag routes
	app.Get("/channels/:id/tags", func(c *fiber.Ctx) error {
		channelID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid channel ID"})
		}
		tags, err := mockTag.getChannelTagsFunc(c.Context(), channelID)
		if err != nil {
			return HandleServiceError(c, err)
		}
		return c.JSON(fiber.Map{"tags": tags})
	})

	app.Post("/channels/:id/tags", func(c *fiber.Ctx) error {
		channelID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid channel ID"})
		}
		userID := c.Locals("userID").(uuid.UUID)

		var req models.CreateForumTagRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}

		tag, err := mockTag.createTagFunc(c.Context(), channelID, userID, &req)
		if err != nil {
			return HandleServiceError(c, err)
		}

		return c.Status(fiber.StatusCreated).JSON(tag)
	})

	app.Patch("/channels/:id/tags/:tagId", func(c *fiber.Ctx) error {
		tagID, err := uuid.Parse(c.Params("tagId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid tag ID"})
		}
		userID := c.Locals("userID").(uuid.UUID)

		var req models.UpdateForumTagRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}

		tag, err := mockTag.updateTagFunc(c.Context(), tagID, userID, &req)
		if err != nil {
			return HandleServiceError(c, err)
		}

		return c.JSON(tag)
	})

	app.Delete("/channels/:id/tags/:tagId", func(c *fiber.Ctx) error {
		tagID, err := uuid.Parse(c.Params("tagId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid tag ID"})
		}
		userID := c.Locals("userID").(uuid.UUID)

		err = mockTag.deleteTagFunc(c.Context(), tagID, userID)
		if err != nil {
			return HandleServiceError(c, err)
		}

		return c.SendStatus(fiber.StatusNoContent)
	})

	return app
}

// ============================================================================
// Thread Endpoint Tests
// ============================================================================

func TestGetChannelThreads_Success(t *testing.T) {
	channelID := uuid.New()
	threadID := uuid.New()
	userID := uuid.New()

	mockThread := &mockThreadServiceForForumChannels{
		getChannelThreadsPaginatedFunc: func(ctx context.Context, cID, uID uuid.UUID, sortOrder int, limit, offset int, includeArchived bool) ([]models.Thread, int, error) {
			assert.Equal(t, channelID, cID)
			assert.Equal(t, userID, uID)
			assert.Equal(t, 0, sortOrder)
			assert.Equal(t, 25, limit)
			return []models.Thread{
				{
					ID:              threadID,
					ParentChannelID: channelID,
					Name:            "Test Thread",
					MessageCount:    5,
					MemberCount:     2,
					OwnerID:         userID,
				},
			}, 1, nil
		},
	}

	mockTag := &mockForumTagServiceForForumChannels{}

	app := setupForumChannelsApp(mockThread, mockTag)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest(http.MethodGet, "/channels/"+channelID.String()+"/threads", nil)
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.NotNil(t, result["threads"])
	assert.Equal(t, float64(1), result["total"])
	assert.Equal(t, false, result["has_more"])
}

func TestGetChannelThreads_WithPagination(t *testing.T) {
	channelID := uuid.New()
	userID := uuid.New()

	mockThread := &mockThreadServiceForForumChannels{
		getChannelThreadsPaginatedFunc: func(ctx context.Context, cID, uID uuid.UUID, sortOrder int, limit, offset int, includeArchived bool) ([]models.Thread, int, error) {
			assert.Equal(t, channelID, cID)
			assert.Equal(t, 1, sortOrder)   // sort by creation_date
			assert.Equal(t, 10, limit)
			assert.Equal(t, 20, offset)
			assert.Equal(t, true, includeArchived)

			threads := make([]models.Thread, 10)
			for i := 0; i < 10; i++ {
				threads[i] = models.Thread{
					ID:              uuid.New(),
					ParentChannelID: channelID,
					Name:            "Thread",
				}
			}
			return threads, 31, nil
		},
	}

	mockTag := &mockForumTagServiceForForumChannels{}

	app := setupForumChannelsApp(mockThread, mockTag)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest(http.MethodGet, "/channels/"+channelID.String()+"/threads?sort=1&limit=10&offset=20&include_archived=true", nil)
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, true, result["has_more"]) // 20+10 < 31
}

func TestGetChannelThreads_InvalidChannelID(t *testing.T) {
	mockThread := &mockThreadServiceForForumChannels{}
	mockTag := &mockForumTagServiceForForumChannels{}

	app := setupForumChannelsApp(mockThread, mockTag)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest(http.MethodGet, "/channels/invalid-id/threads", nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestCreateThread_Success(t *testing.T) {
	channelID := uuid.New()
	userID := uuid.New()
	threadID := uuid.New()

	mockThread := &mockThreadServiceForForumChannels{
		createThreadFunc: func(ctx context.Context, cID, creatorID uuid.UUID, name string, autoArchive *int, parentMessageID *uuid.UUID) (*models.Thread, error) {
			assert.Equal(t, channelID, cID)
			assert.Equal(t, userID, creatorID)
			assert.Equal(t, "New Thread", name)
			assert.Nil(t, parentMessageID)
			return &models.Thread{
				ID:              threadID,
				ParentChannelID: channelID,
				OwnerID:         userID,
				Name:            name,
			}, nil
		},
	}

	mockTag := &mockForumTagServiceForForumChannels{}

	app := setupForumChannelsApp(mockThread, mockTag)
	t.Cleanup(func() { _ = app.Shutdown() })

	reqBody := `{"name": "New Thread"}`
	req := httptest.NewRequest(http.MethodPost, "/channels/"+channelID.String()+"/threads", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
}

func TestCreateThread_InvalidRequest(t *testing.T) {
	channelID := uuid.New()
	userID := uuid.New()

	mockThread := &mockThreadServiceForForumChannels{}
	mockTag := &mockForumTagServiceForForumChannels{}

	app := setupForumChannelsApp(mockThread, mockTag)
	t.Cleanup(func() { _ = app.Shutdown() })

	// Empty body
	req := httptest.NewRequest(http.MethodPost, "/channels/"+channelID.String()+"/threads", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ============================================================================
// Tag Endpoint Tests
// ============================================================================

func TestForumChannelListTags_Success(t *testing.T) {
	channelID := uuid.New()
	tagID := uuid.New()

	mockThread := &mockThreadServiceForForumChannels{}
	mockTag := &mockForumTagServiceForForumChannels{
		getChannelTagsFunc: func(ctx context.Context, cID uuid.UUID) ([]models.ForumTag, error) {
			assert.Equal(t, channelID, cID)
			return []models.ForumTag{
				{ID: tagID, ChannelID: channelID, Name: "bug", Position: 0},
				{ID: uuid.New(), ChannelID: channelID, Name: "feature", Position: 1},
			}, nil
		},
	}

	app := setupForumChannelsApp(mockThread, mockTag)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest(http.MethodGet, "/channels/"+channelID.String()+"/tags", nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.NotNil(t, result["tags"])
}

func TestForumChannelCreateTag_Success(t *testing.T) {
	channelID := uuid.New()
	userID := uuid.New()
	tagID := uuid.New()

	mockThread := &mockThreadServiceForForumChannels{}
	mockTag := &mockForumTagServiceForForumChannels{
		createTagFunc: func(ctx context.Context, cID, uID uuid.UUID, req *models.CreateForumTagRequest) (*models.ForumTag, error) {
			assert.Equal(t, channelID, cID)
			assert.Equal(t, userID, uID)
			assert.Equal(t, "Help Needed", req.Name)
			return &models.ForumTag{
				ID:        tagID,
				ChannelID: channelID,
				Name:      req.Name,
				Position:  0,
			}, nil
		},
	}

	app := setupForumChannelsApp(mockThread, mockTag)
	t.Cleanup(func() { _ = app.Shutdown() })

	reqBody := `{"name": "Help Needed"}`
	req := httptest.NewRequest(http.MethodPost, "/channels/"+channelID.String()+"/tags", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
}

func TestForumChannelCreateTag_WithPosition(t *testing.T) {
	channelID := uuid.New()
	userID := uuid.New()
	tagID := uuid.New()
	position := 2

	mockThread := &mockThreadServiceForForumChannels{}
	mockTag := &mockForumTagServiceForForumChannels{
		createTagFunc: func(ctx context.Context, cID, uID uuid.UUID, req *models.CreateForumTagRequest) (*models.ForumTag, error) {
			assert.Equal(t, channelID, cID)
			assert.Equal(t, userID, uID)
			assert.Equal(t, "Question", req.Name)
			assert.NotNil(t, req.Position)
			assert.Equal(t, position, *req.Position)
			return &models.ForumTag{
				ID:        tagID,
				ChannelID: channelID,
				Name:      req.Name,
				Position:  position,
			}, nil
		},
	}

	app := setupForumChannelsApp(mockThread, mockTag)
	t.Cleanup(func() { _ = app.Shutdown() })

	reqBody := `{"name": "Question", "position": 2}`
	req := httptest.NewRequest(http.MethodPost, "/channels/"+channelID.String()+"/tags", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
}

func TestForumChannelUpdateTag_Success(t *testing.T) {
	channelID := uuid.New()
	userID := uuid.New()
	tagID := uuid.New()

	mockThread := &mockThreadServiceForForumChannels{}
	mockTag := &mockForumTagServiceForForumChannels{
		updateTagFunc: func(ctx context.Context, tID, uID uuid.UUID, req *models.UpdateForumTagRequest) (*models.ForumTag, error) {
			assert.Equal(t, tagID, tID)
			assert.Equal(t, userID, uID)
			assert.NotNil(t, req.Name)
			assert.Equal(t, "Updated Name", *req.Name)
			return &models.ForumTag{
				ID:        tagID,
				ChannelID: channelID,
				Name:      *req.Name,
				Position:  0,
			}, nil
		},
	}

	app := setupForumChannelsApp(mockThread, mockTag)
	t.Cleanup(func() { _ = app.Shutdown() })

	reqBody := `{"name": "Updated Name"}`
	req := httptest.NewRequest(http.MethodPatch, "/channels/"+channelID.String()+"/tags/"+tagID.String(), strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestForumChannelDeleteTag_Success(t *testing.T) {
	channelID := uuid.New()
	userID := uuid.New()
	tagID := uuid.New()

	mockThread := &mockThreadServiceForForumChannels{}
	mockTag := &mockForumTagServiceForForumChannels{
		deleteTagFunc: func(ctx context.Context, tID, uID uuid.UUID) error {
			assert.Equal(t, tagID, tID)
			assert.Equal(t, userID, uID)
			return nil
		},
	}

	app := setupForumChannelsApp(mockThread, mockTag)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest(http.MethodDelete, "/channels/"+channelID.String()+"/tags/"+tagID.String(), nil)
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestForumChannelDeleteTag_NotFound(t *testing.T) {
	channelID := uuid.New()
	userID := uuid.New()
	tagID := uuid.New()

	mockThread := &mockThreadServiceForForumChannels{}
	mockTag := &mockForumTagServiceForForumChannels{
		deleteTagFunc: func(ctx context.Context, tID, uID uuid.UUID) error {
			return services.ErrTagNotFound
		},
	}

	app := setupForumChannelsApp(mockThread, mockTag)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest(http.MethodDelete, "/channels/"+channelID.String()+"/tags/"+tagID.String(), nil)
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	
	// Read response body for debugging
	body := make([]byte, 1000)
	n, _ := resp.Body.Read(body)
	t.Logf("Response body: %s", string(body[:n]))
	t.Logf("Response status: %d", resp.StatusCode)
	
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// ============================================================================
// Channel Type Field Tests
// ============================================================================

func TestChannelResponse_HasChannelTypeField(t *testing.T) {
	// Test that Channel struct has ChannelType field
	channel := models.Channel{
		ID:   uuid.New(),
		Type: models.ChannelTypeForum,
		Name: "test-forum",
	}

	// Verify ChannelType is set
	channel.ChannelType = channel.Type

	assert.Equal(t, models.ChannelTypeForum, channel.Type)
	assert.Equal(t, models.ChannelTypeForum, channel.ChannelType)
}

// ============================================================================
// Forum Post Filter Tests
// ============================================================================

func TestForumPostFilter_Struct(t *testing.T) {
	filter := &models.ForumPostFilter{
		TagIDs:    []uuid.UUID{uuid.New(), uuid.New()},
		SortOrder: 0,
	}

	assert.Len(t, filter.TagIDs, 2)
	assert.Equal(t, 0, filter.SortOrder)
}

// ============================================================================
// Permission Constants Tests
// ============================================================================

func TestForumPermissions(t *testing.T) {
	// Test that forum permissions are defined
	assert.Equal(t, int64(1<<36), models.PermCreateForumPosts)
	assert.Equal(t, int64(1<<37), models.PermParticipateInForums)

	// Test that permissions are included in default
	assert.True(t, models.DefaultPermissions&models.PermCreateForumPosts != 0)
	assert.True(t, models.DefaultPermissions&models.PermParticipateInForums != 0)

	// Test that permissions are included in all
	assert.True(t, models.PermissionAll&models.PermCreateForumPosts != 0)
	assert.True(t, models.PermissionAll&models.PermParticipateInForums != 0)
}
