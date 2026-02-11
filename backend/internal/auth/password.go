package auth

import (
	"errors"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

const (
	// Cost factor for bcrypt (12 is recommended for production)
	bcryptCost = 12
	
	// Minimum password length
	minPasswordLength = 8
	
	// Maximum password length (bcrypt has a 72 byte limit)
	maxPasswordLength = 72
)

var (
	ErrPasswordTooShort   = errors.New("password must be at least 8 characters")
	ErrPasswordTooLong    = errors.New("password must be at most 72 characters")
	ErrPasswordWeak       = errors.New("password must contain at least one uppercase, lowercase, and number")
	ErrPasswordMismatch   = errors.New("invalid password")
)

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	if err := ValidatePasswordStrength(password); err != nil {
		return "", err
	}
	
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	
	return string(hash), nil
}

// CheckPassword compares a password with its hash
func CheckPassword(password, hash string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		return ErrPasswordMismatch
	}
	return nil
}

// ValidatePasswordStrength checks if a password meets requirements
func ValidatePasswordStrength(password string) error {
	if len(password) < minPasswordLength {
		return ErrPasswordTooShort
	}
	
	if len(password) > maxPasswordLength {
		return ErrPasswordTooLong
	}
	
	var hasUpper, hasLower, hasNumber bool
	
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		}
	}
	
	if !hasUpper || !hasLower || !hasNumber {
		return ErrPasswordWeak
	}
	
	return nil
}

// NeedsRehash checks if a password hash needs to be upgraded
func NeedsRehash(hash string) bool {
	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		return true
	}
	return cost < bcryptCost
}
