package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"hearth/internal/models"
	"hearth/internal/services"
	"hearth/internal/storage"
)

// --- mock services ---

type mockEmojiService struct {
	createFunc          func(ctx context.Context, serverID uuid.UUID, name, url string, animated bool) (*services.CustomEmoji, error)
	getServerEmojisFunc func(ctx context.Context, serverID uuid.UUID) ([]*services.CustomEmoji, error)
	getFunc             func(ctx context.Context, emojiID uuid.UUID) (*services.CustomEmoji, error)
	updateFunc          func(ctx context.Context, emojiID uuid.UUID, name string) (*services.CustomEmoji, error)
	deleteFunc          func(ctx context.Context, emojiID uuid.UUID) error
}

func (m *mockEmojiService) Create(ctx context.Context, serverID uuid.UUID, name, url string, animated bool) (*services.CustomEmoji, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, serverID, name, url, animated)
	}
	return &services.CustomEmoji{ID: uuid.New(), ServerID: serverID, Name: name, URL: url, Animated: animated}, nil
}

func (m *mockEmojiService) GetServerEmojis(ctx context.Context, serverID uuid.UUID) ([]*services.CustomEmoji, error) {
	if m.getServerEmojisFunc != nil {
		return m.getServerEmojisFunc(ctx, serverID)
	}
	return nil, nil
}

func (m *mockEmojiService) Get(ctx context.Context, emojiID uuid.UUID) (*services.CustomEmoji, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, emojiID)
	}
	return nil, services.ErrEmojiNotFound
}

func (m *mockEmojiService) Update(ctx context.Context, emojiID uuid.UUID, name string) (*services.CustomEmoji, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, emojiID, name)
	}
	return &services.CustomEmoji{ID: emojiID, Name: name}, nil
}

func (m *mockEmojiService) Delete(ctx context.Context, emojiID uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, emojiID)
	}
	return nil
}

type mockPermServiceEmoji struct {
	requirePermFunc func(ctx context.Context, serverID, userID uuid.UUID, permission int64) error
}

func (m *mockPermServiceEmoji) RequirePermission(ctx context.Context, serverID, userID uuid.UUID, permission int64) error {
	if m.requirePermFunc != nil {
		return m.requirePermFunc(ctx, serverID, userID, permission)
	}
	return nil
}

type mockStorageServiceEmoji struct {
	uploadFileFunc func(ctx context.Context, file *multipart.FileHeader, uploaderID uuid.UUID, category string) (*storage.FileInfo, error)
}

func (m *mockStorageServiceEmoji) UploadFile(ctx context.Context, file *multipart.FileHeader, uploaderID uuid.UUID, category string) (*storage.FileInfo, error) {
	if m.uploadFileFunc != nil {
		return m.uploadFileFunc(ctx, file, uploaderID, category)
	}
	return &storage.FileInfo{URL: "https://example.com/emojis/test.png"}, nil
}

// --- test app setup helpers ---

func setupEmojiTestApp() *fiber.App {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		if uidStr := c.Get("X-Test-User-ID"); uidStr != "" {
			if uid, err := uuid.Parse(uidStr); err == nil {
				c.Locals("userID", uid)
			}
		}
		return c.Next()
	})
	return app
}

