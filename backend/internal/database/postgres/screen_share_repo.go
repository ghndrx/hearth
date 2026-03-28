package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"hearth/internal/models"
)

// ScreenShareRepository handles screen share database operations
type ScreenShareRepository struct {
	db *sqlx.DB
}

// NewScreenShareRepository creates a new screen share repository
func NewScreenShareRepository(db *sqlx.DB) *ScreenShareRepository {
	return &ScreenShareRepository{db: db}
}

// CreateSession creates a new stream session
func (r *ScreenShareRepository) CreateSession(ctx context.Context, session *models.StreamSession) error {
	query := `
		INSERT INTO stream_sessions (id, server_id, channel_id, user_id, stream_type, status, resolution, frame_rate, started_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.ExecContext(ctx, query,
		session.ID, session.ServerID, session.ChannelID, session.UserID,
		session.StreamType, session.Status, session.Resolution, session.FrameRate, session.StartedAt,
	)
	return err
}

// GetSession retrieves a stream session by ID
func (r *ScreenShareRepository) GetSession(ctx context.Context, id uuid.UUID) (*models.StreamSession, error) {
	var session models.StreamSession
	query := `SELECT * FROM stream_sessions WHERE id = $1`
	err := r.db.GetContext(ctx, &session, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// GetActiveSessionByChannel retrieves the active stream session for a channel
func (r *ScreenShareRepository) GetActiveSessionByChannel(ctx context.Context, channelID uuid.UUID) (*models.StreamSession, error) {
	var session models.StreamSession
	query := `SELECT * FROM stream_sessions WHERE channel_id = $1 AND status = 1`
	err := r.db.GetContext(ctx, &session, query, channelID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// UpdateSession updates a stream session
func (r *ScreenShareRepository) UpdateSession(ctx context.Context, session *models.StreamSession) error {
	query := `
		UPDATE stream_sessions SET
			stream_type = $2,
			status = $3,
			resolution = $4,
			frame_rate = $5,
			ended_at = $6
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query,
		session.ID, session.StreamType, session.Status, session.Resolution, session.FrameRate, session.EndedAt,
	)
	return err
}

// EndSession marks a stream session as ended
func (r *ScreenShareRepository) EndSession(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	query := `UPDATE stream_sessions SET status = 2, ended_at = $2 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, now)
	return err
}

// ListActiveSessions returns all active stream sessions
func (r *ScreenShareRepository) ListActiveSessions(ctx context.Context) ([]*models.StreamSession, error) {
	var sessions []*models.StreamSession
	query := `SELECT * FROM stream_sessions WHERE status = 1 ORDER BY started_at DESC`
	err := r.db.SelectContext(ctx, &sessions, query)
	return sessions, err
}

// ListActiveSessionsByServer returns all active stream sessions for a server
func (r *ScreenShareRepository) ListActiveSessionsByServer(ctx context.Context, serverID uuid.UUID) ([]*models.StreamSession, error) {
	var sessions []*models.StreamSession
	query := `SELECT * FROM stream_sessions WHERE server_id = $1 AND status = 1 ORDER BY started_at DESC`
	err := r.db.SelectContext(ctx, &sessions, query, serverID)
	return sessions, err
}

// AddViewer adds a viewer to a stream session
func (r *ScreenShareRepository) AddViewer(ctx context.Context, sessionID, userID uuid.UUID) error {
	query := `
		INSERT INTO stream_viewers (session_id, user_id, joined_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (session_id, user_id) DO NOTHING
	`
	_, err := r.db.ExecContext(ctx, query, sessionID, userID, time.Now())
	return err
}

// RemoveViewer removes a viewer from a stream session
func (r *ScreenShareRepository) RemoveViewer(ctx context.Context, sessionID, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM stream_viewers WHERE session_id = $1 AND user_id = $2`,
		sessionID, userID,
	)
	return err
}

// GetViewerCount returns the number of viewers for a stream session
func (r *ScreenShareRepository) GetViewerCount(ctx context.Context, sessionID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM stream_viewers WHERE session_id = $1`
	err := r.db.GetContext(ctx, &count, query, sessionID)
	return count, err
}

// GetViewers returns all viewers for a stream session
func (r *ScreenShareRepository) GetViewers(ctx context.Context, sessionID uuid.UUID) ([]models.StreamViewer, error) {
	var viewers []models.StreamViewer
	query := `SELECT session_id, user_id, joined_at FROM stream_viewers WHERE session_id = $1`
	err := r.db.SelectContext(ctx, &viewers, query, sessionID)
	return viewers, err
}

// IsViewing checks if a user is viewing a specific stream
func (r *ScreenShareRepository) IsViewing(ctx context.Context, sessionID, userID uuid.UUID) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM stream_viewers WHERE session_id = $1 AND user_id = $2)`
	err := r.db.GetContext(ctx, &exists, query, sessionID, userID)
	return exists, err
}

// GetActiveStreamForUser returns the active stream session for a user (if they're streaming)
func (r *ScreenShareRepository) GetActiveStreamForUser(ctx context.Context, userID uuid.UUID) (*models.StreamSession, error) {
	var session models.StreamSession
	query := `SELECT * FROM stream_sessions WHERE user_id = $1 AND status = 1 LIMIT 1`
	err := r.db.GetContext(ctx, &session, query, userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &session, nil
}
