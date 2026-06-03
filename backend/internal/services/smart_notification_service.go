package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"hearth/internal/models"
)

// SmartNotificationCache defines the cache operations needed for smart notifications
type SmartNotificationCache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

// SmartNotificationService handles AI-powered notification intelligence
type SmartNotificationService struct {
	notifRepo NotificationRepository
	cache     SmartNotificationCache
	eventBus  EventBus
	mu        sync.RWMutex
}

// NewSmartNotificationService creates a new smart notification service
func NewSmartNotificationService(
	notifRepo NotificationRepository,
	cache SmartNotificationCache,
	eventBus EventBus,
) *SmartNotificationService {
	return &SmartNotificationService{
		notifRepo: notifRepo,
		cache:     cache,
		eventBus:  eventBus,
	}
}

// --- Priority Scoring ---

// ScoreNotification computes a priority score (0-100) for a notification
func (s *SmartNotificationService) ScoreNotification(ctx context.Context, input *models.PriorityScoringInput) (*models.SmartNotification, error) {
	score := 0
	priority := models.NotificationPriorityNormal
	delivery := models.DeliveryBatched
	category := classifyCategory(input.NotificationType)

	// Base score by notification type
	switch input.NotificationType {
	case models.NotificationTypeDirectMessage:
		score += 80
	case models.NotificationTypeMention:
		score += 75
	case models.NotificationTypeReply:
		score += 60
	case models.NotificationTypeFriendRequest, models.NotificationTypeFriendAccept:
		score += 50
	case models.NotificationTypeReaction:
		score += 20
	case models.NotificationTypeServerInvite, models.NotificationTypeServerJoin:
		score += 40
	case models.NotificationTypeSystem:
		score += 30
	case models.NotificationTypeEventInvite, models.NotificationTypeEventStart:
		score += 55
	case models.NotificationTypeEventRSVP, models.NotificationTypeEventReminder:
		score += 45
	default:
		score += 25
	}

	// Boost for mentions
	if input.HasMention {
		score += 15
	}

	// Boost for DMs
	if input.IsDM {
		score += 10
	}

	// Boost for replies to user's own messages
	if input.IsReply {
		score += 10
	}

	// Check engagement history for sender importance
	if input.SenderID != nil {
		engagementBoost := s.getSenderEngagementBoost(ctx, input.RecipientID, *input.SenderID)
		score += engagementBoost
	}

	// Cap at 100
	if score > 100 {
		score = 100
	}

	// Determine priority level
	switch {
	case score >= 75:
		priority = models.NotificationPriorityUrgent
		delivery = models.DeliveryImmediate
	case score >= 50:
		priority = models.NotificationPriorityHigh
		delivery = models.DeliveryImmediate
	case score >= 25:
		priority = models.NotificationPriorityNormal
		delivery = models.DeliveryBatched
	default:
		priority = models.NotificationPriorityLow
		delivery = models.DeliveryBatched
	}

	return &models.SmartNotification{
		PriorityScore: score,
		Priority:      priority,
		DeliveryMode:  delivery,
		Category:      category,
	}, nil
}

// getSenderEngagementBoost returns a priority boost based on how often the recipient engages with the sender
func (s *SmartNotificationService) getSenderEngagementBoost(ctx context.Context, recipientID, senderID uuid.UUID) int {
	key := fmt.Sprintf("smart_notif:engagement:%s:%s", recipientID.String(), senderID.String())
	data, err := s.cache.Get(ctx, key)
	if err != nil {
		return 0
	}

	var interactions int
	if err := json.Unmarshal(data, &interactions); err != nil {
		return 0
	}

	// More interactions = higher boost (max 10)
	boost := interactions / 5
	if boost > 10 {
		boost = 10
	}
	return boost
}

// --- Snooze / Mute ---

// SnoozeNotifications snoozes notifications for a user
func (s *SmartNotificationService) SnoozeNotifications(ctx context.Context, userID uuid.UUID, req *models.SnoozeRequest) (*models.SnoozeConfig, error) {
	until := time.Now().Add(time.Duration(req.DurationMins) * time.Minute)

	config := &models.SnoozeConfig{
		UserID:    userID,
		Active:    true,
		Until:     until,
		ServerID:  req.ServerID,
		ChannelID: req.ChannelID,
		CreatedAt: time.Now(),
	}

	key := snoozeKey(userID, req.ServerID, req.ChannelID)
	data, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal snooze config: %w", err)
	}

	ttl := time.Duration(req.DurationMins) * time.Minute
	if err := s.cache.Set(ctx, key, data, ttl); err != nil {
		return nil, fmt.Errorf("failed to save snooze config: %w", err)
	}

	s.eventBus.Publish("notification.snoozed", &SmartNotificationSnoozedEvent{
		UserID: userID,
		Until:  until,
	})

	return config, nil
}

// UnsnoozeNotifications removes a snooze for a user
func (s *SmartNotificationService) UnsnoozeNotifications(ctx context.Context, userID uuid.UUID, serverID, channelID *uuid.UUID) error {
	key := snoozeKey(userID, serverID, channelID)
	if err := s.cache.Delete(ctx, key); err != nil {
		return fmt.Errorf("failed to remove snooze: %w", err)
	}

	s.eventBus.Publish("notification.unsnoozed", &SmartNotificationUnsnoozedEvent{
		UserID: userID,
	})

	return nil
}

// IsNotificationSnoozed checks if a user has snoozed notifications
func (s *SmartNotificationService) IsNotificationSnoozed(ctx context.Context, userID uuid.UUID, serverID, channelID *uuid.UUID) (bool, *time.Time, error) {
	// Check channel-level snooze first, then server, then global
	checks := []struct{ s, c *uuid.UUID }{
		{serverID, channelID},
		{serverID, nil},
		{nil, nil},
	}

	for _, check := range checks {
		key := snoozeKey(userID, check.s, check.c)
		data, err := s.cache.Get(ctx, key)
		if err != nil {
			continue
		}

		var config models.SnoozeConfig
		if err := json.Unmarshal(data, &config); err != nil {
			continue
		}

		if config.Active && config.Until.After(time.Now()) {
			return true, &config.Until, nil
		}
	}

	return false, nil, nil
}

