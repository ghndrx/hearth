// Package services provides domain logic for the Hearth application.
package services

import (
	"context"
	"errors"
)

var (
	// ErrChannelNotFound is returned when a channel cannot be located.
	ErrChannelNotFound = errors.New("channel not found")
)

// Channel represents a text or voice channel in a guild.
type Channel struct {
	ID          string
	Name        string
	Type        ChannelType
	Description string
	MemberCount int
}

// ChannelType distinguishes between chat and voice channels.
type ChannelType int

const (
	ChannelTypeText ChannelType = iota
	ChannelTypeVoice
)

// ChannelRepository defines the contract for persisting Channel entities.
type ChannelRepository interface {
	GetByID(ctx context.Context, id string) (*Channel, error)
	Create(ctx context.Context, channel *Channel) error
}

// ChannelService handles business logic surrounding channels.
type ChannelService struct {
	repo ChannelRepository
}

// NewChannelService creates a new instance of ChannelService.
func NewChannelService(repo ChannelRepository) *ChannelService {
	return &ChannelService{
		repo: repo,
	}
}

// CreateChannel creates a new text channel for the guild.
// Command: /create-channel
func (s *ChannelService) CreateChannel(ctx context.Context, name string, channelType ChannelType) (*Channel, error) {
	if name == "" {
		return nil, errors.New("channel name cannot be empty")
	}

	newChannel := &Channel{
		ID:          randomString(6), // Simplified ID generation
		Name:        name,
		Type:        channelType,
		MemberCount: 0,
	}

	if err := s.repo.Create(ctx, newChannel); err != nil {
		return nil, err
	}

	return newChannel, nil
}

// GetChannel retrieves a specific channel by ID.
func (s *ChannelService) GetChannel(ctx context.Context, id string) (*Channel, error) {
	if id == "" {
		return nil, errors.New("channel id cannot be empty")
	}

	channel, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return channel, nil
}