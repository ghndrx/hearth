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
