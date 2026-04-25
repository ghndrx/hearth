package matrixfederation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockAuthEventProvider implements AuthEventProvider for testing.
type mockAuthEventProvider struct {
	events []*Event
}

func (m *mockAuthEventProvider) GetAuthEvents(_ *Event) ([]*Event, error) {
	return m.events, nil
}

func TestAuthChecker_CheckAuthRules_ValidCreateEvent(t *testing.T) {
	checker := NewAuthChecker("example.com")

	createEvent := &Event{
		EventID:        "$abc:example.com",
		RoomID:         "!room:example.com",
		Sender:         "@alice:example.com",
		Type:           EventTypeCreate,
		StateKey:       stringPtr(""),
		Content:        map[string]interface{}{"creator": "@alice:example.com", "room_version": "9", "federate": true},
		PrevEvents:     []string{},
		AuthEvents:     []string{},
		Depth:          1,
		Origin:         "example.com",
		OriginServerTS: 1234567890000,
		Hashes:         EventHashes{SHA256: "dummy"},
	}

	provider := &mockAuthEventProvider{events: []*Event{createEvent}}
	result := checker.CheckAuthRules(createEvent, provider)
	assert.True(t, result.Allowed, "create event should be allowed: %s", result.Reason)
}

func TestAuthChecker_CheckAuthRules_MissingCreateEvent(t *testing.T) {
	checker := NewAuthChecker("example.com")

	messageEvent := &Event{
		EventID:        "$msg:example.com",
		RoomID:         "!room:example.com",
		Sender:         "@alice:example.com",
		Type:           EventTypeMessage,
		Content:        map[string]interface{}{"msgtype": "m.text", "body": "hello"},
		PrevEvents:     []string{"$abc:example.com"},
		AuthEvents:     []string{"$abc:example.com"},
		Depth:          2,
		Origin:         "example.com",
		OriginServerTS: 1234567890000,
		Hashes:         EventHashes{SHA256: "dummy"},
	}

	memberEvent := &Event{
		EventID:        "$member:example.com",
		RoomID:         "!room:example.com",
		Sender:         "@alice:example.com",
		Type:           EventTypeMember,
		StateKey:       stringPtr("@alice:example.com"),
		Content:        map[string]interface{}{"membership": "join"},
		PrevEvents:     []string{"$abc:example.com"},
		AuthEvents:     []string{"$abc:example.com"},
		Depth:          2,
		Origin:         "example.com",
		OriginServerTS: 1234567890000,
		Hashes:         EventHashes{SHA256: "dummy"},
	}

	// No create event in auth events - should fail
	provider := &mockAuthEventProvider{events: []*Event{memberEvent}}
	result := checker.CheckAuthRules(messageEvent, provider)
	assert.False(t, result.Allowed, "should not allow without create event")
	assert.Contains(t, result.Reason, "m.room.create")
}

func TestAuthChecker_CheckAuthRules_SenderNotMember(t *testing.T) {
	checker := NewAuthChecker("example.com")

	createEvent := &Event{
		EventID:        "$abc:example.com",
		RoomID:         "!room:example.com",
		Sender:         "@alice:example.com",
		Type:           EventTypeCreate,
		StateKey:       stringPtr(""),
		Content:        map[string]interface{}{"creator": "@alice:example.com", "room_version": "9"},
		Depth:          1,
		Origin:         "example.com",
		OriginServerTS: 1234567890000,
		Hashes:         EventHashes{SHA256: "dummy"},
	}

	messageEvent := &Event{
		EventID:        "$msg:example.com",
		RoomID:         "!room:example.com",
		Sender:         "@bob:example.com",
		Type:           EventTypeMessage,
		Content:        map[string]interface{}{"msgtype": "m.text", "body": "hello"},
		PrevEvents:     []string{"$abc:example.com"},
		AuthEvents:     []string{"$abc:example.com"},
		Depth:          2,
		Origin:         "example.com",
		OriginServerTS: 1234567890000,
		Hashes:         EventHashes{SHA256: "dummy"},
	}

	// No member event for @bob
	provider := &mockAuthEventProvider{events: []*Event{createEvent}}
	result := checker.CheckAuthRules(messageEvent, provider)
	assert.False(t, result.Allowed, "should not allow sender who is not a member")
	assert.Contains(t, result.Reason, "not a member")
}

