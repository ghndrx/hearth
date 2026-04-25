package matrixfederation

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"hearth/internal/database/postgres"
)

// PostgresFederationEventStore is a PostgreSQL-backed implementation of FederationEventStore.
type PostgresFederationEventStore struct {
	repo *postgres.FederationRepository
}

// NewPostgresFederationEventStore creates a new Postgres-backed event store.
func NewPostgresFederationEventStore(repo *postgres.FederationRepository) *PostgresFederationEventStore {
	return &PostgresFederationEventStore{repo: repo}
}

// StoreEvent stores an event in the PostgreSQL store.
func (s *PostgresFederationEventStore) StoreEvent(ctx context.Context, event *Event) error {
	// Convert Event to FederationEvent
	content, _ := json.Marshal(event.Content)
	sigs, _ := json.Marshal(event.Signatures)
	hashes, _ := json.Marshal(event.Hashes)

	fe := &postgres.FederationEvent{
		EventID:      event.EventID,
		RoomID:       event.RoomID,
		Type:         event.Type,
		Sender:       event.Sender,
		OriginServer: event.Origin,
		OriginTS:     event.OriginServerTS,
		Content:      content,
		AuthEvents:   event.AuthEvents,
		PrevEvents:   event.PrevEvents,
		Signatures:   sigs,
		Hashes:       hashes,
		Depth:        event.Depth,
		ReceivedAt:   time.Now(),
	}

	return s.repo.StoreEvent(ctx, fe)
}

// GetEvent retrieves an event by its event ID.
func (s *PostgresFederationEventStore) GetEvent(ctx context.Context, eventID string) (*Event, error) {
	fe, err := s.repo.GetEvent(ctx, eventID)
	if err != nil {
		return nil, err
	}
	if fe == nil {
		return nil, nil
	}
	return feToEvent(fe)
}

// GetEventsByRoom retrieves events for a room.
func (s *PostgresFederationEventStore) GetEventsByRoom(ctx context.Context, roomID string, limit int) ([]*Event, error) {
	events, err := s.repo.GetEventsByRoom(ctx, roomID, limit)
	if err != nil {
		return nil, err
	}
	result := make([]*Event, len(events))
	for i, fe := range events {
		e, err := feToEvent(&fe)
		if err != nil {
			return nil, err
		}
		result[i] = e
	}
	return result, nil
}

