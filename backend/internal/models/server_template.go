package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ServerTemplate represents a template created from a server
type ServerTemplate struct {
	ID             uuid.UUID       `json:"id" db:"id"`
	Code           string          `json:"code" db:"code"`
	Name           string          `json:"name" db:"name"`
	Description    string          `json:"description,omitempty" db:"description"`
	SourceServerID *uuid.UUID      `json:"source_server_id,omitempty" db:"source_server_id"`
	CreatorID      uuid.UUID       `json:"creator_id" db:"creator_id"`
	SerializedData json.RawMessage `json:"serialized_data" db:"serialized_data"`
	UsageCount     int             `json:"usage_count" db:"usage_count"`
	IsPublic       bool            `json:"is_public" db:"is_public"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at" db:"updated_at"`
}

// TemplateSerializedData contains the full server structure serialized into a template
type TemplateSerializedData struct {
	Channels []TemplateChannel `json:"channels"`
	Roles    []TemplateRole    `json:"roles"`
	Settings TemplateSettings  `json:"settings"`
}

// TemplateChannel represents a channel in a template
type TemplateChannel struct {
	Name       string      `json:"name"`
	Type       ChannelType `json:"type"`
	Topic      string      `json:"topic,omitempty"`
	Position   int         `json:"position"`
	ParentName string      `json:"parent_name,omitempty"` // For organizing channels under categories
	NSFW       bool        `json:"nsfw"`
	Slowmode   int         `json:"slowmode"`
	Bitrate    *int        `json:"bitrate,omitempty"`
	UserLimit  *int        `json:"user_limit,omitempty"`
}

// TemplateRole represents a role in a template
type TemplateRole struct {
	Name        string `json:"name"`
	Color       int    `json:"color"`
	Permissions int64  `json:"permissions"`
	Position    int    `json:"position"`
	Hoist       bool   `json:"hoist"`
	Mentionable bool   `json:"mentionable"`
}

// TemplateSettings represents server settings in a template
type TemplateSettings struct {
	VerificationLevel     int `json:"verification_level"`
	ExplicitContentFilter int `json:"explicit_content_filter"`
	DefaultNotifications  int `json:"default_notifications"`
	AFKTimeout            int `json:"afk_timeout"`
}

// CreateTemplateRequest is the request body for creating a template
type CreateTemplateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IsPublic    bool   `json:"is_public"`
}

// UpdateTemplateRequest is the request body for updating a template
type UpdateTemplateRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	IsPublic    *bool   `json:"is_public,omitempty"`
}

// TemplateListResponse is the response for listing templates
type TemplateListResponse struct {
	Templates []*ServerTemplate `json:"templates"`
	NextID    *uuid.UUID        `json:"next_id,omitempty"`
}

// ToResponse converts a ServerTemplate to its API response format
func (t *ServerTemplate) ToResponse() *ServerTemplateResponse {
	return &ServerTemplateResponse{
		ID:             t.ID,
		Code:           t.Code,
		Name:           t.Name,
		Description:    t.Description,
		SourceServerID: t.SourceServerID,
		CreatorID:      t.CreatorID,
		UsageCount:     t.UsageCount,
		IsPublic:       t.IsPublic,
		CreatedAt:      t.CreatedAt,
		UpdatedAt:      t.UpdatedAt,
	}
}

// ServerTemplateResponse is the public API response for a template
type ServerTemplateResponse struct {
	ID             uuid.UUID  `json:"id"`
	Code           string     `json:"code"`
	Name           string     `json:"name"`
	Description    string     `json:"description,omitempty"`
	SourceServerID *uuid.UUID `json:"source_server_id,omitempty"`
	CreatorID      uuid.UUID  `json:"creator_id"`
	UsageCount     int        `json:"usage_count"`
	IsPublic       bool       `json:"is_public"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
