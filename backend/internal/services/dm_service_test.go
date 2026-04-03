package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"hearth/internal/models"
)

// Mock channel repository for DM service tests
type mockDMChannelRepo struct {
	channels    map[uuid.UUID]*models.Channel
	getByIDFunc func(ctx context.Context, id uuid.UUID) (*models.Channel, error)
	addRecFunc  func(ctx context.Context, channelID, userID uuid.UUID) error
	remRecFunc  func(ctx context.Context, channelID, userID uuid.UUID) error
}

func (m *mockDMChannelRepo) Create(ctx context.Context, channel *models.Channel) error {
	if m.channels == nil {
		m.channels = make(map[uuid.UUID]*models.Channel)
	}
	m.channels[channel.ID] = channel
	return nil
}

func (m *mockDMChannelRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	if ch, ok := m.channels[id]; ok {
		return ch, nil
	}
	return nil, nil
}

func (m *mockDMChannelRepo) Update(ctx context.Context, channel *models.Channel) error {
	m.channels[channel.ID] = channel
	return nil
}

func (m *mockDMChannelRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.channels, id)
	return nil
}

func (m *mockDMChannelRepo) GetByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.Channel, error) {
	return nil, nil
}

func (m *mockDMChannelRepo) GetDMChannel(ctx context.Context, user1ID, user2ID uuid.UUID) (*models.Channel, error) {
	return nil, nil
}

func (m *mockDMChannelRepo) GetUserDMs(ctx context.Context, userID uuid.UUID) ([]*models.Channel, error) {
	return nil, nil
}

func (m *mockDMChannelRepo) UpdateLastMessage(ctx context.Context, channelID, messageID uuid.UUID, at time.Time) error {
	return nil
}

func (m *mockDMChannelRepo) AddRecipient(ctx context.Context, channelID, userID uuid.UUID) error {
	if m.addRecFunc != nil {
		return m.addRecFunc(ctx, channelID, userID)
	}
	ch, ok := m.channels[channelID]
	if !ok {
		return ErrChannelNotFound
	}
	ch.Recipients = append(ch.Recipients, userID)
	return nil
}

func (m *mockDMChannelRepo) RemoveRecipient(ctx context.Context, channelID, userID uuid.UUID) error {
	if m.remRecFunc != nil {
		return m.remRecFunc(ctx, channelID, userID)
	}
	ch, ok := m.channels[channelID]
	if !ok {
		return ErrChannelNotFound
	}
	for i, r := range ch.Recipients {
		if r == userID {
			ch.Recipients = append(ch.Recipients[:i], ch.Recipients[i+1:]...)
			break
		}
	}
	return nil
}

func (m *mockDMChannelRepo) CountRecipients(ctx context.Context, channelID uuid.UUID) (int, error) {
	ch, ok := m.channels[channelID]
	if !ok {
		return 0, ErrChannelNotFound
	}
	return len(ch.Recipients), nil
}

func (m *mockDMChannelRepo) BulkUpdatePositions(ctx context.Context, entries []models.ReorderChannelEntry) error {
	return nil
}

func (m *mockDMChannelRepo) UpdateForumConfig(ctx context.Context, channelID uuid.UUID, configJSON []byte) error {
	return nil
}

func (m *mockDMChannelRepo) GetPermissionOverrides(ctx context.Context, channelID uuid.UUID) ([]models.PermissionOverride, error) {
	return nil, nil
}

func (m *mockDMChannelRepo) UpsertPermissionOverride(ctx context.Context, override *models.PermissionOverride) error {
	return nil
}

func (m *mockDMChannelRepo) DeletePermissionOverride(ctx context.Context, channelID, targetID uuid.UUID, targetType string) error {
	return nil
}

// Mock event bus for DM service tests
type mockDMEventBus struct {
	published []string
}

func (m *mockDMEventBus) Publish(event string, data interface{}) {
	m.published = append(m.published, event)
}

func (m *mockDMEventBus) Subscribe(event string, handler func(data interface{}))   {}
func (m *mockDMEventBus) Unsubscribe(event string, handler func(data interface{})) {}

// Mock cache for DM service tests
type mockDMCache struct{}

