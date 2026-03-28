package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"hearth/internal/models"
)

// OAuth Provider errors
var (
	ErrOAuthAppNotFound          = errors.New("oauth application not found")
	ErrOAuthAppInactive          = errors.New("oauth application is inactive")
	ErrOAuthInvalidClientSecret  = errors.New("invalid client secret")
	ErrOAuthInvalidRedirectURI   = errors.New("invalid redirect URI")
	ErrOAuthInvalidScope         = errors.New("invalid scope")
	ErrOAuthInvalidCodeChallenge = errors.New("invalid PKCE code challenge")
	ErrOAuthCodeExpired          = errors.New("authorization code expired")
	ErrOAuthCodeAlreadyUsed      = errors.New("authorization code already used")
	ErrOAuthCodeNotFound         = errors.New("authorization code not found")
	ErrOAuthInvalidCodeVerifier  = errors.New("invalid PKCE code verifier")
	ErrOAuthRefreshTokenExpired  = errors.New("refresh token expired")
	ErrOAuthRefreshTokenRevoked  = errors.New("refresh token revoked")
	ErrOAuthRefreshTokenReused   = errors.New("refresh token reuse detected")
	ErrOAuthAccessTokenNotFound  = errors.New("access token not found")
	ErrOAuthAccessTokenExpired   = errors.New("access token expired")
	ErrOAuthAccessTokenRevoked   = errors.New("access token revoked")
	ErrOAuthPKCERequired         = errors.New("PKCE is required for public clients")
	ErrOAuthClientAuthRequired   = errors.New("client authentication required")
	ErrOAuthConsentRequired      = errors.New("user consent required")
)

// OAuthAppRepository defines the interface for OAuth app data storage
type OAuthAppRepository interface {
	CreateApp(ctx context.Context, app *models.OAuthApp) error
	GetAppByID(ctx context.Context, id uuid.UUID) (*models.OAuthApp, error)
	GetAppByClientID(ctx context.Context, clientID string) (*models.OAuthApp, error)
	GetAppsByOwner(ctx context.Context, ownerID uuid.UUID) ([]*models.OAuthApp, error)
	UpdateApp(ctx context.Context, app *models.OAuthApp) error
	DeleteApp(ctx context.Context, id, ownerID uuid.UUID) error
	RegenerateSecret(ctx context.Context, id, ownerID uuid.UUID, newSecretHash string) error

	CreateAuthorizationCode(ctx context.Context, code *models.OAuthAuthorizationCode) error
	GetAuthorizationCode(ctx context.Context, codeHash string) (*models.OAuthAuthorizationCode, error)
	MarkAuthorizationCodeUsed(ctx context.Context, id uuid.UUID) error
	CleanupExpiredAuthCodes(ctx context.Context) (int64, error)

	CreateAccessToken(ctx context.Context, token *models.OAuthAccessToken) error
	GetAccessTokenByHash(ctx context.Context, tokenHash string) (*models.OAuthAccessToken, error)
	RevokeAccessToken(ctx context.Context, id uuid.UUID) error
	RevokeAccessTokensByUser(ctx context.Context, userID uuid.UUID, clientID string) error

	CreateRefreshToken(ctx context.Context, token *models.OAuthRefreshToken) error
	GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*models.OAuthRefreshToken, error)
	RotateRefreshToken(ctx context.Context, oldID, newID uuid.UUID) error
	RevokeRefreshToken(ctx context.Context, id uuid.UUID, reason string) error
	RevokeRefreshTokenFamily(ctx context.Context, accessTokenID uuid.UUID, reason string) error
	RevokeRefreshTokensByUser(ctx context.Context, userID uuid.UUID, clientID string) error

	CreateOrUpdateUserAuthorization(ctx context.Context, auth *models.OAuthUserAuthorization) error
	GetUserAuthorization(ctx context.Context, userID uuid.UUID, clientID string) (*models.OAuthUserAuthorization, error)
	GetUserAuthorizations(ctx context.Context, userID uuid.UUID) ([]*models.OAuthUserAuthorization, error)
	RevokeUserAuthorization(ctx context.Context, userID uuid.UUID, clientID string) error
	UpdateLastUsed(ctx context.Context, userID uuid.UUID, clientID string) error
}

