package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AIConversation represents a chat conversation with an AI
type AIConversation struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	UserID        uuid.UUID  `json:"user_id" db:"user_id"`
	Title         string     `json:"title" db:"title"`
	ModelID       *string    `json:"model_id,omitempty" db:"model_id"`
	ProviderID    *uuid.UUID `json:"provider_id,omitempty" db:"provider_id"`
	SystemPrompt  *string    `json:"system_prompt,omitempty" db:"system_prompt"`
	Temperature   float32    `json:"temperature" db:"temperature"`
	MaxTokens     int        `json:"max_tokens" db:"max_tokens"`
	IsArchived    bool       `json:"is_archived" db:"is_archived"`
	IsPinned      bool       `json:"is_pinned" db:"is_pinned"`
	MessageCount  int        `json:"message_count" db:"message_count"`
	LastMessageAt *time.Time `json:"last_message_at,omitempty" db:"last_message_at"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}

// AIConversationWithMessages includes the conversation and its messages
type AIConversationWithMessages struct {
	AIConversation
	Messages []ConversationMessage `json:"messages,omitempty"`
}

// ConversationMessageRole represents the role of a conversation message
type ConversationMessageRole string

const (
	ConvRoleSystem    ConversationMessageRole = "system"
	ConvRoleUser      ConversationMessageRole = "user"
	ConvRoleAssistant ConversationMessageRole = "assistant"
	ConvRoleTool      ConversationMessageRole = "tool"
)

// Valid returns true if the role is valid
func (r ConversationMessageRole) Valid() bool {
	switch r {
	case ConvRoleSystem, ConvRoleUser, ConvRoleAssistant, ConvRoleTool:
		return true
	default:
		return false
	}
}

// ConversationToolCall represents a tool call in a conversation message
type ConversationToolCall struct {
	ID       string                       `json:"id"`
	Type     string                       `json:"type"`
	Function ConversationToolCallFunction `json:"function"`
}

// ConversationToolCallFunction represents a function call
type ConversationToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolCallsJSON is a JSON array of tool calls
type ToolCallsJSON []ConversationToolCall

// Value implements driver.Valuer for database storage
func (t ToolCallsJSON) Value() (driver.Value, error) {
	if t == nil {
		return nil, nil
	}
	return json.Marshal(t)
}

// Scan implements sql.Scanner for database retrieval
func (t *ToolCallsJSON) Scan(value interface{}) error {
	if value == nil {
		*t = nil
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(b, t)
}

// ConversationMessage represents a single message in a conversation
type ConversationMessage struct {
	ID              uuid.UUID               `json:"id" db:"id"`
	ConversationID  uuid.UUID               `json:"conversation_id" db:"conversation_id"`
	Role            ConversationMessageRole `json:"role" db:"role"`
	Content         string                  `json:"content" db:"content"`
	ToolCalls       ToolCallsJSON           `json:"tool_calls,omitempty" db:"tool_calls"`
	ToolCallID      *string                 `json:"tool_call_id,omitempty" db:"tool_call_id"`
	Name            *string                 `json:"name,omitempty" db:"name"`
	TokensUsed      *int                    `json:"tokens_used,omitempty" db:"tokens_used"`
	ModelUsed       *string                 `json:"model_used,omitempty" db:"model_used"`
	ProviderUsed    *string                 `json:"provider_used,omitempty" db:"provider_used"`
	FinishReason    *string                 `json:"finish_reason,omitempty" db:"finish_reason"`
	IsEdited        bool                    `json:"is_edited" db:"is_edited"`
	ParentMessageID *uuid.UUID              `json:"parent_message_id,omitempty" db:"parent_message_id"`
	ErrorMessage    *string                 `json:"error_message,omitempty" db:"error_message"`
	CreatedAt       time.Time               `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time               `json:"updated_at" db:"updated_at"`
}

// ToAPIMessage converts to the simple API message format
func (m *ConversationMessage) ToAPIMessage() AIChatMessage {
	return AIChatMessage{
		Role:    string(m.Role),
		Content: m.Content,
		Name:    stringPtrToString(m.Name),
	}
}

func stringPtrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// InitialMessagesJSON is a JSON array of initial messages for templates
type InitialMessagesJSON []struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Value implements driver.Valuer
func (i InitialMessagesJSON) Value() (driver.Value, error) {
	if i == nil {
		return nil, nil
	}
	return json.Marshal(i)
}

