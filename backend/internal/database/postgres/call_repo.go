package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"

	"hearth/internal/models"
)

// CallRepository handles call-related database operations
type CallRepository struct {
	db *sql.DB
}

// NewCallRepository creates a new call repository
func NewCallRepository(db *sql.DB) *CallRepository {
	return &CallRepository{db: db}
}

// Create inserts a new call record
func (r *CallRepository) Create(ctx context.Context, call *models.Call) error {
	call.ID = uuid.New()
	call.CreatedAt = time.Now()
	if call.StartedAt.IsZero() {
		call.StartedAt = call.CreatedAt
	}
	if call.Status == "" {
		call.Status = models.CallStatusRinging
	}

	query := `
		INSERT INTO calls (id, channel_id, server_id, initiator_id, type, status, started_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.db.ExecContext(ctx, query,
		call.ID, call.ChannelID, call.ServerID, call.InitiatorID,
		call.Type, call.Status, call.StartedAt, call.CreatedAt,
	)
	return err
}

// GetByID retrieves a call by its ID
func (r *CallRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Call, error) {
	query := `SELECT id, channel_id, server_id, initiator_id, type, status,
		started_at, ended_at, end_reason, created_at
		FROM calls WHERE id = $1`

	call := &models.Call{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&call.ID, &call.ChannelID, &call.ServerID, &call.InitiatorID,
		&call.Type, &call.Status, &call.StartedAt, &call.EndedAt,
		&call.EndReason, &call.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	participants, err := r.GetParticipants(ctx, id)
	if err != nil {
		return nil, err
	}
	call.Participants = participants

	return call, nil
}

// GetActiveByChannel returns active (non-ended) calls for a channel
func (r *CallRepository) GetActiveByChannel(ctx context.Context, channelID uuid.UUID) ([]*models.Call, error) {
	query := `SELECT id, channel_id, server_id, initiator_id, type, status,
		started_at, ended_at, end_reason, created_at
		FROM calls WHERE channel_id = $1 AND status != 'ended'
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, channelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var calls []*models.Call
	for rows.Next() {
		call := &models.Call{}
		if err := rows.Scan(
			&call.ID, &call.ChannelID, &call.ServerID, &call.InitiatorID,
			&call.Type, &call.Status, &call.StartedAt, &call.EndedAt,
			&call.EndReason, &call.CreatedAt,
		); err != nil {
			return nil, err
		}
		calls = append(calls, call)
	}
	return calls, rows.Err()
}

// UpdateStatus updates the call status
func (r *CallRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status models.CallStatus, endReason string) error {
	if status == models.CallStatusEnded {
		now := time.Now()
		query := `UPDATE calls SET status = $1, ended_at = $2, end_reason = $3 WHERE id = $4`
		_, err := r.db.ExecContext(ctx, query, status, now, endReason, id)
		return err
	}

	query := `UPDATE calls SET status = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, status, id)
	return err
}

// AddParticipant adds a participant to a call
func (r *CallRepository) AddParticipant(ctx context.Context, participant *models.CallParticipant) error {
	participant.ID = uuid.New()
	participant.JoinedAt = time.Now()

	query := `
		INSERT INTO call_participants (id, call_id, user_id, joined_at, is_muted, is_video_on)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (call_id, user_id) DO UPDATE SET
			left_at = NULL, joined_at = EXCLUDED.joined_at`

	_, err := r.db.ExecContext(ctx, query,
		participant.ID, participant.CallID, participant.UserID,
		participant.JoinedAt, participant.IsMuted, participant.IsVideoOn,
	)
	return err
}

// RemoveParticipant marks a participant as having left
func (r *CallRepository) RemoveParticipant(ctx context.Context, callID, userID uuid.UUID) error {
	now := time.Now()
	query := `UPDATE call_participants SET left_at = $1 WHERE call_id = $2 AND user_id = $3 AND left_at IS NULL`
	_, err := r.db.ExecContext(ctx, query, now, callID, userID)
	return err
}

// GetParticipants returns all participants for a call
func (r *CallRepository) GetParticipants(ctx context.Context, callID uuid.UUID) ([]models.CallParticipant, error) {
	query := `
		SELECT cp.id, cp.call_id, cp.user_id, cp.joined_at, cp.left_at,
			cp.is_muted, cp.is_video_on,
			u.username, u.display_name, u.avatar
		FROM call_participants cp
		JOIN users u ON u.id = cp.user_id
		WHERE cp.call_id = $1
		ORDER BY cp.joined_at ASC`

	rows, err := r.db.QueryContext(ctx, query, callID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var participants []models.CallParticipant
	for rows.Next() {
		p := models.CallParticipant{}
		if err := rows.Scan(
			&p.ID, &p.CallID, &p.UserID, &p.JoinedAt, &p.LeftAt,
			&p.IsMuted, &p.IsVideoOn,
			&p.Username, &p.DisplayName, &p.Avatar,
		); err != nil {
			return nil, err
		}
		participants = append(participants, p)
	}
	return participants, rows.Err()
}

// GetActiveParticipantCount returns the count of active participants in a call
func (r *CallRepository) GetActiveParticipantCount(ctx context.Context, callID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM call_participants WHERE call_id = $1 AND left_at IS NULL`
	var count int
	err := r.db.QueryRowContext(ctx, query, callID).Scan(&count)
	return count, err
}

// CreateSession creates a new call session record
func (r *CallRepository) CreateSession(ctx context.Context, session *models.CallSession) error {
	session.ID = uuid.New()
	session.ConnectedAt = time.Now()
	if session.ConnectionType == "" {
		session.ConnectionType = "peer"
	}

	query := `
		INSERT INTO call_sessions (id, call_id, user_id, session_id, peer_id, connected_at, connection_type)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.db.ExecContext(ctx, query,
		session.ID, session.CallID, session.UserID, session.SessionID,
		session.PeerID, session.ConnectedAt, session.ConnectionType,
	)
	return err
}

// EndSession marks a session as disconnected
func (r *CallRepository) EndSession(ctx context.Context, sessionID uuid.UUID) error {
	now := time.Now()
	query := `UPDATE call_sessions SET disconnected_at = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, now, sessionID)
	return err
}
