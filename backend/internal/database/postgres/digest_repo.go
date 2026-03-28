package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"hearth/internal/models"
)

// DigestRepository handles digest data access
type DigestRepository struct {
	db *sqlx.DB
}

// NewDigestRepository creates a new digest repository
func NewDigestRepository(db *sqlx.DB) *DigestRepository {
	return &DigestRepository{db: db}
}

// --- Digest Preferences ---

// GetPreferences retrieves digest preferences for a user
func (r *DigestRepository) GetPreferences(ctx context.Context, userID uuid.UUID) (*models.DigestPreferences, error) {
	var prefs models.DigestPreferences
	query := `
		SELECT id, user_id, enabled, frequency, preferred_hour, preferred_day,
		       aggregation_mode, max_messages_per_source, muted_channels_only,
		       timezone, created_at, updated_at
		FROM digest_preferences
		WHERE user_id = $1
	`
	err := r.db.GetContext(ctx, &prefs, query, userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &prefs, err
}

// CreatePreferences creates digest preferences for a user
func (r *DigestRepository) CreatePreferences(ctx context.Context, prefs *models.DigestPreferences) error {
	prefs.ID = uuid.New()
	prefs.CreatedAt = time.Now()
	prefs.UpdatedAt = time.Now()

	query := `
		INSERT INTO digest_preferences (
			id, user_id, enabled, frequency, preferred_hour, preferred_day,
			aggregation_mode, max_messages_per_source, muted_channels_only,
			timezone, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)
	`
	_, err := r.db.ExecContext(ctx, query,
		prefs.ID, prefs.UserID, prefs.Enabled, prefs.Frequency,
		prefs.PreferredHour, prefs.PreferredDay, prefs.AggregationMode,
		prefs.MaxMessagesPerSource, prefs.MutedChannelsOnly,
		prefs.Timezone, prefs.CreatedAt, prefs.UpdatedAt,
	)
	return err
}

// UpdatePreferences updates digest preferences for a user
func (r *DigestRepository) UpdatePreferences(ctx context.Context, prefs *models.DigestPreferences) error {
	prefs.UpdatedAt = time.Now()

	query := `
		UPDATE digest_preferences SET
			enabled = $2, frequency = $3, preferred_hour = $4,
			preferred_day = $5, aggregation_mode = $6,
			max_messages_per_source = $7, muted_channels_only = $8,
			timezone = $9, updated_at = $10
		WHERE user_id = $1
	`
	result, err := r.db.ExecContext(ctx, query,
		prefs.UserID, prefs.Enabled, prefs.Frequency,
		prefs.PreferredHour, prefs.PreferredDay, prefs.AggregationMode,
		prefs.MaxMessagesPerSource, prefs.MutedChannelsOnly,
		prefs.Timezone, prefs.UpdatedAt,
	)
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

// UpsertPreferences creates or updates digest preferences
func (r *DigestRepository) UpsertPreferences(ctx context.Context, prefs *models.DigestPreferences) error {
	prefs.UpdatedAt = time.Now()
	if prefs.ID == uuid.Nil {
		prefs.ID = uuid.New()
	}
	if prefs.CreatedAt.IsZero() {
		prefs.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO digest_preferences (
			id, user_id, enabled, frequency, preferred_hour, preferred_day,
			aggregation_mode, max_messages_per_source, muted_channels_only,
			timezone, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)
		ON CONFLICT (user_id) DO UPDATE SET
			enabled = EXCLUDED.enabled,
			frequency = EXCLUDED.frequency,
			preferred_hour = EXCLUDED.preferred_hour,
			preferred_day = EXCLUDED.preferred_day,
			aggregation_mode = EXCLUDED.aggregation_mode,
			max_messages_per_source = EXCLUDED.max_messages_per_source,
			muted_channels_only = EXCLUDED.muted_channels_only,
			timezone = EXCLUDED.timezone,
			updated_at = EXCLUDED.updated_at
	`
	_, err := r.db.ExecContext(ctx, query,
		prefs.ID, prefs.UserID, prefs.Enabled, prefs.Frequency,
		prefs.PreferredHour, prefs.PreferredDay, prefs.AggregationMode,
		prefs.MaxMessagesPerSource, prefs.MutedChannelsOnly,
		prefs.Timezone, prefs.CreatedAt, prefs.UpdatedAt,
	)
	return err
}

// --- Channel Preferences ---

// GetChannelPreference retrieves channel-specific digest preference
func (r *DigestRepository) GetChannelPreference(ctx context.Context, userID, channelID uuid.UUID) (*models.DigestChannelPreference, error) {
	var pref models.DigestChannelPreference
	query := `
		SELECT id, user_id, channel_id, digest_mode, created_at, updated_at
		FROM digest_channel_preferences
		WHERE user_id = $1 AND channel_id = $2
	`
	err := r.db.GetContext(ctx, &pref, query, userID, channelID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &pref, err
}

// GetChannelPreferences retrieves all channel-specific digest preferences for a user
func (r *DigestRepository) GetChannelPreferences(ctx context.Context, userID uuid.UUID) ([]models.DigestChannelPreference, error) {
	var prefs []models.DigestChannelPreference
	query := `
		SELECT id, user_id, channel_id, digest_mode, created_at, updated_at
		FROM digest_channel_preferences
		WHERE user_id = $1
	`
	err := r.db.SelectContext(ctx, &prefs, query, userID)
	if err != nil {
		return nil, err
	}
	if prefs == nil {
		prefs = []models.DigestChannelPreference{}
	}
	return prefs, nil
}

// UpsertChannelPreference creates or updates a channel digest preference
func (r *DigestRepository) UpsertChannelPreference(ctx context.Context, pref *models.DigestChannelPreference) error {
	pref.UpdatedAt = time.Now()
	if pref.ID == uuid.Nil {
		pref.ID = uuid.New()
	}
	if pref.CreatedAt.IsZero() {
		pref.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO digest_channel_preferences (
			id, user_id, channel_id, digest_mode, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, channel_id) DO UPDATE SET
			digest_mode = EXCLUDED.digest_mode,
			updated_at = EXCLUDED.updated_at
	`
	_, err := r.db.ExecContext(ctx, query,
		pref.ID, pref.UserID, pref.ChannelID, pref.DigestMode,
		pref.CreatedAt, pref.UpdatedAt,
	)
	return err
}

// DeleteChannelPreference deletes a channel digest preference
func (r *DigestRepository) DeleteChannelPreference(ctx context.Context, userID, channelID uuid.UUID) error {
	query := `DELETE FROM digest_channel_preferences WHERE user_id = $1 AND channel_id = $2`
	_, err := r.db.ExecContext(ctx, query, userID, channelID)
	return err
}

// --- Server Preferences ---

// GetServerPreference retrieves server-specific digest preference
func (r *DigestRepository) GetServerPreference(ctx context.Context, userID, serverID uuid.UUID) (*models.DigestServerPreference, error) {
	var pref models.DigestServerPreference
	query := `
		SELECT id, user_id, server_id, digest_mode, created_at, updated_at
		FROM digest_server_preferences
		WHERE user_id = $1 AND server_id = $2
	`
	err := r.db.GetContext(ctx, &pref, query, userID, serverID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &pref, err
}

// GetServerPreferences retrieves all server-specific digest preferences for a user
func (r *DigestRepository) GetServerPreferences(ctx context.Context, userID uuid.UUID) ([]models.DigestServerPreference, error) {
	var prefs []models.DigestServerPreference
	query := `
		SELECT id, user_id, server_id, digest_mode, created_at, updated_at
		FROM digest_server_preferences
		WHERE user_id = $1
	`
	err := r.db.SelectContext(ctx, &prefs, query, userID)
	if err != nil {
		return nil, err
	}
	if prefs == nil {
		prefs = []models.DigestServerPreference{}
	}
	return prefs, nil
}

// UpsertServerPreference creates or updates a server digest preference
func (r *DigestRepository) UpsertServerPreference(ctx context.Context, pref *models.DigestServerPreference) error {
	pref.UpdatedAt = time.Now()
	if pref.ID == uuid.Nil {
		pref.ID = uuid.New()
	}
	if pref.CreatedAt.IsZero() {
		pref.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO digest_server_preferences (
			id, user_id, server_id, digest_mode, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, server_id) DO UPDATE SET
			digest_mode = EXCLUDED.digest_mode,
			updated_at = EXCLUDED.updated_at
	`
	_, err := r.db.ExecContext(ctx, query,
		pref.ID, pref.UserID, pref.ServerID, pref.DigestMode,
		pref.CreatedAt, pref.UpdatedAt,
	)
	return err
}

// DeleteServerPreference deletes a server digest preference
func (r *DigestRepository) DeleteServerPreference(ctx context.Context, userID, serverID uuid.UUID) error {
	query := `DELETE FROM digest_server_preferences WHERE user_id = $1 AND server_id = $2`
	_, err := r.db.ExecContext(ctx, query, userID, serverID)
	return err
}

// --- Digest Queue ---

// QueueMessage adds a message to the digest queue
func (r *DigestRepository) QueueMessage(ctx context.Context, item *models.DigestQueueItem) error {
	item.ID = uuid.New()
	item.QueuedAt = time.Now()

	query := `
		INSERT INTO digest_queue (
			id, user_id, server_id, channel_id, message_id,
			message_content, message_author_id, message_author_name,
			message_created_at, is_mention, notification_type,
			queued_at, digest_period
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)
	`
	_, err := r.db.ExecContext(ctx, query,
		item.ID, item.UserID, item.ServerID, item.ChannelID, item.MessageID,
		item.MessageContent, item.MessageAuthorID, item.MessageAuthorName,
		item.MessageCreatedAt, item.IsMention, item.NotificationType,
		item.QueuedAt, item.DigestPeriod,
	)
	return err
}

// GetQueuedItems retrieves queued items for a user and period
func (r *DigestRepository) GetQueuedItems(ctx context.Context, userID uuid.UUID, before time.Time) ([]models.DigestQueueItem, error) {
	var items []models.DigestQueueItem
	query := `
		SELECT id, user_id, server_id, channel_id, message_id,
		       message_content, message_author_id, message_author_name,
		       message_created_at, is_mention, notification_type,
		       queued_at, digest_period
		FROM digest_queue
		WHERE user_id = $1 AND queued_at <= $2
		ORDER BY message_created_at ASC
	`
	err := r.db.SelectContext(ctx, &items, query, userID, before)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []models.DigestQueueItem{}
	}
	return items, nil
}

// GetQueuePreview gets summary stats for pending digest items
func (r *DigestRepository) GetQueuePreview(ctx context.Context, userID uuid.UUID) (*models.DigestPreview, error) {
	var preview models.DigestPreview
	query := `
		SELECT 
			COUNT(*) as pending_count,
			COUNT(*) FILTER (WHERE is_mention = true) as pending_mentions,
			COUNT(DISTINCT server_id) FILTER (WHERE server_id IS NOT NULL) as pending_servers,
			COUNT(DISTINCT channel_id) FILTER (WHERE channel_id IS NOT NULL) as pending_channels,
			MIN(queued_at) as oldest_pending
		FROM digest_queue
		WHERE user_id = $1
	`
	row := r.db.QueryRowContext(ctx, query, userID)
	err := row.Scan(
		&preview.PendingCount,
		&preview.PendingMentions,
		&preview.PendingServers,
		&preview.PendingChannels,
		&preview.OldestPending,
	)
	if err != nil {
		return nil, err
	}
	return &preview, nil
}

// DeleteQueuedItems deletes queued items for a user up to a time
func (r *DigestRepository) DeleteQueuedItems(ctx context.Context, userID uuid.UUID, before time.Time) (int64, error) {
	query := `DELETE FROM digest_queue WHERE user_id = $1 AND queued_at <= $2`
	result, err := r.db.ExecContext(ctx, query, userID, before)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// ClearQueue clears all queued items for a user
func (r *DigestRepository) ClearQueue(ctx context.Context, userID uuid.UUID) (int64, error) {
	query := `DELETE FROM digest_queue WHERE user_id = $1`
	result, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// --- Digest History ---

// CreateHistory creates a digest history entry
func (r *DigestRepository) CreateHistory(ctx context.Context, history *models.DigestHistory) error {
	history.ID = uuid.New()
	history.SentAt = time.Now()

	query := `
		INSERT INTO digest_history (
			id, user_id, sent_at, period_start, period_end, frequency,
			total_messages, total_mentions, servers_included, channels_included,
			content_json, status, error_message, retry_count
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		)
	`
	_, err := r.db.ExecContext(ctx, query,
		history.ID, history.UserID, history.SentAt, history.PeriodStart,
		history.PeriodEnd, history.Frequency, history.TotalMessages,
		history.TotalMentions, history.ServersIncluded, history.ChannelsIncluded,
		history.ContentJSON, history.Status, history.ErrorMessage, history.RetryCount,
	)
	return err
}

// UpdateHistoryStatus updates the status of a digest history entry
func (r *DigestRepository) UpdateHistoryStatus(ctx context.Context, id uuid.UUID, status models.DigestStatus, errorMessage *string) error {
	query := `
		UPDATE digest_history SET
			status = $2, error_message = $3,
			retry_count = retry_count + CASE WHEN $2 = 'failed' THEN 1 ELSE 0 END
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, id, status, errorMessage)
	return err
}

// GetHistory retrieves digest history for a user
func (r *DigestRepository) GetHistory(ctx context.Context, userID uuid.UUID, opts models.DigestHistoryListOptions) ([]models.DigestHistory, error) {
	if opts.Limit <= 0 || opts.Limit > 100 {
		opts.Limit = 20
	}

	var history []models.DigestHistory
	query := `
		SELECT id, user_id, sent_at, period_start, period_end, frequency,
		       total_messages, total_mentions, servers_included, channels_included,
		       content_json, status, error_message, retry_count
		FROM digest_history
		WHERE user_id = $1
		ORDER BY sent_at DESC
		LIMIT $2 OFFSET $3
	`
	err := r.db.SelectContext(ctx, &history, query, userID, opts.Limit, opts.Offset)
	if err != nil {
		return nil, err
	}
	if history == nil {
		history = []models.DigestHistory{}
	}
	return history, nil
}

// GetHistoryByID retrieves a specific digest history entry
func (r *DigestRepository) GetHistoryByID(ctx context.Context, id uuid.UUID) (*models.DigestHistory, error) {
	var history models.DigestHistory
	query := `
		SELECT id, user_id, sent_at, period_start, period_end, frequency,
		       total_messages, total_mentions, servers_included, channels_included,
		       content_json, status, error_message, retry_count
		FROM digest_history
		WHERE id = $1
	`
	err := r.db.GetContext(ctx, &history, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &history, err
}

// GetLastDigest retrieves the most recent digest for a user
func (r *DigestRepository) GetLastDigest(ctx context.Context, userID uuid.UUID) (*models.DigestHistory, error) {
	var history models.DigestHistory
	query := `
		SELECT id, user_id, sent_at, period_start, period_end, frequency,
		       total_messages, total_mentions, servers_included, channels_included,
		       content_json, status, error_message, retry_count
		FROM digest_history
		WHERE user_id = $1 AND status = 'sent'
		ORDER BY sent_at DESC
		LIMIT 1
	`
	err := r.db.GetContext(ctx, &history, query, userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &history, err
}

// --- Scheduling Helpers ---

// GetUsersForDigest retrieves users who should receive digests at a given time
func (r *DigestRepository) GetUsersForDigest(ctx context.Context, frequency models.DigestFrequency, hour int, day int) ([]uuid.UUID, error) {
	var userIDs []uuid.UUID

	var query string
	var args []interface{}

	switch frequency {
	case models.DigestFrequencyHourly:
		query = `
			SELECT user_id FROM digest_preferences
			WHERE enabled = true AND frequency = 'hourly'
		`
	case models.DigestFrequencyDaily:
		query = `
			SELECT user_id FROM digest_preferences
			WHERE enabled = true AND frequency = 'daily' AND preferred_hour = $1
		`
		args = []interface{}{hour}
	case models.DigestFrequencyWeekly:
		query = `
			SELECT user_id FROM digest_preferences
			WHERE enabled = true AND frequency = 'weekly' 
			AND preferred_hour = $1 AND preferred_day = $2
		`
		args = []interface{}{hour, day}
	default:
		return nil, fmt.Errorf("invalid frequency: %s", frequency)
	}

	err := r.db.SelectContext(ctx, &userIDs, query, args...)
	if err != nil {
		return nil, err
	}
	if userIDs == nil {
		userIDs = []uuid.UUID{}
	}
	return userIDs, nil
}

// CountPendingDigests counts how many pending digests exist
func (r *DigestRepository) CountPendingDigests(ctx context.Context) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM digest_history WHERE status = 'pending'`
	err := r.db.GetContext(ctx, &count, query)
	return count, err
}

// GetPendingDigests retrieves digests that need to be sent/retried
func (r *DigestRepository) GetPendingDigests(ctx context.Context, limit int) ([]models.DigestHistory, error) {
	var history []models.DigestHistory
	query := `
		SELECT id, user_id, sent_at, period_start, period_end, frequency,
		       total_messages, total_mentions, servers_included, channels_included,
		       content_json, status, error_message, retry_count
		FROM digest_history
		WHERE status = 'pending' AND retry_count < 3
		ORDER BY sent_at ASC
		LIMIT $1
	`
	err := r.db.SelectContext(ctx, &history, query, limit)
	if err != nil {
		return nil, err
	}
	if history == nil {
		history = []models.DigestHistory{}
	}
	return history, nil
}