// MuteNotifications mutes/unmutes notifications for a user
func (s *SmartNotificationService) MuteNotifications(ctx context.Context, userID uuid.UUID, req *models.MuteRequest) (*models.MuteConfig, error) {
	config := &models.MuteConfig{
		UserID:    userID,
		ServerID:  req.ServerID,
		ChannelID: req.ChannelID,
		Muted:     req.Muted,
		CreatedAt: time.Now(),
	}

	key := muteKey(userID, req.ServerID, req.ChannelID)

	if req.Muted {
		data, err := json.Marshal(config)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal mute config: %w", err)
		}
		// Mutes don't expire
		if err := s.cache.Set(ctx, key, data, 0); err != nil {
			return nil, fmt.Errorf("failed to save mute config: %w", err)
		}
	} else {
		if err := s.cache.Delete(ctx, key); err != nil {
			return nil, fmt.Errorf("failed to remove mute: %w", err)
		}
	}

	s.eventBus.Publish("notification.mute_changed", &SmartNotificationMuteEvent{
		UserID: userID,
		Muted:  req.Muted,
	})

	return config, nil
}

// IsNotificationMuted checks if notifications are muted for a given scope
func (s *SmartNotificationService) IsNotificationMuted(ctx context.Context, userID uuid.UUID, serverID, channelID *uuid.UUID) (bool, error) {
	checks := []struct{ s, c *uuid.UUID }{
		{serverID, channelID},
		{serverID, nil},
		{nil, nil},
	}

	for _, check := range checks {
		key := muteKey(userID, check.s, check.c)
		data, err := s.cache.Get(ctx, key)
		if err != nil {
			continue
		}

		var config models.MuteConfig
		if err := json.Unmarshal(data, &config); err != nil {
			continue
		}

		if config.Muted {
			return true, nil
		}
	}

	return false, nil
}

// --- Engagement Tracking ---

// TrackNotificationClick records a notification click for engagement tracking
func (s *SmartNotificationService) TrackNotificationClick(ctx context.Context, userID uuid.UUID, notificationID uuid.UUID) error {
	event := &models.NotificationClickEvent{
		NotificationID: notificationID,
		UserID:         userID,
		ClickedAt:      time.Now(),
	}

	// Update click count in cache
	clickKey := fmt.Sprintf("smart_notif:clicks:%s", userID.String())
	s.incrementCounter(ctx, clickKey, 24*time.Hour)

	// Mark notification as read via the original repo
	_ = s.notifRepo.MarkAsRead(ctx, notificationID, userID)

	s.eventBus.Publish("notification.clicked", event)

	return nil
}

// TrackSenderEngagement records an interaction between two users for sender importance scoring
func (s *SmartNotificationService) TrackSenderEngagement(ctx context.Context, recipientID, senderID uuid.UUID) error {
	key := fmt.Sprintf("smart_notif:engagement:%s:%s", recipientID.String(), senderID.String())
	s.incrementCounter(ctx, key, 7*24*time.Hour) // 7-day rolling window
	return nil
}

// GetUserEngagement returns engagement stats for a user
func (s *SmartNotificationService) GetUserEngagement(ctx context.Context, userID uuid.UUID) (*models.UserEngagement, error) {
	clickKey := fmt.Sprintf("smart_notif:clicks:%s", userID.String())
	receivedKey := fmt.Sprintf("smart_notif:received:%s", userID.String())
	dismissedKey := fmt.Sprintf("smart_notif:dismissed:%s", userID.String())

	clicks := s.getCounter(ctx, clickKey)
	received := s.getCounter(ctx, receivedKey)
	dismissed := s.getCounter(ctx, dismissedKey)

	var clickRate float64
	if received > 0 {
		clickRate = float64(clicks) / float64(received)
	}

	engagement := &models.UserEngagement{
		UserID:         userID,
		TotalReceived:  received,
		TotalClicked:   clicks,
		TotalDismissed: dismissed,
		ClickRate:      clickRate,
		UpdatedAt:      time.Now(),
	}

	// Get top channels from cache
	topChannelsKey := fmt.Sprintf("smart_notif:top_channels:%s", userID.String())
	if data, err := s.cache.Get(ctx, topChannelsKey); err == nil {
		var channels []uuid.UUID
		if json.Unmarshal(data, &channels) == nil {
			engagement.TopChannels = channels
		}
	}

	return engagement, nil
}

// TrackNotificationReceived increments the received counter for a user
func (s *SmartNotificationService) TrackNotificationReceived(ctx context.Context, userID uuid.UUID) error {
	key := fmt.Sprintf("smart_notif:received:%s", userID.String())
	s.incrementCounter(ctx, key, 24*time.Hour)
	return nil
}

// TrackNotificationDismissed increments the dismissed counter for a user
func (s *SmartNotificationService) TrackNotificationDismissed(ctx context.Context, userID uuid.UUID) error {
	key := fmt.Sprintf("smart_notif:dismissed:%s", userID.String())
	s.incrementCounter(ctx, key, 24*time.Hour)
	return nil
}

// --- Notification Routing ---

// RouteNotification determines how to deliver a notification based on priority and user preferences
func (s *SmartNotificationService) RouteNotification(ctx context.Context, userID uuid.UUID, smartNotif *models.SmartNotification) (*models.SmartNotification, error) {
	// Check if snoozed
	snoozed, until, _ := s.IsNotificationSnoozed(ctx, userID, smartNotif.ServerID, smartNotif.ChannelID)
	if snoozed {
		// Urgent notifications bypass snooze
		if smartNotif.Priority == models.NotificationPriorityUrgent {
			prefs := s.getUserPreferences(ctx, userID)
			if prefs.UrgentAlwaysDeliver {
				smartNotif.DeliveryMode = models.DeliveryImmediate
				return smartNotif, nil
			}
		}
		// Queue for delivery after snooze ends
		smartNotif.DeliveryMode = models.DeliveryBatched
		_ = until // snooze end time available for scheduling
		return smartNotif, nil
	}

	// Check if muted
	muted, _ := s.IsNotificationMuted(ctx, userID, smartNotif.ServerID, smartNotif.ChannelID)
	if muted {
		// Muted notifications are always batched (digests only)
		smartNotif.DeliveryMode = models.DeliveryBatched
		return smartNotif, nil
	}

	// Apply user preferences
	prefs := s.getUserPreferences(ctx, userID)
	if !prefs.Enabled {
		// Smart notifications disabled — deliver everything immediately
		smartNotif.DeliveryMode = models.DeliveryImmediate
		return smartNotif, nil
	}

	// Urgent always gets immediate delivery
	if smartNotif.Priority == models.NotificationPriorityUrgent && prefs.UrgentAlwaysDeliver {
		smartNotif.DeliveryMode = models.DeliveryImmediate
	}

	return smartNotif, nil
}

// GetUserPreferences returns the user's smart notification preferences
func (s *SmartNotificationService) GetUserPreferences(ctx context.Context, userID uuid.UUID) *models.SmartNotificationPreferences {
	return s.getUserPreferences(ctx, userID)
}

