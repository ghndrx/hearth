package services

import (
	"context"
	"errors"

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

// AuthService defines the business logic for authentication.
type AuthService interface {
	Register(ctx context.Context, email, username, password string) (*models.User, *AuthTokens, error)
	Login(ctx context.Context, email, password string) (*models.User, *AuthTokens, error)
	RefreshTokens(ctx context.Context, refreshToken string) (*AuthTokens, error)
	ValidateToken(ctx context.Context, token string) (uuid.UUID, error)
}

// authRepository defines the storage interface required by the auth service.
type authRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByEmail(ctx context.Context, email string) (*models.User, error)
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
