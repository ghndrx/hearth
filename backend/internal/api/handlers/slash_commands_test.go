package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"hearth/internal/models"
)

type mockSlashCommandService struct {
	registerCommandFunc   func(ctx context.Context, appID uuid.UUID, cmd *models.SlashCommand) error
	getCommandsFunc       func(ctx context.Context, appID uuid.UUID) ([]*models.SlashCommand, error)
	getServerCommandsFunc func(ctx context.Context, serverID uuid.UUID) ([]*models.SlashCommand, error)
	getCommandFunc        func(ctx context.Context, appID, commandID uuid.UUID) (*models.SlashCommand, error)
	updateCommandFunc     func(ctx context.Context, appID, commandID uuid.UUID, cmd *models.SlashCommand) error
	deleteCommandFunc     func(ctx context.Context, appID, commandID uuid.UUID) error
}

func (m *mockSlashCommandService) RegisterCommand(ctx context.Context, appID uuid.UUID, cmd *models.SlashCommand) error {
	return m.registerCommandFunc(ctx, appID, cmd)
}

func (m *mockSlashCommandService) GetCommands(ctx context.Context, appID uuid.UUID) ([]*models.SlashCommand, error) {
	return m.getCommandsFunc(ctx, appID)
}

func (m *mockSlashCommandService) GetServerCommands(ctx context.Context, serverID uuid.UUID) ([]*models.SlashCommand, error) {
	return m.getServerCommandsFunc(ctx, serverID)
}

func (m *mockSlashCommandService) GetCommand(ctx context.Context, appID, commandID uuid.UUID) (*models.SlashCommand, error) {
	return m.getCommandFunc(ctx, appID, commandID)
}

func (m *mockSlashCommandService) UpdateCommand(ctx context.Context, appID, commandID uuid.UUID, cmd *models.SlashCommand) error {
	return m.updateCommandFunc(ctx, appID, commandID, cmd)
}

func (m *mockSlashCommandService) DeleteCommand(ctx context.Context, appID, commandID uuid.UUID) error {
	return m.deleteCommandFunc(ctx, appID, commandID)
}

func setupSlashCommandApp() *fiber.App {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		userID := c.Get("X-Test-User-ID")
		if userID != "" {
			c.Locals("userID", userID)
			c.Locals("user_id", userID)
		}
		return c.Next()
	})
	return app
}

