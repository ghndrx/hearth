package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"hearth/internal/models"
)

// MockSettingsService mocks the SettingsService for testing
type MockSettingsService struct {
	mock.Mock
}

func (m *MockSettingsService) GetSettings(ctx context.Context, userID uuid.UUID) (*models.UserSettings, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserSettings), args.Error(1)
}

func (m *MockSettingsService) UpdateSettings(ctx context.Context, userID uuid.UUID, updates *models.UpdateUserSettingsRequest) (*models.UserSettings, error) {
	args := m.Called(ctx, userID, updates)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserSettings), args.Error(1)
}

func (m *MockSettingsService) ResetSettings(ctx context.Context, userID uuid.UUID) (*models.UserSettings, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserSettings), args.Error(1)
}

// testSettingsHandler creates a test settings handler with mocks
type testSettingsHandler struct {
	handler         *SettingsHandler
	settingsService *MockSettingsService
	app             *fiber.App
	userID          uuid.UUID
}

func newTestSettingsHandler() *testSettingsHandler {
	settingsService := new(MockSettingsService)
	handler := NewSettingsHandler(settingsService)

	app := fiber.New()
	userID := uuid.New()

	// Add middleware to set userID in locals
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", userID)
		return c.Next()
	})

	// Setup routes
	app.Get("/users/@me/settings", handler.GetSettings)
	app.Patch("/users/@me/settings", handler.UpdateSettings)
	app.Delete("/users/@me/settings", handler.ResetSettings)

	return &testSettingsHandler{
		handler:         handler,
		settingsService: settingsService,
		app:             app,
		userID:          userID,
	}
}

func TestSettingsHandler_GetSettings(t *testing.T) {
	th := newTestSettingsHandler()

	settings := &models.UserSettings{
		UserID:                    th.userID,
		Theme:                     "dark",
		MessageDisplay:            "cozy",
		CompactMode:               false,
		DeveloperMode:             false,
		InlineEmbeds:              true,
		InlineAttachments:         true,
		RenderReactions:           true,
		AnimateEmoji:              true,
		EnableTTS:                 true,
		NotificationsEnabled:      true,
		NotificationsSound:        true,
		NotificationsDesktop:      true,
		NotificationsMentionsOnly: false,
		NotificationsDM:           true,
		NotificationsServerDefaults: true,
		PrivacyDMFromServers:      true,
		PrivacyDMFromFriendsOnly:  false,
		PrivacyShowActivity:       true,
		PrivacyFriendRequestsAll:  true,
		PrivacyReadReceipts:       true,
		Locale:                    "en-US",
		UpdatedAt:                 time.Now(),
	}

	th.settingsService.On("GetSettings", mock.Anything, th.userID).Return(settings, nil)

	req := httptest.NewRequest(http.MethodGet, "/users/@me/settings", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result models.UserSettings
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, settings.UserID, result.UserID)
	assert.Equal(t, settings.Theme, result.Theme)
	assert.Equal(t, settings.NotificationsEnabled, result.NotificationsEnabled)
	assert.Equal(t, settings.PrivacyShowActivity, result.PrivacyShowActivity)

	th.settingsService.AssertExpectations(t)
}

