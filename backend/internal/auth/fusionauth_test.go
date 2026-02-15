package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hearth/internal/models"
)

func TestNewFusionAuthProvider(t *testing.T) {
	cfg := &FusionAuthConfig{
		Host:   "https://auth.example.com",
		APIKey: "test-api-key",
	}

	provider, err := NewFusionAuthProvider(cfg)

	require.NoError(t, err)
	assert.NotNil(t, provider)
}

func TestFusionAuthProvider_Name(t *testing.T) {
	provider, _ := NewFusionAuthProvider(&FusionAuthConfig{})

	assert.Equal(t, "FusionAuth", provider.Name())
}

func TestFusionAuthProvider_Type(t *testing.T) {
	provider, _ := NewFusionAuthProvider(&FusionAuthConfig{})

	assert.Equal(t, ProviderFusionAuth, provider.Type())
}

func TestFusionAuthProvider_Register(t *testing.T) {
	provider, _ := NewFusionAuthProvider(&FusionAuthConfig{})
	ctx := context.Background()

	req := &RegisterRequest{
		Email:    "test@example.com",
		Username: "testuser",
		Password: "password123",
	}

	result, err := provider.Register(ctx, req)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrFusionAuthNotImplemented)
}

func TestFusionAuthProvider_Login(t *testing.T) {
	provider, _ := NewFusionAuthProvider(&FusionAuthConfig{})
	ctx := context.Background()

	req := &LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	result, err := provider.Login(ctx, req)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrFusionAuthNotImplemented)
}

func TestFusionAuthProvider_Logout(t *testing.T) {
	provider, _ := NewFusionAuthProvider(&FusionAuthConfig{})
	ctx := context.Background()

	err := provider.Logout(ctx, uuid.New())

	assert.ErrorIs(t, err, ErrFusionAuthNotImplemented)
}

func TestFusionAuthProvider_RefreshToken(t *testing.T) {
	provider, _ := NewFusionAuthProvider(&FusionAuthConfig{})
	ctx := context.Background()

	result, err := provider.RefreshToken(ctx, "some-refresh-token")

	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrFusionAuthNotImplemented)
}

func TestFusionAuthProvider_ChangePassword(t *testing.T) {
	provider, _ := NewFusionAuthProvider(&FusionAuthConfig{})
	ctx := context.Background()

	req := &ChangePasswordRequest{
		CurrentPassword: "old",
		NewPassword:     "new",
	}

	err := provider.ChangePassword(ctx, uuid.New(), req)

	assert.ErrorIs(t, err, ErrFusionAuthNotImplemented)
}

func TestFusionAuthProvider_RequestPasswordReset(t *testing.T) {
	provider, _ := NewFusionAuthProvider(&FusionAuthConfig{})
	ctx := context.Background()

	err := provider.RequestPasswordReset(ctx, "test@example.com")

	assert.ErrorIs(t, err, ErrFusionAuthNotImplemented)
}

func TestFusionAuthProvider_ConfirmPasswordReset(t *testing.T) {
	provider, _ := NewFusionAuthProvider(&FusionAuthConfig{})
	ctx := context.Background()

	err := provider.ConfirmPasswordReset(ctx, "token", "newpassword")

	assert.ErrorIs(t, err, ErrFusionAuthNotImplemented)
}

func TestFusionAuthProvider_EnableMFA(t *testing.T) {
	provider, _ := NewFusionAuthProvider(&FusionAuthConfig{})
	ctx := context.Background()

	result, err := provider.EnableMFA(ctx, uuid.New())

	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrFusionAuthNotImplemented)
}

func TestFusionAuthProvider_VerifyMFA(t *testing.T) {
	provider, _ := NewFusionAuthProvider(&FusionAuthConfig{})
	ctx := context.Background()

	err := provider.VerifyMFA(ctx, uuid.New(), "123456")

	assert.ErrorIs(t, err, ErrFusionAuthNotImplemented)
}

func TestFusionAuthProvider_DisableMFA(t *testing.T) {
	provider, _ := NewFusionAuthProvider(&FusionAuthConfig{})
	ctx := context.Background()

	err := provider.DisableMFA(ctx, uuid.New())

	assert.ErrorIs(t, err, ErrFusionAuthNotImplemented)
}

func TestFusionAuthProvider_GetSessions(t *testing.T) {
	provider, _ := NewFusionAuthProvider(&FusionAuthConfig{})
	ctx := context.Background()

	result, err := provider.GetSessions(ctx, uuid.New())

	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrFusionAuthNotImplemented)
}

func TestFusionAuthProvider_RevokeSession(t *testing.T) {
	provider, _ := NewFusionAuthProvider(&FusionAuthConfig{})
	ctx := context.Background()

	err := provider.RevokeSession(ctx, uuid.New())

	assert.ErrorIs(t, err, ErrFusionAuthNotImplemented)
}

func TestFusionAuthProvider_RevokeAllSessions(t *testing.T) {
	provider, _ := NewFusionAuthProvider(&FusionAuthConfig{})
	ctx := context.Background()

	err := provider.RevokeAllSessions(ctx, uuid.New())

	assert.ErrorIs(t, err, ErrFusionAuthNotImplemented)
}

func TestFusionAuthProvider_GetUser(t *testing.T) {
	provider, _ := NewFusionAuthProvider(&FusionAuthConfig{})
	ctx := context.Background()

	result, err := provider.GetUser(ctx, uuid.New())

	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrFusionAuthNotImplemented)
}

func TestFusionAuthProvider_UpdateUser(t *testing.T) {
	provider, _ := NewFusionAuthProvider(&FusionAuthConfig{})
	ctx := context.Background()

	req := &models.UpdateUserRequest{}

	result, err := provider.UpdateUser(ctx, uuid.New(), req)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrFusionAuthNotImplemented)
}

func TestFusionAuthProvider_DeleteUser(t *testing.T) {
	provider, _ := NewFusionAuthProvider(&FusionAuthConfig{})
	ctx := context.Background()

	err := provider.DeleteUser(ctx, uuid.New())

	assert.ErrorIs(t, err, ErrFusionAuthNotImplemented)
}

func TestFusionAuthProvider_GetAuthorizationURL(t *testing.T) {
	provider, _ := NewFusionAuthProvider(&FusionAuthConfig{})
	ctx := context.Background()

	result, err := provider.GetAuthorizationURL(ctx, "state-string")

	assert.Empty(t, result)
	assert.ErrorIs(t, err, ErrFusionAuthNotImplemented)
}

func TestFusionAuthProvider_HandleCallback(t *testing.T) {
	provider, _ := NewFusionAuthProvider(&FusionAuthConfig{})
	ctx := context.Background()

	result, err := provider.HandleCallback(ctx, "code", "state")

	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrFusionAuthNotImplemented)
}
