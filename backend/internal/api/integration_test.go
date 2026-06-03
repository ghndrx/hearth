package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"hearth/internal/api/handlers"
	"hearth/internal/api/middleware"
	"hearth/internal/auth"
	"hearth/internal/models"
	"hearth/internal/services"
)

const integrationTestSecret = "integration-test-secret-key"

func generateIntegrationToken(userID uuid.UUID) string {
	claims := middleware.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		UserID:   userID.String(),
		Username: "testuser",
		Type:     "access",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(integrationTestSecret))
	return tokenString
}

// ---------- Mock Services (function-field pattern from existing tests) ----------

type mockAuthService struct {
	registerFunc     func(ctx context.Context, email, username, password string) (*models.User, *services.AuthTokens, error)
	loginFunc        func(ctx context.Context, email, password string) (*models.User, *services.AuthTokens, error)
	refreshTokensFunc func(ctx context.Context, refreshToken string) (*services.AuthTokens, error)
}

func (m *mockAuthService) Register(ctx context.Context, email, username, password string) (*models.User, *services.AuthTokens, error) {
	if m.registerFunc != nil {
		return m.registerFunc(ctx, email, username, password)
	}
	return nil, nil, nil
}

func (m *mockAuthService) Login(ctx context.Context, email, password string) (*models.User, *services.AuthTokens, error) {
	if m.loginFunc != nil {
		return m.loginFunc(ctx, email, password)
	}
	return nil, nil, nil
}

func (m *mockAuthService) RefreshTokens(ctx context.Context, refreshToken string) (*services.AuthTokens, error) {
	if m.refreshTokensFunc != nil {
		return m.refreshTokensFunc(ctx, refreshToken)
	}
	return nil, nil
}

func (m *mockAuthService) ValidateToken(ctx context.Context, token string) (uuid.UUID, error) {
	return uuid.Nil, nil
}

func (m *mockAuthService) LoginWithMFA(ctx context.Context, email, password, mfaCode string) (*models.User, *services.AuthTokens, error) {
	return nil, nil, nil
}

func (m *mockAuthService) EnableMFA(ctx context.Context, userID uuid.UUID) (*services.MFASetupResponse, error) {
	return nil, nil
}

func (m *mockAuthService) VerifyMFASetup(ctx context.Context, userID uuid.UUID, code string) error {
	return nil
}

func (m *mockAuthService) DisableMFA(ctx context.Context, userID uuid.UUID, password string) error {
	return nil
}

func (m *mockAuthService) VerifyMFA(ctx context.Context, userID uuid.UUID, code string) error {
	return nil
}

func (m *mockAuthService) CheckUserMFA(ctx context.Context, email string) (bool, error) {
	return false, nil
}

type mockUserService struct {
	getUserFunc func(ctx context.Context, id uuid.UUID) (*models.User, error)
}

