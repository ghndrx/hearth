package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"hearth/internal/models"
	"hearth/internal/services"
)

// --- mockMentionRepository implements services.MentionRepository for testing ---

type mockMentionRepository struct {
	createFunc                  func(ctx context.Context, mention *models.Mention) error
	createBatchFunc             func(ctx context.Context, mentions []*models.Mention) error
	getByIDFunc                  func(ctx context.Context, id uuid.UUID) (*models.Mention, error)
	getMentionsWithContextFunc   func(ctx context.Context, filter *models.MentionFilter) ([]models.MentionWithContext, int, error)
	getUnreadCountFunc           func(ctx context.Context, userID uuid.UUID) (int, error)
	getStatsFunc                func(ctx context.Context, userID uuid.UUID) (*models.MentionStats, error)
	markAsReadFunc              func(ctx context.Context, id, userID uuid.UUID) error
	markAllAsReadFunc           func(ctx context.Context, userID uuid.UUID) (int, error)
	markChannelMentionsAsReadFunc func(ctx context.Context, userID, channelID uuid.UUID) (int, error)
	deleteByMessageFunc         func(ctx context.Context, messageID uuid.UUID) error
	deleteByChannelFunc         func(ctx context.Context, channelID uuid.UUID) error
	deleteByUserFunc            func(ctx context.Context, userID uuid.UUID) error
	searchFunc                  func(ctx context.Context, userID uuid.UUID, query string, limit int) ([]models.MentionWithContext, error)
}

func (m *mockMentionRepository) Create(ctx context.Context, mention *models.Mention) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, mention)
	}
	return nil
}

func (m *mockMentionRepository) CreateBatch(ctx context.Context, mentions []*models.Mention) error {
	if m.createBatchFunc != nil {
		return m.createBatchFunc(ctx, mentions)
	}
	return nil
}

func (m *mockMentionRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Mention, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockMentionRepository) GetMentionsWithContext(ctx context.Context, filter *models.MentionFilter) ([]models.MentionWithContext, int, error) {
	if m.getMentionsWithContextFunc != nil {
		return m.getMentionsWithContextFunc(ctx, filter)
	}
	return []models.MentionWithContext{}, 0, nil
}

func (m *mockMentionRepository) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	if m.getUnreadCountFunc != nil {
		return m.getUnreadCountFunc(ctx, userID)
	}
	return 0, nil
}

func (m *mockMentionRepository) GetStats(ctx context.Context, userID uuid.UUID) (*models.MentionStats, error) {
	if m.getStatsFunc != nil {
		return m.getStatsFunc(ctx, userID)
	}
	return &models.MentionStats{}, nil
}

func (m *mockMentionRepository) MarkAsRead(ctx context.Context, id, userID uuid.UUID) error {
	if m.markAsReadFunc != nil {
		return m.markAsReadFunc(ctx, id, userID)
	}
	return nil
}

func (m *mockMentionRepository) MarkAllAsRead(ctx context.Context, userID uuid.UUID) (int, error) {
	if m.markAllAsReadFunc != nil {
		return m.markAllAsReadFunc(ctx, userID)
	}
	return 0, nil
}

func (m *mockMentionRepository) MarkChannelMentionsAsRead(ctx context.Context, userID, channelID uuid.UUID) (int, error) {
	if m.markChannelMentionsAsReadFunc != nil {
		return m.markChannelMentionsAsReadFunc(ctx, userID, channelID)
	}
	return 0, nil
}

func (m *mockMentionRepository) DeleteByMessage(ctx context.Context, messageID uuid.UUID) error {
	if m.deleteByMessageFunc != nil {
		return m.deleteByMessageFunc(ctx, messageID)
	}
	return nil
}

func (m *mockMentionRepository) DeleteByChannel(ctx context.Context, channelID uuid.UUID) error {
	if m.deleteByChannelFunc != nil {
		return m.deleteByChannelFunc(ctx, channelID)
	}
	return nil
}