// registerEmojiRoutes registers emoji routes with the given mocks
func registerEmojiRoutes(app *fiber.App, emojiSvc *mockEmojiService, permSvc *mockPermServiceEmoji, storageSvc *mockStorageServiceEmoji) {
	app.Get("/servers/:id/emojis", func(c *fiber.Ctx) error {
		serverID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid server id"})
		}
		emojis, err := emojiSvc.GetServerEmojis(c.Context(), serverID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to get emojis"})
		}
		type emojiResp struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			URL   string `json:"url"`
			Roles []any  `json:"roles,omitempty"`
		}
		var response []emojiResp
		for _, e := range emojis {
			response = append(response, emojiResp{ID: e.ID.String(), Name: e.Name, URL: e.URL})
		}
		return c.JSON(response)
	})

	app.Post("/servers/:id/emojis", func(c *fiber.Ctx) error {
		userID, ok := c.Locals("userID").(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		serverID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid server id"})
		}
		name := c.FormValue("name")
		if name == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "emoji name is required"})
		}
		if len(name) < 2 || len(name) > 32 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "emoji name must be 2-32 characters"})
		}
		for _, r := range name {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "emoji name can only contain letters, numbers, and underscores"})
			}
		}
		file, err := c.FormFile("image")
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "image file is required"})
		}
		if file.Size > 256*1024 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "emoji image must be under 256KB"})
		}
		contentType := file.Header.Get("Content-Type")
		allowedTypes := map[string]bool{"image/png": true, "image/gif": true, "image/jpeg": true, "image/webp": true}
		if !allowedTypes[contentType] {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid image type. Use PNG, GIF, JPEG, or WebP"})
		}
		animated := contentType == "image/gif"
		var url string
		if storageSvc != nil {
			fi, err := storageSvc.UploadFile(c.Context(), file, userID, "emojis")
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to upload emoji image"})
			}
			url = fi.URL
		}
		emoji, err := emojiSvc.Create(c.Context(), serverID, name, url, animated)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create emoji"})
		}
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"id":             emoji.ID.String(),
			"name":           emoji.Name,
			"url":            emoji.URL,
			"animated":       emoji.Animated,
			"require_colons": true,
		})
	})

	app.Get("/servers/:id/emojis/:emojiId", func(c *fiber.Ctx) error {
		serverID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid server id"})
		}
		emojiID, err := uuid.Parse(c.Params("emojiId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid emoji id"})
		}
		emojis, err := emojiSvc.GetServerEmojis(c.Context(), serverID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to get emojis"})
		}
		for _, e := range emojis {
			if e.ID == emojiID {
				return c.JSON(fiber.Map{
					"id":       e.ID.String(),
					"name":     e.Name,
					"url":      e.URL,
					"animated": e.Animated,
				})
			}
		}
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "emoji not found"})
	})

	app.Patch("/servers/:id/emojis/:emojiId", func(c *fiber.Ctx) error {
		userID, ok := c.Locals("userID").(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		emojiID, err := uuid.Parse(c.Params("emojiId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid emoji id"})
		}
		var req struct {
			Name string `json:"name"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}
		if req.Name != "" && (len(req.Name) < 2 || len(req.Name) > 32) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "emoji name must be 2-32 characters"})
		}
		emoji, err := emojiSvc.Get(c.Context(), emojiID)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "emoji not found"})
		}
		if permSvc != nil {
			if err := permSvc.RequirePermission(c.Context(), emoji.ServerID, userID, models.PermManageEmoji); err != nil {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "missing MANAGE_EMOJI permission"})
			}
		}
		updated, err := emojiSvc.Update(c.Context(), emojiID, req.Name)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update emoji"})
		}
		return c.JSON(updated)
	})

	app.Delete("/servers/:id/emojis/:emojiId", func(c *fiber.Ctx) error {
		userID, ok := c.Locals("userID").(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		emojiID, err := uuid.Parse(c.Params("emojiId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid emoji id"})
		}
		emoji, err := emojiSvc.Get(c.Context(), emojiID)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "emoji not found"})
		}
		if permSvc != nil {
			if err := permSvc.RequirePermission(c.Context(), emoji.ServerID, userID, models.PermManageEmoji); err != nil {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "missing MANAGE_EMOJI permission"})
			}
		}
		if err := emojiSvc.Delete(c.Context(), emojiID); err != nil {
			if errors.Is(err, services.ErrEmojiNotFound) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "emoji not found"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete emoji"})
		}
		return c.SendStatus(fiber.StatusNoContent)
	})
}

// --- tests for ListServerEmojis ---

func TestListServerEmojis_Success(t *testing.T) {
	emojiSvc := &mockEmojiService{
		getServerEmojisFunc: func(ctx context.Context, serverID uuid.UUID) ([]*services.CustomEmoji, error) {
			return []*services.CustomEmoji{
				{ID: uuid.New(), ServerID: serverID, Name: "smile", URL: "https://example.com/smile.png", Animated: false},
				{ID: uuid.New(), ServerID: serverID, Name: "wave", URL: "https://example.com/wave.gif", Animated: true},
			}, nil
		},
	}
	app := setupEmojiTestApp()
	registerEmojiRoutes(app, emojiSvc, nil, nil)
	t.Cleanup(func() { _ = app.Shutdown() })

	serverID := uuid.New()
	req := httptest.NewRequest("GET", "/servers/"+serverID.String()+"/emojis", nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result []map[string]any
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestListServerEmojis_NoEmojis(t *testing.T) {
	emojiSvc := &mockEmojiService{
		getServerEmojisFunc: func(ctx context.Context, serverID uuid.UUID) ([]*services.CustomEmoji, error) {
			return []*services.CustomEmoji{}, nil
		},
	}
	app := setupEmojiTestApp()
	registerEmojiRoutes(app, emojiSvc, nil, nil)
	t.Cleanup(func() { _ = app.Shutdown() })

	serverID := uuid.New()
	req := httptest.NewRequest("GET", "/servers/"+serverID.String()+"/emojis", nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result []map[string]any
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Len(t, result, 0)
}

func TestListServerEmojis_InvalidServerID(t *testing.T) {
	emojiSvc := &mockEmojiService{}
	app := setupEmojiTestApp()
	registerEmojiRoutes(app, emojiSvc, nil, nil)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("GET", "/servers/not-a-uuid/emojis", nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestListServerEmojis_ServiceError(t *testing.T) {
	emojiSvc := &mockEmojiService{
		getServerEmojisFunc: func(ctx context.Context, serverID uuid.UUID) ([]*services.CustomEmoji, error) {
			return nil, errors.New("db error")
		},
	}
	app := setupEmojiTestApp()
	registerEmojiRoutes(app, emojiSvc, nil, nil)
	t.Cleanup(func() { _ = app.Shutdown() })

	serverID := uuid.New()
	req := httptest.NewRequest("GET", "/servers/"+serverID.String()+"/emojis", nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

// --- tests for GetEmoji ---

func TestGetEmoji_Success(t *testing.T) {
	serverID := uuid.New()
	emojiID := uuid.New()
	emojiSvc := &mockEmojiService{
		getServerEmojisFunc: func(ctx context.Context, sid uuid.UUID) ([]*services.CustomEmoji, error) {
			return []*services.CustomEmoji{
				{ID: emojiID, ServerID: sid, Name: "smile", URL: "https://example.com/smile.png", Animated: false},
			}, nil
		},
	}
	app := setupEmojiTestApp()
	registerEmojiRoutes(app, emojiSvc, nil, nil)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("GET", "/servers/"+serverID.String()+"/emojis/"+emojiID.String(), nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]any
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "smile", result["name"])
}

func TestGetEmoji_NotFound(t *testing.T) {
	emojiSvc := &mockEmojiService{
		getServerEmojisFunc: func(ctx context.Context, sid uuid.UUID) ([]*services.CustomEmoji, error) {
			return []*services.CustomEmoji{}, nil
		},
	}
	app := setupEmojiTestApp()
	registerEmojiRoutes(app, emojiSvc, nil, nil)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("GET", "/servers/"+uuid.New().String()+"/emojis/"+uuid.New().String(), nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestGetEmoji_InvalidServerID(t *testing.T) {
	emojiSvc := &mockEmojiService{}
	app := setupEmojiTestApp()
	registerEmojiRoutes(app, emojiSvc, nil, nil)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("GET", "/servers/not-a-uuid/emojis/"+uuid.New().String(), nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestGetEmoji_InvalidEmojiID(t *testing.T) {
	emojiSvc := &mockEmojiService{}
	app := setupEmojiTestApp()
	registerEmojiRoutes(app, emojiSvc, nil, nil)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("GET", "/servers/"+uuid.New().String()+"/emojis/not-a-uuid", nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

// --- tests for DeleteEmoji ---

func TestDeleteEmoji_Success(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()
	emojiID := uuid.New()
	deleted := false
	emojiSvc := &mockEmojiService{
		getFunc: func(ctx context.Context, id uuid.UUID) (*services.CustomEmoji, error) {
			return &services.CustomEmoji{ID: id, ServerID: serverID, Name: "to_delete"}, nil
		},
		deleteFunc: func(ctx context.Context, id uuid.UUID) error {
			deleted = true
			return nil
		},
	}
	app := setupEmojiTestApp()
	registerEmojiRoutes(app, emojiSvc, nil, nil)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("DELETE", "/servers/"+serverID.String()+"/emojis/"+emojiID.String(), nil)
	req.Header.Set("X-Test-User-ID", userID.String())
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)
	assert.True(t, deleted)
}

func TestDeleteEmoji_NotFound(t *testing.T) {
	userID := uuid.New()
	emojiSvc := &mockEmojiService{
		getFunc: func(ctx context.Context, id uuid.UUID) (*services.CustomEmoji, error) {
			return nil, services.ErrEmojiNotFound
		},
	}
	app := setupEmojiTestApp()
	registerEmojiRoutes(app, emojiSvc, nil, nil)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("DELETE", "/servers/"+uuid.New().String()+"/emojis/"+uuid.New().String(), nil)
	req.Header.Set("X-Test-User-ID", userID.String())
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestDeleteEmoji_InvalidEmojiID(t *testing.T) {
	userID := uuid.New()
	emojiSvc := &mockEmojiService{}
	app := setupEmojiTestApp()
	registerEmojiRoutes(app, emojiSvc, nil, nil)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("DELETE", "/servers/"+uuid.New().String()+"/emojis/not-a-uuid", nil)
	req.Header.Set("X-Test-User-ID", userID.String())
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestDeleteEmoji_PermissionDenied(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()
	emojiID := uuid.New()
	emojiSvc := &mockEmojiService{
		getFunc: func(ctx context.Context, id uuid.UUID) (*services.CustomEmoji, error) {
			return &services.CustomEmoji{ID: id, ServerID: serverID, Name: "test"}, nil
		},
	}
	permSvc := &mockPermServiceEmoji{
		requirePermFunc: func(ctx context.Context, sid, uid uuid.UUID, perm int64) error {
			return errors.New("forbidden")
		},
	}
	app := setupEmojiTestApp()
	registerEmojiRoutes(app, emojiSvc, permSvc, nil)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("DELETE", "/servers/"+serverID.String()+"/emojis/"+emojiID.String(), nil)
	req.Header.Set("X-Test-User-ID", userID.String())
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

func TestDeleteEmoji_Unauthorized(t *testing.T) {
	emojiSvc := &mockEmojiService{}
	app := setupEmojiTestApp()
	registerEmojiRoutes(app, emojiSvc, nil, nil)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("DELETE", "/servers/"+uuid.New().String()+"/emojis/"+uuid.New().String(), nil)
	// No X-Test-User-ID header
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

// --- tests for UpdateEmoji ---

func TestUpdateEmoji_Success(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()
	emojiID := uuid.New()
	emojiSvc := &mockEmojiService{
		getFunc: func(ctx context.Context, id uuid.UUID) (*services.CustomEmoji, error) {
			return &services.CustomEmoji{ID: id, ServerID: serverID, Name: "old_name"}, nil
		},
		updateFunc: func(ctx context.Context, id uuid.UUID, name string) (*services.CustomEmoji, error) {
			return &services.CustomEmoji{ID: id, ServerID: serverID, Name: name}, nil
		},
	}
	app := setupEmojiTestApp()
	registerEmojiRoutes(app, emojiSvc, nil, nil)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"name": "new_name"}`
	req := httptest.NewRequest("PATCH", "/servers/"+serverID.String()+"/emojis/"+emojiID.String(), bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestUpdateEmoji_InvalidBody(t *testing.T) {
	userID := uuid.New()
	emojiSvc := &mockEmojiService{}
	app := setupEmojiTestApp()
	registerEmojiRoutes(app, emojiSvc, nil, nil)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("PATCH", "/servers/"+uuid.New().String()+"/emojis/"+uuid.New().String(), bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestUpdateEmoji_NameTooShort(t *testing.T) {
	userID := uuid.New()
	emojiSvc := &mockEmojiService{}
	app := setupEmojiTestApp()
	registerEmojiRoutes(app, emojiSvc, nil, nil)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"name": "a"}`
	req := httptest.NewRequest("PATCH", "/servers/"+uuid.New().String()+"/emojis/"+uuid.New().String(), bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestUpdateEmoji_NameTooLong(t *testing.T) {
	userID := uuid.New()
	emojiSvc := &mockEmojiService{}
	app := setupEmojiTestApp()
	registerEmojiRoutes(app, emojiSvc, nil, nil)
	t.Cleanup(func() { _ = app.Shutdown() })

	longName := make([]byte, 33)
	for i := range longName {
		longName[i] = 'a'
	}
	body := `{"name": "` + string(longName) + `"}`
	req := httptest.NewRequest("PATCH", "/servers/"+uuid.New().String()+"/emojis/"+uuid.New().String(), bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestUpdateEmoji_NotFound(t *testing.T) {
	userID := uuid.New()
	emojiSvc := &mockEmojiService{
		getFunc: func(ctx context.Context, id uuid.UUID) (*services.CustomEmoji, error) {
			return nil, services.ErrEmojiNotFound
		},
	}
	app := setupEmojiTestApp()
	registerEmojiRoutes(app, emojiSvc, nil, nil)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"name": "new_name"}`
	req := httptest.NewRequest("PATCH", "/servers/"+uuid.New().String()+"/emojis/"+uuid.New().String(), bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestUpdateEmoji_PermissionDenied(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()
	emojiID := uuid.New()
	emojiSvc := &mockEmojiService{
		getFunc: func(ctx context.Context, id uuid.UUID) (*services.CustomEmoji, error) {
			return &services.CustomEmoji{ID: id, ServerID: serverID, Name: "test"}, nil
		},
	}
	permSvc := &mockPermServiceEmoji{
		requirePermFunc: func(ctx context.Context, sid, uid uuid.UUID, perm int64) error {
			return errors.New("forbidden")
		},
	}
	app := setupEmojiTestApp()
	registerEmojiRoutes(app, emojiSvc, permSvc, nil)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"name": "new_name"}`
	req := httptest.NewRequest("PATCH", "/servers/"+serverID.String()+"/emojis/"+emojiID.String(), bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

// --- tests for CreateEmoji ---

func TestCreateEmoji_MissingName(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()
	emojiSvc := &mockEmojiService{}
	app := setupEmojiTestApp()
	registerEmojiRoutes(app, emojiSvc, nil, nil)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("image", "test.png")
	_, _ = io.WriteString(part, "fake png content")
	writer.Close()

	req := httptest.NewRequest("POST", "/servers/"+serverID.String()+"/emojis", bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Test-User-ID", userID.String())
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestCreateEmoji_NameTooShort(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()
	emojiSvc := &mockEmojiService{}
	app := setupEmojiTestApp()
	registerEmojiRoutes(app, emojiSvc, nil, nil)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("name", "a")
	part, _ := writer.CreateFormFile("image", "test.png")
	_, _ = io.WriteString(part, "fake png content")
	writer.Close()

	req := httptest.NewRequest("POST", "/servers/"+serverID.String()+"/emojis", bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Test-User-ID", userID.String())
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestCreateEmoji_NameWithInvalidChars(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()
	emojiSvc := &mockEmojiService{}
	app := setupEmojiTestApp()
	registerEmojiRoutes(app, emojiSvc, nil, nil)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("name", "bad-emoji!")
	part, _ := writer.CreateFormFile("image", "test.png")
	_, _ = io.WriteString(part, "fake png content")
	writer.Close()

	req := httptest.NewRequest("POST", "/servers/"+serverID.String()+"/emojis", bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Test-User-ID", userID.String())
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestCreateEmoji_MissingImage(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()
	emojiSvc := &mockEmojiService{}
	app := setupEmojiTestApp()
	registerEmojiRoutes(app, emojiSvc, nil, nil)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("name", "valid_emoji")
	writer.Close()

	req := httptest.NewRequest("POST", "/servers/"+serverID.String()+"/emojis", bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Test-User-ID", userID.String())
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestCreateEmoji_InvalidServerID(t *testing.T) {
	userID := uuid.New()
	emojiSvc := &mockEmojiService{}
	app := setupEmojiTestApp()
	registerEmojiRoutes(app, emojiSvc, nil, nil)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("name", "valid_emoji")
	part, _ := writer.CreateFormFile("image", "test.png")
	_, _ = io.WriteString(part, "fake png content")
	writer.Close()

	req := httptest.NewRequest("POST", "/servers/not-a-uuid/emojis", bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Test-User-ID", userID.String())
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestCreateEmoji_InvalidContentType(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()
	emojiSvc := &mockEmojiService{}
	app := setupEmojiTestApp()
	registerEmojiRoutes(app, emojiSvc, nil, nil)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("name", "valid_emoji")
	part, _ := writer.CreateFormFile("image", "test.svg")
	_, _ = io.WriteString(part, "<svg></svg>")
	writer.Close()

	req := httptest.NewRequest("POST", "/servers/"+serverID.String()+"/emojis", bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Test-User-ID", userID.String())
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestCreateEmoji_Unauthorized(t *testing.T) {
	serverID := uuid.New()
	emojiSvc := &mockEmojiService{}
	app := setupEmojiTestApp()
	registerEmojiRoutes(app, emojiSvc, nil, nil)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("name", "valid_emoji")
	part, _ := writer.CreateFormFile("image", "test.png")
	_, _ = io.WriteString(part, "fake png content")
	writer.Close()

	req := httptest.NewRequest("POST", "/servers/"+serverID.String()+"/emojis", bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", writer.FormDataContentType())
	// No X-Test-User-ID header
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestCreateEmoji_Success(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()
	emojiSvc := &mockEmojiService{
		createFunc: func(ctx context.Context, sid uuid.UUID, name, url string, animated bool) (*services.CustomEmoji, error) {
			return &services.CustomEmoji{
				ID:       uuid.New(),
				ServerID: sid,
				Name:     name,
				URL:      url,
				Animated: animated,
			}, nil
		},
	}
	storageSvc := &mockStorageServiceEmoji{
		uploadFileFunc: func(ctx context.Context, file *multipart.FileHeader, uploaderID uuid.UUID, category string) (*storage.FileInfo, error) {
			return &storage.FileInfo{URL: "https://cdn.example.com/emojis/test.png"}, nil
		},
	}
	app := setupEmojiTestApp()
	registerEmojiRoutes(app, emojiSvc, nil, storageSvc)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("name", "valid_emoji")
	hdr := make(textproto.MIMEHeader)
	hdr.Set("Content-Disposition", `form-data; name="image"; filename="test.png"`)
	hdr.Set("Content-Type", "image/png")
	part, _ := writer.CreatePart(hdr)
	_, _ = io.WriteString(part, "fake png content")
	writer.Close()

	req := httptest.NewRequest("POST", "/servers/"+serverID.String()+"/emojis", bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Test-User-ID", userID.String())
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var result map[string]any
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "valid_emoji", result["name"])
}

func TestCreateEmoji_SuccessAnimatedGIF(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()
	emojiSvc := &mockEmojiService{
		createFunc: func(ctx context.Context, sid uuid.UUID, name, url string, animated bool) (*services.CustomEmoji, error) {
			return &services.CustomEmoji{
				ID:       uuid.New(),
				ServerID: sid,
				Name:     name,
				URL:      url,
				Animated: animated,
			}, nil
		},
	}
	storageSvc := &mockStorageServiceEmoji{
		uploadFileFunc: func(ctx context.Context, file *multipart.FileHeader, uploaderID uuid.UUID, category string) (*storage.FileInfo, error) {
			return &storage.FileInfo{URL: "https://cdn.example.com/emojis/anim.gif"}, nil
		},
	}
	app := setupEmojiTestApp()
	registerEmojiRoutes(app, emojiSvc, nil, storageSvc)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("name", "animated_emoji")
	hdr := make(textproto.MIMEHeader)
	hdr.Set("Content-Disposition", `form-data; name="image"; filename="anim.gif"`)
	hdr.Set("Content-Type", "image/gif")
	part, _ := writer.CreatePart(hdr)
	_, _ = io.WriteString(part, "GIF89a")
	writer.Close()

	req := httptest.NewRequest("POST", "/servers/"+serverID.String()+"/emojis", bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Test-User-ID", userID.String())
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var result map[string]any
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, true, result["animated"])
}
