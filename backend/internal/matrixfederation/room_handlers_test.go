package matrixfederation

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hearth/internal/matrix"
)

func setupRoomTestApp(t *testing.T) (*fiber.App, *InMemoryRoomAliasStore, *RoomDirectoryHandler) {
	app := fiber.New()
	store := NewInMemoryRoomAliasStore()
	cfg := &matrix.HomeserverConfig{
		ServerName: "hearth.example.com",
		BaseURL:    "https://hearth.example.com",
	}

	handler := NewRoomDirectoryHandler(store, cfg)
	SetupDirectoryRoutes(app, handler, "/_matrix/client/v3", "/_matrix/federation/v1")

	return app, store, handler
}

func TestRoomDirectoryHandler_ResolveAlias_Success(t *testing.T) {
	app, store, _ := setupRoomTestApp(t)
	ctx := context.Background()

	roomID := RoomID{Localpart: "abc123", ServerName: "hearth.example.com"}
	channelID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	alias := Alias{Localpart: "general", ServerName: "hearth.example.com"}

	err := store.CreateMapping(ctx, roomID, channelID, []Alias{alias})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/_matrix/client/v3/directory/room/%23general:hearth.example.com", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result RoomDirectoryResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, roomID.String(), result.RoomID)
	assert.Contains(t, result.Servers, "hearth.example.com")
}

func TestRoomDirectoryHandler_ResolveAlias_NotFound(t *testing.T) {
	app, _, _ := setupRoomTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/_matrix/client/v3/directory/room/%23nonexistent:hearth.example.com", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestRoomDirectoryHandler_ResolveAlias_RemoteServer(t *testing.T) {
	app, _, _ := setupRoomTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/_matrix/client/v3/directory/room/%23general:matrix.org", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestRoomDirectoryHandler_ResolveAlias_Invalid(t *testing.T) {
	app, _, _ := setupRoomTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/_matrix/client/v3/directory/room/invalid-alias", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestRoomDirectoryHandler_Federation_GetRoomIDAlias(t *testing.T) {
	app, store, _ := setupRoomTestApp(t)
	ctx := context.Background()

	roomID := RoomID{Localpart: "abc123", ServerName: "hearth.example.com"}
	channelID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	alias := Alias{Localpart: "general", ServerName: "hearth.example.com"}

	err := store.CreateMapping(ctx, roomID, channelID, []Alias{alias})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/_matrix/federation/v1/query/directory?room_alias=%23general:hearth.example.com", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result RoomDirectoryResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, roomID.String(), result.RoomID)
}

func TestRoomDirectoryHandler_CreateAlias(t *testing.T) {
	app, store, _ := setupRoomTestApp(t)
	ctx := context.Background()

	roomID := RoomID{Localpart: "abc123", ServerName: "hearth.example.com"}
	channelID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	// Pre-create the room mapping.
	err := store.CreateMapping(ctx, roomID, channelID, nil)
	require.NoError(t, err)

	body := CreateRoomAliasRequest{
		RoomID: roomID.String(),
	}
	bodyJSON, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/_matrix/client/v3/directory/room/%23general:hearth.example.com", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify alias was created.
	_, gotChanID, err := store.GetByAlias(ctx, Alias{Localpart: "general", ServerName: "hearth.example.com"})
	require.NoError(t, err)
	assert.Equal(t, channelID, gotChanID)
}

func TestRoomDirectoryHandler_CreateAlias_Duplicate(t *testing.T) {
	app, store, _ := setupRoomTestApp(t)
	ctx := context.Background()

	roomID := RoomID{Localpart: "abc123", ServerName: "hearth.example.com"}
	channelID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	alias := Alias{Localpart: "general", ServerName: "hearth.example.com"}

	err := store.CreateMapping(ctx, roomID, channelID, []Alias{alias})
	require.NoError(t, err)

	body := CreateRoomAliasRequest{
		RoomID: roomID.String(),
	}
	bodyJSON, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/_matrix/client/v3/directory/room/%23general:hearth.example.com", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
}

func TestRoomDirectoryHandler_DeleteAlias(t *testing.T) {
	app, store, _ := setupRoomTestApp(t)
	ctx := context.Background()

	roomID := RoomID{Localpart: "abc123", ServerName: "hearth.example.com"}
	channelID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	alias := Alias{Localpart: "general", ServerName: "hearth.example.com"}

	err := store.CreateMapping(ctx, roomID, channelID, []Alias{alias})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodDelete, "/_matrix/client/v3/directory/room/%23general:hearth.example.com", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify alias was deleted.
	_, _, err = store.GetByAlias(ctx, alias)
	assert.ErrorIs(t, err, ErrAliasNotFound)
}

func TestRoomDirectoryHandler_ListRoomAliases(t *testing.T) {
	app, store, _ := setupRoomTestApp(t)
	ctx := context.Background()

	roomID := RoomID{Localpart: "abc123", ServerName: "hearth.example.com"}
	channelID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	aliases := []Alias{
		{Localpart: "general", ServerName: "hearth.example.com"},
		{Localpart: "main", ServerName: "hearth.example.com"},
	}

	err := store.CreateMapping(ctx, roomID, channelID, aliases)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/_matrix/client/v3/rooms/!abc123:hearth.example.com/aliases", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Aliases []string `json:"aliases"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	assert.Len(t, result.Aliases, 2)
	assert.Contains(t, result.Aliases, "#general:hearth.example.com")
	assert.Contains(t, result.Aliases, "#main:hearth.example.com")
}

func TestRoomDirectoryHandler_ListRoomAliases_NotFound(t *testing.T) {
	app, _, _ := setupRoomTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/_matrix/client/v3/rooms/!nonexistent:hearth.example.com/aliases", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestPublicRoomsHandler_GetPublicRooms(t *testing.T) {
	app := fiber.New()
	cfg := &matrix.HomeserverConfig{
		ServerName: "hearth.example.com",
	}
	handler := NewPublicRoomsHandler(NewInMemoryRoomAliasStore(), cfg)
	SetupPublicRoomsRoutes(app, handler, "/_matrix/client/v3")

	req := httptest.NewRequest(http.MethodGet, "/_matrix/client/v3/publicRooms", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result PublicRoomsResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	assert.Empty(t, result.Chunk)
}

func TestPublicRoomsHandler_PostPublicRooms(t *testing.T) {
	app := fiber.New()
	cfg := &matrix.HomeserverConfig{
		ServerName: "hearth.example.com",
	}
	handler := NewPublicRoomsHandler(NewInMemoryRoomAliasStore(), cfg)
	SetupPublicRoomsRoutes(app, handler, "/_matrix/client/v3")

	body := PublicRoomsRequest{
		Limit: 10,
	}
	bodyJSON, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/_matrix/client/v3/publicRooms", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result PublicRoomsResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	assert.Empty(t, result.Chunk)
}

func TestSetupDirectoryRoutes_NilPrefixes(t *testing.T) {
	app := fiber.New()
	store := NewInMemoryRoomAliasStore()
	cfg := &matrix.HomeserverConfig{
		ServerName: "hearth.example.com",
	}
	handler := NewRoomDirectoryHandler(store, cfg)

	// Should not panic with empty prefixes.
	SetupDirectoryRoutes(app, handler, "", "")
}
