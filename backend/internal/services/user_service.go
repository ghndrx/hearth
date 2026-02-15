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
	
	// Friend Requests
	GetRelationship(ctx context.Context, userID, targetID uuid.UUID) (int, error)
	SendFriendRequest(ctx context.Context, senderID, receiverID uuid.UUID) error
	GetIncomingFriendRequests(ctx context.Context, userID uuid.UUID) ([]*models.User, error)
	GetOutgoingFriendRequests(ctx context.Context, userID uuid.UUID) ([]*models.User, error)
	AcceptFriendRequest(ctx context.Context, receiverID, senderID uuid.UUID) error
	DeclineFriendRequest(ctx context.Context, userID, otherID uuid.UUID) error
	
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
	if err := s.repo.UnblockUser(ctx, userID, blockedID); err != nil {
		return err
	}

	s.eventBus.Publish("user.unblocked", &UserUnblockedEvent{
		UserID:      userID,
		UnblockedID: blockedID,
	})
	return nil
}

// SendFriendRequest sends a friend request from sender to receiver
func (s *UserService) SendFriendRequest(ctx context.Context, senderID, receiverID uuid.UUID) error {
	if senderID == receiverID {
		return errors.New("cannot send friend request to yourself")
	}
	
	// Check if target user exists
	receiver, err := s.repo.GetByID(ctx, receiverID)
	if err != nil {
		return err
	}
	if receiver == nil {
		return ErrUserNotFound
	}
	
	// Check existing relationship
	relType, err := s.repo.GetRelationship(ctx, senderID, receiverID)
	if err != nil {
		return err
	}
	
	switch relType {
	case 1: // Already friends
		return errors.New("already friends")
	case 2: // Blocked (sender blocked receiver)
		return errors.New("cannot send friend request to blocked user")
	case 4: // Already sent request
		return errors.New("friend request already sent")
	case 3: // They sent us a request - auto-accept
		if err := s.repo.AcceptFriendRequest(ctx, senderID, receiverID); err != nil {
			return err
		}
		s.eventBus.Publish("friend.added", &FriendAddedEvent{
			UserID:   senderID,
			FriendID: receiverID,
		})
		return nil
	}
	
	// Check if receiver blocked sender
	receiverRelType, err := s.repo.GetRelationship(ctx, receiverID, senderID)
	if err != nil {
		return err
	}
	if receiverRelType == 2 {
		return errors.New("cannot send friend request")
	}
	
	if err := s.repo.SendFriendRequest(ctx, senderID, receiverID); err != nil {
		return err
	}
	
	s.eventBus.Publish("friend.request_sent", &FriendRequestSentEvent{
		SenderID:   senderID,
		ReceiverID: receiverID,
	})
	
	return nil
}

// GetIncomingFriendRequests returns all pending incoming friend requests
func (s *UserService) GetIncomingFriendRequests(ctx context.Context, userID uuid.UUID) ([]*models.User, error) {
	return s.repo.GetIncomingFriendRequests(ctx, userID)
}

// GetOutgoingFriendRequests returns all pending outgoing friend requests
func (s *UserService) GetOutgoingFriendRequests(ctx context.Context, userID uuid.UUID) ([]*models.User, error) {
	return s.repo.GetOutgoingFriendRequests(ctx, userID)
}

// AcceptFriendRequest accepts a pending friend request
func (s *UserService) AcceptFriendRequest(ctx context.Context, receiverID, senderID uuid.UUID) error {
	// Verify that there is a pending incoming request
	relType, err := s.repo.GetRelationship(ctx, receiverID, senderID)
	if err != nil {
		return err
	}
	if relType != 3 {
		return errors.New("no pending friend request from this user")
	}
	
	if err := s.repo.AcceptFriendRequest(ctx, receiverID, senderID); err != nil {
		return err
	}
	
	s.eventBus.Publish("friend.added", &FriendAddedEvent{
		UserID:   receiverID,
		FriendID: senderID,
	})
	
	return nil
}

// DeclineFriendRequest declines a pending friend request
func (s *UserService) DeclineFriendRequest(ctx context.Context, userID, otherID uuid.UUID) error {
	// Verify that there is a pending request (either incoming or outgoing)
	relType, err := s.repo.GetRelationship(ctx, userID, otherID)
	if err != nil {
		return err
	}
	if relType != 3 && relType != 4 {
		return errors.New("no pending friend request")
	}
	
	if err := s.repo.DeclineFriendRequest(ctx, userID, otherID); err != nil {
		return err
	}
	
	s.eventBus.Publish("friend.request_declined", &FriendRequestDeclinedEvent{
		UserID:  userID,
		OtherID: otherID,
	})
	
	return nil
}

// GetRelationship returns the relationship type between two users
func (s *UserService) GetRelationship(ctx context.Context, userID, targetID uuid.UUID) (int, error) {
	return s.repo.GetRelationship(ctx, userID, targetID)
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

type UserUnblockedEvent struct {
	UserID      uuid.UUID
	UnblockedID uuid.UUID
}

type FriendRequestSentEvent struct {
	SenderID   uuid.UUID
	ReceiverID uuid.UUID
}

type FriendRequestDeclinedEvent struct {
	UserID  uuid.UUID
	OtherID uuid.UUID
}

// RecentActivityInfo represents a user's recent activity for profile display
type RecentActivityInfo struct {
	LastMessageAt   *time.Time
	ServerName      *string
	ChannelName     *string
	MessageCount24h int
}

// GetMutualFriends returns friends that both users have in common
func (s *UserService) GetMutualFriends(ctx context.Context, userID1, userID2 uuid.UUID, limit int) ([]*models.User, int, error) {
	// Use the repository method if available via type assertion
	if repo, ok := s.repo.(interface {
		GetMutualFriends(ctx context.Context, userID1, userID2 uuid.UUID, limit int) ([]*models.User, int, error)
	}); ok {
		return repo.GetMutualFriends(ctx, userID1, userID2, limit)
	}
	// Fallback: return empty if method not available
	return []*models.User{}, 0, nil
}

// GetRecentActivity returns a user's recent activity visible to the requester
func (s *UserService) GetRecentActivity(ctx context.Context, requesterID, targetID uuid.UUID) (*RecentActivityInfo, error) {
	// Use the repository method if available via type assertion
	if repo, ok := s.repo.(interface {
		GetRecentActivity(ctx context.Context, requesterID, targetID uuid.UUID) (*struct {
			LastMessageAt     *time.Time
			LastMessageServer *uuid.UUID
			ServerName        *string
			ChannelName       *string
			MessageCount24h   int
		}, error)
	}); ok {
		activity, err := repo.GetRecentActivity(ctx, requesterID, targetID)
		if err != nil {
			return nil, err
		}
		if activity == nil {
			return nil, nil
		}
		return &RecentActivityInfo{
			LastMessageAt:   activity.LastMessageAt,
			ServerName:      activity.ServerName,
			ChannelName:     activity.ChannelName,
			MessageCount24h: activity.MessageCount24h,
		}, nil
	}
	// Fallback: return nil if method not available
	return nil, nil
}
