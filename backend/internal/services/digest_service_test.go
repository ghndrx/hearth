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

// --- Mock Implementations ---

type MockDigestRepository struct {
	mock.Mock
}

func (m *MockDigestRepository) GetPreferences(ctx context.Context, userID uuid.UUID) (*models.DigestPreferences, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DigestPreferences), args.Error(1)
}

func (m *MockDigestRepository) CreatePreferences(ctx context.Context, prefs *models.DigestPreferences) error {
	args := m.Called(ctx, prefs)
	return args.Error(0)
}

func (m *MockDigestRepository) UpdatePreferences(ctx context.Context, prefs *models.DigestPreferences) error {
	args := m.Called(ctx, prefs)
	return args.Error(0)
}

func (m *MockDigestRepository) UpsertPreferences(ctx context.Context, prefs *models.DigestPreferences) error {
	args := m.Called(ctx, prefs)
	return args.Error(0)
}

func (m *MockDigestRepository) GetChannelPreference(ctx context.Context, userID, channelID uuid.UUID) (*models.DigestChannelPreference, error) {
	args := m.Called(ctx, userID, channelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DigestChannelPreference), args.Error(1)
}

func (m *MockDigestRepository) GetChannelPreferences(ctx context.Context, userID uuid.UUID) ([]models.DigestChannelPreference, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]models.DigestChannelPreference), args.Error(1)
}

func (m *MockDigestRepository) UpsertChannelPreference(ctx context.Context, pref *models.DigestChannelPreference) error {
	args := m.Called(ctx, pref)
	return args.Error(0)
}

func (m *MockDigestRepository) DeleteChannelPreference(ctx context.Context, userID, channelID uuid.UUID) error {
	args := m.Called(ctx, userID, channelID)
	return args.Error(0)
}

func (m *MockDigestRepository) GetServerPreference(ctx context.Context, userID, serverID uuid.UUID) (*models.DigestServerPreference, error) {
	args := m.Called(ctx, userID, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DigestServerPreference), args.Error(1)
}

func (m *MockDigestRepository) GetServerPreferences(ctx context.Context, userID uuid.UUID) ([]models.DigestServerPreference, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]models.DigestServerPreference), args.Error(1)
}

func (m *MockDigestRepository) UpsertServerPreference(ctx context.Context, pref *models.DigestServerPreference) error {
	args := m.Called(ctx, pref)
	return args.Error(0)
}

func (m *MockDigestRepository) DeleteServerPreference(ctx context.Context, userID, serverID uuid.UUID) error {
	args := m.Called(ctx, userID, serverID)
	return args.Error(0)
}

func (m *MockDigestRepository) QueueMessage(ctx context.Context, item *models.DigestQueueItem) error {
	args := m.Called(ctx, item)
	return args.Error(0)
}

func (m *MockDigestRepository) GetQueuedItems(ctx context.Context, userID uuid.UUID, before time.Time) ([]models.DigestQueueItem, error) {
	args := m.Called(ctx, userID, before)
	return args.Get(0).([]models.DigestQueueItem), args.Error(1)
}

func (m *MockDigestRepository) GetQueuePreview(ctx context.Context, userID uuid.UUID) (*models.DigestPreview, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DigestPreview), args.Error(1)
}

func (m *MockDigestRepository) DeleteQueuedItems(ctx context.Context, userID uuid.UUID, before time.Time) (int64, error) {
	args := m.Called(ctx, userID, before)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockDigestRepository) ClearQueue(ctx context.Context, userID uuid.UUID) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockDigestRepository) CreateHistory(ctx context.Context, history *models.DigestHistory) error {
	args := m.Called(ctx, history)
	return args.Error(0)
}

func (m *MockDigestRepository) UpdateHistoryStatus(ctx context.Context, id uuid.UUID, status models.DigestStatus, errorMessage *string) error {
	args := m.Called(ctx, id, status, errorMessage)
	return args.Error(0)
}

func (m *MockDigestRepository) GetHistory(ctx context.Context, userID uuid.UUID, opts models.DigestHistoryListOptions) ([]models.DigestHistory, error) {
	args := m.Called(ctx, userID, opts)
	return args.Get(0).([]models.DigestHistory), args.Error(1)
}

