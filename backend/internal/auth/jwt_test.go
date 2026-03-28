package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewJWTService(t *testing.T) {
	service := NewJWTService("test-secret", 15*time.Minute, 7*24*time.Hour)

	assert.NotNil(t, service)
	assert.Equal(t, []byte("test-secret"), service.secretKey)
	assert.Equal(t, 15*time.Minute, service.accessExpiry)
	assert.Equal(t, 7*24*time.Hour, service.refreshExpiry)
	assert.Equal(t, "hearth", service.issuer)
}

func TestGenerateAccessToken(t *testing.T) {
	service := NewJWTService("test-secret-key-for-testing", 15*time.Minute, 7*24*time.Hour)
	userID := uuid.New()
	username := "testuser"

	token, err := service.GenerateAccessToken(userID, username)

	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Verify token structure (header.payload.signature)
	parts := strings.Split(token, ".")
	assert.Len(t, parts, 3)

	// Validate the token
	claims, err := service.ValidateAccessToken(token)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, username, claims.Username)
	assert.Equal(t, "access", claims.Type)
	assert.Equal(t, "hearth", claims.Issuer)
}

func TestGenerateRefreshToken(t *testing.T) {
	service := NewJWTService("test-secret-key-for-testing", 15*time.Minute, 7*24*time.Hour)
	userID := uuid.New()

	token, err := service.GenerateRefreshToken(userID)

	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Validate the token
	claims, err := service.ValidateRefreshToken(token)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, "refresh", claims.Type)
	assert.Empty(t, claims.Username) // Refresh tokens don't include username
}

func TestGenerateTokenPair(t *testing.T) {
	service := NewJWTService("test-secret-key-for-testing", 15*time.Minute, 7*24*time.Hour)
	userID := uuid.New()
	username := "testuser"

	accessToken, refreshToken, err := service.GenerateTokenPair(userID, username)

	require.NoError(t, err)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, refreshToken)
	assert.NotEqual(t, accessToken, refreshToken)

	// Validate both tokens
	accessClaims, err := service.ValidateAccessToken(accessToken)
	require.NoError(t, err)
	assert.Equal(t, userID, accessClaims.UserID)
	assert.Equal(t, "access", accessClaims.Type)

	refreshClaims, err := service.ValidateRefreshToken(refreshToken)
	require.NoError(t, err)
	assert.Equal(t, userID, refreshClaims.UserID)
	assert.Equal(t, "refresh", refreshClaims.Type)
}

func TestValidateToken_InvalidFormat(t *testing.T) {
	service := NewJWTService("test-secret", 15*time.Minute, 7*24*time.Hour)

	testCases := []struct {
		name  string
		token string
	}{
		{"empty token", ""},
		{"garbage", "not-a-valid-token"},
		{"missing parts", "header.payload"},
		{"random base64", "aGVsbG8.d29ybGQ.Zm9v"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			claims, err := service.ValidateToken(tc.token)
			assert.Error(t, err)
			assert.Nil(t, claims)
			assert.Equal(t, ErrInvalidToken, err)
		})
	}
}

func TestValidateToken_WrongSigningMethod(t *testing.T) {
	service := NewJWTService("test-secret", 15*time.Minute, 7*24*time.Hour)
	userID := uuid.New()

	// Create token with different signing method (none)
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "hearth",
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
		},
		UserID: userID,
		Type:   "access",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	tokenString, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)

	// Should reject none signature
	result, err := service.ValidateToken(tokenString)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestValidateToken_WrongSecret(t *testing.T) {
	service1 := NewJWTService("secret-1", 15*time.Minute, 7*24*time.Hour)
	service2 := NewJWTService("secret-2", 15*time.Minute, 7*24*time.Hour)
	userID := uuid.New()

	token, err := service1.GenerateAccessToken(userID, "testuser")
	require.NoError(t, err)

	// Validate with different secret
	claims, err := service2.ValidateToken(token)
	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.Equal(t, ErrInvalidToken, err)
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	// Create service with very short expiry
	service := NewJWTService("test-secret", 1*time.Millisecond, 7*24*time.Hour)
	userID := uuid.New()

	token, err := service.GenerateAccessToken(userID, "testuser")
	require.NoError(t, err)

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	claims, err := service.ValidateToken(token)
	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.Equal(t, ErrExpiredToken, err)
}

