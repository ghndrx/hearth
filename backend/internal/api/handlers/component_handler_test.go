package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"hearth/internal/models"
)

// =============================================================================
// Mock implementations for ComponentHandler testing
// =============================================================================

// mockComponentHandlerComponentService mocks ComponentServiceInterface
type mockComponentHandlerComponentService struct {
	getMessageComponentsFunc    func(ctx context.Context, messageID uuid.UUID) ([]*models.MessageComponent, error)
	handleInteractionFunc       func(ctx context.Context, userID, channelID, messageID, componentID uuid.UUID, customID string, values []string) (*models.ComponentInteraction, error)
	updateMessageComponentsFunc func(ctx context.Context, messageID uuid.UUID, components []*models.MessageComponent) ([]*models.MessageComponent, error)
	removeAllComponentsFunc    func(ctx context.Context, messageID uuid.UUID) error
}

func (m *mockComponentHandlerComponentService) GetMessageComponents(ctx context.Context, messageID uuid.UUID) ([]*models.MessageComponent, error) {
	if m.getMessageComponentsFunc != nil {
		return m.getMessageComponentsFunc(ctx, messageID)
	}
	return nil, nil
}

func (m *mockComponentHandlerComponentService) HandleInteraction(ctx context.Context, userID, channelID, messageID, componentID uuid.UUID, customID string, values []string) (*models.ComponentInteraction, error) {
	if m.handleInteractionFunc != nil {
		return m.handleInteractionFunc(ctx, userID, channelID, messageID, componentID, customID, values)
	}
	return nil, nil
}

func (m *mockComponentHandlerComponentService) UpdateMessageComponents(ctx context.Context, messageID uuid.UUID, components []*models.MessageComponent) ([]*models.MessageComponent, error) {
	if m.updateMessageComponentsFunc != nil {
		return m.updateMessageComponentsFunc(ctx, messageID, components)
	}
	return nil, nil
}

func (m *mockComponentHandlerComponentService) RemoveAllComponents(ctx context.Context, messageID uuid.UUID) error {
	if m.removeAllComponentsFunc != nil {
		return m.removeAllComponentsFunc(ctx, messageID)
	}
	return nil
}

func (m *mockComponentHandlerComponentService) CreateModal(ctx context.Context, modal *models.ModalComponent) (*models.ModalComponent, error) {
	return nil, nil
}

func (m *mockComponentHandlerComponentService) GetModalByCustomID(ctx context.Context, customID string) (*models.ModalComponent, error) {
	return nil, nil
}

func (m *mockComponentHandlerComponentService) DeleteModal(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockComponentHandlerComponentService) HandleModalSubmit(ctx context.Context, userID, channelID, msgID, modalID, componentID uuid.UUID, customID string, values map[string]string) (*models.ModalInteraction, error) {
	return nil, nil
}

// mockComponentHandlerMessageService mocks MessageServiceGetMessageInterface
type mockComponentHandlerMessageService struct {
	getMessageFunc func(ctx context.Context, messageID, userID uuid.UUID) (*models.Message, error)
}

func (m *mockComponentHandlerMessageService) GetMessage(ctx context.Context, messageID, userID uuid.UUID) (*models.Message, error) {
	if m.getMessageFunc != nil {
		return m.getMessageFunc(ctx, messageID, userID)
	}
	return nil, nil
}

// mockComponentHandlerChannelService mocks ChannelServiceGetChannelInterface
type mockComponentHandlerChannelService struct {
	getChannelFunc func(ctx context.Context, channelID uuid.UUID) (*models.Channel, error)
}

func (m *mockComponentHandlerChannelService) GetChannel(ctx context.Context, channelID uuid.UUID) (*models.Channel, error) {
	if m.getChannelFunc != nil {
		return m.getChannelFunc(ctx, channelID)
	}
	return nil, nil
}

// mockComponentHandlerPermissionService mocks PermissionServiceGetChannelPermissionsInterface
type mockComponentHandlerPermissionService struct {
	getChannelPermissionsFunc func(ctx context.Context, channel *models.Channel, userID uuid.UUID) (int64, error)
}

