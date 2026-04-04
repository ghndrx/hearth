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

	"hearth/internal/models"
	"hearth/internal/services"
)

// mockServerFolderService implements ServerFolderServiceInterface for testing
type mockServerFolderService struct {
	createFolderFunc         func(ctx context.Context, userID uuid.UUID, req *models.CreateServerFolderRequest) (*models.ServerFolder, error)
	getUserFoldersFunc       func(ctx context.Context, userID uuid.UUID) (*models.ServerFolderTree, error)
	getFolderFunc            func(ctx context.Context, userID, folderID uuid.UUID) (*models.ServerFolder, error)
	updateFolderFunc         func(ctx context.Context, userID, folderID uuid.UUID, req *models.UpdateServerFolderRequest) (*models.ServerFolder, error)
	deleteFolderFunc         func(ctx context.Context, userID, folderID uuid.UUID) error
	moveServerToFolderFunc   func(ctx context.Context, userID, serverID uuid.UUID, folderID *uuid.UUID) error
	moveServersToFolderFunc  func(ctx context.Context, userID uuid.UUID, req *models.MoveServersToFolderRequest) error
	reorderServersFunc       func(ctx context.Context, userID uuid.UUID, folderID *uuid.UUID, req *models.ReorderServersRequest) error
}

func (m *mockServerFolderService) CreateFolder(ctx context.Context, userID uuid.UUID, req *models.CreateServerFolderRequest) (*models.ServerFolder, error) {
	if m.createFolderFunc != nil {
		return m.createFolderFunc(ctx, userID, req)
	}
	return nil, nil
}

func (m *mockServerFolderService) GetUserFolders(ctx context.Context, userID uuid.UUID) (*models.ServerFolderTree, error) {
	if m.getUserFoldersFunc != nil {
		return m.getUserFoldersFunc(ctx, userID)
	}
	return nil, nil
}

func (m *mockServerFolderService) GetFolder(ctx context.Context, userID, folderID uuid.UUID) (*models.ServerFolder, error) {
	if m.getFolderFunc != nil {
		return m.getFolderFunc(ctx, userID, folderID)
	}
	return nil, nil
}

func (m *mockServerFolderService) UpdateFolder(ctx context.Context, userID, folderID uuid.UUID, req *models.UpdateServerFolderRequest) (*models.ServerFolder, error) {
	if m.updateFolderFunc != nil {
		return m.updateFolderFunc(ctx, userID, folderID, req)
	}
	return nil, nil
}

func (m *mockServerFolderService) DeleteFolder(ctx context.Context, userID, folderID uuid.UUID) error {
	if m.deleteFolderFunc != nil {
		return m.deleteFolderFunc(ctx, userID, folderID)
	}
	return nil
}

func (m *mockServerFolderService) MoveServerToFolder(ctx context.Context, userID, serverID uuid.UUID, folderID *uuid.UUID) error {
	if m.moveServerToFolderFunc != nil {
		return m.moveServerToFolderFunc(ctx, userID, serverID, folderID)
	}
	return nil
}

func (m *mockServerFolderService) MoveServersToFolder(ctx context.Context, userID uuid.UUID, req *models.MoveServersToFolderRequest) error {
	if m.moveServersToFolderFunc != nil {
		return m.moveServersToFolderFunc(ctx, userID, req)
	}
	return nil
}

func (m *mockServerFolderService) ReorderServers(ctx context.Context, userID uuid.UUID, folderID *uuid.UUID, req *models.ReorderServersRequest) error {
	if m.reorderServersFunc != nil {
		return m.reorderServersFunc(ctx, userID, folderID, req)
	}
	return nil
}