func TestAuthChecker_CheckAuthRules_BannedSender(t *testing.T) {
	checker := NewAuthChecker("example.com")

	createEvent := &Event{
		EventID:        "$abc:example.com",
		RoomID:         "!room:example.com",
		Sender:         "@alice:example.com",
		Type:           EventTypeCreate,
		StateKey:       stringPtr(""),
		Content:        map[string]interface{}{"creator": "@alice:example.com", "room_version": "9"},
		Depth:          1,
		Origin:         "example.com",
		OriginServerTS: 1234567890000,
		Hashes:         EventHashes{SHA256: "dummy"},
	}

	bannedMember := &Event{
		EventID:        "$member:example.com",
		RoomID:         "!room:example.com",
		Sender:         "@bob:example.com",
		Type:           EventTypeMember,
		StateKey:       stringPtr("@bob:example.com"),
		Content:        map[string]interface{}{"membership": "ban"},
		PrevEvents:     []string{"$abc:example.com"},
		AuthEvents:     []string{"$abc:example.com"},
		Depth:          2,
		Origin:         "example.com",
		OriginServerTS: 1234567890000,
		Hashes:         EventHashes{SHA256: "dummy"},
	}

	messageEvent := &Event{
		EventID:        "$msg:example.com",
		RoomID:         "!room:example.com",
		Sender:         "@bob:example.com",
		Type:           EventTypeMessage,
		Content:        map[string]interface{}{"msgtype": "m.text", "body": "hello"},
		PrevEvents:     []string{"$abc:example.com"},
		AuthEvents:     []string{"$abc:example.com"},
		Depth:          3,
		Origin:         "example.com",
		OriginServerTS: 1234567890000,
		Hashes:         EventHashes{SHA256: "dummy"},
	}

	provider := &mockAuthEventProvider{events: []*Event{createEvent, bannedMember}}
	result := checker.CheckAuthRules(messageEvent, provider)
	assert.False(t, result.Allowed, "should not allow banned user")
	assert.Contains(t, result.Reason, "banned")
}

func TestAuthChecker_CheckAuthRules_JoinedUserCanSend(t *testing.T) {
	checker := NewAuthChecker("example.com")

	createEvent := &Event{
		EventID:        "$abc:example.com",
		RoomID:         "!room:example.com",
		Sender:         "@alice:example.com",
		Type:           EventTypeCreate,
		StateKey:       stringPtr(""),
		Content:        map[string]interface{}{"creator": "@alice:example.com", "room_version": "9"},
		Depth:          1,
		Origin:         "example.com",
		OriginServerTS: 1234567890000,
		Hashes:         EventHashes{SHA256: "dummy"},
	}

	joinedMember := &Event{
		EventID:        "$member:example.com",
		RoomID:         "!room:example.com",
		Sender:         "@alice:example.com",
		Type:           EventTypeMember,
		StateKey:       stringPtr("@alice:example.com"),
		Content:        map[string]interface{}{"membership": "join"},
		PrevEvents:     []string{"$abc:example.com"},
		AuthEvents:     []string{"$abc:example.com"},
		Depth:          2,
		Origin:         "example.com",
		OriginServerTS: 1234567890000,
		Hashes:         EventHashes{SHA256: "dummy"},
	}

	powerLevels := &Event{
		EventID:  "$pl:example.com",
		RoomID:   "!room:example.com",
		Sender:   "@alice:example.com",
		Type:     EventTypePowerLevels,
		StateKey: stringPtr(""),
		Content: map[string]interface{}{
			"users":          map[string]interface{}{"@alice:example.com": float64(100)},
			"users_default":  float64(0),
			"events_default": float64(0),
		},
		PrevEvents:     []string{"$abc:example.com", "$member:example.com"},
		AuthEvents:     []string{"$abc:example.com"},
		Depth:          3,
		Origin:         "example.com",
		OriginServerTS: 1234567890000,
		Hashes:         EventHashes{SHA256: "dummy"},
	}

	messageEvent := &Event{
		EventID:        "$msg:example.com",
		RoomID:         "!room:example.com",
		Sender:         "@alice:example.com",
		Type:           EventTypeMessage,
		Content:        map[string]interface{}{"msgtype": "m.text", "body": "hello"},
		PrevEvents:     []string{"$abc:example.com"},
		AuthEvents:     []string{"$abc:example.com"},
		Depth:          4,
		Origin:         "example.com",
		OriginServerTS: 1234567890000,
		Hashes:         EventHashes{SHA256: "dummy"},
	}

	provider := &mockAuthEventProvider{events: []*Event{createEvent, joinedMember, powerLevels}}
	result := checker.CheckAuthRules(messageEvent, provider)
	assert.True(t, result.Allowed, "joined user should be able to send: %s", result.Reason)
}

