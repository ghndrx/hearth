package matrixfederation

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInMemoryFederationEventStore(t *testing.T) {
	store := NewInMemoryFederationEventStore()
	require.NotNil(t, store)
	assert.NotNil(t, store.byEventID)
	assert.NotNil(t, store.byRoomID)
}

func TestInMemoryFederationEventStore_StoreEvent(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryFederationEventStore()

	originServer := "example.com"

	createEvent := NewCreateEvent(RoomID{Localpart: "testroom", ServerName: "example.com"}, "@creator:example.com", originServer)
	require.NotNil(t, createEvent)

	t.Run("store valid event", func(t *testing.T) {
		err := store.StoreEvent(ctx, createEvent)
		require.NoError(t, err)
	})

	t.Run("store nil event", func(t *testing.T) {
		err := store.StoreEvent(ctx, nil)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidEventFormat)
	})

	t.Run("store event without event_id", func(t *testing.T) {
		event := &Event{RoomID: "!testroom:example.com", Type: EventTypeMessage}
		err := store.StoreEvent(ctx, event)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrMissingRequiredField)
	})

	t.Run("store event without room_id", func(t *testing.T) {
		event := &Event{EventID: "$test:example.com", Type: EventTypeMessage}
		err := store.StoreEvent(ctx, event)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrMissingRequiredField)
	})

	t.Run("store duplicate event is no-op", func(t *testing.T) {
		err := store.StoreEvent(ctx, createEvent)
		require.NoError(t, err)
	})

	t.Run("stores multiple events in same room", func(t *testing.T) {
		msgEvent := NewMessageEvent(uuid.New(), RoomID{Localpart: "testroom", ServerName: "example.com"}, "@user:example.com", "hello", originServer, []string{createEvent.EventID}, []string{createEvent.EventID}, 2)
		err := store.StoreEvent(ctx, msgEvent)
		require.NoError(t, err)
	})
}

func TestInMemoryFederationEventStore_GetEvent(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryFederationEventStore()

	originServer := "example.com"
	createEvent := NewCreateEvent(RoomID{Localpart: "testroom", ServerName: "example.com"}, "@creator:example.com", originServer)
	require.NoError(t, store.StoreEvent(ctx, createEvent))

	t.Run("get existing event", func(t *testing.T) {
		event, err := store.GetEvent(ctx, createEvent.EventID)
		require.NoError(t, err)
		assert.Equal(t, createEvent.EventID, event.EventID)
		assert.Equal(t, createEvent.RoomID, event.RoomID)
	})

	t.Run("get non-existent event", func(t *testing.T) {
		event, err := store.GetEvent(ctx, "$nonexistent:example.com")
		require.Error(t, err)
		assert.Nil(t, event)
		assert.Contains(t, err.Error(), "event not found")
	})

	t.Run("get event with empty event_id", func(t *testing.T) {
		event, err := store.GetEvent(ctx, "")
		require.Error(t, err)
		assert.Nil(t, event)
		assert.ErrorIs(t, err, ErrMissingRequiredField)
	})
}

