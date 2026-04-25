// Package matrixfederation implements Matrix Federation protocol support for Hearth.
// This file tests the in-memory room state tracking.
package matrixfederation

import (
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFederatedRoomState(t *testing.T) {
	roomID := RoomID{Localpart: "testroom", ServerName: "example.com"}
	state := NewFederatedRoomState(roomID)

	assert.Equal(t, roomID, state.RoomID)
	assert.NotNil(t, state.Members)
	assert.Empty(t, state.Members)
	assert.NotNil(t, state.ForwardExtremities)
	assert.Empty(t, state.ForwardExtremities)
	assert.Equal(t, int64(0), state.CurrentDepth)
}

func TestFederatedRoomState_AddEvent_Nil(t *testing.T) {
	roomID := RoomID{Localpart: "testroom", ServerName: "example.com"}
	state := NewFederatedRoomState(roomID)

	err := state.AddEvent(nil)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidEventFormat)
}

func TestFederatedRoomState_AddEvent_WrongRoom(t *testing.T) {
	roomID := RoomID{Localpart: "testroom", ServerName: "example.com"}
	state := NewFederatedRoomState(roomID)

	event := &Event{
		EventID: "$abc:other.com",
		RoomID:  "!otherroom:other.com",
		Type:    EventTypeMessage,
	}

	err := state.AddEvent(event)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not match state room")
}

func TestFederatedRoomState_AddEvent_Member(t *testing.T) {
	roomID := RoomID{Localpart: "testroom", ServerName: "example.com"}
	state := NewFederatedRoomState(roomID)

	userID := "@alice:example.com"
	event := NewMemberEvent(roomID, "@admin:example.com", userID, MembershipJoin, "example.com", []string{}, []string{}, 1)

	err := state.AddEvent(event)
	require.NoError(t, err)

	assert.True(t, state.HasMember(userID))
	assert.Equal(t, MembershipJoin, state.Members[userID])

	members := state.GetMembers()
	assert.Equal(t, MembershipJoin, members[userID])

	// Update membership to leave.
	leaveEvent := NewMemberEvent(roomID, userID, userID, MembershipLeave, "example.com", []string{event.EventID}, []string{}, 2)
	err = state.AddEvent(leaveEvent)
	require.NoError(t, err)

	assert.Equal(t, MembershipLeave, state.Members[userID])
}

func TestFederatedRoomState_AddEvent_PowerLevels(t *testing.T) {
	roomID := RoomID{Localpart: "testroom", ServerName: "example.com"}
	state := NewFederatedRoomState(roomID)

	pl := RoomPowerLevelsContent{
		Ban:           50,
		Kick:          50,
		Redact:        50,
		Invite:        0,
		EventsDefault: 0,
		UsersDefault:  0,
		Users: map[string]int64{
			"@admin:example.com": 100,
			"@mod:example.com":   50,
		},
		Events: map[string]int64{
			EventTypePowerLevels: 100,
		},
	}

	createEvent := NewCreateEvent(roomID, "@admin:example.com", "example.com")
	_ = state.AddEvent(createEvent)

	plEvent := NewPowerLevelsEvent(roomID, "@admin:example.com", "example.com", pl, []string{createEvent.EventID}, []string{}, 2)
	err := state.AddEvent(plEvent)
	require.NoError(t, err)

	assert.Equal(t, int64(100), state.GetMemberPower("@admin:example.com"))
	assert.Equal(t, int64(50), state.GetMemberPower("@mod:example.com"))
	assert.Equal(t, int64(0), state.GetMemberPower("@nobody:example.com"))
}

