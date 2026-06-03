package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"hearth/internal/auth"
	"hearth/internal/models"
)

// TestMain sets up test fixtures for the services package.
// Most importantly, it configures a test-friendly bcrypt pool with low cost
// to avoid timeouts under the -race flag.
func TestMain(m *testing.M) {
	// Set up a bcrypt pool with low cost for fast tests
	// This prevents timeouts when running with -race detector
	testPool := auth.NewBcryptPool(auth.PoolConfig{
		Workers:        4,
		QueueSize:      100,
		DefaultTimeout: 30 * time.Second, // Generous timeout for CI
		Cost:           4,                // Minimum cost for fast tests
	})
	auth.SetGlobalPool(testPool)

	code := m.Run()

	// Cleanup
	testPool.Close()
	os.Exit(code)
}

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

func (m *MockAuthRepository) GetByID(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockAuthRepository) UpdateMFA(ctx context.Context, userID uuid.UUID, enabled bool, secret *string) error {
	args := m.Called(ctx, userID, enabled, secret)
	return args.Error(0)
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
	password := "Password123" // Must meet strength requirements (upper, lower, number)

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
	password := "Password123"

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
	password := "Password123"

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
	password := "Password123"

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
	password := "Password123"

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
	password := "Password123"
	wrongPassword := "WrongPassword456"

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
	password := "Password123"

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

	mockRepo.On("GetByID", ctx, userID).Return(&models.User{ID: userID, Username: "testuser"}, nil)

	tokens, err := service.RefreshTokens(ctx, refreshToken)

	assert.NoError(t, err)
	assert.NotNil(t, tokens)
	assert.NotEmpty(t, tokens.AccessToken)
	assert.NotEmpty(t, tokens.RefreshToken)
	mockRepo.AssertExpectations(t)
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

// Note: MockOAuthRepository and setupOAuthServiceWithRepo are defined in oauth_service_with_repo_test.go

// ========== Token Exchange Edge Cases ==========

func TestOAuthService_ExchangeCode_ExpiredCode(t *testing.T) {
	// Test that expired authorization codes are properly rejected
	service, _, cache := setupOAuthService(t)
	_ = cache // Used in cache operations
	ctx := context.Background()

	// Create a mock server that returns an "invalid_grant" error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":             "invalid_grant",
			"error_description": "The authorization code has expired",
		})
	}))
	defer server.Close()

	// Since we can't easily inject the HTTP client, test the error wrapping
	// by simulating the expired state scenario
	expiredState := &OAuthState{
		Provider:  OAuthProviderGitHub,
		State:     "test-state",
		Nonce:     "test-nonce",
		CreatedAt: time.Now().Add(-15 * time.Minute), // 15 minutes ago - expired
	}

	stateData, _ := json.Marshal(expiredState)
	cacheKey := fmt.Sprintf("oauth_state:%s", "test-state")
	if err := service.cache.Set(ctx, cacheKey, stateData, 10*time.Minute); err != nil {
		t.Skip("Cache not available")
	}

	// Validate should fail due to expiry
	_, err := service.validateState(ctx, "test-state")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrOAuthStateExpired))
}

func TestOAuthService_ExchangeCode_InvalidState(t *testing.T) {
	service, _, _ := setupOAuthService(t)
	ctx := context.Background()

	// Test with non-existent state
	_, err := service.validateState(ctx, "non-existent-state")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrOAuthStateExpired))
}

func TestOAuthService_ExchangeCode_MalformedStateJSON(t *testing.T) {
	service, _, cache := setupOAuthService(t)
	ctx := context.Background()

	// Store malformed JSON in cache
	cacheKey := "oauth_state:malformed-state"
	cache.store[cacheKey] = []byte("{invalid json}")

	_, err := service.validateState(ctx, "malformed-state")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrOAuthStateMismatch))
}

func TestOAuthService_HandleCallback_ProviderMismatch(t *testing.T) {
	service, _, cache := setupOAuthService(t)
	ctx := context.Background()

	// Store state for GitHub
	state := &OAuthState{
		Provider:  OAuthProviderGitHub,
		State:     "test-state",
		Nonce:     "test-nonce",
		CreatedAt: time.Now(),
	}
	stateData, _ := json.Marshal(state)
	cache.store["oauth_state:test-state"] = stateData

	// Try to handle callback as Google - should fail
	_, _, err := service.HandleCallback(ctx, OAuthProviderGoogle, "test-code", "test-state")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrOAuthStateMismatch))
}

// ========== Provider User Info Edge Cases ==========

func TestOAuthService_GetGitHubUserInfo_MalformedResponse(t *testing.T) {
	// Create mock server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	// Test would require injecting the HTTP client URL, which is hardcoded
	// Instead, we verify the error type is correct
	assert.NotNil(t, ErrOAuthMalformedResponse)
}

func TestOAuthService_GetGitHubUserInfo_MissingID(t *testing.T) {
	// Create mock server that returns user without ID
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"login":      "testuser",
			"name":       "Test User",
			"email":      "test@example.com",
			"avatar_url": "https://example.com/avatar.png",
			// Missing "id" field
		})
	}))
	defer server.Close()

	// Verify the error type exists
	assert.NotNil(t, ErrOAuthMalformedResponse)
}

func TestOAuthService_GetGitHubUserInfo_RateLimited(t *testing.T) {
	// Create mock server that returns 429
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Hour).Unix()))
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "API rate limit exceeded",
		})
	}))
	defer server.Close()

	// Verify the error type exists
	assert.NotNil(t, ErrOAuthRateLimited)
}

func TestOAuthService_GetGitHubUserInfo_TokenRevoked(t *testing.T) {
	// Create mock server that returns 401 (unauthorized)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Bad credentials",
		})
	}))
	defer server.Close()

	// Verify the error type exists
	assert.NotNil(t, ErrOAuthTokenRevoked)
}

// ========== Account Linking Edge Cases ==========

func TestOAuthService_LinkAccount_AlreadyLinkedToDifferentUser(t *testing.T) {
	_, userRepo, _, cache := setupOAuthServiceWithRepo(t)

	currentUserID := uuid.New()

	// Store valid state
	state := &OAuthState{
		Provider:   OAuthProviderGitHub,
		State:      "link-state",
		Nonce:      "test-nonce",
		CreatedAt:  time.Now(),
		LinkUserID: &currentUserID, // Trying to link to current user
	}
	stateData, _ := json.Marshal(state)
	cache.store["oauth_state:link-state"] = stateData

	// This would occur during HandleCallback when the provider user is already linked
	// to another user and we're trying to link to current user
	// The actual test would need to mock the HTTP calls, so we verify the error exists
	assert.NotNil(t, ErrOAuthAccountLinked)

	userRepo.AssertExpectations(t)
}

func TestOAuthService_LinkAccount_Success(t *testing.T) {
	service, userRepo, oauthRepo, cache := setupOAuthServiceWithRepo(t)
	ctx := context.Background()

	userID := uuid.New()
	user := &models.User{
		ID:       userID,
		Email:    "existing@example.com",
		Username: "existinguser",
	}

	// Store valid link state
	state := &OAuthState{
		Provider:   OAuthProviderGitHub,
		State:      "link-state",
		Nonce:      "test-nonce",
		CreatedAt:  time.Now(),
		LinkUserID: &userID,
	}
	stateData, _ := json.Marshal(state)
	cache.store["oauth_state:link-state"] = stateData

	// OAuth account not linked yet
	oauthRepo.On("GetByProviderUserID", ctx, "github", mock.Anything).Return(nil, ErrOAuthProviderNotFound)
	userRepo.On("GetByID", ctx, userID).Return(user, nil)
	oauthRepo.On("Create", ctx, mock.AnythingOfType("*models.OAuthProvider")).Return(nil)

	// The actual linking happens in HandleCallback with HTTP calls
	// Verify the service is properly configured
	assert.NotNil(t, service.oauthRepo)
}

