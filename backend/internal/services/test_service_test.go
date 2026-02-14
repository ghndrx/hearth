package services

import (
	"context"
	"testing"
)

// mockMessageStore implements the MessageStore interface for testing.
type mockMessageStore struct {
	messages []UserMessage
	saveErr  error
}

func (m *mockMessageStore) SaveMessage(ctx context.Context, msg UserMessage) (int64, error) {
	if m.saveErr != nil {
		return 0, m.saveErr
	}
	m.messages = append(m.messages, msg)
	// Return a pseudo-random ID or just the current length of the slice + 1
	return int64(len(m.messages) + 1), nil
}

func (m *mockMessageStore) GetMessagesByChannel(ctx context.Context, channelID string) ([]UserMessage, error) {
	if m.saveErr != nil {
		return nil, m.saveErr
	}
	// Return errors only if channelID contains "error"
	if len(channelID) > 6 && channelID[:6] == "error" {
		return nil, errors.New("simulated database failure")
	}
	
	var filtered []UserMessage
	for _, msg := range m.messages {
		if msg.ChannelID == channelID {
			filtered = append(filtered, msg)
		}
	}
	return filtered, nil
}

func TestSendMessage_Success(t *testing.T) {
	tests := []struct {
		name          string
		channelID     string
		author        string
		content       string
		setupStore    func() MessageStore
		expectedID    int64
		expectError   bool
		expectedError string
	}{
		{
			name:        "Valid input",
			channelID:   "ch-123",
			author:      "Alice",
			content:     "Hello World",
			setupStore:  func() MessageStore { return &mockMessageStore{} },
			expectedID:  1, // mock creates ID 1
			expectError: false,
		},
		{
			name:        "Empty content",
			channelID:   "ch-123",
			author:      "Bob",
			content:     "",
			setupStore:  func() MessageStore { return &mockMessageStore{} },
			expectedID:  -1,
			expectError: true,
		},
		{
			name:        "Empty author",
			channelID:   "ch-123",
			author:      "",
			content:     "Bob's message",
			setupStore:  func() MessageStore { return &mockMessageStore{} },
			expectedID:  -1,
			expectError: true,
		},
		{
			name:        "Store failure",
			channelID:   "ch-123",
			author:      "Charlie",
			content:     "Crash test",
			setupStore:  func() MessageStore { 
				m := &mockMessageStore{saveErr: errors.New("hard drive full")}
				return m
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := tt.setupStore()
			service := NewChatService(store)

			id, err := service.SendMessage(context.Background(), tt.channelID, tt.author, tt.content)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected an error but got none")
				}
				if tt.expectedError != "" && err.Error() != tt.expectedError {
					t.Errorf("Expected error '%s', got '%s'", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Did not expect an error, but got: %v", err)
				}
				if id != tt.expectedID {
					t.Errorf("Expected ID %d, got %d", tt.expectedID, id)
				}
			}
		})
	}
}

func TestGetChannelMessages(t *testing.T) {
	// Setup initial data in mock store
	mock := &mockMessageStore{
		messages: []UserMessage{
			{ChannelID: "ch-1", Content: "Hi"},
			{ChannelID: "ch-2", Content: "Bye"},
			{ChannelID: "ch-1", Content: "Hey"},
		},
	}
	store := mock
	service := NewChatService(store)

	ctx := context.Background()

	// Test 1: Getting existing channel
	messages, err := service.GetChannelMessages(ctx, "ch-1")
	if err != nil {
		t.Fatalf("Failed to get messages: %v", err)
	}
	if len(messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(messages))
	}
	// Verify the content matches the order of insertion
	if messages[0].Content != "Hi" {
		t.Errorf("Expected first message 'Hi', got '%s'", messages[0].Content)
	}

	// Test 2: Getting non-existing channel
	messages, err = service.GetChannelMessages(ctx, "ch-999")
	if err != nil {
		t.Fatalf("Unexpected error for non-existent channel: %v", err)
	}
	if len(messages) != 0 {
		t.Errorf("Expected an empty slice, got %d messages", len(messages))
	}

	// Test 3: Invalid ID handling (Pseudo test of error path)
	_, err = service.GetChannelMessages(ctx, "error-channel")
	if err == nil {
		t.Error("Expected error for 'error-channel', got nil")
	}
}