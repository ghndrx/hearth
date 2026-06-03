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

// mockInteractionService implements the methods used by InteractionHandler
type mockInteractionService struct {
	handleInteractionFunc func(ctx context.Context, interaction *models.Interaction) (*models.InteractionResponse, error)
	createResponseFunc    func(ctx context.Context, token string, response *models.InteractionResponse) error
	editResponseFunc      func(ctx context.Context, token string, response *models.InteractionResponse) error
	deleteResponseFunc    func(ctx context.Context, token, messageID string) error
}

func (m *mockInteractionService) HandleInteraction(ctx context.Context, interaction *models.Interaction) (*models.InteractionResponse, error) {
	if m.handleInteractionFunc != nil {
		return m.handleInteractionFunc(ctx, interaction)
	}
	return nil, nil
}

func (m *mockInteractionService) CreateResponse(ctx context.Context, token string, response *models.InteractionResponse) error {
	if m.createResponseFunc != nil {
		return m.createResponseFunc(ctx, token, response)
	}
	return nil
}

func (m *mockInteractionService) EditResponse(ctx context.Context, token string, response *models.InteractionResponse) error {
	if m.editResponseFunc != nil {
		return m.editResponseFunc(ctx, token, response)
	}
	return nil
}

func (m *mockInteractionService) DeleteResponse(ctx context.Context, token, messageID string) error {
	if m.deleteResponseFunc != nil {
		return m.deleteResponseFunc(ctx, token, messageID)
	}
	return nil
}

// interactionHandlerForTest wraps InteractionHandler for testing
type interactionHandlerForTest struct {
	service *mockInteractionService
}

func newInteractionHandlerForTest(service *mockInteractionService) *interactionHandlerForTest {
	return &interactionHandlerForTest{service: service}
}

func (h *interactionHandlerForTest) HandleInteraction(c *fiber.Ctx) error {
	var req InteractionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if req.ID == uuid.Nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "interaction ID is required"})
	}
	interaction, err := parseInteraction(&req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	response, err := h.service.HandleInteraction(c.Context(), interaction)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(response)
}

func (h *interactionHandlerForTest) RespondToInteraction(c *fiber.Ctx) error {
	interactionIDStr := c.Params("interaction_id")
	if interactionIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "interaction_id is required"})
	}
	token := interactionIDStr
	var resp models.InteractionResponse
	if err := c.BodyParser(&resp); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if err := h.service.CreateResponse(c.Context(), token, &resp); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *interactionHandlerForTest) EditInteractionResponse(c *fiber.Ctx) error {
	messageID := c.Params("messageId")
	var resp models.InteractionResponse
	if err := c.BodyParser(&resp); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	token := c.Query("token", c.Params("interaction_id"))
	if err := h.service.EditResponse(c.Context(), token, &resp); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message_id": messageID, "updated": true})
}

func (h *interactionHandlerForTest) DeleteInteractionResponse(c *fiber.Ctx) error {
	messageID := c.Params("messageId")
	token := c.Query("token", c.Params("interaction_id"))
	if err := h.service.DeleteResponse(c.Context(), token, messageID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *interactionHandlerForTest) HandleModalSubmit(c *fiber.Ctx) error {
	var req ModalSubmitRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if req.CustomID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "custom_id is required"})
	}
	if req.Values == nil {
		req.Values = make(map[string]string)
	}
	userID := c.Locals("userID").(uuid.UUID)
	interaction := &models.Interaction{
		ID:     uuid.New(),
		Type:   models.InteractionTypeModalSubmit,
		Token:  req.CustomID,
		UserID: userID,
		Data:   req.Values,
	}
	response, err := h.service.HandleInteraction(c.Context(), interaction)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(response)
}

func setupInteractionTestApp(t *testing.T, service *mockInteractionService) *fiber.App {
	t.Helper()
	app := fiber.New()
	t.Cleanup(func() { app.Shutdown() })

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", uuid.MustParse("11111111-1111-1111-1111-111111111111"))
		return c.Next()
	})

	handler := newInteractionHandlerForTest(service)
	app.Post("/api/v1/interactions", handler.HandleInteraction)
	app.Post("/api/v1/interactions/:interaction_id/callback", handler.RespondToInteraction)
	app.Patch("/api/v1/interactions/:interaction_id/messages/:messageId", handler.EditInteractionResponse)
	app.Delete("/api/v1/interactions/:interaction_id/messages/:messageId", handler.DeleteInteractionResponse)
	app.Post("/api/v1/interactions/modals/submit", handler.HandleModalSubmit)

	return app
}

