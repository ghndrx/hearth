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

// MockInviteRepository is a mock implementation of InviteRepository
type MockInviteRepository struct {
	mock.Mock
}

func (m *MockInviteRepository) Create(ctx context.Context, invite *models.Invite) error {
	args := m.Called(ctx, invite)
	return args.Error(0)
}

func (m *MockInviteRepository) GetByCode(ctx context.Context, code string) (*models.Invite, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Invite), args.Error(1)
}

func (m *MockInviteRepository) GetByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.Invite, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Invite), args.Error(1)
}

func (m *MockInviteRepository) IncrementUses(ctx context.Context, code string) error {
	args := m.Called(ctx, code)
	return args.Error(0)
}

func (m *MockInviteRepository) Delete(ctx context.Context, code string) error {
	args := m.Called(ctx, code)
	return args.Error(0)
}

func (m *MockInviteRepository) DeleteExpired(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

// MockBanRepository is a mock implementation of BanRepository
type MockBanRepository struct {
	mock.Mock
}

func (m *MockBanRepository) Create(ctx context.Context, ban *models.Ban) error {
	args := m.Called(ctx, ban)
	return args.Error(0)
}

func (m *MockBanRepository) GetByServerAndUser(ctx context.Context, serverID, userID uuid.UUID) (*models.Ban, error) {
	args := m.Called(ctx, serverID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Ban), args.Error(1)
}

func (m *MockBanRepository) GetByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.Ban, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Ban), args.Error(1)
}

func (m *MockBanRepository) Delete(ctx context.Context, serverID, userID uuid.UUID) error {
	args := m.Called(ctx, serverID, userID)
	return args.Error(0)
}

// MockServerRepoForInvite is a mock implementation of ServerRepository for invite tests
type MockServerRepoForInvite struct {
	mock.Mock
}

func (m *MockServerRepoForInvite) Create(ctx context.Context, server *models.Server) error {
	args := m.Called(ctx, server)
	return args.Error(0)
}

func (m *MockServerRepoForInvite) GetByID(ctx context.Context, id uuid.UUID) (*models.Server, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Server), args.Error(1)
}

func (m *MockServerRepoForInvite) Update(ctx context.Context, server *models.Server) error {
	args := m.Called(ctx, server)
	return args.Error(0)
}

func (m *MockServerRepoForInvite) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockServerRepoForInvite) TransferOwnership(ctx context.Context, serverID, newOwnerID uuid.UUID) error {
	args := m.Called(ctx, serverID, newOwnerID)
	return args.Error(0)
}

func (m *MockServerRepoForInvite) GetMember(ctx context.Context, serverID, userID uuid.UUID) (*models.Member, error) {
	args := m.Called(ctx, serverID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Member), args.Error(1)
}

func (m *MockServerRepoForInvite) AddMember(ctx context.Context, member *models.Member) error {
	args := m.Called(ctx, member)
	return args.Error(0)
}

func (m *MockServerRepoForInvite) RemoveMember(ctx context.Context, serverID, userID uuid.UUID) error {
	args := m.Called(ctx, serverID, userID)
	return args.Error(0)
}

func (m *MockServerRepoForInvite) GetUserServers(ctx context.Context, userID uuid.UUID) ([]*models.Server, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Server), args.Error(1)
}

func (m *MockServerRepoForInvite) GetBan(ctx context.Context, serverID, userID uuid.UUID) (*models.Ban, error) {
	args := m.Called(ctx, serverID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Ban), args.Error(1)
}

func (m *MockServerRepoForInvite) AddBan(ctx context.Context, ban *models.Ban) error {
	args := m.Called(ctx, ban)
	return args.Error(0)
}

func (m *MockServerRepoForInvite) RemoveBan(ctx context.Context, serverID, userID uuid.UUID) error {
	args := m.Called(ctx, serverID, userID)
	return args.Error(0)
}

func (m *MockServerRepoForInvite) GetBans(ctx context.Context, serverID uuid.UUID) ([]*models.Ban, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Ban), args.Error(1)
}

func (m *MockServerRepoForInvite) UpdateMember(ctx context.Context, member *models.Member) error {
	args := m.Called(ctx, member)
	return args.Error(0)
}

