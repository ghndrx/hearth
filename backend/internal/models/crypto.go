package models

import "github.com/google/uuid"

// PreKeyBundle contains the pre-keys needed for E2EE key exchange
type PreKeyBundle struct {
	UserID         uuid.UUID `json:"user_id" db:"user_id"`
	IdentityKey    string    `json:"identity_key" db:"identity_key"`
	SignedPreKeyID int       `json:"signed_pre_key_id" db:"signed_pre_key_id"`
	SignedPreKey   string    `json:"signed_pre_key" db:"signed_pre_key"`
	SignedKeySign  string    `json:"signed_key_signature" db:"signed_key_signature"`
	PreKeyID       int       `json:"pre_key_id" db:"pre_key_id"`
	PreKey         string    `json:"pre_key" db:"pre_key"`
}

// MLSGroup represents an MLS group for group E2EE
type MLSGroup struct {
	ChannelID  uuid.UUID `json:"channel_id" db:"channel_id"`
	GroupState []byte    `json:"-" db:"group_state"`
	Epoch      int64     `json:"epoch" db:"epoch"`
}
