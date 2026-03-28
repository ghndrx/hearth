package services

import (
	"context"
	"errors"
	"fmt"

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

	// Generate new token pair
	accessToken, newRefreshToken, err := s.jwtService.GenerateTokenPair(claims.UserID, claims.Username)
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
	accessToken, refreshToken, err := s.jwtService.GenerateTokenPair(user.ID, user.Username)
	if err != nil {
		return nil, err
	}

	return &AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    s.jwtService.GetExpirySeconds(),
	}, nil
}
