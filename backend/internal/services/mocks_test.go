package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"hearth/internal/models"
)

// MockCacheService is a mock implementation of CacheService
type MockCacheService struct {
	mock.Mock
}

func (m *MockCacheService) GetUser(ctx context.Context, id uuid.UUID) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockCacheService) SetUser(ctx context.Context, user *models.User, ttl time.Duration) error {
	args := m.Called(ctx, user, ttl)
	return args.Error(0)
}

func (m *MockCacheService) DeleteUser(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockCacheService) GetServer(ctx context.Context, id uuid.UUID) (*models.Server, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Server), args.Error(1)
}

func (m *MockCacheService) SetServer(ctx context.Context, server *models.Server, ttl time.Duration) error {
	args := m.Called(ctx, server, ttl)
	return args.Error(0)
}

func (m *MockCacheService) DeleteServer(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockCacheService) GetChannel(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Channel), args.Error(1)
}

func (m *MockCacheService) SetChannel(ctx context.Context, channel *models.Channel, ttl time.Duration) error {
	args := m.Called(ctx, channel, ttl)
	return args.Error(0)
}

func (m *MockCacheService) DeleteChannel(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockCacheService) Get(ctx context.Context, key string) ([]byte, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockCacheService) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

func (m *MockCacheService) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

// MockEventBus is a mock implementation of EventBus
type MockEventBus struct {
	mock.Mock
}

func (m *MockEventBus) Publish(event string, data interface{}) {
	m.Called(event, data)
}

func (m *MockEventBus) Subscribe(event string, handler func(data interface{})) {
	m.Called(event, handler)
}

func (m *MockEventBus) Unsubscribe(event string, handler func(data interface{})) {
	m.Called(event, handler)
}

// Limits type alias for tests
type Limits = EffectiveLimits
