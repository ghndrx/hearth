package services

import (
	"context"
	"encoding/json"
	"fmt"
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

	summary := ""
	for cat, count := range categoryCounts {
		if summary != "" {
			summary += ", "
		}
		summary += fmt.Sprintf("%d %s", count, cat)
	}

	return summary
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
