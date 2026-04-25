package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// FederationEvent represents a persisted federation event
type FederationEvent struct {
	EventID       string          `json:"event_id" db:"event_id"`
	RoomID        string          `json:"room_id" db:"room_id"`
	Type          string          `json:"type" db:"type"`
	Sender        string          `json:"sender" db:"sender"`
	OriginServer  string          `json:"origin_server" db:"origin_server"`
	OriginTS      int64           `json:"origin_ts" db:"origin_ts"`
	Content       json.RawMessage `json:"content" db:"content"`
	AuthEvents    []string        `json:"auth_events" db:"auth_events"`
	PrevEvents    []string        `json:"prev_events" db:"prev_events"`
	Signatures    json.RawMessage `json:"signatures" db:"signatures"`
	Hashes        json.RawMessage `json:"hashes" db:"hashes"`
	Depth         int64           `json:"depth" db:"depth"`
	Redacts       *string         `json:"redacts" db:"redacts"`
	Rejected      bool            `json:"rejected" db:"rejected"`
	RejectReason  *string         `json:"reject_reason" db:"reject_reason"`
	ReceivedAt    time.Time       `json:"received_at" db:"received_at"`
}

// FederationTransaction represents an inbound transaction
type FederationTransaction struct {
	Origin     string    `json:"origin" db:"origin"`
	TxnID      string    `json:"txn_id" db:"txn_id"`
	ReceivedAt time.Time `json:"received_at" db:"received_at"`
	PDUCount   int       `json:"pdu_count" db:"pdu_count"`
}

// FederationOutboundQueue represents an outbound queued PDU
type FederationOutboundQueue struct {
	ID            uuid.UUID       `json:"id" db:"id"`
	Destination   string          `json:"destination_server" db:"destination_server"`
	PDU           json.RawMessage `json:"pdu" db:"pdu"`
	Attempt       int             `json:"attempt" db:"attempt"`
	NextRetryAt   time.Time       `json:"next_retry_at" db:"next_retry_at"`
	LastError     *string         `json:"last_error" db:"last_error"`
	Status        string          `json:"status" db:"status"`
	CreatedAt     time.Time       `json:"created_at" db:"created_at"`
	SentAt        *time.Time      `json:"sent_at" db:"sent_at"`
}

