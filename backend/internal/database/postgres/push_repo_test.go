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

func setupPushRepoMock(t *testing.T) (*PushRepository, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	sqlxDB := sqlx.NewDb(db, "postgres")
	return NewPushRepository(sqlxDB), mock
}

func setupNotifPrefsRepoMock(t *testing.T) (*NotificationPreferencesRepository, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	sqlxDB := sqlx.NewDb(db, "postgres")
	return NewNotificationPreferencesRepository(sqlxDB), mock
}

// --- PushRepository Tests ---

func TestPushRepository_CreateSubscription(t *testing.T) {
	repo, mock := setupPushRepoMock(t)
	ctx := context.Background()

	sub := &models.PushSubscription{
		UserID:    uuid.New(),
		Endpoint:  "https://push.example.com/abc123",
		P256dh:    "BNcRdreALRFXTkOOUHK1EtK2wtaz5Ry4YfYCA_0QTpQtUbVlUls0VJXg7A8u-Ts1XbjxAkDm-yhPZ3XU2MyEAMI8",
		Auth:      "tBHItJI5svbICP7XYCA-xA",
		UserAgent: "Mozilla/5.0",
		ExpiresAt: nil,
	}

	mock.ExpectExec("INSERT INTO push_subscriptions").
		WithArgs(sqlmock.AnyArg(), sub.UserID, sub.Endpoint, sub.P256dh, sub.Auth, sub.UserAgent, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.CreateSubscription(ctx, sub)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPushRepository_CreateSubscription_Error(t *testing.T) {
	repo, mock := setupPushRepoMock(t)
	ctx := context.Background()

	sub := &models.PushSubscription{
		UserID:    uuid.New(),
		Endpoint:  "https://push.example.com/abc",
		P256dh:    "key",
		Auth:      "auth",
		UserAgent: "agent",
	}

	mock.ExpectExec("INSERT INTO push_subscriptions").
		WillReturnError(sql.ErrConnDone)

	err := repo.CreateSubscription(ctx, sub)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPushRepository_GetSubscriptionByEndpoint(t *testing.T) {
	repo, mock := setupPushRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()
	endpoint := "https://push.example.com/abc"

	rows := sqlmock.NewRows([]string{"id", "user_id", "endpoint", "p256dh", "auth", "user_agent", "created_at", "expires_at"}).
		AddRow(uuid.New(), userID, endpoint, "key", "auth", "agent", time.Now(), nil)

	mock.ExpectQuery("SELECT \\* FROM push_subscriptions").
		WithArgs(userID, endpoint).
		WillReturnRows(rows)

	sub, err := repo.GetSubscriptionByEndpoint(ctx, userID, endpoint)
	assert.NoError(t, err)
	assert.NotNil(t, sub)
	assert.Equal(t, endpoint, sub.Endpoint)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPushRepository_GetSubscriptionByEndpoint_NotFound(t *testing.T) {
	repo, mock := setupPushRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()
	endpoint := "https://push.example.com/nonexistent"

	mock.ExpectQuery("SELECT \\* FROM push_subscriptions").
		WithArgs(userID, endpoint).
		WillReturnError(sql.ErrNoRows)

	sub, err := repo.GetSubscriptionByEndpoint(ctx, userID, endpoint)
	assert.NoError(t, err)
	assert.Nil(t, sub)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPushRepository_GetSubscriptionByEndpoint_Error(t *testing.T) {
	repo, mock := setupPushRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()
	endpoint := "https://push.example.com/abc"

	mock.ExpectQuery("SELECT \\* FROM push_subscriptions").
		WithArgs(userID, endpoint).
		WillReturnError(sql.ErrConnDone)

	sub, err := repo.GetSubscriptionByEndpoint(ctx, userID, endpoint)
	// The implementation returns empty struct along with error
	assert.Error(t, err)
	assert.NotNil(t, sub) // returns empty struct, not nil
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPushRepository_GetUserSubscriptions(t *testing.T) {
	repo, mock := setupPushRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "user_id", "endpoint", "p256dh", "auth", "user_agent", "created_at", "expires_at"}).
		AddRow(uuid.New(), userID, "https://push1.example.com", "key1", "auth1", "agent1", now, nil).
		AddRow(uuid.New(), userID, "https://push2.example.com", "key2", "auth2", "agent2", now, &now)

	mock.ExpectQuery("SELECT \\* FROM push_subscriptions").
		WithArgs(userID).
		WillReturnRows(rows)

	subs, err := repo.GetUserSubscriptions(ctx, userID)
	assert.NoError(t, err)
	assert.Len(t, subs, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPushRepository_GetUserSubscriptions_Empty(t *testing.T) {
	repo, mock := setupPushRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()

	rows := sqlmock.NewRows([]string{"id", "user_id", "endpoint", "p256dh", "auth", "user_agent", "created_at", "expires_at"})

	mock.ExpectQuery("SELECT \\* FROM push_subscriptions").
		WithArgs(userID).
		WillReturnRows(rows)

	subs, err := repo.GetUserSubscriptions(ctx, userID)
	assert.NoError(t, err)
	assert.Len(t, subs, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPushRepository_GetUserSubscriptions_Error(t *testing.T) {
	repo, mock := setupPushRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()

	mock.ExpectQuery("SELECT \\* FROM push_subscriptions").
		WithArgs(userID).
		WillReturnError(sql.ErrConnDone)

	subs, err := repo.GetUserSubscriptions(ctx, userID)
	assert.Error(t, err)
	assert.Nil(t, subs)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPushRepository_DeleteSubscription(t *testing.T) {
	repo, mock := setupPushRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()
	endpoint := "https://push.example.com/abc"

	mock.ExpectExec("DELETE FROM push_subscriptions").
		WithArgs(userID, endpoint).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.DeleteSubscription(ctx, userID, endpoint)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPushRepository_DeleteSubscription_Error(t *testing.T) {
	repo, mock := setupPushRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()
	endpoint := "https://push.example.com/abc"

	mock.ExpectExec("DELETE FROM push_subscriptions").
		WillReturnError(sql.ErrConnDone)

	err := repo.DeleteSubscription(ctx, userID, endpoint)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPushRepository_DeleteExpiredSubscriptions(t *testing.T) {
	repo, mock := setupPushRepoMock(t)
	ctx := context.Background()

	mock.ExpectExec("DELETE FROM push_subscriptions").
		WithArgs().
		WillReturnResult(sqlmock.NewResult(0, 5))

	count, err := repo.DeleteExpiredSubscriptions(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPushRepository_DeleteExpiredSubscriptions_Error(t *testing.T) {
	repo, mock := setupPushRepoMock(t)
	ctx := context.Background()

	mock.ExpectExec("DELETE FROM push_subscriptions").
		WillReturnError(sql.ErrConnDone)

	count, err := repo.DeleteExpiredSubscriptions(ctx)
	assert.Error(t, err)
	assert.Equal(t, int64(0), count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPushRepository_DeleteUserSubscriptions(t *testing.T) {
	repo, mock := setupPushRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()

	mock.ExpectExec("DELETE FROM push_subscriptions").
		WithArgs(userID).
		WillReturnResult(sqlmock.NewResult(0, 3))

	err := repo.DeleteUserSubscriptions(ctx, userID)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPushRepository_DeleteUserSubscriptions_Error(t *testing.T) {
	repo, mock := setupPushRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()

	mock.ExpectExec("DELETE FROM push_subscriptions").
		WillReturnError(sql.ErrConnDone)

	err := repo.DeleteUserSubscriptions(ctx, userID)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// --- NotificationPreferencesRepository Tests ---

func TestNotificationPreferencesRepository_GetPreferences(t *testing.T) {
	repo, mock := setupNotifPrefsRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()
	now := time.Now().Truncate(time.Second)

	rows := sqlmock.NewRows([]string{
		"user_id", "push_enabled", "push_mentions", "push_direct_messages", "push_replies",
		"push_friend_requests", "push_server_invites", "sound_enabled", "sound_message",
		"sound_mention", "desktop_enabled", "desktop_previews", "do_not_disturb",
		"do_not_disturb_until", "updated_at",
	}).AddRow(
		userID, true, true, false, true,
		false, true, true, "chime.wav",
		"mention.wav", true, true, false,
		nil, now,
	)

	mock.ExpectQuery("SELECT \\* FROM notification_preferences").
		WithArgs(userID).
		WillReturnRows(rows)

	prefs, err := repo.GetPreferences(ctx, userID)
	assert.NoError(t, err)
	assert.NotNil(t, prefs)
	assert.Equal(t, userID, prefs.UserID)
	assert.True(t, prefs.PushEnabled)
	assert.True(t, prefs.PushMentions)
	assert.False(t, prefs.PushDirectMessages)
	assert.Equal(t, "chime.wav", prefs.SoundMessage)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNotificationPreferencesRepository_GetPreferences_Defaults(t *testing.T) {
	repo, mock := setupNotifPrefsRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()

	mock.ExpectQuery("SELECT \\* FROM notification_preferences").
		WithArgs(userID).
		WillReturnError(sql.ErrNoRows)

	prefs, err := repo.GetPreferences(ctx, userID)
	assert.NoError(t, err)
	assert.NotNil(t, prefs)
	assert.Equal(t, userID, prefs.UserID)
	assert.True(t, prefs.PushEnabled) // default
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNotificationPreferencesRepository_GetPreferences_Error(t *testing.T) {
	repo, mock := setupNotifPrefsRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()

	mock.ExpectQuery("SELECT \\* FROM notification_preferences").
		WithArgs(userID).
		WillReturnError(sql.ErrConnDone)

	prefs, err := repo.GetPreferences(ctx, userID)
	// The implementation returns empty struct along with error
	assert.Error(t, err)
	assert.NotNil(t, prefs) // returns empty struct, not nil
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNotificationPreferencesRepository_UpsertPreferences(t *testing.T) {
	repo, mock := setupNotifPrefsRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()
	prefs := &models.NotificationPreferences{
		UserID:             userID,
		PushEnabled:        true,
		PushMentions:       true,
		PushDirectMessages: false,
		PushReplies:        true,
		PushFriendRequests: false,
		PushServerInvites:  true,
		SoundEnabled:       true,
		SoundMessage:       "ping.wav",
		SoundMention:       "ding.wav",
		DesktopEnabled:     true,
		DesktopPreviews:    false,
		DoNotDisturb:       false,
		UpdatedAt:          time.Now(),
	}

	mock.ExpectExec("INSERT INTO notification_preferences").
		WithArgs(
			prefs.UserID, prefs.PushEnabled, prefs.PushMentions, prefs.PushDirectMessages,
			prefs.PushReplies, prefs.PushFriendRequests, prefs.PushServerInvites,
			prefs.SoundEnabled, prefs.SoundMessage, prefs.SoundMention,
			prefs.DesktopEnabled, prefs.DesktopPreviews, prefs.DoNotDisturb,
			sqlmock.AnyArg(), sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.UpsertPreferences(ctx, prefs)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNotificationPreferencesRepository_UpsertPreferences_Error(t *testing.T) {
	repo, mock := setupNotifPrefsRepoMock(t)
	ctx := context.Background()

	prefs := &models.NotificationPreferences{
		UserID:             uuid.New(),
		PushEnabled:        true,
		PushMentions:       false,
		PushDirectMessages: false,
		PushReplies:        false,
		PushFriendRequests: false,
		PushServerInvites:  false,
		SoundEnabled:       false,
		SoundMessage:       "",
		SoundMention:       "",
		DesktopEnabled:     false,
		DesktopPreviews:    false,
		DoNotDisturb:       false,
		UpdatedAt:          time.Now(),
	}

	mock.ExpectExec("INSERT INTO notification_preferences").
		WillReturnError(sql.ErrConnDone)

	err := repo.UpsertPreferences(ctx, prefs)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
