package services

import (
	"context"
	"errors"
	"testing"
)

// mockRepository is an in-memory implementation of IChannelRepository
// used strictly for testing purposes.
type mockRepository struct {
	channels map[string][]string // guildID -> list of channels
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		channels: make(map[string][]string),
	}
}

func (m *mockRepository) FindChannelsByGuildID(ctx context.Context, guildID string) ([]string, error) {
	if m.channels[guildID] == nil {
		return []string{}, nil // Empty list if no channels found
	}
	return m.channels[guildID], nil
}

func (m *mockRepository) AddChannel(ctx context.Context, guildID, channelName string) error {
	if m.channels[guildID] == nil {
		m.channels[guildID] = make([]string, 0)
	}
	m.channels[guildID] = append(m.channels[guildID], channelName)
	return nil
}

// Test_GetChannels_Success tests the happy path.
func Test_GetChannels_Success(t *testing.T) {
	repo := newMockRepository()
	// Mock Data
	repo.channels["guild-123"] = []string{"general", "random"}
	service := NewChannelService(repo)

	got, err := service.GetChannels(context.Background(), "guild-123")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(got) != 2 {
		t.Errorf("expected 2 channels, got %d", len(got))
	}
	if got[0] != "general" || got[1] != "random" {
		t.Errorf("unexpected channel list: %v", got)
	}
}

// Test_GetChannels_EmptyGuildID tests error handling for invalid input.
func Test_GetChannels_EmptyGuildID(t *testing.T) {
	repo := newMockRepository()
	service := NewChannelService(repo)

	_, err := service.GetChannels(context.Background(), "")

	if err == nil {
		t.Error("expected error for empty guildID, but got nil")
	}
	// More granular check depends on how you want to check error strings vs types
	if err.Error() != "guildID cannot be empty" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// Test_CreateChannel_Success tests business logic execution.
func Test_CreateChannel_Success(t *testing.T) {
	repo := newMockRepository()
	service := NewChannelService(repo)

	err := service.CreateChannel(context.Background(), "guild-123", "music")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify the repo was called correctly using mock properties
	channels, _ := repo.FindChannelsByGuildID(context.Background(), "guild-123")
	if len(channels) != 1 {
		t.Fatalf("Expected 1 channel, got %d", len(channels))
	}
	if channels[0] != "music" {
		t.Errorf("Expected 'music', got %s", channels[0])
	}
}

// Test_CreateChannel_ValidationError tests the guard clause in the service.
func Test_CreateChannel_ValidationError(t *testing.T) {
	repo := newMockRepository()
	service := NewChannelService(repo)

	// Test empty name
	err := service.CreateChannel(context.Background(), "guild-123", "")
	if err == nil || err.Error() != "channelName cannot be empty" {
		t.Errorf("expected validation error for empty name, got: %v", err)
	}
	// Test empty guildID
	err = service.CreateChannel(context.Background(), "", "general")
	if err == nil || err.Error() != "guildID cannot be empty" {
		t.Errorf("expected validation error for empty guildID, got: %v", err)
	}
}

// Test_CreateChannel_RepositoryError tests error propagation from the repository.
func Test_CreateChannel_RepositoryError(t *testing.T) {
	// Create a mock that *always* returns an error
	repo := newMockRepository()
	service := NewChannelService(repo)

	// Overriding the repo method locally for this test instance
	repo.AddChannel = func(ctx context.Context, guildID, channelName string) error {
		return errors.New("database connection failed")
	}

	err := service.CreateChannel(context.Background(), "guild-123", "test")

	if err == nil {
		t.Error("expected error from repository, but got nil")
	}
	if err.Error() != "failed to add channel: database connection failed" {
		t.Errorf("unexpected error message: %v", err)
	}
}