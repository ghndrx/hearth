package ai

import (
	"time"

	"github.com/google/uuid"
)

// ProviderType represents supported AI provider types
type ProviderType string

const (
	// Cloud providers
	ProviderOpenRouter ProviderType = "openrouter"
	ProviderBedrock    ProviderType = "bedrock"
	ProviderVertexAI   ProviderType = "vertex_ai"
	ProviderOpenAI     ProviderType = "openai"
	ProviderAnthropic  ProviderType = "anthropic"

	// Local providers
	ProviderOllama   ProviderType = "ollama"
	ProviderLlamaCpp ProviderType = "llama_cpp"
	ProviderLMStudio ProviderType = "lm_studio"
	ProviderVLLM     ProviderType = "vllm"
	ProviderLocalAI  ProviderType = "local_ai"
)

// IsCloud returns true if the provider is a cloud provider
func (p ProviderType) IsCloud() bool {
	switch p {
	case ProviderOpenRouter, ProviderBedrock, ProviderVertexAI, ProviderOpenAI, ProviderAnthropic:
		return true
	default:
		return false
	}
}

// IsLocal returns true if the provider is a local provider
func (p ProviderType) IsLocal() bool {
	return !p.IsCloud()
}

// Valid returns true if the provider type is valid
func (p ProviderType) Valid() bool {
	switch p {
	case ProviderOpenRouter, ProviderBedrock, ProviderVertexAI, ProviderOpenAI, ProviderAnthropic,
		ProviderOllama, ProviderLlamaCpp, ProviderLMStudio, ProviderVLLM, ProviderLocalAI:
		return true
	default:
		return false
	}
}

// AllProviderTypes returns all valid provider types
func AllProviderTypes() []ProviderType {
	return []ProviderType{
		ProviderOpenRouter, ProviderBedrock, ProviderVertexAI, ProviderOpenAI, ProviderAnthropic,
		ProviderOllama, ProviderLlamaCpp, ProviderLMStudio, ProviderVLLM, ProviderLocalAI,
	}
}

// FeatureType represents AI feature types for model routing
type FeatureType string

const (
	FeatureSummary   FeatureType = "summary"   // Cheap models for summarization
	FeatureSearch    FeatureType = "search"    // Smart models for semantic search
	FeatureChat      FeatureType = "chat"      // General chat/completion
	FeatureEmbed     FeatureType = "embed"     // Embedding generation
	FeatureModerate  FeatureType = "moderate"  // Content moderation
	FeatureTranslate FeatureType = "translate" // Translation
)

// Valid returns true if the feature type is valid
func (f FeatureType) Valid() bool {
	switch f {
	case FeatureSummary, FeatureSearch, FeatureChat, FeatureEmbed, FeatureModerate, FeatureTranslate:
		return true
	default:
		return false
	}
}

// AllFeatureTypes returns all valid feature types
func AllFeatureTypes() []FeatureType {
	return []FeatureType{
		FeatureSummary, FeatureSearch, FeatureChat, FeatureEmbed, FeatureModerate, FeatureTranslate,
	}
}

// AIProviderConfig represents a configured AI provider
type AIProviderConfig struct {
	ID           uuid.UUID    `json:"id" db:"id"`
	ProviderType ProviderType `json:"provider_type" db:"provider_type"`
	Name         string       `json:"name" db:"name"`
	DisplayName  string       `json:"display_name" db:"display_name"`
	BaseURL      *string      `json:"base_url,omitempty" db:"base_url"`
	IsEnabled    bool         `json:"is_enabled" db:"is_enabled"`
	IsDefault    bool         `json:"is_default" db:"is_default"`
	Priority     int          `json:"priority" db:"priority"`
	Config       *string      `json:"-" db:"config"` // JSON config (encrypted at rest)
	CreatedAt    time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at" db:"updated_at"`
}

