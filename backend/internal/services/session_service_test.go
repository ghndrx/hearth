package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"hearth/internal/auth"
	"hearth/internal/models"
)

// MockSessionRepository implements session repository for testing
type MockSessionRepository struct {
	mock.Mock
}

func (m *MockSessionRepository) CreateSession(ctx context.Context, session *models.Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockSessionRepository) GetSessionByID(ctx context.Context, id uuid.UUID) (*models.Session, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Session), args.Error(1)
}

func (m *MockSessionRepository) GetUserSessions(ctx context.Context, userID uuid.UUID) ([]*models.Session, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Session), args.Error(1)
}

func (m *MockSessionRepository) UpdateSessionActivity(ctx context.Context, sessionID uuid.UUID) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

func (m *MockSessionRepository) SetCurrentSession(ctx context.Context, userID, sessionID uuid.UUID) error {
	args := m.Called(ctx, userID, sessionID)
	return args.Error(0)
}

func (m *MockSessionRepository) RevokeSession(ctx context.Context, sessionID uuid.UUID) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

func (m *MockSessionRepository) RevokeAllUserSessions(ctx context.Context, userID uuid.UUID, exceptSessionID *uuid.UUID) error {
	args := m.Called(ctx, userID, exceptSessionID)
	return args.Error(0)
}

func (m *MockSessionRepository) CreateRefreshToken(ctx context.Context, token *models.RefreshToken) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockSessionRepository) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error) {
	args := m.Called(ctx, tokenHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RefreshToken), args.Error(1)
}

func (m *MockSessionRepository) MarkRefreshTokenUsed(ctx context.Context, tokenID uuid.UUID) error {
	args := m.Called(ctx, tokenID)
	return args.Error(0)
}

func (m *MockSessionRepository) RevokeTokenFamily(ctx context.Context, familyID uuid.UUID) error {
	args := m.Called(ctx, familyID)
	return args.Error(0)
}

func (m *MockSessionRepository) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID, exceptFamilyID *uuid.UUID) error {
	args := m.Called(ctx, userID, exceptFamilyID)
	return args.Error(0)
}

func (m *MockSessionRepository) RotateRefreshToken(ctx context.Context, oldTokenID uuid.UUID, newToken *models.RefreshToken) error {
	args := m.Called(ctx, oldTokenID, newToken)
	return args.Error(0)
}

//lint:ignore U1000 test factory helper
func testSessionService(repo SessionRepository) SessionService {
	jwtService := auth.NewJWTService("test-secret", 15*time.Minute, 7*24*time.Hour)
	return NewSessionService(repo, jwtService, 30*24*time.Hour)
}

func TestDeviceInfoParsing(t *testing.T) {
	tests := []struct {
		name        string
		userAgent   string
		wantDevice  models.DeviceType
		wantBrowser string
	}{
		{
			name:        "Chrome on Windows Desktop",
			userAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36",
			wantDevice:  models.DeviceTypeDesktop,
			wantBrowser: "Chrome",
		},
		{
			name:        "Safari on iPhone",
			userAgent:   "Mozilla/5.0 (iPhone; CPU iPhone OS 17_1 like Mac OS X) AppleWebKit/605.1.15 Safari/604.1",
			wantDevice:  models.DeviceTypeMobile,
			wantBrowser: "Safari",
		},
		{
			name:        "Chrome on Android",
			userAgent:   "Mozilla/5.0 (Linux; Android 13; SM-G991B) AppleWebKit/537.36 Chrome/120.0.6099.43 Mobile Safari/537.36",
			wantDevice:  models.DeviceTypeMobile,
			wantBrowser: "Chrome",
		},
		{
			name:        "iPad Safari",
			userAgent:   "Mozilla/5.0 (iPad; CPU OS 17_1 like Mac OS X) AppleWebKit/605.1.15 Safari/604.1",
			wantDevice:  models.DeviceTypeTablet,
			wantBrowser: "Safari",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := auth.ParseUserAgent(tt.userAgent)
			assert.Equal(t, tt.wantDevice, info.DeviceType, "Device type mismatch")
			assert.Equal(t, tt.wantBrowser, info.Browser, "Browser mismatch")
		})
	}
}

