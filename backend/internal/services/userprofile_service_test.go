package services

import (
	"context"
	"errors"
	"testing"
)

// mockUserProfileStore is a concrete implementation of UserProfileStore for testing.
// It allows controlled responses (success or error).
type mockUserProfileStore struct {
	users map[string]*UserProfile
}

func newMockStore() *mockUserProfileStore {
	return &mockUserProfileStore{
		users: make(map[string]*UserProfile),
	}
}

func (m *mockUserProfileStore) GetUser(ctx context.Context, userID string) (*UserProfile, error) {
	if user, exists := m.users[userID]; exists {
		return user, nil
	}
	return nil, errors.New("user not found in mock store")
}

func (m *mockUserProfileStore) SaveUser(ctx context.Context, userID string, profile *UserProfile) error {
	m.users[userID] = profile
	return nil
}

func TestGetUserProfile_Success(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	service := NewUserProfileService(store)

	expectedUser := &UserProfile{
		UserID:   "user123",
		Username: "HearthPlayer",
		Bio:      "Keep track of the packs I opened!",
		Status:   "Playing Hearthstone",
	}
	store.users["user123"] = expectedUser

	got, err := service.GetUserProfile(ctx, "user123")

	if err != nil {
		t.Fatalf("GetUserProfile() unexpected error: %v", err)
	}

	if got.UserID != expectedUser.UserID {
		t.Errorf("GetUserProfile() UserID = %v, want %v", got.UserID, expectedUser.UserID)
	}
}

func TestGetUserProfile_EmptyID(t *testing.T) {
	ctx := context.Background()
	service := NewUserProfileService(newMockStore())

	_, err := service.GetUserProfile(ctx, "")

	if err == nil {
		t.Error("GetUserProfile() expected error for empty ID, got nil")
	}
}

func TestUpdateUserProfile_Success(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	service := NewUserProfileService(store)

	// 1. Setup initial state in the mock store
	initialProfile := &UserProfile{
		UserID: "user123",
		Bio:    "Old bio text",
	}
	store.users["user123"] = initialProfile

	// 2. Prepare updates
	updates := &UserProfile{
		Username: "NewHearthPlayer",
		Status:   "Online",
	}

	// 3. Execute Update
	err := service.UpdateUserProfile(ctx, "user123", updates)
	if err != nil {
		t.Fatalf("UpdateUserProfile() failed: %v", err)
	}

	// 4. Verify result
	updatedProfile, _ := store.GetUser(ctx, "user123")
	if updatedProfile.Username != "NewHearthPlayer" {
		t.Errorf("Username was not updated. got %s, want %s", updatedProfile.Username, "NewHearthPlayer")
	}
	if updatedProfile.Status != "Online" {
		t.Errorf("Status was not updated. got %s, want %s", updatedProfile.Status, "Online")
	}
	// Verify other fields are preserved (Merge logic)
	if updatedProfile.Bio != "Old bio text" {
		t.Errorf("Bio was unexpectedly modified. got %s, want %s", updatedProfile.Bio, "Old bio text")
	}
}

func TestUpdateUserProfile_NilUpdates(t *testing.T) {
	ctx := context.Background()
	service := NewUserProfileService(newMockStore())

	err := service.UpdateUserProfile(ctx, "user123", nil)

	if err == nil {
		t.Error("UpdateUserProfile() expected error for nil updates, got nil")
	}
}