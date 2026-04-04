package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"hearth/internal/models"
)

const (
	// DefaultTokenExpirySeconds is the default token expiry time in seconds (1 hour)
	DefaultTokenExpirySeconds = 3600
)

// FusionAuthProvider implements Provider for FusionAuth
type FusionAuthProvider struct {
	config *FusionAuthConfig
	client *http.Client
}

// NewFusionAuthProvider creates a new FusionAuth provider
func NewFusionAuthProvider(cfg *FusionAuthConfig) (*FusionAuthProvider, error) {
	if cfg.Host == "" {
		return nil, fmt.Errorf("fusionauth: host is required")
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("fusionauth: api_key is required")
	}
	if cfg.ApplicationID == "" {
		return nil, fmt.Errorf("fusionauth: application_id is required")
	}

	host := strings.TrimRight(cfg.Host, "/")
	cfg.Host = host

	return &FusionAuthProvider{
		config: cfg,
		client: &http.Client{Timeout: 15 * time.Second},
	}, nil
}

func (p *FusionAuthProvider) Name() string       { return "FusionAuth" }
func (p *FusionAuthProvider) Type() ProviderType { return ProviderFusionAuth }

// doRequest executes an HTTP request against the FusionAuth API.
func (p *FusionAuthProvider) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("fusionauth: marshal request: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, p.config.Host+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("fusionauth: create request: %w", err)
	}

	req.Header.Set("Authorization", p.config.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if p.config.TenantID != "" {
		req.Header.Set("X-FusionAuth-TenantId", p.config.TenantID)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fusionauth: request failed: %w", err)
	}

	return resp, nil
}

// doFormRequest executes a form-encoded POST (used for OAuth2 token endpoint).
func (p *FusionAuthProvider) doFormRequest(ctx context.Context, path string, values url.Values) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.config.Host+path, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, fmt.Errorf("fusionauth: create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fusionauth: request failed: %w", err)
	}

	return resp, nil
}

func decodeBody(resp *http.Response, v interface{}) error {
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("fusionauth: read response: %w", err)
	}
	if len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("fusionauth: decode response: %w", err)
	}
	return nil
}

func fusionAuthError(resp *http.Response) error {
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("fusionauth: failed to read error response body: %w", err)
	}

	var faErr struct {
		GeneralErrors []struct {
			Message string `json:"message"`
		} `json:"generalErrors"`
		FieldErrors map[string][]struct {
			Message string `json:"message"`
		} `json:"fieldErrors"`
	}
	if err := json.Unmarshal(data, &faErr); err == nil {
		if len(faErr.GeneralErrors) > 0 {
			return fmt.Errorf("fusionauth: %s", faErr.GeneralErrors[0].Message)
		}
		for field, errs := range faErr.FieldErrors {
			if len(errs) > 0 {
				return fmt.Errorf("fusionauth: %s: %s", field, errs[0].Message)
			}
		}
	}

	return fmt.Errorf("fusionauth: request failed with status %d", resp.StatusCode)
}

// FusionAuth API response types

type faUser struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	Username      string `json:"username"`
	FirstName     string `json:"firstName"`
	LastName      string `json:"lastName"`
	ImageURL      string `json:"imageUrl"`
	Verified      bool   `json:"verified"`
	Active        bool   `json:"active"`
	MobilePhone   string `json:"mobilePhone"`
	TwoFactor     faMFA  `json:"twoFactor"`
	InsertInstant int64  `json:"insertInstant"`
	Data          faData `json:"data"`
}

type faMFA struct {
	Methods []faMFAMethod `json:"methods"`
}

type faMFAMethod struct {
	ID          string `json:"id"`
	Method      string `json:"method"`
	Secret      string `json:"secret"`
	MobilePhone string `json:"mobilePhone"`
}

type faData struct {
	DisplayName string `json:"displayName"`
	Bio         string `json:"bio"`
	Pronouns    string `json:"pronouns"`
}

type faLoginResponse struct {
	Token                  string `json:"token"`
	RefreshToken           string `json:"refreshToken"`
	TokenExpirationInstant int64  `json:"tokenExpirationInstant"`
	User                   faUser `json:"user"`
	TwoFactorID            string `json:"twoFactorId"`
}

type faRegistrationResponse struct {
	Token                  string         `json:"token"`
	RefreshToken           string         `json:"refreshToken"`
	TokenExpirationInstant int64          `json:"tokenExpirationInstant"`
	User                   faUser         `json:"user"`
	Registration           faRegistration `json:"registration"`
}

