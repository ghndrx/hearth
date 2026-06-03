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

// MockEmbedService mocks the EmbedServiceInterface for testing
type MockEmbedService struct {
	mock.Mock
}

func (m *MockEmbedService) FetchURLMetadata(ctx context.Context, rawURL string) (*models.LinkPreviewResponse, error) {
	args := m.Called(ctx, rawURL)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.LinkPreviewResponse), args.Error(1)
}

func (m *MockEmbedService) CreateTemplate(ctx context.Context, userID uuid.UUID, req *models.CreateEmbedTemplateRequest) (*models.EmbedTemplate, error) {
	args := m.Called(ctx, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.EmbedTemplate), args.Error(1)
}

func (m *MockEmbedService) GetTemplates(ctx context.Context, userID uuid.UUID) ([]models.EmbedTemplate, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.EmbedTemplate), args.Error(1)
}

func (m *MockEmbedService) GetTemplate(ctx context.Context, userID, templateID uuid.UUID) (*models.EmbedTemplate, error) {
	args := m.Called(ctx, userID, templateID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.EmbedTemplate), args.Error(1)
}

func (m *MockEmbedService) UpdateTemplate(ctx context.Context, userID, templateID uuid.UUID, req *models.UpdateEmbedTemplateRequest) (*models.EmbedTemplate, error) {
	args := m.Called(ctx, userID, templateID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.EmbedTemplate), args.Error(1)
}

func (m *MockEmbedService) DeleteTemplate(ctx context.Context, userID, templateID uuid.UUID) error {
	args := m.Called(ctx, userID, templateID)
	return args.Error(0)
}

// testEmbedHandler creates a test embed handler with mocks
type testEmbedHandler struct {
	handler     *EmbedHandler
	embedService *MockEmbedService
	app         *fiber.App
	userID      uuid.UUID
}

func newTestEmbedHandler(tb testing.TB) *testEmbedHandler {
	embedService := new(MockEmbedService)
	handler := NewEmbedHandler(embedService)

	app := fiber.New()
	tb.Cleanup(func() { _ = app.Shutdown() })
	userID := uuid.New()

	// Add middleware to set userID in locals
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", userID)
		return c.Next()
	})

	// Setup routes
	app.Get("/embeds/fetch", handler.FetchURLMetadata)
	app.Post("/embeds/templates", handler.CreateTemplate)
	app.Get("/embeds/templates", handler.ListTemplates)
	app.Get("/embeds/templates/:id", handler.GetTemplate)
	app.Put("/embeds/templates/:id", handler.UpdateTemplate)
	app.Delete("/embeds/templates/:id", handler.DeleteTemplate)

	return &testEmbedHandler{
		handler:     handler,
		embedService: embedService,
		app:         app,
		userID:      userID,
	}
}

// ---------------------------------------------------------------------------
// GetEmbedPreview (FetchURLMetadata)
// ---------------------------------------------------------------------------

func TestEmbedHandler_FetchURLMetadata_Success(t *testing.T) {
	th := newTestEmbedHandler(t)

	preview := &models.LinkPreviewResponse{
		ID:    uuid.New(),
		URL:   "https://example.com",
		Title: &[]string{"Example"}[0],
	}

	th.embedService.On("FetchURLMetadata", mock.Anything, "https://example.com").Return(preview, nil)

	req := httptest.NewRequest(http.MethodGet, "/embeds/fetch?url=https://example.com", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result models.LinkPreviewResponse
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, preview.ID, result.ID)
	assert.Equal(t, "https://example.com", result.URL)

	th.embedService.AssertExpectations(t)
}

func TestEmbedHandler_FetchURLMetadata_InvalidURLEmpty(t *testing.T) {
	th := newTestEmbedHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/embeds/fetch", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result ErrorResponse
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "validation_error", result.Error)
}

func TestEmbedHandler_FetchURLMetadata_InvalidURLFormat(t *testing.T) {
	th := newTestEmbedHandler(t)

	th.embedService.On("FetchURLMetadata", mock.Anything, "not-a-url").Return(nil, services.ErrInvalidURL)

	req := httptest.NewRequest(http.MethodGet, "/embeds/fetch?url=not-a-url", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result ErrorResponse
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "validation_error", result.Error)

	th.embedService.AssertExpectations(t)
}

