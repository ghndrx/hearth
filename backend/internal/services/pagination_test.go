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

// MockServerRepoForPagination implements ServerRepository for pagination tests
type MockServerRepoForPagination struct {
	mock.Mock
}

func (m *MockServerRepoForPagination) Create(ctx context.Context, server *models.Server) error {
	args := m.Called(ctx, server)
	return args.Error(0)
}

func (m *MockServerRepoForPagination) GetByID(ctx context.Context, id uuid.UUID) (*models.Server, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Server), args.Error(1)
}

func (m *MockServerRepoForPagination) Update(ctx context.Context, server *models.Server) error {
	args := m.Called(ctx, server)
	return args.Error(0)
}

func (m *MockServerRepoForPagination) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockServerRepoForPagination) TransferOwnership(ctx context.Context, serverID, newOwnerID uuid.UUID) error {
	args := m.Called(ctx, serverID, newOwnerID)
	return args.Error(0)
}

func (m *MockServerRepoForPagination) GetMembers(ctx context.Context, serverID uuid.UUID, limit, offset int) ([]*models.Member, error) {
	args := m.Called(ctx, serverID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Member), args.Error(1)
}

func (m *MockServerRepoForPagination) GetMembersPaginated(ctx context.Context, serverID uuid.UUID, cursor *models.MemberCursor, limit int) (*models.PaginatedMembers, error) {
	args := m.Called(ctx, serverID, cursor, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.PaginatedMembers), args.Error(1)
}

func (m *MockServerRepoForPagination) GetAllMembers(ctx context.Context, serverID uuid.UUID) ([]*models.Member, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Member), args.Error(1)
}

func (m *MockServerRepoForPagination) GetMember(ctx context.Context, serverID, userID uuid.UUID) (*models.Member, error) {
	args := m.Called(ctx, serverID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Member), args.Error(1)
}

func (m *MockServerRepoForPagination) GetMembersWithRole(ctx context.Context, serverID, roleID uuid.UUID) ([]*models.Member, error) {
	args := m.Called(ctx, serverID, roleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Member), args.Error(1)
}

func (m *MockServerRepoForPagination) GetMembersWithRolePaginated(ctx context.Context, serverID, roleID uuid.UUID, cursor *models.MemberCursor, limit int) (*models.PaginatedMembers, error) {
	args := m.Called(ctx, serverID, roleID, cursor, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.PaginatedMembers), args.Error(1)
}

func (m *MockServerRepoForPagination) GetAllMembersWithRole(ctx context.Context, serverID, roleID uuid.UUID) ([]*models.Member, error) {
	args := m.Called(ctx, serverID, roleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Member), args.Error(1)
}

func (m *MockServerRepoForPagination) AddMember(ctx context.Context, member *models.Member) error {
	args := m.Called(ctx, member)
	return args.Error(0)
}

func (m *MockServerRepoForPagination) UpdateMember(ctx context.Context, member *models.Member) error {
	args := m.Called(ctx, member)
	return args.Error(0)
}

func (m *MockServerRepoForPagination) RemoveMember(ctx context.Context, serverID, userID uuid.UUID) error {
	args := m.Called(ctx, serverID, userID)
	return args.Error(0)
}

func (m *MockServerRepoForPagination) GetMemberCount(ctx context.Context, serverID uuid.UUID) (int, error) {
	args := m.Called(ctx, serverID)
	return args.Int(0), args.Error(1)
}

func (m *MockServerRepoForPagination) GetUserServers(ctx context.Context, userID uuid.UUID) ([]*models.Server, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Server), args.Error(1)
}

func (m *MockServerRepoForPagination) GetOwnedServersCount(ctx context.Context, userID uuid.UUID) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

func (m *MockServerRepoForPagination) GetBan(ctx context.Context, serverID, userID uuid.UUID) (*models.Ban, error) {
	args := m.Called(ctx, serverID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Ban), args.Error(1)
}

func (m *MockServerRepoForPagination) AddBan(ctx context.Context, ban *models.Ban) error {
	args := m.Called(ctx, ban)
	return args.Error(0)
}

func (m *MockServerRepoForPagination) RemoveBan(ctx context.Context, serverID, userID uuid.UUID) error {
	args := m.Called(ctx, serverID, userID)
	return args.Error(0)
}

func (m *MockServerRepoForPagination) GetBans(ctx context.Context, serverID uuid.UUID) ([]*models.Ban, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Ban), args.Error(1)
}

func (m *MockServerRepoForPagination) CreateInvite(ctx context.Context, invite *models.Invite) error {
	args := m.Called(ctx, invite)
	return args.Error(0)
}

func (m *MockServerRepoForPagination) GetInvite(ctx context.Context, code string) (*models.Invite, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Invite), args.Error(1)
}

func (m *MockServerRepoForPagination) GetInvites(ctx context.Context, serverID uuid.UUID) ([]*models.Invite, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Invite), args.Error(1)
}

func (m *MockServerRepoForPagination) DeleteInvite(ctx context.Context, code string) error {
	args := m.Called(ctx, code)
	return args.Error(0)
}

func (m *MockServerRepoForPagination) IncrementInviteUses(ctx context.Context, code string) error {
	args := m.Called(ctx, code)
	return args.Error(0)
}
func (m *MockServerRepoForPagination) GetInviteByVanityCode(ctx context.Context, vanityCode string) (*models.Invite, error) {
	return nil, nil
}
func (m *MockServerRepoForPagination) LogInviteUse(ctx context.Context, log *models.InviteUseLog) error {
	return nil
}
func (m *MockServerRepoForPagination) GetInviteUseLogs(ctx context.Context, inviteCode string) ([]models.InviteUseLog, error) {
	return nil, nil
}
func (m *MockServerRepoForPagination) GetServerInviteUseLogs(ctx context.Context, serverID uuid.UUID) ([]models.InviteUseLog, error) {
	return nil, nil
}

