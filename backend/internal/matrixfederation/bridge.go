// Package matrixfederation implements Matrix Federation protocol support for Hearth.
// This file implements the FederationBridge: a translation layer between
// Hearth message events and Matrix federation PDUs.
package matrixfederation

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// FederationBridge translates Hearth messages into signed Matrix PDUs and
// dispatches them via the TransactionQueue.
type FederationBridge struct {
	queue          *TransactionQueue
	store          FederationEventStore
	state          StateStore
	keyStore       *KeyStore
	roomAliasStore RoomAliasStore
	userService    UserGetter
	serverName     string
}

// NewFederationBridge creates a new FederationBridge.
func NewFederationBridge(
	serverName string,
	queue *TransactionQueue,
	store FederationEventStore,
	state StateStore,
	keyStore *KeyStore,
	roomAliasStore RoomAliasStore,
	userService UserGetter,
) *FederationBridge {
	return &FederationBridge{
		queue:          queue,
		store:          store,
		state:          state,
		keyStore:       keyStore,
		roomAliasStore: roomAliasStore,
		userService:    userService,
		serverName:     serverName,
	}
}

// OnHearthMessage converts a Hearth message into a signed Matrix PDU
// and enqueues it for outgoing federation. If the channel is not federated,
// the call is a no-op (returns nil).
func (b *FederationBridge) OnHearthMessage(
	ctx context.Context,
	messageID uuid.UUID,
	channelID uuid.UUID,
	senderMXID string,
	content string,
) error {
	if b.queue == nil || !b.queue.IsRunning() {
		return nil
	}

	roomID, _, err := b.roomAliasStore.GetByChannelID(ctx, channelID)
	if err != nil {
		// Channel is not federated - silently skip.
		return nil
	}

	rs := b.state.GetOrCreateRoomState(roomID)

	prevEvents := rs.GetForwardExtremities()
	if len(prevEvents) == 0 {
		// Room not yet bootstrapped: no forward extremities yet.
		// Caller must bootstrap room state via create/member events first.
		return fmt.Errorf("room %s has no forward extremities", roomID.String())
	}

	authEvents := []string{}
	if createEvent, err := rs.GetStateEvent(EventTypeCreate, ""); err == nil && createEvent != nil {
		authEvents = append(authEvents, createEvent.EventID)
	}
	if memberEvent, err := rs.GetStateEvent(EventTypeMember, senderMXID); err == nil && memberEvent != nil {
		authEvents = append(authEvents, memberEvent.EventID)
	}
	if plEvent, err := rs.GetStateEvent(EventTypePowerLevels, ""); err == nil && plEvent != nil {
		authEvents = append(authEvents, plEvent.EventID)
	}

	depth := rs.GetCurrentDepth() + 1
	event := NewMessageEvent(messageID, roomID, senderMXID, content, b.serverName, prevEvents, authEvents, depth)

	hash, err := event.ComputeContentHash()
	if err != nil {
		return fmt.Errorf("compute content hash: %w", err)
	}
	event.Hashes = EventHashes{SHA256: hash}

	key, err := b.keyStore.GetPrimaryKey()
	if err != nil {
		return fmt.Errorf("get primary signing key: %w", err)
	}
	if err := event.SignWithServer(b.serverName, key); err != nil {
		return fmt.Errorf("sign event: %w", err)
	}

	if err := b.store.StoreEvent(ctx, event); err != nil {
		return fmt.Errorf("store event: %w", err)
	}

	if err := rs.AddEvent(event); err != nil {
		return fmt.Errorf("update room state: %w", err)
	}

	if err := b.queue.Enqueue(roomID, event); err != nil {
		// Local message is stored; federation enqueue failure is non-fatal.
		return fmt.Errorf("enqueue for federation: %w", err)
	}

	return nil
}

// BroadcastEvent signs an already-constructed event and enqueues it for
// outgoing federation. Used for non-message PDUs (membership, state events).
func (b *FederationBridge) BroadcastEvent(ctx context.Context, roomID RoomID, event *Event) error {
	if b.queue == nil || !b.queue.IsRunning() {
		return nil
	}

	key, err := b.keyStore.GetPrimaryKey()
	if err != nil {
		return fmt.Errorf("get primary signing key: %w", err)
	}
	if err := event.SignWithServer(b.serverName, key); err != nil {
		return fmt.Errorf("sign event: %w", err)
	}

	if err := b.store.StoreEvent(ctx, event); err != nil {
		return fmt.Errorf("store event: %w", err)
	}

	return b.queue.Enqueue(roomID, event)
}