func (m *MockServerRepoForInvite) GetMembers(ctx context.Context, serverID uuid.UUID, limit, offset int) ([]*models.Member, error) {
	args := m.Called(ctx, serverID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Member), args.Error(1)
}

func (m *MockServerRepoForInvite) GetMemberCount(ctx context.Context, serverID uuid.UUID) (int, error) {
	args := m.Called(ctx, serverID)
	return args.Get(0).(int), args.Error(1)
}

func (m *MockServerRepoForInvite) GetOwnedServersCount(ctx context.Context, userID uuid.UUID) (int, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int), args.Error(1)
}

func (m *MockServerRepoForInvite) CreateInvite(ctx context.Context, invite *models.Invite) error {
	args := m.Called(ctx, invite)
	return args.Error(0)
}

func (m *MockServerRepoForInvite) GetInvite(ctx context.Context, code string) (*models.Invite, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Invite), args.Error(1)
}

func (m *MockServerRepoForInvite) GetInvites(ctx context.Context, serverID uuid.UUID) ([]*models.Invite, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Invite), args.Error(1)
}

func (m *MockServerRepoForInvite) DeleteInvite(ctx context.Context, code string) error {
	args := m.Called(ctx, code)
	return args.Error(0)
}

func (m *MockServerRepoForInvite) IncrementInviteUses(ctx context.Context, code string) error {
	args := m.Called(ctx, code)
	return args.Error(0)
}

// Helper function to create an InviteService with mocks
func newTestInviteService() (*InviteService, *MockInviteRepository, *MockBanRepository, *MockServerRepoForInvite, *MockCacheService, *MockEventBus) {
	inviteRepo := new(MockInviteRepository)
	banRepo := new(MockBanRepository)
	serverRepo := new(MockServerRepoForInvite)
	cache := new(MockCacheService)
	eventBus := new(MockEventBus)

	service := NewInviteService(inviteRepo, banRepo, serverRepo, cache, eventBus)

	return service, inviteRepo, banRepo, serverRepo, cache, eventBus
}

// ============================================================================
// CreateInvite Tests
// ============================================================================

func TestInviteService_CreateInvite_Success(t *testing.T) {
	service, inviteRepo, _, serverRepo, _, _ := newTestInviteService()
	ctx := context.Background()

	serverID := uuid.New()
	channelID := uuid.New()
	creatorID := uuid.New()

	member := &models.Member{
		ServerID: serverID,
		UserID:   creatorID,
		JoinedAt: time.Now(),
	}

	serverRepo.On("GetMember", ctx, serverID, creatorID).Return(member, nil)
	inviteRepo.On("Create", ctx, mock.AnythingOfType("*models.Invite")).Return(nil)

	req := &CreateInviteRequest{
		ServerID:  serverID,
		ChannelID: channelID,
		CreatorID: creatorID,
		MaxUses:   10,
		MaxAge:    24 * time.Hour,
		Temporary: false,
	}

	invite, err := service.CreateInvite(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, invite)
	assert.Equal(t, serverID, invite.ServerID)
	assert.Equal(t, channelID, invite.ChannelID)
	assert.Equal(t, creatorID, invite.CreatorID)
	assert.Equal(t, 10, invite.MaxUses)
	assert.Equal(t, 0, invite.Uses)
	assert.NotEmpty(t, invite.Code)
	assert.NotNil(t, invite.ExpiresAt)
	assert.False(t, invite.Temporary)

	serverRepo.AssertExpectations(t)
	inviteRepo.AssertExpectations(t)
}

func TestInviteService_CreateInvite_PermanentNoExpiry(t *testing.T) {
	service, inviteRepo, _, serverRepo, _, _ := newTestInviteService()
	ctx := context.Background()

	serverID := uuid.New()
	channelID := uuid.New()
	creatorID := uuid.New()

	member := &models.Member{
		ServerID: serverID,
		UserID:   creatorID,
	}

	serverRepo.On("GetMember", ctx, serverID, creatorID).Return(member, nil)
	inviteRepo.On("Create", ctx, mock.AnythingOfType("*models.Invite")).Return(nil)

	req := &CreateInviteRequest{
		ServerID:  serverID,
		ChannelID: channelID,
		CreatorID: creatorID,
		MaxUses:   0, // unlimited
		MaxAge:    0, // never expires
	}

	invite, err := service.CreateInvite(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, invite)
	assert.Equal(t, 0, invite.MaxUses)
	assert.Nil(t, invite.ExpiresAt)
}