func (m *mockMentionRepository) DeleteByUser(ctx context.Context, userID uuid.UUID) error {
	if m.deleteByUserFunc != nil {
		return m.deleteByUserFunc(ctx, userID)
	}
	return nil
}

func (m *mockMentionRepository) Search(ctx context.Context, userID uuid.UUID, query string, limit int) ([]models.MentionWithContext, error) {
	if m.searchFunc != nil {
		return m.searchFunc(ctx, userID, query, limit)
	}
	return []models.MentionWithContext{}, nil
}

// --- testMentionsHandler helper ---

type testMentionsHandler struct {
	handler *NotificationHandler
	repo    *mockMentionRepository
	app     *fiber.App
	userID  uuid.UUID
}

func newTestMentionsHandler(tb testing.TB) *testMentionsHandler {
	repo := &mockMentionRepository{}
	svc := services.NewMentionAPIService(repo, nil)
	handler := &NotificationHandler{mentionService: svc}

	app := fiber.New()
	app.Get("/mentions", handler.GetMentions)
	app.Get("/mentions/unread/count", handler.GetUnreadCount)
	app.Get("/mentions/stats", handler.GetStats)
	app.Get("/mentions/search", handler.Search)
	app.Patch("/mentions/:id/read", handler.MarkMentionAsRead)
	app.Post("/mentions/read-all", handler.MarkAllMentionsAsRead)
	app.Post("/mentions/channel/:channelId/read-all", handler.MarkChannelMentionsAsRead)

	// Add userID middleware
	userID := uuid.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", userID)
		return c.Next()
	})

	return &testMentionsHandler{
		handler: handler,
		repo:    repo,
		app:     app,
		userID:  userID,
	}
}

// ====================================================================================================================================================
// Test MentionsHandler.GetMentions
// ====================================================================================================================================================

