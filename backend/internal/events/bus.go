package events

import (
	"log"
	"sync"
	"sync/atomic"
)

// Event represents a domain event
type Event struct {
	Type string
	Data interface{}
}

// Handler is a function that handles events
type Handler func(event Event)

// handlerEntry wraps a handler with a unique ID
type handlerEntry struct {
	id      uint64
	handler Handler
}

// Bus is an in-memory event bus
type Bus struct {
	handlers map[string][]handlerEntry
	nextID   atomic.Uint64
	mu       sync.RWMutex
}

// NewBus creates a new event bus
func NewBus() *Bus {
	return &Bus{
		handlers: make(map[string][]handlerEntry),
	}
}

// Subscribe registers a handler for an event type
// Returns an unsubscribe function that can be called to remove the handler
func (b *Bus) Subscribe(eventType string, handler Handler) func() {
	b.mu.Lock()
	defer b.mu.Unlock()

	id := b.nextID.Add(1)
	entry := handlerEntry{id: id, handler: handler}
	b.handlers[eventType] = append(b.handlers[eventType], entry)

	// Return unsubscribe function
	return func() {
		b.unsubscribeByID(eventType, id)
	}
}

// unsubscribeByID removes a handler by its ID
func (b *Bus) unsubscribeByID(eventType string, id uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	entries := b.handlers[eventType]
	for i, entry := range entries {
		if entry.id == id {
			// Remove handler while preserving order
			b.handlers[eventType] = append(entries[:i], entries[i+1:]...)
			break
		}
	}

	// Clean up empty handler lists
	if len(b.handlers[eventType]) == 0 {
		delete(b.handlers, eventType)
	}
}

// Publish dispatches an event to all registered handlers
func (b *Bus) Publish(eventType string, data interface{}) {
	b.mu.RLock()
	entries := b.handlers[eventType]
	// Also get wildcard handlers
	wildcardEntries := b.handlers["*"]
	b.mu.RUnlock()

	event := Event{
		Type: eventType,
		Data: data,
	}

	// Combine regular and wildcard handlers
	allEntries := make([]handlerEntry, 0, len(entries)+len(wildcardEntries))
	allEntries = append(allEntries, entries...)
	allEntries = append(allEntries, wildcardEntries...)

	for _, entry := range allEntries {
		// Run handlers asynchronously to avoid blocking
		go func(h Handler) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Event handler panic recovered: %v", r)
				}
			}()
			h(event)
		}(entry.handler)
	}
}

// PublishSync dispatches an event synchronously (blocks until all handlers complete)
func (b *Bus) PublishSync(eventType string, data interface{}) {
	b.mu.RLock()
	entries := b.handlers[eventType]
	// Also get wildcard handlers
	wildcardEntries := b.handlers["*"]
	b.mu.RUnlock()

	event := Event{
		Type: eventType,
		Data: data,
	}

	// Combine regular and wildcard handlers
	allEntries := make([]handlerEntry, 0, len(entries)+len(wildcardEntries))
	allEntries = append(allEntries, entries...)
	allEntries = append(allEntries, wildcardEntries...)

	var wg sync.WaitGroup
	for _, entry := range allEntries {
		wg.Add(1)
		go func(h Handler) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Event handler panic recovered: %v", r)
				}
			}()
			h(event)
		}(entry.handler)
	}
	wg.Wait()
}

// SubscribeAll registers a handler for all events
// Returns an unsubscribe function
func (b *Bus) SubscribeAll(handler Handler) func() {
	return b.Subscribe("*", handler)
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
