package services

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"hearth/internal/models"
)

var (
	ErrUserExists = errors.New("user already exists")
)

// AuthService defines the business logic for authentication.
type AuthService interface {
	Register(ctx context.Context, email, username, password string) (*models.User, error)
	Login(ctx context.Context, email, password string) (string, *models.User, error) // Returns session token and user
}

// authRepository defines the storage interface required by the auth service.
type authRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByEmail(ctx context.Context, email string) (*models.User, error)
}

type authService struct {
	repo authRepository
}

// NewAuthService creates a new auth service instance.
func NewAuthService(repo authRepository) AuthService {
	return &authService{
		repo: repo,
	}
}

// Register handles new user registration.
func (s *authService) Register(ctx context.Context, email, username, password string) (*models.User, error) {
	// Check if user already exists
	_, err := s.repo.GetByEmail(ctx, email)
	if err == nil {
		return nil, ErrUserExists
	}
	if !errors.Is(err, ErrUserNotFound) {
		return nil, err // Return unexpected database errors
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		ID:           uuid.New(),
		Email:        email,
		Username:     username,
		PasswordHash: string(hashedPassword),
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// Login handles user login and credentials verification.
func (s *authService) Login(ctx context.Context, email, password string) (string, *models.User, error) {
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return "", nil, ErrInvalidCredentials
		}
		return "", nil, err
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", nil, ErrInvalidCredentials
	}

	// Generate a simple session token (in a real app, use JWT or refresh tokens)
	sessionToken := uuid.New().String()

	return sessionToken, user, nil
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
	if s.cache == nil {
		return nil
	}
	
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return nil // Already expired
	}
	
	key := "revoked:" + tokenID
	return s.cache.Set(ctx, key, []byte("1"), ttl)
}

func (s *AuthService) isTokenRevoked(ctx context.Context, tokenID string) (bool, error) {
	if s.cache == nil {
		return false, nil
	}
	
	key := "revoked:" + tokenID
	_, err := s.cache.Get(ctx, key)
	if err != nil {
		return false, nil // Not found = not revoked
	}
	return true, nil
}
