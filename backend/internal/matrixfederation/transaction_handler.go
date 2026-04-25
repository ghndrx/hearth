// Package matrixfederation implements Matrix Federation protocol support for Hearth.
// This file implements the /send endpoint for receiving federation transactions.
//
// Matrix Spec References:
//   - Pushing Events: https://spec.matrix.org/v1.16/server-server-api/#pushing-events--pdus
//   - Transaction Format: https://spec.matrix.org/v1.16/server-server-api/#transactions
package matrixfederation

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

// TransactionRequest represents an incoming /send request.
type TransactionRequest struct {
	// Origin is the server name of the sending server.
	Origin string `json:"origin"`

	// OriginServerTS is the timestamp when the transaction was created.
	OriginServerTS int64 `json:"origin_server_ts"`

	// PDUs is a list of events (PDUs) being sent.
	PDUs []json.RawMessage `json:"pdus"`

	// EDUs is a list of ephemeral data units (e.g., typing, presence).
	EDUs []json.RawMessage `json:"edus,omitempty"`
}

// TransactionResponse is the response to a /send request.
type TransactionResponse struct {
	// PDUs is a map of event ID -> result (empty string = success, error code = failure).
	PDUs map[string]interface{} `json:"pdus,omitempty"`
}

// TransactionProcessor handles incoming federation transactions.
type TransactionProcessor struct {
	// eventStore stores received events.
	eventStore FederationEventStore

	// stateStore manages room state.
	stateStore *InMemoryStateStore

	// authChecker validates event authorization.
	authChecker *AuthChecker

	// serverName is the name of this homeserver.
	serverName string
}

// NewTransactionProcessor creates a new transaction processor.
func NewTransactionProcessor(
	serverName string,
	eventStore FederationEventStore,
	stateStore *InMemoryStateStore,
	authChecker *AuthChecker,
) *TransactionProcessor {
	return &TransactionProcessor{
		eventStore:  eventStore,
		stateStore:  stateStore,
		authChecker: authChecker,
		serverName:  serverName,
	}
}

// TransactionHandler provides HTTP handlers for incoming transactions.
type TransactionHandler struct {
	processor *TransactionProcessor
}

// NewTransactionHandler creates a new transaction handler.
func NewTransactionHandler(processor *TransactionProcessor) *TransactionHandler {
	return &TransactionHandler{processor: processor}
}

// Send handles POST /_matrix/federation/v1/send/{txnId}.
func (h *TransactionHandler) Send(c *fiber.Ctx) error {
	// Get the verified origin from the middleware context
	origin := c.Locals("matrix_origin")
	if origin == nil {
		return c.Status(http.StatusForbidden).JSON(map[string]interface{}{
			"errcode": "M_FORBIDDEN",
			"error":   "Missing verified origin",
		})
	}

	originServer, ok := origin.(string)
	if !ok {
		return c.Status(http.StatusInternalServerError).JSON(map[string]interface{}{
			"errcode": "M_UNKNOWN",
			"error":   "Invalid origin type",
		})
	}

	// Parse the transaction
	var txn TransactionRequest
	if err := c.BodyParser(&txn); err != nil {
		return c.Status(http.StatusBadRequest).JSON(map[string]interface{}{
			"errcode": "M_NOT_JSON",
			"error":   fmt.Sprintf("Invalid JSON: %v", err),
		})
	}

	// Verify the origin matches the authenticated origin
	if txn.Origin != originServer {
		return c.Status(http.StatusForbidden).JSON(map[string]interface{}{
			"errcode": "M_FORBIDDEN",
			"error":   fmt.Sprintf("Origin mismatch: got %q, expected %q", txn.Origin, originServer),
		})
	}

	// Process the transaction
	result := h.processor.ProcessTransaction(&txn)

	return c.Status(http.StatusOK).JSON(result)
}

