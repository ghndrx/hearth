package services

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	"hearth/internal/models"
)

// ErrNotificationNotFound is defined in errors.go

// NotificationRepository defines the interface for notification data access
type NotificationRepository interface {
	Create(ctx context.Context, notification *models.Notification) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Notification, error)
	GetByIDWithActor(ctx context.Context, id uuid.UUID) (*models.NotificationWithActor, error)
	List(ctx context.Context, userID uuid.UUID, opts models.NotificationListOptions) ([]models.NotificationWithActor, error)
	GetStats(ctx context.Context, userID uuid.UUID) (*models.NotificationStats, error)
	MarkAsRead(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	MarkAllAsRead(ctx context.Context, userID uuid.UUID) (int64, error)
	Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	DeleteAllRead(ctx context.Context, userID uuid.UUID) (int64, error)
	DeleteOlderThan(ctx context.Context, userID uuid.UUID, before time.Time) (int64, error)
}

// NotificationService handles notification business logic
type NotificationService struct {
	repo     NotificationRepository
	eventBus EventBus
}

// NewNotificationService creates a new notification service
func NewNotificationService(repo NotificationRepository, eventBus EventBus) *NotificationService {
	return &NotificationService{
		repo:     repo,
		eventBus: eventBus,
	}
}

// CreateNotification creates a new notification
func (s *NotificationService) CreateNotification(ctx context.Context, req *models.CreateNotificationRequest) (*models.Notification, error) {
	notification := &models.Notification{
		UserID:    req.UserID,
		Type:      req.Type,
		Title:     req.Title,
		Body:      req.Body,
		Data:      req.Data,
		ActorID:   req.ActorID,
		ServerID:  req.ServerID,
		ChannelID: req.ChannelID,
		MessageID: req.MessageID,
	}

	if err := s.repo.Create(ctx, notification); err != nil {
		return nil, err
	}

	// Emit event
	s.eventBus.Publish("notification.created", &NotificationCreatedEvent{
		Notification: notification,
	})

	return notification, nil
}

// GetNotification retrieves a notification by ID
func (s *NotificationService) GetNotification(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*models.NotificationWithActor, error) {
	notification, err := s.repo.GetByIDWithActor(ctx, id)
	if err != nil {
		return nil, err
	}
	if notification == nil || notification.UserID != userID {
		return nil, ErrNotificationNotFound
	}
	return notification, nil
}

// ListNotifications retrieves notifications for a user
func (s *NotificationService) ListNotifications(ctx context.Context, userID uuid.UUID, opts models.NotificationListOptions) ([]models.NotificationWithActor, error) {
	return s.repo.List(ctx, userID, opts)
}

// GetNotificationStats retrieves notification statistics for a user
func (s *NotificationService) GetNotificationStats(ctx context.Context, userID uuid.UUID) (*models.NotificationStats, error) {
	return s.repo.GetStats(ctx, userID)
}

// MarkAsRead marks a notification as read
func (s *NotificationService) MarkAsRead(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	err := s.repo.MarkAsRead(ctx, id, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotificationNotFound
		}
		return err
	}

	// Emit event
	s.eventBus.Publish("notification.read", &NotificationReadEvent{
		NotificationID: id,
		UserID:         userID,
	})

	return nil
}

// MarkAllAsRead marks all notifications as read for a user
func (s *NotificationService) MarkAllAsRead(ctx context.Context, userID uuid.UUID) (int64, error) {
	count, err := s.repo.MarkAllAsRead(ctx, userID)
	if err != nil {
		return 0, err
	}

	if count > 0 {
		// Emit event
		s.eventBus.Publish("notification.all_read", &NotificationAllReadEvent{
			UserID: userID,
			Count:  count,
		})
	}

	return count, nil
}

// DeleteNotification deletes a notification
func (s *NotificationService) DeleteNotification(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	err := s.repo.Delete(ctx, id, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotificationNotFound
		}
		return err
	}

	// Emit event
	s.eventBus.Publish("notification.deleted", &NotificationDeletedEvent{
		NotificationID: id,
		UserID:         userID,
	})

	return nil
}

// DeleteAllReadNotifications deletes all read notifications for a user
func (s *NotificationService) DeleteAllReadNotifications(ctx context.Context, userID uuid.UUID) (int64, error) {
	count, err := s.repo.DeleteAllRead(ctx, userID)
	if err != nil {
		return 0, err
	}

	if count > 0 {
		s.eventBus.Publish("notification.read_deleted", &NotificationReadDeletedEvent{
			UserID: userID,
			Count:  count,
		})
	}

	return count, nil
}

// Events

// NotificationCreatedEvent is emitted when a notification is created
type NotificationCreatedEvent struct {
	Notification *models.Notification
}

// NotificationReadEvent is emitted when a notification is marked as read
type NotificationReadEvent struct {
	NotificationID uuid.UUID
	UserID         uuid.UUID
}

// NotificationAllReadEvent is emitted when all notifications are marked as read
type NotificationAllReadEvent struct {
	UserID uuid.UUID
	Count  int64
}

// NotificationDeletedEvent is emitted when a notification is deleted
type NotificationDeletedEvent struct {
	NotificationID uuid.UUID
	UserID         uuid.UUID
}

// NotificationReadDeletedEvent is emitted when all read notifications are deleted
type NotificationReadDeletedEvent struct {
	UserID uuid.UUID
	Count  int64
}
