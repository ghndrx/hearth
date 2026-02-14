package services

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

// MockStore implements NotificationStore for testing. 
// It fulfills the interface contract and allows us to control the state.
type MockStore struct {
	Notifications map[string]Notification
	ShouldFail    bool // Simulates a database failure
}

// Helper to intialize mock data
func NewMockStore() *MockStore {
	return &MockStore{
		Notifications: make(map[string]Notification),
	}
}

func (m *MockStore) Create(ctx context.Context, notif Notification) (Notification, error) {
	if m.ShouldFail {
		return Notification{}, errors.New("simulated DB error")
	}
	m.Notifications[notif.ID] = notif
	return notif, nil
}

func (m *MockStore) MarkRead(ctx context.Context, id, userID string) error {
	if m.ShouldFail {
		return errors.New("simulated DB error")
	}

	// Security check: ensure the user owns the notification or matches expectation
	if n, ok := m.Notifications[id]; !ok {
		return ErrNotFound
	}
	if n.UserID != userID {
		return errors.New("unauthorized: user does not own notification")
	}

	m.Notifications[id].IsRead = true
	return nil
}

func (m *MockStore) GetUnread(ctx context.Context, userID string, limit int) ([]Notification, error) {
	if m.ShouldFail {
		return nil, errors.New("simulated DB error")
	}

	var results []Notification
	for _, n := range m.Notifications {
		if !n.IsRead && n.UserID == userID {
			results = append(results, n)
			if limit > 0 && len(results) >= limit {
				break
			}
		}
	}
	return results, nil
}

func TestCreate_Success(t *testing.T) {
	mockStore := NewMockStore()
	service := NewNotificationService(mockStore)

	ctx := context.Background()
	userID := "user-123"
	content := "Welcome to Hearth!"

	notif, err := service.Create(ctx, userID, content)

	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if notif.UserID != userID || !notif.IsRead {
		t.Errorf("Unexpected notification fields: User=%s, IsRead=%v", notif.UserID, notif.IsRead)
	}
}

func TestCreate_ValidationError(t *testing.T) {
	mockStore := NewMockStore()
	service := NewNotificationService(mockStore)

	ctx := context.Background()

	_, err := service.Create(ctx, "", "No user ID")
	if err == nil {
		t.Error("Expected error for empty user ID, got nil")
	}
}

func TestMarkRead_Success(t *testing.T) {
	mockStore := NewMockStore()
	service := NewNotificationService(mockStore)

	ctx := context.Background()
	userID := "user-123"
	notifID := "notif-10001"

	mockStore.Notifications[notifID] = Notification{
		ID:     notifID,
		UserID: userID,
	}

	err := service.MarkRead(ctx, notifID, userID)
	if err != nil {
		t.Errorf("MarkRead failed: %v", err)
	}

	// Verify state in store
	if !mockStore.Notifications[notifID].IsRead {
		t.Error("Notification was not marked as read in mock store")
	}
}

func TestMarkRead_NotFound(t *testing.T) {
	mockStore := NewMockStore()
	service := NewNotificationService(mockStore)

	ctx := context.Background()
	err := service.MarkRead(ctx, "non-existent-id", "user-123")
	
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestGetUnread_Success(t *testing.T) {
	mockStore := NewMockStore()
	service := NewNotificationService(mockStore)

	ctx := context.Background()
	userID := "user-123"

	// Add two unread notifications
	mockStore.Notifications["1"] = Notification{ID: "1", UserID: userID, IsRead: false}
	mockStore.Notifications["2"] = Notification{ID: "2", UserID: "user-456", IsRead: false} // Another user
	mockStore.Notifications["3"] = Notification{ID: "3", UserID: userID, IsRead: true}   // Read notification

	// Get only unread
	notifications, err := service.GetUnread(ctx, userID, 5)

	if err != nil {
		t.Fatalf("GetUnread failed: %v", err)
	}

	if len(notifications) != 1 {
		t.Errorf("Expected 1 unread notification, got %d", len(notifications))
	}

	if notifications[0].ID != "1" {
		t.Errorf("Expected notification ID 1, got %s", notifications[0].ID)
	}
}

func TestGetUnread_EmptyResult(t *testing.T) {
	mockStore := NewMockStore()
	service := NewNotificationService(mockStore)
	ctx := context.Background()

	notifications, err := service.GetUnread(ctx, "user-999", 10)

	if err != nil {
		t.Fatalf("GetUnread failed: %v", err)
	}

	if len(notifications) != 0 {
		t.Errorf("Expected 0 notifications, got %d", len(notifications))
	}
}