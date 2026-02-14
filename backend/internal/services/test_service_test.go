package services

import (
	"context"
	"errors"
	"testing"
	"time"
)

// mockStorage implements the Storage interface for testing.
type mockStorage struct {
	users         []*User
	messages      []*Message
	usersMap      map[string]*User // Tracks state for quick lookup
	messagesMap   map[ChannelID][]*Message
	errOnCreate   bool
	errOnGet      bool
	returnMsg     *Message // Custom return value for specific mock behaviors
	debug         bool
}

func (m *mockStorage) CreateUser(ctx context.Context, email, name string) (*User, error) {
	if m.errOnCreate {
		return nil, errors.New("db connection error")
	}
	user := &User{ID: email, Name: name, Email: email}
	m.users = append(m.users, user)
	m.usersMap[email] = user
	if m.debug {
		println("Mock: Created user", email)
	}
	return user, nil
}

func (m *mockStorage) GetUserByID(ctx context.Context, id string) (*User, error) {
	if m.errOnGet {
		return nil, errors.New("db connection error")
	}
	if u, exists := m.usersMap[id]; exists {
		return u, nil
	}
	return nil, errors.New("user not found")
}

func (m *mockStorage) CreateMessage(ctx context.Context, channelID ChannelID, senderID string, content string) (*Message, error) {
	// Simulate successful save
	msg := &Message{
		ID:        senderID + "-" + content,
		Content:   content,
		SenderID:  senderID,
		CreatedAt: time.Now(),
	}
	m.messages = append(m.messages, msg)
	if m.messagesMap[channelID] == nil {
		m.messagesMap[channelID] = make([]*Message, 0)
	}
	m.messagesMap[channelID] = append(m.messagesMap[channelID], msg)
	if m.returnMsg != nil {
		return m.returnMsg, nil // Return specific custom value if set
	}
	return msg, nil
}

func (m *mockStorage) GetMessagesInChannel(ctx context.Context, channelID ChannelID, limit int) ([]*Message, error) {
	if len(m.messagesMap[channelID]) == 0 {
		return []*Message{}, nil
	}
	// Allow limit to slice the mock data
	if limit <= 0 {
		return m.messagesMap[channelID], nil
	}
	return m.messagesMap[channelID], nil
}

// Helper function to create a fresh mock storage for each test
func newMockStorage() *mockStorage {
	return &mockStorage{
		usersMap:    make(map[string]*User),
		messagesMap: make(map[ChannelID][]*Message),
	}
}

func TestCreateUser(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		name     string
		wantErr  bool
		errMsg   string
	}{
		{"Valid User", "test@example.com", "Alice", false, ""},
		{"Empty Name", "test@example.com", "", true, "email and name cannot be empty"},
		{"Empty Email", "", "Alice", true, "email and name cannot be empty"},
		{"Duplicate Email", "test@example.com", "Bob", true, "user with email test@example.com already exists"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockDB := newMockStorage()
			service := NewHearthService(mockDB)

			// Execute
			result, err := service.CreateUser(context.Background(), tt.email, tt.name)

			// Assert
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if err.Error() != tt.errMsg {
					t.Errorf("CreateUser() error message = %v, want %v", err.Error(), tt.errMsg)
				}
				return
			}
			if result == nil {
				t.Error("CreateUser() returned nil user on success")
				return
			}
			if result.Name != tt.name {
				t.Errorf("CreateUser() Name = %v, want %v", result.Name, tt.name)
			}
		})
	}
}

func TestSendMessage(t *testing.T) {
	// Setup with a known user context
	mockDB := newMockStorage()
	service := NewHearthService(mockDB)
	senderEmail := "alice@example.com"
	_, _ = service.CreateUser(context.Background(), senderEmail, "Alice")

	ctx := context.Background()
	testContent := "Hello World"

	t.Run("Success", func(t *testing.T) {
		msg, err := service.CreateMessage(ctx, "general", senderEmail, testContent)
		if err != nil {
			t.Fatalf("CreateMessage() failed with error: %v", err)
		}
		if msg.Content != testContent {
			t.Errorf("Message content mismatch; got %s, want %s", msg.Content, testContent)
		}
		if msg.SenderID != senderEmail {
			t.Errorf("Sender ID mismatch; got %s, want %s", msg.SenderID, senderEmail)
		}
	})

	t.Run("Empty Content", func(t *testing.T) {
		_, err := service.CreateMessage(ctx, "general", senderEmail, "")
		if err == nil {
			t.Error("Expected error for empty content, got nil")
		}
	})
}

func TestGetChatHistory(t *testing.T) {
	mockDB := newMockStorage()
	service := NewHearthService(mockDB)

	// Pre-populate mock data
	userAlice := &User{ID: "alice", Name: "Alice"}
	userBob := &User{ID: "bob", Name: "Bob"}
	mockDB.usersMap["alice"] = userAlice
	mockDB.usersMap["bob"] = userBob

	mockDB.messagesMap["general"] = []*Message{
		{ID: "1", Content: "Hi Alice", SenderID: "bob", CreatedAt: time.Now()},
		{ID: "2", Content: "Hey Bob", SenderID: "alice", CreatedAt: time.Now()},
	}

	ctx := context.Background()
	channel := ChannelID("general")

	t.Run("Fetch Messages", func(t *testing.T) {
		msgs, err := service.GetChatHistory(ctx, channel, 10)
		if err != nil {
			t.Fatalf("GetChatHistory() error = %v", err)
		}

		if len(msgs) != 2 {
			t.Errorf("Expected 2 messages, got %d", len(msgs))
		}
	})

	t.Run("No Limit (Default)", func(t *testing.T) {
		// Ensure slice order is correct
		mockDB.messagesMap["general"] = []*Message{
			{ID: "1", Content: "Msg 1"},
			{ID: "2", Content: "Msg 2"},
		}
		msgs, _ := service.GetChatHistory(ctx, channel, 0)
		if len(msgs) != 2 {
			t.Errorf("Expected 2 messages (default limit handling), got %d", len(msgs))
		}
	})
}

func TestErrorHandling_DatabaseFailure(t *testing.T) {
	// Test what happens when the storage layer fails
	mockDB := newMockStorage()
	mockDB.errOnCreate = true // Simulate DB crash
	service := NewHearthService(mockDB)

	user, err := service.CreateUser(context.Background(), "fail@test.com", "FailUser")

	if err == nil {
		t.Error("Expected error on DB failure, got nil")
	}
	if user != nil {
		t.Error("Expected nil user on DB failure")
	}
}