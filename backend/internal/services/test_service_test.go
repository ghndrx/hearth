package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mock Implementations ---

// MockUserRepository acts as a database mock for tests.
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) FindByID(ctx context.Context, id int) (*User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*User), args.Error(1)
}

func (m *MockUserRepository) Create(ctx context.Context, user *User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

// MockChatService acts as a chat dispatcher mock.
type MockChatService struct {
	mock.Mock
}

func (m *MockChatService) GetMessages(ctx context.Context, channel string, limit int) ([]Message, error) {
	args := m.Called(ctx, channel, limit)
	return args.Get(0).([]Message), args.Error(1)
}

func (m *MockChatService) SendMessage(ctx context.Context, user *User, channel, content string) error {
	args := m.Called(ctx, user, channel, content)
	return args.Error(0)
}

// --- Tests ---

func TestNewTestService(t *testing.T) {
	repo := new(MockUserRepository)
	chat := new(MockChatService)

	service := NewTestService(repo, chat)

	assert.NotNil(t, service)
	assert.Same(t, repo, service.userRepo)
	assert.Same(t, chat, service.chatSvc)
}

func TestGetUserService(t *testing.T) {
	ctx := context.Background()
	userID := 123
	expectedUser := &User{
		ID:       userID,
		Username: "TestUser",
		Created:  time.Now(),
	}

	tests := []struct {
		name        string
		userID      int
		mockRepo    func(*MockUserRepository)
		expectedErr error
	}{
		{
			name:   "Happy Path: User found",
			userID: userID,
			mockRepo: func(r *MockUserRepository) {
				r.On("FindByID", ctx, userID).Return(expectedUser, nil)
			},
			expectedErr: nil,
		},
		{
			name:   "Error: User not found in repository",
			userID: 999,
			mockRepo: func(r *MockUserRepository) {
				r.On("FindByID", ctx, 999).Return(nil, errors.New("db error"))
			},
			expectedErr: errors.New("db error"),
		},
		{
			name:   "Boundary: Negative ID returns ErrUserNotFound",
			userID: -1,
			mockRepo: func(r *MockUserRepository) {
				// Should not be called
			},
			expectedErr: ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(MockUserRepository)
			tt.mockRepo(repo)
			chat := new(MockChatService)

			service := NewTestService(repo, chat)
			user, err := service.GetUserService(ctx, tt.userID)

			assert.Equal(t, tt.expectedErr, err)
			if tt.expectedErr == nil {
				assert.Equal(t, expectedUser, user)
				repo.AssertExpectations(t)
			} else {
				repo.AssertNotCalled(t, "FindByID") // Ensure we returned early
			}
		})
	}
}

func TestCreateChannelMessage(t *testing.T) {
	ctx := context.Background()
	userID := 1
	channelName := "general"
	content := "Hello World"
	validUser := &User{ID: userID, Username: "Felix"}

	tests := []struct {
		name        string
		userID      int
		channel     string
		content     string
		mockRepo    func(*MockUserRepository)
		mockChatSvc func(*MockChatService)
		expectedErr error
	}{
		{
			name:   "Happy Path: Message sent successfully",
			userID: userID,
			channel: channelName,
			content: content,
			mockRepo: func(r *MockUserRepository) {
				r.On("FindByID", ctx, userID).Return(validUser, nil)
			},
			mockChatSvc: func(c *MockChatService) {
				c.On("SendMessage", ctx, validUser, channelName, content).Return(nil)
			},
			expectedErr: nil,
		},
		{
			name:   "Error: Empty channel",
			userID: userID,
			channel: "",
			content: content,
			mockRepo: func(r *MockUserRepository) {
				// Never called
			},
			mockChatSvc: func(c *MockChatService) {},
			expectedErr: errors.New("channel and content cannot be empty"),
		},
		{
			name:   "Error: Empty content",
			userID: userID,
			channel: channelName,
			content: "",
			mockRepo: func(r *MockUserRepository) {
				// Never called
			},
			mockChatSvc: func(c *MockChatService) {},
			expectedErr: errors.New("channel and content cannot be empty"),
		},
		{
			name:   "Error: Service returns user error (e.g., unauthorized)",
			userID: 404,
			channel: channelName,
			content: content,
			mockRepo: func(r *MockUserRepository) {
				r.On("FindByID", ctx, 404).Return(nil, ErrUserNotFound)
			},
			mockChatSvc: func(c *MockChatService) {},
			expectedErr: ErrUserNotFound,
		},
		{
			name:   "Error: Chat service fails",
			userID: userID,
			channel: channelName,
			content: content,
			mockRepo: func(r *MockUserRepository) {
				r.On("FindByID", ctx, userID).Return(validUser, nil)
			},
			mockChatSvc: func(c *MockChatService) {
				c.On("SendMessage", ctx, validUser, channelName, content).Return(errors.New("network error"))
			},
			expectedErr: errors.New("network error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(MockUserRepository)
			chat := new(MockChatService)

			tt.mockRepo(repo)
			tt.mockChatSvc(chat)

			service := NewTestService(repo, chat)
			err := service.CreateChannelMessage(ctx, tt.userID, tt.channel, tt.content)

			assert.Equal(t, tt.expectedErr, err)

			if tt.expectedErr == nil {
				// Verify all mocks were called as expected
				repo.AssertExpectations(t)
				chat.AssertExpectations(t)
			}
		})
	}
}