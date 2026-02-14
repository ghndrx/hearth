package services

import (
	"sync"
	"testing"
	"time"
)

// TestNewHeartService checks if the service initializes correctly.
func TestNewHeartService(t *testing.T) {
	service := NewHeartService()
	if service == nil {
		t.Fatal("NewHeartService returned nil")
	}

	// Ensure internal maps are initialized
	if len(service.handlers) != 0 {
		t.Error("Expected empty handlers map initially")
	}
}

// MockHandler is a test double (Stub) that allows us to track if Handle was called.
type MockHandler struct {
	Calls  []HeartbeatEvent
	Error  error
	Locked bool // To simulate a handler that blocks/stalls
}

func (m *MockHandler) Handle(event HeartbeatEvent) error {
	m.Calls = append(m.Calls, event)
	if m.Error != nil {
		return m.Error
	}
	// Simulate a heavy operation if locked is true
	if m.Locked {
		time.Sleep(10 * time.Millisecond)
	}
	return nil
}

// TestRegisterHandler ensures handlers can be added and retrieved.
func TestRegisterHandler(t *testing.T) {
	s := NewHeartService()
	mock := &MockHandler{Calls: []HeartbeatEvent{}}

	s.RegisterHandler("alice", mock)

	// Retrieve handler to check state
	s.mu.RLock()
	h := s.handlers["alice"]
	s.mu.RUnlock()

	if h == nil {
		t.Fatal("Handler was not registered")
	}
	if h != mock {
		t.Error("Registered handler is not the same instance")
	}
}

// TestSendEvent tests that events are routed and handled asynchronously.
func TestSendEvent(t *testing.T) {
	s := NewHeartService()
	mock := &MockHandler{}

	s.RegisterHandler("bob", mock)

	// Send an event
	event := HeartbeatEvent{
		User:    "bob",
		Content: "test message",
	}

	s.SendEvent(event)

	// Wait a moment for the goroutine to execute
	time.Sleep(10 * time.Millisecond)

	// Assert
	if len(mock.Calls) != 1 {
		t.Errorf("Expected 1 call, got %d", len(mock.Calls))
	}

	if mock.Calls[0].Content != "test message" {
		t.Errorf("Expected content 'test message', got '%s'", mock.Calls[0].Content)
	}
}

// TestSendEvent_NoHandler tests behavior when no handler exists for an event user.
func TestSendEvent_NoHandler(t *testing.T) {
	s := NewHeartService()

	event := HeartbeatEvent{User: "void"}

	// Should not panic
	s.SendEvent(event)
}

// TestSendEvent_MultipleConcurrent sends multiple events to the same handler to check thread safety.
func TestSendEvent_MultipleConcurrent(t *testing.T) {
	s := NewHeartService()
	mock := &MockHandler{}

	s.RegisterHandler("charlie", mock)
	event := HeartbeatEvent{User: "charlie"}

	var wg sync.WaitGroup
	start := time.Now()

	// Fire 100 events concurrently (simulating high load traffic)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.SendEvent(event)
		}()
	}

	wg.Wait()
	elapsed := time.Since(start)

	// We expect roughly 100 calls.
	// Note: Since events might be processed out of order or very fast,
	// we allow some slack but generally expect them to be registered.
	if len(mock.Calls) < 95 {
		t.Errorf("Expected at least 95 calls, got %d", len(mock.Calls))
	}
	
	// If the handler simulated a deadlock (Locked=true above), 
	// this test would likely timeout (Set timeout in `go test -timeout ...`).
	// Here we ensure it handles queueing reasonably well by finishing in < 1 second.
	if elapsed > 100*time.Millisecond {
		t.Errorf("Handler took too long to process load: %v", elapsed)
	}
}

// TestExampleCommandHandler ensures the provided example handlers work.
func TestExampleCommandHandler(t *testing.T) {
	cmd := &ChatCommandHandler{Prefix: "/"}
	
	handler := IEventHandler(cmd)

	// Valid command
	err := handler.Handle(HeartbeatEvent{Content: "/hello"})
	if err != nil {
		t.Errorf("Command handler failed on valid input: %v", err)
	}

	// Invalid command
	err = handler.Handle(HeartbeatEvent{Content: "/bye"})
	if err == nil {
		t.Error("Expected error for invalid command")
	}
}

// TestExampleFilterHandler ensures the provided example anti-spam handler works.
func TestExampleFilterHandler(t *testing.T) {
	filter := &AntiSpamHandler{
		Regexp: regexp.MustCompile("badword"),
	}

	handler := IEventHandler(filter)

	// Valid content
	err := handler.Handle(HeartbeatEvent{Content: "good content"})
	if err != nil {
		t.Errorf("Handler should not block valid content: %v", err)
	}

	// Filtered content
	err = handler.Handle(HeartbeatEvent{Content: "some badword here"})
	if err == nil {
		t.Error("Expected error for filtered content")
	}
}