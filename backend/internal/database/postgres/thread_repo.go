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

type ThreadRepository struct {
	db *sqlx.DB
}

func NewThreadRepository(db *sqlx.DB) *ThreadRepository {
	return &ThreadRepository{db: db}
}

// Create creates a new thread
func (r *ThreadRepository) Create(ctx context.Context, thread *models.Thread) error {
	query := `
		INSERT INTO threads (id, parent_channel_id, parent_message_id, owner_id, name, message_count, member_count, archived, auto_archive, locked, created_at, applied_tags, is_pinned, pin_weight, is_solved, solved_by, solved_at, solved_message_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`
	_, err := r.db.ExecContext(ctx, query,
		thread.ID, thread.ParentChannelID, thread.ParentMessageID, thread.OwnerID, thread.Name,
		thread.MessageCount, thread.MemberCount, thread.Archived, thread.AutoArchive,
		thread.Locked, thread.CreatedAt, thread.AppliedTags, thread.IsPinned, thread.PinWeight,
		thread.IsSolved, thread.SolvedBy, thread.SolvedAt, thread.SolvedMessageID,
	)
	if err != nil {
		return err
	}

	// Add owner as thread member
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO thread_members (thread_id, user_id) VALUES ($1, $2)`,
		thread.ID, thread.OwnerID,
	)
	return err
}

// GetByID retrieves a thread by ID
func (r *ThreadRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Thread, error) {
	var thread models.Thread
	query := `SELECT id, parent_channel_id, parent_message_id, owner_id, name, message_count, member_count, archived, auto_archive, locked, created_at, archive_timestamp, applied_tags, is_pinned, pin_weight, is_solved, solved_by, solved_at, solved_message_id FROM threads WHERE id = $1`
	err := r.db.GetContext(ctx, &thread, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &thread, nil
}

// Update updates a thread
func (r *ThreadRepository) Update(ctx context.Context, thread *models.Thread) error {
	query := `
		UPDATE threads SET
			name = $2, archived = $3, auto_archive = $4, locked = $5,
			message_count = $6, member_count = $7, archive_timestamp = $8,
			applied_tags = $9, is_pinned = $10, pin_weight = $11,
			is_solved = $12, solved_by = $13, solved_at = $14, solved_message_id = $15
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query,
		thread.ID, thread.Name, thread.Archived, thread.AutoArchive,
		thread.Locked, thread.MessageCount, thread.MemberCount, thread.ArchivedAt,
		thread.AppliedTags, thread.IsPinned, thread.PinWeight,
		thread.IsSolved, thread.SolvedBy, thread.SolvedAt, thread.SolvedMessageID,
	)
	return err
}