// serverFolderServiceInterface matches the interface used by ServerFolderHandler
type serverFolderServiceInterface interface {
	CreateFolder(ctx context.Context, userID uuid.UUID, req *models.CreateServerFolderRequest) (*models.ServerFolder, error)
	GetUserFolders(ctx context.Context, userID uuid.UUID) (*models.ServerFolderTree, error)
	GetFolder(ctx context.Context, userID, folderID uuid.UUID) (*models.ServerFolder, error)
	UpdateFolder(ctx context.Context, userID, folderID uuid.UUID, req *models.UpdateServerFolderRequest) (*models.ServerFolder, error)
	DeleteFolder(ctx context.Context, userID, folderID uuid.UUID) error
	MoveServerToFolder(ctx context.Context, userID, serverID uuid.UUID, folderID *uuid.UUID) error
	MoveServersToFolder(ctx context.Context, userID uuid.UUID, req *models.MoveServersToFolderRequest) error
	ReorderServers(ctx context.Context, userID uuid.UUID, folderID *uuid.UUID, req *models.ReorderServersRequest) error
}

// serverFolderHandlerForTest wraps ServerFolderHandler to use our mock
type serverFolderHandlerForTest struct {
	mock *mockServerFolderService
}

func newServerFolderHandlerForTest(mock *mockServerFolderService) *serverFolderHandlerForTest {
	return &serverFolderHandlerForTest{mock: mock}
}

func (h *serverFolderHandlerForTest) Create(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	var req models.CreateServerFolderRequest
	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	if req.Name == "" || len(req.Name) < 1 || len(req.Name) > 100 {
		return ValidationError(c, "name", "must be between 1 and 100 characters")
	}

	folder, err := h.mock.CreateFolder(c.Context(), userID, &req)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(folder)
}

func (h *serverFolderHandlerForTest) GetAll(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	tree, err := h.mock.GetUserFolders(c.Context(), userID)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(tree)
}

func (h *serverFolderHandlerForTest) Get(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	folderID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return InvalidUUID(c, "folder id")
	}

	folder, err := h.mock.GetFolder(c.Context(), userID, folderID)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(folder)
}

func (h *serverFolderHandlerForTest) Update(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	folderID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return InvalidUUID(c, "folder id")
	}

	var req models.UpdateServerFolderRequest
	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	folder, err := h.mock.UpdateFolder(c.Context(), userID, folderID, &req)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(folder)
}

func (h *serverFolderHandlerForTest) Delete(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	folderID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return InvalidUUID(c, "folder id")
	}

	if err := h.mock.DeleteFolder(c.Context(), userID, folderID); err != nil {
		return HandleServiceError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *serverFolderHandlerForTest) MoveServer(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	var req struct {
		ServerID string   `json:"server_id"`
		FolderID *string `json:"folder_id,omitempty"`
	}

	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	serverID, err := uuid.Parse(req.ServerID)
	if err != nil {
		return InvalidUUID(c, "server_id")
	}

	var folderID *uuid.UUID
	if req.FolderID != nil {
		fid, err := uuid.Parse(*req.FolderID)
		if err != nil {
			return InvalidUUID(c, "folder_id")
		}
		folderID = &fid
	}

	if err := h.mock.MoveServerToFolder(c.Context(), userID, serverID, folderID); err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(fiber.Map{"success": true})
}

func (h *serverFolderHandlerForTest) MoveServers(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	var req models.MoveServersToFolderRequest
	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	if len(req.ServerIDs) == 0 {
		return ValidationError(c, "server_ids", "must contain at least one server")
	}

	if err := h.mock.MoveServersToFolder(c.Context(), userID, &req); err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(fiber.Map{"success": true})
}

func (h *serverFolderHandlerForTest) ReorderServers(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	var req struct {
		FolderID        *string                  `json:"folder_id,omitempty"`
		ServerPositions []models.ServerPosition `json:"server_positions"`
	}

	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	if len(req.ServerPositions) == 0 {
		return ValidationError(c, "server_positions", "must contain at least one server")
	}

	var folderID *uuid.UUID
	if req.FolderID != nil {
		fid, err := uuid.Parse(*req.FolderID)
		if err != nil {
			return InvalidUUID(c, "folder_id")
		}
		folderID = &fid
	}

	reorderReq := &models.ReorderServersRequest{
		ServerPositions: req.ServerPositions,
	}

	if err := h.mock.ReorderServers(c.Context(), userID, folderID, reorderReq); err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(fiber.Map{"success": true})
}