func TestValidateAccessToken_WithRefreshToken(t *testing.T) {
	service := NewJWTService("test-secret", 15*time.Minute, 7*24*time.Hour)
	userID := uuid.New()

	// Generate refresh token
	refreshToken, err := service.GenerateRefreshToken(userID)
	require.NoError(t, err)

	// Try to validate as access token (should fail)
	claims, err := service.ValidateAccessToken(refreshToken)
	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.Equal(t, ErrInvalidToken, err)
}

func TestValidateRefreshToken_WithAccessToken(t *testing.T) {
	service := NewJWTService("test-secret", 15*time.Minute, 7*24*time.Hour)
	userID := uuid.New()

	// Generate access token
	accessToken, err := service.GenerateAccessToken(userID, "testuser")
	require.NoError(t, err)

	// Try to validate as refresh token (should fail)
	claims, err := service.ValidateRefreshToken(accessToken)
	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.Equal(t, ErrInvalidToken, err)
}

func TestGetExpirySeconds(t *testing.T) {
	testCases := []struct {
		name     string
		expiry   time.Duration
		expected int
	}{
		{"15 minutes", 15 * time.Minute, 900},
		{"1 hour", 1 * time.Hour, 3600},
		{"24 hours", 24 * time.Hour, 86400},
		{"30 seconds", 30 * time.Second, 30},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := NewJWTService("secret", tc.expiry, 7*24*time.Hour)
			assert.Equal(t, tc.expected, service.GetExpirySeconds())
		})
	}
}

func TestClaims_TokenExpiry(t *testing.T) {
	accessExpiry := 15 * time.Minute
	refreshExpiry := 7 * 24 * time.Hour
	service := NewJWTService("test-secret", accessExpiry, refreshExpiry)
	userID := uuid.New()

	// Check access token expiry
	accessToken, err := service.GenerateAccessToken(userID, "testuser")
	require.NoError(t, err)

	accessClaims, err := service.ValidateAccessToken(accessToken)
	require.NoError(t, err)

	// Expiry should be roughly accessExpiry from now
	expectedExpiry := time.Now().Add(accessExpiry)
	actualExpiry := accessClaims.ExpiresAt.Time
	assert.WithinDuration(t, expectedExpiry, actualExpiry, 2*time.Second)

	// Check refresh token expiry
	refreshToken, err := service.GenerateRefreshToken(userID)
	require.NoError(t, err)

	refreshClaims, err := service.ValidateRefreshToken(refreshToken)
	require.NoError(t, err)

	expectedRefreshExpiry := time.Now().Add(refreshExpiry)
	actualRefreshExpiry := refreshClaims.ExpiresAt.Time
	assert.WithinDuration(t, expectedRefreshExpiry, actualRefreshExpiry, 2*time.Second)
}

func TestClaims_UniqueJTI(t *testing.T) {
	service := NewJWTService("test-secret", 15*time.Minute, 7*24*time.Hour)
	userID := uuid.New()

	// Generate multiple tokens
	token1, _ := service.GenerateAccessToken(userID, "testuser")
	token2, _ := service.GenerateAccessToken(userID, "testuser")
	token3, _ := service.GenerateAccessToken(userID, "testuser")

	claims1, _ := service.ValidateAccessToken(token1)
	claims2, _ := service.ValidateAccessToken(token2)
	claims3, _ := service.ValidateAccessToken(token3)

	// Each token should have a unique JTI
	assert.NotEqual(t, claims1.ID, claims2.ID)
	assert.NotEqual(t, claims2.ID, claims3.ID)
	assert.NotEqual(t, claims1.ID, claims3.ID)
}

func TestClaims_SubjectMatchesUserID(t *testing.T) {
	service := NewJWTService("test-secret", 15*time.Minute, 7*24*time.Hour)
	userID := uuid.New()

	token, err := service.GenerateAccessToken(userID, "testuser")
	require.NoError(t, err)

	claims, err := service.ValidateAccessToken(token)
	require.NoError(t, err)

	// Subject should match UserID string
	assert.Equal(t, userID.String(), claims.Subject)
	assert.Equal(t, userID, claims.UserID)
}
