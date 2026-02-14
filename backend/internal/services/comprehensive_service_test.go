package services

import (
	"context"
	"testing"

	"hearth/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockcomprehensiveRepository is a mock implementation of comprehensiveRepository.
type MockcomprehensiveRepository struct {
	mock.Mock
}

// User Methods
func (m *MockcomprehensiveRepository) CreateUser(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockcomprehensiveRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	args := m.Called(ctx, id)
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

// Server Methods
func (m *MockcomprehensiveRepository) CreateServer(ctx context.Context, server *models.Server) (*models.Server, error) {
	args := m.Called(ctx, server)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Server), args.Error(1)
}

func (m *MockcomprehensiveRepository) GetServerByID(ctx context.Context, id uuid.UUID) (*models.Server, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Server), args.Error(1)
}

func (m *MockcomprehensiveRepository) AddMemberToServer(ctx context.Context, serverID, userID uuid.UUID) error {
	args := m.Called(ctx, serverID, userID)
	return args.Error(0)
}

func (m *MockcomprehensiveRepository) IsMemberOfServer(ctx context.Context, serverID, userID uuid.UUID) (bool, error) {
	args := m.Called(ctx, serverID, userID)
	return args.Bool(0), args.Error(1)
}

// Message Methods
func (m *MockcomprehensiveRepository) CreateMessage(ctx context.Context, message *models.Message) (*models.Message, error) {
	args := m.Called(ctx, message)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Message), args.Error(1)
}

func (m *MockcomprehensiveRepository) GetMessagesByChannel(ctx context.Context, channelID uuid.UUID, limit, offset int) ([]*models.Message, error) {
	args := m.Called(ctx, channelID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Message), args.Error(1)
}

// --- Tests ---

func TestComprehensiveService_RegisterUser_Success(t *testing.T) {
	mockRepo := new(MockcomprehensiveRepository)
	service := NewComprehensiveService(mockRepo)
	ctx := context.Background()

	username := "testuser"
	email := "test@example.com"

	// Setup expectations
	mockRepo.On("GetUserByUsername", ctx, username).Return((*models.User)(nil), nil) // No existing user
	mockRepo.On("CreateUser", ctx, mock.AnythingOfType("*models.User")).Return(nil)

	user, err := service.RegisterUser(ctx, username, email)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, username, user.Username)
	assert.Equal(t, email, user.Email)
	mockRepo.AssertExpectations(t)
}

func TestComprehensiveService_RegisterUser_AlreadyExists(t *testing.T) {
	mockRepo := new(MockcomprehensiveRepository)
	service := NewComprehensiveService(mockRepo)
	ctx := context.Background()

	username := "testuser"
	email := "test@example.com"
	existingUser := &models.User{ID: uuid.New(), Username: username}

	// Setup expectations: User found means conflict
	mockRepo.On("GetUserByUsername", ctx, username).Return(existingUser, nil)

	user, err := service.RegisterUser(ctx, username, email)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Equal(t, ErrAlreadyExists, err)
	mockRepo.AssertExpectations(t)
}

func TestComprehensiveService_RegisterUser_InvalidInput(t *testing.T) {
	mockRepo := new(MockcomprehensiveRepository)
	service := NewComprehensiveService(mockRepo)
	ctx := context.Background()

	_, err := service.RegisterUser(ctx, "", "test@example.com")
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidInput, err)

	_, err = service.RegisterUser(ctx, "testuser", "")
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidInput, err)
}

func TestComprehensiveService_CreateServer_Success(t *testing.T) {
	mockRepo := new(MockcomprehensiveRepository)
	service := NewComprehensiveService(mockRepo)
	ctx := context.Background()

	ownerID := uuid.New()
	serverName := "Test Server"
	ownerUser := &models.User{ID: ownerID}
	expectedServer := &models.Server{ID: uuid.New(), Name: serverName, OwnerID: ownerID}

	mockRepo.On("GetUserByID", ctx, ownerID).Return(ownerUser, nil)
	mockRepo.On("CreateServer", ctx, mock.AnythingOfType("*models.Server")).Return(expectedServer, nil)
	mockRepo.On("AddMemberToServer", ctx, expectedServer.ID, ownerID).Return(nil)

	server, err := service.CreateServer(ctx, serverName, ownerID)

	assert.NoError(t, err)
	assert.NotNil(t, server)
	assert.Equal(t, serverName, server.Name)
	assert.Equal(t, ownerID, server.OwnerID)
	mockRepo.AssertExpectations(t)
}

