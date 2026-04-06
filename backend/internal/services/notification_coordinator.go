package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"hearth/internal/models"
)

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
