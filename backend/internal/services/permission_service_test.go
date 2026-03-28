package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"hearth/internal/models"
)

// MockServerRepoForPermissions is a mock for ServerRepository
type MockServerRepoForPermissions struct {
	mock.Mock
}

func (m *MockServerRepoForPermissions) Create(ctx context.Context, server *models.Server) error {
	args := m.Called(ctx, server)
	return args.Error(0)
}

func (m *MockServerRepoForPermissions) GetByID(ctx context.Context, id uuid.UUID) (*models.Server, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Server), args.Error(1)
}

func (m *MockServerRepoForPermissions) Update(ctx context.Context, server *models.Server) error {
	args := m.Called(ctx, server)
	return args.Error(0)
}

func (m *MockServerRepoForPermissions) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockServerRepoForPermissions) TransferOwnership(ctx context.Context, serverID, newOwnerID uuid.UUID) error {
	args := m.Called(ctx, serverID, newOwnerID)
	return args.Error(0)
}

func (m *MockServerRepoForPermissions) GetMembers(ctx context.Context, serverID uuid.UUID, limit, offset int) ([]*models.Member, error) {
	args := m.Called(ctx, serverID, limit, offset)
	return args.Get(0).([]*models.Member), args.Error(1)
}

func (m *MockServerRepoForPermissions) GetAllMembers(ctx context.Context, serverID uuid.UUID) ([]*models.Member, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Member), args.Error(1)
}

func (m *MockServerRepoForPermissions) GetMembersPaginated(ctx context.Context, serverID uuid.UUID, cursor *models.MemberCursor, limit int) (*models.PaginatedMembers, error) {
	args := m.Called(ctx, serverID, cursor, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.PaginatedMembers), args.Error(1)
}

func (m *MockServerRepoForPermissions) GetMember(ctx context.Context, serverID, userID uuid.UUID) (*models.Member, error) {
	args := m.Called(ctx, serverID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Member), args.Error(1)
}

func (m *MockServerRepoForPermissions) GetMembersWithRole(ctx context.Context, serverID, roleID uuid.UUID) ([]*models.Member, error) {
	args := m.Called(ctx, serverID, roleID)
	return args.Get(0).([]*models.Member), args.Error(1)
}

func (m *MockServerRepoForPermissions) GetMembersWithRolePaginated(ctx context.Context, serverID, roleID uuid.UUID, cursor *models.MemberCursor, limit int) (*models.PaginatedMembers, error) {
	args := m.Called(ctx, serverID, roleID, cursor, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.PaginatedMembers), args.Error(1)
}

func (m *MockServerRepoForPermissions) GetAllMembersWithRole(ctx context.Context, serverID, roleID uuid.UUID) ([]*models.Member, error) {
	args := m.Called(ctx, serverID, roleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Member), args.Error(1)
}

func (m *MockServerRepoForPermissions) AddMember(ctx context.Context, member *models.Member) error {
	args := m.Called(ctx, member)
	return args.Error(0)
}

func (m *MockServerRepoForPermissions) UpdateMember(ctx context.Context, member *models.Member) error {
	args := m.Called(ctx, member)
	return args.Error(0)
}

func (m *MockServerRepoForPermissions) RemoveMember(ctx context.Context, serverID, userID uuid.UUID) error {
	args := m.Called(ctx, serverID, userID)
	return args.Error(0)
}

func (m *MockServerRepoForPermissions) GetMemberCount(ctx context.Context, serverID uuid.UUID) (int, error) {
	args := m.Called(ctx, serverID)
	return args.Int(0), args.Error(1)
}

func (m *MockServerRepoForPermissions) GetUserServers(ctx context.Context, userID uuid.UUID) ([]*models.Server, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*models.Server), args.Error(1)
}

func (m *MockServerRepoForPermissions) GetOwnedServersCount(ctx context.Context, userID uuid.UUID) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

func (m *MockServerRepoForPermissions) GetBan(ctx context.Context, serverID, userID uuid.UUID) (*models.Ban, error) {
	args := m.Called(ctx, serverID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Ban), args.Error(1)
}

func (m *MockServerRepoForPermissions) AddBan(ctx context.Context, ban *models.Ban) error {
	args := m.Called(ctx, ban)
	return args.Error(0)
}