// GetEventsAfter returns events after a given event.
func (s *PostgresFederationEventStore) GetEventsAfter(ctx context.Context, roomID string, afterEventID string, limit int) ([]*Event, error) {
	// Get all events after the given event
	events, err := s.repo.GetEventsByRoom(ctx, roomID, limit*2)
	if err != nil {
		return nil, err
	}
	// Find the afterEvent and return events after it
	var result []*Event
	found := false
	for _, fe := range events {
		if fe.EventID == afterEventID {
			found = true
			continue
		}
		if found {
			e, err := feToEvent(&fe)
			if err != nil {
				return nil, err
			}
			result = append(result, e)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

// HasEvent checks whether an event exists.
func (s *PostgresFederationEventStore) HasEvent(ctx context.Context, eventID string) (bool, error) {
	return s.repo.HasEvent(ctx, eventID)
}

// GetAuthEvents retrieves auth events for a given event.
func (s *PostgresFederationEventStore) GetAuthEvents(ctx context.Context, event *Event) ([]*Event, error) {
	events, err := s.repo.GetAuthEvents(ctx, event.RoomID)
	if err != nil {
		return nil, err
	}
	result := make([]*Event, len(events))
	for i, fe := range events {
		e, err := feToEvent(&fe)
		if err != nil {
			return nil, err
		}
		result[i] = e
	}
	return result, nil
}

// feToEvent converts a Postgres FederationEvent to a matrixfederation Event
func feToEvent(fe *postgres.FederationEvent) (*Event, error) {
	var content map[string]interface{}
	if err := json.Unmarshal(fe.Content, &content); err != nil {
		return nil, fmt.Errorf("unmarshal content: %w", err)
	}

	var sigs map[string]map[string]string
	if err := json.Unmarshal(fe.Signatures, &sigs); err != nil {
		return nil, fmt.Errorf("unmarshal signatures: %w", err)
	}

	var hashes EventHashes
	if err := json.Unmarshal(fe.Hashes, &hashes); err != nil {
		return nil, fmt.Errorf("unmarshal hashes: %w", err)
	}

	return &Event{
		EventID:        fe.EventID,
		RoomID:         fe.RoomID,
		Sender:         fe.Sender,
		Type:           fe.Type,
		Content:        content,
		Origin:         fe.OriginServer,
		OriginServerTS: fe.OriginTS,
		AuthEvents:     fe.AuthEvents,
		PrevEvents:     fe.PrevEvents,
		Depth:          fe.Depth,
		Signatures:     sigs,
		Hashes:         hashes,
	}, nil
}

// PostgresRoomAliasStore is a PostgreSQL-backed implementation of RoomAliasStore.
type PostgresRoomAliasStore struct {
	repo *postgres.FederationRepository
}

// NewPostgresRoomAliasStore creates a new Postgres-backed room alias store.
func NewPostgresRoomAliasStore(repo *postgres.FederationRepository) *PostgresRoomAliasStore {
	return &PostgresRoomAliasStore{repo: repo}
}

// CreateMapping creates a new mapping between a room ID and a Hearth channel.
func (s *PostgresRoomAliasStore) CreateMapping(ctx context.Context, roomID RoomID, channelID uuid.UUID, aliases []Alias) error {
	mapping := &postgres.RoomChannelMap{
		RoomID:    roomID.String(),
		ChannelID: channelID,
		CreatedAt: time.Now(),
	}
	return s.repo.SetRoomChannelMap(ctx, mapping)
}

// GetByRoomID returns the channel ID for a given room ID.
func (s *PostgresRoomAliasStore) GetByRoomID(ctx context.Context, roomID RoomID) (uuid.UUID, []Alias, error) {
	channelID, err := s.repo.GetChannelByRoomID(ctx, roomID.String())
	if err != nil {
		return uuid.Nil, nil, err
	}
	if channelID == nil {
		return uuid.Nil, nil, ErrRoomNotFound
	}
	return *channelID, nil, nil
}

// GetByAlias returns the room ID and channel ID for a given alias.
func (s *PostgresRoomAliasStore) GetByAlias(ctx context.Context, alias Alias) (RoomID, uuid.UUID, error) {
	// This would require a different query - for now return not found
	return RoomID{}, uuid.Nil, ErrAliasNotFound
}

// GetByChannelID returns the room ID and aliases for a given Hearth channel.
func (s *PostgresRoomAliasStore) GetByChannelID(ctx context.Context, channelID uuid.UUID) (RoomID, []Alias, error) {
	roomIDStr, err := s.repo.GetRoomByChannelID(ctx, channelID)
	if err != nil {
		return RoomID{}, nil, err
	}
	if roomIDStr == nil {
		return RoomID{}, nil, ErrRoomNotFound
	}
	roomID, err := ParseRoomID(*roomIDStr)
	if err != nil {
		return RoomID{}, nil, err
	}
	return roomID, nil, nil
}

// AddAlias adds an alias to an existing room mapping.
func (s *PostgresRoomAliasStore) AddAlias(ctx context.Context, roomID RoomID, alias Alias) error {
	// Would need additional table or field - not implemented
	return nil
}

// ListAliases returns all aliases for a room.
func (s *PostgresRoomAliasStore) ListAliases(ctx context.Context, roomID RoomID) ([]Alias, error) {
	// Would need additional query - return empty for now
	return nil, nil
}

// RemoveAlias removes an alias from a room.
func (s *PostgresRoomAliasStore) RemoveAlias(ctx context.Context, alias Alias) error {
	// Not implemented
	return nil
}
