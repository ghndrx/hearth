package services

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"hearth/internal/models"
)

// MockSmartNotificationCache mocks the SmartNotificationCache interface
type MockSmartNotificationCache struct {
	mock.Mock
}

func (m *MockSmartNotificationCache) Get(ctx context.Context, key string) ([]byte, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockSmartNotificationCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

func (m *MockSmartNotificationCache) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

type testSmartNotifService struct {
	service  *SmartNotificationService
	repo     *MockNotificationRepository
	cache    *MockSmartNotificationCache
	eventBus *MockEventBus
}

func newTestSmartNotifService() *testSmartNotifService {
	repo := new(MockNotificationRepository)
	cache := new(MockSmartNotificationCache)
	eventBus := new(MockEventBus)
	service := NewSmartNotificationService(repo, cache, eventBus)

	return &testSmartNotifService{
		service:  service,
		repo:     repo,
		cache:    cache,
		eventBus: eventBus,
	}
}

// --- Priority Scoring Tests ---

func TestSmartNotificationService_ScoreNotification_DM(t *testing.T) {
	ts := newTestSmartNotifService()
	ctx := context.Background()

	input := &models.PriorityScoringInput{
		NotificationType: models.NotificationTypeDirectMessage,
		RecipientID:      uuid.New(),
		IsDM:             true,
	}

	// No sender engagement cache hit
	ts.cache.On("Get", ctx, mock.AnythingOfType("string")).Return(nil, fmt.Errorf("not found"))

	result, err := ts.service.ScoreNotification(ctx, input)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.NotificationPriorityUrgent, result.Priority)
	assert.Equal(t, models.DeliveryImmediate, result.DeliveryMode)
	assert.Equal(t, models.NotifCategoryDirectMessage, result.Category)
	assert.GreaterOrEqual(t, result.PriorityScore, 75)
}

func TestSmartNotificationService_ScoreNotification_Mention(t *testing.T) {
	ts := newTestSmartNotifService()
	ctx := context.Background()

	input := &models.PriorityScoringInput{
		NotificationType: models.NotificationTypeMention,
		RecipientID:      uuid.New(),
		HasMention:       true,
	}

	ts.cache.On("Get", ctx, mock.AnythingOfType("string")).Return(nil, fmt.Errorf("not found"))

	result, err := ts.service.ScoreNotification(ctx, input)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.NotificationPriorityUrgent, result.Priority)
	assert.Equal(t, models.DeliveryImmediate, result.DeliveryMode)
	assert.Equal(t, models.NotifCategoryMention, result.Category)
}

func TestSmartNotificationService_ScoreNotification_Reaction(t *testing.T) {
	ts := newTestSmartNotifService()
	ctx := context.Background()

	input := &models.PriorityScoringInput{
		NotificationType: models.NotificationTypeReaction,
		RecipientID:      uuid.New(),
	}

	ts.cache.On("Get", ctx, mock.AnythingOfType("string")).Return(nil, fmt.Errorf("not found"))

	result, err := ts.service.ScoreNotification(ctx, input)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.NotificationPriorityLow, result.Priority)
	assert.Equal(t, models.DeliveryBatched, result.DeliveryMode)
	assert.Equal(t, models.NotifCategoryReaction, result.Category)
}

func TestSmartNotificationService_ScoreNotification_WithSenderEngagement(t *testing.T) {
	ts := newTestSmartNotifService()
	ctx := context.Background()

	senderID := uuid.New()
	recipientID := uuid.New()

	input := &models.PriorityScoringInput{
		NotificationType: models.NotificationTypeReaction,
		SenderID:         &senderID,
		RecipientID:      recipientID,
	}

	// Return a high interaction count for the sender
	interactionData, _ := json.Marshal(50)
	engagementKey := fmt.Sprintf("smart_notif:engagement:%s:%s", recipientID.String(), senderID.String())
	ts.cache.On("Get", ctx, engagementKey).Return(interactionData, nil)

	result, err := ts.service.ScoreNotification(ctx, input)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	// Reaction (20) + engagement boost (10 max) = 30, so Normal priority
	assert.Equal(t, models.NotificationPriorityNormal, result.Priority)
}

