package services

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
	"hearth/internal/models"
)

// ========== Token Refresh During Expiry Edge Cases ==========

func TestOAuthProviderService_RefreshToken_ExpiredDuringRefresh(t *testing.T) {
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
		IsActive:     true,
		IsPublic:     false,
	}

	refreshToken := "expiring-refresh-token"
	tokenHash := testSha256Hash(refreshToken)
	userID := uuid.New()

	// Token that expires during the request (edge case)
	refreshTokenRecord := &models.OAuthRefreshToken{
		ID:            uuid.New(),
		TokenHash:     tokenHash,
		AccessTokenID: uuid.New(),
		ClientID:      clientID,
		UserID:        userID,
		Scopes:        []string{"read"},
		ExpiresAt:     time.Now().Add(-1 * time.Second), // Just expired!
		CreatedAt:     time.Now().Add(-30 * 24 * time.Hour),
	}

	mockRepo.On("GetAppByClientID", ctx, clientID).Return(app, nil)
	mockRepo.On("GetRefreshTokenByHash", ctx, tokenHash).Return(refreshTokenRecord, nil)

	req := &TokenRequest{
		GrantType:    "refresh_token",
		RefreshToken: &refreshToken,
		ClientID:     clientID,
		ClientSecret: &clientSecret,
	}

	result, err := service.ExchangeToken(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, ErrOAuthRefreshTokenExpired, err)
	assert.Nil(t, result)
	mockRepo.AssertExpectations(t)
}

func TestOAuthProviderService_RefreshToken_RevokedBeforeUse(t *testing.T) {
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
		IsActive:     true,
		IsPublic:     false,
	}

	refreshToken := "revoked-refresh-token"
	tokenHash := testSha256Hash(refreshToken)
	userID := uuid.New()
	revokedAt := time.Now().Add(-1 * time.Hour)

	// Token that was revoked
	refreshTokenRecord := &models.OAuthRefreshToken{
		ID:            uuid.New(),
		TokenHash:     tokenHash,
		AccessTokenID: uuid.New(),
		ClientID:      clientID,
		UserID:        userID,
		Scopes:        []string{"read"},
		ExpiresAt:     time.Now().Add(24 * time.Hour),
		RevokedAt:     &revokedAt,
		RevokedReason: testStrPtr("user_revoked"),
		CreatedAt:     time.Now().Add(-1 * time.Hour),
	}

	mockRepo.On("GetAppByClientID", ctx, clientID).Return(app, nil)
	mockRepo.On("GetRefreshTokenByHash", ctx, tokenHash).Return(refreshTokenRecord, nil)

	req := &TokenRequest{
		GrantType:    "refresh_token",
		RefreshToken: &refreshToken,
		ClientID:     clientID,
		ClientSecret: &clientSecret,
	}

	result, err := service.ExchangeToken(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, ErrOAuthRefreshTokenRevoked, err)
	assert.Nil(t, result)
	mockRepo.AssertExpectations(t)
}

// ========== Token Reuse Detection ==========

func TestOAuthProviderService_RefreshToken_ReuseDetection_RevokesFamily(t *testing.T) {
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
		IsActive:     true,
		IsPublic:     false,
	}

	refreshToken := "rotated-refresh-token"
	tokenHash := testSha256Hash(refreshToken)
	userID := uuid.New()
	accessTokenID := uuid.New()
	rotatedAt := time.Now().Add(-5 * time.Minute)
	rotatedToID := uuid.New()

	// Token that was already rotated (reuse attempt)
	refreshTokenRecord := &models.OAuthRefreshToken{
		ID:            uuid.New(),
		TokenHash:     tokenHash,
		AccessTokenID: accessTokenID,
		ClientID:      clientID,
		UserID:        userID,
		Scopes:        []string{"read"},
		ExpiresAt:     time.Now().Add(24 * time.Hour),
		RotatedAt:     &rotatedAt,
		RotatedToID:   &rotatedToID,
		CreatedAt:     time.Now().Add(-1 * time.Hour),
	}

	mockRepo.On("GetAppByClientID", ctx, clientID).Return(app, nil)
	mockRepo.On("GetRefreshTokenByHash", ctx, tokenHash).Return(refreshTokenRecord, nil)
	// Should revoke entire token family
	mockRepo.On("RevokeRefreshTokenFamily", ctx, accessTokenID, "reuse_detected").Return(nil)
	mockRepo.On("RevokeAccessToken", ctx, accessTokenID).Return(nil)

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
	// Verify revocation was called
	mockRepo.AssertCalled(t, "RevokeRefreshTokenFamily", ctx, accessTokenID, "reuse_detected")
	mockRepo.AssertCalled(t, "RevokeAccessToken", ctx, accessTokenID)
}

