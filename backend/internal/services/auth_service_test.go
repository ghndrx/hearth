package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"

	"hearth/internal/auth"
	"hearth/internal/models"
)

// ============================================================================
// Mock Repository for JWT-based AuthService
// ============================================================================

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

// testJWTService creates a JWT service for tests
func testJWTService() *auth.JWTService {
	return auth.NewJWTService("test-secret-key", 15*time.Minute, 7*24*time.Hour)
}

// ============================================================================
// JWT-based AuthService Tests
// ============================================================================

func TestAuthService_Register_Success(t *testing.T) {
	mockRepo := new(MockAuthRepository)
	jwtService := testJWTService()
	service := NewAuthService(mockRepo, jwtService)
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

	user, tokens, err := service.Register(ctx, email, username, password)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.NotNil(t, tokens)
	assert.NotEmpty(t, tokens.AccessToken)
	assert.NotEmpty(t, tokens.RefreshToken)
	assert.Equal(t, email, user.Email)
	assert.Equal(t, username, user.Username)
	mockRepo.AssertExpectations(t)
}

func TestAuthService_Register_UserExists(t *testing.T) {
	mockRepo := new(MockAuthRepository)
	jwtService := testJWTService()
	service := NewAuthService(mockRepo, jwtService)
	ctx := context.Background()

	email := "test@example.com"
	username := "testuser"
	password := "password123"

	existingUser := &models.User{Email: email}
	mockRepo.On("GetByEmail", ctx, email).Return(existingUser, nil)

	user, tokens, err := service.Register(ctx, email, username, password)

	assert.ErrorIs(t, err, ErrEmailTaken)
	assert.Nil(t, user)
	assert.Nil(t, tokens)
	mockRepo.AssertExpectations(t)
}

func TestAuthService_Register_RepositoryError(t *testing.T) {
	mockRepo := new(MockAuthRepository)
	jwtService := testJWTService()
	service := NewAuthService(mockRepo, jwtService)
	ctx := context.Background()

	email := "test@example.com"
	username := "testuser"
	password := "password123"

	// Database error when checking for existing user
	mockRepo.On("GetByEmail", ctx, email).Return(nil, errors.New("db error"))

	user, tokens, err := service.Register(ctx, email, username, password)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Nil(t, tokens)
	mockRepo.AssertExpectations(t)
}

func TestAuthService_Login_Success(t *testing.T) {
	mockRepo := new(MockAuthRepository)
	jwtService := testJWTService()
	service := NewAuthService(mockRepo, jwtService)
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

	returnedUser, tokens, err := service.Login(ctx, email, password)

	assert.NoError(t, err)
	assert.NotNil(t, tokens)
	assert.NotEmpty(t, tokens.AccessToken)
	assert.NotEmpty(t, tokens.RefreshToken)
	assert.NotNil(t, returnedUser)
	assert.Equal(t, user.ID, returnedUser.ID)
	mockRepo.AssertExpectations(t)
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	mockRepo := new(MockAuthRepository)
	jwtService := testJWTService()
	service := NewAuthService(mockRepo, jwtService)
	ctx := context.Background()

	email := "test@example.com"
	password := "password123"

	mockRepo.On("GetByEmail", ctx, email).Return(nil, ErrUserNotFound)

	user, tokens, err := service.Login(ctx, email, password)

	assert.ErrorIs(t, err, ErrInvalidCredentials)
	assert.Nil(t, user)
	assert.Nil(t, tokens)
	mockRepo.AssertExpectations(t)
}

func TestAuthService_Login_InvalidPassword(t *testing.T) {
	mockRepo := new(MockAuthRepository)
	jwtService := testJWTService()
	service := NewAuthService(mockRepo, jwtService)
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

	returnedUser, tokens, err := service.Login(ctx, email, wrongPassword)

	assert.ErrorIs(t, err, ErrInvalidCredentials)
	assert.Nil(t, returnedUser)
	assert.Nil(t, tokens)
	mockRepo.AssertExpectations(t)
}

func TestAuthService_Login_RepositoryError(t *testing.T) {
	mockRepo := new(MockAuthRepository)
	jwtService := testJWTService()
	service := NewAuthService(mockRepo, jwtService)
	ctx := context.Background()

	email := "test@example.com"
	password := "password123"

	mockRepo.On("GetByEmail", ctx, email).Return(nil, errors.New("db error"))

	user, tokens, err := service.Login(ctx, email, password)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Nil(t, tokens)
	mockRepo.AssertExpectations(t)
}

func TestAuthService_ValidateToken_Success(t *testing.T) {
	mockRepo := new(MockAuthRepository)
	jwtService := testJWTService()
	service := NewAuthService(mockRepo, jwtService)
	ctx := context.Background()

	userID := uuid.New()
	accessToken, _ := jwtService.GenerateAccessToken(userID, "testuser")

	validatedUserID, err := service.ValidateToken(ctx, accessToken)

	assert.NoError(t, err)
	assert.Equal(t, userID, validatedUserID)
}

func TestAuthService_ValidateToken_Invalid(t *testing.T) {
	mockRepo := new(MockAuthRepository)
	jwtService := testJWTService()
	service := NewAuthService(mockRepo, jwtService)
	ctx := context.Background()

	validatedUserID, err := service.ValidateToken(ctx, "invalid-token")

	assert.Error(t, err)
	assert.Equal(t, uuid.Nil, validatedUserID)
}

func TestAuthService_RefreshTokens_Success(t *testing.T) {
	mockRepo := new(MockAuthRepository)
	jwtService := testJWTService()
	service := NewAuthService(mockRepo, jwtService)
	ctx := context.Background()

	userID := uuid.New()
	_, refreshToken, _ := jwtService.GenerateTokenPair(userID, "testuser")

	tokens, err := service.RefreshTokens(ctx, refreshToken)

	assert.NoError(t, err)
	assert.NotNil(t, tokens)
	assert.NotEmpty(t, tokens.AccessToken)
	assert.NotEmpty(t, tokens.RefreshToken)
}

func TestAuthService_RefreshTokens_Invalid(t *testing.T) {
	mockRepo := new(MockAuthRepository)
	jwtService := testJWTService()
	service := NewAuthService(mockRepo, jwtService)
	ctx := context.Background()

	tokens, err := service.RefreshTokens(ctx, "invalid-refresh-token")

	assert.Error(t, err)
	assert.Nil(t, tokens)
}
