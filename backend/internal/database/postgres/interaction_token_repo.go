package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"hearth/internal/services"
)

// InteractionTokenRepository handles interaction token database operations
type InteractionTokenRepository struct {
	db *sqlx.DB
}

// NewInteractionTokenRepository creates a new interaction token repository
func NewInteractionTokenRepository(db *sqlx.DB) *InteractionTokenRepository {
	return &InteractionTokenRepository{db: db}
}

// SaveToken stores an interaction token
func (r *InteractionTokenRepository) SaveToken(ctx context.Context, token *services.InteractionToken) error {
	query := `
		INSERT INTO interaction_tokens (token, interaction_id, app_id, user_id, server_id, channel_id, expires_at, used, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (token) DO UPDATE SET
			interaction_id = EXCLUDED.interaction_id,
			expires_at = EXCLUDED.expires_at,
			used = EXCLUDED.used
	`
	_, err := r.db.ExecContext(ctx, query,
		token.Token, token.InteractionID, token.AppID, token.UserID,
		token.ServerID, token.ChannelID, token.ExpiresAt, token.Used, token.CreatedAt,
	)
	return err
}

// GetToken retrieves a token
func (r *InteractionTokenRepository) GetToken(ctx context.Context, token string) (*services.InteractionToken, error) {
	query := `
		SELECT token, interaction_id, app_id, user_id, server_id, channel_id, expires_at, used, created_at
		FROM interaction_tokens WHERE token = $1
	`
	var t services.InteractionToken
	var serverID sql.NullString
	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&t.Token, &t.InteractionID, &t.AppID, &t.UserID, &serverID,
		&t.ChannelID, &t.ExpiresAt, &t.Used, &t.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if serverID.Valid {
		id, _ := uuid.Parse(serverID.String)
		t.ServerID = &id
	}
	return &t, nil
}

// MarkTokenUsed marks a token as used
func (r *InteractionTokenRepository) MarkTokenUsed(ctx context.Context, token string) error {
	query := `UPDATE interaction_tokens SET used = true WHERE token = $1`
	_, err := r.db.ExecContext(ctx, query, token)
	return err
}

// CleanupExpired removes expired tokens
func (r *InteractionTokenRepository) CleanupExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM interaction_tokens WHERE expires_at < $1`
	result, err := r.db.ExecContext(ctx, query, time.Now())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