// Scan implements sql.Scanner
func (i *InitialMessagesJSON) Scan(value interface{}) error {
	if value == nil {
		*i = nil
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(b, i)
}

// SuggestedPromptsJSON is a JSON array of suggested prompts
type SuggestedPromptsJSON []string

// Value implements driver.Valuer
func (s SuggestedPromptsJSON) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

// Scan implements sql.Scanner
func (s *SuggestedPromptsJSON) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(b, s)
}

// AIChatTemplate represents a reusable conversation template
type AIChatTemplate struct {
	ID               uuid.UUID            `json:"id" db:"id"`
	UserID           *uuid.UUID           `json:"user_id,omitempty" db:"user_id"`
	Name             string               `json:"name" db:"name"`
	Description      *string              `json:"description,omitempty" db:"description"`
	SystemPrompt     string               `json:"system_prompt" db:"system_prompt"`
	InitialMessages  InitialMessagesJSON  `json:"initial_messages,omitempty" db:"initial_messages"`
	SuggestedPrompts SuggestedPromptsJSON `json:"suggested_prompts,omitempty" db:"suggested_prompts"`
	Icon             *string              `json:"icon,omitempty" db:"icon"`
	Category         *string              `json:"category,omitempty" db:"category"`
	IsPublic         bool                 `json:"is_public" db:"is_public"`
	UsageCount       int                  `json:"usage_count" db:"usage_count"`
	CreatedAt        time.Time            `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time            `json:"updated_at" db:"updated_at"`
}

// AIConversationShare represents a shareable link to a conversation
type AIConversationShare struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	ConversationID uuid.UUID  `json:"conversation_id" db:"conversation_id"`
	SharedBy       uuid.UUID  `json:"shared_by" db:"shared_by"`
	ShareCode      string     `json:"share_code" db:"share_code"`
	IsPublic       bool       `json:"is_public" db:"is_public"`
	CanContinue    bool       `json:"can_continue" db:"can_continue"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	ViewCount      int        `json:"view_count" db:"view_count"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
}

// CreateConversationRequest is the request body for creating a conversation
type CreateConversationRequest struct {
	Title        string  `json:"title,omitempty"`
	ModelID      *string `json:"model_id,omitempty"`
	ProviderID   *string `json:"provider_id,omitempty"`
	SystemPrompt *string `json:"system_prompt,omitempty"`
	Temperature  float32 `json:"temperature,omitempty"`
	MaxTokens    int     `json:"max_tokens,omitempty"`
	TemplateID   *string `json:"template_id,omitempty"`
}

// UpdateConversationRequest is the request body for updating a conversation
type UpdateConversationRequest struct {
	Title        *string  `json:"title,omitempty"`
	ModelID      *string  `json:"model_id,omitempty"`
	ProviderID   *string  `json:"provider_id,omitempty"`
	SystemPrompt *string  `json:"system_prompt,omitempty"`
	Temperature  *float32 `json:"temperature,omitempty"`
	MaxTokens    *int     `json:"max_tokens,omitempty"`
	IsArchived   *bool    `json:"is_archived,omitempty"`
	IsPinned     *bool    `json:"is_pinned,omitempty"`
}

// SendChatMessageRequest is the request body for sending a message
type SendChatMessageRequest struct {
	Content     string   `json:"content"`
	Stream      bool     `json:"stream,omitempty"`
	ModelID     string   `json:"model_id,omitempty"`    // Override conversation model
	Temperature *float32 `json:"temperature,omitempty"` // Override conversation temperature
	MaxTokens   *int     `json:"max_tokens,omitempty"`  // Override conversation max_tokens
}

// RegenerateMessageRequest is the request body for regenerating a message
type RegenerateMessageRequest struct {
	MessageID   string   `json:"message_id"`
	Stream      bool     `json:"stream,omitempty"`
	ModelID     string   `json:"model_id,omitempty"`
	Temperature *float32 `json:"temperature,omitempty"`
}

// ConversationListParams contains filter/pagination params for listing conversations
type ConversationListParams struct {
	UserID          uuid.UUID
	IncludeArchived bool
	OnlyPinned      bool
	Search          string
	Limit           int
	Offset          int
}

// StreamChunk represents a chunk of a streaming response
type StreamChunk struct {
	ID           string         `json:"id"`
	Object       string         `json:"object"`
	Created      int64          `json:"created"`
	Model        string         `json:"model,omitempty"`
	Choices      []StreamChoice `json:"choices"`
	FinishReason *string        `json:"finish_reason,omitempty"`
}

// StreamChoice represents a choice in a stream chunk
type StreamChoice struct {
	Index        int         `json:"index"`
	Delta        StreamDelta `json:"delta"`
	FinishReason *string     `json:"finish_reason,omitempty"`
}

// StreamDelta represents the delta content in a stream chunk
type StreamDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}
