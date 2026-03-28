package services

import (
	"context"

	"github.com/google/uuid"

	"hearth/internal/models"
)

// UserChannelSettingsRepository defines the interface for user channel settings data access
type UserChannelSettingsRepository interface {
	Upsert(ctx context.Context, settings *models.UserChannelSettings) error
	Get(ctx context.Context, userID, channelID uuid.UUID) (*models.UserChannelSettings, error)
	IsChannelMuted(ctx context.Context, userID, channelID uuid.UUID) (bool, error)
	GetMutedChannelIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}

// UserChannelSettingsService handles user channel settings business logic
type UserChannelSettingsService struct {
	repo     UserChannelSettingsRepository
	eventBus EventBus
}

// NewUserChannelSettingsService creates a new user channel settings service
func NewUserChannelSettingsService(repo UserChannelSettingsRepository, eventBus EventBus) *UserChannelSettingsService {
	return &UserChannelSettingsService{
		repo:     repo,
		eventBus: eventBus,
	}
}

// SetChannelMuted sets the mute state for a channel for a user
func (s *UserChannelSettingsService) SetChannelMuted(ctx context.Context, userID, channelID uuid.UUID, muted bool) (*models.UserChannelSettings, error) {
	settings := &models.UserChannelSettings{
		UserID:    userID,
		ChannelID: channelID,
		Muted:     muted,
	}

	if err := s.repo.Upsert(ctx, settings); err != nil {
		return nil, err
	}

	s.eventBus.Publish("channel.mute_updated", &ChannelMuteUpdatedEvent{
		UserID:    userID,
		ChannelID: channelID,
		Muted:     muted,
	})

	return settings, nil
}

// IsChannelMuted checks if a channel is muted for a user
func (s *UserChannelSettingsService) IsChannelMuted(ctx context.Context, userID, channelID uuid.UUID) (bool, error) {
	return s.repo.IsChannelMuted(ctx, userID, channelID)
}

// GetMutedChannelIDs returns all muted channel IDs for a user
func (s *UserChannelSettingsService) GetMutedChannelIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	return s.repo.GetMutedChannelIDs(ctx, userID)
}

// ChannelMuteUpdatedEvent is emitted when a user mutes or unmutes a channel
type ChannelMuteUpdatedEvent struct {
	UserID    uuid.UUID `json:"user_id"`
	ChannelID uuid.UUID `json:"channel_id"`
	Muted     bool      `json:"muted"`
}