// ========== PKCE Edge Cases ==========

func TestOAuthProviderService_PKCE_S256_Validation(t *testing.T) {
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"

	// Calculate expected challenge
	hash := sha256.Sum256([]byte(verifier))
	expectedChallenge := base64.RawURLEncoding.EncodeToString(hash[:])

	method := "S256"

	// Valid verifier
	assert.True(t, verifyPKCE(expectedChallenge, &method, verifier))

	// Invalid verifier
	assert.False(t, verifyPKCE(expectedChallenge, &method, "wrong-verifier"))

	// Tampered challenge
	assert.False(t, verifyPKCE("tampered-challenge", &method, verifier))
}

func TestOAuthProviderService_PKCE_Plain_Validation(t *testing.T) {
	verifier := "test-code-verifier-12345"
	method := "plain"

	// Valid: challenge equals verifier
	assert.True(t, verifyPKCE(verifier, &method, verifier))

	// Invalid: different verifier
	assert.False(t, verifyPKCE(verifier, &method, "different-verifier"))
}

func TestOAuthProviderService_PKCE_NoMethod_DefaultsToPlain(t *testing.T) {
	verifier := "test-code-verifier"

	// nil method should default to plain
	assert.True(t, verifyPKCE(verifier, nil, verifier))
	assert.False(t, verifyPKCE(verifier, nil, "different"))
}

func TestOAuthProviderService_PKCE_InvalidMethod(t *testing.T) {
	verifier := "test-code-verifier"
	method := "invalid-method"

	// Invalid method should fail
	assert.False(t, verifyPKCE(verifier, &method, verifier))
}

// ========== Authorization Code Edge Cases ==========

func TestOAuthProviderService_AuthCode_AlreadyUsed(t *testing.T) {
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
		IsActive:     true,
		IsPublic:     false,
	}

	authCode := "used-auth-code"
	codeHash := testSha256Hash(authCode)
	userID := uuid.New()

	// Code that was already used
	authCodeRecord := &models.OAuthAuthorizationCode{
		ID:          uuid.New(),
		Code:        codeHash,
		ClientID:    clientID,
		UserID:      userID,
		Scopes:      []string{"read"},
		RedirectURI: "https://example.com/callback",
		ExpiresAt:   time.Now().Add(10 * time.Minute),
		Used:        true, // Already used!
		CreatedAt:   time.Now(),
	}

	mockRepo.On("GetAppByClientID", ctx, clientID).Return(app, nil)
	mockRepo.On("GetAuthorizationCode", ctx, codeHash).Return(authCodeRecord, nil)

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
	assert.Equal(t, ErrOAuthCodeAlreadyUsed, err)
	assert.Nil(t, result)
	mockRepo.AssertExpectations(t)
}

