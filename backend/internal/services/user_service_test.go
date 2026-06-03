package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"hearth/internal/models"
)

// MockUserRepository is a mock implementation of UserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserRepository) GetFriends(ctx context.Context, userID uuid.UUID) ([]*models.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockUserRepository) AddFriend(ctx context.Context, userID, friendID uuid.UUID) error {
	args := m.Called(ctx, userID, friendID)
	return args.Error(0)
}

func (m *MockUserRepository) RemoveFriend(ctx context.Context, userID, friendID uuid.UUID) error {
	args := m.Called(ctx, userID, friendID)
	return args.Error(0)
}

func (m *MockUserRepository) GetBlockedUsers(ctx context.Context, userID uuid.UUID) ([]*models.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockUserRepository) BlockUser(ctx context.Context, userID, blockedID uuid.UUID) error {
	args := m.Called(ctx, userID, blockedID)
	return args.Error(0)
}

func (m *MockUserRepository) UnblockUser(ctx context.Context, userID, blockedID uuid.UUID) error {
	args := m.Called(ctx, userID, blockedID)
	return args.Error(0)
}

func (m *MockUserRepository) UpdatePresence(ctx context.Context, userID uuid.UUID, status models.PresenceStatus) error {
	args := m.Called(ctx, userID, status)
	return args.Error(0)
}

func (m *MockUserRepository) GetPresence(ctx context.Context, userID uuid.UUID) (*models.Presence, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Presence), args.Error(1)
}

func (m *MockUserRepository) GetPresenceBulk(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*models.Presence, error) {
	args := m.Called(ctx, userIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[uuid.UUID]*models.Presence), args.Error(1)
}

func (m *MockUserRepository) GetRelationship(ctx context.Context, userID, targetID uuid.UUID) (int, error) {
	args := m.Called(ctx, userID, targetID)
	return args.Int(0), args.Error(1)
}

func (m *MockUserRepository) SendFriendRequest(ctx context.Context, senderID, receiverID uuid.UUID) error {
	args := m.Called(ctx, senderID, receiverID)
	return args.Error(0)
}

func (m *MockUserRepository) GetIncomingFriendRequests(ctx context.Context, userID uuid.UUID) ([]*models.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockUserRepository) GetOutgoingFriendRequests(ctx context.Context, userID uuid.UUID) ([]*models.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockUserRepository) AcceptFriendRequest(ctx context.Context, receiverID, senderID uuid.UUID) error {
	args := m.Called(ctx, receiverID, senderID)
	return args.Error(0)
}

func (m *MockUserRepository) DeclineFriendRequest(ctx context.Context, userID, otherID uuid.UUID) error {
	args := m.Called(ctx, userID, otherID)
	return args.Error(0)
}

func (m *MockUserRepository) GetCustomStatus(ctx context.Context, userID uuid.UUID) (*models.UserCustomStatus, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserCustomStatus), args.Error(1)
}

func (m *MockUserRepository) SetCustomStatus(ctx context.Context, status *models.UserCustomStatus) error {
	args := m.Called(ctx, status)
	return args.Error(0)
}

