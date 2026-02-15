package handlers

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"hearth/internal/models"
	"hearth/internal/services"
)

// MockMutualServersService implements the MutualServersService interface
type MockMutualServersService struct {
	mock.Mock
}

func (m *MockMutualServersService) GetMutualServersLimited(ctx context.Context, userID1, userID2 uuid.UUID, limit int) ([]*models.Server, int, error) {
	args := m.Called(ctx, userID1, userID2, limit)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*models.Server), args.Int(1), args.Error(2)
}

// Also satisfy the ServerServiceForUsersInterface
func (m *MockMutualServersService) GetUserServers(ctx context.Context, userID uuid.UUID) ([]*models.Server, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Server), args.Error(1)
}

// MockMutualFriendsService implements the MutualFriendsService interface
type MockMutualFriendsService struct {
	mock.Mock
}

func (m *MockMutualFriendsService) GetUser(ctx context.Context, id uuid.UUID) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockMutualFriendsService) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockMutualFriendsService) UpdateUser(ctx context.Context, id uuid.UUID, updates *models.UserUpdate) (*models.User, error) {
	args := m.Called(ctx, id, updates)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockMutualFriendsService) GetFriends(ctx context.Context, userID uuid.UUID) ([]*models.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockMutualFriendsService) AddFriend(ctx context.Context, userID, friendID uuid.UUID) error {
	args := m.Called(ctx, userID, friendID)
	return args.Error(0)
}

func (m *MockMutualFriendsService) RemoveFriend(ctx context.Context, userID, friendID uuid.UUID) error {
	args := m.Called(ctx, userID, friendID)
	return args.Error(0)
}

func (m *MockMutualFriendsService) BlockUser(ctx context.Context, userID, blockedID uuid.UUID) error {
	args := m.Called(ctx, userID, blockedID)
	return args.Error(0)
}

func (m *MockMutualFriendsService) UnblockUser(ctx context.Context, userID, blockedID uuid.UUID) error {
	args := m.Called(ctx, userID, blockedID)
	return args.Error(0)
}

func (m *MockMutualFriendsService) SendFriendRequest(ctx context.Context, senderID, receiverID uuid.UUID) error {
	args := m.Called(ctx, senderID, receiverID)
	return args.Error(0)
}

func (m *MockMutualFriendsService) GetIncomingFriendRequests(ctx context.Context, userID uuid.UUID) ([]*models.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockMutualFriendsService) GetOutgoingFriendRequests(ctx context.Context, userID uuid.UUID) ([]*models.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockMutualFriendsService) AcceptFriendRequest(ctx context.Context, receiverID, senderID uuid.UUID) error {
	args := m.Called(ctx, receiverID, senderID)
	return args.Error(0)
}

func (m *MockMutualFriendsService) DeclineFriendRequest(ctx context.Context, userID, otherID uuid.UUID) error {
	args := m.Called(ctx, userID, otherID)
	return args.Error(0)
}

func (m *MockMutualFriendsService) GetRelationship(ctx context.Context, userID, targetID uuid.UUID) (int, error) {
	args := m.Called(ctx, userID, targetID)
	return args.Int(0), args.Error(1)
}

func (m *MockMutualFriendsService) GetMutualFriends(ctx context.Context, userID1, userID2 uuid.UUID, limit int) ([]*models.User, int, error) {
	args := m.Called(ctx, userID1, userID2, limit)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*models.User), args.Int(1), args.Error(2)
}

func (m *MockMutualFriendsService) GetRecentActivity(ctx context.Context, requesterID, targetID uuid.UUID) (*services.RecentActivityInfo, error) {
	args := m.Called(ctx, requesterID, targetID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.RecentActivityInfo), args.Error(1)
}

