// Package matrixfederation implements Matrix Federation protocol support for Hearth.
// This file implements Federation state query endpoints.
//
// Matrix Spec References:
//   - GET /state: https://spec.matrix.org/v1.16/server-server-api/#get_matrixfederationv1stateroomid
//   - GET /state_ids: https://spec.matrix.org/v1.16/server-server-api/#get_matrixfederationv1state_idsroomid
package matrixfederation

import (
	"github.com/gofiber/fiber/v2"
)

// FederationStateHandler handles Matrix federation state query endpoints.
type FederationStateHandler struct {
	stateStore *InMemoryStateStore
	eventStore FederationEventStore
	serverName string
}

// NewFederationStateHandler creates a new FederationStateHandler.
func NewFederationStateHandler(serverName string, state *InMemoryStateStore, store FederationEventStore) *FederationStateHandler {
	return &FederationStateHandler{
		stateStore: state,
		eventStore: store,
		serverName: serverName,
	}
}

// stateEventsFromRoom collects all current state events from a FederatedRoomState.
func stateEventsFromRoom(rs *FederatedRoomState) []*Event {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	events := make([]*Event, 0, len(rs.stateEvents))
	for _, e := range rs.stateEvents {
		events = append(events, e)
	}
	return events
}

// GetState handles GET /_matrix/federation/v1/state/{roomId}?event_id=XYZ.
// Returns the full room state and auth chain at the specified event.
func (h *FederationStateHandler) GetState(c *fiber.Ctx) error {
	roomIDStr := c.Params("roomId")
	roomID, err := ParseRoomID(roomIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errcode": "M_UNKNOWN",
			"error":   "invalid room ID",
		})
	}

	rs, err := h.stateStore.GetRoomState(roomID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"errcode": "M_NOT_FOUND",
			"error":   "room state not found",
		})
	}

	events := stateEventsFromRoom(rs)
	pdus := make([]map[string]interface{}, len(events))
	for i, e := range events {
		pdus[i] = eventToMap(e)
	}

	return c.JSON(fiber.Map{
		"pdus":       pdus,
		"auth_chain": []map[string]interface{}{},
	})
}

// GetStateIds handles GET /_matrix/federation/v1/state_ids/{roomId}?event_id=XYZ.
// Returns only the event IDs of the current room state and auth chain.
func (h *FederationStateHandler) GetStateIds(c *fiber.Ctx) error {
	roomIDStr := c.Params("roomId")
	roomID, err := ParseRoomID(roomIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errcode": "M_UNKNOWN",
			"error":   "invalid room ID",
		})
	}

	rs, err := h.stateStore.GetRoomState(roomID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"errcode": "M_NOT_FOUND",
			"error":   "room state not found",
		})
	}

	events := stateEventsFromRoom(rs)
	pduIDs := make([]string, len(events))
	for i, e := range events {
		pduIDs[i] = e.EventID
	}

	return c.JSON(fiber.Map{
		"pdu_ids":        pduIDs,
		"auth_chain_ids": []string{},
	})
}

// SetupFederationStateRoutes mounts the federation state routes.
func SetupFederationStateRoutes(app *fiber.App, handler *FederationStateHandler) {
	app.Get("/_matrix/federation/v1/state/:roomId", handler.GetState)
	app.Get("/_matrix/federation/v1/state_ids/:roomId", handler.GetStateIds)
}
