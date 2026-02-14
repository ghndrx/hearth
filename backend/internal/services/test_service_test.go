package services

import (
	"context"
	"testing"
	"time"
)

func TestHeartService_CreateMessage(t *testing.T) {
	// 1. Setup: Arrange
	mockClient := NewMockDiscordClient()
	service := NewHeartService(mockClient)
	
	// Create a context with a mock user ID (dependency injection)
	ctx := context.WithValue(context.Background(), "user_id", "user_12345")
	
	validContent := "Hello Hearth!"
	
	// 2. Execution: Act
	result, err := service.CreateMessage(ctx, "channel_general", validContent)
	
	// 3. Validation: Assert
	if err != nil {
		t.Fatalf("CreateMessage failed: %v", err)
	}
	
	if result == nil {
		t.Fatal("Expected a message to be returned, got nil")
	}
	
	if result.Content != validContent {
		t.Errorf("Expected content '%s', got '%s'", validContent, result.Content)
	}
	
	if result.Author != "user_12345" {
		t.Errorf("Expected author 'user_12345', got '%s'", result.Author)
	}
	
	// Verify it was actually stored in the mock client
	storedMsgs := mockClient.fetchHistory("channel_general", 1)
	if len(storedMsgs) != 1 {
		t.Fatalf("Expected 1 message in client store, found %d", len(storedMsgs))
	}
}

func TestHeartService_CreateMessage_EmptyContent(t *testing.T) {
	// Arrange
	mockClient := NewMockDiscordClient()
	service := NewHeartService(mockClient)
	ctx := context.Background()

	// Act & Assert
	_, err := service.CreateMessage(ctx, "channel_general", "")
	
	if err == nil {
		t.Error("Expected an error when sending empty content, got nil")
	}
}

func TestHeartService_GetHistory(t *testing.T) {
	// 1. Setup: Arrange
	mockClient := NewMockDiscordClient()
	service := NewHeartService(mockClient)
	
	// Pre-populate some history
	msg1 := &Message{ID: "msg-1", Content: "First post"}
	msg2 := &Message{ID: "msg-2", Content: "Second post"}
	
	mockClient.storeMsg("channel_1", msg1)
	mockClient.storeMsg("channel_1", msg2)
	
	ctx := context.Background()
	
	// 2. Execution: Act
	history, err := service.GetHistory(ctx, "channel_1", 5)
	
	// 3. Validation: Assert
	if err != nil {
		t.Fatalf("GetHistory failed: %v", err)
	}
	
	if len(history) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(history))
	}
	
	// Verify order (simulated by how we append to slice, latest first)
	if history[0].Content != "Second post" {
		t.Error("Expected latest message to be first")
	}
}

func TestHeartService_GetHistory_NoMessages(t *testing.T) {
	// Arrange
	mockClient := NewMockDiscordClient()
	service := NewHeartService(mockClient)
	ctx := context.Background()

	// Act
	history, err := service.GetHistory(ctx, "empty_channel", 10)

	// Assert
	if err != nil {
		t.Fatalf("GetHistory failed: %v", err)
	}
	if len(history) != 0 {
		t.Errorf("Expected empty slice, got %d messages", len(history))
	}
}

// Benchmarks are often included in service layers to test concurrency/speed
func BenchmarkHeartService_CreateMessage(b *testing.B) {
	mockClient := NewMockDiscordClient()
	service := NewHeartService(mockClient)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.CreateMessage(ctx, "b_ch", "Benchmark Data")
	}
}