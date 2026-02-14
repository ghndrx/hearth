package services

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

var ErrThreadNotFound = errors.New("thread not found")

type Thread struct {
	ID        uuid.UUID
	ChannelID uuid.UUID
	ParentID  uuid.UUID
	Name      string
	CreatedAt time.Time
	Archived  bool
}

type ThreadService struct {
	mu      sync.RWMutex
	threads map[uuid.UUID]*Thread
}

func NewThreadService() *ThreadService {
	return &ThreadService{
		threads: make(map[uuid.UUID]*Thread),
	}
}

func (s *ThreadService) CreateThread(ctx context.Context, channelID, parentID uuid.UUID, name string) (*Thread, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	thread := &Thread{
		ID:        uuid.New(),
		ChannelID: channelID,
		ParentID:  parentID,
		Name:      name,
		CreatedAt: time.Now(),
	}
	s.threads[thread.ID] = thread
	return thread, nil
}

func (s *ThreadService) GetThread(ctx context.Context, threadID uuid.UUID) (*Thread, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if thread, ok := s.threads[threadID]; ok {
		return thread, nil
	}
	return nil, ErrThreadNotFound
}

func (s *ThreadService) ArchiveThread(ctx context.Context, threadID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if thread, ok := s.threads[threadID]; ok {
		thread.Archived = true
		return nil
	}
	return ErrThreadNotFound
}