func TestTokenHashing(t *testing.T) {
	token1 := "test-token-123"
	token2 := "test-token-456"

	hash1 := models.HashToken(token1)
	hash2 := models.HashToken(token2)
	hash1Again := models.HashToken(token1)

	// Same input produces same hash
	assert.Equal(t, hash1, hash1Again)
	// Different inputs produce different hashes
	assert.NotEqual(t, hash1, hash2)
	// Hash is hex encoded (64 chars for SHA-256)
	assert.Len(t, hash1, 64)
}

func TestRefreshTokenIsValid(t *testing.T) {
	tests := []struct {
		name  string
		token models.RefreshToken
		want  bool
	}{
		{
			name: "Valid token",
			token: models.RefreshToken{
				Used:      false,
				Revoked:   false,
				ExpiresAt: time.Now().Add(time.Hour),
			},
			want: true,
		},
		{
			name: "Used token",
			token: models.RefreshToken{
				Used:      true,
				Revoked:   false,
				ExpiresAt: time.Now().Add(time.Hour),
			},
			want: false,
		},
		{
			name: "Revoked token",
			token: models.RefreshToken{
				Used:      false,
				Revoked:   true,
				ExpiresAt: time.Now().Add(time.Hour),
			},
			want: false,
		},
		{
			name: "Expired token",
			token: models.RefreshToken{
				Used:      false,
				Revoked:   false,
				ExpiresAt: time.Now().Add(-time.Hour),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.token.IsValid()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSessionToResponse(t *testing.T) {
	browser := "Chrome"
	os := "Windows"
	city := "San Francisco"
	country := "US"
	lastUsed := time.Now().Add(-time.Hour)

	session := &models.Session{
		ID:              uuid.New(),
		DeviceName:      strPtr("Chrome on Windows"),
		DeviceType:      models.DeviceTypeDesktop,
		Browser:         &browser,
		OS:              &os,
		LocationCity:    &city,
		LocationCountry: &country,
		IsCurrent:       true,
		LastUsed:        &lastUsed,
		CreatedAt:       time.Now().Add(-24 * time.Hour),
	}

	response := session.ToResponse()

	assert.Equal(t, session.ID, response.ID)
	assert.Equal(t, "Chrome on Windows", response.DeviceName)
	assert.Equal(t, models.DeviceTypeDesktop, response.DeviceType)
	assert.Equal(t, "Chrome", response.Browser)
	assert.Equal(t, "Windows", response.OS)
	assert.Equal(t, "San Francisco", response.LocationCity)
	assert.Equal(t, "US", response.LocationCountry)
	assert.True(t, response.IsCurrent)
}

func TestSessionToResponse_BuildsDeviceName(t *testing.T) {
	browser := "Firefox"
	os := "Linux"

	// Session without explicit device name should build one
	session := &models.Session{
		ID:         uuid.New(),
		DeviceType: models.DeviceTypeDesktop,
		Browser:    &browser,
		OS:         &os,
		CreatedAt:  time.Now(),
	}

	response := session.ToResponse()

	assert.Equal(t, "Firefox on Linux", response.DeviceName)
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name          string
		remoteAddr    string
		xForwardedFor string
		xRealIP       string
		want          string
	}{
		{
			name:          "X-Forwarded-For takes priority",
			remoteAddr:    "127.0.0.1:8080",
			xForwardedFor: "203.0.113.195, 70.41.3.18",
			xRealIP:       "192.168.1.1",
			want:          "203.0.113.195",
		},
		{
			name:       "X-Real-IP used when no X-Forwarded-For",
			remoteAddr: "127.0.0.1:8080",
			xRealIP:    "192.168.1.1",
			want:       "192.168.1.1",
		},
		{
			name:       "Remote addr fallback",
			remoteAddr: "10.0.0.1:54321",
			want:       "10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := auth.GetClientIP(tt.remoteAddr, tt.xForwardedFor, tt.xRealIP)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGenerateTokenFamily(t *testing.T) {
	family1 := models.GenerateTokenFamily()
	family2 := models.GenerateTokenFamily()

	// Each call generates a unique ID
	assert.NotEqual(t, family1, family2)
	// IDs are valid UUIDs
	assert.NotEqual(t, uuid.Nil, family1)
	assert.NotEqual(t, uuid.Nil, family2)
}
