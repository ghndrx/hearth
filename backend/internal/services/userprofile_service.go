package services

import (
	"context"
	"errors"
)

// UserProfile represents the data and metadata of a user in Hearth.
type UserProfile struct {
	UserID   string
	Username string
	Avatar   string // URL to the avatar image
	Bio      string
	Status   string // e.g., "Online", "Playing Hearthstone", "Away"
}

// UserProfileService defines the interface for user profile operations.
type UserProfileService interface {
	GetUserProfile(ctx context.Context, userID string) (*UserProfile, error)
	UpdateUserProfile(ctx context.Context, userID string, updates *UserProfile) error
}

// userProfileService implements UserProfileService.
type userProfileService struct {
	store UserProfileStore
}

// UserProfileStore abstraction for data persistence.
type UserProfileStore interface {
	GetUser(ctx context.Context, userID string) (*UserProfile, error)
	SaveUser(ctx context.Context, userID string, profile *UserProfile) error
}

// NewUserProfileService creates a new instance of the service.
func NewUserProfileService(store UserProfileStore) UserProfileService {
	return &userProfileService{
		store: store,
	}
}

// GetUserProfile retrieves a user profile by ID.
// Implements the persona of a frontend component accessing backend data.
// Handles "NotFound" by returning an explicit error.
func (s *userProfileService) GetUserProfile(ctx context.Context, userID string) (*UserProfile, error) {
	if userID == "" {
		return nil, errors.New("invalid user ID: empty string")
	}

	profile, err := s.store.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	return profile, nil
}

// UpdateUserProfile applies changes to a user's profile.
// In a true edit scenario, we might merge fields, but here we perform a set operation.
func (s *userProfileService) UpdateUserProfile(ctx context.Context, userID string, updates *UserProfile) error {
	if updates == nil {
		return errors.New("cannot update with nil profile data")
	}
	if userID == "" {
		return errors.New("invalid user ID: empty string")
	}

	// Fetch current profile to ensure valid data merge (optional safety check)
	currentProfile, err := s.store.GetUser(ctx, userID)
	if err != nil {
		return err
	}

	// Merge distinct fields
	// (In a real DB update, this would just update fields sent in updates)
	if updates.Username != "" {
		currentProfile.Username = updates.Username
	}
	if updates.Avatar != "" {
		currentProfile.Avatar = updates.Avatar
	}
	if updates.Bio != "" {
		currentProfile.Bio = updates.Bio
	}
	if updates.Status != "" {
		currentProfile.Status = updates.Status
	}

	return s.store.SaveUser(ctx, userID, currentProfile)
}