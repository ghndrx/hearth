package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"hearth/internal/models"
	"hearth/internal/services"
)

// MockVoiceStateService mocks the voice state service
type MockVoiceStateService struct {
	mock.Mock
}

func (m *MockVoiceStateService) Join(ctx context.Context, userID, channelID, serverID uuid.UUID) error {
	args := m.Called(ctx, userID, channelID, serverID)
	return args.Error(0)
}

func (m *MockVoiceStateService) Leave(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockVoiceStateService) SetMuted(ctx context.Context, userID uuid.UUID, muted bool) error {
	args := m.Called(ctx, userID, muted)
	return args.Error(0)
}

func (m *MockVoiceStateService) SetDeafened(ctx context.Context, userID uuid.UUID, deafened bool) error {
	args := m.Called(ctx, userID, deafened)
	return args.Error(0)
}

func (m *MockVoiceStateService) GetChannelUsers(ctx context.Context, channelID uuid.UUID) ([]*services.VoiceState, error) {
	args := m.Called(ctx, channelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*services.VoiceState), args.Error(1)
}

func (m *MockVoiceStateService) GetUserState(ctx context.Context, userID uuid.UUID) (*services.VoiceState, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.VoiceState), args.Error(1)
}

// MockChannelServiceForVoice mocks the channel service for voice operations
type MockChannelServiceForVoice struct {
	mock.Mock
}

func (m *MockChannelServiceForVoice) GetChannel(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Channel), args.Error(1)
}

// MockServerServiceForVoice mocks the server service for voice operations
type MockServerServiceForVoice struct {
	mock.Mock
}

func (m *MockServerServiceForVoice) GetMember(ctx context.Context, serverID, userID uuid.UUID) (*models.Member, error) {
	args := m.Called(ctx, serverID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Member), args.Error(1)
}

func setupVoiceTestApp(handler *VoiceHandler) *fiber.App {
	app := fiber.New()

	// Middleware to set userID
	app.Use(func(c *fiber.Ctx) error {
		userIDHeader := c.Get("X-User-ID")
		if userIDHeader != "" {
			userID, err := uuid.Parse(userIDHeader)
			if err == nil {
				c.Locals("userID", userID)
			}
		}
		return c.Next()
	})

	// Voice routes
	voice := app.Group("/voice")
	voice.Get("/regions", handler.GetRegions)
	voice.Post("/join", handler.JoinVoice)
	voice.Post("/leave", handler.LeaveVoice)
	voice.Patch("/state", handler.UpdateVoiceState)
	voice.Get("/state/@me", handler.GetMyVoiceState)

	// Channel voice states
	channels := app.Group("/channels")
	channels.Get("/:id/voice-states", handler.GetChannelVoiceStates)

	return app
}

func TestVoiceHandler_GetRegions(t *testing.T) {
	handler := NewVoiceHandler()
	app := setupVoiceTestApp(handler)

	req := httptest.NewRequest("GET", "/voice/regions", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var regions []VoiceRegion
	body, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &regions)
	assert.NoError(t, err)
	assert.Len(t, regions, 6)

	// Verify first region
	assert.Equal(t, "us-west", regions[0].ID)
	assert.Equal(t, "US West", regions[0].Name)
	assert.True(t, regions[0].Optimal)
}

func TestVoiceHandler_JoinVoice_Success(t *testing.T) {
	mockVoice := new(MockVoiceStateService)
	mockChannel := new(MockChannelServiceForVoice)
	mockServer := new(MockServerServiceForVoice)

	handler := NewVoiceHandlerWithService(mockVoice, mockChannel, mockServer)
	app := setupVoiceTestApp(handler)

	userID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	// Setup mocks
	mockChannel.On("GetChannel", mock.Anything, channelID).Return(&models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Type:     models.ChannelTypeVoice,
		Name:     "General Voice",
	}, nil)

	mockServer.On("GetMember", mock.Anything, serverID, userID).Return(&models.Member{
		UserID:   userID,
		ServerID: serverID,
	}, nil)

	mockVoice.On("Join", mock.Anything, userID, channelID, serverID).Return(nil)
	mockVoice.On("GetUserState", mock.Anything, userID).Return(&services.VoiceState{
		UserID:    userID,
		ChannelID: channelID,
		ServerID:  serverID,
		Muted:     false,
		Deafened:  false,
	}, nil)

	// Make request
	body, _ := json.Marshal(JoinVoiceRequest{ChannelID: channelID})
	req := httptest.NewRequest("POST", "/voice/join", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var state VoiceStateResponse
	respBody, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(respBody, &state)
	assert.NoError(t, err)
	assert.Equal(t, userID, state.UserID)
	assert.Equal(t, channelID, state.ChannelID)
	assert.Equal(t, serverID, state.ServerID)
	assert.False(t, state.Muted)
	assert.False(t, state.Deafened)

	mockChannel.AssertExpectations(t)
	mockServer.AssertExpectations(t)
	mockVoice.AssertExpectations(t)
}

