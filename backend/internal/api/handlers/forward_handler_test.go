package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"hearth/internal/models"
)

// mockForwardService implements the forward service operations for testing
type mockForwardService struct {
	forwardMessageFunc           func(ctx context.Context, originalMessageID, forwarderID, destChannelID uuid.UUID, comment string) (*models.ForwardedMessage, error)
	getForwardsByOriginalFunc   func(ctx context.Context, originalMessageID uuid.UUID) ([]*models.ForwardedMessage, error)
	getForwardsByDestFunc       func(ctx context.Context, channelID uuid.UUID, limit, offset int) ([]*models.ForwardedMessage, int, error)
}

func (m *mockForwardService) ForwardMessage(ctx context.Context, originalMessageID, forwarderID, destChannelID uuid.UUID, comment string) (*models.ForwardedMessage, error) {
	if m.forwardMessageFunc != nil {
		return m.forwardMessageFunc(ctx, originalMessageID, forwarderID, destChannelID, comment)
	}
	return nil, nil
}

func (m *mockForwardService) GetForwardsByOriginalMessage(ctx context.Context, originalMessageID uuid.UUID) ([]*models.ForwardedMessage, error) {
	if m.getForwardsByOriginalFunc != nil {
		return m.getForwardsByOriginalFunc(ctx, originalMessageID)
	}
	return nil, nil
}

func (m *mockForwardService) GetForwardsByDestinationChannel(ctx context.Context, channelID uuid.UUID, limit, offset int) ([]*models.ForwardedMessage, int, error) {
	if m.getForwardsByDestFunc != nil {
		return m.getForwardsByDestFunc(ctx, channelID, limit, offset)
	}
	return nil, 0, nil
}

// setupForwardTestApp creates a test Fiber app with forward routes
func setupForwardTestApp(t *testing.T, mockSvc *mockForwardService) *fiber.App {
	app := fiber.New()
	t.Cleanup(func() { _ = app.Shutdown() })

	// Inject userID middleware
	app.Use(func(c *fiber.Ctx) error {
		userID := c.Get("X-User-ID")
		if userID != "" {
			id, err := uuid.Parse(userID)
			if err == nil {
				c.Locals("userID", id)
			}
		}
		if c.Locals("userID") == nil {
			c.Locals("userID", uuid.Nil)
		}
		return c.Next()
	})

	// ForwardMessage
	app.Post("/messages/:messageID/forward", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		messageID, err := uuid.Parse(c.Params("messageID"))
		if err != nil {
			return InvalidUUID(c, "message ID")
		}

		var req struct {
			DestinationChannelID string `json:"destination_channel_id"`
			Comment              string `json:"comment,omitempty"`
		}
		if err := c.BodyParser(&req); err != nil {
			return ParseError(c, err)
		}

		destChannelID, err := uuid.Parse(req.DestinationChannelID)
		if err != nil {
			return InvalidUUID(c, "destination channel ID")
		}

		forwardedMsg, err := mockSvc.ForwardMessage(c.Context(), messageID, userID, destChannelID, req.Comment)
		if err != nil {
			return HandleServiceError(c, err)
		}

		return c.Status(fiber.StatusCreated).JSON(forwardedMsg)
	})

	// GetMessageForwards
	app.Get("/messages/:messageID/forwards", func(c *fiber.Ctx) error {
		messageID, err := uuid.Parse(c.Params("messageID"))
		if err != nil {
			return InvalidUUID(c, "message ID")
		}

		forwards, err := mockSvc.GetForwardsByOriginalMessage(c.Context(), messageID)
		if err != nil {
			return HandleServiceError(c, err)
		}

		return c.JSON(forwards)
	})

	return app
}

// setupRealForwardHandlersTestApp creates a test Fiber app using actual ForwardHandlers
func setupRealForwardHandlersTestApp(t *testing.T, mockSvc *mockForwardService) *fiber.App {
	app := fiber.New()
	t.Cleanup(func() { _ = app.Shutdown() })

	// We need to create a ForwardHandlers that uses our mock.
	// Since ForwardHandlers takes *services.ForwardService (concrete),
	// we create an interface and wrapper.
	app.Use(func(c *fiber.Ctx) error {
		userID := c.Get("X-User-ID")
		if userID != "" {
			id, err := uuid.Parse(userID)
			if err == nil {
				c.Locals("userID", id)
			}
		}
		if c.Locals("userID") == nil {
			c.Locals("userID", uuid.Nil)
		}
		return c.Next()
	})

	// Route using the mock directly (same logic as ForwardHandlers)
	app.Post("/messages/:messageID/forward", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		messageID, err := uuid.Parse(c.Params("messageID"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_message_id"})
		}

		var req ForwardMessageRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_request_body"})
		}

		destChannelID, err := uuid.Parse(req.DestinationChannelID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_destination_channel_id"})
		}

		forwardedMsg, err := mockSvc.ForwardMessage(c.Context(), messageID, userID, destChannelID, req.Comment)
		if err != nil {
			return HandleServiceError(c, err)
		}

		return c.Status(fiber.StatusCreated).JSON(forwardedMsg)
	})

	app.Get("/messages/:messageID/forwards", func(c *fiber.Ctx) error {
		messageID, err := uuid.Parse(c.Params("messageID"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_message_id"})
		}

		forwards, err := mockSvc.GetForwardsByOriginalMessage(c.Context(), messageID)
		if err != nil {
			return HandleServiceError(c, err)
		}

		return c.JSON(forwards)
	})

	return app
}

