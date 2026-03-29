package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"hearth/internal/models"
	"hearth/internal/services"
)

// mockTemplateService is a mock implementation of TemplateService for testing
type mockTemplateService struct {
	createTemplateFunc       func(ctx context.Context, serverID, userID uuid.UUID, name, description string, isPublic bool) (*models.ServerTemplate, error)
	getTemplateFunc          func(ctx context.Context, code string) (*models.ServerTemplate, error)
	getTemplateByIDFunc     func(ctx context.Context, templateID uuid.UUID) (*models.ServerTemplate, error)
	listMyTemplatesFunc     func(ctx context.Context, userID uuid.UUID) ([]*models.ServerTemplate, error)
	listPublicTemplatesFunc func(ctx context.Context, cursor *uuid.UUID, limit int) ([]*models.ServerTemplate, *uuid.UUID, error)
	updateTemplateFunc      func(ctx context.Context, templateID uuid.UUID, name, description *string, isPublic *bool) (*models.ServerTemplate, error)
	deleteTemplateFunc      func(ctx context.Context, templateID uuid.UUID) error
	useTemplateFunc          func(ctx context.Context, code string, userID uuid.UUID, name string) (*models.Server, error)
}

// setupTemplateTestApp creates a Fiber app with user ID middleware for testing
func setupTemplateTestApp(mock *mockTemplateService) *fiber.App {
	app := fiber.New()

	// Add user ID extraction middleware
	app.Use(func(c *fiber.Ctx) error {
		userIDStr := c.Get("X-Test-User-ID")
		if userIDStr != "" {
			uid, err := uuid.Parse(userIDStr)
			if err == nil {
				c.Locals("userID", uid)
			}
		}
		return c.Next()
	})

	// Create template
	app.Post("/servers/:id/templates", func(c *fiber.Ctx) error {
		uid, ok := c.Locals("userID").(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid server id"})
		}
		var req models.CreateTemplateRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}
		if req.Name == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "template name is required"})
		}
		if len(req.Name) > 100 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "template name must be 100 characters or less"})
		}
		tmpl, err := mock.createTemplateFunc(c.Context(), id, uid, req.Name, req.Description, req.IsPublic)
		if err != nil {
			if err == services.ErrServerNotFound {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "server not found"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create template"})
		}
		return c.Status(fiber.StatusCreated).JSON(tmpl.ToResponse())
	})

	// Get template by code
	app.Get("/templates/:code", func(c *fiber.Ctx) error {
		code := c.Params("code")
		if code == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "template code is required"})
		}
		tmpl, err := mock.getTemplateFunc(c.Context(), code)
		if err != nil {
			if err == services.ErrTemplateNotFound {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "template not found"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to get template"})
		}
		return c.JSON(tmpl.ToResponse())
	})

	// List my templates
	app.Get("/users/me/templates", func(c *fiber.Ctx) error {
		uid, ok := c.Locals("userID").(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		templates, err := mock.listMyTemplatesFunc(c.Context(), uid)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list templates"})
		}
		response := make([]*models.ServerTemplateResponse, 0, len(templates))
		for _, t := range templates {
			response = append(response, t.ToResponse())
		}
		return c.JSON(response)
	})

	// List public templates
	app.Get("/templates", func(c *fiber.Ctx) error {
		var cursor *uuid.UUID
		cursorStr := c.Query("cursor")
		if cursorStr != "" {
			id, err := uuid.Parse(cursorStr)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid cursor"})
			}
			cursor = &id
		}
		limit := c.QueryInt("limit", 20)
		if limit <= 0 || limit > 50 {
			limit = 20
		}
		templates, nextID, err := mock.listPublicTemplatesFunc(c.Context(), cursor, limit)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list templates"})
		}
		response := make([]*models.ServerTemplateResponse, 0, len(templates))
		for _, t := range templates {
			response = append(response, t.ToResponse())
		}
		return c.JSON(fiber.Map{"templates": response, "next_cursor": nextID})
	})

	// Update template
	app.Patch("/templates/:templateId", func(c *fiber.Ctx) error {
		uid, ok := c.Locals("userID").(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		tID, err := uuid.Parse(c.Params("templateId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid template id"})
		}
		existing, err := mock.getTemplateByIDFunc(c.Context(), tID)
		if err != nil {
			if err == services.ErrTemplateNotFound {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "template not found"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to get template"})
		}
		if existing.CreatorID != uid {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "you can only update your own templates"})
		}
		var req models.UpdateTemplateRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}
		tmpl, err := mock.updateTemplateFunc(c.Context(), tID, req.Name, req.Description, req.IsPublic)
		if err != nil {
			if err == services.ErrTemplateNotFound {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "template not found"})
			}
			if err == services.ErrTemplateNameTooLong {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "template name must be 100 characters or less"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update template"})
		}
		return c.JSON(tmpl.ToResponse())
	})

	// Delete template
	app.Delete("/templates/:templateId", func(c *fiber.Ctx) error {
		uid, ok := c.Locals("userID").(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		tID, err := uuid.Parse(c.Params("templateId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid template id"})
		}
		existing, err := mock.getTemplateByIDFunc(c.Context(), tID)
		if err != nil {
			if err == services.ErrTemplateNotFound {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "template not found"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to get template"})
		}
		if existing.CreatorID != uid {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "you can only delete your own templates"})
		}
		if err := mock.deleteTemplateFunc(c.Context(), tID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete template"})
		}
		return c.SendStatus(fiber.StatusNoContent)
	})

	// Use template
	app.Post("/templates/:code/use", func(c *fiber.Ctx) error {
		uid, ok := c.Locals("userID").(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		code := c.Params("code")
		if code == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "template code is required"})
		}
		var req struct {
			Name string `json:"name"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}
		if req.Name == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "server name is required"})
		}
		server, err := mock.useTemplateFunc(c.Context(), code, uid, req.Name)
		if err != nil {
			if err == services.ErrTemplateNotFound {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "template not found"})
			}
			if err == services.ErrInvalidTemplateData {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "template data is invalid"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create server from template"})
		}
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"id":         server.ID,
			"name":       server.Name,
			"owner_id":   server.OwnerID,
			"created_at": server.CreatedAt,
		})
	})

	return app
}

// Test CreateTemplate
func TestCreateTemplate_Success(t *testing.T) {
	serverID := uuid.New()
	userID := uuid.New()
	templateID := uuid.New()

	mock := &mockTemplateService{
		createTemplateFunc: func(ctx context.Context, sID, uID uuid.UUID, name, description string, isPublic bool) (*models.ServerTemplate, error) {
			assert.Equal(t, serverID, sID)
			assert.Equal(t, userID, uID)
			assert.Equal(t, "My Template", name)
			assert.Equal(t, "A great template", description)
			assert.False(t, isPublic)
			return &models.ServerTemplate{
				ID:             templateID,
				SourceServerID: &sID,
				CreatorID:      uID,
				Name:           name,
				Description:    description,
				IsPublic:       isPublic,
				Code:           "template-code",
			}, nil
		},
	}

	app := setupTemplateTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"name":"My Template","description":"A great template","is_public":false}`
	req := httptest.NewRequest(http.MethodPost, "/servers/"+serverID.String()+"/templates", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, templateID.String(), result["id"])
	assert.Equal(t, "My Template", result["name"])
}