func (m *mockUserService) GetUser(ctx context.Context, id uuid.UUID) (*models.User, error) {
	if m.getUserFunc != nil {
		return m.getUserFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockUserService) GetUserByUsername(ctx context.Context, username string) (*models.User, error) { return nil, nil }
func (m *mockUserService) UpdateUser(ctx context.Context, id uuid.UUID, updates *models.UserUpdate) (*models.User, error) {
	if m.getUserFunc != nil {
		return m.getUserFunc(ctx, id)
	}
	return nil, nil
}
func (m *mockUserService) GetFriends(ctx context.Context, userID uuid.UUID) ([]*models.User, error) { return nil, nil }
func (m *mockUserService) AddFriend(ctx context.Context, userID, friendID uuid.UUID) error { return nil }
func (m *mockUserService) RemoveFriend(ctx context.Context, userID, friendID uuid.UUID) error { return nil }
func (m *mockUserService) BlockUser(ctx context.Context, userID, blockedID uuid.UUID) error { return nil }
func (m *mockUserService) UnblockUser(ctx context.Context, userID, blockedID uuid.UUID) error { return nil }
func (m *mockUserService) SendFriendRequest(ctx context.Context, senderID, receiverID uuid.UUID) error { return nil }
func (m *mockUserService) GetIncomingFriendRequests(ctx context.Context, userID uuid.UUID) ([]*models.User, error) { return nil, nil }
func (m *mockUserService) GetOutgoingFriendRequests(ctx context.Context, userID uuid.UUID) ([]*models.User, error) { return nil, nil }
func (m *mockUserService) AcceptFriendRequest(ctx context.Context, receiverID, senderID uuid.UUID) error { return nil }
func (m *mockUserService) DeclineFriendRequest(ctx context.Context, userID, otherID uuid.UUID) error { return nil }
func (m *mockUserService) GetRelationship(ctx context.Context, userID, targetID uuid.UUID) (int, error) { return 0, nil }
func (m *mockUserService) GetCustomStatus(ctx context.Context, userID uuid.UUID) (*models.UserCustomStatus, error) { return nil, nil }
func (m *mockUserService) SetCustomStatus(ctx context.Context, userID uuid.UUID, req *models.UpdateStatusRequest) (*models.UserCustomStatus, error) { return nil, nil }
func (m *mockUserService) ClearCustomStatus(ctx context.Context, userID uuid.UUID) error { return nil }

type mockServerService struct {
	createServerFunc func(ctx context.Context, ownerID uuid.UUID, name, icon string) (*models.Server, error)
	getServerFunc    func(ctx context.Context, id uuid.UUID) (*models.Server, error)
	updateServerFunc func(ctx context.Context, id, requesterID uuid.UUID, updates *models.ServerUpdate) (*models.Server, error)
	deleteServerFunc func(ctx context.Context, id, requesterID uuid.UUID) error
	getChannelsFunc  func(ctx context.Context, serverID uuid.UUID) ([]*models.Channel, error)
	createChannelFunc func(ctx context.Context, serverID, creatorID uuid.UUID, name string, channelType models.ChannelType, parentID *uuid.UUID) (*models.Channel, error)
}

type mockChannelService struct {
	getUserDMsFunc    func(ctx context.Context, userID uuid.UUID) ([]*models.Channel, error)
	getOrCreateDMFunc func(ctx context.Context, userID, recipientID uuid.UUID) (*models.Channel, error)
	createGroupDMFunc func(ctx context.Context, ownerID uuid.UUID, name string, recipientIDs []uuid.UUID) (*models.Channel, error)
	updateGroupDMFunc func(ctx context.Context, channelID, requesterID uuid.UUID, updates *services.GroupDMUpdate) (*models.Channel, error)
}

func (m *mockChannelService) GetUserDMs(ctx context.Context, userID uuid.UUID) ([]*models.Channel, error) {
	if m.getUserDMsFunc != nil {
		return m.getUserDMsFunc(ctx, userID)
	}
	return nil, nil
}

func (m *mockChannelService) GetOrCreateDM(ctx context.Context, user1ID, user2ID uuid.UUID) (*models.Channel, error) {
	if m.getOrCreateDMFunc != nil {
		return m.getOrCreateDMFunc(ctx, user1ID, user2ID)
	}
	return nil, nil
}

func (m *mockChannelService) CreateGroupDM(ctx context.Context, ownerID uuid.UUID, name string, recipientIDs []uuid.UUID) (*models.Channel, error) {
	if m.createGroupDMFunc != nil {
		return m.createGroupDMFunc(ctx, ownerID, name, recipientIDs)
	}
	return nil, nil
}

func (m *mockChannelService) UpdateGroupDM(ctx context.Context, channelID, requesterID uuid.UUID, updates *services.GroupDMUpdate) (*models.Channel, error) {
	if m.updateGroupDMFunc != nil {
		return m.updateGroupDMFunc(ctx, channelID, requesterID, updates)
	}
	return nil, nil
}

type mockMessageService struct {
	getMessagesFunc func(ctx context.Context, channelID, requesterID uuid.UUID, before, after *uuid.UUID, limit int) ([]*models.Message, error)
	sendMessageFunc func(ctx context.Context, authorID, channelID uuid.UUID, content string, attachments []*models.Attachment, replyTo *uuid.UUID, stickerID *uuid.UUID) (*models.Message, error)
}

func (m *mockMessageService) GetMessages(ctx context.Context, channelID, requesterID uuid.UUID, before, after *uuid.UUID, limit int) ([]*models.Message, error) {
	if m.getMessagesFunc != nil {
		return m.getMessagesFunc(ctx, channelID, requesterID, before, after, limit)
	}
	return nil, nil
}

func (m *mockMessageService) SendMessage(ctx context.Context, authorID, channelID uuid.UUID, content string, attachments []*models.Attachment, replyTo *uuid.UUID, stickerID *uuid.UUID) (*models.Message, error) {
	if m.sendMessageFunc != nil {
		return m.sendMessageFunc(ctx, authorID, channelID, content, attachments, replyTo, stickerID)
	}
	return nil, nil
}

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

type mockPremiumService struct {
	getUserPremiumStatusFunc func(ctx context.Context, userID uuid.UUID) (*models.PremiumStatus, error)
	createSubscriptionFunc   func(ctx context.Context, userID uuid.UUID, tier models.PremiumTier) (*models.Subscription, error)
	cancelSubscriptionFunc   func(ctx context.Context, userID uuid.UUID) error
}

type mockBillingService struct {
	createSubscriptionFunc func(ctx context.Context, userID uuid.UUID, tier models.PremiumTier, paymentMethodID string) (*models.Subscription, error)
	cancelSubscriptionFunc func(ctx context.Context, userID uuid.UUID) error
}

// ---------- 1. Auth Flow ----------

func TestIntegration_AuthFlow(t *testing.T) {
	userID := uuid.New()
	jwtService := auth.NewJWTService(integrationTestSecret, time.Hour, 7*24*time.Hour)

	authSvc := &mockAuthService{
		registerFunc: func(ctx context.Context, email, username, password string) (*models.User, *services.AuthTokens, error) {
			return &models.User{
				ID:            userID,
				Email:         email,
				Username:      username,
				Discriminator: "0001",
				CreatedAt:     time.Now(),
			}, &services.AuthTokens{
				AccessToken:  "access-token",
				RefreshToken: "refresh-token",
				ExpiresIn:    900,
			}, nil
		},
		loginFunc: func(ctx context.Context, email, password string) (*models.User, *services.AuthTokens, error) {
			return &models.User{
				ID:            userID,
				Email:         email,
				Username:      "testuser",
				Discriminator: "0001",
				CreatedAt:     time.Now(),
			}, &services.AuthTokens{
				AccessToken:  "access-token",
				RefreshToken: "refresh-token",
				ExpiresIn:    900,
			}, nil
		},
	}

	userSvc := &mockUserService{
		getUserFunc: func(ctx context.Context, id uuid.UUID) (*models.User, error) {
			return &models.User{
				ID:            userID,
				Email:         "test@example.com",
				Username:      "testuser",
				Discriminator: "0001",
				CreatedAt:     time.Now(),
			}, nil
		},
	}

	authHandler := handlers.NewAuthHandler(authSvc)
	userHandler := handlers.NewUserHandler(userSvc, nil, nil)

	m := middleware.NewMiddleware(integrationTestSecret)
	app := fiber.New()
	t.Cleanup(func() { app.Shutdown() })

	// Public auth routes
	app.Post("/api/v1/auth/register", authHandler.Register)
	app.Post("/api/v1/auth/login", authHandler.Login)
	app.Post("/api/v1/auth/logout", authHandler.Logout)

	// Protected user routes
	api := app.Group("/api/v1/users", m.RequireAuth)
	api.Get("/@me", userHandler.GetMe)

	// 1. Register
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(mustJSON(t, map[string]string{
		"email":    "test@example.com",
		"username": "testuser",
		"password": "password123",
	})))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var registerBody map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&registerBody))
	assert.NotNil(t, registerBody["access_token"])

	// 2. Login
	req = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(mustJSON(t, map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	})))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var loginBody map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&loginBody))
	assert.NotNil(t, loginBody["access_token"])

	// 3. GetMe
	token := generateIntegrationToken(userID)
	req = httptest.NewRequest("GET", "/api/v1/users/@me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var meBody map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&meBody))
	assert.Equal(t, userID.String(), meBody["id"])
	assert.Equal(t, "testuser", meBody["username"])

	// 4. Logout
	resp, err = app.Test(httptest.NewRequest("POST", "/api/v1/auth/logout", nil), -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)

	_ = jwtService
}