func TestAuthChecker_CheckAuthRules_InvalidCreateEvent(t *testing.T) {
	checker := NewAuthChecker("example.com")

	// Create event with depth != 1
	createEvent := &Event{
		EventID:        "$abc:example.com",
		RoomID:         "!room:example.com",
		Sender:         "@alice:example.com",
		Type:           EventTypeCreate,
		StateKey:       stringPtr(""),
		Content:        map[string]interface{}{"creator": "@alice:example.com", "room_version": "9"},
		PrevEvents:     []string{},
		AuthEvents:     []string{},
		Depth:          2, // Invalid: should be 1
		Origin:         "example.com",
		OriginServerTS: 1234567890000,
		Hashes:         EventHashes{SHA256: "dummy"},
	}

	provider := &mockAuthEventProvider{events: []*Event{createEvent}}
	result := checker.CheckAuthRules(createEvent, provider)
	assert.False(t, result.Allowed, "create event with wrong depth should not be allowed")
	assert.Contains(t, result.Reason, "depth must be 1")
}

func TestAuthChecker_CheckAuthRules_CreateEventWithPrevEvents(t *testing.T) {
	checker := NewAuthChecker("example.com")

	createEvent := &Event{
		EventID:        "$abc:example.com",
		RoomID:         "!room:example.com",
		Sender:         "@alice:example.com",
		Type:           EventTypeCreate,
		StateKey:       stringPtr(""),
		Content:        map[string]interface{}{"creator": "@alice:example.com", "room_version": "9"},
		PrevEvents:     []string{"$prev:example.com"}, // Invalid: should be empty
		AuthEvents:     []string{},
		Depth:          1,
		Origin:         "example.com",
		OriginServerTS: 1234567890000,
		Hashes:         EventHashes{SHA256: "dummy"},
	}

	provider := &mockAuthEventProvider{events: []*Event{createEvent}}
	result := checker.CheckAuthRules(createEvent, provider)
	assert.False(t, result.Allowed, "create event with prev_events should not be allowed")
	assert.Contains(t, result.Reason, "must not have prev_events")
}

func TestAuthChecker_CheckAuthRules_InvalidEvent(t *testing.T) {
	checker := NewAuthChecker("example.com")

	// Missing required fields
	event := &Event{
		EventID:        "invalid", // Missing $ prefix
		RoomID:         "!room:example.com",
		Sender:         "@alice:example.com",
		Type:           EventTypeMessage,
		Content:        map[string]interface{}{},
		Origin:         "example.com",
		OriginServerTS: 1234567890000,
		Hashes:         EventHashes{SHA256: "dummy"},
	}

	provider := &mockAuthEventProvider{events: []*Event{}}
	result := checker.CheckAuthRules(event, provider)
	assert.False(t, result.Allowed, "invalid event should not be allowed")
	assert.Contains(t, result.Reason, "invalid event")
}