// OAuthServerConfig holds configuration for the OAuth authorization server
type OAuthServerConfig struct {
	AuthCodeExpiry     time.Duration // Default: 10 minutes
	AccessTokenExpiry  time.Duration // Default: 1 hour
	RefreshTokenExpiry time.Duration // Default: 30 days
	AllowedScopes      []string      // Allowed scopes
	RequirePKCE        bool          // Require PKCE for all clients
	IssuerURL          string        // Issuer URL for OpenID Connect
}

// DefaultOAuthServerConfig returns default configuration
func DefaultOAuthServerConfig() *OAuthServerConfig {
	return &OAuthServerConfig{
		AuthCodeExpiry:     10 * time.Minute,
		AccessTokenExpiry:  1 * time.Hour,
		RefreshTokenExpiry: 30 * 24 * time.Hour,
		AllowedScopes:      []string{"read", "write", "admin", "openid", "profile", "email", "servers", "messages"},
		RequirePKCE:        false,
	}
}

// OAuthProviderService handles OAuth 2.0 provider functionality
type OAuthProviderService struct {
	repo     OAuthAppRepository
	userRepo UserRepository
	config   *OAuthServerConfig
}

// NewOAuthProviderService creates a new OAuth provider service
func NewOAuthProviderService(repo OAuthAppRepository, userRepo UserRepository, config *OAuthServerConfig) *OAuthProviderService {
	if config == nil {
		config = DefaultOAuthServerConfig()
	}
	return &OAuthProviderService{
		repo:     repo,
		userRepo: userRepo,
		config:   config,
	}
}

// ======== App Management ========

// CreateAppRequest contains parameters for creating an OAuth app
type CreateAppRequest struct {
	Name         string   `json:"name"`
	Description  *string  `json:"description,omitempty"`
	RedirectURIs []string `json:"redirect_uris"`
	Scopes       []string `json:"scopes"`
	IconURL      *string  `json:"icon_url,omitempty"`
	HomepageURL  *string  `json:"homepage_url,omitempty"`
	PrivacyURL   *string  `json:"privacy_url,omitempty"`
	TermsURL     *string  `json:"terms_url,omitempty"`
	IsPublic     bool     `json:"is_public"`
}

// CreateAppResponse contains the created app info including the plain client secret
type CreateAppResponse struct {
	App          *models.OAuthApp `json:"app"`
	ClientSecret string           `json:"client_secret"` // Only returned on creation
}

// CreateApp creates a new OAuth application
func (s *OAuthProviderService) CreateApp(ctx context.Context, ownerID uuid.UUID, req *CreateAppRequest) (*CreateAppResponse, error) {
	// Validate name
	if len(req.Name) < 2 || len(req.Name) > 100 {
		return nil, errors.New("app name must be between 2 and 100 characters")
	}

	// Validate redirect URIs
	for _, uri := range req.RedirectURIs {
		if err := validateRedirectURI(uri); err != nil {
			return nil, fmt.Errorf("invalid redirect URI '%s': %w", uri, err)
		}
	}

	// Validate scopes
	for _, scope := range req.Scopes {
		if !models.ValidScopes[models.OAuthScope(scope)] {
			return nil, fmt.Errorf("invalid scope: %s", scope)
		}
	}

	// Generate client ID and secret
	clientID, err := generateSecureToken(24)
	if err != nil {
		return nil, fmt.Errorf("failed to generate client ID: %w", err)
	}

	clientSecret, err := generateSecureToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate client secret: %w", err)
	}

	// Hash the client secret
	secretHash, err := bcrypt.GenerateFromPassword([]byte(clientSecret), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash client secret: %w", err)
	}

	now := time.Now()
	app := &models.OAuthApp{
		ID:           uuid.New(),
		OwnerID:      ownerID,
		Name:         req.Name,
		Description:  req.Description,
		ClientID:     clientID,
		ClientSecret: string(secretHash),
		RedirectURIs: req.RedirectURIs,
		Scopes:       req.Scopes,
		IconURL:      req.IconURL,
		HomepageURL:  req.HomepageURL,
		PrivacyURL:   req.PrivacyURL,
		TermsURL:     req.TermsURL,
		IsPublic:     req.IsPublic,
		IsVerified:   false,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.CreateApp(ctx, app); err != nil {
		return nil, fmt.Errorf("failed to create app: %w", err)
	}

	return &CreateAppResponse{
		App:          app,
		ClientSecret: clientSecret,
	}, nil
}

