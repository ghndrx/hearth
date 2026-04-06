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

// ChannelNotificationOverrideRepository defines the interface for channel override data access
type ChannelNotificationOverrideRepository interface {
	Set(ctx context.Context, override *models.ChannelNotificationOverride) error
	Get(ctx context.Context, userID, channelID uuid.UUID) (*models.ChannelNotificationOverride, error)
	GetByUser(ctx context.Context, userID uuid.UUID) ([]models.ChannelNotificationOverride, error)
	Delete(ctx context.Context, userID, channelID uuid.UUID) error
	GetForChannels(ctx context.Context, userID uuid.UUID, channelIDs []uuid.UUID) (map[uuid.UUID]models.ChannelNotificationLevel, error)
}

// NotificationService handles notification business logic
type NotificationService struct {
	repo             NotificationRepository
	channelOverrideRepo ChannelNotificationOverrideRepository
	eventBus         EventBus
}

// NewNotificationService creates a new notification service
func NewNotificationService(repo NotificationRepository, eventBus EventBus) *NotificationService {
	return &NotificationService{
		repo:     repo,
		eventBus: eventBus,
	}
}

// NewNotificationServiceWithOverrides creates a new notification service with channel override support
func NewNotificationServiceWithOverrides(repo NotificationRepository, channelOverrideRepo ChannelNotificationOverrideRepository, eventBus EventBus) *NotificationService {
	return &NotificationService{
		repo:             repo,
		channelOverrideRepo: channelOverrideRepo,
		eventBus:         eventBus,
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

// SetChannelOverride sets a notification override for a specific channel
func (s *NotificationService) SetChannelOverride(ctx context.Context, userID, channelID uuid.UUID, level models.ChannelNotificationLevel) (*models.ChannelNotificationOverride, error) {
	if s.channelOverrideRepo == nil {
		return nil, errors.New("channel override repository not configured")
	}

	override := &models.ChannelNotificationOverride{
		UserID:            userID,
		ChannelID:         channelID,
		NotificationLevel: level,
	}

	if err := s.channelOverrideRepo.Set(ctx, override); err != nil {
		return nil, err
	}

	// Emit event
	s.eventBus.Publish("notification.channel_override_set", &ChannelOverrideSetEvent{
		UserID:            userID,
		ChannelID:         channelID,
		NotificationLevel: level,
	})

	return override, nil
}

// GetChannelOverride retrieves the notification override for a specific channel
func (s *NotificationService) GetChannelOverride(ctx context.Context, userID, channelID uuid.UUID) (*models.ChannelNotificationOverride, error) {
	if s.channelOverrideRepo == nil {
		return nil, errors.New("channel override repository not configured")
	}

	override, err := s.channelOverrideRepo.Get(ctx, userID, channelID)
	if err != nil {
		return nil, err
	}

	// If no override exists, return default
	if override == nil {
		return models.DefaultChannelNotificationOverride(userID, channelID), nil
	}

	return override, nil
}

// ClearChannelOverride removes the notification override for a specific channel
func (s *NotificationService) ClearChannelOverride(ctx context.Context, userID, channelID uuid.UUID) error {
	if s.channelOverrideRepo == nil {
		return errors.New("channel override repository not configured")
	}

	if err := s.channelOverrideRepo.Delete(ctx, userID, channelID); err != nil {
		return err
	}

	// Emit event
	s.eventBus.Publish("notification.channel_override_cleared", &ChannelOverrideClearedEvent{
		UserID:    userID,
		ChannelID: channelID,
	})

	return nil
}

// ListChannelOverrides retrieves all channel notification overrides for a user
func (s *NotificationService) ListChannelOverrides(ctx context.Context, userID uuid.UUID) ([]models.ChannelNotificationOverride, error) {
	if s.channelOverrideRepo == nil {
		return []models.ChannelNotificationOverride{}, nil
	}

	return s.channelOverrideRepo.GetByUser(ctx, userID)
}

// GetChannelOverrideForNotification retrieves the effective notification level for a channel
// Returns the override level or "all_messages" if no override exists
func (s *NotificationService) GetChannelOverrideForNotification(ctx context.Context, userID, channelID uuid.UUID) models.ChannelNotificationLevel {
	if s.channelOverrideRepo == nil {
		return models.ChannelNotificationLevelAllMessages
	}

	override, err := s.channelOverrideRepo.Get(ctx, userID, channelID)
	if err != nil || override == nil {
		return models.ChannelNotificationLevelAllMessages
	}

	return override.NotificationLevel
}

// ShouldNotify determines if a notification should be sent based on channel override
// Returns true if the notification should proceed, false if it should be suppressed
func (s *NotificationService) ShouldNotify(ctx context.Context, userID, channelID uuid.UUID, isMention bool) bool {
	level := s.GetChannelOverrideForNotification(ctx, userID, channelID)

	switch level {
	case models.ChannelNotificationLevelNothing:
		return false
	case models.ChannelNotificationLevelMentionsOnly:
		return isMention
	case models.ChannelNotificationLevelAllMessages:
		return true
	default:
		return true
	}
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

// ChannelOverrideSetEvent is emitted when a channel notification override is set
type ChannelOverrideSetEvent struct {
	UserID            uuid.UUID
	ChannelID         uuid.UUID
	NotificationLevel models.ChannelNotificationLevel
}

// ChannelOverrideClearedEvent is emitted when a channel notification override is cleared
type ChannelOverrideClearedEvent struct {
	UserID    uuid.UUID
	ChannelID uuid.UUID
}