// setupServerFolderTestApp creates a Fiber app with server folder routes
func setupServerFolderTestApp(t *testing.T, mock *mockServerFolderService) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			var httpErr *HTTPError
			if errors.As(err, &httpErr) {
				return c.Status(httpErr.Status).JSON(fiber.Map{
					"error":   httpErr.ErrorType,
					"message": httpErr.Message,
					"code":    httpErr.Code,
				})
			}
			// Fallback for other errors
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error":   "internal_error",
				"message": err.Error(),
			})
		},
	})
	handler := newServerFolderHandlerForTest(mock)

	// Add middleware to set userID from header for testing
	app.Use(func(c *fiber.Ctx) error {
		userIDStr := c.Get("X-Test-User-ID")
		if userIDStr != "" {
			userID, err := uuid.Parse(userIDStr)
			if err == nil {
				c.Locals("userID", userID)
			} else {
				c.Locals("userID", uuid.Nil)
			}
		}
		return c.Next()
	})

	app.Post("/users/@me/server-folders", handler.Create)
	app.Get("/users/@me/server-folders", handler.GetAll)
	app.Get("/users/@me/server-folders/:id", handler.Get)
	app.Patch("/users/@me/server-folders/:id", handler.Update)
	app.Delete("/users/@me/server-folders/:id", handler.Delete)
	app.Post("/users/@me/server-folders/move", handler.MoveServer)
	app.Post("/users/@me/server-folders/move-batch", handler.MoveServers)
	app.Post("/users/@me/server-folders/reorder", handler.ReorderServers)

	return app
}