func TestOAuthProviderService_AuthCode_WrongClient(t *testing.T) {
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
		IsActive:     true,
		IsPublic:     false,
	}

	authCode := "test-auth-code"
	codeHash := testSha256Hash(authCode)
	userID := uuid.New()

	// Code issued to a different client
	authCodeRecord := &models.OAuthAuthorizationCode{
		ID:          uuid.New(),
		Code:        codeHash,
		ClientID:    "different-client-id", // Wrong client!
		UserID:      userID,
		Scopes:      []string{"read"},
		RedirectURI: "https://example.com/callback",
		ExpiresAt:   time.Now().Add(10 * time.Minute),
		Used:        false,
		CreatedAt:   time.Now(),
	}

	mockRepo.On("GetAppByClientID", ctx, clientID).Return(app, nil)
	mockRepo.On("GetAuthorizationCode", ctx, codeHash).Return(authCodeRecord, nil)

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
	assert.Equal(t, ErrOAuthCodeNotFound, err)
	assert.Nil(t, result)
	mockRepo.AssertExpectations(t)
}

func TestOAuthProviderService_AuthCode_WrongRedirectURI(t *testing.T) {
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
		RedirectURI: "https://example.com/callback", // Original redirect URI
		ExpiresAt:   time.Now().Add(10 * time.Minute),
		Used:        false,
		CreatedAt:   time.Now(),
	}

	mockRepo.On("GetAppByClientID", ctx, clientID).Return(app, nil)
	mockRepo.On("GetAuthorizationCode", ctx, codeHash).Return(authCodeRecord, nil)

	// Different redirect URI!
	wrongRedirectURI := "https://evil.com/callback"
	req := &TokenRequest{
		GrantType:    "authorization_code",
		Code:         &authCode,
		RedirectURI:  &wrongRedirectURI,
		ClientID:     clientID,
		ClientSecret: &clientSecret,
	}

	result, err := service.ExchangeToken(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, ErrOAuthInvalidRedirectURI, err)
	assert.Nil(t, result)
	mockRepo.AssertExpectations(t)
}

// ========== Access Token Validation Edge Cases ==========

func TestOAuthProviderService_ValidateAccessToken_Expired(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockOAuthAppRepository)
	service := NewOAuthProviderService(mockRepo, nil, nil)

	accessToken := "expired-access-token"
	tokenHash := testSha256Hash(accessToken)
	userID := uuid.New()

	// Expired token
	tokenRecord := &models.OAuthAccessToken{
		ID:        uuid.New(),
		TokenHash: tokenHash,
		ClientID:  "test-client",
		UserID:    userID,
		Scopes:    []string{"read"},
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired!
		CreatedAt: time.Now().Add(-2 * time.Hour),
	}

	mockRepo.On("GetAccessTokenByHash", ctx, tokenHash).Return(tokenRecord, nil)

	result, err := service.ValidateAccessToken(ctx, accessToken)

	assert.Error(t, err)
	assert.Equal(t, ErrOAuthAccessTokenExpired, err)
	assert.Nil(t, result)
	mockRepo.AssertExpectations(t)
}

func TestOAuthProviderService_ValidateAccessToken_Revoked(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockOAuthAppRepository)
	service := NewOAuthProviderService(mockRepo, nil, nil)

	accessToken := "revoked-access-token"
	tokenHash := testSha256Hash(accessToken)
	userID := uuid.New()
	revokedAt := time.Now().Add(-30 * time.Minute)

	// Revoked token
	tokenRecord := &models.OAuthAccessToken{
		ID:        uuid.New(),
		TokenHash: tokenHash,
		ClientID:  "test-client",
		UserID:    userID,
		Scopes:    []string{"read"},
		ExpiresAt: time.Now().Add(1 * time.Hour),
		RevokedAt: &revokedAt,
		CreatedAt: time.Now().Add(-30 * time.Minute),
	}

	mockRepo.On("GetAccessTokenByHash", ctx, tokenHash).Return(tokenRecord, nil)

	result, err := service.ValidateAccessToken(ctx, accessToken)

	assert.Error(t, err)
	assert.Equal(t, ErrOAuthAccessTokenRevoked, err)
	assert.Nil(t, result)
	mockRepo.AssertExpectations(t)
}

