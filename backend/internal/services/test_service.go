package services

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// --- Interfaces ---

// StorageService handles data persistence (simulated here with an internal map or DB).
type StorageService interface {
	SaveMessage(context.Context, string, string, string) (string, error)
	GetMessages(context.Context, string) ([]Message, error)
}

// ChannelService handles channel logic (e.g., validation, retrieval).
type ChannelService interface {
	ValidateChannelID(string) bool
	GetChannelName(string) (string, error)
}

// Message represents a chat message.
type Message struct {
	ID        string
	ChannelID string
	Content   string
	Timestamp time.Time
	AuthorID  string
}

// Channel represents a chat channel.
type Channel struct {
	ID   string
	Name string
	Type string
}

// --- Middleware ---

// Middleware type allows for request interception
type Middleware func(MessageService) MessageService

// --- Core Business Logic ---

// MessageService is the main interface for interacting with chat functionality.
type MessageService interface {
	SendMessage(ctx context.Context, channelID, content, authorID string) (string, error)
	GetHistory(ctx context.Context, channelID string) ([]Message, error)
}

// messageService implements MessageService
type messageService struct {
	channelSvc ChannelService
	storageSvc StorageService
}

// AuthService simulates user authentication
type AuthService interface {
	ValidateToken(string) (bool, string)
}

type authService struct {
	users map[string]struct{} // simple mock: token -> user existence
}

func NewAuthService() AuthService {
	return &authService{
		users: map[string]struct{}{
			"token-1": {},
		},
	}
}

func (a *authService) ValidateToken(token string) (bool, string) {
	_, exists := a.users[token]
	return exists, "user-123"
}

// NewMessageService constructs the application logic
func NewMessageService(channelSvc ChannelService, storageSvc StorageService) MessageService {
	return &messageService{
		channelSvc: channelSvc,
		storageSvc: storageSvc,
	}
}

// NewAuthService is a helper constructor
func NewAuthService() AuthService {
	return &authService{
		users: map[string]struct{}{
			"public-channel": {role: "user"},
			"private-channel": {role: "mod"},
		},
	}
}

// Code snippet Fixed: Ensure validator logic matches function existence
// Uppercase V for exported function
func (a *authService) ValidateToken(token string) (bool, string) {
	// Mocking a simple lookup
	return token != "" && token == "valid-token", "user-123"
}

// --- Implementation Methods ---

// SendMessage sends a message to a channel
func (s *messageService) SendMessage(ctx context.Context, channelID, content, authorID string) (string, error) {
	if !s.channelSvc.ValidateChannelID(channelID) {
		return "", fmt.Errorf("service: invalid channel ID format")
	}

	// In a real app, validate content length here
	if content == "" {
		return "", fmt.Errorf("service: content cannot be empty")
	}

	msgID := fmt.Sprintf("msg-%d", time.Now().UnixNano())

	// Simulate a "Wait" on DB write (slow operation test)
	time.Sleep(10 * time.Millisecond)
	
	err := s.storageSvc.SaveMessage(ctx, msgID, channelID, content)
	if err != nil {
		return "", err
	}

	return msgID, nil
}

// GetHistory retrieves messages from a channel
func (s *messageService) GetHistory(ctx context.Context, channelID string) ([]Message, error) {
	if !s.channelSvc.ValidateChannelID(channelID) {
		return nil, fmt.Errorf("service: access denied or invalid channel")
	}

	return s.storageSvc.GetMessages(ctx, channelID)
}

// --- Implementation Methods ---

func (s *messageService) SendMessage(ctx context.Context, channelID, content, authorID string) (string, error) {
	// Validate Channel
	if !s.channelSvc.ValidateChannelID(channelID) {
		return "", fmt.Errorf("service: invalid channel ID format")
	}

	// Validate Message
	if content == "" {
		return "", fmt.Errorf("service: content cannot be empty")
	}

	msgID := time.Now().Format("20060102-15-04-05")

	return s.storageSvc.SaveMessage(ctx, msgID, channelID, content)
}

func (s *messageService) GetHistory(ctx context.Context, channelID string) ([]Message, error) {
	if !s.channelSvc.ValidateChannelID(channelID) {
		return nil, fmt.Errorf("service: service: access denied or invalid channel")
	}

	return s.storageSvc.GetMessages(ctx, channelID)
}

// BuildMiddlewareChain allows functional composition of logic
func BuildMiddlewareChain(svc MessageService, middlewares []Middleware) MessageService {
	current := svc
	for i := len(middlewares) - 1; i >= 0; i-- {
		current = middlewares[i](current)
	}
	return current
}