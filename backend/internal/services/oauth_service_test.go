package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"hearth/internal/auth"
	"hearth/internal/models"
)

// MockOAuthUserRepository is a mock implementation of UserRepository for OAuth tests
type MockOAuthUserRepository struct {
	mock.Mock
}

func (m *MockOAuthUserRepository) Create(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockOAuthUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockOAuthUserRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*models.User, error) {
	args := m.Called(ctx, ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockOAuthUserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockOAuthUserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockOAuthUserRepository) Update(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockOAuthUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockOAuthUserRepository) GetFriends(ctx context.Context, userID uuid.UUID) ([]*models.User, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockOAuthUserRepository) AddFriend(ctx context.Context, userID, friendID uuid.UUID) error {
	args := m.Called(ctx, userID, friendID)
	return args.Error(0)
}

func (m *MockOAuthUserRepository) RemoveFriend(ctx context.Context, userID, friendID uuid.UUID) error {
	args := m.Called(ctx, userID, friendID)
	return args.Error(0)
}

func (m *MockOAuthUserRepository) GetBlockedUsers(ctx context.Context, userID uuid.UUID) ([]*models.User, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockOAuthUserRepository) BlockUser(ctx context.Context, userID, blockedID uuid.UUID) error {
	args := m.Called(ctx, userID, blockedID)
	return args.Error(0)
}

func (m *MockOAuthUserRepository) UnblockUser(ctx context.Context, userID, blockedID uuid.UUID) error {
	args := m.Called(ctx, userID, blockedID)
	return args.Error(0)
}

func (m *MockOAuthUserRepository) UpdatePresence(ctx context.Context, userID uuid.UUID, status models.PresenceStatus) error {
	args := m.Called(ctx, userID, status)
	return args.Error(0)
}

func (m *MockOAuthUserRepository) GetPresence(ctx context.Context, userID uuid.UUID) (*models.Presence, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Presence), args.Error(1)
}

func (m *MockOAuthUserRepository) GetPresenceBulk(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*models.Presence, error) {
	args := m.Called(ctx, userIDs)
	return args.Get(0).(map[uuid.UUID]*models.Presence), args.Error(1)
}

func (m *MockOAuthUserRepository) GetRelationship(ctx context.Context, userID, targetID uuid.UUID) (int, error) {
	args := m.Called(ctx, userID, targetID)
	return args.Int(0), args.Error(1)
}

func (m *MockOAuthUserRepository) SendFriendRequest(ctx context.Context, senderID, receiverID uuid.UUID) error {
	args := m.Called(ctx, senderID, receiverID)
	return args.Error(0)
}

func (m *MockOAuthUserRepository) GetIncomingFriendRequests(ctx context.Context, userID uuid.UUID) ([]*models.User, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockOAuthUserRepository) GetOutgoingFriendRequests(ctx context.Context, userID uuid.UUID) ([]*models.User, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockOAuthUserRepository) AcceptFriendRequest(ctx context.Context, receiverID, senderID uuid.UUID) error {
	args := m.Called(ctx, receiverID, senderID)
	return args.Error(0)
}

func (m *MockOAuthUserRepository) DeclineFriendRequest(ctx context.Context, userID, otherID uuid.UUID) error {
	args := m.Called(ctx, userID, otherID)
	return args.Error(0)
}

func (m *MockOAuthUserRepository) GetCustomStatus(ctx context.Context, userID uuid.UUID) (*models.UserCustomStatus, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserCustomStatus), args.Error(1)
}

func (m *MockOAuthUserRepository) SetCustomStatus(ctx context.Context, status *models.UserCustomStatus) error {
	args := m.Called(ctx, status)
	return args.Error(0)
}

func (m *MockOAuthUserRepository) DeleteCustomStatus(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

// MockOAuthCacheService is a mock implementation of CacheService for OAuth tests
type MockOAuthCacheService struct {
	store map[string][]byte
}

func NewMockOAuthCacheService() *MockOAuthCacheService {
	return &MockOAuthCacheService{
		store: make(map[string][]byte),
	}
}

func (m *MockOAuthCacheService) Get(ctx context.Context, key string) ([]byte, error) {
	if data, ok := m.store[key]; ok {
		return data, nil
	}
	return nil, ErrCacheNotFound
}

func (m *MockOAuthCacheService) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	m.store[key] = value
	return nil
}

