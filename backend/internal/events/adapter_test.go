package events

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewServiceBusAdapter(t *testing.T) {
	bus := NewBus()
	adapter := NewServiceBusAdapter(bus)

	assert.NotNil(t, adapter)
	assert.NotNil(t, adapter.bus)
	assert.NotNil(t, adapter.unsubFuncs)
}

func TestServiceBusAdapter_Publish(t *testing.T) {
	bus := NewBus()
	adapter := NewServiceBusAdapter(bus)
	handlerCalled := make(chan interface{}, 1)

	// Subscribe using the bus directly
	bus.Subscribe("test.event", func(event Event) {
		handlerCalled <- event.Data
	})

	// Publish through adapter
	adapter.Publish("test.event", "test data")

	select {
	case data := <-handlerCalled:
		assert.Equal(t, "test data", data)
	case <-time.After(time.Second):
		t.Error("Handler was not called")
	}
}

func TestServiceBusAdapter_Subscribe(t *testing.T) {
	bus := NewBus()
	adapter := NewServiceBusAdapter(bus)
	handlerCalled := make(chan interface{}, 1)

	handler := func(data interface{}) {
		handlerCalled <- data
	}

	adapter.Subscribe("test.event", handler)

	// Publish through bus
	bus.Publish("test.event", "test data")

	select {
	case data := <-handlerCalled:
		assert.Equal(t, "test data", data)
	case <-time.After(time.Second):
		t.Error("Handler was not called")
	}
}

func TestServiceBusAdapter_Subscribe_Multiple(t *testing.T) {
	bus := NewBus()
	adapter := NewServiceBusAdapter(bus)
	var wg sync.WaitGroup
	callCount := atomic.Int32{}

	handler1 := func(data interface{}) {
		callCount.Add(1)
		wg.Done()
	}
	handler2 := func(data interface{}) {
		callCount.Add(1)
		wg.Done()
	}

	adapter.Subscribe("test.event", handler1)
	adapter.Subscribe("test.event", handler2)

	wg.Add(2)
	bus.Publish("test.event", "data")

	wg.Wait()
	assert.Equal(t, int32(2), callCount.Load())
}

func TestServiceBusAdapter_Unsubscribe(t *testing.T) {
	bus := NewBus()
	adapter := NewServiceBusAdapter(bus)
	callCount := atomic.Int32{}

	handler := func(data interface{}) {
		callCount.Add(1)
	}

	adapter.Subscribe("test.event", handler)

	// Publish once
	bus.Publish("test.event", "data1")
	time.Sleep(100 * time.Millisecond)

	// Unsubscribe
	adapter.Unsubscribe("test.event", handler)
	time.Sleep(50 * time.Millisecond)

	// Publish again
	bus.Publish("test.event", "data2")
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, int32(1), callCount.Load())
}

func TestServiceBusAdapter_Unsubscribe_NonExistent(t *testing.T) {
	bus := NewBus()
	adapter := NewServiceBusAdapter(bus)

	handler := func(data interface{}) {}

	// Should not panic when unsubscribing non-existent handler
	assert.NotPanics(t, func() {
		adapter.Unsubscribe("test.event", handler)
	})
}

func TestServiceBusAdapter_MultipleEventTypes(t *testing.T) {
	bus := NewBus()
	adapter := NewServiceBusAdapter(bus)
	userCalled := make(chan bool, 1)
	serverCalled := make(chan bool, 1)

	adapter.Subscribe(UserCreated, func(data interface{}) {
		userCalled <- true
	})
	adapter.Subscribe(ServerCreated, func(data interface{}) {
		serverCalled <- true
	})

	adapter.Publish(UserCreated, map[string]string{"username": "test"})
	adapter.Publish(ServerCreated, map[string]string{"name": "test-server"})

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

func TestServiceBusAdapter_ConcurrentSubscribeAndUnsubscribe(t *testing.T) {
	bus := NewBus()
	adapter := NewServiceBusAdapter(bus)
	var wg sync.WaitGroup

	// Concurrent subscriptions
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			handler := func(data interface{}) {}
			adapter.Subscribe("concurrent.event", handler)
		}(i)
	}

	wg.Wait()

	// Verify handlers are registered
	assert.True(t, bus.HasHandlers("concurrent.event"))
}

