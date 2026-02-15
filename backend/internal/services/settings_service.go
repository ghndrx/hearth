package services

import (
	"context"
	"time"

	"github.com/google/uuid"

	"hearth/internal/models"
)

// SettingsRepository defines the interface for settings data access
type SettingsRepository interface {
	Get(ctx context.Context, userID uuid.UUID) (*models.UserSettings, error)
	Create(ctx context.Context, settings *models.UserSettings) error
	Update(ctx context.Context, settings *models.UserSettings) error
	Upsert(ctx context.Context, settings *models.UserSettings) error
	Delete(ctx context.Context, userID uuid.UUID) error
}

// SettingsService handles user settings business logic
type SettingsService struct {
	repo     SettingsRepository
	eventBus EventBus
}

// NewSettingsService creates a new settings service
func NewSettingsService(repo SettingsRepository, eventBus EventBus) *SettingsService {
	return &SettingsService{
		repo:     repo,
		eventBus: eventBus,
	}
}

// GetSettings retrieves user settings, creating defaults if they don't exist
func (s *SettingsService) GetSettings(ctx context.Context, userID uuid.UUID) (*models.UserSettings, error) {
	settings, err := s.repo.Get(ctx, userID)
	if err != nil {
		return nil, err
	}

	// If no settings exist, create and return defaults
	if settings == nil {
		settings = models.DefaultUserSettings(userID)
		if err := s.repo.Create(ctx, settings); err != nil {
			// If creation fails (race condition), try to get again
			settings, err = s.repo.Get(ctx, userID)
			if err != nil {
				return nil, err
			}
			if settings == nil {
				// Still nil, return defaults without persisting
				return models.DefaultUserSettings(userID), nil
			}
		}
	}

	return settings, nil
}

// UpdateSettings updates user settings with the provided values
func (s *SettingsService) UpdateSettings(ctx context.Context, userID uuid.UUID, updates *models.UpdateUserSettingsRequest) (*models.UserSettings, error) {
	// Get current settings (or defaults)
	settings, err := s.GetSettings(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Apply theme updates
	if updates.Theme != nil {
		settings.Theme = *updates.Theme
	}
	if updates.MessageDisplay != nil {
		settings.MessageDisplay = *updates.MessageDisplay
	}
	if updates.CompactMode != nil {
		settings.CompactMode = *updates.CompactMode
	}
	if updates.DeveloperMode != nil {
		settings.DeveloperMode = *updates.DeveloperMode
	}

	// Apply display updates
	if updates.InlineEmbeds != nil {
		settings.InlineEmbeds = *updates.InlineEmbeds
	}
	if updates.InlineAttachments != nil {
		settings.InlineAttachments = *updates.InlineAttachments
	}
	if updates.RenderReactions != nil {
		settings.RenderReactions = *updates.RenderReactions
	}
	if updates.AnimateEmoji != nil {
		settings.AnimateEmoji = *updates.AnimateEmoji
	}
	if updates.EnableTTS != nil {
		settings.EnableTTS = *updates.EnableTTS
	}
	if updates.CustomCSS != nil {
		settings.CustomCSS = updates.CustomCSS
	}

	// Apply notification updates
	if updates.NotificationsEnabled != nil {
		settings.NotificationsEnabled = *updates.NotificationsEnabled
	}
	if updates.NotificationsSound != nil {
		settings.NotificationsSound = *updates.NotificationsSound
	}
	if updates.NotificationsDesktop != nil {
		settings.NotificationsDesktop = *updates.NotificationsDesktop
	}
	if updates.NotificationsMentionsOnly != nil {
		settings.NotificationsMentionsOnly = *updates.NotificationsMentionsOnly
	}
	if updates.NotificationsDM != nil {
		settings.NotificationsDM = *updates.NotificationsDM
	}
	if updates.NotificationsServerDefaults != nil {
		settings.NotificationsServerDefaults = *updates.NotificationsServerDefaults
	}

	// Apply privacy updates
	if updates.PrivacyDMFromServers != nil {
		settings.PrivacyDMFromServers = *updates.PrivacyDMFromServers
	}
	if updates.PrivacyDMFromFriendsOnly != nil {
		settings.PrivacyDMFromFriendsOnly = *updates.PrivacyDMFromFriendsOnly
	}
	if updates.PrivacyShowActivity != nil {
		settings.PrivacyShowActivity = *updates.PrivacyShowActivity
	}
	if updates.PrivacyFriendRequestsAll != nil {
		settings.PrivacyFriendRequestsAll = *updates.PrivacyFriendRequestsAll
	}
	if updates.PrivacyReadReceipts != nil {
		settings.PrivacyReadReceipts = *updates.PrivacyReadReceipts
	}

	// Apply locale update
	if updates.Locale != nil {
		settings.Locale = *updates.Locale
	}

	settings.UpdatedAt = time.Now()

	// Upsert the settings
	if err := s.repo.Upsert(ctx, settings); err != nil {
		return nil, err
	}

	// Emit event
	s.eventBus.Publish("user.settings_updated", &UserSettingsUpdatedEvent{
		UserID:    userID,
		Settings:  settings,
		UpdatedAt: settings.UpdatedAt,
	})

	return settings, nil
}

// ResetSettings resets user settings to defaults
func (s *SettingsService) ResetSettings(ctx context.Context, userID uuid.UUID) (*models.UserSettings, error) {
	settings := models.DefaultUserSettings(userID)

	if err := s.repo.Upsert(ctx, settings); err != nil {
		return nil, err
	}

	s.eventBus.Publish("user.settings_reset", &UserSettingsResetEvent{
		UserID:    userID,
		Settings:  settings,
		ResetAt:   settings.UpdatedAt,
	})

	return settings, nil
}

// Events

// UserSettingsUpdatedEvent is emitted when user settings are updated
type UserSettingsUpdatedEvent struct {
	UserID    uuid.UUID
	Settings  *models.UserSettings
	UpdatedAt time.Time
}

// UserSettingsResetEvent is emitted when user settings are reset to defaults
type UserSettingsResetEvent struct {
	UserID   uuid.UUID
	Settings *models.UserSettings
	ResetAt  time.Time
}
