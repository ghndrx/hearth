package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/matrix"
	"hearth/internal/matrixfederation"
	"hearth/internal/models"
	"hearth/internal/services"
)

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

type mockFedChannelRepo struct {
	getByIDFunc func(ctx context.Context, id uuid.UUID) (*models.Channel, error)
}

func (m *mockFedChannelRepo) Create(ctx context.Context, channel *models.Channel) error { return nil }
func (m *mockFedChannelRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, nil
}
func (m *mockFedChannelRepo) Update(ctx context.Context, channel *models.Channel) error { return nil }
func (m *mockFedChannelRepo) Delete(ctx context.Context, id uuid.UUID) error            { return nil }
func (m *mockFedChannelRepo) GetByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.Channel, error) {
	return nil, nil
}
func (m *mockFedChannelRepo) GetDMChannel(ctx context.Context, user1ID, user2ID uuid.UUID) (*models.Channel, error) {
	return nil, nil
}
func (m *mockFedChannelRepo) GetUserDMs(ctx context.Context, userID uuid.UUID) ([]*models.Channel, error) {
	return nil, nil
}
func (m *mockFedChannelRepo) UpdateLastMessage(ctx context.Context, channelID, messageID uuid.UUID, at time.Time) error {
	return nil
}
func (m *mockFedChannelRepo) UpdateForumConfig(ctx context.Context, channelID uuid.UUID, configJSON []byte) error {
	return nil
}
func (m *mockFedChannelRepo) AddRecipient(ctx context.Context, channelID, userID uuid.UUID) error {
	return nil
}
func (m *mockFedChannelRepo) RemoveRecipient(ctx context.Context, channelID, userID uuid.UUID) error {
	return nil
}
func (m *mockFedChannelRepo) CountRecipients(ctx context.Context, channelID uuid.UUID) (int, error) {
	return 0, nil
}
func (m *mockFedChannelRepo) BulkUpdatePositions(ctx context.Context, entries []models.ReorderChannelEntry) error {
	return nil
}
func (m *mockFedChannelRepo) GetPermissionOverrides(ctx context.Context, channelID uuid.UUID) ([]models.PermissionOverride, error) {
	return nil, nil
}
func (m *mockFedChannelRepo) UpsertPermissionOverride(ctx context.Context, override *models.PermissionOverride) error {
	return nil
}
func (m *mockFedChannelRepo) DeletePermissionOverride(ctx context.Context, channelID, targetID uuid.UUID, targetType string) error {
	return nil
}

type mockFedServerRepo struct {
	getByIDFunc    func(ctx context.Context, id uuid.UUID) (*models.Server, error)
	getMemberFunc  func(ctx context.Context, serverID, userID uuid.UUID) (*models.Member, error)
	getMembersFunc func(ctx context.Context, serverID uuid.UUID, limit, offset int) ([]*models.Member, error)
}