// UpdateUserPreferences saves the user's smart notification preferences
func (s *SmartNotificationService) UpdateUserPreferences(ctx context.Context, userID uuid.UUID, prefs *models.SmartNotificationPreferences) error {
	prefs.UserID = userID
	key := fmt.Sprintf("smart_notif:prefs:%s", userID.String())
	data, err := json.Marshal(prefs)
	if err != nil {
		return fmt.Errorf("failed to marshal preferences: %w", err)
	}
	return s.cache.Set(ctx, key, data, 0) // no expiry
}

// --- Digest Operations ---

// GetPendingDigestNotifications retrieves batched notifications not yet included in a digest
func (s *SmartNotificationService) GetPendingDigestNotifications(ctx context.Context, userID uuid.UUID) ([]models.SmartNotification, error) {
	key := fmt.Sprintf("smart_notif:pending_digest:%s", userID.String())
	data, err := s.cache.Get(ctx, key)
	if err != nil {
		return nil, nil // no pending notifications
	}

	var notifications []models.SmartNotification
	if err := json.Unmarshal(data, &notifications); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pending notifications: %w", err)
	}

	return notifications, nil
}

// AddToDigestQueue adds a notification to the digest queue
func (s *SmartNotificationService) AddToDigestQueue(ctx context.Context, userID uuid.UUID, notif *models.SmartNotification) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("smart_notif:pending_digest:%s", userID.String())

	var pending []models.SmartNotification
	if data, err := s.cache.Get(ctx, key); err == nil {
		_ = json.Unmarshal(data, &pending)
	}

	pending = append(pending, *notif)

	data, err := json.Marshal(pending)
	if err != nil {
		return fmt.Errorf("failed to marshal pending notifications: %w", err)
	}

	return s.cache.Set(ctx, key, data, 24*time.Hour)
}

// ClearDigestQueue clears the digest queue for a user
func (s *SmartNotificationService) ClearDigestQueue(ctx context.Context, userID uuid.UUID) error {
	key := fmt.Sprintf("smart_notif:pending_digest:%s", userID.String())
	return s.cache.Delete(ctx, key)
}

// CreateDigest creates a digest from pending notifications
func (s *SmartNotificationService) CreateDigest(ctx context.Context, userID uuid.UUID) (*models.NotificationDigest, error) {
	pending, err := s.GetPendingDigestNotifications(ctx, userID)
	if err != nil {
		return nil, err
	}

	if len(pending) == 0 {
		return nil, nil
	}

	ids := make([]uuid.UUID, len(pending))
	for i, n := range pending {
		ids[i] = n.ID
	}

	now := time.Now()
	digest := &models.NotificationDigest{
		ID:              uuid.New(),
		UserID:          userID,
		Title:           fmt.Sprintf("You have %d new notifications", len(pending)),
		Summary:         buildDigestSummary(pending),
		NotificationIDs: ids,
		Count:           len(pending),
		DeliveredAt:     &now,
		CreatedAt:       now,
	}

	// Store digest
	key := fmt.Sprintf("smart_notif:digest:%s:%s", userID.String(), digest.ID.String())
	data, err := json.Marshal(digest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal digest: %w", err)
	}

	if err := s.cache.Set(ctx, key, data, 7*24*time.Hour); err != nil {
		return nil, fmt.Errorf("failed to save digest: %w", err)
	}

	// Store digest ID in the user's digest list
	s.addToDigestList(ctx, userID, digest.ID)

	// Clear the pending queue
	_ = s.ClearDigestQueue(ctx, userID)

	s.eventBus.Publish("notification.digest_created", &SmartNotificationDigestEvent{
		UserID:   userID,
		DigestID: digest.ID,
		Count:    len(pending),
	})

	return digest, nil
}

// GetDigest retrieves a specific digest
func (s *SmartNotificationService) GetDigest(ctx context.Context, userID uuid.UUID, digestID uuid.UUID) (*models.NotificationDigest, error) {
	key := fmt.Sprintf("smart_notif:digest:%s:%s", userID.String(), digestID.String())
	data, err := s.cache.Get(ctx, key)
	if err != nil {
		return nil, ErrDigestNotFound
	}

	var digest models.NotificationDigest
	if err := json.Unmarshal(data, &digest); err != nil {
		return nil, fmt.Errorf("failed to unmarshal digest: %w", err)
	}

	if digest.UserID != userID {
		return nil, ErrDigestNotFound
	}

	return &digest, nil
}

// ListDigests returns digests for a user
func (s *SmartNotificationService) ListDigests(ctx context.Context, userID uuid.UUID, opts models.DigestListOptions) ([]models.NotificationDigest, error) {
	listKey := fmt.Sprintf("smart_notif:digest_list:%s", userID.String())
	data, err := s.cache.Get(ctx, listKey)
	if err != nil {
		return nil, nil
	}

	var digestIDs []uuid.UUID
	if err := json.Unmarshal(data, &digestIDs); err != nil {
		return nil, nil
	}

	var digests []models.NotificationDigest
	for _, id := range digestIDs {
		digest, err := s.GetDigest(ctx, userID, id)
		if err != nil {
			continue
		}

		if opts.Unread != nil {
			isUnread := digest.ReadAt == nil
			if *opts.Unread != isUnread {
				continue
			}
		}

		digests = append(digests, *digest)
	}

	// Apply pagination
	if opts.Offset > 0 && opts.Offset < len(digests) {
		digests = digests[opts.Offset:]
	} else if opts.Offset >= len(digests) {
		return nil, nil
	}

	if opts.Limit > 0 && opts.Limit < len(digests) {
		digests = digests[:opts.Limit]
	}

	return digests, nil
}

// MarkDigestRead marks a digest as read
func (s *SmartNotificationService) MarkDigestRead(ctx context.Context, userID uuid.UUID, digestID uuid.UUID) error {
	digest, err := s.GetDigest(ctx, userID, digestID)
	if err != nil {
		return err
	}

	now := time.Now()
	digest.ReadAt = &now

	key := fmt.Sprintf("smart_notif:digest:%s:%s", userID.String(), digestID.String())
	data, err := json.Marshal(digest)
	if err != nil {
		return fmt.Errorf("failed to marshal digest: %w", err)
	}

	return s.cache.Set(ctx, key, data, 7*24*time.Hour)
}

// --- Internal helpers ---

func (s *SmartNotificationService) getUserPreferences(ctx context.Context, userID uuid.UUID) *models.SmartNotificationPreferences {
	key := fmt.Sprintf("smart_notif:prefs:%s", userID.String())
	data, err := s.cache.Get(ctx, key)
	if err != nil {
		// Return defaults
		return &models.SmartNotificationPreferences{
			UserID:               userID,
			Enabled:              true,
			DigestEnabled:        true,
			DigestIntervalMins:   30,
			UrgentAlwaysDeliver:  true,
			ClickTrackingEnabled: true,
		}
	}

	var prefs models.SmartNotificationPreferences
	if err := json.Unmarshal(data, &prefs); err != nil {
		return &models.SmartNotificationPreferences{
			UserID:               userID,
			Enabled:              true,
			DigestEnabled:        true,
			DigestIntervalMins:   30,
			UrgentAlwaysDeliver:  true,
			ClickTrackingEnabled: true,
		}
	}

	return &prefs
}

