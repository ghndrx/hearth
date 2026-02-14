package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"hearth/internal/models"
)

// UserRepository defines the interface for user data access
type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	
	// Relationships
	GetFriends(ctx context.Context, userID uuid.UUID) ([]*models.User, error)
	AddFriend(ctx context.Context, userID, friendID uuid.UUID) error
	RemoveFriend(ctx context.Context, userID, friendID uuid.UUID) error
	GetBlockedUsers(ctx context.Context, userID uuid.UUID) ([]*models.User, error)
	BlockUser(ctx context.Context, userID, blockedID uuid.UUID) error
	UnblockUser(ctx context.Context, userID, blockedID uuid.UUID) error
	
	// Presence
	UpdatePresence(ctx context.Context, userID uuid.UUID, status models.PresenceStatus) error
	GetPresence(ctx context.Context, userID uuid.UUID) (*models.Presence, error)
	GetPresenceBulk(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*models.Presence, error)
}

// UserService handles user-related business logic
type UserService struct {
	repo     UserRepository
	cache    CacheService
	eventBus EventBus
}

// NewUserService creates a new user service
func NewUserService(repo UserRepository, cache CacheService, eventBus EventBus) *UserService {
	return &UserService{
		repo:     repo,
		cache:    cache,
		eventBus: eventBus,
	}
}

// GetUser retrieves a user by ID
func (s *UserService) GetUser(ctx context.Context, id uuid.UUID) (*models.User, error) {
	// Try cache first
	if s.cache != nil {
		if cached, err := s.cache.GetUser(ctx, id); err == nil && cached != nil {
			return cached, nil
		}
	}
	
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	
	// Cache for next time
	if s.cache != nil {
		_ = s.cache.SetUser(ctx, user, 5*time.Minute)
	}
	
	return user, nil
}

// GetUserByUsername retrieves a user by username
func (s *UserService) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	user, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

// UpdateUser updates user profile
func (s *UserService) UpdateUser(ctx context.Context, id uuid.UUID, updates *models.UserUpdate) (*models.User, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	
	// Check username uniqueness if changing
	if updates.Username != nil && *updates.Username != user.Username {
		existing, _ := s.repo.GetByUsername(ctx, *updates.Username)
		if existing != nil {
			return nil, ErrUsernameTaken
		}
		user.Username = *updates.Username
	}
	
	// Apply updates
	if updates.AvatarURL != nil {
		user.AvatarURL = updates.AvatarURL
	}
	if updates.BannerURL != nil {
		user.BannerURL = updates.BannerURL
	}
	if updates.Bio != nil {
		user.Bio = updates.Bio
	}
	if updates.CustomStatus != nil {
		user.CustomStatus = updates.CustomStatus
	}
	
	user.UpdatedAt = time.Now()
	
	if err := s.repo.Update(ctx, user); err != nil {
		return nil, err
	}
	
	// Invalidate cache
	if s.cache != nil {
		_ = s.cache.DeleteUser(ctx, id)
	}
	
	// Emit event
	s.eventBus.Publish("user.updated", &UserUpdatedEvent{
		UserID:    id,
		User:      user,
		UpdatedAt: user.UpdatedAt,
	})
	
	return user, nil
}

// UpdatePresence updates user's online status
func (s *UserService) UpdatePresence(ctx context.Context, userID uuid.UUID, status models.PresenceStatus, customStatus *string) error {
	if err := s.repo.UpdatePresence(ctx, userID, status); err != nil {
		return err
	}
	
	presence := &models.Presence{
		UserID:       userID,
		Status:       status,
		CustomStatus: customStatus,
		UpdatedAt:    time.Now(),
	}
	
	// Emit presence update to connected clients
	s.eventBus.Publish("presence.updated", &PresenceUpdatedEvent{
		UserID:   userID,
		Presence: presence,
	})
	
	return nil
}

// GetFriends retrieves user's friend list
func (s *UserService) GetFriends(ctx context.Context, userID uuid.UUID) ([]*models.User, error) {
	return s.repo.GetFriends(ctx, userID)
}

// AddFriend adds a friend (after request accepted)
func (s *UserService) AddFriend(ctx context.Context, userID, friendID uuid.UUID) error {
	if userID == friendID {
		return errors.New("cannot add yourself as friend")
	}
	
	if err := s.repo.AddFriend(ctx, userID, friendID); err != nil {
		return err
	}
	
	s.eventBus.Publish("friend.added", &FriendAddedEvent{
		UserID:   userID,
		FriendID: friendID,
	})
	
	return nil
}

// RemoveFriend removes a friend
func (s *UserService) RemoveFriend(ctx context.Context, userID, friendID uuid.UUID) error {
	if err := s.repo.RemoveFriend(ctx, userID, friendID); err != nil {
		return err
	}
	
	s.eventBus.Publish("friend.removed", &FriendRemovedEvent{
		UserID:   userID,
		FriendID: friendID,
	})
	
	return nil
}

// BlockUser blocks another user
func (s *UserService) BlockUser(ctx context.Context, userID, blockedID uuid.UUID) error {
	if userID == blockedID {
		return errors.New("cannot block yourself")
	}
	
	// Remove friend relationship if exists
	_ = s.repo.RemoveFriend(ctx, userID, blockedID)
	
	if err := s.repo.BlockUser(ctx, userID, blockedID); err != nil {
		return err
	}
	
	s.eventBus.Publish("user.blocked", &UserBlockedEvent{
		UserID:    userID,
		BlockedID: blockedID,
	})
	
	return nil
}

// UnblockUser unblocks a user
func (s *UserService) UnblockUser(ctx context.Context, userID, blockedID uuid.UUID) error {
	return s.repo.UnblockUser(ctx, userID, blockedID)
}

// Events

type UserUpdatedEvent struct {
	UserID    uuid.UUID
	User      *models.User
	UpdatedAt time.Time
}

type PresenceUpdatedEvent struct {
	UserID   uuid.UUID
	Presence *models.Presence
}

type FriendAddedEvent struct {
	UserID   uuid.UUID
	FriendID uuid.UUID
}

type FriendRemovedEvent struct {
	UserID   uuid.UUID
	FriendID uuid.UUID
}

type UserBlockedEvent struct {
	UserID    uuid.UUID
	BlockedID uuid.UUID
}