func (m *mockDMCache) GetUser(ctx context.Context, id uuid.UUID) (*models.User, error) {
	return nil, nil
}
func (m *mockDMCache) SetUser(ctx context.Context, user *models.User, ttl time.Duration) error {
	return nil
}
func (m *mockDMCache) DeleteUser(ctx context.Context, id uuid.UUID) error {
	return nil
}
func (m *mockDMCache) GetServer(ctx context.Context, id uuid.UUID) (*models.Server, error) {
	return nil, nil
}
func (m *mockDMCache) SetServer(ctx context.Context, server *models.Server, ttl time.Duration) error {
	return nil
}
func (m *mockDMCache) DeleteServer(ctx context.Context, id uuid.UUID) error {
	return nil
}
func (m *mockDMCache) GetChannel(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
	return nil, nil
}
func (m *mockDMCache) SetChannel(ctx context.Context, channel *models.Channel, ttl time.Duration) error {
	return nil
}
func (m *mockDMCache) DeleteChannel(ctx context.Context, id uuid.UUID) error {
	return nil
}
func (m *mockDMCache) Get(ctx context.Context, key string) ([]byte, error) {
	return nil, nil
}
func (m *mockDMCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return nil
}
func (m *mockDMCache) Delete(ctx context.Context, key string) error {
	return nil
}

// Helper to create a test group DM channel
func createTestGroupDM(t *testing.T, repo *mockDMChannelRepo, ownerID uuid.UUID, recipientIDs []uuid.UUID) *models.Channel {
	ownerPtr := ownerID
	channel := &models.Channel{
		ID:         uuid.New(),
		Type:       models.ChannelTypeGroupDM,
		Name:       "Test Group DM",
		OwnerID:    &ownerPtr,
		Recipients: append([]uuid.UUID{ownerID}, recipientIDs...),
	}
	repo.channels[channel.ID] = channel
	return channel
}

// Test AddUserToGroupDM - Success
func TestAddUserToGroupDM_Success(t *testing.T) {
	repo := &mockDMChannelRepo{channels: make(map[uuid.UUID]*models.Channel)}
	eventBus := &mockDMEventBus{}
	cache := &mockDMCache{}
	svc := NewDMService(repo, eventBus, cache)

	ownerID := uuid.New()
	existingUserID := uuid.New()
	newUserID := uuid.New()
	channel := createTestGroupDM(t, repo, ownerID, []uuid.UUID{existingUserID})

	// Set up getByID to return channel
	repo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
		return repo.channels[id], nil
	}

	result, err := svc.AddUserToGroupDM(context.Background(), channel.ID, ownerID, newUserID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result.Recipients, newUserID)
	assert.Contains(t, eventBus.published, "dm.recipient_add")
}

// Test AddUserToGroupDM - Not owner
func TestAddUserToGroupDM_NotOwner(t *testing.T) {
	repo := &mockDMChannelRepo{channels: make(map[uuid.UUID]*models.Channel)}
	eventBus := &mockDMEventBus{}
	cache := &mockDMCache{}
	svc := NewDMService(repo, eventBus, cache)

	ownerID := uuid.New()
	nonOwnerID := uuid.New()
	newUserID := uuid.New()
	channel := createTestGroupDM(t, repo, ownerID, []uuid.UUID{})

	repo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
		return repo.channels[id], nil
	}

	result, err := svc.AddUserToGroupDM(context.Background(), channel.ID, nonOwnerID, newUserID)
	assert.Error(t, err)
	assert.Equal(t, ErrNotGroupDMOwner, err)
	assert.Nil(t, result)
}

// Test AddUserToGroupDM - Already recipient
func TestAddUserToGroupDM_AlreadyRecipient(t *testing.T) {
	repo := &mockDMChannelRepo{channels: make(map[uuid.UUID]*models.Channel)}
	eventBus := &mockDMEventBus{}
	cache := &mockDMCache{}
	svc := NewDMService(repo, eventBus, cache)

	ownerID := uuid.New()
	existingUserID := uuid.New()
	channel := createTestGroupDM(t, repo, ownerID, []uuid.UUID{existingUserID})

	repo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
		return repo.channels[id], nil
	}

	result, err := svc.AddUserToGroupDM(context.Background(), channel.ID, ownerID, existingUserID)
	assert.Error(t, err)
	assert.Equal(t, ErrAlreadyDMRecipient, err)
	assert.Nil(t, result)
}

