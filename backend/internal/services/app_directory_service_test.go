// File: services/app_directory_service_test.go
package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"hearth/internal/models"
)

// MockAppDirectoryRepository is a mock implementation of AppDirectoryRepository
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

func (m *MockAppDirectoryRepository) ListApps(ctx context.Context, params ListAppsParams) ([]*models.App, error) {
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

func TestAppDirectoryService_SubmitApp_Success(t *testing.T) {
	mockRepo := new(MockAppDirectoryRepository)
	service := NewAppDirectoryService(mockRepo)

	developerID := uuid.New()
	req := &models.CreateAppRequest{
		Name:        "Test Bot",
		Description: "A test bot for testing",
		Category:    "moderation",
	}

	mockRepo.On("CreateApp", mock.Anything, mock.AnythingOfType("*models.App")).Return(nil).Once()
	mockRepo.On("AddDeveloperTeamMember", mock.Anything, mock.AnythingOfType("uuid.UUID"), developerID, models.AppDeveloperRoleOwner).Return(nil).Once()

	app, err := service.SubmitApp(context.Background(), req, developerID)

	assert.NoError(t, err)
	assert.NotNil(t, app)
	assert.Equal(t, req.Name, app.Name)
	assert.Equal(t, req.Description, app.Description)
	assert.Equal(t, developerID, app.DeveloperID)
	assert.Equal(t, models.AppStatusPending, app.Status)
	mockRepo.AssertExpectations(t)
}

func TestAppDirectoryService_SubmitApp_InvalidCategory(t *testing.T) {
	mockRepo := new(MockAppDirectoryRepository)
	service := NewAppDirectoryService(mockRepo)

	developerID := uuid.New()
	req := &models.CreateAppRequest{
		Name:        "Test Bot",
		Description: "A test bot for testing",
		Category:    "invalid_category",
	}

	app, err := service.SubmitApp(context.Background(), req, developerID)

	assert.Error(t, err)
	assert.Nil(t, app)
	assert.Equal(t, ErrInvalidAppCategory, err)
}

func TestAppDirectoryService_GetApp_Success(t *testing.T) {
	mockRepo := new(MockAppDirectoryRepository)
	service := NewAppDirectoryService(mockRepo)

	appID := uuid.New()
	expectedApp := &models.App{
		ID:          appID,
		Name:        "Test Bot",
		Description: "A test bot",
		Status:      models.AppStatusApproved,
	}

	mockRepo.On("GetAppByIDWithDeveloper", mock.Anything, appID).Return(expectedApp, nil).Once()

	app, err := service.GetApp(context.Background(), appID)

	assert.NoError(t, err)
	assert.NotNil(t, app)
	assert.Equal(t, expectedApp.ID, app.ID)
	mockRepo.AssertExpectations(t)
}

func TestAppDirectoryService_GetApp_NotFound(t *testing.T) {
	mockRepo := new(MockAppDirectoryRepository)
	service := NewAppDirectoryService(mockRepo)

	appID := uuid.New()

	mockRepo.On("GetAppByIDWithDeveloper", mock.Anything, appID).Return(nil, nil).Once()

	app, err := service.GetApp(context.Background(), appID)

	assert.Error(t, err)
	assert.Nil(t, app)
	assert.Equal(t, ErrAppNotFound, err)
	mockRepo.AssertExpectations(t)
}

func TestAppDirectoryService_InstallApp_Success(t *testing.T) {
	mockRepo := new(MockAppDirectoryRepository)
	service := NewAppDirectoryService(mockRepo)

	appID := uuid.New()
	serverID := uuid.New()
	installerID := uuid.New()

	app := &models.App{
		ID:     appID,
		Status: models.AppStatusApproved,
	}

	mockRepo.On("GetAppByID", mock.Anything, appID).Return(app, nil).Once()
	mockRepo.On("IsAppInstalled", mock.Anything, appID, serverID).Return(false, nil).Once()
	mockRepo.On("InstallApp", mock.Anything, appID, serverID, installerID).Return(nil).Once()

	err := service.InstallApp(context.Background(), appID, serverID, installerID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAppDirectoryService_InstallApp_AlreadyInstalled(t *testing.T) {
	mockRepo := new(MockAppDirectoryRepository)
	service := NewAppDirectoryService(mockRepo)

	appID := uuid.New()
	serverID := uuid.New()
	installerID := uuid.New()

	app := &models.App{
		ID:     appID,
		Status: models.AppStatusApproved,
	}

	mockRepo.On("GetAppByID", mock.Anything, appID).Return(app, nil).Once()
	mockRepo.On("IsAppInstalled", mock.Anything, appID, serverID).Return(true, nil).Once()

	err := service.InstallApp(context.Background(), appID, serverID, installerID)

	assert.Error(t, err)
	assert.Equal(t, ErrAlreadyInstalled, err)
	mockRepo.AssertExpectations(t)
}

func TestAppDirectoryService_InstallApp_NotApproved(t *testing.T) {
	mockRepo := new(MockAppDirectoryRepository)
	service := NewAppDirectoryService(mockRepo)

	appID := uuid.New()
	serverID := uuid.New()
	installerID := uuid.New()

	app := &models.App{
		ID:     appID,
		Status: models.AppStatusPending,
	}

	mockRepo.On("GetAppByID", mock.Anything, appID).Return(app, nil).Once()

	err := service.InstallApp(context.Background(), appID, serverID, installerID)

	assert.Error(t, err)
	assert.Equal(t, ErrAppNotApproved, err)
	mockRepo.AssertExpectations(t)
}

func TestAppDirectoryService_CreateReview_Success(t *testing.T) {
	mockRepo := new(MockAppDirectoryRepository)
	service := NewAppDirectoryService(mockRepo)

	appID := uuid.New()
	userID := uuid.New()
	req := &models.CreateReviewRequest{
		Rating:     5,
		ReviewText: "Great app!",
	}

	app := &models.App{
		ID:     appID,
		Status: models.AppStatusApproved,
	}

	mockRepo.On("GetAppByID", mock.Anything, appID).Return(app, nil).Once()
	mockRepo.On("GetUserReviewForApp", mock.Anything, appID, userID).Return(nil, nil).Once()
	mockRepo.On("CreateReview", mock.Anything, mock.AnythingOfType("*models.AppReview")).Return(nil).Once()

	review, err := service.CreateReview(context.Background(), appID, userID, req)

	assert.NoError(t, err)
	assert.NotNil(t, review)
	assert.Equal(t, req.Rating, review.Rating)
	mockRepo.AssertExpectations(t)
}

func TestAppDirectoryService_CreateReview_AlreadyReviewed(t *testing.T) {
	mockRepo := new(MockAppDirectoryRepository)
	service := NewAppDirectoryService(mockRepo)

	appID := uuid.New()
	userID := uuid.New()
	req := &models.CreateReviewRequest{
		Rating:     5,
		ReviewText: "Great app!",
	}

	app := &models.App{
		ID:     appID,
		Status: models.AppStatusApproved,
	}

	existingReview := &models.AppReview{
		ID:     uuid.New(),
		AppID:  appID,
		UserID: userID,
		Rating: 4,
	}

	mockRepo.On("GetAppByID", mock.Anything, appID).Return(app, nil).Once()
	mockRepo.On("GetUserReviewForApp", mock.Anything, appID, userID).Return(existingReview, nil).Once()

	review, err := service.CreateReview(context.Background(), appID, userID, req)

	assert.Error(t, err)
	assert.Nil(t, review)
	assert.Equal(t, ErrAlreadyReviewed, err)
	mockRepo.AssertExpectations(t)
}

func TestAppDirectoryService_ApproveApp_Success(t *testing.T) {
	mockRepo := new(MockAppDirectoryRepository)
	service := NewAppDirectoryService(mockRepo)

	appID := uuid.New()
	app := &models.App{
		ID:     appID,
		Status: models.AppStatusPending,
	}

	mockRepo.On("GetAppByID", mock.Anything, appID).Return(app, nil).Once()
	mockRepo.On("UpdateAppStatus", mock.Anything, appID, models.AppStatusApproved).Return(nil).Once()

	err := service.ApproveApp(context.Background(), appID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAppDirectoryService_ApproveApp_InvalidStatus(t *testing.T) {
	mockRepo := new(MockAppDirectoryRepository)
	service := NewAppDirectoryService(mockRepo)

	appID := uuid.New()
	app := &models.App{
		ID:     appID,
		Status: models.AppStatusApproved,
	}

	mockRepo.On("GetAppByID", mock.Anything, appID).Return(app, nil).Once()

	err := service.ApproveApp(context.Background(), appID)

	assert.Error(t, err)
	assert.Equal(t, ErrInvalidStatus, err)
	mockRepo.AssertExpectations(t)
}

func TestAppCategory_ParseAppCategory(t *testing.T) {
	tests := []struct {
		input    string
		expected models.AppCategory
		ok       bool
	}{
		{"moderation", models.AppCategoryModeration, true},
		{"music", models.AppCategoryMusic, true},
		{"gaming", models.AppCategoryGaming, true},
		{"utility", models.AppCategoryUtility, true},
		{"fun", models.AppCategoryFun, true},
		{"education", models.AppCategoryEducation, true},
		{"roleplay", models.AppCategoryRoleplay, true},
		{"economy", models.AppCategoryEconomy, true},
		{"invalid", 0, false},
		{"", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			cat, ok := models.ParseAppCategory(tt.input)
			assert.Equal(t, tt.expected, cat)
			assert.Equal(t, tt.ok, ok)
		})
	}
}

func TestAppCategory_String(t *testing.T) {
	tests := []struct {
		category models.AppCategory
		expected string
	}{
		{models.AppCategoryModeration, "moderation"},
		{models.AppCategoryMusic, "music"},
		{models.AppCategoryGaming, "gaming"},
		{models.AppCategoryUtility, "utility"},
		{models.AppCategoryFun, "fun"},
		{models.AppCategoryEducation, "education"},
		{models.AppCategoryRoleplay, "roleplay"},
		{models.AppCategoryEconomy, "economy"},
		{models.AppCategory(100), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.category.String())
		})
	}
}

func TestAppStatus_String(t *testing.T) {
	tests := []struct {
		status   models.AppStatus
		expected string
	}{
		{models.AppStatusPending, "pending"},
		{models.AppStatusApproved, "approved"},
		{models.AppStatusRejected, "rejected"},
		{models.AppStatusSuspended, "suspended"},
		{models.AppStatus(100), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.String())
		})
	}
}