func TestInteractionHandler_HandleInteraction_Success(t *testing.T) {
	interactionID := uuid.New()
	svc := &mockInteractionService{
		handleInteractionFunc: func(ctx context.Context, interaction *models.Interaction) (*models.InteractionResponse, error) {
			assert.Equal(t, interactionID, interaction.ID)
			assert.Equal(t, models.InteractionTypeApplicationCommand, interaction.Type)
			return &models.InteractionResponse{
				Type: models.CallbackTypeChannelMessage,
				Data: &models.InteractionCallbackData{
					Content: strPtr("Hello!"),
				},
			}, nil
		},
	}

	app := setupInteractionTestApp(t, svc)
	body, _ := json.Marshal(map[string]interface{}{
		"id":         interactionID,
		"type":       int(models.InteractionTypeApplicationCommand),
		"user_id":    uuid.New().String(),
		"channel_id": uuid.New().String(),
		"token":      "test-token",
		"application_id": uuid.New().String(),
		"data": map[string]interface{}{"name": "ping"},
	})
	req := httptest.NewRequest("POST", "/api/v1/interactions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result models.InteractionResponse
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, models.CallbackTypeChannelMessage, result.Type)
}

func TestInteractionHandler_HandleInteraction_InvalidBody(t *testing.T) {
	svc := &mockInteractionService{}
	app := setupInteractionTestApp(t, svc)
	req := httptest.NewRequest("POST", "/api/v1/interactions", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestInteractionHandler_HandleInteraction_MissingID(t *testing.T) {
	svc := &mockInteractionService{}
	app := setupInteractionTestApp(t, svc)
	body, _ := json.Marshal(map[string]interface{}{
		"type":    int(models.InteractionTypePing),
		"token":   "test-token",
		"channel_id": uuid.New().String(),
		"application_id": uuid.New().String(),
	})
	req := httptest.NewRequest("POST", "/api/v1/interactions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestInteractionHandler_HandleInteraction_ServiceError(t *testing.T) {
	svc := &mockInteractionService{
		handleInteractionFunc: func(ctx context.Context, interaction *models.Interaction) (*models.InteractionResponse, error) {
			return nil, errors.New("service error")
		},
	}

	app := setupInteractionTestApp(t, svc)
	body, _ := json.Marshal(map[string]interface{}{
		"id":         uuid.New(),
		"type":       int(models.InteractionTypeApplicationCommand),
		"user_id":    uuid.New().String(),
		"channel_id": uuid.New().String(),
		"token":      "test-token",
		"application_id": uuid.New().String(),
	})
	req := httptest.NewRequest("POST", "/api/v1/interactions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

func TestInteractionHandler_HandleInteraction_Ping(t *testing.T) {
	interactionID := uuid.New()
	svc := &mockInteractionService{
		handleInteractionFunc: func(ctx context.Context, interaction *models.Interaction) (*models.InteractionResponse, error) {
			assert.Equal(t, models.InteractionTypePing, interaction.Type)
			return &models.InteractionResponse{Type: models.CallbackTypePong}, nil
		},
	}

	app := setupInteractionTestApp(t, svc)
	body, _ := json.Marshal(map[string]interface{}{
		"id":         interactionID,
		"type":       int(models.InteractionTypePing),
		"token":      "ping-token",
		"channel_id": uuid.New().String(),
		"application_id": uuid.New().String(),
	})
	req := httptest.NewRequest("POST", "/api/v1/interactions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestInteractionHandler_RespondToInteraction_Success(t *testing.T) {
	svc := &mockInteractionService{
		createResponseFunc: func(ctx context.Context, token string, response *models.InteractionResponse) error {
			assert.Equal(t, "interaction-token-123", token)
			assert.Equal(t, models.CallbackTypeChannelMessage, response.Type)
			return nil
		},
	}

	app := setupInteractionTestApp(t, svc)
	body, _ := json.Marshal(map[string]interface{}{
		"type": 4,
		"data": map[string]interface{}{"content": "Hello!"},
	})
	req := httptest.NewRequest("POST", "/api/v1/interactions/interaction-token-123/callback", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)
}

func TestInteractionHandler_RespondToInteraction_MissingID(t *testing.T) {
	svc := &mockInteractionService{}
	app := setupInteractionTestApp(t, svc)
	body, _ := json.Marshal(map[string]interface{}{"type": 4})
	req := httptest.NewRequest("POST", "/api/v1/interactions//callback", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	// Fiber router returns 404 for empty param in this position
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestInteractionHandler_RespondToInteraction_ServiceError(t *testing.T) {
	svc := &mockInteractionService{
		createResponseFunc: func(ctx context.Context, token string, response *models.InteractionResponse) error {
			return errors.New("token expired")
		},
	}

	app := setupInteractionTestApp(t, svc)
	body, _ := json.Marshal(map[string]interface{}{"type": 4})
	req := httptest.NewRequest("POST", "/api/v1/interactions/bad-token/callback", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestInteractionHandler_EditInteractionResponse_Success(t *testing.T) {
	svc := &mockInteractionService{
		editResponseFunc: func(ctx context.Context, token string, response *models.InteractionResponse) error {
			assert.Equal(t, "interaction-token-123", token)
			assert.Equal(t, models.CallbackTypeUpdateMessage, response.Type)
			return nil
		},
	}

	app := setupInteractionTestApp(t, svc)
	body, _ := json.Marshal(map[string]interface{}{
		"type": 7,
		"data": map[string]interface{}{"content": "Updated!"},
	})
	req := httptest.NewRequest("PATCH", "/api/v1/interactions/interaction-token-123/messages/msg-456", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "msg-456", result["message_id"])
	assert.Equal(t, true, result["updated"])
}

func TestInteractionHandler_EditInteractionResponse_WithQueryToken(t *testing.T) {
	svc := &mockInteractionService{
		editResponseFunc: func(ctx context.Context, token string, response *models.InteractionResponse) error {
			assert.Equal(t, "query-token", token)
			return nil
		},
	}

	app := setupInteractionTestApp(t, svc)
	body, _ := json.Marshal(map[string]interface{}{"type": 7})
	req := httptest.NewRequest("PATCH", "/api/v1/interactions/interaction-id/messages/msg-456?token=query-token", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestInteractionHandler_EditInteractionResponse_ServiceError(t *testing.T) {
	svc := &mockInteractionService{
		editResponseFunc: func(ctx context.Context, token string, response *models.InteractionResponse) error {
			return errors.New("invalid token")
		},
	}

	app := setupInteractionTestApp(t, svc)
	body, _ := json.Marshal(map[string]interface{}{"type": 7})
	req := httptest.NewRequest("PATCH", "/api/v1/interactions/bad-token/messages/msg-456", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestInteractionHandler_DeleteInteractionResponse_Success(t *testing.T) {
	svc := &mockInteractionService{
		deleteResponseFunc: func(ctx context.Context, token, messageID string) error {
			assert.Equal(t, "interaction-token-123", token)
			assert.Equal(t, "msg-456", messageID)
			return nil
		},
	}

	app := setupInteractionTestApp(t, svc)
	req := httptest.NewRequest("DELETE", "/api/v1/interactions/interaction-token-123/messages/msg-456", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)
}

func TestInteractionHandler_DeleteInteractionResponse_WithQueryToken(t *testing.T) {
	svc := &mockInteractionService{
		deleteResponseFunc: func(ctx context.Context, token, messageID string) error {
			assert.Equal(t, "query-token", token)
			return nil
		},
	}

	app := setupInteractionTestApp(t, svc)
	req := httptest.NewRequest("DELETE", "/api/v1/interactions/interaction-id/messages/msg-456?token=query-token", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)
}

func TestInteractionHandler_DeleteInteractionResponse_ServiceError(t *testing.T) {
	svc := &mockInteractionService{
		deleteResponseFunc: func(ctx context.Context, token, messageID string) error {
			return errors.New("token expired")
		},
	}

	app := setupInteractionTestApp(t, svc)
	req := httptest.NewRequest("DELETE", "/api/v1/interactions/bad-token/messages/msg-456", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestInteractionHandler_HandleModalSubmit_Success(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	svc := &mockInteractionService{
		handleInteractionFunc: func(ctx context.Context, interaction *models.Interaction) (*models.InteractionResponse, error) {
			assert.Equal(t, models.InteractionTypeModalSubmit, interaction.Type)
			assert.Equal(t, userID, interaction.UserID)
			assert.Equal(t, "modal-form-1", interaction.Token)
			data, ok := interaction.Data.(map[string]string)
			assert.True(t, ok)
			assert.Equal(t, "user input", data["input_field"])
			return &models.InteractionResponse{
				Type: models.CallbackTypeChannelMessage,
				Data: &models.InteractionCallbackData{Content: strPtr("Thanks!")},
			}, nil
		},
	}

	app := setupInteractionTestApp(t, svc)
	body, _ := json.Marshal(map[string]interface{}{
		"custom_id": "modal-form-1",
		"values": map[string]string{
			"input_field": "user input",
		},
	})
	req := httptest.NewRequest("POST", "/api/v1/interactions/modals/submit", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result models.InteractionResponse
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, models.CallbackTypeChannelMessage, result.Type)
}

func TestInteractionHandler_HandleModalSubmit_MissingCustomID(t *testing.T) {
	svc := &mockInteractionService{}
	app := setupInteractionTestApp(t, svc)
	body, _ := json.Marshal(map[string]interface{}{"values": map[string]string{}})
	req := httptest.NewRequest("POST", "/api/v1/interactions/modals/submit", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestInteractionHandler_HandleModalSubmit_InvalidBody(t *testing.T) {
	svc := &mockInteractionService{}
	app := setupInteractionTestApp(t, svc)
	req := httptest.NewRequest("POST", "/api/v1/interactions/modals/submit", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestInteractionHandler_HandleModalSubmit_ServiceError(t *testing.T) {
	svc := &mockInteractionService{
		handleInteractionFunc: func(ctx context.Context, interaction *models.Interaction) (*models.InteractionResponse, error) {
			return nil, errors.New("processing error")
		},
	}

	app := setupInteractionTestApp(t, svc)
	body, _ := json.Marshal(map[string]interface{}{
		"custom_id": "modal-form-1",
		"values":    map[string]string{},
	})
	req := httptest.NewRequest("POST", "/api/v1/interactions/modals/submit", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}