func TestMentionsHandler_GetMentions(t *testing.T) {
	th := newTestMentionsHandler(t)

	tests := []struct {
		name           string
		query          string
		setupMock      func(*mockMentionRepository)
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name:  "success - returns mentions list",
			query: "",
			setupMock: func(repo *mockMentionRepository) {
				channelID := uuid.New()
				repo.getMentionsWithContextFunc = func(ctx context.Context, filter *models.MentionFilter) ([]models.MentionWithContext, int, error) {
					return []models.MentionWithContext{
						{
							Mention: models.Mention{
								ID:        uuid.New(),
								UserID:    th.userID,
								ChannelID: channelID,
								MentionType: models.MentionKindUser,
							},
							AuthorName: "TestUser",
							Preview:   "Hello world",
						},
					}, 1, nil
				}
			},
			expectedStatus: fiber.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp models.MentionsListResponse
				err := json.Unmarshal(body, &resp)
				assert.NoError(t, err)
				assert.Len(t, resp.Mentions, 1)
				assert.Equal(t, 1, resp.TotalCount)
				assert.False(t, resp.HasMore)
			},
		},
		{
			name:  "success - with unread filter",
			query: "?unread=true",
			setupMock: func(repo *mockMentionRepository) {
				isUnread := true
				repo.getMentionsWithContextFunc = func(ctx context.Context, filter *models.MentionFilter) ([]models.MentionWithContext, int, error) {
					assert.NotNil(t, filter.Unread)
					assert.Equal(t, isUnread, *filter.Unread)
					return []models.MentionWithContext{}, 0, nil
				}
			},
			expectedStatus: fiber.StatusOK,
		},
		{
			name:  "success - with type filter",
			query: "?type=user",
			setupMock: func(repo *mockMentionRepository) {
				repo.getMentionsWithContextFunc = func(ctx context.Context, filter *models.MentionFilter) ([]models.MentionWithContext, int, error) {
					assert.NotNil(t, filter.MentionType)
					assert.Equal(t, models.MentionKindUser, *filter.MentionType)
					return []models.MentionWithContext{}, 0, nil
				}
			},
			expectedStatus: fiber.StatusOK,
		},
		{
			name:  "success - with channel_id filter",
			query: "?channel_id=" + uuid.New().String(),
			setupMock: func(repo *mockMentionRepository) {
				repo.getMentionsWithContextFunc = func(ctx context.Context, filter *models.MentionFilter) ([]models.MentionWithContext, int, error) {
					assert.NotNil(t, filter.ChannelID)
					return []models.MentionWithContext{}, 0, nil
				}
			},
			expectedStatus: fiber.StatusOK,
		},
		{
			name:  "success - with guild_id filter",
			query: "?guild_id=" + uuid.New().String(),
			setupMock: func(repo *mockMentionRepository) {
				repo.getMentionsWithContextFunc = func(ctx context.Context, filter *models.MentionFilter) ([]models.MentionWithContext, int, error) {
					assert.NotNil(t, filter.GuildID)
					return []models.MentionWithContext{}, 0, nil
				}
			},
			expectedStatus: fiber.StatusOK,
		},
		{
			name:  "success - with pagination",
			query: "?limit=10&offset=5",
			setupMock: func(repo *mockMentionRepository) {
				repo.getMentionsWithContextFunc = func(ctx context.Context, filter *models.MentionFilter) ([]models.MentionWithContext, int, error) {
					assert.Equal(t, 10, filter.Limit)
					assert.Equal(t, 5, filter.Offset)
					return []models.MentionWithContext{}, 0, nil
				}
			},
			expectedStatus: fiber.StatusOK,
		},
		{
			name:  "success - with before timestamp",
			query: "?before=2026-01-01T00:00:00Z",
			setupMock: func(repo *mockMentionRepository) {
				repo.getMentionsWithContextFunc = func(ctx context.Context, filter *models.MentionFilter) ([]models.MentionWithContext, int, error) {
					assert.NotNil(t, filter.Before)
					return []models.MentionWithContext{}, 0, nil
				}
			},
			expectedStatus: fiber.StatusOK,
		},
		{
			name:  "success - with after timestamp",
			query: "?after=2026-01-01T00:00:00Z",
			setupMock: func(repo *mockMentionRepository) {
				repo.getMentionsWithContextFunc = func(ctx context.Context, filter *models.MentionFilter) ([]models.MentionWithContext, int, error) {
					assert.NotNil(t, filter.After)
					return []models.MentionWithContext{}, 0, nil
				}
			},
			expectedStatus: fiber.StatusOK,
		},
		{
			name:  "success - has_more when more results available",
			query: "?limit=1",
			setupMock: func(repo *mockMentionRepository) {
				repo.getMentionsWithContextFunc = func(ctx context.Context, filter *models.MentionFilter) ([]models.MentionWithContext, int, error) {
					// Return 2 results but limit is 1, so HasMore should be true
					return []models.MentionWithContext{
						{Mention: models.Mention{ID: uuid.New()}},
						{Mention: models.Mention{ID: uuid.New()}},
					}, 2, nil
				}
			},
			expectedStatus: fiber.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp models.MentionsListResponse
				err := json.Unmarshal(body, &resp)
				assert.NoError(t, err)
				assert.Len(t, resp.Mentions, 1) // truncated to limit
				assert.True(t, resp.HasMore)
			},
		},
		{
			name:  "service error",
			query: "",
			setupMock: func(repo *mockMentionRepository) {
				repo.getMentionsWithContextFunc = func(ctx context.Context, filter *models.MentionFilter) ([]models.MentionWithContext, int, error) {
					return nil, 0, errors.New("database error")
				}
			},
			expectedStatus: fiber.StatusInternalServerError,
		},
		{
			name:           "unauthorized - no user in context",
			query:          "",
			setupMock:      func(repo *mockMentionRepository) {},
			expectedStatus: fiber.StatusUnauthorized,
		},
	}

	// Override the middleware for the unauthorized test
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th.repo = &mockMentionRepository{} // fresh mock
			tt.setupMock(th.repo)
			svc := services.NewMentionAPIService(th.repo, nil)
			th.handler = &NotificationHandler{mentionService: svc}

			app := fiber.New()
			if tt.expectedStatus == fiber.StatusUnauthorized {
				// No userID set - simulates unauthorized
				app.Get("/mentions", th.handler.GetMentions)
			} else {
				app.Use(func(c *fiber.Ctx) error {
					c.Locals("userID", th.userID)
					return c.Next()
				})
				app.Get("/mentions", th.handler.GetMentions)
			}

			req := httptest.NewRequest("GET", "/mentions"+tt.query, nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.checkResponse != nil {
				body := make([]byte, resp.ContentLength)
				resp.Body.Read(body)
				tt.checkResponse(t, body)
			}
		})
	}
}

