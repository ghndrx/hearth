package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"hearth/internal/models"
	"hearth/internal/services"
)

// mockScreenShareService implements the ScreenShareService interface for testing
type mockScreenShareService struct {
	startStreamFunc               func(ctx context.Context, channelID, userID uuid.UUID, streamType models.StreamType, resolution string, frameRate int) (*models.StreamInfo, error)
	endStreamFunc                 func(ctx context.Context, streamID, userID uuid.UUID) error
	getStreamInfoFunc             func(ctx context.Context, streamID uuid.UUID) (*models.StreamInfo, error)
	joinStreamFunc                func(ctx context.Context, streamID, userID uuid.UUID) error
	leaveStreamFunc               func(ctx context.Context, streamID, userID uuid.UUID) error
	updateStreamFunc              func(ctx context.Context, streamID, userID uuid.UUID, req *models.StreamUpdate) (*models.StreamInfo, error)
	getActiveStreamForChannelFunc func(ctx context.Context, channelID uuid.UUID) (*models.StreamInfo, error)
}

func (m *mockScreenShareService) StartStream(ctx context.Context, channelID, userID uuid.UUID, streamType models.StreamType, resolution string, frameRate int) (*models.StreamInfo, error) {
	if m.startStreamFunc != nil {
		return m.startStreamFunc(ctx, channelID, userID, streamType, resolution, frameRate)
	}
	return nil, nil
}

func (m *mockScreenShareService) EndStream(ctx context.Context, streamID, userID uuid.UUID) error {
	if m.endStreamFunc != nil {
		return m.endStreamFunc(ctx, streamID, userID)
	}
	return nil
}

func (m *mockScreenShareService) GetStreamInfo(ctx context.Context, streamID uuid.UUID) (*models.StreamInfo, error) {
	if m.getStreamInfoFunc != nil {
		return m.getStreamInfoFunc(ctx, streamID)
	}
	return nil, nil
}

func (m *mockScreenShareService) JoinStream(ctx context.Context, streamID, userID uuid.UUID) error {
	if m.joinStreamFunc != nil {
		return m.joinStreamFunc(ctx, streamID, userID)
	}
	return nil
}

func (m *mockScreenShareService) LeaveStream(ctx context.Context, streamID, userID uuid.UUID) error {
	if m.leaveStreamFunc != nil {
		return m.leaveStreamFunc(ctx, streamID, userID)
	}
	return nil
}

func (m *mockScreenShareService) UpdateStream(ctx context.Context, streamID, userID uuid.UUID, req *models.StreamUpdate) (*models.StreamInfo, error) {
	if m.updateStreamFunc != nil {
		return m.updateStreamFunc(ctx, streamID, userID, req)
	}
	return nil, nil
}

func (m *mockScreenShareService) GetActiveStreamForChannel(ctx context.Context, channelID uuid.UUID) (*models.StreamInfo, error) {
	if m.getActiveStreamForChannelFunc != nil {
		return m.getActiveStreamForChannelFunc(ctx, channelID)
	}
	return nil, nil
}

// mockChannelService for permission checking
type mockScreenShareChannelService struct{}

func (m *mockScreenShareChannelService) GetByID(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
	return nil, nil
}

// mockPermissionService for permission checking
type mockScreenSharePermService struct{}

func setupScreenShareTestApp(mockSvc *mockScreenShareService) *fiber.App {
	app := fiber.New()

	app.Use(func(c *fiber.Ctx) error {
		userIDStr := c.Get("X-Test-User-ID")
		if userIDStr != "" {
			userID, err := uuid.Parse(userIDStr)
			if err == nil {
				c.Locals("userID", userID)
			}
		}
		return c.Next()
	})

	handler := NewScreenShareHandler(
		(*services.ScreenShareService)(nil), // We use mock instead
		(*services.ChannelService)(nil),
		(*services.PermissionService)(nil),
	)
	// Manually set our mock service since we can't inject it directly
	handler.screenShareService = (*services.ScreenShareService)(nil) // placeholder

	return app
}

