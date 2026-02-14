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

// MockWebSocketRepository is a mock implementation of WebSocketRepository for testing.
type MockWebSocketRepository struct {
	mock.Mock
}

func (m *MockWebSocketRepository) SaveActiveSession(ctx context.Context, session *models.Session) error {
 args := m.Called(ctx, session)
 return args.Error(0)
}

func (m *MockWebSocketRepository) RemoveActiveSession(ctx context.Context, sessionID uuid.UUID) error {
 args := m.Called(ctx, sessionID)
 return args.Error(0)
}

func (m *MockWebSocketRepository) FindActiveSession(ctx context.Context, sessionID uuid.UUID) (*models.Session, error) {
 args := m.Called(ctx, sessionID)
 if args.Get(0) == nil {
  return nil, args.Error(1)
 }
 return args.Get(0).(*models.Session), args.Error(1)
}

func TestNewWebSocketService(t *testing.T) {
 mockRepo := new(MockWebSocketRepository)
 service := NewWebSocketService(mockRepo)

 assert.NotNil(t, service)
 assert.Equal(t, mockRepo, service.repo)
}

func TestConnectSession_Success(t *testing.T) {
 mockRepo := new(MockWebSocketRepository)
 service := NewWebSocketService(mockRepo)

 ctx := context.Background()
 userID := uuid.New()
 connID := uuid.New()

 // Mock expectations
 mockRepo.On("SaveActiveSession", ctx, mock.AnythingOfType("*models.Session")).Return(nil)

 session, err := service.ConnectSession(ctx, userID, connID)

 assert.NoError(t, err)
 assert.NotNil(t, session)
 assert.Equal(t, userID, session.UserID)
 assert.Equal(t, connID, session.ConnectionID)
 mockRepo.AssertExpectations(t)
}

func TestConnectSession_InvalidInput(t *testing.T) {
 mockRepo := new(MockWebSocketRepository)
 service := NewWebSocketService(mockRepo)

 ctx := context.Background()

 _, err := service.ConnectSession(ctx, uuid.Nil, uuid.New())
 assert.Error(t, err)

 _, err = service.ConnectSession(ctx, uuid.New(), uuid.Nil)
 assert.Error(t, err)
}

func TestConnectSession_RepositoryError(t *testing.T) {
 mockRepo := new(MockWebSocketRepository)
 service := NewWebSocketService(mockRepo)

 ctx := context.Background()
 userID := uuid.New()
 connID := uuid.New()

 dbErr := errors.New("database connection failed")
 mockRepo.On("SaveActiveSession", ctx, mock.AnythingOfType("*models.Session")).Return(dbErr)

 session, err := service.ConnectSession(ctx, userID, connID)

 assert.Error(t, err)
 assert.Nil(t, session)
 assert.Contains(t, err.Error(), "failed to establish session")
 mockRepo.AssertExpectations(t)
}

func TestDisconnectSession_Success(t *testing.T) {
 mockRepo := new(MockWebSocketRepository)
 service := NewWebSocketService(mockRepo)

 ctx := context.Background()
 sessionID := uuid.New()
 dummySession := &models.Session{ID: sessionID}

 // Mock expectations
 mockRepo.On("FindActiveSession", ctx, sessionID).Return(dummySession, nil)
 mockRepo.On("RemoveActiveSession", ctx, sessionID).Return(nil)

 err := service.DisconnectSession(ctx, sessionID)

 assert.NoError(t, err)
 mockRepo.AssertExpectations(t)
}

func TestDisconnectSession_SessionNotFound(t *testing.T) {
 mockRepo := new(MockWebSocketRepository)
 service := NewWebSocketService(mockRepo)

 ctx := context.Background()
 sessionID := uuid.New()

 // Mock expectations
 mockRepo.On("FindActiveSession", ctx, sessionID).Return(nil, errors.New("not found"))

 err := service.DisconnectSession(ctx, sessionID)

 assert.Error(t, err)
 assert.Contains(t, err.Error(), "session not found")
 mockRepo.AssertExpectations(t)
}