// Test AddUserToGroupDM - Channel full
func TestAddUserToGroupDM_ChannelFull(t *testing.T) {
	repo := &mockDMChannelRepo{channels: make(map[uuid.UUID]*models.Channel)}
	eventBus := &mockDMEventBus{}
	cache := &mockDMCache{}
	svc := NewDMService(repo, eventBus, cache)

	ownerID := uuid.New()
	// Create a channel with MaxGroupDMUsers (50) recipients
	recipientIDs := make([]uuid.UUID, models.MaxGroupDMUsers-1)
	for i := range recipientIDs {
		recipientIDs[i] = uuid.New()
	}
	channel := createTestGroupDM(t, repo, ownerID, recipientIDs)

	repo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
		return repo.channels[id], nil
	}

	newUserID := uuid.New()
	result, err := svc.AddUserToGroupDM(context.Background(), channel.ID, ownerID, newUserID)
	assert.Error(t, err)
	assert.Equal(t, ErrGroupDMFull, err)
	assert.Nil(t, result)
}

// Test AddUserToGroupDM - Channel not found
func TestAddUserToGroupDM_ChannelNotFound(t *testing.T) {
	repo := &mockDMChannelRepo{channels: make(map[uuid.UUID]*models.Channel)}
	eventBus := &mockDMEventBus{}
	cache := &mockDMCache{}
	svc := NewDMService(repo, eventBus, cache)

	repo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
		return nil, nil // Channel not found
	}

	result, err := svc.AddUserToGroupDM(context.Background(), uuid.New(), uuid.New(), uuid.New())
	assert.Error(t, err)
	assert.Equal(t, ErrChannelNotFound, err)
	assert.Nil(t, result)
}

// Test AddUserToGroupDM - Not a group DM
func TestAddUserToGroupDM_NotGroupDM(t *testing.T) {
	repo := &mockDMChannelRepo{channels: make(map[uuid.UUID]*models.Channel)}
	eventBus := &mockDMEventBus{}
	cache := &mockDMCache{}
	svc := NewDMService(repo, eventBus, cache)

	ownerID := uuid.New()
	channel := &models.Channel{
		ID:         uuid.New(),
		Type:       models.ChannelTypeDM, // Not a group DM
		OwnerID:    &ownerID,
		Recipients: []uuid.UUID{ownerID},
	}
	repo.channels[channel.ID] = channel

	repo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
		return repo.channels[id], nil
	}

	result, err := svc.AddUserToGroupDM(context.Background(), channel.ID, ownerID, uuid.New())
	assert.Error(t, err)
	assert.Equal(t, ErrNotGroupDM, err)
	assert.Nil(t, result)
}

// Test RemoveUserFromGroupDM - Success
func TestRemoveUserFromGroupDM_Success(t *testing.T) {
	repo := &mockDMChannelRepo{channels: make(map[uuid.UUID]*models.Channel)}
	eventBus := &mockDMEventBus{}
	cache := &mockDMCache{}
	svc := NewDMService(repo, eventBus, cache)

	ownerID := uuid.New()
	userToRemove := uuid.New()
	channel := createTestGroupDM(t, repo, ownerID, []uuid.UUID{userToRemove})

	repo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
		return repo.channels[id], nil
	}

	err := svc.RemoveUserFromGroupDM(context.Background(), channel.ID, ownerID, userToRemove)
	assert.NoError(t, err)
	assert.NotContains(t, repo.channels[channel.ID].Recipients, userToRemove)
	assert.Contains(t, eventBus.published, "dm.recipient_remove")
}

// Test RemoveUserFromGroupDM - Owner removes self (allowed)
func TestRemoveUserFromGroupDM_OwnerRemovesSelf(t *testing.T) {
	repo := &mockDMChannelRepo{channels: make(map[uuid.UUID]*models.Channel)}
	eventBus := &mockDMEventBus{}
	cache := &mockDMCache{}
	svc := NewDMService(repo, eventBus, cache)

	ownerID := uuid.New()
	channel := createTestGroupDM(t, repo, ownerID, []uuid.UUID{})

	repo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
		return repo.channels[id], nil
	}

	err := svc.RemoveUserFromGroupDM(context.Background(), channel.ID, ownerID, ownerID)
	assert.NoError(t, err)
	assert.NotContains(t, repo.channels[channel.ID].Recipients, ownerID)
}

