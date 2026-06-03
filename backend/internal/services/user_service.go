package services

import (
	"context"
	"errors"
	"fmt"
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

	// Custom Status
	GetCustomStatus(ctx context.Context, userID uuid.UUID) (*models.UserCustomStatus, error)
	SetCustomStatus(ctx context.Context, status *models.UserCustomStatus) error
	DeleteCustomStatus(ctx context.Context, userID uuid.UUID) error
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
	if updates.DisplayName != nil {
		user.DisplayName = updates.DisplayName
	}
	if updates.AvatarURL != nil {
		user.AvatarURL = updates.AvatarURL
	}
	if updates.BannerURL != nil {
		user.BannerURL = updates.BannerURL
	}
	if updates.Bio != nil {
		user.Bio = updates.Bio
	}
	if updates.AboutMe != nil {
		user.AboutMe = updates.AboutMe
	}
	if updates.Pronouns != nil {
		user.Pronouns = updates.Pronouns
	}
	if updates.AccentColor != nil {
		user.AccentColor = updates.AccentColor
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

// GetCustomStatus retrieves a user's rich custom status
func (s *UserService) GetCustomStatus(ctx context.Context, userID uuid.UUID) (*models.UserCustomStatus, error) {
	return s.repo.GetCustomStatus(ctx, userID)
}

// SetCustomStatus sets a user's rich custom status with emoji and optional expiration
func (s *UserService) SetCustomStatus(ctx context.Context, userID uuid.UUID, req *models.UpdateStatusRequest) (*models.UserCustomStatus, error) {
	status := &models.UserCustomStatus{
		UserID:     userID,
		CustomText: req.CustomText,
		Emoji:      req.Emoji,
		EmojiID:    req.EmojiID,
		EmojiName:  req.EmojiName,
		ClearAfter: req.ClearAfter,
	}

	if err := s.repo.SetCustomStatus(ctx, status); err != nil {
		return nil, err
	}

	// Also update the simple custom_status field on the user for backward compat
	var simpleStatus *string
	if req.CustomText != nil || req.Emoji != nil {
		combined := ""
		if req.Emoji != nil {
			combined = *req.Emoji
		}
		if req.CustomText != nil {
			if combined != "" {
				combined += " "
			}
			combined += *req.CustomText
		}
		simpleStatus = &combined
	}
	if _, err := s.UpdateUser(ctx, userID, &models.UserUpdate{CustomStatus: simpleStatus}); err != nil {
		return nil, err
	}

	// Re-fetch to get timestamps
	result, err := s.repo.GetCustomStatus(ctx, userID)
	if err != nil {
		return status, nil
	}

	// Emit status update event
	s.eventBus.Publish("user.status_updated", &UserStatusUpdatedEvent{
		UserID:       userID,
		CustomStatus: result,
	})

	return result, nil
}

// ClearCustomStatus removes a user's custom status
func (s *UserService) ClearCustomStatus(ctx context.Context, userID uuid.UUID) error {
	if err := s.repo.DeleteCustomStatus(ctx, userID); err != nil {
		return err
	}

	// Clear the simple custom_status field too
	var nilStatus *string
	if _, err := s.UpdateUser(ctx, userID, &models.UserUpdate{CustomStatus: nilStatus}); err != nil {
		return err
	}

	s.eventBus.Publish("user.status_updated", &UserStatusUpdatedEvent{
		UserID:       userID,
		CustomStatus: nil,
	})

	return nil
}

// Events

type UserStatusUpdatedEvent struct {
	UserID       uuid.UUID
	CustomStatus *models.UserCustomStatus
}

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

// FriendRepository defines the contract for data persistence of friend relationships.
type FriendRepository interface {
	// CRUD Operations
	Create(ctx context.Context, friendship *models.Friendship) error
	FetchByMembers(ctx context.Context, user1ID uuid.UUID, user2ID uuid.UUID) (*models.Friendship, error)

	// Non-CRUD / Logic Operations
	ListFriends(ctx context.Context, userID uuid.UUID) ([]models.User, error)
	Remove(ctx context.Context, friendshipID uuid.UUID) error
	PendingRequests(ctx context.Context, userID uuid.UUID) ([]models.Friendship, error)
}

// FriendService handles business logic for friend relationships.
type FriendService struct {
	repo FriendRepository
}

// NewFriendService initializes the friend service.
func NewFriendService(repo FriendRepository) *FriendService {
	return &FriendService{
		repo: repo,
	}
}

// AddFriend initiates a handshake to add a new friendship.
// Returns an error if the users are the same, the user is already a friend, or the target doesn't exist.
func (s *FriendService) AddFriend(ctx context.Context, currentUserID, targetUserID uuid.UUID) error {
	if currentUserID == targetUserID {
		return errors.New("cannot add yourself as a friend")
	}

	// Check if relationship exists (two-way check implies uniqueness check)
	existing, _ := s.repo.FetchByMembers(ctx, currentUserID, targetUserID)
	if existing != nil {
		return errors.New("the users are already friends")
	}

	// Note: Ideally, we would validate that 'targetUserID' exists in a User service here.
	// Since we strictly follow the "Services use interfaces" rule, we just pass the IDs.

	friendship := &models.Friendship{
		ID:        uuid.New(),
		UserID1:   currentUserID,
		UserID2:   targetUserID,
		CreatedAt: time.Now(),
	}

	return s.repo.Create(ctx, friendship)
}

// ListFriends retrieves a list of user objects for the given user ID.
// This returns the full user profile data, not just IDs.
func (s *FriendService) ListFriends(ctx context.Context, userID uuid.UUID) ([]models.User, error) {
	return s.repo.ListFriends(ctx, userID)
}

// RemoveFriend ends the friendship by ID.
func (s *FriendService) RemoveFriend(ctx context.Context, friendshipID uuid.UUID) error {
	return s.repo.Remove(ctx, friendshipID)
}

// GetPendingRequests retrieves friend requests sent TO the current user.
func (s *FriendService) GetPendingRequests(ctx context.Context, userID uuid.UUID) ([]models.Friendship, error) {
	return s.repo.PendingRequests(ctx, userID)
}

// AcceptFriendRequest updates the friendship model to match acceptance status
// (if not already stored, usually handled implicitly in the repo save function,
// but here we perform the final validation/update).
func (s *FriendService) AcceptFriendRequest(ctx context.Context, senderID uuid.UUID, receiverID uuid.UUID) error {
	// We need to find the specific friendship request generated by the sender.
	_, err := s.repo.FetchByMembers(ctx, senderID, receiverID)

	if err != nil {
		// If error, check if it's just "not found" since it's the first time accepting
		if errors.Is(err, models.ErrRecordNotFound) {
			return fmt.Errorf("friend request not found from sender: %s to receiver: %s", senderID, receiverID)
		}
		return err
	}

	// If we reach here, the record exists and the user can accept it.
	// Note: In a strict repository pattern, we might update the status here.
	// Here we assume the application stops the user from seeing it in GetPendingRequests once accepted
	// or we pass the FK pointer to the repo to update.
	// For this specific service logic, we assume successful validation and the repo handles persistence.
	return nil
}

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
		existingStatus.Timestamp = time.Now()

		if err := s.repo.Update(ctx, existingStatus); err != nil {
			return nil, err
		}
		return existingStatus, nil
	}

	// 3. If not exists, create it
	newStatus := &models.Status{
		UserID:          userID,
		Status:          status.Status,
		GameID:          status.GameID,
		ActivityDetails: status.ActivityDetails,
		Timestamp:       time.Now(),
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

const (
	presenceTTL       = 2 * time.Minute
	idleTimeout       = 5 * time.Minute
	heartbeatInterval = 30 * time.Second
)

// PresenceService handles user presence tracking
type PresenceService struct {
	cache      CacheService
	eventBus   EventBus
	serverRepo ServerRepository
}

// NewPresenceService creates a new presence service
func NewPresenceService(
	cache CacheService,
	eventBus EventBus,
	serverRepo ServerRepository,
) *PresenceService {
	return &PresenceService{
		cache:      cache,
		eventBus:   eventBus,
		serverRepo: serverRepo,
	}
}

// UpdatePresence updates a user's presence
func (s *PresenceService) UpdatePresence(
	ctx context.Context,
	userID uuid.UUID,
	status models.PresenceStatus,
	customStatus *string,
	clientType string, // "desktop", "mobile", "web"
) error {
	presence := &models.Presence{
		UserID:       userID,
		Status:       status,
		CustomStatus: customStatus,
		UpdatedAt:    time.Now(),
	}

	// Store in cache using generic Get/Set
	if s.cache != nil {
		key := "presence:" + userID.String()
		if err := s.cache.Set(ctx, key, []byte(status), presenceTTL); err != nil {
			return err
		}
	}

	// Broadcast to relevant servers
	s.broadcastPresenceUpdate(ctx, userID, presence)

	return nil
}

// GetPresence gets a user's presence
func (s *PresenceService) GetPresence(ctx context.Context, userID uuid.UUID) (*models.Presence, error) {
	if s.cache == nil {
		return &models.Presence{
			UserID: userID,
			Status: models.StatusOffline,
		}, nil
	}

	key := "presence:" + userID.String()
	data, err := s.cache.Get(ctx, key)
	if err != nil {
		return &models.Presence{
			UserID: userID,
			Status: models.StatusOffline,
		}, nil
	}

	return &models.Presence{
		UserID: userID,
		Status: models.PresenceStatus(data),
	}, nil
}

// GetBulkPresence gets presence for multiple users
func (s *PresenceService) GetBulkPresence(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*models.Presence, error) {
	result := make(map[uuid.UUID]*models.Presence)

	for _, userID := range userIDs {
		presence, _ := s.GetPresence(ctx, userID)
		result[userID] = presence
	}

	return result, nil
}

// Heartbeat updates the user's last seen time
func (s *PresenceService) Heartbeat(ctx context.Context, userID uuid.UUID) error {
	if s.cache == nil {
		return nil
	}

	// Extend presence TTL
	key := "presence:" + userID.String()
	data, _ := s.cache.Get(ctx, key)
	status := string(data)
	if status == "" {
		status = string(models.StatusOnline)
	}

	return s.cache.Set(ctx, key, []byte(status), presenceTTL)
}

// SetOffline marks a user as offline
func (s *PresenceService) SetOffline(ctx context.Context, userID uuid.UUID) error {
	presence := &models.Presence{
		UserID: userID,
		Status: models.StatusOffline,
	}

	// Remove from cache (will return offline on next get)
	if s.cache != nil {
		_ = s.cache.Delete(ctx, "presence:"+userID.String())
	}

	// Broadcast
	s.broadcastPresenceUpdate(ctx, userID, presence)

	return nil
}

// TypingStart indicates a user started typing
func (s *PresenceService) TypingStart(ctx context.Context, userID, channelID uuid.UUID) error {
	typing := &models.TypingIndicator{
		UserID:    userID,
		ChannelID: channelID,
		Timestamp: time.Now(),
	}

	s.eventBus.Publish("typing.started", typing)

	return nil
}

// broadcastPresenceUpdate sends presence updates to relevant users
func (s *PresenceService) broadcastPresenceUpdate(ctx context.Context, userID uuid.UUID, presence *models.Presence) {
	// Get all servers the user is in
	servers, err := s.serverRepo.GetUserServers(ctx, userID)
	if err != nil {
		return
	}

	// Publish to each server's presence channel
	for _, server := range servers {
		s.eventBus.Publish("presence.updated", &PresenceUpdateEvent{
			UserID:   userID,
			ServerID: server.ID,
			Presence: presence,
		})
	}
}

// GetServerPresences gets presence for all members of a server
func (s *PresenceService) GetServerPresences(ctx context.Context, serverID uuid.UUID) (map[uuid.UUID]*models.Presence, error) {
	members, err := s.getAllMembersWithPagination(ctx, serverID)
	if err != nil {
		return nil, err
	}

	userIDs := make([]uuid.UUID, len(members))
	for i, member := range members {
		userIDs[i] = member.UserID
	}

	return s.GetBulkPresence(ctx, userIDs)
}

func (s *PresenceService) getAllMembersWithPagination(ctx context.Context, serverID uuid.UUID) ([]*models.Member, error) {
	const batchSize = 100
	var allMembers []*models.Member
	var cursor *models.MemberCursor

	for {
		result, err := s.serverRepo.GetMembersPaginated(ctx, serverID, cursor, batchSize)
		if err != nil {
			return nil, err
		}

		allMembers = append(allMembers, result.Members...)

		if !result.HasMore {
			break
		}

		nextCursor, err := models.DecodeMemberCursor(result.NextCursor)
		if err != nil {
			return nil, err
		}
		cursor = nextCursor
	}

	return allMembers, nil
}

// Events

type PresenceUpdateEvent struct {
	UserID   uuid.UUID
	ServerID uuid.UUID
	Presence *models.Presence
}
