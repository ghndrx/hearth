package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"hearth/internal/models"
)

// NotificationRepository handles notification data access
type NotificationRepository struct {
	db *sqlx.DB
}

// NewNotificationRepository creates a new notification repository
func NewNotificationRepository(db *sqlx.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

// Create creates a new notification
func (r *NotificationRepository) Create(ctx context.Context, notification *models.Notification) error {
	notification.ID = uuid.New()
	notification.CreatedAt = time.Now()
	notification.Read = false

	query := `
		INSERT INTO notifications (
			id, user_id, type, title, body, read, data,
			actor_id, server_id, channel_id, message_id, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)
	`
	_, err := r.db.ExecContext(ctx, query,
		notification.ID, notification.UserID, notification.Type, notification.Title,
		notification.Body, notification.Read, notification.Data,
		notification.ActorID, notification.ServerID, notification.ChannelID,
		notification.MessageID, notification.CreatedAt,
	)
	return err
}

// GetByID retrieves a notification by ID
func (r *NotificationRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Notification, error) {
	var notification models.Notification
	query := `
		SELECT id, user_id, type, title, body, read, data,
		       actor_id, server_id, channel_id, message_id, created_at
		FROM notifications
		WHERE id = $1
	`
	err := r.db.GetContext(ctx, &notification, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &notification, err
}

// GetByIDWithActor retrieves a notification with actor info
func (r *NotificationRepository) GetByIDWithActor(ctx context.Context, id uuid.UUID) (*models.NotificationWithActor, error) {
	var notification models.NotificationWithActor
	query := `
		SELECT n.id, n.user_id, n.type, n.title, n.body, n.read, n.data,
		       n.actor_id, n.server_id, n.channel_id, n.message_id, n.created_at,
		       u.username as actor_username, u.avatar_url as actor_avatar,
		       s.name as server_name, c.name as channel_name
		FROM notifications n
		LEFT JOIN users u ON n.actor_id = u.id
		LEFT JOIN servers s ON n.server_id = s.id
		LEFT JOIN channels c ON n.channel_id = c.id
		WHERE n.id = $1
	`
	err := r.db.GetContext(ctx, &notification, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &notification, err
}

// List retrieves notifications for a user with options
func (r *NotificationRepository) List(ctx context.Context, userID uuid.UUID, opts models.NotificationListOptions) ([]models.NotificationWithActor, error) {
	// Set defaults
	if opts.Limit <= 0 || opts.Limit > 100 {
		opts.Limit = 50
	}

	var conditions []string
	var args []interface{}
	argNum := 1

	conditions = append(conditions, fmt.Sprintf("n.user_id = $%d", argNum))
	args = append(args, userID)
	argNum++

	if opts.Unread != nil {
		conditions = append(conditions, fmt.Sprintf("n.read = $%d", argNum))
		args = append(args, !*opts.Unread)
		argNum++
	}

	if len(opts.Types) > 0 {
		placeholders := make([]string, len(opts.Types))
		for i, t := range opts.Types {
			placeholders[i] = fmt.Sprintf("$%d", argNum)
			args = append(args, t)
			argNum++
		}
		conditions = append(conditions, fmt.Sprintf("n.type IN (%s)", strings.Join(placeholders, ",")))
	}

	whereClause := strings.Join(conditions, " AND ")

	query := fmt.Sprintf(`
		SELECT n.id, n.user_id, n.type, n.title, n.body, n.read, n.data,
		       n.actor_id, n.server_id, n.channel_id, n.message_id, n.created_at,
		       u.username as actor_username, u.avatar_url as actor_avatar,
		       s.name as server_name, c.name as channel_name
		FROM notifications n
		LEFT JOIN users u ON n.actor_id = u.id
		LEFT JOIN servers s ON n.server_id = s.id
		LEFT JOIN channels c ON n.channel_id = c.id
		WHERE %s
		ORDER BY n.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)

	args = append(args, opts.Limit, opts.Offset)

	var notifications []models.NotificationWithActor
	err := r.db.SelectContext(ctx, &notifications, query, args...)
	if err != nil {
		return nil, err
	}

	if notifications == nil {
		notifications = []models.NotificationWithActor{}
	}
	return notifications, nil
}

// GetStats retrieves notification statistics for a user
func (r *NotificationRepository) GetStats(ctx context.Context, userID uuid.UUID) (*models.NotificationStats, error) {
	var stats models.NotificationStats
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE read = false) as unread
		FROM notifications
		WHERE user_id = $1
	`
	err := r.db.GetContext(ctx, &stats, query, userID)
	return &stats, err
}

// MarkAsRead marks a single notification as read
func (r *NotificationRepository) MarkAsRead(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	query := `UPDATE notifications SET read = true WHERE id = $1 AND user_id = $2`
	result, err := r.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// MarkAllAsRead marks all notifications as read for a user
func (r *NotificationRepository) MarkAllAsRead(ctx context.Context, userID uuid.UUID) (int64, error) {
	query := `UPDATE notifications SET read = true WHERE user_id = $1 AND read = false`
	result, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// Delete deletes a notification
func (r *NotificationRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	query := `DELETE FROM notifications WHERE id = $1 AND user_id = $2`
	result, err := r.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// DeleteAllRead deletes all read notifications for a user
func (r *NotificationRepository) DeleteAllRead(ctx context.Context, userID uuid.UUID) (int64, error) {
	query := `DELETE FROM notifications WHERE user_id = $1 AND read = true`
	result, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// DeleteOlderThan deletes notifications older than the given duration
func (r *NotificationRepository) DeleteOlderThan(ctx context.Context, userID uuid.UUID, before time.Time) (int64, error) {
	query := `DELETE FROM notifications WHERE user_id = $1 AND created_at < $2`
	result, err := r.db.ExecContext(ctx, query, userID, before)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

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

// MentionRepository handles mention database operations
type MentionRepository struct {
	db *sqlx.DB
}

// NewMentionRepository creates a new mention repository
func NewMentionRepository(db *sqlx.DB) *MentionRepository {
	return &MentionRepository{db: db}
}

// Create creates a new mention record
func (r *MentionRepository) Create(ctx context.Context, mention *models.Mention) error {
	if mention.ID == uuid.Nil {
		mention.ID = uuid.New()
	}
	if mention.CreatedAt.IsZero() {
		mention.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO mentions (
			id, user_id, message_id, mentioned_by, channel_id, guild_id, 
			mention_type, mentioned_role_id, mentioned_channel_id, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT DO NOTHING
	`
	_, err := r.db.ExecContext(ctx, query,
		mention.ID,
		mention.UserID,
		mention.MessageID,
		mention.MentionedBy,
		mention.ChannelID,
		mention.GuildID,
		mention.MentionType,
		mention.MentionedRoleID,
		mention.MentionedChannelID,
		mention.CreatedAt,
	)
	return err
}

// CreateBatch creates multiple mention records efficiently
func (r *MentionRepository) CreateBatch(ctx context.Context, mentions []*models.Mention) error {
	if len(mentions) == 0 {
		return nil
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO mentions (
			id, user_id, message_id, mentioned_by, channel_id, guild_id, 
			mention_type, mentioned_role_id, mentioned_channel_id, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT DO NOTHING
	`

	for _, m := range mentions {
		if m.ID == uuid.Nil {
			m.ID = uuid.New()
		}
		if m.CreatedAt.IsZero() {
			m.CreatedAt = time.Now()
		}

		_, err := tx.ExecContext(ctx, query,
			m.ID,
			m.UserID,
			m.MessageID,
			m.MentionedBy,
			m.ChannelID,
			m.GuildID,
			m.MentionType,
			m.MentionedRoleID,
			m.MentionedChannelID,
			m.CreatedAt,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetByID retrieves a mention by ID
func (r *MentionRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Mention, error) {
	var mention models.Mention
	query := `
		SELECT id, user_id, message_id, mentioned_by, channel_id, guild_id,
			   mention_type, mentioned_role_id, mentioned_channel_id, read_at, created_at
		FROM mentions
		WHERE id = $1
	`
	err := r.db.GetContext(ctx, &mention, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &mention, nil
}

// GetMentionsWithContext retrieves mentions with full context for API responses
func (r *MentionRepository) GetMentionsWithContext(ctx context.Context, filter *models.MentionFilter) ([]models.MentionWithContext, int, error) {
	filter.SetDefaults()

	// Build the base query
	baseQuery := `
		SELECT 
			m.id, m.user_id, m.message_id, m.mentioned_by, m.channel_id, m.guild_id,
			m.mention_type, m.mentioned_role_id, m.mentioned_channel_id, m.read_at, m.created_at,
			s.name as server_name,
			c.name as channel_name,
			u.username as author_name,
			u.avatar as author_avatar,
			COALESCE(SUBSTRING(msg.content, 1, 200), '') as preview
		FROM mentions m
		LEFT JOIN servers s ON m.guild_id = s.id
		LEFT JOIN channels c ON m.channel_id = c.id
		LEFT JOIN users u ON m.mentioned_by = u.id
		LEFT JOIN messages msg ON m.message_id = msg.id
		WHERE m.user_id = $1
	`

	countQuery := `SELECT COUNT(*) FROM mentions m WHERE m.user_id = $1`

	var conditions []string
	args := []interface{}{filter.UserID}
	argIndex := 2

	if filter.Unread != nil {
		if *filter.Unread {
			conditions = append(conditions, "m.read_at IS NULL")
		} else {
			conditions = append(conditions, "m.read_at IS NOT NULL")
		}
	}

	if filter.MentionType != nil {
		conditions = append(conditions, fmt.Sprintf("m.mention_type = $%d", argIndex))
		args = append(args, *filter.MentionType)
		argIndex++
	}

	if filter.ChannelID != nil {
		conditions = append(conditions, fmt.Sprintf("m.channel_id = $%d", argIndex))
		args = append(args, *filter.ChannelID)
		argIndex++
	}

	if filter.GuildID != nil {
		conditions = append(conditions, fmt.Sprintf("m.guild_id = $%d", argIndex))
		args = append(args, *filter.GuildID)
		argIndex++
	}

	if filter.Before != nil {
		conditions = append(conditions, fmt.Sprintf("m.created_at < $%d", argIndex))
		args = append(args, *filter.Before)
		argIndex++
	}

	if filter.After != nil {
		conditions = append(conditions, fmt.Sprintf("m.created_at > $%d", argIndex))
		args = append(args, *filter.After)
		argIndex++
	}

	// Build full query
	if len(conditions) > 0 {
		condStr := " AND " + strings.Join(conditions, " AND ")
		baseQuery += condStr
		countQuery += condStr
	}

	baseQuery += " ORDER BY m.created_at DESC"
	baseQuery += fmt.Sprintf(" LIMIT %d OFFSET %d", filter.Limit+1, filter.Offset) // +1 to check for more

	// Execute queries
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	var mentions []models.MentionWithContext
	if err := r.db.SelectContext(ctx, &mentions, baseQuery, args...); err != nil {
		return nil, 0, err
	}

	return mentions, total, nil
}

// GetUnreadCount returns the count of unread mentions for a user
func (r *MentionRepository) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM mentions WHERE user_id = $1 AND read_at IS NULL`
	err := r.db.GetContext(ctx, &count, query, userID)
	return count, err
}

// GetStats returns mention statistics for a user
func (r *MentionRepository) GetStats(ctx context.Context, userID uuid.UUID) (*models.MentionStats, error) {
	stats := &models.MentionStats{}

	// Total count
	err := r.db.GetContext(ctx, &stats.TotalCount,
		`SELECT COUNT(*) FROM mentions WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}

	// Unread count
	err = r.db.GetContext(ctx, &stats.UnreadCount,
		`SELECT COUNT(*) FROM mentions WHERE user_id = $1 AND read_at IS NULL`, userID)
	if err != nil {
		return nil, err
	}

	// Today count
	err = r.db.GetContext(ctx, &stats.TodayCount,
		`SELECT COUNT(*) FROM mentions WHERE user_id = $1 AND created_at >= CURRENT_DATE`, userID)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

// MarkAsRead marks a mention as read
func (r *MentionRepository) MarkAsRead(ctx context.Context, id, userID uuid.UUID) error {
	query := `UPDATE mentions SET read_at = $1 WHERE id = $2 AND user_id = $3 AND read_at IS NULL`
	_, err := r.db.ExecContext(ctx, query, time.Now(), id, userID)
	return err
}

// MarkAllAsRead marks all mentions as read for a user
func (r *MentionRepository) MarkAllAsRead(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `UPDATE mentions SET read_at = $1 WHERE user_id = $2 AND read_at IS NULL`
	result, err := r.db.ExecContext(ctx, query, time.Now(), userID)
	if err != nil {
		return 0, err
	}
	affected, _ := result.RowsAffected()
	return int(affected), nil
}

// MarkChannelMentionsAsRead marks all mentions in a channel as read for a user
func (r *MentionRepository) MarkChannelMentionsAsRead(ctx context.Context, userID, channelID uuid.UUID) (int, error) {
	query := `UPDATE mentions SET read_at = $1 WHERE user_id = $2 AND channel_id = $3 AND read_at IS NULL`
	result, err := r.db.ExecContext(ctx, query, time.Now(), userID, channelID)
	if err != nil {
		return 0, err
	}
	affected, _ := result.RowsAffected()
	return int(affected), nil
}

// DeleteByMessage deletes all mentions for a message (when message is deleted)
func (r *MentionRepository) DeleteByMessage(ctx context.Context, messageID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM mentions WHERE message_id = $1`, messageID)
	return err
}

// DeleteByChannel deletes all mentions in a channel (when channel is deleted)
func (r *MentionRepository) DeleteByChannel(ctx context.Context, channelID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM mentions WHERE channel_id = $1`, channelID)
	return err
}

// DeleteByUser deletes all mentions for a user (when user is deleted)
func (r *MentionRepository) DeleteByUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM mentions WHERE user_id = $1 OR mentioned_by = $1`, userID)
	return err
}

// Search searches mentions by content or author
func (r *MentionRepository) Search(ctx context.Context, userID uuid.UUID, query string, limit int) ([]models.MentionWithContext, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	searchQuery := `
		SELECT 
			m.id, m.user_id, m.message_id, m.mentioned_by, m.channel_id, m.guild_id,
			m.mention_type, m.mentioned_role_id, m.mentioned_channel_id, m.read_at, m.created_at,
			s.name as server_name,
			c.name as channel_name,
			u.username as author_name,
			u.avatar as author_avatar,
			COALESCE(SUBSTRING(msg.content, 1, 200), '') as preview
		FROM mentions m
		LEFT JOIN servers s ON m.guild_id = s.id
		LEFT JOIN channels c ON m.channel_id = c.id
		LEFT JOIN users u ON m.mentioned_by = u.id
		LEFT JOIN messages msg ON m.message_id = msg.id
		WHERE m.user_id = $1
		  AND (
			  msg.content ILIKE '%' || $2 || '%'
			  OR u.username ILIKE '%' || $2 || '%'
			  OR c.name ILIKE '%' || $2 || '%'
		  )
		ORDER BY m.created_at DESC
		LIMIT $3
	`

	var mentions []models.MentionWithContext
	err := r.db.SelectContext(ctx, &mentions, searchQuery, userID, query, limit)
	return mentions, err
}
