package services

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"hearth/internal/models"
)

// MockEventBusForTyping is a mock for the EventBus interface
type MockEventBusForTyping struct {
	mock.Mock
	mu       sync.Mutex
	events   []interface{}
}

func NewMockEventBusForTyping() *MockEventBusForTyping {
	return &MockEventBusForTyping{
		events: make([]interface{}, 0),
	}
}

func (m *MockEventBusForTyping) Publish(event string, data interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, data)
	m.Called(event, data)
}

func (m *MockEventBusForTyping) Subscribe(event string, handler func(data interface{})) {
	m.Called(event, handler)
}

func (m *MockEventBusForTyping) Unsubscribe(event string, handler func(data interface{})) {
	m.Called(event, handler)
}

func (m *MockEventBusForTyping) GetEvents() []interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]interface{}{}, m.events...)
}

func TestTypingService_StartTyping(t *testing.T) {
	mockEventBus := NewMockEventBusForTyping()
	mockEventBus.On("Publish", "typing.start", mock.AnythingOfType("*models.TypingIndicator")).Return()

	svc := NewTypingService(mockEventBus)
	ctx := context.Background()

	channelID := uuid.New()
	userID := uuid.New()

	err := svc.StartTyping(ctx, channelID, userID)
	assert.NoError(t, err)

	// Verify typing was recorded
	indicators, err := svc.GetTypingUsers(ctx, channelID)
	assert.NoError(t, err)
	assert.Len(t, indicators, 1)
	assert.Equal(t, userID, indicators[0].UserID)
	assert.Equal(t, channelID, indicators[0].ChannelID)

	// Verify event was published
	mockEventBus.AssertCalled(t, "Publish", "typing.start", mock.AnythingOfType("*models.TypingIndicator"))
}

func TestTypingService_StartTyping_MultipleUsers(t *testing.T) {
	mockEventBus := NewMockEventBusForTyping()
	mockEventBus.On("Publish", "typing.start", mock.AnythingOfType("*models.TypingIndicator")).Return()

	svc := NewTypingService(mockEventBus)
	ctx := context.Background()

	channelID := uuid.New()
	user1 := uuid.New()
	user2 := uuid.New()
	user3 := uuid.New()

	_ = svc.StartTyping(ctx, channelID, user1)
	_ = svc.StartTyping(ctx, channelID, user2)
	_ = svc.StartTyping(ctx, channelID, user3)

	indicators, err := svc.GetTypingUsers(ctx, channelID)
	assert.NoError(t, err)
	assert.Len(t, indicators, 3)

	// Verify all users are present
	userIDs := make(map[uuid.UUID]bool)
	for _, ind := range indicators {
		userIDs[ind.UserID] = true
	}
	assert.True(t, userIDs[user1])
	assert.True(t, userIDs[user2])
	assert.True(t, userIDs[user3])
}

func TestTypingService_StopTyping(t *testing.T) {
	mockEventBus := NewMockEventBusForTyping()
	mockEventBus.On("Publish", "typing.start", mock.AnythingOfType("*models.TypingIndicator")).Return()

	svc := NewTypingService(mockEventBus)
	ctx := context.Background()

	channelID := uuid.New()
	userID := uuid.New()

	_ = svc.StartTyping(ctx, channelID, userID)
	err := svc.StopTyping(ctx, channelID, userID)
	assert.NoError(t, err)

	indicators, _ := svc.GetTypingUsers(ctx, channelID)
	assert.Len(t, indicators, 0)
}

func TestTypingService_StopTyping_NonExistent(t *testing.T) {
	svc := NewTypingService(nil)
	ctx := context.Background()

	channelID := uuid.New()
	userID := uuid.New()

	// Should not error when stopping non-existent typing
	err := svc.StopTyping(ctx, channelID, userID)
	assert.NoError(t, err)
}

func TestTypingService_GetTypingUsers_EmptyChannel(t *testing.T) {
	svc := NewTypingService(nil)
	ctx := context.Background()

	channelID := uuid.New()

	indicators, err := svc.GetTypingUsers(ctx, channelID)
	assert.NoError(t, err)
	assert.Len(t, indicators, 0)
}