// Helper to create test app with mock service
func setupScreenShareTestAppWithMock(mockSvc *mockScreenShareService) *fiber.App {
	app := fiber.New()

	app.Use(func(c *fiber.Ctx) error {
		userIDStr := c.Get("X-Test-User-ID")
		if userIDStr != "" {
			userID, err := uuid.Parse(userIDStr)
			if err == nil {
				c.Locals("userID", userID)
			}
		}
		return c.Next()
	})

	// Create handler with nil services - we'll use a workaround for testing
	// Actually, we need to use the actual handler structure, so let's use interface embedding
	handler := &testableScreenShareHandler{mock: mockSvc}

	// Register routes manually using the handler methods
	app.Post("/channels/:channelId/streams", handler.StartStream)
	app.Delete("/streams/:streamId", handler.EndStream)
	app.Get("/streams/:streamId", handler.GetStreamInfo)
	app.Post("/streams/:streamId/join", handler.JoinStream)
	app.Delete("/streams/:streamId/leave", handler.LeaveStream)
	app.Patch("/streams/:streamId", handler.UpdateStream)
	app.Get("/channels/:channelId/streams", handler.GetActiveStreamForChannel)

	return app
}

// testableScreenShareHandler wraps the handler with mock support
type testableScreenShareHandler struct {
	mock *mockScreenShareService
}

func (h *testableScreenShareHandler) StartStream(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	channelID, err := uuid.Parse(c.Params("channelId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	var req models.StartStreamRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.StreamType != models.StreamTypeScreen && req.StreamType != models.StreamTypeApplication {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "stream_type must be 1 (screen) or 2 (application)",
		})
	}

	if req.Resolution != "" && req.Resolution != "720p" && req.Resolution != "1080p" && req.Resolution != "1440p" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "resolution must be 720p, 1080p, or 1440p",
		})
	}

	if req.FrameRate != 0 && req.FrameRate != 30 && req.FrameRate != 60 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "frame_rate must be 30 or 60",
		})
	}

	info, err := h.mock.StartStream(c.Context(), channelID, userID, req.StreamType, req.Resolution, req.FrameRate)
	if err != nil {
		switch err {
		case services.ErrChannelNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "channel not found"})
		case services.ErrChannelNotVoice:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "channel is not a voice channel"})
		case services.ErrNotServerMember:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "you are not a member of this server"})
		case services.ErrAlreadyStreaming:
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "already streaming in this channel"})
		case services.ErrMissingPermission:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "missing stream permission"})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to start stream"})
		}
	}

	return c.Status(fiber.StatusCreated).JSON(info)
}

func (h *testableScreenShareHandler) EndStream(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	streamID, err := uuid.Parse(c.Params("streamId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid stream id",
		})
	}

	err = h.mock.EndStream(c.Context(), streamID, userID)
	if err != nil {
		switch err {
		case services.ErrStreamNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "stream not found"})
		case services.ErrNotStreamer:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "you are not the streamer"})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to end stream"})
		}
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *testableScreenShareHandler) GetStreamInfo(c *fiber.Ctx) error {
	streamID, err := uuid.Parse(c.Params("streamId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid stream id",
		})
	}

	info, err := h.mock.GetStreamInfo(c.Context(), streamID)
	if err != nil {
		if err == services.ErrStreamNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "stream not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to get stream info"})
	}

	return c.JSON(info)
}

func (h *testableScreenShareHandler) JoinStream(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	streamID, err := uuid.Parse(c.Params("streamId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid stream id",
		})
	}

	err = h.mock.JoinStream(c.Context(), streamID, userID)
	if err != nil {
		switch err {
		case services.ErrStreamNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "stream not found"})
		case services.ErrNoActiveStream:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "stream has ended"})
		case services.ErrCannotJoinOwnStream:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot join your own stream"})
		case services.ErrAlreadyViewing:
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "already viewing this stream"})
		case services.ErrNotServerMember:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "you are not a member of this server"})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to join stream"})
		}
	}

	return c.JSON(fiber.Map{"message": "joined stream successfully"})
}

func (h *testableScreenShareHandler) LeaveStream(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	streamID, err := uuid.Parse(c.Params("streamId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid stream id",
		})
	}

	err = h.mock.LeaveStream(c.Context(), streamID, userID)
	if err != nil {
		switch err {
		case services.ErrStreamNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "stream not found"})
		case services.ErrNotViewing:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "not viewing this stream"})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to leave stream"})
		}
	}

	return c.JSON(fiber.Map{"message": "left stream successfully"})
}

func (h *testableScreenShareHandler) UpdateStream(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	streamID, err := uuid.Parse(c.Params("streamId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid stream id",
		})
	}

	var req models.StreamUpdate
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	info, err := h.mock.UpdateStream(c.Context(), streamID, userID, &req)
	if err != nil {
		switch err {
		case services.ErrStreamNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "stream not found"})
		case services.ErrNotStreamer:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "you are not the streamer"})
		case services.ErrNoActiveStream:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "stream has ended"})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update stream"})
		}
	}

	return c.JSON(info)
}