// Delete deletes a thread
func (r *ThreadRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM threads WHERE id = $1`, id)
	return err
}

// GetByChannelID retrieves all threads for a channel
func (r *ThreadRepository) GetByChannelID(ctx context.Context, channelID uuid.UUID) ([]*models.Thread, error) {
	var threads []*models.Thread
	query := `SELECT id, parent_channel_id, parent_message_id, owner_id, name, message_count, member_count, archived, auto_archive, locked, created_at, archive_timestamp, applied_tags, is_pinned, pin_weight, is_solved, solved_by, solved_at, solved_message_id FROM threads WHERE parent_channel_id = $1 ORDER BY is_pinned DESC, pin_weight DESC, archive_timestamp DESC NULLS LAST, created_at DESC`
	err := r.db.SelectContext(ctx, &threads, query, channelID)
	return threads, err
}

// GetByParentMessageID retrieves a thread by its parent message ID
func (r *ThreadRepository) GetByParentMessageID(ctx context.Context, messageID uuid.UUID) (*models.Thread, error) {
	var thread models.Thread
	query := `SELECT id, parent_channel_id, parent_message_id, owner_id, name, message_count, member_count, archived, auto_archive, locked, created_at, archive_timestamp FROM threads WHERE parent_message_id = $1`
	err := r.db.GetContext(ctx, &thread, query, messageID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &thread, nil
}

// GetActiveByChannelID retrieves non-archived threads for a channel
func (r *ThreadRepository) GetActiveByChannelID(ctx context.Context, channelID uuid.UUID) ([]*models.Thread, error) {
	var threads []*models.Thread
	query := `SELECT id, parent_channel_id, parent_message_id, owner_id, name, message_count, member_count, archived, auto_archive, locked, created_at, archive_timestamp FROM threads WHERE parent_channel_id = $1 AND archived = false ORDER BY created_at DESC`
	err := r.db.SelectContext(ctx, &threads, query, channelID)
	return threads, err
}

// GetThreadsPaginated retrieves threads for a forum channel with pagination, sorted by last_message_at
func (r *ThreadRepository) GetThreadsPaginated(ctx context.Context, channelID uuid.UUID, sortOrder int, limit, offset int, includeArchived bool) ([]models.Thread, int, error) {
	if limit <= 0 || limit > 50 {
		limit = 25
	}

	var threads []models.Thread
	var total int

	// Get total count
	countQuery := `SELECT COUNT(*) FROM threads WHERE parent_channel_id = $1`
	if !includeArchived {
		countQuery += ` AND archived = false`
	}
	err := r.db.GetContext(ctx, &total, countQuery, channelID)
	if err != nil {
		return nil, 0, err
	}

	// Build thread query with sorting
	threadQuery := `
		SELECT t.id, t.parent_channel_id, t.parent_message_id, t.owner_id, t.name, t.message_count, t.member_count,
		       t.archived, t.auto_archive, t.locked, t.created_at, t.archive_timestamp,
		       t.applied_tags, t.is_pinned, t.pin_weight,
		       COALESCE(tm.last_message_at, t.created_at) as last_message_at
		FROM threads t
		LEFT JOIN (
			SELECT thread_id, MAX(created_at) as last_message_at
			FROM thread_messages
			GROUP BY thread_id
		) tm ON tm.thread_id = t.id
		WHERE t.parent_channel_id = $1
	`

	args := []interface{}{channelID}

	if !includeArchived {
		threadQuery += ` AND t.archived = false`
	}

	// Determine sort order
	var orderClause string
	switch sortOrder {
	case 1: // creation_date
		orderClause = ` ORDER BY t.is_pinned DESC, t.pin_weight DESC, t.created_at DESC`
	case 2: // pin_weight
		orderClause = ` ORDER BY t.is_pinned DESC, t.pin_weight DESC, last_message_at DESC NULLS LAST`
	default: // latest_activity (0)
		orderClause = ` ORDER BY t.is_pinned DESC, t.pin_weight DESC, last_message_at DESC NULLS LAST`
	}

	threadQuery += orderClause + ` LIMIT $2 OFFSET $3`
	args = append(args, limit, offset)

	err = r.db.SelectContext(ctx, &threads, threadQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	return threads, total, nil
}

// GetThreadCount returns the number of threads in a channel
func (r *ThreadRepository) GetThreadCount(ctx context.Context, channelID uuid.UUID, includeArchived bool) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM threads WHERE parent_channel_id = $1`
	if !includeArchived {
		query += ` AND archived = false`
	}
	err := r.db.GetContext(ctx, &count, query, channelID)
	return count, err
}

// GetTotalMessageCount returns the total message count across all threads in a channel
func (r *ThreadRepository) GetTotalMessageCount(ctx context.Context, channelID uuid.UUID) (int, error) {
	var count int
	query := `
		SELECT COALESCE(SUM(message_count), 0)
		FROM threads
		WHERE parent_channel_id = $1
	`
	err := r.db.GetContext(ctx, &count, query, channelID)
	return count, err
}

// Archive archives a thread
func (r *ThreadRepository) Archive(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	query := `UPDATE threads SET archived = true, archive_timestamp = $2 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, now)
	return err
}

// Unarchive unarchives a thread
func (r *ThreadRepository) Unarchive(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE threads SET archived = false, archive_timestamp = NULL WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// AddMember adds a user to a thread
func (r *ThreadRepository) AddMember(ctx context.Context, threadID, userID uuid.UUID) error {
	query := `INSERT INTO thread_members (thread_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
	result, err := r.db.ExecContext(ctx, query, threadID, userID)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		// Increment member count
		_, err = r.db.ExecContext(ctx, `UPDATE threads SET member_count = member_count + 1 WHERE id = $1`, threadID)
	}
	return err
}

// RemoveMember removes a user from a thread
func (r *ThreadRepository) RemoveMember(ctx context.Context, threadID, userID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM thread_members WHERE thread_id = $1 AND user_id = $2`, threadID, userID)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		// Decrement member count
		_, err = r.db.ExecContext(ctx, `UPDATE threads SET member_count = GREATEST(member_count - 1, 0) WHERE id = $1`, threadID)
	}
	return err
}

// IsMember checks if a user is a member of a thread
func (r *ThreadRepository) IsMember(ctx context.Context, threadID, userID uuid.UUID) (bool, error) {
	var count int
	err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM thread_members WHERE thread_id = $1 AND user_id = $2`, threadID, userID)
	return count > 0, err
}