type faRegistration struct {
	ApplicationID string `json:"applicationId"`
	ID            string `json:"id"`
	Username      string `json:"username"`
}

type faTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

type faUserResponse struct {
	User faUser `json:"user"`
}

type faRefreshTokensResponse struct {
	RefreshTokens []faRefreshToken `json:"refreshTokens"`
}

type faRefreshToken struct {
	ID            string     `json:"id"`
	ApplicationID string     `json:"applicationId"`
	InsertInstant int64      `json:"insertInstant"`
	StartInstant  int64      `json:"startInstant"`
	Token         string     `json:"token"`
	MetaData      faMetaData `json:"metaData"`
}

type faMetaData struct {
	Device struct {
		Name        string `json:"name"`
		Type        string `json:"type"`
		Description string `json:"description"`
	} `json:"device"`
	OperatingSystem     string `json:"operatingSystem"`
	OperatingSystemName string `json:"operatingSystemName"`
	UserAgent           string `json:"userAgent"`
	LastAccessInstant   int64  `json:"lastAccessInstant"`
}

type faMFAEnableResponse struct {
	Code         string `json:"code"`
	SecretBase32 string `json:"secretBase32"`
	Secret       string `json:"secret"`
}

// mapFAUserToPublic converts a FusionAuth user to Hearth's PublicUser.
func mapFAUserToPublic(faU *faUser) (*models.PublicUser, error) {
	uid, err := uuid.Parse(faU.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID from FusionAuth: %w", err)
	}

	var displayName *string
	if faU.Data.DisplayName != "" {
		displayName = &faU.Data.DisplayName
	} else if faU.FirstName != "" || faU.LastName != "" {
		dn := strings.TrimSpace(faU.FirstName + " " + faU.LastName)
		displayName = &dn
	}

	var avatarURL *string
	if faU.ImageURL != "" {
		avatarURL = &faU.ImageURL
	}

	var bio *string
	if faU.Data.Bio != "" {
		bio = &faU.Data.Bio
	}

	var pronouns *string
	if faU.Data.Pronouns != "" {
		pronouns = &faU.Data.Pronouns
	}

	return &models.PublicUser{
		ID:          uid,
		Username:    faU.Username,
		DisplayName: displayName,
		AvatarURL:   avatarURL,
		Bio:         bio,
		Pronouns:    pronouns,
		Status:      models.StatusOffline,
		Flags:       0,
	}, nil
}

// mapFAUserToModel converts a FusionAuth user to Hearth's internal User model.
func mapFAUserToModel(faU *faUser) (*models.User, error) {
	uid, err := uuid.Parse(faU.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID from FusionAuth: %w", err)
	}

	var displayName *string
	if faU.Data.DisplayName != "" {
		displayName = &faU.Data.DisplayName
	} else if faU.FirstName != "" || faU.LastName != "" {
		dn := strings.TrimSpace(faU.FirstName + " " + faU.LastName)
		displayName = &dn
	}

	var avatarURL *string
	if faU.ImageURL != "" {
		avatarURL = &faU.ImageURL
	}

	var bio *string
	if faU.Data.Bio != "" {
		bio = &faU.Data.Bio
	}

	var pronouns *string
	if faU.Data.Pronouns != "" {
		pronouns = &faU.Data.Pronouns
	}

	createdAt := time.UnixMilli(faU.InsertInstant)

	return &models.User{
		ID:          uid,
		Email:       faU.Email,
		Username:    faU.Username,
		DisplayName: displayName,
		AvatarURL:   avatarURL,
		Bio:         bio,
		Pronouns:    pronouns,
		Status:      models.StatusOffline,
		MFAEnabled:  len(faU.TwoFactor.Methods) > 0,
		Verified:    faU.Verified,
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
	}, nil
}

func tokenExpiresIn(expirationInstant int64) int {
	if expirationInstant == 0 {
		return DefaultTokenExpirySeconds
	}
	remaining := time.UnixMilli(expirationInstant).Sub(time.Now())
	secs := int(remaining.Seconds())
	if secs < 0 {
		return 0
	}
	return secs
}

