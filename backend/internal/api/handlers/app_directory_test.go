package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"hearth/internal/models"
	"hearth/internal/services"
)

// --- Mock AppDirectoryRepository ---

// MockAppDirectoryRepository is a mock implementation of services.AppDirectoryRepository
type MockAppDirectoryRepository struct {
	mock.Mock
}

func (m *MockAppDirectoryRepository) CreateApp(ctx context.Context, app *models.App) error {
	args := m.Called(ctx, app)
	return args.Error(0)
}

func (m *MockAppDirectoryRepository) GetAppByID(ctx context.Context, id uuid.UUID) (*models.App, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.App), args.Error(1)
}

func (m *MockAppDirectoryRepository) GetAppByIDWithDeveloper(ctx context.Context, id uuid.UUID) (*models.App, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.App), args.Error(1)
}

func (m *MockAppDirectoryRepository) UpdateApp(ctx context.Context, app *models.App) error {
	args := m.Called(ctx, app)
	return args.Error(0)
}

func (m *MockAppDirectoryRepository) UpdateAppStatus(ctx context.Context, appID uuid.UUID, status models.AppStatus) error {
	args := m.Called(ctx, appID, status)
	return args.Error(0)
}

func (m *MockAppDirectoryRepository) DeleteApp(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAppDirectoryRepository) ListApps(ctx context.Context, params services.ListAppsParams) ([]*models.App, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.App), args.Error(1)
}

func (m *MockAppDirectoryRepository) ListDeveloperApps(ctx context.Context, developerID uuid.UUID) ([]*models.App, error) {
	args := m.Called(ctx, developerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.App), args.Error(1)
}

func (m *MockAppDirectoryRepository) InstallApp(ctx context.Context, appID, serverID, installerID uuid.UUID) error {
	args := m.Called(ctx, appID, serverID, installerID)
	return args.Error(0)
}

func (m *MockAppDirectoryRepository) UninstallApp(ctx context.Context, appID, serverID uuid.UUID) error {
	args := m.Called(ctx, appID, serverID)
	return args.Error(0)
}

func (m *MockAppDirectoryRepository) GetServerInstallations(ctx context.Context, serverID uuid.UUID) ([]*models.AppInstallation, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.AppInstallation), args.Error(1)
}

func (m *MockAppDirectoryRepository) IsAppInstalled(ctx context.Context, appID, serverID uuid.UUID) (bool, error) {
	args := m.Called(ctx, appID, serverID)
	return args.Bool(0), args.Error(1)
}

func (m *MockAppDirectoryRepository) CreateReview(ctx context.Context, review *models.AppReview) error {
	args := m.Called(ctx, review)
	return args.Error(0)
}

func (m *MockAppDirectoryRepository) UpdateReview(ctx context.Context, review *models.AppReview) error {
	args := m.Called(ctx, review)
	return args.Error(0)
}

func (m *MockAppDirectoryRepository) DeleteReview(ctx context.Context, reviewID, userID uuid.UUID) (uuid.UUID, error) {
	args := m.Called(ctx, reviewID, userID)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockAppDirectoryRepository) GetReviewByID(ctx context.Context, id uuid.UUID) (*models.AppReview, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AppReview), args.Error(1)
}

func (m *MockAppDirectoryRepository) ListAppReviews(ctx context.Context, appID uuid.UUID, limit, offset int) ([]*models.AppReview, error) {
	args := m.Called(ctx, appID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.AppReview), args.Error(1)
}

func (m *MockAppDirectoryRepository) GetUserReviewForApp(ctx context.Context, appID, userID uuid.UUID) (*models.AppReview, error) {
	args := m.Called(ctx, appID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AppReview), args.Error(1)
}

func (m *MockAppDirectoryRepository) AddDeveloperTeamMember(ctx context.Context, appID, userID uuid.UUID, role string) error {
	args := m.Called(ctx, appID, userID, role)
	return args.Error(0)
}

func (m *MockAppDirectoryRepository) RemoveDeveloperTeamMember(ctx context.Context, appID, userID uuid.UUID) error {
	args := m.Called(ctx, appID, userID)
	return args.Error(0)
}

func (m *MockAppDirectoryRepository) GetDeveloperRole(ctx context.Context, appID, userID uuid.UUID) (string, error) {
	args := m.Called(ctx, appID, userID)
	return args.String(0), args.Error(1)
}

