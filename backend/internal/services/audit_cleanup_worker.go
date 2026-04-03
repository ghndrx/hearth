package services

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// AuditCleanupConfig holds configuration for the cleanup worker
type AuditCleanupConfig struct {
	// Default retention period in days
	DefaultRetentionDays int
	// How often to run cleanup (default: every hour)
	CleanupInterval time.Duration
	// Batch size for deletion
	BatchSize int
}

// DefaultAuditCleanupConfig returns sensible defaults
func DefaultAuditCleanupConfig() *AuditCleanupConfig {
	return &AuditCleanupConfig{
		DefaultRetentionDays: 90,
		CleanupInterval:      time.Hour,
		BatchSize:            1000,
	}
}

// AuditRetentionRepository interface for per-server retention settings
type AuditRetentionRepository interface {
	GetRetentionDays(ctx context.Context, serverID string) (int, error)
	GetAllServerIDs(ctx context.Context) ([]string, error)
}

// AuditLogCleanupRepository interface for audit log cleanup operations
type AuditLogCleanupRepository interface {
	DeleteOlderThan(ctx context.Context, serverID uuid.UUID, before time.Time) (int64, error)
	DeleteAllOlderThan(ctx context.Context, before time.Time) (int64, error)
}

// AuditCleanupWorker handles automatic cleanup of old audit logs
type AuditCleanupWorker struct {
	repo            AuditLogCleanupRepository
	retentionRepo   AuditRetentionRepository
	config          *AuditCleanupConfig
	stopCh          chan struct{}
	doneCh          chan struct{}
	mu              sync.Mutex
	isRunning       bool
}

// NewAuditCleanupWorker creates a new cleanup worker
func NewAuditCleanupWorker(
	repo AuditLogCleanupRepository,
	retentionRepo AuditRetentionRepository,
	config *AuditCleanupConfig,
) *AuditCleanupWorker {
	if config == nil {
		config = DefaultAuditCleanupConfig()
	}
	return &AuditCleanupWorker{
		repo:          repo,
		retentionRepo: retentionRepo,
		config:        config,
		stopCh:        make(chan struct{}),
		doneCh:        make(chan struct{}),
	}
}

// Start begins the cleanup worker
func (w *AuditCleanupWorker) Start() {
	w.mu.Lock()
	if w.isRunning {
		w.mu.Unlock()
		return
	}
	w.isRunning = true
	w.mu.Unlock()

	go w.run()
	log.Println("Audit cleanup worker started")
}

// Stop gracefully stops the cleanup worker
func (w *AuditCleanupWorker) Stop() {
	w.mu.Lock()
	if !w.isRunning {
		w.mu.Unlock()
		return
	}
	w.mu.Unlock()

	close(w.stopCh)
	<-w.doneCh
	log.Println("Audit cleanup worker stopped")
}

func (w *AuditCleanupWorker) run() {
	defer close(w.doneCh)

	// Run immediately on start
	w.cleanup()

	ticker := time.NewTicker(w.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.cleanup()
		case <-w.stopCh:
			return
		}
	}
}

func (w *AuditCleanupWorker) cleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	log.Printf("Starting audit log cleanup with retention period of %d days", w.config.DefaultRetentionDays)

	// Clean up using the global retention period
	cutoff := time.Now().AddDate(0, 0, -w.config.DefaultRetentionDays)
	deleted, err := w.repo.DeleteAllOlderThan(ctx, cutoff)
	if err != nil {
		log.Printf("Error during audit log cleanup: %v", err)
		return
	}

	if deleted > 0 {
		log.Printf("Audit log cleanup completed: deleted %d entries older than %v", deleted, cutoff)
	} else {
		log.Printf("Audit log cleanup completed: no entries to delete")
	}
}

// RunOnce runs cleanup manually (useful for testing or on-demand cleanup)
func (w *AuditCleanupWorker) RunOnce(ctx context.Context) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -w.config.DefaultRetentionDays)
	return w.repo.DeleteAllOlderThan(ctx, cutoff)
}

// RunOnceForServer runs cleanup for a specific server
func (w *AuditCleanupWorker) RunOnceForServer(ctx context.Context, serverID uuid.UUID, retentionDays int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	return w.repo.DeleteOlderThan(ctx, serverID, cutoff)
}

// GetRetentionInfo returns information about the current retention settings
func (w *AuditCleanupWorker) GetRetentionInfo() map[string]interface{} {
	return map[string]interface{}{
		"default_retention_days": w.config.DefaultRetentionDays,
		"cleanup_interval":        w.config.CleanupInterval.String(),
		"batch_size":              w.config.BatchSize,
		"is_running":              w.isRunning,
	}
}