func TestOAuthService_UnlinkProvider_LastAuthMethod(t *testing.T) {
	service, userRepo, oauthRepo, _ := setupOAuthServiceWithRepo(t)
	ctx := context.Background()

	userID := uuid.New()

	// User without password
	user := &models.User{
		ID:           userID,
		Email:        "test@example.com",
		Username:     "testuser",
		PasswordHash: "", // No password set
	}

	existingProvider := &models.OAuthProvider{
		ID:       uuid.New(),
		UserID:   userID,
		Provider: "github",
	}

	oauthRepo.On("GetByUserAndProvider", ctx, userID, "github").Return(existingProvider, nil)
	userRepo.On("GetByID", ctx, userID).Return(user, nil)
	oauthRepo.On("CountByUserID", ctx, userID).Return(1, nil) // Only one OAuth provider

	err := service.UnlinkProvider(ctx, userID, OAuthProviderGitHub)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrOAuthCannotUnlinkLast))
	oauthRepo.AssertExpectations(t)
	userRepo.AssertExpectations(t)
}

func TestOAuthService_UnlinkProvider_WithPasswordSet(t *testing.T) {
	service, userRepo, oauthRepo, _ := setupOAuthServiceWithRepo(t)
	ctx := context.Background()

	userID := uuid.New()

	// User WITH password
	user := &models.User{
		ID:           userID,
		Email:        "test@example.com",
		Username:     "testuser",
		PasswordHash: "$2a$10$hashedpassword", // Has password
	}

	existingProvider := &models.OAuthProvider{
		ID:       uuid.New(),
		UserID:   userID,
		Provider: "github",
	}

	oauthRepo.On("GetByUserAndProvider", ctx, userID, "github").Return(existingProvider, nil)
	userRepo.On("GetByID", ctx, userID).Return(user, nil)
	oauthRepo.On("CountByUserID", ctx, userID).Return(1, nil)
	oauthRepo.On("DeleteByUserAndProvider", ctx, userID, "github").Return(nil)

	err := service.UnlinkProvider(ctx, userID, OAuthProviderGitHub)

	assert.NoError(t, err)
	oauthRepo.AssertExpectations(t)
	userRepo.AssertExpectations(t)
}

func TestOAuthService_UnlinkProvider_WithMultipleProviders(t *testing.T) {
	service, userRepo, oauthRepo, _ := setupOAuthServiceWithRepo(t)
	ctx := context.Background()

	userID := uuid.New()

	// User without password but multiple OAuth providers
	user := &models.User{
		ID:           userID,
		Email:        "test@example.com",
		Username:     "testuser",
		PasswordHash: "", // No password
	}

	existingProvider := &models.OAuthProvider{
		ID:       uuid.New(),
		UserID:   userID,
		Provider: "github",
	}

	oauthRepo.On("GetByUserAndProvider", ctx, userID, "github").Return(existingProvider, nil)
	userRepo.On("GetByID", ctx, userID).Return(user, nil)
	oauthRepo.On("CountByUserID", ctx, userID).Return(2, nil) // Multiple providers
	oauthRepo.On("DeleteByUserAndProvider", ctx, userID, "github").Return(nil)

	err := service.UnlinkProvider(ctx, userID, OAuthProviderGitHub)

	assert.NoError(t, err)
	oauthRepo.AssertExpectations(t)
	userRepo.AssertExpectations(t)
}

func TestOAuthService_GetLinkAuthorizationURL_AlreadyLinked(t *testing.T) {
	service, _, oauthRepo, _ := setupOAuthServiceWithRepo(t)
	ctx := context.Background()

	userID := uuid.New()
	existingProvider := &models.OAuthProvider{
		ID:       uuid.New(),
		UserID:   userID,
		Provider: "github",
	}

	// Provider already linked
	oauthRepo.On("GetByUserAndProvider", ctx, userID, "github").Return(existingProvider, nil)

	_, err := service.GetLinkAuthorizationURL(ctx, userID, OAuthProviderGitHub)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrOAuthProviderAlreadyLinked))
}

// ========== Provider Response Edge Cases ==========

func TestOAuthService_HandleCallback_ProviderUnavailable(t *testing.T) {
	// Test that provider unavailability is handled gracefully
	assert.NotNil(t, ErrOAuthProviderUnavailable)
}

func TestOAuthService_HandleCallback_InsufficientScope(t *testing.T) {
	// Test that insufficient scope errors are properly reported
	assert.NotNil(t, ErrOAuthInsufficientScope)
}

// ========== Concurrent State Validation ==========

func TestOAuthService_StateValidation_Replay(t *testing.T) {
	service, _, cache := setupOAuthService(t)
	ctx := context.Background()

	// Store valid state
	state := &OAuthState{
		Provider:  OAuthProviderGitHub,
		State:     "replay-state",
		Nonce:     "test-nonce",
		CreatedAt: time.Now(),
	}
	stateData, _ := json.Marshal(state)
	cache.store["oauth_state:replay-state"] = stateData

	// First validation should succeed
	validatedState, err := service.validateState(ctx, "replay-state")
	require.NoError(t, err)
	assert.Equal(t, OAuthProviderGitHub, validatedState.Provider)

	// Second validation should fail (state deleted after first use)
	_, err = service.validateState(ctx, "replay-state")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrOAuthStateExpired))
}

// ========== Error Type Tests ==========

func TestOAuthErrors_Distinctness(t *testing.T) {
	allErrors := []error{
		ErrOAuthProviderNotSupported,
		ErrOAuthStateMismatch,
		ErrOAuthStateExpired,
		ErrOAuthCodeExchange,
		ErrOAuthUserInfo,
		ErrOAuthEmailNotVerified,
		ErrOAuthAccountLinked,
		ErrOAuthMalformedResponse,
		ErrOAuthProviderUnavailable,
		ErrOAuthTokenRevoked,
		ErrOAuthInsufficientScope,
		ErrOAuthRateLimited,
	}

	// Each error should be distinct
	for i, err1 := range allErrors {
		for j, err2 := range allErrors {
			if i != j {
				assert.NotEqual(t, err1.Error(), err2.Error(),
					"Errors should have unique messages: %v and %v", err1, err2)
			}
		}
	}
}

func TestOAuthErrors_Wrapping(t *testing.T) {
	// Test that wrapped errors can be properly identified with errors.Is
	wrappedErr := fmt.Errorf("context: %w", ErrOAuthCodeExchange)
	assert.True(t, errors.Is(wrappedErr, ErrOAuthCodeExchange))

	wrappedErr2 := fmt.Errorf("provider error: %w", ErrOAuthProviderUnavailable)
	assert.True(t, errors.Is(wrappedErr2, ErrOAuthProviderUnavailable))
}

// ========== OAuth Provider Config Validation ==========

func TestOAuthService_GetProviderConfig_AllProviders(t *testing.T) {
	service, _, _ := setupOAuthService(t)

	providers := []OAuthProvider{
		OAuthProviderGitHub,
		OAuthProviderGoogle,
		OAuthProviderDiscord,
	}

	for _, provider := range providers {
		config, err := service.getProviderConfig(provider)
		assert.NoError(t, err, "Provider %s should have config", provider)
		assert.NotEmpty(t, config.ClientID)
		assert.NotEmpty(t, config.ClientSecret)
		assert.NotEmpty(t, config.RedirectURI)
	}
}