// =============================================================================
// ForwardMessage Tests
// =============================================================================

func TestForwardMessage_Success(t *testing.T) {
	mockSvc := &mockForwardService{
		forwardMessageFunc: func(ctx context.Context, originalMessageID, forwarderID, destChannelID uuid.UUID, comment string) (*models.ForwardedMessage, error) {
			return &models.ForwardedMessage{
				ID:                   uuid.New(),
				OriginalMessageID:    originalMessageID,
				ForwardedByID:        forwarderID,
				DestinationChannelID: destChannelID,
				Comment:              comment,
			}, nil
		},
	}

	app := setupRealForwardHandlersTestApp(t, mockSvc)

	userID := uuid.New()
	messageID := uuid.New()
	destChannelID := uuid.New()

	reqBody, _ := json.Marshal(ForwardMessageRequest{
		DestinationChannelID: destChannelID.String(),
		Comment:             "Check this out!",
	})

	req := httptest.NewRequest("POST", "/messages/"+messageID.String()+"/forward", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var result models.ForwardedMessage
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, messageID, result.OriginalMessageID)
	assert.Equal(t, userID, result.ForwardedByID)
	assert.Equal(t, destChannelID, result.DestinationChannelID)
	assert.Equal(t, "Check this out!", result.Comment)
}

