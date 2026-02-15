package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// mockChannelMessageService mocks MessageService for channel handler tests
type mockChannelMessageService struct {
	sendMessageFunc      func(ctx context.Context, authorID, channelID uuid.UUID, content string, attachments []*models.Attachment, replyTo *uuid.UUID) (*models.Message, error)
	getMessagesFunc      func(ctx context.Context, channelID, requesterID uuid.UUID, before, after *uuid.UUID, limit int) ([]*models.Message, error)
	editMessageFunc      func(ctx context.Context, messageID, authorID uuid.UUID, newContent string) (*models.Message, error)
	deleteMessageFunc    func(ctx context.Context, messageID, requesterID uuid.UUID) error
	addReactionFunc      func(ctx context.Context, messageID, userID uuid.UUID, emoji string) error
	removeReactionFunc   func(ctx context.Context, messageID, userID uuid.UUID, emoji string) error
	getReactionsFunc     func(ctx context.Context, messageID, requesterID uuid.UUID) ([]*models.Reaction, error)
	getReactionUsersFunc func(ctx context.Context, messageID uuid.UUID, emoji string, requesterID uuid.UUID, limit int) ([]*models.ReactionUser, error)
	pinMessageFunc       func(ctx context.Context, messageID, requesterID uuid.UUID) error
}

func (m *mockChannelMessageService) SendMessage(ctx context.Context, authorID, channelID uuid.UUID, content string, attachments []*models.Attachment, replyTo *uuid.UUID) (*models.Message, error) {
	if m.sendMessageFunc != nil {
		return m.sendMessageFunc(ctx, authorID, channelID, content, attachments, replyTo)
	}
	return nil, nil
}

func (m *mockChannelMessageService) GetMessages(ctx context.Context, channelID, requesterID uuid.UUID, before, after *uuid.UUID, limit int) ([]*models.Message, error) {
	if m.getMessagesFunc != nil {
		return m.getMessagesFunc(ctx, channelID, requesterID, before, after, limit)
	}
	return nil, nil
}

func (m *mockChannelMessageService) EditMessage(ctx context.Context, messageID, authorID uuid.UUID, newContent string) (*models.Message, error) {
	if m.editMessageFunc != nil {
		return m.editMessageFunc(ctx, messageID, authorID, newContent)
	}
	return nil, nil
}

func (m *mockChannelMessageService) DeleteMessage(ctx context.Context, messageID, requesterID uuid.UUID) error {
	if m.deleteMessageFunc != nil {
		return m.deleteMessageFunc(ctx, messageID, requesterID)
	}
	return nil
}

func (m *mockChannelMessageService) AddReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) error {
	if m.addReactionFunc != nil {
		return m.addReactionFunc(ctx, messageID, userID, emoji)
	}
	return nil
}

func (m *mockChannelMessageService) RemoveReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) error {
	if m.removeReactionFunc != nil {
		return m.removeReactionFunc(ctx, messageID, userID, emoji)
	}
	return nil
}

func (m *mockChannelMessageService) GetReactions(ctx context.Context, messageID, requesterID uuid.UUID) ([]*models.Reaction, error) {
	if m.getReactionsFunc != nil {
		return m.getReactionsFunc(ctx, messageID, requesterID)
	}
	return nil, nil
}

func (m *mockChannelMessageService) GetReactionUsers(ctx context.Context, messageID uuid.UUID, emoji string, requesterID uuid.UUID, limit int) ([]*models.ReactionUser, error) {
	if m.getReactionUsersFunc != nil {
		return m.getReactionUsersFunc(ctx, messageID, emoji, requesterID, limit)
	}
	return nil, nil
}

func (m *mockChannelMessageService) PinMessage(ctx context.Context, messageID, requesterID uuid.UUID) error {
	if m.pinMessageFunc != nil {
		return m.pinMessageFunc(ctx, messageID, requesterID)
	}
	return nil
}