func TestFederatedRoomState_AddEvent_ForwardExtremities(t *testing.T) {
	roomID := RoomID{Localpart: "testroom", ServerName: "example.com"}
	state := NewFederatedRoomState(roomID)

	e1 := NewCreateEvent(roomID, "@admin:example.com", "example.com")
	err := state.AddEvent(e1)
	require.NoError(t, err)

	extremities := state.GetForwardExtremities()
	require.Len(t, extremities, 1)
	assert.Contains(t, extremities, e1.EventID)

	e2 := NewMemberEvent(roomID, "@admin:example.com", "@alice:example.com", MembershipJoin, "example.com", []string{e1.EventID}, []string{}, 2)
	err = state.AddEvent(e2)
	require.NoError(t, err)

	extremities = state.GetForwardExtremities()
	require.Len(t, extremities, 1)
	assert.Contains(t, extremities, e2.EventID)
	assert.NotContains(t, extremities, e1.EventID)

	// Branch: two events referencing the same prev_event.
	e3a := NewMemberEvent(roomID, "@admin:example.com", "@bob:example.com", MembershipJoin, "example.com", []string{e2.EventID}, []string{}, 3)
	e3b := NewMemberEvent(roomID, "@admin:example.com", "@charlie:example.com", MembershipJoin, "example.com", []string{e2.EventID}, []string{}, 3)

	err = state.AddEvent(e3a)
	require.NoError(t, err)
	err = state.AddEvent(e3b)
	require.NoError(t, err)

	extremities = state.GetForwardExtremities()
	require.Len(t, extremities, 2)
	assert.Contains(t, extremities, e3a.EventID)
	assert.Contains(t, extremities, e3b.EventID)
	assert.NotContains(t, extremities, e2.EventID)
}

func TestFederatedRoomState_AddEvent_CurrentDepth(t *testing.T) {
	roomID := RoomID{Localpart: "testroom", ServerName: "example.com"}
	state := NewFederatedRoomState(roomID)

	assert.Equal(t, int64(0), state.GetCurrentDepth())

	e1 := NewCreateEvent(roomID, "@admin:example.com", "example.com")
	_ = state.AddEvent(e1)
	assert.Equal(t, int64(1), state.GetCurrentDepth())

	e2 := NewMemberEvent(roomID, "@admin:example.com", "@alice:example.com", MembershipJoin, "example.com", []string{e1.EventID}, []string{}, 5)
	_ = state.AddEvent(e2)
	assert.Equal(t, int64(5), state.GetCurrentDepth())

	// Adding a lower-depth event should not reduce current depth.
	e3 := NewMemberEvent(roomID, "@admin:example.com", "@bob:example.com", MembershipJoin, "example.com", []string{e2.EventID}, []string{}, 3)
	_ = state.AddEvent(e3)
	assert.Equal(t, int64(5), state.GetCurrentDepth())
}

func TestFederatedRoomState_GetStateEvent(t *testing.T) {
	roomID := RoomID{Localpart: "testroom", ServerName: "example.com"}
	state := NewFederatedRoomState(roomID)

	// Not found before adding.
	_, err := state.GetStateEvent(EventTypeCreate, "")
	assert.Error(t, err)

	createEvent := NewCreateEvent(roomID, "@admin:example.com", "example.com")
	err = state.AddEvent(createEvent)
	require.NoError(t, err)

	retrieved, err := state.GetStateEvent(EventTypeCreate, "")
	require.NoError(t, err)
	assert.Equal(t, createEvent.EventID, retrieved.EventID)

	// Add a member event and retrieve it.
	memberEvent := NewMemberEvent(roomID, "@admin:example.com", "@alice:example.com", MembershipJoin, "example.com", []string{createEvent.EventID}, []string{}, 2)
	err = state.AddEvent(memberEvent)
	require.NoError(t, err)

	retrieved, err = state.GetStateEvent(EventTypeMember, "@alice:example.com")
	require.NoError(t, err)
	assert.Equal(t, memberEvent.EventID, retrieved.EventID)

	// Updating a state event should replace the previous one.
	leaveEvent := NewMemberEvent(roomID, "@alice:example.com", "@alice:example.com", MembershipLeave, "example.com", []string{memberEvent.EventID}, []string{}, 3)
	err = state.AddEvent(leaveEvent)
	require.NoError(t, err)

	retrieved, err = state.GetStateEvent(EventTypeMember, "@alice:example.com")
	require.NoError(t, err)
	assert.Equal(t, leaveEvent.EventID, retrieved.EventID)
}

func TestFederatedRoomState_GetMembers_Copy(t *testing.T) {
	roomID := RoomID{Localpart: "testroom", ServerName: "example.com"}
	state := NewFederatedRoomState(roomID)

	memberEvent := NewMemberEvent(roomID, "@admin:example.com", "@alice:example.com", MembershipJoin, "example.com", []string{}, []string{}, 1)
	_ = state.AddEvent(memberEvent)

	members := state.GetMembers()
	members["@fake:example.com"] = MembershipJoin

	assert.False(t, state.HasMember("@fake:example.com"))
}

