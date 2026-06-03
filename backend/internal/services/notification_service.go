package services

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"strings"
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

// NotificationCoordinator orchestrates the entire notification pipeline
// from scoring to routing to delivery across all channels (push, email, in-app)
type NotificationCoordinator struct {
	smartService        *SmartNotificationService
	pushService         *PushDeliveryService
	notifService        *NotificationService
	eventBus            EventBus
	cache               SmartNotificationCache
	queueRepo           NotificationQueueRepository
	channelPrefsRepo    ChannelNotificationPrefsRepo
	channelOverrideRepo ChannelNotificationOverrideRepository
	serverPrefsRepo     ServerNotificationPrefsRepo
}

// NotificationQueueRepository defines notification queue data access
type NotificationQueueRepository interface {
	Create(ctx context.Context, item *models.NotificationQueueItem) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status models.QueueItemStatus, lastError *string) error
	MarkProcessed(ctx context.Context, id uuid.UUID) error
	IncrementAttempts(ctx context.Context, id uuid.UUID, lastError string) error
	GetPending(ctx context.Context, limit int) ([]models.NotificationQueueItem, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// ChannelNotificationPrefsRepo defines channel notification preferences data access
type ChannelNotificationPrefsRepo interface {
	Get(ctx context.Context, userID, channelID uuid.UUID) (*models.ChannelNotificationPreference, error)
	Upsert(ctx context.Context, pref *models.ChannelNotificationPreference) error
	GetByUser(ctx context.Context, userID uuid.UUID) ([]models.ChannelNotificationPreference, error)
	Delete(ctx context.Context, userID, channelID uuid.UUID) error
}

// ServerNotificationPrefsRepo defines server notification preferences data access
type ServerNotificationPrefsRepo interface {
	Get(ctx context.Context, userID, serverID uuid.UUID) (*models.ServerNotificationPreference, error)
	Upsert(ctx context.Context, pref *models.ServerNotificationPreference) error
	GetByUser(ctx context.Context, userID uuid.UUID) ([]models.ServerNotificationPreference, error)
	Delete(ctx context.Context, userID, serverID uuid.UUID) error
}

// NewNotificationCoordinator creates a new notification coordinator
func NewNotificationCoordinator(
	smartService *SmartNotificationService,
	pushService *PushDeliveryService,
	notifService *NotificationService,
	eventBus EventBus,
	cache SmartNotificationCache,
) *NotificationCoordinator {
	return &NotificationCoordinator{
		smartService: smartService,
		pushService:  pushService,
		notifService: notifService,
		eventBus:     eventBus,
		cache:        cache,
	}
}

// NotificationInput contains all inputs needed to process a notification
type NotificationInput struct {
	Type        models.NotificationType
	Title       string
	Body        string
	Data        string
	ActorID     *uuid.UUID
	SenderID    *uuid.UUID
	RecipientID uuid.UUID
	ServerID    *uuid.UUID
	ChannelID   *uuid.UUID
	MessageID   *uuid.UUID
	HasMention  bool
	IsDM        bool
	IsReply     bool
}

// ProcessNotification processes a notification through the entire pipeline:
// scoring -> routing -> storage -> delivery
func (c *NotificationCoordinator) ProcessNotification(ctx context.Context, input *NotificationInput) (*models.NotificationRoutingDecision, error) {
	// 1. Score the notification for priority
	scoringInput := &models.PriorityScoringInput{
		NotificationType: input.Type,
		SenderID:         input.SenderID,
		RecipientID:      input.RecipientID,
		ServerID:         input.ServerID,
		ChannelID:        input.ChannelID,
		HasMention:       input.HasMention,
		IsDM:             input.IsDM,
		IsReply:          input.IsReply,
	}

	smartNotif, err := c.smartService.ScoreNotification(ctx, scoringInput)
	if err != nil {
		return nil, fmt.Errorf("failed to score notification: %w", err)
	}

	// 2. Check per-channel overrides (highest specificity - simplified level-based system)
	if input.ChannelID != nil {
		if override, err := c.getChannelOverride(ctx, input.RecipientID, *input.ChannelID); err == nil && override != nil {
			switch override.NotificationLevel {
			case models.ChannelNotificationLevelNothing:
				return &models.NotificationRoutingDecision{
					Channel:    models.NotificationChannelSuppressed,
					ShouldSend: false,
					Reason:     "channel notification override: nothing",
				}, nil
			case models.ChannelNotificationLevelMentionsOnly:
				if !input.HasMention {
					return &models.NotificationRoutingDecision{
						Channel:    models.NotificationChannelSuppressed,
						ShouldSend: false,
						Reason:     "channel notification override: mentions only",
					}, nil
				}
				// Has mention - proceed with immediate delivery
				smartNotif.DeliveryMode = models.DeliveryImmediate
			case models.ChannelNotificationLevelAllMessages:
				// All messages allowed - proceed with normal flow
			}
		}
	}

	// 3. Check detailed per-channel preferences (granular toggle-based system)
	if input.ChannelID != nil {
		if channelPref, err := c.getChannelPreference(ctx, input.RecipientID, *input.ChannelID); err == nil && channelPref != nil {
			// Apply channel-specific delivery mode override
			if channelPref.DeliveryMode == models.DeliveryImmediate {
				smartNotif.DeliveryMode = models.DeliveryImmediate
			} else if channelPref.DeliveryMode == models.DeliveryBatched {
				smartNotif.DeliveryMode = models.DeliveryBatched
			}
			// Check if this notification type is enabled for this channel
			if !c.isNotificationTypeEnabled(input.Type, channelPref) {
				return &models.NotificationRoutingDecision{
					Channel:    models.NotificationChannelSuppressed,
					ShouldSend: false,
					Reason:     fmt.Sprintf("notification type %s disabled for channel", input.Type),
				}, nil
			}
			// Check mute status
			if channelPref.Muted {
				smartNotif.DeliveryMode = models.DeliveryBatched
			}
		}
	}

	// 4. Check per-server preferences
	if input.ServerID != nil {
		if serverPref, err := c.getServerPreference(ctx, input.RecipientID, *input.ServerID); err == nil && serverPref != nil {
			if serverPref.Muted {
				smartNotif.DeliveryMode = models.DeliveryBatched
			}
			// Check role-based filtering
			if len(serverPref.NotifyRoles) > 0 && input.ActorID != nil {
				// Would need to check if actor has one of the notify roles
				// This is handled separately via member role lookup
			}
		}
	}

	// 5. Route notification based on priority and preferences
	routed, err := c.smartService.RouteNotification(ctx, input.RecipientID, smartNotif)
	if err != nil {
		return nil, fmt.Errorf("failed to route notification: %w", err)
	}

	decision := &models.NotificationRoutingDecision{
		Priority:   routed.Priority,
		Channel:    c.deliveryModeToChannel(routed.DeliveryMode),
		ShouldSend: true,
		DelaySecs:  c.computeDelay(routed.DeliveryMode, routed.Priority),
		Reason:     fmt.Sprintf("priority=%s, delivery=%s", routed.Priority, routed.DeliveryMode),
	}

	// 6. Create the notification record
	notif, err := c.notifService.CreateNotification(ctx, &models.CreateNotificationRequest{
		UserID:    input.RecipientID,
		Type:      input.Type,
		Title:     input.Title,
		Body:      input.Body,
		Data:      &input.Data,
		ActorID:   input.ActorID,
		ServerID:  input.ServerID,
		ChannelID: input.ChannelID,
		MessageID: input.MessageID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create notification: %w", err)
	}

	smartNotif.ID = notif.ID
	smartNotif.PriorityScore = routed.PriorityScore
	smartNotif.Priority = routed.Priority
	smartNotif.DeliveryMode = routed.DeliveryMode

	// 7. Route to appropriate channel(s)
	switch decision.Channel {
	case models.NotificationChannelPush:
		if err := c.deliverPush(ctx, input.RecipientID, smartNotif); err != nil {
			log.Printf("coordinator: push delivery failed: %v", err)
		}
	case models.NotificationChannelInApp:
		// In-app notifications are stored; no immediate action needed
		c.eventBus.Publish("notification.created", &NotificationCreatedEvent{
			Notification: notif,
		})
	case models.NotificationChannelSuppressed:
		// Notification stored but no delivery
		return decision, nil
	}

	// 8. If batched, add to digest queue
	if routed.DeliveryMode == models.DeliveryBatched {
		if err := c.smartService.AddToDigestQueue(ctx, input.RecipientID, smartNotif); err != nil {
			log.Printf("coordinator: failed to add to digest queue: %v", err)
		}
	}

	return decision, nil
}

// deliveryModeToChannel converts delivery mode to notification channel
func (c *NotificationCoordinator) deliveryModeToChannel(mode models.NotificationDeliveryMode) models.NotificationChannel {
	switch mode {
	case models.DeliveryImmediate:
		return models.NotificationChannelPush
	case models.DeliveryBatched:
		return models.NotificationChannelInApp
	default:
		return models.NotificationChannelInApp
	}
}

// computeDelay computes delivery delay based on delivery mode and priority
func (c *NotificationCoordinator) computeDelay(mode models.NotificationDeliveryMode, priority models.NotificationPriority) int {
	switch {
	case priority == models.NotificationPriorityUrgent:
		return 0
	case mode == models.DeliveryImmediate:
		return 0
	case mode == models.DeliveryBatched:
		return 30 // 30 second grace period before batching
	default:
		return 0
	}
}

// deliverPush sends a push notification
func (c *NotificationCoordinator) deliverPush(ctx context.Context, userID uuid.UUID, smartNotif *models.SmartNotification) error {
	prefs, err := c.pushService.GetPreferences(ctx, userID)
	if err != nil {
		return err
	}

	// Check notification type preference
	switch smartNotif.Type {
	case models.NotificationTypeMention:
		if !prefs.PushMentions {
			return nil
		}
	case models.NotificationTypeDirectMessage:
		if !prefs.PushDirectMessages {
			return nil
		}
	case models.NotificationTypeReply:
		if !prefs.PushReplies {
			return nil
		}
	}

	payload := &models.PushPayload{
		Title:     smartNotif.Title,
		Body:      smartNotif.Body,
		Icon:      "/icons/notification-icon.png",
		Badge:     "/icons/badge-icon.png",
		Tag:       smartNotif.ID.String(),
		Data:      nil,
		Timestamp: smartNotif.CreatedAt.Unix(),
	}

	return c.pushService.SendPushNotification(ctx, userID, payload, prefs)
}

// getChannelPreference retrieves channel notification preferences
func (c *NotificationCoordinator) getChannelPreference(ctx context.Context, userID, channelID uuid.UUID) (*models.ChannelNotificationPreference, error) {
	if c.channelPrefsRepo == nil {
		return nil, fmt.Errorf("channel prefs repo not configured")
	}
	return c.channelPrefsRepo.Get(ctx, userID, channelID)
}

// getChannelOverride retrieves simplified channel notification override
func (c *NotificationCoordinator) getChannelOverride(ctx context.Context, userID, channelID uuid.UUID) (*models.ChannelNotificationOverride, error) {
	if c.channelOverrideRepo == nil {
		return nil, nil // No override repo configured, use defaults
	}
	return c.channelOverrideRepo.Get(ctx, userID, channelID)
}

// getServerPreference retrieves server notification preferences
func (c *NotificationCoordinator) getServerPreference(ctx context.Context, userID, serverID uuid.UUID) (*models.ServerNotificationPreference, error) {
	if c.serverPrefsRepo == nil {
		return nil, fmt.Errorf("server prefs repo not configured")
	}
	return c.serverPrefsRepo.Get(ctx, userID, serverID)
}

// isNotificationTypeEnabled checks if a notification type is enabled for a channel preference
func (c *NotificationCoordinator) isNotificationTypeEnabled(t models.NotificationType, pref *models.ChannelNotificationPreference) bool {
	switch t {
	case models.NotificationTypeMention:
		return pref.EnableMentions
	case models.NotificationTypeDirectMessage:
		return pref.EnableMessages
	case models.NotificationTypeReaction:
		return pref.EnableReactions
	case models.NotificationTypeReply:
		return pref.EnableThreads
	default:
		return true
	}
}

// --- Channel Notification Preferences ---

// GetChannelPreference returns channel notification preferences
func (c *NotificationCoordinator) GetChannelPreference(ctx context.Context, userID, channelID, serverID uuid.UUID) (*models.ChannelNotificationPreference, error) {
	pref, err := c.channelPrefsRepo.Get(ctx, userID, channelID)
	if err != nil {
		// Return defaults if not found
		return models.DefaultChannelNotificationPreference(userID, channelID, serverID), nil
	}
	return pref, nil
}

// UpdateChannelPreference updates channel notification preferences
func (c *NotificationCoordinator) UpdateChannelPreference(ctx context.Context, userID, channelID, serverID uuid.UUID, req *models.UpdateChannelNotificationPreferenceRequest) (*models.ChannelNotificationPreference, error) {
	pref, err := c.channelPrefsRepo.Get(ctx, userID, channelID)
	if err != nil || pref == nil {
		pref = models.DefaultChannelNotificationPreference(userID, channelID, serverID)
	}

	if req.EnableMentions != nil {
		pref.EnableMentions = *req.EnableMentions
	}
	if req.EnableMessages != nil {
		pref.EnableMessages = *req.EnableMessages
	}
	if req.EnableReactions != nil {
		pref.EnableReactions = *req.EnableReactions
	}
	if req.EnableThreads != nil {
		pref.EnableThreads = *req.EnableThreads
	}
	if req.EnablePins != nil {
		pref.EnablePins = *req.EnablePins
	}
	if req.EnableVoiceActivity != nil {
		pref.EnableVoiceActivity = *req.EnableVoiceActivity
	}
	if req.DeliveryMode != nil {
		pref.DeliveryMode = *req.DeliveryMode
	}
	if req.Muted != nil {
		pref.Muted = *req.Muted
	}
	pref.UpdatedAt = time.Now()

	if err := c.channelPrefsRepo.Upsert(ctx, pref); err != nil {
		return nil, fmt.Errorf("failed to upsert channel preference: %w", err)
	}

	return pref, nil
}

// --- Channel Notification Overrides (Simplified) ---

// GetChannelOverride returns the simplified channel notification override for a user
func (c *NotificationCoordinator) GetChannelOverride(ctx context.Context, userID, channelID uuid.UUID) (*models.ChannelNotificationOverride, error) {
	if c.channelOverrideRepo == nil {
		// Return default if no repo configured
		return models.DefaultChannelNotificationOverride(userID, channelID), nil
	}

	override, err := c.channelOverrideRepo.Get(ctx, userID, channelID)
	if err != nil {
		return nil, err
	}

	if override == nil {
		return models.DefaultChannelNotificationOverride(userID, channelID), nil
	}

	return override, nil
}

// SetChannelOverride sets or updates a channel notification override
func (c *NotificationCoordinator) SetChannelOverride(ctx context.Context, userID, channelID uuid.UUID, level models.ChannelNotificationLevel) (*models.ChannelNotificationOverride, error) {
	if c.channelOverrideRepo == nil {
		return nil, fmt.Errorf("channel override repository not configured")
	}

	override := &models.ChannelNotificationOverride{
		UserID:            userID,
		ChannelID:         channelID,
		NotificationLevel: level,
	}

	if err := c.channelOverrideRepo.Set(ctx, override); err != nil {
		return nil, fmt.Errorf("failed to set channel override: %w", err)
	}

	c.eventBus.Publish("notification.channel_override_set", map[string]interface{}{
		"user_id":            userID,
		"channel_id":         channelID,
		"notification_level": level,
	})

	return override, nil
}

// ClearChannelOverride removes a channel notification override
func (c *NotificationCoordinator) ClearChannelOverride(ctx context.Context, userID, channelID uuid.UUID) error {
	if c.channelOverrideRepo == nil {
		return fmt.Errorf("channel override repository not configured")
	}

	if err := c.channelOverrideRepo.Delete(ctx, userID, channelID); err != nil {
		return fmt.Errorf("failed to clear channel override: %w", err)
	}

	c.eventBus.Publish("notification.channel_override_cleared", map[string]interface{}{
		"user_id":    userID,
		"channel_id": channelID,
	})

	return nil
}

// ListChannelOverrides returns all channel notification overrides for a user
func (c *NotificationCoordinator) ListChannelOverrides(ctx context.Context, userID uuid.UUID) ([]models.ChannelNotificationOverride, error) {
	if c.channelOverrideRepo == nil {
		return []models.ChannelNotificationOverride{}, nil
	}

	return c.channelOverrideRepo.GetByUser(ctx, userID)
}

// --- Server Notification Preferences ---

// GetServerPreference returns server notification preferences
func (c *NotificationCoordinator) GetServerPreference(ctx context.Context, userID, serverID uuid.UUID) (*models.ServerNotificationPreference, error) {
	pref, err := c.serverPrefsRepo.Get(ctx, userID, serverID)
	if err != nil {
		return models.DefaultServerNotificationPreference(userID, serverID), nil
	}
	return pref, nil
}

// UpdateServerPreference updates server notification preferences
func (c *NotificationCoordinator) UpdateServerPreference(ctx context.Context, userID, serverID uuid.UUID, req *models.UpdateServerNotificationPreferenceRequest) (*models.ServerNotificationPreference, error) {
	pref, err := c.serverPrefsRepo.Get(ctx, userID, serverID)
	if err != nil || pref == nil {
		pref = models.DefaultServerNotificationPreference(userID, serverID)
	}

	if req.EnableMentions != nil {
		pref.EnableMentions = *req.EnableMentions
	}
	if req.EnableMessages != nil {
		pref.EnableMessages = *req.EnableMessages
	}
	if req.EnableReactions != nil {
		pref.EnableReactions = *req.EnableReactions
	}
	if req.EnableThreads != nil {
		pref.EnableThreads = *req.EnableThreads
	}
	if req.NotifyRoles != nil {
		pref.NotifyRoles = req.NotifyRoles
	}
	if req.Muted != nil {
		pref.Muted = *req.Muted
	}
	pref.UpdatedAt = time.Now()

	if err := c.serverPrefsRepo.Upsert(ctx, pref); err != nil {
		return nil, fmt.Errorf("failed to upsert server preference: %w", err)
	}

	return pref, nil
}

// --- Batch Delivery Worker ---

// ProcessQueue processes pending notification queue items
func (c *NotificationCoordinator) ProcessQueue(ctx context.Context, limit int) error {
	items, err := c.queueRepo.GetPending(ctx, limit)
	if err != nil {
		return fmt.Errorf("failed to get pending queue items: %w", err)
	}

	for _, item := range items {
		if err := c.processQueueItem(ctx, &item); err != nil {
			log.Printf("coordinator: failed to process queue item %s: %v", item.ID, err)
			_ = c.queueRepo.IncrementAttempts(ctx, item.ID, err.Error())
			continue
		}
		_ = c.queueRepo.MarkProcessed(ctx, item.ID)
	}

	return nil
}

// processQueueItem processes a single queue item
func (c *NotificationCoordinator) processQueueItem(ctx context.Context, item *models.NotificationQueueItem) error {
	notif, err := c.notifService.GetNotification(ctx, item.NotificationID, item.UserID)
	if err != nil {
		return fmt.Errorf("failed to get notification: %w", err)
	}

	smartNotif := &models.SmartNotification{
		Notification: notif.Notification,
		Priority:     item.Priority,
	}

	switch item.Channel {
	case models.NotificationChannelPush:
		return c.deliverPush(ctx, item.UserID, smartNotif)
	case models.NotificationChannelEmail:
		return c.deliverEmail(ctx, item.UserID, smartNotif)
	default:
		return nil
	}
}

// deliverEmail sends an email notification (placeholder for email service integration)
func (c *NotificationCoordinator) deliverEmail(ctx context.Context, userID uuid.UUID, smartNotif *models.SmartNotification) error {
	// Email delivery would integrate with an email service (SendGrid, SES, SMTP)
	// For now, this is a placeholder that logs the intent
	key := fmt.Sprintf("notif_email:%s", smartNotif.ID.String())
	data, _ := json.Marshal(map[string]interface{}{
		"user_id":         userID,
		"notification_id": smartNotif.ID,
		"title":           smartNotif.Title,
		"body":            smartNotif.Body,
		"timestamp":       time.Now(),
	})
	_ = c.cache.Set(ctx, key, data, 7*24*time.Hour) // retain for 7 days

	c.eventBus.Publish("notification.email_queued", map[string]interface{}{
		"user_id":         userID,
		"notification_id": smartNotif.ID,
	})

	return nil
}

// PushDeliveryService handles sending push notifications to Web Push endpoints
type PushDeliveryService struct {
	VAPIDPublicKey  string
	VAPIDPrivateKey string
	VAPIDSubject    string

	httpClient *http.Client
	subsRepo   PushSubscriptionRepository
	cache      SmartNotificationCache
	eventBus   EventBus
}

// PushSubscriptionRepository defines push subscription data access
type PushSubscriptionRepository interface {
	Create(ctx context.Context, sub *models.PushSubscription) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByEndpoint(ctx context.Context, userID uuid.UUID, endpoint string) error
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.PushSubscription, error)
	GetActiveByUserID(ctx context.Context, userID uuid.UUID) ([]*models.PushSubscription, error)
}

// NewPushDeliveryService creates a new push delivery service
func NewPushDeliveryService(
	VAPIDPublicKey, VAPIDPrivateKey, VAPIDSubject string,
	subsRepo PushSubscriptionRepository,
	cache SmartNotificationCache,
	eventBus EventBus,
) *PushDeliveryService {
	return &PushDeliveryService{
		VAPIDPublicKey:  VAPIDPublicKey,
		VAPIDPrivateKey: VAPIDPrivateKey,
		VAPIDSubject:    VAPIDSubject,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		subsRepo: subsRepo,
		cache:    cache,
		eventBus: eventBus,
	}
}

// RegisterSubscription registers a push subscription for a user
func (s *PushDeliveryService) RegisterSubscription(ctx context.Context, userID uuid.UUID, req *models.CreatePushSubscriptionRequest) error {
	sub := &models.PushSubscription{
		ID:        uuid.New(),
		UserID:    userID,
		Endpoint:  req.Endpoint,
		P256dh:    req.P256dh,
		Auth:      req.Auth,
		UserAgent: req.UserAgent,
		CreatedAt: time.Now(),
	}

	// Check if subscription already exists
	existing, err := s.subsRepo.GetActiveByUserID(ctx, userID)
	if err == nil {
		for _, existingSub := range existing {
			if existingSub.Endpoint == req.Endpoint {
				return nil // already registered
			}
		}
	}

	return s.subsRepo.Create(ctx, sub)
}

// UnregisterSubscription removes a push subscription
func (s *PushDeliveryService) UnregisterSubscription(ctx context.Context, userID uuid.UUID, endpoint string) error {
	return s.subsRepo.DeleteByEndpoint(ctx, userID, endpoint)
}

// GetPreferences returns notification preferences for a user
func (s *PushDeliveryService) GetPreferences(ctx context.Context, userID uuid.UUID) (*models.NotificationPreferences, error) {
	key := fmt.Sprintf("notif_prefs:%s", userID.String())
	data, err := s.cache.Get(ctx, key)
	if err != nil {
		return models.DefaultNotificationPreferences(userID), nil
	}

	var prefs models.NotificationPreferences
	if err := json.Unmarshal(data, &prefs); err != nil {
		return models.DefaultNotificationPreferences(userID), nil
	}

	return &prefs, nil
}

// UpdatePreferences updates notification preferences for a user
func (s *PushDeliveryService) UpdatePreferences(ctx context.Context, userID uuid.UUID, req *models.UpdateNotificationPreferencesRequest) (*models.NotificationPreferences, error) {
	prefs, err := s.GetPreferences(ctx, userID)
	if err != nil {
		prefs = models.DefaultNotificationPreferences(userID)
	}

	if req.PushEnabled != nil {
		prefs.PushEnabled = *req.PushEnabled
	}
	if req.PushMentions != nil {
		prefs.PushMentions = *req.PushMentions
	}
	if req.PushDirectMessages != nil {
		prefs.PushDirectMessages = *req.PushDirectMessages
	}
	if req.PushReplies != nil {
		prefs.PushReplies = *req.PushReplies
	}
	if req.PushFriendRequests != nil {
		prefs.PushFriendRequests = *req.PushFriendRequests
	}
	if req.PushServerInvites != nil {
		prefs.PushServerInvites = *req.PushServerInvites
	}
	if req.SoundEnabled != nil {
		prefs.SoundEnabled = *req.SoundEnabled
	}
	if req.SoundMessage != nil {
		prefs.SoundMessage = *req.SoundMessage
	}
	if req.SoundMention != nil {
		prefs.SoundMention = *req.SoundMention
	}
	if req.DesktopEnabled != nil {
		prefs.DesktopEnabled = *req.DesktopEnabled
	}
	if req.DesktopPreviews != nil {
		prefs.DesktopPreviews = *req.DesktopPreviews
	}
	if req.DoNotDisturb != nil {
		prefs.DoNotDisturb = *req.DoNotDisturb
	}
	if req.DoNotDisturbUntil != nil {
		prefs.DoNotDisturbUntil = req.DoNotDisturbUntil
	}
	prefs.UpdatedAt = time.Now()

	key := fmt.Sprintf("notif_prefs:%s", userID.String())
	data, _ := json.Marshal(prefs)
	_ = s.cache.Set(ctx, key, data, 0)

	return prefs, nil
}

// SendPushNotification sends a push notification to all of a user's active subscriptions
func (s *PushDeliveryService) SendPushNotification(ctx context.Context, userID uuid.UUID, payload *models.PushPayload, prefs *models.NotificationPreferences) error {
	if !prefs.PushEnabled {
		return nil
	}

	// Check DND
	if prefs.DoNotDisturb {
		if prefs.DoNotDisturbUntil != nil && time.Now().After(*prefs.DoNotDisturbUntil) {
			// DND expired
		} else if prefs.DoNotDisturbUntil == nil {
			// Indefinite DND
			return nil
		}
	}

	subs, err := s.subsRepo.GetActiveByUserID(ctx, userID)
	if err != nil || len(subs) == 0 {
		return fmt.Errorf("no active push subscriptions")
	}

	var lastErr error
	for _, sub := range subs {
		if err := s.sendToSubscription(ctx, sub, payload); err != nil {
			lastErr = err
			if strings.Contains(err.Error(), "410") || strings.Contains(err.Error(), "400") {
				_ = s.subsRepo.Delete(ctx, sub.ID)
			}
		}
	}

	return lastErr
}

// sendToSubscription sends a push notification to a single subscription
func (s *PushDeliveryService) sendToSubscription(ctx context.Context, sub *models.PushSubscription, payload *models.PushPayload) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	encrypted, err := s.encryptPayload(sub, payloadBytes)
	if err != nil {
		return fmt.Errorf("failed to encrypt payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sub.Endpoint, bytes.NewReader(encrypted))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("TTL", "0")
	req.Header.Set("Urgency", "normal")
	req.Header.Set("Authorization", fmt.Sprintf("WebPush %s", s.VAPIDPublicKey))

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send push: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusGone {
		return fmt.Errorf("subscription expired (410)")
	}
	if resp.StatusCode == http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("bad request: %s (400)", string(body))
	}
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("push service returned %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// encryptPayload encrypts a payload for Web Push using ECIES-AES-GCM
func (s *PushDeliveryService) encryptPayload(sub *models.PushSubscription, plaintext []byte) ([]byte, error) {
	// Decode subscription keys
	p256dh, err := base64.URLEncoding.DecodeString(sub.P256dh)
	if err != nil {
		return nil, fmt.Errorf("failed to decode p256dh: %w", err)
	}

	auth, err := base64.URLEncoding.DecodeString(sub.Auth)
	if err != nil {
		return nil, fmt.Errorf("failed to decode auth: %w", err)
	}

	// Generate ephemeral ECDH key pair (P-256)
	ephemeralPrivKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ephemeral key: %w", err)
	}
	ephemeralX, ephemeralY := ephemeralPrivKey.PublicKey.X, ephemeralPrivKey.PublicKey.Y

	// Parse peer public key (uncompressed format 0x04 || X || Y)
	peerX := new(big.Int).SetBytes(p256dh[1:33])
	peerY := new(big.Int).SetBytes(p256dh[33:65])

	// Derive shared point
	sharedX, _ := elliptic.P256().ScalarMult(peerX, peerY, ephemeralX.Bytes())

	// Shared secret is SHA-256 of the X coordinate
	sharedSecret := sha256.Sum256(sharedX.Bytes())

	// HKDF-like key derivation with auth as salt
	// Content Encryption Key (CEK) = HMAC-SHA256(auth, sharedSecret)[0:16]
	authHMAC := hmacSHA256(auth, sharedSecret[:])
	cek := authHMAC[:16]

	// Generate random 12-byte nonce
	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// AES-128-GCM encryption
	block, err := aes.NewCipher(cek)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// Build Web Push message format:
	// [0x04 || ephemeralX (32 bytes) || ephemeralY (32 bytes)] (65 bytes)
	// [nonce (12 bytes)]
	// [ciphertext]
	result := make([]byte, 0, 65+12+len(ciphertext))
	result = append(result, 0x04)
	result = append(result, elliptic.Marshal(elliptic.P256(), ephemeralX, ephemeralY)...)
	result = append(result, nonce...)
	result = append(result, ciphertext...)

	return result, nil
}

// hmacSHA256 computes HMAC-SHA256
func hmacSHA256(key, msg []byte) [32]byte {
	// Use crypto/hkdf-like approach with crypto/hmac
	// Simplified implementation
	var kpad [64]byte
	var opad [64]byte

	copy(kpad[:], key)
	if len(key) > 64 {
		h := sha256.Sum256(key)
		copy(kpad[:], h[:])
	}

	for i := range kpad {
		opad[i] = kpad[i] ^ 0x5c
		kpad[i] ^= 0x36
	}

	h1 := sha256.Sum256(append(kpad[:], msg...))
	return sha256.Sum256(append(opad[:], h1[:]...))
}

// MentionRepository defines the interface for mention data access
type MentionRepository interface {
	Create(ctx context.Context, mention *models.Mention) error
	CreateBatch(ctx context.Context, mentions []*models.Mention) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Mention, error)
	GetMentionsWithContext(ctx context.Context, filter *models.MentionFilter) ([]models.MentionWithContext, int, error)
	GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error)
	GetStats(ctx context.Context, userID uuid.UUID) (*models.MentionStats, error)
	MarkAsRead(ctx context.Context, id, userID uuid.UUID) error
	MarkAllAsRead(ctx context.Context, userID uuid.UUID) (int, error)
	MarkChannelMentionsAsRead(ctx context.Context, userID, channelID uuid.UUID) (int, error)
	DeleteByMessage(ctx context.Context, messageID uuid.UUID) error
	DeleteByChannel(ctx context.Context, channelID uuid.UUID) error
	DeleteByUser(ctx context.Context, userID uuid.UUID) error
	Search(ctx context.Context, userID uuid.UUID, query string, limit int) ([]models.MentionWithContext, error)
}

