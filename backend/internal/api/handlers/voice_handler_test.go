package handlers

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"hearth/internal/models"
	"hearth/internal/services"
)

// ─── GenerateToken Tests ─────────────────────────────────────────────────────

func TestGenerateToken_Success(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	voiceSvc := new(MockVoiceService)
	userSvc := new(MockUserServiceForVoice)
	channelSvc := new(MockChannelServiceForVoice)

	displayName := "Test User"
	userSvc.On("GetUser", mock.Anything, userID).Return(&models.User{
		ID:          userID,
		Username:    "testuser",
		DisplayName: &displayName,
		AvatarURL:   nil,
	}, nil)

	voiceSvc.On("GenerateToken", mock.Anything, userID, channelID, "testuser", "Test User", "").
		Return(&services.VoiceTokenResponse{
			Token: "lk-token-abc123",
			URL:   "wss://livekit.example.com",
		}, nil)

	app := fiber.New()
	setupUserLocals(app, userID)
	handler := NewLiveKitVoiceHandler(voiceSvc, userSvc, channelSvc, nil)
	app.Post("/voice/token", handler.GenerateToken)

	body := `{"channel_id":"` + channelID.String() + `"}`
	req := httptest.NewRequest("POST", "/voice/token", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result GenerateTokenResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "lk-token-abc123", result.Token)
	assert.Equal(t, "wss://livekit.example.com", result.URL)

	voiceSvc.AssertExpectations(t)
	userSvc.AssertExpectations(t)
}