func TestComprehensiveService_CreateServer_OwnerNotFound(t *testing.T) {
	mockRepo := new(MockcomprehensiveRepository)
	service := NewComprehensiveService(mockRepo)
	ctx := context.Background()

	ownerID := uuid.New()
	
	mockRepo.On("GetUserByID", ctx, ownerID).Return((*models.User)(nil), ErrNotFound)

	_, err := service.CreateServer(ctx, "Some Server", ownerID)

	assert.Error(t, err)
	assert.Equal(t, ErrNotFound, err)
	mockRepo.AssertExpectations(t)
}

func TestComprehensiveService_JoinServer_Success(t *testing.T) {
	mockRepo := new(MockcomprehensiveRepository)
	service := NewComprehensiveService(mockRepo)
	ctx := context.Background()

	serverID := uuid.New()
	userID := uuid.New()
	targetServer := &models.Server{ID: serverID}

	mockRepo.On("GetServerByID", ctx, serverID).Return(targetServer, nil)
	mockRepo.On("IsMemberOfServer", ctx, serverID, userID).Return(false, nil)
	mockRepo.On("AddMemberToServer", ctx, serverID, userID).Return(nil)

	err := service.JoinServer(ctx, serverID, userID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestComprehensiveService_JoinServer_AlreadyMember(t *testing.T) {
	mockRepo := new(MockcomprehensiveRepository)
	service := NewComprehensiveService(mockRepo)
	ctx := context.Background()

	serverID := uuid.New()
	userID := uuid.New()
	targetServer := &models.Server{ID: serverID}

	mockRepo.On("GetServerByID", ctx, serverID).Return(targetServer, nil)
	mockRepo.On("IsMemberOfServer", ctx, serverID, userID).Return(true, nil)

	err := service.JoinServer(ctx, serverID, userID)

	assert.Error(t, err)
	assert.Equal(t, ErrAlreadyExists, err)
	// AddMemberToServer should NOT be called
	mockRepo.AssertNotCalled(t, "AddMemberToServer", ctx, serverID, userID)
	mockRepo.AssertExpectations(t)
}

func TestComprehensiveService_SendMessage_Success(t *testing.T) {
	mockRepo := new(MockcomprehensiveRepository)
	service := NewComprehensiveService(mockRepo)
	ctx := context.Background()

	channelID := uuid.New()
	authorID := uuid.New()
	content := "Hello, World!"
	author := &models.User{ID: authorID}
	expectedMsg := &models.Message{
		ID:        uuid.New(),
		ChannelID: channelID,
		AuthorID:  authorID,
		Content:   content,
	}

	mockRepo.On("GetUserByID", ctx, authorID).Return(author, nil)
	mockRepo.On("CreateMessage", ctx, mock.AnythingOfType("*models.Message")).Return(expectedMsg, nil)

	msg, err := service.SendMessage(ctx, channelID, authorID, content)

	assert.NoError(t, err)
	assert.NotNil(t, msg)
	assert.Equal(t, content, msg.Content)
	mockRepo.AssertExpectations(t)
}

func TestComprehensiveService_SendMessage_UserNotFound(t *testing.T) {
	mockRepo := new(MockcomprehensiveRepository)
	service := NewComprehensiveService(mockRepo)
	ctx := context.Background()

	channelID := uuid.New()
	authorID := uuid.New()

	mockRepo.On("GetUserByID", ctx, authorID).Return((*models.User)(nil), ErrNotFound)

	_, err := service.SendMessage(ctx, channelID, authorID, "content")

	assert.Error(t, err)
	assert.Equal(t, ErrNotFound, err)
	mockRepo.AssertNotCalled(t, "CreateMessage", ctx, mock.Anything)
	mockRepo.AssertExpectations(t)
}