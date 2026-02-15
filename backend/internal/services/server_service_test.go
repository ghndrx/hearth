package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"hearth/internal/models"
)

// MockRoleRepository is a mock implementation of RoleRepository
type MockRoleRepository struct {
	mock.Mock
}

func (m *MockRoleRepository) Create(ctx context.Context, role *models.Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *MockRoleRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Role, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Role), args.Error(1)
}

func (m *MockRoleRepository) GetByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.Role, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Role), args.Error(1)
}

func (m *MockRoleRepository) Update(ctx context.Context, role *models.Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *MockRoleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRoleRepository) UpdatePositions(ctx context.Context, serverID uuid.UUID, positions map[uuid.UUID]int) error {
	args := m.Called(ctx, serverID, positions)
	return args.Error(0)
}

func (m *MockRoleRepository) AddRoleToMember(ctx context.Context, serverID, userID, roleID uuid.UUID) error {
	args := m.Called(ctx, serverID, userID, roleID)
	return args.Error(0)
}

func (m *MockRoleRepository) RemoveRoleFromMember(ctx context.Context, serverID, userID, roleID uuid.UUID) error {
	args := m.Called(ctx, serverID, userID, roleID)
	return args.Error(0)
}

func (m *MockRoleRepository) GetMemberRoles(ctx context.Context, serverID, userID uuid.UUID) ([]*models.Role, error) {
	args := m.Called(ctx, serverID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Role), args.Error(1)
}

// Helper function to create a test ServerService with mocks
func newTestServerService() (*ServerService, *MockServerRepository, *MockChannelRepository, *MockRoleRepository, *MockCacheService, *MockEventBus) {
	serverRepo := new(MockServerRepository)
	channelRepo := new(MockChannelRepository)
	roleRepo := new(MockRoleRepository)
	cache := new(MockCacheService)
	eventBus := new(MockEventBus)

	quotaConfig := &models.QuotaConfig{
		Messages: models.MessageQuotaConfig{
			MaxMessageLength: 2000,
		},
		Servers: models.ServerQuotaConfig{
			MaxServersOwned:  10,
			MaxServersJoined: 100,
		},
		Storage: models.StorageQuotaConfig{
			UserStorageMB: 1024,
			MaxFileSizeMB: 100,
		},
	}

	// Mock repos for quota service (these won't be called in most tests)
	quotaUserRepo := new(MockUserRepository)
	quotaService := NewQuotaService(quotaConfig, serverRepo, quotaUserRepo, roleRepo)

	service := NewServerService(serverRepo, channelRepo, roleRepo, quotaService, cache, eventBus)
	return service, serverRepo, channelRepo, roleRepo, cache, eventBus
}

// ============================================
// CreateServer Tests
// ============================================

func TestCreateServer_Success(t *testing.T) {
	service, serverRepo, channelRepo, roleRepo, _, eventBus := newTestServerService()
	ctx := context.Background()
	ownerID := uuid.New()

	// Setup mocks
	serverRepo.On("GetOwnedServersCount", ctx, ownerID).Return(0, nil)
	serverRepo.On("Create", ctx, mock.AnythingOfType("*models.Server")).Return(nil)
	roleRepo.On("Create", ctx, mock.AnythingOfType("*models.Role")).Return(nil)
	channelRepo.On("Create", ctx, mock.AnythingOfType("*models.Channel")).Return(nil)
	serverRepo.On("AddMember", ctx, mock.AnythingOfType("*models.Member")).Return(nil)
	eventBus.On("Publish", "server.created", mock.Anything).Return()

	// Execute
	server, err := service.CreateServer(ctx, ownerID, "Test Server", "")

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, server)
	assert.Equal(t, "Test Server", server.Name)
	assert.Equal(t, ownerID, server.OwnerID)
	assert.Nil(t, server.IconURL)

	serverRepo.AssertExpectations(t)
	roleRepo.AssertExpectations(t)
	eventBus.AssertExpectations(t)
}

func TestCreateServer_WithIcon(t *testing.T) {
	service, serverRepo, channelRepo, roleRepo, _, eventBus := newTestServerService()
	ctx := context.Background()
	ownerID := uuid.New()
	iconURL := "https://example.com/icon.png"

	serverRepo.On("GetOwnedServersCount", ctx, ownerID).Return(0, nil)
	serverRepo.On("Create", ctx, mock.MatchedBy(func(s *models.Server) bool {
		return s.IconURL != nil && *s.IconURL == iconURL
	})).Return(nil)
	roleRepo.On("Create", ctx, mock.AnythingOfType("*models.Role")).Return(nil)
	channelRepo.On("Create", ctx, mock.AnythingOfType("*models.Channel")).Return(nil)
	serverRepo.On("AddMember", ctx, mock.AnythingOfType("*models.Member")).Return(nil)
	eventBus.On("Publish", "server.created", mock.Anything).Return()

	server, err := service.CreateServer(ctx, ownerID, "Test Server", iconURL)

	require.NoError(t, err)
	assert.NotNil(t, server.IconURL)
	assert.Equal(t, iconURL, *server.IconURL)
}