// GetMembers gets all members of a thread
func (r *ThreadRepository) GetMembers(ctx context.Context, threadID uuid.UUID) ([]uuid.UUID, error) {
	var members []uuid.UUID
	err := r.db.SelectContext(ctx, &members, `SELECT user_id FROM thread_members WHERE thread_id = $1`, threadID)
	return members, err
}

// CreateMessage creates a message in a thread
func (r *ThreadRepository) CreateMessage(ctx context.Context, threadID, authorID uuid.UUID, content string) (*models.ThreadMessage, error) {
	msg := &models.ThreadMessage{
		ID:        uuid.New(),
		ThreadID:  threadID,
		AuthorID:  authorID,
		Content:   content,
		CreatedAt: time.Now(),
	}

	query := `INSERT INTO thread_messages (id, thread_id, author_id, content, created_at) VALUES ($1, $2, $3, $4, $5)`
	_, err := r.db.ExecContext(ctx, query, msg.ID, msg.ThreadID, msg.AuthorID, msg.Content, msg.CreatedAt)
	if err != nil {
		return nil, err
	}

	// Increment message count
	_, err = r.db.ExecContext(ctx, `UPDATE threads SET message_count = message_count + 1 WHERE id = $1`, threadID)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

// GetMessages retrieves messages from a thread with pagination
func (r *ThreadRepository) GetMessages(ctx context.Context, threadID uuid.UUID, before *uuid.UUID, limit int) ([]*models.ThreadMessage, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var messages []*models.ThreadMessage
	var err error

	if before != nil {
		query := `
			SELECT id, thread_id, author_id, content, created_at, edited_at 
			FROM thread_messages 
			WHERE thread_id = $1 AND created_at < (SELECT created_at FROM thread_messages WHERE id = $2)
			ORDER BY created_at DESC 
			LIMIT $3
		`
		err = r.db.SelectContext(ctx, &messages, query, threadID, *before, limit)
	} else {
		query := `
			SELECT id, thread_id, author_id, content, created_at, edited_at 
			FROM thread_messages 
			WHERE thread_id = $1 
			ORDER BY created_at DESC 
			LIMIT $2
		`
		err = r.db.SelectContext(ctx, &messages, query, threadID, limit)
	}

	return messages, err
}

// IncrementMessageCount increments the message count for a thread
func (r *ThreadRepository) IncrementMessageCount(ctx context.Context, threadID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `UPDATE threads SET message_count = message_count + 1 WHERE id = $1`, threadID)
	return err
}

// ============================================================================
// Thread Notification Preferences
// ============================================================================

// GetNotificationPreference gets a user's notification preference for a thread
func (r *ThreadRepository) GetNotificationPreference(ctx context.Context, threadID, userID uuid.UUID) (*models.ThreadNotificationPreference, error) {
	var pref models.ThreadNotificationPreference
	query := `SELECT thread_id, user_id, level, created_at, updated_at FROM thread_notification_preferences WHERE thread_id = $1 AND user_id = $2`
	err := r.db.GetContext(ctx, &pref, query, threadID, userID)
	if err == sql.ErrNoRows {
		// Return default preference
		return &models.ThreadNotificationPreference{
			ThreadID: threadID,
			UserID:   userID,
			Level:    models.ThreadNotifyAll,
		}, nil
	}
	if err != nil {
		return nil, err
	}
	return &pref, nil
}

// SetNotificationPreference sets a user's notification preference for a thread
func (r *ThreadRepository) SetNotificationPreference(ctx context.Context, pref *models.ThreadNotificationPreference) error {
	query := `
		INSERT INTO thread_notification_preferences (thread_id, user_id, level, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		ON CONFLICT (thread_id, user_id) 
		DO UPDATE SET level = EXCLUDED.level, updated_at = NOW()
	`
	_, err := r.db.ExecContext(ctx, query, pref.ThreadID, pref.UserID, pref.Level)
	return err
}