func TestSettingsHandler_GetSettings_Error(t *testing.T) {
	th := newTestSettingsHandler()

	th.settingsService.On("GetSettings", mock.Anything, th.userID).Return(nil, assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/users/@me/settings", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "failed to get settings", result["error"])

	th.settingsService.AssertExpectations(t)
}

func TestSettingsHandler_UpdateSettings_Theme(t *testing.T) {
	th := newTestSettingsHandler()

	theme := "light"
	updatedSettings := &models.UserSettings{
		UserID:               th.userID,
		Theme:                theme,
		MessageDisplay:       "cozy",
		NotificationsEnabled: true,
		PrivacyShowActivity:  true,
		Locale:               "en-US",
		UpdatedAt:            time.Now(),
	}

	th.settingsService.On("UpdateSettings", mock.Anything, th.userID, mock.MatchedBy(func(u *models.UpdateUserSettingsRequest) bool {
		return u.Theme != nil && *u.Theme == theme
	})).Return(updatedSettings, nil)

	body := map[string]interface{}{
		"theme": theme,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/users/@me/settings", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result models.UserSettings
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, theme, result.Theme)

	th.settingsService.AssertExpectations(t)
}

func TestSettingsHandler_UpdateSettings_InvalidTheme(t *testing.T) {
	th := newTestSettingsHandler()

	body := map[string]interface{}{
		"theme": "invalid_theme",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/users/@me/settings", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Contains(t, result["error"], "invalid theme")
}

func TestSettingsHandler_UpdateSettings_Notifications(t *testing.T) {
	th := newTestSettingsHandler()

	updatedSettings := &models.UserSettings{
		UserID:                    th.userID,
		Theme:                     "dark",
		NotificationsEnabled:      false,
		NotificationsSound:        false,
		NotificationsMentionsOnly: true,
		UpdatedAt:                 time.Now(),
	}

	th.settingsService.On("UpdateSettings", mock.Anything, th.userID, mock.MatchedBy(func(u *models.UpdateUserSettingsRequest) bool {
		return u.NotificationsEnabled != nil && *u.NotificationsEnabled == false &&
			u.NotificationsSound != nil && *u.NotificationsSound == false &&
			u.NotificationsMentionsOnly != nil && *u.NotificationsMentionsOnly == true
	})).Return(updatedSettings, nil)

	body := map[string]interface{}{
		"notifications_enabled":       false,
		"notifications_sound":         false,
		"notifications_mentions_only": true,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/users/@me/settings", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result models.UserSettings
	json.NewDecoder(resp.Body).Decode(&result)
	assert.False(t, result.NotificationsEnabled)
	assert.False(t, result.NotificationsSound)
	assert.True(t, result.NotificationsMentionsOnly)

	th.settingsService.AssertExpectations(t)
}

func TestSettingsHandler_UpdateSettings_Privacy(t *testing.T) {
	th := newTestSettingsHandler()

	updatedSettings := &models.UserSettings{
		UserID:                   th.userID,
		Theme:                    "dark",
		PrivacyDMFromServers:     false,
		PrivacyDMFromFriendsOnly: true,
		PrivacyShowActivity:      false,
		PrivacyReadReceipts:      false,
		UpdatedAt:                time.Now(),
	}

	th.settingsService.On("UpdateSettings", mock.Anything, th.userID, mock.MatchedBy(func(u *models.UpdateUserSettingsRequest) bool {
		return u.PrivacyDMFromServers != nil && *u.PrivacyDMFromServers == false &&
			u.PrivacyDMFromFriendsOnly != nil && *u.PrivacyDMFromFriendsOnly == true &&
			u.PrivacyShowActivity != nil && *u.PrivacyShowActivity == false &&
			u.PrivacyReadReceipts != nil && *u.PrivacyReadReceipts == false
	})).Return(updatedSettings, nil)

	body := map[string]interface{}{
		"privacy_dm_from_servers":      false,
		"privacy_dm_from_friends_only": true,
		"privacy_show_activity":        false,
		"privacy_read_receipts":        false,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/users/@me/settings", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result models.UserSettings
	json.NewDecoder(resp.Body).Decode(&result)
	assert.False(t, result.PrivacyDMFromServers)
	assert.True(t, result.PrivacyDMFromFriendsOnly)
	assert.False(t, result.PrivacyShowActivity)
	assert.False(t, result.PrivacyReadReceipts)

	th.settingsService.AssertExpectations(t)
}

func TestSettingsHandler_UpdateSettings_InvalidMessageDisplay(t *testing.T) {
	th := newTestSettingsHandler()

	body := map[string]interface{}{
		"message_display": "invalid",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/users/@me/settings", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Contains(t, result["error"], "invalid message_display")
}

func TestSettingsHandler_UpdateSettings_InvalidLocale(t *testing.T) {
	th := newTestSettingsHandler()

	body := map[string]interface{}{
		"locale": "x", // Too short
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/users/@me/settings", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Contains(t, result["error"], "invalid locale")
}

func TestSettingsHandler_UpdateSettings_InvalidBody(t *testing.T) {
	th := newTestSettingsHandler()

	req := httptest.NewRequest(http.MethodPatch, "/users/@me/settings", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "invalid request body", result["error"])
}

func TestSettingsHandler_UpdateSettings_ServiceError(t *testing.T) {
	th := newTestSettingsHandler()

	th.settingsService.On("UpdateSettings", mock.Anything, th.userID, mock.Anything).
		Return(nil, assert.AnError)

	body := map[string]interface{}{
		"theme": "dark",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/users/@me/settings", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "failed to update settings", result["error"])

	th.settingsService.AssertExpectations(t)
}

func TestSettingsHandler_UpdateSettings_MultipleFields(t *testing.T) {
	th := newTestSettingsHandler()

	updatedSettings := &models.UserSettings{
		UserID:               th.userID,
		Theme:                "system",
		MessageDisplay:       "compact",
		CompactMode:          true,
		DeveloperMode:        true,
		NotificationsEnabled: true,
		PrivacyShowActivity:  false,
		Locale:               "de-DE",
		UpdatedAt:            time.Now(),
	}

	th.settingsService.On("UpdateSettings", mock.Anything, th.userID, mock.MatchedBy(func(u *models.UpdateUserSettingsRequest) bool {
		return u.Theme != nil && *u.Theme == "system" &&
			u.MessageDisplay != nil && *u.MessageDisplay == "compact" &&
			u.CompactMode != nil && *u.CompactMode == true &&
			u.DeveloperMode != nil && *u.DeveloperMode == true &&
			u.PrivacyShowActivity != nil && *u.PrivacyShowActivity == false &&
			u.Locale != nil && *u.Locale == "de-DE"
	})).Return(updatedSettings, nil)

	body := map[string]interface{}{
		"theme":                  "system",
		"message_display":        "compact",
		"compact_mode":           true,
		"developer_mode":         true,
		"privacy_show_activity":  false,
		"locale":                 "de-DE",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/users/@me/settings", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result models.UserSettings
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "system", result.Theme)
	assert.Equal(t, "compact", result.MessageDisplay)
	assert.True(t, result.CompactMode)
	assert.True(t, result.DeveloperMode)
	assert.False(t, result.PrivacyShowActivity)
	assert.Equal(t, "de-DE", result.Locale)

	th.settingsService.AssertExpectations(t)
}

func TestSettingsHandler_ResetSettings(t *testing.T) {
	th := newTestSettingsHandler()

	defaultSettings := models.DefaultUserSettings(th.userID)

	th.settingsService.On("ResetSettings", mock.Anything, th.userID).Return(defaultSettings, nil)

	req := httptest.NewRequest(http.MethodDelete, "/users/@me/settings", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result models.UserSettings
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, th.userID, result.UserID)
	assert.Equal(t, "dark", result.Theme)
	assert.Equal(t, "cozy", result.MessageDisplay)
	assert.True(t, result.NotificationsEnabled)
	assert.True(t, result.PrivacyShowActivity)
	assert.Equal(t, "en-US", result.Locale)

	th.settingsService.AssertExpectations(t)
}

func TestSettingsHandler_ResetSettings_Error(t *testing.T) {
	th := newTestSettingsHandler()

	th.settingsService.On("ResetSettings", mock.Anything, th.userID).Return(nil, assert.AnError)

	req := httptest.NewRequest(http.MethodDelete, "/users/@me/settings", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "failed to reset settings", result["error"])

	th.settingsService.AssertExpectations(t)
}

func TestSettingsHandler_UpdateSettings_DisplaySettings(t *testing.T) {
	th := newTestSettingsHandler()

	updatedSettings := &models.UserSettings{
		UserID:            th.userID,
		Theme:             "dark",
		InlineEmbeds:      false,
		InlineAttachments: false,
		RenderReactions:   false,
		AnimateEmoji:      false,
		EnableTTS:         false,
		UpdatedAt:         time.Now(),
	}

	th.settingsService.On("UpdateSettings", mock.Anything, th.userID, mock.MatchedBy(func(u *models.UpdateUserSettingsRequest) bool {
		return u.InlineEmbeds != nil && *u.InlineEmbeds == false &&
			u.InlineAttachments != nil && *u.InlineAttachments == false &&
			u.RenderReactions != nil && *u.RenderReactions == false &&
			u.AnimateEmoji != nil && *u.AnimateEmoji == false &&
			u.EnableTTS != nil && *u.EnableTTS == false
	})).Return(updatedSettings, nil)

	body := map[string]interface{}{
		"inline_embeds":      false,
		"inline_attachments": false,
		"render_reactions":   false,
		"animate_emoji":      false,
		"enable_tts":         false,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/users/@me/settings", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result models.UserSettings
	json.NewDecoder(resp.Body).Decode(&result)
	assert.False(t, result.InlineEmbeds)
	assert.False(t, result.InlineAttachments)
	assert.False(t, result.RenderReactions)
	assert.False(t, result.AnimateEmoji)
	assert.False(t, result.EnableTTS)

	th.settingsService.AssertExpectations(t)
}

func TestSettingsHandler_UpdateSettings_AllNotificationSettings(t *testing.T) {
	th := newTestSettingsHandler()

	updatedSettings := &models.UserSettings{
		UserID:                      th.userID,
		Theme:                       "dark",
		NotificationsEnabled:        true,
		NotificationsSound:          false,
		NotificationsDesktop:        false,
		NotificationsMentionsOnly:   true,
		NotificationsDM:             false,
		NotificationsServerDefaults: false,
		UpdatedAt:                   time.Now(),
	}

	th.settingsService.On("UpdateSettings", mock.Anything, th.userID, mock.MatchedBy(func(u *models.UpdateUserSettingsRequest) bool {
		return u.NotificationsEnabled != nil && *u.NotificationsEnabled == true &&
			u.NotificationsSound != nil && *u.NotificationsSound == false &&
			u.NotificationsDesktop != nil && *u.NotificationsDesktop == false &&
			u.NotificationsMentionsOnly != nil && *u.NotificationsMentionsOnly == true &&
			u.NotificationsDM != nil && *u.NotificationsDM == false &&
			u.NotificationsServerDefaults != nil && *u.NotificationsServerDefaults == false
	})).Return(updatedSettings, nil)

	body := map[string]interface{}{
		"notifications_enabled":         true,
		"notifications_sound":           false,
		"notifications_desktop":         false,
		"notifications_mentions_only":   true,
		"notifications_dm":              false,
		"notifications_server_defaults": false,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/users/@me/settings", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result models.UserSettings
	json.NewDecoder(resp.Body).Decode(&result)
	assert.True(t, result.NotificationsEnabled)
	assert.False(t, result.NotificationsSound)
	assert.False(t, result.NotificationsDesktop)
	assert.True(t, result.NotificationsMentionsOnly)
	assert.False(t, result.NotificationsDM)
	assert.False(t, result.NotificationsServerDefaults)

	th.settingsService.AssertExpectations(t)
}

func TestSettingsHandler_UpdateSettings_FriendRequestPrivacy(t *testing.T) {
	th := newTestSettingsHandler()

	updatedSettings := &models.UserSettings{
		UserID:                   th.userID,
		Theme:                    "dark",
		PrivacyFriendRequestsAll: false,
		UpdatedAt:                time.Now(),
	}

	th.settingsService.On("UpdateSettings", mock.Anything, th.userID, mock.MatchedBy(func(u *models.UpdateUserSettingsRequest) bool {
		return u.PrivacyFriendRequestsAll != nil && *u.PrivacyFriendRequestsAll == false
	})).Return(updatedSettings, nil)

	body := map[string]interface{}{
		"privacy_friend_requests_all": false,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/users/@me/settings", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result models.UserSettings
	json.NewDecoder(resp.Body).Decode(&result)
	assert.False(t, result.PrivacyFriendRequestsAll)

	th.settingsService.AssertExpectations(t)
}

func TestSettingsHandler_UpdateSettings_EmptyBody(t *testing.T) {
	th := newTestSettingsHandler()

	// Empty update should still be valid
	currentSettings := models.DefaultUserSettings(th.userID)

	th.settingsService.On("UpdateSettings", mock.Anything, th.userID, mock.Anything).
		Return(currentSettings, nil)

	body := map[string]interface{}{}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/users/@me/settings", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	th.settingsService.AssertExpectations(t)
}