func (m *mockFedServerRepo) Create(ctx context.Context, server *models.Server) error { return nil }
func (m *mockFedServerRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Server, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, nil
}
func (m *mockFedServerRepo) Update(ctx context.Context, server *models.Server) error { return nil }
func (m *mockFedServerRepo) Delete(ctx context.Context, id uuid.UUID) error          { return nil }
func (m *mockFedServerRepo) TransferOwnership(ctx context.Context, serverID, newOwnerID uuid.UUID) error {
	return nil
}
func (m *mockFedServerRepo) GetMembers(ctx context.Context, serverID uuid.UUID, limit, offset int) ([]*models.Member, error) {
	if m.getMembersFunc != nil {
		return m.getMembersFunc(ctx, serverID, limit, offset)
	}
	return nil, nil
}
func (m *mockFedServerRepo) GetMembersPaginated(ctx context.Context, serverID uuid.UUID, cursor *models.MemberCursor, limit int) (*models.PaginatedMembers, error) {
	return nil, nil
}
func (m *mockFedServerRepo) GetMember(ctx context.Context, serverID, userID uuid.UUID) (*models.Member, error) {
	if m.getMemberFunc != nil {
		return m.getMemberFunc(ctx, serverID, userID)
	}
	return nil, nil
}
func (m *mockFedServerRepo) GetMembersWithRole(ctx context.Context, serverID, roleID uuid.UUID) ([]*models.Member, error) {
	return nil, nil
}
func (m *mockFedServerRepo) AddMember(ctx context.Context, member *models.Member) error { return nil }
func (m *mockFedServerRepo) UpdateMember(ctx context.Context, member *models.Member) error {
	return nil
}
func (m *mockFedServerRepo) RemoveMember(ctx context.Context, serverID, userID uuid.UUID) error {
	return nil
}
func (m *mockFedServerRepo) GetMemberCount(ctx context.Context, serverID uuid.UUID) (int, error) {
	return 0, nil
}
func (m *mockFedServerRepo) GetUserServers(ctx context.Context, userID uuid.UUID) ([]*models.Server, error) {
	return nil, nil
}
func (m *mockFedServerRepo) GetOwnedServersCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return 0, nil
}
func (m *mockFedServerRepo) GetBan(ctx context.Context, serverID, userID uuid.UUID) (*models.Ban, error) {
	return nil, nil
}
func (m *mockFedServerRepo) AddBan(ctx context.Context, ban *models.Ban) error { return nil }
func (m *mockFedServerRepo) RemoveBan(ctx context.Context, serverID, userID uuid.UUID) error {
	return nil
}
func (m *mockFedServerRepo) GetBans(ctx context.Context, serverID uuid.UUID) ([]*models.Ban, error) {
	return nil, nil
}
func (m *mockFedServerRepo) CreateInvite(ctx context.Context, invite *models.Invite) error {
	return nil
}
func (m *mockFedServerRepo) GetInvite(ctx context.Context, code string) (*models.Invite, error) {
	return nil, nil
}
func (m *mockFedServerRepo) GetInviteByVanityCode(ctx context.Context, vanityCode string) (*models.Invite, error) {
	return nil, nil
}
func (m *mockFedServerRepo) GetInvites(ctx context.Context, serverID uuid.UUID) ([]*models.Invite, error) {
	return nil, nil
}
func (m *mockFedServerRepo) DeleteInvite(ctx context.Context, code string) error        { return nil }
func (m *mockFedServerRepo) IncrementInviteUses(ctx context.Context, code string) error { return nil }
func (m *mockFedServerRepo) LogInviteUse(ctx context.Context, log *models.InviteUseLog) error {
	return nil
}
func (m *mockFedServerRepo) GetInviteUseLogs(ctx context.Context, inviteCode string) ([]models.InviteUseLog, error) {
	return nil, nil
}
func (m *mockFedServerRepo) GetServerInviteUseLogs(ctx context.Context, serverID uuid.UUID) ([]models.InviteUseLog, error) {
	return nil, nil
}

type mockFedRoleRepo struct {
	getByServerIDFunc func(ctx context.Context, serverID uuid.UUID) ([]*models.Role, error)
}

func (m *mockFedRoleRepo) Create(ctx context.Context, role *models.Role) error { return nil }
func (m *mockFedRoleRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Role, error) {
	return nil, nil
}
func (m *mockFedRoleRepo) GetByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.Role, error) {
	if m.getByServerIDFunc != nil {
		return m.getByServerIDFunc(ctx, serverID)
	}
	return nil, nil
}
func (m *mockFedRoleRepo) Update(ctx context.Context, role *models.Role) error { return nil }
func (m *mockFedRoleRepo) Delete(ctx context.Context, id uuid.UUID) error      { return nil }
func (m *mockFedRoleRepo) UpdatePositions(ctx context.Context, serverID uuid.UUID, positions map[uuid.UUID]int) error {
	return nil
}
func (m *mockFedRoleRepo) AddRoleToMember(ctx context.Context, serverID, userID, roleID uuid.UUID) error {
	return nil
}
func (m *mockFedRoleRepo) RemoveRoleFromMember(ctx context.Context, serverID, userID, roleID uuid.UUID) error {
	return nil
}
func (m *mockFedRoleRepo) GetMemberRoles(ctx context.Context, serverID, userID uuid.UUID) ([]*models.Role, error) {
	return nil, nil
}
func (m *mockFedRoleRepo) GetMemberPermissions(ctx context.Context, serverID, userID uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockFedRoleRepo) GetDefaultRole(ctx context.Context, serverID uuid.UUID) (*models.Role, error) {
	return nil, nil
}

type mockFedRoomAliasStore struct {
	createMappingFunc func(ctx context.Context, roomID matrixfederation.RoomID, channelID uuid.UUID, aliases []matrixfederation.Alias) error
}

