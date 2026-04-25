// Package matrixfederation implements Matrix Federation protocol support for Hearth.
// This file implements the room join handshake (make_join / send_join).
//
// Matrix Spec References:
//   - GET /make_join: https://spec.matrix.org/v1.16/server-server-api/#get_matrixfederationv1make_joinroomiduserid
//   - PUT /send_join: https://spec.matrix.org/v1.16/server-server-api/#put_matrixfederationv2send_joinroomideventid
package matrixfederation

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
)

// JoinHandler implements the make_join / send_join handshake.
type JoinHandler struct {
	stateStore     *InMemoryStateStore
	eventStore     FederationEventStore
	roomAliasStore RoomAliasStore
	authChecker    *AuthChecker
	serverName     string
}

// NewJoinHandler creates a new JoinHandler.
func NewJoinHandler(
	serverName string,
	state *InMemoryStateStore,
	store FederationEventStore,
	roomAlias RoomAliasStore,
	auth *AuthChecker,
) *JoinHandler {
	return &JoinHandler{
		stateStore:     state,
		eventStore:     store,
		roomAliasStore: roomAlias,
		authChecker:    auth,
		serverName:     serverName,
	}
}

// MakeJoin handles GET /_matrix/federation/v1/make_join/{roomId}/{userId}.
// Returns a placeholder join event for the requesting server to complete and sign.
func (h *JoinHandler) MakeJoin(c *fiber.Ctx) error {
	roomIDStr := c.Params("roomId")
	userID := c.Params("userId")

	roomID, err := ParseRoomID(roomIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errcode": "M_UNKNOWN",
			"error":   "invalid room ID",
		})
	}

	if _, _, err := h.roomAliasStore.GetByRoomID(context.Background(), roomID); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"errcode": "M_NOT_FOUND",
			"error":   "room not found",
		})
	}

	rs, err := h.stateStore.GetRoomState(roomID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"errcode": "M_NOT_FOUND",
			"error":   "room state not found",
		})
	}

	stateEvents := stateEventsFromRoom(rs)

	// Build auth events list (create, power_levels, join_rules)
	authEventIDs := []string{}
	for _, e := range stateEvents {
		switch e.Type {
		case EventTypeCreate, EventTypePowerLevels, EventTypeJoinRules:
			authEventIDs = append(authEventIDs, e.EventID)
		}
	}

	prevEvents := rs.GetForwardExtremities()
	depth := rs.GetCurrentDepth() + 1

	// Build placeholder join event for the requesting server to sign
	placeholderEvent := map[string]interface{}{
		"type":             "m.room.member",
		"room_id":          roomIDStr,
		"sender":           userID,
		"state_key":        userID,
		"content":          map[string]interface{}{"membership": "join"},
		"prev_events":      prevEvents,
		"auth_events":      authEventIDs,
		"depth":            depth,
		"origin":           h.serverName,
		"origin_server_ts": time.Now().UnixMilli(),
	}

	return c.JSON(fiber.Map{
		"event":         placeholderEvent,
		"room_version":  "10",
	})
}

// SendJoinRequest is the body for PUT /send_join.
type SendJoinRequest struct {
	Event map[string]interface{} `json:"event"`
}

// SendJoin handles PUT /_matrix/federation/v2/send_join/{roomId}/{eventId}.
// Receives the requesting server's signed join event, validates and stores it,
// and returns the room state.
func (h *JoinHandler) SendJoin(c *fiber.Ctx) error {
	roomIDStr := c.Params("roomId")
	eventIDParam := c.Params("eventId")
	_ = eventIDParam

	roomID, err := ParseRoomID(roomIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errcode": "M_UNKNOWN",
			"error":   "invalid room ID",
		})
	}

	var body SendJoinRequest
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errcode": "M_NOT_JSON",
			"error":   "invalid JSON",
		})
	}

	em := body.Event
	event := &Event{
		EventID:        getString(em, "event_id"),
		RoomID:         getString(em, "room_id"),
		Sender:         getString(em, "sender"),
		Type:           getString(em, "type"),
		Content:        getMap(em, "content"),
		PrevEvents:     getStringSlice(em, "prev_events"),
		AuthEvents:     getStringSlice(em, "auth_events"),
		Depth:          getInt(em, "depth"),
		Origin:         getString(em, "origin"),
		OriginServerTS: getInt(em, "origin_server_ts"),
	}
	if sk, ok := em["state_key"].(string); ok {
		event.StateKey = &sk
	}
	if hashes, ok := em["hashes"].(map[string]interface{}); ok {
		if sha, ok := hashes["sha256"].(string); ok {
			event.Hashes.SHA256 = sha
		}
	}

	if err := event.Validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errcode": "M_INVALID",
			"error":   fmt.Sprintf("invalid event: %v", err),
		})
	}

	membership, _ := event.Content["membership"].(string)
	if event.Type != EventTypeMember || membership != "join" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errcode": "M_INVALID",
			"error":   "not a join event",
		})
	}

	if err := h.eventStore.StoreEvent(context.Background(), event); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"errcode": "M_UNKNOWN",
			"error":   fmt.Sprintf("failed to store event: %v", err),
		})
	}

	rs := h.stateStore.GetOrCreateRoomState(roomID)
	if err := rs.AddEvent(event); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"errcode": "M_UNKNOWN",
			"error":   fmt.Sprintf("failed to update room state: %v", err),
		})
	}

	stateEvents := stateEventsFromRoom(rs)
	statePDUs := make([]map[string]interface{}, len(stateEvents))
	for i, e := range stateEvents {
		statePDUs[i] = eventToMap(e)
	}

	return c.JSON(fiber.Map{
		"event":      em,
		"state":      statePDUs,
		"auth_chain": []map[string]interface{}{},
		"origin":     h.serverName,
	})
}

// SetupJoinRoutes mounts the join handshake routes.
func SetupJoinRoutes(app *fiber.App, handler *JoinHandler) {
	app.Get("/_matrix/federation/v1/make_join/:roomId/:userId", handler.MakeJoin)
	app.Put("/_matrix/federation/v2/send_join/:roomId/:eventId", handler.SendJoin)
}

// ----- helpers -----

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int64 {
	switch v := m[key].(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	}
	return 0
}

func getMap(m map[string]interface{}, key string) map[string]interface{} {
	if v, ok := m[key].(map[string]interface{}); ok {
		return v
	}
	return nil
}

func getStringSlice(m map[string]interface{}, key string) []string {
	if v, ok := m[key].([]interface{}); ok {
		out := make([]string, len(v))
		for i, e := range v {
			if s, ok := e.(string); ok {
				out[i] = s
			}
		}
		return out
	}
	return nil
}