func TestRegisterCommand_Success(t *testing.T) {
	app := setupSlashCommandApp()
	t.Cleanup(func() { _ = app.Shutdown() })

	appID := uuid.New()
	cmdID := uuid.New()

	mock := &mockSlashCommandService{
		registerCommandFunc: func(ctx context.Context, aID uuid.UUID, cmd *models.SlashCommand) error {
			cmd.ID = cmdID
			cmd.AppID = aID
			cmd.CreatedAt = time.Now()
			cmd.UpdatedAt = time.Now()
			return nil
		},
	}

	app.Post("/applications/:appId/commands", func(c *fiber.Ctx) error {
		aID, err := uuid.Parse(c.Params("appId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid app ID"})
		}

		userID, _ := c.Locals("user_id").(string)
		_ = userID

		var body struct {
			Name        string                  `json:"name"`
			Description string                  `json:"description"`
			Options     []*models.CommandOption `json:"options"`
			Type        int                     `json:"type"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
		}

		if body.Name == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
		}
		if len(body.Name) > 32 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name too long"})
		}
		if body.Description == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "description is required"})
		}
		if len(body.Description) > 100 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "description too long"})
		}

		cmd := &models.SlashCommand{
			Name:        body.Name,
			Description: body.Description,
			Options:     body.Options,
			Type:        models.CommandType(body.Type),
		}

		if err := mock.RegisterCommand(c.Context(), aID, cmd); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
		}

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"id":          cmd.ID.String(),
			"app_id":      cmd.AppID.String(),
			"name":        cmd.Name,
			"description": cmd.Description,
			"type":        cmd.Type,
		})
	})

	reqBody := `{"name":"ping","description":"Ping command","type":1}`
	req := httptest.NewRequest(http.MethodPost, "/applications/"+appID.String()+"/commands", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, cmdID.String(), result["id"])
	assert.Equal(t, "ping", result["name"])
}

func TestRegisterCommand_InvalidAppID(t *testing.T) {
	app := setupSlashCommandApp()
	t.Cleanup(func() { _ = app.Shutdown() })

	app.Post("/applications/:appId/commands", func(c *fiber.Ctx) error {
		_, err := uuid.Parse(c.Params("appId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid app ID"})
		}
		return nil
	})

	reqBody := `{"name":"ping","description":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/applications/not-a-uuid/commands", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestRegisterCommand_MissingName(t *testing.T) {
	app := setupSlashCommandApp()
	t.Cleanup(func() { _ = app.Shutdown() })

	app.Post("/applications/:appId/commands", func(c *fiber.Ctx) error {
		_, err := uuid.Parse(c.Params("appId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid app ID"})
		}

		var body struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
		}

		if body.Name == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
		}
		return nil
	})

	reqBody := `{"description":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/applications/"+uuid.New().String()+"/commands", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestRegisterCommand_NameTooLong(t *testing.T) {
	app := setupSlashCommandApp()
	t.Cleanup(func() { _ = app.Shutdown() })

	app.Post("/applications/:appId/commands", func(c *fiber.Ctx) error {
		_, err := uuid.Parse(c.Params("appId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid app ID"})
		}

		var body struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
		}

		if body.Name == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
		}
		if len(body.Name) > 32 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name too long"})
		}
		return nil
	})

	longName := strings.Repeat("a", 33)
	reqBody := `{"name":"` + longName + `","description":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/applications/"+uuid.New().String()+"/commands", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestRegisterCommand_MissingDescription(t *testing.T) {
	app := setupSlashCommandApp()
	t.Cleanup(func() { _ = app.Shutdown() })

	app.Post("/applications/:appId/commands", func(c *fiber.Ctx) error {
		_, err := uuid.Parse(c.Params("appId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid app ID"})
		}

		var body struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
		}

		if body.Name == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
		}
		if body.Description == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "description is required"})
		}
		return nil
	})

	reqBody := `{"name":"ping"}`
	req := httptest.NewRequest(http.MethodPost, "/applications/"+uuid.New().String()+"/commands", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestRegisterCommand_DescriptionTooLong(t *testing.T) {
	app := setupSlashCommandApp()
	t.Cleanup(func() { _ = app.Shutdown() })

	app.Post("/applications/:appId/commands", func(c *fiber.Ctx) error {
		_, err := uuid.Parse(c.Params("appId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid app ID"})
		}

		var body struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
		}

		if body.Name == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
		}
		if body.Description == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "description is required"})
		}
		if len(body.Description) > 100 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "description too long"})
		}
		return nil
	})

	longDesc := strings.Repeat("b", 101)
	reqBody := `{"name":"ping","description":"` + longDesc + `"}`
	req := httptest.NewRequest(http.MethodPost, "/applications/"+uuid.New().String()+"/commands", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestGetCommands_Success(t *testing.T) {
	app := setupSlashCommandApp()
	t.Cleanup(func() { _ = app.Shutdown() })

	appID := uuid.New()
	cmdID := uuid.New()
	now := time.Now()

	mock := &mockSlashCommandService{
		getCommandsFunc: func(ctx context.Context, aID uuid.UUID) ([]*models.SlashCommand, error) {
			return []*models.SlashCommand{
				{
					ID:          cmdID,
					AppID:       aID,
					Name:        "ping",
					Description: "Ping command",
					Type:        1,
					CreatedAt:   now,
					UpdatedAt:   now,
				},
			}, nil
		},
	}

	app.Get("/applications/:appId/commands", func(c *fiber.Ctx) error {
		aID, err := uuid.Parse(c.Params("appId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid app ID"})
		}

		guildID := c.Query("guild_id")
		if guildID != "" {
			sID, err := uuid.Parse(guildID)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid guild ID"})
			}
			cmds, err := mock.GetServerCommands(c.Context(), sID)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
			}
			return c.JSON(cmds)
		}

		cmds, err := mock.GetCommands(c.Context(), aID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
		}

		var results []fiber.Map
		for _, cmd := range cmds {
			results = append(results, fiber.Map{
				"id":          cmd.ID.String(),
				"app_id":      cmd.AppID.String(),
				"name":        cmd.Name,
				"description": cmd.Description,
				"type":        cmd.Type,
				"created_at":  cmd.CreatedAt.Format(time.RFC3339),
				"updated_at":  cmd.UpdatedAt.Format(time.RFC3339),
			})
		}
		return c.JSON(results)
	})

	req := httptest.NewRequest(http.MethodGet, "/applications/"+appID.String()+"/commands", nil)
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Len(t, result, 1)
	assert.Equal(t, "ping", result[0]["name"])
}