func TestGetUserProfile_Success(t *testing.T) {
	// Setup
	app := fiber.New()
	
	requesterID := uuid.New()
	targetID := uuid.New()
	
	targetUser := &models.User{
		ID:            targetID,
		Username:      "testuser",
		Discriminator: "1234",
		Email:         "test@example.com",
		CreatedAt:     time.Now(),
	}
	
	mockUserService := new(MockMutualFriendsService)
	mockServerService := new(MockMutualServersService)
	
	// Setup expectations
	mockUserService.On("GetUser", mock.Anything, targetID).Return(targetUser, nil)
	
	mutualServers := []*models.Server{
		{ID: uuid.New(), Name: "Server 1"},
		{ID: uuid.New(), Name: "Server 2"},
	}
	mockServerService.On("GetMutualServersLimited", mock.Anything, requesterID, targetID, 10).Return(mutualServers, 2, nil)
	
	mutualFriends := []*models.User{
		{ID: uuid.New(), Username: "friend1"},
	}
	mockUserService.On("GetMutualFriends", mock.Anything, requesterID, targetID, 10).Return(mutualFriends, 1, nil)
	
	lastMessage := time.Now().Add(-1 * time.Hour)
	recentActivity := &services.RecentActivityInfo{
		LastMessageAt:   &lastMessage,
		ServerName:      stringPtr("Server 1"),
		ChannelName:     stringPtr("general"),
		MessageCount24h: 5,
	}
	mockUserService.On("GetRecentActivity", mock.Anything, requesterID, targetID).Return(recentActivity, nil)
	
	handler := &UserHandler{
		userService:   mockUserService,
		serverService: mockServerService,
	}
	
	app.Get("/users/:id/profile", func(c *fiber.Ctx) error {
		c.Locals("userID", requesterID)
		return handler.GetUserProfile(c)
	})
	
	// Execute
	req := httptest.NewRequest("GET", "/users/"+targetID.String()+"/profile", nil)
	resp, err := app.Test(req)
	
	// Assert
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	
	var response UserProfileResponse
	json.NewDecoder(resp.Body).Decode(&response)
	
	assert.Equal(t, targetID, response.User.ID)
	assert.Equal(t, "testuser", response.User.Username)
	assert.Len(t, response.MutualServers, 2)
	assert.Len(t, response.MutualFriends, 1)
	assert.Equal(t, 2, response.TotalMutual.Servers)
	assert.Equal(t, 1, response.TotalMutual.Friends)
	assert.NotNil(t, response.RecentActivity)
	assert.Equal(t, 5, response.RecentActivity.MessageCount24h)
}

func TestGetUserProfile_OwnProfile(t *testing.T) {
	// Setup
	app := fiber.New()
	
	userID := uuid.New()
	
	user := &models.User{
		ID:            userID,
		Username:      "myself",
		Discriminator: "1234",
		Email:         "me@example.com",
		CreatedAt:     time.Now(),
	}
	
	mockUserService := new(MockMutualFriendsService)
	mockUserService.On("GetUser", mock.Anything, userID).Return(user, nil)
	
	handler := &UserHandler{
		userService: mockUserService,
	}
	
	app.Get("/users/:id/profile", func(c *fiber.Ctx) error {
		c.Locals("userID", userID)
		return handler.GetUserProfile(c)
	})
	
	// Execute
	req := httptest.NewRequest("GET", "/users/"+userID.String()+"/profile", nil)
	resp, err := app.Test(req)
	
	// Assert
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	
	var response UserProfileResponse
	json.NewDecoder(resp.Body).Decode(&response)
	
	// For own profile, mutual data should be empty
	assert.Equal(t, userID, response.User.ID)
	assert.Empty(t, response.MutualServers)
	assert.Empty(t, response.MutualFriends)
	assert.Empty(t, response.SharedChannels)
}

func TestGetUserProfile_UserNotFound(t *testing.T) {
	// Setup
	app := fiber.New()
	
	requesterID := uuid.New()
	targetID := uuid.New()
	
	mockUserService := new(MockMutualFriendsService)
	mockUserService.On("GetUser", mock.Anything, targetID).Return(nil, services.ErrUserNotFound)
	
	handler := &UserHandler{
		userService: mockUserService,
	}
	
	app.Get("/users/:id/profile", func(c *fiber.Ctx) error {
		c.Locals("userID", requesterID)
		return handler.GetUserProfile(c)
	})
	
	// Execute
	req := httptest.NewRequest("GET", "/users/"+targetID.String()+"/profile", nil)
	resp, err := app.Test(req)
	
	// Assert
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestGetUserProfile_InvalidUUID(t *testing.T) {
	// Setup
	app := fiber.New()
	
	requesterID := uuid.New()
	
	handler := &UserHandler{}
	
	app.Get("/users/:id/profile", func(c *fiber.Ctx) error {
		c.Locals("userID", requesterID)
		return handler.GetUserProfile(c)
	})
	
	// Execute
	req := httptest.NewRequest("GET", "/users/invalid-uuid/profile", nil)
	resp, err := app.Test(req)
	
	// Assert
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func stringPtr(s string) *string {
	return &s
}