func (s *SmartNotificationService) incrementCounter(ctx context.Context, key string, ttl time.Duration) {
	data, err := s.cache.Get(ctx, key)
	var count int
	if err == nil {
		_ = json.Unmarshal(data, &count)
	}
	count++
	newData, _ := json.Marshal(count)
	_ = s.cache.Set(ctx, key, newData, ttl)
}

func (s *SmartNotificationService) getCounter(ctx context.Context, key string) int {
	data, err := s.cache.Get(ctx, key)
	if err != nil {
		return 0
	}
	var count int
	if err := json.Unmarshal(data, &count); err != nil {
		return 0
	}
	return count
}

func (s *SmartNotificationService) addToDigestList(ctx context.Context, userID uuid.UUID, digestID uuid.UUID) {
	listKey := fmt.Sprintf("smart_notif:digest_list:%s", userID.String())

	var ids []uuid.UUID
	if data, err := s.cache.Get(ctx, listKey); err == nil {
		_ = json.Unmarshal(data, &ids)
	}

	// Prepend new ID and limit to 100
	ids = append([]uuid.UUID{digestID}, ids...)
	if len(ids) > 100 {
		ids = ids[:100]
	}

	data, _ := json.Marshal(ids)
	_ = s.cache.Set(ctx, listKey, data, 7*24*time.Hour)
}

func snoozeKey(userID uuid.UUID, serverID, channelID *uuid.UUID) string {
	key := fmt.Sprintf("smart_notif:snooze:%s", userID.String())
	if serverID != nil {
		key += ":" + serverID.String()
	}
	if channelID != nil {
		key += ":" + channelID.String()
	}
	return key
}

func muteKey(userID uuid.UUID, serverID, channelID *uuid.UUID) string {
	key := fmt.Sprintf("smart_notif:mute:%s", userID.String())
	if serverID != nil {
		key += ":" + serverID.String()
	}
	if channelID != nil {
		key += ":" + channelID.String()
	}
	return key
}

func classifyCategory(notifType models.NotificationType) models.NotificationCategory {
	switch notifType {
	case models.NotificationTypeMention:
		return models.NotifCategoryMention
	case models.NotificationTypeDirectMessage:
		return models.NotifCategoryDirectMessage
	case models.NotificationTypeReply:
		return models.NotifCategoryReply
	case models.NotificationTypeReaction:
		return models.NotifCategoryReaction
	case models.NotificationTypeFriendRequest, models.NotificationTypeFriendAccept:
		return models.NotifCategorySocial
	case models.NotificationTypeServerInvite, models.NotificationTypeServerJoin:
		return models.NotifCategoryServerEvent
	case models.NotificationTypeEventInvite, models.NotificationTypeEventRSVP,
		models.NotificationTypeEventReminder, models.NotificationTypeEventStart:
		return models.NotifCategoryServerEvent
	case models.NotificationTypeSystem:
		return models.NotifCategorySystem
	default:
		return models.NotifCategorySystem
	}
}

func buildDigestSummary(notifications []models.SmartNotification) string {
	categoryCounts := make(map[models.NotificationCategory]int)
	for _, n := range notifications {
		categoryCounts[n.Category]++
	}

	var summary strings.Builder
	first := true
	for cat, count := range categoryCounts {
		if !first {
			summary.WriteString(", ")
		}
		first = false
		fmt.Fprintf(&summary, "%d %s", count, cat)
	}

	return summary.String()
}

// --- Events ---

// SmartNotificationSnoozedEvent is emitted when notifications are snoozed
type SmartNotificationSnoozedEvent struct {
	UserID uuid.UUID
	Until  time.Time
}

// SmartNotificationUnsnoozedEvent is emitted when a snooze is removed
type SmartNotificationUnsnoozedEvent struct {
	UserID uuid.UUID
}

// SmartNotificationMuteEvent is emitted when mute state changes
type SmartNotificationMuteEvent struct {
	UserID uuid.UUID
	Muted  bool
}

// SmartNotificationDigestEvent is emitted when a digest is created
type SmartNotificationDigestEvent struct {
	UserID   uuid.UUID
	DigestID uuid.UUID
	Count    int
}

// Digest-related errors are defined in errors.go:
// - ErrDigestNotFound
// - ErrDigestDisabled
// - ErrInvalidFrequency
// - ErrInvalidTimezone

// DigestRepository defines the interface for digest data access
type DigestRepository interface {
	// Preferences
	GetPreferences(ctx context.Context, userID uuid.UUID) (*models.DigestPreferences, error)
	CreatePreferences(ctx context.Context, prefs *models.DigestPreferences) error
	UpdatePreferences(ctx context.Context, prefs *models.DigestPreferences) error
	UpsertPreferences(ctx context.Context, prefs *models.DigestPreferences) error

	// Channel preferences
	GetChannelPreference(ctx context.Context, userID, channelID uuid.UUID) (*models.DigestChannelPreference, error)
	GetChannelPreferences(ctx context.Context, userID uuid.UUID) ([]models.DigestChannelPreference, error)
	UpsertChannelPreference(ctx context.Context, pref *models.DigestChannelPreference) error
	DeleteChannelPreference(ctx context.Context, userID, channelID uuid.UUID) error

	// Server preferences
	GetServerPreference(ctx context.Context, userID, serverID uuid.UUID) (*models.DigestServerPreference, error)
	GetServerPreferences(ctx context.Context, userID uuid.UUID) ([]models.DigestServerPreference, error)
	UpsertServerPreference(ctx context.Context, pref *models.DigestServerPreference) error
	DeleteServerPreference(ctx context.Context, userID, serverID uuid.UUID) error

	// Queue
	QueueMessage(ctx context.Context, item *models.DigestQueueItem) error
	GetQueuedItems(ctx context.Context, userID uuid.UUID, before time.Time) ([]models.DigestQueueItem, error)
	GetQueuePreview(ctx context.Context, userID uuid.UUID) (*models.DigestPreview, error)
	DeleteQueuedItems(ctx context.Context, userID uuid.UUID, before time.Time) (int64, error)
	ClearQueue(ctx context.Context, userID uuid.UUID) (int64, error)

	// History
	CreateHistory(ctx context.Context, history *models.DigestHistory) error
	UpdateHistoryStatus(ctx context.Context, id uuid.UUID, status models.DigestStatus, errorMessage *string) error
	GetHistory(ctx context.Context, userID uuid.UUID, opts models.DigestHistoryListOptions) ([]models.DigestHistory, error)
	GetHistoryByID(ctx context.Context, id uuid.UUID) (*models.DigestHistory, error)
	GetLastDigest(ctx context.Context, userID uuid.UUID) (*models.DigestHistory, error)

	// Scheduling
	GetUsersForDigest(ctx context.Context, frequency models.DigestFrequency, hour int, day int) ([]uuid.UUID, error)
	GetPendingDigests(ctx context.Context, limit int) ([]models.DigestHistory, error)
}

