package services

import (
	"context"
	"hearth/internal/models"
)

// StatusRepository defines the interface for data operations related to statuses.
// Note: This is defined in the services package and not re-exported from an internal database package
// to adhere to dependency inversion principles.
type StatusRepository interface {
	// GetByID retrieves a status by its unique identifier.
	GetByID(ctx context.Context, id uuid.UUID) (*models.Status, error)

	// GetByUserID retrieves the status of a specific user.
	GetByUserID(ctx context.Context, userID uuid.UUID) (*models.Status, error)

	// Update updates an existing status record.
	Update(ctx context.Context, status *models.Status) error

	// Create adds a new Status record.
	Create(ctx context.Context, status *models.Status) error
}

// StatusService handles logic related to user presence and statuses.
type StatusService struct {
	repo StatusRepository
}

// NewStatusService initializes a StatusService with a repository dependency.
func NewStatusService(repo StatusRepository) *StatusService {
	return &StatusService{
		repo: repo,
	}
}

// UpdateOrCreateStatus attempts to update an existing status or create a new one if it doesn't exist.
func (s *StatusService) UpdateOrCreateStatus(ctx context.Context, userID uuid.UUID, status models.Status) (*models.Status, error) {
	// 1. Attempt to find existing status
	existingStatus, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		// If it's a DB error (like not found, or actual sql error), return it.
		return nil, err
	}

	// 2. If exists, update it
	if existingStatus != nil {
		existingStatus.Status = status.Status
		existingStatus.GameID = status.GameID
		existingStatus.ActivityDetails = status.ActivityDetails
		existingStatus.Timestamp = models.Now()

		if err := s.repo.Update(ctx, existingStatus); err != nil {
			return nil, err
		}
		return existingStatus, nil
	}

	// 3. If not exists, create it
	newStatus := &models.Status{
		UserID:         userID,
		Status:         status.Status,
		GameID:         status.GameID,
		ActivityDetails: status.ActivityDetails,
		Timestamp: models.Now(),
	}

	if err := s.repo.Create(ctx, newStatus); err != nil {
		return nil, err
	}
	return newStatus, nil
}

// GetUserStatus retrieves the current status for a given user.
func (s *StatusService) GetUserStatus(ctx context.Context, userID uuid.UUID) (*models.Status, error) {
	return s.repo.GetByUserID(ctx, userID)
}