// GetApp retrieves an OAuth app by ID (owner only)
func (s *OAuthProviderService) GetApp(ctx context.Context, id, ownerID uuid.UUID) (*models.OAuthApp, error) {
	app, err := s.repo.GetAppByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if app == nil || app.OwnerID != ownerID {
		return nil, ErrOAuthAppNotFound
	}
	return app, nil
}

// GetAppByClientID retrieves an OAuth app by client ID
func (s *OAuthProviderService) GetAppByClientID(ctx context.Context, clientID string) (*models.OAuthApp, error) {
	app, err := s.repo.GetAppByClientID(ctx, clientID)
	if err != nil {
		return nil, err
	}
	if app == nil {
		return nil, ErrOAuthAppNotFound
	}
	if !app.IsActive {
		return nil, ErrOAuthAppInactive
	}
	return app, nil
}

// GetAppsByOwner retrieves all apps owned by a user
func (s *OAuthProviderService) GetAppsByOwner(ctx context.Context, ownerID uuid.UUID) ([]*models.OAuthApp, error) {
	return s.repo.GetAppsByOwner(ctx, ownerID)
}

// UpdateApp updates an OAuth application
func (s *OAuthProviderService) UpdateApp(ctx context.Context, id, ownerID uuid.UUID, req *CreateAppRequest) (*models.OAuthApp, error) {
	app, err := s.repo.GetAppByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if app == nil || app.OwnerID != ownerID {
		return nil, ErrOAuthAppNotFound
	}

	// Validate redirect URIs
	for _, uri := range req.RedirectURIs {
		if err := validateRedirectURI(uri); err != nil {
			return nil, fmt.Errorf("invalid redirect URI '%s': %w", uri, err)
		}
	}

	// Validate scopes
	for _, scope := range req.Scopes {
		if !models.ValidScopes[models.OAuthScope(scope)] {
			return nil, fmt.Errorf("invalid scope: %s", scope)
		}
	}

	app.Name = req.Name
	app.Description = req.Description
	app.RedirectURIs = req.RedirectURIs
	app.Scopes = req.Scopes
	app.IconURL = req.IconURL
	app.HomepageURL = req.HomepageURL
	app.PrivacyURL = req.PrivacyURL
	app.TermsURL = req.TermsURL
	app.IsPublic = req.IsPublic

	if err := s.repo.UpdateApp(ctx, app); err != nil {
		return nil, err
	}

	return app, nil
}

// DeleteApp deletes an OAuth application
func (s *OAuthProviderService) DeleteApp(ctx context.Context, id, ownerID uuid.UUID) error {
	return s.repo.DeleteApp(ctx, id, ownerID)
}

// RegenerateClientSecret generates a new client secret
func (s *OAuthProviderService) RegenerateClientSecret(ctx context.Context, id, ownerID uuid.UUID) (string, error) {
	app, err := s.repo.GetAppByID(ctx, id)
	if err != nil {
		return "", err
	}
	if app == nil || app.OwnerID != ownerID {
		return "", ErrOAuthAppNotFound
	}

	clientSecret, err := generateSecureToken(32)
	if err != nil {
		return "", fmt.Errorf("failed to generate client secret: %w", err)
	}

	secretHash, err := bcrypt.GenerateFromPassword([]byte(clientSecret), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash client secret: %w", err)
	}

	if err := s.repo.RegenerateSecret(ctx, id, ownerID, string(secretHash)); err != nil {
		return "", err
	}

	return clientSecret, nil
}

// ======== Authorization Flow ========