// ====================================================================================================================================================
// Test MentionsHandler.GetUnreadCount
// ====================================================================================================================================================

func TestMentionsHandler_GetUnreadCount(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*mockMentionRepository)
		setupUser      bool
		expectedStatus int
		expectedCount  int
	}{
		{
			name: "success - returns count",
			setupMock: func(repo *mockMentionRepository) {
				repo.getUnreadCountFunc = func(ctx context.Context, userID uuid.UUID) (int, error) {
					return 5, nil
				}
			},
			setupUser:      true,
			expectedStatus: fiber.StatusOK,
			expectedCount:  5,
		},
		{
			name: "success - zero count",
			setupMock: func(repo *mockMentionRepository) {
				repo.getUnreadCountFunc = func(ctx context.Context, userID uuid.UUID) (int, error) {
					return 0, nil
				}
			},
			setupUser:      true,
			expectedStatus: fiber.StatusOK,
			expectedCount:  0,
		},
		{
			name:           "service error",
			setupMock: func(repo *mockMentionRepository) {
				repo.getUnreadCountFunc = func(ctx context.Context, userID uuid.UUID) (int, error) {
					return 0, errors.New("database error")
				}
			},
			setupUser:      true,
			expectedStatus: fiber.StatusInternalServerError,
		},
		{
			name:           "unauthorized",
			setupMock:      func(repo *mockMentionRepository) {},
			setupUser:      false,
			expectedStatus: fiber.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockMentionRepository{}
			tt.setupMock(repo)
			svc := services.NewMentionAPIService(repo, nil)
			handler := &NotificationHandler{mentionService: svc}

			app := fiber.New()
			if tt.setupUser {
				userID := uuid.New()
				app.Use(func(c *fiber.Ctx) error {
					c.Locals("userID", userID)
					return c.Next()
				})
			}
			app.Get("/mentions/unread/count", handler.GetUnreadCount)

			req := httptest.NewRequest("GET", "/mentions/unread/count", nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == fiber.StatusOK {
				var result map[string]int
				json.NewDecoder(resp.Body).Decode(&result)
				assert.Equal(t, tt.expectedCount, result["count"])
			}
		})
	}
}

// ====================================================================================================================================================
// Test MentionsHandler.GetStats
// ====================================================================================================================================================