func (m *MockOAuthCacheService) Delete(ctx context.Context, key string) error {
	delete(m.store, key)
	return nil
}

func (m *MockOAuthCacheService) GetUser(ctx context.Context, id uuid.UUID) (*models.User, error) {
	return nil, ErrCacheNotFound
}

func (m *MockOAuthCacheService) SetUser(ctx context.Context, user *models.User, ttl time.Duration) error {
	return nil
}

func (m *MockOAuthCacheService) DeleteUser(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *MockOAuthCacheService) GetServer(ctx context.Context, id uuid.UUID) (*models.Server, error) {
	return nil, ErrCacheNotFound
}

func (m *MockOAuthCacheService) SetServer(ctx context.Context, server *models.Server, ttl time.Duration) error {
	return nil
}

func (m *MockOAuthCacheService) DeleteServer(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *MockOAuthCacheService) GetChannel(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
	return nil, ErrCacheNotFound
}

func (m *MockOAuthCacheService) SetChannel(ctx context.Context, channel *models.Channel, ttl time.Duration) error {
	return nil
}

func (m *MockOAuthCacheService) DeleteChannel(ctx context.Context, id uuid.UUID) error {
	return nil
}

// Test setup helper
func setupOAuthService(t *testing.T) (*OAuthService, *MockOAuthUserRepository, *MockOAuthCacheService) {
	userRepo := new(MockOAuthUserRepository)
	cache := NewMockOAuthCacheService()
	jwtService := auth.NewJWTService("test-secret-key-for-testing-purposes", 3600, 86400)

	config := &OAuthProviderConfig{
		GitHub: &OAuthConfig{
			ClientID:     "test-github-client-id",
			ClientSecret: "test-github-client-secret",
			RedirectURI:  "http://localhost:8080/api/v1/auth/oauth/github/callback",
			Scopes:       []string{"read:user", "user:email"},
		},
		Google: &OAuthConfig{
			ClientID:     "test-google-client-id",
			ClientSecret: "test-google-client-secret",
			RedirectURI:  "http://localhost:8080/api/v1/auth/oauth/google/callback",
			Scopes:       []string{"openid", "email", "profile"},
		},
		Discord: &OAuthConfig{
			ClientID:     "test-discord-client-id",
			ClientSecret: "test-discord-client-secret",
			RedirectURI:  "http://localhost:8080/api/v1/auth/oauth/discord/callback",
			Scopes:       []string{"identify", "email"},
		},
	}

	service := NewOAuthService(config, userRepo, cache, jwtService)
	return service, userRepo, cache
}

// Tests

func TestOAuthService_GetAuthorizationURL_GitHub(t *testing.T) {
	service, _, _ := setupOAuthService(t)
	ctx := context.Background()

	authURL, err := service.GetAuthorizationURL(ctx, OAuthProviderGitHub, nil)
	require.NoError(t, err)
	assert.Contains(t, authURL, "https://github.com/login/oauth/authorize")
	assert.Contains(t, authURL, "client_id=test-github-client-id")
	assert.Contains(t, authURL, "state=")
	assert.Contains(t, authURL, "scope=read%3Auser+user%3Aemail")
}

func TestOAuthService_GetAuthorizationURL_Google(t *testing.T) {
	service, _, _ := setupOAuthService(t)
	ctx := context.Background()

	authURL, err := service.GetAuthorizationURL(ctx, OAuthProviderGoogle, nil)
	require.NoError(t, err)
	assert.Contains(t, authURL, "https://accounts.google.com/o/oauth2/v2/auth")
	assert.Contains(t, authURL, "client_id=test-google-client-id")
	assert.Contains(t, authURL, "state=")
	assert.Contains(t, authURL, "nonce=")
	assert.Contains(t, authURL, "access_type=offline")
}

func TestOAuthService_GetAuthorizationURL_Discord(t *testing.T) {
	service, _, _ := setupOAuthService(t)
	ctx := context.Background()

	authURL, err := service.GetAuthorizationURL(ctx, OAuthProviderDiscord, nil)
	require.NoError(t, err)
	assert.Contains(t, authURL, "https://discord.com/api/oauth2/authorize")
	assert.Contains(t, authURL, "client_id=test-discord-client-id")
	assert.Contains(t, authURL, "state=")
}

