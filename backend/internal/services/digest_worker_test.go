package services

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"hearth/internal/models"
)

func TestDigestWorker_NewDigestWorker(t *testing.T) {
	cache := &mapCache{data: make(map[string][]byte)}
	eventBus := new(MockEventBus)
	repo := new(MockNotificationRepository)
	service := NewSmartNotificationService(repo, cache, eventBus)

	worker := NewDigestWorker(service, cache, eventBus, 5*time.Minute)

	assert.NotNil(t, worker)
	assert.Equal(t, 5*time.Minute, worker.interval)
	assert.False(t, worker.IsRunning())
}

func TestDigestWorker_NewDigestWorker_DefaultInterval(t *testing.T) {
	cache := &mapCache{data: make(map[string][]byte)}
	eventBus := new(MockEventBus)
	repo := new(MockNotificationRepository)
	service := NewSmartNotificationService(repo, cache, eventBus)

	worker := NewDigestWorker(service, cache, eventBus, 0)

	assert.Equal(t, 5*time.Minute, worker.interval)
}

func TestDigestWorker_StartStop(t *testing.T) {
	cache := &mapCache{data: make(map[string][]byte)}
	eventBus := new(MockEventBus)
	repo := new(MockNotificationRepository)
	service := NewSmartNotificationService(repo, cache, eventBus)

	worker := NewDigestWorker(service, cache, eventBus, 1*time.Hour)

	worker.Start()
	assert.True(t, worker.IsRunning())

	// Starting again should be a no-op
	worker.Start()
	assert.True(t, worker.IsRunning())

	worker.Stop()
	assert.False(t, worker.IsRunning())

	// Stopping again should be a no-op
	worker.Stop()
	assert.False(t, worker.IsRunning())
}

func TestDigestWorker_Tick_NoEligibleUsers(t *testing.T) {
	cache := &mapCache{data: make(map[string][]byte)}
	eventBus := new(MockEventBus)
	repo := new(MockNotificationRepository)
	service := NewSmartNotificationService(repo, cache, eventBus)

	worker := NewDigestWorker(service, cache, eventBus, 5*time.Minute)

	// Should not panic with no eligible users
	worker.Tick()
}

func TestDigestWorker_Tick_ProcessesEligibleUser(t *testing.T) {
	cache := &mapCache{data: make(map[string][]byte)}
	eventBus := new(MockEventBus)
	repo := new(MockNotificationRepository)
	service := NewSmartNotificationService(repo, cache, eventBus)
	worker := NewDigestWorker(service, cache, eventBus, 5*time.Minute)

	userID := uuid.New()

	// Register user as eligible
	eligibleKey := "smart_notif:digest_eligible_users"
	userIDs := []uuid.UUID{userID}
	eligibleData, _ := json.Marshal(userIDs)
	cache.data[eligibleKey] = eligibleData

	// Set up preferences (digest enabled)
	prefsKey := fmt.Sprintf("smart_notif:prefs:%s", userID.String())
	prefs := &models.SmartNotificationPreferences{
		UserID:             userID,
		Enabled:            true,
		DigestEnabled:      true,
		DigestIntervalMins: 1, // 1 minute interval for testing
	}
	prefsData, _ := json.Marshal(prefs)
	cache.data[prefsKey] = prefsData

	// Add pending notifications
	pending := []models.SmartNotification{
		{
			Notification: models.Notification{ID: uuid.New(), UserID: userID, Type: models.NotificationTypeReaction, Title: "React"},
			Category:     models.NotifCategoryReaction,
		},
	}
	pendingKey := fmt.Sprintf("smart_notif:pending_digest:%s", userID.String())
	pendingData, _ := json.Marshal(pending)
	cache.data[pendingKey] = pendingData

	eventBus.On("Publish", "notification.digest_created", mock.Anything).Return()

	worker.Tick()

	// Pending queue should be cleared
	_, exists := cache.data[pendingKey]
	assert.False(t, exists)

	eventBus.AssertExpectations(t)
}

