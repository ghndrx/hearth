package auth

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"hearth/internal/models"
)

// FusionAuthProvider implements Provider for FusionAuth
// This is a stub - implement when FusionAuth integration is needed
type FusionAuthProvider struct {
	// config would go here
}

// FusionAuthConfig is defined in provider.go

var ErrFusionAuthNotImplemented = errors.New("FusionAuth provider not implemented")

// NewFusionAuthProvider creates a new FusionAuth provider
func NewFusionAuthProvider(cfg *FusionAuthConfig) (*FusionAuthProvider, error) {
	return &FusionAuthProvider{}, nil
}

// Implementation stubs

func (p *FusionAuthProvider) Name() string                { return "FusionAuth" }
func (p *FusionAuthProvider) Type() ProviderType          { return ProviderFusionAuth }

func (p *FusionAuthProvider) Register(ctx context.Context, req *RegisterRequest) (*AuthResult, error) {
	return nil, ErrFusionAuthNotImplemented
}

func (p *FusionAuthProvider) Login(ctx context.Context, req *LoginRequest) (*AuthResult, error) {
	return nil, ErrFusionAuthNotImplemented
}

func (p *FusionAuthProvider) Logout(ctx context.Context, sessionID uuid.UUID) error {
	return ErrFusionAuthNotImplemented
}

func (p *FusionAuthProvider) RefreshToken(ctx context.Context, refreshToken string) (*AuthResult, error) {
	return nil, ErrFusionAuthNotImplemented
}

func (p *FusionAuthProvider) ChangePassword(ctx context.Context, userID uuid.UUID, req *ChangePasswordRequest) error {
	return ErrFusionAuthNotImplemented
}

func (p *FusionAuthProvider) RequestPasswordReset(ctx context.Context, email string) error {
	return ErrFusionAuthNotImplemented
}

func (p *FusionAuthProvider) ConfirmPasswordReset(ctx context.Context, token, newPassword string) error {
	return ErrFusionAuthNotImplemented
}

func (p *FusionAuthProvider) EnableMFA(ctx context.Context, userID uuid.UUID) (*MFASetup, error) {
	return nil, ErrFusionAuthNotImplemented
}

func (p *FusionAuthProvider) VerifyMFA(ctx context.Context, userID uuid.UUID, code string) error {
	return ErrFusionAuthNotImplemented
}

func (p *FusionAuthProvider) DisableMFA(ctx context.Context, userID uuid.UUID) error {
	return ErrFusionAuthNotImplemented
}

func (p *FusionAuthProvider) GetSessions(ctx context.Context, userID uuid.UUID) ([]*models.Session, error) {
	return nil, ErrFusionAuthNotImplemented
}

func (p *FusionAuthProvider) RevokeSession(ctx context.Context, sessionID uuid.UUID) error {
	return ErrFusionAuthNotImplemented
}

func (p *FusionAuthProvider) RevokeAllSessions(ctx context.Context, userID uuid.UUID) error {
	return ErrFusionAuthNotImplemented
}

func (p *FusionAuthProvider) GetUser(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	return nil, ErrFusionAuthNotImplemented
}

func (p *FusionAuthProvider) UpdateUser(ctx context.Context, userID uuid.UUID, req *models.UpdateUserRequest) (*models.User, error) {
	return nil, ErrFusionAuthNotImplemented
}

func (p *FusionAuthProvider) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	return ErrFusionAuthNotImplemented
}

func (p *FusionAuthProvider) GetAuthorizationURL(ctx context.Context, state string) (string, error) {
	return "", ErrFusionAuthNotImplemented
}

func (p *FusionAuthProvider) HandleCallback(ctx context.Context, code, state string) (*AuthResult, error) {
	return nil, ErrFusionAuthNotImplemented
}