func TestCreateServer_MaxServersReached(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	ownerID := uuid.New()

	// User already owns max servers (10)
	serverRepo.On("GetOwnedServersCount", ctx, ownerID).Return(10, nil)

	server, err := service.CreateServer(ctx, ownerID, "Test Server", "")

	assert.Nil(t, server)
	assert.ErrorIs(t, err, ErrMaxServersReached)
}

func TestCreateServer_CreateFails(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	ownerID := uuid.New()
	dbErr := errors.New("database error")

	serverRepo.On("GetOwnedServersCount", ctx, ownerID).Return(0, nil)
	serverRepo.On("Create", ctx, mock.AnythingOfType("*models.Server")).Return(dbErr)

	server, err := service.CreateServer(ctx, ownerID, "Test Server", "")

	assert.Nil(t, server)
	assert.Error(t, err)
	assert.Equal(t, dbErr, err)
}

func TestCreateServer_RoleCreateFails_RollsBack(t *testing.T) {
	service, serverRepo, _, roleRepo, _, _ := newTestServerService()
	ctx := context.Background()
	ownerID := uuid.New()
	roleErr := errors.New("role creation failed")

	serverRepo.On("GetOwnedServersCount", ctx, ownerID).Return(0, nil)
	serverRepo.On("Create", ctx, mock.AnythingOfType("*models.Server")).Return(nil)
	roleRepo.On("Create", ctx, mock.AnythingOfType("*models.Role")).Return(roleErr)
	serverRepo.On("Delete", ctx, mock.AnythingOfType("uuid.UUID")).Return(nil)

	server, err := service.CreateServer(ctx, ownerID, "Test Server", "")

	assert.Nil(t, server)
	assert.Error(t, err)
	// Verify rollback was called
	serverRepo.AssertCalled(t, "Delete", ctx, mock.AnythingOfType("uuid.UUID"))
}

// ============================================
// GetServer Tests
// ============================================

func TestGetServer_Success(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	serverID := uuid.New()
	ownerID := uuid.New()

	expectedServer := &models.Server{
		ID:        serverID,
		Name:      "Test Server",
		OwnerID:   ownerID,
		CreatedAt: time.Now(),
	}

	serverRepo.On("GetByID", ctx, serverID).Return(expectedServer, nil)

	server, err := service.GetServer(ctx, serverID)

	require.NoError(t, err)
	assert.Equal(t, expectedServer.ID, server.ID)
	assert.Equal(t, expectedServer.Name, server.Name)
}

func TestGetServer_NotFound(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	serverID := uuid.New()

	serverRepo.On("GetByID", ctx, serverID).Return(nil, nil)

	server, err := service.GetServer(ctx, serverID)

	assert.Nil(t, server)
	assert.ErrorIs(t, err, ErrServerNotFound)
}

// ============================================
// UpdateServer Tests
// ============================================

