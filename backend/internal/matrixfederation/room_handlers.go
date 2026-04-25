package matrixfederation

import (
	"net/url"

	"github.com/gofiber/fiber/v2"

	"hearth/internal/matrix"
)

// RoomDirectoryResponse represents the response for resolving a room alias.
// Maps to Matrix Client-Server API § 10.5.2.
type RoomDirectoryResponse struct {
	// RoomID is the canonical room ID for the alias.
	RoomID string `json:"room_id"`

	// Servers is a list of servers that can be used as federation destinations.
	Servers []string `json:"servers"`
}

// RoomDirectoryHandler handles Matrix room directory (alias resolution) endpoints.
// Implements the Client-Server API § 10.5 room directory endpoints.
type RoomDirectoryHandler struct {
	store         RoomAliasStore
	homeserverCfg *matrix.HomeserverConfig
}

// NewRoomDirectoryHandler creates a new room directory handler.
func NewRoomDirectoryHandler(store RoomAliasStore, homeserverCfg *matrix.HomeserverConfig) *RoomDirectoryHandler {
	return &RoomDirectoryHandler{
		store:         store,
		homeserverCfg: homeserverCfg,
	}
}

// ResolveAlias handles GET /_matrix/client/v3/directory/room/{roomAlias}.
//
// Matrix spec § 10.5.2: Get the room ID for an alias.
// Returns the room ID and a list of servers that can be used as federation destinations.
func (h *RoomDirectoryHandler) ResolveAlias(c *fiber.Ctx) error {
	roomAlias := c.Params("roomAlias")
	if roomAlias == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errcode": "M_INVALID_PARAM",
			"error":   "roomAlias is required",
		})
	}

	// URL-decode the alias.
	decodedAlias, err := url.PathUnescape(roomAlias)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errcode": "M_INVALID_PARAM",
			"error":   "Invalid room alias encoding",
		})
	}

	alias, err := ParseAlias(decodedAlias)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errcode": "M_INVALID_ROOM_ALIAS",
			"error":   "Invalid room alias format",
		})
	}

	// Check if this is our server's alias.
	if !h.homeserverCfg.IsLocalServer(alias.ServerName) {
		// Remote alias - we could query the remote server.
		// For now, return not found (Phase 1/2 only handles local).
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"errcode": "M_NOT_FOUND",
			"error":   "Remote room aliases not yet supported",
		})
	}

	ctx := c.Context()

	roomID, _, err := h.store.GetByAlias(ctx, alias)
	if err != nil {
		if err == ErrAliasNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"errcode": "M_NOT_FOUND",
				"error":   "Room alias not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"errcode": "M_UNKNOWN",
			"error":   "Failed to resolve alias",
		})
	}

	// Build the response.
	// Include our server and any other known servers in the list.
	servers := []string{h.homeserverCfg.ServerName}

	// If the room is on a different server, include that too.
	if roomID.ServerName != h.homeserverCfg.ServerName {
		servers = append(servers, roomID.ServerName)
	}

	resp := RoomDirectoryResponse{
		RoomID:  roomID.String(),
		Servers: servers,
	}

	return c.JSON(resp)
}

// GetRoomIDAlias handles GET /_matrix/federation/v1/query/directory.
//
// Federation API: Query room alias from a remote server.
// Returns the room ID for a given alias on this server.
func (h *RoomDirectoryHandler) GetRoomIDAlias(c *fiber.Ctx) error {
	roomAlias := c.Query("room_alias")
	if roomAlias == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errcode": "M_INVALID_PARAM",
			"error":   "room_alias query parameter is required",
		})
	}

	alias, err := ParseAlias(roomAlias)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errcode": "M_INVALID_ROOM_ALIAS",
			"error":   "Invalid room alias format",
		})
	}

	ctx := c.Context()

	roomID, _, err := h.store.GetByAlias(ctx, alias)
	if err != nil {
		if err == ErrAliasNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"errcode": "M_NOT_FOUND",
				"error":   "Room alias not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"errcode": "M_UNKNOWN",
			"error":   "Failed to resolve alias",
		})
	}

	resp := RoomDirectoryResponse{
		RoomID:  roomID.String(),
		Servers: []string{h.homeserverCfg.ServerName},
	}

	return c.JSON(resp)
}

