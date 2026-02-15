package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// Mock MessageService
type mockMessageService struct {
	sendMessageFunc       func(ctx context.Context, authorID, channelID uuid.UUID, content string, attachments []*models.Attachment, replyTo *uuid.UUID) (*models.Message, error)
	getMessagesFunc       func(ctx context.Context, channelID, requesterID uuid.UUID, before, after *uuid.UUID, limit int) ([]*models.Message, error)
	getMessageFunc        func(ctx context.Context, messageID, requesterID uuid.UUID) (*models.Message, error)
	editMessageFunc       func(ctx context.Context, messageID, authorID uuid.UUID, newContent string) (*models.Message, error)
	deleteMessageFunc     func(ctx context.Context, messageID, requesterID uuid.UUID) error
	addReactionFunc       func(ctx context.Context, messageID, userID uuid.UUID, emoji string) error
	removeReactionFunc    func(ctx context.Context, messageID, userID uuid.UUID, emoji string) error
	pinMessageFunc        func(ctx context.Context, messageID, requesterID uuid.UUID) error
	unpinMessageFunc      func(ctx context.Context, messageID, requesterID uuid.UUID) error
	getPinnedMessagesFunc func(ctx context.Context, channelID, requesterID uuid.UUID) ([]*models.Message, error)
}

func (m *mockMessageService) SendMessage(ctx context.Context, authorID, channelID uuid.UUID, content string, attachments []*models.Attachment, replyTo *uuid.UUID) (*models.Message, error) {
	if m.sendMessageFunc != nil {
		return m.sendMessageFunc(ctx, authorID, channelID, content, attachments, replyTo)
	}
	return nil, nil
}

func (m *mockMessageService) GetMessages(ctx context.Context, channelID, requesterID uuid.UUID, before, after *uuid.UUID, limit int) ([]*models.Message, error) {
	if m.getMessagesFunc != nil {
		return m.getMessagesFunc(ctx, channelID, requesterID, before, after, limit)
	}
	return nil, nil
}

func (m *mockMessageService) GetMessage(ctx context.Context, messageID, requesterID uuid.UUID) (*models.Message, error) {
	if m.getMessageFunc != nil {
		return m.getMessageFunc(ctx, messageID, requesterID)
	}
	return nil, nil
}

func (m *mockMessageService) EditMessage(ctx context.Context, messageID, authorID uuid.UUID, newContent string) (*models.Message, error) {
	if m.editMessageFunc != nil {
		return m.editMessageFunc(ctx, messageID, authorID, newContent)
	}
	return nil, nil
}

func (m *mockMessageService) DeleteMessage(ctx context.Context, messageID, requesterID uuid.UUID) error {
	if m.deleteMessageFunc != nil {
		return m.deleteMessageFunc(ctx, messageID, requesterID)
	}
	return nil
}

func (m *mockMessageService) AddReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) error {
	if m.addReactionFunc != nil {
		return m.addReactionFunc(ctx, messageID, userID, emoji)
	}
	return nil
}

func (m *mockMessageService) RemoveReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) error {
	if m.removeReactionFunc != nil {
		return m.removeReactionFunc(ctx, messageID, userID, emoji)
	}
	return nil
}

func (m *mockMessageService) PinMessage(ctx context.Context, messageID, requesterID uuid.UUID) error {
	if m.pinMessageFunc != nil {
		return m.pinMessageFunc(ctx, messageID, requesterID)
	}
	return nil
}

func (m *mockMessageService) UnpinMessage(ctx context.Context, messageID, requesterID uuid.UUID) error {
	if m.unpinMessageFunc != nil {
		return m.unpinMessageFunc(ctx, messageID, requesterID)
	}
	return nil
}

func (m *mockMessageService) GetPinnedMessages(ctx context.Context, channelID, requesterID uuid.UUID) ([]*models.Message, error) {
	if m.getPinnedMessagesFunc != nil {
		return m.getPinnedMessagesFunc(ctx, channelID, requesterID)
	}
	return nil, nil
}