// DigestServerRepo interface for digest (subset of ServerRepository)
type DigestServerRepo interface {
	GetByID(ctx context.Context, id uuid.UUID) (*models.Server, error)
}

// DigestService handles digest notification business logic
type DigestService struct {
	repo       DigestRepository
	serverRepo DigestServerRepo
	eventBus   EventBus
	mu         sync.RWMutex
	running    bool
	stopCh     chan struct{}
}

// NewDigestService creates a new digest service
func NewDigestService(repo DigestRepository, serverRepo DigestServerRepo, eventBus EventBus) *DigestService {
	return &DigestService{
		repo:       repo,
		serverRepo: serverRepo,
		eventBus:   eventBus,
		stopCh:     make(chan struct{}),
	}
}

// --- Preferences Management ---

// GetPreferences retrieves or creates default digest preferences for a user
func (s *DigestService) GetPreferences(ctx context.Context, userID uuid.UUID) (*models.DigestPreferences, error) {
	prefs, err := s.repo.GetPreferences(ctx, userID)
	if err != nil {
		return nil, err
	}
	if prefs == nil {
		// Return default preferences (not persisted until user modifies them)
		return models.DefaultDigestPreferences(userID), nil
	}
	return prefs, nil
}

// UpdatePreferences updates digest preferences for a user
func (s *DigestService) UpdatePreferences(ctx context.Context, userID uuid.UUID, req *models.UpdateDigestPreferencesRequest) (*models.DigestPreferences, error) {
	// Get existing preferences or create defaults
	prefs, err := s.repo.GetPreferences(ctx, userID)
	if err != nil {
		return nil, err
	}
	if prefs == nil {
		prefs = models.DefaultDigestPreferences(userID)
	}

	// Apply updates
	if req.Enabled != nil {
		prefs.Enabled = *req.Enabled
	}
	if req.Frequency != nil {
		if !models.ValidateFrequency(*req.Frequency) {
			return nil, ErrInvalidFrequency
		}
		prefs.Frequency = *req.Frequency
	}
	if req.PreferredHour != nil {
		if *req.PreferredHour < 0 || *req.PreferredHour > 23 {
			return nil, fmt.Errorf("preferred_hour must be between 0 and 23")
		}
		prefs.PreferredHour = *req.PreferredHour
	}
	if req.PreferredDay != nil {
		if *req.PreferredDay < 0 || *req.PreferredDay > 6 {
			return nil, fmt.Errorf("preferred_day must be between 0 and 6")
		}
		prefs.PreferredDay = *req.PreferredDay
	}
	if req.AggregationMode != nil {
		if !models.ValidateAggregationMode(*req.AggregationMode) {
			return nil, fmt.Errorf("invalid aggregation_mode")
		}
		prefs.AggregationMode = *req.AggregationMode
	}
	if req.MaxMessagesPerSource != nil {
		if *req.MaxMessagesPerSource < 1 || *req.MaxMessagesPerSource > 200 {
			return nil, fmt.Errorf("max_messages_per_source must be between 1 and 200")
		}
		prefs.MaxMessagesPerSource = *req.MaxMessagesPerSource
	}
	if req.MutedChannelsOnly != nil {
		prefs.MutedChannelsOnly = *req.MutedChannelsOnly
	}
	if req.Timezone != nil {
		// Validate timezone
		_, err := time.LoadLocation(*req.Timezone)
		if err != nil {
			return nil, ErrInvalidTimezone
		}
		prefs.Timezone = *req.Timezone
	}

	// Upsert preferences
	if err := s.repo.UpsertPreferences(ctx, prefs); err != nil {
		return nil, err
	}

	// Emit event
	s.eventBus.Publish("digest.preferences_updated", &DigestPreferencesUpdatedEvent{
		UserID:      userID,
		Preferences: prefs,
	})

	return prefs, nil
}

// --- Channel Preferences ---

// GetChannelPreference retrieves channel-specific digest preference
func (s *DigestService) GetChannelPreference(ctx context.Context, userID, channelID uuid.UUID) (*models.DigestChannelPreference, error) {
	pref, err := s.repo.GetChannelPreference(ctx, userID, channelID)
	if err != nil {
		return nil, err
	}
	if pref == nil {
		// Return default (inherit)
		return &models.DigestChannelPreference{
			UserID:     userID,
			ChannelID:  channelID,
			DigestMode: models.DigestModeInherit,
		}, nil
	}
	return pref, nil
}

// GetChannelPreferences retrieves all channel-specific preferences for a user
func (s *DigestService) GetChannelPreferences(ctx context.Context, userID uuid.UUID) ([]models.DigestChannelPreference, error) {
	return s.repo.GetChannelPreferences(ctx, userID)
}

// UpdateChannelPreference updates a channel-specific digest preference
func (s *DigestService) UpdateChannelPreference(ctx context.Context, userID, channelID uuid.UUID, mode models.DigestMode) error {
	if !models.ValidateDigestMode(mode) {
		return fmt.Errorf("invalid digest mode: %s", mode)
	}

	// If setting to inherit, delete the preference
	if mode == models.DigestModeInherit {
		return s.repo.DeleteChannelPreference(ctx, userID, channelID)
	}

	pref := &models.DigestChannelPreference{
		UserID:     userID,
		ChannelID:  channelID,
		DigestMode: mode,
	}
	return s.repo.UpsertChannelPreference(ctx, pref)
}

// --- Server Preferences ---

// GetServerPreference retrieves server-specific digest preference
func (s *DigestService) GetServerPreference(ctx context.Context, userID, serverID uuid.UUID) (*models.DigestServerPreference, error) {
	pref, err := s.repo.GetServerPreference(ctx, userID, serverID)
	if err != nil {
		return nil, err
	}
	if pref == nil {
		return &models.DigestServerPreference{
			UserID:     userID,
			ServerID:   serverID,
			DigestMode: models.DigestModeInherit,
		}, nil
	}
	return pref, nil
}

// GetServerPreferences retrieves all server-specific preferences for a user
func (s *DigestService) GetServerPreferences(ctx context.Context, userID uuid.UUID) ([]models.DigestServerPreference, error) {
	return s.repo.GetServerPreferences(ctx, userID)
}