func TestInviteService_CreateInvite_NotServerMember(t *testing.T) {
	service, _, _, serverRepo, _, _ := newTestInviteService()
	ctx := context.Background()

	serverID := uuid.New()
	channelID := uuid.New()
	creatorID := uuid.New()

	serverRepo.On("GetMember", ctx, serverID, creatorID).Return(nil, nil)

	req := &CreateInviteRequest{
		ServerID:  serverID,
		ChannelID: channelID,
		CreatorID: creatorID,
	}

	invite, err := service.CreateInvite(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, ErrNotServerMember, err)
	assert.Nil(t, invite)
}

func TestInviteService_CreateInvite_RepositoryError(t *testing.T) {
	service, inviteRepo, _, serverRepo, _, _ := newTestInviteService()
	ctx := context.Background()

	serverID := uuid.New()
	channelID := uuid.New()
	creatorID := uuid.New()

	member := &models.Member{
		ServerID: serverID,
		UserID:   creatorID,
	}

	serverRepo.On("GetMember", ctx, serverID, creatorID).Return(member, nil)
	inviteRepo.On("Create", ctx, mock.AnythingOfType("*models.Invite")).Return(errors.New("database error"))

	req := &CreateInviteRequest{
		ServerID:  serverID,
		ChannelID: channelID,
		CreatorID: creatorID,
	}

	invite, err := service.CreateInvite(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, invite)
}

func TestInviteService_CreateInvite_TemporaryInvite(t *testing.T) {
	service, inviteRepo, _, serverRepo, _, _ := newTestInviteService()
	ctx := context.Background()

	serverID := uuid.New()
	channelID := uuid.New()
	creatorID := uuid.New()

	member := &models.Member{
		ServerID: serverID,
		UserID:   creatorID,
	}

	serverRepo.On("GetMember", ctx, serverID, creatorID).Return(member, nil)
	inviteRepo.On("Create", ctx, mock.AnythingOfType("*models.Invite")).Return(nil)

	req := &CreateInviteRequest{
		ServerID:  serverID,
		ChannelID: channelID,
		CreatorID: creatorID,
		Temporary: true,
	}

	invite, err := service.CreateInvite(ctx, req)

	assert.NoError(t, err)
	assert.True(t, invite.Temporary)
}

// ============================================================================
// GetInvite Tests
// ============================================================================

func TestGetInvite_Success(t *testing.T) {
	service, inviteRepo, _, serverRepo, _, _ := newTestInviteService()
	ctx := context.Background()

	serverID := uuid.New()
	code := "abc123"

	invite := &models.Invite{
		Code:      code,
		ServerID:  serverID,
		CreatorID: uuid.New(),
		CreatedAt: time.Now(),
	}

	server := &models.Server{
		ID:   serverID,
		Name: "Test Server",
	}

	inviteRepo.On("GetByCode", ctx, code).Return(invite, nil)
	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)

	result, err := service.GetInvite(ctx, code)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, code, result.Code)
	assert.Equal(t, server, result.Server)
}

func TestGetInvite_NotFound(t *testing.T) {
	service, inviteRepo, _, _, _, _ := newTestInviteService()
	ctx := context.Background()

	code := "nonexistent"
	inviteRepo.On("GetByCode", ctx, code).Return(nil, nil)

	result, err := service.GetInvite(ctx, code)

	assert.Error(t, err)
	assert.Equal(t, ErrInviteNotFound, err)
	assert.Nil(t, result)
}

