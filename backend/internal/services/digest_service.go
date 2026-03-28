package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"hearth/internal/models"
)

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
