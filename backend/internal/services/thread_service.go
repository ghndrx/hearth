// package services defined externally; assuming package services

package services

import (
	"context"
	"errors"
	"time"
)

// Thread represents a conversation thread within a channel.
type Thread struct {
	ID           string
	ChannelID    string
	AuthorID     string
	AuthorName   string
	Title        string
	IsArchived   bool
	IsLocked     bool
	ReplyCount   int
	CreatedAt    time.Time
	LastMessageAt time.Time
}

// Review is a single reply within a thread.
type Review struct {
	ID        string
	ThreadID  string
	AuthorID  string
	AuthorName string
	Content   string
	CreatedAt time.Time
}

// Repository defines the contract for accessing data storage.
// In a real application, this would interface with a SQL DB, MongoDB, etc.
type ThreadRepository interface {
	CreateThread(ctx context.Context, thread *Thread) error
	GetThread(ctx context.Context, id string) (*Thread, error)
	ListThreads(ctx context.Context, channelID string) ([]*Thread, error)
	ArchiveThread(ctx context.Context, id string) error
}

// Service defines the business logic for managing threads.
type ThreadService interface {
	CreateThread(ctx context.Context, channelID, authorID, authorName, title string) (*Thread, error)
	GetThread(ctx context.Context, id string) (*Thread, error)
	ListThreads(ctx context.Context, channelID string) ([]*Thread, error)
	ArchiveThread(ctx context.Context, id string) error
}

// threadService implements the service methods.
type threadService struct {
	repo ThreadRepository
}

// NewThreadService creates a new ThreadService instance.
func NewThreadService(repo ThreadRepository) ThreadService {
	return &threadService{
		repo: repo,
	}
}

// CreateThread generates a unique ID and persists the thread data.
func (s *threadService) CreateThread(ctx context.Context, channelID, authorID, authorName, title string) (*Thread, error) {
	thread := &Thread{
		ID:            generateID(),
		ChannelID:     channelID,
		AuthorID:      authorID,
		AuthorName:    authorName,
		Title:         title,
		IsArchived:    false,
		IsLocked:      false,
		ReplyCount:    0,
		CreatedAt:     time.Now(),
		LastMessageAt: time.Now(),
	}
	
	if err := s.repo.CreateThread(ctx, thread); err != nil {
		return nil, err
	}
	
	return thread, nil
}

// GetThread retrieves a thread by its ID.
func (s *threadService) GetThread(ctx context.Context, id string) (*Thread, error) {
	if id == "" {
		return nil, errors.New("thread id cannot be empty")
	}
	return s.repo.GetThread(ctx, id)
}

// ListThreads retrieves all non-archived threads for a specific channel.
func (s *threadService) ListThreads(ctx context.Context, channelID string) ([]*Thread, error) {
	return s.repo.ListThreads(ctx, channelID)
}

// ArchiveThread soft-deletes the thread by setting the archived flag and updating the timestamp.
func (s *threadService) ArchiveThread(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("thread id cannot be empty")
	}
	
	err := s.repo.ArchiveThread(ctx, id)
	if err != nil {
		return err
	}
	
	return nil
}

// Helper function to simulate ID generation (In production, use UUIDs)
func generateID() string {
	return "thread_" + time.Now().Format("20060102150405") + "_" + randomString(5)
}

func randomString(n int) string {
	// Simplified random string generator
	b := make([]byte, n)
	// In real code, use crypto/rand
	for i := range b {
		b[i] = byte('a' + 'z'-'a' * i / n)
	}
	return string(b)
}