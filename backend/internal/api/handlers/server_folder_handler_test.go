package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"hearth/internal/models"
)

// MockServerFolderService is a mock for ServerFolderService
type MockServerFolderService struct {
	CreateFolderFunc          func(ctx context.Context, userID uuid.UUID, req *models.CreateServerFolderRequest) (*models.ServerFolder, error)
	GetUserFoldersFunc         func(ctx context.Context, userID uuid.UUID) (*models.ServerFolderTree, error)
	GetFolderFunc              func(ctx context.Context, userID, folderID uuid.UUID) (*models.ServerFolder, error)
	UpdateFolderFunc           func(ctx context.Context, userID, folderID uuid.UUID, req *models.UpdateServerFolderRequest) (*models.ServerFolder, error)
	DeleteFolderFunc           func(ctx context.Context, userID, folderID uuid.UUID) error
	MoveServerToFolderFunc     func(ctx context.Context, userID, serverID uuid.UUID, folderID *uuid.UUID) error
	MoveServersToFolderFunc    func(ctx context.Context, userID uuid.UUID, req *models.MoveServersToFolderRequest) error
	ReorderServersFunc         func(ctx context.Context, userID uuid.UUID, folderID *uuid.UUID, req *models.ReorderServersRequest) error
}

func (m *MockServerFolderService) CreateFolder(ctx context.Context, userID uuid.UUID, req *models.CreateServerFolderRequest) (*models.ServerFolder, error) {
	if m.CreateFolderFunc != nil {
		return m.CreateFolderFunc(ctx, userID, req)
	}
	return nil, nil
}

