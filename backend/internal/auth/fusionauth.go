package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ghndrx/hearth/internal/models"
	"github.com/google/uuid"
)

// FusionAuthProvider implements authentication via FusionAuth
type FusionAuthProvider struct {
	config     *FusionAuthConfig
	jwtConfig  JWTConfig
	httpClient *http.Client
	userRepo   UserRepository
	sessionRepo SessionRepository
	tokenGen   TokenGenerator
}

// UserRepository for local user storage
type UserRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindByExternalID(ctx context.Context, provider, externalID string) (*models.User, error)
	Create(ctx context.Context, user *models.User) error
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// SessionRepository for session storage
type SessionRepository interface {
	Create(ctx context.Context, session *models.Session) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.Session, error)
	FindByRefreshToken(ctx context.Context, tokenHash string) (*models.Session, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Session, error)
	Update(ctx context.Context, session *models.Session) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
}

// TokenGenerator for JWT operations
type TokenGenerator interface {
	GenerateAccessToken(claims *Claims) (string, error)
	GenerateRefreshToken() (string, string, error) // returns token, hash
	ValidateAccessToken(token string) (*Claims, error)
	HashRefreshToken(token string) string
}

// NewFusionAuthProvider creates a new FusionAuth provider
func NewFusionAuthProvider(
	config *FusionAuthConfig,
	jwtConfig JWTConfig,
	userRepo UserRepository,
	sessionRepo SessionRepository,
	tokenGen TokenGenerator,
) *FusionAuthProvider {
	return &FusionAuthProvider{
		config:    config,
		jwtConfig: jwtConfig,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		tokenGen:    tokenGen,
	}
}

// Name returns the provider name
func (p *FusionAuthProvider) Name() string {
	return "FusionAuth"
}

// Type returns the provider type
func (p *FusionAuthProvider) Type() ProviderType {
	return ProviderFusionAuth
}

