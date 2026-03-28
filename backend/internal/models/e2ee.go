package models

import (
	"time"

	"github.com/google/uuid"
)

// DeviceKey represents an E2EE device identity key
type DeviceKey struct {
	ID             uuid.UUID `json:"id" db:"id"`
	UserID         uuid.UUID `json:"user_id" db:"user_id"`
	DeviceID       string    `json:"device_id" db:"device_id"`
	IdentityKey    []byte    `json:"identity_key" db:"identity_key"`
	RegistrationID int       `json:"registration_id" db:"registration_id"`
	DeviceName     *string   `json:"device_name,omitempty" db:"device_name"`
	DeviceType     string    `json:"device_type" db:"device_type"`
	LastSeen       time.Time `json:"last_seen" db:"last_seen"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// E2EE DeviceType constants (different from session DeviceType)
const (
	E2EEDeviceTypeWeb           = "web"
	E2EEDeviceTypeDesktop       = "desktop"
	E2EEDeviceTypeMobileIOS     = "mobile_ios"
	E2EEDeviceTypeMobileAndroid = "mobile_android"
	E2EEDeviceTypeUnknown       = "unknown"
)

// SignedPreKey represents a signed pre-key for X3DH key exchange
type SignedPreKey struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	DeviceKeyID uuid.UUID  `json:"device_key_id" db:"device_keys_id"`
	KeyID       int        `json:"key_id" db:"key_id"`
	PublicKey   []byte     `json:"public_key" db:"public_key"`
	Signature   []byte     `json:"signature" db:"signature"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty" db:"expires_at"`
}

// OneTimePreKey represents a single-use pre-key for session establishment
type OneTimePreKey struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	DeviceKeyID     uuid.UUID  `json:"device_key_id" db:"device_keys_id"`
	KeyID           int        `json:"key_id" db:"key_id"`
	PublicKey       []byte     `json:"public_key" db:"public_key"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	ClaimedAt       *time.Time `json:"claimed_at,omitempty" db:"claimed_at"`
	ClaimedByUserID *uuid.UUID `json:"claimed_by_user_id,omitempty" db:"claimed_by_user_id"`
}

// E2EEPreKeyBundle is the complete set of keys needed to establish an E2EE session
type E2EEPreKeyBundle struct {
	UserID         uuid.UUID `json:"user_id"`
	DeviceID       string    `json:"device_id"`
	RegistrationID int       `json:"registration_id"`
	IdentityKey    []byte    `json:"identity_key"`
	SignedPreKeyID int       `json:"signed_pre_key_id"`
	SignedPreKey   []byte    `json:"signed_pre_key"`
	SignedKeySign  []byte    `json:"signed_key_signature"`
	PreKeyID       *int      `json:"pre_key_id,omitempty"`
	PreKey         []byte    `json:"pre_key,omitempty"`
}

// KeyUploadRequest represents a request to upload device keys
type KeyUploadRequest struct {
	DeviceID       string           `json:"device_id" validate:"required,min=1,max=64"`
	DeviceName     string           `json:"device_name,omitempty" validate:"max=64"`
	DeviceType     string           `json:"device_type,omitempty" validate:"oneof=web desktop mobile_ios mobile_android unknown"`
	IdentityKey    []byte           `json:"identity_key" validate:"required,min=32,max=64"`
	RegistrationID int              `json:"registration_id" validate:"required,min=1"`
	SignedPreKey   SignedPreKeyData `json:"signed_pre_key" validate:"required"`
	OneTimePreKeys []PreKeyData     `json:"one_time_pre_keys,omitempty" validate:"max=100,dive"`
}

// SignedPreKeyData represents signed pre-key data in requests
type SignedPreKeyData struct {
	KeyID     int    `json:"key_id" validate:"required,min=0"`
	PublicKey []byte `json:"public_key" validate:"required,min=32,max=64"`
	Signature []byte `json:"signature" validate:"required,min=64,max=128"`
}

// PreKeyData represents one-time pre-key data in requests
type PreKeyData struct {
	KeyID     int    `json:"key_id" validate:"required,min=0"`
	PublicKey []byte `json:"public_key" validate:"required,min=32,max=64"`
}

// E2EEDeviceInfo represents public info about a user's E2EE-enabled device
type E2EEDeviceInfo struct {
	DeviceID         string    `json:"device_id"`
	DeviceName       *string   `json:"device_name,omitempty"`
	DeviceType       string    `json:"device_type"`
	LastSeen         time.Time `json:"last_seen"`
	CreatedAt        time.Time `json:"created_at"`
	HasPreKeys       bool      `json:"has_pre_keys"`
	RemainingPreKeys int       `json:"remaining_pre_keys"`
}

// PreKeyCount represents the count of available pre-keys for a device
type PreKeyCount struct {
	DeviceID           string `json:"device_id"`
	SignedPreKeys      int    `json:"signed_pre_keys"`
	OneTimePreKeys     int    `json:"one_time_pre_keys"`
	MinRecommended     int    `json:"min_recommended"`
	NeedsReplenishment bool   `json:"needs_replenishment"`
}

// Constants for key management
const (
	// MinOneTimePreKeys is the minimum recommended one-time prekeys per device
	MinOneTimePreKeys = 10

	// DefaultOneTimePreKeyCount is the default number of prekeys to upload
	DefaultOneTimePreKeyCount = 100

	// MaxOneTimePreKeysPerUpload limits batch uploads
	MaxOneTimePreKeysPerUpload = 100

	// SignedPreKeyRotationDays is how often signed prekeys should be rotated
	SignedPreKeyRotationDays = 7
)

// E2EECapabilities indicates what E2EE features a user/device supports
type E2EECapabilities struct {
	SupportsE2EE      bool `json:"supports_e2ee"`
	SupportsGroupE2EE bool `json:"supports_group_e2ee"`
	ProtocolVersion   int  `json:"protocol_version"`
}
