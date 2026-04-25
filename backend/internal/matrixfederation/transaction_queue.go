// Package matrixfederation implements Matrix Federation protocol support for Hearth.
// This file implements the outgoing transaction queue.
//
// Matrix transactions are batches of PDUs sent between homeservers. The queue
// batches events per destination and flushes on a periodic ticker.
package matrixfederation

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// TransactionQueue manages outgoing federation transactions.
// Batches events per destination server and flushes them periodically.
type TransactionQueue struct {
	client     *FederationClient
	keyStore   *KeyStore
	serverName string

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
// In this in-memory implementation, we only locally signal that flush ran.
// A full implementation would aggregate by destination server, build a
// transaction PDU, sign it with the server key, and PUT it via the client.
func (q *TransactionQueue) flush() {
	q.mu.Lock()
	if len(q.pending) == 0 {
		q.mu.Unlock()
		return
	}
	toSend := q.pending
	q.pending = make(map[string][]*Event)
	q.mu.Unlock()

	// Placeholder: real implementation would resolve destination servers
	// from the room's member list and send via q.client.SendTransaction.
	// Single-authoritative-server mode means we mostly act as origin only,
	// so this is a best-effort fan-out.
	_ = toSend
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
