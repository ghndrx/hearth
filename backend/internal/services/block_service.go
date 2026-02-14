package services

import (
	"context"
	"errors"
)

// BlockService defines the interface for blocking/unblocking and querying users.
// This interface allows for mocking during unit testing.
type BlockService interface {
	BlockUser(ctx context.Context, userID, targetUserID string) error
	UnblockUser(ctx context.Context, userID, targetUserID string) error
	IsBlocked(ctx context.Context, userID, targetUserID string) (bool, error)
	GetBlockedUsers(ctx context.Context, userID string) ([]string, error)
}

// blockService is the concrete implementation of BlockService.
type blockService struct {
	// In a real application, this would interact with a database layer
	// like Postgres or Redis. To keep the example self-contained,
	// no concrete data store is used here.
	blockMap map[string]map[string]bool // key: userID, val: map[targetUserID]bool
}

// NewBlockService initializes and returns a new BlockService.
func NewBlockService() BlockService {
	return &blockService{
		blockMap: make(map[string]map[string]bool),
	}
}

// BlockUser adds a target user to the blocking list of the requesting user.
func (s *blockService) BlockUser(ctx context.Context, userID, targetUserID string) error {
	if userID == "" || targetUserID == "" {
		return errors.New("user IDs cannot be empty")
	}

	// Ensure the user's block map exists
	if _, exists := s.blockMap[userID]; !exists {
		s.blockMap[userID] = make(map[string]bool)
	}

	// block
	s.blockMap[userID][targetUserID] = true
	return nil
}

// UnblockUser removes a target user from the blocking list of the requesting user.
func (s *blockService) UnblockUser(ctx context.Context, userID, targetUserID string) error {
	if userID == "" || targetUserID == "" {
		return errors.New("user IDs cannot be empty")
	}

	blockMap, exists := s.blockMap[userID]
	if !exists {
		return nil // Nothing to unblock
	}

	// Remove the user from the map (if present)
	delete(blockMap, targetUserID)
	return nil
}

// IsBlocked checks if the target_user is currently blocked by the requesting_user.
func (s *blockService) IsBlocked(ctx context.Context, userID, targetUserID string) (bool, error) {
	if userID == "" || targetUserID == "" {
		return false, errors.New("user IDs cannot be empty")
	}

	blockMap, exists := s.blockMap[userID]
	if !exists {
		return false, nil
	}

	return blockMap[targetUserID], nil
}

// GetBlockedUsers returns a slice of UserIDs that are currently blocked by the specified userID.
func (s *blockService) GetBlockedUsers(ctx context.Context, userID string) ([]string, error) {
	if userID == "" {
		return nil, errors.New("user ID cannot be empty")
	}

	blockMap, exists := s.blockMap[userID]
	if !exists {
		return []string{}, nil
	}

	blockedUsers := make([]string, 0, len(blockMap))
	for user := range blockMap {
		blockedUsers = append(blockedUsers, user)
	}

	return blockedUsers, nil
}