func TestInMemoryFederationEventStore_GetEventsByRoom(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryFederationEventStore()

	roomID := RoomID{Localpart: "testroom", ServerName: "example.com"}
	originServer := "example.com"

	createEvent := NewCreateEvent(roomID, "@creator:example.com", originServer)
	require.NoError(t, store.StoreEvent(ctx, createEvent))

	msg1 := NewMessageEvent(uuid.New(), roomID, "@user:example.com", "hello 1", originServer, []string{createEvent.EventID}, []string{createEvent.EventID}, 2)
	msg2 := NewMessageEvent(uuid.New(), roomID, "@user:example.com", "hello 2", originServer, []string{msg1.EventID}, []string{createEvent.EventID}, 3)
	msg3 := NewMessageEvent(uuid.New(), roomID, "@user:example.com", "hello 3", originServer, []string{msg2.EventID}, []string{createEvent.EventID}, 4)

	require.NoError(t, store.StoreEvent(ctx, msg1))
	require.NoError(t, store.StoreEvent(ctx, msg2))
	require.NoError(t, store.StoreEvent(ctx, msg3))

	t.Run("get events with limit", func(t *testing.T) {
		events, err := store.GetEventsByRoom(ctx, roomID.String(), 2)
		require.NoError(t, err)
		require.Len(t, events, 2)
		assert.Equal(t, msg2.EventID, events[0].EventID)
		assert.Equal(t, msg3.EventID, events[1].EventID)
	})

	t.Run("get events with limit larger than count", func(t *testing.T) {
		events, err := store.GetEventsByRoom(ctx, roomID.String(), 10)
		require.NoError(t, err)
		require.Len(t, events, 4)
	})

	t.Run("get events with limit zero", func(t *testing.T) {
		events, err := store.GetEventsByRoom(ctx, roomID.String(), 0)
		require.NoError(t, err)
		require.Len(t, events, 4)
	})

	t.Run("get events for non-existent room", func(t *testing.T) {
		events, err := store.GetEventsByRoom(ctx, "!nonexistent:example.com", 10)
		require.NoError(t, err)
		assert.Empty(t, events)
	})

	t.Run("get events with empty room_id", func(t *testing.T) {
		events, err := store.GetEventsByRoom(ctx, "", 10)
		require.Error(t, err)
		assert.Nil(t, events)
		assert.ErrorIs(t, err, ErrMissingRequiredField)
	})

	t.Run("get events with negative limit", func(t *testing.T) {
		events, err := store.GetEventsByRoom(ctx, roomID.String(), -1)
		require.Error(t, err)
		assert.Nil(t, events)
	})
}

func TestInMemoryFederationEventStore_GetEventsAfter(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryFederationEventStore()

	roomID := RoomID{Localpart: "testroom", ServerName: "example.com"}
	originServer := "example.com"

	createEvent := NewCreateEvent(roomID, "@creator:example.com", originServer)
	require.NoError(t, store.StoreEvent(ctx, createEvent))

	msg1 := NewMessageEvent(uuid.New(), roomID, "@user:example.com", "hello 1", originServer, []string{createEvent.EventID}, []string{createEvent.EventID}, 2)
	msg2 := NewMessageEvent(uuid.New(), roomID, "@user:example.com", "hello 2", originServer, []string{msg1.EventID}, []string{createEvent.EventID}, 3)
	msg3 := NewMessageEvent(uuid.New(), roomID, "@user:example.com", "hello 3", originServer, []string{msg2.EventID}, []string{createEvent.EventID}, 4)
	msg4 := NewMessageEvent(uuid.New(), roomID, "@user:example.com", "hello 4", originServer, []string{msg3.EventID}, []string{createEvent.EventID}, 5)

	require.NoError(t, store.StoreEvent(ctx, msg1))
	require.NoError(t, store.StoreEvent(ctx, msg2))
	require.NoError(t, store.StoreEvent(ctx, msg3))
	require.NoError(t, store.StoreEvent(ctx, msg4))

	t.Run("get events after with limit", func(t *testing.T) {
		events, err := store.GetEventsAfter(ctx, roomID.String(), msg2.EventID, 2)
		require.NoError(t, err)
		require.Len(t, events, 2)
		assert.Equal(t, msg3.EventID, events[0].EventID)
		assert.Equal(t, msg4.EventID, events[1].EventID)
	})

	t.Run("get events after with limit larger than remaining", func(t *testing.T) {
		events, err := store.GetEventsAfter(ctx, roomID.String(), msg2.EventID, 10)
		require.NoError(t, err)
		require.Len(t, events, 2)
	})

	t.Run("get events after with limit zero", func(t *testing.T) {
		events, err := store.GetEventsAfter(ctx, roomID.String(), msg2.EventID, 0)
		require.NoError(t, err)
		require.Len(t, events, 2)
	})

	t.Run("get events after last event", func(t *testing.T) {
		events, err := store.GetEventsAfter(ctx, roomID.String(), msg4.EventID, 10)
		require.NoError(t, err)
		assert.Empty(t, events)
	})

	t.Run("get events after non-existent event", func(t *testing.T) {
		events, err := store.GetEventsAfter(ctx, roomID.String(), "$nonexistent:example.com", 10)
		require.Error(t, err)
		assert.Nil(t, events)
		assert.Contains(t, err.Error(), "after event not found")
	})

	t.Run("get events after in non-existent room", func(t *testing.T) {
		events, err := store.GetEventsAfter(ctx, "!nonexistent:example.com", "$something:example.com", 10)
		require.NoError(t, err)
		assert.Empty(t, events)
	})

	t.Run("get events after with empty room_id", func(t *testing.T) {
		events, err := store.GetEventsAfter(ctx, "", msg1.EventID, 10)
		require.Error(t, err)
		assert.Nil(t, events)
		assert.ErrorIs(t, err, ErrMissingRequiredField)
	})

	t.Run("get events after with empty after_event_id", func(t *testing.T) {
		events, err := store.GetEventsAfter(ctx, roomID.String(), "", 10)
		require.Error(t, err)
		assert.Nil(t, events)
	})

	t.Run("get events after with negative limit", func(t *testing.T) {
		events, err := store.GetEventsAfter(ctx, roomID.String(), msg1.EventID, -1)
		require.Error(t, err)
		assert.Nil(t, events)
	})
}