func (m *MockServerRepoForPermissions) RemoveBan(ctx context.Context, serverID, userID uuid.UUID) error {
	args := m.Called(ctx, serverID, userID)
	return args.Error(0)
}

func (m *MockServerRepoForPermissions) GetBans(ctx context.Context, serverID uuid.UUID) ([]*models.Ban, error) {
	args := m.Called(ctx, serverID)
	return args.Get(0).([]*models.Ban), args.Error(1)
}

func (m *MockServerRepoForPermissions) CreateInvite(ctx context.Context, invite *models.Invite) error {
	args := m.Called(ctx, invite)
	return args.Error(0)
}

func (m *MockServerRepoForPermissions) GetInvite(ctx context.Context, code string) (*models.Invite, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Invite), args.Error(1)
}

func (m *MockServerRepoForPermissions) GetInvites(ctx context.Context, serverID uuid.UUID) ([]*models.Invite, error) {
	args := m.Called(ctx, serverID)
	return args.Get(0).([]*models.Invite), args.Error(1)
}

func (m *MockServerRepoForPermissions) DeleteInvite(ctx context.Context, code string) error {
	args := m.Called(ctx, code)
	return args.Error(0)
}

func (m *MockServerRepoForPermissions) IncrementInviteUses(ctx context.Context, code string) error {
	args := m.Called(ctx, code)
	return args.Error(0)
}

func (m *MockServerRepoForPermissions) GetInviteByVanityCode(ctx context.Context, vanityCode string) (*models.Invite, error) {
	args := m.Called(ctx, vanityCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Invite), args.Error(1)
}

func (m *MockServerRepoForPermissions) LogInviteUse(ctx context.Context, log *models.InviteUseLog) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

func (m *MockServerRepoForPermissions) GetInviteUseLogs(ctx context.Context, inviteCode string) ([]models.InviteUseLog, error) {
	args := m.Called(ctx, inviteCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.InviteUseLog), args.Error(1)
}

func (m *MockServerRepoForPermissions) GetServerInviteUseLogs(ctx context.Context, serverID uuid.UUID) ([]models.InviteUseLog, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.InviteUseLog), args.Error(1)
}

// MockRoleRepoForPermissions is a mock for RoleRepository
type MockRoleRepoForPermissions struct {
	mock.Mock
}

func (m *MockRoleRepoForPermissions) Create(ctx context.Context, role *models.Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *MockRoleRepoForPermissions) GetByID(ctx context.Context, id uuid.UUID) (*models.Role, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Role), args.Error(1)
}

func (m *MockRoleRepoForPermissions) GetByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.Role, error) {
	args := m.Called(ctx, serverID)
	return args.Get(0).([]*models.Role), args.Error(1)
}

func (m *MockRoleRepoForPermissions) Update(ctx context.Context, role *models.Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *MockRoleRepoForPermissions) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRoleRepoForPermissions) UpdatePositions(ctx context.Context, serverID uuid.UUID, positions map[uuid.UUID]int) error {
	args := m.Called(ctx, serverID, positions)
	return args.Error(0)
}

func (m *MockRoleRepoForPermissions) AddRoleToMember(ctx context.Context, serverID, userID, roleID uuid.UUID) error {
	args := m.Called(ctx, serverID, userID, roleID)
	return args.Error(0)
}

func (m *MockRoleRepoForPermissions) RemoveRoleFromMember(ctx context.Context, serverID, userID, roleID uuid.UUID) error {
	args := m.Called(ctx, serverID, userID, roleID)
	return args.Error(0)
}

func (m *MockRoleRepoForPermissions) GetMemberRoles(ctx context.Context, serverID, userID uuid.UUID) ([]*models.Role, error) {
	args := m.Called(ctx, serverID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Role), args.Error(1)
}

func (m *MockRoleRepoForPermissions) GetMemberPermissions(ctx context.Context, serverID, userID uuid.UUID) (int64, error) {
	args := m.Called(ctx, serverID, userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockRoleRepoForPermissions) GetDefaultRole(ctx context.Context, serverID uuid.UUID) (*models.Role, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Role), args.Error(1)
}

