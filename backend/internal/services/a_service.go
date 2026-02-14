package services

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrChannelNotFound is returned when a channel ID does not exist.
var ErrChannelNotFound = errors.New("channel not found")

// ErrUserNotFound is returned when trying to send a message to a non-existent user channel.
var ErrUserNotFound = errors.New("user not found")

// Message represents a single text message in the chat.
type Message struct {
	ID        string
	Content   string
	Timestamp time.Time
	SenderID  string
	// ChannelID is optional for direct messages, required for guild channels
	ChannelID *string 
}

// ChatService is the core business logic for the chat.
// It satisfies the Service interface.
type ChatService struct {
	mux sync.RWMutex

	// Database abstraction: In-memory map for demonstration.
	channels map[string]*ServiceChannel
	messages map[string][]Message 
}

// ServiceChannel holds the state for a single channel (e.g., a Discord guild text channel).
// This allows for auto-generation of IDs in a real scenario.
type ServiceChannel struct {
	ID          string
	Name        string
	Type        string // e.g., "TEXT", "VOICE"
	Description string
}

// ChannelService defines the public contract for creating and retrieving channels.
type ChannelService interface {
	GetChannel(ctx context.Context, id string) (*ServiceChannel, error)
	CreateChannel(ctx context.Context, name string, channelType string) (string, error)
}

// MessageService defines the public contract for sending and retrieving messages.
type MessageService interface {
	SendMessage(ctx context.Context, msg *Message) error
	GetMessages(ctx context.Context, channelID string) ([]Message, error)
	DeleteMessage(ctx context.Context, msgID string) error
}

// ChatService implements ChannelService and MessageService.
func NewChatService() *ChatService {
	return &ChatService{
		channels: make(map[string]*ServiceChannel),
		messages: make(map[string][]Message),
	}
}

// --- Channel Operations ---

// CreateChannel initializes a new service channel.
func (s *ChatService) CreateChannel(ctx context.Context, name string, channelType string) (string, error) {
	s.mux.Lock()
	defer s.mux.Unlock()

	chID := "ch_" + name + "_" + time.Now().Format("micro")
	
	s.channels[chID] = &ServiceChannel{
		ID:          chID,
		Name:        name,
		Type:        channelType,
		Description: "Automatically created via service layer",
	}

	return chID, nil
}

// GetChannel retrieves channel details by ID.
func (s *ChatService) GetChannel(ctx context.Context, id string) (*ServiceChannel, error) {
	s.mux.RLock()
	defer s.mux.RUnlock()
	
	channel, exists := s.channels[id]
	if !exists {
		return nil, ErrChannelNotFound
	}
	return channel, nil
}

// --- Message Operations ---

// SendMessage records a new message in the memory store.
func (s *ChatService) SendMessage(ctx context.Context, msg *Message) error {
	if msg.Content == "" {
		return errors.New("message content cannot be empty")
	}
	if msg.SenderID == "" {
		return errors.New("sender ID cannot be empty")
	}

	// Optional: Validate channel exists if ChannelID is provided.
	if msg.ChannelID != nil {
		s.mux.RLock()
		_, exists := s.channels[*msg.ChannelID]
		s.mux.RUnlock()
		if !exists {
			return ErrChannelNotFound
		}
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	msg.ID = "msg_" + time.Now().Format("nano")
	msg.Timestamp = time.Now()

	s.messages[msg.SenderID] = append(s.messages[msg.SenderID], *msg)
	if msg.ChannelID != nil {
		s.messages[*msg.ChannelID] = append(s.messages[*msg.ChannelID], *msg)
	}

	return nil
}

// GetMessages retrieves the message history for a specific source.
// A "source" acts as the key. This could be a Channel ID or a User ID (Private Message).
func (s *ChatService) GetMessages(ctx context.Context, sourceID string) ([]Message, error) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	msgs, exists := s.messages[sourceID]
	if !exists {
		// Returning an empty list is often preferred over an error for "history of non-existent channel"
		return []Message{}, nil
	}
	
	// Return a copy to prevent external modification of internal state
	customCopy := make([]Message, len(msgs))
	copy(customCopy, msgs)
	return customCopy, nil
}

// DeleteMessage removes a message from history.
func (s *ChatService) DeleteMessage(ctx context.Context, msgID string) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	for key, msgs := range s.messages {
		for i, m := range msgs {
			if m.ID == msgID {
				// Remove element
				s.messages[key] = append(msgs[:i], msgs[i+1:]...)
				return nil
			}
		}
	}
	return errors.New("message not found")
}