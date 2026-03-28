package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// VoiceState represents a user's voice connection state
type VoiceState struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	ChannelID    uuid.UUID `json:"channel_id"`
	ServerID     uuid.UUID `json:"server_id"`
	SelfMuted    bool      `json:"self_muted"`
	SelfDeafened bool      `json:"self_deafened"`
	SelfVideo    bool      `json:"self_video"`
	SelfStream   bool      `json:"self_stream"`
	Muted        bool      `json:"muted"`
	Deafened     bool      `json:"deafened"`
	SessionID    string    `json:"session_id"`
	ConnectedAt  time.Time `json:"connected_at"`
}

// VoiceStateWithUser includes user information for display
type VoiceStateWithUser struct {
	VoiceState
	Username    string  `json:"username"`
	DisplayName *string `json:"display_name,omitempty"`
	Avatar      *string `json:"avatar,omitempty"`
}

// VoiceStateRepository handles voice state database operations
type VoiceStateRepository struct {
	db *sql.DB
}

// NewVoiceStateRepository creates a new voice state repository
func NewVoiceStateRepository(db *sql.DB) *VoiceStateRepository {
	return &VoiceStateRepository{db: db}
}

// JoinChannel adds or updates a user's voice state for a channel
func (r *VoiceStateRepository) JoinChannel(ctx context.Context, userID, channelID, serverID uuid.UUID, sessionID string) (*VoiceState, error) {
	state := &VoiceState{
		ID:          uuid.New(),
		UserID:      userID,
		ChannelID:   channelID,
		ServerID:    serverID,
		SessionID:   sessionID,
		ConnectedAt: time.Now(),
	}

	query := `
		INSERT INTO voice_states (id, user_id, channel_id, server_id, session_id, connected_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, server_id) DO UPDATE SET
			channel_id = EXCLUDED.channel_id,
			session_id = EXCLUDED.session_id,
			connected_at = EXCLUDED.connected_at,
			self_muted = FALSE,
			self_deafened = FALSE,
			self_video = FALSE,
			self_stream = FALSE
		RETURNING id, self_muted, self_deafened, self_video, self_stream, muted, deafened, connected_at
	`

	err := r.db.QueryRowContext(ctx, query,
		state.ID, state.UserID, state.ChannelID, state.ServerID, state.SessionID, state.ConnectedAt,
	).Scan(
		&state.ID, &state.SelfMuted, &state.SelfDeafened,
		&state.SelfVideo, &state.SelfStream, &state.Muted, &state.Deafened, &state.ConnectedAt,
	)

	if err != nil {
		return nil, err
	}

	return state, nil
}

// LeaveChannel removes a user's voice state
func (r *VoiceStateRepository) LeaveChannel(ctx context.Context, userID, serverID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM voice_states WHERE user_id = $1 AND server_id = $2`,
		userID, serverID,
	)
	return err
}

// LeaveBySession removes voice states for a specific session (for disconnect cleanup)
func (r *VoiceStateRepository) LeaveBySession(ctx context.Context, sessionID string) ([]VoiceState, error) {
	// First get the states we're about to delete
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, channel_id, server_id, self_muted, self_deafened, 
		        self_video, self_stream, muted, deafened, session_id, connected_at
		 FROM voice_states WHERE session_id = $1`,
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var states []VoiceState
	for rows.Next() {
		var s VoiceState
		if err := rows.Scan(
			&s.ID, &s.UserID, &s.ChannelID, &s.ServerID,
			&s.SelfMuted, &s.SelfDeafened, &s.SelfVideo, &s.SelfStream,
			&s.Muted, &s.Deafened, &s.SessionID, &s.ConnectedAt,
		); err != nil {
			return nil, err
		}
		states = append(states, s)
	}

	// Now delete them
	_, err = r.db.ExecContext(ctx,
		`DELETE FROM voice_states WHERE session_id = $1`,
		sessionID,
	)
	if err != nil {
		return nil, err
	}

	return states, nil
}

// UpdateSelfState updates a user's self-controlled voice state
func (r *VoiceStateRepository) UpdateSelfState(ctx context.Context, userID, serverID uuid.UUID, selfMuted, selfDeafened, selfVideo, selfStream bool) (*VoiceState, error) {
	var state VoiceState

	query := `
		UPDATE voice_states 
		SET self_muted = $3, self_deafened = $4, self_video = $5, self_stream = $6
		WHERE user_id = $1 AND server_id = $2
		RETURNING id, user_id, channel_id, server_id, self_muted, self_deafened, 
		          self_video, self_stream, muted, deafened, session_id, connected_at
	`

	err := r.db.QueryRowContext(ctx, query,
		userID, serverID, selfMuted, selfDeafened, selfVideo, selfStream,
	).Scan(
		&state.ID, &state.UserID, &state.ChannelID, &state.ServerID,
		&state.SelfMuted, &state.SelfDeafened, &state.SelfVideo, &state.SelfStream,
		&state.Muted, &state.Deafened, &state.SessionID, &state.ConnectedAt,
	)

	if err != nil {
		return nil, err
	}

	return &state, nil
}

