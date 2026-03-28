// File: services/comprehensive_service_test.go
package services

import (
	"context"
	"testing"

	"hearth/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockcomprehensiveRepository is a mock implementation of comprehensiveRepository using testify/mock.
type MockcomprehensiveRepository struct {
	mock.Mock
}

// --- User Mock Methods ---
func (m *MockcomprehensiveRepository) CreateUser(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockcomprehensiveRepository) GetUserByID(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockcomprehensiveRepository) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

// --- Server Mock Methods ---
func (m *MockcomprehensiveRepository) CreateServer(ctx context.Context, server *models.Server) error {
	args := m.Called(ctx, server)
	return args.Error(0)
}

func (m *MockcomprehensiveRepository) GetServerByID(ctx context.Context, serverID uuid.UUID) (*models.Server, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Server), args.Error(1)
}

func (m *MockcomprehensiveRepository) AddMemberToServer(ctx context.Context, serverID, userID uuid.UUID) error {
	args := m.Called(ctx, serverID, userID)
	return args.Error(0)
}

func (m *MockcomprehensiveRepository) IsServerMember(ctx context.Context, serverID, userID uuid.UUID) (bool, error) {
	args := m.Called(ctx, serverID, userID)
	return args.Bool(0), args.Error(1)
}

// --- Channel Mock Methods ---
func (m *MockcomprehensiveRepository) CreateChannel(ctx context.Context, channel *models.Channel) error {
	args := m.Called(ctx, channel)
	return args.Error(0)
}

func (m *MockcomprehensiveRepository) GetChannelByID(ctx context.Context, channelID uuid.UUID) (*models.Channel, error) {
	args := m.Called(ctx, channelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Channel), args.Error(1)
}

func (m *MockcomprehensiveRepository) GetChannelsByServer(ctx context.Context, serverID uuid.UUID) ([]*models.Channel, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Channel), args.Error(1)
}

// --- Message Mock Methods ---
func (m *MockcomprehensiveRepository) CreateMessage(ctx context.Context, message *models.Message) error {
	args := m.Called(ctx, message)
	return args.Error(0)
}

func (m *MockcomprehensiveRepository) GetMessageByID(ctx context.Context, messageID uuid.UUID) (*models.Message, error) {
	args := m.Called(ctx, messageID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Message), args.Error(1)
}

func (m *MockcomprehensiveRepository) GetMessagesByChannel(ctx context.Context, channelID uuid.UUID, limit int) ([]*models.Message, error) {
	args := m.Called(ctx, channelID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Message), args.Error(1)
}

// --- Tests ---

func TestComprehensiveService_RegisterUser(t *testing.T) {
	mockRepo := new(MockcomprehensiveRepository)
	service := NewComprehensiveService(mockRepo)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		username := "testuser"
		email := "test@example.com"
		pass := "hash"

		// Mock chain: username check returns nil, create returns nil
		mockRepo.On("GetUserByUsername", ctx, username).Return(nil, nil).Once()
		mockRepo.On("CreateUser", ctx, mock.AnythingOfType("*models.User")).Return(nil).Once()

		user, err := service.RegisterUser(ctx, username, email, pass)

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, username, user.Username)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Username Taken", func(t *testing.T) {
		username := "takenuser"
		email := "test@example.com"
		pass := "hash"

		existingUser := &models.User{ID: uuid.New(), Username: username}
		mockRepo.On("GetUserByUsername", ctx, username).Return(existingUser, nil).Once()

		user, err := service.RegisterUser(ctx, username, email, pass)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, ErrUsernameTaken, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Invalid Input", func(t *testing.T) {
		user, err := service.RegisterUser(ctx, "", "", "")
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidInput, err)
		assert.Nil(t, user)
	})
}

