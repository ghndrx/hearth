// Package matrixfederation implements Matrix Federation protocol support for Hearth.
// This file implements the Matrix Event PDU format for Room Version 9.
//
// Matrix Spec References:
//   - Room Version 9: https://spec.matrix.org/v1.16/rooms/v9/
//   - Event Format: https://spec.matrix.org/v1.16/appendices/#event-format
//   - Signing Events: https://spec.matrix.org/v1.16/appendices/#signing-events
package matrixfederation

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Common errors for event operations.
var (
	ErrInvalidEventFormat   = errors.New("matrix: invalid event format")
	ErrInvalidEventID       = errors.New("matrix: invalid event ID format")
	ErrInvalidSender        = errors.New("matrix: invalid sender in event")
	ErrMissingRequiredField = errors.New("matrix: missing required event field")
	ErrHashMismatch         = errors.New("matrix: event content hash mismatch")
	ErrSignatureMismatch    = errors.New("matrix: event signature verification failed")
)

// EventType constants for Matrix room events.
const (
	EventTypeCreate            = "m.room.create"
	EventTypeMember            = "m.room.member"
	EventTypePowerLevels       = "m.room.power_levels"
	EventTypeJoinRules         = "m.room.join_rules"
	EventTypeName              = "m.room.name"
	EventTypeTopic             = "m.room.topic"
	EventTypeMessage           = "m.room.message"
	EventTypeRedaction         = "m.room.redaction"
	EventTypeHistoryVisibility = "m.room.history_visibility"
	EventTypeCanonicalAlias    = "m.room.canonical_alias"
	EventTypeEncryption        = "m.room.encryption"
	EventTypeAvatar            = "m.room.avatar"
)

// Membership constants.
const (
	MembershipInvite = "invite"
	MembershipJoin   = "join"
	MembershipLeave  = "leave"
	MembershipBan    = "ban"
	MembershipKnock  = "knock"
)

// MessageType constants for m.room.message events.
const (
	MsgTypeText     = "m.text"
	MsgTypeEmote    = "m.emote"
	MsgTypeNotice   = "m.notice"
	MsgTypeImage    = "m.image"
	MsgTypeFile     = "m.file"
	MsgTypeAudio    = "m.audio"
	MsgTypeVideo    = "m.video"
	MsgTypeLocation = "m.location"
)

// Event represents a Matrix room event PDU.
// This format is compatible with Room Version 9.
type Event struct {
	// EventID is the globally unique event identifier.
	// Format: $base64url_hash:origin_server
	EventID string `json:"event_id"`

	// RoomID is the room this event belongs to.
	// Format: !localpart:server_name
	RoomID string `json:"room_id"`

	// Sender is the MXID of the user who sent this event.
	// Format: @localpart:server_name
	Sender string `json:"sender"`

	// Type is the event type (e.g., "m.room.message").
	Type string `json:"type"`

	// StateKey is present for state events. Nil for message events.
	StateKey *string `json:"state_key,omitempty"`

	// Content is the event content (varies by type).
	Content map[string]interface{} `json:"content"`

	// PrevEvents are the event IDs that this event references as predecessors.
	PrevEvents []string `json:"prev_events"`

	// AuthEvents are the event IDs that this event uses for authorization.
	AuthEvents []string `json:"auth_events"`

	// Depth is the depth of this event in the DAG (create=1, etc.).
	Depth int64 `json:"depth"`

	// Origin is the server name of the sending server.
	Origin string `json:"origin"`

	// OriginServerTS is the timestamp when the event was created (ms since epoch).
	OriginServerTS int64 `json:"origin_server_ts"`

	// Hashes contains content hashes for integrity verification.
	Hashes EventHashes `json:"hashes"`

	// Signatures contains server signatures over this event.
	// Map: server_name -> key_id -> signature_base64
	Signatures map[string]map[string]string `json:"signatures,omitempty"`

	// Unsigned contains data that is not signed (e.g., age, transaction_id).
	Unsigned map[string]interface{} `json:"unsigned,omitempty"`
}

// EventHashes contains the content hash for an event.
type EventHashes struct {
	// SHA256 is the base64-encoded SHA-256 hash of the canonical JSON content.
	SHA256 string `json:"sha256"`
}

// IsStateEvent returns true if this is a state event (has a state_key).
func (e *Event) IsStateEvent() bool {
	return e.StateKey != nil
}

// StateKeyString returns the state key string, or empty if not a state event.
func (e *Event) StateKeyString() string {
	if e.StateKey == nil {
		return ""
	}
	return *e.StateKey
}

