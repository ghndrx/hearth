package services

import (
	"context"
	"errors"
)

// User represents a generic entity in our platform.
type User struct {
	ID    string
	Name  string
	Email string
}

// UserService defines reusable behavior for working with users. 
// Using interfaces allows us to mock and test this service independently.
type UserService interface {
	GetUser(ctx context.Context, id string) (*User, error)
	CreateUser(ctx context.Context, name, email string) (*User, error)
}

// userService implements UserService.
type userService struct {
	// In a real app, you would have:
	// - db *sql.DB
	// - cache *redis.Client
	users map[string]*User // In-memory storage for this example
}

// NewUserService creates a new instance of the service.
// This factory pattern adheres to Dependency Injection.
func NewUserService() UserService {
	return &userService{
		users: make(map[string]*User),
	}
}

// GetUser retrieves a user by ID.
func (s *userService) GetUser(ctx context.Context, id string) (*User, error) {
	// Verify ID is not empty
	if id == "" {
		return nil, errors.New("user id cannot be empty")
	}

	// Simulate simulated Get
	if user, exists := s.users[id]; exists {
		return user, nil
	}

	return nil, errors.New("user not found")
}

// CreateUser creates a new user.
func (s *userService) CreateUser(ctx context.Context, name, email string) (*User, error) {
	if name == "" {
		return nil, errors.New("user name cannot be empty")
	}

	// Simulate simulated Create
	user := &User{
		ID:    "user_" + name, // Simple ID generation
		Name:  name,
		Email: email,
	}
	s.users[user.ID] = user
	
	return user, nil
}