func TestUpdateServer_Success_ByOwner(t *testing.T) {
	service, serverRepo, _, _, _, eventBus := newTestServerService()
	ctx := context.Background()
	serverID := uuid.New()
	ownerID := uuid.New()

	existingServer := &models.Server{
		ID:        serverID,
		Name:      "Old Name",
		OwnerID:   ownerID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	newName := "New Name"
	updates := &models.ServerUpdate{
		Name: &newName,
	}

	serverRepo.On("GetByID", ctx, serverID).Return(existingServer, nil)
	serverRepo.On("Update", ctx, mock.MatchedBy(func(s *models.Server) bool {
		return s.Name == newName
	})).Return(nil)
	eventBus.On("Publish", "server.updated", mock.Anything).Return()

	server, err := service.UpdateServer(ctx, serverID, ownerID, updates)

	require.NoError(t, err)
	assert.Equal(t, newName, server.Name)
}

func TestUpdateServer_NotFound(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	serverID := uuid.New()
	requesterID := uuid.New()

	serverRepo.On("GetByID", ctx, serverID).Return(nil, nil)

	server, err := service.UpdateServer(ctx, serverID, requesterID, &models.ServerUpdate{})

	assert.Nil(t, server)
	assert.ErrorIs(t, err, ErrServerNotFound)
}

func TestUpdateServer_NotMember(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	serverID := uuid.New()
	ownerID := uuid.New()
	requesterID := uuid.New() // Different from owner

	existingServer := &models.Server{
		ID:      serverID,
		Name:    "Test Server",
		OwnerID: ownerID,
	}

	serverRepo.On("GetByID", ctx, serverID).Return(existingServer, nil)
	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(nil, nil)

	server, err := service.UpdateServer(ctx, serverID, requesterID, &models.ServerUpdate{})

	assert.Nil(t, server)
	assert.ErrorIs(t, err, ErrNotServerMember)
}

// ============================================
// DeleteServer Tests
// ============================================

func TestDeleteServer_Success(t *testing.T) {
	service, serverRepo, _, _, _, eventBus := newTestServerService()
	ctx := context.Background()
	serverID := uuid.New()
	ownerID := uuid.New()

	existingServer := &models.Server{
		ID:      serverID,
		Name:    "Test Server",
		OwnerID: ownerID,
	}

	serverRepo.On("GetByID", ctx, serverID).Return(existingServer, nil)
	serverRepo.On("Delete", ctx, serverID).Return(nil)
	eventBus.On("Publish", "server.deleted", mock.Anything).Return()

	err := service.DeleteServer(ctx, serverID, ownerID)

	require.NoError(t, err)
	serverRepo.AssertCalled(t, "Delete", ctx, serverID)
}

func TestDeleteServer_NotOwner(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	serverID := uuid.New()
	ownerID := uuid.New()
	requesterID := uuid.New() // Not the owner

	existingServer := &models.Server{
		ID:      serverID,
		Name:    "Test Server",
		OwnerID: ownerID,
	}

	serverRepo.On("GetByID", ctx, serverID).Return(existingServer, nil)

	err := service.DeleteServer(ctx, serverID, requesterID)

	assert.ErrorIs(t, err, ErrNotServerOwner)
}

func TestDeleteServer_NotFound(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	serverID := uuid.New()
	requesterID := uuid.New()

	serverRepo.On("GetByID", ctx, serverID).Return(nil, nil)

	err := service.DeleteServer(ctx, serverID, requesterID)

	assert.ErrorIs(t, err, ErrServerNotFound)
}

// ============================================
// TransferOwnership Tests
// ============================================

func TestTransferOwnership_Success(t *testing.T) {
	service, serverRepo, _, _, _, eventBus := newTestServerService()
	ctx := context.Background()
	serverID := uuid.New()
	ownerID := uuid.New()
	newOwnerID := uuid.New()

	existingServer := &models.Server{
		ID:      serverID,
		Name:    "Test Server",
		OwnerID: ownerID,
	}

	newOwnerMember := &models.Member{
		UserID:   newOwnerID,
		ServerID: serverID,
	}

	serverRepo.On("GetByID", ctx, serverID).Return(existingServer, nil)
	serverRepo.On("GetMember", ctx, serverID, newOwnerID).Return(newOwnerMember, nil)
	serverRepo.On("TransferOwnership", ctx, serverID, newOwnerID).Return(nil)
	eventBus.On("Publish", "server.ownership_transferred", mock.Anything).Return()

	server, err := service.TransferOwnership(ctx, serverID, ownerID, newOwnerID)

	require.NoError(t, err)
	assert.Equal(t, newOwnerID, server.OwnerID)
	serverRepo.AssertExpectations(t)
	eventBus.AssertExpectations(t)
}

func TestTransferOwnership_NotOwner(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	serverID := uuid.New()
	ownerID := uuid.New()
	requesterID := uuid.New() // Not the owner
	newOwnerID := uuid.New()

	existingServer := &models.Server{
		ID:      serverID,
		Name:    "Test Server",
		OwnerID: ownerID,
	}

	serverRepo.On("GetByID", ctx, serverID).Return(existingServer, nil)

	server, err := service.TransferOwnership(ctx, serverID, requesterID, newOwnerID)

	assert.Nil(t, server)
	assert.ErrorIs(t, err, ErrNotServerOwner)
}

func TestTransferOwnership_ServerNotFound(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	serverID := uuid.New()
	requesterID := uuid.New()
	newOwnerID := uuid.New()

	serverRepo.On("GetByID", ctx, serverID).Return(nil, nil)

	server, err := service.TransferOwnership(ctx, serverID, requesterID, newOwnerID)

	assert.Nil(t, server)
	assert.ErrorIs(t, err, ErrServerNotFound)
}

func TestTransferOwnership_SelfAction(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	serverID := uuid.New()
	ownerID := uuid.New()

	existingServer := &models.Server{
		ID:      serverID,
		Name:    "Test Server",
		OwnerID: ownerID,
	}

	serverRepo.On("GetByID", ctx, serverID).Return(existingServer, nil)

	// Try to transfer to self
	server, err := service.TransferOwnership(ctx, serverID, ownerID, ownerID)

	assert.Nil(t, server)
	assert.ErrorIs(t, err, ErrSelfAction)
}

func TestTransferOwnership_NewOwnerNotMember(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	serverID := uuid.New()
	ownerID := uuid.New()
	newOwnerID := uuid.New()

	existingServer := &models.Server{
		ID:      serverID,
		Name:    "Test Server",
		OwnerID: ownerID,
	}

	serverRepo.On("GetByID", ctx, serverID).Return(existingServer, nil)
	serverRepo.On("GetMember", ctx, serverID, newOwnerID).Return(nil, nil)

	server, err := service.TransferOwnership(ctx, serverID, ownerID, newOwnerID)

	assert.Nil(t, server)
	assert.ErrorIs(t, err, ErrNotServerMember)
}

func TestTransferOwnership_DatabaseError(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	serverID := uuid.New()
	ownerID := uuid.New()
	newOwnerID := uuid.New()
	dbErr := errors.New("database error")

	existingServer := &models.Server{
		ID:      serverID,
		Name:    "Test Server",
		OwnerID: ownerID,
	}

	newOwnerMember := &models.Member{
		UserID:   newOwnerID,
		ServerID: serverID,
	}

	serverRepo.On("GetByID", ctx, serverID).Return(existingServer, nil)
	serverRepo.On("GetMember", ctx, serverID, newOwnerID).Return(newOwnerMember, nil)
	serverRepo.On("TransferOwnership", ctx, serverID, newOwnerID).Return(dbErr)

	server, err := service.TransferOwnership(ctx, serverID, ownerID, newOwnerID)

	assert.Nil(t, server)
	assert.Error(t, err)
	assert.Equal(t, dbErr, err)
}

// ============================================
// JoinServer Tests
// ============================================

func TestJoinServer_Success(t *testing.T) {
	service, serverRepo, _, roleRepo, _, eventBus := newTestServerService()
	ctx := context.Background()
	userID := uuid.New()
	serverID := uuid.New()
	inviteCode := "abc123"
	everyoneRoleID := uuid.New()

	invite := &models.Invite{
		Code:      inviteCode,
		ServerID:  serverID,
		MaxUses:   10,
		Uses:      0,
		ExpiresAt: nil,
	}

	server := &models.Server{
		ID:   serverID,
		Name: "Test Server",
	}

	everyoneRole := &models.Role{
		ID:        everyoneRoleID,
		ServerID:  serverID,
		Name:      "@everyone",
		IsDefault: true,
	}

	serverRepo.On("GetInvite", ctx, inviteCode).Return(invite, nil)
	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)
	serverRepo.On("GetBan", ctx, serverID, userID).Return(nil, nil)
	serverRepo.On("GetMember", ctx, serverID, userID).Return(nil, nil)
	serverRepo.On("GetUserServers", ctx, userID).Return([]*models.Server{}, nil)
	roleRepo.On("GetByServerID", ctx, serverID).Return([]*models.Role{everyoneRole}, nil)
	serverRepo.On("AddMember", ctx, mock.AnythingOfType("*models.Member")).Return(nil)
	serverRepo.On("IncrementInviteUses", ctx, inviteCode).Return(nil)
	eventBus.On("Publish", "server.member_joined", mock.Anything).Return()

	result, err := service.JoinServer(ctx, userID, inviteCode)

	require.NoError(t, err)
	assert.Equal(t, server.ID, result.ID)
}