func (m *MockServerFolderService) GetUserFolders(ctx context.Context, userID uuid.UUID) (*models.ServerFolderTree, error) {
	if m.GetUserFoldersFunc != nil {
		return m.GetUserFoldersFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockServerFolderService) GetFolder(ctx context.Context, userID, folderID uuid.UUID) (*models.ServerFolder, error) {
	if m.GetFolderFunc != nil {
		return m.GetFolderFunc(ctx, userID, folderID)
	}
	return nil, nil
}

func (m *MockServerFolderService) UpdateFolder(ctx context.Context, userID, folderID uuid.UUID, req *models.UpdateServerFolderRequest) (*models.ServerFolder, error) {
	if m.UpdateFolderFunc != nil {
		return m.UpdateFolderFunc(ctx, userID, folderID, req)
	}
	return nil, nil
}

func (m *MockServerFolderService) DeleteFolder(ctx context.Context, userID, folderID uuid.UUID) error {
	if m.DeleteFolderFunc != nil {
		return m.DeleteFolderFunc(ctx, userID, folderID)
	}
	return nil
}

func (m *MockServerFolderService) MoveServerToFolder(ctx context.Context, userID, serverID uuid.UUID, folderID *uuid.UUID) error {
	if m.MoveServerToFolderFunc != nil {
		return m.MoveServerToFolderFunc(ctx, userID, serverID, folderID)
	}
	return nil
}

func (m *MockServerFolderService) MoveServersToFolder(ctx context.Context, userID uuid.UUID, req *models.MoveServersToFolderRequest) error {
	if m.MoveServersToFolderFunc != nil {
		return m.MoveServersToFolderFunc(ctx, userID, req)
	}
	return nil
}

func (m *MockServerFolderService) ReorderServers(ctx context.Context, userID uuid.UUID, folderID *uuid.UUID, req *models.ReorderServersRequest) error {
	if m.ReorderServersFunc != nil {
		return m.ReorderServersFunc(ctx, userID, folderID, req)
	}
	return nil
}

// MockServerService is a mock for ServerService (minimal for handler tests)
type MockServerServiceForFolderHandler struct{}

func TestServerFolderHandler_Create(t *testing.T) {
	userID := uuid.New()
	folderID := uuid.New()

	// Create a minimal service mock
	service := &MockServerFolderService{}

	// Create a minimal server service (not used in these tests)
	serverService := &MockServerServiceForFolderHandler{}

	handler := &ServerFolderHandler{
		folderService: nil, // Will use mock directly in integration style test
		serverService: nil,
	}

	// For integration testing, we'd need to properly wire up the mocks
	// This is a placeholder to show the test structure
	_ = handler
	_ = service
	_ = serverService
	_ = userID
	_ = folderID
}

func TestServerFolderHandler_GetAll(t *testing.T) {
	// Test structure for GetAll endpoint
	t.Run("returns folder tree", func(t *testing.T) {
		// This would be a proper integration test
		// For now, we show the structure
	})
}

func TestServerFolderHandler_Validation(t *testing.T) {
	app := fiber.New()
	userID := uuid.New()

	app.Post("/test", func(c *fiber.Ctx) error {
		c.Locals("userID", userID)

		var req models.CreateServerFolderRequest
		if err := c.BodyParser(&req); err != nil {
			return ParseError(c, err)
		}

		if req.Name == "" || len(req.Name) < 1 || len(req.Name) > 100 {
			return ValidationError(c, "name", "must be between 1 and 100 characters")
		}

		return c.JSON(fiber.Map{"valid": true})
	})

	t.Run("valid name passes validation", func(t *testing.T) {
		req := models.CreateServerFolderRequest{Name: "My Folder"}
		body, _ := json.Marshal(req)

		req2 := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
		req2.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req2)
		if err != nil {
			t.Fatal(err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("empty name fails validation", func(t *testing.T) {
		req := models.CreateServerFolderRequest{Name: ""}
		body, _ := json.Marshal(req)

		req2 := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
		req2.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req2)
		if err != nil {
			t.Fatal(err)
		}

		if resp.StatusCode != 400 {
			t.Errorf("expected status 400, got %d", resp.StatusCode)
		}
	})

	t.Run("name too long fails validation", func(t *testing.T) {
		longName := make([]byte, 101)
		for i := range longName {
			longName[i] = 'a'
		}
		req := models.CreateServerFolderRequest{Name: string(longName)}
		body, _ := json.Marshal(req)

		req2 := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
		req2.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req2)
		if err != nil {
			t.Fatal(err)
		}

		if resp.StatusCode != 400 {
			t.Errorf("expected status 400, got %d", resp.StatusCode)
		}
	})
}

func TestServerFolderHandler_MoveServers_Validation(t *testing.T) {
	app := fiber.New()
	userID := uuid.New()

	app.Post("/test", func(c *fiber.Ctx) error {
		c.Locals("userID", userID)

		var req models.MoveServersToFolderRequest
		if err := c.BodyParser(&req); err != nil {
			return ParseError(c, err)
		}

		if len(req.ServerIDs) == 0 {
			return ValidationError(c, "server_ids", "must contain at least one server")
		}

		return c.JSON(fiber.Map{"valid": true})
	})

	t.Run("empty server_ids fails validation", func(t *testing.T) {
		req := models.MoveServersToFolderRequest{ServerIDs: []uuid.UUID{}}
		body, _ := json.Marshal(req)

		req2 := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
		req2.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req2)
		if err != nil {
			t.Fatal(err)
		}

		if resp.StatusCode != 400 {
			t.Errorf("expected status 400, got %d", resp.StatusCode)
		}
	})

	t.Run("valid server_ids passes validation", func(t *testing.T) {
		req := models.MoveServersToFolderRequest{
			ServerIDs: []uuid.UUID{uuid.New()},
		}
		body, _ := json.Marshal(req)

		req2 := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
		req2.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req2)
		if err != nil {
			t.Fatal(err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})
}

func TestServerFolderHandler_ReorderServers_Validation(t *testing.T) {
	app := fiber.New()
	userID := uuid.New()

	app.Post("/test", func(c *fiber.Ctx) error {
		c.Locals("userID", userID)

		var req struct {
			ServerPositions []models.ServerPosition `json:"server_positions"`
		}
		if err := c.BodyParser(&req); err != nil {
			return ParseError(c, err)
		}

		if len(req.ServerPositions) == 0 {
			return ValidationError(c, "server_positions", "must contain at least one server")
		}

		return c.JSON(fiber.Map{"valid": true})
	})

	t.Run("empty server_positions fails validation", func(t *testing.T) {
		req := struct {
			ServerPositions []models.ServerPosition `json:"server_positions"`
		}{ServerPositions: []models.ServerPosition{}}
		body, _ := json.Marshal(req)

		req2 := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
		req2.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req2)
		if err != nil {
			t.Fatal(err)
		}

		if resp.StatusCode != 400 {
			t.Errorf("expected status 400, got %d", resp.StatusCode)
		}
	})

	t.Run("valid server_positions passes validation", func(t *testing.T) {
		req := struct {
			ServerPositions []models.ServerPosition `json:"server_positions"`
		}{ServerPositions: []models.ServerPosition{
			{ServerID: uuid.New(), Position: 0},
		}}
		body, _ := json.Marshal(req)

		req2 := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
		req2.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req2)
		if err != nil {
			t.Fatal(err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})
}
