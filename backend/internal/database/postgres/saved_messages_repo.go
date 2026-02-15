package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"hearth/internal/models"
)

// SavedMessagesRepository handles saved/bookmarked messages database operations
type SavedMessagesRepository struct {
	db *sqlx.DB
}

// NewSavedMessagesRepository creates a new saved messages repository
func NewSavedMessagesRepository(db *sqlx.DB) *SavedMessagesRepository {
	return &SavedMessagesRepository{db: db}
}

// Save creates a new saved message entry
func (r *SavedMessagesRepository) Save(ctx context.Context, userID, messageID uuid.UUID, note *string) (*models.SavedMessage, error) {
	saved := &models.SavedMessage{
		ID:        uuid.New(),
		UserID:    userID,
		MessageID: messageID,
		Note:      note,
		CreatedAt: time.Now(),
	}

	query := `
		INSERT INTO saved_messages (id, user_id, message_id, note, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, message_id) DO UPDATE SET note = EXCLUDED.note
		RETURNING id, created_at
	`
	err := r.db.QueryRowContext(ctx, query,
		saved.ID, saved.UserID, saved.MessageID, saved.Note, saved.CreatedAt,
	).Scan(&saved.ID, &saved.CreatedAt)
	if err != nil {
		return nil, err
	}

	return saved, nil
}

// GetByID retrieves a saved message by its ID
func (r *SavedMessagesRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.SavedMessage, error) {
	var saved models.SavedMessage
	query := `SELECT * FROM saved_messages WHERE id = $1`
	err := r.db.GetContext(ctx, &saved, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &saved, nil
}

// GetByUserAndMessage retrieves a saved message by user and message ID
func (r *SavedMessagesRepository) GetByUserAndMessage(ctx context.Context, userID, messageID uuid.UUID) (*models.SavedMessage, error) {
	var saved models.SavedMessage
	query := `SELECT * FROM saved_messages WHERE user_id = $1 AND message_id = $2`
	err := r.db.GetContext(ctx, &saved, query, userID, messageID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &saved, nil
}

// GetByUser retrieves all saved messages for a user with pagination
func (r *SavedMessagesRepository) GetByUser(ctx context.Context, userID uuid.UUID, opts *models.SavedMessagesQueryOptions) ([]*models.SavedMessage, error) {
	var savedMessages []*models.SavedMessage
	var query string
	var args []interface{}

	limit := 50
	if opts != nil && opts.Limit > 0 && opts.Limit <= 100 {
		limit = opts.Limit
	}

	if opts != nil && opts.Before != nil {
		query = `
			SELECT sm.* FROM saved_messages sm
			WHERE sm.user_id = $1 AND sm.id < $2
			ORDER BY sm.created_at DESC
			LIMIT $3
		`
		args = []interface{}{userID, *opts.Before, limit}
	} else if opts != nil && opts.After != nil {
		query = `
			SELECT sm.* FROM saved_messages sm
			WHERE sm.user_id = $1 AND sm.id > $2
			ORDER BY sm.created_at ASC
			LIMIT $3
		`
		args = []interface{}{userID, *opts.After, limit}
	} else {
		query = `
			SELECT sm.* FROM saved_messages sm
			WHERE sm.user_id = $1
			ORDER BY sm.created_at DESC
			LIMIT $2
		`
		args = []interface{}{userID, limit}
	}

	err := r.db.SelectContext(ctx, &savedMessages, query, args...)
	if err != nil {
		return nil, err
	}

	return savedMessages, nil
}

// GetByUserWithMessages retrieves saved messages with full message data
func (r *SavedMessagesRepository) GetByUserWithMessages(ctx context.Context, userID uuid.UUID, opts *models.SavedMessagesQueryOptions) ([]*models.SavedMessage, error) {
	savedMessages, err := r.GetByUser(ctx, userID, opts)
	if err != nil {
		return nil, err
	}

	if len(savedMessages) == 0 {
		return savedMessages, nil
	}

	// Collect message IDs
	messageIDs := make([]uuid.UUID, len(savedMessages))
	for i, sm := range savedMessages {
		messageIDs[i] = sm.MessageID
	}

	// Fetch messages
	var messages []models.Message
	query, args, _ := sqlx.In(`SELECT * FROM messages WHERE id IN (?)`, messageIDs)
	query = r.db.Rebind(query)
	err = r.db.SelectContext(ctx, &messages, query, args...)
	if err != nil {
		return nil, err
	}

	// Map messages by ID
	messageMap := make(map[uuid.UUID]*models.Message)
	for i := range messages {
		messageMap[messages[i].ID] = &messages[i]
	}

	// Attach messages to saved messages
	for _, sm := range savedMessages {
		if msg, ok := messageMap[sm.MessageID]; ok {
			sm.Message = msg
		}
	}

	return savedMessages, nil
}

// UpdateNote updates the note on a saved message
func (r *SavedMessagesRepository) UpdateNote(ctx context.Context, id uuid.UUID, note *string) error {
	query := `UPDATE saved_messages SET note = $2 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, note)
	return err
}

// Delete removes a saved message
func (r *SavedMessagesRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM saved_messages WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// DeleteByUserAndMessage removes a saved message by user and message ID
func (r *SavedMessagesRepository) DeleteByUserAndMessage(ctx context.Context, userID, messageID uuid.UUID) error {
	query := `DELETE FROM saved_messages WHERE user_id = $1 AND message_id = $2`
	_, err := r.db.ExecContext(ctx, query, userID, messageID)
	return err
}

// Count returns the number of saved messages for a user
func (r *SavedMessagesRepository) Count(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM saved_messages WHERE user_id = $1`
	err := r.db.GetContext(ctx, &count, query, userID)
	return count, err
}

// IsSaved checks if a message is saved by a user
func (r *SavedMessagesRepository) IsSaved(ctx context.Context, userID, messageID uuid.UUID) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM saved_messages WHERE user_id = $1 AND message_id = $2)`
	err := r.db.GetContext(ctx, &exists, query, userID, messageID)
	return exists, err
}