func (m *MockUserRepository) DeleteCustomStatus(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func setupUserService() (*UserService, *MockUserRepository, *MockCacheService, *MockEventBus) {
	repo := new(MockUserRepository)
	cache := new(MockCacheService)
	eventBus := new(MockEventBus)
	service := NewUserService(repo, cache, eventBus)
	return service, repo, cache, eventBus
}

func TestGetUser_Success(t *testing.T) {
	service, repo, cache, _ := setupUserService()
	ctx := context.Background()
	userID := uuid.New()

	expectedUser := &models.User{
		ID:        userID,
		Username:  "testuser",
		Email:     "test@example.com",
		CreatedAt: time.Now(),
	}

	// Cache miss
	cache.On("GetUser", ctx, userID).Return(nil, nil)
	repo.On("GetByID", ctx, userID).Return(expectedUser, nil)
	cache.On("SetUser", ctx, expectedUser, 5*time.Minute).Return(nil)

	user, err := service.GetUser(ctx, userID)

	assert.NoError(t, err)
	assert.Equal(t, expectedUser.Username, user.Username)
	repo.AssertExpectations(t)
	cache.AssertExpectations(t)
}

func TestGetUser_FromCache(t *testing.T) {
	service, repo, cache, _ := setupUserService()
	ctx := context.Background()
	userID := uuid.New()

	expectedUser := &models.User{
		ID:       userID,
		Username: "cacheduser",
	}

	// Cache hit
	cache.On("GetUser", ctx, userID).Return(expectedUser, nil)

	user, err := service.GetUser(ctx, userID)

	assert.NoError(t, err)
	assert.Equal(t, "cacheduser", user.Username)
	// Repo should not be called
	repo.AssertNotCalled(t, "GetByID")
}

func TestGetUser_NotFound(t *testing.T) {
	service, repo, cache, _ := setupUserService()
	ctx := context.Background()
	userID := uuid.New()

	cache.On("GetUser", ctx, userID).Return(nil, nil)
	repo.On("GetByID", ctx, userID).Return(nil, nil)

	user, err := service.GetUser(ctx, userID)

	assert.Error(t, err)
	assert.Equal(t, ErrUserNotFound, err)
	assert.Nil(t, user)
}

func TestUpdateUser_Success(t *testing.T) {
	service, repo, cache, eventBus := setupUserService()
	ctx := context.Background()
	userID := uuid.New()

	existingUser := &models.User{
		ID:       userID,
		Username: "oldname",
		Email:    "test@example.com",
	}

	newUsername := "newname"
	newBio := "New bio"
	updates := &models.UserUpdate{
		Username: &newUsername,
		Bio:      &newBio,
	}

	repo.On("GetByID", ctx, userID).Return(existingUser, nil)
	repo.On("GetByUsername", ctx, newUsername).Return(nil, nil)
	repo.On("Update", ctx, mock.AnythingOfType("*models.User")).Return(nil)
	cache.On("DeleteUser", ctx, userID).Return(nil)
	eventBus.On("Publish", "user.updated", mock.AnythingOfType("*services.UserUpdatedEvent")).Return()

	user, err := service.UpdateUser(ctx, userID, updates)

	assert.NoError(t, err)
	assert.Equal(t, newUsername, user.Username)
	assert.Equal(t, &newBio, user.Bio)
	repo.AssertExpectations(t)
	cache.AssertExpectations(t)
	eventBus.AssertExpectations(t)
}

func TestUpdateUser_UsernameTaken(t *testing.T) {
	service, repo, _, _ := setupUserService()
	ctx := context.Background()
	userID := uuid.New()

	existingUser := &models.User{
		ID:       userID,
		Username: "oldname",
	}

	takenUsername := "takenname"
	updates := &models.UserUpdate{
		Username: &takenUsername,
	}

	existingOther := &models.User{
		ID:       uuid.New(),
		Username: takenUsername,
	}

	repo.On("GetByID", ctx, userID).Return(existingUser, nil)
	repo.On("GetByUsername", ctx, takenUsername).Return(existingOther, nil)

	user, err := service.UpdateUser(ctx, userID, updates)

	assert.Error(t, err)
	assert.Equal(t, ErrUsernameTaken, err)
	assert.Nil(t, user)
}

func TestAddFriend_Success(t *testing.T) {
	service, repo, _, eventBus := setupUserService()
	ctx := context.Background()
	userID := uuid.New()
	friendID := uuid.New()

	repo.On("AddFriend", ctx, userID, friendID).Return(nil)
	eventBus.On("Publish", "friend.added", mock.AnythingOfType("*services.FriendAddedEvent")).Return()

	err := service.AddFriend(ctx, userID, friendID)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
	eventBus.AssertExpectations(t)
}

func TestAddFriend_CannotAddSelf(t *testing.T) {
	service, _, _, _ := setupUserService()
	ctx := context.Background()
	userID := uuid.New()

	err := service.AddFriend(ctx, userID, userID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot add yourself")
}

func TestBlockUser_Success(t *testing.T) {
	service, repo, _, eventBus := setupUserService()
	ctx := context.Background()
	userID := uuid.New()
	blockedID := uuid.New()

	repo.On("RemoveFriend", ctx, userID, blockedID).Return(nil)
	repo.On("BlockUser", ctx, userID, blockedID).Return(nil)
	eventBus.On("Publish", "user.blocked", mock.AnythingOfType("*services.UserBlockedEvent")).Return()

	err := service.BlockUser(ctx, userID, blockedID)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
	eventBus.AssertExpectations(t)
}

func TestBlockUser_CannotBlockSelf(t *testing.T) {
	service, _, _, _ := setupUserService()
	ctx := context.Background()
	userID := uuid.New()

	err := service.BlockUser(ctx, userID, userID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot block yourself")
}

func TestUpdatePresence_Success(t *testing.T) {
	service, repo, _, eventBus := setupUserService()
	ctx := context.Background()
	userID := uuid.New()
	status := models.StatusOnline

	repo.On("UpdatePresence", ctx, userID, status).Return(nil)
	eventBus.On("Publish", "presence.updated", mock.AnythingOfType("*services.PresenceUpdatedEvent")).Return()

	err := service.UpdatePresence(ctx, userID, status, nil)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
	eventBus.AssertExpectations(t)
}

func TestGetFriends_Success(t *testing.T) {
	service, repo, _, _ := setupUserService()
	ctx := context.Background()
	userID := uuid.New()

	expectedFriends := []*models.User{
		{ID: uuid.New(), Username: "friend1"},
		{ID: uuid.New(), Username: "friend2"},
	}

	repo.On("GetFriends", ctx, userID).Return(expectedFriends, nil)

	friends, err := service.GetFriends(ctx, userID)

	assert.NoError(t, err)
	assert.Len(t, friends, 2)
	assert.Equal(t, "friend1", friends[0].Username)
}

func TestGetUserByUsername_Success(t *testing.T) {
	service, repo, _, _ := setupUserService()
	ctx := context.Background()
	username := "testuser"

	expectedUser := &models.User{
		ID:       uuid.New(),
		Username: username,
		Email:    "test@example.com",
	}

	repo.On("GetByUsername", ctx, username).Return(expectedUser, nil)

	user, err := service.GetUserByUsername(ctx, username)

	assert.NoError(t, err)
	assert.Equal(t, username, user.Username)
	repo.AssertExpectations(t)
}

func TestGetUserByUsername_NotFound(t *testing.T) {
	service, repo, _, _ := setupUserService()
	ctx := context.Background()
	username := "nonexistent"

	repo.On("GetByUsername", ctx, username).Return(nil, nil)

	user, err := service.GetUserByUsername(ctx, username)

	assert.Error(t, err)
	assert.Equal(t, ErrUserNotFound, err)
	assert.Nil(t, user)
}

func TestRemoveFriend_Success(t *testing.T) {
	service, repo, _, eventBus := setupUserService()
	ctx := context.Background()
	userID := uuid.New()
	friendID := uuid.New()

	repo.On("RemoveFriend", ctx, userID, friendID).Return(nil)
	eventBus.On("Publish", "friend.removed", mock.AnythingOfType("*services.FriendRemovedEvent")).Return()

	err := service.RemoveFriend(ctx, userID, friendID)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
	eventBus.AssertExpectations(t)
}

func TestUnblockUser_Success(t *testing.T) {
	service, repo, _, eventBus := setupUserService()
	ctx := context.Background()
	userID := uuid.New()
	blockedID := uuid.New()

	repo.On("UnblockUser", ctx, userID, blockedID).Return(nil)
	eventBus.On("Publish", "user.unblocked", mock.AnythingOfType("*services.UserUnblockedEvent")).Return()

	err := service.UnblockUser(ctx, userID, blockedID)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
	eventBus.AssertExpectations(t)
}

// =========================================
// Friend Request System Tests
// =========================================

func TestSendFriendRequest_Success(t *testing.T) {
	service, repo, _, eventBus := setupUserService()
	ctx := context.Background()
	senderID := uuid.New()
	receiverID := uuid.New()

	receiver := &models.User{ID: receiverID, Username: "receiver"}

	repo.On("GetByID", ctx, receiverID).Return(receiver, nil)
	repo.On("GetRelationship", ctx, senderID, receiverID).Return(0, nil)
	repo.On("GetRelationship", ctx, receiverID, senderID).Return(0, nil)
	repo.On("SendFriendRequest", ctx, senderID, receiverID).Return(nil)
	eventBus.On("Publish", "friend.request_sent", mock.AnythingOfType("*services.FriendRequestSentEvent")).Return()

	err := service.SendFriendRequest(ctx, senderID, receiverID)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
	eventBus.AssertExpectations(t)
}

func TestSendFriendRequest_ToSelf(t *testing.T) {
	service, _, _, _ := setupUserService()
	ctx := context.Background()
	userID := uuid.New()

	err := service.SendFriendRequest(ctx, userID, userID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "yourself")
}

func TestSendFriendRequest_AlreadyFriends(t *testing.T) {
	service, repo, _, _ := setupUserService()
	ctx := context.Background()
	senderID := uuid.New()
	receiverID := uuid.New()

	receiver := &models.User{ID: receiverID, Username: "receiver"}

	repo.On("GetByID", ctx, receiverID).Return(receiver, nil)
	repo.On("GetRelationship", ctx, senderID, receiverID).Return(1, nil) // 1 = friends

	err := service.SendFriendRequest(ctx, senderID, receiverID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already friends")
}

func TestSendFriendRequest_AlreadySent(t *testing.T) {
	service, repo, _, _ := setupUserService()
	ctx := context.Background()
	senderID := uuid.New()
	receiverID := uuid.New()

	receiver := &models.User{ID: receiverID, Username: "receiver"}

	repo.On("GetByID", ctx, receiverID).Return(receiver, nil)
	repo.On("GetRelationship", ctx, senderID, receiverID).Return(4, nil) // 4 = pending outgoing

	err := service.SendFriendRequest(ctx, senderID, receiverID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already sent")
}

func TestSendFriendRequest_BlockedUser(t *testing.T) {
	service, repo, _, _ := setupUserService()
	ctx := context.Background()
	senderID := uuid.New()
	receiverID := uuid.New()

	receiver := &models.User{ID: receiverID, Username: "receiver"}

	repo.On("GetByID", ctx, receiverID).Return(receiver, nil)
	repo.On("GetRelationship", ctx, senderID, receiverID).Return(2, nil) // 2 = blocked

	err := service.SendFriendRequest(ctx, senderID, receiverID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "blocked")
}

func TestSendFriendRequest_ReceiverBlocked(t *testing.T) {
	service, repo, _, _ := setupUserService()
	ctx := context.Background()
	senderID := uuid.New()
	receiverID := uuid.New()

	receiver := &models.User{ID: receiverID, Username: "receiver"}

	repo.On("GetByID", ctx, receiverID).Return(receiver, nil)
	repo.On("GetRelationship", ctx, senderID, receiverID).Return(0, nil)
	repo.On("GetRelationship", ctx, receiverID, senderID).Return(2, nil) // Receiver blocked sender

	err := service.SendFriendRequest(ctx, senderID, receiverID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot send friend request")
}

func TestSendFriendRequest_AutoAcceptMutual(t *testing.T) {
	service, repo, _, eventBus := setupUserService()
	ctx := context.Background()
	senderID := uuid.New()
	receiverID := uuid.New()

	receiver := &models.User{ID: receiverID, Username: "receiver"}

	repo.On("GetByID", ctx, receiverID).Return(receiver, nil)
	repo.On("GetRelationship", ctx, senderID, receiverID).Return(3, nil) // 3 = pending incoming (they sent us a request)
	repo.On("AcceptFriendRequest", ctx, senderID, receiverID).Return(nil)
	eventBus.On("Publish", "friend.added", mock.AnythingOfType("*services.FriendAddedEvent")).Return()

	err := service.SendFriendRequest(ctx, senderID, receiverID)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
	eventBus.AssertExpectations(t)
}

func TestSendFriendRequest_UserNotFound(t *testing.T) {
	service, repo, _, _ := setupUserService()
	ctx := context.Background()
	senderID := uuid.New()
	receiverID := uuid.New()

	repo.On("GetByID", ctx, receiverID).Return(nil, nil)

	err := service.SendFriendRequest(ctx, senderID, receiverID)

	assert.Error(t, err)
	assert.Equal(t, ErrUserNotFound, err)
}

func TestGetIncomingFriendRequests_Success(t *testing.T) {
	service, repo, _, _ := setupUserService()
	ctx := context.Background()
	userID := uuid.New()

	incoming := []*models.User{
		{ID: uuid.New(), Username: "user1"},
		{ID: uuid.New(), Username: "user2"},
	}

	repo.On("GetIncomingFriendRequests", ctx, userID).Return(incoming, nil)

	result, err := service.GetIncomingFriendRequests(ctx, userID)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	repo.AssertExpectations(t)
}

func TestGetOutgoingFriendRequests_Success(t *testing.T) {
	service, repo, _, _ := setupUserService()
	ctx := context.Background()
	userID := uuid.New()

	outgoing := []*models.User{
		{ID: uuid.New(), Username: "user1"},
	}

	repo.On("GetOutgoingFriendRequests", ctx, userID).Return(outgoing, nil)

	result, err := service.GetOutgoingFriendRequests(ctx, userID)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	repo.AssertExpectations(t)
}

func TestAcceptFriendRequest_Success(t *testing.T) {
	service, repo, _, eventBus := setupUserService()
	ctx := context.Background()
	receiverID := uuid.New()
	senderID := uuid.New()

	repo.On("GetRelationship", ctx, receiverID, senderID).Return(3, nil) // 3 = pending incoming
	repo.On("AcceptFriendRequest", ctx, receiverID, senderID).Return(nil)
	eventBus.On("Publish", "friend.added", mock.AnythingOfType("*services.FriendAddedEvent")).Return()

	err := service.AcceptFriendRequest(ctx, receiverID, senderID)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
	eventBus.AssertExpectations(t)
}

func TestAcceptFriendRequest_NoPendingRequest(t *testing.T) {
	service, repo, _, _ := setupUserService()
	ctx := context.Background()
	receiverID := uuid.New()
	senderID := uuid.New()

	repo.On("GetRelationship", ctx, receiverID, senderID).Return(0, nil) // No relationship

	err := service.AcceptFriendRequest(ctx, receiverID, senderID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no pending friend request")
}

func TestAcceptFriendRequest_AlreadyFriends(t *testing.T) {
	service, repo, _, _ := setupUserService()
	ctx := context.Background()
	receiverID := uuid.New()
	senderID := uuid.New()

	repo.On("GetRelationship", ctx, receiverID, senderID).Return(1, nil) // Already friends

	err := service.AcceptFriendRequest(ctx, receiverID, senderID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no pending friend request")
}

func TestDeclineFriendRequest_IncomingSuccess(t *testing.T) {
	service, repo, _, eventBus := setupUserService()
	ctx := context.Background()
	userID := uuid.New()
	otherID := uuid.New()

	repo.On("GetRelationship", ctx, userID, otherID).Return(3, nil) // 3 = pending incoming
	repo.On("DeclineFriendRequest", ctx, userID, otherID).Return(nil)
	eventBus.On("Publish", "friend.request_declined", mock.AnythingOfType("*services.FriendRequestDeclinedEvent")).Return()

	err := service.DeclineFriendRequest(ctx, userID, otherID)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
	eventBus.AssertExpectations(t)
}

func TestDeclineFriendRequest_OutgoingSuccess(t *testing.T) {
	service, repo, _, eventBus := setupUserService()
	ctx := context.Background()
	userID := uuid.New()
	otherID := uuid.New()

	repo.On("GetRelationship", ctx, userID, otherID).Return(4, nil) // 4 = pending outgoing
	repo.On("DeclineFriendRequest", ctx, userID, otherID).Return(nil)
	eventBus.On("Publish", "friend.request_declined", mock.AnythingOfType("*services.FriendRequestDeclinedEvent")).Return()

	err := service.DeclineFriendRequest(ctx, userID, otherID)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
	eventBus.AssertExpectations(t)
}

func TestDeclineFriendRequest_NoPendingRequest(t *testing.T) {
	service, repo, _, _ := setupUserService()
	ctx := context.Background()
	userID := uuid.New()
	otherID := uuid.New()

	repo.On("GetRelationship", ctx, userID, otherID).Return(0, nil) // No relationship

	err := service.DeclineFriendRequest(ctx, userID, otherID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no pending friend request")
}

func TestGetRelationship_Success(t *testing.T) {
	service, repo, _, _ := setupUserService()
	ctx := context.Background()
	userID := uuid.New()
	targetID := uuid.New()

	repo.On("GetRelationship", ctx, userID, targetID).Return(1, nil) // Friends

	relType, err := service.GetRelationship(ctx, userID, targetID)

	assert.NoError(t, err)
	assert.Equal(t, 1, relType)
	repo.AssertExpectations(t)
}
type MockFriendRepository struct {
	mock.Mock
}

func (m *MockFriendRepository) Create(ctx context.Context, friendship *models.Friendship) error {
	args := m.Called(ctx, friendship)
	return args.Error(0)
}

func (m *MockFriendRepository) FetchByMembers(ctx context.Context, user1ID uuid.UUID, user2ID uuid.UUID) (*models.Friendship, error) {
	args := m.Called(ctx, user1ID, user2ID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Friendship), args.Error(1)
}

func (m *MockFriendRepository) ListFriends(ctx context.Context, userID uuid.UUID) ([]models.User, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]models.User), args.Error(1)
}

func (m *MockFriendRepository) Remove(ctx context.Context, friendshipID uuid.UUID) error {
	args := m.Called(ctx, friendshipID)
	return args.Error(0)
}

func (m *MockFriendRepository) PendingRequests(ctx context.Context, userID uuid.UUID) ([]models.Friendship, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]models.Friendship), args.Error(1)
}

// Test valid addition
func TestFriendService_AddFriend_Success(t *testing.T) {
	mockClient := new(MockFriendRepository)
	ctx := context.Background()
	userA := uuid.New()
	userB := uuid.New()
	mockClient.On("FetchByMembers", ctx, userA, userB).Return(nil, models.ErrRecordNotFound)
	mockClient.On("Create", ctx, mock.MatchedBy(func(f *models.Friendship) bool {
		return f.UserID1 == userA && f.UserID2 == userB
	})).Return(nil)

	service := NewFriendService(mockClient)
	err := service.AddFriend(ctx, userA, userB)

	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

// Test adding yourself
func TestAddFriend_Self(t *testing.T) {
	mockClient := new(MockFriendRepository)
	ctx := context.Background()
	userID := uuid.New()

	service := NewFriendService(mockClient)
	err := service.AddFriend(ctx, userID, userID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot add yourself")
	mockClient.AssertNotCalled(t, "FetchByMembers", mock.Anything, userID, userID)
}

// Test adding an existing friend
func TestAddFriend_AlreadyFriends(t *testing.T) {
	mockClient := new(MockFriendRepository)
	ctx := context.Background()
	userA := uuid.New()
	userB := uuid.New()
	existing := &models.Friendship{}

	mockClient.On("FetchByMembers", ctx, userA, userB).Return(existing, nil)

	service := NewFriendService(mockClient)
	err := service.AddFriend(ctx, userA, userB)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already friends")
	mockClient.AssertExpectations(t)
}

// Test ListFriends
func TestListFriends(t *testing.T) {
	mockClient := new(MockFriendRepository)
	ctx := context.Background()
	userID := uuid.New()
	expectedUsers := []models.User{
		{ID: uuid.New(), Username: "alice"},
		{ID: uuid.New(), Username: "bob"},
	}

	mockClient.On("ListFriends", ctx, userID).Return(expectedUsers, nil)

	service := NewFriendService(mockClient)
	users, err := service.ListFriends(ctx, userID)

	assert.NoError(t, err)
	assert.Len(t, users, 2)
	mockClient.AssertExpectations(t)
}
// MockStatusRepository is a test double for StatusRepository.
type MockStatusRepository struct {
	mock.Mock
}

func (m *MockStatusRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Status, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Status), args.Error(1)
}

func (m *MockStatusRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*models.Status, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Status), args.Error(1)
}

func (m *MockStatusRepository) Update(ctx context.Context, status *models.Status) error {
	args := m.Called(ctx, status)
	return args.Error(0)
}

func (m *MockStatusRepository) Create(ctx context.Context, status *models.Status) error {
	args := m.Called(ctx, status)
	return args.Error(0)
}

func TestStatusService_UpdateOrCreateStatus_Success(t *testing.T) {
	// Arrange
	userID := uuid.New()
	gameID := "valheim"
	activityDetails := "Entering the woods..."
	statusInput := models.Status{
		Status:          "Playing Valheim",
		GameID:          &gameID,
		ActivityDetails: &activityDetails,
	}

	// Mock existing status
	expectedStatus := &models.Status{
		ID:              uuid.New(),
		UserID:          userID,
		Status:          "Playing Valheim",
		GameID:          &gameID,
		ActivityDetails: &activityDetails,
	}

	mockRepo := new(MockStatusRepository)
	mockRepo.On("GetByUserID", mock.Anything, userID).Return(expectedStatus, nil)
	mockRepo.On("Update", mock.Anything, expectedStatus).Return(nil)

	service := NewStatusService(mockRepo)

	// Act
	result, err := service.UpdateOrCreateStatus(context.Background(), userID, statusInput)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	mockRepo.AssertExpectations(t)
}

func TestStatusService_UpdateOrCreateStatus_New(t *testing.T) {
	// Arrange
	userID := uuid.New()
	emptyGameID := ""
	emptyActivityDetails := ""
	statusInput := models.Status{
		Status:          "Online",
		GameID:          &emptyGameID,
		ActivityDetails: &emptyActivityDetails,
	}

	mockRepo := new(MockStatusRepository)
	// Mock not found (GetByUserID returns nil, nil)
	mockRepo.On("GetByUserID", mock.Anything, userID).Return(nil, nil)
	// Mock creation
	mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(s *models.Status) bool {
		return s.UserID == userID && s.Status == "Online"
	})).Return(nil)

	service := NewStatusService(mockRepo)

	// Act
	result, err := service.UpdateOrCreateStatus(context.Background(), userID, statusInput)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, statusInput.Status, result.Status)
	mockRepo.AssertExpectations(t)
}

func TestStatusService_UpdateOrCreateStatus_DBError(t *testing.T) {
	// Arrange
	userID := uuid.New()
	statusInput := models.Status{Status: "Offline"}

	mockRepo := new(MockStatusRepository)
	mockRepo.On("GetByUserID", mock.Anything, userID).Return(nil, errors.New("database connection lost"))

	service := NewStatusService(mockRepo)

	// Act
	result, err := service.UpdateOrCreateStatus(context.Background(), userID, statusInput)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	mockRepo.AssertExpectations(t)
}

func TestStatusService_GetUserStatus(t *testing.T) {
	// Arrange
	userID := uuid.New()
	expectedStatus := &models.Status{
		ID:     uuid.New(),
		UserID: userID,
		Status: "Playing Minecraft",
	}
	mockRepo := new(MockStatusRepository)
	mockRepo.On("GetByUserID", mock.Anything, userID).Return(expectedStatus, nil)

	service := NewStatusService(mockRepo)

	// Act
	result, err := service.GetUserStatus(context.Background(), userID)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expectedStatus, result)
	mockRepo.AssertExpectations(t)
}
// MockServerRepositoryForPresence is a mock for ServerRepository used in presence tests
type MockServerRepositoryForPresence struct {
	mock.Mock
}

func (m *MockServerRepositoryForPresence) Create(ctx context.Context, server *models.Server) error {
	args := m.Called(ctx, server)
	return args.Error(0)
}

func (m *MockServerRepositoryForPresence) GetByID(ctx context.Context, id uuid.UUID) (*models.Server, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Server), args.Error(1)
}

