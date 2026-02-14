package services

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"discord-clone/pkg/models"
)

func setupMockDB(t *testing.T) (FriendRepository, UserRepository, DBTransactions) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %v", err)
	}
	return NewFriendRepository(db), NewUserRepository(db), db // In real code, you implement these concrete repositories
}

func TestFriendService_SendRequest_Success(t *testing.T) {
	// Arrange
	fromID := int64(1)
	username := "john_doe"
	recipientID := int64(2)

	mockRepo := &mockFriendRepository{}
	mockUserRepo := &mockUserRepository{}
	mockTx := &mockTransaction{}

	serv := NewFriendService(mockRepo, mockUserRepo, mockTx)

	// Define expectations mockRepo.GetUserIDsByName -> returns 2
	mockRepo.On("GetUserIDsByName", mock.Anything, username).Return([]int64{recipientID}, nil)
	mockTx.On("BeginTx", mock.Anything).Return(nil, nil)
	// mockRepo.InsertRequest and mockTx.Commit would be expected here in full implementation

	// Act
	err := serv.SendRequest(context.Background(), fromID, SendRequestInput{Username: username})

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestFriendService_SendRequest_ToSelf(t *testing.T) {
	serv := NewFriendService(nil, nil, nil)
	
	// Act
	err := serv.SendRequest(context.Background(), 1, SendRequestInput{Username: "me"})

	// Assert
	assert.Error(t, err)
	assert.Equal(t, ErrSelfRequest, err)
}

func TestFriendService_SendRequest_AlreadyFriends(t *testing.T) {
	// Arrange
	fromID := int64(1)
	targetID := int64(2)

	mockRepo := &mockFriendRepository{}
	mockUserRepo := &mockUserRepository{}
	
	serv := NewFriendService(mockRepo, mockUserRepo, nil)

	// Return existing friends
	mockRepo.On("GetFriendsList", mock.Anything, fromID).Return([]models.Friend{
		{ID: targetID, Name: "Target"},
	}, nil)

	// Act
	err := serv.SendRequest(context.Background(), fromID, SendRequestInput{Username: "Target"})

	// Assert
	assert.Error(t, err)
	assert.Equal(t, ErrAlreadyFriends, err)
}

func TestFriendService_AcceptRequest_Success(t *testing.T) {
	// Arrange
	senderID := int64(10)
	recipientID := int64(20)
	ctx := context.Background()

	mockDB := setupMockDB(t)
	mockRepo := &mockFriendRepository{}
	mockUserRepo := &mockUserRepository{}

	// Mock expectations
	// 1. Find Request
	mockRepo.On("GetFriendRequest", ctx, senderID, recipientID).Return(&models.FriendRequest{
		Status: models.RequestPending,
	}, nil)
	// 2. Create Friend Entry
	mockRepo.On("CreateFriend", ctx, mock.Anything, recipientID, senderID).Return(nil)
	// 3. Update Request Status (Optional if implemented in 1 step)

	// Assuming mockRepo is used as the concrete implementation here for simplicity
	serv := NewFriendService(mockRepo, mockUserRepo, mockDB)

	// Act
	err := serv.AcceptRequest(ctx, recipientID, senderID)

	// Assert
	assert.NoError(t, err)
}

func TestFriendService_RetrieveFriends_Success(t *testing.T) {
	ctx := context.Background()
	userID := int64(99)

	// Expected Data
	friends := []models.Friend{
		{ID: 1, Name: "Alice"},
		{ID: 2, Name: "Bob"},
	}

	mockRepo := &mockFriendRepository{}
	// Return mock friends
	mockRepo.On("GetFriendsList", ctx, userID).Return(friends, nil)

	serv := NewFriendService(mockRepo, nil, nil)

	// Act
	result, err := serv.GetFriends(ctx, userID)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, friends, result)
}

// --- Mock Repositories & Interfaces for Testing ---

type mockFriendRepository struct{}

func (m *mockFriendRepository) GetUserIDsByName(ctx context.Context, username string) ([]int64, error) { return nil, nil }
func (m *mockFriendRepository) GetFriendsList(ctx context.Context, userID int64) ([]models.Friend, error) { return nil, nil }
func (m *mockFriendRepository) GetFriendRequest(ctx context.Context, senderID, recipientID int64) (*models.FriendRequest, error) { return nil, nil }
func (m *mockFriendRepository) GetUserByID(ctx context.Context, id int64) (*models.User, error) { return nil, nil }

// Method names below are placeholders representing logic used in the service
func (m *mockFriendRepository) CreateFriend(ctx context.Context, tx *sql.Tx, userID, friendID int64) error { return nil }
// Note: These are not explicitly called in test file above if generic mocks are used, 
// but implied.

// Add method stubs so mockRepo satisfies interface expectation in AcceptRequest test
func (m *mockFriendRepository) GetFriendRequest(ctx context.Context, senderID, recipientID int64) (*models.FriendRequest, error) {
	return &models.FriendRequest{ID: senderID, ReceiverID: recipientID, SenderID: senderID, Status: models.RequestPending}, nil
}

type mockUserRepository struct{}
func (m *mockUserRepository) GetUserByID(ctx context.Context, id int64) (*models.User, error) { return nil, nil }

type mockTransaction struct{}
func (m *mockTransaction) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return nil, nil
}