func (m *MockDigestRepository) GetHistoryByID(ctx context.Context, id uuid.UUID) (*models.DigestHistory, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DigestHistory), args.Error(1)
}

func (m *MockDigestRepository) GetLastDigest(ctx context.Context, userID uuid.UUID) (*models.DigestHistory, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DigestHistory), args.Error(1)
}

func (m *MockDigestRepository) GetUsersForDigest(ctx context.Context, frequency models.DigestFrequency, hour int, day int) ([]uuid.UUID, error) {
	args := m.Called(ctx, frequency, hour, day)
	return args.Get(0).([]uuid.UUID), args.Error(1)
}

func (m *MockDigestRepository) GetPendingDigests(ctx context.Context, limit int) ([]models.DigestHistory, error) {
	args := m.Called(ctx, limit)
	return args.Get(0).([]models.DigestHistory), args.Error(1)
}

type MockServerRepo struct {
	mock.Mock
}

func (m *MockServerRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Server, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Server), args.Error(1)
}

type MockDigestEventBus struct {
	publishedEvents []string
	publishedData   []interface{}
}

func (m *MockDigestEventBus) Publish(event string, data interface{}) {
	m.publishedEvents = append(m.publishedEvents, event)
	m.publishedData = append(m.publishedData, data)
}

func (m *MockDigestEventBus) Subscribe(event string, handler func(data interface{}))   {}
func (m *MockDigestEventBus) Unsubscribe(event string, handler func(data interface{})) {}

// --- Tests ---

func TestDigestService_GetPreferences_NotFound(t *testing.T) {
	repo := new(MockDigestRepository)
	eventBus := &MockDigestEventBus{}
	service := NewDigestService(repo, nil, eventBus)

	ctx := context.Background()
	userID := uuid.New()

	repo.On("GetPreferences", ctx, userID).Return(nil, nil)

	prefs, err := service.GetPreferences(ctx, userID)

	assert.NoError(t, err)
	assert.NotNil(t, prefs)
	assert.Equal(t, userID, prefs.UserID)
	assert.False(t, prefs.Enabled)
	assert.Equal(t, models.DigestFrequencyDaily, prefs.Frequency)
	repo.AssertExpectations(t)
}

func TestDigestService_GetPreferences_Found(t *testing.T) {
	repo := new(MockDigestRepository)
	eventBus := &MockDigestEventBus{}
	service := NewDigestService(repo, nil, eventBus)

	ctx := context.Background()
	userID := uuid.New()

	existingPrefs := &models.DigestPreferences{
		ID:            uuid.New(),
		UserID:        userID,
		Enabled:       true,
		Frequency:     models.DigestFrequencyHourly,
		PreferredHour: 14,
	}

	repo.On("GetPreferences", ctx, userID).Return(existingPrefs, nil)

	prefs, err := service.GetPreferences(ctx, userID)

	assert.NoError(t, err)
	assert.NotNil(t, prefs)
	assert.True(t, prefs.Enabled)
	assert.Equal(t, models.DigestFrequencyHourly, prefs.Frequency)
	assert.Equal(t, 14, prefs.PreferredHour)
	repo.AssertExpectations(t)
}

func TestDigestService_UpdatePreferences_Enable(t *testing.T) {
	repo := new(MockDigestRepository)
	eventBus := &MockDigestEventBus{}
	service := NewDigestService(repo, nil, eventBus)

	ctx := context.Background()
	userID := uuid.New()

	repo.On("GetPreferences", ctx, userID).Return(nil, nil)
	repo.On("UpsertPreferences", ctx, mock.AnythingOfType("*models.DigestPreferences")).Return(nil)

	enabled := true
	req := &models.UpdateDigestPreferencesRequest{
		Enabled: &enabled,
	}

	prefs, err := service.UpdatePreferences(ctx, userID, req)

	assert.NoError(t, err)
	assert.NotNil(t, prefs)
	assert.True(t, prefs.Enabled)
	assert.Len(t, eventBus.publishedEvents, 1)
	assert.Equal(t, "digest.preferences_updated", eventBus.publishedEvents[0])
	repo.AssertExpectations(t)
}

