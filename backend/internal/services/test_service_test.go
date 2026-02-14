package services

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// --- Interfaces ---

// StorageService defines the contract for data access
type StorageService interface {
	SaveMessage(ctx context.Context, id, channelID, content string) (string, error)
	GetMessages(ctx context.Context, channelID string) ([]Message, error)
}

// ChannelService defines the contract for channel logic
type ChannelService interface {
	ValidateChannelID(id string) bool
	GetChannelName(id string) (string, error)
}

// Message represents a chat message
type Message struct {
	ID        string
	ChannelID string
	Content   string
	Timestamp time.Time
	AuthorID  string
}

// Channel represents a chat channel
type Channel struct {
	ID   string
	Name string
	Type string
}

// AuthService simulates authentication
type AuthService interface {
	ValidateToken(token string) (bool, string)
}

// --- Core Application Logic ---

// MessageService is the main service entry point
type MessageService interface {
	SendMessage(ctx context.Context, channelID, content, authorID string) (string, error)
	GetHistory(ctx context.Context, channelID string) ([]Message, error)
}

// messageService is the private implementation
type messageService struct {
	channelSvc ChannelService
	storageSvc StorageService
	authSvc    AuthService
	mu         sync.Mutex // Ensures mock database isolation
}

// NewMessageService configures the service
func NewMessageService(channelSvc ChannelService, storageSvc StorageService, authSvc AuthService) MessageService {
	return &messageService{
		channelSvc: channelSvc,
		storageSvc: storageSvc,
		authSvc:    authSvc,
	}
}

// SendMessage handles the request logic
func (s *messageService) SendMessage(ctx context.Context, channelID, content, authorID string) (string, error) {
	// 1. Validate Channel
	if !s.channelSvc.ValidateChannelID(channelID) {
		return "", fmt.Errorf("service: invalid channel ID '%s'", channelID)
	}

	// 2. Validate Content
	if content == "" {
		return "", fmt.Errorf("service: content cannot be empty")
	}

	// 3. Check Authentication (Business Rule)
	// In this example, we don't use authorID in storage, but we validate it.
	if authorID == "" {
		return "", fmt.Errorf("service: author ID cannot be empty")
	}
	_, _ = s.authSvc.ValidateToken(authorID)

	// 4. Generate ID
	s.mu.Lock()
	msgID := fmt.Sprintf("msg-%d", time.Now().UnixNano())
	s.mu.Unlock()

	// 5. Persist
	return msgID, s.storageSvc.SaveMessage(ctx, msgID, channelID, content)
}

// GetHistory retrieves data
func (s *messageService) GetHistory(ctx context.Context, channelID string) ([]Message, error) {
	if !s.channelSvc.ValidateChannelID(channelID) {
		return nil, fmt.Errorf("service: access denied or invalid channel ID '%s'", channelID)
	}

	return s.storageSvc.GetMessages(ctx, channelID)
}

// MockDatabase simulates a SQL database
type MockDatabase struct {
	Messages map[string]Message
	Channels map[string]Channel
	mu       sync.RWMutex
}

func NewMockDatabase() *MockDatabase {
	return &MockDatabase{
		Messages: make(map[string]Message),
		Channels: make(map[string]Channel),
	}
}

func (db *MockDatabase) SaveMessage(ctx context.Context, id, channelID, content string) (string, error) {
	db.mu.Lock()
	defer db.mu.Unlock()
	
	msg := Message{
		ID:        id,
		ChannelID: channelID,
		Content:   content,
		Timestamp: time.Now(),
		AuthorID:  "system",
	}
	db.Messages[id] = msg
	return id, nil
}

func (db *MockDatabase) GetMessages(ctx context.Context, channelID string) ([]Message, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var results []Message
	for _, msg := range db.Messages {
		if msg.ChannelID == channelID {
			results = append(results, msg)
		}
	}
	return results, nil
}

// MockChannelService is a mock implementation of ChannelService
type MockChannelService struct {
	ValidIDs map[string]bool // true = valid
}

func NewMockChannelService() *MockChannelService {
	return &MockChannelService{
		ValidIDs: map[string]bool{
			"general":   true,
			"random":    true,
			"admin":     false, // Invalid
			"deep-deep": true,
		},
	}
}

func (m *MockChannelService) ValidateChannelID(id string) bool {
	return m.ValidIDs[id]
}

func (m *MockChannelService) GetChannelName(id string) (string, error) {
	if !m.ValidIDs[id] {
		return "", fmt.Errorf("channel not found")
	}
	return "General Chat", nil
}