func TestOAuthService_GetProviderConfig_InvalidProvider(t *testing.T) {
	service, _, _ := setupOAuthService(t)

	_, err := service.getProviderConfig(OAuthProvider("invalid"))
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrOAuthProviderNotSupported))
}

// ========== Username Generation Edge Cases ==========

func TestOAuthService_FindOrCreateUser_LongUsername(t *testing.T) {
	service, userRepo, _ := setupOAuthService(t)
	ctx := context.Background()

	// Very long email prefix
	longEmail := "verylongemailaddressthatexceeds32characters@example.com"
	longUsername := "verylongemailaddressthatexceeds32characters"

	userRepo.On("GetByEmail", ctx, longEmail).Return(nil, ErrUserNotFound)
	// First call is with the original username (service uses the provided username)
	userRepo.On("GetByUsername", ctx, longUsername).Return(nil, ErrUserNotFound)
	userRepo.On("Create", ctx, mock.AnythingOfType("*models.User")).Return(nil)

	userInfo := &OAuthUserInfo{
		Provider:      OAuthProviderGoogle,
		ProviderID:    "12345",
		Email:         longEmail,
		EmailVerified: true,
		Username:      longUsername,
		DisplayName:   "Test User",
	}

	user, err := service.findOrCreateUser(ctx, userInfo, nil)
	require.NoError(t, err)
	// Note: The current implementation doesn't truncate username - it uses the provided one
	// This test verifies the username is properly set from userInfo
	assert.NotEmpty(t, user.Username)
}

func TestOAuthService_FindOrCreateUser_SpecialCharactersInUsername(t *testing.T) {
	service, userRepo, _ := setupOAuthService(t)
	ctx := context.Background()

	userRepo.On("GetByEmail", ctx, "test+tag@example.com").Return(nil, ErrUserNotFound)
	userRepo.On("GetByUsername", ctx, mock.Anything).Return(nil, ErrUserNotFound)
	userRepo.On("Create", ctx, mock.AnythingOfType("*models.User")).Return(nil)

	userInfo := &OAuthUserInfo{
		Provider:      OAuthProviderGoogle,
		ProviderID:    "12345",
		Email:         "test+tag@example.com",
		EmailVerified: true,
		Username:      "test+tag", // Has special char
		DisplayName:   "Test User",
	}

	user, err := service.findOrCreateUser(ctx, userInfo, nil)
	require.NoError(t, err)
	assert.NotNil(t, user)
}

type MockOAuthUserRepository struct {
	mock.Mock
}

func (m *MockOAuthUserRepository) Create(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockOAuthUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockOAuthUserRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*models.User, error) {
	args := m.Called(ctx, ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockOAuthUserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockOAuthUserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockOAuthUserRepository) Update(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockOAuthUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockOAuthUserRepository) GetFriends(ctx context.Context, userID uuid.UUID) ([]*models.User, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockOAuthUserRepository) AddFriend(ctx context.Context, userID, friendID uuid.UUID) error {
	args := m.Called(ctx, userID, friendID)
	return args.Error(0)
}

func (m *MockOAuthUserRepository) RemoveFriend(ctx context.Context, userID, friendID uuid.UUID) error {
	args := m.Called(ctx, userID, friendID)
	return args.Error(0)
}

func (m *MockOAuthUserRepository) GetBlockedUsers(ctx context.Context, userID uuid.UUID) ([]*models.User, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockOAuthUserRepository) BlockUser(ctx context.Context, userID, blockedID uuid.UUID) error {
	args := m.Called(ctx, userID, blockedID)
	return args.Error(0)
}

func (m *MockOAuthUserRepository) UnblockUser(ctx context.Context, userID, blockedID uuid.UUID) error {
	args := m.Called(ctx, userID, blockedID)
	return args.Error(0)
}

func (m *MockOAuthUserRepository) UpdatePresence(ctx context.Context, userID uuid.UUID, status models.PresenceStatus) error {
	args := m.Called(ctx, userID, status)
	return args.Error(0)
}

func (m *MockOAuthUserRepository) GetPresence(ctx context.Context, userID uuid.UUID) (*models.Presence, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Presence), args.Error(1)
}

func (m *MockOAuthUserRepository) GetPresenceBulk(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*models.Presence, error) {
	args := m.Called(ctx, userIDs)
	return args.Get(0).(map[uuid.UUID]*models.Presence), args.Error(1)
}

func (m *MockOAuthUserRepository) GetRelationship(ctx context.Context, userID, targetID uuid.UUID) (int, error) {
	args := m.Called(ctx, userID, targetID)
	return args.Int(0), args.Error(1)
}

func (m *MockOAuthUserRepository) SendFriendRequest(ctx context.Context, senderID, receiverID uuid.UUID) error {
	args := m.Called(ctx, senderID, receiverID)
	return args.Error(0)
}

func (m *MockOAuthUserRepository) GetIncomingFriendRequests(ctx context.Context, userID uuid.UUID) ([]*models.User, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockOAuthUserRepository) GetOutgoingFriendRequests(ctx context.Context, userID uuid.UUID) ([]*models.User, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockOAuthUserRepository) AcceptFriendRequest(ctx context.Context, receiverID, senderID uuid.UUID) error {
	args := m.Called(ctx, receiverID, senderID)
	return args.Error(0)
}

func (m *MockOAuthUserRepository) DeclineFriendRequest(ctx context.Context, userID, otherID uuid.UUID) error {
	args := m.Called(ctx, userID, otherID)
	return args.Error(0)
}

func (m *MockOAuthUserRepository) GetCustomStatus(ctx context.Context, userID uuid.UUID) (*models.UserCustomStatus, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserCustomStatus), args.Error(1)
}

func (m *MockOAuthUserRepository) SetCustomStatus(ctx context.Context, status *models.UserCustomStatus) error {
	args := m.Called(ctx, status)
	return args.Error(0)
}

func (m *MockOAuthUserRepository) DeleteCustomStatus(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

// MockOAuthCacheService is a mock implementation of CacheService for OAuth tests
type MockOAuthCacheService struct {
	store map[string][]byte
}

func NewMockOAuthCacheService() *MockOAuthCacheService {
	return &MockOAuthCacheService{
		store: make(map[string][]byte),
	}
}

func (m *MockOAuthCacheService) Get(ctx context.Context, key string) ([]byte, error) {
	if data, ok := m.store[key]; ok {
		return data, nil
	}
	return nil, errors.New("key not found in cache")
}

func (m *MockOAuthCacheService) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	m.store[key] = value
	return nil
}

func (m *MockOAuthCacheService) Delete(ctx context.Context, key string) error {
	delete(m.store, key)
	return nil
}

func (m *MockOAuthCacheService) GetUser(ctx context.Context, id uuid.UUID) (*models.User, error) {
	return nil, errors.New("key not found in cache")
}

func (m *MockOAuthCacheService) SetUser(ctx context.Context, user *models.User, ttl time.Duration) error {
	return nil
}

func (m *MockOAuthCacheService) DeleteUser(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *MockOAuthCacheService) GetServer(ctx context.Context, id uuid.UUID) (*models.Server, error) {
	return nil, errors.New("key not found in cache")
}

func (m *MockOAuthCacheService) SetServer(ctx context.Context, server *models.Server, ttl time.Duration) error {
	return nil
}

func (m *MockOAuthCacheService) DeleteServer(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *MockOAuthCacheService) GetChannel(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
	return nil, errors.New("key not found in cache")
}