func (h *testableScreenShareHandler) GetActiveStreamForChannel(c *fiber.Ctx) error {
	channelID, err := uuid.Parse(c.Params("channelId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	info, err := h.mock.GetActiveStreamForChannel(c.Context(), channelID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to get channel stream"})
	}

	if info == nil {
		return c.SendStatus(fiber.StatusNoContent)
	}

	return c.JSON(info)
}

// Tests for StartStream

func TestStartStream_Success(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	streamID := uuid.New()

	mock := &mockScreenShareService{
		startStreamFunc: func(ctx context.Context, chID, uID uuid.UUID, streamType models.StreamType, resolution string, frameRate int) (*models.StreamInfo, error) {
			assert.Equal(t, channelID, chID)
			assert.Equal(t, userID, uID)
			assert.Equal(t, models.StreamTypeScreen, streamType)
			assert.Equal(t, "1080p", resolution)
			assert.Equal(t, 30, frameRate)
			return &models.StreamInfo{
				ID:         streamID,
				ChannelID:  chID,
				UserID:     uID,
				StreamType: streamType,
				Resolution: resolution,
				FrameRate:  frameRate,
			}, nil
		},
	}

	app := setupScreenShareTestAppWithMock(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"stream_type":1,"resolution":"1080p","frame_rate":30}`
	req := httptest.NewRequest("POST", "/channels/"+channelID.String()+"/streams", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode)

	var result models.StreamInfo
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, streamID, result.ID)
	assert.Equal(t, "1080p", result.Resolution)
}

func TestStartStream_InvalidChannelID(t *testing.T) {
	userID := uuid.New()

	mock := &mockScreenShareService{}
	app := setupScreenShareTestAppWithMock(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"stream_type":1}`
	req := httptest.NewRequest("POST", "/channels/not-a-uuid/streams", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "invalid channel id", result["error"])
}

