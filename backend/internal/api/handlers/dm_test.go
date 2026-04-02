package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"hearth/internal/models"
)

// Mock services for DM tests

type mockDMService struct {
	addUserFunc    func(ctx context.Context, channelID, requesterID, userID uuid.UUID) (*models.Channel, error)
	removeUserFunc func(ctx context.Context, channelID, requesterID, userID uuid.UUID) error
	leaveDMFunc    func(ctx context.Context, channelID, userID uuid.UUID) error
	transferFunc   func(ctx context.Context, channelID, currentOwnerID, newOwnerID uuid.UUID) (*models.Channel, error)
}

func (m *mockDMService) AddUserToGroupDM(ctx context.Context, channelID, requesterID, userID uuid.UUID) (*models.Channel, error) {
	if m.addUserFunc != nil {
		return m.addUserFunc(ctx, channelID, requesterID, userID)
	}
	return nil, nil
}

func (m *mockDMService) RemoveUserFromGroupDM(ctx context.Context, channelID, requesterID, userID uuid.UUID) error {
	if m.removeUserFunc != nil {
		return m.removeUserFunc(ctx, channelID, requesterID, userID)
	}
	return nil
}

func (m *mockDMService) LeaveDM(ctx context.Context, channelID, userID uuid.UUID) error {
	if m.leaveDMFunc != nil {
		return m.leaveDMFunc(ctx, channelID, userID)
	}
	return nil
}

func (m *mockDMService) TransferGroupDMOwnership(ctx context.Context, channelID, currentOwnerID, newOwnerID uuid.UUID) (*models.Channel, error) {
	if m.transferFunc != nil {
		return m.transferFunc(ctx, channelID, currentOwnerID, newOwnerID)
	}
	return nil, nil
}

type mockDMChannelService struct {
	getUserDMsFunc    func(ctx context.Context, userID uuid.UUID) ([]*models.Channel, error)
	getOrCreateDMFunc func(ctx context.Context, userID, recipientID uuid.UUID) (*models.Channel, error)
	createGroupDMFunc func(ctx context.Context, ownerID uuid.UUID, name string, recipientIDs []uuid.UUID) (*models.Channel, error)
}

func (m *mockDMChannelService) GetUserDMs(ctx context.Context, userID uuid.UUID) ([]*models.Channel, error) {
	if m.getUserDMsFunc != nil {
		return m.getUserDMsFunc(ctx, userID)
	}
	return nil, nil
}

func (m *mockDMChannelService) GetOrCreateDM(ctx context.Context, userID, recipientID uuid.UUID) (*models.Channel, error) {
	if m.getOrCreateDMFunc != nil {
		return m.getOrCreateDMFunc(ctx, userID, recipientID)
	}
	return nil, nil
}

func (m *mockDMChannelService) CreateGroupDM(ctx context.Context, ownerID uuid.UUID, name string, recipientIDs []uuid.UUID) (*models.Channel, error) {
	if m.createGroupDMFunc != nil {
		return m.createGroupDMFunc(ctx, ownerID, name, recipientIDs)
	}
	return nil, nil
}

type mockDMMessageService struct {
	getMessagesFunc func(ctx context.Context, channelID, requesterID uuid.UUID, before, after *uuid.UUID, limit int) ([]*models.Message, error)
	sendMessageFunc func(ctx context.Context, authorID, channelID uuid.UUID, content string, attachments []*models.Attachment, replyTo *uuid.UUID, stickerID *uuid.UUID) (*models.Message, error)
}

func (m *mockDMMessageService) GetMessages(ctx context.Context, channelID, requesterID uuid.UUID, before, after *uuid.UUID, limit int) ([]*models.Message, error) {
	if m.getMessagesFunc != nil {
		return m.getMessagesFunc(ctx, channelID, requesterID, before, after, limit)
	}
	return nil, nil
}

func (m *mockDMMessageService) SendMessage(ctx context.Context, authorID, channelID uuid.UUID, content string, attachments []*models.Attachment, replyTo *uuid.UUID, stickerID *uuid.UUID) (*models.Message, error) {
	if m.sendMessageFunc != nil {
		return m.sendMessageFunc(ctx, authorID, channelID, content, attachments, replyTo, stickerID)
	}
	return nil, nil
}