func (m *mockComponentHandlerPermissionService) GetChannelPermissions(ctx context.Context, channel *models.Channel, userID uuid.UUID) (int64, error) {
	if m.getChannelPermissionsFunc != nil {
		return m.getChannelPermissionsFunc(ctx, channel, userID)
	}
	return 0, nil
}

// componentHandlerRouter creates a Fiber app with ComponentHandler routes for testing
func setupComponentHandlerTestApp(t *testing.T, h *ComponentHandler) *fiber.App {
	app := fiber.New()
	t.Cleanup(func() { app.Shutdown() })

	// Inject userID middleware
	app.Use(func(c *fiber.Ctx) error {
		userID := c.Get("X-User-ID")
		if userID != "" {
			id, _ := uuid.Parse(userID)
			c.Locals("userID", id)
		}
		return c.Next()
	})

	// HandleComponentInteractionV2 - POST /api/v1/interactions/components
	app.Post("/api/v1/interactions/components", h.HandleComponentInteractionV2)

	// GetMessageComponents - GET /api/v1/channels/:id/messages/:messageId/components
	app.Get("/api/v1/channels/:id/messages/:messageId/components", h.GetMessageComponents)

	// UpdateMessageComponents - PATCH /api/v1/channels/:id/messages/:messageId/components
	app.Patch("/api/v1/channels/:id/messages/:messageId/components", h.UpdateMessageComponents)

	// RemoveAllComponents - DELETE /api/v1/channels/:id/messages/:messageId/components
	app.Delete("/api/v1/channels/:id/messages/:messageId/components", h.RemoveAllComponents)

	return app
}



// =============================================================================
// Tests for HandleComponentInteractionV2
// =============================================================================

func TestComponentHandler_HandleComponentInteractionV2_Success(t *testing.T) {
	componentSvc := &mockComponentHandlerComponentService{}
	handler := &ComponentHandler{componentService: componentSvc}
	app := setupComponentHandlerTestApp(t, handler)

	userID := uuid.New()
	messageID := uuid.New()
	channelID := uuid.New()
	componentID := uuid.New()
	customID := "my_button"

	componentSvc.getMessageComponentsFunc = func(ctx context.Context, msgID uuid.UUID) ([]*models.MessageComponent, error) {
		return []*models.MessageComponent{{ID: componentID, CustomID: customID, Type: models.ComponentTypeButton}}, nil
	}
	componentSvc.handleInteractionFunc = func(ctx context.Context, uid, chID, msgID, compID uuid.UUID, custID string, vals []string) (*models.ComponentInteraction, error) {
		return &models.ComponentInteraction{ID: uuid.New(), UserID: uid, ChannelID: chID, MessageID: msgID, ComponentID: compID, CustomID: custID}, nil
	}

	body := map[string]interface{}{"message_id": messageID.String(), "channel_id": channelID.String(), "custom_id": customID}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/interactions/components", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var interaction models.ComponentInteraction
	err = json.NewDecoder(resp.Body).Decode(&interaction)
	assert.NoError(t, err)
	assert.Equal(t, userID, interaction.UserID)
	assert.Equal(t, channelID, interaction.ChannelID)
}

