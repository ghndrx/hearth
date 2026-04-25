// Package matrixfederation implements Matrix Federation protocol support for Hearth.
// This file implements the federation backfill endpoint.
//
// Matrix Spec Reference:
//   - GET /backfill: https://spec.matrix.org/v1.16/server-server-api/#get_matrixfederationv1backfillroomid
package matrixfederation

import (
	"context"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// BackfillHandler returns historical events by walking the prev_events DAG.
type BackfillHandler struct {
	eventStore FederationEventStore
	stateStore *InMemoryStateStore
	serverName string
}

// NewBackfillHandler creates a new BackfillHandler.
func NewBackfillHandler(serverName string, store FederationEventStore, state *InMemoryStateStore) *BackfillHandler {
	return &BackfillHandler{
		eventStore: store,
		stateStore: state,
		serverName: serverName,
	}
}

// Backfill handles GET /_matrix/federation/v1/backfill/{roomId}?v=...&limit=...
// Walks the prev_events DAG backwards from the given event IDs and returns
// up to `limit` events. The default limit is 100, max 1000.
func (h *BackfillHandler) Backfill(c *fiber.Ctx) error {
	roomIDStr := c.Params("roomId")
	if _, err := ParseRoomID(roomIDStr); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errcode": "M_UNKNOWN",
			"error":   "invalid room ID",
		})
	}

	// 'v' query param can be repeated (?v=a&v=b) or comma-separated.
	startIDs := c.Context().QueryArgs().PeekMulti("v")
	var eventIDs []string
	for _, b := range startIDs {
		s := string(b)
		if s == "" {
			continue
		}
		if strings.Contains(s, ",") {
			for _, p := range strings.Split(s, ",") {
				if p = strings.TrimSpace(p); p != "" {
					eventIDs = append(eventIDs, p)
				}
			}
		} else {
			eventIDs = append(eventIDs, s)
		}
	}

	if len(eventIDs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errcode": "M_UNKNOWN",
			"error":   "missing v parameter",
		})
	}

	limit := c.QueryInt("limit", 100)
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	ctx := context.Background()
	events := []*Event{}
	seen := make(map[string]bool)
	queue := append([]string{}, eventIDs...)

	for len(events) < limit && len(queue) > 0 {
		currentID := queue[0]
		queue = queue[1:]

		if seen[currentID] {
			continue
		}
		seen[currentID] = true

		ev, err := h.eventStore.GetEvent(ctx, currentID)
		if err != nil {
			continue
		}
		if ev.RoomID != roomIDStr {
			continue
		}

		events = append(events, ev)

		for _, prev := range ev.PrevEvents {
			if !seen[prev] {
				queue = append(queue, prev)
			}
		}
	}

	pdus := make([]map[string]interface{}, len(events))
	for i, e := range events {
		pdus[i] = eventToMap(e)
	}

	return c.JSON(fiber.Map{
		"origin":           h.serverName,
		"origin_server_ts": time.Now().UnixMilli(),
		"pdus":             pdus,
	})
}

// SetupBackfillRoutes mounts the federation backfill route.
func SetupBackfillRoutes(app *fiber.App, handler *BackfillHandler) {
	app.Get("/_matrix/federation/v1/backfill/:roomId", handler.Backfill)
}
