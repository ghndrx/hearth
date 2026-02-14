package events

import (
	"log"
	"sync"
)

// Event represents a domain event
type Event struct {
	Type string
	Data interface{}
}

// Handler is a function that handles events
type Handler func(event Event)

// Bus is an in-memory event bus
type Bus struct {
	handlers map[string][]Handler
	mu       sync.RWMutex
}

// NewBus creates a new event bus
func NewBus() *Bus {
	return &Bus{
		handlers: make(map[string][]Handler),
	}
}

// Subscribe registers a handler for an event type
func (b *Bus) Subscribe(eventType string, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

// Unsubscribe removes a handler for an event type
// Note: Due to Go's function value semantics, this removes all handlers for the type
// Use Subscribe with caution or implement handler IDs for fine-grained control
func (b *Bus) Unsubscribe(eventType string, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()

	_ = handler // Mark as used to avoid unused parameter warning
	delete(b.handlers, eventType)
}

// Publish dispatches an event to all registered handlers
func (b *Bus) Publish(eventType string, data interface{}) {
	b.mu.RLock()
	handlers := b.handlers[eventType]
	b.mu.RUnlock()

	event := Event{
		Type: eventType,
		Data: data,
	}

	for _, handler := range handlers {
		// Run handlers asynchronously to avoid blocking
		go func(h Handler) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Event handler panic recovered: %v", r)
				}
			}()
			h(event)
		}(handler)
	}
}

// PublishSync dispatches an event synchronously (blocks until all handlers complete)
func (b *Bus) PublishSync(eventType string, data interface{}) {
	b.mu.RLock()
	handlers := b.handlers[eventType]
	b.mu.RUnlock()

	event := Event{
		Type: eventType,
		Data: data,
	}

	var wg sync.WaitGroup
	for _, handler := range handlers {
		wg.Add(1)
		go func(h Handler) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Event handler panic recovered: %v", r)
				}
			}()
			h(event)
		}(handler)
	}
	wg.Wait()
}

// SubscribeAll registers a handler for all events
func (b *Bus) SubscribeAll(handler Handler) {
	b.Subscribe("*", handler)
}

// HasHandlers checks if there are handlers for an event type
func (b *Bus) HasHandlers(eventType string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return len(b.handlers[eventType]) > 0 || len(b.handlers["*"]) > 0
}

// Event types
const (
	// User events
	UserCreated    = "user.created"
	UserUpdated    = "user.updated"
	UserDeleted    = "user.deleted"
	PresenceUpdate = "presence.updated"

	// Server events
	ServerCreated = "server.created"
	ServerUpdated = "server.updated"
	ServerDeleted = "server.deleted"

	// Member events
	MemberJoined  = "server.member_joined"
	MemberLeft    = "server.member_left"
	MemberKicked  = "server.member_kicked"
	MemberBanned  = "server.member_banned"
	MemberUpdated = "server.member_updated"

	// Channel events
	ChannelCreated = "channel.created"
	ChannelUpdated = "channel.updated"
	ChannelDeleted = "channel.deleted"

	// Message events
	MessageCreated = "message.created"
	MessageUpdated = "message.updated"
	MessageDeleted = "message.deleted"
	MessagePinned  = "message.pinned"

	// Reaction events
	ReactionAdded   = "reaction.added"
	ReactionRemoved = "reaction.removed"

	// Typing events
	TypingStarted = "typing.started"

	// Voice events
	VoiceJoined   = "voice.joined"
	VoiceLeft     = "voice.left"
	VoiceMuted    = "voice.muted"
	VoiceDeafened = "voice.deafened"
)
