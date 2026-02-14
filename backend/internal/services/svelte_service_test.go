package services

import (
	"testing"
	"time"
	"reflect"
)

// MockChannelService implements ChannelService but uses an in-memory channel.
// This allows us to drive messages programmatically for testing.
type MockChannelService struct {
	connected chan Message
	closed    bool
}

func (m *MockChannelService) Connect(username string) (chan Message, error) {
	if m.closed {
		return nil, ErrServiceUnavailable
	}
	m.connected = make(chan Message, 100)
	return m.connected, nil
}

func (m *MockChannelService) Close() error {
	m.closed = true
	return nil
}

func TestNewChannelService(t *testing.T) {
	host := "ws://hearth-backend.internal:8080"
	username := "test_user"

	// Act
	svc := NewChannelService(host, username)

	// Assert
	if svc == nil {
		t.Fatal("Expected service instance, got nil")
	}

	defaultSvc, ok := svc.(*DefaultChannelService)
	if !ok {
		t.Fatalf("Expected DefaultChannelService, got %T", svc)
	}

	if defaultSvc.host != host {
		t.Errorf("Host mismatch. Expected %s, got %s", host, defaultSvc.host)
	}

	if defaultSvc.username != username {
		t.Errorf("Username mismatch. Expected %s, got %s", username, defaultSvc.username)
	}
}

func TestServiceLifecycle(t *testing.T) {
	mock := &MockChannelService{}
	
	// Simulate successful connection
	chnl, err := mock.Connect("tester")
	if err != nil {
		t.Fatalf("Failed to connect mock: %v", err)
	}

	// Verify channel is ready
	if chnl == nil {
		t.Fatal("Expected non-nil channel on connect")
	}

	// Test sending a message down the wire
	expectedMsg := Message{
		ID:        "msg_123",
		Type:      MsgSendMessage,
		Content:   "Hello World",
		Timestamp: time.Now().Unix(),
		Username:  "tester",
	}

	// Write to the service
	chnl <- expectedMsg

	// Read from the service
	receivedMsg := <-chnl

	// Assert data integrity
	if receivedMsg.ID != expectedMsg.ID {
		t.Errorf("ID mismatch. Got %s", receivedMsg.ID)
	}
	
	if receivedMsg.Type != expectedMsg.Type {
		t.Errorf("Type mismatch. Got %s", receivedMsg.Type)
	}
}

func TestMockConnectionFailure(t *testing.T) {
	mock := &MockChannelService{closed: true}

	_, err := mock.Connect("blocked_user")
	if err == nil {
		t.Error("Expected ErrServiceUnavailable when connection is closed")
	}

	// Verify error type
	svcErr, ok := err.(*ServiceError)
	if !ok {
		t.Fatalf("Expected ServiceError type, got %T", err)
	}

	if svcErr.Code != "503" {
		t.Errorf("Expected error code 503, got %s", svcErr.Code)
	}
}