func TestJoinServer_InviteNotFound(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	userID := uuid.New()
	inviteCode := "invalid"

	serverRepo.On("GetInvite", ctx, inviteCode).Return(nil, nil)

	result, err := service.JoinServer(ctx, userID, inviteCode)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrInviteNotFound)
}

func TestJoinServer_InviteExpired(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	userID := uuid.New()
	inviteCode := "expired"
	expiredTime := time.Now().Add(-1 * time.Hour)

	invite := &models.Invite{
		Code:      inviteCode,
		ServerID:  uuid.New(),
		ExpiresAt: &expiredTime,
	}

	serverRepo.On("GetInvite", ctx, inviteCode).Return(invite, nil)

	result, err := service.JoinServer(ctx, userID, inviteCode)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrInviteExpired)
}

func TestJoinServer_MaxUsesReached(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	userID := uuid.New()
	inviteCode := "maxed"

	invite := &models.Invite{
		Code:     inviteCode,
		ServerID: uuid.New(),
		MaxUses:  5,
		Uses:     5,
	}

	serverRepo.On("GetInvite", ctx, inviteCode).Return(invite, nil)

	result, err := service.JoinServer(ctx, userID, inviteCode)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrInviteExpired)
}

func TestJoinServer_UserBanned(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	userID := uuid.New()
	serverID := uuid.New()
	inviteCode := "abc123"

	invite := &models.Invite{
		Code:     inviteCode,
		ServerID: serverID,
	}

	server := &models.Server{
		ID: serverID,
	}

	ban := &models.Ban{
		ServerID: serverID,
		UserID:   userID,
	}

	serverRepo.On("GetInvite", ctx, inviteCode).Return(invite, nil)
	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)
	serverRepo.On("GetBan", ctx, serverID, userID).Return(ban, nil)

	result, err := service.JoinServer(ctx, userID, inviteCode)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrBannedFromServer)
}

