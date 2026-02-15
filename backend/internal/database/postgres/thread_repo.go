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
