package events

// ServiceBusAdapter wraps Bus to satisfy services.EventBus interface
type ServiceBusAdapter struct {
	bus *Bus
}

// NewServiceBusAdapter creates an adapter for use with services
func NewServiceBusAdapter(bus *Bus) *ServiceBusAdapter {
	return &ServiceBusAdapter{bus: bus}
}

// Publish dispatches an event with raw data
func (a *ServiceBusAdapter) Publish(eventType string, data interface{}) {
	a.bus.Publish(eventType, data)
}

// Subscribe registers a handler using the simpler func(interface{}) signature
func (a *ServiceBusAdapter) Subscribe(eventType string, handler func(data interface{})) {
	// Wrap the simple handler into the typed Handler
	a.bus.Subscribe(eventType, func(event Event) {
		handler(event.Data)
	})
}

// Unsubscribe removes a handler (simplified - removes all handlers for type)
func (a *ServiceBusAdapter) Unsubscribe(eventType string, handler func(data interface{})) {
	// Note: simplified implementation that removes all handlers
	// A production implementation would track handler references
	a.bus.Unsubscribe(eventType, nil)
}
