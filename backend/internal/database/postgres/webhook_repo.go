package postgres

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"hearth/internal/models"
)

type WebhookRepository struct {
	db *sqlx.DB
}

func NewWebhookRepository(db *sqlx.DB) *WebhookRepository {
	return &WebhookRepository{db: db}
}

func (r *WebhookRepository) Create(ctx context.Context, webhook *models.Webhook) error {
	query := `
		INSERT INTO webhooks (id, type, server_id, channel_id, creator_id, name, avatar, token, application_id, source_server_id, source_channel_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := r.db.ExecContext(ctx, query,
		webhook.ID, webhook.Type, webhook.ServerID, webhook.ChannelID, webhook.CreatorID,
		webhook.Name, webhook.Avatar, webhook.Token, webhook.ApplicationID,
		webhook.SourceServerID, webhook.SourceChannelID, webhook.CreatedAt,
	)
	return err
}

func (r *WebhookRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Webhook, error) {
	var webhook models.Webhook
	query := `SELECT * FROM webhooks WHERE id = $1`
	err := r.db.GetContext(ctx, &webhook, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &webhook, err
}

func (r *WebhookRepository) GetByChannelID(ctx context.Context, channelID uuid.UUID) ([]*models.Webhook, error) {
	var webhooks []*models.Webhook
	query := `SELECT * FROM webhooks WHERE channel_id = $1 ORDER BY created_at DESC`
	err := r.db.SelectContext(ctx, &webhooks, query, channelID)
	return webhooks, err
}

func (r *WebhookRepository) GetByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.Webhook, error) {
	var webhooks []*models.Webhook
	query := `SELECT * FROM webhooks WHERE server_id = $1 ORDER BY created_at DESC`
	err := r.db.SelectContext(ctx, &webhooks, query, serverID)
	return webhooks, err
}

func (r *WebhookRepository) Update(ctx context.Context, webhook *models.Webhook) error {
	query := `
		UPDATE webhooks 
		SET name = $1, avatar = $2, channel_id = $3
		WHERE id = $4
	`
	_, err := r.db.ExecContext(ctx, query, webhook.Name, webhook.Avatar, webhook.ChannelID, webhook.ID)
	return err
}

func (r *WebhookRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM webhooks WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *WebhookRepository) CountByChannelID(ctx context.Context, channelID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM webhooks WHERE channel_id = $1`
	err := r.db.GetContext(ctx, &count, query, channelID)
	return count, err
}
