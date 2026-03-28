package services

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
	"hearth/internal/models"
)

// MockOAuthAppRepository mocks the OAuthAppRepository interface
type MockOAuthAppRepository struct {
	mock.Mock
}

func (m *MockOAuthAppRepository) CreateApp(ctx context.Context, app *models.OAuthApp) error {
	args := m.Called(ctx, app)
	return args.Error(0)
}

func (m *MockOAuthAppRepository) GetAppByID(ctx context.Context, id uuid.UUID) (*models.OAuthApp, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OAuthApp), args.Error(1)
}

func (m *MockOAuthAppRepository) GetAppByClientID(ctx context.Context, clientID string) (*models.OAuthApp, error) {
	args := m.Called(ctx, clientID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OAuthApp), args.Error(1)
}

func (m *MockOAuthAppRepository) GetAppsByOwner(ctx context.Context, ownerID uuid.UUID) ([]*models.OAuthApp, error) {
	args := m.Called(ctx, ownerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.OAuthApp), args.Error(1)
}

func (m *MockOAuthAppRepository) UpdateApp(ctx context.Context, app *models.OAuthApp) error {
	args := m.Called(ctx, app)
	return args.Error(0)
}

func (m *MockOAuthAppRepository) DeleteApp(ctx context.Context, id, ownerID uuid.UUID) error {
	args := m.Called(ctx, id, ownerID)
	return args.Error(0)
}

func (m *MockOAuthAppRepository) RegenerateSecret(ctx context.Context, id, ownerID uuid.UUID, newSecretHash string) error {
	args := m.Called(ctx, id, ownerID, newSecretHash)
	return args.Error(0)
}

func (m *MockOAuthAppRepository) CreateAuthorizationCode(ctx context.Context, code *models.OAuthAuthorizationCode) error {
	args := m.Called(ctx, code)
	return args.Error(0)
}

func (m *MockOAuthAppRepository) GetAuthorizationCode(ctx context.Context, codeHash string) (*models.OAuthAuthorizationCode, error) {
	args := m.Called(ctx, codeHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OAuthAuthorizationCode), args.Error(1)
}

func (m *MockOAuthAppRepository) MarkAuthorizationCodeUsed(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockOAuthAppRepository) CleanupExpiredAuthCodes(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockOAuthAppRepository) CreateAccessToken(ctx context.Context, token *models.OAuthAccessToken) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockOAuthAppRepository) GetAccessTokenByHash(ctx context.Context, tokenHash string) (*models.OAuthAccessToken, error) {
	args := m.Called(ctx, tokenHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OAuthAccessToken), args.Error(1)
}

func (m *MockOAuthAppRepository) RevokeAccessToken(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockOAuthAppRepository) RevokeAccessTokensByUser(ctx context.Context, userID uuid.UUID, clientID string) error {
	args := m.Called(ctx, userID, clientID)
	return args.Error(0)
}

func (m *MockOAuthAppRepository) CreateRefreshToken(ctx context.Context, token *models.OAuthRefreshToken) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockOAuthAppRepository) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*models.OAuthRefreshToken, error) {
	args := m.Called(ctx, tokenHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OAuthRefreshToken), args.Error(1)
}

func (m *MockOAuthAppRepository) RotateRefreshToken(ctx context.Context, oldID, newID uuid.UUID) error {
	args := m.Called(ctx, oldID, newID)
	return args.Error(0)
}

func (m *MockOAuthAppRepository) RevokeRefreshToken(ctx context.Context, id uuid.UUID, reason string) error {
	args := m.Called(ctx, id, reason)
	return args.Error(0)
}

func (m *MockOAuthAppRepository) RevokeRefreshTokenFamily(ctx context.Context, accessTokenID uuid.UUID, reason string) error {
	args := m.Called(ctx, accessTokenID, reason)
	return args.Error(0)
}

func (m *MockOAuthAppRepository) RevokeRefreshTokensByUser(ctx context.Context, userID uuid.UUID, clientID string) error {
	args := m.Called(ctx, userID, clientID)
	return args.Error(0)
}

func (m *MockOAuthAppRepository) CreateOrUpdateUserAuthorization(ctx context.Context, auth *models.OAuthUserAuthorization) error {
	args := m.Called(ctx, auth)
	return args.Error(0)
}