func TestCreateTemplate_Unauthorized(t *testing.T) {
	mock := &mockTemplateService{}
	app := setupTemplateTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	// No X-Test-User-ID header
	req := httptest.NewRequest(http.MethodPost, "/servers/"+uuid.New().String()+"/templates", nil)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestCreateTemplate_InvalidServerID(t *testing.T) {
	mock := &mockTemplateService{}
	app := setupTemplateTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest(http.MethodPost, "/servers/invalid-id/templates", nil)
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestCreateTemplate_InvalidBody(t *testing.T) {
	mock := &mockTemplateService{}
	app := setupTemplateTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest(http.MethodPost, "/servers/"+uuid.New().String()+"/templates", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestCreateTemplate_MissingName(t *testing.T) {
	mock := &mockTemplateService{}
	app := setupTemplateTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"description":"A template without name"}`
	req := httptest.NewRequest(http.MethodPost, "/servers/"+uuid.New().String()+"/templates", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "template name is required", result["error"])
}

func TestCreateTemplate_NameTooLong(t *testing.T) {
	mock := &mockTemplateService{}
	app := setupTemplateTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	longName := string(make([]byte, 101))
	body := `{"name":"` + longName + `"}`
	req := httptest.NewRequest(http.MethodPost, "/servers/"+uuid.New().String()+"/templates", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestCreateTemplate_ServerNotFound(t *testing.T) {
	serverID := uuid.New()
	userID := uuid.New()

	mock := &mockTemplateService{
		createTemplateFunc: func(ctx context.Context, sID, uID uuid.UUID, name, description string, isPublic bool) (*models.ServerTemplate, error) {
			return nil, services.ErrServerNotFound
		},
	}

	app := setupTemplateTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"name":"My Template"}`
	req := httptest.NewRequest(http.MethodPost, "/servers/"+serverID.String()+"/templates", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// Test GetTemplate
func TestGetTemplate_Success(t *testing.T) {
	templateID := uuid.New()
	mock := &mockTemplateService{
		getTemplateFunc: func(ctx context.Context, code string) (*models.ServerTemplate, error) {
			assert.Equal(t, "template-code", code)
			return &models.ServerTemplate{
				ID:       templateID,
				Name:     "My Template",
				Code:     code,
				IsPublic: true,
			}, nil
		},
	}

	app := setupTemplateTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest(http.MethodGet, "/templates/template-code", nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestGetTemplate_NotFound(t *testing.T) {
	mock := &mockTemplateService{
		getTemplateFunc: func(ctx context.Context, code string) (*models.ServerTemplate, error) {
			return nil, services.ErrTemplateNotFound
		},
	}

	app := setupTemplateTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest(http.MethodGet, "/templates/nonexistent", nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// Test ListMyTemplates
func TestListMyTemplates_Success(t *testing.T) {
	userID := uuid.New()
	mock := &mockTemplateService{
		listMyTemplatesFunc: func(ctx context.Context, uID uuid.UUID) ([]*models.ServerTemplate, error) {
			assert.Equal(t, userID, uID)
			return []*models.ServerTemplate{
				{ID: uuid.New(), Name: "Template 1", Code: "t1"},
				{ID: uuid.New(), Name: "Template 2", Code: "t2"},
			}, nil
		},
	}

	app := setupTemplateTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest(http.MethodGet, "/users/me/templates", nil)
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result []*models.ServerTemplateResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestListMyTemplates_Unauthorized(t *testing.T) {
	mock := &mockTemplateService{}
	app := setupTemplateTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest(http.MethodGet, "/users/me/templates", nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// Test ListPublicTemplates
func TestListPublicTemplates_Success(t *testing.T) {
	nextCursor := uuid.New()
	mock := &mockTemplateService{
		listPublicTemplatesFunc: func(ctx context.Context, cursor *uuid.UUID, limit int) ([]*models.ServerTemplate, *uuid.UUID, error) {
			assert.Nil(t, cursor)
			assert.Equal(t, 20, limit)
			return []*models.ServerTemplate{
				{ID: uuid.New(), Name: "Public Template 1", Code: "pub1", IsPublic: true},
			}, &nextCursor, nil
		},
	}

	app := setupTemplateTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest(http.MethodGet, "/templates", nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.NotNil(t, result["templates"])
	assert.Equal(t, nextCursor.String(), result["next_cursor"])
}

func TestListPublicTemplates_WithCursor(t *testing.T) {
	cursor := uuid.New()
	mock := &mockTemplateService{
		listPublicTemplatesFunc: func(ctx context.Context, cur *uuid.UUID, limit int) ([]*models.ServerTemplate, *uuid.UUID, error) {
			assert.NotNil(t, cur)
			assert.Equal(t, cursor, *cur)
			return []*models.ServerTemplate{}, nil, nil
		},
	}

	app := setupTemplateTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest(http.MethodGet, "/templates?cursor="+cursor.String(), nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestListPublicTemplates_InvalidCursor(t *testing.T) {
	mock := &mockTemplateService{}
	app := setupTemplateTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest(http.MethodGet, "/templates?cursor=invalid", nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// Test UpdateTemplate
func TestUpdateTemplate_Success(t *testing.T) {
	userID := uuid.New()
	templateID := uuid.New()

	mock := &mockTemplateService{
		getTemplateByIDFunc: func(ctx context.Context, tID uuid.UUID) (*models.ServerTemplate, error) {
			assert.Equal(t, templateID, tID)
			return &models.ServerTemplate{ID: tID, CreatorID: userID, Name: "Old Name"}, nil
		},
		updateTemplateFunc: func(ctx context.Context, tID uuid.UUID, name, description *string, isPublic *bool) (*models.ServerTemplate, error) {
			assert.Equal(t, templateID, tID)
			return &models.ServerTemplate{ID: tID, Name: "New Name"}, nil
		},
	}

	app := setupTemplateTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"name":"New Name"}`
	req := httptest.NewRequest(http.MethodPatch, "/templates/"+templateID.String(), bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUpdateTemplate_NotOwner(t *testing.T) {
	ownerID := uuid.New()
	otherUserID := uuid.New()
	templateID := uuid.New()

	mock := &mockTemplateService{
		getTemplateByIDFunc: func(ctx context.Context, tID uuid.UUID) (*models.ServerTemplate, error) {
			return &models.ServerTemplate{ID: tID, CreatorID: ownerID}, nil
		},
	}

	app := setupTemplateTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"name":"New Name"}`
	req := httptest.NewRequest(http.MethodPatch, "/templates/"+templateID.String(), bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", otherUserID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestUpdateTemplate_NotFound(t *testing.T) {
	userID := uuid.New()
	templateID := uuid.New()

	mock := &mockTemplateService{
		getTemplateByIDFunc: func(ctx context.Context, tID uuid.UUID) (*models.ServerTemplate, error) {
			return nil, services.ErrTemplateNotFound
		},
	}

	app := setupTemplateTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"name":"New Name"}`
	req := httptest.NewRequest(http.MethodPatch, "/templates/"+templateID.String(), bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// Test DeleteTemplate
func TestDeleteTemplate_Success(t *testing.T) {
	userID := uuid.New()
	templateID := uuid.New()

	mock := &mockTemplateService{
		getTemplateByIDFunc: func(ctx context.Context, tID uuid.UUID) (*models.ServerTemplate, error) {
			return &models.ServerTemplate{ID: tID, CreatorID: userID}, nil
		},
		deleteTemplateFunc: func(ctx context.Context, tID uuid.UUID) error {
			assert.Equal(t, templateID, tID)
			return nil
		},
	}

	app := setupTemplateTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest(http.MethodDelete, "/templates/"+templateID.String(), nil)
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestDeleteTemplate_NotFound(t *testing.T) {
	userID := uuid.New()
	templateID := uuid.New()

	mock := &mockTemplateService{
		getTemplateByIDFunc: func(ctx context.Context, tID uuid.UUID) (*models.ServerTemplate, error) {
			return nil, services.ErrTemplateNotFound
		},
	}

	app := setupTemplateTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest(http.MethodDelete, "/templates/"+templateID.String(), nil)
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// Test UseTemplate
func TestUseTemplate_Success(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()

	mock := &mockTemplateService{
		useTemplateFunc: func(ctx context.Context, code string, uID uuid.UUID, name string) (*models.Server, error) {
			assert.Equal(t, "template-code", code)
			assert.Equal(t, userID, uID)
			assert.Equal(t, "My New Server", name)
			return &models.Server{
				ID:      serverID,
				Name:    name,
				OwnerID: uID,
			}, nil
		},
	}

	app := setupTemplateTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"name":"My New Server"}`
	req := httptest.NewRequest(http.MethodPost, "/templates/template-code/use", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, serverID.String(), result["id"])
	assert.Equal(t, "My New Server", result["name"])
}

func TestUseTemplate_TemplateNotFound(t *testing.T) {
	userID := uuid.New()

	mock := &mockTemplateService{
		useTemplateFunc: func(ctx context.Context, code string, uID uuid.UUID, name string) (*models.Server, error) {
			return nil, services.ErrTemplateNotFound
		},
	}

	app := setupTemplateTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"name":"My New Server"}`
	req := httptest.NewRequest(http.MethodPost, "/templates/nonexistent/use", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestUseTemplate_MissingName(t *testing.T) {
	userID := uuid.New()

	mock := &mockTemplateService{}
	app := setupTemplateTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"name":""}`
	req := httptest.NewRequest(http.MethodPost, "/templates/template-code/use", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestUseTemplate_Unauthorized(t *testing.T) {
	mock := &mockTemplateService{}
	app := setupTemplateTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest(http.MethodPost, "/templates/template-code/use", nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}