func TestFederatedRoomState_GetMemberPower(t *testing.T) {
	roomID := RoomID{Localpart: "testroom", ServerName: "example.com"}
	state := NewFederatedRoomState(roomID)

	// Default before any power levels event.
	assert.Equal(t, int64(0), state.GetMemberPower("@anyone:example.com"))

	pl := NewRoomPowerLevelsContent("@admin:example.com")
	pl.Users["@mod:example.com"] = 50
	pl.UsersDefault = 10

	createEvent := NewCreateEvent(roomID, "@admin:example.com", "example.com")
	_ = state.AddEvent(createEvent)

	plEvent := NewPowerLevelsEvent(roomID, "@admin:example.com", "example.com", pl, []string{createEvent.EventID}, []string{}, 2)
	_ = state.AddEvent(plEvent)

	assert.Equal(t, int64(100), state.GetMemberPower("@admin:example.com"))
	assert.Equal(t, int64(50), state.GetMemberPower("@mod:example.com"))
	assert.Equal(t, int64(10), state.GetMemberPower("@nobody:example.com"))
}

func TestFederatedRoomState_HasMember(t *testing.T) {
	roomID := RoomID{Localpart: "testroom", ServerName: "example.com"}
	state := NewFederatedRoomState(roomID)

	assert.False(t, state.HasMember("@alice:example.com"))

	memberEvent := NewMemberEvent(roomID, "@admin:example.com", "@alice:example.com", MembershipJoin, "example.com", []string{}, []string{}, 1)
	_ = state.AddEvent(memberEvent)

	assert.True(t, state.HasMember("@alice:example.com"))
	assert.False(t, state.HasMember("@bob:example.com"))
}

func TestFederatedRoomState_GetForwardExtremities(t *testing.T) {
	roomID := RoomID{Localpart: "testroom", ServerName: "example.com"}
	state := NewFederatedRoomState(roomID)

	assert.Empty(t, state.GetForwardExtremities())

	e1 := NewCreateEvent(roomID, "@admin:example.com", "example.com")
	_ = state.AddEvent(e1)

	extremities := state.GetForwardExtremities()
	assert.Len(t, extremities, 1)
}

func TestFederatedRoomState_GetCurrentDepth(t *testing.T) {
	roomID := RoomID{Localpart: "testroom", ServerName: "example.com"}
	state := NewFederatedRoomState(roomID)

	assert.Equal(t, int64(0), state.GetCurrentDepth())

	e1 := NewCreateEvent(roomID, "@admin:example.com", "example.com")
	_ = state.AddEvent(e1)
	assert.Equal(t, int64(1), state.GetCurrentDepth())
}

func TestFederatedRoomState_CanSendEvent(t *testing.T) {
	roomID := RoomID{Localpart: "testroom", ServerName: "example.com"}
	state := NewFederatedRoomState(roomID)

	pl := NewRoomPowerLevelsContent("@admin:example.com")
	pl.Users["@mod:example.com"] = 50
	pl.Events[EventTypeName] = 25

	createEvent := NewCreateEvent(roomID, "@admin:example.com", "example.com")
	_ = state.AddEvent(createEvent)

	plEvent := NewPowerLevelsEvent(roomID, "@admin:example.com", "example.com", pl, []string{createEvent.EventID}, []string{}, 2)
	_ = state.AddEvent(plEvent)

	// Admin (100) can send anything.
	assert.True(t, state.CanSendEvent("@admin:example.com", EventTypePowerLevels))
	assert.True(t, state.CanSendEvent("@admin:example.com", EventTypeMessage))
	assert.True(t, state.CanSendEvent("@admin:example.com", EventTypeName))

	// Mod (50) can send state events but not power levels (requires 100).
	assert.False(t, state.CanSendEvent("@mod:example.com", EventTypePowerLevels))
	assert.True(t, state.CanSendEvent("@mod:example.com", EventTypeName)) // 25 required
	assert.True(t, state.CanSendEvent("@mod:example.com", EventTypeJoinRules))

	// Regular user (0 default) can send messages but not state events.
	assert.False(t, state.CanSendEvent("@user:example.com", EventTypePowerLevels))
	assert.True(t, state.CanSendEvent("@user:example.com", EventTypeMessage))
	assert.False(t, state.CanSendEvent("@user:example.com", EventTypeName)) // 25 required
}