// setupChannelTestApp creates a test Fiber app with channel routes
func setupChannelTestApp(messageService *mockChannelMessageService) *fiber.App {
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

	// GetMessages
	app.Get("/channels/:id/messages", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		channelID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid channel id",
			})
		}

		var before, after *uuid.UUID
		if b := c.Query("before"); b != "" {
			if id, err := uuid.Parse(b); err == nil {
				before = &id
			}
		}
		if a := c.Query("after"); a != "" {
			if id, err := uuid.Parse(a); err == nil {
				after = &id
			}
		}

		limit := c.QueryInt("limit", 50)
		if limit < 1 {
			limit = 1
		}
		if limit > 100 {
			limit = 100
		}

		messages, err := messageService.GetMessages(c.Context(), channelID, userID, before, after, limit)
		if err != nil {
			if errors.Is(err, services.ErrChannelNotFound) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": "channel not found",
				})
			}
			if errors.Is(err, services.ErrNoPermission) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "no permission",
				})
			}
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(messages)
	})

	// SendMessage
	app.Post("/channels/:id/messages", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		channelID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid channel id",
			})
		}

		var req struct {
			Content string     `json:"content"`
			ReplyTo *uuid.UUID `json:"reply_to"`
		}

		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid request body",
			})
		}

		message, err := messageService.SendMessage(c.Context(), userID, channelID, req.Content, nil, req.ReplyTo)
		if err != nil {
			if errors.Is(err, services.ErrChannelNotFound) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": "channel not found",
				})
			}
			if errors.Is(err, services.ErrNoPermission) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "no permission",
				})
			}
			if errors.Is(err, services.ErrEmptyMessage) {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "message cannot be empty",
				})
			}
			if errors.Is(err, services.ErrMessageTooLong) {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "message too long",
				})
			}
			if errors.Is(err, services.ErrRateLimited) {
				return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
					"error": "rate limited",
				})
			}
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.Status(fiber.StatusCreated).JSON(message)
	})

	// EditMessage
	app.Patch("/channels/:id/messages/:messageId", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		messageID, err := uuid.Parse(c.Params("messageId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid message id",
			})
		}

		var req struct {
			Content string `json:"content"`
		}

		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid request body",
			})
		}

		message, err := messageService.EditMessage(c.Context(), messageID, userID, req.Content)
		if err != nil {
			if errors.Is(err, services.ErrMessageNotFound) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": "message not found",
				})
			}
			if errors.Is(err, services.ErrNotMessageAuthor) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "not message author",
				})
			}
			if errors.Is(err, services.ErrEmptyMessage) {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "message cannot be empty",
				})
			}
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(message)
	})

	// DeleteMessage
	app.Delete("/channels/:id/messages/:messageId", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		messageID, err := uuid.Parse(c.Params("messageId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid message id",
			})
		}

		if err := messageService.DeleteMessage(c.Context(), messageID, userID); err != nil {
			if errors.Is(err, services.ErrMessageNotFound) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": "message not found",
				})
			}
			if errors.Is(err, services.ErrNotMessageAuthor) || errors.Is(err, services.ErrNoPermission) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "no permission",
				})
			}
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.SendStatus(fiber.StatusNoContent)
	})

	// AddReaction
	app.Put("/channels/:id/messages/:messageId/reactions/:emoji/@me", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		messageID, err := uuid.Parse(c.Params("messageId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid message id",
			})
		}
		emoji := c.Params("emoji")

		if err := messageService.AddReaction(c.Context(), messageID, userID, emoji); err != nil {
			if errors.Is(err, services.ErrMessageNotFound) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": "message not found",
				})
			}
			if errors.Is(err, services.ErrNoPermission) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "no permission",
				})
			}
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.SendStatus(fiber.StatusNoContent)
	})

	// RemoveReaction
	app.Delete("/channels/:id/messages/:messageId/reactions/:emoji/@me", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		messageID, err := uuid.Parse(c.Params("messageId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid message id",
			})
		}
		emoji := c.Params("emoji")

		if err := messageService.RemoveReaction(c.Context(), messageID, userID, emoji); err != nil {
			if errors.Is(err, services.ErrMessageNotFound) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": "message not found",
				})
			}
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.SendStatus(fiber.StatusNoContent)
	})

	// GetReactions
	app.Get("/channels/:id/messages/:messageId/reactions", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		messageID, err := uuid.Parse(c.Params("messageId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid message id",
			})
		}

		reactions, err := messageService.GetReactions(c.Context(), messageID, userID)
		if err != nil {
			if errors.Is(err, services.ErrMessageNotFound) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": "message not found",
				})
			}
			if errors.Is(err, services.ErrChannelNotFound) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": "channel not found",
				})
			}
			if errors.Is(err, services.ErrNotServerMember) || errors.Is(err, services.ErrNoPermission) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "access denied",
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to get reactions",
			})
		}

		return c.JSON(reactions)
	})

	// GetReactionUsers
	app.Get("/channels/:id/messages/:messageId/reactions/:emoji", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		messageID, err := uuid.Parse(c.Params("messageId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid message id",
			})
		}
		emoji := c.Params("emoji")
		limit := c.QueryInt("limit", 25)

		reactionUsers, err := messageService.GetReactionUsers(c.Context(), messageID, emoji, userID, limit)
		if err != nil {
			if errors.Is(err, services.ErrMessageNotFound) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": "message not found",
				})
			}
			if errors.Is(err, services.ErrChannelNotFound) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": "channel not found",
				})
			}
			if errors.Is(err, services.ErrNotServerMember) || errors.Is(err, services.ErrNoPermission) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "access denied",
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to get reaction users",
			})
		}

		return c.JSON(reactionUsers)
	})

	// PinMessage
	app.Put("/channels/:id/pins/:messageId", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		messageID, err := uuid.Parse(c.Params("messageId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid message id",
			})
		}

		if err := messageService.PinMessage(c.Context(), messageID, userID); err != nil {
			if errors.Is(err, services.ErrMessageNotFound) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": "message not found",
				})
			}
			if errors.Is(err, services.ErrNoPermission) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "no permission",
				})
			}
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.SendStatus(fiber.StatusNoContent)
	})

	// GetPins (stub)
	app.Get("/channels/:id/pins", func(c *fiber.Ctx) error {
		return c.JSON([]interface{}{})
	})

	// TriggerTyping (stub)
	app.Post("/channels/:id/typing", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})

	return app
}

// Test Helpers

func createChannelTestMessage(channelID, authorID uuid.UUID, content string) *models.Message {
	return &models.Message{
		ID:        uuid.New(),
		ChannelID: channelID,
		AuthorID:  authorID,
		Content:   content,
		CreatedAt: time.Now(),
	}
}