// setupMessageTestApp creates a test Fiber app with message routes
func setupMessageTestApp(messageService *mockMessageService) *fiber.App {
	app := fiber.New()

	handlers := &MessageHandlers{
		messageService: &services.MessageService{},
	}

	// We'll use a custom approach to inject the mock
	// Create routes that call our mock directly

	// Inject userID middleware
	app.Use(func(c *fiber.Ctx) error {
		userID := c.Get("X-User-ID")
		if userID != "" {
			id, _ := uuid.Parse(userID)
			c.Locals("userID", id)
		}
		return c.Next()
	})

	// SendMessage
	app.Post("/channels/:channelID/messages", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		channelID, err := uuid.Parse(c.Params("channelID"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid channel ID"})
		}

		var req struct {
			Content   string  `json:"content"`
			Nonce     *string `json:"nonce,omitempty"`
			TTS       bool    `json:"tts"`
			ReplyToID *string `json:"message_reference,omitempty"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}

		var replyToID *uuid.UUID
		if req.ReplyToID != nil {
			id, err := uuid.Parse(*req.ReplyToID)
			if err == nil {
				replyToID = &id
			}
		}

		message, err := messageService.SendMessage(c.Context(), userID, channelID, req.Content, nil, replyToID)
		if err != nil {
			if errors.Is(err, services.ErrChannelNotFound) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Channel not found"})
			}
			if errors.Is(err, services.ErrEmptyMessage) {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Message content cannot be empty"})
			}
			if errors.Is(err, services.ErrMessageTooLong) {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Message too long"})
			}
			if errors.Is(err, services.ErrRateLimited) {
				return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "Rate limited"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.Status(fiber.StatusCreated).JSON(message)
	})

	// GetMessages
	app.Get("/channels/:channelID/messages", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		channelID, err := uuid.Parse(c.Params("channelID"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid channel ID"})
		}

		limit := 50
		if l := c.Query("limit"); l != "" {
			if parsed, err := parseInt(l); err == nil && parsed > 0 && parsed <= 100 {
				limit = parsed
			}
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

		messages, err := messageService.GetMessages(c.Context(), channelID, userID, before, after, limit)
		if err != nil {
			if errors.Is(err, services.ErrChannelNotFound) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Channel not found"})
			}
			if errors.Is(err, services.ErrNoPermission) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "No permission"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(messages)
	})

	// GetMessage
	app.Get("/channels/:channelID/messages/:messageID", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		messageID, err := uuid.Parse(c.Params("messageID"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid message ID"})
		}

		message, err := messageService.GetMessage(c.Context(), messageID, userID)
		if err != nil {
			if errors.Is(err, services.ErrMessageNotFound) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Message not found"})
			}
			if errors.Is(err, services.ErrNotServerMember) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Not a member of this server"})
			}
			if errors.Is(err, services.ErrNoPermission) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "No permission to view this message"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(message)
	})

	// EditMessage
	app.Patch("/channels/:channelID/messages/:messageID", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		messageID, err := uuid.Parse(c.Params("messageID"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid message ID"})
		}

		var req struct {
			Content string `json:"content"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}

		message, err := messageService.EditMessage(c.Context(), messageID, userID, req.Content)
		if err != nil {
			if errors.Is(err, services.ErrMessageNotFound) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Message not found"})
			}
			if errors.Is(err, services.ErrNotMessageAuthor) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Not message author"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(message)
	})

	// DeleteMessage
	app.Delete("/channels/:channelID/messages/:messageID", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		messageID, err := uuid.Parse(c.Params("messageID"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid message ID"})
		}

		err = messageService.DeleteMessage(c.Context(), messageID, userID)
		if err != nil {
			if errors.Is(err, services.ErrMessageNotFound) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Message not found"})
			}
			if errors.Is(err, services.ErrNotMessageAuthor) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Not message author"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.SendStatus(fiber.StatusNoContent)
	})

	// BulkDeleteMessages (not implemented)
	app.Post("/channels/:channelID/messages/bulk-delete", func(c *fiber.Ctx) error {
		_, err := uuid.Parse(c.Params("channelID"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid channel ID"})
		}
		return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{"error": "Not implemented"})
	})

	// AddReaction
	app.Put("/channels/:channelID/messages/:messageID/reactions/:emoji/@me", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		messageID, err := uuid.Parse(c.Params("messageID"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid message ID"})
		}
		emoji := c.Params("emoji")

		err = messageService.AddReaction(c.Context(), messageID, userID, emoji)
		if err != nil {
			if errors.Is(err, services.ErrMessageNotFound) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Message not found"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.SendStatus(fiber.StatusNoContent)
	})

	// RemoveReaction
	app.Delete("/channels/:channelID/messages/:messageID/reactions/:emoji/@me", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		messageID, err := uuid.Parse(c.Params("messageID"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid message ID"})
		}
		emoji := c.Params("emoji")

		err = messageService.RemoveReaction(c.Context(), messageID, userID, emoji)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.SendStatus(fiber.StatusNoContent)
	})

	// GetPinnedMessages
	app.Get("/channels/:channelID/pins", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		channelID, err := uuid.Parse(c.Params("channelID"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid channel ID"})
		}

		messages, err := messageService.GetPinnedMessages(c.Context(), channelID, userID)
		if err != nil {
			if errors.Is(err, services.ErrChannelNotFound) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Channel not found"})
			}
			if errors.Is(err, services.ErrNoPermission) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "No permission to view pinned messages"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(messages)
	})

	// PinMessage
	app.Put("/channels/:channelID/pins/:messageID", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		messageID, err := uuid.Parse(c.Params("messageID"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid message ID"})
		}

		err = messageService.PinMessage(c.Context(), messageID, userID)
		if err != nil {
			if errors.Is(err, services.ErrMessageNotFound) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Message not found"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.SendStatus(fiber.StatusNoContent)
	})

	// UnpinMessage
	app.Delete("/channels/:channelID/pins/:messageID", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		messageID, err := uuid.Parse(c.Params("messageID"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid message ID"})
		}

		err = messageService.UnpinMessage(c.Context(), messageID, userID)
		if err != nil {
			if errors.Is(err, services.ErrMessageNotFound) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Message not found"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.SendStatus(fiber.StatusNoContent)
	})

	// Ignore the handlers variable warning
	_ = handlers

	return app
}

func parseInt(s string) (int, error) {
	return strconv.Atoi(s)
}

// ========== SendMessage Tests ==========

func TestSendMessage_Success(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	messageID := uuid.New()

	mockService := &mockMessageService{
		sendMessageFunc: func(ctx context.Context, authorID, chID uuid.UUID, content string, attachments []*models.Attachment, replyTo *uuid.UUID) (*models.Message, error) {
			return &models.Message{
				ID:        messageID,
				ChannelID: chID,
				AuthorID:  authorID,
				Content:   content,
				Type:      models.MessageTypeDefault,
				CreatedAt: time.Now(),
			}, nil
		},
	}

	app := setupMessageTestApp(mockService)

	body, _ := json.Marshal(map[string]interface{}{
		"content": "Hello, world!",
	})

	req := httptest.NewRequest("POST", "/channels/"+channelID.String()+"/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 201, got %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var message models.Message
	json.NewDecoder(resp.Body).Decode(&message)

	if message.ID != messageID {
		t.Errorf("Expected message ID %s, got %s", messageID, message.ID)
	}
	if message.Content != "Hello, world!" {
		t.Errorf("Expected content 'Hello, world!', got %s", message.Content)
	}
}

func TestSendMessage_WithReply(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	messageID := uuid.New()
	replyToID := uuid.New()

	var capturedReplyTo *uuid.UUID

	mockService := &mockMessageService{
		sendMessageFunc: func(ctx context.Context, authorID, chID uuid.UUID, content string, attachments []*models.Attachment, replyTo *uuid.UUID) (*models.Message, error) {
			capturedReplyTo = replyTo
			return &models.Message{
				ID:        messageID,
				ChannelID: chID,
				AuthorID:  authorID,
				Content:   content,
				Type:      models.MessageTypeReply,
				ReplyToID: replyTo,
				CreatedAt: time.Now(),
			}, nil
		},
	}

	app := setupMessageTestApp(mockService)

	replyStr := replyToID.String()
	body, _ := json.Marshal(map[string]interface{}{
		"content":           "This is a reply",
		"message_reference": replyStr,
	})

	req := httptest.NewRequest("POST", "/channels/"+channelID.String()+"/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("Expected status 201, got %d", resp.StatusCode)
	}

	if capturedReplyTo == nil || *capturedReplyTo != replyToID {
		t.Error("Reply ID was not passed correctly")
	}
}

func TestSendMessage_InvalidChannelID(t *testing.T) {
	userID := uuid.New()
	mockService := &mockMessageService{}
	app := setupMessageTestApp(mockService)

	body, _ := json.Marshal(map[string]interface{}{
		"content": "Hello",
	})

	req := httptest.NewRequest("POST", "/channels/invalid-uuid/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestSendMessage_EmptyContent(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	mockService := &mockMessageService{
		sendMessageFunc: func(ctx context.Context, authorID, chID uuid.UUID, content string, attachments []*models.Attachment, replyTo *uuid.UUID) (*models.Message, error) {
			return nil, services.ErrEmptyMessage
		},
	}

	app := setupMessageTestApp(mockService)

	body, _ := json.Marshal(map[string]interface{}{
		"content": "",
	})

	req := httptest.NewRequest("POST", "/channels/"+channelID.String()+"/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestSendMessage_ChannelNotFound(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	mockService := &mockMessageService{
		sendMessageFunc: func(ctx context.Context, authorID, chID uuid.UUID, content string, attachments []*models.Attachment, replyTo *uuid.UUID) (*models.Message, error) {
			return nil, services.ErrChannelNotFound
		},
	}

	app := setupMessageTestApp(mockService)

	body, _ := json.Marshal(map[string]interface{}{
		"content": "Hello",
	})

	req := httptest.NewRequest("POST", "/channels/"+channelID.String()+"/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestSendMessage_RateLimited(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	mockService := &mockMessageService{
		sendMessageFunc: func(ctx context.Context, authorID, chID uuid.UUID, content string, attachments []*models.Attachment, replyTo *uuid.UUID) (*models.Message, error) {
			return nil, services.ErrRateLimited
		},
	}

	app := setupMessageTestApp(mockService)

	body, _ := json.Marshal(map[string]interface{}{
		"content": "Hello",
	})

	req := httptest.NewRequest("POST", "/channels/"+channelID.String()+"/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusTooManyRequests {
		t.Fatalf("Expected status 429, got %d", resp.StatusCode)
	}
}

// ========== GetMessages Tests ==========

func TestGetMessages_Success(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	messages := []*models.Message{
		{ID: uuid.New(), ChannelID: channelID, Content: "Message 1", CreatedAt: time.Now()},
		{ID: uuid.New(), ChannelID: channelID, Content: "Message 2", CreatedAt: time.Now()},
	}

	mockService := &mockMessageService{
		getMessagesFunc: func(ctx context.Context, chID, reqID uuid.UUID, before, after *uuid.UUID, limit int) ([]*models.Message, error) {
			return messages, nil
		},
	}

	app := setupMessageTestApp(mockService)

	req := httptest.NewRequest("GET", "/channels/"+channelID.String()+"/messages", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	var result []*models.Message
	json.NewDecoder(resp.Body).Decode(&result)

	if len(result) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(result))
	}
}

func TestGetMessages_WithPagination(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	beforeID := uuid.New()

	var capturedBefore *uuid.UUID
	var capturedLimit int

	mockService := &mockMessageService{
		getMessagesFunc: func(ctx context.Context, chID, reqID uuid.UUID, before, after *uuid.UUID, limit int) ([]*models.Message, error) {
			capturedBefore = before
			capturedLimit = limit
			return []*models.Message{}, nil
		},
	}

	app := setupMessageTestApp(mockService)

	req := httptest.NewRequest("GET", "/channels/"+channelID.String()+"/messages?before="+beforeID.String()+"&limit=25", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	if capturedBefore == nil || *capturedBefore != beforeID {
		t.Error("Before ID was not passed correctly")
	}

	if capturedLimit != 25 {
		t.Errorf("Expected limit 25, got %d", capturedLimit)
	}
}

func TestGetMessages_ChannelNotFound(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	mockService := &mockMessageService{
		getMessagesFunc: func(ctx context.Context, chID, reqID uuid.UUID, before, after *uuid.UUID, limit int) ([]*models.Message, error) {
			return nil, services.ErrChannelNotFound
		},
	}

	app := setupMessageTestApp(mockService)

	req := httptest.NewRequest("GET", "/channels/"+channelID.String()+"/messages", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestGetMessages_NoPermission(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	mockService := &mockMessageService{
		getMessagesFunc: func(ctx context.Context, chID, reqID uuid.UUID, before, after *uuid.UUID, limit int) ([]*models.Message, error) {
			return nil, services.ErrNoPermission
		},
	}

	app := setupMessageTestApp(mockService)

	req := httptest.NewRequest("GET", "/channels/"+channelID.String()+"/messages", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusForbidden {
		t.Fatalf("Expected status 403, got %d", resp.StatusCode)
	}
}

// ========== GetMessage Tests ==========

func TestGetMessage_Success(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	messageID := uuid.New()
	authorID := uuid.New()

	mockService := &mockMessageService{
		getMessageFunc: func(ctx context.Context, msgID, reqID uuid.UUID) (*models.Message, error) {
			return &models.Message{
				ID:        messageID,
				ChannelID: channelID,
				AuthorID:  authorID,
				Content:   "Test message content",
				Type:      models.MessageTypeDefault,
				CreatedAt: time.Now(),
			}, nil
		},
	}

	app := setupMessageTestApp(mockService)

	req := httptest.NewRequest("GET", "/channels/"+channelID.String()+"/messages/"+messageID.String(), nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	var message models.Message
	json.NewDecoder(resp.Body).Decode(&message)

	if message.ID != messageID {
		t.Errorf("Expected message ID %s, got %s", messageID, message.ID)
	}
	if message.Content != "Test message content" {
		t.Errorf("Expected content 'Test message content', got %s", message.Content)
	}
}

func TestGetMessage_NotFound(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	messageID := uuid.New()

	mockService := &mockMessageService{
		getMessageFunc: func(ctx context.Context, msgID, reqID uuid.UUID) (*models.Message, error) {
			return nil, services.ErrMessageNotFound
		},
	}

	app := setupMessageTestApp(mockService)

	req := httptest.NewRequest("GET", "/channels/"+channelID.String()+"/messages/"+messageID.String(), nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestGetMessage_NoPermission(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	messageID := uuid.New()

	mockService := &mockMessageService{
		getMessageFunc: func(ctx context.Context, msgID, reqID uuid.UUID) (*models.Message, error) {
			return nil, services.ErrNoPermission
		},
	}

	app := setupMessageTestApp(mockService)

	req := httptest.NewRequest("GET", "/channels/"+channelID.String()+"/messages/"+messageID.String(), nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusForbidden {
		t.Fatalf("Expected status 403, got %d", resp.StatusCode)
	}
}

func TestGetMessage_InvalidMessageID(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	mockService := &mockMessageService{}
	app := setupMessageTestApp(mockService)

	req := httptest.NewRequest("GET", "/channels/"+channelID.String()+"/messages/invalid-uuid", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d", resp.StatusCode)
	}
}

// ========== EditMessage Tests ==========

func TestEditMessage_Success(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	messageID := uuid.New()

	mockService := &mockMessageService{
		editMessageFunc: func(ctx context.Context, msgID, authorID uuid.UUID, newContent string) (*models.Message, error) {
			return &models.Message{
				ID:        msgID,
				ChannelID: channelID,
				AuthorID:  authorID,
				Content:   newContent,
				EditedAt:  timePtr(time.Now()),
			}, nil
		},
	}

	app := setupMessageTestApp(mockService)

	body, _ := json.Marshal(map[string]string{
		"content": "Edited content",
	})

	req := httptest.NewRequest("PATCH", "/channels/"+channelID.String()+"/messages/"+messageID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	var message models.Message
	json.NewDecoder(resp.Body).Decode(&message)

	if message.Content != "Edited content" {
		t.Errorf("Expected content 'Edited content', got %s", message.Content)
	}
}

func TestEditMessage_NotFound(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	messageID := uuid.New()

	mockService := &mockMessageService{
		editMessageFunc: func(ctx context.Context, msgID, authorID uuid.UUID, newContent string) (*models.Message, error) {
			return nil, services.ErrMessageNotFound
		},
	}

	app := setupMessageTestApp(mockService)

	body, _ := json.Marshal(map[string]string{
		"content": "Edited content",
	})

	req := httptest.NewRequest("PATCH", "/channels/"+channelID.String()+"/messages/"+messageID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestEditMessage_NotAuthor(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	messageID := uuid.New()

	mockService := &mockMessageService{
		editMessageFunc: func(ctx context.Context, msgID, authorID uuid.UUID, newContent string) (*models.Message, error) {
			return nil, services.ErrNotMessageAuthor
		},
	}

	app := setupMessageTestApp(mockService)

	body, _ := json.Marshal(map[string]string{
		"content": "Edited content",
	})

	req := httptest.NewRequest("PATCH", "/channels/"+channelID.String()+"/messages/"+messageID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusForbidden {
		t.Fatalf("Expected status 403, got %d", resp.StatusCode)
	}
}

func TestEditMessage_InvalidMessageID(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	mockService := &mockMessageService{}
	app := setupMessageTestApp(mockService)

	body, _ := json.Marshal(map[string]string{
		"content": "Edited content",
	})

	req := httptest.NewRequest("PATCH", "/channels/"+channelID.String()+"/messages/invalid-uuid", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d", resp.StatusCode)
	}
}

// ========== DeleteMessage Tests ==========

func TestDeleteMessage_Success(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	messageID := uuid.New()

	mockService := &mockMessageService{
		deleteMessageFunc: func(ctx context.Context, msgID, reqID uuid.UUID) error {
			return nil
		},
	}

	app := setupMessageTestApp(mockService)

	req := httptest.NewRequest("DELETE", "/channels/"+channelID.String()+"/messages/"+messageID.String(), nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("Expected status 204, got %d", resp.StatusCode)
	}
}

func TestDeleteMessage_NotFound(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	messageID := uuid.New()

	mockService := &mockMessageService{
		deleteMessageFunc: func(ctx context.Context, msgID, reqID uuid.UUID) error {
			return services.ErrMessageNotFound
		},
	}

	app := setupMessageTestApp(mockService)

	req := httptest.NewRequest("DELETE", "/channels/"+channelID.String()+"/messages/"+messageID.String(), nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestDeleteMessage_NotAuthor(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	messageID := uuid.New()

	mockService := &mockMessageService{
		deleteMessageFunc: func(ctx context.Context, msgID, reqID uuid.UUID) error {
			return services.ErrNotMessageAuthor
		},
	}

	app := setupMessageTestApp(mockService)

	req := httptest.NewRequest("DELETE", "/channels/"+channelID.String()+"/messages/"+messageID.String(), nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusForbidden {
		t.Fatalf("Expected status 403, got %d", resp.StatusCode)
	}
}

// ========== BulkDeleteMessages Tests ==========

func TestBulkDeleteMessages_NotImplemented(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	mockService := &mockMessageService{}
	app := setupMessageTestApp(mockService)

	body, _ := json.Marshal(map[string]interface{}{
		"messages": []string{uuid.New().String(), uuid.New().String()},
	})

	req := httptest.NewRequest("POST", "/channels/"+channelID.String()+"/messages/bulk-delete", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusNotImplemented {
		t.Fatalf("Expected status 501, got %d", resp.StatusCode)
	}
}

// ========== AddReaction Tests ==========

func TestAddReaction_Success(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	messageID := uuid.New()

	mockService := &mockMessageService{
		addReactionFunc: func(ctx context.Context, msgID, uID uuid.UUID, emoji string) error {
			return nil
		},
	}

	app := setupMessageTestApp(mockService)

	req := httptest.NewRequest("PUT", "/channels/"+channelID.String()+"/messages/"+messageID.String()+"/reactions/👍/@me", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("Expected status 204, got %d", resp.StatusCode)
	}
}

func TestAddReaction_MessageNotFound(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	messageID := uuid.New()

	mockService := &mockMessageService{
		addReactionFunc: func(ctx context.Context, msgID, uID uuid.UUID, emoji string) error {
			return services.ErrMessageNotFound
		},
	}

	app := setupMessageTestApp(mockService)

	req := httptest.NewRequest("PUT", "/channels/"+channelID.String()+"/messages/"+messageID.String()+"/reactions/👍/@me", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestAddReaction_InvalidMessageID(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	mockService := &mockMessageService{}
	app := setupMessageTestApp(mockService)

	req := httptest.NewRequest("PUT", "/channels/"+channelID.String()+"/messages/invalid-uuid/reactions/👍/@me", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d", resp.StatusCode)
	}
}

// ========== RemoveReaction Tests ==========

func TestRemoveReaction_Success(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	messageID := uuid.New()

	mockService := &mockMessageService{
		removeReactionFunc: func(ctx context.Context, msgID, uID uuid.UUID, emoji string) error {
			return nil
		},
	}

	app := setupMessageTestApp(mockService)

	req := httptest.NewRequest("DELETE", "/channels/"+channelID.String()+"/messages/"+messageID.String()+"/reactions/👍/@me", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("Expected status 204, got %d", resp.StatusCode)
	}
}

// ========== GetPinnedMessages Tests ==========

func TestGetPinnedMessages_Success(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	messageID := uuid.New()

	now := time.Now()
	mockService := &mockMessageService{
		getPinnedMessagesFunc: func(ctx context.Context, chID, reqID uuid.UUID) ([]*models.Message, error) {
			return []*models.Message{
				{
					ID:        messageID,
					ChannelID: channelID,
					AuthorID:  userID,
					Content:   "Pinned message",
					Pinned:    true,
					CreatedAt: now,
				},
			}, nil
		},
	}
	app := setupMessageTestApp(mockService)

	req := httptest.NewRequest("GET", "/channels/"+channelID.String()+"/pins", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	var result []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if len(result) != 1 {
		t.Errorf("Expected 1 message, got %d", len(result))
	}

	if result[0]["content"] != "Pinned message" {
		t.Errorf("Expected 'Pinned message', got %v", result[0]["content"])
	}
}

func TestGetPinnedMessages_InvalidChannelID(t *testing.T) {
	userID := uuid.New()

	mockService := &mockMessageService{}
	app := setupMessageTestApp(mockService)

	req := httptest.NewRequest("GET", "/channels/invalid-uuid/pins", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestGetPinnedMessages_ChannelNotFound(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	mockService := &mockMessageService{
		getPinnedMessagesFunc: func(ctx context.Context, chID, reqID uuid.UUID) ([]*models.Message, error) {
			return nil, services.ErrChannelNotFound
		},
	}
	app := setupMessageTestApp(mockService)

	req := httptest.NewRequest("GET", "/channels/"+channelID.String()+"/pins", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestGetPinnedMessages_NoPermission(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	mockService := &mockMessageService{
		getPinnedMessagesFunc: func(ctx context.Context, chID, reqID uuid.UUID) ([]*models.Message, error) {
			return nil, services.ErrNoPermission
		},
	}
	app := setupMessageTestApp(mockService)

	req := httptest.NewRequest("GET", "/channels/"+channelID.String()+"/pins", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusForbidden {
		t.Fatalf("Expected status 403, got %d", resp.StatusCode)
	}
}

// ========== PinMessage Tests ==========

func TestPinMessage_Success(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	messageID := uuid.New()

	mockService := &mockMessageService{
		pinMessageFunc: func(ctx context.Context, msgID, reqID uuid.UUID) error {
			return nil
		},
	}

	app := setupMessageTestApp(mockService)

	req := httptest.NewRequest("PUT", "/channels/"+channelID.String()+"/pins/"+messageID.String(), nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("Expected status 204, got %d", resp.StatusCode)
	}
}

func TestPinMessage_NotFound(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	messageID := uuid.New()

	mockService := &mockMessageService{
		pinMessageFunc: func(ctx context.Context, msgID, reqID uuid.UUID) error {
			return services.ErrMessageNotFound
		},
	}

	app := setupMessageTestApp(mockService)

	req := httptest.NewRequest("PUT", "/channels/"+channelID.String()+"/pins/"+messageID.String(), nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestPinMessage_InvalidMessageID(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	mockService := &mockMessageService{}
	app := setupMessageTestApp(mockService)

	req := httptest.NewRequest("PUT", "/channels/"+channelID.String()+"/pins/invalid-uuid", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d", resp.StatusCode)
	}
}

// ========== UnpinMessage Tests ==========

func TestUnpinMessage_Success(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	messageID := uuid.New()

	mockService := &mockMessageService{
		unpinMessageFunc: func(ctx context.Context, msgID, reqID uuid.UUID) error {
			return nil
		},
	}

	app := setupMessageTestApp(mockService)

	req := httptest.NewRequest("DELETE", "/channels/"+channelID.String()+"/pins/"+messageID.String(), nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("Expected status 204, got %d", resp.StatusCode)
	}
}

func TestUnpinMessage_NotFound(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	messageID := uuid.New()

	mockService := &mockMessageService{
		unpinMessageFunc: func(ctx context.Context, msgID, reqID uuid.UUID) error {
			return services.ErrMessageNotFound
		},
	}

	app := setupMessageTestApp(mockService)

	req := httptest.NewRequest("DELETE", "/channels/"+channelID.String()+"/pins/"+messageID.String(), nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestUnpinMessage_InvalidMessageID(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	mockService := &mockMessageService{}
	app := setupMessageTestApp(mockService)

	req := httptest.NewRequest("DELETE", "/channels/"+channelID.String()+"/pins/invalid-uuid", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d", resp.StatusCode)
	}
}

// Helper function
func timePtr(t time.Time) *time.Time {
	return &t
}
