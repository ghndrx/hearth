package models

import (
	"time"

	"github.com/google/uuid"
)

// AIProviderType represents supported AI provider types
type AIProviderType string

const (
	// Cloud providers
	AIProviderOpenRouter AIProviderType = "openrouter"
	AIProviderBedrock    AIProviderType = "bedrock"
	AIProviderVertexAI   AIProviderType = "vertex_ai"
	AIProviderOpenAI     AIProviderType = "openai"
	AIProviderAnthropic  AIProviderType = "anthropic"

	// Local providers
	AIProviderOllama   AIProviderType = "ollama"
	AIProviderLlamaCpp AIProviderType = "llama_cpp"
	AIProviderLMStudio AIProviderType = "lm_studio"
	AIProviderVLLM     AIProviderType = "vllm"
	AIProviderLocalAI  AIProviderType = "local_ai"
)

// IsCloud returns true if the provider is a cloud provider
func (p AIProviderType) IsCloud() bool {
	switch p {
	case AIProviderOpenRouter, AIProviderBedrock, AIProviderVertexAI, AIProviderOpenAI, AIProviderAnthropic:
		return true
	default:
		return false
	}
}

// IsLocal returns true if the provider is a local provider
func (p AIProviderType) IsLocal() bool {
	return !p.IsCloud()
}

// Valid returns true if the provider type is valid
func (p AIProviderType) Valid() bool {
	switch p {
	case AIProviderOpenRouter, AIProviderBedrock, AIProviderVertexAI, AIProviderOpenAI, AIProviderAnthropic,
		AIProviderOllama, AIProviderLlamaCpp, AIProviderLMStudio, AIProviderVLLM, AIProviderLocalAI:
		return true
	default:
		return false
	}
}

// AIFeatureType represents AI feature types for model routing
type AIFeatureType string

const (
	AIFeatureSummary   AIFeatureType = "summary"
	AIFeatureSearch    AIFeatureType = "search"
	AIFeatureChat      AIFeatureType = "chat"
	AIFeatureEmbed     AIFeatureType = "embed"
	AIFeatureModerate  AIFeatureType = "moderate"
	AIFeatureTranslate AIFeatureType = "translate"
)

// Valid returns true if the feature type is valid
func (f AIFeatureType) Valid() bool {
	switch f {
	case AIFeatureSummary, AIFeatureSearch, AIFeatureChat, AIFeatureEmbed, AIFeatureModerate, AIFeatureTranslate:
		return true
	default:
		return false
	}
}

// AIProviderConfig represents a configured AI provider (database model)
type AIProviderConfig struct {
	ID           uuid.UUID      `json:"id" db:"id"`
	ProviderType AIProviderType `json:"provider_type" db:"provider_type"`
	Name         string         `json:"name" db:"name"`
	DisplayName  string         `json:"display_name" db:"display_name"`
	BaseURL      *string        `json:"base_url,omitempty" db:"base_url"`
	IsEnabled    bool           `json:"is_enabled" db:"is_enabled"`
	IsDefault    bool           `json:"is_default" db:"is_default"`
	Priority     int            `json:"priority" db:"priority"`
	Config       *string        `json:"-" db:"config"` // Encrypted JSON credentials
	CreatedAt    time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at" db:"updated_at"`
}

// UserAICredential represents user-specific AI provider credentials
type UserAICredential struct {
	ID           uuid.UUID      `json:"id" db:"id"`
	UserID       uuid.UUID      `json:"user_id" db:"user_id"`
	ProviderID   uuid.UUID      `json:"provider_id" db:"provider_id"`
	ProviderType AIProviderType `json:"provider_type" db:"provider_type"`
	Credentials  string         `json:"-" db:"credentials"` // Encrypted JSON credentials
	IsEnabled    bool           `json:"is_enabled" db:"is_enabled"`
	LastUsedAt   *time.Time     `json:"last_used_at,omitempty" db:"last_used_at"`
	CreatedAt    time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at" db:"updated_at"`
}

