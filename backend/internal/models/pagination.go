package models

import (
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// PaginationDefaults
const (
	DefaultPageLimit = 50
	MaxPageLimit     = 1000
)

// MemberCursor represents a cursor for member pagination
// Based on joined_at (DESC) and user_id for tie-breaking
type MemberCursor struct {
	JoinedAt time.Time `json:"j"`
	UserID   uuid.UUID `json:"u"`
}

// Encode encodes the cursor to a base64 string
func (c *MemberCursor) Encode() string {
	if c == nil {
		return ""
	}
	data, _ := json.Marshal(c)
	return base64.URLEncoding.EncodeToString(data)
}

// DecodeMemberCursor decodes a base64 cursor string
func DecodeMemberCursor(cursor string) (*MemberCursor, error) {
	if cursor == "" {
		return nil, nil
	}
	data, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, err
	}
	var c MemberCursor
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// PaginatedMembers represents a paginated response of members
type PaginatedMembers struct {
	Members    []*Member `json:"members"`
	NextCursor string    `json:"next_cursor,omitempty"`
	HasMore    bool      `json:"has_more"`
}

// PresenceCursor represents a cursor for presence pagination
// Based on user_id only since presence doesn't have natural ordering
type PresenceCursor struct {
	UserID uuid.UUID `json:"u"`
}

// Encode encodes the cursor to a base64 string
func (c *PresenceCursor) Encode() string {
	if c == nil {
		return ""
	}
	data, _ := json.Marshal(c)
	return base64.URLEncoding.EncodeToString(data)
}

// DecodePresenceCursor decodes a base64 cursor string
func DecodePresenceCursor(cursor string) (*PresenceCursor, error) {
	if cursor == "" {
		return nil, nil
	}
	data, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, err
	}
	var c PresenceCursor
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// PaginatedPresences represents a paginated response of presences
type PaginatedPresences struct {
	Presences  map[uuid.UUID]*Presence `json:"presences"`
	NextCursor string                  `json:"next_cursor,omitempty"`
	HasMore    bool                    `json:"has_more"`
}

// NormalizeLimit ensures limit is within valid bounds
func NormalizeLimit(limit int) int {
	if limit <= 0 {
		return DefaultPageLimit
	}
	if limit > MaxPageLimit {
		return MaxPageLimit
	}
	return limit
}