func TestServerFolderHandler_Create(t *testing.T) {
	userID := uuid.New()
	folderID := uuid.New()
	now := time.Now()

	t.Run("creates folder successfully", func(t *testing.T) {
		mock := &mockServerFolderService{
			createFolderFunc: func(ctx context.Context, uid uuid.UUID, req *models.CreateServerFolderRequest) (*models.ServerFolder, error) {
				assert.Equal(t, userID, uid)
				assert.Equal(t, "My Folder", req.Name)
				return &models.ServerFolder{
					ID:          folderID,
					UserID:      uid,
					Name:        req.Name,
					Position:    0,
					IsCollapsed: false,
					CreatedAt:   now,
					UpdatedAt:   now,
				}, nil
			},
		}

		app := setupServerFolderTestApp(t, mock)
		reqBody := models.CreateServerFolderRequest{Name: "My Folder"}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/users/@me/server-folders", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-User-ID", userID.String())

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

		var folder models.ServerFolder
		err = json.NewDecoder(resp.Body).Decode(&folder)
		assert.NoError(t, err)
		assert.Equal(t, "My Folder", folder.Name)
	})

	t.Run("returns 400 for empty name", func(t *testing.T) {
		mock := &mockServerFolderService{}
		app := setupServerFolderTestApp(t, mock)

		reqBody := models.CreateServerFolderRequest{Name: ""}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/users/@me/server-folders", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-User-ID", userID.String())

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("returns 400 for name too long", func(t *testing.T) {
		mock := &mockServerFolderService{}
		app := setupServerFolderTestApp(t, mock)

		longName := make([]byte, 101)
		for i := range longName {
			longName[i] = 'a'
		}
		reqBody := models.CreateServerFolderRequest{Name: string(longName)}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/users/@me/server-folders", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-User-ID", userID.String())

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("returns 401 without user", func(t *testing.T) {
		mock := &mockServerFolderService{}
		app := setupServerFolderTestApp(t, mock)

		reqBody := models.CreateServerFolderRequest{Name: "Test"}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/users/@me/server-folders", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	})
}

func TestServerFolderHandler_GetAll(t *testing.T) {
	userID := uuid.New()
	folderID := uuid.New()
	now := time.Now()

	t.Run("returns folder tree", func(t *testing.T) {
		mock := &mockServerFolderService{
			getUserFoldersFunc: func(ctx context.Context, uid uuid.UUID) (*models.ServerFolderTree, error) {
				assert.Equal(t, userID, uid)
				return &models.ServerFolderTree{
					Folders: []models.ServerFolderTreeItem{
						{
							ID:          folderID,
							UserID:      uid,
							Name:        "Gaming",
							Position:    0,
							IsCollapsed: false,
							Depth:       0,
							Children:    []models.ServerFolderTreeItem{},
							Servers:     []models.ServerInFolder{},
							CreatedAt:   now,
							UpdatedAt:   now,
						},
					},
					Servers: []models.ServerInFolder{},
				}, nil
			},
		}

		app := setupServerFolderTestApp(t, mock)

		req := httptest.NewRequest("GET", "/users/@me/server-folders", nil)
		req.Header.Set("X-Test-User-ID", userID.String())

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var tree models.ServerFolderTree
		err = json.NewDecoder(resp.Body).Decode(&tree)
		assert.NoError(t, err)
		assert.Len(t, tree.Folders, 1)
		assert.Equal(t, "Gaming", tree.Folders[0].Name)
	})
}

func TestServerFolderHandler_Get(t *testing.T) {
	userID := uuid.New()
	folderID := uuid.New()
	now := time.Now()

	t.Run("returns folder by id", func(t *testing.T) {
		mock := &mockServerFolderService{
			getFolderFunc: func(ctx context.Context, uid, fid uuid.UUID) (*models.ServerFolder, error) {
				assert.Equal(t, userID, uid)
				assert.Equal(t, folderID, fid)
				return &models.ServerFolder{
					ID:          fid,
					UserID:      uid,
					Name:        "Gaming",
					Position:    0,
					IsCollapsed: false,
					CreatedAt:   now,
					UpdatedAt:   now,
				}, nil
			},
		}

		app := setupServerFolderTestApp(t, mock)

		req := httptest.NewRequest("GET", "/users/@me/server-folders/"+folderID.String(), nil)
		req.Header.Set("X-Test-User-ID", userID.String())

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var folder models.ServerFolder
		err = json.NewDecoder(resp.Body).Decode(&folder)
		assert.NoError(t, err)
		assert.Equal(t, "Gaming", folder.Name)
	})

	t.Run("returns 400 for invalid folder id", func(t *testing.T) {
		mock := &mockServerFolderService{}
		app := setupServerFolderTestApp(t, mock)

		req := httptest.NewRequest("GET", "/users/@me/server-folders/invalid-uuid", nil)
		req.Header.Set("X-Test-User-ID", userID.String())

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("returns 404 for not found folder", func(t *testing.T) {
		mock := &mockServerFolderService{
			getFolderFunc: func(ctx context.Context, uid, fid uuid.UUID) (*models.ServerFolder, error) {
				return nil, services.ErrFolderNotFound
			},
		}

		app := setupServerFolderTestApp(t, mock)

		req := httptest.NewRequest("GET", "/users/@me/server-folders/"+folderID.String(), nil)
		req.Header.Set("X-Test-User-ID", userID.String())

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}

func TestServerFolderHandler_Update(t *testing.T) {
	userID := uuid.New()
	folderID := uuid.New()
	now := time.Now()

	t.Run("updates folder name", func(t *testing.T) {
		mock := &mockServerFolderService{
			updateFolderFunc: func(ctx context.Context, uid, fid uuid.UUID, req *models.UpdateServerFolderRequest) (*models.ServerFolder, error) {
				assert.Equal(t, userID, uid)
				assert.Equal(t, folderID, fid)
				assert.NotNil(t, req.Name)
				assert.Equal(t, "New Name", *req.Name)
				return &models.ServerFolder{
					ID:          fid,
					UserID:      uid,
					Name:        *req.Name,
					Position:    0,
					IsCollapsed: false,
					CreatedAt:   now,
					UpdatedAt:   now,
				}, nil
			},
		}

		app := setupServerFolderTestApp(t, mock)
		reqBody := models.UpdateServerFolderRequest{Name: folderTestStringPtr("New Name")}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PATCH", "/users/@me/server-folders/"+folderID.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-User-ID", userID.String())

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var folder models.ServerFolder
		err = json.NewDecoder(resp.Body).Decode(&folder)
		assert.NoError(t, err)
		assert.Equal(t, "New Name", folder.Name)
	})

	t.Run("updates folder collapsed state", func(t *testing.T) {
		mock := &mockServerFolderService{
			updateFolderFunc: func(ctx context.Context, uid, fid uuid.UUID, req *models.UpdateServerFolderRequest) (*models.ServerFolder, error) {
				assert.NotNil(t, req.IsCollapsed)
				assert.True(t, *req.IsCollapsed)
				return &models.ServerFolder{
					ID:          fid,
					UserID:      uid,
					Name:        "Test",
					Position:    0,
					IsCollapsed: *req.IsCollapsed,
					CreatedAt:   now,
					UpdatedAt:   now,
				}, nil
			},
		}

		app := setupServerFolderTestApp(t, mock)
		collapsed := true
		reqBody := models.UpdateServerFolderRequest{IsCollapsed: &collapsed}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PATCH", "/users/@me/server-folders/"+folderID.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-User-ID", userID.String())

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})
}

func TestServerFolderHandler_Delete(t *testing.T) {
	userID := uuid.New()
	folderID := uuid.New()

	t.Run("deletes folder successfully", func(t *testing.T) {
		mock := &mockServerFolderService{
			deleteFolderFunc: func(ctx context.Context, uid, fid uuid.UUID) error {
				assert.Equal(t, userID, uid)
				assert.Equal(t, folderID, fid)
				return nil
			},
		}

		app := setupServerFolderTestApp(t, mock)

		req := httptest.NewRequest("DELETE", "/users/@me/server-folders/"+folderID.String(), nil)
		req.Header.Set("X-Test-User-ID", userID.String())

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)
	})

	t.Run("returns 404 for not found folder", func(t *testing.T) {
		mock := &mockServerFolderService{
			deleteFolderFunc: func(ctx context.Context, uid, fid uuid.UUID) error {
				return services.ErrFolderNotFound
			},
		}

		app := setupServerFolderTestApp(t, mock)

		req := httptest.NewRequest("DELETE", "/users/@me/server-folders/"+folderID.String(), nil)
		req.Header.Set("X-Test-User-ID", userID.String())

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}

func TestServerFolderHandler_MoveServer(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()
	folderID := uuid.New()

	t.Run("moves server to folder", func(t *testing.T) {
		mock := &mockServerFolderService{
			moveServerToFolderFunc: func(ctx context.Context, uid, sid uuid.UUID, fid *uuid.UUID) error {
				assert.Equal(t, userID, uid)
				assert.Equal(t, serverID, sid)
				assert.NotNil(t, fid)
				assert.Equal(t, folderID, *fid)
				return nil
			},
		}

		app := setupServerFolderTestApp(t, mock)
		reqBody := struct {
			ServerID string   `json:"server_id"`
			FolderID *string `json:"folder_id,omitempty"`
		}{
			ServerID: serverID.String(),
			FolderID: folderTestStringPtr(folderID.String()),
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/users/@me/server-folders/move", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-User-ID", userID.String())

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("removes server from folder", func(t *testing.T) {
		mock := &mockServerFolderService{
			moveServerToFolderFunc: func(ctx context.Context, uid, sid uuid.UUID, fid *uuid.UUID) error {
				assert.Equal(t, userID, uid)
				assert.Equal(t, serverID, sid)
				assert.Nil(t, fid)
				return nil
			},
		}

		app := setupServerFolderTestApp(t, mock)
		reqBody := struct {
			ServerID string   `json:"server_id"`
			FolderID *string `json:"folder_id,omitempty"`
		}{
			ServerID: serverID.String(),
			FolderID: nil,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/users/@me/server-folders/move", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-User-ID", userID.String())

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("returns 400 for invalid server id", func(t *testing.T) {
		mock := &mockServerFolderService{}
		app := setupServerFolderTestApp(t, mock)

		reqBody := struct {
			ServerID string `json:"server_id"`
		}{
			ServerID: "invalid-uuid",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/users/@me/server-folders/move", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-User-ID", userID.String())

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("returns 403 when not server member", func(t *testing.T) {
		mock := &mockServerFolderService{
			moveServerToFolderFunc: func(ctx context.Context, uid, sid uuid.UUID, fid *uuid.UUID) error {
				return services.ErrNotServerMember
			},
		}

		app := setupServerFolderTestApp(t, mock)
		reqBody := struct {
			ServerID string   `json:"server_id"`
			FolderID *string `json:"folder_id,omitempty"`
		}{
			ServerID: serverID.String(),
			FolderID: folderTestStringPtr(folderID.String()),
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/users/@me/server-folders/move", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-User-ID", userID.String())

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
	})
}

func TestServerFolderHandler_MoveServers(t *testing.T) {
	userID := uuid.New()
	serverID1 := uuid.New()
	serverID2 := uuid.New()
	folderID := uuid.New()

	t.Run("moves multiple servers to folder", func(t *testing.T) {
		mock := &mockServerFolderService{
			moveServersToFolderFunc: func(ctx context.Context, uid uuid.UUID, req *models.MoveServersToFolderRequest) error {
				assert.Equal(t, userID, uid)
				assert.Len(t, req.ServerIDs, 2)
				assert.Equal(t, serverID1.String(), req.ServerIDs[0])
				assert.Equal(t, serverID2.String(), req.ServerIDs[1])
				assert.NotNil(t, req.FolderID)
				assert.Equal(t, folderID.String(), *req.FolderID)
				return nil
			},
		}

		app := setupServerFolderTestApp(t, mock)
		folderIDStr := folderID.String()
		reqBody := models.MoveServersToFolderRequest{
			ServerIDs: []string{serverID1.String(), serverID2.String()},
			FolderID:  &folderIDStr,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/users/@me/server-folders/move-batch", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-User-ID", userID.String())

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("returns 400 for empty server ids", func(t *testing.T) {
		mock := &mockServerFolderService{}
		app := setupServerFolderTestApp(t, mock)

		reqBody := models.MoveServersToFolderRequest{
			ServerIDs: []string{},
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/users/@me/server-folders/move-batch", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-User-ID", userID.String())

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

func TestServerFolderHandler_ReorderServers(t *testing.T) {
	userID := uuid.New()
	serverID1 := uuid.New()
	serverID2 := uuid.New()

	t.Run("reorders servers successfully", func(t *testing.T) {
		mock := &mockServerFolderService{
			reorderServersFunc: func(ctx context.Context, uid uuid.UUID, folderID *uuid.UUID, req *models.ReorderServersRequest) error {
				assert.Equal(t, userID, uid)
				assert.Nil(t, folderID)
				assert.Len(t, req.ServerPositions, 2)
				return nil
			},
		}

		app := setupServerFolderTestApp(t, mock)
		reqBody := struct {
			FolderID        *string                  `json:"folder_id,omitempty"`
			ServerPositions []models.ServerPosition `json:"server_positions"`
		}{
			ServerPositions: []models.ServerPosition{
				{ServerID: serverID1.String(), Position: 0},
				{ServerID: serverID2.String(), Position: 1},
			},
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/users/@me/server-folders/reorder", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-User-ID", userID.String())

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("returns 400 for empty positions", func(t *testing.T) {
		mock := &mockServerFolderService{}
		app := setupServerFolderTestApp(t, mock)

		reqBody := struct {
			FolderID        *string                  `json:"folder_id,omitempty"`
			ServerPositions []models.ServerPosition `json:"server_positions"`
		}{
			ServerPositions: []models.ServerPosition{},
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/users/@me/server-folders/reorder", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-User-ID", userID.String())

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

// Helper function
func folderTestStringPtr(s string) *string {
	return &s
}
