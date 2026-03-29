package postgres

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hearth/internal/models"
)

func setupNotificationRepoMock(t *testing.T) (*NotificationRepository, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	sqlxDB := sqlx.NewDb(db, "postgres")
	return NewNotificationRepository(sqlxDB), mock
}

// --- NotificationRepository Tests ---

func TestNotificationRepository_Create(t *testing.T) {
	repo, mock := setupNotificationRepoMock(t)
	ctx := context.Background()

	notification := &models.Notification{
		UserID:  uuid.New(),
		Type:    models.NotificationTypeMention,
		Title:   "New Mention",
		Body:    "You were mentioned in a message",
		Read:    false,
		Data:    nil,
		ActorID: ptrUUID(uuid.New()),
	}

	mock.ExpectExec("INSERT INTO notifications").
		WithArgs(
			sqlmock.AnyArg(), // id - generated
			notification.UserID,
			notification.Type,
			notification.Title,
			notification.Body,
			sqlmock.AnyArg(), // read - set to false
			notification.Data,
			notification.ActorID,
			sqlmock.AnyArg(), // server_id - nil
			sqlmock.AnyArg(), // channel_id - nil
			sqlmock.AnyArg(), // message_id - nil
			sqlmock.AnyArg(), // created_at - set to now
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(ctx, notification)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, notification.ID)
	assert.NotZero(t, notification.CreatedAt)
	assert.False(t, notification.Read)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNotificationRepository_Create_Error(t *testing.T) {
	repo, mock := setupNotificationRepoMock(t)
	ctx := context.Background()

	notification := &models.Notification{
		UserID: uuid.New(),
		Type:   models.NotificationTypeMention,
		Title:  "Test",
		Body:   "Test body",
	}

	mock.ExpectExec("INSERT INTO notifications").
		WillReturnError(sql.ErrConnDone)

	err := repo.Create(ctx, notification)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNotificationRepository_GetByID(t *testing.T) {
	repo, mock := setupNotificationRepoMock(t)
	ctx := context.Background()

	notificationID := uuid.New()
	userID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "type", "title", "body", "read", "data",
		"actor_id", "server_id", "channel_id", "message_id", "created_at",
	}).AddRow(
		notificationID, userID, "mention", "Test Title", "Test Body",
		false, nil, nil, nil, nil, nil, now,
	)

	mock.ExpectQuery("SELECT .+ FROM notifications").
		WithArgs(notificationID).
		WillReturnRows(rows)

	notification, err := repo.GetByID(ctx, notificationID)
	assert.NoError(t, err)
	assert.NotNil(t, notification)
	assert.Equal(t, notificationID, notification.ID)
	assert.Equal(t, userID, notification.UserID)
	assert.Equal(t, "mention", string(notification.Type))
	assert.Equal(t, "Test Title", notification.Title)
	assert.Equal(t, "Test Body", notification.Body)
	assert.False(t, notification.Read)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNotificationRepository_GetByID_NotFound(t *testing.T) {
	repo, mock := setupNotificationRepoMock(t)
	ctx := context.Background()

	notificationID := uuid.New()

	mock.ExpectQuery("SELECT .+ FROM notifications").
		WithArgs(notificationID).
		WillReturnError(sql.ErrNoRows)

	notification, err := repo.GetByID(ctx, notificationID)
	assert.NoError(t, err)
	assert.Nil(t, notification)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNotificationRepository_GetByID_Error(t *testing.T) {
	repo, mock := setupNotificationRepoMock(t)
	ctx := context.Background()

	notificationID := uuid.New()

	mock.ExpectQuery("SELECT .+ FROM notifications").
		WithArgs(notificationID).
		WillReturnError(sql.ErrConnDone)

	notification, err := repo.GetByID(ctx, notificationID)
	// The implementation returns empty struct along with error
	assert.Error(t, err)
	assert.NotNil(t, notification) // returns empty struct, not nil
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNotificationRepository_GetByIDWithActor(t *testing.T) {
	repo, mock := setupNotificationRepoMock(t)
	ctx := context.Background()

	notificationID := uuid.New()
	userID := uuid.New()
	actorID := uuid.New()
	serverID := uuid.New()
	channelID := uuid.New()
	now := time.Now()
	actorUsername := "testuser"
	actorAvatar := "https://example.com/avatar.png"
	serverName := "Test Server"
	channelName := "general"

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "type", "title", "body", "read", "data",
		"actor_id", "server_id", "channel_id", "message_id", "created_at",
		"actor_username", "actor_avatar", "server_name", "channel_name",
	}).AddRow(
		notificationID, userID, "mention", "Test Title", "Test Body",
		false, nil, actorID, serverID, channelID, nil, now,
		actorUsername, actorAvatar, serverName, channelName,
	)

	mock.ExpectQuery("SELECT .+ FROM notifications n").
		WithArgs(notificationID).
		WillReturnRows(rows)

	notification, err := repo.GetByIDWithActor(ctx, notificationID)
	assert.NoError(t, err)
	assert.NotNil(t, notification)
	assert.Equal(t, notificationID, notification.ID)
	assert.NotNil(t, notification.ActorUsername)
	assert.Equal(t, actorUsername, *notification.ActorUsername)
	assert.NotNil(t, notification.ServerName)
	assert.Equal(t, serverName, *notification.ServerName)
	assert.NotNil(t, notification.ChannelName)
	assert.Equal(t, channelName, *notification.ChannelName)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNotificationRepository_GetByIDWithActor_NotFound(t *testing.T) {
	repo, mock := setupNotificationRepoMock(t)
	ctx := context.Background()

	notificationID := uuid.New()

	mock.ExpectQuery("SELECT .+ FROM notifications n").
		WithArgs(notificationID).
		WillReturnError(sql.ErrNoRows)

	notification, err := repo.GetByIDWithActor(ctx, notificationID)
	assert.NoError(t, err)
	assert.Nil(t, notification)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNotificationRepository_List(t *testing.T) {
	repo, mock := setupNotificationRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()
	opts := models.NotificationListOptions{
		Limit:  10,
		Offset: 0,
	}

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "type", "title", "body", "read", "data",
		"actor_id", "server_id", "channel_id", "message_id", "created_at",
		"actor_username", "actor_avatar", "server_name", "channel_name",
	}).AddRow(
		uuid.New(), userID, "mention", "Title1", "Body1",
		false, nil, nil, nil, nil, nil, time.Now(),
		nil, nil, nil, nil,
	).AddRow(
		uuid.New(), userID, "reply", "Title2", "Body2",
		true, nil, nil, nil, nil, nil, time.Now(),
		nil, nil, nil, nil,
	)

	mock.ExpectQuery("SELECT .+ FROM notifications n").
		WithArgs(userID, opts.Limit, opts.Offset).
		WillReturnRows(rows)

	notifications, err := repo.List(ctx, userID, opts)
	assert.NoError(t, err)
	assert.Len(t, notifications, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNotificationRepository_List_WithUnreadFilter(t *testing.T) {
	repo, mock := setupNotificationRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()
	unread := true
	opts := models.NotificationListOptions{
		Limit:  10,
		Offset: 0,
		Unread: &unread,
	}

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "type", "title", "body", "read", "data",
		"actor_id", "server_id", "channel_id", "message_id", "created_at",
		"actor_username", "actor_avatar", "server_name", "channel_name",
	}).AddRow(
		uuid.New(), userID, "mention", "Title1", "Body1",
		false, nil, nil, nil, nil, nil, time.Now(),
		nil, nil, nil, nil,
	)

	mock.ExpectQuery("SELECT .+ FROM notifications n").
		WithArgs(userID, false, opts.Limit, opts.Offset).
		WillReturnRows(rows)

	notifications, err := repo.List(ctx, userID, opts)
	assert.NoError(t, err)
	assert.Len(t, notifications, 1)
	assert.False(t, notifications[0].Read)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNotificationRepository_List_WithTypeFilter(t *testing.T) {
	repo, mock := setupNotificationRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()
	opts := models.NotificationListOptions{
		Limit:  10,
		Offset: 0,
		Types:  []models.NotificationType{models.NotificationTypeMention, models.NotificationTypeReply},
	}

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "type", "title", "body", "read", "data",
		"actor_id", "server_id", "channel_id", "message_id", "created_at",
		"actor_username", "actor_avatar", "server_name", "channel_name",
	})

	mock.ExpectQuery("SELECT .+ FROM notifications n").
		WithArgs(userID, "mention", "reply", opts.Limit, opts.Offset).
		WillReturnRows(rows)

	notifications, err := repo.List(ctx, userID, opts)
	assert.NoError(t, err)
	assert.Len(t, notifications, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNotificationRepository_List_Empty(t *testing.T) {
	repo, mock := setupNotificationRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()
	opts := models.NotificationListOptions{
		Limit:  10,
		Offset: 0,
	}

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "type", "title", "body", "read", "data",
		"actor_id", "server_id", "channel_id", "message_id", "created_at",
		"actor_username", "actor_avatar", "server_name", "channel_name",
	})

	mock.ExpectQuery("SELECT .+ FROM notifications n").
		WithArgs(userID, opts.Limit, opts.Offset).
		WillReturnRows(rows)

	notifications, err := repo.List(ctx, userID, opts)
	assert.NoError(t, err)
	assert.Len(t, notifications, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNotificationRepository_List_Error(t *testing.T) {
	repo, mock := setupNotificationRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()
	opts := models.NotificationListOptions{
		Limit:  10,
		Offset: 0,
	}

	mock.ExpectQuery("SELECT .+ FROM notifications n").
		WithArgs(userID, opts.Limit, opts.Offset).
		WillReturnError(sql.ErrConnDone)

	notifications, err := repo.List(ctx, userID, opts)
	assert.Error(t, err)
	assert.Nil(t, notifications)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNotificationRepository_List_DefaultLimit(t *testing.T) {
	repo, mock := setupNotificationRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()
	opts := models.NotificationListOptions{
		Limit:  0, // Should default to 50
		Offset: 0,
	}

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "type", "title", "body", "read", "data",
		"actor_id", "server_id", "channel_id", "message_id", "created_at",
		"actor_username", "actor_avatar", "server_name", "channel_name",
	})

	mock.ExpectQuery("SELECT .+ FROM notifications n").
		WithArgs(userID, 50, 0). // Should be capped to 50
		WillReturnRows(rows)

	notifications, err := repo.List(ctx, userID, opts)
	assert.NoError(t, err)
	assert.Len(t, notifications, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNotificationRepository_List_ExcessiveLimit(t *testing.T) {
	repo, mock := setupNotificationRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()
	opts := models.NotificationListOptions{
		Limit:  500, // Should default to 50
		Offset: 0,
	}

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "type", "title", "body", "read", "data",
		"actor_id", "server_id", "channel_id", "message_id", "created_at",
		"actor_username", "actor_avatar", "server_name", "channel_name",
	})

	mock.ExpectQuery("SELECT .+ FROM notifications n").
		WithArgs(userID, 50, 0). // Should be capped to 50
		WillReturnRows(rows)

	notifications, err := repo.List(ctx, userID, opts)
	assert.NoError(t, err)
	assert.Len(t, notifications, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNotificationRepository_GetStats(t *testing.T) {
	repo, mock := setupNotificationRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()

	rows := sqlmock.NewRows([]string{"total", "unread"}).
		AddRow(100, 25)

	mock.ExpectQuery("SELECT .+ FROM notifications").
		WithArgs(userID).
		WillReturnRows(rows)

	stats, err := repo.GetStats(ctx, userID)
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, 100, stats.Total)
	assert.Equal(t, 25, stats.Unread)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNotificationRepository_GetStats_Error(t *testing.T) {
	repo, mock := setupNotificationRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()

	mock.ExpectQuery("SELECT .+ FROM notifications").
		WithArgs(userID).
		WillReturnError(sql.ErrConnDone)

	stats, err := repo.GetStats(ctx, userID)
	// The implementation returns empty struct along with error
	assert.Error(t, err)
	assert.NotNil(t, stats) // returns empty struct, not nil
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNotificationRepository_MarkAsRead(t *testing.T) {
	repo, mock := setupNotificationRepoMock(t)
	ctx := context.Background()

	notificationID := uuid.New()
	userID := uuid.New()

	mock.ExpectExec("UPDATE notifications SET read = true").
		WithArgs(notificationID, userID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.MarkAsRead(ctx, notificationID, userID)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNotificationRepository_MarkAsRead_NotFound(t *testing.T) {
	repo, mock := setupNotificationRepoMock(t)
	ctx := context.Background()

	notificationID := uuid.New()
	userID := uuid.New()

	mock.ExpectExec("UPDATE notifications SET read = true").
		WithArgs(notificationID, userID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.MarkAsRead(ctx, notificationID, userID)
	assert.Error(t, err)
	assert.Equal(t, sql.ErrNoRows, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNotificationRepository_MarkAsRead_Error(t *testing.T) {
	repo, mock := setupNotificationRepoMock(t)
	ctx := context.Background()

	notificationID := uuid.New()
	userID := uuid.New()

	mock.ExpectExec("UPDATE notifications SET read = true").
		WithArgs(notificationID, userID).
		WillReturnError(sql.ErrConnDone)

	err := repo.MarkAsRead(ctx, notificationID, userID)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNotificationRepository_MarkAllAsRead(t *testing.T) {
	repo, mock := setupNotificationRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()

	mock.ExpectExec("UPDATE notifications SET read = true").
		WithArgs(userID).
		WillReturnResult(sqlmock.NewResult(0, 5))

	count, err := repo.MarkAllAsRead(ctx, userID)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNotificationRepository_MarkAllAsRead_Error(t *testing.T) {
	repo, mock := setupNotificationRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()

	mock.ExpectExec("UPDATE notifications SET read = true").
		WithArgs(userID).
		WillReturnError(sql.ErrConnDone)

	count, err := repo.MarkAllAsRead(ctx, userID)
	assert.Error(t, err)
	assert.Equal(t, int64(0), count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNotificationRepository_Delete(t *testing.T) {
	repo, mock := setupNotificationRepoMock(t)
	ctx := context.Background()

	notificationID := uuid.New()
	userID := uuid.New()

	mock.ExpectExec("DELETE FROM notifications").
		WithArgs(notificationID, userID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Delete(ctx, notificationID, userID)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNotificationRepository_Delete_NotFound(t *testing.T) {
	repo, mock := setupNotificationRepoMock(t)
	ctx := context.Background()

	notificationID := uuid.New()
	userID := uuid.New()

	mock.ExpectExec("DELETE FROM notifications").
		WithArgs(notificationID, userID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.Delete(ctx, notificationID, userID)
	assert.Error(t, err)
	assert.Equal(t, sql.ErrNoRows, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNotificationRepository_Delete_Error(t *testing.T) {
	repo, mock := setupNotificationRepoMock(t)
	ctx := context.Background()

	notificationID := uuid.New()
	userID := uuid.New()

	mock.ExpectExec("DELETE FROM notifications").
		WithArgs(notificationID, userID).
		WillReturnError(sql.ErrConnDone)

	err := repo.Delete(ctx, notificationID, userID)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNotificationRepository_DeleteAllRead(t *testing.T) {
	repo, mock := setupNotificationRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()

	mock.ExpectExec("DELETE FROM notifications").
		WithArgs(userID).
		WillReturnResult(sqlmock.NewResult(0, 10))

	count, err := repo.DeleteAllRead(ctx, userID)
	assert.NoError(t, err)
	assert.Equal(t, int64(10), count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNotificationRepository_DeleteAllRead_Error(t *testing.T) {
	repo, mock := setupNotificationRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()

	mock.ExpectExec("DELETE FROM notifications").
		WithArgs(userID).
		WillReturnError(sql.ErrConnDone)

	count, err := repo.DeleteAllRead(ctx, userID)
	assert.Error(t, err)
	assert.Equal(t, int64(0), count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNotificationRepository_DeleteOlderThan(t *testing.T) {
	repo, mock := setupNotificationRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()
	before := time.Now().Add(-24 * time.Hour)

	mock.ExpectExec("DELETE FROM notifications").
		WithArgs(userID, before).
		WillReturnResult(sqlmock.NewResult(0, 3))

	count, err := repo.DeleteOlderThan(ctx, userID, before)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNotificationRepository_DeleteOlderThan_Error(t *testing.T) {
	repo, mock := setupNotificationRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()
	before := time.Now().Add(-24 * time.Hour)

	mock.ExpectExec("DELETE FROM notifications").
		WithArgs(userID, before).
		WillReturnError(sql.ErrConnDone)

	count, err := repo.DeleteOlderThan(ctx, userID, before)
	assert.Error(t, err)
	assert.Equal(t, int64(0), count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// --- Helper functions ---

func ptrUUID(u uuid.UUID) *uuid.UUID {
	return &u
}
