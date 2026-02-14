package services

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// Storage defines the interface for persisting Hearth state.
// This allows us to mock it in tests without touching a real database.
type Storage interface {
	// SaveMessage persists a new message.
	SaveMessage(msg *Message) error
	// GetUser fetches a user by ID.
	GetUser(userID string) (*User, error)
	// GetMessagesByChannel fetches historical messages.
	GetMessagesByChannel(channelID string) ([]*Message, error)
	// CreateUser ensures a user exists.
	CreateUser(user *User) error
}

// Message represents a chat message in Hearth.
type Message struct {
	ID        string
	ChannelID string
	Content   string
	UserID    string
	Timestamp time.Time
}

// User represents a Discord account in Hearth.
type User struct {
	ID       string
	Username string
	Email    string
}

// MainService handles the core business logic of Hearth.
type MainService struct {
	store Storage
	mu    sync.RWMutex // Locking strategy for safe concurrent access
}

// NewMainService creates a new instance of the Hearth main service.
func NewMainService(store Storage) *MainService {
	return &MainService{
		store: store,
	}
}

// SendMessage handles the logic to send a message into a channel.
func (s *MainService) SendMessage(channelID, content, userID string) (*Message, error) {
	if channelID == "" {
		return nil, errors.New("channel ID cannot be empty")
	}
	if content == "" {
		return nil, errors.New("content cannot be empty")
	}
	if userID == "" {
		return nil, errors.New("user ID cannot be empty")
	}

	// In a real production app, we might verify the user's permissions here.
	// For simplicity, we just fetch them to check existence.
	_, err := s.store.GetUser(userID)
	if err != nil {
		return nil, fmt.Errorf("user %s not found", userID)
	}

	msg := &Message{
		ID:        generateID(),
		ChannelID: channelID,
		Content:   content,
		UserID:    userID,
		Timestamp: time.Now(),
	}

	if err := s.store.SaveMessage(msg); err != nil {
		return nil, fmt.Errorf("failed to save message: %w", err)
	}

	return msg, nil
}

// GetChannelHistory retrieves messages for a specific channel.
func (s *MainService) GetChannelHistory(channelID string, limit int) ([]*Message, error) {
	if channelID == "" {
		return nil, errors.New("channel ID cannot be empty")
	}
	if limit <= 0 {
		return nil, errors.New("limit must be > 0")
	}

	return s.store.GetMessagesByChannel(channelID)
}

// CreateUser creates a new user in the system.
func (s *MainService) CreateUser(username, email, userID string) error {
	if username == "" || email == "" || userID == "" {
		return errors.New("username, email, and ID are required")
	}

	user := &User{
		ID:       userID,
		Username: username,
		Email:    email,
	}

	// Attempt to create; store handles duplicates
	return s.store.CreateUser(user)
}

// generateID is a simple pseudo-random ID generator for this demo.
func generateID() string {
	return fmt.Sprintf("msg-%d", time.Now().UnixNano())
}