func (m *MockServerRepositoryForPresence) Update(ctx context.Context, server *models.Server) error {
	args := m.Called(ctx, server)
	return args.Error(0)
}

func (m *MockServerRepositoryForPresence) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockServerRepositoryForPresence) TransferOwnership(ctx context.Context, serverID, newOwnerID uuid.UUID) error {
	args := m.Called(ctx, serverID, newOwnerID)
	return args.Error(0)
}

func (m *MockServerRepositoryForPresence) GetMember(ctx context.Context, serverID, userID uuid.UUID) (*models.Member, error) {
	args := m.Called(ctx, serverID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Member), args.Error(1)
}

func (m *MockServerRepositoryForPresence) AddMember(ctx context.Context, member *models.Member) error {
	args := m.Called(ctx, member)
	return args.Error(0)
}

func (m *MockServerRepositoryForPresence) RemoveMember(ctx context.Context, serverID, userID uuid.UUID) error {
	args := m.Called(ctx, serverID, userID)
	return args.Error(0)
}

func (m *MockServerRepositoryForPresence) GetUserServers(ctx context.Context, userID uuid.UUID) ([]*models.Server, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Server), args.Error(1)
}

func (m *MockServerRepositoryForPresence) GetMembers(ctx context.Context, serverID uuid.UUID, limit, offset int) ([]*models.Member, error) {
	args := m.Called(ctx, serverID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Member), args.Error(1)
}