// AuthorizeRequest represents an authorization request
type AuthorizeRequest struct {
	ClientID            string  `json:"client_id"`
	RedirectURI         string  `json:"redirect_uri"`
	ResponseType        string  `json:"response_type"` // "code"
	Scope               string  `json:"scope"`
	State               *string `json:"state,omitempty"`
	CodeChallenge       *string `json:"code_challenge,omitempty"`        // PKCE
	CodeChallengeMethod *string `json:"code_challenge_method,omitempty"` // "plain" or "S256"
	Nonce               *string `json:"nonce,omitempty"`                 // OpenID Connect
}

// AuthorizeResponse contains authorization info for the consent page
type AuthorizeResponse struct {
	App             *models.OAuthApp               `json:"app"`
	RequestedScopes []string                       `json:"requested_scopes"`
	ScopeDetails    []ScopeDetail                  `json:"scope_details"`
	ExistingAuth    *models.OAuthUserAuthorization `json:"existing_auth,omitempty"`
	RequiresConsent bool                           `json:"requires_consent"`
	ConsentURL      string                         `json:"consent_url,omitempty"`
}

// ScopeDetail provides details about a requested scope
type ScopeDetail struct {
	Scope       string `json:"scope"`
	Description string `json:"description"`
}

// ValidateAuthorization validates an authorization request and returns info for consent
func (s *OAuthProviderService) ValidateAuthorization(ctx context.Context, userID uuid.UUID, req *AuthorizeRequest) (*AuthorizeResponse, error) {
	// Validate response_type
	if req.ResponseType != "code" {
		return nil, errors.New("unsupported response_type, only 'code' is supported")
	}

	// Get app
	app, err := s.repo.GetAppByClientID(ctx, req.ClientID)
	if err != nil {
		return nil, ErrOAuthAppNotFound
	}
	if app == nil {
		return nil, ErrOAuthAppNotFound
	}
	if !app.IsActive {
		return nil, ErrOAuthAppInactive
	}

	// Validate redirect URI
	if !isValidRedirectURI(app.RedirectURIs, req.RedirectURI) {
		return nil, ErrOAuthInvalidRedirectURI
	}

	// Validate PKCE for public clients
	if app.IsPublic && req.CodeChallenge == nil {
		return nil, ErrOAuthPKCERequired
	}
	if req.CodeChallengeMethod != nil && *req.CodeChallengeMethod != "plain" && *req.CodeChallengeMethod != "S256" {
		return nil, ErrOAuthInvalidCodeChallenge
	}

	// Parse and validate scopes
	requestedScopes := parseScopes(req.Scope)
	for _, scope := range requestedScopes {
		if !containsScope(app.Scopes, scope) {
			return nil, fmt.Errorf("%w: %s", ErrOAuthInvalidScope, scope)
		}
	}

	// Get scope details
	scopeDetails := make([]ScopeDetail, len(requestedScopes))
	for i, scope := range requestedScopes {
		scopeDetails[i] = ScopeDetail{
			Scope:       scope,
			Description: models.ScopeDescriptions[models.OAuthScope(scope)],
		}
	}

	// Check for existing authorization
	existingAuth, _ := s.repo.GetUserAuthorization(ctx, userID, req.ClientID)
	requiresConsent := existingAuth == nil || !scopesContained(existingAuth.Scopes, requestedScopes)

	return &AuthorizeResponse{
		App:             app,
		RequestedScopes: requestedScopes,
		ScopeDetails:    scopeDetails,
		ExistingAuth:    existingAuth,
		RequiresConsent: requiresConsent,
	}, nil
}

