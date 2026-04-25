// Package matrixfederation implements Matrix Federation protocol support for Hearth.
// This file implements an in-memory federation event store for testing and development.
package matrixfederation

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// FederationEventStore manages federation events for rooms.
type FederationEventStore interface {
	// StoreEvent stores an event in the store.
	StoreEvent(ctx context.Context, event *Event) error

	// GetEvent retrieves an event by its event ID.
	GetEvent(ctx context.Context, eventID string) (*Event, error)

	// GetEventsByRoom retrieves events for a room, ordered by insertion, limited to the most recent 'limit' events.
	GetEventsByRoom(ctx context.Context, roomID string, limit int) ([]*Event, error)

	// GetEventsAfter returns events after a given event (by position in room event list), limited to 'limit' events.
	GetEventsAfter(ctx context.Context, roomID string, afterEventID string, limit int) ([]*Event, error)

	// HasEvent checks whether an event with the given ID exists in the store.
	HasEvent(ctx context.Context, eventID string) (bool, error)

	// GetAuthEvents retrieves the authorization events for a given event.
	// Auth events are typically: create event, power levels, member event for sender, and join rules from the room's current state.
	GetAuthEvents(ctx context.Context, event *Event) ([]*Event, error)
}

// InMemoryFederationEventStore is a thread-safe in-memory implementation of FederationEventStore.
// Suitable for development and testing.
type InMemoryFederationEventStore struct {
	mu        sync.RWMutex
	byEventID map[string]*Event
	byRoomID  map[string][]*Event
}

// NewInMemoryFederationEventStore creates a new in-memory federation event store.
func NewInMemoryFederationEventStore() *InMemoryFederationEventStore {
	return &InMemoryFederationEventStore{
		byEventID: make(map[string]*Event),
		byRoomID:  make(map[string][]*Event),
	}
}

// StoreEvent stores an event in the store.
func (s *InMemoryFederationEventStore) StoreEvent(_ context.Context, event *Event) error {
	if event == nil {
		return fmt.Errorf("%w: event is nil", ErrInvalidEventFormat)
	}
	if event.EventID == "" {
		return fmt.Errorf("%w: event_id is required", ErrMissingRequiredField)
	}
	if event.RoomID == "" {
		return fmt.Errorf("%w: room_id is required", ErrMissingRequiredField)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Only store if not already present to maintain insertion order.
	if _, exists := s.byEventID[event.EventID]; exists {
		return nil
	}

	s.byEventID[event.EventID] = event
	s.byRoomID[event.RoomID] = append(s.byRoomID[event.RoomID], event)

	return nil
}

// GetEvent retrieves an event by its event ID.
func (s *InMemoryFederationEventStore) GetEvent(_ context.Context, eventID string) (*Event, error) {
	if eventID == "" {
		return nil, fmt.Errorf("%w: event_id is required", ErrMissingRequiredField)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	event, ok := s.byEventID[eventID]
	if !ok {
		return nil, fmt.Errorf("event not found: %s", eventID)
	}

	return event, nil
}

// GetEventsByRoom retrieves events for a room, ordered by insertion, limited to the most recent 'limit' events.
func (s *InMemoryFederationEventStore) GetEventsByRoom(_ context.Context, roomID string, limit int) ([]*Event, error) {
	if roomID == "" {
		return nil, fmt.Errorf("%w: room_id is required", ErrMissingRequiredField)
	}
	if limit < 0 {
		return nil, fmt.Errorf("limit must be non-negative")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	events := s.byRoomID[roomID]
	if len(events) == 0 {
		return []*Event{}, nil
	}

	if limit == 0 || limit > len(events) {
		limit = len(events)
	}

	// Return a copy of the most recent events in insertion order.
	start := len(events) - limit
	result := make([]*Event, limit)
	copy(result, events[start:])

	return result, nil
}

// GetEventsAfter returns events after a given event (by position in room event list), limited to 'limit' events.
func (s *InMemoryFederationEventStore) GetEventsAfter(_ context.Context, roomID string, afterEventID string, limit int) ([]*Event, error) {
	if roomID == "" {
		return nil, fmt.Errorf("%w: room_id is required", ErrMissingRequiredField)
	}
	if afterEventID == "" {
		return nil, fmt.Errorf("after_event_id is required")
	}
	if limit < 0 {
		return nil, fmt.Errorf("limit must be non-negative")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	events := s.byRoomID[roomID]
	if len(events) == 0 {
		return []*Event{}, nil
	}

	// Find the position of the after event.
	idx := -1
	for i, e := range events {
		if e.EventID == afterEventID {
			idx = i
			break
		}
	}

	if idx == -1 {
		return nil, fmt.Errorf("after event not found: %s", afterEventID)
	}

	// Events after the found event.
	start := idx + 1
	if start >= len(events) {
		return []*Event{}, nil
	}

	if limit == 0 || limit > len(events)-start {
		limit = len(events) - start
	}

	result := make([]*Event, limit)
	copy(result, events[start:start+limit])

	return result, nil
}

// HasEvent checks whether an event with the given ID exists in the store.
func (s *InMemoryFederationEventStore) HasEvent(_ context.Context, eventID string) (bool, error) {
	if eventID == "" {
		return false, fmt.Errorf("%w: event_id is required", ErrMissingRequiredField)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.byEventID[eventID]
	return ok, nil
}

// GetAuthEvents retrieves the authorization events for a given event.
// Auth events are: create event, power levels, member event for sender, and join rules from the room's current state.
func (s *InMemoryFederationEventStore) GetAuthEvents(_ context.Context, event *Event) ([]*Event, error) {
	if event == nil {
		return nil, fmt.Errorf("event is nil")
	}
	if event.RoomID == "" {
		return nil, fmt.Errorf("%w: room_id is required", ErrMissingRequiredField)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	events := s.byRoomID[event.RoomID]
	if len(events) == 0 {
		return nil, fmt.Errorf("no events found for room: %s", event.RoomID)
	}

	var createEvent, powerLevelsEvent, joinRulesEvent *Event
	var memberEvent *Event

	// Scan all events in the room to find the latest state events.
	for _, e := range events {
		if !e.IsStateEvent() {
			continue
		}

		switch e.Type {
		case EventTypeCreate:
			createEvent = e
		case EventTypePowerLevels:
			powerLevelsEvent = e
		case EventTypeJoinRules:
			joinRulesEvent = e
		case EventTypeMember:
			if e.StateKey != nil && *e.StateKey == event.Sender {
				memberEvent = e
			}
		}
	}

	// Build the auth events list in a deterministic order.
	var authEvents []*Event
	if createEvent != nil {
		authEvents = append(authEvents, createEvent)
	}
	if powerLevelsEvent != nil {
		authEvents = append(authEvents, powerLevelsEvent)
	}
	if memberEvent != nil {
		authEvents = append(authEvents, memberEvent)
	}
	if joinRulesEvent != nil {
		authEvents = append(authEvents, joinRulesEvent)
	}

	// Sort auth events by depth to ensure deterministic ordering.
	sort.Slice(authEvents, func(i, j int) bool {
		if authEvents[i].Depth != authEvents[j].Depth {
			return authEvents[i].Depth < authEvents[j].Depth
		}
		return strings.Compare(authEvents[i].EventID, authEvents[j].EventID) < 0
	})

	return authEvents, nil
}
