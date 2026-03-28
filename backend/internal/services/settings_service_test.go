package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"hearth/internal/models"
)

// MockSettingsRepository mocks the SettingsRepository for testing
type MockSettingsRepository struct {
	mock.Mock
}

func (m *MockSettingsRepository) Get(ctx context.Context, userID uuid.UUID) (*models.UserSettings, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserSettings), args.Error(1)
}

func (m *MockSettingsRepository) Create(ctx context.Context, settings *models.UserSettings) error {
	args := m.Called(ctx, settings)
	return args.Error(0)
}

func (m *MockSettingsRepository) Update(ctx context.Context, settings *models.UserSettings) error {
	args := m.Called(ctx, settings)
	return args.Error(0)
}

func (m *MockSettingsRepository) Upsert(ctx context.Context, settings *models.UserSettings) error {
	args := m.Called(ctx, settings)
	return args.Error(0)
}

func (m *MockSettingsRepository) Delete(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func newTestSettingsService() (*SettingsService, *MockSettingsRepository, *MockEventBus) {
	repo := new(MockSettingsRepository)
	eventBus := new(MockEventBus)
	eventBus.On("Publish", mock.Anything, mock.Anything).Return()
	service := NewSettingsService(repo, eventBus)
	return service, repo, eventBus
}

func TestSettingsService_GetSettings_Exists(t *testing.T) {
	service, repo, _ := newTestSettingsService()
	ctx := context.Background()
	userID := uuid.New()

	existingSettings := &models.UserSettings{
		UserID:               userID,
		Theme:                "light",
		MessageDisplay:       "compact",
		NotificationsEnabled: true,
		PrivacyShowActivity:  true,
		Locale:               "en-US",
		UpdatedAt:            time.Now(),
	}

	repo.On("Get", ctx, userID).Return(existingSettings, nil)

	settings, err := service.GetSettings(ctx, userID)

	assert.NoError(t, err)
	assert.NotNil(t, settings)
	assert.Equal(t, "light", settings.Theme)
	assert.Equal(t, "compact", settings.MessageDisplay)

	repo.AssertExpectations(t)
}

func TestSettingsService_GetSettings_NotExists_CreatesDefaults(t *testing.T) {
	service, repo, _ := newTestSettingsService()
	ctx := context.Background()
	userID := uuid.New()

	repo.On("Get", ctx, userID).Return(nil, nil).Once()
	repo.On("Create", ctx, mock.MatchedBy(func(s *models.UserSettings) bool {
		return s.UserID == userID && s.Theme == "dark" && s.MessageDisplay == "cozy"
	})).Return(nil)

	settings, err := service.GetSettings(ctx, userID)

	assert.NoError(t, err)
	assert.NotNil(t, settings)
	assert.Equal(t, userID, settings.UserID)
	assert.Equal(t, "dark", settings.Theme)
	assert.Equal(t, "cozy", settings.MessageDisplay)
	assert.True(t, settings.NotificationsEnabled)
	assert.True(t, settings.PrivacyShowActivity)

	repo.AssertExpectations(t)
}

func TestSettingsService_GetSettings_RepoError(t *testing.T) {
	service, repo, _ := newTestSettingsService()
	ctx := context.Background()
	userID := uuid.New()

	repo.On("Get", ctx, userID).Return(nil, assert.AnError)

	settings, err := service.GetSettings(ctx, userID)

	assert.Error(t, err)
	assert.Nil(t, settings)

	repo.AssertExpectations(t)
}

func TestSettingsService_UpdateSettings_Theme(t *testing.T) {
	service, repo, eventBus := newTestSettingsService()
	ctx := context.Background()
	userID := uuid.New()

	existingSettings := &models.UserSettings{
		UserID:               userID,
		Theme:                "dark",
		MessageDisplay:       "cozy",
		NotificationsEnabled: true,
		PrivacyShowActivity:  true,
		Locale:               "en-US",
		UpdatedAt:            time.Now(),
	}

	repo.On("Get", ctx, userID).Return(existingSettings, nil)
	repo.On("Upsert", ctx, mock.MatchedBy(func(s *models.UserSettings) bool {
		return s.Theme == "light"
	})).Return(nil)

	theme := "light"
	updates := &models.UpdateUserSettingsRequest{
		Theme: &theme,
	}

	settings, err := service.UpdateSettings(ctx, userID, updates)

	assert.NoError(t, err)
	assert.NotNil(t, settings)
	assert.Equal(t, "light", settings.Theme)

	repo.AssertExpectations(t)
	eventBus.AssertCalled(t, "Publish", "user.settings_updated", mock.Anything)
}

func TestSettingsService_UpdateSettings_Notifications(t *testing.T) {
	service, repo, eventBus := newTestSettingsService()
	ctx := context.Background()
	userID := uuid.New()

	existingSettings := models.DefaultUserSettings(userID)

	repo.On("Get", ctx, userID).Return(existingSettings, nil)
	repo.On("Upsert", ctx, mock.MatchedBy(func(s *models.UserSettings) bool {
		return s.NotificationsEnabled == false && s.NotificationsSound == false && s.NotificationsMentionsOnly == true
	})).Return(nil)

	notificationsEnabled := false
	notificationsSound := false
	notificationsMentionsOnly := true
	updates := &models.UpdateUserSettingsRequest{
		NotificationsEnabled:      &notificationsEnabled,
		NotificationsSound:        &notificationsSound,
		NotificationsMentionsOnly: &notificationsMentionsOnly,
	}

	settings, err := service.UpdateSettings(ctx, userID, updates)

	assert.NoError(t, err)
	assert.NotNil(t, settings)
	assert.False(t, settings.NotificationsEnabled)
	assert.False(t, settings.NotificationsSound)
	assert.True(t, settings.NotificationsMentionsOnly)

	repo.AssertExpectations(t)
	eventBus.AssertCalled(t, "Publish", "user.settings_updated", mock.Anything)
}

func TestSettingsService_UpdateSettings_Privacy(t *testing.T) {
	service, repo, eventBus := newTestSettingsService()
	ctx := context.Background()
	userID := uuid.New()

	existingSettings := models.DefaultUserSettings(userID)

	repo.On("Get", ctx, userID).Return(existingSettings, nil)
	repo.On("Upsert", ctx, mock.MatchedBy(func(s *models.UserSettings) bool {
		return s.PrivacyDMFromServers == false && s.PrivacyDMFromFriendsOnly == true && s.PrivacyShowActivity == false
	})).Return(nil)

	privacyDMFromServers := false
	privacyDMFromFriendsOnly := true
	privacyShowActivity := false
	updates := &models.UpdateUserSettingsRequest{
		PrivacyDMFromServers:     &privacyDMFromServers,
		PrivacyDMFromFriendsOnly: &privacyDMFromFriendsOnly,
		PrivacyShowActivity:      &privacyShowActivity,
	}

	settings, err := service.UpdateSettings(ctx, userID, updates)

	assert.NoError(t, err)
	assert.NotNil(t, settings)
	assert.False(t, settings.PrivacyDMFromServers)
	assert.True(t, settings.PrivacyDMFromFriendsOnly)
	assert.False(t, settings.PrivacyShowActivity)

	repo.AssertExpectations(t)
	eventBus.AssertCalled(t, "Publish", "user.settings_updated", mock.Anything)
}

func TestSettingsService_UpdateSettings_MultipleFields(t *testing.T) {
	service, repo, eventBus := newTestSettingsService()
	ctx := context.Background()
	userID := uuid.New()

	existingSettings := models.DefaultUserSettings(userID)

	repo.On("Get", ctx, userID).Return(existingSettings, nil)
	repo.On("Upsert", ctx, mock.MatchedBy(func(s *models.UserSettings) bool {
		return s.Theme == "system" && s.CompactMode == true && s.DeveloperMode == true && s.Locale == "de-DE"
	})).Return(nil)

	theme := "system"
	compactMode := true
	developerMode := true
	locale := "de-DE"
	updates := &models.UpdateUserSettingsRequest{
		Theme:         &theme,
		CompactMode:   &compactMode,
		DeveloperMode: &developerMode,
		Locale:        &locale,
	}

	settings, err := service.UpdateSettings(ctx, userID, updates)

	assert.NoError(t, err)
	assert.NotNil(t, settings)
	assert.Equal(t, "system", settings.Theme)
	assert.True(t, settings.CompactMode)
	assert.True(t, settings.DeveloperMode)
	assert.Equal(t, "de-DE", settings.Locale)

	repo.AssertExpectations(t)
	eventBus.AssertCalled(t, "Publish", "user.settings_updated", mock.Anything)
}

func TestSettingsService_UpdateSettings_UpsertError(t *testing.T) {
	service, repo, _ := newTestSettingsService()
	ctx := context.Background()
	userID := uuid.New()

	existingSettings := models.DefaultUserSettings(userID)

	repo.On("Get", ctx, userID).Return(existingSettings, nil)
	repo.On("Upsert", ctx, mock.Anything).Return(assert.AnError)

	theme := "light"
	updates := &models.UpdateUserSettingsRequest{
		Theme: &theme,
	}

	settings, err := service.UpdateSettings(ctx, userID, updates)

	assert.Error(t, err)
	assert.Nil(t, settings)

	repo.AssertExpectations(t)
}

func TestSettingsService_ResetSettings(t *testing.T) {
	service, repo, eventBus := newTestSettingsService()
	ctx := context.Background()
	userID := uuid.New()

	repo.On("Upsert", ctx, mock.MatchedBy(func(s *models.UserSettings) bool {
		return s.UserID == userID && s.Theme == "dark" && s.MessageDisplay == "cozy" &&
			s.NotificationsEnabled == true && s.PrivacyShowActivity == true
	})).Return(nil)

	settings, err := service.ResetSettings(ctx, userID)

	assert.NoError(t, err)
	assert.NotNil(t, settings)
	assert.Equal(t, userID, settings.UserID)
	assert.Equal(t, "dark", settings.Theme)
	assert.Equal(t, "cozy", settings.MessageDisplay)
	assert.True(t, settings.NotificationsEnabled)
	assert.True(t, settings.PrivacyShowActivity)
	assert.Equal(t, "en-US", settings.Locale)

	repo.AssertExpectations(t)
	eventBus.AssertCalled(t, "Publish", "user.settings_reset", mock.Anything)
}

func TestSettingsService_ResetSettings_UpsertError(t *testing.T) {
	service, repo, _ := newTestSettingsService()
	ctx := context.Background()
	userID := uuid.New()

	repo.On("Upsert", ctx, mock.Anything).Return(assert.AnError)

	settings, err := service.ResetSettings(ctx, userID)

	assert.Error(t, err)
	assert.Nil(t, settings)

	repo.AssertExpectations(t)
}

func TestSettingsService_UpdateSettings_DisplaySettings(t *testing.T) {
	service, repo, eventBus := newTestSettingsService()
	ctx := context.Background()
	userID := uuid.New()

	existingSettings := models.DefaultUserSettings(userID)

	repo.On("Get", ctx, userID).Return(existingSettings, nil)
	repo.On("Upsert", ctx, mock.MatchedBy(func(s *models.UserSettings) bool {
		return s.InlineEmbeds == false && s.InlineAttachments == false &&
			s.RenderReactions == false && s.AnimateEmoji == false && s.EnableTTS == false
	})).Return(nil)

	inlineEmbeds := false
	inlineAttachments := false
	renderReactions := false
	animateEmoji := false
	enableTTS := false
	updates := &models.UpdateUserSettingsRequest{
		InlineEmbeds:      &inlineEmbeds,
		InlineAttachments: &inlineAttachments,
		RenderReactions:   &renderReactions,
		AnimateEmoji:      &animateEmoji,
		EnableTTS:         &enableTTS,
	}

	settings, err := service.UpdateSettings(ctx, userID, updates)

	assert.NoError(t, err)
	assert.NotNil(t, settings)
	assert.False(t, settings.InlineEmbeds)
	assert.False(t, settings.InlineAttachments)
	assert.False(t, settings.RenderReactions)
	assert.False(t, settings.AnimateEmoji)
	assert.False(t, settings.EnableTTS)

	repo.AssertExpectations(t)
	eventBus.AssertCalled(t, "Publish", "user.settings_updated", mock.Anything)
}

func TestSettingsService_UpdateSettings_CustomCSS(t *testing.T) {
	service, repo, eventBus := newTestSettingsService()
	ctx := context.Background()
	userID := uuid.New()

	existingSettings := models.DefaultUserSettings(userID)

	repo.On("Get", ctx, userID).Return(existingSettings, nil)
	repo.On("Upsert", ctx, mock.MatchedBy(func(s *models.UserSettings) bool {
		return s.CustomCSS != nil && *s.CustomCSS == ".custom { color: red; }"
	})).Return(nil)

	customCSS := ".custom { color: red; }"
	updates := &models.UpdateUserSettingsRequest{
		CustomCSS: &customCSS,
	}

	settings, err := service.UpdateSettings(ctx, userID, updates)

	assert.NoError(t, err)
	assert.NotNil(t, settings)
	assert.NotNil(t, settings.CustomCSS)
	assert.Equal(t, customCSS, *settings.CustomCSS)

	repo.AssertExpectations(t)
	eventBus.AssertCalled(t, "Publish", "user.settings_updated", mock.Anything)
}

func TestSettingsService_UpdateSettings_AllNotificationSettings(t *testing.T) {
	service, repo, eventBus := newTestSettingsService()
	ctx := context.Background()
	userID := uuid.New()

	existingSettings := models.DefaultUserSettings(userID)

	repo.On("Get", ctx, userID).Return(existingSettings, nil)
	repo.On("Upsert", ctx, mock.MatchedBy(func(s *models.UserSettings) bool {
		return s.NotificationsEnabled == true && s.NotificationsSound == false &&
			s.NotificationsDesktop == false && s.NotificationsMentionsOnly == true &&
			s.NotificationsDM == false && s.NotificationsServerDefaults == false
	})).Return(nil)

	notificationsEnabled := true
	notificationsSound := false
	notificationsDesktop := false
	notificationsMentionsOnly := true
	notificationsDM := false
	notificationsServerDefaults := false
	updates := &models.UpdateUserSettingsRequest{
		NotificationsEnabled:        &notificationsEnabled,
		NotificationsSound:          &notificationsSound,
		NotificationsDesktop:        &notificationsDesktop,
		NotificationsMentionsOnly:   &notificationsMentionsOnly,
		NotificationsDM:             &notificationsDM,
		NotificationsServerDefaults: &notificationsServerDefaults,
	}

	settings, err := service.UpdateSettings(ctx, userID, updates)

	assert.NoError(t, err)
	assert.NotNil(t, settings)
	assert.True(t, settings.NotificationsEnabled)
	assert.False(t, settings.NotificationsSound)
	assert.False(t, settings.NotificationsDesktop)
	assert.True(t, settings.NotificationsMentionsOnly)
	assert.False(t, settings.NotificationsDM)
	assert.False(t, settings.NotificationsServerDefaults)

	repo.AssertExpectations(t)
	eventBus.AssertCalled(t, "Publish", "user.settings_updated", mock.Anything)
}

func TestSettingsService_UpdateSettings_AllPrivacySettings(t *testing.T) {
	service, repo, eventBus := newTestSettingsService()
	ctx := context.Background()
	userID := uuid.New()

	existingSettings := models.DefaultUserSettings(userID)

	repo.On("Get", ctx, userID).Return(existingSettings, nil)
	repo.On("Upsert", ctx, mock.MatchedBy(func(s *models.UserSettings) bool {
		return s.PrivacyDMFromServers == false && s.PrivacyDMFromFriendsOnly == true &&
			s.PrivacyShowActivity == false && s.PrivacyFriendRequestsAll == false &&
			s.PrivacyReadReceipts == false
	})).Return(nil)

	privacyDMFromServers := false
	privacyDMFromFriendsOnly := true
	privacyShowActivity := false
	privacyFriendRequestsAll := false
	privacyReadReceipts := false
	updates := &models.UpdateUserSettingsRequest{
		PrivacyDMFromServers:     &privacyDMFromServers,
		PrivacyDMFromFriendsOnly: &privacyDMFromFriendsOnly,
		PrivacyShowActivity:      &privacyShowActivity,
		PrivacyFriendRequestsAll: &privacyFriendRequestsAll,
		PrivacyReadReceipts:      &privacyReadReceipts,
	}

	settings, err := service.UpdateSettings(ctx, userID, updates)

	assert.NoError(t, err)
	assert.NotNil(t, settings)
	assert.False(t, settings.PrivacyDMFromServers)
	assert.True(t, settings.PrivacyDMFromFriendsOnly)
	assert.False(t, settings.PrivacyShowActivity)
	assert.False(t, settings.PrivacyFriendRequestsAll)
	assert.False(t, settings.PrivacyReadReceipts)

	repo.AssertExpectations(t)
	eventBus.AssertCalled(t, "Publish", "user.settings_updated", mock.Anything)
}

func TestSettingsService_GetSettings_CreateFails_RetryGet(t *testing.T) {
	service, repo, _ := newTestSettingsService()
	ctx := context.Background()
	userID := uuid.New()

	existingSettings := &models.UserSettings{
		UserID:               userID,
		Theme:                "dark",
		NotificationsEnabled: true,
		UpdatedAt:            time.Now(),
	}

	// First Get returns nil (no settings)
	repo.On("Get", ctx, userID).Return(nil, nil).Once()
	// Create fails (race condition)
	repo.On("Create", ctx, mock.Anything).Return(assert.AnError)
	// Retry Get returns the settings
	repo.On("Get", ctx, userID).Return(existingSettings, nil)

	settings, err := service.GetSettings(ctx, userID)

	assert.NoError(t, err)
	assert.NotNil(t, settings)
	assert.Equal(t, "dark", settings.Theme)

	repo.AssertExpectations(t)
}