// DeleteNotificationPreference removes a user's notification preference for a thread
func (r *ThreadRepository) DeleteNotificationPreference(ctx context.Context, threadID, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM thread_notification_preferences WHERE thread_id = $1 AND user_id = $2`, threadID, userID)
	return err
}

// GetNotificationPreferencesForUser gets all notification preferences for a user
func (r *ThreadRepository) GetNotificationPreferencesForUser(ctx context.Context, userID uuid.UUID) ([]*models.ThreadNotificationPreference, error) {
	var prefs []*models.ThreadNotificationPreference
	query := `SELECT thread_id, user_id, level, created_at, updated_at FROM thread_notification_preferences WHERE user_id = $1`
	err := r.db.SelectContext(ctx, &prefs, query, userID)
	return prefs, err
}

// ============================================================================
// Thread Presence (Active Viewers)
// ============================================================================

// SetPresence marks a user as actively viewing a thread
func (r *ThreadRepository) SetPresence(ctx context.Context, threadID, userID uuid.UUID) error {
	query := `
		INSERT INTO thread_presence (thread_id, user_id, last_seen_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (thread_id, user_id) 
		DO UPDATE SET last_seen_at = NOW()
	`
	_, err := r.db.ExecContext(ctx, query, threadID, userID)
	return err
}

// RemovePresence removes a user's presence from a thread
func (r *ThreadRepository) RemovePresence(ctx context.Context, threadID, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM thread_presence WHERE thread_id = $1 AND user_id = $2`, threadID, userID)
	return err
}

// GetActiveViewers gets users currently viewing a thread (seen in last 5 minutes)
func (r *ThreadRepository) GetActiveViewers(ctx context.Context, threadID uuid.UUID) ([]models.ThreadPresenceUser, error) {
	var viewers []models.ThreadPresenceUser
	query := `
		SELECT u.id, u.username, u.display_name, u.avatar_url
		FROM thread_presence tp
		JOIN users u ON u.id = tp.user_id
		WHERE tp.thread_id = $1 AND tp.last_seen_at > NOW() - INTERVAL '5 minutes'
		ORDER BY tp.last_seen_at DESC
	`
	err := r.db.SelectContext(ctx, &viewers, query, threadID)
	return viewers, err
}

