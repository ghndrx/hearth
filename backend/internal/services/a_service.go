package services

import (
	"database/sql"
	"errors"
	"fmt"
)

// ErrUserNotFound represents an error case when a user is not found.
var ErrUserNotFound = errors.New("user not found")

// UserService defines the contract for any user-related operations.
// This allows for mocking in tests or swapping implementations.
type UserService interface {
	GetUserByID(id int) (*User, error)
}

// User represents the data structure for a User entity.
type User struct {
	ID       int
	Username string
	Email    string
}

// UserServiceImpl is a concrete implementation of the UserService.
type UserServiceImpl struct {
	db *sql.DB
}

// NewUserService creates a new instance of UserService.
func NewUserService(db *sql.DB) UserService {
	return &UserServiceImpl{db: db}
}

// GetUserByID retrieves a user from the database by their ID.
func (s *UserServiceImpl) GetUserByID(id int) (*User, error) {
	// In a real application, sanitize the ID to prevent SQL injection.
	// Here we perform a basic raw query for demonstration.
	query := "SELECT id, username, email FROM users WHERE id = $1"

	row := s.db.QueryRow(query, id)

	var user User
	err := row.Scan(&user.ID, &user.Username, &user.Email)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	return &user, nil
}