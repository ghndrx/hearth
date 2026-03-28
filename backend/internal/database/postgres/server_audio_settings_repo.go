package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"hearth/internal/models"
)

// ServerAudioSettingsRepository handles server audio settings data access
type ServerAudioSettingsRepository struct {
	db *sqlx.DB
}

// NewServerAudioSettingsRepository creates a new server audio settings repository
func NewServerAudioSettingsRepository(db *sqlx.DB) *ServerAudioSettingsRepository {
	return &ServerAudioSettingsRepository{db: db}
}

// Get retrieves audio settings for a user in a specific server
func (r *ServerAudioSettingsRepository) Get(ctx context.Context, userID, serverID uuid.UUID) (*models.ServerAudioSettings, error) {
	var settings models.ServerAudioSettings
	query := `
		SELECT user_id, server_id, input_device_id, output_device_id,
			input_volume, output_volume, push_to_talk_enabled, push_to_talk_key, updated_at
		FROM server_audio_settings
		WHERE user_id = $1 AND server_id = $2
	`
	err := r.db.GetContext(ctx, &settings, query, userID, serverID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &settings, err
}

// GetAllForUser retrieves audio settings for all servers for a user
func (r *ServerAudioSettingsRepository) GetAllForUser(ctx context.Context, userID uuid.UUID) ([]*models.ServerAudioSettings, error) {
	var settings []*models.ServerAudioSettings
	query := `
		SELECT user_id, server_id, input_device_id, output_device_id,
			input_volume, output_volume, push_to_talk_enabled, push_to_talk_key, updated_at
		FROM server_audio_settings
		WHERE user_id = $1
		ORDER BY updated_at DESC
	`
	err := r.db.SelectContext(ctx, &settings, query, userID)
	return settings, err
}

// Upsert creates or updates audio settings for a user in a server
func (r *ServerAudioSettingsRepository) Upsert(ctx context.Context, settings *models.ServerAudioSettings) error {
	settings.UpdatedAt = time.Now()
	query := `
		INSERT INTO server_audio_settings (
			user_id, server_id, input_device_id, output_device_id,
			input_volume, output_volume, push_to_talk_enabled, push_to_talk_key, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (user_id, server_id) DO UPDATE SET
			input_device_id = EXCLUDED.input_device_id,
			output_device_id = EXCLUDED.output_device_id,
			input_volume = EXCLUDED.input_volume,
			output_volume = EXCLUDED.output_volume,
			push_to_talk_enabled = EXCLUDED.push_to_talk_enabled,
			push_to_talk_key = EXCLUDED.push_to_talk_key,
			updated_at = EXCLUDED.updated_at
	`
	_, err := r.db.ExecContext(ctx, query,
		settings.UserID, settings.ServerID, settings.InputDeviceID, settings.OutputDeviceID,
		settings.InputVolume, settings.OutputVolume, settings.PushToTalkEnabled, settings.PushToTalkKey,
		settings.UpdatedAt,
	)
	return err
}

// Delete deletes audio settings for a user in a server
func (r *ServerAudioSettingsRepository) Delete(ctx context.Context, userID, serverID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM server_audio_settings WHERE user_id = $1 AND server_id = $2`,
		userID, serverID,
	)
	return err
}