func (m *mockFedRoomAliasStore) CreateMapping(ctx context.Context, roomID matrixfederation.RoomID, channelID uuid.UUID, aliases []matrixfederation.Alias) error {
	if m.createMappingFunc != nil {
		return m.createMappingFunc(ctx, roomID, channelID, aliases)
	}
	return nil
}
func (m *mockFedRoomAliasStore) GetByRoomID(ctx context.Context, roomID matrixfederation.RoomID) (uuid.UUID, []matrixfederation.Alias, error) {
	return uuid.Nil, nil, nil
}
func (m *mockFedRoomAliasStore) GetByAlias(ctx context.Context, alias matrixfederation.Alias) (matrixfederation.RoomID, uuid.UUID, error) {
	return matrixfederation.RoomID{}, uuid.Nil, nil
}
func (m *mockFedRoomAliasStore) GetByChannelID(ctx context.Context, channelID uuid.UUID) (matrixfederation.RoomID, []matrixfederation.Alias, error) {
	return matrixfederation.RoomID{}, nil, nil
}
func (m *mockFedRoomAliasStore) AddAlias(ctx context.Context, roomID matrixfederation.RoomID, alias matrixfederation.Alias) error {
	return nil
}
func (m *mockFedRoomAliasStore) RemoveAlias(ctx context.Context, alias matrixfederation.Alias) error {
	return nil
}
func (m *mockFedRoomAliasStore) RemoveMapping(ctx context.Context, roomID matrixfederation.RoomID) error {
	return nil
}
func (m *mockFedRoomAliasStore) ListAliases(ctx context.Context, roomID matrixfederation.RoomID) ([]matrixfederation.Alias, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Test setup
// ---------------------------------------------------------------------------

func setupChannelFederationTestApp(
	t *testing.T,
	channelRepo *mockFedChannelRepo,
	serverRepo *mockFedServerRepo,
	roleRepo *mockFedRoleRepo,
	roomStore *mockFedRoomAliasStore,
) *fiber.App {
	t.Helper()
	app := fiber.New()
	t.Cleanup(func() { app.Shutdown() })

	app.Use(func(c *fiber.Ctx) error {
		userID := c.Get("X-User-ID")
		if userID != "" {
			id, _ := uuid.Parse(userID)
			c.Locals("userID", id)
		}
		return c.Next()
	})

	channelSvc := services.NewChannelService(channelRepo, nil, nil, nil, nil)
	serverSvc := services.NewServerService(serverRepo, nil, roleRepo, nil, nil, nil, nil, nil)

	handler := NewChannelHandler(channelSvc, nil)
	handler.SetServerService(serverSvc)
	handler.SetRoomAliasStore(roomStore)
	handler.SetHomeserverConfig(&matrix.HomeserverConfig{ServerName: "test.example.com"})

	app.Post("/channels/:id/federate", handler.Federate)

	return app
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestChannelHandler_Federate_Success(t *testing.T) {
	ownerID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	channelRepo := &mockFedChannelRepo{
		getByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
			if id == channelID {
				return &models.Channel{
					ID:       channelID,
					ServerID: &serverID,
					Name:     "general",
					Type:     models.ChannelTypeText,
				}, nil
			}
			return nil, services.ErrChannelNotFound
		},
	}

	serverRepo := &mockFedServerRepo{
		getByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.Server, error) {
			if id == serverID {
				return &models.Server{
					ID:      serverID,
					Name:    "Test Server",
					OwnerID: ownerID,
				}, nil
			}
			return nil, nil
		},
		getMembersFunc: func(ctx context.Context, sID uuid.UUID, limit, offset int) ([]*models.Member, error) {
			return []*models.Member{}, nil
		},
	}

	roleRepo := &mockFedRoleRepo{}
	roomStore := &mockFedRoomAliasStore{}

	app := setupChannelFederationTestApp(t, channelRepo, serverRepo, roleRepo, roomStore)

	req := httptest.NewRequest("POST", "/channels/"+channelID.String()+"/federate", nil)
	req.Header.Set("X-User-ID", ownerID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	expectedRoomID := matrixfederation.GenerateRoomID(channelID, "test.example.com").String()
	expectedAlias := matrixfederation.Alias{
		Localpart:  "hearth-" + channelID.String(),
		ServerName: "test.example.com",
	}.String()

	if result["room_id"] != expectedRoomID {
		t.Errorf("Expected room_id %q, got %v", expectedRoomID, result["room_id"])
	}
	if result["alias"] != expectedAlias {
		t.Errorf("Expected alias %q, got %v", expectedAlias, result["alias"])
	}
}

func TestChannelHandler_Federate_PermissionDenied(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()
	memberRoleID := uuid.New()

	channelRepo := &mockFedChannelRepo{
		getByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
			if id == channelID {
				return &models.Channel{
					ID:       channelID,
					ServerID: &serverID,
					Name:     "general",
					Type:     models.ChannelTypeText,
				}, nil
			}
			return nil, services.ErrChannelNotFound
		},
	}

	serverRepo := &mockFedServerRepo{
		getByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.Server, error) {
			if id == serverID {
				return &models.Server{
					ID:      serverID,
					Name:    "Test Server",
					OwnerID: uuid.New(), // different owner
				}, nil
			}
			return nil, nil
		},
		getMemberFunc: func(ctx context.Context, sID, uID uuid.UUID) (*models.Member, error) {
			return &models.Member{
				UserID:   uID,
				ServerID: sID,
				Roles:    []uuid.UUID{memberRoleID},
			}, nil
		},
	}

	roleRepo := &mockFedRoleRepo{
		getByServerIDFunc: func(ctx context.Context, sID uuid.UUID) ([]*models.Role, error) {
			return []*models.Role{
				{
					ID:          memberRoleID,
					ServerID:    sID,
					Name:        "Member",
					Permissions: 0, // no admin
				},
			}, nil
		},
	}

	roomStore := &mockFedRoomAliasStore{}

	app := setupChannelFederationTestApp(t, channelRepo, serverRepo, roleRepo, roomStore)

	req := httptest.NewRequest("POST", "/channels/"+channelID.String()+"/federate", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusForbidden {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 403, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestChannelHandler_Federate_ChannelNotFound(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	channelRepo := &mockFedChannelRepo{
		getByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
			return nil, services.ErrChannelNotFound
		},
	}

	serverRepo := &mockFedServerRepo{}
	roleRepo := &mockFedRoleRepo{}
	roomStore := &mockFedRoomAliasStore{}

	app := setupChannelFederationTestApp(t, channelRepo, serverRepo, roleRepo, roomStore)

	req := httptest.NewRequest("POST", "/channels/"+channelID.String()+"/federate", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 404, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestChannelHandler_Federate_DMChannelRejected(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	channelRepo := &mockFedChannelRepo{
		getByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
			if id == channelID {
				return &models.Channel{
					ID:   channelID,
					Name: "dm-channel",
					Type: models.ChannelTypeDM,
					// ServerID is nil => DM
				}, nil
			}
			return nil, services.ErrChannelNotFound
		},
	}

	serverRepo := &mockFedServerRepo{}
	roleRepo := &mockFedRoleRepo{}
	roomStore := &mockFedRoomAliasStore{}

	app := setupChannelFederationTestApp(t, channelRepo, serverRepo, roleRepo, roomStore)

	req := httptest.NewRequest("POST", "/channels/"+channelID.String()+"/federate", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 400, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestChannelHandler_Federate_ServerNotFound(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	channelRepo := &mockFedChannelRepo{
		getByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
			if id == channelID {
				return &models.Channel{
					ID:       channelID,
					ServerID: &serverID,
					Name:     "general",
					Type:     models.ChannelTypeText,
				}, nil
			}
			return nil, services.ErrChannelNotFound
		},
	}

	serverRepo := &mockFedServerRepo{
		getByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.Server, error) {
			return nil, services.ErrServerNotFound
		},
	}

	roleRepo := &mockFedRoleRepo{}
	roomStore := &mockFedRoomAliasStore{}

	app := setupChannelFederationTestApp(t, channelRepo, serverRepo, roleRepo, roomStore)

	req := httptest.NewRequest("POST", "/channels/"+channelID.String()+"/federate", nil)
	req.Header.Set("X-User-ID", userID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 404, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestChannelHandler_Federate_AlreadyFederated(t *testing.T) {
	ownerID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	channelRepo := &mockFedChannelRepo{
		getByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
			if id == channelID {
				return &models.Channel{
					ID:       channelID,
					ServerID: &serverID,
					Name:     "general",
					Type:     models.ChannelTypeText,
				}, nil
			}
			return nil, services.ErrChannelNotFound
		},
	}

	serverRepo := &mockFedServerRepo{
		getByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.Server, error) {
			if id == serverID {
				return &models.Server{
					ID:      serverID,
					Name:    "Test Server",
					OwnerID: ownerID,
				}, nil
			}
			return nil, nil
		},
		getMembersFunc: func(ctx context.Context, sID uuid.UUID, limit, offset int) ([]*models.Member, error) {
			return []*models.Member{}, nil
		},
	}

	roleRepo := &mockFedRoleRepo{}
	roomStore := &mockFedRoomAliasStore{
		createMappingFunc: func(ctx context.Context, roomID matrixfederation.RoomID, chID uuid.UUID, aliases []matrixfederation.Alias) error {
			return matrixfederation.ErrAliasAlreadyExists
		},
	}

	app := setupChannelFederationTestApp(t, channelRepo, serverRepo, roleRepo, roomStore)

	req := httptest.NewRequest("POST", "/channels/"+channelID.String()+"/federate", nil)
	req.Header.Set("X-User-ID", ownerID.String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusConflict {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 409, got %d: %s", resp.StatusCode, string(body))
	}
}