// Errors
var (
	ErrMentionNotFound = errors.New("mention not found")
)

// MentionAPIService handles mention-related business logic for the API
type MentionAPIService struct {
	mentionRepo MentionRepository
	eventBus    EventBus
}

// NewMentionAPIService creates a new mention API service
func NewMentionAPIService(mentionRepo MentionRepository, eventBus EventBus) *MentionAPIService {
	return &MentionAPIService{
		mentionRepo: mentionRepo,
		eventBus:    eventBus,
	}
}

// GetMentions retrieves mentions for a user with the given filter
func (s *MentionAPIService) GetMentions(ctx context.Context, filter *models.MentionFilter) ([]models.MentionWithContext, int, error) {
	return s.mentionRepo.GetMentionsWithContext(ctx, filter)
}

// GetUnreadCount returns the count of unread mentions for a user
func (s *MentionAPIService) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.mentionRepo.GetUnreadCount(ctx, userID)
}

// GetStats returns mention statistics for a user
func (s *MentionAPIService) GetStats(ctx context.Context, userID uuid.UUID) (*models.MentionStats, error) {
	return s.mentionRepo.GetStats(ctx, userID)
}

// MarkAsRead marks a mention as read
func (s *MentionAPIService) MarkAsRead(ctx context.Context, mentionID, userID uuid.UUID) error {
	// Verify the mention exists and belongs to the user
	mention, err := s.mentionRepo.GetByID(ctx, mentionID)
	if err != nil {
		return err
	}
	if mention == nil || mention.UserID != userID {
		return ErrMentionNotFound
	}

	if err := s.mentionRepo.MarkAsRead(ctx, mentionID, userID); err != nil {
		return err
	}

	// Emit event
	if s.eventBus != nil {
		s.eventBus.Publish("mentions.read", &MentionReadEvent{
			UserID:    userID,
			MentionID: mentionID,
		})
	}

	return nil
}