// RoomChannelMap maps Matrix room IDs to Hearth channel IDs
type RoomChannelMap struct {
	RoomID    string    `json:"room_id" db:"room_id"`
	ChannelID uuid.UUID `json:"channel_id" db:"channel_id"`
	ServerID  *uuid.UUID `json:"server_id" db:"server_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// FederationRoomState represents persisted room state
type FederationRoomState struct {
	RoomID    string          `json:"room_id" db:"room_id"`
	StateKey  string          `json:"state_key" db:"state_key"`
	Sender    string          `json:"sender" db:"sender"`
	Type      string          `json:"type" db:"type"`
	Content   json.RawMessage `json:"content" db:"content"`
	CreatedAt time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt time.Time      `json:"updated_at" db:"updated_at"`
}

// FederationRepository handles federation data access
type FederationRepository struct {
	db *sqlx.DB
}

// NewFederationRepository creates a new federation repository
func NewFederationRepository(db *sqlx.DB) *FederationRepository {
	return &FederationRepository{db: db}
}

// StoreTransaction stores an inbound transaction for idempotency
func (r *FederationRepository) StoreTransaction(ctx context.Context, txn *FederationTransaction) error {
	query := `
		INSERT INTO federation_transactions (origin, txn_id, received_at, pdu_count)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (origin, txn_id) DO NOTHING
	`
	_, err := r.db.ExecContext(ctx, query, txn.Origin, txn.TxnID, txn.ReceivedAt, txn.PDUCount)
	return err
}

// GetTransaction retrieves a transaction by origin and txn_id
func (r *FederationRepository) GetTransaction(ctx context.Context, origin, txnID string) (*FederationTransaction, error) {
	var txn FederationTransaction
	query := `SELECT origin, txn_id, received_at, pdu_count FROM federation_transactions WHERE origin = $1 AND txn_id = $2`
	err := r.db.GetContext(ctx, &txn, query, origin, txnID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &txn, nil
}

// StoreEvent stores a federation event
func (r *FederationRepository) StoreEvent(ctx context.Context, event *FederationEvent) error {
	query := `
		INSERT INTO federation_events (
			event_id, room_id, type, sender, origin_server, origin_ts,
			content, auth_events, prev_events, signatures, hashes, depth,
			redacts, rejected, reject_reason, received_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (event_id) DO NOTHING
	`
	_, err := r.db.ExecContext(ctx, query,
		event.EventID, event.RoomID, event.Type, event.Sender, event.OriginServer, event.OriginTS,
		event.Content, event.AuthEvents, event.PrevEvents, event.Signatures, event.Hashes, event.Depth,
		event.Redacts, event.Rejected, event.RejectReason, event.ReceivedAt,
	)
	return err
}

// GetEvent retrieves an event by event_id
func (r *FederationRepository) GetEvent(ctx context.Context, eventID string) (*FederationEvent, error) {
	var event FederationEvent
	query := `SELECT * FROM federation_events WHERE event_id = $1`
	err := r.db.GetContext(ctx, &event, query, eventID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &event, nil
}

// GetEventsByRoom retrieves events for a room ordered by depth
func (r *FederationRepository) GetEventsByRoom(ctx context.Context, roomID string, limit int) ([]FederationEvent, error) {
	var events []FederationEvent
	query := `SELECT * FROM federation_events WHERE room_id = $1 ORDER BY origin_ts DESC LIMIT $2`
	err := r.db.SelectContext(ctx, &events, query, roomID, limit)
	if err != nil {
		return nil, err
	}
	return events, nil
}

// HasEvent checks if an event exists
func (r *FederationRepository) HasEvent(ctx context.Context, eventID string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM federation_events WHERE event_id = $1)`
	err := r.db.GetContext(ctx, &exists, query, eventID)
	return exists, err
}

// SetRoomChannelMap sets the mapping between a Matrix room and Hearth channel
func (r *FederationRepository) SetRoomChannelMap(ctx context.Context, mapping *RoomChannelMap) error {
	query := `
		INSERT INTO room_channel_map (room_id, channel_id, server_id, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (room_id) DO UPDATE SET
			channel_id = EXCLUDED.channel_id,
			server_id = EXCLUDED.server_id
	`
	_, err := r.db.ExecContext(ctx, query, mapping.RoomID, mapping.ChannelID, mapping.ServerID, mapping.CreatedAt)
	return err
}

// GetChannelByRoomID retrieves the Hearth channel ID for a Matrix room
func (r *FederationRepository) GetChannelByRoomID(ctx context.Context, roomID string) (*uuid.UUID, error) {
	var channelID uuid.UUID
	query := `SELECT channel_id FROM room_channel_map WHERE room_id = $1`
	err := r.db.GetContext(ctx, &channelID, query, roomID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &channelID, nil
}

// GetRoomByChannelID retrieves the Matrix room ID for a Hearth channel
func (r *FederationRepository) GetRoomByChannelID(ctx context.Context, channelID uuid.UUID) (*string, error) {
	var roomID string
	query := `SELECT room_id FROM room_channel_map WHERE channel_id = $1`
	err := r.db.GetContext(ctx, &roomID, query, channelID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &roomID, nil
}

// EnqueueOutbound adds an event to the outbound queue
func (r *FederationRepository) EnqueueOutbound(ctx context.Context, queue *FederationOutboundQueue) error {
	query := `
		INSERT INTO federation_outbound_queue (id, destination_server, pdu, attempt, next_retry_at, last_error, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.ExecContext(ctx, query,
		queue.ID, queue.Destination, queue.PDU, queue.Attempt, queue.NextRetryAt, queue.LastError, queue.Status, queue.CreatedAt,
	)
	return err
}

// GetPendingOutbound retrieves pending outbound events for a destination
func (r *FederationRepository) GetPendingOutbound(ctx context.Context, destination string, limit int) ([]FederationOutboundQueue, error) {
	var items []FederationOutboundQueue
	query := `
		SELECT * FROM federation_outbound_queue
		WHERE destination_server = $1 AND status IN ('pending', 'failed') AND next_retry_at <= $2
		ORDER BY created_at ASC LIMIT $3
	`
	err := r.db.SelectContext(ctx, &items, query, destination, time.Now(), limit)
	if err != nil {
		return nil, err
	}
	return items, nil
}

// MarkOutboundSent marks an outbound event as sent
func (r *FederationRepository) MarkOutboundSent(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE federation_outbound_queue SET status = 'sent', sent_at = $2 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, time.Now())
	return err
}

// MarkOutboundFailed marks an outbound event as failed with retry
func (r *FederationRepository) MarkOutboundFailed(ctx context.Context, id uuid.UUID, errMsg string, nextRetry time.Time) error {
	query := `UPDATE federation_outbound_queue SET status = 'failed', last_error = $2, next_retry_at = $3, attempt = attempt + 1 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, errMsg, nextRetry)
	return err
}

// StoreRoomState stores or updates room state
func (r *FederationRepository) StoreRoomState(ctx context.Context, state *FederationRoomState) error {
	query := `
		INSERT INTO federation_room_state (room_id, state_key, sender, type, content, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (room_id, state_key, type) DO UPDATE SET
			sender = EXCLUDED.sender,
			content = EXCLUDED.content,
			updated_at = EXCLUDED.updated_at
	`
	now := time.Now()
	if state.CreatedAt.IsZero() {
		state.CreatedAt = now
	}
	state.UpdatedAt = now
	_, err := r.db.ExecContext(ctx, query, state.RoomID, state.StateKey, state.Sender, state.Type, state.Content, state.CreatedAt, state.UpdatedAt)
	return err
}

// GetRoomState retrieves room state by room_id, state_key, and type
func (r *FederationRepository) GetRoomState(ctx context.Context, roomID, stateKey, eventType string) (*FederationRoomState, error) {
	var state FederationRoomState
	query := `SELECT * FROM federation_room_state WHERE room_id = $1 AND state_key = $2 AND type = $3`
	err := r.db.GetContext(ctx, &state, query, roomID, stateKey, eventType)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &state, nil
}

// GetRoomStates retrieves all state for a room
func (r *FederationRepository) GetRoomStates(ctx context.Context, roomID string) ([]FederationRoomState, error) {
	var states []FederationRoomState
	query := `SELECT * FROM federation_room_state WHERE room_id = $1 ORDER BY type, state_key`
	err := r.db.SelectContext(ctx, &states, query, roomID)
	if err != nil {
		return nil, err
	}
	return states, nil
}

// GetAuthEvents retrieves auth events for a room (create, members, power levels)
func (r *FederationRepository) GetAuthEvents(ctx context.Context, roomID string) ([]FederationEvent, error) {
	var events []FederationEvent
	// Get: m.room.create, m.room.member events, m.room.power_levels
	types := []string{"m.room.create", "m.room.member", "m.room.power_levels", "m.room.join_rules"}
	placeholders := make([]string, len(types))
	args := make([]interface{}, len(types)+1)
	for i, t := range types {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args[i+1] = t
	}
	args[0] = roomID
	query := fmt.Sprintf(
		`SELECT * FROM federation_events WHERE room_id = $1 AND type IN (%s) ORDER BY depth ASC`,
		strings.Join(placeholders, ","),
	)
	err := r.db.SelectContext(ctx, &events, query, args...)
	if err != nil {
		return nil, err
	}
	return events, nil
}
