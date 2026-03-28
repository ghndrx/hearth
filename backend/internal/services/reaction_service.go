package services

import (
	"context"
	"errors"
	"sync"
)

var ErrReactionExists = errors.New("reaction already exists")
var ErrReactionNotFound = errors.New("reaction not found")

type Reaction struct {
	MessageID string
	UserID    string
	Emoji     string
}

type ReactionService struct {
	mu        sync.RWMutex
	reactions map[string][]Reaction // messageID -> reactions
}

func NewReactionService() *ReactionService {
	return &ReactionService{
		reactions: make(map[string][]Reaction),
	}
}

func (s *ReactionService) AddReaction(ctx context.Context, messageID, userID, emoji string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, r := range s.reactions[messageID] {
		if r.UserID == userID && r.Emoji == emoji {
			return ErrReactionExists
		}
	}
	s.reactions[messageID] = append(s.reactions[messageID], Reaction{
		MessageID: messageID,
		UserID:    userID,
		Emoji:     emoji,
	})
	return nil
}

func (s *ReactionService) RemoveReaction(ctx context.Context, messageID, userID, emoji string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	reactions := s.reactions[messageID]
	for i, r := range reactions {
		if r.UserID == userID && r.Emoji == emoji {
			s.reactions[messageID] = append(reactions[:i], reactions[i+1:]...)
			return nil
		}
	}
	return ErrReactionNotFound
}

func (s *ReactionService) GetReactions(ctx context.Context, messageID string) ([]Reaction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.reactions[messageID], nil
}
