package services

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"hearth/internal/models"
)

const (
	// TypingTTL is how long a typing indicator lasts
	TypingTTL = 10 * time.Second
)

// TypingService manages typing indicators
type TypingService struct {
	mu       sync.RWMutex
	typing   map[uuid.UUID]map[uuid.UUID]time.Time // channelID -> userID -> timestamp
	eventBus EventBus
}

// NewTypingService creates a new typing service
func NewTypingService(eventBus EventBus) *TypingService {
	svc := &TypingService{
		typing:   make(map[uuid.UUID]map[uuid.UUID]time.Time),
		eventBus: eventBus,
	}

	// Start background cleanup goroutine
	go svc.cleanupLoop()

	return svc
}

// StartTyping records that a user started typing in a channel
func (s *TypingService) StartTyping(ctx context.Context, channelID, userID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.typing[channelID] == nil {
		s.typing[channelID] = make(map[uuid.UUID]time.Time)
	}

	now := time.Now()
	s.typing[channelID][userID] = now

	// Publish typing event for WebSocket broadcast
	if s.eventBus != nil {
		indicator := &models.TypingIndicator{
			ChannelID: channelID,
			UserID:    userID,
			Timestamp: now,
		}
		s.eventBus.Publish("typing.start", indicator)
	}

	return nil
}

// StopTyping removes a user's typing indicator (e.g., when they send a message)
func (s *TypingService) StopTyping(ctx context.Context, channelID, userID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.typing[channelID] != nil {
		delete(s.typing[channelID], userID)
		if len(s.typing[channelID]) == 0 {
			delete(s.typing, channelID)
		}
	}

	return nil
}

// GetTypingUsers returns a list of users currently typing in a channel
func (s *TypingService) GetTypingUsers(ctx context.Context, channelID uuid.UUID) ([]models.TypingIndicator, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var indicators []models.TypingIndicator
	now := time.Now()

	if channelUsers, ok := s.typing[channelID]; ok {
		for userID, ts := range channelUsers {
			if now.Sub(ts) < TypingTTL {
				indicators = append(indicators, models.TypingIndicator{
					ChannelID: channelID,
					UserID:    userID,
					Timestamp: ts,
				})
			}
		}
	}

	return indicators, nil
}

// IsTyping checks if a specific user is currently typing in a channel
func (s *TypingService) IsTyping(ctx context.Context, channelID, userID uuid.UUID) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if channelUsers, ok := s.typing[channelID]; ok {
		if ts, exists := channelUsers[userID]; exists {
			return time.Since(ts) < TypingTTL, nil
		}
	}

	return false, nil
}

// cleanupLoop periodically cleans up expired typing indicators
func (s *TypingService) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.cleanup()
	}
}

// cleanup removes expired typing indicators
func (s *TypingService) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	for channelID, channelUsers := range s.typing {
		for userID, ts := range channelUsers {
			if now.Sub(ts) >= TypingTTL {
				delete(channelUsers, userID)
			}
		}
		if len(channelUsers) == 0 {
			delete(s.typing, channelID)
		}
	}
}

// GetTypingUserIDs returns just the user IDs currently typing (convenience method)
func (s *TypingService) GetTypingUserIDs(ctx context.Context, channelID uuid.UUID) ([]uuid.UUID, error) {
	indicators, err := s.GetTypingUsers(ctx, channelID)
	if err != nil {
		return nil, err
	}

	userIDs := make([]uuid.UUID, len(indicators))
	for i, ind := range indicators {
		userIDs[i] = ind.UserID
	}

	return userIDs, nil
}

// ClearChannel removes all typing indicators for a channel (e.g., when channel is deleted)
func (s *TypingService) ClearChannel(ctx context.Context, channelID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.typing, channelID)
	return nil
}
