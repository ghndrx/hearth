// Package matrixfederation implements Matrix Federation protocol support for Hearth.
// This file implements in-memory room state tracking for federated rooms.
package matrixfederation

import (
	"encoding/json"
	"fmt"
	"sync"
)

// FederatedRoomState tracks the current state of a federated room,
// including members, power levels, and DAG forward extremities.
// All methods are safe for concurrent use.
type FederatedRoomState struct {
	mu sync.RWMutex

	// RoomID is the room identifier.
	RoomID RoomID

	// Members maps MXIDs to their current membership state string.
	Members map[string]string

	// PowerLevels holds the current room power levels.
	PowerLevels RoomPowerLevelsContent

	// ForwardExtremities is a set of event IDs at the frontier of the DAG.
	ForwardExtremities map[string]struct{}

	// CurrentDepth is the maximum depth seen in the DAG.
	CurrentDepth int64

	// stateEvents stores the current state events by (eventType, stateKey).
	stateEvents map[string]*Event
}

// NewFederatedRoomState creates a new FederatedRoomState for the given room.
func NewFederatedRoomState(roomID RoomID) *FederatedRoomState {
	return &FederatedRoomState{
		RoomID:             roomID,
		Members:            make(map[string]string),
		PowerLevels:        NewRoomPowerLevelsContent(""),
		ForwardExtremities: make(map[string]struct{}),
		CurrentDepth:       0,
		stateEvents:        make(map[string]*Event),
	}
}

// AddEvent updates the room state based on the provided event.
// It updates members, power levels, forward extremities, and current depth.
func (s *FederatedRoomState) AddEvent(event *Event) error {
	if event == nil {
		return fmt.Errorf("%w: event is nil", ErrInvalidEventFormat)
	}
	if event.RoomID != s.RoomID.String() {
		return fmt.Errorf("event room_id %q does not match state room %q", event.RoomID, s.RoomID.String())
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Update type-specific state.
	switch event.Type {
	case EventTypeMember:
		if event.StateKey != nil {
			if membership, ok := event.Content["membership"].(string); ok {
				s.Members[*event.StateKey] = membership
			}
		}
	case EventTypePowerLevels:
		s.PowerLevels = parsePowerLevelsFromEvent(event)
	}

	// Store state event for GetStateEvent.
	if event.IsStateEvent() {
		key := stateEventKey(event.Type, event.StateKeyString())
		s.stateEvents[key] = event
	}

	// Update forward extremities: remove prev_events from frontier, add this event.
	for _, prev := range event.PrevEvents {
		delete(s.ForwardExtremities, prev)
	}
	s.ForwardExtremities[event.EventID] = struct{}{}

	// Update current depth to max of current and event depth.
	if event.Depth > s.CurrentDepth {
		s.CurrentDepth = event.Depth
	}

	return nil
}

// GetStateEvent returns the current state event for the given event type and state key.
func (s *FederatedRoomState) GetStateEvent(eventType string, stateKey string) (*Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := stateEventKey(eventType, stateKey)
	event, ok := s.stateEvents[key]
	if !ok {
		return nil, fmt.Errorf("state event not found: %s/%s", eventType, stateKey)
	}
	return event, nil
}

// GetMembers returns a copy of the current members map.
func (s *FederatedRoomState) GetMembers() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	members := make(map[string]string, len(s.Members))
	for k, v := range s.Members {
		members[k] = v
	}
	return members
}

// GetMemberPower returns the power level for the given user ID.
func (s *FederatedRoomState) GetMemberPower(userID string) int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if p, ok := s.PowerLevels.Users[userID]; ok {
		return p
	}
	return s.PowerLevels.UsersDefault
}

// HasMember reports whether the given user ID is in the members map.
func (s *FederatedRoomState) HasMember(userID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.Members[userID]
	return ok
}

// GetForwardExtremities returns the current frontier event IDs.
func (s *FederatedRoomState) GetForwardExtremities() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]string, 0, len(s.ForwardExtremities))
	for id := range s.ForwardExtremities {
		result = append(result, id)
	}
	return result
}

// GetCurrentDepth returns the maximum depth seen in the DAG.
func (s *FederatedRoomState) GetCurrentDepth() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.CurrentDepth
}