// UpdateServerPreference updates a server-specific digest preference
func (s *DigestService) UpdateServerPreference(ctx context.Context, userID, serverID uuid.UUID, mode models.DigestMode) error {
	if mode == models.DigestModeImmediate {
		return fmt.Errorf("immediate mode is only available for channels")
	}
	if !models.ValidateDigestMode(mode) {
		return fmt.Errorf("invalid digest mode: %s", mode)
	}

	if mode == models.DigestModeInherit {
		return s.repo.DeleteServerPreference(ctx, userID, serverID)
	}

	pref := &models.DigestServerPreference{
		UserID:     userID,
		ServerID:   serverID,
		DigestMode: mode,
	}
	return s.repo.UpsertServerPreference(ctx, pref)
}

// --- Queue Management ---

// QueueNotification adds a notification to the digest queue
func (s *DigestService) QueueNotification(ctx context.Context, userID uuid.UUID, notification *models.Notification, messageContent, authorName string, messageCreatedAt time.Time) error {
	// Check if digest is enabled for this user
	prefs, err := s.repo.GetPreferences(ctx, userID)
	if err != nil {
		return err
	}
	if prefs == nil || !prefs.Enabled {
		return ErrDigestDisabled
	}

	// Determine digest period based on frequency
	digestPeriod := s.calculateDigestPeriod(prefs.Frequency, time.Now())

	item := &models.DigestQueueItem{
		UserID:            userID,
		ServerID:          notification.ServerID,
		ChannelID:         notification.ChannelID,
		MessageID:         notification.MessageID,
		MessageContent:    messageContent,
		MessageAuthorID:   notification.ActorID,
		MessageAuthorName: authorName,
		MessageCreatedAt:  messageCreatedAt,
		IsMention:         notification.Type == models.NotificationTypeMention,
		NotificationType:  notification.Type,
		DigestPeriod:      digestPeriod,
	}

	return s.repo.QueueMessage(ctx, item)
}

// GetDigestPreview returns a preview of pending digest items
func (s *DigestService) GetDigestPreview(ctx context.Context, userID uuid.UUID) (*models.DigestPreview, error) {
	prefs, err := s.GetPreferences(ctx, userID)
	if err != nil {
		return nil, err
	}

	preview, err := s.repo.GetQueuePreview(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Calculate next digest time
	preview.NextDigestAt = s.calculateNextDigestTime(prefs)

	return preview, nil
}

// ClearDigestQueue clears all pending digest items for a user
func (s *DigestService) ClearDigestQueue(ctx context.Context, userID uuid.UUID) (int64, error) {
	return s.repo.ClearQueue(ctx, userID)
}

// --- History ---

// GetDigestHistory retrieves digest history for a user
func (s *DigestService) GetDigestHistory(ctx context.Context, userID uuid.UUID, opts models.DigestHistoryListOptions) ([]models.DigestHistory, error) {
	return s.repo.GetHistory(ctx, userID, opts)
}

// GetDigestByID retrieves a specific digest
func (s *DigestService) GetDigestByID(ctx context.Context, userID uuid.UUID, digestID uuid.UUID) (*models.DigestHistory, error) {
	history, err := s.repo.GetHistoryByID(ctx, digestID)
	if err != nil {
		return nil, err
	}
	if history == nil || history.UserID != userID {
		return nil, ErrDigestNotFound
	}
	return history, nil
}

// --- Digest Generation ---

// GenerateDigest generates a digest for a user
func (s *DigestService) GenerateDigest(ctx context.Context, userID uuid.UUID) (*models.DigestHistory, error) {
	prefs, err := s.repo.GetPreferences(ctx, userID)
	if err != nil {
		return nil, err
	}
	if prefs == nil || !prefs.Enabled {
		return nil, ErrDigestDisabled
	}

	now := time.Now()
	periodEnd := now
	periodStart := s.calculatePeriodStart(prefs.Frequency, now)

	// Get queued items
	items, err := s.repo.GetQueuedItems(ctx, userID, now)
	if err != nil {
		return nil, err
	}

	if len(items) == 0 {
		// Nothing to send, create a skipped entry
		history := &models.DigestHistory{
			UserID:      userID,
			PeriodStart: periodStart,
			PeriodEnd:   periodEnd,
			Frequency:   prefs.Frequency,
			Status:      models.DigestStatusSkipped,
			ContentJSON: "{}",
		}
		if err := s.repo.CreateHistory(ctx, history); err != nil {
			return nil, err
		}
		return history, nil
	}

	// Generate digest content
	content := s.generateDigestContent(items, prefs, periodStart, periodEnd)
	contentJSON, err := content.ToJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize digest content: %w", err)
	}

	// Create history entry
	history := &models.DigestHistory{
		UserID:           userID,
		PeriodStart:      periodStart,
		PeriodEnd:        periodEnd,
		Frequency:        prefs.Frequency,
		TotalMessages:    content.TotalStats.MessageCount,
		TotalMentions:    content.TotalStats.MentionCount,
		ServersIncluded:  len(content.Servers),
		ChannelsIncluded: s.countChannels(content),
		ContentJSON:      contentJSON,
		Status:           models.DigestStatusPending,
	}

	if err := s.repo.CreateHistory(ctx, history); err != nil {
		return nil, err
	}

	// Clear processed queue items
	if _, err := s.repo.DeleteQueuedItems(ctx, userID, now); err != nil {
		// Log but don't fail - digest is already created
		log.Printf("failed to clear digest queue: %v", err)
	}

	// Emit event for delivery
	s.eventBus.Publish("digest.generated", &DigestGeneratedEvent{
		UserID:  userID,
		Digest:  history,
		Content: content,
	})

	return history, nil
}