func TestGetInvite_RepositoryError(t *testing.T) {
	service, inviteRepo, _, _, _, _ := newTestInviteService()
	ctx := context.Background()

	code := "abc123"
	inviteRepo.On("GetByCode", ctx, code).Return(nil, errors.New("database error"))

	result, err := service.GetInvite(ctx, code)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// ============================================================================
// UseInvite Tests
// ============================================================================

func TestUseInvite_Success(t *testing.T) {
	service, inviteRepo, banRepo, serverRepo, _, eventBus := newTestInviteService()
	ctx := context.Background()

	serverID := uuid.New()
	userID := uuid.New()
	code := "validcode"

	invite := &models.Invite{
		Code:      code,
		ServerID:  serverID,
		CreatorID: uuid.New(),
		MaxUses:   10,
		Uses:      5,
		CreatedAt: time.Now(),
	}

	server := &models.Server{
		ID:   serverID,
		Name: "Test Server",
	}

	inviteRepo.On("GetByCode", ctx, code).Return(invite, nil)
	banRepo.On("GetByServerAndUser", ctx, serverID, userID).Return(nil, nil)
	serverRepo.On("GetMember", ctx, serverID, userID).Return(nil, nil)
	serverRepo.On("AddMember", ctx, mock.AnythingOfType("*models.Member")).Return(nil)
	inviteRepo.On("IncrementUses", ctx, code).Return(nil)
	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)
	eventBus.On("Publish", "server.member_joined", mock.AnythingOfType("*services.MemberJoinedEvent")).Return()

	result, err := service.UseInvite(ctx, code, userID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, serverID, result.ID)
	assert.Equal(t, "Test Server", result.Name)

	inviteRepo.AssertExpectations(t)
	banRepo.AssertExpectations(t)
	serverRepo.AssertExpectations(t)
	eventBus.AssertExpectations(t)
}

func TestUseInvite_NotFound(t *testing.T) {
	service, inviteRepo, _, _, _, _ := newTestInviteService()
	ctx := context.Background()

	code := "nonexistent"
	userID := uuid.New()

	inviteRepo.On("GetByCode", ctx, code).Return(nil, nil)

	result, err := service.UseInvite(ctx, code, userID)

	assert.Error(t, err)
	assert.Equal(t, ErrInviteNotFound, err)
	assert.Nil(t, result)
}

func TestUseInvite_Expired(t *testing.T) {
	service, inviteRepo, _, _, _, _ := newTestInviteService()
	ctx := context.Background()

	code := "expiredcode"
	userID := uuid.New()
	expiredTime := time.Now().Add(-24 * time.Hour)

	invite := &models.Invite{
		Code:      code,
		ServerID:  uuid.New(),
		ExpiresAt: &expiredTime,
		CreatedAt: time.Now().Add(-48 * time.Hour),
	}

	inviteRepo.On("GetByCode", ctx, code).Return(invite, nil)

	result, err := service.UseInvite(ctx, code, userID)

	assert.Error(t, err)
	assert.Equal(t, ErrInviteExpired, err)
	assert.Nil(t, result)
}

func TestUseInvite_MaxUsesReached(t *testing.T) {
	service, inviteRepo, _, _, _, _ := newTestInviteService()
	ctx := context.Background()

	code := "maxedout"
	userID := uuid.New()

	invite := &models.Invite{
		Code:      code,
		ServerID:  uuid.New(),
		MaxUses:   10,
		Uses:      10, // Max reached
		CreatedAt: time.Now(),
	}

	inviteRepo.On("GetByCode", ctx, code).Return(invite, nil)

	result, err := service.UseInvite(ctx, code, userID)

	assert.Error(t, err)
	assert.Equal(t, ErrInviteMaxUses, err)
	assert.Nil(t, result)
}

func TestUseInvite_UserBanned(t *testing.T) {
	service, inviteRepo, banRepo, _, _, _ := newTestInviteService()
	ctx := context.Background()

	serverID := uuid.New()
	userID := uuid.New()
	code := "validcode"

	invite := &models.Invite{
		Code:      code,
		ServerID:  serverID,
		CreatorID: uuid.New(),
		CreatedAt: time.Now(),
	}

	ban := &models.Ban{
		ServerID:  serverID,
		UserID:    userID,
		CreatedAt: time.Now(),
	}

	inviteRepo.On("GetByCode", ctx, code).Return(invite, nil)
	banRepo.On("GetByServerAndUser", ctx, serverID, userID).Return(ban, nil)

	result, err := service.UseInvite(ctx, code, userID)

	assert.Error(t, err)
	assert.Equal(t, ErrBannedFromServer, err)
	assert.Nil(t, result)
}