func TestComponentHandler_HandleComponentInteractionV2_InvalidJSON(t *testing.T) {
	handler := &ComponentHandler{componentService: &mockComponentHandlerComponentService{}}
	app := setupComponentHandlerTestApp(t, handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/interactions/components", bytes.NewReader([]byte(`{invalid`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", uuid.New().String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestComponentHandler_HandleComponentInteractionV2_MissingFields(t *testing.T) {
	handler := &ComponentHandler{componentService: &mockComponentHandlerComponentService{}}
	app := setupComponentHandlerTestApp(t, handler)

	body := map[string]interface{}{"message_id": uuid.New().String()}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/interactions/components", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", uuid.New().String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	respBody := readBody(resp.Body)
	assert.Contains(t, respBody, "required")
}

func TestComponentHandler_HandleComponentInteractionV2_InvalidMessageUUID(t *testing.T) {
	handler := &ComponentHandler{componentService: &mockComponentHandlerComponentService{}}
	app := setupComponentHandlerTestApp(t, handler)

	body := map[string]interface{}{"message_id": "not-a-uuid", "channel_id": uuid.New().String(), "custom_id": "btn"}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/interactions/components", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", uuid.New().String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	respBody := readBody(resp.Body)
	assert.Contains(t, respBody, "message ID")
}

func TestComponentHandler_HandleComponentInteractionV2_InvalidChannelUUID(t *testing.T) {
	handler := &ComponentHandler{componentService: &mockComponentHandlerComponentService{}}
	app := setupComponentHandlerTestApp(t, handler)

	body := map[string]interface{}{"message_id": uuid.New().String(), "channel_id": "not-a-uuid", "custom_id": "btn"}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/interactions/components", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", uuid.New().String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	respBody := readBody(resp.Body)
	assert.Contains(t, respBody, "channel ID")
}

func TestComponentHandler_HandleComponentInteractionV2_ComponentNotFound(t *testing.T) {
	componentSvc := &mockComponentHandlerComponentService{}
	handler := &ComponentHandler{componentService: componentSvc}
	app := setupComponentHandlerTestApp(t, handler)

	componentSvc.getMessageComponentsFunc = func(ctx context.Context, msgID uuid.UUID) ([]*models.MessageComponent, error) {
		return []*models.MessageComponent{{ID: uuid.New(), CustomID: "other_btn", Type: models.ComponentTypeButton}}, nil
	}

	body := map[string]interface{}{"message_id": uuid.New().String(), "channel_id": uuid.New().String(), "custom_id": "my_btn"}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/interactions/components", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", uuid.New().String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	respBody := readBody(resp.Body)
	assert.Contains(t, respBody, "not found")
}

func TestComponentHandler_HandleComponentInteractionV2_ServiceError(t *testing.T) {
	componentSvc := &mockComponentHandlerComponentService{}
	handler := &ComponentHandler{componentService: componentSvc}
	app := setupComponentHandlerTestApp(t, handler)

	componentSvc.getMessageComponentsFunc = func(ctx context.Context, msgID uuid.UUID) ([]*models.MessageComponent, error) {
		return []*models.MessageComponent{{ID: uuid.New(), CustomID: "btn", Type: models.ComponentTypeButton}}, nil
	}
	componentSvc.handleInteractionFunc = func(ctx context.Context, uid, chID, msgID, compID uuid.UUID, custID string, vals []string) (*models.ComponentInteraction, error) {
		return nil, errors.New("service error")
	}

	body := map[string]interface{}{"message_id": uuid.New().String(), "channel_id": uuid.New().String(), "custom_id": "btn"}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/interactions/components", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", uuid.New().String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

// =============================================================================
// Tests for GetMessageComponents
// =============================================================================

func TestComponentHandler_GetMessageComponents_Success(t *testing.T) {
	componentSvc := &mockComponentHandlerComponentService{}
	messageSvc := &mockComponentHandlerMessageService{}
	handler := &ComponentHandler{componentService: componentSvc, messageService: messageSvc}
	app := setupComponentHandlerTestApp(t, handler)

	userID := uuid.New()
	messageID := uuid.New()
	channelID := uuid.New()

	messageSvc.getMessageFunc = func(ctx context.Context, msgID, uID uuid.UUID) (*models.Message, error) {
		return &models.Message{ID: msgID, ChannelID: channelID, AuthorID: uID}, nil
	}
	componentSvc.getMessageComponentsFunc = func(ctx context.Context, msgID uuid.UUID) ([]*models.MessageComponent, error) {
		return []*models.MessageComponent{
			{ID: uuid.New(), CustomID: "btn_1", Type: models.ComponentTypeButton, Label: "Click me"},
			{ID: uuid.New(), CustomID: "btn_2", Type: models.ComponentTypeButton, Label: "Or me"},
		}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/channels/"+channelID.String()+"/messages/"+messageID.String()+"/components", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var components []*models.MessageComponent
	err = json.NewDecoder(resp.Body).Decode(&components)
	assert.NoError(t, err)
	assert.Len(t, components, 2)
}

func TestComponentHandler_GetMessageComponents_EmptyComponents(t *testing.T) {
	componentSvc := &mockComponentHandlerComponentService{}
	messageSvc := &mockComponentHandlerMessageService{}
	handler := &ComponentHandler{componentService: componentSvc, messageService: messageSvc}
	app := setupComponentHandlerTestApp(t, handler)

	userID := uuid.New()
	messageID := uuid.New()
	channelID := uuid.New()

	messageSvc.getMessageFunc = func(ctx context.Context, msgID, uID uuid.UUID) (*models.Message, error) {
		return &models.Message{ID: msgID, ChannelID: channelID, AuthorID: uID}, nil
	}
	componentSvc.getMessageComponentsFunc = func(ctx context.Context, msgID uuid.UUID) ([]*models.MessageComponent, error) {
		return []*models.MessageComponent{}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/channels/"+channelID.String()+"/messages/"+messageID.String()+"/components", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var components []*models.MessageComponent
	err = json.NewDecoder(resp.Body).Decode(&components)
	assert.NoError(t, err)
	assert.Len(t, components, 0)
}

func TestComponentHandler_GetMessageComponents_NilComponents(t *testing.T) {
	componentSvc := &mockComponentHandlerComponentService{}
	messageSvc := &mockComponentHandlerMessageService{}
	handler := &ComponentHandler{componentService: componentSvc, messageService: messageSvc}
	app := setupComponentHandlerTestApp(t, handler)

	userID := uuid.New()
	messageID := uuid.New()
	channelID := uuid.New()

	messageSvc.getMessageFunc = func(ctx context.Context, msgID, uID uuid.UUID) (*models.Message, error) {
		return &models.Message{ID: msgID, ChannelID: channelID, AuthorID: uID}, nil
	}
	componentSvc.getMessageComponentsFunc = func(ctx context.Context, msgID uuid.UUID) ([]*models.MessageComponent, error) {
		return nil, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/channels/"+channelID.String()+"/messages/"+messageID.String()+"/components", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestComponentHandler_GetMessageComponents_InvalidMessageID(t *testing.T) {
	handler := &ComponentHandler{componentService: &mockComponentHandlerComponentService{}, messageService: &mockComponentHandlerMessageService{}}
	app := setupComponentHandlerTestApp(t, handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/channels/"+uuid.New().String()+"/messages/not-a-uuid/components", nil)
	req.Header.Set("X-User-ID", uuid.New().String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	respBody := readBody(resp.Body)
	assert.Contains(t, respBody, "message ID")
}

func TestComponentHandler_GetMessageComponents_MessageNotFound(t *testing.T) {
	messageSvc := &mockComponentHandlerMessageService{}
	handler := &ComponentHandler{messageService: messageSvc}
	app := setupComponentHandlerTestApp(t, handler)

	messageSvc.getMessageFunc = func(ctx context.Context, msgID, uID uuid.UUID) (*models.Message, error) {
		return nil, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/channels/"+uuid.New().String()+"/messages/"+uuid.New().String()+"/components", nil)
	req.Header.Set("X-User-ID", uuid.New().String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestComponentHandler_GetMessageComponents_ServiceError(t *testing.T) {
	componentSvc := &mockComponentHandlerComponentService{}
	messageSvc := &mockComponentHandlerMessageService{}
	handler := &ComponentHandler{componentService: componentSvc, messageService: messageSvc}
	app := setupComponentHandlerTestApp(t, handler)

	channelID := uuid.New()
	messageID := uuid.New()

	messageSvc.getMessageFunc = func(ctx context.Context, msgID, uID uuid.UUID) (*models.Message, error) {
		return &models.Message{ID: msgID, ChannelID: channelID, AuthorID: uID}, nil
	}
	componentSvc.getMessageComponentsFunc = func(ctx context.Context, msgID uuid.UUID) ([]*models.MessageComponent, error) {
		return nil, errors.New("database error")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/channels/"+channelID.String()+"/messages/"+messageID.String()+"/components", nil)
	req.Header.Set("X-User-ID", uuid.New().String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

// =============================================================================
// Tests for UpdateMessageComponents
// =============================================================================

func TestComponentHandler_UpdateMessageComponents_AsAuthor_Success(t *testing.T) {
	componentSvc := &mockComponentHandlerComponentService{}
	messageSvc := &mockComponentHandlerMessageService{}
	handler := &ComponentHandler{componentService: componentSvc, messageService: messageSvc}
	app := setupComponentHandlerTestApp(t, handler)

	userID := uuid.New()
	messageID := uuid.New()
	channelID := uuid.New()

	messageSvc.getMessageFunc = func(ctx context.Context, msgID, uID uuid.UUID) (*models.Message, error) {
		return &models.Message{ID: msgID, ChannelID: channelID, AuthorID: userID}, nil
	}
	componentSvc.updateMessageComponentsFunc = func(ctx context.Context, msgID uuid.UUID, components []*models.MessageComponent) ([]*models.MessageComponent, error) {
		return components, nil
	}

	body := models.UpdateComponentsRequest{Components: []models.CreateComponentRequest{{Type: models.ComponentTypeButton, Style: models.ButtonStylePrimary, Label: "New Button", CustomID: "new_btn"}}}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/channels/"+channelID.String()+"/messages/"+messageID.String()+"/components", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var updated []*models.MessageComponent
	err = json.NewDecoder(resp.Body).Decode(&updated)
	assert.NoError(t, err)
	assert.Len(t, updated, 1)
	assert.Equal(t, "New Button", updated[0].Label)
}

func TestComponentHandler_UpdateMessageComponents_WithManageMessagesPermission(t *testing.T) {
	componentSvc := &mockComponentHandlerComponentService{}
	messageSvc := &mockComponentHandlerMessageService{}
	channelSvc := &mockComponentHandlerChannelService{}
	permSvc := &mockComponentHandlerPermissionService{}
	handler := &ComponentHandler{componentService: componentSvc, messageService: messageSvc, channelService: channelSvc, permissionService: permSvc}
	app := setupComponentHandlerTestApp(t, handler)

	userID := uuid.New()
	authorID := uuid.New()
	messageID := uuid.New()
	channelID := uuid.New()

	messageSvc.getMessageFunc = func(ctx context.Context, msgID, uID uuid.UUID) (*models.Message, error) {
		return &models.Message{ID: msgID, ChannelID: channelID, AuthorID: authorID}, nil
	}
	channelSvc.getChannelFunc = func(ctx context.Context, chID uuid.UUID) (*models.Channel, error) {
		return &models.Channel{ID: channelID}, nil
	}
	permSvc.getChannelPermissionsFunc = func(ctx context.Context, channel *models.Channel, uID uuid.UUID) (int64, error) {
		return models.PermManageMessages, nil
	}
	componentSvc.updateMessageComponentsFunc = func(ctx context.Context, msgID uuid.UUID, components []*models.MessageComponent) ([]*models.MessageComponent, error) {
		return components, nil
	}

	body := models.UpdateComponentsRequest{Components: []models.CreateComponentRequest{{Type: models.ComponentTypeButton, CustomID: "btn"}}}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/channels/"+channelID.String()+"/messages/"+messageID.String()+"/components", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestComponentHandler_UpdateMessageComponents_NotAuthorNoPermission(t *testing.T) {
	componentSvc := &mockComponentHandlerComponentService{}
	messageSvc := &mockComponentHandlerMessageService{}
	channelSvc := &mockComponentHandlerChannelService{}
	permSvc := &mockComponentHandlerPermissionService{}
	handler := &ComponentHandler{componentService: componentSvc, messageService: messageSvc, channelService: channelSvc, permissionService: permSvc}
	app := setupComponentHandlerTestApp(t, handler)

	userID := uuid.New()
	authorID := uuid.New()
	messageID := uuid.New()
	channelID := uuid.New()

	messageSvc.getMessageFunc = func(ctx context.Context, msgID, uID uuid.UUID) (*models.Message, error) {
		return &models.Message{ID: msgID, ChannelID: channelID, AuthorID: authorID}, nil
	}
	channelSvc.getChannelFunc = func(ctx context.Context, chID uuid.UUID) (*models.Channel, error) {
		return &models.Channel{ID: channelID}, nil
	}
	permSvc.getChannelPermissionsFunc = func(ctx context.Context, channel *models.Channel, uID uuid.UUID) (int64, error) {
		return 0, nil
	}

	body := models.UpdateComponentsRequest{Components: []models.CreateComponentRequest{{Type: models.ComponentTypeButton, CustomID: "btn"}}}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/channels/"+channelID.String()+"/messages/"+messageID.String()+"/components", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
	respBody := readBody(resp.Body)
	assert.Contains(t, respBody, "permission")
}

func TestComponentHandler_UpdateMessageComponents_InvalidJSON(t *testing.T) {
	messageSvc := &mockComponentHandlerMessageService{}
	handler := &ComponentHandler{messageService: messageSvc}
	app := setupComponentHandlerTestApp(t, handler)

	userID := uuid.New()
	messageID := uuid.New()
	channelID := uuid.New()

	messageSvc.getMessageFunc = func(ctx context.Context, msgID, uID uuid.UUID) (*models.Message, error) {
		return &models.Message{ID: msgID, ChannelID: channelID, AuthorID: userID}, nil
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/channels/"+channelID.String()+"/messages/"+messageID.String()+"/components", bytes.NewReader([]byte(`{invalid}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestComponentHandler_UpdateMessageComponents_InvalidMessageID(t *testing.T) {
	handler := &ComponentHandler{messageService: &mockComponentHandlerMessageService{}}
	app := setupComponentHandlerTestApp(t, handler)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/channels/"+uuid.New().String()+"/messages/not-a-uuid/components", nil)
	req.Header.Set("X-User-ID", uuid.New().String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	respBody := readBody(resp.Body)
	assert.Contains(t, respBody, "message ID")
}

func TestComponentHandler_UpdateMessageComponents_MessageNotFound(t *testing.T) {
	messageSvc := &mockComponentHandlerMessageService{}
	handler := &ComponentHandler{messageService: messageSvc}
	app := setupComponentHandlerTestApp(t, handler)

	messageSvc.getMessageFunc = func(ctx context.Context, msgID, uID uuid.UUID) (*models.Message, error) {
		return nil, nil
	}

	body := models.UpdateComponentsRequest{Components: []models.CreateComponentRequest{{Type: models.ComponentTypeButton}}}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/channels/"+uuid.New().String()+"/messages/"+uuid.New().String()+"/components", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", uuid.New().String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestComponentHandler_UpdateMessageComponents_ChannelNotFound(t *testing.T) {
	componentSvc := &mockComponentHandlerComponentService{}
	messageSvc := &mockComponentHandlerMessageService{}
	channelSvc := &mockComponentHandlerChannelService{}
	handler := &ComponentHandler{componentService: componentSvc, messageService: messageSvc, channelService: channelSvc}
	app := setupComponentHandlerTestApp(t, handler)

	userID := uuid.New()
	authorID := uuid.New()
	messageID := uuid.New()
	channelID := uuid.New()

	messageSvc.getMessageFunc = func(ctx context.Context, msgID, uID uuid.UUID) (*models.Message, error) {
		return &models.Message{ID: msgID, ChannelID: channelID, AuthorID: authorID}, nil
	}
	channelSvc.getChannelFunc = func(ctx context.Context, chID uuid.UUID) (*models.Channel, error) {
		return nil, nil
	}

	body := models.UpdateComponentsRequest{Components: []models.CreateComponentRequest{{Type: models.ComponentTypeButton}}}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/channels/"+channelID.String()+"/messages/"+messageID.String()+"/components", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

// =============================================================================
// Tests for RemoveAllComponents
// =============================================================================

func TestComponentHandler_RemoveAllComponents_AsAuthor_Success(t *testing.T) {
	componentSvc := &mockComponentHandlerComponentService{}
	messageSvc := &mockComponentHandlerMessageService{}
	handler := &ComponentHandler{componentService: componentSvc, messageService: messageSvc}
	app := setupComponentHandlerTestApp(t, handler)

	userID := uuid.New()
	messageID := uuid.New()
	channelID := uuid.New()

	messageSvc.getMessageFunc = func(ctx context.Context, msgID, uID uuid.UUID) (*models.Message, error) {
		return &models.Message{ID: msgID, ChannelID: channelID, AuthorID: userID}, nil
	}
	componentSvc.removeAllComponentsFunc = func(ctx context.Context, msgID uuid.UUID) error {
		return nil
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/channels/"+channelID.String()+"/messages/"+messageID.String()+"/components", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)
}

func TestComponentHandler_RemoveAllComponents_NotAuthor(t *testing.T) {
	messageSvc := &mockComponentHandlerMessageService{}
	handler := &ComponentHandler{messageService: messageSvc}
	app := setupComponentHandlerTestApp(t, handler)

	userID := uuid.New()
	authorID := uuid.New()
	messageID := uuid.New()
	channelID := uuid.New()

	messageSvc.getMessageFunc = func(ctx context.Context, msgID, uID uuid.UUID) (*models.Message, error) {
		return &models.Message{ID: msgID, ChannelID: channelID, AuthorID: authorID}, nil
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/channels/"+channelID.String()+"/messages/"+messageID.String()+"/components", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
	respBody := readBody(resp.Body)
	assert.Contains(t, respBody, "author")
}

func TestComponentHandler_RemoveAllComponents_InvalidMessageID(t *testing.T) {
	handler := &ComponentHandler{messageService: &mockComponentHandlerMessageService{}}
	app := setupComponentHandlerTestApp(t, handler)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/channels/"+uuid.New().String()+"/messages/not-a-uuid/components", nil)
	req.Header.Set("X-User-ID", uuid.New().String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	respBody := readBody(resp.Body)
	assert.Contains(t, respBody, "message ID")
}

func TestComponentHandler_RemoveAllComponents_MessageNotFound(t *testing.T) {
	messageSvc := &mockComponentHandlerMessageService{}
	handler := &ComponentHandler{messageService: messageSvc}
	app := setupComponentHandlerTestApp(t, handler)

	messageSvc.getMessageFunc = func(ctx context.Context, msgID, uID uuid.UUID) (*models.Message, error) {
		return nil, nil
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/channels/"+uuid.New().String()+"/messages/"+uuid.New().String()+"/components", nil)
	req.Header.Set("X-User-ID", uuid.New().String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestComponentHandler_RemoveAllComponents_ServiceError(t *testing.T) {
	componentSvc := &mockComponentHandlerComponentService{}
	messageSvc := &mockComponentHandlerMessageService{}
	handler := &ComponentHandler{componentService: componentSvc, messageService: messageSvc}
	app := setupComponentHandlerTestApp(t, handler)

	userID := uuid.New()
	messageID := uuid.New()
	channelID := uuid.New()

	messageSvc.getMessageFunc = func(ctx context.Context, msgID, uID uuid.UUID) (*models.Message, error) {
		return &models.Message{ID: msgID, ChannelID: channelID, AuthorID: userID}, nil
	}
	componentSvc.removeAllComponentsFunc = func(ctx context.Context, msgID uuid.UUID) error {
		return errors.New("database error")
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/channels/"+channelID.String()+"/messages/"+messageID.String()+"/components", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}