// generateDigestContent creates the structured digest content from queue items
func (s *DigestService) generateDigestContent(items []models.DigestQueueItem, prefs *models.DigestPreferences, start, end time.Time) *models.DigestContent {
	content := &models.DigestContent{
		Period: models.DigestPeriodInfo{
			Start:     start,
			End:       end,
			Frequency: prefs.Frequency,
		},
		Servers:    []models.DigestServerSummary{},
		DMChannels: []models.DigestChannelSummary{},
		TotalStats: models.DigestStats{},
	}

	// Group by server, then by channel
	serverMap := make(map[uuid.UUID]*models.DigestServerSummary)
	dmChannelMap := make(map[uuid.UUID]*models.DigestChannelSummary)

	for _, item := range items {
		content.TotalStats.MessageCount++
		if item.IsMention {
			content.TotalStats.MentionCount++
		}

		msgSummary := models.DigestMessageSummary{
			MessageID:  item.MessageID,
			AuthorID:   item.MessageAuthorID,
			AuthorName: item.MessageAuthorName,
			Content:    truncateContent(item.MessageContent, 200),
			IsMention:  item.IsMention,
			CreatedAt:  item.MessageCreatedAt,
		}

		if item.ServerID == nil {
			// DM channel
			channelID := uuid.Nil
			if item.ChannelID != nil {
				channelID = *item.ChannelID
			}
			if _, exists := dmChannelMap[channelID]; !exists {
				dmChannelMap[channelID] = &models.DigestChannelSummary{
					ChannelID:   channelID,
					ChannelName: "Direct Message",
					Messages:    []models.DigestMessageSummary{},
					Stats:       models.DigestStats{},
				}
			}
			ch := dmChannelMap[channelID]
			if len(ch.Messages) < prefs.MaxMessagesPerSource {
				ch.Messages = append(ch.Messages, msgSummary)
			}
			ch.Stats.MessageCount++
			if item.IsMention {
				ch.Stats.MentionCount++
			}
		} else {
			// Server channel
			serverID := *item.ServerID
			if _, exists := serverMap[serverID]; !exists {
				serverMap[serverID] = &models.DigestServerSummary{
					ServerID:   serverID,
					ServerName: "Server", // Would need to fetch actual name
					Channels:   []models.DigestChannelSummary{},
					Stats:      models.DigestStats{},
				}
			}
			srv := serverMap[serverID]
			srv.Stats.MessageCount++
			if item.IsMention {
				srv.Stats.MentionCount++
			}

			// Find or create channel in server
			channelID := uuid.Nil
			if item.ChannelID != nil {
				channelID = *item.ChannelID
			}
			var channel *models.DigestChannelSummary
			for i := range srv.Channels {
				if srv.Channels[i].ChannelID == channelID {
					channel = &srv.Channels[i]
					break
				}
			}
			if channel == nil {
				srv.Channels = append(srv.Channels, models.DigestChannelSummary{
					ChannelID:   channelID,
					ChannelName: "Channel",
					Messages:    []models.DigestMessageSummary{},
					Stats:       models.DigestStats{},
				})
				channel = &srv.Channels[len(srv.Channels)-1]
			}

			if len(channel.Messages) < prefs.MaxMessagesPerSource {
				channel.Messages = append(channel.Messages, msgSummary)
			}
			channel.Stats.MessageCount++
			if item.IsMention {
				channel.Stats.MentionCount++
			}
		}
	}

	// Convert maps to slices
	for _, srv := range serverMap {
		content.Servers = append(content.Servers, *srv)
	}
	for _, ch := range dmChannelMap {
		content.DMChannels = append(content.DMChannels, *ch)
	}

	// Sort by message count (most active first)
	sort.Slice(content.Servers, func(i, j int) bool {
		return content.Servers[i].Stats.MessageCount > content.Servers[j].Stats.MessageCount
	})

	return content
}

// countChannels counts total channels in digest content
func (s *DigestService) countChannels(content *models.DigestContent) int {
	count := len(content.DMChannels)
	for _, srv := range content.Servers {
		count += len(srv.Channels)
	}
	return count
}

// --- Scheduling ---

// StartScheduler starts the digest scheduling goroutine
func (s *DigestService) StartScheduler(ctx context.Context) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.stopCh = make(chan struct{})
	s.mu.Unlock()

	go s.runScheduler(ctx)
}

// StopScheduler stops the digest scheduler
func (s *DigestService) StopScheduler() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		close(s.stopCh)
		s.running = false
	}
}

// runScheduler runs the scheduling loop
func (s *DigestService) runScheduler(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.processScheduledDigests(ctx)
		}
	}
}

// processScheduledDigests processes digests that should be sent now
func (s *DigestService) processScheduledDigests(ctx context.Context) {
	now := time.Now().UTC()
	hour := now.Hour()
	day := int(now.Weekday())

	// Process hourly digests
	s.processDigestsForFrequency(ctx, models.DigestFrequencyHourly, hour, day)

	// Process daily digests (at the preferred hour)
	s.processDigestsForFrequency(ctx, models.DigestFrequencyDaily, hour, day)

	// Process weekly digests (at the preferred hour on the preferred day)
	s.processDigestsForFrequency(ctx, models.DigestFrequencyWeekly, hour, day)
}

// processDigestsForFrequency processes digests for a specific frequency
func (s *DigestService) processDigestsForFrequency(ctx context.Context, frequency models.DigestFrequency, hour, day int) {
	userIDs, err := s.repo.GetUsersForDigest(ctx, frequency, hour, day)
	if err != nil {
		log.Printf("failed to get users for digest: %v", err)
		return
	}

	for _, userID := range userIDs {
		_, err := s.GenerateDigest(ctx, userID)
		if err != nil && !errors.Is(err, ErrDigestDisabled) && !errors.Is(err, sql.ErrNoRows) {
			log.Printf("failed to generate digest for user %s: %v", userID, err)
		}
	}
}

// --- Helper Functions ---

// calculateDigestPeriod calculates the digest period a message belongs to
func (s *DigestService) calculateDigestPeriod(frequency models.DigestFrequency, t time.Time) time.Time {
	switch frequency {
	case models.DigestFrequencyHourly:
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location())
	case models.DigestFrequencyDaily:
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	case models.DigestFrequencyWeekly:
		// Start of week (Sunday)
		weekday := int(t.Weekday())
		return time.Date(t.Year(), t.Month(), t.Day()-weekday, 0, 0, 0, 0, t.Location())
	default:
		return t
	}
}

// calculatePeriodStart calculates the start of the current digest period
func (s *DigestService) calculatePeriodStart(frequency models.DigestFrequency, t time.Time) time.Time {
	switch frequency {
	case models.DigestFrequencyHourly:
		return t.Add(-time.Hour)
	case models.DigestFrequencyDaily:
		return t.Add(-24 * time.Hour)
	case models.DigestFrequencyWeekly:
		return t.Add(-7 * 24 * time.Hour)
	default:
		return t.Add(-24 * time.Hour)
	}
}

// calculateNextDigestTime calculates when the next digest will be sent
func (s *DigestService) calculateNextDigestTime(prefs *models.DigestPreferences) time.Time {
	now := time.Now()
	loc, err := time.LoadLocation(prefs.Timezone)
	if err != nil {
		loc = time.UTC
	}
	localNow := now.In(loc)

	switch prefs.Frequency {
	case models.DigestFrequencyHourly:
		return localNow.Add(time.Hour).Truncate(time.Hour)
	case models.DigestFrequencyDaily:
		nextTime := time.Date(localNow.Year(), localNow.Month(), localNow.Day(),
			prefs.PreferredHour, 0, 0, 0, loc)
		if nextTime.Before(localNow) {
			nextTime = nextTime.Add(24 * time.Hour)
		}
		return nextTime
	case models.DigestFrequencyWeekly:
		daysUntil := (prefs.PreferredDay - int(localNow.Weekday()) + 7) % 7
		if daysUntil == 0 && localNow.Hour() >= prefs.PreferredHour {
			daysUntil = 7
		}
		nextTime := time.Date(localNow.Year(), localNow.Month(), localNow.Day()+daysUntil,
			prefs.PreferredHour, 0, 0, 0, loc)
		return nextTime
	default:
		return localNow.Add(24 * time.Hour)
	}
}