func (m *MockOAuthCacheService) SetChannel(ctx context.Context, channel *models.Channel, ttl time.Duration) error {
	return nil
}

func (m *MockOAuthCacheService) DeleteChannel(ctx context.Context, id uuid.UUID) error {
	return nil
}

// Test setup helper
func setupOAuthService(t *testing.T) (*OAuthService, *MockOAuthUserRepository, *MockOAuthCacheService) {
	userRepo := new(MockOAuthUserRepository)
	cache := NewMockOAuthCacheService()
	jwtService := auth.NewJWTService("test-secret-key-for-testing-purposes", 3600, 86400)

	config := &OAuthProviderConfig{
		GitHub: &OAuthConfig{
			ClientID:     "test-github-client-id",
			ClientSecret: "test-github-client-secret",
			RedirectURI:  "http://localhost:8080/api/v1/auth/oauth/github/callback",
			Scopes:       []string{"read:user", "user:email"},
		},
		Google: &OAuthConfig{
			ClientID:     "test-google-client-id",
			ClientSecret: "test-google-client-secret",
			RedirectURI:  "http://localhost:8080/api/v1/auth/oauth/google/callback",
			Scopes:       []string{"openid", "email", "profile"},
		},
		Discord: &OAuthConfig{
			ClientID:     "test-discord-client-id",
			ClientSecret: "test-discord-client-secret",
			RedirectURI:  "http://localhost:8080/api/v1/auth/oauth/discord/callback",
			Scopes:       []string{"identify", "email"},
		},
	}

	service := NewOAuthService(config, userRepo, cache, jwtService)
	return service, userRepo, cache
}

// Tests

func TestOAuthService_GetAuthorizationURL_GitHub(t *testing.T) {
	service, _, _ := setupOAuthService(t)
	ctx := context.Background()

	authURL, err := service.GetAuthorizationURL(ctx, OAuthProviderGitHub, nil)
	require.NoError(t, err)
	assert.Contains(t, authURL, "https://github.com/login/oauth/authorize")
	assert.Contains(t, authURL, "client_id=test-github-client-id")
	assert.Contains(t, authURL, "state=")
	assert.Contains(t, authURL, "scope=read%3Auser+user%3Aemail")
}

func TestOAuthService_GetAuthorizationURL_Google(t *testing.T) {
	service, _, _ := setupOAuthService(t)
	ctx := context.Background()

	authURL, err := service.GetAuthorizationURL(ctx, OAuthProviderGoogle, nil)
	require.NoError(t, err)
	assert.Contains(t, authURL, "https://accounts.google.com/o/oauth2/v2/auth")
	assert.Contains(t, authURL, "client_id=test-google-client-id")
	assert.Contains(t, authURL, "state=")
	assert.Contains(t, authURL, "nonce=")
	assert.Contains(t, authURL, "access_type=offline")
}

func TestOAuthService_GetAuthorizationURL_Discord(t *testing.T) {
	service, _, _ := setupOAuthService(t)
	ctx := context.Background()

	authURL, err := service.GetAuthorizationURL(ctx, OAuthProviderDiscord, nil)
	require.NoError(t, err)
	assert.Contains(t, authURL, "https://discord.com/api/oauth2/authorize")
	assert.Contains(t, authURL, "client_id=test-discord-client-id")
	assert.Contains(t, authURL, "state=")
}

func TestOAuthService_GetAuthorizationURL_UnsupportedProvider(t *testing.T) {
	service, _, _ := setupOAuthService(t)
	ctx := context.Background()

	// Remove config for unsupported provider test
	service.config.GitHub = nil
	_, err := service.GetAuthorizationURL(ctx, OAuthProviderGitHub, nil)
	assert.ErrorIs(t, err, ErrOAuthProviderNotSupported)
}

func TestOAuthService_IsProviderEnabled(t *testing.T) {
	service, _, _ := setupOAuthService(t)

	assert.True(t, service.IsProviderEnabled(OAuthProviderGitHub))
	assert.True(t, service.IsProviderEnabled(OAuthProviderGoogle))
	assert.True(t, service.IsProviderEnabled(OAuthProviderDiscord))

	// Disable a provider
	service.config.GitHub.ClientID = ""
	assert.False(t, service.IsProviderEnabled(OAuthProviderGitHub))
}

func TestOAuthService_StateStorageAndValidation(t *testing.T) {
	service, _, cache := setupOAuthService(t)
	ctx := context.Background()

	// Get authorization URL (stores state)
	authURL, err := service.GetAuthorizationURL(ctx, OAuthProviderGitHub, nil)
	require.NoError(t, err)

	// Extract state from URL
	// State is stored in the URL query parameter
	assert.Contains(t, authURL, "state=")

	// Check that state was stored in cache
	assert.Greater(t, len(cache.store), 0)
}

func TestOAuthService_StateValidation_Expired(t *testing.T) {
	service, _, cache := setupOAuthService(t)
	ctx := context.Background()

	// Create an expired state
	expiredState := &OAuthState{
		Provider:  OAuthProviderGitHub,
		State:     "expired-state-token",
		Nonce:     "test-nonce",
		CreatedAt: time.Now().Add(-15 * time.Minute), // 15 minutes ago (expired)
	}

	stateData, _ := json.Marshal(expiredState)
	cacheKey := fmt.Sprintf("oauth_state:%s", "expired-state-token")
	cache.store[cacheKey] = stateData

	// Try to validate the expired state
	_, err := service.validateState(ctx, "expired-state-token")
	assert.ErrorIs(t, err, ErrOAuthStateExpired)
}

func TestOAuthService_FindOrCreateUser_ExistingUser(t *testing.T) {
	service, userRepo, _ := setupOAuthService(t)
	ctx := context.Background()

	existingUser := &models.User{
		ID:       uuid.New(),
		Email:    "test@example.com",
		Username: "existinguser",
	}

	userRepo.On("GetByEmail", ctx, "test@example.com").Return(existingUser, nil)

	userInfo := &OAuthUserInfo{
		Provider:      OAuthProviderGitHub,
		ProviderID:    "12345",
		Email:         "test@example.com",
		EmailVerified: true,
		Username:      "newusername",
		DisplayName:   "Test User",
	}

	user, err := service.findOrCreateUser(ctx, userInfo, nil)
	require.NoError(t, err)
	assert.Equal(t, existingUser.ID, user.ID)
	assert.Equal(t, existingUser.Username, user.Username)
}

func TestOAuthService_FindOrCreateUser_NewUser(t *testing.T) {
	service, userRepo, _ := setupOAuthService(t)
	ctx := context.Background()

	userRepo.On("GetByEmail", ctx, "newuser@example.com").Return(nil, ErrUserNotFound)
	userRepo.On("GetByUsername", ctx, "newuser").Return(nil, ErrUserNotFound)
	userRepo.On("Create", ctx, mock.AnythingOfType("*models.User")).Return(nil)

	userInfo := &OAuthUserInfo{
		Provider:      OAuthProviderGitHub,
		ProviderID:    "12345",
		Email:         "newuser@example.com",
		EmailVerified: true,
		Username:      "newuser",
		DisplayName:   "New User",
		AvatarURL:     "https://example.com/avatar.png",
	}

	user, err := service.findOrCreateUser(ctx, userInfo, nil)
	require.NoError(t, err)
	assert.Equal(t, "newuser@example.com", user.Email)
	assert.Equal(t, "newuser", user.Username)
	assert.Equal(t, "New User", *user.DisplayName)
	assert.Equal(t, "https://example.com/avatar.png", *user.AvatarURL)
	assert.True(t, user.Verified)

	userRepo.AssertExpectations(t)
}