func TestInMemoryFederationEventStore_HasEvent(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryFederationEventStore()

	roomID := RoomID{Localpart: "testroom", ServerName: "example.com"}
	originServer := "example.com"
	createEvent := NewCreateEvent(roomID, "@creator:example.com", originServer)
	require.NoError(t, store.StoreEvent(ctx, createEvent))

	t.Run("has existing event", func(t *testing.T) {
		found, err := store.HasEvent(ctx, createEvent.EventID)
		require.NoError(t, err)
		assert.True(t, found)
	})

	t.Run("has non-existent event", func(t *testing.T) {
		found, err := store.HasEvent(ctx, "$nonexistent:example.com")
		require.NoError(t, err)
		assert.False(t, found)
	})

	t.Run("has event with empty event_id", func(t *testing.T) {
		found, err := store.HasEvent(ctx, "")
		require.Error(t, err)
		assert.False(t, found)
		assert.ErrorIs(t, err, ErrMissingRequiredField)
	})
}

func TestInMemoryFederationEventStore_GetAuthEvents(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryFederationEventStore()

	roomID := RoomID{Localpart: "testroom", ServerName: "example.com"}
	originServer := "example.com"
	creator := "@creator:example.com"
	sender := "@user:example.com"

	createEvent := NewCreateEvent(roomID, creator, originServer)
	require.NoError(t, store.StoreEvent(ctx, createEvent))

	powerLevelsContent := NewRoomPowerLevelsContent(creator)
	powerLevels := NewPowerLevelsEvent(roomID, creator, originServer, powerLevelsContent, []string{createEvent.EventID}, []string{createEvent.EventID}, 2)
	require.NoError(t, store.StoreEvent(ctx, powerLevels))

	joinRules := &Event{
		EventID:    "$joinrules:example.com",
		RoomID:     roomID.String(),
		Sender:     creator,
		Type:       EventTypeJoinRules,
		StateKey:   stringPtr(""),
		Content:    map[string]interface{}{"join_rule": "public"},
		PrevEvents: []string{powerLevels.EventID},
		AuthEvents: []string{createEvent.EventID, powerLevels.EventID},
		Depth:      3,
		Origin:     originServer,
	}
	require.NoError(t, store.StoreEvent(ctx, joinRules))

	memberEvent := NewMemberEvent(roomID, creator, sender, MembershipJoin, originServer, []string{joinRules.EventID}, []string{createEvent.EventID, powerLevels.EventID, joinRules.EventID}, 4)
	require.NoError(t, store.StoreEvent(ctx, memberEvent))

	msgEvent := NewMessageEvent(uuid.New(), roomID, sender, "hello", originServer, []string{memberEvent.EventID}, []string{createEvent.EventID, powerLevels.EventID, memberEvent.EventID, joinRules.EventID}, 5)
	require.NoError(t, store.StoreEvent(ctx, msgEvent))

	t.Run("get auth events for message", func(t *testing.T) {
		authEvents, err := store.GetAuthEvents(ctx, msgEvent)
		require.NoError(t, err)
		require.Len(t, authEvents, 4)

		// Verify all expected auth events are present.
		typeSet := make(map[string]bool)
		for _, ae := range authEvents {
			typeSet[ae.Type] = true
		}
		assert.True(t, typeSet[EventTypeCreate])
		assert.True(t, typeSet[EventTypePowerLevels])
		assert.True(t, typeSet[EventTypeMember])
		assert.True(t, typeSet[EventTypeJoinRules])
	})

	t.Run("get auth events for create event", func(t *testing.T) {
		// Create event has no sender member event.
		authEvents, err := store.GetAuthEvents(ctx, createEvent)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(authEvents), 1)
		assert.Equal(t, EventTypeCreate, authEvents[0].Type)
	})

	t.Run("get auth events for nil event", func(t *testing.T) {
		authEvents, err := store.GetAuthEvents(ctx, nil)
		require.Error(t, err)
		assert.Nil(t, authEvents)
	})

	t.Run("get auth events for event without room_id", func(t *testing.T) {
		event := &Event{EventID: "$test:example.com", Type: EventTypeMessage}
		authEvents, err := store.GetAuthEvents(ctx, event)
		require.Error(t, err)
		assert.Nil(t, authEvents)
		assert.ErrorIs(t, err, ErrMissingRequiredField)
	})

	t.Run("get auth events for non-existent room", func(t *testing.T) {
		event := &Event{EventID: "$test:example.com", RoomID: "!nonexistent:example.com", Type: EventTypeMessage}
		authEvents, err := store.GetAuthEvents(ctx, event)
		require.Error(t, err)
		assert.Nil(t, authEvents)
		assert.Contains(t, err.Error(), "no events found for room")
	})

	t.Run("get auth events with updated power levels", func(t *testing.T) {
		newPowerLevelsContent := NewRoomPowerLevelsContent(creator)
		newPowerLevelsContent.Users["@admin:example.com"] = 200
		newPowerLevels := NewPowerLevelsEvent(roomID, creator, originServer, newPowerLevelsContent, []string{powerLevels.EventID}, []string{createEvent.EventID, powerLevels.EventID}, 6)
		require.NoError(t, store.StoreEvent(ctx, newPowerLevels))

		msg2 := NewMessageEvent(uuid.New(), roomID, sender, "hello again", originServer, []string{msgEvent.EventID}, []string{createEvent.EventID, newPowerLevels.EventID, memberEvent.EventID, joinRules.EventID}, 7)
		require.NoError(t, store.StoreEvent(ctx, msg2))

		authEvents, err := store.GetAuthEvents(ctx, msg2)
		require.NoError(t, err)
		require.Len(t, authEvents, 4)

		// Verify the latest power levels event is returned.
		var foundPowerLevels *Event
		for _, ae := range authEvents {
			if ae.Type == EventTypePowerLevels {
				foundPowerLevels = ae
				break
			}
		}
		require.NotNil(t, foundPowerLevels)
		assert.Equal(t, newPowerLevels.EventID, foundPowerLevels.EventID)
		assert.Equal(t, int64(6), foundPowerLevels.Depth)
	})
}