// ========== App Management Edge Cases ==========

func TestOAuthProviderService_CreateApp_InvalidRedirectURI(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockOAuthAppRepository)
	service := NewOAuthProviderService(mockRepo, nil, nil)

	ownerID := uuid.New()

	tests := []struct {
		name         string
		redirectURIs []string
	}{
		{"empty redirect URI", []string{""}},
		// Note: ftp:// and custom schemes are allowed for native apps
		// Only test actually invalid URIs
		{"invalid URL format", []string{"not a valid url ::: ///"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := &CreateAppRequest{
				Name:         "Test App",
				RedirectURIs: tc.redirectURIs,
				Scopes:       []string{"read"},
			}

			result, err := service.CreateApp(ctx, ownerID, req)

			assert.Error(t, err)
			assert.Nil(t, result)
		})
	}
}

func TestOAuthProviderService_CreateApp_ValidLocalhost(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockOAuthAppRepository)
	service := NewOAuthProviderService(mockRepo, nil, nil)

	ownerID := uuid.New()

	mockRepo.On("CreateApp", ctx, mock.AnythingOfType("*models.OAuthApp")).Return(nil)

	// Localhost should be allowed for development
	req := &CreateAppRequest{
		Name:         "Test App",
		RedirectURIs: []string{"http://localhost:3000/callback"},
		Scopes:       []string{"read"},
	}

	result, err := service.CreateApp(ctx, ownerID, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	mockRepo.AssertExpectations(t)
}

func TestOAuthProviderService_CreateApp_CustomScheme(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockOAuthAppRepository)
	service := NewOAuthProviderService(mockRepo, nil, nil)

	ownerID := uuid.New()

	mockRepo.On("CreateApp", ctx, mock.AnythingOfType("*models.OAuthApp")).Return(nil)

	// Custom schemes for native apps should be allowed
	req := &CreateAppRequest{
		Name:         "Native App",
		RedirectURIs: []string{"myapp://callback"},
		Scopes:       []string{"read"},
	}

	result, err := service.CreateApp(ctx, ownerID, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	mockRepo.AssertExpectations(t)
}

// ========== Consent/Authorization Edge Cases ==========

func TestOAuthProviderService_ValidateAuthorization_ExistingConsentWithNewScopes(t *testing.T) {
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
		Scopes:       []string{"read", "write", "admin"},
		IsActive:     true,
		IsPublic:     false,
	}

	// User previously authorized only "read"
	existingAuth := &models.OAuthUserAuthorization{
		ID:           uuid.New(),
		UserID:       userID,
		ClientID:     clientID,
		Scopes:       []string{"read"},
		AuthorizedAt: time.Now().Add(-24 * time.Hour),
		LastUsedAt:   time.Now().Add(-1 * time.Hour),
	}

	mockRepo.On("GetAppByClientID", ctx, clientID).Return(app, nil)
	mockRepo.On("GetUserAuthorization", ctx, userID, clientID).Return(existingAuth, nil)

	// Now requesting "read" AND "write" - needs new consent
	req := &AuthorizeRequest{
		ClientID:     clientID,
		RedirectURI:  "https://example.com/callback",
		ResponseType: "code",
		Scope:        "read write",
	}

	result, err := service.ValidateAuthorization(ctx, userID, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.RequiresConsent) // Should require consent for new scope
	mockRepo.AssertExpectations(t)
}