func (m *MockServerRepositoryForPresence) BanMember(ctx context.Context, ban *models.Ban) error {
	args := m.Called(ctx, ban)
	return args.Error(0)
}

func (m *MockServerRepositoryForPresence) UnbanMember(ctx context.Context, serverID, userID uuid.UUID) error {
	args := m.Called(ctx, serverID, userID)
	return args.Error(0)
}

func (m *MockServerRepositoryForPresence) GetBans(ctx context.Context, serverID uuid.UUID) ([]*models.Ban, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Ban), args.Error(1)
}

func (m *MockServerRepositoryForPresence) IsBanned(ctx context.Context, serverID, userID uuid.UUID) (bool, error) {
	args := m.Called(ctx, serverID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockServerRepositoryForPresence) UpdateMember(ctx context.Context, member *models.Member) error {
	args := m.Called(ctx, member)
	return args.Error(0)
}

func (m *MockServerRepositoryForPresence) GetMemberCount(ctx context.Context, serverID uuid.UUID) (int, error) {
	args := m.Called(ctx, serverID)
	return args.Int(0), args.Error(1)
}

func (m *MockServerRepositoryForPresence) GetOwnedServersCount(ctx context.Context, userID uuid.UUID) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

func (m *MockServerRepositoryForPresence) GetBan(ctx context.Context, serverID, userID uuid.UUID) (*models.Ban, error) {
	args := m.Called(ctx, serverID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Ban), args.Error(1)
}

func (m *MockServerRepositoryForPresence) AddBan(ctx context.Context, ban *models.Ban) error {
	args := m.Called(ctx, ban)
	return args.Error(0)
}

func (m *MockServerRepositoryForPresence) RemoveBan(ctx context.Context, serverID, userID uuid.UUID) error {
	args := m.Called(ctx, serverID, userID)
	return args.Error(0)
}

func (m *MockServerRepositoryForPresence) CreateInvite(ctx context.Context, invite *models.Invite) error {
	args := m.Called(ctx, invite)
	return args.Error(0)
}

func (m *MockServerRepositoryForPresence) GetInvite(ctx context.Context, code string) (*models.Invite, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Invite), args.Error(1)
}

