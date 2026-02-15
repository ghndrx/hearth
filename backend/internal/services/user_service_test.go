package services

import (
	"context"
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
