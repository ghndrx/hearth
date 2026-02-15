package postgres

import (
	"context"
	"database/sql"
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
		INSERT INTO threads (id, parent_channel_id, owner_id, name, message_count, member_count, archived, auto_archive, locked, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.db.ExecContext(ctx, query,
		thread.ID, thread.ParentChannelID, thread.OwnerID, thread.Name,
		thread.MessageCount, thread.MemberCount, thread.Archived, thread.AutoArchive,
		thread.Locked, thread.CreatedAt,
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
	query := `SELECT id, parent_channel_id, owner_id, name, message_count, member_count, archived, auto_archive, locked, created_at, archive_timestamp FROM threads WHERE id = $1`
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
			message_count = $6, member_count = $7, archive_timestamp = $8
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query,
		thread.ID, thread.Name, thread.Archived, thread.AutoArchive,
		thread.Locked, thread.MessageCount, thread.MemberCount, thread.ArchiveTimestamp,
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
	query := `SELECT id, parent_channel_id, owner_id, name, message_count, member_count, archived, auto_archive, locked, created_at, archive_timestamp FROM threads WHERE parent_channel_id = $1 ORDER BY created_at DESC`
	err := r.db.SelectContext(ctx, &threads, query, channelID)
	return threads, err
}

// GetActiveByChannelID retrieves non-archived threads for a channel
func (r *ThreadRepository) GetActiveByChannelID(ctx context.Context, channelID uuid.UUID) ([]*models.Thread, error) {
	var threads []*models.Thread
	query := `SELECT id, parent_channel_id, owner_id, name, message_count, member_count, archived, auto_archive, locked, created_at, archive_timestamp FROM threads WHERE parent_channel_id = $1 AND archived = false ORDER BY created_at DESC`
	err := r.db.SelectContext(ctx, &threads, query, channelID)
	return threads, err
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
		SELECT u.id, u.username, u.display_name, u.avatar
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
			u.display_name as "author.display_name", u.avatar as "author.avatar"
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