func TestSmartNotificationService_ScoreNotification_Reply(t *testing.T) {
	ts := newTestSmartNotifService()
	ctx := context.Background()

	input := &models.PriorityScoringInput{
		NotificationType: models.NotificationTypeReply,
		RecipientID:      uuid.New(),
		IsReply:          true,
	}

	ts.cache.On("Get", ctx, mock.AnythingOfType("string")).Return(nil, fmt.Errorf("not found"))

	result, err := ts.service.ScoreNotification(ctx, input)

	assert.NoError(t, err)
	assert.Equal(t, models.NotificationPriorityHigh, result.Priority)
	assert.Equal(t, models.DeliveryImmediate, result.DeliveryMode)
}

func TestSmartNotificationService_ScoreNotification_AllTypes(t *testing.T) {
	types := []struct {
		notifType models.NotificationType
		category  models.NotificationCategory
	}{
		{models.NotificationTypeMention, models.NotifCategoryMention},
		{models.NotificationTypeReply, models.NotifCategoryReply},
		{models.NotificationTypeDirectMessage, models.NotifCategoryDirectMessage},
		{models.NotificationTypeReaction, models.NotifCategoryReaction},
		{models.NotificationTypeFriendRequest, models.NotifCategorySocial},
		{models.NotificationTypeFriendAccept, models.NotifCategorySocial},
		{models.NotificationTypeServerInvite, models.NotifCategoryServerEvent},
		{models.NotificationTypeServerJoin, models.NotifCategoryServerEvent},
		{models.NotificationTypeSystem, models.NotifCategorySystem},
		{models.NotificationTypeEventInvite, models.NotifCategoryServerEvent},
		{models.NotificationTypeEventStart, models.NotifCategoryServerEvent},
	}

	for _, tt := range types {
		t.Run(string(tt.notifType), func(t *testing.T) {
			ts := newTestSmartNotifService()
			ctx := context.Background()

			input := &models.PriorityScoringInput{
				NotificationType: tt.notifType,
				RecipientID:      uuid.New(),
			}

			ts.cache.On("Get", ctx, mock.AnythingOfType("string")).Return(nil, fmt.Errorf("not found"))

			result, err := ts.service.ScoreNotification(ctx, input)

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.category, result.Category)
			assert.GreaterOrEqual(t, result.PriorityScore, 0)
			assert.LessOrEqual(t, result.PriorityScore, 100)
		})
	}
}

// --- Snooze Tests ---