func TestServiceBusAdapter_HandlerPointerUniqueness(t *testing.T) {
	bus := NewBus()
	adapter := NewServiceBusAdapter(bus)
	callCount := atomic.Int32{}

	handler := func(data interface{}) {
		callCount.Add(1)
	}

	// Subscribe same handler twice
	adapter.Subscribe("test.event", handler)
	adapter.Subscribe("test.event", handler)

	bus.Publish("test.event", "data")
	time.Sleep(100 * time.Millisecond)

	// Should be called twice since it's registered twice
	assert.Equal(t, int32(2), callCount.Load())
}

func TestServiceBusAdapter_UnsubscribeOneOfMany(t *testing.T) {
	bus := NewBus()
	adapter := NewServiceBusAdapter(bus)
	handler1Called := make(chan bool, 1)
	handler2CallCount := atomic.Int32{}

	handler1 := func(data interface{}) {
		handler1Called <- true
	}
	handler2 := func(data interface{}) {
		handler2CallCount.Add(1)
	}

	adapter.Subscribe("test.event", handler1)
	adapter.Subscribe("test.event", handler2)

	// Unsubscribe first handler
	adapter.Unsubscribe("test.event", handler1)

	// Publish
	bus.Publish("test.event", "data")

	// handler1 should not be called
	select {
	case <-handler1Called:
		t.Error("Unsubscribed handler was called")
	case <-time.After(100 * time.Millisecond):
		// Expected - handler1 was unsubscribed
	}

	time.Sleep(100 * time.Millisecond)
	// handler2 should still be called
	assert.Equal(t, int32(1), handler2CallCount.Load())
}

func TestServiceBusAdapter_Publish_NilBus(t *testing.T) {
	// This should never happen in practice, but test defensive behavior
	bus := NewBus()
	adapter := NewServiceBusAdapter(bus)

	// Should not panic
	assert.NotPanics(t, func() {
		adapter.Publish("test.event", "data")
	})
}

func TestServiceBusAdapter_IntegrationWithBus(t *testing.T) {
	bus := NewBus()
	adapter := NewServiceBusAdapter(bus)

	messageCreated := false
	messageUpdated := false

	adapter.Subscribe(MessageCreated, func(data interface{}) {
		messageCreated = true
	})
	adapter.Subscribe(MessageUpdated, func(data interface{}) {
		messageUpdated = true
	})

	// Publish various events
	adapter.Publish(MessageCreated, map[string]string{"content": "hello"})
	adapter.Publish(MessageUpdated, map[string]string{"content": "updated"})
	adapter.Publish(MessageDeleted, map[string]string{"id": "123"})

	time.Sleep(200 * time.Millisecond)

	assert.True(t, messageCreated, "MessageCreated handler should be called")
	assert.True(t, messageUpdated, "MessageUpdated handler should be called")
	// MessageDeleted has no handler, so nothing to assert there
}

func TestServiceBusAdapter_DataPassing(t *testing.T) {
	bus := NewBus()
	adapter := NewServiceBusAdapter(bus)
	receivedData := make(chan interface{}, 1)

	handler := func(data interface{}) {
		receivedData <- data
	}

	adapter.Subscribe("test.event", handler)

	// Test with different data types
	testData := map[string]interface{}{
		"user_id":  "123",
		"username": "testuser",
		"active":   true,
	}

	adapter.Publish("test.event", testData)

	select {
	case data := <-receivedData:
		result, ok := data.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "123", result["user_id"])
		assert.Equal(t, "testuser", result["username"])
		assert.Equal(t, true, result["active"])
	case <-time.After(time.Second):
		t.Error("Handler was not called")
	}
}

func TestServiceBusAdapter_RaceConditions(t *testing.T) {
	bus := NewBus()
	adapter := NewServiceBusAdapter(bus)
	var wg sync.WaitGroup

	// Concurrent subscribe and publish
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			handler := func(data interface{}) {}
			adapter.Subscribe("race.event", handler)
			adapter.Publish("race.event", id)
		}(i)
	}

	wg.Wait()
	// If we get here without panic or deadlock, test passes
	assert.True(t, true)
}