// ApproveAuthorization creates an authorization code after user consent
func (s *OAuthProviderService) ApproveAuthorization(ctx context.Context, userID uuid.UUID, req *AuthorizeRequest) (string, error) {
	// Validate request
	_, err := s.ValidateAuthorization(ctx, userID, req)
	if err != nil {
		return "", err
	}

	app, _ := s.repo.GetAppByClientID(ctx, req.ClientID)

	// Generate authorization code
	code, err := generateSecureToken(32)
	if err != nil {
		return "", fmt.Errorf("failed to generate authorization code: %w", err)
	}

	// Hash the code for storage
	codeHash := sha256Hash(code)

	// Parse scopes
	scopes := parseScopes(req.Scope)

	// Create authorization code record
	authCode := &models.OAuthAuthorizationCode{
		ID:                  uuid.New(),
		Code:                codeHash,
		ClientID:            req.ClientID,
		UserID:              userID,
		Scopes:              scopes,
		RedirectURI:         req.RedirectURI,
		CodeChallenge:       req.CodeChallenge,
		CodeChallengeMethod: req.CodeChallengeMethod,
		Nonce:               req.Nonce,
		State:               req.State,
		ExpiresAt:           time.Now().Add(s.config.AuthCodeExpiry),
		Used:                false,
		CreatedAt:           time.Now(),
	}

	if err := s.repo.CreateAuthorizationCode(ctx, authCode); err != nil {
		return "", fmt.Errorf("failed to store authorization code: %w", err)
	}

	// Create or update user authorization
	now := time.Now()
	userAuth := &models.OAuthUserAuthorization{
		ID:           uuid.New(),
		UserID:       userID,
		ClientID:     app.ClientID,
		Scopes:       scopes,
		AuthorizedAt: now,
		LastUsedAt:   now,
	}
	if err := s.repo.CreateOrUpdateUserAuthorization(ctx, userAuth); err != nil {
		// Non-fatal, continue
	}

	return code, nil
}

// ======== Token Exchange ========

// TokenRequest represents a token exchange request
type TokenRequest struct {
	GrantType    string  `json:"grant_type"` // "authorization_code" or "refresh_token"
	Code         *string `json:"code,omitempty"`
	RedirectURI  *string `json:"redirect_uri,omitempty"`
	ClientID     string  `json:"client_id"`
	ClientSecret *string `json:"client_secret,omitempty"` // Required for confidential clients
	CodeVerifier *string `json:"code_verifier,omitempty"` // PKCE
	RefreshToken *string `json:"refresh_token,omitempty"`
}

// TokenResponse represents a token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope"`
	IDToken      string `json:"id_token,omitempty"` // OpenID Connect
}

// ExchangeToken exchanges an authorization code or refresh token for tokens
func (s *OAuthProviderService) ExchangeToken(ctx context.Context, req *TokenRequest) (*TokenResponse, error) {
	switch req.GrantType {
	case "authorization_code":
		return s.exchangeAuthorizationCode(ctx, req)
	case "refresh_token":
		return s.refreshAccessToken(ctx, req)
	default:
		return nil, errors.New("unsupported grant_type")
	}
}

func (s *OAuthProviderService) exchangeAuthorizationCode(ctx context.Context, req *TokenRequest) (*TokenResponse, error) {
	if req.Code == nil || req.RedirectURI == nil {
		return nil, errors.New("code and redirect_uri are required")
	}

	// Get app
	app, err := s.repo.GetAppByClientID(ctx, req.ClientID)
	if err != nil || app == nil {
		return nil, ErrOAuthAppNotFound
	}
	if !app.IsActive {
		return nil, ErrOAuthAppInactive
	}

	// Authenticate client (required for confidential clients)
	if !app.IsPublic {
		if req.ClientSecret == nil {
			return nil, ErrOAuthClientAuthRequired
		}
		if err := bcrypt.CompareHashAndPassword([]byte(app.ClientSecret), []byte(*req.ClientSecret)); err != nil {
			return nil, ErrOAuthInvalidClientSecret
		}
	}

	// Get authorization code
	codeHash := sha256Hash(*req.Code)
	authCode, err := s.repo.GetAuthorizationCode(ctx, codeHash)
	if err != nil {
		return nil, err
	}
	if authCode == nil {
		return nil, ErrOAuthCodeNotFound
	}

	// Validate code
	if authCode.ClientID != req.ClientID {
		return nil, ErrOAuthCodeNotFound
	}
	if authCode.RedirectURI != *req.RedirectURI {
		return nil, ErrOAuthInvalidRedirectURI
	}
	if authCode.Used {
		return nil, ErrOAuthCodeAlreadyUsed
	}
	if time.Now().After(authCode.ExpiresAt) {
		return nil, ErrOAuthCodeExpired
	}

	// Validate PKCE
	if authCode.CodeChallenge != nil {
		if req.CodeVerifier == nil {
			return nil, ErrOAuthInvalidCodeVerifier
		}
		if !verifyPKCE(*authCode.CodeChallenge, authCode.CodeChallengeMethod, *req.CodeVerifier) {
			return nil, ErrOAuthInvalidCodeVerifier
		}
	}

	// Mark code as used
	if err := s.repo.MarkAuthorizationCodeUsed(ctx, authCode.ID); err != nil {
		return nil, err
	}

	// Generate tokens
	return s.generateTokens(ctx, app, authCode.UserID, authCode.Scopes)
}