func TestDigestService_UpdatePreferences_InvalidFrequency(t *testing.T) {
	repo := new(MockDigestRepository)
	eventBus := &MockDigestEventBus{}
	service := NewDigestService(repo, nil, eventBus)

	ctx := context.Background()
	userID := uuid.New()

	repo.On("GetPreferences", ctx, userID).Return(nil, nil)

	invalidFreq := models.DigestFrequency("invalid")
	req := &models.UpdateDigestPreferencesRequest{
		Frequency: &invalidFreq,
	}

	_, err := service.UpdatePreferences(ctx, userID, req)

	assert.Error(t, err)
	assert.Equal(t, ErrInvalidFrequency, err)
	repo.AssertExpectations(t)
}

func TestDigestService_UpdatePreferences_InvalidTimezone(t *testing.T) {
	repo := new(MockDigestRepository)
	eventBus := &MockDigestEventBus{}
	service := NewDigestService(repo, nil, eventBus)

	ctx := context.Background()
	userID := uuid.New()

	repo.On("GetPreferences", ctx, userID).Return(nil, nil)

	invalidTZ := "Invalid/Timezone"
	req := &models.UpdateDigestPreferencesRequest{
		Timezone: &invalidTZ,
	}

	_, err := service.UpdatePreferences(ctx, userID, req)

	assert.Error(t, err)
	assert.Equal(t, ErrInvalidTimezone, err)
	repo.AssertExpectations(t)
}

func TestDigestService_UpdatePreferences_ValidTimezone(t *testing.T) {
	repo := new(MockDigestRepository)
	eventBus := &MockDigestEventBus{}
	service := NewDigestService(repo, nil, eventBus)

	ctx := context.Background()
	userID := uuid.New()

	repo.On("GetPreferences", ctx, userID).Return(nil, nil)
	repo.On("UpsertPreferences", ctx, mock.AnythingOfType("*models.DigestPreferences")).Return(nil)

	validTZ := "America/New_York"
	req := &models.UpdateDigestPreferencesRequest{
		Timezone: &validTZ,
	}

	prefs, err := service.UpdatePreferences(ctx, userID, req)

	assert.NoError(t, err)
	assert.Equal(t, validTZ, prefs.Timezone)
	repo.AssertExpectations(t)
}

