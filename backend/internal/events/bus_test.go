package events

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewBus(t *testing.T) {
	bus := NewBus()
	assert.NotNil(t, bus)
	assert.NotNil(t, bus.handlers)
	assert.Equal(t, uint64(0), bus.nextID.Load())
}

func TestBus_Subscribe(t *testing.T) {
	bus := NewBus()
	handlerCalled := make(chan bool, 1)

	handler := func(event Event) {
		handlerCalled <- true
	}

	unsub := bus.Subscribe("test.event", handler)
	assert.NotNil(t, unsub)

	bus.Publish("test.event", "test data")

	select {
	case <-handlerCalled:
		// Success
	case <-time.After(time.Second):
		t.Error("Handler was not called")
	}
}

func TestBus_Unsubscribe(t *testing.T) {
	bus := NewBus()
	handlerCallCount := atomic.Int32{}

	handler := func(event Event) {
		handlerCallCount.Add(1)
	}

	unsub := bus.Subscribe("test.event", handler)

	// Publish once
	bus.Publish("test.event", "data1")
	time.Sleep(100 * time.Millisecond)

	// Unsubscribe
	unsub()
	time.Sleep(50 * time.Millisecond)

	// Publish again
	bus.Publish("test.event", "data2")
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, int32(1), handlerCallCount.Load())
}

func TestBus_Publish_MultipleHandlers(t *testing.T) {
	bus := NewBus()
	var wg sync.WaitGroup
	callCount := atomic.Int32{}

	handler1 := func(event Event) {
		callCount.Add(1)
		wg.Done()
	}
	handler2 := func(event Event) {
		callCount.Add(1)
		wg.Done()
	}

	bus.Subscribe("test.event", handler1)
	bus.Subscribe("test.event", handler2)

	wg.Add(2)
	bus.Publish("test.event", "data")

	wg.Wait()
	assert.Equal(t, int32(2), callCount.Load())
}

func TestBus_Publish_Wildcard(t *testing.T) {
	bus := NewBus()
	handlerCalled := make(chan string, 2)

	wildcardHandler := func(event Event) {
		handlerCalled <- event.Type
	}

	specificHandler := func(event Event) {
		handlerCalled <- "specific:" + event.Type
	}

	bus.SubscribeAll(wildcardHandler)
	bus.Subscribe("specific.event", specificHandler)

	bus.Publish("specific.event", "data")

	// Should receive both handlers
	var results []string
	timeout := time.After(time.Second)
	for i := 0; i < 2; i++ {
		select {
		case result := <-handlerCalled:
			results = append(results, result)
		case <-timeout:
			t.Fatal("Timeout waiting for handlers")
		}
	}

	assert.Contains(t, results, "specific.event")
	assert.Contains(t, results, "specific:specific.event")
}

func TestBus_Publish_NoHandlers(t *testing.T) {
	bus := NewBus()
	// Should not panic when publishing to event with no handlers
	assert.NotPanics(t, func() {
		bus.Publish("nonexistent.event", "data")
	})
}

func TestBus_Publish_PanicRecovery(t *testing.T) {
	bus := NewBus()
	handlerCalled := make(chan bool, 1)

	panicHandler := func(event Event) {
		panic("intentional panic")
	}
	normalHandler := func(event Event) {
		handlerCalled <- true
	}

	bus.Subscribe("test.event", panicHandler)
	bus.Subscribe("test.event", normalHandler)

	// Should not panic
	assert.NotPanics(t, func() {
		bus.Publish("test.event", "data")
	})

	// Other handler should still be called
	select {
	case <-handlerCalled:
		// Success
	case <-time.After(time.Second):
		t.Error("Normal handler was not called after panic")
	}
}

func TestBus_PublishSync(t *testing.T) {
	bus := NewBus()
	callOrder := make(chan int, 2)

	handler1 := func(event Event) {
		callOrder <- 1
	}
	handler2 := func(event Event) {
		callOrder <- 2
	}

	bus.Subscribe("test.event", handler1)
	bus.Subscribe("test.event", handler2)

	bus.PublishSync("test.event", "data")

	// Both should be called
	close(callOrder)
	var results []int
	for v := range callOrder {
		results = append(results, v)
	}
	assert.Len(t, results, 2)
}

func TestBus_HasHandlers(t *testing.T) {
	bus := NewBus()

	assert.False(t, bus.HasHandlers("test.event"))

	handler := func(event Event) {}
	bus.Subscribe("test.event", handler)

	assert.True(t, bus.HasHandlers("test.event"))
}