func (s *OAuthProviderService) refreshAccessToken(ctx context.Context, req *TokenRequest) (*TokenResponse, error) {
	if req.RefreshToken == nil {
		return nil, errors.New("refresh_token is required")
	}

	// Get app
	app, err := s.repo.GetAppByClientID(ctx, req.ClientID)
	if err != nil || app == nil {
		return nil, ErrOAuthAppNotFound
	}
	if !app.IsActive {
		return nil, ErrOAuthAppInactive
	}

	// Authenticate client (required for confidential clients)
	if !app.IsPublic {
		if req.ClientSecret == nil {
			return nil, ErrOAuthClientAuthRequired
		}
		if err := bcrypt.CompareHashAndPassword([]byte(app.ClientSecret), []byte(*req.ClientSecret)); err != nil {
			return nil, ErrOAuthInvalidClientSecret
		}
	}

	// Get refresh token
	tokenHash := sha256Hash(*req.RefreshToken)
	refreshToken, err := s.repo.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		return nil, err
	}
	if refreshToken == nil {
		return nil, ErrOAuthRefreshTokenRevoked
	}

	// Check if token belongs to this client
	if refreshToken.ClientID != req.ClientID {
		return nil, ErrOAuthRefreshTokenRevoked
	}

	// Check if token is revoked
	if refreshToken.RevokedAt != nil {
		return nil, ErrOAuthRefreshTokenRevoked
	}

	// Check for reuse (token rotation detection)
	if refreshToken.RotatedAt != nil {
		// This token was already rotated - potential reuse attack!
		// Revoke entire token family
		if err := s.repo.RevokeRefreshTokenFamily(ctx, refreshToken.AccessTokenID, "reuse_detected"); err != nil {
			// Log but continue
		}
		if err := s.repo.RevokeAccessToken(ctx, refreshToken.AccessTokenID); err != nil {
			// Log but continue
		}
		return nil, ErrOAuthRefreshTokenReused
	}

	// Check expiry
	if time.Now().After(refreshToken.ExpiresAt) {
		return nil, ErrOAuthRefreshTokenExpired
	}

	// Update last used
	if err := s.repo.UpdateLastUsed(ctx, refreshToken.UserID, refreshToken.ClientID); err != nil {
		// Non-fatal
	}

	// Generate new tokens with rotation
	return s.generateTokensWithRotation(ctx, app, refreshToken)
}

func (s *OAuthProviderService) generateTokens(ctx context.Context, app *models.OAuthApp, userID uuid.UUID, scopes []string) (*TokenResponse, error) {
	// Generate access token
	accessToken, err := generateSecureToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}
	accessTokenHash := sha256Hash(accessToken)

	now := time.Now()
	accessTokenRecord := &models.OAuthAccessToken{
		ID:        uuid.New(),
		TokenHash: accessTokenHash,
		ClientID:  app.ClientID,
		UserID:    userID,
		Scopes:    scopes,
		ExpiresAt: now.Add(s.config.AccessTokenExpiry),
		CreatedAt: now,
	}

	if err := s.repo.CreateAccessToken(ctx, accessTokenRecord); err != nil {
		return nil, fmt.Errorf("failed to store access token: %w", err)
	}

	// Generate refresh token
	refreshTokenStr, err := generateSecureToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}
	refreshTokenHash := sha256Hash(refreshTokenStr)

	refreshTokenRecord := &models.OAuthRefreshToken{
		ID:            uuid.New(),
		TokenHash:     refreshTokenHash,
		AccessTokenID: accessTokenRecord.ID,
		ClientID:      app.ClientID,
		UserID:        userID,
		Scopes:        scopes,
		ExpiresAt:     now.Add(s.config.RefreshTokenExpiry),
		CreatedAt:     now,
	}

	if err := s.repo.CreateRefreshToken(ctx, refreshTokenRecord); err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	return &TokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.config.AccessTokenExpiry.Seconds()),
		RefreshToken: refreshTokenStr,
		Scope:        strings.Join(scopes, " "),
	}, nil
}

