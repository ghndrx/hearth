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
	sendMessageFunc    func(ctx context.Context, authorID, channelID uuid.UUID, content string, attachments []*models.Attachment, replyTo *uuid.UUID) (*models.Message, error)
	getMessagesFunc    func(ctx context.Context, channelID, requesterID uuid.UUID, before, after *uuid.UUID, limit int) ([]*models.Message, error)
	getMessageFunc     func(ctx context.Context, messageID, requesterID uuid.UUID) (*models.Message, error)
	editMessageFunc    func(ctx context.Context, messageID, authorID uuid.UUID, newContent string) (*models.Message, error)
	deleteMessageFunc  func(ctx context.Context, messageID, requesterID uuid.UUID) error
	addReactionFunc    func(ctx context.Context, messageID, userID uuid.UUID, emoji string) error
	removeReactionFunc func(ctx context.Context, messageID, userID uuid.UUID, emoji string) error
	pinMessageFunc     func(ctx context.Context, messageID, requesterID uuid.UUID) error
	unpinMessageFunc   func(ctx context.Context, messageID, requesterID uuid.UUID) error
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

func (m *mockChannelMessageService) PinMessage(ctx context.Context, messageID, requesterID uuid.UUID) error {
	if m.pinMessageFunc != nil {
		return m.pinMessageFunc(ctx, messageID, requesterID)
	}
	return nil
}

func (m *mockChannelMessageService) UnpinMessage(ctx context.Context, messageID, requesterID uuid.UUID) error {
	if m.unpinMessageFunc != nil {
		return m.unpinMessageFunc(ctx, messageID, requesterID)
	}
	return nil
}

func (m *mockChannelMessageService) GetMessage(ctx context.Context, messageID, requesterID uuid.UUID) (*models.Message, error) {
	if m.getMessageFunc != nil {
		return m.getMessageFunc(ctx, messageID, requesterID)
	}
	return nil, services.ErrMessageNotFound
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

	// UnpinMessage
	app.Delete("/channels/:id/pins/:messageId", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		messageID, err := uuid.Parse(c.Params("messageId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid message id",
			})
		}

		if err := messageService.UnpinMessage(c.Context(), messageID, userID); err != nil {
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

	// GetMessage
	app.Get("/channels/:id/messages/:messageId", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		messageID, err := uuid.Parse(c.Params("messageId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid message id",
			})
		}

		message, err := messageService.GetMessage(c.Context(), messageID, userID)
		if err != nil {
			if errors.Is(err, services.ErrMessageNotFound) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": "message not found",
				})
			}
			if errors.Is(err, services.ErrNoPermission) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "no permission to view this message",
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to get message",
			})
		}

		return c.JSON(message)
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

// GetMessage Tests

func TestChannelHandler_GetMessage_Success(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	messageID := uuid.New()

	expectedMessage := &models.Message{
		ID:        messageID,
		ChannelID: channelID,
		AuthorID:  userID,
		Content:   "Hello, world!",
		Pinned:    false,
		CreatedAt: time.Now(),
	}

	svc := &mockChannelMessageService{
		getMessageFunc: func(ctx context.Context, mID, requesterID uuid.UUID) (*models.Message, error) {
			if mID != messageID {
				t.Errorf("Expected message ID %v, got %v", messageID, mID)
			}
			if requesterID != userID {
				t.Errorf("Expected user ID %v, got %v", userID, requesterID)
			}
			return expectedMessage, nil
		},
	}

	app := setupChannelTestApp(svc)

	req := httptest.NewRequest("GET", "/channels/"+channelID.String()+"/messages/"+messageID.String(), nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	var result models.Message
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if result.ID != messageID {
		t.Errorf("Expected message ID %v, got %v", messageID, result.ID)
	}
	if result.Content != "Hello, world!" {
		t.Errorf("Expected content 'Hello, world!', got %s", result.Content)
	}
}

func TestChannelHandler_GetMessage_NotFound(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	messageID := uuid.New()

	svc := &mockChannelMessageService{
		getMessageFunc: func(ctx context.Context, mID, requesterID uuid.UUID) (*models.Message, error) {
			return nil, services.ErrMessageNotFound
		},
	}

	app := setupChannelTestApp(svc)

	req := httptest.NewRequest("GET", "/channels/"+channelID.String()+"/messages/"+messageID.String(), nil)
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

func TestChannelHandler_GetMessage_NoPermission(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	messageID := uuid.New()

	svc := &mockChannelMessageService{
		getMessageFunc: func(ctx context.Context, mID, requesterID uuid.UUID) (*models.Message, error) {
			return nil, services.ErrNoPermission
		},
	}

	app := setupChannelTestApp(svc)

	req := httptest.NewRequest("GET", "/channels/"+channelID.String()+"/messages/"+messageID.String(), nil)
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

func TestChannelHandler_GetMessage_InvalidID(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	svc := &mockChannelMessageService{}
	app := setupChannelTestApp(svc)

	req := httptest.NewRequest("GET", "/channels/"+channelID.String()+"/messages/invalid-uuid", nil)
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

// UnpinMessage Tests

func TestChannelHandler_UnpinMessage_Success(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	messageID := uuid.New()

	svc := &mockChannelMessageService{
		unpinMessageFunc: func(ctx context.Context, mID, requesterID uuid.UUID) error {
			if mID != messageID {
				t.Errorf("Expected message ID %v, got %v", messageID, mID)
			}
			if requesterID != userID {
				t.Errorf("Expected user ID %v, got %v", userID, requesterID)
			}
			return nil
		},
	}

	app := setupChannelTestApp(svc)

	req := httptest.NewRequest("DELETE", "/channels/"+channelID.String()+"/pins/"+messageID.String(), nil)
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

func TestChannelHandler_UnpinMessage_NotFound(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	messageID := uuid.New()

	svc := &mockChannelMessageService{
		unpinMessageFunc: func(ctx context.Context, mID, requesterID uuid.UUID) error {
			return services.ErrMessageNotFound
		},
	}

	app := setupChannelTestApp(svc)

	req := httptest.NewRequest("DELETE", "/channels/"+channelID.String()+"/pins/"+messageID.String(), nil)
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

func TestChannelHandler_UnpinMessage_InvalidID(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	svc := &mockChannelMessageService{}
	app := setupChannelTestApp(svc)

	req := httptest.NewRequest("DELETE", "/channels/"+channelID.String()+"/pins/invalid-uuid", nil)
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

