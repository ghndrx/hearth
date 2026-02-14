package services

import (
	"context"
	"errors"
	"testing"

	"hearth/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockServerRepository is a mock implementation of ServerRepository.
type MockServerRepository struct {
	mock.Mock
}

func (m *MockServerRepository) Create(ctx context.Context, server *models.Server) error {
	args := m.Called(ctx, server)
	return args.Error(0)
}

func (m *MockServerRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Server, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Server), args.Error(1)
}

func (m *MockServerRepository) GetByOwner(ctx context.Context, ownerID uuid.UUID) ([]*models.Server, error) {
	args := m.Called(ctx, ownerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Server), args.Error(1)
}

func (m *MockServerRepository) Update(ctx context.Context, server *models.Server) error {
	args := m.Called(ctx, server)
	return args.Error(0)
}

func (m *MockServerRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestServerService_CreateServer(t *testing.T) {
	mockRepo := new(MockServerRepository)
	service := NewServerService(mockRepo)
	ctx := context.Background()

	ownerID := uuid.New()
	serverName := "Test Server"

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Server")).Return(nil).Once()

		server, err := service.CreateServer(ctx, serverName, ownerID)

		assert.NoError(t, err)
		assert.NotNil(t, server)
		assert.Equal(t, serverName, server.Name)
		assert.Equal(t, ownerID, server.OwnerID)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Empty Name", func(t *testing.T) {
		_, err := service.CreateServer(ctx, "", ownerID)

		assert.Error(t, err)
		assert.EqualError(t, err, "server name cannot be empty")
	})

	t.Run("Repository Error", func(t *testing.T) {
		dbErr := errors.New("database connection failed")
		mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Server")).Return(dbErr).Once()

		_, err := service.CreateServer(ctx, serverName, ownerID)

		assert.Error(t, err)
		assert.Equal(t, dbErr, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestServerService_GetServer(t *testing.T) {
	mockRepo := new(MockServerRepository)
	service := NewServerService(mockRepo)
	ctx := context.Background()

	serverID := uuid.New()
	expectedServer := &models.Server{ID: serverID, Name: "Go Guild"}

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("GetByID", ctx, serverID).Return(expectedServer, nil).Once()

		server, err := service.GetServer(ctx, serverID)

		assert.NoError(t, err)
		assert.Equal(t, expectedServer, server)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Not Found", func(t *testing.T) {
		mockRepo.On("GetByID", ctx, serverID).Return(nil, ErrServerNotFound).Once()

		server, err := service.GetServer(ctx, serverID)

		assert.Error(t, err)
		assert.Nil(t, server)
		assert.Equal(t, ErrServerNotFound, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestServerService_DeleteServer(t *testing.T) {
	mockRepo := new(MockServerRepository)
	service := NewServerService(mockRepo)
	ctx := context.Background()

	serverID := uuid.New()
	existingServer := &models.Server{ID: serverID, Name: "To Be Deleted"}

	t.Run("Success", func(t *testing.T) {
		// Setup: Get returns server (existence check), Delete returns nil
		mockRepo.On("GetByID", ctx, serverID).Return(existingServer, nil).Once()
		mockRepo.On("Delete", ctx, serverID).Return(nil).Once()

		err := service.DeleteServer(ctx, serverID)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Server Not Found", func(t *testing.T) {
		mockRepo.On("GetByID", ctx, serverID).Return(nil, ErrServerNotFound).Once()

		err := service.DeleteServer(ctx, serverID)

		assert.Error(t, err)
		assert.Equal(t, ErrServerNotFound, err)
		mockRepo.AssertExpectations(t)
	})
}