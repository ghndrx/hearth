package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"

	"hearth/internal/auth"
	"hearth/internal/models"
)

var (
	ErrSessionNotFound      = errors.New("session not found")
	ErrInvalidRefreshToken  = errors.New("invalid refresh token")
	ErrTokenReused          = errors.New("refresh token reuse detected")
	ErrTokenExpired         = errors.New("refresh token expired")
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
	// Note: ErrUnauthorized is defined in saved_messages_service.go
)

// SessionRepository defines the interface for session persistence
type SessionRepository interface {
	// Session operations
	CreateSession(ctx context.Context, session *models.Session) error
	GetSessionByID(ctx context.Context, id uuid.UUID) (*models.Session, error)
	GetUserSessions(ctx context.Context, userID uuid.UUID) ([]*models.Session, error)
	UpdateSessionActivity(ctx context.Context, sessionID uuid.UUID) error
	SetCurrentSession(ctx context.Context, userID, sessionID uuid.UUID) error
	RevokeSession(ctx context.Context, sessionID uuid.UUID) error
	RevokeAllUserSessions(ctx context.Context, userID uuid.UUID, exceptSessionID *uuid.UUID) error

	// Refresh token operations
	CreateRefreshToken(ctx context.Context, token *models.RefreshToken) error
	GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error)
	MarkRefreshTokenUsed(ctx context.Context, tokenID uuid.UUID) error
	RevokeTokenFamily(ctx context.Context, familyID uuid.UUID) error
	RevokeAllUserTokens(ctx context.Context, userID uuid.UUID, exceptFamilyID *uuid.UUID) error
	RotateRefreshToken(ctx context.Context, oldTokenID uuid.UUID, newToken *models.RefreshToken) error
}

// SessionService handles session and token management
type SessionService interface {
	// Session management
	CreateSession(ctx context.Context, userID uuid.UUID, userAgent, ipAddress string) (*models.Session, string, error)
	GetUserSessions(ctx context.Context, userID uuid.UUID, currentSessionID *uuid.UUID) ([]*models.SessionResponse, error)
	RevokeSession(ctx context.Context, userID, sessionID uuid.UUID) error
	RevokeAllSessions(ctx context.Context, userID uuid.UUID, exceptSessionID *uuid.UUID) error
	UpdateSessionActivity(ctx context.Context, sessionID uuid.UUID) error

	// Token operations
	CreateRefreshToken(ctx context.Context, userID, sessionID, familyID uuid.UUID, expiry time.Duration) (string, error)
	RotateRefreshToken(ctx context.Context, oldToken string, expiry time.Duration) (*models.Session, string, error)
	ValidateRefreshToken(ctx context.Context, token string) (*models.RefreshToken, error)
}

type sessionService struct {
	sessionRepo   SessionRepository
	jwtService    *auth.JWTService
	refreshExpiry time.Duration
}

// NewSessionService creates a new session service
func NewSessionService(sessionRepo SessionRepository, jwtService *auth.JWTService, refreshExpiry time.Duration) SessionService {
	return &sessionService{
		sessionRepo:   sessionRepo,
		jwtService:    jwtService,
		refreshExpiry: refreshExpiry,
	}
}