func TestUseInvite_AlreadyMember(t *testing.T) {
	service, inviteRepo, banRepo, serverRepo, _, _ := newTestInviteService()
	ctx := context.Background()

	serverID := uuid.New()
	userID := uuid.New()
	code := "validcode"

	invite := &models.Invite{
		Code:      code,
		ServerID:  serverID,
		CreatorID: uuid.New(),
		CreatedAt: time.Now(),
	}

	existingMember := &models.Member{
		ServerID: serverID,
		UserID:   userID,
		JoinedAt: time.Now(),
	}

	inviteRepo.On("GetByCode", ctx, code).Return(invite, nil)
	banRepo.On("GetByServerAndUser", ctx, serverID, userID).Return(nil, nil)
	serverRepo.On("GetMember", ctx, serverID, userID).Return(existingMember, nil)

	result, err := service.UseInvite(ctx, code, userID)

	assert.Error(t, err)
	assert.Equal(t, ErrAlreadyMember, err)
	assert.Nil(t, result)
}

func TestUseInvite_TemporaryInviteCreatesPendingMember(t *testing.T) {
	service, inviteRepo, banRepo, serverRepo, _, eventBus := newTestInviteService()
	ctx := context.Background()

	serverID := uuid.New()
	userID := uuid.New()
	code := "tempcode"

	invite := &models.Invite{
		Code:      code,
		ServerID:  serverID,
		CreatorID: uuid.New(),
		Temporary: true,
		CreatedAt: time.Now(),
	}

	server := &models.Server{
		ID:   serverID,
		Name: "Test Server",
	}

	inviteRepo.On("GetByCode", ctx, code).Return(invite, nil)
	banRepo.On("GetByServerAndUser", ctx, serverID, userID).Return(nil, nil)
	serverRepo.On("GetMember", ctx, serverID, userID).Return(nil, nil)
	serverRepo.On("AddMember", ctx, mock.MatchedBy(func(m *models.Member) bool {
		return m.Pending == true
	})).Return(nil)
	inviteRepo.On("IncrementUses", ctx, code).Return(nil)
	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)
	eventBus.On("Publish", "server.member_joined", mock.Anything).Return()

	result, err := service.UseInvite(ctx, code, userID)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	serverRepo.AssertExpectations(t)
}

// ============================================================================
// DeleteInvite Tests
// ============================================================================

func TestDeleteInvite_Success_ByCreator(t *testing.T) {
	service, inviteRepo, _, _, _, _ := newTestInviteService()
	ctx := context.Background()

	code := "deleteme"
	creatorID := uuid.New()

	invite := &models.Invite{
		Code:      code,
		ServerID:  uuid.New(),
		CreatorID: creatorID,
		CreatedAt: time.Now(),
	}

	inviteRepo.On("GetByCode", ctx, code).Return(invite, nil)
	inviteRepo.On("Delete", ctx, code).Return(nil)

	err := service.DeleteInvite(ctx, code, creatorID)

	assert.NoError(t, err)
	inviteRepo.AssertExpectations(t)
}

func TestDeleteInvite_Success_ByServerAdmin(t *testing.T) {
	service, inviteRepo, _, serverRepo, _, _ := newTestInviteService()
	ctx := context.Background()

	serverID := uuid.New()
	code := "deleteme"
	creatorID := uuid.New()
	adminID := uuid.New() // Different from creator

	invite := &models.Invite{
		Code:      code,
		ServerID:  serverID,
		CreatorID: creatorID,
		CreatedAt: time.Now(),
	}

	adminMember := &models.Member{
		ServerID: serverID,
		UserID:   adminID,
		JoinedAt: time.Now(),
	}

	inviteRepo.On("GetByCode", ctx, code).Return(invite, nil)
	serverRepo.On("GetMember", ctx, serverID, adminID).Return(adminMember, nil)
	inviteRepo.On("Delete", ctx, code).Return(nil)

	err := service.DeleteInvite(ctx, code, adminID)

	assert.NoError(t, err)
	inviteRepo.AssertExpectations(t)
	serverRepo.AssertExpectations(t)
}

func TestDeleteInvite_NotFound(t *testing.T) {
	service, inviteRepo, _, _, _, _ := newTestInviteService()
	ctx := context.Background()

	code := "nonexistent"
	userID := uuid.New()

	inviteRepo.On("GetByCode", ctx, code).Return(nil, nil)

	err := service.DeleteInvite(ctx, code, userID)

	assert.Error(t, err)
	assert.Equal(t, ErrInviteNotFound, err)
}

