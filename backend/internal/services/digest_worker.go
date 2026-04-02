package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// DigestWorker runs in the background generating notification digests
type DigestWorker struct {
	smartService *SmartNotificationService
	cache        SmartNotificationCache
	eventBus     EventBus
	interval     time.Duration
	stopCh       chan struct{}
	wg           sync.WaitGroup
	running      bool
	mu           sync.Mutex
}

// NewDigestWorker creates a new background digest worker
func NewDigestWorker(
	smartService *SmartNotificationService,
	cache SmartNotificationCache,
	eventBus EventBus,
	interval time.Duration,
) *DigestWorker {
	if interval == 0 {
		interval = 5 * time.Minute
	}
	return &DigestWorker{
		smartService: smartService,
		cache:        cache,
		eventBus:     eventBus,
		interval:     interval,
		stopCh:       make(chan struct{}),
	}
}

// Start begins the digest worker loop
func (w *DigestWorker) Start() {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return
	}
	w.running = true
	w.mu.Unlock()

	w.wg.Add(1)
	go w.run()
}

// Stop gracefully stops the digest worker
func (w *DigestWorker) Stop() {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return
	}
	w.running = false
	w.mu.Unlock()

	close(w.stopCh)
	w.wg.Wait()
}

// IsRunning returns whether the worker is currently running
func (w *DigestWorker) IsRunning() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.running
}

func (w *DigestWorker) run() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.processDigests()
		}
	}
}

// processDigests checks all users with pending digests and creates digests for eligible ones
func (w *DigestWorker) processDigests() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	userIDs, err := w.getEligibleUsers(ctx)
	if err != nil {
		log.Printf("digest worker: failed to get eligible users: %v", err)
		return
	}

	for _, userID := range userIDs {
		if err := w.processUserDigest(ctx, userID); err != nil {
			log.Printf("digest worker: failed to process digest for user %s: %v", userID, err)
		}
	}
}

// processUserDigest creates a digest for a single user if they have enough pending notifications
func (w *DigestWorker) processUserDigest(ctx context.Context, userID uuid.UUID) error {
	prefs := w.smartService.GetUserPreferences(ctx, userID)
	if !prefs.DigestEnabled {
		return nil
	}

	pending, err := w.smartService.GetPendingDigestNotifications(ctx, userID)
	if err != nil {
		return err
	}

	if len(pending) == 0 {
		return nil
	}

	// Check if enough time has elapsed since last digest
	lastDigestKey := fmt.Sprintf("smart_notif:last_digest:%s", userID.String())
	if data, err := w.cache.Get(ctx, lastDigestKey); err == nil {
		var lastDigestTime time.Time
		if json.Unmarshal(data, &lastDigestTime) == nil {
			interval := time.Duration(prefs.DigestIntervalMins) * time.Minute
			if time.Since(lastDigestTime) < interval {
				return nil // not time yet
			}
		}
	}

	// Create the digest
	digest, err := w.smartService.CreateDigest(ctx, userID)
	if err != nil {
		return err
	}

	if digest == nil {
		return nil
	}

	// Record the digest timestamp
	now := time.Now()
	data, _ := json.Marshal(now)
	_ = w.cache.Set(ctx, lastDigestKey, data, 24*time.Hour)

	return nil
}

// getEligibleUsers returns user IDs that have pending digest notifications
func (w *DigestWorker) getEligibleUsers(ctx context.Context) ([]uuid.UUID, error) {
	key := "smart_notif:digest_eligible_users"
	data, err := w.cache.Get(ctx, key)
	if err != nil {
		return nil, nil // no eligible users
	}

	var userIDs []uuid.UUID
	if err := json.Unmarshal(data, &userIDs); err != nil {
		return nil, nil
	}

	return userIDs, nil
}

// RegisterEligibleUser adds a user to the eligible users set for digest processing
func (w *DigestWorker) RegisterEligibleUser(ctx context.Context, userID uuid.UUID) error {
	key := "smart_notif:digest_eligible_users"

	var userIDs []uuid.UUID
	if data, err := w.cache.Get(ctx, key); err == nil {
		_ = json.Unmarshal(data, &userIDs)
	}

	// Check if already registered
	for _, id := range userIDs {
		if id == userID {
			return nil
		}
	}

	userIDs = append(userIDs, userID)
	data, _ := json.Marshal(userIDs)
	return w.cache.Set(ctx, key, data, 24*time.Hour)
}

// UnregisterEligibleUser removes a user from the eligible users set
func (w *DigestWorker) UnregisterEligibleUser(ctx context.Context, userID uuid.UUID) error {
	key := "smart_notif:digest_eligible_users"

	var userIDs []uuid.UUID
	if data, err := w.cache.Get(ctx, key); err == nil {
		_ = json.Unmarshal(data, &userIDs)
	}

	filtered := make([]uuid.UUID, 0, len(userIDs))
	for _, id := range userIDs {
		if id != userID {
			filtered = append(filtered, id)
		}
	}

	data, _ := json.Marshal(filtered)
	return w.cache.Set(ctx, key, data, 24*time.Hour)
}

// Tick forces an immediate digest processing cycle (useful for testing)
func (w *DigestWorker) Tick() {
	w.processDigests()
}