func TestPermissionService_GetMemberPermissions_Owner(t *testing.T) {
	ctx := context.Background()
	serverID := uuid.New()
	ownerID := uuid.New()

	serverRepo := new(MockServerRepoForPermissions)
	roleRepo := new(MockRoleRepoForPermissions)

	server := &models.Server{
		ID:      serverID,
		OwnerID: ownerID,
	}

	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)

	svc := NewPermissionService(serverRepo, roleRepo, nil, nil)

	perms, err := svc.GetMemberPermissions(ctx, serverID, ownerID)

	assert.NoError(t, err)
	assert.Equal(t, models.PermissionAll|models.PermAdministrator, perms)
	serverRepo.AssertExpectations(t)
}

func TestPermissionService_GetMemberPermissions_Administrator(t *testing.T) {
	ctx := context.Background()
	serverID := uuid.New()
	ownerID := uuid.New()
	adminID := uuid.New()

	serverRepo := new(MockServerRepoForPermissions)
	roleRepo := new(MockRoleRepoForPermissions)

	server := &models.Server{
		ID:      serverID,
		OwnerID: ownerID,
	}

	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)
	roleRepo.On("GetMemberPermissions", ctx, serverID, adminID).Return(models.PermAdministrator, nil)
	roleRepo.On("GetDefaultRole", ctx, serverID).Return(&models.Role{
		Permissions: models.DefaultPermissions,
	}, nil)

	svc := NewPermissionService(serverRepo, roleRepo, nil, nil)

	perms, err := svc.GetMemberPermissions(ctx, serverID, adminID)

	assert.NoError(t, err)
	assert.Equal(t, models.PermissionAll|models.PermAdministrator, perms)
}

func TestPermissionService_GetMemberPermissions_RegularMember(t *testing.T) {
	ctx := context.Background()
	serverID := uuid.New()
	ownerID := uuid.New()
	memberID := uuid.New()

	serverRepo := new(MockServerRepoForPermissions)
	roleRepo := new(MockRoleRepoForPermissions)

	server := &models.Server{
		ID:      serverID,
		OwnerID: ownerID,
	}

	memberPerms := models.PermSendMessages | models.PermViewChannels

	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)
	roleRepo.On("GetMemberPermissions", ctx, serverID, memberID).Return(memberPerms, nil)
	roleRepo.On("GetDefaultRole", ctx, serverID).Return(&models.Role{
		Permissions: models.DefaultPermissions,
	}, nil)

	svc := NewPermissionService(serverRepo, roleRepo, nil, nil)

	perms, err := svc.GetMemberPermissions(ctx, serverID, memberID)

	assert.NoError(t, err)
	assert.True(t, models.HasPermission(perms, models.PermSendMessages))
	assert.True(t, models.HasPermission(perms, models.PermViewChannels))
	// Should also have default permissions
	assert.True(t, models.HasPermission(perms, models.DefaultPermissions&models.PermSendMessages))
}

func TestPermissionService_HasPermission(t *testing.T) {
	ctx := context.Background()
	serverID := uuid.New()
	ownerID := uuid.New()
	memberID := uuid.New()

	serverRepo := new(MockServerRepoForPermissions)
	roleRepo := new(MockRoleRepoForPermissions)

	server := &models.Server{
		ID:      serverID,
		OwnerID: ownerID,
	}

	// Member only has SEND_MESSAGES
	memberPerms := models.PermSendMessages

	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)
	roleRepo.On("GetMemberPermissions", ctx, serverID, memberID).Return(memberPerms, nil)
	roleRepo.On("GetDefaultRole", ctx, serverID).Return(&models.Role{
		Permissions: 0, // No default permissions for this test
	}, nil)

	svc := NewPermissionService(serverRepo, roleRepo, nil, nil)

	// Should have SEND_MESSAGES
	has, err := svc.HasPermission(ctx, serverID, memberID, models.PermSendMessages)
	assert.NoError(t, err)
	assert.True(t, has)

	// Should not have MANAGE_MESSAGES
	has, err = svc.HasPermission(ctx, serverID, memberID, models.PermManageMessages)
	assert.NoError(t, err)
	assert.False(t, has)
}

