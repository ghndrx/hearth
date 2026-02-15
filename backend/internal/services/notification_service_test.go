package services

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"hearth/internal/models"
)

// MockNotificationRepository mocks the NotificationRepository
type MockNotificationRepository struct {
	mock.Mock
}

func (m *MockNotificationRepository) Create(ctx context.Context, notification *models.Notification) error {
	args := m.Called(ctx, notification)
	return args.Error(0)
}

func (m *MockNotificationRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Notification, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Notification), args.Error(1)
}

func (m *MockNotificationRepository) GetByIDWithActor(ctx context.Context, id uuid.UUID) (*models.NotificationWithActor, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.NotificationWithActor), args.Error(1)
}

func (m *MockNotificationRepository) List(ctx context.Context, userID uuid.UUID, opts models.NotificationListOptions) ([]models.NotificationWithActor, error) {
	args := m.Called(ctx, userID, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.NotificationWithActor), args.Error(1)
}

func (m *MockNotificationRepository) GetStats(ctx context.Context, userID uuid.UUID) (*models.NotificationStats, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.NotificationStats), args.Error(1)
}

func (m *MockNotificationRepository) MarkAsRead(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockNotificationRepository) MarkAllAsRead(ctx context.Context, userID uuid.UUID) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockNotificationRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockNotificationRepository) DeleteAllRead(ctx context.Context, userID uuid.UUID) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockNotificationRepository) DeleteOlderThan(ctx context.Context, userID uuid.UUID, before time.Time) (int64, error) {
	args := m.Called(ctx, userID, before)
	return args.Get(0).(int64), args.Error(1)
}

// testNotificationService creates a test notification service with mocks
type testNotificationService struct {
	service  *NotificationService
	repo     *MockNotificationRepository
	eventBus *MockEventBus
}

func newTestNotificationService() *testNotificationService {
	repo := new(MockNotificationRepository)
	eventBus := new(MockEventBus)
	service := NewNotificationService(repo, eventBus)

	return &testNotificationService{
		service:  service,
		repo:     repo,
		eventBus: eventBus,
	}
}

func TestNotificationService_CreateNotification(t *testing.T) {
	ts := newTestNotificationService()
	ctx := context.Background()

	userID := uuid.New()
	actorID := uuid.New()

	req := &models.CreateNotificationRequest{
		UserID:  userID,
		Type:    models.NotificationTypeMention,
		Title:   "You were mentioned",
		Body:    "Someone mentioned you",
		ActorID: &actorID,
	}

	ts.repo.On("Create", ctx, mock.MatchedBy(func(n *models.Notification) bool {
		return n.UserID == userID && n.Type == models.NotificationTypeMention && n.Title == "You were mentioned"
	})).Return(nil)

	ts.eventBus.On("Publish", "notification.created", mock.Anything).Return()

	notification, err := ts.service.CreateNotification(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, notification)
	assert.Equal(t, userID, notification.UserID)
	assert.Equal(t, models.NotificationTypeMention, notification.Type)
	assert.Equal(t, "You were mentioned", notification.Title)
	assert.Equal(t, "Someone mentioned you", notification.Body)
	assert.Equal(t, &actorID, notification.ActorID)
	assert.False(t, notification.Read)

	ts.repo.AssertExpectations(t)
	ts.eventBus.AssertExpectations(t)
}

func TestNotificationService_CreateNotification_Error(t *testing.T) {
	ts := newTestNotificationService()
	ctx := context.Background()

	req := &models.CreateNotificationRequest{
		UserID: uuid.New(),
		Type:   models.NotificationTypeMention,
		Title:  "Test",
		Body:   "Test body",
	}

	ts.repo.On("Create", ctx, mock.Anything).Return(assert.AnError)

	notification, err := ts.service.CreateNotification(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, notification)

	ts.repo.AssertExpectations(t)
}

func TestNotificationService_GetNotification(t *testing.T) {
	ts := newTestNotificationService()
	ctx := context.Background()

	notificationID := uuid.New()
	userID := uuid.New()
	actorUsername := "testuser"

	notification := &models.NotificationWithActor{
		Notification: models.Notification{
			ID:        notificationID,
			UserID:    userID,
			Type:      models.NotificationTypeMention,
			Title:     "You were mentioned",
			Body:      "Test body",
			Read:      false,
			CreatedAt: time.Now(),
		},
		ActorUsername: &actorUsername,
	}

	ts.repo.On("GetByIDWithActor", ctx, notificationID).Return(notification, nil)

	result, err := ts.service.GetNotification(ctx, notificationID, userID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, notificationID, result.ID)
	assert.Equal(t, "testuser", *result.ActorUsername)

	ts.repo.AssertExpectations(t)
}

