package services

import (
	"context"
	"hearth/pkg/models"
)

// IReadStateRepository defines the database interaction interface for read state.
type IReadStateRepository interface {
	// GetByUserIDAndChannelID fetches the most recent read state for a user and channel pair.
	GetByUserIDAndChannelID(ctx context.Context, userID, channelID int64) (*models.ReadState, error)
	
	// MarkRead saves or updates the read state.
	MarkRead(ctx context.Context, readState *models.ReadState) error
	
	// GetUnreadCount fetches the aggregate count from the database.
	GetUnreadCount(ctx context.Context, userID, channelID int64) (int, error)
}

// ReadStateService handles business logic for chat read states.
type ReadStateService struct {
	repo IReadStateRepository
}

// NewReadStateService creates a new ReadStateService with its dependencies.
func NewReadStateService(repo IReadStateRepository) *ReadStateService {
	return &ReadStateService{
		repo: repo,
	}
}

// MarkRead updates the read timestamp for a user in a specific channel.
// If the state doesn't exist, it creates a new one.
func (s *ReadStateService) MarkRead(ctx context.Context, userID, channelID int64, lastMessageID int64) error {
	// 1. Fetch existing state to determine if we are INSERTING or UPDATING
	existingState, err := s.repo.GetByUserIDAndChannelID(ctx, userID, channelID)
	if err != nil {
		return err
	}

	// 2. Construct the new state
	newState := &models.ReadState{
		UserID:        userID,
		ChannelID:     channelID,
		LastMessageID: lastMessageID,
		// Heartbeats (Positions) are typically stored in a separate table in Discord,
		// but for this service, we focus on the LastReadMessage.
	}

	if existingState != nil {
		// Update existing state using upsert logic (optimistic approach for simplicity)
		// In a strict SQL implementation, one would use INSERT ... ON CONFLICT UPDATE
		newState.ID = existingState.ID
	}

	return s.repo.MarkRead(ctx, newState)
}

// GetUnreadCount retrieves the number of.messages that a user has not read yet.
// This is calculated by finding the last read message ID and comparing it against the stream.
func (s *ReadStateService) GetUnreadCount(ctx context.Context, userID, channelID int64) (int, error) {
	// 1. Retrieve the user's stored LastReadMessageID
	userState, err := s.repo.GetByUserIDAndChannelID(ctx, userID, channelID)
	if err != nil {
		return 0, err
	}

	// If no read state exists, all messages might be unread (or we default to 0/0 if DB is empty)
	// For this implementation, we assume strict inventory logic:
	// Unread = Total Messages - LastMessageID + 1
	// However, efficient counting usually requires joins in the repository.
	// We delegate the specific logic to the repository.
	return s.repo.GetUnreadCount(ctx, userID, channelID)
}