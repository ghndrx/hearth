package services

import (
	"testing"
)

// MockWSClient is a concrete implementation of WSClient for testing.
type MockWSClient struct {
	Token          Token
	SentMessages   []interface{}
	SendMessageErr error
}

func (m *MockWSClient) SendMessage(payload interface{}) error {
	if m.SendMessageErr != nil {
		return m.SendMessageErr
	}
	m.SentMessages = append(m.SentMessages, payload)
	return nil
}

func (m *MockWSClient) ClientID() Token {
	return m.Token
}

func TestVoiceChannelService_Join(t *testing.T) {
	service := NewVoiceChannelService()
	mockClient := &MockWSClient{Token: "token-123"}

	// Test successful join
	err := service.Join(mockClient)
	if err != nil {
		t.Fatalf("Expected no error on Join, got %v", err)
	}

	// Verify participant was added
	if len(service.GetParticipants()) != 1 {
		t.Fatalf("Expected 1 participant, got %d", len(service.GetParticipants()))
	}

	// Test joining same user again (update state)
	err = service.Join(mockClient)
	if err != nil {
		t.Fatalf("Expected no error on re-join, got %v", err)
	}

	if len(service.GetParticipants()) != 1 {
		t.Fatal("Participant count should not increase on re-join")
	}
}

func TestVoiceChannelService_Leave(t *testing.T) {
	service := NewVoiceChannelService()
	mockClient := &MockWSClient{Token: "token-123"}

	// Must join before leaving
	err := service.Join(mockClient)
	if err != nil {
		t.Fatalf("Failed initial join: %v", err)
	}

	// Successful leave
	err = service.Leave(mockClient)
	if err != nil {
		t.Fatalf("Expected no error on Leave, got %v", err)
	}

	// Verify removed
	if len(service.GetParticipants()) != 0 {
		t.Fatalf("Expected 0 participants after leave, got %d", len(service.GetParticipants()))
	}

	// Test leaving twice
	err = service.Leave(mockClient)
	if err == nil {
		t.Fatal("Expected error on second leave")
	}
}

func TestVoiceChannelService_ToggleMute(t *testing.T) {
	service := NewVoiceChannelService()
	mockClient := &MockWSClient{Token: "token-123"}
	
	// Join first
	service.Join(mockClient)

	// Toggle Mute
	err := service.ToggleMute(mockClient)
	if err != nil {
		t.Fatalf("Failed to toggle mute: %v", err)
	}

	participants := service.GetParticipants()
	if len(participants) != 1 {
		t.Fatal("Participant list size changed")
	}

	if participants[0].IsMuted {
		t.Fatal("Expected speaker to not be muted initially")
	}

	// Toggle again
	err = service.ToggleMute(mockClient)
	if err != nil {
		t.Fatalf("Failed to toggle mute again: %v", err)
	}

	if !participants[0].IsMuted {
		t.Fatal("Expected speaker to be muted after toggle")
	}
}

func TestVoiceChannelService_ToggleDeafen(t *testing.T) {
	service := NewVoiceChannelService()
	mockClient := &MockWSClient{Token: "token-123"}
	service.Join(mockClient)

	err := service.ToggleDeafen(mockClient)
	if err != nil {
		t.Fatalf("Failed to toggle deafen: %v", err)
	}

	participants := service.GetParticipants()
	if !participants[0].IsDeafened {
		t.Fatal("Expected participant to be deafened")
	}
}

func TestVoiceChannelService_InvalidOperation(t *testing.T) {
	service := NewVoiceChannelService()
	mockClient := &MockWSClient{Token: "token-123"}

	tests := []struct {
		name    string
		action  func() error
		wantErr bool
	}{
		{"Mute before join", func() error { return service.ToggleMute(mockClient) }, true},
		{"Deafen before join", func() error { return service.ToggleDeafen(mockClient) }, true},
		{"SelfMute before join", func() error { return service.ToggleSelfMute(mockClient) }, true},
		{"SelfDeafen before join", func() error { return service.ToggleSelfDeafen(mockClient) }, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.action(); (err != nil) != tt.wantErr {
				t.Errorf("Action() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}