func TestInMemoryFederationEventStore_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryFederationEventStore()

	roomID := RoomID{Localpart: "testroom", ServerName: "example.com"}
	originServer := "example.com"
	createEvent := NewCreateEvent(roomID, "@creator:example.com", originServer)
	require.NoError(t, store.StoreEvent(ctx, createEvent))

	// Concurrent stores.
	t.Run("concurrent stores", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			t.Run(fmt.Sprintf("store_%d", i), func(t *testing.T) {
				t.Parallel()
				msg := NewMessageEvent(uuid.New(), roomID, "@user:example.com", fmt.Sprintf("msg %d", i), originServer, []string{createEvent.EventID}, []string{createEvent.EventID}, int64(i+2))
				err := store.StoreEvent(ctx, msg)
				require.NoError(t, err)
			})
		}
	})

	// Concurrent reads and writes.
	t.Run("concurrent reads and writes", func(t *testing.T) {
		for i := 0; i < 50; i++ {
			t.Run(fmt.Sprintf("writer_%d", i), func(t *testing.T) {
				t.Parallel()
				msg := NewMessageEvent(uuid.New(), roomID, "@user:example.com", fmt.Sprintf("msg %d", i), originServer, []string{createEvent.EventID}, []string{createEvent.EventID}, int64(i+102))
				err := store.StoreEvent(ctx, msg)
				require.NoError(t, err)
			})
		}
		for i := 0; i < 50; i++ {
			t.Run(fmt.Sprintf("reader_%d", i), func(t *testing.T) {
				t.Parallel()
				_, err := store.GetEventsByRoom(ctx, roomID.String(), 10)
				require.NoError(t, err)
			})
		}
	})
}

