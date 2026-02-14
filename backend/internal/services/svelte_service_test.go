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

// MockSvelteRepository is a mock implementation of SvelteRepository for testing.
type MockSvelteRepository struct {
	mock.Mock
}

func (m *MockSvelteRepository) Create(ctx context.Context, component *models.SvelteComponent) error {
	args := m.Called(ctx, component)
	return args.Error(0)
}

func (m *MockSvelteRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.SvelteComponent, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SvelteComponent), args.Error(1)
}

func (m *MockSvelteRepository) Update(ctx context.Context, component *models.SvelteComponent) error {
	args := m.Called(ctx, component)
	return args.Error(0)
}

func (m *MockSvelteRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSvelteRepository) List(ctx context.Context, limit, offset int) ([]*models.SvelteComponent, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.SvelteComponent), args.Error(1)
}

func TestSvelteService_CreateComponent(t *testing.T) {
	mockRepo := new(MockSvelteRepository)
	service := NewSvelteService(mockRepo)
	ctx := context.Background()
	userID := uuid.New()

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("Create", ctx, mock.AnythingOfType("*models.SvelteComponent")).Return(nil).Once()

		comp, err := service.CreateComponent(ctx, "TestComp", "<div>test</div>", userID)

		assert.NoError(t, err)
		assert.NotNil(t, comp)
		assert.Equal(t, "TestComp", comp.Name)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Empty Name", func(t *testing.T) {
		comp, err := service.CreateComponent(ctx, "", "<div>test</div>", userID)

		assert.Error(t, err)
		assert.Nil(t, comp)
		assert.Equal(t, ErrInvalidInput, err)
	})
}

func TestSvelteService_GetComponent(t *testing.T) {
	mockRepo := new(MockSvelteRepository)
	service := NewSvelteService(mockRepo)
	ctx := context.Background()
	testID := uuid.New()

	expectedComp := &models.SvelteComponent{
		ID:      testID,
		Name:    "FoundComp",
		Content: "content",
	}

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("GetByID", ctx, testID).Return(expectedComp, nil).Once()

		comp, err := service.GetComponent(ctx, testID)

		assert.NoError(t, err)
		assert.Equal(t, expectedComp.ID, comp.ID)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Not Found", func(t *testing.T) {
		mockRepo.On("GetByID", ctx, testID).Return(nil, ErrSvelteNotFound).Once()

		comp, err := service.GetComponent(ctx, testID)

		assert.Error(t, err)
		assert.Nil(t, comp)
		assert.Equal(t, ErrSvelteNotFound, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestSvelteService_UpdateComponent(t *testing.T) {
	mockRepo := new(MockSvelteRepository)
	service := NewSvelteService(mockRepo)
	ctx := context.Background()
	testID := uuid.New()

	existingComp := &models.SvelteComponent{
		ID:        testID,
		Name:      "OldComp",
		Content:   "old content",
		UpdatedAt: time.Now().Add(-1 * time.Hour),
	}

	t.Run("Success", func(t *testing.T) {
		// Setup GetByID call
		mockRepo.On("GetByID", ctx, testID).Return(existingComp, nil).Once()
		// Setup Update call
		mockRepo.On("Update", ctx, mock.AnythingOfType("*models.SvelteComponent")).Return(nil).Once()

		updatedComp, err := service.UpdateComponent(ctx, testID, "new content")

		assert.NoError(t, err)
		assert.Equal(t, "new content", updatedComp.Content)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Component Not Found", func(t *testing.T) {
		mockRepo.On("GetByID", ctx, testID).Return(nil, ErrSvelteNotFound).Once()

		_, err := service.UpdateComponent(ctx, testID, "new content")

		assert.Error(t, err)
		assert.Equal(t, ErrSvelteNotFound, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestSvelteService_DeleteComponent(t *testing.T) {
	mockRepo := new(MockSvelteRepository)
	service := NewSvelteService(mockRepo)
	ctx := context.Background()
	testID := uuid.New()

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("GetByID", ctx, testID).Return(&models.SvelteComponent{ID: testID}, nil).Once()
		mockRepo.On("Delete", ctx, testID).Return(nil).Once()

		err := service.DeleteComponent(ctx, testID)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Failure", func(t *testing.T) {
		mockRepo.On("GetByID", ctx, testID).Return(nil, errors.New("db error")).Once()

		err := service.DeleteComponent(ctx, testID)

		assert.Error(t, err)
		mockRepo.AssertExpectations(t)
	})
}
