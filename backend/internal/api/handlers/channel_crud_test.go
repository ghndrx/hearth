package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// channelCRUDMock mocks ChannelService for channel CRUD handler tests
type channelCRUDMock struct {
	getChannelFunc    func(ctx context.Context, id uuid.UUID) (*models.Channel, error)
	updateChannelFunc func(ctx context.Context, id, requesterID uuid.UUID, update *models.ChannelUpdate) (*models.Channel, error)
	deleteChannelFunc func(ctx context.Context, id, requesterID uuid.UUID) error
}

func (m *channelCRUDMock) GetChannel(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
	if m.getChannelFunc != nil {
		return m.getChannelFunc(ctx, id)
	}
	return nil, nil
}

func (m *channelCRUDMock) UpdateChannel(ctx context.Context, id, requesterID uuid.UUID, update *models.ChannelUpdate) (*models.Channel, error) {
	if m.updateChannelFunc != nil {
		return m.updateChannelFunc(ctx, id, requesterID, update)
	}
	return nil, nil
}

func (m *channelCRUDMock) DeleteChannel(ctx context.Context, id, requesterID uuid.UUID) error {
	if m.deleteChannelFunc != nil {
		return m.deleteChannelFunc(ctx, id, requesterID)
	}
	return nil
}

// setupCRUDTestApp creates a test Fiber app with channel CRUD routes
func setupCRUDTestApp(channelService *channelCRUDMock) *fiber.App {
	app := fiber.New()

	// Inject userID middleware
	app.Use(func(c *fiber.Ctx) error {
		userID := c.Get("X-User-ID")
		if userID != "" {
			id, _ := uuid.Parse(userID)
			c.Locals("userID", id)
		}
		return c.Next()
	})

	// Get channel
	app.Get("/channels/:id", func(c *fiber.Ctx) error {
		channelID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid channel id",
			})
		}

		channel, err := channelService.GetChannel(c.Context(), channelID)
		if err != nil {
			if err == services.ErrChannelNotFound {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": "channel not found",
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to get channel",
			})
		}

		return c.JSON(channel)
	})

	// Update channel
	app.Patch("/channels/:id", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		channelID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid channel id",
			})
		}

		var req models.UpdateChannelRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid request body",
			})
		}

		update := &models.ChannelUpdate{
			Name:     req.Name,
			Topic:    req.Topic,
			Position: req.Position,
			NSFW:     req.NSFW,
			Slowmode: req.SlowmodeSeconds,
		}

		channel, err := channelService.UpdateChannel(c.Context(), channelID, userID, update)
		if err != nil {
			switch err {
			case services.ErrChannelNotFound:
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": "channel not found",
				})
			case services.ErrNotServerMember:
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "not a server member",
				})
			default:
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "failed to update channel",
				})
			}
		}

		return c.JSON(channel)
	})

	// Delete channel
	app.Delete("/channels/:id", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		channelID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid channel id",
			})
		}

		err = channelService.DeleteChannel(c.Context(), channelID, userID)
		if err != nil {
			switch err {
			case services.ErrChannelNotFound:
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": "channel not found",
				})
			case services.ErrNotServerMember:
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "not a server member",
				})
			case services.ErrCannotDeleteDM:
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "cannot delete DM channel",
				})
			default:
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "failed to delete channel",
				})
			}
		}

		return c.SendStatus(fiber.StatusNoContent)
	})

	return app
}

// GetChannel Tests

func TestCRUD_Get_Success(t *testing.T) {
	channelID := uuid.New()
	serverID := uuid.New()

	expectedChannel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Name:     "general",
		Type:     models.ChannelTypeText,
		Topic:    "General discussion",
		Position: 0,
	}

	svc := &channelCRUDMock{
		getChannelFunc: func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
			if id == channelID {
				return expectedChannel, nil
			}
			return nil, services.ErrChannelNotFound
		},
	}

	app := setupCRUDTestApp(svc)

	req := httptest.NewRequest("GET", "/channels/"+channelID.String(), nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	var channel models.Channel
	if err := json.NewDecoder(resp.Body).Decode(&channel); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if channel.ID != channelID {
		t.Errorf("Expected channel ID %s, got %s", channelID, channel.ID)
	}
	if channel.Name != "general" {
		t.Errorf("Expected channel name 'general', got '%s'", channel.Name)
	}
}

