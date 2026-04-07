package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hearth/internal/models"
	"hearth/internal/services"
)

// --- mock embed service ---

type mockEmbedService struct {
	fetchURLMetadataFunc func(ctx context.Context, rawURL string) (*models.LinkPreviewResponse, error)
	createTemplateFunc  func(ctx context.Context, userID uuid.UUID, req *models.CreateEmbedTemplateRequest) (*models.EmbedTemplate, error)
	getTemplatesFunc     func(ctx context.Context, userID uuid.UUID) ([]models.EmbedTemplate, error)
	getTemplateFunc      func(ctx context.Context, userID, templateID uuid.UUID) (*models.EmbedTemplate, error)
	updateTemplateFunc   func(ctx context.Context, userID, templateID uuid.UUID, req *models.UpdateEmbedTemplateRequest) (*models.EmbedTemplate, error)
	deleteTemplateFunc   func(ctx context.Context, userID, templateID uuid.UUID) error
}

func (m *mockEmbedService) FetchURLMetadata(ctx context.Context, rawURL string) (*models.LinkPreviewResponse, error) {
	if m.fetchURLMetadataFunc != nil {
		return m.fetchURLMetadataFunc(ctx, rawURL)
	}
	title := "Example Site"
	return &models.LinkPreviewResponse{
		ID:          uuid.New(),
		URL:         rawURL,
		Title:       &title,
		Description: strPtr("An example website"),
		ImageURL:    strPtr("https://example.com/image.png"),
		SiteName:    strPtr("Example"),
		Type:        "website",
	}, nil
}

func (m *mockEmbedService) CreateTemplate(ctx context.Context, userID uuid.UUID, req *models.CreateEmbedTemplateRequest) (*models.EmbedTemplate, error) {
	if m.createTemplateFunc != nil {
		return m.createTemplateFunc(ctx, userID, req)
	}
	title := req.Title
	return &models.EmbedTemplate{
		ID:          uuid.New(),
		UserID:      userID,
		Name:        req.Name,
		Title:       title,
		Description: req.Description,
		URL:         req.URL,
		Color:       req.Color,
		AuthorName:  req.AuthorName,
		AuthorURL:   req.AuthorURL,
		AuthorIcon:  req.AuthorIcon,
		FooterText:  req.FooterText,
		FooterIcon:  req.FooterIcon,
		ImageURL:    req.ImageURL,
		ThumbnailURL: req.ThumbnailURL,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}

func (m *mockEmbedService) GetTemplates(ctx context.Context, userID uuid.UUID) ([]models.EmbedTemplate, error) {
	if m.getTemplatesFunc != nil {
		return m.getTemplatesFunc(ctx, userID)
	}
	return []models.EmbedTemplate{}, nil
}

func (m *mockEmbedService) GetTemplate(ctx context.Context, userID, templateID uuid.UUID) (*models.EmbedTemplate, error) {
	if m.getTemplateFunc != nil {
		return m.getTemplateFunc(ctx, userID, templateID)
	}
	return nil, services.ErrTemplateNotFound
}

func (m *mockEmbedService) UpdateTemplate(ctx context.Context, userID, templateID uuid.UUID, req *models.UpdateEmbedTemplateRequest) (*models.EmbedTemplate, error) {
	if m.updateTemplateFunc != nil {
		return m.updateTemplateFunc(ctx, userID, templateID, req)
	}
	return nil, services.ErrTemplateNotFound
}

func (m *mockEmbedService) DeleteTemplate(ctx context.Context, userID, templateID uuid.UUID) error {
	if m.deleteTemplateFunc != nil {
		return m.deleteTemplateFunc(ctx, userID, templateID)
	}
	return nil
}

// --- test app setup ---

func setupEmbedTestApp(svc *mockEmbedService) *fiber.App {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		if uidStr := c.Get("X-Test-User-ID"); uidStr != "" {
			if uid, err := uuid.Parse(uidStr); err == nil {
				c.Locals("userID", uid)
			}
		}
		return c.Next()
	})

	h := NewEmbedHandler(svc)

	app.Get("/embeds/fetch", h.FetchURLMetadata)
	app.Post("/embeds/templates", h.CreateTemplate)
	app.Get("/embeds/templates", h.ListTemplates)
	app.Get("/embeds/templates/:id", h.GetTemplate)
	app.Put("/embeds/templates/:id", h.UpdateTemplate)
	app.Delete("/embeds/templates/:id", h.DeleteTemplate)

	return app
}

// --- tests ---