func TestComprehensiveService_GetUser(t *testing.T) {
	mockRepo := new(MockcomprehensiveRepository)
	service := NewComprehensiveService(mockRepo)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		userID := uuid.New()
		expectedUser := &models.User{ID: userID, Username: "testuser"}

		mockRepo.On("GetUserByID", ctx, userID).Return(expectedUser, nil).Once()

		user, err := service.GetUser(ctx, userID)

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, expectedUser.ID, user.ID)
		assert.Equal(t, expectedUser.Username, user.Username)
		mockRepo.AssertExpectations(t)
	})

	t.Run("User Not Found", func(t *testing.T) {
		userID := uuid.New()

		mockRepo.On("GetUserByID", ctx, userID).Return(nil, ErrUserNotFound).Once()

		user, err := service.GetUser(ctx, userID)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, ErrUserNotFound, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestComprehensiveService_CreateServer(t *testing.T) {
	mockRepo := new(MockcomprehensiveRepository)
	service := NewComprehensiveService(mockRepo)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		ownerID := uuid.New()
		serverName := "Test Server"
		existingUser := &models.User{ID: ownerID}

		mockRepo.On("GetUserByID", ctx, ownerID).Return(existingUser, nil).Once()
		mockRepo.On("CreateServer", ctx, mock.AnythingOfType("*models.Server")).Return(nil).Once()
		mockRepo.On("AddMemberToServer", ctx, mock.AnythingOfType("uuid.UUID"), ownerID).Return(nil).Once()
		mockRepo.On("CreateChannel", ctx, mock.AnythingOfType("*models.Channel")).Return(nil).Once()

		server, err := service.CreateServer(ctx, serverName, ownerID)

		assert.NoError(t, err)
		assert.NotNil(t, server)
		assert.Equal(t, serverName, server.Name)
		assert.Equal(t, ownerID, server.OwnerID)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Owner Not Found", func(t *testing.T) {
		ownerID := uuid.New()
		serverName := "Test Server"

		mockRepo.On("GetUserByID", ctx, ownerID).Return(nil, ErrUserNotFound).Once()

		server, err := service.CreateServer(ctx, serverName, ownerID)

		assert.Error(t, err)
		assert.Nil(t, server)
		assert.Equal(t, ErrUserNotFound, err)
	})

	t.Run("Invalid Input - Empty Name", func(t *testing.T) {
		ownerID := uuid.New()

		server, err := service.CreateServer(ctx, "", ownerID)

		assert.Error(t, err)
		assert.Nil(t, server)
		assert.Equal(t, ErrInvalidInput, err)
	})
}

func TestComprehensiveService_SendMessage(t *testing.T) {
	mockRepo := new(MockcomprehensiveRepository)
	service := NewComprehensiveService(mockRepo)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		userID := uuid.New()
		serverID := uuid.New()
		channelID := uuid.New()
		content := "Hello World"

		mockChannel := &models.Channel{ID: channelID, ServerID: &serverID}

		mockRepo.On("GetChannelByID", ctx, channelID).Return(mockChannel, nil).Once()
		mockRepo.On("IsServerMember", ctx, serverID, userID).Return(true, nil).Once()
		mockRepo.On("CreateMessage", ctx, mock.AnythingOfType("*models.Message")).Return(nil).Once()

		msg, err := service.SendMessage(ctx, channelID, userID, content)

		assert.NoError(t, err)
		assert.NotNil(t, msg)
		assert.Equal(t, content, msg.Content)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		userID := uuid.New()
		serverID := uuid.New()
		channelID := uuid.New()

		mockChannel := &models.Channel{ID: channelID, ServerID: &serverID}

		mockRepo.On("GetChannelByID", ctx, channelID).Return(mockChannel, nil).Once()
		mockRepo.On("IsServerMember", ctx, serverID, userID).Return(false, nil).Once()

		msg, err := service.SendMessage(ctx, channelID, userID, "Hello")

		assert.Error(t, err)
		assert.Nil(t, msg)
		assert.Equal(t, ErrUnauthorizedAccess, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Empty Content", func(t *testing.T) {
		msg, err := service.SendMessage(context.Background(), uuid.New(), uuid.New(), "")
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidInput, err)
		assert.Nil(t, msg)
	})
}

