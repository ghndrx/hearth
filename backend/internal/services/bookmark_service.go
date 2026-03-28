package services

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"
)

var ErrBookmarkNotFound = errors.New("bookmark not found")

type Bookmark struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	MessageID uuid.UUID
	Note      string
}

type BookmarkService struct {
	mu        sync.RWMutex
	bookmarks map[uuid.UUID][]*Bookmark
}

func NewBookmarkService() *BookmarkService {
	return &BookmarkService{
		bookmarks: make(map[uuid.UUID][]*Bookmark),
	}
}

func (s *BookmarkService) Add(ctx context.Context, userID, messageID uuid.UUID, note string) (*Bookmark, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	b := &Bookmark{
		ID:        uuid.New(),
		UserID:    userID,
		MessageID: messageID,
		Note:      note,
	}
	s.bookmarks[userID] = append(s.bookmarks[userID], b)
	return b, nil
}

func (s *BookmarkService) List(ctx context.Context, userID uuid.UUID) ([]*Bookmark, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.bookmarks[userID], nil
}

func (s *BookmarkService) Remove(ctx context.Context, userID, bookmarkID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	bms := s.bookmarks[userID]
	for i, b := range bms {
		if b.ID == bookmarkID {
			s.bookmarks[userID] = append(bms[:i], bms[i+1:]...)
			return nil
		}
	}
	return ErrBookmarkNotFound
}