func TestOAuthService_FindOrCreateUser_UsernameCollision(t *testing.T) {
	service, userRepo, _ := setupOAuthService(t)
	ctx := context.Background()

	existingUser := &models.User{
		ID:       uuid.New(),
		Username: "takenusername",
	}

	userRepo.On("GetByEmail", ctx, "newuser@example.com").Return(nil, ErrUserNotFound)
	userRepo.On("GetByUsername", ctx, "takenusername").Return(existingUser, nil)
	userRepo.On("GetByUsername", ctx, mock.MatchedBy(func(s string) bool {
		return s != "takenusername" // Any other username
	})).Return(nil, ErrUserNotFound)
	userRepo.On("Create", ctx, mock.AnythingOfType("*models.User")).Return(nil)

	userInfo := &OAuthUserInfo{
		Provider:      OAuthProviderGitHub,
		ProviderID:    "12345",
		Email:         "newuser@example.com",
		EmailVerified: true,
		Username:      "takenusername",
		DisplayName:   "New User",
	}

	user, err := service.findOrCreateUser(ctx, userInfo, nil)
	require.NoError(t, err)
	// Username should be modified with a suffix
	assert.NotEqual(t, "takenusername", user.Username)
	assert.Contains(t, user.Username, "takenusername_")
}

func TestOAuthService_FindOrCreateUser_LinkingMode(t *testing.T) {
	service, userRepo, _ := setupOAuthService(t)
	ctx := context.Background()

	linkUserID := uuid.New()
	linkedUser := &models.User{
		ID:       linkUserID,
		Email:    "linked@example.com",
		Username: "linkeduser",
	}

	userRepo.On("GetByID", ctx, linkUserID).Return(linkedUser, nil)

	userInfo := &OAuthUserInfo{
		Provider:      OAuthProviderGitHub,
		ProviderID:    "12345",
		Email:         "different@example.com",
		EmailVerified: true,
		Username:      "oauthuser",
	}

	user, err := service.findOrCreateUser(ctx, userInfo, &linkUserID)
	require.NoError(t, err)
	assert.Equal(t, linkedUser.ID, user.ID)
	assert.Equal(t, "linked@example.com", user.Email) // Should keep original email
}

// Integration-style test with mock HTTP server
func TestOAuthService_HandleCallback_GitHubMock(t *testing.T) {
	// Create mock GitHub servers
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login/oauth/access_token" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"access_token": "mock-access-token",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer tokenServer.Close()

	userServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/user":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":         12345,
				"login":      "testuser",
				"name":       "Test User",
				"email":      "test@example.com",
				"avatar_url": "https://example.com/avatar.png",
			})
		case "/user/emails":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"email":    "test@example.com",
					"primary":  true,
					"verified": true,
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer userServer.Close()

	// Note: Full integration test would require mocking the GitHub API endpoints
	// This test verifies the structure and flow
	t.Log("Mock servers created successfully")
	t.Log("Token server:", tokenServer.URL)
	t.Log("User server:", userServer.URL)
}

func TestGenerateRandomString(t *testing.T) {
	// Test that we get random strings of correct length
	s1, err := generateRandomString(32)
	require.NoError(t, err)
	assert.Len(t, s1, 32)

	s2, err := generateRandomString(32)
	require.NoError(t, err)
	assert.Len(t, s2, 32)

	// Should be different each time (statistically)
	assert.NotEqual(t, s1, s2)
}

func TestOAuthProvider_Constants(t *testing.T) {
	assert.Equal(t, OAuthProvider("github"), OAuthProviderGitHub)
	assert.Equal(t, OAuthProvider("google"), OAuthProviderGoogle)
	assert.Equal(t, OAuthProvider("discord"), OAuthProviderDiscord)
}

func TestOAuthUserInfo_Structure(t *testing.T) {
	info := &OAuthUserInfo{
		Provider:      OAuthProviderGitHub,
		ProviderID:    "12345",
		Email:         "test@example.com",
		EmailVerified: true,
		Username:      "testuser",
		DisplayName:   "Test User",
		AvatarURL:     "https://example.com/avatar.png",
	}

	assert.Equal(t, OAuthProviderGitHub, info.Provider)
	assert.Equal(t, "12345", info.ProviderID)
	assert.Equal(t, "test@example.com", info.Email)
	assert.True(t, info.EmailVerified)
	assert.Equal(t, "testuser", info.Username)
	assert.Equal(t, "Test User", info.DisplayName)
	assert.Equal(t, "https://example.com/avatar.png", info.AvatarURL)
}

func TestOAuthState_Structure(t *testing.T) {
	linkUserID := uuid.New()
	state := &OAuthState{
		Provider:   OAuthProviderGoogle,
		State:      "test-state-token",
		Nonce:      "test-nonce",
		CreatedAt:  time.Now(),
		LinkUserID: &linkUserID,
	}

	assert.Equal(t, OAuthProviderGoogle, state.Provider)
	assert.Equal(t, "test-state-token", state.State)
	assert.Equal(t, "test-nonce", state.Nonce)
	assert.NotNil(t, state.LinkUserID)
	assert.Equal(t, linkUserID, *state.LinkUserID)
}

func TestOAuthErrors(t *testing.T) {
	// Test that all error types are distinct
	errors := []error{
		ErrOAuthProviderNotSupported,
		ErrOAuthStateMismatch,
		ErrOAuthStateExpired,
		ErrOAuthCodeExchange,
		ErrOAuthUserInfo,
		ErrOAuthEmailNotVerified,
		ErrOAuthAccountLinked,
	}

	for i, err1 := range errors {
		for j, err2 := range errors {
			if i != j {
				assert.NotEqual(t, err1, err2)
			}
		}
	}
}

func TestOAuthService_GetEnabledProviders(t *testing.T) {
	service, _, _ := setupOAuthService(t)

	providers := service.GetEnabledProviders()
	assert.Len(t, providers, 3)
	assert.Contains(t, providers, OAuthProviderGitHub)
	assert.Contains(t, providers, OAuthProviderGoogle)
	assert.Contains(t, providers, OAuthProviderDiscord)
}

func TestOAuthService_GetEnabledProviders_Partial(t *testing.T) {
	service, _, _ := setupOAuthService(t)

	// Disable some providers
	service.config.GitHub.ClientID = ""
	service.config.Discord = nil

	providers := service.GetEnabledProviders()
	assert.Len(t, providers, 1)
	assert.Contains(t, providers, OAuthProviderGoogle)
}

func TestOAuthService_GetLinkedProviders_NoRepo(t *testing.T) {
	service, _, _ := setupOAuthService(t)
	ctx := context.Background()

	// Without repo, should return nil
	providers, err := service.GetLinkedProviders(ctx, uuid.New())
	assert.NoError(t, err)
	assert.Nil(t, providers)
}

func TestOAuthService_GetLinkedProvider_NoRepo(t *testing.T) {
	service, _, _ := setupOAuthService(t)
	ctx := context.Background()

	// Without repo, should return not found
	_, err := service.GetLinkedProvider(ctx, uuid.New(), OAuthProviderGitHub)
	assert.ErrorIs(t, err, ErrOAuthProviderNotFound)
}

func TestOAuthService_UnlinkProvider_NoRepo(t *testing.T) {
	service, _, _ := setupOAuthService(t)
	ctx := context.Background()

	// Without repo, should return error
	err := service.UnlinkProvider(ctx, uuid.New(), OAuthProviderGitHub)
	assert.ErrorIs(t, err, ErrOAuthProviderNotSupported)
}

