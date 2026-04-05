package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"hearth/internal/models"
)

// VoiceActivityRepository handles voice activity database operations
type VoiceActivityRepository struct {
	db *sqlx.DB
}

// NewVoiceActivityRepository creates a new voice activity repository
func NewVoiceActivityRepository(db *sqlx.DB) *VoiceActivityRepository {
	return &VoiceActivityRepository{db: db}
}

// Create creates a new voice activity
func (r *VoiceActivityRepository) Create(ctx context.Context, activity *models.VoiceActivity) error {
	activity.ID = uuid.New()
	activity.CreatedAt = time.Now()
	activity.UpdatedAt = activity.CreatedAt

	if activity.Metadata == nil {
		activity.Metadata = json.RawMessage("{}")
	}

	query := `
		INSERT INTO voice_activities (id, channel_id, server_id, creator_id, activity_type, status, max_participants, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.db.ExecContext(ctx, query,
		activity.ID, activity.ChannelID, activity.ServerID, activity.CreatorID,
		activity.ActivityType, activity.Status, activity.MaxParticipants,
		activity.Metadata, activity.CreatedAt, activity.UpdatedAt,
	)
	return err
}

// GetByID retrieves a voice activity by ID
func (r *VoiceActivityRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.VoiceActivity, error) {
	var activity models.VoiceActivity
	query := `SELECT id, channel_id, server_id, creator_id, activity_type, status, max_participants, metadata, created_at, updated_at, ended_at FROM voice_activities WHERE id = $1`
	err := r.db.QueryRowxContext(ctx, query, id).StructScan(&activity)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &activity, nil
}

// GetActiveByChannel retrieves the active activity for a channel
func (r *VoiceActivityRepository) GetActiveByChannel(ctx context.Context, channelID uuid.UUID) (*models.VoiceActivity, error) {
	var activity models.VoiceActivity
	query := `SELECT id, channel_id, server_id, creator_id, activity_type, status, max_participants, metadata, created_at, updated_at, ended_at FROM voice_activities WHERE channel_id = $1 AND status = 'active' LIMIT 1`
	err := r.db.QueryRowxContext(ctx, query, channelID).StructScan(&activity)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &activity, nil
}

// EndActivity marks an activity as finished or cancelled
func (r *VoiceActivityRepository) EndActivity(ctx context.Context, id uuid.UUID, status models.VoiceActivityStatus) error {
	now := time.Now()
	query := `UPDATE voice_activities SET status = $1, ended_at = $2, updated_at = $2 WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, status, now, id)
	return err
}

// AddParticipant adds a participant to an activity
func (r *VoiceActivityRepository) AddParticipant(ctx context.Context, activityID, userID uuid.UUID) error {
	query := `
		INSERT INTO voice_activity_participants (id, activity_id, user_id, joined_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (activity_id, user_id) DO UPDATE SET left_at = NULL, joined_at = $4
	`
	_, err := r.db.ExecContext(ctx, query, uuid.New(), activityID, userID, time.Now())
	return err
}

// RemoveParticipant marks a participant as left
func (r *VoiceActivityRepository) RemoveParticipant(ctx context.Context, activityID, userID uuid.UUID) error {
	query := `UPDATE voice_activity_participants SET left_at = $1 WHERE activity_id = $2 AND user_id = $3 AND left_at IS NULL`
	_, err := r.db.ExecContext(ctx, query, time.Now(), activityID, userID)
	return err
}

// GetParticipants retrieves active participants for an activity
func (r *VoiceActivityRepository) GetParticipants(ctx context.Context, activityID uuid.UUID) ([]models.VoiceActivityParticipantInfo, error) {
	query := `
		SELECT p.user_id, u.username, u.display_name, u.avatar, p.joined_at
		FROM voice_activity_participants p
		JOIN users u ON u.id = p.user_id
		WHERE p.activity_id = $1 AND p.left_at IS NULL
		ORDER BY p.joined_at ASC
	`
	var participants []models.VoiceActivityParticipantInfo
	err := r.db.SelectContext(ctx, &participants, query, activityID)
	if err != nil {
		return nil, err
	}
	return participants, nil
}

// GetParticipantCount returns the number of active participants
func (r *VoiceActivityRepository) GetParticipantCount(ctx context.Context, activityID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM voice_activity_participants WHERE activity_id = $1 AND left_at IS NULL`
	err := r.db.QueryRowContext(ctx, query, activityID).Scan(&count)
	return count, err
}

// SaveGameState creates or updates the game state for an activity
func (r *VoiceActivityRepository) SaveGameState(ctx context.Context, activityID uuid.UUID, state json.RawMessage) (int, error) {
	var version int
	query := `
		INSERT INTO voice_activity_game_states (id, activity_id, state, version, updated_at)
		VALUES ($1, $2, $3, 1, $4)
		ON CONFLICT (activity_id) DO UPDATE SET
			state = EXCLUDED.state,
			version = voice_activity_game_states.version + 1,
			updated_at = EXCLUDED.updated_at
		RETURNING version
	`
	err := r.db.QueryRowContext(ctx, query, uuid.New(), activityID, state, time.Now()).Scan(&version)
	return version, err
}

// GetGameState retrieves the current game state for an activity
func (r *VoiceActivityRepository) GetGameState(ctx context.Context, activityID uuid.UUID) (*models.VoiceActivityGameState, error) {
	var gs models.VoiceActivityGameState
	query := `SELECT id, activity_id, state, version, updated_at FROM voice_activity_game_states WHERE activity_id = $1`
	err := r.db.QueryRowxContext(ctx, query, activityID).StructScan(&gs)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &gs, nil
}

// GetActiveActivitiesByServer lists all active activities in a server
func (r *VoiceActivityRepository) GetActiveActivitiesByServer(ctx context.Context, serverID uuid.UUID) ([]models.VoiceActivity, error) {
	query := `
		SELECT id, channel_id, server_id, creator_id, activity_type, status, max_participants, metadata, created_at, updated_at, ended_at
		FROM voice_activities
		WHERE server_id = $1 AND status = 'active'
		ORDER BY created_at DESC
	`
	var activities []models.VoiceActivity
	err := r.db.SelectContext(ctx, &activities, query, serverID)
	if err != nil {
		return nil, err
	}
	return activities, nil
}

// IsUserInActivity checks if a user is currently in any active activity
func (r *VoiceActivityRepository) IsUserInActivity(ctx context.Context, userID uuid.UUID) (*uuid.UUID, error) {
	var activityID uuid.UUID
	query := `
		SELECT p.activity_id FROM voice_activity_participants p
		JOIN voice_activities a ON a.id = p.activity_id
		WHERE p.user_id = $1 AND p.left_at IS NULL AND a.status = 'active'
		LIMIT 1
	`
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&activityID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &activityID, nil
}

// UpdateMetadata updates the activity metadata
func (r *VoiceActivityRepository) UpdateMetadata(ctx context.Context, activityID uuid.UUID, metadata json.RawMessage) error {
	query := `UPDATE voice_activities SET metadata = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, metadata, time.Now(), activityID)
	if err != nil {
		return fmt.Errorf("failed to update activity metadata: %w", err)
	}
	return nil
}