func setupDMTestApp(
	dmSvc *mockDMService,
	channelSvc *mockDMChannelService,
	msgSvc *mockDMMessageService,
) *fiber.App {
	app := fiber.New()

	app.Use(func(c *fiber.Ctx) error {
		userIDStr := c.Get("X-Test-User-ID")
		if userIDStr != "" {
			userID, err := uuid.Parse(userIDStr)
			if err == nil {
				c.Locals("userID", userID)
			} else {
				c.Locals("userID", uuid.Nil)
			}
		} else {
			c.Locals("userID", uuid.MustParse("00000000-0000-0000-0000-000000000000"))
		}
		return c.Next()
	})

	// POST /dms - Create DM
	app.Post("/dms", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)

		var req struct {
			RecipientID string `json:"recipient_id"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid request body",
			})
		}

		if req.RecipientID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "recipient_id is required",
			})
		}

		recipientUUID, err := uuid.Parse(req.RecipientID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid recipient_id",
			})
		}

		if recipientUUID == userID {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "cannot create DM with yourself",
			})
		}

		channel, err := channelSvc.GetOrCreateDM(c.Context(), userID, recipientUUID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"id":         channel.ID,
			"type":       channel.Type,
			"recipients": channel.Recipients,
			"created_at": channel.CreatedAt,
		})
	})

	// POST /dms/group - Create Group DM
	app.Post("/dms/group", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)

		var req struct {
			RecipientIDs []string `json:"recipient_ids"`
			Name         *string  `json:"name"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid request body",
			})
		}

		if len(req.RecipientIDs) == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "at least one recipient is required",
			})
		}

		if len(req.RecipientIDs) > 49 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "too many recipients (max 49)",
			})
		}

		recipientUUIDs := make([]uuid.UUID, len(req.RecipientIDs))
		for i, id := range req.RecipientIDs {
			parsed, err := uuid.Parse(id)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": fmt.Sprintf("invalid recipient_id: %s", id),
				})
			}
			recipientUUIDs[i] = parsed
		}

		name := ""
		if req.Name != nil {
			name = *req.Name
		}

		channel, err := channelSvc.CreateGroupDM(c.Context(), userID, name, recipientUUIDs)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"id":         channel.ID,
			"type":       channel.Type,
			"name":       channel.Name,
			"owner_id":   channel.OwnerID,
			"recipients": channel.Recipients,
			"created_at": channel.CreatedAt,
		})
	})

	// GET /dms/:channelId/messages - Get DM Messages
	app.Get("/dms/:channelId/messages", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)

		channelID, err := uuid.Parse(c.Params("channelId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid channel ID",
			})
		}

		// Verify user is participant
		channels, err := channelSvc.GetUserDMs(c.Context(), userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		isParticipant := false
		for _, ch := range channels {
			if ch.ID == channelID {
				isParticipant = true
				break
			}
		}
		if !isParticipant {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "you are not a participant of this DM",
			})
		}

		messages, err := msgSvc.GetMessages(c.Context(), channelID, userID, nil, nil, 50)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(messages)
	})

	// POST /dms/:channelId/messages - Send DM Message
	app.Post("/dms/:channelId/messages", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)

		channelID, err := uuid.Parse(c.Params("channelId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid channel ID",
			})
		}

		// Verify user is participant
		channels, err := channelSvc.GetUserDMs(c.Context(), userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		isParticipant := false
		for _, ch := range channels {
			if ch.ID == channelID {
				isParticipant = true
				break
			}
		}
		if !isParticipant {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "you are not a participant of this DM",
			})
		}

		var req struct {
			Content   string  `json:"content"`
			ReplyToID *string `json:"reply_to_id"`
			StickerID *string `json:"sticker_id"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid request body",
			})
		}

		if strings.TrimSpace(req.Content) == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "content is required",
			})
		}

		var replyTo *uuid.UUID
		if req.ReplyToID != nil {
			parsed, err := uuid.Parse(*req.ReplyToID)
			if err == nil {
				replyTo = &parsed
			}
		}

		var stickerID *uuid.UUID
		if req.StickerID != nil {
			parsed, err := uuid.Parse(*req.StickerID)
			if err == nil {
				stickerID = &parsed
			}
		}

		msg, err := msgSvc.SendMessage(c.Context(), userID, channelID, req.Content, nil, replyTo, stickerID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.Status(fiber.StatusCreated).JSON(msg)
	})

	// POST /dms/:channelId/participants - Add Participant
	app.Post("/dms/:channelId/participants", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)

		channelID, err := uuid.Parse(c.Params("channelId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid channel ID",
			})
		}

		var req struct {
			UserID string `json:"user_id"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid request body",
			})
		}

		if req.UserID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "user_id is required",
			})
		}

		targetID, err := uuid.Parse(req.UserID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid user_id",
			})
		}

		channel, err := dmSvc.AddUserToGroupDM(c.Context(), channelID, userID, targetID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"id":         channel.ID,
			"type":       channel.Type,
			"recipients": channel.Recipients,
		})
	})

	// DELETE /dms/:channelId - Leave DM
	app.Delete("/dms/:channelId", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)

		channelID, err := uuid.Parse(c.Params("channelId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid channel ID",
			})
		}

		err = dmSvc.LeaveDM(c.Context(), channelID, userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.SendStatus(fiber.StatusNoContent)
	})

	// PATCH /dms/:channelId/owner - Transfer Ownership
	app.Patch("/dms/:channelId/owner", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)

		channelID, err := uuid.Parse(c.Params("channelId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid channel ID",
			})
		}

		var req struct {
			UserID string `json:"user_id"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid request body",
			})
		}

		if req.UserID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "user_id is required",
			})
		}

		newOwnerID, err := uuid.Parse(req.UserID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid user_id",
			})
		}

		// Verify user is participant
		channels, err := channelSvc.GetUserDMs(c.Context(), userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		isParticipant := false
		for _, ch := range channels {
			if ch.ID == channelID && ch.Type == models.ChannelTypeGroupDM {
				isParticipant = true
				break
			}
		}
		if !isParticipant {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "you are not a participant of this group DM",
			})
		}

		channel, err := dmSvc.TransferGroupDMOwnership(c.Context(), channelID, userID, newOwnerID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"id":         channel.ID,
			"type":       channel.Type,
			"owner_id":   channel.OwnerID,
			"recipients": channel.Recipients,
		})
	})

	return app
}

// Test CreateDM - Success
func TestCreateDM_Success(t *testing.T) {
	testUserID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	recipientID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	channelID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	channelSvc := &mockDMChannelService{
		getOrCreateDMFunc: func(ctx context.Context, userID, recID uuid.UUID) (*models.Channel, error) {
			assert.Equal(t, testUserID, userID)
			assert.Equal(t, recipientID, recID)
			return &models.Channel{
				ID:         channelID,
				Type:       models.ChannelTypeDM,
				Recipients: []uuid.UUID{testUserID, recipientID},
				CreatedAt:  time.Now(),
			}, nil
		},
	}

	app := setupDMTestApp(&mockDMService{}, channelSvc, &mockDMMessageService{})
	t.Cleanup(func() { _ = app.Shutdown() })

	body := fmt.Sprintf(`{"recipient_id":"%s"}`, recipientID.String())
	req := httptest.NewRequest("POST", "/dms", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", testUserID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, channelID.String(), result["id"])
}

// Test CreateDM - Missing recipient_id
func TestCreateDM_MissingRecipientID(t *testing.T) {
	app := setupDMTestApp(&mockDMService{}, &mockDMChannelService{}, &mockDMMessageService{})
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{}`
	req := httptest.NewRequest("POST", "/dms", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", "11111111-1111-1111-1111-111111111111")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Contains(t, result["error"], "recipient_id is required")
}

// Test CreateDM - Self DM
func TestCreateDM_SelfDM(t *testing.T) {
	testUserID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	app := setupDMTestApp(&mockDMService{}, &mockDMChannelService{}, &mockDMMessageService{})
	t.Cleanup(func() { _ = app.Shutdown() })

	body := fmt.Sprintf(`{"recipient_id":"%s"}`, testUserID.String())
	req := httptest.NewRequest("POST", "/dms", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", testUserID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Contains(t, result["error"], "cannot create DM with yourself")
}

// Test CreateDM - Invalid recipient UUID
func TestCreateDM_InvalidRecipientUUID(t *testing.T) {
	app := setupDMTestApp(&mockDMService{}, &mockDMChannelService{}, &mockDMMessageService{})
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"recipient_id":"not-a-uuid"}`
	req := httptest.NewRequest("POST", "/dms", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", "11111111-1111-1111-1111-111111111111")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Contains(t, result["error"], "invalid recipient_id")
}

// Test CreateGroupDM - Success
func TestCreateGroupDM_Success(t *testing.T) {
	testUserID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	recipient1 := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	recipient2 := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	channelID := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	groupName := "Test Group"

	channelSvc := &mockDMChannelService{
		createGroupDMFunc: func(ctx context.Context, ownerID uuid.UUID, name string, recipientIDs []uuid.UUID) (*models.Channel, error) {
			assert.Equal(t, testUserID, ownerID)
			assert.Equal(t, groupName, name)
			assert.Len(t, recipientIDs, 2)
			return &models.Channel{
				ID:         channelID,
				Type:       models.ChannelTypeGroupDM,
				Name:       groupName,
				OwnerID:    &testUserID,
				Recipients: []uuid.UUID{testUserID, recipient1, recipient2},
				CreatedAt:  time.Now(),
			}, nil
		},
	}

	app := setupDMTestApp(&mockDMService{}, channelSvc, &mockDMMessageService{})
	t.Cleanup(func() { _ = app.Shutdown() })

	body := fmt.Sprintf(`{"recipient_ids":["%s","%s"],"name":"%s"}`, recipient1.String(), recipient2.String(), groupName)
	req := httptest.NewRequest("POST", "/dms/group", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", testUserID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, channelID.String(), result["id"])
}

// Test CreateGroupDM - No recipients
func TestCreateGroupDM_NoRecipients(t *testing.T) {
	app := setupDMTestApp(&mockDMService{}, &mockDMChannelService{}, &mockDMMessageService{})
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"recipient_ids":[]}`
	req := httptest.NewRequest("POST", "/dms/group", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", "11111111-1111-1111-1111-111111111111")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Contains(t, result["error"], "at least one recipient")
}

// Test CreateGroupDM - Too many recipients
func TestCreateGroupDM_TooManyRecipients(t *testing.T) {
	app := setupDMTestApp(&mockDMService{}, &mockDMChannelService{}, &mockDMMessageService{})
	t.Cleanup(func() { _ = app.Shutdown() })

	// Build 50 recipient IDs (exceeds max of 49)
	ids := make([]string, 50)
	for i := range ids {
		ids[i] = fmt.Sprintf(`"%s"`, uuid.New().String())
	}
	body := fmt.Sprintf(`{"recipient_ids":[%s]}`, strings.Join(ids, ","))

	req := httptest.NewRequest("POST", "/dms/group", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", "11111111-1111-1111-1111-111111111111")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Contains(t, result["error"], "too many recipients")
}

// Test GetDMMessages - Success
func TestGetDMMessages_Success(t *testing.T) {
	testUserID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	channelID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	msgID := uuid.MustParse("55555555-5555-5555-5555-555555555555")

	channelSvc := &mockDMChannelService{
		getUserDMsFunc: func(ctx context.Context, userID uuid.UUID) ([]*models.Channel, error) {
			return []*models.Channel{
				{ID: channelID, Type: models.ChannelTypeDM},
			}, nil
		},
	}

	msgSvc := &mockDMMessageService{
		getMessagesFunc: func(ctx context.Context, chID, reqID uuid.UUID, before, after *uuid.UUID, limit int) ([]*models.Message, error) {
			assert.Equal(t, channelID, chID)
			return []*models.Message{
				{ID: msgID, ChannelID: channelID, AuthorID: testUserID, Content: "hello", CreatedAt: time.Now()},
			}, nil
		},
	}

	app := setupDMTestApp(&mockDMService{}, channelSvc, msgSvc)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("GET", fmt.Sprintf("/dms/%s/messages", channelID.String()), nil)
	req.Header.Set("X-Test-User-ID", testUserID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var messages []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&messages)
	assert.NoError(t, err)
	assert.Len(t, messages, 1)
}

// Test GetDMMessages - Invalid channel ID
func TestGetDMMessages_InvalidChannelID(t *testing.T) {
	app := setupDMTestApp(&mockDMService{}, &mockDMChannelService{}, &mockDMMessageService{})
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("GET", "/dms/invalid-uuid/messages", nil)
	req.Header.Set("X-Test-User-ID", "11111111-1111-1111-1111-111111111111")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

// Test GetDMMessages - Not participant
func TestGetDMMessages_NotParticipant(t *testing.T) {
	testUserID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	channelID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	otherChannelID := uuid.MustParse("44444444-4444-4444-4444-444444444444")

	channelSvc := &mockDMChannelService{
		getUserDMsFunc: func(ctx context.Context, userID uuid.UUID) ([]*models.Channel, error) {
			// Return a different channel - user is not participant of the requested one
			return []*models.Channel{
				{ID: otherChannelID, Type: models.ChannelTypeDM},
			}, nil
		},
	}

	app := setupDMTestApp(&mockDMService{}, channelSvc, &mockDMMessageService{})
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("GET", fmt.Sprintf("/dms/%s/messages", channelID.String()), nil)
	req.Header.Set("X-Test-User-ID", testUserID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Contains(t, result["error"], "not a participant")
}

// Test SendDMMessage - Success
func TestSendDMMessage_Success(t *testing.T) {
	testUserID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	channelID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	msgID := uuid.MustParse("55555555-5555-5555-5555-555555555555")

	channelSvc := &mockDMChannelService{
		getUserDMsFunc: func(ctx context.Context, userID uuid.UUID) ([]*models.Channel, error) {
			return []*models.Channel{
				{ID: channelID, Type: models.ChannelTypeDM},
			}, nil
		},
	}

	msgSvc := &mockDMMessageService{
		sendMessageFunc: func(ctx context.Context, authorID, chID uuid.UUID, content string, attachments []*models.Attachment, replyTo *uuid.UUID, stickerID *uuid.UUID) (*models.Message, error) {
			assert.Equal(t, testUserID, authorID)
			assert.Equal(t, channelID, chID)
			assert.Equal(t, "hello world", content)
			return &models.Message{
				ID:        msgID,
				ChannelID: channelID,
				AuthorID:  testUserID,
				Content:   content,
				CreatedAt: time.Now(),
			}, nil
		},
	}

	app := setupDMTestApp(&mockDMService{}, channelSvc, msgSvc)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"content":"hello world"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/dms/%s/messages", channelID.String()), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", testUserID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, msgID.String(), result["id"])
}

// Test SendDMMessage - Empty content
func TestSendDMMessage_EmptyContent(t *testing.T) {
	testUserID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	channelID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	channelSvc := &mockDMChannelService{
		getUserDMsFunc: func(ctx context.Context, userID uuid.UUID) ([]*models.Channel, error) {
			return []*models.Channel{
				{ID: channelID, Type: models.ChannelTypeDM},
			}, nil
		},
	}

	app := setupDMTestApp(&mockDMService{}, channelSvc, &mockDMMessageService{})
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"content":""}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/dms/%s/messages", channelID.String()), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", testUserID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Contains(t, result["error"], "content is required")
}

// Test SendDMMessage - Not participant
func TestSendDMMessage_NotParticipant(t *testing.T) {
	testUserID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	channelID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	channelSvc := &mockDMChannelService{
		getUserDMsFunc: func(ctx context.Context, userID uuid.UUID) ([]*models.Channel, error) {
			return []*models.Channel{}, nil // No channels - not a participant
		},
	}

	app := setupDMTestApp(&mockDMService{}, channelSvc, &mockDMMessageService{})
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"content":"hello"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/dms/%s/messages", channelID.String()), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", testUserID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

// Test AddParticipant - Success
func TestAddParticipant_Success(t *testing.T) {
	testUserID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	channelID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	newUserID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	dmSvc := &mockDMService{
		addUserFunc: func(ctx context.Context, chID, requesterID, userID uuid.UUID) (*models.Channel, error) {
			assert.Equal(t, channelID, chID)
			assert.Equal(t, testUserID, requesterID)
			assert.Equal(t, newUserID, userID)
			return &models.Channel{
				ID:         channelID,
				Type:       models.ChannelTypeGroupDM,
				Recipients: []uuid.UUID{testUserID, newUserID},
			}, nil
		},
	}

	app := setupDMTestApp(dmSvc, &mockDMChannelService{}, &mockDMMessageService{})
	t.Cleanup(func() { _ = app.Shutdown() })

	body := fmt.Sprintf(`{"user_id":"%s"}`, newUserID.String())
	req := httptest.NewRequest("POST", fmt.Sprintf("/dms/%s/participants", channelID.String()), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", testUserID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

// Test AddParticipant - Missing user_id
func TestAddParticipant_MissingUserID(t *testing.T) {
	app := setupDMTestApp(&mockDMService{}, &mockDMChannelService{}, &mockDMMessageService{})
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{}`
	req := httptest.NewRequest("POST", "/dms/33333333-3333-3333-3333-333333333333/participants", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", "11111111-1111-1111-1111-111111111111")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Contains(t, result["error"], "user_id is required")
}

// Test LeaveDM - Success
func TestLeaveDM_Success(t *testing.T) {
	testUserID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	channelID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	dmSvc := &mockDMService{
		leaveDMFunc: func(ctx context.Context, chID, userID uuid.UUID) error {
			assert.Equal(t, channelID, chID)
			assert.Equal(t, testUserID, userID)
			return nil
		},
	}

	app := setupDMTestApp(dmSvc, &mockDMChannelService{}, &mockDMMessageService{})
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/dms/%s", channelID.String()), nil)
	req.Header.Set("X-Test-User-ID", testUserID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)
}

// Test LeaveDM - Invalid channel ID
func TestLeaveDM_InvalidChannelID(t *testing.T) {
	app := setupDMTestApp(&mockDMService{}, &mockDMChannelService{}, &mockDMMessageService{})
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("DELETE", "/dms/not-a-uuid", nil)
	req.Header.Set("X-Test-User-ID", "11111111-1111-1111-1111-111111111111")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Contains(t, result["error"], "invalid channel ID")
}

// Test TransferOwnership - Success
func TestTransferOwnership_Success(t *testing.T) {
	testUserID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	channelID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	newOwnerID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	ownerPtr := newOwnerID

	dmSvc := &mockDMService{
		transferFunc: func(ctx context.Context, chID, currentOwnerID, newOwner uuid.UUID) (*models.Channel, error) {
			assert.Equal(t, channelID, chID)
			assert.Equal(t, testUserID, currentOwnerID)
			assert.Equal(t, newOwnerID, newOwner)
			return &models.Channel{
				ID:         channelID,
				Type:       models.ChannelTypeGroupDM,
				OwnerID:    &ownerPtr,
				Recipients: []uuid.UUID{testUserID, newOwnerID},
			}, nil
		},
	}

	channelSvc := &mockDMChannelService{
		getUserDMsFunc: func(ctx context.Context, userID uuid.UUID) ([]*models.Channel, error) {
			return []*models.Channel{
				{ID: channelID, Type: models.ChannelTypeGroupDM},
			}, nil
		},
	}

	app := setupDMTestApp(dmSvc, channelSvc, &mockDMMessageService{})
	t.Cleanup(func() { _ = app.Shutdown() })

	body := fmt.Sprintf(`{"user_id":"%s"}`, newOwnerID.String())
	req := httptest.NewRequest("PATCH", fmt.Sprintf("/dms/%s/owner", channelID.String()), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", testUserID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, channelID.String(), result["id"])
}

// Test TransferOwnership - Not participant
func TestTransferOwnership_NotParticipant(t *testing.T) {
	testUserID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	channelID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	newOwnerID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	channelSvc := &mockDMChannelService{
		getUserDMsFunc: func(ctx context.Context, userID uuid.UUID) ([]*models.Channel, error) {
			return []*models.Channel{}, nil // Not a participant
		},
	}

	app := setupDMTestApp(&mockDMService{}, channelSvc, &mockDMMessageService{})
	t.Cleanup(func() { _ = app.Shutdown() })

	body := fmt.Sprintf(`{"user_id":"%s"}`, newOwnerID.String())
	req := httptest.NewRequest("PATCH", fmt.Sprintf("/dms/%s/owner", channelID.String()), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", testUserID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

// Test TransferOwnership - Missing user_id
func TestTransferOwnership_MissingUserID(t *testing.T) {
	testUserID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	channelID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	channelSvc := &mockDMChannelService{
		getUserDMsFunc: func(ctx context.Context, userID uuid.UUID) ([]*models.Channel, error) {
			return []*models.Channel{
				{ID: channelID, Type: models.ChannelTypeGroupDM},
			}, nil
		},
	}

	app := setupDMTestApp(&mockDMService{}, channelSvc, &mockDMMessageService{})
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{}`
	req := httptest.NewRequest("PATCH", fmt.Sprintf("/dms/%s/owner", channelID.String()), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", testUserID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Contains(t, result["error"], "user_id is required")
}

// Test TransferOwnership - Invalid channel ID
func TestTransferOwnership_InvalidChannelID(t *testing.T) {
	app := setupDMTestApp(&mockDMService{}, &mockDMChannelService{}, &mockDMMessageService{})
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"user_id":"22222222-2222-2222-2222-222222222222"}`
	req := httptest.NewRequest("PATCH", "/dms/not-a-uuid/owner", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", "11111111-1111-1111-1111-111111111111")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Contains(t, result["error"], "invalid channel ID")
}