// Register creates a new user and registers them with the FusionAuth application.
func (p *FusionAuthProvider) Register(ctx context.Context, req *RegisterRequest) (*AuthResult, error) {
	body := map[string]interface{}{
		"user": map[string]interface{}{
			"email":    req.Email,
			"username": req.Username,
			"password": req.Password,
		},
		"registration": map[string]interface{}{
			"applicationId": p.config.ApplicationID,
			"username":      req.Username,
		},
		"sendSetPasswordEmail": false,
		"skipVerification":     false,
	}

	resp, err := p.doRequest(ctx, http.MethodPost, "/api/user/registration", body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fusionAuthError(resp)
	}

	var faResp faRegistrationResponse
	if err := decodeBody(resp, &faResp); err != nil {
		return nil, err
	}

	pub, err := mapFAUserToPublic(&faResp.User)
	if err != nil {
		return nil, err
	}
	return &AuthResult{
		User:         pub,
		AccessToken:  faResp.Token,
		RefreshToken: faResp.RefreshToken,
		ExpiresIn:    tokenExpiresIn(faResp.TokenExpirationInstant),
		TokenType:    "Bearer",
	}, nil
}

// Login authenticates a user via email and password.
func (p *FusionAuthProvider) Login(ctx context.Context, req *LoginRequest) (*AuthResult, error) {
	body := map[string]interface{}{
		"loginId":       req.Email,
		"password":      req.Password,
		"applicationId": p.config.ApplicationID,
	}
	if req.MFACode != "" {
		body["twoFactorTrustId"] = req.MFACode
	}

	resp, err := p.doRequest(ctx, http.MethodPost, "/api/login", body)
	if err != nil {
		return nil, err
	}

	// 242 = MFA required
	if resp.StatusCode == 242 {
		var faResp faLoginResponse
		if err := decodeBody(resp, &faResp); err != nil {
			return nil, err
		}
		return &AuthResult{
			MFARequired: true,
			MFAToken:    faResp.TwoFactorID,
			TokenType:   "Bearer",
		}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fusionAuthError(resp)
	}

	var faResp faLoginResponse
	if err := decodeBody(resp, &faResp); err != nil {
		return nil, err
	}

	pub, err := mapFAUserToPublic(&faResp.User)
	if err != nil {
		return nil, err
	}
	return &AuthResult{
		User:         pub,
		AccessToken:  faResp.Token,
		RefreshToken: faResp.RefreshToken,
		ExpiresIn:    tokenExpiresIn(faResp.TokenExpirationInstant),
		TokenType:    "Bearer",
	}, nil
}

// Logout invalidates a refresh token by its ID (session ID maps to refresh token).
func (p *FusionAuthProvider) Logout(ctx context.Context, sessionID uuid.UUID) error {
	body := map[string]interface{}{
		"refreshToken": sessionID.String(),
		"global":       false,
	}

	resp, err := p.doRequest(ctx, http.MethodPost, "/api/logout", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fusionAuthError(resp)
	}

	return nil
}

// RefreshToken exchanges a refresh token for a new access token.
func (p *FusionAuthProvider) RefreshToken(ctx context.Context, refreshToken string) (*AuthResult, error) {
	values := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {p.config.ClientID},
	}
	if p.config.ClientSecret != "" {
		values.Set("client_secret", p.config.ClientSecret)
	}

	resp, err := p.doFormRequest(ctx, "/oauth2/token", values)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fusionAuthError(resp)
	}

	var tokenResp faTokenResponse
	if err := decodeBody(resp, &tokenResp); err != nil {
		return nil, err
	}

	return &AuthResult{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresIn:    tokenResp.ExpiresIn,
		TokenType:    tokenResp.TokenType,
	}, nil
}

// ChangePassword changes a user's password using the admin API.
func (p *FusionAuthProvider) ChangePassword(ctx context.Context, userID uuid.UUID, req *ChangePasswordRequest) error {
	// First, verify the current password by attempting a login
	loginBody := map[string]interface{}{
		"loginId":       userID.String(),
		"password":      req.CurrentPassword,
		"applicationId": p.config.ApplicationID,
	}

	loginResp, err := p.doRequest(ctx, http.MethodPost, "/api/login", loginBody)
	if err != nil {
		return err
	}
	defer loginResp.Body.Close()

	if loginResp.StatusCode != http.StatusOK {
		return fmt.Errorf("fusionauth: current password is incorrect")
	}

	// Use the admin API to change the password
	body := map[string]interface{}{
		"password": req.NewPassword,
	}

	resp, err := p.doRequest(ctx, http.MethodPut, "/api/user/change-password/"+userID.String(), body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fusionAuthError(resp)
	}

	return nil
}