func TestFederatedRoomState_Concurrency(t *testing.T) {
	roomID := RoomID{Localpart: "testroom", ServerName: "example.com"}
	state := NewFederatedRoomState(roomID)

	// Add initial create event.
	createEvent := NewCreateEvent(roomID, "@admin:example.com", "example.com")
	_ = state.AddEvent(createEvent)

	var wg sync.WaitGroup
	numGoroutines := 50

	// Concurrent writes.
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			userID := "@user" + string(rune('0'+idx%10)) + ":example.com"
			event := NewMemberEvent(roomID, "@admin:example.com", userID, MembershipJoin, "example.com", []string{createEvent.EventID}, []string{}, int64(idx+2))
			_ = state.AddEvent(event)
		}(i)
	}

	// Concurrent reads.
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = state.GetMembers()
			_ = state.GetForwardExtremities()
			_ = state.GetCurrentDepth()
			_ = state.GetMemberPower("@admin:example.com")
			_ = state.HasMember("@user1:example.com")
			_ = state.CanSendEvent("@admin:example.com", EventTypeMessage)
		}()
	}

	wg.Wait()

	// Forward extremities should have at least the create event removed.
	extremities := state.GetForwardExtremities()
	assert.NotContains(t, extremities, createEvent.EventID)
	assert.GreaterOrEqual(t, state.GetCurrentDepth(), int64(2))
}

func TestInMemoryStateStore_GetOrCreateRoomState(t *testing.T) {
	store := NewInMemoryStateStore()

	roomID := RoomID{Localpart: "testroom", ServerName: "example.com"}

	state1 := store.GetOrCreateRoomState(roomID)
	require.NotNil(t, state1)
	assert.Equal(t, roomID, state1.RoomID)

	state2 := store.GetOrCreateRoomState(roomID)
	assert.Same(t, state1, state2)
}

func TestInMemoryStateStore_GetRoomState(t *testing.T) {
	store := NewInMemoryStateStore()

	roomID := RoomID{Localpart: "testroom", ServerName: "example.com"}

	// Not found before creation.
	_, err := store.GetRoomState(roomID)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrRoomNotFound)

	// Create via GetOrCreateRoomState.
	_ = store.GetOrCreateRoomState(roomID)

	state, err := store.GetRoomState(roomID)
	require.NoError(t, err)
	assert.Equal(t, roomID, state.RoomID)
}

func TestInMemoryStateStore_Concurrency(t *testing.T) {
	store := NewInMemoryStateStore()

	var wg sync.WaitGroup
	numGoroutines := 50

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			roomID := RoomID{Localpart: "room" + string(rune('0'+idx%5)), ServerName: "example.com"}
			state := store.GetOrCreateRoomState(roomID)
			_ = state.RoomID
		}(i)
	}

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			roomID := RoomID{Localpart: "room" + string(rune('0'+idx%5)), ServerName: "example.com"}
			_, _ = store.GetRoomState(roomID)
		}(i)
	}

	wg.Wait()

	// Verify exactly 5 distinct rooms were created.
	for i := 0; i < 5; i++ {
		roomID := RoomID{Localpart: "room" + string(rune('0'+i)), ServerName: "example.com"}
		state, err := store.GetRoomState(roomID)
		require.NoError(t, err)
		assert.Equal(t, roomID, state.RoomID)
	}
}

func TestFederatedRoomState_AddEvent_NonStateEventDoesNotAppearInGetStateEvent(t *testing.T) {
	roomID := RoomID{Localpart: "testroom", ServerName: "example.com"}
	state := NewFederatedRoomState(roomID)

	// Message events are not state events.
	msgEvent := NewMessageEvent(uuid.New(), roomID, "@alice:example.com", "hello", "example.com", []string{}, []string{}, 1)
	err := state.AddEvent(msgEvent)
	require.NoError(t, err)

	_, err = state.GetStateEvent(EventTypeMessage, "")
	assert.Error(t, err)
}