func TestMentionsHandler_GetStats(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*mockMentionRepository)
		setupUser      bool
		expectedStatus int
	}{
		{
			name: "success - returns stats",
			setupMock: func(repo *mockMentionRepository) {
				repo.getStatsFunc = func(ctx context.Context, userID uuid.UUID) (*models.MentionStats, error) {
					return &models.MentionStats{
						TotalCount:  100,
						UnreadCount: 10,
						TodayCount:  5,
					}, nil
				}
			},
			setupUser:      true,
			expectedStatus: fiber.StatusOK,
		},
		{
			name: "service error",
			setupMock: func(repo *mockMentionRepository) {
				repo.getStatsFunc = func(ctx context.Context, userID uuid.UUID) (*models.MentionStats, error) {
					return nil, errors.New("database error")
				}
			},
			setupUser:      true,
			expectedStatus: fiber.StatusInternalServerError,
		},
		{
			name:           "unauthorized",
			setupMock:      func(repo *mockMentionRepository) {},
			setupUser:      false,
			expectedStatus: fiber.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockMentionRepository{}
			tt.setupMock(repo)
			svc := services.NewMentionAPIService(repo, nil)
			handler := &NotificationHandler{mentionService: svc}

			app := fiber.New()
			if tt.setupUser {
				userID := uuid.New()
				app.Use(func(c *fiber.Ctx) error {
					c.Locals("userID", userID)
					return c.Next()
				})
			}
			app.Get("/mentions/stats", handler.GetStats)

			req := httptest.NewRequest("GET", "/mentions/stats", nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

// ====================================================================================================================================================
// Test MentionsHandler.MarkAsRead
// ====================================================================================================================================================

func TestMentionsHandler_MarkAsRead(t *testing.T) {
	mentionID := uuid.New()
	testUserID := uuid.New() // Known user ID for tests that need user context

	tests := []struct {
		name           string
		mentionIDParam string
		setupMock      func(*mockMentionRepository, uuid.UUID)
		setupUser      bool
		expectedStatus int
	}{
		{
			name:           "success",
			mentionIDParam: mentionID.String(),
			setupMock: func(repo *mockMentionRepository, userID uuid.UUID) {
				repo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Mention, error) {
					return &models.Mention{ID: id, UserID: userID}, nil
				}
				repo.markAsReadFunc = func(ctx context.Context, id, userID uuid.UUID) error {
					return nil
				}
			},
			setupUser:      true,
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "mention not found",
			mentionIDParam: mentionID.String(),
			setupMock: func(repo *mockMentionRepository, userID uuid.UUID) {
				repo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Mention, error) {
					return nil, nil // not found
				}
			},
			setupUser:      true,
			expectedStatus: fiber.StatusNotFound,
		},
		{
			name:           "mention not found error",
			mentionIDParam: mentionID.String(),
			setupMock: func(repo *mockMentionRepository, userID uuid.UUID) {
				repo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Mention, error) {
					return nil, services.ErrMentionNotFound
				}
			},
			setupUser:      true,
			expectedStatus: fiber.StatusNotFound,
		},
		{
			name:           "invalid mention id",
			mentionIDParam: "not-a-uuid",
			setupMock:      func(repo *mockMentionRepository, userID uuid.UUID) {},
			setupUser:      true,
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:           "service error on mark as read",
			mentionIDParam: mentionID.String(),
			setupMock: func(repo *mockMentionRepository, userID uuid.UUID) {
				repo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Mention, error) {
					return &models.Mention{ID: id, UserID: userID}, nil
				}
				repo.markAsReadFunc = func(ctx context.Context, id, userID uuid.UUID) error {
					return errors.New("database error")
				}
			},
			setupUser:      true,
			expectedStatus: fiber.StatusInternalServerError,
		},
		{
			name:           "unauthorized",
			mentionIDParam: mentionID.String(),
			setupMock:      func(repo *mockMentionRepository, userID uuid.UUID) {},
			setupUser:      false,
			expectedStatus: fiber.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockMentionRepository{}
			tt.setupMock(repo, testUserID)
			svc := services.NewMentionAPIService(repo, nil)
			handler := &NotificationHandler{mentionService: svc}

			app := fiber.New()
			if tt.setupUser {
				app.Use(func(c *fiber.Ctx) error {
					c.Locals("userID", testUserID)
					return c.Next()
				})
			}
			app.Patch("/mentions/:id/read", handler.MarkMentionAsRead)

			req := httptest.NewRequest("PATCH", "/mentions/"+tt.mentionIDParam+"/read", nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

// ====================================================================================================================================================
// Test MentionsHandler.MarkAllAsRead
// ====================================================================================================================================================

func TestMentionsHandler_MarkAllAsRead(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*mockMentionRepository)
		setupUser      bool
		expectedStatus int
		expectedCount  int
	}{
		{
			name: "success - marks all as read",
			setupMock: func(repo *mockMentionRepository) {
				repo.markAllAsReadFunc = func(ctx context.Context, userID uuid.UUID) (int, error) {
					return 10, nil
				}
			},
			setupUser:      true,
			expectedStatus: fiber.StatusOK,
			expectedCount:  10,
		},
		{
			name: "success - zero count",
			setupMock: func(repo *mockMentionRepository) {
				repo.markAllAsReadFunc = func(ctx context.Context, userID uuid.UUID) (int, error) {
					return 0, nil
				}
			},
			setupUser:      true,
			expectedStatus: fiber.StatusOK,
			expectedCount:  0,
		},
		{
			name: "service error",
			setupMock: func(repo *mockMentionRepository) {
				repo.markAllAsReadFunc = func(ctx context.Context, userID uuid.UUID) (int, error) {
					return 0, errors.New("database error")
				}
			},
			setupUser:      true,
			expectedStatus: fiber.StatusInternalServerError,
		},
		{
			name:           "unauthorized",
			setupMock:      func(repo *mockMentionRepository) {},
			setupUser:      false,
			expectedStatus: fiber.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockMentionRepository{}
			tt.setupMock(repo)
			svc := services.NewMentionAPIService(repo, nil)
			handler := &NotificationHandler{mentionService: svc}

			app := fiber.New()
			if tt.setupUser {
				userID := uuid.New()
				app.Use(func(c *fiber.Ctx) error {
					c.Locals("userID", userID)
					return c.Next()
				})
			}
			app.Post("/mentions/read-all", handler.MarkAllMentionsAsRead)

			req := httptest.NewRequest("POST", "/mentions/read-all", nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == fiber.StatusOK {
				var result map[string]interface{}
				json.NewDecoder(resp.Body).Decode(&result)
				assert.Equal(t, true, result["success"])
				assert.Equal(t, float64(tt.expectedCount), result["count"])
			}
		})
	}
}

// ====================================================================================================================================================
// Test MentionsHandler.MarkChannelMentionsAsRead
// ====================================================================================================================================================

func TestMentionsHandler_MarkChannelMentionsAsRead(t *testing.T) {
	channelID := uuid.New()

	tests := []struct {
		name           string
		channelIDParam string
		setupMock      func(*mockMentionRepository)
		setupUser      bool
		expectedStatus int
		expectedCount  int
	}{
		{
			name:           "success",
			channelIDParam: channelID.String(),
			setupMock: func(repo *mockMentionRepository) {
				repo.markChannelMentionsAsReadFunc = func(ctx context.Context, userID, cID uuid.UUID) (int, error) {
					return 5, nil
				}
			},
			setupUser:      true,
			expectedStatus: fiber.StatusOK,
			expectedCount:  5,
		},
		{
			name:           "invalid channel id",
			channelIDParam: "not-a-uuid",
			setupMock:      func(repo *mockMentionRepository) {},
			setupUser:      true,
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:           "service error",
			channelIDParam: channelID.String(),
			setupMock: func(repo *mockMentionRepository) {
				repo.markChannelMentionsAsReadFunc = func(ctx context.Context, userID, cID uuid.UUID) (int, error) {
					return 0, errors.New("database error")
				}
			},
			setupUser:      true,
			expectedStatus: fiber.StatusInternalServerError,
		},
		{
			name:           "unauthorized",
			channelIDParam: channelID.String(),
			setupMock:      func(repo *mockMentionRepository) {},
			setupUser:      false,
			expectedStatus: fiber.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockMentionRepository{}
			tt.setupMock(repo)
			svc := services.NewMentionAPIService(repo, nil)
			handler := &NotificationHandler{mentionService: svc}

			app := fiber.New()
			if tt.setupUser {
				userID := uuid.New()
				app.Use(func(c *fiber.Ctx) error {
					c.Locals("userID", userID)
					return c.Next()
				})
			}
			app.Post("/mentions/channel/:channelId/read-all", handler.MarkChannelMentionsAsRead)

			req := httptest.NewRequest("POST", "/mentions/channel/"+tt.channelIDParam+"/read-all", nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == fiber.StatusOK {
				var result map[string]interface{}
				json.NewDecoder(resp.Body).Decode(&result)
				assert.Equal(t, true, result["success"])
				assert.Equal(t, float64(tt.expectedCount), result["count"])
			}
		})
	}
}

// ====================================================================================================================================================
// Test MentionsHandler.Search
// ====================================================================================================================================================

func TestMentionsHandler_Search(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		setupMock      func(*mockMentionRepository)
		setupUser      bool
		expectedStatus int
	}{
		{
			name:  "success - returns results",
			query: "?q=test",
			setupMock: func(repo *mockMentionRepository) {
				repo.searchFunc = func(ctx context.Context, userID uuid.UUID, query string, limit int) ([]models.MentionWithContext, error) {
					return []models.MentionWithContext{
						{Mention: models.Mention{ID: uuid.New()}},
					}, nil
				}
			},
			setupUser:      true,
			expectedStatus: fiber.StatusOK,
		},
		{
			name:  "success - empty results",
			query: "?q=nonexistent",
			setupMock: func(repo *mockMentionRepository) {
				repo.searchFunc = func(ctx context.Context, userID uuid.UUID, query string, limit int) ([]models.MentionWithContext, error) {
					return []models.MentionWithContext{}, nil
				}
			},
			setupUser:      true,
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "missing query parameter",
			query:          "",
			setupMock:      func(repo *mockMentionRepository) {},
			setupUser:      true,
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:  "service error",
			query: "?q=test",
			setupMock: func(repo *mockMentionRepository) {
				repo.searchFunc = func(ctx context.Context, userID uuid.UUID, query string, limit int) ([]models.MentionWithContext, error) {
					return nil, errors.New("search error")
				}
			},
			setupUser:      true,
			expectedStatus: fiber.StatusInternalServerError,
		},
		{
			name:  "with custom limit",
			query: "?q=test&limit=50",
			setupMock: func(repo *mockMentionRepository) {
				repo.searchFunc = func(ctx context.Context, userID uuid.UUID, query string, limit int) ([]models.MentionWithContext, error) {
					assert.Equal(t, 50, limit)
					return []models.MentionWithContext{}, nil
				}
			},
			setupUser:      true,
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "unauthorized",
			query:          "?q=test",
			setupMock:      func(repo *mockMentionRepository) {},
			setupUser:      false,
			expectedStatus: fiber.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockMentionRepository{}
			tt.setupMock(repo)
			svc := services.NewMentionAPIService(repo, nil)
			handler := &NotificationHandler{mentionService: svc}

			app := fiber.New()
			if tt.setupUser {
				userID := uuid.New()
				app.Use(func(c *fiber.Ctx) error {
					c.Locals("userID", userID)
					return c.Next()
				})
			}
			app.Get("/mentions/search", handler.Search)

			req := httptest.NewRequest("GET", "/mentions/search"+tt.query, nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == fiber.StatusOK && tt.query != "" {
				var result map[string]interface{}
				json.NewDecoder(resp.Body).Decode(&result)
				assert.NotNil(t, result["mentions"])
				assert.NotNil(t, result["count"])
			}
		})
	}
}