func TestPermissionService_RequirePermission_Success(t *testing.T) {
	ctx := context.Background()
	serverID := uuid.New()
	ownerID := uuid.New()

	serverRepo := new(MockServerRepoForPermissions)
	roleRepo := new(MockRoleRepoForPermissions)

	server := &models.Server{
		ID:      serverID,
		OwnerID: ownerID,
	}

	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)

	svc := NewPermissionService(serverRepo, roleRepo, nil, nil)

	// Owner should have any permission
	err := svc.RequirePermission(ctx, serverID, ownerID, models.PermManageServer)
	assert.NoError(t, err)
}

func TestPermissionService_RequirePermission_Denied(t *testing.T) {
	ctx := context.Background()
	serverID := uuid.New()
	ownerID := uuid.New()
	memberID := uuid.New()

	serverRepo := new(MockServerRepoForPermissions)
	roleRepo := new(MockRoleRepoForPermissions)

	server := &models.Server{
		ID:      serverID,
		OwnerID: ownerID,
	}

	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)
	roleRepo.On("GetMemberPermissions", ctx, serverID, memberID).Return(int64(0), nil)
	roleRepo.On("GetDefaultRole", ctx, serverID).Return(&models.Role{
		Permissions: 0,
	}, nil)

	svc := NewPermissionService(serverRepo, roleRepo, nil, nil)

	// Member without permissions should get error
	err := svc.RequirePermission(ctx, serverID, memberID, models.PermManageServer)
	assert.Error(t, err)
	assert.Equal(t, ErrMissingManageServer, err)
}

func TestPermissionService_HasChannelPermission_DM(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	serverRepo := new(MockServerRepoForPermissions)
	roleRepo := new(MockRoleRepoForPermissions)

	// DM channel (no ServerID)
	channel := &models.Channel{
		ID:       uuid.New(),
		ServerID: nil,
		Type:     models.ChannelTypeDM,
	}

	svc := NewPermissionService(serverRepo, roleRepo, nil, nil)

	// DM channels should always return true
	has, err := svc.HasChannelPermission(ctx, channel, userID, models.PermSendMessages)
	assert.NoError(t, err)
	assert.True(t, has)
}

func TestPermissionService_IsServerOwner(t *testing.T) {
	ctx := context.Background()
	serverID := uuid.New()
	ownerID := uuid.New()
	memberID := uuid.New()

	serverRepo := new(MockServerRepoForPermissions)
	roleRepo := new(MockRoleRepoForPermissions)

	server := &models.Server{
		ID:      serverID,
		OwnerID: ownerID,
	}

	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)

	svc := NewPermissionService(serverRepo, roleRepo, nil, nil)

	isOwner, err := svc.IsServerOwner(ctx, serverID, ownerID)
	assert.NoError(t, err)
	assert.True(t, isOwner)

	isOwner, err = svc.IsServerOwner(ctx, serverID, memberID)
	assert.NoError(t, err)
	assert.False(t, isOwner)
}

func TestPermissionService_CanManageRole(t *testing.T) {
	ctx := context.Background()
	serverID := uuid.New()
	ownerID := uuid.New()
	adminID := uuid.New()
	memberID := uuid.New()

	serverRepo := new(MockServerRepoForPermissions)
	roleRepo := new(MockRoleRepoForPermissions)

	server := &models.Server{
		ID:      serverID,
		OwnerID: ownerID,
	}

	// Admin role at position 10
	adminRole := &models.Role{
		ID:          uuid.New(),
		ServerID:    serverID,
		Position:    10,
		Permissions: models.PermManageRoles,
	}

	// Moderator role at position 5
	modRole := &models.Role{
		ID:       uuid.New(),
		ServerID: serverID,
		Position: 5,
	}

	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)

	// Admin has MANAGE_ROLES and is at position 10
	roleRepo.On("GetMemberPermissions", ctx, serverID, adminID).Return(models.PermManageRoles, nil)
	roleRepo.On("GetDefaultRole", ctx, serverID).Return(&models.Role{Permissions: 0}, nil)
	roleRepo.On("GetMemberRoles", ctx, serverID, adminID).Return([]*models.Role{adminRole}, nil)

	// Member has no MANAGE_ROLES
	roleRepo.On("GetMemberPermissions", ctx, serverID, memberID).Return(int64(0), nil)
	roleRepo.On("GetMemberRoles", ctx, serverID, memberID).Return([]*models.Role{}, nil)

	svc := NewPermissionService(serverRepo, roleRepo, nil, nil)

	// Owner can manage any role
	canManage, err := svc.CanManageRole(ctx, serverID, ownerID, modRole)
	assert.NoError(t, err)
	assert.True(t, canManage)

	// Admin can manage roles below their position
	canManage, err = svc.CanManageRole(ctx, serverID, adminID, modRole)
	assert.NoError(t, err)
	assert.True(t, canManage)

	// Admin cannot manage roles at or above their position
	canManage, err = svc.CanManageRole(ctx, serverID, adminID, adminRole)
	assert.NoError(t, err)
	assert.False(t, canManage)

	// Member without MANAGE_ROLES cannot manage any role
	canManage, err = svc.CanManageRole(ctx, serverID, memberID, modRole)
	assert.NoError(t, err)
	assert.False(t, canManage)
}