func TestAuthChecker_CheckAuthRules_PowerLevelsRequireHighPower(t *testing.T) {
	checker := NewAuthChecker("example.com")

	createEvent := &Event{
		EventID:        "$abc:example.com",
		RoomID:         "!room:example.com",
		Sender:         "@alice:example.com",
		Type:           EventTypeCreate,
		StateKey:       stringPtr(""),
		Content:        map[string]interface{}{"creator": "@alice:example.com", "room_version": "9"},
		Depth:          1,
		Origin:         "example.com",
		OriginServerTS: 1234567890000,
		Hashes:         EventHashes{SHA256: "dummy"},
	}

	joinedMember := &Event{
		EventID:        "$member:example.com",
		RoomID:         "!room:example.com",
		Sender:         "@bob:example.com",
		Type:           EventTypeMember,
		StateKey:       stringPtr("@bob:example.com"),
		Content:        map[string]interface{}{"membership": "join"},
		PrevEvents:     []string{"$abc:example.com"},
		AuthEvents:     []string{"$abc:example.com"},
		Depth:          2,
		Origin:         "example.com",
		OriginServerTS: 1234567890000,
		Hashes:         EventHashes{SHA256: "dummy"},
	}

	existingPowerLevels := &Event{
		EventID:  "$pl:example.com",
		RoomID:   "!room:example.com",
		Sender:   "@alice:example.com",
		Type:     EventTypePowerLevels,
		StateKey: stringPtr(""),
		Content: map[string]interface{}{
			"users":          map[string]interface{}{"@alice:example.com": float64(100), "@bob:example.com": float64(50)},
			"users_default":  float64(0),
			"state_default":  float64(100), // Requires 100 power to change state
			"events_default": float64(0),
		},
		PrevEvents:     []string{"$abc:example.com", "$member:example.com"},
		AuthEvents:     []string{"$abc:example.com"},
		Depth:          3,
		Origin:         "example.com",
		OriginServerTS: 1234567890000,
		Hashes:         EventHashes{SHA256: "dummy"},
	}

	// Bob tries to change power levels - he only has 50 power but needs 100
	newPowerLevels := &Event{
		EventID:        "$newpl:example.com",
		RoomID:         "!room:example.com",
		Sender:         "@bob:example.com",
		Type:           EventTypePowerLevels,
		StateKey:       stringPtr(""),
		Content:        map[string]interface{}{"users": map[string]interface{}{"@bob:example.com": float64(200)}},
		PrevEvents:     []string{"$pl:example.com"},
		AuthEvents:     []string{"$abc:example.com", "$pl:example.com"},
		Depth:          4,
		Origin:         "example.com",
		OriginServerTS: 1234567890000,
		Hashes:         EventHashes{SHA256: "dummy"},
	}

	provider := &mockAuthEventProvider{events: []*Event{createEvent, joinedMember, existingPowerLevels}}
	result := checker.CheckAuthRules(newPowerLevels, provider)
	assert.False(t, result.Allowed, "user without sufficient power should not change power levels")
	assert.Contains(t, result.Reason, "power level")
}

func TestAuthChecker_CheckAuthRules_JoinRulesEvent(t *testing.T) {
	checker := NewAuthChecker("example.com")

	createEvent := &Event{
		EventID:        "$abc:example.com",
		RoomID:         "!room:example.com",
		Sender:         "@alice:example.com",
		Type:           EventTypeCreate,
		StateKey:       stringPtr(""),
		Content:        map[string]interface{}{"creator": "@alice:example.com", "room_version": "9"},
		Depth:          1,
		Origin:         "example.com",
		OriginServerTS: 1234567890000,
		Hashes:         EventHashes{SHA256: "dummy"},
	}

	joinedMember := &Event{
		EventID:        "$member:example.com",
		RoomID:         "!room:example.com",
		Sender:         "@alice:example.com",
		Type:           EventTypeMember,
		StateKey:       stringPtr("@alice:example.com"),
		Content:        map[string]interface{}{"membership": "join"},
		PrevEvents:     []string{"$abc:example.com"},
		AuthEvents:     []string{"$abc:example.com"},
		Depth:          2,
		Origin:         "example.com",
		OriginServerTS: 1234567890000,
		Hashes:         EventHashes{SHA256: "dummy"},
	}

	powerLevels := &Event{
		EventID:  "$pl:example.com",
		RoomID:   "!room:example.com",
		Sender:   "@alice:example.com",
		Type:     EventTypePowerLevels,
		StateKey: stringPtr(""),
		Content: map[string]interface{}{
			"users":          map[string]interface{}{"@alice:example.com": float64(100)},
			"users_default":  float64(0),
			"state_default":  float64(50),
			"events_default": float64(0),
		},
		PrevEvents:     []string{"$abc:example.com", "$member:example.com"},
		AuthEvents:     []string{"$abc:example.com"},
		Depth:          3,
		Origin:         "example.com",
		OriginServerTS: 1234567890000,
		Hashes:         EventHashes{SHA256: "dummy"},
	}

	// Test valid join_rules event
	joinRules := &Event{
		EventID:        "$jr:example.com",
		RoomID:         "!room:example.com",
		Sender:         "@alice:example.com",
		Type:           EventTypeJoinRules,
		StateKey:       stringPtr(""),
		Content:        map[string]interface{}{"join_rule": "public"},
		PrevEvents:     []string{"$pl:example.com"},
		AuthEvents:     []string{"$abc:example.com", "$pl:example.com"},
		Depth:          4,
		Origin:         "example.com",
		OriginServerTS: 1234567890000,
		Hashes:         EventHashes{SHA256: "dummy"},
	}

	provider := &mockAuthEventProvider{events: []*Event{createEvent, joinedMember, powerLevels}}
	result := checker.CheckAuthRules(joinRules, provider)
	assert.True(t, result.Allowed, "valid join_rules event should be allowed: %s", result.Reason)
}