// ListRoomAliases handles GET /_matrix/client/v3/rooms/{roomId}/aliases.
//
// Matrix spec § 10.5.3: List aliases for a room.
func (h *RoomDirectoryHandler) ListRoomAliases(c *fiber.Ctx) error {
	roomIDStr := c.Params("roomId")
	if roomIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errcode": "M_INVALID_PARAM",
			"error":   "roomId is required",
		})
	}

	roomID, err := ParseRoomID(roomIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errcode": "M_INVALID_ROOM_ID",
			"error":   "Invalid room ID format",
		})
	}

	ctx := c.Context()

	aliases, err := h.store.ListAliases(ctx, roomID)
	if err != nil {
		if err == ErrRoomNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"errcode": "M_NOT_FOUND",
				"error":   "Room not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"errcode": "M_UNKNOWN",
			"error":   "Failed to list aliases",
		})
	}

	// Convert aliases to strings.
	aliasStrings := make([]string, len(aliases))
	for i, alias := range aliases {
		aliasStrings[i] = alias.String()
	}

	return c.JSON(fiber.Map{
		"aliases": aliasStrings,
	})
}

// CreateRoomAliasRequest represents the body for PUT /_matrix/client/v3/directory/room/{roomAlias}.
type CreateRoomAliasRequest struct {
	RoomID string `json:"room_id"`
}

// CreateAlias handles PUT /_matrix/client/v3/directory/room/{roomAlias}.
//
// Matrix spec § 10.5.1: Create a new room alias.
func (h *RoomDirectoryHandler) CreateAlias(c *fiber.Ctx) error {
	roomAlias := c.Params("roomAlias")
	if roomAlias == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errcode": "M_INVALID_PARAM",
			"error":   "roomAlias is required",
		})
	}

	// URL-decode the alias.
	decodedAlias, err := url.PathUnescape(roomAlias)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errcode": "M_INVALID_PARAM",
			"error":   "Invalid room alias encoding",
		})
	}

	alias, err := ParseAlias(decodedAlias)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errcode": "M_INVALID_ROOM_ALIAS",
			"error":   "Invalid room alias format",
		})
	}

	// Check if this is our server's alias.
	if !h.homeserverCfg.IsLocalServer(alias.ServerName) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"errcode": "M_INVALID_ROOM_ALIAS",
			"error":   "Cannot create alias on remote server",
		})
	}

	var req CreateRoomAliasRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errcode": "M_BAD_JSON",
			"error":   "Invalid JSON body",
		})
	}

	roomID, err := ParseRoomID(req.RoomID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errcode": "M_INVALID_ROOM_ID",
			"error":   "Invalid room ID format",
		})
	}

	ctx := c.Context()

	// Check if alias already exists.
	_, _, err = h.store.GetByAlias(ctx, alias)
	if err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"errcode": "M_EXCLUSIVE",
			"error":   "Room alias already exists",
		})
	}
	if err != ErrAliasNotFound {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"errcode": "M_UNKNOWN",
			"error":   "Failed to check alias",
		})
	}

	// Add alias to the room.
	err = h.store.AddAlias(ctx, roomID, alias)
	if err != nil {
		if err == ErrRoomNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"errcode": "M_NOT_FOUND",
				"error":   "Room not found",
			})
		}
		if err == ErrAliasInUse {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"errcode": "M_EXCLUSIVE",
				"error":   "Alias already in use",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"errcode": "M_UNKNOWN",
			"error":   "Failed to create alias",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"room_id": roomID.String(),
	})
}

// DeleteAlias handles DELETE /_matrix/client/v3/directory/room/{roomAlias}.
//
// Matrix spec § 10.5.4: Delete a room alias.
func (h *RoomDirectoryHandler) DeleteAlias(c *fiber.Ctx) error {
	roomAlias := c.Params("roomAlias")
	if roomAlias == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errcode": "M_INVALID_PARAM",
			"error":   "roomAlias is required",
		})
	}

	// URL-decode the alias.
	decodedAlias, err := url.PathUnescape(roomAlias)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errcode": "M_INVALID_PARAM",
			"error":   "Invalid room alias encoding",
		})
	}

	alias, err := ParseAlias(decodedAlias)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errcode": "M_INVALID_ROOM_ALIAS",
			"error":   "Invalid room alias format",
		})
	}

	ctx := c.Context()

	// Check if alias exists.
	_, _, err = h.store.GetByAlias(ctx, alias)
	if err != nil {
		if err == ErrAliasNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"errcode": "M_NOT_FOUND",
				"error":   "Room alias not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"errcode": "M_UNKNOWN",
			"error":   "Failed to resolve alias",
		})
	}

	// Remove the alias.
	err = h.store.RemoveAlias(ctx, alias)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"errcode": "M_UNKNOWN",
			"error":   "Failed to delete alias",
		})
	}

	return c.JSON(fiber.Map{
		"alias": alias.String(),
	})
}

// SetupDirectoryRoutes registers room directory API routes.
func SetupDirectoryRoutes(app *fiber.App, handler *RoomDirectoryHandler, clientPrefix string, federationPrefix string) {
	// Client-Server API routes.
	if clientPrefix != "" {
		client := app.Group(clientPrefix)
		client.Get("/directory/room/:roomAlias", handler.ResolveAlias)
		client.Put("/directory/room/:roomAlias", handler.CreateAlias)
		client.Delete("/directory/room/:roomAlias", handler.DeleteAlias)
		client.Get("/rooms/:roomId/aliases", handler.ListRoomAliases)
	}

	// Federation API routes.
	if federationPrefix != "" {
		fed := app.Group(federationPrefix)
		fed.Get("/query/directory", handler.GetRoomIDAlias)
	}
}

