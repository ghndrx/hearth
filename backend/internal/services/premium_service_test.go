package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"hearth/internal/models"
)

// MockPremiumRepo is a mock implementation of PremiumRepository
type MockPremiumRepo struct {
	mock.Mock
}

func (m *MockPremiumRepo) CreateSubscription(ctx context.Context, sub *models.Subscription) error {
	args := m.Called(ctx, sub)
	return args.Error(0)
}

func (m *MockPremiumRepo) GetSubscription(ctx context.Context, userID uuid.UUID) (*models.Subscription, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Subscription), args.Error(1)
}

func (m *MockPremiumRepo) GetSubscriptionByStripeID(ctx context.Context, stripeSubID string) (*models.Subscription, error) {
	args := m.Called(ctx, stripeSubID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Subscription), args.Error(1)
}

func (m *MockPremiumRepo) UpdateSubscription(ctx context.Context, sub *models.Subscription) error {
	args := m.Called(ctx, sub)
	return args.Error(0)
}

func (m *MockPremiumRepo) DeleteSubscription(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockPremiumRepo) CreateServerBoost(ctx context.Context, boost *models.ServerBoost) error {
	args := m.Called(ctx, boost)
	return args.Error(0)
}

func (m *MockPremiumRepo) GetServerBoost(ctx context.Context, serverID, userID uuid.UUID) (*models.ServerBoost, error) {
	args := m.Called(ctx, serverID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ServerBoost), args.Error(1)
}

func (m *MockPremiumRepo) GetServerBoosts(ctx context.Context, serverID uuid.UUID) ([]*models.ServerBoost, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ServerBoost), args.Error(1)
}

func (m *MockPremiumRepo) GetServerBoostCount(ctx context.Context, serverID uuid.UUID) (int, error) {
	args := m.Called(ctx, serverID)
	return args.Int(0), args.Error(1)
}

func (m *MockPremiumRepo) DeactivateServerBoost(ctx context.Context, serverID, userID uuid.UUID) error {
	args := m.Called(ctx, serverID, userID)
	return args.Error(0)
}

func (m *MockPremiumRepo) GetUserBoosts(ctx context.Context, userID uuid.UUID) ([]*models.ServerBoost, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ServerBoost), args.Error(1)
}

func (m *MockPremiumRepo) GetUserActiveBoostCount(ctx context.Context, userID uuid.UUID) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

func (m *MockPremiumRepo) UpdateServerBoostLevel(ctx context.Context, serverID uuid.UUID, boostCount int) error {
	args := m.Called(ctx, serverID, boostCount)
	return args.Error(0)
}

func (m *MockPremiumRepo) GetServerBoostLevel(ctx context.Context, serverID uuid.UUID) (*models.ServerPerks, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ServerPerks), args.Error(1)
}

func (m *MockPremiumRepo) CreateBillingCustomer(ctx context.Context, customer *models.Customer) error {
	args := m.Called(ctx, customer)
	return args.Error(0)
}

func (m *MockPremiumRepo) GetBillingCustomer(ctx context.Context, userID uuid.UUID) (*models.Customer, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Customer), args.Error(1)
}

func (m *MockPremiumRepo) GetBillingCustomerByExternalID(ctx context.Context, externalID string) (*models.Customer, error) {
	args := m.Called(ctx, externalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Customer), args.Error(1)
}

func (m *MockPremiumRepo) CreateBillingInvoice(ctx context.Context, invoice *models.BillingInvoice) error {
	args := m.Called(ctx, invoice)
	return args.Error(0)
}

func (m *MockPremiumRepo) GetBillingInvoices(ctx context.Context, userID uuid.UUID, limit int) ([]*models.BillingInvoice, error) {
	args := m.Called(ctx, userID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.BillingInvoice), args.Error(1)
}

func (m *MockPremiumRepo) CreatePaymentMethod(ctx context.Context, userID uuid.UUID, pm *models.PaymentMethod) error {
	args := m.Called(ctx, userID, pm)
	return args.Error(0)
}

func (m *MockPremiumRepo) GetPaymentMethods(ctx context.Context, userID uuid.UUID) ([]*models.PaymentMethod, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.PaymentMethod), args.Error(1)
}

func (m *MockPremiumRepo) GetDefaultPaymentMethod(ctx context.Context, userID uuid.UUID) (*models.PaymentMethod, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.PaymentMethod), args.Error(1)
}

