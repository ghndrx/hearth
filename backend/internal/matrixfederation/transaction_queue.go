// Package matrixfederation implements Matrix Federation protocol support for Hearth.
// This file implements the outgoing transaction queue.
//
// Matrix transactions are batches of PDUs sent between homeservers. The queue
// batches events per destination and flushes on a periodic ticker.
package matrixfederation

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// TransactionQueue manages outgoing federation transactions.
// Batches events per destination server and flushes them periodically.
type TransactionQueue struct {
	client     *FederationClient
	keyStore   *KeyStore
	serverName string
	stateStore StateStore

	mu      sync.Mutex
	pending map[string][]*Event // key = roomID
	running bool
	stopCh  chan struct{}
}

// NewTransactionQueue creates a new TransactionQueue.
func NewTransactionQueue(client *FederationClient, keyStore *KeyStore, serverName string) *TransactionQueue {
	return &TransactionQueue{
		client:     client,
		keyStore:   keyStore,
		serverName: serverName,
		pending:    make(map[string][]*Event),
		stopCh:     make(chan struct{}),
	}
}

// SetStateStore sets the state store used to resolve destination servers.
func (q *TransactionQueue) SetStateStore(store StateStore) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.stateStore = store
}

// Enqueue adds an event to the outgoing queue for the given room.
func (q *TransactionQueue) Enqueue(roomID RoomID, event *Event) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if !q.running {
		return fmt.Errorf("transaction queue not started")
	}

	key := roomID.String()
	q.pending[key] = append(q.pending[key], event)
	return nil
}

// Start begins the periodic flush worker.
func (q *TransactionQueue) Start(ctx context.Context) error {
	q.mu.Lock()
	if q.running {
		q.mu.Unlock()
		return fmt.Errorf("transaction queue already running")
	}
	q.running = true
	q.stopCh = make(chan struct{})
	q.mu.Unlock()

	ticker := time.NewTicker(30 * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				q.flush()
				return
			case <-q.stopCh:
				q.flush()
				return
			case <-ticker.C:
				q.flush()
			}
		}
	}()
	return nil
}

// Stop halts the queue worker after flushing pending events.
func (q *TransactionQueue) Stop() error {
	q.mu.Lock()
	if !q.running {
		q.mu.Unlock()
		return nil
	}
	q.running = false
	close(q.stopCh)
	q.mu.Unlock()
	return nil
}

// FlushNow forces an immediate flush of all pending events.
func (q *TransactionQueue) FlushNow() {
	q.flush()
}

// flush attempts to send all pending events.
// It groups events by destination server, builds a signed transaction PDU,
// and sends it via the federation client.
func (q *TransactionQueue) flush() {
	q.mu.Lock()
	if len(q.pending) == 0 {
		q.mu.Unlock()
		return
	}
	toSend := q.pending
	q.pending = make(map[string][]*Event)
	q.mu.Unlock()

	// Group events by destination server.
	byDestination := make(map[string][]*Event)
	for roomID, events := range toSend {
		destinations := q.resolveDestinations(roomID)
		if len(destinations) == 0 {
			// No known destinations; skip this room.
			continue
		}
		for _, dest := range destinations {
			byDestination[dest] = append(byDestination[dest], events...)
		}
	}

	if len(byDestination) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	for dest, events := range byDestination {
		pdus := make([]map[string]interface{}, len(events))
		for i, e := range events {
			pdus[i] = eventToMap(e)
		}

		txnID := fmt.Sprintf("%d-%s", time.Now().UnixMilli(), uuid.New().String()[:8])
		if err := q.client.SendTransaction(ctx, dest, txnID, pdus, nil); err != nil {
			log.Printf("⚠️  federation flush: failed to send transaction to %s: %v", dest, err)
			// Retry: re-enqueue events for next flush.
			for _, e := range events {
				rid, err := ParseRoomID(e.RoomID)
				if err != nil {
					continue
				}
				_ = q.Enqueue(rid, e)
			}
		}
	}
}

// resolveDestinations returns the list of remote destination servers for a room.
func (q *TransactionQueue) resolveDestinations(roomID string) []string {
	if q.stateStore == nil {
		return nil
	}

	rid, err := ParseRoomID(roomID)
	if err != nil {
		return nil
	}

	rs, err := q.stateStore.GetRoomState(rid)
	if err != nil {
		return nil
	}

	members := rs.GetMembers()
	destinations := make(map[string]struct{})
	for mxid := range members {
		parts := strings.SplitN(mxid, ":", 2)
		if len(parts) == 2 && parts[1] != q.serverName {
			destinations[parts[1]] = struct{}{}
		}
	}

	result := make([]string, 0, len(destinations))
	for d := range destinations {
		result = append(result, d)
	}
	return result
}

// PendingCount returns the total count of pending events across all rooms.
func (q *TransactionQueue) PendingCount() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	total := 0
	for _, evts := range q.pending {
		total += len(evts)
	}
	return total
}

// IsRunning reports whether the queue is started.
func (q *TransactionQueue) IsRunning() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.running
}
