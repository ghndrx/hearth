package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"hearth/internal/models"
)

// StageRepository handles stage channel database operations
type StageRepository struct {
	db *sqlx.DB
}

// NewStageRepository creates a new stage repository
func NewStageRepository(db *sqlx.DB) *StageRepository {
	return &StageRepository{db: db}
}

// Create creates a new stage
func (r *StageRepository) Create(ctx context.Context, stage *models.Stage) error {
	query := `
		INSERT INTO stages (id, channel_id, topic, description, status, host_user_id, discovery_disabled, request_to_speak, moderator_only, max_speakers, created_at, started_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err := r.db.ExecContext(ctx, query,
		stage.ID, stage.ChannelID, stage.Topic, stage.Description, stage.Status,
		stage.HostUserID, stage.DiscoveryDisabled, stage.RequestToSpeak,
		stage.ModeratorOnly, stage.MaxSpeakers, stage.CreatedAt, stage.StartedAt, stage.UpdatedAt,
	)
	return err
}

// GetByID retrieves a stage by its ID
func (r *StageRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Stage, error) {
	var stage models.Stage
	query := `SELECT * FROM stages WHERE id = $1`
	err := r.db.GetContext(ctx, &stage, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &stage, nil
}

// GetByChannelID retrieves the active or scheduled stage for a channel
func (r *StageRepository) GetByChannelID(ctx context.Context, channelID uuid.UUID) (*models.Stage, error) {
	var stage models.Stage
	query := `SELECT * FROM stages WHERE channel_id = $1 AND status != 4 ORDER BY created_at DESC LIMIT 1`
	err := r.db.GetContext(ctx, &stage, query, channelID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &stage, nil
}

// Update updates a stage
func (r *StageRepository) Update(ctx context.Context, stage *models.Stage) error {
	query := `
		UPDATE stages SET
			topic = $2,
			description = $3,
			status = $4,
			discovery_disabled = $5,
			request_to_speak = $6,
			moderator_only = $7,
			max_speakers = $8,
			started_at = $9,
			ended_at = $10,
			updated_at = $11
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query,
		stage.ID, stage.Topic, stage.Description, stage.Status,
		stage.DiscoveryDisabled, stage.RequestToSpeak, stage.ModeratorOnly,
		stage.MaxSpeakers, stage.StartedAt, stage.EndedAt, stage.UpdatedAt,
	)
	return err
}