func (m *MockPremiumRepo) DeletePaymentMethod(ctx context.Context, userID uuid.UUID, paymentMethodID string) error {
	args := m.Called(ctx, userID, paymentMethodID)
	return args.Error(0)
}

func (m *MockPremiumRepo) UpdateUserPremiumTier(ctx context.Context, userID uuid.UUID, tier models.PremiumTier) error {
	args := m.Called(ctx, userID, tier)
	return args.Error(0)
}

func (m *MockPremiumRepo) GetUserPremiumTier(ctx context.Context, userID uuid.UUID) (models.PremiumTier, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(models.PremiumTier), args.Error(1)
}

// minimalUserRepo satisfies UserRepository with just the methods PremiumService needs
type minimalUserRepo struct{}

func (m *minimalUserRepo) Create(ctx context.Context, user *models.User) error {
	return nil
}
func (m *minimalUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	return nil, nil
}
func (m *minimalUserRepo) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	return nil, nil
}
func (m *minimalUserRepo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	return nil, nil
}
func (m *minimalUserRepo) Update(ctx context.Context, user *models.User) error {
	return nil
}
func (m *minimalUserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}
func (m *minimalUserRepo) GetFriends(ctx context.Context, userID uuid.UUID) ([]*models.User, error) {
	return nil, nil
}
func (m *minimalUserRepo) AddFriend(ctx context.Context, userID, friendID uuid.UUID) error {
	return nil
}
func (m *minimalUserRepo) RemoveFriend(ctx context.Context, userID, friendID uuid.UUID) error {
	return nil
}
func (m *minimalUserRepo) GetBlockedUsers(ctx context.Context, userID uuid.UUID) ([]*models.User, error) {
	return nil, nil
}
func (m *minimalUserRepo) BlockUser(ctx context.Context, userID, blockedID uuid.UUID) error {
	return nil
}
func (m *minimalUserRepo) UnblockUser(ctx context.Context, userID, blockedID uuid.UUID) error {
	return nil
}
func (m *minimalUserRepo) GetRelationship(ctx context.Context, userID, targetID uuid.UUID) (int, error) {
	return 0, nil
}
func (m *minimalUserRepo) SendFriendRequest(ctx context.Context, senderID, receiverID uuid.UUID) error {
	return nil
}
func (m *minimalUserRepo) GetIncomingFriendRequests(ctx context.Context, userID uuid.UUID) ([]*models.User, error) {
	return nil, nil
}
func (m *minimalUserRepo) GetOutgoingFriendRequests(ctx context.Context, userID uuid.UUID) ([]*models.User, error) {
	return nil, nil
}
func (m *minimalUserRepo) AcceptFriendRequest(ctx context.Context, receiverID, senderID uuid.UUID) error {
	return nil
}
func (m *minimalUserRepo) DeclineFriendRequest(ctx context.Context, userID, otherID uuid.UUID) error {
	return nil
}
func (m *minimalUserRepo) UpdatePresence(ctx context.Context, userID uuid.UUID, status models.PresenceStatus) error {
	return nil
}
func (m *minimalUserRepo) GetPresence(ctx context.Context, userID uuid.UUID) (*models.Presence, error) {
	return nil, nil
}
func (m *minimalUserRepo) GetPresenceBulk(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*models.Presence, error) {
	return nil, nil
}
func (m *minimalUserRepo) GetCustomStatus(ctx context.Context, userID uuid.UUID) (*models.UserCustomStatus, error) {
	return nil, nil
}
func (m *minimalUserRepo) SetCustomStatus(ctx context.Context, status *models.UserCustomStatus) error {
	return nil
}
func (m *minimalUserRepo) DeleteCustomStatus(ctx context.Context, userID uuid.UUID) error {
	return nil
}

// minimalServerRepo satisfies ServerRepository with just the methods PremiumService needs
type minimalServerRepo struct{}

