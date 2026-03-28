package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"hearth/internal/models"
)

// WebhookRepository handles webhook database operations
type WebhookRepository struct {
	db *sqlx.DB
}

// NewWebhookRepository creates a new webhook repository
func NewWebhookRepository(db *sqlx.DB) *WebhookRepository {
	return &WebhookRepository{db: db}
}

// Create creates a new webhook
func (r *WebhookRepository) Create(ctx context.Context, webhook *models.Webhook) error {
	query := `
		INSERT INTO webhooks (
			id, type, server_id, channel_id, creator_id, name, avatar, token,
			application_id, source_server_id, source_channel_id, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)
	`
	_, err := r.db.ExecContext(ctx, query,
		webhook.ID,
		webhook.Type,
		webhook.ServerID,
		webhook.ChannelID,
		webhook.CreatorID,
		webhook.Name,
		webhook.Avatar,
		webhook.Token,
		webhook.ApplicationID,
		webhook.SourceServerID,
		webhook.SourceChannelID,
		webhook.CreatedAt,
	)
	return err
}

// GetByID retrieves a webhook by ID
func (r *WebhookRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Webhook, error) {
	query := `
		SELECT id, type, server_id, channel_id, creator_id, name, avatar, token,
			   application_id, source_server_id, source_channel_id, created_at
		FROM webhooks
		WHERE id = $1
	`
	var webhook models.Webhook
	err := r.db.GetContext(ctx, &webhook, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &webhook, nil
}

// GetByChannelID retrieves all webhooks for a channel
func (r *WebhookRepository) GetByChannelID(ctx context.Context, channelID uuid.UUID) ([]*models.Webhook, error) {
	query := `
		SELECT id, type, server_id, channel_id, creator_id, name, avatar, token,
			   application_id, source_server_id, source_channel_id, created_at
		FROM webhooks
		WHERE channel_id = $1
		ORDER BY created_at ASC
	`
	var webhooks []*models.Webhook
	err := r.db.SelectContext(ctx, &webhooks, query, channelID)
	if err != nil {
		return nil, err
	}
	return webhooks, nil
}

// GetByServerID retrieves all webhooks for a server
func (r *WebhookRepository) GetByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.Webhook, error) {
	query := `
		SELECT id, type, server_id, channel_id, creator_id, name, avatar, token,
			   application_id, source_server_id, source_channel_id, created_at
		FROM webhooks
		WHERE server_id = $1
		ORDER BY created_at ASC
	`
	var webhooks []*models.Webhook
	err := r.db.SelectContext(ctx, &webhooks, query, serverID)
	if err != nil {
		return nil, err
	}
	return webhooks, nil
}

// Update updates a webhook
func (r *WebhookRepository) Update(ctx context.Context, webhook *models.Webhook) error {
	query := `
		UPDATE webhooks
		SET name = $2, avatar = $3, channel_id = $4
		WHERE id = $1
	`
	result, err := r.db.ExecContext(ctx, query,
		webhook.ID,
		webhook.Name,
		webhook.Avatar,
		webhook.ChannelID,
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

// Delete deletes a webhook
func (r *WebhookRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM webhooks WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
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

// CountByChannelID counts webhooks in a channel
func (r *WebhookRepository) CountByChannelID(ctx context.Context, channelID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM webhooks WHERE channel_id = $1`
	var count int
	err := r.db.GetContext(ctx, &count, query, channelID)
	return count, err
}

// GetByToken retrieves a webhook by its token (for execution)
func (r *WebhookRepository) GetByToken(ctx context.Context, token string) (*models.Webhook, error) {
	query := `
		SELECT id, type, server_id, channel_id, creator_id, name, avatar, token,
			   application_id, source_server_id, source_channel_id, created_at
		FROM webhooks
		WHERE token = $1
	`
	var webhook models.Webhook
	err := r.db.GetContext(ctx, &webhook, query, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &webhook, nil
}