// Validate checks that the event has all required fields.
func (e *Event) Validate() error {
	if e.EventID == "" {
		return fmt.Errorf("%w: event_id is required", ErrMissingRequiredField)
	}
	if !strings.HasPrefix(e.EventID, "$") {
		return fmt.Errorf("%w: event_id must start with $", ErrInvalidEventID)
	}
	if e.RoomID == "" {
		return fmt.Errorf("%w: room_id is required", ErrMissingRequiredField)
	}
	if !strings.HasPrefix(e.RoomID, "!") {
		return fmt.Errorf("%w: room_id must start with !", ErrInvalidRoomID)
	}
	if e.Sender == "" {
		return fmt.Errorf("%w: sender is required", ErrMissingRequiredField)
	}
	if !strings.HasPrefix(e.Sender, "@") {
		return fmt.Errorf("%w: sender must start with @", ErrInvalidSender)
	}
	if e.Type == "" {
		return fmt.Errorf("%w: type is required", ErrMissingRequiredField)
	}
	if e.Origin == "" {
		return fmt.Errorf("%w: origin is required", ErrMissingRequiredField)
	}
	if e.Content == nil {
		return fmt.Errorf("%w: content is required", ErrMissingRequiredField)
	}
	if e.OriginServerTS == 0 {
		return fmt.Errorf("%w: origin_server_ts is required", ErrMissingRequiredField)
	}
	return nil
}

// RedactedEvent returns a redacted copy of the event.
// Per Matrix spec, only specific fields remain after redaction.
func (e *Event) RedactedEvent() *Event {
	redacted := &Event{
		EventID:        e.EventID,
		RoomID:         e.RoomID,
		Sender:         e.Sender,
		Type:           e.Type,
		StateKey:       e.StateKey,
		PrevEvents:     append([]string(nil), e.PrevEvents...),
		AuthEvents:     append([]string(nil), e.AuthEvents...),
		Depth:          e.Depth,
		Origin:         e.Origin,
		OriginServerTS: e.OriginServerTS,
		Hashes:         e.Hashes,
		Signatures:     make(map[string]map[string]string),
		Unsigned:       e.Unsigned,
	}

	// Copy signatures
	for server, keys := range e.Signatures {
		redacted.Signatures[server] = make(map[string]string)
		for keyID, sig := range keys {
			redacted.Signatures[server][keyID] = sig
		}
	}

	// Redacted content only keeps allowed keys per event type
	redacted.Content = redactContent(e.Type, e.Content)

	return redacted
}

// redactContent returns redacted content with only allowed keys.
func redactContent(eventType string, content map[string]interface{}) map[string]interface{} {
	redacted := make(map[string]interface{})

	// Per Matrix spec § 9.7: allowed keys that survive redaction
	switch eventType {
	case EventTypeMember:
		// m.room.member keeps membership
		if v, ok := content["membership"]; ok {
			redacted["membership"] = v
		}
	case EventTypeCreate:
		// m.room.create keeps creator
		if v, ok := content["creator"]; ok {
			redacted["creator"] = v
		}
		if v, ok := content["room_version"]; ok {
			redacted["room_version"] = v
		}
	case EventTypeJoinRules:
		if v, ok := content["join_rule"]; ok {
			redacted["join_rule"] = v
		}
	case EventTypePowerLevels:
		allowed := []string{"ban", "events", "events_default", "kick", "redact",
			"state_default", "users", "users_default"}
		for _, key := range allowed {
			if v, ok := content[key]; ok {
				redacted[key] = v
			}
		}
	case EventTypeCanonicalAlias:
		if v, ok := content["alias"]; ok {
			redacted["alias"] = v
		}
		if v, ok := content["alt_aliases"]; ok {
			redacted["alt_aliases"] = v
		}
	case EventTypeHistoryVisibility:
		if v, ok := content["history_visibility"]; ok {
			redacted["history_visibility"] = v
		}
	case EventTypeMessage:
		// Message events keep nothing after redaction
		// (the body is completely removed)
	default:
		// For unknown types, keep nothing
	}

	return redacted
}

// ComputeContentHash computes the SHA-256 hash of the canonical JSON content.
// This is used for the Hashes.SHA256 field.
func (e *Event) ComputeContentHash() (string, error) {
	canonical, err := eventCanonicalJSON(e.Content)
	if err != nil {
		return "", fmt.Errorf("failed to canonicalize content: %w", err)
	}

	hash := sha256.Sum256(canonical)
	return base64.RawStdEncoding.EncodeToString(hash[:]), nil
}

