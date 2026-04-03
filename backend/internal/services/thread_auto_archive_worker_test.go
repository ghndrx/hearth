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
type simpleMockThreadAutoArchiveRepoForWorker struct {
	threadMeta []*models.ThreadAutoArchiveMeta
}

func (m *simpleMockThreadAutoArchiveRepoForWorker) CreateServerSettings(ctx context.Context, settings *models.ThreadAutoArchiveSettings) error {
	return nil
}

func (m *simpleMockThreadAutoArchiveRepoForWorker) GetServerSettings(ctx context.Context, serverID uuid.UUID) (*models.ThreadAutoArchiveSettings, error) {
	return nil, nil
}

func (m *simpleMockThreadAutoArchiveRepoForWorker) UpdateServerSettings(ctx context.Context, settings *models.ThreadAutoArchiveSettings) error {
	return nil
}

func (m *simpleMockThreadAutoArchiveRepoForWorker) DeleteServerSettings(ctx context.Context, serverID uuid.UUID) error {
	return nil
}

func (m *simpleMockThreadAutoArchiveRepoForWorker) SetChannelOverride(ctx context.Context, override *models.ChannelAutoArchiveOverride) error {
	return nil
}

func (m *simpleMockThreadAutoArchiveRepoForWorker) GetChannelOverride(ctx context.Context, channelID uuid.UUID) (*models.ChannelAutoArchiveOverride, error) {
	return nil, nil
}

func (m *simpleMockThreadAutoArchiveRepoForWorker) DeleteChannelOverride(ctx context.Context, channelID uuid.UUID) error {
	return nil
}

func (m *simpleMockThreadAutoArchiveRepoForWorker) GetOrCreateThreadMeta(ctx context.Context, threadID uuid.UUID) (*models.ThreadAutoArchiveMeta, error) {
	for _, meta := range m.threadMeta {
		if meta.ThreadID == threadID {
			return meta, nil
		}
	}
	return nil, nil
}

func (m *simpleMockThreadAutoArchiveRepoForWorker) UpdateThreadMeta(ctx context.Context, meta *models.ThreadAutoArchiveMeta) error {
	return nil
}

func (m *simpleMockThreadAutoArchiveRepoForWorker) SetThreadNextArchive(ctx context.Context, threadID uuid.UUID, nextArchiveAt *time.Time) error {
	return nil
}

func (m *simpleMockThreadAutoArchiveRepoForWorker) SetThreadArchiveEligible(ctx context.Context, threadID uuid.UUID, eligible bool) error {
	return nil
}

func (m *simpleMockThreadAutoArchiveRepoForWorker) BumpThreadOwnerActivity(ctx context.Context, threadID uuid.UUID) error {
	return nil
}

func (m *simpleMockThreadAutoArchiveRepoForWorker) GetThreadsReadyForArchive(ctx context.Context, limit int) ([]*models.ThreadAutoArchiveMeta, error) {
	return m.threadMeta, nil
}

func (m *simpleMockThreadAutoArchiveRepoForWorker) GetThreadMeta(ctx context.Context, threadID uuid.UUID) (*models.ThreadAutoArchiveMeta, error) {
	return m.GetOrCreateThreadMeta(ctx, threadID)
}

func (m *simpleMockThreadAutoArchiveRepoForWorker) DeleteThreadMeta(ctx context.Context, threadID uuid.UUID) error {
	return nil
}

func (m *simpleMockThreadAutoArchiveRepoForWorker) GetServerStats(ctx context.Context, serverID uuid.UUID) (*models.ThreadAutoArchiveStats, error) {
	return nil, nil
}

func (m *simpleMockThreadAutoArchiveRepoForWorker) GetChannelDuration(ctx context.Context, channelID, serverID uuid.UUID) (int, error) {
	return 1440, nil
}

// simpleMockThreadRepoForWorker is a simple mock for ThreadRepository
type simpleMockThreadRepoForWorker struct {
	threads map[uuid.UUID]*models.Thread
}

func newSimpleMockThreadRepoForWorker() *simpleMockThreadRepoForWorker {
	return &simpleMockThreadRepoForWorker{
		threads: make(map[uuid.UUID]*models.Thread),
	}
}

func (m *simpleMockThreadRepoForWorker) Create(ctx context.Context, thread *models.Thread) error {
	m.threads[thread.ID] = thread
	return nil
}