// UpdateServerState updates a user's server-controlled voice state (moderator action)
func (r *VoiceStateRepository) UpdateServerState(ctx context.Context, userID, serverID uuid.UUID, muted, deafened bool) (*VoiceState, error) {
	var state VoiceState

	query := `
		UPDATE voice_states 
		SET muted = $3, deafened = $4
		WHERE user_id = $1 AND server_id = $2
		RETURNING id, user_id, channel_id, server_id, self_muted, self_deafened, 
		          self_video, self_stream, muted, deafened, session_id, connected_at
	`

	err := r.db.QueryRowContext(ctx, query,
		userID, serverID, muted, deafened,
	).Scan(
		&state.ID, &state.UserID, &state.ChannelID, &state.ServerID,
		&state.SelfMuted, &state.SelfDeafened, &state.SelfVideo, &state.SelfStream,
		&state.Muted, &state.Deafened, &state.SessionID, &state.ConnectedAt,
	)

	if err != nil {
		return nil, err
	}

	return &state, nil
}

// GetByChannel returns all voice states for a channel with user info
func (r *VoiceStateRepository) GetByChannel(ctx context.Context, channelID uuid.UUID) ([]VoiceStateWithUser, error) {
	query := `
		SELECT vs.id, vs.user_id, vs.channel_id, vs.server_id,
		       vs.self_muted, vs.self_deafened, vs.self_video, vs.self_stream,
		       vs.muted, vs.deafened, vs.session_id, vs.connected_at,
		       u.username, u.display_name, u.avatar_url
		FROM voice_states vs
		JOIN users u ON u.id = vs.user_id
		WHERE vs.channel_id = $1
		ORDER BY vs.connected_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, channelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var states []VoiceStateWithUser
	for rows.Next() {
		var s VoiceStateWithUser
		if err := rows.Scan(
			&s.ID, &s.UserID, &s.ChannelID, &s.ServerID,
			&s.SelfMuted, &s.SelfDeafened, &s.SelfVideo, &s.SelfStream,
			&s.Muted, &s.Deafened, &s.SessionID, &s.ConnectedAt,
			&s.Username, &s.DisplayName, &s.Avatar,
		); err != nil {
			return nil, err
		}
		states = append(states, s)
	}

	return states, nil
}

// GetByServer returns all voice states for a server with user info
func (r *VoiceStateRepository) GetByServer(ctx context.Context, serverID uuid.UUID) ([]VoiceStateWithUser, error) {
	query := `
		SELECT vs.id, vs.user_id, vs.channel_id, vs.server_id,
		       vs.self_muted, vs.self_deafened, vs.self_video, vs.self_stream,
		       vs.muted, vs.deafened, vs.session_id, vs.connected_at,
		       u.username, u.display_name, u.avatar_url
		FROM voice_states vs
		JOIN users u ON u.id = vs.user_id
		WHERE vs.server_id = $1
		ORDER BY vs.channel_id, vs.connected_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var states []VoiceStateWithUser
	for rows.Next() {
		var s VoiceStateWithUser
		if err := rows.Scan(
			&s.ID, &s.UserID, &s.ChannelID, &s.ServerID,
			&s.SelfMuted, &s.SelfDeafened, &s.SelfVideo, &s.SelfStream,
			&s.Muted, &s.Deafened, &s.SessionID, &s.ConnectedAt,
			&s.Username, &s.DisplayName, &s.Avatar,
		); err != nil {
			return nil, err
		}
		states = append(states, s)
	}

	return states, nil
}

// GetByUser returns the voice state for a specific user
func (r *VoiceStateRepository) GetByUser(ctx context.Context, userID uuid.UUID) (*VoiceState, error) {
	var state VoiceState

	query := `
		SELECT id, user_id, channel_id, server_id, self_muted, self_deafened,
		       self_video, self_stream, muted, deafened, session_id, connected_at
		FROM voice_states
		WHERE user_id = $1
	`

	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&state.ID, &state.UserID, &state.ChannelID, &state.ServerID,
		&state.SelfMuted, &state.SelfDeafened, &state.SelfVideo, &state.SelfStream,
		&state.Muted, &state.Deafened, &state.SessionID, &state.ConnectedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &state, nil
}

// GetPeersInChannel returns other users in the same channel (excluding the given user)
func (r *VoiceStateRepository) GetPeersInChannel(ctx context.Context, channelID, excludeUserID uuid.UUID) ([]VoiceStateWithUser, error) {
	query := `
		SELECT vs.id, vs.user_id, vs.channel_id, vs.server_id,
		       vs.self_muted, vs.self_deafened, vs.self_video, vs.self_stream,
		       vs.muted, vs.deafened, vs.session_id, vs.connected_at,
		       u.username, u.display_name, u.avatar_url
		FROM voice_states vs
		JOIN users u ON u.id = vs.user_id
		WHERE vs.channel_id = $1 AND vs.user_id != $2
		ORDER BY vs.connected_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, channelID, excludeUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var states []VoiceStateWithUser
	for rows.Next() {
		var s VoiceStateWithUser
		if err := rows.Scan(
			&s.ID, &s.UserID, &s.ChannelID, &s.ServerID,
			&s.SelfMuted, &s.SelfDeafened, &s.SelfVideo, &s.SelfStream,
			&s.Muted, &s.Deafened, &s.SessionID, &s.ConnectedAt,
			&s.Username, &s.DisplayName, &s.Avatar,
		); err != nil {
			return nil, err
		}
		states = append(states, s)
	}

	return states, nil
}
