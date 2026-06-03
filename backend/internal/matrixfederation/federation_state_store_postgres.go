package matrixfederation

import (
	"context"
	"fmt"
	"sync"

	"hearth/internal/database/postgres"
)

// PostgresStateStore is a PostgreSQL-backed implementation of StateStore.
// It reconstructs room state from the federation_events table and caches
// it in an embedded InMemoryStateStore for performance.
type PostgresStateStore struct {
	repo   *postgres.FederationRepository
	cache  *InMemoryStateStore
	mu     sync.Mutex
	loaded map[string]bool
}

// NewPostgresStateStore creates a new Postgres-backed state store.
func NewPostgresStateStore(repo *postgres.FederationRepository) *PostgresStateStore {
	return &PostgresStateStore{
		repo:   repo,
		cache:  NewInMemoryStateStore(),
		loaded: make(map[string]bool),
	}
}

// GetOrCreateRoomState returns the existing room state or creates a new one.
// It loads historical events from Postgres on first access for a given room.
func (s *PostgresStateStore) GetOrCreateRoomState(roomID RoomID) *FederatedRoomState {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := roomID.String()
	rs := s.cache.GetOrCreateRoomState(roomID)
	if s.loaded[key] {
		return rs
	}

	s.loaded[key] = true
	events, err := s.repo.GetEventsByRoom(context.Background(), key, 10000)
	if err != nil {
		return rs
	}

	for _, fe := range events {
		event, err := feToEvent(&fe)
		if err != nil {
			continue
		}
		_ = rs.AddEvent(event)
	}

	return rs
}

// GetRoomState returns the room state for the given room ID.
// It returns ErrRoomNotFound if no events exist for the room in Postgres.
func (s *PostgresStateStore) GetRoomState(roomID RoomID) (*FederatedRoomState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := roomID.String()
	if s.loaded[key] {
		return s.cache.GetRoomState(roomID)
	}

	events, err := s.repo.GetEventsByRoom(context.Background(), key, 10000)
	if err != nil {
		return nil, err
	}
	if len(events) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrRoomNotFound, key)
	}

	rs := s.cache.GetOrCreateRoomState(roomID)
	s.loaded[key] = true
	for _, fe := range events {
		event, err := feToEvent(&fe)
		if err != nil {
			continue
		}
		_ = rs.AddEvent(event)
	}

	return rs, nil
}