func TestTypingService_IsTyping(t *testing.T) {
	mockEventBus := NewMockEventBusForTyping()
	mockEventBus.On("Publish", "typing.start", mock.AnythingOfType("*models.TypingIndicator")).Return()

	svc := NewTypingService(mockEventBus)
	ctx := context.Background()

	channelID := uuid.New()
	userID := uuid.New()
	otherUserID := uuid.New()

	_ = svc.StartTyping(ctx, channelID, userID)

	isTyping, err := svc.IsTyping(ctx, channelID, userID)
	assert.NoError(t, err)
	assert.True(t, isTyping)

	isTyping, err = svc.IsTyping(ctx, channelID, otherUserID)
	assert.NoError(t, err)
	assert.False(t, isTyping)
}

func TestTypingService_GetTypingUserIDs(t *testing.T) {
	mockEventBus := NewMockEventBusForTyping()
	mockEventBus.On("Publish", "typing.start", mock.AnythingOfType("*models.TypingIndicator")).Return()

	svc := NewTypingService(mockEventBus)
	ctx := context.Background()

	channelID := uuid.New()
	user1 := uuid.New()
	user2 := uuid.New()

	_ = svc.StartTyping(ctx, channelID, user1)
	_ = svc.StartTyping(ctx, channelID, user2)

	userIDs, err := svc.GetTypingUserIDs(ctx, channelID)
	assert.NoError(t, err)
	assert.Len(t, userIDs, 2)
	assert.Contains(t, userIDs, user1)
	assert.Contains(t, userIDs, user2)
}

func TestTypingService_ClearChannel(t *testing.T) {
	mockEventBus := NewMockEventBusForTyping()
	mockEventBus.On("Publish", "typing.start", mock.AnythingOfType("*models.TypingIndicator")).Return()

	svc := NewTypingService(mockEventBus)
	ctx := context.Background()

	channelID := uuid.New()
	_ = svc.StartTyping(ctx, channelID, uuid.New())
	_ = svc.StartTyping(ctx, channelID, uuid.New())
	_ = svc.StartTyping(ctx, channelID, uuid.New())

	err := svc.ClearChannel(ctx, channelID)
	assert.NoError(t, err)

	indicators, _ := svc.GetTypingUsers(ctx, channelID)
	assert.Len(t, indicators, 0)
}

func TestTypingService_MultipleChannels(t *testing.T) {
	mockEventBus := NewMockEventBusForTyping()
	mockEventBus.On("Publish", "typing.start", mock.AnythingOfType("*models.TypingIndicator")).Return()

	svc := NewTypingService(mockEventBus)
	ctx := context.Background()

	channel1 := uuid.New()
	channel2 := uuid.New()
	user1 := uuid.New()
	user2 := uuid.New()

	_ = svc.StartTyping(ctx, channel1, user1)
	_ = svc.StartTyping(ctx, channel2, user2)

	indicators1, _ := svc.GetTypingUsers(ctx, channel1)
	indicators2, _ := svc.GetTypingUsers(ctx, channel2)

	assert.Len(t, indicators1, 1)
	assert.Len(t, indicators2, 1)
	assert.Equal(t, user1, indicators1[0].UserID)
	assert.Equal(t, user2, indicators2[0].UserID)
}

func TestTypingService_RefreshTyping(t *testing.T) {
	mockEventBus := NewMockEventBusForTyping()
	mockEventBus.On("Publish", "typing.start", mock.AnythingOfType("*models.TypingIndicator")).Return()

	svc := NewTypingService(mockEventBus)
	ctx := context.Background()

	channelID := uuid.New()
	userID := uuid.New()

	_ = svc.StartTyping(ctx, channelID, userID)
	firstIndicators, _ := svc.GetTypingUsers(ctx, channelID)
	firstTimestamp := firstIndicators[0].Timestamp

	// Small delay then refresh
	time.Sleep(10 * time.Millisecond)
	_ = svc.StartTyping(ctx, channelID, userID)

	indicators, _ := svc.GetTypingUsers(ctx, channelID)
	assert.Len(t, indicators, 1)
	assert.True(t, indicators[0].Timestamp.After(firstTimestamp) || indicators[0].Timestamp.Equal(firstTimestamp))
}

