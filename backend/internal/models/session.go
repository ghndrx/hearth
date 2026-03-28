package models

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
)

// DeviceType represents the type of device
type DeviceType string

const (
	DeviceTypeDesktop DeviceType = "desktop"
	DeviceTypeMobile  DeviceType = "mobile"
	DeviceTypeTablet  DeviceType = "tablet"
	DeviceTypeUnknown DeviceType = "unknown"
)

// Session represents an authenticated user session with device info
type Session struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	UserID          uuid.UUID  `json:"user_id" db:"user_id"`
	TokenHash       string     `json:"-" db:"token_hash"`
	Device          *string    `json:"device,omitempty" db:"device"`
	DeviceName      *string    `json:"device_name,omitempty" db:"device_name"`
	DeviceType      DeviceType `json:"device_type" db:"device_type"`
	Browser         *string    `json:"browser,omitempty" db:"browser"`
	BrowserVersion  *string    `json:"browser_version,omitempty" db:"browser_version"`
	OS              *string    `json:"os,omitempty" db:"os"`
	OSVersion       *string    `json:"os_version,omitempty" db:"os_version"`
	IPAddress       *string    `json:"-" db:"ip_address"`
	UserAgent       *string    `json:"-" db:"user_agent"`
	LocationCity    *string    `json:"location_city,omitempty" db:"location_city"`
	LocationCountry *string    `json:"location_country,omitempty" db:"location_country"`
	IsCurrent       bool       `json:"is_current" db:"is_current"`
	LastUsed        *time.Time `json:"last_used,omitempty" db:"last_used"`
	ExpiresAt       time.Time  `json:"expires_at" db:"expires_at"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
}

// SessionResponse is the API response format for sessions
type SessionResponse struct {
	ID              uuid.UUID  `json:"id"`
	DeviceName      string     `json:"device_name"`
	DeviceType      DeviceType `json:"device_type"`
	Browser         string     `json:"browser,omitempty"`
	BrowserVersion  string     `json:"browser_version,omitempty"`
	OS              string     `json:"os,omitempty"`
	OSVersion       string     `json:"os_version,omitempty"`
	LocationCity    string     `json:"location_city,omitempty"`
	LocationCountry string     `json:"location_country,omitempty"`
	IsCurrent       bool       `json:"is_current"`
	LastUsed        *time.Time `json:"last_used,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// ToResponse converts Session to SessionResponse
func (s *Session) ToResponse() SessionResponse {
	response := SessionResponse{
		ID:         s.ID,
		DeviceType: s.DeviceType,
		IsCurrent:  s.IsCurrent,
		LastUsed:   s.LastUsed,
		CreatedAt:  s.CreatedAt,
	}

	// Build device name if not set
	if s.DeviceName != nil {
		response.DeviceName = *s.DeviceName
	} else {
		response.DeviceName = "Unknown Device"
		if s.Browser != nil && s.OS != nil {
			response.DeviceName = *s.Browser + " on " + *s.OS
		}
	}

	if s.Browser != nil {
		response.Browser = *s.Browser
	}
	if s.BrowserVersion != nil {
		response.BrowserVersion = *s.BrowserVersion
	}
	if s.OS != nil {
		response.OS = *s.OS
	}
	if s.OSVersion != nil {
		response.OSVersion = *s.OSVersion
	}
	if s.LocationCity != nil {
		response.LocationCity = *s.LocationCity
	}
	if s.LocationCountry != nil {
		response.LocationCountry = *s.LocationCountry
	}

	return response
}

// RefreshToken represents a refresh token with rotation support
type RefreshToken struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	UserID    uuid.UUID  `json:"user_id" db:"user_id"`
	TokenHash string     `json:"-" db:"token_hash"`
	FamilyID  uuid.UUID  `json:"family_id" db:"family_id"`
	SessionID uuid.UUID  `json:"session_id" db:"session_id"`
	Used      bool       `json:"used" db:"used"`
	UsedAt    *time.Time `json:"used_at,omitempty" db:"used_at"`
	Revoked   bool       `json:"revoked" db:"revoked"`
	RevokedAt *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`
	ExpiresAt time.Time  `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}

// IsValid checks if the refresh token is still valid
func (rt *RefreshToken) IsValid() bool {
	return !rt.Used && !rt.Revoked && time.Now().Before(rt.ExpiresAt)
}

// DeviceInfo holds parsed device information
type DeviceInfo struct {
	DeviceName     string
	DeviceType     DeviceType
	Browser        string
	BrowserVersion string
	OS             string
	OSVersion      string
}

// HashToken creates a SHA-256 hash of a token
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// GenerateTokenFamily generates a new token family ID
func GenerateTokenFamily() uuid.UUID {
	return uuid.New()
}
