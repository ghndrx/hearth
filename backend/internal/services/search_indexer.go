package services

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// MessageIndexer handles background indexing of messages for full-text search
type MessageIndexer struct {
	db           *sqlx.DB
	indexBatch   int
	indexWorkers int
	indexQueue   chan uuid.UUID
	wg           sync.WaitGroup
	stopCh       chan struct{}
}

// NewMessageIndexer creates a new message indexer
func NewMessageIndexer(db *sqlx.DB, batchSize, workers int) *MessageIndexer {
	return &MessageIndexer{
		db:           db,
		indexBatch:   batchSize,
		indexWorkers: workers,
		indexQueue:   make(chan uuid.UUID, 10000),
		stopCh:       make(chan struct{}),
	}
}

// Start begins the background indexing workers
func (idx *MessageIndexer) Start(ctx context.Context) {
	// Start worker goroutines
	for i := 0; i < idx.indexWorkers; i++ {
		idx.wg.Add(1)
		go idx.worker(ctx, i)
	}

	// Start the queue processor
	idx.wg.Add(1)
	go idx.queueProcessor(ctx)
}

// Stop gracefully stops the indexer
func (idx *MessageIndexer) Stop() {
	close(idx.stopCh)
	idx.wg.Wait()
}

// IndexMessage adds a message to the indexing queue
func (idx *MessageIndexer) IndexMessage(messageID uuid.UUID) {
	select {
	case idx.indexQueue <- messageID:
	default:
		// Queue is full, log and skip
		log.Printf("Warning: message indexing queue is full, skipping message %s", messageID)
	}
}

// IndexMessages adds multiple messages to the indexing queue
func (idx *MessageIndexer) IndexMessages(messageIDs []uuid.UUID) {
	for _, id := range messageIDs {
		idx.IndexMessage(id)
	}
}

// worker processes messages from the queue
func (idx *MessageIndexer) worker(ctx context.Context, workerID int) {
	defer idx.wg.Done()

	batch := make([]uuid.UUID, 0, idx.indexBatch)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Flush remaining batch
			if len(batch) > 0 {
				idx.indexBatchInternal(ctx, batch)
			}
			return

		case <-idx.stopCh:
			// Flush remaining batch
			if len(batch) > 0 {
				idx.indexBatchInternal(ctx, batch)
			}
			return

		case <-ticker.C:
			// Flush batch on timer
			if len(batch) > 0 {
				idx.indexBatchInternal(ctx, batch)
				batch = batch[:0]
			}

		case messageID, ok := <-idx.indexQueue:
			if !ok {
				return
			}
			batch = append(batch, messageID)
			if len(batch) >= idx.indexBatch {
				idx.indexBatchInternal(ctx, batch)
				batch = batch[:0]
			}
		}
	}
}

// queueProcessor monitors queue health
func (idx *MessageIndexer) queueProcessor(ctx context.Context) {
	defer idx.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-idx.stopCh:
			return
		case <-ticker.C:
			queueLen := len(idx.indexQueue)
			if queueLen > 5000 {
				log.Printf("Warning: message indexing queue has %d pending items", queueLen)
			}
		}
	}
}

// indexBatchInternal indexes a batch of messages
func (idx *MessageIndexer) indexBatchInternal(ctx context.Context, messageIDs []uuid.UUID) {
	if len(messageIDs) == 0 {
		return
	}

	// Use PostgreSQL FTS to update search_vector for the messages
	query := `
		UPDATE messages 
		SET search_vector = to_tsvector('english', COALESCE(content, ''))
		WHERE id = ANY($1)
		AND (search_vector IS NULL OR search_vector != to_tsvector('english', COALESCE(content, '')))
	`

	_, err := idx.db.ExecContext(ctx, query, uuidSliceToInterface(messageIDs))
	if err != nil {
		log.Printf("Error indexing messages: %v", err)
	}
}

// IndexAll indexes all messages without search_vector
func (idx *MessageIndexer) IndexAll(ctx context.Context) error {
	var total, indexed int64

	for {
		// Process in batches
		result, err := idx.db.ExecContext(ctx, `
			UPDATE messages 
			SET search_vector = to_tsvector('english', COALESCE(content, ''))
			WHERE id IN (
				SELECT id FROM messages 
				WHERE search_vector IS NULL 
				ORDER BY created_at 
				LIMIT $1
			)
		`, idx.indexBatch)

		if err != nil {
			return err
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			break
		}

		indexed += rowsAffected
		total += rowsAffected

		// Log progress
		log.Printf("Indexed %d messages (total: %d)", rowsAffected, total)

		// Check for context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	log.Printf("Finished indexing %d total messages", total)
	return nil
}

// IndexChannel indexes all messages in a channel
func (idx *MessageIndexer) IndexChannel(ctx context.Context, channelID uuid.UUID) error {
	var total, indexed int64

	for {
		result, err := idx.db.ExecContext(ctx, `
			UPDATE messages 
			SET search_vector = to_tsvector('english', COALESCE(content, ''))
			WHERE channel_id = $1
			AND (search_vector IS NULL OR search_vector != to_tsvector('english', COALESCE(content, '')))
			AND id IN (
				SELECT id FROM messages 
				WHERE channel_id = $1
				AND search_vector IS NULL
				ORDER BY created_at 
				LIMIT $2
			)
		`, channelID, idx.indexBatch)

		if err != nil {
			return err
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			break
		}

		indexed += rowsAffected
		total += rowsAffected

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	log.Printf("Indexed %d messages in channel %s", total, channelID)
	return nil
}

// IndexServer indexes all messages in all channels of a server
func (idx *MessageIndexer) IndexServer(ctx context.Context, serverID uuid.UUID) error {
	// Get all channels for the server
	var channelIDs []uuid.UUID
	err := idx.db.SelectContext(ctx, &channelIDs, `
		SELECT id FROM channels WHERE server_id = $1
	`, serverID)

	if err != nil {
		return err
	}

	for _, channelID := range channelIDs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := idx.IndexChannel(ctx, channelID); err != nil {
			log.Printf("Error indexing channel %s: %v", channelID, err)
		}
	}

	return nil
}

// uuidSliceToInterface converts []uuid.UUID to []interface{}
func uuidSliceToInterface(ids []uuid.UUID) []interface{} {
	result := make([]interface{}, len(ids))
	for i, id := range ids {
		result[i] = id
	}
	return result
}
