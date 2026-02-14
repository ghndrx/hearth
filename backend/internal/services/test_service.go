package services

import (
	"context"
	"errors"
	"time"
)

// MessageStore defines the necessary operations to store/retrieve messages.
// We use an interface to allow passing a mock implementation during tests.
type MessageStore interface {
	SaveMessage(context.Context, UserMessage) (int64, error)
	GetMessagesByChannel(ctx context.Context, channelID string) ([]UserMessage, error)
}

// UserMessage represents a text message in the service.
type UserMessage struct {
	ID        int64     `json:"id"`
	ChannelID string    `json:"channel_id"`
	Content   string    `json:"content"`
	Author    string    `json:"author"`
	Timestamp time.Time `json:"created_at"`
}

// ChatService handles interactions related to the chat module.
type ChatService struct {
	store MessageStore
}

// NewChatService creates a new chat service with the provided storage implementation.
func NewChatService(store MessageStore) *ChatService {
	return &ChatService{
		store: store,
	}
}

// SendMessage creates a new message and stores it.
func (s *ChatService) SendMessage(ctx context.Context, channelID, author, content string) (int64, error) {
	if content == "" {
		return -1, errors.New("message content cannot be empty")
	}
	if author == "" {
		return -1, errors.New("author cannot be empty")
	}

	msg := UserMessage{
		ChannelID: channelID,
		Author:    author,
		Content:   content,
		Timestamp: time.Now(),
	}

	id, err := s.store.SaveMessage(ctx, msg)
	if err != nil {
		// In a real app, you might wrap this or log it.
		return -1, errors.New("failed to persist message")
	}

	return id, nil
}

// GetChannelMessages retrieves all messages for a specific channel.
func (s *ChatService) GetChannelMessages(ctx context.Context, channelID string) ([]UserMessage, error) {
	if channelID == "" {
		return nil, errors.New("channel ID cannot be empty")
	}

	messages, err := s.store.GetMessagesByChannel(ctx, channelID)
	if err != nil {
		return nil, errors.New("failed to retrieve messages")
	}

	return messages, nil
}

// ServiceResult wraps the standard "ServiceResponse" pattern result.
type ServiceResult[T any] struct {
	Data  T     `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

type ChatResponse struct {
	Success bool     `json:"success"`
	Result  ServiceResult[[]UserMessage] `json:"result"`
}