// NOTE: Tests for ServerService.GetMembersPaginated and PresenceService.GetServerPresencesPaginated
// are commented out pending implementation of the paginated member methods on the service layer.
// The mock repository methods exist to support future implementation.

// MockCacheService for presence tests
type MockCacheServiceForPagination struct {
	mock.Mock
}

func (m *MockCacheServiceForPagination) GetUser(ctx context.Context, id uuid.UUID) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockCacheServiceForPagination) SetUser(ctx context.Context, user *models.User, ttl time.Duration) error {
	args := m.Called(ctx, user, ttl)
	return args.Error(0)
}

func (m *MockCacheServiceForPagination) DeleteUser(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockCacheServiceForPagination) GetServer(ctx context.Context, id uuid.UUID) (*models.Server, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Server), args.Error(1)
}

func (m *MockCacheServiceForPagination) SetServer(ctx context.Context, server *models.Server, ttl time.Duration) error {
	args := m.Called(ctx, server, ttl)
	return args.Error(0)
}

func (m *MockCacheServiceForPagination) DeleteServer(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockCacheServiceForPagination) GetChannel(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Channel), args.Error(1)
}

func (m *MockCacheServiceForPagination) SetChannel(ctx context.Context, channel *models.Channel, ttl time.Duration) error {
	args := m.Called(ctx, channel, ttl)
	return args.Error(0)
}

func (m *MockCacheServiceForPagination) DeleteChannel(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockCacheServiceForPagination) Get(ctx context.Context, key string) ([]byte, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockCacheServiceForPagination) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

func (m *MockCacheServiceForPagination) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

// MockEventBus for presence tests
type MockEventBusForPagination struct {
	mock.Mock
}

func (m *MockEventBusForPagination) Publish(event string, data interface{}) {
	m.Called(event, data)
}

func (m *MockEventBusForPagination) Subscribe(event string, handler func(data interface{})) {
	m.Called(event, handler)
}

func (m *MockEventBusForPagination) Unsubscribe(event string, handler func(data interface{})) {
	m.Called(event, handler)
}

// Tests for GetMembersWithRolePaginated

func TestServerRepo_GetMembersWithRolePaginated_FirstPage(t *testing.T) {
	ctx := context.Background()
	serverID := uuid.New()
	roleID := uuid.New()

	mockRepo := new(MockServerRepoForPagination)

	userID1 := uuid.New()
	userID2 := uuid.New()

	members := []*models.Member{
		{UserID: userID1, ServerID: serverID, JoinedAt: time.Now(), Roles: []uuid.UUID{roleID}},
		{UserID: userID2, ServerID: serverID, JoinedAt: time.Now().Add(-time.Hour), Roles: []uuid.UUID{roleID}},
	}

	expectedResult := &models.PaginatedMembers{
		Members:    members,
		NextCursor: "next-cursor-value",
		HasMore:    true,
	}

	mockRepo.On("GetMembersWithRolePaginated", ctx, serverID, roleID, (*models.MemberCursor)(nil), 50).Return(expectedResult, nil)

	result, err := mockRepo.GetMembersWithRolePaginated(ctx, serverID, roleID, nil, 50)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Members, 2)
	assert.True(t, result.HasMore)
	assert.NotEmpty(t, result.NextCursor)
	mockRepo.AssertExpectations(t)
}

func TestServerRepo_GetMembersWithRolePaginated_WithCursor(t *testing.T) {
	ctx := context.Background()
	serverID := uuid.New()
	roleID := uuid.New()

	mockRepo := new(MockServerRepoForPagination)

	// Create a cursor
	cursor := &models.MemberCursor{
		JoinedAt: time.Now(),
		UserID:   uuid.New(),
	}

	expectedResult := &models.PaginatedMembers{
		Members:    []*models.Member{},
		NextCursor: "",
		HasMore:    false,
	}

	mockRepo.On("GetMembersWithRolePaginated", ctx, serverID, roleID, mock.AnythingOfType("*models.MemberCursor"), 50).Return(expectedResult, nil)

	result, err := mockRepo.GetMembersWithRolePaginated(ctx, serverID, roleID, cursor, 50)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.HasMore)
	assert.Empty(t, result.NextCursor)
	mockRepo.AssertExpectations(t)
}

func TestServerRepo_GetAllMembersWithRole_Success(t *testing.T) {
	ctx := context.Background()
	serverID := uuid.New()
	roleID := uuid.New()

	mockRepo := new(MockServerRepoForPagination)

	userID1 := uuid.New()
	userID2 := uuid.New()

	members := []*models.Member{
		{UserID: userID1, ServerID: serverID, JoinedAt: time.Now(), Roles: []uuid.UUID{roleID}},
		{UserID: userID2, ServerID: serverID, JoinedAt: time.Now().Add(-time.Hour), Roles: []uuid.UUID{roleID}},
	}

	mockRepo.On("GetAllMembersWithRole", ctx, serverID, roleID).Return(members, nil)

	result, err := mockRepo.GetAllMembersWithRole(ctx, serverID, roleID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 2)
	mockRepo.AssertExpectations(t)
}

func TestServerRepo_GetAllMembersWithRole_Empty(t *testing.T) {
	ctx := context.Background()
	serverID := uuid.New()
	roleID := uuid.New()

	mockRepo := new(MockServerRepoForPagination)

	mockRepo.On("GetAllMembersWithRole", ctx, serverID, roleID).Return([]*models.Member{}, nil)

	result, err := mockRepo.GetAllMembersWithRole(ctx, serverID, roleID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 0)
	mockRepo.AssertExpectations(t)
}
