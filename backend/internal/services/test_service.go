package services

import (
	"context"
	"errors"
	"time"
)

var (
	ErrUserNotFound = errors.New("user not found")
)

// UserRepository defines the interface for interactions with user data.
type UserRepository interface {
	FindByID(ctx context.Context, id int) (*User, error)
	Create(ctx context.Context, user *User) error
}

// ChatService defines the interface for handling chat operations.
type ChatService interface {
	GetMessages(ctx context.Context, channel string, limit int) ([]Message, error)
	SendMessage(ctx context.Context, user *User, channel, content string) error
}

// User represents a user entity in the system.
type User struct {
	ID       int
	Username string
	Created  time.Time
}

// Message represents a chat message.
type Message struct {
	ID        int
	UserID    int
	Channel   string
	Content   string
	Timestamp time.Time
}

// TestService is the concrete implementation of the Hearth/Discord functions.
type TestService struct {
	userRepo UserRepository
	chatSvc  ChatService
}

// NewTestService constructs a new TestService with dependencies.
func NewTestService(userRepo UserRepository, chatSvc ChatService) *TestService {
	return &TestService{
		userRepo: userRepo,
		chatSvc:  chatSvc,
	}
}

// GetUserService retrieves a user by ID ensuring the user exists.
func (s *TestService) GetUserService(ctx context.Context, userID int) (*User, error) {
	if userID <= 0 {
		return nil, ErrUserNotFound
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, ErrUserNotFound
	}

	return user, nil
}

// CreateChannelMessage sends a message to a specific channel.
func (s *TestService) CreateChannelMessage(ctx context.Context, userID int, channel, content string) error {
	if channel == "" || content == "" {
		return errors.New("channel and content cannot be empty")
	}

	user, err := s.GetUserService(ctx, userID)
	if err != nil {
		return err
	}

	// Pass dependencies down to perform the actual message creation
	return s.chatSvc.SendMessage(ctx, user, channel, content)
}