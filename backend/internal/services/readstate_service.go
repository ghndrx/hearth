package services

import (
	"context"
	"sync"
	"time"
)

type ReadState struct {
	ChannelID    string
	UserID       string
	LastReadAt   time.Time
	MentionCount int
}

type ReadStateRepository interface {
	GetReadState(ctx context.Context, channelID, userID string) (*ReadState, error)
	SetReadState(ctx context.Context, channelID, userID string, lastReadAt time.Time) error
	GetUnreadCount(ctx context.Context, channelID, userID string) (int, error)
}

type ReadStateService struct {
	mu    sync.RWMutex
	state map[string]map[string]*ReadState // channelID -> userID -> state
}

func NewReadStateService() *ReadStateService {
	return &ReadStateService{
		state: make(map[string]map[string]*ReadState),
	}
}

func (s *ReadStateService) MarkAsRead(ctx context.Context, channelID, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state[channelID] == nil {
		s.state[channelID] = make(map[string]*ReadState)
	}
	s.state[channelID][userID] = &ReadState{
		ChannelID:  channelID,
		UserID:     userID,
		LastReadAt: time.Now(),
	}
	return nil
}

func (s *ReadStateService) GetLastRead(ctx context.Context, channelID, userID string) (time.Time, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if ch, ok := s.state[channelID]; ok {
		if state, ok := ch[userID]; ok {
			return state.LastReadAt, nil
		}
	}
	return time.Time{}, nil
}
