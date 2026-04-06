package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"hearth/internal/models"
)

// ChannelNotificationOverrideRepository handles channel notification override data access
type ChannelNotificationOverrideRepository struct {
	db *sqlx.DB
}

// NewChannelNotificationOverrideRepository creates a new repository
func NewChannelNotificationOverrideRepository(db *sqlx.DB) *ChannelNotificationOverrideRepository {
	return &ChannelNotificationOverrideRepository{db: db}
}

// Set creates or updates a channel notification override
func (r *ChannelNotificationOverrideRepository) Set(ctx context.Context, override *models.ChannelNotificationOverride) error {
	override.ID = uuid.New()
	override.CreatedAt = time.Now()
	override.UpdatedAt = time.Now()

	query := `
		INSERT INTO channel_notification_overrides (
			id, user_id, channel_id, notification_level, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6
		)
		ON CONFLICT (user_id, channel_id)
		DO UPDATE SET
			notification_level = EXCLUDED.notification_level,
			updated_at = EXCLUDED.updated_at
		RETURNING id, created_at
	`

	var returnedID uuid.UUID
	var createdAt time.Time
	err := r.db.QueryRowContext(ctx, query,
		override.ID, override.UserID, override.ChannelID,
		override.NotificationLevel, override.CreatedAt, override.UpdatedAt,
	).Scan(&returnedID, &createdAt)

	if err != nil {
		return err
	}

	// Update the override with returned values
	override.ID = returnedID
	override.CreatedAt = createdAt
	return nil
}

// Get retrieves a channel notification override for a user and channel
func (r *ChannelNotificationOverrideRepository) Get(ctx context.Context, userID, channelID uuid.UUID) (*models.ChannelNotificationOverride, error) {
	var override models.ChannelNotificationOverride
	query := `
		SELECT id, user_id, channel_id, notification_level, created_at, updated_at
		FROM channel_notification_overrides
		WHERE user_id = $1 AND channel_id = $2
	`
	err := r.db.GetContext(ctx, &override, query, userID, channelID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &override, nil
}

// GetByUser retrieves all channel notification overrides for a user
func (r *ChannelNotificationOverrideRepository) GetByUser(ctx context.Context, userID uuid.UUID) ([]models.ChannelNotificationOverride, error) {
	var overrides []models.ChannelNotificationOverride
	query := `
		SELECT id, user_id, channel_id, notification_level, created_at, updated_at
		FROM channel_notification_overrides
		WHERE user_id = $1
		ORDER BY updated_at DESC
	`
	err := r.db.SelectContext(ctx, &overrides, query, userID)
	if err != nil {
		return nil, err
	}
	if overrides == nil {
		return []models.ChannelNotificationOverride{}, nil
	}
	return overrides, nil
}

// Delete removes a channel notification override
func (r *ChannelNotificationOverrideRepository) Delete(ctx context.Context, userID, channelID uuid.UUID) error {
	query := `DELETE FROM channel_notification_overrides WHERE user_id = $1 AND channel_id = $2`
	_, err := r.db.ExecContext(ctx, query, userID, channelID)
	return err
}

// DeleteAllForUser removes all channel notification overrides for a user
func (r *ChannelNotificationOverrideRepository) DeleteAllForUser(ctx context.Context, userID uuid.UUID) error {
	query := `DELETE FROM channel_notification_overrides WHERE user_id = $1`
	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}

// GetForChannels retrieves overrides for specific channels
func (r *ChannelNotificationOverrideRepository) GetForChannels(ctx context.Context, userID uuid.UUID, channelIDs []uuid.UUID) (map[uuid.UUID]models.ChannelNotificationLevel, error) {
	if len(channelIDs) == 0 {
		return map[uuid.UUID]models.ChannelNotificationLevel{}, nil
	}

	query, args, err := sqlx.In(`
		SELECT channel_id, notification_level
		FROM channel_notification_overrides
		WHERE user_id = ? AND channel_id IN (?)
	`, userID, channelIDs)
	if err != nil {
		return nil, err
	}

	query = r.db.Rebind(query)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[uuid.UUID]models.ChannelNotificationLevel)
	for rows.Next() {
		var channelID uuid.UUID
		var level models.ChannelNotificationLevel
		if err := rows.Scan(&channelID, &level); err != nil {
			return nil, err
		}
		result[channelID] = level
	}

	return result, rows.Err()
}
