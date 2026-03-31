package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"hearth/internal/models"
)

// simpleMockThreadAutoArchiveRepo is a simple mock for ThreadAutoArchiveRepositoryInterface
type simpleMockThreadAutoArchiveRepo struct {
	serverSettings   *models.ThreadAutoArchiveSettings
	channelOverride *models.ChannelAutoArchiveOverride
	threadMeta      *models.ThreadAutoArchiveMeta
	serverStats     *models.ThreadAutoArchiveStats
}

func (m *simpleMockThreadAutoArchiveRepo) CreateServerSettings(ctx context.Context, settings *models.ThreadAutoArchiveSettings) error {
	m.serverSettings = settings
	return nil
}

func (m *simpleMockThreadAutoArchiveRepo) GetServerSettings(ctx context.Context, serverID uuid.UUID) (*models.ThreadAutoArchiveSettings, error) {
	return m.serverSettings, nil
}

func (m *simpleMockThreadAutoArchiveRepo) UpdateServerSettings(ctx context.Context, settings *models.ThreadAutoArchiveSettings) error {
	m.serverSettings = settings
	return nil
}

func (m *simpleMockThreadAutoArchiveRepo) DeleteServerSettings(ctx context.Context, serverID uuid.UUID) error {
	m.serverSettings = nil
	return nil
}

func (m *simpleMockThreadAutoArchiveRepo) SetChannelOverride(ctx context.Context, override *models.ChannelAutoArchiveOverride) error {
	m.channelOverride = override
	return nil
}

func (m *simpleMockThreadAutoArchiveRepo) GetChannelOverride(ctx context.Context, channelID uuid.UUID) (*models.ChannelAutoArchiveOverride, error) {
	return m.channelOverride, nil
}

func (m *simpleMockThreadAutoArchiveRepo) DeleteChannelOverride(ctx context.Context, channelID uuid.UUID) error {
	m.channelOverride = nil
	return nil
}

func (m *simpleMockThreadAutoArchiveRepo) GetOrCreateThreadMeta(ctx context.Context, threadID uuid.UUID) (*models.ThreadAutoArchiveMeta, error) {
	return m.threadMeta, nil
}

func (m *simpleMockThreadAutoArchiveRepo) UpdateThreadMeta(ctx context.Context, meta *models.ThreadAutoArchiveMeta) error {
	m.threadMeta = meta
	return nil
}

func (m *simpleMockThreadAutoArchiveRepo) SetThreadNextArchive(ctx context.Context, threadID uuid.UUID, nextArchiveAt *time.Time) error {
	return nil
}

func (m *simpleMockThreadAutoArchiveRepo) SetThreadArchiveEligible(ctx context.Context, threadID uuid.UUID, eligible bool) error {
	return nil
}

func (m *simpleMockThreadAutoArchiveRepo) BumpThreadOwnerActivity(ctx context.Context, threadID uuid.UUID) error {
	return nil
}

func (m *simpleMockThreadAutoArchiveRepo) GetThreadsReadyForArchive(ctx context.Context, limit int) ([]*models.ThreadAutoArchiveMeta, error) {
	return nil, nil
}

func (m *simpleMockThreadAutoArchiveRepo) GetThreadMeta(ctx context.Context, threadID uuid.UUID) (*models.ThreadAutoArchiveMeta, error) {
	return m.threadMeta, nil
}

func (m *simpleMockThreadAutoArchiveRepo) DeleteThreadMeta(ctx context.Context, threadID uuid.UUID) error {
	m.threadMeta = nil
	return nil
}

func (m *simpleMockThreadAutoArchiveRepo) GetServerStats(ctx context.Context, serverID uuid.UUID) (*models.ThreadAutoArchiveStats, error) {
	return m.serverStats, nil
}

func (m *simpleMockThreadAutoArchiveRepo) GetChannelDuration(ctx context.Context, channelID, serverID uuid.UUID) (int, error) {
	return 1440, nil
}

// simpleMockThreadRepo is a simple mock for ThreadRepository
type simpleMockThreadRepo struct {
	thread *models.Thread
}