// MarkAllAsRead marks all mentions as read for a user
func (s *MentionAPIService) MarkAllAsRead(ctx context.Context, userID uuid.UUID) (int, error) {
	count, err := s.mentionRepo.MarkAllAsRead(ctx, userID)
	if err != nil {
		return 0, err
	}

	// Emit event
	if s.eventBus != nil && count > 0 {
		s.eventBus.Publish("mentions.read_all", &MentionsReadAllEvent{
			UserID: userID,
			Count:  count,
		})
	}

	return count, nil
}

// MarkChannelMentionsAsRead marks all mentions in a channel as read
func (s *MentionAPIService) MarkChannelMentionsAsRead(ctx context.Context, userID, channelID uuid.UUID) (int, error) {
	count, err := s.mentionRepo.MarkChannelMentionsAsRead(ctx, userID, channelID)
	if err != nil {
		return 0, err
	}

	// Emit event
	if s.eventBus != nil && count > 0 {
		s.eventBus.Publish("mentions.channel_read", &MentionsChannelReadEvent{
			UserID:    userID,
			ChannelID: channelID,
			Count:     count,
		})
	}

	return count, nil
}

// Search searches mentions by query
func (s *MentionAPIService) Search(ctx context.Context, userID uuid.UUID, query string, limit int) ([]models.MentionWithContext, error) {
	return s.mentionRepo.Search(ctx, userID, query, limit)
}