func TestPermissionService_CanManageMember(t *testing.T) {
	ctx := context.Background()
	serverID := uuid.New()
	ownerID := uuid.New()
	adminID := uuid.New()
	memberID := uuid.New()

	serverRepo := new(MockServerRepoForPermissions)
	roleRepo := new(MockRoleRepoForPermissions)

	server := &models.Server{
		ID:      serverID,
		OwnerID: ownerID,
	}

	adminRoles := []*models.Role{{Position: 10}}
	memberRoles := []*models.Role{{Position: 5}}

	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)
	roleRepo.On("GetMemberRoles", ctx, serverID, adminID).Return(adminRoles, nil)
	roleRepo.On("GetMemberRoles", ctx, serverID, memberID).Return(memberRoles, nil)

	svc := NewPermissionService(serverRepo, roleRepo, nil, nil)

	// Owner can manage anyone
	canManage, err := svc.CanManageMember(ctx, serverID, ownerID, adminID)
	assert.NoError(t, err)
	assert.True(t, canManage)

	// Admin can manage member below them
	canManage, err = svc.CanManageMember(ctx, serverID, adminID, memberID)
	assert.NoError(t, err)
	assert.True(t, canManage)

	// Member cannot manage admin above them
	canManage, err = svc.CanManageMember(ctx, serverID, memberID, adminID)
	assert.NoError(t, err)
	assert.False(t, canManage)

	// Cannot manage yourself
	canManage, err = svc.CanManageMember(ctx, serverID, adminID, adminID)
	assert.NoError(t, err)
	assert.False(t, canManage)

	// Cannot manage owner
	canManage, err = svc.CanManageMember(ctx, serverID, adminID, ownerID)
	assert.NoError(t, err)
	assert.False(t, canManage)
}

func TestPermissionError(t *testing.T) {
	tests := []struct {
		permission int64
		expected   error
	}{
		{models.PermSendMessages, ErrMissingSendMessages},
		{models.PermReadMessageHistory, ErrMissingReadMessages},
		{models.PermManageMessages, ErrMissingManageMessages},
		{models.PermAddReactions, ErrMissingAddReactions},
		{models.PermManageRoles, ErrMissingManageRoles},
		{models.PermManageChannels, ErrMissingManageChannels},
		{models.PermKickMembers, ErrMissingKickMembers},
		{models.PermBanMembers, ErrMissingBanMembers},
		{models.PermCreateInvite, ErrMissingCreateInvite},
		{models.PermManageServer, ErrMissingManageServer},
		{models.PermManageWebhooks, ErrMissingManageWebhooks},
		{models.PermManageThreads, ErrMissingManageThreads},
		{models.PermAdministrator, ErrMissingAdministrator},
		{models.PermMoveMembers, ErrMissingMoveMembers},
		{models.PermMuteMembers, ErrMissingMuteMembers},
		{models.PermManageEmoji, ErrMissingManageEmojis},
		{models.PermViewChannels, ErrMissingViewChannels},
		{int64(999), ErrMissingPermission}, // Unknown permission
	}

	for _, tt := range tests {
		err := permissionError(tt.permission)
		assert.Equal(t, tt.expected, err)
	}
}

// MockChannelRepoForPermissions is a mock implementation of ChannelRepository for permission tests
type MockChannelRepoForPermissions struct {
	mock.Mock
}

func (m *MockChannelRepoForPermissions) Create(ctx context.Context, channel *models.Channel) error {
	args := m.Called(ctx, channel)
	return args.Error(0)
}

