package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"hearth/internal/models"
)

// E2EERepository handles E2EE key operations
type E2EERepository struct {
	db *sql.DB
}

// NewE2EERepository creates a new E2EE repository
func NewE2EERepository(db *sql.DB) *E2EERepository {
	return &E2EERepository{db: db}
}

// Common errors
var (
	ErrDeviceNotFound      = errors.New("device not found")
	ErrDeviceAlreadyExists = errors.New("device already exists")
	ErrNoPreKeysAvailable  = errors.New("no one-time prekeys available")
)

// RegisterDevice registers a new device with its identity key
func (r *E2EERepository) RegisterDevice(ctx context.Context, userID uuid.UUID, req *models.KeyUploadRequest) (*models.DeviceKey, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Check if device already exists
	var existingID uuid.UUID
	err = tx.QueryRowContext(ctx, `
		SELECT id FROM device_keys 
		WHERE user_id = $1 AND device_id = $2
	`, userID, req.DeviceID).Scan(&existingID)

	if err == nil {
		// Device exists - update it
		return r.updateDevice(ctx, tx, existingID, userID, req)
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("check existing device: %w", err)
	}

	// Create new device key
	deviceKey := &models.DeviceKey{
		ID:             uuid.New(),
		UserID:         userID,
		DeviceID:       req.DeviceID,
		IdentityKey:    req.IdentityKey,
		RegistrationID: req.RegistrationID,
		DeviceType:     req.DeviceType,
		LastSeen:       time.Now(),
		CreatedAt:      time.Now(),
	}
	if req.DeviceName != "" {
		deviceKey.DeviceName = &req.DeviceName
	}
	if deviceKey.DeviceType == "" {
		deviceKey.DeviceType = models.E2EEDeviceTypeUnknown
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO device_keys (id, user_id, device_id, identity_key, registration_id, device_name, device_type, last_seen, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, deviceKey.ID, deviceKey.UserID, deviceKey.DeviceID, deviceKey.IdentityKey,
		deviceKey.RegistrationID, deviceKey.DeviceName, deviceKey.DeviceType,
		deviceKey.LastSeen, deviceKey.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert device key: %w", err)
	}

	// Insert signed prekey
	_, err = tx.ExecContext(ctx, `
		INSERT INTO signed_prekeys (id, device_keys_id, key_id, public_key, signature, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, uuid.New(), deviceKey.ID, req.SignedPreKey.KeyID, req.SignedPreKey.PublicKey,
		req.SignedPreKey.Signature, time.Now())
	if err != nil {
		return nil, fmt.Errorf("insert signed prekey: %w", err)
	}

	// Insert one-time prekeys
	if len(req.OneTimePreKeys) > 0 {
		err = r.insertOneTimePreKeys(ctx, tx, deviceKey.ID, req.OneTimePreKeys)
		if err != nil {
			return nil, fmt.Errorf("insert one-time prekeys: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return deviceKey, nil
}

// updateDevice updates an existing device's keys
func (r *E2EERepository) updateDevice(ctx context.Context, tx *sql.Tx, deviceKeyID, userID uuid.UUID, req *models.KeyUploadRequest) (*models.DeviceKey, error) {
	// Update the device key
	deviceKey := &models.DeviceKey{
		ID:             deviceKeyID,
		UserID:         userID,
		DeviceID:       req.DeviceID,
		IdentityKey:    req.IdentityKey,
		RegistrationID: req.RegistrationID,
		DeviceType:     req.DeviceType,
		LastSeen:       time.Now(),
	}
	if req.DeviceName != "" {
		deviceKey.DeviceName = &req.DeviceName
	}

	_, err := tx.ExecContext(ctx, `
		UPDATE device_keys 
		SET identity_key = $1, registration_id = $2, device_name = $3, device_type = $4, last_seen = $5
		WHERE id = $6
	`, req.IdentityKey, req.RegistrationID, deviceKey.DeviceName, req.DeviceType, deviceKey.LastSeen, deviceKeyID)
	if err != nil {
		return nil, fmt.Errorf("update device key: %w", err)
	}

	// Delete old signed prekeys for this device
	_, err = tx.ExecContext(ctx, `DELETE FROM signed_prekeys WHERE device_keys_id = $1`, deviceKeyID)
	if err != nil {
		return nil, fmt.Errorf("delete old signed prekeys: %w", err)
	}

	// Insert new signed prekey
	_, err = tx.ExecContext(ctx, `
		INSERT INTO signed_prekeys (id, device_keys_id, key_id, public_key, signature, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, uuid.New(), deviceKeyID, req.SignedPreKey.KeyID, req.SignedPreKey.PublicKey,
		req.SignedPreKey.Signature, time.Now())
	if err != nil {
		return nil, fmt.Errorf("insert signed prekey: %w", err)
	}

	// Insert one-time prekeys (don't delete existing unclaimed ones)
	if len(req.OneTimePreKeys) > 0 {
		err = r.insertOneTimePreKeys(ctx, tx, deviceKeyID, req.OneTimePreKeys)
		if err != nil {
			return nil, fmt.Errorf("insert one-time prekeys: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return deviceKey, nil
}

// insertOneTimePreKeys batch inserts one-time prekeys
func (r *E2EERepository) insertOneTimePreKeys(ctx context.Context, tx *sql.Tx, deviceKeyID uuid.UUID, prekeys []models.PreKeyData) error {
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO one_time_prekeys (id, device_keys_id, key_id, public_key, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (device_keys_id, key_id) DO NOTHING
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now()
	for _, pk := range prekeys {
		_, err = stmt.ExecContext(ctx, uuid.New(), deviceKeyID, pk.KeyID, pk.PublicKey, now)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetUserDevices returns all devices for a user
func (r *E2EERepository) GetUserDevices(ctx context.Context, userID uuid.UUID) ([]*models.E2EEDeviceInfo, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT 
			dk.device_id, dk.device_name, dk.device_type, dk.last_seen, dk.created_at,
			(SELECT COUNT(*) FROM signed_prekeys WHERE device_keys_id = dk.id) > 0 as has_prekeys,
			(SELECT COUNT(*) FROM one_time_prekeys WHERE device_keys_id = dk.id AND claimed_at IS NULL)::int as remaining_prekeys
		FROM device_keys dk
		WHERE dk.user_id = $1
		ORDER BY dk.last_seen DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("query devices: %w", err)
	}
	defer rows.Close()

	var devices []*models.E2EEDeviceInfo
	for rows.Next() {
		d := &models.E2EEDeviceInfo{}
		err := rows.Scan(&d.DeviceID, &d.DeviceName, &d.DeviceType, &d.LastSeen, &d.CreatedAt, &d.HasPreKeys, &d.RemainingPreKeys)
		if err != nil {
			return nil, fmt.Errorf("scan device: %w", err)
		}
		devices = append(devices, d)
	}
	return devices, rows.Err()
}

// GetDeviceKey returns a specific device key
func (r *E2EERepository) GetDeviceKey(ctx context.Context, userID uuid.UUID, deviceID string) (*models.DeviceKey, error) {
	dk := &models.DeviceKey{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, device_id, identity_key, registration_id, device_name, device_type, last_seen, created_at
		FROM device_keys
		WHERE user_id = $1 AND device_id = $2
	`, userID, deviceID).Scan(&dk.ID, &dk.UserID, &dk.DeviceID, &dk.IdentityKey, &dk.RegistrationID,
		&dk.DeviceName, &dk.DeviceType, &dk.LastSeen, &dk.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrDeviceNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query device key: %w", err)
	}
	return dk, nil
}

// GetPreKeyBundle retrieves the full prekey bundle for establishing a session
func (r *E2EERepository) GetPreKeyBundle(ctx context.Context, userID uuid.UUID, deviceID string, claimingUserID uuid.UUID) (*models.E2EEPreKeyBundle, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get device key
	var dk models.DeviceKey
	var deviceKeyID uuid.UUID
	err = tx.QueryRowContext(ctx, `
		SELECT id, identity_key, registration_id
		FROM device_keys
		WHERE user_id = $1 AND device_id = $2
	`, userID, deviceID).Scan(&deviceKeyID, &dk.IdentityKey, &dk.RegistrationID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrDeviceNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query device key: %w", err)
	}

	// Get latest signed prekey
	var spk models.SignedPreKey
	err = tx.QueryRowContext(ctx, `
		SELECT key_id, public_key, signature
		FROM signed_prekeys
		WHERE device_keys_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, deviceKeyID).Scan(&spk.KeyID, &spk.PublicKey, &spk.Signature)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("query signed prekey: %w", err)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("no signed prekey available")
	}

	// Try to claim a one-time prekey atomically
	var opkKeyID int
	var opkPublicKey []byte
	// It's OK if no one-time prekey is available - X3DH can work without it
	_ = tx.QueryRowContext(ctx, `SELECT * FROM claim_one_time_prekey($1, $2)`, deviceKeyID, claimingUserID).Scan(&opkKeyID, &opkPublicKey)

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	bundle := &models.E2EEPreKeyBundle{
		UserID:         userID,
		DeviceID:       deviceID,
		RegistrationID: dk.RegistrationID,
		IdentityKey:    dk.IdentityKey,
		SignedPreKeyID: spk.KeyID,
		SignedPreKey:   spk.PublicKey,
		SignedKeySign:  spk.Signature,
	}

	// Add one-time prekey if available
	if opkPublicKey != nil {
		bundle.PreKeyID = &opkKeyID
		bundle.PreKey = opkPublicKey
	}

	return bundle, nil
}

// UpdateLastSeen updates the last_seen timestamp for a device
func (r *E2EERepository) UpdateLastSeen(ctx context.Context, userID uuid.UUID, deviceID string) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE device_keys SET last_seen = NOW() WHERE user_id = $1 AND device_id = $2
	`, userID, deviceID)
	if err != nil {
		return fmt.Errorf("update last_seen: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrDeviceNotFound
	}
	return nil
}

// DeleteDevice removes a device and all its keys
func (r *E2EERepository) DeleteDevice(ctx context.Context, userID uuid.UUID, deviceID string) error {
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM device_keys WHERE user_id = $1 AND device_id = $2
	`, userID, deviceID)
	if err != nil {
		return fmt.Errorf("delete device: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrDeviceNotFound
	}
	return nil
}

// GetPreKeyCount returns the count of available prekeys for a device
func (r *E2EERepository) GetPreKeyCount(ctx context.Context, userID uuid.UUID, deviceID string) (*models.PreKeyCount, error) {
	var deviceKeyID uuid.UUID
	err := r.db.QueryRowContext(ctx, `
		SELECT id FROM device_keys WHERE user_id = $1 AND device_id = $2
	`, userID, deviceID).Scan(&deviceKeyID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrDeviceNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query device: %w", err)
	}

	count := &models.PreKeyCount{
		DeviceID:       deviceID,
		MinRecommended: models.MinOneTimePreKeys,
	}

	// Count signed prekeys
	err = r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM signed_prekeys WHERE device_keys_id = $1
	`, deviceKeyID).Scan(&count.SignedPreKeys)
	if err != nil {
		return nil, fmt.Errorf("count signed prekeys: %w", err)
	}

	// Count unclaimed one-time prekeys
	err = r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM one_time_prekeys WHERE device_keys_id = $1 AND claimed_at IS NULL
	`, deviceKeyID).Scan(&count.OneTimePreKeys)
	if err != nil {
		return nil, fmt.Errorf("count one-time prekeys: %w", err)
	}

	count.NeedsReplenishment = count.OneTimePreKeys < count.MinRecommended

	return count, nil
}

// UploadPreKeys adds more one-time prekeys to a device
func (r *E2EERepository) UploadPreKeys(ctx context.Context, userID uuid.UUID, deviceID string, prekeys []models.PreKeyData) error {
	if len(prekeys) > models.MaxOneTimePreKeysPerUpload {
		return fmt.Errorf("too many prekeys: max %d", models.MaxOneTimePreKeysPerUpload)
	}

	var deviceKeyID uuid.UUID
	err := r.db.QueryRowContext(ctx, `
		SELECT id FROM device_keys WHERE user_id = $1 AND device_id = $2
	`, userID, deviceID).Scan(&deviceKeyID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrDeviceNotFound
	}
	if err != nil {
		return fmt.Errorf("query device: %w", err)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	err = r.insertOneTimePreKeys(ctx, tx, deviceKeyID, prekeys)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// CleanupExpiredPreKeys removes old claimed prekeys and expired signed prekeys
func (r *E2EERepository) CleanupExpiredPreKeys(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)

	// Delete old claimed one-time prekeys
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM one_time_prekeys WHERE claimed_at IS NOT NULL AND claimed_at < $1
	`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("delete old claimed prekeys: %w", err)
	}

	deleted, _ := result.RowsAffected()
	return deleted, nil
}
