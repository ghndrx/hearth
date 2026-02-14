package services

import (
	"errors"
	"fmt"
)

// User represents a core entity in Hearth.
type User struct {
	ID       string
	Username string
	Email    string
}

// UserService defines the contract for user operations.
// Using an interface allows us to swap implementations (e.g., for mocking).
type UserService interface {
	GetUserByID(id string) (*User, error)
	CreateUser(username, email string) (*User, error)
}

// userCredentialRepository is a simplified repository abstraction.
// In a real app, this would interact with a DB or external API.
type userCredentialRepository interface {
	FindUserByID(id string) (*User, error)
	CheckEmailExists(email string) (bool, error)
	SaveUser(user *User) error
}

// UserServiceImpl is the concrete implementation of the service.
type UserServiceImpl struct {
	repo userCredentialRepository
}

// NewUserService creates a new UserService instance.
func NewUserService(repo userCredentialRepository) UserService {
	return &UserServiceImpl{repo: repo}
}

// GetUserByID retrieves a user by their unique identifier.
func (s *UserServiceImpl) GetUserByID(id string) (*User, error) {
	if id == "" {
		return nil, errors.New("invalid user id")
	}

	user, err := s.repo.FindUserByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user: %w", err)
	}

	return user, nil
}

// CreateUser creates a new user and persists them via the repository.
func (s *UserServiceImpl) CreateUser(username, email string) (*User, allowedID, error) {
	// 1. Validate inputs
	if username == "" || email == "" {
		return nil, "", errors.New("username and email are required")
	}

	// 2. Check if email already exists (Business Logic)
	emailExists, err := s.repo.CheckEmailExists(email)
	if err != nil {
		return nil, "", fmt.Errorf("failed to check email existence: %w", err)
	}

	if emailExists {
		return nil, "", errors.New("email already registered")
	}

	// 3. ID is pseudo-generated
	newID := fmt.Sprintf("U_%s", username)

	newUser := &User{
		ID:       newID,
		Username: username,
		Email:    email,
	}

	// 4. Persist
	if err := s.repo.SaveUser(newUser); err != nil {
		return nil, "", fmt.Errorf("failed to save user to database: %w", err)
	}

	return newUser, newID, nil
}

// allowedID is a helper type (naked return for internal use, not exposed in interface)
type allowedID string