func (m *simpleMockThreadRepoForWorker) GetByID(ctx context.Context, id uuid.UUID) (*models.Thread, error) {
	return m.threads[id], nil
}

func (m *simpleMockThreadRepoForWorker) GetByParentMessageID(ctx context.Context, messageID uuid.UUID) (*models.Thread, error) {
	return nil, nil
}

func (m *simpleMockThreadRepoForWorker) Update(ctx context.Context, thread *models.Thread) error {
	m.threads[thread.ID] = thread
	return nil
}

func (m *simpleMockThreadRepoForWorker) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.threads, id)
	return nil
}

func (m *simpleMockThreadRepoForWorker) GetByChannelID(ctx context.Context, channelID uuid.UUID) ([]*models.Thread, error) {
	var result []*models.Thread
	for _, thread := range m.threads {
		if thread.ParentChannelID == channelID {
			result = append(result, thread)
		}
	}
	return result, nil
}

func (m *simpleMockThreadRepoForWorker) GetActiveByChannelID(ctx context.Context, channelID uuid.UUID) ([]*models.Thread, error) {
	return m.GetByChannelID(ctx, channelID)
}

func (m *simpleMockThreadRepoForWorker) Archive(ctx context.Context, id uuid.UUID) error {
	if t, ok := m.threads[id]; ok {
		t.Archived = true
	}
	return nil
}

func (m *simpleMockThreadRepoForWorker) Unarchive(ctx context.Context, id uuid.UUID) error {
	if t, ok := m.threads[id]; ok {
		t.Archived = false
	}
	return nil
}

func (m *simpleMockThreadRepoForWorker) AddMember(ctx context.Context, threadID, userID uuid.UUID) error {
	return nil
}

func (m *simpleMockThreadRepoForWorker) RemoveMember(ctx context.Context, threadID, userID uuid.UUID) error {
	return nil
}

func (m *simpleMockThreadRepoForWorker) IsMember(ctx context.Context, threadID, userID uuid.UUID) (bool, error) {
	return true, nil
}

func (m *simpleMockThreadRepoForWorker) GetMembers(ctx context.Context, threadID uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}

func (m *simpleMockThreadRepoForWorker) CreateMessage(ctx context.Context, threadID, authorID uuid.UUID, content string) (*models.ThreadMessage, error) {
	return nil, nil
}

func (m *simpleMockThreadRepoForWorker) GetMessages(ctx context.Context, threadID uuid.UUID, before *uuid.UUID, limit int) ([]*models.ThreadMessage, error) {
	return nil, nil
}

func (m *simpleMockThreadRepoForWorker) IncrementMessageCount(ctx context.Context, threadID uuid.UUID) error {
	return nil
}

func (m *simpleMockThreadRepoForWorker) GetNotificationPreference(ctx context.Context, threadID, userID uuid.UUID) (*models.ThreadNotificationPreference, error) {
	return nil, nil
}

func (m *simpleMockThreadRepoForWorker) SetNotificationPreference(ctx context.Context, pref *models.ThreadNotificationPreference) error {
	return nil
}

func (m *simpleMockThreadRepoForWorker) DeleteNotificationPreference(ctx context.Context, threadID, userID uuid.UUID) error {
	return nil
}

func (m *simpleMockThreadRepoForWorker) SetPresence(ctx context.Context, threadID, userID uuid.UUID) error {
	return nil
}

func (m *simpleMockThreadRepoForWorker) RemovePresence(ctx context.Context, threadID, userID uuid.UUID) error {
	return nil
}

func (m *simpleMockThreadRepoForWorker) GetActiveViewers(ctx context.Context, threadID uuid.UUID) ([]models.ThreadPresenceUser, error) {
	return nil, nil
}

func (m *simpleMockThreadRepoForWorker) UpdatePresenceHeartbeat(ctx context.Context, threadID, userID uuid.UUID) error {
	return nil
}

// simpleMockChannelRepoForWorker is a simple mock for ChannelRepository
type simpleMockChannelRepoForWorker struct {
	channels map[uuid.UUID]*models.Channel
}

func newSimpleMockChannelRepoForWorker() *simpleMockChannelRepoForWorker {
	return &simpleMockChannelRepoForWorker{
		channels: make(map[uuid.UUID]*models.Channel),
	}
}