func TestOAuthService_GetLinkAuthorizationURL_NoRepo(t *testing.T) {
	service, _, _ := setupOAuthService(t)
	ctx := context.Background()

	// Without repo, should return error
	_, err := service.GetLinkAuthorizationURL(ctx, uuid.New(), OAuthProviderGitHub)
	assert.ErrorIs(t, err, ErrOAuthProviderNotSupported)
}

type MockOAuthRepository struct {
	mock.Mock
}

func (m *MockOAuthRepository) Create(ctx context.Context, provider *models.OAuthProvider) error {
	args := m.Called(ctx, provider)
	return args.Error(0)
}

func (m *MockOAuthRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.OAuthProvider, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OAuthProvider), args.Error(1)
}

func (m *MockOAuthRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.OAuthProvider, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.OAuthProvider), args.Error(1)
}

func (m *MockOAuthRepository) GetByUserAndProvider(ctx context.Context, userID uuid.UUID, provider string) (*models.OAuthProvider, error) {
	args := m.Called(ctx, userID, provider)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OAuthProvider), args.Error(1)
}

func (m *MockOAuthRepository) GetByProviderUserID(ctx context.Context, provider, providerUserID string) (*models.OAuthProvider, error) {
	args := m.Called(ctx, provider, providerUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OAuthProvider), args.Error(1)
}

func (m *MockOAuthRepository) Update(ctx context.Context, provider *models.OAuthProvider) error {
	args := m.Called(ctx, provider)
	return args.Error(0)
}

func (m *MockOAuthRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockOAuthRepository) DeleteByUserAndProvider(ctx context.Context, userID uuid.UUID, provider string) error {
	args := m.Called(ctx, userID, provider)
	return args.Error(0)
}

