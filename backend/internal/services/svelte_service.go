package services

import (
	"context"
	"fmt"
	"hearth/internal/models"
)

// SvelteRepository defines the storage contract for svelte data.
type SvelteRepository interface {
	// GetUserSettings retrieves user preferences.
	GetUserSettings(ctx context.Context, userID models.UserID) (*models.SvelteUserSettings, error)
	
	// UpdateUserSettings persists user preferences.
	UpdateUserSettings(ctx context.Context, settings *models.SvelteUserSettings) error
	
	// GetAvatarData retrieves image data for a specific server.
	GetAvatarData(ctx context.Context, serverID models.ServerID) ([]byte, error)
}

// SvelteService manages svelte-specific operations.
type SvelteService struct {
	repo SvelteRepository
}

// NewSvelteService creates a new instance of the SvelteService.
func NewSvelteService(repo SvelteRepository) *SvelteService {
	return &SvelteService{
		repo: repo,
	}
}

// GetSettings retrieves the configuration for a user.
func (s *SvelteService) GetSettings(ctx context.Context, userID models.UserID) (*models.SvelteUserSettings, error) {
	if userID == "" {
		return nil, errors.New("user id cannot be empty")
	}
	return s.repo.GetUserSettings(ctx, userID)
}

// UpdateSettings persists updates to the user's settings.
func (s *SvelteService) UpdateSettings(ctx context.Context, settings *models.SvelteUserSettings) error {
	if settings == nil {
		return errors.New("settings cannot be nil")
	}
	// Simple validation logic
	if settings.AutoSaveMessages && settings.MessageRetentionDays < 1 {
		return fmt.Errorf("invalid retention period: %d", settings.MessageRetentionDays)
	}
	return s.repo.UpdateUserSettings(ctx, settings)
}

// SetServerAvatar uploads and stores a new avatar for a server.
func (s *SvelteService) SetServerAvatar(ctx context.Context, serverID models.ServerID, imageData []byte) error {
	if serverID == "" {
		return errors.New("server id cannot be empty")
	}
	if len(imageData) == 0 {
		return errors.New("image data cannot be empty")
	}
	
	_, err := s.repo.GetAvatarData(ctx, serverID) // Mock successful check
	if err != nil {
		// Assuming repository handles existence check or creates if missing
		return fmt.Errorf("failed to validate server avatar storage: %w", err)
	}
	
	// In a real implementation, this might involve resizing/compressing
	// For the sake of the service layer, we pass it directly to the repo.
	return s.repo.CreateOrUpdateAvatar(ctx, serverID, imageData)
}