func TestJoinServer_AlreadyMember(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	userID := uuid.New()
	serverID := uuid.New()
	inviteCode := "abc123"

	invite := &models.Invite{
		Code:     inviteCode,
		ServerID: serverID,
	}

	server := &models.Server{
		ID: serverID,
	}

	existingMember := &models.Member{
		UserID:   userID,
		ServerID: serverID,
	}

	serverRepo.On("GetInvite", ctx, inviteCode).Return(invite, nil)
	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)
	serverRepo.On("GetBan", ctx, serverID, userID).Return(nil, nil)
	serverRepo.On("GetMember", ctx, serverID, userID).Return(existingMember, nil)

	result, err := service.JoinServer(ctx, userID, inviteCode)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrAlreadyMember)
}

// ============================================
// LeaveServer Tests
// ============================================

func TestLeaveServer_Success(t *testing.T) {
	service, serverRepo, _, _, _, eventBus := newTestServerService()
	ctx := context.Background()
	serverID := uuid.New()
	ownerID := uuid.New()
	userID := uuid.New() // Not the owner

	server := &models.Server{
		ID:      serverID,
		OwnerID: ownerID,
	}

	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)
	serverRepo.On("RemoveMember", ctx, serverID, userID).Return(nil)
	eventBus.On("Publish", "server.member_left", mock.Anything).Return()

	err := service.LeaveServer(ctx, serverID, userID)

	require.NoError(t, err)
}

func TestLeaveServer_OwnerCannotLeave(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	serverID := uuid.New()
	ownerID := uuid.New()

	server := &models.Server{
		ID:      serverID,
		OwnerID: ownerID,
	}

	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)

	err := service.LeaveServer(ctx, serverID, ownerID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "owner cannot leave")
}

func TestLeaveServer_NotFound(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()

	serverRepo.On("GetByID", ctx, serverID).Return(nil, nil)

	err := service.LeaveServer(ctx, serverID, userID)

	assert.ErrorIs(t, err, ErrServerNotFound)
}

// ============================================
// KickMember Tests
// ============================================

func TestKickMember_Success(t *testing.T) {
	service, serverRepo, _, _, _, eventBus := newTestServerService()
	ctx := context.Background()
	serverID := uuid.New()
	ownerID := uuid.New()
	targetID := uuid.New()

	server := &models.Server{
		ID:      serverID,
		OwnerID: ownerID,
	}

	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)
	serverRepo.On("RemoveMember", ctx, serverID, targetID).Return(nil)
	eventBus.On("Publish", "server.member_kicked", mock.Anything).Return()

	err := service.KickMember(ctx, serverID, ownerID, targetID, "violated rules")

	require.NoError(t, err)
}

func TestKickMember_CannotKickOwner(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	serverID := uuid.New()
	ownerID := uuid.New()
	adminID := uuid.New()

	server := &models.Server{
		ID:      serverID,
		OwnerID: ownerID,
	}

	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)

	err := service.KickMember(ctx, serverID, adminID, ownerID, "trying to kick owner")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot kick server owner")
}

// ============================================
// BanMember Tests
// ============================================

func TestBanMember_Success(t *testing.T) {
	service, serverRepo, _, _, _, eventBus := newTestServerService()
	ctx := context.Background()
	serverID := uuid.New()
	ownerID := uuid.New()
	targetID := uuid.New()

	server := &models.Server{
		ID:      serverID,
		OwnerID: ownerID,
	}

	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)
	serverRepo.On("RemoveMember", ctx, serverID, targetID).Return(nil)
	serverRepo.On("AddBan", ctx, mock.MatchedBy(func(b *models.Ban) bool {
		return b.ServerID == serverID && b.UserID == targetID
	})).Return(nil)
	eventBus.On("Publish", "server.member_banned", mock.Anything).Return()

	err := service.BanMember(ctx, serverID, ownerID, targetID, "spamming", 0)

	require.NoError(t, err)
}

func TestBanMember_CannotBanOwner(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	serverID := uuid.New()
	ownerID := uuid.New()
	adminID := uuid.New()

	server := &models.Server{
		ID:      serverID,
		OwnerID: ownerID,
	}

	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)

	err := service.BanMember(ctx, serverID, adminID, ownerID, "trying to ban owner", 0)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot ban server owner")
}

// ============================================
// CreateInvite Tests
// ============================================

func TestCreateInvite_Success(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	serverID := uuid.New()
	channelID := uuid.New()
	creatorID := uuid.New()

	member := &models.Member{
		UserID:   creatorID,
		ServerID: serverID,
	}

	serverRepo.On("GetMember", ctx, serverID, creatorID).Return(member, nil)
	serverRepo.On("CreateInvite", ctx, mock.AnythingOfType("*models.Invite")).Return(nil)

	invite, err := service.CreateInvite(ctx, serverID, channelID, creatorID, 10, nil)

	require.NoError(t, err)
	assert.NotEmpty(t, invite.Code)
	assert.Equal(t, serverID, invite.ServerID)
	assert.Equal(t, 10, invite.MaxUses)
}

