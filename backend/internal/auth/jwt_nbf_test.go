package auth

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestImmediateTokenValidation tests that tokens can be validated immediately
// after generation, simulating the registration flow where a token is created
// and immediately used for an authenticated request.
//
// This test specifically targets PERF-000: Token Validation Failure After Registration
func TestImmediateTokenValidation(t *testing.T) {
	service := NewJWTService("test-secret", 15*time.Minute, 7*24*time.Hour)

	// Run this test multiple times to catch timing-sensitive issues
	for i := 0; i < 100; i++ {
		userID := uuid.New()
		username := "testuser"

		// Generate token
		accessToken, err := service.GenerateAccessToken(userID, username)
		require.NoError(t, err, "Failed to generate token on iteration %d", i)

		// IMMEDIATELY validate - no sleep, no delay
		claims, err := service.ValidateAccessToken(accessToken)
		require.NoError(t, err, "Token validation failed on iteration %d: %v", i, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, username, claims.Username)
		assert.Equal(t, "access", claims.Type)
	}
}

// TestImmediateTokenValidationConcurrent tests concurrent token generation
// and immediate validation under load, which is when the PERF-000 bug manifests.
func TestImmediateTokenValidationConcurrent(t *testing.T) {
	service := NewJWTService("test-secret", 15*time.Minute, 7*24*time.Hour)

	const numGoroutines = 50
	const iterationsPerGoroutine = 20

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*iterationsPerGoroutine)

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for i := 0; i < iterationsPerGoroutine; i++ {
				userID := uuid.New()
				username := "testuser"

				// Generate and immediately validate
				accessToken, err := service.GenerateAccessToken(userID, username)
				if err != nil {
					errors <- err
					continue
				}

				claims, err := service.ValidateAccessToken(accessToken)
				if err != nil {
					errors <- err
					continue
				}

				if claims.UserID != userID {
					t.Errorf("UserID mismatch in goroutine %d iteration %d", goroutineID, i)
				}
			}
		}(g)
	}

	wg.Wait()
	close(errors)

	// Collect all errors
	var allErrors []error
	for err := range errors {
		allErrors = append(allErrors, err)
	}

	if len(allErrors) > 0 {
		t.Errorf("Got %d validation errors under concurrent load. First error: %v", len(allErrors), allErrors[0])
	}
}

// TestTokenValidationWithDifferentSecrets ensures that tokens generated with
// one secret fail validation with a different secret. This verifies the
// SECRET_KEY consistency requirement from PERF-000.
func TestTokenValidationWithDifferentSecrets(t *testing.T) {
	// Simulate misconfiguration: auth service uses one secret, middleware uses another
	authService := NewJWTService("auth-service-secret", 15*time.Minute, 7*24*time.Hour)
	middlewareService := NewJWTService("middleware-secret", 15*time.Minute, 7*24*time.Hour)

	userID := uuid.New()

	// Auth service generates token during registration
	accessToken, err := authService.GenerateAccessToken(userID, "testuser")
	require.NoError(t, err)

	// Middleware tries to validate with different secret
	_, err = middlewareService.ValidateAccessToken(accessToken)
	assert.Error(t, err, "Token should fail validation with different secret")
	assert.Equal(t, ErrInvalidToken, err)
}

// TestRegistrationFlowSimulation simulates the exact registration flow:
// 1. Generate tokens (like auth service does during registration)
// 2. Immediately validate the access token (like middleware does on next request)
func TestRegistrationFlowSimulation(t *testing.T) {
	// Use the same secret for both (as should happen in production)
	const sharedSecret = "shared-jwt-secret-key"

	// This simulates what happens in auth service during registration
	authJWT := NewJWTService(sharedSecret, 15*time.Minute, 7*24*time.Hour)

	// This simulates what happens in middleware during token validation
	middlewareJWT := NewJWTService(sharedSecret, 0, 0) // Middleware doesn't care about expiry durations

	userID := uuid.New()
	username := "newuser"

	// Step 1: Registration - auth service generates tokens
	accessToken, refreshToken, err := authJWT.GenerateTokenPair(userID, username)
	require.NoError(t, err, "Token generation should succeed")
	require.NotEmpty(t, accessToken)
	require.NotEmpty(t, refreshToken)

	// Step 2: Next request - middleware validates access token
	// This is where PERF-000 bug manifests (401 "invalid token")
	claims, err := middlewareJWT.ValidateAccessToken(accessToken)
	require.NoError(t, err, "Token validation should succeed immediately after generation")

	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, username, claims.Username)
	assert.Equal(t, "access", claims.Type)
}