func (m *MockServerRepositoryForPresence) GetInvites(ctx context.Context, serverID uuid.UUID) ([]*models.Invite, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Invite), args.Error(1)
}

func (m *MockServerRepositoryForPresence) DeleteInvite(ctx context.Context, code string) error {
	args := m.Called(ctx, code)
	return args.Error(0)
}

func (m *MockServerRepositoryForPresence) IncrementInviteUses(ctx context.Context, code string) error {
	args := m.Called(ctx, code)
	return args.Error(0)
}

func (m *MockServerRepositoryForPresence) GetMembersWithRole(ctx context.Context, serverID, roleID uuid.UUID) ([]*models.Member, error) {
	args := m.Called(ctx, serverID, roleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Member), args.Error(1)
}

func (m *MockServerRepositoryForPresence) GetMembersPaginated(ctx context.Context, serverID uuid.UUID, cursor *models.MemberCursor, limit int) (*models.PaginatedMembers, error) {
	args := m.Called(ctx, serverID, cursor, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.PaginatedMembers), args.Error(1)
}
func (m *MockServerRepositoryForPresence) GetInviteByVanityCode(ctx context.Context, vanityCode string) (*models.Invite, error) {
	return nil, nil
}
func (m *MockServerRepositoryForPresence) LogInviteUse(ctx context.Context, log *models.InviteUseLog) error {
	return nil
}
func (m *MockServerRepositoryForPresence) GetInviteUseLogs(ctx context.Context, inviteCode string) ([]models.InviteUseLog, error) {
	return nil, nil
}
func (m *MockServerRepositoryForPresence) GetServerInviteUseLogs(ctx context.Context, serverID uuid.UUID) ([]models.InviteUseLog, error) {
	return nil, nil
}