func TestCreateInvite_NotMember(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	serverID := uuid.New()
	channelID := uuid.New()
	creatorID := uuid.New()

	serverRepo.On("GetMember", ctx, serverID, creatorID).Return(nil, nil)

	invite, err := service.CreateInvite(ctx, serverID, channelID, creatorID, 10, nil)

	assert.Nil(t, invite)
	assert.ErrorIs(t, err, ErrNotServerMember)
}

func TestCreateInvite_WithExpiration(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	serverID := uuid.New()
	channelID := uuid.New()
	creatorID := uuid.New()
	expiresDuration := 24 * time.Hour

	member := &models.Member{
		UserID:   creatorID,
		ServerID: serverID,
	}

	serverRepo.On("GetMember", ctx, serverID, creatorID).Return(member, nil)
	serverRepo.On("CreateInvite", ctx, mock.MatchedBy(func(i *models.Invite) bool {
		return i.ExpiresAt != nil
	})).Return(nil)

	invite, err := service.CreateInvite(ctx, serverID, channelID, creatorID, 0, &expiresDuration)

	require.NoError(t, err)
	assert.NotNil(t, invite.ExpiresAt)
}

// ============================================
// GetUserServers Tests
// ============================================

func TestGetUserServers_Success(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	userID := uuid.New()

	expectedServers := []*models.Server{
		{ID: uuid.New(), Name: "Server 1", OwnerID: userID},
		{ID: uuid.New(), Name: "Server 2", OwnerID: uuid.New()},
	}

	serverRepo.On("GetUserServers", ctx, userID).Return(expectedServers, nil)

	servers, err := service.GetUserServers(ctx, userID)

	require.NoError(t, err)
	assert.Len(t, servers, 2)
	assert.Equal(t, "Server 1", servers[0].Name)
}

func TestGetUserServers_Empty(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	userID := uuid.New()

	serverRepo.On("GetUserServers", ctx, userID).Return([]*models.Server{}, nil)

	servers, err := service.GetUserServers(ctx, userID)

	require.NoError(t, err)
	assert.Len(t, servers, 0)
}

func TestGetUserServers_Error(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	userID := uuid.New()

	serverRepo.On("GetUserServers", ctx, userID).Return(nil, errors.New("db error"))

	servers, err := service.GetUserServers(ctx, userID)

	assert.Nil(t, servers)
	assert.Error(t, err)
}

// ============================================
// GetMembers Tests
// ============================================

func TestGetMembers_Success(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	serverID := uuid.New()

	expectedMembers := []*models.Member{
		{UserID: uuid.New(), ServerID: serverID},
		{UserID: uuid.New(), ServerID: serverID},
	}

	serverRepo.On("GetMembers", ctx, serverID, 100, 0).Return(expectedMembers, nil)

	members, err := service.GetMembers(ctx, serverID, 100, 0)

	require.NoError(t, err)
	assert.Len(t, members, 2)
}

func TestGetMembers_Empty(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	serverID := uuid.New()

	serverRepo.On("GetMembers", ctx, serverID, 100, 0).Return([]*models.Member{}, nil)

	members, err := service.GetMembers(ctx, serverID, 100, 0)

	require.NoError(t, err)
	assert.Len(t, members, 0)
}

func TestGetMembers_WithPagination(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	serverID := uuid.New()

	expectedMembers := []*models.Member{
		{UserID: uuid.New(), ServerID: serverID},
	}

	serverRepo.On("GetMembers", ctx, serverID, 10, 5).Return(expectedMembers, nil)

	members, err := service.GetMembers(ctx, serverID, 10, 5)

	require.NoError(t, err)
	assert.Len(t, members, 1)
}

// ============================================
// GetMember Tests
// ============================================

func TestGetMember_Success(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()

	expectedMember := &models.Member{
		UserID:   userID,
		ServerID: serverID,
		JoinedAt: time.Now(),
	}

	serverRepo.On("GetMember", ctx, serverID, userID).Return(expectedMember, nil)

	member, err := service.GetMember(ctx, serverID, userID)

	require.NoError(t, err)
	assert.Equal(t, userID, member.UserID)
	assert.Equal(t, serverID, member.ServerID)
}

func TestGetMember_NotFound(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()

	serverRepo.On("GetMember", ctx, serverID, userID).Return(nil, nil)

	member, err := service.GetMember(ctx, serverID, userID)

	assert.Nil(t, member)
	assert.ErrorIs(t, err, ErrNotServerMember)
}

func TestGetMember_Error(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()

	serverRepo.On("GetMember", ctx, serverID, userID).Return(nil, errors.New("db error"))

	member, err := service.GetMember(ctx, serverID, userID)

	assert.Nil(t, member)
	assert.Error(t, err)
}

// ============================================
// UpdateMember Tests
// ============================================