func (m *minimalServerRepo) Create(ctx context.Context, server *models.Server) error {
	return nil
}
func (m *minimalServerRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Server, error) {
	return nil, nil
}
func (m *minimalServerRepo) Update(ctx context.Context, server *models.Server) error {
	return nil
}
func (m *minimalServerRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}
func (m *minimalServerRepo) TransferOwnership(ctx context.Context, serverID, newOwnerID uuid.UUID) error {
	return nil
}
func (m *minimalServerRepo) GetMembers(ctx context.Context, serverID uuid.UUID, limit, offset int) ([]*models.Member, error) {
	return nil, nil
}
func (m *minimalServerRepo) GetMembersPaginated(ctx context.Context, serverID uuid.UUID, cursor *models.MemberCursor, limit int) (*models.PaginatedMembers, error) {
	return nil, nil
}
func (m *minimalServerRepo) GetMember(ctx context.Context, serverID, userID uuid.UUID) (*models.Member, error) {
	return nil, nil
}
func (m *minimalServerRepo) GetMembersWithRole(ctx context.Context, serverID, roleID uuid.UUID) ([]*models.Member, error) {
	return nil, nil
}
func (m *minimalServerRepo) AddMember(ctx context.Context, member *models.Member) error {
	return nil
}
func (m *minimalServerRepo) UpdateMember(ctx context.Context, member *models.Member) error {
	return nil
}
func (m *minimalServerRepo) RemoveMember(ctx context.Context, serverID, userID uuid.UUID) error {
	return nil
}
func (m *minimalServerRepo) GetMemberCount(ctx context.Context, serverID uuid.UUID) (int, error) {
	return 0, nil
}
func (m *minimalServerRepo) GetUserServers(ctx context.Context, userID uuid.UUID) ([]*models.Server, error) {
	return nil, nil
}
func (m *minimalServerRepo) GetOwnedServersCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return 0, nil
}
func (m *minimalServerRepo) GetBan(ctx context.Context, serverID, userID uuid.UUID) (*models.Ban, error) {
	return nil, nil
}
func (m *minimalServerRepo) AddBan(ctx context.Context, ban *models.Ban) error {
	return nil
}
func (m *minimalServerRepo) RemoveBan(ctx context.Context, serverID, userID uuid.UUID) error {
	return nil
}
func (m *minimalServerRepo) GetBans(ctx context.Context, serverID uuid.UUID) ([]*models.Ban, error) {
	return nil, nil
}
func (m *minimalServerRepo) CreateInvite(ctx context.Context, invite *models.Invite) error {
	return nil
}
func (m *minimalServerRepo) GetInvite(ctx context.Context, code string) (*models.Invite, error) {
	return nil, nil
}
func (m *minimalServerRepo) GetInviteByVanityCode(ctx context.Context, vanityCode string) (*models.Invite, error) {
	return nil, nil
}
func (m *minimalServerRepo) GetInvites(ctx context.Context, serverID uuid.UUID) ([]*models.Invite, error) {
	return nil, nil
}
func (m *minimalServerRepo) DeleteInvite(ctx context.Context, code string) error {
	return nil
}
func (m *minimalServerRepo) IncrementInviteUses(ctx context.Context, code string) error {
	return nil
}
func (m *minimalServerRepo) LogInviteUse(ctx context.Context, log *models.InviteUseLog) error {
	return nil
}
func (m *minimalServerRepo) GetInviteUseLogs(ctx context.Context, inviteCode string) ([]models.InviteUseLog, error) {
	return nil, nil
}
func (m *minimalServerRepo) GetServerInviteUseLogs(ctx context.Context, serverID uuid.UUID) ([]models.InviteUseLog, error) {
	return nil, nil
}

// newTestPremiumService creates a PremiumService for testing with mocks
func newTestPremiumService() (*PremiumService, *MockPremiumRepo) {
	repo := new(MockPremiumRepo)
	userRepo := &minimalUserRepo{}
	serverRepo := &minimalServerRepo{}

	service := &PremiumService{
		repo:       repo,
		userRepo:   userRepo,
		serverRepo: serverRepo,
	}

	return service, repo
}

func TestGetUserPremiumStatus_NoSubscription(t *testing.T) {
	svc, repo := newTestPremiumService()
	ctx := context.Background()
	userID := uuid.New()

	repo.On("GetSubscription", ctx, userID).Return(nil, nil)

	status, err := svc.GetUserPremiumStatus(ctx, userID)

	require.NoError(t, err)
	assert.Equal(t, userID, status.UserID)
	assert.Equal(t, models.TierFree, status.Tier)
	assert.Equal(t, models.SubStatusActive, status.Status)
	assert.Equal(t, 0, status.BoostsUsed)
	assert.Equal(t, 0, status.BoostsTotal)
	assert.Equal(t, 0, status.BoostsAvailable)
	repo.AssertExpectations(t)
}

