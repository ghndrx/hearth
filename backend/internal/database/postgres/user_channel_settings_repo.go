package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"hearth/internal/models"
)

// UserChannelSettingsRepository handles user channel settings data access
type UserChannelSettingsRepository struct {
	db *sqlx.DB
}

// NewUserChannelSettingsRepository creates a new user channel settings repository
func NewUserChannelSettingsRepository(db *sqlx.DB) *UserChannelSettingsRepository {
	return &UserChannelSettingsRepository{db: db}
}

// Upsert creates or updates user channel settings
func (r *UserChannelSettingsRepository) Upsert(ctx context.Context, settings *models.UserChannelSettings) error {
	now := time.Now()
	settings.UpdatedAt = now

	query := `
		INSERT INTO user_channel_settings (user_id, channel_id, muted, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, channel_id) DO UPDATE
		SET muted = $3, updated_at = $5
	`
	_, err := r.db.ExecContext(ctx, query,
		settings.UserID, settings.ChannelID, settings.Muted, now, now,
	)
	return err
}

// Get retrieves user channel settings
func (r *UserChannelSettingsRepository) Get(ctx context.Context, userID, channelID uuid.UUID) (*models.UserChannelSettings, error) {
	var settings models.UserChannelSettings
	query := `SELECT user_id, channel_id, muted, created_at, updated_at FROM user_channel_settings WHERE user_id = $1 AND channel_id = $2`
	err := r.db.GetContext(ctx, &settings, query, userID, channelID)
	if err != nil {
		return nil, err
	}
	return &settings, nil
}

// IsChannelMuted checks if a specific channel is muted for a user
func (r *UserChannelSettingsRepository) IsChannelMuted(ctx context.Context, userID, channelID uuid.UUID) (bool, error) {
	var muted bool
	query := `SELECT muted FROM user_channel_settings WHERE user_id = $1 AND channel_id = $2`
	err := r.db.GetContext(ctx, &muted, query, userID, channelID)
	if err != nil {
		return false, nil // Not found means not muted
	}
	return muted, nil
}

// GetMutedChannelIDs returns all muted channel IDs for a user
func (r *UserChannelSettingsRepository) GetMutedChannelIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	query := `SELECT channel_id FROM user_channel_settings WHERE user_id = $1 AND muted = true`
	err := r.db.SelectContext(ctx, &ids, query, userID)
	if err != nil {
		return nil, err
	}
	return ids, nil
}