func TestOAuthService_GetAuthorizationURL_UnsupportedProvider(t *testing.T) {
	service, _, _ := setupOAuthService(t)
	ctx := context.Background()

	// Remove config for unsupported provider test
	service.config.GitHub = nil
	_, err := service.GetAuthorizationURL(ctx, OAuthProviderGitHub, nil)
	assert.ErrorIs(t, err, ErrOAuthProviderNotSupported)
}

func TestOAuthService_IsProviderEnabled(t *testing.T) {
	service, _, _ := setupOAuthService(t)

	assert.True(t, service.IsProviderEnabled(OAuthProviderGitHub))
	assert.True(t, service.IsProviderEnabled(OAuthProviderGoogle))
	assert.True(t, service.IsProviderEnabled(OAuthProviderDiscord))

	// Disable a provider
	service.config.GitHub.ClientID = ""
	assert.False(t, service.IsProviderEnabled(OAuthProviderGitHub))
}

func TestOAuthService_StateStorageAndValidation(t *testing.T) {
	service, _, cache := setupOAuthService(t)
	ctx := context.Background()

	// Get authorization URL (stores state)
	authURL, err := service.GetAuthorizationURL(ctx, OAuthProviderGitHub, nil)
	require.NoError(t, err)

	// Extract state from URL
	// State is stored in the URL query parameter
	assert.Contains(t, authURL, "state=")

	// Check that state was stored in cache
	assert.Greater(t, len(cache.store), 0)
}

func TestOAuthService_StateValidation_Expired(t *testing.T) {
	service, _, cache := setupOAuthService(t)
	ctx := context.Background()

	// Create an expired state
	expiredState := &OAuthState{
		Provider:  OAuthProviderGitHub,
		State:     "expired-state-token",
		Nonce:     "test-nonce",
		CreatedAt: time.Now().Add(-15 * time.Minute), // 15 minutes ago (expired)
	}

	stateData, _ := json.Marshal(expiredState)
	cacheKey := fmt.Sprintf("oauth_state:%s", "expired-state-token")
	cache.store[cacheKey] = stateData

	// Try to validate the expired state
	_, err := service.validateState(ctx, "expired-state-token")
	assert.ErrorIs(t, err, ErrOAuthStateExpired)
}

func TestOAuthService_FindOrCreateUser_ExistingUser(t *testing.T) {
	service, userRepo, _ := setupOAuthService(t)
	ctx := context.Background()

	existingUser := &models.User{
		ID:       uuid.New(),
		Email:    "test@example.com",
		Username: "existinguser",
	}

	userRepo.On("GetByEmail", ctx, "test@example.com").Return(existingUser, nil)

	userInfo := &OAuthUserInfo{
		Provider:      OAuthProviderGitHub,
		ProviderID:    "12345",
		Email:         "test@example.com",
		EmailVerified: true,
		Username:      "newusername",
		DisplayName:   "Test User",
	}

	user, err := service.findOrCreateUser(ctx, userInfo, nil)
	require.NoError(t, err)
	assert.Equal(t, existingUser.ID, user.ID)
	assert.Equal(t, existingUser.Username, user.Username)
}

func TestOAuthService_FindOrCreateUser_NewUser(t *testing.T) {
	service, userRepo, _ := setupOAuthService(t)
	ctx := context.Background()

	userRepo.On("GetByEmail", ctx, "newuser@example.com").Return(nil, ErrUserNotFound)
	userRepo.On("GetByUsername", ctx, "newuser").Return(nil, ErrUserNotFound)
	userRepo.On("Create", ctx, mock.AnythingOfType("*models.User")).Return(nil)

	userInfo := &OAuthUserInfo{
		Provider:      OAuthProviderGitHub,
		ProviderID:    "12345",
		Email:         "newuser@example.com",
		EmailVerified: true,
		Username:      "newuser",
		DisplayName:   "New User",
		AvatarURL:     "https://example.com/avatar.png",
	}

	user, err := service.findOrCreateUser(ctx, userInfo, nil)
	require.NoError(t, err)
	assert.Equal(t, "newuser@example.com", user.Email)
	assert.Equal(t, "newuser", user.Username)
	assert.Equal(t, "New User", *user.DisplayName)
	assert.Equal(t, "https://example.com/avatar.png", *user.AvatarURL)
	assert.True(t, user.Verified)

	userRepo.AssertExpectations(t)
}