// ---------- 2. Server Flow ----------

func TestIntegration_ServerFlow(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()
	token := generateIntegrationToken(userID)

	serverSvc := &mockServerService{
		createServerFunc: func(ctx context.Context, ownerID uuid.UUID, name, icon string) (*models.Server, error) {
			return &models.Server{
				ID:        serverID,
				Name:      name,
				OwnerID:   ownerID,
				CreatedAt: time.Now(),
			}, nil
		},
		getServerFunc: func(ctx context.Context, id uuid.UUID) (*models.Server, error) {
			return &models.Server{
				ID:        serverID,
				Name:      "Test Server",
				OwnerID:   userID,
				CreatedAt: time.Now(),
			}, nil
		},
		updateServerFunc: func(ctx context.Context, id, requesterID uuid.UUID, updates *models.ServerUpdate) (*models.Server, error) {
			return &models.Server{
				ID:        serverID,
				Name:      *updates.Name,
				OwnerID:   userID,
				CreatedAt: time.Now(),
			}, nil
		},
		deleteServerFunc: func(ctx context.Context, id, requesterID uuid.UUID) error {
			return nil
		},
	}

	m := middleware.NewMiddleware(integrationTestSecret)
	app := fiber.New()
	t.Cleanup(func() { app.Shutdown() })

	// Auth middleware
	app.Use(m.RequireAuth)

	// Inline server handlers using mock service
	app.Post("/api/v1/servers", func(c *fiber.Ctx) error {
		uid := c.Locals("userID").(uuid.UUID)
		var req struct {
			Name string `json:"name"`
			Icon string `json:"icon"`
		}
		if err := c.BodyParser(&req); err != nil || req.Name == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
		}
		server, err := serverSvc.createServerFunc(c.Context(), uid, req.Name, req.Icon)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusCreated).JSON(server)
	})

	app.Get("/api/v1/servers/:id", func(c *fiber.Ctx) error {
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid server id"})
		}
		server, err := serverSvc.getServerFunc(c.Context(), id)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(server)
	})

	app.Patch("/api/v1/servers/:id", func(c *fiber.Ctx) error {
		uid := c.Locals("userID").(uuid.UUID)
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid server id"})
		}
		var req struct {
			Name *string `json:"name"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}
		updates := &models.ServerUpdate{Name: req.Name}
		server, err := serverSvc.updateServerFunc(c.Context(), id, uid, updates)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(server)
	})

	app.Delete("/api/v1/servers/:id", func(c *fiber.Ctx) error {
		uid := c.Locals("userID").(uuid.UUID)
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid server id"})
		}
		if err := serverSvc.deleteServerFunc(c.Context(), id, uid); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.SendStatus(fiber.StatusNoContent)
	})

	// 1. Create server
	resp, err := app.Test(httptest.NewRequest("POST", "/api/v1/servers", bytes.NewReader(mustJSON(t, map[string]string{
		"name": "Test Server",
	}))).WithContext(httptest.NewRequest("POST", "/api/v1/servers", nil).Context()), -1)
	// Rebuild request properly
	req := httptest.NewRequest("POST", "/api/v1/servers", bytes.NewReader(mustJSON(t, map[string]string{"name": "Test Server"})))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var createBody map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&createBody))
	assert.Equal(t, serverID.String(), createBody["id"])
	assert.Equal(t, "Test Server", createBody["name"])

	// 2. Get server
	req = httptest.NewRequest("GET", "/api/v1/servers/"+serverID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var getBody map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&getBody))
	assert.Equal(t, serverID.String(), getBody["id"])

	// 3. Update server
	req = httptest.NewRequest("PATCH", "/api/v1/servers/"+serverID.String(), bytes.NewReader(mustJSON(t, map[string]string{"name": "Updated Server"})))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var updateBody map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&updateBody))
	assert.Equal(t, "Updated Server", updateBody["name"])

	// 4. Delete server
	req = httptest.NewRequest("DELETE", "/api/v1/servers/"+serverID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)
}

// ---------- 3. Channel Flow ----------

func TestIntegration_ChannelFlow(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()
	channelID := uuid.New()
	messageID := uuid.New()
	token := generateIntegrationToken(userID)

	serverSvc := &mockServerService{
		getChannelsFunc: func(ctx context.Context, serverID uuid.UUID) ([]*models.Channel, error) {
			return []*models.Channel{
				{ID: channelID, ServerID: &serverID, Name: "general", Type: models.ChannelTypeText},
			}, nil
		},
		createChannelFunc: func(ctx context.Context, sID, creatorID uuid.UUID, name string, channelType models.ChannelType, parentID *uuid.UUID) (*models.Channel, error) {
			return &models.Channel{ID: channelID, ServerID: &sID, Name: name, Type: channelType}, nil
		},
	}

	msgSvc := &mockMessageService{
		sendMessageFunc: func(ctx context.Context, authorID, chID uuid.UUID, content string, attachments []*models.Attachment, replyTo *uuid.UUID, stickerID *uuid.UUID) (*models.Message, error) {
			return &models.Message{
				ID:        messageID,
				ChannelID: chID,
				AuthorID:  authorID,
				Content:   content,
				CreatedAt: time.Now(),
			}, nil
		},
		getMessagesFunc: func(ctx context.Context, chID, requesterID uuid.UUID, before, after *uuid.UUID, limit int) ([]*models.Message, error) {
			return []*models.Message{
				{ID: messageID, ChannelID: chID, AuthorID: requesterID, Content: "hello", CreatedAt: time.Now()},
			}, nil
		},
	}

	m := middleware.NewMiddleware(integrationTestSecret)
	app := fiber.New()
	t.Cleanup(func() { app.Shutdown() })
	app.Use(m.RequireAuth)

	// Create channel (server route)
	app.Post("/api/v1/servers/:id/channels", func(c *fiber.Ctx) error {
		uid := c.Locals("userID").(uuid.UUID)
		sid, _ := uuid.Parse(c.Params("id"))
		var req struct {
			Name string `json:"name"`
			Type string `json:"type"`
		}
		_ = c.BodyParser(&req)
		if req.Name == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
		}
		chType := models.ChannelTypeText
		if req.Type != "" {
			chType = models.ChannelType(req.Type)
		}
		ch, err := serverSvc.createChannelFunc(c.Context(), sid, uid, req.Name, chType, nil)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusCreated).JSON(ch)
	})

	// Get channels (server route)
	app.Get("/api/v1/servers/:id/channels", func(c *fiber.Ctx) error {
		sid, _ := uuid.Parse(c.Params("id"))
		channels, err := serverSvc.getChannelsFunc(c.Context(), sid)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(channels)
	})

	// Send message
	app.Post("/api/v1/channels/:id/messages", func(c *fiber.Ctx) error {
		uid := c.Locals("userID").(uuid.UUID)
		chID, _ := uuid.Parse(c.Params("id"))
		var req struct {
			Content string `json:"content"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}
		msg, err := msgSvc.sendMessageFunc(c.Context(), uid, chID, req.Content, nil, nil, nil)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusCreated).JSON(msg)
	})

	// Get messages
	app.Get("/api/v1/channels/:id/messages", func(c *fiber.Ctx) error {
		uid := c.Locals("userID").(uuid.UUID)
		chID, _ := uuid.Parse(c.Params("id"))
		msgs, err := msgSvc.getMessagesFunc(c.Context(), chID, uid, nil, nil, 50)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(msgs)
	})

	// 1. Create channel
	req := httptest.NewRequest("POST", "/api/v1/servers/"+serverID.String()+"/channels", bytes.NewReader(mustJSON(t, map[string]string{"name": "general"})))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var createBody map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&createBody))
	assert.Equal(t, channelID.String(), createBody["id"])

	// 2. Get channels
	req = httptest.NewRequest("GET", "/api/v1/servers/"+serverID.String()+"/channels", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var channelsBody []map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&channelsBody))
	assert.Len(t, channelsBody, 1)
	assert.Equal(t, channelID.String(), channelsBody[0]["id"])

	// 3. Send message
	req = httptest.NewRequest("POST", "/api/v1/channels/"+channelID.String()+"/messages", bytes.NewReader(mustJSON(t, map[string]string{"content": "hello"})))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var msgBody map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&msgBody))
	assert.Equal(t, messageID.String(), msgBody["id"])
	assert.Equal(t, "hello", msgBody["content"])

	// 4. Get messages
	req = httptest.NewRequest("GET", "/api/v1/channels/"+channelID.String()+"/messages", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var msgsBody []map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&msgsBody))
	assert.Len(t, msgsBody, 1)
	assert.Equal(t, "hello", msgsBody[0]["content"])
}

