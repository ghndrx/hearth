package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"hearth/internal/models"
	"hearth/internal/services"
)

type mockAnnouncementService struct {
	followChannelFunc    func(ctx context.Context, sourceChannelID, targetChannelID, userID uuid.UUID) (*models.Webhook, error)
	unfollowChannelFunc  func(ctx context.Context, sourceChannelID, followerWebhookID, userID uuid.UUID) error
	getFollowersFunc     func(ctx context.Context, channelID uuid.UUID) ([]*models.Webhook, error)
	crosspostMessageFunc func(ctx context.Context, channelID, messageID, userID uuid.UUID) ([]*models.Message, error)
}

func setupAnnouncementTestApp(mock *mockAnnouncementService) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{"error": err.Error()})
		},
	})

	app.Use(func(c *fiber.Ctx) error {
		userID := c.Get("X-Test-User-ID")
		if userID != "" {
			uid, err := uuid.Parse(userID)
			if err == nil {
				c.Locals("userID", uid)
			}
		}
		return c.Next()
	})

	// POST /channels/:channelID/followers
	app.Post("/channels/:channelID/followers", func(c *fiber.Ctx) error {
		channelIDStr := c.Params("channelID")
		channelID, err := uuid.Parse(channelIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid channel ID"})
		}

		userID, ok := c.Locals("userID").(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}

		var body struct {
			WebhookChannelID string `json:"webhook_channel_id"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}

		if body.WebhookChannelID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "webhook_channel_id is required"})
		}

		targetChannelID, err := uuid.Parse(body.WebhookChannelID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid webhook_channel_id"})
		}

		webhook, err := mock.followChannelFunc(c.Context(), channelID, targetChannelID, userID)
		if err != nil {
			switch {
			case errors.Is(err, services.ErrChannelNotFound):
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "channel not found"})
			case errors.Is(err, services.ErrNotAnnouncementChannel):
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "not an announcement channel"})
			case errors.Is(err, services.ErrCannotFollowOwnChannel):
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot follow own channel"})
			case errors.Is(err, services.ErrNotServerMember):
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "not a server member"})
			case errors.Is(err, services.ErrNoPermission):
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "no permission"})
			default:
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
			}
		}

		resp := FollowChannelResponse{
			ID:        webhook.ID.String(),
			Name:      webhook.Name,
			ChannelID: webhook.ChannelID.String(),
			Token:     webhook.Token,
			Type:      int(webhook.Type),
		}
		if webhook.ServerID != nil {
			resp.GuildID = webhook.ServerID.String()
		}
		if webhook.SourceChannelID != nil {
			resp.SourceChannelID = webhook.SourceChannelID.String()
		}
		if webhook.SourceServerID != nil {
			resp.SourceGuildID = webhook.SourceServerID.String()
		}

		return c.Status(fiber.StatusCreated).JSON(resp)
	})

	// DELETE /channels/:channelID/followers/:webhookID
	app.Delete("/channels/:channelID/followers/:webhookID", func(c *fiber.Ctx) error {
		channelIDStr := c.Params("channelID")
		channelID, err := uuid.Parse(channelIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid channel ID"})
		}

		webhookIDStr := c.Params("webhookID")
		webhookID, err := uuid.Parse(webhookIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid webhook ID"})
		}

		userID, ok := c.Locals("userID").(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}

		err = mock.unfollowChannelFunc(c.Context(), channelID, webhookID, userID)
		if err != nil {
			switch {
			case errors.Is(err, services.ErrChannelNotFound):
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "channel not found"})
			case errors.Is(err, services.ErrWebhookNotFound):
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "webhook not found"})
			case errors.Is(err, services.ErrNotFollower):
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "not a follower"})
			case errors.Is(err, services.ErrNoPermission):
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "no permission"})
			default:
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
			}
		}

		return c.SendStatus(fiber.StatusNoContent)
	})

	// GET /channels/:channelID/followers
	app.Get("/channels/:channelID/followers", func(c *fiber.Ctx) error {
		channelIDStr := c.Params("channelID")
		channelID, err := uuid.Parse(channelIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid channel ID"})
		}

		followers, err := mock.getFollowersFunc(c.Context(), channelID)
		if err != nil {
			switch {
			case errors.Is(err, services.ErrChannelNotFound):
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "channel not found"})
			default:
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
			}
		}

		return c.Status(fiber.StatusOK).JSON(followers)
	})

	// POST /channels/:channelID/messages/:messageID/crosspost
	app.Post("/channels/:channelID/messages/:messageID/crosspost", func(c *fiber.Ctx) error {
		channelIDStr := c.Params("channelID")
		channelID, err := uuid.Parse(channelIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid channel ID"})
		}

		messageIDStr := c.Params("messageID")
		messageID, err := uuid.Parse(messageIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid message ID"})
		}

		userID, ok := c.Locals("userID").(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}

		messages, err := mock.crosspostMessageFunc(c.Context(), channelID, messageID, userID)
		if err != nil {
			switch {
			case errors.Is(err, services.ErrChannelNotFound):
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "channel not found"})
			case errors.Is(err, services.ErrMessageNotFound):
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "message not found"})
			case errors.Is(err, services.ErrNotAnnouncementChannel):
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "not an announcement channel"})
			case errors.Is(err, services.ErrNoPermission):
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "no permission"})
			default:
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
			}
		}

		return c.Status(fiber.StatusOK).JSON(messages)
	})

	return app
}

func TestFollowChannel_Success(t *testing.T) {
	sourceChannelID := uuid.New()
	targetChannelID := uuid.New()
	userID := uuid.New()
	serverID := uuid.New()
	webhookID := uuid.New()

	mock := &mockAnnouncementService{
		followChannelFunc: func(ctx context.Context, srcID, tgtID, uID uuid.UUID) (*models.Webhook, error) {
			return &models.Webhook{
				ID:              webhookID,
				Type:            2,
				ServerID:        &serverID,
				ChannelID:       targetChannelID,
				Name:            "Test Webhook",
				Token:           "test-token-123",
				SourceChannelID: &sourceChannelID,
				SourceServerID:  &serverID,
			}, nil
		},
	}

	app := setupAnnouncementTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"webhook_channel_id":"` + targetChannelID.String() + `"}`
	req := httptest.NewRequest(http.MethodPost, "/channels/"+sourceChannelID.String()+"/followers", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var result FollowChannelResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, webhookID.String(), result.ID)
	assert.Equal(t, "Test Webhook", result.Name)
	assert.Equal(t, targetChannelID.String(), result.ChannelID)
	assert.Equal(t, "test-token-123", result.Token)
	assert.Equal(t, 2, result.Type)
	assert.Equal(t, sourceChannelID.String(), result.SourceChannelID)
}

func TestFollowChannel_InvalidChannelID(t *testing.T) {
	mock := &mockAnnouncementService{}
	app := setupAnnouncementTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"webhook_channel_id":"` + uuid.New().String() + `"}`
	req := httptest.NewRequest(http.MethodPost, "/channels/invalid-id/followers", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestFollowChannel_MissingWebhookChannelID(t *testing.T) {
	mock := &mockAnnouncementService{}
	app := setupAnnouncementTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/channels/"+uuid.New().String()+"/followers", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestFollowChannel_ChannelNotFound(t *testing.T) {
	mock := &mockAnnouncementService{
		followChannelFunc: func(ctx context.Context, srcID, tgtID, uID uuid.UUID) (*models.Webhook, error) {
			return nil, services.ErrChannelNotFound
		},
	}
	app := setupAnnouncementTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	targetID := uuid.New()
	body := `{"webhook_channel_id":"` + targetID.String() + `"}`
	req := httptest.NewRequest(http.MethodPost, "/channels/"+uuid.New().String()+"/followers", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestFollowChannel_NotAnnouncementChannel(t *testing.T) {
	mock := &mockAnnouncementService{
		followChannelFunc: func(ctx context.Context, srcID, tgtID, uID uuid.UUID) (*models.Webhook, error) {
			return nil, services.ErrNotAnnouncementChannel
		},
	}
	app := setupAnnouncementTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	targetID := uuid.New()
	body := `{"webhook_channel_id":"` + targetID.String() + `"}`
	req := httptest.NewRequest(http.MethodPost, "/channels/"+uuid.New().String()+"/followers", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestUnfollowChannel_Success(t *testing.T) {
	mock := &mockAnnouncementService{
		unfollowChannelFunc: func(ctx context.Context, srcID, webhookID, uID uuid.UUID) error {
			return nil
		},
	}
	app := setupAnnouncementTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	channelID := uuid.New()
	webhookID := uuid.New()
	req := httptest.NewRequest(http.MethodDelete, "/channels/"+channelID.String()+"/followers/"+webhookID.String(), nil)
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)
}

func TestUnfollowChannel_InvalidChannelID(t *testing.T) {
	mock := &mockAnnouncementService{}
	app := setupAnnouncementTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest(http.MethodDelete, "/channels/invalid-id/followers/"+uuid.New().String(), nil)
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestUnfollowChannel_WebhookNotFound(t *testing.T) {
	mock := &mockAnnouncementService{
		unfollowChannelFunc: func(ctx context.Context, srcID, webhookID, uID uuid.UUID) error {
			return services.ErrWebhookNotFound
		},
	}
	app := setupAnnouncementTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest(http.MethodDelete, "/channels/"+uuid.New().String()+"/followers/"+uuid.New().String(), nil)
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestGetFollowers_Success(t *testing.T) {
	channelID := uuid.New()
	serverID := uuid.New()
	webhookID1 := uuid.New()
	webhookID2 := uuid.New()

	mock := &mockAnnouncementService{
		getFollowersFunc: func(ctx context.Context, chID uuid.UUID) ([]*models.Webhook, error) {
			return []*models.Webhook{
				{
					ID:        webhookID1,
					Type:      2,
					ServerID:  &serverID,
					ChannelID: channelID,
					Name:      "Follower 1",
					Token:     "token-1",
				},
				{
					ID:        webhookID2,
					Type:      2,
					ServerID:  &serverID,
					ChannelID: channelID,
					Name:      "Follower 2",
					Token:     "token-2",
				},
			}, nil
		},
	}
	app := setupAnnouncementTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest(http.MethodGet, "/channels/"+channelID.String()+"/followers", nil)
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestGetFollowers_InvalidChannelID(t *testing.T) {
	mock := &mockAnnouncementService{}
	app := setupAnnouncementTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest(http.MethodGet, "/channels/invalid-id/followers", nil)
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestCrosspostMessage_Success(t *testing.T) {
	channelID := uuid.New()
	messageID := uuid.New()

	mock := &mockAnnouncementService{
		crosspostMessageFunc: func(ctx context.Context, chID, msgID, uID uuid.UUID) ([]*models.Message, error) {
			return []*models.Message{}, nil
		},
	}
	app := setupAnnouncementTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest(http.MethodPost, "/channels/"+channelID.String()+"/messages/"+messageID.String()+"/crosspost", nil)
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestCrosspostMessage_InvalidChannelID(t *testing.T) {
	mock := &mockAnnouncementService{}
	app := setupAnnouncementTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest(http.MethodPost, "/channels/invalid-id/messages/"+uuid.New().String()+"/crosspost", nil)
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestCrosspostMessage_MessageNotFound(t *testing.T) {
	mock := &mockAnnouncementService{
		crosspostMessageFunc: func(ctx context.Context, chID, msgID, uID uuid.UUID) ([]*models.Message, error) {
			return nil, services.ErrMessageNotFound
		},
	}
	app := setupAnnouncementTestApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest(http.MethodPost, "/channels/"+uuid.New().String()+"/messages/"+uuid.New().String()+"/crosspost", nil)
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}