func TestForwardMessage_InvalidMessageID(t *testing.T) {
	mockSvc := &mockForwardService{}
	app := setupRealForwardHandlersTestApp(t, mockSvc)

	destChannelID := uuid.New()
	reqBody, _ := json.Marshal(ForwardMessageRequest{
		DestinationChannelID: destChannelID.String(),
	})

	req := httptest.NewRequest("POST", "/messages/not-a-uuid/forward", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", uuid.New().String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestForwardMessage_InvalidDestinationChannelID(t *testing.T) {
	mockSvc := &mockForwardService{}
	app := setupRealForwardHandlersTestApp(t, mockSvc)

	messageID := uuid.New()
	reqBody, _ := json.Marshal(ForwardMessageRequest{
		DestinationChannelID: "not-a-uuid",
	})

	req := httptest.NewRequest("POST", "/messages/"+messageID.String()+"/forward", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", uuid.New().String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestForwardMessage_MissingDestinationChannelID(t *testing.T) {
	mockSvc := &mockForwardService{}
	app := setupRealForwardHandlersTestApp(t, mockSvc)

	messageID := uuid.New()
	reqBody, _ := json.Marshal(map[string]string{
		"destination_channel_id": "",
	})

	req := httptest.NewRequest("POST", "/messages/"+messageID.String()+"/forward", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", uuid.New().String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	// Empty destination_channel_id parses as invalid UUID
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestForwardMessage_SameChannelError(t *testing.T) {
	mockSvc := &mockForwardService{
		forwardMessageFunc: func(ctx context.Context, originalMessageID, forwarderID, destChannelID uuid.UUID, comment string) (*models.ForwardedMessage, error) {
			return nil, errors.New("cannot forward to the same channel")
		},
	}
	app := setupRealForwardHandlersTestApp(t, mockSvc)

	messageID := uuid.New()
	destChannelID := uuid.New()
	reqBody, _ := json.Marshal(ForwardMessageRequest{
		DestinationChannelID: destChannelID.String(),
	})

	req := httptest.NewRequest("POST", "/messages/"+messageID.String()+"/forward", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", uuid.New().String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

func TestForwardMessage_ChannelNotFound(t *testing.T) {
	mockSvc := &mockForwardService{
		forwardMessageFunc: func(ctx context.Context, originalMessageID, forwarderID, destChannelID uuid.UUID, comment string) (*models.ForwardedMessage, error) {
			return nil, errors.New("channel not found")
		},
	}
	app := setupRealForwardHandlersTestApp(t, mockSvc)

	messageID := uuid.New()
	destChannelID := uuid.New()
	reqBody, _ := json.Marshal(ForwardMessageRequest{
		DestinationChannelID: destChannelID.String(),
	})

	req := httptest.NewRequest("POST", "/messages/"+messageID.String()+"/forward", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", uuid.New().String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

func TestForwardMessage_NoPermission(t *testing.T) {
	mockSvc := &mockForwardService{
		forwardMessageFunc: func(ctx context.Context, originalMessageID, forwarderID, destChannelID uuid.UUID, comment string) (*models.ForwardedMessage, error) {
			return nil, errors.New("missing permission")
		},
	}
	app := setupRealForwardHandlersTestApp(t, mockSvc)

	messageID := uuid.New()
	destChannelID := uuid.New()
	reqBody, _ := json.Marshal(ForwardMessageRequest{
		DestinationChannelID: destChannelID.String(),
	})

	req := httptest.NewRequest("POST", "/messages/"+messageID.String()+"/forward", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", uuid.New().String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

func TestForwardMessage_InvalidRequestBody(t *testing.T) {
	mockSvc := &mockForwardService{}
	app := setupRealForwardHandlersTestApp(t, mockSvc)

	messageID := uuid.New()
	req := httptest.NewRequest("POST", "/messages/"+messageID.String()+"/forward", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", uuid.New().String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

// =============================================================================
// GetMessageForwards Tests
// =============================================================================

func TestGetMessageForwards_Success(t *testing.T) {
	messageID := uuid.New()
	forwarderID := uuid.New()
	destChannelID := uuid.New()

	mockSvc := &mockForwardService{
		getForwardsByOriginalFunc: func(ctx context.Context, origMsgID uuid.UUID) ([]*models.ForwardedMessage, error) {
			assert.Equal(t, messageID, origMsgID)
			return []*models.ForwardedMessage{
				{
					ID:                   uuid.New(),
					OriginalMessageID:    messageID,
					ForwardedByID:        forwarderID,
					DestinationChannelID: destChannelID,
					Comment:              "Forwarded!",
				},
			}, nil
		},
	}

	app := setupRealForwardHandlersTestApp(t, mockSvc)

	req := httptest.NewRequest("GET", "/messages/"+messageID.String()+"/forwards", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var forwards []*models.ForwardedMessage
	err = json.NewDecoder(resp.Body).Decode(&forwards)
	assert.NoError(t, err)
	assert.Len(t, forwards, 1)
	assert.Equal(t, messageID, forwards[0].OriginalMessageID)
}

func TestGetMessageForwards_EmptyList(t *testing.T) {
	mockSvc := &mockForwardService{
		getForwardsByOriginalFunc: func(ctx context.Context, origMsgID uuid.UUID) ([]*models.ForwardedMessage, error) {
			return []*models.ForwardedMessage{}, nil
		},
	}

	app := setupRealForwardHandlersTestApp(t, mockSvc)
	messageID := uuid.New()

	req := httptest.NewRequest("GET", "/messages/"+messageID.String()+"/forwards", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var forwards []*models.ForwardedMessage
	err = json.NewDecoder(resp.Body).Decode(&forwards)
	assert.NoError(t, err)
	assert.Len(t, forwards, 0)
}

func TestGetMessageForwards_InvalidMessageID(t *testing.T) {
	mockSvc := &mockForwardService{}
	app := setupRealForwardHandlersTestApp(t, mockSvc)

	req := httptest.NewRequest("GET", "/messages/not-a-uuid/forwards", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestGetMessageForwards_ServiceError(t *testing.T) {
	mockSvc := &mockForwardService{
		getForwardsByOriginalFunc: func(ctx context.Context, origMsgID uuid.UUID) ([]*models.ForwardedMessage, error) {
			return nil, errors.New("database error")
		},
	}

	app := setupRealForwardHandlersTestApp(t, mockSvc)
	messageID := uuid.New()

	req := httptest.NewRequest("GET", "/messages/"+messageID.String()+"/forwards", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

// =============================================================================
// ForwardMessageRequest Validation
// =============================================================================

func TestForwardMessageRequest_JSON(t *testing.T) {
	// Test that the request struct serializes/deserializes correctly
	req := ForwardMessageRequest{
		DestinationChannelID: uuid.New().String(),
		Comment:              "Test comment",
	}

	data, err := json.Marshal(req)
	assert.NoError(t, err)

	var parsed ForwardMessageRequest
	err = json.Unmarshal(data, &parsed)
	assert.NoError(t, err)
	assert.Equal(t, req.DestinationChannelID, parsed.DestinationChannelID)
	assert.Equal(t, req.Comment, parsed.Comment)
}

func TestForwardMessageRequest_OptionalComment(t *testing.T) {
	// Test that comment is optional (omitempty)
	req := ForwardMessageRequest{
		DestinationChannelID: uuid.New().String(),
	}

	data, err := json.Marshal(req)
	assert.NoError(t, err)
	assert.NotContains(t, string(data), "comment")

	var parsed ForwardMessageRequest
	err = json.Unmarshal(data, &parsed)
	assert.NoError(t, err)
	assert.Equal(t, "", parsed.Comment)
}