func (m *MockChannelRepoForPermissions) GetByID(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Channel), args.Error(1)
}

func (m *MockChannelRepoForPermissions) Update(ctx context.Context, channel *models.Channel) error {
	args := m.Called(ctx, channel)
	return args.Error(0)
}

func (m *MockChannelRepoForPermissions) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockChannelRepoForPermissions) GetByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.Channel, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Channel), args.Error(1)
}

func (m *MockChannelRepoForPermissions) GetDMChannel(ctx context.Context, user1ID, user2ID uuid.UUID) (*models.Channel, error) {
	args := m.Called(ctx, user1ID, user2ID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Channel), args.Error(1)
}

func (m *MockChannelRepoForPermissions) GetUserDMs(ctx context.Context, userID uuid.UUID) ([]*models.Channel, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Channel), args.Error(1)
}

func (m *MockChannelRepoForPermissions) UpdateLastMessage(ctx context.Context, channelID, messageID uuid.UUID, at time.Time) error {
	args := m.Called(ctx, channelID, messageID, at)
	return args.Error(0)
}

func (m *MockChannelRepoForPermissions) GetPermissionOverrides(ctx context.Context, channelID uuid.UUID) ([]models.PermissionOverride, error) {
	args := m.Called(ctx, channelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.PermissionOverride), args.Error(1)
}

func (m *MockChannelRepoForPermissions) AddRecipient(ctx context.Context, channelID, userID uuid.UUID) error {
	args := m.Called(ctx, channelID, userID)
	return args.Error(0)
}

func (m *MockChannelRepoForPermissions) RemoveRecipient(ctx context.Context, channelID, userID uuid.UUID) error {
	args := m.Called(ctx, channelID, userID)
	return args.Error(0)
}

func (m *MockChannelRepoForPermissions) CountRecipients(ctx context.Context, channelID uuid.UUID) (int, error) {
	args := m.Called(ctx, channelID)
	return args.Int(0), args.Error(1)
}