// Delete deletes a stage (marks as ended)
func (r *StageRepository) Delete(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	query := `UPDATE stages SET status = 4, ended_at = $2, updated_at = $2 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, now)
	return err
}

// AddParticipant adds a participant to a stage
func (r *StageRepository) AddParticipant(ctx context.Context, p *models.StageParticipant) error {
	query := `
		INSERT INTO stage_participants (stage_id, user_id, role, joined_at, is_muted, is_deafened, requested_at, approved_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (stage_id, user_id) DO UPDATE SET
			role = EXCLUDED.role,
			joined_at = EXCLUDED.joined_at,
			is_muted = EXCLUDED.is_muted,
			is_deafened = EXCLUDED.is_deafened
	`
	_, err := r.db.ExecContext(ctx, query,
		p.StageID, p.UserID, p.Role, p.JoinedAt, p.IsMuted, p.IsDeafened, p.RequestedAt, p.ApprovedAt,
	)
	return err
}

// GetParticipant retrieves a participant from a stage
func (r *StageRepository) GetParticipant(ctx context.Context, stageID, userID uuid.UUID) (*models.StageParticipant, error) {
	var p models.StageParticipant
	query := `SELECT * FROM stage_participants WHERE stage_id = $1 AND user_id = $2`
	err := r.db.GetContext(ctx, &p, query, stageID, userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// UpdateParticipant updates a participant's role and state
func (r *StageRepository) UpdateParticipant(ctx context.Context, p *models.StageParticipant) error {
	query := `
		UPDATE stage_participants SET
			role = $3,
			is_muted = $4,
			is_deafened = $5,
			requested_at = $6,
			approved_at = $7
		WHERE stage_id = $1 AND user_id = $2
	`
	_, err := r.db.ExecContext(ctx, query,
		p.StageID, p.UserID, p.Role, p.IsMuted, p.IsDeafened, p.RequestedAt, p.ApprovedAt,
	)
	return err
}

// RemoveParticipant removes a participant from a stage
func (r *StageRepository) RemoveParticipant(ctx context.Context, stageID, userID uuid.UUID) error {
	query := `DELETE FROM stage_participants WHERE stage_id = $1 AND user_id = $2`
	_, err := r.db.ExecContext(ctx, query, stageID, userID)
	return err
}

// ListParticipants retrieves all participants for a stage
func (r *StageRepository) ListParticipants(ctx context.Context, stageID uuid.UUID) ([]models.StageParticipant, error) {
	var participants []models.StageParticipant
	query := `SELECT * FROM stage_participants WHERE stage_id = $1 ORDER BY joined_at`
	err := r.db.SelectContext(ctx, &participants, query, stageID)
	if err != nil {
		return nil, err
	}
	return participants, nil
}

// ListParticipantsByRole retrieves participants for a stage filtered by role
func (r *StageRepository) ListParticipantsByRole(ctx context.Context, stageID uuid.UUID, role models.StageRole) ([]models.StageParticipant, error) {
	var participants []models.StageParticipant
	query := `SELECT * FROM stage_participants WHERE stage_id = $1 AND role = $2 ORDER BY joined_at`
	err := r.db.SelectContext(ctx, &participants, query, stageID, role)
	if err != nil {
		return nil, err
	}
	return participants, nil
}

// ListPendingRequests retrieves participants with pending speaker requests
func (r *StageRepository) ListPendingRequests(ctx context.Context, stageID uuid.UUID) ([]models.StageParticipant, error) {
	var participants []models.StageParticipant
	query := `SELECT * FROM stage_participants WHERE stage_id = $1 AND requested_at IS NOT NULL AND approved_at IS NULL ORDER BY requested_at`
	err := r.db.SelectContext(ctx, &participants, query, stageID)
	if err != nil {
		return nil, err
	}
	return participants, nil
}

// CountParticipantsByRole counts participants by role for a stage
func (r *StageRepository) CountParticipantsByRole(ctx context.Context, stageID uuid.UUID) (speakers int, audience int, pending int, err error) {
	err = r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE role IN (2, 3, 4)) as speakers,
			COUNT(*) FILTER (WHERE role = 1) as audience,
			COUNT(*) FILTER (WHERE requested_at IS NOT NULL AND approved_at IS NULL) as pending
		FROM stage_participants WHERE stage_id = $1
	`, stageID).Scan(&speakers, &audience, &pending)
	return
}

// UpdateParticipantMute updates a participant's mute state
func (r *StageRepository) UpdateParticipantMute(ctx context.Context, stageID, userID uuid.UUID, isMuted bool) error {
	query := `UPDATE stage_participants SET is_muted = $3 WHERE stage_id = $1 AND user_id = $2`
	_, err := r.db.ExecContext(ctx, query, stageID, userID, isMuted)
	return err
}

// UpdateParticipantDeaf updates a participant's deaf state
func (r *StageRepository) UpdateParticipantDeaf(ctx context.Context, stageID, userID uuid.UUID, isDeafened bool) error {
	query := `UPDATE stage_participants SET is_deafened = $3 WHERE stage_id = $1 AND user_id = $2`
	_, err := r.db.ExecContext(ctx, query, stageID, userID, isDeafened)
	return err
}

// ApproveSpeakerRequest approves a pending speaker request
func (r *StageRepository) ApproveSpeakerRequest(ctx context.Context, stageID, userID uuid.UUID) error {
	now := time.Now()
	query := `UPDATE stage_participants SET role = 2, approved_at = $3 WHERE stage_id = $1 AND user_id = $2 AND requested_at IS NOT NULL`
	_, err := r.db.ExecContext(ctx, query, stageID, userID, now)
	return err
}

// ClearAllParticipants removes all participants from a stage
func (r *StageRepository) ClearAllParticipants(ctx context.Context, stageID uuid.UUID) error {
	query := `DELETE FROM stage_participants WHERE stage_id = $1`
	_, err := r.db.ExecContext(ctx, query, stageID)
	return err
}
