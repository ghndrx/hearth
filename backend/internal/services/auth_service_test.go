package services

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"

	"hearth/internal/models"
)

// MockAuthRepository implements authRepository for testing.
type MockAuthRepository struct {
	mock.Mock
}

func (m *MockAuthRepository) Create(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockAuthRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func TestAuthService_Register_Success(t *testing.T) {
	mockRepo := new(MockAuthRepository)
	service := NewAuthService(mockRepo)
	ctx := context.Background()

	email := "test@example.com"
	username := "testuser"
	password := "password123"

	// Expect check for existing user returns not found
	mockRepo.On("GetByEmail", ctx, email).Return(nil, ErrUserNotFound)

	// Expect create user to be called
	mockRepo.On("Create", ctx, mock.AnythingOfType("*models.User")).Return(nil).Run(func(args mock.Arguments) {
		user := args.Get(1).(*models.User)
		assert.Equal(t, email, user.Email)
		assert.Equal(t, username, user.Username)
		assert.NotEmpty(t, user.PasswordHash)
		assert.NotEqual(t, password, user.PasswordHash)
	})

	user, err := service.Register(ctx, email, username, password)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, email, user.Email)
	assert.Equal(t, username, user.Username)
	mockRepo.AssertExpectations(t)
}

func TestAuthService_Register_UserExists(t *testing.T) {
	mockRepo := new(MockAuthRepository)
	service := NewAuthService(mockRepo)
	ctx := context.Background()

	email := "test@example.com"
	username := "testuser"
	password := "password123"

	existingUser := &models.User{Email: email}
	mockRepo.On("GetByEmail", ctx, email).Return(existingUser, nil)

	user, err := service.Register(ctx, email, username, password)

	assert.ErrorIs(t, err, ErrUserExists)
	assert.Nil(t, user)
	mockRepo.AssertExpectations(t)
}

func TestAuthService_Register_RepositoryError(t *testing.T) {
	mockRepo := new(MockAuthRepository)
	service := NewAuthService(mockRepo)
	ctx := context.Background()

	email := "test@example.com"
	username := "testuser"
	password := "password123"

	// Database error when checking for existing user
	mockRepo.On("GetByEmail", ctx, email).Return(nil, errors.New("db error"))

	user, err := service.Register(ctx, email, username, password)

	assert.Error(t, err)
	assert.Nil(t, user)
	mockRepo.AssertExpectations(t)
}

func TestAuthService_Login_Success(t *testing.T) {
	mockRepo := new(MockAuthRepository)
	service := NewAuthService(mockRepo)
	ctx := context.Background()

	email := "test@example.com"
	password := "password123"

	// Hash the password manually to match what the mock repo should return
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	user := &models.User{
		ID:           uuid.New(),
		Email:        email,
		Username:     "testuser",
		PasswordHash: string(hashedPassword),
	}

	mockRepo.On("GetByEmail", ctx, email).Return(user, nil)

	token, returnedUser, err := service.Login(ctx, email, password)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.NotNil(t, returnedUser)
	assert.Equal(t, user.ID, returnedUser.ID)
	mockRepo.AssertExpectations(t)
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	mockRepo := new(MockAuthRepository)
	service := NewAuthService(mockRepo)
	ctx := context.Background()

	email := "test@example.com"
	password := "password123"

	mockRepo.On("GetByEmail", ctx, email).Return(nil, ErrUserNotFound)

	token, user, err := service.Login(ctx, email, password)

	assert.ErrorIs(t, err, ErrInvalidCredentials)
	assert.Nil(t, user)
	assert.Empty(t, token)
	mockRepo.AssertExpectations(t)
}

func TestAuthService_Login_InvalidPassword(t *testing.T) {
	mockRepo := new(MockAuthRepository)
	service := NewAuthService(mockRepo)
	ctx := context.Background()

	email := "test@example.com"
	password := "password123"
	wrongPassword := "wrongpassword"

	// Hash the password
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	user := &models.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: string(hashedPassword),
	}

	mockRepo.On("GetByEmail", ctx, email).Return(user, nil)

	token, returnedUser, err := service.Login(ctx, email, wrongPassword)

	assert.ErrorIs(t, err, ErrInvalidCredentials)
	assert.Nil(t, returnedUser)
	assert.Empty(t, token)
	mockRepo.AssertExpectations(t)
}

func TestAuthService_Login_RepositoryError(t *testing.T) {
	mockRepo := new(MockAuthRepository)
	service := NewAuthService(mockRepo)
	ctx := context.Background()

	email := "test@example.com"
	password := "password123"

	mockRepo.On("GetByEmail", ctx, email).Return(nil, errors.New("db error"))

	token, user, err := service.Login(ctx, email, password)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Empty(t, token)
	mockRepo.AssertExpectations(t)
}
