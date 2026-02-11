package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hearth/internal/auth"
	"hearth/internal/models"
)

// Mock implementations

type mockUserRepository struct {
	users    map[uuid.UUID]*models.User
	byEmail  map[string]*models.User
	byName   map[string]*models.User
	createFn func(ctx context.Context, user *models.User) error
	updateFn func(ctx context.Context, user *models.User) error
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{
		users:   make(map[uuid.UUID]*models.User),
		byEmail: make(map[string]*models.User),
		byName:  make(map[string]*models.User),
	}
}

func (m *mockUserRepository) Create(ctx context.Context, user *models.User) error {
	if m.createFn != nil {
		return m.createFn(ctx, user)
	}
	m.users[user.ID] = user
	m.byEmail[user.Email] = user
	m.byName[user.Username] = user
	return nil
}

func (m *mockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	return m.users[id], nil
}

func (m *mockUserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	return m.byName[username], nil
}

func (m *mockUserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	return m.byEmail[email], nil
}

func (m *mockUserRepository) Update(ctx context.Context, user *models.User) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, user)
	}
	m.users[user.ID] = user
	m.byEmail[user.Email] = user
	m.byName[user.Username] = user
	return nil
}

func (m *mockUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	user := m.users[id]
	if user != nil {
		delete(m.users, id)
		delete(m.byEmail, user.Email)
		delete(m.byName, user.Username)
	}
	return nil
}

func (m *mockUserRepository) GetFriends(ctx context.Context, userID uuid.UUID) ([]*models.User, error) {
	return nil, nil
}
func (m *mockUserRepository) AddFriend(ctx context.Context, userID, friendID uuid.UUID) error {
	return nil
}
func (m *mockUserRepository) RemoveFriend(ctx context.Context, userID, friendID uuid.UUID) error {
	return nil
}
func (m *mockUserRepository) GetBlockedUsers(ctx context.Context, userID uuid.UUID) ([]*models.User, error) {
	return nil, nil
}
func (m *mockUserRepository) BlockUser(ctx context.Context, userID, blockedID uuid.UUID) error {
	return nil
}
func (m *mockUserRepository) UnblockUser(ctx context.Context, userID, blockedID uuid.UUID) error {
	return nil
}
func (m *mockUserRepository) UpdatePresence(ctx context.Context, userID uuid.UUID, status models.PresenceStatus) error {
	return nil
}
func (m *mockUserRepository) GetPresence(ctx context.Context, userID uuid.UUID) (*models.Presence, error) {
	return nil, nil
}
func (m *mockUserRepository) GetPresenceBulk(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*models.Presence, error) {
	return nil, nil
}

func (m *mockUserRepository) addUser(user *models.User) {
	m.users[user.ID] = user
	m.byEmail[user.Email] = user
	m.byName[user.Username] = user
}

type mockCacheService struct {
	data map[string][]byte
}

func newMockCacheService() *mockCacheService {
	return &mockCacheService{
		data: make(map[string][]byte),
	}
}

func (m *mockCacheService) GetUser(ctx context.Context, id uuid.UUID) (*models.User, error) {
	return nil, errors.New("not found")
}
func (m *mockCacheService) SetUser(ctx context.Context, user *models.User, ttl time.Duration) error {
	return nil
}
func (m *mockCacheService) DeleteUser(ctx context.Context, id uuid.UUID) error { return nil }
func (m *mockCacheService) GetServer(ctx context.Context, id uuid.UUID) (*models.Server, error) {
	return nil, errors.New("not found")
}
func (m *mockCacheService) SetServer(ctx context.Context, server *models.Server, ttl time.Duration) error {
	return nil
}
func (m *mockCacheService) DeleteServer(ctx context.Context, id uuid.UUID) error { return nil }
func (m *mockCacheService) GetChannel(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
	return nil, errors.New("not found")
}
func (m *mockCacheService) SetChannel(ctx context.Context, channel *models.Channel, ttl time.Duration) error {
	return nil
}
func (m *mockCacheService) DeleteChannel(ctx context.Context, id uuid.UUID) error { return nil }

func (m *mockCacheService) Get(ctx context.Context, key string) ([]byte, error) {
	if v, ok := m.data[key]; ok {
		return v, nil
	}
	return nil, errors.New("not found")
}

func (m *mockCacheService) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	m.data[key] = value
	return nil
}

