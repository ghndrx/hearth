package postgres

import (
	"context"
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hearth/internal/models"
)

func TestE2EERepository_RegisterDevice(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	repo := NewE2EERepository(db)
	userID := createTestUser(t, db)
	ctx := context.Background()

	t.Run("register new device", func(t *testing.T) {
		req := &models.KeyUploadRequest{
			DeviceID:       "test-device-1",
			DeviceName:     "Test Browser",
			DeviceType:     models.E2EEDeviceTypeWeb,
			IdentityKey:    make([]byte, 33), // P-256 public key size
			RegistrationID: 12345,
			SignedPreKey: models.SignedPreKeyData{
				KeyID:     1,
				PublicKey: make([]byte, 33),
				Signature: make([]byte, 64),
			},
			OneTimePreKeys: []models.PreKeyData{
				{KeyID: 1, PublicKey: make([]byte, 33)},
				{KeyID: 2, PublicKey: make([]byte, 33)},
				{KeyID: 3, PublicKey: make([]byte, 33)},
			},
		}

		device, err := repo.RegisterDevice(ctx, userID, req)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, device.ID)
		assert.Equal(t, userID, device.UserID)
		assert.Equal(t, "test-device-1", device.DeviceID)
		assert.Equal(t, models.E2EEDeviceTypeWeb, device.DeviceType)
	})

	t.Run("update existing device", func(t *testing.T) {
		req := &models.KeyUploadRequest{
			DeviceID:       "test-device-1", // Same device ID
			DeviceName:     "Updated Browser",
			DeviceType:     models.E2EEDeviceTypeWeb,
			IdentityKey:    make([]byte, 33),
			RegistrationID: 54321, // New registration ID
			SignedPreKey: models.SignedPreKeyData{
				KeyID:     2, // New signed prekey
				PublicKey: make([]byte, 33),
				Signature: make([]byte, 64),
			},
		}

		device, err := repo.RegisterDevice(ctx, userID, req)
		require.NoError(t, err)
		assert.Equal(t, "test-device-1", device.DeviceID)
		assert.Equal(t, 54321, device.RegistrationID)
	})
}

func TestE2EERepository_GetUserDevices(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	repo := NewE2EERepository(db)
	userID := createTestUser(t, db)
	ctx := context.Background()

	// Register two devices
	for i := 1; i <= 2; i++ {
		req := &models.KeyUploadRequest{
			DeviceID:       uuid.New().String(),
			DeviceType:     models.E2EEDeviceTypeWeb,
			IdentityKey:    make([]byte, 33),
			RegistrationID: 10000 + i,
			SignedPreKey: models.SignedPreKeyData{
				KeyID:     1,
				PublicKey: make([]byte, 33),
				Signature: make([]byte, 64),
			},
			OneTimePreKeys: []models.PreKeyData{
				{KeyID: 1, PublicKey: make([]byte, 33)},
			},
		}
		_, err := repo.RegisterDevice(ctx, userID, req)
		require.NoError(t, err)
	}

	devices, err := repo.GetUserDevices(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, devices, 2)

	for _, d := range devices {
		assert.True(t, d.HasPreKeys)
		assert.Equal(t, 1, d.RemainingPreKeys)
	}
}

func TestE2EERepository_GetPreKeyBundle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	repo := NewE2EERepository(db)
	userID := createTestUser(t, db)
	claimingUserID := createTestUser(t, db)
	ctx := context.Background()

	deviceID := "bundle-test-device"

	// Register device with prekeys
	req := &models.KeyUploadRequest{
		DeviceID:       deviceID,
		DeviceType:     models.E2EEDeviceTypeWeb,
		IdentityKey:    []byte("identity-key-32-bytes-padding!!"),
		RegistrationID: 99999,
		SignedPreKey: models.SignedPreKeyData{
			KeyID:     42,
			PublicKey: []byte("signed-prekey-32-bytes-padding!"),
			Signature: []byte("signature-64-bytes-padding-needs-to-be-this-long-to-work!!!!!!!!"),
		},
		OneTimePreKeys: []models.PreKeyData{
			{KeyID: 100, PublicKey: []byte("onetime-prekey-32-bytes-padding")},
			{KeyID: 101, PublicKey: []byte("onetime-prekey-32-bytes-paddin2")},
		},
	}
	_, err := repo.RegisterDevice(ctx, userID, req)
	require.NoError(t, err)

	t.Run("get bundle with one-time prekey", func(t *testing.T) {
		bundle, err := repo.GetPreKeyBundle(ctx, userID, deviceID, claimingUserID)
		require.NoError(t, err)

		assert.Equal(t, userID, bundle.UserID)
		assert.Equal(t, deviceID, bundle.DeviceID)
		assert.Equal(t, 99999, bundle.RegistrationID)
		assert.Equal(t, 42, bundle.SignedPreKeyID)
		assert.NotNil(t, bundle.PreKeyID)
		assert.Equal(t, 100, *bundle.PreKeyID) // First one-time prekey
	})

	t.Run("prekey is consumed", func(t *testing.T) {
		// Get another bundle - should get different prekey
		bundle, err := repo.GetPreKeyBundle(ctx, userID, deviceID, claimingUserID)
		require.NoError(t, err)

		// Should get second prekey now
		assert.NotNil(t, bundle.PreKeyID)
		assert.Equal(t, 101, *bundle.PreKeyID)
	})

	t.Run("works without one-time prekeys", func(t *testing.T) {
		// Both prekeys consumed, should still work
		bundle, err := repo.GetPreKeyBundle(ctx, userID, deviceID, claimingUserID)
		require.NoError(t, err)

		// No one-time prekey, but bundle still valid
		assert.Nil(t, bundle.PreKeyID)
		assert.Equal(t, 42, bundle.SignedPreKeyID)
	})
}

