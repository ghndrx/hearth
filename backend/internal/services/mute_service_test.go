package services

import (
	"context"
	"sync"
	"testing"
	"time"
)

// mockStore implements the services.Store interface for testing.
type mockStore struct {
	mutes map[string]*Mute // Key: "userID_channelID"
	mu    sync.RWMutex
}

func newMockStore() *mockStore {
	return &mockStore{
		mutes: make(map[string]*Mute),
	}
}

func (m *mockStore) GetUserMuteStatus(ctx context.Context, userID, channelID string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	mute, exists := m.mutes[userID+"_"+channelID]
	return exists && mute.ExpiresAt.After(time.Now()), nil
}

func (m *mockStore) SaveMute(ctx context.Context, mute *Mute) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mutes[mute.UserID+"_"+mute.ChannelID] = mute
	return nil
}

func (m *mockStore) DeleteMute(ctx context.Context, userID, channelID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.mutes, userID+"_"+channelID)
	return nil
}

// Helper for Store.GetAllMutesForChannel (assumed).
// Implemented here for the test to compile if the mock supports it.
func (m *mockStore) GetAllMutesForChannel(ctx context.Context, channelID string) ([]*Mute, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var result []*Mute
	for _, mute := range m.mutes {
		if mute.ChannelID == channelID {
			// Only return valid (non-expired) mutes in tests
			if mute.ExpiresAt.After(time.Now()) {
				result = append(result, mute)
			}
		}
	}
	return result, nil
}

func TestMuteUser(t *testing.T) {
	store := newMockStore()
	service := NewMuteService(store)
	ctx := context.Background()

	t.Run("Successfully mute user", func(t *testing.T) {
		mute, err := service.MuteUser(ctx, "user123", "channel456", 5*time.Minute)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if mute == nil {
			t.Fatal("Expected mute object, got nil")
		}
		if mute.UserID != "user123" {
			t.Errorf("Expected userID user123, got %s", mute.UserID)
		}
	})

	t.Run("Attempt to mute same user again", func(t *testing.T) {
		_, err := service.MuteUser(ctx, "user123", "channel456", 5*time.Minute)
		if err != ErrMuteAlreadyActive {
			t.Errorf("Expected ErrMuteAlreadyActive, got: %v", err)
		}
	})
}

func TestUnmuteUser(t *testing.T) {
	store := newMockStore()
	service := NewMuteService(store)
	ctx := context.Background()

	// Setup: Create a mute
	service.MuteUser(ctx, "user789", "channel101", 10*time.Minute)

	t.Run("Successfully unmute user", func(t *testing.T) {
		err := service.UnmuteUser(ctx, "user789", "channel101")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
	})

	t.Run("Try to unmute already unmuted user", func(t *testing.T) {
		err := service.UnmuteUser(ctx, "user789", "channel101")
		if err != ErrUserNotMuted {
			t.Errorf("Expected ErrUserNotMuted, got: %v", err)
		}
	})
}

func TestGetUserMuted(t *testing.T) {
	store := newMockStore()
	service := NewMuteService(store)
	ctx := context.Background()

	// Setup: No mute initially
	isMuted, err := service.GetUserMuted(ctx, "user001", "channel002")
	if err != nil {
		t.Fatalf("Unexpected error checking status: %v", err)
	}
	if isMuted {
		t.Error("Expected user to not be muted initially")
	}

	// Setup: Mute user
	service.MuteUser(ctx, "user001", "channel002", 10*time.Minute)

	isMuted, err = service.GetUserMuted(ctx, "user001", "channel002")
	if err != nil {
		t.Fatalf("Unexpected error checking status: %v", err)
	}
	if !isMuted {
		t.Error("Expected user to be muted")
	}
}

func TestGetMutes(t *testing.T) {
	store := newMockStore()
	service := NewMuteService(store)
	ctx := context.Background()

	// Setup: Mut different users in same channel
	service.MuteUser(ctx, "userA", "channelX", 1*time.Hour)
	service.MuteUser(ctx, "userB", "channelX", 1*time.Hour)

	// Verify they are active
	toret, err := service.GetMutes(ctx, "channelX")
	if err != nil {
		t.Fatalf("Unexpected error getting mutes: %v", err)
	}
	if len(toret) != 2 {
		t.Errorf("Expected 2 mutes, got %d", len(toret))
	}
}