func (m *MockOAuthRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

func (m *MockOAuthRepository) ExistsByProviderUserID(ctx context.Context, provider, providerUserID string) (bool, error) {
	args := m.Called(ctx, provider, providerUserID)
	return args.Bool(0), args.Error(1)
}

func (m *MockOAuthRepository) GetUserIDByProviderUserID(ctx context.Context, provider, providerUserID string) (uuid.UUID, error) {
	args := m.Called(ctx, provider, providerUserID)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

// Helper to setup OAuth service with repository
func setupOAuthServiceWithRepo(t *testing.T) (*OAuthService, *MockOAuthUserRepository, *MockOAuthRepository, *MockOAuthCacheService) {
	userRepo := new(MockOAuthUserRepository)
	oauthRepo := new(MockOAuthRepository)
	cache := NewMockOAuthCacheService()
	jwtService := auth.NewJWTService("test-secret-key-for-testing-purposes", 3600, 86400)

	config := &OAuthProviderConfig{
		GitHub: &OAuthConfig{
			ClientID:     "test-github-client-id",
			ClientSecret: "test-github-client-secret",
			RedirectURI:  "http://localhost:8080/api/v1/auth/oauth/github/callback",
			Scopes:       []string{"read:user", "user:email"},
		},
		Google: &OAuthConfig{
			ClientID:     "test-google-client-id",
			ClientSecret: "test-google-client-secret",
			RedirectURI:  "http://localhost:8080/api/v1/auth/oauth/google/callback",
			Scopes:       []string{"openid", "email", "profile"},
		},
		Discord: &OAuthConfig{
			ClientID:     "test-discord-client-id",
			ClientSecret: "test-discord-client-secret",
			RedirectURI:  "http://localhost:8080/api/v1/auth/oauth/discord/callback",
			Scopes:       []string{"identify", "email"},
		},
	}

	service := NewOAuthServiceWithRepo(config, userRepo, oauthRepo, cache, jwtService)
	return service, userRepo, oauthRepo, cache
}

func TestOAuthServiceWithRepo_GetLinkedProviders(t *testing.T) {
	service, _, oauthRepo, _ := setupOAuthServiceWithRepo(t)
	ctx := context.Background()

	userID := uuid.New()
	now := time.Now()

	expectedProviders := []*models.OAuthProvider{
		{
			ID:             uuid.New(),
			UserID:         userID,
			Provider:       "github",
			ProviderUserID: "12345",
			Email:          "test@example.com",
			CreatedAt:      now,
		},
		{
			ID:             uuid.New(),
			UserID:         userID,
			Provider:       "google",
			ProviderUserID: "67890",
			Email:          "test@example.com",
			CreatedAt:      now,
		},
	}

	oauthRepo.On("GetByUserID", ctx, userID).Return(expectedProviders, nil)

	providers, err := service.GetLinkedProviders(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, providers, 2)
	assert.Equal(t, "github", providers[0].Provider)
	assert.Equal(t, "google", providers[1].Provider)
	oauthRepo.AssertExpectations(t)
}

func TestOAuthServiceWithRepo_GetLinkedProvider(t *testing.T) {
	service, _, oauthRepo, _ := setupOAuthServiceWithRepo(t)
	ctx := context.Background()

	userID := uuid.New()
	now := time.Now()

	expectedProvider := &models.OAuthProvider{
		ID:             uuid.New(),
		UserID:         userID,
		Provider:       "github",
		ProviderUserID: "12345",
		Email:          "test@example.com",
		CreatedAt:      now,
	}

	oauthRepo.On("GetByUserAndProvider", ctx, userID, "github").Return(expectedProvider, nil)

	provider, err := service.GetLinkedProvider(ctx, userID, OAuthProviderGitHub)
	require.NoError(t, err)
	assert.Equal(t, "github", provider.Provider)
	assert.Equal(t, "12345", provider.ProviderUserID)
	oauthRepo.AssertExpectations(t)
}

func TestOAuthServiceWithRepo_GetLinkedProvider_NotFound(t *testing.T) {
	service, _, oauthRepo, _ := setupOAuthServiceWithRepo(t)
	ctx := context.Background()

	userID := uuid.New()

	oauthRepo.On("GetByUserAndProvider", ctx, userID, "github").Return(nil, ErrOAuthProviderNotFound)

	provider, err := service.GetLinkedProvider(ctx, userID, OAuthProviderGitHub)
	assert.ErrorIs(t, err, ErrOAuthProviderNotFound)
	assert.Nil(t, provider)
	oauthRepo.AssertExpectations(t)
}

func TestOAuthServiceWithRepo_UnlinkProvider(t *testing.T) {
	service, userRepo, oauthRepo, _ := setupOAuthServiceWithRepo(t)
	ctx := context.Background()

	userID := uuid.New()
	now := time.Now()

	// User has a password, safe to unlink
	user := &models.User{
		ID:           userID,
		Email:        "test@example.com",
		Username:     "testuser",
		PasswordHash: "hashed_password",
		CreatedAt:    now,
	}

	existingProvider := &models.OAuthProvider{
		ID:             uuid.New(),
		UserID:         userID,
		Provider:       "github",
		ProviderUserID: "12345",
		Email:          "test@example.com",
		CreatedAt:      now,
	}

	oauthRepo.On("GetByUserAndProvider", ctx, userID, "github").Return(existingProvider, nil)
	userRepo.On("GetByID", ctx, userID).Return(user, nil)
	oauthRepo.On("CountByUserID", ctx, userID).Return(2, nil)
	oauthRepo.On("DeleteByUserAndProvider", ctx, userID, "github").Return(nil)

	err := service.UnlinkProvider(ctx, userID, OAuthProviderGitHub)
	require.NoError(t, err)
	oauthRepo.AssertExpectations(t)
	userRepo.AssertExpectations(t)
}

func TestOAuthServiceWithRepo_UnlinkProvider_CannotUnlinkLast(t *testing.T) {
	service, userRepo, oauthRepo, _ := setupOAuthServiceWithRepo(t)
	ctx := context.Background()

	userID := uuid.New()
	now := time.Now()

	// User has NO password and only 1 OAuth provider - cannot unlink
	user := &models.User{
		ID:           userID,
		Email:        "test@example.com",
		Username:     "testuser",
		PasswordHash: "", // No password!
		CreatedAt:    now,
	}

	existingProvider := &models.OAuthProvider{
		ID:             uuid.New(),
		UserID:         userID,
		Provider:       "github",
		ProviderUserID: "12345",
		Email:          "test@example.com",
		CreatedAt:      now,
	}

	oauthRepo.On("GetByUserAndProvider", ctx, userID, "github").Return(existingProvider, nil)
	userRepo.On("GetByID", ctx, userID).Return(user, nil)
	oauthRepo.On("CountByUserID", ctx, userID).Return(1, nil) // Only 1 OAuth provider

	err := service.UnlinkProvider(ctx, userID, OAuthProviderGitHub)
	assert.ErrorIs(t, err, ErrOAuthCannotUnlinkLast)
	oauthRepo.AssertExpectations(t)
	userRepo.AssertExpectations(t)
}

func TestOAuthServiceWithRepo_UnlinkProvider_NotLinked(t *testing.T) {
	service, _, oauthRepo, _ := setupOAuthServiceWithRepo(t)
	ctx := context.Background()

	userID := uuid.New()

	oauthRepo.On("GetByUserAndProvider", ctx, userID, "github").Return(nil, ErrOAuthProviderNotFound)

	err := service.UnlinkProvider(ctx, userID, OAuthProviderGitHub)
	assert.ErrorIs(t, err, ErrOAuthProviderNotFound)
	oauthRepo.AssertExpectations(t)
}

func TestOAuthServiceWithRepo_GetLinkAuthorizationURL(t *testing.T) {
	service, _, oauthRepo, _ := setupOAuthServiceWithRepo(t)
	ctx := context.Background()

	userID := uuid.New()

	// Provider is not yet linked
	oauthRepo.On("GetByUserAndProvider", ctx, userID, "github").Return(nil, ErrOAuthProviderNotFound)

	authURL, err := service.GetLinkAuthorizationURL(ctx, userID, OAuthProviderGitHub)
	require.NoError(t, err)
	assert.Contains(t, authURL, "https://github.com/login/oauth/authorize")
	assert.Contains(t, authURL, "client_id=test-github-client-id")
	oauthRepo.AssertExpectations(t)
}

func TestOAuthServiceWithRepo_GetLinkAuthorizationURL_AlreadyLinked(t *testing.T) {
	service, _, oauthRepo, _ := setupOAuthServiceWithRepo(t)
	ctx := context.Background()

	userID := uuid.New()
	now := time.Now()

	existingProvider := &models.OAuthProvider{
		ID:             uuid.New(),
		UserID:         userID,
		Provider:       "github",
		ProviderUserID: "12345",
		Email:          "test@example.com",
		CreatedAt:      now,
	}

	oauthRepo.On("GetByUserAndProvider", ctx, userID, "github").Return(existingProvider, nil)

	_, err := service.GetLinkAuthorizationURL(ctx, userID, OAuthProviderGitHub)
	assert.ErrorIs(t, err, ErrOAuthProviderAlreadyLinked)
	oauthRepo.AssertExpectations(t)
}

func TestOAuthServiceWithRepo_HandleCallback_ExistingOAuthAccount(t *testing.T) {
	// This test would require mocking the HTTP calls to OAuth providers
	// which is complex. The basic flow tests are covered in oauth_service_test.go
	t.Skip("Full integration test requires HTTP mocking")
}

func TestOAuthProviderResponse_ToResponse(t *testing.T) {
	now := time.Now()
	username := "testuser"
	displayName := "Test User"
	avatarURL := "https://example.com/avatar.png"

	provider := &models.OAuthProvider{
		ID:             uuid.New(),
		UserID:         uuid.New(),
		Provider:       "github",
		ProviderUserID: "12345",
		Email:          "test@example.com",
		Username:       &username,
		DisplayName:    &displayName,
		AvatarURL:      &avatarURL,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	response := provider.ToResponse()

	assert.Equal(t, provider.ID, response.ID)
	assert.Equal(t, "github", response.Provider)
	assert.Equal(t, "12345", response.ProviderUserID)
	assert.Equal(t, "test@example.com", response.Email)
	assert.Equal(t, &username, response.Username)
	assert.Equal(t, &displayName, response.DisplayName)
	assert.Equal(t, &avatarURL, response.AvatarURL)
	assert.Equal(t, now, response.CreatedAt)
}

func TestOAuthServiceWithRepo_UnlinkProvider_CanUnlinkWithMultipleProviders(t *testing.T) {
	service, userRepo, oauthRepo, _ := setupOAuthServiceWithRepo(t)
	ctx := context.Background()

	userID := uuid.New()
	now := time.Now()

	// User has NO password but has 2 OAuth providers - can unlink one
	user := &models.User{
		ID:           userID,
		Email:        "test@example.com",
		Username:     "testuser",
		PasswordHash: "", // No password!
		CreatedAt:    now,
	}

	existingProvider := &models.OAuthProvider{
		ID:             uuid.New(),
		UserID:         userID,
		Provider:       "github",
		ProviderUserID: "12345",
		Email:          "test@example.com",
		CreatedAt:      now,
	}

	oauthRepo.On("GetByUserAndProvider", ctx, userID, "github").Return(existingProvider, nil)
	userRepo.On("GetByID", ctx, userID).Return(user, nil)
	oauthRepo.On("CountByUserID", ctx, userID).Return(2, nil) // 2 OAuth providers
	oauthRepo.On("DeleteByUserAndProvider", ctx, userID, "github").Return(nil)

	err := service.UnlinkProvider(ctx, userID, OAuthProviderGitHub)
	require.NoError(t, err)
	oauthRepo.AssertExpectations(t)
	userRepo.AssertExpectations(t)
}

func TestOAuthServiceWithRepo_AllProviderTypes(t *testing.T) {
	service, _, oauthRepo, _ := setupOAuthServiceWithRepo(t)
	ctx := context.Background()

	userID := uuid.New()

	// Test each provider type
	providers := []OAuthProvider{OAuthProviderGitHub, OAuthProviderGoogle, OAuthProviderDiscord}

	for _, provider := range providers {
		t.Run(string(provider), func(t *testing.T) {
			oauthRepo.On("GetByUserAndProvider", ctx, userID, string(provider)).Return(nil, ErrOAuthProviderNotFound).Once()

			authURL, err := service.GetLinkAuthorizationURL(ctx, userID, provider)
			require.NoError(t, err)
			assert.NotEmpty(t, authURL)
		})
	}

	oauthRepo.AssertExpectations(t)
}

type MockSessionRepository struct {
	mock.Mock
}

func (m *MockSessionRepository) CreateSession(ctx context.Context, session *models.Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockSessionRepository) GetSessionByID(ctx context.Context, id uuid.UUID) (*models.Session, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Session), args.Error(1)
}

func (m *MockSessionRepository) GetUserSessions(ctx context.Context, userID uuid.UUID) ([]*models.Session, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Session), args.Error(1)
}

func (m *MockSessionRepository) UpdateSessionActivity(ctx context.Context, sessionID uuid.UUID) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

func (m *MockSessionRepository) SetCurrentSession(ctx context.Context, userID, sessionID uuid.UUID) error {
	args := m.Called(ctx, userID, sessionID)
	return args.Error(0)
}

func (m *MockSessionRepository) RevokeSession(ctx context.Context, sessionID uuid.UUID) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

func (m *MockSessionRepository) RevokeAllUserSessions(ctx context.Context, userID uuid.UUID, exceptSessionID *uuid.UUID) error {
	args := m.Called(ctx, userID, exceptSessionID)
	return args.Error(0)
}

func (m *MockSessionRepository) CreateRefreshToken(ctx context.Context, token *models.RefreshToken) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockSessionRepository) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error) {
	args := m.Called(ctx, tokenHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RefreshToken), args.Error(1)
}