// GetMessages Tests

func TestChannelHandler_GetMessages_Success(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	msg1 := createChannelTestMessage(channelID, userID, "Hello world")
	msg2 := createChannelTestMessage(channelID, userID, "Second message")

	svc := &mockChannelMessageService{
		getMessagesFunc: func(ctx context.Context, cID, rID uuid.UUID, before, after *uuid.UUID, limit int) ([]*models.Message, error) {
			if cID == channelID && rID == userID {
				return []*models.Message{msg1, msg2}, nil
			}
			return nil, services.ErrChannelNotFound
		},
	}

	app := setupChannelTestApp(svc)
	req := httptest.NewRequest("GET", "/channels/"+channelID.String()+"/messages", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	var messages []*models.Message
	if err := json.NewDecoder(resp.Body).Decode(&messages); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(messages))
	}
}

func TestChannelHandler_GetMessages_WithPagination(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	beforeID := uuid.New()

	svc := &mockChannelMessageService{
		getMessagesFunc: func(ctx context.Context, cID, rID uuid.UUID, before, after *uuid.UUID, limit int) ([]*models.Message, error) {
			if before != nil && *before == beforeID && limit == 25 {
				return []*models.Message{}, nil
			}
			return nil, errors.New("unexpected params")
		},
	}

	app := setupChannelTestApp(svc)
	req := httptest.NewRequest("GET", "/channels/"+channelID.String()+"/messages?before="+beforeID.String()+"&limit=25", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestChannelHandler_GetMessages_InvalidChannelID(t *testing.T) {
	svc := &mockChannelMessageService{}
	app := setupChannelTestApp(svc)

	req := httptest.NewRequest("GET", "/channels/invalid-uuid/messages", nil)
	req.Header.Set("X-User-ID", uuid.New().String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestChannelHandler_GetMessages_ChannelNotFound(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	svc := &mockChannelMessageService{
		getMessagesFunc: func(ctx context.Context, cID, rID uuid.UUID, before, after *uuid.UUID, limit int) ([]*models.Message, error) {
			return nil, services.ErrChannelNotFound
		},
	}

	app := setupChannelTestApp(svc)
	req := httptest.NewRequest("GET", "/channels/"+channelID.String()+"/messages", nil)
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

func TestChannelHandler_GetMessages_NoPermission(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	svc := &mockChannelMessageService{
		getMessagesFunc: func(ctx context.Context, cID, rID uuid.UUID, before, after *uuid.UUID, limit int) ([]*models.Message, error) {
			return nil, services.ErrNoPermission
		},
	}

	app := setupChannelTestApp(svc)
	req := httptest.NewRequest("GET", "/channels/"+channelID.String()+"/messages", nil)
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

// SendMessage Tests

func TestChannelHandler_SendMessage_Success(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	messageContent := "Hello, world!"

	svc := &mockChannelMessageService{
		sendMessageFunc: func(ctx context.Context, authorID, cID uuid.UUID, content string, attachments []*models.Attachment, replyTo *uuid.UUID) (*models.Message, error) {
			return &models.Message{
				ID:        uuid.New(),
				ChannelID: cID,
				AuthorID:  authorID,
				Content:   content,
				CreatedAt: time.Now(),
			}, nil
		},
	}

	app := setupChannelTestApp(svc)

	body, _ := json.Marshal(map[string]string{"content": messageContent})
	req := httptest.NewRequest("POST", "/channels/"+channelID.String()+"/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 201, got %d: %s", resp.StatusCode, string(respBody))
	}

	var message models.Message
	if err := json.NewDecoder(resp.Body).Decode(&message); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if message.Content != messageContent {
		t.Errorf("Expected content %q, got %q", messageContent, message.Content)
	}
}

func TestChannelHandler_SendMessage_EmptyContent(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	svc := &mockChannelMessageService{
		sendMessageFunc: func(ctx context.Context, authorID, cID uuid.UUID, content string, attachments []*models.Attachment, replyTo *uuid.UUID) (*models.Message, error) {
			return nil, services.ErrEmptyMessage
		},
	}

	app := setupChannelTestApp(svc)

	body, _ := json.Marshal(map[string]string{"content": ""})
	req := httptest.NewRequest("POST", "/channels/"+channelID.String()+"/messages", bytes.NewReader(body))
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

func TestChannelHandler_SendMessage_TooLong(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	svc := &mockChannelMessageService{
		sendMessageFunc: func(ctx context.Context, authorID, cID uuid.UUID, content string, attachments []*models.Attachment, replyTo *uuid.UUID) (*models.Message, error) {
			return nil, services.ErrMessageTooLong
		},
	}

	app := setupChannelTestApp(svc)

	body, _ := json.Marshal(map[string]string{"content": "some very long message"})
	req := httptest.NewRequest("POST", "/channels/"+channelID.String()+"/messages", bytes.NewReader(body))
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

func TestChannelHandler_SendMessage_RateLimited(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	svc := &mockChannelMessageService{
		sendMessageFunc: func(ctx context.Context, authorID, cID uuid.UUID, content string, attachments []*models.Attachment, replyTo *uuid.UUID) (*models.Message, error) {
			return nil, services.ErrRateLimited
		},
	}

	app := setupChannelTestApp(svc)

	body, _ := json.Marshal(map[string]string{"content": "message"})
	req := httptest.NewRequest("POST", "/channels/"+channelID.String()+"/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusTooManyRequests {
		t.Errorf("Expected 429, got %d", resp.StatusCode)
	}
}

func TestChannelHandler_SendMessage_InvalidBody(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	svc := &mockChannelMessageService{}
	app := setupChannelTestApp(svc)

	req := httptest.NewRequest("POST", "/channels/"+channelID.String()+"/messages", bytes.NewReader([]byte("invalid json")))
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

func TestChannelHandler_SendMessage_WithReply(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	replyToID := uuid.New()

	var capturedReplyTo *uuid.UUID
	svc := &mockChannelMessageService{
		sendMessageFunc: func(ctx context.Context, authorID, cID uuid.UUID, content string, attachments []*models.Attachment, replyTo *uuid.UUID) (*models.Message, error) {
			capturedReplyTo = replyTo
			return &models.Message{
				ID:        uuid.New(),
				ChannelID: cID,
				AuthorID:  authorID,
				Content:   content,
				ReplyToID: replyTo,
				CreatedAt: time.Now(),
			}, nil
		},
	}

	app := setupChannelTestApp(svc)

	body, _ := json.Marshal(map[string]interface{}{
		"content":  "Reply message",
		"reply_to": replyToID.String(),
	})
	req := httptest.NewRequest("POST", "/channels/"+channelID.String()+"/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 201, got %d: %s", resp.StatusCode, string(respBody))
	}

	if capturedReplyTo == nil || *capturedReplyTo != replyToID {
		t.Errorf("Expected reply_to to be %s", replyToID.String())
	}
}

// EditMessage Tests

func TestChannelHandler_EditMessage_Success(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	messageID := uuid.New()
	newContent := "Updated content"

	svc := &mockChannelMessageService{
		editMessageFunc: func(ctx context.Context, mID, authorID uuid.UUID, content string) (*models.Message, error) {
			return &models.Message{
				ID:        mID,
				ChannelID: channelID,
				AuthorID:  authorID,
				Content:   content,
				EditedAt:  channelTimePtr(time.Now()),
			}, nil
		},
	}

	app := setupChannelTestApp(svc)

	body, _ := json.Marshal(map[string]string{"content": newContent})
	req := httptest.NewRequest("PATCH", "/channels/"+channelID.String()+"/messages/"+messageID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, string(respBody))
	}
}

func TestChannelHandler_EditMessage_NotAuthor(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	messageID := uuid.New()

	svc := &mockChannelMessageService{
		editMessageFunc: func(ctx context.Context, mID, authorID uuid.UUID, content string) (*models.Message, error) {
			return nil, services.ErrNotMessageAuthor
		},
	}

	app := setupChannelTestApp(svc)

	body, _ := json.Marshal(map[string]string{"content": "new content"})
	req := httptest.NewRequest("PATCH", "/channels/"+channelID.String()+"/messages/"+messageID.String(), bytes.NewReader(body))
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

func TestChannelHandler_EditMessage_NotFound(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	messageID := uuid.New()

	svc := &mockChannelMessageService{
		editMessageFunc: func(ctx context.Context, mID, authorID uuid.UUID, content string) (*models.Message, error) {
			return nil, services.ErrMessageNotFound
		},
	}

	app := setupChannelTestApp(svc)

	body, _ := json.Marshal(map[string]string{"content": "new content"})
	req := httptest.NewRequest("PATCH", "/channels/"+channelID.String()+"/messages/"+messageID.String(), bytes.NewReader(body))
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

// DeleteMessage Tests

func TestChannelHandler_DeleteMessage_Success(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	messageID := uuid.New()

	svc := &mockChannelMessageService{
		deleteMessageFunc: func(ctx context.Context, mID, requesterID uuid.UUID) error {
			return nil
		},
	}

	app := setupChannelTestApp(svc)

	req := httptest.NewRequest("DELETE", "/channels/"+channelID.String()+"/messages/"+messageID.String(), nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusNoContent {
		t.Errorf("Expected 204, got %d", resp.StatusCode)
	}
}

func TestChannelHandler_DeleteMessage_NotFound(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	messageID := uuid.New()

	svc := &mockChannelMessageService{
		deleteMessageFunc: func(ctx context.Context, mID, requesterID uuid.UUID) error {
			return services.ErrMessageNotFound
		},
	}

	app := setupChannelTestApp(svc)

	req := httptest.NewRequest("DELETE", "/channels/"+channelID.String()+"/messages/"+messageID.String(), nil)
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

func TestChannelHandler_DeleteMessage_NoPermission(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	messageID := uuid.New()

	svc := &mockChannelMessageService{
		deleteMessageFunc: func(ctx context.Context, mID, requesterID uuid.UUID) error {
			return services.ErrNoPermission
		},
	}

	app := setupChannelTestApp(svc)

	req := httptest.NewRequest("DELETE", "/channels/"+channelID.String()+"/messages/"+messageID.String(), nil)
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

// AddReaction Tests

func TestChannelHandler_AddReaction_Success(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	messageID := uuid.New()
	emoji := "👍"

	svc := &mockChannelMessageService{
		addReactionFunc: func(ctx context.Context, mID, uID uuid.UUID, e string) error {
			if mID == messageID && uID == userID && e == emoji {
				return nil
			}
			return errors.New("unexpected params")
		},
	}

	app := setupChannelTestApp(svc)

	req := httptest.NewRequest("PUT", "/channels/"+channelID.String()+"/messages/"+messageID.String()+"/reactions/"+emoji+"/@me", nil)
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

func TestChannelHandler_AddReaction_MessageNotFound(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	messageID := uuid.New()

	svc := &mockChannelMessageService{
		addReactionFunc: func(ctx context.Context, mID, uID uuid.UUID, e string) error {
			return services.ErrMessageNotFound
		},
	}

	app := setupChannelTestApp(svc)

	req := httptest.NewRequest("PUT", "/channels/"+channelID.String()+"/messages/"+messageID.String()+"/reactions/👍/@me", nil)
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

// RemoveReaction Tests

func TestChannelHandler_RemoveReaction_Success(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	messageID := uuid.New()
	emoji := "👍"

	svc := &mockChannelMessageService{
		removeReactionFunc: func(ctx context.Context, mID, uID uuid.UUID, e string) error {
			return nil
		},
	}

	app := setupChannelTestApp(svc)

	req := httptest.NewRequest("DELETE", "/channels/"+channelID.String()+"/messages/"+messageID.String()+"/reactions/"+emoji+"/@me", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusNoContent {
		t.Errorf("Expected 204, got %d", resp.StatusCode)
	}
}

func TestChannelHandler_RemoveReaction_MessageNotFound(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	messageID := uuid.New()

	svc := &mockChannelMessageService{
		removeReactionFunc: func(ctx context.Context, mID, uID uuid.UUID, e string) error {
			return services.ErrMessageNotFound
		},
	}

	app := setupChannelTestApp(svc)

	req := httptest.NewRequest("DELETE", "/channels/"+channelID.String()+"/messages/"+messageID.String()+"/reactions/👍/@me", nil)
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

// PinMessage Tests

func TestChannelHandler_PinMessage_Success(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	messageID := uuid.New()

	svc := &mockChannelMessageService{
		pinMessageFunc: func(ctx context.Context, mID, requesterID uuid.UUID) error {
			return nil
		},
	}

	app := setupChannelTestApp(svc)

	req := httptest.NewRequest("PUT", "/channels/"+channelID.String()+"/pins/"+messageID.String(), nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusNoContent {
		t.Errorf("Expected 204, got %d", resp.StatusCode)
	}
}

func TestChannelHandler_PinMessage_NoPermission(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	messageID := uuid.New()

	svc := &mockChannelMessageService{
		pinMessageFunc: func(ctx context.Context, mID, requesterID uuid.UUID) error {
			return services.ErrNoPermission
		},
	}

	app := setupChannelTestApp(svc)

	req := httptest.NewRequest("PUT", "/channels/"+channelID.String()+"/pins/"+messageID.String(), nil)
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

// GetPins Tests

func TestChannelHandler_GetPins_Success(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	svc := &mockChannelMessageService{}
	app := setupChannelTestApp(svc)

	req := httptest.NewRequest("GET", "/channels/"+channelID.String()+"/pins", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}

// TriggerTyping Tests

func TestChannelHandler_TriggerTyping_Success(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	svc := &mockChannelMessageService{}
	app := setupChannelTestApp(svc)

	req := httptest.NewRequest("POST", "/channels/"+channelID.String()+"/typing", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusNoContent {
		t.Errorf("Expected 204, got %d", resp.StatusCode)
	}
}

// Helper function
func channelTimePtr(t time.Time) *time.Time {
	return &t
}

// mockChannelCRUDService mocks ChannelService for channel CRUD tests
type mockChannelCRUDService struct {
	getChannelFunc    func(ctx context.Context, id uuid.UUID) (*models.Channel, error)
	updateChannelFunc func(ctx context.Context, id, requesterID uuid.UUID, updates *models.ChannelUpdate) (*models.Channel, error)
	deleteChannelFunc func(ctx context.Context, id, requesterID uuid.UUID) error
}

func (m *mockChannelCRUDService) GetChannel(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
	if m.getChannelFunc != nil {
		return m.getChannelFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockChannelCRUDService) UpdateChannel(ctx context.Context, id, requesterID uuid.UUID, updates *models.ChannelUpdate) (*models.Channel, error) {
	if m.updateChannelFunc != nil {
		return m.updateChannelFunc(ctx, id, requesterID, updates)
	}
	return nil, nil
}

func (m *mockChannelCRUDService) DeleteChannel(ctx context.Context, id, requesterID uuid.UUID) error {
	if m.deleteChannelFunc != nil {
		return m.deleteChannelFunc(ctx, id, requesterID)
	}
	return nil
}

// setupChannelCRUDTestApp creates a test Fiber app with channel CRUD routes
func setupChannelCRUDTestApp(channelSvc *mockChannelCRUDService) *fiber.App {
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

	// Get Channel
	app.Get("/channels/:id", func(c *fiber.Ctx) error {
		channelID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid channel id",
			})
		}

		channel, err := channelSvc.GetChannel(c.Context(), channelID)
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

	// Update Channel
	app.Patch("/channels/:id", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		channelID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid channel id",
			})
		}

		var req struct {
			Name        *string `json:"name"`
			Topic       *string `json:"topic"`
			Position    *int    `json:"position"`
			Slowmode    *int    `json:"slowmode"`
			NSFW        *bool   `json:"nsfw"`
			E2EEEnabled *bool   `json:"e2ee_enabled"`
		}

		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid request body",
			})
		}

		updates := &models.ChannelUpdate{
			Name:        req.Name,
			Topic:       req.Topic,
			Position:    req.Position,
			Slowmode:    req.Slowmode,
			NSFW:        req.NSFW,
			E2EEEnabled: req.E2EEEnabled,
		}

		channel, err := channelSvc.UpdateChannel(c.Context(), channelID, userID, updates)
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

	// Delete Channel
	app.Delete("/channels/:id", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		channelID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid channel id",
			})
		}

		if err := channelSvc.DeleteChannel(c.Context(), channelID, userID); err != nil {
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
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "cannot delete DM channels",
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

// Get Channel Tests

func TestChannelHandler_Get_Success(t *testing.T) {
	channelID := uuid.New()
	serverID := uuid.New()

	expectedChannel := &models.Channel{
		ID:        channelID,
		ServerID:  &serverID,
		Name:      "general",
		Type:      models.ChannelTypeText,
		Position:  0,
		CreatedAt: time.Now(),
	}

	svc := &mockChannelCRUDService{
		getChannelFunc: func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
			if id == channelID {
				return expectedChannel, nil
			}
			return nil, services.ErrChannelNotFound
		},
	}

	app := setupChannelCRUDTestApp(svc)

	req := httptest.NewRequest("GET", "/channels/"+channelID.String(), nil)
	req.Header.Set("X-User-ID", uuid.New().String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
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

func TestChannelHandler_Get_NotFound(t *testing.T) {
	channelID := uuid.New()

	svc := &mockChannelCRUDService{
		getChannelFunc: func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
			return nil, services.ErrChannelNotFound
		},
	}

	app := setupChannelCRUDTestApp(svc)

	req := httptest.NewRequest("GET", "/channels/"+channelID.String(), nil)
	req.Header.Set("X-User-ID", uuid.New().String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestChannelHandler_Get_InvalidID(t *testing.T) {
	svc := &mockChannelCRUDService{}
	app := setupChannelCRUDTestApp(svc)

	req := httptest.NewRequest("GET", "/channels/invalid-uuid", nil)
	req.Header.Set("X-User-ID", uuid.New().String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

// Update Channel Tests

func TestChannelHandler_Update_Success(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()
	newName := "updated-channel"
	newTopic := "Updated topic"

	svc := &mockChannelCRUDService{
		updateChannelFunc: func(ctx context.Context, id, requesterID uuid.UUID, updates *models.ChannelUpdate) (*models.Channel, error) {
			return &models.Channel{
				ID:        channelID,
				ServerID:  &serverID,
				Name:      *updates.Name,
				Topic:     *updates.Topic,
				Type:      models.ChannelTypeText,
				CreatedAt: time.Now(),
			}, nil
		},
	}

	app := setupChannelCRUDTestApp(svc)

	body := map[string]interface{}{
		"name":  newName,
		"topic": newTopic,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("PATCH", "/channels/"+channelID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())

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

	if channel.Name != newName {
		t.Errorf("Expected name '%s', got '%s'", newName, channel.Name)
	}
	if channel.Topic != newTopic {
		t.Errorf("Expected topic '%s', got '%s'", newTopic, channel.Topic)
	}
}

func TestChannelHandler_Update_NotFound(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	svc := &mockChannelCRUDService{
		updateChannelFunc: func(ctx context.Context, id, requesterID uuid.UUID, updates *models.ChannelUpdate) (*models.Channel, error) {
			return nil, services.ErrChannelNotFound
		},
	}

	app := setupChannelCRUDTestApp(svc)

	body := map[string]interface{}{
		"name": "new-name",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("PATCH", "/channels/"+channelID.String(), bytes.NewReader(bodyBytes))
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

func TestChannelHandler_Update_NotServerMember(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	svc := &mockChannelCRUDService{
		updateChannelFunc: func(ctx context.Context, id, requesterID uuid.UUID, updates *models.ChannelUpdate) (*models.Channel, error) {
			return nil, services.ErrNotServerMember
		},
	}

	app := setupChannelCRUDTestApp(svc)

	body := map[string]interface{}{
		"name": "new-name",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("PATCH", "/channels/"+channelID.String(), bytes.NewReader(bodyBytes))
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

func TestChannelHandler_Update_InvalidBody(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	svc := &mockChannelCRUDService{}
	app := setupChannelCRUDTestApp(svc)

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

// Delete Channel Tests

func TestChannelHandler_Delete_Success(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	svc := &mockChannelCRUDService{
		deleteChannelFunc: func(ctx context.Context, id, requesterID uuid.UUID) error {
			return nil
		},
	}

	app := setupChannelCRUDTestApp(svc)

	req := httptest.NewRequest("DELETE", "/channels/"+channelID.String(), nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusNoContent {
		t.Errorf("Expected 204, got %d", resp.StatusCode)
	}
}

func TestChannelHandler_Delete_NotFound(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	svc := &mockChannelCRUDService{
		deleteChannelFunc: func(ctx context.Context, id, requesterID uuid.UUID) error {
			return services.ErrChannelNotFound
		},
	}

	app := setupChannelCRUDTestApp(svc)

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

func TestChannelHandler_Delete_NotServerMember(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	svc := &mockChannelCRUDService{
		deleteChannelFunc: func(ctx context.Context, id, requesterID uuid.UUID) error {
			return services.ErrNotServerMember
		},
	}

	app := setupChannelCRUDTestApp(svc)

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

func TestChannelHandler_Delete_CannotDeleteDM(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	svc := &mockChannelCRUDService{
		deleteChannelFunc: func(ctx context.Context, id, requesterID uuid.UUID) error {
			return services.ErrCannotDeleteDM
		},
	}

	app := setupChannelCRUDTestApp(svc)

	req := httptest.NewRequest("DELETE", "/channels/"+channelID.String(), nil)
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

func TestChannelHandler_Delete_InvalidID(t *testing.T) {
	userID := uuid.New()

	svc := &mockChannelCRUDService{}
	app := setupChannelCRUDTestApp(svc)

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

// ============================================================
// Typing Indicator Tests
// ============================================================

// mockTypingService mocks the TypingService for tests
type mockTypingService struct {
	startTypingFunc     func(ctx context.Context, channelID, userID uuid.UUID) error
	stopTypingFunc      func(ctx context.Context, channelID, userID uuid.UUID) error
	getTypingUsersFunc  func(ctx context.Context, channelID uuid.UUID) ([]models.TypingIndicator, error)
	isTypingFunc        func(ctx context.Context, channelID, userID uuid.UUID) (bool, error)
}

func (m *mockTypingService) StartTyping(ctx context.Context, channelID, userID uuid.UUID) error {
	if m.startTypingFunc != nil {
		return m.startTypingFunc(ctx, channelID, userID)
	}
	return nil
}

func (m *mockTypingService) StopTyping(ctx context.Context, channelID, userID uuid.UUID) error {
	if m.stopTypingFunc != nil {
		return m.stopTypingFunc(ctx, channelID, userID)
	}
	return nil
}

func (m *mockTypingService) GetTypingUsers(ctx context.Context, channelID uuid.UUID) ([]models.TypingIndicator, error) {
	if m.getTypingUsersFunc != nil {
		return m.getTypingUsersFunc(ctx, channelID)
	}
	return []models.TypingIndicator{}, nil
}

func (m *mockTypingService) IsTyping(ctx context.Context, channelID, userID uuid.UUID) (bool, error) {
	if m.isTypingFunc != nil {
		return m.isTypingFunc(ctx, channelID, userID)
	}
	return false, nil
}

// setupTypingTestApp creates a test Fiber app with typing routes
func setupTypingTestApp(channelSvc *mockChannelCRUDService, typingSvc *mockTypingService) *fiber.App {
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

	// TriggerTyping (POST /channels/:id/typing)
	app.Post("/channels/:id/typing", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		channelID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid channel id",
			})
		}

		// Verify channel exists
		_, err = channelSvc.GetChannel(c.Context(), channelID)
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

		// Start typing
		if typingSvc != nil {
			if err := typingSvc.StartTyping(c.Context(), channelID, userID); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "failed to trigger typing",
				})
			}
		}

		return c.SendStatus(fiber.StatusNoContent)
	})

	// GetTypingUsers (GET /channels/:id/typing)
	app.Get("/channels/:id/typing", func(c *fiber.Ctx) error {
		channelID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid channel id",
			})
		}

		// Verify channel exists
		_, err = channelSvc.GetChannel(c.Context(), channelID)
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

		// Get typing users
		if typingSvc == nil {
			return c.JSON([]interface{}{})
		}

		indicators, err := typingSvc.GetTypingUsers(c.Context(), channelID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to get typing users",
			})
		}

		return c.JSON(indicators)
	})

	return app
}

func TestTypingHandler_TriggerTyping_Success(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	channelSvc := &mockChannelCRUDService{
		getChannelFunc: func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
			return &models.Channel{
				ID:       channelID,
				ServerID: &serverID,
				Name:     "general",
				Type:     models.ChannelTypeText,
			}, nil
		},
	}

	startCalled := false
	typingSvc := &mockTypingService{
		startTypingFunc: func(ctx context.Context, cID, uID uuid.UUID) error {
			startCalled = true
			if cID != channelID {
				t.Errorf("Expected channel ID %s, got %s", channelID, cID)
			}
			if uID != userID {
				t.Errorf("Expected user ID %s, got %s", userID, uID)
			}
			return nil
		},
	}

	app := setupTypingTestApp(channelSvc, typingSvc)

	req := httptest.NewRequest("POST", "/channels/"+channelID.String()+"/typing", nil)
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

	if !startCalled {
		t.Error("Expected StartTyping to be called")
	}
}

func TestTypingHandler_TriggerTyping_ChannelNotFound(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	channelSvc := &mockChannelCRUDService{
		getChannelFunc: func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
			return nil, services.ErrChannelNotFound
		},
	}

	typingSvc := &mockTypingService{}

	app := setupTypingTestApp(channelSvc, typingSvc)

	req := httptest.NewRequest("POST", "/channels/"+channelID.String()+"/typing", nil)
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

func TestTypingHandler_TriggerTyping_InvalidChannelID(t *testing.T) {
	userID := uuid.New()

	channelSvc := &mockChannelCRUDService{}
	typingSvc := &mockTypingService{}

	app := setupTypingTestApp(channelSvc, typingSvc)

	req := httptest.NewRequest("POST", "/channels/invalid-uuid/typing", nil)
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

func TestTypingHandler_TriggerTyping_ServiceError(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	channelSvc := &mockChannelCRUDService{
		getChannelFunc: func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
			return &models.Channel{
				ID:       channelID,
				ServerID: &serverID,
				Name:     "general",
				Type:     models.ChannelTypeText,
			}, nil
		},
	}

	typingSvc := &mockTypingService{
		startTypingFunc: func(ctx context.Context, cID, uID uuid.UUID) error {
			return errors.New("service error")
		},
	}

	app := setupTypingTestApp(channelSvc, typingSvc)

	req := httptest.NewRequest("POST", "/channels/"+channelID.String()+"/typing", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Errorf("Expected 500, got %d", resp.StatusCode)
	}
}

func TestTypingHandler_GetTypingUsers_Success(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()
	typingUser1 := uuid.New()
	typingUser2 := uuid.New()

	channelSvc := &mockChannelCRUDService{
		getChannelFunc: func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
			return &models.Channel{
				ID:       channelID,
				ServerID: &serverID,
				Name:     "general",
				Type:     models.ChannelTypeText,
			}, nil
		},
	}

	now := time.Now()
	typingSvc := &mockTypingService{
		getTypingUsersFunc: func(ctx context.Context, cID uuid.UUID) ([]models.TypingIndicator, error) {
			return []models.TypingIndicator{
				{ChannelID: channelID, UserID: typingUser1, Timestamp: now},
				{ChannelID: channelID, UserID: typingUser2, Timestamp: now},
			}, nil
		},
	}

	app := setupTypingTestApp(channelSvc, typingSvc)

	req := httptest.NewRequest("GET", "/channels/"+channelID.String()+"/typing", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	var indicators []models.TypingIndicator
	if err := json.NewDecoder(resp.Body).Decode(&indicators); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(indicators) != 2 {
		t.Errorf("Expected 2 typing indicators, got %d", len(indicators))
	}
}

func TestTypingHandler_GetTypingUsers_EmptyChannel(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	channelSvc := &mockChannelCRUDService{
		getChannelFunc: func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
			return &models.Channel{
				ID:       channelID,
				ServerID: &serverID,
				Name:     "general",
				Type:     models.ChannelTypeText,
			}, nil
		},
	}

	typingSvc := &mockTypingService{
		getTypingUsersFunc: func(ctx context.Context, cID uuid.UUID) ([]models.TypingIndicator, error) {
			return []models.TypingIndicator{}, nil
		},
	}

	app := setupTypingTestApp(channelSvc, typingSvc)

	req := httptest.NewRequest("GET", "/channels/"+channelID.String()+"/typing", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	var indicators []models.TypingIndicator
	if err := json.NewDecoder(resp.Body).Decode(&indicators); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(indicators) != 0 {
		t.Errorf("Expected 0 typing indicators, got %d", len(indicators))
	}
}

func TestTypingHandler_GetTypingUsers_ChannelNotFound(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	channelSvc := &mockChannelCRUDService{
		getChannelFunc: func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
			return nil, services.ErrChannelNotFound
		},
	}

	typingSvc := &mockTypingService{}

	app := setupTypingTestApp(channelSvc, typingSvc)

	req := httptest.NewRequest("GET", "/channels/"+channelID.String()+"/typing", nil)
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

func TestTypingHandler_GetTypingUsers_InvalidChannelID(t *testing.T) {
	userID := uuid.New()

	channelSvc := &mockChannelCRUDService{}
	typingSvc := &mockTypingService{}

	app := setupTypingTestApp(channelSvc, typingSvc)

	req := httptest.NewRequest("GET", "/channels/invalid-uuid/typing", nil)
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

func TestTypingHandler_GetTypingUsers_NilTypingService(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	channelSvc := &mockChannelCRUDService{
		getChannelFunc: func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
			return &models.Channel{
				ID:       channelID,
				ServerID: &serverID,
				Name:     "general",
				Type:     models.ChannelTypeText,
			}, nil
		},
	}

	// nil typing service
	app := setupTypingTestApp(channelSvc, nil)

	req := httptest.NewRequest("GET", "/channels/"+channelID.String()+"/typing", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	// Should return empty array
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "[]" {
		t.Errorf("Expected empty array, got %s", string(body))
	}
}

func TestTypingHandler_GetTypingUsers_ServiceError(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	channelSvc := &mockChannelCRUDService{
		getChannelFunc: func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
			return &models.Channel{
				ID:       channelID,
				ServerID: &serverID,
				Name:     "general",
				Type:     models.ChannelTypeText,
			}, nil
		},
	}

	typingSvc := &mockTypingService{
		getTypingUsersFunc: func(ctx context.Context, cID uuid.UUID) ([]models.TypingIndicator, error) {
			return nil, errors.New("service error")
		},
	}

	app := setupTypingTestApp(channelSvc, typingSvc)

	req := httptest.NewRequest("GET", "/channels/"+channelID.String()+"/typing", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Errorf("Expected 500, got %d", resp.StatusCode)
	}
}
