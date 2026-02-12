package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
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

// MockUserService mocks the UserService for testing
type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) GetUser(ctx context.Context, id uuid.UUID) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserService) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserService) UpdateUser(ctx context.Context, id uuid.UUID, updates *models.UserUpdate) (*models.User, error) {
	args := m.Called(ctx, id, updates)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserService) GetFriends(ctx context.Context, userID uuid.UUID) ([]*models.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockUserService) AddFriend(ctx context.Context, userID, friendID uuid.UUID) error {
	args := m.Called(ctx, userID, friendID)
	return args.Error(0)
}

func (m *MockUserService) RemoveFriend(ctx context.Context, userID, friendID uuid.UUID) error {
	args := m.Called(ctx, userID, friendID)
	return args.Error(0)
}

func (m *MockUserService) BlockUser(ctx context.Context, userID, blockedID uuid.UUID) error {
	args := m.Called(ctx, userID, blockedID)
	return args.Error(0)
}

func (m *MockUserService) UnblockUser(ctx context.Context, userID, blockedID uuid.UUID) error {
	args := m.Called(ctx, userID, blockedID)
	return args.Error(0)
}

// MockServerServiceForUsers mocks the ServerService for user handler testing
type MockServerServiceForUsers struct {
	mock.Mock
}

func (m *MockServerServiceForUsers) GetUserServers(ctx context.Context, userID uuid.UUID) ([]*models.Server, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Server), args.Error(1)
}

// MockChannelServiceForUsers mocks the ChannelService for user handler testing
type MockChannelServiceForUsers struct {
	mock.Mock
}

func (m *MockChannelServiceForUsers) GetUserDMs(ctx context.Context, userID uuid.UUID) ([]*models.Channel, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Channel), args.Error(1)
}

// testUserHandler creates a test user handler with mocks
type testUserHandler struct {
	handler        *UserHandler
	userService    *MockUserService
	serverService  *MockServerServiceForUsers
	channelService *MockChannelServiceForUsers
	app            *fiber.App
	userID         uuid.UUID
}

func newTestUserHandler() *testUserHandler {
	userService := new(MockUserService)
	serverService := new(MockServerServiceForUsers)
	channelService := new(MockChannelServiceForUsers)

	handler := &UserHandler{
		userService:    userService,
		serverService:  serverService,
		channelService: channelService,
	}

	app := fiber.New()
	userID := uuid.New()

	// Add middleware to set userID in locals
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", userID)
		return c.Next()
	})

	// Setup routes
	app.Get("/users/@me", handler.GetMe)
	app.Patch("/users/@me", handler.UpdateMe)
	app.Get("/users/@me/servers", handler.GetMyServers)
	app.Get("/users/@me/channels", handler.GetMyDMs)
	app.Get("/users/@me/relationships", handler.GetRelationships)
	app.Post("/users/@me/relationships", handler.CreateRelationship)
	app.Delete("/users/@me/relationships/:id", handler.DeleteRelationship)
	app.Get("/users/:id", handler.GetUser)

	return &testUserHandler{
		handler:        handler,
		userService:    userService,
		serverService:  serverService,
		channelService: channelService,
		app:            app,
		userID:         userID,
	}
}

func TestUserHandler_GetMe(t *testing.T) {
	th := newTestUserHandler()

	user := &models.User{
		ID:            th.userID,
		Username:      "testuser",
		Discriminator: "0001",
		Email:         "test@example.com",
		CreatedAt:     time.Now(),
	}

	th.userService.On("GetUser", mock.Anything, th.userID).Return(user, nil)

	req := httptest.NewRequest(http.MethodGet, "/users/@me", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result UserResponse
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, user.ID, result.ID)
	assert.Equal(t, user.Username, result.Username)
	assert.Equal(t, user.Discriminator, result.Discriminator)
	assert.NotNil(t, result.Email)
	assert.Equal(t, user.Email, *result.Email)

	th.userService.AssertExpectations(t)
}

