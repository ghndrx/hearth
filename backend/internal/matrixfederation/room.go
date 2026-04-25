// Package matrixfederation implements Matrix Federation protocol support for Hearth.
// This file implements room alias and room ID support.
//
// Matrix Spec References:
//   - Client-Server API r0.6.1 § 10: Room Directory
//   - Federation API r0.1.4 § 11: Room Joining
//
// https://spec.matrix.org/v1.12/client-server-api/#room-aliases
// https://spec.matrix.org/v1.12/server-server-api/#room-joining
package matrixfederation

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/google/uuid"
)

// Common errors for room operations.
var (
	ErrRoomNotFound       = errors.New("matrix: room not found")
	ErrAliasNotFound      = errors.New("matrix: room alias not found")
	ErrInvalidRoomID      = errors.New("matrix: invalid room ID format")
	ErrInvalidAlias       = errors.New("matrix: invalid room alias format")
	ErrAliasAlreadyExists = errors.New("matrix: room alias already exists")
	ErrAliasInUse         = errors.New("matrix: room alias already points to a different room")
)

// RoomID represents a Matrix room ID.
// Format: !localpart:server_name
// Example: !abcdef:hearth.example.com
type RoomID struct {
	Localpart  string
	ServerName string
}

// String returns the canonical string representation.
func (r RoomID) String() string {
	return "!" + r.Localpart + ":" + r.ServerName
}

// IsValid reports whether this room ID is well-formed.
func (r RoomID) IsValid() bool {
	return r.Localpart != "" && r.ServerName != ""
}

// Alias represents a Matrix room alias.
// Format: #alias:server_name
// Example: #general:hearth.example.com
type Alias struct {
	Localpart  string
	ServerName string
}

// String returns the canonical string representation.
func (a Alias) String() string {
	return "#" + a.Localpart + ":" + a.ServerName
}

// IsValid reports whether this alias is well-formed.
func (a Alias) IsValid() bool {
	return a.Localpart != "" && a.ServerName != ""
}

// validRoomID matches a full room ID: !localpart:server_name.
var validRoomID = regexp.MustCompile(`^!([^@:]+):(.+)$`)

// validAlias matches a full alias: #localpart:server_name.
var validAlias = regexp.MustCompile(`^#([^@:]+):(.+)$`)

// ParseRoomID parses a string into a RoomID.
func ParseRoomID(raw string) (RoomID, error) {
	if raw == "" || !strings.HasPrefix(raw, "!") {
		return RoomID{}, ErrInvalidRoomID
	}

	matches := validRoomID.FindStringSubmatch(raw)
	if matches == nil {
		return RoomID{}, ErrInvalidRoomID
	}

	return RoomID{
		Localpart:  matches[1],
		ServerName: matches[2],
	}, nil
}

// ParseAlias parses a string into an Alias.
func ParseAlias(raw string) (Alias, error) {
	if raw == "" || !strings.HasPrefix(raw, "#") {
		return Alias{}, ErrInvalidAlias
	}

	matches := validAlias.FindStringSubmatch(raw)
	if matches == nil {
		return Alias{}, ErrInvalidAlias
	}

	return Alias{
		Localpart:  matches[1],
		ServerName: matches[2],
	}, nil
}

// RoomAliasStore manages mappings between Matrix room IDs/aliases and Hearth channels.
type RoomAliasStore interface {
	// CreateMapping creates a new mapping between a room ID and a Hearth channel.
	CreateMapping(ctx context.Context, roomID RoomID, channelID uuid.UUID, aliases []Alias) error

	// GetByRoomID returns the channel ID for a given room ID.
	GetByRoomID(ctx context.Context, roomID RoomID) (uuid.UUID, []Alias, error)

	// GetByAlias returns the room ID and channel ID for a given alias.
	GetByAlias(ctx context.Context, alias Alias) (RoomID, uuid.UUID, error)

	// GetByChannelID returns the room ID and aliases for a given Hearth channel.
	GetByChannelID(ctx context.Context, channelID uuid.UUID) (RoomID, []Alias, error)

	// AddAlias adds an alias to an existing room mapping.
	AddAlias(ctx context.Context, roomID RoomID, alias Alias) error

	// RemoveAlias removes an alias from a room mapping.
	RemoveAlias(ctx context.Context, alias Alias) error

	// RemoveMapping removes a room mapping and all its aliases.
	RemoveMapping(ctx context.Context, roomID RoomID) error

	// ListAliases returns all aliases for a room.
	ListAliases(ctx context.Context, roomID RoomID) ([]Alias, error)
}

// RoomAliasMapping represents an in-memory room alias mapping.
// This is used for testing and development; production should use a database-backed store.
type RoomAliasMapping struct {
	RoomID    RoomID
	ChannelID uuid.UUID
	Aliases   []Alias
}

