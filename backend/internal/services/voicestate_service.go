package services

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

type VoiceState struct {
	UserID    uuid.UUID
	ChannelID uuid.UUID
	ServerID  uuid.UUID
	Muted     bool
	Deafened  bool
	Streaming bool
}

type VoiceStateService struct {
	mu     sync.RWMutex
	states map[uuid.UUID]*VoiceState // userID -> state
}

func NewVoiceStateService() *VoiceStateService {
	return &VoiceStateService{states: make(map[uuid.UUID]*VoiceState)}
}

func (s *VoiceStateService) Join(ctx context.Context, userID, channelID, serverID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.states[userID] = &VoiceState{UserID: userID, ChannelID: channelID, ServerID: serverID}
	return nil
}

func (s *VoiceStateService) Leave(ctx context.Context, userID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.states, userID)
	return nil
}

func (s *VoiceStateService) SetMuted(ctx context.Context, userID uuid.UUID, muted bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if state, ok := s.states[userID]; ok {
		state.Muted = muted
	}
	return nil
}

func (s *VoiceStateService) GetChannelUsers(ctx context.Context, channelID uuid.UUID) ([]*VoiceState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var users []*VoiceState
	for _, state := range s.states {
		if state.ChannelID == channelID {
			users = append(users, state)
		}
	}
	return users, nil
}