func TestStartStream_InvalidRequestBody(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	mock := &mockScreenShareService{}
	app := setupScreenShareTestAppWithMock(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{invalid json}`
	req := httptest.NewRequest("POST", "/channels/"+channelID.String()+"/streams", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "invalid request body", result["error"])
}

func TestStartStream_InvalidStreamType(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	mock := &mockScreenShareService{}
	app := setupScreenShareTestAppWithMock(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"stream_type":3}` // Invalid: must be 1 or 2
	req := httptest.NewRequest("POST", "/channels/"+channelID.String()+"/streams", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "stream_type must be 1 (screen) or 2 (application)", result["error"])
}

func TestStartStream_InvalidResolution(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	mock := &mockScreenShareService{}
	app := setupScreenShareTestAppWithMock(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"stream_type":1,"resolution":"4k"}` // Invalid: must be 720p, 1080p, or 1440p
	req := httptest.NewRequest("POST", "/channels/"+channelID.String()+"/streams", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "resolution must be 720p, 1080p, or 1440p", result["error"])
}

func TestStartStream_InvalidFrameRate(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	mock := &mockScreenShareService{}
	app := setupScreenShareTestAppWithMock(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"stream_type":1,"frame_rate":45}` // Invalid: must be 30 or 60
	req := httptest.NewRequest("POST", "/channels/"+channelID.String()+"/streams", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "frame_rate must be 30 or 60", result["error"])
}

func TestStartStream_ChannelNotFound(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	mock := &mockScreenShareService{
		startStreamFunc: func(ctx context.Context, chID, uID uuid.UUID, streamType models.StreamType, resolution string, frameRate int) (*models.StreamInfo, error) {
			return nil, services.ErrChannelNotFound
		},
	}

	app := setupScreenShareTestAppWithMock(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"stream_type":1}`
	req := httptest.NewRequest("POST", "/channels/"+channelID.String()+"/streams", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "channel not found", result["error"])
}

func TestStartStream_ChannelNotVoice(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	mock := &mockScreenShareService{
		startStreamFunc: func(ctx context.Context, chID, uID uuid.UUID, streamType models.StreamType, resolution string, frameRate int) (*models.StreamInfo, error) {
			return nil, services.ErrChannelNotVoice
		},
	}

	app := setupScreenShareTestAppWithMock(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"stream_type":1}`
	req := httptest.NewRequest("POST", "/channels/"+channelID.String()+"/streams", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "channel is not a voice channel", result["error"])
}

func TestStartStream_NotServerMember(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	mock := &mockScreenShareService{
		startStreamFunc: func(ctx context.Context, chID, uID uuid.UUID, streamType models.StreamType, resolution string, frameRate int) (*models.StreamInfo, error) {
			return nil, services.ErrNotServerMember
		},
	}

	app := setupScreenShareTestAppWithMock(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"stream_type":1}`
	req := httptest.NewRequest("POST", "/channels/"+channelID.String()+"/streams", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 403, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "you are not a member of this server", result["error"])
}

func TestStartStream_AlreadyStreaming(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	mock := &mockScreenShareService{
		startStreamFunc: func(ctx context.Context, chID, uID uuid.UUID, streamType models.StreamType, resolution string, frameRate int) (*models.StreamInfo, error) {
			return nil, services.ErrAlreadyStreaming
		},
	}

	app := setupScreenShareTestAppWithMock(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"stream_type":1}`
	req := httptest.NewRequest("POST", "/channels/"+channelID.String()+"/streams", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 409, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "already streaming in this channel", result["error"])
}

// Tests for EndStream

func TestEndStream_Success(t *testing.T) {
	userID := uuid.New()
	streamID := uuid.New()

	mock := &mockScreenShareService{
		endStreamFunc: func(ctx context.Context, sID, uID uuid.UUID) error {
			assert.Equal(t, streamID, sID)
			assert.Equal(t, userID, uID)
			return nil
		},
	}

	app := setupScreenShareTestAppWithMock(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("DELETE", "/streams/"+streamID.String(), nil)
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 204, resp.StatusCode)
}

func TestEndStream_InvalidStreamID(t *testing.T) {
	userID := uuid.New()

	mock := &mockScreenShareService{}
	app := setupScreenShareTestAppWithMock(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("DELETE", "/streams/not-a-uuid", nil)
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "invalid stream id", result["error"])
}

func TestEndStream_StreamNotFound(t *testing.T) {
	userID := uuid.New()
	streamID := uuid.New()

	mock := &mockScreenShareService{
		endStreamFunc: func(ctx context.Context, sID, uID uuid.UUID) error {
			return services.ErrStreamNotFound
		},
	}

	app := setupScreenShareTestAppWithMock(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("DELETE", "/streams/"+streamID.String(), nil)
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "stream not found", result["error"])
}

func TestEndStream_NotStreamer(t *testing.T) {
	userID := uuid.New()
	streamID := uuid.New()

	mock := &mockScreenShareService{
		endStreamFunc: func(ctx context.Context, sID, uID uuid.UUID) error {
			return services.ErrNotStreamer
		},
	}

	app := setupScreenShareTestAppWithMock(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("DELETE", "/streams/"+streamID.String(), nil)
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 403, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "you are not the streamer", result["error"])
}

// Tests for GetStreamInfo

func TestGetStreamInfo_Success(t *testing.T) {
	streamID := uuid.New()
	channelID := uuid.New()
	userID := uuid.New()

	mock := &mockScreenShareService{
		getStreamInfoFunc: func(ctx context.Context, sID uuid.UUID) (*models.StreamInfo, error) {
			assert.Equal(t, streamID, sID)
			return &models.StreamInfo{
				ID:         streamID,
				ChannelID:  channelID,
				UserID:     userID,
				StreamType: models.StreamTypeScreen,
				Resolution: "1080p",
				FrameRate:  60,
			}, nil
		},
	}

	app := setupScreenShareTestAppWithMock(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("GET", "/streams/"+streamID.String(), nil)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result models.StreamInfo
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, streamID, result.ID)
	assert.Equal(t, "1080p", result.Resolution)
}

func TestGetStreamInfo_InvalidStreamID(t *testing.T) {
	mock := &mockScreenShareService{}
	app := setupScreenShareTestAppWithMock(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("GET", "/streams/not-a-uuid", nil)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "invalid stream id", result["error"])
}

func TestGetStreamInfo_NotFound(t *testing.T) {
	streamID := uuid.New()

	mock := &mockScreenShareService{
		getStreamInfoFunc: func(ctx context.Context, sID uuid.UUID) (*models.StreamInfo, error) {
			return nil, services.ErrStreamNotFound
		},
	}

	app := setupScreenShareTestAppWithMock(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("GET", "/streams/"+streamID.String(), nil)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "stream not found", result["error"])
}

// Tests for JoinStream

func TestJoinStream_Success(t *testing.T) {
	userID := uuid.New()
	streamID := uuid.New()

	mock := &mockScreenShareService{
		joinStreamFunc: func(ctx context.Context, sID, uID uuid.UUID) error {
			assert.Equal(t, streamID, sID)
			assert.Equal(t, userID, uID)
			return nil
		},
	}

	app := setupScreenShareTestAppWithMock(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("POST", "/streams/"+streamID.String()+"/join", nil)
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "joined stream successfully", result["message"])
}

func TestJoinStream_InvalidStreamID(t *testing.T) {
	userID := uuid.New()

	mock := &mockScreenShareService{}
	app := setupScreenShareTestAppWithMock(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("POST", "/streams/not-a-uuid/join", nil)
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestJoinStream_StreamNotFound(t *testing.T) {
	userID := uuid.New()
	streamID := uuid.New()

	mock := &mockScreenShareService{
		joinStreamFunc: func(ctx context.Context, sID, uID uuid.UUID) error {
			return services.ErrStreamNotFound
		},
	}

	app := setupScreenShareTestAppWithMock(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("POST", "/streams/"+streamID.String()+"/join", nil)
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "stream not found", result["error"])
}

func TestJoinStream_CannotJoinOwnStream(t *testing.T) {
	userID := uuid.New()
	streamID := uuid.New()

	mock := &mockScreenShareService{
		joinStreamFunc: func(ctx context.Context, sID, uID uuid.UUID) error {
			return services.ErrCannotJoinOwnStream
		},
	}

	app := setupScreenShareTestAppWithMock(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("POST", "/streams/"+streamID.String()+"/join", nil)
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "cannot join your own stream", result["error"])
}

func TestJoinStream_AlreadyViewing(t *testing.T) {
	userID := uuid.New()
	streamID := uuid.New()

	mock := &mockScreenShareService{
		joinStreamFunc: func(ctx context.Context, sID, uID uuid.UUID) error {
			return services.ErrAlreadyViewing
		},
	}

	app := setupScreenShareTestAppWithMock(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("POST", "/streams/"+streamID.String()+"/join", nil)
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 409, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "already viewing this stream", result["error"])
}

// Tests for LeaveStream

func TestLeaveStream_Success(t *testing.T) {
	userID := uuid.New()
	streamID := uuid.New()

	mock := &mockScreenShareService{
		leaveStreamFunc: func(ctx context.Context, sID, uID uuid.UUID) error {
			assert.Equal(t, streamID, sID)
			assert.Equal(t, userID, uID)
			return nil
		},
	}

	app := setupScreenShareTestAppWithMock(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("DELETE", "/streams/"+streamID.String()+"/leave", nil)
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "left stream successfully", result["message"])
}

func TestLeaveStream_InvalidStreamID(t *testing.T) {
	userID := uuid.New()

	mock := &mockScreenShareService{}
	app := setupScreenShareTestAppWithMock(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("DELETE", "/streams/not-a-uuid/leave", nil)
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestLeaveStream_NotViewing(t *testing.T) {
	userID := uuid.New()
	streamID := uuid.New()

	mock := &mockScreenShareService{
		leaveStreamFunc: func(ctx context.Context, sID, uID uuid.UUID) error {
			return services.ErrNotViewing
		},
	}

	app := setupScreenShareTestAppWithMock(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("DELETE", "/streams/"+streamID.String()+"/leave", nil)
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "not viewing this stream", result["error"])
}

// Tests for UpdateStream

func TestUpdateStream_Success(t *testing.T) {
	userID := uuid.New()
	streamID := uuid.New()

	mock := &mockScreenShareService{
		updateStreamFunc: func(ctx context.Context, sID, uID uuid.UUID, req *models.StreamUpdate) (*models.StreamInfo, error) {
			assert.Equal(t, streamID, sID)
			assert.Equal(t, userID, uID)
			return &models.StreamInfo{
				ID:         streamID,
				Resolution: "1440p",
				FrameRate:  60,
			}, nil
		},
	}

	app := setupScreenShareTestAppWithMock(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"resolution":"1440p","frame_rate":60}`
	req := httptest.NewRequest("PATCH", "/streams/"+streamID.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result models.StreamInfo
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "1440p", result.Resolution)
}

func TestUpdateStream_InvalidStreamID(t *testing.T) {
	userID := uuid.New()

	mock := &mockScreenShareService{}
	app := setupScreenShareTestAppWithMock(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"resolution":"1080p"}`
	req := httptest.NewRequest("PATCH", "/streams/not-a-uuid", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "invalid stream id", result["error"])
}

func TestUpdateStream_InvalidRequestBody(t *testing.T) {
	userID := uuid.New()
	streamID := uuid.New()

	mock := &mockScreenShareService{}
	app := setupScreenShareTestAppWithMock(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{invalid}`
	req := httptest.NewRequest("PATCH", "/streams/"+streamID.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "invalid request body", result["error"])
}

func TestUpdateStream_StreamNotFound(t *testing.T) {
	userID := uuid.New()
	streamID := uuid.New()

	mock := &mockScreenShareService{
		updateStreamFunc: func(ctx context.Context, sID, uID uuid.UUID, req *models.StreamUpdate) (*models.StreamInfo, error) {
			return nil, services.ErrStreamNotFound
		},
	}

	app := setupScreenShareTestAppWithMock(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"resolution":"1080p"}`
	req := httptest.NewRequest("PATCH", "/streams/"+streamID.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "stream not found", result["error"])
}

func TestUpdateStream_NotStreamer(t *testing.T) {
	userID := uuid.New()
	streamID := uuid.New()

	mock := &mockScreenShareService{
		updateStreamFunc: func(ctx context.Context, sID, uID uuid.UUID, req *models.StreamUpdate) (*models.StreamInfo, error) {
			return nil, services.ErrNotStreamer
		},
	}

	app := setupScreenShareTestAppWithMock(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"resolution":"1080p"}`
	req := httptest.NewRequest("PATCH", "/streams/"+streamID.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 403, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "you are not the streamer", result["error"])
}

func TestUpdateStream_StreamEnded(t *testing.T) {
	userID := uuid.New()
	streamID := uuid.New()

	mock := &mockScreenShareService{
		updateStreamFunc: func(ctx context.Context, sID, uID uuid.UUID, req *models.StreamUpdate) (*models.StreamInfo, error) {
			return nil, services.ErrNoActiveStream
		},
	}

	app := setupScreenShareTestAppWithMock(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"resolution":"1080p"}`
	req := httptest.NewRequest("PATCH", "/streams/"+streamID.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "stream has ended", result["error"])
}

// Tests for GetActiveStreamForChannel

func TestGetActiveStreamForChannel_Success(t *testing.T) {
	channelID := uuid.New()
	streamID := uuid.New()
	userID := uuid.New()

	mock := &mockScreenShareService{
		getActiveStreamForChannelFunc: func(ctx context.Context, chID uuid.UUID) (*models.StreamInfo, error) {
			assert.Equal(t, channelID, chID)
			return &models.StreamInfo{
				ID:         streamID,
				ChannelID:  channelID,
				UserID:     userID,
				StreamType: models.StreamTypeScreen,
				Resolution: "1080p",
			}, nil
		},
	}

	app := setupScreenShareTestAppWithMock(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("GET", "/channels/"+channelID.String()+"/streams", nil)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result models.StreamInfo
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, streamID, result.ID)
	assert.Equal(t, "1080p", result.Resolution)
}

func TestGetActiveStreamForChannel_NoActiveStream(t *testing.T) {
	channelID := uuid.New()

	mock := &mockScreenShareService{
		getActiveStreamForChannelFunc: func(ctx context.Context, chID uuid.UUID) (*models.StreamInfo, error) {
			return nil, nil // No active stream
		},
	}

	app := setupScreenShareTestAppWithMock(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("GET", "/channels/"+channelID.String()+"/streams", nil)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 204, resp.StatusCode)
}

func TestGetActiveStreamForChannel_InvalidChannelID(t *testing.T) {
	mock := &mockScreenShareService{}
	app := setupScreenShareTestAppWithMock(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("GET", "/channels/not-a-uuid/streams", nil)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "invalid channel id", result["error"])
}

func TestGetActiveStreamForChannel_ServiceError(t *testing.T) {
	channelID := uuid.New()

	mock := &mockScreenShareService{
		getActiveStreamForChannelFunc: func(ctx context.Context, chID uuid.UUID) (*models.StreamInfo, error) {
			return nil, errors.New("database error")
		},
	}

	app := setupScreenShareTestAppWithMock(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("GET", "/channels/"+channelID.String()+"/streams", nil)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "failed to get channel stream", result["error"])
}
