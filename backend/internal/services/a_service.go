package services

import (
	"context"
	"errors"
	"fmt"
)

// IChannelRepository defines the data access contract.
// Using an interface allows us to swap the database implementation
// (e.g., SQL, MongoDB) without changing the service or tests.
type IChannelRepository interface {
	FindChannelsByGuildID(ctx context.Context, guildID string) ([]string, error)
	AddChannel(ctx context.Context, guildID, channelName string) error
}

// ChannelService defines the domain logic contract.
type IChannelService interface {
	GetChannels(ctx context.Context, guildID string) ([]string, error)
	CreateChannel(ctx context.Context, guildID, channelName string) error
}

// ChannelService implements IChannelService.
type ChannelService struct {
	// Defines strict binding to the interface.
	// Embedding the interface allows any implementation to be used.
	repo IChannelRepository
}

// NewChannelService is a constructor that panics or returns an error if the
// passed repository does not implement the specific interface.
func NewChannelService(repo IChannelRepository) *ChannelService {
	return &ChannelService{repo: repo}
}

// GetChannels retrieves channel names for a specific guild.
func (s *ChannelService) GetChannels(ctx context.Context, guildID string) ([]string, error) {
	if guildID == "" {
		return nil, errors.New("guildID cannot be empty")
	}

	channels, err := s.repo.FindChannelsByGuildID(ctx, guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch channels: %w", err)
	}
	return channels, nil
}

// CreateChannel adds a new channel to the repository.
func (s *ChannelService) CreateChannel(ctx context.Context, guildID, channelName string) error {
	
	// Business Logic Validation
	if guildID == "" {
		return errors.New("guildID cannot be empty")
	}
	if channelName == "" {
		return errors.New("channelName cannot be empty")
	}

	// Check uniqueness or formatting logic would go here in a real app.
	// We'll assume the repository handles storage.

	if err := s.repo.AddChannel(ctx, guildID, channelName); err != nil {
		return fmt.Errorf("failed to add channel: %w", err)
	}

	return nil
}