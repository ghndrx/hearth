package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

// --- Mock Repository Setup ---

// We define the mock interfaces here or import them if they are in another package
// For this single-file example, we assume you have gomock installed or implement
// a very simple manual mock. Here is a basic manual mock implementation 
// for the code to run standalone without gomock:
type MockUser Repository struct{}

func NewMockUser(ctrl *gomock.Controller) *MockUserRepository {
	return &MockUserRepository{ctrl}
}

func (m *MockUserRepository) EXPECT() *MockUserRepositoryMockRecorder {
	return m.mockRecorder
}
func (m *MockUserRepository) GetByID(ctx context.Context, id string) (*User, error) { panic("implement method") }
func (m *MockUserRepository) Create(ctx context.Context, user *User) error { panic("implement method") }

type MockChatRepository struct{}
func NewMockChatRepository(ctrl *gomock.Controller) *MockChatRepository {
	return &MockChatRepository{ctrl}
}
func (m *MockChatRepository) EXPECT() *MockChatRepositoryMockRecorder { panic("implement method") }
func (m *MockChatRepository) GetMessagesByChannel(ctx context.Context, channelID string, limit int) ([]Message, error) { panic("implement method") }
func (m *MockChatRepository) SendMessage(ctx context.Context, channelID, userID, content string) (*Message, error) { panic("implement method") }

// If you do not want to use gomock, you can use a custom struct below:

// ManualMockUserRepo provides a mock implementation without gomock
type ManualMockUserRepo struct {
	CreateFunc func(ctx context.Context, user *User) error
}

func (m *ManualMockUserRepo) Create(ctx context.Context, user *User) error {
	return m.CreateFunc(ctx, user)
}

type ManualMockChatRepo struct {
	GetMessagesFunc func(ctx context.Context, channelID string, limit int) ([]Message, error)
	SendMessageFunc func(ctx context.Context, channelID string, userID string, content string) (*Message, error)
}

func (m *ManualMockChatRepo) GetMessagesByChannel(ctx context.Context, channelID string, limit int) ([]Message, error) {
	return m.GetMessagesFunc(ctx, channelID, limit)
}

func (m *ManualMockChatRepo) SendMessage(ctx context.Context, channelID, userID string, content string) (*Message, error) {
	return m.SendMessageFunc(ctx, channelID, userID, content)
}

// --- Actual Tests ---

func TestHeartService_CreateUser_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockUserRepo := &ManualMockUserRepo{}
	mockChatRepo := &ManualMockChatRepo{}
	service := NewHeartService(mockUserRepo, mockChatRepo)

	inputUser := "SniperWolf"
	inputEmail := "alex@example.com"

	mockUserRepo.CreateFunc = func(ctx context.Context, user *User) error {
		// Ensure ID is auto-generated
		assert.Equal(t, inputUser, user.Username)
		assert.Equal(t, trimSpace(inputEmail), user.Email)
		assert.Equal(t, ctx, ctx) // Verify context passed
		return nil
	}

	// Act
	user, err := service.CreateUser(ctx, inputUser, inputEmail)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, inputUser, user.Username)
	assert.Equal(t, trimSpace(inputEmail), user.Email)
}

func TestHeartService_CreateUser_EmptyInput(t *testing.T) {
	ctx := context.Background()
	service := NewHeartService(&ManualMockUserRepo{}, &ManualMockChatRepo{})

	// Test Empty Username
	_, err := service.CreateUser(ctx, "", "valid@email.com")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "username")

	// Test Empty Email
	_, err = service.CreateUser(ctx, "validUser", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "email")
}

func TestHeartService_SendAndRetrieveMessage_Success(t *testing.T) {
	ctx := context.Background()
	mockUserRepo := &ManualMockUserRepo{}
	mockChatRepo := &ManualMockChatRepo{}
	service := NewHeartService(mockUserRepo, mockChatRepo)

	ctxValue := "123"
	contentValue := "Hello world!"
	outputMsg := &Message{
		ID:        "msg_123",
		ChannelID: ctxValue,
		Content:   contentValue,
		TimeStamp: time.Now(),
	}

	// Mock Return
	mockChatRepo.SendMessageFunc = func(ctx context.Context, channelID, userID string, content string) (*Message, error) {
		return outputMsg, nil
	}
	mockChatRepo.GetMessagesByChannelFunc = func(ctx context.Context, channelID string, limit int) ([]Message, error) {
		return []Message{}, nil // Valid channel
	}

	// Act
	msg, err := service.SendAndRetrieveMessage(ctx, ctxValue, "user_1", contentValue)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, msg)
	assert.Equal(t, contentValue, msg.Content)
}

func TestHeartService_SendAndRetrieveMessage_EmptyContent(t *testing.T) {
	ctx := context.Background()
	service := NewHeartService(&ManualMockUserRepo{}, &ManualMockChatRepo{})

	_, err := service.SendAndRetrieveMessage(ctx, "123", "1", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "message content cannot be empty")
}

func TestHeartService_SendAndRetrieveMessage_ChannelNotFound(t *testing.T) {
	ctx := context.Background()
	mockUserRepo := &ManualMockUserRepo{}
	mockChatRepo := &ManualMockChatRepo{}
	service := NewHeartService(mockUserRepo, mockChatRepo)

	// Mock Return: Simulate a channel not existing
	mockChatRepo.GetMessagesByChannelFunc = func(ctx context.Context, channelID string, limit int) ([]Message, error) {
		return nil, errors.New("channel not found")
	}

	_, err := service.SendAndRetrieveMessage(ctx, "dead_channel", "user_1", "msg")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to retrieve channel")
}

func TestHeartService_TimeMocking(t *testing.T) {
	ctx := context.Background()
	service := NewHeartService(&ManualMockUserRepo{}, &ManualMockChatRepo{})

	// Switch time to a fixed point
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	service.SetTimeSource(func() time.Time {
		return fixedTime
	})

	// Act
	user, err := service.CreateUser(ctx, "TestUser", "test@test.com")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, fixedTime, user.CreatedAt)
}