func TestPermissionService_GetChannelPermissions_WithOverrides(t *testing.T) {
	ctx := context.Background()
	serverID := uuid.New()
	channelID := uuid.New()
	ownerID := uuid.New()
	memberID := uuid.New()
	roleID := uuid.New()

	t.Run("channel override denies permission", func(t *testing.T) {
		serverRepo := new(MockServerRepoForPermissions)
		roleRepo := new(MockRoleRepoForPermissions)
		channelRepo := new(MockChannelRepoForPermissions)

		server := &models.Server{
			ID:      serverID,
			OwnerID: ownerID,
		}

		channel := &models.Channel{
			ID:       channelID,
			ServerID: &serverID,
		}

		member := &models.Member{
			UserID:   memberID,
			ServerID: serverID,
			Roles:    []uuid.UUID{roleID},
		}

		role := &models.Role{
			ID:          roleID,
			ServerID:    serverID,
			Position:    1,
			Permissions: models.PermSendMessages | models.PermReadMessageHistory,
		}

		defaultRole := &models.Role{
			ID:          serverID, // @everyone has same ID as server
			ServerID:    serverID,
			Position:    0,
			Permissions: models.DefaultPermissions,
			IsDefault:   true,
		}

		// Channel override denies SEND_MESSAGES for @everyone role
		overrides := []models.PermissionOverride{
			{
				ChannelID:  channelID,
				TargetType: "role",
				TargetID:   serverID, // @everyone
				Allow:      0,
				Deny:       models.PermSendMessages,
			},
		}

		serverRepo.On("GetByID", ctx, serverID).Return(server, nil)
		serverRepo.On("GetMember", ctx, serverID, memberID).Return(member, nil)
		roleRepo.On("GetByServerID", ctx, serverID).Return([]*models.Role{defaultRole, role}, nil)
		channelRepo.On("GetPermissionOverrides", ctx, channelID).Return(overrides, nil)

		svc := NewPermissionService(serverRepo, roleRepo, channelRepo, nil)

		perms, err := svc.GetChannelPermissions(ctx, channel, memberID)
		assert.NoError(t, err)

		// SEND_MESSAGES should be denied by the channel override
		assert.False(t, models.HasPermission(perms, models.PermSendMessages))
		// But READ_MESSAGE_HISTORY should still be allowed
		assert.True(t, models.HasPermission(perms, models.PermReadMessageHistory))
	})

	t.Run("channel override allows permission denied at role level", func(t *testing.T) {
		serverRepo := new(MockServerRepoForPermissions)
		roleRepo := new(MockRoleRepoForPermissions)
		channelRepo := new(MockChannelRepoForPermissions)

		server := &models.Server{
			ID:      serverID,
			OwnerID: ownerID,
		}

		channel := &models.Channel{
			ID:       channelID,
			ServerID: &serverID,
		}

		member := &models.Member{
			UserID:   memberID,
			ServerID: serverID,
			Roles:    []uuid.UUID{roleID},
		}

		// Role does NOT have MANAGE_MESSAGES
		role := &models.Role{
			ID:          roleID,
			ServerID:    serverID,
			Position:    1,
			Permissions: models.PermSendMessages | models.PermReadMessageHistory,
		}

		defaultRole := &models.Role{
			ID:          serverID,
			ServerID:    serverID,
			Position:    0,
			Permissions: 0, // No permissions
			IsDefault:   true,
		}

		// Channel override allows MANAGE_MESSAGES for this specific role
		overrides := []models.PermissionOverride{
			{
				ChannelID:  channelID,
				TargetType: "role",
				TargetID:   roleID,
				Allow:      models.PermManageMessages,
				Deny:       0,
			},
		}

		serverRepo.On("GetByID", ctx, serverID).Return(server, nil)
		serverRepo.On("GetMember", ctx, serverID, memberID).Return(member, nil)
		roleRepo.On("GetByServerID", ctx, serverID).Return([]*models.Role{defaultRole, role}, nil)
		channelRepo.On("GetPermissionOverrides", ctx, channelID).Return(overrides, nil)

		svc := NewPermissionService(serverRepo, roleRepo, channelRepo, nil)

		perms, err := svc.GetChannelPermissions(ctx, channel, memberID)
		assert.NoError(t, err)

		// MANAGE_MESSAGES should be allowed by the channel override
		assert.True(t, models.HasPermission(perms, models.PermManageMessages))
	})

	t.Run("user-specific override takes priority", func(t *testing.T) {
		serverRepo := new(MockServerRepoForPermissions)
		roleRepo := new(MockRoleRepoForPermissions)
		channelRepo := new(MockChannelRepoForPermissions)

		server := &models.Server{
			ID:      serverID,
			OwnerID: ownerID,
		}

		channel := &models.Channel{
			ID:       channelID,
			ServerID: &serverID,
		}

		member := &models.Member{
			UserID:   memberID,
			ServerID: serverID,
			Roles:    []uuid.UUID{roleID},
		}

		role := &models.Role{
			ID:          roleID,
			ServerID:    serverID,
			Position:    1,
			Permissions: models.PermSendMessages,
		}

		defaultRole := &models.Role{
			ID:          serverID,
			ServerID:    serverID,
			Position:    0,
			Permissions: 0,
			IsDefault:   true,
		}

		// Role override denies SEND_MESSAGES
		// But user-specific override allows it
		overrides := []models.PermissionOverride{
			{
				ChannelID:  channelID,
				TargetType: "role",
				TargetID:   roleID,
				Allow:      0,
				Deny:       models.PermSendMessages,
			},
			{
				ChannelID:  channelID,
				TargetType: "user",
				TargetID:   memberID,
				Allow:      models.PermSendMessages,
				Deny:       0,
			},
		}

		serverRepo.On("GetByID", ctx, serverID).Return(server, nil)
		serverRepo.On("GetMember", ctx, serverID, memberID).Return(member, nil)
		roleRepo.On("GetByServerID", ctx, serverID).Return([]*models.Role{defaultRole, role}, nil)
		channelRepo.On("GetPermissionOverrides", ctx, channelID).Return(overrides, nil)

		svc := NewPermissionService(serverRepo, roleRepo, channelRepo, nil)

		perms, err := svc.GetChannelPermissions(ctx, channel, memberID)
		assert.NoError(t, err)

		// User-specific override should allow SEND_MESSAGES even though role override denies it
		assert.True(t, models.HasPermission(perms, models.PermSendMessages))
	})

	t.Run("owner bypasses all overrides", func(t *testing.T) {
		serverRepo := new(MockServerRepoForPermissions)
		roleRepo := new(MockRoleRepoForPermissions)
		channelRepo := new(MockChannelRepoForPermissions)

		server := &models.Server{
			ID:      serverID,
			OwnerID: ownerID,
		}

		channel := &models.Channel{
			ID:       channelID,
			ServerID: &serverID,
		}

		// Channel override denies everything for @everyone
		overrides := []models.PermissionOverride{
			{
				ChannelID:  channelID,
				TargetType: "role",
				TargetID:   serverID,
				Allow:      0,
				Deny:       models.PermissionAll,
			},
		}

		serverRepo.On("GetByID", ctx, serverID).Return(server, nil)
		channelRepo.On("GetPermissionOverrides", ctx, channelID).Return(overrides, nil)

		svc := NewPermissionService(serverRepo, roleRepo, channelRepo, nil)

		// Owner should have all permissions regardless of overrides
		perms, err := svc.GetChannelPermissions(ctx, channel, ownerID)
		assert.NoError(t, err)
		assert.Equal(t, models.PermissionAll|models.PermAdministrator, perms)
	})
}

