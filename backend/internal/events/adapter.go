package events

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// ServiceBusAdapter wraps Bus to satisfy services.EventBus interface
type ServiceBusAdapter struct {
	bus    *Bus
	nextID atomic.Uint64
	// Map from handler function pointer (as string) to unsubscribe function
	unsubFuncs map[string]func()
	mu         sync.Mutex
}

// NewServiceBusAdapter creates an adapter for use with services
func NewServiceBusAdapter(bus *Bus) *ServiceBusAdapter {
	return &ServiceBusAdapter{
		bus:        bus,
		unsubFuncs: make(map[string]func()),
	}
}

// Publish dispatches an event with raw data
func (a *ServiceBusAdapter) Publish(eventType string, data interface{}) {
	a.bus.Publish(eventType, data)
}

// Subscribe registers a handler using the simpler func(interface{}) signature
func (a *ServiceBusAdapter) Subscribe(eventType string, handler func(data interface{})) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Use handler pointer as unique key
	handlerKey := fmt.Sprintf("%p", handler)

	// Wrap the simple handler into the typed Handler
	unsub := a.bus.Subscribe(eventType, func(event Event) {
		handler(event.Data)
	})

	// Store the unsubscribe function
	a.unsubFuncs[handlerKey] = unsub
}

// Unsubscribe removes a specific handler
func (a *ServiceBusAdapter) Unsubscribe(eventType string, handler func(data interface{})) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Use handler pointer to find the subscription
	handlerKey := fmt.Sprintf("%p", handler)
	if unsub, ok := a.unsubFuncs[handlerKey]; ok {
		unsub()
		delete(a.unsubFuncs, handlerKey)
	}
}