func TestNotificationService_GetNotification_NotFound(t *testing.T) {
	ts := newTestNotificationService()
	ctx := context.Background()

	notificationID := uuid.New()
	userID := uuid.New()

	ts.repo.On("GetByIDWithActor", ctx, notificationID).Return(nil, nil)

	result, err := ts.service.GetNotification(ctx, notificationID, userID)

	assert.Error(t, err)
	assert.Equal(t, ErrNotificationNotFound, err)
	assert.Nil(t, result)

	ts.repo.AssertExpectations(t)
}

func TestNotificationService_GetNotification_WrongUser(t *testing.T) {
	ts := newTestNotificationService()
	ctx := context.Background()

	notificationID := uuid.New()
	ownerID := uuid.New()
	otherUserID := uuid.New()

	notification := &models.NotificationWithActor{
		Notification: models.Notification{
			ID:     notificationID,
			UserID: ownerID, // Belongs to ownerID
		},
	}

	ts.repo.On("GetByIDWithActor", ctx, notificationID).Return(notification, nil)

	result, err := ts.service.GetNotification(ctx, notificationID, otherUserID)

	assert.Error(t, err)
	assert.Equal(t, ErrNotificationNotFound, err)
	assert.Nil(t, result)

	ts.repo.AssertExpectations(t)
}

func TestNotificationService_ListNotifications(t *testing.T) {
	ts := newTestNotificationService()
	ctx := context.Background()

	userID := uuid.New()
	opts := models.NotificationListOptions{Limit: 50}

	notifications := []models.NotificationWithActor{
		{Notification: models.Notification{ID: uuid.New(), UserID: userID}},
		{Notification: models.Notification{ID: uuid.New(), UserID: userID}},
	}

	ts.repo.On("List", ctx, userID, opts).Return(notifications, nil)

	result, err := ts.service.ListNotifications(ctx, userID, opts)

	assert.NoError(t, err)
	assert.Len(t, result, 2)

	ts.repo.AssertExpectations(t)
}

func TestNotificationService_GetNotificationStats(t *testing.T) {
	ts := newTestNotificationService()
	ctx := context.Background()

	userID := uuid.New()
	stats := &models.NotificationStats{Total: 10, Unread: 3}

	ts.repo.On("GetStats", ctx, userID).Return(stats, nil)

	result, err := ts.service.GetNotificationStats(ctx, userID)

	assert.NoError(t, err)
	assert.Equal(t, 10, result.Total)
	assert.Equal(t, 3, result.Unread)

	ts.repo.AssertExpectations(t)
}

func TestNotificationService_MarkAsRead(t *testing.T) {
	ts := newTestNotificationService()
	ctx := context.Background()

	notificationID := uuid.New()
	userID := uuid.New()

	ts.repo.On("MarkAsRead", ctx, notificationID, userID).Return(nil)
	ts.eventBus.On("Publish", "notification.read", mock.MatchedBy(func(e *NotificationReadEvent) bool {
		return e.NotificationID == notificationID && e.UserID == userID
	})).Return()

	err := ts.service.MarkAsRead(ctx, notificationID, userID)

	assert.NoError(t, err)

	ts.repo.AssertExpectations(t)
	ts.eventBus.AssertExpectations(t)
}

func TestNotificationService_MarkAsRead_NotFound(t *testing.T) {
	ts := newTestNotificationService()
	ctx := context.Background()

	notificationID := uuid.New()
	userID := uuid.New()

	ts.repo.On("MarkAsRead", ctx, notificationID, userID).Return(sql.ErrNoRows)

	err := ts.service.MarkAsRead(ctx, notificationID, userID)

	assert.Error(t, err)
	assert.Equal(t, ErrNotificationNotFound, err)

	ts.repo.AssertExpectations(t)
}

func TestNotificationService_MarkAllAsRead(t *testing.T) {
	ts := newTestNotificationService()
	ctx := context.Background()

	userID := uuid.New()

	ts.repo.On("MarkAllAsRead", ctx, userID).Return(int64(5), nil)
	ts.eventBus.On("Publish", "notification.all_read", mock.MatchedBy(func(e *NotificationAllReadEvent) bool {
		return e.UserID == userID && e.Count == 5
	})).Return()

	count, err := ts.service.MarkAllAsRead(ctx, userID)

	assert.NoError(t, err)
	assert.Equal(t, int64(5), count)

	ts.repo.AssertExpectations(t)
	ts.eventBus.AssertExpectations(t)
}

func TestNotificationService_MarkAllAsRead_ZeroUpdated(t *testing.T) {
	ts := newTestNotificationService()
	ctx := context.Background()

	userID := uuid.New()

	ts.repo.On("MarkAllAsRead", ctx, userID).Return(int64(0), nil)
	// Should not publish event when count is 0

	count, err := ts.service.MarkAllAsRead(ctx, userID)

	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)

	ts.repo.AssertExpectations(t)
	// eventBus.Publish should NOT be called
}