func TestFederatedRoomState_GetForwardExtremities_IndependentCopy(t *testing.T) {
	roomID := RoomID{Localpart: "testroom", ServerName: "example.com"}
	state := NewFederatedRoomState(roomID)

	e1 := NewCreateEvent(roomID, "@admin:example.com", "example.com")
	_ = state.AddEvent(e1)

	extremities := state.GetForwardExtremities()
	require.Len(t, extremities, 1)

	// Modifying the returned slice should not affect internal state.
	extremities[0] = "tampered"
	extremities2 := state.GetForwardExtremities()
	assert.Contains(t, extremities2, e1.EventID)
	assert.NotContains(t, extremities2, "tampered")
}

func TestFederatedRoomState_AddEvent_PowerLevelsParsing(t *testing.T) {
	roomID := RoomID{Localpart: "testroom", ServerName: "example.com"}
	state := NewFederatedRoomState(roomID)

	createEvent := NewCreateEvent(roomID, "@admin:example.com", "example.com")
	_ = state.AddEvent(createEvent)

	// Manually construct a power levels event with float64 values (as would come from JSON).
	plEvent := &Event{
		EventID:  "$pl:example.com",
		RoomID:   roomID.String(),
		Sender:   "@admin:example.com",
		Type:     EventTypePowerLevels,
		StateKey: stringPtr(""),
		Content: map[string]interface{}{
			"ban":            float64(60),
			"kick":           float64(40),
			"redact":         float64(30),
			"invite":         float64(0),
			"events_default": float64(5),
			"users_default":  float64(1),
			"users": map[string]interface{}{
				"@admin:example.com": float64(100),
				"@mod:example.com":   float64(50),
			},
			"events": map[string]interface{}{
				"m.room.name": float64(25),
			},
		},
		PrevEvents:     []string{createEvent.EventID},
		AuthEvents:     []string{},
		Depth:          2,
		Origin:         "example.com",
		OriginServerTS: 1234567890,
	}

	err := state.AddEvent(plEvent)
	require.NoError(t, err)

	assert.Equal(t, int64(60), state.PowerLevels.Ban)
	assert.Equal(t, int64(40), state.PowerLevels.Kick)
	assert.Equal(t, int64(30), state.PowerLevels.Redact)
	assert.Equal(t, int64(0), state.PowerLevels.Invite)
	assert.Equal(t, int64(5), state.PowerLevels.EventsDefault)
	assert.Equal(t, int64(1), state.PowerLevels.UsersDefault)
	assert.Equal(t, int64(100), state.GetMemberPower("@admin:example.com"))
	assert.Equal(t, int64(50), state.GetMemberPower("@mod:example.com"))
	assert.Equal(t, int64(1), state.GetMemberPower("@unknown:example.com"))
}

func TestFederatedRoomState_CanSendEvent_DefaultPowerLevels(t *testing.T) {
	roomID := RoomID{Localpart: "testroom", ServerName: "example.com"}
	state := NewFederatedRoomState(roomID)

	// Before any power levels event, everyone has default 0 power.
	// Only message events (events_default=0) should be allowed.
	assert.True(t, state.CanSendEvent("@anyone:example.com", EventTypeMessage))
	assert.False(t, state.CanSendEvent("@anyone:example.com", EventTypePowerLevels))
	assert.False(t, state.CanSendEvent("@anyone:example.com", EventTypeJoinRules))
	assert.False(t, state.CanSendEvent("@anyone:example.com", EventTypeName))
}

func TestStateEventKey(t *testing.T) {
	key1 := stateEventKey("m.room.member", "@alice:example.com")
	key2 := stateEventKey("m.room.member", "@bob:example.com")
	key3 := stateEventKey("m.room.power_levels", "")

	assert.NotEqual(t, key1, key2)
	assert.NotEqual(t, key1, key3)
	assert.True(t, strings.Contains(key1, "m.room.member"))
	assert.True(t, strings.Contains(key1, "@alice:example.com"))
}

func TestToInt64(t *testing.T) {
	assert.Equal(t, int64(42), toInt64(float64(42)))
	assert.Equal(t, int64(42), toInt64(int64(42)))
	assert.Equal(t, int64(42), toInt64(int(42)))
	assert.Equal(t, int64(0), toInt64("not a number"))
	assert.Equal(t, int64(0), toInt64(nil))
}
