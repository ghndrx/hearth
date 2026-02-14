package services

import (
	"context"
	"testing"
	"hearth/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSvelteRepository is a mock implementation of SvelteRepository using testify/mock.
type MockSvelteRepository struct {
	mock.Mock
}

func (m *MockSvelteRepository) Create(ctx context.Context, svelte *models.Svelte) error {
	args := m.Called(ctx, svelte)
	return args.Error(0)
}

func (m *MockSvelteRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Svelte, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Svelte), args.Error(1)
}

func (m *MockSvelteRepository) Update(ctx context.Context, svelte *models.Svelte) error {
	args := m.Called(ctx, svelte)
	return args.Error(0)
}

func (m *MockSvelteRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestSvelteService_CreateComponent(t *testing.T) {
	mockRepo := new(MockSvelteRepository)
	service := NewSvelteService(mockRepo)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		name := "TestComponent"
		code := "<div>Hello</div>"
		
		// Setup expectation
		mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Svelte")).Return(nil).Once()

		result, err := service.CreateComponent(ctx, name, code)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, name, result.Name)
		assert.Equal(t, code, result.Code)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Empty Name", func(t *testing.T) {
		_, err := service.CreateComponent(ctx, "", "code")
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidInput, err)
	})
}

func TestSvelteService_GetComponent(t *testing.T) {
	mockRepo := new(MockSvelteRepository)
	service := NewSvelteService(mockRepo)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		id := uuid.New()
		expected := &models.Svelte{ID: id, Name: "Test", Code: "code"}

		mockRepo.On("GetByID", ctx, id).Return(expected, nil).Once()

		result, err := service.GetComponent(ctx, id)

		assert.NoError(t, err)
		assert.Equal(t, expected, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Not Found", func(t *testing.T) {
		id := uuid.New()
		
		// Simulate DB returning nil, not found error logic
		mockRepo.On("GetByID", ctx, id).Return(nil, nil).Once()

		result, err := service.GetComponent(ctx, id)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, ErrSvelteNotFound, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestSvelteService_DeleteComponent(t *testing.T) {
	mockRepo := new(MockSvelteRepository)
	service := NewSvelteService(mockRepo)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		id := uuid.New()
		existing := &models.Svelte{ID: id}

		mockRepo.On("GetByID", ctx, id).Return(existing, nil).Once()
		mockRepo.On("Delete", ctx, id).Return(nil).Once()

		err := service.DeleteComponent(ctx, id)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Not Found", func(t *testing.T) {
		id := uuid.New()
		
		mockRepo.On("GetByID", ctx, id).Return(nil, nil).Once()

		err := service.DeleteComponent(ctx, id)

		assert.Error(t, err)
		assert.Equal(t, ErrSvelteNotFound, err)
		mockRepo.AssertExpectations(t)
	})
}