func (s *OAuthProviderService) generateTokensWithRotation(ctx context.Context, app *models.OAuthApp, oldRefreshToken *models.OAuthRefreshToken) (*TokenResponse, error) {
	// Generate new access token
	accessToken, err := generateSecureToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}
	accessTokenHash := sha256Hash(accessToken)

	now := time.Now()
	accessTokenRecord := &models.OAuthAccessToken{
		ID:        uuid.New(),
		TokenHash: accessTokenHash,
		ClientID:  app.ClientID,
		UserID:    oldRefreshToken.UserID,
		Scopes:    oldRefreshToken.Scopes,
		ExpiresAt: now.Add(s.config.AccessTokenExpiry),
		CreatedAt: now,
	}

	if err := s.repo.CreateAccessToken(ctx, accessTokenRecord); err != nil {
		return nil, fmt.Errorf("failed to store access token: %w", err)
	}

	// Generate new refresh token
	newRefreshTokenStr, err := generateSecureToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}
	newRefreshTokenHash := sha256Hash(newRefreshTokenStr)

	newRefreshTokenRecord := &models.OAuthRefreshToken{
		ID:            uuid.New(),
		TokenHash:     newRefreshTokenHash,
		AccessTokenID: accessTokenRecord.ID,
		ClientID:      app.ClientID,
		UserID:        oldRefreshToken.UserID,
		Scopes:        oldRefreshToken.Scopes,
		ExpiresAt:     now.Add(s.config.RefreshTokenExpiry),
		CreatedAt:     now,
	}

	if err := s.repo.CreateRefreshToken(ctx, newRefreshTokenRecord); err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	// Rotate old refresh token (mark as rotated, point to new one)
	if err := s.repo.RotateRefreshToken(ctx, oldRefreshToken.ID, newRefreshTokenRecord.ID); err != nil {
		// Non-fatal, but concerning
	}

	return &TokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.config.AccessTokenExpiry.Seconds()),
		RefreshToken: newRefreshTokenStr,
		Scope:        strings.Join(oldRefreshToken.Scopes, " "),
	}, nil
}

// ======== Token Validation ========

// ValidatedToken contains validated token information
type ValidatedToken struct {
	UserID   uuid.UUID
	ClientID string
	Scopes   []string
}

// ValidateAccessToken validates an access token and returns its claims
func (s *OAuthProviderService) ValidateAccessToken(ctx context.Context, accessToken string) (*ValidatedToken, error) {
	tokenHash := sha256Hash(accessToken)
	token, err := s.repo.GetAccessTokenByHash(ctx, tokenHash)
	if err != nil {
		return nil, err
	}
	if token == nil {
		return nil, ErrOAuthAccessTokenNotFound
	}
	if token.RevokedAt != nil {
		return nil, ErrOAuthAccessTokenRevoked
	}
	if time.Now().After(token.ExpiresAt) {
		return nil, ErrOAuthAccessTokenExpired
	}

	return &ValidatedToken{
		UserID:   token.UserID,
		ClientID: token.ClientID,
		Scopes:   token.Scopes,
	}, nil
}

