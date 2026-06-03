// Package matrixfederation implements Matrix Federation protocol support for Hearth.
// This file implements event query endpoints for federation.
//
// Matrix Spec References:
//   - GET /event: https://spec.matrix.org/v1.16/server-server-api/#get_matrixfederationv1eventeventid
//   - GET /event_auth: https://spec.matrix.org/v1.16/server-server-api/#get_matrixfederationv1event_authroomideventid
//   - POST /get_missing_events: https://spec.matrix.org/v1.16/server-server-api/#post_matrixfederationv1get_missing_eventsroomid
package matrixfederation

import (
	"context"

	"github.com/gofiber/fiber/v2"
)

// FederationEventHandler handles event query endpoints.
type FederationEventHandler struct {
	eventStore FederationEventStore
	stateStore StateStore
	serverName string
}

// NewFederationEventHandler creates a new federation event handler.
func NewFederationEventHandler(serverName string, store FederationEventStore, state StateStore) *FederationEventHandler {
	return &FederationEventHandler{
		eventStore: store,
		stateStore: state,
		serverName: serverName,
	}
}

// eventToMap converts an Event to a generic map for JSON responses.
func eventToMap(e *Event) map[string]interface{} {
	m := map[string]interface{}{
		"event_id":         e.EventID,
		"room_id":          e.RoomID,
		"sender":           e.Sender,
		"type":             e.Type,
		"content":          e.Content,
		"prev_events":      e.PrevEvents,
		"auth_events":      e.AuthEvents,
		"depth":            e.Depth,
		"origin":           e.Origin,
		"origin_server_ts": e.OriginServerTS,
		"hashes":           map[string]interface{}{"sha256": e.Hashes.SHA256},
	}
	if e.StateKey != nil {
		m["state_key"] = *e.StateKey
	}
	if e.Signatures != nil {
		m["signatures"] = e.Signatures
	}
	if e.Unsigned != nil {
		m["unsigned"] = e.Unsigned
	}
	return m
}

// GetEvent handles GET /_matrix/federation/v1/event/{eventId}
// Returns the full event PDU or 404.
func (h *FederationEventHandler) GetEvent(c *fiber.Ctx) error {
	eventID := c.Params("eventId")
	if eventID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errcode": "M_UNKNOWN",
			"error":   "missing eventId",
		})
	}

	event, err := h.eventStore.GetEvent(context.Background(), eventID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"errcode": "M_NOT_FOUND",
			"error":   "event not found",
		})
	}

	return c.JSON(fiber.Map{
		"origin":           h.serverName,
		"origin_server_ts": event.OriginServerTS,
		"pdus":             []map[string]interface{}{eventToMap(event)},
	})
}

// GetEventAuth handles GET /_matrix/federation/v1/event_auth/{roomId}/{eventId}
// Returns the full auth chain for the event.
func (h *FederationEventHandler) GetEventAuth(c *fiber.Ctx) error {
	eventID := c.Params("eventId")

	event, err := h.eventStore.GetEvent(context.Background(), eventID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"errcode": "M_NOT_FOUND",
			"error":   "event not found",
		})
	}

	authEvents, err := h.eventStore.GetAuthEvents(context.Background(), event)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"errcode": "M_UNKNOWN",
			"error":   "failed to get auth events",
		})
	}

	result := make([]map[string]interface{}, len(authEvents))
	for i, ae := range authEvents {
		result[i] = eventToMap(ae)
	}

	return c.JSON(fiber.Map{"auth_chain": result})
}

// GetMissingEventsRequest is the body for POST /get_missing_events.
type GetMissingEventsRequest struct {
	EarliestEvents []string `json:"earliest_events"`
	LatestEvents   []string `json:"latest_events"`
	Limit          int      `json:"limit"`
	MinDepth       int64    `json:"min_depth"`
}

// GetMissingEvents handles POST /_matrix/federation/v1/get_missing_events/{roomId}
// Returns events between earliest_events and latest_events for the requesting server.
func (h *FederationEventHandler) GetMissingEvents(c *fiber.Ctx) error {
	roomID := c.Params("roomId")

	var req GetMissingEventsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errcode": "M_NOT_JSON",
			"error":   "invalid JSON",
		})
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	// Walk backwards from latest_events, stopping at earliest_events or limit.
	earliestSet := make(map[string]bool)
	for _, id := range req.EarliestEvents {
		earliestSet[id] = true
	}

	events := []*Event{}
	seen := make(map[string]bool)
	queue := append([]string{}, req.LatestEvents...)

	for len(events) < limit && len(queue) > 0 {
		currentID := queue[0]
		queue = queue[1:]

		if seen[currentID] || earliestSet[currentID] {
			continue
		}
		seen[currentID] = true

		event, err := h.eventStore.GetEvent(context.Background(), currentID)
		if err != nil {
			continue
		}

		// Only include events for the requested room
		if event.RoomID != roomID {
			continue
		}

		// Apply min_depth filter
		if req.MinDepth > 0 && event.Depth < req.MinDepth {
			continue
		}

		events = append(events, event)

		// Walk backwards via prev_events
		for _, prevID := range event.PrevEvents {
			if !seen[prevID] {
				queue = append(queue, prevID)
			}
		}
	}

	pdus := make([]map[string]interface{}, len(events))
	for i, e := range events {
		pdus[i] = eventToMap(e)
	}

	return c.JSON(fiber.Map{"events": pdus})
}

// SetupFederationEventRoutes mounts the federation event routes.
func SetupFederationEventRoutes(app *fiber.App, handler *FederationEventHandler) {
	app.Get("/_matrix/federation/v1/event/:eventId", handler.GetEvent)
	app.Get("/_matrix/federation/v1/event_auth/:roomId/:eventId", handler.GetEventAuth)
	app.Post("/_matrix/federation/v1/get_missing_events/:roomId", handler.GetMissingEvents)
}