func TestEmbedHandler_FetchURLMetadata(t *testing.T) {
	validURL := "https://example.com"
	invalidURL := "not-a-url"
	unreachableURL := "https://192.0.2.1/" // TEST-NET-1, never reachable

	t.Run("success", func(t *testing.T) {
		svc := &mockEmbedService{
			fetchURLMetadataFunc: func(ctx context.Context, rawURL string) (*models.LinkPreviewResponse, error) {
				assert.Equal(t, validURL, rawURL)
				title := "Example"
				return &models.LinkPreviewResponse{
					ID:    uuid.New(),
					URL:   rawURL,
					Title: &title,
					Type:  "website",
				}, nil
			},
		}
		app := setupEmbedTestApp(svc)

		req := httptest.NewRequest("GET", "/embeds/fetch?url="+validURL, nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var body models.LinkPreviewResponse
		err = json.NewDecoder(resp.Body).Decode(&body)
		require.NoError(t, err)
		assert.Equal(t, validURL, body.URL)
		assert.Equal(t, "website", body.Type)
	})

	t.Run("missing_url_param", func(t *testing.T) {
		svc := &mockEmbedService{}
		app := setupEmbedTestApp(svc)

		req := httptest.NewRequest("GET", "/embeds/fetch", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("invalid_url_format", func(t *testing.T) {
		svc := &mockEmbedService{
			fetchURLMetadataFunc: func(ctx context.Context, rawURL string) (*models.LinkPreviewResponse, error) {
				return nil, services.ErrInvalidURL
			},
		}
		app := setupEmbedTestApp(svc)

		req := httptest.NewRequest("GET", "/embeds/fetch?url="+invalidURL, nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("unreachable_url", func(t *testing.T) {
		svc := &mockEmbedService{
			fetchURLMetadataFunc: func(ctx context.Context, rawURL string) (*models.LinkPreviewResponse, error) {
				return nil, services.ErrUnreachableURL
			},
		}
		app := setupEmbedTestApp(svc)

		req := httptest.NewRequest("GET", "/embeds/fetch?url="+unreachableURL, nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadGateway, resp.StatusCode)
	})

	t.Run("service_error", func(t *testing.T) {
		svc := &mockEmbedService{
			fetchURLMetadataFunc: func(ctx context.Context, rawURL string) (*models.LinkPreviewResponse, error) {
				return nil, errors.New("unexpected error")
			},
		}
		app := setupEmbedTestApp(svc)

		req := httptest.NewRequest("GET", "/embeds/fetch?url="+validURL, nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	})
}

func TestEmbedHandler_CreateTemplate(t *testing.T) {
	testUserID := uuid.New()

	t.Run("success", func(t *testing.T) {
		svc := &mockEmbedService{
			createTemplateFunc: func(ctx context.Context, userID uuid.UUID, req *models.CreateEmbedTemplateRequest) (*models.EmbedTemplate, error) {
				assert.Equal(t, testUserID, userID)
				assert.Equal(t, "My Embed", req.Name)
				title := "Test Title"
				return &models.EmbedTemplate{
					ID:          uuid.New(),
					UserID:      userID,
					Name:        req.Name,
					Title:       &title,
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}, nil
			},
		}
		app := setupEmbedTestApp(svc)

		body := map[string]interface{}{
			"name":        "My Embed",
			"title":       "Test Title",
			"description": "A test description",
		}
		bodyBytes, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", "/embeds/templates", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-User-ID", testUserID.String())

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

		var result models.EmbedTemplateResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.Equal(t, "My Embed", result.Name)
	})

	t.Run("unauthorized_no_user", func(t *testing.T) {
		svc := &mockEmbedService{}
		app := setupEmbedTestApp(svc)

		body := map[string]interface{}{"name": "My Embed"}
		bodyBytes, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", "/embeds/templates", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("invalid_json", func(t *testing.T) {
		svc := &mockEmbedService{}
		app := setupEmbedTestApp(svc)

		req := httptest.NewRequest("POST", "/embeds/templates", bytes.NewReader([]byte("not json")))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-User-ID", testUserID.String())

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("missing_name", func(t *testing.T) {
		svc := &mockEmbedService{}
		app := setupEmbedTestApp(svc)

		body := map[string]interface{}{"title": "Test Title"}
		bodyBytes, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", "/embeds/templates", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-User-ID", testUserID.String())

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("service_error", func(t *testing.T) {
		svc := &mockEmbedService{
			createTemplateFunc: func(ctx context.Context, userID uuid.UUID, req *models.CreateEmbedTemplateRequest) (*models.EmbedTemplate, error) {
				return nil, errors.New("database error")
			},
		}
		app := setupEmbedTestApp(svc)

		body := map[string]interface{}{"name": "My Embed"}
		bodyBytes, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", "/embeds/templates", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-User-ID", testUserID.String())

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	})
}

func TestEmbedHandler_ListTemplates(t *testing.T) {
	testUserID := uuid.New()

	t.Run("success_empty", func(t *testing.T) {
		svc := &mockEmbedService{
			getTemplatesFunc: func(ctx context.Context, userID uuid.UUID) ([]models.EmbedTemplate, error) {
				assert.Equal(t, testUserID, userID)
				return []models.EmbedTemplate{}, nil
			},
		}
		app := setupEmbedTestApp(svc)

		req := httptest.NewRequest("GET", "/embeds/templates", nil)
		req.Header.Set("X-Test-User-ID", testUserID.String())

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var result []*models.EmbedTemplateResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("success_with_templates", func(t *testing.T) {
		templateID := uuid.New()
		title := "Test Title"
		svc := &mockEmbedService{
			getTemplatesFunc: func(ctx context.Context, userID uuid.UUID) ([]models.EmbedTemplate, error) {
				assert.Equal(t, testUserID, userID)
				return []models.EmbedTemplate{
					{
						ID:     templateID,
						UserID: userID,
						Name:   "Template 1",
						Title:  &title,
					},
				}, nil
			},
		}
		app := setupEmbedTestApp(svc)

		req := httptest.NewRequest("GET", "/embeds/templates", nil)
		req.Header.Set("X-Test-User-ID", testUserID.String())

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var result []*models.EmbedTemplateResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "Template 1", result[0].Name)
	})

	t.Run("unauthorized_no_user", func(t *testing.T) {
		svc := &mockEmbedService{}
		app := setupEmbedTestApp(svc)

		req := httptest.NewRequest("GET", "/embeds/templates", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("service_error", func(t *testing.T) {
		svc := &mockEmbedService{
			getTemplatesFunc: func(ctx context.Context, userID uuid.UUID) ([]models.EmbedTemplate, error) {
				return nil, errors.New("database error")
			},
		}
		app := setupEmbedTestApp(svc)

		req := httptest.NewRequest("GET", "/embeds/templates", nil)
		req.Header.Set("X-Test-User-ID", testUserID.String())

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	})
}

func TestEmbedHandler_GetTemplate(t *testing.T) {
	testUserID := uuid.New()
	templateID := uuid.New()

	t.Run("success", func(t *testing.T) {
		title := "Test Title"
		svc := &mockEmbedService{
			getTemplateFunc: func(ctx context.Context, userID, tid uuid.UUID) (*models.EmbedTemplate, error) {
				assert.Equal(t, testUserID, userID)
				assert.Equal(t, templateID, tid)
				return &models.EmbedTemplate{
					ID:     tid,
					UserID: userID,
					Name:   "My Template",
					Title:  &title,
				}, nil
			},
		}
		app := setupEmbedTestApp(svc)

		req := httptest.NewRequest("GET", "/embeds/templates/"+templateID.String(), nil)
		req.Header.Set("X-Test-User-ID", testUserID.String())

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var result models.EmbedTemplateResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.Equal(t, "My Template", result.Name)
	})

	t.Run("not_found", func(t *testing.T) {
		svc := &mockEmbedService{
			getTemplateFunc: func(ctx context.Context, userID, tid uuid.UUID) (*models.EmbedTemplate, error) {
				return nil, services.ErrTemplateNotFound
			},
		}
		app := setupEmbedTestApp(svc)

		req := httptest.NewRequest("GET", "/embeds/templates/"+uuid.New().String(), nil)
		req.Header.Set("X-Test-User-ID", testUserID.String())

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})

	t.Run("unauthorized_no_user", func(t *testing.T) {
		svc := &mockEmbedService{}
		app := setupEmbedTestApp(svc)

		req := httptest.NewRequest("GET", "/embeds/templates/"+templateID.String(), nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("invalid_uuid", func(t *testing.T) {
		svc := &mockEmbedService{}
		app := setupEmbedTestApp(svc)

		req := httptest.NewRequest("GET", "/embeds/templates/not-a-uuid", nil)
		req.Header.Set("X-Test-User-ID", testUserID.String())

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

func TestEmbedHandler_UpdateTemplate(t *testing.T) {
	testUserID := uuid.New()
	templateID := uuid.New()

	t.Run("success", func(t *testing.T) {
		newTitle := "Updated Title"
		svc := &mockEmbedService{
			updateTemplateFunc: func(ctx context.Context, userID, tid uuid.UUID, req *models.UpdateEmbedTemplateRequest) (*models.EmbedTemplate, error) {
				assert.Equal(t, testUserID, userID)
				assert.Equal(t, templateID, tid)
				assert.NotNil(t, req.Title)
				assert.Equal(t, newTitle, *req.Title)
				title := newTitle
				return &models.EmbedTemplate{
					ID:     tid,
					UserID: userID,
					Name:   "Updated Template",
					Title:  &title,
				}, nil
			},
		}
		app := setupEmbedTestApp(svc)

		body := map[string]interface{}{"title": newTitle}
		bodyBytes, _ := json.Marshal(body)
		req := httptest.NewRequest("PUT", "/embeds/templates/"+templateID.String(), bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-User-ID", testUserID.String())

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var result models.EmbedTemplateResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.Equal(t, "Updated Template", result.Name)
	})

	t.Run("not_found", func(t *testing.T) {
		svc := &mockEmbedService{
			updateTemplateFunc: func(ctx context.Context, userID, tid uuid.UUID, req *models.UpdateEmbedTemplateRequest) (*models.EmbedTemplate, error) {
				return nil, services.ErrTemplateNotFound
			},
		}
		app := setupEmbedTestApp(svc)

		body := map[string]interface{}{"title": "Updated"}
		bodyBytes, _ := json.Marshal(body)
		req := httptest.NewRequest("PUT", "/embeds/templates/"+uuid.New().String(), bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-User-ID", testUserID.String())

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})

	t.Run("unauthorized_no_user", func(t *testing.T) {
		svc := &mockEmbedService{}
		app := setupEmbedTestApp(svc)

		body := map[string]interface{}{"title": "Updated"}
		bodyBytes, _ := json.Marshal(body)
		req := httptest.NewRequest("PUT", "/embeds/templates/"+templateID.String(), bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("invalid_uuid", func(t *testing.T) {
		svc := &mockEmbedService{}
		app := setupEmbedTestApp(svc)

		body := map[string]interface{}{"title": "Updated"}
		bodyBytes, _ := json.Marshal(body)
		req := httptest.NewRequest("PUT", "/embeds/templates/not-a-uuid", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-User-ID", testUserID.String())

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("invalid_json", func(t *testing.T) {
		svc := &mockEmbedService{}
		app := setupEmbedTestApp(svc)

		req := httptest.NewRequest("PUT", "/embeds/templates/"+templateID.String(), bytes.NewReader([]byte("not json")))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-User-ID", testUserID.String())

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

func TestEmbedHandler_DeleteTemplate(t *testing.T) {
	testUserID := uuid.New()
	templateID := uuid.New()

	t.Run("success", func(t *testing.T) {
		svc := &mockEmbedService{
			deleteTemplateFunc: func(ctx context.Context, userID, tid uuid.UUID) error {
				assert.Equal(t, testUserID, userID)
				assert.Equal(t, templateID, tid)
				return nil
			},
		}
		app := setupEmbedTestApp(svc)

		req := httptest.NewRequest("DELETE", "/embeds/templates/"+templateID.String(), nil)
		req.Header.Set("X-Test-User-ID", testUserID.String())

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)
	})

	t.Run("not_found", func(t *testing.T) {
		svc := &mockEmbedService{
			deleteTemplateFunc: func(ctx context.Context, userID, tid uuid.UUID) error {
				return services.ErrTemplateNotFound
			},
		}
		app := setupEmbedTestApp(svc)

		req := httptest.NewRequest("DELETE", "/embeds/templates/"+uuid.New().String(), nil)
		req.Header.Set("X-Test-User-ID", testUserID.String())

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})

	t.Run("unauthorized_no_user", func(t *testing.T) {
		svc := &mockEmbedService{}
		app := setupEmbedTestApp(svc)

		req := httptest.NewRequest("DELETE", "/embeds/templates/"+templateID.String(), nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("invalid_uuid", func(t *testing.T) {
		svc := &mockEmbedService{}
		app := setupEmbedTestApp(svc)

		req := httptest.NewRequest("DELETE", "/embeds/templates/not-a-uuid", nil)
		req.Header.Set("X-Test-User-ID", testUserID.String())

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

// strPtr is already defined in interactions_integration_test.go