// CanSendEvent returns true if the given user has sufficient power to send the given event type.
func (s *FederatedRoomState) CanSendEvent(userID string, eventType string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userPower := s.PowerLevels.UsersDefault
	if p, ok := s.PowerLevels.Users[userID]; ok {
		userPower = p
	}

	var requiredPower int64

	switch eventType {
	case EventTypePowerLevels:
		// m.room.power_levels typically requires state_default if set, otherwise 100.
		requiredPower = s.PowerLevels.StateDefault
		if requiredPower == 0 {
			requiredPower = 100
		}
	case EventTypeJoinRules, EventTypeName, EventTypeTopic, EventTypeAvatar,
		EventTypeCanonicalAlias, EventTypeHistoryVisibility:
		requiredPower = s.PowerLevels.StateDefault
		if requiredPower == 0 {
			requiredPower = 50
		}
	default:
		// Message events and other types default to events_default.
		requiredPower = s.PowerLevels.EventsDefault
	}

	// Check event-specific override.
	if p, ok := s.PowerLevels.Events[eventType]; ok {
		requiredPower = p
	}

	return userPower >= requiredPower
}

// stateEventKey returns a unique key for a state event.
func stateEventKey(eventType, stateKey string) string {
	return eventType + "\x00" + stateKey
}

// parsePowerLevelsFromEvent extracts RoomPowerLevelsContent from an event.
func parsePowerLevelsFromEvent(event *Event) RoomPowerLevelsContent {
	var pl RoomPowerLevelsContent

	if v, ok := event.Content["ban"]; ok {
		pl.Ban = toInt64(v)
	}
	if v, ok := event.Content["kick"]; ok {
		pl.Kick = toInt64(v)
	}
	if v, ok := event.Content["redact"]; ok {
		pl.Redact = toInt64(v)
	}
	if v, ok := event.Content["invite"]; ok {
		pl.Invite = toInt64(v)
	}
	if v, ok := event.Content["events_default"]; ok {
		pl.EventsDefault = toInt64(v)
	}
	if v, ok := event.Content["users_default"]; ok {
		pl.UsersDefault = toInt64(v)
	}
	if v, ok := event.Content["state_default"]; ok {
		pl.StateDefault = toInt64(v)
	}

	pl.Users = make(map[string]int64)
	if users, ok := event.Content["users"].(map[string]interface{}); ok {
		for u, p := range users {
			pl.Users[u] = toInt64(p)
		}
	} else if users, ok := event.Content["users"].(map[string]int64); ok {
		for u, p := range users {
			pl.Users[u] = p
		}
	}

	pl.Events = make(map[string]int64)
	if events, ok := event.Content["events"].(map[string]interface{}); ok {
		for et, p := range events {
			pl.Events[et] = toInt64(p)
		}
	} else if events, ok := event.Content["events"].(map[string]int64); ok {
		for et, p := range events {
			pl.Events[et] = p
		}
	}

	return pl
}

// toInt64 converts a numeric interface{} value to int64.
func toInt64(v interface{}) int64 {
	switch val := v.(type) {
	case float64:
		return int64(val)
	case int64:
		return val
	case int:
		return int64(val)
	case json.Number:
		n, _ := val.Int64()
		return n
	default:
		return 0
	}
}

// InMemoryStateStore manages multiple FederatedRoomState instances.
// All methods are safe for concurrent use.
type InMemoryStateStore struct {
	mu    sync.RWMutex
	rooms map[string]*FederatedRoomState
}

// NewInMemoryStateStore creates a new InMemoryStateStore.
func NewInMemoryStateStore() *InMemoryStateStore {
	return &InMemoryStateStore{
		rooms: make(map[string]*FederatedRoomState),
	}
}

// GetOrCreateRoomState returns the existing room state or creates a new one.
func (s *InMemoryStateStore) GetOrCreateRoomState(roomID RoomID) *FederatedRoomState {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := roomID.String()
	if state, ok := s.rooms[key]; ok {
		return state
	}

	state := NewFederatedRoomState(roomID)
	s.rooms[key] = state
	return state
}

// GetRoomState returns the room state for the given room ID.
func (s *InMemoryStateStore) GetRoomState(roomID RoomID) (*FederatedRoomState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := roomID.String()
	state, ok := s.rooms[key]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrRoomNotFound, roomID.String())
	}
	return state, nil
}