// VerifyContentHash verifies that the content hash matches the stored hash.
func (e *Event) VerifyContentHash() error {
	computed, err := e.ComputeContentHash()
	if err != nil {
		return err
	}

	if e.Hashes.SHA256 != computed {
		return fmt.Errorf("%w: computed=%q stored=%q", ErrHashMismatch, computed, e.Hashes.SHA256)
	}

	return nil
}

// SignWithServer signs the event using the provided server name and signing key.
// This is the correct method as it knows the server name.
func (e *Event) SignWithServer(serverName string, key *SigningKey) error {
	if !key.CanSign() {
		return ErrKeyExpired
	}

	// Create a copy for signing (without signatures and unsigned)
	signingCopy := map[string]interface{}{
		"event_id":         e.EventID,
		"room_id":          e.RoomID,
		"sender":           e.Sender,
		"type":             e.Type,
		"content":          e.Content,
		"prev_events":      e.PrevEvents,
		"auth_events":      e.AuthEvents,
		"depth":            e.Depth,
		"origin":           e.Origin,
		"origin_server_ts": e.OriginServerTS,
		"hashes": map[string]interface{}{
			"sha256": e.Hashes.SHA256,
		},
	}

	if e.StateKey != nil {
		signingCopy["state_key"] = *e.StateKey
	}

	canonical, err := eventCanonicalJSON(signingCopy)
	if err != nil {
		return fmt.Errorf("failed to canonicalize event for signing: %w", err)
	}

	sig, err := key.Sign(canonical)
	if err != nil {
		return fmt.Errorf("failed to sign event: %w", err)
	}

	if e.Signatures == nil {
		e.Signatures = make(map[string]map[string]string)
	}
	if e.Signatures[serverName] == nil {
		e.Signatures[serverName] = make(map[string]string)
	}

	e.Signatures[serverName][key.KeyID] = sig
	return nil
}

// VerifySignature verifies a signature on this event.
func (e *Event) VerifySignature(serverName, keyID string, key *SigningKey) error {
	if e.Signatures == nil {
		return fmt.Errorf("%w: no signatures present", ErrSignatureMismatch)
	}

	serverSigs, ok := e.Signatures[serverName]
	if !ok {
		return fmt.Errorf("%w: no signature from server %s", ErrSignatureMismatch, serverName)
	}

	sig, ok := serverSigs[keyID]
	if !ok {
		return fmt.Errorf("%w: no signature with key %s", ErrSignatureMismatch, keyID)
	}

	// Rebuild the signing copy
	signingCopy := map[string]interface{}{
		"event_id":         e.EventID,
		"room_id":          e.RoomID,
		"sender":           e.Sender,
		"type":             e.Type,
		"content":          e.Content,
		"prev_events":      e.PrevEvents,
		"auth_events":      e.AuthEvents,
		"depth":            e.Depth,
		"origin":           e.Origin,
		"origin_server_ts": e.OriginServerTS,
		"hashes": map[string]interface{}{
			"sha256": e.Hashes.SHA256,
		},
	}

	if e.StateKey != nil {
		signingCopy["state_key"] = *e.StateKey
	}

	canonical, err := eventCanonicalJSON(signingCopy)
	if err != nil {
		return fmt.Errorf("failed to canonicalize event for verification: %w", err)
	}

	if err := key.Verify(canonical, sig); err != nil {
		return fmt.Errorf("%w: %v", ErrSignatureMismatch, err)
	}

	return nil
}

// GenerateEventID creates a deterministic Matrix event ID from a message UUID.
// Format: $base64url(hash):originServer
func GenerateEventID(messageID uuid.UUID, originServer string) string {
	data := append(messageID[:], []byte(originServer)...)
	hash := sha256.Sum256(data)
	b64 := base64.RawURLEncoding.EncodeToString(hash[:16])
	return "$" + b64 + ":" + originServer
}

// NewMessageEvent creates an m.room.message event from a Hearth message.
func NewMessageEvent(messageID uuid.UUID, roomID RoomID, senderMXID string, content string, originServer string, prevEvents []string, authEvents []string, depth int64) *Event {
	eventID := GenerateEventID(messageID, originServer)

	return &Event{
		EventID: eventID,
		RoomID:  roomID.String(),
		Sender:  senderMXID,
		Type:    EventTypeMessage,
		Content: map[string]interface{}{
			"msgtype": MsgTypeText,
			"body":    content,
		},
		PrevEvents:     append([]string(nil), prevEvents...),
		AuthEvents:     append([]string(nil), authEvents...),
		Depth:          depth,
		Origin:         originServer,
		OriginServerTS: time.Now().UnixMilli(),
	}
}