func TestUpdateMember_Success_Nickname(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	serverID := uuid.New()
	requesterID := uuid.New()
	targetID := uuid.New()
	nickname := "NewNickname"

	existingMember := &models.Member{
		UserID:   targetID,
		ServerID: serverID,
		JoinedAt: time.Now(),
	}

	serverRepo.On("GetMember", ctx, serverID, targetID).Return(existingMember, nil)
	serverRepo.On("UpdateMember", ctx, mock.MatchedBy(func(m *models.Member) bool {
		return m.Nickname != nil && *m.Nickname == nickname
	})).Return(nil)

	member, err := service.UpdateMember(ctx, serverID, requesterID, targetID, &nickname, nil)

	require.NoError(t, err)
	assert.Equal(t, &nickname, member.Nickname)
}

func TestUpdateMember_Success_Roles(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	serverID := uuid.New()
	requesterID := uuid.New()
	targetID := uuid.New()
	roles := []uuid.UUID{uuid.New(), uuid.New()}

	existingMember := &models.Member{
		UserID:   targetID,
		ServerID: serverID,
		JoinedAt: time.Now(),
	}

	serverRepo.On("GetMember", ctx, serverID, targetID).Return(existingMember, nil)
	serverRepo.On("UpdateMember", ctx, mock.MatchedBy(func(m *models.Member) bool {
		return len(m.Roles) == 2
	})).Return(nil)

	member, err := service.UpdateMember(ctx, serverID, requesterID, targetID, nil, roles)

	require.NoError(t, err)
	assert.Len(t, member.Roles, 2)
}

func TestUpdateMember_NotFound(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	serverID := uuid.New()
	requesterID := uuid.New()
	targetID := uuid.New()
	nickname := "NewNickname"

	serverRepo.On("GetMember", ctx, serverID, targetID).Return(nil, nil)

	member, err := service.UpdateMember(ctx, serverID, requesterID, targetID, &nickname, nil)

	assert.Nil(t, member)
	assert.ErrorIs(t, err, ErrNotServerMember)
}

func TestUpdateMember_UpdateFails(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	serverID := uuid.New()
	requesterID := uuid.New()
	targetID := uuid.New()
	nickname := "NewNickname"

	existingMember := &models.Member{
		UserID:   targetID,
		ServerID: serverID,
	}

	serverRepo.On("GetMember", ctx, serverID, targetID).Return(existingMember, nil)
	serverRepo.On("UpdateMember", ctx, mock.Anything).Return(errors.New("db error"))

	member, err := service.UpdateMember(ctx, serverID, requesterID, targetID, &nickname, nil)

	assert.Nil(t, member)
	assert.Error(t, err)
}

// ============================================
// GetBans Tests
// ============================================

func TestGetBans_Success(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	serverID := uuid.New()
	reason1 := "spam"
	reason2 := "harassment"

	expectedBans := []*models.Ban{
		{ServerID: serverID, UserID: uuid.New(), Reason: &reason1},
		{ServerID: serverID, UserID: uuid.New(), Reason: &reason2},
	}

	serverRepo.On("GetBans", ctx, serverID).Return(expectedBans, nil)

	bans, err := service.GetBans(ctx, serverID)

	require.NoError(t, err)
	assert.Len(t, bans, 2)
}

func TestGetBans_Empty(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	serverID := uuid.New()

	serverRepo.On("GetBans", ctx, serverID).Return([]*models.Ban{}, nil)

	bans, err := service.GetBans(ctx, serverID)

	require.NoError(t, err)
	assert.Len(t, bans, 0)
}

// ============================================
// UnbanMember Tests
// ============================================

func TestUnbanMember_Success(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	serverID := uuid.New()
	requesterID := uuid.New()
	targetID := uuid.New()

	serverRepo.On("RemoveBan", ctx, serverID, targetID).Return(nil)

	err := service.UnbanMember(ctx, serverID, requesterID, targetID)

	require.NoError(t, err)
}

func TestUnbanMember_Error(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	serverID := uuid.New()
	requesterID := uuid.New()
	targetID := uuid.New()

	serverRepo.On("RemoveBan", ctx, serverID, targetID).Return(errors.New("db error"))

	err := service.UnbanMember(ctx, serverID, requesterID, targetID)

	assert.Error(t, err)
}

// ============================================
// GetInvites Tests
// ============================================

func TestGetInvites_Success(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	serverID := uuid.New()

	expectedInvites := []*models.Invite{
		{Code: "abc123", ServerID: serverID},
		{Code: "xyz789", ServerID: serverID},
	}

	serverRepo.On("GetInvites", ctx, serverID).Return(expectedInvites, nil)

	invites, err := service.GetInvites(ctx, serverID)

	require.NoError(t, err)
	assert.Len(t, invites, 2)
}

