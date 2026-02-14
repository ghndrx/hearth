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

// MockMemberSidebarRepository is a mock implementation of MemberSidebarRepository using testify/mock.
type MockMemberSidebarRepository struct {
	mock.Mock
}

func (m *MockMemberSidebarRepository) GetMembersByServer(ctx context.Context, serverID uuid.UUID) ([]*models.Member, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Member), args.Error(1)
}

func (m *MockMemberSidebarRepository) GetMemberPresence(ctx context.Context, userID uuid.UUID) (*models.Presence, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Presence), args.Error(1)
}

func (m *MockMemberSidebarRepository) GetServerRoles(ctx context.Context, serverID uuid.UUID) ([]*models.Role, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Role), args.Error(1)
}

func (m *MockMemberSidebarRepository) GetMembersPresence(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*models.Presence, error) {
	args := m.Called(ctx, userIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[uuid.UUID]*models.Presence), args.Error(1)
}

func TestGetServerSidebar(t *testing.T) {
	ctx := context.Background()
	serverID := uuid.New()
	adminRoleID := uuid.New()
	userID := uuid.New()

	// Setup Data
	roles := []*models.Role{
		{ID: adminRoleID, Name: "Admin", Position: 1},
	}

	members := []*models.Member{
		{UserID: userID, ServerID: serverID, Roles: []uuid.UUID{adminRoleID}},
	}

	presences := map[uuid.UUID]*models.Presence{
		userID: {UserID: userID, Status: models.StatusOnline, UpdatedAt: time.Now()},
	}

	t.Run("Success - Groups members correctly", func(t *testing.T) {
		mockRepo := new(MockMemberSidebarRepository)
		service := NewMemberSidebarService(mockRepo)

		// Setup Expectations
		mockRepo.On("GetServerRoles", ctx, serverID).Return(roles, nil)
		mockRepo.On("GetMembersByServer", ctx, serverID).Return(members, nil)
		mockRepo.On("GetMembersPresence", ctx, []uuid.UUID{userID}).Return(presences, nil)

		// Execute
		result, err := service.GetServerSidebar(ctx, serverID)

		// Assertions
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result, 1) // Should have one group "Admin"

		adminGroup := result[0]
		assert.Equal(t, "Admin", adminGroup.RoleName)
		assert.Len(t, adminGroup.Members, 1)
		assert.Equal(t, models.StatusOnline, adminGroup.Members[0].Presence.Status)

		// Verify all mocks were called as expected
		mockRepo.AssertExpectations(t)
	})

	t.Run("Success - Handles missing presence gracefully", func(t *testing.T) {
		mockRepo := new(MockMemberSidebarRepository)
		service := NewMemberSidebarService(mockRepo)

		// Setup Expectations: Presence returns error
		mockRepo.On("GetServerRoles", ctx, serverID).Return(roles, nil)
		mockRepo.On("GetMembersByServer", ctx, serverID).Return(members, nil)
		mockRepo.On("GetMembersPresence", ctx, []uuid.UUID{userID}).Return(nil, errors.New("db error"))

		// Execute
		result, err := service.GetServerSidebar(ctx, serverID)

		// Assertions
		assert.NoError(t, err) // Should still succeed
		assert.Len(t, result, 1)

		// Should default to Offline
		assert.Equal(t, models.StatusOffline, result[0].Members[0].Presence.Status)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Invalid Server ID", func(t *testing.T) {
		mockRepo := new(MockMemberSidebarRepository)
		service := NewMemberSidebarService(mockRepo)

		// Execute with Nil UUID
		_, err := service.GetServerSidebar(ctx, uuid.Nil)

		// Assertions
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidInput, err)
	})
}
