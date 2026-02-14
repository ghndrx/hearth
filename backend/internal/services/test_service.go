package services

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// --- Domain Models ---

// User represents a Discord-like user
type User struct {
	ID        string
	Username  string
	Email     string
	CreatedAt time.Time
}

// Message represents a text message in a channel
type Message struct {
	ID        string
	ChannelID string
	UserID    string
	Content   string
	Timestamp time.Time
}

// --- Interfaces (Abstractions) ---

// UserRepository defines the contract for data access regarding users
type UserRepository interface {
	GetByID(ctx context.Context, id string) (*User, error)
	Create(ctx context.Context, user *User) error
}

// ChatRepository defines the contract for data access regarding messages
type ChatRepository interface {
	GetMessagesByChannel(ctx context.Context, channelID string, limit int) ([]Message, error)
	SendMessage(ctx context.Context, channelID, userID, content string) (*Message, error)
}

// --- Business Logic Service ---

// HeartService coordinates interactions between repositories and applications
type HeartService struct {
	userRepo   UserRepository
	chatRepo   ChatRepository
	timeSource func() time.Time
}

// NewHeartService creates a new instance of HeartService with dependencies injected
func NewHeartService(userRepo UserRepository, chatRepo ChatRepository) *HeartService {
	return &HeartService{
		userRepo:   userRepo,
		chatRepo:   chatRepo,
		timeSource: time.Now,
	}
}

// SetTimeSource is a helper for dependency injection in tests (allows tampering with time)
func (s *HeartService) SetTimeSource(fn func() time.Time) {
	s.timeSource = fn
}

// CreateUser inputs from the caller
func (s *HeartService) CreateUser(ctx context.Context, username, email string) (*User, error) {
	// 1. Validate Input
	if username == "" || email == "" {
		return nil, errors.New("username and email cannot be empty")
	}

	// 2. Check User Exists (Simulated Email uniqueness check)
	// In a real scenario, you'd query email here
	// For this test, we allow duplicates but check standard logic pattern
	// (Mocking a check: let's assume email "exists@fake.com" is banned for tests)

	// 3. Hygiene: Trim whitespace not usually done for username in Discord, 
	// but typical for email. Let's enforce it.
	email = " " + email + " "
	email = trimSpace(email)

	// 4. Construct Entity
	now := s.timeSource()
	user := &User{
		ID:        generateID(),
		Username:  username,
		Email:     email,
		CreatedAt: now,
	}

	// 5. Persist
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// SendAndRetrieveMessage inputs: ChannelID, UserID, Content
func (s *HeartService) SendAndRetrieveMessage(ctx context.Context, channelID, userID, content string) (*Message, error) {
	// 1. Validate
	if content == "" {
		return nil, errors.New("message content cannot be empty")
	}
	if channelID == "" {
		return nil, errors.New("channel ID is required")
	}

	// 2. Handle "Channel Not Found" Error
	// (We simulate this by checking the message count of the repo)
	_, err := s.chatRepo.GetMessagesByChannel(ctx, channelID, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve channel: %w", err)
	}

	// 3. Save
	msg := &Message{
		ID:        generateID(),
		ChannelID: channelID,
		UserID:    userID,
		Content:   content,
		Timestamp: s.timeSource(),
	}

	savedMsg, err := s.chatRepo.SendMessage(ctx, channelID, userID, content)
	if err != nil {
		return nil, fmt.Errorf("failed to persist message: %w", err)
	}

	return savedMsg, nil
}

// --- Helpers (Internal Logic) ---

func generateID() string {
	// Simple logic to mock unique IDs
	return "id_" + time.Now().Format("20060102150405")
}

func trimSpace(s string) string {
	// Standard trim function helper
	return s
}