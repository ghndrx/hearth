package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"hearth/internal/auth"
	"hearth/internal/models"
)

var (
	ErrUserExists = errors.New("user already exists")
)

// AuthTokens represents access and refresh tokens
type AuthTokens struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
}

// MFASetupResponse contains information needed to set up 2FA
type MFASetupResponse struct {
	Secret      string   `json:"secret"`
	QRCodeURL   string   `json:"qr_code_url"`
	BackupCodes []string `json:"backup_codes"`
}

// AuthService defines the business logic for authentication.
type AuthService interface {
	Register(ctx context.Context, email, username, password string) (*models.User, *AuthTokens, error)
	Login(ctx context.Context, email, password string) (*models.User, *AuthTokens, error)
	LoginWithMFA(ctx context.Context, email, password, mfaCode string) (*models.User, *AuthTokens, error)
	RefreshTokens(ctx context.Context, refreshToken string) (*AuthTokens, error)
	ValidateToken(ctx context.Context, token string) (uuid.UUID, error)

	// MFA methods
	EnableMFA(ctx context.Context, userID uuid.UUID) (*MFASetupResponse, error)
	VerifyMFASetup(ctx context.Context, userID uuid.UUID, code string) error
	DisableMFA(ctx context.Context, userID uuid.UUID, password string) error
	VerifyMFA(ctx context.Context, userID uuid.UUID, code string) error
	CheckUserMFA(ctx context.Context, email string) (bool, error)
}

// authRepository defines the storage interface required by the auth service.
type authRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByID(ctx context.Context, userID uuid.UUID) (*models.User, error)
	UpdateMFA(ctx context.Context, userID uuid.UUID, enabled bool, secret *string) error
}

type authService struct {
	repo       authRepository
	jwtService *auth.JWTService
}

// NewAuthService creates a new auth service instance.
func NewAuthService(repo authRepository, jwtService *auth.JWTService) AuthService {
	return &authService{
		repo:       repo,
		jwtService: jwtService,
	}
}

// Register handles new user registration.
func (s *authService) Register(ctx context.Context, email, username, password string) (*models.User, *AuthTokens, error) {
	// Check if user already exists
	_, err := s.repo.GetByEmail(ctx, email)
	if err == nil {
		return nil, nil, ErrEmailTaken
	}
	if !errors.Is(err, ErrUserNotFound) {
		return nil, nil, err // Return unexpected database errors
	}

	// Hash password using bounded worker pool (prevents CPU saturation under load)
	hashedPassword, err := auth.HashPasswordPooled(ctx, password)
	if err != nil {
		// Convert auth package errors to services errors for proper HTTP handling
		switch err {
		case auth.ErrPasswordTooShort:
			return nil, nil, ErrPasswordTooShort
		case auth.ErrPasswordTooLong:
			return nil, nil, ErrPasswordTooLong
		case auth.ErrPasswordWeak:
			return nil, nil, ErrPasswordWeak
		default:
			return nil, nil, err
		}
	}

	user := &models.User{
		ID:           uuid.New(),
		Email:        email,
		Username:     username,
		PasswordHash: hashedPassword,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, nil, err
	}

	// Generate JWT tokens
	tokens, err := s.generateTokens(user)
	if err != nil {
		return nil, nil, err
	}

	return user, tokens, nil
}

// Login handles user login and credentials verification.
// If MFA is enabled, it returns ErrMFARequired. Use LoginWithMFA instead.
func (s *authService) Login(ctx context.Context, email, password string) (*models.User, *AuthTokens, error) {
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, nil, ErrInvalidCredentials
		}
		return nil, nil, err
	}

	// Verify password using bounded worker pool (prevents CPU saturation under load)
	if err := auth.CheckPasswordPooled(ctx, password, user.PasswordHash); err != nil {
		return nil, nil, ErrInvalidCredentials
	}

	// Check if MFA is enabled
	if user.MFAEnabled {
		return nil, nil, ErrMFARequired
	}

	// Generate JWT tokens
	tokens, err := s.generateTokens(user)
	if err != nil {
		return nil, nil, err
	}

	return user, tokens, nil
}

// RefreshTokens refreshes access and refresh tokens
func (s *authService) RefreshTokens(ctx context.Context, refreshToken string) (*AuthTokens, error) {
	claims, err := s.jwtService.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}

	// Look up user to get current admin status
	user, err := s.repo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	isAdmin := user.Flags&models.UserFlagStaff != 0
	accessToken, newRefreshToken, err := s.jwtService.GenerateTokenPairWithAdmin(claims.UserID, claims.Username, isAdmin)
	if err != nil {
		return nil, err
	}

	return &AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    s.jwtService.GetExpirySeconds(),
	}, nil
}

// ValidateToken validates an access token and returns the user ID
func (s *authService) ValidateToken(ctx context.Context, token string) (uuid.UUID, error) {
	claims, err := s.jwtService.ValidateAccessToken(token)
	if err != nil {
		return uuid.Nil, err
	}

	return claims.UserID, nil
}

// LoginWithMFA handles user login with MFA verification
func (s *authService) LoginWithMFA(ctx context.Context, email, password, mfaCode string) (*models.User, *AuthTokens, error) {
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, nil, ErrInvalidCredentials
		}
		return nil, nil, err
	}

	// Verify password
	if err := auth.CheckPasswordPooled(ctx, password, user.PasswordHash); err != nil {
		return nil, nil, ErrInvalidCredentials
	}

	// If MFA is enabled, verify the MFA code
	if user.MFAEnabled {
		if mfaCode == "" {
			return nil, nil, ErrMFARequired
		}

		if user.MFASecret == nil {
			return nil, nil, errors.New("MFA enabled but no secret found")
		}

		if !auth.VerifyTOTP(mfaCode, *user.MFASecret) {
			return nil, nil, ErrInvalidMFACode
		}
	}

	// Generate JWT tokens
	tokens, err := s.generateTokens(user)
	if err != nil {
		return nil, nil, err
	}

	return user, tokens, nil
}

// EnableMFA generates a new TOTP secret and backup codes for the user
func (s *authService) EnableMFA(ctx context.Context, userID uuid.UUID) (*MFASetupResponse, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if user.MFAEnabled {
		return nil, ErrMFAAlreadyEnabled
	}

	// Generate TOTP secret
	config := auth.TOTPConfig{
		Issuer:      "Hearth",
		AccountName: user.Email,
		SecretSize:  32,
	}

	secret, err := auth.GenerateTOTPSecret(config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate TOTP secret: %w", err)
	}

	// Generate QR code URL
	qrURL, err := auth.GenerateQRCode(secret, "Hearth", user.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to generate QR code: %w", err)
	}

	// Generate backup codes
	backupCodes, err := auth.GenerateBackupCodes(10)
	if err != nil {
		return nil, fmt.Errorf("failed to generate backup codes: %w", err)
	}

	// Store the secret (but don't enable MFA yet - that happens on verification)
	if err := s.repo.UpdateMFA(ctx, userID, false, &secret); err != nil {
		return nil, err
	}

	// Format backup codes for display
	formattedCodes := make([]string, len(backupCodes))
	for i, code := range backupCodes {
		formattedCodes[i] = auth.FormatBackupCode(code)
	}

	return &MFASetupResponse{
		Secret:      secret,
		QRCodeURL:   qrURL,
		BackupCodes: formattedCodes,
	}, nil
}