func (m *simpleMockChannelRepoForWorker) GetByID(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
	return m.channels[id], nil
}

func (m *simpleMockChannelRepoForWorker) GetByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.Channel, error) {
	return nil, nil
}

func (m *simpleMockChannelRepoForWorker) Create(ctx context.Context, channel *models.Channel) error {
	m.channels[channel.ID] = channel
	return nil
}

func (m *simpleMockChannelRepoForWorker) Update(ctx context.Context, channel *models.Channel) error {
	m.channels[channel.ID] = channel
	return nil
}

func (m *simpleMockChannelRepoForWorker) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.channels, id)
	return nil
}

func (m *simpleMockChannelRepoForWorker) GetDMChannel(ctx context.Context, user1ID, user2ID uuid.UUID) (*models.Channel, error) {
	return nil, nil
}
func (m *simpleMockChannelRepoForWorker) GetUserDMs(ctx context.Context, userID uuid.UUID) ([]*models.Channel, error) {
	return nil, nil
}
func (m *simpleMockChannelRepoForWorker) UpdateLastMessage(ctx context.Context, channelID, messageID uuid.UUID, at time.Time) error {
	return nil
}
func (m *simpleMockChannelRepoForWorker) AddRecipient(ctx context.Context, channelID, userID uuid.UUID) error {
	return nil
}
func (m *simpleMockChannelRepoForWorker) RemoveRecipient(ctx context.Context, channelID, userID uuid.UUID) error {
	return nil
}
func (m *simpleMockChannelRepoForWorker) CountRecipients(ctx context.Context, channelID uuid.UUID) (int, error) {
	return 0, nil
}
func (m *simpleMockChannelRepoForWorker) BulkUpdatePositions(ctx context.Context, entries []models.ReorderChannelEntry) error {
	return nil
}
func (m *simpleMockChannelRepoForWorker) GetPermissionOverrides(ctx context.Context, channelID uuid.UUID) ([]models.PermissionOverride, error) {
	return nil, nil
}
func (m *simpleMockChannelRepoForWorker) UpsertPermissionOverride(ctx context.Context, override *models.PermissionOverride) error {
	return nil
}
func (m *simpleMockChannelRepoForWorker) DeletePermissionOverride(ctx context.Context, channelID, targetID uuid.UUID, targetType string) error {
	return nil
}

// simpleMockEventBusForWorker is a simple mock for EventBus
type simpleMockEventBusForWorker struct{}

func (m *simpleMockEventBusForWorker) Publish(event string, data interface{})                   {}
func (m *simpleMockEventBusForWorker) Subscribe(event string, handler func(data interface{}))   {}
func (m *simpleMockEventBusForWorker) Unsubscribe(event string, handler func(data interface{})) {}

func TestThreadAutoArchiveWorker_ProcessThreadActivity(t *testing.T) {
	ctx := context.Background()
	mockRepo := &simpleMockThreadAutoArchiveRepoForWorker{}
	mockThreadRepo := newSimpleMockThreadRepoForWorker()
	mockChannelRepo := newSimpleMockChannelRepoForWorker()
	mockEventBus := &simpleMockEventBusForWorker{}

	worker := NewThreadAutoArchiveWorker(mockRepo, mockThreadRepo, mockChannelRepo, mockEventBus)

	threadID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()
	userID := uuid.New()

	thread := &models.Thread{ID: threadID, ParentChannelID: channelID, OwnerID: userID, Archived: false}
	channel := &models.Channel{ID: channelID, ServerID: &serverID}
	meta := &models.ThreadAutoArchiveMeta{
		ThreadID:       threadID,
		LastActivityAt: time.Now().Add(-1 * time.Hour),
	}

	mockThreadRepo.threads[threadID] = thread
	mockChannelRepo.channels[channelID] = channel
	mockRepo.threadMeta = []*models.ThreadAutoArchiveMeta{meta}

	err := worker.ProcessThreadActivity(ctx, threadID, userID)
	assert.NoError(t, err)
}

