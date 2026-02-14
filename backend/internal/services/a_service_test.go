package services

import (
	"context"
	"strings"
	"testing"
)

// MockRepository implements MessageRepository for testing.
type MockRepository struct {
	StoreCallCount int
	StoreError     error
	Messages       map[string]*Message
}

func (m *MockRepository) StoreMessage(ctx context.Context, msg *Message) error {
	m.StoreCallCount++
	if m.StoreError != nil {
		return m.StoreError
	}
	m.Messages[msg.ID] = msg
	return nil
}

func (m *MockRepository) GetMessages(ctx context.Context) (map[string]*Message, error) {
	if m.Messages == nil {
		m.Messages = make(map[string]*Message)
	}
	return m.Messages, nil
}

// MockLogger implements ConsoleWriter for testing.
type MockLogger struct {
	LogCalls []string // Stores the exact logs generated
}

func (m *MockLogger) WriteLine(format string, v ...interface{}) error {
	// Simulate logging by storing the formatted string
	msg := fmt.Sprintf(format, v...)
	m.LogCalls = append(m.LogCalls, msg)
	return nil
}

// TestGarrison_CreateMessage tests the happy path of creating a message.
func TestGarrison_CreateMessage(t *testing.T) {
	ctx := context.Background()
	
	// 1. Setup Dependencies
	mockRepo := &MockRepository{Messages: make(map[string]*Message)}
	mockLogger := &MockLogger{LogCalls: []string{}}
	service := NewGarrison(mockRepo, mockLogger)

	// 2. Execute
	user := "TestUser"
	content := "Hello World"
	id, err := service.CreateMessage(ctx, user, content)

	// 3. Assertions
	if err != nil {
		t.Fatalf("CreateMessage failed unexpectedly: %v", err)
	}
	if id == "" {
		t.Error("Expected a non-empty message ID")
	}

	// Verify Repository interaction
	if mockRepo.StoreCallCount != 1 {
		t.Errorf("Expected Repo.StoreMessage to be called once, was called %d times", mockRepo.StoreCallCount)
	}

	// Verify Logger interaction (Side effect check)
	assertTrue := func(t *testing.T, condition bool, message string) {
		if !condition {
			t.Errorf(message)
		}
	}

	assertTrue(t, strings.Contains(mockLogger.LogCalls[0], "Created message id="+id), 
		"Log should contain the created message ID and User")
	assertTrue(t, strings.Contains(mockLogger.LogCalls[0], user), "Log should contain the user name")
}

// TestGarrison_CreateMessage_InvalidInput tests input validation logic.
func TestGarrison_CreateMessage_InvalidInput(t *testing.T) {
	ctx := context.Background()
	service := NewGarrison(&MockRepository{}, &MockLogger{})

	tests := []struct {
		name    string
		user    string
		content string
		wantErr bool
	}{
		{"Empty Content", "User", "", true},
		{"Empty User", "", "Hello", true},
		{"Both Empty", "", "", true},
		{"Valid", "ValidUser", "Valid Content", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.CreateMessage(ctx, tt.user, tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestGarrison_List tests retrieval logic.
func TestGarrison_List(t *testing.T) {
	ctx := context.Background()
	
	// Setup: Pre-populate some mock messages
	mockRepo := &MockRepository{
		Messages: map[string]*Message{
			"msg-1": {ID: "msg-1", User: "Alice", Content: "Hi"},
			"msg-2": {ID: "msg-2", User: "Bob", Content: "Bye"},
		},
	}
	service := NewGarrison(mockRepo, &MockLogger{})

	output, err := service.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	// Assertions on output format
	if !strings.Contains(output, "Alice") {
		t.Error("Output should contain Alice")
	}
	if !strings.Contains(output, "Bob") {
		t.Error("Output should contain Bob")
	}
}