// ========== UpdatePresence Tests ==========

func TestPresenceService_UpdatePresence_Success(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	serverID := uuid.New()

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	// Setup expectations
	mockCache.On("Set", ctx, "presence:"+userID.String(), []byte(models.StatusOnline), presenceTTL).Return(nil)
	mockServerRepo.On("GetUserServers", ctx, userID).Return([]*models.Server{
		{ID: serverID, Name: "Test Server"},
	}, nil)
	mockEventBus.On("Publish", "presence.updated", mock.AnythingOfType("*services.PresenceUpdateEvent")).Return()

	// Execute
	err := service.UpdatePresence(ctx, userID, models.StatusOnline, nil, "desktop")

	// Assert
	assert.NoError(t, err)
	mockCache.AssertExpectations(t)
	mockServerRepo.AssertExpectations(t)
	mockEventBus.AssertExpectations(t)
}

func TestPresenceService_UpdatePresence_WithCustomStatus(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	customStatus := "Coding in Go 🚀"

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	mockCache.On("Set", ctx, "presence:"+userID.String(), []byte(models.StatusDND), presenceTTL).Return(nil)
	mockServerRepo.On("GetUserServers", ctx, userID).Return([]*models.Server{}, nil)

	err := service.UpdatePresence(ctx, userID, models.StatusDND, &customStatus, "web")

	assert.NoError(t, err)
	mockCache.AssertExpectations(t)
}

func TestPresenceService_UpdatePresence_CacheError(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	mockCache.On("Set", ctx, "presence:"+userID.String(), []byte(models.StatusOnline), presenceTTL).Return(errors.New("cache error"))

	err := service.UpdatePresence(ctx, userID, models.StatusOnline, nil, "mobile")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cache error")
}

func TestPresenceService_UpdatePresence_AllStatuses(t *testing.T) {
	testCases := []struct {
		name   string
		status models.PresenceStatus
	}{
		{"Online", models.StatusOnline},
		{"Idle", models.StatusIdle},
		{"DND", models.StatusDND},
		{"Invisible", models.StatusInvisible},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			userID := uuid.New()

			mockCache := new(MockCacheService)
			mockEventBus := new(MockEventBus)
			mockServerRepo := new(MockServerRepositoryForPresence)

			service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

			mockCache.On("Set", ctx, "presence:"+userID.String(), []byte(tc.status), presenceTTL).Return(nil)
			mockServerRepo.On("GetUserServers", ctx, userID).Return([]*models.Server{}, nil)

			err := service.UpdatePresence(ctx, userID, tc.status, nil, "desktop")

			assert.NoError(t, err)
		})
	}
}

// ========== GetPresence Tests ==========

func TestPresenceService_GetPresence_Online(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	mockCache.On("Get", ctx, "presence:"+userID.String()).Return([]byte(models.StatusOnline), nil)

	presence, err := service.GetPresence(ctx, userID)

	assert.NoError(t, err)
	assert.NotNil(t, presence)
	assert.Equal(t, userID, presence.UserID)
	assert.Equal(t, models.StatusOnline, presence.Status)
}

func TestPresenceService_GetPresence_Offline_CacheMiss(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	mockCache.On("Get", ctx, "presence:"+userID.String()).Return(nil, errors.New("cache miss"))

	presence, err := service.GetPresence(ctx, userID)

	assert.NoError(t, err) // Should not error, just return offline
	assert.NotNil(t, presence)
	assert.Equal(t, userID, presence.UserID)
	assert.Equal(t, models.StatusOffline, presence.Status)
}

func TestPresenceService_GetPresence_AllStatuses(t *testing.T) {
	testCases := []models.PresenceStatus{
		models.StatusOnline,
		models.StatusIdle,
		models.StatusDND,
		models.StatusInvisible,
	}

	for _, status := range testCases {
		t.Run(string(status), func(t *testing.T) {
			ctx := context.Background()
			userID := uuid.New()

			mockCache := new(MockCacheService)
			mockEventBus := new(MockEventBus)
			mockServerRepo := new(MockServerRepositoryForPresence)

			service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

			mockCache.On("Get", ctx, "presence:"+userID.String()).Return([]byte(status), nil)

			presence, err := service.GetPresence(ctx, userID)

			assert.NoError(t, err)
			assert.Equal(t, status, presence.Status)
		})
	}
}

// ========== GetBulkPresence Tests ==========