// truncateContent truncates content to a maximum length
func truncateContent(content string, maxLen int) string {
	if len(content) <= maxLen {
		return content
	}
	return content[:maxLen-3] + "..."
}

// --- Events ---

// DigestPreferencesUpdatedEvent is emitted when preferences are updated
type DigestPreferencesUpdatedEvent struct {
	UserID      uuid.UUID
	Preferences *models.DigestPreferences
}

// DigestGeneratedEvent is emitted when a digest is generated
type DigestGeneratedEvent struct {
	UserID  uuid.UUID
	Digest  *models.DigestHistory
	Content *models.DigestContent
}

// DigestSentEvent is emitted when a digest is successfully sent
type DigestSentEvent struct {
	UserID   uuid.UUID
	DigestID uuid.UUID
}

// DigestFailedEvent is emitted when digest sending fails
type DigestFailedEvent struct {
	UserID   uuid.UUID
	DigestID uuid.UUID
	Error    string
}

// DigestWorker runs in the background generating notification digests
type DigestWorker struct {
	smartService *SmartNotificationService
	cache        SmartNotificationCache
	eventBus     EventBus
	interval     time.Duration
	stopCh       chan struct{}
	wg           sync.WaitGroup
	running      bool
	mu           sync.Mutex
}

// NewDigestWorker creates a new background digest worker
func NewDigestWorker(
	smartService *SmartNotificationService,
	cache SmartNotificationCache,
	eventBus EventBus,
	interval time.Duration,
) *DigestWorker {
	if interval == 0 {
		interval = 5 * time.Minute
	}
	return &DigestWorker{
		smartService: smartService,
		cache:        cache,
		eventBus:     eventBus,
		interval:     interval,
		stopCh:       make(chan struct{}),
	}
}

// Start begins the digest worker loop
func (w *DigestWorker) Start() {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return
	}
	w.running = true
	w.mu.Unlock()

	w.wg.Add(1)
	go w.run()
}

// Stop gracefully stops the digest worker
func (w *DigestWorker) Stop() {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return
	}
	w.running = false
	w.mu.Unlock()

	close(w.stopCh)
	w.wg.Wait()
}

// IsRunning returns whether the worker is currently running
func (w *DigestWorker) IsRunning() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.running
}

func (w *DigestWorker) run() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.processDigests()
		}
	}
}

// processDigests checks all users with pending digests and creates digests for eligible ones
func (w *DigestWorker) processDigests() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	userIDs, err := w.getEligibleUsers(ctx)
	if err != nil {
		log.Printf("digest worker: failed to get eligible users: %v", err)
		return
	}

	for _, userID := range userIDs {
		if err := w.processUserDigest(ctx, userID); err != nil {
			log.Printf("digest worker: failed to process digest for user %s: %v", userID, err)
		}
	}
}

// processUserDigest creates a digest for a single user if they have enough pending notifications
func (w *DigestWorker) processUserDigest(ctx context.Context, userID uuid.UUID) error {
	prefs := w.smartService.GetUserPreferences(ctx, userID)
	if !prefs.DigestEnabled {
		return nil
	}

	pending, err := w.smartService.GetPendingDigestNotifications(ctx, userID)
	if err != nil {
		return err
	}

	if len(pending) == 0 {
		return nil
	}

	// Check if enough time has elapsed since last digest
	lastDigestKey := fmt.Sprintf("smart_notif:last_digest:%s", userID.String())
	if data, err := w.cache.Get(ctx, lastDigestKey); err == nil {
		var lastDigestTime time.Time
		if json.Unmarshal(data, &lastDigestTime) == nil {
			interval := time.Duration(prefs.DigestIntervalMins) * time.Minute
			if time.Since(lastDigestTime) < interval {
				return nil // not time yet
			}
		}
	}

	// Create the digest
	digest, err := w.smartService.CreateDigest(ctx, userID)
	if err != nil {
		return err
	}

	if digest == nil {
		return nil
	}

	// Record the digest timestamp
	now := time.Now()
	data, _ := json.Marshal(now)
	_ = w.cache.Set(ctx, lastDigestKey, data, 24*time.Hour)

	return nil
}

// getEligibleUsers returns user IDs that have pending digest notifications
func (w *DigestWorker) getEligibleUsers(ctx context.Context) ([]uuid.UUID, error) {
	key := "smart_notif:digest_eligible_users"
	data, err := w.cache.Get(ctx, key)
	if err != nil {
		return nil, nil // no eligible users
	}

	var userIDs []uuid.UUID
	if err := json.Unmarshal(data, &userIDs); err != nil {
		return nil, nil
	}

	return userIDs, nil
}

// RegisterEligibleUser adds a user to the eligible users set for digest processing
func (w *DigestWorker) RegisterEligibleUser(ctx context.Context, userID uuid.UUID) error {
	key := "smart_notif:digest_eligible_users"

	var userIDs []uuid.UUID
	if data, err := w.cache.Get(ctx, key); err == nil {
		_ = json.Unmarshal(data, &userIDs)
	}

	// Check if already registered
	for _, id := range userIDs {
		if id == userID {
			return nil
		}
	}

	userIDs = append(userIDs, userID)
	data, _ := json.Marshal(userIDs)
	return w.cache.Set(ctx, key, data, 24*time.Hour)
}

// UnregisterEligibleUser removes a user from the eligible users set
func (w *DigestWorker) UnregisterEligibleUser(ctx context.Context, userID uuid.UUID) error {
	key := "smart_notif:digest_eligible_users"

	var userIDs []uuid.UUID
	if data, err := w.cache.Get(ctx, key); err == nil {
		_ = json.Unmarshal(data, &userIDs)
	}

	filtered := make([]uuid.UUID, 0, len(userIDs))
	for _, id := range userIDs {
		if id != userID {
			filtered = append(filtered, id)
		}
	}

	data, _ := json.Marshal(filtered)
	return w.cache.Set(ctx, key, data, 24*time.Hour)
}

// Tick forces an immediate digest processing cycle (useful for testing)
func (w *DigestWorker) Tick() {
	w.processDigests()
}