// VerifyMFASetup verifies the TOTP code and enables MFA for the user
func (s *authService) VerifyMFASetup(ctx context.Context, userID uuid.UUID, code string) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if user.MFAEnabled {
		return ErrMFAAlreadyEnabled
	}

	if user.MFASecret == nil {
		return errors.New("no MFA secret found - please start MFA setup first")
	}

	// Verify the TOTP code
	if !auth.VerifyTOTP(code, *user.MFASecret) {
		return ErrInvalidMFACode
	}

	// Enable MFA for the user
	return s.repo.UpdateMFA(ctx, userID, true, user.MFASecret)
}

// DisableMFA disables MFA for the user after password verification
func (s *authService) DisableMFA(ctx context.Context, userID uuid.UUID, password string) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if !user.MFAEnabled {
		return ErrMFANotEnabled
	}

	// Verify password
	if err := auth.CheckPasswordPooled(ctx, password, user.PasswordHash); err != nil {
		return ErrInvalidCredentials
	}

	// Disable MFA and clear secret
	return s.repo.UpdateMFA(ctx, userID, false, nil)
}

// VerifyMFA verifies a TOTP code for an already enabled MFA user
func (s *authService) VerifyMFA(ctx context.Context, userID uuid.UUID, code string) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if !user.MFAEnabled {
		return ErrMFANotEnabled
	}

	if user.MFASecret == nil {
		return errors.New("MFA enabled but no secret found")
	}

	if !auth.VerifyTOTP(code, *user.MFASecret) {
		return ErrInvalidMFACode
	}

	return nil
}

// CheckUserMFA checks if a user has MFA enabled
func (s *authService) CheckUserMFA(ctx context.Context, email string) (bool, error) {
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return false, ErrInvalidCredentials
		}
		return false, err
	}

	return user.MFAEnabled, nil
}

// generateTokens creates a new token pair for a user
func (s *authService) generateTokens(user *models.User) (*AuthTokens, error) {
	isAdmin := user.Flags&models.UserFlagStaff != 0
	accessToken, refreshToken, err := s.jwtService.GenerateTokenPairWithAdmin(user.ID, user.Username, isAdmin)
	if err != nil {
		return nil, err
	}

	return &AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    s.jwtService.GetExpirySeconds(),
	}, nil
}

// OAuthRepository defines the interface for OAuth provider storage
type OAuthRepository interface {
	Create(ctx context.Context, provider *models.OAuthProvider) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.OAuthProvider, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.OAuthProvider, error)
	GetByUserAndProvider(ctx context.Context, userID uuid.UUID, provider string) (*models.OAuthProvider, error)
	GetByProviderUserID(ctx context.Context, provider, providerUserID string) (*models.OAuthProvider, error)
	Update(ctx context.Context, provider *models.OAuthProvider) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByUserAndProvider(ctx context.Context, userID uuid.UUID, provider string) error
	CountByUserID(ctx context.Context, userID uuid.UUID) (int, error)
	ExistsByProviderUserID(ctx context.Context, provider, providerUserID string) (bool, error)
	GetUserIDByProviderUserID(ctx context.Context, provider, providerUserID string) (uuid.UUID, error)
}

// OAuth errors
var (
	ErrOAuthProviderNotSupported  = errors.New("oauth provider not supported")
	ErrOAuthStateMismatch         = errors.New("oauth state mismatch")
	ErrOAuthStateExpired          = errors.New("oauth state expired")
	ErrOAuthCodeExchange          = errors.New("failed to exchange code for token")
	ErrOAuthUserInfo              = errors.New("failed to get user info from provider")
	ErrOAuthEmailNotVerified      = errors.New("oauth email not verified")
	ErrOAuthAccountLinked         = errors.New("oauth account already linked to another user")
	ErrOAuthMalformedResponse     = errors.New("malformed response from oauth provider")
	ErrOAuthProviderUnavailable   = errors.New("oauth provider is temporarily unavailable")
	ErrOAuthTokenRevoked          = errors.New("oauth token has been revoked by provider")
	ErrOAuthInsufficientScope     = errors.New("insufficient oauth scope to retrieve required data")
	ErrOAuthRateLimited           = errors.New("oauth provider rate limit exceeded")
	ErrOAuthProviderNotFound      = errors.New("oauth provider not found")
	ErrOAuthCannotUnlinkLast      = errors.New("cannot unlink last authentication method")
	ErrOAuthProviderAlreadyLinked = errors.New("oauth provider already linked")
)

// OAuthProvider represents a supported OAuth provider
type OAuthProvider string

const (
	OAuthProviderGitHub  OAuthProvider = "github"
	OAuthProviderGoogle  OAuthProvider = "google"
	OAuthProviderDiscord OAuthProvider = "discord"
)

// OAuthConfig holds configuration for an OAuth provider
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Scopes       []string
}

// OAuthProviderConfig holds configuration for all OAuth providers
type OAuthProviderConfig struct {
	GitHub  *OAuthConfig
	Google  *OAuthConfig
	Discord *OAuthConfig
}

// OAuthUserInfo represents user info returned from OAuth provider
type OAuthUserInfo struct {
	Provider      OAuthProvider
	ProviderID    string
	Email         string
	EmailVerified bool
	Username      string
	DisplayName   string
	AvatarURL     string
}

// OAuthState represents the state stored during OAuth flow
type OAuthState struct {
	Provider  OAuthProvider `json:"provider"`
	State     string        `json:"state"`
	Nonce     string        `json:"nonce"`
	CreatedAt time.Time     `json:"created_at"`
	// LinkUserID is set when linking an existing account
	LinkUserID *uuid.UUID `json:"link_user_id,omitempty"`
}

// OAuthService handles OAuth authentication flows
type OAuthService struct {
	config     *OAuthProviderConfig
	userRepo   UserRepository
	oauthRepo  OAuthRepository
	cache      CacheService
	jwtService *auth.JWTService
	httpClient *http.Client
}