func TestTypingService_NilEventBus(t *testing.T) {
	// Should work without event bus
	svc := NewTypingService(nil)
	ctx := context.Background()

	channelID := uuid.New()
	userID := uuid.New()

	err := svc.StartTyping(ctx, channelID, userID)
	assert.NoError(t, err)

	indicators, err := svc.GetTypingUsers(ctx, channelID)
	assert.NoError(t, err)
	assert.Len(t, indicators, 1)
}

func TestTypingService_ConcurrentAccess(t *testing.T) {
	mockEventBus := NewMockEventBusForTyping()
	mockEventBus.On("Publish", "typing.start", mock.AnythingOfType("*models.TypingIndicator")).Return()

	svc := NewTypingService(mockEventBus)
	ctx := context.Background()

	channelID := uuid.New()
	var wg sync.WaitGroup

	// Simulate concurrent typing from multiple users
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			userID := uuid.New()
			_ = svc.StartTyping(ctx, channelID, userID)
			_, _ = svc.GetTypingUsers(ctx, channelID)
			_ = svc.StopTyping(ctx, channelID, userID)
		}()
	}

	wg.Wait()

	// Should complete without deadlock or panic
	indicators, err := svc.GetTypingUsers(ctx, channelID)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, 0, len(indicators)) // May be 0 or more
}

func TestTypingService_EventBusReceivesIndicator(t *testing.T) {
	mockEventBus := NewMockEventBusForTyping()
	
	channelID := uuid.New()
	userID := uuid.New()
	
	mockEventBus.On("Publish", "typing.start", mock.MatchedBy(func(indicator *models.TypingIndicator) bool {
		return indicator.ChannelID == channelID && indicator.UserID == userID
	})).Return()

	svc := NewTypingService(mockEventBus)
	ctx := context.Background()

	err := svc.StartTyping(ctx, channelID, userID)
	assert.NoError(t, err)

	// Verify the event was published with correct data
	mockEventBus.AssertExpectations(t)
}

func TestTypingService_CleanupExpired(t *testing.T) {
	// Create a service with very short TTL for testing
	svc := &TypingService{
		typing:   make(map[uuid.UUID]map[uuid.UUID]time.Time),
		eventBus: nil,
	}

	ctx := context.Background()
	channelID := uuid.New()
	userID := uuid.New()

	// Manually set an old timestamp (expired)
	svc.typing[channelID] = make(map[uuid.UUID]time.Time)
	svc.typing[channelID][userID] = time.Now().Add(-15 * time.Second)

	// Run cleanup
	svc.cleanup()

	// Verify expired indicator was removed
	svc.mu.RLock()
	_, channelExists := svc.typing[channelID]
	svc.mu.RUnlock()
	assert.False(t, channelExists)

	// GetTypingUsers should return empty
	indicators, err := svc.GetTypingUsers(ctx, channelID)
	assert.NoError(t, err)
	assert.Len(t, indicators, 0)
}

func TestTypingService_GetTypingUsersFiltersExpired(t *testing.T) {
	svc := &TypingService{
		typing:   make(map[uuid.UUID]map[uuid.UUID]time.Time),
		eventBus: nil,
	}

	ctx := context.Background()
	channelID := uuid.New()
	activeUser := uuid.New()
	expiredUser := uuid.New()

	// Set up one active and one expired user
	svc.typing[channelID] = make(map[uuid.UUID]time.Time)
	svc.typing[channelID][activeUser] = time.Now()
	svc.typing[channelID][expiredUser] = time.Now().Add(-15 * time.Second)

	indicators, err := svc.GetTypingUsers(ctx, channelID)
	assert.NoError(t, err)
	assert.Len(t, indicators, 1)
	assert.Equal(t, activeUser, indicators[0].UserID)
}