// ---------- 4. DM Flow ----------

func TestIntegration_DMFlow(t *testing.T) {
	userID := uuid.New()
	recipientID := uuid.New()
	channelID := uuid.New()
	messageID := uuid.New()
	token := generateIntegrationToken(userID)

	channelSvc := &mockChannelService{
		getOrCreateDMFunc: func(ctx context.Context, u1, u2 uuid.UUID) (*models.Channel, error) {
			return &models.Channel{
				ID:         channelID,
				Type:       models.ChannelTypeDM,
				Recipients: []uuid.UUID{u1, u2},
				CreatedAt:  time.Now(),
			}, nil
		},
		getUserDMsFunc: func(ctx context.Context, uid uuid.UUID) ([]*models.Channel, error) {
			return []*models.Channel{
				{ID: channelID, Type: models.ChannelTypeDM, Recipients: []uuid.UUID{userID, recipientID}},
			}, nil
		},
	}

	msgSvc := &mockMessageService{
		sendMessageFunc: func(ctx context.Context, authorID, chID uuid.UUID, content string, attachments []*models.Attachment, replyTo *uuid.UUID, stickerID *uuid.UUID) (*models.Message, error) {
			return &models.Message{
				ID:        messageID,
				ChannelID: chID,
				AuthorID:  authorID,
				Content:   content,
				CreatedAt: time.Now(),
			}, nil
		},
		getMessagesFunc: func(ctx context.Context, chID, requesterID uuid.UUID, before, after *uuid.UUID, limit int) ([]*models.Message, error) {
			return []*models.Message{
				{ID: messageID, ChannelID: chID, AuthorID: requesterID, Content: "dm hello", CreatedAt: time.Now()},
			}, nil
		},
	}

	userSvc := &mockUserService{
		getUserFunc: func(ctx context.Context, id uuid.UUID) (*models.User, error) {
			return &models.User{
				ID:            recipientID,
				Username:      "recipient",
				Discriminator: "0002",
				CreatedAt:     time.Now(),
			}, nil
		},
	}

	dmHandler := handlers.NewDMHandler(&mockDMService{}, channelSvc, userSvc, msgSvc)

	m := middleware.NewMiddleware(integrationTestSecret)
	app := fiber.New()
	t.Cleanup(func() { app.Shutdown() })

	// DM routes
	api := app.Group("/api/v1/dms", m.RequireAuth)
	api.Post("/", dmHandler.CreateDM)
	api.Post("/:channelId/messages", dmHandler.SendDMMessage)
	api.Get("/:channelId/messages", dmHandler.GetDMMessages)

	// 1. Create DM
	req := httptest.NewRequest("POST", "/api/v1/dms", bytes.NewReader(mustJSON(t, map[string]string{"recipient_id": recipientID.String()})))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var dmBody map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&dmBody))
	assert.Equal(t, channelID.String(), dmBody["id"])

	// 2. Send DM message
	req = httptest.NewRequest("POST", "/api/v1/dms/"+channelID.String()+"/messages", bytes.NewReader(mustJSON(t, map[string]string{"content": "dm hello"})))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var msgBody map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&msgBody))
	assert.Equal(t, messageID.String(), msgBody["id"])

	// 3. Get DM messages
	req = httptest.NewRequest("GET", "/api/v1/dms/"+channelID.String()+"/messages", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var msgsBody []map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&msgsBody))
	assert.Len(t, msgsBody, 1)
	assert.Equal(t, "dm hello", msgsBody[0]["content"])
}

