package services

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"
)

var ErrEmojiNotFound = errors.New("emoji not found")

type CustomEmoji struct {
	ID       uuid.UUID
	ServerID uuid.UUID
	Name     string
	URL      string
	Animated bool
}

type EmojiService struct {
	mu     sync.RWMutex
	emojis map[uuid.UUID]*CustomEmoji
}

func NewEmojiService() *EmojiService {
	return &EmojiService{emojis: make(map[uuid.UUID]*CustomEmoji)}
}

func (s *EmojiService) Create(ctx context.Context, serverID uuid.UUID, name, url string, animated bool) (*CustomEmoji, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	e := &CustomEmoji{ID: uuid.New(), ServerID: serverID, Name: name, URL: url, Animated: animated}
	s.emojis[e.ID] = e
	return e, nil
}

func (s *EmojiService) GetServerEmojis(ctx context.Context, serverID uuid.UUID) ([]*CustomEmoji, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var emojis []*CustomEmoji
	for _, e := range s.emojis {
		if e.ServerID == serverID {
			emojis = append(emojis, e)
		}
	}
	return emojis, nil
}

func (s *EmojiService) Delete(ctx context.Context, emojiID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.emojis[emojiID]; !ok {
		return ErrEmojiNotFound
	}
	delete(s.emojis, emojiID)
	return nil
}