func TestGenerateToken_InvalidBody(t *testing.T) {
	userID := uuid.New()

	voiceSvc := new(MockVoiceService)
	userSvc := new(MockUserServiceForVoice)
	channelSvc := new(MockChannelServiceForVoice)

	app := fiber.New()
	setupUserLocals(app, userID)
	handler := NewLiveKitVoiceHandler(voiceSvc, userSvc, channelSvc, nil)
	app.Post("/voice/token", handler.GenerateToken)

	req := httptest.NewRequest("POST", "/voice/token", bytes.NewReader([]byte(`{invalid json}`)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestGenerateToken_MissingChannelID(t *testing.T) {
	userID := uuid.New()

	voiceSvc := new(MockVoiceService)
	userSvc := new(MockUserServiceForVoice)
	channelSvc := new(MockChannelServiceForVoice)

	app := fiber.New()
	setupUserLocals(app, userID)
	handler := NewLiveKitVoiceHandler(voiceSvc, userSvc, channelSvc, nil)
	app.Post("/voice/token", handler.GenerateToken)

	body := `{"channel_id":""}`
	req := httptest.NewRequest("POST", "/voice/token", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestGenerateToken_InvalidChannelIDFormat(t *testing.T) {
	userID := uuid.New()

	voiceSvc := new(MockVoiceService)
	userSvc := new(MockUserServiceForVoice)
	channelSvc := new(MockChannelServiceForVoice)

	app := fiber.New()
	setupUserLocals(app, userID)
	handler := NewLiveKitVoiceHandler(voiceSvc, userSvc, channelSvc, nil)
	app.Post("/voice/token", handler.GenerateToken)

	body := `{"channel_id":"not-a-uuid"}`
	req := httptest.NewRequest("POST", "/voice/token", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestGenerateToken_UserServiceError(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	voiceSvc := new(MockVoiceService)
	userSvc := new(MockUserServiceForVoice)
	channelSvc := new(MockChannelServiceForVoice)

	userSvc.On("GetUser", mock.Anything, userID).Return(nil, assert.AnError)

	app := fiber.New()
	setupUserLocals(app, userID)
	handler := NewLiveKitVoiceHandler(voiceSvc, userSvc, channelSvc, nil)
	app.Post("/voice/token", handler.GenerateToken)

	body := `{"channel_id":"` + channelID.String() + `"}`
	req := httptest.NewRequest("POST", "/voice/token", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

func TestGenerateToken_ServiceNotConfigured(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	voiceSvc := new(MockVoiceService)
	userSvc := new(MockUserServiceForVoice)
	channelSvc := new(MockChannelServiceForVoice)

	displayName := "Test User"
	userSvc.On("GetUser", mock.Anything, userID).Return(&models.User{
		ID:          userID,
		Username:    "testuser",
		DisplayName: &displayName,
	}, nil)

	voiceSvc.On("GenerateToken", mock.Anything, userID, channelID, "testuser", "Test User", "").
		Return(nil, services.ErrLiveKitNotConfigured)

	app := fiber.New()
	setupUserLocals(app, userID)
	handler := NewLiveKitVoiceHandler(voiceSvc, userSvc, channelSvc, nil)
	app.Post("/voice/token", handler.GenerateToken)

	body := `{"channel_id":"` + channelID.String() + `"}`
	req := httptest.NewRequest("POST", "/voice/token", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)
}

func TestGenerateToken_ChannelNotFound(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	voiceSvc := new(MockVoiceService)
	userSvc := new(MockUserServiceForVoice)
	channelSvc := new(MockChannelServiceForVoice)

	displayName := "Test User"
	userSvc.On("GetUser", mock.Anything, userID).Return(&models.User{
		ID:          userID,
		Username:    "testuser",
		DisplayName: &displayName,
	}, nil)

	voiceSvc.On("GenerateToken", mock.Anything, userID, channelID, "testuser", "Test User", "").
		Return(nil, services.ErrVoiceChannelNotFound)

	app := fiber.New()
	setupUserLocals(app, userID)
	handler := NewLiveKitVoiceHandler(voiceSvc, userSvc, channelSvc, nil)
	app.Post("/voice/token", handler.GenerateToken)

	body := `{"channel_id":"` + channelID.String() + `"}`
	req := httptest.NewRequest("POST", "/voice/token", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestGenerateToken_NotAVoiceChannel(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	voiceSvc := new(MockVoiceService)
	userSvc := new(MockUserServiceForVoice)
	channelSvc := new(MockChannelServiceForVoice)

	displayName := "Test User"
	userSvc.On("GetUser", mock.Anything, userID).Return(&models.User{
		ID:          userID,
		Username:    "testuser",
		DisplayName: &displayName,
	}, nil)

	voiceSvc.On("GenerateToken", mock.Anything, userID, channelID, "testuser", "Test User", "").
		Return(nil, services.ErrNotVoiceChannel)

	app := fiber.New()
	setupUserLocals(app, userID)
	handler := NewLiveKitVoiceHandler(voiceSvc, userSvc, channelSvc, nil)
	app.Post("/voice/token", handler.GenerateToken)

	body := `{"channel_id":"` + channelID.String() + `"}`
	req := httptest.NewRequest("POST", "/voice/token", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestGenerateToken_NotServerMember(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	voiceSvc := new(MockVoiceService)
	userSvc := new(MockUserServiceForVoice)
	channelSvc := new(MockChannelServiceForVoice)

	displayName := "Test User"
	userSvc.On("GetUser", mock.Anything, userID).Return(&models.User{
		ID:          userID,
		Username:    "testuser",
		DisplayName: &displayName,
	}, nil)

	voiceSvc.On("GenerateToken", mock.Anything, userID, channelID, "testuser", "Test User", "").
		Return(nil, services.ErrNotServerMember)

	app := fiber.New()
	setupUserLocals(app, userID)
	handler := NewLiveKitVoiceHandler(voiceSvc, userSvc, channelSvc, nil)
	app.Post("/voice/token", handler.GenerateToken)

	body := `{"channel_id":"` + channelID.String() + `"}`
	req := httptest.NewRequest("POST", "/voice/token", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

func TestGenerateToken_GenericError(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	voiceSvc := new(MockVoiceService)
	userSvc := new(MockUserServiceForVoice)
	channelSvc := new(MockChannelServiceForVoice)

	displayName := "Test User"
	userSvc.On("GetUser", mock.Anything, userID).Return(&models.User{
		ID:          userID,
		Username:    "testuser",
		DisplayName: &displayName,
	}, nil)

	voiceSvc.On("GenerateToken", mock.Anything, userID, channelID, "testuser", "Test User", "").
		Return(nil, assert.AnError)

	app := fiber.New()
	setupUserLocals(app, userID)
	handler := NewLiveKitVoiceHandler(voiceSvc, userSvc, channelSvc, nil)
	app.Post("/voice/token", handler.GenerateToken)

	body := `{"channel_id":"` + channelID.String() + `"}`
	req := httptest.NewRequest("POST", "/voice/token", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

// ─── GetParticipants Tests ───────────────────────────────────────────────────

func TestGetParticipants_Success(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	voiceSvc := new(MockVoiceService)
	userSvc := new(MockUserServiceForVoice)
	channelSvc := new(MockChannelServiceForVoice)

	channelSvc.On("GetChannel", mock.Anything, channelID).Return(&models.Channel{
		ID:       channelID,
		Type:     "voice",
		ServerID: &serverID,
	}, nil)
	channelSvc.On("GetServerChannels", mock.Anything, serverID, userID).Return([]*models.Channel{
		{ID: channelID, Type: "voice", ServerID: &serverID},
	}, nil)
	voiceSvc.On("GetRoomParticipants", mock.Anything, channelID).Return([]services.Participant{
		{ID: "user1", DisplayName: "User One", JoinedAt: 1000},
		{ID: "user2", DisplayName: "User Two", JoinedAt: 2000},
	}, nil)

	app := fiber.New()
	setupUserLocals(app, userID)
	handler := NewLiveKitVoiceHandler(voiceSvc, userSvc, channelSvc, nil)
	app.Get("/voice/participants/:channelId", handler.GetParticipants)

	req := httptest.NewRequest("GET", "/voice/participants/"+channelID.String(), nil)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result ParticipantsResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Len(t, result.Participants, 2)
	assert.Equal(t, "user1", result.Participants[0].ID)

	voiceSvc.AssertExpectations(t)
	channelSvc.AssertExpectations(t)
}

func TestGetParticipants_EmptyChannel(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	voiceSvc := new(MockVoiceService)
	userSvc := new(MockUserServiceForVoice)
	channelSvc := new(MockChannelServiceForVoice)

	channelSvc.On("GetChannel", mock.Anything, channelID).Return(&models.Channel{
		ID:       channelID,
		Type:     "voice",
		ServerID: &serverID,
	}, nil)
	channelSvc.On("GetServerChannels", mock.Anything, serverID, userID).Return([]*models.Channel{
		{ID: channelID, Type: "voice", ServerID: &serverID},
	}, nil)
	voiceSvc.On("GetRoomParticipants", mock.Anything, channelID).Return([]services.Participant{}, nil)

	app := fiber.New()
	setupUserLocals(app, userID)
	handler := NewLiveKitVoiceHandler(voiceSvc, userSvc, channelSvc, nil)
	app.Get("/voice/participants/:channelId", handler.GetParticipants)

	req := httptest.NewRequest("GET", "/voice/participants/"+channelID.String(), nil)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result ParticipantsResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Len(t, result.Participants, 0)
}

func TestGetParticipants_InvalidChannelID(t *testing.T) {
	userID := uuid.New()

	voiceSvc := new(MockVoiceService)
	userSvc := new(MockUserServiceForVoice)
	channelSvc := new(MockChannelServiceForVoice)

	app := fiber.New()
	setupUserLocals(app, userID)
	handler := NewLiveKitVoiceHandler(voiceSvc, userSvc, channelSvc, nil)
	app.Get("/voice/participants/:channelId", handler.GetParticipants)

	req := httptest.NewRequest("GET", "/voice/participants/not-a-uuid", nil)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestGetParticipants_ChannelNotFound(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	voiceSvc := new(MockVoiceService)
	userSvc := new(MockUserServiceForVoice)
	channelSvc := new(MockChannelServiceForVoice)

	channelSvc.On("GetChannel", mock.Anything, channelID).Return(nil, services.ErrChannelNotFound)

	app := fiber.New()
	setupUserLocals(app, userID)
	handler := NewLiveKitVoiceHandler(voiceSvc, userSvc, channelSvc, nil)
	app.Get("/voice/participants/:channelId", handler.GetParticipants)

	req := httptest.NewRequest("GET", "/voice/participants/"+channelID.String(), nil)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestGetParticipants_NotServerMember(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	voiceSvc := new(MockVoiceService)
	userSvc := new(MockUserServiceForVoice)
	channelSvc := new(MockChannelServiceForVoice)

	channelSvc.On("GetChannel", mock.Anything, channelID).Return(&models.Channel{
		ID:       channelID,
		Type:     "voice",
		ServerID: &serverID,
	}, nil)
	channelSvc.On("GetServerChannels", mock.Anything, serverID, userID).Return(nil, services.ErrNotServerMember)

	app := fiber.New()
	setupUserLocals(app, userID)
	handler := NewLiveKitVoiceHandler(voiceSvc, userSvc, channelSvc, nil)
	app.Get("/voice/participants/:channelId", handler.GetParticipants)

	req := httptest.NewRequest("GET", "/voice/participants/"+channelID.String(), nil)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

func TestGetParticipants_ServiceNotConfigured(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	voiceSvc := new(MockVoiceService)
	userSvc := new(MockUserServiceForVoice)
	channelSvc := new(MockChannelServiceForVoice)

	channelSvc.On("GetChannel", mock.Anything, channelID).Return(&models.Channel{
		ID:       channelID,
		Type:     "voice",
		ServerID: &serverID,
	}, nil)
	channelSvc.On("GetServerChannels", mock.Anything, serverID, userID).Return([]*models.Channel{
		{ID: channelID, Type: "voice", ServerID: &serverID},
	}, nil)
	voiceSvc.On("GetRoomParticipants", mock.Anything, channelID).Return(nil, services.ErrLiveKitNotConfigured)

	app := fiber.New()
	setupUserLocals(app, userID)
	handler := NewLiveKitVoiceHandler(voiceSvc, userSvc, channelSvc, nil)
	app.Get("/voice/participants/:channelId", handler.GetParticipants)

	req := httptest.NewRequest("GET", "/voice/participants/"+channelID.String(), nil)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)
}

// ─── DisconnectParticipant Tests ────────────────────────────────────────────

func TestDisconnectParticipant_Success(t *testing.T) {
	requesterID := uuid.New()
	targetUserID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	voiceSvc := new(MockVoiceService)
	userSvc := new(MockUserServiceForVoice)
	channelSvc := new(MockChannelServiceForVoice)
	permSvc := new(MockPermissionServiceForVoice)

	channelSvc.On("GetChannel", mock.Anything, channelID).Return(&models.Channel{
		ID:       channelID,
		Type:     "voice",
		ServerID: &serverID,
	}, nil)
	permSvc.On("RequirePermission", mock.Anything, serverID, requesterID, models.PermMoveMembers).Return(nil)
	voiceSvc.On("DisconnectParticipant", mock.Anything, channelID, targetUserID).Return(nil)

	app := fiber.New()
	setupUserLocals(app, requesterID)
	handler := NewLiveKitVoiceHandler(voiceSvc, userSvc, channelSvc, permSvc)
	app.Delete("/voice/participants/:channelId/:userId", handler.DisconnectParticipant)

	req := httptest.NewRequest("DELETE", "/voice/participants/"+channelID.String()+"/"+targetUserID.String(), nil)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)

	voiceSvc.AssertExpectations(t)
	permSvc.AssertExpectations(t)
}

func TestDisconnectParticipant_InvalidChannelID(t *testing.T) {
	requesterID := uuid.New()
	targetUserID := uuid.New()

	voiceSvc := new(MockVoiceService)
	userSvc := new(MockUserServiceForVoice)
	channelSvc := new(MockChannelServiceForVoice)
	permSvc := new(MockPermissionServiceForVoice)

	app := fiber.New()
	setupUserLocals(app, requesterID)
	handler := NewLiveKitVoiceHandler(voiceSvc, userSvc, channelSvc, permSvc)
	app.Delete("/voice/participants/:channelId/:userId", handler.DisconnectParticipant)

	req := httptest.NewRequest("DELETE", "/voice/participants/not-a-uuid/"+targetUserID.String(), nil)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestDisconnectParticipant_InvalidUserID(t *testing.T) {
	requesterID := uuid.New()
	channelID := uuid.New()

	voiceSvc := new(MockVoiceService)
	userSvc := new(MockUserServiceForVoice)
	channelSvc := new(MockChannelServiceForVoice)
	permSvc := new(MockPermissionServiceForVoice)

	app := fiber.New()
	setupUserLocals(app, requesterID)
	handler := NewLiveKitVoiceHandler(voiceSvc, userSvc, channelSvc, permSvc)
	app.Delete("/voice/participants/:channelId/:userId", handler.DisconnectParticipant)

	req := httptest.NewRequest("DELETE", "/voice/participants/"+channelID.String()+"/not-a-uuid", nil)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestDisconnectParticipant_ChannelNotFound(t *testing.T) {
	requesterID := uuid.New()
	targetUserID := uuid.New()
	channelID := uuid.New()

	voiceSvc := new(MockVoiceService)
	userSvc := new(MockUserServiceForVoice)
	channelSvc := new(MockChannelServiceForVoice)
	permSvc := new(MockPermissionServiceForVoice)

	channelSvc.On("GetChannel", mock.Anything, channelID).Return(nil, services.ErrChannelNotFound)

	app := fiber.New()
	setupUserLocals(app, requesterID)
	handler := NewLiveKitVoiceHandler(voiceSvc, userSvc, channelSvc, permSvc)
	app.Delete("/voice/participants/:channelId/:userId", handler.DisconnectParticipant)

	req := httptest.NewRequest("DELETE", "/voice/participants/"+channelID.String()+"/"+targetUserID.String(), nil)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestDisconnectParticipant_MissingMoveMembersPermission(t *testing.T) {
	requesterID := uuid.New()
	targetUserID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	voiceSvc := new(MockVoiceService)
	userSvc := new(MockUserServiceForVoice)
	channelSvc := new(MockChannelServiceForVoice)
	permSvc := new(MockPermissionServiceForVoice)

	channelSvc.On("GetChannel", mock.Anything, channelID).Return(&models.Channel{
		ID:       channelID,
		Type:     "voice",
		ServerID: &serverID,
	}, nil)
	permSvc.On("RequirePermission", mock.Anything, serverID, requesterID, models.PermMoveMembers).
		Return(services.ErrMissingMoveMembers)

	app := fiber.New()
	setupUserLocals(app, requesterID)
	handler := NewLiveKitVoiceHandler(voiceSvc, userSvc, channelSvc, permSvc)
	app.Delete("/voice/participants/:channelId/:userId", handler.DisconnectParticipant)

	req := httptest.NewRequest("DELETE", "/voice/participants/"+channelID.String()+"/"+targetUserID.String(), nil)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

func TestDisconnectParticipant_ServiceNotConfigured(t *testing.T) {
	requesterID := uuid.New()
	targetUserID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	voiceSvc := new(MockVoiceService)
	userSvc := new(MockUserServiceForVoice)
	channelSvc := new(MockChannelServiceForVoice)
	permSvc := new(MockPermissionServiceForVoice)

	channelSvc.On("GetChannel", mock.Anything, channelID).Return(&models.Channel{
		ID:       channelID,
		Type:     "voice",
		ServerID: &serverID,
	}, nil)
	permSvc.On("RequirePermission", mock.Anything, serverID, requesterID, models.PermMoveMembers).Return(nil)
	voiceSvc.On("DisconnectParticipant", mock.Anything, channelID, targetUserID).
		Return(services.ErrLiveKitNotConfigured)

	app := fiber.New()
	setupUserLocals(app, requesterID)
	handler := NewLiveKitVoiceHandler(voiceSvc, userSvc, channelSvc, permSvc)
	app.Delete("/voice/participants/:channelId/:userId", handler.DisconnectParticipant)

	req := httptest.NewRequest("DELETE", "/voice/participants/"+channelID.String()+"/"+targetUserID.String(), nil)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)
}

// ─── MuteParticipant Tests ───────────────────────────────────────────────────

func TestMuteParticipant_Success_Mute(t *testing.T) {
	requesterID := uuid.New()
	targetUserID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	voiceSvc := new(MockVoiceService)
	userSvc := new(MockUserServiceForVoice)
	channelSvc := new(MockChannelServiceForVoice)
	permSvc := new(MockPermissionServiceForVoice)

	channelSvc.On("GetChannel", mock.Anything, channelID).Return(&models.Channel{
		ID:       channelID,
		Type:     "voice",
		ServerID: &serverID,
	}, nil)
	permSvc.On("RequirePermission", mock.Anything, serverID, requesterID, models.PermMuteMembers).Return(nil)
	voiceSvc.On("MuteParticipant", mock.Anything, channelID, targetUserID, true).Return(nil)

	app := fiber.New()
	setupUserLocals(app, requesterID)
	handler := NewLiveKitVoiceHandler(voiceSvc, userSvc, channelSvc, permSvc)
	app.Post("/voice/participants/:channelId/:userId/mute", handler.MuteParticipant)

	body := `{"muted":true}`
	req := httptest.NewRequest("POST", "/voice/participants/"+channelID.String()+"/"+targetUserID.String()+"/mute",
		bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)

	voiceSvc.AssertExpectations(t)
	permSvc.AssertExpectations(t)
}

func TestMuteParticipant_Success_Unmute(t *testing.T) {
	requesterID := uuid.New()
	targetUserID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	voiceSvc := new(MockVoiceService)
	userSvc := new(MockUserServiceForVoice)
	channelSvc := new(MockChannelServiceForVoice)
	permSvc := new(MockPermissionServiceForVoice)

	channelSvc.On("GetChannel", mock.Anything, channelID).Return(&models.Channel{
		ID:       channelID,
		Type:     "voice",
		ServerID: &serverID,
	}, nil)
	permSvc.On("RequirePermission", mock.Anything, serverID, requesterID, models.PermMuteMembers).Return(nil)
	voiceSvc.On("MuteParticipant", mock.Anything, channelID, targetUserID, false).Return(nil)

	app := fiber.New()
	setupUserLocals(app, requesterID)
	handler := NewLiveKitVoiceHandler(voiceSvc, userSvc, channelSvc, permSvc)
	app.Post("/voice/participants/:channelId/:userId/mute", handler.MuteParticipant)

	body := `{"muted":false}`
	req := httptest.NewRequest("POST", "/voice/participants/"+channelID.String()+"/"+targetUserID.String()+"/mute",
		bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)
}

func TestMuteParticipant_InvalidChannelID(t *testing.T) {
	requesterID := uuid.New()
	targetUserID := uuid.New()

	voiceSvc := new(MockVoiceService)
	userSvc := new(MockUserServiceForVoice)
	channelSvc := new(MockChannelServiceForVoice)
	permSvc := new(MockPermissionServiceForVoice)

	app := fiber.New()
	setupUserLocals(app, requesterID)
	handler := NewLiveKitVoiceHandler(voiceSvc, userSvc, channelSvc, permSvc)
	app.Post("/voice/participants/:channelId/:userId/mute", handler.MuteParticipant)

	body := `{"muted":true}`
	req := httptest.NewRequest("POST", "/voice/participants/not-a-uuid/"+targetUserID.String()+"/mute",
		bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestMuteParticipant_InvalidUserID(t *testing.T) {
	requesterID := uuid.New()
	channelID := uuid.New()

	voiceSvc := new(MockVoiceService)
	userSvc := new(MockUserServiceForVoice)
	channelSvc := new(MockChannelServiceForVoice)
	permSvc := new(MockPermissionServiceForVoice)

	app := fiber.New()
	setupUserLocals(app, requesterID)
	handler := NewLiveKitVoiceHandler(voiceSvc, userSvc, channelSvc, permSvc)
	app.Post("/voice/participants/:channelId/:userId/mute", handler.MuteParticipant)

	body := `{"muted":true}`
	req := httptest.NewRequest("POST", "/voice/participants/"+channelID.String()+"/not-a-uuid/mute",
		bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestMuteParticipant_InvalidBody(t *testing.T) {
	requesterID := uuid.New()
	targetUserID := uuid.New()
	channelID := uuid.New()

	voiceSvc := new(MockVoiceService)
	userSvc := new(MockUserServiceForVoice)
	channelSvc := new(MockChannelServiceForVoice)
	permSvc := new(MockPermissionServiceForVoice)

	app := fiber.New()
	setupUserLocals(app, requesterID)
	handler := NewLiveKitVoiceHandler(voiceSvc, userSvc, channelSvc, permSvc)
	app.Post("/voice/participants/:channelId/:userId/mute", handler.MuteParticipant)

	body := `{invalid json}`
	req := httptest.NewRequest("POST", "/voice/participants/"+channelID.String()+"/"+targetUserID.String()+"/mute",
		bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestMuteParticipant_ChannelNotFound(t *testing.T) {
	requesterID := uuid.New()
	targetUserID := uuid.New()
	channelID := uuid.New()

	voiceSvc := new(MockVoiceService)
	userSvc := new(MockUserServiceForVoice)
	channelSvc := new(MockChannelServiceForVoice)
	permSvc := new(MockPermissionServiceForVoice)

	channelSvc.On("GetChannel", mock.Anything, channelID).Return(nil, services.ErrChannelNotFound)

	app := fiber.New()
	setupUserLocals(app, requesterID)
	handler := NewLiveKitVoiceHandler(voiceSvc, userSvc, channelSvc, permSvc)
	app.Post("/voice/participants/:channelId/:userId/mute", handler.MuteParticipant)

	body := `{"muted":true}`
	req := httptest.NewRequest("POST", "/voice/participants/"+channelID.String()+"/"+targetUserID.String()+"/mute",
		bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestMuteParticipant_MissingMuteMembersPermission(t *testing.T) {
	requesterID := uuid.New()
	targetUserID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	voiceSvc := new(MockVoiceService)
	userSvc := new(MockUserServiceForVoice)
	channelSvc := new(MockChannelServiceForVoice)
	permSvc := new(MockPermissionServiceForVoice)

	channelSvc.On("GetChannel", mock.Anything, channelID).Return(&models.Channel{
		ID:       channelID,
		Type:     "voice",
		ServerID: &serverID,
	}, nil)
	permSvc.On("RequirePermission", mock.Anything, serverID, requesterID, models.PermMuteMembers).
		Return(services.ErrMissingMuteMembers)

	app := fiber.New()
	setupUserLocals(app, requesterID)
	handler := NewLiveKitVoiceHandler(voiceSvc, userSvc, channelSvc, permSvc)
	app.Post("/voice/participants/:channelId/:userId/mute", handler.MuteParticipant)

	body := `{"muted":true}`
	req := httptest.NewRequest("POST", "/voice/participants/"+channelID.String()+"/"+targetUserID.String()+"/mute",
		bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

func TestMuteParticipant_ServiceNotConfigured(t *testing.T) {
	requesterID := uuid.New()
	targetUserID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	voiceSvc := new(MockVoiceService)
	userSvc := new(MockUserServiceForVoice)
	channelSvc := new(MockChannelServiceForVoice)
	permSvc := new(MockPermissionServiceForVoice)

	channelSvc.On("GetChannel", mock.Anything, channelID).Return(&models.Channel{
		ID:       channelID,
		Type:     "voice",
		ServerID: &serverID,
	}, nil)
	permSvc.On("RequirePermission", mock.Anything, serverID, requesterID, models.PermMuteMembers).Return(nil)
	voiceSvc.On("MuteParticipant", mock.Anything, channelID, targetUserID, true).
		Return(services.ErrLiveKitNotConfigured)

	app := fiber.New()
	setupUserLocals(app, requesterID)
	handler := NewLiveKitVoiceHandler(voiceSvc, userSvc, channelSvc, permSvc)
	app.Post("/voice/participants/:channelId/:userId/mute", handler.MuteParticipant)

	body := `{"muted":true}`
	req := httptest.NewRequest("POST", "/voice/participants/"+channelID.String()+"/"+targetUserID.String()+"/mute",
		bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)
}

// ─── Helper ─────────────────────────────────────────────────────────────────

// setupUserLocals installs middleware that sets userID locals for tests.
func setupUserLocals(app *fiber.App, userID uuid.UUID) {
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", userID)
		return c.Next()
	})
}