func TestNotificationService_DeleteNotification(t *testing.T) {
	ts := newTestNotificationService()
	ctx := context.Background()

	notificationID := uuid.New()
	userID := uuid.New()

	ts.repo.On("Delete", ctx, notificationID, userID).Return(nil)
	ts.eventBus.On("Publish", "notification.deleted", mock.MatchedBy(func(e *NotificationDeletedEvent) bool {
		return e.NotificationID == notificationID && e.UserID == userID
	})).Return()

	err := ts.service.DeleteNotification(ctx, notificationID, userID)

	assert.NoError(t, err)

	ts.repo.AssertExpectations(t)
	ts.eventBus.AssertExpectations(t)
}

func TestNotificationService_DeleteNotification_NotFound(t *testing.T) {
	ts := newTestNotificationService()
	ctx := context.Background()

	notificationID := uuid.New()
	userID := uuid.New()

	ts.repo.On("Delete", ctx, notificationID, userID).Return(sql.ErrNoRows)

	err := ts.service.DeleteNotification(ctx, notificationID, userID)

	assert.Error(t, err)
	assert.Equal(t, ErrNotificationNotFound, err)

	ts.repo.AssertExpectations(t)
}

func TestNotificationService_DeleteAllReadNotifications(t *testing.T) {
	ts := newTestNotificationService()
	ctx := context.Background()

	userID := uuid.New()

	ts.repo.On("DeleteAllRead", ctx, userID).Return(int64(3), nil)
	ts.eventBus.On("Publish", "notification.read_deleted", mock.MatchedBy(func(e *NotificationReadDeletedEvent) bool {
		return e.UserID == userID && e.Count == 3
	})).Return()

	count, err := ts.service.DeleteAllReadNotifications(ctx, userID)

	assert.NoError(t, err)
	assert.Equal(t, int64(3), count)

	ts.repo.AssertExpectations(t)
	ts.eventBus.AssertExpectations(t)
}

func TestNotificationService_DeleteAllReadNotifications_ZeroDeleted(t *testing.T) {
	ts := newTestNotificationService()
	ctx := context.Background()

	userID := uuid.New()

	ts.repo.On("DeleteAllRead", ctx, userID).Return(int64(0), nil)
	// Should not publish event when count is 0

	count, err := ts.service.DeleteAllReadNotifications(ctx, userID)

	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)

	ts.repo.AssertExpectations(t)
}

func TestNotificationService_CreateNotification_AllTypes(t *testing.T) {
	types := []models.NotificationType{
		models.NotificationTypeMention,
		models.NotificationTypeReply,
		models.NotificationTypeDirectMessage,
		models.NotificationTypeFriendRequest,
		models.NotificationTypeFriendAccept,
		models.NotificationTypeServerInvite,
		models.NotificationTypeServerJoin,
		models.NotificationTypeReaction,
		models.NotificationTypeSystem,
	}

	for _, notifType := range types {
		t.Run(string(notifType), func(t *testing.T) {
			ts := newTestNotificationService()
			ctx := context.Background()

			req := &models.CreateNotificationRequest{
				UserID: uuid.New(),
				Type:   notifType,
				Title:  "Test notification",
				Body:   "Test body",
			}

			ts.repo.On("Create", ctx, mock.MatchedBy(func(n *models.Notification) bool {
				return n.Type == notifType
			})).Return(nil)
			ts.eventBus.On("Publish", "notification.created", mock.Anything).Return()

			notification, err := ts.service.CreateNotification(ctx, req)

			assert.NoError(t, err)
			assert.NotNil(t, notification)
			assert.Equal(t, notifType, notification.Type)

			ts.repo.AssertExpectations(t)
		})
	}
}

func TestNotificationService_CreateNotification_WithReferences(t *testing.T) {
	ts := newTestNotificationService()
	ctx := context.Background()

	userID := uuid.New()
	actorID := uuid.New()
	serverID := uuid.New()
	channelID := uuid.New()
	messageID := uuid.New()

	req := &models.CreateNotificationRequest{
		UserID:    userID,
		Type:      models.NotificationTypeMention,
		Title:     "Mention in server",
		Body:      "You were mentioned",
		ActorID:   &actorID,
		ServerID:  &serverID,
		ChannelID: &channelID,
		MessageID: &messageID,
	}

	ts.repo.On("Create", ctx, mock.MatchedBy(func(n *models.Notification) bool {
		return n.ActorID != nil && *n.ActorID == actorID &&
			n.ServerID != nil && *n.ServerID == serverID &&
			n.ChannelID != nil && *n.ChannelID == channelID &&
			n.MessageID != nil && *n.MessageID == messageID
	})).Return(nil)
	ts.eventBus.On("Publish", "notification.created", mock.Anything).Return()

	notification, err := ts.service.CreateNotification(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, notification)
	assert.Equal(t, &actorID, notification.ActorID)
	assert.Equal(t, &serverID, notification.ServerID)
	assert.Equal(t, &channelID, notification.ChannelID)
	assert.Equal(t, &messageID, notification.MessageID)

	ts.repo.AssertExpectations(t)
}