func TestSmartNotificationService_SnoozeNotifications(t *testing.T) {
	ts := newTestSmartNotifService()
	ctx := context.Background()
	userID := uuid.New()

	req := &models.SnoozeRequest{
		DurationMins: 30,
	}

	ts.cache.On("Set", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8"), 30*time.Minute).Return(nil)
	ts.eventBus.On("Publish", "notification.snoozed", mock.Anything).Return()

	config, err := ts.service.SnoozeNotifications(ctx, userID, req)

	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.True(t, config.Active)
	assert.Equal(t, userID, config.UserID)
	assert.True(t, config.Until.After(time.Now()))

	ts.cache.AssertExpectations(t)
	ts.eventBus.AssertExpectations(t)
}

func TestSmartNotificationService_SnoozeNotifications_WithServer(t *testing.T) {
	ts := newTestSmartNotifService()
	ctx := context.Background()
	userID := uuid.New()
	serverID := uuid.New()

	req := &models.SnoozeRequest{
		DurationMins: 60,
		ServerID:     &serverID,
	}

	ts.cache.On("Set", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8"), 60*time.Minute).Return(nil)
	ts.eventBus.On("Publish", "notification.snoozed", mock.Anything).Return()

	config, err := ts.service.SnoozeNotifications(ctx, userID, req)

	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, &serverID, config.ServerID)
}

func TestSmartNotificationService_UnsnoozeNotifications(t *testing.T) {
	ts := newTestSmartNotifService()
	ctx := context.Background()
	userID := uuid.New()

	ts.cache.On("Delete", ctx, mock.AnythingOfType("string")).Return(nil)
	ts.eventBus.On("Publish", "notification.unsnoozed", mock.Anything).Return()

	err := ts.service.UnsnoozeNotifications(ctx, userID, nil, nil)

	assert.NoError(t, err)

	ts.cache.AssertExpectations(t)
	ts.eventBus.AssertExpectations(t)
}

func TestSmartNotificationService_IsNotificationSnoozed_Active(t *testing.T) {
	ts := newTestSmartNotifService()
	ctx := context.Background()
	userID := uuid.New()

	config := &models.SnoozeConfig{
		UserID: userID,
		Active: true,
		Until:  time.Now().Add(30 * time.Minute),
	}

	data, _ := json.Marshal(config)
	ts.cache.On("Get", ctx, mock.AnythingOfType("string")).Return(data, nil).Once()

	snoozed, until, err := ts.service.IsNotificationSnoozed(ctx, userID, nil, nil)

	assert.NoError(t, err)
	assert.True(t, snoozed)
	assert.NotNil(t, until)
}

func TestSmartNotificationService_IsNotificationSnoozed_Expired(t *testing.T) {
	ts := newTestSmartNotifService()
	ctx := context.Background()
	userID := uuid.New()

	config := &models.SnoozeConfig{
		UserID: userID,
		Active: true,
		Until:  time.Now().Add(-10 * time.Minute), // expired
	}

	data, _ := json.Marshal(config)
	// The first check (channel+server) returns expired snooze
	ts.cache.On("Get", ctx, mock.AnythingOfType("string")).Return(data, nil)

	snoozed, until, err := ts.service.IsNotificationSnoozed(ctx, userID, nil, nil)

	assert.NoError(t, err)
	assert.False(t, snoozed)
	assert.Nil(t, until)
}

func TestSmartNotificationService_IsNotificationSnoozed_NotSnoozed(t *testing.T) {
	ts := newTestSmartNotifService()
	ctx := context.Background()
	userID := uuid.New()

	ts.cache.On("Get", ctx, mock.AnythingOfType("string")).Return(nil, fmt.Errorf("not found"))

	snoozed, until, err := ts.service.IsNotificationSnoozed(ctx, userID, nil, nil)

	assert.NoError(t, err)
	assert.False(t, snoozed)
	assert.Nil(t, until)
}

// --- Mute Tests ---

func TestSmartNotificationService_MuteNotifications(t *testing.T) {
	ts := newTestSmartNotifService()
	ctx := context.Background()
	userID := uuid.New()

	req := &models.MuteRequest{
		Muted: true,
	}

	ts.cache.On("Set", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8"), time.Duration(0)).Return(nil)
	ts.eventBus.On("Publish", "notification.mute_changed", mock.Anything).Return()

	config, err := ts.service.MuteNotifications(ctx, userID, req)

	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.True(t, config.Muted)
	assert.Equal(t, userID, config.UserID)

	ts.cache.AssertExpectations(t)
	ts.eventBus.AssertExpectations(t)
}

func TestSmartNotificationService_UnmuteNotifications(t *testing.T) {
	ts := newTestSmartNotifService()
	ctx := context.Background()
	userID := uuid.New()

	req := &models.MuteRequest{
		Muted: false,
	}

	ts.cache.On("Delete", ctx, mock.AnythingOfType("string")).Return(nil)
	ts.eventBus.On("Publish", "notification.mute_changed", mock.Anything).Return()

	config, err := ts.service.MuteNotifications(ctx, userID, req)

	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.False(t, config.Muted)

	ts.cache.AssertExpectations(t)
	ts.eventBus.AssertExpectations(t)
}

func TestSmartNotificationService_IsNotificationMuted(t *testing.T) {
	ts := newTestSmartNotifService()
	ctx := context.Background()
	userID := uuid.New()

	config := &models.MuteConfig{
		UserID: userID,
		Muted:  true,
	}

	data, _ := json.Marshal(config)
	ts.cache.On("Get", ctx, mock.AnythingOfType("string")).Return(data, nil).Once()

	muted, err := ts.service.IsNotificationMuted(ctx, userID, nil, nil)

	assert.NoError(t, err)
	assert.True(t, muted)
}

func TestSmartNotificationService_IsNotificationMuted_NotMuted(t *testing.T) {
	ts := newTestSmartNotifService()
	ctx := context.Background()
	userID := uuid.New()

	ts.cache.On("Get", ctx, mock.AnythingOfType("string")).Return(nil, fmt.Errorf("not found"))

	muted, err := ts.service.IsNotificationMuted(ctx, userID, nil, nil)

	assert.NoError(t, err)
	assert.False(t, muted)
}

// --- Engagement Tests ---

func TestSmartNotificationService_TrackNotificationClick(t *testing.T) {
	ts := newTestSmartNotifService()
	ctx := context.Background()
	userID := uuid.New()
	notifID := uuid.New()

	// incrementCounter calls Get then Set
	ts.cache.On("Get", ctx, mock.AnythingOfType("string")).Return(nil, fmt.Errorf("not found"))
	ts.cache.On("Set", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8"), mock.AnythingOfType("time.Duration")).Return(nil)
	ts.repo.On("MarkAsRead", ctx, notifID, userID).Return(nil)
	ts.eventBus.On("Publish", "notification.clicked", mock.Anything).Return()

	err := ts.service.TrackNotificationClick(ctx, userID, notifID)

	assert.NoError(t, err)
	ts.eventBus.AssertExpectations(t)
}

func TestSmartNotificationService_GetUserEngagement(t *testing.T) {
	ts := newTestSmartNotifService()
	ctx := context.Background()
	userID := uuid.New()

	clicksData, _ := json.Marshal(10)
	receivedData, _ := json.Marshal(50)
	dismissedData, _ := json.Marshal(5)

	clickKey := fmt.Sprintf("smart_notif:clicks:%s", userID.String())
	receivedKey := fmt.Sprintf("smart_notif:received:%s", userID.String())
	dismissedKey := fmt.Sprintf("smart_notif:dismissed:%s", userID.String())
	topChannelsKey := fmt.Sprintf("smart_notif:top_channels:%s", userID.String())

	ts.cache.On("Get", ctx, clickKey).Return(clicksData, nil)
	ts.cache.On("Get", ctx, receivedKey).Return(receivedData, nil)
	ts.cache.On("Get", ctx, dismissedKey).Return(dismissedData, nil)
	ts.cache.On("Get", ctx, topChannelsKey).Return(nil, fmt.Errorf("not found"))

	engagement, err := ts.service.GetUserEngagement(ctx, userID)

	assert.NoError(t, err)
	assert.NotNil(t, engagement)
	assert.Equal(t, 10, engagement.TotalClicked)
	assert.Equal(t, 50, engagement.TotalReceived)
	assert.Equal(t, 5, engagement.TotalDismissed)
	assert.InDelta(t, 0.2, engagement.ClickRate, 0.01)
}

func TestSmartNotificationService_GetUserEngagement_NoData(t *testing.T) {
	ts := newTestSmartNotifService()
	ctx := context.Background()
	userID := uuid.New()

	ts.cache.On("Get", ctx, mock.AnythingOfType("string")).Return(nil, fmt.Errorf("not found"))

	engagement, err := ts.service.GetUserEngagement(ctx, userID)

	assert.NoError(t, err)
	assert.NotNil(t, engagement)
	assert.Equal(t, 0, engagement.TotalClicked)
	assert.Equal(t, 0, engagement.TotalReceived)
	assert.Equal(t, float64(0), engagement.ClickRate)
}

func TestSmartNotificationService_TrackSenderEngagement(t *testing.T) {
	ts := newTestSmartNotifService()
	ctx := context.Background()
	recipientID := uuid.New()
	senderID := uuid.New()

	ts.cache.On("Get", ctx, mock.AnythingOfType("string")).Return(nil, fmt.Errorf("not found"))
	ts.cache.On("Set", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8"), 7*24*time.Hour).Return(nil)

	err := ts.service.TrackSenderEngagement(ctx, recipientID, senderID)

	assert.NoError(t, err)
}

// --- Preferences Tests ---

func TestSmartNotificationService_GetUserPreferences_Defaults(t *testing.T) {
	ts := newTestSmartNotifService()
	ctx := context.Background()
	userID := uuid.New()

	ts.cache.On("Get", ctx, mock.AnythingOfType("string")).Return(nil, fmt.Errorf("not found"))

	prefs := ts.service.GetUserPreferences(ctx, userID)

	assert.NotNil(t, prefs)
	assert.True(t, prefs.Enabled)
	assert.True(t, prefs.DigestEnabled)
	assert.Equal(t, 30, prefs.DigestIntervalMins)
	assert.True(t, prefs.UrgentAlwaysDeliver)
	assert.True(t, prefs.ClickTrackingEnabled)
}

func TestSmartNotificationService_UpdateUserPreferences(t *testing.T) {
	ts := newTestSmartNotifService()
	ctx := context.Background()
	userID := uuid.New()

	prefs := &models.SmartNotificationPreferences{
		Enabled:              true,
		DigestEnabled:        false,
		DigestIntervalMins:   60,
		UrgentAlwaysDeliver:  true,
		ClickTrackingEnabled: false,
	}

	ts.cache.On("Set", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8"), time.Duration(0)).Return(nil)

	err := ts.service.UpdateUserPreferences(ctx, userID, prefs)

	assert.NoError(t, err)
	assert.Equal(t, userID, prefs.UserID)
	ts.cache.AssertExpectations(t)
}

func TestSmartNotificationService_GetUserPreferences_Stored(t *testing.T) {
	ts := newTestSmartNotifService()
	ctx := context.Background()
	userID := uuid.New()

	stored := &models.SmartNotificationPreferences{
		UserID:              userID,
		Enabled:             false,
		DigestEnabled:       true,
		DigestIntervalMins:  60,
		UrgentAlwaysDeliver: false,
	}

	data, _ := json.Marshal(stored)
	prefsKey := fmt.Sprintf("smart_notif:prefs:%s", userID.String())
	ts.cache.On("Get", ctx, prefsKey).Return(data, nil)

	prefs := ts.service.GetUserPreferences(ctx, userID)

	assert.NotNil(t, prefs)
	assert.False(t, prefs.Enabled)
	assert.Equal(t, 60, prefs.DigestIntervalMins)
}

// --- Routing Tests ---

func TestSmartNotificationService_RouteNotification_Normal(t *testing.T) {
	ts := newTestSmartNotifService()
	ctx := context.Background()
	userID := uuid.New()

	smartNotif := &models.SmartNotification{
		PriorityScore: 50,
		Priority:      models.NotificationPriorityHigh,
		DeliveryMode:  models.DeliveryImmediate,
	}

	// Not snoozed
	ts.cache.On("Get", ctx, mock.AnythingOfType("string")).Return(nil, fmt.Errorf("not found"))

	result, err := ts.service.RouteNotification(ctx, userID, smartNotif)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.DeliveryImmediate, result.DeliveryMode)
}

func TestSmartNotificationService_RouteNotification_Snoozed(t *testing.T) {
	ts := newTestSmartNotifService()
	ctx := context.Background()
	userID := uuid.New()

	smartNotif := &models.SmartNotification{
		PriorityScore: 30,
		Priority:      models.NotificationPriorityNormal,
		DeliveryMode:  models.DeliveryBatched,
	}

	snoozeConfig := &models.SnoozeConfig{
		UserID: userID,
		Active: true,
		Until:  time.Now().Add(30 * time.Minute),
	}

	snoozeData, _ := json.Marshal(snoozeConfig)
	// First call for snooze check returns the snooze
	ts.cache.On("Get", ctx, mock.AnythingOfType("string")).Return(snoozeData, nil).Once()
	// Remaining calls (for prefs, etc.) return not found
	ts.cache.On("Get", ctx, mock.AnythingOfType("string")).Return(nil, fmt.Errorf("not found")).Maybe()

	result, err := ts.service.RouteNotification(ctx, userID, smartNotif)

	assert.NoError(t, err)
	assert.Equal(t, models.DeliveryBatched, result.DeliveryMode)
}

func TestSmartNotificationService_RouteNotification_Muted(t *testing.T) {
	ts := newTestSmartNotifService()
	ctx := context.Background()
	userID := uuid.New()

	smartNotif := &models.SmartNotification{
		PriorityScore: 80,
		Priority:      models.NotificationPriorityUrgent,
		DeliveryMode:  models.DeliveryImmediate,
	}

	// Not snoozed - the snooze check tries 3 keys, all miss
	muteConfig := &models.MuteConfig{UserID: userID, Muted: true}
	muteData, _ := json.Marshal(muteConfig)

	callCount := 0
	ts.cache.On("Get", ctx, mock.AnythingOfType("string")).Return(
		func(ctx context.Context, key string) []byte {
			callCount++
			// First 3 calls are snooze checks (not found)
			if callCount <= 3 {
				return nil
			}
			// 4th call is mute check
			if callCount == 4 {
				return muteData
			}
			return nil
		},
		func(ctx context.Context, key string) error {
			callCount++ // we need separate counters, let's simplify
			return nil
		},
	)

	// This approach is fragile with counters. Let's use a simpler mock approach.
	// Reset and use a simple map-based cache instead.
	ts = newTestSmartNotifService()
	simpleCache := &mapCache{data: make(map[string][]byte)}

	// Set up mute config in cache
	muteKey := muteKey(userID, nil, nil)
	muteBytes, _ := json.Marshal(muteConfig)
	simpleCache.data[muteKey] = muteBytes

	ts.service = NewSmartNotificationService(ts.repo, simpleCache, ts.eventBus)

	result, err := ts.service.RouteNotification(ctx, userID, smartNotif)

	assert.NoError(t, err)
	assert.Equal(t, models.DeliveryBatched, result.DeliveryMode)
}

func TestSmartNotificationService_RouteNotification_UrgentBypassesSnoozed(t *testing.T) {
	ts := newTestSmartNotifService()
	ctx := context.Background()
	userID := uuid.New()

	smartNotif := &models.SmartNotification{
		PriorityScore: 90,
		Priority:      models.NotificationPriorityUrgent,
		DeliveryMode:  models.DeliveryImmediate,
	}

	// Use simple map cache for cleaner test
	simpleCache := &mapCache{data: make(map[string][]byte)}

	// Set snooze
	sKey := snoozeKey(userID, nil, nil)
	snoozeConfig := &models.SnoozeConfig{
		UserID: userID,
		Active: true,
		Until:  time.Now().Add(30 * time.Minute),
	}
	snoozeData, _ := json.Marshal(snoozeConfig)
	simpleCache.data[sKey] = snoozeData

	// Set preferences with urgent bypass
	prefsKey := fmt.Sprintf("smart_notif:prefs:%s", userID.String())
	prefs := &models.SmartNotificationPreferences{
		UserID:              userID,
		Enabled:             true,
		UrgentAlwaysDeliver: true,
	}
	prefsData, _ := json.Marshal(prefs)
	simpleCache.data[prefsKey] = prefsData

	service := NewSmartNotificationService(ts.repo, simpleCache, ts.eventBus)

	result, err := service.RouteNotification(ctx, userID, smartNotif)

	assert.NoError(t, err)
	assert.Equal(t, models.DeliveryImmediate, result.DeliveryMode)
}

// --- Digest Tests ---

func TestSmartNotificationService_AddToDigestQueue(t *testing.T) {
	ts := newTestSmartNotifService()
	ctx := context.Background()
	userID := uuid.New()

	notif := &models.SmartNotification{
		Notification: models.Notification{
			ID:     uuid.New(),
			UserID: userID,
			Type:   models.NotificationTypeReaction,
			Title:  "Test",
		},
		PriorityScore: 20,
		Priority:      models.NotificationPriorityLow,
		DeliveryMode:  models.DeliveryBatched,
		Category:      models.NotifCategoryReaction,
	}

	ts.cache.On("Get", ctx, mock.AnythingOfType("string")).Return(nil, fmt.Errorf("not found"))
	ts.cache.On("Set", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8"), 24*time.Hour).Return(nil)

	err := ts.service.AddToDigestQueue(ctx, userID, notif)

	assert.NoError(t, err)
	ts.cache.AssertExpectations(t)
}

func TestSmartNotificationService_CreateDigest(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	simpleCache := &mapCache{data: make(map[string][]byte)}
	eventBus := new(MockEventBus)
	repo := new(MockNotificationRepository)
	service := NewSmartNotificationService(repo, simpleCache, eventBus)

	// Add some pending notifications
	pending := []models.SmartNotification{
		{
			Notification: models.Notification{ID: uuid.New(), UserID: userID, Type: models.NotificationTypeReaction, Title: "React 1"},
			Category:     models.NotifCategoryReaction,
		},
		{
			Notification: models.Notification{ID: uuid.New(), UserID: userID, Type: models.NotificationTypeMention, Title: "Mention 1"},
			Category:     models.NotifCategoryMention,
		},
	}

	pendingKey := fmt.Sprintf("smart_notif:pending_digest:%s", userID.String())
	pendingData, _ := json.Marshal(pending)
	simpleCache.data[pendingKey] = pendingData

	eventBus.On("Publish", "notification.digest_created", mock.Anything).Return()

	digest, err := service.CreateDigest(ctx, userID)

	assert.NoError(t, err)
	assert.NotNil(t, digest)
	assert.Equal(t, 2, digest.Count)
	assert.Equal(t, userID, digest.UserID)
	assert.Contains(t, digest.Title, "2 new notifications")
	assert.Len(t, digest.NotificationIDs, 2)

	eventBus.AssertExpectations(t)
}

func TestSmartNotificationService_CreateDigest_Empty(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	simpleCache := &mapCache{data: make(map[string][]byte)}
	eventBus := new(MockEventBus)
	repo := new(MockNotificationRepository)
	service := NewSmartNotificationService(repo, simpleCache, eventBus)

	digest, err := service.CreateDigest(ctx, userID)

	assert.NoError(t, err)
	assert.Nil(t, digest)
}

func TestSmartNotificationService_GetDigest(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	digestID := uuid.New()

	simpleCache := &mapCache{data: make(map[string][]byte)}
	eventBus := new(MockEventBus)
	repo := new(MockNotificationRepository)
	service := NewSmartNotificationService(repo, simpleCache, eventBus)

	now := time.Now()
	digest := &models.NotificationDigest{
		ID:        digestID,
		UserID:    userID,
		Title:     "Test digest",
		Summary:   "1 mention",
		Count:     1,
		CreatedAt: now,
	}

	key := fmt.Sprintf("smart_notif:digest:%s:%s", userID.String(), digestID.String())
	data, _ := json.Marshal(digest)
	simpleCache.data[key] = data

	result, err := service.GetDigest(ctx, userID, digestID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, digestID, result.ID)
	assert.Equal(t, "Test digest", result.Title)
}

func TestSmartNotificationService_GetDigest_NotFound(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	digestID := uuid.New()

	simpleCache := &mapCache{data: make(map[string][]byte)}
	eventBus := new(MockEventBus)
	repo := new(MockNotificationRepository)
	service := NewSmartNotificationService(repo, simpleCache, eventBus)

	result, err := service.GetDigest(ctx, userID, digestID)

	assert.Error(t, err)
	assert.Equal(t, ErrDigestNotFound, err)
	assert.Nil(t, result)
}

func TestSmartNotificationService_MarkDigestRead(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	digestID := uuid.New()

	simpleCache := &mapCache{data: make(map[string][]byte)}
	eventBus := new(MockEventBus)
	repo := new(MockNotificationRepository)
	service := NewSmartNotificationService(repo, simpleCache, eventBus)

	digest := &models.NotificationDigest{
		ID:        digestID,
		UserID:    userID,
		Title:     "Test",
		Count:     1,
		CreatedAt: time.Now(),
	}

	key := fmt.Sprintf("smart_notif:digest:%s:%s", userID.String(), digestID.String())
	data, _ := json.Marshal(digest)
	simpleCache.data[key] = data

	err := service.MarkDigestRead(ctx, userID, digestID)

	assert.NoError(t, err)

	// Verify it was updated
	result, err := service.GetDigest(ctx, userID, digestID)
	assert.NoError(t, err)
	assert.NotNil(t, result.ReadAt)
}

func TestSmartNotificationService_ListDigests(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	simpleCache := &mapCache{data: make(map[string][]byte)}
	eventBus := new(MockEventBus)
	repo := new(MockNotificationRepository)
	service := NewSmartNotificationService(repo, simpleCache, eventBus)

	// Create digest list with IDs
	d1ID := uuid.New()
	d2ID := uuid.New()

	listKey := fmt.Sprintf("smart_notif:digest_list:%s", userID.String())
	ids := []uuid.UUID{d1ID, d2ID}
	listData, _ := json.Marshal(ids)
	simpleCache.data[listKey] = listData

	// Store actual digests
	for _, id := range ids {
		d := &models.NotificationDigest{
			ID:        id,
			UserID:    userID,
			Title:     "Digest " + id.String()[:8],
			Count:     3,
			CreatedAt: time.Now(),
		}
		key := fmt.Sprintf("smart_notif:digest:%s:%s", userID.String(), id.String())
		data, _ := json.Marshal(d)
		simpleCache.data[key] = data
	}

	opts := models.DigestListOptions{Limit: 10}
	results, err := service.ListDigests(ctx, userID, opts)

	assert.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestSmartNotificationService_ListDigests_WithPagination(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	simpleCache := &mapCache{data: make(map[string][]byte)}
	eventBus := new(MockEventBus)
	repo := new(MockNotificationRepository)
	service := NewSmartNotificationService(repo, simpleCache, eventBus)

	ids := make([]uuid.UUID, 5)
	for i := range ids {
		ids[i] = uuid.New()
		d := &models.NotificationDigest{
			ID:        ids[i],
			UserID:    userID,
			Title:     fmt.Sprintf("Digest %d", i),
			Count:     1,
			CreatedAt: time.Now(),
		}
		key := fmt.Sprintf("smart_notif:digest:%s:%s", userID.String(), ids[i].String())
		data, _ := json.Marshal(d)
		simpleCache.data[key] = data
	}

	listKey := fmt.Sprintf("smart_notif:digest_list:%s", userID.String())
	listData, _ := json.Marshal(ids)
	simpleCache.data[listKey] = listData

	opts := models.DigestListOptions{Limit: 2, Offset: 1}
	results, err := service.ListDigests(ctx, userID, opts)

	assert.NoError(t, err)
	assert.Len(t, results, 2)
}

// --- Helper: mapCache for tests ---

type mapCache struct {
	data map[string][]byte
}

func (c *mapCache) Get(_ context.Context, key string) ([]byte, error) {
	if v, ok := c.data[key]; ok {
		return v, nil
	}
	return nil, fmt.Errorf("not found")
}

func (c *mapCache) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	c.data[key] = value
	return nil
}

func (c *mapCache) Delete(_ context.Context, key string) error {
	delete(c.data, key)
	return nil
}

// --- Classify Category Tests ---

func TestClassifyCategory(t *testing.T) {
	tests := []struct {
		input    models.NotificationType
		expected models.NotificationCategory
	}{
		{models.NotificationTypeMention, models.NotifCategoryMention},
		{models.NotificationTypeDirectMessage, models.NotifCategoryDirectMessage},
		{models.NotificationTypeReply, models.NotifCategoryReply},
		{models.NotificationTypeReaction, models.NotifCategoryReaction},
		{models.NotificationTypeFriendRequest, models.NotifCategorySocial},
		{models.NotificationTypeFriendAccept, models.NotifCategorySocial},
		{models.NotificationTypeServerInvite, models.NotifCategoryServerEvent},
		{models.NotificationTypeServerJoin, models.NotifCategoryServerEvent},
		{models.NotificationTypeSystem, models.NotifCategorySystem},
		{models.NotificationTypeEventInvite, models.NotifCategoryServerEvent},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			result := classifyCategory(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildDigestSummary(t *testing.T) {
	notifications := []models.SmartNotification{
		{Category: models.NotifCategoryMention},
		{Category: models.NotifCategoryMention},
		{Category: models.NotifCategoryReaction},
	}

	summary := buildDigestSummary(notifications)

	assert.NotEmpty(t, summary)
	assert.Contains(t, summary, "mention")
	assert.Contains(t, summary, "reaction")
}
