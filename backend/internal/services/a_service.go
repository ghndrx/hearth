package services

import (
	"errors"
	"regexp"
	"sync"
)

// HeartbeatEvent represents a message sent in a Discord-like channel.
type HeartbeatEvent struct {
	User      string
	Content   string
	Timestamp int64
}

// IEventHandler defines the contract for handling incoming events.
// This allows the core service to remain decoupled from the specific logic implementation.
type IEventHandler interface {
	Handle(event HeartbeatEvent) error
}

// IService defines the core contract for the Hearth service.
type IService interface {
	// RegisterHandler adds specific logic to handle events for a specific user or channel.
	RegisterHandler(user string, handler IEventHandler)

	// SendEvent publishes an event to the service for processing.
	SendEvent(event HeartbeatEvent)

	// Close gracefully shuts down any internal goroutines or connections.
	Close() error
}

// HeartService is the concrete implementation of IService.
// It maintains a registry of handlers:
// Key: Username
// Value: IEventHandler
type HeartService struct {
	handlers map[string]IEventHandler
	mu       sync.RWMutex
	done     chan struct{}
}

// NewHeartService initializes a new instance of HeartService.
func NewHeartService() *HeartService {
	return &HeartService{
		handlers: make(map[string]IEventHandler),
		done:     make(chan struct{}),
	}
}

// RegisterHandler creates a rule for a specific user.
// We assume the "user" is the trigger for the rule in this example.
func (s *HeartService) RegisterHandler(user string, handler IEventHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// If you want to allow multiple handlers per user, return multiple channels
	// For simplicity here, we overwrite.
	s.handlers[user] = handler
}

// SendEvent acts as the API Gateway. It accepts events and routes them to handlers.
func (s *HeartService) SendEvent(event HeartbeatEvent) {
	s.mu.RLock()
	handler, found := s.handlers[event.User]
	s.mu.RUnlock()

	if !found {
		return // Silent fail or could log: "No handler for user..."
	}

	// Execute handler in a goroutine to prevent blocking the gateway directly
	go func() {
		if err := handler.Handle(event); err != nil {
			// In a real app, this error would likely go to an ErrorChannel of the service
			// or a specific logging package.
			_ = err
		}
	}()
}

// Close cleans up resources.
func (s *HeartService) Close() error {
	close(s.done)
	return nil
}

// Misc: Example of a command processor that implements IEventHandler.
type ChatCommandHandler struct {
	Prefix string
}

func (h *ChatCommandHandler) Handle(event HeartbeatEvent) error {
	if event.Content == h.Prefix+"hello" {
		// Simulating a response/side effect
		event.Content = "Hello back!"
		return nil
	}
	return errors.New("command not recognized")
}

// Misc: Example of a filter that implements IEventHandler.
type AntiSpamHandler struct {
	Regexp *regexp.Regexp
}

func (h *AntiSpamHandler) Handle(event HeartbeatEvent) error {
	if h.Regexp.MatchString(event.Content) {
		return errors.New("content blocked by anti-spam filter")
	}
	return nil
}