package services

import (
	"context"
	"sync"
	"time"
)

// TypingService manages typing indicators
type TypingService struct {
	mu     sync.RWMutex
	typing map[string]map[string]time.Time // channelID -> userID -> timestamp
	ttl    time.Duration
}

func NewTypingService() *TypingService {
	return &TypingService{
		typing: make(map[string]map[string]time.Time),
		ttl:    10 * time.Second,
	}
}

func (s *TypingService) StartTyping(ctx context.Context, channelID, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.typing[channelID] == nil {
		s.typing[channelID] = make(map[string]time.Time)
	}
	s.typing[channelID][userID] = time.Now()
	return nil
}

func (s *TypingService) StopTyping(ctx context.Context, channelID, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.typing[channelID] != nil {
		delete(s.typing[channelID], userID)
	}
	return nil
}

func (s *TypingService) GetTypingUsers(ctx context.Context, channelID string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var users []string
	now := time.Now()

	if channelUsers, ok := s.typing[channelID]; ok {
		for userID, ts := range channelUsers {
			if now.Sub(ts) < s.ttl {
				users = append(users, userID)
			}
		}
	}
	return users, nil
}
