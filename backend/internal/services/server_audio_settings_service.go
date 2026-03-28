package services

import (
	"context"
	"time"

	"github.com/google/uuid"

	"hearth/internal/models"
)

// ServerAudioSettingsRepository defines the interface for server audio settings data access
type ServerAudioSettingsRepository interface {
	Get(ctx context.Context, userID, serverID uuid.UUID) (*models.ServerAudioSettings, error)
	GetAllForUser(ctx context.Context, userID uuid.UUID) ([]*models.ServerAudioSettings, error)
	Upsert(ctx context.Context, settings *models.ServerAudioSettings) error
	Delete(ctx context.Context, userID, serverID uuid.UUID) error
}

// ServerAudioSettingsService handles per-server audio settings business logic
type ServerAudioSettingsService struct {
	repo     ServerAudioSettingsRepository
	eventBus EventBus
}

// NewServerAudioSettingsService creates a new server audio settings service
func NewServerAudioSettingsService(repo ServerAudioSettingsRepository, eventBus EventBus) *ServerAudioSettingsService {
	return &ServerAudioSettingsService{
		repo:     repo,
		eventBus: eventBus,
	}
}

// GetSettings retrieves audio settings for a user in a server, returning defaults if none exist
func (s *ServerAudioSettingsService) GetSettings(ctx context.Context, userID, serverID uuid.UUID) (*models.ServerAudioSettings, error) {
	settings, err := s.repo.Get(ctx, userID, serverID)
	if err != nil {
		return nil, err
	}
	if settings == nil {
		return models.DefaultServerAudioSettings(userID, serverID), nil
	}
	return settings, nil
}

// GetAllForUser retrieves audio settings for all servers for a user
func (s *ServerAudioSettingsService) GetAllForUser(ctx context.Context, userID uuid.UUID) ([]*models.ServerAudioSettings, error) {
	return s.repo.GetAllForUser(ctx, userID)
}

// UpdateSettings updates audio settings for a user in a server
func (s *ServerAudioSettingsService) UpdateSettings(ctx context.Context, userID, serverID uuid.UUID, updates *models.UpdateServerAudioSettingsRequest) (*models.ServerAudioSettings, error) {
	settings, err := s.GetSettings(ctx, userID, serverID)
	if err != nil {
		return nil, err
	}

	if updates.InputDeviceID != nil {
		settings.InputDeviceID = *updates.InputDeviceID
	}
	if updates.OutputDeviceID != nil {
		settings.OutputDeviceID = *updates.OutputDeviceID
	}
	if updates.InputVolume != nil {
		settings.InputVolume = *updates.InputVolume
	}
	if updates.OutputVolume != nil {
		settings.OutputVolume = *updates.OutputVolume
	}
	if updates.PushToTalkEnabled != nil {
		settings.PushToTalkEnabled = *updates.PushToTalkEnabled
	}
	if updates.PushToTalkKey != nil {
		settings.PushToTalkKey = *updates.PushToTalkKey
	}

	settings.UpdatedAt = time.Now()

	if err := s.repo.Upsert(ctx, settings); err != nil {
		return nil, err
	}

	s.eventBus.Publish("server.audio_settings_updated", &ServerAudioSettingsUpdatedEvent{
		UserID:   userID,
		ServerID: serverID,
		Settings: settings,
	})

	return settings, nil
}

// DeleteSettings deletes audio settings for a user in a server
func (s *ServerAudioSettingsService) DeleteSettings(ctx context.Context, userID, serverID uuid.UUID) error {
	return s.repo.Delete(ctx, userID, serverID)
}

// ServerAudioSettingsUpdatedEvent is emitted when server audio settings are updated
type ServerAudioSettingsUpdatedEvent struct {
	UserID   uuid.UUID
	ServerID uuid.UUID
	Settings *models.ServerAudioSettings
}