func (m *simpleMockThreadRepo) Create(ctx context.Context, thread *models.Thread) error {
	return nil
}

func (m *simpleMockThreadRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Thread, error) {
	return m.thread, nil
}

func (m *simpleMockThreadRepo) GetByParentMessageID(ctx context.Context, messageID uuid.UUID) (*models.Thread, error) {
	return nil, nil
}

func (m *simpleMockThreadRepo) Update(ctx context.Context, thread *models.Thread) error {
	return nil
}

func (m *simpleMockThreadRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *simpleMockThreadRepo) GetByChannelID(ctx context.Context, channelID uuid.UUID) ([]*models.Thread, error) {
	return nil, nil
}

func (m *simpleMockThreadRepo) GetActiveByChannelID(ctx context.Context, channelID uuid.UUID) ([]*models.Thread, error) {
	return nil, nil
}

func (m *simpleMockThreadRepo) Archive(ctx context.Context, id uuid.UUID) error {
	if m.thread != nil {
		m.thread.Archived = true
	}
	return nil
}

func (m *simpleMockThreadRepo) Unarchive(ctx context.Context, id uuid.UUID) error {
	if m.thread != nil {
		m.thread.Archived = false
	}
	return nil
}

func (m *simpleMockThreadRepo) AddMember(ctx context.Context, threadID, userID uuid.UUID) error {
	return nil
}

func (m *simpleMockThreadRepo) RemoveMember(ctx context.Context, threadID, userID uuid.UUID) error {
	return nil
}

func (m *simpleMockThreadRepo) IsMember(ctx context.Context, threadID, userID uuid.UUID) (bool, error) {
	return true, nil
}

func (m *simpleMockThreadRepo) GetMembers(ctx context.Context, threadID uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}

func (m *simpleMockThreadRepo) CreateMessage(ctx context.Context, threadID, authorID uuid.UUID, content string) (*models.ThreadMessage, error) {
	return nil, nil
}

func (m *simpleMockThreadRepo) GetMessages(ctx context.Context, threadID uuid.UUID, before *uuid.UUID, limit int) ([]*models.ThreadMessage, error) {
	return nil, nil
}

func (m *simpleMockThreadRepo) IncrementMessageCount(ctx context.Context, threadID uuid.UUID) error {
	return nil
}

func (m *simpleMockThreadRepo) GetNotificationPreference(ctx context.Context, threadID, userID uuid.UUID) (*models.ThreadNotificationPreference, error) {
	return nil, nil
}

func (m *simpleMockThreadRepo) SetNotificationPreference(ctx context.Context, pref *models.ThreadNotificationPreference) error {
	return nil
}

func (m *simpleMockThreadRepo) DeleteNotificationPreference(ctx context.Context, threadID, userID uuid.UUID) error {
	return nil
}

func (m *simpleMockThreadRepo) SetPresence(ctx context.Context, threadID, userID uuid.UUID) error {
	return nil
}

func (m *simpleMockThreadRepo) RemovePresence(ctx context.Context, threadID, userID uuid.UUID) error {
	return nil
}

func (m *simpleMockThreadRepo) GetActiveViewers(ctx context.Context, threadID uuid.UUID) ([]models.ThreadPresenceUser, error) {
	return nil, nil
}

func (m *simpleMockThreadRepo) UpdatePresenceHeartbeat(ctx context.Context, threadID, userID uuid.UUID) error {
	return nil
}

// simpleMockChannelRepo is a simple mock for ChannelRepository
type simpleMockChannelRepo struct {
	channel *models.Channel
}

func (m *simpleMockChannelRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
	return m.channel, nil
}

func (m *simpleMockChannelRepo) GetByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.Channel, error) {
	return nil, nil
}

func (m *simpleMockChannelRepo) Create(ctx context.Context, channel *models.Channel) error {
	return nil
}

func (m *simpleMockChannelRepo) Update(ctx context.Context, channel *models.Channel) error {
	return nil
}