func TestInMemoryFederationEventStore_MultipleRooms(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryFederationEventStore()

	room1 := RoomID{Localpart: "room1", ServerName: "example.com"}
	room2 := RoomID{Localpart: "room2", ServerName: "example.com"}
	originServer := "example.com"

	create1 := NewCreateEvent(room1, "@creator:example.com", originServer)
	create2 := NewCreateEvent(room2, "@creator:example.com", originServer)

	require.NoError(t, store.StoreEvent(ctx, create1))
	require.NoError(t, store.StoreEvent(ctx, create2))

	msg1 := NewMessageEvent(uuid.New(), room1, "@user:example.com", "room1 msg", originServer, []string{create1.EventID}, []string{create1.EventID}, 2)
	msg2 := NewMessageEvent(uuid.New(), room2, "@user:example.com", "room2 msg", originServer, []string{create2.EventID}, []string{create2.EventID}, 2)

	require.NoError(t, store.StoreEvent(ctx, msg1))
	require.NoError(t, store.StoreEvent(ctx, msg2))

	t.Run("events are isolated by room", func(t *testing.T) {
		events1, err := store.GetEventsByRoom(ctx, room1.String(), 10)
		require.NoError(t, err)
		require.Len(t, events1, 2)

		events2, err := store.GetEventsByRoom(ctx, room2.String(), 10)
		require.NoError(t, err)
		require.Len(t, events2, 2)

		// Verify events belong to correct rooms.
		for _, e := range events1 {
			assert.Equal(t, room1.String(), e.RoomID)
		}
		for _, e := range events2 {
			assert.Equal(t, room2.String(), e.RoomID)
		}
	})

	t.Run("get auth events per room", func(t *testing.T) {
		auth1, err := store.GetAuthEvents(ctx, msg1)
		require.NoError(t, err)
		require.Len(t, auth1, 1)
		assert.Equal(t, EventTypeCreate, auth1[0].Type)

		auth2, err := store.GetAuthEvents(ctx, msg2)
		require.NoError(t, err)
		require.Len(t, auth2, 1)
		assert.Equal(t, EventTypeCreate, auth2[0].Type)
	})
}