// Test RemoveUserFromGroupDM - Not recipient
func TestRemoveUserFromGroupDM_NotRecipient(t *testing.T) {
	repo := &mockDMChannelRepo{channels: make(map[uuid.UUID]*models.Channel)}
	eventBus := &mockDMEventBus{}
	cache := &mockDMCache{}
	svc := NewDMService(repo, eventBus, cache)

	ownerID := uuid.New()
	nonRecipient := uuid.New()
	channel := createTestGroupDM(t, repo, ownerID, []uuid.UUID{})

	repo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
		return repo.channels[id], nil
	}

	err := svc.RemoveUserFromGroupDM(context.Background(), channel.ID, ownerID, nonRecipient)
	assert.Error(t, err)
	assert.Equal(t, ErrNotDMRecipient, err)
}

// Test RemoveUserFromGroupDM - Non-owner tries to remove another user
func TestRemoveUserFromGroupDM_NonOwnerRemovesOther(t *testing.T) {
	repo := &mockDMChannelRepo{channels: make(map[uuid.UUID]*models.Channel)}
	eventBus := &mockDMEventBus{}
	cache := &mockDMCache{}
	svc := NewDMService(repo, eventBus, cache)

	ownerID := uuid.New()
	nonOwnerID := uuid.New()
	userToRemove := uuid.New()
	channel := createTestGroupDM(t, repo, ownerID, []uuid.UUID{nonOwnerID, userToRemove})

	repo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
		return repo.channels[id], nil
	}

	err := svc.RemoveUserFromGroupDM(context.Background(), channel.ID, nonOwnerID, userToRemove)
	assert.Error(t, err)
	assert.Equal(t, ErrNotGroupDMOwner, err)
}

// Test LeaveDM - Success
func TestLeaveDM_Success(t *testing.T) {
	repo := &mockDMChannelRepo{channels: make(map[uuid.UUID]*models.Channel)}
	eventBus := &mockDMEventBus{}
	cache := &mockDMCache{}
	svc := NewDMService(repo, eventBus, cache)

	ownerID := uuid.New()
	userID := uuid.New()
	channel := createTestGroupDM(t, repo, ownerID, []uuid.UUID{userID})

	repo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
		return repo.channels[id], nil
	}

	err := svc.LeaveDM(context.Background(), channel.ID, userID)
	assert.NoError(t, err)
	assert.NotContains(t, repo.channels[channel.ID].Recipients, userID)
	assert.Contains(t, eventBus.published, "dm.recipient_remove")
}

// Test LeaveDM - Not a DM channel
func TestLeaveDM_NotDMChannel(t *testing.T) {
	repo := &mockDMChannelRepo{channels: make(map[uuid.UUID]*models.Channel)}
	eventBus := &mockDMEventBus{}
	cache := &mockDMCache{}
	svc := NewDMService(repo, eventBus, cache)

	channel := &models.Channel{
		ID:         uuid.New(),
		Type:       models.ChannelTypeText, // Not a DM
		Recipients: []uuid.UUID{},
	}
	repo.channels[channel.ID] = channel

	repo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
		return repo.channels[id], nil
	}

	err := svc.LeaveDM(context.Background(), channel.ID, uuid.New())
	assert.Error(t, err)
	assert.Equal(t, ErrNotDMChannel, err)
}

// Test LeaveDM - Not a recipient
func TestLeaveDM_NotRecipient(t *testing.T) {
	repo := &mockDMChannelRepo{channels: make(map[uuid.UUID]*models.Channel)}
	eventBus := &mockDMEventBus{}
	cache := &mockDMCache{}
	svc := NewDMService(repo, eventBus, cache)

	ownerID := uuid.New()
	nonParticipant := uuid.New()
	channel := createTestGroupDM(t, repo, ownerID, []uuid.UUID{})

	repo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
		return repo.channels[id], nil
	}

	err := svc.LeaveDM(context.Background(), channel.ID, nonParticipant)
	assert.Error(t, err)
	assert.Equal(t, ErrNotDMRecipient, err)
}

