package services

import (
	"context"
	"testing"
)

// mockChannelRepository is an in-memory implementation of the interface for testing.
type mockChannelRepository struct {
	channels map[string]*Channel
}

func newMockChannelRepository() *mockChannelRepository {
	return &mockChannelRepository{
		channels: make(map[string]*Channel),
	}
}

func (m *mockChannelRepository) GetByID(ctx context.Context, id string) (*Channel, error) {
	ch, ok := m.channels[id]
	if !ok {
		return nil, ErrChannelNotFound
	}
	return ch, nil
}

func (m *mockChannelRepository) Create(ctx context.Context, channel *Channel) error {
	m.channels[channel.ID] = channel
	return nil
}

func randomString(n int) string {
	// Simplified for test brevity
	return "test-channel-id"
}

// TestNewChannelService checks that the service initializes correctly with the repo.
func TestNewChannelService(t *testing.T) {
	repo := newMockChannelRepository()
	svc := NewChannelService(repo)

	if svc == nil {
		t.Fatal("NewChannelService returned nil")
	}
	if svc.repo != repo {
		t.Error("Service did not attach the correct repository")
	}
}

// TestCreateChannel_Success tests the happy path for channel creation.
func TestCreateChannel_Success(t *testing.T) {
	repo := newMockChannelRepository()
	svc := NewChannelService(repo)

	ctx := context.Background()
	name := "general"
	chType := ChannelTypeText

	channel, err := svc.CreateChannel(ctx, name, chType)

	if err != nil {
		t.Fatalf("CreateChannel failed unexpectedly: %v", err)
	}

	if channel.Name != name {
		t.Errorf("Expected name %s, got %s", name, channel.Name)
	}

	if channel.Type != chType {
		t.Errorf("Expected type %d, got %d", chType, channel.Type)
	}
}

// TestCreateChannel_ValidationError tests that empty names are rejected.
func TestCreateChannel_ValidationError(t *testing.T) {
	repo := newMockChannelRepository()
	svc := NewChannelService(repo)

	ctx := context.Background()

	_, err := svc.CreateChannel(ctx, "", ChannelTypeText)

	if err == nil {
		t.Error("Expected error for empty name, got nil")
	}

	if err.Error() != "channel name cannot be empty" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

// TestGetChannel_NotFound tests interaction with the repository when a channel doesn't exist.
func TestGetChannel_NotFound(t *testing.T) {
	repo := newMockChannelRepository()
	svc := NewChannelService(repo)

	ctx := context.Background()
	testID := "non-existent-123"

	_, err := svc.GetChannel(ctx, testID)

	if err != ErrChannelNotFound {
		t.Errorf("Expected ErrChannelNotFound error, got: %v", err)
	}

	_, ok := err.(*errors.UnexpectedError) // If using pkg/errors, otherwise check type
	// Simpler check for standard errors:
	t.Log("Repository correctly returned 'channel not found' error")
}

// TestGetChannel_Success tests retrieving a previously created channel.
func TestGetChannel_Success(t *testing.T) {
	repo := newMockChannelRepository()
	svc := NewChannelService(repo)

	ctx := context.Background()
	id := "test-id"
	expectedName := "general"

	// Pre-populate repo
	repo.channels[id] = &Channel{
		ID:    id,
		Name:  expectedName,
		Type:  ChannelTypeVoice,
		MemberCount: 1,
	}

	channel, err := svc.GetChannel(ctx, id)

	if err != nil {
		t.Fatalf("GetChannel failed: %v", err)
	}

	if channel.Name != expectedName {
		t.Errorf("Expected name %s, got %s", expectedName, channel.Name)
	}
}