func (m *MockOAuthAppRepository) GetUserAuthorization(ctx context.Context, userID uuid.UUID, clientID string) (*models.OAuthUserAuthorization, error) {
	args := m.Called(ctx, userID, clientID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OAuthUserAuthorization), args.Error(1)
}

func (m *MockOAuthAppRepository) GetUserAuthorizations(ctx context.Context, userID uuid.UUID) ([]*models.OAuthUserAuthorization, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.OAuthUserAuthorization), args.Error(1)
}

func (m *MockOAuthAppRepository) RevokeUserAuthorization(ctx context.Context, userID uuid.UUID, clientID string) error {
	args := m.Called(ctx, userID, clientID)
	return args.Error(0)
}

func (m *MockOAuthAppRepository) UpdateLastUsed(ctx context.Context, userID uuid.UUID, clientID string) error {
	args := m.Called(ctx, userID, clientID)
	return args.Error(0)
}

// Helper functions for tests
func testStrPtr(s string) *string {
	return &s
}

func testSha256Hash(data string) string {
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// Test CreateApp
func TestOAuthProviderService_CreateApp(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockOAuthAppRepository)
	service := NewOAuthProviderService(mockRepo, nil, nil)

	ownerID := uuid.New()

	t.Run("successful app creation", func(t *testing.T) {
		mockRepo.On("CreateApp", ctx, mock.AnythingOfType("*models.OAuthApp")).Return(nil).Once()

		req := &CreateAppRequest{
			Name:         "Test App",
			Description:  testStrPtr("A test application"),
			RedirectURIs: []string{"https://example.com/callback"},
			Scopes:       []string{"read", "profile"},
			IsPublic:     false,
		}

		result, err := service.CreateApp(ctx, ownerID, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.ClientSecret)
		assert.Equal(t, "Test App", result.App.Name)
		assert.Equal(t, ownerID, result.App.OwnerID)
		assert.True(t, result.App.IsActive)
		mockRepo.AssertExpectations(t)
	})

	t.Run("invalid name length", func(t *testing.T) {
		req := &CreateAppRequest{
			Name:         "A", // Too short
			RedirectURIs: []string{"https://example.com/callback"},
			Scopes:       []string{"read"},
		}

		result, err := service.CreateApp(ctx, ownerID, req)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "between 2 and 100")
	})

	t.Run("invalid scope", func(t *testing.T) {
		req := &CreateAppRequest{
			Name:         "Test App",
			RedirectURIs: []string{"https://example.com/callback"},
			Scopes:       []string{"invalid_scope"},
		}

		result, err := service.CreateApp(ctx, ownerID, req)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid scope")
	})
}