// HasScope checks if a validated token has a specific scope
func (v *ValidatedToken) HasScope(scope string) bool {
	for _, s := range v.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// ======== User Authorization Management ========

// GetUserAuthorizations returns all apps a user has authorized
func (s *OAuthProviderService) GetUserAuthorizations(ctx context.Context, userID uuid.UUID) ([]*models.OAuthUserAuthorizationResponse, error) {
	auths, err := s.repo.GetUserAuthorizations(ctx, userID)
	if err != nil {
		return nil, err
	}

	responses := make([]*models.OAuthUserAuthorizationResponse, 0, len(auths))
	for _, auth := range auths {
		app, err := s.repo.GetAppByClientID(ctx, auth.ClientID)
		if err != nil || app == nil {
			continue
		}

		responses = append(responses, &models.OAuthUserAuthorizationResponse{
			ID:           auth.ID,
			App:          app.ToResponse(),
			Scopes:       auth.Scopes,
			AuthorizedAt: auth.AuthorizedAt,
			LastUsedAt:   auth.LastUsedAt,
		})
	}

	return responses, nil
}

// RevokeUserAuthorization revokes a user's authorization for an app
func (s *OAuthProviderService) RevokeUserAuthorization(ctx context.Context, userID uuid.UUID, clientID string) error {
	// Revoke all tokens
	if err := s.repo.RevokeAccessTokensByUser(ctx, userID, clientID); err != nil {
		// Non-fatal
	}
	if err := s.repo.RevokeRefreshTokensByUser(ctx, userID, clientID); err != nil {
		// Non-fatal
	}

	// Revoke authorization
	return s.repo.RevokeUserAuthorization(ctx, userID, clientID)
}

// ======== Token Revocation ========

// RevokeToken revokes a token (access or refresh)
func (s *OAuthProviderService) RevokeToken(ctx context.Context, token string, tokenTypeHint string) error {
	tokenHash := sha256Hash(token)

	// Try as refresh token first if hinted or unknown
	if tokenTypeHint == "" || tokenTypeHint == "refresh_token" {
		refreshToken, err := s.repo.GetRefreshTokenByHash(ctx, tokenHash)
		if err == nil && refreshToken != nil {
			// Revoke refresh token and associated access token
			if err := s.repo.RevokeRefreshToken(ctx, refreshToken.ID, "user_revoked"); err != nil {
				return err
			}
			return s.repo.RevokeAccessToken(ctx, refreshToken.AccessTokenID)
		}
	}

	// Try as access token
	if tokenTypeHint == "" || tokenTypeHint == "access_token" {
		accessToken, err := s.repo.GetAccessTokenByHash(ctx, tokenHash)
		if err == nil && accessToken != nil {
			return s.repo.RevokeAccessToken(ctx, accessToken.ID)
		}
	}

	// Token not found - this is OK per RFC 7009
	return nil
}

// ======== Helper Functions ========

func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

func sha256Hash(data string) string {
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func validateRedirectURI(uri string) error {
	parsed, err := url.Parse(uri)
	if err != nil {
		return errors.New("invalid URL format")
	}

	// Allow localhost for development
	if parsed.Host == "localhost" || strings.HasPrefix(parsed.Host, "localhost:") || strings.HasPrefix(parsed.Host, "127.0.0.1") {
		return nil
	}

	// Require HTTPS for production
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		// Allow custom schemes for native apps (e.g., myapp://)
		if !strings.Contains(parsed.Scheme, ".") && len(parsed.Scheme) > 0 {
			return nil
		}
		return errors.New("redirect URI must use HTTPS")
	}

	return nil
}

func isValidRedirectURI(allowedURIs []string, uri string) bool {
	for _, allowed := range allowedURIs {
		if allowed == uri {
			return true
		}
	}
	return false
}

func parseScopes(scopeString string) []string {
	if scopeString == "" {
		return []string{}
	}
	return strings.Fields(scopeString)
}

func containsScope(scopes []string, scope string) bool {
	for _, s := range scopes {
		if s == scope {
			return true
		}
	}
	return false
}

func scopesContained(existing, requested []string) bool {
	existingSet := make(map[string]bool)
	for _, s := range existing {
		existingSet[s] = true
	}
	for _, s := range requested {
		if !existingSet[s] {
			return false
		}
	}
	return true
}

func verifyPKCE(challenge string, method *string, verifier string) bool {
	if method == nil || *method == "plain" {
		return subtle.ConstantTimeCompare([]byte(challenge), []byte(verifier)) == 1
	}
	if *method == "S256" {
		hash := sha256.Sum256([]byte(verifier))
		computed := base64.RawURLEncoding.EncodeToString(hash[:])
		return subtle.ConstantTimeCompare([]byte(challenge), []byte(computed)) == 1
	}
	return false
}
