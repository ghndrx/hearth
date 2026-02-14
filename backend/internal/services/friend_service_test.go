package services

import (
	"context"
	"errors"
	"testing"
	
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"hearth/internal/models"
)

// MockFriendRepository implements the FriendRepository interface for testing.
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
func TestAddFriend_Success(t *testing.T) {
	mockClient := new(MockFriendRepository)
	ctx := context.Background()
	userA := uuid.New()
	userB := uuid.New()
	expectedFriendship := &models.Friendship{
		ID:    uuid.New(),
		UserID1: userA,
		UserID2: userB,
		CreatedAt: models.Now(),
	}

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