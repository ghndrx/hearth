package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultNativeConfig(t *testing.T) {
	config := DefaultNativeConfig()

	assert.NotNil(t, config)
	assert.Equal(t, 8, config.PasswordMinLength)
	assert.False(t, config.PasswordRequireUppercase)
	assert.False(t, config.PasswordRequireNumber)
	assert.False(t, config.PasswordRequireSpecial)
	assert.Equal(t, 5, config.MaxLoginAttempts)
	assert.Equal(t, 15*time.Minute, config.LockoutDuration)
	assert.False(t, config.RequireEmailVerification)
	assert.Equal(t, "Hearth", config.MFAIssuer)
}

func TestDefaultJWTConfig(t *testing.T) {
	config := DefaultJWTConfig()

	assert.Equal(t, 15*time.Minute, config.AccessTokenTTL)
	assert.Equal(t, 7*24*time.Hour, config.RefreshTokenTTL)
	assert.Equal(t, "hearth", config.Issuer)
	assert.Contains(t, config.Audience, "hearth-api")
	assert.Len(t, config.Audience, 1)
}

func TestProviderTypes(t *testing.T) {
	// Test provider type constants
	assert.Equal(t, ProviderType("native"), ProviderNative)
	assert.Equal(t, ProviderType("fusionauth"), ProviderFusionAuth)
	assert.Equal(t, ProviderType("oidc"), ProviderOIDC)
}

func TestNativeConfig_Defaults(t *testing.T) {
	native := DefaultNativeConfig()

	// Verify sensible security defaults
	assert.GreaterOrEqual(t, native.PasswordMinLength, 8)
	assert.Greater(t, native.MaxLoginAttempts, 0)
	assert.Greater(t, native.LockoutDuration, time.Duration(0))
}

func TestJWTConfig_Defaults(t *testing.T) {
	jwt := DefaultJWTConfig()

	// Access token should be shorter than refresh token
	assert.Less(t, jwt.AccessTokenTTL, jwt.RefreshTokenTTL)

	// Refresh token should be reasonable (not longer than 30 days)
	assert.LessOrEqual(t, jwt.RefreshTokenTTL, 30*24*time.Hour)

	// Access token should be short (not longer than 1 hour)
	assert.LessOrEqual(t, jwt.AccessTokenTTL, 1*time.Hour)
}

func TestProviderConfig_Structure(t *testing.T) {
	config := &ProviderConfig{
		Type:    ProviderNative,
		Enabled: true,
		Native:  DefaultNativeConfig(),
	}

	assert.Equal(t, ProviderNative, config.Type)
	assert.True(t, config.Enabled)
	assert.NotNil(t, config.Native)
	assert.Nil(t, config.FusionAuth)
	assert.Nil(t, config.OIDC)
}

func TestProviderConfig_FusionAuth(t *testing.T) {
	config := &ProviderConfig{
		Type:    ProviderFusionAuth,
		Enabled: true,
		FusionAuth: &FusionAuthConfig{
			Host:   "https://auth.example.com",
			APIKey: "test-api-key",
		},
	}

	assert.Equal(t, ProviderFusionAuth, config.Type)
	assert.NotNil(t, config.FusionAuth)
	assert.Equal(t, "https://auth.example.com", config.FusionAuth.Host)
}

func TestProviderConfig_OIDC(t *testing.T) {
	config := &ProviderConfig{
		Type:    ProviderOIDC,
		Enabled: true,
		OIDC: &OIDCConfig{
			Issuer:                "https://accounts.google.com",
			ClientID:              "test-client-id",
			ClientSecret:          "test-secret",
			AuthorizationEndpoint: "https://accounts.google.com/o/oauth2/auth",
			TokenEndpoint:         "https://oauth2.googleapis.com/token",
			Scopes:                []string{"openid", "email", "profile"},
		},
	}

	assert.Equal(t, ProviderOIDC, config.Type)
	assert.NotNil(t, config.OIDC)
	assert.Equal(t, "https://accounts.google.com", config.OIDC.Issuer)
	assert.Contains(t, config.OIDC.Scopes, "openid")
}