func TestEmbedHandler_FetchURLMetadata_UnreachableURL(t *testing.T) {
	th := newTestEmbedHandler(t)

	th.embedService.On("FetchURLMetadata", mock.Anything, "https://unreachable.example.com").Return(nil, services.ErrUnreachableURL)

	req := httptest.NewRequest(http.MethodGet, "/embeds/fetch?url=https://unreachable.example.com", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadGateway, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "URL is unreachable", result["error"])

	th.embedService.AssertExpectations(t)
}

func TestEmbedHandler_FetchURLMetadata_GenericError(t *testing.T) {
	th := newTestEmbedHandler(t)

	th.embedService.On("FetchURLMetadata", mock.Anything, "https://example.com").Return(nil, assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/embeds/fetch?url=https://example.com", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	th.embedService.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// CreateEmbed (CreateTemplate)
// ---------------------------------------------------------------------------

func TestEmbedHandler_CreateTemplate_Success(t *testing.T) {
	th := newTestEmbedHandler(t)

	templateID := uuid.New()
	name := "My Embed"
	template := &models.EmbedTemplate{
		ID:        templateID,
		UserID:    th.userID,
		Name:      name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	th.embedService.On("CreateTemplate", mock.Anything, th.userID, mock.MatchedBy(func(req *models.CreateEmbedTemplateRequest) bool {
		return req.Name == name
	})).Return(template, nil)

	body := `{"name":"My Embed"}`
	req := httptest.NewRequest(http.MethodPost, "/embeds/templates", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result models.EmbedTemplateResponse
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, templateID, result.ID)
	assert.Equal(t, name, result.Name)

	th.embedService.AssertExpectations(t)
}

func TestEmbedHandler_CreateTemplate_ValidationError(t *testing.T) {
	th := newTestEmbedHandler(t)

	body := `{"name":""}`
	req := httptest.NewRequest(http.MethodPost, "/embeds/templates", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result ErrorResponse
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "validation_error", result.Error)
}

func TestEmbedHandler_CreateTemplate_Unauthorized(t *testing.T) {
	// Create app without userID middleware
	embedService := new(MockEmbedService)
	handler := NewEmbedHandler(embedService)
	app := fiber.New()
	t.Cleanup(func() { _ = app.Shutdown() })
	app.Post("/embeds/templates", handler.CreateTemplate)

	body := `{"name":"My Embed"}`
	req := httptest.NewRequest(http.MethodPost, "/embeds/templates", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// ---------------------------------------------------------------------------
// GetEmbed (GetTemplate)
// ---------------------------------------------------------------------------

func TestEmbedHandler_GetTemplate_Success(t *testing.T) {
	th := newTestEmbedHandler(t)

	templateID := uuid.New()
	template := &models.EmbedTemplate{
		ID:        templateID,
		UserID:    th.userID,
		Name:      "My Embed",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	th.embedService.On("GetTemplate", mock.Anything, th.userID, templateID).Return(template, nil)

	req := httptest.NewRequest(http.MethodGet, "/embeds/templates/"+templateID.String(), nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result models.EmbedTemplateResponse
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, templateID, result.ID)

	th.embedService.AssertExpectations(t)
}

func TestEmbedHandler_GetTemplate_NotFound(t *testing.T) {
	th := newTestEmbedHandler(t)

	templateID := uuid.New()

	th.embedService.On("GetTemplate", mock.Anything, th.userID, templateID).Return(nil, services.ErrTemplateNotFound)

	req := httptest.NewRequest(http.MethodGet, "/embeds/templates/"+templateID.String(), nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "template not found", result["error"])

	th.embedService.AssertExpectations(t)
}

func TestEmbedHandler_GetTemplate_InvalidUUID(t *testing.T) {
	th := newTestEmbedHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/embeds/templates/invalid-uuid", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result ErrorResponse
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "validation_error", result.Error)
}

// ---------------------------------------------------------------------------
// UpdateEmbed (UpdateTemplate)
// ---------------------------------------------------------------------------

func TestEmbedHandler_UpdateTemplate_Success(t *testing.T) {
	th := newTestEmbedHandler(t)

	templateID := uuid.New()
	newName := "Updated Embed"
	template := &models.EmbedTemplate{
		ID:        templateID,
		UserID:    th.userID,
		Name:      newName,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	th.embedService.On("UpdateTemplate", mock.Anything, th.userID, templateID, mock.MatchedBy(func(req *models.UpdateEmbedTemplateRequest) bool {
		return req.Name != nil && *req.Name == newName
	})).Return(template, nil)

	body := `{"name":"Updated Embed"}`
	req := httptest.NewRequest(http.MethodPut, "/embeds/templates/"+templateID.String(), bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result models.EmbedTemplateResponse
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, templateID, result.ID)
	assert.Equal(t, newName, result.Name)

	th.embedService.AssertExpectations(t)
}

func TestEmbedHandler_UpdateTemplate_NotFound(t *testing.T) {
	th := newTestEmbedHandler(t)

	templateID := uuid.New()

	th.embedService.On("UpdateTemplate", mock.Anything, th.userID, templateID, mock.Anything).Return(nil, services.ErrTemplateNotFound)

	body := `{"name":"Updated Embed"}`
	req := httptest.NewRequest(http.MethodPut, "/embeds/templates/"+templateID.String(), bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "template not found", result["error"])

	th.embedService.AssertExpectations(t)
}

func TestEmbedHandler_UpdateTemplate_InvalidUUID(t *testing.T) {
	th := newTestEmbedHandler(t)

	body := `{"name":"Updated Embed"}`
	req := httptest.NewRequest(http.MethodPut, "/embeds/templates/invalid-uuid", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result ErrorResponse
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "validation_error", result.Error)
}

// ---------------------------------------------------------------------------
// DeleteEmbed (DeleteTemplate)
// ---------------------------------------------------------------------------

func TestEmbedHandler_DeleteTemplate_Success(t *testing.T) {
	th := newTestEmbedHandler(t)

	templateID := uuid.New()

	th.embedService.On("DeleteTemplate", mock.Anything, th.userID, templateID).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/embeds/templates/"+templateID.String(), nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	th.embedService.AssertExpectations(t)
}

func TestEmbedHandler_DeleteTemplate_NotFound(t *testing.T) {
	th := newTestEmbedHandler(t)

	templateID := uuid.New()

	th.embedService.On("DeleteTemplate", mock.Anything, th.userID, templateID).Return(services.ErrTemplateNotFound)

	req := httptest.NewRequest(http.MethodDelete, "/embeds/templates/"+templateID.String(), nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "template not found", result["error"])

	th.embedService.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// ListEmbeds (ListTemplates)
// ---------------------------------------------------------------------------

func TestEmbedHandler_ListTemplates_Success(t *testing.T) {
	th := newTestEmbedHandler(t)

	templates := []models.EmbedTemplate{
		{ID: uuid.New(), UserID: th.userID, Name: "Embed 1"},
		{ID: uuid.New(), UserID: th.userID, Name: "Embed 2"},
	}

	th.embedService.On("GetTemplates", mock.Anything, th.userID).Return(templates, nil)

	req := httptest.NewRequest(http.MethodGet, "/embeds/templates", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result []models.EmbedTemplateResponse
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Len(t, result, 2)

	th.embedService.AssertExpectations(t)
}

func TestEmbedHandler_ListTemplates_EmptyList(t *testing.T) {
	th := newTestEmbedHandler(t)

	templates := []models.EmbedTemplate{}

	th.embedService.On("GetTemplates", mock.Anything, th.userID).Return(templates, nil)

	req := httptest.NewRequest(http.MethodGet, "/embeds/templates", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result []models.EmbedTemplateResponse
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Len(t, result, 0)

	th.embedService.AssertExpectations(t)
}


