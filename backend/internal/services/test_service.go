package services

import (
	"context"
	"errors"
	"time"
)

// MessageService defines the interface for the heart of the application.
type MessageService interface {
	// CreateMessage handles the business logic of sending a message to a channel.
	// It returns the created message or an error.
	CreateMessage(ctx context.Context, channelID string, content string) (*Message, error)
	
	// GetHistory retrieves previous messages for the specified ID.
	GetHistory(ctx context.Context, channelID string, limit int) ([]*Message, error)
}

// Message represents the data structure of a Discord-like message.
type Message struct {
	ID        string
	Content   string
	Author    string
	ChannelID string
	Timestamp time.Time
}

// HeartService is the concrete implementation of MessageService.
type HeartService struct {
	// In a real app, this would interact with DB repositories (sqlx, sqlc, etc.)
	// Here we mock that behavior using in-memory structures.
	client *MockDiscordClient
}

// NewHeartService creates a new instance of HeartService.
func NewHeartService(client *MockDiscordClient) *HeartService {
	return &HeartService{
		client: client,
	}
}

// CreateMessage stores the message and returns it.
func (s *HeartService) CreateMessage(ctx context.Context, channelID string, content string) (*Message, error) {
	if content == "" {
		return nil, errors.New("message content cannot be empty")
	}

	msg := &Message{
		ID:        generateID(),
		Content:   content,
		Author:    ctx.Value("user_id").(string), // In prod, this comes from middleware/gRPC
		ChannelID: channelID,
		Timestamp: time.Now(),
	}

	// Simulate writing to the "database"
	s.client.storeMsg(channelID, msg)

	return msg, nil
}

// GetHistory retrieves messages from the in-memory store.
func (s *HeartService) GetHistory(ctx context.Context, channelID string, limit int) ([]*Message, error) {
	return s.client.fetchHistory(channelID, limit), nil
}

// generateID is a helper to simulate UUID generation.
func generateID() string {
	return "msg-" + time.Now().Format("20060102-150405")
}

// MockDiscordClient is a dependency for testing. 
// It simulates the persistence layer.
type MockDiscordClient struct {
	messages map[string]map[string]*Message // map[channelID]map[msgID]Message
}

func NewMockDiscordClient() *MockDiscordClient {
	return &MockDiscordClient{
		messages: make(map[string]map[string]*Message),
	}
}

// storeMsg adds a message for specific channelID
func (m *MockDiscordClient) storeMsg(channelID string, msg *Message) {
	if _, exists := m.messages[channelID]; !exists {
		m.messages[channelID] = make(map[string]*Message)
	}
	m.messages[channelID][msg.ID] = msg
}

// fetchHistory retrieves messages for a channelID
func (m *MockDiscordClient) fetchHistory(channelID string, limit int) []*Message {
	msgs, ok := m.messages[channelID]
	if !ok {
		return []*Message{}
	}

	// Convert map to slice
	var result []*Message
	count := 0
	for _, msg := range msgs {
		result = append(result, msg)
		count++
		if limit > 0 && count >= limit {
			break
		}
	}
	return result
}