// NewMemberEvent creates an m.room.member event.
func NewMemberEvent(roomID RoomID, sender, userID, membership, originServer string, prevEvents, authEvents []string, depth int64) *Event {
	eventID := GenerateEventID(uuid.New(), originServer)

	return &Event{
		EventID:  eventID,
		RoomID:   roomID.String(),
		Sender:   sender,
		Type:     EventTypeMember,
		StateKey: &userID,
		Content: map[string]interface{}{
			"membership": membership,
		},
		PrevEvents:     append([]string(nil), prevEvents...),
		AuthEvents:     append([]string(nil), authEvents...),
		Depth:          depth,
		Origin:         originServer,
		OriginServerTS: time.Now().UnixMilli(),
	}
}

// NewCreateEvent creates an m.room.create event (the first event in a room).
func NewCreateEvent(roomID RoomID, creatorMXID, originServer string) *Event {
	eventID := GenerateEventID(uuid.New(), originServer)

	return &Event{
		EventID:  eventID,
		RoomID:   roomID.String(),
		Sender:   creatorMXID,
		Type:     EventTypeCreate,
		StateKey: stringPtr(""),
		Content: map[string]interface{}{
			"creator":      creatorMXID,
			"room_version": "9",
			"federate":     true,
		},
		PrevEvents:     []string{},
		AuthEvents:     []string{},
		Depth:          1,
		Origin:         originServer,
		OriginServerTS: time.Now().UnixMilli(),
	}
}

// NewPowerLevelsEvent creates an m.room.power_levels event.
func NewPowerLevelsEvent(roomID RoomID, sender, originServer string, content RoomPowerLevelsContent, prevEvents, authEvents []string, depth int64) *Event {
	eventID := GenerateEventID(uuid.New(), originServer)

	// Convert content to map
	contentMap := map[string]interface{}{
		"ban":            content.Ban,
		"kick":           content.Kick,
		"redact":         content.Redact,
		"invite":         content.Invite,
		"events_default": content.EventsDefault,
		"users_default":  content.UsersDefault,
		"state_default":  content.StateDefault,
		"users":          content.Users,
		"events":         content.Events,
	}

	return &Event{
		EventID:        eventID,
		RoomID:         roomID.String(),
		Sender:         sender,
		Type:           EventTypePowerLevels,
		StateKey:       stringPtr(""),
		Content:        contentMap,
		PrevEvents:     append([]string(nil), prevEvents...),
		AuthEvents:     append([]string(nil), authEvents...),
		Depth:          depth,
		Origin:         originServer,
		OriginServerTS: time.Now().UnixMilli(),
	}
}

// EventJSONForSigning returns the canonical JSON representation for signing.
// This excludes signatures and unsigned fields.
func (e *Event) EventJSONForSigning() (map[string]interface{}, error) {
	obj := map[string]interface{}{
		"event_id":         e.EventID,
		"room_id":          e.RoomID,
		"sender":           e.Sender,
		"type":             e.Type,
		"content":          e.Content,
		"prev_events":      e.PrevEvents,
		"auth_events":      e.AuthEvents,
		"depth":            e.Depth,
		"origin":           e.Origin,
		"origin_server_ts": e.OriginServerTS,
		"hashes": map[string]interface{}{
			"sha256": e.Hashes.SHA256,
		},
	}

	if e.StateKey != nil {
		obj["state_key"] = *e.StateKey
	}

	return obj, nil
}

// stringPtr is a helper to get a string pointer.
func stringPtr(s string) *string {
	return &s
}

// canonicalJSON produces canonical JSON encoding per Matrix spec.
// Keys are sorted alphabetically, no extra whitespace.
// This is a simplified version that delegates to json.Marshal which
// naturally sorts map keys alphabetically.
func eventCanonicalJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// canonicalJSONObject recursively sorts all map keys in a JSON object.
// This ensures consistent canonical encoding.
func canonicalJSONObject(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		// Sort keys
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		result := make(map[string]interface{}, len(val))
		for _, k := range keys {
			result[k] = canonicalJSONObject(val[k])
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(val))
		for i, item := range val {
			result[i] = canonicalJSONObject(item)
		}
		return result
	default:
		return val
	}
}

// ComputeCanonicalHash computes the SHA-256 hash of the canonical JSON encoding.
func ComputeCanonicalHash(v interface{}) (string, error) {
	canonical := canonicalJSONObject(v)
	data, err := json.Marshal(canonical)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)
	return base64.RawStdEncoding.EncodeToString(hash[:]), nil
}
