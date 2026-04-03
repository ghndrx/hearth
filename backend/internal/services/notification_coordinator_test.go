package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"hearth/internal/models"
)

type mockNotificationQueueRepo struct {
	items    []models.NotificationQueueItem
	createFn func(ctx context.Context, item *models.NotificationQueueItem) error
}

func (m *mockNotificationQueueRepo) Create(ctx context.Context, item *models.NotificationQueueItem) error {
	if m.createFn != nil {
		return m.createFn(ctx, item)
	}
	m.items = append(m.items, *item)
	return nil
}

func (m *mockNotificationQueueRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status models.QueueItemStatus, lastError *string) error {
	return nil
}

func (m *mockNotificationQueueRepo) MarkProcessed(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockNotificationQueueRepo) IncrementAttempts(ctx context.Context, id uuid.UUID, lastError string) error {
	return nil
}

func (m *mockNotificationQueueRepo) GetPending(ctx context.Context, limit int) ([]models.NotificationQueueItem, error) {
	return m.items, nil
}

func (m *mockNotificationQueueRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

type mockChannelPrefsRepo struct {
	prefs map[uuid.UUID]*models.ChannelNotificationPreference
}

func (m *mockChannelPrefsRepo) Get(ctx context.Context, userID, channelID uuid.UUID) (*models.ChannelNotificationPreference, error) {
	if m.prefs == nil {
		return nil, assert.AnError
	}
	return m.prefs[channelID], nil
}

func (m *mockChannelPrefsRepo) Upsert(ctx context.Context, pref *models.ChannelNotificationPreference) error {
	if m.prefs == nil {
		m.prefs = make(map[uuid.UUID]*models.ChannelNotificationPreference)
	}
	m.prefs[pref.ChannelID] = pref
	return nil
}

func (m *mockChannelPrefsRepo) GetByUser(ctx context.Context, userID uuid.UUID) ([]models.ChannelNotificationPreference, error) {
	return nil, nil
}

func (m *mockChannelPrefsRepo) Delete(ctx context.Context, userID, channelID uuid.UUID) error {
	return nil
}

type mockServerPrefsRepo struct {
	prefs map[uuid.UUID]*models.ServerNotificationPreference
}

func (m *mockServerPrefsRepo) Get(ctx context.Context, userID, serverID uuid.UUID) (*models.ServerNotificationPreference, error) {
	if m.prefs == nil {
		return nil, assert.AnError
	}
	return m.prefs[serverID], nil
}

func (m *mockServerPrefsRepo) Upsert(ctx context.Context, pref *models.ServerNotificationPreference) error {
	if m.prefs == nil {
		m.prefs = make(map[uuid.UUID]*models.ServerNotificationPreference)
	}
	m.prefs[pref.ServerID] = pref
	return nil
}

func (m *mockServerPrefsRepo) GetByUser(ctx context.Context, userID uuid.UUID) ([]models.ServerNotificationPreference, error) {
	return nil, nil
}

func (m *mockServerPrefsRepo) Delete(ctx context.Context, userID, serverID uuid.UUID) error {
	return nil
}

func TestNotificationCoordinator_GetChannelPreference(t *testing.T) {
	coordinator := &NotificationCoordinator{
		channelPrefsRepo: &mockChannelPrefsRepo{},
	}

	userID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	pref, err := coordinator.GetChannelPreference(context.Background(), userID, channelID, serverID)

	assert.NoError(t, err)
	assert.NotNil(t, pref)
	assert.Equal(t, userID, pref.UserID)
	assert.Equal(t, channelID, pref.ChannelID)
	assert.Equal(t, serverID, pref.ServerID)
	assert.True(t, pref.EnableMentions)
	assert.True(t, pref.EnableMessages)
	assert.False(t, pref.Muted)
}

func TestNotificationCoordinator_GetServerPreference(t *testing.T) {
	coordinator := &NotificationCoordinator{
		serverPrefsRepo: &mockServerPrefsRepo{},
	}

	userID := uuid.New()
	serverID := uuid.New()

	pref, err := coordinator.GetServerPreference(context.Background(), userID, serverID)

	assert.NoError(t, err)
	assert.NotNil(t, pref)
	assert.Equal(t, userID, pref.UserID)
	assert.Equal(t, serverID, pref.ServerID)
	assert.True(t, pref.EnableMentions)
	assert.True(t, pref.EnableMessages)
	assert.False(t, pref.Muted)
}

func TestNotificationCoordinator_UpdateChannelPreference(t *testing.T) {
	repo := &mockChannelPrefsRepo{
		prefs: make(map[uuid.UUID]*models.ChannelNotificationPreference),
	}
	coordinator := &NotificationCoordinator{
		channelPrefsRepo: repo,
	}

	userID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	muted := true
	req := &models.UpdateChannelNotificationPreferenceRequest{
		Muted: &muted,
	}

	pref, err := coordinator.UpdateChannelPreference(context.Background(), userID, channelID, serverID, req)

	assert.NoError(t, err)
	assert.NotNil(t, pref)
	assert.True(t, pref.Muted)

	// Verify it was persisted
	saved := repo.prefs[channelID]
	assert.NotNil(t, saved)
	assert.True(t, saved.Muted)
}

func TestNotificationCoordinator_UpdateServerPreference(t *testing.T) {
	repo := &mockServerPrefsRepo{
		prefs: make(map[uuid.UUID]*models.ServerNotificationPreference),
	}
	coordinator := &NotificationCoordinator{
		serverPrefsRepo: repo,
	}

	userID := uuid.New()
	serverID := uuid.New()

	mentions := false
	req := &models.UpdateServerNotificationPreferenceRequest{
		EnableMentions: &mentions,
	}

	pref, err := coordinator.UpdateServerPreference(context.Background(), userID, serverID, req)

	assert.NoError(t, err)
	assert.NotNil(t, pref)
	assert.False(t, pref.EnableMentions)

	// Verify it was persisted
	saved := repo.prefs[serverID]
	assert.NotNil(t, saved)
	assert.False(t, saved.EnableMentions)
}

func TestNotificationCoordinator_isNotificationTypeEnabled(t *testing.T) {
	coordinator := &NotificationCoordinator{}

	tests := []struct {
		name      string
		notifType models.NotificationType
		pref      *models.ChannelNotificationPreference
		expected  bool
	}{
		{
			name:      "mention enabled",
			notifType: models.NotificationTypeMention,
			pref:      &models.ChannelNotificationPreference{EnableMentions: true},
			expected:  true,
		},
		{
			name:      "mention disabled",
			notifType: models.NotificationTypeMention,
			pref:      &models.ChannelNotificationPreference{EnableMentions: false},
			expected:  false,
		},
		{
			name:      "DM uses messages setting",
			notifType: models.NotificationTypeDirectMessage,
			pref:      &models.ChannelNotificationPreference{EnableMessages: true},
			expected:  true,
		},
		{
			name:      "reaction uses reactions setting",
			notifType: models.NotificationTypeReaction,
			pref:      &models.ChannelNotificationPreference{EnableReactions: false},
			expected:  false,
		},
		{
			name:      "unknown type defaults to enabled",
			notifType: models.NotificationTypeSystem,
			pref:      &models.ChannelNotificationPreference{EnableMentions: false},
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := coordinator.isNotificationTypeEnabled(tt.notifType, tt.pref)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNotificationCoordinator_deliveryModeToChannel(t *testing.T) {
	coordinator := &NotificationCoordinator{}

	tests := []struct {
		mode     models.NotificationDeliveryMode
		expected models.NotificationChannel
	}{
		{mode: models.DeliveryImmediate, expected: models.NotificationChannelPush},
		{mode: models.DeliveryBatched, expected: models.NotificationChannelInApp},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			result := coordinator.deliveryModeToChannel(tt.mode)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNotificationCoordinator_computeDelay(t *testing.T) {
	coordinator := &NotificationCoordinator{}

	tests := []struct {
		name     string
		mode     models.NotificationDeliveryMode
		priority models.NotificationPriority
		expected int
	}{
		{
			name:     "urgent always immediate",
			mode:     models.DeliveryBatched,
			priority: models.NotificationPriorityUrgent,
			expected: 0,
		},
		{
			name:     "immediate mode immediate",
			mode:     models.DeliveryImmediate,
			priority: models.NotificationPriorityNormal,
			expected: 0,
		},
		{
			name:     "batched has delay",
			mode:     models.DeliveryBatched,
			priority: models.NotificationPriorityNormal,
			expected: 30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := coordinator.computeDelay(tt.mode, tt.priority)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDefaultChannelNotificationPreference(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	pref := models.DefaultChannelNotificationPreference(userID, channelID, serverID)

	assert.Equal(t, userID, pref.UserID)
	assert.Equal(t, channelID, pref.ChannelID)
	assert.Equal(t, serverID, pref.ServerID)
	assert.True(t, pref.EnableMentions)
	assert.True(t, pref.EnableMessages)
	assert.True(t, pref.EnableReactions)
	assert.True(t, pref.EnableThreads)
	assert.True(t, pref.EnablePins)
	assert.True(t, pref.EnableVoiceActivity)
	assert.Equal(t, models.DeliveryBatched, pref.DeliveryMode)
	assert.False(t, pref.Muted)
	assert.False(t, pref.CreatedAt.IsZero())
	assert.False(t, pref.UpdatedAt.IsZero())
}

func TestDefaultServerNotificationPreference(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()

	pref := models.DefaultServerNotificationPreference(userID, serverID)

	assert.Equal(t, userID, pref.UserID)
	assert.Equal(t, serverID, pref.ServerID)
	assert.True(t, pref.EnableMentions)
	assert.True(t, pref.EnableMessages)
	assert.True(t, pref.EnableReactions)
	assert.True(t, pref.EnableThreads)
	assert.False(t, pref.Muted)
	assert.Nil(t, pref.MutedUntil)
	assert.False(t, pref.CreatedAt.IsZero())
	assert.False(t, pref.UpdatedAt.IsZero())
}
