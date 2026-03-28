package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"hearth/internal/auth"
	"hearth/internal/models"
)

// MockOAuthRepository is a mock implementation of OAuthRepository
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
