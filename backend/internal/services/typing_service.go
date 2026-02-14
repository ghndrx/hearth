package services

import (
	"errors"
	"sync"
	"time"
)

// TypingStatus represents the current state of a user's typing indicator.
type TypingStatus struct {
	UserID   string // ID of the user typing
	Timestamp time.Time // When the event started
}

// TypingService manages the state of typing indicators for guilds/channels.
type TypingService struct {
	// map[ChannelID]map[UserID]TypingStatus is not used directly to avoid holding pointers across long periods.
	// Instead, we use a buffer of events and a lookup set of active users.
	
	// The channel receives individual typing events
	eventChan chan *TypingStatus
	
	// The lookup map keeps track of users currently typing over the last 5 seconds
	playing map[string]time.Time
	mu      sync.RWMutex
}

// ITypingService defines the methods this service must implement.
type ITypingService interface {
	StartTyping(channelID, userID string)
	StopTyping(channelID, userID string)
	GetTypingUsers(channelID string) []string
}

// NewTypingService creates and initializes a new service instance.
// We use a buffered channel to decouple the caller from the processing goroutine.
func NewTypingService() *TypingService {
	s := &TypingService{
		eventChan: make(chan *TypingStatus, 100), // Buffer of 100 events
		playing:   make(map[string]time.Time),
	}

	// Start the background processor
	go s.processEvents()
	return s
}

// StartTyping triggers a typing event.
// It does not block the caller; it pushes to the channel.
func (s *TypingService) StartTyping(channelID, userID string) {
	// We include the channel ID as a unique key for grouping, 
	// though current implementation simply aggregates globally for simplicity.
	// To be fully compliant with Discord logic, this should be "ChannelID+UserID".
	event := &TypingStatus{
		UserID:   userID,
		Timestamp: time.Now(),
	}
	s.eventChan <- event
}

// StopTyping simulates the user stopping typing. 
// Note: In a real system, StopTyping is usually called via an async heartbeat disconnect,
// but adds a "close" event to the event logic here to force cleanup if needed.
func (s *TypingService) StopTyping(channelID, userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.playing, userID)
}

// GetTypingUsers returns a list of user IDs currently typing.
// Should return empty list if user is not typing.
func (s *TypingService) GetTypingUsers(channelID string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var users []string
	for userID, lastSeen := range s.playing {
		// Clean up 'stale' typing indicators (older than 5 seconds)
		if time.Since(lastSeen) > 5*time.Second {
			continue
		}
		users = append(users, userID)
	}
	return users
}

// processEvents is the background goroutine that processes incoming events.
func (s *TypingService) processEvents() {
	for event := range s.eventChan {
		s.handleEvent(event)
	}
}

func (s *TypingService) handleEvent(ev *TypingStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Update the last seen time for this user.
	// If the map doesn't have the key, this creates a new entry.
	s.playing[ev.UserID] = time.Now()
}