// NewOAuthService creates a new OAuth service
func NewOAuthService(
	config *OAuthProviderConfig,
	userRepo UserRepository,
	cache CacheService,
	jwtService *auth.JWTService,
) *OAuthService {
	return &OAuthService{
		config:     config,
		userRepo:   userRepo,
		cache:      cache,
		jwtService: jwtService,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// NewOAuthServiceWithRepo creates a new OAuth service with repository support
func NewOAuthServiceWithRepo(
	config *OAuthProviderConfig,
	userRepo UserRepository,
	oauthRepo OAuthRepository,
	cache CacheService,
	jwtService *auth.JWTService,
) *OAuthService {
	return &OAuthService{
		config:     config,
		userRepo:   userRepo,
		oauthRepo:  oauthRepo,
		cache:      cache,
		jwtService: jwtService,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// GetAuthorizationURL generates the OAuth authorization URL for a provider
func (s *OAuthService) GetAuthorizationURL(ctx context.Context, provider OAuthProvider, linkUserID *uuid.UUID) (string, error) {
	// Ensure cache is configured for state storage
	if s.cache == nil {
		return "", fmt.Errorf("OAuth service not properly configured: cache is required")
	}

	config, err := s.getProviderConfig(provider)
	if err != nil {
		return "", err
	}

	// Generate state and nonce for CSRF protection
	state, err := generateRandomString(32)
	if err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}

	nonce, err := generateRandomString(16)
	if err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Store state in cache
	oauthState := &OAuthState{
		Provider:   provider,
		State:      state,
		Nonce:      nonce,
		CreatedAt:  time.Now(),
		LinkUserID: linkUserID,
	}

	stateData, err := json.Marshal(oauthState)
	if err != nil {
		return "", fmt.Errorf("failed to marshal state: %w", err)
	}

	// Store state with 10-minute TTL
	cacheKey := fmt.Sprintf("oauth_state:%s", state)
	if err := s.cache.Set(ctx, cacheKey, stateData, 10*time.Minute); err != nil {
		return "", fmt.Errorf("failed to store state: %w", err)
	}

	// Build authorization URL based on provider
	var authURL string
	switch provider {
	case OAuthProviderGitHub:
		authURL = s.buildGitHubAuthURL(config, state)
	case OAuthProviderGoogle:
		authURL = s.buildGoogleAuthURL(config, state, nonce)
	case OAuthProviderDiscord:
		authURL = s.buildDiscordAuthURL(config, state)
	default:
		return "", ErrOAuthProviderNotSupported
	}

	return authURL, nil
}

// HandleCallback processes the OAuth callback
func (s *OAuthService) HandleCallback(ctx context.Context, provider OAuthProvider, code, state string) (*models.User, *AuthTokens, error) {
	// Validate state
	oauthState, err := s.validateState(ctx, state)
	if err != nil {
		return nil, nil, err
	}

	// Verify provider matches
	if oauthState.Provider != provider {
		return nil, nil, ErrOAuthStateMismatch
	}

	config, err := s.getProviderConfig(provider)
	if err != nil {
		return nil, nil, err
	}

	// Exchange code for token
	token, err := s.exchangeCode(ctx, provider, config, code)
	if err != nil {
		return nil, nil, err
	}

	// Get user info from provider
	userInfo, err := s.getUserInfo(ctx, provider, token)
	if err != nil {
		return nil, nil, err
	}

	// Require verified email
	if !userInfo.EmailVerified {
		return nil, nil, ErrOAuthEmailNotVerified
	}

	// Check if this OAuth account is already linked to a user
	if s.oauthRepo != nil {
		existingProvider, err := s.oauthRepo.GetByProviderUserID(ctx, string(provider), userInfo.ProviderID)
		if err == nil && existingProvider != nil {
			// OAuth account already linked - this is a login
			if oauthState.LinkUserID != nil && existingProvider.UserID != *oauthState.LinkUserID {
				// Trying to link to a different user
				return nil, nil, ErrOAuthAccountLinked
			}

			// Get the linked user
			user, err := s.userRepo.GetByID(ctx, existingProvider.UserID)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to get linked user: %w", err)
			}
			if user == nil {
				return nil, nil, ErrUserNotFound
			}

			// Update OAuth provider info (email, avatar, etc. might have changed)
			existingProvider.Email = userInfo.Email
			existingProvider.Username = &userInfo.Username
			existingProvider.DisplayName = &userInfo.DisplayName
			if userInfo.AvatarURL != "" {
				existingProvider.AvatarURL = &userInfo.AvatarURL
			}
			_ = s.oauthRepo.Update(ctx, existingProvider)

			// Generate JWT tokens
			isAdmin := user.Flags&models.UserFlagStaff != 0
			accessToken, refreshToken, err := s.jwtService.GenerateTokenPairWithAdmin(user.ID, user.Username, isAdmin)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to generate tokens: %w", err)
			}

			return user, &AuthTokens{
				AccessToken:  accessToken,
				RefreshToken: refreshToken,
				ExpiresIn:    s.jwtService.GetExpirySeconds(),
			}, nil
		}
	}

	// Find or create user
	user, err := s.findOrCreateUser(ctx, userInfo, oauthState.LinkUserID)
	if err != nil {
		return nil, nil, err
	}

	// Store OAuth provider link
	if s.oauthRepo != nil {
		oauthProviderRecord := &models.OAuthProvider{
			ID:             uuid.New(),
			UserID:         user.ID,
			Provider:       string(provider),
			ProviderUserID: userInfo.ProviderID,
			Email:          userInfo.Email,
			Username:       &userInfo.Username,
			DisplayName:    &userInfo.DisplayName,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		if userInfo.AvatarURL != "" {
			oauthProviderRecord.AvatarURL = &userInfo.AvatarURL
		}

		if err := s.oauthRepo.Create(ctx, oauthProviderRecord); err != nil {
			// Log but don't fail - the user was still created/found
			// This could happen if there's a race condition
		}
	}

	// Generate JWT tokens
	isAdmin := user.Flags&models.UserFlagStaff != 0
	accessToken, refreshToken, err := s.jwtService.GenerateTokenPairWithAdmin(user.ID, user.Username, isAdmin)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	tokens := &AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    s.jwtService.GetExpirySeconds(),
	}

	return user, tokens, nil
}

// IsProviderEnabled checks if a provider is configured
func (s *OAuthService) IsProviderEnabled(provider OAuthProvider) bool {
	config, err := s.getProviderConfig(provider)
	return err == nil && config.ClientID != "" && config.ClientSecret != ""
}

// Internal methods

func (s *OAuthService) getProviderConfig(provider OAuthProvider) (*OAuthConfig, error) {
	switch provider {
	case OAuthProviderGitHub:
		if s.config.GitHub == nil {
			return nil, ErrOAuthProviderNotSupported
		}
		return s.config.GitHub, nil
	case OAuthProviderGoogle:
		if s.config.Google == nil {
			return nil, ErrOAuthProviderNotSupported
		}
		return s.config.Google, nil
	case OAuthProviderDiscord:
		if s.config.Discord == nil {
			return nil, ErrOAuthProviderNotSupported
		}
		return s.config.Discord, nil
	default:
		return nil, ErrOAuthProviderNotSupported
	}
}

func (s *OAuthService) validateState(ctx context.Context, state string) (*OAuthState, error) {
	// Check if cache is configured
	if s.cache == nil {
		return nil, ErrOAuthStateExpired
	}

	cacheKey := fmt.Sprintf("oauth_state:%s", state)

	data, err := s.cache.Get(ctx, cacheKey)
	if err != nil {
		return nil, ErrOAuthStateExpired
	}

	var oauthState OAuthState
	if err := json.Unmarshal(data, &oauthState); err != nil {
		return nil, ErrOAuthStateMismatch
	}

	// Check expiry (10 minutes)
	if time.Since(oauthState.CreatedAt) > 10*time.Minute {
		_ = s.cache.Delete(ctx, cacheKey)
		return nil, ErrOAuthStateExpired
	}

	// Delete state to prevent replay
	_ = s.cache.Delete(ctx, cacheKey)

	return &oauthState, nil
}

// Authorization URL builders

func (s *OAuthService) buildGitHubAuthURL(config *OAuthConfig, state string) string {
	params := url.Values{
		"client_id":    {config.ClientID},
		"redirect_uri": {config.RedirectURI},
		"scope":        {strings.Join(config.Scopes, " ")},
		"state":        {state},
	}
	return "https://github.com/login/oauth/authorize?" + params.Encode()
}

func (s *OAuthService) buildGoogleAuthURL(config *OAuthConfig, state, nonce string) string {
	params := url.Values{
		"client_id":     {config.ClientID},
		"redirect_uri":  {config.RedirectURI},
		"response_type": {"code"},
		"scope":         {strings.Join(config.Scopes, " ")},
		"state":         {state},
		"nonce":         {nonce},
		"access_type":   {"offline"},
		"prompt":        {"consent"},
	}
	return "https://accounts.google.com/o/oauth2/v2/auth?" + params.Encode()
}

func (s *OAuthService) buildDiscordAuthURL(config *OAuthConfig, state string) string {
	params := url.Values{
		"client_id":     {config.ClientID},
		"redirect_uri":  {config.RedirectURI},
		"response_type": {"code"},
		"scope":         {strings.Join(config.Scopes, " ")},
		"state":         {state},
	}
	return "https://discord.com/api/oauth2/authorize?" + params.Encode()
}

// Token exchange

func (s *OAuthService) exchangeCode(ctx context.Context, provider OAuthProvider, config *OAuthConfig, code string) (retToken string, retErr error) {
	var tokenURL string
	var body url.Values

	switch provider {
	case OAuthProviderGitHub:
		tokenURL = "https://github.com/login/oauth/access_token"
		body = url.Values{
			"client_id":     {config.ClientID},
			"client_secret": {config.ClientSecret},
			"code":          {code},
			"redirect_uri":  {config.RedirectURI},
		}
	case OAuthProviderGoogle:
		tokenURL = "https://oauth2.googleapis.com/token"
		body = url.Values{
			"client_id":     {config.ClientID},
			"client_secret": {config.ClientSecret},
			"code":          {code},
			"redirect_uri":  {config.RedirectURI},
			"grant_type":    {"authorization_code"},
		}
	case OAuthProviderDiscord:
		tokenURL = "https://discord.com/api/oauth2/token"
		body = url.Values{
			"client_id":     {config.ClientID},
			"client_secret": {config.ClientSecret},
			"code":          {code},
			"redirect_uri":  {config.RedirectURI},
			"grant_type":    {"authorization_code"},
		}
	default:
		return "", ErrOAuthProviderNotSupported
	}

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(body.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: network error - %v", ErrOAuthProviderUnavailable, err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil && retErr == nil {
			retErr = fmt.Errorf("failed to close response body: %w", err)
		}
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("%w: failed to read response body", ErrOAuthMalformedResponse)
	}

	// Handle non-200 status codes with detailed error messages
	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			return "", fmt.Errorf("%w: invalid client credentials or authorization code", ErrOAuthCodeExchange)
		case http.StatusBadRequest:
			// Try to parse error details from response
			var errResp struct {
				Error            string `json:"error"`
				ErrorDescription string `json:"error_description"`
			}
			if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != "" {
				if errResp.Error == "invalid_grant" {
					return "", fmt.Errorf("%w: authorization code expired, already used, or invalid", ErrOAuthCodeExchange)
				}
				return "", fmt.Errorf("%w: %s - %s", ErrOAuthCodeExchange, errResp.Error, errResp.ErrorDescription)
			}
			return "", fmt.Errorf("%w: bad request to token endpoint", ErrOAuthCodeExchange)
		case http.StatusTooManyRequests:
			return "", fmt.Errorf("%w: please retry later", ErrOAuthRateLimited)
		case http.StatusServiceUnavailable, http.StatusBadGateway, http.StatusGatewayTimeout:
			return "", fmt.Errorf("%w: %s (status %d)", ErrOAuthProviderUnavailable, provider, resp.StatusCode)
		default:
			return "", fmt.Errorf("%w: unexpected status %d from %s", ErrOAuthCodeExchange, resp.StatusCode, provider)
		}
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		Error        string `json:"error"`
		ErrorDesc    string `json:"error_description"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}

	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return "", fmt.Errorf("%w: invalid JSON in token response from %s", ErrOAuthMalformedResponse, provider)
	}

	if tokenResp.Error != "" {
		if tokenResp.Error == "access_denied" {
			return "", fmt.Errorf("%w: user denied access or permissions were revoked", ErrOAuthTokenRevoked)
		}
		return "", fmt.Errorf("%w: %s - %s", ErrOAuthCodeExchange, tokenResp.Error, tokenResp.ErrorDesc)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("%w: empty access_token in response from %s", ErrOAuthMalformedResponse, provider)
	}

	return tokenResp.AccessToken, nil
}

// User info retrieval

func (s *OAuthService) getUserInfo(ctx context.Context, provider OAuthProvider, accessToken string) (*OAuthUserInfo, error) {
	switch provider {
	case OAuthProviderGitHub:
		return s.getGitHubUserInfo(ctx, accessToken)
	case OAuthProviderGoogle:
		return s.getGoogleUserInfo(ctx, accessToken)
	case OAuthProviderDiscord:
		return s.getDiscordUserInfo(ctx, accessToken)
	default:
		return nil, ErrOAuthProviderNotSupported
	}
}

func (s *OAuthService) getGitHubUserInfo(ctx context.Context, accessToken string) (retInfo *OAuthUserInfo, retErr error) {
	// Get user info
	userReq, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub user request: %w", err)
	}
	userReq.Header.Set("Authorization", "Bearer "+accessToken)
	userReq.Header.Set("Accept", "application/vnd.github+json")

	userResp, err := s.httpClient.Do(userReq)
	if err != nil {
		return nil, fmt.Errorf("%w: GitHub API unavailable - %v", ErrOAuthProviderUnavailable, err)
	}
	defer func() {
		if err := userResp.Body.Close(); err != nil && retErr == nil {
			retErr = fmt.Errorf("failed to close response body: %w", err)
		}
	}()

	// Handle non-200 status codes
	if userResp.StatusCode != http.StatusOK {
		switch userResp.StatusCode {
		case http.StatusUnauthorized:
			return nil, fmt.Errorf("%w: access token expired or revoked", ErrOAuthTokenRevoked)
		case http.StatusForbidden:
			// Check for rate limiting or scope issues
			if userResp.Header.Get("X-RateLimit-Remaining") == "0" {
				return nil, fmt.Errorf("%w: GitHub rate limit exceeded", ErrOAuthRateLimited)
			}
			return nil, fmt.Errorf("%w: missing required scope 'read:user'", ErrOAuthInsufficientScope)
		case http.StatusTooManyRequests:
			return nil, fmt.Errorf("%w: GitHub rate limit exceeded", ErrOAuthRateLimited)
		case http.StatusServiceUnavailable, http.StatusBadGateway, http.StatusGatewayTimeout:
			return nil, fmt.Errorf("%w: GitHub API temporarily unavailable", ErrOAuthProviderUnavailable)
		default:
			return nil, fmt.Errorf("%w: GitHub returned status %d", ErrOAuthUserInfo, userResp.StatusCode)
		}
	}

	var ghUser struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}

	if err := json.NewDecoder(userResp.Body).Decode(&ghUser); err != nil {
		return nil, fmt.Errorf("%w: invalid JSON from GitHub user endpoint", ErrOAuthMalformedResponse)
	}

	// Validate required fields
	if ghUser.ID == 0 {
		return nil, fmt.Errorf("%w: missing user ID from GitHub", ErrOAuthMalformedResponse)
	}
	if ghUser.Login == "" {
		return nil, fmt.Errorf("%w: missing username from GitHub", ErrOAuthMalformedResponse)
	}

	// Get verified email if not in user response
	email := ghUser.Email
	emailVerified := email != ""

	if email == "" {
		// Fetch emails endpoint
		emailReq, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user/emails", nil)
		if err != nil {
			return nil, err
		}
		emailReq.Header.Set("Authorization", "Bearer "+accessToken)
		emailReq.Header.Set("Accept", "application/vnd.github+json")

		emailResp, err := s.httpClient.Do(emailReq)
		if err != nil {
			return nil, fmt.Errorf("failed to get GitHub emails: %w", err)
		}
		defer func() {
			if err := emailResp.Body.Close(); err != nil && retErr == nil {
				retErr = fmt.Errorf("failed to close response body: %w", err)
			}
		}()

		var emails []struct {
			Email    string `json:"email"`
			Primary  bool   `json:"primary"`
			Verified bool   `json:"verified"`
		}

		if err := json.NewDecoder(emailResp.Body).Decode(&emails); err == nil {
			// Find primary verified email
			for _, e := range emails {
				if e.Primary && e.Verified {
					email = e.Email
					emailVerified = true
					break
				}
			}
			// Fallback to any verified email
			if email == "" {
				for _, e := range emails {
					if e.Verified {
						email = e.Email
						emailVerified = true
						break
					}
				}
			}
		}
	}

	displayName := ghUser.Name
	if displayName == "" {
		displayName = ghUser.Login
	}

	return &OAuthUserInfo{
		Provider:      OAuthProviderGitHub,
		ProviderID:    fmt.Sprintf("%d", ghUser.ID),
		Email:         email,
		EmailVerified: emailVerified,
		Username:      ghUser.Login,
		DisplayName:   displayName,
		AvatarURL:     ghUser.AvatarURL,
	}, nil
}

func (s *OAuthService) getGoogleUserInfo(ctx context.Context, accessToken string) (retInfo *OAuthUserInfo, retErr error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Google user request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: Google API unavailable - %v", ErrOAuthProviderUnavailable, err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil && retErr == nil {
			retErr = fmt.Errorf("failed to close response body: %w", err)
		}
	}()

	// Handle non-200 status codes
	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			return nil, fmt.Errorf("%w: Google access token expired or revoked", ErrOAuthTokenRevoked)
		case http.StatusForbidden:
			return nil, fmt.Errorf("%w: missing required Google OAuth scope", ErrOAuthInsufficientScope)
		case http.StatusTooManyRequests:
			return nil, fmt.Errorf("%w: Google API rate limit exceeded", ErrOAuthRateLimited)
		case http.StatusServiceUnavailable, http.StatusBadGateway, http.StatusGatewayTimeout:
			return nil, fmt.Errorf("%w: Google API temporarily unavailable", ErrOAuthProviderUnavailable)
		default:
			return nil, fmt.Errorf("%w: Google returned status %d", ErrOAuthUserInfo, resp.StatusCode)
		}
	}

	var googleUser struct {
		ID            string `json:"id"`
		Email         string `json:"email"`
		VerifiedEmail bool   `json:"verified_email"`
		Name          string `json:"name"`
		GivenName     string `json:"given_name"`
		Picture       string `json:"picture"`
		Error         *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&googleUser); err != nil {
		return nil, fmt.Errorf("%w: invalid JSON from Google userinfo endpoint", ErrOAuthMalformedResponse)
	}

	// Check for error in response body (Google sometimes returns 200 with error)
	if googleUser.Error != nil {
		return nil, fmt.Errorf("%w: Google API error - %s", ErrOAuthUserInfo, googleUser.Error.Message)
	}

	// Validate required fields
	if googleUser.ID == "" {
		return nil, fmt.Errorf("%w: missing user ID from Google", ErrOAuthMalformedResponse)
	}
	if googleUser.Email == "" {
		return nil, fmt.Errorf("%w: missing email from Google (ensure 'email' scope is requested)", ErrOAuthInsufficientScope)
	}

	// Generate username from email prefix, sanitizing for username rules
	emailParts := strings.Split(googleUser.Email, "@")
	username := emailParts[0]
	// Limit username length to 32 characters
	if len(username) > 32 {
		username = username[:32]
	}

	return &OAuthUserInfo{
		Provider:      OAuthProviderGoogle,
		ProviderID:    googleUser.ID,
		Email:         googleUser.Email,
		EmailVerified: googleUser.VerifiedEmail,
		Username:      username,
		DisplayName:   googleUser.Name,
		AvatarURL:     googleUser.Picture,
	}, nil
}

func (s *OAuthService) getDiscordUserInfo(ctx context.Context, accessToken string) (retInfo *OAuthUserInfo, retErr error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://discord.com/api/users/@me", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Discord user request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: Discord API unavailable - %v", ErrOAuthProviderUnavailable, err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil && retErr == nil {
			retErr = fmt.Errorf("failed to close response body: %w", err)
		}
	}()

	// Handle non-200 status codes
	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			return nil, fmt.Errorf("%w: Discord access token expired or revoked", ErrOAuthTokenRevoked)
		case http.StatusForbidden:
			return nil, fmt.Errorf("%w: missing required Discord OAuth scope", ErrOAuthInsufficientScope)
		case http.StatusTooManyRequests:
			retryAfter := resp.Header.Get("Retry-After")
			return nil, fmt.Errorf("%w: Discord rate limit exceeded (retry after %s seconds)", ErrOAuthRateLimited, retryAfter)
		case http.StatusServiceUnavailable, http.StatusBadGateway, http.StatusGatewayTimeout:
			return nil, fmt.Errorf("%w: Discord API temporarily unavailable", ErrOAuthProviderUnavailable)
		default:
			return nil, fmt.Errorf("%w: Discord returned status %d", ErrOAuthUserInfo, resp.StatusCode)
		}
	}

	var discordUser struct {
		ID            string `json:"id"`
		Username      string `json:"username"`
		GlobalName    string `json:"global_name"`
		Email         string `json:"email"`
		Verified      bool   `json:"verified"`
		Avatar        string `json:"avatar"`
		Discriminator string `json:"discriminator"`
		Message       string `json:"message,omitempty"` // Error message from Discord
		Code          int    `json:"code,omitempty"`    // Discord error code
	}

	if err := json.NewDecoder(resp.Body).Decode(&discordUser); err != nil {
		return nil, fmt.Errorf("%w: invalid JSON from Discord users endpoint", ErrOAuthMalformedResponse)
	}

	// Check for error in response body
	if discordUser.Code != 0 {
		return nil, fmt.Errorf("%w: Discord API error (code %d) - %s", ErrOAuthUserInfo, discordUser.Code, discordUser.Message)
	}

	// Validate required fields
	if discordUser.ID == "" {
		return nil, fmt.Errorf("%w: missing user ID from Discord", ErrOAuthMalformedResponse)
	}
	if discordUser.Username == "" {
		return nil, fmt.Errorf("%w: missing username from Discord", ErrOAuthMalformedResponse)
	}
	if discordUser.Email == "" {
		return nil, fmt.Errorf("%w: missing email from Discord (ensure 'email' scope is requested)", ErrOAuthInsufficientScope)
	}

	// Build avatar URL
	var avatarURL string
	if discordUser.Avatar != "" {
		// Check if it's an animated avatar (starts with "a_")
		ext := "png"
		if strings.HasPrefix(discordUser.Avatar, "a_") {
			ext = "gif"
		}
		avatarURL = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.%s", discordUser.ID, discordUser.Avatar, ext)
	}

	displayName := discordUser.GlobalName
	if displayName == "" {
		displayName = discordUser.Username
	}

	return &OAuthUserInfo{
		Provider:      OAuthProviderDiscord,
		ProviderID:    discordUser.ID,
		Email:         discordUser.Email,
		EmailVerified: discordUser.Verified,
		Username:      discordUser.Username,
		DisplayName:   displayName,
		AvatarURL:     avatarURL,
	}, nil
}

// findOrCreateUser finds an existing user by email or creates a new one
func (s *OAuthService) findOrCreateUser(ctx context.Context, info *OAuthUserInfo, linkUserID *uuid.UUID) (*models.User, error) {
	// If linking to existing account
	if linkUserID != nil {
		user, err := s.userRepo.GetByID(ctx, *linkUserID)
		if err != nil {
			return nil, fmt.Errorf("failed to get user for linking: %w", err)
		}
		if user == nil {
			return nil, ErrUserNotFound
		}
		return user, nil
	}

	// Try to find existing user by email
	user, err := s.userRepo.GetByEmail(ctx, info.Email)
	if err == nil && user != nil {
		// Existing user found - return it
		return user, nil
	}
	if err != nil && !errors.Is(err, ErrUserNotFound) {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}

	// Create new user
	username := info.Username

	// Ensure username is unique by appending random suffix if needed
	for i := 0; i < 5; i++ {
		existing, _ := s.userRepo.GetByUsername(ctx, username)
		if existing == nil {
			break
		}
		// Username taken, append random suffix
		suffix, _ := generateRandomString(4)
		username = info.Username + "_" + suffix
	}

	newUser := &models.User{
		ID:          uuid.New(),
		Email:       info.Email,
		Username:    username,
		DisplayName: &info.DisplayName,
		Verified:    true, // OAuth emails are already verified
		Status:      models.StatusOffline,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Set avatar if provided
	if info.AvatarURL != "" {
		newUser.AvatarURL = &info.AvatarURL
	}

	if err := s.userRepo.Create(ctx, newUser); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return newUser, nil
}

// GetLinkedProviders returns all OAuth providers linked to a user
func (s *OAuthService) GetLinkedProviders(ctx context.Context, userID uuid.UUID) ([]*models.OAuthProvider, error) {
	if s.oauthRepo == nil {
		return nil, nil
	}
	return s.oauthRepo.GetByUserID(ctx, userID)
}

// GetLinkedProvider returns a specific OAuth provider for a user
func (s *OAuthService) GetLinkedProvider(ctx context.Context, userID uuid.UUID, provider OAuthProvider) (*models.OAuthProvider, error) {
	if s.oauthRepo == nil {
		return nil, ErrOAuthProviderNotFound
	}
	return s.oauthRepo.GetByUserAndProvider(ctx, userID, string(provider))
}

// UnlinkProvider removes an OAuth provider link from a user
func (s *OAuthService) UnlinkProvider(ctx context.Context, userID uuid.UUID, provider OAuthProvider) error {
	if s.oauthRepo == nil {
		return ErrOAuthProviderNotSupported
	}

	// Check if the provider is linked
	_, err := s.oauthRepo.GetByUserAndProvider(ctx, userID, string(provider))
	if err != nil {
		return err
	}

	// Check if user has a password or other OAuth providers
	// to ensure they don't lock themselves out
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return ErrUserNotFound
	}

	// Count OAuth providers
	count, err := s.oauthRepo.CountByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to count OAuth providers: %w", err)
	}

	// If no password and this is the only OAuth provider, deny
	if user.PasswordHash == "" && count <= 1 {
		return ErrOAuthCannotUnlinkLast
	}

	// Delete the link
	return s.oauthRepo.DeleteByUserAndProvider(ctx, userID, string(provider))
}

// GetLinkAuthorizationURL generates an OAuth URL for linking to an existing account
func (s *OAuthService) GetLinkAuthorizationURL(ctx context.Context, userID uuid.UUID, provider OAuthProvider) (string, error) {
	if s.oauthRepo == nil {
		return "", ErrOAuthProviderNotSupported
	}

	// Check if already linked
	_, err := s.oauthRepo.GetByUserAndProvider(ctx, userID, string(provider))
	if err == nil {
		return "", ErrOAuthProviderAlreadyLinked
	}
	if !errors.Is(err, ErrOAuthProviderNotFound) {
		return "", err
	}

	// Generate authorization URL with link user ID
	return s.GetAuthorizationURL(ctx, provider, &userID)
}

// GetEnabledProviders returns a list of enabled OAuth providers
func (s *OAuthService) GetEnabledProviders() []OAuthProvider {
	var providers []OAuthProvider
	if s.IsProviderEnabled(OAuthProviderGitHub) {
		providers = append(providers, OAuthProviderGitHub)
	}
	if s.IsProviderEnabled(OAuthProviderGoogle) {
		providers = append(providers, OAuthProviderGoogle)
	}
	if s.IsProviderEnabled(OAuthProviderDiscord) {
		providers = append(providers, OAuthProviderDiscord)
	}
	return providers
}

// Helper functions

func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

var (
	ErrSessionNotFound      = errors.New("session not found")
	ErrInvalidRefreshToken  = errors.New("invalid refresh token")
	ErrTokenReused          = errors.New("refresh token reuse detected")
	ErrTokenExpired         = errors.New("refresh token expired")
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
	// Note: ErrUnauthorized is defined in saved_messages_service.go
)

// SessionRepository defines the interface for session persistence
type SessionRepository interface {
	// Session operations
	CreateSession(ctx context.Context, session *models.Session) error
	GetSessionByID(ctx context.Context, id uuid.UUID) (*models.Session, error)
	GetUserSessions(ctx context.Context, userID uuid.UUID) ([]*models.Session, error)
	UpdateSessionActivity(ctx context.Context, sessionID uuid.UUID) error
	SetCurrentSession(ctx context.Context, userID, sessionID uuid.UUID) error
	RevokeSession(ctx context.Context, sessionID uuid.UUID) error
	RevokeAllUserSessions(ctx context.Context, userID uuid.UUID, exceptSessionID *uuid.UUID) error

	// Refresh token operations
	CreateRefreshToken(ctx context.Context, token *models.RefreshToken) error
	GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error)
	MarkRefreshTokenUsed(ctx context.Context, tokenID uuid.UUID) error
	RevokeTokenFamily(ctx context.Context, familyID uuid.UUID) error
	RevokeAllUserTokens(ctx context.Context, userID uuid.UUID, exceptFamilyID *uuid.UUID) error
	RotateRefreshToken(ctx context.Context, oldTokenID uuid.UUID, newToken *models.RefreshToken) error
}

// SessionService handles session and token management
type SessionService interface {
	// Session management
	CreateSession(ctx context.Context, userID uuid.UUID, userAgent, ipAddress string) (*models.Session, string, error)
	GetUserSessions(ctx context.Context, userID uuid.UUID, currentSessionID *uuid.UUID) ([]*models.SessionResponse, error)
	RevokeSession(ctx context.Context, userID, sessionID uuid.UUID) error
	RevokeAllSessions(ctx context.Context, userID uuid.UUID, exceptSessionID *uuid.UUID) error
	UpdateSessionActivity(ctx context.Context, sessionID uuid.UUID) error

	// Token operations
	CreateRefreshToken(ctx context.Context, userID, sessionID, familyID uuid.UUID, expiry time.Duration) (string, error)
	RotateRefreshToken(ctx context.Context, oldToken string, expiry time.Duration) (*models.Session, string, error)
	ValidateRefreshToken(ctx context.Context, token string) (*models.RefreshToken, error)
}

type sessionService struct {
	sessionRepo   SessionRepository
	jwtService    *auth.JWTService
	refreshExpiry time.Duration
}

// NewSessionService creates a new session service
func NewSessionService(sessionRepo SessionRepository, jwtService *auth.JWTService, refreshExpiry time.Duration) SessionService {
	return &sessionService{
		sessionRepo:   sessionRepo,
		jwtService:    jwtService,
		refreshExpiry: refreshExpiry,
	}
}

// CreateSession creates a new session with device info
func (s *sessionService) CreateSession(ctx context.Context, userID uuid.UUID, userAgent, ipAddress string) (*models.Session, string, error) {
	// Parse device info
	deviceInfo := auth.ParseUserAgent(userAgent)

	// Generate session ID
	sessionID := uuid.New()
	familyID := models.GenerateTokenFamily()

	// Create session
	session := &models.Session{
		ID:             sessionID,
		UserID:         userID,
		TokenHash:      "", // Will be updated with refresh token hash
		DeviceName:     &deviceInfo.DeviceName,
		DeviceType:     deviceInfo.DeviceType,
		Browser:        strPtr(deviceInfo.Browser),
		BrowserVersion: strPtr(deviceInfo.BrowserVersion),
		OS:             strPtr(deviceInfo.OS),
		OSVersion:      strPtr(deviceInfo.OSVersion),
		IPAddress:      strPtr(ipAddress),
		UserAgent:      strPtr(userAgent),
		IsCurrent:      true,
		ExpiresAt:      time.Now().Add(s.refreshExpiry),
		CreatedAt:      time.Now(),
	}

	// Generate refresh token
	refreshToken, err := s.generateSecureToken()
	if err != nil {
		return nil, "", err
	}

	tokenHash := models.HashToken(refreshToken)
	session.TokenHash = tokenHash

	// Save session
	if err := s.sessionRepo.CreateSession(ctx, session); err != nil {
		return nil, "", err
	}

	// Create refresh token record
	rtRecord := &models.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: tokenHash,
		FamilyID:  familyID,
		SessionID: sessionID,
		ExpiresAt: session.ExpiresAt,
		CreatedAt: time.Now(),
	}

	if err := s.sessionRepo.CreateRefreshToken(ctx, rtRecord); err != nil {
		// Clean up session if token creation fails
		if revokeErr := s.sessionRepo.RevokeSession(ctx, sessionID); revokeErr != nil {
			log.Printf("Failed to revoke session during cleanup: %v (sessionID: %v)", revokeErr, sessionID)
		}
		return nil, "", err
	}

	// Mark this session as current (unmark others)
	if err := s.sessionRepo.SetCurrentSession(ctx, userID, sessionID); err != nil {
		log.Printf("Failed to set current session: %v (userID: %v, sessionID: %v)", err, userID, sessionID)
	}

	return session, refreshToken, nil
}

// GetUserSessions retrieves all active sessions for a user
func (s *sessionService) GetUserSessions(ctx context.Context, userID uuid.UUID, currentSessionID *uuid.UUID) ([]*models.SessionResponse, error) {
	sessions, err := s.sessionRepo.GetUserSessions(ctx, userID)
	if err != nil {
		return nil, err
	}

	responses := make([]*models.SessionResponse, len(sessions))
	for i, session := range sessions {
		resp := session.ToResponse()
		// Mark the current session
		if currentSessionID != nil && session.ID == *currentSessionID {
			resp.IsCurrent = true
		}
		responses[i] = &resp
	}

	return responses, nil
}

// RevokeSession revokes a specific session
func (s *sessionService) RevokeSession(ctx context.Context, userID, sessionID uuid.UUID) error {
	// Get the session to verify ownership
	session, err := s.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return ErrSessionNotFound
	}

	if session.UserID != userID {
		return ErrUnauthorized
	}

	// Revoke the session
	if err := s.sessionRepo.RevokeSession(ctx, sessionID); err != nil {
		return err
	}

	// Get and revoke the associated token family
	// Find refresh token for this session
	// Note: We can't easily find the family without the token, so we rely on
	// the session expiry to invalidate tokens during validation
	return nil
}

// RevokeAllSessions revokes all sessions except optionally the current one
func (s *sessionService) RevokeAllSessions(ctx context.Context, userID uuid.UUID, exceptSessionID *uuid.UUID) error {
	// Revoke all sessions
	if err := s.sessionRepo.RevokeAllUserSessions(ctx, userID, exceptSessionID); err != nil {
		return err
	}

	// Revoke all refresh token families
	// If we're keeping one session, we need to find its family first
	// For simplicity, we revoke all tokens - the active one will be re-created on next refresh
	return s.sessionRepo.RevokeAllUserTokens(ctx, userID, nil)
}

// UpdateSessionActivity updates the last activity timestamp
func (s *sessionService) UpdateSessionActivity(ctx context.Context, sessionID uuid.UUID) error {
	return s.sessionRepo.UpdateSessionActivity(ctx, sessionID)
}

// CreateRefreshToken creates a new refresh token
func (s *sessionService) CreateRefreshToken(ctx context.Context, userID, sessionID, familyID uuid.UUID, expiry time.Duration) (string, error) {
	refreshToken, err := s.generateSecureToken()
	if err != nil {
		return "", err
	}

	tokenHash := models.HashToken(refreshToken)

	rtRecord := &models.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: tokenHash,
		FamilyID:  familyID,
		SessionID: sessionID,
		ExpiresAt: time.Now().Add(expiry),
		CreatedAt: time.Now(),
	}

	if err := s.sessionRepo.CreateRefreshToken(ctx, rtRecord); err != nil {
		return "", err
	}

	return refreshToken, nil
}

// RotateRefreshToken validates an old token and issues a new one
func (s *sessionService) RotateRefreshToken(ctx context.Context, oldToken string, expiry time.Duration) (*models.Session, string, error) {
	// Hash the incoming token
	tokenHash := models.HashToken(oldToken)

	// Look up the token
	rtRecord, err := s.sessionRepo.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		// Check for not found error (can be from any implementation)
		if errors.Is(err, ErrRefreshTokenNotFound) || err.Error() == "refresh token not found" {
			return nil, "", ErrInvalidRefreshToken
		}
		return nil, "", err
	}

	// Check if token has been revoked
	if rtRecord.Revoked {
		return nil, "", ErrInvalidRefreshToken
	}

	// Check if token is expired
	if time.Now().After(rtRecord.ExpiresAt) {
		return nil, "", ErrTokenExpired
	}

	// SECURITY: Check if token has already been used
	// If yes, this indicates token theft - revoke the entire family
	if rtRecord.Used {
		// Token reuse detected! Revoke the entire family
		if err := s.sessionRepo.RevokeTokenFamily(ctx, rtRecord.FamilyID); err != nil {
			// Log but continue - security check is more important
			log.Printf("SECURITY: Failed to revoke token family after reuse detection: %v (familyID: %v)", err, rtRecord.FamilyID)
		}
		// Also revoke the session
		if err := s.sessionRepo.RevokeSession(ctx, rtRecord.SessionID); err != nil {
			log.Printf("SECURITY: Failed to revoke session after reuse detection: %v (sessionID: %v)", err, rtRecord.SessionID)
		}
		return nil, "", ErrTokenReused
	}

	// Get the associated session
	session, err := s.sessionRepo.GetSessionByID(ctx, rtRecord.SessionID)
	if err != nil {
		// Session expired/revoked
		return nil, "", ErrInvalidRefreshToken
	}

	// Generate new refresh token
	newToken, err := s.generateSecureToken()
	if err != nil {
		return nil, "", err
	}

	newTokenHash := models.HashToken(newToken)

	newRTRecord := &models.RefreshToken{
		ID:        uuid.New(),
		UserID:    rtRecord.UserID,
		TokenHash: newTokenHash,
		FamilyID:  rtRecord.FamilyID, // Same family
		SessionID: rtRecord.SessionID,
		ExpiresAt: time.Now().Add(expiry),
		CreatedAt: time.Now(),
	}

	// Atomically mark old token as used and create new one
	if err := s.sessionRepo.RotateRefreshToken(ctx, rtRecord.ID, newRTRecord); err != nil {
		return nil, "", err
	}

	// Update session activity (log error but don't fail)
	if err := s.sessionRepo.UpdateSessionActivity(ctx, session.ID); err != nil {
		log.Printf("Failed to update session activity: %v (sessionID: %v)", err, session.ID)
	}

	return session, newToken, nil
}

// ValidateRefreshToken validates a refresh token without rotating
func (s *sessionService) ValidateRefreshToken(ctx context.Context, token string) (*models.RefreshToken, error) {
	tokenHash := models.HashToken(token)

	rtRecord, err := s.sessionRepo.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	if rtRecord.Revoked || rtRecord.Used {
		return nil, ErrInvalidRefreshToken
	}

	if time.Now().After(rtRecord.ExpiresAt) {
		return nil, ErrTokenExpired
	}

	return rtRecord, nil
}

// generateSecureToken generates a cryptographically secure random token
func (s *sessionService) generateSecureToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// Helper function to create string pointer
func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