func TestPresenceService_GetBulkPresence_MultipleUsers(t *testing.T) {
	ctx := context.Background()
	user1 := uuid.New()
	user2 := uuid.New()
	user3 := uuid.New()

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	mockCache.On("Get", ctx, "presence:"+user1.String()).Return([]byte(models.StatusOnline), nil)
	mockCache.On("Get", ctx, "presence:"+user2.String()).Return([]byte(models.StatusIdle), nil)
	mockCache.On("Get", ctx, "presence:"+user3.String()).Return(nil, errors.New("not found"))

	result, err := service.GetBulkPresence(ctx, []uuid.UUID{user1, user2, user3})

	assert.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Equal(t, models.StatusOnline, result[user1].Status)
	assert.Equal(t, models.StatusIdle, result[user2].Status)
	assert.Equal(t, models.StatusOffline, result[user3].Status)
}

func TestPresenceService_GetBulkPresence_EmptyList(t *testing.T) {
	ctx := context.Background()

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	result, err := service.GetBulkPresence(ctx, []uuid.UUID{})

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestPresenceService_GetBulkPresence_SingleUser(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	mockCache.On("Get", ctx, "presence:"+userID.String()).Return([]byte(models.StatusDND), nil)

	result, err := service.GetBulkPresence(ctx, []uuid.UUID{userID})

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, models.StatusDND, result[userID].Status)
}

// ========== Heartbeat Tests ==========

func TestPresenceService_Heartbeat_ExtendsOnlineStatus(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	mockCache.On("Get", ctx, "presence:"+userID.String()).Return([]byte(models.StatusOnline), nil)
	mockCache.On("Set", ctx, "presence:"+userID.String(), []byte(models.StatusOnline), presenceTTL).Return(nil)

	err := service.Heartbeat(ctx, userID)

	assert.NoError(t, err)
	mockCache.AssertExpectations(t)
}

func TestPresenceService_Heartbeat_DefaultsToOnlineIfMissing(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	mockCache.On("Get", ctx, "presence:"+userID.String()).Return(nil, errors.New("not found"))
	mockCache.On("Set", ctx, "presence:"+userID.String(), []byte(models.StatusOnline), presenceTTL).Return(nil)

	err := service.Heartbeat(ctx, userID)

	assert.NoError(t, err)
	mockCache.AssertExpectations(t)
}

func TestPresenceService_Heartbeat_PreservesExistingStatus(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	mockCache.On("Get", ctx, "presence:"+userID.String()).Return([]byte(models.StatusDND), nil)
	mockCache.On("Set", ctx, "presence:"+userID.String(), []byte(models.StatusDND), presenceTTL).Return(nil)

	err := service.Heartbeat(ctx, userID)

	assert.NoError(t, err)
	mockCache.AssertExpectations(t)
}

func TestPresenceService_Heartbeat_CacheError(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	mockCache.On("Get", ctx, "presence:"+userID.String()).Return([]byte(models.StatusOnline), nil)
	mockCache.On("Set", ctx, "presence:"+userID.String(), []byte(models.StatusOnline), presenceTTL).Return(errors.New("cache error"))

	err := service.Heartbeat(ctx, userID)

	assert.Error(t, err)
}

// ========== SetOffline Tests ==========

func TestPresenceService_SetOffline_Success(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	serverID := uuid.New()

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	mockCache.On("Delete", ctx, "presence:"+userID.String()).Return(nil)
	mockServerRepo.On("GetUserServers", ctx, userID).Return([]*models.Server{
		{ID: serverID, Name: "Test Server"},
	}, nil)
	mockEventBus.On("Publish", "presence.updated", mock.AnythingOfType("*services.PresenceUpdateEvent")).Return()

	err := service.SetOffline(ctx, userID)

	assert.NoError(t, err)
	mockCache.AssertExpectations(t)
	mockEventBus.AssertExpectations(t)
}

func TestPresenceService_SetOffline_DeleteError_StillBroadcasts(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	// Even if delete fails, we should still broadcast
	mockCache.On("Delete", ctx, "presence:"+userID.String()).Return(errors.New("delete failed"))
	mockServerRepo.On("GetUserServers", ctx, userID).Return([]*models.Server{}, nil)

	err := service.SetOffline(ctx, userID)

	assert.NoError(t, err) // SetOffline doesn't propagate delete errors
}

func TestPresenceService_SetOffline_NoServers(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	mockCache.On("Delete", ctx, "presence:"+userID.String()).Return(nil)
	mockServerRepo.On("GetUserServers", ctx, userID).Return([]*models.Server{}, nil)

	err := service.SetOffline(ctx, userID)

	assert.NoError(t, err)
	mockEventBus.AssertNotCalled(t, "Publish", mock.Anything, mock.Anything)
}

// ========== TypingStart Tests ==========

func TestPresenceService_TypingStart_Success(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	channelID := uuid.New()

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	mockEventBus.On("Publish", "typing.started", mock.MatchedBy(func(indicator *models.TypingIndicator) bool {
		return indicator.UserID == userID && indicator.ChannelID == channelID
	})).Return()

	err := service.TypingStart(ctx, userID, channelID)

	assert.NoError(t, err)
	mockEventBus.AssertExpectations(t)
}

func TestPresenceService_TypingStart_SetsTimestamp(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	channelID := uuid.New()
	beforeCall := time.Now()

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	var capturedIndicator *models.TypingIndicator
	mockEventBus.On("Publish", "typing.started", mock.MatchedBy(func(indicator *models.TypingIndicator) bool {
		capturedIndicator = indicator
		return true
	})).Return()

	err := service.TypingStart(ctx, userID, channelID)

	assert.NoError(t, err)
	assert.NotNil(t, capturedIndicator)
	assert.False(t, capturedIndicator.Timestamp.Before(beforeCall))
	assert.False(t, capturedIndicator.Timestamp.After(time.Now()))
}

// ========== GetServerPresences Tests ==========

func TestPresenceService_GetServerPresences_Success(t *testing.T) {
	ctx := context.Background()
	serverID := uuid.New()
	user1 := uuid.New()
	user2 := uuid.New()

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	members := []*models.Member{
		{UserID: user1, ServerID: serverID},
		{UserID: user2, ServerID: serverID},
	}

	mockServerRepo.On("GetMembersPaginated", ctx, serverID, (*models.MemberCursor)(nil), 100).Return(&models.PaginatedMembers{
		Members:    members,
		NextCursor: "",
		HasMore:    false,
	}, nil)
	mockCache.On("Get", ctx, "presence:"+user1.String()).Return([]byte(models.StatusOnline), nil)
	mockCache.On("Get", ctx, "presence:"+user2.String()).Return([]byte(models.StatusIdle), nil)

	result, err := service.GetServerPresences(ctx, serverID)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, models.StatusOnline, result[user1].Status)
	assert.Equal(t, models.StatusIdle, result[user2].Status)
}