// Test TransferGroupDMOwnership - Success
func TestTransferGroupDMOwnership_Success(t *testing.T) {
	repo := &mockDMChannelRepo{channels: make(map[uuid.UUID]*models.Channel)}
	eventBus := &mockDMEventBus{}
	cache := &mockDMCache{}
	svc := NewDMService(repo, eventBus, cache)

	ownerID := uuid.New()
	newOwnerID := uuid.New()
	channel := createTestGroupDM(t, repo, ownerID, []uuid.UUID{newOwnerID})

	repo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
		return repo.channels[id], nil
	}

	result, err := svc.TransferGroupDMOwnership(context.Background(), channel.ID, ownerID, newOwnerID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, newOwnerID, *result.OwnerID)
	assert.Contains(t, eventBus.published, "dm.ownership_transfer")
}

// Test TransferGroupDMOwnership - Not owner
func TestTransferGroupDMOwnership_NotOwner(t *testing.T) {
	repo := &mockDMChannelRepo{channels: make(map[uuid.UUID]*models.Channel)}
	eventBus := &mockDMEventBus{}
	cache := &mockDMCache{}
	svc := NewDMService(repo, eventBus, cache)

	ownerID := uuid.New()
	nonOwnerID := uuid.New()
	newOwnerID := uuid.New()
	channel := createTestGroupDM(t, repo, ownerID, []uuid.UUID{newOwnerID})

	repo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
		return repo.channels[id], nil
	}

	result, err := svc.TransferGroupDMOwnership(context.Background(), channel.ID, nonOwnerID, newOwnerID)
	assert.Error(t, err)
	assert.Equal(t, ErrNotGroupDMOwner, err)
	assert.Nil(t, result)
}

// Test TransferGroupDMOwnership - Not a member
func TestTransferGroupDMOwnership_NotMember(t *testing.T) {
	repo := &mockDMChannelRepo{channels: make(map[uuid.UUID]*models.Channel)}
	eventBus := &mockDMEventBus{}
	cache := &mockDMCache{}
	svc := NewDMService(repo, eventBus, cache)

	ownerID := uuid.New()
	nonMemberID := uuid.New()
	channel := createTestGroupDM(t, repo, ownerID, []uuid.UUID{})

	repo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
		return repo.channels[id], nil
	}

	result, err := svc.TransferGroupDMOwnership(context.Background(), channel.ID, ownerID, nonMemberID)
	assert.Error(t, err)
	assert.Equal(t, ErrCannotTransferToNonMember, err)
	assert.Nil(t, result)
}

// Test TransferGroupDMOwnership - Not a group DM
func TestTransferGroupDMOwnership_NotGroupDM(t *testing.T) {
	repo := &mockDMChannelRepo{channels: make(map[uuid.UUID]*models.Channel)}
	eventBus := &mockDMEventBus{}
	cache := &mockDMCache{}
	svc := NewDMService(repo, eventBus, cache)

	ownerID := uuid.New()
	newOwnerID := uuid.New()
	channel := &models.Channel{
		ID:         uuid.New(),
		Type:       models.ChannelTypeDM, // Not a group DM
		OwnerID:    &ownerID,
		Recipients: []uuid.UUID{ownerID, newOwnerID},
	}
	repo.channels[channel.ID] = channel

	repo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
		return repo.channels[id], nil
	}

	result, err := svc.TransferGroupDMOwnership(context.Background(), channel.ID, ownerID, newOwnerID)
	assert.Error(t, err)
	assert.Equal(t, ErrNotGroupDM, err)
	assert.Nil(t, result)
}

// Test TransferGroupDMOwnership - Channel not found
func TestTransferGroupDMOwnership_ChannelNotFound(t *testing.T) {
	repo := &mockDMChannelRepo{channels: make(map[uuid.UUID]*models.Channel)}
	eventBus := &mockDMEventBus{}
	cache := &mockDMCache{}
	svc := NewDMService(repo, eventBus, cache)

	repo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
		return nil, nil // Channel not found
	}

	result, err := svc.TransferGroupDMOwnership(context.Background(), uuid.New(), uuid.New(), uuid.New())
	assert.Error(t, err)
	assert.Equal(t, ErrChannelNotFound, err)
	assert.Nil(t, result)
}