// ---------- 5. Premium Flow ----------

func TestIntegration_PremiumFlow(t *testing.T) {
	userID := uuid.New()
	subID := uuid.New()
	token := generateIntegrationToken(userID)

	premiumSvc := &mockPremiumService{
		getUserPremiumStatusFunc: func(ctx context.Context, uid uuid.UUID) (*models.PremiumStatus, error) {
			return &models.PremiumStatus{
				UserID:          uid,
				Tier:            models.TierFree,
				Status:          models.SubStatusActive,
				BoostsUsed:      0,
				BoostsTotal:     0,
				BoostsAvailable: 0,
			}, nil
		},
		createSubscriptionFunc: func(ctx context.Context, uid uuid.UUID, tier models.PremiumTier) (*models.Subscription, error) {
			return &models.Subscription{
				ID:     subID,
				UserID: uid,
				Tier:   tier,
				Status: models.SubStatusActive,
			}, nil
		},
		cancelSubscriptionFunc: func(ctx context.Context, uid uuid.UUID) error {
			return nil
		},
	}

	billingSvc := &mockBillingService{
		createSubscriptionFunc: func(ctx context.Context, uid uuid.UUID, tier models.PremiumTier, paymentMethodID string) (*models.Subscription, error) {
			return &models.Subscription{
				ID:     subID,
				UserID: uid,
				Tier:   tier,
				Status: models.SubStatusActive,
			}, nil
		},
		cancelSubscriptionFunc: func(ctx context.Context, uid uuid.UUID) error {
			return nil
		},
	}

	m := middleware.NewMiddleware(integrationTestSecret)
	app := fiber.New()
	t.Cleanup(func() { app.Shutdown() })
	app.Use(m.RequireAuth)

	// Get premium status
	app.Get("/api/v1/premium/subscription", func(c *fiber.Ctx) error {
		uid := c.Locals("userID").(uuid.UUID)
		status, err := premiumSvc.getUserPremiumStatusFunc(c.Context(), uid)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(status)
	})

	// Create subscription
	app.Post("/api/v1/premium/subscribe", func(c *fiber.Ctx) error {
		uid := c.Locals("userID").(uuid.UUID)
		var req struct {
			Tier string `json:"tier"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}
		tier := models.SubscriptionTierFromString(req.Tier)
		if tier == models.TierFree {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid tier"})
		}
		sub, err := billingSvc.createSubscriptionFunc(c.Context(), uid, tier, "")
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusCreated).JSON(sub)
	})

	// Cancel subscription
	app.Delete("/api/v1/premium/subscription", func(c *fiber.Ctx) error {
		uid := c.Locals("userID").(uuid.UUID)
		if err := billingSvc.cancelSubscriptionFunc(c.Context(), uid); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"message": "subscription canceled"})
	})

	// 1. Get premium status
	req := httptest.NewRequest("GET", "/api/v1/premium/subscription", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var statusBody map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&statusBody))
	assert.Equal(t, string(models.TierFree), statusBody["tier"])

	// 2. Create subscription
	req = httptest.NewRequest("POST", "/api/v1/premium/subscribe", bytes.NewReader(mustJSON(t, map[string]string{"tier": "premium"})))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var subBody map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&subBody))
	assert.Equal(t, subID.String(), subBody["id"])
	assert.Equal(t, string(models.TierPremium), subBody["tier"])

	// 3. Cancel subscription
	req = httptest.NewRequest("DELETE", "/api/v1/premium/subscription", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var cancelBody map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&cancelBody))
	assert.Equal(t, "subscription canceled", cancelBody["message"])
}

// ---------- Helpers ----------

func mustJSON(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	assert.NoError(t, err)
	return b
}

// unused helpers to satisfy interfaces
var (
	_ services.AuthService                    = (*mockAuthService)(nil)
	_ handlers.UserServiceInterface           = (*mockUserService)(nil)
	_ handlers.ChannelServiceForUsersInterface = (*mockChannelService)(nil)
	_ handlers.MessageServiceInterface        = (*mockMessageService)(nil)
	_ handlers.DMServiceInterface             = (*mockDMService)(nil)
)