func (m *simpleMockChannelRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

// simpleMockServerRepo is a simple mock for ServerRepository
type simpleMockServerRepo struct {
	server *models.Server
}

func (m *simpleMockServerRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Server, error) {
	return m.server, nil
}

func (m *simpleMockServerRepo) GetMember(ctx context.Context, serverID, userID uuid.UUID) (*models.ServerMember, error) {
	return nil, nil
}

// simpleMockEventBus is a simple mock for EventBus
type simpleMockEventBus struct{}

func (m *simpleMockEventBus) Publish(event string, data interface{}) {}
func (m *simpleMockEventBus) Subscribe(event string, handler func(data interface{})) {}
func (m *simpleMockEventBus) Unsubscribe(event string, handler func(data interface{})) {}

// simpleMockPermService is a simple mock for PermissionService
type simpleMockPermService struct{}

func (m *simpleMockPermService) HasPermission(ctx context.Context, serverID, userID uuid.UUID, permission int64) (bool, error) {
	return true, nil
}

func (m *simpleMockPermService) GetMemberPermissions(ctx context.Context, serverID, userID uuid.UUID) (int64, error) {
	return 0, nil
}

func TestThreadAutoArchiveService_GetOrCreateServerSettings(t *testing.T) {
	ctx := context.Background()
	mockRepo := &simpleMockThreadAutoArchiveRepo{}
	mockThreadRepo := &simpleMockThreadRepo{}
	mockChannelRepo := &simpleMockChannelRepo{}
	mockServerRepo := &simpleMockServerRepo{}
	mockEventBus := &simpleMockEventBus{}
	mockPermService := &simpleMockPermService{}

	service := NewThreadAutoArchiveService(mockRepo, mockThreadRepo, mockChannelRepo, mockServerRepo, mockPermService, mockEventBus)

	serverID := uuid.New()

	// Test getting existing settings
	existingSettings := &models.ThreadAutoArchiveSettings{
		ID:              uuid.New(),
		ServerID:        serverID,
		DefaultDuration: 1440,
		AllowOverride:   true,
	}
	mockRepo.serverSettings = existingSettings

	settings, err := service.GetOrCreateServerSettings(ctx, serverID)
	assert.NoError(t, err)
	assert.Equal(t, existingSettings, settings)
}

func TestThreadAutoArchiveService_GetOrCreateServerSettings_CreatesDefault(t *testing.T) {
	ctx := context.Background()
	mockRepo := &simpleMockThreadAutoArchiveRepo{}
	mockThreadRepo := &simpleMockThreadRepo{}
	mockChannelRepo := &simpleMockChannelRepo{}
	mockServerRepo := &simpleMockServerRepo{}
	mockEventBus := &simpleMockEventBus{}
	mockPermService := &simpleMockPermService{}

	service := NewThreadAutoArchiveService(mockRepo, mockThreadRepo, mockChannelRepo, mockServerRepo, mockPermService, mockEventBus)

	serverID := uuid.New()
	// mockRepo.serverSettings is nil, so it should create defaults

	settings, err := service.GetOrCreateServerSettings(ctx, serverID)
	assert.NoError(t, err)
	assert.NotNil(t, settings)
	assert.Equal(t, serverID, settings.ServerID)
	assert.Equal(t, 1440, settings.DefaultDuration)
	assert.True(t, settings.AllowOverride)
}

func TestThreadAutoArchiveService_GetThreadAutoArchiveStatus(t *testing.T) {
	ctx := context.Background()
	mockRepo := &simpleMockThreadAutoArchiveRepo{}
	mockThreadRepo := &simpleMockThreadRepo{}
	mockChannelRepo := &simpleMockChannelRepo{}
	mockServerRepo := &simpleMockServerRepo{}
	mockEventBus := &simpleMockEventBus{}
	mockPermService := &simpleMockPermService{}

	service := NewThreadAutoArchiveService(mockRepo, mockThreadRepo, mockChannelRepo, mockServerRepo, mockPermService, mockEventBus)

	threadID := uuid.New()
	nextArchive := time.Now().Add(1 * time.Hour)

	thread := &models.Thread{ID: threadID, Archived: false}
	meta := &models.ThreadAutoArchiveMeta{
		ThreadID:        threadID,
		NextArchiveAt:   &nextArchive,
		ArchiveEligible: true,
	}

	mockThreadRepo.thread = thread
	mockRepo.threadMeta = meta

	status, err := service.GetThreadAutoArchiveStatus(ctx, threadID)
	assert.NoError(t, err)
	assert.Equal(t, threadID, status.ThreadID)
	assert.Equal(t, "scheduled", status.Status)
}

func TestThreadAutoArchiveService_GetThreadAutoArchiveStatus_Archived(t *testing.T) {
	ctx := context.Background()
	mockRepo := &simpleMockThreadAutoArchiveRepo{}
	mockThreadRepo := &simpleMockThreadRepo{}
	mockChannelRepo := &simpleMockChannelRepo{}
	mockServerRepo := &simpleMockServerRepo{}
	mockEventBus := &simpleMockEventBus{}
	mockPermService := &simpleMockPermService{}

	service := NewThreadAutoArchiveService(mockRepo, mockThreadRepo, mockChannelRepo, mockServerRepo, mockPermService, mockEventBus)

	threadID := uuid.New()
	nextArchive := time.Now().Add(-1 * time.Hour)

	thread := &models.Thread{ID: threadID, Archived: true}
	meta := &models.ThreadAutoArchiveMeta{
		ThreadID:        threadID,
		NextArchiveAt:   &nextArchive,
		ArchiveEligible: true,
	}

	mockThreadRepo.thread = thread
	mockRepo.threadMeta = meta

	status, err := service.GetThreadAutoArchiveStatus(ctx, threadID)
	assert.NoError(t, err)
	assert.Equal(t, threadID, status.ThreadID)
	assert.Equal(t, "archived", status.Status)
}

func TestThreadAutoArchiveService_ArchiveThread(t *testing.T) {
	ctx := context.Background()
	mockRepo := &simpleMockThreadAutoArchiveRepo{}
	mockThreadRepo := &simpleMockThreadRepo{}
	mockChannelRepo := &simpleMockChannelRepo{}
	mockServerRepo := &simpleMockServerRepo{}
	mockEventBus := &simpleMockEventBus{}
	mockPermService := &simpleMockPermService{}

	service := NewThreadAutoArchiveService(mockRepo, mockThreadRepo, mockChannelRepo, mockServerRepo, mockPermService, mockEventBus)

	threadID := uuid.New()
	channelID := uuid.New()

	thread := &models.Thread{ID: threadID, ParentChannelID: channelID, Archived: false}
	meta := &models.ThreadAutoArchiveMeta{
		ThreadID:        threadID,
		ArchiveEligible: true,
	}

	mockThreadRepo.thread = thread
	mockRepo.threadMeta = meta

	err := service.ArchiveThread(ctx, threadID)
	assert.NoError(t, err)
	assert.True(t, mockThreadRepo.thread.Archived)
}

func TestThreadAutoArchiveService_ArchiveThread_AlreadyArchived(t *testing.T) {
	ctx := context.Background()
	mockRepo := &simpleMockThreadAutoArchiveRepo{}
	mockThreadRepo := &simpleMockThreadRepo{}
	mockChannelRepo := &simpleMockChannelRepo{}
	mockServerRepo := &simpleMockServerRepo{}
	mockEventBus := &simpleMockEventBus{}
	mockPermService := &simpleMockPermService{}

	service := NewThreadAutoArchiveService(mockRepo, mockThreadRepo, mockChannelRepo, mockServerRepo, mockPermService, mockEventBus)

	threadID := uuid.New()

	thread := &models.Thread{ID: threadID, Archived: true}
	mockThreadRepo.thread = thread

	err := service.ArchiveThread(ctx, threadID)
	assert.NoError(t, err)
	// Should still be archived (no change)
	assert.True(t, mockThreadRepo.thread.Archived)
}

func TestIsValidDuration(t *testing.T) {
	tests := []struct {
		duration int
		expected bool
	}{
		{60, true},
		{1440, true},
		{4320, true},
		{10080, true},
		{0, false},
		{100, false},
		{1441, false},
		{-60, false},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := isValidDuration(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}
