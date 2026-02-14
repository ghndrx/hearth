package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Heart represents a user's "like" action on another user
type Heart struct {
	ID        uuid.UUID
	AuthorID  uuid.UUID
	TargetID  uuid.UUID
	Active    bool
	CreatedAt time.Time
}

// PublicMetrics contains public engagement metrics
type PublicMetrics struct {
	LikeCount   int64
	RepostCount int64
	QuoteCount  int64
}

// HeartRepository defines the interface for heart operations.
// It abstracts the underlying database implementation (e.g., PostgreSQL).
type HeartRepository interface {
	Create(ctx context.Context, heart *Heart) error
	GetByID(ctx context.Context, id uuid.UUID) (*Heart, error)
	GetByTargetID(ctx context.Context, targetUserID uuid.UUID) (int64, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
}

// HeartService handles logic related to hearts.
type HeartService struct {
	repo HeartRepository
}

// NewHeartService initializes the service with the required repository.
func NewHeartService(repo HeartRepository) *HeartService {
	return &HeartService{repo: repo}
}

// CreateHeart creates a new heart for a user towards another user.
// Returns an error if the target user is the same as the author.
func (s *HeartService) CreateHeart(ctx context.Context, authorID, targetUserID uuid.UUID) (*Heart, error) {
	if authorID == targetUserID {
		return nil, ErrSelfAction
	}

	// NOTE: In a real application, you might want to check against a banned list
	// or user status here before proceeding.

	heart := &Heart{
		ID:        uuid.New(),
		AuthorID:  authorID,
		TargetID:  targetUserID,
		Active:    true,
		CreatedAt: time.Now(),
	}

	if err := s.repo.Create(ctx, heart); err != nil {
		return nil, fmt.Errorf("failed to create heart: %w", err)
	}

	return heart, nil
}

// GetHeartSummary fetches a summary of hearts: counts for the user who requested it
// and the total number received.
func (s *HeartService) GetHeartSummary(ctx context.Context, userID, requestFromUserID uuid.UUID) (PublicMetrics, error) {
	// 1. Count hearts sent by the requesting user
	sentCount, err := s.repo.GetByUserID(ctx, requestFromUserID)
	if err != nil {
		return PublicMetrics{}, fmt.Errorf("failed to get sent hearts count: %w", err)
	}

	// 2. Count hearts received by the target user
	receivedCount, err := s.repo.GetByTargetID(ctx, userID)
	if err != nil {
		return PublicMetrics{}, fmt.Errorf("failed to get received hearts count: %w", err)
	}

	return PublicMetrics{
		LikeCount:   receivedCount,
		RepostCount: sentCount,
		QuoteCount:  0,
	}, nil
}