func TestGetUserPremiumStatus_WithSubscription(t *testing.T) {
	svc, repo := newTestPremiumService()
	ctx := context.Background()
	userID := uuid.New()

	now := time.Now()
	sub := &models.Subscription{
		ID:          uuid.New(),
		UserID:      userID,
		Tier:        models.TierBasic,
		Status:      models.SubStatusActive,
		BoostsUsed:  1,
		BoostsTotal: 2,
		NextBilling: &now,
	}

	repo.On("GetSubscription", ctx, userID).Return(sub, nil)

	status, err := svc.GetUserPremiumStatus(ctx, userID)

	require.NoError(t, err)
	assert.Equal(t, userID, status.UserID)
	assert.Equal(t, models.TierBasic, status.Tier)
	assert.Equal(t, models.SubStatusActive, status.Status)
	assert.Equal(t, 1, status.BoostsUsed)
	assert.Equal(t, 2, status.BoostsTotal)
	assert.Equal(t, 1, status.BoostsAvailable)
	assert.NotNil(t, status.Subscription)
	repo.AssertExpectations(t)
}

func TestCreateSubscription_BasicTier(t *testing.T) {
	svc, repo := newTestPremiumService()
	ctx := context.Background()
	userID := uuid.New()

	repo.On("GetSubscription", ctx, userID).Return(nil, nil)
	repo.On("CreateSubscription", ctx, mock.AnythingOfType("*models.Subscription")).Return(nil)
	repo.On("UpdateUserPremiumTier", ctx, userID, models.TierBasic).Return(nil)

	sub, err := svc.CreateSubscription(ctx, userID, models.TierBasic)

	require.NoError(t, err)
	assert.Equal(t, models.TierBasic, sub.Tier)
	assert.Equal(t, models.SubStatusActive, sub.Status)
	assert.Equal(t, 2, sub.BoostsTotal)
	assert.Equal(t, 0, sub.BoostsUsed)
	repo.AssertExpectations(t)
}

func TestCreateSubscription_PremiumTier(t *testing.T) {
	svc, repo := newTestPremiumService()
	ctx := context.Background()
	userID := uuid.New()

	repo.On("GetSubscription", ctx, userID).Return(nil, nil)
	repo.On("CreateSubscription", ctx, mock.AnythingOfType("*models.Subscription")).Return(nil)
	repo.On("UpdateUserPremiumTier", ctx, userID, models.TierPremium).Return(nil)

	sub, err := svc.CreateSubscription(ctx, userID, models.TierPremium)

	require.NoError(t, err)
	assert.Equal(t, models.TierPremium, sub.Tier)
	assert.Equal(t, models.SubStatusActive, sub.Status)
	assert.Equal(t, 2, sub.BoostsTotal)
	repo.AssertExpectations(t)
}

func TestCreateSubscription_FreeTier(t *testing.T) {
	svc, _ := newTestPremiumService()
	ctx := context.Background()
	userID := uuid.New()

	sub, err := svc.CreateSubscription(ctx, userID, models.TierFree)

	assert.Error(t, err)
	assert.Equal(t, ErrInvalidTier, err)
	assert.Nil(t, sub)
}

func TestCreateSubscription_UpdateExisting(t *testing.T) {
	svc, repo := newTestPremiumService()
	ctx := context.Background()
	userID := uuid.New()

	existing := &models.Subscription{
		ID:          uuid.New(),
		UserID:      userID,
		Tier:        models.TierBasic,
		Status:      models.SubStatusActive,
		BoostsUsed:  1,
		BoostsTotal: 2,
	}

	repo.On("GetSubscription", ctx, userID).Return(existing, nil)
	repo.On("UpdateSubscription", ctx, mock.AnythingOfType("*models.Subscription")).Return(nil)

	sub, err := svc.CreateSubscription(ctx, userID, models.TierPremium)

	require.NoError(t, err)
	assert.Equal(t, models.TierPremium, sub.Tier)
	assert.Equal(t, 2, sub.BoostsTotal)
	repo.AssertExpectations(t)
}