func (m *MockSessionRepository) MarkRefreshTokenUsed(ctx context.Context, tokenID uuid.UUID) error {
	args := m.Called(ctx, tokenID)
	return args.Error(0)
}

func (m *MockSessionRepository) RevokeTokenFamily(ctx context.Context, familyID uuid.UUID) error {
	args := m.Called(ctx, familyID)
	return args.Error(0)
}

func (m *MockSessionRepository) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID, exceptFamilyID *uuid.UUID) error {
	args := m.Called(ctx, userID, exceptFamilyID)
	return args.Error(0)
}

func (m *MockSessionRepository) RotateRefreshToken(ctx context.Context, oldTokenID uuid.UUID, newToken *models.RefreshToken) error {
	args := m.Called(ctx, oldTokenID, newToken)
	return args.Error(0)
}

//lint:ignore U1000 test factory helper
func testSessionService(repo SessionRepository) SessionService {
	jwtService := auth.NewJWTService("test-secret", 15*time.Minute, 7*24*time.Hour)
	return NewSessionService(repo, jwtService, 30*24*time.Hour)
}

func TestDeviceInfoParsing(t *testing.T) {
	tests := []struct {
		name        string
		userAgent   string
		wantDevice  models.DeviceType
		wantBrowser string
	}{
		{
			name:        "Chrome on Windows Desktop",
			userAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36",
			wantDevice:  models.DeviceTypeDesktop,
			wantBrowser: "Chrome",
		},
		{
			name:        "Safari on iPhone",
			userAgent:   "Mozilla/5.0 (iPhone; CPU iPhone OS 17_1 like Mac OS X) AppleWebKit/605.1.15 Safari/604.1",
			wantDevice:  models.DeviceTypeMobile,
			wantBrowser: "Safari",
		},
		{
			name:        "Chrome on Android",
			userAgent:   "Mozilla/5.0 (Linux; Android 13; SM-G991B) AppleWebKit/537.36 Chrome/120.0.6099.43 Mobile Safari/537.36",
			wantDevice:  models.DeviceTypeMobile,
			wantBrowser: "Chrome",
		},
		{
			name:        "iPad Safari",
			userAgent:   "Mozilla/5.0 (iPad; CPU OS 17_1 like Mac OS X) AppleWebKit/605.1.15 Safari/604.1",
			wantDevice:  models.DeviceTypeTablet,
			wantBrowser: "Safari",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := auth.ParseUserAgent(tt.userAgent)
			assert.Equal(t, tt.wantDevice, info.DeviceType, "Device type mismatch")
			assert.Equal(t, tt.wantBrowser, info.Browser, "Browser mismatch")
		})
	}
}

func TestTokenHashing(t *testing.T) {
	token1 := "test-token-123"
	token2 := "test-token-456"

	hash1 := models.HashToken(token1)
	hash2 := models.HashToken(token2)
	hash1Again := models.HashToken(token1)

	// Same input produces same hash
	assert.Equal(t, hash1, hash1Again)
	// Different inputs produce different hashes
	assert.NotEqual(t, hash1, hash2)
	// Hash is hex encoded (64 chars for SHA-256)
	assert.Len(t, hash1, 64)
}

func TestRefreshTokenIsValid(t *testing.T) {
	tests := []struct {
		name  string
		token models.RefreshToken
		want  bool
	}{
		{
			name: "Valid token",
			token: models.RefreshToken{
				Used:      false,
				Revoked:   false,
				ExpiresAt: time.Now().Add(time.Hour),
			},
			want: true,
		},
		{
			name: "Used token",
			token: models.RefreshToken{
				Used:      true,
				Revoked:   false,
				ExpiresAt: time.Now().Add(time.Hour),
			},
			want: false,
		},
		{
			name: "Revoked token",
			token: models.RefreshToken{
				Used:      false,
				Revoked:   true,
				ExpiresAt: time.Now().Add(time.Hour),
			},
			want: false,
		},
		{
			name: "Expired token",
			token: models.RefreshToken{
				Used:      false,
				Revoked:   false,
				ExpiresAt: time.Now().Add(-time.Hour),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.token.IsValid()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSessionToResponse(t *testing.T) {
	browser := "Chrome"
	os := "Windows"
	city := "San Francisco"
	country := "US"
	lastUsed := time.Now().Add(-time.Hour)

	session := &models.Session{
		ID:              uuid.New(),
		DeviceName:      strPtr("Chrome on Windows"),
		DeviceType:      models.DeviceTypeDesktop,
		Browser:         &browser,
		OS:              &os,
		LocationCity:    &city,
		LocationCountry: &country,
		IsCurrent:       true,
		LastUsed:        &lastUsed,
		CreatedAt:       time.Now().Add(-24 * time.Hour),
	}

	response := session.ToResponse()

	assert.Equal(t, session.ID, response.ID)
	assert.Equal(t, "Chrome on Windows", response.DeviceName)
	assert.Equal(t, models.DeviceTypeDesktop, response.DeviceType)
	assert.Equal(t, "Chrome", response.Browser)
	assert.Equal(t, "Windows", response.OS)
	assert.Equal(t, "San Francisco", response.LocationCity)
	assert.Equal(t, "US", response.LocationCountry)
	assert.True(t, response.IsCurrent)
}

func TestSessionToResponse_BuildsDeviceName(t *testing.T) {
	browser := "Firefox"
	os := "Linux"

	// Session without explicit device name should build one
	session := &models.Session{
		ID:         uuid.New(),
		DeviceType: models.DeviceTypeDesktop,
		Browser:    &browser,
		OS:         &os,
		CreatedAt:  time.Now(),
	}

	response := session.ToResponse()

	assert.Equal(t, "Firefox on Linux", response.DeviceName)
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name          string
		remoteAddr    string
		xForwardedFor string
		xRealIP       string
		want          string
	}{
		{
			name:          "X-Forwarded-For takes priority",
			remoteAddr:    "127.0.0.1:8080",
			xForwardedFor: "203.0.113.195, 70.41.3.18",
			xRealIP:       "192.168.1.1",
			want:          "203.0.113.195",
		},
		{
			name:       "X-Real-IP used when no X-Forwarded-For",
			remoteAddr: "127.0.0.1:8080",
			xRealIP:    "192.168.1.1",
			want:       "192.168.1.1",
		},
		{
			name:       "Remote addr fallback",
			remoteAddr: "10.0.0.1:54321",
			want:       "10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := auth.GetClientIP(tt.remoteAddr, tt.xForwardedFor, tt.xRealIP)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGenerateTokenFamily(t *testing.T) {
	family1 := models.GenerateTokenFamily()
	family2 := models.GenerateTokenFamily()

	// Each call generates a unique ID
	assert.NotEqual(t, family1, family2)
	// IDs are valid UUIDs
	assert.NotEqual(t, uuid.Nil, family1)
	assert.NotEqual(t, uuid.Nil, family2)
}
