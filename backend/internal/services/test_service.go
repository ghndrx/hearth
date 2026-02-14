package services

import (
	"context"
	"fmt"
	"time"
)

// User represents a user entity in the system.
type User struct {
	ID        string
	Name      string
	Email     string
	CreatedAt time.Time
}

// Message represents a text message sent within a channel.
type Message struct {
	ID        string
	Content   string
	SenderID  string
	CreatedAt time.Time
}

// ChannelID represents the unique identifier for a text channel.
type ChannelID string

// Storage defines the interface for database operations.
// This ensures the service can be easily tested using mocks.
type Storage interface {
	CreateUser(ctx context.Context, email, name string) (*User, error)
	GetUserByID(ctx context.Context, id string) (*User, error)
	CreateMessage(ctx context.Context, channelID ChannelID, senderID string, content string) (*Message, error)
	GetMessagesInChannel(ctx context.Context, channelID ChannelID, limit int) ([]*Message, error)
}

// HearthService aggregates logic for the Discord clone.
type HearthService struct {
	store Storage
}

// NewHearthService initializes the service with a given storage interface.
func NewHearthService(store Storage) *HearthService {
	return &HearthService{store: store}
}

// CreateUser registers a new user.
func (s *HearthService) CreateUser(ctx context.Context, email, name string) (*User, error) {
	// Basic validation
	if email == "" || name == "" {
		return nil, fmt.Errorf("email and name cannot be empty")
	}

	// Business logic: Check if user exists (simplified here)
	existingUser, _ := s.store.GetUserByID(ctx, email) // Using email as ID for simplicity
	if existingUser != nil {
		return nil, fmt.Errorf("user with email %s already exists", email)
	}

	user := &User{
		ID:        email,
		Name:      name,
		Email:     email,
		CreatedAt: time.Now(),
	}

	if err := s.store.CreateUser(ctx, email, name); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// CreateMessage sends a message to a specific channel.
func (s *HearthService) CreateMessage(ctx context.Context, channelID ChannelID, senderID string, content string) (*Message, error) {
	if content == "" {
		return nil, fmt.Errorf("message content cannot be empty")
	}

	msg := &Message{
		ID:        fmt.Sprintf("%s-%d", senderID, time.Now().UnixNano()),
		Content:   content,
		SenderID:  senderID,
		CreatedAt: time.Now(),
	}

	createdMsg, err := s.store.CreateMessage(ctx, channelID, senderID, content)
	if err != nil {
		return nil, fmt.Errorf("failed to persist message: %w", err)
	}

	return createdMsg, nil
}

// GetChatHistory fetches messages from a channel, optionally limiting the amount.
func (s *HearthService) GetChatHistory(ctx context.Context, channelID ChannelID, limit int) ([]*Message, error) {
	if limit <= 0 {
		limit = 50 // Default limit
	}

	messages, err := s.store.GetMessagesInChannel(ctx, channelID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve chat history: %w", err)
	}

	return messages, nil
}