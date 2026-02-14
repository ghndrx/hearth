package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mock Implementations ---

// MockMemberStore mocks the MemberStore interface
type MockMemberStore struct {
	mock.Mock
}

func (m *MockMemberStore) FetchByID(ctx context.Context, guildID, memberID string) (*GuildMember, error) {
	args := m.Called(ctx, guildID, memberID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*GuildMember), args.Error(1)
}

// MockEventBus mocks the EventBus interface
type MockEventBus struct {
	mock.Mock
}

func (m *MockEventBus) Publish(topic string, payload interface{}) error {
	args := m.Called(topic, payload)
	return args.Error(0)
}

// --- Unit Tests ---

func TestNewMemberService(t *testing.T) {
	store := &MockMemberStore{}
	bus := &MockEventBus{}

	service := NewMemberService(store, bus)

	// Assertions: Check that the state was initialized
	assert.NotNil(t, service)
	assert.NotNil(t, service.memberStore)
	assert.NotNil(t, service.eventBus)

	// Test that nil store/wire works (using nil ptr checks)
	serviceWithStoreNil := NewMemberService(nil, bus)
	assert.Nil(t, serviceWithStoreNil.memberStore)
}

func TestMemberService_GetMember_HitCache(t *testing.T) {
	store := &MockMemberStore{}
	bus := &MockEventBus()
	service := NewMemberService(store, bus)

	guildID := "guild123"
	memberID := "user456"
	cachedMember := &GuildMember{ID: memberID, Name: "CachedUser"}

	// 1. Populate cache via service
	service.membersMap.Store(cacheKey(guildID, memberID), cachedMember)

	// 2. Call GetMember (Should not hit mock store)
	member, err := service.GetMember(context.Background(), guildID, memberID)

	// 3. Assertions
	assert.NoError(t, err)
	assert.Equal(t, "CachedUser", member.Name)
	store.AssertNotCalled(t, "FetchByID")
}

func TestMemberService_GetMember_MissCache_HitStore(t *testing.T) {
	store := &MockMemberStore{}
	bus := &MockEventBus()
	service := NewMemberService(store, bus)

	guildID := "guild123"
	memberID := "user456"
	dbMember := &GuildMember{ID: memberID, Name: "DBUser"}

	// 1. Setup Mock Expectation
	store.On("FetchByID", mock.Anything, guildID, memberID).Return(dbMember, nil)

	// 2. Call GetMember
	member, err := service.GetMember(context.Background(), guildID, memberID)

	// 3. Assertions
	assert.NoError(t, err)
	assert.Equal(t, "DBUser", member.Name)
	assert.Equal(t, "DBUser", member.Name)
	
	// Verify mock was called
	store.AssertExpectations(t)
}

func TestMemberService_GetMember_MissCache_NotFound(t *testing.T) {
	store := &MockMemberStore{}
	bus := &MockEventBus()
	service := NewMemberService(store, bus)

	guildID := "guild123"
	memberID := "fakeUser"

	// 1. Setup Mock Expectation for NotFound
	store.On("FetchByID", mock.Anything, guildID, memberID).Return(nil, ErrMemberNotFound)

	// 2. Call GetMember
	member, err := service.GetMember(context.Background(), guildID, memberID)

	// 3. Assertions
	assert.Error(t, err)
	assert.Equal(t, ErrMemberNotFound, err)
	assert.Nil(t, member)
	store.AssertExpectations(t)
}

func TestMemberService_UpdateMember(t *testing.T) {
	store := &MockMemberStore{}
	bus := &MockEventBus()
	service := NewMemberService(store, bus)

	guildID := "guild123"
	memberID := "user456"

	// 1. Setup Mock
	store.On("FetchByID", mock.Anything, guildID, memberID).Return(&GuildMember{ID: memberID}, nil)
	bus.On("Publish", "member.updated", guildID).Return(nil)

	// 2. Execute
	err := service.UpdateMember(context.Background(), guildID, memberID, "NewName")

	// 3. Assertions
	assert.NoError(t, err)
	bus.AssertExpectations(t)
}