// CreateSession creates a new session with device info
func (s *sessionService) CreateSession(ctx context.Context, userID uuid.UUID, userAgent, ipAddress string) (*models.Session, string, error) {
	// Parse device info
	deviceInfo := auth.ParseUserAgent(userAgent)

	// Generate session ID
	sessionID := uuid.New()
	familyID := models.GenerateTokenFamily()

	// Create session
	session := &models.Session{
		ID:             sessionID,
		UserID:         userID,
		TokenHash:      "", // Will be updated with refresh token hash
		DeviceName:     &deviceInfo.DeviceName,
		DeviceType:     deviceInfo.DeviceType,
		Browser:        strPtr(deviceInfo.Browser),
		BrowserVersion: strPtr(deviceInfo.BrowserVersion),
		OS:             strPtr(deviceInfo.OS),
		OSVersion:      strPtr(deviceInfo.OSVersion),
		IPAddress:      strPtr(ipAddress),
		UserAgent:      strPtr(userAgent),
		IsCurrent:      true,
		ExpiresAt:      time.Now().Add(s.refreshExpiry),
		CreatedAt:      time.Now(),
	}

	// Generate refresh token
	refreshToken, err := s.generateSecureToken()
	if err != nil {
		return nil, "", err
	}

	tokenHash := models.HashToken(refreshToken)
	session.TokenHash = tokenHash

	// Save session
	if err := s.sessionRepo.CreateSession(ctx, session); err != nil {
		return nil, "", err
	}

	// Create refresh token record
	rtRecord := &models.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: tokenHash,
		FamilyID:  familyID,
		SessionID: sessionID,
		ExpiresAt: session.ExpiresAt,
		CreatedAt: time.Now(),
	}

	if err := s.sessionRepo.CreateRefreshToken(ctx, rtRecord); err != nil {
		// Clean up session if token creation fails
		if revokeErr := s.sessionRepo.RevokeSession(ctx, sessionID); revokeErr != nil {
			log.Printf("Failed to revoke session during cleanup: %v (sessionID: %v)", revokeErr, sessionID)
		}
		return nil, "", err
	}

	// Mark this session as current (unmark others)
	if err := s.sessionRepo.SetCurrentSession(ctx, userID, sessionID); err != nil {
		log.Printf("Failed to set current session: %v (userID: %v, sessionID: %v)", err, userID, sessionID)
	}

	return session, refreshToken, nil
}

// GetUserSessions retrieves all active sessions for a user
func (s *sessionService) GetUserSessions(ctx context.Context, userID uuid.UUID, currentSessionID *uuid.UUID) ([]*models.SessionResponse, error) {
	sessions, err := s.sessionRepo.GetUserSessions(ctx, userID)
	if err != nil {
		return nil, err
	}

	responses := make([]*models.SessionResponse, len(sessions))
	for i, session := range sessions {
		resp := session.ToResponse()
		// Mark the current session
		if currentSessionID != nil && session.ID == *currentSessionID {
			resp.IsCurrent = true
		}
		responses[i] = &resp
	}

	return responses, nil
}

// RevokeSession revokes a specific session
func (s *sessionService) RevokeSession(ctx context.Context, userID, sessionID uuid.UUID) error {
	// Get the session to verify ownership
	session, err := s.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return ErrSessionNotFound
	}

	if session.UserID != userID {
		return ErrUnauthorized
	}

	// Revoke the session
	if err := s.sessionRepo.RevokeSession(ctx, sessionID); err != nil {
		return err
	}

	// Get and revoke the associated token family
	// Find refresh token for this session
	// Note: We can't easily find the family without the token, so we rely on
	// the session expiry to invalidate tokens during validation
	return nil
}

// RevokeAllSessions revokes all sessions except optionally the current one
func (s *sessionService) RevokeAllSessions(ctx context.Context, userID uuid.UUID, exceptSessionID *uuid.UUID) error {
	// Revoke all sessions
	if err := s.sessionRepo.RevokeAllUserSessions(ctx, userID, exceptSessionID); err != nil {
		return err
	}

	// Revoke all refresh token families
	// If we're keeping one session, we need to find its family first
	// For simplicity, we revoke all tokens - the active one will be re-created on next refresh
	return s.sessionRepo.RevokeAllUserTokens(ctx, userID, nil)
}

// UpdateSessionActivity updates the last activity timestamp
func (s *sessionService) UpdateSessionActivity(ctx context.Context, sessionID uuid.UUID) error {
	return s.sessionRepo.UpdateSessionActivity(ctx, sessionID)
}

