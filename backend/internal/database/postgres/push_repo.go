package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"hearth/internal/models"
)

// PushRepository handles push subscription data access
type PushRepository struct {
	db *sqlx.DB
}

// NewPushRepository creates a new push repository
func NewPushRepository(db *sqlx.DB) *PushRepository {
	return &PushRepository{db: db}
}

// CreateSubscription creates a new push subscription
func (r *PushRepository) CreateSubscription(ctx context.Context, sub *models.PushSubscription) error {
	sub.ID = uuid.New()
	sub.CreatedAt = time.Now()

	query := `
		INSERT INTO push_subscriptions (id, user_id, endpoint, p256dh, auth, user_agent, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id, endpoint) DO UPDATE SET
			p256dh = $4,
			auth = $5,
			user_agent = $6,
			expires_at = $8
	`
	_, err := r.db.ExecContext(ctx, query,
		sub.ID, sub.UserID, sub.Endpoint, sub.P256dh, sub.Auth, sub.UserAgent, sub.CreatedAt, sub.ExpiresAt,
	)
	return err
}

// GetSubscriptionByEndpoint retrieves a subscription by endpoint
func (r *PushRepository) GetSubscriptionByEndpoint(ctx context.Context, userID uuid.UUID, endpoint string) (*models.PushSubscription, error) {
	var sub models.PushSubscription
	query := `SELECT * FROM push_subscriptions WHERE user_id = $1 AND endpoint = $2`
	err := r.db.GetContext(ctx, &sub, query, userID, endpoint)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &sub, err
}

// GetUserSubscriptions retrieves all push subscriptions for a user
func (r *PushRepository) GetUserSubscriptions(ctx context.Context, userID uuid.UUID) ([]*models.PushSubscription, error) {
	var subs []*models.PushSubscription
	query := `SELECT * FROM push_subscriptions WHERE user_id = $1 AND (expires_at IS NULL OR expires_at > NOW())`
	err := r.db.SelectContext(ctx, &subs, query, userID)
	if err != nil {
		return nil, err
	}
	if subs == nil {
		subs = []*models.PushSubscription{}
	}
	return subs, nil
}

// DeleteSubscription deletes a push subscription
func (r *PushRepository) DeleteSubscription(ctx context.Context, userID uuid.UUID, endpoint string) error {
	query := `DELETE FROM push_subscriptions WHERE user_id = $1 AND endpoint = $2`
	_, err := r.db.ExecContext(ctx, query, userID, endpoint)
	return err
}

// DeleteExpiredSubscriptions deletes all expired subscriptions
func (r *PushRepository) DeleteExpiredSubscriptions(ctx context.Context) (int64, error) {
	query := `DELETE FROM push_subscriptions WHERE expires_at IS NOT NULL AND expires_at < NOW()`
	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// DeleteUserSubscriptions deletes all subscriptions for a user
func (r *PushRepository) DeleteUserSubscriptions(ctx context.Context, userID uuid.UUID) error {
	query := `DELETE FROM push_subscriptions WHERE user_id = $1`
	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}

// NotificationPreferencesRepository handles notification preferences data access
type NotificationPreferencesRepository struct {
	db *sqlx.DB
}

// NewNotificationPreferencesRepository creates a new notification preferences repository
func NewNotificationPreferencesRepository(db *sqlx.DB) *NotificationPreferencesRepository {
	return &NotificationPreferencesRepository{db: db}
}

// GetPreferences retrieves notification preferences for a user
func (r *NotificationPreferencesRepository) GetPreferences(ctx context.Context, userID uuid.UUID) (*models.NotificationPreferences, error) {
	var prefs models.NotificationPreferences
	query := `SELECT * FROM notification_preferences WHERE user_id = $1`
	err := r.db.GetContext(ctx, &prefs, query, userID)
	if err == sql.ErrNoRows {
		// Return defaults
		return models.DefaultNotificationPreferences(userID), nil
	}
	return &prefs, err
}

// UpsertPreferences creates or updates notification preferences
func (r *NotificationPreferencesRepository) UpsertPreferences(ctx context.Context, prefs *models.NotificationPreferences) error {
	prefs.UpdatedAt = time.Now()

	query := `
		INSERT INTO notification_preferences (
			user_id, push_enabled, push_mentions, push_direct_messages, push_replies,
			push_friend_requests, push_server_invites, sound_enabled, sound_message,
			sound_mention, desktop_enabled, desktop_previews, do_not_disturb,
			do_not_disturb_until, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
		)
		ON CONFLICT (user_id) DO UPDATE SET
			push_enabled = $2, push_mentions = $3, push_direct_messages = $4,
			push_replies = $5, push_friend_requests = $6, push_server_invites = $7,
			sound_enabled = $8, sound_message = $9, sound_mention = $10,
			desktop_enabled = $11, desktop_previews = $12, do_not_disturb = $13,
			do_not_disturb_until = $14, updated_at = $15
	`
	_, err := r.db.ExecContext(ctx, query,
		prefs.UserID, prefs.PushEnabled, prefs.PushMentions, prefs.PushDirectMessages,
		prefs.PushReplies, prefs.PushFriendRequests, prefs.PushServerInvites,
		prefs.SoundEnabled, prefs.SoundMessage, prefs.SoundMention,
		prefs.DesktopEnabled, prefs.DesktopPreviews, prefs.DoNotDisturb,
		prefs.DoNotDisturbUntil, prefs.UpdatedAt,
	)
	return err
}