func TestCancelSubscription(t *testing.T) {
	svc, repo := newTestPremiumService()
	ctx := context.Background()
	userID := uuid.New()

	sub := &models.Subscription{
		ID:        uuid.New(),
		UserID:    userID,
		Tier:      models.TierBasic,
		Status:    models.SubStatusActive,
	}

	repo.On("GetSubscription", ctx, userID).Return(sub, nil)
	repo.On("UpdateSubscription", ctx, mock.AnythingOfType("*models.Subscription")).Return(nil)
	repo.On("UpdateUserPremiumTier", ctx, userID, models.TierFree).Return(nil)

	err := svc.CancelSubscription(ctx, userID)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestCancelSubscription_NoSubscription(t *testing.T) {
	svc, repo := newTestPremiumService()
	ctx := context.Background()
	userID := uuid.New()

	repo.On("GetSubscription", ctx, userID).Return(nil, nil)

	err := svc.CancelSubscription(ctx, userID)

	assert.Error(t, err)
	assert.Equal(t, ErrNoSubscription, err)
	repo.AssertExpectations(t)
}

func TestBoostServer_Success(t *testing.T) {
	svc, repo := newTestPremiumService()
	ctx := context.Background()
	userID := uuid.New()
	serverID := uuid.New()

	sub := &models.Subscription{
		ID:          uuid.New(),
		UserID:      userID,
		Tier:        models.TierBasic,
		Status:      models.SubStatusActive,
		BoostsUsed:  0,
		BoostsTotal: 2,
	}

	repo.On("GetSubscription", ctx, userID).Return(sub, nil)
	repo.On("GetServerBoost", ctx, serverID, userID).Return(nil, nil)
	repo.On("CreateServerBoost", ctx, mock.AnythingOfType("*models.ServerBoost")).Return(nil)
	repo.On("GetSubscription", ctx, userID).Return(sub, nil)
	repo.On("UpdateSubscription", ctx, mock.AnythingOfType("*models.Subscription")).Return(nil)
	repo.On("GetServerBoostCount", ctx, serverID).Return(1, nil)
	repo.On("UpdateServerBoostLevel", ctx, serverID, 1).Return(nil)

	err := svc.BoostServer(ctx, userID, serverID)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestBoostServer_NoSubscription(t *testing.T) {
	svc, repo := newTestPremiumService()
	ctx := context.Background()
	userID := uuid.New()
	serverID := uuid.New()

	repo.On("GetSubscription", ctx, userID).Return(nil, nil)

	err := svc.BoostServer(ctx, userID, serverID)

	assert.Error(t, err)
	assert.Equal(t, ErrNoSubscription, err)
	repo.AssertExpectations(t)
}

func TestBoostServer_LimitReached(t *testing.T) {
	svc, repo := newTestPremiumService()
	ctx := context.Background()
	userID := uuid.New()
	serverID := uuid.New()

	sub := &models.Subscription{
		ID:          uuid.New(),
		UserID:      userID,
		Tier:        models.TierBasic,
		Status:      models.SubStatusActive,
		BoostsUsed:  2,
		BoostsTotal: 2,
	}

	repo.On("GetSubscription", ctx, userID).Return(sub, nil)

	err := svc.BoostServer(ctx, userID, serverID)

	assert.Error(t, err)
	assert.Equal(t, ErrBoostLimitReached, err)
	repo.AssertExpectations(t)
}

func TestBoostServer_AlreadyBoosted(t *testing.T) {
	svc, repo := newTestPremiumService()
	ctx := context.Background()
	userID := uuid.New()
	serverID := uuid.New()

	sub := &models.Subscription{
		ID:          uuid.New(),
		UserID:      userID,
		Tier:        models.TierBasic,
		Status:      models.SubStatusActive,
		BoostsUsed:  0,
		BoostsTotal: 2,
	}

	existingBoost := &models.ServerBoost{
		ID:       uuid.New(),
		ServerID: serverID,
		UserID:   userID,
		Active:   true,
	}

	repo.On("GetSubscription", ctx, userID).Return(sub, nil)
	repo.On("GetServerBoost", ctx, serverID, userID).Return(existingBoost, nil)

	err := svc.BoostServer(ctx, userID, serverID)

	assert.Error(t, err)
	assert.Equal(t, ErrAlreadyBoosted, err)
	repo.AssertExpectations(t)
}

func TestUnboostServer_Success(t *testing.T) {
	svc, repo := newTestPremiumService()
	ctx := context.Background()
	userID := uuid.New()
	serverID := uuid.New()

	boost := &models.ServerBoost{
		ID:       uuid.New(),
		ServerID: serverID,
		UserID:   userID,
		Active:   true,
	}

	sub := &models.Subscription{
		ID:          uuid.New(),
		UserID:      userID,
		Tier:        models.TierBasic,
		Status:      models.SubStatusActive,
		BoostsUsed:  1,
		BoostsTotal: 2,
	}

	repo.On("GetServerBoost", ctx, serverID, userID).Return(boost, nil)
	repo.On("DeactivateServerBoost", ctx, serverID, userID).Return(nil)
	repo.On("GetSubscription", ctx, userID).Return(sub, nil)
	repo.On("UpdateSubscription", ctx, mock.AnythingOfType("*models.Subscription")).Return(nil)
	repo.On("GetServerBoostCount", ctx, serverID).Return(0, nil)
	repo.On("UpdateServerBoostLevel", ctx, serverID, 0).Return(nil)

	err := svc.UnboostServer(ctx, userID, serverID)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestUnboostServer_NotBoosted(t *testing.T) {
	svc, repo := newTestPremiumService()
	ctx := context.Background()
	userID := uuid.New()
	serverID := uuid.New()

	repo.On("GetServerBoost", ctx, serverID, userID).Return(nil, nil)

	err := svc.UnboostServer(ctx, userID, serverID)

	assert.Error(t, err)
	assert.Equal(t, ErrNotBoosted, err)
	repo.AssertExpectations(t)
}

func TestCheckFeatureAccess_FreeTier(t *testing.T) {
	svc, repo := newTestPremiumService()
	ctx := context.Background()
	userID := uuid.New()

	sub := &models.Subscription{
		ID:          uuid.New(),
		UserID:      userID,
		Tier:        models.TierFree,
		Status:      models.SubStatusActive,
		BoostsUsed:  0,
		BoostsTotal: 0,
	}

	repo.On("GetSubscription", ctx, userID).Return(sub, nil)

	hasAccess, err := svc.CheckFeatureAccess(ctx, userID, "cross_server_emojis")

	require.NoError(t, err)
	assert.False(t, hasAccess)
	repo.AssertExpectations(t)
}

func TestCheckFeatureAccess_PremiumTier(t *testing.T) {
	svc, repo := newTestPremiumService()
	ctx := context.Background()
	userID := uuid.New()

	sub := &models.Subscription{
		ID:          uuid.New(),
		UserID:      userID,
		Tier:        models.TierPremium,
		Status:      models.SubStatusActive,
		BoostsUsed:  0,
		BoostsTotal: 2,
	}

	repo.On("GetSubscription", ctx, userID).Return(sub, nil)

	hasAccess, err := svc.CheckFeatureAccess(ctx, userID, "cross_server_emojis")

	require.NoError(t, err)
	assert.True(t, hasAccess)
	repo.AssertExpectations(t)
}

func TestGetServerBoostLevel(t *testing.T) {
	svc, repo := newTestPremiumService()
	ctx := context.Background()
	serverID := uuid.New()

	perks := &models.ServerPerks{
		Level:           2,
		BoostCount:      15,
		BoostsRequired:  30,
		EmojiLimit:      150,
		FileUploadLimit: 100 * 1024 * 1024,
		VoiceBitrate:    256000,
		HasBanner:       true,
	}

	repo.On("GetServerBoostLevel", ctx, serverID).Return(perks, nil)

	result, err := svc.GetServerBoostLevel(ctx, serverID)

	require.NoError(t, err)
	assert.Equal(t, 2, result.Level)
	assert.Equal(t, 15, result.BoostCount)
	assert.Equal(t, 150, result.EmojiLimit)
	repo.AssertExpectations(t)
}

func TestCalculateServerPerks(t *testing.T) {
	svc, repo := newTestPremiumService()
	ctx := context.Background()
	serverID := uuid.New()

	repo.On("GetServerBoostCount", ctx, serverID).Return(15, nil)

	perks, err := svc.CalculateServerPerks(ctx, serverID)

	require.NoError(t, err)
	assert.Equal(t, 2, perks.Level)
	assert.Equal(t, 15, perks.BoostCount)
	assert.Equal(t, 30, perks.BoostsRequired)
	assert.Equal(t, 150, perks.EmojiLimit)
	repo.AssertExpectations(t)
}

func TestGetUserBoostsAvailable(t *testing.T) {
	svc, repo := newTestPremiumService()
	ctx := context.Background()
	userID := uuid.New()

	sub := &models.Subscription{
		ID:          uuid.New(),
		UserID:      userID,
		Tier:        models.TierBasic,
		Status:      models.SubStatusActive,
		BoostsUsed:  1,
		BoostsTotal: 2,
	}

	repo.On("GetSubscription", ctx, userID).Return(sub, nil)

	available, err := svc.GetUserBoostsAvailable(ctx, userID)

	require.NoError(t, err)
	assert.Equal(t, 1, available)
	repo.AssertExpectations(t)
}

// Test model helper functions

func TestCalculateServerLevel(t *testing.T) {
	tests := []struct {
		boostCount int
		expected   int
	}{
		{0, 0},
		{1, 0},
		{2, 1},
		{14, 1},
		{15, 2},
		{29, 2},
		{30, 3},
		{100, 3},
	}

	for _, tt := range tests {
		level := models.CalculateServerLevel(tt.boostCount)
		assert.Equal(t, tt.expected, level, "boostCount=%d", tt.boostCount)
	}
}

func TestGetServerPerks(t *testing.T) {
	// Level 0 - no boosts
	perks := models.GetServerPerks(0)
	assert.Equal(t, 0, perks.Level)
	assert.Equal(t, 50, perks.EmojiLimit)
	assert.Equal(t, int64(8*1024*1024), perks.FileUploadLimit)

	// Level 1 - 2 boosts
	perks = models.GetServerPerks(2)
	assert.Equal(t, 1, perks.Level)
	assert.Equal(t, 100, perks.EmojiLimit)
	assert.True(t, perks.HasVanityURL)

	// Level 2 - 15 boosts
	perks = models.GetServerPerks(15)
	assert.Equal(t, 2, perks.Level)
	assert.Equal(t, 150, perks.EmojiLimit)
	assert.True(t, perks.HasBanner)

	// Level 3 - 30 boosts
	perks = models.GetServerPerks(30)
	assert.Equal(t, 3, perks.Level)
	assert.Equal(t, 250, perks.EmojiLimit)
	assert.True(t, perks.HasSplashScreen)
}

func TestGetPremiumFeatures(t *testing.T) {
	// Free tier
	features := models.GetPremiumFeatures(models.TierFree)
	assert.Equal(t, float64(0), features.MonthlyPrice)
	assert.Equal(t, 0, features.ServerBoosts)
	assert.Equal(t, int64(8*1024*1024), features.FileUploadSize)
	assert.False(t, features.CrossServerEmojis)
	assert.False(t, features.PremiumBadge)
	assert.False(t, features.MessageEditHistory)
	assert.False(t, features.PremiumStickers)

	// Basic tier
	features = models.GetPremiumFeatures(models.TierBasic)
	assert.Equal(t, 2.99, features.MonthlyPrice)
	assert.Equal(t, 2, features.ServerBoosts)
	assert.Equal(t, int64(50*1024*1024), features.FileUploadSize)
	assert.True(t, features.CrossServerEmojis)
	assert.True(t, features.HighQualityVideo)
	assert.True(t, features.CustomDiscriminator)
	assert.True(t, features.PrioritySupport)
	assert.False(t, features.PremiumBadge)
	assert.False(t, features.NoAds)
	assert.False(t, features.MessageEditHistory)

	// Premium tier
	features = models.GetPremiumFeatures(models.TierPremium)
	assert.Equal(t, 9.99, features.MonthlyPrice)
	assert.Equal(t, 2, features.ServerBoosts)
	assert.Equal(t, int64(100*1024*1024), features.FileUploadSize)
	assert.True(t, features.CrossServerEmojis)
	assert.True(t, features.HighQualityVideo)
	assert.True(t, features.CustomDiscriminator)
	assert.True(t, features.PremiumBadge)
	assert.True(t, features.NoAds)
	assert.True(t, features.MessageEditHistory)
	assert.True(t, features.PremiumStickers)
	assert.True(t, features.CustomStatusEmoji)
	assert.True(t, features.HDScreenShare)
}

func TestSubscriptionTierFromString(t *testing.T) {
	assert.Equal(t, models.TierFree, models.SubscriptionTierFromString("free"))
	assert.Equal(t, models.TierBasic, models.SubscriptionTierFromString("basic"))
	assert.Equal(t, models.TierPremium, models.SubscriptionTierFromString("premium"))
	assert.Equal(t, models.TierFree, models.SubscriptionTierFromString("invalid"))
	assert.Equal(t, models.TierFree, models.SubscriptionTierFromString(""))
}

func TestUpdateSubscriptionTier_Upgrade(t *testing.T) {
	svc, repo := newTestPremiumService()
	ctx := context.Background()
	userID := uuid.New()

	sub := &models.Subscription{
		ID:          uuid.New(),
		UserID:      userID,
		Tier:        models.TierBasic,
		Status:      models.SubStatusActive,
		BoostsUsed:  0,
		BoostsTotal: 2,
	}

	repo.On("GetSubscription", ctx, userID).Return(sub, nil)
	repo.On("UpdateSubscription", ctx, mock.AnythingOfType("*models.Subscription")).Return(nil)
	repo.On("UpdateUserPremiumTier", ctx, userID, models.TierPremium).Return(nil)

	err := svc.UpdateSubscriptionTier(ctx, userID, models.TierPremium)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestUpdateSubscriptionTier_Downgrade(t *testing.T) {
	svc, repo := newTestPremiumService()
	ctx := context.Background()
	userID := uuid.New()

	sub := &models.Subscription{
		ID:          uuid.New(),
		UserID:      userID,
		Tier:        models.TierPremium,
		Status:      models.SubStatusActive,
		BoostsUsed:  0,
		BoostsTotal: 2,
	}

	repo.On("GetSubscription", ctx, userID).Return(sub, nil)
	repo.On("UpdateSubscription", ctx, mock.AnythingOfType("*models.Subscription")).Return(nil)
	repo.On("UpdateUserPremiumTier", ctx, userID, models.TierBasic).Return(nil)

	err := svc.UpdateSubscriptionTier(ctx, userID, models.TierBasic)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestUpdateSubscriptionTier_NoSubscription(t *testing.T) {
	svc, repo := newTestPremiumService()
	ctx := context.Background()
	userID := uuid.New()

	repo.On("GetSubscription", ctx, userID).Return(nil, nil)

	err := svc.UpdateSubscriptionTier(ctx, userID, models.TierPremium)

	assert.Error(t, err)
	assert.Equal(t, ErrNoSubscription, err)
	repo.AssertExpectations(t)
}

func TestReactivateSubscription(t *testing.T) {
	svc, repo := newTestPremiumService()
	ctx := context.Background()
	userID := uuid.New()

	now := time.Now()
	sub := &models.Subscription{
		ID:         uuid.New(),
		UserID:     userID,
		Tier:       models.TierBasic,
		Status:     models.SubStatusCanceled,
		CanceledAt: &now,
	}

	repo.On("GetSubscription", ctx, userID).Return(sub, nil)
	repo.On("UpdateSubscription", ctx, mock.AnythingOfType("*models.Subscription")).Return(nil)
	repo.On("UpdateUserPremiumTier", ctx, userID, models.TierBasic).Return(nil)

	err := svc.ReactivateSubscription(ctx, userID)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestReactivateSubscription_NotCanceled(t *testing.T) {
	svc, repo := newTestPremiumService()
	ctx := context.Background()
	userID := uuid.New()

	sub := &models.Subscription{
		ID:     uuid.New(),
		UserID: userID,
		Tier:   models.TierBasic,
		Status: models.SubStatusActive,
	}

	repo.On("GetSubscription", ctx, userID).Return(sub, nil)

	err := svc.ReactivateSubscription(ctx, userID)

	assert.Error(t, err)
	repo.AssertExpectations(t)
}

func TestPremiumPricing(t *testing.T) {
	assert.Equal(t, 2.99, models.PremiumTierPricing[models.TierBasic])
	assert.Equal(t, 9.99, models.PremiumTierPricing[models.TierPremium])
	assert.Equal(t, float64(0), models.PremiumTierPricing[models.TierFree])
}