func TestComprehensiveService_JoinServer(t *testing.T) {
	mockRepo := new(MockcomprehensiveRepository)
	service := NewComprehensiveService(mockRepo)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		userID := uuid.New()
		serverID := uuid.New()

		mockUser := &models.User{ID: userID}
		mockServer := &models.Server{ID: serverID}

		mockRepo.On("GetUserByID", ctx, userID).Return(mockUser, nil).Once()
		mockRepo.On("GetServerByID", ctx, serverID).Return(mockServer, nil).Once()
		mockRepo.On("IsServerMember", ctx, serverID, userID).Return(false, nil).Once()
		mockRepo.On("AddMemberToServer", ctx, serverID, userID).Return(nil).Once()

		err := service.JoinServer(ctx, serverID, userID)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Already Member", func(t *testing.T) {
		userID := uuid.New()
		serverID := uuid.New()

		mockUser := &models.User{ID: userID}
		mockServer := &models.Server{ID: serverID}

		mockRepo.On("GetUserByID", ctx, userID).Return(mockUser, nil).Once()
		mockRepo.On("GetServerByID", ctx, serverID).Return(mockServer, nil).Once()
		mockRepo.On("IsServerMember", ctx, serverID, userID).Return(true, nil).Once()

		err := service.JoinServer(ctx, serverID, userID)
		assert.Error(t, err)
		assert.Equal(t, ErrAlreadyMember, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestComprehensiveService_CreateChannel(t *testing.T) {
	mockRepo := new(MockcomprehensiveRepository)
	service := NewComprehensiveService(mockRepo)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		serverID := uuid.New()
		channelName := "general"
		channelType := "text"

		mockServer := &models.Server{ID: serverID}

		mockRepo.On("GetServerByID", ctx, serverID).Return(mockServer, nil).Once()
		mockRepo.On("CreateChannel", ctx, mock.AnythingOfType("*models.Channel")).Return(nil).Once()

		channel, err := service.CreateChannel(ctx, serverID, channelName, channelType)

		assert.NoError(t, err)
		assert.NotNil(t, channel)
		assert.Equal(t, channelName, channel.Name)
		assert.Equal(t, models.ChannelType(channelType), channel.Type)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Server Not Found", func(t *testing.T) {
		serverID := uuid.New()

		mockRepo.On("GetServerByID", ctx, serverID).Return(nil, ErrServerNotFound).Once()

		channel, err := service.CreateChannel(ctx, serverID, "general", "text")

		assert.Error(t, err)
		assert.Nil(t, channel)
		assert.Equal(t, ErrServerNotFound, err)
	})

	t.Run("Invalid Input", func(t *testing.T) {
		serverID := uuid.New()
		mockServer := &models.Server{ID: serverID}

		mockRepo.On("GetServerByID", ctx, serverID).Return(mockServer, nil).Once()

		channel, err := service.CreateChannel(ctx, serverID, "", "text")

		assert.Error(t, err)
		assert.Nil(t, channel)
		assert.Equal(t, ErrInvalidInput, err)
	})
}

func TestComprehensiveService_GetChannels(t *testing.T) {
	mockRepo := new(MockcomprehensiveRepository)
	service := NewComprehensiveService(mockRepo)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		serverID := uuid.New()
		expectedChannels := []*models.Channel{
			{ID: uuid.New(), ServerID: &serverID, Name: "general"},
			{ID: uuid.New(), ServerID: &serverID, Name: "random"},
		}

		mockRepo.On("GetChannelsByServer", ctx, serverID).Return(expectedChannels, nil).Once()

		channels, err := service.GetChannels(ctx, serverID)

		assert.NoError(t, err)
		assert.Len(t, channels, 2)
		mockRepo.AssertExpectations(t)
	})
}

func TestComprehensiveService_GetMessages(t *testing.T) {
	mockRepo := new(MockcomprehensiveRepository)
	service := NewComprehensiveService(mockRepo)
	ctx := context.Background()

	t.Run("Success with Default Limit", func(t *testing.T) {
		channelID := uuid.New()
		expectedMessages := []*models.Message{
			{ID: uuid.New(), ChannelID: channelID, Content: "Hello"},
			{ID: uuid.New(), ChannelID: channelID, Content: "World"},
		}

		mockRepo.On("GetMessagesByChannel", ctx, channelID, 50).Return(expectedMessages, nil).Once()

		messages, err := service.GetMessages(ctx, channelID, 0)

		assert.NoError(t, err)
		assert.Len(t, messages, 2)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Success with Custom Limit", func(t *testing.T) {
		channelID := uuid.New()
		expectedMessages := []*models.Message{
			{ID: uuid.New(), ChannelID: channelID, Content: "Test"},
		}

		mockRepo.On("GetMessagesByChannel", ctx, channelID, 10).Return(expectedMessages, nil).Once()

		messages, err := service.GetMessages(ctx, channelID, 10)

		assert.NoError(t, err)
		assert.Len(t, messages, 1)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Capped at 100", func(t *testing.T) {
		channelID := uuid.New()

		mockRepo.On("GetMessagesByChannel", ctx, channelID, 100).Return([]*models.Message{}, nil).Once()

		messages, err := service.GetMessages(ctx, channelID, 200)

		assert.NoError(t, err)
		assert.Empty(t, messages)
		mockRepo.AssertExpectations(t)
	})
}