func TestAuthChecker_CheckAuthRules_InvalidJoinRule(t *testing.T) {
	checker := NewAuthChecker("example.com")

	createEvent := &Event{
		EventID:        "$abc:example.com",
		RoomID:         "!room:example.com",
		Sender:         "@alice:example.com",
		Type:           EventTypeCreate,
		StateKey:       stringPtr(""),
		Content:        map[string]interface{}{"creator": "@alice:example.com", "room_version": "9"},
		Depth:          1,
		Origin:         "example.com",
		OriginServerTS: 1234567890000,
		Hashes:         EventHashes{SHA256: "dummy"},
	}

	joinedMember := &Event{
		EventID:        "$member:example.com",
		RoomID:         "!room:example.com",
		Sender:         "@alice:example.com",
		Type:           EventTypeMember,
		StateKey:       stringPtr("@alice:example.com"),
		Content:        map[string]interface{}{"membership": "join"},
		PrevEvents:     []string{"$abc:example.com"},
		AuthEvents:     []string{"$abc:example.com"},
		Depth:          2,
		Origin:         "example.com",
		OriginServerTS: 1234567890000,
		Hashes:         EventHashes{SHA256: "dummy"},
	}

	powerLevels := &Event{
		EventID:  "$pl:example.com",
		RoomID:   "!room:example.com",
		Sender:   "@alice:example.com",
		Type:     EventTypePowerLevels,
		StateKey: stringPtr(""),
		Content: map[string]interface{}{
			"users":          map[string]interface{}{"@alice:example.com": float64(100)},
			"users_default":  float64(0),
			"state_default":  float64(50),
			"events_default": float64(0),
		},
		PrevEvents:     []string{"$abc:example.com", "$member:example.com"},
		AuthEvents:     []string{"$abc:example.com"},
		Depth:          3,
		Origin:         "example.com",
		OriginServerTS: 1234567890000,
		Hashes:         EventHashes{SHA256: "dummy"},
	}

	// Invalid join_rule value
	joinRules := &Event{
		EventID:        "$jr:example.com",
		RoomID:         "!room:example.com",
		Sender:         "@alice:example.com",
		Type:           EventTypeJoinRules,
		StateKey:       stringPtr(""),
		Content:        map[string]interface{}{"join_rule": "invalid_value"},
		PrevEvents:     []string{"$pl:example.com"},
		AuthEvents:     []string{"$abc:example.com", "$pl:example.com"},
		Depth:          4,
		Origin:         "example.com",
		OriginServerTS: 1234567890000,
		Hashes:         EventHashes{SHA256: "dummy"},
	}

	provider := &mockAuthEventProvider{events: []*Event{createEvent, joinedMember, powerLevels}}
	result := checker.CheckAuthRules(joinRules, provider)
	assert.False(t, result.Allowed, "invalid join_rule should not be allowed")
	assert.Contains(t, result.Reason, "invalid join_rule")
}