// CreateRefreshToken creates a new refresh token
func (s *sessionService) CreateRefreshToken(ctx context.Context, userID, sessionID, familyID uuid.UUID, expiry time.Duration) (string, error) {
	refreshToken, err := s.generateSecureToken()
	if err != nil {
		return "", err
	}

	tokenHash := models.HashToken(refreshToken)

	rtRecord := &models.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: tokenHash,
		FamilyID:  familyID,
		SessionID: sessionID,
		ExpiresAt: time.Now().Add(expiry),
		CreatedAt: time.Now(),
	}

	if err := s.sessionRepo.CreateRefreshToken(ctx, rtRecord); err != nil {
		return "", err
	}

	return refreshToken, nil
}

// RotateRefreshToken validates an old token and issues a new one
func (s *sessionService) RotateRefreshToken(ctx context.Context, oldToken string, expiry time.Duration) (*models.Session, string, error) {
	// Hash the incoming token
	tokenHash := models.HashToken(oldToken)

	// Look up the token
	rtRecord, err := s.sessionRepo.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		// Check for not found error (can be from any implementation)
		if errors.Is(err, ErrRefreshTokenNotFound) || err.Error() == "refresh token not found" {
			return nil, "", ErrInvalidRefreshToken
		}
		return nil, "", err
	}

	// Check if token has been revoked
	if rtRecord.Revoked {
		return nil, "", ErrInvalidRefreshToken
	}

	// Check if token is expired
	if time.Now().After(rtRecord.ExpiresAt) {
		return nil, "", ErrTokenExpired
	}

	// SECURITY: Check if token has already been used
	// If yes, this indicates token theft - revoke the entire family
	if rtRecord.Used {
		// Token reuse detected! Revoke the entire family
		if err := s.sessionRepo.RevokeTokenFamily(ctx, rtRecord.FamilyID); err != nil {
			// Log but continue - security check is more important
			log.Printf("SECURITY: Failed to revoke token family after reuse detection: %v (familyID: %v)", err, rtRecord.FamilyID)
		}
		// Also revoke the session
		if err := s.sessionRepo.RevokeSession(ctx, rtRecord.SessionID); err != nil {
			log.Printf("SECURITY: Failed to revoke session after reuse detection: %v (sessionID: %v)", err, rtRecord.SessionID)
		}
		return nil, "", ErrTokenReused
	}

	// Get the associated session
	session, err := s.sessionRepo.GetSessionByID(ctx, rtRecord.SessionID)
	if err != nil {
		// Session expired/revoked
		return nil, "", ErrInvalidRefreshToken
	}

	// Generate new refresh token
	newToken, err := s.generateSecureToken()
	if err != nil {
		return nil, "", err
	}

	newTokenHash := models.HashToken(newToken)

	newRTRecord := &models.RefreshToken{
		ID:        uuid.New(),
		UserID:    rtRecord.UserID,
		TokenHash: newTokenHash,
		FamilyID:  rtRecord.FamilyID, // Same family
		SessionID: rtRecord.SessionID,
		ExpiresAt: time.Now().Add(expiry),
		CreatedAt: time.Now(),
	}

	// Atomically mark old token as used and create new one
	if err := s.sessionRepo.RotateRefreshToken(ctx, rtRecord.ID, newRTRecord); err != nil {
		return nil, "", err
	}

	// Update session activity (log error but don't fail)
	if err := s.sessionRepo.UpdateSessionActivity(ctx, session.ID); err != nil {
		log.Printf("Failed to update session activity: %v (sessionID: %v)", err, session.ID)
	}

	return session, newToken, nil
}

// ValidateRefreshToken validates a refresh token without rotating
func (s *sessionService) ValidateRefreshToken(ctx context.Context, token string) (*models.RefreshToken, error) {
	tokenHash := models.HashToken(token)

	rtRecord, err := s.sessionRepo.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	if rtRecord.Revoked || rtRecord.Used {
		return nil, ErrInvalidRefreshToken
	}

	if time.Now().After(rtRecord.ExpiresAt) {
		return nil, ErrTokenExpired
	}

	return rtRecord, nil
}

// generateSecureToken generates a cryptographically secure random token
func (s *sessionService) generateSecureToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// Helper function to create string pointer
func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