func (m *MockAppDirectoryRepository) IsAppDeveloper(ctx context.Context, appID, userID uuid.UUID) (bool, error) {
	args := m.Called(ctx, appID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockAppDirectoryRepository) ListDeveloperTeamMembers(ctx context.Context, appID uuid.UUID) ([]*models.AppDeveloperTeamMember, error) {
	args := m.Called(ctx, appID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.AppDeveloperTeamMember), args.Error(1)
}

func (m *MockAppDirectoryRepository) GetDeveloperAnalytics(ctx context.Context, developerID uuid.UUID) (*models.AppDeveloperAnalytics, error) {
	args := m.Called(ctx, developerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AppDeveloperAnalytics), args.Error(1)
}

// --- Test App Setup Helper ---

func setupAppDirectoryTestApp(t testing.TB, repo *MockAppDirectoryRepository) (*fiber.App, *AppDirectoryHandler) {
	app := fiber.New()
	t.Cleanup(func() { _ = app.Shutdown() })

	app.Use(func(c *fiber.Ctx) error {
		userID := c.Get("X-Test-User-ID")
		if userID != "" {
			uid, _ := uuid.Parse(userID)
			c.Locals("userID", uid)
		}
		return c.Next()
	})

	svc := services.NewAppDirectoryService(repo)
	handler := NewAppDirectoryHandler(svc)

	apps := app.Group("/api/v1/apps")
	// Static routes first (more specific)
	apps.Get("/categories", handler.ListCategories)
	apps.Get("/developers/me/apps", handler.ListDeveloperApps)
	apps.Get("/developers/me/analytics", handler.GetDeveloperAnalytics)
	// Then parameterized routes
	apps.Get("/", handler.ListApps)
	apps.Get("/:id", handler.GetApp)
	apps.Post("/", handler.CreateApp)
	apps.Patch("/:id", handler.UpdateApp)
	apps.Delete("/:id", handler.DeleteApp)
	apps.Post("/:id/install/:serverId", handler.InstallApp)
	apps.Post("/:id/uninstall/:serverId", handler.UninstallApp)
	apps.Get("/:id/reviews", handler.ListAppReviews)
	apps.Post("/:id/reviews", handler.CreateReview)
	apps.Patch("/reviews/:reviewId", handler.UpdateReview)
	apps.Delete("/reviews/:reviewId", handler.DeleteReview)
	apps.Get("/:id/my-review", handler.GetMyReviewForApp)
	apps.Post("/:id/approve", handler.ApproveApp)
	apps.Post("/:id/reject", handler.RejectApp)
	apps.Post("/:id/suspend", handler.SuspendApp)

	return app, handler
}

// --- Handler Tests for ListApps ---

func TestAppDirectoryHandler_ListApps_Success(t *testing.T) {
	repo := new(MockAppDirectoryRepository)
	app, _ := setupAppDirectoryTestApp(t, repo)

	apps := []*models.App{
		{
			ID:          uuid.New(),
			Name:        "TestApp",
			Description: "A test application",
			Category:    models.AppCategoryGaming,
			Status:     models.AppStatusApproved,
		},
	}

	repo.On("ListApps", mock.Anything, mock.MatchedBy(func(p services.ListAppsParams) bool {
		return p.ApprovedOnly == true && p.Limit == 20 && p.Offset == 0
	})).Return(apps, nil)

	req := httptest.NewRequest("GET", "/api/v1/apps", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	repo.AssertExpectations(t)
}

func TestAppDirectoryHandler_ListApps_Empty(t *testing.T) {
	repo := new(MockAppDirectoryRepository)
	app, _ := setupAppDirectoryTestApp(t, repo)

	repo.On("ListApps", mock.Anything, mock.Anything).Return([]*models.App{}, nil)

	req := httptest.NewRequest("GET", "/api/v1/apps", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	repo.AssertExpectations(t)
}

func TestAppDirectoryHandler_ListApps_ServiceError(t *testing.T) {
	repo := new(MockAppDirectoryRepository)
	app, _ := setupAppDirectoryTestApp(t, repo)

	repo.On("ListApps", mock.Anything, mock.Anything).Return(nil, assert.AnError)

	req := httptest.NewRequest("GET", "/api/v1/apps", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)

	repo.AssertExpectations(t)
}

// --- Handler Tests for GetApp ---

func TestAppDirectoryHandler_GetApp_Success(t *testing.T) {
	repo := new(MockAppDirectoryRepository)
	app, _ := setupAppDirectoryTestApp(t, repo)

	appID := uuid.New()
	testApp := &models.App{
		ID:          appID,
		Name:        "TestApp",
		Description: "A test application",
		Category:    models.AppCategoryGaming,
		Status:     models.AppStatusApproved,
	}

	repo.On("GetAppByIDWithDeveloper", mock.Anything, appID).Return(testApp, nil)

	req := httptest.NewRequest("GET", "/api/v1/apps/"+appID.String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var result models.App
	err = json.Unmarshal(body, &result)
	require.NoError(t, err)
	assert.Equal(t, "TestApp", result.Name)

	repo.AssertExpectations(t)
}

func TestAppDirectoryHandler_GetApp_NotFound(t *testing.T) {
	repo := new(MockAppDirectoryRepository)
	app, _ := setupAppDirectoryTestApp(t, repo)

	appID := uuid.New()
	repo.On("GetAppByIDWithDeveloper", mock.Anything, appID).Return(nil, services.ErrAppNotFound)

	req := httptest.NewRequest("GET", "/api/v1/apps/"+appID.String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)

	repo.AssertExpectations(t)
}

func TestAppDirectoryHandler_GetApp_InvalidID(t *testing.T) {
	repo := new(MockAppDirectoryRepository)
	app, _ := setupAppDirectoryTestApp(t, repo)

	req := httptest.NewRequest("GET", "/api/v1/apps/invalid-uuid", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

// --- Handler Tests for CreateApp ---

func TestAppDirectoryHandler_CreateApp_Success(t *testing.T) {
	repo := new(MockAppDirectoryRepository)
	app, _ := setupAppDirectoryTestApp(t, repo)

	userID := uuid.New()

	repo.On("AddDeveloperTeamMember", mock.Anything, mock.AnythingOfType("uuid.UUID"), userID, models.AppDeveloperRoleOwner).Return(nil)
	repo.On("CreateApp", mock.Anything, mock.AnythingOfType("*models.App")).Return(nil)

	body := map[string]interface{}{
		"name":        "MyNewApp",
		"description": "This is a great new app for everyone",
		"category":    "gaming",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/apps", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	repo.AssertExpectations(t)
}

func TestAppDirectoryHandler_CreateApp_Unauthorized(t *testing.T) {
	repo := new(MockAppDirectoryRepository)
	app, _ := setupAppDirectoryTestApp(t, repo)

	body := map[string]interface{}{
		"name":        "MyNewApp",
		"description": "This is a great new app for everyone",
		"category":    "gaming",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/apps", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestAppDirectoryHandler_CreateApp_InvalidName_TooShort(t *testing.T) {
	repo := new(MockAppDirectoryRepository)
	app, _ := setupAppDirectoryTestApp(t, repo)

	userID := uuid.New()

	body := map[string]interface{}{
		"name":        "A",
		"description": "This is a great new app for everyone",
		"category":    "gaming",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/apps", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestAppDirectoryHandler_CreateApp_InvalidCategory(t *testing.T) {
	repo := new(MockAppDirectoryRepository)
	app, _ := setupAppDirectoryTestApp(t, repo)

	userID := uuid.New()

	body := map[string]interface{}{
		"name":        "MyNewApp",
		"description": "This is a great new app for everyone",
		"category":    "invalid_category",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/apps", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

// --- Handler Tests for ListCategories ---

func TestAppDirectoryHandler_ListCategories_Success(t *testing.T) {
	repo := new(MockAppDirectoryRepository)
	app, _ := setupAppDirectoryTestApp(t, repo)

	req := httptest.NewRequest("GET", "/api/v1/apps/categories", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var result map[string][]string
	err = json.Unmarshal(body, &result)
	require.NoError(t, err)
	assert.Contains(t, result["categories"], "gaming")
	assert.Contains(t, result["categories"], "moderation")
	assert.Contains(t, result["categories"], "music")
}

// --- Model Tests ---

func TestAppCategory_ParseAndValidate(t *testing.T) {
	t.Run("valid categories", func(t *testing.T) {
		cats := []string{"gaming", "moderation", "music", "utility", "fun", "education", "roleplay", "economy"}
		for _, cat := range cats {
			category, ok := models.ParseAppCategory(cat)
			assert.True(t, ok, "expected %s to be valid", cat)
			assert.GreaterOrEqual(t, int(category), 0)
		}
	})

	t.Run("invalid category", func(t *testing.T) {
		_, ok := models.ParseAppCategory("invalid")
		assert.False(t, ok)
	})
}

func TestAppStatus_Values(t *testing.T) {
	t.Run("status values", func(t *testing.T) {
		assert.Equal(t, 0, int(models.AppStatusPending))
		assert.Equal(t, 1, int(models.AppStatusApproved))
		assert.Equal(t, 2, int(models.AppStatusRejected))
		assert.Equal(t, 3, int(models.AppStatusSuspended))
	})
}

func TestAppDeveloperRole_Values(t *testing.T) {
	t.Run("developer role values", func(t *testing.T) {
		assert.Equal(t, "owner", models.AppDeveloperRoleOwner)
		assert.Equal(t, "admin", models.AppDeveloperRoleAdmin)
		assert.Equal(t, "member", models.AppDeveloperRoleMember)
	})
}

func TestApp_InstallCount(t *testing.T) {
	t.Run("install count defaults to 0", func(t *testing.T) {
		app := &models.App{
			ID:          uuid.New(),
			Name:        "TestApp",
			Description: "Test",
			Category:    models.AppCategoryGaming,
		}
		assert.Equal(t, 0, app.InstallCount)
	})
}

func TestAppInstallation_PopulatedFields(t *testing.T) {
	t.Run("app and server can be populated", func(t *testing.T) {
		installation := &models.AppInstallation{
			AppID:    uuid.New(),
			ServerID: uuid.New(),
		}
		installation.App = &models.App{
			ID:   installation.AppID,
			Name: "TestApp",
		}
		installation.Server = &models.Server{
			ID:   installation.ServerID,
			Name: "TestServer",
		}
		assert.NotNil(t, installation.App)
		assert.NotNil(t, installation.Server)
		assert.Equal(t, "TestApp", installation.App.Name)
		assert.Equal(t, "TestServer", installation.Server.Name)
	})
}

func TestAppReview_WithOptionalFields(t *testing.T) {
	t.Run("review text is optional", func(t *testing.T) {
		review := &models.AppReview{
			ID:     uuid.New(),
			AppID:  uuid.New(),
			UserID: uuid.New(),
			Rating: 5,
		}
		assert.Nil(t, review.ReviewText)
		assert.Nil(t, review.User)
	})

	t.Run("review with text and user", func(t *testing.T) {
		reviewText := "Great app!"
		review := &models.AppReview{
			ID:         uuid.New(),
			AppID:      uuid.New(),
			UserID:     uuid.New(),
			Rating:     5,
			ReviewText: &reviewText,
		}
		assert.NotNil(t, review.ReviewText)
		assert.Equal(t, "Great app!", *review.ReviewText)
	})
}

func TestAppDeveloperTeamMember_Roles(t *testing.T) {
	t.Run("team member with different roles", func(t *testing.T) {
		member := &models.AppDeveloperTeamMember{
			AppID:  uuid.New(),
			UserID: uuid.New(),
			Role:   models.AppDeveloperRoleOwner,
		}
		assert.Equal(t, models.AppDeveloperRoleOwner, member.Role)
	})
}

func TestCreateAppRequest_Validation(t *testing.T) {
	t.Run("valid request", func(t *testing.T) {
		req := &models.CreateAppRequest{
			Name:        "TestApp",
			Description: "This is a test application",
			Category:    "gaming",
			Tags:        pq.StringArray{"tag1", "tag2"},
		}
		assert.NotEmpty(t, req.Name)
		assert.NotEmpty(t, req.Description)
		assert.NotEmpty(t, req.Category)
	})

	t.Run("request with optional fields", func(t *testing.T) {
		longDesc := "This is a very long description"
		iconURL := "https://example.com/icon.png"
		req := &models.CreateAppRequest{
			Name:            "TestApp",
			Description:     "This is a test application",
			Category:        "gaming",
			LongDescription: &longDesc,
			IconURL:         &iconURL,
		}
		assert.NotNil(t, req.LongDescription)
		assert.NotNil(t, req.IconURL)
	})
}

func TestUpdateAppRequest_PartialUpdate(t *testing.T) {
	t.Run("partial update with only name", func(t *testing.T) {
		newName := "UpdatedApp"
		req := &models.UpdateAppRequest{
			Name: &newName,
		}
		assert.NotNil(t, req.Name)
		assert.Nil(t, req.Description)
		assert.Nil(t, req.Category)
	})

	t.Run("partial update with multiple fields", func(t *testing.T) {
		newName := "UpdatedApp"
		newDesc := "Updated description"
		req := &models.UpdateAppRequest{
			Name:        &newName,
			Description: &newDesc,
		}
		assert.NotNil(t, req.Name)
		assert.NotNil(t, req.Description)
		assert.Nil(t, req.Category)
	})
}

func TestListAppsRequest_Defaults(t *testing.T) {
	t.Run("zero values for new struct", func(t *testing.T) {
		req := &models.ListAppsRequest{}
		assert.Equal(t, "", req.Category)
		assert.Equal(t, "", req.Query)
		assert.Equal(t, false, req.Featured)
		assert.Equal(t, 0, req.Limit)  // Go defaults int to 0
		assert.Equal(t, 0, req.Offset)
	})
}

func TestAppDeveloperAnalytics_Fields(t *testing.T) {
	t.Run("analytics with all fields", func(t *testing.T) {
		analytics := &models.AppDeveloperAnalytics{
			TotalApps:     5,
			TotalInstalls: 10000,
			TotalReviews:  200,
			AverageRating: 4.5,
		}
		assert.Equal(t, 5, analytics.TotalApps)
		assert.Equal(t, 10000, analytics.TotalInstalls)
		assert.Equal(t, 200, analytics.TotalReviews)
		assert.Equal(t, 4.5, analytics.AverageRating)
	})
}