func TestVoiceHandler_JoinVoice_ChannelNotFound(t *testing.T) {
	mockVoice := new(MockVoiceStateService)
	mockChannel := new(MockChannelServiceForVoice)
	mockServer := new(MockServerServiceForVoice)

	handler := NewVoiceHandlerWithService(mockVoice, mockChannel, mockServer)
	app := setupVoiceTestApp(handler)

	userID := uuid.New()
	channelID := uuid.New()

	mockChannel.On("GetChannel", mock.Anything, channelID).Return(nil, services.ErrChannelNotFound)

	body, _ := json.Marshal(JoinVoiceRequest{ChannelID: channelID})
	req := httptest.NewRequest("POST", "/voice/join", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestVoiceHandler_JoinVoice_NotVoiceChannel(t *testing.T) {
	mockVoice := new(MockVoiceStateService)
	mockChannel := new(MockChannelServiceForVoice)
	mockServer := new(MockServerServiceForVoice)

	handler := NewVoiceHandlerWithService(mockVoice, mockChannel, mockServer)
	app := setupVoiceTestApp(handler)

	userID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	mockChannel.On("GetChannel", mock.Anything, channelID).Return(&models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Type:     models.ChannelTypeText, // Not a voice channel
		Name:     "General",
	}, nil)

	body, _ := json.Marshal(JoinVoiceRequest{ChannelID: channelID})
	req := httptest.NewRequest("POST", "/voice/join", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestVoiceHandler_JoinVoice_NotServerMember(t *testing.T) {
	mockVoice := new(MockVoiceStateService)
	mockChannel := new(MockChannelServiceForVoice)
	mockServer := new(MockServerServiceForVoice)

	handler := NewVoiceHandlerWithService(mockVoice, mockChannel, mockServer)
	app := setupVoiceTestApp(handler)

	userID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	mockChannel.On("GetChannel", mock.Anything, channelID).Return(&models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Type:     models.ChannelTypeVoice,
		Name:     "Voice",
	}, nil)

	mockServer.On("GetMember", mock.Anything, serverID, userID).Return(nil, services.ErrNotServerMember)

	body, _ := json.Marshal(JoinVoiceRequest{ChannelID: channelID})
	req := httptest.NewRequest("POST", "/voice/join", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

func TestVoiceHandler_JoinVoice_MissingChannelID(t *testing.T) {
	mockVoice := new(MockVoiceStateService)
	mockChannel := new(MockChannelServiceForVoice)
	mockServer := new(MockServerServiceForVoice)

	handler := NewVoiceHandlerWithService(mockVoice, mockChannel, mockServer)
	app := setupVoiceTestApp(handler)

	userID := uuid.New()

	body, _ := json.Marshal(JoinVoiceRequest{}) // Empty channel ID
	req := httptest.NewRequest("POST", "/voice/join", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestVoiceHandler_LeaveVoice_Success(t *testing.T) {
	mockVoice := new(MockVoiceStateService)
	mockChannel := new(MockChannelServiceForVoice)
	mockServer := new(MockServerServiceForVoice)

	handler := NewVoiceHandlerWithService(mockVoice, mockChannel, mockServer)
	app := setupVoiceTestApp(handler)

	userID := uuid.New()

	mockVoice.On("Leave", mock.Anything, userID).Return(nil)

	req := httptest.NewRequest("POST", "/voice/leave", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)

	mockVoice.AssertExpectations(t)
}

func TestVoiceHandler_LeaveVoice_NotInChannel(t *testing.T) {
	mockVoice := new(MockVoiceStateService)
	mockChannel := new(MockChannelServiceForVoice)
	mockServer := new(MockServerServiceForVoice)

	handler := NewVoiceHandlerWithService(mockVoice, mockChannel, mockServer)
	app := setupVoiceTestApp(handler)

	userID := uuid.New()

	mockVoice.On("Leave", mock.Anything, userID).Return(services.ErrUserNotInVoice)

	req := httptest.NewRequest("POST", "/voice/leave", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	// Should still be 204 - leaving when not in voice is not an error
	assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)
}

func TestVoiceHandler_UpdateVoiceState_Mute(t *testing.T) {
	mockVoice := new(MockVoiceStateService)
	mockChannel := new(MockChannelServiceForVoice)
	mockServer := new(MockServerServiceForVoice)

	handler := NewVoiceHandlerWithService(mockVoice, mockChannel, mockServer)
	app := setupVoiceTestApp(handler)

	userID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	mockVoice.On("SetMuted", mock.Anything, userID, true).Return(nil)
	mockVoice.On("GetUserState", mock.Anything, userID).Return(&services.VoiceState{
		UserID:    userID,
		ChannelID: channelID,
		ServerID:  serverID,
		Muted:     true,
		Deafened:  false,
	}, nil)

	muted := true
	body, _ := json.Marshal(UpdateVoiceStateRequest{Muted: &muted})
	req := httptest.NewRequest("PATCH", "/voice/state", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var state VoiceStateResponse
	respBody, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(respBody, &state)
	assert.NoError(t, err)
	assert.True(t, state.Muted)

	mockVoice.AssertExpectations(t)
}

func TestVoiceHandler_UpdateVoiceState_Deafen(t *testing.T) {
	mockVoice := new(MockVoiceStateService)
	mockChannel := new(MockChannelServiceForVoice)
	mockServer := new(MockServerServiceForVoice)

	handler := NewVoiceHandlerWithService(mockVoice, mockChannel, mockServer)
	app := setupVoiceTestApp(handler)

	userID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	mockVoice.On("SetDeafened", mock.Anything, userID, true).Return(nil)
	mockVoice.On("GetUserState", mock.Anything, userID).Return(&services.VoiceState{
		UserID:    userID,
		ChannelID: channelID,
		ServerID:  serverID,
		Muted:     false,
		Deafened:  true,
	}, nil)

	deafened := true
	body, _ := json.Marshal(UpdateVoiceStateRequest{Deafened: &deafened})
	req := httptest.NewRequest("PATCH", "/voice/state", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var state VoiceStateResponse
	respBody, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(respBody, &state)
	assert.NoError(t, err)
	assert.True(t, state.Deafened)

	mockVoice.AssertExpectations(t)
}

func TestVoiceHandler_UpdateVoiceState_NotInVoice(t *testing.T) {
	mockVoice := new(MockVoiceStateService)
	mockChannel := new(MockChannelServiceForVoice)
	mockServer := new(MockServerServiceForVoice)

	handler := NewVoiceHandlerWithService(mockVoice, mockChannel, mockServer)
	app := setupVoiceTestApp(handler)

	userID := uuid.New()

	mockVoice.On("SetMuted", mock.Anything, userID, true).Return(services.ErrUserNotInVoice)

	muted := true
	body, _ := json.Marshal(UpdateVoiceStateRequest{Muted: &muted})
	req := httptest.NewRequest("PATCH", "/voice/state", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestVoiceHandler_GetMyVoiceState_Success(t *testing.T) {
	mockVoice := new(MockVoiceStateService)
	mockChannel := new(MockChannelServiceForVoice)
	mockServer := new(MockServerServiceForVoice)

	handler := NewVoiceHandlerWithService(mockVoice, mockChannel, mockServer)
	app := setupVoiceTestApp(handler)

	userID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	mockVoice.On("GetUserState", mock.Anything, userID).Return(&services.VoiceState{
		UserID:    userID,
		ChannelID: channelID,
		ServerID:  serverID,
		Muted:     true,
		Deafened:  false,
		Streaming: true,
	}, nil)

	req := httptest.NewRequest("GET", "/voice/state/@me", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var state VoiceStateResponse
	respBody, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(respBody, &state)
	assert.NoError(t, err)
	assert.Equal(t, userID, state.UserID)
	assert.Equal(t, channelID, state.ChannelID)
	assert.True(t, state.Muted)
	assert.True(t, state.Streaming)

	mockVoice.AssertExpectations(t)
}

func TestVoiceHandler_GetMyVoiceState_NotInVoice(t *testing.T) {
	mockVoice := new(MockVoiceStateService)
	mockChannel := new(MockChannelServiceForVoice)
	mockServer := new(MockServerServiceForVoice)

	handler := NewVoiceHandlerWithService(mockVoice, mockChannel, mockServer)
	app := setupVoiceTestApp(handler)

	userID := uuid.New()

	mockVoice.On("GetUserState", mock.Anything, userID).Return(nil, services.ErrUserNotInVoice)

	req := httptest.NewRequest("GET", "/voice/state/@me", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestVoiceHandler_GetChannelVoiceStates_Success(t *testing.T) {
	mockVoice := new(MockVoiceStateService)
	mockChannel := new(MockChannelServiceForVoice)
	mockServer := new(MockServerServiceForVoice)

	handler := NewVoiceHandlerWithService(mockVoice, mockChannel, mockServer)
	app := setupVoiceTestApp(handler)

	userID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()
	user2ID := uuid.New()
	user3ID := uuid.New()

	mockChannel.On("GetChannel", mock.Anything, channelID).Return(&models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Type:     models.ChannelTypeVoice,
		Name:     "Voice",
	}, nil)

	mockServer.On("GetMember", mock.Anything, serverID, userID).Return(&models.Member{
		UserID:   userID,
		ServerID: serverID,
	}, nil)

	mockVoice.On("GetChannelUsers", mock.Anything, channelID).Return([]*services.VoiceState{
		{UserID: user2ID, ChannelID: channelID, ServerID: serverID, Muted: false},
		{UserID: user3ID, ChannelID: channelID, ServerID: serverID, Muted: true},
	}, nil)

	req := httptest.NewRequest("GET", "/channels/"+channelID.String()+"/voice-states", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var states []VoiceStateResponse
	respBody, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(respBody, &states)
	assert.NoError(t, err)
	assert.Len(t, states, 2)

	mockChannel.AssertExpectations(t)
	mockServer.AssertExpectations(t)
	mockVoice.AssertExpectations(t)
}

func TestVoiceHandler_GetChannelVoiceStates_ChannelNotFound(t *testing.T) {
	mockVoice := new(MockVoiceStateService)
	mockChannel := new(MockChannelServiceForVoice)
	mockServer := new(MockServerServiceForVoice)

	handler := NewVoiceHandlerWithService(mockVoice, mockChannel, mockServer)
	app := setupVoiceTestApp(handler)

	userID := uuid.New()
	channelID := uuid.New()

	mockChannel.On("GetChannel", mock.Anything, channelID).Return(nil, services.ErrChannelNotFound)

	req := httptest.NewRequest("GET", "/channels/"+channelID.String()+"/voice-states", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestVoiceHandler_GetChannelVoiceStates_NotServerMember(t *testing.T) {
	mockVoice := new(MockVoiceStateService)
	mockChannel := new(MockChannelServiceForVoice)
	mockServer := new(MockServerServiceForVoice)

	handler := NewVoiceHandlerWithService(mockVoice, mockChannel, mockServer)
	app := setupVoiceTestApp(handler)

	userID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	mockChannel.On("GetChannel", mock.Anything, channelID).Return(&models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Type:     models.ChannelTypeVoice,
		Name:     "Voice",
	}, nil)

	mockServer.On("GetMember", mock.Anything, serverID, userID).Return(nil, services.ErrNotServerMember)

	req := httptest.NewRequest("GET", "/channels/"+channelID.String()+"/voice-states", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

func TestVoiceHandler_GetChannelVoiceStates_InvalidChannelID(t *testing.T) {
	mockVoice := new(MockVoiceStateService)
	mockChannel := new(MockChannelServiceForVoice)
	mockServer := new(MockServerServiceForVoice)

	handler := NewVoiceHandlerWithService(mockVoice, mockChannel, mockServer)
	app := setupVoiceTestApp(handler)

	userID := uuid.New()

	req := httptest.NewRequest("GET", "/channels/invalid-uuid/voice-states", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestVoiceHandler_ServiceNotAvailable(t *testing.T) {
	// Test with nil services (service unavailable state)
	handler := NewVoiceHandler() // No services attached
	app := setupVoiceTestApp(handler)

	userID := uuid.New()
	channelID := uuid.New()

	// Test join
	body, _ := json.Marshal(JoinVoiceRequest{ChannelID: channelID})
	req := httptest.NewRequest("POST", "/voice/join", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)

	// Test leave
	req = httptest.NewRequest("POST", "/voice/leave", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err = app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)

	// Test update state
	muted := true
	body, _ = json.Marshal(UpdateVoiceStateRequest{Muted: &muted})
	req = httptest.NewRequest("PATCH", "/voice/state", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())

	resp, err = app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)

	// Test get my state
	req = httptest.NewRequest("GET", "/voice/state/@me", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err = app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)

	// Test get channel states
	req = httptest.NewRequest("GET", "/channels/"+channelID.String()+"/voice-states", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err = app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)
}

func TestVoiceHandler_JoinStageChannel(t *testing.T) {
	mockVoice := new(MockVoiceStateService)
	mockChannel := new(MockChannelServiceForVoice)
	mockServer := new(MockServerServiceForVoice)

	handler := NewVoiceHandlerWithService(mockVoice, mockChannel, mockServer)
	app := setupVoiceTestApp(handler)

	userID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	// Setup mocks - stage channel should work too
	mockChannel.On("GetChannel", mock.Anything, channelID).Return(&models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Type:     models.ChannelTypeStage, // Stage channel
		Name:     "Stage Channel",
	}, nil)

	mockServer.On("GetMember", mock.Anything, serverID, userID).Return(&models.Member{
		UserID:   userID,
		ServerID: serverID,
	}, nil)

	mockVoice.On("Join", mock.Anything, userID, channelID, serverID).Return(nil)
	mockVoice.On("GetUserState", mock.Anything, userID).Return(&services.VoiceState{
		UserID:    userID,
		ChannelID: channelID,
		ServerID:  serverID,
	}, nil)

	body, _ := json.Marshal(JoinVoiceRequest{ChannelID: channelID})
	req := httptest.NewRequest("POST", "/voice/join", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	mockChannel.AssertExpectations(t)
	mockServer.AssertExpectations(t)
	mockVoice.AssertExpectations(t)
}

func TestVoiceHandler_UpdateVoiceState_BothMuteAndDeafen(t *testing.T) {
	mockVoice := new(MockVoiceStateService)
	mockChannel := new(MockChannelServiceForVoice)
	mockServer := new(MockServerServiceForVoice)

	handler := NewVoiceHandlerWithService(mockVoice, mockChannel, mockServer)
	app := setupVoiceTestApp(handler)

	userID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	mockVoice.On("SetMuted", mock.Anything, userID, true).Return(nil)
	mockVoice.On("SetDeafened", mock.Anything, userID, true).Return(nil)
	mockVoice.On("GetUserState", mock.Anything, userID).Return(&services.VoiceState{
		UserID:    userID,
		ChannelID: channelID,
		ServerID:  serverID,
		Muted:     true,
		Deafened:  true,
	}, nil)

	muted := true
	deafened := true
	body, _ := json.Marshal(UpdateVoiceStateRequest{Muted: &muted, Deafened: &deafened})
	req := httptest.NewRequest("PATCH", "/voice/state", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var state VoiceStateResponse
	respBody, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(respBody, &state)
	assert.NoError(t, err)
	assert.True(t, state.Muted)
	assert.True(t, state.Deafened)

	mockVoice.AssertExpectations(t)
}