func TestPresenceService_GetServerPresences_EmptyServer(t *testing.T) {
	ctx := context.Background()
	serverID := uuid.New()

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	mockServerRepo.On("GetMembersPaginated", ctx, serverID, (*models.MemberCursor)(nil), 100).Return(&models.PaginatedMembers{
		Members:    []*models.Member{},
		NextCursor: "",
		HasMore:    false,
	}, nil)

	result, err := service.GetServerPresences(ctx, serverID)

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestPresenceService_GetServerPresences_RepoError(t *testing.T) {
	ctx := context.Background()
	serverID := uuid.New()

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	mockServerRepo.On("GetMembersPaginated", ctx, serverID, (*models.MemberCursor)(nil), 100).Return(nil, errors.New("database error"))

	result, err := service.GetServerPresences(ctx, serverID)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestPresenceService_GetServerPresences_MixedPresence(t *testing.T) {
	ctx := context.Background()
	serverID := uuid.New()
	user1 := uuid.New() // online
	user2 := uuid.New() // idle
	user3 := uuid.New() // dnd
	user4 := uuid.New() // offline (cache miss)

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	members := []*models.Member{
		{UserID: user1, ServerID: serverID},
		{UserID: user2, ServerID: serverID},
		{UserID: user3, ServerID: serverID},
		{UserID: user4, ServerID: serverID},
	}

	mockServerRepo.On("GetMembersPaginated", ctx, serverID, (*models.MemberCursor)(nil), 100).Return(&models.PaginatedMembers{
		Members:    members,
		NextCursor: "",
		HasMore:    false,
	}, nil)
	mockCache.On("Get", ctx, "presence:"+user1.String()).Return([]byte(models.StatusOnline), nil)
	mockCache.On("Get", ctx, "presence:"+user2.String()).Return([]byte(models.StatusIdle), nil)
	mockCache.On("Get", ctx, "presence:"+user3.String()).Return([]byte(models.StatusDND), nil)
	mockCache.On("Get", ctx, "presence:"+user4.String()).Return(nil, errors.New("not found"))

	result, err := service.GetServerPresences(ctx, serverID)

	assert.NoError(t, err)
	assert.Len(t, result, 4)
	assert.Equal(t, models.StatusOnline, result[user1].Status)
	assert.Equal(t, models.StatusIdle, result[user2].Status)
	assert.Equal(t, models.StatusDND, result[user3].Status)
	assert.Equal(t, models.StatusOffline, result[user4].Status)
}

// ========== BroadcastPresenceUpdate Tests ==========

func TestPresenceService_BroadcastPresenceUpdate_MultipleServers(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	server1 := uuid.New()
	server2 := uuid.New()
	server3 := uuid.New()

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	servers := []*models.Server{
		{ID: server1, Name: "Server 1"},
		{ID: server2, Name: "Server 2"},
		{ID: server3, Name: "Server 3"},
	}

	mockCache.On("Set", ctx, "presence:"+userID.String(), []byte(models.StatusOnline), presenceTTL).Return(nil)
	mockServerRepo.On("GetUserServers", ctx, userID).Return(servers, nil)
	mockEventBus.On("Publish", "presence.updated", mock.AnythingOfType("*services.PresenceUpdateEvent")).Return().Times(3)

	err := service.UpdatePresence(ctx, userID, models.StatusOnline, nil, "desktop")

	assert.NoError(t, err)
	mockEventBus.AssertNumberOfCalls(t, "Publish", 3)
}

func TestPresenceService_BroadcastPresenceUpdate_ServerRepoError(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	mockCache.On("Set", ctx, "presence:"+userID.String(), []byte(models.StatusOnline), presenceTTL).Return(nil)
	mockServerRepo.On("GetUserServers", ctx, userID).Return(nil, errors.New("db error"))

	// Even with server repo error, the main operation succeeds
	err := service.UpdatePresence(ctx, userID, models.StatusOnline, nil, "desktop")

	assert.NoError(t, err)
	// No events should be published when server lookup fails
	mockEventBus.AssertNotCalled(t, "Publish", mock.Anything, mock.Anything)
}

// ========== PresenceUpdateEvent Tests ==========

func TestPresenceUpdateEvent_Structure(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()
	presence := &models.Presence{
		UserID: userID,
		Status: models.StatusOnline,
	}

	event := &PresenceUpdateEvent{
		UserID:   userID,
		ServerID: serverID,
		Presence: presence,
	}

	assert.Equal(t, userID, event.UserID)
	assert.Equal(t, serverID, event.ServerID)
	assert.NotNil(t, event.Presence)
	assert.Equal(t, models.StatusOnline, event.Presence.Status)
}

// ========== Constants Tests ==========

func TestPresenceService_Constants(t *testing.T) {
	// Verify constants are reasonable
	assert.Equal(t, 2*time.Minute, presenceTTL)
	assert.Equal(t, 5*time.Minute, idleTimeout)
	assert.Equal(t, 30*time.Second, heartbeatInterval)
}

// ========== Integration-style Tests ==========

func TestPresenceService_FullLifecycle(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	serverID := uuid.New()

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	// 1. User comes online
	mockCache.On("Set", ctx, "presence:"+userID.String(), []byte(models.StatusOnline), presenceTTL).Return(nil)
	mockServerRepo.On("GetUserServers", ctx, userID).Return([]*models.Server{{ID: serverID}}, nil)
	mockEventBus.On("Publish", "presence.updated", mock.AnythingOfType("*services.PresenceUpdateEvent")).Return()

	err := service.UpdatePresence(ctx, userID, models.StatusOnline, nil, "desktop")
	assert.NoError(t, err)

	// 2. Check presence
	mockCache.On("Get", ctx, "presence:"+userID.String()).Return([]byte(models.StatusOnline), nil)
	presence, err := service.GetPresence(ctx, userID)
	assert.NoError(t, err)
	assert.Equal(t, models.StatusOnline, presence.Status)

	// 3. Heartbeat
	mockCache.On("Set", ctx, "presence:"+userID.String(), []byte(models.StatusOnline), presenceTTL).Return(nil)
	err = service.Heartbeat(ctx, userID)
	assert.NoError(t, err)

	// 4. User goes offline
	mockCache.On("Delete", ctx, "presence:"+userID.String()).Return(nil)
	err = service.SetOffline(ctx, userID)
	assert.NoError(t, err)
}