// AIProviderConfigResponse is the safe API response format
type AIProviderConfigResponse struct {
	ID           uuid.UUID    `json:"id"`
	ProviderType ProviderType `json:"provider_type"`
	Name         string       `json:"name"`
	DisplayName  string       `json:"display_name"`
	BaseURL      *string      `json:"base_url,omitempty"`
	IsEnabled    bool         `json:"is_enabled"`
	IsDefault    bool         `json:"is_default"`
	Priority     int          `json:"priority"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

// ToResponse converts AIProviderConfig to safe API response
func (c *AIProviderConfig) ToResponse() AIProviderConfigResponse {
	return AIProviderConfigResponse{
		ID:           c.ID,
		ProviderType: c.ProviderType,
		Name:         c.Name,
		DisplayName:  c.DisplayName,
		BaseURL:      c.BaseURL,
		IsEnabled:    c.IsEnabled,
		IsDefault:    c.IsDefault,
		Priority:     c.Priority,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
	}
}

// UserAICredential represents user-specific AI provider credentials
type UserAICredential struct {
	ID           uuid.UUID    `json:"id" db:"id"`
	UserID       uuid.UUID    `json:"user_id" db:"user_id"`
	ProviderID   uuid.UUID    `json:"provider_id" db:"provider_id"`
	ProviderType ProviderType `json:"provider_type" db:"provider_type"`
	Credentials  string       `json:"-" db:"credentials"` // Encrypted JSON credentials
	IsEnabled    bool         `json:"is_enabled" db:"is_enabled"`
	LastUsedAt   *time.Time   `json:"last_used_at,omitempty" db:"last_used_at"`
	CreatedAt    time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at" db:"updated_at"`
}

// UserAICredentialResponse is the safe API response format
type UserAICredentialResponse struct {
	ID             uuid.UUID    `json:"id"`
	ProviderID     uuid.UUID    `json:"provider_id"`
	ProviderType   ProviderType `json:"provider_type"`
	IsEnabled      bool         `json:"is_enabled"`
	HasCredentials bool         `json:"has_credentials"`
	LastUsedAt     *time.Time   `json:"last_used_at,omitempty"`
	CreatedAt      time.Time    `json:"created_at"`
}

// ToResponse converts UserAICredential to safe API response
func (c *UserAICredential) ToResponse() UserAICredentialResponse {
	return UserAICredentialResponse{
		ID:             c.ID,
		ProviderID:     c.ProviderID,
		ProviderType:   c.ProviderType,
		IsEnabled:      c.IsEnabled,
		HasCredentials: c.Credentials != "",
		LastUsedAt:     c.LastUsedAt,
		CreatedAt:      c.CreatedAt,
	}
}

// ModelRouting represents per-feature model routing configuration
type ModelRouting struct {
	ID         uuid.UUID   `json:"id" db:"id"`
	ServerID   *uuid.UUID  `json:"server_id,omitempty" db:"server_id"` // nil = global default
	UserID     *uuid.UUID  `json:"user_id,omitempty" db:"user_id"`     // nil = server/global default
	Feature    FeatureType `json:"feature" db:"feature"`
	ProviderID uuid.UUID   `json:"provider_id" db:"provider_id"`
	ModelID    string      `json:"model_id" db:"model_id"`
	Priority   int         `json:"priority" db:"priority"`
	IsEnabled  bool        `json:"is_enabled" db:"is_enabled"`
	CreatedAt  time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at" db:"updated_at"`
}

// ModelInfo represents information about an available model
type ModelInfo struct {
	ID            string       `json:"id"`
	Name          string       `json:"name"`
	ProviderType  ProviderType `json:"provider_type"`
	ProviderID    uuid.UUID    `json:"provider_id"`
	Description   string       `json:"description,omitempty"`
	ContextWindow int          `json:"context_window,omitempty"`
	MaxTokens     int          `json:"max_tokens,omitempty"`
	InputCost     float64      `json:"input_cost,omitempty"`   // Per 1M tokens
	OutputCost    float64      `json:"output_cost,omitempty"`  // Per 1M tokens
	Capabilities  []string     `json:"capabilities,omitempty"` // e.g., ["chat", "embed", "vision"]
}

// ProviderCredentials is the generic credentials structure
type ProviderCredentials struct {
	APIKey         string `json:"api_key,omitempty"`
	SecretKey      string `json:"secret_key,omitempty"`
	Region         string `json:"region,omitempty"`
	ProjectID      string `json:"project_id,omitempty"`
	ServiceAccount string `json:"service_account,omitempty"` // For Vertex AI
	SessionToken   string `json:"session_token,omitempty"`   // For Bedrock assume role
}

// DefaultModels returns the recommended default models for each feature
func DefaultModels() map[FeatureType]string {
	return map[FeatureType]string{
		FeatureSummary:   "gpt-3.5-turbo", // Cheap, fast
		FeatureSearch:    "gpt-4",         // Smart, accurate
		FeatureChat:      "gpt-4-turbo",   // Balanced
		FeatureEmbed:     "text-embedding-3-small",
		FeatureModerate:  "text-moderation-stable",
		FeatureTranslate: "gpt-3.5-turbo",
	}
}