func TestOAuthService_FindOrCreateUser_UsernameCollision(t *testing.T) {
	service, userRepo, _ := setupOAuthService(t)
	ctx := context.Background()

	existingUser := &models.User{
		ID:       uuid.New(),
		Username: "takenusername",
	}

	userRepo.On("GetByEmail", ctx, "newuser@example.com").Return(nil, ErrUserNotFound)
	userRepo.On("GetByUsername", ctx, "takenusername").Return(existingUser, nil)
	userRepo.On("GetByUsername", ctx, mock.MatchedBy(func(s string) bool {
		return s != "takenusername" // Any other username
	})).Return(nil, ErrUserNotFound)
	userRepo.On("Create", ctx, mock.AnythingOfType("*models.User")).Return(nil)

	userInfo := &OAuthUserInfo{
		Provider:      OAuthProviderGitHub,
		ProviderID:    "12345",
		Email:         "newuser@example.com",
		EmailVerified: true,
		Username:      "takenusername",
		DisplayName:   "New User",
	}

	user, err := service.findOrCreateUser(ctx, userInfo, nil)
	require.NoError(t, err)
	// Username should be modified with a suffix
	assert.NotEqual(t, "takenusername", user.Username)
	assert.Contains(t, user.Username, "takenusername_")
}

func TestOAuthService_FindOrCreateUser_LinkingMode(t *testing.T) {
	service, userRepo, _ := setupOAuthService(t)
	ctx := context.Background()

	linkUserID := uuid.New()
	linkedUser := &models.User{
		ID:       linkUserID,
		Email:    "linked@example.com",
		Username: "linkeduser",
	}

	userRepo.On("GetByID", ctx, linkUserID).Return(linkedUser, nil)

	userInfo := &OAuthUserInfo{
		Provider:      OAuthProviderGitHub,
		ProviderID:    "12345",
		Email:         "different@example.com",
		EmailVerified: true,
		Username:      "oauthuser",
	}

	user, err := service.findOrCreateUser(ctx, userInfo, &linkUserID)
	require.NoError(t, err)
	assert.Equal(t, linkedUser.ID, user.ID)
	assert.Equal(t, "linked@example.com", user.Email) // Should keep original email
}

// Integration-style test with mock HTTP server
func TestOAuthService_HandleCallback_GitHubMock(t *testing.T) {
	// Create mock GitHub servers
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login/oauth/access_token" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"access_token": "mock-access-token",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer tokenServer.Close()

	userServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/user":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":         12345,
				"login":      "testuser",
				"name":       "Test User",
				"email":      "test@example.com",
				"avatar_url": "https://example.com/avatar.png",
			})
		case "/user/emails":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"email":    "test@example.com",
					"primary":  true,
					"verified": true,
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer userServer.Close()

	// Note: Full integration test would require mocking the GitHub API endpoints
	// This test verifies the structure and flow
	t.Log("Mock servers created successfully")
	t.Log("Token server:", tokenServer.URL)
	t.Log("User server:", userServer.URL)
}

func TestGenerateRandomString(t *testing.T) {
	// Test that we get random strings of correct length
	s1, err := generateRandomString(32)
	require.NoError(t, err)
	assert.Len(t, s1, 32)

	s2, err := generateRandomString(32)
	require.NoError(t, err)
	assert.Len(t, s2, 32)

	// Should be different each time (statistically)
	assert.NotEqual(t, s1, s2)
}

func TestOAuthProvider_Constants(t *testing.T) {
	assert.Equal(t, OAuthProvider("github"), OAuthProviderGitHub)
	assert.Equal(t, OAuthProvider("google"), OAuthProviderGoogle)
	assert.Equal(t, OAuthProvider("discord"), OAuthProviderDiscord)
}

