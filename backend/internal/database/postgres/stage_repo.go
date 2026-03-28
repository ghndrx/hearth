package postgres

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"hearth/internal/models"
)

// StageRepository handles stage instance data access
type StageRepository struct {
	db *sqlx.DB
}

// NewStageRepository creates a new stage repository
func NewStageRepository(db *sqlx.DB) *StageRepository {
	return &StageRepository{db: db}
}

// Create inserts a new stage instance
func (r *StageRepository) Create(ctx context.Context, stage *models.StageInstance) error {
	query := `INSERT INTO stage_instances (id, channel_id, server_id, topic, privacy_level, started_by, speaker_count, audience_count, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := r.db.ExecContext(ctx, query,
		stage.ID, stage.ChannelID, stage.ServerID, stage.Topic,
		stage.PrivacyLevel, stage.StartedBy, stage.SpeakerCount, stage.AudienceCount,
		stage.CreatedAt,
	)
	return err
}

// GetByID retrieves a stage instance by ID
func (r *StageRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.StageInstance, error) {
	var stage models.StageInstance
	err := r.db.GetContext(ctx, &stage,
		`SELECT * FROM stage_instances WHERE id = $1`, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &stage, err
}

// GetActiveByChannel retrieves the active stage for a channel
func (r *StageRepository) GetActiveByChannel(ctx context.Context, channelID uuid.UUID) (*models.StageInstance, error) {
	var stage models.StageInstance
	err := r.db.GetContext(ctx, &stage,
		`SELECT * FROM stage_instances WHERE channel_id = $1 AND ended_at IS NULL`, channelID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &stage, err
}

// End marks a stage as ended
func (r *StageRepository) End(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE stage_instances SET ended_at = NOW() WHERE id = $1`, id)
	return err
}

// Update updates a stage instance
func (r *StageRepository) Update(ctx context.Context, stage *models.StageInstance) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE stage_instances SET topic = $1, privacy_level = $2, speaker_count = $3, audience_count = $4 WHERE id = $5`,
		stage.Topic, stage.PrivacyLevel, stage.SpeakerCount, stage.AudienceCount, stage.ID)
	return err
}

// AddParticipant adds a participant to a stage
func (r *StageRepository) AddParticipant(ctx context.Context, stageID, userID uuid.UUID, role models.StageParticipantRole) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO stage_participants (stage_id, user_id, role) VALUES ($1, $2, $3)
		ON CONFLICT (stage_id, user_id) DO UPDATE SET role = $3`,
		stageID, userID, role)
	return err
}

// RemoveParticipant removes a participant from a stage
func (r *StageRepository) RemoveParticipant(ctx context.Context, stageID, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM stage_participants WHERE stage_id = $1 AND user_id = $2`,
		stageID, userID)
	return err
}

// GetParticipant gets a single participant
func (r *StageRepository) GetParticipant(ctx context.Context, stageID, userID uuid.UUID) (*models.StageParticipant, error) {
	var p models.StageParticipant
	err := r.db.GetContext(ctx, &p,
		`SELECT * FROM stage_participants WHERE stage_id = $1 AND user_id = $2`,
		stageID, userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &p, err
}

// GetParticipants retrieves all participants for a stage
func (r *StageRepository) GetParticipants(ctx context.Context, stageID uuid.UUID) ([]models.StageParticipant, error) {
	var participants []models.StageParticipant
	err := r.db.SelectContext(ctx, &participants,
		`SELECT * FROM stage_participants WHERE stage_id = $1 ORDER BY joined_at`, stageID)
	if err != nil {
		return nil, err
	}
	return participants, nil
}

// UpdateParticipantRole updates a participant's role
func (r *StageRepository) UpdateParticipantRole(ctx context.Context, stageID, userID uuid.UUID, role models.StageParticipantRole) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE stage_participants SET role = $1 WHERE stage_id = $2 AND user_id = $3`,
		role, stageID, userID)
	return err
}

// RemoveAllParticipants removes all participants from a stage
func (r *StageRepository) RemoveAllParticipants(ctx context.Context, stageID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM stage_participants WHERE stage_id = $1`, stageID)
	return err
}

// CountParticipants returns speaker and audience counts
func (r *StageRepository) CountParticipants(ctx context.Context, stageID uuid.UUID) (speakers int, audience int, err error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT role, COUNT(*) FROM stage_participants WHERE stage_id = $1 GROUP BY role`, stageID)
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var role string
		var count int
		if err := rows.Scan(&role, &count); err != nil {
			return 0, 0, err
		}
		if role == string(models.StageRoleSpeaker) {
			speakers = count
		} else {
			audience = count
		}
	}
	return speakers, audience, rows.Err()
}
