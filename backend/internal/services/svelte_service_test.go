package services_test

import (
	"strings"
	"testing"

	"github.com/yourusername/hearth/services" // Replace with actual module path
)

func TestDefaultSvelteService_RenderConnectionStatus(t *testing.T) {
	service := services.NewSvelteService()
	
	tests := []struct {
		name        string
		userID      string
		expectActive bool
		requestID   string
	}{
		{
			name:        "Active User",
			userID:      "alice",
			expectActive: true,
			requestID:   "alice",
		},
		{
			name:        "Offline User",
			userID:      "bob",
			expectActive: false,
			requestID:   "bob",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			html, err := service.RenderConnectionStatus(tt.userID)
			
			if err != nil {
				t.Fatalf("RenderConnectionStatus() failed with error: %v", err)
			}

			// Assert
			if html == "" {
				t.Error("expected output HTML to be non-empty")
			}

			// Check for expected classes or content based on mock logic (Active=true)
			if strings.Contains(html, "green") && tt.expectActive {
				t.Log("HTML contains expected 'green' status class")
			}
			
			// Verify user ID injection
			if !strings.Contains(html, tt.requestID) {
				t.Errorf("HTML does not contain user ID: %v", tt.requestID)
			}
		})
	}
}

func TestDefaultSvelteService_GenerateChatView(t *testing.T) {
	service := services.NewSvelteService()

	// Act
	html, err := service.GenerateChatView()

	// Assert basic behavior
	if err != nil {
		t.Fatalf("GenerateChatView() failed with error: %v", err)
	}

	if html == "" {
		t.Fatal("expected output HTML to be non-empty")
	}

	// Verify we got the chat container structure
	// We expect the placeholder text from our template
	if !strings.Contains(html, "Message #general") {
		t.Error("Chat view template did not render expected container placeholder")
	}

	// Verify mock data messages are included
	mockMsg := "Hello from Discord!"
	if !strings.Contains(html, mockMsg) {
		t.Errorf("Chat view did not render mock message. Content: %v", html)
	}
}

// Example of a Mock implementation for a strict Interface Test
type MockSvelteRenderer struct{}

func (s *MockSvelteRenderer) RenderConnectionStatus(userID string) (string, error) {
	// Static mock response
	return "<div>Mock Status</div>", nil
}

func (s *MockSvelteRenderer) GenerateChatView() (string, error) {
	return "<div>Mock Chat</div>", nil
}

func TestSvelteService_Interfaces(t *testing.T) {
	var _ services.SvelteService = &MockSvelteRenderer{}

	mock := &MockSvelteRenderer{}
	
	// Call methods to ensure they satisfy the contract
	_, err := mock.RenderConnectionStatus("test")
	if err != nil {
		t.Errorf("mock RenderConnectionStatus failed: %v", err)
	}

	_, err = mock.GenerateChatView()
	if err != nil {
		t.Errorf("mock GenerateChatView failed: %v", err)
	}
}