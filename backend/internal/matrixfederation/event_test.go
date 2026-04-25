package matrixfederation

import (
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvent_Validate(t *testing.T) {
	stateKey := ""
	tests := []struct {
		name    string
		event   *Event
		wantErr bool
	}{
		{
			name: "valid create event",
			event: &Event{
				EventID:        "$abc:example.com",
				RoomID:         "!room:example.com",
				Sender:         "@user:example.com",
				Type:           EventTypeCreate,
				StateKey:       &stateKey,
				Content:        map[string]interface{}{"creator": "@user:example.com"},
				PrevEvents:     []string{},
				AuthEvents:     []string{},
				Depth:          1,
				Origin:         "example.com",
				OriginServerTS: time.Now().UnixMilli(),
				Hashes:         EventHashes{SHA256: "dummy"},
			},
			wantErr: false,
		},
		{
			name: "valid message event",
			event: &Event{
				EventID:        "$def:example.com",
				RoomID:         "!room:example.com",
				Sender:         "@alice:example.com",
				Type:           EventTypeMessage,
				Content:        map[string]interface{}{"body": "hello"},
				PrevEvents:     []string{"$abc:example.com"},
				AuthEvents:     []string{},
				Depth:          2,
				Origin:         "example.com",
				OriginServerTS: time.Now().UnixMilli(),
				Hashes:         EventHashes{SHA256: "dummy"},
			},
			wantErr: false,
		},
		{
			name: "missing event_id",
			event: &Event{
				EventID:        "",
				RoomID:         "!room:example.com",
				Sender:         "@user:example.com",
				Type:           EventTypeMessage,
				Content:        map[string]interface{}{},
				Origin:         "example.com",
				OriginServerTS: time.Now().UnixMilli(),
			},
			wantErr: true,
		},
		{
			name: "invalid event_id format",
			event: &Event{
				EventID:        "not-dollar-prefixed",
				RoomID:         "!room:example.com",
				Sender:         "@user:example.com",
				Type:           EventTypeMessage,
				Content:        map[string]interface{}{},
				Origin:         "example.com",
				OriginServerTS: time.Now().UnixMilli(),
			},
			wantErr: true,
		},
		{
			name: "missing room_id",
			event: &Event{
				EventID:        "$abc:example.com",
				RoomID:         "",
				Sender:         "@user:example.com",
				Type:           EventTypeMessage,
				Content:        map[string]interface{}{},
				Origin:         "example.com",
				OriginServerTS: time.Now().UnixMilli(),
			},
			wantErr: true,
		},
		{
			name: "invalid room_id format",
			event: &Event{
				EventID:        "$abc:example.com",
				RoomID:         "not-exclamation-prefixed",
				Sender:         "@user:example.com",
				Type:           EventTypeMessage,
				Content:        map[string]interface{}{},
				Origin:         "example.com",
				OriginServerTS: time.Now().UnixMilli(),
			},
			wantErr: true,
		},
		{
			name: "missing sender",
			event: &Event{
				EventID:        "$abc:example.com",
				RoomID:         "!room:example.com",
				Sender:         "",
				Type:           EventTypeMessage,
				Content:        map[string]interface{}{},
				Origin:         "example.com",
				OriginServerTS: time.Now().UnixMilli(),
			},
			wantErr: true,
		},
		{
			name: "invalid sender format",
			event: &Event{
				EventID:        "$abc:example.com",
				RoomID:         "!room:example.com",
				Sender:         "not-at-prefixed",
				Type:           EventTypeMessage,
				Content:        map[string]interface{}{},
				Origin:         "example.com",
				OriginServerTS: time.Now().UnixMilli(),
			},
			wantErr: true,
		},
		{
			name: "missing type",
			event: &Event{
				EventID:        "$abc:example.com",
				RoomID:         "!room:example.com",
				Sender:         "@user:example.com",
				Type:           "",
				Content:        map[string]interface{}{},
				Origin:         "example.com",
				OriginServerTS: time.Now().UnixMilli(),
			},
			wantErr: true,
		},
		{
			name: "missing origin",
			event: &Event{
				EventID:        "$abc:example.com",
				RoomID:         "!room:example.com",
				Sender:         "@user:example.com",
				Type:           EventTypeMessage,
				Content:        map[string]interface{}{},
				Origin:         "",
				OriginServerTS: time.Now().UnixMilli(),
			},
			wantErr: true,
		},
		{
			name: "missing content",
			event: &Event{
				EventID:        "$abc:example.com",
				RoomID:         "!room:example.com",
				Sender:         "@user:example.com",
				Type:           EventTypeMessage,
				Content:        nil,
				Origin:         "example.com",
				OriginServerTS: time.Now().UnixMilli(),
			},
			wantErr: true,
		},
		{
			name: "missing origin_server_ts",
			event: &Event{
				EventID: "$abc:example.com",
				RoomID:  "!room:example.com",
				Sender:  "@user:example.com",
				Type:    EventTypeMessage,
				Content: map[string]interface{}{},
				Origin:  "example.com",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.event.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEvent_IsStateEvent(t *testing.T) {
	stateKey := ""
	msgEvent := &Event{Type: EventTypeMessage}
	stateEvent := &Event{Type: EventTypeMember, StateKey: &stateKey}

	assert.False(t, msgEvent.IsStateEvent())
	assert.True(t, stateEvent.IsStateEvent())
}

func TestEvent_StateKeyString(t *testing.T) {
	stateKey := "@alice:example.com"
	event := &Event{Type: EventTypeMember, StateKey: &stateKey}
	assert.Equal(t, "@alice:example.com", event.StateKeyString())

	msgEvent := &Event{Type: EventTypeMessage}
	assert.Equal(t, "", msgEvent.StateKeyString())
}

func TestRedactContent(t *testing.T) {
	tests := []struct {
		name         string
		eventType    string
		content      map[string]interface{}
		expectedKeys []string
	}{
		{
			name:         "m.room.message - everything removed",
			eventType:    EventTypeMessage,
			content:      map[string]interface{}{"msgtype": "m.text", "body": "hello world"},
			expectedKeys: []string{},
		},
		{
			name:         "m.room.member - keeps membership",
			eventType:    EventTypeMember,
			content:      map[string]interface{}{"membership": "join", "displayname": "Alice"},
			expectedKeys: []string{"membership"},
		},
		{
			name:         "m.room.create - keeps creator and room_version",
			eventType:    EventTypeCreate,
			content:      map[string]interface{}{"creator": "@alice:example.com", "room_version": "9", "extra": "data"},
			expectedKeys: []string{"creator", "room_version"},
		},
		{
			name:         "m.room.join_rules - keeps join_rule",
			eventType:    EventTypeJoinRules,
			content:      map[string]interface{}{"join_rule": "public", "extra": "data"},
			expectedKeys: []string{"join_rule"},
		},
		{
			name:      "m.room.power_levels - keeps allowed keys",
			eventType: EventTypePowerLevels,
			content: map[string]interface{}{
				"ban": 50, "events": map[string]interface{}{}, "events_default": 0,
				"kick": 50, "redact": 100, "state_default": 50,
				"users": map[string]interface{}{}, "users_default": 0,
				"extra": "data",
			},
			expectedKeys: []string{"ban", "events", "events_default", "kick", "redact", "state_default", "users", "users_default"},
		},
		{
			name:         "m.room.canonical_alias - keeps alias and alt_aliases",
			eventType:    EventTypeCanonicalAlias,
			content:      map[string]interface{}{"alias": "#test:example.com", "alt_aliases": []string{}},
			expectedKeys: []string{"alias", "alt_aliases"},
		},
		{
			name:         "m.room.history_visibility - keeps history_visibility",
			eventType:    EventTypeHistoryVisibility,
			content:      map[string]interface{}{"history_visibility": "shared", "extra": "data"},
			expectedKeys: []string{"history_visibility"},
		},
		{
			name:         "unknown event type - keeps nothing",
			eventType:    "m.room.custom",
			content:      map[string]interface{}{"anything": "value"},
			expectedKeys: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			redacted := redactContent(tt.eventType, tt.content)
			assert.Len(t, redacted, len(tt.expectedKeys))
			for _, key := range tt.expectedKeys {
				assert.Contains(t, redacted, key, "expected key %s not found", key)
			}
		})
	}
}

func TestEvent_RedactedEvent(t *testing.T) {
	original := &Event{
		EventID:        "$abc:example.com",
		RoomID:         "!room:example.com",
		Sender:         "@bob:example.com",
		Type:           EventTypeMessage,
		Content:        map[string]interface{}{"msgtype": "m.text", "body": "hello"},
		PrevEvents:     []string{"$prev1:example.com", "$prev2:example.com"},
		AuthEvents:     []string{"$auth1:example.com"},
		Depth:          5,
		Origin:         "example.com",
		OriginServerTS: 1234567890000,
		Hashes:         EventHashes{SHA256: "abc123"},
		Signatures:     map[string]map[string]string{"example.com": {"ed25519:a": "sig"}},
		Unsigned:       map[string]interface{}{"age_ts": 1234567890000},
	}

	redacted := original.RedactedEvent()

	assert.Equal(t, original.EventID, redacted.EventID)
	assert.Equal(t, original.RoomID, redacted.RoomID)
	assert.Equal(t, original.Sender, redacted.Sender)
	assert.Equal(t, original.Type, redacted.Type)
	assert.Nil(t, redacted.StateKey)
	assert.Equal(t, original.Depth, redacted.Depth)
	assert.Equal(t, original.Origin, redacted.Origin)
	assert.Equal(t, original.OriginServerTS, redacted.OriginServerTS)
	assert.Equal(t, original.Hashes, redacted.Hashes)
	assert.Equal(t, original.Unsigned, redacted.Unsigned)

	// Content should be empty for message events
	assert.Empty(t, redacted.Content)

	// PrevEvents should be a copy
	assert.Equal(t, original.PrevEvents, redacted.PrevEvents)
	assert.NotSame(t, &original.PrevEvents[0], &redacted.PrevEvents[0])

	// Signatures should be copied
	assert.Equal(t, original.Signatures, redacted.Signatures)
	// Verify signatures map is a deep copy by modifying the copy and checking original
	redacted.Signatures["example.com"]["ed25519:b"] = "new_sig"
	assert.NotEqual(t, original.Signatures["example.com"], redacted.Signatures["example.com"])
}

func TestGenerateEventID(t *testing.T) {
	messageID := uuid.MustParse("12345678-1234-1234-1234-123456789abc")
	originServer := "example.com"

	eventID := GenerateEventID(messageID, originServer)

	assert.True(t, len(eventID) > 0)
	assert.True(t, len(eventID) < 255)
	assert.True(t, eventID[0] == '$')
	assert.Contains(t, eventID, ":example.com")

	// Should be deterministic
	eventID2 := GenerateEventID(messageID, originServer)
	assert.Equal(t, eventID, eventID2)

	// Different message ID should produce different event ID
	differentID := uuid.MustParse("abcdef12-1234-1234-1234-123456789abc")
	eventID3 := GenerateEventID(differentID, originServer)
	assert.NotEqual(t, eventID, eventID3)

	// Different origin should produce different event ID
	eventID4 := GenerateEventID(messageID, "other.com")
	assert.NotEqual(t, eventID, eventID4)
}

func TestNewMessageEvent(t *testing.T) {
	messageID := uuid.New()
	roomID, err := ParseRoomID("!test:example.com")
	require.NoError(t, err)
	sender := "@alice:example.com"
	content := "Hello, World!"
	originServer := "example.com"
	prevEvents := []string{"$prev:example.com"}
	authEvents := []string{"$auth:example.com"}
	var depth int64 = 5

	event := NewMessageEvent(messageID, roomID, sender, content, originServer, prevEvents, authEvents, depth)

	assert.NotEmpty(t, event.EventID)
	assert.True(t, event.EventID[0] == '$')
	assert.Equal(t, "!test:example.com", event.RoomID)
	assert.Equal(t, "@alice:example.com", event.Sender)
	assert.Equal(t, EventTypeMessage, event.Type)
	assert.Nil(t, event.StateKey)
	assert.Equal(t, "m.text", event.Content["msgtype"])
	assert.Equal(t, "Hello, World!", event.Content["body"])
	assert.Equal(t, prevEvents, event.PrevEvents)
	assert.Equal(t, authEvents, event.AuthEvents)
	assert.Equal(t, depth, event.Depth)
	assert.Equal(t, originServer, event.Origin)
	assert.Greater(t, event.OriginServerTS, int64(0))
	assert.Nil(t, event.Signatures)
	assert.Nil(t, event.Unsigned)
}

func TestNewMemberEvent(t *testing.T) {
	roomID, err := ParseRoomID("!room:example.com")
	require.NoError(t, err)
	sender := "@alice:example.com"
	userID := "@bob:example.com"
	membership := MembershipJoin
	originServer := "example.com"
	prevEvents := []string{"$prev:example.com"}
	authEvents := []string{"$auth:example.com"}
	var depth int64 = 3

	event := NewMemberEvent(roomID, sender, userID, membership, originServer, prevEvents, authEvents, depth)

	assert.NotEmpty(t, event.EventID)
	assert.Equal(t, "!room:example.com", event.RoomID)
	assert.Equal(t, sender, event.Sender)
	assert.Equal(t, EventTypeMember, event.Type)
	assert.NotNil(t, event.StateKey)
	assert.Equal(t, userID, *event.StateKey)
	assert.Equal(t, "join", event.Content["membership"])
	assert.Equal(t, prevEvents, event.PrevEvents)
	assert.Equal(t, authEvents, event.AuthEvents)
	assert.Equal(t, depth, event.Depth)
	assert.Equal(t, originServer, event.Origin)
}

func TestNewCreateEvent(t *testing.T) {
	roomID, err := ParseRoomID("!room:example.com")
	require.NoError(t, err)
	creator := "@alice:example.com"
	originServer := "example.com"

	event := NewCreateEvent(roomID, creator, originServer)

	assert.NotEmpty(t, event.EventID)
	assert.Equal(t, "!room:example.com", event.RoomID)
	assert.Equal(t, creator, event.Sender)
	assert.Equal(t, EventTypeCreate, event.Type)
	assert.NotNil(t, event.StateKey)
	assert.Equal(t, "", *event.StateKey)
	assert.Equal(t, creator, event.Content["creator"])
	assert.Equal(t, "9", event.Content["room_version"])
	assert.Equal(t, true, event.Content["federate"])
	assert.Empty(t, event.PrevEvents)
	assert.Empty(t, event.AuthEvents)
	assert.Equal(t, int64(1), event.Depth)
	assert.Equal(t, originServer, event.Origin)
}

func TestNewPowerLevelsEvent(t *testing.T) {
	roomID, err := ParseRoomID("!room:example.com")
	require.NoError(t, err)
	sender := "@alice:example.com"
	originServer := "example.com"
	prevEvents := []string{"$prev:example.com"}
	authEvents := []string{"$auth:example.com"}
	var depth int64 = 2

	content := RoomPowerLevelsContent{
		Ban:           50,
		Kick:          50,
		Redact:        100,
		Invite:        50,
		EventsDefault: 0,
		UsersDefault:  0,
		Users:         map[string]int64{"@alice:example.com": 100},
		Events:        map[string]int64{"m.room.message": 0},
	}

	event := NewPowerLevelsEvent(roomID, sender, originServer, content, prevEvents, authEvents, depth)

	assert.NotEmpty(t, event.EventID)
	assert.Equal(t, "!room:example.com", event.RoomID)
	assert.Equal(t, EventTypePowerLevels, event.Type)
	assert.NotNil(t, event.StateKey)
	assert.Equal(t, int64(50), event.Content["ban"])
	assert.Equal(t, int64(100), event.Content["redact"])
	assert.NotNil(t, event.Content["users"])
	assert.NotNil(t, event.Content["events"])
}

func TestEvent_EventJSONForSigning(t *testing.T) {
	stateKey := "@alice:example.com"
	event := &Event{
		EventID:        "$abc:example.com",
		RoomID:         "!room:example.com",
		Sender:         "@bob:example.com",
		Type:           EventTypeMember,
		StateKey:       &stateKey,
		Content:        map[string]interface{}{"membership": "join"},
		PrevEvents:     []string{"$prev:example.com"},
		AuthEvents:     []string{"$auth:example.com"},
		Depth:          3,
		Origin:         "example.com",
		OriginServerTS: 1234567890000,
		Hashes:         EventHashes{SHA256: "hash123"},
	}

	jsonMap, err := event.EventJSONForSigning()
	require.NoError(t, err)

	assert.Equal(t, "$abc:example.com", jsonMap["event_id"])
	assert.Equal(t, "!room:example.com", jsonMap["room_id"])
	assert.Equal(t, "@bob:example.com", jsonMap["sender"])
	assert.Equal(t, EventTypeMember, jsonMap["type"])
	assert.Equal(t, "@alice:example.com", jsonMap["state_key"])
	assert.Equal(t, map[string]interface{}{"membership": "join"}, jsonMap["content"])
	assert.Equal(t, []string{"$prev:example.com"}, jsonMap["prev_events"])
	assert.Equal(t, []string{"$auth:example.com"}, jsonMap["auth_events"])
	assert.Equal(t, int64(3), jsonMap["depth"])
	assert.Equal(t, "example.com", jsonMap["origin"])
	assert.Equal(t, int64(1234567890000), jsonMap["origin_server_ts"])

	hashes, ok := jsonMap["hashes"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "hash123", hashes["sha256"])
}

func TestEvent_EventJSONForSigning_NoStateKey(t *testing.T) {
	event := &Event{
		EventID:        "$abc:example.com",
		RoomID:         "!room:example.com",
		Sender:         "@bob:example.com",
		Type:           EventTypeMessage,
		Content:        map[string]interface{}{"body": "hello"},
		PrevEvents:     []string{},
		AuthEvents:     []string{},
		Depth:          2,
		Origin:         "example.com",
		OriginServerTS: 1234567890000,
		Hashes:         EventHashes{SHA256: "hash123"},
	}

	jsonMap, err := event.EventJSONForSigning()
	require.NoError(t, err)

	_, hasStateKey := jsonMap["state_key"]
	assert.False(t, hasStateKey, "state_key should not be present for non-state events")
}

func TestComputeContentHash(t *testing.T) {
	event := &Event{
		Content: map[string]interface{}{
			"msgtype": "m.text",
			"body":    "hello",
		},
	}

	hash1, err := event.ComputeContentHash()
	require.NoError(t, err)
	assert.NotEmpty(t, hash1)

	// Same content should produce same hash
	hash2, err := event.ComputeContentHash()
	require.NoError(t, err)
	assert.Equal(t, hash1, hash2)

	// Different content should produce different hash
	event.Content = map[string]interface{}{
		"msgtype": "m.text",
		"body":    "world",
	}
	hash3, err := event.ComputeContentHash()
	require.NoError(t, err)
	assert.NotEqual(t, hash1, hash3)
}

func TestCanonicalJSONObject(t *testing.T) {
	input := map[string]interface{}{
		"z_key": "last",
		"a_key": "first",
		"m_key": map[string]interface{}{
			"z_nested": "nested_last",
			"a_nested": "nested_first",
		},
		"arr": []interface{}{"b", "a"},
	}

	result := canonicalJSONObject(input)
	resultMap, ok := result.(map[string]interface{})
	require.True(t, ok)

	// Keys should be sorted - sort them to check since Go map iteration is random
	keys := make([]string, 0, len(resultMap))
	for k := range resultMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	assert.Equal(t, []string{"a_key", "arr", "m_key", "z_key"}, keys)

	// Nested map keys should also be sorted
	nestedMap, ok := resultMap["m_key"].(map[string]interface{})
	require.True(t, ok)
	nestedKeys := make([]string, 0, len(nestedMap))
	for k := range nestedMap {
		nestedKeys = append(nestedKeys, k)
	}
	sort.Strings(nestedKeys)
	assert.Equal(t, []string{"a_nested", "z_nested"}, nestedKeys)

	// Arrays should be preserved as-is
	arr, ok := resultMap["arr"].([]interface{})
	require.True(t, ok)
	assert.Equal(t, []interface{}{"b", "a"}, arr)
}

func TestComputeCanonicalHash(t *testing.T) {
	data := map[string]interface{}{
		"a": 1,
		"b": "test",
	}

	hash1, err := ComputeCanonicalHash(data)
	require.NoError(t, err)
	assert.NotEmpty(t, hash1)

	// Same data, different key order should produce same hash
	data2 := map[string]interface{}{
		"b": "test",
		"a": 1,
	}
	hash2, err := ComputeCanonicalHash(data2)
	require.NoError(t, err)
	assert.Equal(t, hash1, hash2)

	// Different data should produce different hash
	data3 := map[string]interface{}{
		"a": 2,
		"b": "test",
	}
	hash3, err := ComputeCanonicalHash(data3)
	require.NoError(t, err)
	assert.NotEqual(t, hash1, hash3)
}
