package services

import (
	"errors"
	"sync"
	"testing"
	"time"
)

// MockStorage implements the Storage interface using an internal map
// This makes it entirely deterministic and fast for tests.
type MockStorage struct {
	Messages map[string][]*Message
	Users    map[string]*User
	mu       sync.RWMutex
	errToReturn error
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		Messages: make(map[string][]*Message),
		Users:    make(map[string]*User),
	}
}

// Implementing Storage interface methods

func (m *MockStorage) SaveMessage(msg *Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Simulate a database error for specific conditions (optional)
	if m.errToReturn != nil {
		return m.errToReturn
	}

	m.Messages[msg.ChannelID] = append(m.Messages[msg.ChannelID], msg)
	return nil
}

func (m *MockStorage) GetUser(userID string) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	user, exists := m.Users[userID]
	if !exists {
		return nil, errors.New("user not found")
	}
	return user, nil
}

func (m *MockStorage) GetMessagesByChannel(channelID string) ([]*Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return m.Messages[channelID], nil
}

func (m *MockStorage) CreateUser(user *User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if existing, exists := m.Users[user.ID]; exists {
		return errors.New("user already exists")
	}
	
	m.Users[user.ID] = user
	return nil
}

// TestHappyPath tests standard functionality
func TestHappyPath(t *testing.T) {
	store := NewMockStorage()
	service := NewMainService(store)

	t.Run("User creation follows pattern", func(t *testing.T) {
		err := service.CreateUser("alice", "alice@example.com", "u1")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
	})

	t.Run("Sending a message to a channel", func(t *testing.T) {
		channelID := "channel-1"
		content := "Hello World"
		userID := "u1"

		msg, err := service.SendMessage(channelID, content, userID)
		if err != nil {
			t.Fatalf("Expected success, got: %v", err)
		}

		if msg.Content != content {
			t.Errorf("Expected content '%s', got '%s'", content, msg.Content)
		}
		if msg.ChannelID != channelID {
			t.Errorf("Expected channel ID '%s', got '%s'", channelID, msg.ChannelID)
		}
		if msg.UserID != userID {
			t.Errorf("Expected user ID '%s', got '%s'", userID, msg.UserID)
		}

		// Verify message is in storage
		history, _ := service.GetChannelHistory(channelID, 0)
		if len(history) != 1 {
			t.Fatalf("Expected 1 message in storage, got %d", len(history))
		}
	})
}

// TestErrorConditions ensures bad inputs are handled
func TestErrorConditions(t *testing.T) {
	store := NewMockStorage(service := NewMainService(store)

	t.Run("SendMessage fails with empty user", func(t *testing.T) {
		_, err := service.SendMessage("c1", "content", "")
		if err == nil {
			t.Error("Expected an error for empty userID, got nil")
		}
	})

	t.Run("SendMessage fails with empty channel", func(t *testing.T) {
		_, err := service.SendMessage("", "content", "u1")
		if err == nil {
			t.Error("Expected an error for empty channelID, got nil")
		}
	})

	t.Run("SendMessage fails with empty content", func(t *testing.T) {
		_, err := service.SendMessage("c1", "", "u1")
		if err == nil {
			t.Error("Expected an error for empty content, got nil")
		}
	})

	t.Run("SendMessage fails with unknown user", func(t *testing.T) {
		_, err := service.SendMessage("c1", "hello", "notfound-user")
		if err == nil {
			t.Error("Expected an error for unknown user, got nil")
		}
		if err.Error() != "user notfound-user not found" {
			t.Errorf("Unexpected error message: %v", err)
		}
	})

	t.Run("CreateUser validates inputs", func(t *testing.T) {
		err := service.CreateUser("", "email", "id")
		if err == nil {
			t.Error("Expected error for empty username, got nil")
		}
	})
}

// TestGetChannelHistory tests pagination and history fetching
func TestGetChannelHistory(t *testing.T) {
	store := NewMockStorage()
	service := NewMainService(store)

	// Pre-seed data
 msgs := []*Message{
		{ID: "1", ChannelID: "c1", Content: "msg1", UserID: "u1", Timestamp: time.Now().Add(-10 * time.Second)},
		{ID: "2", ChannelID: "c1", Content: "msg2", UserID: "u2", Timestamp: time.Now().Add(-5 * time.Second)},
	}
	for _, m := range msgs {
		store.SaveMessage(m)
	}

	t.Run("Returns correct limit gently", func(t *testing.T) {
		history, err := service.GetChannelHistory("c1", 2)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(history) != 2 {
			t.Fatalf("Expected 2 messages, got %d", len(history))
		}
	})

	t.Run("Empty limit should fail", func(t *testing.T) {
		_, err := service.GetChannelHistory("c1", 0)
		if err == nil {
			t.Error("Expected error for limit 0, got nil")
		}
	})
}

// TestConcurrency tests the RWMutex implementation is working by
// hammering the service with multiple goroutines.
func TestConcurrency(t *testing.T) {
	store := NewMockStorage()
	service := NewMainService(store)
	
	// Pre-create a valid user
	service.CreateUser("alice", "a@e.com", "u1")
	
	wg := sync.WaitGroup{}
	numGoroutines := 100
	messagesPerGoroutine := 10
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(offset int) {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				_, _ = service.SendMessage("c1", fmt.Sprintf("msg-%d-%d", offset, j), "u1")
			}
		}(i)
	}
	
	wg.Wait()
	
	// Assert all messages were received
	history, err := service.GetChannelHistory("c1", 10000)
	if err != nil {
		t.Fatalf("Error retrieving history after concurrency: %v", err)
	}
	
	expectedCount := numGoroutines * messagesPerGoroutine
	if len(history) != expectedCount {
		t.Errorf("Expected %d messages after concurrency, got %d", expectedCount, len(history))
	}
}