// RoomInfo represents basic room information for federation.
type RoomInfo struct {
	// RoomID is the canonical room ID.
	RoomID string `json:"room_id"`
	// Name is the human-readable room name.
	Name string `json:"name,omitempty"`
	// Topic is the room topic.
	Topic string `json:"topic,omitempty"`
	// Alias is the primary alias (if any).
	Alias string `json:"alias,omitempty"`
	// NumJoinedMembers is the number of joined members.
	NumJoinedMembers int `json:"num_joined_members"`
	// WorldReadable indicates if the room history is world-readable.
	WorldReadable bool `json:"world_readable"`
	// GuestCanJoin indicates if guests can join.
	GuestCanJoin bool `json:"guest_can_join"`
}

// PublicRoomsResponse represents the response for GET /_matrix/client/v3/publicRooms.
type PublicRoomsResponse struct {
	// Chunk is the list of public rooms.
	Chunk []RoomInfo `json:"chunk"`
	// NextBatch is the pagination token for the next page.
	NextBatch string `json:"next_batch,omitempty"`
	// PrevBatch is the pagination token for the previous page.
	PrevBatch string `json:"prev_batch,omitempty"`
	// TotalRoomCountEstimate is an estimate of the total number of public rooms.
	TotalRoomCountEstimate int `json:"total_room_count_estimate,omitempty"`
}

// PublicRoomsRequest represents the body for POST /_matrix/client/v3/publicRooms.
type PublicRoomsRequest struct {
	// Server is the server to query (optional, for federation).
	Server string `json:"server,omitempty"`
	// Limit is the maximum number of rooms to return.
	Limit int `json:"limit,omitempty"`
	// Since is the pagination token.
	Since string `json:"since,omitempty"`
	// Filter is a filter string to apply.
	Filter *PublicRoomsFilter `json:"filter,omitempty"`
	// IncludeAllNetworks indicates whether to include all networks.
	IncludeAllNetworks bool `json:"include_all_networks,omitempty"`
}

// PublicRoomsFilter represents the filter for public rooms.
type PublicRoomsFilter struct {
	// GenericSearchTerm is a string to search for in room names/topics.
	GenericSearchTerm string `json:"generic_search_term,omitempty"`
}

// PublicRoomsHandler handles public rooms listing endpoints.
type PublicRoomsHandler struct {
	store         RoomAliasStore
	homeserverCfg *matrix.HomeserverConfig
	// roomService is used to fetch room details (optional, can be nil for basic implementation).
	roomService interface{}
}

// NewPublicRoomsHandler creates a new public rooms handler.
func NewPublicRoomsHandler(store RoomAliasStore, homeserverCfg *matrix.HomeserverConfig) *PublicRoomsHandler {
	return &PublicRoomsHandler{
		store:         store,
		homeserverCfg: homeserverCfg,
	}
}

// GetPublicRooms handles GET /_matrix/client/v3/publicRooms.
//
// Matrix spec § 10.5.5: List public rooms.
func (h *PublicRoomsHandler) GetPublicRooms(c *fiber.Ctx) error {
	// Parse query parameters.
	limit := c.QueryInt("limit", 50)
	if limit < 1 {
		limit = 1
	}
	if limit > 100 {
		limit = 100
	}

	// For now, return empty list (Phase 1/2).
	// Full implementation would query all public rooms.
	resp := PublicRoomsResponse{
		Chunk:                  []RoomInfo{},
		TotalRoomCountEstimate: 0,
	}

	return c.JSON(resp)
}

// PostPublicRooms handles POST /_matrix/client/v3/publicRooms.
//
// Matrix spec § 10.5.5: List public rooms with filter.
func (h *PublicRoomsHandler) PostPublicRooms(c *fiber.Ctx) error {
	var req PublicRoomsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errcode": "M_BAD_JSON",
			"error":   "Invalid JSON body",
		})
	}

	// For now, return empty list (Phase 1/2).
	resp := PublicRoomsResponse{
		Chunk:                  []RoomInfo{},
		TotalRoomCountEstimate: 0,
	}

	return c.JSON(resp)
}

// SetupPublicRoomsRoutes registers public rooms API routes.
func SetupPublicRoomsRoutes(app *fiber.App, handler *PublicRoomsHandler, prefix string) {
	if prefix != "" {
		client := app.Group(prefix)
		client.Get("/publicRooms", handler.GetPublicRooms)
		client.Post("/publicRooms", handler.PostPublicRooms)
	}
}