func TestInMemoryFederationEventStore_GetAuthEventsDeterministicOrder(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryFederationEventStore()

	roomID := RoomID{Localpart: "testroom", ServerName: "example.com"}
	originServer := "example.com"
	creator := "@creator:example.com"
	sender := "@user:example.com"

	createEvent := NewCreateEvent(roomID, creator, originServer)
	require.NoError(t, store.StoreEvent(ctx, createEvent))

	powerLevelsContent := NewRoomPowerLevelsContent(creator)
	powerLevels := NewPowerLevelsEvent(roomID, creator, originServer, powerLevelsContent, []string{createEvent.EventID}, []string{createEvent.EventID}, 2)
	require.NoError(t, store.StoreEvent(ctx, powerLevels))

	joinRules := &Event{
		EventID:    "$joinrules:example.com",
		RoomID:     roomID.String(),
		Sender:     creator,
		Type:       EventTypeJoinRules,
		StateKey:   stringPtr(""),
		Content:    map[string]interface{}{"join_rule": "public"},
		PrevEvents: []string{powerLevels.EventID},
		AuthEvents: []string{createEvent.EventID, powerLevels.EventID},
		Depth:      3,
		Origin:     originServer,
	}
	require.NoError(t, store.StoreEvent(ctx, joinRules))

	memberEvent := NewMemberEvent(roomID, creator, sender, MembershipJoin, originServer, []string{joinRules.EventID}, []string{createEvent.EventID, powerLevels.EventID, joinRules.EventID}, 4)
	require.NoError(t, store.StoreEvent(ctx, memberEvent))

	msgEvent := NewMessageEvent(uuid.New(), roomID, sender, "hello", originServer, []string{memberEvent.EventID}, []string{createEvent.EventID, powerLevels.EventID, memberEvent.EventID, joinRules.EventID}, 5)
	require.NoError(t, store.StoreEvent(ctx, msgEvent))

	// Call multiple times and verify deterministic ordering.
	var firstOrder []string
	for i := 0; i < 5; i++ {
		authEvents, err := store.GetAuthEvents(ctx, msgEvent)
		require.NoError(t, err)
		require.Len(t, authEvents, 4)

		order := make([]string, 4)
		for j, ae := range authEvents {
			order[j] = ae.Type
		}

		if i == 0 {
			firstOrder = order
		} else {
			assert.Equal(t, firstOrder, order, "auth events order should be deterministic")
		}
	}
}

func TestFederationEventStore_InterfaceCompliance(t *testing.T) {
	// Verify that InMemoryFederationEventStore implements FederationEventStore.
	var _ FederationEventStore = (*InMemoryFederationEventStore)(nil)
}

func TestFederationStore(t *testing.T) {
	// Top-level runner so -run TestFederationStore catches all store tests.
	// Actual tests are in the named functions above; this ensures the parent
	// agent's requested filter works even though sub-tests use different prefixes.
	t.Run("StoreEvent", TestInMemoryFederationEventStore_StoreEvent)
	t.Run("GetEvent", TestInMemoryFederationEventStore_GetEvent)
	t.Run("GetEventsByRoom", TestInMemoryFederationEventStore_GetEventsByRoom)
	t.Run("GetEventsAfter", TestInMemoryFederationEventStore_GetEventsAfter)
	t.Run("HasEvent", TestInMemoryFederationEventStore_HasEvent)
	t.Run("GetAuthEvents", TestInMemoryFederationEventStore_GetAuthEvents)
	t.Run("ConcurrentAccess", TestInMemoryFederationEventStore_ConcurrentAccess)
	t.Run("MultipleRooms", TestInMemoryFederationEventStore_MultipleRooms)
	t.Run("AuthEventsDeterministicOrder", TestInMemoryFederationEventStore_GetAuthEventsDeterministicOrder)
	t.Run("InterfaceCompliance", TestFederationEventStore_InterfaceCompliance)
}