func TestE2EERepository_DeleteDevice(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	repo := NewE2EERepository(db)
	userID := createTestUser(t, db)
	ctx := context.Background()

	deviceID := "delete-test-device"

	req := &models.KeyUploadRequest{
		DeviceID:       deviceID,
		DeviceType:     models.E2EEDeviceTypeWeb,
		IdentityKey:    make([]byte, 33),
		RegistrationID: 11111,
		SignedPreKey: models.SignedPreKeyData{
			KeyID:     1,
			PublicKey: make([]byte, 33),
			Signature: make([]byte, 64),
		},
	}
	_, err := repo.RegisterDevice(ctx, userID, req)
	require.NoError(t, err)

	// Verify device exists
	_, err = repo.GetDeviceKey(ctx, userID, deviceID)
	require.NoError(t, err)

	// Delete device
	err = repo.DeleteDevice(ctx, userID, deviceID)
	require.NoError(t, err)

	// Verify device is gone
	_, err = repo.GetDeviceKey(ctx, userID, deviceID)
	assert.ErrorIs(t, err, ErrDeviceNotFound)
}

func TestE2EERepository_PreKeyCount(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	repo := NewE2EERepository(db)
	userID := createTestUser(t, db)
	ctx := context.Background()

	deviceID := "count-test-device"

	// Register with 5 prekeys
	req := &models.KeyUploadRequest{
		DeviceID:       deviceID,
		DeviceType:     models.E2EEDeviceTypeWeb,
		IdentityKey:    make([]byte, 33),
		RegistrationID: 22222,
		SignedPreKey: models.SignedPreKeyData{
			KeyID:     1,
			PublicKey: make([]byte, 33),
			Signature: make([]byte, 64),
		},
		OneTimePreKeys: make([]models.PreKeyData, 5),
	}
	for i := range req.OneTimePreKeys {
		req.OneTimePreKeys[i] = models.PreKeyData{KeyID: i, PublicKey: make([]byte, 33)}
	}
	_, err := repo.RegisterDevice(ctx, userID, req)
	require.NoError(t, err)

	count, err := repo.GetPreKeyCount(ctx, userID, deviceID)
	require.NoError(t, err)
	assert.Equal(t, 1, count.SignedPreKeys)
	assert.Equal(t, 5, count.OneTimePreKeys)
	assert.True(t, count.NeedsReplenishment) // 5 < 10 (MinOneTimePreKeys)
}

func TestE2EERepository_UploadPreKeys(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	repo := NewE2EERepository(db)
	userID := createTestUser(t, db)
	ctx := context.Background()

	deviceID := "upload-test-device"

	// Register device with no prekeys
	req := &models.KeyUploadRequest{
		DeviceID:       deviceID,
		DeviceType:     models.E2EEDeviceTypeWeb,
		IdentityKey:    make([]byte, 33),
		RegistrationID: 33333,
		SignedPreKey: models.SignedPreKeyData{
			KeyID:     1,
			PublicKey: make([]byte, 33),
			Signature: make([]byte, 64),
		},
	}
	_, err := repo.RegisterDevice(ctx, userID, req)
	require.NoError(t, err)

	// Upload additional prekeys
	newPrekeys := []models.PreKeyData{
		{KeyID: 10, PublicKey: make([]byte, 33)},
		{KeyID: 11, PublicKey: make([]byte, 33)},
		{KeyID: 12, PublicKey: make([]byte, 33)},
	}
	err = repo.UploadPreKeys(ctx, userID, deviceID, newPrekeys)
	require.NoError(t, err)

	// Verify count
	count, err := repo.GetPreKeyCount(ctx, userID, deviceID)
	require.NoError(t, err)
	assert.Equal(t, 3, count.OneTimePreKeys)
}

// Helper functions

func setupTestDB(t *testing.T) *sql.DB {
	// Use test database URL or skip if not available
	testDBURL := "postgres://hearth:hearth@localhost:5432/hearth_test?sslmode=disable"
	db, err := sql.Open("postgres", testDBURL)
	if err != nil {
		t.Skipf("Skipping test: could not connect to test database: %v", err)
	}

	if err := db.Ping(); err != nil {
		t.Skipf("Skipping test: could not ping test database: %v", err)
	}

	// Clean up E2EE tables before test
	_, _ = db.Exec(`TRUNCATE device_keys CASCADE`)

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

func createTestUser(t *testing.T, db *sql.DB) uuid.UUID {
	userID := uuid.New()
	_, err := db.Exec(`
		INSERT INTO users (id, username, discriminator, email, password_hash)
		VALUES ($1, $2, $3, $4, $5)
	`, userID, "testuser_"+userID.String()[:8], "0001", userID.String()+"@test.com", "hash")
	require.NoError(t, err)
	return userID
}
