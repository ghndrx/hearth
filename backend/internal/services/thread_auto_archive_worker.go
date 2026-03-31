package services

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"

	"hearth/internal/models"
)

// ThreadAutoArchiveWorker processes threads that are ready for auto-archive
type ThreadAutoArchiveWorker struct {
	autoArchiveRepo ThreadAutoArchiveRepositoryInterface
	threadRepo      ThreadRepository
	channelRepo     ChannelRepository
	eventBus        EventBus
	
	stopCh          chan struct{}
	wg              sync.WaitGroup
	batchSize       int
	checkInterval   time.Duration
	isRunning       bool
	mu              sync.Mutex
}

// NewThreadAutoArchiveWorker creates a new auto-archive worker
func NewThreadAutoArchiveWorker(
	autoArchiveRepo ThreadAutoArchiveRepositoryInterface,
	threadRepo ThreadRepository,
	channelRepo ChannelRepository,
	eventBus EventBus,
) *ThreadAutoArchiveWorker {
	return &ThreadAutoArchiveWorker{
		autoArchiveRepo: autoArchiveRepo,
		threadRepo:       threadRepo,
		channelRepo:      channelRepo,
		eventBus:         eventBus,
		stopCh:          make(chan struct{}),
		batchSize:       50,
		checkInterval:   1 * time.Minute,
	}
}

// Start begins the background worker
func (w *ThreadAutoArchiveWorker) Start() {
	w.mu.Lock()
	if w.isRunning {
		w.mu.Unlock()
		return
	}
	w.isRunning = true
	w.mu.Unlock()

	w.wg.Add(1)
	go w.run()

	log.Println("Thread auto-archive worker started")
}

// Stop gracefully stops the worker
func (w *ThreadAutoArchiveWorker) Stop() {
	w.mu.Lock()
	if !w.isRunning {
		w.mu.Unlock()
		return
	}
	w.isRunning = false
	w.mu.Unlock()

	close(w.stopCh)
	w.wg.Wait()

	log.Println("Thread auto-archive worker stopped")
}

func (w *ThreadAutoArchiveWorker) run() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.checkInterval)
	defer ticker.Stop()

	// Run immediately on start
	w.processReadyThreads()

	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.processReadyThreads()
		}
	}
}

func (w *ThreadAutoArchiveWorker) processReadyThreads() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Get threads ready for archive
	metas, err := w.autoArchiveRepo.GetThreadsReadyForArchive(ctx, w.batchSize)
	if err != nil {
		log.Printf("Error fetching threads ready for archive: %v", err)
		return
	}

	if len(metas) == 0 {
		return
	}

	log.Printf("Processing %d threads for auto-archive", len(metas))

	for _, meta := range metas {
		select {
		case <-w.stopCh:
			return
		default:
			w.archiveThread(ctx, meta)
		}
	}
}

func (w *ThreadAutoArchiveWorker) archiveThread(ctx context.Context, meta *models.ThreadAutoArchiveMeta) {
	threadID := meta.ThreadID

	// Get thread details
	thread, err := w.threadRepo.GetByID(ctx, threadID)
	if err != nil || thread == nil {
		log.Printf("Error getting thread %s: %v", threadID, err)
		return
	}

	// Skip if already archived
	if thread.Archived {
		// Clear the meta
		meta.NextArchiveAt = nil
		meta.ArchiveEligible = false
		w.autoArchiveRepo.UpdateThreadMeta(ctx, meta)
		return
	}

	// Check if owner has bumped since last check
	if meta.BumpedByOwner {
		// Reset bumped flag and update next archive time
		meta.BumpedByOwner = false
		meta.ArchiveEligible = false
		w.autoArchiveRepo.UpdateThreadMeta(ctx, meta)
		
		// Emit bump event
		w.eventBus.Publish(EventThreadAutoArchiveBumped, map[string]interface{}{
			"thread_id":    threadID,
			"bumped_by_owner": true,
		})
		return
	}

	// Archive the thread
	if err := w.threadRepo.Archive(ctx, threadID); err != nil {
		log.Printf("Error archiving thread %s: %v", threadID, err)
		return
	}

	// Update meta
	meta.NextArchiveAt = nil
	meta.ArchiveEligible = false
	w.autoArchiveRepo.UpdateThreadMeta(ctx, meta)

	// Emit archive event
	w.eventBus.Publish(EventThreadAutoArchived, map[string]interface{}{
		"thread_id":    threadID,
		"channel_id":   thread.ParentChannelID,
		"auto_archive": true,
	})

	log.Printf("Auto-archived thread %s", threadID)
}

// ProcessThreadActivity updates auto-archive metadata when a thread receives activity
func (w *ThreadAutoArchiveWorker) ProcessThreadActivity(ctx context.Context, threadID, userID uuid.UUID) error {
	thread, err := w.threadRepo.GetByID(ctx, threadID)
	if err != nil || thread == nil {
		return ErrThreadNotFound
	}

	// Skip if archived
	if thread.Archived {
		return nil
	}

	channel, err := w.channelRepo.GetByID(ctx, thread.ParentChannelID)
	if err != nil || channel == nil {
		return ErrChannelNotFound
	}

	var serverID uuid.UUID
	if channel.ServerID != nil {
		serverID = *channel.ServerID
	}

	// Get the effective auto-archive duration
	duration := 1440 // Default 24 hours
	if serverID != uuid.Nil {
		duration, err = w.autoArchiveRepo.GetChannelDuration(ctx, thread.ParentChannelID, serverID)
		if err != nil {
			return err
		}
	}

	// Get or create meta
	meta, err := w.autoArchiveRepo.GetOrCreateThreadMeta(ctx, threadID)
	if err != nil {
		return err
	}

	// Update last activity
	meta.LastActivityAt = time.Now()
	meta.LastActivityUserID = &userID

	// Check if bumped by owner
	if thread.OwnerID == userID {
		meta.BumpedByOwner = true
	}

	// Calculate next archive time
	nextArchive := time.Now().Add(time.Duration(duration) * time.Minute)
	meta.NextArchiveAt = &nextArchive
	meta.ArchiveEligible = true

	return w.autoArchiveRepo.UpdateThreadMeta(ctx, meta)
}

// GetWorkerStatus returns the current status of the worker
func (w *ThreadAutoArchiveWorker) GetWorkerStatus() map[string]interface{} {
	w.mu.Lock()
	defer w.mu.Unlock()

	return map[string]interface{}{
		"is_running":     w.isRunning,
		"batch_size":     w.batchSize,
		"check_interval": w.checkInterval.String(),
	}
}

// SetBatchSize sets the number of threads to process per batch
func (w *ThreadAutoArchiveWorker) SetBatchSize(size int) {
	if size > 0 && size <= 500 {
		w.batchSize = size
	}
}

// SetCheckInterval sets how often the worker checks for threads to archive
func (w *ThreadAutoArchiveWorker) SetCheckInterval(interval time.Duration) {
	if interval >= 10*time.Second {
		w.checkInterval = interval
	}
}