func TestPermissionService_RequireChannelPermission(t *testing.T) {
	ctx := context.Background()
	serverID := uuid.New()
	channelID := uuid.New()
	ownerID := uuid.New()
	memberID := uuid.New()

	t.Run("returns error when permission denied", func(t *testing.T) {
		serverRepo := new(MockServerRepoForPermissions)
		roleRepo := new(MockRoleRepoForPermissions)
		channelRepo := new(MockChannelRepoForPermissions)

		server := &models.Server{
			ID:      serverID,
			OwnerID: ownerID,
		}

		channel := &models.Channel{
			ID:       channelID,
			ServerID: &serverID,
		}

		member := &models.Member{
			UserID:   memberID,
			ServerID: serverID,
			Roles:    []uuid.UUID{},
		}

		defaultRole := &models.Role{
			ID:          serverID,
			ServerID:    serverID,
			Position:    0,
			Permissions: 0, // No permissions
			IsDefault:   true,
		}

		serverRepo.On("GetByID", ctx, serverID).Return(server, nil)
		serverRepo.On("GetMember", ctx, serverID, memberID).Return(member, nil)
		roleRepo.On("GetByServerID", ctx, serverID).Return([]*models.Role{defaultRole}, nil)
		channelRepo.On("GetPermissionOverrides", ctx, channelID).Return([]models.PermissionOverride{}, nil)

		svc := NewPermissionService(serverRepo, roleRepo, channelRepo, nil)

		err := svc.RequireChannelPermission(ctx, channel, memberID, models.PermSendMessages)
		assert.Error(t, err)
		assert.Equal(t, ErrMissingSendMessages, err)
	})

	t.Run("returns nil when permission granted", func(t *testing.T) {
		serverRepo := new(MockServerRepoForPermissions)
		roleRepo := new(MockRoleRepoForPermissions)
		channelRepo := new(MockChannelRepoForPermissions)

		server := &models.Server{
			ID:      serverID,
			OwnerID: ownerID,
		}

		channel := &models.Channel{
			ID:       channelID,
			ServerID: &serverID,
		}

		member := &models.Member{
			UserID:   memberID,
			ServerID: serverID,
			Roles:    []uuid.UUID{},
		}

		defaultRole := &models.Role{
			ID:          serverID,
			ServerID:    serverID,
			Position:    0,
			Permissions: models.PermSendMessages,
			IsDefault:   true,
		}

		serverRepo.On("GetByID", ctx, serverID).Return(server, nil)
		serverRepo.On("GetMember", ctx, serverID, memberID).Return(member, nil)
		roleRepo.On("GetByServerID", ctx, serverID).Return([]*models.Role{defaultRole}, nil)
		channelRepo.On("GetPermissionOverrides", ctx, channelID).Return([]models.PermissionOverride{}, nil)

		svc := NewPermissionService(serverRepo, roleRepo, channelRepo, nil)

		err := svc.RequireChannelPermission(ctx, channel, memberID, models.PermSendMessages)
		assert.NoError(t, err)
	})
}
