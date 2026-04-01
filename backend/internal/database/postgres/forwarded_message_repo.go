package postgres

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"hearth/internal/models"
)

type ForwardedMessageRepository struct {
	db *sqlx.DB
}

func NewForwardedMessageRepository(db *sqlx.DB) *ForwardedMessageRepository {
	return &ForwardedMessageRepository{db: db}
}

// Create inserts a new forwarded message record
func (r *ForwardedMessageRepository) Create(ctx context.Context, fm *models.ForwardedMessage) error {
	query := `
		INSERT INTO forwarded_messages (id, original_message_id, forwarded_by_id, destination_channel_id, comment, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.ExecContext(ctx, query,
		fm.ID, fm.OriginalMessageID, fm.ForwardedByID, fm.DestinationChannelID, fm.Comment, fm.CreatedAt,
	)
	return err
}

// GetByOriginalMessageID returns all forwarded message records for a given original message
func (r *ForwardedMessageRepository) GetByOriginalMessageID(ctx context.Context, originalMessageID uuid.UUID) ([]*models.ForwardedMessage, error) {
	var messages []*models.ForwardedMessage
	query := `
		SELECT id, original_message_id, forwarded_by_id, destination_channel_id, comment, created_at
		FROM forwarded_messages
		WHERE original_message_id = $1
		ORDER BY created_at DESC
	`
	err := r.db.SelectContext(ctx, &messages, query, originalMessageID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return messages, err
}

// GetByDestinationChannelID returns all forwarded messages sent to a given channel
func (r *ForwardedMessageRepository) GetByDestinationChannelID(ctx context.Context, channelID uuid.UUID, limit, offset int) ([]*models.ForwardedMessage, int, error) {
	var count int
	countQuery := `SELECT COUNT(*) FROM forwarded_messages WHERE destination_channel_id = $1`
	if err := r.db.GetContext(ctx, &count, countQuery, channelID); err != nil {
		return nil, 0, err
	}

	var messages []*models.ForwardedMessage
	query := `
		SELECT id, original_message_id, forwarded_by_id, destination_channel_id, comment, created_at
		FROM forwarded_messages
		WHERE destination_channel_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	err := r.db.SelectContext(ctx, &messages, query, channelID, limit, offset)
	if err == sql.ErrNoRows {
		return nil, count, nil
	}
	return messages, count, err
}

// GetByID returns a single forwarded message by ID
func (r *ForwardedMessageRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.ForwardedMessage, error) {
	var message models.ForwardedMessage
	query := `
		SELECT id, original_message_id, forwarded_by_id, destination_channel_id, comment, created_at
		FROM forwarded_messages
		WHERE id = $1
	`
	err := r.db.GetContext(ctx, &message, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &message, err
}