// ProcessTransaction processes all PDUs in a transaction and returns the result.
func (tp *TransactionProcessor) ProcessTransaction(txn *TransactionRequest) *TransactionResponse {
	result := &TransactionResponse{
		PDUs: make(map[string]interface{}),
	}

	for _, pduRaw := range txn.PDUs {
		// Parse the event
		var event Event
		if err := json.Unmarshal(pduRaw, &event); err != nil {
			// Can't parse - add error for unknown event
			result.PDUs["unknown"] = fmt.Sprintf("M_NOT_JSON: %v", err)
			continue
		}

		// Process the event
		err := tp.ProcessEvent(&event)
		if err != nil {
			result.PDUs[event.EventID] = err.Error()
		} else {
			result.PDUs[event.EventID] = struct{}{}
		}
	}

	return result
}

// ProcessEvent processes a single incoming event.
func (tp *TransactionProcessor) ProcessEvent(event *Event) error {
	// Basic validation
	if err := event.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Check if we already have this event
	exists, err := tp.eventStore.HasEvent(nil, event.EventID)
	if err != nil {
		return fmt.Errorf("failed to check event existence: %w", err)
	}
	if exists {
		// Event already processed - idempotent success
		return nil
	}

	// Get the room state
	roomID, err := ParseRoomID(event.RoomID)
	if err != nil {
		return fmt.Errorf("invalid room ID: %w", err)
	}

	// Get or create room state
	roomState := tp.stateStore.GetOrCreateRoomState(roomID)

	// Create auth event provider using the event store
	authProvider := &storeAuthProvider{
		store:      tp.eventStore,
		roomState:  roomState,
		event:      event,
		serverName: tp.serverName,
	}

	// Check authorization
	authResult := tp.authChecker.CheckAuthRules(event, authProvider)
	if !authResult.Allowed {
		return fmt.Errorf("auth failed: %s", authResult.Reason)
	}

	// Store the event
	if err := tp.eventStore.StoreEvent(nil, event); err != nil {
		return fmt.Errorf("failed to store event: %w", err)
	}

	// Update room state
	if err := roomState.AddEvent(event); err != nil {
		return fmt.Errorf("failed to update room state: %w", err)
	}

	return nil
}

// storeAuthProvider implements AuthEventProvider using the event store and room state.
type storeAuthProvider struct {
	store      FederationEventStore
	roomState  *FederatedRoomState
	event      *Event
	serverName string
}

// GetAuthEvents retrieves the auth events for an event.
func (ap *storeAuthProvider) GetAuthEvents(_ *Event) ([]*Event, error) {
	var authEvents []*Event

	// Always include the create event
	if ap.roomState != nil {
		// Try to get create event from room state
		if createEvent, err := ap.roomState.GetStateEvent(EventTypeCreate, ""); err == nil && createEvent != nil {
			authEvents = append(authEvents, createEvent)
		}
	}

	// Get power levels
	if ap.roomState != nil {
		if plEvent, err := ap.roomState.GetStateEvent(EventTypePowerLevels, ""); err == nil && plEvent != nil {
			authEvents = append(authEvents, plEvent)
		}
	}

	// Get member event for sender
	if ap.roomState != nil {
		if memberEvent, err := ap.roomState.GetStateEvent(EventTypeMember, ap.event.Sender); err == nil && memberEvent != nil {
			authEvents = append(authEvents, memberEvent)
		}
	}

	// Get join rules
	if ap.roomState != nil {
		if jrEvent, err := ap.roomState.GetStateEvent(EventTypeJoinRules, ""); err == nil && jrEvent != nil {
			authEvents = append(authEvents, jrEvent)
		}
	}

	return authEvents, nil
}

// SetupTransactionRoutes configures the federation transaction routes.
func SetupTransactionRoutes(app *fiber.App, handler *TransactionHandler) {
	app.Put("/_matrix/federation/v1/send/:txnId", handler.Send)
	app.Post("/_matrix/federation/v1/send/:txnId", handler.Send)
}