// AIModelRouting represents per-feature model routing configuration
type AIModelRouting struct {
	ID         uuid.UUID     `json:"id" db:"id"`
	ServerID   *uuid.UUID    `json:"server_id,omitempty" db:"server_id"`
	UserID     *uuid.UUID    `json:"user_id,omitempty" db:"user_id"`
	Feature    AIFeatureType `json:"feature" db:"feature"`
	ProviderID uuid.UUID     `json:"provider_id" db:"provider_id"`
	ModelID    string        `json:"model_id" db:"model_id"`
	Priority   int           `json:"priority" db:"priority"`
	IsEnabled  bool          `json:"is_enabled" db:"is_enabled"`
	CreatedAt  time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at" db:"updated_at"`
}

// AIUsageLog tracks AI API usage for billing/quotas
type AIUsageLog struct {
	ID           uuid.UUID      `json:"id" db:"id"`
	UserID       uuid.UUID      `json:"user_id" db:"user_id"`
	ServerID     *uuid.UUID     `json:"server_id,omitempty" db:"server_id"`
	ProviderID   uuid.UUID      `json:"provider_id" db:"provider_id"`
	ProviderType AIProviderType `json:"provider_type" db:"provider_type"`
	ModelID      string         `json:"model_id" db:"model_id"`
	Feature      AIFeatureType  `json:"feature" db:"feature"`
	InputTokens  int            `json:"input_tokens" db:"input_tokens"`
	OutputTokens int            `json:"output_tokens" db:"output_tokens"`
	TotalTokens  int            `json:"total_tokens" db:"total_tokens"`
	DurationMs   int64          `json:"duration_ms" db:"duration_ms"`
	Success      bool           `json:"success" db:"success"`
	ErrorMessage *string        `json:"error_message,omitempty" db:"error_message"`
	RequestID    string         `json:"request_id" db:"request_id"`
	CreatedAt    time.Time      `json:"created_at" db:"created_at"`
}

// AIModelInfo represents information about an available model
type AIModelInfo struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	ProviderType  AIProviderType `json:"provider_type"`
	ProviderID    uuid.UUID      `json:"provider_id"`
	Description   string         `json:"description,omitempty"`
	ContextWindow int            `json:"context_window,omitempty"`
	MaxTokens     int            `json:"max_tokens,omitempty"`
	InputCost     float64        `json:"input_cost,omitempty"`
	OutputCost    float64        `json:"output_cost,omitempty"`
	Capabilities  []string       `json:"capabilities,omitempty"`
}

// AIChatMessage represents a message in a chat conversation
type AIChatMessage struct {
	Role       string                 `json:"role"` // system, user, assistant, tool
	Content    string                 `json:"content"`
	Name       string                 `json:"name,omitempty"`
	ToolCallID string                 `json:"tool_call_id,omitempty"`
	ToolCalls  []AIToolCall           `json:"tool_calls,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// AIToolCall represents a tool/function call
type AIToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// AIChatRequest represents a chat completion request
type AIChatRequest struct {
	Model       string          `json:"model,omitempty"`
	Feature     string          `json:"feature,omitempty"`
	Messages    []AIChatMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature *float64        `json:"temperature,omitempty"`
	TopP        *float64        `json:"top_p,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
	ServerID    *string         `json:"server_id,omitempty"`
}

// AIChatResponse represents a chat completion response
type AIChatResponse struct {
	ID           string   `json:"id"`
	Model        string   `json:"model"`
	Created      int64    `json:"created"`
	Content      string   `json:"content"`
	FinishReason string   `json:"finish_reason,omitempty"`
	Usage        *AIUsage `json:"usage,omitempty"`
}

// AIUsage represents token usage
type AIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// AIEmbeddingRequest represents an embedding request
type AIEmbeddingRequest struct {
	Model    string   `json:"model,omitempty"`
	Input    []string `json:"input"`
	ServerID *string  `json:"server_id,omitempty"`
}

// AIEmbeddingResponse represents an embedding response
type AIEmbeddingResponse struct {
	Model string        `json:"model"`
	Data  []AIEmbedding `json:"data"`
	Usage *AIUsage      `json:"usage,omitempty"`
}

// AIEmbedding represents a single embedding
type AIEmbedding struct {
	Index     int       `json:"index"`
	Embedding []float32 `json:"embedding"`
}

// AIProviderHealth represents provider health status
type AIProviderHealth struct {
	Name      string        `json:"name"`
	Available bool          `json:"available"`
	Latency   time.Duration `json:"latency"`
	Error     string        `json:"error,omitempty"`
	CheckedAt time.Time     `json:"checked_at"`
}