func TestDigestService_UpdateChannelPreference_Include(t *testing.T) {
	repo := new(MockDigestRepository)
	eventBus := &MockDigestEventBus{}
	service := NewDigestService(repo, nil, eventBus)

	ctx := context.Background()
	userID := uuid.New()
	channelID := uuid.New()

	repo.On("UpsertChannelPreference", ctx, mock.AnythingOfType("*models.DigestChannelPreference")).Return(nil)

	err := service.UpdateChannelPreference(ctx, userID, channelID, models.DigestModeInclude)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestDigestService_UpdateChannelPreference_InheritDeletes(t *testing.T) {
	repo := new(MockDigestRepository)
	eventBus := &MockDigestEventBus{}
	service := NewDigestService(repo, nil, eventBus)

	ctx := context.Background()
	userID := uuid.New()
	channelID := uuid.New()

	repo.On("DeleteChannelPreference", ctx, userID, channelID).Return(nil)

	err := service.UpdateChannelPreference(ctx, userID, channelID, models.DigestModeInherit)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestDigestService_UpdateServerPreference_ImmediateFails(t *testing.T) {
	repo := new(MockDigestRepository)
	eventBus := &MockDigestEventBus{}
	service := NewDigestService(repo, nil, eventBus)

	ctx := context.Background()
	userID := uuid.New()
	serverID := uuid.New()

	err := service.UpdateServerPreference(ctx, userID, serverID, models.DigestModeImmediate)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "immediate mode")
}

func TestDigestService_QueueNotification_DigestDisabled(t *testing.T) {
	repo := new(MockDigestRepository)
	eventBus := &MockDigestEventBus{}
	service := NewDigestService(repo, nil, eventBus)

	ctx := context.Background()
	userID := uuid.New()

	repo.On("GetPreferences", ctx, userID).Return(nil, nil)

	notification := &models.Notification{
		UserID: userID,
		Type:   models.NotificationTypeMention,
	}

	err := service.QueueNotification(ctx, userID, notification, "test content", "TestUser", time.Now())

	assert.Error(t, err)
	assert.Equal(t, ErrDigestDisabled, err)
	repo.AssertExpectations(t)
}

func TestDigestService_QueueNotification_Success(t *testing.T) {
	repo := new(MockDigestRepository)
	eventBus := &MockDigestEventBus{}
	service := NewDigestService(repo, nil, eventBus)

	ctx := context.Background()
	userID := uuid.New()

	prefs := &models.DigestPreferences{
		UserID:    userID,
		Enabled:   true,
		Frequency: models.DigestFrequencyDaily,
	}

	repo.On("GetPreferences", ctx, userID).Return(prefs, nil)
	repo.On("QueueMessage", ctx, mock.AnythingOfType("*models.DigestQueueItem")).Return(nil)

	notification := &models.Notification{
		UserID: userID,
		Type:   models.NotificationTypeMention,
	}

	err := service.QueueNotification(ctx, userID, notification, "test content", "TestUser", time.Now())

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestDigestService_GetDigestPreview(t *testing.T) {
	repo := new(MockDigestRepository)
	eventBus := &MockDigestEventBus{}
	service := NewDigestService(repo, nil, eventBus)

	ctx := context.Background()
	userID := uuid.New()

	prefs := models.DefaultDigestPreferences(userID)
	repo.On("GetPreferences", ctx, userID).Return(prefs, nil)

	oldestTime := time.Now().Add(-2 * time.Hour)
	preview := &models.DigestPreview{
		PendingCount:    10,
		PendingMentions: 3,
		PendingServers:  2,
		PendingChannels: 5,
		OldestPending:   &oldestTime,
	}
	repo.On("GetQueuePreview", ctx, userID).Return(preview, nil)

	result, err := service.GetDigestPreview(ctx, userID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 10, result.PendingCount)
	assert.Equal(t, 3, result.PendingMentions)
	assert.NotZero(t, result.NextDigestAt)
	repo.AssertExpectations(t)
}

func TestDigestService_GenerateDigest_NoItems(t *testing.T) {
	repo := new(MockDigestRepository)
	eventBus := &MockDigestEventBus{}
	service := NewDigestService(repo, nil, eventBus)

	ctx := context.Background()
	userID := uuid.New()

	prefs := &models.DigestPreferences{
		UserID:    userID,
		Enabled:   true,
		Frequency: models.DigestFrequencyDaily,
	}

	repo.On("GetPreferences", ctx, userID).Return(prefs, nil)
	repo.On("GetQueuedItems", ctx, userID, mock.AnythingOfType("time.Time")).Return([]models.DigestQueueItem{}, nil)
	repo.On("CreateHistory", ctx, mock.AnythingOfType("*models.DigestHistory")).Return(nil)

	digest, err := service.GenerateDigest(ctx, userID)

	assert.NoError(t, err)
	assert.NotNil(t, digest)
	assert.Equal(t, models.DigestStatusSkipped, digest.Status)
	repo.AssertExpectations(t)
}

func TestDigestService_GenerateDigest_WithItems(t *testing.T) {
	repo := new(MockDigestRepository)
	eventBus := &MockDigestEventBus{}
	service := NewDigestService(repo, nil, eventBus)

	ctx := context.Background()
	userID := uuid.New()
	serverID := uuid.New()
	channelID := uuid.New()

	prefs := &models.DigestPreferences{
		UserID:               userID,
		Enabled:              true,
		Frequency:            models.DigestFrequencyDaily,
		MaxMessagesPerSource: 50,
	}

	items := []models.DigestQueueItem{
		{
			ID:                uuid.New(),
			UserID:            userID,
			ServerID:          &serverID,
			ChannelID:         &channelID,
			MessageContent:    "Hello world",
			MessageAuthorName: "TestUser",
			MessageCreatedAt:  time.Now().Add(-1 * time.Hour),
			IsMention:         true,
			NotificationType:  models.NotificationTypeMention,
		},
		{
			ID:                uuid.New(),
			UserID:            userID,
			ServerID:          &serverID,
			ChannelID:         &channelID,
			MessageContent:    "Another message",
			MessageAuthorName: "AnotherUser",
			MessageCreatedAt:  time.Now().Add(-30 * time.Minute),
			IsMention:         false,
			NotificationType:  models.NotificationTypeReply,
		},
	}

	repo.On("GetPreferences", ctx, userID).Return(prefs, nil)
	repo.On("GetQueuedItems", ctx, userID, mock.AnythingOfType("time.Time")).Return(items, nil)
	repo.On("CreateHistory", ctx, mock.AnythingOfType("*models.DigestHistory")).Return(nil)
	repo.On("DeleteQueuedItems", ctx, userID, mock.AnythingOfType("time.Time")).Return(int64(2), nil)

	digest, err := service.GenerateDigest(ctx, userID)

	assert.NoError(t, err)
	assert.NotNil(t, digest)
	assert.Equal(t, models.DigestStatusPending, digest.Status)
	assert.Equal(t, 2, digest.TotalMessages)
	assert.Equal(t, 1, digest.TotalMentions)
	assert.Len(t, eventBus.publishedEvents, 1)
	assert.Equal(t, "digest.generated", eventBus.publishedEvents[0])
	repo.AssertExpectations(t)
}

func TestDigestService_GetDigestByID_NotFound(t *testing.T) {
	repo := new(MockDigestRepository)
	eventBus := &MockDigestEventBus{}
	service := NewDigestService(repo, nil, eventBus)

	ctx := context.Background()
	userID := uuid.New()
	digestID := uuid.New()

	repo.On("GetHistoryByID", ctx, digestID).Return(nil, nil)

	_, err := service.GetDigestByID(ctx, userID, digestID)

	assert.Error(t, err)
	assert.Equal(t, ErrDigestNotFound, err)
	repo.AssertExpectations(t)
}

func TestDigestService_GetDigestByID_WrongUser(t *testing.T) {
	repo := new(MockDigestRepository)
	eventBus := &MockDigestEventBus{}
	service := NewDigestService(repo, nil, eventBus)

	ctx := context.Background()
	userID := uuid.New()
	otherUserID := uuid.New()
	digestID := uuid.New()

	digest := &models.DigestHistory{
		ID:     digestID,
		UserID: otherUserID,
	}

	repo.On("GetHistoryByID", ctx, digestID).Return(digest, nil)

	_, err := service.GetDigestByID(ctx, userID, digestID)

	assert.Error(t, err)
	assert.Equal(t, ErrDigestNotFound, err)
	repo.AssertExpectations(t)
}

func TestDigestService_GetDigestHistory(t *testing.T) {
	repo := new(MockDigestRepository)
	eventBus := &MockDigestEventBus{}
	service := NewDigestService(repo, nil, eventBus)

	ctx := context.Background()
	userID := uuid.New()

	history := []models.DigestHistory{
		{ID: uuid.New(), UserID: userID, Status: models.DigestStatusSent},
		{ID: uuid.New(), UserID: userID, Status: models.DigestStatusSent},
	}

	opts := models.DigestHistoryListOptions{Limit: 20, Offset: 0}
	repo.On("GetHistory", ctx, userID, opts).Return(history, nil)

	result, err := service.GetDigestHistory(ctx, userID, opts)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	repo.AssertExpectations(t)
}

func TestDigestService_ClearDigestQueue(t *testing.T) {
	repo := new(MockDigestRepository)
	eventBus := &MockDigestEventBus{}
	service := NewDigestService(repo, nil, eventBus)

	ctx := context.Background()
	userID := uuid.New()

	repo.On("ClearQueue", ctx, userID).Return(int64(15), nil)

	count, err := service.ClearDigestQueue(ctx, userID)

	assert.NoError(t, err)
	assert.Equal(t, int64(15), count)
	repo.AssertExpectations(t)
}

func TestDigestService_CalculateDigestPeriod(t *testing.T) {
	repo := new(MockDigestRepository)
	eventBus := &MockDigestEventBus{}
	service := NewDigestService(repo, nil, eventBus)

	testTime := time.Date(2024, 3, 15, 14, 30, 45, 0, time.UTC)

	// Hourly
	hourlyPeriod := service.calculateDigestPeriod(models.DigestFrequencyHourly, testTime)
	assert.Equal(t, 14, hourlyPeriod.Hour())
	assert.Equal(t, 0, hourlyPeriod.Minute())

	// Daily
	dailyPeriod := service.calculateDigestPeriod(models.DigestFrequencyDaily, testTime)
	assert.Equal(t, 0, dailyPeriod.Hour())
	assert.Equal(t, 15, dailyPeriod.Day())

	// Weekly
	weeklyPeriod := service.calculateDigestPeriod(models.DigestFrequencyWeekly, testTime)
	assert.Equal(t, 10, weeklyPeriod.Day()) // Sunday before March 15
}

func TestDigestService_CalculateNextDigestTime(t *testing.T) {
	repo := new(MockDigestRepository)
	eventBus := &MockDigestEventBus{}
	service := NewDigestService(repo, nil, eventBus)

	// Hourly - should be start of next hour
	hourlyPrefs := &models.DigestPreferences{
		Frequency: models.DigestFrequencyHourly,
		Timezone:  "UTC",
	}
	hourlyNext := service.calculateNextDigestTime(hourlyPrefs)
	assert.Equal(t, 0, hourlyNext.Minute())
	assert.True(t, hourlyNext.After(time.Now()))

	// Daily - should be at preferred hour
	dailyPrefs := &models.DigestPreferences{
		Frequency:     models.DigestFrequencyDaily,
		PreferredHour: 9,
		Timezone:      "UTC",
	}
	dailyNext := service.calculateNextDigestTime(dailyPrefs)
	assert.Equal(t, 9, dailyNext.Hour())
	assert.True(t, dailyNext.After(time.Now()) || dailyNext.Equal(time.Now().Truncate(time.Hour)))
}

func TestTruncateContent(t *testing.T) {
	// Short content
	short := "Hello world"
	assert.Equal(t, short, truncateContent(short, 50))

	// Long content
	long := "This is a very long message that should be truncated to fit within the maximum length limit"
	truncated := truncateContent(long, 30)
	assert.Len(t, truncated, 30)
	assert.True(t, len(truncated) <= 30)
	assert.Equal(t, "...", truncated[len(truncated)-3:])
}

func TestValidateFrequency(t *testing.T) {
	assert.True(t, models.ValidateFrequency(models.DigestFrequencyHourly))
	assert.True(t, models.ValidateFrequency(models.DigestFrequencyDaily))
	assert.True(t, models.ValidateFrequency(models.DigestFrequencyWeekly))
	assert.False(t, models.ValidateFrequency("invalid"))
}

func TestValidateAggregationMode(t *testing.T) {
	assert.True(t, models.ValidateAggregationMode(models.DigestAggregationChannel))
	assert.True(t, models.ValidateAggregationMode(models.DigestAggregationServer))
	assert.False(t, models.ValidateAggregationMode("invalid"))
}

func TestValidateDigestMode(t *testing.T) {
	assert.True(t, models.ValidateDigestMode(models.DigestModeInherit))
	assert.True(t, models.ValidateDigestMode(models.DigestModeInclude))
	assert.True(t, models.ValidateDigestMode(models.DigestModeExclude))
	assert.True(t, models.ValidateDigestMode(models.DigestModeImmediate))
	assert.False(t, models.ValidateDigestMode("invalid"))
}

func TestDefaultDigestPreferences(t *testing.T) {
	userID := uuid.New()
	prefs := models.DefaultDigestPreferences(userID)

	assert.Equal(t, userID, prefs.UserID)
	assert.False(t, prefs.Enabled)
	assert.Equal(t, models.DigestFrequencyDaily, prefs.Frequency)
	assert.Equal(t, 9, prefs.PreferredHour)
	assert.Equal(t, 1, prefs.PreferredDay)
	assert.Equal(t, models.DigestAggregationServer, prefs.AggregationMode)
	assert.Equal(t, 50, prefs.MaxMessagesPerSource)
	assert.True(t, prefs.MutedChannelsOnly)
	assert.Equal(t, "UTC", prefs.Timezone)
}