// Register creates a new user via FusionAuth
func (p *FusionAuthProvider) Register(ctx context.Context, req *RegisterRequest) (*AuthResult, error) {
	// Register user in FusionAuth
	faReq := map[string]interface{}{
		"registration": map[string]interface{}{
			"applicationId": p.config.ApplicationID,
		},
		"user": map[string]interface{}{
			"email":    req.Email,
			"username": req.Username,
			"password": req.Password,
		},
	}

	body, err := json.Marshal(faReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", 
		p.config.Host+"/api/user/registration", 
		strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", p.config.APIKey)
	if p.config.TenantID != "" {
		httpReq.Header.Set("X-FusionAuth-TenantId", p.config.TenantID)
	}

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call FusionAuth: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("FusionAuth registration failed: %s", string(bodyBytes))
	}

	var faResp struct {
		User struct {
			ID       string `json:"id"`
			Email    string `json:"email"`
			Username string `json:"username"`
			ImageURL string `json:"imageUrl"`
			Verified bool   `json:"verified"`
		} `json:"user"`
		Token        string `json:"token"`
		RefreshToken string `json:"refreshToken"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&faResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Sync user to local database
	user, err := p.syncUser(ctx, &faResp.User)
	if err != nil {
		return nil, fmt.Errorf("failed to sync user: %w", err)
	}

	// Create local session
	return p.createSession(ctx, user, req.UserAgent, req.IPAddress)
}

// Login authenticates via FusionAuth
func (p *FusionAuthProvider) Login(ctx context.Context, req *LoginRequest) (*AuthResult, error) {
	faReq := map[string]interface{}{
		"applicationId": p.config.ApplicationID,
		"loginId":       req.Email,
		"password":      req.Password,
	}

	if req.MFACode != "" {
		faReq["twoFactorTrustId"] = req.MFACode
	}

	body, err := json.Marshal(faReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		p.config.Host+"/api/login",
		strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if p.config.TenantID != "" {
		httpReq.Header.Set("X-FusionAuth-TenantId", p.config.TenantID)
	}

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call FusionAuth: %w", err)
	}
	defer resp.Body.Close()

	// Handle MFA challenge
	if resp.StatusCode == 242 {
		var mfaResp struct {
			TwoFactorId string `json:"twoFactorId"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&mfaResp); err != nil {
			return nil, fmt.Errorf("failed to decode MFA response: %w", err)
		}
		return &AuthResult{
			MFARequired: true,
			MFAToken:    mfaResp.TwoFactorId,
		}, nil
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("FusionAuth login failed: %s", string(bodyBytes))
	}

	var faResp struct {
		User struct {
			ID       string `json:"id"`
			Email    string `json:"email"`
			Username string `json:"username"`
			ImageURL string `json:"imageUrl"`
			Verified bool   `json:"verified"`
		} `json:"user"`
		Token        string `json:"token"`
		RefreshToken string `json:"refreshToken"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&faResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Sync user to local database
	user, err := p.syncUser(ctx, &faResp.User)
	if err != nil {
		return nil, fmt.Errorf("failed to sync user: %w", err)
	}

	// Create local session
	return p.createSession(ctx, user, req.UserAgent, req.IPAddress)
}

// Logout invalidates a session
func (p *FusionAuthProvider) Logout(ctx context.Context, sessionID uuid.UUID) error {
	return p.sessionRepo.Delete(ctx, sessionID)
}

// RefreshToken exchanges a refresh token for new tokens
func (p *FusionAuthProvider) RefreshToken(ctx context.Context, refreshToken string) (*AuthResult, error) {
	// Find session by refresh token hash
	hash := p.tokenGen.HashRefreshToken(refreshToken)
	session, err := p.sessionRepo.FindByRefreshToken(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token")
	}

	// Check expiration
	if time.Now().After(session.ExpiresAt) {
		p.sessionRepo.Delete(ctx, session.ID)
		return nil, fmt.Errorf("refresh token expired")
	}

	// Get user
	user, err := p.userRepo.FindByID(ctx, session.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	// Generate new tokens
	newRefreshToken, newHash, err := p.tokenGen.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Update session with new refresh token
	session.RefreshToken = newHash
	session.ExpiresAt = time.Now().Add(p.jwtConfig.RefreshTokenTTL)
	now := time.Now()
	session.LastUsedAt = &now
	if err := p.sessionRepo.Update(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	// Generate access token
	claims := &Claims{
		UserID:    user.ID,
		Username:  user.Username,
		SessionID: session.ID,
		Provider:  ProviderFusionAuth,
		Flags:     user.Flags,
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(p.jwtConfig.AccessTokenTTL),
		Issuer:    p.jwtConfig.Issuer,
		Audience:  p.jwtConfig.Audience,
	}

	accessToken, err := p.tokenGen.GenerateAccessToken(claims)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	return &AuthResult{
		User:         &models.PublicUser{ID: user.ID, Username: user.Username},
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int(p.jwtConfig.AccessTokenTTL.Seconds()),
		TokenType:    "Bearer",
	}, nil
}

// GetAuthorizationURL returns the FusionAuth OAuth URL
func (p *FusionAuthProvider) GetAuthorizationURL(ctx context.Context, state string) (string, error) {
	params := url.Values{
		"client_id":     {p.config.ClientID},
		"redirect_uri":  {p.config.RedirectURI},
		"response_type": {"code"},
		"scope":         {strings.Join(p.config.Scopes, " ")},
		"state":         {state},
	}

	if p.config.TenantID != "" {
		params.Set("tenantId", p.config.TenantID)
	}

	return fmt.Sprintf("%s/oauth2/authorize?%s", p.config.Host, params.Encode()), nil
}

// HandleCallback processes the OAuth callback
func (p *FusionAuthProvider) HandleCallback(ctx context.Context, code, state string) (*AuthResult, error) {
	// Exchange code for tokens
	params := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"redirect_uri": {p.config.RedirectURI},
		"client_id":    {p.config.ClientID},
		"client_secret":{p.config.ClientSecret},
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		p.config.Host+"/oauth2/token",
		strings.NewReader(params.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed: %s", string(bodyBytes))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		IDToken      string `json:"id_token"`
		UserID       string `json:"userId"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	// Get user info
	userReq, _ := http.NewRequestWithContext(ctx, "GET",
		p.config.Host+"/oauth2/userinfo", nil)
	userReq.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)

	userResp, err := p.httpClient.Do(userReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer userResp.Body.Close()

	var userInfo struct {
		Sub               string `json:"sub"`
		Email             string `json:"email"`
		PreferredUsername string `json:"preferred_username"`
		Picture           string `json:"picture"`
		EmailVerified     bool   `json:"email_verified"`
	}

	if err := json.NewDecoder(userResp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	// Sync user
	faUser := struct {
		ID       string
		Email    string
		Username string
		ImageURL string
		Verified bool
	}{
		ID:       userInfo.Sub,
		Email:    userInfo.Email,
		Username: userInfo.PreferredUsername,
		ImageURL: userInfo.Picture,
		Verified: userInfo.EmailVerified,
	}

	user, err := p.syncUser(ctx, &faUser)
	if err != nil {
		return nil, fmt.Errorf("failed to sync user: %w", err)
	}

	// Create session
	return p.createSession(ctx, user, "", "")
}

// syncUser creates or updates a local user from FusionAuth data
func (p *FusionAuthProvider) syncUser(ctx context.Context, faUser *struct {
	ID       string
	Email    string
	Username string
	ImageURL string
	Verified bool
}) (*models.User, error) {
	// Try to find existing user
	user, err := p.userRepo.FindByExternalID(ctx, "fusionauth", faUser.ID)
	if err != nil {
		// Create new user
		user = &models.User{
			ID:            uuid.New(),
			Email:         faUser.Email,
			Username:      faUser.Username,
			Discriminator: generateDiscriminator(),
			Status:        models.StatusOffline,
			Verified:      faUser.Verified,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		if faUser.ImageURL != "" {
			user.AvatarURL = &faUser.ImageURL
		}
		if err := p.userRepo.Create(ctx, user); err != nil {
			return nil, err
		}
		return user, nil
	}

	// Update existing user if sync enabled
	if p.config.SyncProfileFields {
		user.Email = faUser.Email
		user.Verified = faUser.Verified
		if faUser.ImageURL != "" {
			user.AvatarURL = &faUser.ImageURL
		}
		user.UpdatedAt = time.Now()
		if err := p.userRepo.Update(ctx, user); err != nil {
			return nil, err
		}
	}

	return user, nil
}

// createSession generates tokens and creates a session
func (p *FusionAuthProvider) createSession(ctx context.Context, user *models.User, userAgent, ipAddress string) (*AuthResult, error) {
	// Generate refresh token
	refreshToken, refreshHash, err := p.tokenGen.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Create session
	session := &models.Session{
		ID:           uuid.New(),
		UserID:       user.ID,
		RefreshToken: refreshHash,
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(p.jwtConfig.RefreshTokenTTL),
	}
	if userAgent != "" {
		session.UserAgent = &userAgent
	}
	if ipAddress != "" {
		session.IPAddress = &ipAddress
	}

	if err := p.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Generate access token
	claims := &Claims{
		UserID:    user.ID,
		Username:  user.Username,
		SessionID: session.ID,
		Provider:  ProviderFusionAuth,
		Flags:     user.Flags,
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(p.jwtConfig.AccessTokenTTL),
		Issuer:    p.jwtConfig.Issuer,
		Audience:  p.jwtConfig.Audience,
	}

	accessToken, err := p.tokenGen.GenerateAccessToken(claims)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	publicUser := user.ToPublic()
	return &AuthResult{
		User:         &publicUser,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(p.jwtConfig.AccessTokenTTL.Seconds()),
		TokenType:    "Bearer",
	}, nil
}

// Stub implementations for other Provider interface methods

func (p *FusionAuthProvider) ChangePassword(ctx context.Context, userID uuid.UUID, req *ChangePasswordRequest) error {
	// Delegate to FusionAuth
	return fmt.Errorf("password change should be done via FusionAuth")
}

func (p *FusionAuthProvider) RequestPasswordReset(ctx context.Context, email string) error {
	// Trigger FusionAuth password reset
	return fmt.Errorf("not implemented")
}

func (p *FusionAuthProvider) ConfirmPasswordReset(ctx context.Context, token, newPassword string) error {
	return fmt.Errorf("password reset handled by FusionAuth")
}

func (p *FusionAuthProvider) EnableMFA(ctx context.Context, userID uuid.UUID) (*MFASetup, error) {
	return nil, fmt.Errorf("MFA should be configured via FusionAuth")
}

func (p *FusionAuthProvider) VerifyMFA(ctx context.Context, userID uuid.UUID, code string) error {
	return fmt.Errorf("MFA verification handled during login")
}

func (p *FusionAuthProvider) DisableMFA(ctx context.Context, userID uuid.UUID) error {
	return fmt.Errorf("MFA should be disabled via FusionAuth")
}

func (p *FusionAuthProvider) GetSessions(ctx context.Context, userID uuid.UUID) ([]*models.Session, error) {
	return p.sessionRepo.FindByUserID(ctx, userID)
}

func (p *FusionAuthProvider) RevokeSession(ctx context.Context, sessionID uuid.UUID) error {
	return p.sessionRepo.Delete(ctx, sessionID)
}

func (p *FusionAuthProvider) RevokeAllSessions(ctx context.Context, userID uuid.UUID) error {
	return p.sessionRepo.DeleteByUserID(ctx, userID)
}

func (p *FusionAuthProvider) GetUser(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	return p.userRepo.FindByID(ctx, userID)
}

func (p *FusionAuthProvider) UpdateUser(ctx context.Context, userID uuid.UUID, req *models.UpdateUserRequest) (*models.User, error) {
	user, err := p.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if req.Username != nil {
		user.Username = *req.Username
	}
	if req.AvatarURL != nil {
		user.AvatarURL = req.AvatarURL
	}
	if req.BannerURL != nil {
		user.BannerURL = req.BannerURL
	}
	if req.Bio != nil {
		user.Bio = req.Bio
	}
	if req.CustomStatus != nil {
		user.CustomStatus = req.CustomStatus
	}
	user.UpdatedAt = time.Now()

	if err := p.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (p *FusionAuthProvider) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	// Delete from FusionAuth too if needed
	return p.userRepo.Delete(ctx, userID)
}

// generateDiscriminator creates a random 4-digit discriminator
func generateDiscriminator() string {
	return fmt.Sprintf("%04d", time.Now().UnixNano()%10000)
}