// ====================================================================================================================================================
// Test MentionsHandler query parameter edge cases
// ====================================================================================================================================================

func TestMentionsHandler_GetMentions_QueryParamEdgeCases(t *testing.T) {
	t.Run("invalid unread param treated as false", func(t *testing.T) {
		repo := &mockMentionRepository{}
		repo.getMentionsWithContextFunc = func(ctx context.Context, filter *models.MentionFilter) ([]models.MentionWithContext, int, error) {
			// unread=invalid should be treated as false
			if filter.Unread != nil {
				assert.False(t, *filter.Unread)
			}
			return []models.MentionWithContext{}, 0, nil
		}
		svc := services.NewMentionAPIService(repo, nil)
		handler := &NotificationHandler{mentionService: svc}

		app := fiber.New()
		app.Use(func(c *fiber.Ctx) error {
			c.Locals("userID", uuid.New())
			return c.Next()
		})
		app.Get("/mentions", handler.GetMentions)

		req := httptest.NewRequest("GET", "/mentions?unread=invalid", nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("invalid channel_id param ignored", func(t *testing.T) {
		repo := &mockMentionRepository{}
		repo.getMentionsWithContextFunc = func(ctx context.Context, filter *models.MentionFilter) ([]models.MentionWithContext, int, error) {
			// Invalid UUID should result in nil ChannelID
			assert.Nil(t, filter.ChannelID)
			return []models.MentionWithContext{}, 0, nil
		}
		svc := services.NewMentionAPIService(repo, nil)
		handler := &NotificationHandler{mentionService: svc}

		app := fiber.New()
		app.Use(func(c *fiber.Ctx) error {
			c.Locals("userID", uuid.New())
			return c.Next()
		})
		app.Get("/mentions", handler.GetMentions)

		req := httptest.NewRequest("GET", "/mentions?channel_id=invalid-uuid", nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("invalid guild_id param ignored", func(t *testing.T) {
		repo := &mockMentionRepository{}
		repo.getMentionsWithContextFunc = func(ctx context.Context, filter *models.MentionFilter) ([]models.MentionWithContext, int, error) {
			assert.Nil(t, filter.GuildID)
			return []models.MentionWithContext{}, 0, nil
		}
		svc := services.NewMentionAPIService(repo, nil)
		handler := &NotificationHandler{mentionService: svc}

		app := fiber.New()
		app.Use(func(c *fiber.Ctx) error {
			c.Locals("userID", uuid.New())
			return c.Next()
		})
		app.Get("/mentions", handler.GetMentions)

		req := httptest.NewRequest("GET", "/mentions?guild_id=invalid-uuid", nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("default limit when not specified", func(t *testing.T) {
		repo := &mockMentionRepository{}
		repo.getMentionsWithContextFunc = func(ctx context.Context, filter *models.MentionFilter) ([]models.MentionWithContext, int, error) {
			assert.Equal(t, 50, filter.Limit) // default limit
			return []models.MentionWithContext{}, 0, nil
		}
		svc := services.NewMentionAPIService(repo, nil)
		handler := &NotificationHandler{mentionService: svc}

		app := fiber.New()
		app.Use(func(c *fiber.Ctx) error {
			c.Locals("userID", uuid.New())
			return c.Next()
		})
		app.Get("/mentions", handler.GetMentions)

		req := httptest.NewRequest("GET", "/mentions", nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("default offset when not specified", func(t *testing.T) {
		repo := &mockMentionRepository{}
		repo.getMentionsWithContextFunc = func(ctx context.Context, filter *models.MentionFilter) ([]models.MentionWithContext, int, error) {
			assert.Equal(t, 0, filter.Offset)
			return []models.MentionWithContext{}, 0, nil
		}
		svc := services.NewMentionAPIService(repo, nil)
		handler := &NotificationHandler{mentionService: svc}

		app := fiber.New()
		app.Use(func(c *fiber.Ctx) error {
			c.Locals("userID", uuid.New())
			return c.Next()
		})
		app.Get("/mentions", handler.GetMentions)

		req := httptest.NewRequest("GET", "/mentions", nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})
}