// RequestPasswordReset sends a forgot-password email to the user.
func (p *FusionAuthProvider) RequestPasswordReset(ctx context.Context, email string) error {
	body := map[string]interface{}{
		"loginId":                 email,
		"sendForgotPasswordEmail": true,
	}
	if p.config.ApplicationID != "" {
		body["applicationId"] = p.config.ApplicationID
	}

	resp, err := p.doRequest(ctx, http.MethodPost, "/api/user/forgot-password", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fusionAuthError(resp)
	}

	return nil
}

// ConfirmPasswordReset resets a password using a change password token.
func (p *FusionAuthProvider) ConfirmPasswordReset(ctx context.Context, token, newPassword string) error {
	body := map[string]interface{}{
		"changePasswordId": token,
		"password":         newPassword,
	}

	resp, err := p.doRequest(ctx, http.MethodPost, "/api/user/change-password/"+token, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fusionAuthError(resp)
	}

	return nil
}

// EnableMFA enables TOTP-based two-factor authentication for a user.
func (p *FusionAuthProvider) EnableMFA(ctx context.Context, userID uuid.UUID) (*MFASetup, error) {
	body := map[string]interface{}{
		"method":       "authenticator",
		"code":         "",
		"secretBase32": "",
	}

	resp, err := p.doRequest(ctx, http.MethodPost, "/api/user/two-factor/"+userID.String(), body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fusionAuthError(resp)
	}

	var faResp faMFAEnableResponse
	if err := decodeBody(resp, &faResp); err != nil {
		return nil, err
	}

	secret := faResp.SecretBase32
	if secret == "" {
		secret = faResp.Secret
	}

	qrURL := fmt.Sprintf("otpauth://totp/Hearth:%s?secret=%s&issuer=Hearth", userID.String(), secret)

	return &MFASetup{
		Secret:    secret,
		QRCodeURL: qrURL,
	}, nil
}

// VerifyMFA validates a TOTP code for a user.
func (p *FusionAuthProvider) VerifyMFA(ctx context.Context, userID uuid.UUID, code string) error {
	body := map[string]interface{}{
		"code":   code,
		"method": "authenticator",
	}

	resp, err := p.doRequest(ctx, http.MethodPost, "/api/two-factor/login", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fusionAuthError(resp)
	}

	return nil
}

// DisableMFA disables two-factor authentication for a user.
func (p *FusionAuthProvider) DisableMFA(ctx context.Context, userID uuid.UUID) error {
	// Get user to find MFA method ID
	userResp, err := p.doRequest(ctx, http.MethodGet, "/api/user/"+userID.String(), nil)
	if err != nil {
		return err
	}

	if userResp.StatusCode != http.StatusOK {
		return fusionAuthError(userResp)
	}

	var faResp faUserResponse
	if err := decodeBody(userResp, &faResp); err != nil {
		return err
	}

	if len(faResp.User.TwoFactor.Methods) == 0 {
		return fmt.Errorf("fusionauth: MFA is not enabled for this user")
	}

	methodID := faResp.User.TwoFactor.Methods[0].ID

	resp, err := p.doRequest(ctx, http.MethodDelete, "/api/user/two-factor/"+userID.String()+"?methodId="+methodID, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fusionAuthError(resp)
	}

	return nil
}

// GetSessions returns all active refresh tokens (sessions) for a user.
func (p *FusionAuthProvider) GetSessions(ctx context.Context, userID uuid.UUID) ([]*models.Session, error) {
	resp, err := p.doRequest(ctx, http.MethodGet, "/api/jwt/refresh?userId="+userID.String(), nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fusionAuthError(resp)
	}

	var faResp faRefreshTokensResponse
	if err := decodeBody(resp, &faResp); err != nil {
		return nil, err
	}

	sessions := make([]*models.Session, 0, len(faResp.RefreshTokens))
	for _, rt := range faResp.RefreshTokens {
		sid, err := uuid.Parse(rt.ID)
		if err != nil {
			// Skip sessions with invalid UUIDs rather than failing the entire operation
			continue
		}
		created := time.UnixMilli(rt.InsertInstant)
		lastUsed := time.UnixMilli(rt.MetaData.LastAccessInstant)
		ua := rt.MetaData.UserAgent
		os := rt.MetaData.OperatingSystemName
		deviceName := rt.MetaData.Device.Name

		sess := &models.Session{
			ID:        sid,
			UserID:    userID,
			CreatedAt: created,
			LastUsed:  &lastUsed,
			UserAgent: &ua,
		}
		if os != "" {
			sess.OS = &os
		}
		if deviceName != "" {
			sess.DeviceName = &deviceName
		}
		switch rt.MetaData.Device.Type {
		case "MOBILE":
			sess.DeviceType = models.DeviceTypeMobile
		case "TABLET":
			sess.DeviceType = models.DeviceTypeTablet
		case "DESKTOP":
			sess.DeviceType = models.DeviceTypeDesktop
		default:
			sess.DeviceType = models.DeviceTypeUnknown
		}

		sessions = append(sessions, sess)
	}

	return sessions, nil
}