func TestDigestWorker_Tick_RespectsInterval(t *testing.T) {
	cache := &mapCache{data: make(map[string][]byte)}
	eventBus := new(MockEventBus)
	repo := new(MockNotificationRepository)
	service := NewSmartNotificationService(repo, cache, eventBus)
	worker := NewDigestWorker(service, cache, eventBus, 5*time.Minute)

	userID := uuid.New()

	// Register user
	eligibleKey := "smart_notif:digest_eligible_users"
	cache.data[eligibleKey], _ = json.Marshal([]uuid.UUID{userID})

	// Set preferences
	prefsKey := fmt.Sprintf("smart_notif:prefs:%s", userID.String())
	prefs := &models.SmartNotificationPreferences{
		UserID:             userID,
		Enabled:            true,
		DigestEnabled:      true,
		DigestIntervalMins: 60, // 1 hour
	}
	cache.data[prefsKey], _ = json.Marshal(prefs)

	// Set last digest time to now (so it's too soon)
	lastDigestKey := fmt.Sprintf("smart_notif:last_digest:%s", userID.String())
	cache.data[lastDigestKey], _ = json.Marshal(time.Now())

	// Add pending notifications
	pending := []models.SmartNotification{
		{Notification: models.Notification{ID: uuid.New()}, Category: models.NotifCategoryReaction},
	}
	pendingKey := fmt.Sprintf("smart_notif:pending_digest:%s", userID.String())
	cache.data[pendingKey], _ = json.Marshal(pending)

	worker.Tick()

	// Pending should still be there since interval hasn't elapsed
	_, exists := cache.data[pendingKey]
	assert.True(t, exists)
}

func TestDigestWorker_Tick_SkipsDisabledDigest(t *testing.T) {
	cache := &mapCache{data: make(map[string][]byte)}
	eventBus := new(MockEventBus)
	repo := new(MockNotificationRepository)
	service := NewSmartNotificationService(repo, cache, eventBus)
	worker := NewDigestWorker(service, cache, eventBus, 5*time.Minute)

	userID := uuid.New()

	cache.data["smart_notif:digest_eligible_users"], _ = json.Marshal([]uuid.UUID{userID})

	// Digest disabled in preferences
	prefsKey := fmt.Sprintf("smart_notif:prefs:%s", userID.String())
	prefs := &models.SmartNotificationPreferences{
		UserID:        userID,
		Enabled:       true,
		DigestEnabled: false,
	}
	cache.data[prefsKey], _ = json.Marshal(prefs)

	// Add pending notifications
	pending := []models.SmartNotification{
		{Notification: models.Notification{ID: uuid.New()}, Category: models.NotifCategoryReaction},
	}
	pendingKey := fmt.Sprintf("smart_notif:pending_digest:%s", userID.String())
	cache.data[pendingKey], _ = json.Marshal(pending)

	worker.Tick()

	// Pending should still be there since digest is disabled
	_, exists := cache.data[pendingKey]
	assert.True(t, exists)
}

func TestDigestWorker_RegisterEligibleUser(t *testing.T) {
	cache := &mapCache{data: make(map[string][]byte)}
	eventBus := new(MockEventBus)
	repo := new(MockNotificationRepository)
	service := NewSmartNotificationService(repo, cache, eventBus)
	worker := NewDigestWorker(service, cache, eventBus, 5*time.Minute)

	ctx := context.Background()
	userID := uuid.New()

	err := worker.RegisterEligibleUser(ctx, userID)
	assert.NoError(t, err)

	// Register again (should be no-op)
	err = worker.RegisterEligibleUser(ctx, userID)
	assert.NoError(t, err)

	// Verify only one entry
	data := cache.data["smart_notif:digest_eligible_users"]
	var ids []uuid.UUID
	json.Unmarshal(data, &ids)
	assert.Len(t, ids, 1)
}

func TestDigestWorker_UnregisterEligibleUser(t *testing.T) {
	cache := &mapCache{data: make(map[string][]byte)}
	eventBus := new(MockEventBus)
	repo := new(MockNotificationRepository)
	service := NewSmartNotificationService(repo, cache, eventBus)
	worker := NewDigestWorker(service, cache, eventBus, 5*time.Minute)

	ctx := context.Background()
	user1 := uuid.New()
	user2 := uuid.New()

	_ = worker.RegisterEligibleUser(ctx, user1)
	_ = worker.RegisterEligibleUser(ctx, user2)

	err := worker.UnregisterEligibleUser(ctx, user1)
	assert.NoError(t, err)

	data := cache.data["smart_notif:digest_eligible_users"]
	var ids []uuid.UUID
	json.Unmarshal(data, &ids)
	assert.Len(t, ids, 1)
	assert.Equal(t, user2, ids[0])
}