func TestBus_MultipleEventTypes(t *testing.T) {
	bus := NewBus()
	userCalled := make(chan bool, 1)
	serverCalled := make(chan bool, 1)

	bus.Subscribe(UserCreated, func(event Event) {
		userCalled <- true
	})
	bus.Subscribe(ServerCreated, func(event Event) {
		serverCalled <- true
	})

	bus.Publish(UserCreated, map[string]string{"username": "test"})
	bus.Publish(ServerCreated, map[string]string{"name": "test-server"})

	select {
	case <-userCalled:
		// Success
	case <-time.After(time.Second):
		t.Error("User handler was not called")
	}

	select {
	case <-serverCalled:
		// Success
	case <-time.After(time.Second):
		t.Error("Server handler was not called")
	}
}

func TestBus_ConcurrentSubscribeAndPublish(t *testing.T) {
	bus := NewBus()
	var wg sync.WaitGroup
	successCount := atomic.Int32{}

	// Concurrent subscriptions
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			handler := func(event Event) {
				successCount.Add(1)
			}
			bus.Subscribe("concurrent.event", handler)
		}()
	}

	wg.Wait()

	// Publish event
	bus.Publish("concurrent.event", "data")
	time.Sleep(200 * time.Millisecond)

	assert.Equal(t, int32(10), successCount.Load())
}

func TestBus_Unsubscribe_RemovesFromList(t *testing.T) {
	bus := NewBus()
	callCount := atomic.Int32{}

	handler := func(event Event) {
		callCount.Add(1)
	}

	unsub := bus.Subscribe("test.event", handler)

	// Verify handler is registered
	assert.True(t, bus.HasHandlers("test.event"))

	// Unsubscribe
	unsub()

	// Handler should be removed
	assert.False(t, bus.HasHandlers("test.event"))
}

func TestBus_CleanupEmptyHandlerList(t *testing.T) {
	bus := NewBus()

	handler := func(event Event) {}

	unsub := bus.Subscribe("test.event", handler)

	// Verify registration
	bus.mu.RLock()
	_, exists := bus.handlers["test.event"]
	bus.mu.RUnlock()
	assert.True(t, exists)

	// Unsubscribe
	unsub()

	// Should be cleaned up
	bus.mu.RLock()
	_, exists = bus.handlers["test.event"]
	bus.mu.RUnlock()
	assert.False(t, exists)
}

func TestEvent_Construction(t *testing.T) {
	event := Event{
		Type: "test.event",
		Data: map[string]string{"key": "value"},
	}

	assert.Equal(t, "test.event", event.Type)
	data, ok := event.Data.(map[string]string)
	assert.True(t, ok)
	assert.Equal(t, "value", data["key"])
}

func TestEventTypes_Constants(t *testing.T) {
	// Verify all event type constants are defined
	assert.Equal(t, "user.created", UserCreated)
	assert.Equal(t, "user.updated", UserUpdated)
	assert.Equal(t, "user.deleted", UserDeleted)
	assert.Equal(t, "presence.updated", PresenceUpdate)
	assert.Equal(t, "server.created", ServerCreated)
	assert.Equal(t, "server.updated", ServerUpdated)
	assert.Equal(t, "server.deleted", ServerDeleted)
	assert.Equal(t, "server.member_joined", MemberJoined)
	assert.Equal(t, "server.member_left", MemberLeft)
	assert.Equal(t, "server.member_kicked", MemberKicked)
	assert.Equal(t, "server.member_banned", MemberBanned)
	assert.Equal(t, "server.member_updated", MemberUpdated)
	assert.Equal(t, "channel.created", ChannelCreated)
	assert.Equal(t, "channel.updated", ChannelUpdated)
	assert.Equal(t, "channel.deleted", ChannelDeleted)
	assert.Equal(t, "message.created", MessageCreated)
	assert.Equal(t, "message.updated", MessageUpdated)
	assert.Equal(t, "message.deleted", MessageDeleted)
	assert.Equal(t, "message.pinned", MessagePinned)
	assert.Equal(t, "reaction.added", ReactionAdded)
	assert.Equal(t, "reaction.removed", ReactionRemoved)
	assert.Equal(t, "typing.started", TypingStarted)
	assert.Equal(t, "voice.joined", VoiceJoined)
	assert.Equal(t, "voice.left", VoiceLeft)
	assert.Equal(t, "voice.muted", VoiceMuted)
	assert.Equal(t, "voice.deafened", VoiceDeafened)
}
