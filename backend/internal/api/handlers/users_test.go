package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
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
	"hearth/internal/storage"
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

func (m *MockUserService) SendFriendRequest(ctx context.Context, senderID, receiverID uuid.UUID) error {
	args := m.Called(ctx, senderID, receiverID)
	return args.Error(0)
}

func (m *MockUserService) GetIncomingFriendRequests(ctx context.Context, userID uuid.UUID) ([]*models.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockUserService) GetOutgoingFriendRequests(ctx context.Context, userID uuid.UUID) ([]*models.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockUserService) AcceptFriendRequest(ctx context.Context, receiverID, senderID uuid.UUID) error {
	args := m.Called(ctx, receiverID, senderID)
	return args.Error(0)
}

func (m *MockUserService) DeclineFriendRequest(ctx context.Context, userID, otherID uuid.UUID) error {
	args := m.Called(ctx, userID, otherID)
	return args.Error(0)
}

func (m *MockUserService) GetRelationship(ctx context.Context, userID, targetID uuid.UUID) (int, error) {
	args := m.Called(ctx, userID, targetID)
	return args.Int(0), args.Error(1)
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

func (m *MockChannelServiceForUsers) GetOrCreateDM(ctx context.Context, user1ID, user2ID uuid.UUID) (*models.Channel, error) {
	args := m.Called(ctx, user1ID, user2ID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Channel), args.Error(1)
}

func (m *MockChannelServiceForUsers) CreateGroupDM(ctx context.Context, ownerID uuid.UUID, name string, recipientIDs []uuid.UUID) (*models.Channel, error) {
	args := m.Called(ctx, ownerID, name, recipientIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Channel), args.Error(1)
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
	app.Post("/users/@me/channels", handler.CreateDM)
	app.Post("/users/@me/channels/group", handler.CreateGroupDM)
	app.Get("/users/@me/relationships", handler.GetRelationships)
	app.Post("/users/@me/relationships", handler.CreateRelationship)
	app.Delete("/users/@me/relationships/:id", handler.DeleteRelationship)
	app.Get("/users/@me/friends", handler.GetFriends)
	app.Get("/users/@me/friends/pending", handler.GetPendingFriendRequests)
	app.Put("/users/@me/friends/:id", handler.AcceptFriendRequest)
	app.Delete("/users/@me/friends/:id/request", handler.DeclineFriendRequest)
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

func TestUserHandler_CreateDM(t *testing.T) {
	th := newTestUserHandler()

	recipientID := uuid.New()
	channelID := uuid.New()
	channel := &models.Channel{
		ID:         channelID,
		Type:       models.ChannelTypeDM,
		Recipients: []uuid.UUID{th.userID, recipientID},
		CreatedAt:  time.Now(),
	}

	th.channelService.On("GetOrCreateDM", mock.Anything, th.userID, recipientID).Return(channel, nil)

	body := map[string]interface{}{
		"recipient_id": recipientID.String(),
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users/@me/channels", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result DMChannelResponse
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, channelID, result.ID)
	assert.Equal(t, models.ChannelTypeDM, result.Type)

	th.channelService.AssertExpectations(t)
}

func TestUserHandler_CreateDM_MissingRecipient(t *testing.T) {
	th := newTestUserHandler()

	body := map[string]interface{}{}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users/@me/channels", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "recipient_id is required", result["error"])
}

func TestUserHandler_CreateDM_InvalidRecipient(t *testing.T) {
	th := newTestUserHandler()

	body := map[string]interface{}{
		"recipient_id": "not-a-uuid",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users/@me/channels", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "invalid recipient_id", result["error"])
}

func TestUserHandler_CreateDM_Self(t *testing.T) {
	th := newTestUserHandler()

	body := map[string]interface{}{
		"recipient_id": th.userID.String(),
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users/@me/channels", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "cannot create DM with yourself", result["error"])
}

func TestUserHandler_CreateDM_ServiceError(t *testing.T) {
	th := newTestUserHandler()

	recipientID := uuid.New()

	th.channelService.On("GetOrCreateDM", mock.Anything, th.userID, recipientID).
		Return(nil, services.ErrUserNotFound)

	body := map[string]interface{}{
		"recipient_id": recipientID.String(),
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users/@me/channels", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	th.channelService.AssertExpectations(t)
}

func TestUserHandler_CreateGroupDM(t *testing.T) {
	th := newTestUserHandler()

	recipient1 := uuid.New()
	recipient2 := uuid.New()
	channelID := uuid.New()
	name := "Test Group"

	channel := &models.Channel{
		ID:         channelID,
		Name:       name,
		Type:       models.ChannelTypeGroupDM,
		OwnerID:    &th.userID,
		Recipients: []uuid.UUID{th.userID, recipient1, recipient2},
		CreatedAt:  time.Now(),
	}

	th.channelService.On("CreateGroupDM", mock.Anything, th.userID, name, mock.MatchedBy(func(ids []uuid.UUID) bool {
		return len(ids) == 2
	})).Return(channel, nil)

	body := map[string]interface{}{
		"recipient_ids": []string{recipient1.String(), recipient2.String()},
		"name":          name,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users/@me/channels/group", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result DMChannelResponse
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, channelID, result.ID)
	assert.Equal(t, models.ChannelTypeGroupDM, result.Type)

	th.channelService.AssertExpectations(t)
}

func TestUserHandler_CreateGroupDM_NoRecipients(t *testing.T) {
	th := newTestUserHandler()

	body := map[string]interface{}{
		"recipient_ids": []string{},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users/@me/channels/group", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "at least one recipient is required", result["error"])
}

func TestUserHandler_CreateGroupDM_TooManyRecipients(t *testing.T) {
	th := newTestUserHandler()

	// Create 10 recipients (max is 9 + owner = 10)
	recipients := make([]string, 10)
	for i := range recipients {
		recipients[i] = uuid.New().String()
	}

	body := map[string]interface{}{
		"recipient_ids": recipients,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users/@me/channels/group", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "group DM can have at most 10 members", result["error"])
}

func TestUserHandler_CreateGroupDM_InvalidRecipient(t *testing.T) {
	th := newTestUserHandler()

	body := map[string]interface{}{
		"recipient_ids": []string{"invalid-uuid"},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users/@me/channels/group", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Contains(t, result["error"], "invalid recipient_id")
}

func TestUserHandler_CreateGroupDM_OnlySelf(t *testing.T) {
	th := newTestUserHandler()

	body := map[string]interface{}{
		"recipient_ids": []string{th.userID.String()},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users/@me/channels/group", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "at least one other recipient is required", result["error"])
}

func TestUserHandler_CreateGroupDM_ServiceError(t *testing.T) {
	th := newTestUserHandler()

	recipientID := uuid.New()

	th.channelService.On("CreateGroupDM", mock.Anything, th.userID, "", mock.Anything).
		Return(nil, services.ErrUserNotFound)

	body := map[string]interface{}{
		"recipient_ids": []string{recipientID.String()},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users/@me/channels/group", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	th.channelService.AssertExpectations(t)
}

func TestUserHandler_CreateGroupDM_NoName(t *testing.T) {
	th := newTestUserHandler()

	recipient1 := uuid.New()
	channelID := uuid.New()

	channel := &models.Channel{
		ID:         channelID,
		Name:       "", // No name
		Type:       models.ChannelTypeGroupDM,
		OwnerID:    &th.userID,
		Recipients: []uuid.UUID{th.userID, recipient1},
		CreatedAt:  time.Now(),
	}

	th.channelService.On("CreateGroupDM", mock.Anything, th.userID, "", mock.MatchedBy(func(ids []uuid.UUID) bool {
		return len(ids) == 1 && ids[0] == recipient1
	})).Return(channel, nil)

	body := map[string]interface{}{
		"recipient_ids": []string{recipient1.String()},
		// No name provided
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users/@me/channels/group", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result DMChannelResponse
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, channelID, result.ID)
	assert.Equal(t, models.ChannelTypeGroupDM, result.Type)

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

// MockStorageService mocks the storage service for testing
type MockStorageService struct {
	mock.Mock
}

func (m *MockStorageService) UploadFile(ctx context.Context, file *multipart.FileHeader, uploaderID uuid.UUID, category string) (*storage.FileInfo, error) {
	args := m.Called(ctx, file, uploaderID, category)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.FileInfo), args.Error(1)
}

func (m *MockStorageService) DeleteFile(ctx context.Context, path string) error {
	args := m.Called(ctx, path)
	return args.Error(0)
}

// testUserHandlerWithStorage creates a test user handler with storage support
type testUserHandlerWithStorage struct {
	*testUserHandler
	storageService *MockStorageService
}

func newTestUserHandlerWithStorage() *testUserHandlerWithStorage {
	th := newTestUserHandler()
	storageService := new(MockStorageService)
	th.handler.storageService = storageService

	// Add avatar routes
	th.app.Patch("/users/@me/avatar", th.handler.UpdateAvatar)
	th.app.Delete("/users/@me/avatar", th.handler.DeleteAvatar)

	return &testUserHandlerWithStorage{
		testUserHandler: th,
		storageService:  storageService,
	}
}

func TestUserHandler_UpdateAvatar_NoStorageService(t *testing.T) {
	// Create handler without storage service
	userService := new(MockUserService)
	serverService := new(MockServerServiceForUsers)
	channelService := new(MockChannelServiceForUsers)

	handler := &UserHandler{
		userService:    userService,
		serverService:  serverService,
		channelService: channelService,
		storageService: nil, // No storage service
	}

	app := fiber.New()
	userID := uuid.New()

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", userID)
		return c.Next()
	})

	app.Patch("/users/@me/avatar", handler.UpdateAvatar)

	req := httptest.NewRequest(http.MethodPatch, "/users/@me/avatar", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotImplemented, resp.StatusCode)
}

func TestUserHandler_UpdateAvatar_NoFile(t *testing.T) {
	th := newTestUserHandlerWithStorage()

	req := httptest.NewRequest(http.MethodPatch, "/users/@me/avatar", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "avatar file required", result["error"])
}

func TestUserHandler_DeleteAvatar_Success(t *testing.T) {
	th := newTestUserHandlerWithStorage()

	updatedUser := &models.User{
		ID:            th.userID,
		Username:      "testuser",
		Discriminator: "0001",
		Email:         "test@example.com",
		AvatarURL:     nil, // Avatar removed
		CreatedAt:     time.Now(),
	}

	th.userService.On("UpdateUser", mock.Anything, th.userID, mock.MatchedBy(func(u *models.UserUpdate) bool {
		return u.AvatarURL == nil || *u.AvatarURL == ""
	})).Return(updatedUser, nil)

	req := httptest.NewRequest(http.MethodDelete, "/users/@me/avatar", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result UserResponse
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, th.userID, result.ID)
	assert.Nil(t, result.AvatarURL)

	th.userService.AssertExpectations(t)
}

func TestUserHandler_DeleteAvatar_UpdateError(t *testing.T) {
	th := newTestUserHandlerWithStorage()

	th.userService.On("UpdateUser", mock.Anything, th.userID, mock.Anything).
		Return(nil, services.ErrUserNotFound)

	req := httptest.NewRequest(http.MethodDelete, "/users/@me/avatar", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	th.userService.AssertExpectations(t)
}