// CreateMention creates a mention record (called from message service)
func (s *MentionAPIService) CreateMention(ctx context.Context, req *models.CreateMentionRequest) error {
	mention := &models.Mention{
		UserID:             req.UserID,
		MessageID:          req.MessageID,
		MentionedBy:        req.MentionedBy,
		ChannelID:          req.ChannelID,
		GuildID:            req.GuildID,
		MentionType:        req.MentionType,
		MentionedRoleID:    req.MentionedRoleID,
		MentionedChannelID: req.MentionedChannelID,
	}

	if err := s.mentionRepo.Create(ctx, mention); err != nil {
		return err
	}

	// Emit event for real-time delivery
	if s.eventBus != nil {
		s.eventBus.Publish("mention.created", &MentionCreatedEvent{
			Mention: mention,
		})
	}

	return nil
}

// CreateMentionsBatch creates multiple mention records
func (s *MentionAPIService) CreateMentionsBatch(ctx context.Context, mentions []*models.Mention) error {
	if len(mentions) == 0 {
		return nil
	}

	if err := s.mentionRepo.CreateBatch(ctx, mentions); err != nil {
		return err
	}

	// Emit events for each mention
	if s.eventBus != nil {
		for _, m := range mentions {
			s.eventBus.Publish("mention.created", &MentionCreatedEvent{
				Mention: m,
			})
		}
	}

	return nil
}

// DeleteByMessage deletes mentions when a message is deleted
func (s *MentionAPIService) DeleteByMessage(ctx context.Context, messageID uuid.UUID) error {
	return s.mentionRepo.DeleteByMessage(ctx, messageID)
}

// Events

// MentionCreatedEvent is emitted when a mention is created
type MentionCreatedEvent struct {
	Mention *models.Mention
}

// MentionReadEvent is emitted when a mention is marked as read
type MentionReadEvent struct {
	UserID    uuid.UUID
	MentionID uuid.UUID
}

// MentionsReadAllEvent is emitted when all mentions are marked as read
type MentionsReadAllEvent struct {
	UserID uuid.UUID
	Count  int
}

// MentionsChannelReadEvent is emitted when channel mentions are marked as read
type MentionsChannelReadEvent struct {
	UserID    uuid.UUID
	ChannelID uuid.UUID
	Count     int
}