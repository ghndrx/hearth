package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"hearth/internal/models"
)

// E2EEServiceImpl handles E2EE key management operations
type E2EEServiceImpl struct {
	repo E2EERepository
}

// NewE2EEServiceImpl creates a new E2EE service
func NewE2EEServiceImpl(repo E2EERepository) *E2EEServiceImpl {
	return &E2EEServiceImpl{repo: repo}
}

// Common errors
var (
	ErrInvalidIdentityKey  = errors.New("invalid identity key")
	ErrInvalidSignedPreKey = errors.New("invalid signed prekey")
	ErrInvalidPreKey       = errors.New("invalid prekey")
	ErrDeviceNotFound      = errors.New("device not found")
	ErrSelfKeyRequest      = errors.New("cannot request prekey bundle for own device")
)

// RegisterDevice registers a new device and uploads its keys
func (s *E2EEServiceImpl) RegisterDevice(ctx context.Context, userID uuid.UUID, req *models.KeyUploadRequest) (*models.DeviceKey, error) {
	// Validate identity key (should be 32 or 33 bytes for Curve25519 or P-256)
	if len(req.IdentityKey) < 32 || len(req.IdentityKey) > 65 {
		return nil, ErrInvalidIdentityKey
	}

	// Validate signed prekey
	if len(req.SignedPreKey.PublicKey) < 32 || len(req.SignedPreKey.PublicKey) > 65 {
		return nil, ErrInvalidSignedPreKey
	}
	if len(req.SignedPreKey.Signature) < 64 {
		return nil, ErrInvalidSignedPreKey
	}

	// Validate one-time prekeys
	for i, pk := range req.OneTimePreKeys {
		if len(pk.PublicKey) < 32 || len(pk.PublicKey) > 65 {
			return nil, fmt.Errorf("%w at index %d", ErrInvalidPreKey, i)
		}
	}

	// Set default device type
	if req.DeviceType == "" {
		req.DeviceType = models.E2EEDeviceTypeUnknown
	}

	return s.repo.RegisterDevice(ctx, userID, req)
}

// GetUserDevices returns all devices for a user
func (s *E2EEServiceImpl) GetUserDevices(ctx context.Context, userID uuid.UUID) ([]*models.E2EEDeviceInfo, error) {
	return s.repo.GetUserDevices(ctx, userID)
}

// GetPreKeyBundle retrieves a prekey bundle for establishing an E2EE session
// This should be called when initiating a DM or adding someone to an encrypted conversation
func (s *E2EEServiceImpl) GetPreKeyBundle(ctx context.Context, targetUserID uuid.UUID, targetDeviceID string, requestingUserID uuid.UUID) (*models.E2EEPreKeyBundle, error) {
	// Prevent requesting your own prekey bundle (would waste prekeys)
	if targetUserID == requestingUserID {
		return nil, ErrSelfKeyRequest
	}

	bundle, err := s.repo.GetPreKeyBundle(ctx, targetUserID, targetDeviceID, requestingUserID)
	if errors.Is(err, ErrDeviceNotFound) {
		return nil, ErrDeviceNotFound
	}
	return bundle, err
}

// GetPreKeyBundlesForUser retrieves prekey bundles for all of a user's devices
func (s *E2EEServiceImpl) GetPreKeyBundlesForUser(ctx context.Context, targetUserID uuid.UUID, requestingUserID uuid.UUID) ([]*models.E2EEPreKeyBundle, error) {
	if targetUserID == requestingUserID {
		return nil, ErrSelfKeyRequest
	}

	// Get all devices
	devices, err := s.repo.GetUserDevices(ctx, targetUserID)
	if err != nil {
		return nil, err
	}

	var bundles []*models.E2EEPreKeyBundle
	for _, device := range devices {
		if !device.HasPreKeys {
			continue
		}

		bundle, err := s.repo.GetPreKeyBundle(ctx, targetUserID, device.DeviceID, requestingUserID)
		if err != nil {
			// Log but continue - some devices may not have prekeys
			continue
		}
		bundles = append(bundles, bundle)
	}

	return bundles, nil
}

// UpdateLastSeen updates the last seen timestamp for a device
func (s *E2EEServiceImpl) UpdateLastSeen(ctx context.Context, userID uuid.UUID, deviceID string) error {
	err := s.repo.UpdateLastSeen(ctx, userID, deviceID)
	if errors.Is(err, ErrDeviceNotFound) {
		return ErrDeviceNotFound
	}
	return err
}

// DeleteDevice removes a device and all its keys
func (s *E2EEServiceImpl) DeleteDevice(ctx context.Context, userID uuid.UUID, deviceID string) error {
	err := s.repo.DeleteDevice(ctx, userID, deviceID)
	if errors.Is(err, ErrDeviceNotFound) {
		return ErrDeviceNotFound
	}
	return err
}

// GetPreKeyCount returns the count of available prekeys
func (s *E2EEServiceImpl) GetPreKeyCount(ctx context.Context, userID uuid.UUID, deviceID string) (*models.PreKeyCount, error) {
	count, err := s.repo.GetPreKeyCount(ctx, userID, deviceID)
	if errors.Is(err, ErrDeviceNotFound) {
		return nil, ErrDeviceNotFound
	}
	return count, err
}

// UploadPreKeys adds more one-time prekeys to a device
func (s *E2EEServiceImpl) UploadPreKeys(ctx context.Context, userID uuid.UUID, deviceID string, prekeys []models.PreKeyData) error {
	// Validate prekeys
	for i, pk := range prekeys {
		if len(pk.PublicKey) < 32 || len(pk.PublicKey) > 65 {
			return fmt.Errorf("%w at index %d", ErrInvalidPreKey, i)
		}
	}

	err := s.repo.UploadPreKeys(ctx, userID, deviceID, prekeys)
	if errors.Is(err, ErrDeviceNotFound) {
		return ErrDeviceNotFound
	}
	return err
}

// CheckAndNotifyLowPreKeys checks if prekeys are running low and returns whether replenishment is needed
func (s *E2EEServiceImpl) CheckAndNotifyLowPreKeys(ctx context.Context, userID uuid.UUID, deviceID string) (bool, error) {
	count, err := s.GetPreKeyCount(ctx, userID, deviceID)
	if err != nil {
		return false, err
	}
	return count.NeedsReplenishment, nil
}

// GetE2EECapabilities returns the E2EE capabilities for a user
// This checks if the user has any devices with E2EE keys registered
func (s *E2EEServiceImpl) GetE2EECapabilities(ctx context.Context, userID uuid.UUID) (*models.E2EECapabilities, error) {
	devices, err := s.repo.GetUserDevices(ctx, userID)
	if err != nil {
		return nil, err
	}

	caps := &models.E2EECapabilities{
		ProtocolVersion: 1, // Signal Protocol X3DH
	}

	for _, d := range devices {
		if d.HasPreKeys {
			caps.SupportsE2EE = true
			break
		}
	}

	return caps, nil
}