// CleanupStalePresence removes stale presence records (older than 5 minutes)
func (r *ThreadRepository) CleanupStalePresence(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM thread_presence WHERE last_seen_at < NOW() - INTERVAL '5 minutes'`)
	return err
}

// UpdatePresenceHeartbeat updates the last_seen_at for an active viewer
func (r *ThreadRepository) UpdatePresenceHeartbeat(ctx context.Context, threadID, userID uuid.UUID) error {
	query := `UPDATE thread_presence SET last_seen_at = NOW() WHERE thread_id = $1 AND user_id = $2`
	_, err := r.db.ExecContext(ctx, query, threadID, userID)
	return err
}

// GetMessagesWithAuthors retrieves messages from a thread with author details
func (r *ThreadRepository) GetMessagesWithAuthors(ctx context.Context, threadID uuid.UUID, before *uuid.UUID, limit int) ([]*models.ThreadMessage, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var messages []*models.ThreadMessage
	var err error

	baseQuery := `
		SELECT 
			tm.id, tm.thread_id, tm.author_id, tm.content, tm.created_at, tm.edited_at,
			u.id as "author.id", u.username as "author.username", 
			u.display_name as "author.display_name", u.avatar_url as "author.avatar"
		FROM thread_messages tm
		LEFT JOIN users u ON u.id = tm.author_id
		WHERE tm.thread_id = $1
	`

	if before != nil {
		query := baseQuery + ` AND tm.created_at < (SELECT created_at FROM thread_messages WHERE id = $2)
			ORDER BY tm.created_at DESC LIMIT $3`
		err = r.db.SelectContext(ctx, &messages, query, threadID, *before, limit)
	} else {
		query := baseQuery + ` ORDER BY tm.created_at DESC LIMIT $2`
		err = r.db.SelectContext(ctx, &messages, query, threadID, limit)
	}

	return messages, err
}

// ThreadAutoArchiveRepository handles database operations for thread auto-archive
type ThreadAutoArchiveRepository struct {
	db *sqlx.DB
}

// NewThreadAutoArchiveRepository creates a new thread auto-archive repository
func NewThreadAutoArchiveRepository(db *sqlx.DB) *ThreadAutoArchiveRepository {
	return &ThreadAutoArchiveRepository{db: db}
}

// CreateServerSettings creates server-level auto-archive settings
func (r *ThreadAutoArchiveRepository) CreateServerSettings(ctx context.Context, settings *models.ThreadAutoArchiveSettings) error {
	query := `
		INSERT INTO thread_auto_archive_settings (id, server_id, default_duration, allow_override, archive_duration_options, require_post_author, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.ExecContext(ctx, query,
		settings.ID, settings.ServerID, settings.DefaultDuration, settings.AllowOverride,
		settings.ArchiveDurationOptions, settings.RequirePostAuthor, settings.CreatedAt, settings.UpdatedAt,
	)
	return err
}

// GetServerSettings retrieves auto-archive settings for a server
func (r *ThreadAutoArchiveRepository) GetServerSettings(ctx context.Context, serverID uuid.UUID) (*models.ThreadAutoArchiveSettings, error) {
	var settings models.ThreadAutoArchiveSettings
	query := `
		SELECT id, server_id, default_duration, allow_override, archive_duration_options, require_post_author, created_at, updated_at 
		FROM thread_auto_archive_settings WHERE server_id = $1
	`
	err := r.db.GetContext(ctx, &settings, query, serverID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &settings, nil
}

// UpdateServerSettings updates server-level auto-archive settings
func (r *ThreadAutoArchiveRepository) UpdateServerSettings(ctx context.Context, settings *models.ThreadAutoArchiveSettings) error {
	query := `
		UPDATE thread_auto_archive_settings SET
			default_duration = $2, allow_override = $3, archive_duration_options = $4,
			require_post_author = $5, updated_at = $6
		WHERE server_id = $1
	`
	_, err := r.db.ExecContext(ctx, query,
		settings.ServerID, settings.DefaultDuration, settings.AllowOverride,
		settings.ArchiveDurationOptions, settings.RequirePostAuthor, time.Now(),
	)
	return err
}

// DeleteServerSettings deletes server-level auto-archive settings
func (r *ThreadAutoArchiveRepository) DeleteServerSettings(ctx context.Context, serverID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM thread_auto_archive_settings WHERE server_id = $1`, serverID)
	return err
}

// SetChannelOverride creates or updates channel-level auto-archive override
func (r *ThreadAutoArchiveRepository) SetChannelOverride(ctx context.Context, override *models.ChannelAutoArchiveOverride) error {
	query := `
		INSERT INTO channel_auto_archive_override (id, channel_id, auto_archive_duration, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (channel_id) DO UPDATE SET
			auto_archive_duration = $3, updated_at = $5
	`
	_, err := r.db.ExecContext(ctx, query,
		override.ID, override.ChannelID, override.AutoArchiveDuration, override.CreatedAt, override.UpdatedAt,
	)
	return err
}

// GetChannelOverride retrieves channel-level auto-archive override
func (r *ThreadAutoArchiveRepository) GetChannelOverride(ctx context.Context, channelID uuid.UUID) (*models.ChannelAutoArchiveOverride, error) {
	var override models.ChannelAutoArchiveOverride
	query := `
		SELECT id, channel_id, auto_archive_duration, created_at, updated_at 
		FROM channel_auto_archive_override WHERE channel_id = $1
	`
	err := r.db.GetContext(ctx, &override, query, channelID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &override, nil
}

// DeleteChannelOverride deletes channel-level auto-archive override
func (r *ThreadAutoArchiveRepository) DeleteChannelOverride(ctx context.Context, channelID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM channel_auto_archive_override WHERE channel_id = $1`, channelID)
	return err
}

// GetOrCreateThreadMeta retrieves or creates auto-archive metadata for a thread
func (r *ThreadAutoArchiveRepository) GetOrCreateThreadMeta(ctx context.Context, threadID uuid.UUID) (*models.ThreadAutoArchiveMeta, error) {
	var meta models.ThreadAutoArchiveMeta

	// First try to get existing
	query := `
		SELECT thread_id, last_activity_at, last_activity_message_id, last_activity_user_id, 
		       next_archive_at, archive_eligible, bumped_by_owner, created_at, updated_at 
		FROM thread_auto_archive_meta WHERE thread_id = $1
	`
	err := r.db.GetContext(ctx, &meta, query, threadID)
	if err == sql.ErrNoRows {
		// Create new
		now := time.Now()
		insertQuery := `
			INSERT INTO thread_auto_archive_meta (thread_id, last_activity_at, created_at, updated_at)
			VALUES ($1, $2, $3, $4)
			RETURNING thread_id, last_activity_at, last_activity_message_id, last_activity_user_id, 
			          next_archive_at, archive_eligible, bumped_by_owner, created_at, updated_at
		`
		err = r.db.GetContext(ctx, &meta, insertQuery, threadID, now, now, now)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	return &meta, nil
}

// UpdateThreadMeta updates auto-archive metadata for a thread
func (r *ThreadAutoArchiveRepository) UpdateThreadMeta(ctx context.Context, meta *models.ThreadAutoArchiveMeta) error {
	query := `
		UPDATE thread_auto_archive_meta SET
			last_activity_at = $2, last_activity_message_id = $3, last_activity_user_id = $4,
			next_archive_at = $5, archive_eligible = $6, bumped_by_owner = $7, updated_at = $8
		WHERE thread_id = $1
	`
	_, err := r.db.ExecContext(ctx, query,
		meta.ThreadID, meta.LastActivityAt, meta.LastActivityMessageID, meta.LastActivityUserID,
		meta.NextArchiveAt, meta.ArchiveEligible, meta.BumpedByOwner, time.Now(),
	)
	return err
}

// SetThreadNextArchive sets the next archive time for a thread
func (r *ThreadAutoArchiveRepository) SetThreadNextArchive(ctx context.Context, threadID uuid.UUID, nextArchiveAt *time.Time) error {
	query := `UPDATE thread_auto_archive_meta SET next_archive_at = $2, updated_at = $3 WHERE thread_id = $1`
	_, err := r.db.ExecContext(ctx, query, threadID, nextArchiveAt, time.Now())
	return err
}

// SetThreadArchiveEligible sets the archive eligibility for a thread
func (r *ThreadAutoArchiveRepository) SetThreadArchiveEligible(ctx context.Context, threadID uuid.UUID, eligible bool) error {
	query := `UPDATE thread_auto_archive_meta SET archive_eligible = $2, updated_at = $3 WHERE thread_id = $1`
	_, err := r.db.ExecContext(ctx, query, threadID, eligible, time.Now())
	return err
}

// BumpThreadOwnerActivity marks that the owner has posted in the thread
func (r *ThreadAutoArchiveRepository) BumpThreadOwnerActivity(ctx context.Context, threadID uuid.UUID) error {
	query := `UPDATE thread_auto_archive_meta SET bumped_by_owner = TRUE, updated_at = $2 WHERE thread_id = $1`
	_, err := r.db.ExecContext(ctx, query, threadID, time.Now())
	return err
}

// GetThreadsReadyForArchive retrieves threads ready to be archived
func (r *ThreadAutoArchiveRepository) GetThreadsReadyForArchive(ctx context.Context, limit int) ([]*models.ThreadAutoArchiveMeta, error) {
	var metas []*models.ThreadAutoArchiveMeta
	query := `
		SELECT tam.thread_id, tam.last_activity_at, tam.last_activity_message_id, tam.last_activity_user_id, 
		       tam.next_archive_at, tam.archive_eligible, tam.bumped_by_owner, tam.created_at, tam.updated_at
		FROM thread_auto_archive_meta tam
		JOIN threads t ON t.id = tam.thread_id
		WHERE tam.next_archive_at IS NOT NULL 
		  AND tam.next_archive_at <= NOW() 
		  AND tam.archive_eligible = TRUE
		  AND t.archived = FALSE
		ORDER BY tam.next_archive_at ASC
		LIMIT $1
	`
	err := r.db.SelectContext(ctx, &metas, query, limit)
	return metas, err
}

// GetThreadMeta retrieves auto-archive metadata for a thread
func (r *ThreadAutoArchiveRepository) GetThreadMeta(ctx context.Context, threadID uuid.UUID) (*models.ThreadAutoArchiveMeta, error) {
	var meta models.ThreadAutoArchiveMeta
	query := `
		SELECT thread_id, last_activity_at, last_activity_message_id, last_activity_user_id, 
		       next_archive_at, archive_eligible, bumped_by_owner, created_at, updated_at 
		FROM thread_auto_archive_meta WHERE thread_id = $1
	`
	err := r.db.GetContext(ctx, &meta, query, threadID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &meta, nil
}

// DeleteThreadMeta deletes auto-archive metadata for a thread
func (r *ThreadAutoArchiveRepository) DeleteThreadMeta(ctx context.Context, threadID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM thread_auto_archive_meta WHERE thread_id = $1`, threadID)
	return err
}

// GetServerStats retrieves auto-archive statistics for a server
func (r *ThreadAutoArchiveRepository) GetServerStats(ctx context.Context, serverID uuid.UUID) (*models.ThreadAutoArchiveStats, error) {
	var stats models.ThreadAutoArchiveStats
	stats.ServerID = serverID

	// Get total threads in server channels
	query := `
		SELECT COUNT(*) FROM threads t
		JOIN channels c ON c.id = t.parent_channel_id
		WHERE c.server_id = $1
	`
	err := r.db.GetContext(ctx, &stats.TotalThreads, query, serverID)
	if err != nil {
		return nil, err
	}

	// Get archived threads
	query = `
		SELECT COUNT(*) FROM threads t
		JOIN channels c ON c.id = t.parent_channel_id
		WHERE c.server_id = $1 AND t.archived = TRUE
	`
	err = r.db.GetContext(ctx, &stats.ArchivedThreads, query, serverID)
	if err != nil {
		return nil, err
	}

	// Get scheduled threads (have next_archive_at set and not archived)
	query = `
		SELECT COUNT(*) FROM thread_auto_archive_meta tam
		JOIN threads t ON t.id = tam.thread_id
		JOIN channels c ON c.id = t.parent_channel_id
		WHERE c.server_id = $1 AND tam.next_archive_at IS NOT NULL AND t.archived = FALSE
	`
	err = r.db.GetContext(ctx, &stats.ScheduledThreads, query, serverID)
	if err != nil {
		return nil, err
	}

	// Get threads ready to archive
	query = `
		SELECT COUNT(*) FROM thread_auto_archive_meta tam
		JOIN threads t ON t.id = tam.thread_id
		JOIN channels c ON c.id = t.parent_channel_id
		WHERE c.server_id = $1 AND tam.next_archive_at IS NOT NULL 
		  AND tam.next_archive_at <= NOW() AND tam.archive_eligible = TRUE AND t.archived = FALSE
	`
	err = r.db.GetContext(ctx, &stats.ReadyToArchiveThreads, query, serverID)
	if err != nil {
		return nil, err
	}

	return &stats, nil
}

// GetChannelDuration gets the effective auto-archive duration for a channel
func (r *ThreadAutoArchiveRepository) GetChannelDuration(ctx context.Context, channelID, serverID uuid.UUID) (int, error) {
	// First check for channel override
	var overrideDuration int
	query := `SELECT auto_archive_duration FROM channel_auto_archive_override WHERE channel_id = $1`
	err := r.db.GetContext(ctx, &overrideDuration, query, channelID)
	if err == nil {
		return overrideDuration, nil
	}
	if err != sql.ErrNoRows {
		return 0, err
	}

	// Fall back to server default
	var settings models.ThreadAutoArchiveSettings
	query = `SELECT default_duration FROM thread_auto_archive_settings WHERE server_id = $1`
	err = r.db.GetContext(ctx, &settings, query, serverID)
	if err == sql.ErrNoRows {
		return 1440, nil // Default 24 hours
	}
	if err != nil {
		return 0, err
	}

	return settings.DefaultDuration, nil
}

// ForumTagRepository handles forum tag data access
type ForumTagRepository struct {
	db *sqlx.DB
}

// NewForumTagRepository creates a new forum tag repository
func NewForumTagRepository(db *sqlx.DB) *ForumTagRepository {
	return &ForumTagRepository{db: db}
}

// Create creates a new forum tag
func (r *ForumTagRepository) Create(ctx context.Context, tag *models.ForumTag) error {
	query := `
		INSERT INTO forum_tags (id, server_id, channel_id, name, color, emoji_name, moderated, position, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.ExecContext(ctx, query,
		tag.ID, tag.ServerID, tag.ChannelID, tag.Name, tag.Color, tag.EmojiName, tag.Moderated, tag.Position, tag.CreatedAt,
	)
	return err
}

// GetByID retrieves a tag by ID
func (r *ForumTagRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.ForumTag, error) {
	var tag models.ForumTag
	query := `SELECT id, server_id, channel_id, name, color, emoji_name, moderated, position, created_at FROM forum_tags WHERE id = $1`
	err := r.db.GetContext(ctx, &tag, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &tag, err
}

// GetByChannel retrieves all tags for a channel
func (r *ForumTagRepository) GetByChannel(ctx context.Context, channelID uuid.UUID) ([]models.ForumTag, error) {
	var tags []models.ForumTag
	query := `SELECT id, server_id, channel_id, name, color, emoji_name, moderated, position, created_at FROM forum_tags WHERE channel_id = $1 ORDER BY position, name`
	err := r.db.SelectContext(ctx, &tags, query, channelID)
	if err != nil {
		return nil, err
	}
	return tags, nil
}

// GetByIDs retrieves tags by their IDs
func (r *ForumTagRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]models.ForumTag, error) {
	if len(ids) == 0 {
		return []models.ForumTag{}, nil
	}
	var tags []models.ForumTag
	query := `SELECT id, server_id, channel_id, name, color, emoji_name, moderated, position, created_at FROM forum_tags WHERE id = ANY($1) ORDER BY position, name`
	err := r.db.SelectContext(ctx, &tags, query, ids)
	if err != nil {
		return nil, err
	}
	return tags, nil
}

// Update updates a forum tag
func (r *ForumTagRepository) Update(ctx context.Context, tag *models.ForumTag) error {
	query := `
		UPDATE forum_tags
		SET name = $2, color = $3, emoji_name = $4, moderated = $5, position = $6
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, tag.ID, tag.Name, tag.Color, tag.EmojiName, tag.Moderated, tag.Position)
	return err
}

// Delete deletes a forum tag
func (r *ForumTagRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM forum_tags WHERE id = $1`, id)
	return err
}

// ApplyTags applies tags to a thread
func (r *ForumTagRepository) ApplyTags(ctx context.Context, threadID uuid.UUID, tagIDs []uuid.UUID) error {
	query := `UPDATE threads SET applied_tags = $2 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, threadID, tagIDs)
	return err
}

// GetThreadTags retrieves tags applied to a thread
func (r *ForumTagRepository) GetThreadTags(ctx context.Context, threadID uuid.UUID) ([]models.ForumTag, error) {
	var tags []models.ForumTag
	query := `
		SELECT ft.id, ft.server_id, ft.channel_id, ft.name, ft.color, ft.emoji_name, ft.moderated, ft.position, ft.created_at
		FROM forum_tags ft
		JOIN threads t ON t.applied_tags @> ARRAY[ft.id]
		WHERE t.id = $1
		ORDER BY ft.position, ft.name
	`
	err := r.db.SelectContext(ctx, &tags, query, threadID)
	if err != nil {
		return nil, err
	}
	return tags, nil
}

// FilterThreads filters threads with full ForumPostFilter support
func (r *ForumTagRepository) FilterThreads(ctx context.Context, channelID uuid.UUID, filter *models.ForumPostFilter, limit, offset int) ([]models.Thread, int, error) {
	var threads []models.Thread
	var total int

	countQuery := `SELECT COUNT(*) FROM threads WHERE parent_channel_id = $1`
	threadQuery := `
		SELECT id, parent_channel_id, parent_message_id, owner_id, name, message_count, member_count,
		       archived, auto_archive, locked, created_at, archive_timestamp,
		       applied_tags, is_pinned, pin_weight, is_solved, solved_by, solved_at, solved_message_id
		FROM threads
		WHERE parent_channel_id = $1
	`

	args := []interface{}{channelID}
	argIndex := 2

	// Filter by tags
	if len(filter.TagIDs) > 0 {
		countQuery += fmt.Sprintf(` AND applied_tags && $%d`, argIndex)
		threadQuery += fmt.Sprintf(` AND applied_tags && $%d`, argIndex)
		args = append(args, filter.TagIDs)
		argIndex++
	}

	// Filter by author
	if filter.AuthorID != nil {
		countQuery += fmt.Sprintf(` AND owner_id = $%d`, argIndex)
		threadQuery += fmt.Sprintf(` AND owner_id = $%d`, argIndex)
		args = append(args, *filter.AuthorID)
		argIndex++
	}

	// Filter by pinned only
	if filter.PinnedOnly {
		countQuery += ` AND is_pinned = TRUE`
		threadQuery += ` AND is_pinned = TRUE`
	}

	// Filter by search query (searches in name)
	if filter.SearchQuery != "" {
		searchPattern := "%" + filter.SearchQuery + "%"
		countQuery += fmt.Sprintf(` AND name ILIKE $%d`, argIndex)
		threadQuery += fmt.Sprintf(` AND name ILIKE $%d`, argIndex)
		args = append(args, searchPattern)
		argIndex++
	}

	// Determine sort order
	var orderClause string
	switch filter.SortOrder {
	case 1: // creation_date
		orderClause = ` ORDER BY is_pinned DESC, pin_weight DESC, created_at DESC`
	case 2: // pin_weight
		orderClause = ` ORDER BY is_pinned DESC, pin_weight DESC, archive_timestamp DESC NULLS LAST, created_at DESC`
	case 3: // most_reactions (would need join with reactions table - for now use message_count as proxy)
		orderClause = ` ORDER BY is_pinned DESC, pin_weight DESC, message_count DESC, created_at DESC`
	case 4: // solved_first
		orderClause = ` ORDER BY is_pinned DESC, pin_weight DESC, is_solved DESC, archive_timestamp DESC NULLS LAST, created_at DESC`
	default: // latest_activity (0)
		orderClause = ` ORDER BY is_pinned DESC, pin_weight DESC, archive_timestamp DESC NULLS LAST, created_at DESC`
	}

	threadQuery += orderClause + fmt.Sprintf(` LIMIT $%d OFFSET $%d`, argIndex, argIndex+1)

	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	err = r.db.SelectContext(ctx, &threads, threadQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	return threads, total, nil
}