func TestOAuthProviderService_ValidateAuthorization_SameScopes_NoConsent(t *testing.T) {
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
		Scopes:       []string{"read", "write"},
		IsActive:     true,
		IsPublic:     false,
	}

	// User previously authorized "read" and "write"
	existingAuth := &models.OAuthUserAuthorization{
		ID:           uuid.New(),
		UserID:       userID,
		ClientID:     clientID,
		Scopes:       []string{"read", "write"},
		AuthorizedAt: time.Now().Add(-24 * time.Hour),
		LastUsedAt:   time.Now().Add(-1 * time.Hour),
	}

	mockRepo.On("GetAppByClientID", ctx, clientID).Return(app, nil)
	mockRepo.On("GetUserAuthorization", ctx, userID, clientID).Return(existingAuth, nil)

	// Requesting only "read" - subset of existing scopes
	req := &AuthorizeRequest{
		ClientID:     clientID,
		RedirectURI:  "https://example.com/callback",
		ResponseType: "code",
		Scope:        "read",
	}

	result, err := service.ValidateAuthorization(ctx, userID, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.RequiresConsent) // No new consent needed
	mockRepo.AssertExpectations(t)
}

// ========== Token Revocation Tests ==========

func TestOAuthProviderService_RevokeToken_AccessToken(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockOAuthAppRepository)
	service := NewOAuthProviderService(mockRepo, nil, nil)

	accessToken := "test-access-token"
	tokenHash := testSha256Hash(accessToken)
	tokenID := uuid.New()

	accessTokenRecord := &models.OAuthAccessToken{
		ID:        tokenID,
		TokenHash: tokenHash,
		ClientID:  "test-client",
		UserID:    uuid.New(),
		Scopes:    []string{"read"},
		ExpiresAt: time.Now().Add(1 * time.Hour),
		CreatedAt: time.Now(),
	}

	mockRepo.On("GetRefreshTokenByHash", ctx, tokenHash).Return(nil, nil) // Not a refresh token
	mockRepo.On("GetAccessTokenByHash", ctx, tokenHash).Return(accessTokenRecord, nil)
	mockRepo.On("RevokeAccessToken", ctx, tokenID).Return(nil)

	err := service.RevokeToken(ctx, accessToken, "access_token")

	assert.NoError(t, err)
	mockRepo.AssertCalled(t, "RevokeAccessToken", ctx, tokenID)
}

func TestOAuthProviderService_RevokeToken_RefreshToken(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockOAuthAppRepository)
	service := NewOAuthProviderService(mockRepo, nil, nil)

	refreshToken := "test-refresh-token"
	tokenHash := testSha256Hash(refreshToken)
	refreshTokenID := uuid.New()
	accessTokenID := uuid.New()

	refreshTokenRecord := &models.OAuthRefreshToken{
		ID:            refreshTokenID,
		TokenHash:     tokenHash,
		AccessTokenID: accessTokenID,
		ClientID:      "test-client",
		UserID:        uuid.New(),
		Scopes:        []string{"read"},
		ExpiresAt:     time.Now().Add(30 * 24 * time.Hour),
		CreatedAt:     time.Now(),
	}

	mockRepo.On("GetRefreshTokenByHash", ctx, tokenHash).Return(refreshTokenRecord, nil)
	mockRepo.On("RevokeRefreshToken", ctx, refreshTokenID, "user_revoked").Return(nil)
	mockRepo.On("RevokeAccessToken", ctx, accessTokenID).Return(nil)

	err := service.RevokeToken(ctx, refreshToken, "refresh_token")

	assert.NoError(t, err)
	mockRepo.AssertCalled(t, "RevokeRefreshToken", ctx, refreshTokenID, "user_revoked")
	mockRepo.AssertCalled(t, "RevokeAccessToken", ctx, accessTokenID)
}

func TestOAuthProviderService_RevokeToken_NonExistent_NoError(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockOAuthAppRepository)
	service := NewOAuthProviderService(mockRepo, nil, nil)

	token := "non-existent-token"
	tokenHash := testSha256Hash(token)

	mockRepo.On("GetRefreshTokenByHash", ctx, tokenHash).Return(nil, nil)
	mockRepo.On("GetAccessTokenByHash", ctx, tokenHash).Return(nil, nil)

	// Per RFC 7009, revocation of non-existent token should not return error
	err := service.RevokeToken(ctx, token, "")

	assert.NoError(t, err)
}

// Note: strPtr helper is defined in session_service.go