// RevokeSession revokes a specific refresh token by its token value.
func (p *FusionAuthProvider) RevokeSession(ctx context.Context, sessionID uuid.UUID) error {
	resp, err := p.doRequest(ctx, http.MethodDelete, "/api/jwt/refresh/"+sessionID.String(), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fusionAuthError(resp)
	}

	return nil
}

// RevokeAllSessions revokes all refresh tokens for a user.
func (p *FusionAuthProvider) RevokeAllSessions(ctx context.Context, userID uuid.UUID) error {
	resp, err := p.doRequest(ctx, http.MethodDelete, "/api/jwt/refresh?userId="+userID.String(), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fusionAuthError(resp)
	}

	return nil
}

// GetUser retrieves a user by ID from FusionAuth.
func (p *FusionAuthProvider) GetUser(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	resp, err := p.doRequest(ctx, http.MethodGet, "/api/user/"+userID.String(), nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, fmt.Errorf("fusionauth: user not found")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fusionAuthError(resp)
	}

	var faResp faUserResponse
	if err := decodeBody(resp, &faResp); err != nil {
		return nil, err
	}

	return mapFAUserToModel(&faResp.User)
}

// UpdateUser updates a user's profile in FusionAuth.
func (p *FusionAuthProvider) UpdateUser(ctx context.Context, userID uuid.UUID, req *models.UpdateUserRequest) (*models.User, error) {
	userData := map[string]interface{}{}
	dataFields := map[string]interface{}{}

	if req.Username != nil {
		userData["username"] = *req.Username
	}
	if req.AvatarURL != nil {
		userData["imageUrl"] = *req.AvatarURL
	}
	if req.DisplayName != nil {
		dataFields["displayName"] = *req.DisplayName
	}
	if req.Bio != nil {
		dataFields["bio"] = *req.Bio
	}
	if req.Pronouns != nil {
		dataFields["pronouns"] = *req.Pronouns
	}

	if len(dataFields) > 0 {
		userData["data"] = dataFields
	}

	body := map[string]interface{}{
		"user": userData,
	}

	resp, err := p.doRequest(ctx, http.MethodPut, "/api/user/"+userID.String(), body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fusionAuthError(resp)
	}

	var faResp faUserResponse
	if err := decodeBody(resp, &faResp); err != nil {
		return nil, err
	}

	return mapFAUserToModel(&faResp.User)
}

// DeleteUser deactivates and deletes a user from FusionAuth.
func (p *FusionAuthProvider) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	resp, err := p.doRequest(ctx, http.MethodDelete, "/api/user/"+userID.String()+"?hardDelete=true", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fusionAuthError(resp)
	}

	return nil
}

// GetAuthorizationURL returns the FusionAuth OAuth2 authorization URL.
func (p *FusionAuthProvider) GetAuthorizationURL(ctx context.Context, state string) (string, error) {
	scopes := "openid offline_access"
	if len(p.config.Scopes) > 0 {
		scopes = strings.Join(p.config.Scopes, " ")
	}

	params := url.Values{
		"client_id":     {p.config.ClientID},
		"response_type": {"code"},
		"redirect_uri":  {p.config.RedirectURI},
		"state":         {state},
		"scope":         {scopes},
	}
	if p.config.TenantID != "" {
		params.Set("tenantId", p.config.TenantID)
	}

	return p.config.Host + "/oauth2/authorize?" + params.Encode(), nil
}

// HandleCallback exchanges an authorization code for tokens.
func (p *FusionAuthProvider) HandleCallback(ctx context.Context, code, state string) (*AuthResult, error) {
	values := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"client_id":    {p.config.ClientID},
		"redirect_uri": {p.config.RedirectURI},
	}
	if p.config.ClientSecret != "" {
		values.Set("client_secret", p.config.ClientSecret)
	}

	resp, err := p.doFormRequest(ctx, "/oauth2/token", values)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fusionAuthError(resp)
	}

	var tokenResp faTokenResponse
	if err := decodeBody(resp, &tokenResp); err != nil {
		return nil, err
	}

	return &AuthResult{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresIn:    tokenResp.ExpiresIn,
		TokenType:    tokenResp.TokenType,
	}, nil
}