func TestThreadAutoArchiveWorker_ProcessThreadActivity_OwnerBump(t *testing.T) {
	ctx := context.Background()
	mockRepo := &simpleMockThreadAutoArchiveRepoForWorker{}
	mockThreadRepo := newSimpleMockThreadRepoForWorker()
	mockChannelRepo := newSimpleMockChannelRepoForWorker()
	mockEventBus := &simpleMockEventBusForWorker{}

	worker := NewThreadAutoArchiveWorker(mockRepo, mockThreadRepo, mockChannelRepo, mockEventBus)

	threadID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()
	ownerID := uuid.New()

	thread := &models.Thread{ID: threadID, ParentChannelID: channelID, OwnerID: ownerID, Archived: false}
	channel := &models.Channel{ID: channelID, ServerID: &serverID}
	meta := &models.ThreadAutoArchiveMeta{
		ThreadID:      threadID,
		BumpedByOwner: false,
	}

	mockThreadRepo.threads[threadID] = thread
	mockChannelRepo.channels[channelID] = channel
	mockRepo.threadMeta = []*models.ThreadAutoArchiveMeta{meta}

	err := worker.ProcessThreadActivity(ctx, threadID, ownerID)
	assert.NoError(t, err)
}

func TestThreadAutoArchiveWorker_ProcessThreadActivity_SkipsArchived(t *testing.T) {
	ctx := context.Background()
	mockRepo := &simpleMockThreadAutoArchiveRepoForWorker{}
	mockThreadRepo := newSimpleMockThreadRepoForWorker()
	mockChannelRepo := newSimpleMockChannelRepoForWorker()
	mockEventBus := &simpleMockEventBusForWorker{}

	worker := NewThreadAutoArchiveWorker(mockRepo, mockThreadRepo, mockChannelRepo, mockEventBus)

	threadID := uuid.New()
	channelID := uuid.New()
	userID := uuid.New()

	thread := &models.Thread{ID: threadID, ParentChannelID: channelID, OwnerID: userID, Archived: true}

	mockThreadRepo.threads[threadID] = thread

	err := worker.ProcessThreadActivity(ctx, threadID, userID)
	assert.NoError(t, err)
}

func TestThreadAutoArchiveWorker_GetWorkerStatus(t *testing.T) {
	mockRepo := &simpleMockThreadAutoArchiveRepoForWorker{}
	mockThreadRepo := newSimpleMockThreadRepoForWorker()
	mockChannelRepo := newSimpleMockChannelRepoForWorker()
	mockEventBus := &simpleMockEventBusForWorker{}

	worker := NewThreadAutoArchiveWorker(mockRepo, mockThreadRepo, mockChannelRepo, mockEventBus)

	status := worker.GetWorkerStatus()
	assert.Equal(t, false, status["is_running"])
	assert.Equal(t, 50, status["batch_size"])
	assert.Equal(t, "1m0s", status["check_interval"])
}

func TestThreadAutoArchiveWorker_SetBatchSize(t *testing.T) {
	mockRepo := &simpleMockThreadAutoArchiveRepoForWorker{}
	mockThreadRepo := newSimpleMockThreadRepoForWorker()
	mockChannelRepo := newSimpleMockChannelRepoForWorker()
	mockEventBus := &simpleMockEventBusForWorker{}

	worker := NewThreadAutoArchiveWorker(mockRepo, mockThreadRepo, mockChannelRepo, mockEventBus)

	worker.SetBatchSize(100)
	assert.Equal(t, 100, worker.batchSize)

	// Test invalid sizes
	worker.SetBatchSize(0)
	assert.Equal(t, 100, worker.batchSize) // Should not change

	worker.SetBatchSize(-1)
	assert.Equal(t, 100, worker.batchSize) // Should not change

	worker.SetBatchSize(501)
	assert.Equal(t, 100, worker.batchSize) // Should not change (max is 500)
}

func TestThreadAutoArchiveWorker_SetCheckInterval(t *testing.T) {
	mockRepo := &simpleMockThreadAutoArchiveRepoForWorker{}
	mockThreadRepo := newSimpleMockThreadRepoForWorker()
	mockChannelRepo := newSimpleMockChannelRepoForWorker()
	mockEventBus := &simpleMockEventBusForWorker{}

	worker := NewThreadAutoArchiveWorker(mockRepo, mockThreadRepo, mockChannelRepo, mockEventBus)

	worker.SetCheckInterval(15 * time.Second)
	assert.Equal(t, 15*time.Second, worker.checkInterval)
}
