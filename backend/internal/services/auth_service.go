package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"hearth/internal/auth"
	"hearth/internal/models"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailTaken         = errors.New("email already registered")
	ErrUsernameTaken      = errors.New("username already taken")
	ErrRegistrationClosed = errors.New("registration is currently closed")
	ErrInviteRequired     = errors.New("an invite is required to register")
)

// AuthService handles authentication
type AuthService struct {
	userRepo     UserRepository
	jwtService   *auth.JWTService
	cache        CacheService
	
	registrationEnabled bool
	inviteOnly          bool
}

// NewAuthService creates a new auth service
func NewAuthService(
	userRepo UserRepository,
	jwtService *auth.JWTService,
	cache CacheService,
	registrationEnabled, inviteOnly bool,
) *AuthService {
	return &AuthService{
		userRepo:            userRepo,
		jwtService:          jwtService,
		cache:               cache,
		registrationEnabled: registrationEnabled,
		inviteOnly:          inviteOnly,
	}
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Email       string
	Username    string
	DisplayName string
	Password    string
	InviteCode  string
}

// TokenResponse represents authentication tokens
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// Register creates a new user account
func (s *AuthService) Register(ctx context.Context, req *RegisterRequest) (*models.User, *TokenResponse, error) {
	// Check if registration is enabled
	if !s.registrationEnabled {
		return nil, nil, ErrRegistrationClosed
	}
	
	// Check if invite is required
	if s.inviteOnly && req.InviteCode == "" {
		return nil, nil, ErrInviteRequired
	}
	
	// TODO: Validate invite code if provided
	
	// Check if email is taken
	existing, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, nil, err
	}
	if existing != nil {
		return nil, nil, ErrEmailTaken
	}
	
	// Check if username is taken
	existing, err = s.userRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		return nil, nil, err
	}
	if existing != nil {
		return nil, nil, ErrUsernameTaken
	}
	
	// Hash password
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, nil, err
	}
	
	// Create user
	user := &models.User{
		ID:           uuid.New(),
		Username:     req.Username,
		DisplayName:  req.DisplayName,
		Email:        req.Email,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	
	if user.DisplayName == "" {
		user.DisplayName = user.Username
	}
	
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, nil, err
	}
	
	// Generate tokens
	accessToken, refreshToken, err := s.jwtService.GenerateTokenPair(user.ID, user.Username)
	if err != nil {
		return nil, nil, err
	}
	
	tokens := &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    s.jwtService.GetExpirySeconds(),
		TokenType:    "Bearer",
	}
	
	return user, tokens, nil
}

// Login authenticates a user
func (s *AuthService) Login(ctx context.Context, email, password string) (*models.User, *TokenResponse, error) {
	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, nil, err
	}
	if user == nil {
		return nil, nil, ErrInvalidCredentials
	}
	
	// Check password
	if err := auth.CheckPassword(password, user.PasswordHash); err != nil {
		return nil, nil, ErrInvalidCredentials
	}
	
	// Check if password needs rehash
	if auth.NeedsRehash(user.PasswordHash) {
		if newHash, err := auth.HashPassword(password); err == nil {
			user.PasswordHash = newHash
			_ = s.userRepo.Update(ctx, user)
		}
	}
	
	// Generate tokens
	accessToken, refreshToken, err := s.jwtService.GenerateTokenPair(user.ID, user.Username)
	if err != nil {
		return nil, nil, err
	}
	
	tokens := &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    s.jwtService.GetExpirySeconds(),
		TokenType:    "Bearer",
	}
	
	return user, tokens, nil
}

// RefreshToken generates new tokens from a refresh token
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	// Validate refresh token
	claims, err := s.jwtService.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}
	
	// Check if token is revoked
	revoked, _ := s.isTokenRevoked(ctx, claims.ID)
	if revoked {
		return nil, auth.ErrInvalidToken
	}
	
	// Get user
	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil || user == nil {
		return nil, auth.ErrInvalidToken
	}
	
	// Revoke old refresh token
	_ = s.revokeToken(ctx, claims.ID, claims.ExpiresAt.Time)
	
	// Generate new tokens
	accessToken, newRefreshToken, err := s.jwtService.GenerateTokenPair(user.ID, user.Username)
	if err != nil {
		return nil, err
	}
	
	return &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    s.jwtService.GetExpirySeconds(),
		TokenType:    "Bearer",
	}, nil
}

// Logout invalidates tokens
func (s *AuthService) Logout(ctx context.Context, accessToken, refreshToken string) error {
	// Revoke access token
	if claims, err := s.jwtService.ValidateAccessToken(accessToken); err == nil {
		_ = s.revokeToken(ctx, claims.ID, claims.ExpiresAt.Time)
	}
	
	// Revoke refresh token
	if claims, err := s.jwtService.ValidateRefreshToken(refreshToken); err == nil {
		_ = s.revokeToken(ctx, claims.ID, claims.ExpiresAt.Time)
	}
	
	return nil
}

// ValidateToken validates an access token and returns the user ID
func (s *AuthService) ValidateToken(ctx context.Context, token string) (uuid.UUID, error) {
	claims, err := s.jwtService.ValidateAccessToken(token)
	if err != nil {
		return uuid.Nil, err
	}
	
	// Check if token is revoked
	revoked, _ := s.isTokenRevoked(ctx, claims.ID)
	if revoked {
		return uuid.Nil, auth.ErrInvalidToken
	}
	
	return claims.UserID, nil
}

// Token revocation helpers

func (s *AuthService) revokeToken(ctx context.Context, tokenID string, expiresAt time.Time) error {
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return nil // Already expired
	}
	
	key := "revoked:" + tokenID
	return s.cache.Set(ctx, key, []byte("1"), ttl)
}

func (s *AuthService) isTokenRevoked(ctx context.Context, tokenID string) (bool, error) {
	key := "revoked:" + tokenID
	_, err := s.cache.Get(ctx, key)
	if err != nil {
		return false, nil // Not found = not revoked
	}
	return true, nil
}