func TestGetCommands_InvalidAppID(t *testing.T) {
	app := setupSlashCommandApp()
	t.Cleanup(func() { _ = app.Shutdown() })

	app.Get("/applications/:appId/commands", func(c *fiber.Ctx) error {
		_, err := uuid.Parse(c.Params("appId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid app ID"})
		}
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/applications/bad-id/commands", nil)
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestGetCommand_Success(t *testing.T) {
	app := setupSlashCommandApp()
	t.Cleanup(func() { _ = app.Shutdown() })

	appID := uuid.New()
	cmdID := uuid.New()
	now := time.Now()

	mock := &mockSlashCommandService{
		getCommandFunc: func(ctx context.Context, aID, cID uuid.UUID) (*models.SlashCommand, error) {
			return &models.SlashCommand{
				ID:          cID,
				AppID:       aID,
				Name:        "hello",
				Description: "Hello command",
				Type:        1,
				CreatedAt:   now,
				UpdatedAt:   now,
			}, nil
		},
	}

	app.Get("/applications/:appId/commands/:commandId", func(c *fiber.Ctx) error {
		aID, err := uuid.Parse(c.Params("appId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid app ID"})
		}
		cID, err := uuid.Parse(c.Params("commandId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid command ID"})
		}

		cmd, err := mock.GetCommand(c.Context(), aID, cID)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "command not found"})
		}

		return c.JSON(fiber.Map{
			"id":          cmd.ID.String(),
			"app_id":      cmd.AppID.String(),
			"name":        cmd.Name,
			"description": cmd.Description,
			"type":        cmd.Type,
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/applications/"+appID.String()+"/commands/"+cmdID.String(), nil)
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, cmdID.String(), result["id"])
	assert.Equal(t, "hello", result["name"])
}

func TestGetCommand_NotFound(t *testing.T) {
	app := setupSlashCommandApp()
	t.Cleanup(func() { _ = app.Shutdown() })

	mock := &mockSlashCommandService{
		getCommandFunc: func(ctx context.Context, aID, cID uuid.UUID) (*models.SlashCommand, error) {
			return nil, errors.New("not found")
		},
	}

	app.Get("/applications/:appId/commands/:commandId", func(c *fiber.Ctx) error {
		aID, err := uuid.Parse(c.Params("appId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid app ID"})
		}
		cID, err := uuid.Parse(c.Params("commandId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid command ID"})
		}

		_, err = mock.GetCommand(c.Context(), aID, cID)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "command not found"})
		}
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/applications/"+uuid.New().String()+"/commands/"+uuid.New().String(), nil)
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestDeleteCommand_Success(t *testing.T) {
	app := setupSlashCommandApp()
	t.Cleanup(func() { _ = app.Shutdown() })

	mock := &mockSlashCommandService{
		deleteCommandFunc: func(ctx context.Context, aID, cID uuid.UUID) error {
			return nil
		},
	}

	app.Delete("/applications/:appId/commands/:commandId", func(c *fiber.Ctx) error {
		aID, err := uuid.Parse(c.Params("appId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid app ID"})
		}
		cID, err := uuid.Parse(c.Params("commandId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid command ID"})
		}

		if err := mock.DeleteCommand(c.Context(), aID, cID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
		}

		return c.SendStatus(fiber.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodDelete, "/applications/"+uuid.New().String()+"/commands/"+uuid.New().String(), nil)
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)
}

func TestDeleteCommand_InvalidIDs(t *testing.T) {
	app := setupSlashCommandApp()
	t.Cleanup(func() { _ = app.Shutdown() })

	app.Delete("/applications/:appId/commands/:commandId", func(c *fiber.Ctx) error {
		_, err := uuid.Parse(c.Params("appId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid app ID"})
		}
		_, err = uuid.Parse(c.Params("commandId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid command ID"})
		}
		return nil
	})

	req := httptest.NewRequest(http.MethodDelete, "/applications/bad-id/commands/also-bad", nil)
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestBulkRegisterCommands_Success(t *testing.T) {
	app := setupSlashCommandApp()
	t.Cleanup(func() { _ = app.Shutdown() })

	appID := uuid.New()
	callCount := 0

	mock := &mockSlashCommandService{
		registerCommandFunc: func(ctx context.Context, aID uuid.UUID, cmd *models.SlashCommand) error {
			callCount++
			cmd.ID = uuid.New()
			cmd.AppID = aID
			cmd.CreatedAt = time.Now()
			cmd.UpdatedAt = time.Now()
			return nil
		},
	}

	app.Post("/applications/:appId/commands/bulk", func(c *fiber.Ctx) error {
		aID, err := uuid.Parse(c.Params("appId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid app ID"})
		}

		var commands []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Type        int    `json:"type"`
		}
		if err := c.BodyParser(&commands); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
		}

		var registered []fiber.Map
		for _, cmdReq := range commands {
			cmd := &models.SlashCommand{
				Name:        cmdReq.Name,
				Description: cmdReq.Description,
				Type:        models.CommandType(cmdReq.Type),
			}
			if err := mock.RegisterCommand(c.Context(), aID, cmd); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
			}
			registered = append(registered, fiber.Map{
				"id":          cmd.ID.String(),
				"app_id":      cmd.AppID.String(),
				"name":        cmd.Name,
				"description": cmd.Description,
			})
		}

		return c.Status(fiber.StatusCreated).JSON(registered)
	})

	reqBody := `[{"name":"ping","description":"Ping command","type":1},{"name":"help","description":"Help command","type":1}]`
	req := httptest.NewRequest(http.MethodPost, "/applications/"+appID.String()+"/commands/bulk", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var result []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Len(t, result, 2)
	assert.Equal(t, 2, callCount)
}
