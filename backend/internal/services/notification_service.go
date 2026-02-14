package services

import (
	"context"
	"errors"
	"fmt"
)

// ErrNotFound is returned when a requested notification is not found.
var ErrNotFound = errors.New("notification not found")

// Notification represents a user notification message.
type Notification struct {
	ID        string
	UserID    string
	Content   string
	IsRead    bool
	Version   int // Optimistic locking version
}

// NotificationStore defines the contract for persistence operations.
// This allows for swapping out SQL, Mongo, etc., without changing the service.
type NotificationStore interface {
	// Create persists a new notification.
	Create(ctx context.Context, notif Notification) (Notification, error)
	
	// MarkRead updates the notification status to read.
	MarkRead(ctx context.Context, id string, userID string) error
	
	// GetUnread retrieves unread notifications for the specified user.
	GetUnread(ctx context.Context, userID string, limit int) ([]Notification, error)
}

// NotificationService handles business logic for notifications.
type NotificationService struct {
	store NotificationStore
}

// NewNotificationService creates a new instance of the NotificationService.
func NewNotificationService(store NotificationStore) *NotificationService {
	return &NotificationService{
		store: store,
	}
}

// Create creates and persists a new notification.
func (s *NotificationService) Create(ctx context.Context, userID, content string) (Notification, error) {
	if userID == "" {
		return Notification{}, fmt.Errorf("user ID cannot be empty")
	}
	if content == "" {
		return Notification{}, fmt.Errorf("content cannot be empty")
	}

	notif := Notification{
		ID:      generateID(), // In a real app, use UUID or ID generator
		UserID:  userID,
		Content: content,
		IsRead:  false,
		Version: 1,
	}

	// Call the underlying store
	return s.store.Create(ctx, notif)
}

// MarkRead marks a specific notification as read by a user.
func (s *NotificationService) MarkRead(ctx context.Context, id, userID string) error {
	if id == "" {
		return fmt.Errorf("notification ID cannot be empty")
	}
	if userID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	// Call the underlying store
	return s.store.MarkRead(ctx, id, userID)
}

// GetUnread retrieves a list of unread notifications for a given user.
// It supports pagination via the limit parameter (limit 0 means no limit).
func (s *NotificationService) GetUnread(ctx context.Context, userID string, limit int) ([]Notification, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	notifications, err := s.store.GetUnread(ctx, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch unread notifications: %w", err)
	}

	return notifications, nil
}

// Helper to generate a simple ID for testing purposes.
// In production, use a proper UUID generation library.
func generateID() string {
	return "notif-" + fmt.Sprintf("%d", 10000)
}