func (m *mockCacheService) Delete(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

// Test helpers

func newTestAuthService(repo UserRepository, cache CacheService) *AuthService {
	jwtService := auth.NewJWTService("test-secret-key-32-characters!!", 15*time.Minute, 7*24*time.Hour)
	return NewAuthService(repo, jwtService, cache, true, false)
}

func createTestUser(email, username, password string) *models.User {
	hash, _ := auth.HashPassword(password)
	return &models.User{
		ID:            uuid.New(),
		Email:         email,
		Username:      username,
		Discriminator: "0001",
		PasswordHash:  hash,
		Status:        models.StatusOffline,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

// Tests

func TestAuthService_Register_Success(t *testing.T) {
	repo := newMockUserRepository()
	cache := newMockCacheService()
	svc := newTestAuthService(repo, cache)

	req := &RegisterRequest{
		Email:    "test@example.com",
		Username: "testuser",
		Password: "SecurePass123",
	}

	user, tokens, err := svc.Register(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, user)
	assert.NotNil(t, tokens)
	assert.Equal(t, req.Email, user.Email)
	assert.Equal(t, req.Username, user.Username)
	assert.NotEmpty(t, tokens.AccessToken)
	assert.NotEmpty(t, tokens.RefreshToken)
	assert.Equal(t, "Bearer", tokens.TokenType)
}

func TestAuthService_Register_GeneratesDiscriminator(t *testing.T) {
	repo := newMockUserRepository()
	cache := newMockCacheService()
	svc := newTestAuthService(repo, cache)

	req := &RegisterRequest{
		Email:    "test@example.com",
		Username: "testuser",
		Password: "SecurePass123",
	}

	user, _, err := svc.Register(context.Background(), req)

	require.NoError(t, err)
	assert.NotEmpty(t, user.Discriminator, "Discriminator should be generated")
	assert.Len(t, user.Discriminator, 4, "Discriminator should be 4 characters")
}

func TestAuthService_Register_EmailTaken(t *testing.T) {
	repo := newMockUserRepository()
	cache := newMockCacheService()
	svc := newTestAuthService(repo, cache)

	// Pre-populate user
	existingUser := createTestUser("test@example.com", "existinguser", "Password123")
	repo.addUser(existingUser)

	req := &RegisterRequest{
		Email:    "test@example.com",
		Username: "newuser",
		Password: "SecurePass123",
	}

	user, tokens, err := svc.Register(context.Background(), req)

	assert.ErrorIs(t, err, ErrEmailTaken)
	assert.Nil(t, user)
	assert.Nil(t, tokens)
}

func TestAuthService_Register_UsernameTaken(t *testing.T) {
	repo := newMockUserRepository()
	cache := newMockCacheService()
	svc := newTestAuthService(repo, cache)

	// Pre-populate user
	existingUser := createTestUser("existing@example.com", "testuser", "Password123")
	repo.addUser(existingUser)

	req := &RegisterRequest{
		Email:    "new@example.com",
		Username: "testuser",
		Password: "SecurePass123",
	}

	user, tokens, err := svc.Register(context.Background(), req)

	assert.ErrorIs(t, err, ErrUsernameTaken)
	assert.Nil(t, user)
	assert.Nil(t, tokens)
}

func TestAuthService_Register_WeakPassword(t *testing.T) {
	repo := newMockUserRepository()
	cache := newMockCacheService()
	svc := newTestAuthService(repo, cache)

	testCases := []struct {
		name     string
		password string
		err      error
	}{
		{"too short", "Pass1", auth.ErrPasswordTooShort},
		{"no uppercase", "password123", auth.ErrPasswordWeak},
		{"no lowercase", "PASSWORD123", auth.ErrPasswordWeak},
		{"no number", "SecurePassword", auth.ErrPasswordWeak},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &RegisterRequest{
				Email:    "test@example.com",
				Username: "testuser",
				Password: tc.password,
			}

			user, tokens, err := svc.Register(context.Background(), req)

			assert.ErrorIs(t, err, tc.err)
			assert.Nil(t, user)
			assert.Nil(t, tokens)
		})
	}
}

func TestAuthService_Register_Disabled(t *testing.T) {
	repo := newMockUserRepository()
	cache := newMockCacheService()
	jwtService := auth.NewJWTService("test-secret-key-32-characters!!", 15*time.Minute, 7*24*time.Hour)
	svc := NewAuthService(repo, jwtService, cache, false, false) // registration disabled

	req := &RegisterRequest{
		Email:    "test@example.com",
		Username: "testuser",
		Password: "SecurePass123",
	}

	user, tokens, err := svc.Register(context.Background(), req)

	assert.ErrorIs(t, err, ErrRegistrationClosed)
	assert.Nil(t, user)
	assert.Nil(t, tokens)
}

func TestAuthService_Register_InviteRequired(t *testing.T) {
	repo := newMockUserRepository()
	cache := newMockCacheService()
	jwtService := auth.NewJWTService("test-secret-key-32-characters!!", 15*time.Minute, 7*24*time.Hour)
	svc := NewAuthService(repo, jwtService, cache, true, true) // invite only

	req := &RegisterRequest{
		Email:    "test@example.com",
		Username: "testuser",
		Password: "SecurePass123",
		// No invite code
	}

	user, tokens, err := svc.Register(context.Background(), req)

	assert.ErrorIs(t, err, ErrInviteRequired)
	assert.Nil(t, user)
	assert.Nil(t, tokens)
}

func TestAuthService_Login_Success(t *testing.T) {
	repo := newMockUserRepository()
	cache := newMockCacheService()
	svc := newTestAuthService(repo, cache)

	password := "SecurePass123"
	existingUser := createTestUser("test@example.com", "testuser", password)
	repo.addUser(existingUser)

	user, tokens, err := svc.Login(context.Background(), "test@example.com", password)

	require.NoError(t, err)
	assert.NotNil(t, user)
	assert.NotNil(t, tokens)
	assert.Equal(t, existingUser.ID, user.ID)
	assert.NotEmpty(t, tokens.AccessToken)
	assert.NotEmpty(t, tokens.RefreshToken)
}

func TestAuthService_Login_InvalidEmail(t *testing.T) {
	repo := newMockUserRepository()
	cache := newMockCacheService()
	svc := newTestAuthService(repo, cache)

	user, tokens, err := svc.Login(context.Background(), "nonexistent@example.com", "Password123")

	assert.ErrorIs(t, err, ErrInvalidCredentials)
	assert.Nil(t, user)
	assert.Nil(t, tokens)
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	repo := newMockUserRepository()
	cache := newMockCacheService()
	svc := newTestAuthService(repo, cache)

	existingUser := createTestUser("test@example.com", "testuser", "CorrectPass123")
	repo.addUser(existingUser)

	user, tokens, err := svc.Login(context.Background(), "test@example.com", "WrongPass123")

	assert.ErrorIs(t, err, ErrInvalidCredentials)
	assert.Nil(t, user)
	assert.Nil(t, tokens)
}

func TestAuthService_RefreshToken_Success(t *testing.T) {
	repo := newMockUserRepository()
	cache := newMockCacheService()
	svc := newTestAuthService(repo, cache)

	password := "SecurePass123"
	existingUser := createTestUser("test@example.com", "testuser", password)
	repo.addUser(existingUser)

	// Login to get initial tokens
	_, initialTokens, err := svc.Login(context.Background(), "test@example.com", password)
	require.NoError(t, err)

	// Refresh tokens
	newTokens, err := svc.RefreshToken(context.Background(), initialTokens.RefreshToken)

	require.NoError(t, err)
	assert.NotNil(t, newTokens)
	assert.NotEmpty(t, newTokens.AccessToken)
	assert.NotEmpty(t, newTokens.RefreshToken)
	assert.NotEqual(t, initialTokens.AccessToken, newTokens.AccessToken)
	assert.NotEqual(t, initialTokens.RefreshToken, newTokens.RefreshToken)
}

func TestAuthService_RefreshToken_Invalid(t *testing.T) {
	repo := newMockUserRepository()
	cache := newMockCacheService()
	svc := newTestAuthService(repo, cache)

	tokens, err := svc.RefreshToken(context.Background(), "invalid-token")

	assert.Error(t, err)
	assert.Nil(t, tokens)
}

func TestAuthService_RefreshToken_WithAccessToken(t *testing.T) {
	repo := newMockUserRepository()
	cache := newMockCacheService()
	svc := newTestAuthService(repo, cache)

	password := "SecurePass123"
	existingUser := createTestUser("test@example.com", "testuser", password)
	repo.addUser(existingUser)

	// Login to get tokens
	_, initialTokens, err := svc.Login(context.Background(), "test@example.com", password)
	require.NoError(t, err)

	// Try to refresh with access token (should fail)
	tokens, err := svc.RefreshToken(context.Background(), initialTokens.AccessToken)

	assert.Error(t, err)
	assert.Nil(t, tokens)
}

func TestAuthService_ValidateToken_Success(t *testing.T) {
	repo := newMockUserRepository()
	cache := newMockCacheService()
	svc := newTestAuthService(repo, cache)

	password := "SecurePass123"
	existingUser := createTestUser("test@example.com", "testuser", password)
	repo.addUser(existingUser)

	// Login
	_, tokens, err := svc.Login(context.Background(), "test@example.com", password)
	require.NoError(t, err)

	// Validate token
	userID, err := svc.ValidateToken(context.Background(), tokens.AccessToken)

	require.NoError(t, err)
	assert.Equal(t, existingUser.ID, userID)
}

func TestAuthService_ValidateToken_Invalid(t *testing.T) {
	repo := newMockUserRepository()
	cache := newMockCacheService()
	svc := newTestAuthService(repo, cache)

	userID, err := svc.ValidateToken(context.Background(), "invalid-token")

	assert.Error(t, err)
	assert.Equal(t, uuid.Nil, userID)
}

func TestAuthService_ValidateToken_RefreshTokenNotAllowed(t *testing.T) {
	repo := newMockUserRepository()
	cache := newMockCacheService()
	svc := newTestAuthService(repo, cache)

	password := "SecurePass123"
	existingUser := createTestUser("test@example.com", "testuser", password)
	repo.addUser(existingUser)

	// Login
	_, tokens, err := svc.Login(context.Background(), "test@example.com", password)
	require.NoError(t, err)

	// Try to validate with refresh token (should fail)
	userID, err := svc.ValidateToken(context.Background(), tokens.RefreshToken)

	assert.Error(t, err)
	assert.Equal(t, uuid.Nil, userID)
}

func TestAuthService_Logout(t *testing.T) {
	repo := newMockUserRepository()
	cache := newMockCacheService()
	svc := newTestAuthService(repo, cache)

	password := "SecurePass123"
	existingUser := createTestUser("test@example.com", "testuser", password)
	repo.addUser(existingUser)

	// Login
	_, tokens, err := svc.Login(context.Background(), "test@example.com", password)
	require.NoError(t, err)

	// Logout
	err = svc.Logout(context.Background(), tokens.AccessToken, tokens.RefreshToken)
	require.NoError(t, err)

	// Token should be revoked - verify through cache
	// Check that at least one key starts with "revoked:"
	hasRevokedKey := false
	for key := range cache.data {
		if len(key) > 8 && key[:8] == "revoked:" {
			hasRevokedKey = true
			break
		}
	}
	assert.True(t, hasRevokedKey, "Should have revoked token entries in cache")
}

func TestAuthService_TokenResponse_Fields(t *testing.T) {
	repo := newMockUserRepository()
	cache := newMockCacheService()
	svc := newTestAuthService(repo, cache)

	password := "SecurePass123"
	existingUser := createTestUser("test@example.com", "testuser", password)
	repo.addUser(existingUser)

	_, tokens, err := svc.Login(context.Background(), "test@example.com", password)

	require.NoError(t, err)
	assert.Equal(t, "Bearer", tokens.TokenType)
	assert.Greater(t, tokens.ExpiresIn, 0)
	// 15 minutes = 900 seconds
	assert.Equal(t, 900, tokens.ExpiresIn)
}

// Benchmark tests

func BenchmarkAuthService_Register(b *testing.B) {
	repo := newMockUserRepository()
	cache := newMockCacheService()
	svc := newTestAuthService(repo, cache)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := &RegisterRequest{
			Email:    "test@example.com",
			Username: "testuser",
			Password: "SecurePass123",
		}
		// Reset repo between iterations
		repo.users = make(map[uuid.UUID]*models.User)
		repo.byEmail = make(map[string]*models.User)
		repo.byName = make(map[string]*models.User)

		_, _, _ = svc.Register(context.Background(), req)
	}
}

func BenchmarkAuthService_Login(b *testing.B) {
	repo := newMockUserRepository()
	cache := newMockCacheService()
	svc := newTestAuthService(repo, cache)

	password := "SecurePass123"
	existingUser := createTestUser("test@example.com", "testuser", password)
	repo.addUser(existingUser)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = svc.Login(context.Background(), "test@example.com", password)
	}
}

func BenchmarkAuthService_ValidateToken(b *testing.B) {
	repo := newMockUserRepository()
	cache := newMockCacheService()
	svc := newTestAuthService(repo, cache)

	password := "SecurePass123"
	existingUser := createTestUser("test@example.com", "testuser", password)
	repo.addUser(existingUser)

	_, tokens, _ := svc.Login(context.Background(), "test@example.com", password)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.ValidateToken(context.Background(), tokens.AccessToken)
	}
}