func TestDeleteInvite_NotMember(t *testing.T) {
	service, inviteRepo, _, serverRepo, _, _ := newTestInviteService()
	ctx := context.Background()

	serverID := uuid.New()
	code := "deleteme"
	creatorID := uuid.New()
	randomID := uuid.New()

	invite := &models.Invite{
		Code:      code,
		ServerID:  serverID,
		CreatorID: creatorID,
		CreatedAt: time.Now(),
	}

	inviteRepo.On("GetByCode", ctx, code).Return(invite, nil)
	serverRepo.On("GetMember", ctx, serverID, randomID).Return(nil, nil)

	err := service.DeleteInvite(ctx, code, randomID)

	assert.Error(t, err)
	assert.Equal(t, ErrNotServerMember, err)
}

// ============================================================================
// GetServerInvites Tests
// ============================================================================

func TestGetServerInvites_Success(t *testing.T) {
	service, inviteRepo, _, serverRepo, _, _ := newTestInviteService()
	ctx := context.Background()

	serverID := uuid.New()
	userID := uuid.New()

	member := &models.Member{
		ServerID: serverID,
		UserID:   userID,
		JoinedAt: time.Now(),
	}

	invites := []*models.Invite{
		{Code: "invite1", ServerID: serverID, CreatorID: userID},
		{Code: "invite2", ServerID: serverID, CreatorID: uuid.New()},
	}

	serverRepo.On("GetMember", ctx, serverID, userID).Return(member, nil)
	inviteRepo.On("GetByServerID", ctx, serverID).Return(invites, nil)

	result, err := service.GetServerInvites(ctx, serverID, userID)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestGetServerInvites_NotMember(t *testing.T) {
	service, _, _, serverRepo, _, _ := newTestInviteService()
	ctx := context.Background()

	serverID := uuid.New()
	userID := uuid.New()

	serverRepo.On("GetMember", ctx, serverID, userID).Return(nil, nil)

	result, err := service.GetServerInvites(ctx, serverID, userID)

	assert.Error(t, err)
	assert.Equal(t, ErrNotServerMember, err)
	assert.Nil(t, result)
}

// ============================================================================
// CleanupExpiredInvites Tests
// ============================================================================

func TestCleanupExpiredInvites_Success(t *testing.T) {
	service, inviteRepo, _, _, _, _ := newTestInviteService()
	ctx := context.Background()

	inviteRepo.On("DeleteExpired", ctx).Return(int64(5), nil)

	count, err := service.CleanupExpiredInvites(ctx)

	assert.NoError(t, err)
	assert.Equal(t, int64(5), count)
}

func TestCleanupExpiredInvites_Error(t *testing.T) {
	service, inviteRepo, _, _, _, _ := newTestInviteService()
	ctx := context.Background()

	inviteRepo.On("DeleteExpired", ctx).Return(int64(0), errors.New("database error"))

	count, err := service.CleanupExpiredInvites(ctx)

	assert.Error(t, err)
	assert.Equal(t, int64(0), count)
}

// ============================================================================
// BanMember Tests
// ============================================================================

func TestInviteService_BanMember_Success(t *testing.T) {
	service, _, banRepo, serverRepo, _, eventBus := newTestInviteService()
	ctx := context.Background()

	serverID := uuid.New()
	userID := uuid.New()
	moderatorID := uuid.New()
	reason := "Rule violation"

	moderatorMember := &models.Member{
		ServerID: serverID,
		UserID:   moderatorID,
		JoinedAt: time.Now(),
	}

	serverRepo.On("GetMember", ctx, serverID, moderatorID).Return(moderatorMember, nil)
	serverRepo.On("RemoveMember", ctx, serverID, userID).Return(nil)
	banRepo.On("Create", ctx, mock.AnythingOfType("*models.Ban")).Return(nil)
	eventBus.On("Publish", "server.member_banned", mock.AnythingOfType("*services.MemberBannedEvent")).Return()

	err := service.BanMember(ctx, serverID, userID, moderatorID, reason)

	assert.NoError(t, err)
	banRepo.AssertExpectations(t)
	serverRepo.AssertExpectations(t)
	eventBus.AssertExpectations(t)
}

func TestInviteService_BanMember_CannotBanSelf(t *testing.T) {
	service, _, _, serverRepo, _, _ := newTestInviteService()
	ctx := context.Background()

	serverID := uuid.New()
	userID := uuid.New() // Same as moderatorID

	member := &models.Member{
		ServerID: serverID,
		UserID:   userID,
		JoinedAt: time.Now(),
	}

	serverRepo.On("GetMember", ctx, serverID, userID).Return(member, nil)

	err := service.BanMember(ctx, serverID, userID, userID, "reason")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot ban yourself")
}

func TestInviteService_BanMember_NotMember(t *testing.T) {
	service, _, _, serverRepo, _, _ := newTestInviteService()
	ctx := context.Background()

	serverID := uuid.New()
	userID := uuid.New()
	moderatorID := uuid.New()

	serverRepo.On("GetMember", ctx, serverID, moderatorID).Return(nil, nil)

	err := service.BanMember(ctx, serverID, userID, moderatorID, "reason")

	assert.Error(t, err)
	assert.Equal(t, ErrNotServerMember, err)
}

// ============================================================================
// UnbanMember Tests
// ============================================================================

func TestInviteService_UnbanMember_Success(t *testing.T) {
	service, _, banRepo, serverRepo, _, eventBus := newTestInviteService()
	ctx := context.Background()

	serverID := uuid.New()
	userID := uuid.New()
	moderatorID := uuid.New()

	moderatorMember := &models.Member{
		ServerID: serverID,
		UserID:   moderatorID,
		JoinedAt: time.Now(),
	}

	serverRepo.On("GetMember", ctx, serverID, moderatorID).Return(moderatorMember, nil)
	banRepo.On("Delete", ctx, serverID, userID).Return(nil)
	eventBus.On("Publish", "server.member_unbanned", mock.AnythingOfType("*services.MemberUnbannedEvent")).Return()

	err := service.UnbanMember(ctx, serverID, userID, moderatorID)

	assert.NoError(t, err)
	banRepo.AssertExpectations(t)
	eventBus.AssertExpectations(t)
}

func TestInviteService_UnbanMember_NotMember(t *testing.T) {
	service, _, _, serverRepo, _, _ := newTestInviteService()
	ctx := context.Background()

	serverID := uuid.New()
	userID := uuid.New()
	moderatorID := uuid.New()

	serverRepo.On("GetMember", ctx, serverID, moderatorID).Return(nil, nil)

	err := service.UnbanMember(ctx, serverID, userID, moderatorID)

	assert.Error(t, err)
	assert.Equal(t, ErrNotServerMember, err)
}

// ============================================================================
// GetServerBans Tests
// ============================================================================

func TestInviteService_GetServerBans_Success(t *testing.T) {
	service, _, banRepo, serverRepo, _, _ := newTestInviteService()
	ctx := context.Background()

	serverID := uuid.New()
	userID := uuid.New()

	member := &models.Member{
		ServerID: serverID,
		UserID:   userID,
		JoinedAt: time.Now(),
	}

	reason1 := "Spam"
	reason2 := "Harassment"
	bans := []*models.Ban{
		{ServerID: serverID, UserID: uuid.New(), Reason: &reason1, CreatedAt: time.Now()},
		{ServerID: serverID, UserID: uuid.New(), Reason: &reason2, CreatedAt: time.Now()},
	}

	serverRepo.On("GetMember", ctx, serverID, userID).Return(member, nil)
	banRepo.On("GetByServerID", ctx, serverID).Return(bans, nil)

	result, err := service.GetServerBans(ctx, serverID, userID)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestInviteService_GetServerBans_NotMember(t *testing.T) {
	service, _, _, serverRepo, _, _ := newTestInviteService()
	ctx := context.Background()

	serverID := uuid.New()
	userID := uuid.New()

	serverRepo.On("GetMember", ctx, serverID, userID).Return(nil, nil)

	result, err := service.GetServerBans(ctx, serverID, userID)

	assert.Error(t, err)
	assert.Equal(t, ErrNotServerMember, err)
	assert.Nil(t, result)
}

// ============================================================================
// Helper Function Tests
// ============================================================================

func TestGenerateInviteCode(t *testing.T) {
	codes := make(map[string]bool)

	// Generate multiple codes and verify uniqueness
	for i := 0; i < 100; i++ {
		code, err := generateInviteCode()
		assert.NoError(t, err)
		assert.NotEmpty(t, code)
		assert.Len(t, code, 8) // base64 encoded 6 bytes = 8 characters

		// Check uniqueness
		assert.False(t, codes[code], "Generated duplicate code")
		codes[code] = true
	}
}