// Test ValidateAuthorization
func TestOAuthProviderService_ValidateAuthorization(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockOAuthAppRepository)
	service := NewOAuthProviderService(mockRepo, nil, nil)

	userID := uuid.New()
	clientID := "test-client-id"

	app := &models.OAuthApp{
		ID:           uuid.New(),
		ClientID:     clientID,
		Name:         "Test App",
		RedirectURIs: []string{"https://example.com/callback"},
		Scopes:       []string{"read", "profile"},
		IsActive:     true,
		IsPublic:     false,
	}

	t.Run("valid authorization request", func(t *testing.T) {
		mockRepo.On("GetAppByClientID", ctx, clientID).Return(app, nil).Once()
		mockRepo.On("GetUserAuthorization", ctx, userID, clientID).Return(nil, nil).Once()

		req := &AuthorizeRequest{
			ClientID:     clientID,
			RedirectURI:  "https://example.com/callback",
			ResponseType: "code",
			Scope:        "read profile",
		}

		result, err := service.ValidateAuthorization(ctx, userID, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.RequiresConsent)
		assert.Equal(t, app.Name, result.App.Name)
		assert.Len(t, result.RequestedScopes, 2)
		mockRepo.AssertExpectations(t)
	})

	t.Run("invalid redirect URI", func(t *testing.T) {
		mockRepo.On("GetAppByClientID", ctx, clientID).Return(app, nil).Once()

		req := &AuthorizeRequest{
			ClientID:     clientID,
			RedirectURI:  "https://evil.com/callback", // Not in allowed list
			ResponseType: "code",
			Scope:        "read",
		}

		result, err := service.ValidateAuthorization(ctx, userID, req)

		assert.Error(t, err)
		assert.Equal(t, ErrOAuthInvalidRedirectURI, err)
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("PKCE required for public client", func(t *testing.T) {
		publicApp := *app
		publicApp.IsPublic = true
		mockRepo.On("GetAppByClientID", ctx, clientID).Return(&publicApp, nil).Once()

		req := &AuthorizeRequest{
			ClientID:     clientID,
			RedirectURI:  "https://example.com/callback",
			ResponseType: "code",
			Scope:        "read",
			// No CodeChallenge provided
		}

		result, err := service.ValidateAuthorization(ctx, userID, req)

		assert.Error(t, err)
		assert.Equal(t, ErrOAuthPKCERequired, err)
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
	})
}

// Test ExchangeToken - Authorization Code Flow
func TestOAuthProviderService_ExchangeToken_AuthorizationCode(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockOAuthAppRepository)
	service := NewOAuthProviderService(mockRepo, nil, nil)

	clientID := "test-client-id"
	clientSecret := "test-secret"
	secretHash, _ := bcrypt.GenerateFromPassword([]byte(clientSecret), bcrypt.DefaultCost)

	app := &models.OAuthApp{
		ID:           uuid.New(),
		ClientID:     clientID,
		ClientSecret: string(secretHash),
		Name:         "Test App",
		RedirectURIs: []string{"https://example.com/callback"},
		Scopes:       []string{"read"},
		IsActive:     true,
		IsPublic:     false,
	}

	authCode := "test-auth-code"
	codeHash := testSha256Hash(authCode)
	userID := uuid.New()

	authCodeRecord := &models.OAuthAuthorizationCode{
		ID:          uuid.New(),
		Code:        codeHash,
		ClientID:    clientID,
		UserID:      userID,
		Scopes:      []string{"read"},
		RedirectURI: "https://example.com/callback",
		ExpiresAt:   time.Now().Add(10 * time.Minute),
		Used:        false,
		CreatedAt:   time.Now(),
	}

	t.Run("successful code exchange", func(t *testing.T) {
		mockRepo.On("GetAppByClientID", ctx, clientID).Return(app, nil).Once()
		mockRepo.On("GetAuthorizationCode", ctx, codeHash).Return(authCodeRecord, nil).Once()
		mockRepo.On("MarkAuthorizationCodeUsed", ctx, authCodeRecord.ID).Return(nil).Once()
		mockRepo.On("CreateAccessToken", ctx, mock.AnythingOfType("*models.OAuthAccessToken")).Return(nil).Once()
		mockRepo.On("CreateRefreshToken", ctx, mock.AnythingOfType("*models.OAuthRefreshToken")).Return(nil).Once()

		redirectURI := "https://example.com/callback"
		req := &TokenRequest{
			GrantType:    "authorization_code",
			Code:         &authCode,
			RedirectURI:  &redirectURI,
			ClientID:     clientID,
			ClientSecret: &clientSecret,
		}

		result, err := service.ExchangeToken(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.AccessToken)
		assert.NotEmpty(t, result.RefreshToken)
		assert.Equal(t, "Bearer", result.TokenType)
		assert.Equal(t, "read", result.Scope)
		mockRepo.AssertExpectations(t)
	})

	t.Run("invalid client secret", func(t *testing.T) {
		mockRepo.On("GetAppByClientID", ctx, clientID).Return(app, nil).Once()

		wrongSecret := "wrong-secret"
		redirectURI := "https://example.com/callback"
		req := &TokenRequest{
			GrantType:    "authorization_code",
			Code:         &authCode,
			RedirectURI:  &redirectURI,
			ClientID:     clientID,
			ClientSecret: &wrongSecret,
		}

		result, err := service.ExchangeToken(ctx, req)

		assert.Error(t, err)
		assert.Equal(t, ErrOAuthInvalidClientSecret, err)
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("expired code", func(t *testing.T) {
		expiredCode := *authCodeRecord
		expiredCode.ExpiresAt = time.Now().Add(-1 * time.Minute)

		mockRepo.On("GetAppByClientID", ctx, clientID).Return(app, nil).Once()
		mockRepo.On("GetAuthorizationCode", ctx, codeHash).Return(&expiredCode, nil).Once()

		redirectURI := "https://example.com/callback"
		req := &TokenRequest{
			GrantType:    "authorization_code",
			Code:         &authCode,
			RedirectURI:  &redirectURI,
			ClientID:     clientID,
			ClientSecret: &clientSecret,
		}

		result, err := service.ExchangeToken(ctx, req)

		assert.Error(t, err)
		assert.Equal(t, ErrOAuthCodeExpired, err)
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
	})
}

// Test ExchangeToken - Refresh Token Flow with Rotation
func TestOAuthProviderService_ExchangeToken_RefreshToken(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockOAuthAppRepository)
	service := NewOAuthProviderService(mockRepo, nil, nil)

	clientID := "test-client-id"
	clientSecret := "test-secret"
	secretHash, _ := bcrypt.GenerateFromPassword([]byte(clientSecret), bcrypt.DefaultCost)

	app := &models.OAuthApp{
		ID:           uuid.New(),
		ClientID:     clientID,
		ClientSecret: string(secretHash),
		Name:         "Test App",
		IsActive:     true,
		IsPublic:     false,
	}

	refreshToken := "test-refresh-token"
	tokenHash := testSha256Hash(refreshToken)
	userID := uuid.New()
	accessTokenID := uuid.New()

	refreshTokenRecord := &models.OAuthRefreshToken{
		ID:            uuid.New(),
		TokenHash:     tokenHash,
		AccessTokenID: accessTokenID,
		ClientID:      clientID,
		UserID:        userID,
		Scopes:        []string{"read"},
		ExpiresAt:     time.Now().Add(30 * 24 * time.Hour),
		CreatedAt:     time.Now(),
	}

	t.Run("successful refresh with rotation", func(t *testing.T) {
		mockRepo.On("GetAppByClientID", ctx, clientID).Return(app, nil).Once()
		mockRepo.On("GetRefreshTokenByHash", ctx, tokenHash).Return(refreshTokenRecord, nil).Once()
		mockRepo.On("UpdateLastUsed", ctx, userID, clientID).Return(nil).Once()
		mockRepo.On("CreateAccessToken", ctx, mock.AnythingOfType("*models.OAuthAccessToken")).Return(nil).Once()
		mockRepo.On("CreateRefreshToken", ctx, mock.AnythingOfType("*models.OAuthRefreshToken")).Return(nil).Once()
		mockRepo.On("RotateRefreshToken", ctx, refreshTokenRecord.ID, mock.AnythingOfType("uuid.UUID")).Return(nil).Once()

		req := &TokenRequest{
			GrantType:    "refresh_token",
			RefreshToken: &refreshToken,
			ClientID:     clientID,
			ClientSecret: &clientSecret,
		}

		result, err := service.ExchangeToken(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.AccessToken)
		assert.NotEmpty(t, result.RefreshToken)
		assert.NotEqual(t, refreshToken, result.RefreshToken) // New refresh token
		mockRepo.AssertExpectations(t)
	})

	t.Run("reuse detection - rotated token", func(t *testing.T) {
		rotatedAt := time.Now().Add(-1 * time.Hour)
		rotatedToken := *refreshTokenRecord
		rotatedToken.RotatedAt = &rotatedAt

		mockRepo.On("GetAppByClientID", ctx, clientID).Return(app, nil).Once()
		mockRepo.On("GetRefreshTokenByHash", ctx, tokenHash).Return(&rotatedToken, nil).Once()
		mockRepo.On("RevokeRefreshTokenFamily", ctx, accessTokenID, "reuse_detected").Return(nil).Once()
		mockRepo.On("RevokeAccessToken", ctx, accessTokenID).Return(nil).Once()

		req := &TokenRequest{
			GrantType:    "refresh_token",
			RefreshToken: &refreshToken,
			ClientID:     clientID,
			ClientSecret: &clientSecret,
		}

		result, err := service.ExchangeToken(ctx, req)

		assert.Error(t, err)
		assert.Equal(t, ErrOAuthRefreshTokenReused, err)
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
	})
}

// Test ValidateAccessToken
func TestOAuthProviderService_ValidateAccessToken(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockOAuthAppRepository)
	service := NewOAuthProviderService(mockRepo, nil, nil)

	accessToken := "test-access-token"
	tokenHash := testSha256Hash(accessToken)
	userID := uuid.New()
	clientID := "test-client"

	t.Run("valid token", func(t *testing.T) {
		tokenRecord := &models.OAuthAccessToken{
			ID:        uuid.New(),
			TokenHash: tokenHash,
			ClientID:  clientID,
			UserID:    userID,
			Scopes:    []string{"read", "profile"},
			ExpiresAt: time.Now().Add(1 * time.Hour),
			CreatedAt: time.Now(),
		}

		mockRepo.On("GetAccessTokenByHash", ctx, tokenHash).Return(tokenRecord, nil).Once()

		result, err := service.ValidateAccessToken(ctx, accessToken)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, userID, result.UserID)
		assert.Equal(t, clientID, result.ClientID)
		assert.True(t, result.HasScope("read"))
		assert.True(t, result.HasScope("profile"))
		assert.False(t, result.HasScope("admin"))
		mockRepo.AssertExpectations(t)
	})

	t.Run("expired token", func(t *testing.T) {
		// GetAccessTokenByHash returns nil for expired tokens
		mockRepo.On("GetAccessTokenByHash", ctx, tokenHash).Return(nil, nil).Once()

		result, err := service.ValidateAccessToken(ctx, accessToken)

		assert.Error(t, err)
		assert.Equal(t, ErrOAuthAccessTokenNotFound, err)
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("revoked token", func(t *testing.T) {
		// GetAccessTokenByHash returns nil for revoked tokens
		mockRepo.On("GetAccessTokenByHash", ctx, tokenHash).Return(nil, nil).Once()

		result, err := service.ValidateAccessToken(ctx, accessToken)

		assert.Error(t, err)
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
	})
}

// Test PKCE verification
func TestPKCEVerification(t *testing.T) {
	t.Run("plain method", func(t *testing.T) {
		verifier := "test-verifier-12345678901234567890123456"
		challenge := verifier
		method := "plain"

		assert.True(t, verifyPKCE(challenge, &method, verifier))
		assert.False(t, verifyPKCE(challenge, &method, "wrong-verifier"))
	})

	t.Run("S256 method", func(t *testing.T) {
		verifier := "test-verifier-12345678901234567890123456"
		hash := sha256.Sum256([]byte(verifier))
		challenge := base64.RawURLEncoding.EncodeToString(hash[:])
		method := "S256"

		assert.True(t, verifyPKCE(challenge, &method, verifier))
		assert.False(t, verifyPKCE(challenge, &method, "wrong-verifier"))
	})

	t.Run("nil method defaults to plain", func(t *testing.T) {
		verifier := "test-verifier"
		challenge := verifier

		assert.True(t, verifyPKCE(challenge, nil, verifier))
	})
}

// Test scope utilities
func TestScopeUtilities(t *testing.T) {
	t.Run("parseScopes", func(t *testing.T) {
		assert.Equal(t, []string{"read", "write", "admin"}, parseScopes("read write admin"))
		assert.Equal(t, []string{"read"}, parseScopes("read"))
		assert.Equal(t, []string{}, parseScopes(""))
	})

	t.Run("containsScope", func(t *testing.T) {
		scopes := []string{"read", "write"}
		assert.True(t, containsScope(scopes, "read"))
		assert.True(t, containsScope(scopes, "write"))
		assert.False(t, containsScope(scopes, "admin"))
	})

	t.Run("scopesContained", func(t *testing.T) {
		existing := []string{"read", "write", "admin"}
		assert.True(t, scopesContained(existing, []string{"read"}))
		assert.True(t, scopesContained(existing, []string{"read", "write"}))
		assert.False(t, scopesContained(existing, []string{"read", "delete"}))
	})
}

// Test RevokeUserAuthorization
func TestOAuthProviderService_RevokeUserAuthorization(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockOAuthAppRepository)
	service := NewOAuthProviderService(mockRepo, nil, nil)

	userID := uuid.New()
	clientID := "test-client"

	t.Run("successful revocation", func(t *testing.T) {
		mockRepo.On("RevokeAccessTokensByUser", ctx, userID, clientID).Return(nil).Once()
		mockRepo.On("RevokeRefreshTokensByUser", ctx, userID, clientID).Return(nil).Once()
		mockRepo.On("RevokeUserAuthorization", ctx, userID, clientID).Return(nil).Once()

		err := service.RevokeUserAuthorization(ctx, userID, clientID)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})
}
