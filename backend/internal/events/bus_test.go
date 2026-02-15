package events

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewBus(t *testing.T) {
	bus := NewBus()
	if bus == nil {
		t.Fatal("NewBus returned nil")
	}
	if bus.handlers == nil {
		t.Fatal("handlers map not initialized")
	}
}

func TestSubscribe(t *testing.T) {
	bus := NewBus()

	handler := func(e Event) {}

	bus.Subscribe("test.event", handler)

	if !bus.HasHandlers("test.event") {
		t.Error("expected HasHandlers to return true after subscription")
	}
}

func TestPublishSync(t *testing.T) {
	bus := NewBus()

	var received Event
	var wg sync.WaitGroup
	wg.Add(1)

	bus.Subscribe("test.event", func(e Event) {
		received = e
		wg.Done()
	})

	bus.PublishSync("test.event", "test-data")

	if received.Type != "test.event" {
		t.Errorf("expected event type 'test.event', got '%s'", received.Type)
	}
	if received.Data != "test-data" {
		t.Errorf("expected data 'test-data', got '%v'", received.Data)
	}
}

func TestPublish(t *testing.T) {
	bus := NewBus()

	var received atomic.Value
	done := make(chan struct{})

	bus.Subscribe("async.event", func(e Event) {
		received.Store(e)
		close(done)
	})

	bus.Publish("async.event", "async-data")

	select {
	case <-done:
		e := received.Load().(Event)
		if e.Type != "async.event" {
			t.Errorf("expected event type 'async.event', got '%s'", e.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for async handler")
	}
}

func TestUnsubscribe(t *testing.T) {
	bus := NewBus()

	handler := func(e Event) {}
	bus.Subscribe("test.event", handler)

	if !bus.HasHandlers("test.event") {
		t.Fatal("expected handler to be registered")
	}

	bus.Unsubscribe("test.event", handler)

	if bus.HasHandlers("test.event") {
		t.Error("expected no handlers after unsubscribe")
	}
}

func TestHasHandlers(t *testing.T) {
	bus := NewBus()

	if bus.HasHandlers("nonexistent") {
		t.Error("expected no handlers for nonexistent event")
	}

	bus.Subscribe("exists", func(e Event) {})

	if !bus.HasHandlers("exists") {
		t.Error("expected handler to exist")
	}
}

func TestHasHandlersWildcard(t *testing.T) {
	bus := NewBus()

	bus.SubscribeAll(func(e Event) {})

	// Wildcard handlers should make HasHandlers return true for any event
	if !bus.HasHandlers("any.event") {
		t.Error("expected wildcard handler to match any event")
	}
}

func TestMultipleHandlers(t *testing.T) {
	bus := NewBus()

	var count int32

	for i := 0; i < 3; i++ {
		bus.Subscribe("multi.event", func(e Event) {
			atomic.AddInt32(&count, 1)
		})
	}

	bus.PublishSync("multi.event", nil)

	if atomic.LoadInt32(&count) != 3 {
		t.Errorf("expected 3 handlers to be called, got %d", count)
	}
}

func TestHandlerPanicRecovery(t *testing.T) {
	bus := NewBus()

	var normalHandlerCalled atomic.Bool

	// Add a handler that panics
	bus.Subscribe("panic.event", func(e Event) {
		panic("test panic")
	})

	// Add a normal handler
	bus.Subscribe("panic.event", func(e Event) {
		normalHandlerCalled.Store(true)
	})

	// Should not panic
	bus.PublishSync("panic.event", nil)

	// Give time for async handlers
	time.Sleep(100 * time.Millisecond)

	if !normalHandlerCalled.Load() {
		t.Error("expected normal handler to be called despite panic in other handler")
	}
}

func TestEventTypes(t *testing.T) {
	// Test that event type constants are defined
	eventTypes := []string{
		UserCreated, UserUpdated, UserDeleted, PresenceUpdate,
		ServerCreated, ServerUpdated, ServerDeleted,
		MemberJoined, MemberLeft, MemberKicked, MemberBanned, MemberUpdated,
		ChannelCreated, ChannelUpdated, ChannelDeleted,
		MessageCreated, MessageUpdated, MessageDeleted, MessagePinned,
		ReactionAdded, ReactionRemoved,
		TypingStarted,
		VoiceJoined, VoiceLeft, VoiceMuted, VoiceDeafened,
	}

	for _, et := range eventTypes {
		if et == "" {
			t.Error("event type constant is empty")
		}
	}
}

func TestPublishNoHandlers(t *testing.T) {
	bus := NewBus()

	// Should not panic when publishing to event with no handlers
	bus.Publish("no.handlers", "data")
	bus.PublishSync("no.handlers", "data")
}

func TestConcurrentPublish(t *testing.T) {
	bus := NewBus()

	var count int32

	bus.Subscribe("concurrent.event", func(e Event) {
		atomic.AddInt32(&count, 1)
	})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.Publish("concurrent.event", nil)
		}()
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond) // Wait for async handlers

	if atomic.LoadInt32(&count) != 100 {
		t.Errorf("expected 100 events, got %d", count)
	}
}