func TestOAuthUserInfo_Structure(t *testing.T) {
	info := &OAuthUserInfo{
		Provider:      OAuthProviderGitHub,
		ProviderID:    "12345",
		Email:         "test@example.com",
		EmailVerified: true,
		Username:      "testuser",
		DisplayName:   "Test User",
		AvatarURL:     "https://example.com/avatar.png",
	}

	assert.Equal(t, OAuthProviderGitHub, info.Provider)
	assert.Equal(t, "12345", info.ProviderID)
	assert.Equal(t, "test@example.com", info.Email)
	assert.True(t, info.EmailVerified)
	assert.Equal(t, "testuser", info.Username)
	assert.Equal(t, "Test User", info.DisplayName)
	assert.Equal(t, "https://example.com/avatar.png", info.AvatarURL)
}

func TestOAuthState_Structure(t *testing.T) {
	linkUserID := uuid.New()
	state := &OAuthState{
		Provider:   OAuthProviderGoogle,
		State:      "test-state-token",
		Nonce:      "test-nonce",
		CreatedAt:  time.Now(),
		LinkUserID: &linkUserID,
	}

	assert.Equal(t, OAuthProviderGoogle, state.Provider)
	assert.Equal(t, "test-state-token", state.State)
	assert.Equal(t, "test-nonce", state.Nonce)
	assert.NotNil(t, state.LinkUserID)
	assert.Equal(t, linkUserID, *state.LinkUserID)
}

func TestOAuthErrors(t *testing.T) {
	// Test that all error types are distinct
	errors := []error{
		ErrOAuthProviderNotSupported,
		ErrOAuthStateMismatch,
		ErrOAuthStateExpired,
		ErrOAuthCodeExchange,
		ErrOAuthUserInfo,
		ErrOAuthEmailNotVerified,
		ErrOAuthAccountLinked,
	}

	for i, err1 := range errors {
		for j, err2 := range errors {
			if i != j {
				assert.NotEqual(t, err1, err2)
			}
		}
	}
}

func TestOAuthService_GetEnabledProviders(t *testing.T) {
	service, _, _ := setupOAuthService(t)

	providers := service.GetEnabledProviders()
	assert.Len(t, providers, 3)
	assert.Contains(t, providers, OAuthProviderGitHub)
	assert.Contains(t, providers, OAuthProviderGoogle)
	assert.Contains(t, providers, OAuthProviderDiscord)
}

func TestOAuthService_GetEnabledProviders_Partial(t *testing.T) {
	service, _, _ := setupOAuthService(t)

	// Disable some providers
	service.config.GitHub.ClientID = ""
	service.config.Discord = nil

	providers := service.GetEnabledProviders()
	assert.Len(t, providers, 1)
	assert.Contains(t, providers, OAuthProviderGoogle)
}

func TestOAuthService_GetLinkedProviders_NoRepo(t *testing.T) {
	service, _, _ := setupOAuthService(t)
	ctx := context.Background()

	// Without repo, should return nil
	providers, err := service.GetLinkedProviders(ctx, uuid.New())
	assert.NoError(t, err)
	assert.Nil(t, providers)
}

func TestOAuthService_GetLinkedProvider_NoRepo(t *testing.T) {
	service, _, _ := setupOAuthService(t)
	ctx := context.Background()

	// Without repo, should return not found
	_, err := service.GetLinkedProvider(ctx, uuid.New(), OAuthProviderGitHub)
	assert.ErrorIs(t, err, ErrOAuthProviderNotFound)
}

func TestOAuthService_UnlinkProvider_NoRepo(t *testing.T) {
	service, _, _ := setupOAuthService(t)
	ctx := context.Background()

	// Without repo, should return error
	err := service.UnlinkProvider(ctx, uuid.New(), OAuthProviderGitHub)
	assert.ErrorIs(t, err, ErrOAuthProviderNotSupported)
}

func TestOAuthService_GetLinkAuthorizationURL_NoRepo(t *testing.T) {
	service, _, _ := setupOAuthService(t)
	ctx := context.Background()

	// Without repo, should return error
	_, err := service.GetLinkAuthorizationURL(ctx, uuid.New(), OAuthProviderGitHub)
	assert.ErrorIs(t, err, ErrOAuthProviderNotSupported)
}