// InMemoryRoomAliasStore is a thread-safe in-memory implementation of RoomAliasStore.
// Suitable for development and testing.
type InMemoryRoomAliasStore struct {
	mu       sync.RWMutex
	byRoomID map[string]RoomAliasMapping
	byAlias  map[string]RoomAliasMapping
	byChanID map[uuid.UUID]RoomAliasMapping
}

// NewInMemoryRoomAliasStore creates a new in-memory room alias store.
func NewInMemoryRoomAliasStore() *InMemoryRoomAliasStore {
	return &InMemoryRoomAliasStore{
		byRoomID: make(map[string]RoomAliasMapping),
		byAlias:  make(map[string]RoomAliasMapping),
		byChanID: make(map[uuid.UUID]RoomAliasMapping),
	}
}

// CreateMapping implements RoomAliasStore.
func (s *InMemoryRoomAliasStore) CreateMapping(_ context.Context, roomID RoomID, channelID uuid.UUID, aliases []Alias) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	roomKey := roomID.String()
	if _, exists := s.byRoomID[roomKey]; exists {
		return fmt.Errorf("room mapping already exists: %w", ErrAliasAlreadyExists)
	}

	// Check if any alias is already in use.
	for _, alias := range aliases {
		if _, exists := s.byAlias[alias.String()]; exists {
			return fmt.Errorf("alias %s already in use: %w", alias.String(), ErrAliasInUse)
		}
	}

	mapping := RoomAliasMapping{
		RoomID:    roomID,
		ChannelID: channelID,
		Aliases:   aliases,
	}

	s.byRoomID[roomKey] = mapping
	s.byChanID[channelID] = mapping
	for _, alias := range aliases {
		s.byAlias[alias.String()] = mapping
	}

	return nil
}

// GetByRoomID implements RoomAliasStore.
func (s *InMemoryRoomAliasStore) GetByRoomID(_ context.Context, roomID RoomID) (uuid.UUID, []Alias, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	mapping, ok := s.byRoomID[roomID.String()]
	if !ok {
		return uuid.Nil, nil, ErrRoomNotFound
	}

	// Return a copy of aliases.
	aliases := make([]Alias, len(mapping.Aliases))
	copy(aliases, mapping.Aliases)

	return mapping.ChannelID, aliases, nil
}

// GetByAlias implements RoomAliasStore.
func (s *InMemoryRoomAliasStore) GetByAlias(_ context.Context, alias Alias) (RoomID, uuid.UUID, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	mapping, ok := s.byAlias[alias.String()]
	if !ok {
		return RoomID{}, uuid.Nil, ErrAliasNotFound
	}

	return mapping.RoomID, mapping.ChannelID, nil
}

// GetByChannelID implements RoomAliasStore.
func (s *InMemoryRoomAliasStore) GetByChannelID(_ context.Context, channelID uuid.UUID) (RoomID, []Alias, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	mapping, ok := s.byChanID[channelID]
	if !ok {
		return RoomID{}, nil, ErrRoomNotFound
	}

	aliases := make([]Alias, len(mapping.Aliases))
	copy(aliases, mapping.Aliases)

	return mapping.RoomID, aliases, nil
}

// AddAlias implements RoomAliasStore.
func (s *InMemoryRoomAliasStore) AddAlias(_ context.Context, roomID RoomID, alias Alias) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	mapping, ok := s.byRoomID[roomID.String()]
	if !ok {
		return ErrRoomNotFound
	}

	// Check if alias already exists for a different room.
	if existing, exists := s.byAlias[alias.String()]; exists && existing.RoomID.String() != roomID.String() {
		return ErrAliasInUse
	}

	// Check if alias already exists for this room.
	for _, a := range mapping.Aliases {
		if a.String() == alias.String() {
			return nil // Already exists, no-op.
		}
	}

	// Update mapping.
	mapping.Aliases = append(mapping.Aliases, alias)
	s.byRoomID[roomID.String()] = mapping
	s.byChanID[mapping.ChannelID] = mapping
	s.byAlias[alias.String()] = mapping

	return nil
}

// RemoveAlias implements RoomAliasStore.
func (s *InMemoryRoomAliasStore) RemoveAlias(_ context.Context, alias Alias) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	mapping, ok := s.byAlias[alias.String()]
	if !ok {
		return ErrAliasNotFound
	}

	// Remove from mapping's aliases.
	newAliases := make([]Alias, 0, len(mapping.Aliases)-1)
	for _, a := range mapping.Aliases {
		if a.String() != alias.String() {
			newAliases = append(newAliases, a)
		}
	}

	mapping.Aliases = newAliases
	s.byRoomID[mapping.RoomID.String()] = mapping
	s.byChanID[mapping.ChannelID] = mapping
	delete(s.byAlias, alias.String())

	return nil
}

// RemoveMapping implements RoomAliasStore.
func (s *InMemoryRoomAliasStore) RemoveMapping(_ context.Context, roomID RoomID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	mapping, ok := s.byRoomID[roomID.String()]
	if !ok {
		return ErrRoomNotFound
	}

	delete(s.byRoomID, roomID.String())
	delete(s.byChanID, mapping.ChannelID)
	for _, alias := range mapping.Aliases {
		delete(s.byAlias, alias.String())
	}

	return nil
}