func TestGetInvites_Empty(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	serverID := uuid.New()

	serverRepo.On("GetInvites", ctx, serverID).Return([]*models.Invite{}, nil)

	invites, err := service.GetInvites(ctx, serverID)

	require.NoError(t, err)
	assert.Len(t, invites, 0)
}

// ============================================
// GetInvite Tests (ServerService)
// ============================================

func TestServerGetInvite_Success(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	code := "test-invite-code"

	expectedInvite := &models.Invite{
		Code:     code,
		ServerID: uuid.New(),
		MaxUses:  10,
	}

	serverRepo.On("GetInvite", ctx, code).Return(expectedInvite, nil)

	invite, err := service.GetInvite(ctx, code)

	require.NoError(t, err)
	assert.Equal(t, code, invite.Code)
}

func TestServerGetInvite_NotFound(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	code := "nonexistent"

	serverRepo.On("GetInvite", ctx, code).Return(nil, nil)

	invite, err := service.GetInvite(ctx, code)

	assert.Nil(t, invite)
	assert.ErrorIs(t, err, ErrInviteNotFound)
}

func TestServerGetInvite_Error(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	code := "test-code"

	serverRepo.On("GetInvite", ctx, code).Return(nil, errors.New("db error"))

	invite, err := service.GetInvite(ctx, code)

	assert.Nil(t, invite)
	assert.Error(t, err)
}

// ============================================
// DeleteInvite Tests (ServerService)
// ============================================

func TestServerDeleteInvite_Success(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	code := "test-invite-code"
	requesterID := uuid.New()

	existingInvite := &models.Invite{
		Code:      code,
		ServerID:  uuid.New(),
		CreatorID: requesterID,
	}

	serverRepo.On("GetInvite", ctx, code).Return(existingInvite, nil)
	serverRepo.On("DeleteInvite", ctx, code).Return(nil)

	err := service.DeleteInvite(ctx, code, requesterID)

	require.NoError(t, err)
}

func TestServerDeleteInvite_NotFound(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	code := "nonexistent"
	requesterID := uuid.New()

	serverRepo.On("GetInvite", ctx, code).Return(nil, nil)

	err := service.DeleteInvite(ctx, code, requesterID)

	assert.ErrorIs(t, err, ErrInviteNotFound)
}

func TestServerDeleteInvite_Error(t *testing.T) {
	service, serverRepo, _, _, _, _ := newTestServerService()
	ctx := context.Background()
	code := "test-code"
	requesterID := uuid.New()

	existingInvite := &models.Invite{
		Code:      code,
		ServerID:  uuid.New(),
		CreatorID: requesterID,
	}

	serverRepo.On("GetInvite", ctx, code).Return(existingInvite, nil)
	serverRepo.On("DeleteInvite", ctx, code).Return(errors.New("db error"))

	err := service.DeleteInvite(ctx, code, requesterID)

	assert.Error(t, err)
}

// ============================================
// GetChannels Tests
// ============================================

func TestGetChannels_Success(t *testing.T) {
	service, _, channelRepo, _, _, _ := newTestServerService()
	ctx := context.Background()
	serverID := uuid.New()

	expectedChannels := []*models.Channel{
		{ID: uuid.New(), Name: "general", ServerID: &serverID},
		{ID: uuid.New(), Name: "random", ServerID: &serverID},
	}

	channelRepo.On("GetByServerID", ctx, serverID).Return(expectedChannels, nil)

	channels, err := service.GetChannels(ctx, serverID)

	require.NoError(t, err)
	assert.Len(t, channels, 2)
}

func TestGetChannels_Empty(t *testing.T) {
	service, _, channelRepo, _, _, _ := newTestServerService()
	ctx := context.Background()
	serverID := uuid.New()

	channelRepo.On("GetByServerID", ctx, serverID).Return([]*models.Channel{}, nil)

	channels, err := service.GetChannels(ctx, serverID)

	require.NoError(t, err)
	assert.Len(t, channels, 0)
}

// ============================================
// GetRoles Tests
// ============================================

func TestGetRoles_Success(t *testing.T) {
	service, _, _, roleRepo, _, _ := newTestServerService()
	ctx := context.Background()
	serverID := uuid.New()

	expectedRoles := []*models.Role{
		{ID: uuid.New(), Name: "Admin", ServerID: serverID},
		{ID: uuid.New(), Name: "Member", ServerID: serverID},
	}

	roleRepo.On("GetByServerID", ctx, serverID).Return(expectedRoles, nil)

	roles, err := service.GetRoles(ctx, serverID)

	require.NoError(t, err)
	assert.Len(t, roles, 2)
}

func TestGetRoles_Empty(t *testing.T) {
	service, _, _, roleRepo, _, _ := newTestServerService()
	ctx := context.Background()
	serverID := uuid.New()

	roleRepo.On("GetByServerID", ctx, serverID).Return([]*models.Role{}, nil)

	roles, err := service.GetRoles(ctx, serverID)

	require.NoError(t, err)
	assert.Len(t, roles, 0)
}
