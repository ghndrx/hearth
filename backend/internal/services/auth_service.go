package services

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"hearth/internal/models"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserExists        = errors.New("user already exists")
)

// AuthService defines the business logic for authentication.
type AuthService interface {
	Register(ctx context.Context, email, username, password string) (*models.User, error)
	Login(ctx context.Context, email, password string) (string, *models.User, error) // Returns session token and user
}

// authRepository defines the storage interface required by the auth service.
type authRepository interface {
	CreateUser(ctx context.Context, user *models.User) error
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	// Ideally, we would have a CreateSession method here, but for simplicity
	// we will return a generated token in the service layer for this implementation.
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
	_, err := s.repo.GetUserByEmail(ctx, email)
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
		ID:       uuid.New(),
		Email:    email,
		Username: username,
		Password: string(hashedPassword),
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// Login handles user login and credentials verification.
func (s *authService) Login(ctx context.Context, email, password string) (string, *models.User, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return "", nil, ErrInvalidCredentials
		}
		return "", nil, err
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", nil, ErrInvalidCredentials
	}

	// Generate a simple session token (in a real app, use JWT or refresh tokens)
	sessionToken := uuid.New().String()

	return sessionToken, user, nil
}