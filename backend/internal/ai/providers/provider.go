package providers

import (
	"context"
	"errors"
	"io"
	"time"
)

// Common errors
var (
	ErrProviderNotConfigured = errors.New("provider not configured")
	ErrInvalidCredentials    = errors.New("invalid credentials")
	ErrModelNotFound         = errors.New("model not found")
	ErrRateLimitExceeded     = errors.New("rate limit exceeded")
	ErrContextLengthExceeded = errors.New("context length exceeded")
	ErrProviderUnavailable   = errors.New("provider temporarily unavailable")
	ErrInvalidRequest        = errors.New("invalid request")
	ErrStreamingNotSupported = errors.New("streaming not supported by provider")
)

// Role represents a message role in a conversation
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Message represents a chat message
type Message struct {
	Role       Role                   `json:"role"`
	Content    string                 `json:"content"`
	Name       string                 `json:"name,omitempty"` // For tool calls
	ToolCallID string                 `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall             `json:"tool_calls,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ToolCall represents a tool/function call
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"` // "function"
	Function FunctionCall `json:"function"`
}

// FunctionCall represents a function invocation
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// Tool represents a tool definition
type Tool struct {
	Type     string       `json:"type"` // "function"
	Function ToolFunction `json:"function"`
}

// ToolFunction represents a function definition
type ToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// ChatRequest represents a chat completion request
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature *float64  `json:"temperature,omitempty"`
	TopP        *float64  `json:"top_p,omitempty"`
	Stop        []string  `json:"stop,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
	Tools       []Tool    `json:"tools,omitempty"`
	ToolChoice  string    `json:"tool_choice,omitempty"` // "auto", "none", or {"type": "function", "function": {"name": "..."}}
	User        string    `json:"user,omitempty"`

	// Provider-specific options
	ProviderOptions map[string]interface{} `json:"provider_options,omitempty"`
}

// ChatResponse represents a chat completion response
type ChatResponse struct {
	ID           string   `json:"id"`
	Model        string   `json:"model"`
	Object       string   `json:"object"`
	Created      int64    `json:"created"`
	Choices      []Choice `json:"choices"`
	Usage        *Usage   `json:"usage,omitempty"`
	FinishReason string   `json:"finish_reason,omitempty"`

	// Streaming support
	Delta *Message `json:"delta,omitempty"`
}

// Choice represents a completion choice
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason,omitempty"`
}

// Usage represents token usage
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// EmbeddingRequest represents an embedding request
type EmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
	User  string   `json:"user,omitempty"`

	// Provider-specific options
	ProviderOptions map[string]interface{} `json:"provider_options,omitempty"`
}

// EmbeddingResponse represents an embedding response
type EmbeddingResponse struct {
	Object string      `json:"object"`
	Model  string      `json:"model"`
	Data   []Embedding `json:"data"`
	Usage  *Usage      `json:"usage,omitempty"`
}

// Embedding represents a single embedding
type Embedding struct {
	Object    string    `json:"object"`
	Index     int       `json:"index"`
	Embedding []float32 `json:"embedding"`
}

// StreamCallback is called for each streaming chunk
type StreamCallback func(response *ChatResponse) error

// HealthStatus represents provider health
type HealthStatus struct {
	Available bool          `json:"available"`
	Latency   time.Duration `json:"latency"`
	Error     string        `json:"error,omitempty"`
	CheckedAt time.Time     `json:"checked_at"`
}

// ModelInfo represents information about a model
type ModelInfo struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Description   string   `json:"description,omitempty"`
	ContextWindow int      `json:"context_window,omitempty"`
	MaxTokens     int      `json:"max_tokens,omitempty"`
	Capabilities  []string `json:"capabilities,omitempty"` // ["chat", "embed", "vision", "function_calling"]
	InputCost     float64  `json:"input_cost,omitempty"`   // Per 1M tokens
	OutputCost    float64  `json:"output_cost,omitempty"`  // Per 1M tokens
}

// Provider defines the interface that all AI providers must implement
type Provider interface {
	// Name returns the provider name
	Name() string

	// Type returns the provider type identifier
	Type() string

	// Chat performs a chat completion
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)

	// ChatStream performs a streaming chat completion
	ChatStream(ctx context.Context, req *ChatRequest, callback StreamCallback) error

	// Embed generates embeddings
	Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error)

	// ListModels returns available models
	ListModels(ctx context.Context) ([]ModelInfo, error)

	// HealthCheck checks if the provider is available
	HealthCheck(ctx context.Context) (*HealthStatus, error)

	// SupportsStreaming returns true if streaming is supported
	SupportsStreaming() bool

	// SupportsEmbeddings returns true if embeddings are supported
	SupportsEmbeddings() bool

	// SupportsFunctionCalling returns true if function calling is supported
	SupportsFunctionCalling() bool

	// Close releases any resources
	Close() error
}

// StreamReader provides a reader interface for streaming responses
type StreamReader interface {
	io.Reader
	io.Closer
	Next() (*ChatResponse, error)
}

// ProviderConfig is the base configuration for providers
type ProviderConfig struct {
	BaseURL        string                 `json:"base_url,omitempty"`
	APIKey         string                 `json:"api_key,omitempty"`
	SecretKey      string                 `json:"secret_key,omitempty"`
	Region         string                 `json:"region,omitempty"`
	ProjectID      string                 `json:"project_id,omitempty"`
	ServiceAccount string                 `json:"service_account,omitempty"`
	Timeout        time.Duration          `json:"timeout,omitempty"`
	MaxRetries     int                    `json:"max_retries,omitempty"`
	ExtraHeaders   map[string]string      `json:"extra_headers,omitempty"`
	Custom         map[string]interface{} `json:"custom,omitempty"`
}

// DefaultConfig returns a default provider configuration
func DefaultConfig() *ProviderConfig {
	return &ProviderConfig{
		Timeout:    30 * time.Second,
		MaxRetries: 3,
	}
}