func TestCRUD_Get_NotFound(t *testing.T) {
	channelID := uuid.New()

	svc := &channelCRUDMock{
		getChannelFunc: func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
			return nil, services.ErrChannelNotFound
		},
	}

	app := setupCRUDTestApp(svc)

	req := httptest.NewRequest("GET", "/channels/"+channelID.String(), nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestCRUD_Get_InvalidID(t *testing.T) {
	svc := &channelCRUDMock{}
	app := setupCRUDTestApp(svc)

	req := httptest.NewRequest("GET", "/channels/invalid-uuid", nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

// UpdateChannel Tests

func TestCRUD_Update_Success(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()
	newName := "updated-channel"
	newTopic := "Updated topic"

	svc := &channelCRUDMock{
		updateChannelFunc: func(ctx context.Context, id, requesterID uuid.UUID, update *models.ChannelUpdate) (*models.Channel, error) {
			if id == channelID && requesterID == userID {
				return &models.Channel{
					ID:       channelID,
					ServerID: &serverID,
					Name:     *update.Name,
					Topic:    *update.Topic,
					Type:     models.ChannelTypeText,
				}, nil
			}
			return nil, errors.New("unexpected params")
		},
	}

	app := setupCRUDTestApp(svc)

	reqBody := models.UpdateChannelRequest{
		Name:  &newName,
		Topic: &newTopic,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("PATCH", "/channels/"+channelID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		bodyResp, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected 200, got %d: %s", resp.StatusCode, string(bodyResp))
	}

	var channel models.Channel
	if err := json.NewDecoder(resp.Body).Decode(&channel); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if channel.Name != newName {
		t.Errorf("Expected name '%s', got '%s'", newName, channel.Name)
	}
	if channel.Topic != newTopic {
		t.Errorf("Expected topic '%s', got '%s'", newTopic, channel.Topic)
	}
}

func TestCRUD_Update_NotFound(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	svc := &channelCRUDMock{
		updateChannelFunc: func(ctx context.Context, id, requesterID uuid.UUID, update *models.ChannelUpdate) (*models.Channel, error) {
			return nil, services.ErrChannelNotFound
		},
	}

	app := setupCRUDTestApp(svc)

	newName := "updated-channel"
	reqBody := models.UpdateChannelRequest{Name: &newName}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("PATCH", "/channels/"+channelID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestCRUD_Update_NotServerMember(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	svc := &channelCRUDMock{
		updateChannelFunc: func(ctx context.Context, id, requesterID uuid.UUID, update *models.ChannelUpdate) (*models.Channel, error) {
			return nil, services.ErrNotServerMember
		},
	}

	app := setupCRUDTestApp(svc)

	newName := "updated-channel"
	reqBody := models.UpdateChannelRequest{Name: &newName}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("PATCH", "/channels/"+channelID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestCRUD_Update_InvalidBody(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	svc := &channelCRUDMock{}
	app := setupCRUDTestApp(svc)

	req := httptest.NewRequest("PATCH", "/channels/"+channelID.String(), bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

// DeleteChannel Tests

func TestCRUD_Delete_Success(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	svc := &channelCRUDMock{
		deleteChannelFunc: func(ctx context.Context, id, requesterID uuid.UUID) error {
			if id == channelID && requesterID == userID {
				return nil
			}
			return errors.New("unexpected params")
		},
	}

	app := setupCRUDTestApp(svc)

	req := httptest.NewRequest("DELETE", "/channels/"+channelID.String(), nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected 204, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestCRUD_Delete_NotFound(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	svc := &channelCRUDMock{
		deleteChannelFunc: func(ctx context.Context, id, requesterID uuid.UUID) error {
			return services.ErrChannelNotFound
		},
	}

	app := setupCRUDTestApp(svc)

	req := httptest.NewRequest("DELETE", "/channels/"+channelID.String(), nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestCRUD_Delete_NotServerMember(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	svc := &channelCRUDMock{
		deleteChannelFunc: func(ctx context.Context, id, requesterID uuid.UUID) error {
			return services.ErrNotServerMember
		},
	}

	app := setupCRUDTestApp(svc)

	req := httptest.NewRequest("DELETE", "/channels/"+channelID.String(), nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestCRUD_Delete_CannotDeleteDM(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	svc := &channelCRUDMock{
		deleteChannelFunc: func(ctx context.Context, id, requesterID uuid.UUID) error {
			return services.ErrCannotDeleteDM
		},
	}

	app := setupCRUDTestApp(svc)

	req := httptest.NewRequest("DELETE", "/channels/"+channelID.String(), nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestCRUD_Delete_InvalidID(t *testing.T) {
	userID := uuid.New()
	svc := &channelCRUDMock{}
	app := setupCRUDTestApp(svc)

	req := httptest.NewRequest("DELETE", "/channels/invalid-uuid", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}