func TestUserHandler_GetMe_NotFound(t *testing.T) {
	th := newTestUserHandler()

	th.userService.On("GetUser", mock.Anything, th.userID).Return(nil, services.ErrUserNotFound)

	req := httptest.NewRequest(http.MethodGet, "/users/@me", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	th.userService.AssertExpectations(t)
}

func TestUserHandler_UpdateMe(t *testing.T) {
	th := newTestUserHandler()

	newUsername := "newusername"
	newBio := "Test bio"

	updatedUser := &models.User{
		ID:            th.userID,
		Username:      newUsername,
		Discriminator: "0001",
		Email:         "test@example.com",
		Bio:           &newBio,
		CreatedAt:     time.Now(),
	}

	th.userService.On("UpdateUser", mock.Anything, th.userID, mock.MatchedBy(func(u *models.UserUpdate) bool {
		return u.Username != nil && *u.Username == newUsername
	})).Return(updatedUser, nil)

	body := map[string]interface{}{
		"username": newUsername,
		"bio":      newBio,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/users/@me", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result UserResponse
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, newUsername, result.Username)
	assert.NotNil(t, result.Bio)
	assert.Equal(t, newBio, *result.Bio)

	th.userService.AssertExpectations(t)
}

func TestUserHandler_UpdateMe_UsernameTooShort(t *testing.T) {
	th := newTestUserHandler()

	body := map[string]interface{}{
		"username": "a", // Too short
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/users/@me", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestUserHandler_UpdateMe_UsernameTaken(t *testing.T) {
	th := newTestUserHandler()

	th.userService.On("UpdateUser", mock.Anything, th.userID, mock.Anything).Return(nil, services.ErrUsernameTaken)

	body := map[string]interface{}{
		"username": "takenuser",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/users/@me", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)

	th.userService.AssertExpectations(t)
}

func TestUserHandler_GetMyServers(t *testing.T) {
	th := newTestUserHandler()

	serverID := uuid.New()
	servers := []*models.Server{
		{
			ID:        serverID,
			Name:      "Test Server",
			OwnerID:   th.userID,
			Features:  []string{"COMMUNITY"},
			CreatedAt: time.Now(),
		},
	}

	th.serverService.On("GetUserServers", mock.Anything, th.userID).Return(servers, nil)

	req := httptest.NewRequest(http.MethodGet, "/users/@me/servers", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result []ServerResponse
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Len(t, result, 1)
	assert.Equal(t, serverID, result[0].ID)
	assert.Equal(t, "Test Server", result[0].Name)

	th.serverService.AssertExpectations(t)
}

func TestUserHandler_GetMyDMs(t *testing.T) {
	th := newTestUserHandler()

	channelID := uuid.New()
	channels := []*models.Channel{
		{
			ID:        channelID,
			Type:      models.ChannelTypeDM,
			CreatedAt: time.Now(),
		},
	}

	th.channelService.On("GetUserDMs", mock.Anything, th.userID).Return(channels, nil)

	req := httptest.NewRequest(http.MethodGet, "/users/@me/channels", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result []DMChannelResponse
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Len(t, result, 1)
	assert.Equal(t, channelID, result[0].ID)
	assert.Equal(t, models.ChannelTypeDM, result[0].Type)

	th.channelService.AssertExpectations(t)
}

func TestUserHandler_GetUser(t *testing.T) {
	th := newTestUserHandler()

	targetID := uuid.New()
	user := &models.User{
		ID:            targetID,
		Username:      "otheruser",
		Discriminator: "0002",
		Email:         "other@example.com", // Should not be in response
		CreatedAt:     time.Now(),
	}

	th.userService.On("GetUser", mock.Anything, targetID).Return(user, nil)

	req := httptest.NewRequest(http.MethodGet, "/users/"+targetID.String(), nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result UserResponse
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, user.ID, result.ID)
	assert.Equal(t, user.Username, result.Username)
	assert.Nil(t, result.Email) // Email should not be exposed for other users

	th.userService.AssertExpectations(t)
}

func TestUserHandler_GetRelationships(t *testing.T) {
	th := newTestUserHandler()

	friendID := uuid.New()
	friends := []*models.User{
		{
			ID:            friendID,
			Username:      "friend",
			Discriminator: "0003",
			CreatedAt:     time.Now(),
		},
	}

	th.userService.On("GetFriends", mock.Anything, th.userID).Return(friends, nil)

	req := httptest.NewRequest(http.MethodGet, "/users/@me/relationships", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result []RelationshipResponse
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Len(t, result, 1)
	assert.Equal(t, friendID, result[0].ID)
	assert.Equal(t, RelationshipTypeFriend, result[0].Type)

	th.userService.AssertExpectations(t)
}

func TestUserHandler_CreateRelationship_AddFriend(t *testing.T) {
	th := newTestUserHandler()

	friendID := uuid.New()
	th.userService.On("AddFriend", mock.Anything, th.userID, friendID).Return(nil)

	body := map[string]interface{}{
		"user_id": friendID.String(),
		"type":    1, // Friend
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users/@me/relationships", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	th.userService.AssertExpectations(t)
}

func TestUserHandler_CreateRelationship_BlockUser(t *testing.T) {
	th := newTestUserHandler()

	blockedID := uuid.New()
	th.userService.On("BlockUser", mock.Anything, th.userID, blockedID).Return(nil)

	body := map[string]interface{}{
		"user_id": blockedID.String(),
		"type":    2, // Block
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users/@me/relationships", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	th.userService.AssertExpectations(t)
}

func TestUserHandler_CreateRelationship_Self(t *testing.T) {
	th := newTestUserHandler()

	body := map[string]interface{}{
		"user_id": th.userID.String(), // Self
		"type":    1,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users/@me/relationships", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestUserHandler_DeleteRelationship(t *testing.T) {
	th := newTestUserHandler()

	friendID := uuid.New()
	th.userService.On("RemoveFriend", mock.Anything, th.userID, friendID).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/users/@me/relationships/"+friendID.String(), nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	th.userService.AssertExpectations(t)
}