// ListAliases implements RoomAliasStore.
func (s *InMemoryRoomAliasStore) ListAliases(_ context.Context, roomID RoomID) ([]Alias, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	mapping, ok := s.byRoomID[roomID.String()]
	if !ok {
		return nil, ErrRoomNotFound
	}

	aliases := make([]Alias, len(mapping.Aliases))
	copy(aliases, mapping.Aliases)
	return aliases, nil
}

// GenerateRoomID creates a new room ID for a given channel on a homeserver.
func GenerateRoomID(channelID uuid.UUID, serverName string) RoomID {
	// Use a URL-safe base64 encoding of the UUID as the localpart.
	// This makes the room ID deterministic for a given channel.
	localpart := base64.RawURLEncoding.EncodeToString(channelID[:])
	localpart = strings.ReplaceAll(localpart, "-", "_")

	return RoomID{
		Localpart:  localpart,
		ServerName: serverName,
	}
}

// ParseRoomIDLocalpart extracts the channel UUID from a room ID's localpart.
func ParseRoomIDLocalpart(localpart string) (uuid.UUID, error) {
	// Replace back any underscores.
	localpart = strings.ReplaceAll(localpart, "_", "-")

	// Try base64 decoding first.
	decoded, err := base64.RawURLEncoding.DecodeString(localpart)
	if err == nil && len(decoded) == 16 {
		var id uuid.UUID
		copy(id[:], decoded)
		return id, nil
	}

	// Fallback: try as plain UUID string.
	return uuid.Parse(localpart)
}

// RoomStateResponse represents the response for GET /_matrix/federation/v1/make_join/{roomId}.
// This is part of the room join handshake.
type RoomStateResponse struct {
	// Event is a placeholder event for the join.
	Event map[string]interface{} `json:"event"`
	// AuthChain is the auth chain events needed for the join.
	AuthChain []map[string]interface{} `json:"auth_chain"`
	// State is the current room state.
	State []map[string]interface{} `json:"state"`
}

// RoomMemberEvent represents a m.room.member event.
type RoomMemberEvent struct {
	// Type is always "m.room.member".
	Type string `json:"type"`
	// StateKey is the user's MXID.
	StateKey string `json:"state_key"`
	// Sender is the MXID of the user who sent this event.
	Sender string `json:"sender"`
	// Content contains membership state.
	Content RoomMemberContent `json:"content"`
}

// RoomMemberContent contains the content of a m.room.member event.
type RoomMemberContent struct {
	// Membership is one of: invite, join, leave, ban.
	Membership string `json:"membership"`
	// DisplayName is the user's display name.
	DisplayName string `json:"displayname,omitempty"`
	// AvatarURL is the user's avatar URL.
	AvatarURL string `json:"avatar_url,omitempty"`
}

// RoomCreateContent contains the content of a m.room.create event.
type RoomCreateContent struct {
	// Creator is the MXID of the room creator.
	Creator string `json:"creator"`
	// RoomVersion is the version of the room.
	RoomVersion string `json:"room_version"`
	// Federate indicates whether the room is federated.
	Federate bool `json:"m.federate"`
}

// RoomNameContent contains the content of a m.room.name event.
type RoomNameContent struct {
	// Name is the human-readable room name.
	Name string `json:"name"`
}

// RoomTopicContent contains the content of a m.room.topic event.
type RoomTopicContent struct {
	// Topic is the room topic.
	Topic string `json:"topic"`
}

// RoomPowerLevelsContent contains the content of a m.room.power_levels event.
type RoomPowerLevelsContent struct {
	// Ban is the power level required to ban users.
	Ban int64 `json:"ban"`
	// Kick is the power level required to kick users.
	Kick int64 `json:"kick"`
	// Redact is the power level required to redact events.
	Redact int64 `json:"redact"`
	// Invite is the power level required to invite users.
	Invite int64 `json:"invite"`
	// EventsDefault is the default power level for events.
	EventsDefault int64 `json:"events_default"`
	// UsersDefault is the default power level for users.
	UsersDefault int64 `json:"users_default"`
	// Users is a map of MXIDs to their power levels.
	Users map[string]int64 `json:"users"`
	// Events is a map of event types to their power levels.
	Events map[string]int64 `json:"events"`
}

// NewRoomPowerLevelsContent creates default power levels for a new room.
func NewRoomPowerLevelsContent(creatorMXID string) RoomPowerLevelsContent {
	return RoomPowerLevelsContent{
		Ban:           50,
		Kick:          50,
		Redact:        50,
		Invite:        0,
		EventsDefault: 0,
		UsersDefault:  0,
		Users: map[string]int64{
			creatorMXID: 100,
		},
		Events: map[string]int64{
			"m.room.name":               50,
			"m.room.power_levels":       100,
			"m.room.history_visibility": 100,
		},
	}
}
